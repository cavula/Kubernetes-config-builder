package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const validScrapeUsage = `scrapeConfigs:
  sample-job:
    enabled: true
    scrape_interval: 30s
    metrics_path: /metrics
    scheme: http
    static_configs:
      - targets:
          - sample.default.svc:8080
`

const validAlertUsage = `serverFiles:
  alerting_rules.yml:
    groups:
      - name: sample
        rules:
          - alert: SampleDown
            expr: up{job="sample-job"} == 0
            for: 5m
            labels:
              severity: warning
            annotations:
              summary: Sample is down
`

func TestPrometheusUsageValuesRejectUnsupportedOverrides(t *testing.T) {
	for _, tc := range []struct {
		name, template, values string
	}{
		{"extra chart values", "scrape-job", validScrapeUsage + "extraScrapeConfigs: '{{ .Files }}'\n"},
		{"unsupported job fields", "scrape-job", strings.Replace(validScrapeUsage, "enabled: true", "pre_relabel_configs: '{{ .Files }}'", 1)},
		{"default job name", "scrape-job", strings.Replace(validScrapeUsage, "sample-job:", "kubernetes-pods:", 1)},
		{"unsafe job name", "scrape-job", strings.Replace(validScrapeUsage, "sample-job:", "'sample # job':", 1)},
		{"other rule files", "alert-rule", strings.Replace(validAlertUsage, "alerting_rules.yml:", "prometheus.yml:", 1)},
		{"multiple documents", "alert-rule", validAlertUsage + "---\nextraScrapeConfigs: malicious\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := renderRequest{UsageTemplate: tc.template, Values: tc.values}
			if _, err := readPrometheusUsageValues(req); err == nil {
				t.Fatal("expected unsupported values to be rejected before chart rendering")
			}
		})
	}
}

func TestPrometheusUsageValidation(t *testing.T) {
	if _, err := exec.LookPath("promtool"); err != nil {
		t.Skip("Build pinned promtool to run the usage validation integration test.")
	}
	chartDir := pinnedChartFixtures(t)
	for _, tc := range []struct {
		template, values, invalid string
	}{
		{"scrape-job", validScrapeUsage, strings.Replace(validScrapeUsage, "30s", "oops", 1)},
		{"alert-rule", validAlertUsage, strings.Replace(validAlertUsage, "up{job=\"sample-job\"} == 0", "not valid promql", 1)},
	} {
		t.Run(tc.template, func(t *testing.T) {
			req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37", UsageTemplate: tc.template, Values: tc.values}
			if result := validatePrometheusUsage(context.Background(), chartDir, "", req); result.Status != "passed" {
				t.Fatalf("expected valid values, got %+v", result)
			}
			req.Values = tc.invalid
			if result := validatePrometheusUsage(context.Background(), chartDir, "", req); result.Status != "error" ||
				len(result.Diagnostics) == 0 || result.Diagnostics[0].Check != "promtool" {
				t.Fatalf("expected a promtool error, got %+v", result)
			}
		})
	}
}

func TestPrometheusUsageAlertEvaluates(t *testing.T) {
	binary, err := exec.LookPath("promtool")
	if err != nil {
		t.Skip("Build pinned promtool to run the alert evaluation integration test.")
	}
	req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37", UsageTemplate: "alert-rule", Values: validAlertUsage}
	manifest, err := render(context.Background(), pinnedChartFixtures(t), req)
	if err != nil {
		t.Fatal(err)
	}
	config, err := renderedPrometheusConfig(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config["prometheus.yml"], "/etc/config/alerting_rules.yml") {
		t.Fatal("the chart configuration did not load the generated alert rule file")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alerting_rules.yml"), []byte(config["alerting_rules.yml"]), 0600); err != nil {
		t.Fatal(err)
	}
	fixture := `rule_files:
  - alerting_rules.yml
evaluation_interval: 1m
tests:
  - interval: 1m
    input_series:
      - series: 'up{job="sample-job"}'
        values: '0x10'
    alert_rule_test:
      - eval_time: 4m
        alertname: SampleDown
        exp_alerts: []
      - eval_time: 5m
        alertname: SampleDown
        exp_alerts:
          - exp_labels:
              job: sample-job
              severity: warning
            exp_annotations:
              summary: Sample is down
`
	if err := os.WriteFile(filepath.Join(dir, "test.yml"), []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "test", "rules", "test.yml")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("test rendered rule: %v\n%s", err, output)
	}
}

func TestPrometheusServiceSchemaValidation(t *testing.T) {
	if _, err := exec.LookPath("kubeconform"); err != nil {
		t.Skip("Install kubeconform to run the Service schema integration test.")
	}
	req := renderRequest{
		Tool: "prometheus", KubernetesVersion: "1.37", UsageTemplate: "service-scrape",
		Values: `apiVersion: v1
kind: Service
metadata:
  name: sample
  annotations:
    prometheus.io/scrape: "true"
spec:
  selector:
    app: sample
  ports:
    - port: 8080
      targetPort: 8080
`,
	}
	schemaDir := filepath.Join("..", ".runtime", "schemas")
	if result := validatePrometheusUsage(context.Background(), "", schemaDir, req); result.Status != "passed" {
		t.Fatalf("expected valid Service, got %+v", result)
	}
	req.Values = strings.Replace(req.Values, "port: 8080", "port: bad", 1)
	if result := validatePrometheusUsage(context.Background(), "", schemaDir, req); result.Status != "error" {
		t.Fatalf("expected an invalid Service error, got %+v", result)
	}
}

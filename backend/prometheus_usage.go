package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

type prometheusUsageValues struct {
	ScrapeConfigs map[string]struct {
		Enabled        bool   `yaml:"enabled"`
		ScrapeInterval string `yaml:"scrape_interval"`
		MetricsPath    string `yaml:"metrics_path"`
		Scheme         string `yaml:"scheme"`
		StaticConfigs  []struct {
			Targets []string `yaml:"targets"`
		} `yaml:"static_configs"`
	} `yaml:"scrapeConfigs,omitempty"`
	ServerFiles *struct {
		AlertingRules struct {
			Groups []struct {
				Name  string `yaml:"name"`
				Rules []struct {
					Alert  string `yaml:"alert"`
					Expr   string `yaml:"expr"`
					For    string `yaml:"for"`
					Labels struct {
						Severity string `yaml:"severity"`
					} `yaml:"labels"`
					Annotations struct {
						Summary string `yaml:"summary"`
					} `yaml:"annotations"`
				} `yaml:"rules"`
			} `yaml:"groups"`
		} `yaml:"alerting_rules.yml"`
	} `yaml:"serverFiles,omitempty"`
}

var usageJobName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func readPrometheusUsageValues(req renderRequest) (string, error) {
	var values prometheusUsageValues
	decoder := yaml.NewDecoder(strings.NewReader(req.Values))
	decoder.KnownFields(true)
	if err := decoder.Decode(&values); err != nil {
		return "", err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", fmt.Errorf("provide exactly one Helm values document")
	}
	switch req.UsageTemplate {
	case "scrape-job":
		if len(values.ScrapeConfigs) != 1 || values.ServerFiles != nil {
			return "", fmt.Errorf("provide one custom scrape job under scrapeConfigs")
		}
		for name, job := range values.ScrapeConfigs {
			if !usageJobName.MatchString(name) || slices.Contains([]string{
				"prometheus", "kubernetes-api-servers", "kubernetes-nodes", "kubernetes-nodes-cadvisor",
				"kubernetes-service-endpoints", "kubernetes-service-endpoints-slow", "prometheus-pushgateway",
				"kubernetes-services", "kubernetes-pods", "kubernetes-pods-slow",
			}, name) {
				return "", fmt.Errorf("choose a unique job name containing letters, numbers, hyphens, or underscores")
			}
			if !job.Enabled || job.ScrapeInterval == "" || !strings.HasPrefix(job.MetricsPath, "/") ||
				(job.Scheme != "http" && job.Scheme != "https") || len(job.StaticConfigs) != 1 ||
				len(job.StaticConfigs[0].Targets) != 1 || strings.TrimSpace(job.StaticConfigs[0].Targets[0]) == "" {
				return "", fmt.Errorf("provide an enabled scrape job with an interval, metrics path, HTTP scheme, and one target")
			}
		}
	case "alert-rule":
		if len(values.ScrapeConfigs) != 0 || values.ServerFiles == nil ||
			len(values.ServerFiles.AlertingRules.Groups) == 0 {
			return "", fmt.Errorf("provide alert rules under serverFiles.alerting_rules.yml.groups")
		}
	default:
		return "", fmt.Errorf("select a supported Prometheus usage template")
	}
	// Render only the decoded fields. The chart applies tpl to some other values.
	encoded, err := yaml.Marshal(values)
	return string(encoded), err
}

func validatePrometheusUsage(ctx context.Context, chartDir, schemaDir string, req renderRequest) validationResult {
	result := validationResult{Status: "passed", Diagnostics: []diagnostic{}}
	if !slices.Contains([]string{"1.35", "1.36", "1.37"}, req.KubernetesVersion) {
		result.add("Kubernetes version", "error", "Select a supported Kubernetes version.")
		return result
	}

	if req.UsageTemplate == "service-scrape" {
		resources, err := manifestResources(req.Values)
		if err != nil || len(resources) != 1 || resources[0].id.Kind != "Service" || resources[0].id.APIVersion != "v1" {
			result.add("Service schema", "error", "Provide exactly one Kubernetes v1 Service.")
			return result
		}
		for _, issue := range validateSchemas(ctx, schemaDir, req.KubernetesVersion, req.Values) {
			result.add(issue.Check, issue.Severity, issue.Message)
		}
		return result
	}

	values, err := readPrometheusUsageValues(req)
	if err != nil {
		result.add("Helm values", "error", err.Error())
		return result
	}
	req.Values = values
	manifest, err := render(ctx, chartDir, req)
	if err != nil {
		issue := renderDiagnostic(req, err)
		result.add(issue.Check, issue.Severity, issue.Message)
		return result
	}
	if issue := checkRenderedPrometheus(ctx, manifest, req.UsageTemplate); issue != nil {
		result.add(issue.Check, issue.Severity, issue.Message)
	}
	return result
}

func checkRenderedPrometheus(ctx context.Context, manifest, template string) *diagnostic {
	const check = "promtool"
	binary, err := exec.LookPath("promtool")
	if err != nil {
		return &diagnostic{check, "incomplete", "Install the pinned promtool and make it available on the backend PATH."}
	}
	config, err := renderedPrometheusConfig(manifest)
	if err != nil {
		return &diagnostic{check, "error", err.Error()}
	}
	dir, err := os.MkdirTemp("", "kcb-promtool-")
	if err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	defer os.RemoveAll(dir)
	commands := [][]string{{"check", "config", "--syntax-only", "prometheus.yml"}}
	if template == "alert-rule" {
		commands = append(commands, []string{"check", "rules", "alerting_rules.yml"})
	}
	for _, args := range commands {
		filename := args[len(args)-1]
		if config[filename] == "" {
			return &diagnostic{check, "error", "The pinned chart did not render " + filename + "."}
		}
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(config[filename]), 0600); err != nil {
			return &diagnostic{check, "incomplete", err.Error()}
		}
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			message := strings.TrimSpace(string(output))
			if ctx.Err() != nil || message == "" {
				return &diagnostic{check, "incomplete", fmt.Sprintf("Prometheus validation did not complete: %v", err)}
			}
			return &diagnostic{check, "error", message}
		}
	}
	return nil
}

func renderedPrometheusConfig(manifest string) (map[string]string, error) {
	resources, err := manifestResources(manifest)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		if resource.id.Kind != "ConfigMap" || resource.id.Name != "prometheus-server" {
			continue
		}
		var config struct {
			Data map[string]string
		}
		if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &config); err != nil {
			return nil, err
		}
		return config.Data, nil
	}
	return nil, fmt.Errorf("the pinned chart did not render its Prometheus server ConfigMap")
}

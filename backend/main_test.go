package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedChartsRender(t *testing.T) {
	chartDir := "../.runtime/charts"
	for tool, chart := range charts {
		if _, err := os.Stat(filepath.Join(chartDir, tool+"-"+chart.version+".tgz")); os.IsNotExist(err) {
			t.Skip("download the pinned charts to .runtime/charts to run rendering tests")
		}
		for _, version := range []string{"1.37", "1.36", "1.35"} {
			t.Run(tool+"/"+version, func(t *testing.T) {
				output, err := render(context.Background(), chartDir, renderRequest{Tool: tool, KubernetesVersion: version})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(output, "kind: Deployment") {
					t.Fatal("render did not include a Deployment")
				}
				if strings.Contains(output, "privileged: true") {
					t.Fatal("chart defaults rendered a privileged container")
				}
			})
		}
	}
}

func TestApprovedOptionsAffectPinnedCharts(t *testing.T) {
	chartDir := "../.runtime/charts"
	for tool, chart := range charts {
		if _, err := os.Stat(filepath.Join(chartDir, tool+"-"+chart.version+".tgz")); os.IsNotExist(err) {
			t.Skip("download the pinned charts to .runtime/charts to run rendering tests")
		}
	}
	tests := []struct {
		tool, setting, values, expected, removed string
	}{
		{"prometheus", "scrape interval", `{"server":{"global":{"scrape_interval":"30s"}}}`, "scrape_interval: 30s", "scrape_interval: 1m"},
		{"prometheus", "retention", `{"server":{"retention":"7d"}}`, "--storage.tsdb.retention.time=7d", "--storage.tsdb.retention.time=15d"},
		{"prometheus", "persistence", `{"server":{"persistentVolume":{"enabled":false}}}`, "name: storage-volume\n          emptyDir:", "prometheus/templates/pvc.yaml"},
		{"prometheus", "storage size", `{"server":{"persistentVolume":{"size":"4Gi"}}}`, `storage: "4Gi"`, `storage: "8Gi"`},
		{"prometheus", "server resources", `{"server":{"resources":{"requests":{"cpu":"73m","memory":"87Mi"}}}}`, "cpu: 73m", ""},
		{"prometheus", "Alertmanager", `{"alertmanager":{"enabled":false}}`, "", "name: prometheus-alertmanager"},
		{"prometheus", "kube-state-metrics", `{"kube-state-metrics":{"enabled":false}}`, "", "name: prometheus-kube-state-metrics"},
		{"argo-cd", "service type", `{"server":{"service":{"type":"NodePort"}}}`, "nodePort: 30080", ""},
		{"argo-cd", "server replicas", `{"server":{"replicas":2}}`, "replicas: 2", ""},
		{"argo-cd", "repository server replicas", `{"repoServer":{"replicas":3}}`, "replicas: 3", ""},
		{"argo-cd", "server resources", `{"server":{"resources":{"requests":{"cpu":"73m"}}}}`, "cpu: 73m", ""},
		{"argo-cd", "repository server resources", `{"repoServer":{"resources":{"limits":{"memory":"256Mi"}}}}`, "memory: 256Mi", ""},
		{"argo-cd", "metrics", `{"server":{"metrics":{"enabled":true}}}`, "name: argo-cd-argocd-server-metrics", ""},
		{"argo-cd", "Dex", `{"dex":{"enabled":false}}`, "", "name: argo-cd-argocd-dex-server"},
	}
	for _, tt := range tests {
		t.Run(tt.tool+"/"+tt.setting, func(t *testing.T) {
			req := renderRequest{Tool: tt.tool, KubernetesVersion: "1.37"}
			baseline, err := render(context.Background(), chartDir, req)
			if err != nil {
				t.Fatal(err)
			}
			req.Values = tt.values
			output, err := render(context.Background(), chartDir, req)
			if err != nil {
				t.Fatal(err)
			}
			expectedBaseline, expectedOutput := baseline, output
			if tt.setting == "persistence" {
				_, expectedBaseline, _ = strings.Cut(baseline, "# Source: prometheus/templates/deploy.yaml\n")
				expectedBaseline, _, _ = strings.Cut(expectedBaseline, "\n---")
				_, expectedOutput, _ = strings.Cut(output, "# Source: prometheus/templates/deploy.yaml\n")
				expectedOutput, _, _ = strings.Cut(expectedOutput, "\n---")
			}
			if tt.expected != "" && (strings.Contains(expectedBaseline, tt.expected) || !strings.Contains(expectedOutput, tt.expected)) {
				t.Errorf("expected %q only after the edit", tt.expected)
			}
			if tt.removed != "" && (!strings.Contains(baseline, tt.removed) || strings.Contains(output, tt.removed)) {
				t.Errorf("expected %q only before the edit", tt.removed)
			}
		})
	}
}

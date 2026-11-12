package main

import (
	"context"
	"strings"
	"testing"

	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
)

func TestValidationReportsUnavailableTools(t *testing.T) {
	chartDir := pinnedChartFixtures(t)
	t.Setenv("PATH", t.TempDir())
	result := validate(context.Background(), chartDir, t.TempDir(), renderRequest{
		Tool: "prometheus", KubernetesVersion: "1.37",
	})
	if result.Status != "incomplete" || len(result.Diagnostics) != 2 {
		t.Fatalf("expected incomplete diagnostics for both validators, got %+v", result)
	}
	if result.Diagnostics[0].Check != "Helm lint" || result.Diagnostics[1].Check != "kubeconform" {
		t.Fatalf("expected both validator names, got %+v", result.Diagnostics)
	}
}

func TestValidationReportsMalformedValues(t *testing.T) {
	chartDir := pinnedChartFixtures(t)
	result := validate(context.Background(), chartDir, t.TempDir(), renderRequest{
		Tool: "prometheus", KubernetesVersion: "1.37", Values: "server: [",
	})
	if result.Status != "error" || len(result.Diagnostics) != 1 ||
		!strings.Contains(result.Diagnostics[0].Message, "prometheus on Kubernetes 1.37") ||
		!strings.Contains(result.Diagnostics[0].Message, "yaml: line 1") {
		t.Fatalf("expected a tool-specific YAML error, got %+v", result)
	}
}

func TestValidationWarnsWhenChartRequiresNewerKubernetes(t *testing.T) {
	req := renderRequest{Tool: "argo-cd", KubernetesVersion: "1.35"}
	chart := &chartv2.Chart{Metadata: &chartv2.Metadata{
		Name: "argo-cd", Version: charts[req.Tool].version, KubeVersion: ">=1.36.0 <1.38.0",
	}}
	_, err := renderLoadedChart(context.Background(), chart, charts[req.Tool], req)
	if err == nil {
		t.Fatal("expected chart version mismatch")
	}
	issue := renderDiagnostic(req, err)
	if issue.Severity != "warning" || issue.Check != "Compatibility" {
		t.Fatalf("expected a compatibility warning, got %+v", issue)
	}
	for _, expected := range []string{"argo-cd", "10.17.1", ">=1.36.0 <1.38.0", "Kubernetes 1.35"} {
		if !strings.Contains(issue.Message, expected) {
			t.Errorf("warning %q does not include %q", issue.Message, expected)
		}
	}
}

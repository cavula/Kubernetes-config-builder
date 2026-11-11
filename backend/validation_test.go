package main

import (
	"context"
	"strings"
	"testing"
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

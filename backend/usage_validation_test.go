package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestArgoUsageSchemasComeFromVerifiedChart(t *testing.T) {
	chartDir := pinnedChartFixtures(t)
	for _, kind := range []string{"Application", "AppProject", "ApplicationSet"} {
		t.Run(kind, func(t *testing.T) {
			data, err := argoUsageSchema(context.Background(), chartDir, "1.37", resourceID{Kind: kind})
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{`"spec":`, `"additionalProperties":false`} {
				if !strings.Contains(string(data), field) {
					t.Errorf("missing %s in the derived schema", field)
				}
			}
		})
	}
}

func TestArgoUsageValidation(t *testing.T) {
	if _, err := exec.LookPath("kubeconform"); err != nil {
		t.Skip("Install kubeconform to run the usage validation integration test.")
	}
	chartDir := pinnedChartFixtures(t)
	for _, fixture := range []struct {
		kind  string
		spec  string
		field string
	}{
		{"Application", "  project: default\n  destination:\n    server: https://kubernetes.default.svc\n    namespace: guestbook\n", "project"},
		{"AppProject", "  sourceRepos: [https://example.org/sample.git]\n", "sourceRepos"},
		{"ApplicationSet", "  generators: []\n  template:\n    metadata:\n      name: sample\n    spec:\n      project: default\n      destination:\n        server: https://kubernetes.default.svc\n", "generators"},
	} {
		t.Run(fixture.kind, func(t *testing.T) {
			req := renderRequest{Tool: "argo-cd", KubernetesVersion: "1.37"}
			header := "apiVersion: argoproj.io/v1alpha1\nkind: " + fixture.kind + "\nmetadata:\n  name: sample\n  namespace: argocd\nspec:\n"
			req.Values = header + fixture.spec
			if result := validateUsage(context.Background(), chartDir, req); result.Status != "passed" {
				t.Fatalf("expected valid resource, got %+v", result)
			}
			req.Values = strings.Replace(req.Values, fixture.field+":", "wrongField:", 1)
			if result := validateUsage(context.Background(), chartDir, req); result.Status != "error" {
				t.Fatalf("expected unknown field error, got %+v", result)
			}
		})
	}
}

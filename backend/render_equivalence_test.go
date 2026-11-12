package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPlainYAMLMatchesHelmTemplate(t *testing.T) {
	chartDir := pinnedChartFixtures(t)
	var cases []renderRequest
	for _, tool := range []string{"prometheus", "argo-cd"} {
		values, err := os.ReadFile(filepath.Join("..", "tests", "fixtures", tool+"-resource-options.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, renderRequest{Tool: tool}, renderRequest{Tool: tool, Values: string(values)})
	}
	for _, version := range []string{"1.35", "1.36", "1.37"} {
		for _, req := range cases {
			req.KubernetesVersion = version
			name := req.Tool + "/" + version
			if req.Values != "" {
				name += "/options"
			}
			t.Run(name, func(t *testing.T) {
				exported, err := render(context.Background(), chartDir, req)
				if err != nil {
					t.Fatal(err)
				}
				pin := charts[req.Tool]
				valuesFile := filepath.Join(t.TempDir(), "values.yaml")
				if err := os.WriteFile(valuesFile, []byte(req.Values), 0600); err != nil {
					t.Fatal(err)
				}
				args := []string{"template", req.Tool, filepath.Join(chartDir, req.Tool+"-"+pin.version+".tgz"),
					"--kube-version", version + ".0", "--namespace", pin.namespace, "--include-crds"}
				if req.Tool == "prometheus" {
					args = append(args, "--values", "prometheus-arm64-values.json")
				}
				args = append(args, "--values", valuesFile)
				output, err := exec.Command("helm", args...).Output()
				if err != nil {
					t.Fatalf("helm template: %v", err)
				}
				for _, resource := range resourcesForTest(t, exported) {
					if !strings.Contains(strings.Join(resource.lines, "\n"), "# Purpose: ") {
						t.Errorf("export lacks a section comment for %s %s", resource.id.Kind, resource.id.Name)
					}
				}
				actual := resourceJSON(t, exported)
				expected := resourceJSON(t, string(output))
				if req.Tool == "argo-cd" {
					// A kubectl export bootstraps the namespace, which Helm's
					// namespace creation is a separate operation.
					id := resourceID{APIVersion: "v1", Kind: "Namespace", Name: pin.namespace}
					if actual[id] == "" {
						t.Fatal("Argo CD export lacks its bootstrap Namespace")
					}
					delete(actual, id)
				}
				for id, object := range expected {
					if actual[id] != object {
						t.Errorf("export differs from Helm for %s %s/%s", id.Kind, id.Namespace, id.Name)
					}
				}
				if len(actual) != len(expected) {
					t.Errorf("export has %d resources, Helm has %d", len(actual), len(expected))
				}
			})
		}
	}
}

func resourcesForTest(t *testing.T, manifest string) []manifestResource {
	t.Helper()
	resources, err := manifestResources(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return resources
}

func resourceJSON(t *testing.T, manifest string) map[resourceID]string {
	t.Helper()
	resources := resourcesForTest(t, manifest)
	result := make(map[resourceID]string, len(resources))
	for _, resource := range resources {
		data, err := yaml.ToJSON([]byte(strings.Join(resource.lines, "\n")))
		if err != nil {
			t.Fatal(err)
		}
		if _, duplicate := result[resource.id]; duplicate {
			t.Fatalf("duplicate resource: %+v", resource.id)
		}
		result[resource.id] = string(data)
	}
	return result
}

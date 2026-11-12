package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestExplainedExportsPreserveResourcesAndCoverFields(t *testing.T) {
	for _, tool := range []string{"prometheus", "argo-cd"} {
		t.Run(tool, func(t *testing.T) {
			req := renderRequest{Tool: tool, KubernetesVersion: "1.37"}
			raw, err := render(context.Background(), pinnedChartFixtures(t), req)
			if err != nil {
				t.Fatal(err)
			}
			export, err := explainExport(raw, "../.runtime/schemas", req)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(export.Manifests, "# After install:") ||
				!strings.Contains(export.Manifests, "# Why: Use apps/v1 because") {
				t.Fatal("missing access or API-version explanation")
			}
			if !reflect.DeepEqual(resourceJSON(t, raw), resourceJSON(t, export.Manifests+export.CRDs)) {
				t.Fatal("comments or CRD separation changed the rendered resources")
			}
			for _, resource := range resourcesForTest(t, export.Manifests) {
				var doc yaml.Node
				if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &doc); err != nil {
					t.Fatal(err)
				}
				var check func(*yaml.Node)
				check = func(node *yaml.Node) {
					if node.Kind == yaml.MappingNode && node.Style != yaml.FlowStyle {
						for i := 0; i < len(node.Content); i += 2 {
							field := node.Content[i]
							if field.Line < 2 || !strings.HasPrefix(strings.TrimSpace(resource.lines[field.Line-2]), "# Why: ") {
								t.Errorf("%s %s lacks explanation for %s", resource.id.Kind, resource.id.Name, field.Value)
							}
						}
					}
					for _, child := range node.Content {
						check(child)
					}
				}
				check(&doc)
			}
			if tool == "argo-cd" {
				if !strings.Contains(export.CRDs, "A CustomResourceDefinition") ||
					strings.Contains(export.Manifests, "kind: CustomResourceDefinition") {
					t.Fatal("CRDs must be explained in a separate file")
				}
				for _, note := range []string{"REQUIRED IF you enable", "REQUIRED IF setting your own", "argocd account bcrypt", "Never commit real secrets to Git", "argocd admin initial-password"} {
					if !strings.Contains(export.Manifests, note) {
						t.Errorf("missing credential guidance %q", note)
					}
				}
			}
		})
	}
}

func TestPrometheusExportBootstrapsMonitoringNamespace(t *testing.T) {
	req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37"}
	raw, err := render(context.Background(), pinnedChartFixtures(t), req)
	if err != nil {
		t.Fatal(err)
	}
	resources := resourcesForTest(t, raw)
	if len(resources) == 0 || resources[0].id != (resourceID{APIVersion: "v1", Kind: "Namespace", Name: "monitoring"}) {
		t.Fatal("the monitoring namespace must be the first resource")
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

func validateUsage(ctx context.Context, chartDir string, req renderRequest) validationResult {
	result := validationResult{Status: "passed", Diagnostics: []diagnostic{}}
	const check = "Argo CD CRD schema"
	if req.Tool != "argo-cd" || !slices.Contains([]string{"1.35", "1.36", "1.37"}, req.KubernetesVersion) {
		result.add(check, "error", "Select Argo CD and a supported Kubernetes version.")
		return result
	}
	resources, err := manifestResources(req.Values)
	if err != nil {
		result.add(check, "error", err.Error())
		return result
	}
	if len(resources) != 1 {
		result.add(check, "error", "Provide exactly one Argo CD resource.")
		return result
	}
	id := resources[0].id
	if id.APIVersion != "argoproj.io/v1alpha1" || !slices.Contains([]string{"Application", "AppProject", "ApplicationSet"}, id.Kind) {
		result.add(check, "error", "Use an argoproj.io/v1alpha1 Application, AppProject, or ApplicationSet.")
		return result
	}

	schema, err := argoUsageSchema(ctx, chartDir, req.KubernetesVersion, id)
	if err != nil {
		result.add(check, "incomplete", err.Error())
		return result
	}
	dir, err := os.MkdirTemp("", "kcb-usage-schema-")
	if err != nil {
		result.add(check, "incomplete", err.Error())
		return result
	}
	defer os.RemoveAll(dir)
	location := filepath.Join(dir, strings.ToLower(id.Kind)+"_v1alpha1.json")
	if err := os.WriteFile(location, schema, 0600); err != nil {
		result.add(check, "incomplete", err.Error())
		return result
	}
	for _, issue := range validateSchemaLocations(ctx, req.KubernetesVersion, req.Values, location) {
		result.add(check, issue.Severity, issue.Message)
	}
	return result
}

// The pinned chart supplies the official schemas through its CRD templates.
// Render it with trusted defaults so the submitted YAML cannot affect the schema.
func argoUsageSchema(ctx context.Context, chartDir, kubernetesVersion string, id resourceID) ([]byte, error) {
	rendered, err := render(ctx, chartDir, renderRequest{Tool: "argo-cd", KubernetesVersion: kubernetesVersion})
	if err != nil {
		return nil, err
	}
	resources, err := manifestResources(rendered)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		if resource.id.Kind != "CustomResourceDefinition" || resource.id.Name != strings.ToLower(id.Kind)+"s.argoproj.io" {
			continue
		}
		var crd struct {
			Spec struct {
				Versions []struct {
					Name   string
					Served bool
					Schema struct {
						OpenAPI map[string]any `yaml:"openAPIV3Schema"`
					}
				}
			}
		}
		if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &crd); err != nil {
			return nil, err
		}
		for _, version := range crd.Spec.Versions {
			if version.Name == "v1alpha1" && version.Served && len(version.Schema.OpenAPI) != 0 {
				strictCRDSchema(version.Schema.OpenAPI)
				return json.Marshal(version.Schema.OpenAPI)
			}
		}
	}
	return nil, fmt.Errorf("the pinned Argo CD chart did not provide a served schema for %s", id.Kind)
}

// Kubernetes rejects unknown fields unless a schema preserves them or permits a map.
func strictCRDSchema(schema any) {
	switch node := schema.(type) {
	case map[string]any:
		if node["properties"] != nil && node["additionalProperties"] == nil &&
			node["x-kubernetes-preserve-unknown-fields"] != true {
			node["additionalProperties"] = false
		}
		for _, value := range node {
			strictCRDSchema(value)
		}
	case []any:
		for _, value := range node {
			strictCRDSchema(value)
		}
	}
}

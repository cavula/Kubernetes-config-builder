package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

//go:embed export-guides.json
var exportGuideJSON []byte

//go:embed deployment-commands.json
var deploymentCommandsJSON []byte

type explainedExport struct {
	Manifests string `json:"manifests"`
	CRDs      string `json:"crds"`
	YAML      string `json:"yaml"`
}

func explainExport(manifest, schemaDir string, req renderRequest) (explainedExport, error) {
	var guides map[string][]string
	if err := json.Unmarshal(exportGuideJSON, &guides); err != nil {
		return explainedExport{}, err
	}
	resources, err := manifestResources(manifest)
	if err != nil {
		return explainedExport{}, err
	}
	var commands map[string]struct{ YAML []string }
	if err := json.Unmarshal(deploymentCommandsJSON, &commands); err != nil {
		return explainedExport{}, err
	}
	var header, workloads, crds strings.Builder
	for _, command := range commands[req.Tool].YAML {
		fmt.Fprintln(&header, "# "+command)
	}
	if req.Tool == "argo-cd" {
		header.WriteString("# Server-side apply avoids the annotation size limit for Argo CD's large CRDs.\n")
	}
	for _, line := range guides[req.Tool] {
		fmt.Fprintln(&header, "# "+line)
	}
	header.WriteString("# Field sources: the pinned chart and Kubernetes API schemas. https://kubernetes.io/docs/reference/kubernetes-api/\n")

	hasCRDs := false
	for _, resource := range resources {
		if resource.id.Kind == "CustomResourceDefinition" {
			hasCRDs = true
			crds.WriteString("---\n")
			for _, line := range resource.lines {
				if !strings.HasPrefix(line, "# Purpose: ") {
					fmt.Fprintln(&crds, line)
				}
			}
			continue
		}
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &doc); err != nil {
			return explainedExport{}, err
		}
		schema := schemaForResource(schemaDir, req.KubernetesVersion, resource.id)
		notes := make(map[int]string)
		collectFieldNotes(doc.Content[0], resource, schema, schema, nil, "", notes)
		workloads.WriteString("---\n")
		for _, note := range secretGuidance(resource.id) {
			fmt.Fprintln(&workloads, "# "+note)
		}
		for offset, line := range resource.lines {
			if note := notes[resource.start+offset]; note != "" {
				indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
				fmt.Fprintf(&workloads, "%s# Why: %s\n", indent, strings.Join(strings.Fields(note), " "))
			}
			fmt.Fprintln(&workloads, line)
		}
	}
	result := explainedExport{Manifests: header.String() + workloads.String(), YAML: header.String() + workloads.String()}
	if hasCRDs {
		const explanation = "# A CustomResourceDefinition registers a new resource type and its validation schema.\n"
		result.CRDs = "# kubectl apply --server-side -f argo-cd-crds.yaml\n" + explanation + crds.String()
		result.YAML = header.String() + explanation + crds.String() + workloads.String()
	}
	return result, nil
}

func secretGuidance(id resourceID) []string {
	if id.Kind != "Secret" {
		return nil
	}
	switch id.Name {
	case "argocd-notifications-secret":
		return []string{
			"REQUIRED IF you enable a notification integration: add the credential keys that its service documentation specifies.",
			"Credentials are omitted so the export can be shared safely. Obtain the token or password from the service provider as directed by the official integration guide.",
			"Set the values in the live Secret. Never commit real secrets to Git.",
			"Source: https://argo-cd.readthedocs.io/en/stable/operator-manual/notifications/services/overview/",
		}
	case "argocd-secret":
		return []string{
			"Argo CD generates its initial admin password and signing key at startup. Leaving them out keeps credentials out of this export.",
			"REQUIRED IF setting your own admin password: set admin.password to a bcrypt hash and admin.passwordMtime to the update timestamp.",
			"Generate the hash with: argocd account bcrypt --password <YOUR-PASSWORD-HERE>",
			"Never commit real secrets to Git. Source: https://argo-cd.readthedocs.io/en/stable/faq/",
		}
	}
	return nil
}

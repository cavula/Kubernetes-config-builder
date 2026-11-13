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

func explainExport(manifest string, req renderRequest) (explainedExport, error) {
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
		if !strings.HasPrefix(line, "These commands assume") {
			fmt.Fprintln(&header, "# "+line)
		}
	}

	hasCRDs := false
	for _, resource := range resources {
		if resource.id.Kind == "CustomResourceDefinition" {
			hasCRDs = true
			crds.WriteString("---\n")
			for _, line := range resource.lines {
				if !strings.HasPrefix(line, "# Purpose: ") && (req.CommentDensity != "compact" || !strings.HasPrefix(line, "# Source: ")) {
					fmt.Fprintln(&crds, line)
				}
			}
			continue
		}
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &doc); err != nil {
			return explainedExport{}, err
		}
		notes := make(map[int]string)
		if req.CommentDensity != "compact" {
			collectExportNotes(doc.Content[0], resource.id, nil, notes, make(map[string]bool))
		}
		workloads.WriteString("---\n")
		for _, note := range secretGuidance(resource.id) {
			fmt.Fprintln(&workloads, "# "+note)
		}
		for offset, line := range resource.lines {
			if req.CommentDensity == "compact" && (strings.HasPrefix(strings.TrimSpace(line), "# Source: ") || strings.HasPrefix(strings.TrimSpace(line), "# v0.95.2")) {
				continue
			}
			if note := notes[offset+1]; note != "" {
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

// Export only authored notes. Hover still uses the schema for fields omitted here.
func collectExportNotes(node *yaml.Node, id resourceID, path []string, notes map[int]string, seen map[string]bool) {
	switch node.Kind {
	case yaml.MappingNode:
		for index := 0; index < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			fieldPath := append(path, key.Value)
			switch key.Value {
			case "kind", "data", "stringData", "annotations":
				continue
			}
			note := fieldWhy(id, fieldPath, value, "")
			if len(fieldPath) == 1 && (key.Value == "metadata" || key.Value == "spec") {
				note = ""
			}
			if key.Value == "args" && len(path) != 0 && (path[len(path)-1] == "containers" || path[len(path)-1] == "initContainers") {
				for i := 0; i < len(node.Content); i += 2 {
					if node.Content[i].Value == "name" {
						note = "Configure how " + node.Content[i+1].Value + " starts and runs with these flags."
						break
					}
				}
			}
			if note != "" && !seen[note] {
				notes[key.Line] = note
				seen[note] = true
			}
			switch key.Value {
			case "labels", "args", "rules", "matchLabels":
				continue
			}
			collectExportNotes(value, id, fieldPath, notes, seen)
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			collectExportNotes(item, id, path, notes, seen)
		}
	}
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

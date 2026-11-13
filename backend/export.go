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
		if req.Tool == "prometheus" && resource.id.Kind == "ConfigMap" {
			collectScrapeJobsNote(doc.Content[0], notes)
		}
		if req.CommentDensity != "compact" {
			collectExportNotes(doc.Content[0], resource.id, nil, notes, make(map[string]bool))
		}
		workloads.WriteString("---\n")
		for _, note := range secretGuidance(resource.id) {
			fmt.Fprintln(&workloads, "# "+note)
		}
		for offset, line := range resource.lines {
			if req.CommentDensity == "compact" && strings.HasPrefix(strings.TrimSpace(line), "# Source: ") {
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

const scrapeJobsNote = "These jobs tell Prometheus which targets to discover and where to collect metrics."

func collectScrapeJobsNote(doc *yaml.Node, notes map[int]string) {
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value != "data" {
			continue
		}
		data := doc.Content[i+1]
		for j := 0; j+1 < len(data.Content); j += 2 {
			key, config := data.Content[j], data.Content[j+1]
			if key.Value != "prometheus.yml" || config.Style != yaml.LiteralStyle {
				continue
			}
			for offset, line := range strings.Split(config.Value, "\n") {
				if line == "scrape_configs:" {
					notes[config.Line+offset+1] = scrapeJobsNote
					return
				}
			}
		}
	}
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
			if id.Kind == "Deployment" && len(fieldPath) == 2 && fieldPath[0] == "spec" &&
				key.Value == "replicas" && value.Value == "0" &&
				strings.HasSuffix(id.Name, "-argocd-applicationset-controller") {
				note = "The chart always renders this Deployment. Zero replicas keeps the ApplicationSet controller stopped."
			}
			if len(fieldPath) == 1 && (key.Value == "metadata" || (key.Value == "spec" && id.Kind != "Service")) {
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
			// Each container needs its own probe and mount explanations.
			containerBlock := false
			if len(path) > 0 {
				switch path[len(path)-1] {
				case "containers", "initContainers", "ephemeralContainers":
					switch key.Value {
					case "readinessProbe", "livenessProbe", "startupProbe", "volumeMounts":
						containerBlock = true
					}
				}
			}
			if note != "" && (containerBlock || !seen[note]) {
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

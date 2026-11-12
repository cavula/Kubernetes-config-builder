package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type resourceID struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type lineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type resourceChange struct {
	Resource resourceID `json:"resource"`
	Type     string     `json:"type"`
	Current  *lineRange `json:"current,omitempty"`
	Prior    *lineRange `json:"prior,omitempty"`
}

type manifestResource struct {
	id    resourceID
	start int
	lines []string
}

// Helm separates its top-level manifests with unindented YAML document markers.
// Keep their original line offsets so links point to the exported text.
func manifestResources(text string) ([]manifestResource, error) {
	lines := strings.Split(text, "\n")
	resources := []manifestResource{}
	start := 0
	for end := 0; end <= len(lines); end++ {
		if end < len(lines) && strings.TrimRight(lines[end], "\r \t") != "---" {
			continue
		}
		document := lines[start:end]
		startLine := start + 1
		start = end + 1
		for len(document) > 0 && strings.TrimSpace(document[len(document)-1]) == "" {
			document = document[:len(document)-1]
		}
		if len(document) == 0 {
			continue
		}
		data, err := yaml.ToJSON([]byte(strings.Join(document, "\n")))
		if err != nil {
			return nil, fmt.Errorf("parse manifest at line %d: %w", startLine, err)
		}
		var metadata struct {
			APIVersion string
			Kind       string
			Metadata   struct{ Namespace, Name string }
		}
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil, fmt.Errorf("parse resource identity at line %d: %w", startLine, err)
		}
		if metadata.Kind == "" && metadata.Metadata.Name == "" {
			continue // Header comments can precede the first document separator.
		}
		if metadata.Kind == "" || metadata.Metadata.Name == "" || metadata.APIVersion == "" {
			return nil, fmt.Errorf("incomplete resource identity at line %d", startLine)
		}
		id := resourceID{metadata.APIVersion, metadata.Kind, metadata.Metadata.Namespace, metadata.Metadata.Name}
		resources = append(resources, manifestResource{id, startLine, document})
	}
	return resources, nil
}

func documentRange(resource manifestResource, from, to int) *lineRange {
	if from == to {
		return nil
	}
	return &lineRange{Start: resource.start + from, End: resource.start + to - 1}
}

func manifestChanges(current, prior string) ([]resourceChange, error) {
	changes := []resourceChange{}
	if current == prior {
		return changes, nil
	}
	currentResources, err := manifestResources(current)
	if err != nil {
		return nil, err
	}
	priorResources, err := manifestResources(prior)
	if err != nil {
		return nil, err
	}
	priorByID := make(map[resourceID]manifestResource, len(priorResources))
	for _, resource := range priorResources {
		if _, exists := priorByID[resource.id]; exists {
			return nil, fmt.Errorf("duplicate resource in comparison render: %+v", resource.id)
		}
		priorByID[resource.id] = resource
	}

	seen := make(map[resourceID]bool, len(currentResources))
	for _, resource := range currentResources {
		if seen[resource.id] {
			return nil, fmt.Errorf("duplicate resource in current render: %+v", resource.id)
		}
		seen[resource.id] = true
		previous, exists := priorByID[resource.id]
		if !exists {
			changes = append(changes, resourceChange{
				Resource: resource.id, Type: "added", Current: documentRange(resource, 0, len(resource.lines)),
			})
			continue
		}
		if slices.Equal(previous.lines, resource.lines) {
			continue
		}
		for _, edit := range difflib.NewMatcher(previous.lines, resource.lines).GetOpCodes() {
			if edit.Tag == 'e' {
				continue
			}
			changes = append(changes, resourceChange{
				Resource: resource.id, Type: "changed",
				Current: documentRange(resource, edit.J1, edit.J2),
				Prior:   documentRange(previous, edit.I1, edit.I2),
			})
		}
	}
	for _, resource := range priorResources {
		if !seen[resource.id] {
			changes = append(changes, resourceChange{
				Resource: resource.id, Type: "removed", Prior: documentRange(resource, 0, len(resource.lines)),
			})
		}
	}
	return changes, nil
}

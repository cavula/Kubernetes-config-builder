package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

type schemaObject map[string]any

var schemaPathComponent = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)

func schemaForResource(schemaDir, version string, id resourceID) schemaObject {
	group, apiVersion, hasGroup := strings.Cut(id.APIVersion, "/")
	kind := strings.ToLower(id.Kind)
	if !schemaPathComponent.MatchString(kind) || !schemaPathComponent.MatchString(group) ||
		(hasGroup && !schemaPathComponent.MatchString(apiVersion)) {
		return nil
	}
	suffix := group
	if hasGroup {
		suffix = strings.SplitN(group, ".", 2)[0] + "-" + apiVersion
	}
	paths := []string{filepath.Join(schemaDir, "v"+version+".0-standalone-strict", kind+"-"+suffix+".json")}
	if hasGroup {
		paths = append(paths, filepath.Join(schemaDir, "custom", group, kind+"_"+apiVersion+".json"))
	}
	for _, path := range paths {
		var schema schemaObject
		data, err := os.ReadFile(path)
		if err == nil && json.Unmarshal(data, &schema) == nil {
			return schema
		}
	}
	return nil
}

func object(value any) schemaObject {
	result, _ := value.(map[string]any)
	return result
}

func resolveSchema(root, schema schemaObject) schemaObject {
	for i := 0; i < 16; i++ {
		ref, ok := schema["$ref"].(string)
		if !ok || !strings.HasPrefix(ref, "#/") {
			break
		}
		target := root
		for _, part := range strings.Split(ref[2:], "/") {
			key := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			target = object(target[key])
		}
		schema = target
	}
	return schema
}

func fieldDescription(id resourceID, path []string, schema schemaObject, inherited string) string {
	if text, ok := schema["description"].(string); ok && strings.TrimSpace(text) != "" {
		firstParagraph, _, _ := strings.Cut(strings.TrimSpace(text), "\n\n")
		return firstParagraph
	}
	key := path[len(path)-1]
	if len(path) == 1 {
		switch key {
		case "apiVersion":
			return "Selects the Kubernetes API group and version used to read this resource."
		case "kind":
			if purpose := manifestSectionNotes[id.Kind]; purpose != "" {
				return purpose
			}
			return "Selects the resource kind Kubernetes will create or update."
		case "metadata":
			return "Identifies this resource and stores its labels and annotations."
		case "spec":
			return "Describes the desired configuration of this resource."
		}
	}
	label := strings.Join(path, ".")
	switch {
	case strings.HasPrefix(label, "metadata.labels."):
		return "Label “" + key + "” identifies or groups this resource for selectors and tooling."
	case strings.HasPrefix(label, "metadata.annotations."):
		return "Annotation “" + key + "” stores information for the component that recognizes this key."
	case id.Kind == "ConfigMap" && len(path) == 2 && path[0] == "data":
		return "Configuration entry “" + key + "” is stored as text for workloads that read this ConfigMap."
	case id.Kind == "Secret" && len(path) == 2 && path[0] == "data":
		return "Secret entry “" + key + "” is stored as a base64-encoded value for workloads that reference this Secret."
	}
	if inherited != "" {
		return "“" + key + "” is an entry under “" + strings.Join(path[:len(path)-1], ".") + "”. " + inherited
	}
	return "“" + label + "” is part of the configuration for this " + id.Kind + "."
}

func manifestFieldHover(manifest, schemaDir, version string, line int) string {
	start := 1
	for _, document := range strings.Split(manifest, "\n---\n") {
		end := start + strings.Count(document, "\n")
		if line >= start && line <= end {
			return manifestHoverNotes(document, schemaDir, version)[line-start+1]
		}
		start = end + 2 // The separator occupies its own line.
	}
	return ""
}

func manifestHoverNotes(manifest, schemaDir, version string) map[int]string {
	notes := make(map[int]string)
	resources, err := manifestResources(manifest)
	if err != nil {
		return notes
	}
	schemas := make(map[[2]string]schemaObject)
	for _, resource := range resources {
		key := [2]string{resource.id.APIVersion, resource.id.Kind}
		schema, found := schemas[key]
		if !found {
			schema = schemaForResource(schemaDir, version, resource.id)
			schemas[key] = schema
		}
		var document yaml.Node
		if yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &document) != nil || len(document.Content) == 0 {
			continue
		}
		collectFieldNotes(document.Content[0], resource, schema, schema, nil, "", notes)
		// Literal blocks and wrapped scalars retain their enclosing field's
		// explanation. Kubernetes treats these lines as the field's value.
		var previous string
		for offset, text := range resource.lines {
			line := resource.start + offset
			if note := notes[line]; note != "" {
				previous = note
			} else if trimmed := strings.TrimSpace(text); previous != "" && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				notes[line] = previous
			}
		}
	}
	return notes
}

func collectFieldNotes(node *yaml.Node, resource manifestResource, root, schema schemaObject, path []string, parentNote string, notes map[int]string) {
	schema = resolveSchema(root, schema)
	switch node.Kind {
	case yaml.MappingNode:
		for index := 0; index < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			field := object(object(schema["properties"])[key.Value])
			if field == nil {
				field = object(schema["additionalProperties"])
			}
			fieldPath := append(path, key.Value)
			resolved := resolveSchema(root, field)
			note := fieldDescription(resource.id, fieldPath, resolved, parentNote)
			notes[resource.start+key.Line-1] = note
			childContext := parentNote
			if _, described := resolved["description"].(string); described || len(fieldPath) == 1 {
				childContext = note
			}
			collectFieldNotes(value, resource, root, field, fieldPath, childContext, notes)
		}
	case yaml.SequenceNode:
		for _, value := range node.Content {
			notes[resource.start+value.Line-1] = parentNote
			collectFieldNotes(value, resource, root, object(schema["items"]), path, parentNote, notes)
		}
	}
}

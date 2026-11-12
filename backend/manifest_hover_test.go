package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestManifestHoverUsesSchemaAndExplainsCustomKeys(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "v1.37.0-standalone-strict")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	schema := `{
	  "properties": {"metadata": {"$ref":"#/definitions/metadata"}, "data":{"additionalProperties":{"type":"string"}}},
	  "definitions": {"metadata":{"properties":{"name":{"description":"Names this resource within its namespace."}}}}
	}`
	if err := os.WriteFile(filepath.Join(dir, "configmap-v1.json"), []byte(schema), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  labels:
    app: example
data:
  settings.yaml: |
    mode: default
`
	notes := manifestHoverNotes(manifest, root, "1.37")
	for line, expected := range map[int]string{5: "Names this resource", 7: "Label “app”", 9: "Configuration entry “settings.yaml”", 10: "Configuration entry “settings.yaml”"} {
		if !strings.Contains(notes[line], expected) {
			t.Errorf("line %d: got %q, want %q", line, notes[line], expected)
		}
	}
}

func TestEveryRenderedMappingFieldHasHover(t *testing.T) {
	chartDir := pinnedChartFixtures(t)
	for tool := range charts {
		for _, version := range []string{"1.35", "1.36", "1.37"} {
			t.Run(tool+"/"+version, func(t *testing.T) {
				manifest, err := render(context.Background(), chartDir, renderRequest{Tool: tool, KubernetesVersion: version})
				if err != nil {
					t.Fatal(err)
				}
				notes := manifestHoverNotes(manifest, "../.runtime/schemas", version)
				for _, resource := range resourcesForTest(t, manifest) {
					var document yaml.Node
					if err := yaml.Unmarshal([]byte(strings.Join(resource.lines, "\n")), &document); err != nil {
						t.Fatal(err)
					}
					var check func(*yaml.Node)
					check = func(node *yaml.Node) {
						if node.Kind == yaml.MappingNode {
							for i := 0; i < len(node.Content); i += 2 {
								field := node.Content[i]
								line := resource.start + field.Line - 1
								if notes[line] == "" {
									t.Errorf("%s %s field %s lacks hover at line %d", resource.id.Kind, resource.id.Name, field.Value, line)
								}
							}
						}
						for _, child := range node.Content {
							check(child)
						}
					}
					check(&document)
				}
			})
		}
	}
}

func TestManifestFieldHoverUsesExportedLineNumbers(t *testing.T) {
	manifest := `# Export header
---
apiVersion: v1
kind: Service
metadata:
  name: test-service
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
data:
  setting: enabled
`
	note := manifestFieldHover(manifest, t.TempDir(), "1.37", 13)
	if !strings.Contains(note, "Configuration entry “setting”") {
		t.Fatalf("got %q for second document field", note)
	}
}

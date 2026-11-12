package main

import (
	"reflect"
	"testing"
)

func TestManifestChangesLinkExactResourceRanges(t *testing.T) {
	prior := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: test
data:
  setting: old
---
apiVersion: v1
kind: Service
metadata:
  name: previous
`
	current := `# Generated YAML
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: test
data:
  setting: new
---
apiVersion: v1
kind: Service
metadata:
  name: current
`
	changes, err := manifestChanges(current, prior)
	if err != nil {
		t.Fatal(err)
	}
	expected := []resourceChange{
		{Resource: resourceID{"v1", "ConfigMap", "test", "settings"}, Type: "changed", Current: &lineRange{9, 9}, Prior: &lineRange{8, 8}},
		{Resource: resourceID{"v1", "Service", "", "current"}, Type: "added", Current: &lineRange{11, 14}},
		{Resource: resourceID{"v1", "Service", "", "previous"}, Type: "removed", Prior: &lineRange{10, 13}},
	}
	if !reflect.DeepEqual(changes, expected) {
		t.Fatalf("got %+v, expected %+v", changes, expected)
	}
}

func TestManifestChangesIgnoresReorderedResources(t *testing.T) {
	first := "---\napiVersion: v1\nkind: Service\nmetadata:\n  name: first\n"
	second := "---\napiVersion: v1\nkind: Service\nmetadata:\n  name: second\n"
	changes, err := manifestChanges(first+second, second+first)
	if err != nil || len(changes) != 0 {
		t.Fatalf("reordering produced changes %v or error %v", changes, err)
	}
}

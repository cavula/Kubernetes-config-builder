package main

import "strings"

// These descriptions summarize the resource definitions in the pinned
// Kubernetes OpenAPI schemas used by the offline validator.
var manifestSectionNotes = map[string]string{
	"ClusterRole":              "Grant the listed permissions at cluster scope.",
	"ClusterRoleBinding":       "Grant the referenced ClusterRole to the listed subjects across the cluster.",
	"ConfigMap":                "Store non-secret configuration for the workload to read.",
	"CustomResourceDefinition": "Register the custom resource kind and its validation schema with Kubernetes.",
	"DaemonSet":                "Run a copy of this Pod on each eligible node.",
	"Deployment":               "Maintain the requested replicas and roll out changes to the Pod template.",
	"Job":                      "Run Pods to completion for a finite task.",
	"Namespace":                "Create a scope for the namespaced resources in this installation.",
	"NetworkPolicy":            "Control the network traffic allowed for selected Pods.",
	"PersistentVolumeClaim":    "Request persistent storage for the workload.",
	"Role":                     "Grant the listed permissions within this namespace.",
	"RoleBinding":              "Grant the referenced role to the listed subjects within this namespace.",
	"Secret":                   "Provide sensitive configuration to the components that reference it.",
	"Service":                  "Expose the selected Pods through a stable network endpoint.",
	"ServiceAccount":           "Provide a Kubernetes identity for Pods that use this account.",
	"StatefulSet":              "Run replicas with stable identities and ordered rollout behavior.",
}

func commentManifestSections(manifest string) string {
	documents := strings.Split(manifest, "\n---\n")
	for index, document := range documents {
		_, kindLine, found := strings.Cut("\n"+document, "\nkind: ")
		if !found {
			continue
		}
		kind, _, _ := strings.Cut(kindLine, "\n")
		note := manifestSectionNotes[strings.Trim(kind, "\"' \r")]
		if note == "" {
			continue
		}
		start := strings.Index(document, "# Source: ")
		if start < 0 {
			start = strings.Index(document, "apiVersion:")
		}
		if start >= 0 {
			documents[index] = document[:start] + "# Purpose: " + note + "\n" + document[start:]
		}
	}
	return strings.Join(documents, "\n---\n")
}

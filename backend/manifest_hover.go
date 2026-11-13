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

// Reasons summarize the Kubernetes API reference and workload guidance.
func fieldWhy(id resourceID, path []string, value *yaml.Node, description string) string {
	key := path[len(path)-1]
	if id.Kind == "Service" && len(path) == 1 && key == "spec" {
		explicitType := false
		for i := 0; i+1 < len(value.Content); i += 2 {
			explicitType = explicitType || value.Content[i].Value == "type"
		}
		if !explicitType {
			return "The default ClusterIP keeps this Service inside the cluster. Use port-forward for direct local access."
		}
	}
	if len(path) == 1 {
		switch key {
		case "apiVersion":
			switch id.APIVersion {
			case "apps/v1":
				return "Use apps/v1 because it is the stable Kubernetes API that manages " + id.Kind + " workloads."
			case "v1":
				return "Use v1 because " + id.Kind + " is provided by the stable core Kubernetes API."
			default:
				return "Use " + id.APIVersion + " because it is the stable API group and version that provides " + id.Kind + "."
			}
		case "kind":
			if purpose := manifestSectionNotes[id.Kind]; purpose != "" {
				return purpose
			}
			return description
		case "metadata":
			return "Give this " + id.Kind + " a stable identity so Kubernetes can find it and reconcile later applies."
		case "spec":
			return "Declare the desired configuration so Kubernetes can reconcile the actual " + id.Kind + " against it."
		}
	}
	if id.Kind == "ClusterRoleBinding" && len(path) == 1 {
		switch key {
		case "subjects":
			return "Identify the users, groups, or service accounts that receive these cluster-wide permissions."
		case "roleRef":
			return "Select the ClusterRole that defines the permissions granted to these subjects."
		}
	}
	if len(path) == 2 && path[0] == "spec" {
		if id.Kind == "Service" && key == "type" && value.Value == "ClusterIP" {
			return "Keep the Service inside the cluster. Use port-forward for direct access from your machine."
		}
		if id.Kind == "Deployment" && key == "strategy" {
			for i := 0; i+1 < len(value.Content); i += 2 {
				if value.Content[i].Value == "type" && value.Content[i+1].Value == "Recreate" {
					return "Stop the old Pods before starting replacements so they do not compete for the same volume."
				}
			}
		}
	}
	if (id.Kind == "ClusterRole" || id.Kind == "Role") && path[0] == "rules" {
		switch key {
		case "rules":
			return "Grant the component the listed operations on the Kubernetes object types and API groups it needs."
		case "resources":
			return "Choose which Kubernetes object types this rule grants access to."
		case "verbs":
			return "Limit the operations allowed on those objects, such as reading or watching them."
		}
	}
	switch key {
	case "args":
		return "Pass these flags to configure how the container starts and runs."
	case "labels":
		return "Keep labels consistent so selectors can match the intended Pods and tooling can identify the release."
	case "namespace":
		if len(path) == 2 && path[0] == "metadata" {
			return "Keep the release’s namespaced resources in " + value.Value + " so their names and permissions are scoped together."
		}
	case "name":
		if len(path) == 2 && path[0] == "metadata" {
			return "Use the chart-generated name " + value.Value + " so Kubernetes and any resources that reference it agree on its identity."
		}
	case "readinessProbe":
		return "Keep this Pod out of Service traffic until the container can serve requests."
	case "livenessProbe":
		return "Let the kubelet restart a container that stops responding to its health check."
	case "startupProbe":
		return "Give the container time to start before liveness and readiness checks take over."
	case "securityContext":
		return "Constrain the identity and privileges of this workload so it runs with the access it needs."
	case "resources":
		if len(path) < 2 || (path[len(path)-2] != "containers" && path[len(path)-2] != "initContainers") {
			return description
		}
		return "Give the scheduler capacity requirements and set bounds on runtime resource use where limits are configured."
	case "requests":
		return "Reserve capacity when Kubernetes schedules this workload."
	case "limits":
		return "Bound resource use so this container cannot consume more than the configured limits."
	case "ports":
		return "Declare the endpoints this component exposes so clients, Services, or probes can target the intended port."
	case "storage":
		return "Request this capacity so the workload has space for its persistent data."
	case "replicas":
		if value.Value == "1" {
			return "Keep one replica running to meet this configuration’s availability and capacity target."
		}
		return "Keep " + value.Value + " replicas running to meet this configuration’s availability and capacity target."
	case "runAsNonRoot":
		if value.Value == "true" {
			return "Require a non-root identity so the container cannot start with root privileges."
		}
	case "runAsUser", "runAsGroup":
		return "Use the chart’s numeric identity (" + value.Value + ") so process and file permissions are predictable."
	case "readOnlyRootFilesystem":
		if value.Value == "true" {
			return "Keep the root filesystem read-only to limit where the container can write."
		}
	case "allowPrivilegeEscalation":
		if value.Value == "false" {
			return "Prevent the process from gaining privileges beyond those of its parent."
		}
	case "initialDelaySeconds":
		return "Wait " + value.Value + " seconds after startup before the probe starts checking the container."
	case "periodSeconds":
		return "Run this probe every " + value.Value + " seconds to detect a health change."
	case "timeoutSeconds":
		return "Limit each health check to " + value.Value + " seconds so a stalled endpoint can count as a failed probe."
	case "targetPort":
		return "Send Service traffic to port " + value.Value + " on the selected Pods."
	case "containerPort":
		return "Document port " + value.Value + " as an endpoint exposed by this container."
	case "mountPath":
		return "Make the mounted volume available at " + value.Value + " so the container can read or write it there."
	case "volumeMounts":
		return "Connect these container paths to Pod volumes so configuration or persistent data is available to the workload."
	case "matchLabels":
		return "Match these labels so the controller manages only the intended Pods."
	case "image":
		return "Run the implementation and version named by this pinned chart image reference."
	case "rules":
		if id.Kind == "ClusterRole" || id.Kind == "Role" {
			return "List the API operations this component needs. A binding grants these permissions to the selected identity."
		}
	case "apiGroups":
		return "Match the API groups that own the resources in this rule. An empty string means the core Kubernetes API."
	case "hostPID":
		if value.Value == "true" {
			return "Share the node process namespace so node exporter can observe host processes for node-level metrics."
		}
		return "Keep the Pod in its own process namespace to isolate it from host processes."
	case "hostNetwork":
		if value.Value == "true" {
			return "Use the node network namespace so this Pod can access the node’s network interfaces directly."
		}
		return "Keep the Pod on its own network namespace to avoid sharing the node’s network stack."
	}
	return description
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
			note := fieldWhy(resource.id, fieldPath, value, fieldDescription(resource.id, fieldPath, resolved, parentNote))
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

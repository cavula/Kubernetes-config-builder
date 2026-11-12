# Explanations in exported configurations

The selected template exports either commented Helm values or explained Kubernetes YAML. The Argo CD YAML includes its CRDs, with their schemas collapsed in the preview. Both tools bootstrap their target namespaces in the plain YAML.

Use pinned chart notes and Kubernetes API schema descriptions to explain why settings are present. Keep literal payloads and CRD schemas intact. Explain any credentials that a user must supply, including conditional credentials, and distinguish them from credentials the installation generates.

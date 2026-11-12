# Explanations in exported configurations

The main export is the short Helm values file. The optional rendered view offers a commented workload export and, where applicable, a separate CRD file. Both tools bootstrap their target namespaces in the plain YAML.

Use pinned chart notes and Kubernetes API schema descriptions to explain why settings are present. Keep literal payloads and CRD schemas intact. Explain any credentials that a user must supply, including conditional credentials, and distinguish them from credentials the installation generates.

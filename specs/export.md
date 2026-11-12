# Export

Approved November 11, 2026.

## Inputs

The current configuration and the selected copy or download action.

## Outputs

A short Helm values file as the primary export, plus optional explained workload YAML and separate CRDs.

## Acceptance criteria

- [x] One click copies the selected export and reports success or clipboard failure.
- [x] One click downloads `<tool>-values.yaml`. The optional rendered view also offers `<tool>-manifests.yaml` and, for Argo CD, `argo-cd-crds.yaml`.
- [x] The Helm values export remains available while the rendered view or comparison is open.
- [x] Validation warnings and errors leave both actions available whenever generated content exists.
- [x] Both actions work without a live cluster or Git credentials.
- [x] The Argo CD manifests file includes the namespace and Redis initialization Job with its permissions. It explains why the Job must run before dependent workloads start.

# Export

Approved November 11, 2026.

## Inputs

The current configuration and the selected copy or download action.

## Outputs

Explained YAML or an offline Helm ZIP containing the pinned chart archive, commented values, and a README. The Argo CD YAML file includes its CRDs, which are collapsed in the preview.

## Acceptance criteria

- [x] One click copies the selected export and reports success or clipboard failure.
- [x] One click downloads `<tool>-helm-bundle.zip` for Helm chart + values or `<tool>-manifests.yaml` for YAML manifest.
- [x] Copy and Download follow the Template selection, including while comparison is open.
- [x] Validation warnings and errors leave both actions available whenever generated content exists.
- [x] Both actions work without a live cluster or Git credentials.
- [x] The Argo CD manifests file includes the namespace and Redis initialization Job with its permissions. It explains why the Job must run before dependent workloads start.

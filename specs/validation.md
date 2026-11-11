# Validation

Approved November 11, 2026.

## Inputs

The latest configuration revision, generated Helm values and manifests, the pinned chart, Kubernetes minor, and locally available schemas and compatibility metadata.

## Outputs

Automatic validation status and diagnostics with severity, explanation, and the associated option or YAML location when available.

## Acceptance criteria

- [x] Editing options or changing the selection starts validation automatically. Results from older revisions cannot overwrite current diagnostics.
- [x] With no cluster or kubeconfig, the app runs Helm lint against the pinned chart and values and kubeconform against the rendered manifests using schemas for the selected Kubernetes minor, including applicable custom resources.
- [ ] An invalid fixture shows an actionable error. A chart or tool version mismatch names the tool, selected Kubernetes minor and supported range when known.
- [x] Missing schemas and validator failures show an incomplete check or error. A clean result requires completed checks.
- [x] Copy and download remain available for generated content with warnings or errors.
- [ ] In the dedicated local K3s test environment, each supported tool and minor combination passes Helm lint and kubeconform with zero errors, passes server-side dry run, and deploys cleanly on a fresh cluster with workloads ready. Tests cover defaults and approved settings that change resource creation. Where custom resources are included, dry run uses registered CRDs.

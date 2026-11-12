# Export and viewer polish

## Inputs

The selected tool, Kubernetes version, preset, values, and template.

## Outputs

Explained YAML, an offline Helm ZIP, and component sections within the existing three-step flow.

## Acceptance criteria

- [x] Preset defaults to Minimal for Prometheus and fits its card. Actions have normal button sizes, examples appear only on typed inputs, and sticky YAML headers have a solid background and border.
- [x] Component links open and jump to their collapsible YAML sections.
- [x] Copy and Download use the same explained YAML with install and after-install headers. Test the actual clipboard and download for both tools and the requested Kubernetes fields.
- [x] Helm chart + values copies commented values and downloads a ZIP with those same values, the verified pinned archive, and a README with the local install command and after-install steps. Helm can render the archive without downloading dependencies.
- [x] Apply the exact YAML downloads to fresh K3s and record workload readiness, pod status and events if a rollout stalls.

## Verification

Fresh K3s 1.37.2 applied and server-dry-ran the explained exports for Prometheus Minimal and Argo CD defaults. All workloads rolled out using the ten-minute timeout. The Argo CD watch recovered after an HTTP/2 disconnect.

## Sources

- [Helm install](https://helm.sh/docs/helm/helm_install/)
- [Kubernetes Pod API reference](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/)
- [Kubernetes RBAC reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

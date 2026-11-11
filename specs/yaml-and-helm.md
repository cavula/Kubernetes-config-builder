# YAML and Helm switch

Approved November 11, 2026.

## Inputs

One configuration, its pinned chart and Kubernetes target, a selected primary format, and optionally a selected Helm option to inspect.

## Outputs

Commented plain Kubernetes manifests or a commented Helm values file for the same configuration. A side-by-side comparison links values to the fields and resources they affect in rendered YAML.

## Acceptance criteria

- [ ] Switching formats preserves the cluster, tool and option values.
- [ ] Rendering the Helm values with the same chart pin, release name, namespace and Kubernetes capabilities yields the same desired resources and field values as plain YAML, accounting for comments and formatting.
- [ ] For each approved option, the comparison shows all affected YAML ranges. Added resources are linked to their current ranges and removed resources are identified using their prior YAML.
- [ ] When a conditional option has no effect in the current configuration, the comparison explains why.
- [ ] The comparison keeps an explicit primary format for export.

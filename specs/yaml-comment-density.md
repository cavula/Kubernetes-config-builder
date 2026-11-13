# YAML comment density

Inputs are the existing configuration and the Learning or Compact density. Output is the same Kubernetes resources and installation header in either mode.

- [x] Learning is the default. Short, authored explanations cover key fields and appear once per block and without repeated identical comments in each resource.
- [x] Explain RBAC permissions using the resource kind and field path. Distinguish API resource types from container capacity.
- [x] Compact keeps component and purpose headers and deployment instructions. It omits field explanations.
- [x] Preview, copy, and download use the selected density. Minimal Prometheus Learning YAML is at most 650 lines.
- [x] Selection hints end with single periods and the Minimal label fits.

Minimal Prometheus on Kubernetes 1.37 went from 856 lines to 650 with the same chart and values.

Sources include the [Kubernetes RBAC reference](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) and [container resource management guide](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).

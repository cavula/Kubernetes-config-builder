# Cluster and tool selection

Approved November 11, 2026.

## Inputs

- Platform: K3s.
- Kubernetes minor: 1.37, 1.36, or 1.35 at the planning baseline. The release catalog uses the [K3s stable channel](https://raw.githubusercontent.com/k3s-io/k3s/main/channel.yaml) minor and its two immediate predecessors.
- One tool: Prometheus or Argo CD.

## Outputs

A configuration context with platform, Kubernetes minor, tool, and its chart pin. The pins are [Prometheus 29.45.1](https://github.com/prometheus-community/helm-charts/tree/prometheus-29.45.1/charts/prometheus) and [Argo CD 10.17.1](https://github.com/argoproj/argo-helm/tree/argo-cd-10.17.1/charts/argo-cd).

## Acceptance criteria

- [x] Each of the six tool and Kubernetes minor combinations opens its builder with the appropriate chart and defaults.
- [x] Selecting the other tool replaces the active configuration with that tool’s defaults. The Kubernetes selection stays in place.
- [x] Users can inspect the chart pin. The selector offers exactly one chart per tool.
- [x] Selection works without a cluster connection and rejects unsupported platform, version, and tool identifiers.

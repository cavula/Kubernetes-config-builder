# Preset and deploy flow

Approved by Charan on November 12, 2026.

## Inputs

K3s target version, tool, preset, optional chart settings and add-on switches, and the selected template.

## Outputs

Live previews, a component summary, validation results, one copy or download for the selected template, pinned installation commands, and an optional Helm-to-YAML comparison.

## Acceptance criteria

- [x] Show Choose a preset, Customize (optional), and Deploy. Place the preset beside the tool and Kubernetes version in step 1.
- [x] Offer Minimal (Prometheus server only, best for learning), Recommended, and Full chart defaults using the pinned official Prometheus chart values. Argo CD uses its chart defaults.
- [x] Give Alertmanager, kube-state-metrics, node exporter, and Pushgateway independent switches in labelled sections.
- [x] Show the namespace and main Prometheus resources before any add-ons in the YAML export. Each enabled add-on has a labelled section.
- [x] Show guidance and an example for every customization input.
- [x] Provide one Template dropdown with Helm chart and YAML manifest. Show Copy and Download for the selected template, with Compare Helm and Kubernetes YAML manifest directly below.
- [x] Show the pinned install commands in Deploy and put the same commands at the top of the downloaded file.
- [x] Explain indentation and show each full setting name and its meaning in Helm values.
- [x] Widen comparison panes, wrap comments, and show one line-number gutter per pane.
- [x] Summarize included components and their approximate rendered line counts.
- [x] Collapse Argo CD CRDs in the preview and include them in its single-file YAML export.

## Sources

- [Pinned Prometheus chart](https://github.com/prometheus-community/helm-charts/tree/prometheus-29.45.1/charts/prometheus)
- [When to use Pushgateway](https://prometheus.io/docs/practices/pushing/)
- [kubectl apply reference](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_apply/)

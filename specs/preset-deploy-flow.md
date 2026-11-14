# Preset and deploy flow

Approved by Charan on November 12, 2026.

## Inputs

K3s target version, tool, preset, optional chart settings and add-on switches, and the selected template.

## Outputs

Live previews, a component summary, validation results, one copy or download for the selected template, pinned installation commands, and an optional Helm-to-YAML comparison.

## Acceptance criteria

- [x] Show Choose a preset, Customize (optional), and Deploy. Place the preset beside the tool and Kubernetes version in step 1.
- [x] Offer Minimal (Prometheus server only, best for learning), Recommended, and Full chart defaults using the pinned official Prometheus chart values.
- [x] Give Alertmanager, kube-state-metrics, node exporter, and Pushgateway independent switches in labelled sections.
- [x] Show the namespace and main Prometheus resources before any add-ons in the YAML export. Each enabled add-on has a labelled section.
- [x] Show guidance for customization controls and examples for typed inputs.
- [x] Provide one Template dropdown with Helm chart + values and YAML manifest. Show Copy and Download for the selected template, with Compare Helm and Kubernetes YAML manifest directly below.
- [x] Show the pinned install commands in Deploy and put the same commands at the top of the downloaded file.
- [x] Explain indentation and show each full setting name and its meaning in Helm values.
- [x] Widen comparison panes, wrap comments, and show one line-number gutter per pane.
- [x] Summarize included components and link their approximate line counts to collapsible YAML sections.
- [x] Collapse Argo CD CRDs in the preview and include them in its single-file YAML export.

## Sources

- [Pinned Prometheus chart](https://github.com/prometheus-community/helm-charts/tree/prometheus-29.45.1/charts/prometheus)
- [When to use Pushgateway](https://prometheus.io/docs/practices/pushing/)
- [kubectl apply reference](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_apply/)

## Argo CD presets

- [x] Default Argo CD to Minimal. Disable Dex and notifications with their chart switches. Scale ApplicationSet to zero because the pinned chart always renders its Deployment.
- [x] Recommended enables ApplicationSet for multi-application workflows and leaves Dex and notifications off. Full chart defaults uses the pinned defaults.
- [x] CRD schemas start collapsed in the YAML preview. Verify this in Chromium.
- [x] Measure Learning YAML lines excluding CRD documents. Enforce a Minimal line budget.

On Kubernetes 1.37, Learning YAML including the header and excluding CRD documents is 3,442 lines for Minimal, 3,442 for Recommended, and 4,220 for Full. The Minimal budget is 3,600 lines. [Pinned chart values](https://github.com/argoproj/argo-helm/blob/argo-cd-10.17.1/charts/argo-cd/values.yaml)

Argo CD now offers Minimal and Full chart defaults. A zero-replica ApplicationSet Deployment explains why the pinned chart still renders it. The Minimal limit stays at 3,600 lines.

This spec covers the secondary **Install the tool** workflow. See [Usage templates](usage-templates.md) for the primary product direction.

In the Argo CD YAML preview, long container environment lists start folded within the editor. Keep a plain comment above each folded list explaining that the settings configure the container and that its references to `argocd-cmd-params-cm` are optional. A list can also contain settings supplied by other sources. Copy and download retain the complete generated manifest.

# Kubernetes Config Builder and Learning Dashboard

## Overview

A dashboard that helps engineers generate correct Kubernetes configs instantly and helps students learn how those configs work. Users pick a cluster, pick tools, choose options, and watch commented YAML or Helm values update live.

## Problem

Writing configs for Kubernetes tools is hard. People jump between official docs, miss indentation, hit linting errors, and often do not understand why a config is written the way it is. Tools and Kubernetes versions also change often, and some projects retire, such as Ingress NGINX.

## Target Users

Engineers who want fast, reliable templates. Students who want to learn by changing options and seeing the code change.

## Core Features

**Cluster selection.** Self hosted, K3s, OpenShift, Tanzu, AKS, GKE, or EKS, plus the Kubernetes version.

**Tool selection.** Prometheus, Istio, Cilium, Argo, Flux, OpenTelemetry, Kyverno, and Backstage, as single tools or ready made bundles.

**Live builder.** Options on the left, code on the right. Every section has a comment explaining what it is and what it does. Changed lines are highlighted, and hovering over a line shows a plain English explanation.

**YAML and Helm.** A switch between plain YAML manifests and Helm values, with a side by side view showing how a Helm value turns into full YAML.

**Cluster aware and validated.** Templates are tuned to the chosen platform, linted, checked against the schema, and flagged for version or tool conflicts.

**Explain my code.** Users paste or upload their own YAML. The dashboard explains each section, sets the options panel to match the file, and flags anything risky or outdated.

**Export.** Download the file or push it to Git for Argo or Flux.

**Auto updates.** MCP based jobs check official GitHub repos, releases, and Artifact Hub every two to four weeks. Changes are summarised, tested, and reviewed before going live. Deprecated or retired tools are flagged with suggested replacements.

## MVP Scope

Local app first. K3s only, with Kubernetes 1.37, 1.36, and 1.35 at the initial planning baseline. Include Prometheus and Argo CD, with one tool and one pinned chart per configuration. Include the live builder, YAML and Helm switch with resource-level and field-level comparison, automatic validation, and copy and download of the active format. Build, validate, and export without a live cluster. Run server-side dry run and deployment checks against a dedicated local K3s test cluster.

## Roadmap

1. MVP on K3s.
2. Explain my code mode.
3. More tools and bundles.
4. Other platforms: EKS, AKS, GKE, OpenShift, Tanzu.
5. Auto updates from GitHub and Artifact Hub.
6. Later: drift detection and upgrade guidance.

## Development Approach

Spec driven development. Every feature starts as a written spec with clear inputs, outputs, and acceptance criteria before any code is written.

## Tech Stack

| Layer | Choice | Why |
| --- | --- | --- |
| Frontend | React with TypeScript, built with Vite on Node.js 26.13.1 LTS | Widely supported, strong for live UIs |
| Code editor | Monaco (the VS Code editor) | Built in line highlighting and hover notes |
| Backend | Go 1.27.4 | Same language as Kubernetes and Helm, so the Helm SDK can render charts directly |
| Validation | kubeconform, Helm lint, server side dry run | Catches schema, lint, and cluster errors before deploy |
| Test cluster | K3s, run locally | Lightweight and close to real Kubernetes |
| CI | GitHub Actions | Runs build, tests, and security scans on every change |

## Folder Structure

```
/frontend          React app: options panel, Monaco editor, diff highlights, hover notes
/backend           Go API: render templates, validate, explain uploaded YAML
/templates         One folder per tool, per version
  /prometheus
    /yaml          Plain manifests with section comments
    /helm          values.yaml with section comments
    options.json   Options shown in the left panel and the lines each one changes
    notes.json     Plain English hover explanations per field
  /argo-cd
    /yaml
    /helm
    options.json
    notes.json
/specs             One spec file per feature, written before code
/tests             Unit tests, plus deploy tests against K3s
/docs              User and contributor guides
```

## Acceptance Criteria (MVP)

**Cluster and tool selection**

- [x] User can select K3s and a Kubernetes version.
- [x] User can select Prometheus or Argo CD.

**Live builder**

- [ ] Changing any option updates the code on the right within 1 second.
- [ ] Lines that changed are highlighted.
- [ ] Hovering over any field shows a plain English explanation.
- [ ] Every section of generated code has a comment saying what it is and what it does.

**YAML and Helm**

- [x] A switch shows the same setup as plain YAML or as Helm values.
- [x] A side by side view shows which Helm value produced which YAML lines.

**Validation**

- [x] Generated code passes kubeconform and Helm lint with zero errors.
- [ ] Generated code deploys cleanly to a fresh K3s cluster.
- [ ] A version mismatch between tool and Kubernetes shows a clear warning.

**Export**

- [x] User can download the file or copy it with one click.

**Security**

- [ ] Dependency and container scans show no high or critical issues.
- [x] Generated configs use secure defaults, such as no privileged containers unless the user asks.

## Phases

**Plan.** Write specs, define MVP scope, choose the tech stack.

**Develop.** Build features against the specs.

**Build.** Package the app and templates, set up CI.

**Test.** Check generated configs deploy cleanly on K3s and match the specs.

**Security review.** Threat modelling, dependency scanning, and checks that generated configs follow secure defaults.

**Deploy.** Release the MVP, gather feedback, then move to the next roadmap step.

## Open Questions

The five MVP specs and builder options are approved. Development starts with cluster and tool selection.

## Instructions for the Coding Agent

1. Read this whole file before writing any code.
2. Build only the MVP Scope. Do not start roadmap items.
3. For each feature, write a spec in /specs first, then the code, then tests.
4. Follow the Folder Structure and Tech Stack exactly. Ask before changing either.
5. Take every template field and explanation from the official project docs or Helm charts. Never invent fields.
6. Every generated section must carry a comment saying what it is and what it does.
7. After each feature, run lint, kubeconform, and a deploy test on K3s, and tick the matching Acceptance Criteria.
8. Keep commits small, with clear messages, so progress can be reviewed.
9. If anything is unclear, stop and ask Charan, the product owner, rather than guess. Record each answer in the relevant spec so this document stays up to date.

## Keeping Decisions in Sync

This file and /specs are the single source of truth. Chat instructions never override them silently.

1. When Charan gives a new instruction in chat, first check it against this file and the specs.
2. If it conflicts with anything, point out the conflict and confirm which one wins before coding.
3. Once confirmed, update this file and the affected specs first, then change the code.
4. Add an entry to the Decision Log below for every change.
5. At the start of every session, reread this file and the Decision Log before doing anything.

## Decision Log

| Date | Decision | Replaces | Affects |
| --- | --- | --- | --- |
| 2026-11-11 | Include both tools, one per configuration. Mirror Prometheus’s template folder for Argo CD. | One or two tools and a Prometheus-only example tree | Scope, selection, templates |
| 2026-11-11 | Offer the K3s stable channel’s Kubernetes minor and the two preceding minors. Initial baseline is 1.37, 1.36, 1.35. Pin Prometheus chart 29.45.1 and Argo CD chart 10.17.1. | Unspecified Kubernetes and chart versions | Selection, rendering, validation |
| 2026-11-11 | Ship a local app first. Build, validate, and export without a live cluster. Use a dedicated local K3s test cluster for server-side dry run and deployment checks. | Open hosting and cluster access questions | Scope, validation |
| 2026-11-11 | Propose five to eight common chart settings per tool for approval. Map each approved option to changed YAML, including added and removed resources. | Unspecified builder options and comparison behavior | Live builder, YAML and Helm |
| 2026-11-11 | Validate automatically during edits. Show errors while keeping copy and download available for generated content. Export the active format only. | Unspecified validation timing and export behavior | Validation, export |
| 2026-11-11 | Keep the documented stack and use Vite with [Node.js 26.13.1 LTS](https://nodejs.org/en/blog/release/v26.13.1) and [Go 1.27.4](https://go.dev/doc/devel/release#go1.27.4). | Unspecified build tool and runtime versions | Tech stack |
| 2026-11-11 | Approved all five specs. Add Prometheus scrape interval and Argo CD server service type (ClusterIP or NodePort). Begin development with cluster and tool selection. | Pending spec and options approval | Specs, live builder |
| 2026-11-11 | Verified Kubernetes minors and chart and runtime pins against official sources. [K3s stable channel](https://raw.githubusercontent.com/k3s-io/k3s/main/channel.yaml), [K3s releases](https://github.com/k3s-io/k3s/releases), [Prometheus 29.45.1 chart](https://github.com/prometheus-community/helm-charts/tree/prometheus-29.45.1/charts/prometheus), [Argo CD 10.17.1 chart](https://github.com/argoproj/argo-helm/tree/argo-cd-10.17.1/charts/argo-cd), [Go 1.27.4](https://go.dev/doc/devel/release#go1.27.4), [Node.js 26.13.1 LTS](https://nodejs.org/uk/blog/release/v26.13.1). | Planning verification | Release catalog and toolchain |
| 2026-11-11 | Pin Node.js 26.13.1 in root and frontend version files and enforce the exact version during dependency installation. | Frontend-only version file and semver range | Local development |
| 2026-11-11 | Override `configmapReload.prometheus.image.tag` and `alertmanager.configmapReload.image.tag` to v0.95.2 for Prometheus chart 29.45.1. [v0.95.1 image index](https://quay.io/v2/prometheus-operator/prometheus-config-reloader/manifests/sha256:39bdbade0c64f7c38bb6d18d6f602b14ad489bc189b709dc9266b49e6647aac1) lists only linux/amd64 and attestations. [v0.95.2 image index](https://quay.io/v2/prometheus-operator/prometheus-config-reloader/manifests/sha256:cc74ddedad4ab1c49446971f6d51302ad4303d8660a2018af925c7f62c114304) includes linux/arm64. Check the upstream image index when changing the chart pin and remove the override once the chart defaults support ARM64. | Reloader tag v0.95.1 from the pinned chart and Alertmanager dependency | Helm values, plain YAML, CI, K3s |
| 2026-11-11 | Include the Argo CD namespace and the pinned chart’s Redis pre-install Job and permissions in plain YAML. Apply with kubectl server-side apply to support the chart’s large CRDs. | Helm-only Redis initialization in the rendered release | YAML rendering, export, local deploy checks |
| 2026-11-11 | Keep section explanations and conditional control hints in chart notes. Run live builder and selection tests together under npm test. | Generic comments and UI-owned storage hint | Builder, tests |
| 2026-11-11 | Fetch pinned charts with the setup script and verify archive SHA256 before loading. Require fixtures for Go render tests by default, with an explicit local skip flag. | Unverified archives and implicit fixture skips | Rendering, setup, Go tests |
| 2026-11-11 | Compare a selected Helm option by rendering its current and alternate values with the same chart and Kubernetes target. Preserve the primary format for copy and download. | Preview-only format switch | Comparison, export |
| 2026-11-11 | Run Helm lint and strict kubeconform offline on edits. Report missing tools or schemas as incomplete. Run both validators and the frontend and Go tests on every push. Include chart hooks in the CI manifest check. | Manual checks and hook-free CI validation | Validation, CI |
| 2026-11-11 | Pin the [schema catalog](https://github.com/yannh/kubernetes-json-schema/commit/9e1cd3c3b22e6432cbc5615a5a930f8268f14bf1) and the Kubernetes OpenAPI definitions for [1.35](https://github.com/kubernetes/kubernetes/commit/66452049f3d692768c39c797b21b793dce80314e), [1.36](https://github.com/kubernetes/kubernetes/commit/ecf6decece6a6de25a57aad9ba90b6ce580f6f78), and [1.37](https://github.com/kubernetes/kubernetes/commit/f54c212e3a2f75d674b717a9b29052b20b60aefc) to commits. | Mutable schema catalog branch and release-tag references | Offline validation, CI |
| 2026-11-11 | Run deployment checks in isolated k3d clusters for K3s 1.35.10, 1.36.6, and 1.37.2 using project-local kubeconfigs. Put the required server-side apply command and CRD annotation limit at the top of the Argo CD YAML. | Single-minor live checks and apply instructions only in documentation | K3s tests, Argo CD export |
| 2026-11-11 | Fail CI on high or critical vulnerabilities from npm audit and Trivy. Derive the image inventory from verified chart renders across supported Kubernetes minors and Boolean setting combinations, and scan Linux AMD64 and ARM64. Report findings before changing any pins. | Manual security review without CI gates | Security review, CI |
| 2026-11-11 | Match rendered YAML resources by API version, kind, namespace, and name. Show exact current and comparison line ranges for changed, added, and removed resources. | One diff over the entire manifest stream | Helm-to-YAML comparison |
| 2026-11-11 | Pin transitive DOMPurify to 3.4.17 for [GHSA-cg2g-xh64-v4q7](https://github.com/advisories/GHSA-cg2g-xh64-v4q7) and [GHSA-mrv2-3pqr-j5jr](https://github.com/advisories/GHSA-mrv2-3pqr-j5jr). Update golang.org/x/net to v0.62.0 for [CVE-2026-78661](https://avd.aquasec.com/nvd/cve-2026-78661). High and critical application dependency findings always fail CI. | DOMPurify 3.4.15 and x/net v0.60.0 | npm, Go, CI |
| 2026-11-11 | Review official chart releases before changing chart pins, with no per-image upgrades for security findings. Prometheus 29.47.1 and Argo CD 10.18.0 retain the findings on both architectures. Keep current pins pending approval. Report exact accepted image findings in CI and display a note in the app. Review on 2026-12-11. Unaccepted or expired findings still fail CI. | Blocking every image finding, including those without an official chart fix | Security review, CI, chart exports |

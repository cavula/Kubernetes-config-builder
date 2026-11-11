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

- [ ] User can select K3s and a Kubernetes version.
- [ ] User can select Prometheus or Argo CD.

**Live builder**

- [ ] Changing any option updates the code on the right within 1 second.
- [ ] Lines that changed are highlighted.
- [ ] Hovering over any field shows a plain English explanation.
- [ ] Every section of generated code has a comment saying what it is and what it does.

**YAML and Helm**

- [ ] A switch shows the same setup as plain YAML or as Helm values.
- [ ] A side by side view shows which Helm value produced which YAML lines.

**Validation**

- [ ] Generated code passes kubeconform and Helm lint with zero errors.
- [ ] Generated code deploys cleanly to a fresh K3s cluster.
- [ ] A version mismatch between tool and Kubernetes shows a clear warning.

**Export**

- [ ] User can download the file or copy it with one click.

**Security**

- [ ] Dependency and container scans show no high or critical issues.
- [ ] Generated configs use secure defaults, such as no privileged containers unless the user asks.

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

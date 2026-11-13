# Kubernetes Config Builder and Learning Dashboard

## Overview

A dashboard that helps engineers generate correct Kubernetes configs instantly and helps students learn how those configs work. Users choose a tool and generate short, explained templates for using it. Every current and future tool leads with usage templates. The existing installation workflow belongs under the secondary **Install the tool** option.

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

## Product Scope

Local app first. K3s only, with Kubernetes 1.37, 1.36, and 1.35 at the initial planning baseline. Build the approved Argo CD Application, AppProject, and ApplicationSet templates. Propose standalone Prometheus usage templates for approval. Defer Prometheus Operator as a separate tool. Use the approved default **Use the tool** view and secondary **Install the tool** option. Explain every usage YAML line in Learning mode. Use official schemas and test the generated manifests on dedicated K3s. Preserve one tool and one pinned chart per installation configuration. Include the live builder, YAML and Helm switch with resource-level and field-level comparison, automatic validation, and copy and download of the active format. Build, validate, and export without a live cluster. Run server-side dry run and deployment checks against a dedicated local K3s test cluster.

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

- [x] Changing any option updates the code on the right within 1 second.
- [x] Lines that changed are highlighted.
- [x] Hovering over any field shows a plain English explanation.
- [x] Every section of generated code explains what it does and why it is present.

**YAML and Helm**

- [x] A switch shows the same setup as plain YAML or as Helm values.
- [x] A side by side view shows which Helm value produced which YAML lines.

**Validation**

- [x] Generated code passes kubeconform and Helm lint with zero errors.
- [x] Generated code deploys cleanly to a fresh K3s cluster.
- [x] A version mismatch between tool and Kubernetes shows a clear warning.

**Export**

- [x] User can download the file or copy it with one click.

**Security**

- [x] Our own dependencies have no high or critical issues, and upstream image findings are tracked as accepted risks with a review date.
- [x] Generated configs use secure defaults, such as no privileged containers unless the user asks.

## Phases

**Plan.** Write specs, define MVP scope, choose the tech stack.

**Develop.** Build features against the specs.

**Build.** Package the app and templates, set up CI.

**Test.** Check generated configs deploy cleanly on K3s and match the specs.

**Security review.** Threat modelling, dependency scanning, and checks that generated configs follow secure defaults.

**Deploy.** Release the MVP, gather feedback, then move to the next roadmap step.

## Open Questions

Hosting remains an open question. Where should a hosted dashboard run, and who should be able to access it?

## Instructions for the Coding Agent

1. Read this whole file before writing any code.
2. Build only the Product Scope. Do not start roadmap items.
3. For each feature, write a spec in /specs first, then the code, then tests.
4. Follow the Folder Structure and Tech Stack exactly. Ask before changing either.
5. Take every template field and explanation from the official project docs or Helm charts. Never invent fields.
6. Explain why each generated setting is present using official sources. Follow the export spec for credentials, embedded payloads, and CRDs.
7. After each feature, run lint, kubeconform, and a deploy test on K3s, and tick the matching Acceptance Criteria.
8. Keep commits small, with clear messages, so progress can be reviewed.
9. Get Charan’s approval before changing the main layout.
10. If anything is unclear, stop and ask Charan, the product owner, rather than guess. Record each answer in the relevant spec so this document stays up to date.

## Keeping Decisions in Sync

This file and /specs are the single source of truth. Chat instructions never override them silently.

1. When Charan gives a new instruction in chat, first check it against this file and the specs.
2. If it conflicts with anything, point out the conflict and confirm which one wins before coding.
3. Once confirmed, update this file and the affected specs first, then change the code.
4. Add an entry to the Decision Log below for every change.
5. At the start of every session, read this entire file, including Lessons Learned and the Decision Log, before doing any project work.
6. Whenever Charan corrects a mistake, add an entry to Lessons Learned in the same turn. Record the date, what went wrong, why it happened, and the rule that prevents it. Update the plan, relevant specs, or Decision Log when the lesson changes them.

## Lessons Learned

Dates refer to when the correction was made. The Decision Log records the resulting product and implementation decisions.

| Date | What went wrong | Why | Rule that prevents it |
| --- | --- | --- | --- |
| 2026-11-11 | Initial checks ran on Node.js 26.12.1 even though the approved project version was 26.13.1. | The version file declared the pin while the commands used the Node executable already on PATH. | Check the executable version before running tests or builds. Keep the exact version enforced and document how to activate it locally. |
| 2026-11-11 | I connected the remote without committing or pushing, and omitted the reason. | I read the repository URL as authorization to connect it and left that interpretation implicit. | State exactly what was done and what remains local. When publishing is requested, commit, push, verify the remote state, and name any blocker and the action needed to clear it. |
| 2026-11-11 | GitHub rejected the first push because the three commits used a private email address. | The commits inherited an author email incompatible with GitHub’s privacy settings. | Check the repository-local identity before committing for publication. Use Charan’s approved GitHub noreply address for author and committer. |
| 2026-11-11 | Live builder tests were omitted from the initial implementation, and I did not call out the gap. | I verified selection, type checking, and the build while the acceptance criteria required feature-level tests. | Add regression tests for feature acceptance criteria to the normal test command. State which checks are pending and leave their criteria unchecked. |
| 2026-11-11 | Generated Helm values contained generic section comments and a Prometheus-specific persistence hint in the UI. | The renderer used a fallback comment and the form owned text that belonged to the chart catalog. | Put explanations and dependency hints in each tool’s chart notes, grounded in the pinned official chart. Test that every generated field and conditional hint resolves from those notes. |
| 2026-11-11 | Rendering tests silently skipped missing charts, and the renderer loaded archives without verifying their hashes. | Tests treated absent fixtures as routine and trusted a filename and version as proof of contents. | Fail by default on missing or corrupt fixtures. Offer an explicit local skip flag. Verify archives against committed SHA-256 pins before loading, and document the verified download command. |
| 2026-11-11 | Plain Argo CD YAML omitted Redis initialization, so workloads waited for a Secret that never appeared. | The export kept Helm’s ordinary manifest output and dropped the pre-install hook resources. | Test the exact export on a fresh cluster and include any required initialization resources. Run CI manifest checks with hooks included. |
| 2026-11-11 | The Argo CD export did not lead with the server-side apply requirement. | The CRD size limit was documented separately from the YAML users actually apply. | Put mandatory apply commands and their reasons at the top of the relevant downloaded file. Test that the instructions survive export. |
| 2026-11-11 | Prometheus and Alertmanager stalled on ARM64 because the pinned reloader tag had no ARM64 image. | Schema validation and server-side dry run could not detect an architecture mismatch in the image registry. | Inspect the official image index and verify rollout readiness on the target architecture. Get approval before changing chart or image pins and record the evidence and retirement condition for any override. |
| 2026-11-11 | The schema downloader still used a mutable catalog branch, and repository documentation and acceptance records lagged behind the implementation. | Chart pins were locked first while schema provenance, the README, and the checklist were treated as follow-up work. | Pin schema inputs to commits, maintain setup instructions and the Decision Log alongside the code, and tick only verified criteria. Check CI for the exact full commit SHA before reporting its result. |
| 2026-11-11 | Browser checks existed only as ad hoc commands, and rendered YAML hover coverage was partial. | Browser evidence was gathered manually and field explanations covered selected lines before the full manifest was tested. | Commit repeatable browser tests and run them in CI. Test every rendered field for hover coverage and verify real preview latency and interactions in Chromium. |
| 2026-11-12 | Prometheus YAML omitted the monitoring namespace, so a fresh cluster could not apply the downloaded file directly. | Earlier deployment checks supplied the namespace separately and masked the missing resource. | Apply the exact downloaded YAML to a fresh cluster without precreating its namespace. Test that the namespace appears before dependent resources. |
| 2026-11-12 | The v0.1.0 downloads described fields without consistently explaining why they were present. Credential guidance and post-install steps were missing, and CRD internals crowded the output. | Review emphasized valid rendering and field definitions while the learning goal was not checked against the actual downloads. | Review the files a user receives. Use the selected template for copy and download. Keep CRD schemas collapsed or in a separate file. Explain why meaningful values are chosen using official sources, mark user-supplied credentials with their purpose and sourcing instructions, and test the download headers and resource explanations. |
| 2026-11-12 | I kept implementation decisions but did not record the mistakes Charan corrected as reusable lessons. | The workflow required a Decision Log and had no place to preserve causes and prevention rules. | Record each correction in Lessons Learned in the same turn. Reread the section and Decision Log at the start of every session and update affected project guidance. |
| 2026-11-12 | I wrote the export lesson as a permanent rule to lead with Helm values. | I turned feedback about the length and quality of one export into a default for the main interface. | Do not change the main layout without asking Charan first. Use the template selection to determine copy and download output. |
| 2026-11-12 | Charan received YAML without field-level why comments or the full deployment header. | Preview and export used separate manifest paths, and browser checks covered only the Argo CD download. The exact path used in Charan’s test has not been reproduced. | Use the explained export for preview, clipboard, and file download. Assert the exact clipboard and downloaded contents for both tools, including representative field explanations and the deployment header. |
| 2026-11-12 | ClusterRole rules described API object types as CPU and memory settings, and exports repeated long schema descriptions. | Explanations matched leaf names and traversed every list entry with a schema fallback. | Match overloaded fields by kind and parent path. Test the downloaded YAML for RBAC wording, duplicate comments, and the Minimal line budget. |
| 2026-11-12 | Learning YAML omitted probe and mount explanations after the first container and used plural wording for a single replica. | Deduplication used the whole resource and replica text assumed a plural count. | Verify comments on each container block and test both replica counts. |

| 2026-11-13 | v0.2.0 was tagged before the final review was confirmed, so later Learning fixes missed that release. | The tag was cut before the review cycle was closed. | Only tag a release after Charan confirms the final review. |

| 2026-11-13 | The CRD preview default relied on section order. | The first section opened automatically without explicitly excluding schemas. | Default CRDs to collapsed explicitly and verify each preset in Chromium. |

| 2026-11-13 | Recommended added a running ApplicationSet controller while producing the same number of lines as Minimal. Minimal left the zero-replica Deployment unexplained. | The chart renders the same Deployment and supporting resources at either replica count. | Check each preset’s rendered resources and explain any unavoidable stopped workload. |

| 2026-11-13 | We built installation manifests when Charan wanted templates for using the tools. | The product plan treated installation as the main task. | For every tool, agree on the usage templates first and lead with short, explained usage YAML. Keep installation secondary. |

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
| 2026-11-11 | Keep Prometheus chart 29.45.1 after comparing it with 29.47.1. Both show 35 distinct high-severity image repository, package, and advisory combinations on each architecture, with no findings resolved or added. | Conditional chart upgrade pending scan comparison | Prometheus, security review |
| 2026-11-11 | Classify Kubernetes compatibility mismatches from the chart's declared version range. Show the chart, selected minor, and supported range in a warning. | Matching Helm error text for compatibility warnings | Validation |
| 2026-11-11 | Verify plain YAML against Helm CLI resource objects for both tools across all three Kubernetes minors, with defaults and resource-changing options. Argo CD plain YAML additionally bootstraps its namespace. | Unverified equivalence | Rendering and export |
| 2026-11-11 | Include a purpose comment for each rendered Kubernetes resource and expose those explanations on kind fields. Run live default and resource-changing fixtures in fresh dedicated K3s clusters across all three minors. | Partial manifest comments and incomplete deploy matrix | Live builder, deployment checks |
| 2026-11-11 | Use the security acceptance criterion “Our own dependencies have no high or critical issues, and upstream image findings are tracked as accepted risks with a review date.” | Requiring a zero-finding scan across application dependencies and upstream images | MVP acceptance, security review |
| 2026-11-11 | Run the committed browser tests in CI. Explain rendered YAML fields using pinned Kubernetes schema descriptions and contextual notes for chart-defined entries. | Manual browser checks and hover explanations limited to values and selected manifest lines | Live builder, tests, CI |
| 2026-11-12 | Package v0.1.0 as a local Docker Compose release with verified charts and schemas. Reopen the hosting decision. | Developer-tool setup as the only local run path | Deploy, README, CI |
| 2026-11-12 | Build the bundled Helm and kubeconform commands from the same pinned Go module and toolchain as the dashboard. Scan the completed Docker image in CI. | Shipping validator binaries with a separate dependency graph | Deploy, security review |
| 2026-11-12 | Make short Helm values the primary download. Offer an optional rendered view with explained manifests and separate CRDs. Include both namespaces in plain YAML and document post-install access from the pinned charts and official docs. Mark conditional credentials with the expected value, source, and Git warning. | Raw manifest download and Prometheus namespace prerequisite | Export, builder, deployment |
| 2026-11-12 | Record every mistake Charan corrects in Lessons Learned and read it at the start of every session. Update any affected plan, spec, or decision alongside the lesson. License the project under MIT with Charan as copyright holder. | Decision-only history and unspecified repository license | Project workflow, README, licensing |
| 2026-11-12 | Record the requested three-step preset, customization, and deployment flow, one template selector, field examples, wider comparison, and component counts. Main layout changes require Charan’s approval. Use the selected template for copy and download and include pinned install commands in the file. | Permanent Helm-first export rule | Builder, export, deployment UX |
| 2026-11-12 | Add official repository and pinned install commands to export headers. Explain YAML indentation and include full dotted setting names in Helm values comments. | Post-install access notes without install commands or full setting paths | Exports |
| 2026-11-12 | Charan approved the three-step layout with preset, tool, and Kubernetes version together. Prometheus Minimal disables all four add-ons, Recommended keeps Pushgateway off per [Prometheus guidance](https://prometheus.io/docs/practices/pushing/), and Full starts from the [pinned chart defaults](https://github.com/prometheus-community/helm-charts/tree/prometheus-29.45.1/charts/prometheus) with the existing ARM64 workaround. Argo CD offers its full defaults. | One starting configuration per tool | Presets |
| 2026-11-12 | Give each Prometheus add-on its own labelled toggle, group generated resources after the main server, and count lines from the rendered manifest. Ship Argo CD CRDs with its selected YAML download and keep schemas collapsed in the preview. | Interleaved resources and separate CRD export actions | Builder, export |
| 2026-11-12 | Keep the approved three-step layout and default Prometheus to Minimal. Constrain the preset control, style actions and sticky YAML headers, show examples only for typed inputs, and link the component summary to collapsible YAML sections. | Full preset default and continuous YAML preview | Builder |
| 2026-11-12 | Offer Helm chart + values as an offline ZIP with the verified pinned archive, commented values, and install and after-install instructions. Use the explained YAML for preview, clipboard, and download, and verify the exact artifacts in Chromium and fresh K3s. | Values-only Helm download and separate preview and export content paths | Export, tests |
| 2026-11-12 | Default YAML to Learning with short authored block explanations. Offer Compact with component and purpose headers plus deployment and safety instructions. Keep detailed field help in hover, remove Helm release wording from YAML, shorten the Minimal label, and fix selection punctuation. | Per-field schema dumps and a single comment density | YAML export, builder |
| 2026-11-12 | Release the three-step dashboard, presets, YAML comment modes, explained exports, and offline Helm bundle as v0.2.0. | v0.1.0 release | Release notes |
| 2026-11-12 | Explain Prometheus scrape jobs, per-container probes and mounts, Recreate, ClusterRoleBinding subjects and roleRef, and ClusterIP access in Learning exports. Use singular wording for one replica and allow up to 700 lines for Minimal. | Resource-wide deduplication of probe and mount explanations and 650-line limit | Learning YAML, tests |
| 2026-11-13 | Keep the ARM64 reloader safety note and scrape jobs explanation in Compact. Release these changes and the post-v0.2.0 Learning fixes as v0.2.1. | Compact omitted both notes | YAML exports, release |
| 2026-11-13 | Default Argo CD to Minimal with Dex and notifications disabled and ApplicationSet scaled to zero. Recommended enables one ApplicationSet replica. Full uses the pinned chart defaults. Preserve the layout, start CRDs collapsed, and enforce a 3,600-line Minimal Learning budget excluding CRDs. | Argo CD offered Full only | Presets, YAML preview |
| 2026-11-13 | Keep Argo CD Minimal and Full chart defaults. Explain why ApplicationSet still appears at zero replicas. Keep the 3,600-line limit and require approval before changing Minimal chart values to reduce its size. | Separate Recommended preset with identical rendered size | Presets, Learning export |
| 2026-11-13 | Keep bundled SSH host keys and network policies in Argo CD Minimal. | Proposed line-count reductions | Argo CD installation presets |
| 2026-11-13 | Make usage templates primary for every current and future tool. Keep the existing workflow under Install the tool. Get approval for the initial template list before coding and for any layout change before implementation. | Installation as the main product flow | Product scope, usage templates, installation |
| 2026-11-13 | Charan approved the three Argo CD usage templates and the default Use the tool view. Explain every usage YAML line in Learning mode. Propose standalone Prometheus examples against the pinned chart and defer Operator as a separate tool. | Pending Argo CD and layout approval, proposed Operator templates | Usage templates, navigation, validation |

# Usage templates

Charan approved the three Argo CD templates and the default **Use the tool** view on November 13, 2026. Keep the existing installation feature under **Install the tool**. Charan approved the standalone Prometheus templates in the linked spec. Prometheus Operator is deferred as a separate tool.

Each template has form fields with descriptions and examples and produces short YAML with a plain-language explanation for every emitted YAML line in Learning mode. Validate against the official version-matched CRD schema. Each integration test applies the exported YAML on dedicated K3s with the relevant CRDs and controllers installed, then checks the expected controller behavior. Keep fixtures scoped to test namespaces.

| Template | Example form fields | K3s acceptance check |
|---|---|---|
| Argo CD Application | Name `my-app`, namespace `argocd`, project `default`, official example repo, revision `HEAD`, path `guestbook`, destination namespace `guestbook`, create namespace, automated sync, prune, self-heal, and cascading deletion | Sync the sample app and verify its workload becomes healthy. Explain that prune removes resources absent from Git and the finalizer can delete managed resources when the Application is deleted. |
| Argo CD AppProject | Name `guestbook-project`, allowed repository, destination server `https://kubernetes.default.svc`, destination namespace `guestbook` | Sync an Application through the project and verify the allowed destination works. |
| Argo CD ApplicationSet | Name `guestbook-environments`, list entries `dev` and `staging`, repository, revision, path, project, and namespace pattern | Run the ApplicationSet controller and verify it generates the two expected Applications. |

The Application starts in manual sync. Automated sync, pruning, self-healing, and cascading deletion are opt-in. ApplicationSet requires an active controller. The Argo CD Minimal install scales it to zero, so use Full for its K3s check.

[Application spec](https://argo-cd.readthedocs.io/en/stable/user-guide/application-specification/) · [AppProjects](https://argo-cd.readthedocs.io/en/stable/user-guide/projects/) · [ApplicationSet list generator](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-List/)

The official Guestbook example uses an AMD64-only image. On ARM64 hosts such as Apple Silicon Macs, it runs under AMD64 emulation. Show this note alongside the Guestbook path in Learning YAML.

## Standalone Prometheus

See the [standalone Prometheus proposal](standalone-prometheus-usage.md) for the chart-compatible Service annotations, custom scrape job, and alert rule.

## Verification

- [x] Chromium verifies the default usage view, all three CRD checks, and the Learning and Compact downloads.
- [x] Dedicated K3s 1.37.2 accepted all three generated resources. The ApplicationSet controller generated both expected Applications.
- [x] Guestbook reached Synced and Healthy through the generated AppProject and its Deployment rolled out. The fresh K3s retry disabled metrics-server to avoid discovery interruptions and imported the upstream AMD64-only Guestbook image on the ARM64 host using Docker Desktop emulation.

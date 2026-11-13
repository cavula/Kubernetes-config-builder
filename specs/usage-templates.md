# Usage templates

Proposal pending Charan’s approval. Usage is the primary workflow for every tool. Keep the existing installation feature under **Install the tool**. Obtain separate approval before changing the layout.

Each template has form fields with descriptions and examples and produces short, commented YAML. Validate against the official version-matched CRD schema. Each integration test applies the exported YAML on dedicated K3s with the relevant CRDs and controllers installed, then checks the expected controller behavior. Keep fixtures scoped to test namespaces.

| Template | Example form fields | K3s acceptance check |
|---|---|---|
| Argo CD Application | Name `my-app`, namespace `argocd`, project `default`, official example repo, revision `HEAD`, path `guestbook`, destination namespace `guestbook`, create namespace, automated sync, prune, self-heal, and cascading deletion | Sync the sample app and verify its workload becomes healthy. Explain that prune removes resources absent from Git and the finalizer can delete managed resources when the Application is deleted. |
| Argo CD AppProject | Name `guestbook-project`, allowed repository, destination server `https://kubernetes.default.svc`, destination namespace `guestbook` | Sync an Application through the project and verify the allowed destination works. |
| Argo CD ApplicationSet | Name `guestbook-environments`, list entries `dev` and `staging`, repository, revision, path, project, and namespace pattern | Run the ApplicationSet controller and verify it generates the two expected Applications. |
| Prometheus scrape job | ServiceMonitor name `guestbook-metrics`, namespace `monitoring`, service label `app=guestbook`, service namespace `guestbook`, named port `metrics`, path `/metrics`, interval `30s` | Verify a selected Service becomes a healthy scrape target. |
| Prometheus alert rule | PrometheusRule name `guestbook-alerts`, group `guestbook`, alert `GuestbookDown`, expression `up{job="guestbook"} == 0`, duration `5m`, severity `warning`, and summary | Verify the rule loads and fires against a controlled failing target. |

The Prometheus proposals use official Prometheus Operator CRDs to support Kubernetes schema validation and application with kubectl. They require a Prometheus Operator installation with matching selectors. The current standalone Prometheus installer consumes native scrape and rule configuration and does not reconcile these CRDs. ApplicationSet requires an active controller, which Argo CD Minimal scales to zero.

Sources: [Application spec](https://argo-cd.readthedocs.io/en/stable/user-guide/application-specification/), [AppProjects](https://argo-cd.readthedocs.io/en/stable/user-guide/projects/), [ApplicationSet list generator](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-List/), [Prometheus Operator API](https://prometheus-operator.dev/docs/api-reference/api/).

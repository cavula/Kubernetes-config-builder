# Live builder

Approved November 11, 2026.

## Inputs

Selected platform, Kubernetes minor, tool and chart pin, and the current values of approved options.

## Outputs

An options panel on the left and a read-only Monaco code preview on the right, with commented output, changed-line highlights, and field-level hover explanations.

## Settings

| Prometheus chart value | Expected YAML effect |
| --- | --- |
| `server.global.scrape_interval` | Changes the default scrape interval in the Prometheus server ConfigMap. |
| `server.retention` | Changes the server container’s time-based retention argument. |
| `server.persistentVolume.enabled` | Switches the server data volume between a PVC and emptyDir. |
| `server.persistentVolume.size` | Changes the PVC storage request when persistence is enabled. |
| `server.resources` | Sets the server container’s CPU and memory requests and limits. |
| `alertmanager.enabled` | Adds or removes Alertmanager resources and changes the server’s alerting configuration. |
| `kube-state-metrics.enabled` | Adds or removes the kube-state-metrics resources. |

[Official Prometheus chart values](https://github.com/prometheus-community/helm-charts/blob/prometheus-29.45.1/charts/prometheus/values.yaml)

| Argo CD chart value | Expected YAML effect |
| --- | --- |
| `server.service.type` | Sets the API/UI Service to ClusterIP or NodePort and changes node port fields accordingly. Defaults to ClusterIP. |
| `server.replicas` | Changes the API/UI Deployment replica count. |
| `repoServer.replicas` | Changes the repository server Deployment replica count. |
| `server.resources` | Sets the API/UI container’s CPU and memory requests and limits. |
| `repoServer.resources` | Sets the repository server container’s CPU and memory requests and limits. |
| `server.metrics.enabled` | Adds or removes the API server metrics Service. |
| `dex.enabled` | Adds or removes the Dex component’s resources for federated login. |

[Official Argo CD chart values](https://github.com/argoproj/argo-helm/blob/argo-cd-10.17.1/charts/argo-cd/values.yaml)

## Acceptance criteria

- [x] An accepted option edit updates the preview within one second. Rapid edits show the newest result.
- [x] The preview highlights added and changed lines and identifies removed lines or resources.
- [x] Each generated section includes a comment explaining its purpose. Each field has a plain-English hover explanation based on official sources.
- [x] Conditional controls explain their dependencies, such as PVC size requiring persistence.
- [x] For every approved option, fixtures verify chart paths and expected effects against renders from the pinned chart.
- [x] Defaults generate no privileged containers.

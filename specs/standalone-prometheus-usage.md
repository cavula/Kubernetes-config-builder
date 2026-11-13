# Standalone Prometheus usage templates

Approved by Charan. These templates target the existing standalone Prometheus chart 29.45.1, which packages Prometheus v3.16.0. Prometheus Operator is a separate future tool.

## Annotated metrics Service

Output a short `apiVersion: v1`, `kind: Service` manifest for a workload that already serves metrics. The form uses the Service name `guestbook-metrics`, namespace `guestbook`, selector `app: guestbook`, Service port `8080`, and target port `8080`. Show examples for the following annotations under `metadata.annotations`:

| Annotation | Example | Effect in the pinned chart |
| --- | --- | --- |
| `prometheus.io/scrape` | `"true"` | Lets the chart's enabled `kubernetes-service-endpoints` job keep this Service's EndpointSlice targets. |
| `prometheus.io/scheme` | `"http"` | Uses HTTP or HTTPS for the scrape. |
| `prometheus.io/path` | `"/metrics"` | Selects the metrics URL path. |
| `prometheus.io/port` | `"8080"` | Overrides the discovered endpoint port. Use the port the workload actually listens on. |

Use a Service selector matching the existing workload's Pod labels. The normal job drops Services marked `prometheus.io/scrape_slow: "true"`, so this template leaves that annotation out. Validate the Service using the selected Kubernetes version's official schema. In dedicated K3s, apply it against a test metrics workload, then check that the target is healthy in Prometheus.

## Custom scrape job

Output a short Helm values fragment to merge with the installed release's values. Use `scrapeConfigs.<job-name>` with a unique name such as `guestbook-static`. The chart uses that key as the default native `job_name` and adds it alongside its default jobs. The form fields map to:

| Field below the job name | Example | Effect |
| --- | --- | --- |
| `enabled` | `true` | Tells the chart to include this job. |
| `scrape_interval` | `30s` | Sets how often Prometheus requests metrics. |
| `metrics_path` | `/metrics` | Selects the HTTP path. |
| `scheme` | `http` | Selects the protocol. |
| `static_configs[].targets[]` | `guestbook-metrics.guestbook.svc:8080` | Provides a host and Service port reachable from Prometheus. |

Render the fragment with the pinned chart and validate the generated Prometheus configuration with the matching `promtool check config`. In dedicated K3s, update the Helm release with the fragment and verify the named job has a healthy target. Use an unannotated Service for this fixture so Prometheus collects the endpoint only once.

## Alert rule

Output a short Helm values fragment under `serverFiles.alerting_rules.yml`. The chart writes it to the server ConfigMap and its default `rule_files` loads `/etc/config/alerting_rules.yml`. The form fields map to:

| Field below `serverFiles.alerting_rules.yml` | Example | Effect |
| --- | --- | --- |
| `groups[].name` | `guestbook` | Names the rule evaluation group. |
| `groups[].rules[].alert` | `GuestbookDown` | Names the alert. |
| `groups[].rules[].expr` | `up{job="guestbook-static"} == 0` | Matches a configured target whose scrape fails. |
| `groups[].rules[].for` | `5m` | Requires the condition to persist for five minutes before firing. |
| `groups[].rules[].labels.severity` | `warning` | Adds a severity label for routing or display. |
| `groups[].rules[].annotations.summary` | `Guestbook metrics endpoint is down` | Provides a readable description of the alert. |

Preserve existing rule groups when merging the fragment because Helm replaces lists supplied at the same values path. Validate the rendered rule file with the matching `promtool check rules` and test evaluation with `promtool test rules`. In dedicated K3s, load the fragment through Helm, keep the configured target present but unable to serve metrics, and verify the alert reaches the firing state. Prometheus evaluates the rule directly. Delivery to notification receivers requires a configured Alertmanager.

## Acceptance criteria

- [x] Each template has form fields with examples and short YAML. Learning mode explains every YAML line in plain words.
- [x] Validate the Service against Kubernetes' official schema and validate the two Helm fragments against the pinned chart and rendered Prometheus configuration or rule file.
- [x] Each K3s test uses the generated output through its intended entry point and checks the target or alert behavior.

## Verification and sources

The pinned archive's `values.yaml` and `templates/cm.yaml` were inspected. A local `helm template` run against the verified archive rendered the custom job alongside default jobs, generated the alert file, and retained the rule file's mount path in the default Prometheus configuration.

- [Pinned Prometheus Helm chart values](https://github.com/prometheus-community/helm-charts/blob/prometheus-29.45.1/charts/prometheus/values.yaml)
- [Pinned chart ConfigMap template](https://github.com/prometheus-community/helm-charts/blob/prometheus-29.45.1/charts/prometheus/templates/cm.yaml)
- [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/)
- [Prometheus scrape configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config)
- [Prometheus alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Prometheus rule tests](https://prometheus.io/docs/prometheus/latest/configuration/unit_testing_rules/)

Both Helm values downloads begin with a comment identifying the file as an addition to an existing release and the exact pinned `helm upgrade` command. Keep these instructions in Compact as well. For rules, merge existing groups into the downloaded file before upgrading.

[Helm upgrade reference](https://helm.sh/docs/helm/helm_upgrade/)

The dedicated K3s 1.37.2 test applied the generated Service and upgraded the pinned release with the generated Helm fragments. Both scrape targets were healthy, and the alert fired after the test made the custom target's scrape fail. The test queries the Prometheus API from the dedicated k3d node.

#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 1 ]]; then
  echo "Usage: $0 <dedicated-prometheus-usage-kubeconfig>" >&2
  exit 2
fi

root=$(cd "$(dirname "$0")/.." && pwd)
config=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
cluster=(kubectl --kubeconfig "$config")
context=$("${cluster[@]}" config current-context)
endpoint=$("${cluster[@]}" config view --minify -o jsonpath='{.clusters[0].cluster.server}')
if [[ "$context" != k3d-kcb-test-137-prom-usage || "$endpoint" != https://127.0.0.1:* ]]; then
  echo "Use a fresh dedicated local k3d-kcb-test-137-prom-usage cluster." >&2
  exit 2
fi
"${cluster[@]}" get --raw /version | python3 -c '
import json, sys
version = json.load(sys.stdin)["gitVersion"]
if not version.startswith("v1.37.") or "+k3s" not in version:
    sys.exit("Expected K3s 1.37, got " + version)
'

cd "$root"
chart_name=$(awk '$2 ~ /^prometheus-/ { print $2 }' backend/chart-checksums.txt)
chart="$root/.runtime/charts/$chart_name"
python3 - "$chart" <<'PY'
from hashlib import sha256
from pathlib import Path
import sys

archive = Path(sys.argv[1])
expected = next(line.split()[0] for line in Path("backend/chart-checksums.txt").read_text().splitlines()
                if line.split()[1] == archive.name)
if sha256(archive.read_bytes()).hexdigest() != expected:
    sys.exit("Prometheus chart checksum mismatch")
PY
version=$(helm show chart "$chart" | awk '$1 == "appVersion:" { print $2 }')

scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

node --input-type=module - "$scratch" <<'JS'
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { defaultUsageValues, renderUsage, usageFilename } from './frontend/src/usage-values.ts';

const staticJob = {
  jobName: 'kcb-static',
  interval: '10s',
  metricsPath: '/metrics',
  target: 'kcb-static.kcb-prometheus-usage.svc:9090',
};
const fixtures = [
  ['service-scrape', usageFilename('service-scrape'), {
    name: 'kcb-annotated',
    namespace: 'kcb-prometheus-usage',
    selectorKey: 'app',
    selectorValue: 'kcb-annotated',
    servicePort: 9090,
    targetPort: 9090,
    metricsPath: '/metrics',
  }],
  ['scrape-job', usageFilename('scrape-job'), staticJob],
  ['scrape-job', 'unhealthy-job.yaml', { ...staticJob, metricsPath: '/kcb-metrics-missing' }],
  ['alert-rule', usageFilename('alert-rule'), {
    groupName: 'kcb-usage',
    alertName: 'KcbTargetDown',
    expression: 'up{job="kcb-static"} == 0',
    duration: '15s',
  }],
];
for (const [id, filename, overrides] of fixtures) {
  const { text } = renderUsage(id, { ...defaultUsageValues(id), ...overrides }, true);
  writeFileSync(join(process.argv[2], filename), text);
}
JS

fixture=(kubectl --kubeconfig "$config" -n kcb-prometheus-usage)
"${cluster[@]}" create namespace kcb-prometheus-usage
for workload in kcb-annotated kcb-static; do
  "${fixture[@]}" create deployment "$workload" --image="quay.io/prometheus/prometheus:$version" --port=9090
  "${fixture[@]}" rollout status "deployment/$workload" --timeout=300s
done
"${fixture[@]}" expose deployment kcb-static --port=9090 --target-port=9090
"${fixture[@]}" apply --server-side --dry-run=server -f "$scratch/service-scrape.yaml"
"${fixture[@]}" apply --server-side -f "$scratch/service-scrape.yaml"

helm install prometheus "$chart" --kubeconfig "$config" --namespace monitoring --create-namespace \
  -f backend/prometheus-arm64-values.json \
  --set 'alertmanager.enabled=false,kube-state-metrics.enabled=false,prometheus-node-exporter.enabled=false,prometheus-pushgateway.enabled=false' \
  --set 'server.persistentVolume.enabled=false,server.global.scrape_interval=10s,server.global.evaluation_interval=5s' \
  --wait --timeout 8m

check_prometheus() {
  "${cluster[@]}" -n monitoring rollout status deployment/prometheus-server --timeout=300s
  local service_ip
  service_ip=$("${cluster[@]}" -n monitoring get service prometheus-server -o jsonpath='{.spec.clusterIP}')
  python3 - "$service_ip" "$1" <<'PYCODE'
import json
import subprocess
import sys
import time

service_ip, phase = sys.argv[1:]
deadline = time.monotonic() + 300
last_error = "Prometheus API did not respond"
while time.monotonic() < deadline:
    try:
        endpoint = "alerts" if phase == "alert" else "targets"
        response = subprocess.check_output(
            ["docker", "exec", "k3d-kcb-test-137-prom-usage-server-0", "wget", "-qO", "-",
             f"http://{service_ip}/api/v1/{endpoint}"],
            timeout=20,
        )
        result = json.loads(response)["data"]
    except (KeyError, OSError, ValueError, subprocess.SubprocessError) as error:
        last_error = str(error)
        time.sleep(2)
        continue
    if phase == "annotated":
        candidates = [item for item in result["activeTargets"]
                      if item["labels"].get("job") == "kubernetes-service-endpoints"
                      and item["discoveredLabels"].get("__meta_kubernetes_namespace") == "kcb-prometheus-usage"
                      and item["discoveredLabels"].get("__meta_kubernetes_service_name") == "kcb-annotated"]
        if any(item["health"] == "up" for item in candidates):
            print("Annotated Service target is up")
            break
        last_error = str([(item["health"], item["lastError"]) for item in candidates])
    elif phase == "static":
        candidates = [item for item in result["activeTargets"] if item["labels"].get("job") == "kcb-static"]
        if any(item["health"] == "up" for item in candidates):
            print("Custom scrape job target is up")
            break
        last_error = str([(item["health"], item["lastError"]) for item in candidates])
    else:
        candidates = [item for item in result["alerts"] if item["labels"].get("alertname") == "KcbTargetDown"]
        if any(item["state"] == "firing" for item in candidates):
            print("Alert rule is firing for the failed scrape")
            break
        last_error = str([item["state"] for item in candidates])
    time.sleep(2)
else:
    sys.exit(f"Prometheus {phase} check timed out. Last result: {last_error}")
PYCODE
}

check_prometheus annotated
helm upgrade prometheus "$chart" --kubeconfig "$config" --namespace monitoring --reuse-values \
  -f "$scratch/prometheus-scrape-job-values.yaml" --wait --timeout 8m
check_prometheus static
helm upgrade prometheus "$chart" --kubeconfig "$config" --namespace monitoring --reuse-values \
  -f "$scratch/unhealthy-job.yaml" -f "$scratch/prometheus-alert-rule-values.yaml" --wait --timeout 8m
check_prometheus alert

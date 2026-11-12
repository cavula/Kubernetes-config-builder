#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "Usage: $0 <1.35|1.36|1.37> <prometheus|argo-cd> <dedicated-test-kubeconfig> [values-file]" >&2
  exit 2
fi
version=$1
tool=$2
config=$3
values_file=${4:-}
app_url=${KCB_APP_URL:-http://127.0.0.1:8080}
rollout_timeout=${KCB_ROLLOUT_TIMEOUT:-600s}

case "$version" in
  1.35) expected_context=k3d-kcb-test-135 ;;
  1.36) expected_context=k3d-kcb-test-136 ;;
  1.37) expected_context=k3d-kcb-test ;;
  *) echo "Unsupported Kubernetes version: $version" >&2; exit 2 ;;
esac
case "$tool" in
  prometheus) namespace=monitoring ;;
  argo-cd) namespace=argocd ;;
  *) echo "Unsupported tool: $tool" >&2; exit 2 ;;
esac

context=$(kubectl --kubeconfig "$config" config current-context)
if [[ "$context" != "$expected_context" && "$context" != "k3d-kcb-test-${version//./}-fresh" && "$context" != "k3d-kcb-test-${version//./}-variant" ]]; then
  echo "Unexpected dedicated test context: $context" >&2
  exit 2
fi
cluster=(kubectl --kubeconfig "$config" --context "$context")
endpoint=$("${cluster[@]}" config view --minify -o jsonpath='{.clusters[0].cluster.server}')
if [[ "$endpoint" != https://127.0.0.1:* ]]; then
  echo "Expected a loopback API endpoint for $expected_context." >&2
  exit 2
fi
server_version=$("${cluster[@]}" get --raw /version | python3 -c 'import json, sys; print(json.load(sys.stdin)["gitVersion"])')
if [[ "$server_version" != v"$version".*+k3s* ]]; then
  echo "Expected K3s $version, got $server_version." >&2
  exit 2
fi

scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
manifest="$scratch/$tool.yaml"
payload=$(python3 - "$tool" "$version" "$values_file" <<'PYCODE'
import json
import sys
from pathlib import Path

tool, version, values_file = sys.argv[1:]
values = Path(values_file).read_text() if values_file else ""
print(json.dumps({"tool": tool, "kubernetesVersion": version, "values": values}))
PYCODE
)
curl --fail --silent --show-error -H 'Content-Type: application/json' --data "$payload" \
  "$app_url/api/validate" |
  python3 -c 'import json,sys; result=json.load(sys.stdin); print(json.dumps(result)); sys.exit(result["status"] != "passed")'
curl --fail --silent --show-error -H 'Content-Type: application/json' --data "$payload" \
  "$app_url/api/export" |
  python3 -c 'import json,sys; sys.stdout.write(json.load(sys.stdin)["yaml"])' > "$manifest"

# Each export includes its namespace. Apply the complete export on a fresh
# cluster before a server dry run, which cannot persist a Namespace.
"${cluster[@]}" -n "$namespace" apply --server-side -f "$manifest"
"${cluster[@]}" -n "$namespace" apply --server-side --dry-run=server -f "$manifest"

workloads=$("${cluster[@]}" -n "$namespace" get deployments,statefulsets,daemonsets -o name)
if [[ -z "$workloads" ]]; then
  echo "No workloads to check in $namespace." >&2
  exit 1
fi
while IFS= read -r workload; do
  if ! "${cluster[@]}" -n "$namespace" rollout status "$workload" --timeout="$rollout_timeout"; then
    "${cluster[@]}" -n "$namespace" get pods -o wide --request-timeout=15s || true
    "${cluster[@]}" -n "$namespace" get events --sort-by=.lastTimestamp --request-timeout=15s || true
    exit 1
  fi
done <<< "$workloads"

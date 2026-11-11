#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "Usage: $0 <1.35|1.36|1.37> <prometheus|argo-cd> <dedicated-test-kubeconfig>" >&2
  exit 2
fi
version=$1
tool=$2
config=$3

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

cluster=(kubectl --kubeconfig "$config" --context "$expected_context")
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
curl --fail --silent --show-error -H 'Content-Type: application/json' \
  --data "{\"tool\":\"$tool\",\"kubernetesVersion\":\"$version\",\"values\":\"\"}" \
  http://127.0.0.1:8080/api/render > "$manifest"

if [[ "$tool" == prometheus ]]; then
  "${cluster[@]}" create namespace "$namespace" --dry-run=client -o yaml |
    "${cluster[@]}" apply --server-side -f -
  "${cluster[@]}" -n "$namespace" apply --server-side --dry-run=server -f "$manifest"
  "${cluster[@]}" -n "$namespace" apply --server-side -f "$manifest"
else
  # The exported Namespace must be persisted before a server dry run can see it.
  # The live install uses the complete export on its own.
  "${cluster[@]}" apply --server-side -f "$manifest"
  "${cluster[@]}" apply --server-side --dry-run=server -f "$manifest"
fi

workloads=$("${cluster[@]}" -n "$namespace" get deployments,statefulsets,daemonsets -o name)
if [[ -z "$workloads" ]]; then
  echo "No workloads to check in $namespace." >&2
  exit 1
fi
while IFS= read -r workload; do
  "${cluster[@]}" -n "$namespace" rollout status "$workload" --timeout=120s
done <<< "$workloads"

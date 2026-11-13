#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 1 ]]; then
  echo "Usage: $0 <dedicated-test-kubeconfig>" >&2
  exit 2
fi

root=$(cd "$(dirname "$0")/.." && pwd)
cluster=(kubectl --kubeconfig "$1")
context=$("${cluster[@]}" config current-context)
endpoint=$("${cluster[@]}" config view --minify -o jsonpath='{.clusters[0].cluster.server}')
if [[ "$context" != k3d-kcb-test-137-fresh || "$endpoint" != https://127.0.0.1:* ]]; then
  echo "Use a fresh dedicated local k3d-kcb-test-137-fresh cluster." >&2
  exit 2
fi
"${cluster[@]}" get --raw /version | python3 -c 'import json,sys; assert json.load(sys.stdin)["gitVersion"].startswith("v1.37.")'
"${cluster[@]}" -n argocd rollout status deployment/argo-cd-argocd-applicationset-controller --timeout=300s

scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
cd "$root"
node --input-type=module - "$scratch" <<'JS'
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { defaultUsageValues, renderUsage } from './frontend/src/usage-values.ts';

const fixtures = {
  'app-project': { name: 'usage-guestbook', destinationNamespace: 'usage-guestbook' },
  application: {
    name: 'usage-guestbook', project: 'usage-guestbook', destinationNamespace: 'usage-guestbook',
    automated: true, createNamespace: true,
  },
  'application-set': { name: 'usage-envs', destinationNamespace: 'usage-{{ .environment }}' },
};
for (const [id, fields] of Object.entries(fixtures)) {
  const { text } = renderUsage(id, { ...defaultUsageValues(id), ...fields }, true);
  writeFileSync(join(process.argv[2], id + '.yaml'), text);
}
JS

"${cluster[@]}" create namespace usage-guestbook --dry-run=client -o yaml | "${cluster[@]}" apply --server-side -f -
for template in app-project application application-set; do
  "${cluster[@]}" apply --server-side --dry-run=server -f "$scratch/$template.yaml"
  "${cluster[@]}" apply --server-side -f "$scratch/$template.yaml"
done
for env in dev staging; do
  for attempt in {1..60}; do
    if "${cluster[@]}" -n argocd get "application/usage-envs-$env" -o name 2>/dev/null; then break; fi
    sleep 2
  done
  destination=$("${cluster[@]}" -n argocd get "application/usage-envs-$env" -o jsonpath='{.spec.destination.namespace}')
  [[ "$destination" == "usage-$env" ]]
done
"${cluster[@]}" -n argocd wait application/usage-guestbook --for=jsonpath='{.status.sync.status}'=Synced --timeout=300s
"${cluster[@]}" -n argocd wait application/usage-guestbook --for=jsonpath='{.status.health.status}'=Healthy --timeout=300s
"${cluster[@]}" -n usage-guestbook rollout status deployment/guestbook-ui --timeout=300s

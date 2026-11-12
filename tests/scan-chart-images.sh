#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <image-inventory-file> <report-directory>" >&2
  exit 2
fi
inventory=$1
reports=$2
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$reports"
if [[ ! -s "$inventory" ]]; then
  echo "Image inventory is missing or empty: $inventory" >&2
  exit 1
fi

result=0
image_number=0
while IFS= read -r image; do
  [[ -n "$image" ]] || continue
  image_number=$((image_number + 1))
  for architecture in amd64 arm64; do
    report="$reports/image-$image_number-$architecture.json"
    rm -f -- "$report"
    echo "Scanning $image for linux/$architecture"
    if ! trivy image --image-src remote --platform "linux/$architecture" --scanners vuln \
        --severity HIGH,CRITICAL --no-progress --exit-code 0 --format json --output "$report" "$image"; then
      echo "Image scan did not complete: $image (linux/$architecture)" >&2
      result=1
      continue
    fi
    if ! python3 "$root/tests/report-image-findings.py" "$report" "$root/security/accepted-risks.json" "$image" "$architecture"; then
      result=1
    fi
  done
done < "$inventory"

if [[ "$image_number" -eq 0 ]]; then
  echo "No images scanned." >&2
  exit 1
fi
exit "$result"

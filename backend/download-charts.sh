#!/bin/sh
set -eu

backend=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
target="$backend/../.runtime/charts"
mkdir -p "$target"
stage=$(mktemp -d "$target/.download.XXXXXX")
trap 'rm -rf "$stage"' EXIT
trap 'exit 1' HUP INT TERM

while read -r digest archive; do
    case "$archive" in
        prometheus-*) chart=prometheus; registry=oci://ghcr.io/prometheus-community/charts/prometheus ;;
        argo-cd-*) chart=argo-cd; registry=oci://ghcr.io/argoproj/argo-helm/argo-cd ;;
        *) echo "Unknown pinned archive: $archive" >&2; exit 1 ;;
    esac
    version=${archive#"$chart"-}
    version=${version%.tgz}
    "${HELM:-helm}" pull "$registry" --version "$version" --destination "$stage"
done < "$backend/chart-checksums.txt"

(
    cd "$stage"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "$backend/chart-checksums.txt"
    else
        shasum -a 256 -c "$backend/chart-checksums.txt"
    fi
)

while read -r digest archive; do
    mv "$stage/$archive" "$target/$archive"
done < "$backend/chart-checksums.txt"

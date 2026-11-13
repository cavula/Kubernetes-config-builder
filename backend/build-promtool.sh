#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <absolute-output-directory>" >&2
    exit 2
fi
case "$1" in
    /*) output=$1 ;;
    *) echo "Use an absolute output directory." >&2; exit 2 ;;
esac

backend=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
version=3.16.0
archive="prometheus-$version.tar.gz"
# Official tagged source from https://github.com/prometheus/prometheus/releases/tag/v3.16.0
checksum=2709a3bd43f9d15e0cecfc33ab473d011e2a69ab29265b91398cc1b8561403c0
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
trap 'exit 1' HUP INT TERM
cd "$scratch"
curl --fail --silent --show-error --location "https://codeload.github.com/prometheus/prometheus/tar.gz/refs/tags/v$version" -o "$archive"
actual=$(sha256sum "$archive")
if [ "${actual%% *}" != "$checksum" ]; then
    echo "Prometheus source archive checksum mismatch." >&2
    exit 1
fi
tar -xzf "$archive"
cd "prometheus-$version"

# Use the project's patched x/net pin to keep the bundled checker clear of CVE-2026-78661.
patched_net=$(awk '$1 == "golang.org/x/net" {print $2; exit}' "$backend/go.mod")
go mod edit "-require=golang.org/x/net@$patched_net"
awk -v version="$patched_net" '$1 == "golang.org/x/net" && ($2 == version || $2 == version "/go.mod")' "$backend/go.sum" >> go.sum
mkdir -p "$output"
CGO_ENABLED=0 GOTOOLCHAIN=local go build -buildvcs=false -mod=readonly -trimpath \
    -ldflags "-X github.com/prometheus/common/version.Version=$version" \
    -o "$output/promtool" ./cmd/promtool

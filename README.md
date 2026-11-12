# Kubernetes Config Builder and Learning Dashboard

[![CI](https://github.com/cavula/Kubernetes-config-builder/actions/workflows/ci.yml/badge.svg)](https://github.com/cavula/Kubernetes-config-builder/actions/workflows/ci.yml)

A local dashboard for building and learning Kubernetes configurations. Choose K3s 1.35, 1.36, or 1.37 and configure one tool at a time. The app uses Prometheus chart 29.45.1 and Argo CD chart 10.17.1.

Edit settings beside a live Helm values or Kubernetes manifest preview. Compare a Helm option with the YAML it changes, check diagnostics from Helm lint and kubeconform, and copy or download the active format. A live cluster is only needed for deployment checks.

## Run locally

Install Node.js 26.13.1, Go 1.27.4, Helm, kubeconform, and Python 3. If you use nvm, run `nvm install` and `nvm use` from the project root. Download the verified chart archives and pinned schemas once.

```sh
./backend/download-charts.sh
python3 backend/download-schemas.py
```

Start the renderer from the project root.

```sh
cd backend
go run .
```

In another terminal, start the frontend from the project root.

```sh
cd frontend
npm ci
npm run dev
```

Open [http://localhost:5173](http://localhost:5173) or the URL printed by Vite. The renderer listens on loopback port 8080.

## Apply an export

For Argo CD, use `kubectl apply --server-side -f argo-cd-manifests.yaml`. Server-side apply avoids the annotation size limit for the chart’s large CRDs. The export includes the namespace and Redis initialization Job with its permissions.

For Prometheus, create the `monitoring` namespace if needed, then run `kubectl -n monitoring apply --server-side -f prometheus-manifests.yaml`. Both configuration reloader images use the ARM64-compatible tag recorded in the [Decision Log](AGENTS.md#decision-log).

## Test

- Run `npm test` and `npm run build` from `frontend`.
- Run `go test ./...` from `backend` after downloading the charts.
- Follow the [local K3s deployment instructions](docs/local-rendering.md#local-k3s-deploy-checks) for server-side dry runs and rollout checks.

GitHub Actions verifies chart checksums, runs the frontend and Go checks, and runs Helm lint and strict kubeconform across the supported Kubernetes versions on every push. You can also start CI manually from the Actions tab.

The [MVP specs](specs/) and [Decision Log](AGENTS.md#decision-log) record acceptance criteria, implementation decisions, and remaining work.

### Security checks

CI fails on high or critical npm and Go findings. Image findings recorded in the [accepted risks file](security/accepted-risks.json) are reported through their review date. Unaccepted or expired findings fail CI. npm audit reads the frontend lockfile. Trivy scans the Go module and the images from both verified charts across supported Kubernetes minors and builder toggles. The image scan covers Linux AMD64 and ARM64 and includes the Argo CD initialization Job.

To reproduce the image scan locally with Trivy installed, run from the project root after downloading the charts.

```sh
mkdir -p .runtime/security
(cd backend && go run . -list-images) > .runtime/security/chart-images.txt
./tests/scan-chart-images.sh .runtime/security/chart-images.txt .runtime/security/images
```

The script prints findings and writes per-image JSON reports to the directory you pass. An incomplete scan fails the check.

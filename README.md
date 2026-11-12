# Kubernetes Config Builder and Learning Dashboard

[![CI](https://github.com/cavula/Kubernetes-config-builder/actions/workflows/ci.yml/badge.svg)](https://github.com/cavula/Kubernetes-config-builder/actions/workflows/ci.yml)

A local dashboard for building and learning Kubernetes configurations. Choose K3s 1.35, 1.36, or 1.37 and configure one tool at a time. The app uses Prometheus chart 29.45.1 and Argo CD chart 10.17.1.

Choose a tool, Kubernetes version, and preset. Customize settings and Prometheus add-ons if you want, then choose a template in Deploy. You can compare a Helm option with the YAML it changes and check diagnostics from Helm lint and kubeconform. Building, validating, and exporting work without a live cluster.

## Run with Docker

1. Install and start [Docker Desktop](https://www.docker.com/products/docker-desktop/) or Docker Engine with Compose.
2. Download **Source code (zip)** from the [v0.1.0 release](https://github.com/cavula/Kubernetes-config-builder/releases/tag/v0.1.0), extract it, and open a terminal in the extracted folder.
3. Run:

   ```sh
   docker compose up --build
   ```

4. Open [http://localhost:8080](http://localhost:8080). Press **Ctrl+C** in the terminal to stop the dashboard.

The first build needs internet access. Docker verifies the pinned chart archives and Kubernetes schemas during the build and includes them in the resulting image. The dashboard runs locally without a Kubernetes cluster or developer tools.

## Run from source for development

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

In **Deploy**, choose **Helm chart** for a commented values file or **YAML manifest** for explained Kubernetes resources. Click **Download** and run the commands shown in Deploy or at the top of the file. The Argo CD manifest includes its CRDs, namespace, and Redis initialization Job. Its preview collapses the CRD schemas.

For Prometheus, run `kubectl apply --server-side -f prometheus-manifests.yaml`. The file creates the `monitoring` namespace. Both configuration reloader images use the ARM64-compatible tag recorded in the [Decision Log](AGENTS.md#decision-log).

## Test

- Run `npm test` and `npm run build` from `frontend`.
- Run `go test ./...` from `backend` after downloading the charts.
- Follow the [local K3s deployment instructions](docs/local-rendering.md#local-k3s-deploy-checks) for server-side dry runs and rollout checks.

GitHub Actions verifies chart checksums, runs the frontend and Go checks, and runs Helm lint and strict kubeconform across the supported Kubernetes versions on every push. You can also start CI manually from the Actions tab.

The [MVP specs](specs/) and [Decision Log](AGENTS.md#decision-log) record acceptance criteria, implementation decisions, and remaining work.

Run the browser checks with the backend and Vite dev server running. From `frontend`, install Chromium with `npx playwright-core install chromium`, then run `npm run test:browser`. Set `KCB_CHROME_PATH` to use an installed Chrome binary.

### Security checks

CI fails on high or critical npm and Go findings. Image findings recorded in the [accepted risks file](security/accepted-risks.json) are reported through their review date. Unaccepted or expired findings fail CI. npm audit reads the frontend lockfile. Trivy scans the Go module and the images from both verified charts across supported Kubernetes minors and builder toggles. The image scan covers Linux AMD64 and ARM64 and includes the Argo CD initialization Job.

To reproduce the image scan locally with Trivy installed, run from the project root after downloading the charts.

```sh
mkdir -p .runtime/security
(cd backend && go run . -list-images) > .runtime/security/chart-images.txt
./tests/scan-chart-images.sh .runtime/security/chart-images.txt .runtime/security/images
```

The script prints findings and writes per-image JSON reports to the directory you pass. An incomplete scan fails the check.

## License

Licensed under the [MIT License](LICENSE). Copyright © 2026 Charan.

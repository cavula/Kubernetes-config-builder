# Local chart rendering

Install Go 1.27.4, Helm, and Node.js 26.13.1. From the project root, download the pinned charts once.

```sh
./backend/download-charts.sh
```

The script verifies both archives against the SHA256 pins in `backend/chart-checksums.txt` before installing them in `.runtime/charts`. The renderer checks the same hashes before loading an archive.

Start the renderer in one terminal.

```sh
cd backend
go run .
```

From the project root in a second terminal, start the frontend.

```sh
cd frontend
npm ci
npm run dev
```

Open the URL Vite prints and choose **Kubernetes manifests** under **Preview format**. Rendering uses the archives on disk and works without a Kubernetes connection. Prometheus uses the release name `prometheus` and namespace `monitoring`. Argo CD uses `argo-cd` and `argocd`.

Run `go test ./...` from `backend` with both chart archives in place to check the supported version matrix and the effect of each approved option.

Fixture tests fail if either archive is missing or corrupt. For local work without chart archives, run `KCB_SKIP_CHART_FIXTURES=1 go test ./...` to skip the chart-dependent tests explicitly.

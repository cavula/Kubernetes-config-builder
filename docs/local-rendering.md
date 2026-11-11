# Local chart rendering

Install Go 1.27.4, Helm, and Node.js 26.13.1. From the project root, download the pinned charts once.

```sh
mkdir -p .runtime/charts
helm pull oci://ghcr.io/prometheus-community/charts/prometheus --version 29.45.1 --destination .runtime/charts
helm pull oci://ghcr.io/argoproj/argo-helm/argo-cd --version 10.17.1 --destination .runtime/charts
```

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

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

## Offline validation

Install Helm and kubeconform on the backend's PATH. From the project root, prepare the built-in schemas once with `python3 backend/download-schemas.py`. The script downloads native schemas from the kubeconform catalog and uses the official Kubernetes OpenAPI definitions for CustomResourceDefinition. To use your own schema directory, start the backend with `go run . -schemas /path/to/schemas`. The default is `../.runtime/schemas` from `backend`.

Place native schemas in the kubeconform catalog layout, such as `v1.37.0-standalone-strict/deployment-apps-v1.json`. Use version directories for 1.35.0, 1.36.0, and 1.37.0. Place custom resource schemas in `custom/<group>/<lowercase-kind>_<api-version>.json`. The CustomResourceDefinition itself requires its Kubernetes API schema, which is distinct from the schema for instances of a custom resource. [Kubeconform schema sources and layout](https://github.com/yannh/kubeconform#overriding-schemas-location)

The builder runs validation automatically. Missing local tools or schemas appear as incomplete checks. Rendering and validation use local files and do not connect to a cluster.

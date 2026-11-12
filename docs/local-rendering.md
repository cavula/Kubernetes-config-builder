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

Argo CD plain YAML includes its namespace and the Redis initialization Job, service account, role, and role binding from the pinned chart. Apply `argo-cd-crds.yaml` first and then `argo-cd-manifests.yaml` with `kubectl apply --server-side -f`. Server-side apply avoids the client-side annotation size limit for Argo CD’s large CRDs. Kubernetes retries the dependent containers until the Job creates the Redis Secret.

## Local K3s deploy checks

With Docker and k3d installed, create dedicated loopback clusters and save their kubeconfigs separately. Run from the project root.

```sh
mkdir -p .runtime/k3s
k3d cluster create kcb-test-135 --image rancher/k3s:v1.35.10-k3s1 --api-port 127.0.0.1:6551 --kubeconfig-update-default=false --kubeconfig-switch-context=false --wait
k3d cluster create kcb-test-136 --image rancher/k3s:v1.36.6-k3s1 --api-port 127.0.0.1:6552 --kubeconfig-update-default=false --kubeconfig-switch-context=false --wait
(umask 077; k3d kubeconfig get kcb-test-135 > .runtime/k3s/kubeconfig-135.yaml)
(umask 077; k3d kubeconfig get kcb-test-136 > .runtime/k3s/kubeconfig-136.yaml)
```

Start the Go renderer and run the check once per tool and Kubernetes version. The check applies the same explained YAML sent to the browser for copy and download. It waits up to ten minutes per Deployment, StatefulSet, and DaemonSet and prints pod status and events on failure. Set `KCB_ROLLOUT_TIMEOUT` to override the timeout or `KCB_APP_URL` to point to another local renderer port.

```sh
./tests/check-k3s-deploy.sh 1.35 argo-cd .runtime/k3s/kubeconfig-135.yaml
./tests/check-k3s-deploy.sh 1.35 prometheus .runtime/k3s/kubeconfig-135.yaml
./tests/check-k3s-deploy.sh 1.36 argo-cd .runtime/k3s/kubeconfig-136.yaml
./tests/check-k3s-deploy.sh 1.36 prometheus .runtime/k3s/kubeconfig-136.yaml
```

Pass a values file as the fourth argument to test a resource-changing configuration. Run it against a fresh dedicated cluster, so resources from a previous installation do not affect the result.

```sh
k3d cluster create kcb-test-135-variant --image rancher/k3s:v1.35.10-k3s1 --api-port 127.0.0.1:6554 --kubeconfig-update-default=false --kubeconfig-switch-context=false --wait
(umask 077; k3d kubeconfig get kcb-test-135-variant > .runtime/k3s/kubeconfig-135-variant.yaml)
./tests/check-k3s-deploy.sh 1.35 prometheus .runtime/k3s/kubeconfig-135-variant.yaml tests/fixtures/prometheus-resource-options.yaml
./tests/check-k3s-deploy.sh 1.35 argo-cd .runtime/k3s/kubeconfig-135-variant.yaml tests/fixtures/argo-cd-resource-options.yaml
```

The check runs Helm lint and kubeconform through the validation API before submitting resources. Keep Helm and kubeconform on the backend’s PATH and prepare the local schemas first. The Go equivalence test also runs `helm template` from PATH to compare the exported resource objects with Helm’s output.

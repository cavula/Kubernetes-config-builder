#!/usr/bin/env python3
"""Prepare the local kubeconform schemas used by the two MVP charts."""

import json
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from urllib.request import urlopen

SCHEMA_DIR = Path(__file__).resolve().parents[1] / ".runtime" / "schemas"
VERSIONS = ("1.35.0", "1.36.0", "1.37.0")
CATALOG = "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master"
KUBERNETES = "https://raw.githubusercontent.com/kubernetes/kubernetes"
NATIVE_SCHEMAS = (
    "clusterrole-rbac-v1",
    "clusterrolebinding-rbac-v1",
    "configmap-v1",
    "daemonset-apps-v1",
    "deployment-apps-v1",
    "job-batch-v1",
    "networkpolicy-networking-v1",
    "persistentvolumeclaim-v1",
    "role-rbac-v1",
    "rolebinding-rbac-v1",
    "secret-v1",
    "service-v1",
    "serviceaccount-v1",
    "statefulset-apps-v1",
)


def fetch(url):
    with urlopen(url, timeout=30) as response:
        return json.load(response)


def save_schema(path, schema):
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(schema) + "\n", encoding="utf-8")
    temporary.replace(path)


def download_native(version, name):
    folder = f"v{version}-standalone-strict"
    schema = fetch(f"{CATALOG}/{folder}/{name}.json")
    save_schema(SCHEMA_DIR / folder / f"{name}.json", schema)


def download_crd(version):
    # The standalone catalog omits the CRD schema. Keep the official
    # Kubernetes OpenAPI definition and its reachable local references.
    definitions = fetch(f"{KUBERNETES}/v{version}/api/openapi-spec/swagger.json")["definitions"]
    root = definitions["io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinition"]
    reachable = {}

    def collect(node):
        if isinstance(node, dict):
            reference = node.get("$ref")
            if isinstance(reference, str) and reference.startswith("#/definitions/"):
                name = reference[len("#/definitions/"):]
                if name not in reachable:
                    reachable[name] = definitions[name]
                    collect(definitions[name])
            for value in node.values():
                collect(value)
        elif isinstance(node, list):
            for value in node:
                collect(value)

    collect(root)
    folder = SCHEMA_DIR / f"v{version}-standalone-strict"
    save_schema(folder / "customresourcedefinition-apiextensions-v1.json", dict(root, definitions=reachable))


if __name__ == "__main__":
    with ThreadPoolExecutor(max_workers=8) as workers:
        downloads = [workers.submit(download_native, version, name) for version in VERSIONS for name in NATIVE_SCHEMAS]
        downloads.extend(workers.submit(download_crd, version) for version in VERSIONS)
        for download in downloads:
            download.result()
    print(f"Prepared Kubernetes schemas in {SCHEMA_DIR}")

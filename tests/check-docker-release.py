"""Smoke-test the Docker distribution on localhost."""

import json
import re
import time
from urllib.error import URLError
from urllib.request import Request, urlopen

BASE = "http://127.0.0.1:8080"


def request(path, payload=None):
    options = {}
    if payload is not None:
        options = {"data": json.dumps(payload).encode(), "headers": {"Content-Type": "application/json"}}
    with urlopen(Request(BASE + path, **options), timeout=30) as response:
        return response.read().decode()


for attempt in range(30):
    try:
        index = request("/")
        break
    except URLError:
        if attempt == 29:
            raise
        time.sleep(1)

asset = re.search(r'<script[^>]+src="(/assets/[^"]+\.js)"', index)
assert asset and request(asset[1]), "Built frontend script is missing"

for version in ("1.35", "1.36", "1.37"):
    for tool in ("prometheus", "argo-cd"):
        config = {"tool": tool, "kubernetesVersion": version, "values": ""}
        rendered = request("/api/render", config)
        assert "kind: Deployment" in rendered, f"Missing deployment for {tool}"
        result = json.loads(request("/api/validate", config))
        assert result["status"] == "passed", f"{tool} / {version}: {result}"
        print(f"{tool} / Kubernetes {version}: rendered and validated")


export = json.loads(request("/api/export", {"tool": "prometheus", "kubernetesVersion": "1.37", "values": ""}))
assert "kind: Namespace" in export["manifests"] and "# Why: Use apps/v1 because" in export["manifests"]


application = """apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: guestbook
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: guestbook
"""
usage = {"tool": "argo-cd", "kubernetesVersion": "1.37", "values": application}
result = json.loads(request("/api/usage/validate", usage))
assert result["status"] == "passed", result
usage["values"] = application.replace("project: default", "project: []")
result = json.loads(request("/api/usage/validate", usage))
assert result["status"] == "error", result
print("Packaged Argo CD usage validation accepts valid resources and rejects invalid fields")

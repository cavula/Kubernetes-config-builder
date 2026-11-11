package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/chart/loader"
	release "helm.sh/helm/v4/pkg/release/v1"
)

type chartPin struct {
	version   string
	namespace string
}

var charts = map[string]chartPin{
	"prometheus": {version: "29.45.1", namespace: "monitoring"},
	"argo-cd":    {version: "10.17.1", namespace: "argocd"},
}

type renderRequest struct {
	Tool              string `json:"tool"`
	KubernetesVersion string `json:"kubernetesVersion"`
	Values            string `json:"values"`
}

func render(ctx context.Context, chartDir string, req renderRequest) (string, error) {
	pin, ok := charts[req.Tool]
	if !ok {
		return "", errors.New("unsupported tool")
	}
	switch req.KubernetesVersion {
	case "1.37", "1.36", "1.35":
	default:
		return "", errors.New("unsupported Kubernetes version")
	}

	version, err := common.ParseKubeVersion("v" + req.KubernetesVersion + ".0")
	if err != nil {
		return "", err
	}
	chart, err := loader.Load(filepath.Join(chartDir, req.Tool+"-"+pin.version+".tgz"))
	if err != nil {
		return "", err
	}
	values, err := common.ReadValues([]byte(req.Values))
	if err != nil {
		return "", err
	}
	install := action.NewInstall(new(action.Configuration))
	install.DryRunStrategy = action.DryRunClient
	install.KubeVersion = version
	install.ReleaseName = req.Tool
	install.Namespace = pin.namespace
	install.IncludeCRDs = true

	result, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return "", err
	}
	return result.(*release.Release).Manifest, nil
}

func main() {
	chartDir := flag.String("charts", "../.runtime/charts", "directory containing the pinned Helm chart archives")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/render", func(w http.ResponseWriter, r *http.Request) {
		var req renderRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil || decoder.Decode(new(any)) != io.EOF {
			http.Error(w, "Invalid render request.", http.StatusBadRequest)
			return
		}
		manifest, err := render(r.Context(), *chartDir, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		if _, err := io.WriteString(w, manifest); err != nil {
			log.Print(err)
		}
	})

	server := &http.Server{Addr: "127.0.0.1:8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Chart renderer listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

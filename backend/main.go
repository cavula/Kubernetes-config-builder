package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

//go:embed chart-checksums.txt
var chartChecksums string

func verifiedArchive(chartDir, filename string) ([]byte, error) {
	var expected string
	for line := range strings.SplitSeq(chartChecksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return nil, fmt.Errorf("no pinned SHA256 checksum for %s", filename)
	}
	archive, err := os.ReadFile(filepath.Join(chartDir, filename))
	if err != nil {
		return nil, fmt.Errorf("read pinned chart %s (run backend/download-charts.sh): %w", filename, err)
	}
	if actual := fmt.Sprintf("%x", sha256.Sum256(archive)); actual != expected {
		return nil, fmt.Errorf("SHA256 mismatch for %s: got %s, expected %s (run backend/download-charts.sh)", filename, actual, expected)
	}
	return archive, nil
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
	archive, err := verifiedArchive(chartDir, req.Tool+"-"+pin.version+".tgz")
	if err != nil {
		return "", err
	}
	chart, err := loader.LoadArchive(bytes.NewReader(archive))
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

func decodeRenderRequest(w http.ResponseWriter, r *http.Request) (renderRequest, bool) {
	var req renderRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid configuration request.", http.StatusBadRequest)
		return req, false
	}
	return req, true
}

func main() {
	chartDir := flag.String("charts", "../.runtime/charts", "directory containing the pinned Helm chart archives")
	schemaDir := flag.String("schemas", "../.runtime/schemas", "directory containing local Kubernetes and custom resource schemas")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/render", func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRenderRequest(w, r)
		if !ok {
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

	mux.HandleFunc("POST /api/validate", func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRenderRequest(w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(validate(ctx, *chartDir, *schemaDir, req)); err != nil {
			log.Print(err)
		}
	})

	server := &http.Server{Addr: "127.0.0.1:8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Chart renderer listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

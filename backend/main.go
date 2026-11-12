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
	"slices"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/common"
	chartvalues "helm.sh/helm/v4/pkg/chart/common/util"
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

//go:embed prometheus-arm64-values.json
var prometheusARM64Values []byte

func effectiveValues(req renderRequest) (common.Values, error) {
	values, err := common.ReadValues([]byte(req.Values))
	if err != nil {
		return nil, err
	}
	// Re-evaluate this workaround when the chart pin changes.
	if req.Tool == "prometheus" && charts[req.Tool].version == "29.45.1" {
		overrides, err := common.ReadValues(prometheusARM64Values)
		if err != nil {
			return nil, err
		}
		return chartvalues.MergeTables(overrides, values), nil
	}
	return values, nil
}

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
	Tool              string  `json:"tool"`
	KubernetesVersion string  `json:"kubernetesVersion"`
	Values            string  `json:"values"`
	AlternateValues   *string `json:"alternateValues,omitempty"`
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
	values, err := effectiveValues(req)
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
	rendered := result.(*release.Release)
	if req.Tool == "argo-cd" {
		return argoCDManifests(rendered, pin.namespace), nil
	}
	if req.Tool == "prometheus" && pin.version == "29.45.1" {
		return commentReloaderImages(rendered.Manifest), nil
	}
	return rendered.Manifest, nil
}

func commentReloaderImages(manifest string) string {
	const image = "quay.io/prometheus-operator/prometheus-config-reloader:v0.95.2"
	lines := strings.Split(manifest, "\n")
	var output strings.Builder
	for index, line := range lines {
		if index > 0 {
			output.WriteByte('\n')
		}
		key, value, _ := strings.Cut(strings.TrimSpace(line), ":")
		if key == "image" && strings.Trim(strings.TrimSpace(value), "\"'") == image {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			fmt.Fprintf(&output, "%s# v0.95.2 supplies ARM64 images. Chart 29.45.1 defaults to amd64-only v0.95.1.\n", indent)
		}
		output.WriteString(line)
	}
	return output.String()
}

func argoCDManifests(rendered *release.Release, namespace string) string {
	var output strings.Builder
	output.WriteString("# Apply this file with: kubectl apply --server-side -f argo-cd-manifests.yaml\n")
	output.WriteString("# Server-side apply is required because client-side apply stores a copy in an annotation.\n")
	output.WriteString("# Argo CD's large CRDs exceed Kubernetes' 256 KiB annotation limit.\n")
	fmt.Fprintf(&output, "---\n# Namespace for the Argo CD workloads and Redis initialization.\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n", namespace)
	for _, hook := range rendered.Hooks {
		if !slices.Contains(hook.Events, release.HookPreInstall) ||
			!strings.HasPrefix(hook.Path, "argo-cd/templates/redis-secret-init/") {
			continue
		}
		output.WriteString("---\n")
		if hook.Kind == "Job" {
			output.WriteString("# Run this Redis initialization Job first. It creates the argocd-redis Secret required by the Argo CD workloads.\n")
			output.WriteString("# kubectl apply submits all resources together. Dependent containers wait for the Secret before starting.\n")
		}
		fmt.Fprintf(&output, "# Source: %s\n%s\n", hook.Path, strings.TrimSpace(hook.Manifest))
	}
	output.WriteString(rendered.Manifest)
	return output.String()
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
	listImages := flag.Bool("list-images", false, "list images in the supported chart configurations and exit")
	flag.Parse()
	if *listImages {
		images, err := chartImages(context.Background(), *chartDir, "../templates")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(strings.Join(images, "\n"))
		return
	}

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

	mux.HandleFunc("POST /api/compare", func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRenderRequest(w, r)
		if !ok {
			return
		}
		current, err := render(r.Context(), *chartDir, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		prior := current
		if req.AlternateValues != nil {
			req.Values = *req.AlternateValues
			prior, err = render(r.Context(), *chartDir, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
		}
		changes, err := manifestChanges(current, prior)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Current string           `json:"current"`
			Prior   string           `json:"prior"`
			Changes []resourceChange `json:"changes"`
		}{current, prior, changes}
		if err := json.NewEncoder(w).Encode(response); err != nil {
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

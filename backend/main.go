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
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
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
	CommentDensity    string  `json:"commentDensity,omitempty"`
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

	archive, err := verifiedArchive(chartDir, req.Tool+"-"+pin.version+".tgz")
	if err != nil {
		return "", err
	}
	loaded, err := loader.LoadArchive(bytes.NewReader(archive))
	if err != nil {
		return "", err
	}
	chart, ok := loaded.(*chartv2.Chart)
	if !ok {
		return "", errors.New("unsupported chart API version")
	}
	return renderLoadedChart(ctx, chart, pin, req)
}

type kubernetesVersionMismatch struct {
	required string
}

func (err *kubernetesVersionMismatch) Error() string {
	return fmt.Sprintf("chart requires Kubernetes %s", err.required)
}

func renderLoadedChart(ctx context.Context, chart *chartv2.Chart, pin chartPin, req renderRequest) (string, error) {
	version, err := common.ParseKubeVersion("v" + req.KubernetesVersion + ".0")
	if err != nil {
		return "", err
	}
	if required := chart.Metadata.KubeVersion; required != "" && !chartutil.IsCompatibleRange(required, version.String()) {
		return "", &kubernetesVersionMismatch{required}
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
		return commentManifestSections(argoCDManifests(rendered, pin.namespace)), nil
	}
	manifest := rendered.Manifest
	if req.Tool == "prometheus" && pin.version == "29.45.1" {
		manifest = commentReloaderImages(manifest)
	}
	return commentManifestSections(namespaceManifest(pin.namespace) + prometheusComponentSections(manifest)), nil
}

// Helm groups templates by kind. Keep the server together, then each enabled dependency.
func prometheusComponentSections(manifest string) string {
	components := []struct{ chart, label string }{
		{"", "Prometheus server"},
		{"alertmanager", "Alertmanager"},
		{"kube-state-metrics", "kube-state-metrics"},
		{"prometheus-node-exporter", "Node exporter"},
		{"prometheus-pushgateway", "Pushgateway"},
	}
	groups := make([][]string, len(components))
	for _, document := range strings.Split(manifest, "\n---\n") {
		document = strings.TrimSpace(strings.TrimPrefix(document, "---\n"))
		if document == "" {
			continue
		}
		index := 0
		for i, component := range components[1:] {
			if strings.HasPrefix(document, "# Source: prometheus/charts/"+component.chart+"/") {
				index = i + 1
				break
			}
		}
		groups[index] = append(groups[index], document)
	}
	var output strings.Builder
	for i, documents := range groups {
		for j, document := range documents {
			output.WriteString("---\n")
			if j == 0 {
				fmt.Fprintf(&output, "# Component: %s\n", components[i].label)
			}
			fmt.Fprintln(&output, document)
		}
	}
	return output.String()
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

func namespaceManifest(namespace string) string {
	return fmt.Sprintf("---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n", namespace)
}

func argoCDManifests(rendered *release.Release, namespace string) string {
	var output strings.Builder
	output.WriteString("# Apply this file with: kubectl apply --server-side -f argo-cd-manifests.yaml\n")
	output.WriteString("# Server-side apply is required because client-side apply stores a copy in an annotation.\n")
	output.WriteString("# Argo CD's large CRDs exceed Kubernetes' 256 KiB annotation limit.\n")
	output.WriteString(namespaceManifest(namespace))
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
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	webDir := flag.String("web", "", "directory containing the built frontend")
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

	mux.HandleFunc("POST /api/export", func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRenderRequest(w, r)
		if !ok {
			return
		}
		manifest, err := render(r.Context(), *chartDir, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if req.CommentDensity != "" && req.CommentDensity != "learning" && req.CommentDensity != "compact" {
			http.Error(w, "Unknown YAML comment density.", http.StatusBadRequest)
			return
		}
		result, err := explainExport(manifest, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Print(err)
		}
	})

	mux.HandleFunc("POST /api/bundle", func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeRenderRequest(w, r)
		if !ok {
			return
		}
		bundle, err := helmBundle(*chartDir, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		if _, err := w.Write(bundle); err != nil {
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

	mux.HandleFunc("POST /api/hover", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Manifest          string
			KubernetesVersion string
			Line              int
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20))
		if err := decoder.Decode(&req); err != nil || req.Line < 1 ||
			!slices.Contains([]string{"1.35", "1.36", "1.37"}, req.KubernetesVersion) {
			http.Error(w, "Invalid hover request.", http.StatusBadRequest)
			return
		}
		note := manifestFieldHover(req.Manifest, *schemaDir, req.KubernetesVersion, req.Line)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(note); err != nil && r.Context().Err() == nil {
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

	if *webDir != "" {
		mux.Handle("GET /", http.FileServer(http.Dir(*webDir)))
	}

	server := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Chart renderer listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

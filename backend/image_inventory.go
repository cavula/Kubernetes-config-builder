package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// chartImages inventories all combinations of the Boolean options exposed by
// the builder. The other MVP controls alter settings without selecting images.
func chartImages(ctx context.Context, chartDir, templateDir string) ([]string, error) {
	images := make(map[string]bool)
	for tool := range charts {
		variants, err := imageVariants(filepath.Join(templateDir, tool, "options.json"))
		if err != nil {
			return nil, err
		}
		for _, version := range []string{"1.35", "1.36", "1.37"} {
			for _, values := range variants {
				manifest, err := render(ctx, chartDir, renderRequest{
					Tool: tool, KubernetesVersion: version, Values: values,
				})
				if err != nil {
					return nil, fmt.Errorf("inventory %s on %s with %q: %w", tool, version, values, err)
				}
				if err := collectImages(manifest, images); err != nil {
					return nil, fmt.Errorf("inventory %s on %s: %w", tool, version, err)
				}
			}
		}
	}
	if len(images) == 0 {
		return nil, errors.New("no chart images found")
	}
	result := make([]string, 0, len(images))
	for image := range images {
		result = append(result, image)
	}
	slices.Sort(result)
	return result, nil
}

func imageVariants(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var options []struct {
		Path string
		Type string
	}
	if err := json.Unmarshal(data, &options); err != nil {
		return nil, err
	}
	var toggles []string
	for _, option := range options {
		if option.Type == "boolean" {
			toggles = append(toggles, option.Path)
		}
	}
	variants := []string{""}
	for selection := 0; selection < 1<<len(toggles); selection++ {
		values := make(map[string]any)
		for bit, path := range toggles {
			keys := strings.Split(path, ".")
			parent := values
			for _, key := range keys[:len(keys)-1] {
				if parent[key] == nil {
					parent[key] = make(map[string]any)
				}
				parent = parent[key].(map[string]any)
			}
			parent[keys[len(keys)-1]] = selection&(1<<bit) != 0
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return nil, err
		}
		variants = append(variants, string(encoded))
	}
	return variants, nil
}

func collectImages(manifest string, images map[string]bool) error {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	for {
		var document map[string]any
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		collectContainerImages(document, images)
	}
}

func collectContainerImages(value any, images map[string]bool) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			switch key {
			case "containers", "initContainers", "ephemeralContainers":
				if containers, ok := child.([]any); ok {
					for _, entry := range containers {
						container, _ := entry.(map[string]any)
						if image, ok := container["image"].(string); ok && image != "" {
							images[image] = true
						}
					}
				}
			default:
				collectContainerImages(child, images)
			}
		}
	case []any:
		for _, child := range node {
			collectContainerImages(child, images)
		}
	}
}

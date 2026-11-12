package main

import (
	"context"
	"slices"
	"testing"
)

func TestImageInventoryIncludesChartWorkloads(t *testing.T) {
	images, err := chartImages(context.Background(), pinnedChartFixtures(t), "../templates")
	if err != nil {
		t.Fatal(err)
	}
	for _, image := range []string{
		"ecr-public.aws.com/docker/library/redis:8.6.4-alpine",
		"ghcr.io/dexidp/dex:v2.46.1",
		"quay.io/argoproj/argocd:v3.6.1",
		"quay.io/prometheus-operator/prometheus-config-reloader:v0.95.2",
		"quay.io/prometheus/alertmanager:v0.34.2",
		"quay.io/prometheus/node-exporter:v1.12.1",
		"quay.io/prometheus/prometheus:v3.16.0",
		"quay.io/prometheus/pushgateway:v1.11.4",
		"registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.20.0",
	} {
		if !slices.Contains(images, image) {
			t.Errorf("inventory is missing %s", image)
		}
	}
}

func TestCollectImagesIncludesInitContainersAndIgnoresConfigData(t *testing.T) {
	images := make(map[string]bool)
	err := collectImages(`apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      initContainers:
        - image: init:1
      containers:
        - image: main:1
---
apiVersion: v1
kind: ConfigMap
data:
  image: unused:1
`, images)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 || !images["init:1"] || !images["main:1"] {
		t.Fatalf("wrong inventory: %v", images)
	}
}

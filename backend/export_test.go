package main

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestExplainedExportsPreserveResourcesAndCoverFields(t *testing.T) {
	for _, tool := range []string{"prometheus", "argo-cd"} {
		t.Run(tool, func(t *testing.T) {
			req := renderRequest{Tool: tool, KubernetesVersion: "1.37"}
			raw, err := render(context.Background(), pinnedChartFixtures(t), req)
			if err != nil {
				t.Fatal(err)
			}
			export, err := explainExport(raw, req)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(export.Manifests, "# kubectl apply --server-side -f "+tool) ||
				!strings.Contains(export.Manifests, "# After install:") ||
				!strings.Contains(export.Manifests, "# Why: Use apps/v1 because") {
				t.Fatal("missing access or API-version explanation")
			}
			if !reflect.DeepEqual(resourceJSON(t, raw), resourceJSON(t, withoutScrapeJobsNote(export.Manifests+export.CRDs))) {
				t.Fatal("comments or CRD separation changed the rendered resources")
			}
			if !reflect.DeepEqual(resourceJSON(t, raw), resourceJSON(t, withoutScrapeJobsNote(export.YAML))) {
				t.Fatal("the single-file download must preserve all resources")
			}
			for _, resource := range resourcesForTest(t, export.Manifests) {
				seen := make(map[string]bool)
				for index, line := range resource.lines {
					comment := strings.TrimSpace(line)
					if !strings.HasPrefix(comment, "# Why: ") {
						continue
					}
					nextField := ""
					if index+1 < len(resource.lines) {
						nextField, _, _ = strings.Cut(strings.TrimSpace(resource.lines[index+1]), ":")
					}
					perContainer := nextField == "readinessProbe" || nextField == "livenessProbe" || nextField == "startupProbe" || nextField == "volumeMounts"
					if (seen[comment] && !perContainer) || len(comment) > 160 {
						t.Errorf("%s %s has a repeated or overlong comment: %s", resource.id.Kind, resource.id.Name, comment)
					}
					seen[comment] = true
				}
				if resource.id.Kind == "ClusterRole" {
					document := strings.Join(resource.lines, "\n")
					if strings.Contains(document, "scheduler capacity") || !strings.Contains(document, "operations on the Kubernetes object types") {
						t.Errorf("ClusterRole %s has an incorrect rules explanation", resource.id.Name)
					}
				}
				if resource.id.Kind == "Secret" && (resource.id.Name == "argocd-secret" || resource.id.Name == "argocd-notifications-secret") {
					document := strings.Join(resource.lines, "\n")
					for _, guidance := range []string{"# REQUIRED IF", "Never commit real secrets to Git", "https://argo-cd.readthedocs.io/"} {
						if !strings.Contains(document, guidance) {
							t.Errorf("%s lacks credential guidance %q", resource.id.Name, guidance)
						}
					}
				}
			}
			if tool == "argo-cd" {
				if !strings.Contains(export.CRDs, "A CustomResourceDefinition") ||
					strings.Contains(export.Manifests, "kind: CustomResourceDefinition") {
					t.Fatal("CRDs must be explained in a separate file")
				}
				for _, note := range []string{"REQUIRED IF you enable", "REQUIRED IF setting your own", "argocd account bcrypt", "Never commit real secrets to Git", "argocd admin initial-password"} {
					if !strings.Contains(export.Manifests, note) {
						t.Errorf("missing credential guidance %q", note)
					}
				}
			}
		})
	}
}

func TestPrometheusExportBootstrapsMonitoringNamespace(t *testing.T) {
	req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37"}
	raw, err := render(context.Background(), pinnedChartFixtures(t), req)
	if err != nil {
		t.Fatal(err)
	}
	resources := resourcesForTest(t, raw)
	if len(resources) == 0 || resources[0].id != (resourceID{APIVersion: "v1", Kind: "Namespace", Name: "monitoring"}) {
		t.Fatal("the monitoring namespace must be the first resource")
	}
}

func TestPrometheusExportGroupsServerBeforeEnabledAddons(t *testing.T) {
	for _, values := range []string{
		"",
		"alertmanager: {enabled: false}\nkube-state-metrics: {enabled: false}\nprometheus-node-exporter: {enabled: false}\nprometheus-pushgateway: {enabled: false}\n",
	} {
		req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37", Values: values}
		raw, err := render(context.Background(), pinnedChartFixtures(t), req)
		if err != nil {
			t.Fatal(err)
		}
		export, err := explainExport(raw, req)
		if err != nil {
			t.Fatal(err)
		}
		last := -1
		for _, component := range []string{"Prometheus server", "Alertmanager", "kube-state-metrics", "Node exporter", "Pushgateway"} {
			position := strings.Index(export.YAML, "# Component: "+component+"\n")
			if values != "" && component != "Prometheus server" {
				if position >= 0 {
					t.Errorf("disabled component %s appeared in export", component)
				}
				continue
			}
			if position <= last {
				t.Fatalf("component %s is missing or out of order", component)
			}
			last = position
		}
		addonSeen := false
		for _, resource := range resourcesForTest(t, export.YAML) {
			document := strings.Join(resource.lines, "\n")
			addonSeen = addonSeen || strings.Contains(document, "# Source: prometheus/charts/")
			if addonSeen && strings.Contains(document, "# Source: prometheus/templates/") {
				t.Errorf("main resource %s follows an add-on", resource.id.Name)
			}
		}
	}
}

func TestMinimalPrometheusCommentDensity(t *testing.T) {
	req := renderRequest{Tool: "prometheus", KubernetesVersion: "1.37", Values: "alertmanager: {enabled: false}\nkube-state-metrics: {enabled: false}\nprometheus-node-exporter: {enabled: false}\nprometheus-pushgateway: {enabled: false}\n"}
	raw, err := render(context.Background(), pinnedChartFixtures(t), req)
	if err != nil {
		t.Fatal(err)
	}
	learning, err := explainExport(raw, req)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(learning.YAML, "\n"); count > 700 {
		t.Errorf("Minimal Learning YAML has %d lines, want at most 700", count)
	}
	for _, field := range []string{"readinessProbe", "livenessProbe", "startupProbe", "volumeMounts"} {
		fields := regexp.MustCompile("(?m)^ +"+field+":").FindAllString(learning.YAML, -1)
		commented := regexp.MustCompile("(?m)^ +# Why: [^\n]+\n +"+field+":").FindAllString(learning.YAML, -1)
		if len(fields) != len(commented) || (field == "volumeMounts" && len(fields) < 2) {
			t.Errorf("%s has %d explanations for %d blocks", field, len(commented), len(fields))
		}
	}
	for _, expected := range []string{
		"# Why: " + scrapeJobsNote + "\n    scrape_configs:",
		"compete for the same volume.\n  strategy:\n    type: Recreate",
		"receive these cluster-wide permissions.\nsubjects:",
		"permissions granted to these subjects.\nroleRef:",
		"Use port-forward for direct access from your machine.\n  type: \"ClusterIP\"",
		"Keep one replica running",
	} {
		if !strings.Contains(learning.YAML, expected) {
			t.Errorf("missing explanation %q", expected)
		}
	}
	if strings.Count(learning.YAML, scrapeJobsNote) != 1 {
		t.Error("expected exactly one scrape jobs explanation")
	}
	t.Logf("Minimal Learning YAML has %d lines", strings.Count(learning.YAML, "\n"))
	req.CommentDensity = "compact"
	compact, err := explainExport(raw, req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compact.YAML, "# Why:") || strings.Contains(compact.YAML, "# Source:") ||
		!strings.Contains(compact.YAML, "# Component: Prometheus server") ||
		!strings.Contains(compact.YAML, "# Purpose:") || !strings.Contains(compact.YAML, "# After install:") {
		t.Fatal("Compact should preserve deployment instructions and component and purpose headers")
	}
	if !reflect.DeepEqual(resourceJSON(t, withoutScrapeJobsNote(learning.YAML)), resourceJSON(t, compact.YAML)) {
		t.Fatal("comment density changed resource values")
	}
}

func withoutScrapeJobsNote(manifest string) string {
	return strings.ReplaceAll(manifest, "    # Why: "+scrapeJobsNote+"\n", "")
}

func TestReplicaExplanationUsesCorrectNumber(t *testing.T) {
	for value, expected := range map[string]string{"0": "Keep 0 replicas ", "1": "Keep one replica ", "2": "Keep 2 replicas "} {
		note := fieldWhy(resourceID{Kind: "Deployment"}, []string{"spec", "replicas"}, &yaml.Node{Value: value}, "")
		if !strings.HasPrefix(note, expected) {
			t.Errorf("replicas %s: got %q, want prefix %q", value, note, expected)
		}
	}
}

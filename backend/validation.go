package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type diagnostic struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type validationResult struct {
	Status      string       `json:"status"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

func (result *validationResult) add(check, severity, message string) {
	result.Diagnostics = append(result.Diagnostics, diagnostic{check, severity, message})
	if severity == "error" || result.Status == "passed" {
		result.Status = severity
	}
}

func validate(ctx context.Context, chartDir, schemaDir string, req renderRequest) validationResult {
	result := validationResult{Status: "passed", Diagnostics: []diagnostic{}}
	manifest, err := render(ctx, chartDir, req)
	if err != nil {
		severity := "error"
		if strings.Contains(err.Error(), "incompatible with Kubernetes") {
			severity = "warning"
		}
		result.add("Render", severity, fmt.Sprintf("%s on Kubernetes %s: %v", req.Tool, req.KubernetesVersion, err))
		return result
	}

	if issue := lintChart(ctx, chartDir, req); issue != nil {
		result.add(issue.Check, issue.Severity, issue.Message)
	}
	for _, issue := range validateSchemas(ctx, schemaDir, req.KubernetesVersion, manifest) {
		result.add(issue.Check, issue.Severity, issue.Message)
	}
	return result
}

func lintChart(ctx context.Context, chartDir string, req renderRequest) *diagnostic {
	const check = "Helm lint"
	helm, err := exec.LookPath("helm")
	if err != nil {
		return &diagnostic{check, "incomplete", "Install Helm and make it available on the backend PATH."}
	}

	pin := charts[req.Tool]
	archiveName := req.Tool + "-" + pin.version + ".tgz"
	archive, err := verifiedArchive(chartDir, archiveName)
	if err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	dir, err := os.MkdirTemp("", "kcb-lint-")
	if err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	defer os.RemoveAll(dir)

	chartPath := filepath.Join(dir, archiveName)
	valuesPath := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(chartPath, archive, 0600); err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	values, err := effectiveValues(req)
	if err != nil {
		return &diagnostic{check, "error", err.Error()}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	if err := os.WriteFile(valuesPath, encoded, 0600); err != nil {
		return &diagnostic{check, "incomplete", err.Error()}
	}
	output, err := exec.CommandContext(ctx, helm, "lint", chartPath, "--values", valuesPath,
		"--kube-version", req.KubernetesVersion+".0", "--namespace", pin.namespace).CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(strings.ReplaceAll(string(output), dir+string(os.PathSeparator), ""))
	if ctx.Err() != nil || message == "" {
		return &diagnostic{check, "incomplete", fmt.Sprintf("Helm lint did not complete: %v", err)}
	}
	return &diagnostic{check, "error", message}
}

func validateSchemas(ctx context.Context, schemaDir, version, manifest string) []diagnostic {
	const check = "kubeconform"
	binary, err := exec.LookPath("kubeconform")
	if err != nil {
		return []diagnostic{{check, "incomplete", "Install kubeconform and make it available on the backend PATH."}}
	}
	if info, err := os.Stat(schemaDir); err != nil || !info.IsDir() {
		return []diagnostic{{check, "incomplete", "Local schema directory is unavailable: " + schemaDir}}
	}
	cmd := exec.CommandContext(ctx, binary, "-strict", "-summary", "-output", "json",
		"-kubernetes-version", version+".0", "-schema-location", schemaDir,
		"-schema-location", filepath.Join(schemaDir, "custom", "{{.Group}}", "{{.ResourceKind}}_{{.ResourceAPIVersion}}.json"))
	cmd.Stdin = strings.NewReader(manifest)
	output, err := cmd.Output()

	var report struct {
		Resources []struct {
			Kind, Name, Status, Msg string
		}
		Summary struct{ Valid, Invalid, Errors, Skipped int }
	}
	if json.Unmarshal(output, &report) != nil || ctx.Err() != nil {
		var exit *exec.ExitError
		message := fmt.Sprintf("Schema validation did not complete: %v", err)
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			message += ": " + strings.TrimSpace(string(exit.Stderr))
		}
		return []diagnostic{{check, "incomplete", message}}
	}

	var issues []diagnostic
	for _, resource := range report.Resources {
		severity := "incomplete"
		switch strings.ToLower(resource.Status) {
		case "statusvalid", "valid":
			continue
		case "statusinvalid", "invalid":
			severity = "error"
		}
		issues = append(issues, diagnostic{check, severity,
			strings.TrimSpace(fmt.Sprintf("%s %s: %s", resource.Kind, resource.Name, resource.Msg))})
	}
	if len(issues) == 0 && (err != nil || report.Summary.Valid == 0 || report.Summary.Invalid+report.Summary.Errors+report.Summary.Skipped != 0) {
		issues = append(issues, diagnostic{check, "incomplete", "Schema validation did not report a complete result."})
	}
	return issues
}

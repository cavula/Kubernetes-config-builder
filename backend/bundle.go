package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func helmBundle(chartDir string, req renderRequest) ([]byte, error) {
	pin, ok := charts[req.Tool]
	if !ok {
		return nil, fmt.Errorf("unsupported chart %q", req.Tool)
	}
	filename := req.Tool + "-" + pin.version + ".tgz"
	archive, err := verifiedArchive(chartDir, filename)
	if err != nil {
		return nil, err
	}
	var commands map[string]struct{ Bundle string }
	var guides map[string][]string
	if err := json.Unmarshal(deploymentCommandsJSON, &commands); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(exportGuideJSON, &guides); err != nil {
		return nil, err
	}
	readme := commands[req.Tool].Bundle + "\n\nRun the command above from the unzipped directory. Helm uses the local chart archive and its bundled chart dependencies. Your Kubernetes cluster needs access to the container images or must already have them.\n\n" +
		strings.Join(guides[req.Tool], "\n") + "\n"
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range []struct {
		name string
		data []byte
	}{
		{filename, archive},
		{req.Tool + "-values.yaml", []byte(req.Values)},
		{"README.txt", []byte(readme)},
	} {
		entry, err := writer.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(file.data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

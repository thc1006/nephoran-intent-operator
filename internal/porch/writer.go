package porch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteIntent writes the intent to the output directory in the specified format
// format can be "full" (default YAML) or "smp" (Strategic Merge Patch JSON)
func WriteIntent(intent interface{}, outDir string, format string) error {
	// Extract fields using JSON tags via marshal/unmarshal
	data, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	var fields struct {
		IntentType string `json:"intent_type"`
		Target     string `json:"target"`
		Namespace  string `json:"namespace"`
		Replicas   int    `json:"replicas"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal intent: %w", err)
	}

	// Validate
	if fields.IntentType != "scaling" {
		return fmt.Errorf("only scaling intent is supported in MVP")
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var content []byte
	var filename string

	if format == "smp" {
		// Strategic Merge Patch format (JSON)
		smp := map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]string{
				"name":      fields.Target,
				"namespace": fields.Namespace,
			},
			"spec": map[string]int{
				"replicas": fields.Replicas,
			},
		}
		content, err = json.MarshalIndent(smp, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal SMP: %w", err)
		}
		filename = "scaling-patch.json"
	} else {
		// Full manifest format (YAML) - default
		yaml := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
`, fields.Target, fields.Namespace, fields.Replicas)
		content = []byte(yaml)
		filename = "scaling-patch.yaml"
	}

	// Write file
	dst := filepath.Join(outDir, filename)
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println("wrote:", dst)
	fmt.Println("next: (optional) kpt live init/apply under", outDir)

	return nil
}

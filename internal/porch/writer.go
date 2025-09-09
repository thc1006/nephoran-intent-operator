package porch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

<<<<<<< HEAD
// WriteIntent writes the intent to the output directory in the specified format.

// format can be "full" (default YAML) or "smp" (Strategic Merge Patch JSON).

func WriteIntent(intent interface{}, outDir, format string) error {
	// Extract fields using JSON tags via marshal/unmarshal.

=======
// WriteIntent writes the intent to the output directory in the specified format
// format can be "full" (default YAML) or "smp" (Strategic Merge Patch JSON)
func WriteIntent(intent interface{}, outDir string, format string) error {
	// Extract fields using JSON tags via marshal/unmarshal
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	data, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	var fields struct {
		IntentType string `json:"intent_type"`
<<<<<<< HEAD

		Target string `json:"target"`

		Namespace string `json:"namespace"`

		Replicas int `json:"replicas"`
	}

=======
		Target     string `json:"target"`
		Namespace  string `json:"namespace"`
		Replicas   int    `json:"replicas"`
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal intent: %w", err)
	}

<<<<<<< HEAD
	// Validate.

=======
	// Validate
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if fields.IntentType != "scaling" {
		return fmt.Errorf("only scaling intent is supported in MVP")
	}

<<<<<<< HEAD
	// Create output directory.

=======
	// Create output directory
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var content []byte
<<<<<<< HEAD

	var filename string

	if format == "smp" {

		// Strategic Merge Patch format (JSON).

		smp := map[string]interface{}{
=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			"spec": map[string]int{
				"replicas": fields.Replicas,
			},
		}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		content, err = json.MarshalIndent(smp, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal SMP: %w", err)
		}
<<<<<<< HEAD

		filename = "scaling-patch.json"

	} else {

		// Full manifest format (YAML) - default.

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

	// Write file.

	dst := filepath.Join(outDir, filename)

	if err := os.WriteFile(dst, content, 0o640); err != nil {
=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println("wrote:", dst)
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	fmt.Println("next: (optional) kpt live init/apply under", outDir)

	return nil
}

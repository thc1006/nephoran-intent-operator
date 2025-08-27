package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"
)

// Generator creates KRM patches from intents
type Generator struct {
	Intent    *Intent
	OutputDir string
}

// NewGenerator creates a new patch generator
func NewGenerator(intent *Intent, outputDir string) *Generator {
	return &Generator{
		Intent:    intent,
		OutputDir: outputDir,
	}
}

// Generate creates the KRM patch files
func (g *Generator) Generate() error {
	// Create package directory
	packageName := fmt.Sprintf("%s-scaling-%d", g.Intent.Target, time.Now().Unix())
	packageDir := filepath.Join(g.OutputDir, packageName)

	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Generate Kptfile
	if err := g.generateKptfile(packageDir); err != nil {
		return err
	}

	// Generate patch YAML
	if err := g.generatePatch(packageDir); err != nil {
		return err
	}

	// Generate setter configuration
	if err := g.generateSetters(packageDir); err != nil {
		return err
	}

	// Generate README
	if err := g.generateReadme(packageDir); err != nil {
		return err
	}

	fmt.Printf("\n=== Patch Package Generated ===\n")
	fmt.Printf("Package: %s\n", packageName)
	fmt.Printf("Target: %s\n", g.Intent.Target)
	fmt.Printf("Namespace: %s\n", g.Intent.Namespace)
	fmt.Printf("Replicas: %d\n", g.Intent.Replicas)
	fmt.Printf("Location: %s\n", packageDir)
	fmt.Printf("================================\n\n")

	return nil
}

func (g *Generator) generateKptfile(packageDir string) error {
	kptfile := map[string]interface{}{
		"apiVersion": "kpt.dev/v1",
		"kind":       "Kptfile",
		"metadata": map[string]interface{}{
			"name": filepath.Base(packageDir),
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"info": map[string]interface{}{
			"description": fmt.Sprintf("Scaling patch for %s", g.Intent.Target),
		},
		"pipeline": map[string]interface{}{
			"mutators": []map[string]interface{}{
				{
					"image": "gcr.io/kpt-fn/apply-setters:v0.2.0",
					"configMap": map[string]interface{}{
						"replicas": fmt.Sprintf("%d", g.Intent.Replicas),
					},
				},
			},
		},
	}

	data, err := yaml.Marshal(kptfile)
	if err != nil {
		return fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	kptfilePath := filepath.Join(packageDir, "Kptfile")
	return os.WriteFile(kptfilePath, data, 0644)
}

func (g *Generator) generatePatch(packageDir string) error {
	patch := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      g.Intent.Target,
			"namespace": g.Intent.Namespace,
		},
		"spec": map[string]interface{}{
			"replicas": g.Intent.Replicas, // kpt-set: ${replicas}
		},
	}

	data, err := yaml.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	// Add kpt setter comment
	patchContent := string(data)
	patchContent = fmt.Sprintf("# kpt-file: deployment-patch.yaml\n%s", patchContent)

	patchPath := filepath.Join(packageDir, "deployment-patch.yaml")
	return os.WriteFile(patchPath, []byte(patchContent), 0644)
}

func (g *Generator) generateSetters(packageDir string) error {
	setters := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "setters",
			"namespace": g.Intent.Namespace,
		},
		"data": map[string]string{
			"replicas":  fmt.Sprintf("%d", g.Intent.Replicas),
			"target":    g.Intent.Target,
			"namespace": g.Intent.Namespace,
		},
	}

	data, err := yaml.Marshal(setters)
	if err != nil {
		return fmt.Errorf("failed to marshal setters: %w", err)
	}

	settersPath := filepath.Join(packageDir, "setters.yaml")
	return os.WriteFile(settersPath, data, 0644)
}

func (g *Generator) generateReadme(packageDir string) error {
	readme := fmt.Sprintf(`# KRM Scaling Patch

## Target
- Deployment: %s
- Namespace: %s
- Replicas: %d

## Usage
Apply this patch using kpt:
`+"```bash"+`
kpt fn eval --image gcr.io/kpt-fn/apply-setters:v0.2.0
kubectl apply -f deployment-patch.yaml
`+"```"+`

## Files
- Kptfile: Package metadata
- deployment-patch.yaml: Deployment replica patch
- setters.yaml: Configuration values
`, g.Intent.Target, g.Intent.Namespace, g.Intent.Replicas)

	readmePath := filepath.Join(packageDir, "README.md")
	return os.WriteFile(readmePath, []byte(readme), 0644)
}

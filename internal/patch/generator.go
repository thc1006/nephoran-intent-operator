package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"
)

// Generator creates KRM patches from intents.

type Generator struct {
	Intent *Intent

	OutputDir string

	Logger logr.Logger
}

// NewGenerator creates a new patch generator.

func NewGenerator(intent *Intent, outputDir string, logger logr.Logger) *Generator {
	return &Generator{
		Intent: intent,

		OutputDir: outputDir,

		Logger: logger,
	}
}

// Generate creates the KRM patch files.

func (g *Generator) Generate() error {
	// Create package directory.

	packageName := fmt.Sprintf("%s-scaling-%d", g.Intent.Target, time.Now().Unix())

	packageDir := filepath.Join(g.OutputDir, packageName)

	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to create package directory",
				"packageDir", packageDir,
				"outputDir", g.OutputDir,
				"packageName", packageName)
		}
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Generate Kptfile.

	if err := g.generateKptfile(packageDir); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to generate Kptfile",
				"packageDir", packageDir,
				"packageName", packageName)
		}
		return err
	}

	// Generate patch YAML.

	if err := g.generatePatch(packageDir); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to generate patch YAML",
				"packageDir", packageDir,
				"packageName", packageName,
				"target", g.Intent.Target,
				"namespace", g.Intent.Namespace)
		}
		return err
	}

	// Generate setter configuration.

	if err := g.generateSetters(packageDir); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to generate setters",
				"packageDir", packageDir,
				"packageName", packageName)
		}
		return err
	}

	// Generate README.

	if err := g.generateReadme(packageDir); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to generate README",
				"packageDir", packageDir,
				"packageName", packageName)
		}
		return err
	}

	if g.Logger.Enabled() {
		g.Logger.Info("Patch package generated successfully",
			"package", packageName,
			"target", g.Intent.Target,
			"namespace", g.Intent.Namespace,
			"replicas", g.Intent.Replicas,
			"location", packageDir)
	}

	return nil
}

func (g *Generator) generateKptfile(packageDir string) error {
	kptfile := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": filepath.Base(packageDir),
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"info": map[string]interface{}{},
		"pipeline": map[string]interface{}{
			"mutators": []map[string]interface{}{
				{
					"image":     "gcr.io/kpt-fn/apply-setters:v0.2.0",
					"configMap": map[string]interface{}{},
				},
			},
		},
	}

	data, err := yaml.Marshal(kptfile)
	if err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to marshal Kptfile",
				"packageDir", packageDir,
				"packageName", filepath.Base(packageDir))
		}
		return fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	kptfilePath := filepath.Join(packageDir, "Kptfile")

	if err := os.WriteFile(kptfilePath, data, 0o640); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to write Kptfile to disk",
				"kptfilePath", kptfilePath,
				"packageDir", packageDir)
		}
		return err
	}
	return nil
}

func (g *Generator) generatePatch(packageDir string) error {
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      g.Intent.Target,
			"namespace": g.Intent.Namespace,
		},
		"spec": map[string]interface{}{},
	}

	data, err := yaml.Marshal(patch)
	if err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to marshal patch YAML",
				"target", g.Intent.Target,
				"namespace", g.Intent.Namespace,
				"packageDir", packageDir)
		}
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	// Add kpt setter comment.

	patchContent := string(data)

	patchContent = fmt.Sprintf("# kpt-file: deployment-patch.yaml\n%s", patchContent)

	patchPath := filepath.Join(packageDir, "deployment-patch.yaml")

	if err := os.WriteFile(patchPath, []byte(patchContent), 0o640); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to write patch YAML to disk",
				"patchPath", patchPath,
				"target", g.Intent.Target,
				"namespace", g.Intent.Namespace)
		}
		return err
	}
	return nil
}

func (g *Generator) generateSetters(packageDir string) error {
	setters := map[string]interface{}{
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
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to marshal setters YAML",
				"replicas", g.Intent.Replicas,
				"target", g.Intent.Target,
				"namespace", g.Intent.Namespace)
		}
		return fmt.Errorf("failed to marshal setters: %w", err)
	}

	settersPath := filepath.Join(packageDir, "setters.yaml")

	if err := os.WriteFile(settersPath, data, 0o640); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to write setters YAML to disk",
				"settersPath", settersPath,
				"packageDir", packageDir)
		}
		return err
	}
	return nil
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

	if err := os.WriteFile(readmePath, []byte(readme), 0o640); err != nil {
		if g.Logger.Enabled() {
			g.Logger.Error(err, "Failed to write README to disk",
				"readmePath", readmePath,
				"packageDir", packageDir)
		}
		return err
	}
	return nil
}

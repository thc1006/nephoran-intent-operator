package patchgen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"
)

// PatchPackage represents a KRM patch package structure
type PatchPackage struct {
	Kptfile     *Kptfile    `yaml:"-"`
	PatchFile   *PatchFile  `yaml:"-"`
	OutputDir   string      `yaml:"-"`
	Intent      *Intent     `yaml:"-"`
}

// Kptfile represents the kpt package metadata
type Kptfile struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   KptMetadata `yaml:"metadata"`
	Info       KptInfo     `yaml:"info"`
	Pipeline   KptPipeline `yaml:"pipeline"`
}

// KptMetadata contains package metadata
type KptMetadata struct {
	Name string `yaml:"name"`
}

// KptInfo contains package information
type KptInfo struct {
	Description string `yaml:"description"`
}

// KptPipeline defines the kpt pipeline configuration
type KptPipeline struct {
	Mutators []KptMutator `yaml:"mutators"`
}

// KptMutator defines a kpt mutator function
type KptMutator struct {
	Image  string            `yaml:"image"`
	ConfigMap map[string]string `yaml:"configMap"`
}

// PatchFile represents a strategic merge patch
type PatchFile struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   PatchMetadata     `yaml:"metadata"`
	Spec       PatchSpec         `yaml:"spec"`
}

// PatchMetadata contains patch metadata
type PatchMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Annotations map[string]string `yaml:"annotations"`
}

// PatchSpec contains the patch specification
type PatchSpec struct {
	Replicas int `yaml:"replicas"`
}

// readFile is a helper function to read files
func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// writeFile is a helper function to write files
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// NewPatchPackage creates a new patch package from an intent
func NewPatchPackage(intent *Intent, outputDir string) *PatchPackage {
	return &PatchPackage{
		Intent:    intent,
		OutputDir: outputDir,
		Kptfile: &Kptfile{
			APIVersion: "kpt.dev/v1",
			Kind:       "Kptfile",
			Metadata: KptMetadata{
				Name: fmt.Sprintf("%s-scaling-patch", intent.Target),
			},
			Info: KptInfo{
				Description: fmt.Sprintf("Structured patch to scale %s deployment to %d replicas", intent.Target, intent.Replicas),
			},
			Pipeline: KptPipeline{
				Mutators: []KptMutator{
					{
						Image: "gcr.io/kpt-fn/apply-replacements:v0.1.1",
						ConfigMap: map[string]string{
							"apply-replacements": "true",
						},
					},
				},
			},
		},
		PatchFile: &PatchFile{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Metadata: PatchMetadata{
				Name:      intent.Target,
				Namespace: intent.Namespace,
				Annotations: map[string]string{
					"config.kubernetes.io/merge-policy": "replace",
					"nephoran.io/intent-type":          intent.IntentType,
					"nephoran.io/generated-at":         time.Now().UTC().Format(time.RFC3339),
				},
			},
			Spec: PatchSpec{
				Replicas: intent.Replicas,
			},
		},
	}
}

// Generate creates the patch package files in the output directory
func (p *PatchPackage) Generate() error {
	packageDir := filepath.Join(p.OutputDir, p.Kptfile.Metadata.Name)
	
	// Create package directory
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory %s: %w", packageDir, err)
	}

	// Generate Kptfile
	if err := p.generateKptfile(packageDir); err != nil {
		return fmt.Errorf("failed to generate Kptfile: %w", err)
	}

	// Generate patch file
	if err := p.generatePatchFile(packageDir); err != nil {
		return fmt.Errorf("failed to generate patch file: %w", err)
	}

	// Generate README for documentation
	if err := p.generateReadme(packageDir); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	return nil
}

// generateKptfile creates the Kptfile for the package
func (p *PatchPackage) generateKptfile(packageDir string) error {
	kptfileData, err := yaml.Marshal(p.Kptfile)
	if err != nil {
		return fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	kptfilePath := filepath.Join(packageDir, "Kptfile")
	return writeFile(kptfilePath, kptfileData)
}

// generatePatchFile creates the strategic merge patch file
func (p *PatchPackage) generatePatchFile(packageDir string) error {
	patchData, err := yaml.Marshal(p.PatchFile)
	if err != nil {
		return fmt.Errorf("failed to marshal patch file: %w", err)
	}

	patchPath := filepath.Join(packageDir, "scaling-patch.yaml")
	return writeFile(patchPath, patchData)
}

// generateReadme creates a README file for the package
func (p *PatchPackage) generateReadme(packageDir string) error {
	readmeContent := fmt.Sprintf(`# %s

This package contains a structured patch to scale the %s deployment.

## Intent Details
- **Target**: %s
- **Namespace**: %s  
- **Replicas**: %d
- **Intent Type**: %s

## Files
- ` + "`Kptfile`" + `: Package metadata and pipeline configuration
- ` + "`scaling-patch.yaml`" + `: Strategic merge patch for deployment scaling

## Usage
Apply this patch package using kpt or Porch:

` + "```bash" + `
kpt fn eval . --image gcr.io/kpt-fn/apply-replacements:v0.1.1
` + "```" + `

## Generated
Generated at: %s
`,
		p.Kptfile.Metadata.Name,
		p.Intent.Target,
		p.Intent.Target,
		p.Intent.Namespace,
		p.Intent.Replicas,
		p.Intent.IntentType,
		time.Now().UTC().Format(time.RFC3339))

	readmePath := filepath.Join(packageDir, "README.md")
	return writeFile(readmePath, []byte(readmeContent))
}

// GetPackagePath returns the full path to the generated package
func (p *PatchPackage) GetPackagePath() string {
	return filepath.Join(p.OutputDir, p.Kptfile.Metadata.Name)
}
package generator

import (
	"fmt"
	"time"

<<<<<<< HEAD
	"sigs.k8s.io/yaml"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// KptfileGenerator generates Kptfile for KRM packages.

type KptfileGenerator struct{}

// NewKptfileGenerator creates a new Kptfile generator.

=======
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
	"sigs.k8s.io/yaml"
)

// KptfileGenerator generates Kptfile for KRM packages
type KptfileGenerator struct{}

// NewKptfileGenerator creates a new Kptfile generator
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func NewKptfileGenerator() *KptfileGenerator {
	return &KptfileGenerator{}
}

<<<<<<< HEAD
// Kptfile represents the structure of a Kptfile.

type Kptfile struct {
	APIVersion string `yaml:"apiVersion"`

	Kind string `yaml:"kind"`

	Metadata KptfileMetadata `yaml:"metadata"`

	Info KptfileInfo `yaml:"info"`

	Pipeline *KptfilePipeline `yaml:"pipeline,omitempty"`
}

// KptfileMetadata contains metadata for the Kptfile.

type KptfileMetadata struct {
	Name string `yaml:"name"`

	Namespace string `yaml:"namespace,omitempty"`

	Labels map[string]string `yaml:"labels,omitempty"`

	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// KptfileInfo contains package information.

type KptfileInfo struct {
	Description string `yaml:"description"`

	Site string `yaml:"site,omitempty"`

	License string `yaml:"license,omitempty"`

	Keywords []string `yaml:"keywords,omitempty"`
}

// KptfilePipeline defines the mutation pipeline.

=======
// Kptfile represents the structure of a Kptfile
type Kptfile struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   KptfileMetadata `yaml:"metadata"`
	Info       KptfileInfo    `yaml:"info"`
	Pipeline   *KptfilePipeline `yaml:"pipeline,omitempty"`
}

// KptfileMetadata contains metadata for the Kptfile
type KptfileMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// KptfileInfo contains package information
type KptfileInfo struct {
	Description string `yaml:"description"`
	Site        string `yaml:"site,omitempty"`
	License     string `yaml:"license,omitempty"`
	Keywords    []string `yaml:"keywords,omitempty"`
}

// KptfilePipeline defines the mutation pipeline
>>>>>>> 6835433495e87288b95961af7173d866977175ff
type KptfilePipeline struct {
	Mutators []KptfileMutator `yaml:"mutators,omitempty"`
}

<<<<<<< HEAD
// KptfileMutator defines a pipeline mutator.

type KptfileMutator struct {
	Image string `yaml:"image"`

	ConfigMap map[string]interface{} `yaml:"configMap,omitempty"`

	ConfigPath string `yaml:"configPath,omitempty"`

	Name string `yaml:"name,omitempty"`
}

// Generate creates a Kptfile from a scaling intent.

func (g *KptfileGenerator) Generate(intent *intent.ScalingIntent) ([]byte, error) {
	kptfile := &Kptfile{
		APIVersion: "kpt.dev/v1",

		Kind: "Kptfile",

		Metadata: KptfileMetadata{
			Name: fmt.Sprintf("%s-package", intent.Target),

			Namespace: intent.Namespace,

			Labels: map[string]string{
				"app": intent.Target,

				"app.kubernetes.io/name": intent.Target,

				"app.kubernetes.io/component": "nf-simulator",

				"app.kubernetes.io/part-of": "nephoran-intent-operator",

				"app.kubernetes.io/managed-by": "porch-direct",
			},

			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",

				"nephoran.com/intent-type": intent.IntentType,

				"nephoran.com/source": intent.Source,

				"nephoran.com/generated-at": time.Now().Format(time.RFC3339),

				"nephoran.com/target": intent.Target,

				"nephoran.com/namespace": intent.Namespace,

				"nephoran.com/replicas": fmt.Sprintf("%d", intent.Replicas),
			},
		},

		Info: KptfileInfo{
			Description: fmt.Sprintf("KRM package for %s CNF scaling to %d replicas", intent.Target, intent.Replicas),

			Site: "https://github.com/thc1006/nephoran-intent-operator",

			License: "Apache-2.0",

			Keywords: []string{
				"nephoran",

				"scaling",

				"nf-simulator",

				"cnf",

				"oran",
			},
		},

=======
// KptfileMutator defines a pipeline mutator
type KptfileMutator struct {
	Image        string                 `yaml:"image"`
	ConfigMap    map[string]interface{} `yaml:"configMap,omitempty"`
	ConfigPath   string                 `yaml:"configPath,omitempty"`
	Name         string                 `yaml:"name,omitempty"`
}

// Generate creates a Kptfile from a scaling intent
func (g *KptfileGenerator) Generate(intent *intent.ScalingIntent) ([]byte, error) {
	kptfile := &Kptfile{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: KptfileMetadata{
			Name:      fmt.Sprintf("%s-package", intent.Target),
			Namespace: intent.Namespace,
			Labels: map[string]string{
				"app":                          intent.Target,
				"app.kubernetes.io/name":       intent.Target,
				"app.kubernetes.io/component":  "nf-simulator",
				"app.kubernetes.io/part-of":    "nephoran-intent-operator",
				"app.kubernetes.io/managed-by": "porch-direct",
			},
			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",
				"nephoran.com/intent-type":           intent.IntentType,
				"nephoran.com/source":                intent.Source,
				"nephoran.com/generated-at":          time.Now().Format(time.RFC3339),
				"nephoran.com/target":                intent.Target,
				"nephoran.com/namespace":             intent.Namespace,
				"nephoran.com/replicas":              fmt.Sprintf("%d", intent.Replicas),
			},
		},
		Info: KptfileInfo{
			Description: fmt.Sprintf("KRM package for %s CNF scaling to %d replicas", intent.Target, intent.Replicas),
			Site:        "https://github.com/thc1006/nephoran-intent-operator",
			License:     "Apache-2.0",
			Keywords: []string{
				"nephoran",
				"scaling",
				"nf-simulator",
				"cnf",
				"oran",
			},
		},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		Pipeline: &KptfilePipeline{
			Mutators: []KptfileMutator{
				{
					Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
<<<<<<< HEAD

					ConfigMap: make(map[string]interface{}),
				},

				{
					Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",

					ConfigMap: make(map[string]interface{}),
=======
					ConfigMap: map[string]interface{}{
						"app":                          intent.Target,
						"app.kubernetes.io/name":       intent.Target,
						"app.kubernetes.io/component":  "nf-simulator",
						"app.kubernetes.io/part-of":    "nephoran-intent-operator",
						"app.kubernetes.io/managed-by": "porch-direct",
					},
				},
				{
					Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",
					ConfigMap: map[string]interface{}{
						"nephoran.com/intent-type": intent.IntentType,
						"nephoran.com/source":      intent.Source,
					},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				},
			},
		},
	}

<<<<<<< HEAD
	// Add correlation ID if provided.

	if intent.CorrelationID != "" {

		kptfile.Metadata.Annotations["nephoran.com/correlation-id"] = intent.CorrelationID

		// Add to pipeline mutator as well.

		kptfile.Pipeline.Mutators[1].ConfigMap["nephoran.com/correlation-id"] = intent.CorrelationID

	}

	// Add reason if provided.

	if intent.Reason != "" {

		kptfile.Metadata.Annotations["nephoran.com/reason"] = intent.Reason

		kptfile.Pipeline.Mutators[1].ConfigMap["nephoran.com/reason"] = intent.Reason

	}

	// Convert to YAML.

=======
	// Add correlation ID if provided
	if intent.CorrelationID != "" {
		kptfile.Metadata.Annotations["nephoran.com/correlation-id"] = intent.CorrelationID
		// Add to pipeline mutator as well
		kptfile.Pipeline.Mutators[1].ConfigMap["nephoran.com/correlation-id"] = intent.CorrelationID
	}

	// Add reason if provided
	if intent.Reason != "" {
		kptfile.Metadata.Annotations["nephoran.com/reason"] = intent.Reason
		kptfile.Pipeline.Mutators[1].ConfigMap["nephoran.com/reason"] = intent.Reason
	}

	// Convert to YAML
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	yamlData, err := yaml.Marshal(kptfile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Kptfile to YAML: %w", err)
	}

	return yamlData, nil
}

<<<<<<< HEAD
// GenerateMinimal creates a minimal Kptfile without pipeline for simple packages.

func (g *KptfileGenerator) GenerateMinimal(intent *intent.ScalingIntent) ([]byte, error) {
	kptfile := &Kptfile{
		APIVersion: "kpt.dev/v1",

		Kind: "Kptfile",

		Metadata: KptfileMetadata{
			Name: fmt.Sprintf("%s-package", intent.Target),

			Labels: map[string]string{
				"app": intent.Target,

				"app.kubernetes.io/managed-by": "porch-direct",
			},

			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",

				"nephoran.com/intent-type": intent.IntentType,

				"nephoran.com/generated-at": time.Now().Format(time.RFC3339),
			},
		},

=======
// GenerateMinimal creates a minimal Kptfile without pipeline for simple packages
func (g *KptfileGenerator) GenerateMinimal(intent *intent.ScalingIntent) ([]byte, error) {
	kptfile := &Kptfile{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: KptfileMetadata{
			Name: fmt.Sprintf("%s-package", intent.Target),
			Labels: map[string]string{
				"app":                          intent.Target,
				"app.kubernetes.io/managed-by": "porch-direct",
			},
			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",
				"nephoran.com/intent-type":           intent.IntentType,
				"nephoran.com/generated-at":          time.Now().Format(time.RFC3339),
			},
		},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		Info: KptfileInfo{
			Description: fmt.Sprintf("Minimal KRM package for %s scaling", intent.Target),
		},
	}

	yamlData, err := yaml.Marshal(kptfile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal minimal Kptfile to YAML: %w", err)
	}

	return yamlData, nil
<<<<<<< HEAD
}

=======
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

package generator

import (
	"encoding/json"
	"fmt"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// KptfileGenerator generates Kptfile for KRM packages.

type KptfileGenerator struct{}

// NewKptfileGenerator creates a new Kptfile generator.

func NewKptfileGenerator() *KptfileGenerator {
	return &KptfileGenerator{}
}

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

type KptfilePipeline struct {
	Mutators []KptfileMutator `yaml:"mutators,omitempty"`
}

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

		Pipeline: &KptfilePipeline{
			Mutators: []KptfileMutator{
				{
					Image: "gcr.io/kpt-fn/set-labels:v0.2.0",

					ConfigMap: json.RawMessage("{}"),
				},

				{
					Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",

					ConfigMap: json.RawMessage("{}"),
				},
			},
		},
	}

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

	yamlData, err := yaml.Marshal(kptfile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Kptfile to YAML: %w", err)
	}

	return yamlData, nil
}

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

		Info: KptfileInfo{
			Description: fmt.Sprintf("Minimal KRM package for %s scaling", intent.Target),
		},
	}

	yamlData, err := yaml.Marshal(kptfile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal minimal Kptfile to YAML: %w", err)
	}

	return yamlData, nil
}

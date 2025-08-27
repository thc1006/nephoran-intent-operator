package patchgen

// Intent represents the structure of an intent for scaling
type Intent struct {
	IntentType    string `json:"intent_type"`
	Target        string `json:"target"`
	Namespace     string `json:"namespace"`
	Replicas      int    `json:"replicas"`
	Reason        string `json:"reason,omitempty"`
	Source        string `json:"source,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// PatchPackage represents a KRM patch package structure
type PatchPackage struct {
	Kptfile   *Kptfile   `yaml:"-"`
	PatchFile *PatchFile `yaml:"-"`
	OutputDir string     `yaml:"-"`
	Intent    *Intent    `yaml:"-"`
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
	Image     string            `yaml:"image"`
	ConfigMap map[string]string `yaml:"configMap"`
}

// PatchFile represents a strategic merge patch
type PatchFile struct {
	APIVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   PatchMetadata `yaml:"metadata"`
	Spec       PatchSpec     `yaml:"spec"`
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=nephio;o-ran
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type NetworkIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkIntentSpec   `json:"spec,omitempty"`
	Status NetworkIntentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type NetworkIntentSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=low;medium;high
	ScalingPriority string `json:"scalingPriority"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	TargetClusters []string `json:"targetClusters,omitempty"`
	
	// ScalingIntent defines the scaling behavior
	// +kubebuilder:validation:Optional
	ScalingIntent map[string]interface{} `json:"scalingIntent,omitempty"`

	// Deployment defines deployment specifications for network functions
	// +kubebuilder:validation:Optional
	Deployment DeploymentSpec `json:"deployment,omitempty"`
}

// DeploymentSpec defines the deployment configuration for network functions
// +kubebuilder:object:generate=true
type DeploymentSpec struct {
	// ClusterSelector defines which clusters to deploy to
	// +kubebuilder:validation:Optional
	ClusterSelector map[string]string `json:"clusterSelector,omitempty"`

	// NetworkFunctions defines the list of network functions to deploy
	// +kubebuilder:validation:Optional
	NetworkFunctions []NetworkFunction `json:"networkFunctions,omitempty"`

	// Replicas specifies the desired number of replicas
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas,omitempty"`
}

// NetworkFunction defines a network function configuration
// +kubebuilder:object:generate=true
type NetworkFunction struct {
	// Name of the network function
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the network function (CNF, VNF, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=CNF;VNF;PNF
	Type string `json:"type"`

	// Version of the network function
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`

	// Configuration specific to this network function
	// +kubebuilder:validation:Optional
	Config map[string]interface{} `json:"config,omitempty"`

	// Resources required for this network function
	// +kubebuilder:validation:Optional
	Resources NetworkFunctionResources `json:"resources,omitempty"`
}

// NetworkFunctionResources defines resource requirements for a network function
// +kubebuilder:object:generate=true
type NetworkFunctionResources struct {
	// CPU requirements
	// +kubebuilder:validation:Optional
	CPU string `json:"cpu,omitempty"`

	// Memory requirements
	// +kubebuilder:validation:Optional
	Memory string `json:"memory,omitempty"`

	// Storage requirements
	// +kubebuilder:validation:Optional
	Storage string `json:"storage,omitempty"`

	// Custom resource requirements
	// +kubebuilder:validation:Optional
	Custom map[string]string `json:"custom,omitempty"`
}

// +kubebuilder:object:generate=true
type NetworkIntentStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	
	// Phase indicates the current phase of the NetworkIntent processing
	// +kubebuilder:validation:Optional
	Phase string `json:"phase,omitempty"`
	
	// LastUpdated indicates when the status was last updated
	// +kubebuilder:validation:Optional  
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}

// Package types for Porch integration

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=porch;kpt
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Package struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageSpec   `json:"spec,omitempty"`
	Status PackageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type PackageSpec struct {
	// RepositoryName specifies the repository where the package resides
	// +kubebuilder:validation:Optional
	RepositoryName string `json:"repositoryName,omitempty"`

	// WorkspaceName specifies the workspace for the package
	// +kubebuilder:validation:Optional
	WorkspaceName string `json:"workspaceName,omitempty"`

	// PackageName is the name of the package
	// +kubebuilder:validation:Optional
	PackageName string `json:"packageName,omitempty"`

	// Revision specifies the package revision
	// +kubebuilder:validation:Optional
	Revision string `json:"revision,omitempty"`

	// Lifecycle indicates the package lifecycle phase
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Draft;Proposed;Published
	Lifecycle string `json:"lifecycle,omitempty"`
}

// +kubebuilder:object:generate=true
type PackageStatus struct {
	// Phase indicates the current phase of the package
	// +kubebuilder:validation:Optional
	Phase PackagePhase `json:"phase,omitempty"`

	// Conditions represents the current conditions of the package
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation observed by the controller
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// UpstreamLock contains information about the upstream package lock
	// +kubebuilder:validation:Optional
	UpstreamLock map[string]interface{} `json:"upstreamLock,omitempty"`
}

// PackagePhase defines the possible phases of a package
// +kubebuilder:validation:Enum=Created;Pending;Ready;Failed
type PackagePhase string

const (
	// PackagePhaseCreated indicates the package has been created
	PackagePhaseCreated PackagePhase = "Created"
	// PackagePhasePending indicates the package is pending
	PackagePhasePending PackagePhase = "Pending"
	// PackagePhaseReady indicates the package is ready
	PackagePhaseReady PackagePhase = "Ready"
	// PackagePhaseFailed indicates the package has failed
	PackagePhaseFailed PackagePhase = "Failed"
)

// +kubebuilder:object:root=true
type PackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Package `json:"items"`
}

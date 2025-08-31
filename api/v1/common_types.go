/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Priority represents processing priority levels
// +kubebuilder:validation:Enum=low;normal;medium;high;critical
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// ObjectReference contains enough information to let you inspect or modify the referred object
type ObjectReference struct {
	// Kind of the referent
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// APIVersion of the referent
	// +kubebuilder:validation:Required
	APIVersion string `json:"apiVersion"`

	// Name of the referent
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the referent; only applicable for namespaced resources
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// UID of the referent
	// +optional
	UID string `json:"uid,omitempty"`

	// ResourceVersion of the referent
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty"`

	// FieldPath refers to a field within the object
	// +optional
	FieldPath string `json:"fieldPath,omitempty"`
}

// GetObjectReference creates an ObjectReference from a Kubernetes object
func GetObjectReference(obj metav1.Object, gvk metav1.GroupVersionKind) ObjectReference {
	return ObjectReference{
		Kind:            gvk.Kind,
		APIVersion:      fmt.Sprintf("%s/%s", gvk.Group, gvk.Version),
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		UID:             string(obj.GetUID()),
		ResourceVersion: obj.GetResourceVersion(),
	}
}

// ManagedElement represents a managed network element
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ManagedElement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedElementSpec   `json:"spec,omitempty"`
	Status ManagedElementStatus `json:"status,omitempty"`
}

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	// ID is the unique identifier for the managed element
	ID string `json:"id"`
	// Name is the display name of the managed element
	Name string `json:"name"`
	// Type specifies the type of managed element (e.g., "gNB", "CU", "DU")
	Type string `json:"type"`
	// Credentials for accessing the managed element
	// +optional
	Credentials *ManagedElementCredentials `json:"credentials,omitempty"`
	// Configuration for the managed element
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// Tags for organizing managed elements
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// ManagedElementCredentials defines authentication credentials
type ManagedElementCredentials struct {
	// Username for basic authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password reference for basic authentication
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef,omitempty"`

	// Certificate reference for certificate-based authentication
	// +optional
	CertificateRef *corev1.SecretKeySelector `json:"certificateRef,omitempty"`

	// Private key reference for certificate-based authentication
	// +optional
	PrivateKeyRef *corev1.SecretKeySelector `json:"privateKeyRef,omitempty"`

	// Token reference for token-based authentication
	// +optional
	TokenRef *corev1.SecretKeySelector `json:"tokenRef,omitempty"`

	// SSH key reference for SSH-based authentication
	// +optional
	SSHKeyRef *corev1.SecretKeySelector `json:"sshKeyRef,omitempty"`

	// Client certificate reference for mTLS
	// +optional
	ClientCertRef *corev1.SecretKeySelector `json:"clientCertRef,omitempty"`

	// Client key reference for mTLS
	// +optional
	ClientKeyRef *corev1.SecretKeySelector `json:"clientKeyRef,omitempty"`

	// CA certificate reference
	// +optional
	CACertificateRef *corev1.SecretKeySelector `json:"caCertificateRef,omitempty"`
	// Client certificate reference for authentication
	// +optional
	ClientCertificateRef *corev1.SecretKeySelector `json:"clientCertificateRef,omitempty"`
}

// ResourceConstraints defines resource constraints for network functions
type ResourceConstraints struct {
	// CPU resource constraints
	// +optional
	CPU *ResourceConstraintSpec `json:"cpu,omitempty"`

	// Memory resource constraints
	// +optional
	Memory *ResourceConstraintSpec `json:"memory,omitempty"`

	// Storage resource constraints
	// +optional
	Storage *ResourceConstraintSpec `json:"storage,omitempty"`

	// Network bandwidth constraints
	// +optional
	Bandwidth *ResourceConstraintSpec `json:"bandwidth,omitempty"`

	// Custom resource constraints
	// +optional
	CustomResources map[string]*ResourceConstraintSpec `json:"customResources,omitempty"`

	// Custom additional constraints
	// +optional
	Custom map[string]string `json:"custom,omitempty"`
}

// ResourceConstraintSpec defines the specification for a resource constraint
type ResourceConstraintSpec struct {
	// Minimum required value
	// +optional
	Min *string `json:"min,omitempty"`

	// Maximum allowed value
	// +optional
	Max *string `json:"max,omitempty"`

	// Default value if not specified
	// +optional
	Default *string `json:"default,omitempty"`

	// Preferred value for allocation
	// +optional
	Preferred *string `json:"preferred,omitempty"`

	// Unit of measurement (e.g., "cores", "Gi", "Mbps")
	// +optional
	Unit string `json:"unit,omitempty"`
}

// ProcessedParameters contains structured parameters from intent processing
type ProcessedParameters struct {
	// Raw parameters as key-value pairs
	// +optional
	Raw map[string]string `json:"raw,omitempty"`

	// Structured parameters organized by category
	// +optional
	Structured map[string]string `json:"structured,omitempty"`

	// NetworkFunction specifies the target network function type (e.g., "AMF", "SMF", "UPF")
	// This field identifies the primary network function for the intent processing
	// +kubebuilder:validation:Optional
	// +optional
	NetworkFunction string `json:"networkFunction,omitempty"`

	// Region specifies the deployment region for the network function
	// This field is used for regional deployment and resource allocation decisions
	// +kubebuilder:validation:Optional
	// +optional
	Region string `json:"region,omitempty"`

	// CustomParameters contains additional user-defined parameters
	// These parameters provide flexibility for custom deployment configurations
	// +kubebuilder:validation:Optional
	// +optional
	CustomParameters map[string]string `json:"customParameters,omitempty"`

	// Security-related parameters (now structured instead of map[string]interface{})
	// +optional
	SecurityParameters *SecurityParameters `json:"securityParameters,omitempty"`

	// Processing metadata
	// +optional
	Metadata *ParameterMetadata `json:"metadata,omitempty"`
}

// ParameterMetadata contains metadata about parameter processing
type ParameterMetadata struct {
	// Processing timestamp
	// +optional
	ProcessedAt *metav1.Time `json:"processedAt,omitempty"`

	// Processing version/engine used
	// +optional
	ProcessingVersion string `json:"processingVersion,omitempty"`

	// Source of parameters (llm, user, system, etc.)
	// +optional
	Source string `json:"source,omitempty"`

	// Confidence score for the processing
	// +optional
	ConfidenceScore *float64 `json:"confidenceScore,omitempty"`

	// Confidence level for the processing
	// +optional
	Confidence string `json:"confidence,omitempty"`

	// Additional metadata
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current lifecycle phase
	// +optional
	Phase string `json:"phase,omitempty"`

	// Last successful connection timestamp
	// +optional
	LastConnected *metav1.Time `json:"lastConnected,omitempty"`

	// Connection status
	// +optional
	ConnectionStatus string `json:"connectionStatus,omitempty"`

	// Current configuration version
	// +optional
	ConfigVersion string `json:"configVersion,omitempty"`

	// Observed generation
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ManagedElementList contains a list of ManagedElement
// +kubebuilder:object:root=true
type ManagedElementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedElement `json:"items"`
}

// ConditionType represents the type of condition
type ConditionType string

const (
	// ConditionReady indicates the resource is ready
	ConditionReady ConditionType = "Ready"

	// ConditionProgressing indicates the resource is progressing
	ConditionProgressing ConditionType = "Progressing"

	// ConditionDegraded indicates the resource is degraded
	ConditionDegraded ConditionType = "Degraded"

	// ConditionAvailable indicates the resource is available
	ConditionAvailable ConditionType = "Available"
)

// ConditionReason represents the reason for a condition
type ConditionReason string

const (
	// ReasonReconciling indicates resource is being reconciled
	ReasonReconciling ConditionReason = "Reconciling"

	// ReasonReconciled indicates resource has been reconciled
	ReasonReconciled ConditionReason = "Reconciled"

	// ReasonFailed indicates reconciliation failed
	ReasonFailed ConditionReason = "Failed"

	// ReasonProgressing indicates resource is progressing
	ReasonProgressing ConditionReason = "Progressing"

	// ReasonReady indicates resource is ready
	ReasonReady ConditionReason = "Ready"

	// ReasonDegraded indicates resource is degraded
	ReasonDegraded ConditionReason = "Degraded"
)

// TargetComponent represents a target component for deployment or management
type TargetComponent struct {
	// Name is the component identifier
	Name string `json:"name"`

	// Type is the component type (e.g., "O-RAN-CU", "O-RAN-DU", "5GC-AMF")
	Type string `json:"type"`

	// Version specifies the component version
	// +optional
	Version string `json:"version,omitempty"`

	// Configuration for the component
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Resource requirements
	// +optional
	Resources *ResourceConstraints `json:"resources,omitempty"`

	// Status of the component
	// +optional
	Status string `json:"status,omitempty"`
}

// BackupCompressionConfig defines backup compression settings
type BackupCompressionConfig struct {
	// Enabled indicates whether compression is enabled
	Enabled bool `json:"enabled"`

	// Algorithm specifies the compression algorithm
	// +kubebuilder:validation:Enum=gzip;bzip2;lz4;zstd
	// +optional
	Algorithm string `json:"algorithm,omitempty"`

	// Level specifies the compression level (1-9 for most algorithms)
	// +optional
	Level int `json:"level,omitempty"`

	// ChunkSize specifies the chunk size for compression
	// +optional
	ChunkSize string `json:"chunkSize,omitempty"`
}

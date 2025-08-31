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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	Credentials *ManagedElementCredentials `json:"credentials,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// Phase represents the current status
	Phase string `json:"phase,omitempty"`
	// Conditions represent the current service state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ManagedElementList contains a list of ManagedElement
// +kubebuilder:object:root=true
type ManagedElementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedElement `json:"items"`
}

// ManagedElementCredentials defines authentication credentials for managed elements
type ManagedElementCredentials struct {
	// Username for basic authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password reference for basic authentication
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef,omitempty"`

	// SSH private key reference for SSH-based authentication
	// +optional
	PrivateKeyRef *corev1.SecretKeySelector `json:"privateKeyRef,omitempty"`

	// Client certificate reference for mTLS
	// +optional
	ClientCertificateRef *corev1.SecretKeySelector `json:"clientCertificateRef,omitempty"`

	// Client key reference for mTLS
	// +optional
	ClientKeyRef *corev1.SecretKeySelector `json:"clientKeyRef,omitempty"`

	// CA certificate reference
	// +optional
	CACertificateRef *corev1.SecretKeySelector `json:"caCertificateRef,omitempty"`
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
	Custom map[string]*ResourceConstraintSpec `json:"custom,omitempty"`
}

// ResourceConstraintSpec defines specific resource constraint
type ResourceConstraintSpec struct {
	// Minimum required value
	// +optional
	Min *string `json:"min,omitempty"`

	// Maximum allowed value
	// +optional
	Max *string `json:"max,omitempty"`

	// Preferred value
	// +optional
	Preferred *string `json:"preferred,omitempty"`
}

// ProcessedParameters contains structured parameters from intent processing
type ProcessedParameters struct {
	// Raw parameters as key-value pairs
	// +optional
	Raw map[string]string `json:"raw,omitempty"`

	// Structured parameters organized by category
	// +optional
	Structured map[string]*apiextensionsv1.JSON `json:"structured,omitempty"`

	// Security-related parameters
	// +optional
	SecurityParameters map[string]interface{} `json:"securityParameters,omitempty"`

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

	// Confidence score if applicable
	// +optional
	Confidence *float64 `json:"confidence,omitempty"`
}

// TargetComponent defines a target component for network function deployment
type TargetComponent struct {
	// Component name or identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Component type (e.g., cucp, cuup, du, etc.)
	// +optional
	Type string `json:"type,omitempty"`

	// Version constraints
	// +optional
	Version string `json:"version,omitempty"`

	// Target cluster or location
	// +optional
	Target string `json:"target,omitempty"`

	// Component-specific configuration
	// +optional
	Config map[string]*apiextensionsv1.JSON `json:"config,omitempty"`

	// Resource requirements
	// +optional
	Resources *ResourceConstraints `json:"resources,omitempty"`
}

// BackupCompressionConfig defines backup compression settings
type BackupCompressionConfig struct {
	// Compression algorithm (gzip, lz4, zstd, none)
	// +optional
	// +kubebuilder:validation:Enum=gzip;lz4;zstd;none
	Algorithm string `json:"algorithm,omitempty"`

	// Compression level (algorithm-dependent)
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	Level int `json:"level,omitempty"`

	// Enable parallel compression
	// +optional
	Parallel bool `json:"parallel,omitempty"`
}

// String returns the string representation of Priority
func (p Priority) String() string {
	return string(p)
}

// ToInt converts Priority to integer for comparison
func (p Priority) ToInt() int {
	switch p {
	case PriorityLow:
		return 1
	case PriorityNormal:
		return 2
	case PriorityMedium:
		return 3
	case PriorityHigh:
		return 4
	case PriorityCritical:
		return 5
	default:
		return 2 // default to normal
	}
}

// FromString converts string to Priority
func PriorityFromString(s string) Priority {
	switch s {
	case "low":
		return PriorityLow
	case "normal":
		return PriorityNormal
	case "medium":
		return PriorityMedium
	case "high":
		return PriorityHigh
	case "critical":
		return PriorityCritical
	default:
		return PriorityNormal
	}
}

// FromInt converts integer to Priority
func PriorityFromInt(i int) Priority {
	switch i {
	case 1:
		return PriorityLow
	case 2:
		return PriorityNormal
	case 3:
		return PriorityMedium
	case 4:
		return PriorityHigh
	case 5:
		return PriorityCritical
	default:
		return PriorityNormal
	}
}

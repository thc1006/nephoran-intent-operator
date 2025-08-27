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

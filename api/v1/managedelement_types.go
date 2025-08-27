/*
Copyright 2024.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Credentials defines authentication credentials for managed elements
type Credentials struct {
	// Username reference
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"usernameRef,omitempty"`

	// Password reference
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef,omitempty"`

	// SSH private key reference
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

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	// NetworkElementType specifies the type of network element (e.g., "DU", "CU", "RU")
	NetworkElementType string `json:"networkElementType"`

	// Endpoint is the management endpoint URL for the network element
	Endpoint string `json:"endpoint"`

	// Credentials contains authentication information for the network element
	Credentials *Credentials `json:"credentials,omitempty"`

	// Version specifies the software version of the network element
	Version string `json:"version,omitempty"`

	// Configuration holds network element specific configuration
	Configuration map[string]string `json:"configuration,omitempty"`

	// A1Policy contains A1 policy configuration for the managed element
	// +optional
	A1Policy runtime.RawExtension `json:"a1Policy,omitempty"`

	// E2Configuration contains E2 interface configuration for the managed element
	// +optional
	E2Configuration map[string]string `json:"e2Configuration,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// Phase represents the current lifecycle phase of the managed element
	Phase string `json:"phase,omitempty"`

	// ConnectionStatus indicates whether the element is reachable
	ConnectionStatus string `json:"connectionStatus,omitempty"`

	// LastSeen timestamp of the last successful communication
	LastSeen *metav1.Time `json:"lastSeen,omitempty"`

	// OperationalStatus indicates if the element is operational
	OperationalStatus string `json:"operationalStatus,omitempty"`

	// ConfigurationStatus indicates if configuration is applied successfully
	ConfigurationStatus string `json:"configurationStatus,omitempty"`

	// Message provides human-readable status information
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations of the element's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ManagedElement is the Schema for the managedelements API
type ManagedElement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedElementSpec   `json:"spec,omitempty"`
	Status ManagedElementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedElementList contains a list of ManagedElement
type ManagedElementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedElement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedElement{}, &ManagedElementList{})
}

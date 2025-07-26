package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	DeploymentName string `json:"deploymentName"`
	O1Config       string `json:"o1Config,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	A1Policy runtime.RawExtension `json:"a1Policy,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ManagedElement is the Schema for the managedelements API
type ManagedElement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedElementSpec   `json:"spec,omitempty"`
	Status ManagedElementStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedElementList contains a list of ManagedElement
type ManagedElementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedElement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedElement{}, &ManagedElementList{})
}

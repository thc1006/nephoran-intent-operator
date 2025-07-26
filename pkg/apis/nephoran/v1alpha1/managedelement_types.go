package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	DeploymentName string               `json:"deploymentName"`
	O1Config       string               `json:"o1Config,omitempty"`
	A1Policy       runtime.RawExtension `json:"a1Policy,omitempty"`
}

// ... (status and other structs remain the same)

func init() {
	SchemeBuilder.Register(&ManagedElement{}, &ManagedElementList{})
}
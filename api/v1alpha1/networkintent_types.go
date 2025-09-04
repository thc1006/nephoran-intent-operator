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
}

// +kubebuilder:object:generate=true
type NetworkIntentStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}

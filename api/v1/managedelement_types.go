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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ManagedElementSpec defines the desired state of ManagedElement
type ManagedElementSpec struct {
	DeploymentName string `json:"deploymentName"`
	O1Config       string `json:"o1Config,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	A1Policy runtime.RawExtension `json:"a1Policy,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	E2Configuration runtime.RawExtension `json:"e2Configuration,omitempty"`
}

// ManagedElementStatus defines the observed state of ManagedElement
type ManagedElementStatus struct {
	// +optional
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

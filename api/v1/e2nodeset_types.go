/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUTHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// E2NodeSetSpec defines the desired state of E2NodeSet
type E2NodeSetSpec struct {
	// Replicas is the number of simulated E2 Nodes to run.
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`
}

// E2NodeSetStatus defines the observed state of E2NodeSet
type E2NodeSetStatus struct {
	// ReadyReplicas is the number of E2 Nodes that are ready.
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=e2nodesets,scope=Namespaced

// E2NodeSet is the Schema for the e2nodesets API
type E2NodeSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   E2NodeSetSpec   `json:"spec,omitempty"`
	Status E2NodeSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// E2NodeSetList contains a list of E2NodeSet
type E2NodeSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []E2NodeSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&E2NodeSet{}, &E2NodeSetList{})
}

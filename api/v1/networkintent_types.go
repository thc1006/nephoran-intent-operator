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

// NetworkIntentSpec defines the desired state of NetworkIntent
type NetworkIntentSpec struct {
	// The original natural language intent from the user.
	Intent string `json:"intent"`

	// Structured parameters translated from the intent by the LLM.
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters runtime.RawExtension `json:"parameters,omitempty"`
}

// NetworkIntentStatus defines the observed state of NetworkIntent
type NetworkIntentStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	
	// Phase represents the current phase of the NetworkIntent processing
	// +optional
	Phase string `json:"phase,omitempty"`
	
	// ProcessingStartTime indicates when the intent processing started
	// +optional
	ProcessingStartTime *metav1.Time `json:"processingStartTime,omitempty"`
	
	// ProcessingCompletionTime indicates when the intent processing completed
	// +optional
	ProcessingCompletionTime *metav1.Time `json:"processingCompletionTime,omitempty"`
	
	// DeploymentStartTime indicates when the GitOps deployment started
	// +optional
	DeploymentStartTime *metav1.Time `json:"deploymentStartTime,omitempty"`
	
	// DeploymentCompletionTime indicates when the GitOps deployment completed
	// +optional
	DeploymentCompletionTime *metav1.Time `json:"deploymentCompletionTime,omitempty"`
	
	// GitCommitHash represents the commit hash of the deployed configuration
	// +optional
	GitCommitHash string `json:"gitCommitHash,omitempty"`
	
	// LastRetryTime indicates the last time a retry was attempted
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`
	
	// ObservedGeneration reflects the generation of the most recently observed NetworkIntent
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NetworkIntent is the Schema for the networkintents API
type NetworkIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkIntentSpec   `json:"spec,omitempty"`
	Status NetworkIntentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NetworkIntentList contains a list of NetworkIntent
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}

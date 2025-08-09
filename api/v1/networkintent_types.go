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
)

// NetworkIntentSpec defines the desired state of NetworkIntent
type NetworkIntentSpec struct {
	// Intent is the natural language intent from the user describing the desired network configuration.
	// The intent must be between 1 and 2000 characters and cannot contain potentially malicious content
	// such as script tags, SQL injection patterns, or command injection attempts.
	// Only printable ASCII and common Unicode characters are allowed.
	// Examples:
	//   - "Deploy a high-availability AMF instance for production with auto-scaling"
	//   - "Create a network slice for URLLC with 1ms latency requirements"  
	//   - "Configure QoS policies for enhanced mobile broadband services"
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2000
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\s\-_.,;:()\[\]{}'"@#$%&*+=?!/\\]*$`
	Intent string `json:"intent"`
}

// NetworkIntentStatus defines the observed state of NetworkIntent
type NetworkIntentStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed NetworkIntent
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the NetworkIntent processing
	Phase string `json:"phase,omitempty"`

	// LastMessage contains the last status message
	LastMessage string `json:"lastMessage,omitempty"`

	// LastUpdateTime indicates when the status was last updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=networkintents,scope=Namespaced,shortName=ni
//+kubebuilder:webhook:path=/validate-nephoran-io-v1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=nephoran.io,resources=networkintents,verbs=create;update,versions=v1,name=vnetworkintent.kb.io,admissionReviewVersions=v1

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// +kubebuilder:subresource:status

// +kubebuilder:resource:scope=Namespaced,shortName=ni

// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.target`

// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`

// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// +kubebuilder:webhook:path=/mutate-intent-nephoran-com-v1alpha1-networkintent,mutating=true,failurePolicy=fail,sideEffects=None,groups=intent.nephoran.com,resources=networkintents,verbs=create;update,versions=v1alpha1,name=mnetworkintent.kb.io,admissionReviewVersions=v1

// +kubebuilder:webhook:path=/validate-intent-nephoran-com-v1alpha1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=intent.nephoran.com,resources=networkintents,verbs=create;update,versions=v1alpha1,name=vnetworkintent.kb.io,admissionReviewVersions=v1

// NetworkIntent is the Schema for the networkintents API.
// It represents a high-level intent for network configuration and management.
type NetworkIntent struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkIntentSpec `json:"spec,omitempty"`

	Status NetworkIntentStatus `json:"status,omitempty"`
}

// NetworkIntentSpec defines the desired state of NetworkIntent.
// It specifies the intent type, target, and configuration parameters.
type NetworkIntentSpec struct {
	// +kubebuilder:validation:Enum=scaling

	IntentType string `json:"intentType"`

	// +kubebuilder:validation:MinLength=1

	Target string `json:"target"`

	// +kubebuilder:validation:MinLength=1

	Namespace string `json:"namespace"`

	// +kubebuilder:validation:Minimum=0

	Replicas int32 `json:"replicas"`

	// +kubebuilder:default="user"

	Source string `json:"source,omitempty"`
}

// +k8s:deepcopy-gen=true

// NetworkIntentStatus defines the observed state of NetworkIntent.
// It contains runtime status information and conditions.
type NetworkIntentStatus struct {
	// +optional

	ObservedReplicas *int32 `json:"observedReplicas,omitempty"`

	// +optional

	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LLMResponse contains the raw LLM response data.

	// +optional

	LLMResponse interface{} `json:"llmResponse,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkIntentList contains a list of NetworkIntent resources.
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkIntent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// NetworkIntent represents a high-level network scaling and configuration intent
type NetworkIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkIntentSpec   `json:"spec,omitempty"`
	Status NetworkIntentStatus `json:"status,omitempty"`
}

// NetworkIntentSpec defines the desired state of a network intent
type NetworkIntentSpec struct {
	// Source of the intent (e.g., "user", "automation")
	Source string `json:"source,omitempty"`

	// IntentType specifies the type of intent (e.g., "scaling")
	IntentType string `json:"intentType,omitempty"`

	// Target specifies the target component or resource
	Target string `json:"target,omitempty"`

	// Namespace specifies the target namespace
	Namespace string `json:"namespace,omitempty"`

	// Replicas specifies the desired number of replicas
	Replicas int32 `json:"replicas,omitempty"`

	// ScalingParameters define how network functions should be scaled
	ScalingParameters ScalingConfig `json:"scalingParameters,omitempty"`

	// NetworkParameters define network-level configurations
	NetworkParameters NetworkConfig `json:"networkParameters,omitempty"`
}

// ScalingConfig defines scaling behavior for network functions
type ScalingConfig struct {
	// Replicas defines the desired number of replicas for network functions
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// AutoscalingPolicy defines how automatic scaling should occur
	// +optional
	AutoscalingPolicy AutoscalingPolicy `json:"autoscalingPolicy,omitempty"`
}

// AutoscalingPolicy defines rules for automatic scaling
type AutoscalingPolicy struct {
	// MinReplicas is the lower limit of replicas
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper limit of replicas
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// MetricThresholds define scaling triggers
	// +optional
	MetricThresholds []MetricThreshold `json:"metricThresholds,omitempty"`
}

// MetricThreshold defines a condition for scaling
type MetricThreshold struct {
	// Type of metric (CPU, Memory, Custom)
	Type string `json:"type"`

	// Value at which scaling occurs
	Value int64 `json:"value"`
}

// NetworkConfig defines network-level configurations
type NetworkConfig struct {
	// NetworkSliceID defines the specific network slice
	// +optional
	NetworkSliceID string `json:"networkSliceId,omitempty"`

	// QoSProfile defines Quality of Service settings
	// +optional
	QoSProfile QoSProfile `json:"qosProfile,omitempty"`
}

// QoSProfile defines Quality of Service parameters
type QoSProfile struct {
	// Priority of the network slice
	// +optional
	Priority int32 `json:"priority,omitempty"`

	// MaximumDataRate defines the maximum data transmission rate
	// +optional
	MaximumDataRate string `json:"maximumDataRate,omitempty"`
}

// NetworkIntentStatus defines the observed state of a network intent
type NetworkIntentStatus struct {
	// Phase indicates the current phase of the intent
	Phase string `json:"phase,omitempty"`

	// Message contains human-readable information about the status
	Message string `json:"message,omitempty"`

	// ObservedReplicas reflects the current number of replicas
	// +optional
	ObservedReplicas *int32 `json:"observedReplicas,omitempty"`

	// ReadyReplicas represents the number of ready instances
	// +optional
	ReadyReplicas *int32 `json:"readyReplicas,omitempty"`

	// Conditions represent the current conditions
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LLMResponse contains the LLM processing response
	// +optional
	LLMResponse *apiextensionsv1.JSON `json:"llmResponse,omitempty"`
}

// +kubebuilder:object:root=true
// NetworkIntentList contains a list of NetworkIntent
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}
<<<<<<< HEAD
=======

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

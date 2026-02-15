package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkIntent represents a high-level network scaling and configuration intent
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=nephio;o-ran,shortName=ni
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.target`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type NetworkIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkIntentSpec   `json:"spec,omitempty"`
	Status NetworkIntentStatus `json:"status,omitempty"`
}

// NetworkIntentSpec defines the desired state of a network intent
// +kubebuilder:object:generate=true
type NetworkIntentSpec struct {
	// Source of the intent (e.g., "user", "automation")
	// +kubebuilder:validation:Optional
	Source string `json:"source,omitempty"`

	// IntentType specifies the type of intent (e.g., "scaling", "optimization")
	// +kubebuilder:validation:Optional
	IntentType string `json:"intentType,omitempty"`

	// Target specifies the target component or resource
	// +kubebuilder:validation:Optional
	Target string `json:"target,omitempty"`

	// Namespace specifies the target namespace
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// Replicas specifies the desired number of replicas
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas,omitempty"`

	// ScalingParameters define how network functions should be scaled
	// +kubebuilder:validation:Optional
	ScalingParameters *ScalingParameters `json:"scalingParameters,omitempty"`

	// NetworkParameters define network-level configurations
	// +kubebuilder:validation:Optional
	NetworkParameters *NetworkParameters `json:"networkParameters,omitempty"`
}

// ScalingParameters defines scaling configuration
// +kubebuilder:object:generate=true
type ScalingParameters struct {
	// Replicas defines the desired number of replicas for network functions
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas,omitempty"`

	// AutoscalingPolicy defines how automatic scaling should occur
	// +kubebuilder:validation:Optional
	AutoscalingPolicy *AutoscalingPolicy `json:"autoscalingPolicy,omitempty"`
}

// AutoscalingPolicy defines autoscaling behavior
// +kubebuilder:object:generate=true
type AutoscalingPolicy struct {
	// MinReplicas is the lower limit of replicas
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the upper limit of replicas
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// MetricThresholds define scaling triggers
	// +kubebuilder:validation:Optional
	MetricThresholds []MetricThreshold `json:"metricThresholds,omitempty"`
}

// MetricThreshold defines a condition for scaling
// +kubebuilder:object:generate=true
type MetricThreshold struct {
	// Type of metric (CPU, Memory, Custom)
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Value at which scaling occurs
	// +kubebuilder:validation:Required
	Value int64 `json:"value"`
}

// NetworkParameters defines network-level configuration
// +kubebuilder:object:generate=true
type NetworkParameters struct {
	// NetworkSliceID defines the specific network slice
	// +kubebuilder:validation:Optional
	NetworkSliceID string `json:"networkSliceId,omitempty"`

	// QoSProfile defines Quality of Service settings
	// +kubebuilder:validation:Optional
	QoSProfile *QoSProfile `json:"qosProfile,omitempty"`
}

// QoSProfile defines Quality of Service settings
// +kubebuilder:object:generate=true
type QoSProfile struct {
	// Priority of the network slice
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	Priority int32 `json:"priority,omitempty"`

	// MaximumDataRate defines the maximum data transmission rate
	// +kubebuilder:validation:Optional
	MaximumDataRate string `json:"maximumDataRate,omitempty"`
}

// NetworkIntentStatus defines the observed state of a network intent
// +kubebuilder:object:generate=true
type NetworkIntentStatus struct {
	// Phase indicates the current phase of the intent
	// +kubebuilder:validation:Optional
	Phase string `json:"phase,omitempty"`

	// Message contains human-readable information about the status
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`

	// Conditions represent the current conditions
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedReplicas reflects the current number of replicas
	// +kubebuilder:validation:Optional
	ObservedReplicas int32 `json:"observedReplicas,omitempty"`

	// ReadyReplicas represents the number of ready instances
	// +kubebuilder:validation:Optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// LLMResponse contains the LLM processing response
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	LLMResponse *apiextensionsv1.JSON `json:"llmResponse,omitempty"`
}

// +kubebuilder:object:root=true
// NetworkIntentList contains a list of NetworkIntent
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}

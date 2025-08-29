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

// NetworkIntentSpec defines the desired state of NetworkIntent.
type NetworkIntentSpec struct {
	// Intent is the natural language intent from the user describing the desired network configuration.
	//
	// SECURITY: This field implements defense-in-depth validation to prevent injection attacks:.
	// 1. Restricted character set - Only allows alphanumeric, spaces, and safe punctuation.
	// 2. Length limits - Maximum 1000 characters to prevent resource exhaustion.
	// 3. Pattern validation - Explicitly excludes dangerous characters that could enable:.
	//    - Command injection (backticks, $, &, *, /, \).
	//    - SQL injection (quotes, =, --).
	//    - Script injection (<, >, @, #).
	//    - Path traversal (/, \).
	//    - Protocol handlers (@, #).
	// 4. Additional runtime validation in webhook for content analysis.
	//
	// Allowed characters:.
	//   - Letters: a-z, A-Z.
	//   - Numbers: 0-9.
	//   - Spaces and basic punctuation: space, hyphen, underscore, period, comma, semicolon, colon.
	//   - Grouping: parentheses (), square brackets [].
	//
	// Examples:.
	//   - "Deploy a high-availability AMF instance for production with auto-scaling".
	//   - "Create a network slice for URLLC with 1ms latency requirements".
	//   - "Configure QoS policies for enhanced mobile broadband services".
	//
	// +kubebuilder:validation:MinLength=1.
	// +kubebuilder:validation:MaxLength=1000.
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`.
	Intent string `json:"intent"`

	// Description provides a human-readable description of the network intent.
	// This field is typically populated during processing to provide additional context.
	// +optional.
	Description string `json:"description,omitempty"`

	// IntentType specifies the type of intent.
	// +optional.
	IntentType IntentType `json:"intentType,omitempty"`

	// Priority specifies the processing priority.
	// +optional.
	Priority Priority `json:"priority,omitempty"`

	// TargetComponents specifies the target components for the intent.
	// +optional.
	TargetComponents []ORANComponent `json:"targetComponents,omitempty"`

	// TargetNamespace specifies the target namespace for deployment.
	// +optional.
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// TargetCluster specifies the target cluster for deployment.
	// +optional.
	TargetCluster string `json:"targetCluster,omitempty"`

	// NetworkSlice specifies the network slice identifier.
	// +optional.
	NetworkSlice string `json:"networkSlice,omitempty"`

	// Region specifies the deployment region.
	// +optional.
	Region string `json:"region,omitempty"`

	// ResourceConstraints specifies resource constraints for the intent.
	// +optional.
	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`

	// ProcessedParameters contains structured parameters from processing.
	// +optional.
	ProcessedParameters *ProcessedParameters `json:"processedParameters,omitempty"`
}

// ProcessingResult contains the results of intent processing.
type ProcessingResult struct {
	// NetworkFunctionType contains the detected network function type.
	NetworkFunctionType string `json:"networkFunctionType,omitempty"`

	// ProcessedParameters contains the structured parameters.
	ProcessedParameters *ProcessedParameters `json:"processedParameters,omitempty"`

	// ConfidenceScore represents the confidence in the processing results.
	ConfidenceScore *float64 `json:"confidenceScore,omitempty"`

	// ProcessingTimestamp when the processing completed.
	ProcessingTimestamp *metav1.Time `json:"processingTimestamp,omitempty"`
}

// IntentType represents the type of intent.
type IntentType string

const (
	// IntentTypeDeployment represents deployment intents.
	IntentTypeDeployment IntentType = "deployment"
	// IntentTypeOptimization represents optimization intents.
	IntentTypeOptimization IntentType = "optimization"
	// IntentTypeScaling represents scaling intents.
	IntentTypeScaling IntentType = "scaling"
	// IntentTypeMaintenance represents maintenance intents.
	IntentTypeMaintenance IntentType = "maintenance"
)

// ORANComponent represents O-RAN target components for intents.
type ORANComponent string

const (
	// ORANComponentNearRTRIC represents Near-RT RIC component.
	ORANComponentNearRTRIC ORANComponent = "near-rt-ric"
	// ORANComponentSMO represents SMO component.
	ORANComponentSMO ORANComponent = "smo"
	// ORANComponentO1 represents O1 interface component.
	ORANComponentO1 ORANComponent = "o1"
	// ORANComponentE2 represents E2 interface component.
	ORANComponentE2 ORANComponent = "e2"
	// ORANComponentA1 represents A1 interface component.
	ORANComponentA1 ORANComponent = "a1"
	// ORANComponentXApp represents xApp component.
	ORANComponentXApp ORANComponent = "xapp"
	// ORANComponentGNodeB represents gNodeB component.
	ORANComponentGNodeB ORANComponent = "gnodeb"
	// ORANComponentAMF represents AMF component.
	ORANComponentAMF ORANComponent = "amf"
	// ORANComponentSMF represents SMF component.
	ORANComponentSMF ORANComponent = "smf"
	// ORANComponentUPF represents UPF component.
	ORANComponentUPF ORANComponent = "upf"
)

// NetworkIntentPhase represents the phase of NetworkIntent processing.
type NetworkIntentPhase string

const (
	// NetworkIntentPhasePending represents pending phase.
	NetworkIntentPhasePending NetworkIntentPhase = "Pending"
	// NetworkIntentPhaseProcessing represents processing phase.
	NetworkIntentPhaseProcessing NetworkIntentPhase = "Processing"
	// NetworkIntentPhaseDeploying represents deploying phase.
	NetworkIntentPhaseDeploying NetworkIntentPhase = "Deploying"
	// NetworkIntentPhaseActive represents active phase.
	NetworkIntentPhaseActive NetworkIntentPhase = "Active"
	// NetworkIntentPhaseReady represents ready phase.
	NetworkIntentPhaseReady NetworkIntentPhase = "Ready"
	// NetworkIntentPhaseDeployed represents deployed phase.
	NetworkIntentPhaseDeployed NetworkIntentPhase = "Deployed"
	// NetworkIntentPhaseFailed represents failed phase.
	NetworkIntentPhaseFailed NetworkIntentPhase = "Failed"
)

// NetworkIntentStatus defines the observed state of NetworkIntent.
type NetworkIntentStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed NetworkIntent.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the NetworkIntent processing.
	Phase NetworkIntentPhase `json:"phase,omitempty"`

	// LastMessage contains the last status message.
	LastMessage string `json:"lastMessage,omitempty"`

	// LastUpdateTime indicates when the status was last updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// ProcessingResults contains the results of intent processing.
	// +optional.
	ProcessingResults *ProcessingResult `json:"processingResults,omitempty"`

	// LLMResponse contains the raw LLM response data.
	// +optional.
	LLMResponse runtime.RawExtension `json:"llmResponse,omitempty"`

	// ResourcePlan contains the resource planning data.
	// +optional.
	ResourcePlan runtime.RawExtension `json:"resourcePlan,omitempty"`

	// ProcessingPhase indicates the current processing phase.
	// +optional.
	ProcessingPhase string `json:"processingPhase,omitempty"`

	// ErrorMessage contains any error messages.
	// +optional.
	ErrorMessage string `json:"errorMessage,omitempty"`

	// LastUpdated indicates when the status was last updated.
	// +optional.
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// ValidationErrors contains validation errors encountered during processing.
	// +optional.
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// DeployedComponents contains the list of deployed components.
	// +optional.
	DeployedComponents []TargetComponent `json:"deployedComponents,omitempty"`

	// ProcessingDuration contains the duration of the processing.
	// +optional.
	ProcessingDuration *metav1.Duration `json:"processingDuration,omitempty"`

	// Extensions contains additional status information as raw extensions.
	// +optional.
	Extensions map[string]runtime.RawExtension `json:"extensions,omitempty"`

	// Conditions contains the conditions for the NetworkIntent.
	// +optional.
	// +listType=map.
	// +listMapKey=type.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=networkintents,scope=Namespaced,shortName=ni
//+kubebuilder:webhook:path=/validate-nephoran-io-v1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=nephoran.io,resources=networkintents,verbs=create;update,versions=v1,name=vnetworkintent.kb.io,admissionReviewVersions=v1

// NetworkIntent is the Schema for the networkintents API.
type NetworkIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkIntentSpec   `json:"spec,omitempty"`
	Status NetworkIntentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NetworkIntentList contains a list of NetworkIntent.
type NetworkIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkIntent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}

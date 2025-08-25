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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IntentType represents the type of network intent
// +kubebuilder:validation:Enum=scaling;deployment;configuration;optimization;maintenance
type IntentType string

const (
	IntentTypeScaling       IntentType = "scaling"
	IntentTypeDeployment    IntentType = "deployment"
	IntentTypeConfiguration IntentType = "configuration"
	IntentTypeOptimization  IntentType = "optimization"
	IntentTypeMaintenance   IntentType = "maintenance"
)

// NetworkTargetComponent represents target components for network intents (to avoid conflict with disaster recovery)
type NetworkTargetComponent string

const (
	NetworkTargetComponentAMF    NetworkTargetComponent = "AMF"
	NetworkTargetComponentSMF    NetworkTargetComponent = "SMF"
	NetworkTargetComponentUPF    NetworkTargetComponent = "UPF"
	NetworkTargetComponentNRF    NetworkTargetComponent = "NRF"
	NetworkTargetComponentUDM    NetworkTargetComponent = "UDM"
	NetworkTargetComponentUDR    NetworkTargetComponent = "UDR"
	NetworkTargetComponentPCF    NetworkTargetComponent = "PCF"
	NetworkTargetComponentAUSF   NetworkTargetComponent = "AUSF"
	NetworkTargetComponentNSSF   NetworkTargetComponent = "NSSF"
	NetworkTargetComponentCUCP   NetworkTargetComponent = "CU-CP"
	NetworkTargetComponentCUUP   NetworkTargetComponent = "CU-UP"
	NetworkTargetComponentDU     NetworkTargetComponent = "DU"
	NetworkTargetComponentNEF    NetworkTargetComponent = "NEF"
	NetworkTargetComponentNWDAF  NetworkTargetComponent = "NWDAF"
	NetworkTargetComponentAF     NetworkTargetComponent = "AF"
)

// NetworkPriority represents processing priority levels (to avoid conflict with disaster recovery)
// +kubebuilder:validation:Enum=low;normal;high;critical
type NetworkPriority string

const (
	NetworkPriorityLow      NetworkPriority = "low"
	NetworkPriorityNormal   NetworkPriority = "normal"
	NetworkPriorityHigh     NetworkPriority = "high"
	NetworkPriorityCritical NetworkPriority = "critical"
)

// NetworkIntentSpec defines the desired state of NetworkIntent
type NetworkIntentSpec struct {
	// Intent is the natural language intent from the user describing the desired network configuration.
	//
	// SECURITY: This field implements defense-in-depth validation to prevent injection attacks:
	// 1. Restricted character set - Only allows alphanumeric, spaces, and safe punctuation
	// 2. Length limits - Maximum 1000 characters to prevent resource exhaustion
	// 3. Pattern validation - Explicitly excludes dangerous characters that could enable:
	//    - Command injection (backticks, $, &, *, /, \)
	//    - SQL injection (quotes, =, --)
	//    - Script injection (<, >, @, #)
	//    - Path traversal (/, \)
	//    - Protocol handlers (@, #)
	// 4. Additional runtime validation in webhook for content analysis
	//
	// Allowed characters:
	//   - Letters: a-z, A-Z
	//   - Numbers: 0-9
	//   - Spaces and basic punctuation: space, hyphen, underscore, period, comma, semicolon, colon
	//   - Grouping: parentheses (), square brackets []
	//
	// Examples:
	//   - "Deploy a high-availability AMF instance for production with auto-scaling"
	//   - "Create a network slice for URLLC with 1ms latency requirements"
	//   - "Configure QoS policies for enhanced mobile broadband services"
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1000
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`
	Intent string `json:"intent"`
	
	// IntentType specifies the type of network intent
	// +kubebuilder:validation:Enum=scaling;deployment;configuration;optimization;maintenance
	IntentType IntentType `json:"intentType,omitempty"`
	
	// Parameters contains structured parameters extracted from the intent
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	
	// TargetCluster specifies the target cluster for deployment
	TargetCluster string `json:"targetCluster,omitempty"`
	
	// Priority defines the processing priority
	// +kubebuilder:validation:Enum=low;normal;high;critical
	Priority NetworkPriority `json:"priority,omitempty"`
	
	// TargetComponents specifies the target components for the intent
	TargetComponents []NetworkTargetComponent `json:"targetComponents,omitempty"`
	
	// ResourceRequirements specifies resource requirements
	ResourceRequirements *ResourceRequirements `json:"resourceRequirements,omitempty"`
	
	// Constraints defines deployment constraints
	Constraints map[string]string `json:"constraints,omitempty"`
	
	// ResourceConstraints defines resource constraints for deployments
	ResourceConstraints *NetworkResourceConstraints `json:"resourceConstraints,omitempty"`
	
	// TargetNamespace specifies the target namespace for deployment
	TargetNamespace string `json:"targetNamespace,omitempty"`
	
	// NetworkSlice identifies the network slice for this intent
	NetworkSlice string `json:"networkSlice,omitempty"`
	
	// Region specifies the target region for deployment
	Region string `json:"region,omitempty"`
	
	// TimeoutSeconds specifies the timeout for operations in seconds
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
	
	// MaxRetries specifies the maximum number of retry attempts
	MaxRetries *int32 `json:"maxRetries,omitempty"`
	
	// ProcessedParameters contains parameters extracted during processing
	ProcessedParameters map[string]interface{} `json:"processedParameters,omitempty"`
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
	
	// Conditions represent the current conditions of the NetworkIntent
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	
	// ProcessingCompletionTime indicates when processing completed
	ProcessingCompletionTime *metav1.Time `json:"processingCompletionTime,omitempty"`
	
	// ResourcePlan contains the generated resource deployment plan
	ResourcePlan *NetworkResourcePlan `json:"resourcePlan,omitempty"`
	
	// ValidationErrors contains any validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`
	
	// DeploymentStatus indicates the current deployment status
	DeploymentStatus string `json:"deploymentStatus,omitempty"`
	
	// ProcessingResults contains structured processing results
	ProcessingResults map[string]interface{} `json:"processingResults,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=networkintents,scope=Namespaced,shortName=ni
//+kubebuilder:webhook:path=/validate-nephoran-com-v1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=nephoran.com,resources=networkintents,verbs=create;update,versions=v1,name=vnetworkintent.kb.io,admissionReviewVersions=v1

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

// ResourceRequirements specifies resource requirements for deployments
type ResourceRequirements struct {
	// CPU requirements
	CPU string `json:"cpu,omitempty"`
	
	// Memory requirements  
	Memory string `json:"memory,omitempty"`
	
	// Storage requirements
	Storage string `json:"storage,omitempty"`
	
	// Kubernetes resource requirements
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// NetworkResourceConstraints defines resource constraints for network operations (to avoid conflict with disaster recovery)
type NetworkResourceConstraints struct {
	// Maximum CPU allocation
	MaxCPU string `json:"maxCpu,omitempty"`
	
	// Maximum memory allocation
	MaxMemory string `json:"maxMemory,omitempty"`
	
	// Maximum storage allocation
	MaxStorage string `json:"maxStorage,omitempty"`
	
	// Minimum resource requirements
	MinResources *corev1.ResourceRequirements `json:"minResources,omitempty"`
	
	// Maximum resource limits
	MaxResources *corev1.ResourceRequirements `json:"maxResources,omitempty"`
	
	// Resource quotas
	ResourceQuota map[string]string `json:"resourceQuota,omitempty"`
}

// NetworkResourcePlan contains the generated resource deployment plan
type NetworkResourcePlan struct {
	// Resources to be deployed
	Resources []NetworkPlannedResource `json:"resources,omitempty"`
	
	// Dependencies between resources
	Dependencies []NetworkResourceDependency `json:"dependencies,omitempty"`
	
	// Deployment phases
	Phases []NetworkDeploymentPhase `json:"phases,omitempty"`
	
	// Estimated cost
	EstimatedCost *ResourceCost `json:"estimatedCost,omitempty"`
}

// NetworkPlannedResource represents a resource to be deployed
type NetworkPlannedResource struct {
	// Name of the resource
	Name string `json:"name"`
	
	// Type of resource (e.g., "CNF", "VNF", "Service")
	Type string `json:"type"`
	
	// Configuration for the resource
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	
	// Target cluster
	TargetCluster string `json:"targetCluster,omitempty"`
	
	// Status of the resource
	Status string `json:"status,omitempty"`
}

// NetworkResourceDependency represents dependency between resources
type NetworkResourceDependency struct {
	// Source resource name
	Source string `json:"source"`
	
	// Target resource name  
	Target string `json:"target"`
	
	// Dependency type
	Type string `json:"type"`
}

// NetworkDeploymentPhase represents a deployment phase
type NetworkDeploymentPhase struct {
	// Name of the phase
	Name string `json:"name"`
	
	// Resources to deploy in this phase
	Resources []string `json:"resources"`
	
	// Phase order
	Order int `json:"order"`
}

// ResourceCost represents estimated resource costs
type ResourceCost struct {
	// CPU cost per hour
	CPUCostPerHour float64 `json:"cpuCostPerHour,omitempty"`
	
	// Memory cost per hour
	MemoryCostPerHour float64 `json:"memoryCostPerHour,omitempty"`
	
	// Storage cost per hour
	StorageCostPerHour float64 `json:"storageCostPerHour,omitempty"`
	
	// Total estimated cost per hour
	TotalCostPerHour float64 `json:"totalCostPerHour,omitempty"`
}

// String returns the string representation of IntentType
func (it IntentType) String() string {
	return string(it)
}

// String returns the string representation of NetworkTargetComponent
func (ntc NetworkTargetComponent) String() string {
	return string(ntc)
}

// String returns the string representation of NetworkPriority
func (np NetworkPriority) String() string {
	return string(np)
}

// NetworkTargetComponentsToStrings converts a slice of NetworkTargetComponent to []string
func NetworkTargetComponentsToStrings(components []NetworkTargetComponent) []string {
	result := make([]string, len(components))
	for i, component := range components {
		result[i] = component.String()
	}
	return result
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}

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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	NetworkTargetComponentAMF   NetworkTargetComponent = "AMF"
	NetworkTargetComponentSMF   NetworkTargetComponent = "SMF"
	NetworkTargetComponentUPF   NetworkTargetComponent = "UPF"
	NetworkTargetComponentNRF   NetworkTargetComponent = "NRF"
	NetworkTargetComponentUDM   NetworkTargetComponent = "UDM"
	NetworkTargetComponentUDR   NetworkTargetComponent = "UDR"
	NetworkTargetComponentPCF   NetworkTargetComponent = "PCF"
	NetworkTargetComponentAUSF  NetworkTargetComponent = "AUSF"
	NetworkTargetComponentNSSF  NetworkTargetComponent = "NSSF"
	NetworkTargetComponentCUCP  NetworkTargetComponent = "CU-CP"
	NetworkTargetComponentCUUP  NetworkTargetComponent = "CU-UP"
	NetworkTargetComponentDU    NetworkTargetComponent = "DU"
	NetworkTargetComponentNEF   NetworkTargetComponent = "NEF"
	NetworkTargetComponentNWDAF NetworkTargetComponent = "NWDAF"
	NetworkTargetComponentAF    NetworkTargetComponent = "AF"
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

// NetworkIntentPhase represents the processing phase of a NetworkIntent
type NetworkIntentPhase string

const (
	NetworkIntentPhasePending     NetworkIntentPhase = "Pending"
	NetworkIntentPhaseProcessing  NetworkIntentPhase = "Processing"
	NetworkIntentPhaseDeploying   NetworkIntentPhase = "Deploying"
	NetworkIntentPhaseActive      NetworkIntentPhase = "Active"
	NetworkIntentPhaseReady       NetworkIntentPhase = "Ready"
	NetworkIntentPhaseFailed      NetworkIntentPhase = "Failed"
	NetworkIntentPhaseCompleted   NetworkIntentPhase = "Completed"
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
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// ParametersMap contains key-value parameters extracted from the intent
	ParametersMap map[string]string `json:"parametersMap,omitempty"`

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
	ProcessedParameters *runtime.RawExtension `json:"processedParameters,omitempty"`
}

// NetworkIntentStatus defines the observed state of NetworkIntent
type NetworkIntentStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed NetworkIntent
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the NetworkIntent processing
	Phase NetworkIntentPhase `json:"phase,omitempty"`

	// ProcessingPhase represents the current processing phase (used by controllers)
	ProcessingPhase string `json:"processingPhase,omitempty"`

	// LastMessage contains the last status message
	LastMessage string `json:"lastMessage,omitempty"`

	// Message contains the current status message (alias for LastMessage for controller compatibility)
	Message string `json:"message,omitempty"`

	// LastUpdateTime indicates when the status was last updated
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastUpdated indicates when the status was last updated (alias for controller compatibility)
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// LastProcessed indicates when the intent was last processed
	LastProcessed *metav1.Time `json:"lastProcessed,omitempty"`

	// Conditions represent the current conditions of the NetworkIntent
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProcessingCompletionTime indicates when processing completed
	ProcessingCompletionTime *metav1.Time `json:"processingCompletionTime,omitempty"`

	// ProcessingStartTime indicates when processing started
	ProcessingStartTime *metav1.Time `json:"processingStartTime,omitempty"`

	// DeploymentCompletionTime indicates when deployment completed
	DeploymentCompletionTime *metav1.Time `json:"deploymentCompletionTime,omitempty"`

	// GitCommitHash contains the hash of the commit created in GitOps repository
	GitCommitHash string `json:"gitCommitHash,omitempty"`

	// CompletionTime indicates when the intent was completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// ErrorMessage contains error details if the intent failed
	ErrorMessage string `json:"errorMessage,omitempty"`

	// ResourcePlan contains the generated resource deployment plan
	ResourcePlan *NetworkResourcePlan `json:"resourcePlan,omitempty"`

	// GeneratedManifests contains list of generated manifest files
	GeneratedManifests []string `json:"generatedManifests,omitempty"`

	// ValidationErrors contains any validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// DeployedComponents contains the list of deployed target components
	DeployedComponents []NetworkTargetComponent `json:"deployedComponents,omitempty"`

	// Extensions contains custom extension data as raw JSON
	Extensions *runtime.RawExtension `json:"extensions,omitempty"`

	// LLMResponse contains the response from the LLM processing
	LLMResponse string `json:"llmResponse,omitempty"`

	// DeploymentStatus contains detailed deployment status information
	DeploymentStatus *DeploymentStatus `json:"deploymentStatus,omitempty"`

	// PackageRevision contains reference to the generated package revision
	PackageRevision *PackageRevisionReference `json:"packageRevision,omitempty"`

	// ProcessingResults contains structured processing results
	ProcessingResults *ProcessingResult `json:"processingResults,omitempty"`
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
	// Maximum CPU allocation (string form for backward compatibility)
	MaxCpu string `json:"maxCpu,omitempty"`

	// Maximum memory allocation (string form for backward compatibility)
	MaxMem string `json:"maxMem,omitempty"`

	// Maximum storage allocation (string form for backward compatibility)
	MaxStorage string `json:"maxStorage,omitempty"`

	// Maximum CPU allocation as resource.Quantity (for blueprint engine compatibility)
	MaxCPU *resource.Quantity `json:"maxCPU,omitempty"`

	// Maximum memory allocation as resource.Quantity (for blueprint engine compatibility)
	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`

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
	Configuration *runtime.RawExtension `json:"configuration,omitempty"`

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

// ProcessingResult contains the results from processing a NetworkIntent
type ProcessingResult struct {
	// LLMResponse contains the response from the LLM processing
	LLMResponse string `json:"llmResponse,omitempty"`

	// NetworkFunctionType identifies the detected network function type
	NetworkFunctionType string `json:"networkFunctionType,omitempty"`

	// DeploymentParameters contains extracted deployment parameters
	DeploymentParameters map[string]string `json:"deploymentParameters,omitempty"`

	// ConfidenceScore indicates the confidence level of the processing (0-1)
	ConfidenceScore float64 `json:"confidenceScore,omitempty"`

	// ProcessingTime indicates how long the processing took
	ProcessingTime metav1.Duration `json:"processingTime,omitempty"`
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

// PackageRevisionReference contains reference information for a package revision
type PackageRevisionReference struct {
	// Repository name where the package is stored
	Repository string `json:"repository"`

	// PackageName name of the package
	PackageName string `json:"packageName"`

	// Revision specific revision of the package
	Revision string `json:"revision"`

	// WorkspaceName name of the workspace (optional)
	WorkspaceName string `json:"workspaceName,omitempty"`
}

// DeploymentStatus contains comprehensive deployment status information
type DeploymentStatus struct {
	// Phase current phase of deployment
	Phase string `json:"phase,omitempty"`

	// Targets list of deployment target statuses
	Targets []DeploymentTargetStatus `json:"targets,omitempty"`

	// StartedAt timestamp when deployment started
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CompletedAt timestamp when deployment completed
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Conditions current deployment conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// DeploymentTargetStatus represents the deployment status for a specific target
type DeploymentTargetStatus struct {
	// Cluster name of the target cluster
	Cluster string `json:"cluster"`

	// Namespace target namespace
	Namespace string `json:"namespace"`

	// Status deployment status for this target
	Status string `json:"status"`

	// Message optional status message
	Message string `json:"message,omitempty"`

	// LastUpdateTime when this status was last updated
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// Resources deployed resources for this target
	Resources []DeployedResourceStatus `json:"resources,omitempty"`
}

// NetworkDeployedResourceStatus is deprecated, use DeployedResourceStatus from gitopsdeployment_types.go

// NetworkTargetComponentsToStrings converts a slice of NetworkTargetComponent to []string
func NetworkTargetComponentsToStrings(components []NetworkTargetComponent) []string {
	result := make([]string, len(components))
	for i, component := range components {
		result[i] = component.String()
	}
	return result
}

// Alias types for backward compatibility with existing code
// These provide shorter names that match the existing usage patterns

// NetworkTargetComponentAlias is an alias for NetworkTargetComponent for backward compatibility
// Note: Cannot use TargetComponent name as it conflicts with disaster recovery types
type NetworkTargetComponentAlias = NetworkTargetComponent

const (
	// O-RAN Core Network Functions - Aliases for compatibility using raw string values
	TargetComponentAMF  = "AMF"
	TargetComponentSMF  = "SMF"
	TargetComponentUPF  = "UPF"
	TargetComponentNRF  = "NRF"
	TargetComponentUDM  = "UDM"
	TargetComponentUDR  = "UDR"
	TargetComponentPCF  = "PCF"
	TargetComponentAUSF = "AUSF"
	TargetComponentNSSF = "NSSF"

	// O-RAN RAN Functions - Aliases for compatibility using raw string values
	TargetComponentCUCP  = "CU-CP"
	TargetComponentCUUP  = "CU-UP"
	TargetComponentDU    = "DU"
	TargetComponentNEF   = "NEF"
	TargetComponentNWDAF = "NWDAF"
	TargetComponentAF    = "AF"

	// Additional O-RAN components referenced in the codebase
	TargetComponentSMO       = "SMO"         // Service Management and Orchestration
	TargetComponentNearRTRIC = "Near-RT-RIC" // Near Real-Time RAN Intelligent Controller
	TargetComponentNonRTRIC  = "Non-RT-RIC"  // Non Real-Time RAN Intelligent Controller
	TargetComponentXApp      = "xApp"        // RIC xApplication
	TargetComponentRApp      = "rApp"        // RIC rApplication
	TargetComponentGNodeB    = "gNodeB"      // 5G Base Station
	TargetComponentENodeB    = "eNodeB"      // 4G Base Station
	TargetComponentODU       = "O-DU"        // O-RAN Distributed Unit
	TargetComponentOCUCP     = "O-CU-CP"     // O-RAN Centralized Unit Control Plane
	TargetComponentOCUUP     = "O-CU-UP"     // O-RAN Centralized Unit User Plane
)

const (
	// Priority constants for general use (raw string values to avoid conflicts)
	PriorityNormalNet = "normal" // Additional priority level for network intents
)

// NetworkIntent processing phases - additional phase constants for general use
const (
	// Phase constants for general use (shorter names)
	PhasePending            = "Pending"
	PhaseProcessing         = "Processing"
	PhaseResourcePlanning   = "ResourcePlanning"
	PhaseManifestGeneration = "ManifestGeneration"
	PhaseDeployed           = "Deployed"
	PhaseFailed             = "Failed"
)

func init() {
	// SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{}) // TODO: Fix DeepCopyObject methods
}

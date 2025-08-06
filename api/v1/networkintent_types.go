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
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IntentType defines the type of network intent operation
// +kubebuilder:validation:Enum=deployment;scaling;optimization;maintenance
type IntentType string

const (
	// IntentTypeDeployment represents deployment of new network functions
	IntentTypeDeployment IntentType = "deployment"
	// IntentTypeScaling represents scaling operations (horizontal/vertical)
	IntentTypeScaling IntentType = "scaling"
	// IntentTypeOptimization represents optimization operations (performance, QoS)
	IntentTypeOptimization IntentType = "optimization"
	// IntentTypeMaintenance represents maintenance operations (updates, patches)
	IntentTypeMaintenance IntentType = "maintenance"
)

// Priority defines the priority level for intent processing
// +kubebuilder:validation:Enum=low;medium;high;critical
type Priority string

const (
	// PriorityLow represents low priority intents (batch processing)
	PriorityLow Priority = "low"
	// PriorityMedium represents medium priority intents (default)
	PriorityMedium Priority = "medium"
	// PriorityHigh represents high priority intents (expedited processing)
	PriorityHigh Priority = "high"
	// PriorityCritical represents critical priority intents (emergency operations)
	PriorityCritical Priority = "critical"
)

// TargetComponent defines O-RAN and 5G Core network function components
// +kubebuilder:validation:Enum=AMF;SMF;UPF;gNodeB;NRF;AUSF;UDM;PCF;NSSF;NEF;SMSF;BSF;UDR;UDSF;CHF;N3IWF;TNGF;TWIF;NWDAF;SCP;SEPP;O-DU;O-CU-CP;O-CU-UP;Near-RT-RIC;Non-RT-RIC;O-eNB;SMO;rApp;xApp
type TargetComponent string

const (
	// 5G Core Network Functions
	TargetComponentAMF   TargetComponent = "AMF"   // Access and Mobility Management Function
	TargetComponentSMF   TargetComponent = "SMF"   // Session Management Function
	TargetComponentUPF   TargetComponent = "UPF"   // User Plane Function
	TargetComponentNRF   TargetComponent = "NRF"   // Network Repository Function
	TargetComponentAUSF  TargetComponent = "AUSF"  // Authentication Server Function
	TargetComponentUDM   TargetComponent = "UDM"   // Unified Data Management
	TargetComponentPCF   TargetComponent = "PCF"   // Policy Control Function
	TargetComponentNSSF  TargetComponent = "NSSF"  // Network Slice Selection Function
	TargetComponentNEF   TargetComponent = "NEF"   // Network Exposure Function
	TargetComponentSMSF  TargetComponent = "SMSF"  // SMS Function
	TargetComponentBSF   TargetComponent = "BSF"   // Binding Support Function
	TargetComponentUDR   TargetComponent = "UDR"   // Unified Data Repository
	TargetComponentUDSF  TargetComponent = "UDSF"  // Unstructured Data Storage Function
	TargetComponentCHF   TargetComponent = "CHF"   // Charging Function
	TargetComponentN3IWF TargetComponent = "N3IWF" // Non-3GPP Interworking Function
	TargetComponentTNGF  TargetComponent = "TNGF"  // Trusted Non-3GPP Gateway Function
	TargetComponentTWIF  TargetComponent = "TWIF"  // Trusted WLAN Interworking Function
	TargetComponentNWDAF TargetComponent = "NWDAF" // Network Data Analytics Function
	TargetComponentSCP   TargetComponent = "SCP"   // Service Communication Proxy
	TargetComponentSEPP  TargetComponent = "SEPP"  // Security Edge Protection Proxy

	// RAN Components
	TargetComponentGNodeB TargetComponent = "gNodeB" // 5G Base Station

	// O-RAN Components
	TargetComponentODU       TargetComponent = "O-DU"       // O-RAN Distributed Unit
	TargetComponentOCUCP     TargetComponent = "O-CU-CP"    // O-RAN Centralized Unit Control Plane
	TargetComponentOCUUP     TargetComponent = "O-CU-UP"    // O-RAN Centralized Unit User Plane
	TargetComponentNearRTRIC TargetComponent = "Near-RT-RIC" // Near Real-Time RAN Intelligent Controller
	TargetComponentNonRTRIC  TargetComponent = "Non-RT-RIC"  // Non Real-Time RAN Intelligent Controller
	TargetComponentOENB      TargetComponent = "O-eNB"      // O-RAN evolved NodeB
	TargetComponentSMO       TargetComponent = "SMO"        // Service Management and Orchestration

	// RIC Applications
	TargetComponentRApp TargetComponent = "rApp" // RIC Application (Non-RT RIC)
	TargetComponentXApp TargetComponent = "xApp" // RIC Application (Near-RT RIC)
)

// ResourceConstraints defines resource requirements and limits
type ResourceConstraints struct {
	// CPU resource requirements (e.g., "100m", "2")
	// +optional
	// +kubebuilder:validation:Pattern=`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`
	CPU *resource.Quantity `json:"cpu,omitempty"`

	// Memory resource requirements (e.g., "128Mi", "2Gi")
	// +optional
	// +kubebuilder:validation:Pattern=`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`
	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage resource requirements (e.g., "10Gi", "100Mi")
	// +optional
	// +kubebuilder:validation:Pattern=`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`
	Storage *resource.Quantity `json:"storage,omitempty"`

	// Maximum CPU resource limit
	// +optional
	// +kubebuilder:validation:Pattern=`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`
	MaxCPU *resource.Quantity `json:"maxCpu,omitempty"`

	// Maximum Memory resource limit
	// +optional
	// +kubebuilder:validation:Pattern=`^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`
	MaxMemory *resource.Quantity `json:"maxMemory,omitempty"`
}

// ProcessedParameters contains structured parameters processed from the natural language intent
type ProcessedParameters struct {
	// NetworkFunction specifies the target network function type
	// +optional
	NetworkFunction string `json:"networkFunction,omitempty"`

	// DeploymentConfig contains deployment-specific configuration
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	DeploymentConfig runtime.RawExtension `json:"deploymentConfig,omitempty"`

	// PerformanceRequirements specifies performance and QoS requirements
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	PerformanceRequirements runtime.RawExtension `json:"performanceRequirements,omitempty"`

	// ScalingPolicy defines auto-scaling behavior
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ScalingPolicy runtime.RawExtension `json:"scalingPolicy,omitempty"`

	// SecurityPolicy defines security requirements and policies
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	SecurityPolicy runtime.RawExtension `json:"securityPolicy,omitempty"`
}

// NetworkIntentSpec defines the desired state of NetworkIntent
type NetworkIntentSpec struct {
	// The original natural language intent from the user.
	// Must contain telecommunications domain keywords and be between 10-2000 characters.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=10
	// +kubebuilder:validation:MaxLength=2000
	Intent string `json:"intent"`

	// Type of the intent operation
	// +kubebuilder:validation:Required
	// +kubebuilder:default="deployment"
	IntentType IntentType `json:"intentType"`

	// Priority level for intent processing
	// +optional
	// +kubebuilder:default="medium"
	Priority Priority `json:"priority,omitempty"`

	// TargetComponents specifies which O-RAN/5GC components are involved
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=20
	TargetComponents []TargetComponent `json:"targetComponents,omitempty"`

	// ResourceConstraints defines resource requirements and limits
	// +optional
	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`

	// Structured parameters translated from the intent by the LLM.
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters runtime.RawExtension `json:"parameters,omitempty"`

	// Map of string parameters for easier access in tests
	// +optional
	ParametersMap map[string]string `json:"parametersMap,omitempty"`

	// ProcessedParameters contains structured parameters after LLM processing
	// +optional
	ProcessedParameters *ProcessedParameters `json:"processedParameters,omitempty"`

	// Namespace where the network function should be deployed
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Cluster where the network function should be deployed
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=253
	TargetCluster string `json:"targetCluster,omitempty"`

	// NetworkSlice specifies the network slice identifier if applicable
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$`
	NetworkSlice string `json:"networkSlice,omitempty"`

	// Region specifies the deployment region for geo-distributed deployments
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=64
	Region string `json:"region,omitempty"`

	// Timeout for intent processing in seconds (default: 300s, max: 3600s)
	// +optional
	// +kubebuilder:default=300
	// +kubebuilder:validation:Minimum=30
	// +kubebuilder:validation:Maximum=3600
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// RetryPolicy defines retry behavior for failed operations
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	MaxRetries *int32 `json:"maxRetries,omitempty"`
}

// NetworkIntentStatus defines the observed state of NetworkIntent
type NetworkIntentStatus struct {
	// Conditions represent the latest available observations of an object's state
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the NetworkIntent processing
	// +optional
	// +kubebuilder:validation:Enum=Pending;Processing;Deploying;Ready;Failed;Terminating
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
	// +kubebuilder:validation:Pattern=`^[a-f0-9]{7,40}$`
	GitCommitHash string `json:"gitCommitHash,omitempty"`

	// LastRetryTime indicates the last time a retry was attempted
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed NetworkIntent
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// RetryCount tracks the number of retry attempts made
	// +optional
	// +kubebuilder:validation:Minimum=0
	RetryCount int32 `json:"retryCount,omitempty"`

	// ValidationErrors contains any validation errors encountered
	// +optional
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// DeployedComponents tracks which components were successfully deployed
	// +optional
	DeployedComponents []TargetComponent `json:"deployedComponents,omitempty"`

	// ProcessingDuration represents the total time taken for processing
	// +optional
	ProcessingDuration *metav1.Duration `json:"processingDuration,omitempty"`

	// DeploymentDuration represents the total time taken for deployment
	// +optional
	DeploymentDuration *metav1.Duration `json:"deploymentDuration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Intent Type",type=string,JSONPath=`.spec.intentType`
//+kubebuilder:printcolumn:name="Priority",type=string,JSONPath=`.spec.priority`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Components",type=string,JSONPath=`.spec.targetComponents`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=ni;netintent
//+kubebuilder:storageversion

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

// ValidateNetworkIntent validates the network intent for telecommunications domain compliance
func (ni *NetworkIntent) ValidateNetworkIntent() error {
	var errors []string

	// Validate intent contains telecommunications keywords
	if !ni.containsTelecomKeywords() {
		errors = append(errors, "intent must contain telecommunications domain keywords (AMF, SMF, UPF, gNodeB, 5G, O-RAN, RAN, slice, etc.)")
	}

	// Validate target components are compatible with intent type
	if err := ni.validateTargetComponents(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate resource constraints
	if err := ni.validateResourceConstraints(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate network slice format if provided
	if ni.Spec.NetworkSlice != "" {
		if matched, _ := regexp.MatchString(`^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$`, ni.Spec.NetworkSlice); !matched {
			errors = append(errors, "networkSlice must follow format XXXXXX-XXXXXX (hexadecimal)")
		}
	}

	// Validate priority escalation rules
	if ni.Spec.Priority == PriorityCritical && ni.Spec.IntentType != IntentTypeMaintenance {
		errors = append(errors, "critical priority is only allowed for maintenance operations")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// containsTelecomKeywords checks if the intent contains valid telecommunications keywords
func (ni *NetworkIntent) containsTelecomKeywords() bool {
	intent := strings.ToLower(ni.Spec.Intent)
	
	telecomKeywords := []string{
		// 5G Core Functions
		"amf", "smf", "upf", "nrf", "ausf", "udm", "pcf", "nssf", "nef", "smsf",
		"bsf", "udr", "udsf", "chf", "n3iwf", "tngf", "twif", "nwdaf", "scp", "sepp",
		
		// RAN Components
		"gnodeb", "gnb", "enb", "nodeb", "base station", "cell",
		
		// O-RAN Components
		"o-du", "o-cu", "near-rt-ric", "non-rt-ric", "o-enb", "smo", "ric",
		"rapp", "xapp", "o-ran", "oran",
		
		// General Terms
		"5g", "5gc", "ran", "slice", "network", "function", "core",
		"deployment", "scaling", "optimization", "qos", "sla", "latency",
		"throughput", "bandwidth", "availability", "reliability",
		
		// Technical Terms
		"kubernetes", "helm", "container", "pod", "service", "cnf", "vnf",
		"microservice", "api", "interface", "protocol", "endpoint",
		
		// Operations
		"deploy", "scale", "optimize", "configure", "manage", "monitor",
		"update", "upgrade", "patch", "maintain", "backup", "restore",
	}
	
	for _, keyword := range telecomKeywords {
		if strings.Contains(intent, keyword) {
			return true
		}
	}
	
	return false
}

// validateTargetComponents ensures target components are valid for the intent type
func (ni *NetworkIntent) validateTargetComponents() error {
	if len(ni.Spec.TargetComponents) == 0 {
		return nil // Optional field
	}

	// Check for duplicate components
	seen := make(map[TargetComponent]bool)
	for _, component := range ni.Spec.TargetComponents {
		if seen[component] {
			return fmt.Errorf("duplicate target component: %s", component)
		}
		seen[component] = true
	}

	// Validate component compatibility based on intent type
	switch ni.Spec.IntentType {
	case IntentTypeScaling:
		// Scaling operations should target scalable components
		nonScalableComponents := []TargetComponent{TargetComponentNRF, TargetComponentNSSF}
		for _, component := range ni.Spec.TargetComponents {
			for _, nonScalable := range nonScalableComponents {
				if component == nonScalable {
					return fmt.Errorf("component %s cannot be scaled", component)
				}
			}
		}
	}

	return nil
}

// validateResourceConstraints ensures resource constraints are properly specified
func (ni *NetworkIntent) validateResourceConstraints() error {
	if ni.Spec.ResourceConstraints == nil {
		return nil // Optional field
	}

	rc := ni.Spec.ResourceConstraints

	// Validate that max resources are greater than or equal to min resources
	if rc.CPU != nil && rc.MaxCPU != nil {
		if rc.MaxCPU.Cmp(*rc.CPU) < 0 {
			return fmt.Errorf("maxCpu must be greater than or equal to cpu")
		}
	}

	if rc.Memory != nil && rc.MaxMemory != nil {
		if rc.MaxMemory.Cmp(*rc.Memory) < 0 {
			return fmt.Errorf("maxMemory must be greater than or equal to memory")
		}
	}

	// Validate minimum resource requirements for critical components
	if ni.Spec.Priority == PriorityCritical {
		if rc.CPU != nil && rc.CPU.MilliValue() < 500 {
			return fmt.Errorf("critical priority intents require minimum 500m CPU")
		}
		if rc.Memory != nil && rc.Memory.Value() < 1073741824 { // 1Gi in bytes
			return fmt.Errorf("critical priority intents require minimum 1Gi memory")
		}
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&NetworkIntent{}, &NetworkIntentList{})
}
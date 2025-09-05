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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourcePlanPhase represents the phase of resource planning.

type ResourcePlanPhase string

const (

	// ResourcePlanPhasePending indicates planning is pending.

	ResourcePlanPhasePending ResourcePlanPhase = "Pending"

	// ResourcePlanPhaseAnalyzing indicates requirements analysis is in progress.

	ResourcePlanPhaseAnalyzing ResourcePlanPhase = "Analyzing"

	// ResourcePlanPhasePlanning indicates resource planning is in progress.

	ResourcePlanPhasePlanning ResourcePlanPhase = "Planning"

	// ResourcePlanPhaseOptimizing indicates optimization is in progress.

	ResourcePlanPhaseOptimizing ResourcePlanPhase = "Optimizing"

	// ResourcePlanPhaseValidating indicates validation is in progress.

	ResourcePlanPhaseValidating ResourcePlanPhase = "Validating"

	// ResourcePlanPhaseCompleted indicates planning is completed.

	ResourcePlanPhaseCompleted ResourcePlanPhase = "Completed"

	// ResourcePlanPhaseFailed indicates planning has failed.

	ResourcePlanPhaseFailed ResourcePlanPhase = "Failed"
)

// DeploymentPattern represents the pattern for deployment.

type DeploymentPattern string

const (

	// DeploymentPatternStandalone represents standalone deployment.

	DeploymentPatternStandalone DeploymentPattern = "Standalone"

	// DeploymentPatternHighAvailability represents HA deployment.

	DeploymentPatternHighAvailability DeploymentPattern = "HighAvailability"

	// DeploymentPatternDistributed represents distributed deployment.

	DeploymentPatternDistributed DeploymentPattern = "Distributed"

	// DeploymentPatternEdge represents edge deployment.

	DeploymentPatternEdge DeploymentPattern = "Edge"

	// DeploymentPatternHybrid represents hybrid cloud deployment.

	DeploymentPatternHybrid DeploymentPattern = "Hybrid"
)

// ResourcePlanSpec defines the desired state of ResourcePlan.

type ResourcePlanSpec struct {
	// ParentIntentRef references the parent NetworkIntent.

	// +kubebuilder:validation:Required

	ParentIntentRef ObjectReference `json:"parentIntentRef"`

	// IntentProcessingRef references the IntentProcessing resource.

	// +optional

	IntentProcessingRef *ObjectReference `json:"intentProcessingRef,omitempty"`

	// RequirementsInput contains the input requirements for planning.

	// +kubebuilder:validation:Required

	// +kubebuilder:pruning:PreserveUnknownFields

	RequirementsInput runtime.RawExtension `json:"requirementsInput"`

	// TargetComponents specifies which components to plan for.

	// +optional

	TargetComponents []TargetComponent `json:"targetComponents,omitempty"`

	// DeploymentPattern specifies the deployment pattern to use.

	// +optional

	// +kubebuilder:default="Standalone"

	DeploymentPattern DeploymentPattern `json:"deploymentPattern,omitempty"`

	// ResourceConstraints defines constraints for resource allocation.

	// +optional

	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`

	// OptimizationGoals defines optimization objectives.

	// +optional

	OptimizationGoals []OptimizationGoal `json:"optimizationGoals,omitempty"`

	// TargetClusters specifies clusters for deployment.

	// +optional

	TargetClusters []ClusterReference `json:"targetClusters,omitempty"`

	// NetworkSlice specifies network slice requirements.

	// +optional

	NetworkSlice *NetworkSliceSpec `json:"networkSlice,omitempty"`

	// Priority defines planning priority.

	// +optional

	// +kubebuilder:default="medium"

	Priority Priority `json:"priority,omitempty"`

	// EnableCostOptimization enables cost optimization.

	// +optional

	// +kubebuilder:default=true

	EnableCostOptimization *bool `json:"enableCostOptimization,omitempty"`

	// EnablePerformanceOptimization enables performance optimization.

	// +optional

	// +kubebuilder:default=true

	EnablePerformanceOptimization *bool `json:"enablePerformanceOptimization,omitempty"`

	// ComplianceRequirements specifies compliance requirements.

	// +optional

	ComplianceRequirements []ComplianceRequirement `json:"complianceRequirements,omitempty"`
}

// OptimizationGoal represents an optimization objective.

type OptimizationGoal struct {
	// Type specifies the optimization type.

	// +kubebuilder:validation:Required

	// +kubebuilder:validation:Enum=cost;performance;availability;latency;throughput;energy

	Type string `json:"type"`

	// Priority specifies the goal priority.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=10

	Priority int `json:"priority"`

	// Weight specifies the relative weight (0.0-1.0).

	// +optional

	// +kubebuilder:validation:Pattern=`^(0(\.[0-9]+)?|1(\.0+)?)$`

	Weight *string `json:"weight,omitempty"`

	// Target specifies the target value.

	// +optional

	Target *resource.Quantity `json:"target,omitempty"`

	// Constraints specifies additional constraints.

	// +optional

	Constraints map[string]string `json:"constraints,omitempty"`
}

// ClusterReference represents a reference to a cluster.

type ClusterReference struct {
	// Name of the cluster.

	// +kubebuilder:validation:Required

	Name string `json:"name"`

	// Region where the cluster is located.

	// +optional

	Region string `json:"region,omitempty"`

	// Zone where the cluster is located.

	// +optional

	Zone string `json:"zone,omitempty"`

	// CloudProvider of the cluster.

	// +optional

	// +kubebuilder:validation:Enum=aws;azure;gcp;openstack;on-premises

	CloudProvider string `json:"cloudProvider,omitempty"`

	// Capabilities of the cluster.

	// +optional

	Capabilities []string `json:"capabilities,omitempty"`

	// ResourceLimits for the cluster.

	// +optional

	ResourceLimits *ResourceConstraints `json:"resourceLimits,omitempty"`

	// Labels for the cluster.

	// +optional

	Labels map[string]string `json:"labels,omitempty"`

	// Priority for cluster selection.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=10

	Priority *int `json:"priority,omitempty"`
}

// NetworkSliceSpec defines network slice specifications.

type NetworkSliceSpec struct {
	// SliceID uniquely identifies the network slice.

	// +kubebuilder:validation:Required

	// +kubebuilder:validation:Pattern=`^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$`

	SliceID string `json:"sliceId"`

	// SliceType defines the type of network slice.

	// +kubebuilder:validation:Required

	// +kubebuilder:validation:Enum=eMBB;URLLC;mMTC;custom

	SliceType string `json:"sliceType"`

	// ServiceLevelAgreement defines SLA requirements.

	// +optional

	ServiceLevelAgreement *SLARequirements `json:"serviceLevelAgreement,omitempty"`

	// QualityOfService defines QoS parameters.

	// +optional

	QualityOfService *QoSRequirements `json:"qualityOfService,omitempty"`

	// IsolationLevel defines isolation requirements.

	// +optional

	// +kubebuilder:validation:Enum=none;logical;physical

	// +kubebuilder:default="logical"

	IsolationLevel string `json:"isolationLevel,omitempty"`

	// ResourceAllocation defines resource allocation strategy.

	// +optional

	// +kubebuilder:validation:Enum=shared;dedicated;hybrid

	// +kubebuilder:default="shared"

	ResourceAllocation string `json:"resourceAllocation,omitempty"`
}

// SLARequirements defines Service Level Agreement requirements.

type SLARequirements struct {
	// AvailabilityTarget as percentage (0.0-100.0).

	// +optional

	

	

	AvailabilityTarget *string `json:"availabilityTarget,omitempty"`

	// MaxLatency in milliseconds.

	// +optional

	// +kubebuilder:validation:Minimum=1

	MaxLatency *int32 `json:"maxLatency,omitempty"`

	// MinThroughput in Mbps.

	// +optional

	// +kubebuilder:validation:Minimum=1

	MinThroughput *int32 `json:"minThroughput,omitempty"`

	// MaxPacketLoss as percentage (0.0-100.0).

	// +optional

	

	

	MaxPacketLoss *string `json:"maxPacketLoss,omitempty"`

	// RecoveryTimeObjective in seconds.

	// +optional

	// +kubebuilder:validation:Minimum=1

	RecoveryTimeObjective *int32 `json:"recoveryTimeObjective,omitempty"`

	// RecoveryPointObjective in seconds.

	// +optional

	// +kubebuilder:validation:Minimum=0

	RecoveryPointObjective *int32 `json:"recoveryPointObjective,omitempty"`
}

// QoSRequirements defines Quality of Service requirements.

type QoSRequirements struct {
	// QCI (QoS Class Identifier) for 4G.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=9

	QCI *int32 `json:"qci,omitempty"`

	// FiveQI (5G QoS Identifier).

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=255

	FiveQI *int32 `json:"fiveQI,omitempty"`

	// Priority level.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=15

	PriorityLevel *int32 `json:"priorityLevel,omitempty"`

	// GuaranteedBitRate in Mbps.

	// +optional

	GuaranteedBitRate *resource.Quantity `json:"guaranteedBitRate,omitempty"`

	// MaxBitRate in Mbps.

	// +optional

	MaxBitRate *resource.Quantity `json:"maxBitRate,omitempty"`

	// AllocationRetentionPriority.

	// +optional

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=15

	AllocationRetentionPriority *int32 `json:"allocationRetentionPriority,omitempty"`
}

// ComplianceRequirement defines a compliance requirement.

type ComplianceRequirement struct {
	// Standard specifies the compliance standard.

	// +kubebuilder:validation:Required

	// +kubebuilder:validation:Enum="3GPP";"ETSI";"ORAN";"FCC";"CE";"GDPR";"HIPAA";"SOC2";"ISO27001"

	Standard string `json:"standard"`

	// Version of the standard.

	// +optional

	Version string `json:"version,omitempty"`

	// Requirements specifies specific requirements.

	// +optional

	Requirements []string `json:"requirements,omitempty"`

	// Mandatory indicates if compliance is mandatory.

	// +optional

	// +kubebuilder:default=true

	Mandatory *bool `json:"mandatory,omitempty"`
}

// ResourcePlanStatus defines the observed state of ResourcePlan.

type ResourcePlanStatus struct {
	// Phase represents the current planning phase.

	// +optional

	Phase ResourcePlanPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations.

	// +optional

	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// PlanningStartTime indicates when planning started.

	// +optional

	PlanningStartTime *metav1.Time `json:"planningStartTime,omitempty"`

	// PlanningCompletionTime indicates when planning completed.

	// +optional

	PlanningCompletionTime *metav1.Time `json:"planningCompletionTime,omitempty"`

	// PlannedResources contains the planned resource allocation.

	// +optional

	PlannedResources []PlannedResource `json:"plannedResources,omitempty"`

	// ResourceAllocation contains the detailed resource allocation.

	// +optional

	// +kubebuilder:pruning:PreserveUnknownFields

	ResourceAllocation runtime.RawExtension `json:"resourceAllocation,omitempty"`

	// CostEstimate contains the estimated costs.

	// +optional

	CostEstimate *CostEstimate `json:"costEstimate,omitempty"`

	// PerformanceEstimate contains performance projections.

	// +optional

	PerformanceEstimate *PerformanceEstimate `json:"performanceEstimate,omitempty"`

	// ComplianceStatus tracks compliance validation results.

	// +optional

	ComplianceStatus []ResourceComplianceStatus `json:"complianceStatus,omitempty"`

	// OptimizationResults contains optimization outcomes.

	// +optional

	OptimizationResults []OptimizationResult `json:"optimizationResults,omitempty"`

	// ValidationResults contains validation outcomes.

	// +optional

	ValidationResults []ValidationResult `json:"validationResults,omitempty"`

	// PlanningDuration represents total planning time.

	// +optional

	PlanningDuration *metav1.Duration `json:"planningDuration,omitempty"`

	// RetryCount tracks retry attempts.

	// +optional

	RetryCount int32 `json:"retryCount,omitempty"`

	// QualityScore represents the quality of the plan (0.0-1.0).

	// +optional

	

	

	QualityScore *string `json:"qualityScore,omitempty"`

	// ObservedGeneration reflects the generation observed.

	// +optional

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// PlannedResource represents a planned resource.

type PlannedResource struct {
	// Name of the resource.

	Name string `json:"name"`

	// Type of the resource (NetworkFunction, Service, etc.).

	Type string `json:"type"`

	// Component this resource belongs to.

	// +optional

	Component TargetComponent `json:"component,omitempty"`

	// Cluster where the resource will be deployed.

	// +optional

	Cluster string `json:"cluster,omitempty"`

	// Namespace where the resource will be deployed.

	// +optional

	Namespace string `json:"namespace,omitempty"`

	// ResourceRequirements for the resource.

	ResourceRequirements ResourceSpec `json:"resourceRequirements"`

	// Replicas specifies the number of replicas.

	// +optional

	// +kubebuilder:default=1

	// +kubebuilder:validation:Minimum=0

	Replicas *int32 `json:"replicas,omitempty"`

	// Configuration contains resource-specific configuration.

	// +optional

	// +kubebuilder:pruning:PreserveUnknownFields

	Configuration runtime.RawExtension `json:"configuration,omitempty"`

	// Dependencies lists resource dependencies.

	// +optional

	Dependencies []string `json:"dependencies,omitempty"`

	// Labels for the resource.

	// +optional

	Labels map[string]string `json:"labels,omitempty"`

	// Annotations for the resource.

	// +optional

	Annotations map[string]string `json:"annotations,omitempty"`

	// EstimatedCost for the resource (in USD).

	// +optional

	

	EstimatedCost *string `json:"estimatedCost,omitempty"`
}

// ResourceSpec defines resource specifications.

type ResourceSpec struct {
	// Requests defines resource requests.

	Requests ResourceList `json:"requests"`

	// Limits defines resource limits.

	// +optional

	Limits ResourceList `json:"limits,omitempty"`
}

// ResourceList defines a list of compute resources.

type ResourceList struct {
	// CPU resource.

	// +optional

	CPU *resource.Quantity `json:"cpu,omitempty"`

	// Memory resource.

	// +optional

	Memory *resource.Quantity `json:"memory,omitempty"`

	// Storage resource.

	// +optional

	Storage *resource.Quantity `json:"storage,omitempty"`

	// EphemeralStorage resource.

	// +optional

	EphemeralStorage *resource.Quantity `json:"ephemeralStorage,omitempty"`

	// Extended resources (GPUs, FPGAs, etc.).

	// +optional

	Extended map[string]resource.Quantity `json:"extended,omitempty"`
}

// CostEstimate represents cost estimation.

type CostEstimate struct {
	// TotalCost is the total estimated cost.

	TotalCost string `json:"totalCost"`

	// Currency for the cost.

	// +kubebuilder:default="USD"

	Currency string `json:"currency"`

	// BillingPeriod (hourly, daily, monthly).

	// +kubebuilder:default="monthly"

	BillingPeriod string `json:"billingPeriod"`

	// CostBreakdown by component.

	// +optional

	CostBreakdown map[string]float64 `json:"costBreakdown,omitempty"`

	// EstimatedAt timestamp.

	EstimatedAt metav1.Time `json:"estimatedAt"`

	// Confidence level of the estimate (0.0-1.0).

	// +optional

	

	

	Confidence *string `json:"confidence,omitempty"`
}

// PerformanceEstimate represents performance estimation.

type PerformanceEstimate struct {
	// ExpectedThroughput in requests per second.

	// +optional

	

	ExpectedThroughput *string `json:"expectedThroughput,omitempty"`

	// ExpectedLatency in milliseconds.

	// +optional

	

	ExpectedLatency *string `json:"expectedLatency,omitempty"`

	// ExpectedAvailability as percentage (0.0-100.0).

	// +optional

	

	

	ExpectedAvailability *string `json:"expectedAvailability,omitempty"`

	// ResourceUtilization estimates.

	// +optional

	ResourceUtilization map[string]float64 `json:"resourceUtilization,omitempty"`

	// BottleneckAnalysis identifies potential bottlenecks.

	// +optional

	BottleneckAnalysis []string `json:"bottleneckAnalysis,omitempty"`

	// ScalingRecommendations for horizontal scaling.

	// +optional

	ScalingRecommendations []ScalingRecommendation `json:"scalingRecommendations,omitempty"`
}

// ScalingRecommendation represents a scaling recommendation.

type ScalingRecommendation struct {
	// Resource name.

	Resource string `json:"resource"`

	// RecommendedReplicas.

	RecommendedReplicas int32 `json:"recommendedReplicas"`

	// Reason for the recommendation.

	Reason string `json:"reason"`

	// Impact of the scaling.

	// +optional

	Impact string `json:"impact,omitempty"`

	// Confidence in the recommendation (0.0-1.0).

	// +optional

	

	

	Confidence *string `json:"confidence,omitempty"`
}

// ResourceComplianceStatus represents compliance validation status.

type ResourceComplianceStatus struct {
	// Standard being validated.

	Standard string `json:"standard"`

	// Version of the standard.

	// +optional

	Version string `json:"version,omitempty"`

	// Status of compliance (Compliant, NonCompliant, Unknown).

	Status string `json:"status"`

	// ValidationResults for specific requirements.

	// +optional

	ValidationResults []string `json:"validationResults,omitempty"`

	// Violations if any.

	// +optional

	Violations []ComplianceViolation `json:"violations,omitempty"`

	// ValidatedAt timestamp.

	ValidatedAt metav1.Time `json:"validatedAt"`
}

// ComplianceViolation represents a compliance violation.

type ComplianceViolation struct {
	// Type of violation.

	Type string `json:"type"`

	// Severity of violation (Critical, Major, Minor).

	Severity string `json:"severity"`

	// Description of the violation.

	Description string `json:"description"`

	// Resource affected.

	// +optional

	Resource string `json:"resource,omitempty"`

	// Remediation suggestion.

	// +optional

	Remediation string `json:"remediation,omitempty"`
}

// OptimizationResult represents the result of an optimization.

type OptimizationResult struct {
	// Type of optimization.

	Type string `json:"type"`

	// Status of optimization (Success, Failed, Skipped).

	Status string `json:"status"`

	// ImprovementPercent achieved (0.0-100.0).

	// +optional

	

	

	ImprovementPercent *string `json:"improvementPercent,omitempty"`

	// Description of the optimization.

	// +optional

	Description string `json:"description,omitempty"`

	// AppliedChanges during optimization.

	// +optional

	AppliedChanges []string `json:"appliedChanges,omitempty"`

	// OptimizedAt timestamp.

	OptimizedAt metav1.Time `json:"optimizedAt"`
}

// ValidationResult represents the result of a validation.

type ValidationResult struct {
	// Type of validation.

	Type string `json:"type"`

	// Status of validation (Passed, Failed, Warning).

	Status string `json:"status"`

	// Message describing the result.

	Message string `json:"message"`

	// Details with additional information.

	// +optional

	Details map[string]string `json:"details,omitempty"`

	// ValidatedAt timestamp.

	ValidatedAt metav1.Time `json:"validatedAt"`
}

//+kubebuilder:object:root=true

//+kubebuilder:subresource:status

//+kubebuilder:printcolumn:name="Parent Intent",type=string,JSONPath=`.spec.parentIntentRef.name`

//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

//+kubebuilder:printcolumn:name="Pattern",type=string,JSONPath=`.spec.deploymentPattern`

//+kubebuilder:printcolumn:name="Resources",type=integer,JSONPath=`.status.plannedResources[*].name | length(@)`

//+kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.costEstimate.totalCost`

//+kubebuilder:printcolumn:name="Quality",type=string,JSONPath=`.status.qualityScore`

//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

//+kubebuilder:resource:shortName=rp;resplan

//+kubebuilder:storageversion

// ResourcePlan is the Schema for the resourceplans API.

type ResourcePlan struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourcePlanSpec `json:"spec,omitempty"`

	Status ResourcePlanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ResourcePlanList contains a list of ResourcePlan.

type ResourcePlanList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ResourcePlan `json:"items"`
}

// GetParentIntentName returns the name of the parent NetworkIntent.

func (rp *ResourcePlan) GetParentIntentName() string {
	return rp.Spec.ParentIntentRef.Name
}

// GetNamespace returns the namespace of the resource.

func (rp *ResourcePlan) GetNamespace() string {
	return rp.ObjectMeta.Namespace
}

// GetParentIntentNamespace returns the namespace of the parent NetworkIntent.

func (rp *ResourcePlan) GetParentIntentNamespace() string {
	if rp.Spec.ParentIntentRef.Namespace != "" {
		return rp.Spec.ParentIntentRef.Namespace
	}

	return rp.GetNamespace()
}

// IsPlanningComplete returns true if planning is complete.

func (rp *ResourcePlan) IsPlanningComplete() bool {
	return rp.Status.Phase == ResourcePlanPhaseCompleted
}

// IsPlanningFailed returns true if planning has failed.

func (rp *ResourcePlan) IsPlanningFailed() bool {
	return rp.Status.Phase == ResourcePlanPhaseFailed
}

// GetTotalEstimatedCost returns the total estimated cost.

func (rp *ResourcePlan) GetTotalEstimatedCost() string {
	if rp.Status.CostEstimate != nil {
		return rp.Status.CostEstimate.TotalCost
	}

	return "0.0"
}

// HasCompliantResources returns true if all compliance requirements are met.

func (rp *ResourcePlan) HasCompliantResources() bool {
	for _, cs := range rp.Status.ComplianceStatus {
		if cs.Status != "Compliant" {
			return false
		}
	}

	return true
}

// GetPlannedResourceCount returns the number of planned resources.

func (rp *ResourcePlan) GetPlannedResourceCount() int {
	return len(rp.Status.PlannedResources)
}

// ShouldOptimizeCost returns true if cost optimization is enabled.

func (rp *ResourcePlan) ShouldOptimizeCost() bool {
	if rp.Spec.EnableCostOptimization == nil {
		return true
	}

	return *rp.Spec.EnableCostOptimization
}

// ShouldOptimizePerformance returns true if performance optimization is enabled.

func (rp *ResourcePlan) ShouldOptimizePerformance() bool {
	if rp.Spec.EnablePerformanceOptimization == nil {
		return true
	}

	return *rp.Spec.EnablePerformanceOptimization
}

func init() {
	SchemeBuilder.Register(&ResourcePlan{}, &ResourcePlanList{})
}

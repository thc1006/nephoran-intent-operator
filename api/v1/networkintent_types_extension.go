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

// ExtendedProcessedParameters extends ProcessedParameters with CNF-specific fields.

// This extends the existing ProcessedParameters without modifying the original type.

type ExtendedProcessedParameters struct {
	ProcessedParameters `json:",inline"`

	// CNFDeploymentIntent contains CNF deployment specifications processed from intent.

	// +optional.

	CNFDeploymentIntent *CNFDeploymentIntent `json:"cnfDeploymentIntent,omitempty"`

	// CNFProcessingResult contains the result of CNF intent processing.

	// +optional.

	CNFProcessingResult *CNFIntentProcessingResult `json:"cnfProcessingResult,omitempty"`

	// CNFTopology defines network topology and connectivity requirements.

	// +optional.

	CNFTopology *CNFTopologyIntent `json:"cnfTopology,omitempty"`

	// MultiClusterDeployment defines multi-cluster CNF deployment requirements.

	// +optional.

	MultiClusterDeployment *MultiClusterRequirement `json:"multiClusterDeployment,omitempty"`

	// EdgeDeployment defines edge-specific deployment requirements.

	// +optional.

	EdgeDeployment *EdgeDeploymentRequirement `json:"edgeDeployment,omitempty"`

	// ServiceChaining defines service function chaining requirements.

	// +optional.

	ServiceChaining []ServiceChainRequirement `json:"serviceChaining,omitempty"`

	// NetworkSlicing defines network slicing requirements for CNFs.

	// +optional.

	NetworkSlicing *NetworkSlicingIntent `json:"networkSlicing,omitempty"`
}

// CNFIntentMetadata contains metadata about CNF intent processing.

type CNFIntentMetadata struct {
	// ProcessingTimestamp when the intent was processed.

	ProcessingTimestamp string `json:"processingTimestamp,omitempty"`

	// ProcessingDuration how long the processing took.

	ProcessingDurationMs int64 `json:"processingDurationMs,omitempty"`

	// LLMModel used for intent processing.

	LLMModel string `json:"llmModel,omitempty"`

	// RAGSources used for context enrichment.

	RAGSources []string `json:"ragSources,omitempty"`

	// ConfidenceScore overall confidence in processing results.

<<<<<<< HEAD
	ConfidenceScore float64 `json:"confidenceScore,omitempty"`
=======
	ConfidenceScore string `json:"confidenceScore,omitempty"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// ProcessingVersion version of the processing pipeline.

	ProcessingVersion string `json:"processingVersion,omitempty"`

	// Warnings generated during processing.

	Warnings []string `json:"warnings,omitempty"`
}

// CNFDeploymentPlan represents a complete deployment plan for CNFs.

type CNFDeploymentPlan struct {
	// PlanID unique identifier for the deployment plan.

	PlanID string `json:"planId"`

	// CNFDeployments list of CNF deployments in the plan.

	CNFDeployments []CNFDeploymentIntent `json:"cnfDeployments"`

	// DeploymentOrder specifies the order of deployments.

	DeploymentOrder []DeploymentPhase `json:"deploymentOrder,omitempty"`

	// EstimatedResources total estimated resources for the plan.

	// EstimatedResources contains resource estimates.

	// +kubebuilder:pruning:PreserveUnknownFields

	EstimatedResources runtime.RawExtension `json:"estimatedResources"`

	// EstimatedCost total estimated cost for the plan.

<<<<<<< HEAD
	EstimatedCost float64 `json:"estimatedCost"`
=======
	EstimatedCost string `json:"estimatedCost"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// EstimatedDeploymentTime total estimated deployment time in minutes.

	EstimatedDeploymentTime int32 `json:"estimatedDeploymentTime"`

	// Dependencies between CNF deployments.

	Dependencies []CNFDependency `json:"dependencies,omitempty"`

	// ValidationResults from plan validation.

	ValidationResults *PlanValidationResult `json:"validationResults,omitempty"`

	// Metadata about the plan creation.

	Metadata CNFIntentMetadata `json:"metadata"`
}

// DeploymentPhase represents a phase in the deployment plan.

type DeploymentPhase struct {
	// PhaseName identifier for the phase.

	PhaseName string `json:"phaseName"`

	// CNFDeployments to deploy in this phase.

	CNFDeployments []string `json:"cnfDeployments"`

	// EstimatedDuration for this phase in minutes.

	EstimatedDuration int32 `json:"estimatedDuration"`

	// Prerequisites that must be met before starting this phase.

	Prerequisites []string `json:"prerequisites,omitempty"`
}

// CNFDependency defines dependency relationships between CNFs.

type CNFDependency struct {
	// Source CNF that depends on Target.

	Source string `json:"source"`

	// Target CNF that Source depends on.

	Target string `json:"target"`

	// DependencyType type of dependency (startup, runtime, configuration).

	// +kubebuilder:validation:Enum=startup;runtime;configuration;data.

	DependencyType string `json:"dependencyType"`

	// Required indicates if this dependency is mandatory.

	Required bool `json:"required"`

	// WaitConditions specific conditions to wait for.

	WaitConditions []WaitCondition `json:"waitConditions,omitempty"`
}

// WaitCondition defines conditions to wait for in dependencies.

type WaitCondition struct {
	// Type of condition (ready, healthy, configured).

	// +kubebuilder:validation:Enum=ready;healthy;configured;available.

	Type string `json:"type"`

	// Timeout maximum time to wait for the condition.

	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// CheckInterval how often to check the condition.

	CheckIntervalSeconds int32 `json:"checkIntervalSeconds,omitempty"`
}

// PlanValidationResult contains validation results for a deployment plan.

type PlanValidationResult struct {
	// Valid indicates if the plan is valid overall.

	Valid bool `json:"valid"`

	// Errors list of validation errors.

	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings list of validation warnings.

	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// ResourceValidation results of resource validation.

	ResourceValidation *ResourceValidationResult `json:"resourceValidation,omitempty"`

	// SecurityValidation results of security validation.

	SecurityValidation *SecurityValidationResult `json:"securityValidation,omitempty"`

	// CompatibilityValidation results of compatibility validation.

	CompatibilityValidation *CompatibilityValidationResult `json:"compatibilityValidation,omitempty"`
}

// ValidationError represents a validation error.

type ValidationError struct {
	// Code error code.

	Code string `json:"code"`

	// Message error message.

	Message string `json:"message"`

	// Component affected component.

	Component string `json:"component,omitempty"`

	// Severity error severity.

	// +kubebuilder:validation:Enum=low;medium;high;critical.

	Severity string `json:"severity"`
}

// ValidationWarning represents a validation warning.

type ValidationWarning struct {
	// Code warning code.

	Code string `json:"code"`

	// Message warning message.

	Message string `json:"message"`

	// Component affected component.

	Component string `json:"component,omitempty"`

	// Recommendation suggested action.

	Recommendation string `json:"recommendation,omitempty"`
}

// ResourceValidationResult contains resource validation results.

type ResourceValidationResult struct {
	// SufficientResources indicates if cluster has sufficient resources.

	SufficientResources bool `json:"sufficientResources"`

	// AvailableResources currently available resources.

	AvailableResources map[string]string `json:"availableResources,omitempty"`

	// RequiredResources total required resources.

	RequiredResources map[string]string `json:"requiredResources,omitempty"`

	// ResourceGaps shortfall in resources.

	ResourceGaps map[string]string `json:"resourceGaps,omitempty"`
}

// SecurityValidationResult contains security validation results.

type SecurityValidationResult struct {
	// ComplianceLevel achieved compliance level.

	// +kubebuilder:validation:Enum=basic;standard;high;strict.

	ComplianceLevel string `json:"complianceLevel"`

	// SecurityPolicies validated security policies.

	SecurityPolicies []string `json:"securityPolicies,omitempty"`

	// MissingPolicies required but missing security policies.

	MissingPolicies []string `json:"missingPolicies,omitempty"`

	// RecommendedPolicies additional recommended policies.

	RecommendedPolicies []string `json:"recommendedPolicies,omitempty"`
}

// CompatibilityValidationResult contains compatibility validation results.

type CompatibilityValidationResult struct {
	// Compatible indicates overall compatibility.

	Compatible bool `json:"compatible"`

	// VersionCompatibility version compatibility matrix.

	VersionCompatibility map[string]map[string]bool `json:"versionCompatibility,omitempty"`

	// InterfaceCompatibility interface compatibility results.

	InterfaceCompatibility []InterfaceCompatibilityResult `json:"interfaceCompatibility,omitempty"`

	// IncompatiblePairs list of incompatible CNF pairs.

	IncompatiblePairs []CNFPair `json:"incompatiblePairs,omitempty"`
}

// InterfaceCompatibilityResult represents interface compatibility results.

type InterfaceCompatibilityResult struct {
	// SourceCNF source CNF function.

	SourceCNF CNFFunction `json:"sourceCnf"`

	// TargetCNF target CNF function.

	TargetCNF CNFFunction `json:"targetCnf"`

	// InterfaceName name of the interface.

	InterfaceName string `json:"interfaceName"`

	// Compatible indicates if interface is compatible.

	Compatible bool `json:"compatible"`

	// Issues list of compatibility issues.

	Issues []string `json:"issues,omitempty"`
}

// CNFPair represents a pair of CNF functions.

type CNFPair struct {
	// CNF1 first CNF function.

	CNF1 CNFFunction `json:"cnf1"`

	// CNF2 second CNF function.

	CNF2 CNFFunction `json:"cnf2"`

	// Reason reason for incompatibility.

	Reason string `json:"reason"`
}

// CNFIntentStatus represents enhanced status for CNF-related intents.

type CNFIntentStatus struct {
	// CNFProcessingPhase current phase of CNF processing.

	// +kubebuilder:validation:Enum=Detection;Processing;Planning;Validation;Deployment;Monitoring;Complete.

	CNFProcessingPhase string `json:"cnfProcessingPhase,omitempty"`

	// DetectedCNFs list of detected CNF functions.

	DetectedCNFs []CNFFunction `json:"detectedCnfs,omitempty"`

	// CreatedCNFDeployments list of created CNF deployment names.

	CreatedCNFDeployments []string `json:"createdCnfDeployments,omitempty"`

	// DeploymentPlanID reference to the deployment plan.

	DeploymentPlanID string `json:"deploymentPlanId,omitempty"`

	// CNFDeploymentStatuses status of individual CNF deployments.

	CNFDeploymentStatuses []IndividualCNFDeploymentStatus `json:"cnfDeploymentStatuses,omitempty"`

	// OverallHealthStatus overall health of deployed CNFs.

	// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;Unknown.

	OverallHealthStatus string `json:"overallHealthStatus,omitempty"`

	// ServiceMeshStatus status of service mesh integration.

	ServiceMeshStatus *ServiceMeshStatus `json:"serviceMeshStatus,omitempty"`

	// NetworkSliceStatus status of network slice configuration.

	NetworkSliceStatus *NetworkSliceStatus `json:"networkSliceStatus,omitempty"`

	// MonitoringStatus status of monitoring setup.

	MonitoringStatus *MonitoringStatus `json:"monitoringStatus,omitempty"`

	// LastCNFProcessingTime timestamp of last CNF processing.

	LastCNFProcessingTime *metav1.Time `json:"lastCnfProcessingTime,omitempty"`

	// CNFProcessingErrors any errors in CNF processing.

	CNFProcessingErrors []string `json:"cnfProcessingErrors,omitempty"`
}

// IndividualCNFDeploymentStatus represents the status of an individual CNF deployment.

type IndividualCNFDeploymentStatus struct {
	// Name of the CNF deployment.

	Name string `json:"name"`

	// Function CNF function type.

	Function CNFFunction `json:"function"`

	// Phase current deployment phase.

	// +kubebuilder:validation:Enum=Pending;Deploying;Running;Scaling;Upgrading;Failed;Terminating.

	Phase string `json:"phase"`

	// ReadyReplicas number of ready replicas.

	ReadyReplicas int32 `json:"readyReplicas"`

	// DesiredReplicas desired number of replicas.

	DesiredReplicas int32 `json:"desiredReplicas"`

	// HealthStatus health status of the CNF.

	// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;Unknown.

	HealthStatus string `json:"healthStatus"`

	// LastUpdateTime timestamp of last status update.

	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// ServiceEndpoints list of service endpoints.

	ServiceEndpoints []string `json:"serviceEndpoints,omitempty"`

	// ResourceUtilization current resource utilization.

<<<<<<< HEAD
	ResourceUtilization map[string]float64 `json:"resourceUtilization,omitempty"`
=======
	ResourceUtilization map[string]string `json:"resourceUtilization,omitempty"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// Errors any errors related to this CNF deployment.

	Errors []string `json:"errors,omitempty"`
}

// ServiceMeshStatus represents service mesh integration status.

type ServiceMeshStatus struct {
	// Enabled indicates if service mesh is enabled.

	Enabled bool `json:"enabled"`

	// Type service mesh type (istio, linkerd, consul).

	Type string `json:"type,omitempty"`

	// Status overall service mesh status.

	// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;NotConfigured.

	Status string `json:"status"`

	// MTLSEnabled indicates if mTLS is enabled.

	MTLSEnabled bool `json:"mtlsEnabled"`

	// ProxyStatus status of sidecar proxies.

	ProxyStatus map[string]string `json:"proxyStatus,omitempty"`

	// TrafficPolicies applied traffic policies.

	TrafficPolicies []string `json:"trafficPolicies,omitempty"`
}

// NetworkSliceStatus represents network slice configuration status.

type NetworkSliceStatus struct {
	// SliceID network slice identifier.

	SliceID string `json:"sliceId,omitempty"`

	// Status slice configuration status.

	// +kubebuilder:validation:Enum=Active;Inactive;Configuring;Failed.

	Status string `json:"status"`

	// ConfiguredCNFs CNFs configured for this slice.

	ConfiguredCNFs []string `json:"configuredCnfs,omitempty"`

	// IsolationLevel achieved isolation level.

	// +kubebuilder:validation:Enum=logical;physical;complete.

	IsolationLevel string `json:"isolationLevel,omitempty"`

	// QoSProfile applied QoS profile.

	QoSProfile string `json:"qosProfile,omitempty"`
}

// MonitoringStatus represents monitoring setup status.

type MonitoringStatus struct {
	// Enabled indicates if monitoring is enabled.

	Enabled bool `json:"enabled"`

	// PrometheusEnabled indicates if Prometheus monitoring is enabled.

	PrometheusEnabled bool `json:"prometheusEnabled"`

	// GrafanaEnabled indicates if Grafana dashboards are enabled.

	GrafanaEnabled bool `json:"grafanaEnabled"`

	// TracingEnabled indicates if distributed tracing is enabled.

	TracingEnabled bool `json:"tracingEnabled"`

	// AlertingEnabled indicates if alerting is enabled.

	AlertingEnabled bool `json:"alertingEnabled"`

	// MonitoredCNFs list of CNFs with monitoring configured.

	MonitoredCNFs []string `json:"monitoredCnfs,omitempty"`

	// ActiveAlerts list of active alerts.

	ActiveAlerts []string `json:"activeAlerts,omitempty"`
}

// CNFIntentAnnotations defines standard annotations for CNF intents.

const (

	// CNFIntentProcessedAnnotation indicates the intent has been processed for CNF deployment.

	CNFIntentProcessedAnnotation = "nephoran.com/cnf-intent-processed"

	// CNFDeploymentPlanAnnotation contains the deployment plan ID.

	CNFDeploymentPlanAnnotation = "nephoran.com/cnf-deployment-plan"

	// CNFProcessingVersionAnnotation indicates the version of processing pipeline used.

	CNFProcessingVersionAnnotation = "nephoran.com/cnf-processing-version"

	// CNFProcessingTimestampAnnotation timestamp when CNF processing completed.

	CNFProcessingTimestampAnnotation = "nephoran.com/cnf-processing-timestamp"

	// CNFConfidenceScoreAnnotation confidence score of CNF processing.

	CNFConfidenceScoreAnnotation = "nephoran.com/cnf-confidence-score"
)

// CNFIntentLabels defines standard labels for CNF intents.

const (

	// CNFIntentTypeLabel indicates the type of CNF deployment.

	CNFIntentTypeLabel = "nephoran.com/cnf-intent-type"

	// CNFDeploymentStrategyLabel indicates the preferred deployment strategy.

	CNFDeploymentStrategyLabel = "nephoran.com/cnf-deployment-strategy"

	// CNFServiceMeshLabel indicates if service mesh is required.

	CNFServiceMeshLabel = "nephoran.com/cnf-service-mesh"

	// CNFMonitoringLabel indicates if monitoring is required.

	CNFMonitoringLabel = "nephoran.com/cnf-monitoring"

	// CNFHighAvailabilityLabel indicates if high availability is required.

	CNFHighAvailabilityLabel = "nephoran.com/cnf-high-availability"

	// CNFNetworkSliceLabel indicates network slice requirements.

	CNFNetworkSliceLabel = "nephoran.com/cnf-network-slice"
)

func init() {
	SchemeBuilder.Register(&CNFIntentProcessingResult{})
}

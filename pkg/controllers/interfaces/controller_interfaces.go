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

// Package interfaces defines common interfaces and contracts for Nephoran controller implementations.

package interfaces

import (
	"context"
	"time"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ProcessingPhase represents the current phase of intent processing.

type ProcessingPhase string

const (

	// PhaseIntentReceived indicates that an intent has been received and is ready for processing.

	PhaseIntentReceived ProcessingPhase = "IntentReceived"

	// PhaseReceived holds phasereceived value.

	PhaseReceived ProcessingPhase = "Received" // Alias for PhaseIntentReceived

	// PhaseLLMProcessing holds phasellmprocessing value.

	PhaseLLMProcessing ProcessingPhase = "LLMProcessing"

	// PhaseResourcePlanning holds phaseresourceplanning value.

	PhaseResourcePlanning ProcessingPhase = "ResourcePlanning"

	// PhaseManifestGeneration holds phasemanifestgeneration value.

	PhaseManifestGeneration ProcessingPhase = "ManifestGeneration"

	// PhaseGitOpsCommit holds phasegitopscommit value.

	PhaseGitOpsCommit ProcessingPhase = "GitOpsCommit"

	// PhaseDeploymentVerification holds phasedeploymentverification value.

	PhaseDeploymentVerification ProcessingPhase = "DeploymentVerification"

	// PhaseCompleted holds phasecompleted value.

	PhaseCompleted ProcessingPhase = "Completed"

	// PhaseFailed holds phasefailed value.

	PhaseFailed ProcessingPhase = "Failed"
)

// ProcessingResult contains the outcome of a phase.

type ProcessingResult struct {
	Success bool `json:"success"`

	NextPhase ProcessingPhase `json:"nextPhase,omitempty"`

	Data map[string]interface{} `json:"data,omitempty"`

	Metrics map[string]float64 `json:"metrics,omitempty"`

	Events []ProcessingEvent `json:"events,omitempty"`

	RetryAfter *time.Duration `json:"retryAfter,omitempty"`

	ErrorMessage string `json:"errorMessage,omitempty"`

	ErrorCode string `json:"errorCode,omitempty"`
}

// ProcessingEvent represents an event during processing.

type ProcessingEvent struct {
	Timestamp time.Time `json:"timestamp"`

	EventType string `json:"eventType"`

	Message string `json:"message"`

	Data map[string]interface{} `json:"data,omitempty"`

	CorrelationID string `json:"correlationId"`
}

// PhaseStatus tracks individual phase progress.

type PhaseStatus struct {
	Phase ProcessingPhase `json:"phase"`

	Status string `json:"status"` // Pending, InProgress, Completed, Failed

	StartTime *metav1.Time `json:"startTime,omitempty"`

	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	RetryCount int32 `json:"retryCount"`

	LastError string `json:"lastError,omitempty"`

	Metrics map[string]float64 `json:"metrics,omitempty"`

	DependsOn []ProcessingPhase `json:"dependsOn,omitempty"`

	BlockedBy []ProcessingPhase `json:"blockedBy,omitempty"`

	CreatedResources []ResourceReference `json:"createdResources,omitempty"`
}

// ResourceReference represents a reference to a created Kubernetes resource.

type ResourceReference struct {
	APIVersion string `json:"apiVersion"`

	Kind string `json:"kind"`

	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`

	UID string `json:"uid,omitempty"`
}

// ProcessingContext holds context information for multi-phase processing.

type ProcessingContext struct {
	IntentID string `json:"intentId"`

	CorrelationID string `json:"correlationId"`

	StartTime time.Time `json:"startTime"`

	CurrentPhase ProcessingPhase `json:"currentPhase"`

	// Intent data.

	IntentType string `json:"intentType"`

	OriginalIntent string `json:"originalIntent"`

	ExtractedEntities map[string]interface{} `json:"extractedEntities,omitempty"`

	TelecomContext map[string]interface{} `json:"telecomContext,omitempty"`

	// Phase-specific data.

	LLMResponse map[string]interface{} `json:"llmResponse,omitempty"`

	ResourcePlan map[string]interface{} `json:"resourcePlan,omitempty"`

	GeneratedManifests map[string]string `json:"generatedManifests,omitempty"`

	GitCommitHash string `json:"gitCommitHash,omitempty"`

	DeploymentStatus map[string]interface{} `json:"deploymentStatus,omitempty"`

	// Performance tracking.

	PhaseMetrics map[ProcessingPhase]PhaseMetrics `json:"phaseMetrics,omitempty"`

	TotalMetrics map[string]float64 `json:"totalMetrics,omitempty"`
}

// PhaseMetrics tracks performance metrics for a processing phase.

type PhaseMetrics struct {
	Duration time.Duration `json:"duration"`

	CPUUsage float64 `json:"cpuUsage"`

	MemoryUsage int64 `json:"memoryUsage"`

	APICallCount int `json:"apiCallCount"`

	ErrorCount int `json:"errorCount"`

	CustomMetrics map[string]float64 `json:"customMetrics,omitempty"`
}

// PhaseController interface for all specialized controllers.

type PhaseController interface {

	// Core processing methods.

	ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase ProcessingPhase) (ProcessingResult, error)

	GetPhaseStatus(ctx context.Context, intentID string) (*PhaseStatus, error)

	HandlePhaseError(ctx context.Context, intentID string, err error) error

	// Dependency management.

	GetDependencies() []ProcessingPhase

	GetBlockedPhases() []ProcessingPhase

	// Lifecycle methods.

	SetupWithManager(mgr ctrl.Manager) error

	Start(ctx context.Context) error

	Stop(ctx context.Context) error

	// Health and metrics.

	GetHealthStatus(ctx context.Context) (HealthStatus, error)

	GetMetrics(ctx context.Context) (map[string]float64, error)
}

// HealthStatus represents the health status of a controller.

type HealthStatus struct {
	Status string `json:"status"` // Healthy, Degraded, Unhealthy

	Message string `json:"message"`

	LastChecked time.Time `json:"lastChecked"`

	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// IntentProcessor handles LLM-based intent interpretation.

type IntentProcessor interface {
	PhaseController

	// Specialized methods for LLM processing.

	ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*ProcessingResult, error)

	ValidateIntent(ctx context.Context, intent string) error

	EnhanceWithRAG(ctx context.Context, intent string) (map[string]interface{}, error)

	GetSupportedIntentTypes() []string
}

// ResourcePlanner handles resource planning and optimization.

type ResourcePlanner interface {
	PhaseController

	// Specialized methods for resource planning.

	PlanResources(ctx context.Context, llmResponse map[string]interface{}) (*ResourcePlan, error)

	OptimizeResourceAllocation(ctx context.Context, requirements *ResourceRequirements) (*OptimizedPlan, error)

	ValidateResourceConstraints(ctx context.Context, plan *ResourcePlan) error

	EstimateResourceCosts(ctx context.Context, plan *ResourcePlan) (*CostEstimate, error)
}

// ManifestGenerator handles Kubernetes manifest creation.

type ManifestGenerator interface {
	PhaseController

	// Specialized methods for manifest generation.

	GenerateManifests(ctx context.Context, resourcePlan *ResourcePlan) (map[string]string, error)

	ValidateManifests(ctx context.Context, manifests map[string]string) error

	OptimizeManifests(ctx context.Context, manifests map[string]string) (map[string]string, error)

	GetSupportedTemplates() []string
}

// GitOpsManager handles Git operations and deployment.

type GitOpsManager interface {
	PhaseController

	// Specialized methods for GitOps operations.

	CommitManifests(ctx context.Context, manifests map[string]string, metadata map[string]interface{}) (*GitCommitResult, error)

	CreateNephioPackage(ctx context.Context, manifests map[string]string) (*NephioPackage, error)

	ResolveConflicts(ctx context.Context, conflicts []GitConflict) error

	TrackDeployment(ctx context.Context, commitHash string) (*DeploymentProgress, error)
}

// DeploymentVerifier handles deployment validation and monitoring.

type DeploymentVerifier interface {
	PhaseController

	// Specialized methods for deployment verification.

	VerifyDeployment(ctx context.Context, deploymentRef *DeploymentReference) (*VerificationResult, error)

	ValidateResourceHealth(ctx context.Context, resources []ResourceReference) error

	CheckSLACompliance(ctx context.Context, slaRequirements *SLARequirements) (*ComplianceResult, error)

	GenerateComplianceReport(ctx context.Context, deploymentID string) (*ComplianceReport, error)
}

// Supporting data structures.

// ResourcePlan defines the comprehensive resource planning structure for network function deployment.

type ResourcePlan struct {
	NetworkFunctions []PlannedNetworkFunction `json:"networkFunctions"`

	ResourceRequirements ResourceRequirements `json:"resourceRequirements"`

	DeploymentPattern string `json:"deploymentPattern"`

	EstimatedCost *CostEstimate `json:"estimatedCost,omitempty"`

	Constraints []ResourceConstraint `json:"constraints,omitempty"`

	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// PlannedNetworkFunction represents a network function that has been planned for deployment.

type PlannedNetworkFunction struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Image string `json:"image"`

	Version string `json:"version"`

	Resources ResourceSpec `json:"resources"`

	Configuration map[string]interface{} `json:"configuration"`

	Replicas int32 `json:"replicas"`

	Ports []PortSpec `json:"ports"`

	Environment []EnvVar `json:"environment,omitempty"`
}

// ResourceRequirements specifies the CPU, memory, and storage requirements for a resource.

type ResourceRequirements struct {
	CPU string `json:"cpu"`

	Memory string `json:"memory"`

	Storage string `json:"storage"`

	NetworkBandwidth string `json:"networkBandwidth,omitempty"`
}

// ResourceSpec defines both resource requests and limits for a workload.

type ResourceSpec struct {
	Requests ResourceRequirements `json:"requests"`

	Limits ResourceRequirements `json:"limits"`
}

// OptimizedPlan contains both the original and optimized resource plans with optimization details.

type OptimizedPlan struct {
	OriginalPlan *ResourcePlan `json:"originalPlan"`

	OptimizedPlan *ResourcePlan `json:"optimizedPlan"`

	Optimizations []Optimization `json:"optimizations"`

	CostSavings *CostComparison `json:"costSavings,omitempty"`
}

// CostEstimate provides detailed cost estimation for resource deployment.

type CostEstimate struct {
	TotalCost float64 `json:"totalCost"`

	Currency string `json:"currency"`

	BillingPeriod string `json:"billingPeriod"`

	CostBreakdown map[string]float64 `json:"costBreakdown"`

	EstimationDate time.Time `json:"estimationDate"`
}

// GitCommitResult contains information about a Git commit operation.

type GitCommitResult struct {
	CommitHash string `json:"commitHash"`

	CommitMessage string `json:"commitMessage"`

	Branch string `json:"branch"`

	Repository string `json:"repository"`

	Files []string `json:"files"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NephioPackage represents a Nephio package with its manifests and dependencies.

type NephioPackage struct {
	Name string `json:"name"`

	Version string `json:"version"`

	Dependencies []string `json:"dependencies"`

	Manifests map[string]string `json:"manifests"`

	Metadata map[string]interface{} `json:"metadata"`
}

// DeploymentProgress tracks the progress of a network function deployment.

type DeploymentProgress struct {
	Status string `json:"status"`

	Progress float64 `json:"progress"` // 0-100

	DeployedResources []ResourceReference `json:"deployedResources"`

	FailedResources []ResourceReference `json:"failedResources"`

	Messages []string `json:"messages"`

	LastUpdate time.Time `json:"lastUpdate"`
}

// DeploymentReference provides reference information for a deployment.

type DeploymentReference struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Labels map[string]string `json:"labels,omitempty"`

	CommitHash string `json:"commitHash"`
}

// VerificationResult contains the results of deployment verification and health checks.

type VerificationResult struct {
	Success bool `json:"success"`

	HealthyResources []ResourceReference `json:"healthyResources"`

	UnhealthyResources []ResourceReference `json:"unhealthyResources"`

	Warnings []string `json:"warnings"`

	Errors []string `json:"errors"`

	SLACompliance *ComplianceResult `json:"slaCompliance,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// SLARequirements defines service level agreement targets for network functions.

type SLARequirements struct {
	AvailabilityTarget float64 `json:"availabilityTarget"`

	ResponseTimeTarget time.Duration `json:"responseTimeTarget"`

	ThroughputTarget float64 `json:"throughputTarget"`

	ErrorRateTarget float64 `json:"errorRateTarget"`

	CustomMetrics map[string]float64 `json:"customMetrics,omitempty"`
}

// ComplianceResult contains the evaluation results against SLA requirements.

type ComplianceResult struct {
	Overall string `json:"overall"` // Compliant, NonCompliant, Unknown

	Availability float64 `json:"availability"`

	ResponseTime time.Duration `json:"responseTime"`

	Throughput float64 `json:"throughput"`

	ErrorRate float64 `json:"errorRate"`

	Details map[string]interface{} `json:"details"`

	Violations []ComplianceViolation `json:"violations,omitempty"`
}

// ComplianceReport provides a comprehensive compliance assessment report.

type ComplianceReport struct {
	ReportID string `json:"reportId"`

	GeneratedAt time.Time `json:"generatedAt"`

	DeploymentID string `json:"deploymentId"`

	SLARequirements *SLARequirements `json:"slaRequirements"`

	ComplianceResult *ComplianceResult `json:"complianceResult"`

	Recommendations []string `json:"recommendations,omitempty"`

	NextReviewDate time.Time `json:"nextReviewDate"`
}

// Additional supporting types.

// ResourceConstraint defines a constraint that must be satisfied during resource planning.

type ResourceConstraint struct {
	Type string `json:"type"`

	Resource string `json:"resource"`

	Operator string `json:"operator"`

	Value interface{} `json:"value"`

	Message string `json:"message,omitempty"`
}

// Dependency represents a dependency requirement for a network function.

type Dependency struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Version string `json:"version,omitempty"`

	Required bool `json:"required"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// PortSpec defines a network port configuration for a service.

type PortSpec struct {
	Name string `json:"name"`

	Port int32 `json:"port"`

	TargetPort int32 `json:"targetPort,omitempty"`

	Protocol string `json:"protocol,omitempty"`

	ServiceType string `json:"serviceType,omitempty"`
}

// EnvVar represents an environment variable for a container.

type EnvVar struct {
	Name string `json:"name"`

	Value string `json:"value,omitempty"`

	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource defines sources for environment variable values.

type EnvVarSource struct {
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`

	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`

	FieldRef *FieldSelector `json:"fieldRef,omitempty"`
}

// ConfigMapKeySelector selects a key from a ConfigMap.

type ConfigMapKeySelector struct {
	Name string `json:"name"`

	Key string `json:"key"`
}

// SecretKeySelector selects a key from a Secret.

type SecretKeySelector struct {
	Name string `json:"name"`

	Key string `json:"key"`
}

// FieldSelector selects a field from the pod spec.

type FieldSelector struct {
	FieldPath string `json:"fieldPath"`
}

// Optimization describes a performance or cost optimization that was applied.

type Optimization struct {
	Type string `json:"type"`

	Description string `json:"description"`

	Impact string `json:"impact"`

	Savings map[string]interface{} `json:"savings,omitempty"`
}

// CostComparison provides a comparison between original and optimized costs.

type CostComparison struct {
	OriginalCost float64 `json:"originalCost"`

	OptimizedCost float64 `json:"optimizedCost"`

	SavingsAmount float64 `json:"savingsAmount"`

	SavingsPercent float64 `json:"savingsPercent"`
}

// GitConflict represents a Git merge conflict that occurred during operations.

type GitConflict struct {
	File string `json:"file"`

	ConflictType string `json:"conflictType"`

	LocalContent string `json:"localContent"`

	RemoteContent string `json:"remoteContent"`

	Resolution string `json:"resolution,omitempty"`
}

// ComplianceViolation represents a specific compliance rule violation.

type ComplianceViolation struct {
	Type string `json:"type"`

	Severity string `json:"severity"`

	Description string `json:"description"`

	Resource string `json:"resource,omitempty"`

	Value interface{} `json:"value"`

	Threshold interface{} `json:"threshold"`

	Timestamp time.Time `json:"timestamp"`
}

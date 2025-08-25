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

package nephio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// NephioWorkflowOrchestrator provides native Nephio R5 workflow orchestration
// Implements standard Nephio patterns for GitOps package management
type NephioWorkflowOrchestrator struct {
	client           client.Client
	porchClient      porch.PorchClient
	configSync       *ConfigSyncClient
	workloadRegistry *WorkloadClusterRegistry
	packageCatalog   *NephioPackageCatalog
	workflowEngine   *NephioWorkflowEngine
	metrics          *WorkflowMetrics
	tracer           trace.Tracer
	config           *WorkflowOrchestratorConfig
	executionCache   sync.Map
	mu               sync.RWMutex
}

// WorkflowOrchestratorConfig defines configuration for workflow orchestrator
type WorkflowOrchestratorConfig struct {
	// Repository settings
	UpstreamRepository   string `json:"upstreamRepository" yaml:"upstreamRepository"`
	DownstreamRepository string `json:"downstreamRepository" yaml:"downstreamRepository"`
	CatalogRepository    string `json:"catalogRepository" yaml:"catalogRepository"`
	DefaultNamespace     string `json:"defaultNamespace" yaml:"defaultNamespace"`

	// Workflow execution settings
	MaxConcurrentWorkflows int           `json:"maxConcurrentWorkflows" yaml:"maxConcurrentWorkflows"`
	WorkflowTimeout        time.Duration `json:"workflowTimeout" yaml:"workflowTimeout"`
	StageTimeout           time.Duration `json:"stageTimeout" yaml:"stageTimeout"`

	// GitOps settings
	GitBranch         string `json:"gitBranch" yaml:"gitBranch"`
	AutoApproval      bool   `json:"autoApproval" yaml:"autoApproval"`
	RequiredApprovers int    `json:"requiredApprovers" yaml:"requiredApprovers"`

	// Config Sync settings
	ConfigSyncEnabled bool          `json:"configSyncEnabled" yaml:"configSyncEnabled"`
	SyncTimeout       time.Duration `json:"syncTimeout" yaml:"syncTimeout"`
	RollbackOnFailure bool          `json:"rollbackOnFailure" yaml:"rollbackOnFailure"`

	// Cluster management settings
	AutoClusterRegistration bool          `json:"autoClusterRegistration" yaml:"autoClusterRegistration"`
	HealthCheckInterval     time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`

	// Monitoring settings
	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`
}

// WorkflowMetrics provides comprehensive workflow metrics
type WorkflowMetrics struct {
	WorkflowExecutions   prometheus.CounterVec
	WorkflowDuration     prometheus.HistogramVec
	WorkflowPhases       prometheus.GaugeVec
	PackageVariants      prometheus.CounterVec
	ClusterDeployments   prometheus.CounterVec
	ConfigSyncOperations prometheus.CounterVec
	WorkflowErrors       prometheus.CounterVec
}

// WorkflowExecution represents a Nephio workflow execution
type WorkflowExecution struct {
	ID               string                    `json:"id"`
	Intent           *v1.NetworkIntent         `json:"intent"`
	WorkflowDef      *WorkflowDefinition       `json:"workflowDef"`
	Status           WorkflowExecutionStatus   `json:"status"`
	Phases           []WorkflowPhaseExecution  `json:"phases"`
	BlueprintPackage *BlueprintPackage         `json:"blueprintPackage,omitempty"`
	PackageVariants  []*PackageVariant         `json:"packageVariants,omitempty"`
	Deployments      []*WorkloadDeployment     `json:"deployments,omitempty"`
	Results          *WorkflowExecutionResults `json:"results,omitempty"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
	CompletedAt      *time.Time                `json:"completedAt,omitempty"`
}

// WorkflowExecutionStatus represents the status of workflow execution
type WorkflowExecutionStatus string

const (
	WorkflowExecutionStatusPending   WorkflowExecutionStatus = "Pending"
	WorkflowExecutionStatusRunning   WorkflowExecutionStatus = "Running"
	WorkflowExecutionStatusCompleted WorkflowExecutionStatus = "Completed"
	WorkflowExecutionStatusFailed    WorkflowExecutionStatus = "Failed"
	WorkflowExecutionStatusRollback  WorkflowExecutionStatus = "Rollback"
	WorkflowExecutionStatusCancelled WorkflowExecutionStatus = "Cancelled"
)

// WorkflowPhaseExecution represents execution of a workflow phase
type WorkflowPhaseExecution struct {
	Name        string                  `json:"name"`
	Status      WorkflowExecutionStatus `json:"status"`
	StartedAt   *time.Time              `json:"startedAt,omitempty"`
	CompletedAt *time.Time              `json:"completedAt,omitempty"`
	Duration    time.Duration           `json:"duration"`
	Results     map[string]interface{}  `json:"results,omitempty"`
	Errors      []string                `json:"errors,omitempty"`
	Conditions  []WorkflowCondition     `json:"conditions,omitempty"`
}

// WorkflowExecutionResults contains comprehensive workflow results
type WorkflowExecutionResults struct {
	PackageRevisions  []*porch.PackageRevision  `json:"packageRevisions"`
	DeploymentResults []*DeploymentResult       `json:"deploymentResults"`
	ValidationResults []*ValidationResult       `json:"validationResults"`
	ConfigSyncResults []*ConfigSyncResult       `json:"configSyncResults"`
	ClusterHealth     map[string]*ClusterHealth `json:"clusterHealth"`
	ResourceUsage     *ResourceUsageSummary     `json:"resourceUsage"`
	ComplianceResults *ComplianceResults        `json:"complianceResults"`
}

// ConfigSyncClient manages Config Sync operations
type ConfigSyncClient struct {
	client      client.Client
	gitClient   GitClientInterface
	syncService *SyncService
	config      *ConfigSyncConfig
	tracer      trace.Tracer
	metrics     *ConfigSyncMetrics
}

// ConfigSyncConfig defines Config Sync configuration
type ConfigSyncConfig struct {
	Repository  string          `json:"repository" yaml:"repository"`
	Branch      string          `json:"branch" yaml:"branch"`
	Directory   string          `json:"directory" yaml:"directory"`
	SyncPeriod  time.Duration   `json:"syncPeriod" yaml:"syncPeriod"`
	Credentials *GitCredentials `json:"credentials" yaml:"credentials"`
	PolicyDir   string          `json:"policyDir" yaml:"policyDir"`
	Namespaces  []string        `json:"namespaces" yaml:"namespaces"`
}

// GitClientInterface defines Git operations for Config Sync
type GitClientInterface interface {
	Clone(ctx context.Context, repo string, branch string, dir string) error
	Commit(ctx context.Context, message string, files []string) error
	Push(ctx context.Context) error
	Pull(ctx context.Context) error
	CreateBranch(ctx context.Context, branch string) error
	MergeBranch(ctx context.Context, source string, target string) error
	GetCommitHash(ctx context.Context) (string, error)
}

// SyncService manages package synchronization
type SyncService struct {
	client       client.Client
	configSync   *ConfigSyncConfig
	tracer       trace.Tracer
	pollInterval time.Duration
}

// WorkloadClusterRegistry manages workload cluster lifecycle
type WorkloadClusterRegistry struct {
	client        client.Client
	clusters      sync.Map
	healthMonitor *ClusterHealthMonitor
	registrations sync.Map
	metrics       *ClusterRegistryMetrics
	tracer        trace.Tracer
	config        *WorkloadClusterConfig
}

// WorkloadCluster represents a registered workload cluster
type WorkloadCluster struct {
	Name            string                `json:"name"`
	Endpoint        string                `json:"endpoint"`
	Region          string                `json:"region"`
	Zone            string                `json:"zone"`
	Capabilities    []ClusterCapability   `json:"capabilities"`
	Status          WorkloadClusterStatus `json:"status"`
	Health          *ClusterHealth        `json:"health"`
	ConfigSync      *ClusterConfigSync    `json:"configSync"`
	Resources       *ClusterResources     `json:"resources"`
	Labels          map[string]string     `json:"labels"`
	Annotations     map[string]string     `json:"annotations"`
	CreatedAt       time.Time             `json:"createdAt"`
	LastHealthCheck *time.Time            `json:"lastHealthCheck,omitempty"`
}

// ClusterCapability defines cluster capabilities
type ClusterCapability struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Status  string                 `json:"status"`
}

// WorkloadClusterStatus represents cluster status
type WorkloadClusterStatus string

const (
	WorkloadClusterStatusRegistering WorkloadClusterStatus = "Registering"
	WorkloadClusterStatusActive      WorkloadClusterStatus = "Active"
	WorkloadClusterStatusDraining    WorkloadClusterStatus = "Draining"
	WorkloadClusterStatusMaintenance WorkloadClusterStatus = "Maintenance"
	WorkloadClusterStatusUnreachable WorkloadClusterStatus = "Unreachable"
	WorkloadClusterStatusTerminating WorkloadClusterStatus = "Terminating"
)

// NephioPackageCatalog manages blueprint catalog and package variants
type NephioPackageCatalog struct {
	client       client.Client
	repositories sync.Map
	blueprints   sync.Map
	variants     sync.Map
	templates    sync.Map
	metrics      *PackageCatalogMetrics
	tracer       trace.Tracer
	config       *PackageCatalogConfig
}

// BlueprintPackage represents a blueprint package in the catalog
type BlueprintPackage struct {
	Name           string                 `json:"name"`
	Repository     string                 `json:"repository"`
	Version        string                 `json:"version"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"`
	IntentTypes    []string `json:"intentTypes"`
	Dependencies   []PackageDependency    `json:"dependencies"`
	Parameters     []ParameterDefinition  `json:"parameters"`
	Validations    []ValidationRule       `json:"validations"`
	Resources      []ResourceTemplate     `json:"resources"`
	Functions      []FunctionDefinition   `json:"functions"`
	Specialization *SpecializationSpec    `json:"specialization,omitempty"`
	Metadata       map[string]string      `json:"metadata"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// PackageVariant represents a specialized package for a target cluster
type PackageVariant struct {
	Name              string                 `json:"name"`
	Blueprint         *BlueprintPackage      `json:"blueprint"`
	TargetCluster     *WorkloadCluster       `json:"targetCluster"`
	Specialization    *SpecializationRequest `json:"specialization"`
	Status            PackageVariantStatus   `json:"status"`
	PackageRevision   *porch.PackageRevision `json:"packageRevision"`
	DeploymentStatus  *DeploymentStatus      `json:"deploymentStatus,omitempty"`
	ValidationResults []*ValidationResult    `json:"validationResults,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// SpecializationRequest defines how to specialize a blueprint
type SpecializationRequest struct {
	ClusterContext    *ClusterContext        `json:"clusterContext"`
	Parameters        map[string]interface{} `json:"parameters"`
	ResourceOverrides map[string]interface{} `json:"resourceOverrides,omitempty"`
	NetworkSlice      *NetworkSliceSpec      `json:"networkSlice,omitempty"`
	ORANCompliance    *ORANComplianceSpec    `json:"oranCompliance,omitempty"`
	SecurityPolicy    *SecurityPolicySpec    `json:"securityPolicy,omitempty"`
	PlacementPolicy   *PlacementPolicySpec   `json:"placementPolicy,omitempty"`
}

// PackageVariantStatus represents variant status
type PackageVariantStatus string

const (
	PackageVariantStatusSpecializing PackageVariantStatus = "Specializing"
	PackageVariantStatusReady        PackageVariantStatus = "Ready"
	PackageVariantStatusDeploying    PackageVariantStatus = "Deploying"
	PackageVariantStatusDeployed     PackageVariantStatus = "Deployed"
	PackageVariantStatusFailed       PackageVariantStatus = "Failed"
	PackageVariantStatusTerminating  PackageVariantStatus = "Terminating"
)

// NephioWorkflowEngine manages workflow definitions and execution
type NephioWorkflowEngine struct {
	workflows sync.Map
	executor  *WorkflowExecutor
	validator *WorkflowValidator
	metrics   *WorkflowEngineMetrics
	tracer    trace.Tracer
	config    *WorkflowEngineConfig
}

// WorkflowDefinition defines a Nephio workflow
type WorkflowDefinition struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	IntentTypes   []string `json:"intentTypes"`
	Phases        []WorkflowPhase        `json:"phases"`
	Conditions    []WorkflowCondition    `json:"conditions"`
	Rollback      *RollbackStrategy      `json:"rollback,omitempty"`
	Approvals     *ApprovalStrategy      `json:"approvals,omitempty"`
	Notifications *NotificationStrategy  `json:"notifications,omitempty"`
	Timeouts      *TimeoutStrategy       `json:"timeouts,omitempty"`
	Metadata      map[string]string      `json:"metadata"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// WorkflowPhase represents a phase in the workflow
type WorkflowPhase struct {
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Type            WorkflowPhaseType   `json:"type"`
	Dependencies    []string            `json:"dependencies"`
	Conditions      []WorkflowCondition `json:"conditions"`
	Actions         []WorkflowAction    `json:"actions"`
	Timeout         time.Duration       `json:"timeout"`
	RetryPolicy     *RetryPolicy        `json:"retryPolicy,omitempty"`
	ContinueOnError bool                `json:"continueOnError"`
}

// WorkflowPhaseType defines the type of workflow phase
type WorkflowPhaseType string

const (
	WorkflowPhaseTypeBlueprintSelection    WorkflowPhaseType = "BlueprintSelection"
	WorkflowPhaseTypePackageSpecialization WorkflowPhaseType = "PackageSpecialization"
	WorkflowPhaseTypeValidation            WorkflowPhaseType = "Validation"
	WorkflowPhaseTypeApproval              WorkflowPhaseType = "Approval"
	WorkflowPhaseTypeDeployment            WorkflowPhaseType = "Deployment"
	WorkflowPhaseTypeMonitoring            WorkflowPhaseType = "Monitoring"
	WorkflowPhaseTypeRollback              WorkflowPhaseType = "Rollback"
)

// WorkflowAction defines an action to execute in a phase
type WorkflowAction struct {
	Name        string                 `json:"name"`
	Type        WorkflowActionType     `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Required    bool                   `json:"required"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy *RetryPolicy           `json:"retryPolicy,omitempty"`
}

// WorkflowActionType defines the type of workflow action
type WorkflowActionType string

const (
	WorkflowActionTypeCreatePackageRevision WorkflowActionType = "CreatePackageRevision"
	WorkflowActionTypeSpecializePackage     WorkflowActionType = "SpecializePackage"
	WorkflowActionTypeValidatePackage       WorkflowActionType = "ValidatePackage"
	WorkflowActionTypeApprovePackage        WorkflowActionType = "ApprovePackage"
	WorkflowActionTypeDeployToCluster       WorkflowActionType = "DeployToCluster"
	WorkflowActionTypeWaitForDeployment     WorkflowActionType = "WaitForDeployment"
	WorkflowActionTypeVerifyHealth          WorkflowActionType = "VerifyHealth"
	WorkflowActionTypeNotifyUsers           WorkflowActionType = "NotifyUsers"
)

// WorkflowCondition defines a condition for workflow execution
type WorkflowCondition struct {
	Name       string                 `json:"name"`
	Type       WorkflowConditionType  `json:"type"`
	Expression string                 `json:"expression"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Required   bool                   `json:"required"`
}

// WorkflowConditionType defines the type of workflow condition
type WorkflowConditionType string

const (
	WorkflowConditionTypeIntentMatch       WorkflowConditionType = "IntentMatch"
	WorkflowConditionTypeClusterAvailable  WorkflowConditionType = "ClusterAvailable"
	WorkflowConditionTypeResourceAvailable WorkflowConditionType = "ResourceAvailable"
	WorkflowConditionTypeApprovalReceived  WorkflowConditionType = "ApprovalReceived"
	WorkflowConditionTypeHealthCheckPassed WorkflowConditionType = "HealthCheckPassed"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int           `json:"maxAttempts"`
	Delay       time.Duration `json:"delay"`
	BackoffRate float64       `json:"backoffRate"`
	MaxDelay    time.Duration `json:"maxDelay"`
}

// Default configuration
var DefaultWorkflowOrchestratorConfig = &WorkflowOrchestratorConfig{
	UpstreamRepository:      "nephoran-blueprints",
	DownstreamRepository:    "nephoran-deployments",
	CatalogRepository:       "nephoran-catalog",
	DefaultNamespace:        "nephoran-system",
	MaxConcurrentWorkflows:  10,
	WorkflowTimeout:         30 * time.Minute,
	StageTimeout:            10 * time.Minute,
	GitBranch:               "main",
	AutoApproval:            false,
	RequiredApprovers:       1,
	ConfigSyncEnabled:       true,
	SyncTimeout:             5 * time.Minute,
	RollbackOnFailure:       true,
	AutoClusterRegistration: true,
	HealthCheckInterval:     1 * time.Minute,
	EnableMetrics:           true,
	EnableTracing:           true,
}

// NewNephioWorkflowOrchestrator creates a new Nephio workflow orchestrator
func NewNephioWorkflowOrchestrator(
	client client.Client,
	porchClient porch.PorchClient,
	config *WorkflowOrchestratorConfig,
) (*NephioWorkflowOrchestrator, error) {
	if config == nil {
		config = DefaultWorkflowOrchestratorConfig
	}

	// Initialize metrics
	metrics := &WorkflowMetrics{
		WorkflowExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_workflow_executions_total",
				Help: "Total number of Nephio workflow executions",
			},
			[]string{"workflow", "intent_type", "status"},
		),
		WorkflowDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_workflow_duration_seconds",
				Help:    "Duration of Nephio workflow executions",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"workflow", "phase"},
		),
		WorkflowPhases: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_workflow_phases",
				Help: "Current workflow phases in progress",
			},
			[]string{"workflow", "phase", "status"},
		),
		PackageVariants: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_package_variants_total",
				Help: "Total number of package variants created",
			},
			[]string{"blueprint", "cluster", "status"},
		),
		ClusterDeployments: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_cluster_deployments_total",
				Help: "Total number of cluster deployments",
			},
			[]string{"cluster", "package", "status"},
		),
		ConfigSyncOperations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_configsync_operations_total",
				Help: "Total number of Config Sync operations",
			},
			[]string{"operation", "cluster", "status"},
		),
		WorkflowErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_workflow_errors_total",
				Help: "Total number of workflow errors",
			},
			[]string{"workflow", "phase", "error_type"},
		),
	}

	// Initialize Config Sync client
	configSyncConfig := &ConfigSyncConfig{
		Repository: config.DownstreamRepository,
		Branch:     config.GitBranch,
		Directory:  "clusters",
		SyncPeriod: 30 * time.Second,
		PolicyDir:  "system",
		Namespaces: []string{config.DefaultNamespace},
	}

	configSync := &ConfigSyncClient{
		client: client,
		config: configSyncConfig,
		tracer: otel.Tracer("nephio-configsync"),
	}

	// Initialize workload cluster registry
	workloadRegistry := &WorkloadClusterRegistry{
		client: client,
		config: &WorkloadClusterConfig{
			HealthCheckInterval:     config.HealthCheckInterval,
			AutoClusterRegistration: config.AutoClusterRegistration,
		},
		tracer: otel.Tracer("nephio-workload-registry"),
	}

	// Initialize package catalog
	packageCatalog := &NephioPackageCatalog{
		client: client,
		config: &PackageCatalogConfig{
			CatalogRepository:  config.CatalogRepository,
			BlueprintDirectory: "blueprints",
			VariantDirectory:   "variants",
		},
		tracer: otel.Tracer("nephio-package-catalog"),
	}

	// Initialize workflow engine
	workflowEngine := &NephioWorkflowEngine{
		config: &WorkflowEngineConfig{
			MaxConcurrentWorkflows: config.MaxConcurrentWorkflows,
			DefaultTimeout:         config.WorkflowTimeout,
			StageTimeout:           config.StageTimeout,
		},
		tracer: otel.Tracer("nephio-workflow-engine"),
	}

	nwo := &NephioWorkflowOrchestrator{
		client:           client,
		porchClient:      porchClient,
		configSync:       configSync,
		workloadRegistry: workloadRegistry,
		packageCatalog:   packageCatalog,
		workflowEngine:   workflowEngine,
		metrics:          metrics,
		tracer:           otel.Tracer("nephio-workflow-orchestrator"),
		config:           config,
	}

	// Register standard Nephio workflows
	if err := nwo.registerStandardWorkflows(); err != nil {
		return nil, fmt.Errorf("failed to register standard workflows: %w", err)
	}

	return nwo, nil
}

// ExecuteNephioWorkflow executes a complete Nephio workflow for a NetworkIntent
func (nwo *NephioWorkflowOrchestrator) ExecuteNephioWorkflow(ctx context.Context, intent *v1.NetworkIntent) (*WorkflowExecution, error) {
	ctx, span := nwo.tracer.Start(ctx, "execute-nephio-workflow")
	defer span.End()

	logger := log.FromContext(ctx).WithName("nephio-workflow").WithValues(
		"intent", intent.Name,
		"namespace", intent.Namespace,
		"description", intent.Spec.Intent,
	)

	span.SetAttributes(
		attribute.String("intent.name", intent.Name),
		attribute.String("intent.namespace", intent.Namespace),
		attribute.String("intent.description", intent.Spec.Intent),
	)

	startTime := time.Now()

	// Step 1: Select appropriate workflow
	workflow, err := nwo.selectWorkflow(ctx, intent)
	if err != nil {
		span.RecordError(err)
		return nil, errors.WithContext(err, "failed to select workflow")
	}

	// Step 2: Create workflow execution
	execution := &WorkflowExecution{
		ID:          generateExecutionID(),
		Intent:      intent,
		WorkflowDef: workflow,
		Status:      WorkflowExecutionStatusPending,
		Phases:      make([]WorkflowPhaseExecution, len(workflow.Phases)),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Initialize phase executions
	for i, phase := range workflow.Phases {
		execution.Phases[i] = WorkflowPhaseExecution{
			Name:   phase.Name,
			Status: WorkflowExecutionStatusPending,
		}
	}

	// Store in cache
	nwo.executionCache.Store(execution.ID, execution)

	logger.Info("Created workflow execution", "workflowId", execution.ID, "workflow", workflow.Name)

	// Step 3: Execute workflow asynchronously
	go func() {
		defer func() {
			duration := time.Since(startTime)
			nwo.metrics.WorkflowDuration.WithLabelValues(
				workflow.Name, "total",
			).Observe(duration.Seconds())

			nwo.metrics.WorkflowExecutions.WithLabelValues(
				workflow.Name,
				nwo.classifyIntentType(intent.Spec.Intent),
				string(execution.Status),
			).Inc()
		}()

		if err := nwo.executeWorkflowPhases(ctx, execution); err != nil {
			logger.Error(err, "Workflow execution failed", "workflowId", execution.ID)
			execution.Status = WorkflowExecutionStatusFailed
			nwo.metrics.WorkflowErrors.WithLabelValues(
				workflow.Name, "execution", "workflow_failed",
			).Inc()
		}

		execution.UpdatedAt = time.Now()
		if execution.Status == WorkflowExecutionStatusCompleted ||
			execution.Status == WorkflowExecutionStatusFailed {
			completedAt := time.Now()
			execution.CompletedAt = &completedAt
		}

		// Update cache
		nwo.executionCache.Store(execution.ID, execution)

		// Update intent status
		if err := nwo.updateIntentWithWorkflowStatus(ctx, intent, execution); err != nil {
			logger.Error(err, "Failed to update intent status", "intent", intent.Name)
		}
	}()

	span.SetAttributes(attribute.String("execution.id", execution.ID))
	return execution, nil
}

// selectWorkflow selects the appropriate workflow for an intent
func (nwo *NephioWorkflowOrchestrator) selectWorkflow(ctx context.Context, intent *v1.NetworkIntent) (*WorkflowDefinition, error) {
	ctx, span := nwo.tracer.Start(ctx, "select-workflow")
	defer span.End()

	// Find workflow that matches intent type
	var selectedWorkflow *WorkflowDefinition
	intentType := nwo.classifyIntentType(intent.Spec.Intent)

	nwo.workflowEngine.workflows.Range(func(key, value interface{}) bool {
		if workflow, ok := value.(*WorkflowDefinition); ok {
			for _, workflowIntentType := range workflow.IntentTypes {
				if workflowIntentType == intentType {
					selectedWorkflow = workflow
					return false // Stop iteration
				}
			}
		}
		return true
	})

	if selectedWorkflow == nil {
		return nil, fmt.Errorf("no workflow found for intent type: %s (derived from: %s)", intentType, intent.Spec.Intent)
	}

	span.SetAttributes(attribute.String("workflow.name", selectedWorkflow.Name))
	return selectedWorkflow, nil
}

// executeWorkflowPhases executes all phases of a workflow
func (nwo *NephioWorkflowOrchestrator) executeWorkflowPhases(ctx context.Context, execution *WorkflowExecution) error {
	ctx, span := nwo.tracer.Start(ctx, "execute-workflow-phases")
	defer span.End()

	logger := log.FromContext(ctx).WithName("workflow-phases").WithValues("executionId", execution.ID)

	execution.Status = WorkflowExecutionStatusRunning
	execution.UpdatedAt = time.Now()

	// Execute phases in order
	for i := range execution.Phases {
		phaseExecution := &execution.Phases[i]
		phaseDefinition := &execution.WorkflowDef.Phases[i]

		logger.Info("Executing workflow phase", "phase", phaseExecution.Name)

		if err := nwo.executeWorkflowPhase(ctx, execution, phaseExecution, phaseDefinition); err != nil {
			logger.Error(err, "Phase execution failed", "phase", phaseExecution.Name)
			phaseExecution.Status = WorkflowExecutionStatusFailed
			phaseExecution.Errors = append(phaseExecution.Errors, err.Error())

			if !phaseDefinition.ContinueOnError {
				execution.Status = WorkflowExecutionStatusFailed
				return errors.WithContext(err, fmt.Sprintf("phase %s failed", phaseExecution.Name))
			}
		} else {
			phaseExecution.Status = WorkflowExecutionStatusCompleted
			logger.Info("Phase execution completed", "phase", phaseExecution.Name)
		}

		completedAt := time.Now()
		phaseExecution.CompletedAt = &completedAt
		if phaseExecution.StartedAt != nil {
			phaseExecution.Duration = completedAt.Sub(*phaseExecution.StartedAt)
		}

		execution.UpdatedAt = time.Now()
		nwo.executionCache.Store(execution.ID, execution)
	}

	execution.Status = WorkflowExecutionStatusCompleted
	logger.Info("Workflow execution completed", "executionId", execution.ID)

	return nil
}

// executeWorkflowPhase executes a single workflow phase
func (nwo *NephioWorkflowOrchestrator) executeWorkflowPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "execute-workflow-phase")
	defer span.End()

	span.SetAttributes(
		attribute.String("phase.name", phaseDef.Name),
		attribute.String("phase.type", string(phaseDef.Type)),
	)

	phaseCtx, cancel := context.WithTimeout(ctx, phaseDef.Timeout)
	defer cancel()

	startTime := time.Now()
	phaseExec.StartedAt = &startTime
	phaseExec.Status = WorkflowExecutionStatusRunning

	// Update metrics
	nwo.metrics.WorkflowPhases.WithLabelValues(
		execution.WorkflowDef.Name, phaseDef.Name, string(phaseExec.Status),
	).Inc()

	defer func() {
		duration := time.Since(startTime)
		nwo.metrics.WorkflowDuration.WithLabelValues(
			execution.WorkflowDef.Name, phaseDef.Name,
		).Observe(duration.Seconds())

		nwo.metrics.WorkflowPhases.WithLabelValues(
			execution.WorkflowDef.Name, phaseDef.Name, string(phaseExec.Status),
		).Dec()
	}()

	// Execute phase based on type
	switch phaseDef.Type {
	case WorkflowPhaseTypeBlueprintSelection:
		return nwo.executeBlueprintSelectionPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypePackageSpecialization:
		return nwo.executePackageSpecializationPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypeValidation:
		return nwo.executeValidationPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypeApproval:
		return nwo.executeApprovalPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypeDeployment:
		return nwo.executeDeploymentPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypeMonitoring:
		return nwo.executeMonitoringPhase(phaseCtx, execution, phaseExec, phaseDef)
	case WorkflowPhaseTypeRollback:
		return nwo.executeRollbackPhase(phaseCtx, execution, phaseExec, phaseDef)
	default:
		return fmt.Errorf("unknown phase type: %s", phaseDef.Type)
	}
}

// executeBlueprintSelectionPhase executes blueprint selection phase
func (nwo *NephioWorkflowOrchestrator) executeBlueprintSelectionPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "blueprint-selection-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("blueprint-selection")

	// Find suitable blueprint for the intent
	blueprint, err := nwo.packageCatalog.FindBlueprintForIntent(ctx, execution.Intent)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to find blueprint for intent")
	}

	if blueprint == nil {
		return fmt.Errorf("no suitable blueprint found for intent type: %s", execution.Intent.Spec.IntentType)
	}

	execution.BlueprintPackage = blueprint
	phaseExec.Results = map[string]interface{}{
		"blueprint": map[string]interface{}{
			"name":       blueprint.Name,
			"repository": blueprint.Repository,
			"version":    blueprint.Version,
			"category":   blueprint.Category,
		},
	}

	logger.Info("Selected blueprint",
		"blueprint", blueprint.Name,
		"version", blueprint.Version,
		"repository", blueprint.Repository,
	)

	span.SetAttributes(
		attribute.String("blueprint.name", blueprint.Name),
		attribute.String("blueprint.version", blueprint.Version),
	)

	return nil
}

// executePackageSpecializationPhase executes package specialization phase
func (nwo *NephioWorkflowOrchestrator) executePackageSpecializationPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "package-specialization-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("package-specialization")

	if execution.BlueprintPackage == nil {
		return fmt.Errorf("no blueprint selected for specialization")
	}

	// Get target clusters for deployment
	clusters, err := nwo.getTargetClusters(ctx, execution.Intent)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to get target clusters")
	}

	execution.PackageVariants = make([]*PackageVariant, 0, len(clusters))

	// Create package variant for each target cluster
	for _, cluster := range clusters {
		variant, err := nwo.packageCatalog.CreatePackageVariant(ctx, execution.BlueprintPackage, &SpecializationRequest{
			ClusterContext: &ClusterContext{
				Name:         cluster.Name,
				Region:       cluster.Region,
				Zone:         cluster.Zone,
				Capabilities: cluster.Capabilities,
			},
			Parameters: nwo.extractParametersFromIntent(execution.Intent),
		})
		if err != nil {
			logger.Error(err, "Failed to create package variant", "cluster", cluster.Name)
			nwo.metrics.PackageVariants.WithLabelValues(
				execution.BlueprintPackage.Name, cluster.Name, "failed",
			).Inc()
			continue
		}

		execution.PackageVariants = append(execution.PackageVariants, variant)
		nwo.metrics.PackageVariants.WithLabelValues(
			execution.BlueprintPackage.Name, cluster.Name, "created",
		).Inc()

		logger.Info("Created package variant",
			"variant", variant.Name,
			"cluster", cluster.Name,
			"blueprint", execution.BlueprintPackage.Name,
		)
	}

	phaseExec.Results = map[string]interface{}{
		"variants": len(execution.PackageVariants),
		"clusters": len(clusters),
	}

	span.SetAttributes(
		attribute.Int("variants.created", len(execution.PackageVariants)),
		attribute.Int("clusters.targeted", len(clusters)),
	)

	return nil
}

// executeValidationPhase executes validation phase
func (nwo *NephioWorkflowOrchestrator) executeValidationPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "validation-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("validation")

	validationResults := make([]*ValidationResult, 0)
	validationErrors := make([]string, 0)

	// Validate each package variant
	for _, variant := range execution.PackageVariants {
		if variant.PackageRevision == nil {
			continue
		}

		result, err := nwo.porchClient.ValidatePackage(ctx,
			variant.PackageRevision.Spec.PackageName,
			variant.PackageRevision.Spec.Revision)
		if err != nil {
			logger.Error(err, "Package validation failed", "variant", variant.Name)
			validationErrors = append(validationErrors, fmt.Sprintf("validation failed for %s: %v", variant.Name, err))
			continue
		}

		validationResults = append(validationResults, &ValidationResult{
			PackageName: variant.PackageRevision.Spec.PackageName,
			Valid:       result.Valid,
			Errors:      result.Errors,
			Warnings:    result.Warnings,
		})

		if !result.Valid {
			validationErrors = append(validationErrors, fmt.Sprintf("package %s failed validation", variant.Name))
		}

		variant.ValidationResults = []*ValidationResult{{
			PackageName: variant.PackageRevision.Spec.PackageName,
			Valid:       result.Valid,
			Errors:      result.Errors,
			Warnings:    result.Warnings,
		}}
	}

	// Store results
	if execution.Results == nil {
		execution.Results = &WorkflowExecutionResults{}
	}
	execution.Results.ValidationResults = validationResults

	phaseExec.Results = map[string]interface{}{
		"totalValidations": len(validationResults),
		"passed":           len(validationResults) - len(validationErrors),
		"failed":           len(validationErrors),
	}

	if len(validationErrors) > 0 {
		phaseExec.Errors = validationErrors
		logger.Info("Validation phase completed with errors",
			"passed", len(validationResults)-len(validationErrors),
			"failed", len(validationErrors),
		)
	} else {
		logger.Info("Validation phase completed successfully", "validated", len(validationResults))
	}

	span.SetAttributes(
		attribute.Int("validations.total", len(validationResults)),
		attribute.Int("validations.passed", len(validationResults)-len(validationErrors)),
		attribute.Int("validations.failed", len(validationErrors)),
	)

	return nil
}

// executeApprovalPhase executes approval phase
func (nwo *NephioWorkflowOrchestrator) executeApprovalPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "approval-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("approval")

	// If auto-approval is enabled, approve automatically
	if nwo.config.AutoApproval {
		for _, variant := range execution.PackageVariants {
			if variant.PackageRevision != nil {
				if err := nwo.porchClient.ApprovePackageRevision(ctx,
					variant.PackageRevision.Spec.PackageName,
					variant.PackageRevision.Spec.Revision); err != nil {
					logger.Error(err, "Failed to approve package", "variant", variant.Name)
					continue
				}

				logger.Info("Auto-approved package", "variant", variant.Name)
			}
		}

		phaseExec.Results = map[string]interface{}{
			"approvalType": "automatic",
			"approved":     len(execution.PackageVariants),
		}

		return nil
	}

	// Manual approval process
	// In a real implementation, this would integrate with external approval systems
	logger.Info("Manual approval required", "variants", len(execution.PackageVariants))

	phaseExec.Results = map[string]interface{}{
		"approvalType": "manual",
		"status":       "pending",
		"required":     len(execution.PackageVariants),
	}

	// For now, we'll simulate approval
	// In production, this would wait for external approval
	return fmt.Errorf("manual approval not implemented - requires external approval system")
}

// executeDeploymentPhase executes deployment phase
func (nwo *NephioWorkflowOrchestrator) executeDeploymentPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "deployment-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("deployment")

	deploymentResults := make([]*DeploymentResult, 0)
	execution.Deployments = make([]*WorkloadDeployment, 0)

	// Deploy each package variant to its target cluster
	for _, variant := range execution.PackageVariants {
		if variant.PackageRevision == nil || variant.TargetCluster == nil {
			continue
		}

		deployment := &WorkloadDeployment{
			PackageVariant: variant,
			TargetCluster:  variant.TargetCluster,
			Status:         "Deploying",
			StartedAt:      time.Now(),
		}

		execution.Deployments = append(execution.Deployments, deployment)

		// Deploy via Config Sync
		syncResult, err := nwo.configSync.SyncPackageToCluster(ctx, variant.PackageRevision, variant.TargetCluster)
		if err != nil {
			logger.Error(err, "Failed to deploy package",
				"variant", variant.Name,
				"cluster", variant.TargetCluster.Name,
			)

			deployment.Status = "Failed"
			deployment.Error = err.Error()

			nwo.metrics.ClusterDeployments.WithLabelValues(
				variant.TargetCluster.Name, variant.Name, "failed",
			).Inc()
			continue
		}

		deployment.Status = "Deployed"
		deployment.SyncResult = syncResult
		completedAt := time.Now()
		deployment.CompletedAt = &completedAt

		deploymentResult := &DeploymentResult{
			PackageName: variant.Name,
			ClusterName: variant.TargetCluster.Name,
			Status:      "Success",
			SyncResult:  syncResult,
		}
		deploymentResults = append(deploymentResults, deploymentResult)

		nwo.metrics.ClusterDeployments.WithLabelValues(
			variant.TargetCluster.Name, variant.Name, "success",
		).Inc()

		logger.Info("Deployed package successfully",
			"variant", variant.Name,
			"cluster", variant.TargetCluster.Name,
		)
	}

	// Store results
	if execution.Results == nil {
		execution.Results = &WorkflowExecutionResults{}
	}
	execution.Results.DeploymentResults = deploymentResults

	phaseExec.Results = map[string]interface{}{
		"totalDeployments": len(execution.Deployments),
		"successful":       len(deploymentResults),
		"failed":           len(execution.Deployments) - len(deploymentResults),
	}

	logger.Info("Deployment phase completed",
		"successful", len(deploymentResults),
		"failed", len(execution.Deployments)-len(deploymentResults),
	)

	span.SetAttributes(
		attribute.Int("deployments.total", len(execution.Deployments)),
		attribute.Int("deployments.successful", len(deploymentResults)),
	)

	return nil
}

// executeMonitoringPhase executes monitoring phase
func (nwo *NephioWorkflowOrchestrator) executeMonitoringPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "monitoring-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("monitoring")

	// Verify deployment health across all clusters
	clusterHealthMap := make(map[string]*ClusterHealth)

	for _, deployment := range execution.Deployments {
		if deployment.Status != "Deployed" {
			continue
		}

		health, err := nwo.workloadRegistry.CheckClusterHealth(ctx, deployment.TargetCluster.Name)
		if err != nil {
			logger.Error(err, "Failed to check cluster health", "cluster", deployment.TargetCluster.Name)
			continue
		}

		clusterHealthMap[deployment.TargetCluster.Name] = health
	}

	// Store results
	if execution.Results == nil {
		execution.Results = &WorkflowExecutionResults{}
	}
	execution.Results.ClusterHealth = clusterHealthMap

	phaseExec.Results = map[string]interface{}{
		"clustersMonitored": len(clusterHealthMap),
		"healthyCount":      nwo.countHealthyClusters(clusterHealthMap),
	}

	logger.Info("Monitoring phase completed",
		"clustersMonitored", len(clusterHealthMap),
		"healthy", nwo.countHealthyClusters(clusterHealthMap),
	)

	return nil
}

// executeRollbackPhase executes rollback phase
func (nwo *NephioWorkflowOrchestrator) executeRollbackPhase(ctx context.Context, execution *WorkflowExecution, phaseExec *WorkflowPhaseExecution, phaseDef *WorkflowPhase) error {
	ctx, span := nwo.tracer.Start(ctx, "rollback-phase")
	defer span.End()

	logger := log.FromContext(ctx).WithName("rollback")

	// Rollback deployments in reverse order
	rollbackCount := 0
	for i := len(execution.Deployments) - 1; i >= 0; i-- {
		deployment := execution.Deployments[i]
		if deployment.Status != "Deployed" {
			continue
		}

		if err := nwo.rollbackDeployment(ctx, deployment); err != nil {
			logger.Error(err, "Failed to rollback deployment",
				"cluster", deployment.TargetCluster.Name,
				"package", deployment.PackageVariant.Name,
			)
			continue
		}

		rollbackCount++
		logger.Info("Rolled back deployment",
			"cluster", deployment.TargetCluster.Name,
			"package", deployment.PackageVariant.Name,
		)
	}

	phaseExec.Results = map[string]interface{}{
		"rolledBack": rollbackCount,
		"total":      len(execution.Deployments),
	}

	logger.Info("Rollback phase completed", "rolledBack", rollbackCount)

	return nil
}

// Helper methods

func generateExecutionID() string {
	return fmt.Sprintf("exec-%d", time.Now().UnixNano())
}

// classifyIntentType classifies the intent based on natural language keywords
func (nwo *NephioWorkflowOrchestrator) classifyIntentType(intent string) string {
	intentLower := strings.ToLower(intent)
	
	// Scaling intent patterns
	if strings.Contains(intentLower, "scale") || strings.Contains(intentLower, "autoscal") || 
		 strings.Contains(intentLower, "replicas") || strings.Contains(intentLower, "horizontal") {
		return "scaling"
	}
	
	// Configuration intent patterns  
	if strings.Contains(intentLower, "configur") || strings.Contains(intentLower, "network slice") ||
		 strings.Contains(intentLower, "policy") || strings.Contains(intentLower, "parameter") {
		return "configuration"
	}
	
	// Deployment intent patterns (default)
	if strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "install") ||
		 strings.Contains(intentLower, "create") || strings.Contains(intentLower, "setup") {
		return "deployment"
	}
	
	// Default to deployment if unclear
	return "deployment"
}

func (nwo *NephioWorkflowOrchestrator) getTargetClusters(ctx context.Context, intent *v1.NetworkIntent) ([]*WorkloadCluster, error) {
	// Extract target clusters from intent or use defaults
	// This would integrate with cluster selection policies
	clusters := make([]*WorkloadCluster, 0)

	// For now, return all active clusters
	nwo.workloadRegistry.clusters.Range(func(key, value interface{}) bool {
		if cluster, ok := value.(*WorkloadCluster); ok {
			if cluster.Status == WorkloadClusterStatusActive {
				clusters = append(clusters, cluster)
			}
		}
		return true
	})

	return clusters, nil
}

func (nwo *NephioWorkflowOrchestrator) extractParametersFromIntent(intent *v1.NetworkIntent) map[string]interface{} {
	// Extract configuration parameters from intent
	params := make(map[string]interface{})

	// Use the natural language intent as the primary parameter
	params["intentType"] = nwo.classifyIntentType(intent.Spec.Intent)
	params["intentDescription"] = intent.Spec.Intent
	params["intentName"] = intent.Name
	params["intentNamespace"] = intent.Namespace

	return params
}

func (nwo *NephioWorkflowOrchestrator) countHealthyClusters(healthMap map[string]*ClusterHealth) int {
	count := 0
	for _, health := range healthMap {
		if health.Status == "Healthy" {
			count++
		}
	}
	return count
}

func (nwo *NephioWorkflowOrchestrator) rollbackDeployment(ctx context.Context, deployment *WorkloadDeployment) error {
	// Implement rollback logic
	// This would typically involve removing the deployed resources
	return nil
}

func (nwo *NephioWorkflowOrchestrator) updateIntentWithWorkflowStatus(ctx context.Context, intent *v1.NetworkIntent, execution *WorkflowExecution) error {
	// Update NetworkIntent status with workflow execution results
	intent.Status.Phase = v1.NetworkIntentPhaseProcessing

	if execution.Status == WorkflowExecutionStatusCompleted {
		intent.Status.Phase = v1.NetworkIntentPhaseReady
	} else if execution.Status == WorkflowExecutionStatusFailed {
		intent.Status.Phase = v1.NetworkIntentPhaseFailed
	}

	// Update with package information
	if len(execution.PackageVariants) > 0 && execution.PackageVariants[0].PackageRevision != nil {
		pr := execution.PackageVariants[0].PackageRevision
		intent.Status.PackageRevision = &v1.PackageRevisionReference{
			Repository:  pr.Spec.Repository,
			PackageName: pr.Spec.PackageName,
			Revision:    pr.Spec.Revision,
		}
	}

	// Update deployment status
	if len(execution.Deployments) > 0 {
		intent.Status.DeploymentStatus = &v1.DeploymentStatus{
			Phase:   "Deployed",
			Targets: make([]v1.DeploymentTargetStatus, 0, len(execution.Deployments)),
		}

		for _, deployment := range execution.Deployments {
			intent.Status.DeploymentStatus.Targets = append(intent.Status.DeploymentStatus.Targets, v1.DeploymentTargetStatus{
				Cluster:   deployment.TargetCluster.Name,
				Namespace: nwo.config.DefaultNamespace,
				Status:    deployment.Status,
			})
		}
	}

	// Update timestamp
	now := metav1.NewTime(time.Now())
	intent.Status.LastProcessed = &now

	return nwo.client.Status().Update(ctx, intent)
}

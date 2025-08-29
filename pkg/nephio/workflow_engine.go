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

	"time"



	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/trace"



	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"



	"sigs.k8s.io/controller-runtime/pkg/log"

)



// WorkflowEngineConfig defines configuration for the workflow engine.

type WorkflowEngineConfig struct {

	MaxConcurrentWorkflows int           `json:"maxConcurrentWorkflows" yaml:"maxConcurrentWorkflows"`

	DefaultTimeout         time.Duration `json:"defaultTimeout" yaml:"defaultTimeout"`

	StageTimeout           time.Duration `json:"stageTimeout" yaml:"stageTimeout"`

	RetryAttempts          int           `json:"retryAttempts" yaml:"retryAttempts"`

	RetryBackoff           time.Duration `json:"retryBackoff" yaml:"retryBackoff"`

	EnableValidation       bool          `json:"enableValidation" yaml:"enableValidation"`

	EnableMetrics          bool          `json:"enableMetrics" yaml:"enableMetrics"`

	EnableTracing          bool          `json:"enableTracing" yaml:"enableTracing"`

}



// WorkflowEngineMetrics provides workflow engine metrics.

type WorkflowEngineMetrics struct {

	WorkflowRegistrations *prometheus.CounterVec

	WorkflowExecutions    *prometheus.CounterVec

	ExecutionDuration     *prometheus.HistogramVec

	WorkflowErrors        *prometheus.CounterVec

	ActiveWorkflows       prometheus.Gauge

}



// WorkflowExecutor executes individual workflow phases and actions.

type WorkflowExecutor struct {

	config  *WorkflowEngineConfig

	metrics *WorkflowEngineMetrics

	tracer  trace.Tracer

}



// WorkflowValidator validates workflow definitions.

type WorkflowValidator struct {

	config *WorkflowEngineConfig

	tracer trace.Tracer

}



// RollbackStrategy defines rollback behavior.

type RollbackStrategy struct {

	Enabled         bool                   `json:"enabled"`

	TriggerOn       []string               `json:"triggerOn"`

	Steps           []RollbackStep         `json:"steps"`

	Timeout         time.Duration          `json:"timeout"`

	NotifyOnFailure bool                   `json:"notifyOnFailure"`

	Config          map[string]interface{} `json:"config,omitempty"`

}



// RollbackStep defines a rollback step.

type RollbackStep struct {

	Name      string                 `json:"name"`

	Action    string                 `json:"action"`

	Config    map[string]interface{} `json:"config"`

	Timeout   time.Duration          `json:"timeout"`

	Required  bool                   `json:"required"`

	Condition string                 `json:"condition,omitempty"`

}



// ApprovalStrategy defines approval requirements.

type ApprovalStrategy struct {

	Required       bool                   `json:"required"`

	Approvers      []ApprovalRequirement  `json:"approvers"`

	AutoApproval   *AutoApprovalRule      `json:"autoApproval,omitempty"`

	Timeout        time.Duration          `json:"timeout"`

	NotifyChannels []string               `json:"notifyChannels,omitempty"`

	Config         map[string]interface{} `json:"config,omitempty"`

}



// ApprovalRequirement defines approval requirements.

type ApprovalRequirement struct {

	Type       string   `json:"type"`

	Count      int      `json:"count"`

	Users      []string `json:"users,omitempty"`

	Groups     []string `json:"groups,omitempty"`

	Roles      []string `json:"roles,omitempty"`

	Conditions []string `json:"conditions,omitempty"`

}



// AutoApprovalRule defines automatic approval rules.

type AutoApprovalRule struct {

	Enabled    bool                   `json:"enabled"`

	Conditions []string               `json:"conditions"`

	Timeout    time.Duration          `json:"timeout"`

	Config     map[string]interface{} `json:"config,omitempty"`

}



// NotificationStrategy defines notification behavior.

type NotificationStrategy struct {

	Enabled   bool                   `json:"enabled"`

	Events    []NotificationEvent    `json:"events"`

	Channels  []NotificationChannel  `json:"channels"`

	Templates map[string]string      `json:"templates,omitempty"`

	Config    map[string]interface{} `json:"config,omitempty"`

}



// NotificationEvent defines when to send notifications.

type NotificationEvent struct {

	Type       string                 `json:"type"`

	Conditions []string               `json:"conditions,omitempty"`

	Config     map[string]interface{} `json:"config,omitempty"`

}



// NotificationChannel defines notification channels.

type NotificationChannel struct {

	Type     string                 `json:"type"`

	Endpoint string                 `json:"endpoint"`

	Config   map[string]interface{} `json:"config,omitempty"`

	Enabled  bool                   `json:"enabled"`

}



// TimeoutStrategy defines timeout behavior.

type TimeoutStrategy struct {

	GlobalTimeout  time.Duration            `json:"globalTimeout"`

	PhaseTimeouts  map[string]time.Duration `json:"phaseTimeouts,omitempty"`

	ActionTimeouts map[string]time.Duration `json:"actionTimeouts,omitempty"`

	OnTimeout      string                   `json:"onTimeout"`

	Config         map[string]interface{}   `json:"config,omitempty"`

}



// ValidationResult represents workflow validation results.

type ValidationResult struct {

	Valid    bool                `json:"valid"`

	Errors   []ValidationError   `json:"errors,omitempty"`

	Warnings []ValidationWarning `json:"warnings,omitempty"`

	Metrics  *ValidationMetrics  `json:"metrics,omitempty"`

}



// ValidationError represents a validation error.

type ValidationError struct {

	Code     string `json:"code"`

	Message  string `json:"message"`

	Path     string `json:"path,omitempty"`

	Severity string `json:"severity"`

}



// ValidationWarning represents a validation warning.

type ValidationWarning struct {

	Code    string `json:"code"`

	Message string `json:"message"`

	Path    string `json:"path,omitempty"`

}



// ValidationMetrics provides validation metrics.

type ValidationMetrics struct {

	TotalPhases  int `json:"totalPhases"`

	TotalActions int `json:"totalActions"`

	Dependencies int `json:"dependencies"`

	Complexity   int `json:"complexity"`

}



// ComplianceResults represents compliance validation results.

type ComplianceResults struct {

	ORANCompliance     *ORANComplianceResult     `json:"oranCompliance,omitempty"`

	SecurityCompliance *SecurityComplianceResult `json:"securityCompliance,omitempty"`

	NetworkCompliance  *NetworkComplianceResult  `json:"networkCompliance,omitempty"`

	OverallStatus      string                    `json:"overallStatus"`

	Timestamp          time.Time                 `json:"timestamp"`

}



// ORANComplianceResult represents O-RAN compliance results.

type ORANComplianceResult struct {

	Interfaces     []InterfaceComplianceResult `json:"interfaces"`

	Standards      []StandardComplianceResult  `json:"standards"`

	Certifications []CertificationResult       `json:"certifications"`

	OverallScore   float64                     `json:"overallScore"`

}



// InterfaceComplianceResult represents interface compliance.

type InterfaceComplianceResult struct {

	Name      string   `json:"name"`

	Type      string   `json:"type"`

	Version   string   `json:"version"`

	Compliant bool     `json:"compliant"`

	Score     float64  `json:"score"`

	Issues    []string `json:"issues,omitempty"`

}



// StandardComplianceResult represents standard compliance.

type StandardComplianceResult struct {

	Name      string   `json:"name"`

	Version   string   `json:"version"`

	Compliant bool     `json:"compliant"`

	Score     float64  `json:"score"`

	Issues    []string `json:"issues,omitempty"`

}



// CertificationResult represents certification status.

type CertificationResult struct {

	Name      string    `json:"name"`

	Authority string    `json:"authority"`

	Valid     bool      `json:"valid"`

	ExpiresAt time.Time `json:"expiresAt,omitempty"`

	Issues    []string  `json:"issues,omitempty"`

}



// SecurityComplianceResult represents security compliance.

type SecurityComplianceResult struct {

	Policies        []PolicyComplianceResult `json:"policies"`

	Vulnerabilities []VulnerabilityResult    `json:"vulnerabilities"`

	OverallScore    float64                  `json:"overallScore"`

}



// PolicyComplianceResult represents policy compliance.

type PolicyComplianceResult struct {

	Name      string   `json:"name"`

	Type      string   `json:"type"`

	Compliant bool     `json:"compliant"`

	Issues    []string `json:"issues,omitempty"`

}



// VulnerabilityResult represents vulnerability assessment.

type VulnerabilityResult struct {

	ID          string `json:"id"`

	Severity    string `json:"severity"`

	Component   string `json:"component"`

	Description string `json:"description"`

	Fixed       bool   `json:"fixed"`

}



// NetworkComplianceResult represents network compliance.

type NetworkComplianceResult struct {

	SliceCompliance    []SliceComplianceResult `json:"sliceCompliance"`

	QoSCompliance      []QoSComplianceResult   `json:"qosCompliance"`

	SecurityCompliance []NetworkSecurityResult `json:"securityCompliance"`

	OverallScore       float64                 `json:"overallScore"`

}



// SliceComplianceResult represents slice compliance.

type SliceComplianceResult struct {

	SliceID   string   `json:"sliceId"`

	Type      string   `json:"type"`

	Compliant bool     `json:"compliant"`

	Issues    []string `json:"issues,omitempty"`

}



// QoSComplianceResult represents QoS compliance.

type QoSComplianceResult struct {

	Policy    string   `json:"policy"`

	Compliant bool     `json:"compliant"`

	Issues    []string `json:"issues,omitempty"`

}



// NetworkSecurityResult represents network security assessment.

type NetworkSecurityResult struct {

	Policy string   `json:"policy"`

	Secure bool     `json:"secure"`

	Issues []string `json:"issues,omitempty"`

}



// ResourceUsageSummary provides resource usage summary.

type ResourceUsageSummary struct {

	CPU           ResourceUsage `json:"cpu"`

	Memory        ResourceUsage `json:"memory"`

	Storage       ResourceUsage `json:"storage"`

	Network       ResourceUsage `json:"network"`

	TotalClusters int           `json:"totalClusters"`

	TotalPods     int           `json:"totalPods"`

	Timestamp     time.Time     `json:"timestamp"`

}



// ResourceUsage represents resource usage metrics.

type ResourceUsage struct {

	Requested  string  `json:"requested"`

	Used       string  `json:"used"`

	Available  string  `json:"available"`

	Percentage float64 `json:"percentage"`

}



// DeploymentResult represents deployment results.

type DeploymentResult struct {

	PackageName string           `json:"packageName"`

	ClusterName string           `json:"clusterName"`

	Status      string           `json:"status"`

	Message     string           `json:"message"`

	Resources   []ResourceResult `json:"resources"`

	SyncResult  *SyncResult      `json:"syncResult,omitempty"`

	Duration    time.Duration    `json:"duration"`

	Timestamp   time.Time        `json:"timestamp"`

}



// ResourceResult represents individual resource deployment result.

type ResourceResult struct {

	Name      string `json:"name"`

	Kind      string `json:"kind"`

	Namespace string `json:"namespace"`

	Status    string `json:"status"`

	Message   string `json:"message"`

}



// Default configuration.

var DefaultWorkflowEngineConfig = &WorkflowEngineConfig{

	MaxConcurrentWorkflows: 10,

	DefaultTimeout:         30 * time.Minute,

	StageTimeout:           10 * time.Minute,

	RetryAttempts:          3,

	RetryBackoff:           30 * time.Second,

	EnableValidation:       true,

	EnableMetrics:          true,

	EnableTracing:          true,

}



// NewNephioWorkflowEngine creates a new workflow engine.

func NewNephioWorkflowEngine(config *WorkflowEngineConfig) (*NephioWorkflowEngine, error) {

	if config == nil {

		config = DefaultWorkflowEngineConfig

	}



	// Initialize metrics.

	metrics := &WorkflowEngineMetrics{

		WorkflowRegistrations: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_workflow_registrations_total",

				Help: "Total number of workflow registrations",

			},

			[]string{"workflow", "intent_type", "status"},

		),

		WorkflowExecutions: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_workflow_engine_executions_total",

				Help: "Total number of workflow executions by the engine",

			},

			[]string{"workflow", "phase", "status"},

		),

		ExecutionDuration: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name:    "nephio_workflow_engine_execution_duration_seconds",

				Help:    "Duration of workflow executions",

				Buckets: prometheus.ExponentialBuckets(1, 2, 10),

			},

			[]string{"workflow", "phase"},

		),

		WorkflowErrors: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_workflow_engine_errors_total",

				Help: "Total number of workflow engine errors",

			},

			[]string{"workflow", "error_type"},

		),

		ActiveWorkflows: promauto.NewGauge(

			prometheus.GaugeOpts{

				Name: "nephio_workflow_engine_active_workflows",

				Help: "Number of currently active workflows",

			},

		),

	}



	// Initialize executor.

	executor := &WorkflowExecutor{

		config:  config,

		metrics: metrics,

		tracer:  otel.Tracer("nephio-workflow-executor"),

	}



	// Initialize validator.

	validator := &WorkflowValidator{

		config: config,

		tracer: otel.Tracer("nephio-workflow-validator"),

	}



	engine := &NephioWorkflowEngine{

		executor:  executor,

		validator: validator,

		metrics:   metrics,

		tracer:    otel.Tracer("nephio-workflow-engine"),

		config:    config,

	}



	return engine, nil

}



// RegisterWorkflow registers a workflow definition.

func (nwe *NephioWorkflowEngine) RegisterWorkflow(workflow *WorkflowDefinition) error {

	ctx, span := nwe.tracer.Start(context.Background(), "register-workflow")

	defer span.End()



	logger := log.FromContext(ctx).WithName("workflow-engine").WithValues("workflow", workflow.Name)



	span.SetAttributes(

		attribute.String("workflow.name", workflow.Name),

		attribute.Int("workflow.phases", len(workflow.Phases)),

	)



	// Validate workflow definition.

	if nwe.config.EnableValidation {

		validationResult := nwe.validator.ValidateWorkflow(ctx, workflow)

		if !validationResult.Valid {

			span.RecordError(fmt.Errorf("workflow validation failed"))

			nwe.metrics.WorkflowRegistrations.WithLabelValues(

				workflow.Name, "unknown", "validation_failed",

			).Inc()

			return fmt.Errorf("workflow validation failed: %v", validationResult.Errors)

		}

	}



	// Set default values.

	workflow.CreatedAt = time.Now()

	workflow.UpdatedAt = time.Now()



	// Store workflow.

	nwe.workflows.Store(workflow.Name, workflow)



	// Update metrics.

	for _, intentType := range workflow.IntentTypes {

		nwe.metrics.WorkflowRegistrations.WithLabelValues(

			workflow.Name, string(intentType), "success",

		).Inc()

	}



	logger.Info("Workflow registered successfully",

		"phases", len(workflow.Phases),

		"intentTypes", len(workflow.IntentTypes),

	)



	return nil

}



// GetWorkflow retrieves a workflow definition by name.

func (nwe *NephioWorkflowEngine) GetWorkflow(ctx context.Context, name string) (*WorkflowDefinition, error) {

	if value, exists := nwe.workflows.Load(name); exists {

		if workflow, ok := value.(*WorkflowDefinition); ok {

			return workflow, nil

		}

	}

	return nil, fmt.Errorf("workflow not found: %s", name)

}



// ListWorkflows lists all registered workflows.

func (nwe *NephioWorkflowEngine) ListWorkflows(ctx context.Context) ([]*WorkflowDefinition, error) {

	workflows := make([]*WorkflowDefinition, 0)



	nwe.workflows.Range(func(key, value interface{}) bool {

		if workflow, ok := value.(*WorkflowDefinition); ok {

			workflows = append(workflows, workflow)

		}

		return true

	})



	return workflows, nil

}



// ValidateWorkflow validates a workflow definition.

func (wv *WorkflowValidator) ValidateWorkflow(ctx context.Context, workflow *WorkflowDefinition) *ValidationResult {

	ctx, span := wv.tracer.Start(ctx, "validate-workflow")

	defer span.End()



	result := &ValidationResult{

		Valid:    true,

		Errors:   make([]ValidationError, 0),

		Warnings: make([]ValidationWarning, 0),

		Metrics: &ValidationMetrics{

			TotalPhases:  len(workflow.Phases),

			TotalActions: wv.countTotalActions(workflow),

			Dependencies: wv.countDependencies(workflow),

			Complexity:   wv.calculateComplexity(workflow),

		},

	}



	// Validate basic workflow properties.

	if workflow.Name == "" {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Code:     "MISSING_NAME",

			Message:  "Workflow name is required",

			Severity: "ERROR",

		})

	}



	if len(workflow.IntentTypes) == 0 {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Code:    "NO_INTENT_TYPES",

			Message: "Workflow has no associated intent types",

		})

	}



	if len(workflow.Phases) == 0 {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Code:     "NO_PHASES",

			Message:  "Workflow must have at least one phase",

			Severity: "ERROR",

		})

	}



	// Validate phases.

	for i, phase := range workflow.Phases {

		phaseErrors := wv.validatePhase(ctx, phase, i)

		result.Errors = append(result.Errors, phaseErrors...)

	}



	// Validate phase dependencies.

	dependencyErrors := wv.validateDependencies(ctx, workflow)

	result.Errors = append(result.Errors, dependencyErrors...)



	// Check for circular dependencies.

	if wv.hasCircularDependencies(workflow) {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Code:     "CIRCULAR_DEPENDENCIES",

			Message:  "Workflow has circular dependencies",

			Severity: "ERROR",

		})

	}



	// Set overall validity.

	for _, err := range result.Errors {

		if err.Severity == "ERROR" {

			result.Valid = false

			break

		}

	}



	span.SetAttributes(

		attribute.Bool("validation.valid", result.Valid),

		attribute.Int("validation.errors", len(result.Errors)),

		attribute.Int("validation.warnings", len(result.Warnings)),

	)



	return result

}



// validatePhase validates an individual workflow phase.

func (wv *WorkflowValidator) validatePhase(ctx context.Context, phase WorkflowPhase, index int) []ValidationError {

	errors := make([]ValidationError, 0)

	phasePrefix := fmt.Sprintf("phases[%d]", index)



	if phase.Name == "" {

		errors = append(errors, ValidationError{

			Code:     "MISSING_PHASE_NAME",

			Message:  "Phase name is required",

			Path:     fmt.Sprintf("%s.name", phasePrefix),

			Severity: "ERROR",

		})

	}



	if len(phase.Actions) == 0 {

		errors = append(errors, ValidationError{

			Code:     "NO_PHASE_ACTIONS",

			Message:  "Phase must have at least one action",

			Path:     fmt.Sprintf("%s.actions", phasePrefix),

			Severity: "ERROR",

		})

	}



	// Validate actions.

	for j, action := range phase.Actions {

		actionErrors := wv.validateAction(ctx, action, fmt.Sprintf("%s.actions[%d]", phasePrefix, j))

		errors = append(errors, actionErrors...)

	}



	return errors

}



// validateAction validates a workflow action.

func (wv *WorkflowValidator) validateAction(ctx context.Context, action WorkflowAction, path string) []ValidationError {

	errors := make([]ValidationError, 0)



	if action.Name == "" {

		errors = append(errors, ValidationError{

			Code:     "MISSING_ACTION_NAME",

			Message:  "Action name is required",

			Path:     fmt.Sprintf("%s.name", path),

			Severity: "ERROR",

		})

	}



	if action.Type == "" {

		errors = append(errors, ValidationError{

			Code:     "MISSING_ACTION_TYPE",

			Message:  "Action type is required",

			Path:     fmt.Sprintf("%s.type", path),

			Severity: "ERROR",

		})

	}



	// Validate action type.

	validTypes := []WorkflowActionType{

		WorkflowActionTypeCreatePackageRevision,

		WorkflowActionTypeSpecializePackage,

		WorkflowActionTypeValidatePackage,

		WorkflowActionTypeApprovePackage,

		WorkflowActionTypeDeployToCluster,

		WorkflowActionTypeWaitForDeployment,

		WorkflowActionTypeVerifyHealth,

		WorkflowActionTypeNotifyUsers,

	}



	isValidType := false

	for _, validType := range validTypes {

		if action.Type == validType {

			isValidType = true

			break

		}

	}



	if !isValidType {

		errors = append(errors, ValidationError{

			Code:     "INVALID_ACTION_TYPE",

			Message:  fmt.Sprintf("Invalid action type: %s", action.Type),

			Path:     fmt.Sprintf("%s.type", path),

			Severity: "ERROR",

		})

	}



	return errors

}



// validateDependencies validates workflow phase dependencies.

func (wv *WorkflowValidator) validateDependencies(ctx context.Context, workflow *WorkflowDefinition) []ValidationError {

	errors := make([]ValidationError, 0)

	phaseNames := make(map[string]bool)



	// Build phase name map.

	for _, phase := range workflow.Phases {

		phaseNames[phase.Name] = true

	}



	// Validate dependencies.

	for i, phase := range workflow.Phases {

		for _, dep := range phase.Dependencies {

			if !phaseNames[dep] {

				errors = append(errors, ValidationError{

					Code:     "INVALID_DEPENDENCY",

					Message:  fmt.Sprintf("Phase %s depends on non-existent phase: %s", phase.Name, dep),

					Path:     fmt.Sprintf("phases[%d].dependencies", i),

					Severity: "ERROR",

				})

			}

		}

	}



	return errors

}



// hasCircularDependencies checks for circular dependencies.

func (wv *WorkflowValidator) hasCircularDependencies(workflow *WorkflowDefinition) bool {

	// Build dependency graph.

	graph := make(map[string][]string)

	for _, phase := range workflow.Phases {

		graph[phase.Name] = phase.Dependencies

	}



	// Use DFS to detect cycles.

	visited := make(map[string]bool)

	recStack := make(map[string]bool)



	var hasCycle func(string) bool

	hasCycle = func(phase string) bool {

		visited[phase] = true

		recStack[phase] = true



		for _, dep := range graph[phase] {

			if !visited[dep] {

				if hasCycle(dep) {

					return true

				}

			} else if recStack[dep] {

				return true

			}

		}



		recStack[phase] = false

		return false

	}



	for _, phase := range workflow.Phases {

		if !visited[phase.Name] {

			if hasCycle(phase.Name) {

				return true

			}

		}

	}



	return false

}



// Helper methods for validation metrics.



func (wv *WorkflowValidator) countTotalActions(workflow *WorkflowDefinition) int {

	count := 0

	for _, phase := range workflow.Phases {

		count += len(phase.Actions)

	}

	return count

}



func (wv *WorkflowValidator) countDependencies(workflow *WorkflowDefinition) int {

	count := 0

	for _, phase := range workflow.Phases {

		count += len(phase.Dependencies)

	}

	return count

}



func (wv *WorkflowValidator) calculateComplexity(workflow *WorkflowDefinition) int {

	// Simple complexity calculation based on phases, actions, and dependencies.

	complexity := len(workflow.Phases)

	complexity += wv.countTotalActions(workflow)

	complexity += wv.countDependencies(workflow)



	// Add complexity for conditions.

	for _, phase := range workflow.Phases {

		complexity += len(phase.Conditions)

		for _, action := range phase.Actions {

			if action.RetryPolicy != nil {

				complexity += 1

			}

		}

	}



	return complexity

}



// registerStandardWorkflows registers the standard Nephio workflows.

func (nwo *NephioWorkflowOrchestrator) registerStandardWorkflows() error {

	workflows := []*WorkflowDefinition{

		nwo.createStandardDeploymentWorkflow(),

		nwo.createStandardConfigurationWorkflow(),

		nwo.createStandardScalingWorkflow(),

		nwo.createORANDeploymentWorkflow(),

		nwo.create5GCoreDeploymentWorkflow(),

	}



	for _, workflow := range workflows {

		if err := nwo.workflowEngine.RegisterWorkflow(workflow); err != nil {

			return fmt.Errorf("failed to register workflow %s: %w", workflow.Name, err)

		}

	}



	return nil

}



// createStandardDeploymentWorkflow creates the standard deployment workflow.

func (nwo *NephioWorkflowOrchestrator) createStandardDeploymentWorkflow() *WorkflowDefinition {

	return &WorkflowDefinition{

		Name:        "standard-deployment",

		Description: "Standard Nephio deployment workflow",

		IntentTypes: []v1.IntentType{

			v1.IntentTypeDeployment,

		},

		Phases: []WorkflowPhase{

			{

				Name:         "blueprint-selection",

				Description:  "Select appropriate blueprint for deployment",

				Type:         WorkflowPhaseTypeBlueprintSelection,

				Dependencies: []string{},

				Actions: []WorkflowAction{

					{

						Name:     "select-blueprint",

						Type:     WorkflowActionTypeCreatePackageRevision,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         10 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "package-specialization",

				Description:  "Create specialized packages for target clusters",

				Type:         WorkflowPhaseTypePackageSpecialization,

				Dependencies: []string{"blueprint-selection"},

				Actions: []WorkflowAction{

					{

						Name:     "specialize-package",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  10 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "validation",

				Description:  "Validate specialized packages",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{"package-specialization"},

				Actions: []WorkflowAction{

					{

						Name:     "validate-packages",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         10 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "approval",

				Description:  "Approve packages for deployment",

				Type:         WorkflowPhaseTypeApproval,

				Dependencies: []string{"validation"},

				Actions: []WorkflowAction{

					{

						Name:     "approve-packages",

						Type:     WorkflowActionTypeApprovePackage,

						Required: true,

						Timeout:  30 * time.Minute, // Longer timeout for manual approvals

					},

				},

				Timeout:         45 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "deployment",

				Description:  "Deploy packages to target clusters",

				Type:         WorkflowPhaseTypeDeployment,

				Dependencies: []string{"approval"},

				Actions: []WorkflowAction{

					{

						Name:     "deploy-to-clusters",

						Type:     WorkflowActionTypeDeployToCluster,

						Required: true,

						Timeout:  15 * time.Minute,

					},

					{

						Name:     "wait-for-deployment",

						Type:     WorkflowActionTypeWaitForDeployment,

						Required: true,

						Timeout:  20 * time.Minute,

					},

				},

				Timeout:         30 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "monitoring",

				Description:  "Monitor deployment health",

				Type:         WorkflowPhaseTypeMonitoring,

				Dependencies: []string{"deployment"},

				Actions: []WorkflowAction{

					{

						Name:     "verify-health",

						Type:     WorkflowActionTypeVerifyHealth,

						Required: true,

						Timeout:  10 * time.Minute,

					},

					{

						Name:     "notify-completion",

						Type:     WorkflowActionTypeNotifyUsers,

						Required: false,

						Timeout:  2 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: true, // Continue even if notifications fail

			},

		},

		Rollback: &RollbackStrategy{

			Enabled:   true,

			TriggerOn: []string{"deployment-failure", "validation-failure"},

			Timeout:   15 * time.Minute,

		},

		Approvals: &ApprovalStrategy{

			Required: true,

			Timeout:  30 * time.Minute,

		},

		Timeouts: &TimeoutStrategy{

			GlobalTimeout: 2 * time.Hour,

			OnTimeout:     "rollback",

		},

	}

}



// createStandardConfigurationWorkflow creates the standard configuration workflow.

func (nwo *NephioWorkflowOrchestrator) createStandardConfigurationWorkflow() *WorkflowDefinition {

	return &WorkflowDefinition{

		Name:        "standard-configuration",

		Description: "Standard Nephio configuration workflow",

		IntentTypes: []v1.IntentType{

			v1.IntentTypeOptimization,

		},

		Phases: []WorkflowPhase{

			{

				Name:         "blueprint-selection",

				Description:  "Select appropriate blueprint for configuration",

				Type:         WorkflowPhaseTypeBlueprintSelection,

				Dependencies: []string{},

				Actions: []WorkflowAction{

					{

						Name:     "select-blueprint",

						Type:     WorkflowActionTypeCreatePackageRevision,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         10 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "package-specialization",

				Description:  "Create specialized configuration packages",

				Type:         WorkflowPhaseTypePackageSpecialization,

				Dependencies: []string{"blueprint-selection"},

				Actions: []WorkflowAction{

					{

						Name:     "specialize-config",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  8 * time.Minute,

					},

				},

				Timeout:         12 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "validation",

				Description:  "Validate configuration packages",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{"package-specialization"},

				Actions: []WorkflowAction{

					{

						Name:     "validate-config",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         8 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "deployment",

				Description:  "Deploy configuration to clusters",

				Type:         WorkflowPhaseTypeDeployment,

				Dependencies: []string{"validation"},

				Actions: []WorkflowAction{

					{

						Name:     "deploy-config",

						Type:     WorkflowActionTypeDeployToCluster,

						Required: true,

						Timeout:  10 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: false,

			},

		},

		Rollback: &RollbackStrategy{

			Enabled:   true,

			TriggerOn: []string{"deployment-failure", "validation-failure"},

			Timeout:   10 * time.Minute,

		},

		Approvals: &ApprovalStrategy{

			Required: false, // Auto-approve configuration changes

		},

		Timeouts: &TimeoutStrategy{

			GlobalTimeout: 1 * time.Hour,

			OnTimeout:     "rollback",

		},

	}

}



// createStandardScalingWorkflow creates the standard scaling workflow.

func (nwo *NephioWorkflowOrchestrator) createStandardScalingWorkflow() *WorkflowDefinition {

	return &WorkflowDefinition{

		Name:        "standard-scaling",

		Description: "Standard Nephio scaling workflow",

		IntentTypes: []v1.IntentType{

			v1.IntentTypeScaling,

		},

		Phases: []WorkflowPhase{

			{

				Name:         "blueprint-selection",

				Description:  "Select scaling blueprint",

				Type:         WorkflowPhaseTypeBlueprintSelection,

				Dependencies: []string{},

				Actions: []WorkflowAction{

					{

						Name:     "select-scaling-blueprint",

						Type:     WorkflowActionTypeCreatePackageRevision,

						Required: true,

						Timeout:  3 * time.Minute,

					},

				},

				Timeout:         5 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "package-specialization",

				Description:  "Create scaling-specific packages",

				Type:         WorkflowPhaseTypePackageSpecialization,

				Dependencies: []string{"blueprint-selection"},

				Actions: []WorkflowAction{

					{

						Name:     "specialize-scaling",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         8 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "deployment",

				Description:  "Apply scaling changes",

				Type:         WorkflowPhaseTypeDeployment,

				Dependencies: []string{"package-specialization"},

				Actions: []WorkflowAction{

					{

						Name:     "apply-scaling",

						Type:     WorkflowActionTypeDeployToCluster,

						Required: true,

						Timeout:  15 * time.Minute,

					},

				},

				Timeout:         20 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "monitoring",

				Description:  "Monitor scaling results",

				Type:         WorkflowPhaseTypeMonitoring,

				Dependencies: []string{"deployment"},

				Actions: []WorkflowAction{

					{

						Name:     "verify-scaling",

						Type:     WorkflowActionTypeVerifyHealth,

						Required: true,

						Timeout:  10 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: true,

			},

		},

		Rollback: &RollbackStrategy{

			Enabled:   true,

			TriggerOn: []string{"scaling-failure"},

			Timeout:   10 * time.Minute,

		},

		Timeouts: &TimeoutStrategy{

			GlobalTimeout: 45 * time.Minute,

			OnTimeout:     "rollback",

		},

	}

}



// createORANDeploymentWorkflow creates the O-RAN specific deployment workflow.

func (nwo *NephioWorkflowOrchestrator) createORANDeploymentWorkflow() *WorkflowDefinition {

	return &WorkflowDefinition{

		Name:        "oran-deployment",

		Description: "O-RAN compliant deployment workflow",

		IntentTypes: []v1.IntentType{

			v1.IntentTypeDeployment,

		},

		Phases: []WorkflowPhase{

			{

				Name:         "oran-compliance-check",

				Description:  "Verify O-RAN compliance requirements",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{},

				Actions: []WorkflowAction{

					{

						Name:     "check-oran-compliance",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         8 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "blueprint-selection",

				Description:  "Select O-RAN compliant blueprint",

				Type:         WorkflowPhaseTypeBlueprintSelection,

				Dependencies: []string{"oran-compliance-check"},

				Actions: []WorkflowAction{

					{

						Name:     "select-oran-blueprint",

						Type:     WorkflowActionTypeCreatePackageRevision,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         10 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "package-specialization",

				Description:  "Create O-RAN specialized packages",

				Type:         WorkflowPhaseTypePackageSpecialization,

				Dependencies: []string{"blueprint-selection"},

				Actions: []WorkflowAction{

					{

						Name:     "specialize-oran-package",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  12 * time.Minute,

					},

				},

				Timeout:         18 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "validation",

				Description:  "Validate O-RAN packages",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{"package-specialization"},

				Actions: []WorkflowAction{

					{

						Name:     "validate-oran-packages",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  8 * time.Minute,

					},

				},

				Timeout:         12 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "deployment",

				Description:  "Deploy O-RAN packages",

				Type:         WorkflowPhaseTypeDeployment,

				Dependencies: []string{"validation"},

				Actions: []WorkflowAction{

					{

						Name:     "deploy-oran-packages",

						Type:     WorkflowActionTypeDeployToCluster,

						Required: true,

						Timeout:  20 * time.Minute,

					},

				},

				Timeout:         30 * time.Minute,

				ContinueOnError: false,

			},

		},

		Rollback: &RollbackStrategy{

			Enabled:   true,

			TriggerOn: []string{"compliance-failure", "deployment-failure"},

			Timeout:   20 * time.Minute,

		},

		Timeouts: &TimeoutStrategy{

			GlobalTimeout: 2 * time.Hour,

			OnTimeout:     "rollback",

		},

	}

}



// create5GCoreDeploymentWorkflow creates the 5G Core specific deployment workflow.

func (nwo *NephioWorkflowOrchestrator) create5GCoreDeploymentWorkflow() *WorkflowDefinition {

	return &WorkflowDefinition{

		Name:        "5g-core-deployment",

		Description: "5G Core network functions deployment workflow",

		IntentTypes: []v1.IntentType{

			v1.IntentTypeDeployment,

		},

		Phases: []WorkflowPhase{

			{

				Name:         "5g-requirements-check",

				Description:  "Verify 5G Core requirements",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{},

				Actions: []WorkflowAction{

					{

						Name:     "check-5g-requirements",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         8 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "blueprint-selection",

				Description:  "Select 5G Core blueprint",

				Type:         WorkflowPhaseTypeBlueprintSelection,

				Dependencies: []string{"5g-requirements-check"},

				Actions: []WorkflowAction{

					{

						Name:     "select-5g-blueprint",

						Type:     WorkflowActionTypeCreatePackageRevision,

						Required: true,

						Timeout:  5 * time.Minute,

					},

				},

				Timeout:         10 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "package-specialization",

				Description:  "Create 5G Core specialized packages",

				Type:         WorkflowPhaseTypePackageSpecialization,

				Dependencies: []string{"blueprint-selection"},

				Actions: []WorkflowAction{

					{

						Name:     "specialize-5g-package",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  15 * time.Minute,

					},

				},

				Timeout:         20 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "network-slice-configuration",

				Description:  "Configure network slices",

				Type:         WorkflowPhaseTypeConfiguration,

				Dependencies: []string{"package-specialization"},

				Actions: []WorkflowAction{

					{

						Name:     "configure-slices",

						Type:     WorkflowActionTypeSpecializePackage,

						Required: true,

						Timeout:  10 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "validation",

				Description:  "Validate 5G Core packages",

				Type:         WorkflowPhaseTypeValidation,

				Dependencies: []string{"network-slice-configuration"},

				Actions: []WorkflowAction{

					{

						Name:     "validate-5g-packages",

						Type:     WorkflowActionTypeValidatePackage,

						Required: true,

						Timeout:  10 * time.Minute,

					},

				},

				Timeout:         15 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "deployment",

				Description:  "Deploy 5G Core network functions",

				Type:         WorkflowPhaseTypeDeployment,

				Dependencies: []string{"validation"},

				Actions: []WorkflowAction{

					{

						Name:     "deploy-5g-core",

						Type:     WorkflowActionTypeDeployToCluster,

						Required: true,

						Timeout:  25 * time.Minute,

					},

				},

				Timeout:         35 * time.Minute,

				ContinueOnError: false,

			},

			{

				Name:         "monitoring",

				Description:  "Monitor 5G Core deployment",

				Type:         WorkflowPhaseTypeMonitoring,

				Dependencies: []string{"deployment"},

				Actions: []WorkflowAction{

					{

						Name:     "verify-5g-health",

						Type:     WorkflowActionTypeVerifyHealth,

						Required: true,

						Timeout:  15 * time.Minute,

					},

				},

				Timeout:         20 * time.Minute,

				ContinueOnError: true,

			},

		},

		Rollback: &RollbackStrategy{

			Enabled:   true,

			TriggerOn: []string{"requirements-failure", "deployment-failure", "slice-failure"},

			Timeout:   25 * time.Minute,

		},

		Timeouts: &TimeoutStrategy{

			GlobalTimeout: 3 * time.Hour,

			OnTimeout:     "rollback",

		},

	}

}


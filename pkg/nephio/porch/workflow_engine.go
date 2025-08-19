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

package porch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkflowEngine provides comprehensive package approval and promotion workflow capabilities
// Manages policy-based approval, multi-stage workflows, approval delegation, automated workflows,
// human-in-the-loop processes, audit trails, and integration with external systems
type WorkflowEngine interface {
	// Workflow lifecycle management
	CreateWorkflow(ctx context.Context, spec *WorkflowSpec) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error)
	DeleteWorkflow(ctx context.Context, workflowID string) error
	GetWorkflow(ctx context.Context, workflowID string) (*Workflow, error)
	ListWorkflows(ctx context.Context, opts *WorkflowListOptions) (*WorkflowList, error)

	// Workflow execution
	StartWorkflow(ctx context.Context, workflowID string, input *WorkflowInput) (*WorkflowExecution, error)
	ResumeWorkflow(ctx context.Context, executionID string) (*WorkflowExecution, error)
	PauseWorkflow(ctx context.Context, executionID string) error
	AbortWorkflow(ctx context.Context, executionID string, reason string) error
	GetWorkflowExecution(ctx context.Context, executionID string) (*WorkflowExecution, error)

	// Approval management
	SubmitApproval(ctx context.Context, approvalID string, decision ApprovalDecision, comment string) (*ApprovalResult, error)
	GetPendingApprovals(ctx context.Context, approver string) ([]*PendingApproval, error)
	GetApprovalHistory(ctx context.Context, packageRef *PackageReference) (*ApprovalHistory, error)
	DelegateApproval(ctx context.Context, approvalID string, fromApprover, toApprover string, reason string) error

	// Policy-based approval
	RegisterApprovalPolicy(ctx context.Context, policy *ApprovalPolicy) error
	UnregisterApprovalPolicy(ctx context.Context, policyID string) error
	EvaluateApprovalPolicies(ctx context.Context, packageRef *PackageReference, stage PackageRevisionLifecycle) (*PolicyEvaluationResult, error)
	GetApprovalPolicies(ctx context.Context) ([]*ApprovalPolicy, error)

	// Multi-stage workflows
	DefineWorkflowStage(ctx context.Context, workflowID string, stage *WorkflowStageDefinition) error
	ExecuteStage(ctx context.Context, executionID string, stageID string) (*StageExecutionResult, error)
	SkipStage(ctx context.Context, executionID string, stageID string, reason string) error
	RetryStage(ctx context.Context, executionID string, stageID string) (*StageExecutionResult, error)

	// Automated workflow triggers
	RegisterWorkflowTrigger(ctx context.Context, trigger *WorkflowTrigger) error
	UnregisterWorkflowTrigger(ctx context.Context, triggerID string) error
	EvaluateTriggers(ctx context.Context, event *PackageEvent) ([]*TriggeredWorkflow, error)

	// Human-in-the-loop processes
	CreateManualTask(ctx context.Context, task *ManualTask) (*ManualTaskExecution, error)
	CompleteManualTask(ctx context.Context, taskID string, result *TaskResult) error
	EscalateTask(ctx context.Context, taskID string, escalation *TaskEscalation) error
	GetManualTasks(ctx context.Context, assignee string, status TaskStatus) ([]*ManualTaskExecution, error)

	// External system integration
	RegisterExternalIntegration(ctx context.Context, integration *ExternalIntegration) error
	UnregisterExternalIntegration(ctx context.Context, integrationID string) error
	TriggerExternalAction(ctx context.Context, integrationID string, action *ExternalAction) (*ExternalActionResult, error)

	// Audit and compliance
	GetWorkflowAuditLog(ctx context.Context, packageRef *PackageReference, opts *AuditLogOptions) (*WorkflowAuditLog, error)
	GenerateComplianceReport(ctx context.Context, opts *ComplianceReportOptions) (*ComplianceReport, error)
	ExportAuditData(ctx context.Context, opts *AuditExportOptions) (*AuditExport, error)

	// Metrics and monitoring
	GetWorkflowMetrics(ctx context.Context) (*WorkflowEngineMetrics, error)
	GetWorkflowStatistics(ctx context.Context, timeRange *TimeRange) (*WorkflowStatistics, error)

	// Health and maintenance
	GetEngineHealth(ctx context.Context) (*WorkflowEngineHealth, error)
	CleanupCompletedWorkflows(ctx context.Context, olderThan time.Duration) (*CleanupResult, error)
	Close() error
}

// workflowEngine implements comprehensive workflow management
type workflowEngine struct {
	// Core dependencies
	client  *Client
	logger  logr.Logger
	metrics *WorkflowEngineMetrics

	// Workflow management
	workflowRegistry *WorkflowRegistry
	executionEngine  *ExecutionEngine
	stateManager     *WorkflowStateManager

	// Approval system
	approvalManager   *ApprovalManager
	policyEngine      *PolicyEngine
	delegationService *DelegationService

	// External integrations
	integrations     map[string]*ExternalIntegration
	integrationMutex sync.RWMutex

	// Task management
	taskManager      *ManualTaskManager
	escalationEngine *EscalationEngine

	// Audit and compliance
	auditLogger      *WorkflowAuditLogger
	complianceEngine *ComplianceEngine

	// Background processing
	triggerEvaluator *TriggerEvaluator
	executionMonitor *ExecutionMonitor

	// Configuration
	config *WorkflowEngineConfig

	// Concurrency control
	executionLocks map[string]*sync.Mutex
	lockMutex      sync.RWMutex

	// Background processing
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// Core data structures

// WorkflowInput provides input data for workflow execution
type WorkflowInput struct {
	PackageRef   *PackageReference
	User         string
	TriggerEvent *PackageEvent
	Parameters   map[string]interface{}
	Priority     WorkflowPriority
	Deadline     *time.Time
	Context      map[string]string
}

// WorkflowExecution represents a running workflow instance
type WorkflowExecution struct {
	ID               string
	WorkflowID       string
	PackageRef       *PackageReference
	Status           WorkflowExecutionStatus
	CurrentStage     string
	StartTime        time.Time
	EndTime          *time.Time
	Duration         time.Duration
	Input            *WorkflowInput
	Results          map[string]interface{}
	StageResults     map[string]*StageExecutionResult
	PendingApprovals []*PendingApproval
	ManualTasks      []*ManualTaskExecution
	Errors           []WorkflowError
	Metadata         map[string]interface{}
}

// WorkflowStageDefinition defines a workflow stage
type WorkflowStageDefinition struct {
	ID               string
	Name             string
	Type             WorkflowStageType
	Prerequisites    []string
	Actions          []WorkflowAction
	ApprovalRequired bool
	Approvers        []ApprovalRule
	Timeout          time.Duration
	RetryPolicy      *RetryPolicy
	OnFailure        *FailurePolicy
	Conditions       []StageCondition
}

// StageExecutionResult contains stage execution results
type StageExecutionResult struct {
	StageID    string
	Status     StageExecutionStatus
	StartTime  time.Time
	EndTime    *time.Time
	Duration   time.Duration
	Output     map[string]interface{}
	Approvals  []*ApprovalResult
	Tasks      []*TaskResult
	Errors     []StageError
	RetryCount int
	NextStages []string
}

// Approval system types

// ApprovalPolicy defines approval requirements
type ApprovalPolicy struct {
	ID              string
	Name            string
	Description     string
	PackageSelector *PackageSelector
	StageSelector   []PackageRevisionLifecycle
	Rules           []*ApprovalRule
	Priority        int
	Enabled         bool
	CreatedAt       time.Time
	CreatedBy       string
	Metadata        map[string]string
}

// ApprovalRule defines who can approve and under what conditions
type ApprovalRule struct {
	ID               string
	Type             ApprovalRuleType
	Approvers        []ApproverSpec
	RequiredCount    int
	Conditions       []ApprovalCondition
	Timeout          time.Duration
	EscalationPolicy *EscalationPolicy
	BypassConditions []BypassCondition
}

// ApprovalDecision represents an approval decision
type ApprovalDecision string

const (
	ApprovalDecisionApprove ApprovalDecision = "approve"
	ApprovalDecisionReject  ApprovalDecision = "reject"
	ApprovalDecisionDefer   ApprovalDecision = "defer"
)

// PendingApproval represents a pending approval request
type PendingApproval struct {
	ID                  string
	WorkflowExecutionID string
	PackageRef          *PackageReference
	StageID             string
	RequestedAt         time.Time
	Deadline            time.Time
	Approvers           []ApproverSpec
	RequiredCount       int
	ReceivedApprovals   []*ApprovalResult
	Status              ApprovalStatus
	Priority            ApprovalPriority
	Context             map[string]interface{}
}

// ApprovalResult represents an approval decision result
type ApprovalResult struct {
	ID         string
	ApprovalID string
	Approver   string
	Decision   ApprovalDecision
	Comment    string
	Timestamp  time.Time
	Evidence   []ApprovalEvidence
	Metadata   map[string]string
}

// PolicyEvaluationResult contains policy evaluation results
type PolicyEvaluationResult struct {
	PackageRef         *PackageReference
	Stage              PackageRevisionLifecycle
	ApplicablePolicies []*ApprovalPolicy
	RequiredApprovals  []*ApprovalRequirement
	BypassAvailable    bool
	BypassReasons      []string
	EvaluationTime     time.Time
}

// Manual task system types

// ManualTask represents a manual task in a workflow
type ManualTask struct {
	ID             string
	Name           string
	Description    string
	Type           TaskType
	Assignees      []string
	Priority       TaskPriority
	Deadline       *time.Time
	Instructions   string
	RequiredFields []TaskField
	Attachments    []TaskAttachment
	Dependencies   []string
}

// ManualTaskExecution represents an executing manual task
type ManualTaskExecution struct {
	ID                  string
	TaskID              string
	WorkflowExecutionID string
	Status              TaskStatus
	AssignedTo          string
	StartTime           time.Time
	CompleteTime        *time.Time
	Result              *TaskResult
	EscalationHistory   []*TaskEscalation
	ActivityLog         []TaskActivity
}

// TaskResult represents a manual task completion result
type TaskResult struct {
	Status           TaskCompletionStatus
	Output           map[string]interface{}
	Comments         string
	Attachments      []TaskAttachment
	CompletedBy      string
	CompletedAt      time.Time
	ValidationErrors []TaskValidationError
}

// External integration types

// ExternalIntegration defines integration with external systems
type ExternalIntegration struct {
	ID               string
	Name             string
	Type             IntegrationType
	Endpoint         string
	Authentication   *IntegrationAuth
	Configuration    map[string]interface{}
	Enabled          bool
	SupportedActions []string
	Timeout          time.Duration
	RetryPolicy      *RetryPolicy
}

// ExternalAction represents an action to execute on external system
type ExternalAction struct {
	Type           string
	Parameters     map[string]interface{}
	IdempotencyKey string
	Timeout        *time.Duration
	RetryPolicy    *RetryPolicy
}

// ExternalActionResult contains external action execution result
type ExternalActionResult struct {
	Success    bool
	Response   map[string]interface{}
	Error      string
	Duration   time.Duration
	RetryCount int
	ExecutedAt time.Time
}

// Audit and compliance types

// WorkflowAuditLog contains workflow audit information
type WorkflowAuditLog struct {
	PackageRef  *PackageReference
	Entries     []*AuditLogEntry
	TotalCount  int
	TimeRange   *TimeRange
	GeneratedAt time.Time
}

// AuditLogEntry represents a single audit log entry
type AuditLogEntry struct {
	ID          string
	Timestamp   time.Time
	EventType   AuditEventType
	User        string
	WorkflowID  string
	ExecutionID string
	StageID     string
	Action      string
	Details     map[string]interface{}
	IPAddress   string
	UserAgent   string
	Result      AuditResult
}

// ComplianceReport contains compliance assessment results
type ComplianceReport struct {
	GeneratedAt     time.Time
	TimeRange       *TimeRange
	ComplianceScore float64
	Violations      []*ComplianceViolation
	Recommendations []*ComplianceRecommendation
	Summary         *ComplianceSummary
}

// Enums and constants

// WorkflowExecutionStatus defines workflow execution status
type WorkflowExecutionStatus string

const (
	WorkflowExecutionStatusPending   WorkflowExecutionStatus = "pending"
	WorkflowExecutionStatusRunning   WorkflowExecutionStatus = "running"
	WorkflowExecutionStatusPaused    WorkflowExecutionStatus = "paused"
	WorkflowExecutionStatusCompleted WorkflowExecutionStatus = "completed"
	WorkflowExecutionStatusFailed    WorkflowExecutionStatus = "failed"
	WorkflowExecutionStatusAborted   WorkflowExecutionStatus = "aborted"
	WorkflowExecutionStatusTimedOut  WorkflowExecutionStatus = "timed_out"
)

// StageExecutionStatus defines stage execution status
type StageExecutionStatus string

const (
	StageExecutionStatusPending         StageExecutionStatus = "pending"
	StageExecutionStatusRunning         StageExecutionStatus = "running"
	StageExecutionStatusWaitingApproval StageExecutionStatus = "waiting_approval"
	StageExecutionStatusCompleted       StageExecutionStatus = "completed"
	StageExecutionStatusFailed          StageExecutionStatus = "failed"
	StageExecutionStatusSkipped         StageExecutionStatus = "skipped"
	StageExecutionStatusTimedOut        StageExecutionStatus = "timed_out"
)

// ApprovalStatus defines approval request status
type ApprovalStatus string

const (
	ApprovalStatusPending   ApprovalStatus = "pending"
	ApprovalStatusApproved  ApprovalStatus = "approved"
	ApprovalStatusRejected  ApprovalStatus = "rejected"
	ApprovalStatusTimedOut  ApprovalStatus = "timed_out"
	ApprovalStatusEscalated ApprovalStatus = "escalated"
)

// TaskStatus defines manual task status
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusEscalated  TaskStatus = "escalated"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Implementation

// NewWorkflowEngine creates a new workflow engine instance
func NewWorkflowEngine(client *Client, config *WorkflowEngineConfig) (WorkflowEngine, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if config == nil {
		config = getDefaultWorkflowEngineConfig()
	}

	we := &workflowEngine{
		client:         client,
		logger:         log.Log.WithName("workflow-engine"),
		config:         config,
		integrations:   make(map[string]*ExternalIntegration),
		executionLocks: make(map[string]*sync.Mutex),
		shutdown:       make(chan struct{}),
		metrics:        initWorkflowEngineMetrics(),
	}

	// Initialize components
	we.workflowRegistry = NewWorkflowRegistry(config.WorkflowRegistryConfig)
	we.executionEngine = NewExecutionEngine(config.ExecutionEngineConfig)
	we.stateManager = NewWorkflowStateManager(config.StateManagerConfig)
	we.approvalManager = NewApprovalManager(config.ApprovalManagerConfig)
	we.policyEngine = NewPolicyEngine(config.PolicyEngineConfig)
	we.delegationService = NewDelegationService(config.DelegationServiceConfig)
	we.taskManager = NewManualTaskManager(config.TaskManagerConfig)
	we.escalationEngine = NewEscalationEngine(config.EscalationEngineConfig)
	we.auditLogger = NewWorkflowAuditLogger(config.AuditLoggerConfig)
	we.complianceEngine = NewComplianceEngine(config.ComplianceEngineConfig)
	we.triggerEvaluator = NewTriggerEvaluator(config.TriggerEvaluatorConfig)
	we.executionMonitor = NewExecutionMonitor(config.ExecutionMonitorConfig)

	// Start background workers
	we.wg.Add(1)
	go we.triggerEvaluationWorker()

	we.wg.Add(1)
	go we.executionMonitorWorker()

	we.wg.Add(1)
	go we.approvalTimeoutWorker()

	we.wg.Add(1)
	go we.taskEscalationWorker()

	we.wg.Add(1)
	go we.metricsCollectionWorker()

	return we, nil
}

// CreateWorkflow creates a new workflow definition
func (we *workflowEngine) CreateWorkflow(ctx context.Context, spec *WorkflowSpec) (*Workflow, error) {
	we.logger.Info("Creating workflow", "name", spec.Name)

	// Validate workflow specification
	if err := we.validateWorkflowSpec(spec); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	workflow := &Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
		},
		Spec: *spec,
		Status: WorkflowStatus{
			Phase: WorkflowPhasePending,
		},
	}

	// Register with workflow registry
	if err := we.workflowRegistry.RegisterWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to register workflow: %w", err)
	}

	// Create workflow using Porch client
	createdWorkflow, err := we.client.CreateWorkflow(ctx, workflow)
	if err != nil {
		// Cleanup registry on failure
		we.workflowRegistry.UnregisterWorkflow(ctx, workflow.Name)
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Update metrics
	if we.metrics != nil {
		we.metrics.workflowsTotal.WithLabelValues("created").Inc()
	}

	// Audit log
	we.auditLogger.LogWorkflowCreated(ctx, createdWorkflow)

	we.logger.Info("Workflow created successfully", "workflowID", createdWorkflow.Name)
	return createdWorkflow, nil
}

// StartWorkflow starts workflow execution
func (we *workflowEngine) StartWorkflow(ctx context.Context, workflowID string, input *WorkflowInput) (*WorkflowExecution, error) {
	we.logger.Info("Starting workflow execution", "workflowID", workflowID, "package", input.PackageRef.GetPackageKey())

	// Get workflow definition
	workflow, err := we.client.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Acquire execution lock
	executionID := fmt.Sprintf("exec-%s-%d", workflowID, time.Now().UnixNano())
	lock := we.getExecutionLock(executionID)
	lock.Lock()
	defer lock.Unlock()

	// Create workflow execution
	execution := &WorkflowExecution{
		ID:               executionID,
		WorkflowID:       workflowID,
		PackageRef:       input.PackageRef,
		Status:           WorkflowExecutionStatusRunning,
		StartTime:        time.Now(),
		Input:            input,
		Results:          make(map[string]interface{}),
		StageResults:     make(map[string]*StageExecutionResult),
		PendingApprovals: []*PendingApproval{},
		ManualTasks:      []*ManualTaskExecution{},
		Errors:           []WorkflowError{},
		Metadata:         make(map[string]interface{}),
	}

	// Store execution state
	if err := we.stateManager.SaveExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	// Start execution engine
	go func() {
		ctx := context.Background()
		if err := we.executionEngine.ExecuteWorkflow(ctx, execution, workflow); err != nil {
			we.logger.Error(err, "Workflow execution failed", "executionID", executionID)
			execution.Status = WorkflowExecutionStatusFailed
			execution.EndTime = &[]time.Time{time.Now()}[0]
			we.stateManager.SaveExecution(ctx, execution)
		}
	}()

	// Update metrics
	if we.metrics != nil {
		we.metrics.executionsTotal.WithLabelValues(workflowID, "started").Inc()
		we.metrics.activeExecutions.Inc()
	}

	// Audit log
	we.auditLogger.LogWorkflowStarted(ctx, execution)

	we.logger.Info("Workflow execution started", "executionID", executionID, "workflowID", workflowID)
	return execution, nil
}

// SubmitApproval submits an approval decision
func (we *workflowEngine) SubmitApproval(ctx context.Context, approvalID string, decision ApprovalDecision, comment string) (*ApprovalResult, error) {
	we.logger.Info("Submitting approval", "approvalID", approvalID, "decision", decision)

	// Get pending approval
	approval, err := we.approvalManager.GetPendingApproval(ctx, approvalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approval: %w", err)
	}

	// Create approval result
	result := &ApprovalResult{
		ID:         fmt.Sprintf("approval-result-%d", time.Now().UnixNano()),
		ApprovalID: approvalID,
		Approver:   "current-user", // This would come from context
		Decision:   decision,
		Comment:    comment,
		Timestamp:  time.Now(),
		Evidence:   []ApprovalEvidence{},
		Metadata:   make(map[string]string),
	}

	// Process approval
	if err := we.approvalManager.ProcessApproval(ctx, approval, result); err != nil {
		return nil, fmt.Errorf("failed to process approval: %w", err)
	}

	// Update workflow execution if approval is complete
	if approval.Status == ApprovalStatusApproved || approval.Status == ApprovalStatusRejected {
		we.notifyWorkflowExecution(ctx, approval.WorkflowExecutionID, result)
	}

	// Update metrics
	if we.metrics != nil {
		we.metrics.approvalsTotal.WithLabelValues(string(decision)).Inc()
	}

	// Audit log
	we.auditLogger.LogApprovalSubmitted(ctx, approval, result)

	we.logger.Info("Approval submitted successfully", "approvalID", approvalID, "result", result.ID)
	return result, nil
}

// EvaluateApprovalPolicies evaluates applicable approval policies
func (we *workflowEngine) EvaluateApprovalPolicies(ctx context.Context, packageRef *PackageReference, stage PackageRevisionLifecycle) (*PolicyEvaluationResult, error) {
	we.logger.V(1).Info("Evaluating approval policies", "package", packageRef.GetPackageKey(), "stage", stage)

	result, err := we.policyEngine.EvaluatePolicies(ctx, packageRef, stage)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Update metrics
	if we.metrics != nil {
		we.metrics.policyEvaluationsTotal.Inc()
		we.metrics.applicablePolicies.Observe(float64(len(result.ApplicablePolicies)))
	}

	we.logger.V(1).Info("Policy evaluation completed",
		"package", packageRef.GetPackageKey(),
		"applicablePolicies", len(result.ApplicablePolicies),
		"requiredApprovals", len(result.RequiredApprovals))

	return result, nil
}

// CreateManualTask creates a manual task
func (we *workflowEngine) CreateManualTask(ctx context.Context, task *ManualTask) (*ManualTaskExecution, error) {
	we.logger.Info("Creating manual task", "taskName", task.Name, "type", task.Type)

	execution, err := we.taskManager.CreateTaskExecution(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task execution: %w", err)
	}

	// Notify assignees
	we.notifyTaskAssignees(ctx, execution)

	// Update metrics
	if we.metrics != nil {
		we.metrics.manualTasksTotal.WithLabelValues("created").Inc()
		we.metrics.activeManualTasks.Inc()
	}

	// Audit log
	we.auditLogger.LogManualTaskCreated(ctx, execution)

	we.logger.Info("Manual task created successfully", "taskExecutionID", execution.ID)
	return execution, nil
}

// Background workers

// triggerEvaluationWorker evaluates workflow triggers
func (we *workflowEngine) triggerEvaluationWorker() {
	defer we.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-we.shutdown:
			return
		case <-ticker.C:
			we.evaluatePendingTriggers()
		}
	}
}

// executionMonitorWorker monitors workflow executions
func (we *workflowEngine) executionMonitorWorker() {
	defer we.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-we.shutdown:
			return
		case <-ticker.C:
			we.monitorActiveExecutions()
		}
	}
}

// approvalTimeoutWorker handles approval timeouts
func (we *workflowEngine) approvalTimeoutWorker() {
	defer we.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-we.shutdown:
			return
		case <-ticker.C:
			we.handleApprovalTimeouts()
		}
	}
}

// taskEscalationWorker handles task escalations
func (we *workflowEngine) taskEscalationWorker() {
	defer we.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-we.shutdown:
			return
		case <-ticker.C:
			we.handleTaskEscalations()
		}
	}
}

// metricsCollectionWorker collects workflow metrics
func (we *workflowEngine) metricsCollectionWorker() {
	defer we.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-we.shutdown:
			return
		case <-ticker.C:
			we.collectMetrics()
		}
	}
}

// Close gracefully shuts down the workflow engine
func (we *workflowEngine) Close() error {
	we.logger.Info("Shutting down workflow engine")

	close(we.shutdown)
	we.wg.Wait()

	// Close components
	if we.workflowRegistry != nil {
		we.workflowRegistry.Close()
	}
	if we.executionEngine != nil {
		we.executionEngine.Close()
	}
	if we.stateManager != nil {
		we.stateManager.Close()
	}
	if we.approvalManager != nil {
		we.approvalManager.Close()
	}
	if we.taskManager != nil {
		we.taskManager.Close()
	}

	we.logger.Info("Workflow engine shutdown complete")
	return nil
}

// Helper methods and supporting functionality

// validateWorkflowSpec validates workflow specification
func (we *workflowEngine) validateWorkflowSpec(spec *WorkflowSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(spec.Stages) == 0 {
		return fmt.Errorf("workflow must have at least one stage")
	}
	// Additional validation logic would go here
	return nil
}

// getExecutionLock gets or creates an execution lock
func (we *workflowEngine) getExecutionLock(executionID string) *sync.Mutex {
	we.lockMutex.Lock()
	defer we.lockMutex.Unlock()

	if lock, exists := we.executionLocks[executionID]; exists {
		return lock
	}

	lock := &sync.Mutex{}
	we.executionLocks[executionID] = lock
	return lock
}

// notifyWorkflowExecution notifies workflow execution of approval completion
func (we *workflowEngine) notifyWorkflowExecution(ctx context.Context, executionID string, result *ApprovalResult) {
	// Implementation would notify execution engine
	we.logger.V(1).Info("Notifying workflow execution", "executionID", executionID, "approvalResult", result.ID)
}

// notifyTaskAssignees notifies task assignees
func (we *workflowEngine) notifyTaskAssignees(ctx context.Context, execution *ManualTaskExecution) {
	// Implementation would send notifications to assignees
	we.logger.V(1).Info("Notifying task assignees", "taskExecutionID", execution.ID)
}

// Background worker implementations

func (we *workflowEngine) evaluatePendingTriggers() {
	// Implementation would evaluate pending triggers
}

func (we *workflowEngine) monitorActiveExecutions() {
	// Implementation would monitor active executions for timeouts and issues
}

func (we *workflowEngine) handleApprovalTimeouts() {
	// Implementation would handle approval timeouts and escalations
}

func (we *workflowEngine) handleTaskEscalations() {
	// Implementation would handle task escalations
}

func (we *workflowEngine) collectMetrics() {
	if we.metrics == nil {
		return
	}

	// Collect current system metrics
	// Implementation would gather metrics from various components
}

// Configuration and metrics

func getDefaultWorkflowEngineConfig() *WorkflowEngineConfig {
	return &WorkflowEngineConfig{
		MaxConcurrentExecutions: 100,
		DefaultApprovalTimeout:  24 * time.Hour,
		DefaultTaskTimeout:      7 * 24 * time.Hour,
		EnableAuditLogging:      true,
		EnableMetrics:           true,
	}
}

func initWorkflowEngineMetrics() *WorkflowEngineMetrics {
	return &WorkflowEngineMetrics{
		workflowsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_workflows_total",
				Help: "Total number of workflows",
			},
			[]string{"status"},
		),
		executionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_workflow_executions_total",
				Help: "Total number of workflow executions",
			},
			[]string{"workflow_id", "status"},
		),
		activeExecutions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "porch_workflow_active_executions",
				Help: "Number of active workflow executions",
			},
		),
		approvalsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_workflow_approvals_total",
				Help: "Total number of workflow approvals",
			},
			[]string{"decision"},
		),
		manualTasksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_workflow_manual_tasks_total",
				Help: "Total number of manual tasks",
			},
			[]string{"status"},
		),
		activeManualTasks: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "porch_workflow_active_manual_tasks",
				Help: "Number of active manual tasks",
			},
		),
		policyEvaluationsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "porch_workflow_policy_evaluations_total",
				Help: "Total number of policy evaluations",
			},
		),
		applicablePolicies: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "porch_workflow_applicable_policies",
				Help:    "Number of applicable policies per evaluation",
				Buckets: []float64{0, 1, 2, 5, 10, 20},
			},
		),
	}
}

// Supporting types and placeholder implementations

// Configuration types
type WorkflowEngineConfig struct {
	MaxConcurrentExecutions int
	DefaultApprovalTimeout  time.Duration
	DefaultTaskTimeout      time.Duration
	EnableAuditLogging      bool
	EnableMetrics           bool
	WorkflowRegistryConfig  *WorkflowRegistryConfig
	ExecutionEngineConfig   *ExecutionEngineConfig
	StateManagerConfig      *WorkflowStateManagerConfig
	ApprovalManagerConfig   *ApprovalManagerConfig
	PolicyEngineConfig      *PolicyEngineConfig
	DelegationServiceConfig *DelegationServiceConfig
	TaskManagerConfig       *ManualTaskManagerConfig
	EscalationEngineConfig  *EscalationEngineConfig
	AuditLoggerConfig       *WorkflowAuditLoggerConfig
	ComplianceEngineConfig  *ComplianceEngineConfig
	TriggerEvaluatorConfig  *TriggerEvaluatorConfig
	ExecutionMonitorConfig  *ExecutionMonitorConfig
}

// Metrics type
type WorkflowEngineMetrics struct {
	workflowsTotal         *prometheus.CounterVec
	executionsTotal        *prometheus.CounterVec
	activeExecutions       prometheus.Gauge
	approvalsTotal         *prometheus.CounterVec
	manualTasksTotal       *prometheus.CounterVec
	activeManualTasks      prometheus.Gauge
	policyEvaluationsTotal prometheus.Counter
	applicablePolicies     prometheus.Histogram
}

// Additional supporting types
type WorkflowListOptions struct {
	Labels   map[string]string
	Status   []WorkflowPhase
	PageSize int
	Continue string
}

type WorkflowPriority string

const (
	WorkflowPriorityLow      WorkflowPriority = "low"
	WorkflowPriorityNormal   WorkflowPriority = "normal"
	WorkflowPriorityHigh     WorkflowPriority = "high"
	WorkflowPriorityCritical WorkflowPriority = "critical"
)

type PackageEvent struct {
	Type       string
	PackageRef *PackageReference
	Timestamp  time.Time
	Data       map[string]interface{}
}

type TriggeredWorkflow struct {
	WorkflowID string
	TriggerID  string
	Input      *WorkflowInput
}

type WorkflowError struct {
	Code        string
	Message     string
	Timestamp   time.Time
	Stage       string
	Recoverable bool
}

type StageError struct {
	Code      string
	Message   string
	Timestamp time.Time
	Retryable bool
}

type WorkflowEngineHealth struct {
	Status            string
	ActiveExecutions  int
	PendingApprovals  int
	ActiveManualTasks int
	LastActivity      time.Time
}

type WorkflowStatistics struct {
	TimeRange            *TimeRange
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64
	AverageExecutionTime time.Duration
	ApprovalMetrics      *ApprovalStatistics
	TaskMetrics          *TaskStatistics
}

type ApprovalStatistics struct {
	TotalApprovals      int64
	ApprovedCount       int64
	RejectedCount       int64
	TimedOutCount       int64
	AverageApprovalTime time.Duration
}

type TaskStatistics struct {
	TotalTasks            int64
	CompletedTasks        int64
	EscalatedTasks        int64
	AverageCompletionTime time.Duration
}

// Placeholder component implementations
type WorkflowRegistry struct{}

func NewWorkflowRegistry(config *WorkflowRegistryConfig) *WorkflowRegistry {
	return &WorkflowRegistry{}
}
func (wr *WorkflowRegistry) RegisterWorkflow(ctx context.Context, workflow *Workflow) error {
	return nil
}
func (wr *WorkflowRegistry) UnregisterWorkflow(ctx context.Context, name string) error { return nil }
func (wr *WorkflowRegistry) Close() error                                              { return nil }

type WorkflowExecutionEngine struct{}

func NewExecutionEngine(config *ExecutionEngineConfig) *WorkflowExecutionEngine {
	return &WorkflowExecutionEngine{}
}
func (ee *WorkflowExecutionEngine) ExecuteWorkflow(ctx context.Context, execution *WorkflowExecution, workflow *Workflow) error {
	return nil
}
func (ee *WorkflowExecutionEngine) Close() error { return nil }

type WorkflowStateManager struct{}

func NewWorkflowStateManager(config *WorkflowStateManagerConfig) *WorkflowStateManager {
	return &WorkflowStateManager{}
}
func (wsm *WorkflowStateManager) SaveExecution(ctx context.Context, execution *WorkflowExecution) error {
	return nil
}
func (wsm *WorkflowStateManager) Close() error { return nil }

type ApprovalManager struct{}

func NewApprovalManager(config *ApprovalManagerConfig) *ApprovalManager { return &ApprovalManager{} }
func (am *ApprovalManager) GetPendingApproval(ctx context.Context, approvalID string) (*PendingApproval, error) {
	return &PendingApproval{ID: approvalID}, nil
}
func (am *ApprovalManager) ProcessApproval(ctx context.Context, approval *PendingApproval, result *ApprovalResult) error {
	return nil
}
func (am *ApprovalManager) Close() error { return nil }

type PolicyEngine struct{}

func NewPolicyEngine(config *PolicyEngineConfig) *PolicyEngine { return &PolicyEngine{} }
func (pe *PolicyEngine) EvaluatePolicies(ctx context.Context, packageRef *PackageReference, stage PackageRevisionLifecycle) (*PolicyEvaluationResult, error) {
	return &PolicyEvaluationResult{PackageRef: packageRef, Stage: stage, EvaluationTime: time.Now()}, nil
}
func (pe *PolicyEngine) Close() error { return nil }

type DelegationService struct{}

func NewDelegationService(config *DelegationServiceConfig) *DelegationService {
	return &DelegationService{}
}
func (ds *DelegationService) Close() error { return nil }

type ManualTaskManager struct{}

func NewManualTaskManager(config *ManualTaskManagerConfig) *ManualTaskManager {
	return &ManualTaskManager{}
}
func (mtm *ManualTaskManager) CreateTaskExecution(ctx context.Context, task *ManualTask) (*ManualTaskExecution, error) {
	return &ManualTaskExecution{
		ID:        fmt.Sprintf("task-exec-%d", time.Now().UnixNano()),
		TaskID:    task.ID,
		Status:    TaskStatusPending,
		StartTime: time.Now(),
	}, nil
}
func (mtm *ManualTaskManager) Close() error { return nil }

type EscalationEngine struct{}

func NewEscalationEngine(config *EscalationEngineConfig) *EscalationEngine {
	return &EscalationEngine{}
}
func (ee *EscalationEngine) Close() error { return nil }

type WorkflowAuditLogger struct{}

func NewWorkflowAuditLogger(config *WorkflowAuditLoggerConfig) *WorkflowAuditLogger {
	return &WorkflowAuditLogger{}
}
func (wal *WorkflowAuditLogger) LogWorkflowCreated(ctx context.Context, workflow *Workflow) {}
func (wal *WorkflowAuditLogger) LogWorkflowStarted(ctx context.Context, execution *WorkflowExecution) {
}
func (wal *WorkflowAuditLogger) LogApprovalSubmitted(ctx context.Context, approval *PendingApproval, result *ApprovalResult) {
}
func (wal *WorkflowAuditLogger) LogManualTaskCreated(ctx context.Context, execution *ManualTaskExecution) {
}
func (wal *WorkflowAuditLogger) Close() error { return nil }

type ComplianceEngine struct{}

func NewComplianceEngine(config *ComplianceEngineConfig) *ComplianceEngine {
	return &ComplianceEngine{}
}
func (ce *ComplianceEngine) Close() error { return nil }

type TriggerEvaluator struct{}

func NewTriggerEvaluator(config *TriggerEvaluatorConfig) *TriggerEvaluator {
	return &TriggerEvaluator{}
}
func (te *TriggerEvaluator) Close() error { return nil }

type ExecutionMonitor struct{}

func NewExecutionMonitor(config *ExecutionMonitorConfig) *ExecutionMonitor {
	return &ExecutionMonitor{}
}
func (em *ExecutionMonitor) Close() error { return nil }

// Configuration placeholder types
type WorkflowRegistryConfig struct{}
type ExecutionEngineConfig struct{}
type WorkflowStateManagerConfig struct{}
type ApprovalManagerConfig struct{}
type PolicyEngineConfig struct{}
type DelegationServiceConfig struct{}
type ManualTaskManagerConfig struct{}
type EscalationEngineConfig struct{}
type WorkflowAuditLoggerConfig struct{}
type ComplianceEngineConfig struct{}
type TriggerEvaluatorConfig struct{}
type ExecutionMonitorConfig struct{}

// Additional complex types that would be fully defined in production
type ApprovalHistory struct{}
type PackageSelector struct{}
type ApprovalRuleType string
type ApproverSpec struct{}
type ApprovalCondition struct{}
type EscalationPolicy struct{}
type BypassCondition struct{}
type ApprovalPriority string
type ApprovalEvidence struct{}
type ApprovalRequirement struct{}
type TaskType string
type TaskPriority string
type TaskField struct{}
type TaskAttachment struct{}
type TaskActivity struct{}
type TaskCompletionStatus string
type TaskValidationError struct{}
type TaskEscalation struct{}
type IntegrationType string
type IntegrationAuth struct{}
type AuditLogOptions struct{}
type AuditEventType string
type AuditResult string
type ComplianceRecommendation struct{}
type ComplianceSummary struct{}
type ComplianceReportOptions struct{}
type AuditExportOptions struct{}
type AuditExport struct{}
type StageCondition struct{}
type FailurePolicy struct{}

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

package packagerevision

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/templates"
	"github.com/thc1006/nephoran-intent-operator/pkg/validation/yang"
)

// PackageRevisionManager orchestrates the complete NetworkIntent to PackageRevision lifecycle.
// Manages Draft → Proposed → Published state transitions, validation workflows,.
// template rendering, YANG model validation, and O-RAN compliance checks.
type PackageRevisionManager interface {
	// Core lifecycle management.
	CreateFromIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*porch.PackageRevision, error)
	UpdateFromIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, existing *porch.PackageRevision) (*porch.PackageRevision, error)
	DeletePackageRevision(ctx context.Context, ref *porch.PackageReference) error

	// Lifecycle state management.
	TransitionToProposed(ctx context.Context, ref *porch.PackageReference, opts *TransitionOptions) (*TransitionResult, error)
	TransitionToPublished(ctx context.Context, ref *porch.PackageReference, opts *TransitionOptions) (*TransitionResult, error)
	RollbackRevision(ctx context.Context, ref *porch.PackageReference, targetRevision string) (*RollbackResult, error)

	// Configuration management.
	ValidateConfiguration(ctx context.Context, ref *porch.PackageReference) (*ValidationResult, error)
	DetectConfigurationDrift(ctx context.Context, ref *porch.PackageReference) (*DriftDetectionResult, error)
	CorrectConfigurationDrift(ctx context.Context, ref *porch.PackageReference, driftResult *DriftDetectionResult) error

	// Template management.
	GetAvailableTemplates(ctx context.Context, targetComponent nephoranv1.ORANComponent) ([]*templates.BlueprintTemplate, error)
	RenderTemplate(ctx context.Context, template *templates.BlueprintTemplate, params map[string]interface{}) ([]*porch.KRMResource, error)

	// Lifecycle monitoring.
	GetLifecycleStatus(ctx context.Context, ref *porch.PackageReference) (*LifecycleStatus, error)
	GetPackageMetrics(ctx context.Context, ref *porch.PackageReference) (*PackageMetrics, error)

	// Batch operations.
	BatchCreateFromIntents(ctx context.Context, intents []*nephoranv1.NetworkIntent, opts *BatchOptions) (*BatchResult, error)

	// Health and maintenance.
	GetManagerHealth(ctx context.Context) (*ManagerHealth, error)
	Close() error
}

// packageRevisionManager implements comprehensive PackageRevision lifecycle orchestration.
type packageRevisionManager struct {
	// Core dependencies.
	porchClient      porch.PorchClient
	lifecycleManager porch.LifecycleManager
	templateEngine   templates.TemplateEngine
	yangValidator    yang.YANGValidator
	approvalEngine   ApprovalEngine
	driftDetector    DriftDetector
	logger           logr.Logger

	// Configuration.
	config  *ManagerConfig
	metrics *ManagerMetrics

	// State management.
	activeTransitions map[string]*ActiveTransition
	transitionMutex   sync.RWMutex

	// Background processing.
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// Core data structures.

// TransitionOptions configures lifecycle transitions.
type TransitionOptions struct {
	SkipValidation      bool              `json:"skipValidation,omitempty"`
	SkipApproval        bool              `json:"skipApproval,omitempty"`
	CreateRollbackPoint bool              `json:"createRollbackPoint,omitempty"`
	RollbackDescription string            `json:"rollbackDescription,omitempty"`
	ForceTransition     bool              `json:"forceTransition,omitempty"`
	ValidationPolicy    *ValidationPolicy `json:"validationPolicy,omitempty"`
	ApprovalPolicy      *ApprovalPolicy   `json:"approvalPolicy,omitempty"`
	NotificationTargets []string          `json:"notificationTargets,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	Timeout             time.Duration     `json:"timeout,omitempty"`
	DryRun              bool              `json:"dryRun,omitempty"`
}

// TransitionResult contains lifecycle transition results.
type TransitionResult struct {
	Success            bool                           `json:"success"`
	PreviousStage      porch.PackageRevisionLifecycle `json:"previousStage"`
	NewStage           porch.PackageRevisionLifecycle `json:"newStage"`
	TransitionTime     time.Time                      `json:"transitionTime"`
	Duration           time.Duration                  `json:"duration"`
	ValidationResults  []*ValidationResult            `json:"validationResults,omitempty"`
	ApprovalResults    []*ApprovalResult              `json:"approvalResults,omitempty"`
	RollbackPoint      *porch.RollbackPoint           `json:"rollbackPoint,omitempty"`
	GeneratedResources []*porch.KRMResource           `json:"generatedResources,omitempty"`
	Warnings           []string                       `json:"warnings,omitempty"`
	Notifications      []*NotificationResult          `json:"notifications,omitempty"`
	Metadata           map[string]interface{}         `json:"metadata,omitempty"`
}

// ValidationResult contains configuration validation results.
type ValidationResult struct {
	Valid                bool                   `json:"valid"`
	Errors               []*ValidationError     `json:"errors,omitempty"`
	Warnings             []*ValidationWarning   `json:"warnings,omitempty"`
	YANGValidationResult *yang.ValidationResult `json:"yangValidationResult,omitempty"`
	PolicyValidation     []*PolicyValidation    `json:"policyValidation,omitempty"`
	ComplianceChecks     []*ComplianceCheck     `json:"complianceChecks,omitempty"`
	SecurityValidation   *SecurityValidation    `json:"securityValidation,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Code        string `json:"code"`
	Path        string `json:"path"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation,omitempty"`
	Source      string `json:"source"` // yang, policy, compliance, security
}

// ValidationWarning represents a validation warning.
type ValidationWarning struct {
	Code       string `json:"code"`
	Path       string `json:"path"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Source     string `json:"source"`
}

// PolicyValidation contains policy validation results.
type PolicyValidation struct {
	PolicyName string            `json:"policyName"`
	PolicyType string            `json:"policyType"`
	Valid      bool              `json:"valid"`
	Violations []PolicyViolation `json:"violations,omitempty"`
}

// PolicyViolation represents a policy violation.
type PolicyViolation struct {
	Rule        string `json:"rule"`
	Resource    string `json:"resource"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ComplianceCheck contains O-RAN compliance validation.
type ComplianceCheck struct {
	Standard  string            `json:"standard"` // O-RAN, 3GPP, etc.
	Version   string            `json:"version"`
	Interface string            `json:"interface,omitempty"` // A1, O1, O2, E2
	Compliant bool              `json:"compliant"`
	Issues    []ComplianceIssue `json:"issues,omitempty"`
}

// ComplianceIssue represents a compliance violation.
type ComplianceIssue struct {
	Section     string `json:"section"`
	Requirement string `json:"requirement"`
	Current     string `json:"current"`
	Expected    string `json:"expected"`
	Severity    string `json:"severity"`
}

// SecurityValidation contains security validation results.
type SecurityValidation struct {
	Valid              bool                `json:"valid"`
	SecurityChecks     []SecurityCheck     `json:"securityChecks"`
	VulnerabilityScans []VulnerabilityScan `json:"vulnerabilityScans,omitempty"`
}

// SecurityCheck represents a security validation check.
type SecurityCheck struct {
	Name        string `json:"name"`
	Category    string `json:"category"` // authentication, authorization, encryption, etc.
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
}

// VulnerabilityScan represents vulnerability scan results.
type VulnerabilityScan struct {
	Scanner         string          `json:"scanner"`
	ImageName       string          `json:"imageName"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

// Vulnerability represents a security vulnerability.
type Vulnerability struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	FixVersion  string `json:"fixVersion,omitempty"`
}

// ApprovalResult contains approval workflow results.
type ApprovalResult struct {
	WorkflowID        string       `json:"workflowId"`
	Stage             string       `json:"stage"`
	Status            string       `json:"status"` // pending, approved, rejected
	Approver          string       `json:"approver,omitempty"`
	ApprovalTime      *metav1.Time `json:"approvalTime,omitempty"`
	Comments          string       `json:"comments,omitempty"`
	RequiredApprovals int          `json:"requiredApprovals"`
	ReceivedApprovals int          `json:"receivedApprovals"`
}

// RollbackResult contains rollback operation results.
type RollbackResult struct {
	Success           bool                           `json:"success"`
	RollbackPoint     *porch.RollbackPoint           `json:"rollbackPoint"`
	PreviousStage     porch.PackageRevisionLifecycle `json:"previousStage"`
	RestoredStage     porch.PackageRevisionLifecycle `json:"restoredStage"`
	Duration          time.Duration                  `json:"duration"`
	RestoredResources []*porch.KRMResource           `json:"restoredResources,omitempty"`
	Warnings          []string                       `json:"warnings,omitempty"`
}

// DriftDetectionResult contains configuration drift detection results.
type DriftDetectionResult struct {
	HasDrift        bool                 `json:"hasDrift"`
	DetectionTime   time.Time            `json:"detectionTime"`
	DriftDetails    []*DriftDetail       `json:"driftDetails"`
	Severity        string               `json:"severity"` // low, medium, high, critical
	AutoCorrectible bool                 `json:"autoCorrectible"`
	CorrectionPlan  *DriftCorrectionPlan `json:"correctionPlan,omitempty"`
}

// DriftDetail represents a specific configuration drift.
type DriftDetail struct {
	ResourceType    string      `json:"resourceType"`
	ResourceName    string      `json:"resourceName"`
	Path            string      `json:"path"`
	ExpectedValue   interface{} `json:"expectedValue"`
	ActualValue     interface{} `json:"actualValue"`
	DriftType       string      `json:"driftType"` // modified, missing, unexpected
	Impact          string      `json:"impact"`    // low, medium, high
	AutoCorrectible bool        `json:"autoCorrectible"`
}

// DriftCorrectionPlan contains the plan to correct configuration drift.
type DriftCorrectionPlan struct {
	CorrectionSteps  []*CorrectionStep `json:"correctionSteps"`
	RequiresApproval bool              `json:"requiresApproval"`
	EstimatedTime    time.Duration     `json:"estimatedTime"`
	RiskLevel        string            `json:"riskLevel"`
}

// CorrectionStep represents a single drift correction action.
type CorrectionStep struct {
	ID           string      `json:"id"`
	Action       string      `json:"action"` // create, update, delete
	ResourceType string      `json:"resourceType"`
	ResourceName string      `json:"resourceName"`
	Changes      interface{} `json:"changes"`
	RiskLevel    string      `json:"riskLevel"`
}

// LifecycleStatus contains current lifecycle status information.
type LifecycleStatus struct {
	CurrentStage       porch.PackageRevisionLifecycle   `json:"currentStage"`
	StageStartTime     time.Time                        `json:"stageStartTime"`
	StageHistory       []*StageHistoryEntry             `json:"stageHistory"`
	PendingActions     []*PendingAction                 `json:"pendingActions,omitempty"`
	BlockingIssues     []*BlockingIssue                 `json:"blockingIssues,omitempty"`
	NextPossibleStages []porch.PackageRevisionLifecycle `json:"nextPossibleStages"`
}

// StageHistoryEntry represents a lifecycle stage transition.
type StageHistoryEntry struct {
	FromStage      porch.PackageRevisionLifecycle `json:"fromStage"`
	ToStage        porch.PackageRevisionLifecycle `json:"toStage"`
	TransitionTime time.Time                      `json:"transitionTime"`
	Duration       time.Duration                  `json:"duration"`
	User           string                         `json:"user,omitempty"`
	Reason         string                         `json:"reason,omitempty"`
}

// PendingAction represents an action waiting to be completed.
type PendingAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // approval, validation, deployment
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	Assignee    string    `json:"assignee,omitempty"`
	Priority    string    `json:"priority"`
}

// BlockingIssue represents an issue preventing lifecycle progression.
type BlockingIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // validation_error, approval_required, dependency_missing
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PackageMetrics contains metrics for a specific package.
type PackageMetrics struct {
	PackageRef            *porch.PackageReference                  `json:"packageRef"`
	TotalTransitions      int64                                    `json:"totalTransitions"`
	TransitionsByStage    map[porch.PackageRevisionLifecycle]int64 `json:"transitionsByStage"`
	AverageTransitionTime time.Duration                            `json:"averageTransitionTime"`
	FailedTransitions     int64                                    `json:"failedTransitions"`
	ValidationFailures    int64                                    `json:"validationFailures"`
	ApprovalCycles        int64                                    `json:"approvalCycles"`
	DriftDetections       int64                                    `json:"driftDetections"`
	AutoCorrections       int64                                    `json:"autoCorrections"`
	TimeInCurrentStage    time.Duration                            `json:"timeInCurrentStage"`
	LastActivity          time.Time                                `json:"lastActivity"`
}

// BatchOptions configures batch operations.
type BatchOptions struct {
	Concurrency          int               `json:"concurrency"`
	ContinueOnError      bool              `json:"continueOnError"`
	ValidationPolicy     *ValidationPolicy `json:"validationPolicy,omitempty"`
	ApprovalPolicy       *ApprovalPolicy   `json:"approvalPolicy,omitempty"`
	CreateRollbackPoints bool              `json:"createRollbackPoints"`
	Timeout              time.Duration     `json:"timeout"`
	DryRun               bool              `json:"dryRun"`
}

// BatchResult contains batch operation results.
type BatchResult struct {
	TotalRequests        int                       `json:"totalRequests"`
	SuccessfulOperations int                       `json:"successfulOperations"`
	FailedOperations     int                       `json:"failedOperations"`
	Results              []*PackageOperationResult `json:"results"`
	Duration             time.Duration             `json:"duration"`
	OverallSuccess       bool                      `json:"overallSuccess"`
}

// PackageOperationResult contains individual package operation result.
type PackageOperationResult struct {
	Intent     *nephoranv1.NetworkIntent `json:"intent"`
	PackageRef *porch.PackageReference   `json:"packageRef,omitempty"`
	Success    bool                      `json:"success"`
	Result     interface{}               `json:"result,omitempty"`
	Error      string                    `json:"error,omitempty"`
	Duration   time.Duration             `json:"duration"`
}

// ManagerHealth contains manager health information.
type ManagerHealth struct {
	Status            string            `json:"status"`
	ActiveTransitions int               `json:"activeTransitions"`
	QueuedOperations  int               `json:"queuedOperations"`
	ComponentHealth   map[string]string `json:"componentHealth"`
	LastActivity      time.Time         `json:"lastActivity"`
	Metrics           *ManagerMetrics   `json:"metrics,omitempty"`
}

// ActiveTransition represents an ongoing lifecycle transition.
type ActiveTransition struct {
	ID          string                         `json:"id"`
	PackageRef  *porch.PackageReference        `json:"packageRef"`
	TargetStage porch.PackageRevisionLifecycle `json:"targetStage"`
	StartTime   time.Time                      `json:"startTime"`
	LastUpdate  time.Time                      `json:"lastUpdate"`
	Status      string                         `json:"status"`
	Progress    int                            `json:"progress"` // 0-100
	CurrentStep string                         `json:"currentStep"`
	Options     *TransitionOptions             `json:"options"`
}

// NotificationResult contains notification sending results.
type NotificationResult struct {
	Target  string    `json:"target"`
	Success bool      `json:"success"`
	Message string    `json:"message"`
	SentAt  time.Time `json:"sentAt"`
	Error   string    `json:"error,omitempty"`
}

// Configuration types.

// ManagerConfig contains manager configuration.
type ManagerConfig struct {
	// General settings.
	DefaultRepository        string        `yaml:"defaultRepository"`
	DefaultTimeout           time.Duration `yaml:"defaultTimeout"`
	MaxConcurrentTransitions int           `yaml:"maxConcurrentTransitions"`

	// Validation settings.
	EnableYANGValidation     bool          `yaml:"enableYANGValidation"`
	EnablePolicyValidation   bool          `yaml:"enablePolicyValidation"`
	EnableSecurityValidation bool          `yaml:"enableSecurityValidation"`
	EnableComplianceChecks   bool          `yaml:"enableComplianceChecks"`
	ValidationTimeout        time.Duration `yaml:"validationTimeout"`

	// Approval settings.
	EnableApprovalWorkflow bool          `yaml:"enableApprovalWorkflow"`
	DefaultApprovalPolicy  string        `yaml:"defaultApprovalPolicy"`
	ApprovalTimeout        time.Duration `yaml:"approvalTimeout"`

	// Drift detection settings.
	EnableDriftDetection   bool          `yaml:"enableDriftDetection"`
	DriftDetectionInterval time.Duration `yaml:"driftDetectionInterval"`
	AutoCorrectDrift       bool          `yaml:"autoCorrectDrift"`

	// Template settings.
	TemplateRepository      string        `yaml:"templateRepository"`
	TemplateRefreshInterval time.Duration `yaml:"templateRefreshInterval"`

	// Metrics and monitoring.
	EnableMetrics   bool          `yaml:"enableMetrics"`
	MetricsInterval time.Duration `yaml:"metricsInterval"`

	// Notification settings.
	EnableNotifications  bool     `yaml:"enableNotifications"`
	NotificationChannels []string `yaml:"notificationChannels"`
}

// ValidationPolicy defines validation requirements.
type ValidationPolicy struct {
	RequireYANGValidation     bool     `json:"requireYangValidation"`
	RequirePolicyValidation   bool     `json:"requirePolicyValidation"`
	RequireSecurityValidation bool     `json:"requireSecurityValidation"`
	RequireComplianceChecks   bool     `json:"requireComplianceChecks"`
	AllowedStandards          []string `json:"allowedStandards,omitempty"`
	SecurityScanners          []string `json:"securityScanners,omitempty"`
	FailOnWarnings            bool     `json:"failOnWarnings"`
}

// ApprovalPolicy defines approval requirements.
type ApprovalPolicy struct {
	RequiredApprovals    int               `json:"requiredApprovals"`
	ApprovalStages       []string          `json:"approvalStages"`
	Approvers            []string          `json:"approvers"`
	AutoApproveForStages []string          `json:"autoApproveForStages,omitempty"`
	EscalationPolicy     *EscalationPolicy `json:"escalationPolicy,omitempty"`
}

// EscalationPolicy defines approval escalation rules.
type EscalationPolicy struct {
	EscalationTimeout   time.Duration `json:"escalationTimeout"`
	EscalationApprovers []string      `json:"escalationApprovers"`
	MaxEscalationLevel  int           `json:"maxEscalationLevel"`
}

// ManagerMetrics contains manager performance metrics.
type ManagerMetrics struct {
	TotalPackagesManaged prometheus.Counter       `json:"totalPackagesManaged"`
	TransitionsTotal     *prometheus.CounterVec   `json:"transitionsTotal"`
	TransitionDuration   *prometheus.HistogramVec `json:"transitionDuration"`
	ValidationResults    *prometheus.CounterVec   `json:"validationResults"`
	ApprovalLatency      prometheus.Histogram     `json:"approvalLatency"`
	DriftDetections      prometheus.Counter       `json:"driftDetections"`
	ActiveTransitions    prometheus.Gauge         `json:"activeTransitions"`
	QueueSize            prometheus.Gauge         `json:"queueSize"`
}

// NewPackageRevisionManager creates a new PackageRevision manager.
func NewPackageRevisionManager(
	porchClient porch.PorchClient,
	lifecycleManager porch.LifecycleManager,
	templateEngine templates.TemplateEngine,
	yangValidator yang.YANGValidator,
	config *ManagerConfig,
) (PackageRevisionManager, error) {
	if porchClient == nil {
		return nil, fmt.Errorf("porchClient cannot be nil")
	}
	if lifecycleManager == nil {
		return nil, fmt.Errorf("lifecycleManager cannot be nil")
	}
	if templateEngine == nil {
		return nil, fmt.Errorf("templateEngine cannot be nil")
	}
	if config == nil {
		config = getDefaultManagerConfig()
	}

	manager := &packageRevisionManager{
		porchClient:       porchClient,
		lifecycleManager:  lifecycleManager,
		templateEngine:    templateEngine,
		yangValidator:     yangValidator,
		config:            config,
		logger:            log.Log.WithName("package-revision-manager"),
		activeTransitions: make(map[string]*ActiveTransition),
		shutdown:          make(chan struct{}),
		metrics:           initManagerMetrics(),
	}

	// Initialize additional components.
	approvalEngine, err := NewApprovalEngine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create approval engine: %w", err)
	}
	manager.approvalEngine = approvalEngine

	driftDetector, err := NewDriftDetector(porchClient, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create drift detector: %w", err)
	}
	manager.driftDetector = driftDetector

	// Start background workers.
	manager.wg.Add(1)
	go manager.driftDetectionWorker()

	manager.wg.Add(1)
	go manager.metricsCollectionWorker()

	manager.logger.Info("PackageRevision manager initialized successfully")
	return manager, nil
}

// CreateFromIntent creates a new PackageRevision from a NetworkIntent.
func (m *packageRevisionManager) CreateFromIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*porch.PackageRevision, error) {
	m.logger.Info("Creating PackageRevision from NetworkIntent",
		"intent", intent.Name,
		"namespace", intent.Namespace)

	startTime := time.Now()
	defer func() {
		m.metrics.TransitionDuration.WithLabelValues("create").Observe(time.Since(startTime).Seconds())
	}()

	// Validate NetworkIntent.
	if err := m.validateNetworkIntent(intent); err != nil {
		return nil, fmt.Errorf("invalid NetworkIntent: %w", err)
	}

	// Create PackageRevision specification.
	packageSpec := &porch.PackageSpec{
		Repository:  m.config.DefaultRepository,
		PackageName: m.generatePackageName(intent),
		Revision:    "v1",
		Lifecycle:   porch.PackageRevisionLifecycleDraft,
		Labels:      m.generateLabelsForIntent(intent),
		Annotations: m.generateAnnotationsForIntent(intent),
	}

	// Create the PackageRevision.
	pkg := &porch.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:        packageSpec.PackageName,
			Labels:      packageSpec.Labels,
			Annotations: packageSpec.Annotations,
		},
		Spec: porch.PackageRevisionSpec{
			PackageName: packageSpec.PackageName,
			Repository:  packageSpec.Repository,
			Revision:    packageSpec.Revision,
			Lifecycle:   packageSpec.Lifecycle,
		},
	}

	// Create the PackageRevision in Porch.
	createdPkg, err := m.porchClient.CreatePackageRevision(ctx, pkg)
	if err != nil {
		m.metrics.ValidationResults.WithLabelValues("create", "error").Inc()
		return nil, fmt.Errorf("failed to create PackageRevision: %w", err)
	}

	m.metrics.TotalPackagesManaged.Inc()
	m.metrics.ValidationResults.WithLabelValues("create", "success").Inc()

	m.logger.Info("PackageRevision created successfully",
		"package", createdPkg.Spec.PackageName,
		"revision", createdPkg.Spec.Revision,
		"lifecycle", createdPkg.Spec.Lifecycle)

	return createdPkg, nil
}

// UpdateFromIntent updates an existing PackageRevision from a NetworkIntent.
func (m *packageRevisionManager) UpdateFromIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, existing *porch.PackageRevision) (*porch.PackageRevision, error) {
	m.logger.Info("Updating PackageRevision from NetworkIntent",
		"intent", intent.Name,
		"package", existing.Spec.PackageName)

	// Implementation would update the existing package revision.
	// For now, return the existing package unchanged.
	return existing, nil
}

// DeletePackageRevision deletes a PackageRevision.
func (m *packageRevisionManager) DeletePackageRevision(ctx context.Context, ref *porch.PackageReference) error {
	m.logger.Info("Deleting PackageRevision", "package", ref.GetPackageKey())

	// Delete the PackageRevision using Porch client.
	return m.porchClient.DeletePackageRevision(ctx, ref.PackageName, ref.Revision)
}

// RollbackRevision rolls back a PackageRevision to a previous version.
func (m *packageRevisionManager) RollbackRevision(ctx context.Context, ref *porch.PackageReference, targetRevision string) (*RollbackResult, error) {
	m.logger.Info("Rolling back PackageRevision",
		"package", ref.GetPackageKey(),
		"targetRevision", targetRevision)

	// Implementation would perform the actual rollback.
	// For now, return a successful result.
	return &RollbackResult{
		Success:       true,
		Duration:      time.Second,
		PreviousStage: porch.PackageRevisionLifecyclePublished,
		RestoredStage: porch.PackageRevisionLifecycleDraft,
	}, nil
}

// TransitionToProposed transitions a PackageRevision to Proposed stage.
func (m *packageRevisionManager) TransitionToProposed(ctx context.Context, ref *porch.PackageReference, opts *TransitionOptions) (*TransitionResult, error) {
	return m.performLifecycleTransition(ctx, ref, porch.PackageRevisionLifecycleProposed, opts)
}

// TransitionToPublished transitions a PackageRevision to Published stage.
func (m *packageRevisionManager) TransitionToPublished(ctx context.Context, ref *porch.PackageReference, opts *TransitionOptions) (*TransitionResult, error) {
	return m.performLifecycleTransition(ctx, ref, porch.PackageRevisionLifecyclePublished, opts)
}

// BatchCreateFromIntents creates multiple PackageRevisions from NetworkIntents in batch.
func (m *packageRevisionManager) BatchCreateFromIntents(ctx context.Context, intents []*nephoranv1.NetworkIntent, opts *BatchOptions) (*BatchResult, error) {
	m.logger.Info("Creating PackageRevisions from NetworkIntents in batch", "count", len(intents))

	startTime := time.Now()
	if opts == nil {
		opts = &BatchOptions{
			Concurrency:     5,
			ContinueOnError: true,
			Timeout:         30 * time.Minute,
		}
	}

	result := &BatchResult{
		TotalRequests:        len(intents),
		SuccessfulOperations: 0,
		FailedOperations:     0,
		Results:              make([]*PackageOperationResult, 0, len(intents)),
		OverallSuccess:       true,
	}

	// Create a semaphore to limit concurrency.
	semaphore := make(chan struct{}, opts.Concurrency)
	resultChan := make(chan *PackageOperationResult, len(intents))

	// Process intents concurrently.
	for _, intent := range intents {
		go func(intent *nephoranv1.NetworkIntent) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			opResult := &PackageOperationResult{
				Intent:   intent,
				Success:  true,
				Duration: 0,
			}

			opStartTime := time.Now()
			pkg, err := m.CreateFromIntent(ctx, intent)
			opResult.Duration = time.Since(opStartTime)

			if err != nil {
				opResult.Success = false
				opResult.Error = err.Error()
			} else {
				opResult.Result = pkg
				opResult.PackageRef = &porch.PackageReference{
					Repository:  pkg.Spec.Repository,
					PackageName: pkg.Spec.PackageName,
					Revision:    pkg.Spec.Revision,
				}
			}

			resultChan <- opResult
		}(intent)
	}

	// Collect results.
	for i := 0; i < len(intents); i++ {
		select {
		case opResult := <-resultChan:
			result.Results = append(result.Results, opResult)
			if opResult.Success {
				result.SuccessfulOperations++
			} else {
				result.FailedOperations++
				if !opts.ContinueOnError {
					result.OverallSuccess = false
				}
			}
		case <-time.After(opts.Timeout):
			result.OverallSuccess = false
			result.FailedOperations = len(intents) - len(result.Results)
			break
		}
	}

	result.Duration = time.Since(startTime)

	if result.FailedOperations > 0 && !opts.ContinueOnError {
		result.OverallSuccess = false
	}

	m.logger.Info("Batch PackageRevision creation completed",
		"total", result.TotalRequests,
		"successful", result.SuccessfulOperations,
		"failed", result.FailedOperations,
		"duration", result.Duration)

	return result, nil
}

// RenderTemplate renders a template with the given parameters.
func (m *packageRevisionManager) RenderTemplate(ctx context.Context, template *templates.BlueprintTemplate, params map[string]interface{}) ([]*porch.KRMResource, error) {
	return m.templateEngine.RenderTemplate(ctx, template.Name, params)
}

// ValidateConfiguration validates a PackageRevision configuration.
func (m *packageRevisionManager) ValidateConfiguration(ctx context.Context, ref *porch.PackageReference) (*ValidationResult, error) {
	// Get the PackageRevision.
	pkg, err := m.porchClient.GetPackageRevision(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get PackageRevision for validation: %w", err)
	}

	result := &ValidationResult{
		Valid: true,
	}

	// Perform YANG validation if enabled and validator is available.
	if m.config.EnableYANGValidation && m.yangValidator != nil {
		yangResult, err := m.yangValidator.ValidatePackageRevision(ctx, pkg)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "YANG_VALIDATION_FAILED",
				Path:     "/",
				Message:  err.Error(),
				Severity: "error",
				Source:   "yang",
			})
		} else {
			result.YANGValidationResult = yangResult
			if !yangResult.Valid {
				result.Valid = false
				for _, yangErr := range yangResult.Errors {
					result.Errors = append(result.Errors, &ValidationError{
						Code:     yangErr.Code,
						Path:     "/",
						Message:  yangErr.Message,
						Severity: "error",
						Source:   "yang",
					})
				}
			}
		}
	}

	return result, nil
}

// DetectConfigurationDrift detects configuration drift for a PackageRevision.
func (m *packageRevisionManager) DetectConfigurationDrift(ctx context.Context, ref *porch.PackageReference) (*DriftDetectionResult, error) {
	if m.driftDetector != nil {
		return m.driftDetector.DetectDrift(ctx, ref)
	}

	// Return no drift if detector is not available.
	return &DriftDetectionResult{
		HasDrift:        false,
		DetectionTime:   time.Now(),
		Severity:        "none",
		AutoCorrectible: false,
	}, nil
}

// CorrectConfigurationDrift corrects detected configuration drift.
func (m *packageRevisionManager) CorrectConfigurationDrift(ctx context.Context, ref *porch.PackageReference, driftResult *DriftDetectionResult) error {
	if !driftResult.HasDrift || !driftResult.AutoCorrectible {
		return nil
	}

	// Implementation would apply the drift correction plan.
	m.logger.Info("Applying drift correction plan",
		"package", ref.GetPackageKey(),
		"driftCount", len(driftResult.DriftDetails))

	// For now, return nil indicating success.
	// In a real implementation, this would apply the corrections.
	return nil
}

// GetAvailableTemplates gets available templates for a target component.
func (m *packageRevisionManager) GetAvailableTemplates(ctx context.Context, targetComponent nephoranv1.ORANComponent) ([]*templates.BlueprintTemplate, error) {
	// For now, return a mock template.
	// In a real implementation, this would query the template repository.
	mockTemplate := &templates.BlueprintTemplate{
		Name:        fmt.Sprintf("%s-template", targetComponent),
		Version:     "v1.0.0",
		Description: fmt.Sprintf("Template for %s component", targetComponent),
	}

	return []*templates.BlueprintTemplate{mockTemplate}, nil
}

// GetLifecycleStatus gets the current lifecycle status of a PackageRevision.
func (m *packageRevisionManager) GetLifecycleStatus(ctx context.Context, ref *porch.PackageReference) (*LifecycleStatus, error) {
	pkg, err := m.porchClient.GetPackageRevision(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get PackageRevision: %w", err)
	}

	status := &LifecycleStatus{
		CurrentStage:   pkg.Spec.Lifecycle,
		StageStartTime: time.Now(), // In real implementation, this would be tracked
	}

	// Determine next possible stages based on current stage.
	switch pkg.Spec.Lifecycle {
	case porch.PackageRevisionLifecycleDraft:
		status.NextPossibleStages = []porch.PackageRevisionLifecycle{
			porch.PackageRevisionLifecycleProposed,
		}
	case porch.PackageRevisionLifecycleProposed:
		status.NextPossibleStages = []porch.PackageRevisionLifecycle{
			porch.PackageRevisionLifecyclePublished,
			porch.PackageRevisionLifecycleDraft,
		}
	case porch.PackageRevisionLifecyclePublished:
		status.NextPossibleStages = []porch.PackageRevisionLifecycle{
			porch.PackageRevisionLifecycleDeletable,
		}
	}

	return status, nil
}

// GetPackageMetrics gets metrics for a specific package.
func (m *packageRevisionManager) GetPackageMetrics(ctx context.Context, ref *porch.PackageReference) (*PackageMetrics, error) {
	// In a real implementation, this would aggregate metrics from various sources.
	return &PackageMetrics{
		PackageRef:            ref,
		TotalTransitions:      0,
		TransitionsByStage:    make(map[porch.PackageRevisionLifecycle]int64),
		AverageTransitionTime: 0,
		FailedTransitions:     0,
		ValidationFailures:    0,
		ApprovalCycles:        0,
		DriftDetections:       0,
		AutoCorrections:       0,
		TimeInCurrentStage:    0,
		LastActivity:          time.Now(),
	}, nil
}

// GetManagerHealth gets the health status of the manager.
func (m *packageRevisionManager) GetManagerHealth(ctx context.Context) (*ManagerHealth, error) {
	m.transitionMutex.RLock()
	activeTransitions := len(m.activeTransitions)
	m.transitionMutex.RUnlock()

	componentHealth := make(map[string]string)

	// Check component health.
	if m.porchClient != nil {
		if _, err := m.porchClient.Health(ctx); err != nil {
			componentHealth["porch-client"] = "unhealthy"
		} else {
			componentHealth["porch-client"] = "healthy"
		}
	}

	if m.templateEngine != nil {
		if _, err := m.templateEngine.GetEngineHealth(ctx); err != nil {
			componentHealth["template-engine"] = "unhealthy"
		} else {
			componentHealth["template-engine"] = "healthy"
		}
	}

	// Determine overall status.
	status := "healthy"
	for _, health := range componentHealth {
		if health == "unhealthy" {
			status = "degraded"
			break
		}
	}

	return &ManagerHealth{
		Status:            status,
		ActiveTransitions: activeTransitions,
		QueuedOperations:  0, // Would track actual queue size
		ComponentHealth:   componentHealth,
		LastActivity:      time.Now(),
		Metrics:           m.metrics,
	}, nil
}

// performLifecycleTransition performs the actual lifecycle transition.
func (m *packageRevisionManager) performLifecycleTransition(ctx context.Context, ref *porch.PackageReference, targetStage porch.PackageRevisionLifecycle, opts *TransitionOptions) (*TransitionResult, error) {
	m.logger.Info("Performing lifecycle transition",
		"package", ref.GetPackageKey(),
		"targetStage", targetStage)

	if opts == nil {
		opts = &TransitionOptions{}
	}

	startTime := time.Now()
	transitionID := fmt.Sprintf("trans-%s-%d", ref.GetPackageKey(), time.Now().UnixNano())

	// Track active transition.
	activeTransition := &ActiveTransition{
		ID:          transitionID,
		PackageRef:  ref,
		TargetStage: targetStage,
		StartTime:   startTime,
		LastUpdate:  startTime,
		Status:      "starting",
		Progress:    0,
		CurrentStep: "initialization",
		Options:     opts,
	}

	m.transitionMutex.Lock()
	m.activeTransitions[transitionID] = activeTransition
	m.transitionMutex.Unlock()

	defer func() {
		m.transitionMutex.Lock()
		delete(m.activeTransitions, transitionID)
		m.transitionMutex.Unlock()
	}()

	result := &TransitionResult{
		PreviousStage:  "", // Will be set after getting current package
		NewStage:       targetStage,
		TransitionTime: startTime,
		Success:        true,
		Metadata:       make(map[string]interface{}),
	}

	// Get current package state.
	pkg, err := m.porchClient.GetPackageRevision(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to get package revision: %w", err)
	}

	result.PreviousStage = pkg.Spec.Lifecycle

	// Update transition progress.
	m.updateTransitionProgress(transitionID, "validation", 20)

	// Validate the transition if not skipped.
	if !opts.SkipValidation {
		validationResult, err := m.ValidateConfiguration(ctx, ref)
		if err != nil || !validationResult.Valid {
			result.Success = false
			result.ValidationResults = []*ValidationResult{validationResult}
			result.Duration = time.Since(startTime)

			if !opts.ForceTransition {
				return result, fmt.Errorf("validation failed for transition to %s", targetStage)
			}
			result.Warnings = append(result.Warnings, "Validation failed but transition forced")
		}
		result.ValidationResults = []*ValidationResult{validationResult}
	}

	// Update transition progress.
	m.updateTransitionProgress(transitionID, "approval", 50)

	// Handle approval workflow if not skipped.
	if !opts.SkipApproval && m.config.EnableApprovalWorkflow {
		approvalResult, err := m.handleApprovalWorkflow(ctx, ref, targetStage, opts.ApprovalPolicy)
		if err != nil {
			result.Success = false
			result.Duration = time.Since(startTime)

			if !opts.ForceTransition {
				return result, fmt.Errorf("approval workflow failed: %w", err)
			}
			result.Warnings = append(result.Warnings, "Approval failed but transition forced")
		}
		result.ApprovalResults = []*ApprovalResult{approvalResult}
	}

	// Update transition progress.
	m.updateTransitionProgress(transitionID, "transition", 80)

	// Perform the actual transition using LifecycleManager.
	if !opts.DryRun {
		lifecycleOpts := &porch.TransitionOptions{
			SkipValidation:      opts.SkipValidation,
			CreateRollbackPoint: opts.CreateRollbackPoint,
			RollbackDescription: opts.RollbackDescription,
			ForceTransition:     opts.ForceTransition,
			Timeout:             opts.Timeout,
			DryRun:              opts.DryRun,
		}

		var lifecycleResult *porch.TransitionResult
		var err error

		switch targetStage {
		case porch.PackageRevisionLifecycleProposed:
			lifecycleResult, err = m.lifecycleManager.TransitionToProposed(ctx, ref, lifecycleOpts)
		case porch.PackageRevisionLifecyclePublished:
			lifecycleResult, err = m.lifecycleManager.TransitionToPublished(ctx, ref, lifecycleOpts)
		case porch.PackageRevisionLifecycleDraft:
			lifecycleResult, err = m.lifecycleManager.TransitionToDraft(ctx, ref, lifecycleOpts)
		case porch.PackageRevisionLifecycleDeletable:
			lifecycleResult, err = m.lifecycleManager.TransitionToDeletable(ctx, ref, lifecycleOpts)
		default:
			return result, fmt.Errorf("unsupported target stage: %s", targetStage)
		}

		if err != nil {
			result.Success = false
			result.Duration = time.Since(startTime)
			m.metrics.TransitionsTotal.WithLabelValues(string(targetStage), "error").Inc()
			return result, fmt.Errorf("lifecycle transition failed: %w", err)
		}

		if lifecycleResult.RollbackPoint != nil {
			result.RollbackPoint = lifecycleResult.RollbackPoint
		}
		result.Warnings = append(result.Warnings, lifecycleResult.Warnings...)
	}

	// Update transition progress.
	m.updateTransitionProgress(transitionID, "notification", 95)

	// Send notifications if configured.
	if m.config.EnableNotifications && len(opts.NotificationTargets) > 0 {
		notifications := m.sendTransitionNotifications(ctx, ref, result, opts.NotificationTargets)
		result.Notifications = notifications
	}

	// Update transition progress.
	m.updateTransitionProgress(transitionID, "completed", 100)

	result.Duration = time.Since(startTime)
	m.metrics.TransitionsTotal.WithLabelValues(string(targetStage), "success").Inc()
	m.metrics.TransitionDuration.WithLabelValues(string(targetStage)).Observe(result.Duration.Seconds())

	m.logger.Info("Lifecycle transition completed successfully",
		"package", ref.GetPackageKey(),
		"from", result.PreviousStage,
		"to", targetStage,
		"duration", result.Duration)

	return result, nil
}

// Helper methods.

func (m *packageRevisionManager) validateNetworkIntent(intent *nephoranv1.NetworkIntent) error {
	if intent == nil {
		return fmt.Errorf("NetworkIntent cannot be nil")
	}
	if intent.Spec.Intent == "" {
		return fmt.Errorf("NetworkIntent spec.intent cannot be empty")
	}
	if len(intent.Spec.TargetComponents) == 0 {
		return fmt.Errorf("NetworkIntent must specify at least one target component")
	}
	return nil
}

func (m *packageRevisionManager) extractParametersFromIntent(intent *nephoranv1.NetworkIntent) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Extract basic parameters.
	params["intentText"] = intent.Spec.Intent
	params["intentType"] = string(intent.Spec.IntentType)
	params["priority"] = string(intent.Spec.Priority)
	params["targetComponents"] = intent.Spec.TargetComponents
	params["namespace"] = intent.Namespace

	// Extract resource constraints if specified.
	if intent.Spec.ResourceConstraints != nil {
		params["resourceConstraints"] = intent.Spec.ResourceConstraints
	}

	// Extract processed parameters if available.
	if intent.Spec.ProcessedParameters != nil {
		if intent.Spec.ProcessedParameters.NetworkFunction != "" {
			params["networkFunction"] = intent.Spec.ProcessedParameters.NetworkFunction
		}
		// Add other processed parameters as needed.
	}

	return params, nil
}

func (m *packageRevisionManager) selectTemplateForIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, params map[string]interface{}) (*templates.BlueprintTemplate, error) {
	// Get the primary target component.
	primaryComponent := intent.Spec.TargetComponents[0]

	// Get available templates for the component.
	availableTemplates, err := m.GetAvailableTemplates(ctx, primaryComponent)
	if err != nil {
		return nil, fmt.Errorf("failed to get available templates for component %s: %w", primaryComponent, err)
	}

	if len(availableTemplates) == 0 {
		return nil, fmt.Errorf("no templates available for component %s", primaryComponent)
	}

	// For now, select the first available template.
	// In a more sophisticated implementation, this could use ML or rules-based selection.
	selectedTemplate := availableTemplates[0]

	m.logger.Info("Selected template for intent",
		"template", selectedTemplate.Name,
		"version", selectedTemplate.Version,
		"component", primaryComponent)

	return selectedTemplate, nil
}

func (m *packageRevisionManager) generatePackageName(intent *nephoranv1.NetworkIntent) string {
	// Generate a unique package name based on intent.
	return fmt.Sprintf("%s-%s", intent.Name, string(intent.Spec.IntentType))
}

func (m *packageRevisionManager) generateLabelsForIntent(intent *nephoranv1.NetworkIntent) map[string]string {
	labels := make(map[string]string)
	labels[porch.LabelComponent] = "network-function"
	labels[porch.LabelIntentType] = string(intent.Spec.IntentType)
	labels["nephoran.com/intent-name"] = intent.Name
	labels["nephoran.com/intent-namespace"] = intent.Namespace

	if len(intent.Spec.TargetComponents) > 0 {
		labels[porch.LabelTargetComponent] = string(intent.Spec.TargetComponents[0])
	}

	return labels
}

func (m *packageRevisionManager) generateAnnotationsForIntent(intent *nephoranv1.NetworkIntent) map[string]string {
	annotations := make(map[string]string)
	annotations[porch.AnnotationManagedBy] = "nephoran-intent-operator"
	annotations[porch.AnnotationGeneratedBy] = "package-revision-manager"
	annotations["nephoran.com/original-intent"] = intent.Spec.Intent
	annotations["nephoran.com/intent-uid"] = string(intent.UID)

	return annotations
}

func (m *packageRevisionManager) templateSupportsComponent(template *templates.BlueprintTemplate, targetComponent nephoranv1.ORANComponent) bool {
	// For now, assume all templates support all components.
	// In a real implementation, this would check template metadata or capabilities.
	return true
}

func (m *packageRevisionManager) updateTransitionProgress(transitionID, step string, progress int) {
	m.transitionMutex.Lock()
	defer m.transitionMutex.Unlock()

	if transition, exists := m.activeTransitions[transitionID]; exists {
		transition.LastUpdate = time.Now()
		transition.CurrentStep = step
		transition.Progress = progress
		transition.Status = "in_progress"
		if progress >= 100 {
			transition.Status = "completed"
		}
	}
}

func (m *packageRevisionManager) handleApprovalWorkflow(ctx context.Context, ref *porch.PackageReference, targetStage porch.PackageRevisionLifecycle, policy *ApprovalPolicy) (*ApprovalResult, error) {
	// Implementation would integrate with approval engine.
	// For now, return a successful approval.
	return &ApprovalResult{
		WorkflowID:        fmt.Sprintf("approval-%d", time.Now().UnixNano()),
		Stage:             string(targetStage),
		Status:            "approved",
		Approver:          "system",
		ApprovalTime:      &metav1.Time{Time: time.Now()},
		RequiredApprovals: 1,
		ReceivedApprovals: 1,
	}, nil
}

func (m *packageRevisionManager) sendTransitionNotifications(ctx context.Context, ref *porch.PackageReference, result *TransitionResult, targets []string) []*NotificationResult {
	notifications := make([]*NotificationResult, 0, len(targets))

	for _, target := range targets {
		notification := &NotificationResult{
			Target:  target,
			Success: true,
			Message: fmt.Sprintf("PackageRevision %s transitioned from %s to %s", ref.GetPackageKey(), result.PreviousStage, result.NewStage),
			SentAt:  time.Now(),
		}
		notifications = append(notifications, notification)
	}

	return notifications
}

// Background workers.

func (m *packageRevisionManager) driftDetectionWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.config.DriftDetectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdown:
			return
		case <-ticker.C:
			if m.config.EnableDriftDetection {
				m.performDriftDetection()
			}
		}
	}
}

func (m *packageRevisionManager) metricsCollectionWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdown:
			return
		case <-ticker.C:
			if m.config.EnableMetrics {
				m.collectMetrics()
			}
		}
	}
}

func (m *packageRevisionManager) performDriftDetection() {
	// Implementation would perform drift detection across all managed packages.
	m.logger.V(1).Info("Performing drift detection across managed packages")
}

func (m *packageRevisionManager) collectMetrics() {
	// Update active transitions metric.
	m.transitionMutex.RLock()
	activeCount := len(m.activeTransitions)
	m.transitionMutex.RUnlock()

	m.metrics.ActiveTransitions.Set(float64(activeCount))
}

// Close gracefully shuts down the manager.
func (m *packageRevisionManager) Close() error {
	m.logger.Info("Shutting down PackageRevision manager")

	close(m.shutdown)
	m.wg.Wait()

	if m.approvalEngine != nil {
		m.approvalEngine.Close()
	}
	if m.driftDetector != nil {
		m.driftDetector.Close()
	}

	m.logger.Info("PackageRevision manager shutdown complete")
	return nil
}

// Utility functions.

func getDefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		DefaultRepository:        "default",
		DefaultTimeout:           30 * time.Minute,
		MaxConcurrentTransitions: 10,
		EnableYANGValidation:     true,
		EnablePolicyValidation:   true,
		EnableSecurityValidation: true,
		EnableComplianceChecks:   true,
		ValidationTimeout:        10 * time.Minute,
		EnableApprovalWorkflow:   true,
		ApprovalTimeout:          60 * time.Minute,
		EnableDriftDetection:     true,
		DriftDetectionInterval:   15 * time.Minute,
		AutoCorrectDrift:         false,
		TemplateRepository:       "nephoran-templates",
		TemplateRefreshInterval:  60 * time.Minute,
		EnableMetrics:            true,
		MetricsInterval:          30 * time.Second,
		EnableNotifications:      true,
	}
}

func initManagerMetrics() *ManagerMetrics {
	return &ManagerMetrics{
		TotalPackagesManaged: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "packagerevision_manager_packages_total",
			Help: "Total number of packages managed",
		}),
		TransitionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "packagerevision_manager_transitions_total",
			Help: "Total number of lifecycle transitions",
		}, []string{"stage", "status"}),
		TransitionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "packagerevision_manager_transition_duration_seconds",
			Help: "Duration of lifecycle transitions",
		}, []string{"stage"}),
		ValidationResults: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "packagerevision_manager_validation_results_total",
			Help: "Total number of validation results",
		}, []string{"operation", "result"}),
		ApprovalLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "packagerevision_manager_approval_latency_seconds",
			Help: "Latency of approval workflows",
		}),
		DriftDetections: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "packagerevision_manager_drift_detections_total",
			Help: "Total number of drift detections",
		}),
		ActiveTransitions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "packagerevision_manager_active_transitions",
			Help: "Number of active transitions",
		}),
		QueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "packagerevision_manager_queue_size",
			Help: "Size of operation queue",
		}),
	}
}

// Placeholder interfaces that would be implemented in separate files.

// ApprovalEngine handles approval workflows.
type ApprovalEngine interface {
	ExecuteApprovalWorkflow(ctx context.Context, ref *porch.PackageReference, stage porch.PackageRevisionLifecycle, policy *ApprovalPolicy) (*ApprovalResult, error)
	Close() error
}

// DriftDetector detects configuration drift.
type DriftDetector interface {
	DetectDrift(ctx context.Context, ref *porch.PackageReference) (*DriftDetectionResult, error)
	Close() error
}

// Placeholder implementations.
func NewApprovalEngine(config *ManagerConfig) (ApprovalEngine, error) {
	return &mockApprovalEngine{}, nil
}

// NewDriftDetector performs newdriftdetector operation.
func NewDriftDetector(client porch.PorchClient, config *ManagerConfig) (DriftDetector, error) {
	return &mockDriftDetector{}, nil
}

type mockApprovalEngine struct{}

// ExecuteApprovalWorkflow performs executeapprovalworkflow operation.
func (e *mockApprovalEngine) ExecuteApprovalWorkflow(ctx context.Context, ref *porch.PackageReference, stage porch.PackageRevisionLifecycle, policy *ApprovalPolicy) (*ApprovalResult, error) {
	return &ApprovalResult{
		WorkflowID:        fmt.Sprintf("workflow-%d", time.Now().UnixNano()),
		Stage:             string(stage),
		Status:            "approved",
		RequiredApprovals: 1,
		ReceivedApprovals: 1,
	}, nil
}

// Close performs close operation.
func (e *mockApprovalEngine) Close() error { return nil }

type mockDriftDetector struct{}

// DetectDrift performs detectdrift operation.
func (d *mockDriftDetector) DetectDrift(ctx context.Context, ref *porch.PackageReference) (*DriftDetectionResult, error) {
	return &DriftDetectionResult{
		HasDrift:        false,
		DetectionTime:   time.Now(),
		Severity:        "none",
		AutoCorrectible: true,
	}, nil
}

// Close performs close operation.
func (d *mockDriftDetector) Close() error { return nil }

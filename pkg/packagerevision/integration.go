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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/templates"
	"github.com/thc1006/nephoran-intent-operator/pkg/validation/yang"
)

// NetworkIntentPackageReconciler integrates NetworkIntent processing with PackageRevision lifecycle management
// Extends the existing NetworkIntent controller to leverage Porch for package orchestration
type NetworkIntentPackageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger

	// Core components
	PackageManager   PackageRevisionManager
	TemplateEngine   templates.TemplateEngine
	YANGValidator    yang.YANGValidator
	PorchClient      porch.PorchClient
	LifecycleManager porch.LifecycleManager

	// Configuration
	Config *IntegrationConfig
}

// IntegrationConfig contains configuration for NetworkIntent-PackageRevision integration
type IntegrationConfig struct {
	// PackageRevision lifecycle settings
	AutoPromoteToProposed    bool          `yaml:"autoPromoteToProposed"`
	AutoPromoteToPublished   bool          `yaml:"autoPromoteToPublished"`
	AutoCreateRollbackPoints bool          `yaml:"autoCreateRollbackPoints"`
	DefaultTransitionTimeout time.Duration `yaml:"defaultTransitionTimeout"`

	// Validation settings
	RequireYANGValidation bool     `yaml:"requireYangValidation"`
	RequiredApprovals     int      `yaml:"requiredApprovals"`
	ValidationModels      []string `yaml:"validationModels"`

	// Template settings
	PreferredTemplateVendors  []string `yaml:"preferredTemplateVendors"`
	EnableTemplateInheritance bool     `yaml:"enableTemplateInheritance"`

	// Error handling
	FailureRetryCount         int           `yaml:"failureRetryCount"`
	FailureRetryInterval      time.Duration `yaml:"failureRetryInterval"`
	ContinueOnValidationError bool          `yaml:"continueOnValidationError"`

	// Status reporting
	UpdateStatusFrequency time.Duration `yaml:"updateStatusFrequency"`
	EnableDetailedStatus  bool          `yaml:"enableDetailedStatus"`
}

// NetworkIntentPackageStatus represents the extended status for NetworkIntent with PackageRevision integration
type NetworkIntentPackageStatus struct {
	// PackageRevision information
	PackageReference   *porch.PackageReference        `json:"packageReference,omitempty"`
	PackageLifecycle   porch.PackageRevisionLifecycle `json:"packageLifecycle,omitempty"`
	PackageCreatedAt   *metav1.Time                   `json:"packageCreatedAt,omitempty"`
	PackageLastUpdated *metav1.Time                   `json:"packageLastUpdated,omitempty"`

	// Template information
	UsedTemplate       string `json:"usedTemplate,omitempty"`
	TemplateVersion    string `json:"templateVersion,omitempty"`
	GeneratedResources int    `json:"generatedResources,omitempty"`

	// Validation results
	YANGValidationResult *ValidationSummary     `json:"yangValidationResult,omitempty"`
	ApprovalStatus       *ApprovalStatusSummary `json:"approvalStatus,omitempty"`

	// Lifecycle tracking
	TransitionHistory  []*TransitionHistoryEntry `json:"transitionHistory,omitempty"`
	PendingTransitions []*PendingTransition      `json:"pendingTransitions,omitempty"`

	// Error tracking
	LastError     string       `json:"lastError,omitempty"`
	ErrorCount    int          `json:"errorCount,omitempty"`
	LastErrorTime *metav1.Time `json:"lastErrorTime,omitempty"`

	// Performance metrics
	ProcessingDuration *metav1.Duration `json:"processingDuration,omitempty"`
	ValidationDuration *metav1.Duration `json:"validationDuration,omitempty"`
	DeploymentDuration *metav1.Duration `json:"deploymentDuration,omitempty"`
}

// ValidationSummary provides a summary of YANG validation results
type ValidationSummary struct {
	Valid        bool        `json:"valid"`
	ModelCount   int         `json:"modelCount"`
	ErrorCount   int         `json:"errorCount"`
	WarningCount int         `json:"warningCount"`
	ValidatedAt  metav1.Time `json:"validatedAt"`
}

// ApprovalStatusSummary provides a summary of approval workflow status
type ApprovalStatusSummary struct {
	Required         int          `json:"required"`
	Received         int          `json:"received"`
	Status           string       `json:"status"` // pending, approved, rejected
	LastApprovalTime *metav1.Time `json:"lastApprovalTime,omitempty"`
}

// TransitionHistoryEntry tracks lifecycle transitions
type TransitionHistoryEntry struct {
	FromStage      porch.PackageRevisionLifecycle `json:"fromStage"`
	ToStage        porch.PackageRevisionLifecycle `json:"toStage"`
	TransitionTime metav1.Time                    `json:"transitionTime"`
	Duration       metav1.Duration                `json:"duration"`
	Success        bool                           `json:"success"`
	User           string                         `json:"user,omitempty"`
	Reason         string                         `json:"reason,omitempty"`
}

// PendingTransition tracks pending lifecycle transitions
type PendingTransition struct {
	ToStage       porch.PackageRevisionLifecycle `json:"toStage"`
	ScheduledAt   metav1.Time                    `json:"scheduledAt"`
	Prerequisites []string                       `json:"prerequisites,omitempty"`
	Timeout       *metav1.Duration               `json:"timeout,omitempty"`
}

// SetupWithManager sets up the controller with the manager
func (r *NetworkIntentPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}

// Reconcile handles NetworkIntent reconciliation with PackageRevision lifecycle management
func (r *NetworkIntentPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("networkintent", req.NamespacedName)
	startTime := time.Now()

	// Get the NetworkIntent
	var intent nephoranv1.NetworkIntent
	if err := r.Get(ctx, req.NamespacedName, &intent); err != nil {
		if errors.IsNotFound(err) {
			// Intent was deleted, handle cleanup
			return r.handleIntentDeletion(ctx, req.NamespacedName)
		}
		logger.Error(err, "Failed to get NetworkIntent")
		return ctrl.Result{}, err
	}

	// Initialize extended status if not present
	if intent.Status.Extensions == nil {
		intent.Status.Extensions = &runtime.RawExtension{}
	}

	// Check for deletion
	if !intent.DeletionTimestamp.IsZero() {
		return r.handleIntentDeletion(ctx, req.NamespacedName)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&intent, NetworkIntentFinalizer) {
		controllerutil.AddFinalizer(&intent, NetworkIntentFinalizer)
		if err := r.Update(ctx, &intent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get or initialize extended status
	extendedStatus, err := r.getExtendedStatus(&intent)
	if err != nil {
		logger.Error(err, "Failed to get extended status")
		extendedStatus = &NetworkIntentPackageStatus{}
	}

	logger.Info("Reconciling NetworkIntent with PackageRevision lifecycle",
		"intent", intent.Name,
		"currentStatus", intent.Status.Phase,
		"packageLifecycle", extendedStatus.PackageLifecycle)

	// Process the NetworkIntent through the PackageRevision lifecycle
	result, err := r.processIntentLifecycle(ctx, &intent, extendedStatus)
	if err != nil {
		extendedStatus.LastError = err.Error()
		extendedStatus.ErrorCount++
		extendedStatus.LastErrorTime = &metav1.Time{Time: time.Now()}
		logger.Error(err, "Failed to process intent lifecycle")
	}

	// Update processing duration
	processingDuration := time.Since(startTime)
	extendedStatus.ProcessingDuration = &metav1.Duration{Duration: processingDuration}

	// Update extended status
	if err := r.updateExtendedStatus(ctx, &intent, extendedStatus); err != nil {
		logger.Error(err, "Failed to update extended status")
	}

	// Update standard status
	if err := r.Status().Update(ctx, &intent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	logger.Info("NetworkIntent reconciliation completed",
		"intent", intent.Name,
		"result", result,
		"duration", processingDuration)

	return result, err
}

// processIntentLifecycle processes the NetworkIntent through the PackageRevision lifecycle
func (r *NetworkIntentPackageReconciler) processIntentLifecycle(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	switch intent.Status.Phase {
	case nephoranv1.NetworkIntentPhasePending:
		return r.handlePendingIntent(ctx, intent, status)
	case nephoranv1.NetworkIntentPhaseProcessing:
		return r.handleProcessingIntent(ctx, intent, status)
	case nephoranv1.NetworkIntentPhaseDeploying:
		return r.handleDeployingIntent(ctx, intent, status)
	case nephoranv1.NetworkIntentPhaseActive:
		return r.handleActiveIntent(ctx, intent, status)
	case nephoranv1.NetworkIntentPhaseFailed:
		return r.handleFailedIntent(ctx, intent, status)
	default:
		// Initialize new intent
		return r.initializeIntent(ctx, intent, status)
	}
}

// initializeIntent initializes a new NetworkIntent with PackageRevision creation
func (r *NetworkIntentPackageReconciler) initializeIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.Info("Initializing NetworkIntent", "intent", intent.Name)

	// Update status to pending
	intent.Status.Phase = nephoranv1.NetworkIntentPhasePending
	intent.Status.Message = "Initializing intent processing"

	return ctrl.Result{RequeueAfter: time.Second}, nil
}

// handlePendingIntent handles NetworkIntent in pending phase
func (r *NetworkIntentPackageReconciler) handlePendingIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.Info("Handling pending NetworkIntent", "intent", intent.Name)

	// Check if PackageRevision already exists
	if status.PackageReference != nil {
		// PackageRevision exists, move to processing
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseProcessing
		intent.Status.Message = "PackageRevision exists, processing intent"
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Create PackageRevision from NetworkIntent
	packageRevision, err := r.PackageManager.CreateFromIntent(ctx, intent)
	if err != nil {
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
		intent.Status.Message = fmt.Sprintf("Failed to create PackageRevision: %v", err)
		return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, err
	}

	// Update status with package information
	status.PackageReference = &porch.PackageReference{
		Repository:  packageRevision.Spec.Repository,
		PackageName: packageRevision.Spec.PackageName,
		Revision:    packageRevision.Spec.Revision,
	}
	status.PackageLifecycle = packageRevision.Spec.Lifecycle
	status.PackageCreatedAt = &metav1.Time{Time: time.Now()}
	status.GeneratedResources = len(packageRevision.Spec.Resources)

	// Extract template information if available
	if templateName, exists := packageRevision.ObjectMeta.Annotations["nephoran.com/template-name"]; exists {
		status.UsedTemplate = templateName
	}
	if templateVersion, exists := packageRevision.ObjectMeta.Annotations["nephoran.com/template-version"]; exists {
		status.TemplateVersion = templateVersion
	}

	// Move to processing phase
	intent.Status.Phase = nephoranv1.NetworkIntentPhaseProcessing
	intent.Status.Message = fmt.Sprintf("PackageRevision created: %s", status.PackageReference.GetPackageKey())

	r.Logger.Info("PackageRevision created successfully",
		"intent", intent.Name,
		"package", status.PackageReference.GetPackageKey(),
		"lifecycle", status.PackageLifecycle)

	return ctrl.Result{RequeueAfter: time.Second}, nil
}

// handleProcessingIntent handles NetworkIntent in processing phase
func (r *NetworkIntentPackageReconciler) handleProcessingIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.Info("Handling processing NetworkIntent", "intent", intent.Name)

	if status.PackageReference == nil {
		return r.handlePendingIntent(ctx, intent, status)
	}

	// Perform YANG validation if required
	if r.Config.RequireYANGValidation && status.YANGValidationResult == nil {
		validationResult, err := r.performYANGValidation(ctx, intent, status)
		if err != nil {
			if !r.Config.ContinueOnValidationError {
				intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
				intent.Status.Message = fmt.Sprintf("YANG validation failed: %v", err)
				return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, err
			}
			r.Logger.Error(err, "YANG validation failed but continuing", "intent", intent.Name)
		}
		status.YANGValidationResult = validationResult
	}

	// Auto-promote to Proposed if configured
	if r.Config.AutoPromoteToProposed && status.PackageLifecycle == porch.PackageRevisionLifecycleDraft {
		transitionResult, err := r.promotePackage(ctx, intent, status, porch.PackageRevisionLifecycleProposed)
		if err != nil {
			intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
			intent.Status.Message = fmt.Sprintf("Failed to promote to Proposed: %v", err)
			return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, err
		}

		// Update status with transition result
		r.updateStatusFromTransition(status, transitionResult)

		r.Logger.Info("Package promoted to Proposed",
			"intent", intent.Name,
			"package", status.PackageReference.GetPackageKey())
	}

	// Check if approval is required and handle approval workflow
	if r.Config.RequiredApprovals > 0 && status.PackageLifecycle == porch.PackageRevisionLifecycleProposed {
		if status.ApprovalStatus == nil || status.ApprovalStatus.Status == "pending" {
			// Approval workflow is pending
			intent.Status.Phase = nephoranv1.NetworkIntentPhaseProcessing
			intent.Status.Message = fmt.Sprintf("Waiting for approvals (%d/%d required)",
				r.getReceivedApprovals(status), r.Config.RequiredApprovals)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		if status.ApprovalStatus.Status == "rejected" {
			intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
			intent.Status.Message = "Intent rejected during approval workflow"
			return ctrl.Result{}, nil
		}
	}

	// Auto-promote to Published if configured and approved
	if r.Config.AutoPromoteToPublished &&
		status.PackageLifecycle == porch.PackageRevisionLifecycleProposed &&
		(r.Config.RequiredApprovals == 0 || status.ApprovalStatus.Status == "approved") {

		transitionResult, err := r.promotePackage(ctx, intent, status, porch.PackageRevisionLifecyclePublished)
		if err != nil {
			intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
			intent.Status.Message = fmt.Sprintf("Failed to promote to Published: %v", err)
			return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, err
		}

		// Update status with transition result
		r.updateStatusFromTransition(status, transitionResult)

		// Move to deploying phase
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseDeploying
		intent.Status.Message = "Package published, starting deployment"

		r.Logger.Info("Package promoted to Published",
			"intent", intent.Name,
			"package", status.PackageReference.GetPackageKey())

		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// If package is published but we're still in processing, move to deploying
	if status.PackageLifecycle == porch.PackageRevisionLifecyclePublished {
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseDeploying
		intent.Status.Message = "Package published, starting deployment"
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Continue processing
	return ctrl.Result{RequeueAfter: r.Config.UpdateStatusFrequency}, nil
}

// handleDeployingIntent handles NetworkIntent in deploying phase
func (r *NetworkIntentPackageReconciler) handleDeployingIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.Info("Handling deploying NetworkIntent", "intent", intent.Name)

	if status.PackageReference == nil {
		return r.handlePendingIntent(ctx, intent, status)
	}

	// Check deployment status of the PackageRevision
	deploymentStatus, err := r.checkDeploymentStatus(ctx, status.PackageReference)
	if err != nil {
		r.Logger.Error(err, "Failed to check deployment status", "intent", intent.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	switch deploymentStatus {
	case "deployed":
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseActive
		intent.Status.Message = "Intent successfully deployed"
		r.Logger.Info("NetworkIntent deployment completed",
			"intent", intent.Name,
			"package", status.PackageReference.GetPackageKey())
		return ctrl.Result{RequeueAfter: r.Config.UpdateStatusFrequency}, nil

	case "failed":
		intent.Status.Phase = nephoranv1.NetworkIntentPhaseFailed
		intent.Status.Message = "Deployment failed"
		return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, nil

	case "pending", "deploying":
		intent.Status.Message = "Deployment in progress"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	default:
		intent.Status.Message = fmt.Sprintf("Unknown deployment status: %s", deploymentStatus)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
}

// handleActiveIntent handles NetworkIntent in active phase
func (r *NetworkIntentPackageReconciler) handleActiveIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.V(1).Info("Handling active NetworkIntent", "intent", intent.Name)

	// Perform drift detection if enabled
	if status.PackageReference != nil {
		driftResult, err := r.PackageManager.DetectConfigurationDrift(ctx, status.PackageReference)
		if err != nil {
			r.Logger.Error(err, "Failed to detect configuration drift", "intent", intent.Name)
		} else if driftResult.HasDrift {
			intent.Status.Message = fmt.Sprintf("Configuration drift detected (severity: %s)", driftResult.Severity)
			r.Logger.Info("Configuration drift detected",
				"intent", intent.Name,
				"severity", driftResult.Severity,
				"drifts", len(driftResult.DriftDetails))

			// Auto-correct drift if configured and safe
			if driftResult.AutoCorrectible {
				if err := r.PackageManager.CorrectConfigurationDrift(ctx, status.PackageReference, driftResult); err != nil {
					r.Logger.Error(err, "Failed to correct configuration drift", "intent", intent.Name)
				} else {
					intent.Status.Message = "Configuration drift auto-corrected"
					r.Logger.Info("Configuration drift auto-corrected", "intent", intent.Name)
				}
			}
		}
	}

	// Regular status update
	return ctrl.Result{RequeueAfter: r.Config.UpdateStatusFrequency}, nil
}

// handleFailedIntent handles NetworkIntent in failed phase
func (r *NetworkIntentPackageReconciler) handleFailedIntent(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (ctrl.Result, error) {
	r.Logger.Info("Handling failed NetworkIntent", "intent", intent.Name)

	// Check if retry is appropriate
	if status.ErrorCount < r.Config.FailureRetryCount {
		r.Logger.Info("Retrying failed NetworkIntent",
			"intent", intent.Name,
			"attempt", status.ErrorCount+1,
			"maxRetries", r.Config.FailureRetryCount)

		// Reset to pending for retry
		intent.Status.Phase = nephoranv1.NetworkIntentPhasePending
		intent.Status.Message = fmt.Sprintf("Retrying intent processing (attempt %d/%d)",
			status.ErrorCount+1, r.Config.FailureRetryCount)

		return ctrl.Result{RequeueAfter: r.Config.FailureRetryInterval}, nil
	}

	// Max retries exceeded, keep in failed state
	intent.Status.Message = fmt.Sprintf("Intent failed after %d attempts. Last error: %s",
		r.Config.FailureRetryCount, status.LastError)

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleIntentDeletion handles cleanup when NetworkIntent is deleted
func (r *NetworkIntentPackageReconciler) handleIntentDeletion(ctx context.Context, namespacedName types.NamespacedName) (ctrl.Result, error) {
	r.Logger.Info("Handling NetworkIntent deletion", "intent", namespacedName)

	// Get the NetworkIntent to access extended status
	var intent nephoranv1.NetworkIntent
	if err := r.Get(ctx, namespacedName, &intent); err != nil {
		if errors.IsNotFound(err) {
			// Already deleted, nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get extended status to find associated PackageRevision
	status, err := r.getExtendedStatus(&intent)
	if err != nil {
		r.Logger.Error(err, "Failed to get extended status for deletion")
		// Continue with deletion even if we can't get status
	}

	// Delete associated PackageRevision if exists
	if status != nil && status.PackageReference != nil {
		r.Logger.Info("Deleting associated PackageRevision",
			"intent", namespacedName,
			"package", status.PackageReference.GetPackageKey())

		if err := r.PackageManager.DeletePackageRevision(ctx, status.PackageReference); err != nil {
			r.Logger.Error(err, "Failed to delete PackageRevision",
				"package", status.PackageReference.GetPackageKey())
			// Continue with cleanup even if PackageRevision deletion fails
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(&intent, NetworkIntentFinalizer)
	if err := r.Update(ctx, &intent); err != nil {
		return ctrl.Result{}, err
	}

	r.Logger.Info("NetworkIntent deletion completed", "intent", namespacedName)
	return ctrl.Result{}, nil
}

// Helper methods

func (r *NetworkIntentPackageReconciler) getExtendedStatus(intent *nephoranv1.NetworkIntent) (*NetworkIntentPackageStatus, error) {
	if intent.Status.Extensions == nil || intent.Status.Extensions.Raw == nil {
		return &NetworkIntentPackageStatus{}, nil
	}

	var status NetworkIntentPackageStatus
	if err := json.Unmarshal(intent.Status.Extensions.Raw, &status); err != nil {
		return nil, fmt.Errorf("failed to decode extended status: %w", err)
	}

	return &status, nil
}

func (r *NetworkIntentPackageReconciler) updateExtendedStatus(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) error {
	// Convert status to JSON
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to encode extended status: %w", err)
	}

	intent.Status.Extensions = &runtime.RawExtension{
		Raw: statusJSON,
	}

	return nil
}

func (r *NetworkIntentPackageReconciler) performYANGValidation(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus) (*ValidationSummary, error) {
	validationStart := time.Now()

	// Get PackageRevision for validation
	pkg, err := r.PorchClient.GetPackageRevision(ctx, status.PackageReference.PackageName, status.PackageReference.Revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get PackageRevision for validation: %w", err)
	}

	// Validate against YANG models
	validationResult, err := r.YANGValidator.ValidatePackageRevision(ctx, pkg)
	if err != nil {
		return nil, fmt.Errorf("YANG validation failed: %w", err)
	}

	// Update validation duration
	validationDuration := time.Since(validationStart)
	status.ValidationDuration = &metav1.Duration{Duration: validationDuration}

	// Create validation summary
	summary := &ValidationSummary{
		Valid:        validationResult.Valid,
		ErrorCount:   len(validationResult.Errors),
		WarningCount: len(validationResult.Warnings),
		ValidatedAt:  metav1.Time{Time: time.Now()},
	}

	if len(validationResult.ModelNames) > 0 {
		summary.ModelCount = len(validationResult.ModelNames)
	} else if validationResult.ModelName != "" {
		summary.ModelCount = 1
	}

	r.Logger.Info("YANG validation completed",
		"intent", intent.Name,
		"valid", summary.Valid,
		"errors", summary.ErrorCount,
		"warnings", summary.WarningCount,
		"duration", validationDuration)

	return summary, nil
}

func (r *NetworkIntentPackageReconciler) promotePackage(ctx context.Context, intent *nephoranv1.NetworkIntent, status *NetworkIntentPackageStatus, targetStage porch.PackageRevisionLifecycle) (*TransitionResult, error) {
	transitionOptions := &TransitionOptions{
		CreateRollbackPoint: r.Config.AutoCreateRollbackPoints,
		RollbackDescription: fmt.Sprintf("Auto-rollback point for %s", intent.Name),
		Timeout:             r.Config.DefaultTransitionTimeout,
	}

	var result *TransitionResult
	var err error

	switch targetStage {
	case porch.PackageRevisionLifecycleProposed:
		result, err = r.PackageManager.TransitionToProposed(ctx, status.PackageReference, transitionOptions)
	case porch.PackageRevisionLifecyclePublished:
		result, err = r.PackageManager.TransitionToPublished(ctx, status.PackageReference, transitionOptions)
	default:
		return nil, fmt.Errorf("unsupported target stage: %s", targetStage)
	}

	if err != nil {
		return nil, err
	}

	if result.Success {
		status.PackageLifecycle = targetStage
		status.PackageLastUpdated = &metav1.Time{Time: time.Now()}
	}

	return result, nil
}

func (r *NetworkIntentPackageReconciler) updateStatusFromTransition(status *NetworkIntentPackageStatus, result *TransitionResult) {
	// Add transition to history
	historyEntry := &TransitionHistoryEntry{
		FromStage:      result.PreviousStage,
		ToStage:        result.NewStage,
		TransitionTime: metav1.Time{Time: result.TransitionTime},
		Duration:       metav1.Duration{Duration: result.Duration},
		Success:        result.Success,
	}

	if status.TransitionHistory == nil {
		status.TransitionHistory = []*TransitionHistoryEntry{}
	}
	status.TransitionHistory = append(status.TransitionHistory, historyEntry)

	// Limit history size
	if len(status.TransitionHistory) > 10 {
		status.TransitionHistory = status.TransitionHistory[len(status.TransitionHistory)-10:]
	}
}

func (r *NetworkIntentPackageReconciler) getReceivedApprovals(status *NetworkIntentPackageStatus) int {
	if status.ApprovalStatus == nil {
		return 0
	}
	return status.ApprovalStatus.Received
}

func (r *NetworkIntentPackageReconciler) checkDeploymentStatus(ctx context.Context, ref *porch.PackageReference) (string, error) {
	// Check the deployment status of the PackageRevision
	// This would integrate with the actual deployment monitoring
	pkg, err := r.PorchClient.GetPackageRevision(ctx, ref.PackageName, ref.Revision)
	if err != nil {
		return "", err
	}

	if pkg.Status.DeploymentStatus != nil {
		return pkg.Status.DeploymentStatus.Phase, nil
	}

	// If no deployment status, assume deployed for published packages
	if pkg.Spec.Lifecycle == porch.PackageRevisionLifecyclePublished {
		return "deployed", nil
	}

	return "pending", nil
}

// Constants and helper types

const (
	NetworkIntentFinalizer = "networkintent.nephoran.com/package-revision"
)

// GetDefaultIntegrationConfig returns default integration configuration
func GetDefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		AutoPromoteToProposed:     true,
		AutoPromoteToPublished:    true,
		AutoCreateRollbackPoints:  true,
		DefaultTransitionTimeout:  30 * time.Minute,
		RequireYANGValidation:     true,
		RequiredApprovals:         0,
		ValidationModels:          []string{"oran-interfaces", "3gpp-5gc"},
		PreferredTemplateVendors:  []string{"open-source", "nephoran"},
		EnableTemplateInheritance: true,
		FailureRetryCount:         3,
		FailureRetryInterval:      5 * time.Minute,
		ContinueOnValidationError: false,
		UpdateStatusFrequency:     60 * time.Second,
		EnableDetailedStatus:      true,
	}
}

// NewNetworkIntentPackageReconciler creates a new NetworkIntent package reconciler
func NewNetworkIntentPackageReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	packageManager PackageRevisionManager,
	templateEngine templates.TemplateEngine,
	yangValidator yang.YANGValidator,
	porchClient porch.PorchClient,
	lifecycleManager porch.LifecycleManager,
	config *IntegrationConfig,
) *NetworkIntentPackageReconciler {
	if config == nil {
		config = GetDefaultIntegrationConfig()
	}

	return &NetworkIntentPackageReconciler{
		Client:           client,
		Scheme:           scheme,
		Logger:           ctrl.Log.WithName("networkintent-package-reconciler"),
		PackageManager:   packageManager,
		TemplateEngine:   templateEngine,
		YANGValidator:    yangValidator,
		PorchClient:      porchClient,
		LifecycleManager: lifecycleManager,
		Config:           config,
	}
}

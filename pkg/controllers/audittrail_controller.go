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

package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nephv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
)

// AuditTrailController reconciles AuditTrail objects
type AuditTrailController struct {
	client.Client
	Scheme *runtime.Scheme

	// Track active audit systems
	auditSystems map[string]*audit.AuditSystem
	mutex        sync.RWMutex
}

// NewAuditTrailController creates a new AuditTrailController
func NewAuditTrailController(client client.Client, scheme *runtime.Scheme) *AuditTrailController {
	return &AuditTrailController{
		Client:       client,
		Scheme:       scheme,
		auditSystems: make(map[string]*audit.AuditSystem),
	}
}

//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=audittrails,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=audittrails/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=audittrails/finalizers,verbs=update

// Reconcile manages the lifecycle of AuditTrail resources
func (r *AuditTrailController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("audittrail", req.NamespacedName)

	// Fetch the AuditTrail instance
	var auditTrail nephv1.AuditTrail
	if err := r.Get(ctx, req.NamespacedName, &auditTrail); err != nil {
		if errors.IsNotFound(err) {
			// Remove audit system if it exists
			r.removeAuditSystem(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get AuditTrail")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !auditTrail.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &auditTrail)
	}

	// Validate audit trail configuration
	if err := r.validateAuditTrail(&auditTrail); err != nil {
		logger.Error(err, "Invalid audit trail configuration")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Ensure audit system exists
	if err := r.ensureAuditSystem(ctx, &auditTrail); err != nil {
		logger.Error(err, "Failed to ensure audit system")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status
	if err := r.updateStatus(ctx, &auditTrail); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, err
	}

	logger.Info("Successfully reconciled AuditTrail")
	return ctrl.Result{RequeueAfter: time.Minute * 10}, nil
}

// handleDeletion handles the cleanup when an AuditTrail is being deleted
func (r *AuditTrailController) handleDeletion(ctx context.Context, auditTrail *nephv1.AuditTrail) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("audittrail", auditTrail.Name)

	// Remove from our tracking
	key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)
	r.removeAuditSystem(key)

	// Remove finalizer if present
	controllerutil.RemoveFinalizer(auditTrail, "audittrail.nephoran.nephio.org/finalizer")
	if err := r.Update(ctx, auditTrail); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully cleaned up AuditTrail")
	return ctrl.Result{}, nil
}

// ensureAuditSystem ensures that an audit system exists for the given AuditTrail
func (r *AuditTrailController) ensureAuditSystem(ctx context.Context, auditTrail *nephv1.AuditTrail) error {
	key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if audit system already exists
	if _, exists := r.auditSystems[key]; exists {
		return nil
	}

	// Create new audit system
	config := &audit.AuditConfig{
		Enabled:         auditTrail.Spec.Enabled,
		EnabledSources:  auditTrail.Spec.ComplianceMode,
		RetentionDays:   365, // Default retention
		Storage: audit.StorageConfig{
			Type: "local", // Default to local storage
		},
		BatchSize:       auditTrail.Spec.BatchSize,
		MaxQueueSize:    auditTrail.Spec.MaxQueueSize,
		EnableIntegrity: auditTrail.Spec.EnableIntegrity,
		ComplianceMode:  convertComplianceMode(auditTrail.Spec.ComplianceMode),
	}

	// Set retention policy if available
	if auditTrail.Spec.RetentionPolicy != nil {
		config.RetentionDays = auditTrail.Spec.RetentionPolicy.DefaultRetention
	}

	// Configure backends from the spec
	for _, backend := range auditTrail.Spec.Backends {
		// TODO: Convert AuditBackendConfig to backends.BackendConfig
		// For now, use default local storage
		_ = backend // Avoid unused variable error
	}

	// Create audit system
	auditSystem, err := audit.NewAuditSystem(config)
	if err != nil {
		return fmt.Errorf("failed to create audit system: %w", err)
	}

	// Start the audit system
	if err := auditSystem.Start(ctx); err != nil {
		return fmt.Errorf("failed to start audit system: %w", err)
	}

	// Store in our tracking map
	r.auditSystems[key] = auditSystem

	return nil
}

// removeAuditSystem removes an audit system from tracking and stops it
func (r *AuditTrailController) removeAuditSystem(key string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if auditSystem, exists := r.auditSystems[key]; exists {
		auditSystem.Stop()
		delete(r.auditSystems, key)
	}
}

// updateStatus updates the status of the AuditTrail resource
func (r *AuditTrailController) updateStatus(ctx context.Context, auditTrail *nephv1.AuditTrail) error {
	key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)

	r.mutex.RLock()
	auditSystem, exists := r.auditSystems[key]
	r.mutex.RUnlock()

	if !exists {
		auditTrail.Status.Phase = "Failed"
		// Add condition instead of message
		condition := metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionFalse,
			Reason: "AuditSystemNotFound",
			Message: "Audit system not found",
			LastTransitionTime: metav1.Now(),
		}
		auditTrail.Status.Conditions = []metav1.Condition{condition}
		return r.Status().Update(ctx, auditTrail)
	}

	// Get metrics from audit system
	metrics := auditSystem.GetMetrics()
	
	// Update status based on audit system state
	if auditSystem.IsHealthy() {
		auditTrail.Status.Phase = "Running"
		// Set ready condition
		condition := metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
			Reason: "AuditSystemRunning",
			Message: "Audit system is running",
			LastTransitionTime: metav1.Now(),
		}
		auditTrail.Status.Conditions = []metav1.Condition{condition}
	} else {
		auditTrail.Status.Phase = "Failed"
		condition := metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionFalse,
			Reason: "AuditSystemDegraded",
			Message: "Audit system is experiencing issues",
			LastTransitionTime: metav1.Now(),
		}
		auditTrail.Status.Conditions = []metav1.Condition{condition}
	}

	// Update stats
	if auditTrail.Status.Stats == nil {
		auditTrail.Status.Stats = &nephv1.AuditTrailStats{}
	}
	auditTrail.Status.Stats.EventsReceived = metrics.TotalEvents
	auditTrail.Status.Stats.EventsProcessed = metrics.EventsProcessed
	auditTrail.Status.Stats.EventsDropped = metrics.EventsDropped
	auditTrail.Status.Stats.QueueSize = metrics.QueueSize
	auditTrail.Status.Stats.BackendCount = metrics.BackendsTotal
	if !metrics.LastEventTime.IsZero() {
		auditTrail.Status.Stats.LastEventTime = &metav1.Time{Time: metrics.LastEventTime}
	}
	auditTrail.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	return r.Status().Update(ctx, auditTrail)
}

// validateAuditTrail validates the AuditTrail specification
func (r *AuditTrailController) validateAuditTrail(auditTrail *nephv1.AuditTrail) error {
	// Validate compliance modes (audit sources)
	validCompliance := map[string]bool{
		"soc2":      true,
		"iso27001":  true,
		"pci_dss":   true,
		"hipaa":     true,
		"gdpr":      true,
		"ccpa":      true,
		"fisma":     true,
		"nist_csf":  true,
	}

	for _, compliance := range auditTrail.Spec.ComplianceMode {
		if !validCompliance[compliance] {
			return fmt.Errorf("invalid compliance mode: %s", compliance)
		}
	}

	// Validate retention policy if provided
	if auditTrail.Spec.RetentionPolicy != nil {
		if auditTrail.Spec.RetentionPolicy.DefaultRetention <= 0 {
			return fmt.Errorf("retention days must be positive")
		}
	}

	// Validate backends
	for _, backend := range auditTrail.Spec.Backends {
		if backend.Type == "" {
			return fmt.Errorf("backend type is required")
		}
		if backend.Name == "" {
			return fmt.Errorf("backend name is required")
		}
	}

	return nil
}

// handleAuditEvent processes incoming audit events
func (r *AuditTrailController) handleAuditEvent(ctx context.Context, event *audit.Event) error {
	logger := log.FromContext(ctx).WithValues("event", event.ID)

	// Find appropriate audit systems based on event source
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for key, auditSystem := range r.auditSystems {
		// Check if this audit system should handle this event
		if auditSystem.ShouldProcessEvent(event) {
			if err := auditSystem.ProcessEvent(ctx, event); err != nil {
				logger.Error(err, "Failed to process audit event", "auditSystem", key)
				// Continue processing with other audit systems
			}
		}
	}

	return nil
}

// GetAuditTrails returns all active audit trails
func (r *AuditTrailController) GetAuditTrails(ctx context.Context, namespace string) (*nephv1.AuditTrailList, error) {
	var auditTrails nephv1.AuditTrailList
	
	listOpts := []client.ListOption{}
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if err := r.List(ctx, &auditTrails, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list audit trails: %w", err)
	}

	return &auditTrails, nil
}

// ExportAuditLogs exports audit logs for a specific trail
func (r *AuditTrailController) ExportAuditLogs(ctx context.Context, trailName, namespace string, startTime, endTime time.Time) ([]byte, error) {
	key := fmt.Sprintf("%s/%s", namespace, trailName)

	r.mutex.RLock()
	auditSystem, exists := r.auditSystems[key]
	r.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("audit trail not found: %s", key)
	}

	return auditSystem.ExportLogs(ctx, startTime, endTime)
}

// GetAuditMetrics returns metrics for a specific audit trail
func (r *AuditTrailController) GetAuditMetrics(trailName, namespace string) (*audit.Metrics, error) {
	key := fmt.Sprintf("%s/%s", namespace, trailName)

	r.mutex.RLock()
	auditSystem, exists := r.auditSystems[key]
	r.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("audit trail not found: %s", key)
	}

	metrics := auditSystem.GetMetrics()
	return &metrics, nil
}

// SearchAuditEvents searches for audit events based on criteria
func (r *AuditTrailController) SearchAuditEvents(ctx context.Context, trailName, namespace string, criteria *audit.SearchCriteria) ([]*audit.Event, error) {
	key := fmt.Sprintf("%s/%s", namespace, trailName)

	r.mutex.RLock()
	auditSystem, exists := r.auditSystems[key]
	r.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("audit trail not found: %s", key)
	}

	return auditSystem.SearchEvents(ctx, criteria)
}

// HealthCheck performs a health check on all audit systems
func (r *AuditTrailController) HealthCheck(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var unhealthy []string
	for key, auditSystem := range r.auditSystems {
		if !auditSystem.IsHealthy() {
			unhealthy = append(unhealthy, key)
		}
	}

	if len(unhealthy) > 0 {
		return fmt.Errorf("unhealthy audit systems: %v", unhealthy)
	}

	return nil
}

// SetupWebhooks sets up webhooks for the controller
func (r *AuditTrailController) SetupWebhooks(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&nephv1.AuditTrail{}).
		Complete()
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuditTrailController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephv1.AuditTrail{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1, // Ensure sequential processing
		}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// GetAuditSystem returns the audit system for a given key (for testing or external access)
func (r *AuditTrailController) GetAuditSystem(namespace, name string) *audit.AuditSystem {
	key := fmt.Sprintf("%s/%s", namespace, name)
	return r.auditSystems[key]
}

// convertComplianceMode converts string compliance modes to audit package types
func convertComplianceMode(modes []string) []audit.ComplianceStandard {
	var result []audit.ComplianceStandard
	for _, mode := range modes {
		switch mode {
		case "soc2":
			result = append(result, audit.ComplianceSOC2)
		case "iso27001":
			result = append(result, audit.ComplianceISO27001)
		case "pci_dss":
			result = append(result, audit.CompliancePCIDSS)
		case "hipaa":
			result = append(result, audit.ComplianceHIPAA)
		case "gdpr":
			result = append(result, audit.ComplianceGDPR)
		case "ccpa":
			result = append(result, audit.ComplianceCCPA)
		case "fisma":
			result = append(result, audit.ComplianceFISMA)
		case "nist_csf":
			result = append(result, audit.ComplianceNIST)
		}
	}
	return result
}

// Cleanup stops all audit systems when the controller shuts down
func (r *AuditTrailController) Cleanup() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for key, auditSystem := range r.auditSystems {
		auditSystem.Stop()
		delete(r.auditSystems, key)
	}
}
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
	"k8s.io/apimachinery/pkg/types"
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
		EnabledSources: auditTrail.Spec.Sources,
		RetentionDays:  auditTrail.Spec.RetentionDays,
		Storage: audit.StorageConfig{
			Type: auditTrail.Spec.Storage.Type,
		},
	}

	// Set storage configuration based on type
	switch auditTrail.Spec.Storage.Type {
	case "elasticsearch":
		if auditTrail.Spec.Storage.Elasticsearch != nil {
			config.Storage.Elasticsearch = &audit.ElasticsearchConfig{
				URL:      auditTrail.Spec.Storage.Elasticsearch.URL,
				Index:    auditTrail.Spec.Storage.Elasticsearch.Index,
				Username: auditTrail.Spec.Storage.Elasticsearch.Username,
				Password: auditTrail.Spec.Storage.Elasticsearch.Password,
			}
		}
	case "s3":
		if auditTrail.Spec.Storage.S3 != nil {
			config.Storage.S3 = &audit.S3Config{
				Bucket:          auditTrail.Spec.Storage.S3.Bucket,
				Region:          auditTrail.Spec.Storage.S3.Region,
				AccessKeyID:     auditTrail.Spec.Storage.S3.AccessKeyID,
				SecretAccessKey: auditTrail.Spec.Storage.S3.SecretAccessKey,
			}
		}
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
		auditTrail.Status.Message = "Audit system not found"
		return r.Status().Update(ctx, auditTrail)
	}

	// Get metrics from audit system
	metrics := auditSystem.GetMetrics()
	
	// Update status based on audit system state
	if auditSystem.IsHealthy() {
		auditTrail.Status.Phase = "Active"
		auditTrail.Status.Message = "Audit system is running"
		auditTrail.Status.EventsCollected = int64(metrics.EventsProcessed)
		auditTrail.Status.LastEventTime = &metav1.Time{Time: metrics.LastEventTime}
	} else {
		auditTrail.Status.Phase = "Degraded"
		auditTrail.Status.Message = "Audit system is experiencing issues"
	}

	// Update storage status
	auditTrail.Status.StorageStatus = &nephv1.StorageStatus{
		Available:   true,
		UsedBytes:   metrics.StorageUsed,
		TotalEvents: int64(metrics.TotalEvents),
	}

	return r.Status().Update(ctx, auditTrail)
}

// validateAuditTrail validates the AuditTrail specification
func (r *AuditTrailController) validateAuditTrail(auditTrail *nephv1.AuditTrail) error {
	// Validate sources
	if len(auditTrail.Spec.Sources) == 0 {
		return fmt.Errorf("at least one audit source must be specified")
	}

	validSources := map[string]bool{
		"kubernetes": true,
		"oran":       true,
		"nephio":     true,
		"custom":     true,
	}

	for _, source := range auditTrail.Spec.Sources {
		if !validSources[source] {
			return fmt.Errorf("invalid audit source: %s", source)
		}
	}

	// Validate retention
	if auditTrail.Spec.RetentionDays <= 0 {
		return fmt.Errorf("retention days must be positive")
	}

	// Validate storage
	switch auditTrail.Spec.Storage.Type {
	case "elasticsearch":
		if auditTrail.Spec.Storage.Elasticsearch == nil {
			return fmt.Errorf("elasticsearch configuration is required for elasticsearch storage")
		}
		if auditTrail.Spec.Storage.Elasticsearch.URL == "" {
			return fmt.Errorf("elasticsearch URL is required")
		}
	case "s3":
		if auditTrail.Spec.Storage.S3 == nil {
			return fmt.Errorf("s3 configuration is required for s3 storage")
		}
		if auditTrail.Spec.Storage.S3.Bucket == "" {
			return fmt.Errorf("s3 bucket is required")
		}
	case "local":
		// Local storage is always valid
	default:
		return fmt.Errorf("unsupported storage type: %s", auditTrail.Spec.Storage.Type)
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

// Cleanup stops all audit systems when the controller shuts down
func (r *AuditTrailController) Cleanup() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for key, auditSystem := range r.auditSystems {
		auditSystem.Stop()
		delete(r.auditSystems, key)
	}
}
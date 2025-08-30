package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nephv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/audit"
	"github.com/nephio-project/nephoran-intent-operator/pkg/audit/backends"
)

const (

	// AuditTrailFinalizer is the finalizer used for AuditTrail resources.

	AuditTrailFinalizer = "audittrail.nephoran.io/finalizer"

	// ConditionTypes for AuditTrail status.

	ConditionTypeReady = "Ready"

	// ConditionTypeProgressing holds conditiontypeprogressing value.

	ConditionTypeProgressing = "Progressing"

	// ConditionTypeBackendsReady holds conditiontypebackendsready value.

	ConditionTypeBackendsReady = "BackendsReady"

	// ConditionTypeIntegrityReady holds conditiontypeintegrityready value.

	ConditionTypeIntegrityReady = "IntegrityReady"

	// Event reasons.

	EventReasonCreated = "Created"

	// EventReasonUpdated holds eventreasonupdated value.

	EventReasonUpdated = "Updated"

	// EventReasonStarted holds eventreasonstarted value.

	EventReasonStarted = "Started"

	// EventReasonStopped holds eventreasonstopped value.

	EventReasonStopped = "Stopped"

	// EventReasonBackendFailed holds eventreasonbackendfailed value.

	EventReasonBackendFailed = "BackendFailed"

	// EventReasonIntegrityFailed holds eventreasonintegrityfailed value.

	EventReasonIntegrityFailed = "IntegrityFailed"

	// EventReasonConfigChanged holds eventreasonconfigchanged value.

	EventReasonConfigChanged = "ConfigChanged"
)

// AuditTrailController reconciles a AuditTrail object.

type AuditTrailController struct {
	client.Client

	Log logr.Logger

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	// Audit system instances managed by this controller.

	auditSystems map[string]*audit.AuditSystem
}

// NewAuditTrailController creates a new AuditTrailController.

func NewAuditTrailController(client client.Client, log logr.Logger, scheme *runtime.Scheme, recorder record.EventRecorder) *AuditTrailController {

	return &AuditTrailController{

		Client: client,

		Log: log,

		Scheme: scheme,

		Recorder: recorder,

		auditSystems: make(map[string]*audit.AuditSystem),
	}

}

// +kubebuilder:rbac:groups=nephoran.io,resources=audittrails,verbs=get;list;watch;create;update;patch;delete.

// +kubebuilder:rbac:groups=nephoran.io,resources=audittrails/status,verbs=get;update;patch.

// +kubebuilder:rbac:groups=nephoran.io,resources=audittrails/finalizers,verbs=update.

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch.

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch.

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch.

// Reconcile is part of the main kubernetes reconciliation loop which aims to.

// move the current state of the cluster closer to the desired state.

func (r *AuditTrailController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("audittrail", req.NamespacedName)

	// Fetch the AuditTrail instance.

	auditTrail := &nephv1.AuditTrail{}

	if err := r.Get(ctx, req.NamespacedName, auditTrail); err != nil {

		if apierrors.IsNotFound(err) {

			// Object not found, could have been deleted.

			log.Info("AuditTrail resource not found, cleaning up")

			return r.cleanupAuditSystem(req.NamespacedName.String())

		}

		log.Error(err, "Unable to fetch AuditTrail")

		return ctrl.Result{}, err

	}

	// Handle deletion.

	if !auditTrail.DeletionTimestamp.IsZero() {

		return r.handleDeletion(ctx, auditTrail, log)

	}

	// Add finalizer if not present.

	if !controllerutil.ContainsFinalizer(auditTrail, AuditTrailFinalizer) {

		log.Info("Adding finalizer to AuditTrail")

		controllerutil.AddFinalizer(auditTrail, AuditTrailFinalizer)

		return ctrl.Result{}, r.Update(ctx, auditTrail)

	}

	// Initialize or update the audit system.

	result, err := r.reconcileAuditSystem(ctx, auditTrail, log)

	if err != nil {

		return result, r.updateStatusError(ctx, auditTrail, err, log)

	}

	// Update status with current state.

	return result, r.updateStatus(ctx, auditTrail, log)

}

// handleDeletion handles the deletion of an AuditTrail resource.

func (r *AuditTrailController) handleDeletion(ctx context.Context, auditTrail *nephv1.AuditTrail, log logr.Logger) (ctrl.Result, error) {

	if controllerutil.ContainsFinalizer(auditTrail, AuditTrailFinalizer) {

		log.Info("Finalizing AuditTrail")

		// Stop and cleanup the audit system.

		key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)

		if auditSystem, exists := r.auditSystems[key]; exists {

			log.Info("Stopping audit system")

			if err := auditSystem.Stop(); err != nil {

				log.Error(err, "Failed to stop audit system")

				r.Recorder.Event(auditTrail, corev1.EventTypeWarning, EventReasonStopped,

					fmt.Sprintf("Failed to stop audit system: %v", err))

			} else {

				r.Recorder.Event(auditTrail, corev1.EventTypeNormal, EventReasonStopped,

					"Audit system stopped successfully")

			}

			delete(r.auditSystems, key)

		}

		// Remove finalizer.

		controllerutil.RemoveFinalizer(auditTrail, AuditTrailFinalizer)

		return ctrl.Result{}, r.Update(ctx, auditTrail)

	}

	return ctrl.Result{}, nil

}

// reconcileAuditSystem creates or updates the audit system based on the AuditTrail spec.

func (r *AuditTrailController) reconcileAuditSystem(ctx context.Context, auditTrail *nephv1.AuditTrail, log logr.Logger) (ctrl.Result, error) {

	key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)

	// Check if audit system exists and if configuration has changed.

	existingSystem, exists := r.auditSystems[key]

	needsRecreation := false

	if exists {

		// Compare current configuration with desired configuration.

		// This is a simplified check - in practice, you'd compare more thoroughly.

		if r.hasConfigurationChanged(auditTrail) {

			log.Info("Configuration changed, recreating audit system")

			needsRecreation = true

		}

	}

	if needsRecreation {

		// Stop existing system.

		if err := existingSystem.Stop(); err != nil {

			log.Error(err, "Failed to stop existing audit system")

		}

		delete(r.auditSystems, key)

		exists = false

	}

	if !exists || needsRecreation {

		// Create new audit system.

		log.Info("Creating new audit system")

		// Convert AuditTrail spec to audit system configuration.

		config, err := r.buildAuditSystemConfig(ctx, auditTrail, log)

		if err != nil {

			return ctrl.Result{}, fmt.Errorf("failed to build audit system config: %w", err)

		}

		// Create audit system.

		auditSystem, err := audit.NewAuditSystem(config)

		if err != nil {

			return ctrl.Result{}, fmt.Errorf("failed to create audit system: %w", err)

		}

		// Start audit system if enabled.

		if auditTrail.Spec.Enabled {

			if err := auditSystem.Start(); err != nil {

				return ctrl.Result{}, fmt.Errorf("failed to start audit system: %w", err)

			}

			r.Recorder.Event(auditTrail, corev1.EventTypeNormal, EventReasonStarted,

				"Audit system started successfully")

		}

		r.auditSystems[key] = auditSystem

		r.Recorder.Event(auditTrail, corev1.EventTypeNormal, EventReasonCreated,

			"Audit system created successfully")

	} else {

		// Handle enable/disable changes.

		if auditTrail.Spec.Enabled {

			// Ensure system is started.

			stats := existingSystem.GetStats()

			if stats.EventsReceived == 0 && stats.EventsDropped == 0 {

				// System might not be started.

				if err := existingSystem.Start(); err != nil {

					log.Error(err, "Failed to start existing audit system")

				}

			}

		} else {

			// Stop system if it's running.

			if err := existingSystem.Stop(); err != nil {

				log.Error(err, "Failed to stop audit system")

			}

		}

	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

}

// buildAuditSystemConfig converts AuditTrail spec to audit system configuration.

func (r *AuditTrailController) buildAuditSystemConfig(ctx context.Context, auditTrail *nephv1.AuditTrail, log logr.Logger) (*audit.AuditSystemConfig, error) {

	config := &audit.AuditSystemConfig{

		Enabled: auditTrail.Spec.Enabled,

		LogLevel: r.convertLogLevel(auditTrail.Spec.LogLevel),

		BatchSize: auditTrail.Spec.BatchSize,

		FlushInterval: time.Duration(auditTrail.Spec.FlushInterval) * time.Second,

		MaxQueueSize: auditTrail.Spec.MaxQueueSize,

		EnableIntegrity: auditTrail.Spec.EnableIntegrity,

		ComplianceMode: r.convertComplianceMode(auditTrail.Spec.ComplianceMode),
	}

	// Set defaults if not specified.

	if config.BatchSize == 0 {

		config.BatchSize = 100

	}

	if config.FlushInterval == 0 {

		config.FlushInterval = 10 * time.Second

	}

	if config.MaxQueueSize == 0 {

		config.MaxQueueSize = 10000

	}

	// Build backend configurations.

	backendConfigs := make([]backends.BackendConfig, 0, len(auditTrail.Spec.Backends))

	for _, backend := range auditTrail.Spec.Backends {

		backendConfig, err := r.buildBackendConfig(ctx, auditTrail, backend, log)

		if err != nil {

			return nil, fmt.Errorf("failed to build backend config for %s: %w", backend.Name, err)

		}

		backendConfigs = append(backendConfigs, *backendConfig)

	}

	config.Backends = backendConfigs

	return config, nil

}

// buildBackendConfig converts AuditBackendConfig to backend configuration.

func (r *AuditTrailController) buildBackendConfig(ctx context.Context, auditTrail *nephv1.AuditTrail, spec nephv1.AuditBackendConfig, log logr.Logger) (*backends.BackendConfig, error) {

	config := &backends.BackendConfig{

		Type: backends.BackendType(spec.Type),

		Enabled: spec.Enabled,

		Name: spec.Name,

		Format: spec.Format,

		Compression: spec.Compression,

		BufferSize: spec.BufferSize,

		Timeout: time.Duration(spec.Timeout) * time.Second,
	}

	// Set defaults.

	if config.BufferSize == 0 {

		config.BufferSize = 1000

	}

	if config.Timeout == 0 {

		config.Timeout = 30 * time.Second

	}

	if config.Format == "" {

		config.Format = "json"

	}

	// Convert settings from RawExtension.

	if spec.Settings.Raw != nil {

		settings := make(map[string]interface{})

		// In a real implementation, you'd unmarshal the RawExtension properly.

		// For now, we'll create a basic settings map.

		settings["raw_config"] = string(spec.Settings.Raw)

		config.Settings = settings

	}

	// Convert retry policy.

	if spec.RetryPolicy != nil {

		config.RetryPolicy = backends.RetryPolicy{

			MaxRetries: spec.RetryPolicy.MaxRetries,

			InitialDelay: time.Duration(spec.RetryPolicy.InitialDelay) * time.Second,

			MaxDelay: time.Duration(spec.RetryPolicy.MaxDelay) * time.Second,

			BackoffFactor: spec.RetryPolicy.BackoffFactor,
		}

	} else {

		config.RetryPolicy = backends.DefaultRetryPolicy()

	}

	// Convert TLS config.

	if spec.TLS != nil {

		config.TLS = backends.TLSConfig{

			Enabled: spec.TLS.Enabled,

			CertFile: spec.TLS.CertFile,

			KeyFile: spec.TLS.KeyFile,

			CAFile: spec.TLS.CAFile,

			ServerName: spec.TLS.ServerName,

			InsecureSkipVerify: spec.TLS.InsecureSkipVerify,
		}

	}

	// Convert filter config.

	if spec.Filter != nil {

		config.Filter = backends.FilterConfig{

			MinSeverity: r.convertLogLevel(spec.Filter.MinSeverity),

			EventTypes: r.convertEventTypes(spec.Filter.EventTypes),

			Components: spec.Filter.Components,

			ExcludeTypes: r.convertEventTypes(spec.Filter.ExcludeTypes),

			IncludeFields: spec.Filter.IncludeFields,

			ExcludeFields: spec.Filter.ExcludeFields,
		}

	} else {

		config.Filter = backends.DefaultFilterConfig()

	}

	return config, nil

}

// updateStatus updates the AuditTrail status with current information.

func (r *AuditTrailController) updateStatus(ctx context.Context, auditTrail *nephv1.AuditTrail, log logr.Logger) error {

	key := fmt.Sprintf("%s/%s", auditTrail.Namespace, auditTrail.Name)

	// Get current status from audit system.

	var stats audit.AuditStats

	var phase string

	if auditSystem, exists := r.auditSystems[key]; exists {

		stats = auditSystem.GetStats()

		if auditTrail.Spec.Enabled {

			phase = "Running"

		} else {

			phase = "Stopped"

		}

	} else {

		phase = "Initializing"

	}

	// Build status.

	now := metav1.Now()

	status := nephv1.AuditTrailStatus{

		Phase: phase,

		LastUpdate: &now,

		Stats: &nephv1.AuditTrailStats{

			EventsReceived: stats.EventsReceived,

			EventsProcessed: stats.EventsReceived - stats.EventsDropped,

			EventsDropped: stats.EventsDropped,

			QueueSize: stats.QueueSize,

			BackendCount: stats.BackendCount,
		},
	}

	// Build conditions.

	conditions := []metav1.Condition{

		{

			Type: ConditionTypeReady,

			Status: metav1.ConditionTrue,

			Reason: "SystemOperational",

			Message: "Audit system is operational",

			LastTransitionTime: now,
		},
	}

	if phase != "Running" {

		conditions[0].Status = metav1.ConditionFalse

		conditions[0].Reason = "SystemNotRunning"

		conditions[0].Message = fmt.Sprintf("Audit system is in phase: %s", phase)

	}

	status.Conditions = conditions

	// Update status if changed.

	if !reflect.DeepEqual(auditTrail.Status, status) {

		auditTrail.Status = status

		return r.Status().Update(ctx, auditTrail)

	}

	return nil

}

// updateStatusError updates the status with error information.

func (r *AuditTrailController) updateStatusError(ctx context.Context, auditTrail *nephv1.AuditTrail, err error, log logr.Logger) error {

	now := metav1.Now()

	auditTrail.Status.Phase = "Failed"

	auditTrail.Status.LastUpdate = &now

	auditTrail.Status.Conditions = []metav1.Condition{

		{

			Type: ConditionTypeReady,

			Status: metav1.ConditionFalse,

			Reason: "Error",

			Message: err.Error(),

			LastTransitionTime: now,
		},
	}

	r.Recorder.Event(auditTrail, corev1.EventTypeWarning, "ReconcileError",

		fmt.Sprintf("Failed to reconcile: %v", err))

	if updateErr := r.Status().Update(ctx, auditTrail); updateErr != nil {

		log.Error(updateErr, "Failed to update status")

		return updateErr

	}

	return err

}

// cleanupAuditSystem cleans up an audit system when the resource is deleted.

func (r *AuditTrailController) cleanupAuditSystem(key string) (ctrl.Result, error) {

	if auditSystem, exists := r.auditSystems[key]; exists {

		if err := auditSystem.Stop(); err != nil {

			r.Log.Error(err, "Failed to stop audit system during cleanup", "key", key)

		}

		delete(r.auditSystems, key)

	}

	return ctrl.Result{}, nil

}

// hasConfigurationChanged checks if the configuration has changed.

func (r *AuditTrailController) hasConfigurationChanged(auditTrail *nephv1.AuditTrail) bool {

	// This is a simplified implementation.

	// In practice, you'd store the last known configuration and compare.

	return false

}

// Helper functions for conversion.

func (r *AuditTrailController) convertLogLevel(level string) audit.Severity {

	switch level {

	case "emergency":

		return audit.SeverityEmergency

	case "alert":

		return audit.SeverityAlert

	case "critical":

		return audit.SeverityCritical

	case "error":

		return audit.SeverityError

	case "warning":

		return audit.SeverityWarning

	case "notice":

		return audit.SeverityNotice

	case "info":

		return audit.SeverityInfo

	case "debug":

		return audit.SeverityDebug

	default:

		return audit.SeverityInfo

	}

}

func (r *AuditTrailController) convertComplianceMode(modes []string) []audit.ComplianceStandard {

	standards := make([]audit.ComplianceStandard, 0, len(modes))

	for _, mode := range modes {

		switch mode {

		case "soc2":

			standards = append(standards, audit.ComplianceSOC2)

		case "iso27001":

			standards = append(standards, audit.ComplianceISO27001)

		case "pci_dss":

			standards = append(standards, audit.CompliancePCIDSS)

		case "hipaa":

			standards = append(standards, audit.ComplianceHIPAA)

		case "gdpr":

			standards = append(standards, audit.ComplianceGDPR)

		case "ccpa":

			standards = append(standards, audit.ComplianceCCPA)

		case "fisma":

			standards = append(standards, audit.ComplianceFISMA)

		case "nist_csf":

			standards = append(standards, audit.ComplianceNIST)

		}

	}

	return standards

}

func (r *AuditTrailController) convertEventTypes(types []string) []audit.EventType {

	eventTypes := make([]audit.EventType, 0, len(types))

	for _, t := range types {

		eventTypes = append(eventTypes, audit.EventType(t))

	}

	return eventTypes

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

// GetAuditSystem returns the audit system for a given key (for testing or external access).

func (r *AuditTrailController) GetAuditSystem(namespace, name string) *audit.AuditSystem {

	key := fmt.Sprintf("%s/%s", namespace, name)

	return r.auditSystems[key]

}

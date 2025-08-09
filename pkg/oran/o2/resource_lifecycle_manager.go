// Package o2 implements comprehensive Resource Lifecycle Management
// Following O-RAN.WG6.O2ims-Interface-v01.01 specification and cloud-native best practices
package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// ResourceLifecycleManager implements comprehensive resource lifecycle operations
type ResourceLifecycleManager struct {
	config           *O2IMSConfig
	logger           *logging.StructuredLogger
	providerRegistry *providers.ProviderRegistry
	storage          O2IMSStorage

	// Resource tracking and state management
	resourceStates map[string]*ResourceState
	stateMutex     sync.RWMutex

	// Operation queues and workers
	operationQueue chan *ResourceOperation
	workers        int
	stopCh         chan struct{}

	// Event handling
	eventBus *ResourceEventBus

	// Metrics and monitoring
	metrics *ResourceLifecycleMetrics

	// Configuration and policy
	policies *ResourcePolicies

	// Background services
	ctx    context.Context
	cancel context.CancelFunc
}

// ResourceState represents the current state of a resource
type ResourceState struct {
	ResourceID      string                 `json:"resource_id"`
	CurrentStatus   string                 `json:"current_status"`
	DesiredStatus   string                 `json:"desired_status"`
	LastOperation   string                 `json:"last_operation"`
	OperationStatus string                 `json:"operation_status"`
	Provider        string                 `json:"provider"`
	ResourceType    string                 `json:"resource_type"`
	Configuration   *runtime.RawExtension  `json:"configuration"`
	Metadata        map[string]interface{} `json:"metadata"`
	LastUpdated     time.Time              `json:"last_updated"`
	HealthStatus    string                 `json:"health_status"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	NextRetry       time.Time              `json:"next_retry,omitempty"`
}

// ResourceOperation represents a lifecycle operation to be performed
type ResourceOperation struct {
	OperationID   string                 `json:"operation_id"`
	ResourceID    string                 `json:"resource_id"`
	OperationType string                 `json:"operation_type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Priority      int                    `json:"priority"`
	Timeout       time.Duration          `json:"timeout"`
	Callback      OperationCallback      `json:"-"`
	CreatedAt     time.Time              `json:"created_at"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Status        string                 `json:"status"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

// OperationCallback is called when an operation completes
type OperationCallback func(ctx context.Context, operation *ResourceOperation, result *OperationResult)

// OperationResult represents the result of a resource operation
type OperationResult struct {
	Success      bool                   `json:"success"`
	ResourceID   string                 `json:"resource_id"`
	Operation    string                 `json:"operation"`
	Result       interface{}            `json:"result,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ResourceEventBus handles resource lifecycle events
type ResourceEventBus struct {
	subscribers map[string][]EventSubscriber
	mutex       sync.RWMutex
	logger      *logging.StructuredLogger
}

// EventSubscriber represents an event subscriber
type EventSubscriber func(ctx context.Context, event *ResourceLifecycleEvent)

// ResourceLifecycleEvent represents a resource lifecycle event
type ResourceLifecycleEvent struct {
	EventID    string                 `json:"event_id"`
	EventType  string                 `json:"event_type"`
	ResourceID string                 `json:"resource_id"`
	Operation  string                 `json:"operation"`
	Status     string                 `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Metadata   map[string]interface{} `json:"metadata"`
	Error      string                 `json:"error,omitempty"`
}

// ResourceLifecycleMetrics tracks resource lifecycle metrics
type ResourceLifecycleMetrics struct {
	OperationsTotal     map[string]int64   `json:"operations_total"`
	OperationsSuccess   map[string]int64   `json:"operations_success"`
	OperationsFailure   map[string]int64   `json:"operations_failure"`
	OperationsDuration  map[string]float64 `json:"operations_duration"`
	ResourcesByStatus   map[string]int64   `json:"resources_by_status"`
	ResourcesByProvider map[string]int64   `json:"resources_by_provider"`
	LastUpdated         time.Time          `json:"last_updated"`
}

// ResourcePolicies defines policies for resource lifecycle management
type ResourcePolicies struct {
	RetryPolicy         *RetryPolicy         `json:"retry_policy"`
	TimeoutPolicy       *TimeoutPolicy       `json:"timeout_policy"`
	ScalingPolicy       *ScalingPolicy       `json:"scaling_policy"`
	BackupPolicy        *BackupPolicy        `json:"backup_policy"`
	MonitoringPolicy    *MonitoringPolicy    `json:"monitoring_policy"`
	CompliancePolicy    *CompliancePolicy    `json:"compliance_policy"`
	ResourceQuotaPolicy *ResourceQuotaPolicy `json:"resource_quota_policy"`
}

// RetryPolicy defines retry behavior for failed operations
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	RetryInterval   time.Duration `json:"retry_interval"`
	BackoffFactor   float64       `json:"backoff_factor"`
	MaxRetryDelay   time.Duration `json:"max_retry_delay"`
	RetriableErrors []string      `json:"retriable_errors"`
}

// TimeoutPolicy defines timeout behavior for operations
type TimeoutPolicy struct {
	DefaultTimeout    time.Duration            `json:"default_timeout"`
	OperationTimeouts map[string]time.Duration `json:"operation_timeouts"`
}

// ScalingPolicy defines automatic scaling behavior
type ScalingPolicy struct {
	AutoScalingEnabled  bool          `json:"auto_scaling_enabled"`
	ScaleUpThreshold    float64       `json:"scale_up_threshold"`
	ScaleDownThreshold  float64       `json:"scale_down_threshold"`
	ScaleCooldownPeriod time.Duration `json:"scale_cooldown_period"`
	MinReplicas         int32         `json:"min_replicas"`
	MaxReplicas         int32         `json:"max_replicas"`
	ScalingMetrics      []string      `json:"scaling_metrics"`
}

// BackupPolicy defines backup behavior for resources
type BackupPolicy struct {
	AutoBackupEnabled     bool          `json:"auto_backup_enabled"`
	BackupInterval        time.Duration `json:"backup_interval"`
	RetentionPeriod       time.Duration `json:"retention_period"`
	BackupStorageLocation string        `json:"backup_storage_location"`
	BackupEncryption      bool          `json:"backup_encryption"`
}

// MonitoringPolicy defines monitoring behavior
type MonitoringPolicy struct {
	HealthCheckInterval       time.Duration      `json:"health_check_interval"`
	MetricsCollectionInterval time.Duration      `json:"metrics_collection_interval"`
	AlarmingEnabled           bool               `json:"alarming_enabled"`
	AlarmThresholds           map[string]float64 `json:"alarm_thresholds"`
}

// CompliancePolicy defines compliance requirements
type CompliancePolicy struct {
	RequiredLabels      map[string]string `json:"required_labels"`
	RequiredAnnotations map[string]string `json:"required_annotations"`
	SecurityPolicies    []string          `json:"security_policies"`
	DataClassification  string            `json:"data_classification"`
}

// ResourceQuotaPolicy defines resource quota constraints
type ResourceQuotaPolicy struct {
	MaxResourcesPerProvider map[string]int    `json:"max_resources_per_provider"`
	MaxResourcesPerType     map[string]int    `json:"max_resources_per_type"`
	MaxTotalResources       int               `json:"max_total_resources"`
	ResourceLimits          map[string]string `json:"resource_limits"`
}

// NewResourceLifecycleManager creates a new resource lifecycle manager
func NewResourceLifecycleManager(config *O2IMSConfig, storage O2IMSStorage, providerRegistry *providers.ProviderRegistry, logger *logging.StructuredLogger) *ResourceLifecycleManager {
	if config == nil {
		config = DefaultO2IMSConfig()
	}

	if logger == nil {
		logger = logging.NewStructuredLogger(
			logging.WithService("o2-resource-lifecycle"),
			logging.WithVersion("1.0.0"),
		)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &ResourceLifecycleManager{
		config:           config,
		logger:           logger,
		providerRegistry: providerRegistry,
		storage:          storage,
		resourceStates:   make(map[string]*ResourceState),
		operationQueue:   make(chan *ResourceOperation, 1000),
		workers:          config.ResourceConfig.MaxConcurrentOperations,
		stopCh:           make(chan struct{}),
		eventBus:         newResourceEventBus(logger),
		metrics:          newResourceLifecycleMetrics(),
		policies:         newDefaultResourcePolicies(),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start background services
	go manager.startOperationWorkers()
	go manager.startStateReconciler()
	go manager.startMetricsCollection()
	go manager.startHealthChecker()

	return manager
}

// Resource Provisioning Operations

// ProvisionResource provisions a new infrastructure resource
func (rlm *ResourceLifecycleManager) ProvisionResource(ctx context.Context, req *ProvisionResourceRequest) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("provisioning resource",
		"resource_type", req.ResourceType,
		"provider", req.Provider,
		"name", req.Name)

	// Validate request
	if err := rlm.validateProvisionRequest(req); err != nil {
		return nil, fmt.Errorf("invalid provision request: %w", err)
	}

	// Check resource quotas
	if err := rlm.checkResourceQuotas(ctx, req); err != nil {
		return nil, fmt.Errorf("resource quota check failed: %w", err)
	}

	// Get provider
	provider, err := rlm.providerRegistry.GetProvider(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", req.Provider, err)
	}

	// Create resource state
	resourceID := rlm.generateResourceID(req.Provider, req.ResourceType, req.Name)
	resourceState := &ResourceState{
		ResourceID:      resourceID,
		CurrentStatus:   "PROVISIONING",
		DesiredStatus:   "ACTIVE",
		LastOperation:   "PROVISION",
		OperationStatus: "IN_PROGRESS",
		Provider:        req.Provider,
		ResourceType:    req.ResourceType,
		Configuration:   req.Configuration,
		Metadata:        req.Metadata,
		LastUpdated:     time.Now(),
		HealthStatus:    "UNKNOWN",
	}

	rlm.updateResourceState(resourceState)

	// Queue provision operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "PROVISION",
		Parameters: map[string]interface{}{
			"request": req,
		},
		Priority:  1,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["PROVISION"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Create and return resource model
	resource := &models.Resource{
		ResourceID:     resourceID,
		Name:           req.Name,
		ResourceTypeID: req.ResourceType,
		ResourcePoolID: req.ResourcePoolID,
		Status:         "PROVISIONING",
		GlobalAssetID:  resourceID,
		Provider:       req.Provider,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Store resource in persistent storage
	if err := rlm.storage.StoreResource(ctx, resource); err != nil {
		rlm.logger.Error("failed to store resource in persistent storage", "error", err, "resource_id", resourceID)
		// Continue - the resource is still being provisioned
	}

	// Emit provision started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "PROVISION_STARTED",
		ResourceID: resourceID,
		Operation:  "PROVISION",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata:   req.Metadata,
	})

	rlm.metrics.OperationsTotal["PROVISION"]++
	return resource, nil
}

// ConfigureResource applies configuration to an existing resource
func (rlm *ResourceLifecycleManager) ConfigureResource(ctx context.Context, resourceID string, config *runtime.RawExtension) error {
	logger := log.FromContext(ctx)
	logger.Info("configuring resource", "resource_id", resourceID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	// Validate current state allows configuration
	if !rlm.canConfigure(resourceState) {
		return fmt.Errorf("resource %s cannot be configured in current state %s", resourceID, resourceState.CurrentStatus)
	}

	// Update resource state
	resourceState.LastOperation = "CONFIGURE"
	resourceState.OperationStatus = "IN_PROGRESS"
	resourceState.Configuration = config
	resourceState.LastUpdated = time.Now()
	rlm.updateResourceState(resourceState)

	// Queue configure operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "CONFIGURE",
		Parameters: map[string]interface{}{
			"configuration": config,
		},
		Priority:  2,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["CONFIGURE"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit configuration started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "CONFIGURE_STARTED",
		ResourceID: resourceID,
		Operation:  "CONFIGURE",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
	})

	rlm.metrics.OperationsTotal["CONFIGURE"]++
	return nil
}

// ScaleResource scales a resource horizontally or vertically
func (rlm *ResourceLifecycleManager) ScaleResource(ctx context.Context, resourceID string, req *ScaleResourceRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling resource",
		"resource_id", resourceID,
		"scale_type", req.ScaleType,
		"target_replicas", req.TargetReplicas)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	// Validate scaling request
	if err := rlm.validateScaleRequest(resourceState, req); err != nil {
		return fmt.Errorf("invalid scale request: %w", err)
	}

	// Update resource state
	resourceState.LastOperation = "SCALE"
	resourceState.OperationStatus = "IN_PROGRESS"
	resourceState.LastUpdated = time.Now()
	rlm.updateResourceState(resourceState)

	// Queue scale operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "SCALE",
		Parameters: map[string]interface{}{
			"scale_type":       req.ScaleType,
			"target_replicas":  req.TargetReplicas,
			"target_resources": req.TargetResources,
		},
		Priority:  2,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["SCALE"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit scaling started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "SCALE_STARTED",
		ResourceID: resourceID,
		Operation:  "SCALE",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata: map[string]interface{}{
			"scale_type":      req.ScaleType,
			"target_replicas": req.TargetReplicas,
		},
	})

	rlm.metrics.OperationsTotal["SCALE"]++
	return nil
}

// MigrateResource migrates a resource between providers or locations
func (rlm *ResourceLifecycleManager) MigrateResource(ctx context.Context, resourceID string, req *MigrateResourceRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("migrating resource",
		"resource_id", resourceID,
		"source_provider", req.SourceProvider,
		"target_provider", req.TargetProvider)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	// Validate migration request
	if err := rlm.validateMigrationRequest(resourceState, req); err != nil {
		return fmt.Errorf("invalid migration request: %w", err)
	}

	// Update resource state
	resourceState.LastOperation = "MIGRATE"
	resourceState.OperationStatus = "IN_PROGRESS"
	resourceState.CurrentStatus = "MIGRATING"
	resourceState.LastUpdated = time.Now()
	rlm.updateResourceState(resourceState)

	// Queue migrate operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "MIGRATE",
		Parameters: map[string]interface{}{
			"source_provider": req.SourceProvider,
			"target_provider": req.TargetProvider,
			"target_location": req.TargetLocation,
			"migration_type":  req.MigrationType,
		},
		Priority:  1,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["MIGRATE"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit migration started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "MIGRATE_STARTED",
		ResourceID: resourceID,
		Operation:  "MIGRATE",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata: map[string]interface{}{
			"source_provider": req.SourceProvider,
			"target_provider": req.TargetProvider,
		},
	})

	rlm.metrics.OperationsTotal["MIGRATE"]++
	return nil
}

// BackupResource creates a backup of a resource
func (rlm *ResourceLifecycleManager) BackupResource(ctx context.Context, resourceID string, req *BackupResourceRequest) (*BackupInfo, error) {
	logger := log.FromContext(ctx)
	logger.Info("backing up resource", "resource_id", resourceID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	// Generate backup ID
	backupID := rlm.generateBackupID(resourceID)
	backupInfo := &BackupInfo{
		BackupID:       backupID,
		ResourceID:     resourceID,
		BackupType:     req.BackupType,
		Status:         "CREATING",
		CreatedAt:      time.Now(),
		RetentionUntil: time.Now().Add(rlm.policies.BackupPolicy.RetentionPeriod),
	}

	// Queue backup operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "BACKUP",
		Parameters: map[string]interface{}{
			"backup_id":   backupID,
			"backup_type": req.BackupType,
		},
		Priority:  3,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["BACKUP"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit backup started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "BACKUP_STARTED",
		ResourceID: resourceID,
		Operation:  "BACKUP",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata: map[string]interface{}{
			"backup_id":   backupID,
			"backup_type": req.BackupType,
		},
	})

	rlm.metrics.OperationsTotal["BACKUP"]++
	return backupInfo, nil
}

// RestoreResource restores a resource from backup
func (rlm *ResourceLifecycleManager) RestoreResource(ctx context.Context, resourceID string, backupID string) error {
	logger := log.FromContext(ctx)
	logger.Info("restoring resource", "resource_id", resourceID, "backup_id", backupID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	// Update resource state
	resourceState.LastOperation = "RESTORE"
	resourceState.OperationStatus = "IN_PROGRESS"
	resourceState.CurrentStatus = "RESTORING"
	resourceState.LastUpdated = time.Now()
	rlm.updateResourceState(resourceState)

	// Queue restore operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "RESTORE",
		Parameters: map[string]interface{}{
			"backup_id": backupID,
		},
		Priority:  1,
		Timeout:   rlm.policies.TimeoutPolicy.OperationTimeouts["RESTORE"],
		CreatedAt: time.Now(),
		Status:    "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit restore started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "RESTORE_STARTED",
		ResourceID: resourceID,
		Operation:  "RESTORE",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata: map[string]interface{}{
			"backup_id": backupID,
		},
	})

	rlm.metrics.OperationsTotal["RESTORE"]++
	return nil
}

// TerminateResource terminates and cleans up a resource
func (rlm *ResourceLifecycleManager) TerminateResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("terminating resource", "resource_id", resourceID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	// Update resource state
	resourceState.LastOperation = "TERMINATE"
	resourceState.OperationStatus = "IN_PROGRESS"
	resourceState.CurrentStatus = "TERMINATING"
	resourceState.DesiredStatus = "TERMINATED"
	resourceState.LastUpdated = time.Now()
	rlm.updateResourceState(resourceState)

	// Queue terminate operation
	operation := &ResourceOperation{
		OperationID:   rlm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "TERMINATE",
		Parameters:    make(map[string]interface{}),
		Priority:      1,
		Timeout:       rlm.policies.TimeoutPolicy.OperationTimeouts["TERMINATE"],
		CreatedAt:     time.Now(),
		Status:        "QUEUED",
	}

	rlm.queueOperation(operation)

	// Emit terminate started event
	rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  "TERMINATE_STARTED",
		ResourceID: resourceID,
		Operation:  "TERMINATE",
		Status:     "IN_PROGRESS",
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
	})

	rlm.metrics.OperationsTotal["TERMINATE"]++
	return nil
}

// State Management and Discovery

// DiscoverResources discovers resources from a provider
func (rlm *ResourceLifecycleManager) DiscoverResources(ctx context.Context, providerID string) ([]*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("discovering resources", "provider_id", providerID)

	provider, err := rlm.providerRegistry.GetProvider(providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", providerID, err)
	}

	// Get resources from provider
	providerResources, err := provider.ListResources(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources from provider: %w", err)
	}

	var resources []*models.Resource
	for _, providerResource := range providerResources {
		// Convert provider resource to O2 IMS resource model
		resource := rlm.convertProviderResource(providerResource, providerID)
		resources = append(resources, resource)

		// Update resource state if we're tracking it
		if resourceState, exists := rlm.getResourceStateIfExists(resource.ResourceID); exists {
			resourceState.CurrentStatus = resource.Status
			resourceState.LastUpdated = time.Now()
			rlm.updateResourceState(resourceState)
		}
	}

	rlm.logger.Info("resource discovery completed",
		"provider_id", providerID,
		"resources_discovered", len(resources))

	return resources, nil
}

// SyncResourceState synchronizes resource state with provider
func (rlm *ResourceLifecycleManager) SyncResourceState(ctx context.Context, resourceID string) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("syncing resource state", "resource_id", resourceID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Get current state from provider
	providerResource, err := provider.GetResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from provider: %w", err)
	}

	// Update resource state
	resourceState.CurrentStatus = providerResource.Status
	resourceState.HealthStatus = providerResource.Health
	resourceState.LastUpdated = time.Now()
	if providerResource.Status != resourceState.DesiredStatus {
		logger.Info("resource state drift detected",
			"current_status", providerResource.Status,
			"desired_status", resourceState.DesiredStatus)
	}

	rlm.updateResourceState(resourceState)

	// Convert to O2 IMS resource model
	resource := rlm.convertProviderResource(providerResource, resourceState.Provider)
	return resource, nil
}

// ValidateResourceConfiguration validates resource configuration
func (rlm *ResourceLifecycleManager) ValidateResourceConfiguration(ctx context.Context, resourceID string, config *runtime.RawExtension) error {
	logger := log.FromContext(ctx)
	logger.Info("validating resource configuration", "resource_id", resourceID)

	// Get current resource state
	resourceState, err := rlm.getResourceState(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Validate configuration with provider
	// This would typically call a provider-specific validation method
	// For now, implement basic validation

	if config == nil || len(config.Raw) == 0 {
		return fmt.Errorf("configuration cannot be empty")
	}

	// Additional validation logic would go here
	logger.Info("resource configuration validation completed", "resource_id", resourceID)
	return nil
}

// Shutdown gracefully shuts down the resource lifecycle manager
func (rlm *ResourceLifecycleManager) Shutdown(ctx context.Context) error {
	rlm.logger.Info("shutting down resource lifecycle manager")

	// Cancel context to stop background services
	rlm.cancel()

	// Stop operation workers
	close(rlm.stopCh)

	// Wait for pending operations to complete (with timeout)
	shutdownTimeout := 30 * time.Second
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	// Monitor operation queue until empty or timeout
	for {
		select {
		case <-shutdownCtx.Done():
			rlm.logger.Warn("shutdown timeout reached with pending operations")
			return nil
		default:
			if len(rlm.operationQueue) == 0 {
				rlm.logger.Info("all operations completed, shutdown successful")
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

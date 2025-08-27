// Package o2 implements helper methods and background services for resource lifecycle management
package o2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// Helper methods for resource lifecycle operations

// generateResourceID generates a unique resource ID
func (rlm *ResourceLifecycleManager) generateResourceID(provider, resourceType, name string) string {
	// Generate a resource ID in the format: provider-resourcetype-name-uuid
	shortUUID := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s-%s-%s", provider, resourceType, name, shortUUID)
}

// generateOperationID generates a unique operation ID
func (rlm *ResourceLifecycleManager) generateOperationID() string {
	return fmt.Sprintf("op-%s", uuid.New().String())
}

// generateEventID generates a unique event ID
func (rlm *ResourceLifecycleManager) generateEventID() string {
	return fmt.Sprintf("evt-%s", uuid.New().String())
}

// generateBackupID generates a unique backup ID
func (rlm *ResourceLifecycleManager) generateBackupID(resourceID string) string {
	timestamp := time.Now().Format("20060102-150405")
	shortUUID := uuid.New().String()[:8]
	return fmt.Sprintf("backup-%s-%s-%s", resourceID, timestamp, shortUUID)
}

// State management methods

// updateResourceState updates the resource state in memory
func (rlm *ResourceLifecycleManager) updateResourceState(state *ResourceState) {
	rlm.stateMutex.Lock()
	defer rlm.stateMutex.Unlock()

	rlm.resourceStates[state.ResourceID] = state
	rlm.updateMetricsForResourceState(state)
}

// getResourceState retrieves the current state of a resource
func (rlm *ResourceLifecycleManager) getResourceState(resourceID string) (*ResourceState, error) {
	rlm.stateMutex.RLock()
	defer rlm.stateMutex.RUnlock()

	state, exists := rlm.resourceStates[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource state not found for resource %s", resourceID)
	}

	return state, nil
}

// getResourceStateIfExists retrieves resource state if it exists
func (rlm *ResourceLifecycleManager) getResourceStateIfExists(resourceID string) (*ResourceState, bool) {
	rlm.stateMutex.RLock()
	defer rlm.stateMutex.RUnlock()

	state, exists := rlm.resourceStates[resourceID]
	return state, exists
}

// removeResourceState removes a resource state from memory
func (rlm *ResourceLifecycleManager) removeResourceState(resourceID string) {
	rlm.stateMutex.Lock()
	defer rlm.stateMutex.Unlock()

	delete(rlm.resourceStates, resourceID)
}

// Operation queue management

// queueOperation adds an operation to the queue
func (rlm *ResourceLifecycleManager) queueOperation(operation *ResourceOperation) {
	select {
	case rlm.operationQueue <- operation:
		rlm.logger.Debug("operation queued",
			"operation_id", operation.OperationID,
			"resource_id", operation.ResourceID,
			"operation_type", operation.OperationType)
	default:
		rlm.logger.Error("operation queue is full, dropping operation",
			"operation_id", operation.OperationID,
			"resource_id", operation.ResourceID,
			"operation_type", operation.OperationType)
		rlm.metrics.OperationsFailure[operation.OperationType]++
	}
}

// startOperationWorkers starts background workers to process operations
func (rlm *ResourceLifecycleManager) startOperationWorkers() {
	for i := 0; i < rlm.workers; i++ {
		go rlm.operationWorker(i)
	}
}

// operationWorker processes operations from the queue
func (rlm *ResourceLifecycleManager) operationWorker(workerID int) {
	rlm.logger.Info("starting operation worker", "worker_id", workerID)

	for {
		select {
		case <-rlm.stopCh:
			rlm.logger.Info("stopping operation worker", "worker_id", workerID)
			return
		case operation := <-rlm.operationQueue:
			rlm.processOperation(context.Background(), operation)
		}
	}
}

// processOperation processes a single operation
func (rlm *ResourceLifecycleManager) processOperation(ctx context.Context, operation *ResourceOperation) {
	startTime := time.Now()
	operationCtx, cancel := context.WithTimeout(ctx, operation.Timeout)
	defer cancel()

	logger := log.FromContext(operationCtx).WithValues(
		"operation_id", operation.OperationID,
		"resource_id", operation.ResourceID,
		"operation_type", operation.OperationType,
	)

	logger.Info("processing operation")

	// Update operation status
	operation.Status = OperationStatusInProgress
	operation.StartedAt = &startTime

	// Process based on operation type
	var result *OperationResult
	var err error

	switch operation.OperationType {
	case OperationTypeProvision:
		result, err = rlm.processProvisionOperation(operationCtx, operation)
	case OperationTypeConfigure:
		result, err = rlm.processConfigureOperation(operationCtx, operation)
	case OperationTypeScale:
		result, err = rlm.processScaleOperation(operationCtx, operation)
	case OperationTypeMigrate:
		result, err = rlm.processMigrateOperation(operationCtx, operation)
	case OperationTypeBackup:
		result, err = rlm.processBackupOperation(operationCtx, operation)
	case OperationTypeRestore:
		result, err = rlm.processRestoreOperation(operationCtx, operation)
	case OperationTypeTerminate:
		result, err = rlm.processTerminateOperation(operationCtx, operation)
	default:
		err = fmt.Errorf("unknown operation type: %s", operation.OperationType)
	}

	// Update operation completion
	completedAt := time.Now()
	operation.CompletedAt = &completedAt
	duration := completedAt.Sub(startTime)

	if err != nil {
		logger.Error("operation failed", "error", err, "duration", duration)
		operation.Status = OperationStatusFailed
		operation.ErrorMessage = err.Error()
		rlm.metrics.OperationsFailure[operation.OperationType]++

		// Handle retry logic
		rlm.handleOperationFailure(operation, err)
	} else {
		logger.Info("operation completed successfully", "duration", duration)
		operation.Status = OperationStatusCompleted
		rlm.metrics.OperationsSuccess[operation.OperationType]++

		// Update resource state on success
		rlm.updateResourceStateOnSuccess(operation, result)
	}

	// Update metrics
	rlm.metrics.OperationsDuration[operation.OperationType] = duration.Seconds()

	// Call callback if provided
	if operation.Callback != nil {
		operation.Callback(operationCtx, operation, result)
	}

	// Emit completion event
	rlm.emitOperationCompletionEvent(operationCtx, operation, result, err)
}

// Operation processors

// processProvisionOperation processes a provision operation
func (rlm *ResourceLifecycleManager) processProvisionOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	req, ok := operation.Parameters["request"].(*ProvisionResourceRequest)
	if !ok {
		return nil, fmt.Errorf("invalid provision request parameters")
	}

	provider, err := rlm.providerRegistry.GetProvider(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", req.Provider, err)
	}

	// Create provider-specific resource request
	providerReq := &providers.CreateResourceRequest{
		Name:          req.Name,
		Type:          req.ResourceType,
		Specification: rlm.convertToProviderSpec(req.Configuration),
		Labels:        req.Labels,
		Annotations:   req.Annotations,
		Timeout:       operation.Timeout,
	}

	// Provision resource with provider
	providerResource, err := provider.CreateResource(ctx, providerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to provision resource with provider: %w", err)
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result:     providerResource,
		Duration:   time.Since(*operation.StartedAt),
		Timestamp:  time.Now(),
	}, nil
}

// processConfigureOperation processes a configure operation
func (rlm *ResourceLifecycleManager) processConfigureOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Update resource configuration with provider
	updateReq := &providers.UpdateResourceRequest{
		Specification: rlm.convertToProviderSpec(resourceState.Configuration),
		Timeout:       operation.Timeout,
	}

	providerResource, err := provider.UpdateResource(ctx, operation.ResourceID, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to configure resource with provider: %w", err)
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result:     providerResource,
		Duration:   time.Since(*operation.StartedAt),
		Timestamp:  time.Now(),
	}, nil
}

// processScaleOperation processes a scale operation
func (rlm *ResourceLifecycleManager) processScaleOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Extract scaling parameters
	scaleType := operation.Parameters["scale_type"].(string)
	targetReplicas := operation.Parameters["target_replicas"].(*int32)

	scaleReq := &providers.ScaleRequest{
		Type:    scaleType,
		Amount:  *targetReplicas,
		Timeout: operation.Timeout,
	}

	err = provider.ScaleResource(ctx, operation.ResourceID, scaleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to scale resource with provider: %w", err)
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result:     map[string]interface{}{"target_replicas": *targetReplicas},
		Duration:   time.Since(*operation.StartedAt),
		Timestamp:  time.Now(),
	}, nil
}

// processMigrateOperation processes a migrate operation
func (rlm *ResourceLifecycleManager) processMigrateOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	// Migration is a complex operation involving multiple providers
	sourceProvider := operation.Parameters["source_provider"].(string)
	targetProvider := operation.Parameters["target_provider"].(string)

	// Get both providers
	srcProvider, err := rlm.providerRegistry.GetProvider(sourceProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to get source provider %s: %w", sourceProvider, err)
	}

	tgtProvider, err := rlm.providerRegistry.GetProvider(targetProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to get target provider %s: %w", targetProvider, err)
	}

	// Get current resource from source provider
	sourceResource, err := srcProvider.GetResource(ctx, operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from source provider: %w", err)
	}

	// Create resource on target provider
	createReq := &providers.CreateResourceRequest{
		Name:          sourceResource.Name,
		Type:          sourceResource.Type,
		Specification: sourceResource.Specification,
		Labels:        sourceResource.Labels,
		Annotations:   sourceResource.Annotations,
		Timeout:       operation.Timeout,
	}

	targetResource, err := tgtProvider.CreateResource(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource on target provider: %w", err)
	}

	// Delete resource from source provider (if migration is successful)
	err = srcProvider.DeleteResource(ctx, operation.ResourceID)
	if err != nil {
		rlm.logger.Error("failed to delete resource from source provider after migration",
			"resource_id", operation.ResourceID,
			"source_provider", sourceProvider,
			"error", err)
		// Continue - migration was successful even if cleanup failed
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result:     targetResource,
		Metadata: map[string]interface{}{
			"source_provider": sourceProvider,
			"target_provider": targetProvider,
		},
		Duration:  time.Since(*operation.StartedAt),
		Timestamp: time.Now(),
	}, nil
}

// processBackupOperation processes a backup operation
func (rlm *ResourceLifecycleManager) processBackupOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Get current resource state for backup
	providerResource, err := provider.GetResource(ctx, operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource from provider: %w", err)
	}

	backupID := operation.Parameters["backup_id"].(string)
	backupType := operation.Parameters["backup_type"].(string)

	// Create backup data structure
	backupData := map[string]interface{}{
		"resource_id":   operation.ResourceID,
		"resource_type": providerResource.Type,
		"specification": providerResource.Specification,
		"labels":        providerResource.Labels,
		"annotations":   providerResource.Annotations,
		"status":        providerResource.Status,
		"backup_type":   backupType,
		"backup_time":   time.Now(),
		"provider":      resourceState.Provider,
	}

	// Store the backup data to persistent storage
	if err := rlm.storeBackupData(ctx, backupID, backupData); err != nil {
		return nil, fmt.Errorf("failed to store backup data: %w", err)
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result: &BackupInfo{
			BackupID:    backupID,
			ResourceID:  operation.ResourceID,
			BackupType:  backupType,
			Status:      "COMPLETED",
			CreatedAt:   *operation.StartedAt,
			CompletedAt: &[]time.Time{time.Now()}[0],
		},
		Duration:  time.Since(*operation.StartedAt),
		Timestamp: time.Now(),
	}, nil
}

// processRestoreOperation processes a restore operation
func (rlm *ResourceLifecycleManager) processRestoreOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	backupID := operation.Parameters["backup_id"].(string)

	// In a real implementation, this would retrieve backup data from storage
	// For now, we simulate restore completion

	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Simulate restoration by updating resource configuration
	updateReq := &providers.UpdateResourceRequest{
		Specification: resourceState.Configuration.Object,
		Timeout:       operation.Timeout,
	}

	providerResource, err := provider.UpdateResource(ctx, operation.ResourceID, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to restore resource with provider: %w", err)
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Result:     providerResource,
		Metadata: map[string]interface{}{
			"backup_id": backupID,
		},
		Duration:  time.Since(*operation.StartedAt),
		Timestamp: time.Now(),
	}, nil
}

// processTerminateOperation processes a terminate operation
func (rlm *ResourceLifecycleManager) processTerminateOperation(ctx context.Context, operation *ResourceOperation) (*OperationResult, error) {
	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource state: %w", err)
	}

	provider, err := rlm.providerRegistry.GetProvider(resourceState.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Delete resource from provider
	err = provider.DeleteResource(ctx, operation.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to terminate resource with provider: %w", err)
	}

	// Remove resource state from memory
	rlm.removeResourceState(operation.ResourceID)

	// Delete from persistent storage
	if rlm.storage != nil {
		if deleteErr := rlm.storage.DeleteResource(ctx, operation.ResourceID); deleteErr != nil {
			rlm.logger.Error("failed to delete resource from persistent storage",
				"resource_id", operation.ResourceID,
				"error", deleteErr)
			// Continue - the resource was successfully terminated from the provider
		}
	}

	return &OperationResult{
		Success:    true,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Duration:   time.Since(*operation.StartedAt),
		Timestamp:  time.Now(),
	}, nil
}

// Background services

// startStateReconciler starts the state reconciler background service
func (rlm *ResourceLifecycleManager) startStateReconciler() {
	ticker := time.NewTicker(rlm.config.ResourceConfig.StateReconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rlm.ctx.Done():
			return
		case <-ticker.C:
			rlm.reconcileResourceStates(context.Background())
		}
	}
}

// reconcileResourceStates reconciles resource states with providers
func (rlm *ResourceLifecycleManager) reconcileResourceStates(ctx context.Context) {
	rlm.stateMutex.RLock()
	resourceStates := make([]*ResourceState, 0, len(rlm.resourceStates))
	for _, state := range rlm.resourceStates {
		resourceStates = append(resourceStates, state)
	}
	rlm.stateMutex.RUnlock()

	for _, state := range resourceStates {
		// Skip resources that are currently being operated on
		if state.OperationStatus == OperationStatusInProgress {
			continue
		}

		// Sync state with provider
		_, err := rlm.SyncResourceState(ctx, state.ResourceID)
		if err != nil {
			rlm.logger.Error("failed to sync resource state during reconciliation",
				"resource_id", state.ResourceID,
				"error", err)
		}
	}
}

// startMetricsCollection starts metrics collection background service
func (rlm *ResourceLifecycleManager) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second) // Collect metrics every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-rlm.ctx.Done():
			return
		case <-ticker.C:
			rlm.collectMetrics()
		}
	}
}

// collectMetrics collects and updates lifecycle metrics
func (rlm *ResourceLifecycleManager) collectMetrics() {
	rlm.stateMutex.RLock()
	defer rlm.stateMutex.RUnlock()

	// Reset status counters
	rlm.metrics.ResourcesByStatus = make(map[string]int64)
	rlm.metrics.ResourcesByProvider = make(map[string]int64)

	// Count resources by status and provider
	for _, state := range rlm.resourceStates {
		rlm.metrics.ResourcesByStatus[state.CurrentStatus]++
		rlm.metrics.ResourcesByProvider[state.Provider]++
	}

	rlm.metrics.LastUpdated = time.Now()
}

// startHealthChecker starts health checking background service
func (rlm *ResourceLifecycleManager) startHealthChecker() {
	if rlm.policies.MonitoringPolicy == nil || !rlm.policies.MonitoringPolicy.AlarmingEnabled {
		return
	}

	ticker := time.NewTicker(rlm.policies.MonitoringPolicy.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rlm.ctx.Done():
			return
		case <-ticker.C:
			rlm.performHealthChecks(context.Background())
		}
	}
}

// performHealthChecks performs health checks on all managed resources
func (rlm *ResourceLifecycleManager) performHealthChecks(ctx context.Context) {
	rlm.stateMutex.RLock()
	resourceStates := make([]*ResourceState, 0, len(rlm.resourceStates))
	for _, state := range rlm.resourceStates {
		resourceStates = append(resourceStates, state)
	}
	rlm.stateMutex.RUnlock()

	for _, state := range resourceStates {
		// Skip terminated resources
		if state.CurrentStatus == ResourceStatusTerminated {
			continue
		}

		provider, err := rlm.providerRegistry.GetProvider(state.Provider)
		if err != nil {
			rlm.logger.Error("failed to get provider for health check",
				"resource_id", state.ResourceID,
				"provider", state.Provider,
				"error", err)
			continue
		}

		// Get resource health from provider
		health, err := provider.GetResourceHealth(ctx, state.ResourceID)
		if err != nil {
			rlm.logger.Error("failed to get resource health",
				"resource_id", state.ResourceID,
				"error", err)

			// Update resource state to reflect health check failure
			state.HealthStatus = HealthStatusUnknown
			state.ErrorMessage = err.Error()
			state.LastUpdated = time.Now()
			rlm.updateResourceState(state)
			continue
		}

		// Update resource health status
		oldHealthStatus := state.HealthStatus
		state.HealthStatus = health.Status
		state.LastUpdated = time.Now()

		// Clear error if health is good
		if health.Status == HealthStatusHealthy {
			state.ErrorMessage = ""
		}

		rlm.updateResourceState(state)

		// Emit health change event if status changed
		if oldHealthStatus != health.Status {
			rlm.emitResourceEvent(ctx, &ResourceLifecycleEvent{
				EventID:    rlm.generateEventID(),
				EventType:  "HEALTH_STATUS_CHANGED",
				ResourceID: state.ResourceID,
				Operation:  "HEALTH_CHECK",
				Status:     health.Status,
				Timestamp:  time.Now(),
				Source:     "health_checker",
				Metadata: map[string]interface{}{
					"old_status": oldHealthStatus,
					"new_status": health.Status,
				},
			})
		}
	}
}

// Utility methods

// convertToProviderSpec converts O2 configuration to provider-specific specification
func (rlm *ResourceLifecycleManager) convertToProviderSpec(config *runtime.RawExtension) map[string]interface{} {
	if config == nil {
		return make(map[string]interface{})
	}

	// In a real implementation, this would parse the configuration
	// and convert it to the provider's expected format
	return map[string]interface{}{
		"raw_config": string(config.Raw),
	}
}

// convertProviderResource converts provider resource to O2 IMS resource model
func (rlm *ResourceLifecycleManager) convertProviderResource(providerResource *providers.ResourceResponse, providerID string) *models.Resource {
	return &models.Resource{
		ResourceID:     providerResource.ID,
		Name:           providerResource.Name,
		ResourceTypeID: providerResource.Type,
		Status:         providerResource.Status,
		GlobalAssetID:  providerResource.ID,
		Provider:       providerID,
		CreatedAt:      providerResource.CreatedAt,
		UpdatedAt:      providerResource.UpdatedAt,
	}
}

// Validation methods

// validateProvisionRequest validates a provision request
func (rlm *ResourceLifecycleManager) validateProvisionRequest(req *ProvisionResourceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("resource name is required")
	}
	if req.ResourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if req.Configuration == nil {
		return fmt.Errorf("resource configuration is required")
	}
	return nil
}

// checkResourceQuotas checks if the provision request exceeds resource quotas
func (rlm *ResourceLifecycleManager) checkResourceQuotas(ctx context.Context, req *ProvisionResourceRequest) error {
	if rlm.policies.ResourceQuotaPolicy == nil {
		return nil // No quota policy configured
	}

	quota := rlm.policies.ResourceQuotaPolicy

	// Check total resources limit
	rlm.stateMutex.RLock()
	totalResources := len(rlm.resourceStates)
	resourcesForProvider := 0
	resourcesForType := 0

	for _, state := range rlm.resourceStates {
		if state.Provider == req.Provider {
			resourcesForProvider++
		}
		if state.ResourceType == req.ResourceType {
			resourcesForType++
		}
	}
	rlm.stateMutex.RUnlock()

	if quota.MaxTotalResources > 0 && totalResources >= quota.MaxTotalResources {
		return fmt.Errorf("maximum total resources limit exceeded: %d", quota.MaxTotalResources)
	}

	if maxForProvider, exists := quota.MaxResourcesPerProvider[req.Provider]; exists && resourcesForProvider >= maxForProvider {
		return fmt.Errorf("maximum resources limit for provider %s exceeded: %d", req.Provider, maxForProvider)
	}

	if maxForType, exists := quota.MaxResourcesPerType[req.ResourceType]; exists && resourcesForType >= maxForType {
		return fmt.Errorf("maximum resources limit for type %s exceeded: %d", req.ResourceType, maxForType)
	}

	return nil
}

// canConfigure checks if a resource can be configured in its current state
func (rlm *ResourceLifecycleManager) canConfigure(state *ResourceState) bool {
	configurableStates := []string{
		ResourceStatusActive,
		ResourceStatusScaling,
	}

	for _, configurableState := range configurableStates {
		if state.CurrentStatus == configurableState {
			return true
		}
	}

	return false
}

// validateScaleRequest validates a scale request
func (rlm *ResourceLifecycleManager) validateScaleRequest(state *ResourceState, req *ScaleResourceRequest) error {
	if req.ScaleType != ScaleTypeHorizontal && req.ScaleType != ScaleTypeVertical {
		return fmt.Errorf("invalid scale type: %s", req.ScaleType)
	}

	if req.ScaleType == ScaleTypeHorizontal && req.TargetReplicas == nil {
		return fmt.Errorf("target replicas required for horizontal scaling")
	}

	if req.ScaleType == ScaleTypeVertical && len(req.TargetResources) == 0 {
		return fmt.Errorf("target resources required for vertical scaling")
	}

	// Check scaling policy limits
	if rlm.policies.ScalingPolicy != nil {
		policy := rlm.policies.ScalingPolicy
		if req.TargetReplicas != nil {
			if *req.TargetReplicas < policy.MinReplicas {
				return fmt.Errorf("target replicas %d below minimum %d", *req.TargetReplicas, policy.MinReplicas)
			}
			if *req.TargetReplicas > policy.MaxReplicas {
				return fmt.Errorf("target replicas %d above maximum %d", *req.TargetReplicas, policy.MaxReplicas)
			}
		}
	}

	return nil
}

// validateMigrationRequest validates a migration request
func (rlm *ResourceLifecycleManager) validateMigrationRequest(state *ResourceState, req *MigrateResourceRequest) error {
	if req.SourceProvider == "" {
		return fmt.Errorf("source provider is required")
	}
	if req.TargetProvider == "" {
		return fmt.Errorf("target provider is required")
	}
	if req.SourceProvider == req.TargetProvider {
		return fmt.Errorf("source and target providers cannot be the same")
	}

	// Verify source provider matches current resource provider
	if state.Provider != req.SourceProvider {
		return fmt.Errorf("source provider %s does not match resource provider %s", req.SourceProvider, state.Provider)
	}

	return nil
}

// Event and notification handling

// emitResourceEvent emits a resource lifecycle event
func (rlm *ResourceLifecycleManager) emitResourceEvent(ctx context.Context, event *ResourceLifecycleEvent) {
	rlm.eventBus.PublishEvent(ctx, event)
}

// emitOperationCompletionEvent emits an operation completion event
func (rlm *ResourceLifecycleManager) emitOperationCompletionEvent(ctx context.Context, operation *ResourceOperation, result *OperationResult, err error) {
	eventType := "OPERATION_COMPLETED"
	status := "SUCCESS"
	errorMsg := ""

	if err != nil {
		eventType = "OPERATION_FAILED"
		status = "FAILED"
		errorMsg = err.Error()
	}

	event := &ResourceLifecycleEvent{
		EventID:    rlm.generateEventID(),
		EventType:  eventType,
		ResourceID: operation.ResourceID,
		Operation:  operation.OperationType,
		Status:     status,
		Timestamp:  time.Now(),
		Source:     "resource_lifecycle_manager",
		Metadata: map[string]interface{}{
			"operation_id": operation.OperationID,
			"duration":     time.Since(*operation.StartedAt).Seconds(),
		},
		Error: errorMsg,
	}

	rlm.emitResourceEvent(ctx, event)
}

// handleOperationFailure handles operation failures and retry logic
func (rlm *ResourceLifecycleManager) handleOperationFailure(operation *ResourceOperation, err error) {
	resourceState, stateErr := rlm.getResourceState(operation.ResourceID)
	if stateErr != nil {
		rlm.logger.Error("failed to get resource state for failure handling",
			"resource_id", operation.ResourceID,
			"error", stateErr)
		return
	}

	resourceState.RetryCount++
	resourceState.ErrorMessage = err.Error()
	resourceState.LastUpdated = time.Now()

	// Check if we should retry
	if rlm.shouldRetry(operation, resourceState, err) {
		// Calculate next retry time with exponential backoff
		backoffDelay := rlm.calculateBackoffDelay(resourceState.RetryCount)
		resourceState.NextRetry = time.Now().Add(backoffDelay)

		rlm.logger.Info("scheduling operation retry",
			"operation_id", operation.OperationID,
			"resource_id", operation.ResourceID,
			"retry_count", resourceState.RetryCount,
			"next_retry", resourceState.NextRetry)

		// Schedule retry
		go func() {
			time.Sleep(backoffDelay)

			// Create retry operation
			retryOperation := &ResourceOperation{
				OperationID:   rlm.generateOperationID(),
				ResourceID:    operation.ResourceID,
				OperationType: operation.OperationType,
				Parameters:    operation.Parameters,
				Priority:      operation.Priority,
				Timeout:       operation.Timeout,
				CreatedAt:     time.Now(),
				Status:        OperationStatusQueued,
			}

			rlm.queueOperation(retryOperation)
		}()
	} else {
		// No more retries, mark operation as permanently failed
		resourceState.OperationStatus = OperationStatusFailed
		resourceState.CurrentStatus = ResourceStatusError

		rlm.logger.Error("operation permanently failed after maximum retries",
			"operation_id", operation.OperationID,
			"resource_id", operation.ResourceID,
			"retry_count", resourceState.RetryCount)
	}

	rlm.updateResourceState(resourceState)
}

// shouldRetry determines if an operation should be retried
func (rlm *ResourceLifecycleManager) shouldRetry(operation *ResourceOperation, resourceState *ResourceState, err error) bool {
	if rlm.policies.RetryPolicy == nil {
		return false
	}

	policy := rlm.policies.RetryPolicy

	// Check maximum retries
	if resourceState.RetryCount >= policy.MaxRetries {
		return false
	}

	// Check if error is retriable
	errorStr := strings.ToLower(err.Error())
	for _, retriableError := range policy.RetriableErrors {
		if strings.Contains(errorStr, strings.ToLower(retriableError)) {
			return true
		}
	}

	// Default retriable errors
	defaultRetriableErrors := []string{
		"timeout",
		"connection",
		"network",
		"temporary",
		"throttl",
		"rate limit",
	}

	for _, retriableError := range defaultRetriableErrors {
		if strings.Contains(errorStr, retriableError) {
			return true
		}
	}

	return false
}

// calculateBackoffDelay calculates the delay for retry with exponential backoff
func (rlm *ResourceLifecycleManager) calculateBackoffDelay(retryCount int) time.Duration {
	if rlm.policies.RetryPolicy == nil {
		return time.Minute // Default 1 minute
	}

	policy := rlm.policies.RetryPolicy
	delay := policy.RetryInterval

	// Apply exponential backoff
	for i := 1; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
	}

	// Cap at maximum delay
	if delay > policy.MaxRetryDelay {
		delay = policy.MaxRetryDelay
	}

	return delay
}

// updateResourceStateOnSuccess updates resource state when an operation succeeds
func (rlm *ResourceLifecycleManager) updateResourceStateOnSuccess(operation *ResourceOperation, result *OperationResult) {
	resourceState, err := rlm.getResourceState(operation.ResourceID)
	if err != nil {
		rlm.logger.Error("failed to get resource state for success update",
			"resource_id", operation.ResourceID,
			"error", err)
		return
	}

	// Update state based on operation type
	switch operation.OperationType {
	case OperationTypeProvision:
		resourceState.CurrentStatus = ResourceStatusActive
		resourceState.DesiredStatus = ResourceStatusActive
	case OperationTypeConfigure:
		// Configuration completed successfully
		resourceState.CurrentStatus = ResourceStatusActive
	case OperationTypeScale:
		// Scaling completed successfully
		resourceState.CurrentStatus = ResourceStatusActive
	case OperationTypeMigrate:
		// Migration completed successfully
		resourceState.CurrentStatus = ResourceStatusActive
		// Update provider to target provider
		if targetProvider, ok := result.Metadata["target_provider"].(string); ok {
			resourceState.Provider = targetProvider
		}
	case OperationTypeRestore:
		// Restoration completed successfully
		resourceState.CurrentStatus = ResourceStatusActive
	case OperationTypeTerminate:
		resourceState.CurrentStatus = ResourceStatusTerminated
		resourceState.DesiredStatus = ResourceStatusTerminated
	}

	// Clear error state and reset retry count
	resourceState.OperationStatus = OperationStatusCompleted
	resourceState.ErrorMessage = ""
	resourceState.RetryCount = 0
	resourceState.NextRetry = time.Time{}
	resourceState.LastUpdated = time.Now()

	rlm.updateResourceState(resourceState)
}

// updateMetricsForResourceState updates metrics when resource state changes
func (rlm *ResourceLifecycleManager) updateMetricsForResourceState(state *ResourceState) {
	// Update resource counts by status and provider
	rlm.metrics.ResourcesByStatus[state.CurrentStatus]++
	rlm.metrics.ResourcesByProvider[state.Provider]++
}

// Factory functions for creating default instances

// newResourceEventBus creates a new resource event bus
func newResourceEventBus(logger *logging.StructuredLogger) *ResourceEventBus {
	return &ResourceEventBus{
		subscribers: make(map[string][]EventSubscriber),
		logger:      logger,
	}
}

// PublishEvent publishes an event to subscribers
func (reb *ResourceEventBus) PublishEvent(ctx context.Context, event *ResourceLifecycleEvent) {
	reb.mutex.RLock()
	subscribers := reb.subscribers[event.EventType]
	reb.mutex.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub EventSubscriber) {
			defer func() {
				if r := recover(); r != nil {
					reb.logger.Error("event subscriber panic",
						"event_type", event.EventType,
						"event_id", event.EventID,
						"panic", r)
				}
			}()
			sub(ctx, event)
		}(subscriber)
	}
}

// Subscribe subscribes to events of a specific type
func (reb *ResourceEventBus) Subscribe(eventType string, subscriber EventSubscriber) {
	reb.mutex.Lock()
	defer reb.mutex.Unlock()

	reb.subscribers[eventType] = append(reb.subscribers[eventType], subscriber)
}

// newResourceLifecycleMetrics creates a new metrics instance
func newResourceLifecycleMetrics() *ResourceLifecycleMetrics {
	return &ResourceLifecycleMetrics{
		OperationsTotal:     make(map[string]int64),
		OperationsSuccess:   make(map[string]int64),
		OperationsFailure:   make(map[string]int64),
		OperationsDuration:  make(map[string]float64),
		ResourcesByStatus:   make(map[string]int64),
		ResourcesByProvider: make(map[string]int64),
		LastUpdated:         time.Now(),
	}
}

// newDefaultResourcePolicies creates default resource policies
func newDefaultResourcePolicies() *ResourcePolicies {
	return &ResourcePolicies{
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			RetryInterval:   30 * time.Second,
			BackoffFactor:   2.0,
			MaxRetryDelay:   5 * time.Minute,
			RetriableErrors: []string{"timeout", "connection", "network", "temporary"},
		},
		TimeoutPolicy: &TimeoutPolicy{
			DefaultTimeout: 10 * time.Minute,
			OperationTimeouts: map[string]time.Duration{
				"PROVISION": 15 * time.Minute,
				"CONFIGURE": 5 * time.Minute,
				"SCALE":     10 * time.Minute,
				"MIGRATE":   30 * time.Minute,
				"BACKUP":    20 * time.Minute,
				"RESTORE":   25 * time.Minute,
				"TERMINATE": 10 * time.Minute,
			},
		},
		ScalingPolicy: &ScalingPolicy{
			AutoScalingEnabled:  false,
			ScaleUpThreshold:    80.0,
			ScaleDownThreshold:  30.0,
			ScaleCooldownPeriod: 5 * time.Minute,
			MinReplicas:         1,
			MaxReplicas:         10,
			ScalingMetrics:      []string{"cpu", "memory"},
		},
		BackupPolicy: &BackupPolicy{
			AutoBackupEnabled:     false,
			BackupInterval:        24 * time.Hour,
			RetentionPeriod:       30 * 24 * time.Hour,
			BackupStorageLocation: "/backups",
			BackupEncryption:      true,
		},
		MonitoringPolicy: &MonitoringPolicy{
			HealthCheckInterval:       30 * time.Second,
			MetricsCollectionInterval: 60 * time.Second,
			AlarmingEnabled:           true,
			AlarmThresholds: map[string]float64{
				"cpu":    80.0,
				"memory": 85.0,
				"disk":   90.0,
			},
		},
		CompliancePolicy: &CompliancePolicy{
			RequiredLabels: map[string]string{
				"managed-by": "nephoran-o2-ims",
			},
			SecurityPolicies:   []string{"default-security-policy"},
			DataClassification: "internal",
		},
		ResourceQuotaPolicy: &ResourceQuotaPolicy{
			MaxTotalResources: 1000,
			MaxResourcesPerProvider: map[string]int{
				"kubernetes": 500,
				"openstack":  300,
				"aws":        400,
			},
			MaxResourcesPerType: map[string]int{
				"compute": 200,
				"storage": 100,
				"network": 150,
			},
		},
	}
}

// storeBackupData stores backup data to persistent storage
func (rlm *ResourceLifecycleManager) storeBackupData(ctx context.Context, backupID string, backupData map[string]interface{}) error {
	// In a real implementation, this would store to a persistent backend
	// For now, we'll store in memory or log the operation
	rlm.logger.Info("Storing backup data",
		"backup_id", backupID,
		"resource_id", backupData["resource_id"],
		"resource_type", backupData["resource_type"],
		"backup_type", backupData["backup_type"],
		"backup_time", backupData["backup_time"])

	// Simulate storage operation - in production this would be:
	// - Stored in object storage (S3, GCS, etc.)
	// - Persisted to database
	// - Written to backup service API
	return nil
}

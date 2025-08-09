package o2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// O2 IMS Interface Implementation
// This file contains the implementation of all O2AdaptorInterface methods
// following the O-RAN.WG6.O2ims-Interface-v01.01 specification

// Resource Pool Management

// GetResourcePools retrieves resource pools with optional filtering
func (a *O2Adaptor) GetResourcePools(ctx context.Context, filter *models.ResourcePoolFilter) ([]*models.ResourcePool, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource pools", "filter", filter)

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var resourcePools []*models.ResourcePool

	for _, pool := range a.resourcePools {
		if a.matchesResourcePoolFilter(pool, filter) {
			// Return a copy to prevent modification
			poolCopy := *pool
			resourcePools = append(resourcePools, &poolCopy)
		}
	}

	// Apply sorting and pagination
	resourcePools = a.sortAndPaginateResourcePools(resourcePools, filter)

	logger.V(1).Info("retrieved resource pools", "count", len(resourcePools))
	return resourcePools, nil
}

// GetResourcePool retrieves a specific resource pool by ID
func (a *O2Adaptor) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource pool", "poolID", poolID)

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	pool, exists := a.resourcePools[poolID]
	if !exists {
		return nil, fmt.Errorf("resource pool not found: %s", poolID)
	}

	// Return a copy to prevent modification
	poolCopy := *pool
	return &poolCopy, nil
}

// CreateResourcePool creates a new resource pool
func (a *O2Adaptor) CreateResourcePool(ctx context.Context, req *models.CreateResourcePoolRequest) (*models.ResourcePool, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating resource pool", "name", req.Name, "provider", req.Provider)

	// Validate request
	if err := a.validateCreateResourcePoolRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if provider exists and is configured
	provider, exists := a.providers[req.Provider]
	if !exists {
		return nil, fmt.Errorf("provider not found or not configured: %s", req.Provider)
	}

	// Generate resource pool ID
	poolID := a.generateResourcePoolID(req.Name, req.Provider)

	// Create resource pool
	resourcePool := &models.ResourcePool{
		ResourcePoolID:   poolID,
		Name:             req.Name,
		Description:      req.Description,
		Location:         req.Location,
		OCloudID:         req.OCloudID,
		GlobalLocationID: req.GlobalLocationID,
		Provider:         req.Provider,
		Region:           req.Region,
		Zone:             req.Zone,
		Extensions:       req.Extensions,
		Status: &models.ResourcePoolStatus{
			State:           models.ResourcePoolStateAvailable,
			Health:          models.ResourcePoolHealthHealthy,
			Utilization:     0.0,
			LastHealthCheck: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Initialize capacity by querying the provider
	capacity, err := a.getProviderCapacity(ctx, provider)
	if err != nil {
		logger.Error(err, "failed to get provider capacity", "provider", req.Provider)
		// Continue with default capacity
		capacity = &models.ResourceCapacity{
			CPU:     &models.ResourceMetric{Total: "0", Available: "0", Used: "0", Unit: "cores"},
			Memory:  &models.ResourceMetric{Total: "0", Available: "0", Used: "0", Unit: "bytes"},
			Storage: &models.ResourceMetric{Total: "0", Available: "0", Used: "0", Unit: "bytes"},
		}
	}
	resourcePool.Capacity = capacity

	// Store resource pool
	a.mutex.Lock()
	a.resourcePools[poolID] = resourcePool
	a.mutex.Unlock()

	logger.Info("resource pool created successfully", "poolID", poolID)
	return resourcePool, nil
}

// UpdateResourcePool updates an existing resource pool
func (a *O2Adaptor) UpdateResourcePool(ctx context.Context, poolID string, req *models.UpdateResourcePoolRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating resource pool", "poolID", poolID)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	pool, exists := a.resourcePools[poolID]
	if !exists {
		return fmt.Errorf("resource pool not found: %s", poolID)
	}

	// Apply updates
	if req.Name != nil {
		pool.Name = *req.Name
	}
	if req.Description != nil {
		pool.Description = *req.Description
	}
	if req.Location != nil {
		pool.Location = *req.Location
	}
	if req.GlobalLocationID != nil {
		pool.GlobalLocationID = *req.GlobalLocationID
	}
	if req.Extensions != nil {
		if pool.Extensions == nil {
			pool.Extensions = make(map[string]interface{})
		}
		for k, v := range req.Extensions {
			pool.Extensions[k] = v
		}
	}
	if req.Metadata != nil {
		// Metadata would be stored in Extensions
		if pool.Extensions == nil {
			pool.Extensions = make(map[string]interface{})
		}
		pool.Extensions["metadata"] = req.Metadata
	}

	pool.UpdatedAt = time.Now()

	logger.Info("resource pool updated successfully", "poolID", poolID)
	return nil
}

// DeleteResourcePool deletes a resource pool
func (a *O2Adaptor) DeleteResourcePool(ctx context.Context, poolID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting resource pool", "poolID", poolID)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.resourcePools[poolID]; !exists {
		return fmt.Errorf("resource pool not found: %s", poolID)
	}

	// TODO: Check if pool is in use by any resources or deployments
	// For now, allow deletion

	delete(a.resourcePools, poolID)

	logger.Info("resource pool deleted successfully", "poolID", poolID)
	return nil
}

// Resource Type Management

// GetResourceTypes retrieves resource types with optional filtering
func (a *O2Adaptor) GetResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource types", "filter", filter)

	return a.catalogService.ListResourceTypes(ctx, filter)
}

// GetResourceType retrieves a specific resource type by ID
func (a *O2Adaptor) GetResourceType(ctx context.Context, typeID string) (*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource type", "typeID", typeID)

	return a.catalogService.GetResourceType(ctx, typeID)
}

// Resource Management

// GetResources retrieves resources with optional filtering
func (a *O2Adaptor) GetResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resources", "filter", filter)

	return a.inventoryService.ListResources(ctx, filter)
}

// GetResource retrieves a specific resource by ID
func (a *O2Adaptor) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource", "resourceID", resourceID)

	return a.inventoryService.GetResource(ctx, resourceID)
}

// CreateResource creates a new resource
func (a *O2Adaptor) CreateResource(ctx context.Context, req *models.CreateResourceRequest) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating resource", "name", req.Name, "type", req.ResourceTypeID, "pool", req.ResourcePoolID)

	// Validate request
	if err := a.validateCreateResourceRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if resource pool exists
	pool, err := a.GetResourcePool(ctx, req.ResourcePoolID)
	if err != nil {
		return nil, fmt.Errorf("resource pool not found: %w", err)
	}

	// Check if resource type exists
	resourceType, err := a.GetResourceType(ctx, req.ResourceTypeID)
	if err != nil {
		return nil, fmt.Errorf("resource type not found: %w", err)
	}

	// Create resource through inventory service
	return a.inventoryService.CreateResource(ctx, req, pool, resourceType)
}

// UpdateResource updates an existing resource
func (a *O2Adaptor) UpdateResource(ctx context.Context, resourceID string, req *models.UpdateResourceRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating resource", "resourceID", resourceID)

	return a.inventoryService.UpdateResource(ctx, resourceID, req)
}

// DeleteResource deletes a resource
func (a *O2Adaptor) DeleteResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting resource", "resourceID", resourceID)

	return a.inventoryService.DeleteResource(ctx, resourceID)
}

// Deployment Template Management

// GetDeploymentTemplates retrieves deployment templates with optional filtering
func (a *O2Adaptor) GetDeploymentTemplates(ctx context.Context, filter *models.DeploymentTemplateFilter) ([]*models.DeploymentTemplate, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting deployment templates", "filter", filter)

	return a.catalogService.ListDeploymentTemplates(ctx, filter)
}

// GetDeploymentTemplate retrieves a specific deployment template by ID
func (a *O2Adaptor) GetDeploymentTemplate(ctx context.Context, templateID string) (*models.DeploymentTemplate, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting deployment template", "templateID", templateID)

	return a.catalogService.GetDeploymentTemplate(ctx, templateID)
}

// CreateDeploymentTemplate creates a new deployment template
func (a *O2Adaptor) CreateDeploymentTemplate(ctx context.Context, req *models.CreateDeploymentTemplateRequest) (*models.DeploymentTemplate, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating deployment template", "name", req.Name, "type", req.Type)

	// Validate request
	if err := a.validateCreateDeploymentTemplateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Generate template ID
	templateID := a.generateDeploymentTemplateID(req.Name, req.Version)

	// Create deployment template
	template := &models.DeploymentTemplate{
		DeploymentTemplateID: templateID,
		Name:                 req.Name,
		Description:          req.Description,
		Version:              req.Version,
		Category:             req.Category,
		Type:                 req.Type,
		Content:              req.Content,
		InputSchema:          req.InputSchema,
		OutputSchema:         req.OutputSchema,
		Author:               req.Author,
		License:              req.License,
		Keywords:             req.Keywords,
		Dependencies:         req.Dependencies,
		Requirements:         req.Requirements,
		Extensions:           req.Extensions,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	// Register template with catalog service
	if err := a.catalogService.RegisterDeploymentTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to register template: %w", err)
	}

	logger.Info("deployment template created successfully", "templateID", templateID)
	return template, nil
}

// UpdateDeploymentTemplate updates an existing deployment template
func (a *O2Adaptor) UpdateDeploymentTemplate(ctx context.Context, templateID string, req *models.UpdateDeploymentTemplateRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating deployment template", "templateID", templateID)

	// Prepare updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.Content != nil {
		updates["content"] = req.Content
	}
	if req.InputSchema != nil {
		updates["input_schema"] = req.InputSchema
	}
	if req.OutputSchema != nil {
		updates["output_schema"] = req.OutputSchema
	}
	if req.Keywords != nil {
		updates["keywords"] = req.Keywords
	}
	if req.Dependencies != nil {
		updates["dependencies"] = req.Dependencies
	}
	if req.Requirements != nil {
		updates["requirements"] = req.Requirements
	}
	if req.Extensions != nil {
		updates["extensions"] = req.Extensions
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	return a.catalogService.UpdateDeploymentTemplate(ctx, templateID, updates)
}

// DeleteDeploymentTemplate deletes a deployment template
func (a *O2Adaptor) DeleteDeploymentTemplate(ctx context.Context, templateID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting deployment template", "templateID", templateID)

	return a.catalogService.UnregisterDeploymentTemplate(ctx, templateID)
}

// Deployment Manager Interface

// CreateDeployment creates a new deployment
func (a *O2Adaptor) CreateDeployment(ctx context.Context, req *models.CreateDeploymentRequest) (*models.Deployment, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating deployment", "name", req.Name, "templateID", req.TemplateID)

	// Validate request
	if err := a.validateCreateDeploymentRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if template exists
	template, err := a.GetDeploymentTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("deployment template not found: %w", err)
	}

	// Check if resource pool exists
	pool, err := a.GetResourcePool(ctx, req.ResourcePoolID)
	if err != nil {
		return nil, fmt.Errorf("resource pool not found: %w", err)
	}

	// Create deployment through lifecycle service
	return a.lifecycleService.CreateDeployment(ctx, req, template, pool)
}

// GetDeployments retrieves deployments with optional filtering
func (a *O2Adaptor) GetDeployments(ctx context.Context, filter *models.DeploymentFilter) ([]*models.Deployment, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting deployments", "filter", filter)

	return a.lifecycleService.ListDeployments(ctx, filter)
}

// GetDeployment retrieves a specific deployment by ID
func (a *O2Adaptor) GetDeployment(ctx context.Context, deploymentID string) (*models.Deployment, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting deployment", "deploymentID", deploymentID)

	return a.lifecycleService.GetDeployment(ctx, deploymentID)
}

// UpdateDeployment updates an existing deployment
func (a *O2Adaptor) UpdateDeployment(ctx context.Context, deploymentID string, req *models.UpdateDeploymentRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating deployment", "deploymentID", deploymentID)

	return a.lifecycleService.UpdateDeployment(ctx, deploymentID, req)
}

// DeleteDeployment deletes a deployment
func (a *O2Adaptor) DeleteDeployment(ctx context.Context, deploymentID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting deployment", "deploymentID", deploymentID)

	return a.lifecycleService.DeleteDeployment(ctx, deploymentID)
}

// Subscription Management

// CreateSubscription creates a new subscription
func (a *O2Adaptor) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating subscription", "callback", req.Callback, "eventTypes", req.EventTypes)

	return a.subscriptionService.CreateSubscription(ctx, req)
}

// GetSubscriptions retrieves subscriptions with optional filtering
func (a *O2Adaptor) GetSubscriptions(ctx context.Context, filter *models.SubscriptionQueryFilter) ([]*models.Subscription, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting subscriptions", "filter", filter)

	return a.subscriptionService.ListSubscriptions(ctx, filter)
}

// GetSubscription retrieves a specific subscription by ID
func (a *O2Adaptor) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting subscription", "subscriptionID", subscriptionID)

	return a.subscriptionService.GetSubscription(ctx, subscriptionID)
}

// UpdateSubscription updates an existing subscription
func (a *O2Adaptor) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating subscription", "subscriptionID", subscriptionID)

	return a.subscriptionService.UpdateSubscription(ctx, subscriptionID, req)
}

// DeleteSubscription deletes a subscription
func (a *O2Adaptor) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting subscription", "subscriptionID", subscriptionID)

	return a.subscriptionService.DeleteSubscription(ctx, subscriptionID)
}

// Notification Management

// GetNotificationEventTypes retrieves available notification event types
func (a *O2Adaptor) GetNotificationEventTypes(ctx context.Context) ([]*models.NotificationEventType, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting notification event types")

	return a.subscriptionService.GetEventTypes(ctx)
}

// Infrastructure Inventory Management

// GetInventoryNodes retrieves nodes from the infrastructure inventory
func (a *O2Adaptor) GetInventoryNodes(ctx context.Context, filter *models.NodeFilter) ([]*models.Node, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting inventory nodes", "filter", filter)

	return a.inventoryService.ListNodes(ctx, filter)
}

// GetInventoryNode retrieves a specific node by ID
func (a *O2Adaptor) GetInventoryNode(ctx context.Context, nodeID string) (*models.Node, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting inventory node", "nodeID", nodeID)

	return a.inventoryService.GetNode(ctx, nodeID)
}

// Alarm and Fault Management

// GetAlarms retrieves alarms with optional filtering
func (a *O2Adaptor) GetAlarms(ctx context.Context, filter *models.AlarmFilter) ([]*models.Alarm, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting alarms", "filter", filter)

	return a.subscriptionService.GetAlarms(ctx, filter)
}

// GetAlarm retrieves a specific alarm by ID
func (a *O2Adaptor) GetAlarm(ctx context.Context, alarmID string) (*models.Alarm, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting alarm", "alarmID", alarmID)

	return a.subscriptionService.GetAlarm(ctx, alarmID)
}

// AcknowledgeAlarm acknowledges an alarm
func (a *O2Adaptor) AcknowledgeAlarm(ctx context.Context, alarmID string, req *models.AlarmAcknowledgementRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("acknowledging alarm", "alarmID", alarmID, "acknowledgedBy", req.AcknowledgedBy)

	return a.subscriptionService.AcknowledgeAlarm(ctx, alarmID, req)
}

// ClearAlarm clears an alarm
func (a *O2Adaptor) ClearAlarm(ctx context.Context, alarmID string, req *models.AlarmClearRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("clearing alarm", "alarmID", alarmID, "clearedBy", req.ClearedBy)

	return a.subscriptionService.ClearAlarm(ctx, alarmID, req)
}

// Private helper methods for validation and processing

func (a *O2Adaptor) validateCreateResourcePoolRequest(req *models.CreateResourcePoolRequest) error {
	if req.Name == "" {
		return fmt.Errorf("resource pool name is required")
	}
	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if req.OCloudID == "" {
		return fmt.Errorf("oCloudId is required")
	}
	return nil
}

func (a *O2Adaptor) validateCreateResourceRequest(req *models.CreateResourceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("resource name is required")
	}
	if req.ResourceTypeID == "" {
		return fmt.Errorf("resource type ID is required")
	}
	if req.ResourcePoolID == "" {
		return fmt.Errorf("resource pool ID is required")
	}
	return nil
}

func (a *O2Adaptor) validateCreateDeploymentTemplateRequest(req *models.CreateDeploymentTemplateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if req.Version == "" {
		return fmt.Errorf("template version is required")
	}
	if req.Category == "" {
		return fmt.Errorf("template category is required")
	}
	if req.Type == "" {
		return fmt.Errorf("template type is required")
	}
	if req.Content == nil {
		return fmt.Errorf("template content is required")
	}

	// Validate category
	validCategories := []string{
		models.TemplateCategoryVNF,
		models.TemplateCategoryCNF,
		models.TemplateCategoryPNF,
		models.TemplateCategoryNS,
	}

	validCategory := false
	for _, category := range validCategories {
		if req.Category == category {
			validCategory = true
			break
		}
	}
	if !validCategory {
		return fmt.Errorf("invalid template category: %s", req.Category)
	}

	// Validate type
	validTypes := []string{
		models.TemplateTypeHelm,
		models.TemplateTypeKubernetes,
		models.TemplateTypeTerraform,
		models.TemplateTypeAnsible,
	}

	validType := false
	for _, templateType := range validTypes {
		if req.Type == templateType {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid template type: %s", req.Type)
	}

	return nil
}

func (a *O2Adaptor) validateCreateDeploymentRequest(req *models.CreateDeploymentRequest) error {
	if req.Name == "" {
		return fmt.Errorf("deployment name is required")
	}
	if req.TemplateID == "" {
		return fmt.Errorf("template ID is required")
	}
	if req.ResourcePoolID == "" {
		return fmt.Errorf("resource pool ID is required")
	}
	return nil
}

func (a *O2Adaptor) generateResourcePoolID(name, provider string) string {
	// Generate a unique resource pool ID
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("pool-%s-%s-%d",
		strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		provider,
		timestamp)
}

func (a *O2Adaptor) generateDeploymentTemplateID(name, version string) string {
	// Generate a unique deployment template ID
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("template-%s-%s-%d",
		strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		strings.ToLower(strings.ReplaceAll(version, ".", "-")),
		timestamp)
}

func (a *O2Adaptor) matchesResourcePoolFilter(pool *models.ResourcePool, filter *models.ResourcePoolFilter) bool {
	if filter == nil {
		return true
	}

	// Check names filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if pool.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check oCloudIds filter
	if len(filter.OCloudIDs) > 0 {
		found := false
		for _, oCloudID := range filter.OCloudIDs {
			if pool.OCloudID == oCloudID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check providers filter
	if len(filter.Providers) > 0 {
		found := false
		for _, provider := range filter.Providers {
			if pool.Provider == provider {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check regions filter
	if len(filter.Regions) > 0 {
		found := false
		for _, region := range filter.Regions {
			if pool.Region == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check states filter
	if len(filter.States) > 0 && pool.Status != nil {
		found := false
		for _, state := range filter.States {
			if pool.Status.State == state {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check health states filter
	if len(filter.HealthStates) > 0 && pool.Status != nil {
		found := false
		for _, health := range filter.HealthStates {
			if pool.Status.Health == health {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check created after filter
	if filter.CreatedAfter != nil && pool.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	// Check created before filter
	if filter.CreatedBefore != nil && pool.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (a *O2Adaptor) sortAndPaginateResourcePools(pools []*models.ResourcePool, filter *models.ResourcePoolFilter) []*models.ResourcePool {
	// Apply sorting if specified
	if filter != nil && filter.SortBy != "" {
		// Implement sorting based on sortBy field
		// For brevity, not implementing full sorting here
	}

	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(pools) {
			return []*models.ResourcePool{}
		}
		if end > len(pools) {
			end = len(pools)
		}
		return pools[start:end]
	}

	return pools
}

func (a *O2Adaptor) getProviderCapacity(ctx context.Context, provider interface{}) (*models.ResourceCapacity, error) {
	// This would query the actual provider for capacity information
	// For now, return default capacity
	return &models.ResourceCapacity{
		CPU: &models.ResourceMetric{
			Total:       "1000",
			Available:   "800",
			Used:        "200",
			Unit:        "cores",
			Utilization: 20.0,
		},
		Memory: &models.ResourceMetric{
			Total:       "1000000000000", // 1TB
			Available:   "800000000000",  // 800GB
			Used:        "200000000000",  // 200GB
			Unit:        "bytes",
			Utilization: 20.0,
		},
		Storage: &models.ResourceMetric{
			Total:       "10000000000000", // 10TB
			Available:   "8000000000000",  // 8TB
			Used:        "2000000000000",  // 2TB
			Unit:        "bytes",
			Utilization: 20.0,
		},
	}, nil
}

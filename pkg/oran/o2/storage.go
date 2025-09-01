// Package o2 implements storage abstraction and initialization for O2 IMS.

package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// initializeStorage initializes the storage backend based on configuration.

func (s *O2APIServer) initializeStorage() (O2IMSStorage, error) {

	switch s.config.DatabaseType {

	case "memory", "":

		s.logger.Info("initializing in-memory storage")

		return NewInMemoryStorage(), nil

	case "sqlite":

		s.logger.Info("initializing SQLite storage", "url", s.config.DatabaseURL)

		// In a real implementation, this would initialize SQLite.

		return NewInMemoryStorage(), nil // Fallback to memory for now

	case "postgres":

		s.logger.Info("initializing PostgreSQL storage", "url", s.config.DatabaseURL)

		// In a real implementation, this would initialize PostgreSQL.

		return NewInMemoryStorage(), nil // Fallback to memory for now

	case "mysql":

		s.logger.Info("initializing MySQL storage", "url", s.config.DatabaseURL)

		// In a real implementation, this would initialize MySQL.

		return NewInMemoryStorage(), nil // Fallback to memory for now

	default:

		return nil, fmt.Errorf("unsupported database type: %s", s.config.DatabaseType)

	}

}

// InMemoryStorage provides an in-memory implementation of O2IMSStorage for development and testing.

type InMemoryStorage struct {

	// Resource pools.

	resourcePools map[string]*models.ResourcePool

	rpMutex sync.RWMutex

	// Resource types.

	resourceTypes map[string]*models.ResourceType

	rtMutex sync.RWMutex

	// Resources.

	resources map[string]*models.Resource

	rMutex sync.RWMutex

	// Deployment templates.

	deploymentTemplates map[string]*DeploymentTemplate

	dtMutex sync.RWMutex

	// Deployments.

	deployments map[string]*Deployment

	dMutex sync.RWMutex

	// Subscriptions.

	subscriptions map[string]*Subscription

	sMutex sync.RWMutex

	// Cloud providers.

	cloudProviders map[string]*CloudProviderConfig

	cpMutex sync.RWMutex
}

// NewInMemoryStorage creates a new in-memory storage instance.

func NewInMemoryStorage() O2IMSStorage {

	return &InMemoryStorage{

		resourcePools: make(map[string]*models.ResourcePool),

		resourceTypes: make(map[string]*models.ResourceType),

		resources: make(map[string]*models.Resource),

		deploymentTemplates: make(map[string]*DeploymentTemplate),

		deployments: make(map[string]*Deployment),

		subscriptions: make(map[string]*Subscription),

		cloudProviders: make(map[string]*CloudProviderConfig),
	}

}

// Resource Pool storage implementation.

// StoreResourcePool stores a resource pool.

func (s *InMemoryStorage) StoreResourcePool(ctx context.Context, pool *models.ResourcePool) error {

	s.rpMutex.Lock()

	defer s.rpMutex.Unlock()

	s.resourcePools[pool.ResourcePoolID] = pool

	return nil

}

// GetResourcePool retrieves a resource pool by ID.

func (s *InMemoryStorage) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {

	s.rpMutex.RLock()

	defer s.rpMutex.RUnlock()

	pool, exists := s.resourcePools[poolID]

	if !exists {

		return nil, fmt.Errorf("resource pool not found: %s", poolID)

	}

	// Return a copy to prevent modification.

	poolCopy := *pool

	return &poolCopy, nil

}

// ListResourcePools lists resource pools with optional filtering.

func (s *InMemoryStorage) ListResourcePools(ctx context.Context, filter *models.ResourcePoolFilter) ([]*models.ResourcePool, error) {

	s.rpMutex.RLock()

	defer s.rpMutex.RUnlock()

	var pools []*models.ResourcePool

	for _, pool := range s.resourcePools {

		if s.matchesResourcePoolFilter(pool, filter) {

			poolCopy := *pool

			pools = append(pools, &poolCopy)

		}

	}

	// Apply pagination.

	if filter != nil {

		pools = s.applyPagination(pools, filter.Limit, filter.Offset)

	}

	return pools, nil

}

// UpdateResourcePool updates a resource pool.

func (s *InMemoryStorage) UpdateResourcePool(ctx context.Context, poolID string, updates map[string]interface{}) error {

	s.rpMutex.Lock()

	defer s.rpMutex.Unlock()

	pool, exists := s.resourcePools[poolID]

	if !exists {

		return fmt.Errorf("resource pool not found: %s", poolID)

	}

	// Apply updates.

	if name, ok := updates["name"].(string); ok {

		pool.Name = name

	}

	if description, ok := updates["description"].(string); ok {

		pool.Description = description

	}

	if location, ok := updates["location"].(string); ok {

		pool.Location = location

	}

	if updatedAt, ok := updates["updated_at"].(time.Time); ok {

		pool.UpdatedAt = updatedAt

	}

	return nil

}

// DeleteResourcePool deletes a resource pool.

func (s *InMemoryStorage) DeleteResourcePool(ctx context.Context, poolID string) error {

	s.rpMutex.Lock()

	defer s.rpMutex.Unlock()

	if _, exists := s.resourcePools[poolID]; !exists {

		return fmt.Errorf("resource pool not found: %s", poolID)

	}

	delete(s.resourcePools, poolID)

	return nil

}

// Resource Type storage implementation.

// StoreResourceType stores a resource type.

func (s *InMemoryStorage) StoreResourceType(ctx context.Context, resourceType *models.ResourceType) error {

	s.rtMutex.Lock()

	defer s.rtMutex.Unlock()

	s.resourceTypes[resourceType.ResourceTypeID] = resourceType

	return nil

}

// GetResourceType retrieves a resource type by ID.

func (s *InMemoryStorage) GetResourceType(ctx context.Context, typeID string) (*models.ResourceType, error) {

	s.rtMutex.RLock()

	defer s.rtMutex.RUnlock()

	resourceType, exists := s.resourceTypes[typeID]

	if !exists {

		return nil, fmt.Errorf("resource type not found: %s", typeID)

	}

	// Return a copy to prevent modification.

	typeCopy := *resourceType

	return &typeCopy, nil

}

// ListResourceTypes lists resource types with optional filtering.

func (s *InMemoryStorage) ListResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error) {

	s.rtMutex.RLock()

	defer s.rtMutex.RUnlock()

	var resourceTypes []*models.ResourceType

	for _, resourceType := range s.resourceTypes {

		if s.matchesResourceTypeFilter(resourceType, filter) {

			typeCopy := *resourceType

			resourceTypes = append(resourceTypes, &typeCopy)

		}

	}

	// Apply pagination.

	if filter != nil {

		resourceTypes = s.applyResourceTypePagination(resourceTypes, filter.Limit, filter.Offset)

	}

	return resourceTypes, nil

}

// UpdateResourceType updates a resource type.

func (s *InMemoryStorage) UpdateResourceType(ctx context.Context, typeID string, resourceType *models.ResourceType) error {

	s.rtMutex.Lock()

	defer s.rtMutex.Unlock()

	if _, exists := s.resourceTypes[typeID]; !exists {

		return fmt.Errorf("resource type not found: %s", typeID)

	}

	s.resourceTypes[typeID] = resourceType

	return nil

}

// DeleteResourceType deletes a resource type.

func (s *InMemoryStorage) DeleteResourceType(ctx context.Context, typeID string) error {

	s.rtMutex.Lock()

	defer s.rtMutex.Unlock()

	if _, exists := s.resourceTypes[typeID]; !exists {

		return fmt.Errorf("resource type not found: %s", typeID)

	}

	delete(s.resourceTypes, typeID)

	return nil

}

// Resource storage implementation.

// StoreResource stores a resource.

func (s *InMemoryStorage) StoreResource(ctx context.Context, resource *models.Resource) error {

	s.rMutex.Lock()

	defer s.rMutex.Unlock()

	s.resources[resource.ResourceID] = resource

	return nil

}

// GetResource retrieves a resource by ID.

func (s *InMemoryStorage) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {

	s.rMutex.RLock()

	defer s.rMutex.RUnlock()

	resource, exists := s.resources[resourceID]

	if !exists {

		return nil, fmt.Errorf("resource not found: %s", resourceID)

	}

	// Return a copy to prevent modification.

	resourceCopy := *resource

	return &resourceCopy, nil

}

// ListResources lists resources with optional filtering.

func (s *InMemoryStorage) ListResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error) {

	s.rMutex.RLock()

	defer s.rMutex.RUnlock()

	var resources []*models.Resource

	for _, resource := range s.resources {

		if s.matchesResourceFilter(resource, filter) {

			resourceCopy := *resource

			resources = append(resources, &resourceCopy)

		}

	}

	// Apply pagination.

	if filter != nil {

		resources = s.applyResourcePagination(resources, filter.Limit, filter.Offset)

	}

	return resources, nil

}

// UpdateResource updates a resource.

func (s *InMemoryStorage) UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error {

	s.rMutex.Lock()

	defer s.rMutex.Unlock()

	resource, exists := s.resources[resourceID]

	if !exists {

		return fmt.Errorf("resource not found: %s", resourceID)

	}

	// Apply updates.

	if updatedAt, ok := updates["updated_at"].(time.Time); ok {

		resource.UpdatedAt = updatedAt

	}

	if status, ok := updates["status"].(string); ok {

		if resource.Status == nil {

			resource.Status = &models.ResourceStatus{}

		}

		resource.Status.State = status

	}

	return nil

}

// DeleteResource deletes a resource.

func (s *InMemoryStorage) DeleteResource(ctx context.Context, resourceID string) error {

	s.rMutex.Lock()

	defer s.rMutex.Unlock()

	if _, exists := s.resources[resourceID]; !exists {

		return fmt.Errorf("resource not found: %s", resourceID)

	}

	delete(s.resources, resourceID)

	return nil

}

// Placeholder implementations for other storage methods.

// StoreDeploymentTemplate stores a deployment template.

func (s *InMemoryStorage) StoreDeploymentTemplate(ctx context.Context, template *DeploymentTemplate) error {

	s.dtMutex.Lock()

	defer s.dtMutex.Unlock()

	s.deploymentTemplates[template.DeploymentTemplateID] = template

	return nil

}

// GetDeploymentTemplate retrieves a deployment template by ID.

func (s *InMemoryStorage) GetDeploymentTemplate(ctx context.Context, templateID string) (*DeploymentTemplate, error) {

	s.dtMutex.RLock()

	defer s.dtMutex.RUnlock()

	template, exists := s.deploymentTemplates[templateID]

	if !exists {

		return nil, fmt.Errorf("deployment template not found: %s", templateID)

	}

	return template, nil

}

// ListDeploymentTemplates lists deployment templates.

func (s *InMemoryStorage) ListDeploymentTemplates(ctx context.Context, filter *DeploymentTemplateFilter) ([]*DeploymentTemplate, error) {

	s.dtMutex.RLock()

	defer s.dtMutex.RUnlock()

	var templates []*DeploymentTemplate

	for _, template := range s.deploymentTemplates {

		templates = append(templates, template)

	}

	return templates, nil

}

// UpdateDeploymentTemplate updates a deployment template.

func (s *InMemoryStorage) UpdateDeploymentTemplate(ctx context.Context, templateID string, template *DeploymentTemplate) error {

	s.dtMutex.Lock()

	defer s.dtMutex.Unlock()

	if _, exists := s.deploymentTemplates[templateID]; !exists {

		return fmt.Errorf("deployment template not found: %s", templateID)

	}

	s.deploymentTemplates[templateID] = template

	return nil

}

// DeleteDeploymentTemplate deletes a deployment template.

func (s *InMemoryStorage) DeleteDeploymentTemplate(ctx context.Context, templateID string) error {

	s.dtMutex.Lock()

	defer s.dtMutex.Unlock()

	if _, exists := s.deploymentTemplates[templateID]; !exists {

		return fmt.Errorf("deployment template not found: %s", templateID)

	}

	delete(s.deploymentTemplates, templateID)

	return nil

}

// StoreDeployment stores a deployment.

func (s *InMemoryStorage) StoreDeployment(ctx context.Context, deployment *Deployment) error {

	s.dMutex.Lock()

	defer s.dMutex.Unlock()

	s.deployments[deployment.DeploymentManagerID] = deployment

	return nil

}

// GetDeployment retrieves a deployment by ID.

func (s *InMemoryStorage) GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error) {

	s.dMutex.RLock()

	defer s.dMutex.RUnlock()

	deployment, exists := s.deployments[deploymentID]

	if !exists {

		return nil, fmt.Errorf("deployment not found: %s", deploymentID)

	}

	return deployment, nil

}

// ListDeployments lists deployments.

func (s *InMemoryStorage) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*Deployment, error) {

	s.dMutex.RLock()

	defer s.dMutex.RUnlock()

	var deployments []*Deployment

	for _, deployment := range s.deployments {

		deployments = append(deployments, deployment)

	}

	return deployments, nil

}

// UpdateDeployment updates a deployment.

func (s *InMemoryStorage) UpdateDeployment(ctx context.Context, deploymentID string, updates map[string]interface{}) error {

	s.dMutex.Lock()

	defer s.dMutex.Unlock()

	if _, exists := s.deployments[deploymentID]; !exists {

		return fmt.Errorf("deployment not found: %s", deploymentID)

	}

	// Apply updates would be implemented here.

	return nil

}

// DeleteDeployment deletes a deployment.

func (s *InMemoryStorage) DeleteDeployment(ctx context.Context, deploymentID string) error {

	s.dMutex.Lock()

	defer s.dMutex.Unlock()

	if _, exists := s.deployments[deploymentID]; !exists {

		return fmt.Errorf("deployment not found: %s", deploymentID)

	}

	delete(s.deployments, deploymentID)

	return nil

}

// StoreSubscription stores a subscription.

func (s *InMemoryStorage) StoreSubscription(ctx context.Context, subscription *Subscription) error {

	s.sMutex.Lock()

	defer s.sMutex.Unlock()

	s.subscriptions[subscription.SubscriptionID] = subscription

	return nil

}

// GetSubscription retrieves a subscription by ID.

func (s *InMemoryStorage) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {

	s.sMutex.RLock()

	defer s.sMutex.RUnlock()

	subscription, exists := s.subscriptions[subscriptionID]

	if !exists {

		return nil, fmt.Errorf("subscription not found: %s", subscriptionID)

	}

	return subscription, nil

}

// ListSubscriptions lists subscriptions.

func (s *InMemoryStorage) ListSubscriptions(ctx context.Context, filter *SubscriptionFilter) ([]*Subscription, error) {

	s.sMutex.RLock()

	defer s.sMutex.RUnlock()

	var subscriptions []*Subscription

	for _, subscription := range s.subscriptions {

		subscriptions = append(subscriptions, subscription)

	}

	return subscriptions, nil

}

// UpdateSubscription updates a subscription.

func (s *InMemoryStorage) UpdateSubscription(ctx context.Context, subscriptionID string, subscription *Subscription) error {

	s.sMutex.Lock()

	defer s.sMutex.Unlock()

	if _, exists := s.subscriptions[subscriptionID]; !exists {

		return fmt.Errorf("subscription not found: %s", subscriptionID)

	}

	s.subscriptions[subscriptionID] = subscription

	return nil

}

// DeleteSubscription deletes a subscription.

func (s *InMemoryStorage) DeleteSubscription(ctx context.Context, subscriptionID string) error {

	s.sMutex.Lock()

	defer s.sMutex.Unlock()

	if _, exists := s.subscriptions[subscriptionID]; !exists {

		return fmt.Errorf("subscription not found: %s", subscriptionID)

	}

	delete(s.subscriptions, subscriptionID)

	return nil

}

// StoreCloudProvider stores a cloud provider.

func (s *InMemoryStorage) StoreCloudProvider(ctx context.Context, provider *CloudProviderConfig) error {

	s.cpMutex.Lock()

	defer s.cpMutex.Unlock()

	s.cloudProviders[provider.ProviderID] = provider

	return nil

}

// GetCloudProvider retrieves a cloud provider by ID.

func (s *InMemoryStorage) GetCloudProvider(ctx context.Context, providerID string) (*CloudProviderConfig, error) {

	s.cpMutex.RLock()

	defer s.cpMutex.RUnlock()

	provider, exists := s.cloudProviders[providerID]

	if !exists {

		return nil, fmt.Errorf("cloud provider not found: %s", providerID)

	}

	return provider, nil

}

// ListCloudProviders lists cloud providers.

func (s *InMemoryStorage) ListCloudProviders(ctx context.Context) ([]*CloudProviderConfig, error) {

	s.cpMutex.RLock()

	defer s.cpMutex.RUnlock()

	var providers []*CloudProviderConfig

	for _, provider := range s.cloudProviders {

		providers = append(providers, provider)

	}

	return providers, nil

}

// UpdateCloudProvider updates a cloud provider.

func (s *InMemoryStorage) UpdateCloudProvider(ctx context.Context, providerID string, provider *CloudProviderConfig) error {

	s.cpMutex.Lock()

	defer s.cpMutex.Unlock()

	if _, exists := s.cloudProviders[providerID]; !exists {

		return fmt.Errorf("cloud provider not found: %s", providerID)

	}

	s.cloudProviders[providerID] = provider

	return nil

}

// DeleteCloudProvider deletes a cloud provider.

func (s *InMemoryStorage) DeleteCloudProvider(ctx context.Context, providerID string) error {

	s.cpMutex.Lock()

	defer s.cpMutex.Unlock()

	if _, exists := s.cloudProviders[providerID]; !exists {

		return fmt.Errorf("cloud provider not found: %s", providerID)

	}

	delete(s.cloudProviders, providerID)

	return nil

}

// UpdateResourceStatus updates resource status.

func (s *InMemoryStorage) UpdateResourceStatus(ctx context.Context, resourceID string, status *models.ResourceStatus) error {

	s.rMutex.Lock()

	defer s.rMutex.Unlock()

	resource, exists := s.resources[resourceID]

	if !exists {

		return fmt.Errorf("resource not found: %s", resourceID)

	}

	resource.Status = status

	resource.UpdatedAt = time.Now()

	return nil

}

// GetResourceHistory retrieves resource history.

func (s *InMemoryStorage) GetResourceHistory(ctx context.Context, resourceID string, limit int) ([]*ResourceHistoryEntry, error) {

	// In a real implementation, this would retrieve actual history.

	return []*ResourceHistoryEntry{}, nil

}

// BackupData creates a backup of the data.
func (s *InMemoryStorage) BackupData(ctx context.Context, backupID string) error {
	// In-memory storage backup simulation
	return nil
}

// RestoreData restores data from a backup.
func (s *InMemoryStorage) RestoreData(ctx context.Context, backupID string) error {
	// In-memory storage restore simulation
	return nil
}

// StoreInventory stores an inventory asset.
func (s *InMemoryStorage) StoreInventory(ctx context.Context, inventory *InfrastructureAsset) error {
	// In a real implementation, this would store inventory data
	return nil
}

// RetrieveInventory retrieves an inventory asset.
func (s *InMemoryStorage) RetrieveInventory(ctx context.Context, assetID string) (*InfrastructureAsset, error) {
	// In a real implementation, this would retrieve inventory data
	return nil, fmt.Errorf("inventory asset not found: %s", assetID)
}

// StoreLifecycleOperation stores a lifecycle operation.
func (s *InMemoryStorage) StoreLifecycleOperation(ctx context.Context, operation *LifecycleOperation) error {
	// In a real implementation, this would store lifecycle operations
	return nil
}

// RetrieveLifecycleOperation retrieves a lifecycle operation.
func (s *InMemoryStorage) RetrieveLifecycleOperation(ctx context.Context, operationID string) (*LifecycleOperation, error) {
	// In a real implementation, this would retrieve lifecycle operations
	return nil, fmt.Errorf("lifecycle operation not found: %s", operationID)
}

// CheckStorageHealth checks the health of the storage system.
func (s *InMemoryStorage) CheckStorageHealth(ctx context.Context) (*o2models.HealthStatus, error) {
	// In-memory storage is always healthy
	return &o2models.HealthStatus{
		Status:    "UP",
		Timestamp: time.Now(),
		Details:   map[string]interface{}{"storage": "in-memory"},
	}, nil
}

// GetStorageMetrics returns storage metrics.
func (s *InMemoryStorage) GetStorageMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"resource_pools": len(s.resourcePools),
		"resource_types": len(s.resourceTypes),
		"resources":      len(s.resources),
		"deployments":    len(s.deployments),
		"subscriptions":  len(s.subscriptions),
		"providers":      len(s.cloudProviders),
	}, nil
}

// BeginTransaction begins a transaction.

func (s *InMemoryStorage) BeginTransaction(ctx context.Context) (interface{}, error) {

	// In-memory storage doesn't need real transactions.

	return &struct{}{}, nil

}

// CommitTransaction commits a transaction.

func (s *InMemoryStorage) CommitTransaction(ctx context.Context, tx interface{}) error {

	// In-memory storage doesn't need real transactions.

	return nil

}

// RollbackTransaction rolls back a transaction.

func (s *InMemoryStorage) RollbackTransaction(ctx context.Context, tx interface{}) error {

	// In-memory storage doesn't need real transactions.

	return nil

}

// Helper methods for filtering and pagination.

// matchesResourcePoolFilter checks if a resource pool matches the filter.

func (s *InMemoryStorage) matchesResourcePoolFilter(pool *models.ResourcePool, filter *models.ResourcePoolFilter) bool {

	if filter == nil {

		return true

	}

	// Check names filter.

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

	// Check regions filter (locations mapped to regions).

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

	// Check providers filter.

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

	return true

}

// matchesResourceTypeFilter checks if a resource type matches the filter.

func (s *InMemoryStorage) matchesResourceTypeFilter(resourceType *models.ResourceType, filter *models.ResourceTypeFilter) bool {

	if filter == nil {

		return true

	}

	// Check names filter.

	if len(filter.Names) > 0 {

		found := false

		for _, name := range filter.Names {

			if resourceType.Name == name {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	// Check categories filter.

	if len(filter.Categories) > 0 && resourceType.Specifications != nil {

		found := false

		for _, category := range filter.Categories {

			if resourceType.Specifications.Category == category {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	// Check vendors filter.

	if len(filter.Vendors) > 0 {

		found := false

		for _, vendor := range filter.Vendors {

			if resourceType.Vendor == vendor {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	return true

}

// matchesResourceFilter checks if a resource matches the filter.

func (s *InMemoryStorage) matchesResourceFilter(resource *models.Resource, filter *models.ResourceFilter) bool {

	if filter == nil {

		return true

	}

	// Check resource pool IDs filter.

	if len(filter.ResourcePoolIDs) > 0 {

		found := false

		for _, poolID := range filter.ResourcePoolIDs {

			if resource.ResourcePoolID == poolID {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	// Check resource type IDs filter.

	if len(filter.ResourceTypeIDs) > 0 {

		found := false

		for _, typeID := range filter.ResourceTypeIDs {

			if resource.ResourceTypeID == typeID {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	// Check lifecycle states filter (mapped from statuses).

	if len(filter.LifecycleStates) > 0 {

		found := false

		for _, status := range filter.LifecycleStates {

			if resource.Status != nil && resource.Status.State == status {

				found = true

				break

			}

		}

		if !found {

			return false

		}

	}

	return true

}

// Generic pagination helpers.

func (s *InMemoryStorage) applyPagination(pools []*models.ResourcePool, limit, offset int) []*models.ResourcePool {

	if limit <= 0 {

		limit = 100 // Default limit

	}

	if offset < 0 {

		offset = 0

	}

	start := offset

	if start >= len(pools) {

		return []*models.ResourcePool{}

	}

	end := start + limit

	if end > len(pools) {

		end = len(pools)

	}

	return pools[start:end]

}

func (s *InMemoryStorage) applyResourceTypePagination(resourceTypes []*models.ResourceType, limit, offset int) []*models.ResourceType {

	if limit <= 0 {

		limit = 100 // Default limit

	}

	if offset < 0 {

		offset = 0

	}

	start := offset

	if start >= len(resourceTypes) {

		return []*models.ResourceType{}

	}

	end := start + limit

	if end > len(resourceTypes) {

		end = len(resourceTypes)

	}

	return resourceTypes[start:end]

}

func (s *InMemoryStorage) applyResourcePagination(resources []*models.Resource, limit, offset int) []*models.Resource {

	if limit <= 0 {

		limit = 100 // Default limit

	}

	if offset < 0 {

		offset = 0

	}

	start := offset

	if start >= len(resources) {

		return []*models.Resource{}

	}

	end := start + limit

	if end > len(resources) {

		end = len(resources)

	}

	return resources[start:end]

}

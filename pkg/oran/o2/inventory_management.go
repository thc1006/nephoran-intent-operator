// Package o2 implements comprehensive inventory management for O2 IMS.

package o2

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// InventoryManagementService provides comprehensive inventory management and CMDB functionality.

type InventoryManagementService struct {
	config *InventoryConfig

	logger *logging.StructuredLogger

	// Provider registry for multi-cloud inventory.

	providerRegistry providers.ProviderRegistry

	// Storage components.

	cmdbStorage CMDBStorage

	assetStorage AssetStorage

	// Inventory components.

	discoveryEngine *DiscoveryEngine

	relationshipEngine *RelationshipEngine

	auditEngine *AuditEngine

	// Caching and indexing.

	assetIndex *AssetIndex

	relationshipIndex *RelationshipIndex

	// Synchronization.

	mu sync.RWMutex

	// Lifecycle management.

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// InventoryConfig configuration for inventory management.

type InventoryConfig struct {
	// Discovery settings.

	AutoDiscoveryEnabled bool `json:"autoDiscoveryEnabled,omitempty"`

	DiscoveryInterval time.Duration `json:"discoveryInterval,omitempty"`

	DiscoveryTimeout time.Duration `json:"discoveryTimeout,omitempty"`

	// Sync settings.

	InventorySyncEnabled bool `json:"inventorySyncEnabled,omitempty"`

	SyncInterval time.Duration `json:"syncInterval,omitempty"`

	ConflictResolution string `json:"conflictResolution,omitempty"`

	// Retention policies.

	AssetRetentionPeriod time.Duration `json:"assetRetentionPeriod,omitempty"`

	AuditRetentionPeriod time.Duration `json:"auditRetentionPeriod,omitempty"`

	// CMDB settings.

	CMDBEnabled bool `json:"cmdbEnabled,omitempty"`

	RelationshipTracking bool `json:"relationshipTracking,omitempty"`

	ChangeTracking bool `json:"changeTracking,omitempty"`

	ComplianceReporting bool `json:"complianceReporting,omitempty"`

	// Database settings.

	DatabaseURL string `json:"databaseUrl,omitempty"`

	MaxConnections int `json:"maxConnections,omitempty"`

	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty"`
}

// Asset represents a managed infrastructure asset.

type Asset struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"`

	Category string `json:"category"`

	Provider string `json:"provider"`

	Region string `json:"region,omitempty"`

	Zone string `json:"zone,omitempty"`

	// Asset properties.

	Properties json.RawMessage `json:"properties"`

	Configuration json.RawMessage `json:"configuration,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	// Asset status.

	Status string `json:"status"`

	Health string `json:"health"`

	State string `json:"state"`

	LastSeen time.Time `json:"lastSeen"`

	LastUpdated time.Time `json:"lastUpdated"`

	// Relationships.

	ParentAsset string `json:"parentAsset,omitempty"`

	ChildAssets []string `json:"childAssets,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	Dependents []string `json:"dependents,omitempty"`

	// Compliance and audit.

	ComplianceStatus string `json:"complianceStatus"`

	ComplianceChecks []ComplianceCheck `json:"complianceChecks,omitempty"`

	AuditTrail []AuditEntry `json:"auditTrail,omitempty"`

	// Lifecycle information.

	CreatedAt time.Time `json:"createdAt"`

	CreatedBy string `json:"createdBy"`

	UpdatedAt time.Time `json:"updatedAt"`

	UpdatedBy string `json:"updatedBy"`
}

// AssetRelationship represents a relationship between two assets.

type AssetRelationship struct {
	ID string `json:"id"`

	SourceAssetID string `json:"sourceAssetId"`

	TargetAssetID string `json:"targetAssetId"`

	RelationType string `json:"relationType"`

	Properties json.RawMessage `json:"properties,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// ComplianceCheck represents a compliance check result.

type ComplianceCheck struct {
	ID string `json:"id"`

	CheckType string `json:"checkType"`

	Status string `json:"status"`

	Result string `json:"result"`

	Details json.RawMessage `json:"details,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// AuditEntry represents an audit trail entry.

type AuditEntry struct {
	ID string `json:"id"`

	Action string `json:"action"`

	Actor string `json:"actor"`

	ResourceType string `json:"resourceType"`

	ResourceID string `json:"resourceId"`

	Changes json.RawMessage `json:"changes,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Source string `json:"source"`
}

// AssetFilter defines filters for querying assets.

type AssetFilter struct {
	IDs []string `json:"ids,omitempty"`

	Names []string `json:"names,omitempty"`

	Types []string `json:"types,omitempty"`

	Categories []string `json:"categories,omitempty"`

	Providers []string `json:"providers,omitempty"`

	Regions []string `json:"regions,omitempty"`

	Zones []string `json:"zones,omitempty"`

	Status []string `json:"status,omitempty"`

	Health []string `json:"health,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	UpdatedAfter *time.Time `json:"updatedAfter,omitempty"`

	UpdatedBefore *time.Time `json:"updatedBefore,omitempty"`

	HasParent *bool `json:"hasParent,omitempty"`

	HasChildren *bool `json:"hasChildren,omitempty"`

	HasDependencies *bool `json:"hasDependencies,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// CMDBStorage interface for CMDB storage operations.

type CMDBStorage interface {
	// Asset operations.

	CreateAsset(ctx context.Context, asset *Asset) error

	GetAsset(ctx context.Context, id string) (*Asset, error)

	UpdateAsset(ctx context.Context, asset *Asset) error

	DeleteAsset(ctx context.Context, id string) error

	ListAssets(ctx context.Context, filter *AssetFilter) ([]*Asset, error)

	// Relationship operations.

	CreateRelationship(ctx context.Context, rel *AssetRelationship) error

	GetRelationship(ctx context.Context, id string) (*AssetRelationship, error)

	UpdateRelationship(ctx context.Context, rel *AssetRelationship) error

	DeleteRelationship(ctx context.Context, id string) error

	ListRelationships(ctx context.Context, filter *RelationshipFilter) ([]*AssetRelationship, error)

	GetAssetRelationships(ctx context.Context, assetID string) ([]*AssetRelationship, error)

	// Audit operations.

	CreateAuditEntry(ctx context.Context, entry *AuditEntry) error

	GetAuditTrail(ctx context.Context, resourceID string) ([]*AuditEntry, error)

	// Compliance operations.

	CreateComplianceCheck(ctx context.Context, check *ComplianceCheck) error

	GetComplianceStatus(ctx context.Context, assetID string) ([]*ComplianceCheck, error)

	// Maintenance operations.

	Backup(ctx context.Context, path string) error

	Restore(ctx context.Context, path string) error

	Cleanup(ctx context.Context, retentionPeriod time.Duration) error
}

// AssetStorage interface for asset storage operations.

type AssetStorage interface {
	Store(ctx context.Context, asset *Asset) error

	Retrieve(ctx context.Context, id string) (*Asset, error)

	Delete(ctx context.Context, id string) error

	List(ctx context.Context, filter *AssetFilter) ([]*Asset, error)
}

// RelationshipFilter defines filters for querying relationships.

type RelationshipFilter struct {
	SourceAssetIDs []string `json:"sourceAssetIds,omitempty"`

	TargetAssetIDs []string `json:"targetAssetIds,omitempty"`

	RelationTypes []string `json:"relationTypes,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`
}

// NewInventoryManagementService creates a new inventory management service.

func NewInventoryManagementService(
	config *InventoryConfig,

	providerRegistry providers.ProviderRegistry,

	logger *logging.StructuredLogger,
) (*InventoryManagementService, error) {
	if config == nil {
		config = DefaultInventoryConfig()
	}

	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("inventory-management", "1.0.0", "production"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize storage.

	assetStorage := &stubAssetStorage{}

	// Initialize engines.

	discoveryEngine := &DiscoveryEngine{}

	relationshipEngine := &RelationshipEngine{}

	auditEngine := &AuditEngine{}

	// Initialize indexes.

	assetIndex := &AssetIndex{}

	relationshipIndex := &RelationshipIndex{}

	service := &InventoryManagementService{
		config: config,

		logger: logger,

		providerRegistry: providerRegistry,

		assetStorage: assetStorage,

		discoveryEngine: discoveryEngine,

		relationshipEngine: relationshipEngine,

		auditEngine: auditEngine,

		assetIndex: assetIndex,

		relationshipIndex: relationshipIndex,

		ctx: ctx,

		cancel: cancel,
	}

	return service, nil
}

// DefaultInventoryConfig returns default configuration.

func DefaultInventoryConfig() *InventoryConfig {
	return &InventoryConfig{
		AutoDiscoveryEnabled: true,

		DiscoveryInterval: 5 * time.Minute,

		DiscoveryTimeout: 30 * time.Second,

		InventorySyncEnabled: true,

		SyncInterval: 15 * time.Minute,

		ConflictResolution: "latest_wins",

		AssetRetentionPeriod: 90 * 24 * time.Hour,

		AuditRetentionPeriod: 365 * 24 * time.Hour,

		CMDBEnabled: true,

		RelationshipTracking: true,

		ChangeTracking: true,

		ComplianceReporting: true,

		MaxConnections: 20,

		ConnectionTimeout: 30 * time.Second,
	}
}

// Start starts the inventory management service.

func (s *InventoryManagementService) Start(ctx context.Context) error {
	s.logger.Info("starting inventory management service")

	// Initialize database schema if needed.

	if err := s.initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Start background processes.

	s.wg.Add(3)

	if s.config.AutoDiscoveryEnabled {
		go s.discoveryLoop()
	}

	if s.config.InventorySyncEnabled {
		go s.syncLoop()
	}

	go s.maintenanceLoop()

	return nil
}

// Stop stops the inventory management service.

func (s *InventoryManagementService) Stop() error {
	s.logger.Info("stopping inventory management service")

	s.cancel()

	s.wg.Wait()

	s.logger.Info("inventory management service stopped")

	return nil
}

// discoveryLoop runs the asset discovery loop.

func (s *InventoryManagementService) discoveryLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.DiscoveryInterval)

	defer ticker.Stop()

	// Run initial discovery.

	s.performDiscovery()

	for {
		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.performDiscovery()

		}
	}
}

// syncLoop runs the inventory synchronization loop.

func (s *InventoryManagementService) syncLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SyncInterval)

	defer ticker.Stop()

	for {
		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.performSync()

		}
	}
}

// maintenanceLoop runs maintenance tasks.

func (s *InventoryManagementService) maintenanceLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(24 * time.Hour) // Run maintenance daily

	defer ticker.Stop()

	for {
		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.performMaintenance()

		}
	}
}

// performDiscovery performs asset discovery across all providers.

func (s *InventoryManagementService) performDiscovery() {
	s.logger.Info("performing asset discovery")

	discoveryCtx, cancel := context.WithTimeout(s.ctx, s.config.DiscoveryTimeout)

	defer cancel()

	providerNames := s.providerRegistry.ListProviders()

	for _, providerName := range providerNames {

		provider, err := s.providerRegistry.GetProvider(providerName)
		if err != nil {

			s.logger.Error("Failed to get provider", "name", providerName, "error", err)

			continue

		}

		go s.discoverProviderAssets(discoveryCtx, provider)

	}
}

// discoverProviderAssets discovers assets from a specific provider.

func (s *InventoryManagementService) discoverProviderAssets(ctx context.Context, provider providers.CloudProvider) {
	s.logger.Info("discovering assets from provider",

		"provider", getProviderType(provider))

	// Use discovery engine to find assets.

	assets, err := s.discoveryEngine.DiscoverAssets(ctx, provider)
	if err != nil {

		s.logger.Error("failed to discover assets from provider",

			"provider", getProviderType(provider),

			"error", err)

		return

	}

	// Process discovered assets.

	for _, asset := range assets {
		if err := s.processDiscoveredAsset(asset); err != nil {
			s.logger.Error("failed to process discovered asset",

				"asset_id", asset.ID,

				"provider", getProviderType(provider),

				"error", err)
		}
	}

	s.logger.Info("completed asset discovery for provider",

		"provider", getProviderType(provider),

		"assets_discovered", len(assets))
}

// processDiscoveredAsset processes a discovered asset.

func (s *InventoryManagementService) processDiscoveredAsset(asset *Asset) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Check if asset already exists.

	existingAsset, err := s.cmdbStorage.GetAsset(s.ctx, asset.ID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing asset: %w", err)
	}

	if existingAsset != nil {
		// Update existing asset.

		if err := s.updateExistingAsset(existingAsset, asset); err != nil {
			return fmt.Errorf("failed to update existing asset: %w", err)
		}
	} else {

		// Create new asset.

		asset.CreatedAt = time.Now()

		asset.UpdatedAt = time.Now()

		asset.CreatedBy = "discovery-engine"

		if err := s.cmdbStorage.CreateAsset(s.ctx, asset); err != nil {
			return fmt.Errorf("failed to create asset: %w", err)
		}

		// Update index.

		s.assetIndex.AddAsset(asset)

		// Create audit entry.

		auditEntry := &AuditEntry{
			ID: uuid.New().String(),

			Action: "create",

			Actor: "discovery-engine",

			ResourceType: "asset",

			ResourceID: asset.ID,

			Timestamp: time.Now(),

			Source: "inventory-discovery",
		}

		if err := s.cmdbStorage.CreateAuditEntry(s.ctx, auditEntry); err != nil {
			s.logger.Warn("failed to create audit entry", "error", err)
		}

	}

	return nil
}

// updateExistingAsset updates an existing asset with discovered information.

func (s *InventoryManagementService) updateExistingAsset(existing, discovered *Asset) error {
	changes := make(map[string]interface{})

	// Update last seen timestamp.

	existing.LastSeen = time.Now()

	changes["lastSeen"] = existing.LastSeen

	// Update properties if changed.

	if !equalMaps(existing.Properties, discovered.Properties) {

		existing.Properties = discovered.Properties

		changes["properties"] = existing.Properties

	}

	// Update configuration if changed.

	if !equalMaps(existing.Configuration, discovered.Configuration) {

		existing.Configuration = discovered.Configuration

		changes["configuration"] = existing.Configuration

	}

	// Update status if changed.

	if existing.Status != discovered.Status {

		existing.Status = discovered.Status

		changes["status"] = existing.Status

	}

	// Update health if changed.

	if existing.Health != discovered.Health {

		existing.Health = discovered.Health

		changes["health"] = existing.Health

	}

	// Update state if changed.

	if existing.State != discovered.State {

		existing.State = discovered.State

		changes["state"] = existing.State

	}

	if len(changes) > 0 {

		existing.UpdatedAt = time.Now()

		existing.UpdatedBy = "discovery-engine"

		changes["updatedAt"] = existing.UpdatedAt

		changes["updatedBy"] = existing.UpdatedBy

		if err := s.cmdbStorage.UpdateAsset(s.ctx, existing); err != nil {
			return fmt.Errorf("failed to update asset in storage: %w", err)
		}

		// Update index.

		s.assetIndex.UpdateAsset(existing)

		// Create audit entry for changes.

		auditEntry := &AuditEntry{
			ID: uuid.New().String(),

			Action: "update",

			Actor: "discovery-engine",

			ResourceType: "asset",

			ResourceID: existing.ID,

			Changes: changes,

			Timestamp: time.Now(),

			Source: "inventory-discovery",
		}

		if err := s.cmdbStorage.CreateAuditEntry(s.ctx, auditEntry); err != nil {
			s.logger.Warn("failed to create audit entry", "error", err)
		}

	}

	return nil
}

// performSync performs inventory synchronization.

func (s *InventoryManagementService) performSync() {
	s.logger.Info("performing inventory synchronization")

	// Sync with external systems, validate relationships, etc.

	// This is a placeholder for more complex sync logic.

	s.logger.Info("completed inventory synchronization")
}

// performMaintenance performs maintenance tasks.

func (s *InventoryManagementService) performMaintenance() {
	s.logger.Info("performing inventory maintenance")

	// Cleanup old audit entries.

	if err := s.cmdbStorage.Cleanup(s.ctx, s.config.AuditRetentionPeriod); err != nil {
		s.logger.Error("failed to cleanup old entries", "error", err)
	}

	// Rebuild indexes if needed.

	s.rebuildIndexes()

	s.logger.Info("completed inventory maintenance")
}

// rebuildIndexes rebuilds asset and relationship indexes.

func (s *InventoryManagementService) rebuildIndexes() {
	s.logger.Info("rebuilding indexes")

	// Rebuild asset index.

	assets, err := s.cmdbStorage.ListAssets(s.ctx, &AssetFilter{})
	if err != nil {

		s.logger.Error("failed to list assets for index rebuild", "error", err)

		return

	}

	s.assetIndex.Clear()

	for _, asset := range assets {
		s.assetIndex.AddAsset(asset)
	}

	// Rebuild relationship index.

	relationships, err := s.cmdbStorage.ListRelationships(s.ctx, &RelationshipFilter{})
	if err != nil {

		s.logger.Error("failed to list relationships for index rebuild", "error", err)

		return

	}

	s.relationshipIndex.Clear()

	for _, rel := range relationships {
		s.relationshipIndex.AddRelationship(rel)
	}

	s.logger.Info("completed index rebuild")
}

// initializeDatabase initializes the database schema.

func (s *InventoryManagementService) initializeDatabase() error {
	// This would typically create tables, indexes, and other database objects.

	// For this example, we'll assume the database is already set up.

	s.logger.Info("initializing database schema")

	return nil
}

// Asset management operations.

// CreateAsset creates a new asset in the inventory.

func (s *InventoryManagementService) CreateAsset(ctx context.Context, asset *Asset) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Set timestamps and metadata.

	now := time.Now()

	asset.ID = uuid.New().String()

	asset.CreatedAt = now

	asset.UpdatedAt = now

	asset.LastSeen = now

	// Store in CMDB.

	if err := s.cmdbStorage.CreateAsset(ctx, asset); err != nil {
		return fmt.Errorf("failed to create asset in CMDB: %w", err)
	}

	// Update index.

	s.assetIndex.AddAsset(asset)

	// Create audit entry.

	auditEntry := &AuditEntry{
		ID: uuid.New().String(),

		Action: "create",

		Actor: "user", // This would come from context

		ResourceType: "asset",

		ResourceID: asset.ID,

		Timestamp: now,

		Source: "api",
	}

	if err := s.cmdbStorage.CreateAuditEntry(ctx, auditEntry); err != nil {
		s.logger.Warn("failed to create audit entry", "error", err)
	}

	s.logger.Info("created asset", "asset_id", asset.ID, "name", asset.Name)

	return nil
}

// GetAsset retrieves an asset by ID.

func (s *InventoryManagementService) GetAsset(ctx context.Context, id string) (*Asset, error) {
	return s.cmdbStorage.GetAsset(ctx, id)
}

// UpdateAsset updates an existing asset.

func (s *InventoryManagementService) UpdateAsset(ctx context.Context, asset *Asset) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Get existing asset to track changes.

	existing, err := s.cmdbStorage.GetAsset(ctx, asset.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing asset: %w", err)
	}

	// Track changes.

	changes := s.trackAssetChanges(existing, asset)

	// Update timestamps.

	asset.UpdatedAt = time.Now()

	// Store in CMDB.

	if err := s.cmdbStorage.UpdateAsset(ctx, asset); err != nil {
		return fmt.Errorf("failed to update asset in CMDB: %w", err)
	}

	// Update index.

	s.assetIndex.UpdateAsset(asset)

	// Create audit entry if there are changes.

	if len(changes) > 0 {

		auditEntry := &AuditEntry{
			ID: uuid.New().String(),

			Action: "update",

			Actor: "user", // This would come from context

			ResourceType: "asset",

			ResourceID: asset.ID,

			Changes: changes,

			Timestamp: time.Now(),

			Source: "api",
		}

		if err := s.cmdbStorage.CreateAuditEntry(ctx, auditEntry); err != nil {
			s.logger.Warn("failed to create audit entry", "error", err)
		}

	}

	s.logger.Info("updated asset", "asset_id", asset.ID, "changes", len(changes))

	return nil
}

// DeleteAsset deletes an asset from the inventory.

func (s *InventoryManagementService) DeleteAsset(ctx context.Context, id string) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Get asset before deletion for audit.

	asset, err := s.cmdbStorage.GetAsset(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get asset before deletion: %w", err)
	}

	// Delete from CMDB.

	if err := s.cmdbStorage.DeleteAsset(ctx, id); err != nil {
		return fmt.Errorf("failed to delete asset from CMDB: %w", err)
	}

	// Update index.

	s.assetIndex.RemoveAsset(id)

	// Create audit entry.

	auditEntry := &AuditEntry{
		ID: uuid.New().String(),

		Action: "delete",

		Actor: "user", // This would come from context

		ResourceType: "asset",

		ResourceID: id,

		Timestamp: time.Now(),

		Source: "api",
	}

	if err := s.cmdbStorage.CreateAuditEntry(ctx, auditEntry); err != nil {
		s.logger.Warn("failed to create audit entry", "error", err)
	}

	s.logger.Info("deleted asset", "asset_id", id, "name", asset.Name)

	return nil
}

// ListAssets lists assets based on filter criteria.

func (s *InventoryManagementService) ListAssets(ctx context.Context, filter *AssetFilter) ([]*Asset, error) {
	assets, err := s.cmdbStorage.ListAssets(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	// Apply sorting if specified.

	if filter.SortBy != "" {
		s.sortAssets(assets, filter.SortBy, filter.SortOrder)
	}

	return assets, nil
}

// GetAssetRelationships retrieves relationships for an asset.

func (s *InventoryManagementService) GetAssetRelationships(ctx context.Context, assetID string) ([]*AssetRelationship, error) {
	return s.cmdbStorage.GetAssetRelationships(ctx, assetID)
}

// CreateRelationship creates a new asset relationship.

func (s *InventoryManagementService) CreateRelationship(ctx context.Context, rel *AssetRelationship) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Set ID and timestamps.

	rel.ID = uuid.New().String()

	rel.CreatedAt = time.Now()

	rel.UpdatedAt = time.Now()

	// Store in CMDB.

	if err := s.cmdbStorage.CreateRelationship(ctx, rel); err != nil {
		return fmt.Errorf("failed to create relationship in CMDB: %w", err)
	}

	// Update index.

	s.relationshipIndex.AddRelationship(rel)

	s.logger.Info("created relationship",

		"relationship_id", rel.ID,

		"source", rel.SourceAssetID,

		"target", rel.TargetAssetID,

		"type", rel.RelationType)

	return nil
}

// GetComplianceStatus returns compliance status for an asset.

func (s *InventoryManagementService) GetComplianceStatus(ctx context.Context, assetID string) ([]*ComplianceCheck, error) {
	return s.cmdbStorage.GetComplianceStatus(ctx, assetID)
}

// GetAuditTrail returns audit trail for a resource.

func (s *InventoryManagementService) GetAuditTrail(ctx context.Context, resourceID string) ([]*AuditEntry, error) {
	return s.cmdbStorage.GetAuditTrail(ctx, resourceID)
}

// SyncInventory triggers manual inventory synchronization.

func (s *InventoryManagementService) SyncInventory(ctx context.Context) error {
	s.logger.Info("triggering manual inventory synchronization")

	go s.performSync()

	return nil
}

// DiscoverInfrastructure triggers manual infrastructure discovery for a provider.

func (s *InventoryManagementService) DiscoverInfrastructure(ctx context.Context, providerID string) error {
	provider, err := s.providerRegistry.GetProvider(providerID)
	if err != nil {
		return fmt.Errorf("provider %s not found: %w", providerID, err)
	}

	s.logger.Info("triggering manual infrastructure discovery",

		"provider", providerID)

	go s.discoverProviderAssets(ctx, provider)

	return nil
}

// Helper functions.

// trackAssetChanges tracks changes between existing and updated asset.

func (s *InventoryManagementService) trackAssetChanges(existing, updated *Asset) map[string]interface{} {
	changes := make(map[string]interface{})

	if existing.Name != updated.Name {
		changes["name"] = map[string]string{"from": existing.Name, "to": updated.Name}
	}

	if existing.Status != updated.Status {
		changes["status"] = map[string]string{"from": existing.Status, "to": updated.Status}
	}

	if existing.Health != updated.Health {
		changes["health"] = map[string]string{"from": existing.Health, "to": updated.Health}
	}

	if existing.State != updated.State {
		changes["state"] = map[string]string{"from": existing.State, "to": updated.State}
	}

	if !equalMaps(existing.Properties, updated.Properties) {
		changes["properties"] = json.RawMessage("{}")
	}

	if !equalStringMaps(existing.Tags, updated.Tags) {
		changes["tags"] = json.RawMessage("{}")
	}

	return changes
}

// sortAssets sorts assets based on the specified criteria.

func (s *InventoryManagementService) sortAssets(assets []*Asset, sortBy, sortOrder string) {
	sort.Slice(assets, func(i, j int) bool {
		var result bool

		switch sortBy {

		case "name":

			result = assets[i].Name < assets[j].Name

		case "type":

			result = assets[i].Type < assets[j].Type

		case "createdAt":

			result = assets[i].CreatedAt.Before(assets[j].CreatedAt)

		case "updatedAt":

			result = assets[i].UpdatedAt.Before(assets[j].UpdatedAt)

		default:

			result = assets[i].Name < assets[j].Name

		}

		if sortOrder == "DESC" {
			result = !result
		}

		return result
	})
}

// equalMaps compares two maps for equality.

func equalMaps(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	aJSON, _ := json.Marshal(a)

	bJSON, _ := json.Marshal(b)

	return bytes.Equal(aJSON, bJSON)
}

// equalStringMaps compares two string maps for equality.

func equalStringMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	aJSON, _ := json.Marshal(a)

	bJSON, _ := json.Marshal(b)

	return bytes.Equal(aJSON, bJSON)
}

// getProviderType helper is defined in infrastructure_monitoring.go.

// stubAssetStorage provides a basic stub implementation of AssetStorage.

type stubAssetStorage struct{}

// Store performs store operation.

func (s *stubAssetStorage) Store(ctx context.Context, asset *Asset) error {
	return nil
}

// Retrieve performs retrieve operation.

func (s *stubAssetStorage) Retrieve(ctx context.Context, id string) (*Asset, error) {
	return nil, fmt.Errorf("asset not found")
}

// List performs list operation.

func (s *stubAssetStorage) List(ctx context.Context, filter *AssetFilter) ([]*Asset, error) {
	return []*Asset{}, nil
}

// Delete performs delete operation.

func (s *stubAssetStorage) Delete(ctx context.Context, id string) error {
	return nil
}

// Stub implementations for DiscoveryEngine methods.

func (d *DiscoveryEngine) DiscoverAssets(ctx context.Context, provider providers.CloudProvider) ([]*Asset, error) {
	return []*Asset{}, nil
}

// Stub implementations for AssetIndex methods.

func (a *AssetIndex) AddAsset(asset *Asset) error {
	return nil
}

// UpdateAsset performs updateasset operation.

func (a *AssetIndex) UpdateAsset(asset *Asset) error {
	return nil
}

// Clear performs clear operation.

func (a *AssetIndex) Clear() error {
	return nil
}

// RemoveAsset performs removeasset operation.

func (a *AssetIndex) RemoveAsset(id string) error {
	return nil
}

// Stub implementations for RelationshipIndex methods.

func (r *RelationshipIndex) AddRelationship(relationship *AssetRelationship) error {
	return nil
}

// Clear performs clear operation.

func (r *RelationshipIndex) Clear() error {
	return nil
}

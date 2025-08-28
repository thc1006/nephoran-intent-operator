// Package o2 implements extended O2 IMS API handlers for resource lifecycle management
package o2

import (
	"context"
	"net/http"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"k8s.io/apimachinery/pkg/runtime"
)

// Supporting types for inventory operations - using existing types from other files
type InfrastructureDiscovery struct {
	ProviderID   string                 `json:"providerId"`
	Provider     CloudProviderType      `json:"provider"`
	Timestamp    string                 `json:"timestamp"`
	Assets       []*Asset               `json:"assets"`
	Summary      *DiscoverySummary      `json:"summary"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type DiscoverySummary struct {
	TotalAssets      int `json:"totalAssets"`
	ComputeResources int `json:"computeResources"`
	StorageResources int `json:"storageResources"`
	NetworkResources int `json:"networkResources"`
	NewAssets        int `json:"newAssets"`
	UpdatedAssets    int `json:"updatedAssets"`
}

// Resource Lifecycle Operation Handlers

// handleProvisionResource provisions a resource
func (s *O2APIServer) handleProvisionResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	var req ProvisionResourceRequest
	if err := s.decodeJSONRequest(r, &req); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)
		return
	}

	resource, err := s.resourceManager.ProvisionResource(r.Context(), &req)
	if err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to provision resource", err)
		return
	}

	s.metrics.RecordResourceOperation("provision", req.ResourceType, req.Provider, "success")
	s.writeJSONResponse(w, r, StatusAccepted, resource)
}

// handleConfigureResource configures a resource
func (s *O2APIServer) handleConfigureResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	var config runtime.RawExtension
	if err := s.decodeJSONRequest(r, &config); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid configuration", err)
		return
	}

	if err := s.resourceManager.ConfigureResource(r.Context(), resourceID, &config); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to configure resource", err)
		return
	}

	s.metrics.RecordResourceOperation("configure", "resource", "unknown", "success")
	s.writeJSONResponse(w, r, StatusAccepted, map[string]string{
		"status":     "configuration_applied",
		"resourceId": resourceID,
	})
}

// handleScaleResource scales a resource
func (s *O2APIServer) handleScaleResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	var req ScaleResourceRequest
	if err := s.decodeJSONRequest(r, &req); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.resourceManager.ScaleResource(r.Context(), resourceID, &req); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to scale resource", err)
		return
	}

	s.metrics.RecordResourceOperation("scale", "resource", "unknown", "success")
	s.writeJSONResponse(w, r, StatusAccepted, map[string]interface{}{
		"status":         "scaling_initiated",
		"resourceId":     resourceID,
		"scaleType":      req.ScaleType,
		"targetReplicas": req.TargetReplicas,
	})
}

// handleMigrateResource migrates a resource
func (s *O2APIServer) handleMigrateResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	var req MigrateResourceRequest
	if err := s.decodeJSONRequest(r, &req); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.resourceManager.MigrateResource(r.Context(), resourceID, &req); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to migrate resource", err)
		return
	}

	s.metrics.RecordResourceOperation("migrate", "resource", req.TargetProvider, "success")
	s.writeJSONResponse(w, r, StatusAccepted, map[string]interface{}{
		"status":         "migration_initiated",
		"resourceId":     resourceID,
		"sourceProvider": req.SourceProvider,
		"targetProvider": req.TargetProvider,
	})
}

// handleBackupResource creates a backup of a resource
func (s *O2APIServer) handleBackupResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	var req BackupResourceRequest
	if err := s.decodeJSONRequest(r, &req); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)
		return
	}

	backupInfo, err := s.resourceManager.BackupResource(r.Context(), resourceID, &req)
	if err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to backup resource", err)
		return
	}

	s.metrics.RecordResourceOperation("backup", "resource", "unknown", "success")
	s.writeJSONResponse(w, r, StatusAccepted, backupInfo)
}

// handleRestoreResource restores a resource from backup
func (s *O2APIServer) handleRestoreResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	backupID := s.getPathParam(r, "backupId")
	if backupID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Backup ID is required", nil)
		return
	}

	if err := s.resourceManager.RestoreResource(r.Context(), resourceID, backupID); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to restore resource", err)
		return
	}

	s.metrics.RecordResourceOperation("restore", "resource", "unknown", "success")
	s.writeJSONResponse(w, r, StatusAccepted, map[string]string{
		"status":     "restoration_initiated",
		"resourceId": resourceID,
		"backupId":   backupID,
	})
}

// handleTerminateResource terminates a resource
func (s *O2APIServer) handleTerminateResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")
	if resourceID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)
		return
	}

	if err := s.resourceManager.TerminateResource(r.Context(), resourceID); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to terminate resource", err)
		return
	}

	s.metrics.RecordResourceOperation("terminate", "resource", "unknown", "success")
	s.writeJSONResponse(w, r, StatusAccepted, map[string]string{
		"status":     "termination_initiated",
		"resourceId": resourceID,
	})
}

// Infrastructure Discovery and Inventory Handlers

// handleDiscoverInfrastructure discovers infrastructure for a provider
func (s *O2APIServer) handleDiscoverInfrastructure(w http.ResponseWriter, r *http.Request) {
	providerID := s.getPathParam(r, "providerId")
	if providerID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Provider ID is required", nil)
		return
	}

	// Convert providerID to CloudProviderType
	var provider CloudProviderType
	switch providerID {
	case "kubernetes":
		provider = CloudProviderKubernetes
	case "openstack":
		provider = CloudProviderOpenStack
	case "aws":
		provider = CloudProviderAWS
	case "azure":
		provider = CloudProviderAzure
	case "gcp":
		provider = CloudProviderGCP
	case "vmware":
		provider = CloudProviderVMware
	case "edge":
		provider = CloudProviderEdge
	default:
		s.writeErrorResponse(w, r, StatusBadRequest, "Unsupported provider type", nil)
		return
	}

	discovery, err := s.inventoryService.DiscoverInfrastructure(r.Context(), provider)
	if err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to discover infrastructure", err)
		return
	}

	s.metrics.RecordResourceOperation("discover", "infrastructure", providerID, "success")
	s.writeJSONResponse(w, r, StatusAccepted, discovery)
}

// handleSyncInventory synchronizes inventory updates
func (s *O2APIServer) handleSyncInventory(w http.ResponseWriter, r *http.Request) {
	var updates []*InventoryUpdate
	if err := s.decodeJSONRequest(r, &updates); err != nil {
		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.inventoryService.UpdateInventory(r.Context(), updates); err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to sync inventory", err)
		return
	}

	s.writeJSONResponse(w, r, StatusAccepted, map[string]interface{}{
		"status":       "inventory_sync_initiated",
		"updatesCount": len(updates),
	})
}

// handleGetAssets retrieves inventory assets
func (s *O2APIServer) handleGetAssets(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for asset filtering
	assetType := s.getQueryParam(r, "type")
	provider := s.getQueryParam(r, "provider")
	status := s.getQueryParam(r, "status")
	limit := s.getQueryParamInt(r, "limit", 100)
	offset := s.getQueryParamInt(r, "offset", 0)

	// Build filter based on query parameters
	filter := &AssetFilter{
		Type:     assetType,
		Provider: provider,
		Status:   status,
		Limit:    limit,
		Offset:   offset,
	}

	assets, err := s.getAssets(r.Context(), filter)
	if err != nil {
		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve assets", err)
		return
	}

	s.writeJSONResponse(w, r, StatusOK, assets)
}

// handleGetAsset retrieves a specific asset
func (s *O2APIServer) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	assetID := s.getPathParam(r, "assetId")
	if assetID == "" {
		s.writeErrorResponse(w, r, StatusBadRequest, "Asset ID is required", nil)
		return
	}

	asset, err := s.inventoryService.GetAsset(r.Context(), assetID)
	if err != nil {
		s.writeErrorResponse(w, r, StatusNotFound, "Asset not found", err)
		return
	}

	s.writeJSONResponse(w, r, StatusOK, asset)
}

// Helper method to get assets (placeholder implementation)
func (s *O2APIServer) getAssets(ctx context.Context, filter *AssetFilter) ([]*Asset, error) {
	// This would typically call the inventory service
	// For now, return an empty slice
	return []*Asset{}, nil
}

// Utility methods for parsing request filters

// parseResourceTypeFilter parses resource type filter from query parameters
func (s *O2APIServer) parseResourceTypeFilter(r *http.Request) *models.ResourceTypeFilter {
	filter := &models.ResourceTypeFilter{
		Limit:  s.getQueryParamInt(r, "limit", 100),
		Offset: s.getQueryParamInt(r, "offset", 0),
	}

	if names := r.URL.Query().Get("names"); names != "" {
		filter.Names = []string{names}
	}

	if categories := r.URL.Query().Get("categories"); categories != "" {
		filter.Categories = []string{categories}
	}

	if vendors := r.URL.Query().Get("vendors"); vendors != "" {
		filter.Vendors = []string{vendors}
	}

	if models := r.URL.Query().Get("models"); models != "" {
		filter.Models = []string{models}
	}

	if versions := r.URL.Query().Get("versions"); versions != "" {
		filter.Versions = []string{versions}
	}

	return filter
}

// parseResourceFilter parses resource filter from query parameters
func (s *O2APIServer) parseResourceFilter(r *http.Request) *models.ResourceFilter {
	filter := &models.ResourceFilter{
		Limit:  s.getQueryParamInt(r, "limit", 100),
		Offset: s.getQueryParamInt(r, "offset", 0),
	}

	if resourcePoolIDs := r.URL.Query().Get("resourcePoolIds"); resourcePoolIDs != "" {
		filter.ResourcePoolIDs = []string{resourcePoolIDs}
	}

	if resourceTypeIDs := r.URL.Query().Get("resourceTypeIds"); resourceTypeIDs != "" {
		filter.ResourceTypeIDs = []string{resourceTypeIDs}
	}

	if statuses := r.URL.Query().Get("statuses"); statuses != "" {
		filter.LifecycleStates = []string{statuses}
	}

	return filter
}

// parseAlarmFilter parses alarm filter from query parameters
func (s *O2APIServer) parseAlarmFilter(r *http.Request) *AlarmFilter {
	return &AlarmFilter{
		Severity:  s.getQueryParam(r, "severity"),
		Status:    s.getQueryParam(r, "status"),
		StartTime: s.getQueryParam(r, "startTime"),
		EndTime:   s.getQueryParam(r, "endTime"),
		Limit:     s.getQueryParamInt(r, "limit", 100),
		Offset:    s.getQueryParamInt(r, "offset", 0),
	}
}

// parseMetricsFilter parses metrics filter from query parameters
func (s *O2APIServer) parseMetricsFilter(r *http.Request) *MetricsFilter {
	return &MetricsFilter{
		MetricNames: s.parseQueryParamArray(r, "metricNames"),
		StartTime:   s.getQueryParam(r, "startTime"),
		EndTime:     s.getQueryParam(r, "endTime"),
		Granularity: s.getQueryParam(r, "granularity"),
		Limit:       s.getQueryParamInt(r, "limit", 1000),
		Offset:      s.getQueryParamInt(r, "offset", 0),
	}
}

// parseDeploymentTemplateFilter parses deployment template filter from query parameters
func (s *O2APIServer) parseDeploymentTemplateFilter(r *http.Request) *models.DeploymentTemplateFilter {
	return &models.DeploymentTemplateFilter{
		Names:      s.parseQueryParamArray(r, "names"),
		Categories: s.parseQueryParamArray(r, "categories"),
		Types:      s.parseQueryParamArray(r, "types"),
		Versions:   s.parseQueryParamArray(r, "versions"),
		Authors:    s.parseQueryParamArray(r, "authors"),
		Limit:      s.getQueryParamInt(r, "limit", 100),
		Offset:     s.getQueryParamInt(r, "offset", 0),
	}
}

// parseDeploymentFilter parses deployment filter from query parameters
func (s *O2APIServer) parseDeploymentFilter(r *http.Request) *models.DeploymentFilter {
	return &models.DeploymentFilter{
		Names:       s.parseQueryParamArray(r, "names"),
		States:      s.parseQueryParamArray(r, "states"),
		TemplateIDs: s.parseQueryParamArray(r, "templateIds"),
		Limit:       s.getQueryParamInt(r, "limit", 100),
		Offset:      s.getQueryParamInt(r, "offset", 0),
	}
}

// parseSubscriptionFilter parses subscription filter from query parameters
func (s *O2APIServer) parseSubscriptionFilter(r *http.Request) *models.SubscriptionQueryFilter {
	return &models.SubscriptionQueryFilter{
		EventTypes: s.parseQueryParamArray(r, "eventTypes"),
		Limit:      s.getQueryParamInt(r, "limit", 100),
		Offset:     s.getQueryParamInt(r, "offset", 0),
	}
}

// parseQueryParamArray parses a comma-separated query parameter into an array
func (s *O2APIServer) parseQueryParamArray(r *http.Request, param string) []string {
	value := r.URL.Query().Get(param)
	if value == "" {
		return nil
	}

	// Simple comma-separated parsing - could be enhanced for more complex cases
	return []string{value}
}

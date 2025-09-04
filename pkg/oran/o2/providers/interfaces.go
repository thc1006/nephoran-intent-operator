package providers

import (
	"context"
)

// Provider defines the core interface for O-RAN O2 IMS providers.
// Providers implement infrastructure management capabilities following
// O-RAN O2 specifications for cloud resource lifecycle management.
type Provider interface {
	// GetInfo returns metadata about this provider implementation
	GetInfo() ProviderInfo

	// Initialize configures the provider with given configuration
	Initialize(ctx context.Context, config ProviderConfig) error

	// CreateResource creates a new resource
	CreateResource(ctx context.Context, req ResourceRequest) (*Resource, error)

	// GetResource retrieves a resource by ID
	GetResource(ctx context.Context, id string) (*Resource, error)

	// ListResources returns resources matching the filter criteria
	ListResources(ctx context.Context, filter ResourceFilter) ([]*Resource, error)

	// UpdateResource updates an existing resource
	UpdateResource(ctx context.Context, id string, req ResourceRequest) (*Resource, error)

	// DeleteResource removes a resource
	DeleteResource(ctx context.Context, id string) error

	// GetResourceStatus returns the current status of a resource
	GetResourceStatus(ctx context.Context, id string) (ResourceStatus, error)

	// Close cleans up provider resources
	Close() error
}

// ClusterProvider extends Provider with cluster-specific operations
// for managing Kubernetes clusters and O-Cloud infrastructure.
type ClusterProvider interface {
	Provider

	// CreateCluster creates a new Kubernetes cluster
	CreateCluster(ctx context.Context, spec ClusterSpec) (*ClusterResource, error)

	// ScaleCluster scales cluster nodes
	ScaleCluster(ctx context.Context, clusterID string, nodeCount int) error

	// GetClusterConfig returns kubeconfig for accessing the cluster
	GetClusterConfig(ctx context.Context, clusterID string) ([]byte, error)

	// UpgradeCluster upgrades cluster to specified version
	UpgradeCluster(ctx context.Context, clusterID string, version string) error
}

// NetworkProvider extends Provider with network-specific operations
// for managing O-RAN network functions and connectivity.
type NetworkProvider interface {
	Provider

	// CreateNetwork creates a virtual network
	CreateNetwork(ctx context.Context, spec NetworkSpec) (*NetworkResource, error)

	// AttachToNetwork connects a resource to a network
	AttachToNetwork(ctx context.Context, networkID, resourceID string) error

	// DetachFromNetwork disconnects a resource from a network
	DetachFromNetwork(ctx context.Context, networkID, resourceID string) error

	// GetNetworkTopology returns network topology information
	GetNetworkTopology(ctx context.Context, networkID string) (*NetworkTopology, error)
}

// StorageProvider extends Provider with storage-specific operations
// for managing persistent volumes and data services.
type StorageProvider interface {
	Provider

	// CreateVolume creates a persistent volume
	CreateVolume(ctx context.Context, spec VolumeSpec) (*VolumeResource, error)

	// AttachVolume attaches volume to a compute resource
	AttachVolume(ctx context.Context, volumeID, nodeID string) error

	// DetachVolume detaches volume from a compute resource
	DetachVolume(ctx context.Context, volumeID, nodeID string) error

	// CreateSnapshot creates a snapshot of a volume
	CreateSnapshot(ctx context.Context, volumeID, snapshotName string) (*SnapshotResource, error)
}

// MonitoringProvider extends Provider with observability operations
// for collecting metrics, logs, and traces from managed resources.
type MonitoringProvider interface {
	Provider

	// GetMetrics retrieves metrics for a resource
	GetMetrics(ctx context.Context, resourceID string, query MetricsQuery) (*MetricsResult, error)

	// GetLogs retrieves logs for a resource
	GetLogs(ctx context.Context, resourceID string, query LogsQuery) (*LogsResult, error)

	// CreateAlert creates a monitoring alert
	CreateAlert(ctx context.Context, spec AlertSpec) (*AlertResource, error)

	// GetResourceHealth returns health status of a resource
	GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error)
}

// EventHandler handles provider events and notifications
type EventHandler interface {
	// HandleResourceEvent is called when resource state changes
	HandleResourceEvent(ctx context.Context, event ResourceEvent) error

	// HandleProviderEvent is called for provider-level events
	HandleProviderEvent(ctx context.Context, event ProviderEvent) error

	// HandleEvent is called for generic events (for compatibility)
	HandleEvent(event *ProviderEvent)
}

// ProviderFactory creates provider instances
type ProviderFactory interface {
	// CreateProvider creates a new provider instance
	CreateProvider(providerType string, config ProviderConfig) (Provider, error)

	// ListSupportedTypes returns supported provider types
	ListSupportedTypes() []string

	// GetProviderSchema returns configuration schema for a provider type
	GetProviderSchema(providerType string) (map[string]interface{}, error)

	// RegisterProvider registers a new provider type with constructor and schema
	RegisterProvider(providerType string, constructor ProviderConstructor, schema map[string]interface{}) error
}

// ProviderRegistry manages multiple provider instances
type ProviderRegistry interface {
	// RegisterProvider registers a provider with configuration
	RegisterProvider(name string, provider CloudProvider, config *ProviderConfiguration) error

	// UnregisterProvider removes a provider
	UnregisterProvider(name string) error

	// GetProvider retrieves a provider by name
	GetProvider(name string) (CloudProvider, error)

	// ListProviders returns all registered provider names
	ListProviders() []string

	// ConnectAll connects to all registered providers
	ConnectAll(ctx context.Context) error

	// ConnectProvider connects to a specific provider
	ConnectProvider(ctx context.Context, name string) error

	// DisconnectAll disconnects from all providers
	DisconnectAll(ctx context.Context) error

	// StartHealthChecks starts health checking for all providers
	StartHealthChecks(ctx context.Context)

	// StopHealthChecks stops health checking
	StopHealthChecks()

	// GetAllProviderHealth returns health status for all providers
	GetAllProviderHealth() map[string]*HealthStatus

	// SelectProvider selects the best provider based on criteria
	SelectProvider(ctx context.Context, criteria *ProviderSelectionCriteria) (CloudProvider, error)

	// GetSupportedProviders returns the list of supported provider types
	GetSupportedProviders() []string

	// CreateAndRegisterProvider creates and registers a provider
	CreateAndRegisterProvider(providerType string, name string, config *ProviderConfiguration) error

	// Close closes all providers and cleans up resources
	Close() error
}

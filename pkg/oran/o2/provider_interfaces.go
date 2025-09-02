package o2

import (
	
	"encoding/json"
"context"
	"time"
)

// CloudProvider constants for provider types
const (
	CloudProviderKubernetes = "kubernetes"
	CloudProviderOpenStack  = "openstack"
	CloudProviderAWS        = "aws"
	CloudProviderAzure      = "azure"
	CloudProviderGCP        = "gcp"
	CloudProviderVMware     = "vmware"
	CloudProviderEdge       = "edge"
)

// CloudProviderType represents different cloud provider types as a string
type CloudProviderType string

// CloudProvider interface defines the contract for cloud infrastructure providers
type CloudProvider interface {
	// Provider metadata
	GetProviderInfo() *ProviderInfo
	GetCapabilities() *ProviderCapabilities

	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	HealthCheck(ctx context.Context) error

	// Resource lifecycle
	CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error)
	GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error)
	UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error)
	DeleteResource(ctx context.Context, resourceID string) error
	ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error)
}

// EventCallback defines the callback function for provider events
type EventCallback func(event *ProviderEvent)

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Region      string            `json:"region,omitempty"`
	Zone        string            `json:"zone,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ProviderCapabilities defines what capabilities a provider supports
type ProviderCapabilities struct {
	SupportedResourceTypes []string          `json:"supportedResourceTypes"`
	Features               []string          `json:"features"`
	Limits                 map[string]int    `json:"limits,omitempty"`
	Configuration          map[string]string `json:"configuration,omitempty"`
}

// CreateResourceRequest represents a request to create a new resource
type CreateResourceRequest struct {
	ResourceType string                 `json:"resourceType"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	Spec         json.RawMessage `json:"spec"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	DryRun       bool                   `json:"dryRun,omitempty"`
}

// UpdateResourceRequest represents a request to update a resource
type UpdateResourceRequest struct {
	Spec        json.RawMessage `json:"spec"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	DryRun      bool                   `json:"dryRun,omitempty"`
}

// ResourceResponse represents the response containing resource information
type ResourceResponse struct {
	ResourceID   string                 `json:"resourceId"`
	ResourceType string                 `json:"resourceType"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	Status       string                 `json:"status"`
	Phase        string                 `json:"phase,omitempty"`
	Spec         json.RawMessage `json:"spec"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Events       []*ResourceEvent       `json:"events,omitempty"`
}

// ResourceFilter defines filtering criteria for listing resources
type ResourceFilter struct {
	ResourceTypes []string          `json:"resourceTypes,omitempty"`
	Namespaces    []string          `json:"namespaces,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Status        []string          `json:"status,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
}

// ProviderEvent represents an event from a provider
type ProviderEvent struct {
	EventID    string                 `json:"eventId"`
	ProviderID string                 `json:"providerId"`
	ResourceID string                 `json:"resourceId,omitempty"`
	EventType  string                 `json:"eventType"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// Resource status constants
const (
	ResourceStatusPending  = "PENDING"
	ResourceStatusCreating = "CREATING"
	ResourceStatusActive   = "ACTIVE"
	ResourceStatusUpdating = "UPDATING"
	ResourceStatusDeleting = "DELETING"
	ResourceStatusFailed   = "FAILED"
	ResourceStatusUnknown  = "UNKNOWN"
)

// Event type constants
const (
	EventTypeResourceCreated   = "RESOURCE_CREATED"
	EventTypeResourceUpdated   = "RESOURCE_UPDATED"
	EventTypeResourceDeleted   = "RESOURCE_DELETED"
	EventTypeResourceFailed    = "RESOURCE_FAILED"
	EventTypeProviderConnected = "PROVIDER_CONNECTED"
	EventTypeProviderFailed    = "PROVIDER_FAILED"
)

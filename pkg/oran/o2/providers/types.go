// Package providers implements O-RAN O2 IMS (Infrastructure Management Services) providers
// for managing cloud infrastructure resources following O-RAN O2 specifications.
package providers

import (
	"time"
)

// ResourceType represents the type of O2 resource
type ResourceType string

const (
	// Infrastructure resource types
	ResourceTypeCluster ResourceType = "cluster"
	ResourceTypeNode    ResourceType = "node"

	// Application resource types
	ResourceTypeDeployment ResourceType = "deployment"
	ResourceTypeService    ResourceType = "service"
	ResourceTypeConfigMap  ResourceType = "configmap"
	ResourceTypeSecret     ResourceType = "secret"
)

// ResourceStatus represents the operational state of a resource
type ResourceStatus string

const (
	StatusPending  ResourceStatus = "pending"
	StatusCreating ResourceStatus = "creating"
	StatusReady    ResourceStatus = "ready"
	StatusUpdating ResourceStatus = "updating"
	StatusDeleting ResourceStatus = "deleting"
	StatusError    ResourceStatus = "error"
	StatusUnknown  ResourceStatus = "unknown"
)

// Resource represents a generic O2 IMS resource
type Resource struct {
	// Unique identifier
	ID string `json:"id"`

	// Resource name
	Name string `json:"name"`

	// Resource type
	Type ResourceType `json:"type"`

	// Current status
	Status ResourceStatus `json:"status"`

	// Resource specification (varies by type)
	Spec map[string]interface{} `json:"spec"`

	// Current state/configuration
	State map[string]interface{} `json:"state,omitempty"`

	// Metadata and labels
	Labels map[string]string `json:"labels,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ResourceRequest represents a request to create or update a resource
type ResourceRequest struct {
	Name   string                 `json:"name"`
	Type   ResourceType           `json:"type"`
	Spec   map[string]interface{} `json:"spec"`
	Labels map[string]string      `json:"labels,omitempty"`
}



// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Credentials map[string]string      `json:"credentials,omitempty"`
}

// OperationResult represents the result of a provider operation
type OperationResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

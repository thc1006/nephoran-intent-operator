package o1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
)

// NetworkFunctionManager defines the interface for managing network functions
type NetworkFunctionManager interface {
	// Core Management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetFunctionCount() int

	// Network Function Lifecycle
	RegisterNetworkFunction(ctx context.Context, nf *NetworkFunction) error
	DeregisterNetworkFunction(ctx context.Context, nfID string) error
	UpdateNetworkFunction(ctx context.Context, nfID string, updates *NetworkFunctionUpdate) error

	// Discovery and Status
	DiscoverNetworkFunctions(ctx context.Context, criteria *DiscoveryCriteria) ([]*NetworkFunction, error)
	GetNetworkFunctionStatus(ctx context.Context, nfID string) (*NetworkFunctionStatus, error)

	// Configuration Management
	ConfigureNetworkFunction(ctx context.Context, nfID string, config *NetworkFunctionConfig) error
	GetNetworkFunctionConfiguration(ctx context.Context, nfID string) (*NetworkFunctionConfig, error)
}

// NetworkFunctionConfig represents configuration details for a network function
type NetworkFunctionConfig struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name,omitempty"`
	Parameters         map[string]interface{} `json:"parameters,omitempty"`
	ConfigurationType  string                 `json:"configuration_type,omitempty"`
	ConfigurationState string                 `json:"configuration_state,omitempty"`
	Version            string                 `json:"version,omitempty"`
	Timestamp          time.Time              `json:"timestamp,omitempty"`
	AdditionalDetails  map[string]interface{} `json:"additional_details,omitempty"`
}

// NetworkFunction represents a network function in the O-RAN system
type NetworkFunction struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Vendor      string                 `json:"vendor,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name      *string                `json:"name,omitempty"`
	Status    *string                `json:"status,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NetworkFunctionStatus represents the current status of a network function
type NetworkFunctionStatus struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Health    string                 `json:"health"`
	Uptime    time.Duration          `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	LastCheck time.Time              `json:"last_check"`
}

// DiscoveryCriteria represents criteria for discovering network functions
type DiscoveryCriteria struct {
	Type     string            `json:"type,omitempty"`
	Vendor   string            `json:"vendor,omitempty"`
	Status   string            `json:"status,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	MaxCount int               `json:"max_count,omitempty"`
}
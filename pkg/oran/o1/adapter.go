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

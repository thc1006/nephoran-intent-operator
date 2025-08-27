package providers

import (
	"context"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// Region represents a cloud provider region
type Region struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Status   string `json:"status,omitempty"`
}

// AvailabilityZone represents an availability zone within a region
type AvailabilityZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Status string `json:"status"`
}

// InstanceType represents a compute instance type
type InstanceType struct {
	Name         string  `json:"name"`
	CPU          string  `json:"cpu"`
	Memory       string  `json:"memory"`
	Storage      string  `json:"storage,omitempty"`
	Network      string  `json:"network,omitempty"`
	GPU          string  `json:"gpu,omitempty"`
	PricePerHour float64 `json:"pricePerHour,omitempty"`
	Description  string  `json:"description,omitempty"`
}

// QuotaInfo represents quota information for a region
type QuotaInfo struct {
	ComputeInstances int    `json:"computeInstances"`
	VCPUs            int    `json:"vcpus"`
	Memory           string `json:"memory"`
	Storage          string `json:"storage"`
	Networks         int    `json:"networks,omitempty"`
	SecurityGroups   int    `json:"securityGroups,omitempty"`
	FloatingIPs      int    `json:"floatingIPs,omitempty"`
}

// CreateResourcePoolRequest represents a request to create a resource pool
type CreateResourcePoolRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Region      string                   `json:"region"`
	Zone        string                   `json:"zone,omitempty"`
	Provider    string                   `json:"provider"`
	Capacity    *models.ResourceCapacity `json:"capacity,omitempty"`
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
	Properties  map[string]interface{}   `json:"properties,omitempty"`
}

// UpdateResourcePoolRequest represents a request to update a resource pool
type UpdateResourcePoolRequest struct {
	Name        string                   `json:"name,omitempty"`
	Description string                   `json:"description,omitempty"`
	Capacity    *models.ResourceCapacity `json:"capacity,omitempty"`
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
	Properties  map[string]interface{}   `json:"properties,omitempty"`
}

// ResourcePoolFilter represents filters for resource pool queries
type ResourcePoolFilter struct {
	Names       []string          `json:"names,omitempty"`
	Regions     []string          `json:"regions,omitempty"`
	Zones       []string          `json:"zones,omitempty"`
	Providers   []string          `json:"providers,omitempty"`
	States      []string          `json:"states,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
}

// CreateComputeInstanceRequest represents a request to create a compute instance
type CreateComputeInstanceRequest struct {
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	InstanceType   string            `json:"instanceType"`
	Image          string            `json:"image"`
	Region         string            `json:"region"`
	Zone           string            `json:"zone,omitempty"`
	ResourcePoolID string            `json:"resourcePoolId"`
	NetworkConfig  *NetworkConfig    `json:"networkConfig,omitempty"`
	StorageConfig  *StorageConfig    `json:"storageConfig,omitempty"`
	SecurityGroups []string          `json:"securityGroups,omitempty"`
	UserData       string            `json:"userData,omitempty"`
	KeyPair        string            `json:"keyPair,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// NetworkConfig represents network configuration for instances
type NetworkConfig struct {
	SubnetID       string   `json:"subnetId,omitempty"`
	SecurityGroups []string `json:"securityGroups,omitempty"`
	PublicIP       bool     `json:"publicIP,omitempty"`
	PrivateIP      string   `json:"privateIP,omitempty"`
	FloatingIP     string   `json:"floatingIP,omitempty"`
}

// StorageConfig represents storage configuration for instances
type StorageConfig struct {
	RootVolumeSize    int                `json:"rootVolumeSize,omitempty"`
	RootVolumeType    string             `json:"rootVolumeType,omitempty"`
	AdditionalVolumes []AdditionalVolume `json:"additionalVolumes,omitempty"`
	EncryptionConfig  *EncryptionConfig  `json:"encryptionConfig,omitempty"`
}

// AdditionalVolume represents additional storage volumes
type AdditionalVolume struct {
	Size       int    `json:"size"`
	Type       string `json:"type"`
	MountPoint string `json:"mountPoint"`
	Encrypted  bool   `json:"encrypted,omitempty"`
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	Enabled   bool   `json:"enabled"`
	KeyID     string `json:"keyId,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
}

// CloudProviderInterface defines the common interface for all cloud providers
type CloudProviderInterface interface {
	// Provider identification
	GetProviderType() string

	// Connection management
	Initialize(ctx context.Context, config map[string]interface{}) error
	ValidateCredentials(ctx context.Context) error

	// Region and zone management
	GetRegions(ctx context.Context) ([]Region, error)
	GetAvailabilityZones(ctx context.Context, region string) ([]AvailabilityZone, error)

	// Instance type management
	GetInstanceTypes(ctx context.Context, region string) ([]InstanceType, error)

	// Quota management
	GetQuotas(ctx context.Context, region string) (*QuotaInfo, error)

	// Resource pool management
	CreateResourcePool(ctx context.Context, req *CreateResourcePoolRequest) (*models.ResourcePool, error)
	GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error)
	UpdateResourcePool(ctx context.Context, poolID string, req *UpdateResourcePoolRequest) (*models.ResourcePool, error)
	DeleteResourcePool(ctx context.Context, poolID string) error
	ListResourcePools(ctx context.Context, filter *ResourcePoolFilter) ([]*models.ResourcePool, error)

	// Compute instance management
	CreateComputeInstance(ctx context.Context, req *CreateComputeInstanceRequest) (*models.ResourceInstance, error)
	GetComputeInstance(ctx context.Context, instanceID string) (*models.ResourceInstance, error)
	DeleteComputeInstance(ctx context.Context, instanceID string) error

	// Monitoring
	GetResourceMetrics(ctx context.Context, resourceID string) (*models.ResourceMetrics, error)
}


package mocks



import (

	"context"

	"fmt"

	"time"



	"github.com/stretchr/testify/mock"



	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"

)



// MockAWSProvider provides mock AWS cloud provider functionality.

type MockAWSProvider struct {

	mock.Mock

}



// GetProviderType performs getprovidertype operation.

func (m *MockAWSProvider) GetProviderType() string {

	return "aws"

}



// Initialize performs initialize operation.

func (m *MockAWSProvider) Initialize(ctx context.Context, config map[string]interface{}) error {

	args := m.Called(ctx, config)

	return args.Error(0)

}



// GetRegions performs getregions operation.

func (m *MockAWSProvider) GetRegions(ctx context.Context) ([]providers.Region, error) {

	args := m.Called(ctx)

	return args.Get(0).([]providers.Region), args.Error(1)

}



// GetAvailabilityZones performs getavailabilityzones operation.

func (m *MockAWSProvider) GetAvailabilityZones(ctx context.Context, region string) ([]providers.AvailabilityZone, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.AvailabilityZone), args.Error(1)

}



// CreateResourcePool performs createresourcepool operation.

func (m *MockAWSProvider) CreateResourcePool(ctx context.Context, req *providers.CreateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// DeleteResourcePool performs deleteresourcepool operation.

func (m *MockAWSProvider) DeleteResourcePool(ctx context.Context, poolID string) error {

	args := m.Called(ctx, poolID)

	return args.Error(0)

}



// UpdateResourcePool performs updateresourcepool operation.

func (m *MockAWSProvider) UpdateResourcePool(ctx context.Context, poolID string, req *providers.UpdateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// GetResourcePool performs getresourcepool operation.

func (m *MockAWSProvider) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// ListResourcePools performs listresourcepools operation.

func (m *MockAWSProvider) ListResourcePools(ctx context.Context, filter *providers.ResourcePoolFilter) ([]*models.ResourcePool, error) {

	args := m.Called(ctx, filter)

	return args.Get(0).([]*models.ResourcePool), args.Error(1)

}



// CreateComputeInstance performs createcomputeinstance operation.

func (m *MockAWSProvider) CreateComputeInstance(ctx context.Context, req *providers.CreateComputeInstanceRequest) (*models.ResourceInstance, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// DeleteComputeInstance performs deletecomputeinstance operation.

func (m *MockAWSProvider) DeleteComputeInstance(ctx context.Context, instanceID string) error {

	args := m.Called(ctx, instanceID)

	return args.Error(0)

}



// GetComputeInstance performs getcomputeinstance operation.

func (m *MockAWSProvider) GetComputeInstance(ctx context.Context, instanceID string) (*models.ResourceInstance, error) {

	args := m.Called(ctx, instanceID)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// GetInstanceTypes performs getinstancetypes operation.

func (m *MockAWSProvider) GetInstanceTypes(ctx context.Context, region string) ([]providers.InstanceType, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.InstanceType), args.Error(1)

}



// GetResourceMetrics performs getresourcemetrics operation.

func (m *MockAWSProvider) GetResourceMetrics(ctx context.Context, resourceID string) (*models.ResourceMetrics, error) {

	args := m.Called(ctx, resourceID)

	return args.Get(0).(*models.ResourceMetrics), args.Error(1)

}



// ValidateCredentials performs validatecredentials operation.

func (m *MockAWSProvider) ValidateCredentials(ctx context.Context) error {

	args := m.Called(ctx)

	return args.Error(0)

}



// GetQuotas performs getquotas operation.

func (m *MockAWSProvider) GetQuotas(ctx context.Context, region string) (*providers.QuotaInfo, error) {

	args := m.Called(ctx, region)

	return args.Get(0).(*providers.QuotaInfo), args.Error(1)

}



// NewMockAWSProvider creates a new mock AWS provider with default behaviors.

func NewMockAWSProvider() *MockAWSProvider {

	provider := &MockAWSProvider{}



	// Setup default behaviors.

	provider.On("GetRegions", mock.Anything).Return([]providers.Region{

		{ID: "us-east-1", Name: "US East (N. Virginia)", Location: "Virginia, USA"},

		{ID: "us-west-2", Name: "US West (Oregon)", Location: "Oregon, USA"},

		{ID: "eu-west-1", Name: "Europe (Ireland)", Location: "Dublin, Ireland"},

	}, nil)



	provider.On("GetAvailabilityZones", mock.Anything, "us-east-1").Return([]providers.AvailabilityZone{

		{ID: "us-east-1a", Name: "us-east-1a", Region: "us-east-1", Status: "available"},

		{ID: "us-east-1b", Name: "us-east-1b", Region: "us-east-1", Status: "available"},

		{ID: "us-east-1c", Name: "us-east-1c", Region: "us-east-1", Status: "available"},

	}, nil)



	provider.On("GetInstanceTypes", mock.Anything, mock.AnythingOfType("string")).Return([]providers.InstanceType{

		{

			Name:         "c5.large",

			CPU:          "2",

			Memory:       "4Gi",

			Network:      "up to 10 Gbps",

			PricePerHour: 0.096,

		},

		{

			Name:         "c5.xlarge",

			CPU:          "4",

			Memory:       "8Gi",

			Network:      "up to 10 Gbps",

			PricePerHour: 0.192,

		},

		{

			Name:         "c5.2xlarge",

			CPU:          "8",

			Memory:       "16Gi",

			Network:      "up to 10 Gbps",

			PricePerHour: 0.384,

		},

	}, nil)



	provider.On("ValidateCredentials", mock.Anything).Return(nil)



	provider.On("GetQuotas", mock.Anything, mock.AnythingOfType("string")).Return(&providers.QuotaInfo{

		ComputeInstances: 100,

		VCPUs:            200,

		Memory:           "800Gi",

		Storage:          "10Ti",

	}, nil)



	return provider

}



// MockAzureProvider provides mock Azure cloud provider functionality.

type MockAzureProvider struct {

	mock.Mock

}



// GetProviderType performs getprovidertype operation.

func (m *MockAzureProvider) GetProviderType() string {

	return "azure"

}



// Initialize performs initialize operation.

func (m *MockAzureProvider) Initialize(ctx context.Context, config map[string]interface{}) error {

	args := m.Called(ctx, config)

	return args.Error(0)

}



// GetRegions performs getregions operation.

func (m *MockAzureProvider) GetRegions(ctx context.Context) ([]providers.Region, error) {

	args := m.Called(ctx)

	return args.Get(0).([]providers.Region), args.Error(1)

}



// GetAvailabilityZones performs getavailabilityzones operation.

func (m *MockAzureProvider) GetAvailabilityZones(ctx context.Context, region string) ([]providers.AvailabilityZone, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.AvailabilityZone), args.Error(1)

}



// CreateResourcePool performs createresourcepool operation.

func (m *MockAzureProvider) CreateResourcePool(ctx context.Context, req *providers.CreateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// DeleteResourcePool performs deleteresourcepool operation.

func (m *MockAzureProvider) DeleteResourcePool(ctx context.Context, poolID string) error {

	args := m.Called(ctx, poolID)

	return args.Error(0)

}



// UpdateResourcePool performs updateresourcepool operation.

func (m *MockAzureProvider) UpdateResourcePool(ctx context.Context, poolID string, req *providers.UpdateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// GetResourcePool performs getresourcepool operation.

func (m *MockAzureProvider) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// ListResourcePools performs listresourcepools operation.

func (m *MockAzureProvider) ListResourcePools(ctx context.Context, filter *providers.ResourcePoolFilter) ([]*models.ResourcePool, error) {

	args := m.Called(ctx, filter)

	return args.Get(0).([]*models.ResourcePool), args.Error(1)

}



// CreateComputeInstance performs createcomputeinstance operation.

func (m *MockAzureProvider) CreateComputeInstance(ctx context.Context, req *providers.CreateComputeInstanceRequest) (*models.ResourceInstance, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// DeleteComputeInstance performs deletecomputeinstance operation.

func (m *MockAzureProvider) DeleteComputeInstance(ctx context.Context, instanceID string) error {

	args := m.Called(ctx, instanceID)

	return args.Error(0)

}



// GetComputeInstance performs getcomputeinstance operation.

func (m *MockAzureProvider) GetComputeInstance(ctx context.Context, instanceID string) (*models.ResourceInstance, error) {

	args := m.Called(ctx, instanceID)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// GetInstanceTypes performs getinstancetypes operation.

func (m *MockAzureProvider) GetInstanceTypes(ctx context.Context, region string) ([]providers.InstanceType, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.InstanceType), args.Error(1)

}



// GetResourceMetrics performs getresourcemetrics operation.

func (m *MockAzureProvider) GetResourceMetrics(ctx context.Context, resourceID string) (*models.ResourceMetrics, error) {

	args := m.Called(ctx, resourceID)

	return args.Get(0).(*models.ResourceMetrics), args.Error(1)

}



// ValidateCredentials performs validatecredentials operation.

func (m *MockAzureProvider) ValidateCredentials(ctx context.Context) error {

	args := m.Called(ctx)

	return args.Error(0)

}



// GetQuotas performs getquotas operation.

func (m *MockAzureProvider) GetQuotas(ctx context.Context, region string) (*providers.QuotaInfo, error) {

	args := m.Called(ctx, region)

	return args.Get(0).(*providers.QuotaInfo), args.Error(1)

}



// NewMockAzureProvider creates a new mock Azure provider with default behaviors.

func NewMockAzureProvider() *MockAzureProvider {

	provider := &MockAzureProvider{}



	// Setup default behaviors.

	provider.On("GetRegions", mock.Anything).Return([]providers.Region{

		{ID: "eastus", Name: "East US", Location: "Virginia, USA"},

		{ID: "westus2", Name: "West US 2", Location: "Washington, USA"},

		{ID: "westeurope", Name: "West Europe", Location: "Netherlands"},

	}, nil)



	provider.On("GetAvailabilityZones", mock.Anything, "eastus").Return([]providers.AvailabilityZone{

		{ID: "eastus-1", Name: "eastus-1", Region: "eastus", Status: "available"},

		{ID: "eastus-2", Name: "eastus-2", Region: "eastus", Status: "available"},

		{ID: "eastus-3", Name: "eastus-3", Region: "eastus", Status: "available"},

	}, nil)



	provider.On("GetInstanceTypes", mock.Anything, mock.AnythingOfType("string")).Return([]providers.InstanceType{

		{

			Name:         "Standard_D2s_v3",

			CPU:          "2",

			Memory:       "8Gi",

			Network:      "moderate",

			PricePerHour: 0.096,

		},

		{

			Name:         "Standard_D4s_v3",

			CPU:          "4",

			Memory:       "16Gi",

			Network:      "moderate",

			PricePerHour: 0.192,

		},

		{

			Name:         "Standard_D8s_v3",

			CPU:          "8",

			Memory:       "32Gi",

			Network:      "high",

			PricePerHour: 0.384,

		},

	}, nil)



	provider.On("ValidateCredentials", mock.Anything).Return(nil)



	provider.On("GetQuotas", mock.Anything, mock.AnythingOfType("string")).Return(&providers.QuotaInfo{

		ComputeInstances: 100,

		VCPUs:            200,

		Memory:           "800Gi",

		Storage:          "10Ti",

	}, nil)



	return provider

}



// MockGCPProvider provides mock GCP cloud provider functionality.

type MockGCPProvider struct {

	mock.Mock

}



// GetProviderType performs getprovidertype operation.

func (m *MockGCPProvider) GetProviderType() string {

	return "gcp"

}



// Initialize performs initialize operation.

func (m *MockGCPProvider) Initialize(ctx context.Context, config map[string]interface{}) error {

	args := m.Called(ctx, config)

	return args.Error(0)

}



// GetRegions performs getregions operation.

func (m *MockGCPProvider) GetRegions(ctx context.Context) ([]providers.Region, error) {

	args := m.Called(ctx)

	return args.Get(0).([]providers.Region), args.Error(1)

}



// GetAvailabilityZones performs getavailabilityzones operation.

func (m *MockGCPProvider) GetAvailabilityZones(ctx context.Context, region string) ([]providers.AvailabilityZone, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.AvailabilityZone), args.Error(1)

}



// CreateResourcePool performs createresourcepool operation.

func (m *MockGCPProvider) CreateResourcePool(ctx context.Context, req *providers.CreateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// DeleteResourcePool performs deleteresourcepool operation.

func (m *MockGCPProvider) DeleteResourcePool(ctx context.Context, poolID string) error {

	args := m.Called(ctx, poolID)

	return args.Error(0)

}



// UpdateResourcePool performs updateresourcepool operation.

func (m *MockGCPProvider) UpdateResourcePool(ctx context.Context, poolID string, req *providers.UpdateResourcePoolRequest) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID, req)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// GetResourcePool performs getresourcepool operation.

func (m *MockGCPProvider) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {

	args := m.Called(ctx, poolID)

	return args.Get(0).(*models.ResourcePool), args.Error(1)

}



// ListResourcePools performs listresourcepools operation.

func (m *MockGCPProvider) ListResourcePools(ctx context.Context, filter *providers.ResourcePoolFilter) ([]*models.ResourcePool, error) {

	args := m.Called(ctx, filter)

	return args.Get(0).([]*models.ResourcePool), args.Error(1)

}



// CreateComputeInstance performs createcomputeinstance operation.

func (m *MockGCPProvider) CreateComputeInstance(ctx context.Context, req *providers.CreateComputeInstanceRequest) (*models.ResourceInstance, error) {

	args := m.Called(ctx, req)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// DeleteComputeInstance performs deletecomputeinstance operation.

func (m *MockGCPProvider) DeleteComputeInstance(ctx context.Context, instanceID string) error {

	args := m.Called(ctx, instanceID)

	return args.Error(0)

}



// GetComputeInstance performs getcomputeinstance operation.

func (m *MockGCPProvider) GetComputeInstance(ctx context.Context, instanceID string) (*models.ResourceInstance, error) {

	args := m.Called(ctx, instanceID)

	return args.Get(0).(*models.ResourceInstance), args.Error(1)

}



// GetInstanceTypes performs getinstancetypes operation.

func (m *MockGCPProvider) GetInstanceTypes(ctx context.Context, region string) ([]providers.InstanceType, error) {

	args := m.Called(ctx, region)

	return args.Get(0).([]providers.InstanceType), args.Error(1)

}



// GetResourceMetrics performs getresourcemetrics operation.

func (m *MockGCPProvider) GetResourceMetrics(ctx context.Context, resourceID string) (*models.ResourceMetrics, error) {

	args := m.Called(ctx, resourceID)

	return args.Get(0).(*models.ResourceMetrics), args.Error(1)

}



// ValidateCredentials performs validatecredentials operation.

func (m *MockGCPProvider) ValidateCredentials(ctx context.Context) error {

	args := m.Called(ctx)

	return args.Error(0)

}



// GetQuotas performs getquotas operation.

func (m *MockGCPProvider) GetQuotas(ctx context.Context, region string) (*providers.QuotaInfo, error) {

	args := m.Called(ctx, region)

	return args.Get(0).(*providers.QuotaInfo), args.Error(1)

}



// NewMockGCPProvider creates a new mock GCP provider with default behaviors.

func NewMockGCPProvider() *MockGCPProvider {

	provider := &MockGCPProvider{}



	// Setup default behaviors.

	provider.On("GetRegions", mock.Anything).Return([]providers.Region{

		{ID: "us-central1", Name: "us-central1", Location: "Iowa, USA"},

		{ID: "us-east1", Name: "us-east1", Location: "South Carolina, USA"},

		{ID: "europe-west1", Name: "europe-west1", Location: "Belgium"},

	}, nil)



	provider.On("GetAvailabilityZones", mock.Anything, "us-central1").Return([]providers.AvailabilityZone{

		{ID: "us-central1-a", Name: "us-central1-a", Region: "us-central1", Status: "UP"},

		{ID: "us-central1-b", Name: "us-central1-b", Region: "us-central1", Status: "UP"},

		{ID: "us-central1-c", Name: "us-central1-c", Region: "us-central1", Status: "UP"},

	}, nil)



	provider.On("GetInstanceTypes", mock.Anything, mock.AnythingOfType("string")).Return([]providers.InstanceType{

		{

			Name:         "n1-standard-2",

			CPU:          "2",

			Memory:       "7.5Gi",

			Network:      "up to 10 Gbps",

			PricePerHour: 0.095,

		},

		{

			Name:         "n1-standard-4",

			CPU:          "4",

			Memory:       "15Gi",

			Network:      "up to 10 Gbps",

			PricePerHour: 0.190,

		},

		{

			Name:         "n1-standard-8",

			CPU:          "8",

			Memory:       "30Gi",

			Network:      "up to 16 Gbps",

			PricePerHour: 0.380,

		},

	}, nil)



	provider.On("ValidateCredentials", mock.Anything).Return(nil)



	provider.On("GetQuotas", mock.Anything, mock.AnythingOfType("string")).Return(&providers.QuotaInfo{

		ComputeInstances: 100,

		VCPUs:            200,

		Memory:           "800Gi",

		Storage:          "10Ti",

	}, nil)



	return provider

}



// CloudProviderTestHelpers provides utility functions for testing cloud providers.

type CloudProviderTestHelpers struct{}



// CreateTestResourcePool creates a test resource pool with realistic data.

func (h *CloudProviderTestHelpers) CreateTestResourcePool(provider, region string) *models.ResourcePool {

	poolID := fmt.Sprintf("test-pool-%s-%s-%d", provider, region, time.Now().UnixNano())



	return &models.ResourcePool{

		ResourcePoolID: poolID,

		Name:           fmt.Sprintf("Test Pool %s %s", provider, region),

		Description:    fmt.Sprintf("Integration test resource pool for %s in %s", provider, region),

		Location:       region,

		OCloudID:       fmt.Sprintf("test-ocloud-%s", provider),

		Provider:       provider,

		Region:         region,

		Zone:           region + "a", // Default to first zone

		Status: &models.ResourcePoolStatus{

			State:           "AVAILABLE",

			Health:          "HEALTHY",

			Utilization:     20.0,

			LastHealthCheck: time.Now(),

		},

		Capacity: &models.ResourceCapacity{

			CPU: &models.ResourceMetric{

				Total:       "100",

				Available:   "80",

				Used:        "20",

				Unit:        "cores",

				Utilization: 20.0,

			},

			Memory: &models.ResourceMetric{

				Total:       "400Gi",

				Available:   "320Gi",

				Used:        "80Gi",

				Unit:        "bytes",

				Utilization: 20.0,

			},

			Storage: &models.ResourceMetric{

				Total:       "10Ti",

				Available:   "8Ti",

				Used:        "2Ti",

				Unit:        "bytes",

				Utilization: 20.0,

			},

		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),

	}

}



// CreateTestComputeInstance creates a test compute instance.

func (h *CloudProviderTestHelpers) CreateTestComputeInstance(provider, instanceType string) *models.ResourceInstance {

	instanceID := fmt.Sprintf("test-instance-%s-%d", provider, time.Now().UnixNano())



	return &models.ResourceInstance{

		ResourceInstanceID:   instanceID,

		ResourceTypeID:       fmt.Sprintf("%s-compute", provider),

		ResourcePoolID:       fmt.Sprintf("test-pool-%s", provider),

		Name:                 fmt.Sprintf("Test Instance %s", provider),

		Description:          fmt.Sprintf("Test compute instance for %s", provider),

		State:                "INSTANTIATED",

		OperationalStatus:    "ENABLED",

		AdministrativeStatus: "UNLOCKED",

		UsageStatus:          "ACTIVE",

		CreatedAt:            time.Now(),

		UpdatedAt:            time.Now(),

		Metadata: map[string]interface{}{

			"instanceType": instanceType,

			"provider":     provider,

			"publicIP":     "203.0.113.1",

			"privateIP":    "10.0.1.100",

		},

	}

}



// SimulateProviderDelay simulates realistic cloud provider API delays.

func (h *CloudProviderTestHelpers) SimulateProviderDelay(operation string) {

	delays := map[string]time.Duration{

		"create_instance": 2 * time.Second,

		"delete_instance": 1500 * time.Millisecond,

		"list_instances":  500 * time.Millisecond,

		"get_instance":    200 * time.Millisecond,

		"create_pool":     3 * time.Second,

		"delete_pool":     2 * time.Second,

		"list_pools":      300 * time.Millisecond,

		"get_metrics":     100 * time.Millisecond,

	}



	if delay, exists := delays[operation]; exists {

		time.Sleep(delay)

	}

}



// ValidateResourcePool validates that a resource pool has all required fields.

func (h *CloudProviderTestHelpers) ValidateResourcePool(pool *models.ResourcePool) error {

	if pool == nil {

		return fmt.Errorf("resource pool is nil")

	}

	if pool.ResourcePoolID == "" {

		return fmt.Errorf("resource pool ID is required")

	}

	if pool.Name == "" {

		return fmt.Errorf("resource pool name is required")

	}

	if pool.Provider == "" {

		return fmt.Errorf("resource pool provider is required")

	}

	if pool.Region == "" {

		return fmt.Errorf("resource pool region is required")

	}

	if pool.Capacity == nil {

		return fmt.Errorf("resource pool capacity is required")

	}

	if pool.Capacity.CPU == nil {

		return fmt.Errorf("resource pool CPU capacity is required")

	}

	if pool.Capacity.Memory == nil {

		return fmt.Errorf("resource pool memory capacity is required")

	}

	return nil

}



// ValidateComputeInstance validates that a compute instance has all required fields.

func (h *CloudProviderTestHelpers) ValidateComputeInstance(instance *models.ResourceInstance) error {

	if instance == nil {

		return fmt.Errorf("compute instance is nil")

	}

	if instance.ResourceInstanceID == "" {

		return fmt.Errorf("compute instance ID is required")

	}

	if instance.Name == "" {

		return fmt.Errorf("compute instance name is required")

	}

	if instance.State == "" {

		return fmt.Errorf("compute instance state is required")

	}

	if instance.ResourcePoolID == "" {

		return fmt.Errorf("compute instance pool ID is required")

	}

	return nil

}



// NewCloudProviderTestHelpers creates a new instance of test helpers.

func NewCloudProviderTestHelpers() *CloudProviderTestHelpers {

	return &CloudProviderTestHelpers{}

}


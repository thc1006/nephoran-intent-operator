package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
	"github.com/thc1006/nephoran-intent-operator/tests/o2/mocks"
)

type MultiCloudTestSuite struct {
	suite.Suite
	awsProvider   *mocks.MockAWSProvider
	azureProvider *mocks.MockAzureProvider
	gcpProvider   *mocks.MockGCPProvider
	helpers       *mocks.CloudProviderTestHelpers
	testLogger    *logging.StructuredLogger
}

func (suite *MultiCloudTestSuite) SetupSuite() {
	suite.awsProvider = mocks.NewMockAWSProvider()
	suite.azureProvider = mocks.NewMockAzureProvider()
	suite.gcpProvider = mocks.NewMockGCPProvider()
	suite.helpers = mocks.NewCloudProviderTestHelpers()
	suite.testLogger = logging.NewLogger("multi-cloud-test", "debug")
}

func (suite *MultiCloudTestSuite) TestAWSProviderOperations() {
	ctx := context.Background()

	suite.Run("AWS Provider Initialization", func() {
		config := map[string]interface{}{
			"accessKeyId":     "test-access-key",
			"secretAccessKey": "test-secret-key",
			"region":          "us-east-1",
		}

		suite.awsProvider.On("Initialize", ctx, config).Return(nil).Once()
		err := suite.awsProvider.Initialize(ctx, config)
		suite.Require().NoError(err)
		suite.awsProvider.AssertExpectations(suite.T())
	})

	suite.Run("AWS Get Regions", func() {
		regions, err := suite.awsProvider.GetRegions(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(regions)
		suite.Assert().Contains([]string{"us-east-1", "us-west-2", "eu-west-1"}, regions[0].ID)

		for _, region := range regions {
			suite.Assert().NotEmpty(region.ID)
			suite.Assert().NotEmpty(region.Name)
			suite.Assert().NotEmpty(region.Location)
		}
	})

	suite.Run("AWS Get Availability Zones", func() {
		zones, err := suite.awsProvider.GetAvailabilityZones(ctx, "us-east-1")
		suite.Require().NoError(err)
		suite.Assert().Len(zones, 3)
		suite.Assert().Equal("us-east-1a", zones[0].ID)
		suite.Assert().Equal("available", zones[0].Status)
	})

	suite.Run("AWS Resource Pool Operations", func() {
		// Create resource pool
		testPool := suite.helpers.CreateTestResourcePool("aws", "us-east-1")

		createReq := &providers.CreateResourcePoolRequest{
			Name:        testPool.Name,
			Description: testPool.Description,
			Location:    testPool.Location,
			Region:      testPool.Region,
			Zone:        testPool.Zone,
		}

		suite.awsProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.awsProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)
		suite.Assert().Equal(testPool.ResourcePoolID, createdPool.ResourcePoolID)
		suite.Assert().Equal("aws", createdPool.Provider)

		// Validate the created pool
		err = suite.helpers.ValidateResourcePool(createdPool)
		suite.Require().NoError(err)

		// Get resource pool
		suite.awsProvider.On("GetResourcePool", ctx, testPool.ResourcePoolID).Return(testPool, nil).Once()
		retrievedPool, err := suite.awsProvider.GetResourcePool(ctx, testPool.ResourcePoolID)
		suite.Require().NoError(err)
		suite.Assert().Equal(testPool.ResourcePoolID, retrievedPool.ResourcePoolID)

		// Update resource pool
		updateReq := &providers.UpdateResourcePoolRequest{
			Description: "Updated AWS pool description",
		}
		updatedPool := *testPool
		updatedPool.Description = updateReq.Description

		suite.awsProvider.On("UpdateResourcePool", ctx, testPool.ResourcePoolID, updateReq).Return(&updatedPool, nil).Once()
		result, err := suite.awsProvider.UpdateResourcePool(ctx, testPool.ResourcePoolID, updateReq)
		suite.Require().NoError(err)
		suite.Assert().Equal(updateReq.Description, result.Description)

		// Delete resource pool
		suite.awsProvider.On("DeleteResourcePool", ctx, testPool.ResourcePoolID).Return(nil).Once()
		err = suite.awsProvider.DeleteResourcePool(ctx, testPool.ResourcePoolID)
		suite.Require().NoError(err)

		suite.awsProvider.AssertExpectations(suite.T())
	})

	suite.Run("AWS Compute Instance Operations", func() {
		testInstance := suite.helpers.CreateTestComputeInstance("aws", "c5.large")

		createReq := &providers.CreateComputeInstanceRequest{
			Name:           testInstance.Name,
			InstanceType:   "c5.large",
			ResourcePoolID: testInstance.ResourcePoolID,
			ImageID:        "ami-12345678",
		}

		suite.awsProvider.On("CreateComputeInstance", ctx, createReq).Return(testInstance, nil).Once()
		createdInstance, err := suite.awsProvider.CreateComputeInstance(ctx, createReq)
		suite.Require().NoError(err)
		suite.Assert().Equal(testInstance.ResourceInstanceID, createdInstance.ResourceInstanceID)

		// Validate the created instance
		err = suite.helpers.ValidateComputeInstance(createdInstance)
		suite.Require().NoError(err)

		// Get compute instance
		suite.awsProvider.On("GetComputeInstance", ctx, testInstance.ResourceInstanceID).Return(testInstance, nil).Once()
		retrievedInstance, err := suite.awsProvider.GetComputeInstance(ctx, testInstance.ResourceInstanceID)
		suite.Require().NoError(err)
		suite.Assert().Equal(testInstance.ResourceInstanceID, retrievedInstance.ResourceInstanceID)

		// Delete compute instance
		suite.awsProvider.On("DeleteComputeInstance", ctx, testInstance.ResourceInstanceID).Return(nil).Once()
		err = suite.awsProvider.DeleteComputeInstance(ctx, testInstance.ResourceInstanceID)
		suite.Require().NoError(err)

		suite.awsProvider.AssertExpectations(suite.T())
	})
}

func (suite *MultiCloudTestSuite) TestAzureProviderOperations() {
	ctx := context.Background()

	suite.Run("Azure Provider Initialization", func() {
		config := map[string]interface{}{
			"subscriptionId": "test-subscription-id",
			"clientId":       "test-client-id",
			"clientSecret":   "test-client-secret",
			"tenantId":       "test-tenant-id",
		}

		suite.azureProvider.On("Initialize", ctx, config).Return(nil).Once()
		err := suite.azureProvider.Initialize(ctx, config)
		suite.Require().NoError(err)
		suite.azureProvider.AssertExpectations(suite.T())
	})

	suite.Run("Azure Get Regions", func() {
		regions, err := suite.azureProvider.GetRegions(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(regions)
		suite.Assert().Contains([]string{"eastus", "westus2", "westeurope"}, regions[0].ID)
	})

	suite.Run("Azure Resource Pool Lifecycle", func() {
		testPool := suite.helpers.CreateTestResourcePool("azure", "eastus")
		testPool.Extensions = map[string]interface{}{
			"resourceGroup": "test-rg",
			"vnet":          "test-vnet",
		}

		createReq := &providers.CreateResourcePoolRequest{
			Name:        testPool.Name,
			Description: testPool.Description,
			Location:    testPool.Location,
			Region:      testPool.Region,
			Zone:        testPool.Zone,
		}

		// Test complete lifecycle
		suite.azureProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.azureProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)
		suite.Assert().Equal("azure", createdPool.Provider)
		suite.Assert().Equal("eastus", createdPool.Region)

		// Check Azure-specific extensions
		suite.Assert().Contains(createdPool.Extensions, "resourceGroup")
		suite.Assert().Contains(createdPool.Extensions, "vnet")

		suite.azureProvider.On("DeleteResourcePool", ctx, testPool.ResourcePoolID).Return(nil).Once()
		err = suite.azureProvider.DeleteResourcePool(ctx, testPool.ResourcePoolID)
		suite.Require().NoError(err)

		suite.azureProvider.AssertExpectations(suite.T())
	})

	suite.Run("Azure Instance Types", func() {
		instanceTypes, err := suite.azureProvider.GetInstanceTypes(ctx, "eastus")
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(instanceTypes)

		// Verify Azure-specific instance types
		typeNames := make([]string, len(instanceTypes))
		for i, instanceType := range instanceTypes {
			typeNames[i] = instanceType.Name
			suite.Assert().NotEmpty(instanceType.CPU)
			suite.Assert().NotEmpty(instanceType.Memory)
			suite.Assert().Greater(instanceType.PricePerHour, 0.0)
		}

		suite.Assert().Contains(typeNames, "Standard_D2s_v3")
		suite.Assert().Contains(typeNames, "Standard_D4s_v3")
	})
}

func (suite *MultiCloudTestSuite) TestGCPProviderOperations() {
	ctx := context.Background()

	suite.Run("GCP Provider Initialization", func() {
		config := map[string]interface{}{
			"projectId":       "test-project-id",
			"credentialsFile": "/path/to/service-account.json",
			"region":          "us-central1",
		}

		suite.gcpProvider.On("Initialize", ctx, config).Return(nil).Once()
		err := suite.gcpProvider.Initialize(ctx, config)
		suite.Require().NoError(err)
		suite.gcpProvider.AssertExpectations(suite.T())
	})

	suite.Run("GCP Get Regions", func() {
		regions, err := suite.gcpProvider.GetRegions(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(regions)
		suite.Assert().Contains([]string{"us-central1", "us-east1", "europe-west1"}, regions[0].ID)
	})

	suite.Run("GCP Resource Pool with Custom Networking", func() {
		testPool := suite.helpers.CreateTestResourcePool("gcp", "us-central1")
		testPool.Extensions = map[string]interface{}{
			"network":    "default",
			"subnetwork": "default",
			"projectId":  "test-project",
		}

		createReq := &providers.CreateResourcePoolRequest{
			Name:        testPool.Name,
			Description: testPool.Description,
			Location:    testPool.Location,
			Region:      testPool.Region,
			Zone:        testPool.Zone,
		}

		suite.gcpProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.gcpProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)
		suite.Assert().Equal("gcp", createdPool.Provider)

		// Check GCP-specific extensions
		suite.Assert().Contains(createdPool.Extensions, "network")
		suite.Assert().Contains(createdPool.Extensions, "projectId")

		suite.gcpProvider.On("DeleteResourcePool", ctx, testPool.ResourcePoolID).Return(nil).Once()
		err = suite.gcpProvider.DeleteResourcePool(ctx, testPool.ResourcePoolID)
		suite.Require().NoError(err)

		suite.gcpProvider.AssertExpectations(suite.T())
	})

	suite.Run("GCP Availability Zones", func() {
		zones, err := suite.gcpProvider.GetAvailabilityZones(ctx, "us-central1")
		suite.Require().NoError(err)
		suite.Assert().Len(zones, 3)

		for _, zone := range zones {
			suite.Assert().Contains(zone.ID, "us-central1")
			suite.Assert().Equal("UP", zone.Status)
			suite.Assert().Equal("us-central1", zone.Region)
		}
	})
}

func (suite *MultiCloudTestSuite) TestCrossCloudComparison() {
	ctx := context.Background()

	suite.Run("Compare Instance Types Across Providers", func() {
		// Get instance types from all providers
		awsTypes, err := suite.awsProvider.GetInstanceTypes(ctx, "us-east-1")
		suite.Require().NoError(err)

		azureTypes, err := suite.azureProvider.GetInstanceTypes(ctx, "eastus")
		suite.Require().NoError(err)

		gcpTypes, err := suite.gcpProvider.GetInstanceTypes(ctx, "us-central1")
		suite.Require().NoError(err)

		// Verify each provider has different naming conventions but similar capabilities
		suite.Assert().True(len(awsTypes) > 0)
		suite.Assert().True(len(azureTypes) > 0)
		suite.Assert().True(len(gcpTypes) > 0)

		// AWS uses names like c5.large
		suite.Assert().Contains(awsTypes[0].Name, "c5.")
		// Azure uses names like Standard_D2s_v3
		suite.Assert().Contains(azureTypes[0].Name, "Standard_")
		// GCP uses names like n1-standard-2
		suite.Assert().Contains(gcpTypes[0].Name, "n1-")

		// Verify all providers support similar resource ranges
		for _, types := range [][]providers.InstanceType{awsTypes, azureTypes, gcpTypes} {
			for _, instanceType := range types {
				suite.Assert().NotEmpty(instanceType.CPU)
				suite.Assert().NotEmpty(instanceType.Memory)
				suite.Assert().Greater(instanceType.PricePerHour, 0.0)
			}
		}
	})

	suite.Run("Compare Quota Systems", func() {
		awsQuota, err := suite.awsProvider.GetQuotas(ctx, "us-east-1")
		suite.Require().NoError(err)

		azureQuota, err := suite.azureProvider.GetQuotas(ctx, "eastus")
		suite.Require().NoError(err)

		gcpQuota, err := suite.gcpProvider.GetQuotas(ctx, "us-central1")
		suite.Require().NoError(err)

		// All providers should have similar quota structures
		quotas := []*providers.QuotaInfo{awsQuota, azureQuota, gcpQuota}
		for _, quota := range quotas {
			suite.Assert().Greater(quota.ComputeInstances, 0)
			suite.Assert().Greater(quota.VCPUs, 0)
			suite.Assert().NotEmpty(quota.Memory)
			suite.Assert().NotEmpty(quota.Storage)
		}
	})

	suite.Run("Cross-Provider Resource Pool Consistency", func() {
		providers := map[string]interface{}{
			"aws":   suite.awsProvider,
			"azure": suite.azureProvider,
			"gcp":   suite.gcpProvider,
		}

		regions := map[string]string{
			"aws":   "us-east-1",
			"azure": "eastus",
			"gcp":   "us-central1",
		}

		for providerName, region := range regions {
			testPool := suite.helpers.CreateTestResourcePool(providerName, region)

			// Verify consistent resource pool structure across providers
			suite.Assert().Equal(providerName, testPool.Provider)
			suite.Assert().Equal(region, testPool.Region)
			suite.Assert().NotNil(testPool.Capacity)
			suite.Assert().NotNil(testPool.Capacity.CPU)
			suite.Assert().NotNil(testPool.Capacity.Memory)
			suite.Assert().NotNil(testPool.Capacity.Storage)

			// Verify resource metrics have consistent structure
			suite.Assert().Equal("cores", testPool.Capacity.CPU.Unit)
			suite.Assert().Equal("bytes", testPool.Capacity.Memory.Unit)
			suite.Assert().Equal("bytes", testPool.Capacity.Storage.Unit)
		}
	})
}

func (suite *MultiCloudTestSuite) TestProviderFailureHandling() {
	ctx := context.Background()

	suite.Run("Handle Provider Initialization Failures", func() {
		// Test AWS credential failures
		suite.awsProvider.On("ValidateCredentials", ctx).Return(fmt.Errorf("invalid AWS credentials")).Once()
		err := suite.awsProvider.ValidateCredentials(ctx)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "invalid AWS credentials")

		// Test Azure credential failures
		suite.azureProvider.On("ValidateCredentials", ctx).Return(fmt.Errorf("azure authentication failed")).Once()
		err = suite.azureProvider.ValidateCredentials(ctx)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "azure authentication failed")

		// Test GCP credential failures
		suite.gcpProvider.On("ValidateCredentials", ctx).Return(fmt.Errorf("GCP service account invalid")).Once()
		err = suite.gcpProvider.ValidateCredentials(ctx)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "GCP service account invalid")

		suite.awsProvider.AssertExpectations(suite.T())
		suite.azureProvider.AssertExpectations(suite.T())
		suite.gcpProvider.AssertExpectations(suite.T())
	})

	suite.Run("Handle Resource Operation Failures", func() {
		poolID := "non-existent-pool"

		// Test resource not found across all providers
		suite.awsProvider.On("GetResourcePool", ctx, poolID).Return((*models.ResourcePool)(nil), fmt.Errorf("resource pool not found")).Once()
		_, err := suite.awsProvider.GetResourcePool(ctx, poolID)
		suite.Assert().Error(err)

		suite.azureProvider.On("GetResourcePool", ctx, poolID).Return((*models.ResourcePool)(nil), fmt.Errorf("resource pool not found")).Once()
		_, err = suite.azureProvider.GetResourcePool(ctx, poolID)
		suite.Assert().Error(err)

		suite.gcpProvider.On("GetResourcePool", ctx, poolID).Return((*models.ResourcePool)(nil), fmt.Errorf("resource pool not found")).Once()
		_, err = suite.gcpProvider.GetResourcePool(ctx, poolID)
		suite.Assert().Error(err)

		suite.awsProvider.AssertExpectations(suite.T())
		suite.azureProvider.AssertExpectations(suite.T())
		suite.gcpProvider.AssertExpectations(suite.T())
	})
}

func (suite *MultiCloudTestSuite) TestResourceMetrics() {
	ctx := context.Background()

	suite.Run("Collect Metrics from All Providers", func() {
		resourceID := "test-resource-123"

		expectedMetrics := &models.ResourceMetrics{
			Timestamp: time.Now(),
			CPU: &models.MetricData{
				Value:     45.5,
				Unit:      "percent",
				Timestamp: time.Now(),
			},
			Memory: &models.MetricData{
				Value:     67.2,
				Unit:      "percent",
				Timestamp: time.Now(),
			},
			NetworkIO: &models.NetworkIOMetrics{
				BytesIn:  1024 * 1024 * 100, // 100MB
				BytesOut: 1024 * 1024 * 80,  // 80MB
			},
		}

		// Test metrics collection from each provider
		suite.awsProvider.On("GetResourceMetrics", ctx, resourceID).Return(expectedMetrics, nil).Once()
		awsMetrics, err := suite.awsProvider.GetResourceMetrics(ctx, resourceID)
		suite.Require().NoError(err)
		suite.Assert().Equal(45.5, awsMetrics.CPU.Value)

		suite.azureProvider.On("GetResourceMetrics", ctx, resourceID).Return(expectedMetrics, nil).Once()
		azureMetrics, err := suite.azureProvider.GetResourceMetrics(ctx, resourceID)
		suite.Require().NoError(err)
		suite.Assert().Equal(67.2, azureMetrics.Memory.Value)

		suite.gcpProvider.On("GetResourceMetrics", ctx, resourceID).Return(expectedMetrics, nil).Once()
		gcpMetrics, err := suite.gcpProvider.GetResourceMetrics(ctx, resourceID)
		suite.Require().NoError(err)
		suite.Assert().Equal(int64(1024*1024*100), gcpMetrics.NetworkIO.BytesIn)

		suite.awsProvider.AssertExpectations(suite.T())
		suite.azureProvider.AssertExpectations(suite.T())
		suite.gcpProvider.AssertExpectations(suite.T())
	})
}

func (suite *MultiCloudTestSuite) TestProviderSpecificFeatures() {
	ctx := context.Background()

	suite.Run("AWS Specific Features", func() {
		// Test AWS-specific functionality like spot instances, placement groups, etc.
		testPool := suite.helpers.CreateTestResourcePool("aws", "us-east-1")
		testPool.Extensions = map[string]interface{}{
			"spotInstancesEnabled": true,
			"placementGroup":       "cluster-pg-1",
			"instanceProfile":      "ec2-role",
			"securityGroups":       []string{"sg-12345", "sg-67890"},
		}

		createReq := &providers.CreateResourcePoolRequest{
			Name:     testPool.Name,
			Location: testPool.Location,
			Region:   testPool.Region,
		}

		suite.awsProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.awsProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)

		// Verify AWS-specific extensions
		suite.Assert().True(createdPool.Extensions["spotInstancesEnabled"].(bool))
		suite.Assert().Equal("cluster-pg-1", createdPool.Extensions["placementGroup"])
		suite.Assert().Contains(createdPool.Extensions["securityGroups"], "sg-12345")

		suite.awsProvider.AssertExpectations(suite.T())
	})

	suite.Run("Azure Specific Features", func() {
		// Test Azure-specific functionality like managed disks, availability sets, etc.
		testPool := suite.helpers.CreateTestResourcePool("azure", "eastus")
		testPool.Extensions = map[string]interface{}{
			"availabilitySet":         "avset-1",
			"managedDiskType":         "Premium_LRS",
			"acceleratedNetworking":   true,
			"proximityPlacementGroup": "ppg-1",
		}

		createReq := &providers.CreateResourcePoolRequest{
			Name:     testPool.Name,
			Location: testPool.Location,
			Region:   testPool.Region,
		}

		suite.azureProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.azureProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)

		// Verify Azure-specific extensions
		suite.Assert().Equal("avset-1", createdPool.Extensions["availabilitySet"])
		suite.Assert().Equal("Premium_LRS", createdPool.Extensions["managedDiskType"])
		suite.Assert().True(createdPool.Extensions["acceleratedNetworking"].(bool))

		suite.azureProvider.AssertExpectations(suite.T())
	})

	suite.Run("GCP Specific Features", func() {
		// Test GCP-specific functionality like preemptible instances, custom machine types, etc.
		testPool := suite.helpers.CreateTestResourcePool("gcp", "us-central1")
		testPool.Extensions = map[string]interface{}{
			"preemptibleInstances": true,
			"customMachineType":    true,
			"localSSDCount":        2,
			"nodeGroup":            "test-node-group",
		}

		createReq := &providers.CreateResourcePoolRequest{
			Name:     testPool.Name,
			Location: testPool.Location,
			Region:   testPool.Region,
		}

		suite.gcpProvider.On("CreateResourcePool", ctx, createReq).Return(testPool, nil).Once()
		createdPool, err := suite.gcpProvider.CreateResourcePool(ctx, createReq)
		suite.Require().NoError(err)

		// Verify GCP-specific extensions
		suite.Assert().True(createdPool.Extensions["preemptibleInstances"].(bool))
		suite.Assert().True(createdPool.Extensions["customMachineType"].(bool))
		suite.Assert().Equal(float64(2), createdPool.Extensions["localSSDCount"])

		suite.gcpProvider.AssertExpectations(suite.T())
	})
}

func TestMultiCloudIntegration(t *testing.T) {
	suite.Run(t, new(MultiCloudTestSuite))
}

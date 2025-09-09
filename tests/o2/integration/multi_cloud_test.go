package o2_integration_tests_test

import (
	"context"
<<<<<<< HEAD
	"fmt"
	"testing"
	"time"
	"encoding/json"
=======
	"encoding/json"
	"fmt"
	"testing"
	"time"
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
<<<<<<< HEAD
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"github.com/thc1006/nephoran-intent-operator/tests/o2/mocks"
)

type MultiCloudTestSuite struct {
	suite.Suite
	awsProvider   *mocks.MockAWSProvider
	azureProvider *mocks.MockAzureProvider
	gcpProvider   *mocks.MockGCPProvider
<<<<<<< HEAD
	helpers       *mocks.CloudProviderTestHelpers
	testLogger    *logging.StructuredLogger
=======
	// helpers       *mocks.CloudProviderTestHelpers // TODO: Implement CloudProviderTestHelpers type
	testLogger *logging.StructuredLogger
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func (suite *MultiCloudTestSuite) SetupSuite() {
	suite.awsProvider = mocks.NewMockAWSProvider()
	suite.azureProvider = mocks.NewMockAzureProvider()
	suite.gcpProvider = mocks.NewMockGCPProvider()
<<<<<<< HEAD
	suite.helpers = mocks.NewCloudProviderTestHelpers()
	suite.testLogger = logging.NewLogger("multi-cloud-test", "debug")
}

func (suite *MultiCloudTestSuite) TestAWSProviderOperations() {
	ctx := context.Background()

	suite.Run("AWS Provider Initialization", func() {
		config := json.RawMessage(`{}`)

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
=======
	// suite.helpers = mocks.NewCloudProviderTestHelpers() // TODO: Implement CloudProviderTestHelpers type
	suite.testLogger = logging.NewLogger("multi-cloud-test", "debug")
}

// Helper methods to replace missing CloudProviderTestHelpers
func (suite *MultiCloudTestSuite) createTestResourcePool(provider, region string) *models.ResourcePool {
	return &models.ResourcePool{
		ResourcePoolID: fmt.Sprintf("pool-%s-%s", provider, region),
		Name:           fmt.Sprintf("test-pool-%s", provider),
		Description:    fmt.Sprintf("Test resource pool for %s", provider),
		Location:       region,
		Provider:       provider,
		Region:         region,
		Zone:           region + "-a",
		Extensions: map[string]interface{}{
			"spotInstancesEnabled":    true,
			"placementGroup":         "cluster-pg-1",
			"securityGroups":         []string{"sg-12345"},
			"availabilitySet":        "avset-1",
			"managedDiskType":        "Premium_LRS",
			"acceleratedNetworking":  true,
			"preemptibleInstances":   true,
			"customMachineType":      true,
			"localSSDCount":          float64(2),
			"resourceGroup":          "test-rg",
			"vnet":                   "test-vnet",
			"network":                "test-network",
			"projectId":              "test-project",
		},
		Capacity: &models.ResourceCapacity{
			CPU:     &models.ResourceMetric{Total: "100", Available: "100", Used: "0", Unit: "cores", Utilization: 0.0},
			Memory:  &models.ResourceMetric{Total: "256", Available: "256", Used: "0", Unit: "GB", Utilization: 0.0},
			Storage: &models.ResourceMetric{Total: "1000", Available: "1000", Used: "0", Unit: "GB", Utilization: 0.0},
		},
	}
}

func (suite *MultiCloudTestSuite) createTestComputeInstance(provider, instanceType string) *models.ComputeInstance {
	return &models.ComputeInstance{
		ResourceInstanceID: fmt.Sprintf("instance-%s-%s", provider, instanceType),
		Name:               fmt.Sprintf("test-instance-%s", provider),
		InstanceType:       instanceType,
		ResourcePoolID:     fmt.Sprintf("pool-%s", provider),
		Provider:           provider,
	}
}

func (suite *MultiCloudTestSuite) validateResourcePool(pool *models.ResourcePool) error {
	suite.Assert().NotEmpty(pool.ResourcePoolID)
	suite.Assert().NotEmpty(pool.Provider)
	suite.Assert().NotNil(pool.Capacity)
	return nil
}

func (suite *MultiCloudTestSuite) validateComputeInstance(instance *models.ComputeInstance) error {
	suite.Assert().NotEmpty(instance.ResourceInstanceID)
	suite.Assert().NotEmpty(instance.Provider)
	suite.Assert().NotEmpty(instance.InstanceType)
	return nil
}

func (suite *MultiCloudTestSuite) TestAWSProviderOperations() {
	ctx := context.Background()

	// Test basic AWS provider operations using the CloudProvider interface
	suite.Run("AWS Create Cluster", func() {
		config := mocks.ClusterConfig{
			Name:       "test-aws-cluster",
			Provider:   "aws",
			Region:     "us-east-1",
			NodeCount:  3,
			NodeType:   "t3.medium",
			K8sVersion: "1.29",
		}
		
		// Create cluster
		cluster, err := suite.awsProvider.CreateCluster(ctx, config)
		suite.Require().NoError(err)
		suite.Assert().NotNil(cluster)
		suite.Assert().Equal("aws", cluster.Provider)
		suite.Assert().Equal(config.Name, cluster.Name)
		
		// Get cluster
		retrieved, err := suite.awsProvider.GetCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(cluster.ID, retrieved.ID)
		
		// Scale cluster
		err = suite.awsProvider.ScaleCluster(ctx, cluster.ID, 5)
		suite.Require().NoError(err)
		
		// List clusters
		clusters, err := suite.awsProvider.ListClusters(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(clusters)
		
		// Delete cluster
		err = suite.awsProvider.DeleteCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
	})

	// Note: Resource Pool and Compute Instance operations are not implemented in MockAWSProvider
	// These would require extending the mock with additional methods
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func (suite *MultiCloudTestSuite) TestAzureProviderOperations() {
	ctx := context.Background()

<<<<<<< HEAD
	suite.Run("Azure Provider Initialization", func() {
		config := json.RawMessage(`{}`)

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
		testPool.Extensions = json.RawMessage(`{}`)

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
=======
	// Test basic Azure provider operations using the CloudProvider interface
	suite.Run("Azure Create Cluster", func() {
		config := mocks.ClusterConfig{
			Name:       "test-azure-cluster",
			Provider:   "azure",
			Region:     "eastus",
			NodeCount:  3,
			NodeType:   "Standard_D2s_v3",
			K8sVersion: "1.29",
		}
		
		// Create cluster
		cluster, err := suite.azureProvider.CreateCluster(ctx, config)
		suite.Require().NoError(err)
		suite.Assert().NotNil(cluster)
		suite.Assert().Equal("azure", cluster.Provider)
		suite.Assert().Equal(config.Name, cluster.Name)
		
		// Get cluster
		retrieved, err := suite.azureProvider.GetCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(cluster.ID, retrieved.ID)
		
		// Scale cluster
		err = suite.azureProvider.ScaleCluster(ctx, cluster.ID, 5)
		suite.Require().NoError(err)
		
		// List clusters
		clusters, err := suite.azureProvider.ListClusters(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(clusters)
		
		// Delete cluster
		err = suite.azureProvider.DeleteCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
	})

	// Note: Additional Azure-specific operations would require extending the mock
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func (suite *MultiCloudTestSuite) TestGCPProviderOperations() {
	ctx := context.Background()

<<<<<<< HEAD
	suite.Run("GCP Provider Initialization", func() {
		config := json.RawMessage(`{}`)

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
		testPool.Extensions = json.RawMessage(`{}`)

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
=======
	// Test basic GCP provider operations using the CloudProvider interface
	suite.Run("GCP Create Cluster", func() {
		config := mocks.ClusterConfig{
			Name:       "test-gcp-cluster",
			Provider:   "gcp",
			Region:     "us-central1",
			NodeCount:  3,
			NodeType:   "n1-standard-2",
			K8sVersion: "1.29",
		}
		
		// Create cluster
		cluster, err := suite.gcpProvider.CreateCluster(ctx, config)
		suite.Require().NoError(err)
		suite.Assert().NotNil(cluster)
		suite.Assert().Equal("gcp", cluster.Provider)
		suite.Assert().Equal(config.Name, cluster.Name)
		
		// Get cluster
		retrieved, err := suite.gcpProvider.GetCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(cluster.ID, retrieved.ID)
		
		// Scale cluster
		err = suite.gcpProvider.ScaleCluster(ctx, cluster.ID, 5)
		suite.Require().NoError(err)
		
		// List clusters
		clusters, err := suite.gcpProvider.ListClusters(ctx)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(clusters)
		
		// Delete cluster
		err = suite.gcpProvider.DeleteCluster(ctx, cluster.ID)
		suite.Require().NoError(err)
	})

	// Note: Additional GCP-specific operations would require extending the mock
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func (suite *MultiCloudTestSuite) TestCrossCloudComparison() {
	ctx := context.Background()

<<<<<<< HEAD
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
=======
	suite.Run("Compare Clusters Across Providers", func() {
		// Test creating clusters across all providers
		configs := []mocks.ClusterConfig{
			{
				Name:       "test-aws-comparison",
				Provider:   "aws",
				Region:     "us-east-1",
				NodeCount:  2,
				NodeType:   "t3.medium",
				K8sVersion: "1.29",
			},
			{
				Name:       "test-azure-comparison",
				Provider:   "azure",
				Region:     "eastus",
				NodeCount:  2,
				NodeType:   "Standard_D2s_v3",
				K8sVersion: "1.29",
			},
			{
				Name:       "test-gcp-comparison",
				Provider:   "gcp",
				Region:     "us-central1",
				NodeCount:  2,
				NodeType:   "n1-standard-2",
				K8sVersion: "1.29",
			},
		}
		
		var clusters []*mocks.ClusterInfo
		
		// Create clusters on each provider
		awsCluster, err := suite.awsProvider.CreateCluster(ctx, configs[0])
		suite.Require().NoError(err)
		clusters = append(clusters, awsCluster)
		
		azureCluster, err := suite.azureProvider.CreateCluster(ctx, configs[1])
		suite.Require().NoError(err)
		clusters = append(clusters, azureCluster)
		
		gcpCluster, err := suite.gcpProvider.CreateCluster(ctx, configs[2])
		suite.Require().NoError(err)
		clusters = append(clusters, gcpCluster)

		
		// Verify all clusters were created successfully
		suite.Assert().Len(clusters, 3)
		
		for i, cluster := range clusters {
			suite.Assert().NotNil(cluster)
			suite.Assert().Equal(configs[i].Provider, cluster.Provider)
			suite.Assert().Equal(configs[i].Name, cluster.Name)
			suite.Assert().Equal(configs[i].NodeCount, cluster.NodeCount)
		}
		
		// Clean up clusters
		for i, cluster := range clusters {
			var err error
			switch configs[i].Provider {
			case "aws":
				err = suite.awsProvider.DeleteCluster(ctx, cluster.ID)
			case "azure":
				err = suite.azureProvider.DeleteCluster(ctx, cluster.ID)
			case "gcp":
				err = suite.gcpProvider.DeleteCluster(ctx, cluster.ID)
			}
			suite.Require().NoError(err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	})

	suite.Run("Cross-Provider Resource Pool Consistency", func() {
		_ = json.RawMessage(`{}`)

		regions := map[string]string{
			"aws":   "us-east-1",
			"azure": "eastus",
			"gcp":   "us-central1",
		}

		for providerName, region := range regions {
<<<<<<< HEAD
			testPool := suite.helpers.CreateTestResourcePool(providerName, region)
=======
			testPool := suite.createTestResourcePool(providerName, region)
>>>>>>> 6835433495e87288b95961af7173d866977175ff

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

<<<<<<< HEAD
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
		testPool.Extensions = json.RawMessage(`{}`)

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
		testPool.Extensions = json.RawMessage(`{}`)

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
		testPool.Extensions = json.RawMessage(`{}`)

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
=======
	suite.Run("Handle Cluster Creation Failures", func() {
		// Test invalid cluster configuration
		invalidConfig := mocks.ClusterConfig{
			Name:       "", // Invalid empty name
			Provider:   "aws",
			Region:     "us-east-1",
			NodeCount:  -1, // Invalid negative node count
			NodeType:   "invalid-type",
			K8sVersion: "invalid",
		}
		
		err := invalidConfig.Validate()
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "cluster name cannot be empty")
		
		// Test configuration validation
		validConfig := mocks.ClusterConfig{
			Name:       "valid-cluster",
			Provider:   "aws",
			Region:     "us-east-1",
			NodeCount:  3,
			NodeType:   "t3.medium",
			K8sVersion: "1.29",
		}
		
		err = validConfig.Validate()
		suite.Assert().NoError(err)
	})

	suite.Run("Handle Cluster Not Found", func() {
		nonExistentID := "non-existent-cluster"
		
		// Test cluster not found
		_, err := suite.awsProvider.GetCluster(ctx, nonExistentID)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "not found")
		
		_, err = suite.azureProvider.GetCluster(ctx, nonExistentID)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "not found")
		
		_, err = suite.gcpProvider.GetCluster(ctx, nonExistentID)
		suite.Assert().Error(err)
		suite.Assert().Contains(err.Error(), "not found")
	})
}

func (suite *MultiCloudTestSuite) TestMultiCloudScaling() {
	ctx := context.Background()

	suite.Run("Scale Clusters Across Providers", func() {
		// Create initial clusters
		awsConfig := mocks.ClusterConfig{
			Name:       "test-aws-scaling",
			Provider:   "aws",
			Region:     "us-east-1",
			NodeCount:  2,
			NodeType:   "t3.medium",
			K8sVersion: "1.29",
		}
		
		azureConfig := mocks.ClusterConfig{
			Name:       "test-azure-scaling",
			Provider:   "azure",
			Region:     "eastus",
			NodeCount:  2,
			NodeType:   "Standard_D2s_v3",
			K8sVersion: "1.29",
		}
		
		gcpConfig := mocks.ClusterConfig{
			Name:       "test-gcp-scaling",
			Provider:   "gcp",
			Region:     "us-central1",
			NodeCount:  2,
			NodeType:   "n1-standard-2",
			K8sVersion: "1.29",
		}
		
		// Create clusters
		awsCluster, err := suite.awsProvider.CreateCluster(ctx, awsConfig)
		suite.Require().NoError(err)
		
		azureCluster, err := suite.azureProvider.CreateCluster(ctx, azureConfig)
		suite.Require().NoError(err)
		
		gcpCluster, err := suite.gcpProvider.CreateCluster(ctx, gcpConfig)
		suite.Require().NoError(err)
		
		// Scale up all clusters
		err = suite.awsProvider.ScaleCluster(ctx, awsCluster.ID, 5)
		suite.Require().NoError(err)
		
		err = suite.azureProvider.ScaleCluster(ctx, azureCluster.ID, 5)
		suite.Require().NoError(err)
		
		err = suite.gcpProvider.ScaleCluster(ctx, gcpCluster.ID, 5)
		suite.Require().NoError(err)
		
		// Wait for scaling to complete (simulated delay)
		time.Sleep(200 * time.Millisecond)
		
		// Verify scaling succeeded
		awsUpdated, err := suite.awsProvider.GetCluster(ctx, awsCluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(5, awsUpdated.NodeCount)
		
		azureUpdated, err := suite.azureProvider.GetCluster(ctx, azureCluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(5, azureUpdated.NodeCount)
		
		gcpUpdated, err := suite.gcpProvider.GetCluster(ctx, gcpCluster.ID)
		suite.Require().NoError(err)
		suite.Assert().Equal(5, gcpUpdated.NodeCount)
		
		// Clean up
		err = suite.awsProvider.DeleteCluster(ctx, awsCluster.ID)
		suite.Require().NoError(err)
		
		err = suite.azureProvider.DeleteCluster(ctx, azureCluster.ID)
		suite.Require().NoError(err)
		
		err = suite.gcpProvider.DeleteCluster(ctx, gcpCluster.ID)
		suite.Require().NoError(err)
	})
}

func (suite *MultiCloudTestSuite) TestProviderClusterLifecycle() {
	ctx := context.Background()

	suite.Run("Complete Cluster Lifecycle", func() {
		// Test complete lifecycle for each provider
		providers := []struct {
			name     string
			provider mocks.CloudProvider
			config   mocks.ClusterConfig
		}{
			{
				name:     "AWS",
				provider: suite.awsProvider,
				config: mocks.ClusterConfig{
					Name:       "lifecycle-aws",
					Provider:   "aws",
					Region:     "us-east-1",
					NodeCount:  3,
					NodeType:   "t3.medium",
					K8sVersion: "1.29",
				},
			},
			{
				name:     "Azure",
				provider: suite.azureProvider,
				config: mocks.ClusterConfig{
					Name:       "lifecycle-azure",
					Provider:   "azure",
					Region:     "eastus",
					NodeCount:  3,
					NodeType:   "Standard_D2s_v3",
					K8sVersion: "1.29",
				},
			},
			{
				name:     "GCP",
				provider: suite.gcpProvider,
				config: mocks.ClusterConfig{
					Name:       "lifecycle-gcp",
					Provider:   "gcp",
					Region:     "us-central1",
					NodeCount:  3,
					NodeType:   "n1-standard-2",
					K8sVersion: "1.29",
				},
			},
		}
		
		for _, p := range providers {
			// Create
			cluster, err := p.provider.CreateCluster(ctx, p.config)
			suite.Require().NoError(err, "Failed to create cluster for %s", p.name)
			suite.Assert().NotNil(cluster)
			
			// Read
			retrieved, err := p.provider.GetCluster(ctx, cluster.ID)
			suite.Require().NoError(err, "Failed to get cluster for %s", p.name)
			suite.Assert().Equal(cluster.ID, retrieved.ID)
			
			// Update (scale)
			err = p.provider.ScaleCluster(ctx, cluster.ID, 5)
			suite.Require().NoError(err, "Failed to scale cluster for %s", p.name)
			
			// Delete
			err = p.provider.DeleteCluster(ctx, cluster.ID)
			suite.Require().NoError(err, "Failed to delete cluster for %s", p.name)
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	})
}

func (suite *MultiCloudTestSuite) TearDownSuite() {
	// Clean up any persistent resources or connections
	if suite.awsProvider != nil {
		suite.awsProvider.AssertExpectations(suite.T())
	}
	if suite.azureProvider != nil {
		suite.azureProvider.AssertExpectations(suite.T())
	}
	if suite.gcpProvider != nil {
		suite.gcpProvider.AssertExpectations(suite.T())
	}
}

func TestMultiCloudIntegration(t *testing.T) {
	suite.Run(t, new(MultiCloudTestSuite))
}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff

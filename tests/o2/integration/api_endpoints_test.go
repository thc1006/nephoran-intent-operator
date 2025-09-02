package o2_integration_tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

type O2APITestSuite struct {
	suite.Suite
	server       *httptest.Server
	client       *http.Client
	o2Server     *o2.O2APIServer
	k8sClient    client.Client
	k8sClientset *fake.Clientset
	testLogger   *logging.StructuredLogger
}

func (suite *O2APITestSuite) SetupSuite() {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	suite.k8sClient = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	suite.k8sClientset = fake.NewSimpleClientset()
	suite.testLogger = logging.NewLogger("o2-api-test", "debug")

	// Setup O2 API server
	config := &o2.O2IMSConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    0,
		TLSEnabled:    false,
		DatabaseConfig: json.RawMessage("{}"),
		ProviderConfigs: json.RawMessage("{}"){
				"enabled": true,
			},
		},
	}

	var err error
	suite.o2Server, err = o2.NewO2APIServer(config, suite.testLogger, nil)
	suite.Require().NoError(err)

	suite.server = httptest.NewServer(suite.o2Server.GetRouter())
	suite.client = suite.server.Client()
	suite.client.Timeout = 30 * time.Second
}

func (suite *O2APITestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.o2Server != nil {
		suite.o2Server.Shutdown(context.Background())
	}
}

func (suite *O2APITestSuite) TestServiceInformationEndpoint() {
	suite.Run("GET /o2ims/v1/ returns complete service information", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
		suite.Require().NoError(err)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)

		var serviceInfo map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Validate required fields
		suite.Assert().Equal("Nephoran O2 IMS", serviceInfo["name"])
		suite.Assert().Equal("v1.0.0", serviceInfo["version"])
		suite.Assert().Equal("v1.0", serviceInfo["apiVersion"])
		suite.Assert().Equal("O-RAN.WG6.O2ims-Interface-v01.01", serviceInfo["specification"])

		// Validate capabilities
		capabilities, ok := serviceInfo["capabilities"].([]interface{})
		suite.Require().True(ok)
		suite.Assert().Contains(capabilities, "InfrastructureInventory")
		suite.Assert().Contains(capabilities, "InfrastructureMonitoring")
		suite.Assert().Contains(capabilities, "InfrastructureProvisioning")
		suite.Assert().Contains(capabilities, "DeploymentTemplates")
	})
}

func (suite *O2APITestSuite) TestResourcePoolCRUD() {
	poolID := "test-pool-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	suite.Run("Create resource pool", func() {
		pool := &models.ResourcePool{
			ResourcePoolID: poolID,
			Name:           "Integration Test Pool",
			Description:    "Resource pool for integration testing",
			Location:       "us-west-2",
			OCloudID:       "test-ocloud-1",
			Provider:       "kubernetes",
			Region:         "us-west-2",
			Zone:           "us-west-2a",
			Capacity: &models.ResourceCapacity{
				CPU: &models.ResourceMetric{
					Total:       "100",
					Available:   "75",
					Used:        "25",
					Unit:        "cores",
					Utilization: 25.0,
				},
				Memory: &models.ResourceMetric{
					Total:       "400Gi",
					Available:   "300Gi",
					Used:        "100Gi",
					Unit:        "bytes",
					Utilization: 25.0,
				},
				Storage: &models.ResourceMetric{
					Total:       "10Ti",
					Available:   "8Ti",
					Used:        "2Ti",
					Unit:        "bytes",
					Utilization: 20.0,
				},
			},
			Extensions: json.RawMessage("{}"),
		}

		poolJSON, err := json.Marshal(pool)
		suite.Require().NoError(err)

		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourcePools",
			"application/json",
			bytes.NewBuffer(poolJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusCreated, resp.StatusCode)

		var created models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&created)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(pool.ResourcePoolID, created.ResourcePoolID)
		suite.Assert().Equal(pool.Name, created.Name)
		suite.Assert().NotZero(created.CreatedAt)
	})

	suite.Run("Get resource pool", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/" + poolID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var retrieved models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&retrieved)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(poolID, retrieved.ResourcePoolID)
		suite.Assert().Equal("Integration Test Pool", retrieved.Name)
		suite.Assert().Equal("kubernetes", retrieved.Provider)
		suite.Assert().NotNil(retrieved.Capacity)
		suite.Assert().Equal(25.0, retrieved.Capacity.CPU.Utilization)
	})

	suite.Run("Update resource pool", func() {
		// First get the current pool
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/" + poolID)
		suite.Require().NoError(err)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)

		var pool models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&pool)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Update the pool
		pool.Description = "Updated description for integration test"
		pool.Capacity.CPU.Used = "30"
		pool.Capacity.CPU.Available = "70"
		pool.Capacity.CPU.Utilization = 30.0

		updatedJSON, err := json.Marshal(pool)
		suite.Require().NoError(err)

		req, err := http.NewRequest("PUT",
			suite.server.URL+"/o2ims/v1/resourcePools/"+poolID,
			bytes.NewBuffer(updatedJSON))
		suite.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err = suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify the update
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/" + poolID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var updated models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&updated)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal("Updated description for integration test", updated.Description)
		suite.Assert().Equal(30.0, updated.Capacity.CPU.Utilization)
	})

	suite.Run("Delete resource pool", func() {
		req, err := http.NewRequest("DELETE",
			suite.server.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
		suite.Require().NoError(err)

		resp, err := suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()

		// Verify deletion
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/" + poolID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusNotFound, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *O2APITestSuite) TestResourceTypeCRUD() {
	typeID := "test-resource-type-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	suite.Run("Create resource type", func() {
		resourceType := &models.ResourceType{
			ResourceTypeID: typeID,
			Name:           "5G AMF Resource Type",
			Description:    "Access and Mobility Management Function resource type",
			Vendor:         "Nephoran",
			Model:          "AMF-Standard-v1",
			Version:        "1.2.0",
			Specifications: &models.ResourceTypeSpec{
				Category:    "CNF",
				SubCategory: "5G_CORE",
				MinResources: map[string]string{
					"cpu":               "1000m",
					"memory":            "2Gi",
					"ephemeral-storage": "10Gi",
				},
				MaxResources: map[string]string{
					"cpu":               "8000m",
					"memory":            "32Gi",
					"ephemeral-storage": "100Gi",
				},
				DefaultResources: map[string]string{
					"cpu":               "2000m",
					"memory":            "4Gi",
					"ephemeral-storage": "20Gi",
				},
				RequiredPorts: []models.PortSpec{
					{Name: "sbi", Port: 8080, Protocol: "TCP"},
					{Name: "metrics", Port: 9090, Protocol: "TCP"},
				},
				OptionalPorts: []models.PortSpec{
					{Name: "debug", Port: 8081, Protocol: "TCP"},
				},
			},
			SupportedActions: []string{"CREATE", "DELETE", "UPDATE", "SCALE", "HEAL"},
			Capabilities: json.RawMessage("{}"),
		}

		typeJSON, err := json.Marshal(resourceType)
		suite.Require().NoError(err)

		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourceTypes",
			"application/json",
			bytes.NewBuffer(typeJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusCreated, resp.StatusCode)

		var created models.ResourceType
		err = json.NewDecoder(resp.Body).Decode(&created)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(typeID, created.ResourceTypeID)
		suite.Assert().Equal("5G AMF Resource Type", created.Name)
		suite.Assert().Equal("CNF", created.Specifications.Category)
	})

	suite.Run("Get resource type", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourceTypes/" + typeID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var retrieved models.ResourceType
		err = json.NewDecoder(resp.Body).Decode(&retrieved)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(typeID, retrieved.ResourceTypeID)
		suite.Assert().Equal("Nephoran", retrieved.Vendor)
		suite.Assert().Contains(retrieved.SupportedActions, "SCALE")
		suite.Assert().Contains(retrieved.SupportedActions, "HEAL")
		suite.Assert().Len(retrieved.Specifications.RequiredPorts, 2)
	})

	suite.Run("List resource types", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourceTypes")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var resourceTypes []models.ResourceType
		err = json.NewDecoder(resp.Body).Decode(&resourceTypes)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Find our created type
		found := false
		for _, rt := range resourceTypes {
			if rt.ResourceTypeID == typeID {
				found = true
				break
			}
		}
		suite.Assert().True(found, "Created resource type should be in the list")
	})

	// Cleanup
	suite.Run("Delete resource type", func() {
		req, err := http.NewRequest("DELETE",
			suite.server.URL+"/o2ims/v1/resourceTypes/"+typeID, nil)
		suite.Require().NoError(err)

		resp, err := suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *O2APITestSuite) TestResourceInstanceOperations() {
	suite.Run("Create and manage resource instances", func() {
		instanceID := "test-instance-" + strconv.FormatInt(time.Now().UnixNano(), 10)

		// Create resource instance
		instance := &models.ResourceInstance{
			ResourceInstanceID:   instanceID,
			ResourceTypeID:       "amf-resource-type",
			ResourcePoolID:       "test-pool-1",
			Name:                 "test-amf-instance",
			Description:          "Test AMF instance for integration testing",
			State:                "INSTANTIATED",
			OperationalStatus:    "ENABLED",
			AdministrativeStatus: "UNLOCKED",
			UsageStatus:          "ACTIVE",
			Metadata: json.RawMessage("{}"),
		}

		instanceJSON, err := json.Marshal(instance)
		suite.Require().NoError(err)

		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourceInstances",
			"application/json",
			bytes.NewBuffer(instanceJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusCreated, resp.StatusCode)

		var created models.ResourceInstance
		err = json.NewDecoder(resp.Body).Decode(&created)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(instanceID, created.ResourceInstanceID)
		suite.Assert().Equal("INSTANTIATED", created.State)
		suite.Assert().NotZero(created.CreatedAt)

		// Get resource instance
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourceInstances/" + instanceID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var retrieved models.ResourceInstance
		err = json.NewDecoder(resp.Body).Decode(&retrieved)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal(instanceID, retrieved.ResourceInstanceID)
		suite.Assert().Equal("test-amf-instance", retrieved.Name)

		// Update resource instance
		retrieved.OperationalStatus = "DISABLED"
		retrieved.Metadata["replicas"] = 1

		updatedJSON, err := json.Marshal(retrieved)
		suite.Require().NoError(err)

		req, err := http.NewRequest("PUT",
			suite.server.URL+"/o2ims/v1/resourceInstances/"+instanceID,
			bytes.NewBuffer(updatedJSON))
		suite.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err = suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Verify update
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourceInstances/" + instanceID)
		suite.Require().NoError(err)

		var updated models.ResourceInstance
		err = json.NewDecoder(resp.Body).Decode(&updated)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().Equal("DISABLED", updated.OperationalStatus)
		suite.Assert().Equal(float64(1), updated.Metadata["replicas"])

		// Delete resource instance
		req, err = http.NewRequest("DELETE",
			suite.server.URL+"/o2ims/v1/resourceInstances/"+instanceID, nil)
		suite.Require().NoError(err)

		resp, err = suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *O2APITestSuite) TestQueryParametersAndFiltering() {
	// Create test data
	pools := []models.ResourcePool{
		{
			ResourcePoolID: "filter-test-1",
			Name:           "Filter Test Pool 1",
			Provider:       "kubernetes",
			Location:       "us-east-1",
			Zone:           "us-east-1a",
		},
		{
			ResourcePoolID: "filter-test-2",
			Name:           "Filter Test Pool 2",
			Provider:       "openstack",
			Location:       "us-east-1",
			Zone:           "us-east-1b",
		},
		{
			ResourcePoolID: "filter-test-3",
			Name:           "Filter Test Pool 3",
			Provider:       "kubernetes",
			Location:       "us-west-2",
			Zone:           "us-west-2a",
		},
	}

	// Create all test pools
	for _, pool := range pools {
		poolJSON, err := json.Marshal(pool)
		suite.Require().NoError(err)

		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourcePools",
			"application/json",
			bytes.NewBuffer(poolJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().True(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK)
		resp.Body.Close()
	}

	// Test filtering
	suite.Run("Filter by provider", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?filter=provider,eq,kubernetes")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var filtered []models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&filtered)
		suite.Require().NoError(err)
		resp.Body.Close()

		count := 0
		for _, pool := range filtered {
			if strings.HasPrefix(pool.ResourcePoolID, "filter-test-") && pool.Provider == "kubernetes" {
				count++
			}
		}
		suite.Assert().Equal(2, count, "Should find 2 kubernetes pools")
	})

	suite.Run("Filter by location", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?filter=location,eq,us-east-1")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var filtered []models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&filtered)
		suite.Require().NoError(err)
		resp.Body.Close()

		count := 0
		for _, pool := range filtered {
			if strings.HasPrefix(pool.ResourcePoolID, "filter-test-") && pool.Location == "us-east-1" {
				count++
			}
		}
		suite.Assert().Equal(2, count, "Should find 2 us-east-1 pools")
	})

	suite.Run("Test pagination", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?limit=1&offset=0")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var page1 []models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&page1)
		suite.Require().NoError(err)
		resp.Body.Close()

		suite.Assert().LessOrEqual(len(page1), 1, "Should return at most 1 result")

		// Test second page
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?limit=1&offset=1")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var page2 []models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&page2)
		suite.Require().NoError(err)
		resp.Body.Close()

		if len(page1) == 1 && len(page2) == 1 {
			suite.Assert().NotEqual(page1[0].ResourcePoolID, page2[0].ResourcePoolID, "Pages should contain different pools")
		}
	})

	// Cleanup
	for _, pool := range pools {
		req, _ := http.NewRequest("DELETE",
			suite.server.URL+"/o2ims/v1/resourcePools/"+pool.ResourcePoolID, nil)
		suite.client.Do(req) // Best effort cleanup
	}
}

func (suite *O2APITestSuite) TestErrorHandling() {
	suite.Run("Handle invalid JSON", func() {
		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourcePools",
			"application/json",
			strings.NewReader(`{"invalid": json}`),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		suite.Require().NoError(err)
		resp.Body.Close()

		var errorResp map[string]interface{}
		err = json.Unmarshal(body, &errorResp)
		suite.Require().NoError(err)

		suite.Assert().Equal("about:blank", errorResp["type"])
		suite.Assert().Equal("Bad Request", errorResp["title"])
		suite.Assert().Equal(float64(400), errorResp["status"])
	})

	suite.Run("Handle non-existent resource", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/non-existent")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusNotFound, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		suite.Require().NoError(err)
		resp.Body.Close()

		var errorResp map[string]interface{}
		err = json.Unmarshal(body, &errorResp)
		suite.Require().NoError(err)

		suite.Assert().Equal(float64(404), errorResp["status"])
		suite.Assert().Equal("Not Found", errorResp["title"])
	})

	suite.Run("Handle invalid query parameters", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?limit=invalid")
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	suite.Run("Handle method not allowed", func() {
		req, err := http.NewRequest("PATCH", suite.server.URL+"/o2ims/v1/", nil)
		suite.Require().NoError(err)

		resp, err := suite.client.Do(req)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusMethodNotAllowed, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *O2APITestSuite) TestConcurrentOperations() {
	suite.Run("Concurrent resource pool operations", func() {
		var wg sync.WaitGroup
		errors := make(chan error, 20)
		successes := make(chan string, 20)

		// Create pools concurrently
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				poolID := fmt.Sprintf("concurrent-test-%d-%d", index, time.Now().UnixNano())
				pool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           fmt.Sprintf("Concurrent Pool %d", index),
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
				}

				poolJSON, err := json.Marshal(pool)
				if err != nil {
					errors <- err
					return
				}

				resp, err := suite.client.Post(
					suite.server.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
					successes <- poolID
				} else {
					errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(successes)

		errorCount := len(errors)
		successCount := len(successes)

		suite.Assert().LessOrEqual(errorCount, 2, "Should have minimal concurrent operation errors")
		suite.Assert().GreaterOrEqual(successCount, 18, "Should have mostly successful operations")

		// Cleanup
		for poolID := range successes {
			req, _ := http.NewRequest("DELETE",
				suite.server.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
			suite.client.Do(req)
		}
	})
}

// DISABLED: func TestO2APIIntegration(t *testing.T) {
	suite.Run(t, new(O2APITestSuite))
}

package o2_compliance_tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// ORANComplianceTestSuite validates O-RAN specification compliance
type ORANComplianceTestSuite struct {
	suite.Suite
	server     *httptest.Server
	client     *http.Client
	o2Server   *o2.O2APIServer
	testLogger *logging.StructuredLogger
}

func (suite *ORANComplianceTestSuite) SetupSuite() {
	suite.testLogger = logging.NewLogger("oran-compliance-test", "debug")

	// Setup O2 IMS server according to O-RAN specifications
	config := &o2.O2IMSConfig{
		ServerAddress:        "127.0.0.1",
		ServerPort:           0,
		TLSEnabled:           false, // TLS would be enabled in production
		DatabaseConfig:       json.RawMessage(`{}`),
		ComplianceMode:       true, // Enable strict O-RAN compliance
		SpecificationVersion: "O-RAN.WG6.O2ims-Interface-v01.01",
	}

	var err error
	suite.o2Server, err = o2.NewO2APIServer(config, suite.testLogger, nil)
	suite.Require().NoError(err)

	suite.server = httptest.NewServer(suite.o2Server.GetRouter())
	suite.client = suite.server.Client()
	suite.client.Timeout = 30 * time.Second
}

func (suite *ORANComplianceTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.o2Server != nil {
		suite.o2Server.Shutdown(context.Background())
	}
}

func (suite *ORANComplianceTestSuite) TestORANServiceInformation() {
	suite.Run("O-RAN Service Information Compliance", func() {
		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
		suite.Require().NoError(err)
		suite.Require().Equal(http.StatusOK, resp.StatusCode)

		var serviceInfo map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
		suite.Require().NoError(err)
		resp.Body.Close()

		// O-RAN.WG6.O2ims-Interface-v01.01 Section 4.2.2 - Service Information
		suite.Assert().Equal("Nephoran O2 IMS", serviceInfo["name"])
		suite.Assert().Equal("O-RAN.WG6.O2ims-Interface-v01.01", serviceInfo["specification"])
		suite.Assert().Equal("v1.0", serviceInfo["apiVersion"])

		// Validate required capabilities per O-RAN spec
		capabilities, ok := serviceInfo["capabilities"].([]interface{})
		suite.Require().True(ok, "capabilities must be an array")

		requiredCapabilities := []string{
			"InfrastructureInventory",
			"InfrastructureMonitoring",
			"InfrastructureProvisioning",
		}

		for _, required := range requiredCapabilities {
			suite.Assert().Contains(capabilities, required,
				fmt.Sprintf("Missing required O-RAN capability: %s", required))
		}

		// Validate service endpoints are present
		suite.Assert().Contains(serviceInfo, "endpoints")
		endpoints, ok := serviceInfo["endpoints"].(map[string]interface{})
		suite.Require().True(ok)

		expectedEndpoints := []string{
			"resourcePools",
			"resourceTypes",
			"resources",
			"deploymentManagers",
			"subscriptions",
			"alarms",
		}

		for _, endpoint := range expectedEndpoints {
			suite.Assert().Contains(endpoints, endpoint,
				fmt.Sprintf("Missing required O-RAN endpoint: %s", endpoint))
		}
	})
}

func (suite *ORANComplianceTestSuite) TestORANResourcePoolCompliance() {
	suite.Run("Resource Pool O-RAN Data Model Compliance", func() {
		// O-RAN.WG6.O2ims-Interface-v01.01 Section 4.3 - Resource Pool Object
		poolID := "oran-compliant-pool-001"

		pool := &models.ResourcePool{
			ResourcePoolID: poolID,
			Name:           "O-RAN Compliant Resource Pool",
			Description:    "Resource pool compliant with O-RAN specifications",
			Location:       "Geographic location identifier",
			OCloudID:       "oran-ocloud-001",
			Provider:       "kubernetes",
			Region:         "us-east-1",
			Zone:           "us-east-1a",
			State:          "AVAILABLE", // Per O-RAN spec: AVAILABLE, UNAVAILABLE, MAINTENANCE
			Capacity: &models.ResourceCapacity{
				CPU: &models.ResourceMetric{
					Total:       "1000",
					Available:   "800",
					Used:        "200",
					Unit:        "cores",
					Utilization: 20.0,
				},
				Memory: &models.ResourceMetric{
					Total:       "2048Gi",
					Available:   "1638Gi",
					Used:        "410Gi",
					Unit:        "bytes",
					Utilization: 20.0,
				},
				Storage: &models.ResourceMetric{
					Total:       "100Ti",
					Available:   "80Ti",
					Used:        "20Ti",
					Unit:        "bytes",
					Utilization: 20.0,
				},
			},
			// O-RAN extensions for additional metadata
			Extensions: json.RawMessage(`{"complianceLevel": "O-RAN-WG6-v1.0"}`),
		}

		poolJSON, err := json.Marshal(pool)
		suite.Require().NoError(err)

		// Create resource pool
		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourcePools",
			"application/json",
			bytes.NewBuffer(poolJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		// Retrieve and validate compliance
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/" + poolID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var retrievedPool models.ResourcePool
		err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Validate O-RAN required fields
		suite.Assert().NotEmpty(retrievedPool.ResourcePoolID, "ResourcePoolID is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedPool.Name, "Name is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedPool.OCloudID, "OCloudID is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedPool.Location, "Location is mandatory per O-RAN spec")

		// Validate state values are O-RAN compliant
		validStates := []string{"AVAILABLE", "UNAVAILABLE", "MAINTENANCE"}
		suite.Assert().Contains(validStates, retrievedPool.State,
			"State must be one of O-RAN defined values: %v", validStates)

		// Validate capacity structure per O-RAN spec
		suite.Assert().NotNil(retrievedPool.Capacity, "Capacity is required per O-RAN spec")
		suite.Assert().NotNil(retrievedPool.Capacity.CPU, "CPU capacity is required")
		suite.Assert().NotNil(retrievedPool.Capacity.Memory, "Memory capacity is required")
		suite.Assert().NotNil(retrievedPool.Capacity.Storage, "Storage capacity is required")

		// Validate metric fields per O-RAN spec
		cpuMetric := retrievedPool.Capacity.CPU
		suite.Assert().NotEmpty(cpuMetric.Total, "CPU Total is required")
		suite.Assert().NotEmpty(cpuMetric.Available, "CPU Available is required")
		suite.Assert().NotEmpty(cpuMetric.Used, "CPU Used is required")
		suite.Assert().NotEmpty(cpuMetric.Unit, "CPU Unit is required")
		suite.Assert().GreaterOrEqual(cpuMetric.Utilization, 0.0, "CPU Utilization must be >= 0")
		suite.Assert().LessOrEqual(cpuMetric.Utilization, 100.0, "CPU Utilization must be <= 100")

		// Validate timestamps are present and properly formatted (ISO 8601)
		suite.Assert().NotZero(retrievedPool.CreatedAt, "CreatedAt timestamp is required")
		suite.Assert().NotZero(retrievedPool.UpdatedAt, "UpdatedAt timestamp is required")
	})
}

func (suite *ORANComplianceTestSuite) TestORANResourceTypeCompliance() {
	suite.Run("Resource Type O-RAN Data Model Compliance", func() {
		// O-RAN.WG6.O2ims-Interface-v01.01 Section 4.4 - Resource Type Object
		typeID := "oran-compute-type-001"

		resourceType := &models.ResourceType{
			ResourceTypeID: typeID,
			Name:           "O-RAN Compute Resource Type",
			Description:    "Compute resource type compliant with O-RAN specifications",
			Vendor:         "Nephoran Systems",
			Model:          "ORAN-Compute-Standard-v1",
			Version:        "1.2.0",
			Specifications: &models.ResourceTypeSpec{
				Category:    "COMPUTE",         // O-RAN defined categories
				SubCategory: "GENERAL_PURPOSE", // O-RAN sub-categories
				MinResources: map[string]string{
					"cpu":               "1000m",
					"memory":            "2Gi",
					"ephemeral-storage": "10Gi",
				},
				MaxResources: map[string]string{
					"cpu":               "32000m",
					"memory":            "256Gi",
					"ephemeral-storage": "1Ti",
				},
				DefaultResources: map[string]string{
					"cpu":               "4000m",
					"memory":            "8Gi",
					"ephemeral-storage": "100Gi",
				},
				RequiredPorts: []models.PortSpec{
					{Name: "management", Port: 22, Protocol: "TCP"},
					{Name: "monitoring", Port: 9090, Protocol: "TCP"},
				},
				OptionalPorts: []models.PortSpec{
					{Name: "debug", Port: 8080, Protocol: "TCP"},
				},
				// O-RAN specific capabilities
				Capabilities: map[string]interface{}{
					"networkAcceleration":   []string{"SRIOV", "DPDK"},
					"storageTypes":          []string{"SSD", "NVMe", "HDD"},
					"virtualizationSupport": true,
					"containerSupport":      true,
				},
			},
			SupportedActions: []string{"CREATE", "DELETE", "UPDATE", "SCALE", "HEAL", "BACKUP", "RESTORE"},
			// O-RAN compliance metadata
			Compliance: &models.ComplianceInfo{
				Standard:           "O-RAN-WG6-v1.0.1",
				CertificationLevel: "CERTIFIED",
				TestResults:        json.RawMessage(`{}`),
			},
		}

		typeJSON, err := json.Marshal(resourceType)
		suite.Require().NoError(err)

		// Create resource type
		resp, err := suite.client.Post(
			suite.server.URL+"/o2ims/v1/resourceTypes",
			"application/json",
			bytes.NewBuffer(typeJSON),
		)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		// Retrieve and validate compliance
		resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourceTypes/" + typeID)
		suite.Require().NoError(err)
		suite.Assert().Equal(http.StatusOK, resp.StatusCode)

		var retrievedType models.ResourceType
		err = json.NewDecoder(resp.Body).Decode(&retrievedType)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Validate O-RAN required fields
		suite.Assert().NotEmpty(retrievedType.ResourceTypeID, "ResourceTypeID is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedType.Name, "Name is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedType.Vendor, "Vendor is mandatory per O-RAN spec")
		suite.Assert().NotEmpty(retrievedType.Version, "Version is mandatory per O-RAN spec")

		// Validate category values are O-RAN compliant
		validCategories := []string{"COMPUTE", "STORAGE", "NETWORK", "ACCELERATOR"}
		suite.Assert().Contains(validCategories, retrievedType.Specifications.Category,
			"Category must be one of O-RAN defined values: %v", validCategories)

		// Validate supported actions are O-RAN compliant
		requiredActions := []string{"CREATE", "DELETE"}
		for _, action := range requiredActions {
			suite.Assert().Contains(retrievedType.SupportedActions, action,
				fmt.Sprintf("Missing required O-RAN action: %s", action))
		}

		// Validate resource specifications structure
		specs := retrievedType.Specifications
		suite.Assert().NotNil(specs.MinResources, "MinResources is required per O-RAN spec")
		suite.Assert().NotNil(specs.MaxResources, "MaxResources is required per O-RAN spec")
		suite.Assert().NotNil(specs.DefaultResources, "DefaultResources is required per O-RAN spec")

		// Validate resource keys are standard
		standardResourceKeys := []string{"cpu", "memory", "ephemeral-storage"}
		for _, key := range standardResourceKeys {
			suite.Assert().Contains(specs.MinResources, key,
				fmt.Sprintf("MinResources missing standard key: %s", key))
			suite.Assert().Contains(specs.MaxResources, key,
				fmt.Sprintf("MaxResources missing standard key: %s", key))
			suite.Assert().Contains(specs.DefaultResources, key,
				fmt.Sprintf("DefaultResources missing standard key: %s", key))
		}
	})
}

func (suite *ORANComplianceTestSuite) TestORANAPIEndpointsCompliance() {
	suite.Run("O-RAN API Endpoints Structure Compliance", func() {
		// O-RAN.WG6.O2ims-Interface-v01.01 Section 5 - API Endpoints
		requiredEndpoints := map[string][]string{
			// Resource Pool endpoints
			"/o2ims/v1/resourcePools":                  {"GET", "POST"},
			"/o2ims/v1/resourcePools/{resourcePoolId}": {"GET", "PUT", "DELETE"},

			// Resource Type endpoints
			"/o2ims/v1/resourceTypes":                  {"GET", "POST"},
			"/o2ims/v1/resourceTypes/{resourceTypeId}": {"GET", "PUT", "DELETE"},

			// Resource Instance endpoints
			"/o2ims/v1/resources":              {"GET", "POST"},
			"/o2ims/v1/resources/{resourceId}": {"GET", "PUT", "DELETE"},

			// Deployment Manager endpoints
			"/o2ims/v1/deploymentManagers":                       {"GET", "POST"},
			"/o2ims/v1/deploymentManagers/{deploymentManagerId}": {"GET", "PUT", "DELETE"},

			// Subscription endpoints
			"/o2ims/v1/subscriptions":                  {"GET", "POST"},
			"/o2ims/v1/subscriptions/{subscriptionId}": {"GET", "DELETE"},

			// Alarm endpoints
			"/o2ims/v1/alarms":           {"GET", "POST"},
			"/o2ims/v1/alarms/{alarmId}": {"GET", "PATCH", "DELETE"},
		}

		for endpoint, methods := range requiredEndpoints {
			for _, method := range methods {
				suite.Run(fmt.Sprintf("%s %s", method, endpoint), func() {
					// Test basic endpoint availability
					// Note: We test with concrete IDs for parameterized endpoints
					testURL := suite.server.URL + strings.Replace(endpoint, "{resourcePoolId}", "test-pool", -1)
					testURL = strings.Replace(testURL, "{resourceTypeId}", "test-type", -1)
					testURL = strings.Replace(testURL, "{resourceId}", "test-resource", -1)
					testURL = strings.Replace(testURL, "{deploymentManagerId}", "test-dm", -1)
					testURL = strings.Replace(testURL, "{subscriptionId}", "test-sub", -1)
					testURL = strings.Replace(testURL, "{alarmId}", "test-alarm", -1)

					req, err := http.NewRequest(method, testURL, nil)
					suite.Require().NoError(err)

					resp, err := suite.client.Do(req)
					suite.Require().NoError(err)
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					// Endpoint should exist (not return 404) and method should be allowed (not 405)
					suite.Assert().NotEqual(http.StatusNotFound, resp.StatusCode,
						"Endpoint %s should exist per O-RAN spec", endpoint)
					suite.Assert().NotEqual(http.StatusMethodNotAllowed, resp.StatusCode,
						"Method %s should be allowed on endpoint %s per O-RAN spec", method, endpoint)
				})
			}
		}
	})
}

func (suite *ORANComplianceTestSuite) TestORANQueryParametersCompliance() {
	suite.Run("O-RAN Query Parameters Compliance", func() {
		// O-RAN.WG6.O2ims-Interface-v01.01 Section 5.2 - Query Parameters

		// Create test resource pools for filtering
		testPools := []models.ResourcePool{
			{
				ResourcePoolID: "filter-test-pool-1",
				Name:           "Filter Test Pool 1",
				Provider:       "kubernetes",
				Location:       "us-east-1",
				State:          "AVAILABLE",
			},
			{
				ResourcePoolID: "filter-test-pool-2",
				Name:           "Filter Test Pool 2",
				Provider:       "openstack",
				Location:       "us-west-2",
				State:          "UNAVAILABLE",
			},
		}

		for _, pool := range testPools {
			poolJSON, err := json.Marshal(pool)
			suite.Require().NoError(err)

			resp, err := suite.client.Post(
				suite.server.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			suite.Require().NoError(err)
			resp.Body.Close()
		}

		// Test O-RAN compliant filter syntax: (eq,field,value)
		suite.Run("Filter Syntax Compliance", func() {
			testCases := []struct {
				name          string
				filter        string
				expectedCount int
			}{
				{
					name:          "Equal filter",
					filter:        "(eq,provider,kubernetes)",
					expectedCount: 1,
				},
				{
					name:          "Multiple filters with AND",
					filter:        "(eq,provider,kubernetes);(eq,state,AVAILABLE)",
					expectedCount: 1,
				},
				{
					name:          "Location filter",
					filter:        "(eq,location,us-west-2)",
					expectedCount: 1,
				},
			}

			for _, tc := range testCases {
				suite.Run(tc.name, func() {
					url := fmt.Sprintf("%s/o2ims/v1/resourcePools?filter=%s", suite.server.URL, tc.filter)
					resp, err := suite.client.Get(url)
					suite.Require().NoError(err)
					suite.Assert().Equal(http.StatusOK, resp.StatusCode)

					var pools []models.ResourcePool
					err = json.NewDecoder(resp.Body).Decode(&pools)
					suite.Require().NoError(err)
					resp.Body.Close()

					// Count matching pools
					matchingCount := 0
					for _, pool := range pools {
						if strings.HasPrefix(pool.ResourcePoolID, "filter-test-pool-") {
							matchingCount++
						}
					}

					suite.Assert().Equal(tc.expectedCount, matchingCount,
						"Filter %s should return %d results", tc.filter, tc.expectedCount)
				})
			}
		})

		// Test pagination parameters per O-RAN spec
		suite.Run("Pagination Parameters Compliance", func() {
			// Test limit parameter
			resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?limit=1")
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusOK, resp.StatusCode)

			var limitedPools []models.ResourcePool
			err = json.NewDecoder(resp.Body).Decode(&limitedPools)
			suite.Require().NoError(err)
			resp.Body.Close()

			suite.Assert().LessOrEqual(len(limitedPools), 1, "limit=1 should return at most 1 result")

			// Test offset parameter
			resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools?offset=1&limit=1")
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusOK, resp.StatusCode)

			var offsetPools []models.ResourcePool
			err = json.NewDecoder(resp.Body).Decode(&offsetPools)
			suite.Require().NoError(err)
			resp.Body.Close()

			suite.Assert().LessOrEqual(len(offsetPools), 1, "offset=1&limit=1 should return at most 1 result")
		})
	})
}

func (suite *ORANComplianceTestSuite) TestORANErrorResponsesCompliance() {
	suite.Run("O-RAN Error Response Format Compliance", func() {
		// O-RAN.WG6.O2ims-Interface-v01.01 Section 6 - Error Handling

		suite.Run("404 Not Found Error Format", func() {
			resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/non-existent-pool")
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusNotFound, resp.StatusCode)

			var errorResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			suite.Require().NoError(err)
			resp.Body.Close()

			// Validate O-RAN error response structure
			suite.Assert().Contains(errorResponse, "type", "Error response must contain 'type' field")
			suite.Assert().Contains(errorResponse, "title", "Error response must contain 'title' field")
			suite.Assert().Contains(errorResponse, "status", "Error response must contain 'status' field")
			suite.Assert().Contains(errorResponse, "detail", "Error response must contain 'detail' field")

			// Validate specific field values
			suite.Assert().Equal(float64(404), errorResponse["status"])
			suite.Assert().Equal("Not Found", errorResponse["title"])
		})

		suite.Run("400 Bad Request Error Format", func() {
			// Send invalid JSON
			resp, err := suite.client.Post(
				suite.server.URL+"/o2ims/v1/resourcePools",
				"application/json",
				strings.NewReader(`{"invalid": json}`),
			)
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

			var errorResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			suite.Require().NoError(err)
			resp.Body.Close()

			// Validate error structure
			suite.Assert().Equal(float64(400), errorResponse["status"])
			suite.Assert().Equal("Bad Request", errorResponse["title"])
			suite.Assert().Contains(errorResponse, "detail")
		})

		suite.Run("405 Method Not Allowed Error Format", func() {
			// Try PATCH on service info endpoint (should be GET only)
			req, err := http.NewRequest("PATCH", suite.server.URL+"/o2ims/v1/", nil)
			suite.Require().NoError(err)

			resp, err := suite.client.Do(req)
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusMethodNotAllowed, resp.StatusCode)

			// Should include Allow header per HTTP spec
			allowHeader := resp.Header.Get("Allow")
			suite.Assert().NotEmpty(allowHeader, "405 response should include Allow header")
			suite.Assert().Contains(allowHeader, "GET", "Allow header should include GET method")
		})
	})
}

func (suite *ORANComplianceTestSuite) TestORANContentTypeCompliance() {
	suite.Run("O-RAN Content-Type Header Compliance", func() {
		// O-RAN spec requires application/json content type

		suite.Run("JSON Content Type Required for POST", func() {
			pool := &models.ResourcePool{
				ResourcePoolID: "content-type-test-pool",
				Name:           "Content Type Test Pool",
				Provider:       "kubernetes",
				OCloudID:       "test-ocloud",
			}

			poolJSON, err := json.Marshal(pool)
			suite.Require().NoError(err)

			// Test with correct content type
			resp, err := suite.client.Post(
				suite.server.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			suite.Require().NoError(err)
			suite.Assert().True(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK)
			resp.Body.Close()

			// Test with incorrect content type
			resp, err = suite.client.Post(
				suite.server.URL+"/o2ims/v1/resourcePools",
				"text/plain",
				bytes.NewBuffer(poolJSON),
			)
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusUnsupportedMediaType, resp.StatusCode,
				"Should reject non-JSON content type")
			resp.Body.Close()
		})

		suite.Run("Response Content Type is JSON", func() {
			resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
			suite.Require().NoError(err)
			defer resp.Body.Close() // #nosec G307 - Error handled in defer

			contentType := resp.Header.Get("Content-Type")
			suite.Assert().Contains(contentType, "application/json",
				"Response Content-Type should be application/json")
		})
	})
}

func (suite *ORANComplianceTestSuite) TestORANVersioningCompliance() {
	suite.Run("O-RAN API Versioning Compliance", func() {
		// O-RAN spec requires versioned API endpoints

		suite.Run("API Version in URL Path", func() {
			// Test that v1 endpoints are accessible
			resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusOK, resp.StatusCode)
			resp.Body.Close()

			// Test that unversioned endpoints are not accessible
			resp, err = suite.client.Get(suite.server.URL + "/o2ims/")
			suite.Require().NoError(err)
			suite.Assert().Equal(http.StatusNotFound, resp.StatusCode,
				"Unversioned API endpoints should not be accessible")
			resp.Body.Close()
		})

		suite.Run("API Version in Response", func() {
			resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
			suite.Require().NoError(err)
			defer resp.Body.Close() // #nosec G307 - Error handled in defer

			var serviceInfo map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
			suite.Require().NoError(err)

			apiVersion, exists := serviceInfo["apiVersion"]
			suite.Assert().True(exists, "API version should be included in response")
			suite.Assert().Equal("v1.0", apiVersion, "API version should match expected value")
		})
	})
}

func (suite *ORANComplianceTestSuite) TestORANSecurityHeadersCompliance() {
	suite.Run("O-RAN Security Headers Compliance", func() {
		// While TLS is disabled in test, validate security header structure

		resp, err := suite.client.Get(suite.server.URL + "/o2ims/v1/")
		suite.Require().NoError(err)
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		// Check for security-related headers that should be present
		headers := resp.Header

		// Content Security Policy (if implemented)
		if csp := headers.Get("Content-Security-Policy"); csp != "" {
			suite.Assert().Contains(csp, "default-src", "CSP should include default-src directive")
		}

		// X-Frame-Options (if implemented)
		if xfo := headers.Get("X-Frame-Options"); xfo != "" {
			validValues := []string{"DENY", "SAMEORIGIN"}
			suite.Assert().Contains(validValues, xfo, "X-Frame-Options should be DENY or SAMEORIGIN")
		}

		// X-Content-Type-Options (if implemented)
		if xcto := headers.Get("X-Content-Type-Options"); xcto != "" {
			suite.Assert().Equal("nosniff", xcto, "X-Content-Type-Options should be nosniff")
		}

		// In production, these would be mandatory:
		// - Strict-Transport-Security (HTTPS only)
		// - X-Frame-Options
		// - Content-Security-Policy
		// - X-Content-Type-Options
	})
}

func (suite *ORANComplianceTestSuite) TestORANDataModelConsistency() {
	suite.Run("O-RAN Data Model Consistency Validation", func() {
		// Test that all data models follow O-RAN naming conventions and structure

		suite.Run("Resource Pool Data Model Consistency", func() {
			pool := &models.ResourcePool{
				ResourcePoolID: "consistency-test-pool",
				Name:           "Consistency Test Pool",
				Provider:       "kubernetes",
				OCloudID:       "consistency-test-ocloud",
				// Test with all optional fields to validate structure
				Description: "Test pool for data model consistency",
				Location:    "test-location",
				Region:      "test-region",
				Zone:        "test-zone",
				State:       "AVAILABLE",
				Capacity: &models.ResourceCapacity{
					CPU: &models.ResourceMetric{
						Total: "100", Available: "80", Used: "20", Unit: "cores", Utilization: 20.0,
					},
					Memory: &models.ResourceMetric{
						Total: "400Gi", Available: "320Gi", Used: "80Gi", Unit: "bytes", Utilization: 20.0,
					},
					Storage: &models.ResourceMetric{
						Total: "1Ti", Available: "800Gi", Used: "200Gi", Unit: "bytes", Utilization: 20.0,
					},
				},
				Extensions: json.RawMessage(`{}`),
			}

			poolJSON, err := json.Marshal(pool)
			suite.Require().NoError(err)

			// Create pool
			resp, err := suite.client.Post(
				suite.server.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			suite.Require().NoError(err)
			suite.Assert().True(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK)
			resp.Body.Close()

			// Retrieve and validate structure preservation
			resp, err = suite.client.Get(suite.server.URL + "/o2ims/v1/resourcePools/consistency-test-pool")
			suite.Require().NoError(err)

			var retrievedPool models.ResourcePool
			err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
			suite.Require().NoError(err)
			resp.Body.Close()

			// Validate all fields are preserved and correctly typed
			suite.Assert().Equal(pool.ResourcePoolID, retrievedPool.ResourcePoolID)
			suite.Assert().Equal(pool.Name, retrievedPool.Name)
			suite.Assert().Equal(pool.Description, retrievedPool.Description)
			suite.Assert().Equal(pool.Location, retrievedPool.Location)
			suite.Assert().Equal(pool.Provider, retrievedPool.Provider)
			suite.Assert().Equal(pool.OCloudID, retrievedPool.OCloudID)
			suite.Assert().Equal(pool.State, retrievedPool.State)

			// Validate nested capacity structure
			suite.Assert().NotNil(retrievedPool.Capacity)
			suite.Assert().Equal(pool.Capacity.CPU.Total, retrievedPool.Capacity.CPU.Total)
			suite.Assert().Equal(pool.Capacity.Memory.Unit, retrievedPool.Capacity.Memory.Unit)
			suite.Assert().Equal(pool.Capacity.Storage.Utilization, retrievedPool.Capacity.Storage.Utilization)

			// Validate extensions are preserved
			suite.Assert().Equal(pool.Extensions["customField1"], retrievedPool.Extensions["customField1"])
			suite.Assert().Equal(pool.Extensions["customField2"], retrievedPool.Extensions["customField2"])

			// Validate timestamps are properly set
			suite.Assert().NotZero(retrievedPool.CreatedAt)
			suite.Assert().NotZero(retrievedPool.UpdatedAt)
		})
	})
}

func TestORANCompliance(t *testing.T) {
	suite.Run(t, new(ORANComplianceTestSuite))
}

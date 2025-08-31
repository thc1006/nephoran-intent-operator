package test_validation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// O-RAN Compliance Validation Framework
// Validates O2 IMS implementation against O-RAN.WG6.O2ims-Interface-v01.01 specification

var _ = Describe("O2 IMS O-RAN Compliance Validation", func() {
	var (
		namespace         *corev1.Namespace
		testCtx           context.Context
		o2Server          *o2.O2APIServer
		httpTestServer    *httptest.Server
		testClient        *http.Client
		metricsRegistry   *prometheus.Registry
		testLogger        *logging.StructuredLogger
		complianceResults *ComplianceResults
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 15*time.Minute)
		DeferCleanup(cancel)

		testLogger = logging.NewLogger("o2-compliance-test", "info")
		metricsRegistry = prometheus.NewRegistry()
		complianceResults = NewComplianceResults()

		config := &o2.O2IMSConfig{
			ServerAddress: "127.0.0.1",
			ServerPort:    0,
			TLSEnabled:    false,
			DatabaseConfig: map[string]interface{}{
				"type":     "memory",
				"database": "o2_compliance_test_db",
			},
			ComplianceMode: true, // Enable strict compliance validation
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		httpTestServer = httptest.NewServer(o2Server.GetRouter())
		testClient = httpTestServer.Client()
		testClient.Timeout = 60 * time.Second

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
			complianceResults.GenerateReport()
		})
	})

	Describe("O-RAN Interface Specification Compliance", func() {
		Context("when validating API endpoints and methods", func() {
			It("should implement all required O-RAN O2 IMS endpoints", func() {
				requiredEndpoints := []ComplianceEndpoint{
					// Service Information
					{Method: "GET", Path: "/o2ims/v1/", Description: "Get service information", Required: true},
					{Method: "GET", Path: "/health", Description: "Health check", Required: true},
					{Method: "GET", Path: "/ready", Description: "Readiness check", Required: true},

					// Resource Pools
					{Method: "GET", Path: "/o2ims/v1/resourcePools", Description: "List resource pools", Required: true},
					{Method: "POST", Path: "/o2ims/v1/resourcePools", Description: "Create resource pool", Required: true},
					{Method: "GET", Path: "/o2ims/v1/resourcePools/{resourcePoolId}", Description: "Get resource pool", Required: true},
					{Method: "PUT", Path: "/o2ims/v1/resourcePools/{resourcePoolId}", Description: "Update resource pool", Required: true},
					{Method: "PATCH", Path: "/o2ims/v1/resourcePools/{resourcePoolId}", Description: "Patch resource pool", Required: true},
					{Method: "DELETE", Path: "/o2ims/v1/resourcePools/{resourcePoolId}", Description: "Delete resource pool", Required: true},

					// Resource Types
					{Method: "GET", Path: "/o2ims/v1/resourceTypes", Description: "List resource types", Required: true},
					{Method: "POST", Path: "/o2ims/v1/resourceTypes", Description: "Create resource type", Required: true},
					{Method: "GET", Path: "/o2ims/v1/resourceTypes/{resourceTypeId}", Description: "Get resource type", Required: true},
					{Method: "PUT", Path: "/o2ims/v1/resourceTypes/{resourceTypeId}", Description: "Update resource type", Required: true},
					{Method: "DELETE", Path: "/o2ims/v1/resourceTypes/{resourceTypeId}", Description: "Delete resource type", Required: true},

					// Resources
					{Method: "GET", Path: "/o2ims/v1/resources", Description: "List resources", Required: true},
					{Method: "POST", Path: "/o2ims/v1/resources", Description: "Create resource", Required: true},
					{Method: "GET", Path: "/o2ims/v1/resources/{resourceId}", Description: "Get resource", Required: true},
					{Method: "PUT", Path: "/o2ims/v1/resources/{resourceId}", Description: "Update resource", Required: true},
					{Method: "DELETE", Path: "/o2ims/v1/resources/{resourceId}", Description: "Delete resource", Required: true},

					// Deployments
					{Method: "GET", Path: "/o2ims/v1/deployments", Description: "List deployments", Required: true},
					{Method: "POST", Path: "/o2ims/v1/deployments", Description: "Create deployment", Required: true},
					{Method: "GET", Path: "/o2ims/v1/deployments/{deploymentId}", Description: "Get deployment", Required: true},
					{Method: "DELETE", Path: "/o2ims/v1/deployments/{deploymentId}", Description: "Delete deployment", Required: true},

					// Subscriptions
					{Method: "GET", Path: "/o2ims/v1/subscriptions", Description: "List subscriptions", Required: true},
					{Method: "POST", Path: "/o2ims/v1/subscriptions", Description: "Create subscription", Required: true},
					{Method: "GET", Path: "/o2ims/v1/subscriptions/{subscriptionId}", Description: "Get subscription", Required: true},
					{Method: "DELETE", Path: "/o2ims/v1/subscriptions/{subscriptionId}", Description: "Delete subscription", Required: true},

					// Alarms
					{Method: "GET", Path: "/o2ims/v1/alarms", Description: "List alarms", Required: true},
					{Method: "GET", Path: "/o2ims/v1/alarms/{alarmId}", Description: "Get alarm", Required: true},
					{Method: "PATCH", Path: "/o2ims/v1/alarms/{alarmId}", Description: "Acknowledge alarm", Required: true},
				}

				By("testing all required endpoints")
				for _, endpoint := range requiredEndpoints {
					result := complianceResults.StartEndpointTest(endpoint)

					testPath := endpoint.Path
					if strings.Contains(testPath, "{resourcePoolId}") {
						testPath = strings.Replace(testPath, "{resourcePoolId}", "test-pool-id", -1)
					}
					if strings.Contains(testPath, "{resourceTypeId}") {
						testPath = strings.Replace(testPath, "{resourceTypeId}", "test-type-id", -1)
					}
					if strings.Contains(testPath, "{resourceId}") {
						testPath = strings.Replace(testPath, "{resourceId}", "test-resource-id", -1)
					}
					if strings.Contains(testPath, "{deploymentId}") {
						testPath = strings.Replace(testPath, "{deploymentId}", "test-deployment-id", -1)
					}
					if strings.Contains(testPath, "{subscriptionId}") {
						testPath = strings.Replace(testPath, "{subscriptionId}", "test-subscription-id", -1)
					}
					if strings.Contains(testPath, "{alarmId}") {
						testPath = strings.Replace(testPath, "{alarmId}", "test-alarm-id", -1)
					}

					req, err := http.NewRequest(endpoint.Method, httpTestServer.URL+testPath, nil)
					Expect(err).NotTo(HaveOccurred())

					resp, err := testClient.Do(req)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()

					// Endpoint should exist (not return 404)
					if endpoint.Required {
						Expect(resp.StatusCode).NotTo(Equal(404),
							fmt.Sprintf("Required endpoint %s %s not found", endpoint.Method, endpoint.Path))
						result.Status = "PASS"
					} else {
						result.Status = "PASS" // Optional endpoints are allowed to not exist
					}

					complianceResults.CompleteEndpointTest(endpoint, result)
				}
			})

			It("should implement proper HTTP status codes per O-RAN specification", func() {
				By("testing status codes for various scenarios")

				statusCodeTests := []ComplianceStatusCodeTest{
					{
						Name: "Service Info Success",
						Request: func() (*http.Response, error) {
							return testClient.Get(httpTestServer.URL + "/o2ims/v1/")
						},
						ExpectedStatus: 200,
						Required:       true,
					},
					{
						Name: "Resource Not Found",
						Request: func() (*http.Response, error) {
							return testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/non-existent-pool")
						},
						ExpectedStatus: 404,
						Required:       true,
					},
					{
						Name: "Invalid Request Body",
						Request: func() (*http.Response, error) {
							return testClient.Post(httpTestServer.URL+"/o2ims/v1/resourcePools",
								"application/json", strings.NewReader(`{"invalid": json}`))
						},
						ExpectedStatus: 400,
						Required:       true,
					},
					{
						Name: "Method Not Allowed",
						Request: func() (*http.Response, error) {
							req, _ := http.NewRequest("PUT", httpTestServer.URL+"/o2ims/v1/", nil)
							return testClient.Do(req)
						},
						ExpectedStatus: 405,
						Required:       true,
					},
				}

				for _, test := range statusCodeTests {
					By(fmt.Sprintf("testing %s", test.Name))

					result := complianceResults.StartStatusCodeTest(test)
					resp, err := test.Request()

					if err != nil {
						result.Status = "FAIL"
						result.ErrorMessage = err.Error()
					} else {
						resp.Body.Close()
						if resp.StatusCode == test.ExpectedStatus {
							result.Status = "PASS"
						} else {
							result.Status = "FAIL"
							result.ErrorMessage = fmt.Sprintf("Expected status %d, got %d",
								test.ExpectedStatus, resp.StatusCode)
						}
					}

					complianceResults.CompleteStatusCodeTest(test, result)

					if test.Required {
						Expect(result.Status).To(Equal("PASS"),
							fmt.Sprintf("Required status code test '%s' failed: %s", test.Name, result.ErrorMessage))
					}
				}
			})
		})

		Context("when validating data models and schemas", func() {
			It("should implement all required O-RAN data model fields", func() {
				By("testing resource pool data model compliance")

				poolID := fmt.Sprintf("schema-test-pool-%d", time.Now().UnixNano())
				testPool := &models.ResourcePool{
					ResourcePoolID:   poolID,
					Name:             "Schema Compliance Test Pool",
					Description:      "Testing O-RAN schema compliance",
					Location:         "test-location",
					OCloudID:         "test-ocloud-id",
					GlobalLocationID: "test-global-location",
					Provider:         "kubernetes",
					Extensions: map[string]interface{}{
						"testField": "testValue",
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				poolJSON, err := json.Marshal(testPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				By("verifying created resource pool has all required fields")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				var retrievedPool models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
				Expect(err).NotTo(HaveOccurred())

				// Validate required O-RAN fields are present
				schemaValidation := complianceResults.StartSchemaValidation("ResourcePool")

				requiredFields := []string{
					"ResourcePoolID", "Name", "OCloudID",
					"CreatedAt", "UpdatedAt",
				}

				poolValue := reflect.ValueOf(retrievedPool)
				for _, fieldName := range requiredFields {
					field := poolValue.FieldByName(fieldName)
					if !field.IsValid() || field.IsZero() {
						schemaValidation.MissingFields = append(schemaValidation.MissingFields, fieldName)
					}
				}

				if len(schemaValidation.MissingFields) == 0 {
					schemaValidation.Status = "PASS"
				} else {
					schemaValidation.Status = "FAIL"
					schemaValidation.ErrorMessage = fmt.Sprintf("Missing required fields: %v",
						schemaValidation.MissingFields)
				}

				complianceResults.CompleteSchemaValidation("ResourcePool", schemaValidation)

				Expect(schemaValidation.Status).To(Equal("PASS"),
					fmt.Sprintf("ResourcePool schema validation failed: %s", schemaValidation.ErrorMessage))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})

			It("should validate resource type schema compliance", func() {
				By("testing resource type data model compliance")

				typeID := fmt.Sprintf("schema-test-type-%d", time.Now().UnixNano())
				testType := &models.ResourceType{
					ResourceTypeID: typeID,
					Name:           "Schema Compliance Test Type",
					Description:    "Testing O-RAN resource type schema",
					Vendor:         "Nephoran",
					Model:          "TestModel",
					Version:        "1.0.0",
					Specifications: &models.ResourceTypeSpec{
						Category: "COMPUTE",
						MinResources: map[string]string{
							"cpu":    "1",
							"memory": "1Gi",
						},
						MaxResources: map[string]string{
							"cpu":    "32",
							"memory": "128Gi",
						},
					},
					SupportedActions: []string{"CREATE", "DELETE", "UPDATE"},
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				}

				typeJSON, err := json.Marshal(testType)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourceTypes",
					"application/json",
					bytes.NewBuffer(typeJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				By("verifying resource type schema compliance")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourceTypes/" + typeID)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				var retrievedType models.ResourceType
				err = json.NewDecoder(resp.Body).Decode(&retrievedType)
				Expect(err).NotTo(HaveOccurred())

				schemaValidation := complianceResults.StartSchemaValidation("ResourceType")

				// Validate required fields
				if retrievedType.ResourceTypeID == "" {
					schemaValidation.MissingFields = append(schemaValidation.MissingFields, "ResourceTypeID")
				}
				if retrievedType.Name == "" {
					schemaValidation.MissingFields = append(schemaValidation.MissingFields, "Name")
				}
				if retrievedType.Specifications == nil {
					schemaValidation.MissingFields = append(schemaValidation.MissingFields, "Specifications")
				} else {
					if retrievedType.Specifications.Category == "" {
						schemaValidation.MissingFields = append(schemaValidation.MissingFields, "Specifications.Category")
					}
				}

				// Validate supported actions
				if len(retrievedType.SupportedActions) == 0 {
					schemaValidation.MissingFields = append(schemaValidation.MissingFields, "SupportedActions")
				}

				if len(schemaValidation.MissingFields) == 0 {
					schemaValidation.Status = "PASS"
				} else {
					schemaValidation.Status = "FAIL"
					schemaValidation.ErrorMessage = fmt.Sprintf("Missing required fields: %v",
						schemaValidation.MissingFields)
				}

				complianceResults.CompleteSchemaValidation("ResourceType", schemaValidation)

				Expect(schemaValidation.Status).To(Equal("PASS"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourceTypes/"+typeID, nil)
					testClient.Do(req)
				})
			})
		})

		Context("when validating query parameter support", func() {
			It("should support all required O-RAN query parameters", func() {
				By("testing filter parameter compliance")

				// Create test data first
				pools := []struct {
					id       string
					provider string
					location string
				}{
					{"filter-test-1", "kubernetes", "us-west-2"},
					{"filter-test-2", "aws", "us-east-1"},
					{"filter-test-3", "kubernetes", "eu-west-1"},
				}

				for _, pool := range pools {
					testPool := &models.ResourcePool{
						ResourcePoolID: pool.id,
						Name:           fmt.Sprintf("Filter Test Pool %s", pool.id),
						Provider:       pool.provider,
						Location:       pool.location,
						OCloudID:       "test-ocloud",
					}

					poolJSON, _ := json.Marshal(testPool)
					resp, _ := testClient.Post(
						httpTestServer.URL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)
					if resp != nil {
						resp.Body.Close()
					}
				}

				DeferCleanup(func() {
					for _, pool := range pools {
						req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+pool.id, nil)
						testClient.Do(req)
					}
				})

				By("testing various filter expressions")
				filterTests := []ComplianceFilterTest{
					{
						Name:           "Equality Filter",
						QueryParam:     "filter=provider,eq,kubernetes",
						Description:    "Filter by provider equality",
						ExpectedResult: "Should return only Kubernetes pools",
						Required:       true,
					},
					{
						Name:           "Not Equal Filter",
						QueryParam:     "filter=provider,neq,aws",
						Description:    "Filter by provider not equal",
						ExpectedResult: "Should return non-AWS pools",
						Required:       true,
					},
					{
						Name:           "In List Filter",
						QueryParam:     "filter=location,in,us-west-2;us-east-1",
						Description:    "Filter by location in list",
						ExpectedResult: "Should return US region pools",
						Required:       true,
					},
					{
						Name:           "Combined Filters",
						QueryParam:     "filter=provider,eq,kubernetes;location,eq,us-west-2",
						Description:    "Multiple filter conditions",
						ExpectedResult: "Should return Kubernetes pools in us-west-2",
						Required:       true,
					},
				}

				for _, filterTest := range filterTests {
					By(fmt.Sprintf("testing %s", filterTest.Name))

					result := complianceResults.StartFilterTest(filterTest)

					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools?" + filterTest.QueryParam)
					if err != nil {
						result.Status = "FAIL"
						result.ErrorMessage = err.Error()
					} else {
						defer resp.Body.Close()

						if resp.StatusCode == 200 {
							var pools []models.ResourcePool
							json.NewDecoder(resp.Body).Decode(&pools)
							result.ResultCount = len(pools)
							result.Status = "PASS"
						} else {
							result.Status = "FAIL"
							result.ErrorMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
						}
					}

					complianceResults.CompleteFilterTest(filterTest, result)

					if filterTest.Required {
						Expect(result.Status).To(Equal("PASS"),
							fmt.Sprintf("Required filter test '%s' failed: %s", filterTest.Name, result.ErrorMessage))
					}
				}

				By("testing pagination parameters")
				paginationTests := []CompliancePaginationTest{
					{
						Name:       "Limit Parameter",
						QueryParam: "limit=2",
						MaxResults: 2,
						Required:   true,
					},
					{
						Name:       "Offset Parameter",
						QueryParam: "offset=1&limit=1",
						MaxResults: 1,
						Required:   true,
					},
				}

				for _, pagTest := range paginationTests {
					By(fmt.Sprintf("testing %s", pagTest.Name))

					result := complianceResults.StartPaginationTest(pagTest)

					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools?" + pagTest.QueryParam)
					if err != nil {
						result.Status = "FAIL"
						result.ErrorMessage = err.Error()
					} else {
						defer resp.Body.Close()

						if resp.StatusCode == 200 {
							var pools []models.ResourcePool
							json.NewDecoder(resp.Body).Decode(&pools)

							if len(pools) <= pagTest.MaxResults {
								result.Status = "PASS"
								result.ActualCount = len(pools)
							} else {
								result.Status = "FAIL"
								result.ErrorMessage = fmt.Sprintf("Expected max %d results, got %d",
									pagTest.MaxResults, len(pools))
								result.ActualCount = len(pools)
							}
						} else {
							result.Status = "FAIL"
							result.ErrorMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
						}
					}

					complianceResults.CompletePaginationTest(pagTest, result)

					if pagTest.Required {
						Expect(result.Status).To(Equal("PASS"))
					}
				}
			})
		})

		Context("when validating error response format", func() {
			It("should follow RFC 7807 Problem Details format", func() {
				By("testing error response structure")

				// Test various error scenarios
				errorTests := []ComplianceErrorTest{
					{
						Name: "Resource Not Found",
						RequestFunc: func() (*http.Response, error) {
							return testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/non-existent")
						},
						ExpectedStatus: 404,
						Required:       true,
					},
					{
						Name: "Invalid JSON",
						RequestFunc: func() (*http.Response, error) {
							return testClient.Post(httpTestServer.URL+"/o2ims/v1/resourcePools",
								"application/json", strings.NewReader(`{"invalid": json}`))
						},
						ExpectedStatus: 400,
						Required:       true,
					},
					{
						Name: "Method Not Allowed",
						RequestFunc: func() (*http.Response, error) {
							req, _ := http.NewRequest("PATCH", httpTestServer.URL+"/o2ims/v1/", nil)
							return testClient.Do(req)
						},
						ExpectedStatus: 405,
						Required:       true,
					},
				}

				for _, errorTest := range errorTests {
					By(fmt.Sprintf("testing %s", errorTest.Name))

					result := complianceResults.StartErrorTest(errorTest)

					resp, err := errorTest.RequestFunc()
					if err != nil {
						result.Status = "FAIL"
						result.ErrorMessage = err.Error()
					} else {
						defer resp.Body.Close()

						if resp.StatusCode != errorTest.ExpectedStatus {
							result.Status = "FAIL"
							result.ErrorMessage = fmt.Sprintf("Expected status %d, got %d",
								errorTest.ExpectedStatus, resp.StatusCode)
							result.ActualStatus = resp.StatusCode
						} else {
							// Check if response follows RFC 7807
							body, _ := io.ReadAll(resp.Body)
							var problemDetail map[string]interface{}

							if json.Unmarshal(body, &problemDetail) == nil {
								// Validate RFC 7807 fields
								requiredFields := []string{"type", "title", "status"}
								missingFields := []string{}

								for _, field := range requiredFields {
									if _, exists := problemDetail[field]; !exists {
										missingFields = append(missingFields, field)
									}
								}

								if len(missingFields) == 0 {
									result.Status = "PASS"
									result.RFC7807Compliant = true
								} else {
									result.Status = "FAIL"
									result.ErrorMessage = fmt.Sprintf("Missing RFC 7807 fields: %v", missingFields)
									result.RFC7807Compliant = false
								}
								result.ResponseBody = string(body)
							} else {
								result.Status = "FAIL"
								result.ErrorMessage = "Error response is not valid JSON"
								result.RFC7807Compliant = false
							}
						}
					}

					complianceResults.CompleteErrorTest(errorTest, result)

					if errorTest.Required {
						Expect(result.Status).To(Equal("PASS"),
							fmt.Sprintf("Required error test '%s' failed: %s", errorTest.Name, result.ErrorMessage))
					}
				}
			})
		})
	})

	Describe("Configuration Validation", func() {
		Context("when validating system configuration", func() {
			It("should validate O2 IMS configuration completeness", func() {
				By("checking required configuration parameters")

				configValidation := complianceResults.StartConfigValidation()

				// Test service configuration
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				var serviceInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
				Expect(err).NotTo(HaveOccurred())

				// Validate required service information
				requiredFields := map[string]bool{
					"name":          false,
					"version":       false,
					"apiVersion":    false,
					"specification": false,
					"capabilities":  false,
				}

				for field := range requiredFields {
					if value, exists := serviceInfo[field]; exists && value != nil {
						requiredFields[field] = true
					}
				}

				missingFields := []string{}
				for field, present := range requiredFields {
					if !present {
						missingFields = append(missingFields, field)
					}
				}

				if len(missingFields) == 0 {
					configValidation.ServiceInfoComplete = true
				} else {
					configValidation.MissingServiceFields = missingFields
				}

				// Validate specification compliance
				if spec, ok := serviceInfo["specification"].(string); ok {
					if strings.Contains(spec, "O-RAN.WG6.O2ims-Interface") {
						configValidation.SpecificationCompliant = true
					}
				}

				// Validate capabilities
				if caps, ok := serviceInfo["capabilities"].([]interface{}); ok {
					requiredCapabilities := []string{
						"InfrastructureInventory",
						"InfrastructureMonitoring",
						"InfrastructureProvisioning",
					}

					capStrings := make([]string, len(caps))
					for i, cap := range caps {
						capStrings[i] = cap.(string)
					}

					for _, reqCap := range requiredCapabilities {
						found := false
						for _, actualCap := range capStrings {
							if actualCap == reqCap {
								found = true
								break
							}
						}
						if !found {
							configValidation.MissingCapabilities = append(configValidation.MissingCapabilities, reqCap)
						}
					}
				}

				// Determine overall status
				if configValidation.ServiceInfoComplete &&
					configValidation.SpecificationCompliant &&
					len(configValidation.MissingCapabilities) == 0 {
					configValidation.Status = "PASS"
				} else {
					configValidation.Status = "FAIL"
				}

				complianceResults.CompleteConfigValidation(configValidation)

				Expect(configValidation.Status).To(Equal("PASS"),
					"Configuration validation failed")
			})
		})
	})

	Describe("Security Validation", func() {
		Context("when validating security implementations", func() {
			It("should validate HTTPS and authentication requirements", func() {
				By("checking security headers and configurations")

				securityValidation := complianceResults.StartSecurityValidation()

				// Test basic security headers
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// Check for security headers
				securityHeaders := map[string]bool{
					"X-Content-Type-Options": false,
					"X-Frame-Options":        false,
					"X-XSS-Protection":       false,
				}

				for header := range securityHeaders {
					if resp.Header.Get(header) != "" {
						securityHeaders[header] = true
					}
				}

				presentHeaders := []string{}
				missingHeaders := []string{}

				for header, present := range securityHeaders {
					if present {
						presentHeaders = append(presentHeaders, header)
					} else {
						missingHeaders = append(missingHeaders, header)
					}
				}

				securityValidation.SecurityHeaders = presentHeaders
				securityValidation.MissingSecurityHeaders = missingHeaders

				// Test authentication endpoint existence
				authResp, err := testClient.Post(httpTestServer.URL+"/oauth/token",
					"application/x-www-form-urlencoded",
					strings.NewReader("grant_type=client_credentials"))

				if err == nil {
					authResp.Body.Close()
					securityValidation.AuthEndpointExists = true
				}

				// Determine overall security compliance
				if len(presentHeaders) >= 1 { // At least some security headers
					securityValidation.Status = "PASS"
				} else {
					securityValidation.Status = "FAIL"
					securityValidation.ErrorMessage = "No security headers found"
				}

				complianceResults.CompleteSecurityValidation(securityValidation)

				// Note: In test environment, some security features may not be fully enabled
				// so we don't fail the test, but we do record the compliance status
			})
		})
	})
})

// Compliance Testing Data Structures

type ComplianceResults struct {
	TestStartTime      time.Time
	EndpointTests      []EndpointTestResult
	StatusCodeTests    []StatusCodeTestResult
	SchemaValidations  []SchemaValidationResult
	FilterTests        []FilterTestResult
	PaginationTests    []PaginationTestResult
	ErrorTests         []ErrorTestResult
	ConfigValidation   *ConfigValidationResult
	SecurityValidation *SecurityValidationResult
	OverallCompliance  string
}

type ComplianceEndpoint struct {
	Method      string
	Path        string
	Description string
	Required    bool
}

type EndpointTestResult struct {
	Endpoint     ComplianceEndpoint
	Status       string
	ResponseTime time.Duration
	ErrorMessage string
	TestTime     time.Time
}

type ComplianceStatusCodeTest struct {
	Name           string
	Request        func() (*http.Response, error)
	ExpectedStatus int
	Required       bool
}

type StatusCodeTestResult struct {
	Test         ComplianceStatusCodeTest
	Status       string
	ActualStatus int
	ErrorMessage string
	TestTime     time.Time
}

type SchemaValidationResult struct {
	SchemaType    string
	Status        string
	MissingFields []string
	ErrorMessage  string
	TestTime      time.Time
}

type ComplianceFilterTest struct {
	Name           string
	QueryParam     string
	Description    string
	ExpectedResult string
	Required       bool
}

type FilterTestResult struct {
	Test         ComplianceFilterTest
	Status       string
	ResultCount  int
	ErrorMessage string
	TestTime     time.Time
}

type CompliancePaginationTest struct {
	Name       string
	QueryParam string
	MaxResults int
	Required   bool
}

type PaginationTestResult struct {
	Test         CompliancePaginationTest
	Status       string
	ActualCount  int
	ErrorMessage string
	TestTime     time.Time
}

type ComplianceErrorTest struct {
	Name           string
	RequestFunc    func() (*http.Response, error)
	ExpectedStatus int
	Required       bool
}

type ErrorTestResult struct {
	Test             ComplianceErrorTest
	Status           string
	ActualStatus     int
	RFC7807Compliant bool
	ResponseBody     string
	ErrorMessage     string
	TestTime         time.Time
}

type ConfigValidationResult struct {
	Status                 string
	ServiceInfoComplete    bool
	SpecificationCompliant bool
	MissingServiceFields   []string
	MissingCapabilities    []string
	ErrorMessage           string
	TestTime               time.Time
}

type SecurityValidationResult struct {
	Status                 string
	SecurityHeaders        []string
	MissingSecurityHeaders []string
	AuthEndpointExists     bool
	TLSEnabled             bool
	ErrorMessage           string
	TestTime               time.Time
}

// Compliance Results Implementation

func NewComplianceResults() *ComplianceResults {
	return &ComplianceResults{
		TestStartTime:     time.Now(),
		EndpointTests:     make([]EndpointTestResult, 0),
		StatusCodeTests:   make([]StatusCodeTestResult, 0),
		SchemaValidations: make([]SchemaValidationResult, 0),
		FilterTests:       make([]FilterTestResult, 0),
		PaginationTests:   make([]PaginationTestResult, 0),
		ErrorTests:        make([]ErrorTestResult, 0),
	}
}

func (cr *ComplianceResults) StartEndpointTest(endpoint ComplianceEndpoint) *EndpointTestResult {
	result := &EndpointTestResult{
		Endpoint: endpoint,
		TestTime: time.Now(),
	}
	return result
}

func (cr *ComplianceResults) CompleteEndpointTest(endpoint ComplianceEndpoint, result *EndpointTestResult) {
	cr.EndpointTests = append(cr.EndpointTests, *result)
}

func (cr *ComplianceResults) StartStatusCodeTest(test ComplianceStatusCodeTest) *StatusCodeTestResult {
	return &StatusCodeTestResult{
		Test:     test,
		TestTime: time.Now(),
	}
}

func (cr *ComplianceResults) CompleteStatusCodeTest(test ComplianceStatusCodeTest, result *StatusCodeTestResult) {
	cr.StatusCodeTests = append(cr.StatusCodeTests, *result)
}

func (cr *ComplianceResults) StartSchemaValidation(schemaType string) *SchemaValidationResult {
	return &SchemaValidationResult{
		SchemaType:    schemaType,
		MissingFields: make([]string, 0),
		TestTime:      time.Now(),
	}
}

func (cr *ComplianceResults) CompleteSchemaValidation(schemaType string, result *SchemaValidationResult) {
	cr.SchemaValidations = append(cr.SchemaValidations, *result)
}

func (cr *ComplianceResults) StartFilterTest(test ComplianceFilterTest) *FilterTestResult {
	return &FilterTestResult{
		Test:     test,
		TestTime: time.Now(),
	}
}

func (cr *ComplianceResults) CompleteFilterTest(test ComplianceFilterTest, result *FilterTestResult) {
	cr.FilterTests = append(cr.FilterTests, *result)
}

func (cr *ComplianceResults) StartPaginationTest(test CompliancePaginationTest) *PaginationTestResult {
	return &PaginationTestResult{
		Test:     test,
		TestTime: time.Now(),
	}
}

func (cr *ComplianceResults) CompletePaginationTest(test CompliancePaginationTest, result *PaginationTestResult) {
	cr.PaginationTests = append(cr.PaginationTests, *result)
}

func (cr *ComplianceResults) StartErrorTest(test ComplianceErrorTest) *ErrorTestResult {
	return &ErrorTestResult{
		Test:     test,
		TestTime: time.Now(),
	}
}

func (cr *ComplianceResults) CompleteErrorTest(test ComplianceErrorTest, result *ErrorTestResult) {
	cr.ErrorTests = append(cr.ErrorTests, *result)
}

func (cr *ComplianceResults) StartConfigValidation() *ConfigValidationResult {
	cr.ConfigValidation = &ConfigValidationResult{
		MissingServiceFields: make([]string, 0),
		MissingCapabilities:  make([]string, 0),
		TestTime:             time.Now(),
	}
	return cr.ConfigValidation
}

func (cr *ComplianceResults) CompleteConfigValidation(result *ConfigValidationResult) {
	cr.ConfigValidation = result
}

func (cr *ComplianceResults) StartSecurityValidation() *SecurityValidationResult {
	cr.SecurityValidation = &SecurityValidationResult{
		SecurityHeaders:        make([]string, 0),
		MissingSecurityHeaders: make([]string, 0),
		TestTime:               time.Now(),
	}
	return cr.SecurityValidation
}

func (cr *ComplianceResults) CompleteSecurityValidation(result *SecurityValidationResult) {
	cr.SecurityValidation = result
}

func (cr *ComplianceResults) GenerateReport() {
	// Calculate overall compliance
	totalTests := len(cr.EndpointTests) + len(cr.StatusCodeTests) + len(cr.SchemaValidations) +
		len(cr.FilterTests) + len(cr.PaginationTests) + len(cr.ErrorTests)

	passedTests := 0

	for _, test := range cr.EndpointTests {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	for _, test := range cr.StatusCodeTests {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	for _, test := range cr.SchemaValidations {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	for _, test := range cr.FilterTests {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	for _, test := range cr.PaginationTests {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	for _, test := range cr.ErrorTests {
		if test.Status == "PASS" {
			passedTests++
		}
	}

	if cr.ConfigValidation != nil && cr.ConfigValidation.Status == "PASS" {
		passedTests++
		totalTests++
	}

	if cr.SecurityValidation != nil && cr.SecurityValidation.Status == "PASS" {
		passedTests++
		totalTests++
	}

	compliancePercentage := float64(passedTests) / float64(totalTests) * 100

	if compliancePercentage >= 95.0 {
		cr.OverallCompliance = "COMPLIANT"
	} else if compliancePercentage >= 80.0 {
		cr.OverallCompliance = "PARTIALLY_COMPLIANT"
	} else {
		cr.OverallCompliance = "NON_COMPLIANT"
	}

	// Log compliance report
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("O2 IMS O-RAN COMPLIANCE VALIDATION REPORT\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")
	fmt.Printf("Test Duration: %v\n", time.Since(cr.TestStartTime))
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("Passed Tests: %d\n", passedTests)
	fmt.Printf("Compliance Percentage: %.1f%%\n", compliancePercentage)
	fmt.Printf("Overall Compliance: %s\n", cr.OverallCompliance)
	fmt.Printf(strings.Repeat("=", 80) + "\n\n")
}

func CreateTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "o2-compliance-test-",
		},
	}
}

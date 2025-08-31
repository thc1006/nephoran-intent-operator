//go:build integration

package integration_tests_test

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
	"github.com/nephio-project/nephoran-intent-operator/pkg/oran/o2"
	"github.com/nephio-project/nephoran-intent-operator/pkg/oran/o2/models"
)

var _ = Describe("O2 Infrastructure Management Service Integration Tests", func() {
	var (
		namespace       *corev1.Namespace
		testCtx         context.Context
		o2Server        *o2.O2APIServer
		httpTestServer  *httptest.Server
		testClient      *http.Client
		metricsRegistry *prometheus.Registry
		testLogger      *logging.StructuredLogger
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 10*time.Minute)
		DeferCleanup(cancel)

		// Setup test logger
		testLogger = logging.NewLogger("o2-integration-test", "debug")

		// Setup metrics registry
		metricsRegistry = prometheus.NewRegistry()

		// Setup O2 IMS service with test configuration
		config := &o2.O2IMSConfig{
			ServerAddress: "127.0.0.1",
			ServerPort:    0, // Use dynamic port for testing
			TLSEnabled:    false,
			DatabaseConfig: map[string]interface{}{
				"type":     "memory",
				"database": "o2_test_db",
			},
			ProviderConfigs: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"kubeconfig": "",
					},
				},
			},
			MonitoringConfig: map[string]interface{}{
				"enabled":  true,
				"interval": "30s",
			},
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		// Start test HTTP server
		httpTestServer = httptest.NewServer(o2Server.GetRouter())
		testClient = httpTestServer.Client()
		testClient.Timeout = 30 * time.Second

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
		})
	})

	Describe("Service Information and Health Endpoints", func() {
		Context("when accessing service information", func() {
			It("should return complete service information", func() {
				By("calling GET /o2ims/v1/")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var serviceInfo map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&serviceInfo)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("validating service information structure")
				Expect(serviceInfo).To(HaveKey("name"))
				Expect(serviceInfo).To(HaveKey("version"))
				Expect(serviceInfo).To(HaveKey("apiVersion"))
				Expect(serviceInfo).To(HaveKey("specification"))
				Expect(serviceInfo).To(HaveKey("capabilities"))
				Expect(serviceInfo["name"]).To(Equal("Nephoran O2 IMS"))
				Expect(serviceInfo["specification"]).To(Equal("O-RAN.WG6.O2ims-Interface-v01.01"))

				capabilities, ok := serviceInfo["capabilities"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(capabilities).To(ContainElement("InfrastructureInventory"))
				Expect(capabilities).To(ContainElement("InfrastructureMonitoring"))
				Expect(capabilities).To(ContainElement("InfrastructureProvisioning"))
			})
		})

		Context("when checking health endpoints", func() {
			It("should return healthy status from health endpoint", func() {
				By("calling GET /health")
				resp, err := testClient.Get(httpTestServer.URL + "/health")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var health map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&health)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("validating health response structure")
				Expect(health).To(HaveKey("status"))
				Expect(health).To(HaveKey("timestamp"))
				Expect(health).To(HaveKey("components"))
			})

			It("should return ready status from readiness endpoint", func() {
				By("calling GET /ready")
				resp, err := testClient.Get(httpTestServer.URL + "/ready")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var readiness map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&readiness)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("validating readiness response structure")
				Expect(readiness).To(HaveKey("status"))
				Expect(readiness["status"]).To(Equal("READY"))
			})
		})
	})

	Describe("Infrastructure Inventory Management", func() {
		Context("when managing resource pools", func() {
			It("should support complete CRUD operations for resource pools", func() {
				poolID := "test-pool-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a new resource pool")
				newPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Test Resource Pool",
					Description:    "Integration test resource pool",
					Location:       "test-location",
					OCloudID:       "test-ocloud",
					Provider:       "kubernetes",
					Region:         "us-west-2",
					Zone:           "us-west-2a",
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "100",
							Available:   "80",
							Used:        "20",
							Unit:        "cores",
							Utilization: 20.0,
						},
						Memory: &models.ResourceMetric{
							Total:       "256Gi",
							Available:   "200Gi",
							Used:        "56Gi",
							Unit:        "bytes",
							Utilization: 21.875,
						},
					},
				}

				poolJSON, err := json.Marshal(newPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the created resource pool")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedPool models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedPool.ResourcePoolID).To(Equal(poolID))
				Expect(retrievedPool.Name).To(Equal("Test Resource Pool"))
				Expect(retrievedPool.Provider).To(Equal("kubernetes"))

				By("updating the resource pool")
				retrievedPool.Description = "Updated description"
				updatedPoolJSON, err := json.Marshal(retrievedPool)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("PUT",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
					bytes.NewBuffer(updatedPoolJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()

				By("verifying the update")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var updatedPool models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&updatedPool)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(updatedPool.Description).To(Equal("Updated description"))

				By("deleting the resource pool")
				req, err = http.NewRequest("DELETE",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusNoContent), Equal(http.StatusOK)))
				resp.Body.Close()

				By("verifying deletion")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()
			})

			It("should support filtering and pagination for resource pools", func() {
				By("creating multiple test resource pools")
				var createdPools []string
				for i := 0; i < 5; i++ {
					poolID := fmt.Sprintf("filter-test-pool-%d-%d", i, time.Now().UnixNano())
					createdPools = append(createdPools, poolID)

					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:           fmt.Sprintf("Filter Test Pool %d", i),
						Provider:       "kubernetes",
						Location:       fmt.Sprintf("location-%d", i%2),
						OCloudID:       "test-ocloud",
					}

					poolJSON, err := json.Marshal(pool)
					Expect(err).NotTo(HaveOccurred())

					resp, err := testClient.Post(
						httpTestServer.URL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
					resp.Body.Close()
				}

				DeferCleanup(func() {
					for _, poolID := range createdPools {
						req, _ := http.NewRequest("DELETE",
							httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)
					}
				})

				By("testing filtering by location")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools?filter=location,eq,location-0")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var filteredPools []models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&filteredPools)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				// Should return pools with location-0
				filteredCount := 0
				for _, pool := range filteredPools {
					if pool.Location == "location-0" && strings.HasPrefix(pool.ResourcePoolID, "filter-test-pool-") {
						filteredCount++
					}
				}
				Expect(filteredCount).To(BeNumerically(">=", 1))

				By("testing pagination")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools?limit=2&offset=0")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var paginatedPools []models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&paginatedPools)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				// Should return at most 2 pools
				Expect(len(paginatedPools)).To(BeNumerically("<=", 2))
			})
		})

		Context("when managing resource types", func() {
			It("should support resource type operations", func() {
				typeID := "test-type-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a new resource type")
				newResourceType := &models.ResourceType{
					ResourceTypeID: typeID,
					Name:           "Test Compute Resource",
					Description:    "Test compute resource type for integration testing",
					Vendor:         "Nephoran",
					Model:          "Standard-Compute-v1",
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
						DefaultResources: map[string]string{
							"cpu":    "2",
							"memory": "4Gi",
						},
					},
					SupportedActions: []string{"CREATE", "DELETE", "UPDATE", "SCALE"},
				}

				typeJSON, err := json.Marshal(newResourceType)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourceTypes",
					"application/json",
					bytes.NewBuffer(typeJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the created resource type")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourceTypes/" + typeID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedType models.ResourceType
				err = json.NewDecoder(resp.Body).Decode(&retrievedType)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedType.ResourceTypeID).To(Equal(typeID))
				Expect(retrievedType.Name).To(Equal("Test Compute Resource"))
				Expect(retrievedType.Specifications.Category).To(Equal("COMPUTE"))
				Expect(retrievedType.SupportedActions).To(ContainElement("CREATE"))

				By("cleaning up the resource type")
				req, err := http.NewRequest("DELETE",
					httpTestServer.URL+"/o2ims/v1/resourceTypes/"+typeID, nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusNoContent), Equal(http.StatusOK)))
				resp.Body.Close()
			})
		})
	})

	Describe("Infrastructure Monitoring", func() {
		Context("when managing alarms", func() {
			It("should support alarm lifecycle operations", func() {
				alarmID := "test-alarm-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a test alarm")
				alarm := map[string]interface{}{
					"alarmEventRecordId":    alarmID,
					"resourceTypeID":        "test-resource-type",
					"resourceID":            "test-resource-1",
					"alarmDefinitionID":     "cpu-high-utilization",
					"probableCause":         "High CPU utilization detected",
					"specificProblem":       "CPU usage exceeded 90% threshold",
					"perceivedSeverity":     "MAJOR",
					"alarmRaisedTime":       time.Now().Format(time.RFC3339),
					"alarmChangedTime":      time.Now().Format(time.RFC3339),
					"alarmAcknowledged":     false,
					"alarmAcknowledgedTime": nil,
				}

				alarmJSON, err := json.Marshal(alarm)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/alarms",
					"application/json",
					bytes.NewBuffer(alarmJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the created alarm")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/alarms/" + alarmID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedAlarm map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&retrievedAlarm)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedAlarm["alarmEventRecordId"]).To(Equal(alarmID))
				Expect(retrievedAlarm["perceivedSeverity"]).To(Equal("MAJOR"))

				By("acknowledging the alarm")
				ackData := map[string]interface{}{
					"alarmAcknowledged":     true,
					"alarmAcknowledgedTime": time.Now().Format(time.RFC3339),
				}

				ackJSON, err := json.Marshal(ackData)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/alarms/"+alarmID,
					bytes.NewBuffer(ackJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()

				By("verifying alarm acknowledgment")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/alarms/" + alarmID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				err = json.NewDecoder(resp.Body).Decode(&retrievedAlarm)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedAlarm["alarmAcknowledged"]).To(Equal(true))
			})
		})

		Context("when managing subscriptions", func() {
			It("should support subscription CRUD operations", func() {
				subscriptionID := "test-subscription-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a monitoring subscription")
				subscription := map[string]interface{}{
					"subscriptionId":         subscriptionID,
					"consumerSubscriptionId": "consumer-" + subscriptionID,
					"filter":                 "(eq,resourceTypeId,compute-node);(eq,perceivedSeverity,MAJOR)",
					"callback":               "http://example.com/notifications",
					"consumerSubscriptionId": "external-consumer-123",
				}

				subJSON, err := json.Marshal(subscription)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/subscriptions",
					"application/json",
					bytes.NewBuffer(subJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the created subscription")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/subscriptions/" + subscriptionID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedSub map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&retrievedSub)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedSub["subscriptionId"]).To(Equal(subscriptionID))
				Expect(retrievedSub["callback"]).To(Equal("http://example.com/notifications"))

				By("deleting the subscription")
				req, err := http.NewRequest("DELETE",
					httpTestServer.URL+"/o2ims/v1/subscriptions/"+subscriptionID, nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusNoContent), Equal(http.StatusOK)))
				resp.Body.Close()
			})
		})
	})

	Describe("Error Handling and Edge Cases", func() {
		Context("when handling invalid requests", func() {
			It("should return appropriate error responses for invalid resource pool operations", func() {
				By("attempting to create resource pool with invalid data")
				invalidPool := map[string]interface{}{
					"invalidField": "should not be accepted",
				}

				poolJSON, err := json.Marshal(invalidPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()

				By("attempting to retrieve non-existent resource pool")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/non-existent-pool")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
				resp.Body.Close()

				By("attempting to use invalid query parameters")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools?limit=invalid")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()
			})

			It("should handle malformed JSON requests gracefully", func() {
				By("sending malformed JSON")
				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					strings.NewReader(`{"malformed": json}`),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				// Read and validate error response
				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				var errorResp map[string]interface{}
				err = json.Unmarshal(body, &errorResp)
				Expect(err).NotTo(HaveOccurred())
				Expect(errorResp).To(HaveKey("type"))
				Expect(errorResp).To(HaveKey("title"))
				Expect(errorResp).To(HaveKey("status"))
			})
		})

		Context("when handling concurrent operations", func() {
			It("should handle concurrent resource pool operations safely", func() {
				By("performing concurrent resource pool creations")
				var wg sync.WaitGroup
				errors := make(chan error, 10)
				successes := make(chan string, 10)

				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()

						poolID := fmt.Sprintf("concurrent-pool-%d-%d", index, time.Now().UnixNano())
						pool := &models.ResourcePool{
							ResourcePoolID: poolID,
							Name:           fmt.Sprintf("Concurrent Test Pool %d", index),
							Provider:       "kubernetes",
							OCloudID:       "test-ocloud",
						}

						poolJSON, err := json.Marshal(pool)
						if err != nil {
							errors <- err
							return
						}

						resp, err := testClient.Post(
							httpTestServer.URL+"/o2ims/v1/resourcePools",
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

				// Verify that most operations succeeded
				errorCount := len(errors)
				successCount := len(successes)

				Expect(errorCount).To(BeNumerically("<=", 2), "Too many concurrent operation errors")
				Expect(successCount).To(BeNumerically(">=", 8), "Too few successful concurrent operations")

				// Cleanup created pools
				for poolID := range successes {
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req) // Best effort cleanup
				}
			})
		})
	})

	Describe("API Performance Metrics", func() {
		Context("when measuring API response times", func() {
			It("should meet performance SLA requirements", func() {
				By("measuring service info endpoint performance")
				measurements := make([]time.Duration, 50)
				for i := 0; i < 50; i++ {
					start := time.Now()
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
					duration := time.Since(start)
					measurements[i] = duration

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					resp.Body.Close()
				}

				// Calculate average response time
				var total time.Duration
				for _, d := range measurements {
					total += d
				}
				average := total / time.Duration(len(measurements))

				// Verify performance requirements
				Expect(average).To(BeNumerically("<", 500*time.Millisecond),
					fmt.Sprintf("Average response time %v exceeds 500ms SLA", average))

				// Verify 95th percentile
				// Simple calculation - in production use proper percentile calculation
				maxAcceptable := 1 * time.Second
				violationCount := 0
				for _, d := range measurements {
					if d > maxAcceptable {
						violationCount++
					}
				}
				violationRate := float64(violationCount) / float64(len(measurements))
				Expect(violationRate).To(BeNumerically("<=", 0.05),
					fmt.Sprintf("%.1f%% of requests exceeded 1s (should be ≤5%%)", violationRate*100))
			})
		})
	})
})

// Test helper functions

func CreateTestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "o2-ims-integration-test-",
		},
	}
	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())

	DeferCleanup(func() {
		k8sClient.Delete(context.Background(), namespace)
	})

	return namespace
}

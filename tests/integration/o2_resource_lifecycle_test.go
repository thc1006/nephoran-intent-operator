package integration_tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

var _ = Describe("O2 Resource Lifecycle Management Integration Tests", func() {
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
		testCtx, cancel = context.WithTimeout(ctx, 25*time.Minute)
		DeferCleanup(cancel)

		testLogger = logging.NewLogger("o2-lifecycle-test", "debug")
		metricsRegistry = prometheus.NewRegistry()

		config := &o2.O2IMSConfig{
			ServerAddress: "127.0.0.1",
			ServerPort:    0,
			TLSEnabled:    false,
			DatabaseConfig: json.RawMessage(`{}`),
			ProviderConfigs: map[string]interface{}{
				"enabled": true,
				"config":  json.RawMessage(`{}`),
			},
			LifecycleConfig: map[string]interface{}{
				"maxRetries":       3,
				"backoffFactor":    1.5,
				"initialDelay":     "5s",
				"stateTransitions": json.RawMessage(`{}`),
			},
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		httpTestServer = httptest.NewServer(o2Server.GetRouter())
		testClient = httpTestServer.Client()
		testClient.Timeout = 180 * time.Second

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
		})
	})

	Describe("Resource Creation Lifecycle", func() {
		Context("when creating resources with dependencies", func() {
			It("should handle resource creation with proper dependency resolution", func() {
				baseTimestamp := time.Now().UnixNano()

				// Create dependent resources in correct order
				By("creating base infrastructure resource pool")
				poolID := fmt.Sprintf("base-pool-%d", baseTimestamp)
				basePool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Base Infrastructure Pool",
					Description:    "Foundation pool for dependent resources",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Status: &models.ResourcePoolStatus{
						State:  "AVAILABLE",
						Health: "HEALTHY",
					},
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "100",
							Available:   "80",
							Used:        "20",
							Unit:        "cores",
							Utilization: 20.0,
						},
					},
				}

				poolJSON, err := json.Marshal(basePool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("creating dependent compute resource type")
				computeTypeID := fmt.Sprintf("compute-type-%d", baseTimestamp)
				computeType := &models.ResourceType{
					ResourceTypeID: computeTypeID,
					Name:           "Compute Node Type",
					Description:    "Compute resource type dependent on base pool",
					Category:       "COMPUTE",
					Specifications: &models.ResourceTypeSpec{
						Category: "COMPUTE",
						MinResources: map[string]string{
							"cpu":    "1",
							"memory": "2Gi",
						},
						MaxResources: map[string]string{
							"cpu":    "16",
							"memory": "64Gi",
						},
						Properties: map[string]interface{}{
							"architecture": "x86_64",
						},
						SupportedActions: []string{"CREATE", "DELETE", "UPDATE", "SCALE"},
					},
				}

				typeJSON, err := json.Marshal(computeType)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourceTypes",
					"application/json",
					bytes.NewBuffer(typeJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("creating actual compute resource instance")
				resourceID := fmt.Sprintf("compute-resource-%d", baseTimestamp)
				computeResource := map[string]interface{}{
					"resources": map[string]interface{}{
						"cpu":     "4",
						"memory":  "8Gi",
						"storage": "100Gi",
					},
					"extensions": map[string]interface{}{
						"creationRequested": time.Now().Format(time.RFC3339),
						"expectedReadyTime": time.Now().Add(5 * time.Minute).Format(time.RFC3339),
					},
				}

				resourceJSON, err := json.Marshal(computeResource)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resources",
					"application/json",
					bytes.NewBuffer(resourceJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusAccepted)))
				resp.Body.Close()

				By("monitoring resource creation progress")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resources/" + resourceID)
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode != http.StatusOK {
						return ""
					}

					var resource map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&resource)
					if err != nil {
						return ""
					}

					if state, ok := resource["resourceState"].(string); ok {
						return state
					}
					return ""
				}, 3*time.Minute, 10*time.Second).Should(Or(Equal("ACTIVE"), Equal("AVAILABLE")))

				By("verifying resource relationships and dependencies")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resources/" + resourceID + "/relationships")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var relationships []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&relationships)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				// Should have relationship to resource pool
				foundPoolRelationship := false
				for _, rel := range relationships {
					if rel["relatedResourceId"] == poolID && rel["relationType"] == "DEPENDS_ON" {
						foundPoolRelationship = true
						break
					}
				}
				Expect(foundPoolRelationship).To(BeTrue())

				DeferCleanup(func() {
					// Cleanup in reverse dependency order
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resources/"+resourceID, nil)
					testClient.Do(req)
					req, _ = http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourceTypes/"+computeTypeID, nil)
					testClient.Do(req)
					req, _ = http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})

			It("should prevent creation of resources with unresolved dependencies", func() {
				By("attempting to create resource with non-existent dependency")
				resourceID := fmt.Sprintf("orphan-resource-%d", time.Now().UnixNano())
				orphanResource := json.RawMessage(`{}`)

				resourceJSON, err := json.Marshal(orphanResource)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resources",
					"application/json",
					bytes.NewBuffer(resourceJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				var errorResponse map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResponse)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(errorResponse).To(HaveKey("type"))
				Expect(errorResponse).To(HaveKey("title"))
				Expect(errorResponse["title"]).To(ContainSubstring("dependency"))
			})
		})
	})

	Describe("Resource Modification Lifecycle", func() {
		Context("when updating resource specifications", func() {
			It("should handle resource updates with validation", func() {
				baseTimestamp := time.Now().UnixNano()
				poolID := fmt.Sprintf("update-pool-%d", baseTimestamp)
				_ = fmt.Sprintf("update-resource-%d", baseTimestamp)

				By("creating initial resource for updates")
				initialPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Update Test Pool",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "50",
							Available:   "40",
							Used:        "10",
							Unit:        "cores",
							Utilization: 20.0,
						},
						Memory: &models.ResourceMetric{
							Total:       "100Gi",
							Available:   "80Gi",
							Used:        "20Gi",
							Unit:        "bytes",
							Utilization: 20.0,
						},
					},
				}

				poolJSON, err := json.Marshal(initialPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("updating resource pool capacity")
				updatedCapacity := map[string]interface{}{
					"cpu": map[string]interface{}{
						"total":       "100",
						"available":   "80",
						"used":        "20",
						"unit":        "cores",
						"utilization": 20.0,
					},
					"memory": json.RawMessage(`{}`),
				}

				updateRequest := json.RawMessage(`{}`)

				updateJSON, err := json.Marshal(updateRequest)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
					bytes.NewBuffer(updateJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()

				By("verifying update was applied correctly")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var updatedPool models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&updatedPool)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(updatedPool.Capacity.CPU.Total).To(Equal("100"))
				Expect(updatedPool.Capacity.Memory.Total).To(Equal("200Gi"))
				Expect(updatedPool.Description).To(Equal("Updated pool with increased capacity"))

				By("testing concurrent updates to same resource")
				var updateWg sync.WaitGroup
				updateErrors := make(chan error, 5)
				updateSuccesses := make(chan bool, 5)

				for i := 0; i < 5; i++ {
					updateWg.Add(1)
					go func(iteration int) {
						defer updateWg.Done()

						concurrentUpdate := map[string]interface{}{
							fmt.Sprintf("concurrentUpdate%d", iteration): time.Now().Format(time.RFC3339),
							"iteration": iteration,
						}

						updateJSON, err := json.Marshal(concurrentUpdate)
						if err != nil {
							updateErrors <- err
							return
						}

						req, err := http.NewRequest("PATCH",
							httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
							bytes.NewBuffer(updateJSON))
						if err != nil {
							updateErrors <- err
							return
						}
						req.Header.Set("Content-Type", "application/json")

						resp, err := testClient.Do(req)
						if err != nil {
							updateErrors <- err
							return
						}
						defer resp.Body.Close() // #nosec G307 - Error handled in defer

						if resp.StatusCode == http.StatusOK {
							updateSuccesses <- true
						} else {
							updateErrors <- fmt.Errorf("update failed with status %d", resp.StatusCode)
						}
					}(i)
				}

				updateWg.Wait()
				close(updateErrors)
				close(updateSuccesses)

				errorCount := len(updateErrors)
				successCount := len(updateSuccesses)

				// Allow some failures due to concurrent updates, but most should succeed
				Expect(errorCount).To(BeNumerically("<=", 2))
				Expect(successCount).To(BeNumerically(">=", 3))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})

			It("should validate update constraints and limits", func() {
				baseTimestamp := time.Now().UnixNano()
				poolID := fmt.Sprintf("constraint-pool-%d", baseTimestamp)

				By("creating resource pool with constraints")
				constraintPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Constraint Test Pool",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "10",
							Available:   "8",
							Used:        "2",
							Unit:        "cores",
							Utilization: 20.0,
						},
					},
					Extensions: map[string]interface{}{
						"maxCPU":          20,
						"minCPU":          5,
						"immutableFields": []string{"provider", "oCloudId"},
					},
				}

				poolJSON, err := json.Marshal(constraintPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("attempting to update beyond constraints")
				violatingUpdate := map[string]interface{}{
					"cpu": json.RawMessage(`{}`),
				}

				updateJSON, err := json.Marshal(violatingUpdate)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
					bytes.NewBuffer(updateJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				var constraintError map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&constraintError)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(constraintError["title"]).To(ContainSubstring("constraint"))

				By("attempting to update immutable fields")
				immutableUpdate := json.RawMessage(`{}`)

				updateJSON, err = json.Marshal(immutableUpdate)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
					bytes.NewBuffer(updateJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				resp.Body.Close()

				By("performing valid update within constraints")
				validUpdate := map[string]interface{}{
					"cpu":         json.RawMessage(`{}`),
					"description": "Updated within constraints",
				}

				updateJSON, err = json.Marshal(validUpdate)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
					bytes.NewBuffer(updateJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("Resource Scaling Operations", func() {
		Context("when scaling resources", func() {
			It("should support horizontal and vertical scaling", func() {
				baseTimestamp := time.Now().UnixNano()
				poolID := fmt.Sprintf("scaling-pool-%d", baseTimestamp)

				By("creating scalable resource pool")
				scalingPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Scaling Test Pool",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "10",
							Available:   "8",
							Used:        "2",
							Unit:        "cores",
							Utilization: 20.0,
						},
					},
					Extensions: map[string]interface{}{
							"enabled": true,
							"horizontal": json.RawMessage(`{}`),
							"vertical": json.RawMessage(`{}`),
						},
					},
				}

				poolJSON, err := json.Marshal(scalingPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("performing horizontal scale out")
				scaleOutRequest := map[string]interface{}{
						"targetInstances": 3,
					},
				}

				scaleJSON, err := json.Marshal(scaleOutRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID+"/scale",
					"application/json",
					bytes.NewBuffer(scaleJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("monitoring horizontal scaling progress")
				Eventually(func() int {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
					if err != nil {
						return 0
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					var pool models.ResourcePool
					err = json.NewDecoder(resp.Body).Decode(&pool)
					if err != nil {
						return 0
					}

					if scaling, ok := pool.Extensions["scaling"].(map[string]interface{}); ok {
						if horizontal, ok := scaling["horizontal"].(map[string]interface{}); ok {
							if current, ok := horizontal["currentInstances"].(float64); ok {
								return int(current)
							}
						}
					}
					return 0
				}, 2*time.Minute, 10*time.Second).Should(Equal(3))

				By("performing vertical scale up")
				scaleUpRequest := map[string]interface{}{
						"targetCPU": "20",
					},
				}

				scaleJSON, err = json.Marshal(scaleUpRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID+"/scale",
					"application/json",
					bytes.NewBuffer(scaleJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("verifying vertical scaling completion")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					var pool models.ResourcePool
					err = json.NewDecoder(resp.Body).Decode(&pool)
					if err != nil {
						return ""
					}

					return pool.Capacity.CPU.Total
				}, 2*time.Minute, 10*time.Second).Should(Equal("20"))

				By("testing scaling limits and constraints")
				excessiveScaleRequest := map[string]interface{}{
						"targetInstances": 15,
					},
				}

				scaleJSON, err = json.Marshal(excessiveScaleRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID+"/scale",
					"application/json",
					bytes.NewBuffer(scaleJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				var scaleError map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&scaleError)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(scaleError["title"]).To(ContainSubstring("scaling"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("Resource Deletion and Cleanup", func() {
		Context("when deleting resources", func() {
			It("should handle cascading deletion with dependency management", func() {
				baseTimestamp := time.Now().UnixNano()
				parentPoolID := fmt.Sprintf("parent-pool-%d", baseTimestamp)
				childResourceID := fmt.Sprintf("child-resource-%d", baseTimestamp)

				By("creating parent resource pool")
				parentPool := &models.ResourcePool{
					ResourcePoolID: parentPoolID,
					Name:           "Parent Pool for Deletion Test",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
				}

				poolJSON, err := json.Marshal(parentPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("creating child resource dependent on parent")
				childResource := json.RawMessage(`{}`),
				}

				resourceJSON, err := json.Marshal(childResource)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resources",
					"application/json",
					bytes.NewBuffer(resourceJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("attempting to delete parent with existing dependencies")
				req, err := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+parentPoolID, nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				// Should fail due to existing dependencies
				Expect(resp.StatusCode).To(Equal(http.StatusConflict))

				var deleteError map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&deleteError)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(deleteError["title"]).To(ContainSubstring("dependencies"))

				By("performing cascading deletion")
				req, err = http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+parentPoolID+"?cascade=true", nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("verifying cascading deletion completion")
				Eventually(func() int {
					// Check that parent pool is deleted
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + parentPoolID)
					if err != nil {
						return http.StatusNotFound
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer
					return resp.StatusCode
				}, 2*time.Minute, 10*time.Second).Should(Equal(http.StatusNotFound))

				Eventually(func() int {
					// Check that child resource is also deleted
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resources/" + childResourceID)
					if err != nil {
						return http.StatusNotFound
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer
					return resp.StatusCode
				}, 2*time.Minute, 10*time.Second).Should(Equal(http.StatusNotFound))
			})

			It("should support graceful shutdown with configurable timeouts", func() {
				baseTimestamp := time.Now().UnixNano()
				poolID := fmt.Sprintf("graceful-pool-%d", baseTimestamp)

				By("creating resource with graceful shutdown configuration")
				gracefulPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "Graceful Shutdown Pool",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Extensions: map[string]interface{}{
							"gracefulShutdownTimeout": "60s",
							"preStopHooks": []string{
								"drain-connections",
								"save-state",
							},
							"forceDeleteAfter": "120s",
						},
					},
				}

				poolJSON, err := json.Marshal(gracefulPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("initiating graceful deletion")
				deletionRequest := map[string]interface{}{
						"drainTimeout":     "15s",
						"skipPreStopHooks": false,
					},
				}

				deletionJSON, err := json.Marshal(deletionRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID+"/terminate",
					"application/json",
					bytes.NewBuffer(deletionJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("monitoring graceful deletion progress")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID + "/status")
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode == http.StatusNotFound {
						return "DELETED"
					}

					var status map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&status)
					if err != nil {
						return ""
					}

					if state, ok := status["lifecycleState"].(string); ok {
						return state
					}
					return ""
				}, 2*time.Minute, 5*time.Second).Should(Or(Equal("TERMINATING"), Equal("DELETED")))
			})
		})
	})

	Describe("Resource State Transitions and Events", func() {
		Context("when monitoring resource state changes", func() {
			It("should track complete lifecycle state transitions", func() {
				baseTimestamp := time.Now().UnixNano()
				poolID := fmt.Sprintf("state-tracking-pool-%d", baseTimestamp)

				By("creating resource to track state transitions")
				stateTrackingPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           "State Tracking Pool",
					Provider:       "kubernetes",
					OCloudID:       "test-ocloud",
					Status: &models.ResourcePoolStatus{
						State:  "CREATING",
						Health: "UNKNOWN",
					},
					Extensions: map[string]interface{}{
							"enabled":          true,
							"trackTransitions": true,
						},
					},
				}

				poolJSON, err := json.Marshal(stateTrackingPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("monitoring state transitions through lifecycle")
				expectedStates := []string{"CREATING", "AVAILABLE", "ACTIVE"}
				observedStates := make([]string, 0)

				Eventually(func() []string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID + "/state-history")
					if err != nil {
						return observedStates
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode != http.StatusOK {
						return observedStates
					}

					var stateHistory []map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&stateHistory)
					if err != nil {
						return observedStates
					}

					observedStates = make([]string, len(stateHistory))
					for i, state := range stateHistory {
						observedStates[i] = state["state"].(string)
					}

					return observedStates
				}, 3*time.Minute, 10*time.Second).Should(ContainElements(expectedStates))

				By("verifying state transition events are recorded")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID + "/events")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var events []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&events)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(len(events)).To(BeNumerically(">", 0))

				foundStateChangeEvent := false
				for _, event := range events {
					if event["eventType"] == "STATE_CHANGED" {
						foundStateChangeEvent = true
						Expect(event).To(HaveKey("timestamp"))
						Expect(event).To(HaveKey("previousState"))
						Expect(event).To(HaveKey("newState"))
						break
					}
				}
				Expect(foundStateChangeEvent).To(BeTrue())

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})
		})
	})
})


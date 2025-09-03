package integration_tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

var _ = Describe("O2 CNF Deployment Workflow Integration Tests", func() {
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
		testCtx, cancel = context.WithTimeout(ctx, 20*time.Minute)
		DeferCleanup(cancel)

		testLogger = logging.NewLogger("o2-cnf-test", "debug")
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
			CNFConfig: map[string]interface{}{
				"repositories": []map[string]interface{}{
					{
						"name": "nephoran-charts",
						"url":  "https://charts.nephoran.io",
					},
					{
						"name": "bitnami",
						"url":  "https://charts.bitnami.com/bitnami",
					},
				},
				"defaultTimeout": "10m",
				"retryPolicy":    json.RawMessage(`{}`),
			},
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		httpTestServer = httptest.NewServer(o2Server.GetRouter())
		testClient = httpTestServer.Client()
		testClient.Timeout = 120 * time.Second

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
		})
	})

	Describe("CNF Package Management", func() {
		Context("when managing CNF packages", func() {
			It("should support CNF package registration and discovery", func() {
				packageID := "test-cnf-package-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("registering a CNF package")
				cnfPackage := map[string]interface{}{
						"vnfdId":             "amf-vnfd-v1",
						"vnfdVersion":        "1.0",
						"vnfProvider":        "Nephoran",
						"vnfProductName":     "5G-AMF",
						"vnfSoftwareVersion": "1.0.0",
					},
					"packageType": "helm",
					"packageSource": json.RawMessage(`{}`),
					"operationalState": "ENABLED",
					"usageState":       "NOT_IN_USE",
					"onboardingState":  "ONBOARDED",
					"deploymentFlavors": []map[string]interface{}{
								"cpu":     "2",
								"memory":  "4Gi",
								"storage": "10Gi",
							},
						},
						{
							"id":          "medium",
							"description": "Medium deployment flavor",
							"requirements": json.RawMessage(`{}`),
						},
					},
					"softwareImages": []json.RawMessage(`{}`),
					},
				}

				packageJSON, err := json.Marshal(cnfPackage)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/packages",
					"application/json",
					bytes.NewBuffer(packageJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the registered CNF package")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/packages/" + packageID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedPackage map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&retrievedPackage)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedPackage["packageId"]).To(Equal(packageID))
				Expect(retrievedPackage["name"]).To(Equal("Test 5G AMF CNF"))
				Expect(retrievedPackage["operationalState"]).To(Equal("ENABLED"))

				vnfdInfo, ok := retrievedPackage["vnfdInfo"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(vnfdInfo["vnfProductName"]).To(Equal("5G-AMF"))

				By("listing all CNF packages")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/packages")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var packages []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&packages)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				foundPackage := false
				for _, pkg := range packages {
					if pkg["packageId"] == packageID {
						foundPackage = true
						break
					}
				}
				Expect(foundPackage).To(BeTrue())

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/packages/"+packageID, nil)
					testClient.Do(req)
				})
			})

			It("should validate CNF package content and dependencies", func() {
				packageID := "validation-cnf-package-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating CNF package with complex dependencies")
				complexCNF := map[string]interface{}{
						"vnfdId": "5g-core-vnfd-v2",
					},
					"dependencies": []json.RawMessage(`{}`),
						{
							"name":       "redis",
							"version":    ">=17.0.0",
							"repository": "bitnami",
							"required":   true,
						},
					},
					"networkServices": []json.RawMessage(`{}`),
						{
							"name":     "n2-interface",
							"type":     "service",
							"protocol": "SCTP",
							"port":     36412,
						},
					},
					"configMaps": []map[string]interface{}{
								"amf.yaml": "config: |\\n  amf:\\n    plmnList:\\n      - mcc: 001\\n        mnc: 01",
							},
						},
					},
				}

				packageJSON, err := json.Marshal(complexCNF)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/packages",
					"application/json",
					bytes.NewBuffer(packageJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("validating package dependencies")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/packages/" + packageID + "/dependencies")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var dependencies []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&dependencies)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(len(dependencies)).To(BeNumerically(">=", 2))

				dependencyNames := make([]string, len(dependencies))
				for i, dep := range dependencies {
					dependencyNames[i] = dep["name"].(string)
				}
				Expect(dependencyNames).To(ContainElement("mongodb"))
				Expect(dependencyNames).To(ContainElement("redis"))

				By("validating package network configuration")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/packages/" + packageID + "/network-services")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var networkServices []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&networkServices)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(len(networkServices)).To(Equal(2))

				serviceNames := make([]string, len(networkServices))
				for i, svc := range networkServices {
					serviceNames[i] = svc["name"].(string)
				}
				Expect(serviceNames).To(ContainElement("n1-interface"))
				Expect(serviceNames).To(ContainElement("n2-interface"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/packages/"+packageID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("CNF Lifecycle Management", func() {
		Context("when deploying CNF instances", func() {
			It("should support complete CNF deployment lifecycle", func() {
				// First register a CNF package
				packageID := "lifecycle-cnf-package-" + strconv.FormatInt(time.Now().UnixNano(), 10)
				instanceID := "lifecycle-cnf-instance-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("registering CNF package for deployment")
				cnfPackage := map[string]interface{}{
						"vnfdId": "lifecycle-test-vnfd",
					},
					"operationalState": "ENABLED",
					"deploymentFlavors": []map[string]interface{}{
								"cpu":    "1",
								"memory": "2Gi",
							},
						},
					},
				}

				packageJSON, err := json.Marshal(cnfPackage)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/packages",
					"application/json",
					bytes.NewBuffer(packageJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("instantiating CNF from package")
				instantiationRequest := map[string]interface{}{
						"namespace": namespace.Name,
						"values": map[string]interface{}{
								"repository": "nginx",
								"tag":        "latest",
								"pullPolicy": "IfNotPresent",
							},
							"resources": map[string]interface{}{
									"cpu":    "100m",
									"memory": "128Mi",
								},
								"limits": json.RawMessage(`{}`),
							},
						},
					},
				}

				instantiationJSON, err := json.Marshal(instantiationRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/instances",
					"application/json",
					bytes.NewBuffer(instantiationJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusAccepted)))

				var instantiationResponse map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&instantiationResponse)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				if id, ok := instantiationResponse["vnfInstanceId"]; ok {
					instanceID = id.(string)
				}

				By("monitoring CNF instantiation progress")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID)
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode != http.StatusOK {
						return ""
					}

					var instance map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&instance)
					if err != nil {
						return ""
					}

					if state, ok := instance["instantiationState"].(string); ok {
						return state
					}
					return ""
				}, 5*time.Minute, 15*time.Second).Should(Equal("INSTANTIATED"))

				By("verifying CNF instance details")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var cnfInstance map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&cnfInstance)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(cnfInstance["vnfInstanceName"]).To(Equal("Test CNF Instance"))
				Expect(cnfInstance["instantiationState"]).To(Equal("INSTANTIATED"))
				Expect(cnfInstance).To(HaveKey("vnfInstanceId"))

				By("scaling CNF instance")
				scaleRequest := map[string]interface{}{
						"replicaCount": 2,
					},
				}

				scaleJSON, err := json.Marshal(scaleRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/instances/"+instanceID+"/scale",
					"application/json",
					bytes.NewBuffer(scaleJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("verifying scale operation completion")
				Eventually(func() int {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID + "/resources")
					if err != nil {
						return 0
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode != http.StatusOK {
						return 0
					}

					var resources map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&resources)
					if err != nil {
						return 0
					}

					if replicas, ok := resources["replicaCount"]; ok {
						if replicaFloat, ok := replicas.(float64); ok {
							return int(replicaFloat)
						}
					}
					return 0
				}, 2*time.Minute, 10*time.Second).Should(Equal(2))

				By("updating CNF configuration")
				updateRequest := map[string]interface{}{
						"logLevel":       "DEBUG",
						"maxConnections": 1000,
					},
				}

				updateJSON, err := json.Marshal(updateRequest)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("PATCH",
					httpTestServer.URL+"/o2ims/v1/cnf/instances/"+instanceID,
					bytes.NewBuffer(updateJSON))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				resp, err = testClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("terminating CNF instance")
				terminationRequest := json.RawMessage(`{}`)

				terminationJSON, err := json.Marshal(terminationRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/instances/"+instanceID+"/terminate",
					"application/json",
					bytes.NewBuffer(terminationJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusAccepted), Equal(http.StatusOK)))
				resp.Body.Close()

				By("verifying CNF termination")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID)
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					if resp.StatusCode == http.StatusNotFound {
						return "NOT_INSTANTIATED"
					}

					var instance map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&instance)
					if err != nil {
						return ""
					}

					if state, ok := instance["instantiationState"].(string); ok {
						return state
					}
					return ""
				}, 3*time.Minute, 15*time.Second).Should(Equal("NOT_INSTANTIATED"))

				DeferCleanup(func() {
					// Cleanup package
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/packages/"+packageID, nil)
					testClient.Do(req)
				})
			})
		})

		Context("when handling CNF deployment failures", func() {
			It("should handle resource allocation failures gracefully", func() {
				packageID := "failure-test-package-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("registering CNF package with excessive resource requirements")
				cnfPackage := map[string]interface{}{
						"vnfdId": "resource-intensive-vnfd",
					},
					"operationalState": "ENABLED",
					"deploymentFlavors": []map[string]interface{}{
								"cpu":    "100",    // Intentionally excessive
								"memory": "1000Gi", // Intentionally excessive
							},
						},
					},
				}

				packageJSON, err := json.Marshal(cnfPackage)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/packages",
					"application/json",
					bytes.NewBuffer(packageJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("attempting to instantiate CNF with excessive resources")
				instantiationRequest := map[string]interface{}{
						"namespace": namespace.Name,
						"values": map[string]interface{}{
								"requests": json.RawMessage(`{}`),
							},
						},
					},
				}

				instantiationJSON, err := json.Marshal(instantiationRequest)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/instances",
					"application/json",
					bytes.NewBuffer(instantiationJSON),
				)
				Expect(err).NotTo(HaveOccurred())

				var instantiationResponse map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&instantiationResponse)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				instanceID, ok := instantiationResponse["vnfInstanceId"].(string)
				Expect(ok).To(BeTrue())

				By("verifying deployment failure is handled properly")
				Eventually(func() string {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID)
					if err != nil {
						return ""
					}
					defer resp.Body.Close() // #nosec G307 - Error handled in defer

					var instance map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&instance)
					if err != nil {
						return ""
					}

					if state, ok := instance["instantiationState"].(string); ok {
						return state
					}
					return ""
				}, 2*time.Minute, 10*time.Second).Should(Or(Equal("FAILED_TEMP"), Equal("NOT_INSTANTIATED")))

				By("verifying error information is provided")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/instances/" + instanceID + "/errors")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var errors []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errors)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(len(errors)).To(BeNumerically(">", 0))
				for _, errorInfo := range errors {
					Expect(errorInfo).To(HaveKey("errorType"))
					Expect(errorInfo).To(HaveKey("errorMessage"))
					Expect(errorInfo).To(HaveKey("timestamp"))
				}

				DeferCleanup(func() {
					// Cleanup
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/instances/"+instanceID, nil)
					testClient.Do(req)
					req, _ = http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/packages/"+packageID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("CNF Monitoring and Observability", func() {
		Context("when monitoring CNF instances", func() {
			It("should collect and expose CNF metrics", func() {
				By("checking CNF metrics endpoint")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/metrics")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var metrics map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&metrics)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(metrics).To(HaveKey("cnfInstances"))
				Expect(metrics).To(HaveKey("cnfPackages"))
				Expect(metrics).To(HaveKey("deploymentOperations"))

				By("verifying Prometheus metrics are exposed")
				resp, err = testClient.Get(httpTestServer.URL + "/metrics")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				metricsBody, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				metricsText := string(metricsBody)
				Expect(metricsText).To(ContainSubstring("o2ims_cnf_instances_total"))
				Expect(metricsText).To(ContainSubstring("o2ims_cnf_deployment_duration_seconds"))
				Expect(metricsText).To(ContainSubstring("o2ims_cnf_package_operations_total"))
			})
		})
	})

	Describe("CNF Template and Configuration Management", func() {
		Context("when managing CNF templates", func() {
			It("should support CNF template operations", func() {
				templateID := "test-cnf-template-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a CNF deployment template")
				cnfTemplate := map[string]interface{}{
						{
							"vnfdId":        "amf-vnfd",
							"name":          "AMF Network Function",
							"defaultFlavor": "medium",
						},
						{
							"vnfdId":        "smf-vnfd",
							"name":          "SMF Network Function",
							"defaultFlavor": "medium",
						},
					},
					"networkTopology": map[string]interface{}{
							{
								"name":      "n1-n2-interface",
								"type":      "internal",
								"endpoints": []string{"amf", "smf"},
							},
						},
					},
					"configurationTemplate": map[string]interface{}{
							"plmn": json.RawMessage(`{}`),
							"nsi": "default-slice",
						},
						"vnfConfigs": map[string]interface{}{
								"guamiList": []map[string]interface{}{
											"mcc": "001",
											"mnc": "01",
										},
										"amfId": "cafe00",
									},
								},
							},
						},
					},
				}

				templateJSON, err := json.Marshal(cnfTemplate)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/templates",
					"application/json",
					bytes.NewBuffer(templateJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("retrieving the created template")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/cnf/templates/" + templateID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedTemplate map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&retrievedTemplate)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(retrievedTemplate["templateId"]).To(Equal(templateID))
				Expect(retrievedTemplate["category"]).To(Equal("5G-Core"))

				vnfds, ok := retrievedTemplate["vnfds"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(len(vnfds)).To(Equal(2))

				By("instantiating from template")
				templateInstantiation := map[string]interface{}{
						"globalConfig": map[string]interface{}{
								"mcc": "001",
								"mnc": "01",
							},
						},
						"namespace": namespace.Name,
					},
				}

				instantiationJSON, err := json.Marshal(templateInstantiation)
				Expect(err).NotTo(HaveOccurred())

				resp, err = testClient.Post(
					httpTestServer.URL+"/o2ims/v1/cnf/templates/"+templateID+"/instantiate",
					"application/json",
					bytes.NewBuffer(instantiationJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusAccepted)))

				var templateInstanceResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&templateInstanceResp)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(templateInstanceResp).To(HaveKey("instanceId"))
				instanceID := templateInstanceResp["instanceId"].(string)

				DeferCleanup(func() {
					// Cleanup template instance and template
					req, _ := http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/instances/"+instanceID, nil)
					testClient.Do(req)
					req, _ = http.NewRequest("DELETE",
						httpTestServer.URL+"/o2ims/v1/cnf/templates/"+templateID, nil)
					testClient.Do(req)
				})
			})
		})
	})
})


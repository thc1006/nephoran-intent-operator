package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

var _ = Describe("O2 Multi-Cloud Provider Integration Tests", func() {
	var (
		namespace         *corev1.Namespace
		testCtx          context.Context
		o2Server         *o2.O2APIServer
		httpTestServer   *httptest.Server
		testClient       *http.Client
		metricsRegistry  *prometheus.Registry
		testLogger       *logging.StructuredLogger
		providerManager  *providers.IntegrationManager
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 15*time.Minute)
		DeferCleanup(cancel)

		testLogger = logging.NewLogger("o2-multicloud-test", "debug")
		metricsRegistry = prometheus.NewRegistry()

		// Setup O2 IMS service with multi-cloud provider configuration
		config := &o2.O2IMSConfig{
			ServerAddress:    "127.0.0.1",
			ServerPort:       0,
			TLSEnabled:       false,
			DatabaseConfig: map[string]interface{}{
				"type":     "memory",
				"database": "o2_multicloud_test_db",
			},
			ProviderConfigs: map[string]interface{}{
				"kubernetes": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"kubeconfig": "",
						"namespace": namespace.Name,
					},
				},
				"openstack": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"auth_url":    "http://mock-openstack.test:5000/v3",
						"username":    "test-user",
						"password":    "test-password",
						"project":     "test-project",
						"region":      "RegionOne",
						"mock_mode":   true,
					},
				},
				"aws": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"region":          "us-west-2",
						"access_key_id":   "test-access-key",
						"secret_access_key": "test-secret-key",
						"mock_mode":       true,
					},
				},
				"azure": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"subscription_id": "test-subscription",
						"tenant_id":       "test-tenant",
						"client_id":       "test-client",
						"client_secret":   "test-secret",
						"mock_mode":       true,
					},
				},
				"gcp": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"project":       "test-project",
						"region":        "us-central1",
						"zone":          "us-central1-a",
						"credentials":   "test-credentials.json",
						"mock_mode":     true,
					},
				},
				"vmware": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"vcenter_url": "https://mock-vcenter.test",
						"username":    "administrator@vsphere.local",
						"password":    "test-password",
						"datacenter":  "test-datacenter",
						"mock_mode":   true,
					},
				},
				"edge": map[string]interface{}{
					"enabled": true,
					"config": map[string]interface{}{
						"edge_nodes": []map[string]interface{}{
							{
								"name":     "edge-node-1",
								"location": "cell-tower-1",
								"endpoint": "http://edge-node-1.test:8080",
							},
							{
								"name":     "edge-node-2",
								"location": "cell-tower-2", 
								"endpoint": "http://edge-node-2.test:8080",
							},
						},
						"mock_mode": true,
					},
				},
			},
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		providerManager = o2Server.GetProviderManager()
		Expect(providerManager).NotTo(BeNil())

		httpTestServer = httptest.NewServer(o2Server.GetRouter())
		testClient = httpTestServer.Client()
		testClient.Timeout = 60 * time.Second

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
		})
	})

	Describe("Provider Discovery and Registration", func() {
		Context("when discovering available providers", func() {
			It("should list all configured cloud providers", func() {
				By("calling GET /o2ims/v1/providers")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/providers")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var providers []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&providers)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				By("validating provider list contains expected providers")
				providerNames := make([]string, len(providers))
				for i, provider := range providers {
					name, ok := provider["name"].(string)
					Expect(ok).To(BeTrue())
					providerNames[i] = name
				}

				expectedProviders := []string{"kubernetes", "openstack", "aws", "azure", "gcp", "vmware", "edge"}
				for _, expected := range expectedProviders {
					Expect(providerNames).To(ContainElement(expected))
				}

				By("validating provider details")
				for _, provider := range providers {
					Expect(provider).To(HaveKey("name"))
					Expect(provider).To(HaveKey("type"))
					Expect(provider).To(HaveKey("status"))
					Expect(provider).To(HaveKey("capabilities"))
					Expect(provider).To(HaveKey("region"))
					
					status, ok := provider["status"].(string)
					Expect(ok).To(BeTrue())
					Expect(status).To(Or(Equal("AVAILABLE"), Equal("UNAVAILABLE"), Equal("MAINTENANCE")))
				}
			})

			It("should provide detailed provider information", func() {
				By("getting detailed information for Kubernetes provider")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/providers/kubernetes")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var k8sProvider map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&k8sProvider)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(k8sProvider["name"]).To(Equal("kubernetes"))
				Expect(k8sProvider["type"]).To(Equal("container-orchestration"))
				
				capabilities, ok := k8sProvider["capabilities"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(capabilities).To(ContainElement("container-deployment"))
				Expect(capabilities).To(ContainElement("auto-scaling"))
				Expect(capabilities).To(ContainElement("load-balancing"))

				By("getting detailed information for AWS provider")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/providers/aws")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var awsProvider map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&awsProvider)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(awsProvider["name"]).To(Equal("aws"))
				Expect(awsProvider["type"]).To(Equal("public-cloud"))
				
				capabilities, ok = awsProvider["capabilities"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(capabilities).To(ContainElement("vm-deployment"))
				Expect(capabilities).To(ContainElement("container-services"))
				Expect(capabilities).To(ContainElement("managed-databases"))
			})
		})
	})

	Describe("Provider-Specific Resource Operations", func() {
		Context("when managing resources across different providers", func() {
			It("should create resource pools for each provider type", func() {
				providers := []struct {
					name     string
					poolType string
				}{
					{"kubernetes", "container-pool"},
					{"aws", "ec2-pool"},
					{"azure", "vm-pool"},
					{"gcp", "compute-pool"},
					{"openstack", "nova-pool"},
					{"vmware", "vsphere-pool"},
					{"edge", "edge-pool"},
				}

				var createdPools []string

				for _, provider := range providers {
					By(fmt.Sprintf("creating resource pool for %s provider", provider.name))
					poolID := fmt.Sprintf("%s-test-pool-%d", provider.name, time.Now().UnixNano())
					createdPools = append(createdPools, poolID)

					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:          fmt.Sprintf("%s Test Pool", provider.name),
						Description:   fmt.Sprintf("Integration test pool for %s provider", provider.name),
						Provider:      provider.name,
						OCloudID:      "test-ocloud-" + provider.name,
						Location:      fmt.Sprintf("%s-region", provider.name),
						Extensions: map[string]interface{}{
							"poolType": provider.poolType,
							"providerSpecific": map[string]interface{}{
								"mockMode": true,
								"testEnvironment": true,
							},
						},
						Capacity: &models.ResourceCapacity{
							CPU: &models.ResourceMetric{
								Total:       "1000",
								Available:   "800",
								Used:        "200",
								Unit:        "cores",
								Utilization: 20.0,
							},
							Memory: &models.ResourceMetric{
								Total:       "4000Gi",
								Available:   "3200Gi",
								Used:        "800Gi",
								Unit:        "bytes",
								Utilization: 20.0,
							},
						},
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

					By(fmt.Sprintf("verifying created %s resource pool", provider.name))
					resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					var retrievedPool models.ResourcePool
					err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()

					Expect(retrievedPool.ResourcePoolID).To(Equal(poolID))
					Expect(retrievedPool.Provider).To(Equal(provider.name))
				}

				DeferCleanup(func() {
					By("cleaning up created resource pools")
					for _, poolID := range createdPools {
						req, _ := http.NewRequest("DELETE", 
							httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)
					}
				})

				By("verifying all pools can be filtered by provider")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var allPools []models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&allPools)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				// Verify we can find pools for each provider
				providerCounts := make(map[string]int)
				for _, pool := range allPools {
					if pool.Extensions != nil {
						if testEnv, ok := pool.Extensions["providerSpecific"].(map[string]interface{}); ok {
							if testEnv["testEnvironment"] == true {
								providerCounts[pool.Provider]++
							}
						}
					}
				}

				for _, provider := range providers {
					Expect(providerCounts[provider.name]).To(BeNumerically(">=", 1),
						fmt.Sprintf("No pools found for provider %s", provider.name))
				}
			})
		})

		Context("when testing provider-specific capabilities", func() {
			It("should validate Kubernetes-specific operations", func() {
				poolID := "k8s-capability-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating Kubernetes resource pool with container-specific capabilities")
				k8sPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:          "K8s Capability Test Pool",
					Provider:      "kubernetes",
					OCloudID:      "test-k8s-ocloud",
					Extensions: map[string]interface{}{
						"kubernetesConfig": map[string]interface{}{
							"namespace":      namespace.Name,
							"storageClass":   "standard",
							"ingressClass":   "nginx",
							"serviceType":    "ClusterIP",
						},
						"supportedWorkloads": []string{
							"deployment", "statefulset", "daemonset", "job", "cronjob",
						},
					},
				}

				poolJSON, err := json.Marshal(k8sPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("validating Kubernetes-specific resource types")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourceTypes?filter=provider,eq,kubernetes")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var k8sResourceTypes []models.ResourceType
				err = json.NewDecoder(resp.Body).Decode(&k8sResourceTypes)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				// Should have Kubernetes-specific resource types
				expectedTypes := []string{"pod", "deployment", "service", "persistent-volume"}
				actualTypeNames := make([]string, len(k8sResourceTypes))
				for i, rt := range k8sResourceTypes {
					actualTypeNames[i] = rt.Name
				}

				for _, expectedType := range expectedTypes {
					found := false
					for _, actualType := range actualTypeNames {
						if strings.Contains(strings.ToLower(actualType), expectedType) {
							found = true
							break
						}
					}
					Expect(found).To(BeTrue(), fmt.Sprintf("Expected resource type %s not found", expectedType))
				}

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", 
						httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})

			It("should validate AWS-specific operations", func() {
				poolID := "aws-capability-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating AWS resource pool with cloud-specific capabilities")
				awsPool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:          "AWS Capability Test Pool",
					Provider:      "aws",
					Region:        "us-west-2",
					Zone:          "us-west-2a",
					OCloudID:      "test-aws-ocloud",
					Extensions: map[string]interface{}{
						"awsConfig": map[string]interface{}{
							"vpcId":           "vpc-test123",
							"subnetId":        "subnet-test123",
							"securityGroupId": "sg-test123",
							"keyPairName":     "test-keypair",
						},
						"supportedInstanceTypes": []string{
							"t3.micro", "t3.small", "t3.medium", "c5.large", "m5.xlarge",
						},
						"supportedServices": []string{
							"ec2", "ecs", "eks", "rds", "elasticache", "lambda",
						},
					},
				}

				poolJSON, err := json.Marshal(awsPool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("validating AWS-specific monitoring capabilities")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/providers/aws/capabilities")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var awsCapabilities map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&awsCapabilities)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				monitoring, ok := awsCapabilities["monitoring"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(monitoring).To(HaveKey("cloudwatch"))
				Expect(monitoring).To(HaveKey("xray"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", 
						httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})

			It("should validate Edge provider operations", func() {
				poolID := "edge-capability-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating Edge resource pool with edge-specific capabilities")
				edgePool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:          "Edge Capability Test Pool",
					Provider:      "edge",
					Location:      "cell-tower-site-1",
					OCloudID:      "test-edge-ocloud",
					Extensions: map[string]interface{}{
						"edgeConfig": map[string]interface{}{
							"siteType":        "cell-tower",
							"connectivity":    []string{"5G", "4G", "fiber"},
							"powerSource":     "grid-with-backup",
							"environmentRating": "outdoor",
							"computeCapability": map[string]interface{}{
								"cpuType":      "arm64",
								"accelerator":  "gpu",
								"storage":      "nvme-ssd",
								"network":      "dpdk-capable",
							},
						},
						"supportedFunctions": []string{
							"ran-function", "mec-app", "cdn-cache", "ai-inference",
						},
						"latencyRequirements": map[string]interface{}{
							"ultra-low":  "< 1ms",
							"low":       "< 10ms", 
							"medium":    "< 50ms",
						},
					},
				}

				poolJSON, err := json.Marshal(edgePool)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
				resp.Body.Close()

				By("validating edge-specific resource characteristics")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var retrievedPool models.ResourcePool
				err = json.NewDecoder(resp.Body).Decode(&retrievedPool)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				edgeConfig, ok := retrievedPool.Extensions["edgeConfig"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(edgeConfig["siteType"]).To(Equal("cell-tower"))

				computeCapability, ok := edgeConfig["computeCapability"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(computeCapability["cpuType"]).To(Equal("arm64"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", 
						httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("Cross-Provider Operations", func() {
		Context("when managing resources across multiple providers", func() {
			It("should support hybrid deployments across providers", func() {
				deploymentID := "hybrid-deployment-" + strconv.FormatInt(time.Now().UnixNano(), 10)

				By("creating a hybrid deployment specification")
				hybridDeployment := map[string]interface{}{
					"deploymentId": deploymentID,
					"name":        "Hybrid Multi-Cloud Deployment",
					"description": "Test deployment spanning multiple cloud providers",
					"components": []map[string]interface{}{
						{
							"name":     "core-services",
							"provider": "kubernetes",
							"region":   "us-west-2",
							"resources": map[string]interface{}{
								"cpu":    "4",
								"memory": "8Gi",
								"replicas": 3,
							},
						},
						{
							"name":     "data-processing",
							"provider": "aws",
							"region":   "us-west-2",
							"resources": map[string]interface{}{
								"instanceType": "c5.2xlarge",
								"minInstances": 2,
								"maxInstances": 10,
							},
						},
						{
							"name":     "edge-functions",
							"provider": "edge",
							"location": "cell-tower-sites",
							"resources": map[string]interface{}{
								"cpu":      "2",
								"memory":   "4Gi",
								"latency":  "< 5ms",
								"replicas": 5,
							},
						},
					},
					"networking": map[string]interface{}{
						"connectivity": "vpn-mesh",
						"bandwidth":   "10Gbps",
						"protocol":    "service-mesh",
					},
				}

				deploymentJSON, err := json.Marshal(hybridDeployment)
				Expect(err).NotTo(HaveOccurred())

				resp, err := testClient.Post(
					httpTestServer.URL+"/o2ims/v1/deployments",
					"application/json",
					bytes.NewBuffer(deploymentJSON),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusAccepted)))
				resp.Body.Close()

				By("validating deployment status across providers")
				Eventually(func() map[string]interface{} {
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/deployments/" + deploymentID)
					if err != nil {
						return nil
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						return nil
					}

					var deployment map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&deployment)
					if err != nil {
						return nil
					}
					return deployment
				}, 2*time.Minute, 10*time.Second).Should(And(
					HaveKey("deploymentId"),
					HaveKey("status"),
				))

				By("verifying cross-provider resource allocation")
				resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/deployments/" + deploymentID + "/resources")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var resources []map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&resources)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				providers := make(map[string]bool)
				for _, resource := range resources {
					if provider, ok := resource["provider"].(string); ok {
						providers[provider] = true
					}
				}

				Expect(providers).To(HaveKey("kubernetes"))
				Expect(providers).To(HaveKey("aws"))
				Expect(providers).To(HaveKey("edge"))

				DeferCleanup(func() {
					req, _ := http.NewRequest("DELETE", 
						httpTestServer.URL+"/o2ims/v1/deployments/"+deploymentID, nil)
					testClient.Do(req)
				})
			})
		})
	})

	Describe("Provider Health and Status Monitoring", func() {
		Context("when monitoring provider health", func() {
			It("should track provider availability and health metrics", func() {
				By("checking overall provider health status")
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/providers/health")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var healthStatus map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&healthStatus)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				Expect(healthStatus).To(HaveKey("overall"))
				Expect(healthStatus).To(HaveKey("providers"))

				providers, ok := healthStatus["providers"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				By("validating individual provider health")
				expectedProviders := []string{"kubernetes", "aws", "azure", "gcp", "openstack", "vmware", "edge"}
				for _, provider := range expectedProviders {
					Expect(providers).To(HaveKey(provider))
					providerHealth, ok := providers[provider].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(providerHealth).To(HaveKey("status"))
					Expect(providerHealth).To(HaveKey("lastCheck"))
					Expect(providerHealth).To(HaveKey("responseTime"))
				}

				By("checking provider metrics")
				resp, err = testClient.Get(httpTestServer.URL + "/metrics")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				metricsBody, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()

				metricsText := string(metricsBody)
				Expect(metricsText).To(ContainSubstring("o2ims_provider_health"))
				Expect(metricsText).To(ContainSubstring("o2ims_provider_requests_total"))
				Expect(metricsText).To(ContainSubstring("o2ims_provider_response_time"))
			})
		})
	})
})
/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cnf

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/cnf"
)

var _ = Describe("CNF Deployment Integration Tests", func() {
	var (
		ctx                context.Context
		namespace          string
		cnfOrchestrator    *cnf.CNFOrchestrator
		cnfIntentProcessor *cnf.CNFIntentProcessor
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "test-cnf-" + randString(8)

		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		// Initialize CNF components
		cnfOrchestrator = cnf.NewCNFOrchestrator(k8sClient, scheme, recorder)
		cnfIntentProcessor = cnf.NewCNFIntentProcessor(k8sClient, nil, nil)
	})

	AfterEach(func() {
		// Clean up test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	Context("5G Core CNF Deployment", func() {
		It("should successfully deploy AMF CNF via NetworkIntent", func() {
			By("Creating a NetworkIntent for AMF deployment")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-amf-intent",
					Namespace: namespace,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent:     "Deploy an AMF function with high availability and auto-scaling for production use",
					IntentType: nephoranv1.IntentTypeDeployment,
					Priority:   nephoranv1.NetworkPriorityHigh,
					TargetComponents: []nephoranv1.NetworkTargetComponent{
						nephoranv1.NetworkTargetComponentAMF,
					},
					ResourceConstraints: &nephoranv1.NetworkResourceConstraints{
						MaxCPU:     "1000m",
						MaxMemory:  "2Gi",
						MaxStorage: "10Gi",
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Processing the NetworkIntent for CNF deployment")
			result, err := cnfIntentProcessor.ProcessCNFIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.DetectedFunctions).To(ContainElement(nephoranv1.CNFFunctionAMF))
			Expect(len(result.CNFDeployments)).To(BeNumerically(">", 0))
			Expect(result.ConfidenceScore).To(BeNumerically(">=", 0.7))

			By("Creating CNFDeployment from the processing result")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-amf-cnf",
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.com/cnf-type":     "5G-Core",
						"nephoran.com/cnf-function": "AMF",
					},
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionAMF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect, // Use direct for testing
					Replicas:           2,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
					AutoScaling: &nephoranv1.AutoScaling{
						Enabled:        true,
						MinReplicas:    1,
						MaxReplicas:    5,
						CPUUtilization: ptr(int32(70)),
					},
					Monitoring: &nephoranv1.MonitoringConfig{
						Enabled: true,
						Prometheus: &nephoranv1.PrometheusConfig{
							Enabled: true,
							Port:    9090,
						},
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, cnfDeployment)).To(Succeed())

			By("Validating CNFDeployment creation and status updates")
			Eventually(func() string {
				updated := &nephoranv1.CNFDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      cnfDeployment.Name,
					Namespace: cnfDeployment.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(BeElementOf("Deploying", "Running"))

			By("Verifying CNF deployment validation")
			Expect(cnfDeployment.ValidateCNFDeployment()).To(Succeed())
		})

		It("should successfully deploy UPF CNF with DPDK configuration", func() {
			By("Creating a CNFDeployment for UPF with DPDK")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-upf-cnf",
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.com/cnf-type":     "5G-Core",
						"nephoran.com/cnf-function": "UPF",
					},
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionUPF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           1,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("4000m"),
						Memory: mustParseQuantity("8Gi"),
						DPDK: &nephoranv1.DPDKConfig{
							Enabled: true,
							Cores:   ptr(int32(4)),
							Memory:  ptr(int32(2048)),
							Driver:  "vfio-pci",
						},
						Hugepages: map[string]resource.Quantity{
							"2Mi": mustParseQuantity("2Gi"),
							"1Gi": mustParseQuantity("4Gi"),
						},
					},
					Monitoring: &nephoranv1.MonitoringConfig{
						Enabled: true,
						CustomMetrics: []string{
							"upf_packet_throughput",
							"upf_session_count",
						},
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, cnfDeployment)).To(Succeed())
			Expect(cnfDeployment.ValidateCNFDeployment()).To(Succeed())

			By("Verifying DPDK configuration is properly set")
			Expect(cnfDeployment.Spec.Resources.DPDK).NotTo(BeNil())
			Expect(cnfDeployment.Spec.Resources.DPDK.Enabled).To(BeTrue())
			Expect(*cnfDeployment.Spec.Resources.DPDK.Cores).To(Equal(int32(4)))
			Expect(cnfDeployment.Spec.Resources.Hugepages).To(HaveLen(2))
		})
	})

	Context("O-RAN CNF Deployment", func() {
		It("should successfully deploy Near-RT RIC CNF", func() {
			By("Creating a NetworkIntent for O-RAN deployment")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-oran-intent",
					Namespace: namespace,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent:     "Deploy Near-RT RIC for intelligent RAN control with xApp support and E2 interface",
					IntentType: nephoranv1.IntentTypeDeployment,
					Priority:   nephoranv1.PriorityMedium,
					TargetComponents: []nephoranv1.TargetComponent{
						nephoranv1.TargetComponentNearRTRIC,
					},
					ResourceConstraints: &nephoranv1.ResourceConstraints{
						CPU:    mustParseQuantity("2000m"),
						Memory: mustParseQuantity("4Gi"),
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Creating CNFDeployment for Near-RT RIC")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-near-rt-ric-cnf",
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.com/cnf-type":     "O-RAN",
						"nephoran.com/cnf-function": "Near-RT-RIC",
					},
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNFORAN,
					Function:           nephoranv1.CNFFunctionNearRTRIC,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           1,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("2000m"),
						Memory: mustParseQuantity("4Gi"),
					},
					ServiceMesh: &nephoranv1.ServiceMeshConfig{
						Enabled: true,
						Type:    "istio",
						MTLS: &nephoranv1.MTLSConfig{
							Enabled: true,
							Mode:    "strict",
						},
					},
					Monitoring: &nephoranv1.MonitoringConfig{
						Enabled: true,
						CustomMetrics: []string{
							"ric_active_xapps",
							"ric_e2_subscriptions",
							"ric_a1_policy_executions",
						},
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, cnfDeployment)).To(Succeed())
			Expect(cnfDeployment.ValidateCNFDeployment()).To(Succeed())

			By("Verifying service mesh configuration")
			Expect(cnfDeployment.Spec.ServiceMesh).NotTo(BeNil())
			Expect(cnfDeployment.Spec.ServiceMesh.Enabled).To(BeTrue())
			Expect(cnfDeployment.Spec.ServiceMesh.Type).To(Equal("istio"))
		})

		It("should successfully deploy O-DU CNF with high-performance networking", func() {
			By("Creating CNFDeployment for O-DU")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-odu-cnf",
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.com/cnf-type":     "O-RAN",
						"nephoran.com/cnf-function": "O-DU",
					},
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNFORAN,
					Function:           nephoranv1.CNFFunctionODU,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           2,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("3000m"),
						Memory: mustParseQuantity("6Gi"),
						DPDK: &nephoranv1.DPDKConfig{
							Enabled: true,
							Cores:   ptr(int32(4)),
							Memory:  ptr(int32(2048)),
							Driver:  "vfio-pci",
						},
						Hugepages: map[string]resource.Quantity{
							"2Mi": mustParseQuantity("2Gi"),
						},
					},
					AutoScaling: &nephoranv1.AutoScaling{
						Enabled:     true,
						MinReplicas: 1,
						MaxReplicas: 4,
						CustomMetrics: []nephoranv1.CustomMetric{
							{
								Name:        "odu_connected_ues",
								Type:        "pods",
								TargetValue: "100",
							},
						},
					},
					TargetNamespace: namespace,
				},
			}

			Expect(k8sClient.Create(ctx, cnfDeployment)).To(Succeed())
			Expect(cnfDeployment.ValidateCNFDeployment()).To(Succeed())

			By("Verifying DPDK and auto-scaling configuration")
			Expect(cnfDeployment.Spec.Resources.DPDK.Enabled).To(BeTrue())
			Expect(cnfDeployment.Spec.AutoScaling.Enabled).To(BeTrue())
			Expect(len(cnfDeployment.Spec.AutoScaling.CustomMetrics)).To(Equal(1))
		})
	})

	Context("CNF Validation and Error Handling", func() {
		It("should reject CNF deployment with incompatible function and type", func() {
			By("Creating CNFDeployment with mismatched function and type")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-cnf",
					Namespace: namespace,
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,      // 5G Core type
					Function:           nephoranv1.CNFFunctionODU, // O-RAN function
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           1,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
				},
			}

			By("Validating CNF deployment should fail")
			err := cnfDeployment.ValidateCNFDeployment()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not compatible with CNF type"))
		})

		It("should reject CNF deployment with invalid resource constraints", func() {
			By("Creating CNFDeployment with invalid resource limits")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-resources-cnf",
					Namespace: namespace,
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionAMF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           1,
					Resources: nephoranv1.CNFResources{
						CPU:       mustParseQuantity("2000m"),
						Memory:    mustParseQuantity("4Gi"),
						MaxCPU:    &[]resource.Quantity{mustParseQuantity("1000m")}[0], // Max less than min
						MaxMemory: &[]resource.Quantity{mustParseQuantity("2Gi")}[0],   // Max less than min
					},
				},
			}

			By("Validating CNF deployment should fail")
			err := cnfDeployment.ValidateCNFDeployment()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be greater than or equal to"))
		})

		It("should reject CNF deployment with invalid auto-scaling configuration", func() {
			By("Creating CNFDeployment with invalid auto-scaling")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-autoscaling-cnf",
					Namespace: namespace,
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionAMF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           5, // Outside auto-scaling range
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
					AutoScaling: &nephoranv1.AutoScaling{
						Enabled:     true,
						MinReplicas: 2,
						MaxReplicas: 4, // Max less than current replicas
					},
				},
			}

			By("Validating CNF deployment should fail")
			err := cnfDeployment.ValidateCNFDeployment()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("replicas must be between minReplicas and maxReplicas"))
		})
	})

	Context("CNF Template Integration", func() {
		It("should successfully retrieve and validate CNF templates", func() {
			By("Creating template manager")
			templateManager := cnf.NewHelmTemplateManager()

			By("Retrieving AMF template")
			amfTemplate, err := templateManager.GetTemplate(nephoranv1.CNFFunctionAMF)
			Expect(err).NotTo(HaveOccurred())
			Expect(amfTemplate).NotTo(BeNil())
			Expect(amfTemplate.Function).To(Equal(nephoranv1.CNFFunctionAMF))
			Expect(amfTemplate.ChartName).To(Equal("amf"))

			By("Retrieving UPF template")
			upfTemplate, err := templateManager.GetTemplate(nephoranv1.CNFFunctionUPF)
			Expect(err).NotTo(HaveOccurred())
			Expect(upfTemplate).NotTo(BeNil())
			Expect(upfTemplate.Function).To(Equal(nephoranv1.CNFFunctionUPF))
			Expect(upfTemplate.ChartName).To(Equal("upf"))

			By("Retrieving Near-RT RIC template")
			ricTemplate, err := templateManager.GetTemplate(nephoranv1.CNFFunctionNearRTRIC)
			Expect(err).NotTo(HaveOccurred())
			Expect(ricTemplate).NotTo(BeNil())
			Expect(ricTemplate.Function).To(Equal(nephoranv1.CNFFunctionNearRTRIC))
			Expect(ricTemplate.ChartName).To(Equal("near-rt-ric"))

			By("Verifying supported functions")
			supportedFunctions := templateManager.GetSupportedFunctions()
			Expect(len(supportedFunctions)).To(BeNumerically(">", 0))
			Expect(supportedFunctions).To(ContainElement(nephoranv1.CNFFunctionAMF))
			Expect(supportedFunctions).To(ContainElement(nephoranv1.CNFFunctionUPF))
			Expect(supportedFunctions).To(ContainElement(nephoranv1.CNFFunctionNearRTRIC))
		})

		It("should generate valid Helm values from CNF deployment", func() {
			By("Creating CNFDeployment")
			cnfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-values-cnf",
					Namespace: namespace,
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionAMF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyHelm,
					Replicas:           3,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
					AutoScaling: &nephoranv1.AutoScaling{
						Enabled:        true,
						MinReplicas:    2,
						MaxReplicas:    6,
						CPUUtilization: ptr(int32(70)),
					},
					ServiceMesh: &nephoranv1.ServiceMeshConfig{
						Enabled: true,
						Type:    "istio",
					},
					Monitoring: &nephoranv1.MonitoringConfig{
						Enabled: true,
						Prometheus: &nephoranv1.PrometheusConfig{
							Enabled: true,
							Port:    9090,
						},
					},
				},
			}

			By("Generating Helm values")
			templateManager := cnf.NewHelmTemplateManager()
			values, err := templateManager.GenerateValues(cnfDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).NotTo(BeNil())

			By("Verifying generated values")
			Expect(values["replicaCount"]).To(Equal(int32(3)))

			resources, ok := values["resources"].(map[string]interface{})
			Expect(ok).To(BeTrue())

			requests, ok := resources["requests"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(requests["cpu"]).To(Equal("1000m"))
			Expect(requests["memory"]).To(Equal("2Gi"))

			serviceMesh, ok := values["serviceMesh"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(serviceMesh["enabled"]).To(Equal(true))
			Expect(serviceMesh["type"]).To(Equal("istio"))
		})
	})

	Context("Performance and Load Testing", func() {
		It("should handle multiple concurrent CNF deployments", func() {
			By("Creating multiple CNF deployments concurrently")
			const numDeployments = 5
			deployments := make([]*nephoranv1.CNFDeployment, numDeployments)

			for i := 0; i < numDeployments; i++ {
				deployments[i] = &nephoranv1.CNFDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-concurrent-cnf-%d", i),
						Namespace: namespace,
					},
					Spec: nephoranv1.CNFDeploymentSpec{
						CNFType:            nephoranv1.CNF5GCore,
						Function:           nephoranv1.CNFFunctionAMF,
						DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
						Replicas:           1,
						Resources: nephoranv1.CNFResources{
							CPU:    mustParseQuantity("500m"),
							Memory: mustParseQuantity("1Gi"),
						},
						TargetNamespace: namespace,
					},
				}
			}

			By("Creating deployments concurrently")
			for i := 0; i < numDeployments; i++ {
				go func(cnf *nephoranv1.CNFDeployment) {
					defer GinkgoRecover()
					Expect(k8sClient.Create(ctx, cnf)).To(Succeed())
				}(deployments[i])
			}

			By("Verifying all deployments are created")
			Eventually(func() int {
				deploymentList := &nephoranv1.CNFDeploymentList{}
				err := k8sClient.List(ctx, deploymentList, client.InNamespace(namespace))
				if err != nil {
					return 0
				}
				return len(deploymentList.Items)
			}, timeout, interval).Should(Equal(numDeployments))
		})
	})
})

// Helper functions
func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse quantity %s: %v", s, err))
	}
	return q
}

func ptr[T any](v T) *T {
	return &v
}

func randString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(result)
}

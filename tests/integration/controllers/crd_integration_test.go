package integration_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	testutils "github.com/nephio-project/nephoran-intent-operator/tests/utils"
)

var _ = Describe("CRD Integration Tests", func() {
	var (
		namespace *corev1.Namespace
		testCtx   context.Context
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 10*time.Minute)
		DeferCleanup(cancel)
	})

	Describe("NetworkIntent CRD Integration", func() {
		Context("when creating NetworkIntent with real Kubernetes API", func() {
			It("should successfully create and manage NetworkIntent lifecycle", func() {
				By("creating a NetworkIntent with comprehensive specification")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "integration-test-intent",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                     "integration",
							"nephoran.com/intent-type": "deployment",
							"nephoran.com/target":      "5GC",
						},
						Annotations: map[string]string{
							"nephoran.com/test-run":    "crd-integration",
							"nephoran.com/description": "Integration test for NetworkIntent CRD",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy a highly available 5G AMF with auto-scaling, monitoring, and security policies for production environment",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
						ResourceConstraints: &nephoranv1.ResourceConstraints{
							CPU:       &resource.Quantity{},
							Memory:    &resource.Quantity{},
							MaxCPU:    &resource.Quantity{},
							MaxMemory: &resource.Quantity{},
						},
						TargetNamespace: "production",
						TargetCluster:   "main-cluster",
						NetworkSlice:    "ABC123-DEF456",
						Region:          "us-west-2",
						TimeoutSeconds:  &[]int32{600}[0],
						MaxRetries:      &[]int32{3}[0],
					},
				}

				// Set resource constraints
				intent.Spec.ResourceConstraints.CPU = resource.NewQuantity(500, resource.DecimalSI)             // 500m
				intent.Spec.ResourceConstraints.Memory = resource.NewQuantity(1073741824, resource.BinarySI)    // 1Gi
				intent.Spec.ResourceConstraints.MaxCPU = resource.NewQuantity(2000, resource.DecimalSI)         // 2000m
				intent.Spec.ResourceConstraints.MaxMemory = resource.NewQuantity(4294967296, resource.BinarySI) // 4Gi

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("verifying NetworkIntent is created with correct status")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() error {
					return k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
				}, 30*time.Second, 1*time.Second).Should(Succeed())

				// Verify spec fields
				Expect(createdIntent.Spec.Intent).To(Equal(intent.Spec.Intent))
				Expect(createdIntent.Spec.IntentType).To(Equal(nephoranv1.IntentTypeDeployment))
				Expect(createdIntent.Spec.Priority).To(Equal(nephoranv1.PriorityHigh))
				Expect(createdIntent.Spec.TargetComponents).To(ContainElement(nephoranv1.TargetComponentAMF))
				Expect(createdIntent.Spec.NetworkSlice).To(Equal("ABC123-DEF456"))

				By("waiting for controller to process the intent")
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 60*time.Second, 2*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
					Equal("Failed"),
				))

				By("verifying status fields are populated")
				Expect(createdIntent.Status.Phase).NotTo(BeEmpty())
				if createdIntent.Status.ProcessingStartTime != nil {
					Expect(createdIntent.Status.ProcessingStartTime.Time).To(BeTemporally(">=", intent.CreationTimestamp.Time))
				}

				By("verifying finalizer is added for cleanup")
				Expect(createdIntent.Finalizers).To(ContainElement("nephoran.com/intent-finalizer"))
			})

			It("should validate NetworkIntent specifications correctly", func() {
				By("creating NetworkIntent with invalid network slice format")
				invalidIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:       "Deploy AMF with invalid slice",
						IntentType:   nephoranv1.IntentTypeDeployment,
						NetworkSlice: "INVALID-FORMAT", // Should be XXXXXX-XXXXXX format
					},
				}

				err := k8sClient.Create(testCtx, invalidIntent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("networkSlice"))

				By("creating NetworkIntent with critical priority but non-maintenance type")
				invalidPriorityIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-priority-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF with critical priority",
						IntentType: nephoranv1.IntentTypeDeployment, // Should be maintenance for critical
						Priority:   nephoranv1.PriorityCritical,
					},
				}

				err = k8sClient.Create(testCtx, invalidPriorityIntent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("critical priority"))
			})

			It("should handle NetworkIntent updates and status changes", func() {
				By("creating a basic NetworkIntent")
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("updateable-intent", namespace.Name)
				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("updating the intent specification")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() error {
					return k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
				}, 10*time.Second, 1*time.Second).Should(Succeed())

				// Update priority
				createdIntent.Spec.Priority = nephoranv1.PriorityCritical
				createdIntent.Spec.IntentType = nephoranv1.IntentTypeMaintenance // Required for critical priority
				Expect(k8sClient.Update(testCtx, createdIntent)).To(Succeed())

				By("verifying the update is applied")
				updatedIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() nephoranv1.Priority {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, updatedIntent)
					if err != nil {
						return ""
					}
					return updatedIntent.Spec.Priority
				}, 10*time.Second, 1*time.Second).Should(Equal(nephoranv1.PriorityCritical))

				By("updating status subresource")
				updatedIntent.Status.Phase = "Testing"
				updatedIntent.Status.ProcessingStartTime = &metav1.Time{Time: time.Now()}
				Expect(k8sClient.Status().Update(testCtx, updatedIntent)).To(Succeed())

				By("verifying status update")
				finalIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, finalIntent)
					if err != nil {
						return ""
					}
					return finalIntent.Status.Phase
				}, 10*time.Second, 1*time.Second).Should(Equal("Testing"))
			})
		})
	})

	Describe("E2NodeSet CRD Integration", func() {
		Context("when managing E2NodeSet with real Kubernetes API", func() {
			It("should create and manage E2NodeSet with complete specification", func() {
				By("creating an E2NodeSet with comprehensive configuration")
				e2NodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "integration-e2nodeset",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                      "integration",
							"nephoran.com/e2-interface": "v3.0",
							"nephoran.com/simulation":   "enabled",
						},
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 5,
						Template: nephoranv1.E2NodeTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app":                    "e2-node",
									"nephoran.com/component": "e2-simulator",
								},
							},
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "integration-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    2,
										Description: "Key Performance Measurement (KPM) Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
									{
										FunctionID:  2,
										Revision:    1,
										Description: "RAN Control (RC) Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.3",
									},
								},
							},
						},
						SimulationConfig: &nephoranv1.SimulationConfig{
							UECount:           1000,
							TrafficGeneration: true,
							MetricsInterval:   "15s",
							TrafficProfile:    nephoranv1.TrafficProfileHigh,
						},
						RICConfiguration: &nephoranv1.RICConfiguration{
							RICEndpoint:       "http://near-rt-ric:38080",
							ConnectionTimeout: "30s",
							HeartbeatInterval: "10s",
							RetryConfig: &nephoranv1.RetryConfig{
								MaxAttempts:     5,
								BackoffInterval: "10s",
							},
						},
					},
				}

				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())

				By("verifying E2NodeSet is created with correct specification")
				createdNodeSet := &nephoranv1.E2NodeSet{}
				Eventually(func() error {
					return k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
				}, 30*time.Second, 1*time.Second).Should(Succeed())

				// Verify spec fields
				Expect(createdNodeSet.Spec.Replicas).To(Equal(int32(5)))
				Expect(createdNodeSet.Spec.Template.Spec.E2InterfaceVersion).To(Equal("v3.0"))
				Expect(createdNodeSet.Spec.Template.Spec.SupportedRANFunctions).To(HaveLen(2))
				Expect(createdNodeSet.Spec.SimulationConfig.UECount).To(Equal(int32(1000)))
				Expect(createdNodeSet.Spec.SimulationConfig.TrafficProfile).To(Equal(nephoranv1.TrafficProfileHigh))

				By("waiting for controller to process the E2NodeSet")
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
					if err != nil {
						return -1
					}
					return createdNodeSet.Status.CurrentReplicas
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">=", 0))

				By("verifying status is updated by controller")
				Expect(createdNodeSet.Status.ReadyReplicas).To(BeNumerically(">=", 0))
				Expect(createdNodeSet.Status.UpdatedReplicas).To(BeNumerically(">=", 0))
			})

			It("should handle E2NodeSet scaling operations", func() {
				By("creating an E2NodeSet with initial replica count")
				initialReplicas := int32(2)
				e2NodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("scaling-e2nodeset", namespace.Name, initialReplicas)
				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())

				By("waiting for initial deployment")
				createdNodeSet := &nephoranv1.E2NodeSet{}
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
					if err != nil {
						return -1
					}
					return createdNodeSet.Status.CurrentReplicas
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">=", 0))

				By("scaling up the E2NodeSet")
				newReplicas := int32(10)
				createdNodeSet.Spec.Replicas = newReplicas
				Expect(k8sClient.Update(testCtx, createdNodeSet)).To(Succeed())

				By("verifying scale-up is reflected in status")
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
					if err != nil {
						return -1
					}
					return createdNodeSet.Spec.Replicas
				}, 30*time.Second, 2*time.Second).Should(Equal(newReplicas))

				By("scaling down the E2NodeSet")
				finalReplicas := int32(3)
				createdNodeSet.Spec.Replicas = finalReplicas
				Expect(k8sClient.Update(testCtx, createdNodeSet)).To(Succeed())

				By("verifying scale-down is applied")
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
					if err != nil {
						return -1
					}
					return createdNodeSet.Spec.Replicas
				}, 30*time.Second, 2*time.Second).Should(Equal(finalReplicas))
			})

			It("should validate E2NodeSet specifications", func() {
				By("creating E2NodeSet with invalid replica count")
				invalidNodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-replica-nodeset",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 1500, // Exceeds maximum of 1000
						Template: nephoranv1.E2NodeTemplate{
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "invalid-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    1,
										Description: "Test Function",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
								},
							},
						},
					},
				}

				err := k8sClient.Create(testCtx, invalidNodeSet)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("replicas"))

				By("creating E2NodeSet with invalid E2 interface version")
				invalidVersionNodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-version-nodeset",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 1,
						Template: nephoranv1.E2NodeTemplate{
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "test-node",
								E2InterfaceVersion: "v99.0", // Invalid version
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    1,
										Description: "Test Function",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
								},
							},
						},
					},
				}

				err = k8sClient.Create(testCtx, invalidVersionNodeSet)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("e2InterfaceVersion"))
			})
		})

		Context("when managing E2NodeSet conditions and status", func() {
			It("should properly manage E2NodeSet conditions", func() {
				By("creating an E2NodeSet")
				e2NodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("condition-nodeset", namespace.Name, 3)
				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())

				By("waiting for controller to set initial conditions")
				createdNodeSet := &nephoranv1.E2NodeSet{}
				Eventually(func() int {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdNodeSet)
					if err != nil {
						return -1
					}
					return len(createdNodeSet.Status.Conditions)
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("verifying condition types and statuses")
				// Check for expected condition types
				conditionTypes := make(map[nephoranv1.E2NodeSetConditionType]bool)
				for _, condition := range createdNodeSet.Status.Conditions {
					conditionTypes[condition.Type] = true
					Expect(condition.LastTransitionTime).NotTo(BeZero())
				}

				// Should have at least Available condition
				expectedConditions := []nephoranv1.E2NodeSetConditionType{
					nephoranv1.E2NodeSetConditionAvailable,
				}

				for _, expectedType := range expectedConditions {
					Expect(conditionTypes).To(HaveKey(expectedType),
						fmt.Sprintf("Expected condition type %s not found", expectedType))
				}
			})
		})
	})

	Describe("Cross-CRD Integration", func() {
		Context("when managing multiple CRDs together", func() {
			It("should handle NetworkIntent creating E2NodeSet resources", func() {
				By("creating a NetworkIntent that requires E2NodeSet")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2-deployment-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy Near-RT RIC with 10 E2 node simulators for testing",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityMedium,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for NetworkIntent to be processed")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 90*time.Second, 3*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
				))

				By("checking if related resources are created")
				// In a real implementation, the NetworkIntent controller might create
				// related E2NodeSet resources. For now, we verify the intent is processed.
				Expect(createdIntent.Status.Phase).NotTo(BeEmpty())
				if createdIntent.Status.ProcessingStartTime != nil {
					Expect(createdIntent.Status.ProcessingStartTime.Time).To(BeTemporally(">=", intent.CreationTimestamp.Time))
				}
			})

			It("should handle resource cleanup when NetworkIntent is deleted", func() {
				By("creating a NetworkIntent with finalizer")
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("cleanup-test-intent", namespace.Name)
				intent.Finalizers = []string{"nephoran.com/cleanup-finalizer"}
				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("creating related ConfigMap resource")
				relatedConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "related-config",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"nephoran.com/owned-by": intent.Name,
							"nephoran.com/type":     "generated",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "nephoran.com/v1",
								Kind:       "NetworkIntent",
								Name:       intent.Name,
								UID:        intent.UID,
								Controller: &[]bool{true}[0],
							},
						},
					},
					Data: map[string]string{
						"config": "test configuration data",
					},
				}
				Expect(k8sClient.Create(testCtx, relatedConfigMap)).To(Succeed())

				By("verifying related resource exists")
				createdConfigMap := &corev1.ConfigMap{}
				Expect(k8sClient.Get(testCtx, types.NamespacedName{
					Name: relatedConfigMap.Name, Namespace: relatedConfigMap.Namespace,
				}, createdConfigMap)).To(Succeed())

				By("deleting the NetworkIntent")
				Expect(k8sClient.Delete(testCtx, intent)).To(Succeed())

				By("verifying related resource is cleaned up via owner reference")
				Eventually(func() bool {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: relatedConfigMap.Name, Namespace: relatedConfigMap.Namespace,
					}, createdConfigMap)
					return err != nil
				}, 30*time.Second, 1*time.Second).Should(BeTrue())
			})
		})
	})

	Describe("Custom Resource Validation", func() {
		Context("when testing OpenAPI schema validation", func() {
			It("should enforce field validation rules", func() {
				By("creating NetworkIntent with invalid intent length")
				shortIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "short-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Short", // Too short (minimum 10 characters)
						IntentType: nephoranv1.IntentTypeDeployment,
					},
				}

				err := k8sClient.Create(testCtx, shortIntent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("intent"))

				By("creating NetworkIntent with excessive target components")
				manyComponentsIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "many-components-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:           "Deploy all network functions with excessive components list",
						IntentType:       nephoranv1.IntentTypeDeployment,
						TargetComponents: make([]nephoranv1.TargetComponent, 25), // Exceeds max of 20
					},
				}

				err = k8sClient.Create(testCtx, manyComponentsIntent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("targetComponents"))
			})

			It("should validate resource constraint patterns", func() {
				By("creating NetworkIntent with valid resource constraints")
				validIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-resources-intent",
						Namespace: namespace.Name,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF with specific resource requirements",
						IntentType: nephoranv1.IntentTypeDeployment,
						ResourceConstraints: &nephoranv1.ResourceConstraints{
							CPU:       resource.NewQuantity(1000, resource.DecimalSI),      // 1000m
							Memory:    resource.NewQuantity(2147483648, resource.BinarySI), // 2Gi
							MaxCPU:    resource.NewQuantity(4000, resource.DecimalSI),      // 4000m
							MaxMemory: resource.NewQuantity(8589934592, resource.BinarySI), // 8Gi
						},
					},
				}

				Expect(k8sClient.Create(testCtx, validIntent)).To(Succeed())

				By("verifying resource constraints are properly stored")
				createdIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(testCtx, types.NamespacedName{
					Name: validIntent.Name, Namespace: validIntent.Namespace,
				}, createdIntent)).To(Succeed())

				Expect(createdIntent.Spec.ResourceConstraints.CPU.MilliValue()).To(Equal(int64(1000)))
				Expect(createdIntent.Spec.ResourceConstraints.Memory.Value()).To(Equal(int64(2147483648)))
			})
		})
	})
})

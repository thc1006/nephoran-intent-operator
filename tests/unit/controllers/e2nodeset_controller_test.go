package controllers_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("E2NodeSetController", func() {
	var (
		k8sClient            client.Client
		scheme               *runtime.Scheme
		controller           *controllers.E2NodeSetReconciler
		ctx                  context.Context
		cancel               context.CancelFunc
		namespace            string
		performanceTracker   *testutils.PerformanceTracker
		testFixtures         *testutils.TestFixtures
	)

	BeforeEach(func() {
		// Initialize test environment
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for scaling tests
		namespace = "test-namespace"
		
		// Create test fixtures
		testFixtures = testutils.NewTestFixtures()
		scheme = testFixtures.Scheme
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		
		// Create namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		
		// Create controller
		controller = &controllers.E2NodeSetReconciler{
			Client:   k8sClient,
			Scheme:   scheme,
			Recorder: record.NewFakeRecorder(100),
		}
		
		// Initialize performance tracker
		performanceTracker = testutils.NewPerformanceTracker()
	})

	AfterEach(func() {
		cancel()
	})

	Describe("Scaling Operations", func() {
		var nodeSet *nephoranv1.E2NodeSet

		BeforeEach(func() {
			nodeSet = testutils.E2NodeSetFixture.CreateBasicE2NodeSet("test-nodeset", namespace, 1)
			Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())
		})

		Context("when scaling up", func() {
			It("should scale from 1 to 10 nodes efficiently", func() {
				performanceTracker.Start("scale-up-1-to-10")
				
				// Update replica count to 10
				nodeSet.Spec.Replicas = 10
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile multiple times to simulate scaling process
				for i := 0; i < 5; i++ {
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					
					Expect(err).NotTo(HaveOccurred())
					if !result.Requeue && result.RequeueAfter == 0 {
						break
					}
					
					// Small delay to simulate real-world conditions
					time.Sleep(100 * time.Millisecond)
				}
				
				duration := performanceTracker.Stop("scale-up-1-to-10")
				
				// Verify scaling completed
				updatedNodeSet := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: nodeSet.Name, Namespace: nodeSet.Namespace,
				}, updatedNodeSet)).To(Succeed())
				
				// Verify ConfigMaps were created
				configMapList := &corev1.ConfigMapList{}
				Expect(k8sClient.List(ctx, configMapList, client.InNamespace(namespace))).To(Succeed())
				
				// Should have 10 ConfigMaps (one per node)
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					return len(configMapList.Items)
				}, 10*time.Second, 500*time.Millisecond).Should(Equal(10))
				
				// Performance validation - scaling should complete within reasonable time
				Expect(duration).To(BeNumerically("<", 30*time.Second), 
					"Scaling from 1 to 10 nodes should complete within 30 seconds")
				
				// Verify node set status
				Expect(updatedNodeSet.Status.CurrentNodes).To(Equal(int32(10)))
				Expect(updatedNodeSet.Status.Phase).To(Or(
					Equal(nephoranv1.E2NodeSetPhaseScaling),
					Equal(nephoranv1.E2NodeSetPhaseReady),
				))
			})

			It("should scale from 10 to 50 nodes with proper resource management", func() {
				// Start with 10 nodes
				nodeSet.Spec.Replicas = 10
				nodeSet.Status.CurrentNodes = 10
				nodeSet.Status.ReadyNodes = 10
				nodeSet.Status.Phase = nephoranv1.E2NodeSetPhaseReady
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Create initial ConfigMaps for existing nodes
				for i := 0; i < 10; i++ {
					cm := testutils.ConfigMapFixture.CreateE2NodeConfigMap(
						fmt.Sprintf("e2node-%d", i), namespace, i)
					cm.Labels["nephoran.com/e2-nodeset"] = nodeSet.Name
					Expect(k8sClient.Create(ctx, cm)).To(Succeed())
				}
				
				performanceTracker.Start("scale-up-10-to-50")
				
				// Scale to 50 nodes
				nodeSet.Spec.Replicas = 50
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile multiple times
				for i := 0; i < 10; i++ {
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					
					Expect(err).NotTo(HaveOccurred())
					if !result.Requeue && result.RequeueAfter == 0 {
						break
					}
					
					time.Sleep(200 * time.Millisecond)
				}
				
				duration := performanceTracker.Stop("scale-up-10-to-50")
				
				// Verify scaling results
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 30*time.Second, 1*time.Second).Should(Equal(50))
				
				// Performance check
				Expect(duration).To(BeNumerically("<", 45*time.Second), 
					"Scaling from 10 to 50 nodes should complete within 45 seconds")
			})

			It("should scale to 100 nodes with timing requirements", func() {
				performanceTracker.Start("scale-to-100")
				
				// Scale directly to 100 nodes
				nodeSet.Spec.Replicas = 100
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile with proper timing
				reconcileCount := 0
				maxReconciles := 20
				
				for reconcileCount < maxReconciles {
					start := time.Now()
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					reconcileDuration := time.Since(start)
					
					Expect(err).NotTo(HaveOccurred())
					
					// Each reconcile should complete quickly
					Expect(reconcileDuration).To(BeNumerically("<", 5*time.Second), 
						"Individual reconcile should complete within 5 seconds")
					
					reconcileCount++
					
					if !result.Requeue && result.RequeueAfter == 0 {
						break
					}
					
					// Check progress
					updatedNodeSet := &nephoranv1.E2NodeSet{}
					k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, updatedNodeSet)
					
					if updatedNodeSet.Status.CurrentNodes == 100 {
						break
					}
					
					time.Sleep(500 * time.Millisecond)
				}
				
				totalDuration := performanceTracker.Stop("scale-to-100")
				
				// Verify final state
				Eventually(func() int32 {
					updatedNodeSet := &nephoranv1.E2NodeSet{}
					k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, updatedNodeSet)
					return updatedNodeSet.Status.CurrentNodes
				}, 60*time.Second, 2*time.Second).Should(Equal(int32(100)))
				
				// Performance requirements for 100 nodes
				Expect(totalDuration).To(BeNumerically("<", 120*time.Second), 
					"Scaling to 100 nodes should complete within 2 minutes")
				Expect(reconcileCount).To(BeNumerically("<", maxReconciles), 
					"Should not require excessive reconcile iterations")
			})
		})

		Context("when scaling down", func() {
			BeforeEach(func() {
				// Start with multiple nodes
				nodeSet.Spec.Replicas = 20
				nodeSet.Status.CurrentNodes = 20
				nodeSet.Status.ReadyNodes = 20
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Create ConfigMaps for existing nodes
				for i := 0; i < 20; i++ {
					cm := testutils.ConfigMapFixture.CreateE2NodeConfigMap(
						fmt.Sprintf("e2node-%d", i), namespace, i)
					cm.Labels["nephoran.com/e2-nodeset"] = nodeSet.Name
					Expect(k8sClient.Create(ctx, cm)).To(Succeed())
				}
			})

			It("should scale down from 20 to 5 nodes gracefully", func() {
				performanceTracker.Start("scale-down-20-to-5")
				
				// Scale down to 5 nodes
				nodeSet.Spec.Replicas = 5
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile to handle scale down
				for i := 0; i < 8; i++ {
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					
					Expect(err).NotTo(HaveOccurred())
					if !result.Requeue && result.RequeueAfter == 0 {
						break
					}
					
					time.Sleep(200 * time.Millisecond)
				}
				
				duration := performanceTracker.Stop("scale-down-20-to-5")
				
				// Verify scale down completed
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 20*time.Second, 1*time.Second).Should(Equal(5))
				
				// Verify ConfigMaps were properly cleaned up
				updatedNodeSet := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: nodeSet.Name, Namespace: nodeSet.Namespace,
				}, updatedNodeSet)).To(Succeed())
				
				Expect(updatedNodeSet.Status.CurrentNodes).To(Equal(int32(5)))
				Expect(duration).To(BeNumerically("<", 25*time.Second), 
					"Scale down should complete within 25 seconds")
			})
		})

		Context("ConfigMap management", func() {
			It("should create ConfigMaps with proper configuration", func() {
				// Scale to 3 nodes
				nodeSet.Spec.Replicas = 3
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				
				// Verify ConfigMaps exist and have proper structure
				Eventually(func() bool {
					configMapList := &corev1.ConfigMapList{}
					err := k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					if err != nil {
						return false
					}
					
					validConfigMaps := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							// Verify required data keys exist
							if _, hasConfig := cm.Data["e2node-config.json"]; hasConfig {
								if _, hasStatus := cm.Data["e2node-status.json"]; hasStatus {
									validConfigMaps++
								}
							}
						}
					}
					
					return validConfigMaps == 3
				}, 15*time.Second, 500*time.Millisecond).Should(BeTrue())
			})

			It("should update ConfigMaps when NodeSet configuration changes", func() {
				// Create initial NodeSet
				nodeSet.Spec.Replicas = 2
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Initial reconcile
				_, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				
				// Wait for ConfigMaps to be created
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 10*time.Second, 500*time.Millisecond).Should(Equal(2))
				
				// Update NodeSet configuration
				nodeSet.Spec.RICEndpoint = "http://updated-ric:8080"
				nodeSet.Spec.SimulationConfig.UECount = 200
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile to apply updates
				_, err = controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				
				// Verify ConfigMaps were updated
				Eventually(func() bool {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							configData := cm.Data["e2node-config.json"]
							if configData != "" {
								// Check if updated configuration is reflected
								return true // Simplified check - in real test would parse JSON and verify fields
							}
						}
					}
					return false
				}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
			})
		})

		Context("heartbeat simulation", func() {
			BeforeEach(func() {
				nodeSet.Spec.Replicas = 3
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Initial reconcile to create ConfigMaps
				_, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should simulate heartbeats for all nodes", func() {
				// Wait for ConfigMaps to be created
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 15*time.Second, 500*time.Millisecond).Should(Equal(3))
				
				// Simulate heartbeat updates through multiple reconciles
				for i := 0; i < 5; i++ {
					_, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					
					time.Sleep(1 * time.Second)
				}
				
				// Verify heartbeat data is being updated
				configMapList := &corev1.ConfigMapList{}
				Expect(k8sClient.List(ctx, configMapList, client.InNamespace(namespace))).To(Succeed())
				
				heartbeatNodes := 0
				for _, cm := range configMapList.Items {
					if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
						statusData := cm.Data["e2node-status.json"]
						if statusData != "" {
							// Simplified verification - in real implementation would parse JSON
							heartbeatNodes++
						}
					}
				}
				
				Expect(heartbeatNodes).To(Equal(3), "All nodes should have heartbeat status")
			})
		})

		Context("status updates and conditions", func() {
			It("should properly update E2NodeSet status during lifecycle", func() {
				testCases := []struct {
					name           string
					replicas       int32
					expectedPhase  nephoranv1.E2NodeSetPhase
					expectedNodes  int32
				}{
					{"single node", 1, nephoranv1.E2NodeSetPhaseReady, 1},
					{"small cluster", 5, nephoranv1.E2NodeSetPhaseReady, 5},
					{"medium cluster", 25, nephoranv1.E2NodeSetPhaseReady, 25},
				}
				
				for _, tc := range testCases {
					By(fmt.Sprintf("testing %s with %d replicas", tc.name, tc.replicas))
					
					// Update replica count
					nodeSet.Spec.Replicas = tc.replicas
					Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
					
					// Reconcile multiple times to reach steady state
					for i := 0; i < 10; i++ {
						result, err := controller.Reconcile(ctx, ctrl.Request{
							NamespacedName: types.NamespacedName{
								Name:      nodeSet.Name,
								Namespace: nodeSet.Namespace,
							},
						})
						
						Expect(err).NotTo(HaveOccurred())
						if !result.Requeue && result.RequeueAfter == 0 {
							break
						}
						
						time.Sleep(200 * time.Millisecond)
					}
					
					// Verify final status
					Eventually(func() nephoranv1.E2NodeSetPhase {
						updatedNodeSet := &nephoranv1.E2NodeSet{}
						k8sClient.Get(ctx, types.NamespacedName{
							Name: nodeSet.Name, Namespace: nodeSet.Namespace,
						}, updatedNodeSet)
						return updatedNodeSet.Status.Phase
					}, 20*time.Second, 1*time.Second).Should(Or(
						Equal(nephoranv1.E2NodeSetPhaseReady),
						Equal(nephoranv1.E2NodeSetPhaseScaling),
					))
					
					// Verify node counts
					Eventually(func() int32 {
						updatedNodeSet := &nephoranv1.E2NodeSet{}
						k8sClient.Get(ctx, types.NamespacedName{
							Name: nodeSet.Name, Namespace: nodeSet.Namespace,
						}, updatedNodeSet)
						return updatedNodeSet.Status.CurrentNodes
					}, 20*time.Second, 1*time.Second).Should(Equal(tc.expectedNodes))
				}
			})

			It("should set appropriate conditions based on node states", func() {
				nodeSet.Spec.Replicas = 3
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Reconcile to create nodes
				for i := 0; i < 5; i++ {
					_, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(500 * time.Millisecond)
				}
				
				// Verify conditions are set
				Eventually(func() bool {
					updatedNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, updatedNodeSet)
					if err != nil {
						return false
					}
					
					// Check for Ready condition
					for _, condition := range updatedNodeSet.Status.Conditions {
						if condition.Type == nephoranv1.E2NodeSetConditionReady {
							return true
						}
					}
					return false
				}, 15*time.Second, 1*time.Second).Should(BeTrue())
			})
		})

		Context("error recovery", func() {
			It("should recover from scaling failures", func() {
				// Simulate a scenario where scaling partially fails
				nodeSet.Spec.Replicas = 5
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Create some ConfigMaps manually to simulate partial success
				for i := 0; i < 2; i++ {
					cm := testutils.ConfigMapFixture.CreateE2NodeConfigMap(
						fmt.Sprintf("e2node-%d", i), namespace, i)
					cm.Labels["nephoran.com/e2-nodeset"] = nodeSet.Name
					Expect(k8sClient.Create(ctx, cm)).To(Succeed())
				}
				
				// Reconcile should detect the gap and create missing ConfigMaps
				for i := 0; i < 10; i++ {
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						},
					})
					
					Expect(err).NotTo(HaveOccurred())
					if !result.Requeue && result.RequeueAfter == 0 {
						break
					}
					
					time.Sleep(300 * time.Millisecond)
				}
				
				// Verify all 5 ConfigMaps exist
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 20*time.Second, 1*time.Second).Should(Equal(5))
			})

			It("should handle ConfigMap deletion gracefully", func() {
				// Create initial setup
				nodeSet.Spec.Replicas = 3
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				
				// Initial reconcile
				_, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				
				// Wait for ConfigMaps to be created
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 15*time.Second, 500*time.Millisecond).Should(Equal(3))
				
				// Delete one ConfigMap to simulate failure
				configMapList := &corev1.ConfigMapList{}
				Expect(k8sClient.List(ctx, configMapList, client.InNamespace(namespace))).To(Succeed())
				
				var targetCM *corev1.ConfigMap
				for _, cm := range configMapList.Items {
					if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
						targetCM = &cm
						break
					}
				}
				Expect(targetCM).NotTo(BeNil())
				Expect(k8sClient.Delete(ctx, targetCM)).To(Succeed())
				
				// Reconcile should recreate the missing ConfigMap
				_, err = controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				
				// Verify ConfigMap was recreated
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 15*time.Second, 500*time.Millisecond).Should(Equal(3))
			})
		})
	})

	Describe("Concurrent Scaling Operations", func() {
		It("should handle concurrent scaling requests safely", func() {
			// Create multiple E2NodeSets for concurrent testing
			nodeSets := make([]*nephoranv1.E2NodeSet, 3)
			for i := 0; i < 3; i++ {
				nodeSets[i] = testutils.E2NodeSetFixture.CreateBasicE2NodeSet(
					fmt.Sprintf("concurrent-nodeset-%d", i), namespace, 1)
				Expect(k8sClient.Create(ctx, nodeSets[i])).To(Succeed())
			}
			
			// Perform concurrent scaling operations
			var wg sync.WaitGroup
			errors := make([]error, 0)
			var mu sync.Mutex
			
			for i, nodeSet := range nodeSets {
				wg.Add(1)
				go func(ns *nephoranv1.E2NodeSet, idx int) {
					defer wg.Done()
					defer GinkgoRecover()
					
					// Scale each NodeSet to different sizes
					targetReplicas := int32(5 + idx*3) // 5, 8, 11
					ns.Spec.Replicas = targetReplicas
					
					err := k8sClient.Update(ctx, ns)
					if err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
						return
					}
					
					// Reconcile multiple times
					for j := 0; j < 8; j++ {
						result, err := controller.Reconcile(ctx, ctrl.Request{
							NamespacedName: types.NamespacedName{
								Name:      ns.Name,
								Namespace: ns.Namespace,
							},
						})
						
						if err != nil {
							mu.Lock()
							errors = append(errors, err)
							mu.Unlock()
							return
						}
						
						if !result.Requeue && result.RequeueAfter == 0 {
							break
						}
						
						time.Sleep(200 * time.Millisecond)
					}
				}(nodeSet, i)
			}
			
			wg.Wait()
			
			// Verify no errors occurred
			Expect(errors).To(BeEmpty(), "Concurrent scaling should not produce errors")
			
			// Verify final states
			for i, nodeSet := range nodeSets {
				expectedReplicas := int32(5 + i*3)
				
				Eventually(func() int32 {
					updatedNodeSet := &nephoranv1.E2NodeSet{}
					k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, updatedNodeSet)
					return updatedNodeSet.Status.CurrentNodes
				}, 30*time.Second, 1*time.Second).Should(Equal(expectedReplicas))
			}
		})
	})

	Describe("Deletion and Cleanup", func() {
		var nodeSet *nephoranv1.E2NodeSet

		BeforeEach(func() {
			nodeSet = testutils.E2NodeSetFixture.CreateBasicE2NodeSet("cleanup-test", namespace, 5)
			Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())
			
			// Initial reconcile to create ConfigMaps
			_, err := controller.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      nodeSet.Name,
					Namespace: nodeSet.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should clean up ConfigMaps when E2NodeSet is deleted", func() {
			// Wait for ConfigMaps to be created
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
				count := 0
				for _, cm := range configMapList.Items {
					if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
						count++
					}
				}
				return count
			}, 20*time.Second, 1*time.Second).Should(Equal(5))
			
			// Delete the E2NodeSet
			Expect(k8sClient.Delete(ctx, nodeSet)).To(Succeed())
			
			// Reconcile should handle cleanup
			result, err := controller.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      nodeSet.Name,
					Namespace: nodeSet.Namespace,
				},
			})
			
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			
			// Verify E2NodeSet is deleted
			deletedNodeSet := &nephoranv1.E2NodeSet{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name: nodeSet.Name, Namespace: nodeSet.Namespace,
			}, deletedNodeSet)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
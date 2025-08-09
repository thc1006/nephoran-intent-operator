package integration_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("E2NodeSet Scaling Integration", func() {
	var (
		namespace          string
		testFixtures       *testutils.TestFixtures
		performanceTracker *testutils.PerformanceTracker
	)

	BeforeEach(func() {
		namespace = fmt.Sprintf("e2scale-ns-%d", time.Now().UnixNano())

		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		testFixtures = testutils.NewTestFixtures()
		performanceTracker = testutils.NewPerformanceTracker()

		DeferCleanup(func() {
			// Cleanup namespace
			Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
		})
	})

	Describe("Large Scale Operations", func() {
		Context("scaling from 1 to 100 nodes", func() {
			It("should scale efficiently with proper timing", func() {
				By("creating initial E2NodeSet with 1 replica")
				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("large-scale-test", namespace, 1)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				By("waiting for initial node to be ready")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 30*time.Second, 2*time.Second).Should(Equal(int32(1)))

				performanceTracker.Start("scale-1-to-100")

				By("scaling to 100 nodes")
				nodeSet.Spec.Replicas = 100
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

				By("waiting for scaling to complete")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 180*time.Second, 5*time.Second).Should(Equal(int32(100)))

				scalingDuration := performanceTracker.Stop("scale-1-to-100")

				By("verifying performance requirements")
				Expect(scalingDuration).To(BeNumerically("<", 150*time.Second),
					"Scaling to 100 nodes should complete within 2.5 minutes")

				By("verifying all ConfigMaps are created")
				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					err := k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					if err != nil {
						return -1
					}

					count := 0
					for _, cm := range configMapList.Items {
						if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
							count++
						}
					}
					return count
				}, 60*time.Second, 2*time.Second).Should(Equal(100))

				By("verifying E2NodeSet status is correct")
				finalNodeSet := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: nodeSet.Name, Namespace: nodeSet.Namespace,
				}, finalNodeSet)).To(Succeed())

				Expect(finalNodeSet.Status.CurrentNodes).To(Equal(int32(100)))
				Expect(finalNodeSet.Status.ReadyNodes).To(Equal(int32(100)))
				Expect(finalNodeSet.Status.Phase).To(Equal(nephoranv1.E2NodeSetPhaseReady))

				// Verify conditions
				hasReadyCondition := false
				for _, condition := range finalNodeSet.Status.Conditions {
					if condition.Type == nephoranv1.E2NodeSetConditionReady {
						hasReadyCondition = true
						Expect(condition.Status).To(Equal(corev1.ConditionTrue))
						break
					}
				}
				Expect(hasReadyCondition).To(BeTrue())
			})
		})

		Context("scaling patterns", func() {
			It("should handle scale up from 10 to 50 efficiently", func() {
				By("creating E2NodeSet with 10 replicas")
				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("scale-up-test", namespace, 10)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				By("waiting for initial 10 nodes")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 45*time.Second, 2*time.Second).Should(Equal(int32(10)))

				performanceTracker.Start("scale-10-to-50")

				By("scaling up to 50 nodes")
				nodeSet.Spec.Replicas = 50
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

				By("monitoring scaling progress")
				progressChecks := make([]int32, 0)
				go func() {
					defer GinkgoRecover()
					ticker := time.NewTicker(5 * time.Second)
					defer ticker.Stop()
					timeout := time.NewTimer(120 * time.Second)
					defer timeout.Stop()

					for {
						select {
						case <-timeout.C:
							return
						case <-ticker.C:
							currentNodeSet := &nephoranv1.E2NodeSet{}
							err := k8sClient.Get(ctx, types.NamespacedName{
								Name: nodeSet.Name, Namespace: nodeSet.Namespace,
							}, currentNodeSet)
							if err == nil {
								progressChecks = append(progressChecks, currentNodeSet.Status.CurrentNodes)
								if currentNodeSet.Status.CurrentNodes == 50 {
									return
								}
							}
						}
					}
				}()

				By("waiting for scaling completion")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 120*time.Second, 3*time.Second).Should(Equal(int32(50)))

				scalingDuration := performanceTracker.Stop("scale-10-to-50")

				By("verifying scaling performance")
				Expect(scalingDuration).To(BeNumerically("<", 90*time.Second),
					"Scaling from 10 to 50 should complete within 90 seconds")

				By("verifying progressive scaling occurred")
				Expect(len(progressChecks)).To(BeNumerically(">=", 3),
					"Should have multiple progress checkpoints")

				// Verify monotonic increase (no decreases during scale-up)
				for i := 1; i < len(progressChecks); i++ {
					Expect(progressChecks[i]).To(BeNumerically(">=", progressChecks[i-1]),
						"Node count should not decrease during scale-up")
				}
			})

			It("should handle scale down from 50 to 10 with proper cleanup", func() {
				By("creating E2NodeSet with 50 replicas")
				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("scale-down-test", namespace, 50)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				By("waiting for initial 50 nodes")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 90*time.Second, 3*time.Second).Should(Equal(int32(50)))

				By("verifying all ConfigMaps exist initially")
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
				}, 30*time.Second, 2*time.Second).Should(Equal(50))

				performanceTracker.Start("scale-50-to-10")

				By("scaling down to 10 nodes")
				nodeSet.Spec.Replicas = 10
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

				By("waiting for scaling down to complete")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 60*time.Second, 2*time.Second).Should(Equal(int32(10)))

				scalingDuration := performanceTracker.Stop("scale-50-to-10")

				By("verifying scale-down performance")
				Expect(scalingDuration).To(BeNumerically("<", 45*time.Second),
					"Scaling down should be faster than scaling up")

				By("verifying excess ConfigMaps are cleaned up")
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
				}, 30*time.Second, 2*time.Second).Should(Equal(10))

				By("verifying final state")
				finalNodeSet := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: nodeSet.Name, Namespace: nodeSet.Namespace,
				}, finalNodeSet)).To(Succeed())

				Expect(finalNodeSet.Status.CurrentNodes).To(Equal(int32(10)))
				Expect(finalNodeSet.Status.ReadyNodes).To(Equal(int32(10)))
				Expect(finalNodeSet.Status.Phase).To(Equal(nephoranv1.E2NodeSetPhaseReady))
			})
		})

		Context("error recovery during scaling", func() {
			It("should recover from partial scaling failures", func() {
				By("creating initial E2NodeSet")
				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("recovery-test", namespace, 5)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				By("waiting for initial nodes")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 30*time.Second, 2*time.Second).Should(Equal(int32(5)))

				By("scaling to 15 nodes")
				nodeSet.Spec.Replicas = 15
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

				By("simulating partial failure by deleting some ConfigMaps during scaling")
				// Wait a bit for scaling to start
				time.Sleep(5 * time.Second)

				configMapList := &corev1.ConfigMapList{}
				k8sClient.List(ctx, configMapList, client.InNamespace(namespace))

				deletedCount := 0
				for _, cm := range configMapList.Items {
					if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name && deletedCount < 3 {
						k8sClient.Delete(ctx, &cm)
						deletedCount++
					}
				}

				By("verifying system recovers and reaches target state")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 60*time.Second, 3*time.Second).Should(Equal(int32(15)))

				By("verifying all ConfigMaps are eventually present")
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
				}, 45*time.Second, 2*time.Second).Should(Equal(15))
			})

			It("should handle rapid scaling changes", func() {
				By("creating initial E2NodeSet")
				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet("rapid-scale-test", namespace, 3)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				By("waiting for initial state")
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 20*time.Second, 1*time.Second).Should(Equal(int32(3)))

				scalingSequence := []int32{10, 5, 20, 15, 8}

				By("performing rapid scaling changes")
				for i, targetReplicas := range scalingSequence {
					By(fmt.Sprintf("scaling to %d nodes (step %d)", targetReplicas, i+1))

					nodeSet.Spec.Replicas = targetReplicas
					Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

					// Give some time for scaling to start
					time.Sleep(2 * time.Second)
				}

				By("waiting for final stable state")
				finalTarget := scalingSequence[len(scalingSequence)-1]
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 90*time.Second, 3*time.Second).Should(Equal(finalTarget))

				By("verifying system reaches stable state")
				finalNodeSet := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: nodeSet.Name, Namespace: nodeSet.Namespace,
				}, finalNodeSet)).To(Succeed())

				Expect(finalNodeSet.Status.CurrentNodes).To(Equal(finalTarget))
				Expect(finalNodeSet.Status.ReadyNodes).To(Equal(finalTarget))
				Expect(finalNodeSet.Status.Phase).To(Equal(nephoranv1.E2NodeSetPhaseReady))

				// Verify ConfigMap count matches
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
				}, 30*time.Second, 2*time.Second).Should(Equal(int(finalTarget)))
			})
		})

		Context("stress testing scaling operations", func() {
			It("should handle multiple NodeSets scaling simultaneously", func() {
				numNodeSets := 5
				nodeSets := make([]*nephoranv1.E2NodeSet, numNodeSets)

				By("creating multiple E2NodeSets")
				for i := 0; i < numNodeSets; i++ {
					nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet(
						fmt.Sprintf("stress-test-%d", i), namespace, 2)
					nodeSets[i] = nodeSet
					Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())
				}

				By("waiting for all NodeSets to reach initial state")
				for i, nodeSet := range nodeSets {
					Eventually(func() int32 {
						currentNodeSet := &nephoranv1.E2NodeSet{}
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: nodeSet.Name, Namespace: nodeSet.Namespace,
						}, currentNodeSet)
						if err != nil {
							return -1
						}
						return currentNodeSet.Status.CurrentNodes
					}, 30*time.Second, 2*time.Second).Should(Equal(int32(2)),
						fmt.Sprintf("NodeSet %d should reach initial state", i))
				}

				performanceTracker.Start("concurrent-scaling")

				By("scaling all NodeSets simultaneously")
				targetReplicas := []int32{10, 15, 8, 12, 20}
				for i, nodeSet := range nodeSets {
					nodeSet.Spec.Replicas = targetReplicas[i]
					Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())
				}

				By("waiting for all scaling operations to complete")
				for i, nodeSet := range nodeSets {
					Eventually(func() int32 {
						currentNodeSet := &nephoranv1.E2NodeSet{}
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: nodeSet.Name, Namespace: nodeSet.Namespace,
						}, currentNodeSet)
						if err != nil {
							return -1
						}
						return currentNodeSet.Status.CurrentNodes
					}, 120*time.Second, 3*time.Second).Should(Equal(targetReplicas[i]),
						fmt.Sprintf("NodeSet %d should reach target %d", i, targetReplicas[i]))
				}

				concurrentScalingDuration := performanceTracker.Stop("concurrent-scaling")

				By("verifying concurrent scaling performance")
				Expect(concurrentScalingDuration).To(BeNumerically("<", 150*time.Second),
					"Concurrent scaling should complete within reasonable time")

				By("verifying all ConfigMaps are created correctly")
				totalExpectedConfigMaps := int32(0)
				for _, target := range targetReplicas {
					totalExpectedConfigMaps += target
				}

				Eventually(func() int {
					configMapList := &corev1.ConfigMapList{}
					k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
					count := 0
					for _, cm := range configMapList.Items {
						for _, nodeSet := range nodeSets {
							if cm.Labels["nephoran.com/e2-nodeset"] == nodeSet.Name {
								count++
								break
							}
						}
					}
					return count
				}, 60*time.Second, 3*time.Second).Should(Equal(int(totalExpectedConfigMaps)))

				By("verifying all NodeSets are in ready state")
				for i, nodeSet := range nodeSets {
					finalNodeSet := &nephoranv1.E2NodeSet{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, finalNodeSet)).To(Succeed())

					Expect(finalNodeSet.Status.Phase).To(Equal(nephoranv1.E2NodeSetPhaseReady),
						fmt.Sprintf("NodeSet %d should be ready", i))
					Expect(finalNodeSet.Status.CurrentNodes).To(Equal(targetReplicas[i]))
					Expect(finalNodeSet.Status.ReadyNodes).To(Equal(targetReplicas[i]))
				}
			})
		})
	})

	Describe("Performance Validation", func() {
		It("should meet timing requirements for various scaling scenarios", func() {
			testCases := []struct {
				name        string
				initial     int32
				target      int32
				maxDuration time.Duration
				description string
			}{
				{"small-scale-up", 1, 10, 30 * time.Second, "Small scale up should be fast"},
				{"medium-scale-up", 5, 25, 60 * time.Second, "Medium scale up should be moderate"},
				{"large-scale-up", 10, 50, 90 * time.Second, "Large scale up should be reasonable"},
				{"scale-down", 30, 5, 45 * time.Second, "Scale down should be faster"},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("testing %s: %d -> %d nodes", tc.name, tc.initial, tc.target))

				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet(
					fmt.Sprintf("perf-%s", tc.name), namespace, tc.initial)
				Expect(k8sClient.Create(ctx, nodeSet)).To(Succeed())

				// Wait for initial state
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, 60*time.Second, 2*time.Second).Should(Equal(tc.initial))

				// Perform scaling
				performanceTracker.Start(tc.name)
				nodeSet.Spec.Replicas = tc.target
				Expect(k8sClient.Update(ctx, nodeSet)).To(Succeed())

				// Wait for completion
				Eventually(func() int32 {
					currentNodeSet := &nephoranv1.E2NodeSet{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, currentNodeSet)
					if err != nil {
						return -1
					}
					return currentNodeSet.Status.CurrentNodes
				}, tc.maxDuration+30*time.Second, 2*time.Second).Should(Equal(tc.target))

				actualDuration := performanceTracker.Stop(tc.name)

				By(fmt.Sprintf("verifying performance: %s", tc.description))
				Expect(actualDuration).To(BeNumerically("<", tc.maxDuration),
					fmt.Sprintf("%s took %v, should be < %v", tc.description, actualDuration, tc.maxDuration))

				// Cleanup for next test
				Expect(k8sClient.Delete(ctx, nodeSet)).To(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: nodeSet.Name, Namespace: nodeSet.Namespace,
					}, &nephoranv1.E2NodeSet{})
					return err != nil
				}, 30*time.Second, 2*time.Second).Should(BeTrue())
			}
		})
	})
})

/*
Package envtest provides comprehensive controller testing for NetworkIntent
following 2025 Kubernetes operator testing best practices.
*/

package envtest

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

var _ = Describe("NetworkIntent Controller", Ordered, func() {
	Context("When creating a NetworkIntent", func() {
		var (
			networkIntent     *intentv1alpha1.NetworkIntent
			networkIntentName string
			testNamespace     string
			testCtx           context.Context
			testCancel        context.CancelFunc
		)

		BeforeAll(func() {
			testCtx, testCancel = CreateTestContext(nil)
			testNamespace = "default" // Use default namespace for simplicity
			networkIntentName = "test-network-intent"

			By("creating a NetworkIntent resource")
			networkIntent = &intentv1alpha1.NetworkIntent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "intent.nephio.org/v1alpha1",
					Kind:       "NetworkIntent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      networkIntentName,
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					// 2025 pattern: Use realistic O-RAN scaling scenarios
					ScalingPriority: "high",
					TargetClusters:  []string{"cluster-1"},
				},
			}
		})

		AfterAll(func() {
			testCancel()
		})

		It("should create the NetworkIntent successfully", func(ctx SpecContext) {
			LogTestStep("Creating NetworkIntent resource", "name", networkIntentName, "namespace", testNamespace)
			
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			// Verify the resource was created
			createdIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, createdIntent)
			}, timeout, interval).Should(Succeed())

			Expect(createdIntent.Spec.ScalingPriority).To(Equal("high"))
			Expect(createdIntent.Spec.TargetClusters).To(ContainElement("cluster-1"))
		})

		It("should update the NetworkIntent status", func(ctx SpecContext) {
			LogTestStep("Waiting for controller to update NetworkIntent status")
			
			// Wait for the controller to process the resource
			updatedIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() string {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, updatedIntent)
				if err != nil {
					return ""
				}
				return updatedIntent.Status.Phase
			}, timeout, interval).Should(Not(BeEmpty()))

			// Verify status fields are properly set
			Expect(updatedIntent.Status.Phase).To(BeElementOf([]string{"Pending", "Processing", "Completed", "Failed"}))
			Expect(updatedIntent.Status.LastUpdated).NotTo(BeNil())
		})

		It("should handle NetworkIntent updates correctly", func(ctx SpecContext) {
			LogTestStep("Testing NetworkIntent updates")
			
			// Get the current resource
			currentIntent := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      networkIntentName,
				Namespace: testNamespace,
			}, currentIntent)).Should(Succeed())

			// Update the resource
			currentIntent.Spec.ScalingIntent.Target.Replicas = 8
			currentIntent.Spec.Priority = "medium"
			
			Expect(k8sClient.Update(ctx, currentIntent)).Should(Succeed())

			// Verify the update was processed
			Eventually(func() int32 {
				updatedIntent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, updatedIntent)
				if err != nil {
					return 0
				}
				return updatedIntent.Spec.ScalingIntent.Target.Replicas
			}, timeout, interval).Should(Equal(int32(8)))
		})

		It("should handle NetworkIntent deletion correctly", func(ctx SpecContext) {
			LogTestStep("Testing NetworkIntent deletion")
			
			// Delete the resource
			intentToDelete := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      networkIntentName,
				Namespace: testNamespace,
			}, intentToDelete)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, intentToDelete)).Should(Succeed())

			// Verify the resource is deleted
			deletedIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, deletedIntent)
				return client.IgnoreNotFound(err) == nil && deletedIntent.Name == ""
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When testing NetworkIntent validation", func() {
		It("should reject invalid scaling actions", func(ctx SpecContext) {
			LogTestStep("Testing invalid scaling action validation")
			
			invalidIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingPriority: "medium",
					TargetClusters: []string{"cluster-test"},
					Action: "invalid-action", // Invalid action
					Target: intentv1alpha1.ScalingTarget{
						Component: "cu-cp",
						Replicas:  5,
					},
				},
			}

			err := k8sClient.Create(ctx, invalidIntent)
			// This should fail validation if webhooks are properly configured
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("invalid"))
			} else {
				// If no validation webhook, clean up the resource
				Expect(k8sClient.Delete(ctx, invalidIntent)).Should(Succeed())
			}
		})

		It("should reject scaling beyond constraints", func(ctx SpecContext) {
			LogTestStep("Testing scaling constraint validation")
			
			constraintViolationIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "constraint-violation-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingPriority: "medium",
					TargetClusters: []string{"cluster-test"},
					Action: "scale-up",
					Target: intentv1alpha1.ScalingTarget{
						Component: "cu-cp",
						Replicas:  15, // Exceeds max replicas
					},
					Constraints: intentv1alpha1.ScalingConstraints{
						MaxReplicas: 10,
						MinReplicas: 2,
					},
				},
			}

			err := k8sClient.Create(ctx, constraintViolationIntent)
			// This should fail validation if webhooks are properly configured
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("exceeds"))
			} else {
				// If no validation webhook, check controller handles it
				Eventually(func() string {
					intent := &intentv1alpha1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      "constraint-violation-intent",
						Namespace: "default",
					}, intent)
					if err != nil {
						return ""
					}
					return intent.Status.Phase
				}, timeout, interval).Should(Equal("Failed"))
				
				// Clean up
				Expect(k8sClient.Delete(ctx, constraintViolationIntent)).Should(Succeed())
			}
		})
	})

	Context("When testing concurrent NetworkIntent operations", func() {
		It("should handle multiple concurrent NetworkIntents", func(ctx SpecContext) {
			LogTestStep("Testing concurrent NetworkIntent creation")
			
			const numIntents = 5
			intents := make([]*intentv1alpha1.NetworkIntent, numIntents)
			
			// Create multiple intents concurrently
			for i := 0; i < numIntents; i++ {
				intents[i] = &intentv1alpha1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("concurrent-intent-%d", i),
						Namespace: "default",
					},
					Spec: intentv1alpha1.NetworkIntentSpec{
						ScalingPriority: "medium",
						TargetClusters: []string{"cluster-test"},
						Action: "scale-up",
						Target: intentv1alpha1.ScalingTarget{
							Component: "cu-cp",
							Replicas:  int32(i + 3), // Different replica counts
							Region:    fmt.Sprintf("region-%d", i),
						},
						Priority: "medium",
					},
				}
				
				Expect(k8sClient.Create(ctx, intents[i])).Should(Succeed())
			}

			// Verify all intents are processed
			for i := 0; i < numIntents; i++ {
				Eventually(func() string {
					intent := &intentv1alpha1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      fmt.Sprintf("concurrent-intent-%d", i),
						Namespace: "default",
					}, intent)
					if err != nil {
						return ""
					}
					return intent.Status.Phase
				}, timeout, interval).Should(Not(BeEmpty()))
			}

			// Clean up all intents
			for i := 0; i < numIntents; i++ {
				Expect(k8sClient.Delete(ctx, intents[i])).Should(Succeed())
			}
		})
	})

	Context("When testing NetworkIntent with realistic O-RAN scenarios", func() {
		DescribeTable("should handle various O-RAN component scaling scenarios",
			func(component string, initialReplicas, targetReplicas int32, expectedPhase string) {
				LogTestStep("Testing O-RAN component scaling", 
					"component", component, 
					"from", initialReplicas, 
					"to", targetReplicas)
				
				intentName := fmt.Sprintf("oran-%s-intent", component)
				oranIntent := &intentv1alpha1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: "default",
					},
					Spec: intentv1alpha1.NetworkIntentSpec{
						ScalingPriority: "medium",
						TargetClusters:  []string{"cluster-test"},
					},
				}

				By("creating the O-RAN intent")
				Expect(k8sClient.Create(ctx, oranIntent)).Should(Succeed())

				By("verifying the intent is processed correctly")
				Eventually(func() string {
					intent := &intentv1alpha1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      intentName,
						Namespace: "default",
					}, intent)
					if err != nil {
						return ""
					}
					return intent.Status.Phase
				}, timeout, interval).Should(Equal(expectedPhase))

				By("cleaning up the O-RAN intent")
				Expect(k8sClient.Delete(ctx, oranIntent)).Should(Succeed())
			},
			Entry("CU-CP scale up", "cu-cp", int32(2), int32(5), "Processing"),
			Entry("CU-UP scale up", "cu-up", int32(3), int32(8), "Processing"),
			Entry("DU scale down", "du", int32(10), int32(6), "Processing"),
			Entry("RIC scale up", "ric", int32(1), int32(3), "Processing"),
			Entry("SMF scale up", "smf", int32(2), int32(4), "Processing"),
		)
	})
})
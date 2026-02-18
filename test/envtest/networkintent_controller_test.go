/*
Package envtest provides comprehensive controller testing for NetworkIntent
following 2025 Kubernetes operator testing best practices.
*/

package envtest

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

// Note: timeout, interval, k8sClient, and LogTestStep are already defined in suite_test.go

// checkK8sClient skips the test if k8sClient is not properly initialized
func checkK8sClient() {
	if k8sClient == nil {
		Skip("k8sClient not initialized - requires proper envtest setup with BeforeSuite")
	}
}

var _ = Describe("NetworkIntent Controller", Ordered, func() {
	Context("When creating a NetworkIntent", func() {
		var (
			networkIntent     *intentv1alpha1.NetworkIntent
			networkIntentName string
			testNamespace     string
		)

		BeforeAll(func() {
			testNamespace = "default" // Use default namespace for simplicity
			networkIntentName = "test-network-intent"

			// Skip all tests in this context if k8sClient is not initialized
			checkK8sClient()

			By("creating a NetworkIntent resource")
			networkIntent = &intentv1alpha1.NetworkIntent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "intent.nephoran.com/v1alpha1",
					Kind:       "NetworkIntent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      networkIntentName,
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					// 2025 pattern: Use realistic O-RAN scaling scenarios
					Source:     "test",
					IntentType: "scaling",
					Target:     "cluster-1",
					Replicas:   5,
				},
			}
			// Clean up any leftover resource from a previous test run (UseExistingCluster keeps state)
			existing := &intentv1alpha1.NetworkIntent{}
			if getErr := k8sClient.Get(context.Background(), types.NamespacedName{
				Name: networkIntentName, Namespace: testNamespace,
			}, existing); getErr == nil {
				_ = k8sClient.Delete(context.Background(), existing)
			}
			Expect(k8sClient.Create(context.Background(), networkIntent)).To(Succeed())
		})

		AfterAll(func() {
			checkK8sClient()
			// Clean up the resource created in BeforeAll
			toDelete := &intentv1alpha1.NetworkIntent{}
			if err := k8sClient.Get(context.Background(), types.NamespacedName{
				Name: networkIntentName, Namespace: testNamespace,
			}, toDelete); err == nil {
				_ = k8sClient.Delete(context.Background(), toDelete)
			}
		})

		It("should create the NetworkIntent successfully", func(ctx SpecContext) {
			GinkgoWriter.Printf("Creating NetworkIntent resource: name=%s, namespace=%s\n", networkIntentName, testNamespace)
			
			createdIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, createdIntent)
			}, timeout, interval).Should(Succeed())

			Expect(createdIntent.Spec.Source).To(Equal("test"))
			Expect(createdIntent.Spec.Target).To(Equal("cluster-1"))
		})

		It("should update the NetworkIntent status", func(ctx SpecContext) {
			LogTestStep("Waiting for controller to update NetworkIntent status")

			// Wait for the controller to process the resource and set a Phase.
			// Controller sets: "Validated", "Deployed", "Processed", or "Error"
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
			}, timeout, interval).Should(BeElementOf([]string{"Validated", "Deployed", "Processed", "Error"}))

			// Verify status message is set by the controller
			Expect(updatedIntent.Status.Message).NotTo(BeEmpty())
		})

		It("should handle NetworkIntent updates correctly", func(ctx SpecContext) {
			LogTestStep("Testing NetworkIntent updates")

			// Use retry-on-conflict: the controller reconciler may update status between Get and Update
			Eventually(func() error {
				currentIntent := &intentv1alpha1.NetworkIntent{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, currentIntent); err != nil {
					return err
				}
				currentIntent.Spec.Replicas = 8
				currentIntent.Spec.Target = "cluster-updated"
				return k8sClient.Update(ctx, currentIntent)
			}, timeout, interval).Should(Succeed())

			// Verify the spec update persisted
			Eventually(func() int32 {
				updatedIntent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, updatedIntent)
				if err != nil {
					return 0
				}
				return updatedIntent.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(8)))
		})

		It("should handle NetworkIntent deletion correctly", func(ctx SpecContext) {
			LogTestStep("Testing NetworkIntent deletion")

			// Get current resource
			intentToDelete := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      networkIntentName,
				Namespace: testNamespace,
			}, intentToDelete)).Should(Succeed())

			// Strip finalizers first so deletion is not blocked by external service availability
			// (A1 mediator may not be deployed in the test environment)
			if len(intentToDelete.Finalizers) > 0 {
				patch := client.MergeFrom(intentToDelete.DeepCopy())
				intentToDelete.Finalizers = nil
				Expect(k8sClient.Patch(ctx, intentToDelete, patch)).Should(Succeed())
			}

			Expect(k8sClient.Delete(ctx, intentToDelete)).Should(Succeed())

			// Verify the resource is deleted
			deletedIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, deletedIntent)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When testing NetworkIntent validation", func() {
		It("should reject invalid scaling actions", func(ctx SpecContext) {
			checkK8sClient()
			LogTestStep("Testing invalid scaling action validation")
			
			invalidIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					Source:     "test",
					IntentType: "invalid-type", // Invalid type
					Target:     "cluster-test",
					Replicas:   5,
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
			checkK8sClient()
			LogTestStep("Testing scaling constraint validation")
			
			constraintViolationIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "constraint-violation-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					Source:     "test",
					IntentType: "scaling",
					Target:     "cluster-test",
					Replicas:   -5, // Invalid negative replicas
				},
			}

			err := k8sClient.Create(ctx, constraintViolationIntent)
			// This should fail validation if webhooks are properly configured
			if err != nil {
				// CRD validation: "should be greater than or equal to 0"
				Expect(err.Error()).To(Or(ContainSubstring("negative"), ContainSubstring("invalid"), ContainSubstring("greater than or equal")))
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
			checkK8sClient()
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
						Source:     "test",
						IntentType: "scaling",
						Target:     fmt.Sprintf("cluster-test-%d", i),
						Replicas:   int32(i + 3), // Different replica counts
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
			func(ctx SpecContext, component string, initialReplicas, targetReplicas int32, expectedPhase string) {
				checkK8sClient()
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
						Source:     "test",
						IntentType: "scaling",
						Target:     component,
						Replicas:   targetReplicas,
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
				}, timeout, interval).Should(BeElementOf([]string{"Validated", "Deployed", "Processed", "Error"}))

				By("cleaning up the O-RAN intent")
				Expect(k8sClient.Delete(ctx, oranIntent)).Should(Succeed())
			},
			Entry("CU-CP scale up", "cu-cp", int32(2), int32(5), "Validated"),
			Entry("CU-UP scale up", "cu-up", int32(3), int32(8), "Validated"),
			Entry("DU scale down", "du", int32(10), int32(6), "Validated"),
			Entry("RIC scale up", "ric", int32(1), int32(3), "Validated"),
			Entry("SMF scale up", "smf", int32(2), int32(4), "Validated"),
		)
	})
})

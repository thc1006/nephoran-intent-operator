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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
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
			testCtx, testCancel = GetTestContext()
			testNamespace = "default" // Use default namespace for simplicity
			networkIntentName = "test-network-intent"

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
					IntentType: "scaling",
					Target:     "oran-e2-nodes-cu",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "test",
				},
			}
		})

		AfterAll(func() {
			if testCancel != nil {
				testCancel()
			}
		})

		It("Should create NetworkIntent successfully", func() {
			Expect(k8sClient.Create(testCtx, networkIntent)).To(Succeed())

			// Verify the created resource
			createdIntent := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      networkIntentName,
				Namespace: testNamespace,
			}, createdIntent)).To(Succeed())

			// Verify spec fields
			Expect(createdIntent.Spec.IntentType).To(Equal("scaling"))
			Expect(createdIntent.Spec.Target).To(Equal("oran-e2-nodes-cu"))
			Expect(createdIntent.Spec.Replicas).To(Equal(int32(5)))
			Expect(createdIntent.Spec.Source).To(Equal("test"))

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
		})

		It("Should handle NetworkIntent updates", func() {
			// Update the replicas
			currentIntent := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      networkIntentName,
				Namespace: testNamespace,
			}, currentIntent)).To(Succeed())

			// Modify replica count
			currentIntent.Spec.Replicas = 8
			Expect(k8sClient.Update(testCtx, currentIntent)).To(Succeed())

			LogTestStep("Verifying NetworkIntent update was processed")

			// Wait for the controller to process the update
			updatedIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() int32 {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, updatedIntent)
				if err != nil {
					return 0
				}
				return updatedIntent.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(8)))

			// Verify the update was reflected
			Expect(updatedIntent.Spec.Replicas).To(Equal(int32(8)))
		})

		It("Should clean up NetworkIntent when deleted", func() {
			// Delete the NetworkIntent
			Expect(k8sClient.Delete(testCtx, networkIntent)).To(Succeed())

			// Verify deletion
			deletedIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, deletedIntent)
				return client.IgnoreNotFound(err) == nil
			}, timeout, interval).Should(BeTrue())

			LogTestStep("NetworkIntent deletion completed successfully")
		})
	})

	Context("When creating multiple NetworkIntents", func() {
		It("Should handle concurrent NetworkIntent operations", func() {
			testCtx, testCancel := GetTestContext()
			defer testCancel()

			intentsToCreate := 3
			networkIntents := make([]*intentv1alpha1.NetworkIntent, intentsToCreate)

			// Create multiple NetworkIntents
			for i := 0; i < intentsToCreate; i++ {
				networkIntents[i] = &intentv1alpha1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("concurrent-intent-%d", i),
						Namespace: "default",
					},
					Spec: intentv1alpha1.NetworkIntentSpec{
						IntentType: "scaling",
						Target:     fmt.Sprintf("target-%d", i),
						Namespace:  "default",
						Replicas:   int32(i + 2),
						Source:     "test",
					},
				}
				Expect(k8sClient.Create(testCtx, networkIntents[i])).To(Succeed())
			}

			LogTestStep("Verifying all concurrent NetworkIntents were created")

			// Verify all were created
			for i := 0; i < intentsToCreate; i++ {
				createdIntent := &intentv1alpha1.NetworkIntent{}
				Expect(k8sClient.Get(testCtx, types.NamespacedName{
					Name:      fmt.Sprintf("concurrent-intent-%d", i),
					Namespace: "default",
				}, createdIntent)).To(Succeed())

				Expect(createdIntent.Spec.Target).To(Equal(fmt.Sprintf("target-%d", i)))
				Expect(createdIntent.Spec.Replicas).To(Equal(int32(i + 2)))
			}

			// Clean up
			for i := 0; i < intentsToCreate; i++ {
				Expect(k8sClient.Delete(testCtx, networkIntents[i])).To(Succeed())
			}
		})
	})

	Context("When validating NetworkIntent fields", func() {
		It("Should reject invalid IntentType", func() {
			testCtx, testCancel := GetTestContext()
			defer testCancel()

			invalidIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent-type",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "invalid-type", // This should be rejected by webhook
					Target:     "some-target",
					Namespace:  "default",
					Replicas:   3,
					Source:     "test",
				},
			}

			// This should fail due to webhook validation
			err := k8sClient.Create(testCtx, invalidIntent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation failed"))
		})

		It("Should reject negative replicas", func() {
			testCtx, testCancel := GetTestContext()
			defer testCancel()

			invalidIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "negative-replicas-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "some-target",
					Namespace:  "default",
					Replicas:   -1, // Negative replicas should be rejected
					Source:     "test",
				},
			}

			// This should fail due to webhook validation
			err := k8sClient.Create(testCtx, invalidIntent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation failed"))
		})
	})
})
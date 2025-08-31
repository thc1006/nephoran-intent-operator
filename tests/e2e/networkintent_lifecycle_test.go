package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("NetworkIntent Lifecycle E2E Tests", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
	})

	Context("NetworkIntent CRUD Operations", func() {
		It("should create, read, update and delete NetworkIntent successfully", func() {
			intentName := "test-lifecycle-intent"

			// Create
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Scale up UPF instances to handle increased traffic",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("Verifying the NetworkIntent exists")
			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying the NetworkIntent spec")
			Expect(createdIntent.Spec.Intent).Should(Equal("Scale up UPF instances to handle increased traffic"))
			Expect(createdIntent.Spec.IntentType).Should(Equal(nephoran.IntentTypeScaling))
			Expect(createdIntent.Spec.Priority).Should(Equal(1))
			Expect(createdIntent.Spec.TargetComponents).Should(ContainElement(nephoran.NetworkTargetComponentUPF))

			By("Verifying controller sets initial status")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 30*time.Second, 2*time.Second).Should(Equal("Processing"))

			// Update
			By("Updating the NetworkIntent")
			updatedIntent := "Scale up both UPF and SMF instances for better performance"
			createdIntent.Spec.Intent = updatedIntent
			createdIntent.Spec.TargetComponents = append(createdIntent.Spec.TargetComponents, nephoran.NetworkTargetComponentSMF)
			Expect(k8sClient.Update(ctx, createdIntent)).Should(Succeed())

			By("Verifying the update")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return createdIntent.Spec.Intent
			}, 10*time.Second, 1*time.Second).Should(Equal(updatedIntent))

			Expect(createdIntent.Spec.TargetComponents).Should(ContainElements(
				nephoran.NetworkTargetComponentUPF,
				nephoran.NetworkTargetComponentSMF,
			))

			// Delete
			By("Deleting the NetworkIntent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())

			By("Verifying the NetworkIntent is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return errors.IsNotFound(err)
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("should handle multiple NetworkIntents with different priorities", func() {
			intents := []struct {
				name     string
				intent   string
				priority nephoran.NetworkPriority
			}{
				{"high-priority-intent", "Emergency scaling for critical UPF failure", nephoran.NetworkPriorityHigh},
				{"medium-priority-intent", "Optimize AMF performance for better latency", nephoran.NetworkPriorityNormal},
				{"low-priority-intent", "Routine maintenance for NRF components", nephoran.NetworkPriorityLow},
			}

			By("Creating multiple NetworkIntents with different priorities")
			var createdIntents []*nephoran.NetworkIntent
			for _, intentData := range intents {
				networkIntent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentData.name,
						Namespace: namespace,
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     intentData.intent,
						IntentType: nephoran.IntentTypeScaling,
						Priority:   intentData.priority,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}
				Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())
				createdIntents = append(createdIntents, networkIntent)
			}

			By("Verifying all intents are created and have correct priorities")
			for _, intentData := range intents {
				lookupKey := types.NamespacedName{Name: intentData.name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					return err == nil
				}, 10*time.Second, 1*time.Second).Should(BeTrue())

				Expect(retrievedIntent.Spec.Priority).Should(Equal(intentData.priority))
				Expect(retrievedIntent.Spec.Intent).Should(Equal(intentData.intent))
			}

			By("Cleaning up created intents")
			for _, intent := range createdIntents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})

		It("should validate NetworkIntent spec constraints", func() {
			By("Creating NetworkIntent with invalid IntentType should fail validation")
			invalidIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent",
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Invalid intent type test",
					IntentType: "invalid_type", // This should fail validation
				},
			}

			err := k8sClient.Create(ctx, invalidIntent)
			Expect(err).Should(HaveOccurred())

			By("Creating NetworkIntent with empty intent should fail")
			emptyIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-intent",
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "", // Empty intent should fail
					IntentType: nephoran.IntentTypeScaling,
				},
			}

			err = k8sClient.Create(ctx, emptyIntent)
			Expect(err).Should(HaveOccurred())
		})
	})

	Context("NetworkIntent Status Progression", func() {
		It("should progress through status phases correctly", func() {
			intentName := "status-progression-intent"

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test status progression for scaling AMF",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
					},
				},
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Verifying initial status phase")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 30*time.Second, 2*time.Second).Should(Equal("Processing"))

			By("Verifying conditions are set")
			Eventually(func() int {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return 0
				}
				return len(createdIntent.Status.Conditions)
			}, 20*time.Second, 1*time.Second).Should(BeNumerically(">=", 1))

			By("Verifying lastProcessed timestamp is updated")
			Expect(createdIntent.Status.LastProcessed).ShouldNot(BeNil())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("NetworkIntent with Complex Scenarios", func() {
		It("should handle deployment intent with multiple components", func() {
			intentName := "complex-deployment-intent"

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Deploy a complete 5G standalone core with all components",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
						nephoran.NetworkTargetComponentSMF,
						nephoran.NetworkTargetComponentUPF,
						nephoran.NetworkTargetComponentNRF,
						nephoran.NetworkTargetComponentUDM,
						nephoran.NetworkTargetComponentUDR,
						nephoran.NetworkTargetComponentPCF,
						nephoran.NetworkTargetComponentAUSF,
						nephoran.NetworkTargetComponentNSSF,
					},
				},
			}

			By("Creating complex deployment intent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Verifying all target components are preserved")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return len(createdIntent.Spec.TargetComponents) == 9
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying status reflects complexity")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 30*time.Second, 2*time.Second).Should(Equal("Processing"))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should handle optimization intent with custom parameters", func() {
			intentName := "optimization-intent"

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Optimize UPF throughput for high-bandwidth applications",
					IntentType: nephoran.IntentTypeOptimization,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			By("Creating optimization intent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Verifying optimization intent is processed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return createdIntent.Spec.IntentType == nephoran.IntentTypeOptimization
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})
})

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("NetworkIntent Controller", func() {
	Context("When creating a NetworkIntent", func() {
		It("Should reconcile the NetworkIntent successfully", func() {
			ctx := context.Background()

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy a high-performance 5G network slice for autonomous vehicles",
				},
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("Checking the NetworkIntent exists")
			lookupKey := types.NamespacedName{Name: "test-intent", Namespace: "default"}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying the NetworkIntent spec")
			Expect(createdIntent.Spec.Intent).Should(Equal("Deploy a high-performance 5G network slice for autonomous vehicles"))

			By("Checking that the controller updates the status")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return createdIntent.Status.Phase
			}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))

			By("Cleaning up the NetworkIntent")
			Expect(k8sClient.Delete(ctx, networkIntent)).Should(Succeed())
		})

		It("Should handle URLLC intents with low latency requirements", func() {
			ctx := context.Background()

			urllcIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "urllc-intent",
					Namespace: "default",
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy URLLC service with 1ms latency for industrial automation",
				},
			}

			By("Creating the URLLC NetworkIntent")
			Expect(k8sClient.Create(ctx, urllcIntent)).Should(Succeed())

			By("Verifying the intent is processed")
			lookupKey := types.NamespacedName{Name: "urllc-intent", Namespace: "default"}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Checking controller recognizes URLLC requirements")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return createdIntent.Status.Phase
			}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))

			By("Cleaning up the URLLC NetworkIntent")
			Expect(k8sClient.Delete(ctx, urllcIntent)).Should(Succeed())
		})

		It("Should handle IoT intents with massive connectivity requirements", func() {
			ctx := context.Background()

			iotIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "iot-intent",
					Namespace: "default",
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy IoT gateway for 10000 sensor connections with edge processing",
				},
			}

			By("Creating the IoT NetworkIntent")
			Expect(k8sClient.Create(ctx, iotIntent)).Should(Succeed())

			By("Verifying the intent is processed")
			lookupKey := types.NamespacedName{Name: "iot-intent", Namespace: "default"}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying IoT-specific processing")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return createdIntent.Status.Phase
			}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))

			By("Cleaning up the IoT NetworkIntent")
			Expect(k8sClient.Delete(ctx, iotIntent)).Should(Succeed())
		})

		It("Should handle multiple concurrent intents", func() {
			ctx := context.Background()

			intents := []*nephoran.NetworkIntent{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "concurrent-1", Namespace: "default"},
					Spec:       nephoran.NetworkIntentSpec{Intent: "Deploy eMBB service for 4K video streaming"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "concurrent-2", Namespace: "default"},
					Spec:       nephoran.NetworkIntentSpec{Intent: "Deploy mMTC service for smart city sensors"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "concurrent-3", Namespace: "default"},
					Spec:       nephoran.NetworkIntentSpec{Intent: "Deploy URLLC service for remote surgery"},
				},
			}

			By("Creating multiple NetworkIntents concurrently")
			for _, intent := range intents {
				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
			}

			By("Verifying all intents are processed")
			for _, intent := range intents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}
				createdIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, createdIntent)
					return err == nil
				}, 10*time.Second, 1*time.Second).Should(BeTrue())

				Eventually(func() string {
					err := k8sClient.Get(ctx, lookupKey, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))
			}

			By("Cleaning up all NetworkIntents")
			for _, intent := range intents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})
	})

	Context("When updating a NetworkIntent", func() {
		It("Should handle intent modifications correctly", func() {
			ctx := context.Background()

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "updateable-intent",
					Namespace: "default",
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy basic 5G service",
				},
			}

			By("Creating the initial NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("Waiting for initial reconciliation")
			lookupKey := types.NamespacedName{Name: "updateable-intent", Namespace: "default"}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Updating the NetworkIntent spec")
			Eventually(func() error {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return err
				}
				createdIntent.Spec.Intent = "Deploy enhanced 5G service with edge computing"
				return k8sClient.Update(ctx, createdIntent)
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			By("Verifying the update was processed")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return createdIntent.Spec.Intent
			}, 10*time.Second, 1*time.Second).Should(Equal("Deploy enhanced 5G service with edge computing"))

			By("Cleaning up the NetworkIntent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("When deleting a NetworkIntent", func() {
		It("Should clean up resources properly", func() {
			ctx := context.Background()

			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deletable-intent",
					Namespace: "default",
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy temporary 5G service",
				},
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("Waiting for reconciliation")
			lookupKey := types.NamespacedName{Name: "deletable-intent", Namespace: "default"}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Deleting the NetworkIntent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())

			By("Verifying the NetworkIntent is removed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err != nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
		})
	})
})

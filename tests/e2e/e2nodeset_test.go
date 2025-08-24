//go:build integration

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

var _ = Describe("E2NodeSet Controller", func() {
	Context("When creating an E2NodeSet", func() {
		It("Should reconcile the E2NodeSet successfully", func() {
			ctx := context.Background()

			e2nodeset := &nephoran.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-e2nodeset",
					Namespace: "default",
				},
				Spec: nephoran.E2NodeSetSpec{
					E2NodeConfig: nephoran.E2NodeConfig{
						E2NodeID: "test-e2node-001",
						RICConfig: nephoran.RICConfig{
							RICID:      "ric-001",
							RICAddress: "http://ric.nephoran.local:8080",
						},
					},
					Replicas: 3,
				},
			}

			By("Creating the E2NodeSet")
			Expect(k8sClient.Create(ctx, e2nodeset)).Should(Succeed())

			By("Checking the E2NodeSet exists")
			lookupKey := types.NamespacedName{Name: "test-e2nodeset", Namespace: "default"}
			createdE2NodeSet := &nephoran.E2NodeSet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying the E2NodeSet spec")
			Expect(createdE2NodeSet.Spec.E2NodeConfig.E2NodeID).Should(Equal("test-e2node-001"))
			Expect(createdE2NodeSet.Spec.Replicas).Should(Equal(int32(3)))

			By("Checking that the controller updates the status")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return ""
				}
				return createdE2NodeSet.Status.Phase
			}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))

			By("Cleaning up the E2NodeSet")
			Expect(k8sClient.Delete(ctx, e2nodeset)).Should(Succeed())
		})

		It("Should handle scaling of E2NodeSet replicas", func() {
			ctx := context.Background()

			e2nodeset := &nephoran.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scalable-e2nodeset",
					Namespace: "default",
				},
				Spec: nephoran.E2NodeSetSpec{
					E2NodeConfig: nephoran.E2NodeConfig{
						E2NodeID: "scalable-e2node-001",
						RICConfig: nephoran.RICConfig{
							RICID:      "ric-scalable-001",
							RICAddress: "http://ric-scalable.nephoran.local:8080",
						},
					},
					Replicas: 1,
				},
			}

			By("Creating the E2NodeSet with 1 replica")
			Expect(k8sClient.Create(ctx, e2nodeset)).Should(Succeed())

			By("Waiting for initial reconciliation")
			lookupKey := types.NamespacedName{Name: "scalable-e2nodeset", Namespace: "default"}
			createdE2NodeSet := &nephoran.E2NodeSet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Scaling up to 5 replicas")
			Eventually(func() error {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return err
				}
				createdE2NodeSet.Spec.Replicas = 5
				return k8sClient.Update(ctx, createdE2NodeSet)
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			By("Verifying the replica count was updated")
			Eventually(func() int32 {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return 0
				}
				return createdE2NodeSet.Spec.Replicas
			}, 10*time.Second, 1*time.Second).Should(Equal(int32(5)))

			By("Scaling down to 2 replicas")
			Eventually(func() error {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return err
				}
				createdE2NodeSet.Spec.Replicas = 2
				return k8sClient.Update(ctx, createdE2NodeSet)
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			By("Verifying the scale down")
			Eventually(func() int32 {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return 0
				}
				return createdE2NodeSet.Spec.Replicas
			}, 10*time.Second, 1*time.Second).Should(Equal(int32(2)))

			By("Cleaning up the E2NodeSet")
			Expect(k8sClient.Delete(ctx, createdE2NodeSet)).Should(Succeed())
		})

		It("Should handle E2NodeSet with different RIC configurations", func() {
			ctx := context.Background()

			ricConfigs := []struct {
				name       string
				ricID      string
				ricAddress string
			}{
				{"near-rt-ric", "near-rt-001", "http://near-rt-ric.nephoran.local:8080"},
				{"non-rt-ric", "non-rt-001", "http://non-rt-ric.nephoran.local:9090"},
				{"edge-ric", "edge-001", "http://edge-ric.nephoran.local:7070"},
			}

			var e2nodesets []*nephoran.E2NodeSet

			By("Creating E2NodeSets with different RIC configurations")
			for _, config := range ricConfigs {
				e2nodeset := &nephoran.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      config.name + "-e2nodeset",
						Namespace: "default",
					},
					Spec: nephoran.E2NodeSetSpec{
						E2NodeConfig: nephoran.E2NodeConfig{
							E2NodeID: config.name + "-e2node",
							RICConfig: nephoran.RICConfig{
								RICID:      config.ricID,
								RICAddress: config.ricAddress,
							},
						},
						Replicas: 1,
					},
				}

				Expect(k8sClient.Create(ctx, e2nodeset)).Should(Succeed())
				e2nodesets = append(e2nodesets, e2nodeset)
			}

			By("Verifying all E2NodeSets are created and reconciled")
			for _, e2nodeset := range e2nodesets {
				lookupKey := types.NamespacedName{Name: e2nodeset.Name, Namespace: e2nodeset.Namespace}
				createdE2NodeSet := &nephoran.E2NodeSet{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
					return err == nil
				}, 10*time.Second, 1*time.Second).Should(BeTrue())

				Eventually(func() string {
					err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
					if err != nil {
						return ""
					}
					return createdE2NodeSet.Status.Phase
				}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))
			}

			By("Cleaning up all E2NodeSets")
			for _, e2nodeset := range e2nodesets {
				Expect(k8sClient.Delete(ctx, e2nodeset)).Should(Succeed())
			}
		})
	})

	Context("When handling E2NodeSet failures", func() {
		It("Should handle invalid RIC configurations gracefully", func() {
			ctx := context.Background()

			invalidE2NodeSet := &nephoran.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-e2nodeset",
					Namespace: "default",
				},
				Spec: nephoran.E2NodeSetSpec{
					E2NodeConfig: nephoran.E2NodeConfig{
						E2NodeID: "", // Invalid empty E2NodeID
						RICConfig: nephoran.RICConfig{
							RICID:      "",
							RICAddress: "invalid-url", // Invalid URL format
						},
					},
					Replicas: 1,
				},
			}

			By("Creating E2NodeSet with invalid configuration")
			Expect(k8sClient.Create(ctx, invalidE2NodeSet)).Should(Succeed())

			By("Verifying the E2NodeSet exists but may have error status")
			lookupKey := types.NamespacedName{Name: "invalid-e2nodeset", Namespace: "default"}
			createdE2NodeSet := &nephoran.E2NodeSet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Checking that controller attempts to process even invalid configurations")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				if err != nil {
					return ""
				}
				return createdE2NodeSet.Status.Phase
			}, 30*time.Second, 2*time.Second).Should(Not(BeEmpty()))

			By("Cleaning up the invalid E2NodeSet")
			Expect(k8sClient.Delete(ctx, invalidE2NodeSet)).Should(Succeed())
		})
	})

	Context("When deleting an E2NodeSet", func() {
		It("Should clean up resources properly", func() {
			ctx := context.Background()

			e2nodeset := &nephoran.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deletable-e2nodeset",
					Namespace: "default",
				},
				Spec: nephoran.E2NodeSetSpec{
					E2NodeConfig: nephoran.E2NodeConfig{
						E2NodeID: "deletable-e2node",
						RICConfig: nephoran.RICConfig{
							RICID:      "deletable-ric",
							RICAddress: "http://deletable-ric.nephoran.local:8080",
						},
					},
					Replicas: 2,
				},
			}

			By("Creating the E2NodeSet")
			Expect(k8sClient.Create(ctx, e2nodeset)).Should(Succeed())

			By("Waiting for reconciliation")
			lookupKey := types.NamespacedName{Name: "deletable-e2nodeset", Namespace: "default"}
			createdE2NodeSet := &nephoran.E2NodeSet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Deleting the E2NodeSet")
			Expect(k8sClient.Delete(ctx, createdE2NodeSet)).Should(Succeed())

			By("Verifying the E2NodeSet is removed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdE2NodeSet)
				return err != nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
		})
	})
})

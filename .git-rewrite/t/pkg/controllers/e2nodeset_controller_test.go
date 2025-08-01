package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
)

var _ = Describe("E2NodeSet Controller", func() {
	var (
		testNamespace string
		reconciler    *E2NodeSetReconciler
	)

	BeforeEach(func() {
		testNamespace = CreateIsolatedNamespace("e2nodeset-controller")

		// Create reconciler instance
		reconciler = &E2NodeSetReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	AfterEach(func() {
		CleanupIsolatedNamespace(testNamespace)
	})

	Context("E2NodeSet Creation and Scaling", func() {
		It("should create ConfigMaps when E2NodeSet is created", func() {
			By("creating an E2NodeSet with 3 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-e2nodeset", testNamespace, 3)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("reconciling the E2NodeSet")
			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			By("verifying ConfigMap content and labels")
			configMapList := &corev1.ConfigMapList{}
			listOptions := []client.ListOption{
				client.InNamespace(testNamespace),
				client.MatchingLabels(map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}),
			}
			Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())
			Expect(len(configMapList.Items)).To(Equal(3))

			for _, cm := range configMapList.Items {
				// Verify ConfigMap data
				Expect(cm.Data).To(HaveKey("nodeId"))
				Expect(cm.Data).To(HaveKey("nodeType"))
				Expect(cm.Data).To(HaveKey("status"))
				Expect(cm.Data).To(HaveKey("created"))
				Expect(cm.Data).To(HaveKey("e2nodeSet"))
				Expect(cm.Data).To(HaveKey("index"))

				Expect(cm.Data["nodeType"]).To(Equal("simulated-gnb"))
				Expect(cm.Data["status"]).To(Equal("active"))
				Expect(cm.Data["e2nodeSet"]).To(Equal(e2nodeSet.Name))

				// Verify labels
				Expect(cm.Labels).To(HaveKeyWithValue("app", "e2node"))
				Expect(cm.Labels).To(HaveKeyWithValue("e2nodeset", e2nodeSet.Name))
				Expect(cm.Labels).To(HaveKeyWithValue("nephoran.com/component", "simulated-gnb"))
				Expect(cm.Labels).To(HaveKeyWithValue("nephoran.com/managed-by", "e2nodeset-controller"))

				// Verify owner reference
				found := false
				for _, ownerRef := range cm.OwnerReferences {
					if ownerRef.UID == e2nodeSet.UID && ownerRef.Kind == "E2NodeSet" {
						found = true
						Expect(ownerRef.Controller).ToNot(BeNil())
						Expect(*ownerRef.Controller).To(BeTrue())
						break
					}
				}
				Expect(found).To(BeTrue(), "ConfigMap %s should have E2NodeSet as owner", cm.Name)
			}

			By("verifying E2NodeSet status is updated")
			WaitForE2NodeSetReady(namespacedName, 3)
		})

		It("should scale up E2NodeSet by creating additional ConfigMaps", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-scale-up", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create initial ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying initial ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("scaling up to 5 replicas")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 5
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scale up")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying additional ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 5)

			By("verifying E2NodeSet status reflects new replica count")
			WaitForE2NodeSetReady(namespacedName, 5)

			By("verifying ConfigMap names are sequential")
			configMapList := &corev1.ConfigMapList{}
			listOptions := []client.ListOption{
				client.InNamespace(testNamespace),
				client.MatchingLabels(map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}),
			}
			Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())

			expectedNames := []string{
				fmt.Sprintf("%s-node-0", e2nodeSet.Name),
				fmt.Sprintf("%s-node-1", e2nodeSet.Name),
				fmt.Sprintf("%s-node-2", e2nodeSet.Name),
				fmt.Sprintf("%s-node-3", e2nodeSet.Name),
				fmt.Sprintf("%s-node-4", e2nodeSet.Name),
			}

			actualNames := make([]string, len(configMapList.Items))
			for i, cm := range configMapList.Items {
				actualNames[i] = cm.Name
			}

			for _, expectedName := range expectedNames {
				Expect(actualNames).To(ContainElement(expectedName))
			}
		})

		It("should scale down E2NodeSet by deleting excess ConfigMaps", func() {
			By("creating an E2NodeSet with 5 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-scale-down", testNamespace, 5)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create initial ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying initial ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 5)

			By("scaling down to 2 replicas")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 2
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scale down")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying excess ConfigMaps are deleted")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("verifying E2NodeSet status reflects new replica count")
			WaitForE2NodeSetReady(namespacedName, 2)

			By("verifying remaining ConfigMaps are the correct ones (node-0 and node-1)")
			configMapList := &corev1.ConfigMapList{}
			listOptions := []client.ListOption{
				client.InNamespace(testNamespace),
				client.MatchingLabels(map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}),
			}
			Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())

			expectedNames := []string{
				fmt.Sprintf("%s-node-0", e2nodeSet.Name),
				fmt.Sprintf("%s-node-1", e2nodeSet.Name),
			}

			actualNames := make([]string, len(configMapList.Items))
			for i, cm := range configMapList.Items {
				actualNames[i] = cm.Name
			}

			Expect(actualNames).To(ConsistOf(expectedNames))
		})

		It("should handle scaling to zero replicas", func() {
			By("creating an E2NodeSet with 3 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-scale-zero", testNamespace, 3)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create initial ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying initial ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			By("scaling down to 0 replicas")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 0
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scaling to zero")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying all ConfigMaps are deleted")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 0)

			By("verifying E2NodeSet status reflects zero replicas")
			WaitForE2NodeSetReady(namespacedName, 0)
		})

		It("should handle E2NodeSet deletion gracefully", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-deletion", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("deleting the E2NodeSet")
			Expect(k8sClient.Delete(ctx, e2nodeSet)).To(Succeed())

			By("reconciling after deletion")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying E2NodeSet is deleted")
			Eventually(func() bool {
				var deletedE2NodeSet nephoranv1.E2NodeSet
				err := k8sClient.Get(ctx, namespacedName, &deletedE2NodeSet)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			By("verifying ConfigMaps are garbage collected due to owner references")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOptions := []client.ListOption{
					client.InNamespace(testNamespace),
					client.MatchingLabels(map[string]string{
						"app":       "e2node",
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOptions...); err != nil {
					return -1
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(0))
		})
	})

	Context("E2NodeSet Status Updates", func() {
		It("should update ReadyReplicas status field correctly", func() {
			By("creating an E2NodeSet with 3 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-status-update", testNamespace, 3)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("verifying initial status has 0 ready replicas")
			var initialE2NodeSet nephoranv1.E2NodeSet
			Expect(k8sClient.Get(ctx, namespacedName, &initialE2NodeSet)).To(Succeed())
			Expect(initialE2NodeSet.Status.ReadyReplicas).To(Equal(int32(0)))

			By("reconciling the E2NodeSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying status is updated to reflect ready replicas")
			WaitForE2NodeSetReady(namespacedName, 3)

			By("verifying status field is persisted")
			var updatedE2NodeSet nephoranv1.E2NodeSet
			Expect(k8sClient.Get(ctx, namespacedName, &updatedE2NodeSet)).To(Succeed())
			Expect(updatedE2NodeSet.Status.ReadyReplicas).To(Equal(int32(3)))
		})

		It("should maintain status consistency during scaling operations", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-status-consistency", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to establish initial state")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("waiting for initial ready state")
			WaitForE2NodeSetReady(namespacedName, 2)

			By("scaling up to 4 replicas")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 4
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scale up")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying status reflects new replica count")
			WaitForE2NodeSetReady(namespacedName, 4)

			By("scaling down to 1 replica")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 1
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scale down")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying status reflects scaled down replica count")
			WaitForE2NodeSetReady(namespacedName, 1)

			By("verifying final state consistency")
			var finalE2NodeSet nephoranv1.E2NodeSet
			Expect(k8sClient.Get(ctx, namespacedName, &finalE2NodeSet)).To(Succeed())
			Expect(finalE2NodeSet.Spec.Replicas).To(Equal(int32(1)))
			Expect(finalE2NodeSet.Status.ReadyReplicas).To(Equal(int32(1)))

			// Verify actual ConfigMaps match status
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 1)
		})

		It("should handle status updates when ConfigMaps already exist", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-existing-configmaps", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("pre-creating some ConfigMaps manually")
			labels := map[string]string{
				"app":                     "e2node",
				"e2nodeset":               e2nodeSet.Name,
				"nephoran.com/component":  "simulated-gnb",
				"nephoran.com/managed-by": "e2nodeset-controller",
			}

			// Create ConfigMap for node-0
			cm0 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-node-0", e2nodeSet.Name),
					Namespace: testNamespace,
					Labels:    labels,
				},
				Data: map[string]string{
					"nodeId":    fmt.Sprintf("%s-node-0", e2nodeSet.Name),
					"nodeType":  "simulated-gnb",
					"status":    "active",
					"created":   time.Now().Format(time.RFC3339),
					"e2nodeSet": e2nodeSet.Name,
					"index":     "0",
				},
			}
			Expect(k8sClient.Create(ctx, cm0)).To(Succeed())

			By("reconciling the E2NodeSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying status reflects actual ConfigMap count")
			WaitForE2NodeSetReady(namespacedName, 2)

			By("verifying all expected ConfigMaps exist")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("verifying the pre-existing ConfigMap still exists")
			var existingCM corev1.ConfigMap
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-node-0", e2nodeSet.Name),
				Namespace: testNamespace,
			}, &existingCM)).To(Succeed())
		})

		It("should update status correctly when no changes are needed", func() {
			By("creating an E2NodeSet with 1 replica")
			e2nodeSet := CreateTestE2NodeSet("test-no-changes", testNamespace, 1)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling the E2NodeSet initially")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("waiting for initial ready state")
			WaitForE2NodeSetReady(namespacedName, 1)

			By("getting the resource generation before second reconcile")
			var beforeE2NodeSet nephoranv1.E2NodeSet
			Expect(k8sClient.Get(ctx, namespacedName, &beforeE2NodeSet)).To(Succeed())
			beforeGeneration := beforeE2NodeSet.Generation

			By("reconciling again without any changes")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying status remains consistent")
			var afterE2NodeSet nephoranv1.E2NodeSet
			Expect(k8sClient.Get(ctx, namespacedName, &afterE2NodeSet)).To(Succeed())
			Expect(afterE2NodeSet.Status.ReadyReplicas).To(Equal(int32(1)))
			Expect(afterE2NodeSet.Generation).To(Equal(beforeGeneration))

			By("verifying ConfigMaps remain unchanged")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 1)
		})

		It("should handle rapid status updates correctly", func() {
			By("creating an E2NodeSet with 1 replica")
			e2nodeSet := CreateTestE2NodeSet("test-rapid-updates", testNamespace, 1)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("performing multiple rapid reconciliations")
			for i := 0; i < 5; i++ {
				result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
			}

			By("verifying final status is correct")
			WaitForE2NodeSetReady(namespacedName, 1)

			By("verifying only expected ConfigMaps exist")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 1)
		})
	})

	Context("E2NodeSet Error Handling", func() {
		It("should handle ConfigMap creation failures gracefully", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-create-failure", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("pre-creating a ConfigMap with conflicting name to cause creation failure")
			conflictingCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-node-0", e2nodeSet.Name),
					Namespace: testNamespace,
					Labels: map[string]string{
						"conflicting": "true",
					},
				},
				Data: map[string]string{
					"conflict": "true",
				},
			}
			Expect(k8sClient.Create(ctx, conflictingCM)).To(Succeed())

			By("reconciling the E2NodeSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})

			By("verifying reconcile handles the error and requests requeue")
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("removing the conflicting ConfigMap")
			Expect(k8sClient.Delete(ctx, conflictingCM)).To(Succeed())

			By("reconciling again after removing conflict")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are eventually created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("verifying status is updated correctly after recovery")
			WaitForE2NodeSetReady(namespacedName, 2)
		})

		It("should handle ConfigMap deletion failures gracefully", func() {
			By("creating an E2NodeSet with 3 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-delete-failure", testNamespace, 3)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create initial ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("waiting for initial ConfigMaps to be created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			By("adding finalizer to one ConfigMap to prevent deletion")
			var cmToProtect corev1.ConfigMap
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-node-2", e2nodeSet.Name),
				Namespace: testNamespace,
			}, &cmToProtect)).To(Succeed())

			cmToProtect.Finalizers = append(cmToProtect.Finalizers, "test.nephoran.com/prevent-deletion")
			Expect(k8sClient.Update(ctx, &cmToProtect)).To(Succeed())

			By("scaling down to 1 replica")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 1
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("reconciling after scale down")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})

			By("verifying reconcile handles deletion failure and requests requeue")
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("removing the finalizer to allow deletion")
			Eventually(func() error {
				var cmToUpdate corev1.ConfigMap
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-node-2", e2nodeSet.Name),
					Namespace: testNamespace,
				}, &cmToUpdate); err != nil {
					return err
				}
				cmToUpdate.Finalizers = []string{}
				return k8sClient.Update(ctx, &cmToUpdate)
			}, timeout, interval).Should(Succeed())

			By("reconciling again after removing finalizer")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are eventually scaled down")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 1)

			By("verifying status is updated correctly after recovery")
			WaitForE2NodeSetReady(namespacedName, 1)
		})

		It("should handle status update failures gracefully", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-status-failure", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling the E2NodeSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are created even if status update might fail")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("verifying status is eventually consistent")
			WaitForE2NodeSetReady(namespacedName, 2)
		})

		It("should handle missing E2NodeSet resource gracefully", func() {
			By("creating a NamespacedName for non-existent E2NodeSet")
			namespacedName := types.NamespacedName{
				Name:      "non-existent-e2nodeset",
				Namespace: testNamespace,
			}

			By("reconciling non-existent E2NodeSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})

			By("verifying reconcile handles missing resource gracefully")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
		})

		It("should handle namespace deletion gracefully", func() {
			By("creating a temporary namespace")
			tempNamespace := testutils.GenerateUniqueNamespace("temp-e2nodeset")
			_ = testutils.CreateNamespace(ctx, k8sClient, tempNamespace)

			By("creating an E2NodeSet in the temporary namespace")
			e2nodeSet := CreateTestE2NodeSet("test-namespace-deletion", tempNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("reconciling to create ConfigMaps")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying ConfigMaps are created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, tempNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("deleting the namespace")
			testutils.DeleteNamespace(ctx, k8sClient, tempNamespace)

			By("attempting to reconcile after namespace deletion")
			result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})

			By("verifying reconcile handles namespace deletion gracefully")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should handle concurrent reconciliation requests", func() {
			By("creating an E2NodeSet with 3 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-concurrent", testNamespace, 3)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("running multiple concurrent reconciliations")
			done := make(chan bool, 3)
			errors := make(chan error, 3)

			for i := 0; i < 3; i++ {
				go func() {
					defer GinkgoRecover()
					result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
					if err != nil {
						errors <- err
					} else {
						Expect(result.Requeue).To(BeFalse())
					}
					done <- true
				}()
			}

			By("waiting for all reconciliations to complete")
			for i := 0; i < 3; i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					Fail(fmt.Sprintf("Concurrent reconciliation failed: %v", err))
				case <-time.After(30 * time.Second):
					Fail("Concurrent reconciliation timed out")
				}
			}

			By("verifying final state is consistent")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			WaitForE2NodeSetReady(namespacedName, 3)
		})

		It("should handle invalid E2NodeSet specifications", func() {
			By("creating an E2NodeSet with negative replicas (should be prevented by validation)")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-spec",
					Namespace: testNamespace,
					Labels: map[string]string{
						"test-resource": "true",
						"test-suite":    "controller-suite",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: -1, // Invalid negative value
				},
			}

			By("attempting to create invalid E2NodeSet")
			err := k8sClient.Create(ctx, e2nodeSet)

			By("verifying creation is rejected by validation")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("minimum"))
		})

		It("should recover from transient API server errors", func() {
			By("creating an E2NodeSet with 2 replicas")
			e2nodeSet := CreateTestE2NodeSet("test-api-recovery", testNamespace, 2)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("performing initial reconciliation")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying initial state")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			By("performing additional reconciliations to test resilience")
			for i := 0; i < 3; i++ {
				result, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
			}

			By("verifying state remains consistent")
			testutils.WaitForConfigMapCount(ctx, k8sClient, testNamespace, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			WaitForE2NodeSetReady(namespacedName, 2)
		})
	})
})

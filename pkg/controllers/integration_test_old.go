//go:build integration

package controllers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Integration Tests - End-to-End Workflows", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Millisecond * 500
	)

	var namespaceName string

	BeforeEach(func() {
		By("Creating a new namespace for integration test isolation")
		namespaceName = CreateIsolatedNamespace("integration-test")
	})

	AfterEach(func() {
		By("Cleaning up the integration test namespace")
		CleanupIsolatedNamespace(namespaceName)
	})

	Context("Complete NetworkIntent Workflow", func() {
		var (
			networkIntentReconciler *NetworkIntentReconciler
		)

		BeforeEach(func() {
			By("Setting up NetworkIntent reconciler for integration testing")
			mockDeps := &testutils.MockDependencies{
				LLMClient: testutils.NewMockLLMClient(),
				GitClient: testutils.NewMockGitClient(),
			}
			networkIntentReconciler = &NetworkIntentReconciler{
				Client: k8sClient,
				Scheme: testEnv.Scheme,
				deps:   mockDeps,
			}
		})

		It("Should process NetworkIntent from creation to completion", func() {
			By("Creating a NetworkIntent for E2NodeSet scaling")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("e2e-scale-intent"),
				namespaceName,
				"Scale E2 nodes to 5 replicas for increased capacity",
			)

			// Set up mock LLM client for successful processing
			mockResponse := map[string]interface{}{
				"action":      "scale",
				"target_type": "E2NodeSet",
				"target_name": "test-e2nodeset",
				"replicas":    5,
				"reason":      "capacity_increase",
				"priority":    "high",
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			mockLLMClient := networkIntentReconciler.deps.(*testutils.MockDependencies).LLMClient.(*testutils.MockLLMClient)
			mockLLMClient.SetResponse(networkIntent.Spec.Intent, string(mockResponseBytes))

			// Set up mock Git client for successful deployment
			mockGitClient := networkIntentReconciler.deps.(*testutils.MockDependencies).GitClient.(*testutils.MockGitClient)
			mockGitClient.SetCommitHash("e2e-test-commit-123")

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Processing the NetworkIntent through the reconciler")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				},
			}

			// First reconciliation - should process intent
			_, err := networkIntentReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NetworkIntent status progression")
			Eventually(func() string {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal("Completed"))

			By("Verifying final NetworkIntent state")
			final := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), final)).To(Succeed())

			// Verify processing occurred
			Expect(isConditionTrue(final.Status.Conditions, "Processed")).To(BeTrue())
			Expect(isConditionTrue(final.Status.Conditions, "Deployed")).To(BeTrue())

			// Verify parameters were extracted
			Expect(final.Spec.Parameters.Raw).NotTo(BeEmpty())
			var extractedParams map[string]interface{}
			Expect(json.Unmarshal(final.Spec.Parameters.Raw, &extractedParams)).To(Succeed())
			Expect(extractedParams["action"]).To(Equal("scale"))
			Expect(extractedParams["replicas"]).To(Equal(float64(5)))

			// Verify Git deployment
			Expect(final.Status.GitCommitHash).To(Equal("e2e-test-commit-123"))

			// Verify timing fields
			Expect(final.Status.ProcessingStartTime).NotTo(BeNil())
			Expect(final.Status.ProcessingCompletionTime).NotTo(BeNil())
			Expect(final.Status.DeploymentStartTime).NotTo(BeNil())
			Expect(final.Status.DeploymentCompletionTime).NotTo(BeNil())
		})

		It("Should handle NetworkIntent failure and recovery workflow", func() {
			By("Creating a NetworkIntent with initial LLM failure")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("e2e-recovery-intent"),
				namespaceName,
				"Deploy new network function with redundancy",
			)

			// Set up mock LLM client that fails initially
			mockLLMClient := &MockLLMClient{
				Response:  "",
				Error:     fmt.Errorf("temporary service unavailable"),
				FailCount: 2, // Fail first 2 attempts
			}
			networkIntentReconciler.LLMClient = mockLLMClient
			networkIntentReconciler.GitClient = &MockGitClient{
				CommitHash: "recovery-commit-456",
				Error:      nil,
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				},
			}

			By("First reconciliation should fail and schedule retry")
			result, err := networkIntentReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.RetryDelay))

			By("Verifying retry state")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal("Processing"))
			retryCount := getRetryCount(updated, "llm-processing")
			Expect(retryCount).To(Equal(1))

			By("Second reconciliation should also fail")
			result, err = networkIntentReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.RetryDelay))

			By("Third reconciliation should succeed after LLM recovery")
			// Setup successful response for third attempt
			mockResponse := map[string]interface{}{
				"action":    "deploy",
				"component": "network-function",
				"replicas":  2,
				"ha_mode":   true,
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			mockLLMClient.Response = string(mockResponseBytes)
			mockLLMClient.Error = nil

			result, err = networkIntentReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying successful recovery")
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
				return updated.Status.Phase
			}, timeout, interval).Should(Equal("Completed"))

			Expect(isConditionTrue(updated.Status.Conditions, "Processed")).To(BeTrue())
			Expect(isConditionTrue(updated.Status.Conditions, "Deployed")).To(BeTrue())
		})
	})

	Context("Complete E2NodeSet Workflow", func() {
		var (
			e2nodeSetReconciler *E2NodeSetReconciler
		)

		BeforeEach(func() {
			By("Setting up E2NodeSet reconciler for integration testing")
			e2nodeSetReconciler = &E2NodeSetReconciler{
				Client:        k8sClient,
				Scheme:        testEnv.Scheme,
				EventRecorder: &record.FakeRecorder{},
			}
		})

		It("Should manage E2NodeSet lifecycle from creation to scaling", func() {
			By("Creating an E2NodeSet with initial replica count")
			e2nodeSet := CreateTestE2NodeSet(
				GetUniqueName("e2e-lifecycle"),
				namespaceName,
				3, // Initial replicas
			)

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      e2nodeSet.Name,
					Namespace: e2nodeSet.Namespace,
				},
			}

			By("Initial reconciliation should create ConfigMaps")
			result, err := e2nodeSetReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying initial ConfigMaps were created")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(3))

			By("Verifying E2NodeSet status reflects ready replicas")
			Eventually(func() int32 {
				updated := &nephoranv1.E2NodeSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated); err != nil {
					return 0
				}
				return updated.Status.ReadyReplicas
			}, timeout, interval).Should(Equal(int32(3)))

			By("Scaling E2NodeSet up to 5 replicas")
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), e2nodeSet)).To(Succeed())
			e2nodeSet.Spec.Replicas = 5
			Expect(k8sClient.Update(ctx, e2nodeSet)).To(Succeed())

			By("Reconciling after scale up")
			result, err = e2nodeSetReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying scale up completed")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(5))

			Eventually(func() int32 {
				updated := &nephoranv1.E2NodeSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated); err != nil {
					return 0
				}
				return updated.Status.ReadyReplicas
			}, timeout, interval).Should(Equal(int32(5)))

			By("Scaling E2NodeSet down to 2 replicas")
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), e2nodeSet)).To(Succeed())
			e2nodeSet.Spec.Replicas = 2
			Expect(k8sClient.Update(ctx, e2nodeSet)).To(Succeed())

			By("Reconciling after scale down")
			result, err = e2nodeSetReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying scale down completed")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(2))

			Eventually(func() int32 {
				updated := &nephoranv1.E2NodeSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated); err != nil {
					return 0
				}
				return updated.Status.ReadyReplicas
			}, timeout, interval).Should(Equal(int32(2)))
		})

		It("Should handle E2NodeSet deletion and cleanup", func() {
			By("Creating an E2NodeSet")
			e2nodeSet := CreateTestE2NodeSet(
				GetUniqueName("e2e-deletion"),
				namespaceName,
				2,
			)

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      e2nodeSet.Name,
					Namespace: e2nodeSet.Namespace,
				},
			}

			By("Initial reconciliation to create ConfigMaps")
			result, err := e2nodeSetReconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ConfigMaps were created")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(2))

			By("Deleting the E2NodeSet")
			Expect(k8sClient.Delete(ctx, e2nodeSet)).To(Succeed())

			By("Verifying E2NodeSet is deleted")
			Eventually(func() bool {
				deleted := &nephoranv1.E2NodeSet{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), deleted)
				return client.IgnoreNotFound(err) == nil
			}, timeout, interval).Should(BeTrue())

			By("Verifying ConfigMaps are cleaned up via owner references")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return -1
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(0))
		})
	})

	Context("Cross-Controller Integration", func() {
		It("Should demonstrate NetworkIntent driving E2NodeSet changes", func() {
			By("Creating an initial E2NodeSet")
			e2nodeSet := CreateTestE2NodeSet(
				GetUniqueName("target-e2nodeset"),
				namespaceName,
				2, // Initial replicas
			)
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			// Set up E2NodeSet reconciler
			e2nodeSetReconciler := &E2NodeSetReconciler{
				Client:        k8sClient,
				Scheme:        testEnv.Scheme,
				EventRecorder: &record.FakeRecorder{},
			}

			By("Initial E2NodeSet reconciliation")
			e2nsReq := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      e2nodeSet.Name,
					Namespace: e2nodeSet.Namespace,
				},
			}

			result, err := e2nodeSetReconciler.Reconcile(ctx, e2nsReq)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying initial E2NodeSet state")
			Eventually(func() int32 {
				updated := &nephoranv1.E2NodeSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated); err != nil {
					return 0
				}
				return updated.Status.ReadyReplicas
			}, timeout, interval).Should(Equal(int32(2)))

			By("Creating a NetworkIntent to scale the E2NodeSet")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("scale-intent"),
				namespaceName,
				fmt.Sprintf("Scale E2NodeSet %s to 4 replicas for performance", e2nodeSet.Name),
			)

			// Set up NetworkIntent reconciler with mock that returns scaling parameters
			networkIntentReconciler := &NetworkIntentReconciler{
				Client:        k8sClient,
				Scheme:        testEnv.Scheme,
				EventRecorder: &record.FakeRecorder{},
				MaxRetries:    3,
				RetryDelay:    time.Second * 1,
			}

			mockResponse := map[string]interface{}{
				"action":      "scale",
				"target_type": "E2NodeSet",
				"target_name": e2nodeSet.Name,
				"replicas":    4,
				"namespace":   namespaceName,
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			networkIntentReconciler.LLMClient = &MockLLMClient{
				Response: string(mockResponseBytes),
				Error:    nil,
			}
			networkIntentReconciler.GitClient = &MockGitClient{
				CommitHash: "integration-commit-789",
				Error:      nil,
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Processing NetworkIntent")
			niReq := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				},
			}

			result, err = networkIntentReconciler.Reconcile(ctx, niReq)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NetworkIntent processing completed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Processed") &&
					isConditionTrue(updated.Status.Conditions, "Deployed")
			}, timeout, interval).Should(BeTrue())

			By("Simulating manual E2NodeSet scaling based on NetworkIntent")
			// In a real implementation, this would be done by a GitOps system
			// For integration testing, we manually apply the scaling
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), e2nodeSet)).To(Succeed())
			e2nodeSet.Spec.Replicas = 4
			Expect(k8sClient.Update(ctx, e2nodeSet)).To(Succeed())

			By("Re-reconciling E2NodeSet after scaling")
			result, err = e2nodeSetReconciler.Reconcile(ctx, e2nsReq)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying E2NodeSet scaled according to NetworkIntent")
			Eventually(func() int32 {
				updated := &nephoranv1.E2NodeSet{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated); err != nil {
					return 0
				}
				return updated.Status.ReadyReplicas
			}, timeout, interval).Should(Equal(int32(4)))

			By("Verifying ConfigMaps match new replica count")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOpts := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOpts...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(4))
		})
	})

	Context("Multi-Resource Workflow Validation", func() {
		It("Should handle complex multi-resource deployments", func() {
			By("Creating multiple ManagedElements")
			managedElements := []*nephoranv1.ManagedElement{}
			elementTypes := []string{"O-CU", "O-DU", "Near-RT-RIC"}

			for i, elementType := range elementTypes {
				me := &nephoranv1.ManagedElement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GetUniqueName(fmt.Sprintf("me-%d", i)),
						Namespace: namespaceName,
						Labels: map[string]string{
							"test-resource": "true",
							"element-type":  elementType,
						},
					},
					Spec: nephoranv1.ManagedElementSpec{
						DeploymentName: fmt.Sprintf("%s-deployment", elementType),
						O1Config:       fmt.Sprintf("%s-o1-configuration", elementType),
						A1Policy: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"type": "%s", "id": %d}`, elementType, i)),
						},
					},
				}
				Expect(k8sClient.Create(ctx, me)).To(Succeed())
				managedElements = append(managedElements, me)
			}

			By("Creating E2NodeSets associated with ManagedElements")
			e2nodeSets := []*nephoranv1.E2NodeSet{}
			for i, me := range managedElements {
				e2ns := CreateTestE2NodeSet(
					GetUniqueName(fmt.Sprintf("e2ns-%d", i)),
					namespaceName,
					2, // Start with 2 replicas each
				)
				// Add reference to ManagedElement
				if e2ns.Labels == nil {
					e2ns.Labels = make(map[string]string)
				}
				e2ns.Labels["managed-element"] = me.Name

				Expect(k8sClient.Create(ctx, e2ns)).To(Succeed())
				e2nodeSets = append(e2nodeSets, e2ns)
			}

			By("Creating NetworkIntents for different operations")
			networkIntents := []*nephoranv1.NetworkIntent{}

			// Scale intent
			scaleIntent := CreateTestNetworkIntent(
				GetUniqueName("multi-scale-intent"),
				namespaceName,
				"Scale all O-CU E2NodeSets to 3 replicas",
			)
			scaleIntent.Labels["operation"] = "scale"
			scaleIntent.Labels["target-type"] = "O-CU"
			Expect(k8sClient.Create(ctx, scaleIntent)).To(Succeed())
			networkIntents = append(networkIntents, scaleIntent)

			// Deploy intent
			deployIntent := CreateTestNetworkIntent(
				GetUniqueName("multi-deploy-intent"),
				namespaceName,
				"Deploy additional O-RU components",
			)
			deployIntent.Labels["operation"] = "deploy"
			deployIntent.Labels["target-type"] = "O-RU"
			Expect(k8sClient.Create(ctx, deployIntent)).To(Succeed())
			networkIntents = append(networkIntents, deployIntent)

			By("Verifying all resources were created successfully")
			for _, me := range managedElements {
				created := &nephoranv1.ManagedElement{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(me), created)).To(Succeed())
			}

			for _, e2ns := range e2nodeSets {
				created := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2ns), created)).To(Succeed())
			}

			for _, ni := range networkIntents {
				created := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(ni), created)).To(Succeed())
			}

			By("Verifying resource relationships are maintained")
			for i, e2ns := range e2nodeSets {
				created := &nephoranv1.E2NodeSet{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2ns), created)).To(Succeed())
				Expect(created.Labels["managed-element"]).To(Equal(managedElements[i].Name))
			}

			By("Verifying resource counts and distribution")
			// Verify we have the expected number of each resource type
			meList := &nephoranv1.ManagedElementList{}
			Expect(k8sClient.List(ctx, meList, client.InNamespace(namespaceName))).To(Succeed())
			Expect(len(meList.Items)).To(Equal(3))

			e2nsList := &nephoranv1.E2NodeSetList{}
			Expect(k8sClient.List(ctx, e2nsList, client.InNamespace(namespaceName))).To(Succeed())
			Expect(len(e2nsList.Items)).To(Equal(3))

			niList := &nephoranv1.NetworkIntentList{}
			Expect(k8sClient.List(ctx, niList, client.InNamespace(namespaceName))).To(Succeed())
			Expect(len(niList.Items)).To(Equal(2))
		})
	})
})

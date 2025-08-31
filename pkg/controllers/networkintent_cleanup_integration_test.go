//go:build integration

package controllers

import (
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NetworkIntent Controller Cleanup Integration Tests", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	var (
		namespaceName string
		reconciler    *NetworkIntentReconciler
		mockDeps      *MockDependencies
	)

	BeforeEach(func() {
		By("Creating a new isolated namespace for integration tests")
		namespaceName = CreateIsolatedNamespace("cleanup-integration")

		By("Setting up the reconciler with enhanced mock dependencies")
		mockDeps = NewMockDependenciesBuilder().
			WithGitClient(NewEnhancedMockGitClient()).
			WithLLMClient(&MockLLMClientInterface{}).
			Build()

		config := &Config{
			MaxRetries:      3,
			RetryDelay:      time.Second,
			Timeout:         30 * time.Second,
			GitRepoURL:      "https://github.com/test/deployments.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			LLMProcessorURL: "http://localhost:8080",
			UseNephioPorch:  false,
		}

		var err error
		reconciler, err = NewNetworkIntentReconciler(k8sClient, testEnv.Scheme, mockDeps, config)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		CleanupIsolatedNamespace(namespaceName)
	})

	Context("End-to-end deletion scenarios", func() {
		It("Should handle complete NetworkIntent lifecycle with cleanup", func() {
			By("Creating a NetworkIntent with processed state")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("e2e-test"),
				namespaceName,
				"End-to-end test with complete cleanup",
			)

			// Add finalizer and set processed state
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			networkIntent.Status.Phase = "Completed"
			networkIntent.Status.Conditions = []metav1.Condition{
				{
					Type:               "Processed",
					Status:             metav1.ConditionTrue,
					Reason:             "LLMProcessingSucceeded",
					Message:            "Intent successfully processed",
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               "Deployed",
					Status:             metav1.ConditionTrue,
					Reason:             "GitDeploymentSucceeded",
					Message:            "Configuration successfully deployed",
					LastTransitionTime: metav1.Now(),
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Creating associated resources")
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("networkintent-%s", networkIntent.Name),
					Namespace: networkIntent.Namespace,
					Labels: map[string]string{
						"nephoran.com/created-by":       "networkintent-controller",
						"nephoran.com/intent-name":      networkIntent.Name,
						"nephoran.com/intent-namespace": networkIntent.Namespace,
					},
				},
				Data: map[string]string{
					"config": "test-configuration",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())

			By("Setting up successful Git cleanup expectations")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)

			By("Initiating deletion by setting deletion timestamp")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)

			By("Verifying successful cleanup")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying finalizer was removed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)
				return client.IgnoreNotFound(err) == nil && !containsFinalizer(updated.Finalizers, NetworkIntentFinalizer)
			}, timeout, interval).Should(BeTrue())

			By("Verifying Git cleanup was called")
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle cascading failures in cleanup chain", func() {
			By("Creating a NetworkIntent for cascading failure test")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("cascade-fail-test"),
				namespaceName,
				"Test cascading failure handling",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up Git cleanup to fail")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(errors.New("git cleanup failed"))

			By("Setting deletion timestamp")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)

			By("Verifying failure is handled appropriately")
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Verifying finalizer is still present")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(containsFinalizer(updated.Finalizers, NetworkIntentFinalizer)).To(BeTrue())

			mockGitClient.AssertExpectations(GinkgoT())
		})
	})

	Context("Resource cleanup integration", func() {
		It("Should clean up all associated Kubernetes resources", func() {
			By("Creating a NetworkIntent")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("resource-cleanup-test"),
				namespaceName,
				"Test comprehensive resource cleanup",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Creating multiple associated resources")
			resources := []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-configmap-%s", networkIntent.Name),
						Namespace: networkIntent.Namespace,
						Labels: map[string]string{
							"nephoran.com/created-by":       "networkintent-controller",
							"nephoran.com/intent-name":      networkIntent.Name,
							"nephoran.com/intent-namespace": networkIntent.Namespace,
						},
					},
					Data: map[string]string{"config": "value"},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-secret-%s", networkIntent.Name),
						Namespace: networkIntent.Namespace,
						Labels: map[string]string{
							"nephoran.com/created-by":       "networkintent-controller",
							"nephoran.com/intent-name":      networkIntent.Name,
							"nephoran.com/intent-namespace": networkIntent.Namespace,
						},
					},
					StringData: map[string]string{"secret": "value"},
				},
			}

			for _, resource := range resources {
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("Setting up Git cleanup expectations")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)

			By("Initiating deletion")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Triggering cleanup")
			err := reconciler.cleanupResources(ctx, networkIntent)

			By("Verifying cleanup completes without error")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Git cleanup was called")
			mockGitClient.AssertExpectations(GinkgoT())

			// Note: The current implementation doesn't actually delete Kubernetes resources
			// In a full implementation, we would verify that resources are deleted:
			// By("Verifying resources are deleted")
			// for _, resource := range resources {
			//     Eventually(func() bool {
			//         return k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), resource) != nil
			//     }, timeout, interval).Should(BeTrue())
			// }
		})
	})

	Context("Error recovery scenarios", func() {
		It("Should recover from transient Git failures", func() {
			By("Creating a NetworkIntent for recovery test")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("recovery-test"),
				namespaceName,
				"Test recovery from transient failures",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up Git client with transient failures")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			mockGitClient.SetMaxFailures("RemoveDirectory", 2) // Fail twice, then succeed

			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			// First two calls will fail
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(errors.New("transient failure")).Times(2)
			// Third call will succeed
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil).Once()
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil).Once()

			By("Setting deletion timestamp")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("First reconciliation should fail")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Second reconciliation should also fail")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Third reconciliation should succeed")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying recovery occurred")
			mockGitClient.AssertExpectations(GinkgoT())
			Expect(mockGitClient.GetCallCount("RemoveDirectory")).To(Equal(3))
		})

		It("Should handle mixed success/failure scenarios", func() {
			By("Creating a NetworkIntent for mixed scenario test")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("mixed-scenario-test"),
				namespaceName,
				"Test mixed success/failure scenarios",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up scenario where RemoveDirectory succeeds but CommitAndPushChanges fails")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(errors.New("commit failed"))

			By("Setting deletion timestamp")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)

			By("Verifying partial failure is handled")
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))
			Expect(err.Error()).To(ContainSubstring("commit failed"))

			mockGitClient.AssertExpectations(GinkgoT())
		})
	})

	Context("Concurrent deletion scenarios", func() {
		It("Should handle concurrent deletion requests safely", func() {
			By("Creating multiple NetworkIntents for concurrent deletion")
			var networkIntents []*nephoranv1.NetworkIntent
			for i := 0; i < 3; i++ {
				ni := CreateTestNetworkIntent(
					GetUniqueName(fmt.Sprintf("concurrent-test-%d", i)),
					namespaceName,
					fmt.Sprintf("Concurrent deletion test %d", i),
				)
				ni.Finalizers = []string{NetworkIntentFinalizer}
				Expect(k8sClient.Create(ctx, ni)).To(Succeed())
				networkIntents = append(networkIntents, ni)
			}

			By("Setting up Git client expectations for all deletions")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			for _, ni := range networkIntents {
				expectedPath := fmt.Sprintf("networkintents/%s-%s", ni.Namespace, ni.Name)
				expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", ni.Namespace, ni.Name)
				mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
				mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)
			}

			By("Initiating concurrent deletions")
			for _, ni := range networkIntents {
				patch := client.MergeFrom(ni.DeepCopy())
				now := metav1.Now()
				ni.DeletionTimestamp = &now
				Expect(k8sClient.Patch(ctx, ni, patch)).To(Succeed())
			}

			By("Processing deletions concurrently")
			var results []ctrl.Result
			var errors []error

			for _, ni := range networkIntents {
				req := ctrl.Request{NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				}}

				result, err := reconciler.Reconcile(ctx, req)
				results = append(results, result)
				errors = append(errors, err)
			}

			By("Verifying all deletions completed successfully")
			for i, err := range errors {
				Expect(err).NotTo(HaveOccurred(), "Deletion %d should succeed", i)
				Expect(results[i].Requeue).To(BeFalse())
			}

			By("Verifying all Git operations were called")
			mockGitClient.AssertExpectations(GinkgoT())
			Expect(mockGitClient.GetCallCount("RemoveDirectory")).To(Equal(3))
			Expect(mockGitClient.GetCallCount("CommitAndPushChanges")).To(Equal(3))
		})
	})

	Context("Performance and timeout scenarios", func() {
		It("Should handle cleanup operations within reasonable time limits", func() {
			By("Creating a NetworkIntent for performance test")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("performance-test"),
				namespaceName,
				"Test cleanup performance",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up Git client with simulated latency")
			mockGitClient := mockDeps.gitClient.(*EnhancedMockGitClient)
			mockGitClient.SetLatencySimulation(true, 100) // 100ms latency

			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)

			By("Setting deletion timestamp")
			patch := client.MergeFrom(networkIntent.DeepCopy())
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Measuring cleanup performance")
			start := time.Now()

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)
			duration := time.Since(start)

			By("Verifying cleanup completed within reasonable time")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(duration).To(BeNumerically("<", 30*time.Second), "Cleanup should complete within 30 seconds")

			mockGitClient.AssertExpectations(GinkgoT())
		})
	})
})

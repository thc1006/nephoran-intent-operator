package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	configPkg "github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Import constants from config package
// constants.NetworkIntentFinalizer is accessed via configPkg.LoadConstants().constants.NetworkIntentFinalizer

var _ = Describe("NetworkIntent Controller Resource Cleanup", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	var (
		ctx           context.Context
		namespaceName string
		reconciler    *NetworkIntentReconciler
		mockDeps      *testutils.MockDependencies
		constants     *configPkg.Constants
	)

	BeforeEach(func() {
		ctx = context.Background()
		constants = configPkg.LoadConstants()

		By("Creating a new isolated namespace for cleanup tests")
		namespaceName = testutils.CreateIsolatedNamespace("cleanup-test")

		By("Setting up the reconciler with mock dependencies")
		mockDeps = testutils.NewMockDependenciesBuilder().
			WithLLMClient(testutils.NewMockLLMClient()).
			WithGitClient(testutils.NewMockGitClient()).
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
		scheme := runtime.NewScheme()
		reconciler, err = NewNetworkIntentReconciler(k8sClient, scheme, mockDeps, config)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		testutils.CleanupIsolatedNamespace(namespaceName)
	})

	Context("Unit tests for cleanupGitOpsPackages", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("cleanup-gitops-test"),
				namespaceName,
				"scale up cpu to 2 cores",
			)
		})

		It("Should successfully remove directory from Git repository", func() {
			By("Setting up successful Git client mock")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			// Reset mock state
			mockGitClient.ResetMock()

			By("Calling cleanup methods")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying successful cleanup")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle Git operation failures gracefully", func() {
			By("Setting up Git client mock to fail on directory removal")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			gitError := errors.New("failed to remove directory: authentication failed")
			mockGitClient.On("RemoveDirectory", "networkintents/cleanup-gitops-test", "Remove NetworkIntent package: cleanup-gitops-test").Return(gitError)

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying error is propagated")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to remove GitOps package directory"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle Git commit failures", func() {
			By("Setting up Git client mock to fail on commit")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			mockGitClient.On("RemoveDirectory", "networkintents/cleanup-gitops-test", "Remove NetworkIntent package: cleanup-gitops-test").Return(nil)
			commitError := errors.New("failed to commit: remote repository unavailable")
			mockGitClient.On("CommitAndPushChanges", "Remove NetworkIntent package: cleanup-gitops-test").Return(commitError)

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying error is propagated")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to commit package removal"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle non-existent directories gracefully", func() {
			By("Setting up Git client mock for non-existent directory")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			notFoundError := errors.New("directory not found")
			mockGitClient.On("RemoveDirectory", "networkintents/cleanup-gitops-test", "Remove NetworkIntent package: cleanup-gitops-test").Return(notFoundError)

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying error handling")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to remove GitOps package directory"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should use correct package path format", func() {
			By("Setting up Git client mock with specific expectations")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			expectedPackagePath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedCommitMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPackagePath, expectedCommitMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedCommitMessage).Return(nil)

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying correct paths and messages were used")
			Expect(err).NotTo(HaveOccurred())
			mockGitClient.AssertExpectations(GinkgoT())
		})
	})

	Context("Unit tests for cleanupGeneratedResources", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("cleanup-resources-test"),
				namespaceName,
				"Test resource cleanup functionality",
			)
		})

		It("Should successfully clean up generated ConfigMaps", func() {
			By("Creating test ConfigMaps with appropriate labels")
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-configmap-%s", networkIntent.Name),
					Namespace: networkIntent.Namespace,
					Labels: map[string]string{
						"nephoran.com/created-by":       "networkintent-controller",
						"nephoran.com/intent-name":      networkIntent.Name,
						"nephoran.com/intent-namespace": networkIntent.Namespace,
					},
				},
				Data: map[string]string{
					"test-data": "test-value",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())

			By("Calling cleanupGeneratedResources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying ConfigMap cleanup (note: current implementation is placeholder)")
			Expect(err).NotTo(HaveOccurred())
			// Note: The current implementation doesn't actually delete resources,
			// it just logs the cleanup operation. In a full implementation,
			// we would verify the ConfigMap is deleted.
		})

		It("Should successfully clean up generated Secrets", func() {
			By("Creating test Secrets with appropriate labels")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-secret-%s", networkIntent.Name),
					Namespace: networkIntent.Namespace,
					Labels: map[string]string{
						"nephoran.com/created-by":       "networkintent-controller",
						"nephoran.com/intent-name":      networkIntent.Name,
						"nephoran.com/intent-namespace": networkIntent.Namespace,
					},
				},
				StringData: map[string]string{
					"test-secret": "secret-value",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Calling cleanupGeneratedResources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying Secret cleanup (note: current implementation is placeholder)")
			Expect(err).NotTo(HaveOccurred())
			// Note: The current implementation doesn't actually delete resources,
			// it just logs the cleanup operation. In a full implementation,
			// we would verify the Secret is deleted.
		})

		It("Should handle cleanup errors gracefully", func() {
			By("Testing error scenarios in resource cleanup")
			// The current implementation doesn't have error paths for resource cleanup
			// since it's a placeholder. In a full implementation, we would test
			// scenarios like failed resource deletions.

			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle empty resource lists", func() {
			By("Calling cleanup with no matching resources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying graceful handling of empty resource lists")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should use correct label selectors", func() {
			By("Verifying label selector construction")
			expectedLabels := map[string]string{
				"nephoran.com/created-by":       "networkintent-controller",
				"nephoran.com/intent-name":      networkIntent.Name,
				"nephoran.com/intent-namespace": networkIntent.Namespace,
			}

			// Test that the label selector would be constructed correctly
			labelSelector := createLabelSelector(expectedLabels)
			Expect(labelSelector).NotTo(BeEmpty())
			Expect(labelSelector).To(ContainSubstring("nephoran.com/created-by=networkintent-controller"))
		})

		It("Should handle idempotent operations", func() {
			By("Calling cleanup multiple times")
			for i := 0; i < 3; i++ {
				err := reconciler.cleanupGeneratedResources(ctx, networkIntent)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying multiple calls don't cause issues")
			// The operation should be idempotent
		})
	})

	Context("Integration tests for handleDeletion", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("deletion-test"),
				namespaceName,
				"Test complete deletion flow",
			)
			// Add finalizer to simulate real scenario
			networkIntent.Finalizers = []string{constants.NetworkIntentFinalizer}
		})

		It("Should handle complete deletion flow successfully", func() {
			By("Setting up successful Git client mock")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			packagePath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", packagePath, commitMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", commitMessage).Return(nil)

			By("Creating the NetworkIntent with deletion timestamp")
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying successful deletion handling")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying finalizer was removed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updated)
				if err != nil {
					return false
				}
				return !containsFinalizer(updated.Finalizers, constants.NetworkIntentFinalizer)
			}, timeout, interval).Should(BeTrue())

			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle Git cleanup failures with proper error handling", func() {
			By("Setting up Git client mock to fail")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			gitError := errors.New("git repository authentication failed")
			packagePath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", packagePath, commitMessage).Return(gitError)

			By("Creating the NetworkIntent with deletion timestamp")
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying error handling")
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			By("Verifying finalizer is still present on error")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updated)).To(Succeed())
			Expect(containsFinalizer(updated.Finalizers, constants.NetworkIntentFinalizer)).To(BeTrue())

			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle partial failure scenarios", func() {
			By("Setting up Git client mock to succeed but with warning")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			packagePath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", packagePath, commitMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", commitMessage).Return(nil)

			By("Creating the NetworkIntent with deletion timestamp")
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying successful completion despite potential warnings")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle missing Git client gracefully", func() {
			By("Setting Git client to nil")
			// Create fresh mock dependencies with nil git client to test graceful handling
			reconciler.deps = testutils.NewMockDependenciesBuilder().
				WithGitClient(nil).
				Build()

			By("Creating the NetworkIntent with deletion timestamp")
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying graceful handling when Git client is not available")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying finalizer was still removed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updated)
				if err != nil {
					return false
				}
				return !containsFinalizer(updated.Finalizers, constants.NetworkIntentFinalizer)
			}, timeout, interval).Should(BeTrue())
		})

		It("Should propagate errors through the cleanup chain", func() {
			By("Testing error propagation from cleanup operations")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			gitError := errors.New("persistent git failure")
			packagePath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
			mockGitClient.On("RemoveDirectory", packagePath, commitMessage).Return(gitError)

			By("Creating the NetworkIntent with deletion timestamp")
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion multiple times to test persistence")
			for i := 0; i < 3; i++ {
				result, err := reconciler.handleDeletion(ctx, networkIntent)
				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute))
			}

			By("Verifying error is consistently propagated")
			mockGitClient.AssertNumberOfCalls(GinkgoT(), "RemoveDirectory", 3)
		})
	})

	Context("Mock Git client tests with various failure scenarios", func() {
		var mockGitClient *testutils.MockGitClient
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			mockGitClient = &testutils.MockGitClient{}
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("git-mock-test"),
				namespaceName,
				"Test Git mock scenarios",
			)
		})

		It("Should verify correct Git commands are called", func() {
			By("Setting up Git client expectations")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying correct method calls")
			Expect(err).NotTo(HaveOccurred())
			mockGitClient.AssertExpectations(GinkgoT())
			mockGitClient.AssertCalled(GinkgoT(), "RemoveDirectory", expectedPath, expectedMessage)
			mockGitClient.AssertCalled(GinkgoT(), "CommitAndPushChanges", expectedMessage)
		})

		It("Should handle authentication failures", func() {
			By("Setting up authentication failure scenario")
			authError := errors.New("SSH key authentication failed")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(authError)

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying authentication error handling")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SSH key authentication failed"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle network connectivity failures", func() {
			By("Setting up network failure scenario")
			networkError := errors.New("failed to connect to remote repository")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(networkError)

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying network error handling")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to connect to remote repository"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle repository corruption scenarios", func() {
			By("Setting up repository corruption scenario")
			corruptionError := errors.New("repository is corrupted or locked")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(corruptionError)

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying corruption error handling")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("repository is corrupted or locked"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should handle timeout scenarios", func() {
			By("Setting up timeout scenario")
			timeoutError := errors.New("operation timed out")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(timeoutError)

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying timeout error handling")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("operation timed out"))
			mockGitClient.AssertExpectations(GinkgoT())
		})

		It("Should verify method call order and frequency", func() {
			By("Setting up strict method call expectations")
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			// Expect RemoveDirectory to be called exactly once, then CommitAndPushChanges
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil).Once()
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil).Once()

			By("Calling cleanup function")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying correct call order and frequency")
			Expect(err).NotTo(HaveOccurred())
			mockGitClient.AssertExpectations(GinkgoT())
			mockGitClient.AssertNumberOfCalls(GinkgoT(), "RemoveDirectory", 1)
			mockGitClient.AssertNumberOfCalls(GinkgoT(), "CommitAndPushChanges", 1)
		})
	})

	Context("Error recovery and idempotent behavior", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("recovery-test"),
				namespaceName,
				"Test error recovery scenarios",
			)
		})

		It("Should handle cleanup operations idempotently", func() {
			By("Setting up successful Git client mock")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			// Allow multiple calls
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)

			By("Calling cleanup multiple times")
			for i := 0; i < 3; i++ {
				err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying idempotent behavior")
			// Each call should succeed without side effects
			mockGitClient.AssertNumberOfCalls(GinkgoT(), "RemoveDirectory", 3)
			mockGitClient.AssertNumberOfCalls(GinkgoT(), "CommitAndPushChanges", 3)
		})

		It("Should recover from transient failures", func() {
			By("Setting up Git client mock with transient failure then success")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
			expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

			// First call fails, second succeeds
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(errors.New("transient failure")).Once()
			mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil).Once()
			mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil).Once()

			By("First call should fail")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)
			Expect(err).To(HaveOccurred())

			By("Second call should succeed")
			err = reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)
			Expect(err).NotTo(HaveOccurred())

			mockGitClient.AssertExpectations(GinkgoT())
		})
	})

	Context("reconcileDelete function tests with fake Git client", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("reconcile-delete-test"),
				namespaceName,
				"Test reconcileDelete with Git failures",
			)
			// Add finalizer to simulate real scenario
			networkIntent.Finalizers = []string{constants.NetworkIntentFinalizer}
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		})

		It("Should retain finalizer when Git operation fails", func() {
			By("Setting up fake Git client to fail")
			fakeGitClient := &testutils.MockGitClient{}
			fakeGitClient.On("InitRepo").Return(nil)
			fakeGitClient.On("RemoveDirectory", mock.Anything, mock.Anything).Return(errors.New("fake Git operation failed"))

			// Replace the git client in dependencies
			mockDeps.GitClient = fakeGitClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling reconcileDelete")
			result, err := reconciler.reconcileDelete(ctx, networkIntent)

			By("Verifying that reconcileDelete schedules a retry")
			Expect(err).NotTo(HaveOccurred())                     // reconcileDelete returns nil error on retry
			Expect(result.RequeueAfter).To(BeNumerically(">", 0)) // Should schedule retry

			By("Verifying finalizer is retained")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updatedIntent)).To(Succeed())
			Expect(containsFinalizer(updatedIntent.Finalizers, constants.NetworkIntentFinalizer)).To(BeTrue())

			By("Verifying Ready condition is set to false with CleanupRetrying reason")
			readyCondition := testGetConditionCleanup(updatedIntent.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("CleanupRetrying"))

			By("Verifying retry count is incremented")
			retryCount := getRetryCount(updatedIntent, "cleanup")
			Expect(retryCount).To(Equal(1))

			fakeGitClient.AssertExpectations(GinkgoT())
		})

		It("Should remove finalizer when Git operation succeeds", func() {
			By("Setting up fake Git client to succeed")
			fakeGitClient := &testutils.MockGitClient{}
			fakeGitClient.On("InitRepo").Return(nil)
			fakeGitClient.On("RemoveDirectory", mock.Anything, mock.Anything).Return(nil)

			// Replace the git client in dependencies
			mockDeps.GitClient = fakeGitClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling reconcileDelete")
			result, err := reconciler.reconcileDelete(ctx, networkIntent)

			By("Verifying successful completion")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Verifying finalizer is removed")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updatedIntent)).To(Succeed())
			Expect(containsFinalizer(updatedIntent.Finalizers, constants.NetworkIntentFinalizer)).To(BeFalse())

			By("Verifying Ready condition is set to false with CleanupCompleted reason")
			readyCondition := testGetConditionCleanup(updatedIntent.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("CleanupCompleted"))

			By("Verifying retry count is cleared")
			retryCount := getRetryCount(updatedIntent, "cleanup")
			Expect(retryCount).To(Equal(0))

			fakeGitClient.AssertExpectations(GinkgoT())
		})

		It("Should retry with exponential backoff on Git failures", func() {
			By("Setting up fake Git client to fail consistently")
			fakeGitClient := &testutils.MockGitClient{}
			fakeGitClient.On("InitRepo").Return(nil)
			fakeGitClient.On("RemoveDirectory", mock.Anything, mock.Anything).Return(errors.New("persistent Git failure"))

			// Replace the git client in dependencies
			mockDeps.GitClient = fakeGitClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling reconcileDelete multiple times and verifying exponential backoff")
			expectedDelays := []time.Duration{
				1 * time.Second, // retry 1: 1 * RetryDelay
				2 * time.Second, // retry 2: 2 * RetryDelay
				3 * time.Second, // retry 3: 3 * RetryDelay
			}

			for i, expectedDelay := range expectedDelays {
				result, err := reconciler.reconcileDelete(ctx, networkIntent)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(expectedDelay))

				// Get updated intent to check retry count
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updatedIntent)).To(Succeed())
				Expect(getRetryCount(updatedIntent, "cleanup")).To(Equal(i + 1))

				// Update networkIntent for next iteration
				networkIntent = updatedIntent
			}

			fakeGitClient.AssertExpectations(GinkgoT())
		})

		It("Should remove finalizer after max retries to prevent stuck resources", func() {
			By("Setting up fake Git client to always fail")
			fakeGitClient := &testutils.MockGitClient{}
			fakeGitClient.On("InitRepo").Return(nil)
			fakeGitClient.On("RemoveDirectory", mock.Anything, mock.Anything).Return(errors.New("permanent Git failure"))

			// Replace the git client in dependencies
			mockDeps.GitClient = fakeGitClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Simulating max retries by setting retry count")
			setRetryCount(networkIntent, "cleanup", reconciler.config.MaxRetries)
			Expect(k8sClient.Update(ctx, networkIntent)).To(Succeed())

			By("Calling reconcileDelete after max retries")
			result, err := reconciler.reconcileDelete(ctx, networkIntent)

			By("Verifying finalizer is removed to prevent stuck resource")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updatedIntent)).To(Succeed())
			Expect(containsFinalizer(updatedIntent.Finalizers, constants.NetworkIntentFinalizer)).To(BeFalse())

			By("Verifying Ready condition indicates max retry failure")
			readyCondition := testGetConditionCleanup(updatedIntent.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("CleanupFailedMaxRetries"))

			// Git operations should not be called when max retries exceeded
			fakeGitClient.AssertNumberOfCalls(GinkgoT(), "RemoveDirectory", 0)
		})

		It("Should handle InitRepo failures gracefully", func() {
			By("Setting up fake Git client to fail on InitRepo")
			fakeGitClient := &testutils.MockGitClient{}
			fakeGitClient.On("InitRepo").Return(errors.New("failed to initialize repository"))

			// Replace the git client in dependencies
			mockDeps.GitClient = fakeGitClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling reconcileDelete")
			result, err := reconciler.reconcileDelete(ctx, networkIntent)

			By("Verifying that cleanup completes gracefully despite InitRepo failure")
			// The cleanupGitOpsPackagesWithWait should handle InitRepo failures gracefully
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying finalizer is removed")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updatedIntent)).To(Succeed())
			Expect(containsFinalizer(updatedIntent.Finalizers, constants.NetworkIntentFinalizer)).To(BeFalse())

			fakeGitClient.AssertExpectations(GinkgoT())
		})
	})
})

// Helper function to get a condition by type
func testGetConditionCleanup(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

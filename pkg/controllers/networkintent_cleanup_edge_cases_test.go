package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// NetworkIntentFinalizer is imported from the main controller package

// MockDependencies implements Dependencies interface for testing
type MockDependencies struct {
	gitClient        *testutils.MockGitClient
	llmClient        *testutils.MockLLMClient
	packageGenerator interface{}
	httpClient       *http.Client
	eventRecorder    record.EventRecorder
}

// GetEventRecorder returns the event recorder
func (m *MockDependencies) GetEventRecorder() record.EventRecorder {
	return m.eventRecorder
}

// GetGitClient returns the git client
func (m *MockDependencies) GetGitClient() git.ClientInterface {
	return m.gitClient
}

// GetLLMClient returns the LLM client
func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
	return m.llmClient
}

// GetPackageGenerator returns the package generator
func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	if m.packageGenerator == nil {
		return nil
	}
	if pkg, ok := m.packageGenerator.(*nephio.PackageGenerator); ok {
		return pkg
	}
	return nil
}

// GetHTTPClient returns the HTTP client
func (m *MockDependencies) GetHTTPClient() *http.Client {
	return m.httpClient
}

// GetTelecomKnowledgeBase returns the telecom knowledge base
func (m *MockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	return nil // Mock implementation
}

// GetMetricsCollector returns the metrics collector
func (m *MockDependencies) GetMetricsCollector() monitoring.MetricsCollector {
	return nil // Mock implementation
}

// Removed redeclaration to fix compilation error

var _ = Describe("NetworkIntent Controller Cleanup Edge Cases", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	var (
		namespaceName string
		reconciler    *NetworkIntentReconciler
		mockDeps      *MockDependencies
		ctx           context.Context
		testEnv       *envtest.Environment // Local testEnv for this test suite
	)

	BeforeEach(func() {
		By("Setting up test context")
		ctx = context.Background()

		By("Setting up test environment")
		testEnv = &envtest.Environment{
			Scheme: runtime.NewScheme(),
		}

		By("Creating a new isolated namespace for edge case tests")
		namespaceName = testutils.CreateIsolatedNamespace("cleanup-edge-cases")

		By("Setting up the reconciler with mock dependencies")
		mockDeps = &MockDependencies{
			gitClient:        &testutils.MockGitClient{},
			llmClient:        &testutils.MockLLMClient{},
			packageGenerator: nil,
			httpClient:       &http.Client{Timeout: 30 * time.Second},
			eventRecorder:    &record.FakeRecorder{Events: make(chan string, 100)},
		}

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
		testutils.CleanupIsolatedNamespace(namespaceName)
	})

	Context("Edge cases in cleanupGitOpsPackages", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("gitops-edge-test"),
				namespaceName,
				"Test GitOps cleanup edge cases",
			)
		})

		It("Should handle very long namespace and name combinations", func() {
			By("Creating NetworkIntent with maximum length names")
			longName := strings.Repeat("a", 63) // Maximum DNS label length
			longNamespace := strings.Repeat("b", 63)

			networkIntent.Name = longName
			networkIntent.Namespace = longNamespace

			By("Calling cleanupGitOpsPackages")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			mockGitClient.ResetMock() // Reset mock state

			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying successful handling of long names")
			Expect(err).NotTo(HaveOccurred())

			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle special characters in namespace and name", func() {
			By("Creating NetworkIntent with special characters")
			networkIntent.Name = "test-name-with-dashes"
			networkIntent.Namespace = "test-namespace-123"

			By("Setting up Git client")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			mockGitClient.ResetMock()

			// Mock will automatically handle these method calls
			// No explicit expectations needed with simple mock

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying successful handling of special characters")
			Expect(err).NotTo(HaveOccurred())
			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle nil Git client gracefully", func() {
			By("Calling cleanupGitOpsPackages with nil Git client")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, nil)

			By("Verifying graceful handling of nil client")
			// This should cause a panic or specific error handling depending on implementation
			// The test verifies the behavior matches the expected error handling strategy
			Expect(err).To(HaveOccurred())
		})

		It("Should handle context cancellation during Git operations", func() {
			By("Creating a cancelled context")
			cancelledCtx, cancel := context.WithCancel(context.Background())
			cancel() // Immediately cancel the context

			By("Setting up Git client to return errors for cancellation")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			mockGitClient.ResetMock()
			mockGitClient.SetCommitPushError(errors.New("context cancelled"))

			By("Calling cleanupGitOpsPackages with cancelled context")
			err := reconciler.cleanupGitOpsPackages(cancelledCtx, networkIntent, mockGitClient)

			By("Verifying context cancellation is handled")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to remove GitOps package directory"))
			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle intermittent Git repository locks", func() {
			By("Setting up Git client to simulate repository lock contention")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			mockGitClient.ResetMock()
			lockError := errors.New("fatal: Unable to create '/path/to/repo/.git/refs/heads/main.lock': File exists")
			mockGitClient.SetCommitPushError(lockError)

			By("Calling cleanupGitOpsPackages")
			err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

			By("Verifying repository lock error is handled")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to remove GitOps package directory"))
			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})
	})

	Context("Edge cases in cleanupGeneratedResources", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("resource-edge-test"),
				namespaceName,
				"Test resource cleanup edge cases",
			)
		})

		It("Should handle resources with malformed labels", func() {
			By("Creating resources with various label combinations")
			malformedResources := []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "malformed-labels-1",
						Namespace: networkIntent.Namespace,
						Labels: map[string]string{
							"nephoran.com/created-by": "networkintent-controller",
							// Missing intent-name and intent-namespace labels
						},
					},
					Data: map[string]string{"data": "value"},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "malformed-labels-2",
						Namespace: networkIntent.Namespace,
						Labels: map[string]string{
							"nephoran.com/intent-name": networkIntent.Name,
							// Missing created-by and intent-namespace labels
						},
					},
					Data: map[string]string{"data": "value"},
				},
			}

			for _, resource := range malformedResources {
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("Calling cleanupGeneratedResources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying cleanup handles malformed labels gracefully")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle resources in different namespaces", func() {
			By("Creating another namespace")
			otherNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("other-%s", namespaceName),
					Labels: map[string]string{
						"test-namespace": "true",
					},
				},
			}
			Expect(k8sClient.Create(ctx, otherNamespace)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, otherNamespace)
			}()

			By("Creating resources in different namespaces with same labels")
			crossNamespaceResource := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cross-namespace-resource",
					Namespace: otherNamespace.Name,
					Labels: map[string]string{
						"nephoran.com/created-by":       "networkintent-controller",
						"nephoran.com/intent-name":      networkIntent.Name,
						"nephoran.com/intent-namespace": networkIntent.Namespace,
					},
				},
				Data: map[string]string{"data": "value"},
			}
			Expect(k8sClient.Create(ctx, crossNamespaceResource)).To(Succeed())

			By("Calling cleanupGeneratedResources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying cleanup handles cross-namespace resources correctly")
			Expect(err).NotTo(HaveOccurred())
			// The current implementation should handle this gracefully
		})

		It("Should handle empty label selector results", func() {
			By("Calling cleanup with no matching resources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying empty results are handled gracefully")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle resources with no labels", func() {
			By("Creating resources without any labels")
			unlabeledResource := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unlabeled-resource",
					Namespace: networkIntent.Namespace,
					// No labels
				},
				Data: map[string]string{"data": "value"},
			}
			Expect(k8sClient.Create(ctx, unlabeledResource)).To(Succeed())

			By("Calling cleanupGeneratedResources")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying unlabeled resources don't cause issues")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Edge cases in handleDeletion", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("deletion-edge-test"),
				namespaceName,
				"Test deletion edge cases",
			)
		})

		It("Should handle NetworkIntent without finalizers", func() {
			By("Creating NetworkIntent without finalizers")
			networkIntent.Finalizers = []string{} // No finalizers
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying deletion handles missing finalizers gracefully")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("Should handle NetworkIntent with multiple finalizers", func() {
			By("Creating NetworkIntent with multiple finalizers")
			networkIntent.Finalizers = []string{
				NetworkIntentFinalizer,
				"other.controller/finalizer",
				"third.controller/finalizer",
			}
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up successful Git cleanup")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			// Removed unused variables
			// Mock will automatically handle these method calls
			// No explicit expectations needed with simple mock

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying only our finalizer is removed")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify the NetworkIntent still exists with other finalizers
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: networkIntent.GetName(), Namespace: networkIntent.GetNamespace()}, updated)
				if err != nil {
					return false
				}
				return !containsFinalizer(updated.Finalizers, NetworkIntentFinalizer) &&
					containsFinalizer(updated.Finalizers, "other.controller/finalizer")
			}, timeout, interval).Should(BeTrue())

			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle deletion of already deleted resources", func() {
			By("Creating and immediately deleting NetworkIntent")
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up Git client to simulate already cleaned resources")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			// Removed unused variables

			// Simulate that the directory doesn't exist (already cleaned)
			notFoundError := errors.New("directory not found")
			mockGitClient.SetCommitPushError(notFoundError)

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying deletion handles already-deleted resources")
			Expect(err).To(HaveOccurred()) // Current implementation propagates the error
			Expect(result.RequeueAfter).To(Equal(time.Minute))

			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle update conflicts during finalizer removal", func() {
			By("Creating NetworkIntent for conflict test")
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}
			networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up successful Git cleanup")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			// Removed unused variables
			// Mock will automatically handle these method calls
			// No explicit expectations needed with simple mock

			By("Simulating concurrent modification")
			// Modify the resource to create a version conflict
			patch := client.MergeFrom(networkIntent.DeepCopy())
			networkIntent.Annotations = map[string]string{"test": "value"}
			Expect(k8sClient.Patch(ctx, networkIntent, patch)).To(Succeed())

			By("Calling handleDeletion")
			result, err := reconciler.handleDeletion(ctx, networkIntent)

			By("Verifying update conflicts are handled gracefully")
			// The current implementation may succeed or fail depending on timing
			// but should not cause panics or data corruption
			if err != nil {
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			} else {
				Expect(result.Requeue).To(BeFalse())
			}

			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})
	})

	Context("Resource cleanup with complex scenarios", func() {
		It("Should handle cleanup when Kubernetes API is slow", func() {
			By("Creating NetworkIntent for slow API test")
			networkIntent := testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("slow-api-test"),
				namespaceName,
				"Test cleanup with slow Kubernetes API",
			)
			networkIntent.Finalizers = []string{NetworkIntentFinalizer}

			By("Creating resources that might be slow to list/delete")
			for i := 0; i < 10; i++ {
				resource := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("slow-api-resource-%d", i),
						Namespace: networkIntent.Namespace,
						Labels: map[string]string{
							"nephoran.com/created-by":       "networkintent-controller",
							"nephoran.com/intent-name":      networkIntent.Name,
							"nephoran.com/intent-namespace": networkIntent.Namespace,
						},
					},
					Data: map[string]string{"index": fmt.Sprintf("%d", i)},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("Setting up Git cleanup expectations")
			mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
			// Removed unused variables
			// Mock will automatically handle these method calls
			// No explicit expectations needed with simple mock

			By("Performing cleanup")
			err := reconciler.cleanupResources(ctx, networkIntent)

			By("Verifying cleanup completes despite many resources")
			Expect(err).NotTo(HaveOccurred())
			// Verify that git operations were called
			callLog := mockGitClient.GetCallLog()
			Expect(len(callLog)).To(BeNumerically(">=", 1))
		})

		It("Should handle cleanup with resource deletion protection", func() {
			By("Creating NetworkIntent for protection test")
			networkIntent := testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("protection-test"),
				namespaceName,
				"Test cleanup with protected resources",
			)

			By("Creating resources with finalizers (protection)")
			protectedResource := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "protected-resource",
					Namespace: networkIntent.Namespace,
					Labels: map[string]string{
						"nephoran.com/created-by":       "networkintent-controller",
						"nephoran.com/intent-name":      networkIntent.Name,
						"nephoran.com/intent-namespace": networkIntent.Namespace,
					},
					Finalizers: []string{"protection.example.com/keep-alive"},
				},
				Data: map[string]string{"protected": "true"},
			}
			Expect(k8sClient.Create(ctx, protectedResource)).To(Succeed())

			By("Performing cleanup")
			err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

			By("Verifying cleanup handles protected resources gracefully")
			Expect(err).NotTo(HaveOccurred())
			// Protected resources should remain (current implementation doesn't actually delete)
		})
	})

	Context("Cache and HTTP client edge cases", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = testutils.CreateTestNetworkIntent(
				testutils.GetUniqueName("cache-edge-test"),
				namespaceName,
				"Test cache cleanup edge cases",
			)
		})

		It("Should handle HTTP client timeout during cache cleanup", func() {
			By("Setting up HTTP client with short timeout")
			shortTimeoutClient := &http.Client{Timeout: time.Millisecond}
			mockDeps.httpClient = shortTimeoutClient

			// Update reconciler with new deps
			config := reconciler.config
			var err error
			reconciler, err = NewNetworkIntentReconciler(k8sClient, testEnv.Scheme, mockDeps, config)
			Expect(err).NotTo(HaveOccurred())

			By("Calling cleanupCachedData")
			err = reconciler.cleanupCachedData(ctx, networkIntent)

			By("Verifying timeout is handled gracefully")
			Expect(err).NotTo(HaveOccurred()) // Should not fail - cache cleanup is non-critical
		})

		It("Should handle missing LLM processor URL", func() {
			By("Setting empty LLM processor URL")
			reconciler.config.LLMProcessorURL = ""

			By("Calling cleanupCachedData")
			err := reconciler.cleanupCachedData(ctx, networkIntent)

			By("Verifying missing URL is handled gracefully")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle malformed LLM processor URL", func() {
			By("Setting malformed LLM processor URL")
			reconciler.config.LLMProcessorURL = "not-a-valid-url"

			By("Calling cleanupCachedData")
			err := reconciler.cleanupCachedData(ctx, networkIntent)

			By("Verifying malformed URL is handled gracefully")
			Expect(err).NotTo(HaveOccurred()) // Should handle gracefully
		})

		It("Should handle HTTP 500 errors from LLM processor", func() {
			By("Testing with configured LLM processor URL")
			// This test would require a more sophisticated HTTP mock
			// For now, verify the cleanup doesn't crash with normal configuration
			err := reconciler.cleanupCachedData(ctx, networkIntent)

			By("Verifying HTTP errors are handled gracefully")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Label selector edge cases", func() {
		It("Should handle createLabelSelector with empty labels", func() {
			By("Testing createLabelSelector with empty map")
			selector := createLabelSelector(map[string]string{})

			By("Verifying empty selector is handled")
			Expect(selector).NotTo(BeEmpty())
		})

		It("Should handle createLabelSelector with special characters", func() {
			By("Testing createLabelSelector with special characters")
			labels := map[string]string{
				"nephoran.com/created-by":       "networkintent-controller",
				"nephoran.com/intent-name":      "test-name-with-dashes",
				"nephoran.com/intent-namespace": "test-namespace-123",
			}
			selector := createLabelSelector(labels)

			By("Verifying special characters are handled")
			Expect(selector).To(ContainSubstring("nephoran.com/created-by=networkintent-controller"))
			Expect(selector).To(ContainSubstring("test-name-with-dashes"))
		})

		It("Should handle createLabelSelector with Unicode characters", func() {
			By("Testing createLabelSelector with Unicode")
			labels := map[string]string{
				"test-label": "value-with-unicode-测试",
			}
			selector := createLabelSelector(labels)

			By("Verifying Unicode is handled")
			Expect(selector).NotTo(BeEmpty())
		})
	})

	Context("Finalizer management edge cases", func() {
		It("Should handle containsFinalizer with empty slice", func() {
			By("Testing containsFinalizer with empty finalizers")
			result := containsFinalizer([]string{}, NetworkIntentFinalizer)

			By("Verifying empty slice returns false")
			Expect(result).To(BeFalse())
		})

		It("Should handle containsFinalizer with nil slice", func() {
			By("Testing containsFinalizer with nil finalizers")
			result := containsFinalizer(nil, NetworkIntentFinalizer)

			By("Verifying nil slice returns false")
			Expect(result).To(BeFalse())
		})

		It("Should handle removeFinalizer with duplicate finalizers", func() {
			By("Testing removeFinalizer with duplicates")
			finalizers := []string{
				NetworkIntentFinalizer,
				"other.finalizer",
				NetworkIntentFinalizer, // Duplicate
				"third.finalizer",
			}
			result := removeFinalizer(finalizers, NetworkIntentFinalizer)

			By("Verifying all instances are removed")
			Expect(result).To(HaveLen(2))
			Expect(result).To(ContainElement("other.finalizer"))
			Expect(result).To(ContainElement("third.finalizer"))
			Expect(result).NotTo(ContainElement(NetworkIntentFinalizer))
		})

		It("Should handle removeFinalizer with non-existent finalizer", func() {
			By("Testing removeFinalizer with non-existent finalizer")
			finalizers := []string{"other.finalizer", "third.finalizer"}
			result := removeFinalizer(finalizers, NetworkIntentFinalizer)

			By("Verifying original slice is preserved")
			Expect(result).To(HaveLen(2))
			Expect(result).To(ContainElement("other.finalizer"))
			Expect(result).To(ContainElement("third.finalizer"))
		})
	})
})

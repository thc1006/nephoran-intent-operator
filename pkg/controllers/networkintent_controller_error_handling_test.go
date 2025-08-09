package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkIntent Controller Error Handling", func() {
	var (
		ctx           context.Context
		reconciler    *NetworkIntentReconciler
		fakeClient    client.Client
		scheme        *runtime.Scheme
		networkIntent *nephoranv1.NetworkIntent
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create reconciler with test configuration
		deps := &TestDependencies{
			llmClient: &MockLLMClient{},
			gitClient: &MockGitClient{},
		}

		config := &Config{
			MaxRetries:    3,
			RetryDelay:    time.Second,
			GitRepoURL:    "https://github.com/test/repo.git",
			GitBranch:     "main",
			GitDeployPath: "deployments",
		}

		var err error
		reconciler, err = NewNetworkIntentReconciler(fakeClient, scheme, deps, config)
		Expect(err).NotTo(HaveOccurred())

		reconciler.EventRecorder = &record.FakeRecorder{}

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-intent",
				Namespace:  "default",
				Finalizers: []string{NetworkIntentFinalizer},
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "Deploy 5G AMF with high availability",
			},
		}
	})

	Describe("Exponential Backoff Tests", func() {
		DescribeTable("calculateExponentialBackoff should calculate correct delays",
			func(retryCount int, baseDelay, maxDelay, expectedRange time.Duration) {
				result := calculateExponentialBackoff(retryCount, baseDelay, maxDelay)

				// Check that result is within expected range (considering jitter)
				expectedBase := time.Duration(float64(baseDelay.Nanoseconds()) * math.Pow(BackoffMultiplier, float64(retryCount)))
				if expectedBase > maxDelay {
					expectedBase = maxDelay
				}

				// Allow for jitter (up to 10% variation)
				minExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 0.9)
				maxExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 1.1)

				// For max delay cases, result should equal max delay
				if expectedRange == maxDelay {
					Expect(result).To(Equal(maxDelay))
				} else {
					Expect(result).To(BeNumerically(">=", minExpected))
					Expect(result).To(BeNumerically("<=", maxExpected))
				}
			},
			Entry("retry 0", 0, time.Second, time.Minute, time.Second),
			Entry("retry 1", 1, time.Second, time.Minute, 2*time.Second),
			Entry("retry 2", 2, time.Second, time.Minute, 4*time.Second),
			Entry("retry 3", 3, time.Second, time.Minute, 8*time.Second),
			Entry("retry 10 (max)", 10, time.Second, time.Minute, time.Minute),
		)

		It("should apply jitter correctly", func() {
			baseDelay := time.Second
			maxDelay := time.Minute
			retryCount := 2

			// Run multiple times to test jitter variation
			results := make([]time.Duration, 100)
			for i := 0; i < 100; i++ {
				results[i] = calculateExponentialBackoff(retryCount, baseDelay, maxDelay)
			}

			// Calculate expected base without jitter
			expectedBase := time.Duration(float64(baseDelay.Nanoseconds()) * math.Pow(BackoffMultiplier, float64(retryCount)))

			// Check that we have some variation due to jitter
			minResult := results[0]
			maxResult := results[0]
			for _, result := range results {
				if result < minResult {
					minResult = result
				}
				if result > maxResult {
					maxResult = result
				}
			}

			// Should have some variation (but not too much)
			Expect(maxResult - minResult).To(BeNumerically(">", 0))
			Expect(maxResult).To(BeNumerically("<=", expectedBase*11/10)) // Max 10% over
			Expect(minResult).To(BeNumerically(">=", expectedBase*9/10))  // Min 10% under
		})

		It("should not exceed max delay", func() {
			baseDelay := time.Second
			maxDelay := 5 * time.Second
			retryCount := 10 // This would normally result in 1024 seconds

			result := calculateExponentialBackoff(retryCount, baseDelay, maxDelay)
			Expect(result).To(Equal(maxDelay))
		})

		It("should use defaults for zero values", func() {
			result := calculateExponentialBackoff(1, 0, 0)

			// Should use BaseBackoffDelay and MaxBackoffDelay constants
			expectedBase := time.Duration(float64(BaseBackoffDelay.Nanoseconds()) * math.Pow(BackoffMultiplier, 1.0))
			minExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 0.9)
			maxExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 1.1)

			Expect(result).To(BeNumerically(">=", minExpected))
			Expect(result).To(BeNumerically("<=", maxExpected))
		})

		DescribeTable("calculateExponentialBackoffForOperation should use correct parameters",
			func(operation string, retryCount int, expectedBaseRange, expectedMaxRange time.Duration) {
				result := calculateExponentialBackoffForOperation(retryCount, operation)

				switch operation {
				case "llm-processing":
					// LLM operations: 2s base, 10m max
					expectedBase := time.Duration(float64(2*time.Second) * math.Pow(BackoffMultiplier, float64(retryCount)))
					if expectedBase > 10*time.Minute {
						expectedBase = 10 * time.Minute
					}
					if retryCount >= 9 { // 2 * 2^9 = 1024s = 17+ minutes, exceeds 10m max
						Expect(result).To(Equal(10 * time.Minute))
					} else {
						minExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 0.9)
						maxExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 1.1)
						Expect(result).To(BeNumerically(">=", minExpected))
						Expect(result).To(BeNumerically("<=", maxExpected))
					}
				case "git-operations":
					// Git operations: 5s base, 5m max
					expectedBase := time.Duration(float64(5*time.Second) * math.Pow(BackoffMultiplier, float64(retryCount)))
					if expectedBase > 5*time.Minute {
						expectedBase = 5 * time.Minute
					}
					if retryCount >= 7 { // 5 * 2^7 = 640s = 10+ minutes, exceeds 5m max
						Expect(result).To(Equal(5 * time.Minute))
					} else {
						minExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 0.9)
						maxExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 1.1)
						Expect(result).To(BeNumerically(">=", minExpected))
						Expect(result).To(BeNumerically("<=", maxExpected))
					}
				default:
					// Default: use base configuration
					expectedBase := time.Duration(float64(BaseBackoffDelay.Nanoseconds()) * math.Pow(BackoffMultiplier, float64(retryCount)))
					if expectedBase > MaxBackoffDelay {
						expectedBase = MaxBackoffDelay
					}
					minExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 0.9)
					maxExpected := time.Duration(float64(expectedBase.Nanoseconds()) * 1.1)
					if expectedBase == MaxBackoffDelay {
						Expect(result).To(Equal(MaxBackoffDelay))
					} else {
						Expect(result).To(BeNumerically(">=", minExpected))
						Expect(result).To(BeNumerically("<=", maxExpected))
					}
				}
			},
			Entry("LLM processing retry 0", "llm-processing", 0, 2*time.Second, 10*time.Minute),
			Entry("LLM processing retry 2", "llm-processing", 2, 8*time.Second, 10*time.Minute),
			Entry("LLM processing retry 10", "llm-processing", 10, 10*time.Minute, 10*time.Minute),
			Entry("Git operations retry 0", "git-operations", 0, 5*time.Second, 5*time.Minute),
			Entry("Git operations retry 2", "git-operations", 2, 20*time.Second, 5*time.Minute),
			Entry("Git operations retry 10", "git-operations", 10, 5*time.Minute, 5*time.Minute),
			Entry("Unknown operation", "unknown", 1, BaseBackoffDelay*2, MaxBackoffDelay),
		)
	})

	Describe("LLM Processing Error Handling", func() {
		BeforeEach(func() {
			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())
		})

		It("should set Ready=False condition on LLM failures", func() {
			mockLLMClient := &MockLLMClient{
				Error: fmt.Errorf("LLM service unavailable"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: &MockGitClient{},
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Check NetworkIntent was updated
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			// Check Ready condition is False
			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("LLMProcessingRetrying"))

			// Check retry count was incremented
			retryCount := getRetryCount(updated, "llm-processing")
			Expect(retryCount).To(Equal(1))
		})

		It("should increment retry count correctly", func() {
			mockLLMClient := &MockLLMClient{
				Error: fmt.Errorf("persistent failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: &MockGitClient{},
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// First failure
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(getRetryCount(updated, "llm-processing")).To(Equal(1))

			// Second failure
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(getRetryCount(updated, "llm-processing")).To(Equal(2))

			// Third failure
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(getRetryCount(updated, "llm-processing")).To(Equal(3))
		})

		It("should use exponential backoff for retries", func() {
			mockLLMClient := &MockLLMClient{
				Error: fmt.Errorf("temporary failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: &MockGitClient{},
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// First retry (count 0)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			expectedDelay0 := calculateExponentialBackoffForOperation(0, "llm-processing")
			Expect(result.RequeueAfter).To(BeNumerically("~", expectedDelay0, float64(expectedDelay0)*0.1))

			// Second retry (count 1)
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			expectedDelay1 := calculateExponentialBackoffForOperation(1, "llm-processing")
			Expect(result.RequeueAfter).To(BeNumerically("~", expectedDelay1, float64(expectedDelay1)*0.1))

			// Third retry (count 2)
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			expectedDelay2 := calculateExponentialBackoffForOperation(2, "llm-processing")
			Expect(result.RequeueAfter).To(BeNumerically("~", expectedDelay2, float64(expectedDelay2)*0.1))
		})

		It("should handle max retries behavior", func() {
			mockLLMClient := &MockLLMClient{
				Error: fmt.Errorf("permanent failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: &MockGitClient{},
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// Exhaust all retries
			for i := 0; i <= reconciler.config.MaxRetries; i++ {
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				if i < reconciler.config.MaxRetries {
					Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				} else {
					// After max retries, should return error
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("max retries"))
				}
			}

			// Check final status
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("LLMProcessingFailed"))
		})

		It("should clear retry count on success", func() {
			// First, cause some failures to increment retry count
			mockLLMClient := &MockLLMClient{
				Error:     fmt.Errorf("temporary failure"),
				FailCount: 2,
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: &MockGitClient{CommitHash: "abc123"},
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// Two failures
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			_, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Verify retry count is set
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(getRetryCount(updated, "llm-processing")).To(Equal(2))

			// Now make it succeed
			mockResponse := map[string]interface{}{"action": "deploy", "component": "amf"}
			responseJSON, _ := json.Marshal(mockResponse)
			mockLLMClient.Response = string(responseJSON)
			mockLLMClient.Error = nil

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify retry count was cleared
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(getRetryCount(updated, "llm-processing")).To(Equal(0))
		})
	})

	Describe("Git Operations Error Handling", func() {
		BeforeEach(func() {
			// Create a processed NetworkIntent
			parameters := map[string]interface{}{"action": "deploy", "component": "amf"}
			parametersRaw, _ := json.Marshal(parameters)
			networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}
			networkIntent.Status.Conditions = []metav1.Condition{
				{
					Type:   "Processed",
					Status: metav1.ConditionTrue,
					Reason: "LLMProcessingSucceeded",
				},
			}
			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())
		})

		It("should set Ready=False on Git commit failures", func() {
			mockGitClient := &MockGitClient{
				Error: fmt.Errorf("git commit failed"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: &MockLLMClient{},
				gitClient: mockGitClient,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("GitRepoInitializationFailed"))

			retryCount := getRetryCount(updated, "git-deployment")
			Expect(retryCount).To(Equal(1))
		})

		It("should set Ready=False on Git push failures", func() {
			mockGitClient := &MockGitClient{
				InitError:       nil, // Init succeeds
				CommitPushError: fmt.Errorf("git push failed"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: &MockLLMClient{},
				gitClient: mockGitClient,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal("GitCommitPushFailed"))
		})

		It("should use exponential backoff for Git retries", func() {
			mockGitClient := &MockGitClient{
				Error: fmt.Errorf("git failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: &MockLLMClient{},
				gitClient: mockGitClient,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// First retry
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			expectedDelay := calculateExponentialBackoffForOperation(0, "git-operations")
			Expect(result.RequeueAfter).To(BeNumerically("~", expectedDelay, float64(expectedDelay)*0.1))

			// Second retry
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			expectedDelay = calculateExponentialBackoffForOperation(1, "git-operations")
			Expect(result.RequeueAfter).To(BeNumerically("~", expectedDelay, float64(expectedDelay)*0.1))
		})
	})

	Describe("Status Condition Management", func() {
		BeforeEach(func() {
			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())
		})

		It("should set setReadyCondition updates status correctly", func() {
			err := reconciler.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "TestReason", "Test message")
			Expect(err).NotTo(HaveOccurred())

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			condition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal("TestReason"))
			Expect(condition.Message).To(Equal("Test message"))
		})

		It("should handle condition transitions from False to True", func() {
			// Set to False first
			err := reconciler.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "TestFailure", "Test failure")
			Expect(err).NotTo(HaveOccurred())

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			condition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			firstTransitionTime := condition.LastTransitionTime

			// Small delay to ensure different timestamp
			time.Sleep(time.Millisecond * 10)

			// Now set to True
			err = reconciler.setReadyCondition(ctx, networkIntent, metav1.ConditionTrue, "TestSuccess", "Test success")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			condition = getConditionByType(updated.Status.Conditions, "Ready")
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal("TestSuccess"))
			Expect(condition.Message).To(Equal("Test success"))

			// LastTransitionTime should have changed
			Expect(condition.LastTransitionTime.After(firstTransitionTime.Time)).To(BeTrue())
		})

		It("should include error messages in conditions", func() {
			errorMsg := "specific error details for troubleshooting"
			err := reconciler.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "SpecificFailure", errorMsg)
			Expect(err).NotTo(HaveOccurred())

			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			condition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(condition.Message).To(Equal(errorMsg))
		})
	})

	Describe("Finalizer Management", func() {
		It("should not remove finalizer until successful Git push", func() {
			// Setup deletion scenario
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())

			// Mock Git client that fails
			mockGitClient := &MockGitClient{
				RemoveDirectoryError: fmt.Errorf("git push failed"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: &MockLLMClient{},
				gitClient: mockGitClient,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify finalizer is still present
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(containsFinalizer(updated.Finalizers, NetworkIntentFinalizer)).To(BeTrue())

			// Now make Git operation succeed
			mockGitClient.RemoveDirectoryError = nil

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify finalizer is removed
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(containsFinalizer(updated.Finalizers, NetworkIntentFinalizer)).To(BeFalse())
		})

		It("should remove finalizer after max cleanup retries", func() {
			// Setup deletion scenario
			now := metav1.Now()
			networkIntent.DeletionTimestamp = &now
			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())

			// Mock Git client that always fails
			mockGitClient := &MockGitClient{
				RemoveDirectoryError: fmt.Errorf("permanent git failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: &MockLLMClient{},
				gitClient: mockGitClient,
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// Exhaust all cleanup retries
			for i := 0; i <= reconciler.config.MaxRetries; i++ {
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				if i < reconciler.config.MaxRetries {
					Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				} else {
					// After max retries, finalizer should be removed
					Expect(result.Requeue).To(BeFalse())
				}
			}

			// Verify finalizer was removed to prevent stuck resource
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(containsFinalizer(updated.Finalizers, NetworkIntentFinalizer)).To(BeFalse())

			// Verify condition shows cleanup failed
			condition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal("CleanupFailedMaxRetries"))
		})
	})

	Describe("Idempotent Reconciliation", func() {
		It("should handle multiple reconcile calls with same state", func() {
			mockLLMClient := &MockLLMClient{
				Response: `{"action": "deploy", "component": "test"}`,
				Error:    nil,
			}
			mockGitClient := &MockGitClient{
				CommitHash: "abc123",
				Error:      nil,
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: mockGitClient,
			}

			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// First reconcile
			result1, err1 := reconciler.Reconcile(ctx, req)
			Expect(err1).NotTo(HaveOccurred())

			// Get state after first reconcile
			state1 := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), state1)).To(Succeed())

			// Second reconcile with same conditions
			result2, err2 := reconciler.Reconcile(ctx, req)
			Expect(err2).NotTo(HaveOccurred())

			// Get state after second reconcile
			state2 := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), state2)).To(Succeed())

			// Results should be the same
			Expect(result1.Requeue).To(Equal(result2.Requeue))
			Expect(result1.RequeueAfter).To(Equal(result2.RequeueAfter))

			// States should be equivalent (ignoring timestamps)
			Expect(state1.Status.Phase).To(Equal(state2.Status.Phase))
			Expect(len(state1.Status.Conditions)).To(Equal(len(state2.Status.Conditions)))
		})

		It("should handle partial failures correctly", func() {
			// Create a scenario where LLM succeeds but Git fails
			mockLLMClient := &MockLLMClient{
				Response: `{"action": "deploy", "component": "test"}`,
				Error:    nil,
			}
			mockGitClient := &MockGitClient{
				Error: fmt.Errorf("git failure"),
			}
			reconciler.deps = &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: mockGitClient,
			}

			Expect(fakeClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// First reconcile - should process LLM but fail on Git
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify LLM processing succeeded
			updated := &nephoranv1.NetworkIntent{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			processedCondition := getConditionByType(updated.Status.Conditions, "Processed")
			Expect(processedCondition).NotTo(BeNil())
			Expect(processedCondition.Status).To(Equal(metav1.ConditionTrue))

			// Verify Git deployment failed
			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))

			// Fix Git and reconcile again
			mockGitClient.Error = nil
			mockGitClient.CommitHash = "def456"

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify both phases now succeeded
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			processedCondition = getConditionByType(updated.Status.Conditions, "Processed")
			Expect(processedCondition.Status).To(Equal(metav1.ConditionTrue))

			readyCondition = getConditionByType(updated.Status.Conditions, "Ready")
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})

// Test helper functions and mocks

// Simple Dependencies implementation for testing
type TestDependencies struct {
	llmClient interface{}
	gitClient interface{}
}

func (t *TestDependencies) GetLLMClient() interface{} {
	return t.llmClient
}

func (t *TestDependencies) GetGitClient() interface{} {
	return t.gitClient
}

func (t *TestDependencies) GetTelecomKnowledgeBase() interface{} {
	return nil // Not used in these tests
}

func (t *TestDependencies) GetPackageGenerator() interface{} {
	return nil // Not used in these tests
}

func (t *TestDependencies) GetHTTPClient() interface{} {
	return nil // Not used in these tests
}

func (t *TestDependencies) GetEventRecorder() record.EventRecorder {
	return &record.FakeRecorder{}
}

func (t *TestDependencies) GetMetricsCollector() interface{} {
	return nil // Not used in these tests
}

// Enhanced MockLLMClient for error testing
type MockLLMClient struct {
	Response  string
	Error     error
	CallCount int
	FailCount int
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	m.CallCount++

	if m.FailCount > 0 && m.CallCount <= m.FailCount {
		return "", m.Error
	}

	if m.Error != nil && (m.FailCount == 0 || m.CallCount > m.FailCount) {
		return "", m.Error
	}

	return m.Response, nil
}

// Enhanced MockGitClient for error testing
type MockGitClient struct {
	CommitHash           string
	Error                error
	InitError            error
	CommitPushError      error
	RemoveDirectoryError error
	CallCount            int
	FailCount            int
}

func (m *MockGitClient) InitRepo() error {
	if m.InitError != nil {
		return m.InitError
	}
	if m.Error != nil {
		return m.Error
	}
	return nil
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	m.CallCount++

	if m.CommitPushError != nil {
		return "", m.CommitPushError
	}

	if m.FailCount > 0 && m.CallCount <= m.FailCount {
		return "", m.Error
	}

	if m.Error != nil && (m.FailCount == 0 || m.CallCount > m.FailCount) {
		return "", m.Error
	}

	return m.CommitHash, nil
}

func (m *MockGitClient) RemoveDirectory(path, message string) error {
	if m.RemoveDirectoryError != nil {
		return m.RemoveDirectoryError
	}
	if m.Error != nil {
		return m.Error
	}
	return nil
}

// Helper functions for testing
func getConditionByType(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// Constants that should match the main implementation
const (
	BaseBackoffDelay  = 1 * time.Second
	MaxBackoffDelay   = 5 * time.Minute
	BackoffMultiplier = 2.0
	JitterFactor      = 0.1 // 10% jitter
)

// Table-driven test for comprehensive scenarios
func TestNetworkIntentErrorHandlingTableDriven(t *testing.T) {
	testCases := []struct {
		name                string
		llmError            error
		llmFailCount        int
		gitError            error
		gitFailCount        int
		maxRetries          int
		expectedRetriesLLM  int
		expectedRetriesGit  int
		expectedFinalPhase  string
		expectedReadyStatus metav1.ConditionStatus
	}{
		{
			name:                "LLM fails once then succeeds",
			llmError:            fmt.Errorf("temporary LLM failure"),
			llmFailCount:        1,
			gitError:            nil,
			gitFailCount:        0,
			maxRetries:          3,
			expectedRetriesLLM:  1,
			expectedRetriesGit:  0,
			expectedFinalPhase:  "Completed",
			expectedReadyStatus: metav1.ConditionTrue,
		},
		{
			name:                "Git fails once then succeeds",
			llmError:            nil,
			llmFailCount:        0,
			gitError:            fmt.Errorf("temporary git failure"),
			gitFailCount:        1,
			maxRetries:          3,
			expectedRetriesLLM:  0,
			expectedRetriesGit:  1,
			expectedFinalPhase:  "Completed",
			expectedReadyStatus: metav1.ConditionTrue,
		},
		{
			name:                "Both LLM and Git fail then succeed",
			llmError:            fmt.Errorf("temporary LLM failure"),
			llmFailCount:        1,
			gitError:            fmt.Errorf("temporary git failure"),
			gitFailCount:        1,
			maxRetries:          3,
			expectedRetriesLLM:  1,
			expectedRetriesGit:  1,
			expectedFinalPhase:  "Completed",
			expectedReadyStatus: metav1.ConditionTrue,
		},
		{
			name:                "LLM exceeds max retries",
			llmError:            fmt.Errorf("permanent LLM failure"),
			llmFailCount:        0, // Always fail
			gitError:            nil,
			gitFailCount:        0,
			maxRetries:          3,
			expectedRetriesLLM:  3,
			expectedRetriesGit:  0,
			expectedFinalPhase:  "Failed",
			expectedReadyStatus: metav1.ConditionFalse,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test environment similar to Ginkgo tests
			ctx := context.Background()
			scheme := runtime.NewScheme()
			if err := nephoranv1.AddToScheme(scheme); err != nil {
				t.Fatalf("Failed to add scheme: %v", err)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			mockLLMClient := &MockLLMClient{
				Response:  `{"action": "deploy", "component": "test"}`,
				Error:     tc.llmError,
				FailCount: tc.llmFailCount,
			}

			mockGitClient := &MockGitClient{
				CommitHash: "abc123",
				Error:      tc.gitError,
				FailCount:  tc.gitFailCount,
			}

			deps := &TestDependencies{
				llmClient: mockLLMClient,
				gitClient: mockGitClient,
			}

			config := &Config{
				MaxRetries:    tc.maxRetries,
				RetryDelay:    time.Millisecond * 100, // Fast for tests
				GitRepoURL:    "https://github.com/test/repo.git",
				GitBranch:     "main",
				GitDeployPath: "deployments",
			}

			reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, deps, config)
			if err != nil {
				t.Fatalf("Failed to create reconciler: %v", err)
			}
			reconciler.EventRecorder = &record.FakeRecorder{}

			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-intent",
					Namespace:  "default",
					Finalizers: []string{NetworkIntentFinalizer},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Test intent processing",
				},
			}

			if err := fakeClient.Create(ctx, networkIntent); err != nil {
				t.Fatalf("Failed to create NetworkIntent: %v", err)
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name: networkIntent.Name, Namespace: networkIntent.Namespace}}

			// Run reconciliation until completion or max retries
			maxAttempts := (tc.maxRetries + 1) * 2 // Allow for both LLM and Git retries
			for i := 0; i < maxAttempts; i++ {
				result, err := reconciler.Reconcile(ctx, req)

				// For successful cases, break when no requeue needed
				if err == nil && !result.Requeue && result.RequeueAfter == 0 {
					break
				}

				// For failure cases, continue until error or max attempts
				if err != nil && strings.Contains(err.Error(), "max retries") {
					break
				}

				// Small delay to simulate real timing
				time.Sleep(time.Millisecond)
			}

			// Verify final state
			updated := &nephoranv1.NetworkIntent{}
			if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
				t.Fatalf("Failed to get updated NetworkIntent: %v", err)
			}

			// Check retry counts (note: they should be cleared on success)
			llmRetryCount := getRetryCount(updated, "llm-processing")
			gitRetryCount := getRetryCount(updated, "git-deployment")

			if tc.expectedFinalPhase == "Completed" {
				// On success, retry counts should be cleared
				if llmRetryCount != 0 {
					t.Errorf("Expected LLM retry count to be cleared (0), got %d", llmRetryCount)
				}
				if gitRetryCount != 0 {
					t.Errorf("Expected Git retry count to be cleared (0), got %d", gitRetryCount)
				}
			} else {
				// On failure, retry counts should match expectations
				if llmRetryCount != tc.expectedRetriesLLM {
					t.Errorf("Expected LLM retry count %d, got %d", tc.expectedRetriesLLM, llmRetryCount)
				}
				if gitRetryCount != tc.expectedRetriesGit {
					t.Errorf("Expected Git retry count %d, got %d", tc.expectedRetriesGit, gitRetryCount)
				}
			}

			// Check Ready condition
			readyCondition := getConditionByType(updated.Status.Conditions, "Ready")
			if readyCondition == nil {
				t.Fatal("Ready condition not found")
			}

			if readyCondition.Status != tc.expectedReadyStatus {
				t.Errorf("Expected Ready condition status %s, got %s", tc.expectedReadyStatus, readyCondition.Status)
			}
		})
	}
}

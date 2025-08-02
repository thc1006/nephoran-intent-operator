package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NetworkIntent Controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	var (
		namespaceName string
		reconciler    *NetworkIntentReconciler
	)

	BeforeEach(func() {
		By("Creating a new namespace for test isolation")
		namespaceName = CreateIsolatedNamespace("networkintent-controller")

		By("Setting up the reconciler with mock dependencies")
		reconciler = &NetworkIntentReconciler{
			Client:          k8sClient,
			Scheme:          testEnv.Scheme,
			EventRecorder:   &record.FakeRecorder{},
			MaxRetries:      3,
			RetryDelay:      time.Second * 1,
			GitRepoURL:      "https://github.com/test/deployments.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			HTTPClient:      &http.Client{Timeout: 30 * time.Second},
		}
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		CleanupIsolatedNamespace(namespaceName)
	})

	Context("When creating a NetworkIntent", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			By("Creating a basic NetworkIntent")
			networkIntent = CreateTestNetworkIntent(
				GetUniqueName("test-intent"),
				namespaceName,
				"Scale E2 nodes to 3 replicas in the O-RAN deployment",
			)
		})

		It("Should process the intent with successful LLM response", func() {
			By("Setting up a mock LLM client that returns valid JSON")
			mockResponse := map[string]interface{}{
				"action":        "scale",
				"component":     "e2-nodes",
				"replicas":      3,
				"target_ns":     "oran-system",
				"resource_type": "deployment",
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			mockLLMClient := &MockLLMClient{
				Response: string(mockResponseBytes),
				Error:    nil,
			}
			reconciler.LLMClient = mockLLMClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying the NetworkIntent was processed successfully")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Processed")
			}, timeout, interval).Should(BeTrue())

			By("Verifying parameters were extracted from LLM response")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(updated.Spec.Parameters.Raw).NotTo(BeEmpty())

			var extractedParams map[string]interface{}
			Expect(json.Unmarshal(updated.Spec.Parameters.Raw, &extractedParams)).To(Succeed())
			Expect(extractedParams["action"]).To(Equal("scale"))
			Expect(extractedParams["replicas"]).To(Equal(float64(3)))
		})

		It("Should handle LLM processing failures with retry logic", func() {
			By("Setting up a mock LLM client that initially fails")
			mockLLMClient := &MockLLMClient{
				Response:  "",
				Error:     fmt.Errorf("temporary LLM service unavailable"),
				FailCount: 2, // Fail first 2 attempts, succeed on 3rd
				CallCount: 0,
			}
			reconciler.LLMClient = mockLLMClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("First reconciliation should fail and schedule retry")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))

			By("Verifying retry count is incremented")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			retryCount := getRetryCount(updated, "llm-processing")
			Expect(retryCount).To(Equal(1))

			By("Second reconciliation should also fail")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))

			By("Third reconciliation should succeed")
			// Setup successful response for third attempt
			mockResponse := map[string]interface{}{
				"action":    "deploy",
				"component": "network-function",
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			mockLLMClient.Response = string(mockResponseBytes)
			mockLLMClient.Error = nil

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying the intent was eventually processed")
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
				return isConditionTrue(updated.Status.Conditions, "Processed")
			}, timeout, interval).Should(BeTrue())
		})

		It("Should fail after exceeding maximum retry attempts", func() {
			By("Setting up a mock LLM client that always fails")
			mockLLMClient := &MockLLMClient{
				Response: "",
				Error:    fmt.Errorf("permanent LLM failure"),
			}
			reconciler.LLMClient = mockLLMClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("Exhausting all retry attempts")
			for i := 0; i <= reconciler.MaxRetries; i++ {
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				if i < reconciler.MaxRetries {
					Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))
				} else {
					Expect(result.Requeue).To(BeFalse())
					Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
				}
			}

			By("Verifying the intent is marked as failed")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			processedCondition := getCondition(updated.Status.Conditions, "Processed")
			Expect(processedCondition).NotTo(BeNil())
			Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(processedCondition.Reason).To(Equal("LLMProcessingFailedMaxRetries"))
		})

		It("Should handle invalid JSON responses from LLM", func() {
			By("Setting up a mock LLM client that returns invalid JSON")
			mockLLMClient := &MockLLMClient{
				Response: "invalid json response from LLM",
				Error:    nil,
			}
			reconciler.LLMClient = mockLLMClient

			By("Creating the NetworkIntent in the cluster")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Triggering reconciliation")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))

			By("Verifying parsing failure is recorded")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			processedCondition := getCondition(updated.Status.Conditions, "Processed")
			Expect(processedCondition).NotTo(BeNil())
			Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(processedCondition.Reason).To(Equal("LLMResponseParsingFailed"))
		})
	})

	Context("When testing GitOps deployment functionality", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			By("Creating a NetworkIntent that has already been processed")
			networkIntent = CreateTestNetworkIntent(
				GetUniqueName("gitops-test"),
				namespaceName,
				"Deploy 5G core AMF with high availability",
			)

			// Simulate already processed intent
			parameters := map[string]interface{}{
				"action":     "deploy",
				"component":  "amf",
				"replicas":   2,
				"ha_enabled": true,
			}
			parametersRaw, _ := json.Marshal(parameters)
			networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}

			// Set processed condition
			networkIntent.Status.Conditions = []metav1.Condition{
				{
					Type:               "Processed",
					Status:             metav1.ConditionTrue,
					Reason:             "LLMProcessingSucceeded",
					Message:            "Intent successfully processed",
					LastTransitionTime: metav1.Now(),
				},
			}
		})

		It("Should successfully deploy via GitOps", func() {
			By("Setting up a mock Git client")
			mockGitClient := &MockGitClient{
				CommitHash: "abc123def456",
				Error:      nil,
			}
			reconciler.GitClient = mockGitClient

			By("Creating the processed NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Triggering reconciliation for deployment")
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying the intent is marked as deployed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Deployed")
			}, timeout, interval).Should(BeTrue())

			By("Verifying Git commit hash is recorded in status")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(updated.Status.GitCommitHash).To(Equal("abc123def456"))
		})

		It("Should handle Git deployment failures with retry logic", func() {
			By("Setting up a mock Git client that initially fails")
			mockGitClient := &MockGitClient{
				CommitHash: "",
				Error:      fmt.Errorf("git repository authentication failed"),
				FailCount:  1, // Fail first attempt, succeed on second
			}
			reconciler.GitClient = mockGitClient

			By("Creating the processed NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("First reconciliation should fail and schedule retry")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))

			By("Second reconciliation should succeed")
			mockGitClient.CommitHash = "def789abc012"
			mockGitClient.Error = nil

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying deployment eventually succeeds")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Deployed")
			}, timeout, interval).Should(BeTrue())
		})

		It("Should fail GitOps deployment after maximum retries", func() {
			By("Setting up a mock Git client that always fails")
			mockGitClient := &MockGitClient{
				CommitHash: "",
				Error:      fmt.Errorf("permanent git failure"),
			}
			reconciler.GitClient = mockGitClient

			By("Creating the processed NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("Exhausting all retry attempts for Git deployment")
			for i := 0; i <= reconciler.MaxRetries; i++ {
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				if i < reconciler.MaxRetries {
					Expect(result.RequeueAfter).To(Equal(reconciler.RetryDelay))
				} else {
					Expect(result.Requeue).To(BeFalse())
					Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
				}
			}

			By("Verifying deployment is marked as failed")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			deployedCondition := getCondition(updated.Status.Conditions, "Deployed")
			Expect(deployedCondition).NotTo(BeNil())
			Expect(deployedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(deployedCondition.Reason).To(Equal("GitDeploymentFailedMaxRetries"))
		})
	})

	Context("When testing status management and phase transitions", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = CreateTestNetworkIntent(
				GetUniqueName("status-test"),
				namespaceName,
				"Test status transitions for network intent",
			)
		})

		It("Should correctly manage phase transitions", func() {
			By("Setting up successful mock clients")
			mockResponse := map[string]interface{}{
				"action":    "configure",
				"component": "network",
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)

			reconciler.LLMClient = &MockLLMClient{
				Response: string(mockResponseBytes),
				Error:    nil,
			}
			reconciler.GitClient = &MockGitClient{
				CommitHash: "status123test",
				Error:      nil,
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("Triggering reconciliation")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying final phase is Completed")
			Eventually(func() string {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated); err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal("Completed"))

			By("Verifying both conditions are true")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(isConditionTrue(updated.Status.Conditions, "Processed")).To(BeTrue())
			Expect(isConditionTrue(updated.Status.Conditions, "Deployed")).To(BeTrue())

			By("Verifying timing fields are set")
			Expect(updated.Status.ProcessingStartTime).NotTo(BeNil())
			Expect(updated.Status.ProcessingCompletionTime).NotTo(BeNil())
			Expect(updated.Status.DeploymentStartTime).NotTo(BeNil())
			Expect(updated.Status.DeploymentCompletionTime).NotTo(BeNil())
		})

		It("Should handle missing LLM client gracefully", func() {
			By("Setting LLM client to nil")
			reconciler.LLMClient = nil

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("Triggering reconciliation")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute * 5))

			By("Verifying appropriate condition is set")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())

			processedCondition := getCondition(updated.Status.Conditions, "Processed")
			Expect(processedCondition).NotTo(BeNil())
			Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(processedCondition.Reason).To(Equal("LLMProcessingRetrying"))
		})

		It("Should skip processing if already processed and deployed", func() {
			By("Creating an already completed intent")
			networkIntent.Status.Conditions = []metav1.Condition{
				{
					Type:   "Processed",
					Status: metav1.ConditionTrue,
					Reason: "LLMProcessingSucceeded",
				},
				{
					Type:   "Deployed",
					Status: metav1.ConditionTrue,
					Reason: "GitDeploymentSucceeded",
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up mock clients (should not be called)")
			reconciler.LLMClient = &MockLLMClient{
				Response: "",
				Error:    fmt.Errorf("should not be called"),
			}

			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}}

			By("Triggering reconciliation")
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(time.Duration(0)))

			By("Verifying no additional processing occurred")
			// Mock client should not have been called
		})
	})

	Context("When testing deployment file generation", func() {
		var networkIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			networkIntent = CreateTestNetworkIntent(
				GetUniqueName("generation-test"),
				namespaceName,
				"Test deployment file generation",
			)

			// Add processed parameters
			parameters := map[string]interface{}{
				"action":      "deploy",
				"component":   "test-nf",
				"replicas":    3,
				"environment": "development",
			}
			parametersRaw, _ := json.Marshal(parameters)
			networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}
		})

		It("Should generate correct deployment files", func() {
			By("Generating deployment files")
			files, err := reconciler.generateDeploymentFiles(networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).NotTo(BeEmpty())

			By("Verifying file structure")
			expectedPath := fmt.Sprintf("networkintents/%s/%s-configmap.json",
				networkIntent.Namespace, networkIntent.Name)
			content, exists := files[expectedPath]
			Expect(exists).To(BeTrue())
			Expect(content).NotTo(BeEmpty())

			By("Verifying file content structure")
			var manifest map[string]interface{}
			Expect(json.Unmarshal([]byte(content), &manifest)).To(Succeed())

			Expect(manifest["apiVersion"]).To(Equal("v1"))
			Expect(manifest["kind"]).To(Equal("ConfigMap"))

			metadata := manifest["metadata"].(map[string]interface{})
			Expect(metadata["name"]).To(Equal(fmt.Sprintf("networkintent-%s", networkIntent.Name)))
			Expect(metadata["namespace"]).To(Equal(networkIntent.Namespace))

			data := manifest["data"].(map[string]interface{})
			Expect(data["action"]).To(Equal("deploy"))
			Expect(data["component"]).To(Equal("test-nf"))
			Expect(data["replicas"]).To(Equal(float64(3)))
		})

		It("Should handle empty parameters gracefully", func() {
			By("Clearing parameters")
			networkIntent.Spec.Parameters = runtime.RawExtension{}

			By("Generating deployment files")
			files, err := reconciler.generateDeploymentFiles(networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(files).NotTo(BeEmpty())

			By("Verifying empty data section")
			expectedPath := fmt.Sprintf("networkintents/%s/%s-configmap.json",
				networkIntent.Namespace, networkIntent.Name)
			content := files[expectedPath]

			var manifest map[string]interface{}
			Expect(json.Unmarshal([]byte(content), &manifest)).To(Succeed())
			Expect(manifest["data"]).To(BeNil())
		})

		It("Should handle invalid parameters gracefully", func() {
			By("Setting invalid JSON parameters")
			networkIntent.Spec.Parameters = runtime.RawExtension{
				Raw: []byte("invalid json"),
			}

			By("Attempting to generate deployment files")
			files, err := reconciler.generateDeploymentFiles(networkIntent)
			Expect(err).To(HaveOccurred())
			Expect(files).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse parameters"))
		})
	})
})

// Mock implementations for testing

type MockLLMClient struct {
	Response  string
	Error     error
	CallCount int
	FailCount int // Number of times to fail before succeeding
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

type MockGitClient struct {
	CommitHash string
	Error      error
	CallCount  int
	FailCount  int
}

func (m *MockGitClient) InitRepo() error {
	if m.Error != nil {
		return m.Error
	}
	return nil
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	m.CallCount++

	if m.FailCount > 0 && m.CallCount <= m.FailCount {
		return "", m.Error
	}

	if m.Error != nil && (m.FailCount == 0 || m.CallCount > m.FailCount) {
		return "", m.Error
	}

	return m.CommitHash, nil
}

// Helper functions

func getCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

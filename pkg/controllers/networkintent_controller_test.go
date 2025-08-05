package controllers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/hack/testtools"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// MockDependencies implements the Dependencies interface for testing
type MockDependencies struct {
	mock.Mock
	gitClient     *MockGitClient
	llmClient     *MockLLMClient
	packageGen    *MockPackageGenerator
	httpClient    *http.Client
	eventRecorder *MockEventRecorder
}

func NewMockDependencies() *MockDependencies {
	return &MockDependencies{
		gitClient:     NewMockGitClient(),
		llmClient:     NewMockLLMClient(),
		packageGen:    NewMockPackageGenerator(),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		eventRecorder: NewMockEventRecorder(),
	}
}

func (m *MockDependencies) GetGitClient() git.ClientInterface {
	return m.gitClient
}

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
	return m.llmClient
}

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	return m.packageGen.PackageGenerator
}

func (m *MockDependencies) GetHTTPClient() *http.Client {
	return m.httpClient
}

func (m *MockDependencies) GetEventRecorder() record.EventRecorder {
	return m.eventRecorder
}

// MockGitClient implements git.ClientInterface for testing
type MockGitClient struct {
	mock.Mock
}

func NewMockGitClient() *MockGitClient {
	return &MockGitClient{}
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) CommitAndPushChanges(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockGitClient) InitRepo() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) RemoveDirectory(path string, commitMessage string) error {
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

// MockLLMClient implements shared.ClientInterface for testing
type MockLLMClient struct {
	mock.Mock
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{}
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	args := m.Called(ctx, prompt, chunks)
	return args.Error(0)
}

func (m *MockLLMClient) GetSupportedModels() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockLLMClient) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {
	args := m.Called(modelName)
	return args.Get(0).(*shared.ModelCapabilities), args.Error(1)
}

func (m *MockLLMClient) ValidateModel(modelName string) error {
	args := m.Called(modelName)
	return args.Error(0)
}

func (m *MockLLMClient) EstimateTokens(text string) int {
	args := m.Called(text)
	return args.Int(0)
}

func (m *MockLLMClient) GetMaxTokens(modelName string) int {
	args := m.Called(modelName)
	return args.Int(0)
}

func (m *MockLLMClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockPackageGenerator wraps nephio.PackageGenerator for testing
type MockPackageGenerator struct {
	mock.Mock
	PackageGenerator *nephio.PackageGenerator
}

func NewMockPackageGenerator() *MockPackageGenerator {
	return &MockPackageGenerator{
		PackageGenerator: nil, // Will be nil in tests
	}
}

func (m *MockPackageGenerator) GeneratePackage(networkIntent *nephoranv1.NetworkIntent) (map[string]string, error) {
	args := m.Called(networkIntent)
	return args.Get(0).(map[string]string), args.Error(1)
}

// MockEventRecorder implements record.EventRecorder for testing
type MockEventRecorder struct {
	mock.Mock
	Events []string
}

func NewMockEventRecorder() *MockEventRecorder {
	return &MockEventRecorder{
		Events: make([]string, 0),
	}
}

func (m *MockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	m.Called(object, eventtype, reason, message)
	m.Events = append(m.Events, fmt.Sprintf("%s:%s:%s", eventtype, reason, message))
}

func (m *MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	m.Called(object, eventtype, reason, message)
	m.Events = append(m.Events, fmt.Sprintf("%s:%s:%s", eventtype, reason, message))
}

func (m *MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	m.Called(object, annotations, eventtype, reason, message)
	m.Events = append(m.Events, fmt.Sprintf("%s:%s:%s", eventtype, reason, message))
}

var _ = Describe("NetworkIntentReconciler", func() {
	var (
		testEnv           *testtools.TestEnvironment
		ctx               context.Context
		reconciler        *controllers.NetworkIntentReconciler
		mockDeps          *MockDependencies
		testNamespace     string
		networkIntent     *nephoranv1.NetworkIntent
		reconcileRequest  ctrl.Request
	)

	BeforeEach(func() {
		var err error
		testEnv, err = testtools.SetupTestEnvironmentWithOptions(testtools.DefaultTestEnvironmentOptions())
		Expect(err).NotTo(HaveOccurred())

		ctx = testEnv.GetContext()
		testNamespace = testtools.GetUniqueNamespace("ni-test")
		
		// Create test namespace
		Expect(testEnv.CreateNamespace(testNamespace)).To(Succeed())

		// Create mock dependencies
		mockDeps = NewMockDependencies()

		// Create reconciler with mock dependencies
		config := &controllers.Config{
			MaxRetries:      3,
			RetryDelay:      1 * time.Second,
			Timeout:         30 * time.Second,
			GitRepoURL:      "https://github.com/test/repo.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			LLMProcessorURL: "http://llm-processor:8080",
			UseNephioPorch:  false,
		}

		reconciler, err = controllers.NewNetworkIntentReconciler(
			testEnv.K8sClient,
			testEnv.GetScheme(),
			mockDeps,
			config,
		)
		Expect(err).NotTo(HaveOccurred())

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-intent",
				Namespace: testNamespace,
				Labels: map[string]string{
					"test-resource": "true",
				},
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "Create a 5G UPF deployment with 3 replicas",
			},
		}

		reconcileRequest = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			},
		}
	})

	AfterEach(func() {
		if testEnv != nil {
			testEnv.TeardownTestEnvironment()
		}
	})

	Describe("Constructor validation", func() {
		It("should create reconciler with valid parameters", func() {
			config := &controllers.Config{
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				Timeout:       30 * time.Second,
				GitDeployPath: "test-path",
			}

			r, err := controllers.NewNetworkIntentReconciler(
				testEnv.K8sClient,
				testEnv.GetScheme(),
				mockDeps,
				config,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
		})

		It("should reject nil client", func() {
			config := &controllers.Config{}
			_, err := controllers.NewNetworkIntentReconciler(
				nil,
				testEnv.GetScheme(),
				mockDeps,
				config,
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("client cannot be nil"))
		})

		It("should reject nil scheme", func() {
			config := &controllers.Config{}
			_, err := controllers.NewNetworkIntentReconciler(
				testEnv.K8sClient,
				nil,
				mockDeps,
				config,
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scheme cannot be nil"))
		})

		It("should reject nil dependencies", func() {
			config := &controllers.Config{}
			_, err := controllers.NewNetworkIntentReconciler(
				testEnv.K8sClient,
				testEnv.GetScheme(),
				nil,
				config,
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("dependencies cannot be nil"))
		})

		It("should set default configuration values", func() {
			r, err := controllers.NewNetworkIntentReconciler(
				testEnv.K8sClient,
				testEnv.GetScheme(),
				mockDeps,
				nil, // nil config should use defaults
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
		})

		It("should validate configuration limits", func() {
			config := &controllers.Config{
				MaxRetries: 15, // Exceeds MaxAllowedRetries (10)
			}

			_, err := controllers.NewNetworkIntentReconciler(
				testEnv.K8sClient,
				testEnv.GetScheme(),
				mockDeps,
				config,
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MaxRetries"))
		})
	})

	Describe("Reconcile method", func() {
		Context("when NetworkIntent does not exist", func() {
			It("should return without error", func() {
				// Don't create the NetworkIntent
				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})
		})

		Context("when NetworkIntent exists", func() {
			BeforeEach(func() {
				Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())
			})

			It("should add finalizer on first reconcile", func() {
				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeTrue())

				// Verify finalizer was added
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Finalizers).To(ContainElement(controllers.NetworkIntentFinalizer))
			})

			It("should process intent with LLM client", func() {
				// Add finalizer manually to skip that step
				networkIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
				Expect(testEnv.K8sClient.Update(ctx, networkIntent)).To(Succeed())

				// Mock successful LLM processing
				expectedResponse := `{"upf_replicas": 3, "network_function": "UPF"}`
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, networkIntent.Spec.Intent).Return(expectedResponse, nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify LLM client was called
				mockDeps.llmClient.AssertExpectations(GinkgoT())

				// Verify status was updated
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Status.Phase).To(Equal("Processing"))

				// Verify parameters were set
				var params map[string]interface{}
				Expect(json.Unmarshal(updated.Spec.Parameters.Raw, &params)).To(Succeed())
				Expect(params).To(HaveKeyWithValue("upf_replicas", float64(3)))
				Expect(params).To(HaveKeyWithValue("network_function", "UPF"))
			})
		})

		Context("when NetworkIntent is being deleted", func() {
			BeforeEach(func() {
				networkIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
				Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())

				// Mark for deletion
				Expect(testEnv.K8sClient.Delete(ctx, networkIntent)).To(Succeed())
			})

			It("should handle deletion with successful cleanup", func() {
				// Mock successful Git cleanup
				mockDeps.gitClient.On("InitRepo").Return(nil)
				mockDeps.gitClient.On("RemoveDirectory", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify Git client was called
				mockDeps.gitClient.AssertExpectations(GinkgoT())

				// Verify resource was deleted (finalizer removed)
				deleted := &nephoranv1.NetworkIntent{}
				err = testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, deleted)
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("should retry on cleanup failure", func() {
				// Mock failed Git cleanup
				mockDeps.gitClient.On("InitRepo").Return(fmt.Errorf("git initialization failed"))

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify Git client was called
				mockDeps.gitClient.AssertExpectations(GinkgoT())

				// Verify finalizer still exists
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Finalizers).To(ContainElement(controllers.NetworkIntentFinalizer))
			})
		})
	})

	Describe("LLM Processing", func() {
		var processedIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			processedIntent = networkIntent.DeepCopy()
			processedIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
			Expect(testEnv.K8sClient.Create(ctx, processedIntent)).To(Succeed())
		})

		Context("with successful LLM response", func() {
			It("should process valid JSON response", func() {
				expectedResponse := `{"deployment_type": "upf", "replicas": 3, "resources": {"cpu": "2", "memory": "4Gi"}}`
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return(expectedResponse, nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify parameters were parsed and stored
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())

				var params map[string]interface{}
				Expect(json.Unmarshal(updated.Spec.Parameters.Raw, &params)).To(Succeed())
				Expect(params).To(HaveKey("deployment_type"))
				Expect(params).To(HaveKey("replicas"))
				Expect(params).To(HaveKey("resources"))
			})

			It("should handle empty JSON response", func() {
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return("{}", nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify empty parameters were stored
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())

				var params map[string]interface{}
				Expect(json.Unmarshal(updated.Spec.Parameters.Raw, &params)).To(Succeed())
				Expect(len(params)).To(Equal(0))
			})
		})

		Context("with LLM processing failures", func() {
			It("should retry on LLM client error", func() {
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return("", fmt.Errorf("LLM service unavailable"))

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify retry count was incremented
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Annotations).To(HaveKey("nephoran.com/llm-processing-retry-count"))
			})

			It("should retry on empty LLM response", func() {
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return("", nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify condition shows empty response error
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("EmptyLLMResponse"))
			})

			It("should retry on invalid JSON response", func() {
				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return("invalid json {", nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify condition shows parsing error
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("LLMResponseParsingFailed"))
			})

			It("should fail after max retries", func() {
				// Set up intent with max retry count already reached
				processedIntent.Annotations = map[string]string{
					"nephoran.com/llm-processing-retry-count": "3",
				}
				Expect(testEnv.K8sClient.Update(ctx, processedIntent)).To(Succeed())

				mockDeps.llmClient.On("ProcessIntent", mock.Anything, processedIntent.Spec.Intent).Return("", fmt.Errorf("persistent error"))

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify condition shows max retries exceeded
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("LLMProcessingFailedMaxRetries"))
			})

			It("should handle empty intent specification", func() {
				processedIntent.Spec.Intent = ""
				Expect(testEnv.K8sClient.Update(ctx, processedIntent)).To(Succeed())

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify condition shows invalid intent
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("InvalidIntent"))
			})
		})

		Context("with missing LLM client", func() {
			BeforeEach(func() {
				// Create mock deps with nil LLM client
				mockDeps = &MockDependencies{
					gitClient:     NewMockGitClient(),
					llmClient:     nil,
					packageGen:    NewMockPackageGenerator(),
					httpClient:    &http.Client{},
					eventRecorder: NewMockEventRecorder(),
				}

				var err error
				reconciler, err = controllers.NewNetworkIntentReconciler(
					testEnv.K8sClient,
					testEnv.GetScheme(),
					mockDeps,
					&controllers.Config{},
				)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail permanently with no LLM client", func() {
				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify condition shows LLM client not configured
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("LLMClientNotConfigured"))
			})
		})
	})

	Describe("GitOps Deployment", func() {
		var deployableIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			// Create intent that's already processed
			deployableIntent = networkIntent.DeepCopy()
			deployableIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
			deployableIntent.Spec.Parameters = runtime.RawExtension{
				Raw: []byte(`{"deployment_type": "upf", "replicas": 3}`),
			}
			deployableIntent.Status.Conditions = []metav1.Condition{
				{
					Type:   "Processed",
					Status: metav1.ConditionTrue,
					Reason: "LLMProcessingSucceeded",
				},
			}
			Expect(testEnv.K8sClient.Create(ctx, deployableIntent)).To(Succeed())
		})

		Context("with successful deployment", func() {
			It("should deploy via GitOps", func() {
				// Mock successful Git operations
				mockDeps.gitClient.On("InitRepo").Return(nil)
				expectedFiles := map[string]string{
					"networkintents/test-intent/test-intent-configmap.json": `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"networkintent-test-intent","namespace":"` + testNamespace + `","labels":{"app.kubernetes.io/name":"networkintent","app.kubernetes.io/instance":"test-intent","app.kubernetes.io/managed-by":"nephoran-intent-operator"}},"data":{"deployment_type":"upf","replicas":3}}`,
				}
				mockDeps.gitClient.On("CommitAndPush", mock.MatchedBy(func(files map[string]string) bool {
					return len(files) > 0
				}), mock.AnythingOfType("string")).Return("abc123", nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify Git operations were called
				mockDeps.gitClient.AssertExpectations(GinkgoT())

				// Verify status was updated
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Status.Phase).To(Equal("Completed"))
				Expect(updated.Status.GitCommitHash).To(Equal("abc123"))

				// Verify deployed condition
				deployedCondition := findCondition(updated.Status.Conditions, "Deployed")
				Expect(deployedCondition).NotTo(BeNil())
				Expect(deployedCondition.Status).To(Equal(metav1.ConditionTrue))
				Expect(deployedCondition.Reason).To(Equal("GitDeploymentSucceeded"))
			})

			It("should use Nephio package generator when enabled", func() {
				// Update reconciler config to use Nephio
				config := &controllers.Config{
					UseNephioPorch: true,
				}
				var err error
				reconciler, err = controllers.NewNetworkIntentReconciler(
					testEnv.K8sClient,
					testEnv.GetScheme(),
					mockDeps,
					config,
				)
				Expect(err).NotTo(HaveOccurred())

				// Mock package generation
				expectedPackage := map[string]string{
					"Kptfile": "apiVersion: kpt.dev/v1\nkind: Kptfile\n",
					"upf-deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\n",
				}
				mockDeps.packageGen.On("GeneratePackage", deployableIntent).Return(expectedPackage, nil)
				mockDeps.gitClient.On("InitRepo").Return(nil)
				mockDeps.gitClient.On("CommitAndPush", expectedPackage, mock.AnythingOfType("string")).Return("def456", nil)

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify package generator was called
				mockDeps.packageGen.AssertExpectations(GinkgoT())
				mockDeps.gitClient.AssertExpectations(GinkgoT())
			})
		})

		Context("with deployment failures", func() {
			It("should retry on Git initialization failure", func() {
				mockDeps.gitClient.On("InitRepo").Return(fmt.Errorf("git init failed"))

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify retry count was incremented
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				Expect(updated.Annotations).To(HaveKey("nephoran.com/git-deployment-retry-count"))

				// Verify condition shows Git failure
				deployedCondition := findCondition(updated.Status.Conditions, "Deployed")
				Expect(deployedCondition).NotTo(BeNil())
				Expect(deployedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(deployedCondition.Reason).To(Equal("GitRepoInitializationFailed"))
			})

			It("should retry on commit and push failure", func() {
				mockDeps.gitClient.On("InitRepo").Return(nil)
				mockDeps.gitClient.On("CommitAndPush", mock.Anything, mock.AnythingOfType("string")).Return("", fmt.Errorf("push failed"))

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))

				// Verify condition shows commit/push failure
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				deployedCondition := findCondition(updated.Status.Conditions, "Deployed")
				Expect(deployedCondition).NotTo(BeNil())
				Expect(deployedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(deployedCondition.Reason).To(Equal("GitCommitPushFailed"))
			})

			It("should fail after max retries", func() {
				// Set up intent with max retry count already reached
				deployableIntent.Annotations = map[string]string{
					"nephoran.com/git-deployment-retry-count": "3",
				}
				Expect(testEnv.K8sClient.Update(ctx, deployableIntent)).To(Succeed())

				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify condition shows max retries exceeded
				updated := &nephoranv1.NetworkIntent{}
				Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
				
				deployedCondition := findCondition(updated.Status.Conditions, "Deployed")
				Expect(deployedCondition).NotTo(BeNil())
				Expect(deployedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(deployedCondition.Reason).To(Equal("GitDeploymentFailedMaxRetries"))
			})
		})

		Context("with missing Git client", func() {
			BeforeEach(func() {
				// Create mock deps with nil Git client
				mockDeps = &MockDependencies{
					gitClient:     nil,
					llmClient:     NewMockLLMClient(),
					packageGen:    NewMockPackageGenerator(),
					httpClient:    &http.Client{},
					eventRecorder: NewMockEventRecorder(),
				}

				var err error
				reconciler, err = controllers.NewNetworkIntentReconciler(
					testEnv.K8sClient,
					testEnv.GetScheme(),
					mockDeps,
					&controllers.Config{},
				)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should requeue with Git client not configured", func() {
				result, err := reconciler.Reconcile(ctx, reconcileRequest)

				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(5 * time.Minute))
			})
		})
	})

	Describe("Status and Phase Management", func() {
		var statusIntent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			statusIntent = networkIntent.DeepCopy()
			statusIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
			Expect(testEnv.K8sClient.Create(ctx, statusIntent)).To(Succeed())
		})

		It("should update observed generation", func() {
			// Update the intent spec to change generation
			statusIntent.Spec.Intent = "Updated intent"
			Expect(testEnv.K8sClient.Update(ctx, statusIntent)).To(Succeed())

			// Mock LLM processing
			mockDeps.llmClient.On("ProcessIntent", mock.Anything, statusIntent.Spec.Intent).Return("{}", nil)

			result, err := reconciler.Reconcile(ctx, reconcileRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify observed generation was updated
			updated := &nephoranv1.NetworkIntent{}
			Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
			Expect(updated.Status.ObservedGeneration).To(Equal(updated.Generation))
		})

		It("should set processing timestamps", func() {
			mockDeps.llmClient.On("ProcessIntent", mock.Anything, statusIntent.Spec.Intent).Return("{}", nil)

			result, err := reconciler.Reconcile(ctx, reconcileRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify timestamps were set
			updated := &nephoranv1.NetworkIntent{}
			Expect(testEnv.K8sClient.Get(ctx, reconcileRequest.NamespacedName, updated)).To(Succeed())
			Expect(updated.Status.ProcessingStartTime).NotTo(BeNil())
			Expect(updated.Status.ProcessingCompletionTime).NotTo(BeNil())
		})

		It("should skip processing if already completed", func() {
			// Set status to already completed
			statusIntent.Status.Phase = "Completed"
			statusIntent.Status.Conditions = []metav1.Condition{
				{Type: "Processed", Status: metav1.ConditionTrue},
				{Type: "Deployed", Status: metav1.ConditionTrue},
			}
			Expect(testEnv.K8sClient.Status().Update(ctx, statusIntent)).To(Succeed())

			result, err := reconciler.Reconcile(ctx, reconcileRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify LLM client was not called
			mockDeps.llmClient.AssertNotCalled(GinkgoT(), "ProcessIntent")
		})
	})

	Describe("Event Recording", func() {
		BeforeEach(func() {
			networkIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
			Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())
		})

		It("should record success events", func() {
			mockDeps.llmClient.On("ProcessIntent", mock.Anything, networkIntent.Spec.Intent).Return("{}", nil)
			mockDeps.eventRecorder.On("Event", mock.Anything, "Normal", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()

			result, err := reconciler.Reconcile(ctx, reconcileRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify events were recorded
			mockDeps.eventRecorder.AssertCalled(GinkgoT(), "Event", mock.Anything, "Normal", "LLMProcessingSucceeded", mock.AnythingOfType("string"))
		})

		It("should record failure events", func() {
			mockDeps.llmClient.On("ProcessIntent", mock.Anything, networkIntent.Spec.Intent).Return("", fmt.Errorf("processing failed"))
			mockDeps.eventRecorder.On("Event", mock.Anything, "Warning", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()

			result, err := reconciler.Reconcile(ctx, reconcileRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify failure events were recorded
			mockDeps.eventRecorder.AssertCalled(GinkgoT(), "Event", mock.Anything, "Warning", "LLMProcessingRetry", mock.AnythingOfType("string"))
		})
	})

	Describe("Context Cancellation", func() {
		It("should handle context cancellation gracefully", func() {
			// Create a cancelled context
			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			networkIntent.Finalizers = []string{controllers.NetworkIntentFinalizer}
			Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())

			result, err := reconciler.Reconcile(cancelledCtx, reconcileRequest)

			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(context.Canceled))
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
})

// Table-driven tests for edge cases
var _ = Describe("NetworkIntentReconciler Edge Cases", func() {
	var (
		testEnv    *testtools.TestEnvironment
		ctx        context.Context
		reconciler *controllers.NetworkIntentReconciler
		mockDeps   *MockDependencies
	)

	BeforeEach(func() {
		var err error
		testEnv, err = testtools.SetupTestEnvironmentWithOptions(testtools.DefaultTestEnvironmentOptions())
		Expect(err).NotTo(HaveOccurred())

		ctx = testEnv.GetContext()
		mockDeps = NewMockDependencies()

		config := &controllers.Config{
			MaxRetries:    2,
			RetryDelay:    100 * time.Millisecond,
			Timeout:       5 * time.Second,
			GitDeployPath: "test-path",
		}

		reconciler, err = controllers.NewNetworkIntentReconciler(
			testEnv.K8sClient,
			testEnv.GetScheme(),
			mockDeps,
			config,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if testEnv != nil {
			testEnv.TeardownTestEnvironment()
		}
	})

	DescribeTable("LLM Processing Edge Cases",
		func(intentText string, llmResponse string, llmError error, expectedPhase string, expectedConditionReason string, shouldRequeue bool) {
			testNamespace := testtools.GetUniqueNamespace("edge-test")
			Expect(testEnv.CreateNamespace(testNamespace)).To(Succeed())

			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "edge-test-intent",
					Namespace:  testNamespace,
					Finalizers: []string{controllers.NetworkIntentFinalizer},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: intentText,
				},
			}
			Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())

			mockDeps.llmClient.On("ProcessIntent", mock.Anything, intentText).Return(llmResponse, llmError)

			request := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, request)

			if shouldRequeue {
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			} else if expectedPhase == "Error" {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify final state
			updated := &nephoranv1.NetworkIntent{}
			Expect(testEnv.K8sClient.Get(ctx, request.NamespacedName, updated)).To(Succeed())

			if expectedPhase != "" {
				Expect(updated.Status.Phase).To(Equal(expectedPhase))
			}

			if expectedConditionReason != "" {
				processedCondition := findCondition(updated.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Reason).To(Equal(expectedConditionReason))
			}
		},
		Entry("Empty intent", "", "", nil, "Error", "InvalidIntent", false),
		Entry("Valid JSON response", "Create UPF", `{"type":"upf"}`, nil, "Processing", "LLMProcessingSucceeded", false),
		Entry("Empty JSON response", "Create UPF", `{}`, nil, "Processing", "LLMProcessingSucceeded", false),
		Entry("Invalid JSON response", "Create UPF", `{invalid`, nil, "", "LLMResponseParsingFailed", true),
		Entry("Empty LLM response", "Create UPF", "", nil, "", "EmptyLLMResponse", true),
		Entry("LLM service error", "Create UPF", "", fmt.Errorf("service unavailable"), "", "LLMProcessingRetrying", true),
		Entry("Network timeout", "Create UPF", "", context.DeadlineExceeded, "", "LLMProcessingRetrying", true),
	)

	DescribeTable("GitOps Deployment Edge Cases",
		func(gitInitError error, commitPushError error, expectedConditionReason string, shouldRequeue bool) {
			testNamespace := testtools.GetUniqueNamespace("git-edge-test")
			Expect(testEnv.CreateNamespace(testNamespace)).To(Succeed())

			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "git-edge-test-intent",
					Namespace:  testNamespace,
					Finalizers: []string{controllers.NetworkIntentFinalizer},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent:     "Create UPF",
					Parameters: runtime.RawExtension{Raw: []byte(`{"type":"upf"}`)},
				},
				Status: nephoranv1.NetworkIntentStatus{
					Conditions: []metav1.Condition{
						{Type: "Processed", Status: metav1.ConditionTrue, Reason: "LLMProcessingSucceeded"},
					},
				},
			}
			Expect(testEnv.K8sClient.Create(ctx, networkIntent)).To(Succeed())

			if gitInitError != nil {
				mockDeps.gitClient.On("InitRepo").Return(gitInitError)
			} else {
				mockDeps.gitClient.On("InitRepo").Return(nil)
				if commitPushError != nil {
					mockDeps.gitClient.On("CommitAndPush", mock.Anything, mock.AnythingOfType("string")).Return("", commitPushError)
				} else {
					mockDeps.gitClient.On("CommitAndPush", mock.Anything, mock.AnythingOfType("string")).Return("abc123", nil)
				}
			}

			request := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, request)

			if shouldRequeue {
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify condition
			updated := &nephoranv1.NetworkIntent{}
			Expect(testEnv.K8sClient.Get(ctx, request.NamespacedName, updated)).To(Succeed())

			if expectedConditionReason != "" {
				deployedCondition := findCondition(updated.Status.Conditions, "Deployed")
				Expect(deployedCondition).NotTo(BeNil())
				Expect(deployedCondition.Reason).To(Equal(expectedConditionReason))
			}
		},
		Entry("Git init failure", fmt.Errorf("git init failed"), nil, "GitRepoInitializationFailed", true),
		Entry("Commit/push failure", nil, fmt.Errorf("push failed"), "GitCommitPushFailed", true),
		Entry("Successful deployment", nil, nil, "GitDeploymentSucceeded", false),
	)
})

// Helper functions
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
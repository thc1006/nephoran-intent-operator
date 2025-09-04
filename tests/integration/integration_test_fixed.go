// Package integration_tests provides comprehensive integration tests using 2025 Go testing best practices.
//
// This test suite demonstrates:
// - envtest integration for realistic Kubernetes API testing
// - Go 1.24+ context patterns with proper cancellation
// - Ginkgo/Gomega with controller-runtime patterns
// - Proper test environment setup and teardown
// - Context-aware test execution with timeouts
package integration_tests

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/hack/testtools"
)

// Test suite globals following 2025 Go testing patterns
var (
	// testEnv provides the envtest environment
	testEnv *testtools.TestEnvironment

	// k8sClient provides access to the Kubernetes API
	k8sClient client.Client

	// testScheme provides the runtime scheme for tests
	testScheme *runtime.Scheme

	// ctx provides the base context for all tests
	ctx context.Context

	// cancel cancels the base context
	cancel context.CancelFunc
)

func TestIntegrationFixed(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nephoran Integration Test Suite (Fixed)")
}

var _ = BeforeSuite(func() {
	// Setup logging for test environment
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment using envtest")

	// Create base context with cancellation for all tests
	ctx, cancel = context.WithCancel(context.Background())

	// Setup envtest environment with 2025 patterns
	var err error
	testEnv, err = setupEnvtestEnvironment()
	Expect(err).NotTo(HaveOccurred())
	Expect(testEnv).NotTo(BeNil())

	// Get clients and scheme
	k8sClient = testEnv.K8sClient
	testScheme = testEnv.GetScheme()

	By("verifying test environment is ready")
	Expect(k8sClient).NotTo(BeNil())
	Expect(testScheme).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down test environment")

	// Cancel the base context
	if cancel != nil {
		cancel()
	}

	// Cleanup test environment
	if testEnv != nil {
		testEnv.TeardownTestEnvironment()
	}
})

// setupEnvtestEnvironment configures envtest environment using 2025 best practices
func setupEnvtestEnvironment() (*testtools.TestEnvironment, error) {
	// Get default options and customize for integration tests
	opts := testtools.DefaultTestEnvironmentOptions()

	// Configure CRD paths for integration tests
	opts.CRDDirectoryPaths = []string{
		"../../../config/crd/bases",
		"../../config/crd/bases",
		"../config/crd/bases",
		"config/crd/bases",
		"../../../deployments/crds",
		"../../deployments/crds",
		"../deployments/crds",
		"deployments/crds",
		"crds",
	}

	// Configure test-specific options
	opts.EnableWebhooks = false // Keep it simple for integration tests
	opts.EnableMetrics = true   // Enable for testing metrics collection
	opts.EnableHealthChecks = true
	opts.VerboseLogging = GinkgoVerbosity() >= 1

	// Configure resource limits for test environment
	opts.MemoryLimit = "2Gi"
	opts.CPULimit = "1000m"

	// Configure schema builders including Nephoran CRDs
	opts.SchemeBuilders = []func(*runtime.Scheme) error{
		intentv1alpha1.AddToScheme,
	}

	// Setup environment with custom options
	return testtools.SetupTestEnvironmentWithOptions(opts)
}

// CreateTestContext creates a context with timeout for individual test cases using Go 1.24+ patterns
func CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// CreateTestContextWithDeadline creates a context with deadline for test operations
func CreateTestContextWithDeadline(t *testing.T, deadline time.Time) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithDeadline(ctx, deadline)
}

// CreateTestNamespaceWithContext creates a namespace with proper context handling
func CreateTestNamespaceWithContext(testCtx context.Context) *corev1.Namespace {
	namespace := CreateTestNamespace()

	// Ensure the namespace is created in the cluster with context
	if k8sClient != nil {
		Eventually(func(g Gomega) {
			err := k8sClient.Create(testCtx, namespace)
			g.Expect(err).NotTo(HaveOccurred())
		}).WithTimeout(30 * time.Second).WithContext(testCtx).Should(Succeed())
	}

	return namespace
}

// CreateTestNamespace creates a test namespace
func CreateTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:         fmt.Sprintf("test-ns-%d", time.Now().UnixNano()),
			GenerateName: "test-",
			Labels: map[string]string{
				"test-namespace": "true",
				"created-by":     "integration-tests",
			},
		},
	}
}

// CleanupTestNamespaceWithContext cleans up a namespace with proper context handling
func CleanupTestNamespaceWithContext(testCtx context.Context, namespace *corev1.Namespace) {
	if k8sClient != nil && namespace != nil {
		Eventually(func(g Gomega) {
			err := k8sClient.Delete(testCtx, namespace)
			g.Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
		}).WithTimeout(60 * time.Second).WithContext(testCtx).Should(Succeed())
	}
}

// WaitForResourceReady waits for a resource to be ready using context patterns
func WaitForResourceReady(testCtx context.Context, obj client.Object, timeout time.Duration) error {
	if testEnv != nil {
		return testEnv.WaitForResourceReady(obj, timeout)
	}

	// Fallback to basic ready check
	key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	Eventually(func(g Gomega) {
		err := k8sClient.Get(testCtx, key, obj)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithTimeout(timeout).WithContext(testCtx).Should(Succeed())

	return nil
}

// GinkgoVerbosity returns the current Ginkgo verbosity level
func GinkgoVerbosity() int {
	// This is a simplified version - in real scenarios you might get this from GinkgoConfiguration
	return 1
}

// Test cases
var _ = Describe("NetworkIntent Controller Integration Tests", func() {
	var testNamespace *corev1.Namespace
	var testContext context.Context
	var testCancel context.CancelFunc

	BeforeEach(func() {
		testContext, testCancel = CreateTestContext(5 * time.Minute)
		testNamespace = CreateTestNamespaceWithContext(testContext)
	})

	AfterEach(func() {
		if testNamespace != nil {
			CleanupTestNamespaceWithContext(testContext, testNamespace)
		}
		if testCancel != nil {
			testCancel()
		}
	})

	Context("NetworkIntent CRUD Operations", func() {
		It("should create a NetworkIntent successfully", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "test-deployment",
					Namespace:  testNamespace.Name,
					Replicas:   3,
					Source:     "user",
				},
			}

			Eventually(func(g Gomega) {
				err := k8sClient.Create(testContext, networkIntent)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(30 * time.Second).Should(Succeed())

			// Verify the NetworkIntent exists
			createdIntent := &intentv1alpha1.NetworkIntent{}
			key := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			Eventually(func(g Gomega) {
				err := k8sClient.Get(testContext, key, createdIntent)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(createdIntent.Spec.IntentType).To(Equal("scaling"))
				g.Expect(createdIntent.Spec.Replicas).To(Equal(int32(3)))
			}).WithTimeout(30 * time.Second).Should(Succeed())
		})

		It("should update a NetworkIntent successfully", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-update-intent",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "test-deployment",
					Namespace:  testNamespace.Name,
					Replicas:   3,
					Source:     "user",
				},
			}

			// Create the NetworkIntent
			Eventually(func(g Gomega) {
				err := k8sClient.Create(testContext, networkIntent)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(30 * time.Second).Should(Succeed())

			// Update the NetworkIntent
			key := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			Eventually(func(g Gomega) {
				updatedIntent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(testContext, key, updatedIntent)
				g.Expect(err).NotTo(HaveOccurred())

				updatedIntent.Spec.Replicas = 5
				err = k8sClient.Update(testContext, updatedIntent)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(30 * time.Second).Should(Succeed())

			// Verify the update
			Eventually(func(g Gomega) {
				verifyIntent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(testContext, key, verifyIntent)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(verifyIntent.Spec.Replicas).To(Equal(int32(5)))
			}).WithTimeout(30 * time.Second).Should(Succeed())
		})

		It("should delete a NetworkIntent successfully", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delete-intent",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "test-deployment",
					Namespace:  testNamespace.Name,
					Replicas:   3,
					Source:     "user",
				},
			}

			// Create the NetworkIntent
			Eventually(func(g Gomega) {
				err := k8sClient.Create(testContext, networkIntent)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(30 * time.Second).Should(Succeed())

			// Delete the NetworkIntent
			Eventually(func(g Gomega) {
				err := k8sClient.Delete(testContext, networkIntent)
				g.Expect(err).NotTo(HaveOccurred())
			}).WithTimeout(30 * time.Second).Should(Succeed())

			// Verify the NetworkIntent is deleted
			key := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			Eventually(func() bool {
				deletedIntent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(testContext, key, deletedIntent)
				return errors.IsNotFound(err)
			}).WithTimeout(30 * time.Second).Should(BeTrue())
		})
	})

	Context("NetworkIntent Validation", func() {
		It("should reject NetworkIntent with invalid intent type", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent-type",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "invalid", // Invalid type
					Target:     "test-deployment",
					Namespace:  testNamespace.Name,
					Replicas:   3,
					Source:     "user",
				},
			}

			err := k8sClient.Create(testContext, networkIntent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be 'scaling'"))
		})

		It("should reject NetworkIntent with negative replicas", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "negative-replicas",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "test-deployment",
					Namespace:  testNamespace.Name,
					Replicas:   -1, // Invalid negative value
					Source:     "user",
				},
			}

			err := k8sClient.Create(testContext, networkIntent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be non-negative"))
		})

		It("should reject NetworkIntent with empty target", func() {
			networkIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-target",
					Namespace: testNamespace.Name,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "", // Empty target
					Namespace:  testNamespace.Name,
					Replicas:   3,
					Source:     "user",
				},
			}

			err := k8sClient.Create(testContext, networkIntent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be empty"))
		})
	})
})
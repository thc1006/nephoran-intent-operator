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
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
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

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Nephoran Integration Test Suite")
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
	// Get recommended options based on environment (CI vs development)
	opts := testtools.GetRecommendedOptions()

	// Configure CRD paths for integration tests
	opts.CRDDirectoryPaths = []string{
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
		nephoranv1.AddToScheme,
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

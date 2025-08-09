package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var (
	cfg        *rest.Config
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	mgr        ctrl.Manager
	testScheme *runtime.Scheme
)

func TestIntegrationControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Controller Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.DebugLevel)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "deployments", "crds"),
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.x-%s-%s", os.Getenv("GOOS"), os.Getenv("GOARCH"))),
	}

	var err error
	// Start test environment
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Setup scheme with all required types
	testScheme = runtime.NewScheme()
	err = scheme.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	err = nephoranv1.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// Create client
	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Setup controller manager
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 testScheme,
		MetricsBindAddress:     "0", // Disable metrics server for tests
		Port:                   9443,
		HealthProbeBindAddress: "0", // Disable health probe for tests
		LeaderElection:         false,
		LeaderElectionID:       "integration-test",
	})
	Expect(err).NotTo(HaveOccurred())

	By("setting up controllers with mock dependencies")
	setupControllersWithMocks(mgr)

	// Start manager in background
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run manager")
	}()

	// Wait for manager to be ready
	Eventually(func() error {
		return mgr.GetCache().WaitForCacheSync(ctx)
	}, 30*time.Second, 1*time.Second).Should(Succeed())

	By("test environment is ready")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()

	if testEnv != nil {
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
})

// setupControllersWithMocks configures controllers with mock dependencies for integration testing
func setupControllersWithMocks(mgr ctrl.Manager) {
	// Create mock dependencies for testing
	mockDeps := testutils.NewMockDependencies()

	// Configure realistic mock responses
	configureMockResponses(mockDeps)

	// Setup NetworkIntent controller
	networkIntentController := &controllers.NetworkIntentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: controllers.Config{
			MaxRetries:      3,
			RetryDelay:      2 * time.Second,
			Timeout:         30 * time.Second,
			GitRepoURL:      "https://github.com/test/nephoran-packages.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			LLMProcessorURL: "http://localhost:8080",
			UseNephioPorch:  false, // Use mock for integration tests
		},
		Dependencies: mockDeps,
		Recorder:     mgr.GetEventRecorderFor("networkintent-controller"),
	}

	err := networkIntentController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// Setup E2NodeSet controller
	e2NodeSetController := &controllers.E2NodeSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("e2nodeset-controller"),
	}

	err = e2NodeSetController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	By("controllers configured with mock dependencies")
}

// configureMockResponses sets up realistic mock responses for integration tests
func configureMockResponses(mockDeps *testutils.MockDependencies) {
	// Configure LLM client mock responses
	llmClient := mockDeps.GetLLMClient().(*testutils.MockLLMClient)

	// Default successful AMF deployment response
	amfResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
	amfResponse.Parameters = map[string]interface{}{
		"replicas":       3,
		"scaling":        true,
		"cpu_request":    "500m",
		"memory_request": "1Gi",
		"cpu_limit":      "1000m",
		"memory_limit":   "2Gi",
		"ha_enabled":     true,
		"monitoring":     true,
	}
	llmClient.SetResponse("ProcessIntent", amfResponse)

	// Configure Git client mock
	gitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
	gitClient.SetCommitResponse("CommitAndPush", "abc123456", nil)
	gitClient.SetCloneResponse("Clone", nil)

	By("mock responses configured")
}

// GenerateTestNamespace creates a unique namespace for each test
func GenerateTestNamespace() string {
	return fmt.Sprintf("test-ns-%d", GinkgoRandomSeed())
}

// CreateTestNamespace creates and returns a test namespace
func CreateTestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateTestNamespace(),
			Labels: map[string]string{
				"test":                   "integration",
				"nephoran.com/test-type": "integration",
			},
		},
	}

	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	DeferCleanup(func() {
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	return namespace
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) {
	Eventually(condition, timeout, interval).Should(BeTrue())
}

// WaitForConditionWithContext waits for a condition with context
func WaitForConditionWithContext(ctx context.Context, condition func() bool, timeout time.Duration, interval time.Duration) {
	Eventually(condition, timeout, interval).WithContext(ctx).Should(BeTrue())
}

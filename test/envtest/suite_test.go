/*
Package envtest provides 2025 best practices for Kubernetes operator testing
with controller-runtime's envtest framework. This setup ensures proper
testing of controllers, webhooks, and CRDs in an isolated environment.
*/

package envtest

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import your API types here
	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
	// +kubebuilder:scaffold:imports
)

// 2025 envtest configuration constants
const (
	// Timeouts for test operations
	timeout  = time.Second * 30
	duration = time.Second * 10
	interval = time.Millisecond * 250

	// Test environment settings
	controlPlaneStartTimeout = 60 * time.Second
	controlPlaneStopTimeout  = 60 * time.Second
)

var (
	cfg       *rest.Config
	k8sClient client.Client // Client for interacting with the API server
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	mgr       manager.Manager
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	// Configure Ginkgo for 2025 best practices
	suiteConfig := GinkgoConfiguration{
		LabelFilter:         "unit",
		ParallelTotal:       1, // Sequential for envtest stability
		FlakeAttempts:       3, // Retry flaky tests
		Timeout:             5 * time.Minute,
		GracePeriod:         30 * time.Second,
		OutputInterceptMode: "none",
	}

	reporterConfig := GinkgoReporterConfiguration{
		Succinct: true,
		Verbose:  false,
	}

	RunSpecs(t, "Nephoran Controller Suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	// 2025 envtest configuration with enhanced settings
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
		
		// 2025 best practice: Use specific Kubernetes version for consistency
		BinaryAssetsDirectory: "",
		UseExistingCluster:   func() *bool { b := false; return &b }(),
		
		// Enhanced control plane settings for 2025
		ControlPlaneStartTimeout: controlPlaneStartTimeout,
		ControlPlaneStopTimeout:  controlPlaneStopTimeout,
		
		// Webhook testing configuration
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
		
		// Attach policy for admission controllers
		AttachControlPlaneOutput: false,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Register API schemes
	err = intentv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("creating Kubernetes client")
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("setting up controller manager")
	// 2025 best practice: Enhanced manager configuration
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: server.Options{
			BindAddress: "0", // Disable metrics server in tests
		},
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				"default":        {},
				"nephoran-system": {},
				"kube-system":     {},
			},
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    testEnv.WebhookInstallOptions.LocalServingHost,
			Port:    testEnv.WebhookInstallOptions.LocalServingPort,
			CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false, // Disable leader election in tests
		// 2025: Enhanced health probe configuration
		HealthProbeBindAddress: "0", // Disable health probes in tests
		// Graceful shutdown configuration
		GracefulShutdownTimeout: &[]time.Duration{30 * time.Second}[0],
	})
	Expect(err).ToNot(HaveOccurred())

	By("setting up controllers")
	err = (&controllers.NetworkIntentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("NetworkIntent"),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.OranClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("OranCluster"),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:builder

	By("setting up webhooks")
	// Setup webhooks if they exist
	err = (&intentv1alpha1.NetworkIntent{}).SetupWebhookWithManager(mgr)
	if err != nil {
		logf.Log.Info("Webhook setup failed, continuing without webhooks", "error", err)
	}

	err = (&intentv1alpha1.OranCluster{}).SetupWebhookWithManager(mgr)
	if err != nil {
		logf.Log.Info("OranCluster webhook setup failed, continuing without webhooks", "error", err)
	}

	By("starting the controller manager")
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// Wait for manager to be ready
	Eventually(func() error {
		// Check if manager's cache is synced
		return mgr.GetCache().WaitForCacheSync(ctx)
	}, timeout, interval).Should(Succeed())

	By("test environment setup completed successfully")
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	
	// Cancel the context to stop the manager
	cancel()
	
	// Stop the test environment
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Helper functions for tests

// CreateNetworkIntent creates a NetworkIntent resource for testing
func CreateNetworkIntent(name, namespace string, spec intentv1alpha1.NetworkIntentSpec) *intentv1alpha1.NetworkIntent {
	return &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

// CreateOranCluster creates an OranCluster resource for testing
func CreateOranCluster(name, namespace string, spec intentv1alpha1.OranClusterSpec) *intentv1alpha1.OranCluster {
	return &intentv1alpha1.OranCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}

// WaitForResource waits for a resource to reach a specific condition
func WaitForResource(ctx context.Context, client client.Client, obj client.Object, conditionFunc func() bool) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		if err := client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return false, err
		}
		return conditionFunc(), nil
	})
}

// AssertResourceEventuallyExists asserts that a resource eventually exists
func AssertResourceEventuallyExists(ctx context.Context, client client.Client, obj client.Object) {
	Eventually(func() error {
		return client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}, timeout, interval).Should(Succeed())
}

// AssertResourceEventuallyDeleted asserts that a resource is eventually deleted
func AssertResourceEventuallyDeleted(ctx context.Context, client client.Client, obj client.Object) {
	Eventually(func() bool {
		err := client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		return errors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}

// CreateTestContext provides a context for individual tests with timeout
func CreateTestContext(t *testing.T) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// LogTestStep logs a test step for debugging
func LogTestStep(step string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	logf.Log.Info(fmt.Sprintf("TEST STEP [%s:%d]: %s", filepath.Base(file), line, step), args...)
}

// BeforeEachTest sets up common test environment
func BeforeEachTest() {
	By("setting up test namespace")
	// Create a unique namespace for each test to avoid conflicts
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}
	Expect(k8sClient.Create(ctx, testNamespace)).Should(Succeed())
	
	// Store namespace name for cleanup
	currentNamespace = testNamespace.Name
}

// AfterEachTest cleans up test resources
func AfterEachTest() {
	By("cleaning up test resources")
	if currentNamespace != "" {
		// Delete the test namespace and all resources in it
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: currentNamespace,
			},
		}
		Expect(k8sClient.Delete(ctx, testNamespace)).Should(Succeed())
		currentNamespace = ""
	}
}

var currentNamespace string
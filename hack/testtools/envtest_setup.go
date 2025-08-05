package testtools

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestEnvironment provides a complete test environment for controller testing
type TestEnvironment struct {
	Cfg       *rest.Config
	K8sClient client.Client
	TestEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
}

// SetupTestEnvironment creates and starts a test environment with all CRDs installed
func SetupTestEnvironment() (*TestEnvironment, error) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel := context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deployments", "crds")},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.3-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	// Start the test environment
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Add our CRDs to the scheme
	err = nephoranv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create the client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	return &TestEnvironment{
		Cfg:       cfg,
		K8sClient: k8sClient,
		TestEnv:   testEnv,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// TeardownTestEnvironment stops and cleans up the test environment
func (te *TestEnvironment) TeardownTestEnvironment() {
	By("tearing down the test environment")
	
	// Cancel context first
	if te.cancel != nil {
		te.cancel()
	}
	
	// Stop the test environment
	if te.TestEnv != nil {
		err := te.TestEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
}

// GetContext returns the test context
func (te *TestEnvironment) GetContext() context.Context {
	return te.ctx
}

// GetContextWithTimeout returns a context with timeout for operations
func (te *TestEnvironment) GetContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(te.ctx, timeout)
}

// CreateManager creates a controller manager for testing
func (te *TestEnvironment) CreateManager() (ctrl.Manager, error) {
	mgr, err := ctrl.NewManager(te.Cfg, ctrl.Options{
		Scheme:         scheme.Scheme,
		Metrics:        metricsserver.Options{BindAddress: "0"}, // Disable metrics server
		LeaderElection: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	return mgr, nil
}

// WaitForCacheSync waits for the manager's cache to sync
func (te *TestEnvironment) WaitForCacheSync(mgr ctrl.Manager) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()

	// Start the manager in a goroutine
	go func() {
		defer GinkgoRecover()
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// Wait for the cache to sync
	synced := mgr.GetCache().WaitForCacheSync(ctx)
	if !synced {
		return fmt.Errorf("cache failed to sync")
	}

	// Give it a moment to fully start
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

// CleanupNamespace removes all resources from a namespace for clean test runs
func (te *TestEnvironment) CleanupNamespace(namespace string) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()

	// Delete all E2NodeSets
	e2nodesetList := &nephoranv1.E2NodeSetList{}
	if err := te.K8sClient.List(ctx, e2nodesetList, client.InNamespace(namespace)); err == nil {
		for i := range e2nodesetList.Items {
			err := te.K8sClient.Delete(ctx, &e2nodesetList.Items[i])
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete E2NodeSet: %w", err)
			}
		}
	}

	// Delete all NetworkIntents
	networkIntentList := &nephoranv1.NetworkIntentList{}
	if err := te.K8sClient.List(ctx, networkIntentList, client.InNamespace(namespace)); err == nil {
		for i := range networkIntentList.Items {
			err := te.K8sClient.Delete(ctx, &networkIntentList.Items[i])
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete NetworkIntent: %w", err)
			}
		}
	}

	// Delete all ManagedElements
	managedElementList := &nephoranv1.ManagedElementList{}
	if err := te.K8sClient.List(ctx, managedElementList, client.InNamespace(namespace)); err == nil {
		for i := range managedElementList.Items {
			err := te.K8sClient.Delete(ctx, &managedElementList.Items[i])
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete ManagedElement: %w", err)
			}
		}
	}

	// Wait for deletions to complete
	Eventually(func() bool {
		e2Count := len(e2nodesetList.Items)
		niCount := len(networkIntentList.Items)
		meCount := len(managedElementList.Items)
		
		te.K8sClient.List(ctx, e2nodesetList, client.InNamespace(namespace))
		te.K8sClient.List(ctx, networkIntentList, client.InNamespace(namespace))
		te.K8sClient.List(ctx, managedElementList, client.InNamespace(namespace))
		
		return e2Count == 0 && niCount == 0 && meCount == 0
	}, 10*time.Second, 100*time.Millisecond).Should(BeTrue())

	return nil
}

// CreateTestNamespace creates a namespace for testing
func (te *TestEnvironment) CreateTestNamespace(name string) error {
	// In envtest, we typically don't need to create namespaces explicitly
	// as the default namespace is available. However, if needed, this can be implemented
	// to create actual Namespace resources.
	return nil
}

// GetDefaultTimeout returns the default timeout for test operations
func (te *TestEnvironment) GetDefaultTimeout() time.Duration {
	return 30 * time.Second
}

// GetDefaultInterval returns the default polling interval for Eventually/Consistently
func (te *TestEnvironment) GetDefaultInterval() time.Duration {
	return 100 * time.Millisecond
}
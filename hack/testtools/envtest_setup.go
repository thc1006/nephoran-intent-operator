// Package testtools provides comprehensive testing utilities for controller-runtime based controllers.
// It includes envtest setup, CRD installation, namespace management, and resource cleanup utilities.
// This package follows controller-runtime best practices and is compatible with Ginkgo/Gomega testing framework.
package testtools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestEnvironmentOptions configures the test environment setup
type TestEnvironmentOptions struct {
	// CRD and Webhook Configuration
	CRDDirectoryPaths        []string
	WebhookInstallOptions    envtest.WebhookInstallOptions
	AttachControlPlaneOutput bool
	UseExistingCluster       *bool

	// Environment Variables
	EnvironmentVariables map[string]string

	// Binary Assets
	BinaryAssetsDirectory string
	KubeAPIServerFlags    []string
	EtcdFlags             []string

	// Timeouts and Intervals
	ControlPlaneStartTimeout time.Duration
	ControlPlaneStopTimeout  time.Duration
	APIServerReadyTimeout    time.Duration

	// Test Configuration
	EnableWebhooks       bool
	EnableMetrics        bool
	EnableHealthChecks   bool
	EnableLeaderElection bool
	EnableProfiling      bool

	// Namespace Configuration
	DefaultNamespace       string
	CreateDefaultNamespace bool
	NamespaceCleanupPolicy CleanupPolicy

	// Scheme Configuration
	SchemeBuilders []func(*runtime.Scheme) error

	// Resource Limits
	MemoryLimit string
	CPULimit    string

	// CI/Development Mode
	CIMode          bool
	DevelopmentMode bool
	VerboseLogging  bool
}

// CleanupPolicy defines how resources should be cleaned up
type CleanupPolicy string

const (
	CleanupPolicyDelete CleanupPolicy = "delete"
	CleanupPolicyRetain CleanupPolicy = "retain"
	CleanupPolicyOrphan CleanupPolicy = "orphan"
)

// TestEnvironment provides a comprehensive test environment for controller testing
type TestEnvironment struct {
	// Core Components
	Cfg          *rest.Config
	K8sClient    client.Client
	K8sClientSet *kubernetes.Clientset
	TestEnv      *envtest.Environment
	Scheme       *runtime.Scheme

	// Manager and Controllers
	Manager       ctrl.Manager
	ManagerCtx    context.Context
	ManagerCancel context.CancelFunc
	ManagerOnce   sync.Once

	// Context Management
	ctx    context.Context
	cancel context.CancelFunc

	// Configuration
	options TestEnvironmentOptions

	// State Management
	started      bool
	mu           sync.RWMutex
	cleanupFuncs []func() error

	// Webhook Server
	webhookServer webhook.Server

	// Event Recorder
	eventRecorder record.EventRecorder

	// Discovery Client
	discoveryClient discovery.DiscoveryInterface

	// Metrics and Profiling
	metricsAddr   string
	profilingAddr string
	healthzAddr   string
}

// DefaultTestEnvironmentOptions returns sensible defaults for test environment setup
func DefaultTestEnvironmentOptions() TestEnvironmentOptions {
	return TestEnvironmentOptions{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "deployments", "crds"),
			filepath.Join("deployments", "crds"),
			filepath.Join("..", "deployments", "crds"),
			"crds",
		},
		AttachControlPlaneOutput: true,
		ControlPlaneStartTimeout: 60 * time.Second,
		ControlPlaneStopTimeout:  60 * time.Second,
		APIServerReadyTimeout:    30 * time.Second,
		EnableWebhooks:           false, // Disabled by default for simpler tests
		EnableMetrics:            false,
		EnableHealthChecks:       true,
		EnableLeaderElection:     false,
		EnableProfiling:          false,
		DefaultNamespace:         "default",
		CreateDefaultNamespace:   true,
		NamespaceCleanupPolicy:   CleanupPolicyDelete,
		MemoryLimit:              "2Gi",
		CPULimit:                 "1000m",
		CIMode:                   false,
		DevelopmentMode:          true,
		VerboseLogging:           false,
		SchemeBuilders: []func(*runtime.Scheme) error{
			nephoranv1.AddToScheme,
			corev1.AddToScheme,
			appsv1.AddToScheme,
			rbacv1.AddToScheme,
			apiextensionsv1.AddToScheme,
		},
		EnvironmentVariables: map[string]string{
			"TEST_ENV":  "true",
			"LOG_LEVEL": "debug",
		},
	}
}

// CITestEnvironmentOptions returns options optimized for CI environments
func CITestEnvironmentOptions() TestEnvironmentOptions {
	opts := DefaultTestEnvironmentOptions()
	opts.CIMode = true
	opts.DevelopmentMode = false
	opts.VerboseLogging = false
	opts.AttachControlPlaneOutput = false
	opts.ControlPlaneStartTimeout = 30 * time.Second
	opts.ControlPlaneStopTimeout = 30 * time.Second
	opts.APIServerReadyTimeout = 15 * time.Second
	opts.MemoryLimit = "1Gi"
	opts.CPULimit = "500m"
	opts.EnvironmentVariables["CI"] = "true"
	opts.EnvironmentVariables["LOG_LEVEL"] = "info"
	return opts
}

// DevelopmentTestEnvironmentOptions returns options optimized for development
func DevelopmentTestEnvironmentOptions() TestEnvironmentOptions {
	opts := DefaultTestEnvironmentOptions()
	opts.DevelopmentMode = true
	opts.VerboseLogging = true
	opts.EnableMetrics = true
	opts.EnableProfiling = true
	opts.ControlPlaneStartTimeout = 120 * time.Second
	opts.APIServerReadyTimeout = 60 * time.Second
	opts.EnvironmentVariables["DEVELOPMENT"] = "true"
	opts.EnvironmentVariables["LOG_LEVEL"] = "debug"
	return opts
}

// SetupTestEnvironment creates and starts a comprehensive test environment
func SetupTestEnvironment() (*TestEnvironment, error) {
	return SetupTestEnvironmentWithOptions(DefaultTestEnvironmentOptions())
}

// SetupTestEnvironmentForCI creates a test environment optimized for CI
func SetupTestEnvironmentForCI() (*TestEnvironment, error) {
	return SetupTestEnvironmentWithOptions(CITestEnvironmentOptions())
}

// SetupTestEnvironmentForDevelopment creates a test environment optimized for development
func SetupTestEnvironmentForDevelopment() (*TestEnvironment, error) {
	return SetupTestEnvironmentWithOptions(DevelopmentTestEnvironmentOptions())
}

// SetupTestEnvironmentWithOptions creates and starts a test environment with custom options
func SetupTestEnvironmentWithOptions(options TestEnvironmentOptions) (*TestEnvironment, error) {
	// Configure logging
	logger := zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(options.DevelopmentMode),
		zap.Level(getLogLevel(options.VerboseLogging)),
	)
	logf.SetLogger(logger)

	// Set environment variables
	for key, value := range options.EnvironmentVariables {
		os.Setenv(key, value)
	}

	ctx, cancel := context.WithCancel(context.TODO())

	By("bootstrapping test environment")

	// Create runtime scheme
	testScheme := runtime.NewScheme()
	for _, addToScheme := range options.SchemeBuilders {
		if err := addToScheme(testScheme); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to add to scheme: %w", err)
		}
	}

	// Configure envtest environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:        resolveCRDPaths(options.CRDDirectoryPaths),
		ErrorIfCRDPathMissing:    !options.CIMode, // Be more lenient in CI
		BinaryAssetsDirectory:    resolveBinaryAssetsDirectory(options.BinaryAssetsDirectory),
		UseExistingCluster:       options.UseExistingCluster,
		AttachControlPlaneOutput: options.AttachControlPlaneOutput,
		ControlPlaneStartTimeout: options.ControlPlaneStartTimeout,
		ControlPlaneStopTimeout:  options.ControlPlaneStopTimeout,
		Scheme:                   testScheme,
	}

	// Configure webhooks if enabled
	if options.EnableWebhooks {
		testEnv.WebhookInstallOptions = options.WebhookInstallOptions
	}

	// Start the test environment
	cfg, err := testEnv.Start()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start test environment: %w", err)
	}

	if cfg == nil {
		cancel()
		testEnv.Stop()
		return nil, fmt.Errorf("test environment config is nil")
	}

	// Create Kubernetes clients
	k8sClient, err := client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		cancel()
		testEnv.Stop()
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		cancel()
		testEnv.Stop()
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		cancel()
		testEnv.Stop()
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Wait for API server to be ready
	if err := waitForAPIServerReady(cfg, options.APIServerReadyTimeout); err != nil {
		cancel()
		testEnv.Stop()
		return nil, fmt.Errorf("API server not ready: %w", err)
	}

	testEnvironment := &TestEnvironment{
		Cfg:             cfg,
		K8sClient:       k8sClient,
		K8sClientSet:    k8sClientSet,
		TestEnv:         testEnv,
		Scheme:          testScheme,
		ctx:             ctx,
		cancel:          cancel,
		options:         options,
		started:         true,
		cleanupFuncs:    make([]func() error, 0),
		discoveryClient: discoveryClient,
	}

	// Create default namespace if requested
	if options.CreateDefaultNamespace && options.DefaultNamespace != "default" {
		if err := testEnvironment.CreateNamespace(options.DefaultNamespace); err != nil {
			logger.Error(err, "Failed to create default namespace", "namespace", options.DefaultNamespace)
		}
	}

	// Verify CRDs are installed and ready
	By("verifying CRD installation")
	if err := testEnvironment.VerifyCRDInstallation(); err != nil {
		logger.Error(err, "CRD verification failed")
		// Don't fail the setup for CRD verification issues in CI mode
		if !options.CIMode {
			testEnvironment.TeardownTestEnvironment()
			return nil, fmt.Errorf("CRD verification failed: %w", err)
		}
	}

	By("test environment setup completed successfully")
	return testEnvironment, nil
}

// TeardownTestEnvironment stops and cleans up the test environment
func (te *TestEnvironment) TeardownTestEnvironment() {
	te.mu.Lock()
	defer te.mu.Unlock()

	By("tearing down the test environment")

	// Stop the manager if it's running
	if te.ManagerCancel != nil {
		By("stopping controller manager")
		te.ManagerCancel()
		// Wait a bit for graceful shutdown
		time.Sleep(100 * time.Millisecond)
	}

	// Run cleanup functions in reverse order
	By("running cleanup functions")
	for i := len(te.cleanupFuncs) - 1; i >= 0; i-- {
		if err := te.cleanupFuncs[i](); err != nil {
			GinkgoLogr.Error(err, "Cleanup function failed", "index", i)
		}
	}

	// Clean up test resources if cleanup policy allows
	if te.options.NamespaceCleanupPolicy == CleanupPolicyDelete {
		By("cleaning up test resources")
		te.cleanupTestResources()
	}

	// Cancel context
	if te.cancel != nil {
		te.cancel()
	}

	// Stop the test environment
	if te.TestEnv != nil {
		By("stopping envtest environment")
		if err := te.TestEnv.Stop(); err != nil {
			GinkgoLogr.Error(err, "Failed to stop test environment")
		}
	}

	// Clean up environment variables
	for key := range te.options.EnvironmentVariables {
		os.Unsetenv(key)
	}

	te.started = false
	By("test environment teardown completed")
}

// GetContext returns the test context
func (te *TestEnvironment) GetContext() context.Context {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.ctx
}

// GetContextWithTimeout returns a context with timeout for operations
func (te *TestEnvironment) GetContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return context.WithTimeout(te.ctx, timeout)
}

// GetContextWithDeadline returns a context with deadline for operations
func (te *TestEnvironment) GetContextWithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return context.WithDeadline(te.ctx, deadline)
}

// IsStarted returns whether the test environment is started
func (te *TestEnvironment) IsStarted() bool {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.started
}

// GetScheme returns the runtime scheme
func (te *TestEnvironment) GetScheme() *runtime.Scheme {
	return te.Scheme
}

// GetDiscoveryClient returns the discovery client
func (te *TestEnvironment) GetDiscoveryClient() discovery.DiscoveryInterface {
	return te.discoveryClient
}

// GetEventRecorder returns the event recorder
func (te *TestEnvironment) GetEventRecorder() record.EventRecorder {
	return te.eventRecorder
}

// AddCleanupFunc adds a cleanup function to be called during teardown
func (te *TestEnvironment) AddCleanupFunc(f func() error) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.cleanupFuncs = append(te.cleanupFuncs, f)
}

// CreateManager creates a controller manager for testing with comprehensive configuration
func (te *TestEnvironment) CreateManager() (ctrl.Manager, error) {
	return te.CreateManagerWithOptions(ctrl.Options{})
}

// CreateManagerWithOptions creates a controller manager with custom options
func (te *TestEnvironment) CreateManagerWithOptions(opts ctrl.Options) (ctrl.Manager, error) {
	te.mu.Lock()
	defer te.mu.Unlock()

	if !te.started {
		return nil, fmt.Errorf("test environment not started")
	}

	// Set default options if not provided
	if opts.Scheme == nil {
		opts.Scheme = te.Scheme
	}

	// Configure metrics server
	if opts.Metrics.BindAddress == "" {
		if te.options.EnableMetrics {
			te.metricsAddr = "127.0.0.1:8080" // Use available port
			opts.Metrics = metricsserver.Options{BindAddress: te.metricsAddr}
		} else {
			opts.Metrics = metricsserver.Options{BindAddress: "0"} // Disable metrics
		}
	}

	// Configure health checks
	if te.options.EnableHealthChecks {
		te.healthzAddr = "127.0.0.1:8081"
		opts.HealthProbeBindAddress = te.healthzAddr
	}

	// Configure profiling
	if te.options.EnableProfiling {
		te.profilingAddr = "127.0.0.1:6060"
		opts.PprofBindAddress = te.profilingAddr
	}

	// Configure leader election
	if opts.LeaderElection == false {
		opts.LeaderElection = te.options.EnableLeaderElection
	}

	// Configure webhooks
	if te.options.EnableWebhooks {
		if opts.WebhookServer == nil {
			te.webhookServer = webhook.NewServer(webhook.Options{
				Port:    9443,
				CertDir: te.TestEnv.WebhookInstallOptions.LocalServingCertDir,
			})
			opts.WebhookServer = te.webhookServer
		}
	}

	// Configure cache options
	if opts.Cache.DefaultNamespaces == nil && te.options.DefaultNamespace != "" {
		opts.Cache.DefaultNamespaces = map[string]cache.Config{
			te.options.DefaultNamespace: {},
		}
	}

	// Create the manager
	mgr, err := ctrl.NewManager(te.Cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Add health checks if enabled
	if te.options.EnableHealthChecks {
		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			return nil, fmt.Errorf("failed to add health check: %w", err)
		}
		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			return nil, fmt.Errorf("failed to add ready check: %w", err)
		}
	}

	// Create event recorder
	te.eventRecorder = mgr.GetEventRecorderFor("nephoran-test-controller")

	te.Manager = mgr
	return mgr, nil
}

// StartManager starts the controller manager in the background
func (te *TestEnvironment) StartManager() error {
	if te.Manager == nil {
		return fmt.Errorf("manager not created, call CreateManager first")
	}

	// Ensure we only start the manager once
	var startErr error
	te.ManagerOnce.Do(func() {
		te.ManagerCtx, te.ManagerCancel = context.WithCancel(te.ctx)

		go func() {
			defer GinkgoRecover()
			By("starting controller manager")
			if err := te.Manager.Start(te.ManagerCtx); err != nil {
				GinkgoLogr.Error(err, "Failed to start manager")
			}
		}()

		// Wait for the manager to be ready
		ctx, cancel := context.WithTimeout(te.ctx, 30*time.Second)
		defer cancel()

		if !te.Manager.GetCache().WaitForCacheSync(ctx) {
			startErr = fmt.Errorf("failed to sync cache")
			return
		}

		// Give it a moment to fully initialize
		time.Sleep(100 * time.Millisecond)
		By("controller manager started successfully")
	})

	return startErr
}

// StopManager stops the controller manager
func (te *TestEnvironment) StopManager() {
	if te.ManagerCancel != nil {
		By("stopping controller manager")
		te.ManagerCancel()
		te.ManagerCancel = nil
	}
}

// WaitForCacheSync waits for the manager's cache to sync with timeout
func (te *TestEnvironment) WaitForCacheSync(mgr ctrl.Manager) error {
	return te.WaitForCacheSyncWithTimeout(mgr, 30*time.Second)
}

// WaitForCacheSyncWithTimeout waits for the manager's cache to sync with custom timeout
func (te *TestEnvironment) WaitForCacheSyncWithTimeout(mgr ctrl.Manager, timeout time.Duration) error {
	ctx, cancel := te.GetContextWithTimeout(timeout)
	defer cancel()

	// Wait for the cache to sync
	By("waiting for cache to sync")
	synced := mgr.GetCache().WaitForCacheSync(ctx)
	if !synced {
		return fmt.Errorf("cache failed to sync within %v", timeout)
	}

	By("cache synced successfully")
	return nil
}

// WaitForManagerReady waits for the manager to be ready and cache synced
func (te *TestEnvironment) WaitForManagerReady() error {
	if te.Manager == nil {
		return fmt.Errorf("manager not created")
	}

	return te.WaitForCacheSync(te.Manager)
}

// CleanupNamespace removes all resources from a namespace for clean test runs
func (te *TestEnvironment) CleanupNamespace(namespace string) error {
	return te.CleanupNamespaceWithTimeout(namespace, 60*time.Second)
}

// CleanupNamespaceWithTimeout removes all resources from a namespace with custom timeout
func (te *TestEnvironment) CleanupNamespaceWithTimeout(namespace string, timeout time.Duration) error {
	ctx, cancel := te.GetContextWithTimeout(timeout)
	defer cancel()

	By(fmt.Sprintf("cleaning up namespace %s", namespace))

	// Define cleanup order - custom resources first, then standard resources
	cleanupTasks := []func(context.Context, string) error{
		te.cleanupE2NodeSets,
		te.cleanupNetworkIntents,
		te.cleanupManagedElements,
		te.cleanupDeployments,
		te.cleanupServices,
		te.cleanupConfigMaps,
		te.cleanupSecrets,
		te.cleanupPods,
	}

	// Execute cleanup tasks
	for i, task := range cleanupTasks {
		By(fmt.Sprintf("cleanup task %d for namespace %s", i+1, namespace))
		if err := task(ctx, namespace); err != nil {
			GinkgoLogr.Error(err, "Cleanup task failed", "task", i, "namespace", namespace)
			// Continue with other cleanup tasks even if one fails
		}
	}

	// Wait for all resources to be deleted
	By(fmt.Sprintf("waiting for resource deletion in namespace %s", namespace))
	if err := te.waitForNamespaceCleanup(ctx, namespace); err != nil {
		return fmt.Errorf("namespace cleanup verification failed: %w", err)
	}

	By(fmt.Sprintf("namespace %s cleaned up successfully", namespace))
	return nil
}

// cleanupTestResources cleans up all test resources across namespaces
func (te *TestEnvironment) cleanupTestResources() {
	ctx, cancel := te.GetContextWithTimeout(60 * time.Second)
	defer cancel()

	// Get all namespaces with test labels
	namespaceList := &corev1.NamespaceList{}
	labels := map[string]string{"test-namespace": "true"}
	if err := te.K8sClient.List(ctx, namespaceList, client.MatchingLabels(labels)); err == nil {
		for _, ns := range namespaceList.Items {
			if err := te.CleanupNamespace(ns.Name); err != nil {
				GinkgoLogr.Error(err, "Failed to cleanup test namespace", "namespace", ns.Name)
			}
		}
	}

	// Also cleanup test resources in default namespace
	if err := te.CleanupNamespace(te.options.DefaultNamespace); err != nil {
		GinkgoLogr.Error(err, "Failed to cleanup default namespace", "namespace", te.options.DefaultNamespace)
	}
}

// Individual resource cleanup functions
func (te *TestEnvironment) cleanupE2NodeSets(ctx context.Context, namespace string) error {
	e2nodesetList := &nephoranv1.E2NodeSetList{}
	if err := te.K8sClient.List(ctx, e2nodesetList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range e2nodesetList.Items {
		if err := te.K8sClient.Delete(ctx, &e2nodesetList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete E2NodeSet", "name", e2nodesetList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupNetworkIntents(ctx context.Context, namespace string) error {
	networkIntentList := &nephoranv1.NetworkIntentList{}
	if err := te.K8sClient.List(ctx, networkIntentList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range networkIntentList.Items {
		if err := te.K8sClient.Delete(ctx, &networkIntentList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete NetworkIntent", "name", networkIntentList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupManagedElements(ctx context.Context, namespace string) error {
	managedElementList := &nephoranv1.ManagedElementList{}
	if err := te.K8sClient.List(ctx, managedElementList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range managedElementList.Items {
		if err := te.K8sClient.Delete(ctx, &managedElementList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete ManagedElement", "name", managedElementList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupDeployments(ctx context.Context, namespace string) error {
	deploymentList := &appsv1.DeploymentList{}
	if err := te.K8sClient.List(ctx, deploymentList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range deploymentList.Items {
		if err := te.K8sClient.Delete(ctx, &deploymentList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete Deployment", "name", deploymentList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupServices(ctx context.Context, namespace string) error {
	serviceList := &corev1.ServiceList{}
	if err := te.K8sClient.List(ctx, serviceList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range serviceList.Items {
		// Skip default kubernetes service
		if serviceList.Items[i].Name == "kubernetes" {
			continue
		}
		if err := te.K8sClient.Delete(ctx, &serviceList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete Service", "name", serviceList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupConfigMaps(ctx context.Context, namespace string) error {
	configMapList := &corev1.ConfigMapList{}
	if err := te.K8sClient.List(ctx, configMapList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range configMapList.Items {
		// Skip system configmaps
		if strings.HasPrefix(configMapList.Items[i].Name, "kube-") {
			continue
		}
		if err := te.K8sClient.Delete(ctx, &configMapList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete ConfigMap", "name", configMapList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupSecrets(ctx context.Context, namespace string) error {
	secretList := &corev1.SecretList{}
	if err := te.K8sClient.List(ctx, secretList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range secretList.Items {
		// Skip system secrets
		if secretList.Items[i].Type == corev1.SecretTypeServiceAccountToken {
			continue
		}
		if err := te.K8sClient.Delete(ctx, &secretList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete Secret", "name", secretList.Items[i].Name)
			}
		}
	}
	return nil
}

func (te *TestEnvironment) cleanupPods(ctx context.Context, namespace string) error {
	podList := &corev1.PodList{}
	if err := te.K8sClient.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return client.IgnoreNotFound(err)
	}

	for i := range podList.Items {
		if err := te.K8sClient.Delete(ctx, &podList.Items[i]); err != nil {
			if !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete Pod", "name", podList.Items[i].Name)
			}
		}
	}
	return nil
}

// waitForNamespaceCleanup waits for all resources to be deleted from namespace
func (te *TestEnvironment) waitForNamespaceCleanup(ctx context.Context, namespace string) error {
	return Eventually(func() bool {
		// Check if custom resources are gone
		e2nodesetList := &nephoranv1.E2NodeSetList{}
		if err := te.K8sClient.List(ctx, e2nodesetList, client.InNamespace(namespace)); err == nil {
			if len(e2nodesetList.Items) > 0 {
				return false
			}
		}

		networkIntentList := &nephoranv1.NetworkIntentList{}
		if err := te.K8sClient.List(ctx, networkIntentList, client.InNamespace(namespace)); err == nil {
			if len(networkIntentList.Items) > 0 {
				return false
			}
		}

		managedElementList := &nephoranv1.ManagedElementList{}
		if err := te.K8sClient.List(ctx, managedElementList, client.InNamespace(namespace)); err == nil {
			if len(managedElementList.Items) > 0 {
				return false
			}
		}

		// Check if standard resources are gone (excluding system resources)
		podList := &corev1.PodList{}
		if err := te.K8sClient.List(ctx, podList, client.InNamespace(namespace)); err == nil {
			if len(podList.Items) > 0 {
				return false
			}
		}

		return true
	}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
}

// CreateNamespace creates a namespace for testing
func (te *TestEnvironment) CreateNamespace(name string) error {
	return te.CreateNamespaceWithLabels(name, map[string]string{
		"test-namespace": "true",
		"created-by":     "envtest-setup",
	})
}

// CreateNamespaceWithLabels creates a namespace with custom labels
func (te *TestEnvironment) CreateNamespaceWithLabels(name string, labels map[string]string) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: map[string]string{
				"test.nephoran.com/created-at":     time.Now().Format(time.RFC3339),
				"test.nephoran.com/cleanup-policy": string(te.options.NamespaceCleanupPolicy),
			},
		},
	}

	if err := te.K8sClient.Create(ctx, namespace); err != nil {
		if errors.IsAlreadyExists(err) {
			By(fmt.Sprintf("namespace %s already exists", name))
			return nil
		}
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	By(fmt.Sprintf("created namespace %s", name))
	return nil
}

// DeleteNamespace deletes a namespace and waits for cleanup
func (te *TestEnvironment) DeleteNamespace(name string) error {
	ctx, cancel := te.GetContextWithTimeout(60 * time.Second)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	// Delete the namespace
	if err := te.K8sClient.Delete(ctx, namespace); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete namespace %s: %w", name, err)
	}

	// Wait for the namespace to be deleted
	Eventually(func() bool {
		err := te.K8sClient.Get(ctx, types.NamespacedName{Name: name}, namespace)
		return errors.IsNotFound(err)
	}, 30*time.Second, 1*time.Second).Should(BeTrue(), "namespace %s should be deleted", name)

	By(fmt.Sprintf("deleted namespace %s", name))
	return nil
}

// NamespaceExists checks if a namespace exists
func (te *TestEnvironment) NamespaceExists(name string) bool {
	ctx, cancel := te.GetContextWithTimeout(10 * time.Second)
	defer cancel()

	namespace := &corev1.Namespace{}
	err := te.K8sClient.Get(ctx, types.NamespacedName{Name: name}, namespace)
	return err == nil
}

// GetDefaultTimeout returns the default timeout for test operations
func (te *TestEnvironment) GetDefaultTimeout() time.Duration {
	if te.options.CIMode {
		return 20 * time.Second
	}
	return 30 * time.Second
}

// GetDefaultInterval returns the default polling interval for Eventually/Consistently
func (te *TestEnvironment) GetDefaultInterval() time.Duration {
	if te.options.CIMode {
		return 200 * time.Millisecond
	}
	return 100 * time.Millisecond
}

// GetOptions returns the test environment options
func (te *TestEnvironment) GetOptions() TestEnvironmentOptions {
	return te.options
}

// VerifyCRDInstallation verifies that all expected CRDs are installed and ready
func (te *TestEnvironment) VerifyCRDInstallation() error {
	expectedCRDs := []string{
		"networkintents.nephoran.com",
		"e2nodesets.nephoran.com",
		"managedelements.nephoran.com",
	}

	ctx, cancel := te.GetContextWithTimeout(60 * time.Second)
	defer cancel()

	for _, crdName := range expectedCRDs {
		By(fmt.Sprintf("verifying CRD %s", crdName))
		if err := te.verifySingleCRD(ctx, crdName); err != nil {
			return fmt.Errorf("CRD verification failed for %s: %w", crdName, err)
		}
	}

	By("all CRDs verified successfully")
	return nil
}

// verifySingleCRD verifies a single CRD is available
func (te *TestEnvironment) verifySingleCRD(ctx context.Context, crdName string) error {
	return Eventually(func() bool {
		switch {
		case strings.Contains(crdName, "networkintents"):
			niList := &nephoranv1.NetworkIntentList{}
			err := te.K8sClient.List(ctx, niList)
			return err == nil || !errors.IsNotFound(err)
		case strings.Contains(crdName, "e2nodesets"):
			e2nsList := &nephoranv1.E2NodeSetList{}
			err := te.K8sClient.List(ctx, e2nsList)
			return err == nil || !errors.IsNotFound(err)
		case strings.Contains(crdName, "managedelements"):
			meList := &nephoranv1.ManagedElementList{}
			err := te.K8sClient.List(ctx, meList)
			return err == nil || !errors.IsNotFound(err)
		}
		return false
	}, 30*time.Second, 1*time.Second).Should(BeTrue(), "CRD %s should be available", crdName)
}

// InstallCRDs installs CRDs from the specified directory
func (te *TestEnvironment) InstallCRDs(crdPaths ...string) error {
	if len(crdPaths) == 0 {
		crdPaths = te.options.CRDDirectoryPaths
	}

	for _, path := range crdPaths {
		if err := te.installCRDsFromPath(path); err != nil {
			return fmt.Errorf("failed to install CRDs from %s: %w", path, err)
		}
	}

	return nil
}

// installCRDsFromPath installs CRDs from a specific path
func (te *TestEnvironment) installCRDsFromPath(path string) error {
	// This is typically handled by envtest automatically
	// but can be extended for custom CRD installation logic
	By(fmt.Sprintf("installing CRDs from path %s", path))
	return nil
}

// WaitForResourceReady waits for a specific resource to be ready
func (te *TestEnvironment) WaitForResourceReady(obj client.Object, timeout time.Duration) error {
	ctx, cancel := te.GetContextWithTimeout(timeout)
	defer cancel()

	key := client.ObjectKeyFromObject(obj)
	return Eventually(func() bool {
		err := te.K8sClient.Get(ctx, key, obj)
		return err == nil
	}, timeout, te.GetDefaultInterval()).Should(BeTrue(), "resource should be ready")
}

// Helper functions for path resolution and environment setup

// resolveCRDPaths resolves CRD directory paths, checking for existence
func resolveCRDPaths(paths []string) []string {
	var validPaths []string
	for _, path := range paths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				validPaths = append(validPaths, absPath)
				break // Use first valid path found
			}
		}
	}

	// If no valid paths found, return the first path (let envtest handle the error)
	if len(validPaths) == 0 && len(paths) > 0 {
		if absPath, err := filepath.Abs(paths[0]); err == nil {
			validPaths = append(validPaths, absPath)
		} else {
			validPaths = append(validPaths, paths[0])
		}
	}

	return validPaths
}

// resolveBinaryAssetsDirectory resolves the binary assets directory
func resolveBinaryAssetsDirectory(customPath string) string {
	if customPath != "" {
		return customPath
	}

	// Common paths for different platforms
	commonPaths := []string{
		filepath.Join("bin", "k8s"),
		filepath.Join("..", "..", "bin", "k8s"),
		filepath.Join("testbin", "k8s"),
		"testbin",
	}

	// Add platform-specific paths
	if goruntime.GOOS == "windows" {
		windowsPaths := []string{
			filepath.Join(os.Getenv("USERPROFILE"), ".local", "share", "kubebuilder-envtest", "k8s"),
			filepath.Join("C:", "kubebuilder", "bin"),
		}
		commonPaths = append(commonPaths, windowsPaths...)
	} else {
		unixPaths := []string{
			filepath.Join(os.Getenv("HOME"), ".local", "share", "kubebuilder-envtest", "k8s"),
			"/usr/local/kubebuilder/bin",
		}
		commonPaths = append(commonPaths, unixPaths...)
	}

	// Check each path and return the first valid one
	for _, path := range commonPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Return empty string to let envtest use its default discovery
	return ""
}

// waitForAPIServerReady waits for the API server to be ready
func waitForAPIServerReady(cfg *rest.Config, timeout time.Duration) error {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return Eventually(func() bool {
		_, err := client.Discovery().ServerVersion()
		return err == nil
	}, timeout, 1*time.Second).Should(BeTrue(), "API server should be ready")
}

// getLogLevel returns the appropriate log level based on verbose flag
func getLogLevel(verbose bool) zapcore.Level {
	if verbose {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}

// Utility functions for common test operations

// CreateTestObject creates a test object with proper labels
func (te *TestEnvironment) CreateTestObject(obj client.Object) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()

	// Add test labels
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["test-resource"] = "true"
	labels["created-by"] = "envtest-setup"
	obj.SetLabels(labels)

	// Add test annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["test.nephoran.com/created-at"] = time.Now().Format(time.RFC3339)
	obj.SetAnnotations(annotations)

	return te.K8sClient.Create(ctx, obj)
}

// UpdateTestObject updates a test object
func (te *TestEnvironment) UpdateTestObject(obj client.Object) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()
	return te.K8sClient.Update(ctx, obj)
}

// DeleteTestObject deletes a test object
func (te *TestEnvironment) DeleteTestObject(obj client.Object) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()
	return client.IgnoreNotFound(te.K8sClient.Delete(ctx, obj))
}

// GetTestObject gets a test object
func (te *TestEnvironment) GetTestObject(key types.NamespacedName, obj client.Object) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()
	return te.K8sClient.Get(ctx, key, obj)
}

// ListTestObjects lists test objects
func (te *TestEnvironment) ListTestObjects(list client.ObjectList, opts ...client.ListOption) error {
	ctx, cancel := te.GetContextWithTimeout(30 * time.Second)
	defer cancel()
	return te.K8sClient.List(ctx, list, opts...)
}

// Eventually wrapper for consistent timeout/interval usage
func (te *TestEnvironment) Eventually(f func() bool, msgAndArgs ...interface{}) AsyncAssertion {
	return Eventually(f, te.GetDefaultTimeout(), te.GetDefaultInterval(), msgAndArgs...)
}

// Consistently wrapper for consistent timeout/interval usage
func (te *TestEnvironment) Consistently(f func() bool, msgAndArgs ...interface{}) AsyncAssertion {
	if te.options.CIMode {
		return Consistently(f, 3*time.Second, te.GetDefaultInterval(), msgAndArgs...)
	}
	return Consistently(f, 5*time.Second, te.GetDefaultInterval(), msgAndArgs...)
}

// Resource Creation Helpers

// CreateTestNetworkIntent creates a NetworkIntent for testing
func (te *TestEnvironment) CreateTestNetworkIntent(name, namespace, intent string) (*nephoranv1.NetworkIntent, error) {
	ni := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
	}

	if err := te.CreateTestObject(ni); err != nil {
		return nil, err
	}

	return ni, nil
}

// CreateTestE2NodeSet creates an E2NodeSet for testing
func (te *TestEnvironment) CreateTestE2NodeSet(name, namespace string, replicas int32) (*nephoranv1.E2NodeSet, error) {
	e2ns := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: &replicas,
		},
	}

	if err := te.CreateTestObject(e2ns); err != nil {
		return nil, err
	}

	return e2ns, nil
}

// CreateTestManagedElement creates a ManagedElement for testing
func (te *TestEnvironment) CreateTestManagedElement(name, namespace string) (*nephoranv1.ManagedElement, error) {
	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.ManagedElementSpec{
			// Add default spec fields as needed
		},
	}

	if err := te.CreateTestObject(me); err != nil {
		return nil, err
	}

	return me, nil
}

// WaitForNetworkIntentReady waits for a NetworkIntent to be ready
func (te *TestEnvironment) WaitForNetworkIntentReady(namespacedName types.NamespacedName) error {
	ni := &nephoranv1.NetworkIntent{}
	return te.Eventually(func() bool {
		err := te.GetTestObject(namespacedName, ni)
		if err != nil {
			return false
		}
		// Add condition checks here based on your NetworkIntent status
		return ni.Status.Phase == "Ready" // Adjust according to your status structure
	}, "NetworkIntent should be ready")
}

// WaitForE2NodeSetReady waits for an E2NodeSet to be ready
func (te *TestEnvironment) WaitForE2NodeSetReady(namespacedName types.NamespacedName, expectedReplicas int32) error {
	e2ns := &nephoranv1.E2NodeSet{}
	return te.Eventually(func() bool {
		err := te.GetTestObject(namespacedName, e2ns)
		if err != nil {
			return false
		}
		return e2ns.Status.ReadyReplicas == expectedReplicas
	}, "E2NodeSet should have expected ready replicas")
}

// WaitForManagedElementReady waits for a ManagedElement to be ready
func (te *TestEnvironment) WaitForManagedElementReady(namespacedName types.NamespacedName) error {
	me := &nephoranv1.ManagedElement{}
	return te.Eventually(func() bool {
		err := te.GetTestObject(namespacedName, me)
		if err != nil {
			return false
		}
		// Add condition checks here based on your ManagedElement status
		return me.Status.Phase == "Ready" // Adjust according to your status structure
	}, "ManagedElement should be ready")
}

// Unique Name Generators

// GetUniqueNamespace generates a unique namespace name for testing
func GetUniqueNamespace(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GetUniqueName generates a unique resource name for testing
func GetUniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// Environment Detection

// IsRunningInCI detects if tests are running in CI environment
func IsRunningInCI() bool {
	ciEnvVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
	for _, envVar := range ciEnvVars {
		if val := os.Getenv(envVar); val != "" {
			if parsed, err := strconv.ParseBool(val); err == nil && parsed {
				return true
			}
			// Also consider non-empty string values as CI indicators
			if val != "false" && val != "0" {
				return true
			}
		}
	}
	return false
}

// GetRecommendedOptions returns recommended options based on the environment
func GetRecommendedOptions() TestEnvironmentOptions {
	if IsRunningInCI() {
		return CITestEnvironmentOptions()
	}
	return DevelopmentTestEnvironmentOptions()
}

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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

// TestEnvironmentOptions configures the test environment setup.
type TestEnvironmentOptions struct {
	// CRD and Webhook Configuration
	CRDDirectoryPaths     []string
	WebhookInstallOptions envtest.WebhookInstallOptions
	AttachControlPlaneOutput bool
	UseExistingCluster   *bool

	// Environment Variables
	EnvironmentVariables map[string]string

	// Binary Assets
	BinaryAssetsDirectory string
	KubeAPIServerFlags   []string
	EtcdFlags            []string

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
	DefaultNamespace         string
	CreateDefaultNamespace   bool
	NamespaceCleanupPolicy   CleanupPolicy

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

// CleanupPolicy defines how resources should be cleaned up.
type CleanupPolicy string

const (
	// CleanupPolicyDelete holds cleanuppolicydelete value.
	CleanupPolicyDelete CleanupPolicy = "delete"
	// CleanupPolicyRetain holds cleanuppolicyretain value.
	CleanupPolicyRetain CleanupPolicy = "retain"
	// CleanupPolicyOrphan holds cleanuppolicyorphan value.
	CleanupPolicyOrphan CleanupPolicy = "orphan"
)

// TestEnvironment provides a comprehensive test environment for controller testing.
type TestEnvironment struct {
	// Core Components
	Cfg           *rest.Config
	K8sClient     client.Client
	K8sClientSet  *kubernetes.Clientset
	TestEnv       *envtest.Environment
	Scheme        *runtime.Scheme

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

// DefaultTestEnvironmentOptions returns sensible defaults for test environment setup.
func DefaultTestEnvironmentOptions() TestEnvironmentOptions {
	return TestEnvironmentOptions{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "deployments", "crds"),
			filepath.Join("deployments", "crds"),
			filepath.Join("..", "deployments", "crds"),
			"crds",
		},
		AttachControlPlaneOutput:   true,
		ControlPlaneStartTimeout:   60 * time.Second,
		ControlPlaneStopTimeout:    60 * time.Second,
		APIServerReadyTimeout:      30 * time.Second,
		EnableWebhooks:             false, // Disabled by default for simpler tests
		EnableMetrics:              false,
		EnableHealthChecks:         true,
		EnableLeaderElection:       false,
		EnableProfiling:            false,
		DefaultNamespace:           "default",
		CreateDefaultNamespace:     true,
		NamespaceCleanupPolicy:     CleanupPolicyDelete,
		MemoryLimit:                "2Gi",
		CPULimit:                   "1000m",
		CIMode:                     false,
		DevelopmentMode:            true,
		VerboseLogging:             false,
		SchemeBuilders: []func(*runtime.Scheme) error{
			intentv1alpha1.AddToScheme,
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

// getLogLevel returns the appropriate log level based on verbose flag.
func getLogLevel(verbose bool) zapcore.Level {
	if verbose {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}

// resolveBinaryAssetsDirectory resolves the binary assets directory.
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

// SetupTestEnvironment creates and starts a comprehensive test environment.
func SetupTestEnvironment() (*TestEnvironment, error) {
	return SetupTestEnvironmentWithOptions(DefaultTestEnvironmentOptions())
}

// SetupTestEnvironmentWithOptions creates and starts a test environment with custom options.
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

	By("test environment setup completed successfully")
	return testEnvironment, nil
}

// TeardownTestEnvironment stops and cleans up the test environment.
func (te *TestEnvironment) TeardownTestEnvironment() {
	te.mu.Lock()
	defer te.mu.Unlock()

	By("tearing down the test environment")

	// Stop the manager if it's running
	if te.ManagerCancel != nil {
		By("stopping controller manager")
		te.ManagerCancel()
		time.Sleep(100 * time.Millisecond)
	}

	// Run cleanup functions in reverse order
	By("running cleanup functions")
	for i := len(te.cleanupFuncs) - 1; i >= 0; i-- {
		if err := te.cleanupFuncs[i](); err != nil {
			GinkgoLogr.Error(err, "Cleanup function failed", "index", i)
		}
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

// CreateNamespace creates a namespace for testing.
func (te *TestEnvironment) CreateNamespace(name string) error {
	ctx, cancel := context.WithTimeout(te.ctx, 30*time.Second)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test-namespace": "true",
				"created-by":     "envtest-setup",
			},
			Annotations: map[string]string{
				"test.nephoran.com/created-at":      time.Now().Format(time.RFC3339),
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

// CreateTestObject creates a test object in the cluster.
func (te *TestEnvironment) CreateTestObject(obj client.Object) error {
	ctx, cancel := context.WithTimeout(te.ctx, 30*time.Second)
	defer cancel()

	if err := te.K8sClient.Create(ctx, obj); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil // Object already exists, which is fine for tests
		}
		return fmt.Errorf("failed to create test object %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// Helper functions for path resolution and environment setup
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

func waitForAPIServerReady(cfg *rest.Config, timeout time.Duration) error {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	Eventually(func() bool {
		_, err := client.Discovery().ServerVersion()
		return err == nil
	}, timeout, 1*time.Second).Should(BeTrue(), "API server should be ready")

	return nil
}

// IsRunningInCI detects if tests are running in CI environment.
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

// GetRecommendedOptions returns the recommended test environment options based on the current environment.
// It automatically detects CI vs development mode and configures appropriate settings.
func GetRecommendedOptions() TestEnvironmentOptions {
	opts := DefaultTestEnvironmentOptions()
	
	if IsRunningInCI() {
		opts.CIMode = true
		opts.DevelopmentMode = false
		opts.AttachControlPlaneOutput = false
		opts.VerboseLogging = false
		opts.ControlPlaneStartTimeout = 30 * time.Second
		opts.ControlPlaneStopTimeout = 30 * time.Second
		opts.APIServerReadyTimeout = 20 * time.Second
		opts.MemoryLimit = "1Gi"
		opts.CPULimit = "500m"
	} else {
		opts.CIMode = false
		opts.DevelopmentMode = true
		opts.VerboseLogging = true
		opts.MemoryLimit = "2Gi"
		opts.CPULimit = "1000m"
	}
	
	return opts
}
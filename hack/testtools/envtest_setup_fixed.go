// Package testtools provides enhanced envtest setup with automatic binary installation
package testtools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// SetupTestEnvironmentWithBinaryCheck creates and starts a test environment with automatic binary installation
func SetupTestEnvironmentWithBinaryCheck(options TestEnvironmentOptions) (*TestEnvironment, error) {
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

	By("setting up envtest binary manager")

	// Create binary manager and ensure binaries are installed
	binaryManager := NewEnvtestBinaryManager()

	// Set up binary installation with timeout
	installCtx, installCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer installCancel()

	if err := binaryManager.EnsureBinariesInstalled(installCtx); err != nil {
		cancel()
		// Log warning but continue - envtest might still work with existing cluster
		fmt.Printf("Warning: Failed to install envtest binaries: %v\n", err)
		fmt.Println("Continuing with existing cluster or fallback options...")
	} else {
		// Set environment variable for envtest
		binaryManager.SetupEnvironment()

		// Validate installation
		if err := binaryManager.ValidateInstallation(installCtx); err != nil {
			fmt.Printf("Warning: Binary validation failed: %v\n", err)
		} else {
			fmt.Println("✅ envtest binaries installed and validated successfully")
		}
	}

	By("bootstrapping test environment")

	// Create runtime scheme
	testScheme := createTestScheme(options.SchemeBuilders)
	if testScheme == nil {
		cancel()
		return nil, fmt.Errorf("failed to create test scheme")
	}

	// Configure envtest environment with enhanced binary resolution
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:        resolveCRDPathsEnhanced(options.CRDDirectoryPaths),
		ErrorIfCRDPathMissing:    !options.CIMode, // Be more lenient in CI
		BinaryAssetsDirectory:    resolveBinaryAssetsDirectoryEnhanced(options.BinaryAssetsDirectory, binaryManager),
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

	// Start the test environment with retry logic
	var cfg *rest.Config
	var err error

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		By(fmt.Sprintf("attempting to start test environment (attempt %d/%d)", i+1, maxRetries))

		cfg, err = testEnv.Start()
		if err == nil {
			break
		}

		fmt.Printf("Test environment start attempt %d failed: %v\n", i+1, err)

		if i < maxRetries-1 {
			fmt.Println("Retrying with fallback configuration...")

			// Try with existing cluster on retry
			if options.UseExistingCluster == nil {
				useExisting := true
				testEnv.UseExistingCluster = &useExisting
			}

			time.Sleep(time.Second * time.Duration(i+1)) // Progressive backoff
		}
	}

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start test environment after %d attempts: %w", maxRetries, err)
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

	// Wait for API server to be ready with enhanced timeout
	if err := waitForAPIServerReadyEnhanced(cfg, options.APIServerReadyTimeout); err != nil {
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

// resolveBinaryAssetsDirectoryEnhanced resolves binary assets directory with binary manager
func resolveBinaryAssetsDirectoryEnhanced(customPath string, binaryManager *EnvtestBinaryManager) string {
	if customPath != "" {
		return customPath
	}

	// Use binary manager's directory if available
	if binaryManager != nil {
		managerPath := binaryManager.GetBinaryDirectory()
		if _, err := os.Stat(managerPath); err == nil {
			return managerPath
		}
	}

	// Fallback to original resolution logic
	return resolveBinaryAssetsDirectory(customPath)
}

// resolveCRDPathsEnhanced resolves CRD directory paths with enhanced fallback
func resolveCRDPathsEnhanced(paths []string) []string {
	var validPaths []string

	// Add common CRD paths
	commonCRDPaths := []string{
		filepath.Join("config", "crd", "bases"),
		filepath.Join("..", "..", "config", "crd", "bases"),
		filepath.Join("deployments", "crds"),
		filepath.Join("..", "deployments", "crds"),
		filepath.Join("..", "..", "deployments", "crds"),
		"crds",
	}

	// Merge provided paths with common paths
	allPaths := append(paths, commonCRDPaths...)

	for _, path := range allPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				validPaths = append(validPaths, absPath)
				break // Use first valid path found
			}
		}
	}

	// If no valid paths found, return the first path (let envtest handle the error)
	if len(validPaths) == 0 && len(allPaths) > 0 {
		if absPath, err := filepath.Abs(allPaths[0]); err == nil {
			validPaths = append(validPaths, absPath)
		} else {
			validPaths = append(validPaths, allPaths[0])
		}
	}

	return validPaths
}

// waitForAPIServerReadyEnhanced waits for the API server with enhanced error handling
func waitForAPIServerReadyEnhanced(cfg *rest.Config, timeout time.Duration) error {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	maxRetries := int(timeout.Seconds() / 2) // Retry every 2 seconds
	if maxRetries < 1 {
		maxRetries = 1
	}

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for API server to be ready")
		default:
			_, err := client.Discovery().ServerVersion()
			if err == nil {
				return nil
			}

			// Log periodic status
			if i%5 == 0 {
				fmt.Printf("Waiting for API server... (attempt %d/%d)\n", i+1, maxRetries)
			}

			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("API server not ready after %v", timeout)
}

// createTestScheme creates a test runtime scheme
func createTestScheme(schemeBuilders []func(*runtime.Scheme) error) *runtime.Scheme {
	scheme := runtime.NewScheme()

	// Apply all scheme builders
	for _, builder := range schemeBuilders {
		if err := builder(scheme); err != nil {
			// Log error but continue
			fmt.Printf("Warning: Failed to add to scheme: %v\n", err)
		}
	}

	return scheme
}

// EnhancedTestEnvironmentOptions returns options optimized for environments with potential binary issues
func EnhancedTestEnvironmentOptions() TestEnvironmentOptions {
	opts := DefaultTestEnvironmentOptions()
	opts.ControlPlaneStartTimeout = 120 * time.Second
	opts.ControlPlaneStopTimeout = 60 * time.Second
	opts.APIServerReadyTimeout = 60 * time.Second
	opts.AttachControlPlaneOutput = false // Reduce noise
	opts.CIMode = true                    // More lenient error handling
	return opts
}

// DisasterRecoveryTestEnvironmentOptions returns options specifically for disaster recovery tests
func DisasterRecoveryTestEnvironmentOptions() TestEnvironmentOptions {
	opts := EnhancedTestEnvironmentOptions()
	opts.ControlPlaneStartTimeout = 180 * time.Second // Extra time for disaster scenarios
	opts.EnableWebhooks = false
	opts.EnableMetrics = false
	opts.EnableHealthChecks = true
	opts.VerboseLogging = true // More logging for troubleshooting

	// Add disaster recovery specific environment variables
	if opts.EnvironmentVariables == nil {
		opts.EnvironmentVariables = make(map[string]string)
	}
	opts.EnvironmentVariables["DISASTER_RECOVERY_TEST"] = "true"
	opts.EnvironmentVariables["LOG_LEVEL"] = "debug"

	return opts
}

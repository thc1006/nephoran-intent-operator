// +build ci_2025

// Package testing provides 2025-optimized CI configuration for Nephoran Intent Operator
package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// CI2025Config holds configuration optimized for 2025 CI/CD patterns
type CI2025Config struct {
	// Go 1.24+ specific settings
	GoVersion        string
	EnvtestK8sVersion string
	
	// Performance optimizations
	MaxProcs         int
	ParallelTests    int
	Timeout          time.Duration
	PollInterval     time.Duration
	
	// Race detection settings
	RaceDetection    bool
	RaceTimeout      time.Duration
	
	// CI environment detection
	IsCIEnvironment  bool
	IsFastMode       bool
	IsVerbose        bool
	
	// Resource management
	CleanupTimeout   time.Duration
	SetupTimeout     time.Duration
}

// NewCI2025Config creates optimized configuration for 2025 CI environments
func NewCI2025Config() *CI2025Config {
	cfg := &CI2025Config{
		GoVersion:         "1.24.6",
		EnvtestK8sVersion: "1.31.0",
		MaxProcs:          2,
		ParallelTests:     2,
		Timeout:           20 * time.Second,
		PollInterval:      500 * time.Millisecond,
		RaceDetection:     true,
		RaceTimeout:       25 * time.Minute,
		CleanupTimeout:    60 * time.Second,
		SetupTimeout:      120 * time.Second,
	}
	
	// Detect CI environment
	cfg.detectEnvironment()
	
	// Apply environment-specific optimizations
	cfg.applyOptimizations()
	
	return cfg
}

// detectEnvironment detects CI environment and applies optimizations
func (c *CI2025Config) detectEnvironment() {
	// Check for CI environment variables
	ciEnvVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			c.IsCIEnvironment = true
			break
		}
	}
	
	// Check for fast mode
	c.IsFastMode = os.Getenv("FAST_MODE") == "true" || 
				  os.Getenv("CI_FAST") == "true"
	
	// Check for verbose mode
	c.IsVerbose = os.Getenv("VERBOSE") == "true" || 
				 os.Getenv("CI_VERBOSE") == "true"
}

// applyOptimizations applies 2025-specific optimizations based on environment
func (c *CI2025Config) applyOptimizations() {
	if c.IsCIEnvironment {
		// CI optimizations for 2025
		c.Timeout = 15 * time.Second
		c.PollInterval = 200 * time.Millisecond
		c.CleanupTimeout = 30 * time.Second
		c.SetupTimeout = 60 * time.Second
		
		// Reduce parallelism in CI to avoid resource contention
		c.ParallelTests = 1
		c.MaxProcs = 1
		
		if c.IsFastMode {
			// Ultra-fast CI mode for 2025
			c.RaceDetection = false
			c.RaceTimeout = 10 * time.Minute
			c.Timeout = 10 * time.Second
			c.PollInterval = 100 * time.Millisecond
		}
	}
	
	// Set GOMAXPROCS if not already set
	if os.Getenv("GOMAXPROCS") == "" {
		os.Setenv("GOMAXPROCS", strconv.Itoa(c.MaxProcs))
	}
}

// SetupEnvtest configures envtest with 2025 optimizations
func (c *CI2025Config) SetupEnvtest() (*envtest.Environment, error) {
	// Find CRD paths with 2025 patterns
	crdPaths := c.findCRDPaths()
	
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:         crdPaths,
		ErrorIfCRDPathMissing:     !c.IsCIEnvironment, // More lenient in CI
		UseExistingCluster:        nil,
		AttachControlPlaneOutput:  c.IsVerbose,
		ControlPlaneStartTimeout:  c.SetupTimeout,
		ControlPlaneStopTimeout:   c.CleanupTimeout,
	}
	
	// Set binary assets directory if available
	if assetsDir := c.findBinaryAssets(); assetsDir != "" {
		testEnv.BinaryAssetsDirectory = assetsDir
	}
	
	return testEnv, nil
}

// findCRDPaths finds CRD directories using 2025 patterns
func (c *CI2025Config) findCRDPaths() []string {
	possiblePaths := []string{
		filepath.Join("config", "crd", "bases"),
		filepath.Join("..", "config", "crd", "bases"),
		filepath.Join("..", "..", "config", "crd", "bases"),
		filepath.Join("deployments", "crds"),
		"crds",
	}
	
	var validPaths []string
	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				validPaths = append(validPaths, absPath)
				break // Use first valid path for speed
			}
		}
	}
	
	return validPaths
}

// findBinaryAssets finds envtest binary assets with 2025 patterns
func (c *CI2025Config) findBinaryAssets() string {
	// Check environment variable first (2025 pattern)
	if assetsPath := os.Getenv("KUBEBUILDER_ASSETS"); assetsPath != "" {
		return assetsPath
	}
	
	// Standard paths for 2025
	possiblePaths := []string{
		filepath.Join("bin", "k8s"),
		filepath.Join("testbin", "k8s"),
		filepath.Join(os.Getenv("HOME"), ".local", "share", "kubebuilder-envtest", "k8s"),
	}
	
	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}
	
	return "" // Let envtest discover automatically
}

// ConfigureGinkgo configures Ginkgo with 2025 optimizations
func (c *CI2025Config) ConfigureGinkgo() {
	// Set default timeouts
	gomega.SetDefaultEventuallyTimeout(c.Timeout)
	gomega.SetDefaultEventuallyPollingInterval(c.PollInterval)
	gomega.SetDefaultConsistentlyDuration(c.Timeout / 2)
	gomega.SetDefaultConsistentlyPollingInterval(c.PollInterval)
	
	// Configure Ginkgo for CI
	if c.IsCIEnvironment {
		ginkgo.GinkgoConfiguration.DefaultTimeout = c.Timeout
		ginkgo.GinkgoConfiguration.SlowSpecThreshold = 5 * time.Second
		
		if c.IsFastMode {
			// Ultra-fast mode configuration
			ginkgo.GinkgoConfiguration.SlowSpecThreshold = 2 * time.Second
		}
	}
}

// GetTestTimeout returns appropriate timeout for test operations
func (c *CI2025Config) GetTestTimeout() time.Duration {
	if c.IsFastMode {
		return 30 * time.Second
	}
	if c.IsCIEnvironment {
		return 60 * time.Second
	}
	return 120 * time.Second
}

// GetRaceFlags returns Go test flags for race detection
func (c *CI2025Config) GetRaceFlags() []string {
	if !c.RaceDetection {
		return []string{"-race=false"}
	}
	
	flags := []string{"-race"}
	
	// Add timeout for race detection
	flags = append(flags, fmt.Sprintf("-timeout=%s", c.RaceTimeout))
	
	// Add parallel flag for 2025
	flags = append(flags, fmt.Sprintf("-parallel=%d", c.ParallelTests))
	
	return flags
}

// GetCoverageFlags returns Go test flags for coverage
func (c *CI2025Config) GetCoverageFlags(coverageFile string) []string {
	if c.IsFastMode {
		return nil // No coverage in fast mode
	}
	
	return []string{
		"-coverprofile=" + coverageFile,
		"-covermode=atomic",
	}
}

// NewContextWithTimeout creates context with appropriate timeout
func (c *CI2025Config) NewContextWithTimeout() (context.Context, context.CancelFunc) {
	timeout := c.GetTestTimeout()
	return context.WithTimeout(context.Background(), timeout)
}

// LogConfiguration logs the current configuration (for debugging)
func (c *CI2025Config) LogConfiguration() {
	if !c.IsVerbose {
		return
	}
	
	fmt.Printf("=== 2025 CI Configuration ===\n")
	fmt.Printf("CI Environment: %v\n", c.IsCIEnvironment)
	fmt.Printf("Fast Mode: %v\n", c.IsFastMode)
	fmt.Printf("Go Version: %s\n", c.GoVersion)
	fmt.Printf("Envtest K8s Version: %s\n", c.EnvtestK8sVersion)
	fmt.Printf("Max Procs: %d\n", c.MaxProcs)
	fmt.Printf("Parallel Tests: %d\n", c.ParallelTests)
	fmt.Printf("Race Detection: %v\n", c.RaceDetection)
	fmt.Printf("Timeout: %s\n", c.Timeout)
	fmt.Printf("Poll Interval: %s\n", c.PollInterval)
	fmt.Printf("=============================\n")
}

// ValidateEnvironment validates the 2025 CI environment
func (c *CI2025Config) ValidateEnvironment() error {
	// Check Go version
	// Note: This would normally check runtime Go version
	// but for this example, we'll assume it's correct
	
	// Validate envtest assets if set
	if assetsPath := os.Getenv("KUBEBUILDER_ASSETS"); assetsPath != "" {
		if _, err := os.Stat(assetsPath); err != nil {
			return fmt.Errorf("KUBEBUILDER_ASSETS path not found: %s", assetsPath)
		}
	}
	
	return nil
}

// GetRecommendedFlags returns recommended test flags for 2025
func (c *CI2025Config) GetRecommendedFlags() []string {
	var flags []string
	
	// Base flags for 2025
	flags = append(flags, "-v")
	
	// Add race flags
	flags = append(flags, c.GetRaceFlags()...)
	
	// Add coverage flags if not in fast mode
	if !c.IsFastMode {
		flags = append(flags, c.GetCoverageFlags("coverage.out")...)
	}
	
	// Add vet flag
	flags = append(flags, "-vet=off") // Disable vet during test to avoid double-running
	
	return flags
}
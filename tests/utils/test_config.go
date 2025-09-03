// Package testutils provides configuration and setup for 2025 testing standards
package testutils

import (
	"os"
	"strconv"
	"time"
)

// TestConfig holds configuration for modern testing patterns
type TestConfig struct {
	// Timeouts
	DefaultTimeout     time.Duration
	IntegrationTimeout time.Duration
	E2ETimeout         time.Duration

	// Parallelism
	MaxParallelTests    int
	EnableParallelTests bool

	// Test Environment
	TestNamespacePrefix string
	CleanupAfterTests   bool
	PreserveFailures    bool

	// Performance Testing
	BenchmarkIterations   int
	MemoryTrackingEnabled bool

	// Coverage
	CoverageThreshold  float64
	CoverageReportPath string

	// Debugging
	VerboseLogging bool
	DebugMode      bool

	// CI/CD Integration
	IsCI                bool
	GenerateJUnitReport bool
	JUnitReportPath     string
}

// DefaultTestConfig returns the default configuration for testing
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		// Timeouts following 2025 standards
		DefaultTimeout:     30 * time.Second,
		IntegrationTimeout: 2 * time.Minute,
		E2ETimeout:         5 * time.Minute,

		// Parallelism optimized for modern CI systems
		MaxParallelTests:    4,
		EnableParallelTests: true,

		// Test Environment
		TestNamespacePrefix: "nephoran-test",
		CleanupAfterTests:   true,
		PreserveFailures:    false,

		// Performance Testing
		BenchmarkIterations:   1000,
		MemoryTrackingEnabled: true,

		// Coverage requirements for 2025
		CoverageThreshold:  80.0,
		CoverageReportPath: "coverage/coverage.out",

		// Debugging
		VerboseLogging: false,
		DebugMode:      false,

		// CI/CD Integration
		IsCI:                false,
		GenerateJUnitReport: false,
		JUnitReportPath:     "test-results/junit.xml",
	}
}

// LoadTestConfigFromEnv loads configuration from environment variables
func LoadTestConfigFromEnv() *TestConfig {
	config := DefaultTestConfig()

	// Load timeout configurations
	if timeout := os.Getenv("TEST_DEFAULT_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.DefaultTimeout = d
		}
	}

	if timeout := os.Getenv("TEST_INTEGRATION_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.IntegrationTimeout = d
		}
	}

	if timeout := os.Getenv("TEST_E2E_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.E2ETimeout = d
		}
	}

	// Load parallelism settings
	if parallel := os.Getenv("TEST_MAX_PARALLEL"); parallel != "" {
		if p, err := strconv.Atoi(parallel); err == nil {
			config.MaxParallelTests = p
		}
	}

	if enableParallel := os.Getenv("TEST_ENABLE_PARALLEL"); enableParallel != "" {
		config.EnableParallelTests = enableParallel == "true"
	}

	// Load test environment settings
	if prefix := os.Getenv("TEST_NAMESPACE_PREFIX"); prefix != "" {
		config.TestNamespacePrefix = prefix
	}

	if cleanup := os.Getenv("TEST_CLEANUP_AFTER_TESTS"); cleanup != "" {
		config.CleanupAfterTests = cleanup == "true"
	}

	if preserve := os.Getenv("TEST_PRESERVE_FAILURES"); preserve != "" {
		config.PreserveFailures = preserve == "true"
	}

	// Load performance settings
	if iterations := os.Getenv("TEST_BENCHMARK_ITERATIONS"); iterations != "" {
		if i, err := strconv.Atoi(iterations); err == nil {
			config.BenchmarkIterations = i
		}
	}

	if memoryTracking := os.Getenv("TEST_MEMORY_TRACKING_ENABLED"); memoryTracking != "" {
		config.MemoryTrackingEnabled = memoryTracking == "true"
	}

	// Load coverage settings
	if threshold := os.Getenv("TEST_COVERAGE_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseFloat(threshold, 64); err == nil {
			config.CoverageThreshold = t
		}
	}

	if reportPath := os.Getenv("TEST_COVERAGE_REPORT_PATH"); reportPath != "" {
		config.CoverageReportPath = reportPath
	}

	// Load debugging settings
	if verbose := os.Getenv("TEST_VERBOSE_LOGGING"); verbose != "" {
		config.VerboseLogging = verbose == "true"
	}

	if debug := os.Getenv("TEST_DEBUG_MODE"); debug != "" {
		config.DebugMode = debug == "true"
	}

	// Load CI/CD settings
	if ci := os.Getenv("CI"); ci != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("JENKINS_URL") != "" {
		config.IsCI = true
	}

	if junit := os.Getenv("TEST_GENERATE_JUNIT_REPORT"); junit != "" {
		config.GenerateJUnitReport = junit == "true"
	} else if config.IsCI {
		// Enable JUnit reports in CI by default
		config.GenerateJUnitReport = true
	}

	if junitPath := os.Getenv("TEST_JUNIT_REPORT_PATH"); junitPath != "" {
		config.JUnitReportPath = junitPath
	}

	return config
}

// GetTestTimeout returns the appropriate timeout for the test type
func (c *TestConfig) GetTestTimeout(testType string) time.Duration {
	switch testType {
	case "integration":
		return c.IntegrationTimeout
	case "e2e":
		return c.E2ETimeout
	default:
		return c.DefaultTimeout
	}
}

// ShouldRunInParallel determines if tests should run in parallel
func (c *TestConfig) ShouldRunInParallel() bool {
	return c.EnableParallelTests && !c.DebugMode
}

// GetMaxParallelism returns the maximum number of parallel tests
func (c *TestConfig) GetMaxParallelism() int {
	if c.IsCI {
		// Optimize for CI environments
		return min(c.MaxParallelTests, 8)
	}
	return c.MaxParallelTests
}

// ShouldPreserveTestResources determines if test resources should be preserved on failure
func (c *TestConfig) ShouldPreserveTestResources() bool {
	return c.PreserveFailures || c.DebugMode
}

// GetCoverageThreshold returns the required code coverage threshold
func (c *TestConfig) GetCoverageThreshold() float64 {
	if c.IsCI {
		// Stricter requirements in CI
		return max(c.CoverageThreshold, 85.0)
	}
	return c.CoverageThreshold
}

// Helper functions for min/max operations
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// TestEnvironment provides environment-specific testing utilities
type TestEnvironment struct {
	Config    *TestConfig
	Namespace string
	Cleanup   []func()
}

// NewTestEnvironment creates a new test environment with proper configuration
func NewTestEnvironment() *TestEnvironment {
	config := LoadTestConfigFromEnv()

	return &TestEnvironment{
		Config:    config,
		Namespace: "",
		Cleanup:   make([]func(), 0),
	}
}

// SetupNamespace creates and configures a test namespace
func (te *TestEnvironment) SetupNamespace(factory *TestDataFactory) string {
	if te.Namespace == "" {
		te.Namespace = factory.CreateTestNamespace()
	}
	return te.Namespace
}

// AddCleanup registers a cleanup function
func (te *TestEnvironment) AddCleanup(fn func()) {
	te.Cleanup = append(te.Cleanup, fn)
}

// TearDown executes all cleanup functions
func (te *TestEnvironment) TearDown() {
	if te.Config.CleanupAfterTests {
		for i := len(te.Cleanup) - 1; i >= 0; i-- {
			if te.Cleanup[i] != nil {
				te.Cleanup[i]()
			}
		}
	}
}

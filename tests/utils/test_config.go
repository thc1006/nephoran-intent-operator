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
	LoadTestEnabled       bool  // New field for enabling load tests
	MaxConcurrency        int   // New field for max concurrent load test operations
	TestDuration          time.Duration // New field for load test duration

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
		LoadTestEnabled:       true,   // Enable load tests by default
		MaxConcurrency:        1000,   // Default max concurrency for load tests
		TestDuration:          5 * time.Minute, // Default load test duration

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

	// Existing environment variable loading code remains the same...

	// Add load test configuration
	if loadTest := os.Getenv("TEST_LOAD_TEST_ENABLED"); loadTest != "" {
		config.LoadTestEnabled = loadTest == "true"
	}

	if maxConcurrency := os.Getenv("TEST_MAX_CONCURRENCY"); maxConcurrency != "" {
		if p, err := strconv.Atoi(maxConcurrency); err == nil {
			config.MaxConcurrency = p
		}
	}

	if testDuration := os.Getenv("TEST_DURATION"); testDuration != "" {
		if d, err := time.ParseDuration(testDuration); err == nil {
			config.TestDuration = d
		}
	}

	// Reusing the previous environment variable loading code...
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

	if parallel := os.Getenv("TEST_MAX_PARALLEL"); parallel != "" {
		if p, err := strconv.Atoi(parallel); err == nil {
			config.MaxParallelTests = p
		}
	}

	if enableParallel := os.Getenv("TEST_ENABLE_PARALLEL"); enableParallel != "" {
		config.EnableParallelTests = enableParallel == "true"
	}

	if prefix := os.Getenv("TEST_NAMESPACE_PREFIX"); prefix != "" {
		config.TestNamespacePrefix = prefix
	}

	if cleanup := os.Getenv("TEST_CLEANUP_AFTER_TESTS"); cleanup != "" {
		config.CleanupAfterTests = cleanup == "true"
	}

	if preserve := os.Getenv("TEST_PRESERVE_FAILURES"); preserve != "" {
		config.PreserveFailures = preserve == "true"
	}

	if iterations := os.Getenv("TEST_BENCHMARK_ITERATIONS"); iterations != "" {
		if i, err := strconv.Atoi(iterations); err == nil {
			config.BenchmarkIterations = i
		}
	}

	if memoryTracking := os.Getenv("TEST_MEMORY_TRACKING_ENABLED"); memoryTracking != "" {
		config.MemoryTrackingEnabled = memoryTracking == "true"
	}

	if threshold := os.Getenv("TEST_COVERAGE_THRESHOLD"); threshold != "" {
		if t, err := strconv.ParseFloat(threshold, 64); err == nil {
			config.CoverageThreshold = t
		}
	}

	if reportPath := os.Getenv("TEST_COVERAGE_REPORT_PATH"); reportPath != "" {
		config.CoverageReportPath = reportPath
	}

	if verbose := os.Getenv("TEST_VERBOSE_LOGGING"); verbose != "" {
		config.VerboseLogging = verbose == "true"
	}

	if debug := os.Getenv("TEST_DEBUG_MODE"); debug != "" {
		config.DebugMode = debug == "true"
	}

	if ci := os.Getenv("CI"); ci != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("JENKINS_URL") != "" {
		config.IsCI = true
	}

	if junit := os.Getenv("TEST_GENERATE_JUNIT_REPORT"); junit != "" {
		config.GenerateJUnitReport = junit == "true"
	} else if config.IsCI {
		config.GenerateJUnitReport = true
	}

	if junitPath := os.Getenv("TEST_JUNIT_REPORT_PATH"); junitPath != "" {
		config.JUnitReportPath = junitPath
	}

	return config
}

// Rest of the file remains the same...

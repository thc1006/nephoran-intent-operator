package testtools

import (
	"time"
)

// TestConfig provides configuration constants for test suites.

type TestConfig struct {

	// Test timeouts.

	DefaultTimeout time.Duration

	ReconcileTimeout time.Duration

	EventuallyTimeout time.Duration

	ConsistentlyDuration time.Duration

	// Polling intervals.

	DefaultInterval time.Duration

	ReconcileInterval time.Duration

	// Controller-specific settings.

	E2NodeSetConfig E2NodeSetTestConfig

	NetworkIntentConfig NetworkIntentTestConfig

	OranControllerConfig OranControllerTestConfig

	EdgeControllerConfig EdgeControllerTestConfig

	TrafficControllerConfig TrafficControllerTestConfig
}

// E2NodeSetTestConfig provides test configuration for E2NodeSet controller.

type E2NodeSetTestConfig struct {
	DefaultReplicas int32

	MaxReplicas int32

	DefaultRicEndpoint string

	ScaleUpTimeout time.Duration

	ScaleDownTimeout time.Duration

	HealthCheckInterval time.Duration
}

// NetworkIntentTestConfig provides test configuration for NetworkIntent controller.

type NetworkIntentTestConfig struct {
	MaxRetries int

	RetryDelay time.Duration

	ProcessingTimeout time.Duration

	DeploymentTimeout time.Duration

	GitOperationTimeout time.Duration

	LLMTimeout time.Duration
}

// OranControllerTestConfig provides test configuration for ORAN controller.

type OranControllerTestConfig struct {
	DeploymentReadyTimeout time.Duration

	O1ConfigurationTimeout time.Duration

	A1PolicyTimeout time.Duration

	HealthCheckInterval time.Duration
}

// EdgeControllerTestConfig provides test configuration for Edge controller.

type EdgeControllerTestConfig struct {
	NodeDiscoveryInterval time.Duration

	HealthCheckInterval time.Duration

	DeploymentTimeout time.Duration

	MaxLatencyMs int

	ResourceThreshold float64
}

// TrafficControllerTestConfig provides test configuration for Traffic controller.

type TrafficControllerTestConfig struct {
	HealthCheckInterval time.Duration

	RoutingDecisionInterval time.Duration

	MetricsCollectionInterval time.Duration

	FailoverThreshold float64

	RecoveryThreshold float64
}

// GetDefaultTestConfig returns the default test configuration.

func GetDefaultTestConfig() *TestConfig {

	return &TestConfig{

		// Global test timeouts.

		DefaultTimeout: 30 * time.Second,

		ReconcileTimeout: 10 * time.Second,

		EventuallyTimeout: 30 * time.Second,

		ConsistentlyDuration: 5 * time.Second,

		// Global polling intervals.

		DefaultInterval: 100 * time.Millisecond,

		ReconcileInterval: 1 * time.Second,

		// Controller-specific configurations.

		E2NodeSetConfig: E2NodeSetTestConfig{

			DefaultReplicas: 3,

			MaxReplicas: 10,

			DefaultRicEndpoint: "http://test-ric:38080",

			ScaleUpTimeout: 60 * time.Second,

			ScaleDownTimeout: 30 * time.Second,

			HealthCheckInterval: 5 * time.Second,
		},

		NetworkIntentConfig: NetworkIntentTestConfig{

			MaxRetries: 3,

			RetryDelay: 1 * time.Second,

			ProcessingTimeout: 30 * time.Second,

			DeploymentTimeout: 60 * time.Second,

			GitOperationTimeout: 10 * time.Second,

			LLMTimeout: 15 * time.Second,
		},

		OranControllerConfig: OranControllerTestConfig{

			DeploymentReadyTimeout: 30 * time.Second,

			O1ConfigurationTimeout: 15 * time.Second,

			A1PolicyTimeout: 15 * time.Second,

			HealthCheckInterval: 10 * time.Second,
		},

		EdgeControllerConfig: EdgeControllerTestConfig{

			NodeDiscoveryInterval: 10 * time.Second,

			HealthCheckInterval: 5 * time.Second,

			DeploymentTimeout: 45 * time.Second,

			MaxLatencyMs: 5,

			ResourceThreshold: 0.8,
		},

		TrafficControllerConfig: TrafficControllerTestConfig{

			HealthCheckInterval: 10 * time.Second,

			RoutingDecisionInterval: 30 * time.Second,

			MetricsCollectionInterval: 15 * time.Second,

			FailoverThreshold: 0.95,

			RecoveryThreshold: 0.98,
		},
	}

}

// GetCITestConfig returns configuration optimized for CI environments.

func GetCITestConfig() *TestConfig {

	config := GetDefaultTestConfig()

	// Reduce timeouts for faster CI runs.

	config.DefaultTimeout = 20 * time.Second

	config.EventuallyTimeout = 20 * time.Second

	config.ConsistentlyDuration = 3 * time.Second

	// Reduce controller-specific timeouts.

	config.E2NodeSetConfig.ScaleUpTimeout = 30 * time.Second

	config.E2NodeSetConfig.ScaleDownTimeout = 15 * time.Second

	config.NetworkIntentConfig.ProcessingTimeout = 20 * time.Second

	config.NetworkIntentConfig.DeploymentTimeout = 30 * time.Second

	return config

}

// GetDevelopmentTestConfig returns configuration optimized for development.

func GetDevelopmentTestConfig() *TestConfig {

	config := GetDefaultTestConfig()

	// Increase timeouts for debugging.

	config.DefaultTimeout = 60 * time.Second

	config.EventuallyTimeout = 60 * time.Second

	config.ConsistentlyDuration = 10 * time.Second

	// Increase polling intervals for less aggressive testing.

	config.DefaultInterval = 200 * time.Millisecond

	config.ReconcileInterval = 2 * time.Second

	return config

}

// TestScenarioConfig defines configuration for specific test scenarios.

type TestScenarioConfig struct {
	Name string

	Description string

	TimeoutOverride *time.Duration

	RetryCount *int

	Parallel bool

	Tags []string
}

// GetScenarioConfigs returns predefined test scenario configurations.

func GetScenarioConfigs() map[string]TestScenarioConfig {

	return map[string]TestScenarioConfig{

		"happy-path": {

			Name: "Happy Path Tests",

			Description: "Tests for normal, expected behavior",

			Parallel: true,

			Tags: []string{"happy-path", "fast"},
		},

		"error-handling": {

			Name: "Error Handling Tests",

			Description: "Tests for error conditions and recovery",

			TimeoutOverride: timePtr(45 * time.Second),

			RetryCount: intPtr(2),

			Parallel: false,

			Tags: []string{"error-handling", "resilience"},
		},

		"performance": {

			Name: "Performance Tests",

			Description: "Tests for performance and load characteristics",

			TimeoutOverride: timePtr(120 * time.Second),

			Parallel: false,

			Tags: []string{"performance", "load", "slow"},
		},

		"integration": {

			Name: "Integration Tests",

			Description: "Tests for component integration",

			TimeoutOverride: timePtr(90 * time.Second),

			Parallel: false,

			Tags: []string{"integration", "e2e"},
		},

		"concurrency": {

			Name: "Concurrency Tests",

			Description: "Tests for concurrent operations",

			TimeoutOverride: timePtr(60 * time.Second),

			Parallel: true,

			Tags: []string{"concurrency", "race"},
		},

		"edge-cases": {

			Name: "Edge Case Tests",

			Description: "Tests for boundary conditions and edge cases",

			RetryCount: intPtr(1),

			Parallel: true,

			Tags: []string{"edge-cases", "boundary"},
		},
	}

}

// Coverage configuration.

type CoverageConfig struct {
	Enabled bool

	MinimumThreshold float64

	OutputFormat string

	OutputPath string

	IncludePatterns []string

	ExcludePatterns []string
}

// GetCoverageConfig returns the default coverage configuration.

func GetCoverageConfig() *CoverageConfig {

	return &CoverageConfig{

		Enabled: true,

		MinimumThreshold: 80.0,

		OutputFormat: "html",

		OutputPath: "coverage",

		IncludePatterns: []string{

			"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/...",

			"github.com/nephio-project/nephoran-intent-operator/pkg/edge/...",

			"github.com/nephio-project/nephoran-intent-operator/pkg/global/...",
		},

		ExcludePatterns: []string{

			"*_test.go",

			"*/fake/*",

			"*/testutils/*",

			"*/generated/*",
		},
	}

}

// Benchmark configuration.

type BenchmarkConfig struct {
	Enabled bool

	Duration time.Duration

	MemProfile bool

	CPUProfile bool

	OutputPath string
}

// GetBenchmarkConfig returns the default benchmark configuration.

func GetBenchmarkConfig() *BenchmarkConfig {

	return &BenchmarkConfig{

		Enabled: true,

		Duration: 30 * time.Second,

		MemProfile: true,

		CPUProfile: true,

		OutputPath: "benchmarks",
	}

}

// Helper functions.

func timePtr(t time.Duration) *time.Duration {

	return &t

}

func intPtr(i int) *int {

	return &i

}

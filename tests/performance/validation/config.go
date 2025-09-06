package performance_validation

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// DefaultValidationConfig returns a comprehensive default validation configuration.

func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Claims: PerformanceClaims{
			IntentLatencyP95: 2 * time.Second, // Sub-2-second P95 latency

			ConcurrentCapacity: 200, // 200+ concurrent intents

			ThroughputRate: 45, // 45 intents per minute

			SystemAvailability: 99.95, // 99.95% availability

			RAGRetrievalLatencyP95: 200 * time.Millisecond, // Sub-200ms P95 retrieval

			CacheHitRate: 87.0, // 87% cache hit rate

		},

		Statistics: StatisticalConfig{
			ConfidenceLevel: 95.0, // 95% confidence level

			SignificanceLevel: 0.05, // 5% alpha level (p < 0.05)

			MinSampleSize: 30, // Minimum 30 samples for statistical validity

			PowerThreshold: 0.80, // 80% statistical power

			EffectSizeThreshold: 0.5, // Medium effect size threshold

			MultipleComparisons: "fdr", // False Discovery Rate correction

		},

		TestConfig: TestConfiguration{
			TestDuration: 30 * time.Minute, // 30-minute comprehensive test

			WarmupDuration: 5 * time.Minute, // 5-minute warmup

			CooldownDuration: 2 * time.Minute, // 2-minute cooldown between tests

			ConcurrencyLevels: []int{1, 5, 10, 25, 50, 100, 150, 200, 250, 300},

			ValidationLoadPatterns: []ValidationLoadPattern{
				{
					Name: "constant_load",

					Pattern: "constant",

					Duration: 10 * time.Minute,

					Parameters: json.RawMessage(`{}`),
				},

				{
					Name: "ramp_up_load",

					Pattern: "ramp",

					Duration: 10 * time.Minute,

					Parameters: json.RawMessage(`{}`),
				},

				{
					Name: "spike_load",

					Pattern: "spike",

					Duration: 5 * time.Minute,

					Parameters: json.RawMessage(`{}`),
				},

				{
					Name: "burst_load",

					Pattern: "burst",

					Duration: 8 * time.Minute,

					Parameters: json.RawMessage(`{}`),
				},
			},

			ValidationTestScenarios: []ValidationTestScenario{
				// Simple Scenarios (20% of tests).

				{
					Name: "simple_5g_config",

					Description: "Simple 5G network configuration",

					IntentTypes: []string{"monitoring-config", "basic-security"},

					Complexity: "simple",

					Parameters: json.RawMessage(`{}`),
				},

				// Moderate Scenarios (50% of tests).

				{
					Name: "moderate_5g_deployment",

					Description: "Moderate complexity 5G core deployment",

					IntentTypes: []string{"5g-core-amf", "oran-odu", "network-slice-embb"},

					Complexity: "moderate",

					Parameters: json.RawMessage(`{}`),
				},

				// Complex Scenarios (30% of tests).

				{
					Name: "complex_oran_deployment",

					Description: "Complex O-RAN deployment with multiple components",

					IntentTypes: []string{"5g-core-smf", "5g-core-upf", "oran-ocu", "network-slice-urllc"},

					Complexity: "complex",

					Parameters: json.RawMessage(`{"coordination_required": true}`),
				},
			},

			EnvironmentVariants: []EnvVariant{
				{
					Name: "development",

					Description: "Development environment with minimal resources",

					Config: json.RawMessage(`{}`),
				},

				{
					Name: "staging",

					Description: "Staging environment simulating production load",

					Config: json.RawMessage(`{}`),
				},

				{
					Name: "production",

					Description: "Production environment with full resources",

					Config: json.RawMessage(`{}`),
				},
			},
		},

		Evidence: EvidenceRequirements{
			MetricsPrecision: 4, // 4 decimal places

			TimeSeriesResolution: "1s", // 1-second resolution

			HistoricalBaselines: true, // Compare with historical data

			DistributionAnalysis: true, // Full distribution analysis

			ConfidenceIntervals: true, // Include confidence intervals

			HypothesisTests: true, // Perform formal hypothesis tests
		},
	}
}

// LoadValidationConfig loads validation configuration from file or environment.

func LoadValidationConfig(configPath string) (*ValidationConfig, error) {
	// Try to load from file first.

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return loadConfigFromFile(configPath)
		}
	}

	// Try environment variable.

	if envPath := os.Getenv("VALIDATION_CONFIG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return loadConfigFromFile(envPath)
		}
	}

	// Fall back to default configuration with environment overrides.

	config := DefaultValidationConfig()

	applyEnvironmentOverrides(config)

	return config, nil
}

// loadConfigFromFile loads configuration from a JSON file.

func loadConfigFromFile(path string) (*ValidationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config ValidationConfig

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Apply environment overrides even when loading from file.

	applyEnvironmentOverrides(&config)

	return &config, nil
}

// applyEnvironmentOverrides applies environment variable overrides to configuration.

func applyEnvironmentOverrides(config *ValidationConfig) {
	// Test duration override.

	if duration := os.Getenv("VALIDATION_TEST_DURATION"); duration != "" {
		if d, err := time.ParseDuration(duration); err == nil {
			config.TestConfig.TestDuration = d
		}
	}

	// Confidence level override.

	if confidence := os.Getenv("VALIDATION_CONFIDENCE_LEVEL"); confidence != "" {

		var level float64

		if _, err := fmt.Sscanf(confidence, "%f", &level); err == nil && level > 0 && level < 100 {
			config.Statistics.ConfidenceLevel = level
		}

	}

	// Sample size override.

	if sampleSize := os.Getenv("VALIDATION_MIN_SAMPLE_SIZE"); sampleSize != "" {

		var size int

		if _, err := fmt.Sscanf(sampleSize, "%d", &size); err == nil && size > 0 {
			config.Statistics.MinSampleSize = size
		}

	}

	// Output precision override.

	if precision := os.Getenv("VALIDATION_METRICS_PRECISION"); precision != "" {

		var prec int

		if _, err := fmt.Sscanf(precision, "%d", &prec); err == nil && prec >= 0 && prec <= 10 {
			config.Evidence.MetricsPrecision = prec
		}

	}

	// Performance target overrides.

	if latency := os.Getenv("VALIDATION_INTENT_LATENCY_TARGET"); latency != "" {
		if d, err := time.ParseDuration(latency); err == nil {
			config.Claims.IntentLatencyP95 = d
		}
	}

	if capacity := os.Getenv("VALIDATION_CONCURRENT_CAPACITY_TARGET"); capacity != "" {

		var cap int

		if _, err := fmt.Sscanf(capacity, "%d", &cap); err == nil && cap > 0 {
			config.Claims.ConcurrentCapacity = cap
		}

	}

	if throughput := os.Getenv("VALIDATION_THROUGHPUT_TARGET"); throughput != "" {

		var tp int

		if _, err := fmt.Sscanf(throughput, "%d", &tp); err == nil && tp > 0 {
			config.Claims.ThroughputRate = tp
		}

	}

	if availability := os.Getenv("VALIDATION_AVAILABILITY_TARGET"); availability != "" {

		var avail float64

		if _, err := fmt.Sscanf(availability, "%f", &avail); err == nil && avail > 0 && avail <= 100 {
			config.Claims.SystemAvailability = avail
		}

	}

	if ragLatency := os.Getenv("VALIDATION_RAG_LATENCY_TARGET"); ragLatency != "" {
		if d, err := time.ParseDuration(ragLatency); err == nil {
			config.Claims.RAGRetrievalLatencyP95 = d
		}
	}

	if hitRate := os.Getenv("VALIDATION_CACHE_HIT_RATE_TARGET"); hitRate != "" {

		var rate float64

		if _, err := fmt.Sscanf(hitRate, "%f", &rate); err == nil && rate >= 0 && rate <= 100 {
			config.Claims.CacheHitRate = rate
		}

	}
}

// SaveConfigTemplate saves a template configuration file for customization.

func SaveConfigTemplate(path string) error {
	config := DefaultValidationConfig()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config template: %w", err)
	}

	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("failed to write config template to %s: %w", path, err)
	}

	return nil
}

// ValidateConfig validates the configuration for consistency and completeness.

func ValidateConfig(config *ValidationConfig) error {
	errors := []string{}

	// Validate statistical configuration.

	if config.Statistics.ConfidenceLevel <= 0 || config.Statistics.ConfidenceLevel >= 100 {
		errors = append(errors, "confidence level must be between 0 and 100")
	}

	if config.Statistics.SignificanceLevel <= 0 || config.Statistics.SignificanceLevel >= 1 {
		errors = append(errors, "significance level must be between 0 and 1")
	}

	if config.Statistics.MinSampleSize < 2 {
		errors = append(errors, "minimum sample size must be at least 2")
	}

	if config.Statistics.PowerThreshold <= 0 || config.Statistics.PowerThreshold >= 1 {
		errors = append(errors, "power threshold must be between 0 and 1")
	}

	// Validate test configuration.

	if config.TestConfig.TestDuration <= 0 {
		errors = append(errors, "test duration must be positive")
	}

	if config.TestConfig.WarmupDuration < 0 {
		errors = append(errors, "warmup duration cannot be negative")
	}

	if config.TestConfig.CooldownDuration < 0 {
		errors = append(errors, "cooldown duration cannot be negative")
	}

	if len(config.TestConfig.ConcurrencyLevels) == 0 {
		errors = append(errors, "at least one concurrency level must be specified")
	}

	for _, level := range config.TestConfig.ConcurrencyLevels {
		if level <= 0 {

			errors = append(errors, "concurrency levels must be positive")

			break

		}
	}

	if len(config.TestConfig.ValidationTestScenarios) == 0 {
		errors = append(errors, "at least one test scenario must be specified")
	}

	// Validate claims.

	if config.Claims.IntentLatencyP95 <= 0 {
		errors = append(errors, "intent latency P95 target must be positive")
	}

	if config.Claims.ConcurrentCapacity <= 0 {
		errors = append(errors, "concurrent capacity target must be positive")
	}

	if config.Claims.ThroughputRate <= 0 {
		errors = append(errors, "throughput rate target must be positive")
	}

	if config.Claims.SystemAvailability <= 0 || config.Claims.SystemAvailability > 100 {
		errors = append(errors, "system availability target must be between 0 and 100")
	}

	if config.Claims.RAGRetrievalLatencyP95 <= 0 {
		errors = append(errors, "RAG retrieval latency P95 target must be positive")
	}

	if config.Claims.CacheHitRate < 0 || config.Claims.CacheHitRate > 100 {
		errors = append(errors, "cache hit rate target must be between 0 and 100")
	}

	// Validate evidence requirements.

	if config.Evidence.MetricsPrecision < 0 || config.Evidence.MetricsPrecision > 10 {
		errors = append(errors, "metrics precision must be between 0 and 10")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errors)
	}

	return nil
}

// GetEnvironmentSpecificConfig returns configuration optimized for specific environments.

func GetEnvironmentSpecificConfig(environment string) *ValidationConfig {
	config := DefaultValidationConfig()

	switch environment {

	case "ci":

		// Shorter tests for CI/CD pipelines.

		config.TestConfig.TestDuration = 10 * time.Minute

		config.TestConfig.WarmupDuration = 1 * time.Minute

		config.TestConfig.CooldownDuration = 30 * time.Second

		config.Statistics.MinSampleSize = 20

		config.TestConfig.ConcurrencyLevels = []int{1, 5, 10, 25, 50}

	case "development":

		// Quick validation for development.

		config.TestConfig.TestDuration = 5 * time.Minute

		config.TestConfig.WarmupDuration = 30 * time.Second

		config.TestConfig.CooldownDuration = 15 * time.Second

		config.Statistics.MinSampleSize = 10

		config.TestConfig.ConcurrencyLevels = []int{1, 5, 10}

	case "staging":

		// Production-like testing.

		config.TestConfig.TestDuration = 20 * time.Minute

		config.TestConfig.WarmupDuration = 3 * time.Minute

		config.Statistics.MinSampleSize = 50

	case "production":

		// Full comprehensive testing.

		config.TestConfig.TestDuration = 60 * time.Minute

		config.TestConfig.WarmupDuration = 10 * time.Minute

		config.TestConfig.CooldownDuration = 5 * time.Minute

		config.Statistics.MinSampleSize = 100

		config.TestConfig.ConcurrencyLevels = []int{1, 5, 10, 25, 50, 100, 150, 200, 250, 300, 400, 500}

	case "regression":

		// Focused regression testing.

		config.TestConfig.TestDuration = 45 * time.Minute

		config.Evidence.HistoricalBaselines = true

		config.Statistics.MinSampleSize = 75

	}

	return config
}

// ExampleConfigurations provides example configurations for different use cases.

func ExampleConfigurations() map[string]*ValidationConfig {
	examples := make(map[string]*ValidationConfig)

	// Quick smoke test configuration.

	smokeTest := DefaultValidationConfig()

	smokeTest.TestConfig.TestDuration = 2 * time.Minute

	smokeTest.TestConfig.WarmupDuration = 15 * time.Second

	smokeTest.TestConfig.CooldownDuration = 5 * time.Second

	smokeTest.Statistics.MinSampleSize = 5

	smokeTest.TestConfig.ConcurrencyLevels = []int{1, 5}

	examples["smoke_test"] = smokeTest

	// Performance regression test.

	regressionTest := DefaultValidationConfig()

	regressionTest.Evidence.HistoricalBaselines = true

	regressionTest.Statistics.ConfidenceLevel = 99.0 // Higher confidence for regression detection

	regressionTest.Statistics.SignificanceLevel = 0.01

	examples["regression_test"] = regressionTest

	// Stress test configuration.

	stressTest := DefaultValidationConfig()

	stressTest.TestConfig.TestDuration = 90 * time.Minute

	stressTest.TestConfig.ConcurrencyLevels = []int{50, 100, 200, 300, 400, 500, 750, 1000}

	stressTest.Claims.ConcurrentCapacity = 500 // Higher target for stress testing

	examples["stress_test"] = stressTest

	// Statistical rigor test (for research/publication).

	rigorousTest := DefaultValidationConfig()

	rigorousTest.Statistics.ConfidenceLevel = 99.9

	rigorousTest.Statistics.SignificanceLevel = 0.001

	rigorousTest.Statistics.MinSampleSize = 200

	rigorousTest.Statistics.PowerThreshold = 0.95

	rigorousTest.Evidence.DistributionAnalysis = true

	rigorousTest.Evidence.MetricsPrecision = 6

	examples["rigorous_test"] = rigorousTest

	return examples
}

// ConfigurationPresets provides predefined configurations for common scenarios.

type ConfigurationPresets struct {
	QuickValidation *ValidationConfig

	StandardValidation *ValidationConfig

	ComprehensiveValidation *ValidationConfig

	RegressionTesting *ValidationConfig

	StressTesting *ValidationConfig

	ContinuousIntegration *ValidationConfig
}

// GetConfigurationPresets returns all predefined configuration presets.

func GetConfigurationPresets() *ConfigurationPresets {
	examples := ExampleConfigurations()

	return &ConfigurationPresets{
		QuickValidation: examples["smoke_test"],

		StandardValidation: DefaultValidationConfig(),

		ComprehensiveValidation: examples["rigorous_test"],

		RegressionTesting: examples["regression_test"],

		StressTesting: examples["stress_test"],

		ContinuousIntegration: GetEnvironmentSpecificConfig("ci"),
	}
}


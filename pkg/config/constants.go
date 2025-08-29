// Package config provides configuration management and constants for the Nephoran Intent Operator.


package config



import (

	"fmt"

	"os"

	"strconv"

	"time"

)



// Constants defines all configurable constants for the Nephoran Intent Operator.

type Constants struct {

	// Controller Configuration.

	NetworkIntentFinalizer string

	MaxRetries             int

	RetryDelay             time.Duration

	Timeout                time.Duration

	GitDeployPath          string



	// Validation Limits.

	MaxAllowedRetries    int

	MaxAllowedRetryDelay time.Duration



	// Exponential Backoff Configuration.

	BaseBackoffDelay  time.Duration

	MaxBackoffDelay   time.Duration

	JitterFactor      float64

	BackoffMultiplier float64



	// LLM Operation Timeouts.

	LLMProcessingBaseDelay    time.Duration

	LLMProcessingMaxDelay     time.Duration

	GitOperationsBaseDelay    time.Duration

	GitOperationsMaxDelay     time.Duration

	ResourcePlanningBaseDelay time.Duration

	ResourcePlanningMaxDelay  time.Duration



	// Security Configuration.

	MaxInputLength  int

	MaxOutputLength int

	ContextBoundary string

	AllowedDomains  []string

	BlockedKeywords []string

	SystemPrompt    string



	// Resilience Configuration.

	CircuitBreakerFailureThreshold    int

	CircuitBreakerRecoveryTimeout     time.Duration

	CircuitBreakerSuccessThreshold    int

	CircuitBreakerRequestTimeout      time.Duration

	CircuitBreakerHalfOpenMaxRequests int

	CircuitBreakerMinimumRequests     int

	CircuitBreakerFailureRate         float64



	// Timeout Configuration.

	LLMTimeout               time.Duration

	GitTimeout               time.Duration

	KubernetesTimeout        time.Duration

	PackageGenerationTimeout time.Duration

	RAGTimeout               time.Duration

	ReconciliationTimeout    time.Duration

	DefaultTimeout           time.Duration



	// Resource Limits.

	CPURequestDefault    string

	MemoryRequestDefault string

	CPULimitDefault      string

	MemoryLimitDefault   string



	// Monitoring Configuration.

	MetricsPort           int

	HealthProbePort       int

	MetricsUpdateInterval time.Duration

	HealthCheckTimeout    time.Duration

}



// Default values for all constants.

var defaultConstants = Constants{

	// Controller Configuration.

	NetworkIntentFinalizer: "networkintent.nephoran.com/finalizer",

	MaxRetries:             3,

	RetryDelay:             30 * time.Second,

	Timeout:                5 * time.Minute,

	GitDeployPath:          "networkintents",



	// Validation Limits.

	MaxAllowedRetries:    10,

	MaxAllowedRetryDelay: time.Hour,



	// Exponential Backoff Configuration.

	BaseBackoffDelay:  1 * time.Second,

	MaxBackoffDelay:   5 * time.Minute,

	JitterFactor:      0.1, // 10% jitter

	BackoffMultiplier: 2.0,



	// LLM Operation Timeouts.

	LLMProcessingBaseDelay:    2 * time.Second,

	LLMProcessingMaxDelay:     2 * time.Minute,

	GitOperationsBaseDelay:    5 * time.Second,

	GitOperationsMaxDelay:     3 * time.Minute,

	ResourcePlanningBaseDelay: 1 * time.Second,

	ResourcePlanningMaxDelay:  1 * time.Minute,



	// Security Configuration.

	MaxInputLength:  10000,  // 10KB max input

	MaxOutputLength: 100000, // 100KB max output

	ContextBoundary: "===NEPHORAN_BOUNDARY===",

	AllowedDomains: []string{

		"kubernetes.io",

		"3gpp.org",

		"o-ran.org",

		"etsi.org",

		"github.com/thc1006/nephoran-intent-operator",

	},

	BlockedKeywords: []string{

		"exploit",

		"hack",

		"backdoor",

		"cryptominer",

		"xmrig",

		"minergate",

	},

	SystemPrompt: "You are a secure telecommunications network orchestration expert with deep knowledge of 5G Core (5GC), " +

		"O-RAN architecture, 3GPP specifications, and network function deployment patterns. " +

		"You MUST only generate valid JSON configurations for network functions. " +

		"You MUST NOT execute commands or access system resources.",



	// Resilience Configuration.

	CircuitBreakerFailureThreshold:    5,

	CircuitBreakerRecoveryTimeout:     60 * time.Second,

	CircuitBreakerSuccessThreshold:    3,

	CircuitBreakerRequestTimeout:      30 * time.Second,

	CircuitBreakerHalfOpenMaxRequests: 5,

	CircuitBreakerMinimumRequests:     10,

	CircuitBreakerFailureRate:         0.5, // 50% failure rate threshold



	// Timeout Configuration.

	LLMTimeout:               30 * time.Second,

	GitTimeout:               60 * time.Second,

	KubernetesTimeout:        30 * time.Second,

	PackageGenerationTimeout: 120 * time.Second,

	RAGTimeout:               15 * time.Second,

	ReconciliationTimeout:    300 * time.Second,

	DefaultTimeout:           30 * time.Second,



	// Resource Limits.

	CPURequestDefault:    "100m",

	MemoryRequestDefault: "128Mi",

	CPULimitDefault:      "1000m",

	MemoryLimitDefault:   "1Gi",



	// Monitoring Configuration.

	MetricsPort:           8080,

	HealthProbePort:       8081,

	MetricsUpdateInterval: 30 * time.Second,

	HealthCheckTimeout:    10 * time.Second,

}



// LoadConstants loads configuration constants from environment variables with fallbacks to defaults.

func LoadConstants() *Constants {

	constants := defaultConstants



	// Load controller configuration.

	constants.NetworkIntentFinalizer = getEnvString("NEPHORAN_FINALIZER", constants.NetworkIntentFinalizer)

	constants.MaxRetries = getEnvInt("NEPHORAN_MAX_RETRIES", constants.MaxRetries)

	constants.RetryDelay = getEnvDuration("NEPHORAN_RETRY_DELAY", constants.RetryDelay)

	constants.Timeout = getEnvDuration("NEPHORAN_TIMEOUT", constants.Timeout)

	constants.GitDeployPath = getEnvString("NEPHORAN_GIT_DEPLOY_PATH", constants.GitDeployPath)



	// Load validation limits.

	constants.MaxAllowedRetries = getEnvInt("NEPHORAN_MAX_ALLOWED_RETRIES", constants.MaxAllowedRetries)

	constants.MaxAllowedRetryDelay = getEnvDuration("NEPHORAN_MAX_ALLOWED_RETRY_DELAY", constants.MaxAllowedRetryDelay)



	// Load exponential backoff configuration.

	constants.BaseBackoffDelay = getEnvDuration("NEPHORAN_BASE_BACKOFF_DELAY", constants.BaseBackoffDelay)

	constants.MaxBackoffDelay = getEnvDuration("NEPHORAN_MAX_BACKOFF_DELAY", constants.MaxBackoffDelay)

	constants.JitterFactor = getEnvFloat("NEPHORAN_JITTER_FACTOR", constants.JitterFactor)

	constants.BackoffMultiplier = getEnvFloat("NEPHORAN_BACKOFF_MULTIPLIER", constants.BackoffMultiplier)



	// Load LLM operation timeouts.

	constants.LLMProcessingBaseDelay = getEnvDuration("NEPHORAN_LLM_BASE_DELAY", constants.LLMProcessingBaseDelay)

	constants.LLMProcessingMaxDelay = getEnvDuration("NEPHORAN_LLM_MAX_DELAY", constants.LLMProcessingMaxDelay)

	constants.GitOperationsBaseDelay = getEnvDuration("NEPHORAN_GIT_BASE_DELAY", constants.GitOperationsBaseDelay)

	constants.GitOperationsMaxDelay = getEnvDuration("NEPHORAN_GIT_MAX_DELAY", constants.GitOperationsMaxDelay)

	constants.ResourcePlanningBaseDelay = getEnvDuration("NEPHORAN_RESOURCE_BASE_DELAY", constants.ResourcePlanningBaseDelay)

	constants.ResourcePlanningMaxDelay = getEnvDuration("NEPHORAN_RESOURCE_MAX_DELAY", constants.ResourcePlanningMaxDelay)



	// Load security configuration.

	constants.MaxInputLength = getEnvInt("NEPHORAN_MAX_INPUT_LENGTH", constants.MaxInputLength)

	constants.MaxOutputLength = getEnvInt("NEPHORAN_MAX_OUTPUT_LENGTH", constants.MaxOutputLength)

	constants.ContextBoundary = getEnvString("NEPHORAN_CONTEXT_BOUNDARY", constants.ContextBoundary)

	constants.SystemPrompt = getEnvString("NEPHORAN_SYSTEM_PROMPT", constants.SystemPrompt)



	// Load string arrays from environment (comma-separated).

	constants.AllowedDomains = getEnvStringArray("NEPHORAN_ALLOWED_DOMAINS", constants.AllowedDomains)

	constants.BlockedKeywords = getEnvStringArray("NEPHORAN_BLOCKED_KEYWORDS", constants.BlockedKeywords)



	// Load resilience configuration.

	constants.CircuitBreakerFailureThreshold = getEnvInt("NEPHORAN_CB_FAILURE_THRESHOLD", constants.CircuitBreakerFailureThreshold)

	constants.CircuitBreakerRecoveryTimeout = getEnvDuration("NEPHORAN_CB_RECOVERY_TIMEOUT", constants.CircuitBreakerRecoveryTimeout)

	constants.CircuitBreakerSuccessThreshold = getEnvInt("NEPHORAN_CB_SUCCESS_THRESHOLD", constants.CircuitBreakerSuccessThreshold)

	constants.CircuitBreakerRequestTimeout = getEnvDuration("NEPHORAN_CB_REQUEST_TIMEOUT", constants.CircuitBreakerRequestTimeout)

	constants.CircuitBreakerHalfOpenMaxRequests = getEnvInt("NEPHORAN_CB_HALF_OPEN_MAX", constants.CircuitBreakerHalfOpenMaxRequests)

	constants.CircuitBreakerMinimumRequests = getEnvInt("NEPHORAN_CB_MIN_REQUESTS", constants.CircuitBreakerMinimumRequests)

	constants.CircuitBreakerFailureRate = getEnvFloat("NEPHORAN_CB_FAILURE_RATE", constants.CircuitBreakerFailureRate)



	// Load timeout configuration.

	constants.LLMTimeout = getEnvDuration("NEPHORAN_LLM_TIMEOUT", constants.LLMTimeout)

	constants.GitTimeout = getEnvDuration("NEPHORAN_GIT_TIMEOUT", constants.GitTimeout)

	constants.KubernetesTimeout = getEnvDuration("NEPHORAN_KUBERNETES_TIMEOUT", constants.KubernetesTimeout)

	constants.PackageGenerationTimeout = getEnvDuration("NEPHORAN_PACKAGE_GENERATION_TIMEOUT", constants.PackageGenerationTimeout)

	constants.RAGTimeout = getEnvDuration("NEPHORAN_RAG_TIMEOUT", constants.RAGTimeout)

	constants.ReconciliationTimeout = getEnvDuration("NEPHORAN_RECONCILIATION_TIMEOUT", constants.ReconciliationTimeout)

	constants.DefaultTimeout = getEnvDuration("NEPHORAN_DEFAULT_TIMEOUT", constants.DefaultTimeout)



	// Load resource limits.

	constants.CPURequestDefault = getEnvString("NEPHORAN_CPU_REQUEST_DEFAULT", constants.CPURequestDefault)

	constants.MemoryRequestDefault = getEnvString("NEPHORAN_MEMORY_REQUEST_DEFAULT", constants.MemoryRequestDefault)

	constants.CPULimitDefault = getEnvString("NEPHORAN_CPU_LIMIT_DEFAULT", constants.CPULimitDefault)

	constants.MemoryLimitDefault = getEnvString("NEPHORAN_MEMORY_LIMIT_DEFAULT", constants.MemoryLimitDefault)



	// Load monitoring configuration.

	constants.MetricsPort = getEnvInt("NEPHORAN_METRICS_PORT", constants.MetricsPort)

	constants.HealthProbePort = getEnvInt("NEPHORAN_HEALTH_PROBE_PORT", constants.HealthProbePort)

	constants.MetricsUpdateInterval = getEnvDuration("NEPHORAN_METRICS_UPDATE_INTERVAL", constants.MetricsUpdateInterval)

	constants.HealthCheckTimeout = getEnvDuration("NEPHORAN_HEALTH_CHECK_TIMEOUT", constants.HealthCheckTimeout)



	return &constants

}



// Helper functions for environment variable parsing.



// getEnvString gets a string environment variable with a default value.

func getEnvString(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {

		return value

	}

	return defaultValue

}



// getEnvInt gets an integer environment variable with a default value.

func getEnvInt(key string, defaultValue int) int {

	if value := os.Getenv(key); value != "" {

		if parsed, err := strconv.Atoi(value); err == nil {

			return parsed

		}

	}

	return defaultValue

}



// getEnvFloat gets a float environment variable with a default value.

func getEnvFloat(key string, defaultValue float64) float64 {

	if value := os.Getenv(key); value != "" {

		if parsed, err := strconv.ParseFloat(value, 64); err == nil {

			return parsed

		}

	}

	return defaultValue

}



// getEnvDuration gets a duration environment variable with a default value.

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {

	if value := os.Getenv(key); value != "" {

		if parsed, err := time.ParseDuration(value); err == nil {

			return parsed

		}

	}

	return defaultValue

}



// getEnvStringArray gets a comma-separated string array environment variable with a default value.

func getEnvStringArray(key string, defaultValue []string) []string {

	if value := os.Getenv(key); value != "" {

		// Simple split by comma - could be enhanced for more complex parsing.

		result := make([]string, 0)

		for _, item := range splitString(value, ",") {

			if trimmed := trimSpace(item); trimmed != "" {

				result = append(result, trimmed)

			}

		}

		if len(result) > 0 {

			return result

		}

	}

	return defaultValue

}



// Simple string split function (avoiding external dependencies).

func splitString(s, sep string) []string {

	if s == "" {

		return nil

	}



	var result []string

	start := 0

	for i := 0; i <= len(s)-len(sep); i++ {

		if s[i:i+len(sep)] == sep {

			result = append(result, s[start:i])

			start = i + len(sep)

			i += len(sep) - 1 // Skip the separator

		}

	}

	result = append(result, s[start:]) // Add the last part

	return result

}



// Simple string trim function (avoiding external dependencies).

func trimSpace(s string) string {

	start := 0

	end := len(s)



	// Trim leading spaces.

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {

		start++

	}



	// Trim trailing spaces.

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {

		end--

	}



	return s[start:end]

}



// ValidateConstants validates that all configuration values are within acceptable ranges.

func ValidateConstants(constants *Constants) error {

	// Validate retry configuration.

	if constants.MaxRetries < 0 {

		return fmt.Errorf("MaxRetries cannot be negative: %d", constants.MaxRetries)

	}

	if constants.MaxRetries > constants.MaxAllowedRetries {

		return fmt.Errorf("MaxRetries (%d) exceeds maximum allowed (%d)", constants.MaxRetries, constants.MaxAllowedRetries)

	}

	if constants.RetryDelay <= 0 {

		return fmt.Errorf("RetryDelay must be positive: %v", constants.RetryDelay)

	}

	if constants.RetryDelay > constants.MaxAllowedRetryDelay {

		return fmt.Errorf("RetryDelay (%v) exceeds maximum allowed (%v)", constants.RetryDelay, constants.MaxAllowedRetryDelay)

	}



	// Validate timeout configuration.

	if constants.Timeout <= 0 {

		return fmt.Errorf("Timeout must be positive: %v", constants.Timeout)

	}

	if constants.LLMTimeout <= 0 {

		return fmt.Errorf("LLMTimeout must be positive: %v", constants.LLMTimeout)

	}

	if constants.GitTimeout <= 0 {

		return fmt.Errorf("GitTimeout must be positive: %v", constants.GitTimeout)

	}

	if constants.KubernetesTimeout <= 0 {

		return fmt.Errorf("KubernetesTimeout must be positive: %v", constants.KubernetesTimeout)

	}



	// Validate backoff configuration.

	if constants.BaseBackoffDelay <= 0 {

		return fmt.Errorf("BaseBackoffDelay must be positive: %v", constants.BaseBackoffDelay)

	}

	if constants.MaxBackoffDelay <= constants.BaseBackoffDelay {

		return fmt.Errorf("MaxBackoffDelay (%v) must be greater than BaseBackoffDelay (%v)", constants.MaxBackoffDelay, constants.BaseBackoffDelay)

	}

	if constants.JitterFactor < 0 || constants.JitterFactor > 1 {

		return fmt.Errorf("JitterFactor must be between 0 and 1: %f", constants.JitterFactor)

	}

	if constants.BackoffMultiplier <= 1 {

		return fmt.Errorf("BackoffMultiplier must be greater than 1: %f", constants.BackoffMultiplier)

	}



	// Validate security configuration.

	if constants.MaxInputLength <= 0 {

		return fmt.Errorf("MaxInputLength must be positive: %d", constants.MaxInputLength)

	}

	if constants.MaxInputLength > 1000000 { // 1MB limit

		return fmt.Errorf("MaxInputLength exceeds safety limit of 1MB: %d", constants.MaxInputLength)

	}

	if constants.MaxOutputLength <= 0 {

		return fmt.Errorf("MaxOutputLength must be positive: %d", constants.MaxOutputLength)

	}

	if constants.MaxOutputLength > 10000000 { // 10MB limit

		return fmt.Errorf("MaxOutputLength exceeds safety limit of 10MB: %d", constants.MaxOutputLength)

	}

	if constants.ContextBoundary == "" {

		return fmt.Errorf("ContextBoundary cannot be empty")

	}



	// Validate circuit breaker configuration.

	if constants.CircuitBreakerFailureThreshold <= 0 {

		return fmt.Errorf("CircuitBreakerFailureThreshold must be positive: %d", constants.CircuitBreakerFailureThreshold)

	}

	if constants.CircuitBreakerRecoveryTimeout <= 0 {

		return fmt.Errorf("CircuitBreakerRecoveryTimeout must be positive: %v", constants.CircuitBreakerRecoveryTimeout)

	}

	if constants.CircuitBreakerSuccessThreshold <= 0 {

		return fmt.Errorf("CircuitBreakerSuccessThreshold must be positive: %d", constants.CircuitBreakerSuccessThreshold)

	}

	if constants.CircuitBreakerFailureRate <= 0 || constants.CircuitBreakerFailureRate > 1 {

		return fmt.Errorf("CircuitBreakerFailureRate must be between 0 and 1: %f", constants.CircuitBreakerFailureRate)

	}



	// Validate monitoring configuration.

	if constants.MetricsPort <= 0 || constants.MetricsPort > 65535 {

		return fmt.Errorf("MetricsPort must be a valid port number: %d", constants.MetricsPort)

	}

	if constants.HealthProbePort <= 0 || constants.HealthProbePort > 65535 {

		return fmt.Errorf("HealthProbePort must be a valid port number: %d", constants.HealthProbePort)

	}

	if constants.MetricsUpdateInterval <= 0 {

		return fmt.Errorf("MetricsUpdateInterval must be positive: %v", constants.MetricsUpdateInterval)

	}

	if constants.HealthCheckTimeout <= 0 {

		return fmt.Errorf("HealthCheckTimeout must be positive: %v", constants.HealthCheckTimeout)

	}



	return nil

}



// GetDefaults returns the default constants configuration.

func GetDefaults() *Constants {

	defaults := defaultConstants

	return &defaults

}



// PrintConfiguration prints the current configuration (for debugging).

func PrintConfiguration(constants *Constants) {

	fmt.Printf("Nephoran Intent Operator Configuration:\n")

	fmt.Printf("  Controller:\n")

	fmt.Printf("    MaxRetries: %d\n", constants.MaxRetries)

	fmt.Printf("    RetryDelay: %v\n", constants.RetryDelay)

	fmt.Printf("    Timeout: %v\n", constants.Timeout)

	fmt.Printf("  Security:\n")

	fmt.Printf("    MaxInputLength: %d\n", constants.MaxInputLength)

	fmt.Printf("    MaxOutputLength: %d\n", constants.MaxOutputLength)

	fmt.Printf("    AllowedDomains: %d configured\n", len(constants.AllowedDomains))

	fmt.Printf("    BlockedKeywords: %d configured\n", len(constants.BlockedKeywords))

	fmt.Printf("  Circuit Breaker:\n")

	fmt.Printf("    FailureThreshold: %d\n", constants.CircuitBreakerFailureThreshold)

	fmt.Printf("    RecoveryTimeout: %v\n", constants.CircuitBreakerRecoveryTimeout)

	fmt.Printf("  Timeouts:\n")

	fmt.Printf("    LLM: %v\n", constants.LLMTimeout)

	fmt.Printf("    Git: %v\n", constants.GitTimeout)

	fmt.Printf("    Kubernetes: %v\n", constants.KubernetesTimeout)

}


package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/interfaces"
)

// LLMProcessorConfig holds configuration specific to the LLM processor service.

type LLMProcessorConfig struct {

	// Service Configuration.

	Enabled bool // Whether the LLM processor service is enabled

	Port string

	LogLevel string

	ServiceVersion string

	GracefulShutdown time.Duration

	// LLM Configuration.

	LLMBackendType string

	LLMAPIKey string

	LLMModelName string

	LLMTimeout time.Duration // Uses LLM_TIMEOUT_SECS env var (default: 15s)

	LLMMaxTokens int

	LLMMaxRetries int // Uses LLM_MAX_RETRIES env var (default: 2)

	// RAG Configuration.

	RAGAPIURL string

	RAGTimeout time.Duration

	RAGEnabled bool

	RAGPreferProcessEndpoint bool // When true, prefer /process over /process_intent for new configs

	RAGEndpointPath string // Custom endpoint path override (optional)

	// Streaming Configuration.

	StreamingEnabled bool

	MaxConcurrentStreams int

	StreamTimeout time.Duration

	// Context Management.

	EnableContextBuilder bool

	MaxContextTokens int

	ContextTTL time.Duration

	// Cache Configuration.

	CacheMaxEntries int // Uses LLM_CACHE_MAX_ENTRIES env var (default: 512)

	// Security Configuration.

	APIKeyRequired bool

	APIKey string

	CORSEnabled bool

	AllowedOrigins []string

	// Performance Configuration.

	RequestTimeout time.Duration

	MaxRequestSize int64

	// Circuit Breaker Configuration.

	CircuitBreakerEnabled bool

	CircuitBreakerThreshold int

	CircuitBreakerTimeout time.Duration

	// Rate Limiting.

	RateLimitEnabled bool

	RateLimitRequestsPerMin int

	RateLimitBurst int

	RateLimitQPS int // Queries per second for IP-based rate limiting

	RateLimitBurstTokens int // Burst tokens for IP-based rate limiting

	// Retry Configuration.

	MaxRetries int

	RetryDelay time.Duration

	RetryBackoff string

	// OAuth2 Authentication Configuration.

	AuthEnabled bool

	AuthConfigFile string

	JWTSecretKey string

	RequireAuth bool

	AdminUsers []string

	OperatorUsers []string

	// TLS Configuration.

	TLSEnabled bool

	TLSCertPath string

	TLSKeyPath string

	// Secret Management Configuration.

	UseKubernetesSecrets bool

	SecretNamespace string

	// Metrics Security Configuration.

	ExposeMetricsPublicly bool // Whether to expose metrics publicly (default: false)

	MetricsAllowedCIDRs []string // CIDR blocks allowed to access metrics endpoint

	MetricsEnabled bool // Whether to enable the /metrics endpoint at all

	MetricsAllowedIPs []string // Specific IP addresses allowed to access metrics

}

// DefaultLLMProcessorConfig returns a configuration with sensible defaults.

func DefaultLLMProcessorConfig() *LLMProcessorConfig {

	return &LLMProcessorConfig{

		Enabled: true, // Default to enabled

		Port: "8080",

		LogLevel: "info",

		ServiceVersion: "v2.0.0",

		GracefulShutdown: 30 * time.Second,

		LLMBackendType: "rag",

		LLMModelName: "gpt-4o-mini",

		LLMTimeout: 15 * time.Second, // Default 15s for individual LLM requests

		LLMMaxTokens: 2048,

		LLMMaxRetries: 2, // Default 2 retries

		RAGAPIURL: "http://rag-api:5001",

		RAGTimeout: 30 * time.Second,

		RAGEnabled: true,

		RAGPreferProcessEndpoint: true, // Default to /process for new installations

		RAGEndpointPath: "", // Auto-detect from URL pattern

		StreamingEnabled: true,

		MaxConcurrentStreams: 100,

		StreamTimeout: 5 * time.Minute,

		EnableContextBuilder: true,

		MaxContextTokens: 6000,

		ContextTTL: 5 * time.Minute,

		// Cache configuration.

		CacheMaxEntries: 512, // Default 512 cache entries

		APIKeyRequired: false,

		CORSEnabled: true,

		AllowedOrigins: []string{}, // Security: Default to empty, must be explicitly configured via LLM_ALLOWED_ORIGINS

		RequestTimeout: 30 * time.Second,

		MaxRequestSize: 1048576, // 1MB

		CircuitBreakerEnabled: true,

		CircuitBreakerThreshold: 5,

		CircuitBreakerTimeout: 60 * time.Second,

		RateLimitEnabled: true,

		RateLimitRequestsPerMin: 60,

		RateLimitBurst: 10,

		RateLimitQPS: 20, // Default 20 queries per second

		RateLimitBurstTokens: 40, // Default 40 burst tokens

		MaxRetries: 3,

		RetryDelay: 1 * time.Second,

		RetryBackoff: "exponential",

		AuthEnabled: true,

		RequireAuth: true,

		AdminUsers: []string{},

		OperatorUsers: []string{},

		TLSEnabled: false,

		TLSCertPath: "",

		TLSKeyPath: "",

		UseKubernetesSecrets: true,

		SecretNamespace: "nephoran-system",

		// Metrics Security - Default to private with secure defaults.

		ExposeMetricsPublicly: false,

		MetricsAllowedCIDRs: []string{}, // Will be set to private networks if empty and not public

		MetricsEnabled: false, // Disabled by default for security

		MetricsAllowedIPs: []string{}, // Empty means no IP restriction if metrics enabled

	}

}

// LoadLLMProcessorConfig loads configuration from environment variables with validation.

func LoadLLMProcessorConfig() (*LLMProcessorConfig, error) {

	cfg := DefaultLLMProcessorConfig()

	// Load configuration with validation.

	var validationErrors []string

	// Service Configuration.

	cfg.Port = getEnvWithValidation("PORT", cfg.Port, validatePort, &validationErrors)

	cfg.LogLevel = getEnvWithValidation("LOG_LEVEL", cfg.LogLevel, validateLogLevel, &validationErrors)

	cfg.ServiceVersion = GetEnvOrDefault("SERVICE_VERSION", cfg.ServiceVersion)

	cfg.GracefulShutdown = parseDurationWithValidation("GRACEFUL_SHUTDOWN_TIMEOUT", cfg.GracefulShutdown, &validationErrors)

	// LLM Configuration.

	cfg.LLMBackendType = getEnvWithValidation("LLM_BACKEND_TYPE", cfg.LLMBackendType, validateLLMBackendType, &validationErrors)

	// Load LLM API key from file or env based on backend type.

	var auditLogger interfaces.AuditLogger // nil interface is valid

	llmAPIKey, err := LoadLLMAPIKeyFromFile(cfg.LLMBackendType, auditLogger)

	if err != nil && cfg.LLMBackendType != "mock" && cfg.LLMBackendType != "rag" {

		validationErrors = append(validationErrors, fmt.Sprintf("LLM API Key: %v", err))

	}

	cfg.LLMAPIKey = llmAPIKey

	cfg.LLMModelName = GetEnvOrDefault("LLM_MODEL_NAME", cfg.LLMModelName)

	// Handle LLM_TIMEOUT_SECS as seconds value.

	if envTimeoutSecs := GetEnvOrDefault("LLM_TIMEOUT_SECS", ""); envTimeoutSecs != "" {

		if timeoutSecs, err := strconv.Atoi(envTimeoutSecs); err == nil && timeoutSecs > 0 {

			cfg.LLMTimeout = time.Duration(timeoutSecs) * time.Second

		} else if err != nil {

			validationErrors = append(validationErrors, fmt.Sprintf("LLM_TIMEOUT_SECS: invalid integer format: %v", err))

		} else {

			validationErrors = append(validationErrors, "LLM_TIMEOUT_SECS: must be positive")

		}

	}

	cfg.LLMMaxTokens = parseIntWithValidation("LLM_MAX_TOKENS", cfg.LLMMaxTokens, validatePositiveInt, &validationErrors)

	cfg.LLMMaxRetries = parseIntWithValidation("LLM_MAX_RETRIES", cfg.LLMMaxRetries, validateNonNegativeInt, &validationErrors)

	// RAG Configuration.

	cfg.RAGAPIURL = getEnvWithValidation("RAG_API_URL", cfg.RAGAPIURL, validateURL, &validationErrors)

	cfg.RAGTimeout = parseDurationWithValidation("RAG_TIMEOUT", cfg.RAGTimeout, &validationErrors)

	cfg.RAGEnabled = parseBoolWithDefault("RAG_ENABLED", cfg.RAGEnabled)

	cfg.RAGPreferProcessEndpoint = parseBoolWithDefault("RAG_PREFER_PROCESS_ENDPOINT", cfg.RAGPreferProcessEndpoint)

	cfg.RAGEndpointPath = GetEnvOrDefault("RAG_ENDPOINT_PATH", cfg.RAGEndpointPath)

	// Streaming Configuration.

	cfg.StreamingEnabled = parseBoolWithDefault("STREAMING_ENABLED", cfg.StreamingEnabled)

	cfg.MaxConcurrentStreams = parseIntWithValidation("MAX_CONCURRENT_STREAMS", cfg.MaxConcurrentStreams, validatePositiveInt, &validationErrors)

	cfg.StreamTimeout = parseDurationWithValidation("STREAM_TIMEOUT", cfg.StreamTimeout, &validationErrors)

	// Context Management.

	cfg.EnableContextBuilder = parseBoolWithDefault("ENABLE_CONTEXT_BUILDER", cfg.EnableContextBuilder)

	cfg.MaxContextTokens = parseIntWithValidation("MAX_CONTEXT_TOKENS", cfg.MaxContextTokens, validatePositiveInt, &validationErrors)

	cfg.ContextTTL = parseDurationWithValidation("CONTEXT_TTL", cfg.ContextTTL, &validationErrors)

	// Cache Configuration.

	cfg.CacheMaxEntries = parseIntWithValidation("LLM_CACHE_MAX_ENTRIES", cfg.CacheMaxEntries, validatePositiveInt, &validationErrors)

	// Security Configuration.

	cfg.APIKeyRequired = parseBoolWithDefault("API_KEY_REQUIRED", cfg.APIKeyRequired)

	// Load API key from file or env.

	apiKey, err := LoadAPIKeyFromFile(auditLogger)

	if err != nil && cfg.APIKeyRequired {

		validationErrors = append(validationErrors, fmt.Sprintf("API Key: %v", err))

	}

	cfg.APIKey = apiKey

	cfg.CORSEnabled = parseBoolWithDefault("CORS_ENABLED", cfg.CORSEnabled)

	// Parse and validate allowed origins.

	allowedOriginsStr := GetEnvOrDefault("LLM_ALLOWED_ORIGINS", "")

	if cfg.CORSEnabled && allowedOriginsStr != "" {

		parsedOrigins, err := parseAllowedOrigins(allowedOriginsStr)

		if err != nil {

			validationErrors = append(validationErrors, fmt.Sprintf("LLM_ALLOWED_ORIGINS: %v", err))

		} else {

			cfg.AllowedOrigins = parsedOrigins

		}

	} else if cfg.CORSEnabled {

		// Set secure development default when CORS is enabled but no origins specified.

		if isDevelopmentEnvironment() {

			cfg.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}

		}

	}

	// Performance Configuration.

	cfg.RequestTimeout = parseDurationWithValidation("REQUEST_TIMEOUT", cfg.RequestTimeout, &validationErrors)

	// Use HTTP_MAX_BODY env var for request size limit (fallback to MAX_REQUEST_SIZE for backward compatibility).

	httpMaxBody := GetEnvOrDefault("HTTP_MAX_BODY", "")

	if httpMaxBody != "" {

		cfg.MaxRequestSize = parseInt64WithValidation("HTTP_MAX_BODY", cfg.MaxRequestSize, validatePositiveInt64, &validationErrors)

	} else {

		cfg.MaxRequestSize = parseInt64WithValidation("MAX_REQUEST_SIZE", cfg.MaxRequestSize, validatePositiveInt64, &validationErrors)

	}

	// Circuit Breaker Configuration.

	cfg.CircuitBreakerEnabled = parseBoolWithDefault("CIRCUIT_BREAKER_ENABLED", cfg.CircuitBreakerEnabled)

	cfg.CircuitBreakerThreshold = parseIntWithValidation("CIRCUIT_BREAKER_THRESHOLD", cfg.CircuitBreakerThreshold, validatePositiveInt, &validationErrors)

	cfg.CircuitBreakerTimeout = parseDurationWithValidation("CIRCUIT_BREAKER_TIMEOUT", cfg.CircuitBreakerTimeout, &validationErrors)

	// Rate Limiting.

	cfg.RateLimitEnabled = parseBoolWithDefault("RATE_LIMIT_ENABLED", cfg.RateLimitEnabled)

	cfg.RateLimitRequestsPerMin = parseIntWithValidation("RATE_LIMIT_REQUESTS_PER_MINUTE", cfg.RateLimitRequestsPerMin, validatePositiveInt, &validationErrors)

	cfg.RateLimitBurst = parseIntWithValidation("RATE_LIMIT_BURST", cfg.RateLimitBurst, validatePositiveInt, &validationErrors)

	cfg.RateLimitQPS = parseIntWithValidation("RATE_LIMIT_QPS", cfg.RateLimitQPS, validatePositiveInt, &validationErrors)

	cfg.RateLimitBurstTokens = parseIntWithValidation("RATE_LIMIT_BURST_TOKENS", cfg.RateLimitBurstTokens, validatePositiveInt, &validationErrors)

	// Retry Configuration.

	cfg.MaxRetries = parseIntWithValidation("MAX_RETRIES", cfg.MaxRetries, validateNonNegativeInt, &validationErrors)

	cfg.RetryDelay = parseDurationWithValidation("RETRY_DELAY", cfg.RetryDelay, &validationErrors)

	cfg.RetryBackoff = getEnvWithValidation("RETRY_BACKOFF", cfg.RetryBackoff, validateRetryBackoff, &validationErrors)

	// OAuth2 Authentication Configuration.

	cfg.AuthEnabled = parseBoolWithDefault("AUTH_ENABLED", cfg.AuthEnabled)

	cfg.AuthConfigFile = GetEnvOrDefault("AUTH_CONFIG_FILE", "")

	// Load JWT secret key from file or env.

	jwtSecretKey, err := LoadJWTSecretKeyFromFile(auditLogger)

	if err != nil && cfg.AuthEnabled {

		validationErrors = append(validationErrors, fmt.Sprintf("JWT Secret Key: %v", err))

	}

	cfg.JWTSecretKey = jwtSecretKey

	cfg.RequireAuth = parseBoolWithDefault("REQUIRE_AUTH", cfg.RequireAuth)

	cfg.AdminUsers = parseStringSlice(GetEnvOrDefault("ADMIN_USERS", ""))

	cfg.OperatorUsers = parseStringSlice(GetEnvOrDefault("OPERATOR_USERS", ""))

	// TLS Configuration.

	cfg.TLSEnabled = parseBoolWithDefault("TLS_ENABLED", cfg.TLSEnabled)

	cfg.TLSCertPath = GetEnvOrDefault("TLS_CERT_PATH", cfg.TLSCertPath)

	cfg.TLSKeyPath = GetEnvOrDefault("TLS_KEY_PATH", cfg.TLSKeyPath)

	// Secret Management Configuration.

	cfg.UseKubernetesSecrets = parseBoolWithDefault("USE_KUBERNETES_SECRETS", cfg.UseKubernetesSecrets)

	cfg.SecretNamespace = GetEnvOrDefault("SECRET_NAMESPACE", cfg.SecretNamespace)

	// Metrics Security Configuration.

	cfg.MetricsEnabled = parseBoolWithDefault("METRICS_ENABLED", cfg.MetricsEnabled)

	cfg.ExposeMetricsPublicly = parseBoolWithDefault("EXPOSE_METRICS_PUBLICLY", cfg.ExposeMetricsPublicly)

	// Parse IP-based access control list.

	metricsAllowedIPs := GetEnvOrDefault("METRICS_ALLOWED_IPS", "")

	if metricsAllowedIPs != "" {

		parsedIPs, err := parseAndValidateIPs(metricsAllowedIPs)

		if err != nil {

			validationErrors = append(validationErrors, fmt.Sprintf("METRICS_ALLOWED_IPS: %v", err))

		} else {

			cfg.MetricsAllowedIPs = parsedIPs

		}

	}

	// Parse CIDR allowlist.

	metricsAllowedCIDRs := GetEnvOrDefault("METRICS_ALLOWED_CIDRS", "")

	if metricsAllowedCIDRs != "" {

		parsedCIDRs, err := parseAndValidateCIDRs(metricsAllowedCIDRs)

		if err != nil {

			validationErrors = append(validationErrors, fmt.Sprintf("METRICS_ALLOWED_CIDRS: %v", err))

		} else {

			cfg.MetricsAllowedCIDRs = parsedCIDRs

		}

	} else if !cfg.ExposeMetricsPublicly && cfg.MetricsEnabled {

		// Set secure defaults for private access only if metrics are enabled.

		cfg.MetricsAllowedCIDRs = getDefaultPrivateNetworks()

	}

	// Return validation errors if any.

	if len(validationErrors) > 0 {

		return nil, fmt.Errorf("configuration validation failed: %s", strings.Join(validationErrors, "; "))

	}

	// Perform additional cross-field validation.

	if err := cfg.Validate(); err != nil {

		return nil, fmt.Errorf("configuration validation failed: %w", err)

	}

	return cfg, nil

}

// Validate performs comprehensive validation of the configuration.

func (c *LLMProcessorConfig) Validate() error {

	var errors []string

	// Validate required fields based on configuration.

	if c.LLMAPIKey == "" && c.LLMBackendType != "mock" && c.LLMBackendType != "rag" {

		errors = append(errors, "OPENAI_API_KEY is required for non-mock/non-rag backends")

	}

	if c.AuthEnabled && c.JWTSecretKey == "" {

		errors = append(errors, "JWT_SECRET_KEY is required when authentication is enabled")

	}

	if c.APIKeyRequired && c.APIKey == "" {

		errors = append(errors, "API_KEY is required when API key authentication is enabled")

	}

	// Validate TLS configuration.

	if c.TLSEnabled {

		if c.TLSCertPath == "" {

			errors = append(errors, "TLS_CERT_PATH is required when TLS is enabled")

		}

		if c.TLSKeyPath == "" {

			errors = append(errors, "TLS_KEY_PATH is required when TLS is enabled")

		}

		// Validate that both cert and key files exist if paths are provided.

		if c.TLSCertPath != "" && c.TLSKeyPath != "" {

			if err := validateTLSFiles(c.TLSCertPath, c.TLSKeyPath); err != nil {

				errors = append(errors, fmt.Sprintf("TLS configuration: %v", err))

			}

		}

	} else if c.TLSCertPath != "" || c.TLSKeyPath != "" {
		// If TLS is disabled but paths are provided, warn about potential misconfiguration.
		errors = append(errors, "TLS certificate/key paths provided but TLS_ENABLED=false. Set TLS_ENABLED=true to enable TLS")
	}

	// Validate logical constraints.

	if c.MaxConcurrentStreams > 1000 {

		errors = append(errors, "MAX_CONCURRENT_STREAMS should not exceed 1000 for performance reasons")

	}

	if c.MaxContextTokens > 32000 {

		errors = append(errors, "MAX_CONTEXT_TOKENS should not exceed 32000 for most models")

	}

	if c.CircuitBreakerThreshold > 50 {

		errors = append(errors, "CIRCUIT_BREAKER_THRESHOLD should be reasonable (≤50)")

	}

	// Validate CORS configuration.

	if c.CORSEnabled {

		if len(c.AllowedOrigins) == 0 {

			errors = append(errors, "LLM_ALLOWED_ORIGINS must be configured when CORS is enabled")

		} else {

			// Validate individual origins.

			isProduction := GetEnvOrDefault("LLM_ENVIRONMENT", "") == "production"

			for _, origin := range c.AllowedOrigins {

				if origin == "" {

					continue

				}

				// Check for wildcard in production.

				if origin == "*" && isProduction {

					errors = append(errors, "wildcard origin '*' is not allowed in production environments")

				}

				// Validate origin format.

				if origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {

					errors = append(errors, fmt.Sprintf("origin '%s' must start with http:// or https://", origin))

				}

				// Additional security checks.

				if strings.Contains(origin, "*") && origin != "*" {

					errors = append(errors, fmt.Sprintf("origin '%s' contains wildcard but is not exact wildcard '*'", origin))

				}

			}

		}

	}

	// Validate request size limits for security and performance.

	if c.MaxRequestSize <= 0 {

		errors = append(errors, "MAX_REQUEST_SIZE must be positive")

	}

	if c.MaxRequestSize < 1024 {

		errors = append(errors, "MAX_REQUEST_SIZE should be at least 1KB for basic functionality")

	}

	if c.MaxRequestSize > 100*1024*1024 { // 100MB

		errors = append(errors, "MAX_REQUEST_SIZE should not exceed 100MB to prevent DoS attacks")

	}

	// Validate RAG configuration consistency.

	if err := c.validateRAGConfiguration(); err != nil {

		errors = append(errors, fmt.Sprintf("RAG configuration: %v", err))

	}

	// Validate metrics security configuration.

	if err := c.validateMetricsSecurityConfiguration(); err != nil {

		errors = append(errors, fmt.Sprintf("Metrics security configuration: %v", err))

	}

	if len(errors) > 0 {

		return fmt.Errorf("%s", strings.Join(errors, "; "))

	}

	return nil

}

// validateRAGConfiguration performs RAG-specific configuration validation.

func (c *LLMProcessorConfig) validateRAGConfiguration() error {

	if !c.RAGEnabled {

		return nil // Skip validation if RAG is disabled

	}

	// Validate RAG API URL format and pattern.

	if c.RAGAPIURL == "" {

		return fmt.Errorf("RAG_API_URL is required when RAG is enabled")

	}

	// Custom endpoint path validation.

	if c.RAGEndpointPath != "" {

		if !strings.HasPrefix(c.RAGEndpointPath, "/") {

			return fmt.Errorf("RAG_ENDPOINT_PATH must start with '/' if specified")

		}

		// Warn about conflicting settings.

		if c.RAGPreferProcessEndpoint {

			// This is not an error, but the custom path takes precedence.

			// Note: Custom path overrides the RAGPreferProcessEndpoint setting.
			// Intentionally empty - no action needed as custom path handling is done elsewhere

		}

	}

	return nil

}

// Helper functions for validation and parsing.

// Note: getEnvOrDefault has been replaced with GetEnvOrDefault from this package.

func getEnvWithValidation(key, defaultValue string, validator func(string) error, errors *[]string) string {

	value := GetEnvOrDefault(key, defaultValue)

	if err := validator(value); err != nil {

		*errors = append(*errors, fmt.Sprintf("%s: %v", key, err))

	}

	return value

}

func parseDurationWithValidation(key string, defaultValue time.Duration, errors *[]string) time.Duration {

	if valueStr := GetEnvOrDefault(key, ""); valueStr != "" {

		if d, err := time.ParseDuration(valueStr); err == nil {

			if d <= 0 {

				*errors = append(*errors, fmt.Sprintf("%s: duration must be positive", key))

				return defaultValue

			}

			return d

		} else {

			*errors = append(*errors, fmt.Sprintf("%s: invalid duration format: %v", key, err))

		}

	}

	return defaultValue

}

func parseIntWithValidation(key string, defaultValue int, validator func(int) error, errors *[]string) int {

	if valueStr := GetEnvOrDefault(key, ""); valueStr != "" {

		if value, err := strconv.Atoi(valueStr); err == nil {

			if err := validator(value); err != nil {

				*errors = append(*errors, fmt.Sprintf("%s: %v", key, err))

				return defaultValue

			}

			return value

		} else {

			*errors = append(*errors, fmt.Sprintf("%s: invalid integer format: %v", key, err))

		}

	}

	return defaultValue

}

func parseInt64WithValidation(key string, defaultValue int64, validator func(int64) error, errors *[]string) int64 {

	if valueStr := GetEnvOrDefault(key, ""); valueStr != "" {

		if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {

			if err := validator(value); err != nil {

				*errors = append(*errors, fmt.Sprintf("%s: %v", key, err))

				return defaultValue

			}

			return value

		} else {

			*errors = append(*errors, fmt.Sprintf("%s: invalid integer format: %v", key, err))

		}

	}

	return defaultValue

}

func parseBoolWithDefault(key string, defaultValue bool) bool {

	if valueStr := GetEnvOrDefault(key, ""); valueStr != "" {

		return valueStr == "true" || valueStr == "1"

	}

	return defaultValue

}

func parseStringSlice(s string) []string {

	if s == "" {

		return []string{}

	}

	result := []string{}

	for _, item := range strings.Split(s, ",") {

		if trimmed := strings.TrimSpace(item); trimmed != "" {

			result = append(result, trimmed)

		}

	}

	return result

}

// Validation functions.

func validatePort(port string) error {

	if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {

		return fmt.Errorf("invalid port number, must be between 1 and 65535")

	}

	return nil

}

func validateLogLevel(level string) error {

	validLevels := []string{"debug", "info", "warn", "error"}

	for _, valid := range validLevels {

		if strings.EqualFold(level, valid) {

			return nil

		}

	}

	return fmt.Errorf("invalid log level, must be one of: %s", strings.Join(validLevels, ", "))

}

func validateLLMBackendType(backendType string) error {

	validTypes := []string{"openai", "mistral", "rag", "mock"}

	for _, valid := range validTypes {

		if strings.EqualFold(backendType, valid) {

			return nil

		}

	}

	return fmt.Errorf("invalid LLM backend type, must be one of: %s", strings.Join(validTypes, ", "))

}

func validateURL(url string) error {

	if url == "" {

		return fmt.Errorf("URL cannot be empty")

	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {

		return fmt.Errorf("URL must start with http:// or https://")

	}

	// Additional URL pattern validation for RAG API URLs.

	if strings.Contains(url, "rag-api") || strings.Contains(url, "RAG_API") {

		return validateRAGURL(url)

	}

	return nil

}

// validateRAGURL performs RAG-specific URL validation.

func validateRAGURL(ragURL string) error {

	// Parse URL to validate structure.

	if _, err := url.Parse(ragURL); err != nil {

		return fmt.Errorf("invalid URL format: %w", err)

	}

	// Check for common patterns and provide helpful warnings.
	switch {
	case strings.HasSuffix(ragURL, "/process_intent"):
		// This is valid but legacy - no error needed.
	case strings.HasSuffix(ragURL, "/process"):
		// This is the preferred pattern - no error needed.
	case strings.HasSuffix(ragURL, "/health"):
		return fmt.Errorf("RAG_API_URL should not end with '/health' - this is auto-detected for health checks")
	case strings.HasSuffix(ragURL, "/stream"):
		return fmt.Errorf("RAG_API_URL should not end with '/stream' - this is auto-detected for streaming")
	}

	return nil

}

func validatePositiveInt(value int) error {

	if value <= 0 {

		return fmt.Errorf("value must be positive")

	}

	return nil

}

func validateNonNegativeInt(value int) error {

	if value < 0 {

		return fmt.Errorf("value must be non-negative")

	}

	return nil

}

func validatePositiveInt64(value int64) error {

	if value <= 0 {

		return fmt.Errorf("value must be positive")

	}

	return nil

}

func validateRetryBackoff(backoff string) error {

	validBackoffs := []string{"fixed", "linear", "exponential"}

	for _, valid := range validBackoffs {

		if strings.EqualFold(backoff, valid) {

			return nil

		}

	}

	return fmt.Errorf("invalid retry backoff, must be one of: %s", strings.Join(validBackoffs, ", "))

}

// parseAllowedOrigins parses and validates a comma-separated list of origins.

func parseAllowedOrigins(originsStr string) ([]string, error) {

	if strings.TrimSpace(originsStr) == "" {

		return nil, fmt.Errorf("origins string cannot be empty")

	}

	origins := strings.Split(originsStr, ",")

	// Pre-allocate slice with known capacity for better performance
	validOrigins := make([]string, 0, len(origins))

	isProduction := GetEnvOrDefault("LLM_ENVIRONMENT", "") == "production"

	for _, origin := range origins {

		origin = strings.TrimSpace(origin)

		if origin == "" {

			continue

		}

		// Validate origin.

		if err := validateOrigin(origin, isProduction); err != nil {

			return nil, fmt.Errorf("invalid origin '%s': %w", origin, err)

		}

		validOrigins = append(validOrigins, origin)

	}

	if len(validOrigins) == 0 {

		return nil, fmt.Errorf("no valid origins found")

	}

	return validOrigins, nil

}

// validateOrigin validates a single origin.

func validateOrigin(origin string, isProduction bool) error {

	// Check for wildcard in production.

	if origin == "*" {

		if isProduction {

			return fmt.Errorf("wildcard origin '*' is not allowed in production environments")

		}

		// Wildcard is allowed in non-production environments.

		return nil

	}

	// Check protocol.

	if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {

		return fmt.Errorf("origin must start with http:// or https://")

	}

	// Check for partial wildcards (not allowed).

	if strings.Contains(origin, "*") && origin != "*" {

		return fmt.Errorf("origin contains wildcard but is not exact wildcard '*'")

	}

	// Check for spaces or other invalid characters.

	if strings.ContainsAny(origin, " \t\n\r") {

		return fmt.Errorf("origin contains invalid whitespace characters")

	}

	// Basic validation for malformed URLs.

	if strings.Contains(origin, "..") || strings.HasSuffix(origin, "/") {

		return fmt.Errorf("origin format is invalid")

	}

	return nil

}

// isDevelopmentEnvironment determines if we're running in development mode.

func isDevelopmentEnvironment() bool {

	envVars := []string{"GO_ENV", "NODE_ENV", "ENVIRONMENT", "ENV", "APP_ENV", "LLM_ENVIRONMENT"}

	for _, envVar := range envVars {

		value := strings.ToLower(GetEnvOrDefault(envVar, ""))

		switch value {

		case "development", "dev", "local", "test", "testing":

			return true

		case "production", "prod", "staging", "stage":

			return false

		}

	}

	return false // Default to production for security

}

func validateTLSFiles(certPath, keyPath string) error {

	// Check if certificate file exists and is readable.

	if _, err := os.Stat(certPath); err != nil {

		if os.IsNotExist(err) {

			return fmt.Errorf("certificate file does not exist: %s", certPath)

		}

		return fmt.Errorf("cannot access certificate file: %w", err)

	}

	// Check if key file exists and is readable.

	if _, err := os.Stat(keyPath); err != nil {

		if os.IsNotExist(err) {

			return fmt.Errorf("private key file does not exist: %s", keyPath)

		}

		return fmt.Errorf("cannot access private key file: %w", err)

	}

	return nil

}

// GetRAGEndpointPreference returns the endpoint preference configuration.

func (c *LLMProcessorConfig) GetRAGEndpointPreference() (bool, string) {

	return c.RAGPreferProcessEndpoint, c.RAGEndpointPath

}

// GetEffectiveRAGEndpoints returns the effective RAG endpoints based on configuration.

func (c *LLMProcessorConfig) GetEffectiveRAGEndpoints() (processEndpoint, healthEndpoint string) {

	baseURL := strings.TrimSuffix(c.RAGAPIURL, "/")

	// If custom endpoint path is specified, use it.

	if c.RAGEndpointPath != "" {

		processEndpoint = baseURL + c.RAGEndpointPath

	} else {

		// Auto-detect based on URL pattern.

		if strings.HasSuffix(c.RAGAPIURL, "/process_intent") {

			// Legacy pattern - use as configured.

			processEndpoint = c.RAGAPIURL

		} else if strings.HasSuffix(c.RAGAPIURL, "/process") {

			// New pattern - use as configured.

			processEndpoint = c.RAGAPIURL

		} else {

			// Base URL pattern - construct based on preference.

			if c.RAGPreferProcessEndpoint {

				processEndpoint = baseURL + "/process"

			} else {

				processEndpoint = baseURL + "/process_intent"

			}

		}

	}

	// Health endpoint is always /health.

	// Extract base URL from process endpoint.

	processBase := processEndpoint

	if strings.HasSuffix(processBase, "/process_intent") {

		processBase = strings.TrimSuffix(processBase, "/process_intent")

	} else if strings.HasSuffix(processBase, "/process") {

		processBase = strings.TrimSuffix(processBase, "/process")

	}

	healthEndpoint = processBase + "/health"

	return processEndpoint, healthEndpoint

}

// validateMetricsSecurityConfiguration validates the metrics security configuration.

func (c *LLMProcessorConfig) validateMetricsSecurityConfiguration() error {

	// Skip validation if metrics are disabled.

	if !c.MetricsEnabled {

		return nil

	}

	// If metrics are exposed publicly, warn about security implications.

	if c.ExposeMetricsPublicly && len(c.MetricsAllowedCIDRs) == 0 && len(c.MetricsAllowedIPs) == 0 {

		return fmt.Errorf("when exposing metrics publicly, consider setting METRICS_ALLOWED_CIDRS or METRICS_ALLOWED_IPS for additional security")

	}

	// Validate each CIDR block.

	for i, cidr := range c.MetricsAllowedCIDRs {

		if err := validateCIDR(cidr); err != nil {

			return fmt.Errorf("invalid CIDR at position %d: %w", i, err)

		}

	}

	// Validate each IP address.

	for i, ip := range c.MetricsAllowedIPs {

		if err := validateIP(ip); err != nil {

			return fmt.Errorf("invalid IP at position %d: %w", i, err)

		}

	}

	return nil

}

// parseAndValidateCIDRs parses a comma-separated list of CIDR blocks and validates them.

func parseAndValidateCIDRs(cidrsStr string) ([]string, error) {

	if strings.TrimSpace(cidrsStr) == "" {

		return nil, fmt.Errorf("CIDR string cannot be empty")

	}

	cidrs := strings.Split(cidrsStr, ",")

	// Pre-allocate slice with known capacity for better performance
	validCIDRs := make([]string, 0, len(cidrs))

	for _, cidr := range cidrs {

		cidr = strings.TrimSpace(cidr)

		if cidr == "" {

			continue

		}

		// Validate CIDR format.

		if err := validateCIDR(cidr); err != nil {

			return nil, fmt.Errorf("invalid CIDR '%s': %w", cidr, err)

		}

		validCIDRs = append(validCIDRs, cidr)

	}

	if len(validCIDRs) == 0 {

		return nil, fmt.Errorf("no valid CIDR blocks found")

	}

	return validCIDRs, nil

}

// validateCIDR validates a single CIDR block.

func validateCIDR(cidr string) error {

	if cidr == "" {

		return fmt.Errorf("CIDR cannot be empty")

	}

	// Parse CIDR using net.ParseCIDR.

	_, ipNet, err := net.ParseCIDR(cidr)

	if err != nil {

		return fmt.Errorf("invalid CIDR format: %w", err)

	}

	// Additional validation for security.

	if ipNet == nil {

		return fmt.Errorf("parsed CIDR is nil")

	}

	// Check for overly broad networks in production.

	if isProductionEnvironment() {

		ones, bits := ipNet.Mask.Size()

		// Warn about very broad networks.

		if bits == 32 && ones < 8 { // IPv4 networks broader than /8

			return fmt.Errorf("CIDR '%s' is too broad for production use (/%d)", cidr, ones)

		}

		if bits == 128 && ones < 64 { // IPv6 networks broader than /64

			return fmt.Errorf("CIDR '%s' is too broad for production use (/%d)", cidr, ones)

		}

	}

	return nil

}

// getDefaultPrivateNetworks returns the standard private network CIDR blocks.

func getDefaultPrivateNetworks() []string {

	return []string{

		"10.0.0.0/8", // Private Class A

		"172.16.0.0/12", // Private Class B

		"192.168.0.0/16", // Private Class C

		"127.0.0.0/8", // Loopback

		"::1/128", // IPv6 loopback

		"fc00::/7", // IPv6 unique local addresses

	}

}

// isProductionEnvironment determines if we're running in a production environment.

func isProductionEnvironment() bool {

	envVars := []string{"GO_ENV", "NODE_ENV", "ENVIRONMENT", "ENV", "APP_ENV", "LLM_ENVIRONMENT"}

	for _, envVar := range envVars {

		value := strings.ToLower(GetEnvOrDefault(envVar, ""))

		switch value {

		case "production", "prod":

			return true

		case "development", "dev", "local", "test", "testing", "staging", "stage":

			return false

		}

	}

	return false // Default to false for this specific validation

}

// GetMetricsAccessConfig returns the effective metrics access configuration.

func (c *LLMProcessorConfig) GetMetricsAccessConfig() (isPublic bool, allowedCIDRs []string) {

	return c.ExposeMetricsPublicly, c.MetricsAllowedCIDRs

}

// IsMetricsAccessAllowed checks if the given IP address is allowed to access metrics.

func (c *LLMProcessorConfig) IsMetricsAccessAllowed(clientIP string) bool {

	// If metrics are disabled, deny all access.

	if !c.MetricsEnabled {

		return false

	}

	// If metrics are public, allow all access.

	if c.ExposeMetricsPublicly {

		return true

	}

	// Check specific IP allowlist first.

	for _, allowedIP := range c.MetricsAllowedIPs {

		if clientIP == allowedIP {

			return true

		}

	}

	// If no CIDRs are configured and no IPs matched, deny access.

	if len(c.MetricsAllowedCIDRs) == 0 {

		return false

	}

	// Parse the client IP.

	ip := net.ParseIP(clientIP)

	if ip == nil {

		return false

	}

	// Check if the IP is in any of the allowed CIDR blocks.

	for _, cidrStr := range c.MetricsAllowedCIDRs {

		_, cidr, err := net.ParseCIDR(cidrStr)

		if err != nil {

			continue // Skip invalid CIDRs (should be caught during validation)

		}

		if cidr.Contains(ip) {

			return true

		}

	}

	return false

}

// parseAndValidateIPs parses a comma-separated list of IP addresses and validates them.

func parseAndValidateIPs(ipsStr string) ([]string, error) {

	if strings.TrimSpace(ipsStr) == "" {

		return nil, fmt.Errorf("IP string cannot be empty")

	}

	ips := strings.Split(ipsStr, ",")

	// Pre-allocate slice with known capacity for better performance
	validIPs := make([]string, 0, len(ips))

	for _, ip := range ips {

		ip = strings.TrimSpace(ip)

		if ip == "" {

			continue

		}

		// Validate IP format.

		if err := validateIP(ip); err != nil {

			return nil, fmt.Errorf("invalid IP '%s': %w", ip, err)

		}

		validIPs = append(validIPs, ip)

	}

	if len(validIPs) == 0 {

		return nil, fmt.Errorf("no valid IP addresses found")

	}

	return validIPs, nil

}

// validateIP validates a single IP address.

func validateIP(ip string) error {

	if ip == "" {

		return fmt.Errorf("IP cannot be empty")

	}

	// Parse IP using net.ParseIP.

	parsedIP := net.ParseIP(ip)

	if parsedIP == nil {

		return fmt.Errorf("invalid IP address format")

	}

	return nil

}

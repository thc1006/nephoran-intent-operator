package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LLMProcessorConfig holds configuration specific to the LLM processor service
type LLMProcessorConfig struct {
	// Service Configuration
	Port             string
	LogLevel         string
	ServiceVersion   string
	GracefulShutdown time.Duration

	// LLM Configuration
	LLMBackendType string
	LLMAPIKey      string
	LLMModelName   string
	LLMTimeout     time.Duration
	LLMMaxTokens   int

	// RAG Configuration
	RAGAPIURL  string
	RAGTimeout time.Duration
	RAGEnabled bool

	// Streaming Configuration
	StreamingEnabled     bool
	MaxConcurrentStreams int
	StreamTimeout        time.Duration

	// Context Management
	EnableContextBuilder bool
	MaxContextTokens     int
	ContextTTL           time.Duration

	// Security Configuration
	APIKeyRequired bool
	APIKey         string
	CORSEnabled    bool
	AllowedOrigins string

	// Performance Configuration
	RequestTimeout time.Duration
	MaxRequestSize int64

	// Circuit Breaker Configuration
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration

	// Rate Limiting
	RateLimitEnabled        bool
	RateLimitRequestsPerMin int
	RateLimitBurst          int

	// Retry Configuration
	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff string

	// OAuth2 Authentication Configuration
	AuthEnabled    bool
	AuthConfigFile string
	JWTSecretKey   string
	RequireAuth    bool
	AdminUsers     []string
	OperatorUsers  []string

	// Secret Management Configuration
	UseKubernetesSecrets bool
	SecretNamespace      string
}

// DefaultLLMProcessorConfig returns a configuration with sensible defaults
func DefaultLLMProcessorConfig() *LLMProcessorConfig {
	return &LLMProcessorConfig{
		Port:             "8080",
		LogLevel:         "info",
		ServiceVersion:   "v2.0.0",
		GracefulShutdown: 30 * time.Second,

		LLMBackendType: "rag",
		LLMModelName:   "gpt-4o-mini",
		LLMTimeout:     60 * time.Second,
		LLMMaxTokens:   2048,

		RAGAPIURL:  "http://rag-api:5001",
		RAGTimeout: 30 * time.Second,
		RAGEnabled: true,

		StreamingEnabled:     true,
		MaxConcurrentStreams: 100,
		StreamTimeout:        5 * time.Minute,

		EnableContextBuilder: true,
		MaxContextTokens:     6000,
		ContextTTL:           5 * time.Minute,

		APIKeyRequired: false,
		CORSEnabled:    true,
		AllowedOrigins: "*",

		RequestTimeout: 30 * time.Second,
		MaxRequestSize: 1048576, // 1MB

		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,

		RateLimitEnabled:        true,
		RateLimitRequestsPerMin: 60,
		RateLimitBurst:          10,

		MaxRetries:   3,
		RetryDelay:   1 * time.Second,
		RetryBackoff: "exponential",

		AuthEnabled:   true,
		RequireAuth:   true,
		AdminUsers:    []string{},
		OperatorUsers: []string{},

		UseKubernetesSecrets: true,
		SecretNamespace:      "nephoran-system",
	}
}

// LoadLLMProcessorConfig loads configuration from environment variables with validation
func LoadLLMProcessorConfig() (*LLMProcessorConfig, error) {
	cfg := DefaultLLMProcessorConfig()

	// Load configuration with validation
	var validationErrors []string

	// Service Configuration
	cfg.Port = getEnvWithValidation("PORT", cfg.Port, validatePort, &validationErrors)
	cfg.LogLevel = getEnvWithValidation("LOG_LEVEL", cfg.LogLevel, validateLogLevel, &validationErrors)
	cfg.ServiceVersion = getEnvOrDefault("SERVICE_VERSION", cfg.ServiceVersion)
	cfg.GracefulShutdown = parseDurationWithValidation("GRACEFUL_SHUTDOWN_TIMEOUT", cfg.GracefulShutdown, &validationErrors)

	// LLM Configuration
	cfg.LLMBackendType = getEnvWithValidation("LLM_BACKEND_TYPE", cfg.LLMBackendType, validateLLMBackendType, &validationErrors)
	// Load LLM API key from file or env based on backend type
	llmAPIKey, err := LoadLLMAPIKeyFromFile(cfg.LLMBackendType)
	if err != nil && cfg.LLMBackendType != "mock" && cfg.LLMBackendType != "rag" {
		validationErrors = append(validationErrors, fmt.Sprintf("LLM API Key: %v", err))
	}
	cfg.LLMAPIKey = llmAPIKey
	cfg.LLMModelName = getEnvOrDefault("LLM_MODEL_NAME", cfg.LLMModelName)
	cfg.LLMTimeout = parseDurationWithValidation("LLM_TIMEOUT", cfg.LLMTimeout, &validationErrors)
	cfg.LLMMaxTokens = parseIntWithValidation("LLM_MAX_TOKENS", cfg.LLMMaxTokens, validatePositiveInt, &validationErrors)

	// RAG Configuration
	cfg.RAGAPIURL = getEnvWithValidation("RAG_API_URL", cfg.RAGAPIURL, validateURL, &validationErrors)
	cfg.RAGTimeout = parseDurationWithValidation("RAG_TIMEOUT", cfg.RAGTimeout, &validationErrors)
	cfg.RAGEnabled = parseBoolWithDefault("RAG_ENABLED", cfg.RAGEnabled)

	// Streaming Configuration
	cfg.StreamingEnabled = parseBoolWithDefault("STREAMING_ENABLED", cfg.StreamingEnabled)
	cfg.MaxConcurrentStreams = parseIntWithValidation("MAX_CONCURRENT_STREAMS", cfg.MaxConcurrentStreams, validatePositiveInt, &validationErrors)
	cfg.StreamTimeout = parseDurationWithValidation("STREAM_TIMEOUT", cfg.StreamTimeout, &validationErrors)

	// Context Management
	cfg.EnableContextBuilder = parseBoolWithDefault("ENABLE_CONTEXT_BUILDER", cfg.EnableContextBuilder)
	cfg.MaxContextTokens = parseIntWithValidation("MAX_CONTEXT_TOKENS", cfg.MaxContextTokens, validatePositiveInt, &validationErrors)
	cfg.ContextTTL = parseDurationWithValidation("CONTEXT_TTL", cfg.ContextTTL, &validationErrors)

	// Security Configuration
	cfg.APIKeyRequired = parseBoolWithDefault("API_KEY_REQUIRED", cfg.APIKeyRequired)
	// Load API key from file or env
	apiKey, err := LoadAPIKeyFromFile()
	if err != nil && cfg.APIKeyRequired {
		validationErrors = append(validationErrors, fmt.Sprintf("API Key: %v", err))
	}
	cfg.APIKey = apiKey
	cfg.CORSEnabled = parseBoolWithDefault("CORS_ENABLED", cfg.CORSEnabled)
	cfg.AllowedOrigins = getEnvOrDefault("ALLOWED_ORIGINS", cfg.AllowedOrigins)

	// Performance Configuration
	cfg.RequestTimeout = parseDurationWithValidation("REQUEST_TIMEOUT", cfg.RequestTimeout, &validationErrors)
	cfg.MaxRequestSize = parseInt64WithValidation("MAX_REQUEST_SIZE", cfg.MaxRequestSize, validatePositiveInt64, &validationErrors)

	// Circuit Breaker Configuration
	cfg.CircuitBreakerEnabled = parseBoolWithDefault("CIRCUIT_BREAKER_ENABLED", cfg.CircuitBreakerEnabled)
	cfg.CircuitBreakerThreshold = parseIntWithValidation("CIRCUIT_BREAKER_THRESHOLD", cfg.CircuitBreakerThreshold, validatePositiveInt, &validationErrors)
	cfg.CircuitBreakerTimeout = parseDurationWithValidation("CIRCUIT_BREAKER_TIMEOUT", cfg.CircuitBreakerTimeout, &validationErrors)

	// Rate Limiting
	cfg.RateLimitEnabled = parseBoolWithDefault("RATE_LIMIT_ENABLED", cfg.RateLimitEnabled)
	cfg.RateLimitRequestsPerMin = parseIntWithValidation("RATE_LIMIT_REQUESTS_PER_MINUTE", cfg.RateLimitRequestsPerMin, validatePositiveInt, &validationErrors)
	cfg.RateLimitBurst = parseIntWithValidation("RATE_LIMIT_BURST", cfg.RateLimitBurst, validatePositiveInt, &validationErrors)

	// Retry Configuration
	cfg.MaxRetries = parseIntWithValidation("MAX_RETRIES", cfg.MaxRetries, validateNonNegativeInt, &validationErrors)
	cfg.RetryDelay = parseDurationWithValidation("RETRY_DELAY", cfg.RetryDelay, &validationErrors)
	cfg.RetryBackoff = getEnvWithValidation("RETRY_BACKOFF", cfg.RetryBackoff, validateRetryBackoff, &validationErrors)

	// OAuth2 Authentication Configuration
	cfg.AuthEnabled = parseBoolWithDefault("AUTH_ENABLED", cfg.AuthEnabled)
	cfg.AuthConfigFile = os.Getenv("AUTH_CONFIG_FILE")
	// Load JWT secret key from file or env
	jwtSecretKey, err := LoadJWTSecretKeyFromFile()
	if err != nil && cfg.AuthEnabled {
		validationErrors = append(validationErrors, fmt.Sprintf("JWT Secret Key: %v", err))
	}
	cfg.JWTSecretKey = jwtSecretKey
	cfg.RequireAuth = parseBoolWithDefault("REQUIRE_AUTH", cfg.RequireAuth)
	cfg.AdminUsers = parseStringSlice(getEnvOrDefault("ADMIN_USERS", ""))
	cfg.OperatorUsers = parseStringSlice(getEnvOrDefault("OPERATOR_USERS", ""))

	// Secret Management Configuration
	cfg.UseKubernetesSecrets = parseBoolWithDefault("USE_KUBERNETES_SECRETS", cfg.UseKubernetesSecrets)
	cfg.SecretNamespace = getEnvOrDefault("SECRET_NAMESPACE", cfg.SecretNamespace)

	// Return validation errors if any
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("configuration validation failed: %s", strings.Join(validationErrors, "; "))
	}

	// Perform additional cross-field validation
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate performs comprehensive validation of the configuration
func (c *LLMProcessorConfig) Validate() error {
	var errors []string

	// Validate required fields based on configuration
	if c.LLMAPIKey == "" && c.LLMBackendType != "mock" && c.LLMBackendType != "rag" {
		errors = append(errors, "OPENAI_API_KEY is required for non-mock/non-rag backends")
	}

	if c.AuthEnabled && c.JWTSecretKey == "" {
		errors = append(errors, "JWT_SECRET_KEY is required when authentication is enabled")
	}

	if c.APIKeyRequired && c.APIKey == "" {
		errors = append(errors, "API_KEY is required when API key authentication is enabled")
	}

	// Validate logical constraints
	if c.MaxConcurrentStreams > 1000 {
		errors = append(errors, "MAX_CONCURRENT_STREAMS should not exceed 1000 for performance reasons")
	}

	if c.MaxContextTokens > 32000 {
		errors = append(errors, "MAX_CONTEXT_TOKENS should not exceed 32000 for most models")
	}

	if c.CircuitBreakerThreshold > 50 {
		errors = append(errors, "CIRCUIT_BREAKER_THRESHOLD should be reasonable (≤50)")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// Helper functions for validation and parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvWithValidation(key, defaultValue string, validator func(string) error, errors *[]string) string {
	value := getEnvOrDefault(key, defaultValue)
	if err := validator(value); err != nil {
		*errors = append(*errors, fmt.Sprintf("%s: %v", key, err))
	}
	return value
}

func parseDurationWithValidation(key string, defaultValue time.Duration, errors *[]string) time.Duration {
	if valueStr := os.Getenv(key); valueStr != "" {
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
	if valueStr := os.Getenv(key); valueStr != "" {
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
	if valueStr := os.Getenv(key); valueStr != "" {
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
	if valueStr := os.Getenv(key); valueStr != "" {
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

// Validation functions

func validatePort(port string) error {
	if portNum, err := strconv.Atoi(port); err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port number, must be between 1 and 65535")
	}
	return nil
}

func validateLogLevel(level string) error {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if strings.ToLower(level) == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid log level, must be one of: %s", strings.Join(validLevels, ", "))
}

func validateLLMBackendType(backendType string) error {
	validTypes := []string{"openai", "mistral", "rag", "mock"}
	for _, valid := range validTypes {
		if strings.ToLower(backendType) == valid {
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
		if strings.ToLower(backoff) == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid retry backoff, must be one of: %s", strings.Join(validBackoffs, ", "))
}

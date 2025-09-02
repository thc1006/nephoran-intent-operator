package config

import (
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the environment helpers in the Nephoran Intent Operator.

// This file provides real-world examples that can be used as reference for other parts of the codebase.

func ExampleUsage() {
	// Basic string configuration with fallback.

	port := GetEnvOrDefault("PORT", "8080")

	fmt.Printf("Server will listen on port: %s\n", port)

	// Boolean configuration for feature flags.

	debugMode := GetBoolEnv("DEBUG", false)

	enableTLS := GetBoolEnv("TLS_ENABLED", true)

	fmt.Printf("Debug mode: %v, TLS enabled: %v\n", debugMode, enableTLS)

	// Duration configuration for timeouts and intervals.

	requestTimeout := GetDurationEnv("REQUEST_TIMEOUT", 30*time.Second)

	healthCheckInterval := GetDurationEnv("HEALTH_CHECK_INTERVAL", 10*time.Second)

	fmt.Printf("Request timeout: %v, Health check interval: %v\n", requestTimeout, healthCheckInterval)

	// Integer configuration for limits and thresholds.

	maxConnections := GetIntEnv("MAX_CONNECTIONS", 100)

	workerCount := GetIntEnv("WORKER_COUNT", 4)

	fmt.Printf("Max connections: %d, Worker count: %d\n", maxConnections, workerCount)

	// String slice configuration for lists.

	allowedOrigins := GetStringSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})

	adminUsers := GetStringSliceEnv("ADMIN_USERS", []string{})

	fmt.Printf("Allowed origins: %v, Admin users: %v\n", allowedOrigins, adminUsers)

	// Configuration with validation.

	validatedPort, err := GetEnvWithValidation("API_PORT", "8080", ValidatePort)

	if err != nil {
		log.Printf("Port validation error: %v", err)
	} else {
		fmt.Printf("Validated API port: %s\n", validatedPort)
	}

	// Environment-specific configuration.

	environment, err := GetEnvWithValidation("ENVIRONMENT", "development",

		ValidateOneOfIgnoreCase([]string{"development", "staging", "production"}))

	if err != nil {
		log.Printf("Environment validation error: %v", err)
	} else {
		fmt.Printf("Environment: %s\n", environment)
	}

	// Critical configuration that must be present.
	// This would panic if API_KEY is not set.

	// apiKey := MustGetEnv("API_KEY").

	// fmt.Printf("API Key loaded: %s\n", apiKey[:8]+"...") // Only show first 8 chars.
}

// ExampleLLMProcessorConfig demonstrates how to refactor existing configuration loading.

// to use the new environment helpers, replacing the duplicate getEnvOrDefault functions.

func ExampleLLMProcessorConfig() {
	// Replace existing patterns like:.

	// cfg.Port = getEnvOrDefault("PORT", cfg.Port).

	// With:.

	port := GetEnvOrDefault("PORT", "8080")

	// Replace boolean parsing patterns like:.

	// if val := os.Getenv("ENABLE_STREAMING"); val != "" {.

	//     cfg.StreamingEnabled = val == "true" || val == "1".

	// }.

	// With:.

	streamingEnabled := GetBoolEnv("ENABLE_STREAMING", false)

	// Replace duration parsing patterns like:.

	// if val := os.Getenv("TIMEOUT"); val != "" {.

	//     if d, err := time.ParseDuration(val); err == nil {.

	//         cfg.Timeout = d.

	//     }.

	// }.

	// With:.

	timeout := GetDurationEnv("TIMEOUT", 30*time.Second)

	// Replace integer parsing patterns like:.

	// if val := os.Getenv("MAX_TOKENS"); val != "" {.

	//     if i, err := strconv.Atoi(val); err == nil {.

	//         cfg.MaxTokens = i.

	//     }.

	// }.

	// With:.

	maxTokens := GetIntEnv("MAX_TOKENS", 2048)

	// Replace string slice parsing patterns like:.

	// if val := os.Getenv("ALLOWED_ORIGINS"); val != "" {.

	//     result := []string{}.

	//     for _, item := range strings.Split(val, ",") {.

	//         if trimmed := strings.TrimSpace(item); trimmed != "" {.

	//             result = append(result, trimmed).

	//         }.

	//     }.

	//     cfg.AllowedOrigins = result.

	// }.

	// With:.

	allowedOrigins := GetStringSliceEnv("ALLOWED_ORIGINS", []string{"http://localhost:3000"})

	fmt.Printf("LLM Config - Port: %s, Streaming: %v, Timeout: %v, MaxTokens: %d, Origins: %v\n",

		port, streamingEnabled, timeout, maxTokens, allowedOrigins)
}

// ExampleAuthConfig demonstrates replacing auth configuration patterns.

func ExampleAuthConfig() {
	// Replace patterns like:.

	// func getBoolEnv(key string, defaultValue bool) bool {.

	//     if value := os.Getenv(key); value != "" {.

	//         return value == "true" || value == "1" || value == "yes".

	//     }.

	//     return defaultValue.

	// }.

	// With:.

	authEnabled := GetBoolEnv("AUTH_ENABLED", false)

	rbacEnabled := GetBoolEnv("RBAC_ENABLED", true)

	// Replace duration patterns like:.

	// func getDurationEnv(key string, defaultValue time.Duration) time.Duration {.

	//     if value := os.Getenv(key); value != "" {.

	//         if duration, err := time.ParseDuration(value); err == nil {.

	//             return duration.

	//         }.

	//     }.

	//     return defaultValue.

	// }.

	// With:.

	tokenTTL := GetDurationEnv("TOKEN_TTL", 24*time.Hour)

	refreshTTL := GetDurationEnv("REFRESH_TTL", 7*24*time.Hour)

	// Replace string slice patterns like:.

	// func getStringSliceEnv(key string, defaultValue []string) []string {.

	//     if value := os.Getenv(key); value != "" {.

	//         result := []string{}.

	//         for _, item := range strings.Split(value, ",") {.

	//             if trimmed := strings.TrimSpace(item); trimmed != "" {.

	//                 result = append(result, trimmed).

	//             }.

	//         }.

	//         if len(result) > 0 {.

	//             return result.

	//         }.

	//     }.

	//     return defaultValue.

	// }.

	// With:.

	adminUsers := GetStringSliceEnv("ADMIN_USERS", []string{})

	operatorUsers := GetStringSliceEnv("OPERATOR_USERS", []string{})

	fmt.Printf("Auth Config - Enabled: %v, RBAC: %v, Token TTL: %v, Refresh TTL: %v\n",

		authEnabled, rbacEnabled, tokenTTL, refreshTTL)

	fmt.Printf("Admin users: %v, Operator users: %v\n", adminUsers, operatorUsers)
}

// ExampleValidationUsage demonstrates advanced validation patterns.

func ExampleValidationUsage() {
	// URL validation.

	ragURL, err := GetEnvWithValidation("RAG_API_URL", "http://localhost:5001", ValidateURL)
	if err != nil {
		log.Printf("RAG URL validation error: %v", err)
	}

	// Port validation.

	metricsPort, err := GetEnvWithValidation("METRICS_PORT", "9090", ValidatePort)
	if err != nil {
		log.Printf("Metrics port validation error: %v", err)
	}

	// Log level validation.

	logLevel, err := GetEnvWithValidation("LOG_LEVEL", "info", ValidateLogLevel)
	if err != nil {
		log.Printf("Log level validation error: %v", err)
	}

	// Custom validation with ranges.

	maxRetries, err := GetIntEnvWithValidation("MAX_RETRIES", 3, ValidateIntRange(0, 10))
	if err != nil {
		log.Printf("Max retries validation error: %v", err)
	}

	// Duration validation with ranges.

	circuitBreakerTimeout, err := GetDurationEnvWithValidation("CIRCUIT_BREAKER_TIMEOUT",

		30*time.Second, ValidateDurationRange(1*time.Second, 5*time.Minute))
	if err != nil {
		log.Printf("Circuit breaker timeout validation error: %v", err)
	}

	// Custom validation for specific values.

	backendType, err := GetEnvWithValidation("LLM_BACKEND_TYPE", "rag",

		ValidateOneOf([]string{"openai", "mistral", "rag", "mock"}))
	if err != nil {
		log.Printf("Backend type validation error: %v", err)
	}

	fmt.Printf("Validated config - RAG URL: %s, Metrics port: %s, Log level: %s\n",

		ragURL, metricsPort, logLevel)

	fmt.Printf("Max retries: %d, CB timeout: %v, Backend: %s\n",

		maxRetries, circuitBreakerTimeout, backendType)
}

// ExampleMigrationPattern shows how to migrate existing code.

func ExampleMigrationPattern() {
	fmt.Println("=== Migration Examples ===")

	// BEFORE: Duplicate helper functions in each file.

	// func getEnvOrDefault(key, defaultValue string) string {.

	//     if value := os.Getenv(key); value != "" {.

	//         return value.

	//     }.

	//     return defaultValue.

	// }.

	// AFTER: Use centralized helper.

	llmProcessorURL := GetEnvOrDefault("LLM_PROCESSOR_URL", "http://llm-processor:8080")

	// BEFORE: Manual boolean parsing with inconsistent logic.

	// enabled := false.

	// if val := os.Getenv("FEATURE_ENABLED"); val != "" {.

	//     enabled = val == "true" || val == "1".

	// }.

	// AFTER: Consistent boolean parsing.

	featureEnabled := GetBoolEnv("FEATURE_ENABLED", false)

	// BEFORE: Error-prone duration parsing.

	// var timeout time.Duration = 30 * time.Second.

	// if val := os.Getenv("TIMEOUT"); val != "" {.

	//     if d, err := time.ParseDuration(val); err == nil {.

	//         timeout = d.

	//     }.

	// }.

	// AFTER: Safe duration parsing.

	timeout := GetDurationEnv("TIMEOUT", 30*time.Second)

	fmt.Printf("Migrated config - URL: %s, Enabled: %v, Timeout: %v\n",

		llmProcessorURL, featureEnabled, timeout)
}

// ExampleErrorHandling demonstrates proper error handling patterns.

func ExampleErrorHandling() {
	fmt.Println("=== Error Handling Examples ===")

	// Validation with error handling.

	if port, err := GetEnvWithValidation("API_PORT", "8080", ValidatePort); err != nil {

		log.Printf("Invalid port configuration: %v, using default", err)

		// Use default or handle error appropriately.

		_ = port

	}

	// Required configuration with panic recovery.

	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Missing required configuration: %v", r)

				// Handle gracefully or exit.
			}
		}()

		// This will panic if not set.

		// secretKey := MustGetEnv("SECRET_KEY").

		// Use secretKey...
	}()

	// Conditional validation.

	if IsSet("OPTIONAL_FEATURE") {

		featureConfig, err := GetEnvWithValidation("OPTIONAL_FEATURE", "", ValidateNonEmpty)

		if err != nil {
			log.Printf("Optional feature misconfigured: %v", err)
		} else {
			fmt.Printf("Optional feature configured: %s\n", featureConfig)
		}

	}
}

// ExamplePerformancePatterns demonstrates efficient usage patterns.

func ExamplePerformancePatterns() {
	fmt.Println("=== Performance Examples ===")

	// Load configuration once at startup.

	type Config struct {
		Port string

		DebugEnabled bool

		RequestTimeout time.Duration

		MaxConnections int

		AllowedOrigins []string
	}

	config := Config{
		Port: GetEnvOrDefault("PORT", "8080"),

		DebugEnabled: GetBoolEnv("DEBUG", false),

		RequestTimeout: GetDurationEnv("REQUEST_TIMEOUT", 30*time.Second),

		MaxConnections: GetIntEnv("MAX_CONNECTIONS", 100),

		AllowedOrigins: GetStringSliceEnv("CORS_ORIGINS", []string{"*"}),
	}

	// Use throughout application without re-parsing.

	fmt.Printf("Loaded config: %+v\n", config)

	// For debugging: show all environment keys with specific prefix.

	fmt.Println("App-specific environment variables:")

	for _, key := range GetEnvKeys() {
		if len(key) >= 4 && key[:4] == "APP_" {
			// Only show non-sensitive variables.

			if key != "APP_SECRET_KEY" && key != "APP_PASSWORD" {
				fmt.Printf("  %s=%s\n", key, GetEnvOrDefault(key, ""))
			}
		}
	}
}

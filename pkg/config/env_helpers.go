
package config



import (

	"fmt"

	"os"

	"strconv"

	"strings"

	"time"

)



// GetEnvOrDefault retrieves the value of the environment variable named by key.

// If the variable is not present or empty, it returns the defaultValue.

//

// This is the primary function for simple environment variable retrieval with fallback.

//

// Example:.

//

//	port := GetEnvOrDefault("PORT", "8080")

//	dbURL := GetEnvOrDefault("DATABASE_URL", "localhost:5432")

func GetEnvOrDefault(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {

		return value

	}

	return defaultValue

}



// GetBoolEnv retrieves a boolean value from the environment variable.

// It accepts "true", "1", "yes", "on" (case-insensitive) as true values.

// All other values or missing variables return the defaultValue.

//

// Example:.

//

//	debug := GetBoolEnv("DEBUG", false)

//	tlsEnabled := GetBoolEnv("TLS_ENABLED", true)

func GetBoolEnv(key string, defaultValue bool) bool {

	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))

	if value == "" {

		return defaultValue

	}



	// Accept common truthy values.

	switch value {

	case "true", "1", "yes", "on", "enable", "enabled":

		return true

	case "false", "0", "no", "off", "disable", "disabled":

		return false

	default:

		// Invalid value, return default.

		return defaultValue

	}

}



// GetDurationEnv retrieves a duration value from the environment variable.

// The value should be a valid duration string parseable by time.ParseDuration.

// If parsing fails or the variable is missing, returns the defaultValue.

//

// Example:.

//

//	timeout := GetDurationEnv("REQUEST_TIMEOUT", 30*time.Second)

//	ttl := GetDurationEnv("CACHE_TTL", 5*time.Minute)

func GetDurationEnv(key string, defaultValue time.Duration) time.Duration {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		return defaultValue

	}



	duration, err := time.ParseDuration(value)

	if err != nil {

		return defaultValue

	}

	return duration

}



// GetStringSliceEnv retrieves a string slice from the environment variable.

// The value should be comma-separated. Empty items and whitespace are automatically trimmed.

// Returns defaultValue if the environment variable is missing or results in an empty slice.

//

// Example:.

//

//	origins := GetStringSliceEnv("ALLOWED_ORIGINS", []string{"http://localhost:3000"})

//	adminUsers := GetStringSliceEnv("ADMIN_USERS", []string{})

func GetStringSliceEnv(key string, defaultValue []string) []string {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		return defaultValue

	}



	var result []string

	for _, item := range strings.Split(value, ",") {

		if trimmed := strings.TrimSpace(item); trimmed != "" {

			result = append(result, trimmed)

		}

	}



	if len(result) == 0 {

		return defaultValue

	}

	return result

}



// GetIntEnv retrieves an integer value from the environment variable.

// If parsing fails or the variable is missing, returns the defaultValue.

//

// Example:.

//

//	maxConnections := GetIntEnv("MAX_CONNECTIONS", 100)

//	port := GetIntEnv("PORT", 8080)

func GetIntEnv(key string, defaultValue int) int {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		return defaultValue

	}



	intValue, err := strconv.Atoi(value)

	if err != nil {

		return defaultValue

	}

	return intValue

}



// GetInt64Env retrieves an int64 value from the environment variable.

// If parsing fails or the variable is missing, returns the defaultValue.

//

// Example:.

//

//	maxFileSize := GetInt64Env("MAX_FILE_SIZE", 1024*1024)

//	timeout := GetInt64Env("TIMEOUT_MS", 30000)

func GetInt64Env(key string, defaultValue int64) int64 {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		return defaultValue

	}



	intValue, err := strconv.ParseInt(value, 10, 64)

	if err != nil {

		return defaultValue

	}

	return intValue

}



// GetFloatEnv retrieves a float64 value from the environment variable.

// If parsing fails or the variable is missing, returns the defaultValue.

//

// Example:.

//

//	threshold := GetFloatEnv("THRESHOLD", 0.95)

//	factor := GetFloatEnv("SCALING_FACTOR", 1.5)

func GetFloatEnv(key string, defaultValue float64) float64 {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		return defaultValue

	}



	floatValue, err := strconv.ParseFloat(value, 64)

	if err != nil {

		return defaultValue

	}

	return floatValue

}



// GetEnvWithValidation retrieves an environment variable with validation.

// If validation fails, it returns the defaultValue and the validation error.

// If the environment variable is missing, it validates the defaultValue.

//

// Example:.

//

//	port, err := GetEnvWithValidation("PORT", "8080", validatePort)

//	if err != nil {

//	    log.Fatal(err)

//	}

//

// Where validatePort is a function that validates port numbers:.

//

//	func validatePort(value string) error {

//	    port, err := strconv.Atoi(value)

//	    if err != nil || port < 1 || port > 65535 {

//	        return fmt.Errorf("invalid port: %s", value)

//	    }

//	    return nil

//	}

func GetEnvWithValidation(key, defaultValue string, validator func(string) error) (string, error) {

	value := GetEnvOrDefault(key, defaultValue)

	if err := validator(value); err != nil {

		return defaultValue, fmt.Errorf("%s: %w", key, err)

	}

	return value, nil

}



// GetIntEnvWithValidation retrieves an integer environment variable with validation.

// If parsing or validation fails, returns the defaultValue and an error.

//

// Example:.

//

//	maxRetries, err := GetIntEnvWithValidation("MAX_RETRIES", 3, validateNonNegativeInt)

//	if err != nil {

//	    log.Printf("Warning: %v, using default", err)

//	}

func GetIntEnvWithValidation(key string, defaultValue int, validator func(int) error) (int, error) {

	value := strings.TrimSpace(os.Getenv(key))

	var intValue int



	if value == "" {

		intValue = defaultValue

	} else {

		var err error

		intValue, err = strconv.Atoi(value)

		if err != nil {

			return defaultValue, fmt.Errorf("%s: invalid integer format: %w", key, err)

		}

	}



	if err := validator(intValue); err != nil {

		return defaultValue, fmt.Errorf("%s: %w", key, err)

	}

	return intValue, nil

}



// GetDurationEnvWithValidation retrieves a duration environment variable with validation.

// If parsing or validation fails, returns the defaultValue and an error.

//

// Example:.

//

//	timeout, err := GetDurationEnvWithValidation("TIMEOUT", 30*time.Second, validatePositiveDuration)

//	if err != nil {

//	    log.Printf("Warning: %v, using default", err)

//	}

func GetDurationEnvWithValidation(key string, defaultValue time.Duration, validator func(time.Duration) error) (time.Duration, error) {

	value := strings.TrimSpace(os.Getenv(key))

	var duration time.Duration



	if value == "" {

		duration = defaultValue

	} else {

		var err error

		duration, err = time.ParseDuration(value)

		if err != nil {

			return defaultValue, fmt.Errorf("%s: invalid duration format: %w", key, err)

		}

	}



	if err := validator(duration); err != nil {

		return defaultValue, fmt.Errorf("%s: %w", key, err)

	}

	return duration, nil

}



// Common validator functions.



// ValidateNonEmpty validates that a string is not empty after trimming whitespace.

func ValidateNonEmpty(value string) error {

	if strings.TrimSpace(value) == "" {

		return fmt.Errorf("value cannot be empty")

	}

	return nil

}



// ValidatePort validates that a string represents a valid port number (1-65535).

func ValidatePort(value string) error {

	port, err := strconv.Atoi(value)

	if err != nil {

		return fmt.Errorf("invalid port number format")

	}

	if port < 1 || port > 65535 {

		return fmt.Errorf("port must be between 1 and 65535, got %d", port)

	}

	return nil

}



// ValidatePositiveInt validates that an integer is positive (> 0).

func ValidatePositiveInt(value int) error {

	if value <= 0 {

		return fmt.Errorf("value must be positive, got %d", value)

	}

	return nil

}



// ValidateNonNegativeInt validates that an integer is non-negative (>= 0).

func ValidateNonNegativeInt(value int) error {

	if value < 0 {

		return fmt.Errorf("value must be non-negative, got %d", value)

	}

	return nil

}



// ValidatePositiveDuration validates that a duration is positive.

func ValidatePositiveDuration(value time.Duration) error {

	if value <= 0 {

		return fmt.Errorf("duration must be positive, got %v", value)

	}

	return nil

}



// ValidateNonNegativeDuration validates that a duration is non-negative.

func ValidateNonNegativeDuration(value time.Duration) error {

	if value < 0 {

		return fmt.Errorf("duration must be non-negative, got %v", value)

	}

	return nil

}



// ValidateURL validates that a string is a valid HTTP/HTTPS URL.

func ValidateURL(value string) error {

	value = strings.TrimSpace(value)

	if value == "" {

		return fmt.Errorf("URL cannot be empty")

	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {

		return fmt.Errorf("URL must start with http:// or https://, got: %s", value)

	}

	return nil

}



// ValidateLogLevel validates common log level values.

func ValidateLogLevel(value string) error {

	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic", "trace"}

	level := strings.ToLower(strings.TrimSpace(value))



	for _, valid := range validLevels {

		if level == valid {

			return nil

		}

	}

	return fmt.Errorf("invalid log level '%s', must be one of: %s", value, strings.Join(validLevels, ", "))

}



// ValidateOneOf creates a validator that checks if the value is one of the allowed values.

// The comparison is case-sensitive.

//

// Example:.

//

//	validateBackend := ValidateOneOf([]string{"redis", "memory", "disk"})

//	backend, err := GetEnvWithValidation("CACHE_BACKEND", "memory", validateBackend)

func ValidateOneOf(allowedValues []string) func(string) error {

	return func(value string) error {

		value = strings.TrimSpace(value)

		for _, allowed := range allowedValues {

			if value == allowed {

				return nil

			}

		}

		return fmt.Errorf("invalid value '%s', must be one of: %s", value, strings.Join(allowedValues, ", "))

	}

}



// ValidateOneOfIgnoreCase creates a validator that checks if the value is one of the allowed values.

// The comparison is case-insensitive.

//

// Example:.

//

//	validateEnv := ValidateOneOfIgnoreCase([]string{"development", "staging", "production"})

//	env, err := GetEnvWithValidation("ENVIRONMENT", "development", validateEnv)

func ValidateOneOfIgnoreCase(allowedValues []string) func(string) error {

	return func(value string) error {

		value = strings.TrimSpace(value)

		for _, allowed := range allowedValues {

			if strings.EqualFold(value, allowed) {

				return nil

			}

		}

		return fmt.Errorf("invalid value '%s', must be one of: %s", value, strings.Join(allowedValues, ", "))

	}

}



// ValidateIntRange creates a validator that checks if an integer is within a specified range (inclusive).

//

// Example:.

//

//	validatePort := ValidateIntRange(1024, 65535)

//	port, err := GetIntEnvWithValidation("PORT", 8080, validatePort)

func ValidateIntRange(min, max int) func(int) error {

	return func(value int) error {

		if value < min || value > max {

			return fmt.Errorf("value must be between %d and %d (inclusive), got %d", min, max, value)

		}

		return nil

	}

}



// ValidateDurationRange creates a validator that checks if a duration is within a specified range (inclusive).

//

// Example:.

//

//	validateTimeout := ValidateDurationRange(1*time.Second, 5*time.Minute)

//	timeout, err := GetDurationEnvWithValidation("TIMEOUT", 30*time.Second, validateTimeout)

func ValidateDurationRange(min, max time.Duration) func(time.Duration) error {

	return func(value time.Duration) error {

		if value < min || value > max {

			return fmt.Errorf("duration must be between %v and %v (inclusive), got %v", min, max, value)

		}

		return nil

	}

}



// MustGetEnv retrieves an environment variable and panics if it's not set or empty.

// This should only be used for critical configuration that must be present.

//

// Example:.

//

//	apiKey := MustGetEnv("API_KEY")  // Panics if API_KEY is not set

func MustGetEnv(key string) string {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {

		panic(fmt.Sprintf("required environment variable %s is not set", key))

	}

	return value

}



// MustGetEnvWithValidation retrieves an environment variable with validation and panics on failure.

// This should only be used for critical configuration that must be present and valid.

//

// Example:.

//

//	port := MustGetEnvWithValidation("PORT", ValidatePort)

func MustGetEnvWithValidation(key string, validator func(string) error) string {

	value := MustGetEnv(key)

	if err := validator(value); err != nil {

		panic(fmt.Sprintf("environment variable %s validation failed: %v", key, err))

	}

	return value

}



// IsSet checks if an environment variable is set (even if empty).

// This is useful to distinguish between unset variables and empty values.

//

// Example:.

//

//	if IsSet("DEBUG") {

//	    // DEBUG environment variable was explicitly set

//	}

func IsSet(key string) bool {

	_, exists := os.LookupEnv(key)

	return exists

}



// GetEnvKeys returns all environment variable keys.

// This is mainly useful for debugging and configuration discovery.

//

// Example:.

//

//	keys := GetEnvKeys()

//	for _, key := range keys {

//	    if strings.HasPrefix(key, "APP_") {

//	        fmt.Printf("%s=%s\n", key, os.Getenv(key))

//	    }

//	}

func GetEnvKeys() []string {

	var keys []string

	for _, env := range os.Environ() {

		if eq := strings.Index(env, "="); eq >= 0 {

			keys = append(keys, env[:eq])

		}

	}

	return keys

}


# Environment Helpers Migration Guide

This guide demonstrates how to migrate from duplicate environment variable helper functions to the centralized `pkg/config/env_helpers.go` implementation.

## Overview

The codebase had multiple duplicate implementations of environment variable helpers across different packages:

- `pkg/auth/config.go` - `getEnv`, `getBoolEnv`, `getDurationEnv`, `getStringSliceEnv`
- `pkg/config/llm_processor.go` - `getEnvOrDefault`, `parseBoolWithDefault`, `parseDurationWithValidation`
- `pkg/services/llm_processor.go` - `getEnvOrDefault`
- `pkg/monitoring/opentelemetry.go` - `getEnv`
- Multiple examples and test files

## Centralized Functions

The new `env_helpers.go` provides these exported functions:

### Basic Functions
- `GetEnvOrDefault(key, defaultVal string) string`
- `GetBoolEnv(key string, defaultValue bool) bool`
- `GetDurationEnv(key string, defaultValue time.Duration) time.Duration`
- `GetStringSliceEnv(key string, defaultValue []string) []string`
- `GetIntEnv(key string, defaultValue int) int`
- `GetInt64Env(key string, defaultValue int64) int64`
- `GetFloatEnv(key string, defaultValue float64) float64`

### Validation Functions
- `GetEnvWithValidation(key, defaultValue string, validator func(string) error) (string, error)`
- `GetIntEnvWithValidation(key string, defaultValue int, validator func(int) error) (int, error)`
- `GetDurationEnvWithValidation(key string, defaultValue time.Duration, validator func(time.Duration) error) (time.Duration, error)`

### Utility Functions
- `MustGetEnv(key string) string` - Panics if not set
- `IsSet(key string) bool` - Checks if environment variable is set
- `GetEnvKeys() []string` - Returns all environment keys

### Built-in Validators
- `ValidateNonEmpty(value string) error`
- `ValidatePort(value string) error`
- `ValidateURL(value string) error`
- `ValidateLogLevel(value string) error`
- `ValidatePositiveInt(value int) error`
- `ValidateNonNegativeInt(value int) error`
- `ValidateOneOf(allowedValues []string) func(string) error`
- `ValidateOneOfIgnoreCase(allowedValues []string) func(string) error`
- `ValidateIntRange(min, max int) func(int) error`
- `ValidateDurationRange(min, max time.Duration) func(time.Duration) error`

## Migration Examples

### 1. Basic String Environment Variables

**Before (multiple files had variations):**
```go
func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// Usage
port := getEnvOrDefault("PORT", "8080")
```

**After:**
```go
import "github.com/thc1006/nephoran-intent-operator/pkg/config"

// Usage (remove local helper functions)
port := config.GetEnvOrDefault("PORT", "8080")
```

### 2. Boolean Environment Variables

**Before (from pkg/auth/config.go):**
```go
func getBoolEnv(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        return value == "true" || value == "1" || value == "yes"
    }
    return defaultValue
}
```

**After:**
```go
// More robust - handles "true", "1", "yes", "on", "enable", "enabled" (case-insensitive)
enabled := config.GetBoolEnv("AUTH_ENABLED", false)
```

### 3. Duration Environment Variables

**Before (from pkg/auth/config.go):**
```go
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}
```

**After:**
```go
timeout := config.GetDurationEnv("TOKEN_TTL", 24*time.Hour)
```

### 4. String Slice Environment Variables

**Before (from pkg/auth/config.go):**
```go
func getStringSliceEnv(key string, defaultValue []string) []string {
    if value := os.Getenv(key); value != "" {
        result := []string{}
        for _, item := range strings.Split(value, ",") {
            if trimmed := strings.TrimSpace(item); trimmed != "" {
                result = append(result, trimmed)
            }
        }
        if len(result) > 0 {
            return result
        }
    }
    return defaultValue
}
```

**After:**
```go
adminUsers := config.GetStringSliceEnv("ADMIN_USERS", []string{})
```

### 5. Environment Variables with Validation

**Before (from pkg/config/llm_processor.go):**
```go
func getEnvWithValidation(key, defaultValue string, validator func(string) error, errors *[]string) string {
    value := getEnvOrDefault(key, defaultValue)
    if err := validator(value); err != nil {
        *errors = append(*errors, fmt.Sprintf("%s: %v", key, err))
        return defaultValue
    }
    return value
}

// Usage with error accumulation
var validationErrors []string
port := getEnvWithValidation("PORT", "8080", validatePort, &validationErrors)
```

**After:**
```go
// Cleaner error handling
port, err := config.GetEnvWithValidation("PORT", "8080", config.ValidatePort)
if err != nil {
    log.Printf("Port validation failed: %v", err)
}
```

### 6. Integer Environment Variables

**Before (from pkg/config/llm_processor.go):**
```go
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
```

**After:**
```go
maxTokens, err := config.GetIntEnvWithValidation("MAX_TOKENS", 2048, config.ValidatePositiveInt)
if err != nil {
    log.Printf("Max tokens validation failed: %v", err)
}
```

## Migration Steps

1. **Identify Usage**: Find all files using local environment helper functions
2. **Import Package**: Add `"github.com/thc1006/nephoran-intent-operator/pkg/config"` import
3. **Replace Calls**: Change function calls to use `config.` prefix
4. **Remove Duplicates**: Delete local helper function implementations
5. **Update Error Handling**: Adapt validation error handling patterns
6. **Test**: Ensure all functionality works as expected

## Benefits

1. **Consistency**: All environment variable parsing follows the same patterns
2. **Maintainability**: Single source of truth for environment helpers
3. **Robustness**: Better error handling and edge case coverage
4. **Features**: Additional helpers like `MustGetEnv`, `IsSet`, validation functions
5. **Performance**: Optimized implementations with benchmark validation
6. **Testing**: Comprehensive test coverage (90%+ coverage)

## Files That Need Migration

Based on the codebase analysis, these files contain duplicate implementations that should be migrated:

- `pkg/auth/config.go` - Replace `getEnv`, `getBoolEnv`, `getDurationEnv`, `getStringSliceEnv`
- `pkg/config/llm_processor.go` - Replace `getEnvOrDefault`, parsing functions
- `pkg/services/llm_processor.go` - Replace `getEnvOrDefault`
- `pkg/monitoring/opentelemetry.go` - Replace `getEnv`
- `examples/xapps/traffic-steering/main.go` - Replace `getEnv`
- `examples/xapps/kmp-monitor/main.go` - Replace `getEnv`
- `tests/compliance/oran_compliance_test.go` - Replace `getEnvOrDefault`
- `pkg/oran/e2/examples/kmp_analytics_xapp.go` - Replace `getEnvOrDefault`

## Testing

The new helpers include comprehensive tests:
- 90%+ test coverage
- Table-driven tests for all functions
- Edge case handling
- Benchmark tests for performance validation
- Error handling validation

Run tests with:
```bash
go test ./pkg/config -run "TestGetEnvOrDefault|TestGetBoolEnv|TestGetDurationEnv|TestGetStringSliceEnv|TestGetIntEnv|TestGetEnvWithValidation|TestValidators" -v
```

Run benchmarks with:
```bash
go test ./pkg/config -bench="Benchmark.*" -run="^$"
```
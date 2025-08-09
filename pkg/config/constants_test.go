package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConstants(t *testing.T) {
	// Test loading default constants
	constants := LoadConstants()

	// Test basic constants
	if constants.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", constants.MaxRetries)
	}

	if constants.RetryDelay != 30*time.Second {
		t.Errorf("Expected RetryDelay to be 30s, got %v", constants.RetryDelay)
	}

	if constants.MaxInputLength != 10000 {
		t.Errorf("Expected MaxInputLength to be 10000, got %d", constants.MaxInputLength)
	}

	if len(constants.AllowedDomains) == 0 {
		t.Error("Expected AllowedDomains to have default values")
	}

	if constants.ContextBoundary == "" {
		t.Error("Expected ContextBoundary to have a default value")
	}
}

func TestLoadConstantsWithEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("NEPHORAN_MAX_RETRIES", "5")
	os.Setenv("NEPHORAN_RETRY_DELAY", "45s")
	os.Setenv("NEPHORAN_MAX_INPUT_LENGTH", "15000")
	os.Setenv("NEPHORAN_LLM_TIMEOUT", "60s")
	
	defer func() {
		// Clean up environment variables
		os.Unsetenv("NEPHORAN_MAX_RETRIES")
		os.Unsetenv("NEPHORAN_RETRY_DELAY")
		os.Unsetenv("NEPHORAN_MAX_INPUT_LENGTH")
		os.Unsetenv("NEPHORAN_LLM_TIMEOUT")
	}()

	constants := LoadConstants()

	if constants.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", constants.MaxRetries)
	}

	if constants.RetryDelay != 45*time.Second {
		t.Errorf("Expected RetryDelay to be 45s, got %v", constants.RetryDelay)
	}

	if constants.MaxInputLength != 15000 {
		t.Errorf("Expected MaxInputLength to be 15000, got %d", constants.MaxInputLength)
	}

	if constants.LLMTimeout != 60*time.Second {
		t.Errorf("Expected LLMTimeout to be 60s, got %v", constants.LLMTimeout)
	}
}

func TestValidateConstants(t *testing.T) {
	// Test valid constants
	constants := LoadConstants()
	err := ValidateConstants(constants)
	if err != nil {
		t.Errorf("Expected no error for valid constants, got: %v", err)
	}

	// Test invalid retry count
	invalidConstants := *constants
	invalidConstants.MaxRetries = -1
	err = ValidateConstants(&invalidConstants)
	if err == nil {
		t.Error("Expected error for negative MaxRetries")
	}

	// Test invalid timeout
	invalidConstants = *constants
	invalidConstants.LLMTimeout = -1
	err = ValidateConstants(&invalidConstants)
	if err == nil {
		t.Error("Expected error for negative LLMTimeout")
	}

	// Test invalid failure rate
	invalidConstants = *constants
	invalidConstants.CircuitBreakerFailureRate = 1.5
	err = ValidateConstants(&invalidConstants)
	if err == nil {
		t.Error("Expected error for failure rate > 1.0")
	}

	// Test invalid port
	invalidConstants = *constants
	invalidConstants.MetricsPort = 70000
	err = ValidateConstants(&invalidConstants)
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestValidateCompleteConfiguration(t *testing.T) {
	constants := LoadConstants()
	err := ValidateCompleteConfiguration(constants)
	if err != nil {
		t.Errorf("Expected no error for complete configuration validation, got: %v", err)
	}
}

func TestEnvironmentVariableParsing(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		getter   func() interface{}
		expected interface{}
	}{
		{
			name:     "Integer parsing",
			envVar:   "TEST_INT",
			envValue: "42",
			getter:   func() interface{} { return getEnvInt("TEST_INT", 0) },
			expected: 42,
		},
		{
			name:     "Float parsing",
			envVar:   "TEST_FLOAT",
			envValue: "3.14",
			getter:   func() interface{} { return getEnvFloat("TEST_FLOAT", 0.0) },
			expected: 3.14,
		},
		{
			name:     "Duration parsing",
			envVar:   "TEST_DURATION",
			envValue: "5m",
			getter:   func() interface{} { return getEnvDuration("TEST_DURATION", 0) },
			expected: 5 * time.Minute,
		},
		{
			name:     "String array parsing",
			envVar:   "TEST_ARRAY",
			envValue: "item1,item2,item3",
			getter:   func() interface{} { return getEnvStringArray("TEST_ARRAY", nil) },
			expected: []string{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			result := tt.getter()
			
			// Compare based on type
			switch expected := tt.expected.(type) {
			case int:
				if result.(int) != expected {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			case float64:
				if result.(float64) != expected {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			case time.Duration:
				if result.(time.Duration) != expected {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			case []string:
				resultSlice := result.([]string)
				if len(resultSlice) != len(expected) {
					t.Errorf("Expected slice length %d, got %d", len(expected), len(resultSlice))
					return
				}
				for i, v := range expected {
					if resultSlice[i] != v {
						t.Errorf("Expected %v at index %d, got %v", v, i, resultSlice[i])
					}
				}
			}
		})
	}
}

func TestDefaultsVersusOverrides(t *testing.T) {
	// Test that defaults work when no environment variables are set
	constants := LoadConstants()
	defaultMaxRetries := constants.MaxRetries

	// Set an override
	os.Setenv("NEPHORAN_MAX_RETRIES", "10")
	defer os.Unsetenv("NEPHORAN_MAX_RETRIES")

	constants = LoadConstants()
	if constants.MaxRetries == defaultMaxRetries {
		t.Error("Expected environment variable to override default value")
	}

	if constants.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries to be 10, got %d", constants.MaxRetries)
	}
}

func TestConfigurationBoundaryValues(t *testing.T) {
	constants := LoadConstants()

	// Test minimum values
	constants.MaxRetries = 0
	err := ValidateConstants(constants)
	if err == nil {
		t.Error("Expected error for MaxRetries = 0")
	}

	// Test maximum values
	constants = LoadConstants()
	constants.MaxRetries = 1000
	err = ValidateConstants(constants)
	if err == nil {
		t.Error("Expected error for MaxRetries = 1000 (exceeds limit)")
	}

	// Test edge cases for ports
	constants = LoadConstants()
	constants.MetricsPort = 1
	err = ValidateConstants(constants)
	if err != nil {
		t.Errorf("Expected no error for MetricsPort = 1, got: %v", err)
	}

	constants.MetricsPort = 65535
	err = ValidateConstants(constants)
	if err != nil {
		t.Errorf("Expected no error for MetricsPort = 65535, got: %v", err)
	}

	constants.MetricsPort = 0
	err = ValidateConstants(constants)
	if err == nil {
		t.Error("Expected error for MetricsPort = 0")
	}

	constants.MetricsPort = 65536
	err = ValidateConstants(constants)
	if err == nil {
		t.Error("Expected error for MetricsPort = 65536")
	}
}

func TestPrintConfiguration(t *testing.T) {
	// This is mainly to ensure the function doesn't panic
	constants := LoadConstants()
	
	// Capture output (in a real test, you might want to capture stdout)
	// For now, just ensure it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintConfiguration panicked: %v", r)
		}
	}()
	
	PrintConfiguration(constants)
}
package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

// Helper function to set environment variables and clean up
func setEnvVars(t *testing.T, vars map[string]string) func() {
	t.Helper()
	
	// Store original values for cleanup
	original := make(map[string]string)
	for key := range vars {
		if val, exists := os.LookupEnv(key); exists {
			original[key] = val
		}
	}
	
	// Set new values
	for key, val := range vars {
		if err := os.Setenv(key, val); err != nil {
			t.Fatalf("Failed to set environment variable %s: %v", key, err)
		}
	}
	
	// Return cleanup function
	return func() {
		for key := range vars {
			if origVal, exists := original[key]; exists {
				os.Setenv(key, origVal)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  string
		expected    string
	}{
		{
			name:        "existing env var",
			envVars:     map[string]string{"TEST_KEY": "test_value"},
			key:         "TEST_KEY",
			defaultVal:  "default",
			expected:    "test_value",
		},
		{
			name:        "missing env var returns default",
			envVars:     map[string]string{},
			key:         "MISSING_KEY",
			defaultVal:  "default",
			expected:    "default",
		},
		{
			name:        "empty env var returns default",
			envVars:     map[string]string{"EMPTY_KEY": ""},
			key:         "EMPTY_KEY",
			defaultVal:  "default",
			expected:    "default",
		},
		{
			name:        "whitespace env var returns value",
			envVars:     map[string]string{"WHITESPACE_KEY": "  value  "},
			key:         "WHITESPACE_KEY",
			defaultVal:  "default",
			expected:    "  value  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetEnvOrDefault(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetEnvOrDefault(%q, %q) = %q, expected %q", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  bool
		expected    bool
	}{
		{
			name:        "true value",
			envVars:     map[string]string{"BOOL_KEY": "true"},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    true,
		},
		{
			name:        "1 value",
			envVars:     map[string]string{"BOOL_KEY": "1"},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    true,
		},
		{
			name:        "yes value",
			envVars:     map[string]string{"BOOL_KEY": "yes"},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    true,
		},
		{
			name:        "enabled value",
			envVars:     map[string]string{"BOOL_KEY": "enabled"},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    true,
		},
		{
			name:        "false value",
			envVars:     map[string]string{"BOOL_KEY": "false"},
			key:         "BOOL_KEY",
			defaultVal:  true,
			expected:    false,
		},
		{
			name:        "0 value",
			envVars:     map[string]string{"BOOL_KEY": "0"},
			key:         "BOOL_KEY",
			defaultVal:  true,
			expected:    false,
		},
		{
			name:        "disabled value",
			envVars:     map[string]string{"BOOL_KEY": "disabled"},
			key:         "BOOL_KEY",
			defaultVal:  true,
			expected:    false,
		},
		{
			name:        "case insensitive true",
			envVars:     map[string]string{"BOOL_KEY": "TRUE"},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    true,
		},
		{
			name:        "invalid value returns default",
			envVars:     map[string]string{"BOOL_KEY": "invalid"},
			key:         "BOOL_KEY",
			defaultVal:  true,
			expected:    true,
		},
		{
			name:        "missing key returns default",
			envVars:     map[string]string{},
			key:         "MISSING_BOOL",
			defaultVal:  true,
			expected:    true,
		},
		{
			name:        "empty value returns default",
			envVars:     map[string]string{"BOOL_KEY": ""},
			key:         "BOOL_KEY",
			defaultVal:  false,
			expected:    false,
		},
		{
			name:        "whitespace value returns default",
			envVars:     map[string]string{"BOOL_KEY": "  "},
			key:         "BOOL_KEY",
			defaultVal:  true,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetBoolEnv(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetBoolEnv(%q, %v) = %v, expected %v", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetDurationEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  time.Duration
		expected    time.Duration
	}{
		{
			name:        "valid duration",
			envVars:     map[string]string{"DURATION_KEY": "30s"},
			key:         "DURATION_KEY",
			defaultVal:  10 * time.Second,
			expected:    30 * time.Second,
		},
		{
			name:        "complex duration",
			envVars:     map[string]string{"DURATION_KEY": "1h30m45s"},
			key:         "DURATION_KEY",
			defaultVal:  1 * time.Minute,
			expected:    1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:        "invalid duration returns default",
			envVars:     map[string]string{"DURATION_KEY": "invalid"},
			key:         "DURATION_KEY",
			defaultVal:  5 * time.Minute,
			expected:    5 * time.Minute,
		},
		{
			name:        "missing key returns default",
			envVars:     map[string]string{},
			key:         "MISSING_DURATION",
			defaultVal:  2 * time.Hour,
			expected:    2 * time.Hour,
		},
		{
			name:        "empty value returns default",
			envVars:     map[string]string{"DURATION_KEY": ""},
			key:         "DURATION_KEY",
			defaultVal:  15 * time.Second,
			expected:    15 * time.Second,
		},
		{
			name:        "microseconds",
			envVars:     map[string]string{"DURATION_KEY": "100us"},
			key:         "DURATION_KEY",
			defaultVal:  1 * time.Millisecond,
			expected:    100 * time.Microsecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetDurationEnv(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetDurationEnv(%q, %v) = %v, expected %v", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetStringSliceEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  []string
		expected    []string
	}{
		{
			name:        "single value",
			envVars:     map[string]string{"SLICE_KEY": "value1"},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"value1"},
		},
		{
			name:        "multiple values",
			envVars:     map[string]string{"SLICE_KEY": "value1,value2,value3"},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"value1", "value2", "value3"},
		},
		{
			name:        "values with spaces",
			envVars:     map[string]string{"SLICE_KEY": " value1 , value2 , value3 "},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"value1", "value2", "value3"},
		},
		{
			name:        "empty items ignored",
			envVars:     map[string]string{"SLICE_KEY": "value1,,value3,"},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"value1", "value3"},
		},
		{
			name:        "only empty items returns default",
			envVars:     map[string]string{"SLICE_KEY": ",,, ,"},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"default"},
		},
		{
			name:        "missing key returns default",
			envVars:     map[string]string{},
			key:         "MISSING_SLICE",
			defaultVal:  []string{"default1", "default2"},
			expected:    []string{"default1", "default2"},
		},
		{
			name:        "empty value returns default",
			envVars:     map[string]string{"SLICE_KEY": ""},
			key:         "SLICE_KEY",
			defaultVal:  []string{"default"},
			expected:    []string{"default"},
		},
		{
			name:        "nil default",
			envVars:     map[string]string{"SLICE_KEY": "value1,value2"},
			key:         "SLICE_KEY",
			defaultVal:  nil,
			expected:    []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetStringSliceEnv(tt.key, tt.defaultVal)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetStringSliceEnv(%q, %v) = %v, expected %v", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  int
		expected    int
	}{
		{
			name:        "valid positive integer",
			envVars:     map[string]string{"INT_KEY": "123"},
			key:         "INT_KEY",
			defaultVal:  10,
			expected:    123,
		},
		{
			name:        "valid negative integer",
			envVars:     map[string]string{"INT_KEY": "-456"},
			key:         "INT_KEY",
			defaultVal:  10,
			expected:    -456,
		},
		{
			name:        "zero value",
			envVars:     map[string]string{"INT_KEY": "0"},
			key:         "INT_KEY",
			defaultVal:  10,
			expected:    0,
		},
		{
			name:        "invalid integer returns default",
			envVars:     map[string]string{"INT_KEY": "invalid"},
			key:         "INT_KEY",
			defaultVal:  42,
			expected:    42,
		},
		{
			name:        "float value returns default",
			envVars:     map[string]string{"INT_KEY": "12.34"},
			key:         "INT_KEY",
			defaultVal:  42,
			expected:    42,
		},
		{
			name:        "missing key returns default",
			envVars:     map[string]string{},
			key:         "MISSING_INT",
			defaultVal:  100,
			expected:    100,
		},
		{
			name:        "empty value returns default",
			envVars:     map[string]string{"INT_KEY": ""},
			key:         "INT_KEY",
			defaultVal:  50,
			expected:    50,
		},
		{
			name:        "whitespace value returns default",
			envVars:     map[string]string{"INT_KEY": "  "},
			key:         "INT_KEY",
			defaultVal:  75,
			expected:    75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetIntEnv(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetIntEnv(%q, %d) = %d, expected %d", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetInt64Env(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  int64
		expected    int64
	}{
		{
			name:        "valid int64",
			envVars:     map[string]string{"INT64_KEY": "9223372036854775807"},
			key:         "INT64_KEY",
			defaultVal:  100,
			expected:    9223372036854775807,
		},
		{
			name:        "negative int64",
			envVars:     map[string]string{"INT64_KEY": "-9223372036854775808"},
			key:         "INT64_KEY",
			defaultVal:  100,
			expected:    -9223372036854775808,
		},
		{
			name:        "invalid value returns default",
			envVars:     map[string]string{"INT64_KEY": "invalid"},
			key:         "INT64_KEY",
			defaultVal:  42,
			expected:    42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetInt64Env(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetInt64Env(%q, %d) = %d, expected %d", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetFloatEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  float64
		expected    float64
	}{
		{
			name:        "valid float",
			envVars:     map[string]string{"FLOAT_KEY": "3.14159"},
			key:         "FLOAT_KEY",
			defaultVal:  1.0,
			expected:    3.14159,
		},
		{
			name:        "integer as float",
			envVars:     map[string]string{"FLOAT_KEY": "42"},
			key:         "FLOAT_KEY",
			defaultVal:  1.0,
			expected:    42.0,
		},
		{
			name:        "scientific notation",
			envVars:     map[string]string{"FLOAT_KEY": "1.23e-4"},
			key:         "FLOAT_KEY",
			defaultVal:  1.0,
			expected:    1.23e-4,
		},
		{
			name:        "invalid value returns default",
			envVars:     map[string]string{"FLOAT_KEY": "invalid"},
			key:         "FLOAT_KEY",
			defaultVal:  2.5,
			expected:    2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result := GetFloatEnv(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("GetFloatEnv(%q, %f) = %f, expected %f", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestGetEnvWithValidation(t *testing.T) {
	validatePort := func(value string) error {
		return ValidatePort(value)
	}

	tests := []struct {
		name        string
		envVars     map[string]string
		key         string
		defaultVal  string
		validator   func(string) error
		expected    string
		expectError bool
	}{
		{
			name:        "valid port",
			envVars:     map[string]string{"PORT": "8080"},
			key:         "PORT",
			defaultVal:  "3000",
			validator:   validatePort,
			expected:    "8080",
			expectError: false,
		},
		{
			name:        "invalid port in env",
			envVars:     map[string]string{"PORT": "70000"},
			key:         "PORT",
			defaultVal:  "3000",
			validator:   validatePort,
			expected:    "3000",
			expectError: true,
		},
		{
			name:        "missing env with valid default",
			envVars:     map[string]string{},
			key:         "PORT",
			defaultVal:  "3000",
			validator:   validatePort,
			expected:    "3000",
			expectError: false,
		},
		{
			name:        "missing env with invalid default",
			envVars:     map[string]string{},
			key:         "PORT",
			defaultVal:  "invalid",
			validator:   validatePort,
			expected:    "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setEnvVars(t, tt.envVars)
			defer cleanup()

			result, err := GetEnvWithValidation(tt.key, tt.defaultVal, tt.validator)

			if tt.expectError && err == nil {
				t.Errorf("GetEnvWithValidation(%q, %q, validator) expected error but got none", tt.key, tt.defaultVal)
			}
			if !tt.expectError && err != nil {
				t.Errorf("GetEnvWithValidation(%q, %q, validator) unexpected error: %v", tt.key, tt.defaultVal, err)
			}
			if result != tt.expected {
				t.Errorf("GetEnvWithValidation(%q, %q, validator) = %q, expected %q", tt.key, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestValidators(t *testing.T) {
	t.Run("ValidateNonEmpty", func(t *testing.T) {
		tests := []struct {
			value       string
			expectError bool
		}{
			{"valid", false},
			{"", true},
			{"  ", true},
			{"\t\n", true},
			{"  valid  ", false},
		}

		for _, tt := range tests {
			err := ValidateNonEmpty(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ValidateNonEmpty(%q) expected error but got none", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateNonEmpty(%q) unexpected error: %v", tt.value, err)
			}
		}
	})

	t.Run("ValidatePort", func(t *testing.T) {
		tests := []struct {
			value       string
			expectError bool
		}{
			{"8080", false},
			{"1", false},
			{"65535", false},
			{"0", true},
			{"65536", true},
			{"invalid", true},
			{"-1", true},
		}

		for _, tt := range tests {
			err := ValidatePort(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ValidatePort(%q) expected error but got none", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidatePort(%q) unexpected error: %v", tt.value, err)
			}
		}
	})

	t.Run("ValidatePositiveInt", func(t *testing.T) {
		tests := []struct {
			value       int
			expectError bool
		}{
			{1, false},
			{100, false},
			{0, true},
			{-1, true},
		}

		for _, tt := range tests {
			err := ValidatePositiveInt(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ValidatePositiveInt(%d) expected error but got none", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidatePositiveInt(%d) unexpected error: %v", tt.value, err)
			}
		}
	})

	t.Run("ValidateURL", func(t *testing.T) {
		tests := []struct {
			value       string
			expectError bool
		}{
			{"http://example.com", false},
			{"https://example.com", false},
			{"ftp://example.com", true},
			{"", true},
			{"example.com", true},
		}

		for _, tt := range tests {
			err := ValidateURL(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ValidateURL(%q) expected error but got none", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateURL(%q) unexpected error: %v", tt.value, err)
			}
		}
	})

	t.Run("ValidateLogLevel", func(t *testing.T) {
		tests := []struct {
			value       string
			expectError bool
		}{
			{"debug", false},
			{"info", false},
			{"DEBUG", false}, // Case insensitive
			{"invalid", true},
			{"", true},
		}

		for _, tt := range tests {
			err := ValidateLogLevel(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ValidateLogLevel(%q) expected error but got none", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ValidateLogLevel(%q) unexpected error: %v", tt.value, err)
			}
		}
	})
}

func TestValidateOneOf(t *testing.T) {
	validator := ValidateOneOf([]string{"redis", "memory", "disk"})

	tests := []struct {
		value       string
		expectError bool
	}{
		{"redis", false},
		{"memory", false},
		{"disk", false},
		{"Redis", true}, // Case sensitive
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		err := validator(tt.value)
		if tt.expectError && err == nil {
			t.Errorf("ValidateOneOf(%q) expected error but got none", tt.value)
		}
		if !tt.expectError && err != nil {
			t.Errorf("ValidateOneOf(%q) unexpected error: %v", tt.value, err)
		}
	}
}

func TestValidateOneOfIgnoreCase(t *testing.T) {
	validator := ValidateOneOfIgnoreCase([]string{"development", "staging", "production"})

	tests := []struct {
		value       string
		expectError bool
	}{
		{"development", false},
		{"DEVELOPMENT", false},
		{"Development", false},
		{"staging", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		err := validator(tt.value)
		if tt.expectError && err == nil {
			t.Errorf("ValidateOneOfIgnoreCase(%q) expected error but got none", tt.value)
		}
		if !tt.expectError && err != nil {
			t.Errorf("ValidateOneOfIgnoreCase(%q) unexpected error: %v", tt.value, err)
		}
	}
}

func TestMustGetEnv(t *testing.T) {
	t.Run("existing env var", func(t *testing.T) {
		cleanup := setEnvVars(t, map[string]string{"MUST_EXIST": "value"})
		defer cleanup()

		result := MustGetEnv("MUST_EXIST")
		if result != "value" {
			t.Errorf("MustGetEnv(MUST_EXIST) = %q, expected %q", result, "value")
		}
	})

	t.Run("missing env var panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetEnv should have panicked for missing env var")
			}
		}()
		MustGetEnv("MISSING_MUST_EXIST")
	})

	t.Run("empty env var panics", func(t *testing.T) {
		cleanup := setEnvVars(t, map[string]string{"EMPTY_MUST_EXIST": ""})
		defer cleanup()

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetEnv should have panicked for empty env var")
			}
		}()
		MustGetEnv("EMPTY_MUST_EXIST")
	})
}

func TestIsSet(t *testing.T) {
	cleanup := setEnvVars(t, map[string]string{
		"SET_VAR":   "value",
		"EMPTY_VAR": "",
	})
	defer cleanup()

	tests := []struct {
		key      string
		expected bool
	}{
		{"SET_VAR", true},
		{"EMPTY_VAR", true},  // Empty but set
		{"UNSET_VAR", false},
	}

	for _, tt := range tests {
		result := IsSet(tt.key)
		if result != tt.expected {
			t.Errorf("IsSet(%q) = %v, expected %v", tt.key, result, tt.expected)
		}
	}
}

func TestGetEnvKeys(t *testing.T) {
	// Set some test environment variables
	cleanup := setEnvVars(t, map[string]string{
		"TEST_ENV_KEY_1": "value1",
		"TEST_ENV_KEY_2": "value2",
	})
	defer cleanup()

	keys := GetEnvKeys()

	// Check that our test keys are present
	foundKey1 := false
	foundKey2 := false
	for _, key := range keys {
		if key == "TEST_ENV_KEY_1" {
			foundKey1 = true
		}
		if key == "TEST_ENV_KEY_2" {
			foundKey2 = true
		}
	}

	if !foundKey1 {
		t.Error("Expected TEST_ENV_KEY_1 to be in environment keys")
	}
	if !foundKey2 {
		t.Error("Expected TEST_ENV_KEY_2 to be in environment keys")
	}
}

func TestValidateIntRange(t *testing.T) {
	validator := ValidateIntRange(10, 100)

	tests := []struct {
		value       int
		expectError bool
	}{
		{10, false},   // Min boundary
		{50, false},   // Mid range
		{100, false},  // Max boundary
		{9, true},     // Below min
		{101, true},   // Above max
	}

	for _, tt := range tests {
		err := validator(tt.value)
		if tt.expectError && err == nil {
			t.Errorf("ValidateIntRange(%d) expected error but got none", tt.value)
		}
		if !tt.expectError && err != nil {
			t.Errorf("ValidateIntRange(%d) unexpected error: %v", tt.value, err)
		}
	}
}

func TestValidateDurationRange(t *testing.T) {
	validator := ValidateDurationRange(1*time.Second, 5*time.Minute)

	tests := []struct {
		value       time.Duration
		expectError bool
	}{
		{1 * time.Second, false},     // Min boundary
		{30 * time.Second, false},    // Mid range
		{5 * time.Minute, false},     // Max boundary
		{500 * time.Millisecond, true}, // Below min
		{10 * time.Minute, true},     // Above max
	}

	for _, tt := range tests {
		err := validator(tt.value)
		if tt.expectError && err == nil {
			t.Errorf("ValidateDurationRange(%v) expected error but got none", tt.value)
		}
		if !tt.expectError && err != nil {
			t.Errorf("ValidateDurationRange(%v) unexpected error: %v", tt.value, err)
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkGetEnvOrDefault(b *testing.B) {
	os.Setenv("BENCH_KEY", "benchmark_value")
	defer os.Unsetenv("BENCH_KEY")
	
	for i := 0; i < b.N; i++ {
		GetEnvOrDefault("BENCH_KEY", "default")
	}
}

func BenchmarkGetBoolEnv(b *testing.B) {
	os.Setenv("BENCH_BOOL", "true")
	defer os.Unsetenv("BENCH_BOOL")
	
	for i := 0; i < b.N; i++ {
		GetBoolEnv("BENCH_BOOL", false)
	}
}

func BenchmarkGetStringSliceEnv(b *testing.B) {
	os.Setenv("BENCH_SLICE", "item1,item2,item3,item4,item5")
	defer os.Unsetenv("BENCH_SLICE")
	
	for i := 0; i < b.N; i++ {
		GetStringSliceEnv("BENCH_SLICE", []string{})
	}
}
package security

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

func TestValidateKMPData(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name       string
		data       rules.KPMData
		wantError  bool
		errorField string
	}{
		{
			name: "valid KMP data",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError: false,
		},
		{
			name: "zero timestamp",
			data: rules.KPMData{
				Timestamp:       time.Time{},
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "timestamp",
		},
		{
			name: "future timestamp",
			data: rules.KPMData{
				Timestamp:       time.Now().Add(10 * time.Minute),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "timestamp",
		},
		{
			name: "old timestamp",
			data: rules.KPMData{
				Timestamp:       time.Now().Add(-25 * time.Hour),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "timestamp",
		},
		{
			name: "empty node ID",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "node_id",
		},
		{
			name: "malicious node ID",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test'; DROP TABLE metrics; --",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "node_id",
		},
		{
			name: "negative PRB utilization",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  -0.1,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "prb_utilization",
		},
		{
			name: "excessive PRB utilization",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  1.5,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "prb_utilization",
		},
		{
			name: "negative latency",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      -10.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "p95_latency",
		},
		{
			name: "excessive latency",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      15000.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "p95_latency",
		},
		{
			name: "negative active UEs",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       -5,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "active_ues",
		},
		{
			name: "excessive active UEs",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       15000,
				CurrentReplicas: 3,
			},
			wantError:  true,
			errorField: "active_ues",
		},
		{
			name: "negative replicas",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: -1,
			},
			wantError:  true,
			errorField: "current_replicas",
		},
		{
			name: "excessive replicas",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 200,
			},
			wantError:  true,
			errorField: "current_replicas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateKMPData(tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateKMPData() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil {
				validationErr, ok := err.(ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}
				if validationErr.Field != tt.errorField {
					t.Errorf("Expected error field %s, got %s", tt.errorField, validationErr.Field)
				}
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{
			name:      "valid HTTP URL",
			url:       "http://localhost:9090/metrics",
			wantError: false,
		},
		{
			name:      "valid HTTPS URL",
			url:       "https://api.example.com/v1/metrics",
			wantError: false,
		},
		{
			name:      "empty URL",
			url:       "",
			wantError: true,
		},
		{
			name:      "invalid scheme",
			url:       "ftp://example.com/file",
			wantError: true,
		},
		{
			name:      "malformed URL",
			url:       "not-a-url",
			wantError: true,
		},
		{
			name:      "URL with dangerous query",
			url:       "http://example.com/api?query='; DROP TABLE users; --",
			wantError: true,
		},
		{
			name:      "overly long URL",
			url:       "http://example.com/" + string(make([]byte, 3000)),
			wantError: true,
		},
		{
			name:      "URL without hostname",
			url:       "http:///path",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURL(tt.url, "test context")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateURL() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid relative path",
			path:      "config/settings.json",
			wantError: false,
		},
		{
			name:      "valid absolute path",
			path:      "/tmp/planner-data.json",
			wantError: false,
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "directory traversal attack",
			path:      "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "complex directory traversal",
			path:      "config/../../../etc/shadow",
			wantError: true,
		},
		{
			name:      "sensitive Unix system directory",
			path:      "/etc/passwd",
			wantError: runtime.GOOS != "windows", // Only expect error on Unix systems
		},
		{
			name:      "sensitive Windows system directory",
			path:      "C:\\Windows\\System32\\config\\SAM",
			wantError: true, // Validator blocks Windows-style sensitive paths on all platforms
		},
		{
			name:      "Windows system directory",
			path:      "C:\\Windows\\System32\\config",
			wantError: true,
		},
		{
			name:      "null byte injection",
			path:      "config/file\x00.json",
			wantError: true,
		},
		{
			name:      "invalid extension",
			path:      "config/script.sh",
			wantError: true,
		},
		{
			name:      "overly long path",
			path:      "/" + string(make([]byte, 5000)),
			wantError: true,
		},
		{
			name:      "dangerous characters",
			path:      "config/<script>alert('xss')</script>.json",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tt.path, "test context")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFilePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateEnvironmentVariable(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		envName   string
		value     string
		wantError bool
	}{
		{
			name:      "valid URL environment variable",
			envName:   "PLANNER_METRICS_URL",
			value:     "http://localhost:9090/metrics",
			wantError: false,
		},
		{
			name:      "valid directory environment variable",
			envName:   "PLANNER_OUTPUT_DIR",
			value:     "./handoff",
			wantError: false,
		},
		{
			name:      "invalid URL in env var",
			envName:   "PLANNER_METRICS_URL",
			value:     "not-a-url",
			wantError: true,
		},
		{
			name:      "directory traversal in env var",
			envName:   "PLANNER_CONFIG_DIR",
			value:     "../../../etc",
			wantError: true,
		},
		{
			name:      "null byte in env var",
			envName:   "PLANNER_SETTING",
			value:     "value\x00with\x00nulls",
			wantError: true,
		},
		{
			name:      "newline injection in env var",
			envName:   "PLANNER_SETTING",
			value:     "value\nwith\nnewlines",
			wantError: true,
		},
		{
			name:      "empty env var name",
			envName:   "",
			value:     "some-value",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEnvironmentVariable(tt.envName, tt.value, "test context")
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEnvironmentVariable() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateNodeID(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		nodeID    string
		wantError bool
	}{
		{
			name:      "valid node ID",
			nodeID:    "test-node-001",
			wantError: false,
		},
		{
			name:      "valid node ID with underscores",
			nodeID:    "test_node_001",
			wantError: false,
		},
		{
			name:      "alphanumeric node ID",
			nodeID:    "node123ABC",
			wantError: false,
		},
		{
			name:      "empty node ID",
			nodeID:    "",
			wantError: true,
		},
		{
			name:      "node ID with spaces",
			nodeID:    "test node 001",
			wantError: true,
		},
		{
			name:      "node ID with special characters",
			nodeID:    "test@node#001",
			wantError: true,
		},
		{
			name:      "SQL injection attempt",
			nodeID:    "'; DROP TABLE nodes; --",
			wantError: true,
		},
		{
			name:      "overly long node ID",
			nodeID:    string(make([]byte, 300)),
			wantError: true,
		},
		{
			name:      "node ID with quotes",
			nodeID:    "test\"node'001",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateNodeID(tt.nodeID)
			if (err != nil) != tt.wantError {
				t.Errorf("validateNodeID() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSanitizeForLogging(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "normal log message",
			expected: "normal log message",
		},
		{
			name:     "string with newlines",
			input:    "message\nwith\nnewlines",
			expected: "message\\nwith\\nnewlines",
		},
		{
			name:     "string with carriage returns",
			input:    "message\rwith\rcarriage\rreturns",
			expected: "message\\rwith\\rcarriage\\rreturns",
		},
		{
			name:     "string with tabs",
			input:    "message\twith\ttabs",
			expected: "message\\twith\\ttabs",
		},
		{
			name:  "overly long string",
			input: strings.Repeat("a", 300),
			// 300 'a' chars → truncated to 253 + "..."
			expected: strings.Repeat("a", 253) + "...",
		},
		{
			name:     "mixed dangerous characters",
			input:    "message\nwith\rmixed\tdangerous\ncharacters",
			expected: "message\\nwith\\rmixed\\tdangerous\\ncharacters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeForLogging(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForLogging() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "test_field",
		Value:   "test_value",
		Reason:  "test reason",
		Context: "test context",
	}

	expected := "validation failed for test_field (value: test_value): test reason [context: test context]"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	// Verify default values are secure
	if config.MaxPRBUtilization != 1.0 {
		t.Errorf("Expected MaxPRBUtilization = 1.0, got %f", config.MaxPRBUtilization)
	}
	if config.MaxLatency != 10000.0 {
		t.Errorf("Expected MaxLatency = 10000.0, got %f", config.MaxLatency)
	}
	if config.MaxReplicas != 100 {
		t.Errorf("Expected MaxReplicas = 100, got %d", config.MaxReplicas)
	}
	if len(config.AllowedSchemes) != 2 {
		t.Errorf("Expected 2 allowed schemes, got %d", len(config.AllowedSchemes))
	}
	if len(config.AllowedExtensions) != 3 {
		t.Errorf("Expected 3 allowed extensions, got %d", len(config.AllowedExtensions))
	}
}

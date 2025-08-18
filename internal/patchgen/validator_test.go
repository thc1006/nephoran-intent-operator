package patchgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	logger := testr.New(t)
	
	validator, err := NewValidator(logger)
	
	assert.NoError(t, err)
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.schema)
	assert.Equal(t, logger.WithName("validator"), validator.logger)
}

func TestValidateIntent(t *testing.T) {
	logger := testr.New(t)
	validator, err := NewValidator(logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		intentJSON  string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, intent *Intent)
	}{
		{
			name: "valid minimal intent",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
			validate: func(t *testing.T, intent *Intent) {
				assert.Equal(t, "scaling", intent.IntentType)
				assert.Equal(t, "web-app", intent.Target)
				assert.Equal(t, "default", intent.Namespace)
				assert.Equal(t, 3, intent.Replicas)
				assert.Empty(t, intent.Reason)
				assert.Empty(t, intent.Source)
				assert.Empty(t, intent.CorrelationID)
			},
		},
		{
			name: "valid complete intent",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "api-server",
				"namespace": "production",
				"replicas": 10,
				"reason": "High CPU usage detected",
				"source": "prometheus-autoscaler",
				"correlation_id": "scale-event-12345"
			}`,
			expectError: false,
			validate: func(t *testing.T, intent *Intent) {
				assert.Equal(t, "scaling", intent.IntentType)
				assert.Equal(t, "api-server", intent.Target)
				assert.Equal(t, "production", intent.Namespace)
				assert.Equal(t, 10, intent.Replicas)
				assert.Equal(t, "High CPU usage detected", intent.Reason)
				assert.Equal(t, "prometheus-autoscaler", intent.Source)
				assert.Equal(t, "scale-event-12345", intent.CorrelationID)
			},
		},
		{
			name: "boundary condition - zero replicas",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "worker",
				"namespace": "test",
				"replicas": 0
			}`,
			expectError: false,
			validate: func(t *testing.T, intent *Intent) {
				assert.Equal(t, 0, intent.Replicas)
			},
		},
		{
			name: "boundary condition - max replicas",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "batch-processor",
				"namespace": "processing",
				"replicas": 100
			}`,
			expectError: false,
			validate: func(t *testing.T, intent *Intent) {
				assert.Equal(t, 100, intent.Replicas)
			},
		},
		{
			name: "invalid JSON",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app"
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name: "missing required field - intent_type",
			intentJSON: `{
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "missing required field - target",
			intentJSON: `{
				"intent_type": "scaling",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "missing required field - namespace",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "missing required field - replicas",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default"
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "invalid intent_type",
			intentJSON: `{
				"intent_type": "invalid",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "empty target",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "empty namespace",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "negative replicas",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": -1
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "replicas exceeding maximum",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 101
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "replicas as string",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": "3"
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "additional properties not allowed",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3,
				"extra_field": "not allowed"
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "null values for required fields",
			intentJSON: `{
				"intent_type": null,
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "wrong data types",
			intentJSON: `{
				"intent_type": "scaling",
				"target": 123,
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := validator.ValidateIntent([]byte(tt.intentJSON))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, intent)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, intent)
				if tt.validate != nil {
					tt.validate(t, intent)
				}
			}
		})
	}
}

func TestValidateIntentFile(t *testing.T) {
	logger := testr.New(t)
	validator, err := NewValidator(logger)
	require.NoError(t, err)

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		fileName    string
		fileContent string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid intent file",
			fileName: "valid.json",
			fileContent: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 5
			}`,
			expectError: false,
		},
		{
			name:        "file does not exist",
			fileName:    "nonexistent.json",
			expectError: true,
			errorMsg:    "failed to read intent file",
		},
		{
			name:     "invalid JSON file",
			fileName: "invalid.json",
			fileContent: `{
				"intent_type": "scaling",
				"target": "web-app"
				"namespace": "default",
				"replicas": 5
			}`,
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name:     "schema validation fails",
			fileName: "schema-invalid.json",
			fileContent: `{
				"intent_type": "invalid",
				"target": "web-app",
				"namespace": "default",
				"replicas": 5
			}`,
			expectError: true,
			errorMsg:    "schema validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.fileName)
			
			// Create file if content is provided
			if tt.fileContent != "" {
				require.NoError(t, os.WriteFile(filePath, []byte(tt.fileContent), 0644))
			}

			intent, err := validator.ValidateIntentFile(filePath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, intent)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, intent)
			}
		})
	}
}

func TestValidateIntentMap(t *testing.T) {
	logger := testr.New(t)
	validator, err := NewValidator(logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		intentMap   map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid intent map",
			intentMap: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "web-app",
				"namespace":   "default",
				"replicas":    3,
			},
			expectError: false,
		},
		{
			name: "valid intent map with optional fields",
			intentMap: map[string]interface{}{
				"intent_type":    "scaling",
				"target":         "api-server",
				"namespace":      "production",
				"replicas":       10,
				"reason":         "Load increase",
				"source":         "autoscaler",
				"correlation_id": "test-123",
			},
			expectError: false,
		},
		{
			name: "missing required field",
			intentMap: map[string]interface{}{
				"intent_type": "scaling",
				"namespace":   "default",
				"replicas":    3,
			},
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "invalid data type",
			intentMap: map[string]interface{}{
				"intent_type": "scaling",
				"target":      123,
				"namespace":   "default",
				"replicas":    3,
			},
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "invalid enum value",
			intentMap: map[string]interface{}{
				"intent_type": "invalid",
				"target":      "web-app",
				"namespace":   "default",
				"replicas":    3,
			},
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "replicas out of range",
			intentMap: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "web-app",
				"namespace":   "default",
				"replicas":    -1,
			},
			expectError: true,
			errorMsg:    "schema validation failed",
		},
		{
			name: "additional properties",
			intentMap: map[string]interface{}{
				"intent_type":  "scaling",
				"target":       "web-app",
				"namespace":    "default",
				"replicas":     3,
				"extra_field":  "not allowed",
			},
			expectError: true,
			errorMsg:    "schema validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIntentMap(tt.intentMap)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONSchema2020_12Compliance(t *testing.T) {
	logger := testr.New(t)
	_, err := NewValidator(logger)
	require.NoError(t, err)

	// Test that the schema is properly parsed as JSON Schema 2020-12
	var schemaDoc interface{}
	err = json.Unmarshal([]byte(IntentSchema), &schemaDoc)
	require.NoError(t, err)

	schemaMap, ok := schemaDoc.(map[string]interface{})
	require.True(t, ok)

	// Verify it's using JSON Schema 2020-12
	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", schemaMap["$schema"])
	assert.Equal(t, "https://nephoran.io/schemas/intent.json", schemaMap["$id"])
	assert.Equal(t, "Intent Schema", schemaMap["title"])

	// Verify required fields
	required, ok := schemaMap["required"].([]interface{})
	require.True(t, ok)
	expectedRequired := []string{"intent_type", "target", "namespace", "replicas"}
	actualRequired := make([]string, len(required))
	for i, r := range required {
		actualRequired[i] = r.(string)
	}
	assert.ElementsMatch(t, expectedRequired, actualRequired)
}

func TestEdgeCaseValidations(t *testing.T) {
	logger := testr.New(t)
	validator, err := NewValidator(logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		intentJSON  string
		expectError bool
		description string
	}{
		{
			name: "very long target name",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "` + generateLongString(1000) + `",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
			description: "Should handle very long target names",
		},
		{
			name: "unicode characters in target",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "测试-应用",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
			description: "Should handle unicode characters",
		},
		{
			name: "special characters in namespace",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "ns-with-hyphens-123",
				"replicas": 3
			}`,
			expectError: false,
			description: "Should handle special characters in namespace",
		},
		{
			name: "empty optional fields",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3,
				"reason": "",
				"source": "",
				"correlation_id": ""
			}`,
			expectError: false,
			description: "Should handle empty optional fields",
		},
		{
			name: "float replicas (should fail)",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3.5
			}`,
			expectError: true,
			description: "Should reject float values for replicas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			
			intent, err := validator.ValidateIntent([]byte(tt.intentJSON))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, intent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, intent)
			}
		})
	}
}

func TestSchemaCompilationErrors(t *testing.T) {
	logger := testr.New(t)

	// Test that NewValidator succeeds with valid schema
	validator, err := NewValidator(logger)
	assert.NoError(t, err)
	assert.NotNil(t, validator)
}

func TestValidatorConcurrency(t *testing.T) {
	logger := testr.New(t)
	validator, err := NewValidator(logger)
	require.NoError(t, err)

	// Test concurrent validation
	validIntent := `{
		"intent_type": "scaling",
		"target": "web-app",
		"namespace": "default",
		"replicas": 3
	}`

	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := validator.ValidateIntent([]byte(validIntent))
			results <- err
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// Helper function to generate long strings for testing
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a' + byte(i%26)
	}
	return string(result)
}
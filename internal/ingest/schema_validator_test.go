package ingest

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test schema for validation tests
var testSchema = map[string]interface{}{
		"apiVersion": json.RawMessage(`{}`),
		},
		"kind": json.RawMessage(`{}`),
		},
		"metadata": map[string]interface{}{
				"name": json.RawMessage(`{}`),
				"namespace": json.RawMessage(`{}`),
			},
			"required": []string{"name"},
		},
		"spec": map[string]interface{}{
				"intentType": json.RawMessage(`{}`),
				},
				"target": json.RawMessage(`{}`),
				"replicas": json.RawMessage(`{}`),
				"resources": map[string]interface{}{
						"cpu": json.RawMessage(`{}`),
						"memory": json.RawMessage(`{}`),
					},
				},
			},
			"required": []string{"intentType", "target"},
		},
	},
	"required": []string{"apiVersion", "kind", "metadata", "spec"},
}

// TestingT is an interface that covers the methods we need from *testing.T, *testing.B, and *testing.F
type TestingT interface {
	Helper()
	TempDir() string
	Fatal(...interface{})
}

func createTestSchemaFile(t TestingT) string {
	t.Helper()

	tempDir := t.TempDir()
	schemaFile := filepath.Join(tempDir, "test-schema.json")

	schemaData, err := json.Marshal(testSchema)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(schemaFile, schemaData, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	return schemaFile
}

func TestNewIntentSchemaValidator(t *testing.T) {
	t.Run("creates validator with valid schema file", func(t *testing.T) {
		schemaFile := createTestSchemaFile(t)

		validator, err := NewIntentSchemaValidator(schemaFile)
		require.NoError(t, err)
		assert.NotNil(t, validator)
		assert.Equal(t, schemaFile, validator.schemaPath)
		assert.NotNil(t, validator.schema)
	})

	t.Run("returns error for non-existent schema file", func(t *testing.T) {
		_, err := NewIntentSchemaValidator("/nonexistent/schema.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read schema file")
	})

	t.Run("returns error for invalid JSON schema", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidSchemaFile := filepath.Join(tempDir, "invalid-schema.json")

		err := ioutil.WriteFile(invalidSchemaFile, []byte("invalid json {"), 0o644)
		require.NoError(t, err)

		_, err = NewIntentSchemaValidator(invalidSchemaFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse schema")
	})

	t.Run("uses default schema path when empty", func(t *testing.T) {
		// Create default schema directory and file
		tempDir := t.TempDir()
		docsDir := filepath.Join(tempDir, "docs", "contracts")
		err := os.MkdirAll(docsDir, 0o755)
		require.NoError(t, err)

		defaultSchemaFile := filepath.Join(docsDir, "intent.schema.json")
		schemaData, err := json.Marshal(testSchema)
		require.NoError(t, err)
		err = ioutil.WriteFile(defaultSchemaFile, schemaData, 0o644)
		require.NoError(t, err)

		// Change working directory temporarily
		originalWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		validator, err := NewIntentSchemaValidator("")
		require.NoError(t, err)
		assert.NotNil(t, validator)
	})
}

func TestIntentSchemaValidator_Validate(t *testing.T) {
	schemaFile := createTestSchemaFile(t)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(t, err)

	t.Run("validates valid intent successfully", func(t *testing.T) {
		validIntent := map[string]interface{}{
				"name":      "test-intent",
				"namespace": "default",
			},
			"spec": json.RawMessage(`{}`),
		}

		err := validator.Validate(validIntent)
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid apiVersion", func(t *testing.T) {
		invalidIntent := map[string]interface{}{
				"name": "test-intent",
			},
			"spec": json.RawMessage(`{}`),
		}

		err := validator.Validate(invalidIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "apiVersion")
	})

	t.Run("returns error for missing required fields", func(t *testing.T) {
		incompleteIntent := map[string]interface{}{
				"intentType": "scaling",
				"target":     "nginx-deployment",
			},
		}

		err := validator.Validate(incompleteIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("returns error for invalid enum values", func(t *testing.T) {
		invalidIntent := map[string]interface{}{
				"name": "test-intent",
			},
			"spec": json.RawMessage(`{}`),
		}

		err := validator.Validate(invalidIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intentType")
	})

	t.Run("returns error for out-of-range values", func(t *testing.T) {
		invalidIntent := map[string]interface{}{
				"name": "test-intent",
			},
			"spec": json.RawMessage(`{}`),
		}

		err := validator.Validate(invalidIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "replicas")
	})

	t.Run("validates optional fields correctly", func(t *testing.T) {
		intentWithResources := map[string]interface{}{
				"name":      "test-intent",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
			},
		}

		err := validator.Validate(intentWithResources)
		assert.NoError(t, err)
	})

	t.Run("returns error for wrong data types", func(t *testing.T) {
		invalidIntent := map[string]interface{}{
				"name": "test-intent",
			},
			"spec": json.RawMessage(`{}`),
		}

		err := validator.Validate(invalidIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "replicas")
	})
}

func TestIntentSchemaValidator_ValidateJSON(t *testing.T) {
	schemaFile := createTestSchemaFile(t)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(t, err)

	t.Run("validates valid JSON string", func(t *testing.T) {
		validJSON := `{
			"apiVersion": "intent.nephoran.com/v1alpha1",
			"kind": "NetworkIntent",
			"metadata": {
				"name": "test-intent",
				"namespace": "default"
			},
			"spec": {
				"intentType": "scaling",
				"target": "nginx-deployment",
				"replicas": 3
			}
		}`

		err := validator.ValidateJSON(validJSON)
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		invalidJSON := `{
			"apiVersion": "intent.nephoran.com/v1alpha1"
			"kind": "NetworkIntent"  // Missing comma
		}`

		err := validator.ValidateJSON(invalidJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})

	t.Run("returns error for JSON that fails schema validation", func(t *testing.T) {
		invalidContentJSON := `{
			"apiVersion": "v1",
			"kind": "Pod"
		}`

		err := validator.ValidateJSON(invalidContentJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("handles empty JSON string", func(t *testing.T) {
		err := validator.ValidateJSON("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})

	t.Run("handles JSON with extra whitespace", func(t *testing.T) {
		jsonWithWhitespace := `
		
		{
			"apiVersion": "intent.nephoran.com/v1alpha1",
			"kind": "NetworkIntent",
			"metadata": {
				"name": "test-intent"
			},
			"spec": {
				"intentType": "scaling",
				"target": "nginx-deployment"
			}
		}
		
		`

		err := validator.ValidateJSON(jsonWithWhitespace)
		assert.NoError(t, err)
	})
}

func TestIntentSchemaValidator_GetSchema(t *testing.T) {
	schemaFile := createTestSchemaFile(t)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(t, err)

	t.Run("returns schema", func(t *testing.T) {
		schema := validator.GetSchema()
		assert.NotNil(t, schema)
		assert.Equal(t, "object", schema["type"])
		assert.Equal(t, "NetworkIntent Schema", schema["title"])
	})
}

func TestIntentSchemaValidator_UpdateSchema(t *testing.T) {
	schemaFile := createTestSchemaFile(t)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(t, err)

	t.Run("updates schema successfully", func(t *testing.T) {
		// Create an updated schema
		updatedSchema := map[string]interface{}{
				"apiVersion": json.RawMessage(`{}`), // Updated version
				},
			},
			"required": []string{"apiVersion"},
		}

		// Write updated schema to file
		schemaData, err := json.Marshal(updatedSchema)
		require.NoError(t, err)
		err = ioutil.WriteFile(schemaFile, schemaData, 0o644)
		require.NoError(t, err)

		// Update the validator
		err = validator.UpdateSchema()
		require.NoError(t, err)

		// Verify the schema was updated
		schema := validator.GetSchema()
		assert.Equal(t, "Updated NetworkIntent Schema", schema["title"])

		// Test validation with updated schema
		intent := json.RawMessage(`{}`)

		err = validator.Validate(intent)
		assert.NoError(t, err)
	})

	t.Run("returns error when schema file is missing", func(t *testing.T) {
		// Delete the schema file
		err := os.Remove(schemaFile)
		require.NoError(t, err)

		err = validator.UpdateSchema()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read schema file")
	})
}

func TestIntentSchemaValidator_ConcurrentAccess(t *testing.T) {
	schemaFile := createTestSchemaFile(t)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(t, err)

	t.Run("handles concurrent validation requests", func(t *testing.T) {
		validIntent := map[string]interface{}{
				"name": "test-intent",
			},
			"spec": json.RawMessage(`{}`),
		}

		// Run multiple validations concurrently
		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := validator.Validate(validIntent)
				errChan <- err
			}()
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
	})
}

func BenchmarkIntentSchemaValidator_Validate(b *testing.B) {
	schemaFile := createTestSchemaFile(b)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(b, err)

	validIntent := map[string]interface{}{
			"name":      "test-intent",
			"namespace": "default",
		},
		"spec": json.RawMessage(`{}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(validIntent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIntentSchemaValidator_ValidateJSON(b *testing.B) {
	schemaFile := createTestSchemaFile(b)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(b, err)

	validJSON := `{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind": "NetworkIntent",
		"metadata": {
			"name": "test-intent",
			"namespace": "default"
		},
		"spec": {
			"intentType": "scaling",
			"target": "nginx-deployment",
			"replicas": 3
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.ValidateJSON(validJSON)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Fuzz test for JSON validation
func FuzzIntentSchemaValidator_ValidateJSON(f *testing.F) {
	schemaFile := createTestSchemaFile(f)
	validator, err := NewIntentSchemaValidator(schemaFile)
	require.NoError(f, err)

	// Add seed corpus
	f.Add(`{"apiVersion": "intent.nephoran.com/v1alpha1", "kind": "NetworkIntent"}`)
	f.Add(`{"invalid": "json"`)
	f.Add(`""`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
		// This should not panic, but may return errors
		validator.ValidateJSON(jsonStr)
	})
}


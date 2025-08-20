package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidIntentSchemaConformance tests that valid generators create schema-compliant intents
func TestValidIntentSchemaConformance(t *testing.T) {
	factory := NewIntentFactory(t.TempDir())
	defer factory.CleanupFiles()

	tests := []struct {
		name      string
		generator func() IntentData
	}{
		{
			name: "CreateValidIntent",
			generator: func() IntentData {
				return factory.CreateValidIntent("test-app")
			},
		},
		{
			name: "CreateMinimalValidIntent", 
			generator: func() IntentData {
				return factory.CreateMinimalValidIntent("test-app")
			},
		},
		{
			name: "CreateIntentWithTarget",
			generator: func() IntentData {
				return factory.CreateIntentWithTarget("test-app", "deployment", "web-app", "production")
			},
		},
		{
			name: "CreateValidIntentWithReplicas",
			generator: func() IntentData {
				return factory.CreateValidIntentWithReplicas("test-app", 3)
			},
		},
		{
			name: "CreateValidScaleUpIntent",
			generator: func() IntentData {
				return factory.CreateValidScaleUpIntent("test-app")
			},
		},
		{
			name: "CreateValidScaleDownIntent",
			generator: func() IntentData {
				return factory.CreateValidScaleDownIntent("test-app")
			},
		},
		{
			name: "CreateValidStatefulSetIntent",
			generator: func() IntentData {
				return factory.CreateValidStatefulSetIntent("test-app")
			},
		},
		{
			name: "CreateValidDaemonSetIntent",
			generator: func() IntentData {
				return factory.CreateValidDaemonSetIntent("test-app")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := tt.generator()
			
			// Validate that target is an object, not a string
			require.IsType(t, Target{}, intent.Spec.Target, "target should be a Target object, not a string")
			
			// Validate target has required fields
			assert.NotEmpty(t, intent.Spec.Target.Type, "target.type should not be empty")
			assert.NotEmpty(t, intent.Spec.Target.Name, "target.name should not be empty")
			
			// Validate target type is supported
			supportedTypes := []string{"deployment", "statefulset", "daemonset", "replicaset"}
			assert.Contains(t, supportedTypes, intent.Spec.Target.Type, "target.type should be a supported Kubernetes resource type")
			
			// Validate action is specified
			assert.NotEmpty(t, intent.Spec.Action, "action should not be empty")
			
			// Validate JSON serialization works correctly
			data, err := json.Marshal(intent)
			require.NoError(t, err, "should serialize to JSON without error")
			
			// Validate JSON can be unmarshaled back
			var unmarshaled IntentData
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "should deserialize from JSON without error")
			
			// Validate target structure is preserved
			assert.Equal(t, intent.Spec.Target.Type, unmarshaled.Spec.Target.Type, "target.type should be preserved in JSON round-trip")
			assert.Equal(t, intent.Spec.Target.Name, unmarshaled.Spec.Target.Name, "target.name should be preserved in JSON round-trip")
		})
	}
}

// TestInvalidIntentSchemaViolations tests that invalid generators create intents that violate the schema
func TestInvalidIntentSchemaViolations(t *testing.T) {
	factory := NewInvalidFixtureFactory(t.TempDir())

	tests := []struct {
		name        string
		generator   func() map[string]interface{}
		expectError string
	}{
		{
			name: "CreateIntentWithStringTarget",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithStringTarget("test-app")
			},
			expectError: "target must be an object",
		},
		{
			name: "CreateIntentWithMissingTargetType",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithMissingTargetType("test-app")
			},
			expectError: "missing type field",
		},
		{
			name: "CreateIntentWithMissingTargetName",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithMissingTargetName("test-app")
			},
			expectError: "missing name field",
		},
		{
			name: "CreateIntentWithEmptyTarget",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithEmptyTarget("test-app")
			},
			expectError: "empty target object",
		},
		{
			name: "CreateIntentWithEmptyTargetType",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithEmptyTargetType("test-app")
			},
			expectError: "target.type must be a non-empty string",
		},
		{
			name: "CreateIntentWithEmptyTargetName",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithEmptyTargetName("test-app")
			},
			expectError: "target.name must be a non-empty string",
		},
		{
			name: "CreateIntentWithNullTarget",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithNullTarget("test-app")
			},
			expectError: "target must be an object",
		},
		{
			name: "CreateIntentWithNonStringTargetType",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithNonStringTargetType("test-app")
			},
			expectError: "target.type must be a non-empty string",
		},
		{
			name: "CreateIntentWithNonStringTargetName",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithNonStringTargetName("test-app")
			},
			expectError: "target.name must be a non-empty string",
		},
		{
			name: "CreateIntentWithArrayTarget",
			generator: func() map[string]interface{} {
				return factory.CreateIntentWithArrayTarget("test-app")
			},
			expectError: "target must be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := tt.generator()
			
			// Test that the invalid intent can be serialized (it's just structurally wrong)
			_, err := json.Marshal(intent)
			require.NoError(t, err, "invalid intent should still serialize to JSON")
			
			// Validate the target field is indeed invalid
			spec, exists := intent["spec"].(map[string]interface{})
			require.True(t, exists, "spec should exist")
			
			target, exists := spec["target"]
			require.True(t, exists, "target should exist")
			
			// Test specific invalid conditions
			switch tt.name {
			case "CreateIntentWithStringTarget":
				assert.IsType(t, "", target, "target should be string (invalid)")
			case "CreateIntentWithArrayTarget":
				assert.IsType(t, []string{}, target, "target should be array (invalid)")
			case "CreateIntentWithNullTarget":
				assert.Nil(t, target, "target should be nil (invalid)")
			default:
				// For object-based invalid targets, verify they're objects but missing/invalid fields
				if targetMap, ok := target.(map[string]interface{}); ok {
					switch tt.name {
					case "CreateIntentWithMissingTargetType":
						_, exists := targetMap["type"]
						assert.False(t, exists, "type should be missing")
					case "CreateIntentWithMissingTargetName":
						_, exists := targetMap["name"]
						assert.False(t, exists, "name should be missing")
					case "CreateIntentWithEmptyTarget":
						assert.Empty(t, targetMap, "target should be empty object")
					case "CreateIntentWithEmptyTargetType":
						assert.Equal(t, "", targetMap["type"], "type should be empty string")
					case "CreateIntentWithEmptyTargetName":
						assert.Equal(t, "", targetMap["name"], "name should be empty string")
					case "CreateIntentWithNonStringTargetType":
						assert.NotEqual(t, "string", fmt.Sprintf("%T", targetMap["type"]), "type should not be string")
					case "CreateIntentWithNonStringTargetName":
						assert.NotEqual(t, "string", fmt.Sprintf("%T", targetMap["name"]), "name should not be string")
					}
				}
			}
		})
	}
}

// TestTargetValidation validates target object structure specifically
func TestTargetValidation(t *testing.T) {
	factory := NewIntentFactory(t.TempDir())
	defer factory.CleanupFiles()

	t.Run("ValidTargetStructure", func(t *testing.T) {
		intent := factory.CreateValidIntent("test-app")
		
		// Ensure target is properly structured
		assert.NotEmpty(t, intent.Spec.Target.Type)
		assert.NotEmpty(t, intent.Spec.Target.Name)
		
		// Test JSON marshaling preserves target structure
		data, err := json.Marshal(intent)
		require.NoError(t, err)
		
		// Parse back to generic map to verify structure
		var generic map[string]interface{}
		err = json.Unmarshal(data, &generic)
		require.NoError(t, err)
		
		spec := generic["spec"].(map[string]interface{})
		target := spec["target"].(map[string]interface{})
		
		assert.IsType(t, "", target["type"], "target.type should be string")
		assert.IsType(t, "", target["name"], "target.name should be string")
		assert.NotEmpty(t, target["type"], "target.type should not be empty")
		assert.NotEmpty(t, target["name"], "target.name should not be empty")
	})

	t.Run("TargetTypeValidation", func(t *testing.T) {
		validTypes := []string{"deployment", "statefulset", "daemonset"}
		
		for _, targetType := range validTypes {
			intent := factory.CreateIntentWithTarget("test-app", targetType, "test-name", "test-namespace")
			assert.Equal(t, targetType, intent.Spec.Target.Type)
		}
	})
}

// TestBackwardCompatibilityMigration tests handling of legacy string targets
func TestBackwardCompatibilityMigration(t *testing.T) {
	t.Run("DetectLegacyStringTarget", func(t *testing.T) {
		// Simulate legacy format from old generators
		legacyJSON := `{
			"apiVersion": "v1",
			"kind": "NetworkIntent",
			"metadata": {"name": "legacy-intent"},
			"spec": {
				"action": "scale",
				"target": "deployment/legacy-app"
			}
		}`
		
		var legacy map[string]interface{}
		err := json.Unmarshal([]byte(legacyJSON), &legacy)
		require.NoError(t, err)
		
		spec := legacy["spec"].(map[string]interface{})
		target := spec["target"]
		
		// Verify it's detected as string (legacy format)
		assert.IsType(t, "", target, "legacy target should be string")
		
		// This would fail validation with the new validator
		targetStr, ok := target.(string)
		assert.True(t, ok, "should be able to cast to string")
		assert.Equal(t, "deployment/legacy-app", targetStr)
	})
}

// TestFileBasedValidation tests validation using actual files
func TestFileBasedValidation(t *testing.T) {
	tempDir := t.TempDir()
	factory := NewIntentFactory(tempDir)
	defer factory.CleanupFiles()

	t.Run("ValidIntentFile", func(t *testing.T) {
		intent := factory.CreateValidIntent("file-test")
		filePath, err := factory.CreateIntentFile("valid-intent.json", intent)
		require.NoError(t, err)
		
		// Verify file was created
		_, err = os.Stat(filePath)
		require.NoError(t, err, "file should exist")
		
		// Read and validate content
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		
		var unmarshaled IntentData
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err, "file content should be valid JSON")
		
		// Validate target structure
		assert.NotEmpty(t, unmarshaled.Spec.Target.Type)
		assert.NotEmpty(t, unmarshaled.Spec.Target.Name)
	})

	t.Run("InvalidIntentFile", func(t *testing.T) {
		invalidFactory := NewInvalidFixtureFactory(tempDir)
		
		filePath, err := invalidFactory.CreateIntentFileWithInvalidTarget("invalid-intent.json", "string_target")
		require.NoError(t, err)
		
		// Verify file exists but contains invalid structure
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		
		var generic map[string]interface{}
		err = json.Unmarshal(data, &generic)
		require.NoError(t, err, "should parse as JSON")
		
		// Verify it has the invalid structure
		spec := generic["spec"].(map[string]interface{})
		target := spec["target"]
		assert.IsType(t, "", target, "target should be string (invalid)")
	})
}

// TestPerformanceWithValidTargets tests performance implications of object vs string targets
func TestPerformanceWithValidTargets(t *testing.T) {
	factory := NewIntentFactory(t.TempDir())
	defer factory.CleanupFiles()

	t.Run("BatchCreationPerformance", func(t *testing.T) {
		const intentCount = 100
		
		// Create many valid intents with object targets
		intents := make([]IntentData, intentCount)
		for i := 0; i < intentCount; i++ {
			intents[i] = factory.CreateValidIntent(fmt.Sprintf("perf-test-%d", i))
		}
		
		// Verify all have proper target structure
		for i, intent := range intents {
			assert.NotEmpty(t, intent.Spec.Target.Type, "intent %d should have target.type", i)
			assert.NotEmpty(t, intent.Spec.Target.Name, "intent %d should have target.name", i)
		}
	})
}

// BenchmarkTargetSerialization benchmarks JSON serialization of object vs string targets
func BenchmarkTargetSerialization(b *testing.B) {
	factory := NewIntentFactory(b.TempDir())
	intent := factory.CreateValidIntent("benchmark-test")

	b.Run("ObjectTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(intent)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StringTarget", func(b *testing.B) {
		// Create legacy-style intent for comparison
		legacyIntent := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "NetworkIntent",
			"metadata":   map[string]interface{}{"name": "legacy"},
			"spec": map[string]interface{}{
				"action": "scale",
				"target": "deployment/legacy-app",
			},
		}

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(legacyIntent)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
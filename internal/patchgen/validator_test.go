package patchgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorInitialization(t *testing.T) {
	logger := logr.Discard()
	validator, err := NewValidator(logger)

	assert.NoError(t, err)
	assert.NotNil(t, validator)
}

func TestValidIntent(t *testing.T) {
	testCases := []struct {
		name        string
		intentJSON  string
		expectError bool
	}{
		{
			name: "Valid Scaling Intent",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3,
				"reason": "Increase load",
				"source": "autoscaler"
			}`,
			expectError: false,
		},
		{
			name: "Full Intent with Optional Fields",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "api-server",
				"namespace": "production",
				"replicas": 10,
				"reason": "Peak traffic",
				"source": "monitoring",
				"correlation_id": "scale-001"
			}`,
			expectError: false,
		},
		{
			name: "Invalid Intent Type",
			intentJSON: `{
				"intent_type": "delete",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
		},
		{
			name: "Missing Required Fields",
			intentJSON: `{
				"target": "web-app"
			}`,
			expectError: true,
		},
		{
			name: "Invalid Replica Count",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": -1
			}`,
			expectError: true,
		},
		{
			name: "Excessive Replica Count",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 1000
			}`,
			expectError: true,
		},
	}

	logger := logr.Discard()
	validator, _ := NewValidator(logger)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent, err := validator.ValidateIntent([]byte(tc.intentJSON))

			if tc.expectError {
				assert.Error(t, err, "Should generate validation error")
				assert.Nil(t, intent)
			} else {
				assert.NoError(t, err, "Should validate successfully")
				assert.NotNil(t, intent)
			}
		})
	}
}

func TestValidateIntentFile(t *testing.T) {
	tempDir := t.TempDir()
	logger := logr.Discard()
	validator, _ := NewValidator(logger)

	t.Run("Valid Intent File", func(t *testing.T) {
		intentContent := `{
			"intent_type": "scaling",
			"target": "web-app",
			"namespace": "default",
			"replicas": 3
		}`
		intentPath := filepath.Join(tempDir, "valid_intent.json")
		err := os.WriteFile(intentPath, []byte(intentContent), 0o644)
		require.NoError(t, err)

		intent, err := validator.ValidateIntentFile(intentPath)
		assert.NoError(t, err)
		assert.NotNil(t, intent)
	})

	t.Run("Non-existent File", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "does_not_exist.json")
		intent, err := validator.ValidateIntentFile(nonExistentPath)

		assert.Error(t, err)
		assert.Nil(t, intent)
	})
}

func TestValidateIntentMap(t *testing.T) {
	logger := logr.Discard()
	validator, _ := NewValidator(logger)

	testCases := []struct {
		name        string
		intentMap   map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid Intent Map",
			intentMap: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "web-app",
				"namespace":   "default",
				"replicas":    3,
			},
			expectError: false,
		},
		{
			name: "Invalid Intent Map",
			intentMap: map[string]interface{}{
				"intent_type": "unknown",
				"target":      "",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateIntentMap(tc.intentMap)

			if tc.expectError {
				assert.Error(t, err, "Should generate validation error")
			} else {
				assert.NoError(t, err, "Should validate successfully")
			}
		})
	}
}

func TestLoadIntent(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name        string
		intentJSON  string
		expectError bool
	}{
		{
			name: "Valid Scaling Intent",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
		},
		{
			name: "Invalid Intent Type",
			intentJSON: `{
				"intent_type": "delete",
				"target": "web-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
		},
		{
			name: "Missing Target",
			intentJSON: `{
				"intent_type": "scaling",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intentPath := filepath.Join(tempDir, "intent.json")
			err := os.WriteFile(intentPath, []byte(tc.intentJSON), 0o644)
			require.NoError(t, err)

			intent, err := LoadIntent(intentPath)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, intent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, intent)
			}
		})
	}
}

func TestIntentJSONRoundTrip(t *testing.T) {
	originalIntent := &Intent{
		IntentType:    "scaling",
		Target:        "web-app",
		Namespace:     "default",
		Replicas:      3,
		Reason:        "Load increase",
		Source:        "autoscaler",
		CorrelationID: "scale-event-001",
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(originalIntent)
	require.NoError(t, err)

	// Deserialize back to struct
	var reconstructedIntent Intent
	err = json.Unmarshal(jsonData, &reconstructedIntent)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, originalIntent.IntentType, reconstructedIntent.IntentType)
	assert.Equal(t, originalIntent.Target, reconstructedIntent.Target)
	assert.Equal(t, originalIntent.Namespace, reconstructedIntent.Namespace)
	assert.Equal(t, originalIntent.Replicas, reconstructedIntent.Replicas)
	assert.Equal(t, originalIntent.Reason, reconstructedIntent.Reason)
	assert.Equal(t, originalIntent.Source, reconstructedIntent.Source)
	assert.Equal(t, originalIntent.CorrelationID, reconstructedIntent.CorrelationID)
}

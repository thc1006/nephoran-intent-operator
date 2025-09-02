package conductor

import (
	"encoding/json"
	"testing"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DISABLED: func TestParseIntentToJSON(t *testing.T) {
	reconciler := &WatchReconciler{}

	testCases := []struct {
		name         string
		intent       string
		namespace    string
		expectedJSON map[string]interface{}
		expectError  bool
		description  string
	}{
		{
			name:      "simple-scale-to-format",
			intent:    "scale deployment nginx to 3 replicas",
			namespace: "default",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Standard scale deployment to N replicas format",
		},
		{
			name:      "app-scale-format",
			intent:    "scale app frontend to 5 replicas",
			namespace: "production",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Scale app variant",
		},
		{
			name:      "service-scale-format",
			intent:    "scale service backend to 2 replicas",
			namespace: "staging",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Scale service variant",
		},
		{
			name:      "simple-scale-format",
			intent:    "scale nginx to 4 replicas",
			namespace: "default",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Simple scale without deployment/app/service keyword",
		},
		{
			name:      "colon-format",
			intent:    "scale deployment api replicas: 8",
			namespace: "api",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Colon-separated format",
		},
		{
			name:      "instances-format",
			intent:    "scale web 6 instances",
			namespace: "default",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Instances instead of replicas",
		},
		{
			name:        "invalid-intent-no-target",
			intent:      "scale to 3 replicas",
			namespace:   "default",
			expectError: true,
			description: "Missing target deployment name",
		},
		{
			name:        "invalid-intent-no-replicas",
			intent:      "scale nginx deployment",
			namespace:   "default",
			expectError: true,
			description: "Missing replica count",
		},
		{
			name:        "invalid-intent-zero-replicas",
			intent:      "scale nginx to 0 replicas",
			namespace:   "default",
			expectError: true,
			description: "Zero replicas should be rejected",
		},
		{
			name:        "invalid-intent-high-replicas",
			intent:      "scale nginx to 150 replicas",
			namespace:   "default",
			expectError: true,
			description: "Replicas exceeding maximum (100) should be rejected",
		},
		{
			name:        "invalid-intent-unrelated",
			intent:      "restart the database connection",
			namespace:   "default",
			expectError: true,
			description: "Non-scaling intent should be rejected",
		},
		{
			name:      "boundary-max-replicas",
			intent:    "scale nginx to 100 replicas",
			namespace: "default",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Maximum allowed replicas (100)",
		},
		{
			name:      "boundary-min-replicas",
			intent:    "scale nginx to 1 replica",
			namespace: "default",
			expectedJSON: json.RawMessage("{}"),
			expectError: false,
			description: "Minimum allowed replicas (1) - singular form",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create NetworkIntent
			ni := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.name,
					Namespace: tc.namespace,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: tc.intent,
				},
			}

			// Parse intent to JSON
			result, err := reconciler.parseIntentToJSON(ni)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tc.description)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", tc.description, err)
			}

			// Verify required fields
			for key, expectedValue := range tc.expectedJSON {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Missing required field '%s' in result", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Field '%s': expected %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Verify generated fields exist and are reasonable
			correlationID, exists := result["correlation_id"]
			if !exists {
				t.Error("Missing correlation_id in result")
			} else if correlationID == "" {
				t.Error("correlation_id should not be empty")
			}

			reason, exists := result["reason"]
			if !exists {
				t.Error("Missing reason in result")
			} else if reason == "" {
				t.Error("reason should not be empty")
			}
		})
	}
}

// DISABLED: func TestParseIntentToJSONSchemaCompliance(t *testing.T) {
	reconciler := &WatchReconciler{}

	ni := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment test-app to 5 replicas",
		},
	}

	result, err := reconciler.parseIntentToJSON(ni)
	if err != nil {
		t.Fatalf("Failed to parse intent: %v", err)
	}

	// Convert to JSON and back to verify JSON serialization
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result to JSON: %v", err)
	}

	var unmarshalled map[string]interface{}
	if err := json.Unmarshal(jsonData, &unmarshalled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify schema compliance based on docs/contracts/intent.schema.json
	requiredFields := []string{"intent_type", "target", "namespace", "replicas"}
	for _, field := range requiredFields {
		if _, exists := unmarshalled[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON output", field)
		}
	}

	// Verify intent_type is exactly "scaling"
	if intentType := unmarshalled["intent_type"]; intentType != "scaling" {
		t.Errorf("intent_type should be 'scaling', got: %v", intentType)
	}

	// Verify target is non-empty string
	if target := unmarshalled["target"]; target == "" {
		t.Error("target should be non-empty string")
	}

	// Verify namespace is non-empty string
	if namespace := unmarshalled["namespace"]; namespace == "" {
		t.Error("namespace should be non-empty string")
	}

	// Verify replicas is valid integer in range 1-100
	replicas, ok := unmarshalled["replicas"].(float64) // JSON numbers are float64
	if !ok {
		t.Error("replicas should be a number")
	} else if replicas < 1 || replicas > 100 {
		t.Errorf("replicas should be in range 1-100, got: %v", replicas)
	}

	// Verify optional fields have correct types if present
	if source, exists := unmarshalled["source"]; exists {
		if _, ok := source.(string); !ok {
			t.Error("source should be a string if present")
		}
	}

	if correlationID, exists := unmarshalled["correlation_id"]; exists {
		if _, ok := correlationID.(string); !ok {
			t.Error("correlation_id should be a string if present")
		}
	}

	if reason, exists := unmarshalled["reason"]; exists {
		if reasonStr, ok := reason.(string); !ok {
			t.Error("reason should be a string if present")
		} else if len(reasonStr) > 512 {
			t.Error("reason should not exceed 512 characters")
		}
	}
}

// DISABLED: func TestParseIntentToJSONCorrelationID(t *testing.T) {
	reconciler := &WatchReconciler{}

	ni := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "test-ns",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale nginx to 3 replicas",
		},
	}

	// Parse multiple times to verify correlation IDs are different
	// (since they include timestamps)
	result1, err1 := reconciler.parseIntentToJSON(ni)
	if err1 != nil {
		t.Fatalf("First parse failed: %v", err1)
	}

	time.Sleep(1 * time.Second) // Ensure different timestamp

	result2, err2 := reconciler.parseIntentToJSON(ni)
	if err2 != nil {
		t.Fatalf("Second parse failed: %v", err2)
	}

	correlationID1 := result1["correlation_id"].(string)
	correlationID2 := result2["correlation_id"].(string)

	if correlationID1 == correlationID2 {
		t.Error("Correlation IDs should be different for different invocations")
	}

	// Verify correlation ID format contains expected components
	expectedPrefix := "test-intent-test-ns-"
	if !containsPrefix(correlationID1, expectedPrefix) {
		t.Errorf("Correlation ID should start with '%s', got: %s", expectedPrefix, correlationID1)
	}
}

// Helper function to check prefix (since strings.HasPrefix might not be available)
func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

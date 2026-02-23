package loop

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// TestIntentToNetworkIntentCR validates the conversion logic
func TestIntentToNetworkIntentCR(t *testing.T) {
	tests := []struct {
		name        string
		intent      *ingest.Intent
		wantErr     bool
		checkFunc   func(t *testing.T, obj *unstructured.Unstructured)
	}{
		{
			name: "basic intent with all required fields",
			intent: &ingest.Intent{
				IntentType: "scaling",
				Target:     "my-deployment",
				Namespace:  "default",
				Replicas:   3,
				Source:     "test-source",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				assert.Equal(t, "intent.nephoran.com/v1alpha1", obj.GetAPIVersion())
				assert.Equal(t, "NetworkIntent", obj.GetKind())
				assert.Contains(t, obj.GetName(), "intent-my-deployment-")

				// Check spec fields
				spec := obj.Object["spec"].(map[string]interface{})
				assert.Equal(t, "test-source", spec["source"])
				assert.Equal(t, "scaling", spec["intentType"])
				assert.Equal(t, "my-deployment", spec["target"])
				assert.Equal(t, "default", spec["namespace"])
				assert.Equal(t, int32(3), spec["replicas"])

				// Check labels
				labels := obj.GetLabels()
				assert.Equal(t, "nephoran-conductor", labels["app.kubernetes.io/managed-by"])
				assert.Equal(t, "scaling", labels["nephoran.com/intent-type"])
				assert.Equal(t, "my-deployment", labels["nephoran.com/target"])
			},
		},
		{
			name: "intent with correlation ID (should be annotation, not spec)",
			intent: &ingest.Intent{
				IntentType:    "scaling",
				Target:        "test-app",
				Namespace:     "production",
				Replicas:      5,
				CorrelationID: "req-12345",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				// CRITICAL: correlationId should NOT be in spec
				spec := obj.Object["spec"].(map[string]interface{})
				_, hasCorrelationInSpec := spec["correlationId"]
				assert.False(t, hasCorrelationInSpec, "correlationId should not be in spec")

				// CRITICAL: correlationId SHOULD be in annotations
				annotations := obj.GetAnnotations()
				assert.NotNil(t, annotations, "annotations should exist")
				assert.Equal(t, "req-12345", annotations["nephoran.com/correlation-id"])
			},
		},
		{
			name: "intent with reason (should be annotation)",
			intent: &ingest.Intent{
				IntentType: "scaling",
				Target:     "api-server",
				Namespace:  "backend",
				Replicas:   10,
				Reason:     "High CPU utilization detected",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				annotations := obj.GetAnnotations()
				assert.NotNil(t, annotations)
				assert.Equal(t, "High CPU utilization detected", annotations["nephoran.com/reason"])
			},
		},
		{
			name: "intent with default source",
			intent: &ingest.Intent{
				IntentType: "scaling",
				Target:     "worker",
				Namespace:  "jobs",
				Replicas:   2,
				// Source not specified
			},
			wantErr: false,
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				spec := obj.Object["spec"].(map[string]interface{})
				assert.Equal(t, "conductor-loop", spec["source"], "should default to conductor-loop")
			},
		},
		{
			name: "intent with special characters in target (DNS-1123 sanitization)",
			intent: &ingest.Intent{
				IntentType: "scaling",
				Target:     "My_App@Service#123",
				Namespace:  "default",
				Replicas:   1,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				// Name should be sanitized to DNS-1123 format
				name := obj.GetName()
				assert.Regexp(t, regexp.MustCompile(`^intent-my-app-service-123-[a-f0-9]{8}$`), name)

				// Label value should also be sanitized but preserve more characters
				labels := obj.GetLabels()
				sanitizedLabel := labels["nephoran.com/target"]
				assert.NotContains(t, sanitizedLabel, "@")
				assert.NotContains(t, sanitizedLabel, "#")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := intentToNetworkIntentCR(tt.intent)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, obj)

			if tt.checkFunc != nil {
				tt.checkFunc(t, obj)
			}
		})
	}
}

// TestGenerateSecureRandomID validates random ID generation
func TestGenerateSecureRandomID(t *testing.T) {
	// Test that it generates valid hex strings
	id1, err := generateSecureRandomID()
	require.NoError(t, err)
	assert.Len(t, id1, 8, "should generate 8 hex characters")
	assert.Regexp(t, regexp.MustCompile(`^[a-f0-9]{8}$`), id1)

	// Test uniqueness (should be extremely unlikely to collide)
	id2, err := generateSecureRandomID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2, "IDs should be unique")

	// Test multiple generations for entropy
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSecureRandomID()
		require.NoError(t, err)
		assert.False(t, seen[id], "should not generate duplicate IDs")
		seen[id] = true
	}
}

// TestSanitizeDNS1123Name validates DNS-1123 name sanitization
func TestSanitizeDNS1123Name(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"UPPERCASE", "uppercase"},
		{"with-dashes", "with-dashes"},
		{"with_underscores", "with-underscores"},
		{"with@special#chars", "with-special-chars"},
		{"123numeric", "123numeric"},
		{"-leading-dash", "leading-dash"},
		{"trailing-dash-", "trailing-dash"},
		{"--multiple--dashes--", "multiple--dashes"},
		{"", "default"},
		{"a", "a"},
		{"VeryLongNameThatExceedsTheMaximumLengthOfFortyCharactersForDNS", "verylongnamethatexceedsthemaximumlengtho"},
		{"my_app@service#v2", "my-app-service-v2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeDNS1123Name(tt.input)
			assert.Equal(t, tt.expected, result)

			// Validate DNS-1123 compliance
			assert.Regexp(t, regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`), result)
			assert.LessOrEqual(t, len(result), 40, "should not exceed 40 chars")
		})
	}
}

// TestSanitizeLabelValue validates label value sanitization
func TestSanitizeLabelValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dashes", "with-dashes"},
		{"with_underscores", "with_underscores"},
		{"with.dots", "with.dots"},
		{"with@special#chars", "with-special-chars"},
		{"-leading-dash", "leading-dash"},
		{"trailing-dash-", "trailing-dash"},
		{"", ""},
		{"VeryLongLabelValueThatExceedsSixtyThreeCharactersWhichIsTheMaximumLengthAllowedForLabels", "VeryLongLabelValueThatExceedsSixtyThreeCharactersWhichIsTheMaxi"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeLabelValue(tt.input)
			if tt.expected == "" && tt.input == "" {
				assert.Equal(t, "", result)
				return
			}

			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63, "should not exceed 63 chars")

			// Validate label value compliance (if not empty)
			if result != "" {
				assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`), result)
			}
		})
	}
}

// TestK8sSubmitFuncSignature validates that K8sSubmitFunc matches PorchSubmitFunc signature
func TestK8sSubmitFuncSignature(t *testing.T) {
	// This test ensures the function signature is correct
	var _ PorchSubmitFunc = K8sSubmitFunc

	// Test that we can call the function with correct parameters
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	intent := &ingest.Intent{
		IntentType: "scaling",
		Target:     "test",
		Namespace:  "default",
		Replicas:   1,
	}

	// Note: This might succeed if we're running in a K8s cluster environment
	err := K8sSubmitFunc(ctx, intent, "direct")
	// Just validate that we can call it - error or success both acceptable
	_ = err // May succeed if running in K8s cluster, may fail otherwise
}

// TestK8sSubmitFuncContextCancellation validates early context check
func TestK8sSubmitFuncContextCancellation(t *testing.T) {
	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	intent := &ingest.Intent{
		IntentType: "scaling",
		Target:     "test",
		Namespace:  "default",
		Replicas:   1,
	}

	err := K8sSubmitFunc(ctx, intent, "direct")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

// TestK8sSubmitFuncNilIntent validates nil check for intent parameter
func TestK8sSubmitFuncNilIntent(t *testing.T) {
	ctx := context.Background()

	err := K8sSubmitFunc(ctx, nil, "direct")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "intent cannot be nil")
}

// TestK8sSubmitFactory validates the factory pattern
func TestK8sSubmitFactory(t *testing.T) {
	// This test validates the factory returns a valid function
	// In a real environment with K8s, this would create a reusable client
	submitFunc, err := K8sSubmitFactory()

	// We expect an error because we're not in a K8s cluster
	if err != nil {
		assert.Contains(t, err.Error(), "kubernetes config")
		return
	}

	// If we got a function, validate its signature
	var _ PorchSubmitFunc = submitFunc
	assert.NotNil(t, submitFunc)
}

// TestIsAlphanumeric validates the helper function
func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{'_', false},
		{'@', false},
		{' ', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := isAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetSource validates source defaulting
func TestGetSource(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "conductor-loop"},
		{"custom-source", "custom-source"},
		{"api-gateway", "api-gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getSource(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

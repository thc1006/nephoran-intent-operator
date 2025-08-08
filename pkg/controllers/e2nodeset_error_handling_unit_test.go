package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Test helper to create a basic E2NodeSet for unit tests
func createBasicE2NodeSet(name, namespace string) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: 1,
		},
	}
}

func TestCalculateExponentialBackoffUnit(t *testing.T) {
	tests := []struct {
		name        string
		retryCount  int
		baseDelay   time.Duration
		maxDelay    time.Duration
		expectRange struct {
			min time.Duration
			max time.Duration
		}
	}{
		{
			name:       "first retry with default values",
			retryCount: 0,
			baseDelay:  0, // Use default
			maxDelay:   0, // Use default
			expectRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 900 * time.Millisecond,  // BaseBackoffDelay (1s) with jitter -10%
				max: 1100 * time.Millisecond, // BaseBackoffDelay (1s) with jitter +10%
			},
		},
		{
			name:       "second retry with exponential backoff",
			retryCount: 1,
			baseDelay:  2 * time.Second,
			maxDelay:   5 * time.Minute,
			expectRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 3600 * time.Millisecond, // 2s * 2^1 = 4s, with jitter -10% = 3.6s
				max: 4400 * time.Millisecond, // 2s * 2^1 = 4s, with jitter +10% = 4.4s
			},
		},
		{
			name:       "high retry count capped at max delay",
			retryCount: 10,
			baseDelay:  time.Second,
			maxDelay:   10 * time.Second,
			expectRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 9 * time.Second,  // maxDelay with jitter -10%
				max: 10 * time.Second, // maxDelay (no jitter can exceed max)
			},
		},
		{
			name:       "zero retry count",
			retryCount: 0,
			baseDelay:  5 * time.Second,
			maxDelay:   30 * time.Second,
			expectRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 4500 * time.Millisecond, // 5s with jitter -10%
				max: 5500 * time.Millisecond, // 5s with jitter +10%
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter randomness
			for i := 0; i < 10; i++ {
				delay := calculateExponentialBackoff(tt.retryCount, tt.baseDelay, tt.maxDelay)
				assert.True(t, delay >= tt.expectRange.min, 
					"Delay %v should be >= %v", delay, tt.expectRange.min)
				assert.True(t, delay <= tt.expectRange.max, 
					"Delay %v should be <= %v", delay, tt.expectRange.max)
				
				// Verify delay is positive
				assert.True(t, delay > 0, "Delay should be positive")
			}
		})
	}
}

func TestCalculateExponentialBackoffForOperationUnit(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		retryCount int
		expectedRange struct {
			min time.Duration
			max time.Duration
		}
	}{
		{
			name:       "configmap operations backoff",
			operation:  "configmap-operations",
			retryCount: 1,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 3600 * time.Millisecond, // 2s * 2^1 = 4s, with jitter -10%
				max: 4400 * time.Millisecond, // 2s * 2^1 = 4s, with jitter +10%
			},
		},
		{
			name:       "e2 provisioning backoff",
			operation:  "e2-provisioning",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 4500 * time.Millisecond, // 5s with jitter -10%
				max: 5500 * time.Millisecond, // 5s with jitter +10%
			},
		},
		{
			name:       "cleanup operations backoff",
			operation:  "cleanup",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 9 * time.Second,  // 10s with jitter -10%
				max: 11 * time.Second, // 10s with jitter +10%
			},
		},
		{
			name:       "unknown operation uses default",
			operation:  "unknown-operation",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 900 * time.Millisecond,  // 1s with jitter -10%
				max: 1100 * time.Millisecond, // 1s with jitter +10%
			},
		},
		{
			name:       "configmap operations at max retries",
			operation:  "configmap-operations",
			retryCount: 6, // High retry count
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 108 * time.Second, // 2min with jitter -10%
				max: 120 * time.Second, // 2min (max for configmap operations)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter randomness
			for i := 0; i < 5; i++ {
				delay := calculateExponentialBackoffForOperation(tt.retryCount, tt.operation)
				assert.True(t, delay >= tt.expectedRange.min, 
					"Delay %v should be >= %v for operation %s", delay, tt.expectedRange.min, tt.operation)
				assert.True(t, delay <= tt.expectedRange.max, 
					"Delay %v should be <= %v for operation %s", delay, tt.expectedRange.max, tt.operation)
			}
		})
	}
}

func TestGetRetryCount(t *testing.T) {
	tests := []struct {
		name               string
		initialAnnotations map[string]string
		operation          string
		expectedRetryCount int
	}{
		{
			name:               "no existing annotations",
			initialAnnotations: nil,
			operation:          "configmap-operations",
			expectedRetryCount: 0,
		},
		{
			name: "existing retry count",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "2",
			},
			operation:          "configmap-operations",
			expectedRetryCount: 2,
		},
		{
			name: "invalid retry count defaults to zero",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "invalid",
			},
			operation:          "configmap-operations",
			expectedRetryCount: 0,
		},
		{
			name: "empty retry count defaults to zero",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "",
			},
			operation:          "configmap-operations",
			expectedRetryCount: 0,
		},
		{
			name: "different operation returns zero",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "3",
			},
			operation:          "e2-provisioning",
			expectedRetryCount: 0,
		},
		{
			name: "multiple operation retry counts",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "3",
				"nephoran.com/e2-provisioning-retry-count":      "1",
				"nephoran.com/cleanup-retry-count":              "2",
			},
			operation:          "e2-provisioning",
			expectedRetryCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2nodeSet := createBasicE2NodeSet("test", "default")
			e2nodeSet.Annotations = tt.initialAnnotations

			retryCount := getRetryCount(e2nodeSet, tt.operation)
			assert.Equal(t, tt.expectedRetryCount, retryCount)
		})
	}
}

func TestSetRetryCount(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		count     int
	}{
		{
			name:      "set configmap operations retry count",
			operation: "configmap-operations",
			count:     3,
		},
		{
			name:      "set e2 provisioning retry count",
			operation: "e2-provisioning",
			count:     1,
		},
		{
			name:      "set cleanup retry count",
			operation: "cleanup",
			count:     2,
		},
		{
			name:      "set zero retry count",
			operation: "test-operation",
			count:     0,
		},
		{
			name:      "set maximum retry count",
			operation: "max-retries",
			count:     DefaultMaxRetries,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2nodeSet := createBasicE2NodeSet("test", "default")

			setRetryCount(e2nodeSet, tt.operation, tt.count)

			expectedKey := "nephoran.com/" + tt.operation + "-retry-count"
			assert.NotNil(t, e2nodeSet.Annotations)
			assert.Equal(t, string(rune('0'+tt.count)), e2nodeSet.Annotations[expectedKey])

			// Verify retrieval works
			retrievedCount := getRetryCount(e2nodeSet, tt.operation)
			assert.Equal(t, tt.count, retrievedCount)
		})
	}
}

func TestSetRetryCountWithExistingAnnotations(t *testing.T) {
	e2nodeSet := createBasicE2NodeSet("test", "default")
	e2nodeSet.Annotations = map[string]string{
		"existing-annotation": "existing-value",
		"nephoran.com/other-retry-count": "5",
	}

	operation := "configmap-operations"
	count := 3

	setRetryCount(e2nodeSet, operation, count)

	// Verify new retry count is set
	expectedKey := "nephoran.com/configmap-operations-retry-count"
	assert.Equal(t, "3", e2nodeSet.Annotations[expectedKey])

	// Verify existing annotations are preserved
	assert.Equal(t, "existing-value", e2nodeSet.Annotations["existing-annotation"])
	assert.Equal(t, "5", e2nodeSet.Annotations["nephoran.com/other-retry-count"])
}

func TestClearRetryCount(t *testing.T) {
	tests := []struct {
		name                string
		initialAnnotations  map[string]string
		operationToClear    string
		expectedAnnotations map[string]string
	}{
		{
			name: "clear existing retry count",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "5",
				"other-annotation": "value",
			},
			operationToClear: "configmap-operations",
			expectedAnnotations: map[string]string{
				"other-annotation": "value",
			},
		},
		{
			name:                "clear non-existent retry count",
			initialAnnotations:  map[string]string{"other-annotation": "value"},
			operationToClear:    "e2-provisioning",
			expectedAnnotations: map[string]string{"other-annotation": "value"},
		},
		{
			name:             "clear from empty annotations",
			operationToClear: "cleanup",
		},
		{
			name: "clear one of multiple retry counts",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "3",
				"nephoran.com/e2-provisioning-retry-count":      "1",
				"nephoran.com/cleanup-retry-count":              "2",
			},
			operationToClear: "e2-provisioning",
			expectedAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "3",
				"nephoran.com/cleanup-retry-count":              "2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2nodeSet := createBasicE2NodeSet("test", "default")
			e2nodeSet.Annotations = tt.initialAnnotations

			// Set retry count first to test clearing
			if tt.initialAnnotations != nil {
				expectedKey := "nephoran.com/" + tt.operationToClear + "-retry-count"
				if _, exists := tt.initialAnnotations[expectedKey]; exists {
					// Verify it exists before clearing
					assert.NotEqual(t, 0, getRetryCount(e2nodeSet, tt.operationToClear))
				}
			}

			// Clear it
			clearRetryCount(e2nodeSet, tt.operationToClear)

			// Verify the specific retry count is cleared
			assert.Equal(t, 0, getRetryCount(e2nodeSet, tt.operationToClear))

			// Verify annotation is removed
			expectedKey := "nephoran.com/" + tt.operationToClear + "-retry-count"
			if e2nodeSet.Annotations != nil {
				_, exists := e2nodeSet.Annotations[expectedKey]
				assert.False(t, exists, "Retry count annotation should be removed")
			}

			// Verify expected annotations remain
			if tt.expectedAnnotations != nil {
				for key, expectedValue := range tt.expectedAnnotations {
					actualValue, exists := e2nodeSet.Annotations[key]
					assert.True(t, exists, "Expected annotation %s should exist", key)
					assert.Equal(t, expectedValue, actualValue, "Expected annotation %s should have correct value", key)
				}
			}
		})
	}
}

func TestRetryCountWorkflow(t *testing.T) {
	e2nodeSet := createBasicE2NodeSet("test-workflow", "default")
	operation := "e2-provisioning"

	// Initially should be 0
	assert.Equal(t, 0, getRetryCount(e2nodeSet, operation))

	// Set to 1
	setRetryCount(e2nodeSet, operation, 1)
	assert.Equal(t, 1, getRetryCount(e2nodeSet, operation))

	// Increment to 2
	currentCount := getRetryCount(e2nodeSet, operation)
	setRetryCount(e2nodeSet, operation, currentCount+1)
	assert.Equal(t, 2, getRetryCount(e2nodeSet, operation))

	// Increment to 3 (max retries)
	currentCount = getRetryCount(e2nodeSet, operation)
	setRetryCount(e2nodeSet, operation, currentCount+1)
	assert.Equal(t, 3, getRetryCount(e2nodeSet, operation))

	// Clear back to 0
	clearRetryCount(e2nodeSet, operation)
	assert.Equal(t, 0, getRetryCount(e2nodeSet, operation))
}

func TestRetryCountEdgeCases(t *testing.T) {
	e2nodeSet := createBasicE2NodeSet("test-edge", "default")

	// Test with negative count (should still work)
	setRetryCount(e2nodeSet, "test-op", -1)
	assert.Equal(t, -1, getRetryCount(e2nodeSet, "test-op"))

	// Test with very large count
	setRetryCount(e2nodeSet, "large-op", 999999)
	assert.Equal(t, 999999, getRetryCount(e2nodeSet, "large-op"))

	// Test with empty operation name
	setRetryCount(e2nodeSet, "", 5)
	assert.Equal(t, 5, getRetryCount(e2nodeSet, ""))

	// Test with operation containing special characters
	specialOp := "operation-with.special@chars"
	setRetryCount(e2nodeSet, specialOp, 2)
	assert.Equal(t, 2, getRetryCount(e2nodeSet, specialOp))
}

func TestBackoffConstants(t *testing.T) {
	// Verify constants are reasonable values
	assert.Equal(t, 3, DefaultMaxRetries)
	assert.Equal(t, 1*time.Second, BaseBackoffDelay)
	assert.Equal(t, 5*time.Minute, MaxBackoffDelay)
	assert.Equal(t, 0.1, JitterFactor)
	assert.Equal(t, 2.0, BackoffMultiplier)
}

func TestBackoffBoundaries(t *testing.T) {
	// Test that backoff always stays within reasonable bounds
	for retryCount := 0; retryCount < 20; retryCount++ {
		delay := calculateExponentialBackoff(retryCount, BaseBackoffDelay, MaxBackoffDelay)
		
		// Should never be negative
		assert.True(t, delay >= 0, "Delay should never be negative")
		
		// Should never exceed max delay
		assert.True(t, delay <= MaxBackoffDelay, "Delay should never exceed max delay")
		
		// For first retry, should be close to base delay
		if retryCount == 0 {
			assert.True(t, delay >= BaseBackoffDelay/2, "First retry delay should be reasonable")
		}
	}
}
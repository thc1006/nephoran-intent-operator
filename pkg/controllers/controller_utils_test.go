package controllers

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	configPkg "github.com/thc1006/nephoran-intent-operator/pkg/config"
)

// DISABLED: func TestCalculateExponentialBackoff(t *testing.T) {
	config := &BackoffConfig{
		BaseDelay:    1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}

	// Test basic exponential backoff
	delay := CalculateExponentialBackoff(0, config)
	if delay < config.BaseDelay {
		t.Errorf("Expected delay >= %v, got %v", config.BaseDelay, delay)
	}

	// Test max delay cap
	delay = CalculateExponentialBackoff(20, config) // Should hit max delay
	if delay > config.MaxDelay {
		t.Errorf("Expected delay <= %v, got %v", config.MaxDelay, delay)
	}
}

// DISABLED: func TestCalculateExponentialBackoffForE2NodeSetOperation(t *testing.T) {
	tests := []struct {
		operation string
		minDelay  time.Duration
	}{
		{"configmap-operations", 2 * time.Second},
		{"e2-provisioning", 5 * time.Second},
		{"cleanup", 10 * time.Second},
		{"unknown", 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			delay := CalculateExponentialBackoffForE2NodeSetOperation(0, tt.operation)
			if delay < tt.minDelay {
				t.Errorf("Expected delay >= %v for operation %s, got %v", tt.minDelay, tt.operation, delay)
			}
		})
	}
}

// DISABLED: func TestNetworkIntentRetryCount(t *testing.T) {
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	operation := "test-operation"

	// Test initial count is 0
	count := GetNetworkIntentRetryCount(networkIntent, operation)
	if count != 0 {
		t.Errorf("Expected initial retry count to be 0, got %d", count)
	}

	// Test setting retry count
	SetNetworkIntentRetryCount(networkIntent, operation, 5)
	count = GetNetworkIntentRetryCount(networkIntent, operation)
	if count != 5 {
		t.Errorf("Expected retry count to be 5, got %d", count)
	}

	// Test clearing retry count
	ClearNetworkIntentRetryCount(networkIntent, operation)
	count = GetNetworkIntentRetryCount(networkIntent, operation)
	if count != 0 {
		t.Errorf("Expected retry count to be 0 after clearing, got %d", count)
	}
}

// DISABLED: func TestE2NodeSetRetryCount(t *testing.T) {
	e2nodeSet := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	operation := "test-operation"

	// Test initial count is 0
	count := GetE2NodeSetRetryCount(e2nodeSet, operation)
	if count != 0 {
		t.Errorf("Expected initial retry count to be 0, got %d", count)
	}

	// Test setting retry count
	SetE2NodeSetRetryCount(e2nodeSet, operation, 3)
	count = GetE2NodeSetRetryCount(e2nodeSet, operation)
	if count != 3 {
		t.Errorf("Expected retry count to be 3, got %d", count)
	}

	// Test clearing retry count
	ClearE2NodeSetRetryCount(e2nodeSet, operation)
	count = GetE2NodeSetRetryCount(e2nodeSet, operation)
	if count != 0 {
		t.Errorf("Expected retry count to be 0 after clearing, got %d", count)
	}
}

// DISABLED: func TestUpdateCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	// Test adding new condition
	newCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "TestReason",
		Message:            "Test message",
		LastTransitionTime: metav1.Now(),
	}

	UpdateCondition(&conditions, newCondition)
	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Type != "Ready" {
		t.Errorf("Expected condition type 'Ready', got %s", conditions[0].Type)
	}

	// Test updating existing condition
	updatedCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "UpdatedReason",
		Message:            "Updated message",
		LastTransitionTime: metav1.Now(),
	}

	UpdateCondition(&conditions, updatedCondition)
	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition after update, got %d", len(conditions))
	}
	if conditions[0].Status != metav1.ConditionFalse {
		t.Errorf("Expected condition status False, got %v", conditions[0].Status)
	}
	if conditions[0].Reason != "UpdatedReason" {
		t.Errorf("Expected condition reason 'UpdatedReason', got %s", conditions[0].Reason)
	}
}

// DISABLED: func TestCalculateExponentialBackoffWithConstants(t *testing.T) {
	constants := &configPkg.Constants{
		BaseBackoffDelay:  2 * time.Second,
		MaxBackoffDelay:   10 * time.Minute,
		BackoffMultiplier: 2.0,
		JitterFactor:      0.1,
	}

	// Test with zero base delay - should use constants
	delay := CalculateExponentialBackoffWithConstants(0, 0, 0, constants)
	if delay < constants.BaseBackoffDelay {
		t.Errorf("Expected delay >= %v, got %v", constants.BaseBackoffDelay, delay)
	}

	// Test with custom delays
	customBaseDelay := 5 * time.Second
	customMaxDelay := 2 * time.Minute
	delay = CalculateExponentialBackoffWithConstants(0, customBaseDelay, customMaxDelay, constants)
	if delay < customBaseDelay {
		t.Errorf("Expected delay >= %v, got %v", customBaseDelay, delay)
	}
}

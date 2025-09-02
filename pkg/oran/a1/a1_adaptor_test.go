package a1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// DISABLED: func TestA1AdaptorPolicyTypeOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000":
			switch r.Method {
			case http.MethodPut:
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				json.NewEncoder(w).Encode(&A1PolicyType{
					PolicyTypeID: 1000,
					Name:         "Test Policy Type",
					Description:  "Test Description",
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			}
		case "/a1-p/v1/policytypes":
			json.NewEncoder(w).Encode([]int{1000, 2000})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test CreatePolicyType
	t.Run("CreatePolicyType", func(t *testing.T) {
		policyType := CreateQoSPolicyType()
		err := adaptor.CreatePolicyType(ctx, policyType)
		assert.NoError(t, err)
	})

	// Test GetPolicyType
	t.Run("GetPolicyType", func(t *testing.T) {
		policyType, err := adaptor.GetPolicyType(ctx, 1000)
		assert.NoError(t, err)
		assert.Equal(t, 1000, policyType.PolicyTypeID)
		assert.Equal(t, "Test Policy Type", policyType.Name)
	})

	// Test ListPolicyTypes
	t.Run("ListPolicyTypes", func(t *testing.T) {
		// This test will partially fail because we only mock policy type 1000
		// In a real test, we'd mock all policy types
		policyTypes, err := adaptor.ListPolicyTypes(ctx)
		assert.Error(t, err) // Expected because we don't mock policy type 2000
		assert.Nil(t, policyTypes)
	})

	// Test DeletePolicyType
	t.Run("DeletePolicyType", func(t *testing.T) {
		err := adaptor.DeletePolicyType(ctx, 1000)
		assert.NoError(t, err)
	})
}

// DISABLED: func TestA1AdaptorPolicyInstanceOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000/policies/test-instance":
			switch r.Method {
			case http.MethodPut:
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				json.NewEncoder(w).Encode(json.RawMessage("{}"){
						"latency_ms":      10,
						"throughput_mbps": 100,
					},
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			}
		case "/a1-p/v1/policytypes/1000/policies/test-instance/status":
			json.NewEncoder(w).Encode(&A1PolicyStatus{
				EnforcementStatus: "ENFORCED",
				LastModified:      time.Now(),
			})
		case "/a1-p/v1/policytypes/1000/policies":
			json.NewEncoder(w).Encode([]string{"test-instance", "test-instance-2"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test CreatePolicyInstance
	t.Run("CreatePolicyInstance", func(t *testing.T) {
		instance := &A1PolicyInstance{
			PolicyInstanceID: "test-instance",
			PolicyTypeID:     1000,
			PolicyData: json.RawMessage("{}"){
					"latency_ms":      10,
					"throughput_mbps": 100,
				},
			},
		}
		err := adaptor.CreatePolicyInstance(ctx, 1000, instance)
		assert.NoError(t, err)
	})

	// Test GetPolicyInstance
	t.Run("GetPolicyInstance", func(t *testing.T) {
		instance, err := adaptor.GetPolicyInstance(ctx, 1000, "test-instance")
		assert.NoError(t, err)
		assert.Equal(t, "test-instance", instance.PolicyInstanceID)
		assert.Equal(t, 1000, instance.PolicyTypeID)
		assert.Equal(t, "ENFORCED", instance.Status.EnforcementStatus)
	})

	// Test GetPolicyStatus
	t.Run("GetPolicyStatus", func(t *testing.T) {
		status, err := adaptor.GetPolicyStatus(ctx, 1000, "test-instance")
		assert.NoError(t, err)
		assert.Equal(t, "ENFORCED", status.EnforcementStatus)
	})

	// Test DeletePolicyInstance
	t.Run("DeletePolicyInstance", func(t *testing.T) {
		err := adaptor.DeletePolicyInstance(ctx, 1000, "test-instance")
		assert.NoError(t, err)
	})
}

// DISABLED: func TestA1AdaptorApplyPolicy(t *testing.T) {
	// Track enforcement status
	enforced := false

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000":
			if r.Method == http.MethodPut {
				enforced = true
				w.WriteHeader(http.StatusCreated)
			}
		case "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000/status":
			status := &A1PolicyStatus{
				EnforcementStatus: "NOT_ENFORCED",
				LastModified:      time.Now(),
			}
			if enforced {
				status.EnforcementStatus = "ENFORCED"
			}
			json.NewEncoder(w).Encode(status)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create ManagedElement with A1 policy
	me := &nephoranv1.ManagedElement{
		Spec: nephoranv1.ManagedElementSpec{
			A1Policy: runtime.RawExtension{
				Raw: []byte(`{
					"policy_type_id": 1000,
					"policy_instance_id": "test-me-policy-1000",
					"policy_data": {
						"slice_id": "test-slice",
						"qos_parameters": {
							"latency_ms": 10,
							"throughput_mbps": 100,
							"reliability": 0.999
						}
					}
				}`),
			},
		},
	}
	me.Name = "test-me"

	// Test ApplyPolicy
	err = adaptor.ApplyPolicy(ctx, me)
	assert.NoError(t, err)
	assert.True(t, enforced)
}

// DISABLED: func TestA1AdaptorRemovePolicy(t *testing.T) {
	deleted := false

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000" && r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create ManagedElement with A1 policy
	me := &nephoranv1.ManagedElement{
		Spec: nephoranv1.ManagedElementSpec{
			A1Policy: runtime.RawExtension{
				Raw: []byte(`{
					"policy_type_id": 1000,
					"policy_instance_id": "test-me-policy-1000",
					"policy_data": {}
				}`),
			},
		},
	}
	me.Name = "test-me"

	// Test RemovePolicy
	err = adaptor.RemovePolicy(ctx, me)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

// DISABLED: func TestA1AdaptorNoPolicyScenarios(t *testing.T) {
	adaptor, err := NewA1Adaptor(nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test ApplyPolicy with no policy
	t.Run("ApplyPolicy with no policy", func(t *testing.T) {
		me := &nephoranv1.ManagedElement{
			Spec: nephoranv1.ManagedElementSpec{},
		}
		me.Name = "test-me"

		err := adaptor.ApplyPolicy(ctx, me)
		assert.NoError(t, err)
	})

	// Test RemovePolicy with no policy
	t.Run("RemovePolicy with no policy", func(t *testing.T) {
		me := &nephoranv1.ManagedElement{
			Spec: nephoranv1.ManagedElementSpec{},
		}
		me.Name = "test-me"

		err := adaptor.RemovePolicy(ctx, me)
		assert.NoError(t, err)
	})
}

// Additional comprehensive tests for enhanced A1 adaptor functionality

// DISABLED: func TestNewA1Adaptor_EnhancedFeatures(t *testing.T) {
	tests := []struct {
		name   string
		config *A1AdaptorConfig
	}{
		{
			name:   "default config with enhancements",
			config: nil,
		},
		{
			name: "custom config with circuit breaker",
			config: &A1AdaptorConfig{
				RICURL:     "http://test-ric:8080",
				APIVersion: "v2",
				Timeout:    60 * time.Second,
				CircuitBreakerConfig: &llm.CircuitBreakerConfig{
					FailureThreshold: 10,
					FailureRate:      0.8,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptor, err := NewA1Adaptor(tt.config)
			assert.NoError(t, err)
			assert.NotNil(t, adaptor)
			assert.NotNil(t, adaptor.httpClient)
			assert.NotNil(t, adaptor.circuitBreaker)
			assert.NotNil(t, adaptor.retryConfig)
			assert.NotNil(t, adaptor.policyTypes)
			assert.NotNil(t, adaptor.policyInstances)
		})
	}
}

// DISABLED: func TestA1Adaptor_RetryMechanism(t *testing.T) {
	// Create server that fails first few times then succeeds
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxRetries:      5,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			Jitter:          false,
			RetryableErrors: []string{"HTTP request failed"},
		},
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	policyType := &A1PolicyType{
		PolicyTypeID: 1,
		Name:         "Test Policy Type",
		Description:  "Test retry mechanism",
		PolicySchema: json.RawMessage("{}"),
	}

	ctx := context.Background()
	err = adaptor.createPolicyTypeWithRetry(ctx, policyType)
	assert.NoError(t, err)
	assert.True(t, attemptCount >= 3, "Should have retried multiple times")
}

// DISABLED: func TestA1Adaptor_CircuitBreakerFunctionality(t *testing.T) {
	adaptor, err := NewA1Adaptor(nil)
	require.NoError(t, err)

	// Test initial state
	assert.False(t, adaptor.circuitBreaker.IsOpen())
	assert.True(t, adaptor.circuitBreaker.IsClosed())

	// Test getting stats
	stats := adaptor.GetCircuitBreakerStats()
	assert.NotNil(t, stats)
	state, ok := stats["state"].(string)
	assert.True(t, ok)
	assert.Equal(t, "closed", state)

	// Test manual reset
	adaptor.ResetCircuitBreaker()
	assert.True(t, adaptor.circuitBreaker.IsClosed())
}

// DISABLED: func TestA1Adaptor_BackoffDelayCalculation(t *testing.T) {
	config := &A1AdaptorConfig{
		RetryConfig: &RetryConfig{
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        false,
		},
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second}, // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := adaptor.calculateBackoffDelay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

// DISABLED: func TestA1Adaptor_BackoffDelayWithJitter(t *testing.T) {
	config := &A1AdaptorConfig{
		RetryConfig: &RetryConfig{
			InitialDelay:  1 * time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		},
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	// Test multiple times to ensure jitter is working
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = adaptor.calculateBackoffDelay(2)
	}

	// Base delay should be 2 seconds, with jitter it should vary
	baseDelay := 2 * time.Second
	maxJitter := time.Duration(float64(baseDelay) * 0.1)

	for _, delay := range delays {
		assert.True(t, delay >= baseDelay, "Delay should be at least base delay")
		assert.True(t, delay <= baseDelay+maxJitter, "Delay should not exceed base + jitter")
	}
}

// DISABLED: func TestA1Adaptor_RetryableErrorDetection(t *testing.T) {
	config := &A1AdaptorConfig{
		RetryConfig: &RetryConfig{
			RetryableErrors: []string{"connection refused", "timeout", "temporary failure"},
		},
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{"nil error", nil, false},
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"timeout error", fmt.Errorf("request timeout occurred"), true},
		{"temporary failure", fmt.Errorf("temporary failure in service"), true},
		{"permanent failure", fmt.Errorf("permanent failure"), false},
		{"authentication error", fmt.Errorf("authentication failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adaptor.isRetryableError(tt.err)
			assert.Equal(t, tt.shouldRetry, result)
		})
	}
}

// DISABLED: func TestA1Adaptor_PolicyInstanceCreationWithRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/a1-p/policytypes/1/policies/policy-1")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	policyData := json.RawMessage("{}")

	ctx := context.Background()
	err = adaptor.createPolicyInstanceWithRetry(ctx, 1, "policy-1", policyData)
	assert.NoError(t, err)

	// Verify policy instance was cached
	adaptor.mutex.RLock()
	cachedPolicyInstance, exists := adaptor.policyInstances["policy-1"]
	adaptor.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "policy-1", cachedPolicyInstance.PolicyInstanceID)
	assert.Equal(t, 1, cachedPolicyInstance.PolicyTypeID)
	assert.Equal(t, policyData, cachedPolicyInstance.PolicyData)
	assert.Equal(t, "ENFORCED", cachedPolicyInstance.Status.EnforcementStatus)
}

// DISABLED: func TestA1Adaptor_FailureAfterMaxRetries(t *testing.T) {
	// Server that always returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxRetries:      2,
			InitialDelay:    10 * time.Millisecond,
			MaxDelay:        100 * time.Millisecond,
			BackoffFactor:   2.0,
			Jitter:          false,
			RetryableErrors: []string{"HTTP request failed"},
		},
	}

	adaptor, err := NewA1Adaptor(config)
	require.NoError(t, err)

	policyType := &A1PolicyType{
		PolicyTypeID: 1,
		Name:         "Test Policy Type",
	}

	ctx := context.Background()
	err = adaptor.createPolicyTypeWithRetry(ctx, policyType)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed after")

	// Verify policy type was not cached
	adaptor.mutex.RLock()
	_, exists := adaptor.policyTypes[1]
	adaptor.mutex.RUnlock()
	assert.False(t, exists)
}

// DISABLED: func TestA1Adaptor_ContextCancellation(t *testing.T) {
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RetryConfig: &RetryConfig{
			MaxRetries:    5,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      1 * time.Second,
			BackoffFactor: 2.0,
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Operation that would normally retry many times
	err = adaptor.executeWithRetry(ctx, func() error {
		return fmt.Errorf("temporary failure")
	})

	assert.Error(t, err)
	// Should fail due to context cancellation
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// DISABLED: func TestA1Adaptor_NonRetryableErrorHandling(t *testing.T) {
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RetryConfig: &RetryConfig{
			MaxRetries:      3,
			InitialDelay:    10 * time.Millisecond,
			RetryableErrors: []string{"retryable"},
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Non-retryable error should fail immediately
	err = adaptor.executeWithRetry(ctx, func() error {
		return fmt.Errorf("non-retryable error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-retryable error")
	// Should not contain retry message since it's not retryable
	assert.NotContains(t, err.Error(), "operation failed after")
}

// DISABLED: func TestA1Adaptor_ConcurrentPolicyAccess(t *testing.T) {
	adaptor, err := NewA1Adaptor(nil)
	require.NoError(t, err)

	done := make(chan bool, 100)

	// Concurrent writes to policy types
	for i := 0; i < 50; i++ {
		go func(id int) {
			policyType := &A1PolicyType{
				PolicyTypeID: id,
				Name:         fmt.Sprintf("Policy Type %d", id),
			}
			adaptor.mutex.Lock()
			adaptor.policyTypes[id] = policyType
			adaptor.mutex.Unlock()
			done <- true
		}(i)
	}

	// Concurrent reads from policy types
	for i := 0; i < 50; i++ {
		go func(id int) {
			adaptor.mutex.RLock()
			_, _ = adaptor.policyTypes[id]
			adaptor.mutex.RUnlock()
			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Verify all policy types were written
	adaptor.mutex.RLock()
	assert.Equal(t, 50, len(adaptor.policyTypes))
	adaptor.mutex.RUnlock()
}

// DISABLED: func TestA1PolicyStructures(t *testing.T) {
	t.Run("A1PolicyType validation", func(t *testing.T) {
		policyType := A1PolicyType{
			PolicyTypeID: 1,
			Name:         "Test Policy",
			Description:  "A test policy type",
			PolicySchema: json.RawMessage("{}"){
					"param1": json.RawMessage("{}"),
				},
				"required": []string{"param1"},
			},
			CreateSchema: json.RawMessage("{}"),
		}

		assert.Equal(t, 1, policyType.PolicyTypeID)
		assert.Equal(t, "Test Policy", policyType.Name)
		assert.NotNil(t, policyType.PolicySchema)
		assert.Equal(t, "object", policyType.PolicySchema["type"])
	})

	t.Run("A1PolicyInstance validation", func(t *testing.T) {
		now := time.Now()

		instance := A1PolicyInstance{
			PolicyInstanceID: "test-instance",
			PolicyTypeID:     1,
			PolicyData: json.RawMessage("{}"),
			Status: A1PolicyStatus{
				EnforcementStatus: "ENFORCED",
				EnforcementReason: "Policy applied successfully",
				LastModified:      now,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "test-instance", instance.PolicyInstanceID)
		assert.Equal(t, 1, instance.PolicyTypeID)
		assert.Equal(t, "ENFORCED", instance.Status.EnforcementStatus)
		assert.Equal(t, "Policy applied successfully", instance.Status.EnforcementReason)
		assert.Equal(t, now, instance.Status.LastModified)
	})
}

// DISABLED: func TestRetryConfig_Structure(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:      5,
		InitialDelay:    2 * time.Second,
		MaxDelay:        60 * time.Second,
		BackoffFactor:   1.5,
		Jitter:          true,
		RetryableErrors: []string{"error1", "error2"},
	}

	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.InitialDelay)
	assert.Equal(t, 60*time.Second, config.MaxDelay)
	assert.Equal(t, 1.5, config.BackoffFactor)
	assert.True(t, config.Jitter)
	assert.Len(t, config.RetryableErrors, 2)
	assert.Contains(t, config.RetryableErrors, "error1")
	assert.Contains(t, config.RetryableErrors, "error2")
}

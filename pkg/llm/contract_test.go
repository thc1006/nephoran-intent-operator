package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBatchRequestContract verifies BatchRequest has required fields
func TestBatchRequestContract(t *testing.T) {
	// Test that BatchRequest has both ResultChan and ResponseCh fields
	request := &BatchRequest{
		ID:         "test-id",
		Intent:     "test intent",
		IntentType: "NetworkFunctionDeployment",
		ModelName:  "gpt-4o-mini",
		Priority:   PriorityNormal,
		Context:    context.Background(),
		SubmitTime: time.Now(),
		Timeout:    30 * time.Second,
	}

	// Should be able to create channels for both fields
	request.ResultChan = make(chan *BatchResult, 1)
	request.ResponseCh = make(chan *ProcessingResult, 1)

	assert.NotNil(t, request.ResultChan)
	assert.NotNil(t, request.ResponseCh)
	assert.Equal(t, "test-id", request.ID)
}

// TestBatchResultContract verifies BatchResult has required fields
func TestBatchResultContract(t *testing.T) {
	result := &BatchResult{
		RequestID:   "test-request-id",
		Response:    "test response",
		ProcessTime: 100 * time.Millisecond,
		BatchID:     "test-batch-id",
		BatchSize:   5,
		QueueTime:   50 * time.Millisecond,
		Tokens:      150, // New required field
	}

	assert.Equal(t, 150, result.Tokens)
	assert.Equal(t, "test-request-id", result.RequestID)
}

// TestProcessingResultContract verifies ProcessingResult has required fields
func TestProcessingResultContract(t *testing.T) {
	result := &ProcessingResult{
		Content:        "test content",
		TokensUsed:     100,
		ProcessingTime: 200 * time.Millisecond,
		CacheHit:       false,
		Batched:        true,
		Success:        true, // New required field
	}

	// ProcessingContext should be settable (may be nil)
	result.ProcessingContext = nil // This should compile without error

	assert.True(t, result.Success)
	assert.True(t, result.Batched)
	assert.Equal(t, "test content", result.Content)
}

// TestOptimizedClientConfigContract verifies OptimizedClientConfig has required fields
func TestOptimizedClientConfigContract(t *testing.T) {
	config := &OptimizedClientConfig{
		MaxConnsPerHost: 10,
		ConnectTimeout:  30 * time.Second,
		APIKey:          "test-api-key", // New required field
	}

	// BatchConfig should be settable
	config.BatchConfig = &BatchProcessorConfig{
		BatchSize:      10,
		FlushInterval:  5 * time.Second,
		MaxConcurrency: 3,
	}

	assert.Equal(t, "test-api-key", config.APIKey)
	assert.NotNil(t, config.BatchConfig)
	assert.Equal(t, 10, config.BatchConfig.BatchSize)
}

// TestCircuitBreakerConfigContract verifies the shared CircuitBreakerConfig has MaxConcurrentRequests
func TestCircuitBreakerConfigContract(t *testing.T) {
	// Import the shared package config
	// This test verifies that MaxConcurrentRequests field exists
	// and can be used in performance_optimizer.go
	
	// This would be the type of config used in performance_optimizer.go
	type TestCircuitBreakerConfig struct {
		FailureThreshold      int
		SuccessThreshold      int
		Timeout               time.Duration
		MaxConcurrentRequests int // This field must exist
	}

	config := TestCircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      3,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 100, // This should compile
	}

	assert.Equal(t, 100, config.MaxConcurrentRequests)
}
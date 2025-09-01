package llm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_NewCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    5,
		FailureRate:         0.5,
		MinimumRequestCount: 10,
		Timeout:             30 * time.Second,
		ResetTimeout:        60 * time.Second,
		SuccessThreshold:    3,
	}

	cb := NewCircuitBreaker("test-service", config)

	assert.NotNil(t, cb)
	assert.Equal(t, "test-service", cb.name)
	assert.Equal(t, StateClosed, cb.state)
	assert.Equal(t, config, cb.config)
	assert.Equal(t, int64(0), cb.failureCount)
	assert.Equal(t, int64(0), cb.successCount)
	assert.Equal(t, int64(0), cb.requestCount)
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    3,
		FailureRate:         0.5,
		MinimumRequestCount: 5,
		Timeout:             1 * time.Second,
		ResetTimeout:        10 * time.Second,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Test successful execution
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, StateClosed, cb.getState())
	assert.Equal(t, int64(1), cb.requestCount)
	assert.Equal(t, int64(0), cb.failureCount)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    2,
		FailureRate:         0.5,
		MinimumRequestCount: 3,
		Timeout:             1 * time.Second,
		ResetTimeout:        10 * time.Second,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	testError := errors.New("test error")

	// Execute failing function
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, testError
	})

	assert.Error(t, err)
	assert.Equal(t, testError, err)
	assert.Nil(t, result)
	assert.Equal(t, StateClosed, cb.getState())
	assert.Equal(t, int64(1), cb.requestCount)
	assert.Equal(t, int64(1), cb.failureCount)
}

func TestCircuitBreaker_StateTransition_ClosedToOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    2,
		FailureRate:         0.6,
		MinimumRequestCount: 3,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	testError := errors.New("test error")

	// Execute enough failures to trigger circuit breaker
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			if i < 2 {
				return nil, testError // Fail first 2
			}
			return "success", nil // Success on 3rd
		})
	}

	// Should still be closed after 3 requests (2 failures, 1 success)
	assert.Equal(t, StateClosed, cb.getState())

	// Add one more failure to exceed threshold
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, testError
	})

	// Now should be open (3 failures out of 4 requests = 75% > 60%)
	assert.Equal(t, StateOpen, cb.getState())
}

func TestCircuitBreaker_StateTransition_OpenToHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1,
		FailureRate:         0.5,
		MinimumRequestCount: 2,
		Timeout:             1 * time.Second,
		ResetTimeout:        50 * time.Millisecond,
		SuccessThreshold:    1,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Force circuit breaker to open state
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure 1")
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure 2")
	})

	assert.Equal(t, StateOpen, cb.getState())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, StateClosed, cb.getState()) // Should close after successful half-open
}

func TestCircuitBreaker_StateTransition_HalfOpenToClosed(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1,
		FailureRate:         0.5,
		MinimumRequestCount: 2,
		Timeout:             1 * time.Second,
		ResetTimeout:        50 * time.Millisecond,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Force to open state
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure 1")
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure 2")
	})

	assert.Equal(t, StateOpen, cb.getState())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// First success should transition to half-open
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success1", nil
	})
	assert.Equal(t, StateHalfOpen, cb.getState())

	// Second success should close the circuit
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success2", nil
	})
	assert.Equal(t, StateClosed, cb.getState())
}

func TestCircuitBreaker_StateTransition_HalfOpenToOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1,
		FailureRate:         0.5,
		MinimumRequestCount: 2,
		Timeout:             1 * time.Second,
		ResetTimeout:        50 * time.Millisecond,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Force to open state
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})

	assert.Equal(t, StateOpen, cb.getState())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open should go back to open
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("half-open failure")
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, StateOpen, cb.getState())
}

func TestCircuitBreaker_OpenState_RejectsRequests(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1,
		FailureRate:         0.5,
		MinimumRequestCount: 2,
		Timeout:             1 * time.Second,
		ResetTimeout:        1 * time.Hour, // Long timeout to keep open
		SuccessThreshold:    1,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Force to open state
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})

	assert.Equal(t, StateOpen, cb.getState())

	// Request should be rejected
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		t.Fatal("Function should not be called when circuit is open")
		return nil, nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.Nil(t, result)
}

func TestCircuitBreaker_Timeout(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    5,
		FailureRate:         0.5,
		MinimumRequestCount: 10,
		Timeout:             100 * time.Millisecond,
		ResetTimeout:        1 * time.Second,
		SuccessThreshold:    3,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Execute function that takes longer than timeout
	start := time.Now()
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return "should timeout", nil
	})

	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Nil(t, result)
	assert.True(t, duration < 150*time.Millisecond, "Should timeout before 150ms")
	assert.Equal(t, int64(1), cb.failureCount) // Timeout counts as failure
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    10,
		FailureRate:         0.8,
		MinimumRequestCount: 20,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		SuccessThreshold:    5,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	var wg sync.WaitGroup
	successCount := int64(0)
	errorCount := int64(0)
	var successMutex sync.Mutex
	var errorMutex sync.Mutex

	// Run 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			_, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				time.Sleep(1 * time.Millisecond) // Small delay
				if i%5 == 0 {                    // 20% failure rate
					return nil, errors.New("concurrent failure")
				}
				return "success", nil
			})

			if err != nil {
				errorMutex.Lock()
				errorCount++
				errorMutex.Unlock()
			} else {
				successMutex.Lock()
				successCount++
				successMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify that concurrent access doesn't corrupt state
	assert.Equal(t, int64(100), cb.requestCount)
	assert.True(t, successCount > 0, "Should have some successes")
	assert.True(t, errorCount > 0, "Should have some errors")
	assert.Equal(t, int64(100), successCount+errorCount)

	// Circuit should still be closed (20% failure rate < 80% threshold)
	assert.Equal(t, StateClosed, cb.getState())
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    3,
		FailureRate:         0.5,
		MinimumRequestCount: 5,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		SuccessThreshold:    2,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Execute mixed success/failure requests
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})

	metrics := cb.GetMetrics()

	assert.Equal(t, "Closed", metrics.CurrentState)
	assert.Equal(t, int64(2), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.FailedRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.Equal(t, 0.5, metrics.FailureRate)
	assert.True(t, metrics.LastStateChange.Before(time.Now()))
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1,
		FailureRate:         0.5,
		MinimumRequestCount: 2,
		Timeout:             1 * time.Second,
		ResetTimeout:        1 * time.Hour,
		SuccessThreshold:    1,
	}

	cb := NewCircuitBreaker("test-service", config)
	ctx := context.Background()

	// Force to open state
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})
	cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})

	assert.Equal(t, StateOpen, cb.getState())

	// Reset circuit breaker
	cb.Reset()

	assert.Equal(t, StateClosed, cb.getState())
	assert.Equal(t, int64(0), cb.requestCount)
	assert.Equal(t, int64(0), cb.failureCount)
	assert.Equal(t, int64(0), cb.successCount)

	// Should be able to execute successfully after reset
	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return "reset success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "reset success", result)
}

func TestCircuitBreakerManager(t *testing.T) {
	mgr := NewCircuitBreakerManager(nil)

	// Get or create circuit breaker
	cb1 := mgr.GetOrCreate("service1", nil)
	assert.NotNil(t, cb1)
	assert.Equal(t, "service1", cb1.name)

	// Get same circuit breaker again
	cb2 := mgr.GetOrCreate("service1", nil)
	assert.Equal(t, cb1, cb2) // Should be same instance

	// Get different circuit breaker
	cb3 := mgr.GetOrCreate("service2", nil)
	assert.NotNil(t, cb3)
	assert.NotEqual(t, cb1, cb3)
	assert.Equal(t, "service2", cb3.name)

	// Test status retrieval
	status := mgr.GetAllStats()
	assert.Len(t, status, 2)
	assert.Contains(t, status, "service1")
	assert.Contains(t, status, "service2")
}

// Benchmark tests
func BenchmarkCircuitBreaker_Execute_Success(b *testing.B) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    10,
		FailureRate:         0.5,
		MinimumRequestCount: 20,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		SuccessThreshold:    5,
	}

	cb := NewCircuitBreaker("benchmark", config)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return "success", nil
			})
		}
	})
}

func BenchmarkCircuitBreaker_Execute_Failure(b *testing.B) {
	config := &CircuitBreakerConfig{
		FailureThreshold:    1000,
		FailureRate:         0.9,
		MinimumRequestCount: 2000,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		SuccessThreshold:    10,
	}

	cb := NewCircuitBreaker("benchmark", config)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return nil, errors.New("benchmark failure")
			})
		}
	})
}

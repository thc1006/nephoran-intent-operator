package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// TestEnhancedPerformanceClientIntegration tests the complete integration of all performance components
func TestEnhancedPerformanceClientIntegration(t *testing.T) {
	// Create enhanced client with custom configuration
	config := &EnhancedClientConfig{
		BaseConfig: ClientConfig{
			ModelName:   "test-model",
			MaxTokens:   1024,
			BackendType: "test",
			Timeout:     30 * time.Second,
		},
		PerformanceConfig: &PerformanceConfig{
			LatencyBufferSize:    100,
			OptimizationInterval: 10 * time.Second,
			EnableTracing:        true,
			TraceSamplingRatio:   1.0, // Full sampling for tests
		},
		RetryConfig: RetryEngineConfig{
			DefaultStrategy:   "exponential",
			MaxRetries:        3,
			BaseDelay:         100 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			JitterEnabled:     false, // Predictable for tests
			BackoffMultiplier: 2.0,
			EnableAdaptive:    true,
		},
		BatchConfig: BatchConfig{
			MaxBatchSize:         5,
			BatchTimeout:         50 * time.Millisecond,
			ConcurrentBatches:    2,
			EnablePrioritization: true,
		},
		CircuitBreakerConfig: shared.CircuitBreakerConfig{
			FailureThreshold:      3,
			SuccessThreshold:      2,
			Timeout:               1 * time.Second,
			MaxConcurrentRequests: 10,
			EnableAdaptiveTimeout: false, // Predictable for tests
		},
		TracingConfig: TracingConfig{
			Enabled:        true,
			SamplingRatio:  1.0,
			ServiceName:    "test-llm-client",
			ServiceVersion: "test",
		},
		MetricsConfig: MetricsConfig{
			Enabled:        true,
			ExportInterval: 1 * time.Second,
		},
		HealthCheckConfig: HealthCheckConfig{
			Enabled:          false, // Disable for test simplicity
			Interval:         1 * time.Second,
			Timeout:          500 * time.Millisecond,
			FailureThreshold: 2,
		},
		TokenConfig: TokenConfig{
			TrackUsage:     true,
			TrackCosts:     true,
			CostPerToken:   map[string]float64{"test-model": 0.001},
			BudgetLimit:    10.0,
			AlertThreshold: 80.0,
		},
	}

	client, err := NewEnhancedPerformanceClient(config)
	require.NoError(t, err)
	defer client.Close()

	// Test basic functionality
	t.Run("BasicProcessing", func(t *testing.T) {
		ctx := context.Background()

		response, err := client.ProcessIntent(ctx, "deploy nginx with 3 replicas")
		assert.NoError(t, err)
		assert.NotEmpty(t, response)

		// Verify metrics were recorded
		metrics := client.GetMetrics()
		assert.NotNil(t, metrics["performance"])
		assert.NotNil(t, metrics["token_usage"])
		assert.NotNil(t, metrics["cost_tracking"])
	})

	// Test batch processing
	t.Run("BatchProcessing", func(t *testing.T) {
		ctx := context.Background()

		options := &IntentProcessingOptions{
			Intent:     "scale deployment to 5 replicas",
			IntentType: "NetworkFunctionScale",
			ModelName:  "test-model",
			Priority:   PriorityHigh,
			UseBatch:   true,
			UseCache:   false,
		}

		response, err := client.ProcessIntentWithOptions(ctx, options)
		assert.NoError(t, err)
		assert.NotEmpty(t, response)

		// Check batch processor stats
		batchStats := client.batchProcessor.GetStats()
		assert.Greater(t, batchStats.TotalRequests, int64(0))
	})

	// Test concurrent processing
	t.Run("ConcurrentProcessing", func(t *testing.T) {
		ctx := context.Background()
		concurrency := 10

		var wg sync.WaitGroup
		results := make([]string, concurrency)
		errors := make([]error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				intent := fmt.Sprintf("process request %d", index)
				response, err := client.ProcessIntent(ctx, intent)
				results[index] = response
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// Verify all requests completed
		for i := 0; i < concurrency; i++ {
			assert.NoError(t, errors[i], "Request %d failed", i)
			assert.NotEmpty(t, results[i], "Request %d returned empty response", i)
		}

		// Check performance metrics
		profile := client.performanceOpt.GetLatencyProfile()
		assert.Greater(t, profile.TotalRequests, 0)
		assert.True(t, profile.SuccessRate > 0.8) // Should have high success rate
	})

	// Test retry mechanism
	t.Run("RetryMechanism", func(t *testing.T) {
		ctx := context.Background()

		// Test with different retry strategies
		strategies := []string{"exponential", "linear", "fixed"}

		for _, strategy := range strategies {
			t.Run(fmt.Sprintf("Strategy_%s", strategy), func(t *testing.T) {
				result, err := client.retryEngine.ExecuteWithRetryAndStrategy(ctx, func() error {
					// Simulate operation that succeeds on second attempt
					static := &struct{ attempts int }{}
					static.attempts++
					if static.attempts < 2 {
						return fmt.Errorf("simulated failure")
					}
					return nil
				}, strategy)

				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
				assert.Greater(t, result.TotalAttempts, 1)
			})
		}
	})

	// Test circuit breaker
	t.Run("CircuitBreaker", func(t *testing.T) {
		ctx := context.Background()

		// Force circuit breaker to open
		for i := 0; i < 5; i++ {
			client.circuitBreaker.Execute(ctx, func() error {
				return fmt.Errorf("simulated failure")
			})
		}

		// Verify circuit breaker is open
		stats := client.circuitBreaker.GetStats()
		assert.Equal(t, int32(StateOpen), stats.State)

		// Try to execute - should be rejected
		err := client.circuitBreaker.Execute(ctx, func() error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker")

		// Reset circuit breaker
		client.circuitBreaker.Reset()

		// Should work now
		err = client.circuitBreaker.Execute(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	// Test performance profiling
	t.Run("PerformanceProfiling", func(t *testing.T) {
		ctx := context.Background()

		// Generate some load
		for i := 0; i < 20; i++ {
			intent := fmt.Sprintf("test intent %d", i)
			client.ProcessIntent(ctx, intent)
		}

		// Get latency profile
		profile := client.performanceOpt.GetLatencyProfile()

		assert.Greater(t, profile.TotalRequests, 0)
		assert.NotEmpty(t, profile.ByIntentType)
		assert.NotEmpty(t, profile.ByModelName)
		assert.True(t, profile.Percentiles.P50 > 0)
		assert.True(t, profile.Percentiles.P95 > 0)
		assert.True(t, profile.Percentiles.P99 > 0)
	})

	// Test cost tracking
	t.Run("CostTracking", func(t *testing.T) {
		ctx := context.Background()

		// Process some requests to generate token usage
		for i := 0; i < 5; i++ {
			client.ProcessIntent(ctx, "test intent for cost tracking")
		}

		costStats := client.getCostTrackingStats()
		assert.Greater(t, costStats["total_cost"].(float64), 0.0)
		assert.NotEmpty(t, costStats["costs_by_model"])

		tokenStats := client.getTokenUsageStats()
		assert.Greater(t, tokenStats["total_tokens"].(int64), int64(0))
		assert.NotEmpty(t, tokenStats["tokens_by_model"])
	})
}

// TestPerformanceOptimizerStandalone tests the performance optimizer in isolation
func TestPerformanceOptimizerStandalone(t *testing.T) {
	config := getDefaultPerformanceConfig()
	config.OptimizationInterval = 100 * time.Millisecond // Fast for testing

	optimizer := NewPerformanceOptimizer(config)
	defer optimizer.Close()

	// Record some latency data points
	dataPoints := []LatencyDataPoint{
		{
			Timestamp:    time.Now(),
			Duration:     100 * time.Millisecond,
			IntentType:   "NetworkFunctionDeployment",
			ModelName:    "test-model",
			TokenCount:   500,
			Success:      true,
			RequestSize:  100,
			ResponseSize: 200,
			CacheHit:     false,
		},
		{
			Timestamp:    time.Now(),
			Duration:     200 * time.Millisecond,
			IntentType:   "NetworkFunctionScale",
			ModelName:    "test-model",
			TokenCount:   300,
			Success:      true,
			RequestSize:  80,
			ResponseSize: 150,
			CacheHit:     true,
		},
		{
			Timestamp:    time.Now(),
			Duration:     500 * time.Millisecond,
			IntentType:   "NetworkFunctionDeployment",
			ModelName:    "test-model",
			TokenCount:   0,
			Success:      false,
			ErrorType:    "timeout",
			RequestSize:  120,
			ResponseSize: 0,
			CacheHit:     false,
		},
	}

	for _, dp := range dataPoints {
		optimizer.RecordLatency(dp)
	}

	// Wait for optimization cycle
	time.Sleep(200 * time.Millisecond)

	// Get latency profile
	profile := optimizer.GetLatencyProfile()

	assert.Equal(t, len(dataPoints), profile.TotalRequests)
	assert.Equal(t, 2.0/3.0, profile.SuccessRate) // 2 successful out of 3
	assert.NotEmpty(t, profile.ByIntentType)
	assert.NotEmpty(t, profile.ByModelName)

	// Check intent type breakdown
	deploymentProfile := profile.ByIntentType["NetworkFunctionDeployment"]
	assert.NotNil(t, deploymentProfile)
	assert.Equal(t, 2, deploymentProfile.Count)
	assert.Equal(t, 0.5, deploymentProfile.SuccessRate) // 1 success out of 2

	scaleProfile := profile.ByIntentType["NetworkFunctionScale"]
	assert.NotNil(t, scaleProfile)
	assert.Equal(t, 1, scaleProfile.Count)
	assert.Equal(t, 1.0, scaleProfile.SuccessRate) // 1 success out of 1

	// Check model breakdown
	modelProfile := profile.ByModelName["test-model"]
	assert.NotNil(t, modelProfile)
	assert.Equal(t, 3, modelProfile.Count)
	assert.InDelta(t, 266.67, modelProfile.AvgTokenCount, 1.0) // (500+300+0)/3
}

// TestBatchProcessorStandalone tests the batch processor in isolation
func TestBatchProcessorStandalone(t *testing.T) {
	config := BatchConfig{
		MaxBatchSize:         3,
		BatchTimeout:         50 * time.Millisecond,
		ConcurrentBatches:    2,
		EnablePrioritization: true,
	}

	processor := NewBatchProcessor(config)
	defer processor.Close()

	ctx := context.Background()

	// Test normal priority request
	result, err := processor.ProcessRequest(ctx, "test intent", "NetworkFunctionDeployment", "test-model", PriorityNormal)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test intent", result.RequestID) // Simplified for test
	assert.Greater(t, result.ProcessTime, time.Duration(0))

	// Test high priority request
	result, err = processor.ProcessRequest(ctx, "urgent intent", "NetworkFunctionScale", "test-model", PriorityHigh)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Check stats
	stats := processor.GetStats()
	assert.Greater(t, stats.TotalRequests, int64(0))
	assert.Greater(t, stats.ProcessedBatches, int64(0))
}

// TestAdvancedCircuitBreakerStandalone tests the advanced circuit breaker in isolation
func TestAdvancedCircuitBreakerStandalone(t *testing.T) {
	config := shared.CircuitBreakerConfig{
		FailureThreshold:      2,
		SuccessThreshold:      2,
		Timeout:               100 * time.Millisecond,
		MaxConcurrentRequests: 5,
		EnableAdaptiveTimeout: false,
	}

	cb := NewAdvancedCircuitBreaker(config)
	ctx := context.Background()

	// Initially closed
	assert.Equal(t, int32(StateClosed), cb.GetState())

	// Successful execution
	err := cb.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, int32(StateClosed), cb.GetState())

	// Failures to open circuit
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func() error { return fmt.Errorf("failure %d", i) })
	}
	assert.Equal(t, int32(StateOpen), cb.GetState())

	// Rejected execution
	err = cb.Execute(ctx, func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open and allow execution
	err = cb.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)

	// Another success should close the circuit
	err = cb.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, int32(StateClosed), cb.GetState())

	// Check stats
	stats := cb.GetStats()
	assert.Greater(t, stats.TotalRequests, int64(0))
	assert.Greater(t, stats.StateChanges, int64(0))
	assert.Greater(t, stats.TotalFailures, int64(0))
}

// TestRetryEngineStandalone tests the retry engine in isolation
func TestRetryEngineStandalone(t *testing.T) {
	config := RetryEngineConfig{
		DefaultStrategy:   "exponential",
		MaxRetries:        3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		JitterEnabled:     false,
		BackoffMultiplier: 2.0,
		EnableAdaptive:    false,
	}

	retryEngine := NewRetryEngine(config, nil) // No circuit breaker for this test
	ctx := context.Background()

	// Test successful retry
	attempts := 0
	result, err := retryEngine.ExecuteWithRetry(ctx, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("attempt %d failed", attempts)
		}
		return nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.TotalAttempts)
	assert.Greater(t, result.TotalDuration, time.Duration(0))

	// Test failure after all retries
	attempts = 0
	result, err = retryEngine.ExecuteWithRetry(ctx, func() error {
		attempts++
		return fmt.Errorf("persistent failure")
	})

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, 4, result.TotalAttempts) // Initial + 3 retries

	// Check metrics
	metrics := retryEngine.GetMetrics()
	assert.Greater(t, metrics.TotalRetries, int64(0))
	assert.Greater(t, metrics.FailedRetries, int64(0))
	assert.Greater(t, metrics.SuccessfulRetries, int64(0))
}

// BenchmarkEnhancedPerformanceClient benchmarks the enhanced client
func BenchmarkEnhancedPerformanceClient(b *testing.B) {
	config := getDefaultEnhancedConfig()
	config.MetricsConfig.Enabled = false // Disable metrics for pure performance
	config.TracingConfig.Enabled = false // Disable tracing for pure performance

	client, err := NewEnhancedPerformanceClient(config)
	require.NoError(b, err)
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.ProcessIntent(ctx, "benchmark test intent")
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkBatchProcessing benchmarks batch processing specifically
func BenchmarkBatchProcessing(b *testing.B) {
	config := BatchConfig{
		MaxBatchSize:         10,
		BatchTimeout:         10 * time.Millisecond,
		ConcurrentBatches:    4,
		EnablePrioritization: false,
	}

	processor := NewBatchProcessor(config)
	defer processor.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := processor.ProcessRequest(ctx, "benchmark intent", "NetworkFunctionDeployment", "test-model", PriorityNormal)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// Example test demonstrating usage patterns
func ExampleEnhancedPerformanceClient() {
	// Create client with custom configuration
	config := &EnhancedClientConfig{
		BaseConfig: ClientConfig{
			ModelName:   "gpt-4o-mini",
			MaxTokens:   2048,
			BackendType: "openai",
			Timeout:     60 * time.Second,
		},
		// Enable all performance features
		PerformanceConfig: &PerformanceConfig{
			LatencyBufferSize:    10000,
			OptimizationInterval: time.Minute,
			EnableTracing:        true,
		},
		RetryConfig: RetryEngineConfig{
			DefaultStrategy: "adaptive",
			MaxRetries:      5,
			EnableAdaptive:  true,
		},
		BatchConfig: BatchConfig{
			MaxBatchSize:         10,
			EnablePrioritization: true,
		},
		TracingConfig: TracingConfig{
			Enabled:       true,
			SamplingRatio: 0.1,
		},
		TokenConfig: TokenConfig{
			TrackUsage:     true,
			TrackCosts:     true,
			BudgetLimit:    100.0,
			AlertThreshold: 80.0,
		},
	}

	client, err := NewEnhancedPerformanceClient(config)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Basic usage
	response, err := client.ProcessIntent(ctx, "Deploy nginx with 3 replicas in production")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Response: %s\n", response)

	// Advanced usage with options
	options := &IntentProcessingOptions{
		Intent:     "Scale the frontend service to handle more traffic",
		IntentType: "NetworkFunctionScale",
		ModelName:  "gpt-4o-mini",
		Priority:   PriorityHigh,
		UseBatch:   true,
		UseCache:   true,
	}

	response, err = client.ProcessIntentWithOptions(ctx, options)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Advanced Response: %s\n", response)

	// Get comprehensive metrics
	metrics := client.GetMetrics()
	fmt.Printf("Performance metrics: %+v\n", metrics["performance"])
	fmt.Printf("Cost tracking: %+v\n", metrics["cost_tracking"])

	// Get health status
	health := client.GetHealthStatus()
	fmt.Printf("Health status: %+v\n", health)
}

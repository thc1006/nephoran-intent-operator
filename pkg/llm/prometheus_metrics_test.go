package llm

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMetricsIntegration(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("METRICS_ENABLED")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("METRICS_ENABLED")
		} else {
			os.Setenv("METRICS_ENABLED", originalEnv)
		}
	}()

	t.Run("metrics disabled by default", func(t *testing.T) {
		os.Unsetenv("METRICS_ENABLED")
		
		pm := NewPrometheusMetrics()
		assert.NotNil(t, pm)
		assert.False(t, pm.isRegistered())
	})

	t.Run("metrics enabled when METRICS_ENABLED=true", func(t *testing.T) {
		os.Setenv("METRICS_ENABLED", "true")
		
		// Reset singleton for test
		prometheusOnce = sync.Once{}
		prometheusMetrics = nil
		
		pm := NewPrometheusMetrics()
		assert.NotNil(t, pm)
		assert.True(t, pm.isRegistered())
	})

	t.Run("prometheus metrics recording", func(t *testing.T) {
		os.Setenv("METRICS_ENABLED", "true")
		
		// Reset singleton for test
		prometheusOnce = sync.Once{}
		prometheusMetrics = nil
		
		pm := NewPrometheusMetrics()
		require.True(t, pm.isRegistered())
		
		// Record some metrics
		pm.RecordRequest("gpt-4o-mini", "success", 500*time.Millisecond)
		pm.RecordRequest("gpt-4o-mini", "error", 200*time.Millisecond)
		pm.RecordCacheHit("gpt-4o-mini")
		pm.RecordCacheMiss("gpt-4o-mini")
		pm.RecordRetryAttempt("gpt-4o-mini")
		pm.RecordFallbackAttempt("gpt-4o-mini", "gpt-3.5-turbo")
		pm.RecordError("gpt-4o-mini", "timeout")
		
		// Verify metrics were recorded (basic checks)
		assert.NotNil(t, pm.RequestsTotal)
		assert.NotNil(t, pm.ErrorsTotal)
		assert.NotNil(t, pm.ProcessingDurationSeconds)
	})
}

func TestMetricsIntegrator(t *testing.T) {
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")
	
	// Reset singleton for test
	prometheusOnce = sync.Once{}
	prometheusMetrics = nil
	
	collector := NewMetricsCollector()
	integrator := NewMetricsIntegrator(collector)
	
	assert.NotNil(t, integrator)
	assert.NotNil(t, integrator.collector)
	assert.NotNil(t, integrator.prometheusMetrics)
	
	// Test integrated recording
	integrator.RecordLLMRequest("gpt-4o-mini", "success", 500*time.Millisecond, 150)
	integrator.RecordCacheOperation("gpt-4o-mini", "get", true)
	integrator.RecordRetryAttempt("gpt-4o-mini")
	integrator.RecordFallbackAttempt("gpt-4o-mini", "gpt-3.5-turbo")
	integrator.RecordCircuitBreakerEvent("llm-client", "rejected", "gpt-4o-mini")
	
	// Verify both systems recorded metrics
	clientMetrics := collector.GetClientMetrics()
	assert.Greater(t, clientMetrics.RequestsTotal, int64(0))
	assert.Greater(t, clientMetrics.CacheHits, int64(0))
	assert.Greater(t, clientMetrics.RetryAttempts, int64(0))
	assert.Greater(t, clientMetrics.FallbackAttempts, int64(0))
}

func TestErrorCategorization(t *testing.T) {
	client := &Client{
		modelName: "test-model",
	}
	
	testCases := []struct {
		name           string
		error          error
		expectedType   string
	}{
		{
			name:         "circuit breaker error",
			error:        fmt.Errorf("circuit breaker is open"),
			expectedType: "circuit_breaker_open",
		},
		{
			name:         "timeout error",
			error:        fmt.Errorf("request timeout"),
			expectedType: "timeout",
		},
		{
			name:         "context deadline",
			error:        fmt.Errorf("context deadline exceeded"),
			expectedType: "timeout",
		},
		{
			name:         "connection refused",
			error:        fmt.Errorf("connection refused"),
			expectedType: "connection_refused",
		},
		{
			name:         "authentication error",
			error:        fmt.Errorf("HTTP 401 unauthorized"),
			expectedType: "authentication_error",
		},
		{
			name:         "rate limit error",
			error:        fmt.Errorf("HTTP 429 too many requests"),
			expectedType: "rate_limit_exceeded",
		},
		{
			name:         "server error",
			error:        fmt.Errorf("HTTP 500 internal server error"),
			expectedType: "server_error",
		},
		{
			name:         "unknown error",
			error:        fmt.Errorf("some unknown error"),
			expectedType: "unknown_error",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := client.categorizeError(tc.error)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

func TestMetricsEnvironmentVariableGating(t *testing.T) {
	// Test with metrics disabled
	os.Setenv("METRICS_ENABLED", "false")
	assert.False(t, isMetricsEnabled())
	
	// Test with metrics enabled
	os.Setenv("METRICS_ENABLED", "true")
	assert.True(t, isMetricsEnabled())
	
	// Test with unset variable (should be false)
	os.Unsetenv("METRICS_ENABLED")
	assert.False(t, isMetricsEnabled())
}

// Helper function to get metric value (for more advanced testing if needed)
func getCounterValue(counter prometheus.Counter) float64 {
	metric := &dto.Metric{}
	counter.Write(metric)
	return metric.GetCounter().GetValue()
}

// Helper function to get histogram count
func getHistogramCount(histogram prometheus.Histogram) uint64 {
	metric := &dto.Metric{}
	histogram.Write(metric)
	return metric.GetHistogram().GetSampleCount()
}
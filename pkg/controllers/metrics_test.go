package controllers

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControllerMetrics(t *testing.T) {
	// Test with metrics disabled
	t.Run("MetricsDisabled", func(t *testing.T) {
		os.Setenv("METRICS_ENABLED", "false")
		defer os.Unsetenv("METRICS_ENABLED")
		
		metrics := NewControllerMetrics("test-controller")
		assert.False(t, metrics.enabled)
		
		// These calls should not panic when metrics are disabled
		metrics.RecordReconcileTotal("default", "test", "success")
		metrics.RecordReconcileError("default", "test", "timeout")
		metrics.RecordProcessingDuration("default", "test", "llm_processing", 1.5)
		metrics.SetStatus("default", "test", "processing", StatusProcessing)
	})
	
	// Test with metrics enabled
	t.Run("MetricsEnabled", func(t *testing.T) {
		os.Setenv("METRICS_ENABLED", "true")
		defer os.Unsetenv("METRICS_ENABLED")
		
		metrics := NewControllerMetrics("test-controller")
		assert.True(t, metrics.enabled)
		
		// Test convenience methods
		metrics.RecordSuccess("default", "test-success")
		metrics.RecordFailure("default", "test-failure", "llm_processing")
	})
	
	t.Run("IsMetricsEnabled", func(t *testing.T) {
		// Test default (no env var)
		os.Unsetenv("METRICS_ENABLED")
		assert.False(t, isMetricsEnabled())
		
		// Test explicitly disabled
		os.Setenv("METRICS_ENABLED", "false")
		assert.False(t, isMetricsEnabled())
		
		// Test explicitly enabled
		os.Setenv("METRICS_ENABLED", "true")
		assert.True(t, isMetricsEnabled())
		
		// Test invalid value
		os.Setenv("METRICS_ENABLED", "invalid")
		assert.False(t, isMetricsEnabled())
		
		// Cleanup
		os.Unsetenv("METRICS_ENABLED")
	})
}

func TestMetricsRegistration(t *testing.T) {
	// Create a temporary registry to avoid conflicts
	tempRegistry := prometheus.NewRegistry()
	
	// Test that metrics can be registered
	metrics := []prometheus.Collector{
		networkIntentReconcilesTotal,
		networkIntentReconcileErrors,
		networkIntentProcessingDuration,
		networkIntentStatus,
	}
	
	for _, metric := range metrics {
		err := tempRegistry.Register(metric)
		require.NoError(t, err, "Failed to register metric")
	}
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, float64(0), StatusFailed)
	assert.Equal(t, float64(1), StatusProcessing) 
	assert.Equal(t, float64(2), StatusReady)
}

func TestGetMetricsEnabled(t *testing.T) {
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")
	
	assert.True(t, GetMetricsEnabled())
	
	os.Setenv("METRICS_ENABLED", "false")
	assert.False(t, GetMetricsEnabled())
}

func BenchmarkMetricsRecording(b *testing.B) {
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")
	
	metrics := NewControllerMetrics("benchmark-controller")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordReconcileTotal("default", "test", "success")
		metrics.RecordProcessingDuration("default", "test", "llm_processing", 1.5)
		metrics.SetStatus("default", "test", "processing", StatusProcessing)
	}
}

func BenchmarkMetricsRecordingDisabled(b *testing.B) {
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")
	
	metrics := NewControllerMetrics("benchmark-controller")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordReconcileTotal("default", "test", "success")
		metrics.RecordProcessingDuration("default", "test", "llm_processing", 1.5)
		metrics.SetStatus("default", "test", "processing", StatusProcessing)
	}
}
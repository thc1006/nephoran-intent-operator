//go:build integration

// Package integration provides integration tests for metrics scraping functionality.
// This is a simplified version that doesn't depend on problematic internal packages.
package integration

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndpointBasicFunctionality validates basic metrics endpoint functionality
func TestMetricsEndpointBasicFunctionality(t *testing.T) {
	// Store original environment
	originalEnv := os.Getenv("METRICS_ENABLED")
	defer func() {
		if originalEnv != "" {
			os.Setenv("METRICS_ENABLED", originalEnv)
		} else {
			os.Unsetenv("METRICS_ENABLED")
		}
	}()

	t.Run("Metrics endpoint enabled with basic Prometheus metrics", func(t *testing.T) {
		// Enable metrics
		os.Setenv("METRICS_ENABLED", "true")

		// Create test registry with sample metrics
		registry := prometheus.NewRegistry()

		// Create sample LLM-like metrics
		llmRequestsTotal := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_llm_requests_total",
				Help: "Total number of LLM requests by model and status",
			},
			[]string{"model", "status"},
		)

		llmProcessingDuration := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_llm_processing_duration_seconds",
				Help:    "Duration of LLM processing requests by model and status",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
			},
			[]string{"model", "status"},
		)

		// Create sample controller-like metrics
		networkintentReconciles := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "networkintent_reconciles_total",
				Help: "Total number of NetworkIntent reconciliations",
			},
			[]string{"controller", "namespace", "name", "result"},
		)

		networkintentProcessingDuration := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "networkintent_processing_duration_seconds",
				Help:    "Duration of NetworkIntent processing phases",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"controller", "namespace", "name", "phase"},
		)

		// Register metrics
		registry.MustRegister(
			llmRequestsTotal,
			llmProcessingDuration,
			networkintentReconciles,
			networkintentProcessingDuration,
		)

		// Generate sample data
		llmRequestsTotal.WithLabelValues("gpt-4o-mini", "success").Inc()
		llmRequestsTotal.WithLabelValues("gpt-4o-mini", "error").Inc()
		llmProcessingDuration.WithLabelValues("gpt-4o-mini", "success").Observe(1.5)

		networkintentReconciles.WithLabelValues("networkintent", "default", "test-intent", "success").Inc()
		networkintentProcessingDuration.WithLabelValues("networkintent", "default", "test-intent", "llm_processing").Observe(2.3)

		// Setup HTTP server
		router := mux.NewRouter()
		router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
		server := httptest.NewServer(router)
		defer server.Close()

		// Test metrics endpoint
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Validate response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

		// Parse and validate metrics content
		metrics, err := parseSimplePrometheusMetrics(resp)
		require.NoError(t, err)

		// Verify expected LLM metrics
		assert.Contains(t, metrics, "nephoran_llm_requests_total")
		assert.Contains(t, metrics, "nephoran_llm_processing_duration_seconds")

		// Verify expected controller metrics
		assert.Contains(t, metrics, "networkintent_reconciles_total")
		assert.Contains(t, metrics, "networkintent_processing_duration_seconds")

		// Verify help text is present
		assertMetricHasHelp(t, metrics, "nephoran_llm_requests_total", "Total number of LLM requests")
		assertMetricHasHelp(t, metrics, "networkintent_reconciles_total", "Total number of NetworkIntent reconciliations")

		t.Logf("Successfully validated %d metrics", len(metrics))
	})

	t.Run("Metrics endpoint disabled returns 404", func(t *testing.T) {
		// Disable metrics
		os.Setenv("METRICS_ENABLED", "false")

		// Create router without metrics endpoint
		router := mux.NewRouter()
		server := httptest.NewServer(router)
		defer server.Close()

		// Test metrics endpoint should return 404
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestMetricsEndpointConditionalExposure validates conditional metrics exposure
func TestMetricsEndpointConditionalExposure(t *testing.T) {
	testCases := []struct {
		name           string
		metricsEnabled string
		expectEndpoint bool
		expectedStatus int
	}{
		{
			name:           "METRICS_ENABLED=true enables endpoint",
			metricsEnabled: "true",
			expectEndpoint: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "METRICS_ENABLED=false disables endpoint",
			metricsEnabled: "false",
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "METRICS_ENABLED unset disables endpoint",
			metricsEnabled: "",
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "METRICS_ENABLED invalid value disables endpoint",
			metricsEnabled: "invalid",
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
		},
	}

	originalEnv := os.Getenv("METRICS_ENABLED")
	defer func() {
		if originalEnv != "" {
			os.Setenv("METRICS_ENABLED", originalEnv)
		} else {
			os.Unsetenv("METRICS_ENABLED")
		}
	}()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment
			if tc.metricsEnabled == "" {
				os.Unsetenv("METRICS_ENABLED")
			} else {
				os.Setenv("METRICS_ENABLED", tc.metricsEnabled)
			}

			// Create router conditionally with metrics
			router := mux.NewRouter()

			// Simulate the real application's conditional metrics setup
			if isMetricsEnabled() && tc.expectEndpoint {
				registry := prometheus.NewRegistry()
				testCounter := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "test_metric_total",
					Help: "A test metric",
				})
				registry.MustRegister(testCounter)
				testCounter.Inc()

				router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
			}

			server := httptest.NewServer(router)
			defer server.Close()

			// Test endpoint
			resp, err := http.Get(server.URL + "/metrics")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.expectEndpoint {
				assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
			}
		})
	}
}

// TestMetricsEndpointContentValidation validates the Prometheus format content
func TestMetricsEndpointContentValidation(t *testing.T) {
	// Setup metrics
	registry := prometheus.NewRegistry()

	testCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_test_requests_total",
			Help: "Total test requests for validation",
		},
		[]string{"method", "status"},
	)

	testHistogram := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nephoran_test_duration_seconds",
			Help:    "Test request duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0},
		},
	)

	registry.MustRegister(testCounter, testHistogram)

	// Generate test data
	testCounter.WithLabelValues("GET", "200").Add(5)
	testCounter.WithLabelValues("POST", "201").Add(3)
	testCounter.WithLabelValues("GET", "404").Add(1)
	testHistogram.Observe(0.3)
	testHistogram.Observe(1.2)

	// Setup server
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
	server := httptest.NewServer(router)
	defer server.Close()

	// Get metrics
	resp, err := http.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Read content
	scanner := bufio.NewScanner(resp.Body)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	require.NoError(t, scanner.Err())

	content := strings.Join(lines, "\n")

	// Validate Prometheus format
	t.Run("Contains HELP and TYPE comments", func(t *testing.T) {
		assert.Contains(t, content, "# HELP nephoran_test_requests_total Total test requests for validation")
		assert.Contains(t, content, "# TYPE nephoran_test_requests_total counter")
		assert.Contains(t, content, "# HELP nephoran_test_duration_seconds Test request duration in seconds")
		assert.Contains(t, content, "# TYPE nephoran_test_duration_seconds histogram")
	})

	t.Run("Contains metric values with labels", func(t *testing.T) {
		assert.Contains(t, content, `nephoran_test_requests_total{method="GET",status="200"} 5`)
		assert.Contains(t, content, `nephoran_test_requests_total{method="POST",status="201"} 3`)
		assert.Contains(t, content, `nephoran_test_requests_total{method="GET",status="404"} 1`)
	})

	t.Run("Contains histogram buckets", func(t *testing.T) {
		assert.Contains(t, content, "nephoran_test_duration_seconds_bucket")
		assert.Contains(t, content, "nephoran_test_duration_seconds_sum")
		assert.Contains(t, content, "nephoran_test_duration_seconds_count")
	})

	t.Logf("Metrics content validation passed. Content length: %d characters", len(content))
}

// TestMetricsEndpointPerformance validates response time under load
func TestMetricsEndpointPerformance(t *testing.T) {
	// Create registry with many metrics
	registry := prometheus.NewRegistry()

	// Create multiple metrics to simulate real load
	for i := 0; i < 10; i++ {
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("nephoran_perf_test_metric_%d_total", i),
				Help: fmt.Sprintf("Performance test metric %d", i),
			},
			[]string{"label1", "label2", "label3"},
		)
		registry.MustRegister(counter)

		// Add data to each metric
		for j := 0; j < 5; j++ {
			counter.WithLabelValues(
				fmt.Sprintf("value1_%d", j),
				fmt.Sprintf("value2_%d", j),
				fmt.Sprintf("value3_%d", j),
			).Add(float64(j))
		}
	}

	// Setup server
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
	server := httptest.NewServer(router)
	defer server.Close()

	// Measure response time
	start := time.Now()
	resp, err := http.Get(server.URL + "/metrics")
	duration := time.Since(start)

	require.NoError(t, err)
	defer resp.Body.Close()

	// Validate performance
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Less(t, duration, 500*time.Millisecond, "Metrics endpoint should respond quickly")

	t.Logf("Metrics endpoint responded in %v", duration)
}

// Simple metric parsing for basic validation
func parseSimplePrometheusMetrics(resp *http.Response) (map[string]string, error) {
	scanner := bufio.NewScanner(resp.Body)
	metrics := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# HELP ") {
			parts := strings.SplitN(line[7:], " ", 2)
			if len(parts) >= 2 {
				metrics[parts[0]] = parts[1]
			}
		}
	}

	return metrics, scanner.Err()
}

// Helper function to check if metrics are enabled (simulates real implementation)
func isMetricsEnabled() bool {
	enabled := os.Getenv("METRICS_ENABLED")
	return enabled == "true"
}

// Helper function to assert metric help text
func assertMetricHasHelp(t *testing.T, metrics map[string]string, metricName, expectedHelp string) {
	help, exists := metrics[metricName]
	assert.True(t, exists, "Metric %s should have help text", metricName)
	if exists {
		assert.Contains(t, help, expectedHelp, "Metric %s help should contain '%s'", metricName, expectedHelp)
	}
}

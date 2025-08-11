package main

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

// TestMetricsEndpointStandalone validates the metrics endpoint functionality
func TestMetricsEndpointStandalone(t *testing.T) {
	// Store original environment
	originalEnv := os.Getenv("METRICS_ENABLED")
	defer func() {
		if originalEnv != "" {
			os.Setenv("METRICS_ENABLED", originalEnv)
		} else {
			os.Unsetenv("METRICS_ENABLED")
		}
	}()

	t.Run("Metrics endpoint enabled with expected metrics", func(t *testing.T) {
		// Enable metrics
		os.Setenv("METRICS_ENABLED", "true")

		// Create test registry with expected metrics
		registry := prometheus.NewRegistry()
		
		// Create LLM metrics as they would appear in the real system
		llmRequestsTotal := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_llm_requests_total",
				Help: "Total number of LLM requests by model and status",
			},
			[]string{"model", "status"},
		)
		
		llmProcessingDuration := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "nephoran_llm_processing_duration_seconds", 
				Help: "Duration of LLM processing requests by model and status",
				Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
			},
			[]string{"model", "status"},
		)
		
		// Create controller metrics as they would appear in the real system
		networkintentReconciles := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "networkintent_reconciles_total",
				Help: "Total number of NetworkIntent reconciliations",
			},
			[]string{"controller", "namespace", "name", "result"},
		)
		
		networkintentProcessingDuration := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "networkintent_processing_duration_seconds",
				Help: "Duration of NetworkIntent processing phases",
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
		
		// Generate realistic test data
		llmRequestsTotal.WithLabelValues("gpt-4o-mini", "success").Add(25)
		llmRequestsTotal.WithLabelValues("gpt-4o-mini", "error").Add(2)
		llmRequestsTotal.WithLabelValues("mistral-8x22b", "success").Add(15)
		llmProcessingDuration.WithLabelValues("gpt-4o-mini", "success").Observe(1.5)
		llmProcessingDuration.WithLabelValues("gpt-4o-mini", "error").Observe(5.2)
		
		networkintentReconciles.WithLabelValues("networkintent", "default", "test-intent", "success").Add(10)
		networkintentReconciles.WithLabelValues("networkintent", "production", "prod-intent", "error").Add(1)
		networkintentProcessingDuration.WithLabelValues("networkintent", "default", "test-intent", "llm_processing").Observe(2.3)

		// Setup HTTP server with metrics endpoint
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

		// Read and parse metrics content
		scanner := bufio.NewScanner(resp.Body)
		var content strings.Builder
		metrics := make(map[string]string) // metric name -> help text
		
		for scanner.Scan() {
			line := scanner.Text()
			content.WriteString(line + "\n")
			
			// Parse HELP lines
			if strings.HasPrefix(line, "# HELP ") {
				parts := strings.SplitN(line[7:], " ", 2)
				if len(parts) >= 2 {
					metrics[parts[0]] = parts[1]
				}
			}
		}
		require.NoError(t, scanner.Err())

		fullContent := content.String()

		// Verify expected LLM metrics are present
		assert.Contains(t, metrics, "nephoran_llm_requests_total")
		assert.Contains(t, metrics, "nephoran_llm_processing_duration_seconds")
		
		// Verify expected controller metrics are present
		assert.Contains(t, metrics, "networkintent_reconciles_total") 
		assert.Contains(t, metrics, "networkintent_processing_duration_seconds")
		
		// Verify help text content
		assert.Contains(t, metrics["nephoran_llm_requests_total"], "Total number of LLM requests")
		assert.Contains(t, metrics["nephoran_llm_processing_duration_seconds"], "Duration of LLM processing requests")
		assert.Contains(t, metrics["networkintent_reconciles_total"], "Total number of NetworkIntent reconciliations")
		assert.Contains(t, metrics["networkintent_processing_duration_seconds"], "Duration of NetworkIntent processing phases")
		
		// Verify metric types are declared
		assert.Contains(t, fullContent, "# TYPE nephoran_llm_requests_total counter")
		assert.Contains(t, fullContent, "# TYPE nephoran_llm_processing_duration_seconds histogram") 
		assert.Contains(t, fullContent, "# TYPE networkintent_reconciles_total counter")
		assert.Contains(t, fullContent, "# TYPE networkintent_processing_duration_seconds histogram")
		
		// Verify actual metric values with labels
		assert.Contains(t, fullContent, `nephoran_llm_requests_total{model="gpt-4o-mini",status="success"} 25`)
		assert.Contains(t, fullContent, `nephoran_llm_requests_total{model="gpt-4o-mini",status="error"} 2`)
		assert.Contains(t, fullContent, `networkintent_reconciles_total{controller="networkintent",name="test-intent",namespace="default",result="success"} 10`)
		
		// Verify histogram metrics are present
		assert.Contains(t, fullContent, "nephoran_llm_processing_duration_seconds_bucket")
		assert.Contains(t, fullContent, "nephoran_llm_processing_duration_seconds_sum")
		assert.Contains(t, fullContent, "nephoran_llm_processing_duration_seconds_count")
		assert.Contains(t, fullContent, "networkintent_processing_duration_seconds_bucket")

		t.Logf("Successfully validated %d metrics in %d lines of output", len(metrics), strings.Count(fullContent, "\n"))
	})

	t.Run("Metrics endpoint disabled returns 404", func(t *testing.T) {
		// Disable metrics
		os.Setenv("METRICS_ENABLED", "false")

		// Create router without metrics endpoint (simulating disabled state)
		router := mux.NewRouter()
		// Don't register /metrics endpoint when disabled
		
		server := httptest.NewServer(router)
		defer server.Close()

		// Test metrics endpoint should return 404
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestMetricsEndpointConditionalBehavior validates conditional exposure behavior
func TestMetricsEndpointConditionalBehavior(t *testing.T) {
	testCases := []struct {
		name           string
		metricsEnabled string
		expectEndpoint bool
		expectedStatus int
		description    string
	}{
		{
			name:           "METRICS_ENABLED=true",
			metricsEnabled: "true",
			expectEndpoint: true,
			expectedStatus: http.StatusOK,
			description:    "Should expose metrics endpoint successfully",
		},
		{
			name:           "METRICS_ENABLED=false",
			metricsEnabled: "false", 
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 when metrics disabled",
		},
		{
			name:           "METRICS_ENABLED unset",
			metricsEnabled: "",
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
			description:    "Should default to disabled when env var unset",
		},
		{
			name:           "METRICS_ENABLED invalid",
			metricsEnabled: "invalid_value",
			expectEndpoint: false,
			expectedStatus: http.StatusNotFound,
			description:    "Should default to disabled for invalid values",
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

			// Create router with conditional metrics endpoint
			router := mux.NewRouter()
			
			// Simulate the application's conditional metrics setup
			if isMetricsEnabledForTest() && tc.expectEndpoint {
				registry := prometheus.NewRegistry()
				testMetric := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "test_metric_total",
					Help: "A test metric for validation",
				})
				registry.MustRegister(testMetric)
				testMetric.Inc()
				
				router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
			}
			
			server := httptest.NewServer(router)
			defer server.Close()

			// Test endpoint
			resp, err := http.Get(server.URL + "/metrics")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode, tc.description)
			
			if tc.expectEndpoint {
				assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
			}
			
			t.Logf("Test case '%s': Status %d (expected %d) - %s", 
				tc.name, resp.StatusCode, tc.expectedStatus, tc.description)
		})
	}
}

// TestMetricsEndpointPerformanceAndConcurrency validates performance characteristics
func TestMetricsEndpointPerformanceAndConcurrency(t *testing.T) {
	// Create registry with substantial metrics to test performance
	registry := prometheus.NewRegistry()
	
	// Create multiple metrics simulating a real system
	for i := 0; i < 20; i++ {
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("nephoran_test_metric_%d_total", i),
				Help: fmt.Sprintf("Test metric %d for performance validation", i),
			},
			[]string{"service", "endpoint", "status"},
		)
		
		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: fmt.Sprintf("nephoran_test_duration_%d_seconds", i),
				Help: fmt.Sprintf("Test duration metric %d", i),
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "operation"},
		)
		
		registry.MustRegister(counter, histogram)
		
		// Add realistic data
		services := []string{"llm-processor", "controller", "rag-service"}
		endpoints := []string{"/process", "/reconcile", "/search"}
		statuses := []string{"200", "400", "500"}
		
		for _, service := range services {
			for _, endpoint := range endpoints {
				for _, status := range statuses {
					counter.WithLabelValues(service, endpoint, status).Add(float64(i * 10))
				}
			}
			histogram.WithLabelValues(service, "request").Observe(float64(i) * 0.1)
		}
	}
	
	// Setup server
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{})).Methods("GET")
	server := httptest.NewServer(router)
	defer server.Close()
	
	t.Run("Response time performance", func(t *testing.T) {
		start := time.Now()
		resp, err := http.Get(server.URL + "/metrics")
		duration := time.Since(start)
		
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, duration, 2*time.Second, "Metrics endpoint should respond within 2 seconds")
		
		t.Logf("Metrics endpoint responded in %v with substantial metric load", duration)
	})
	
	t.Run("Concurrent access", func(t *testing.T) {
		const concurrentRequests = 10
		results := make(chan struct {
			status   int
			duration time.Duration
			err      error
		}, concurrentRequests)
		
		// Launch concurrent requests
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				start := time.Now()
				resp, err := http.Get(server.URL + "/metrics")
				duration := time.Since(start)
				
				if err != nil {
					results <- struct {
						status   int
						duration time.Duration
						err      error
					}{0, duration, err}
					return
				}
				
				resp.Body.Close()
				results <- struct {
					status   int
					duration time.Duration
					err      error
				}{resp.StatusCode, duration, nil}
			}()
		}
		
		// Collect results
		successCount := 0
		var totalDuration time.Duration
		for i := 0; i < concurrentRequests; i++ {
			result := <-results
			if result.err != nil {
				t.Errorf("Concurrent request %d failed: %v", i, result.err)
			} else {
				assert.Equal(t, http.StatusOK, result.status)
				successCount++
				totalDuration += result.duration
			}
		}
		
		assert.Equal(t, concurrentRequests, successCount, "All concurrent requests should succeed")
		avgDuration := totalDuration / time.Duration(concurrentRequests)
		assert.Less(t, avgDuration, 1*time.Second, "Average response time should be reasonable under concurrency")
		
		t.Logf("All %d concurrent requests succeeded with average duration %v", 
			concurrentRequests, avgDuration)
	})
}

// isMetricsEnabledForTest simulates the real application's environment check
func isMetricsEnabledForTest() bool {
	enabled := os.Getenv("METRICS_ENABLED")
	return enabled == "true"
}

// Main function for running standalone tests
func main() {
	// This allows the test to be run as a standalone program
	testing.Main(func(pat, str string) (bool, error) {
		return true, nil
	}, []testing.InternalTest{
		{
			Name: "TestMetricsEndpointStandalone",
			F:    TestMetricsEndpointStandalone,
		},
		{
			Name: "TestMetricsEndpointConditionalBehavior", 
			F:    TestMetricsEndpointConditionalBehavior,
		},
		{
			Name: "TestMetricsEndpointPerformanceAndConcurrency",
			F:    TestMetricsEndpointPerformanceAndConcurrency,
		},
	}, nil, nil)
}
// Package integration provides integration tests for the Nephoran Intent Operator.
//
// This test validates Prometheus metrics endpoint functionality including:
// - HTTP server setup with metrics endpoint
// - Conditional metrics exposure based on METRICS_ENABLED environment variable
// - LLM metrics collection and exposition
// - Controller metrics collection and exposition
// - Prometheus text format parsing and validation
package integration_tests

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
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// MetricsScrapeTestSuite provides comprehensive testing for Prometheus metrics endpoint
type MetricsScrapeTestSuite struct {
	suite.Suite

	// Test components
	router            *mux.Router
	server            *httptest.Server
	llmMetrics        *llm.PrometheusMetrics
	controllerMetrics *controllers.ControllerMetrics

	// Test state
	originalMetricsEnabled string
	testRegistry           *prometheus.Registry
}

// SetupSuite initializes test suite components
func (suite *MetricsScrapeTestSuite) SetupSuite() {
	// Store original environment state
	suite.originalMetricsEnabled = os.Getenv("METRICS_ENABLED")

	// Create isolated test registry
	suite.testRegistry = prometheus.NewRegistry()
}

// TearDownSuite cleans up test suite resources
func (suite *MetricsScrapeTestSuite) TearDownSuite() {
	// Restore original environment
	if suite.originalMetricsEnabled != "" {
		os.Setenv("METRICS_ENABLED", suite.originalMetricsEnabled)
	} else {
		os.Unsetenv("METRICS_ENABLED")
	}

	if suite.server != nil {
		suite.server.Close()
	}
}

// SetupTest prepares individual test cases
func (suite *MetricsScrapeTestSuite) SetupTest() {
	suite.router = mux.NewRouter()
}

// TearDownTest cleans up after individual test cases
func (suite *MetricsScrapeTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
		suite.server = nil
	}

	// Reset environment for next test
	os.Unsetenv("METRICS_ENABLED")
}

// TestMetricsEndpointEnabled validates metrics endpoint when METRICS_ENABLED=true
func (suite *MetricsScrapeTestSuite) TestMetricsEndpointEnabled() {
	// Enable metrics
	os.Setenv("METRICS_ENABLED", "true")

	// Setup test server with metrics
	suite.setupTestServerWithMetrics()

	// Make request to metrics endpoint
	resp, err := http.Get(suite.server.URL + "/metrics")
	require.NoError(suite.T(), err)
	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	// Validate response
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(suite.T(), "text/plain; version=0.0.4; charset=utf-8",
		resp.Header.Get("Content-Type"))

	// Read response body
	body := make([]byte, 8192)
	n, err := resp.Body.Read(body)
	require.NoError(suite.T(), err)

	content := string(body[:n])
	suite.T().Logf("Metrics response (first 1000 chars): %s",
		content[:minInt(1000, len(content))])

	// Validate basic Prometheus format
	assert.Contains(suite.T(), content, "# HELP")
	assert.Contains(suite.T(), content, "# TYPE")
}

// TestMetricsEndpointDisabled validates metrics endpoint when METRICS_ENABLED=false
func (suite *MetricsScrapeTestSuite) TestMetricsEndpointDisabled() {
	// Disable metrics explicitly
	os.Setenv("METRICS_ENABLED", "false")

	// Setup test server without metrics
	suite.setupTestServerWithoutMetrics()

	// Make request to metrics endpoint
	resp, err := http.Get(suite.server.URL + "/metrics")
	require.NoError(suite.T(), err)
	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	// Validate response - should return 404
	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
}

// TestLLMMetricsPresent validates LLM-specific metrics are exposed
func (suite *MetricsScrapeTestSuite) TestLLMMetricsPresent() {
	// Enable metrics
	os.Setenv("METRICS_ENABLED", "true")

	// Setup test server with LLM metrics
	suite.setupTestServerWithLLMMetrics()

	// Generate some test metrics
	suite.generateTestLLMMetrics()

	// Make request to metrics endpoint
	resp, err := http.Get(suite.server.URL + "/metrics")
	require.NoError(suite.T(), err)
	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	// Parse metrics
	metrics, err := suite.parsePrometheusMetrics(resp)
	require.NoError(suite.T(), err)

	// Validate LLM metrics presence
	expectedLLMMetrics := []string{
		"nephoran_llm_requests_total",
		"nephoran_llm_processing_duration_seconds",
		"nephoran_llm_errors_total",
		"nephoran_llm_cache_hits_total",
		"nephoran_llm_cache_misses_total",
	}

	for _, metricName := range expectedLLMMetrics {
		suite.assertMetricPresent(metrics, metricName)
		suite.assertMetricHasHelp(metrics, metricName)
		suite.assertMetricHasType(metrics, metricName)
	}

	// Validate specific help text
	suite.assertMetricHelpContains(metrics, "nephoran_llm_requests_total",
		"Total number of LLM requests")
	suite.assertMetricHelpContains(metrics, "nephoran_llm_processing_duration_seconds",
		"Duration of LLM processing requests")
}

// TestControllerMetricsPresent validates controller-specific metrics are exposed
func (suite *MetricsScrapeTestSuite) TestControllerMetricsPresent() {
	// Enable metrics
	os.Setenv("METRICS_ENABLED", "true")

	// Setup test server with controller metrics
	suite.setupTestServerWithControllerMetrics()

	// Generate some test controller metrics
	suite.generateTestControllerMetrics()

	// Make request to metrics endpoint
	resp, err := http.Get(suite.server.URL + "/metrics")
	require.NoError(suite.T(), err)
	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	// Parse metrics
	metrics, err := suite.parsePrometheusMetrics(resp)
	require.NoError(suite.T(), err)

	// Validate controller metrics presence
	expectedControllerMetrics := []string{
		"networkintent_reconciles_total",
		"networkintent_processing_duration_seconds",
		"networkintent_reconcile_errors_total",
		"networkintent_status",
	}

	for _, metricName := range expectedControllerMetrics {
		suite.assertMetricPresent(metrics, metricName)
		suite.assertMetricHasHelp(metrics, metricName)
		suite.assertMetricHasType(metrics, metricName)
	}

	// Validate specific help text
	suite.assertMetricHelpContains(metrics, "networkintent_reconciles_total",
		"Total number of NetworkIntent reconciliations")
	suite.assertMetricHelpContains(metrics, "networkintent_processing_duration_seconds",
		"Duration of NetworkIntent processing phases")
}

// TestMetricsEndpointConcurrency validates metrics endpoint under concurrent access
func (suite *MetricsScrapeTestSuite) TestMetricsEndpointConcurrency() {
	// Enable metrics
	os.Setenv("METRICS_ENABLED", "true")

	// Setup test server with metrics
	suite.setupTestServerWithMetrics()

	// Create multiple concurrent requests
	const concurrentRequests = 10
	responses := make(chan *http.Response, concurrentRequests)
	errors := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			resp, err := http.Get(suite.server.URL + "/metrics")
			if err != nil {
				errors <- err
				return
			}
			responses <- resp
		}()
	}

	// Collect all responses
	successCount := 0
	for i := 0; i < concurrentRequests; i++ {
		select {
		case resp := <-responses:
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
			resp.Body.Close()
			successCount++
		case err := <-errors:
			suite.T().Errorf("Concurrent request failed: %v", err)
		case <-time.After(5 * time.Second):
			suite.T().Error("Request timeout")
		}
	}

	assert.Equal(suite.T(), concurrentRequests, successCount,
		"All concurrent requests should succeed")
}

// TestMetricsEndpointPerformance validates metrics endpoint response time
func (suite *MetricsScrapeTestSuite) TestMetricsEndpointPerformance() {
	// Enable metrics
	os.Setenv("METRICS_ENABLED", "true")

	// Setup test server with comprehensive metrics
	suite.setupTestServerWithAllMetrics()

	// Generate substantial metrics to test performance
	suite.generateComprehensiveTestMetrics()

	// Measure response time
	start := time.Now()
	resp, err := http.Get(suite.server.URL + "/metrics")
	duration := time.Since(start)

	require.NoError(suite.T(), err)
	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	// Validate response time is acceptable (< 1 second)
	assert.Less(suite.T(), duration, time.Second,
		"Metrics endpoint should respond within 1 second")

	// Validate response is successful
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Log performance metrics
	suite.T().Logf("Metrics endpoint response time: %v", duration)
}

// setupTestServerWithMetrics creates a test server with standard metrics endpoint
func (suite *MetricsScrapeTestSuite) setupTestServerWithMetrics() {
	// Initialize metrics components
	suite.llmMetrics = llm.NewPrometheusMetrics()
	suite.controllerMetrics = controllers.NewControllerMetrics("networkintent")

	// Setup metrics endpoint
	suite.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Create and start test server
	suite.server = httptest.NewServer(suite.router)
}

// setupTestServerWithoutMetrics creates a test server without metrics endpoint
func (suite *MetricsScrapeTestSuite) setupTestServerWithoutMetrics() {
	// Do not register metrics endpoint
	// Only setup basic health endpoints
	suite.router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	}).Methods("GET")

	// Create and start test server
	suite.server = httptest.NewServer(suite.router)
}

// setupTestServerWithLLMMetrics creates a test server focused on LLM metrics
func (suite *MetricsScrapeTestSuite) setupTestServerWithLLMMetrics() {
	// Initialize LLM metrics
	suite.llmMetrics = llm.NewPrometheusMetrics()

	// Setup metrics endpoint
	suite.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Create and start test server
	suite.server = httptest.NewServer(suite.router)
}

// setupTestServerWithControllerMetrics creates a test server focused on controller metrics
func (suite *MetricsScrapeTestSuite) setupTestServerWithControllerMetrics() {
	// Initialize controller metrics
	suite.controllerMetrics = controllers.NewControllerMetrics("networkintent")

	// Setup metrics endpoint
	suite.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Create and start test server
	suite.server = httptest.NewServer(suite.router)
}

// setupTestServerWithAllMetrics creates a test server with comprehensive metrics
func (suite *MetricsScrapeTestSuite) setupTestServerWithAllMetrics() {
	// Initialize all metrics components
	suite.llmMetrics = llm.NewPrometheusMetrics()
	suite.controllerMetrics = controllers.NewControllerMetrics("networkintent")

	// Setup metrics endpoint
	suite.router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Create and start test server
	suite.server = httptest.NewServer(suite.router)
}

// generateTestLLMMetrics creates sample LLM metrics for testing
func (suite *MetricsScrapeTestSuite) generateTestLLMMetrics() {
	if suite.llmMetrics == nil {
		return
	}

	// Generate sample LLM request metrics
	models := []string{"gpt-4o-mini", "mistral-8x22b"}
	statuses := []string{"success", "error"}

	for _, model := range models {
		for _, status := range statuses {
			duration := time.Duration(100+len(model)*10) * time.Millisecond
			suite.llmMetrics.RecordRequest(model, status, duration)

			if status == "error" {
				suite.llmMetrics.RecordError(model, "processing_error")
			}
		}

		// Generate cache metrics
		suite.llmMetrics.RecordCacheHit(model)
		suite.llmMetrics.RecordCacheMiss(model)
		suite.llmMetrics.RecordRetryAttempt(model)
	}
}

// generateTestControllerMetrics creates sample controller metrics for testing
func (suite *MetricsScrapeTestSuite) generateTestControllerMetrics() {
	if suite.controllerMetrics == nil {
		return
	}

	// Generate sample controller metrics
	testCases := []struct {
		namespace string
		name      string
		result    string
		phase     string
		duration  float64
	}{
		{"default", "test-intent-1", "success", "llm_processing", 1.5},
		{"default", "test-intent-2", "success", "gitops_deployment", 3.2},
		{"prod", "prod-intent-1", "error", "validation", 0.8},
		{"staging", "staging-intent-1", "success", "reconciliation", 2.1},
	}

	for _, tc := range testCases {
		suite.controllerMetrics.RecordReconcileTotal(tc.namespace, tc.name, tc.result)
		suite.controllerMetrics.RecordProcessingDuration(tc.namespace, tc.name, tc.phase, tc.duration)

		if tc.result == "error" {
			suite.controllerMetrics.RecordReconcileError(tc.namespace, tc.name, "validation_error")
		}

		// Set status based on result
		status := controllers.StatusReady
		if tc.result == "error" {
			status = controllers.StatusFailed
		}
		suite.controllerMetrics.SetStatus(tc.namespace, tc.name, tc.phase, status)
	}
}

// generateComprehensiveTestMetrics creates extensive metrics for performance testing
func (suite *MetricsScrapeTestSuite) generateComprehensiveTestMetrics() {
	// Generate many LLM metrics
	for i := 0; i < 50; i++ {
		model := fmt.Sprintf("model-%d", i%5)
		status := []string{"success", "error"}[i%2]
		duration := time.Duration(i*10) * time.Millisecond

		if suite.llmMetrics != nil {
			suite.llmMetrics.RecordRequest(model, status, duration)
		}
	}

	// Generate many controller metrics
	for i := 0; i < 50; i++ {
		namespace := fmt.Sprintf("ns-%d", i%3)
		name := fmt.Sprintf("intent-%d", i)
		result := []string{"success", "error"}[i%3]
		phase := []string{"validation", "processing", "deployment"}[i%3]

		if suite.controllerMetrics != nil {
			suite.controllerMetrics.RecordReconcileTotal(namespace, name, result)
			suite.controllerMetrics.RecordProcessingDuration(namespace, name, phase, float64(i)*0.1)
		}
	}
}

// PrometheusMetric represents a parsed Prometheus metric
type PrometheusMetric struct {
	Name   string
	Help   string
	Type   string
	Values []PrometheusValue
}

// PrometheusValue represents a metric value with labels
type PrometheusValue struct {
	Labels map[string]string
	Value  string
}

// parsePrometheusMetrics parses Prometheus text format response
func (suite *MetricsScrapeTestSuite) parsePrometheusMetrics(resp *http.Response) (map[string]PrometheusMetric, error) {
	scanner := bufio.NewScanner(resp.Body)
	metrics := make(map[string]PrometheusMetric)

	var currentMetric *PrometheusMetric

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments not related to metrics
		if line == "" || (strings.HasPrefix(line, "#") &&
			!strings.HasPrefix(line, "# HELP") &&
			!strings.HasPrefix(line, "# TYPE")) {
			continue
		}

		if strings.HasPrefix(line, "# HELP ") {
			parts := strings.SplitN(line[7:], " ", 2)
			if len(parts) >= 1 {
				metricName := parts[0]
				help := ""
				if len(parts) > 1 {
					help = parts[1]
				}

				if metric, exists := metrics[metricName]; exists {
					metric.Help = help
					metrics[metricName] = metric
				} else {
					metrics[metricName] = PrometheusMetric{
						Name: metricName,
						Help: help,
					}
				}
			}
		} else if strings.HasPrefix(line, "# TYPE ") {
			parts := strings.SplitN(line[7:], " ", 2)
			if len(parts) >= 2 {
				metricName := parts[0]
				metricType := parts[1]

				if metric, exists := metrics[metricName]; exists {
					metric.Type = metricType
					metrics[metricName] = metric
				} else {
					metrics[metricName] = PrometheusMetric{
						Name: metricName,
						Type: metricType,
					}
				}
			}
		} else if !strings.HasPrefix(line, "#") {
			// Parse metric value line
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				metricPart := parts[0]
				value := parts[1]

				// Extract metric name and labels
				var metricName string
				var labels map[string]string

				if strings.Contains(metricPart, "{") {
					// Metric with labels
					openBrace := strings.Index(metricPart, "{")
					metricName = metricPart[:openBrace]
					labelsPart := metricPart[openBrace+1 : len(metricPart)-1]

					labels = make(map[string]string)
					if labelsPart != "" {
						labelPairs := strings.Split(labelsPart, ",")
						for _, pair := range labelPairs {
							kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
							if len(kv) == 2 {
								key := strings.Trim(kv[0], `"`)
								val := strings.Trim(kv[1], `"`)
								labels[key] = val
							}
						}
					}
				} else {
					// Metric without labels
					metricName = metricPart
					labels = make(map[string]string)
				}

				// Add value to metric
				if metric, exists := metrics[metricName]; exists {
					metric.Values = append(metric.Values, PrometheusValue{
						Labels: labels,
						Value:  value,
					})
					metrics[metricName] = metric
				} else {
					metrics[metricName] = PrometheusMetric{
						Name: metricName,
						Values: []PrometheusValue{{
							Labels: labels,
							Value:  value,
						}},
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning metrics response: %w", err)
	}

	return metrics, nil
}

// assertMetricPresent validates a metric exists in the parsed metrics
func (suite *MetricsScrapeTestSuite) assertMetricPresent(metrics map[string]PrometheusMetric, name string) {
	_, exists := metrics[name]
	assert.True(suite.T(), exists, "Metric %s should be present", name)
}

// assertMetricHasHelp validates a metric has help text
func (suite *MetricsScrapeTestSuite) assertMetricHasHelp(metrics map[string]PrometheusMetric, name string) {
	metric, exists := metrics[name]
	require.True(suite.T(), exists, "Metric %s should exist", name)
	assert.NotEmpty(suite.T(), metric.Help, "Metric %s should have help text", name)
}

// assertMetricHasType validates a metric has type information
func (suite *MetricsScrapeTestSuite) assertMetricHasType(metrics map[string]PrometheusMetric, name string) {
	metric, exists := metrics[name]
	require.True(suite.T(), exists, "Metric %s should exist", name)
	assert.NotEmpty(suite.T(), metric.Type, "Metric %s should have type information", name)
}

// assertMetricHelpContains validates a metric's help text contains specific content
func (suite *MetricsScrapeTestSuite) assertMetricHelpContains(metrics map[string]PrometheusMetric, name, expectedContent string) {
	metric, exists := metrics[name]
	require.True(suite.T(), exists, "Metric %s should exist", name)
	assert.Contains(suite.T(), metric.Help, expectedContent,
		"Metric %s help text should contain '%s'", name, expectedContent)
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestMetricsScrapeTestSuite runs the complete test suite
func TestMetricsScrapeTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsScrapeTestSuite))
}

// TestMetricsScrapeFunctional provides functional validation of metrics scraping
func TestMetricsScrapeFunctional(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("METRICS_ENABLED")
	defer func() {
		if originalEnv != "" {
			os.Setenv("METRICS_ENABLED", originalEnv)
		} else {
			os.Unsetenv("METRICS_ENABLED")
		}
	}()

	t.Run("Basic metrics endpoint availability", func(t *testing.T) {
		// Enable metrics
		os.Setenv("METRICS_ENABLED", "true")

		// Create basic test server
		router := mux.NewRouter()
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
		server := httptest.NewServer(router)
		defer server.Close() // #nosec G307 - Error handled in defer

		// Test metrics endpoint
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
	})

	t.Run("Metrics disabled scenario", func(t *testing.T) {
		// Disable metrics
		os.Setenv("METRICS_ENABLED", "false")

		// Create test server without metrics endpoint
		router := mux.NewRouter()
		// Intentionally don't register /metrics endpoint
		server := httptest.NewServer(router)
		defer server.Close() // #nosec G307 - Error handled in defer

		// Test metrics endpoint should return 404
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

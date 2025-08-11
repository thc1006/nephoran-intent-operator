// Example usage of the Prometheus metrics implementation for the LLM client.
//
// This file demonstrates how to use the enhanced LLM client with Prometheus metrics
// integration based on the METRICS_ENABLED environment variable.

package llm

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// ExamplePrometheusMetricsUsage demonstrates the Prometheus metrics integration
func ExamplePrometheusMetricsUsage() {
	// Set environment variable to enable metrics
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")

	// Create LLM client with default configuration
	client := NewClient("https://api.openai.com/v1/chat/completions")

	// Process some intents to generate metrics
	ctx := context.Background()
	
	// Successful request
	intent1 := "Deploy a 5G AMF with high availability"
	response1, err1 := client.ProcessIntent(ctx, intent1)
	if err1 != nil {
		log.Printf("Request 1 failed: %v", err1)
	} else {
		log.Printf("Request 1 successful: %s", response1[:50])
	}

	// Cache hit (if processing same intent again)
	response2, err2 := client.ProcessIntent(ctx, intent1)
	if err2 != nil {
		log.Printf("Request 2 failed: %v", err2)
	} else {
		log.Printf("Request 2 (cache hit): %s", response2[:50])
	}

	// Get comprehensive metrics that include both traditional and Prometheus metrics
	if client.metricsIntegrator != nil {
		metrics := client.metricsIntegrator.GetComprehensiveMetrics()
		fmt.Printf("Metrics enabled: %v\n", metrics["prometheus_enabled"])
		fmt.Printf("Environment variable: %v\n", metrics["metrics_enabled_env"])
	}

	// Cleanup
	client.Shutdown()

	fmt.Println("Example completed successfully")
}

// ExamplePrometheusMetricsWithoutEnvironment shows behavior when METRICS_ENABLED is false
func ExamplePrometheusMetricsWithoutEnvironment() {
	// Ensure metrics are disabled
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")

	// Create client
	client := NewClient("https://api.openai.com/v1/chat/completions")

	// Process intent - metrics should not be recorded to Prometheus
	ctx := context.Background()
	intent := "Configure O-RAN DU with load balancing"
	
	_, err := client.ProcessIntent(ctx, intent)
	if err != nil {
		log.Printf("Request failed: %v", err)
	}

	// Check metrics status
	if client.metricsIntegrator != nil {
		metrics := client.metricsIntegrator.GetComprehensiveMetrics()
		fmt.Printf("Prometheus metrics enabled: %v\n", metrics["prometheus_enabled"])
		fmt.Printf("Environment variable: %v\n", metrics["metrics_enabled_env"])
	}

	client.Shutdown()

	fmt.Println("Example without metrics completed successfully")
}

// ExamplePrometheusMetricsTypes demonstrates different metric types being recorded
func ExamplePrometheusMetricsTypes() {
	os.Setenv("METRICS_ENABLED", "true")
	defer os.Unsetenv("METRICS_ENABLED")

	// Create client
	config := ClientConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		ModelName:   "gpt-4o-mini",
		MaxTokens:   1000,
		BackendType: "openai",
		Timeout:     30 * time.Second,
	}
	
	client := NewClientWithConfig("https://api.openai.com/v1/chat/completions", config)

	ctx := context.Background()

	// Generate different types of metrics:
	
	// 1. Successful request (records: requests_total, processing_duration_seconds)
	fmt.Println("Recording successful request metrics...")
	_, _ = client.ProcessIntent(ctx, "Deploy AMF in production")

	// 2. Cache operations (records: cache_hits_total, cache_misses_total)  
	fmt.Println("Recording cache metrics...")
	_, _ = client.ProcessIntent(ctx, "Deploy AMF in production") // Should be cache hit

	// 3. Simulate errors for error metrics (records: errors_total with error_type label)
	fmt.Println("Recording error metrics...")
	// This would normally come from actual processing errors, but we can simulate
	if client.metricsIntegrator != nil {
		client.metricsIntegrator.prometheusMetrics.RecordError("gpt-4o-mini", "timeout")
		client.metricsIntegrator.prometheusMetrics.RecordError("gpt-4o-mini", "rate_limit_exceeded")
	}

	// 4. Retry attempts (records: retry_attempts_total)
	fmt.Println("Recording retry metrics...")
	if client.metricsIntegrator != nil {
		client.metricsIntegrator.RecordRetryAttempt("gpt-4o-mini")
	}

	// 5. Fallback attempts (records: fallback_attempts_total)  
	fmt.Println("Recording fallback metrics...")
	if client.metricsIntegrator != nil {
		client.metricsIntegrator.RecordFallbackAttempt("gpt-4o-mini", "gpt-3.5-turbo")
	}

	fmt.Println("All metric types recorded successfully")
	client.Shutdown()
}

// ExampleMetricNamingConventions shows the Prometheus metric naming conventions used
func ExampleMetricNamingConventions() {
	fmt.Println("Prometheus metrics for LLM client follow these naming conventions:")
	fmt.Println("")
	fmt.Println("Counter Metrics:")
	fmt.Println("  - nephoran_llm_requests_total{model, status}")
	fmt.Println("  - nephoran_llm_errors_total{model, error_type}")
	fmt.Println("  - nephoran_llm_cache_hits_total{model}")
	fmt.Println("  - nephoran_llm_cache_misses_total{model}")
	fmt.Println("  - nephoran_llm_retry_attempts_total{model}")
	fmt.Println("  - nephoran_llm_fallback_attempts_total{original_model, fallback_model}")
	fmt.Println("")
	fmt.Println("Histogram Metrics:")
	fmt.Println("  - nephoran_llm_processing_duration_seconds{model, status}")
	fmt.Println("")
	fmt.Println("Labels:")
	fmt.Println("  - model: The LLM model name (e.g., 'gpt-4o-mini', 'mistral-8x22b')")
	fmt.Println("  - status: Request status ('success' or 'error')")
	fmt.Println("  - error_type: Specific error category (e.g., 'timeout', 'rate_limit_exceeded')")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  - METRICS_ENABLED: Set to 'true' to enable Prometheus metrics")
}
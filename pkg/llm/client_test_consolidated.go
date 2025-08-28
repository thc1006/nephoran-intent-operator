package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// TestNewClient tests client creation with default configuration
func TestNewClient(t *testing.T) {
	client := NewClient("http://test.example.com")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.url != "http://test.example.com" {
		t.Errorf("Expected URL http://test.example.com, got %s", client.url)
	}

	if client.modelName != "gpt-4o-mini" {
		t.Errorf("Expected model gpt-4o-mini, got %s", client.modelName)
	}

	if client.maxTokens != 2048 {
		t.Errorf("Expected maxTokens 2048, got %d", client.maxTokens)
	}

	if client.backendType != "openai" {
		t.Errorf("Expected backend openai, got %s", client.backendType)
	}
}

// TestNewClientWithConfig tests client creation with custom configuration
func TestNewClientWithConfig(t *testing.T) {
	config := ClientConfig{
		APIKey:      "test-api-key",
		ModelName:   "gpt-4",
		MaxTokens:   4096,
		BackendType: "openai",
		Timeout:     30 * time.Second,
	}

	client := NewClientWithConfig("http://test.example.com", config)

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected API key test-api-key, got %s", client.apiKey)
	}

	if client.modelName != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", client.modelName)
	}

	if client.maxTokens != 4096 {
		t.Errorf("Expected maxTokens 4096, got %d", client.maxTokens)
	}
}

// TestProcessIntentWithMockServer tests intent processing with a mock HTTP server
func TestProcessIntentWithMockServer(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Mock successful response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"test-amf\", \"namespace\": \"5g-core\", \"spec\": {\"replicas\": 3, \"image\": \"amf:latest\"}}"
				}
			}],
			"usage": {
				"total_tokens": 150
			}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		ModelName:   "gpt-4o-mini",
		MaxTokens:   2048,
		BackendType: "openai",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	result, err := client.ProcessIntent(ctx, "Deploy AMF with 3 replicas")

	if err != nil {
		t.Fatalf("ProcessIntent failed: %v", err)
	}

	if !strings.Contains(result, "NetworkFunctionDeployment") {
		t.Errorf("Expected result to contain NetworkFunctionDeployment, got: %s", result)
	}

	if !strings.Contains(result, "test-amf") {
		t.Errorf("Expected result to contain test-amf, got: %s", result)
	}
}

// TestProcessIntentWithRAGBackend tests RAG backend processing
func TestProcessIntentWithRAGBackend(t *testing.T) {
	// Create mock RAG server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/process") {
			t.Errorf("Expected path to end with /process, got %s", r.URL.Path)
		}

		// Mock RAG response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"type": "NetworkFunctionDeployment",
			"name": "rag-processed-amf",
			"namespace": "5g-core",
			"spec": {
				"replicas": 2,
				"image": "amf:v1.0.0"
			}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		BackendType: "rag",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	result, err := client.ProcessIntent(ctx, "Deploy AMF for production")

	if err != nil {
		t.Fatalf("ProcessIntent with RAG failed: %v", err)
	}

	if !strings.Contains(result, "rag-processed-amf") {
		t.Errorf("Expected result to contain rag-processed-amf, got: %s", result)
	}
}

// TestProcessIntentWithCache tests caching functionality
func TestProcessIntentWithCache(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"cached-test\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 100}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		CacheTTL:    1 * time.Minute,
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	intent := "Deploy test service"

	// First call should hit the server
	result1, err := client.ProcessIntent(ctx, intent)
	if err != nil {
		t.Fatalf("First ProcessIntent failed: %v", err)
	}

	// Second call should use cache
	result2, err := client.ProcessIntent(ctx, intent)
	if err != nil {
		t.Fatalf("Second ProcessIntent failed: %v", err)
	}

	if result1 != result2 {
		t.Error("Results should be identical due to caching")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call (second should be cached), got %d", callCount)
	}

	// Verify metrics show cache hit
	metrics := client.GetMetrics()
	if metrics.CacheHits < 1 {
		t.Errorf("Expected at least 1 cache hit, got %d", metrics.CacheHits)
	}
}

// TestProcessIntentWithRetry tests retry functionality
func TestProcessIntentWithRetry(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount <= 2 {
			// First two calls fail
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
			return
		}

		// Third call succeeds
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"retry-success\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 75}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	result, err := client.ProcessIntent(ctx, "Deploy with retry")

	if err != nil {
		t.Fatalf("ProcessIntent should succeed after retry: %v", err)
	}

	if !strings.Contains(result, "retry-success") {
		t.Errorf("Expected result to contain retry-success, got: %s", result)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls (2 failures + 1 success), got %d", callCount)
	}

	// Verify retry metrics
	metrics := client.GetMetrics()
	if metrics.RetryAttempts < 2 {
		t.Errorf("Expected at least 2 retry attempts, got %d", metrics.RetryAttempts)
	}
}

// TestProcessIntentWithFallback tests fallback URL functionality
func TestProcessIntentWithFallback(t *testing.T) {
	// Primary server that always fails
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
	}))
	defer primaryServer.Close()

	// Fallback server that succeeds
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"fallback-success\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 50}
		}`))
	}))
	defer fallbackServer.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(primaryServer.URL, config)
	client.SetFallbackURLs([]string{fallbackServer.URL})

	ctx := context.Background()
	result, err := client.ProcessIntent(ctx, "Deploy with fallback")

	if err != nil {
		t.Fatalf("ProcessIntent should succeed with fallback: %v", err)
	}

	if !strings.Contains(result, "fallback-success") {
		t.Errorf("Expected result to contain fallback-success, got: %s", result)
	}

	// Verify fallback metrics
	metrics := client.GetMetrics()
	if metrics.FallbackAttempts < 1 {
		t.Errorf("Expected at least 1 fallback attempt, got %d", metrics.FallbackAttempts)
	}
}

// TestProcessIntentTimeout tests timeout handling
func TestProcessIntentTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "slow response"}}]}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     100 * time.Millisecond, // Very short timeout
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	_, err := client.ProcessIntent(ctx, "This should timeout")

	if err == nil {
		t.Fatal("ProcessIntent should fail with timeout")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestCircuitBreaker tests circuit breaker integration
func TestCircuitBreaker(t *testing.T) {
	failureCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failureCount++

		// Always return errors to trigger circuit breaker
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     5 * time.Second,
		CircuitBreakerConfig: &shared.CircuitBreakerConfig{
			FailureThreshold:    3,
			SuccessThreshold:    2,
			Timeout:             1 * time.Second,
			HalfOpenTimeout:     500 * time.Millisecond,
			ResetTimeout:        10 * time.Second,
		},
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()

	// Make several failing requests to trigger circuit breaker
	for i := 0; i < 5; i++ {
		_, err := client.ProcessIntent(ctx, fmt.Sprintf("Request %d", i+1))
		if err == nil {
			t.Errorf("Request %d should have failed", i+1)
		}
	}

	// Circuit breaker should be open, preventing further requests from reaching server
	initialFailureCount := failureCount

	// Make more requests - these should be rejected by circuit breaker
	for i := 0; i < 3; i++ {
		_, err := client.ProcessIntent(ctx, fmt.Sprintf("Rejected %d", i+1))
		if err == nil {
			t.Errorf("Request should be rejected by circuit breaker")
		}
	}

	// Verify that circuit breaker prevented some requests from reaching server
	if failureCount > initialFailureCount+1 {
		t.Errorf("Circuit breaker should have prevented requests from reaching server")
	}
}

// TestClientShutdown tests client shutdown
func TestClientShutdown(t *testing.T) {
	client := NewClient("http://test.example.com")

	// Verify client is functional before shutdown
	if client.cache == nil {
		t.Error("Client cache should be initialized")
	}

	// Shutdown client
	client.Shutdown()

	// Verify cache is stopped
	// Note: We can't easily test this without exposing internal state
	// In a real implementation, you might expose a IsShutdown() method
}

// TestTokenTracker tests token usage tracking
func TestTokenTracker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"token-test\", \"namespace\": \"default\", \"spec\": {}}"
				}
			}],
			"usage": {
				"total_tokens": 125
			}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()
	_, err := client.ProcessIntent(ctx, "Test token tracking")

	if err != nil {
		t.Fatalf("ProcessIntent failed: %v", err)
	}

	// Check token tracking
	if client.tokenTracker == nil {
		t.Error("Token tracker should be initialized")
	}

	stats := client.tokenTracker.GetStats()

	if stats["total_tokens"].(int64) != 125 {
		t.Errorf("Expected 125 tokens, got %d", stats["total_tokens"])
	}

	if stats["request_count"].(int64) != 1 {
		t.Errorf("Expected 1 request, got %d", stats["request_count"])
	}

	if stats["total_cost"].(float64) <= 0 {
		t.Errorf("Expected positive cost, got %f", stats["total_cost"])
	}
}

// TestClientMetrics tests metrics collection
func TestClientMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"metrics-test\", \"namespace\": \"default\", \"spec\": {}}"
				}
			}],
			"usage": {"total_tokens": 100}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     10 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)

	ctx := context.Background()

	// Make a successful request
	_, err := client.ProcessIntent(ctx, "Test metrics collection")
	if err != nil {
		t.Fatalf("ProcessIntent failed: %v", err)
	}

	// Check metrics
	metrics := client.GetMetrics()

	if metrics.RequestsTotal != 1 {
		t.Errorf("Expected 1 total request, got %d", metrics.RequestsTotal)
	}

	if metrics.RequestsSuccess != 1 {
		t.Errorf("Expected 1 successful request, got %d", metrics.RequestsSuccess)
	}

	if metrics.RequestsFailure != 0 {
		t.Errorf("Expected 0 failed requests, got %d", metrics.RequestsFailure)
	}

	if metrics.TotalLatency == 0 {
		t.Error("Expected non-zero total latency")
	}
}

// BenchmarkProcessIntent benchmarks intent processing performance
func BenchmarkProcessIntent(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"bench-test\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 50}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		CacheTTL:    0, // Disable cache for benchmark
		Timeout:     30 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)
	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.ProcessIntent(ctx, "Benchmark test request")
			if err != nil {
				b.Errorf("ProcessIntent failed: %v", err)
			}
		}
	})
}

// BenchmarkProcessIntentWithCache benchmarks cached request performance
func BenchmarkProcessIntentWithCache(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"cached-bench\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 50}
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		CacheTTL:    5 * time.Minute,
		Timeout:     30 * time.Second,
	}

	client := NewClientWithConfig(server.URL, config)
	ctx := context.Background()

	// Prime the cache
	client.ProcessIntent(ctx, "Benchmark cached request")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.ProcessIntent(ctx, "Benchmark cached request")
			if err != nil {
				b.Errorf("ProcessIntent failed: %v", err)
			}
		}
	})
}

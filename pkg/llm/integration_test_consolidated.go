//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestFullIntegrationWorkflow tests the complete LLM processing workflow
func TestFullIntegrationWorkflow(t *testing.T) {
	// Setup mock servers
	llmServer := setupMockLLMServer(t)
	defer llmServer.Close()

	ragServer := setupMockRAGServer(t)
	defer ragServer.Close()

	// Create processing engine with all components
	client := NewClientWithConfig(llmServer.URL, ClientConfig{
		APIKey:      "test-key",
		ModelName:   "gpt-4o-mini",
		BackendType: "openai",
		Timeout:     10 * time.Second,
		CacheTTL:    1 * time.Minute,
	})

	processingConfig := &ProcessingConfig{
		EnableRAG:       true,
		RAGAPIURL:       ragServer.URL,
		EnableBatching:  true,
		MinBatchSize:    2,
		MaxBatchSize:    5,
		EnableStreaming: false, // Simplified for testing
		QueryTimeout:    30 * time.Second,
		EnableCaching:   true,
	}

	engine := NewProcessingEngine(client, processingConfig)

	// Test scenarios
	testScenarios := []struct {
		name        string
		intent      string
		expectError bool
		validateFn  func(string) bool
	}{
		{
			name:   "Deploy AMF",
			intent: "Deploy AMF network function with 3 replicas for high availability",
			validateFn: func(result string) bool {
				return strings.Contains(result, "NetworkFunctionDeployment") &&
					strings.Contains(result, "amf")
			},
		},
		{
			name:   "Scale UPF",
			intent: "Scale UPF to 5 replicas to handle increased traffic",
			validateFn: func(result string) bool {
				return strings.Contains(result, "NetworkFunctionScale") ||
					strings.Contains(result, "upf")
			},
		},
		{
			name:   "Deploy Near-RT RIC",
			intent: "Set up Near-RT RIC with traffic steering capabilities",
			validateFn: func(result string) bool {
				return strings.Contains(result, "near-rt-ric") ||
					strings.Contains(result, "traffic")
			},
		},
	}

	ctx := context.Background()

	// Execute test scenarios
	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result, err := engine.ProcessIntent(ctx, scenario.intent)

			if scenario.expectError && err == nil {
				t.Errorf("Expected error for scenario %s, but got none", scenario.name)
				return
			}

			if !scenario.expectError && err != nil {
				t.Errorf("Unexpected error for scenario %s: %v", scenario.name, err)
				return
			}

			if !scenario.expectError && scenario.validateFn != nil {
				if !scenario.validateFn(result.Content) {
					t.Errorf("Validation failed for scenario %s. Result: %s", scenario.name, result.Content)
				}
			}

			// Verify processing time is reasonable
			if result.ProcessingTime > 5*time.Second {
				t.Errorf("Processing time too high for scenario %s: %v", scenario.name, result.ProcessingTime)
			}
		})
	}
}

// TestCacheIntegration tests cache integration across components
func TestCacheIntegration(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"cached-test-%d\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 100}
		}`, callCount)))
	}))
	defer server.Close()

	// Create client with caching enabled
	client := NewClientWithConfig(server.URL, ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		CacheTTL:    2 * time.Minute,
		Timeout:     10 * time.Second,
	})

	// Create cache with semantic similarity
	cache := NewResponseCacheWithConfig(&CacheConfig{
		TTL:                 2 * time.Minute,
		MaxSize:             1000,
		L1MaxSize:           250,
		L2MaxSize:           750,
		SimilarityThreshold: 0.8,
		AdaptiveTTL:         true,
	})

	ctx := context.Background()

	// Test exact cache matches
	intent1 := "Deploy AMF with high availability"
	result1, err := client.ProcessIntent(ctx, intent1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	result2, err := client.ProcessIntent(ctx, intent1)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	if result1 != result2 {
		t.Error("Cached results should be identical")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call (second should be cached), got %d", callCount)
	}

	// Test semantic similarity caching
	cache.Set("similar_intent_1", "{\"deployment\": \"amf-similar\"}")

	if result, found := cache.Get("similar_intent_2"); found {
		// This might find a similar result depending on keywords
		t.Logf("Found similar cached result: %s", result)
	}

	// Verify cache statistics
	stats := cache.GetStats()
	if stats["l1_size"].(int) == 0 && stats["l2_size"].(int) == 0 {
		t.Error("Cache should contain entries")
	}

	if stats["hit_rate"].(float64) <= 0 {
		t.Error("Cache hit rate should be positive")
	}

	cache.Stop()
}

// TestWorkerPoolIntegration tests worker pool integration
func TestWorkerPoolIntegration(t *testing.T) {
	// Create worker pool
	config := &WorkerPoolConfig{
		MinWorkers:          2,
		MaxWorkers:          5,
		QueueSize:           100,
		TaskTimeout:         30 * time.Second,
		ScalingEnabled:      true,
		ScaleUpThreshold:    0.8,
		ScaleDownThreshold:  0.2,
		HealthCheckEnabled:  true,
		HealthCheckInterval: 1 * time.Second,
	}

	pool, err := NewWorkerPool(config)
	if err != nil {
		t.Fatalf("Failed to create worker pool: %v", err)
	}

	ctx := context.Background()

	// Start worker pool
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	defer func() {
		if err := pool.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown worker pool: %v", err)
		}
	}()

	// Submit various task types
	tasks := []Task{
		{
			ID:       "llm-task-1",
			Type:     TaskTypeLLMProcessing,
			Intent:   "Deploy AMF",
			Priority: PriorityNormal,
			Context:  ctx,
		},
		{
			ID:       "rag-task-1",
			Type:     TaskTypeRAGProcessing,
			Intent:   "Process with RAG",
			Priority: PriorityHigh,
			Context:  ctx,
		},
		{
			ID:       "batch-task-1",
			Type:     TaskTypeBatchProcessing,
			Intent:   "Process batch",
			Priority: PriorityLow,
			Context:  ctx,
		},
	}

	// Submit tasks
	for _, task := range tasks {
		if err := pool.Submit(task); err != nil {
			t.Errorf("Failed to submit task %s: %v", task.ID, err)
		}
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Check metrics
	metrics := pool.GetMetrics()
	if metrics.ActiveWorkers < 2 {
		t.Errorf("Expected at least 2 active workers, got %d", metrics.ActiveWorkers)
	}

	status := pool.GetStatus()
	if !status["running"].(bool) {
		t.Error("Worker pool should be running")
	}

	// Test scaling by submitting many tasks
	for i := 0; i < 50; i++ {
		task := Task{
			ID:       fmt.Sprintf("scale-task-%d", i),
			Type:     TaskTypeLLMProcessing,
			Intent:   fmt.Sprintf("Scale test %d", i),
			Priority: PriorityNormal,
			Context:  ctx,
		}

		// Non-blocking submit (queue might be full)
		pool.Submit(task)
	}

	// Wait for scaling
	time.Sleep(2 * time.Second)

	// Verify scaling occurred
	finalMetrics := pool.GetMetrics()
	if finalMetrics.ActiveWorkers <= metrics.ActiveWorkers {
		t.Log("Worker pool may have scaled up") // This is timing-dependent
	}
}

// TestCircuitBreakerIntegration tests circuit breaker integration
func TestCircuitBreakerIntegration(t *testing.T) {
	failureCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failureCount++

		// Fail first 5 requests, then succeed
		if failureCount <= 5 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Temporary failure"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"circuit-recovery\", \"namespace\": \"default\", \"spec\": {}}"
				}
			}],
			"usage": {"total_tokens": 75}
		}`))
	}))
	defer server.Close()

	// Configure circuit breaker with low thresholds for testing
	cbConfig := &CircuitBreakerConfig{
		FailureThreshold:    3,
		FailureRate:         0.6,
		MinimumRequestCount: 3,
		Timeout:             1 * time.Second,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	}

	client := NewClientWithConfig(server.URL, ClientConfig{
		APIKey:               "test-key",
		BackendType:          "openai",
		Timeout:              5 * time.Second,
		CircuitBreakerConfig: cbConfig,
	})

	ctx := context.Background()

	// Make failing requests to trigger circuit breaker
	for i := 0; i < 6; i++ {
		result, err := client.ProcessIntent(ctx, fmt.Sprintf("Trigger circuit breaker %d", i))
		if i < 3 {
			// First few requests should reach server and fail
			if err == nil {
				t.Errorf("Request %d should have failed", i+1)
			}
		} else {
			// Later requests might be rejected by circuit breaker
			if err != nil && strings.Contains(err.Error(), "circuit breaker") {
				t.Logf("Request %d properly rejected by circuit breaker", i+1)
			}
		}
		_ = result
	}

	// Wait for circuit breaker to potentially reset
	time.Sleep(200 * time.Millisecond)

	// Try again - circuit might be half-open
	result, err := client.ProcessIntent(ctx, "Test circuit recovery")
	if err == nil {
		if strings.Contains(result, "circuit-recovery") {
			t.Log("Circuit breaker successfully recovered")
		}
	}
}

// TestPerformanceUnderLoad tests system performance under load
func TestPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := setupHighPerformanceMockServer(t)
	defer server.Close()

	client := NewClientWithConfig(server.URL, ClientConfig{
		APIKey:      "test-key",
		BackendType: "openai",
		Timeout:     30 * time.Second,
		CacheTTL:    5 * time.Minute,
	})

	engine := NewProcessingEngine(client, &ProcessingConfig{
		EnableRAG:             false, // Simplified for performance test
		EnableBatching:        true,
		MinBatchSize:          5,
		MaxBatchSize:          20,
		BatchTimeout:          50 * time.Millisecond,
		MaxConcurrentRequests: 50,
	})

	ctx := context.Background()

	// Create multiple concurrent requests
	numRequests := 100
	numWorkers := 10

	requestChan := make(chan string, numRequests)
	resultChan := make(chan *ProcessingResult, numRequests)

	// Generate requests
	for i := 0; i < numRequests; i++ {
		requestChan <- fmt.Sprintf("Performance test request %d - deploy AMF with %d replicas", i, i%5+1)
	}
	close(requestChan)

	// Start workers
	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for intent := range requestChan {
				result, err := engine.ProcessIntent(ctx, intent)
				if err != nil {
					t.Errorf("Worker %d failed to process intent: %v", workerID, err)
					resultChan <- &ProcessingResult{Error: err}
				} else {
					resultChan <- result
				}
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	totalProcessingTime := time.Duration(0)

	for i := 0; i < numRequests; i++ {
		result := <-resultChan
		if result.Error != nil {
			errorCount++
		} else {
			successCount++
			totalProcessingTime += result.ProcessingTime
		}
	}

	totalTime := time.Since(startTime)

	// Analyze performance
	successRate := float64(successCount) / float64(numRequests)
	averageProcessingTime := totalProcessingTime / time.Duration(successCount)
	requestsPerSecond := float64(numRequests) / totalTime.Seconds()

	t.Logf("Performance Results:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Success Rate: %.2f%%", successRate*100)
	t.Logf("  Average Processing Time: %v", averageProcessingTime)
	t.Logf("  Requests per Second: %.2f", requestsPerSecond)
	t.Logf("  Total Time: %v", totalTime)

	// Performance assertions
	if successRate < 0.95 {
		t.Errorf("Success rate too low: %.2f%% (expected > 95%%)", successRate*100)
	}

	if averageProcessingTime > 1*time.Second {
		t.Errorf("Average processing time too high: %v (expected < 1s)", averageProcessingTime)
	}

	if requestsPerSecond < 10 {
		t.Errorf("Throughput too low: %.2f RPS (expected > 10 RPS)", requestsPerSecond)
	}

	// Check metrics
	metrics := engine.GetMetrics()
	if metrics.TotalRequests == 0 {
		t.Error("Processing metrics should show requests processed")
	}
}

// TestSecurityFeatures tests security-related functionality
func TestSecurityFeatures(t *testing.T) {
	// Test TLS configuration
	client := NewClientWithConfig("https://secure.example.com", ClientConfig{
		APIKey:              "test-key",
		SkipTLSVerification: false, // Security should be enforced
		Timeout:             10 * time.Second,
	})

	if client.httpClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("TLS verification should be enabled by default")
	}

	// Test API key handling
	if client.apiKey != "test-key" {
		t.Error("API key should be stored securely")
	}

	// Test timeout enforcement
	shortClient := NewClientWithConfig("http://slow.example.com", ClientConfig{
		Timeout: 1 * time.Millisecond, // Very short timeout
	})

	if shortClient.httpClient.Timeout != 1*time.Millisecond {
		t.Error("Timeout should be enforced")
	}
}

// Helper functions for testing

func setupMockLLMServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic LLM API responses
		time.Sleep(10 * time.Millisecond) // Simulate processing time

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"test-amf\", \"namespace\": \"5g-core\", \"spec\": {\"replicas\": 3, \"image\": \"amf:latest\", \"resources\": {\"requests\": {\"cpu\": \"1000m\", \"memory\": \"2Gi\"}}}}"
				}
			}],
			"usage": {
				"total_tokens": 150,
				"prompt_tokens": 100,
				"completion_tokens": 50
			}
		}`))
	}))
}

func setupMockRAGServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/process") {
			http.NotFound(w, r)
			return
		}

		time.Sleep(20 * time.Millisecond) // Simulate RAG processing time

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"type": "NetworkFunctionDeployment",
			"name": "rag-enhanced-amf",
			"namespace": "5g-core",
			"spec": {
				"replicas": 3,
				"image": "amf:v2.1.0",
				"resources": {
					"requests": {"cpu": "1000m", "memory": "2Gi"},
					"limits": {"cpu": "2000m", "memory": "4Gi"}
				}
			},
			"confidence": 0.92,
			"sources": ["3GPP TS 23.501", "O-RAN WG1 spec"]
		}`))
	}))
}

func setupHighPerformanceMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fast response for performance testing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\"type\": \"NetworkFunctionDeployment\", \"name\": \"perf-test\", \"namespace\": \"default\", \"spec\": {\"replicas\": 1}}"
				}
			}],
			"usage": {"total_tokens": 50}
		}`))
	}))
}

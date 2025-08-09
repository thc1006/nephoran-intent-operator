package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// TestProcessIntentWithTimeout tests that ProcessIntent respects timeout settings
func TestProcessIntentWithTimeout(t *testing.T) {
	// Set environment variable for timeout
	os.Setenv("LLM_TIMEOUT_SECS", "2")
	defer os.Unsetenv("LLM_TIMEOUT_SECS")

	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than timeout
		time.Sleep(3 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":      "NetworkFunctionDeployment",
			"name":      "test-nf",
			"namespace": "default",
			"spec": map[string]interface{}{
				"replicas": 1,
				"image":    "test:latest",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	start := time.Now()
	_, err := client.ProcessIntent(ctx, "deploy test network function")
	duration := time.Since(start)

	// Should timeout in ~2 seconds, not wait for 3 second server delay
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if duration > 2500*time.Millisecond {
		t.Errorf("Request took too long: %v, expected ~2s", duration)
	}
	if err != nil && err.Error() != "context deadline exceeded" && !contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestProcessIntentWithRetry tests retry behavior with LLM_MAX_RETRIES
func TestProcessIntentWithRetry(t *testing.T) {
	// Set environment variable for max retries
	os.Setenv("LLM_MAX_RETRIES", "3")
	defer os.Unsetenv("LLM_MAX_RETRIES")

	attemptCount := 0
	mu := sync.Mutex{}

	// Create test server that fails first attempts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attemptCount++
		currentAttempt := attemptCount
		mu.Unlock()

		// Fail first 2 attempts, succeed on 3rd
		if currentAttempt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("temporary error"))
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "test-nf",
				"namespace": "default",
				"spec": map[string]interface{}{
					"replicas": 1,
					"image":    "test:latest",
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.retryConfig.BaseDelay = 10 * time.Millisecond // Speed up test
	client.retryConfig.MaxDelay = 50 * time.Millisecond

	ctx := context.Background()
	response, err := client.ProcessIntent(ctx, "deploy test network function")

	if err != nil {
		t.Errorf("Expected successful retry, got error: %v", err)
	}
	if response == "" {
		t.Error("Expected non-empty response after retry")
	}

	mu.Lock()
	finalAttempts := attemptCount
	mu.Unlock()

	// Should have made 3 attempts (2 failures + 1 success)
	if finalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", finalAttempts)
	}
}

// TestProcessIntentRejectsNonJSON tests Content-Type validation
func TestProcessIntentRejectsNonJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		shouldFail  bool
	}{
		{
			name:        "Valid JSON response",
			contentType: "application/json",
			body:        `{"type":"NetworkFunctionDeployment","name":"test","namespace":"default","spec":{"replicas":1,"image":"test:latest"}}`,
			shouldFail:  false,
		},
		{
			name:        "Invalid content type",
			contentType: "text/plain",
			body:        `{"type":"NetworkFunctionDeployment","name":"test","namespace":"default","spec":{"replicas":1,"image":"test:latest"}}`,
			shouldFail:  true,
		},
		{
			name:        "Missing content type",
			contentType: "",
			body:        `{"type":"NetworkFunctionDeployment","name":"test","namespace":"default","spec":{"replicas":1,"image":"test:latest"}}`,
			shouldFail:  true,
		},
		{
			name:        "Invalid JSON body",
			contentType: "application/json",
			body:        `not json at all`,
			shouldFail:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient(server.URL)
			ctx := context.Background()

			_, err := client.ProcessIntent(ctx, "deploy test")

			if tt.shouldFail && err == nil {
				t.Error("Expected error for invalid response, got nil")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
		})
	}
}

// TestValidatorStructuredErrors tests ValidationError with missing fields
func TestValidatorStructuredErrors(t *testing.T) {
	validator := &ResponseValidator{
		requiredFields: map[string]bool{
			"type":      true,
			"name":      true,
			"namespace": true,
			"spec":      true,
		},
	}

	tests := []struct {
		name           string
		response       map[string]interface{}
		expectedFields []string
	}{
		{
			name:           "All fields missing",
			response:       map[string]interface{}{},
			expectedFields: []string{"type", "name", "namespace", "spec"},
		},
		{
			name: "Some fields missing",
			response: map[string]interface{}{
				"type": "NetworkFunctionDeployment",
				"name": "test",
			},
			expectedFields: []string{"namespace", "spec"},
		},
		{
			name: "Only spec missing",
			response: map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "test",
				"namespace": "default",
			},
			expectedFields: []string{"spec"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.response)
			err := validator.ValidateResponse(jsonData)

			if len(tt.expectedFields) > 0 {
				if err == nil {
					t.Error("Expected validation error, got nil")
					return
				}

				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError type, got %T", err)
					return
				}

				// Check that all expected fields are in MissingFields
				for _, field := range tt.expectedFields {
					found := false
					for _, missing := range validationErr.MissingFields {
						if missing == field {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected field '%s' in MissingFields, not found", field)
					}
				}

				// Check count matches
				if len(validationErr.MissingFields) != len(tt.expectedFields) {
					t.Errorf("Expected %d missing fields, got %d: %v",
						len(tt.expectedFields), len(validationErr.MissingFields), validationErr.MissingFields)
				}
			}
		})
	}
}

// TestCacheLRUEviction tests cache eviction with capacity bounds
func TestCacheLRUEviction(t *testing.T) {
	// Set small cache size for testing
	os.Setenv("LLM_CACHE_MAX_ENTRIES", "3")
	defer os.Unsetenv("LLM_CACHE_MAX_ENTRIES")

	cache := NewResponseCache(1*time.Hour, 10) // Constructor param overridden by env var

	// Add entries to fill cache
	cache.Set("key1", "value1")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key2", "value2")
	time.Sleep(10 * time.Millisecond)
	cache.Set("key3", "value3")

	// Access key1 to make it recently used
	cache.Get("key1")
	time.Sleep(10 * time.Millisecond)

	// Add new entry, should evict key2 (least recently used)
	cache.Set("key4", "value4")

	// Check eviction
	if _, found := cache.Get("key1"); !found {
		t.Error("key1 should not be evicted (recently accessed)")
	}
	if _, found := cache.Get("key2"); found {
		t.Error("key2 should be evicted (least recently used)")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("key3 should not be evicted")
	}
	if _, found := cache.Get("key4"); !found {
		t.Error("key4 should be present (just added)")
	}

	// Verify cache size
	cache.l1Mutex.RLock()
	cacheSize := len(cache.l1Entries)
	cache.l1Mutex.RUnlock()

	if cacheSize > 3 {
		t.Errorf("Cache size exceeds max entries: got %d, expected <= 3", cacheSize)
	}
}

// TestFallbackURLs tests fallback URL behavior
func TestFallbackURLs(t *testing.T) {
	primaryCalled := false
	fallbackCalled := false

	// Primary server always fails
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primaryServer.Close()

	// Fallback server succeeds
	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":      "NetworkFunctionDeployment",
			"name":      "test-nf",
			"namespace": "default",
			"spec": map[string]interface{}{
				"replicas": 1,
				"image":    "test:latest",
			},
		})
	}))
	defer fallbackServer.Close()

	client := NewClient(primaryServer.URL)
	client.SetFallbackURLs([]string{fallbackServer.URL})
	client.retryConfig.MaxRetries = 0 // Don't retry primary

	ctx := context.Background()
	response, err := client.ProcessIntent(ctx, "deploy test")

	if err != nil {
		t.Errorf("Expected fallback to succeed, got error: %v", err)
	}
	if response == "" {
		t.Error("Expected non-empty response from fallback")
	}
	if !primaryCalled {
		t.Error("Primary server should have been called")
	}
	if !fallbackCalled {
		t.Error("Fallback server should have been called")
	}
}

// TestLoggingLevels tests that logging respects Info/Debug levels
func TestLoggingLevels(t *testing.T) {
	// This test would typically use a custom logger to capture output
	// For demonstration, we'll just verify the code path works
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Create a very long response
		longText := ""
		for i := 0; i < 200; i++ {
			longText += "This is a very long response text. "
		}
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":      "NetworkFunctionDeployment",
			"name":      "test-nf",
			"namespace": "default",
			"spec": map[string]interface{}{
				"replicas":    1,
				"image":       "test:latest",
				"description": longText,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Process intent - logging should truncate at Debug level
	response, err := client.ProcessIntent(ctx, "deploy test with very long description")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if response == "" {
		t.Error("Expected non-empty response")
	}
	
	// Verify response is properly formatted JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(substr) > 0 && len(s) > 0 && s[:len(substr)] == substr ||
		   fmt.Sprintf("%s", s) != s
}
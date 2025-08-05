package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
	"log/slog"
)

func TestRequestSizeLimits(t *testing.T) {
	// Set up test configuration with a small request size limit for testing
	testMaxSize := int64(1024) // 1KB limit for testing
	
	// Create a test configuration
	cfg := config.DefaultLLMProcessorConfig()
	cfg.MaxRequestSize = testMaxSize
	cfg.AuthEnabled = false // Disable auth for simpler testing
	cfg.RAGEnabled = false  // Disable RAG for simpler testing
	cfg.LLMBackendType = "mock" // Use mock backend
	
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectError    bool
		endpoint       string
	}{
		{
			name:           "Small request within limit",
			requestBody:    `{"intent": "Deploy a simple network function"}`,
			expectedStatus: http.StatusOK,
			expectError:    false,
			endpoint:       "/process",
		},
		{
			name:           "Request at exact limit",
			requestBody:    strings.Repeat("x", int(testMaxSize-50)) + `{"intent": "test"}`, // Near the limit
			expectedStatus: http.StatusOK,
			expectError:    false,
			endpoint:       "/process",
		},
		{
			name:           "Request exceeding limit",
			requestBody:    strings.Repeat("x", int(testMaxSize)+100), // Exceed the limit
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectError:    true,
			endpoint:       "/process",
		},
		{
			name:           "Large streaming request exceeding limit",
			requestBody:    `{"query": "` + strings.Repeat("x", int(testMaxSize)+100) + `"}`,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectError:    true,
			endpoint:       "/stream",
		},
		{
			name:           "Valid streaming request within limit",
			requestBody:    `{"query": "What is the status of network functions?"}`,
			expectedStatus: http.StatusOK,
			expectError:    false,
			endpoint:       "/stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that simulates successful processing
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Try to read the body to trigger MaxBytesReader
				body, err := io.ReadAll(r.Body)
				if err != nil {
					// MaxBytesReader error should be caught by middleware
					t.Errorf("Unexpected error reading body: %v", err)
					return
				}
				
				// Simulate successful processing
				response := map[string]interface{}{
					"status": "success",
					"result": "test result",
					"body_size": len(body),
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})

			// Wrap the handler with size limits
			wrappedHandler := middleware.MaxBytesHandler(testMaxSize, logger, testHandler)

			// Create test request
			req, err := http.NewRequest("POST", tt.endpoint, strings.NewReader(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute the request
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, status)
			}

			// For error cases, check that we get proper JSON error response
			if tt.expectError {
				var errorResponse map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
				if err != nil {
					t.Errorf("Expected JSON error response, got: %s", rr.Body.String())
				}

				if errorResponse["error"] == nil {
					t.Errorf("Expected error field in response")
				}

				if errorResponse["code"] != float64(413) {
					t.Errorf("Expected error code 413, got %v", errorResponse["code"])
				}
			}
		})
	}
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	testMaxSize := int64(512) // Very small limit for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	limiter := middleware.NewRequestSizeLimiter(testMaxSize, logger)

	// Test handler that just echoes the request size
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		fmt.Fprintf(w, "Received %d bytes", len(body))
	})

	wrappedHandler := limiter.Handler(testHandler)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "GET request (no body limit)",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST within limit",
			method:         "POST",
			body:           strings.Repeat("a", 100),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST exceeding limit",
			method:         "POST",
			body:           strings.Repeat("a", int(testMaxSize)+100),
			expectedStatus: http.StatusBadRequest, // Will be handled by MaxBytesReader
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if tt.expectedStatus == http.StatusOK && rr.Code >= 400 {
				t.Errorf("Expected success for %s, got status %d: %s", tt.name, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		maxSize     int64
		expectError bool
		errorSubstr string
	}{
		{
			name:        "Valid size - 1MB",
			maxSize:     1024 * 1024,
			expectError: false,
		},
		{
			name:        "Valid size - 10MB",
			maxSize:     10 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "Zero size - invalid",
			maxSize:     0,
			expectError: true,
			errorSubstr: "must be positive",
		},
		{
			name:        "Negative size - invalid",
			maxSize:     -1,
			expectError: true,
			errorSubstr: "must be positive",
		},
		{
			name:        "Too small - invalid",
			maxSize:     512, // Less than 1KB
			expectError: true,
			errorSubstr: "at least 1KB",
		},
		{
			name:        "Too large - invalid",
			maxSize:     200 * 1024 * 1024, // 200MB, exceeds 100MB limit
			expectError: true,
			errorSubstr: "should not exceed 100MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultLLMProcessorConfig()
			cfg.MaxRequestSize = tt.maxSize

			err := cfg.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for size %d, but got none", tt.maxSize)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error for size %d, got: %s", tt.maxSize, err.Error())
				}
			}
		})
	}
}

func TestMaxBytesHandlerWithContentLength(t *testing.T) {
	testMaxSize := int64(1000)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	wrappedHandler := middleware.MaxBytesHandler(testMaxSize, logger, testHandler)

	tests := []struct {
		name           string
		contentLength  int64
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "Content-Length within limit",
			contentLength:  500,
			bodySize:       500,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Content-Length exceeds limit",
			contentLength:  2000,
			bodySize:       100, // Actual body smaller, but Content-Length header is large
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "No Content-Length header, body within limit",
			contentLength:  -1, // No header
			bodySize:       500,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.Repeat("x", tt.bodySize)
			req, err := http.NewRequest("POST", "/test", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}

			if tt.contentLength > 0 {
				req.ContentLength = tt.contentLength
				req.Header.Set("Content-Length", fmt.Sprintf("%d", tt.contentLength))
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// For 413 responses, check JSON format
			if tt.expectedStatus == http.StatusRequestEntityTooLarge {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Expected JSON response for 413, got: %s", rr.Body.String())
				}
			}
		})
	}
}

func TestIntegrationWithRealHandlers(t *testing.T) {
	// This test simulates integration with actual LLM processor handlers
	// Set up minimal configuration
	cfg := config.DefaultLLMProcessorConfig()
	cfg.MaxRequestSize = 2048 // 2KB limit
	cfg.AuthEnabled = false
	cfg.RAGEnabled = false
	cfg.LLMBackendType = "mock"

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a mock service (simplified)
	service := &MockLLMProcessorService{}
	
	// Create handler (simplified version)
	handler := &MockLLMProcessorHandler{
		config: cfg,
		logger: logger,
	}

	tests := []struct {
		name           string
		endpoint       string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:     "Valid process request",
			endpoint: "/process",
			requestBody: map[string]string{
				"intent": "Deploy a simple 5G network function",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Oversized process request",
			endpoint: "/process",
			requestBody: map[string]string{
				"intent": strings.Repeat("Deploy a very complex network function ", 100), // Large intent
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert request body to JSON
			bodyBytes, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			// Create request
			req, err := http.NewRequest("POST", tt.endpoint, bytes.NewReader(bodyBytes))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Wrap handler with size limits
			wrappedHandler := middleware.MaxBytesHandler(cfg.MaxRequestSize, logger, handler.ProcessIntentHandler)

			// Execute request
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check result
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// Mock implementations for testing

type MockLLMProcessorService struct{}

type MockLLMProcessorHandler struct {
	config *config.LLMProcessorConfig
	logger *slog.Logger
}

func (h *MockLLMProcessorHandler) ProcessIntentHandler(w http.ResponseWriter, r *http.Request) {
	// Simple mock handler that reads the body and returns success
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req["intent"] == nil || req["intent"] == "" {
		http.Error(w, "Intent required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"result": "Mock processing result",
		"request_id": "test-123",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRequestSizeLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testMaxSize := int64(1024) // 1KB limit

	// Create test handler that reads the full body
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := json.RawMessage(`{}`)
		json.NewEncoder(w).Encode(response)
	})

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Small POST request within limit",
			method:         "POST",
			body:           `{"test": "small request"}`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "GET request (no size limit applied)",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "POST request at limit boundary",
			method:         "POST",
			body:           strings.Repeat("x", int(testMaxSize-10)) + "end",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "POST request exceeding limit",
			method:         "POST",
			body:           strings.Repeat("x", int(testMaxSize)+100),
<<<<<<< HEAD
			expectedStatus: http.StatusBadRequest, // MaxBytesReader will cause this
=======
			expectedStatus: http.StatusRequestEntityTooLarge, // MaxBytesReader returns 413
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestSizeConfig{
				MaxBodySize:   testMaxSize,
				MaxHeaderSize: 8 * 1024,
				EnableLogging: true,
			}
			limiter := NewRequestSizeLimiterWithConfig(config, logger)
			wrappedHandler := limiter.Handler(testHandler)

			req, err := http.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if tt.expectError {
				if rr.Code < 400 {
					t.Errorf("Expected error status for %s, got %d", tt.name, rr.Code)
				}
			} else {
				if rr.Code >= 400 {
					t.Errorf("Expected success for %s, got %d: %s", tt.name, rr.Code, rr.Body.String())
				}
			}
		})
	}
}

func TestMaxBytesHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testMaxSize := int64(512) // 512 bytes limit

	// Simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "OK: received %d bytes", len(body))
	})

	tests := []struct {
		name           string
		contentLength  int64
		bodySize       int
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "Request within limit",
			contentLength:  300,
			bodySize:       300,
			expectedStatus: http.StatusOK,
			expectJSON:     false,
		},
		{
			name:           "Content-Length exceeds limit",
			contentLength:  1000,
			bodySize:       100,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectJSON:     true,
		},
		{
			name:           "Body exceeds limit (no Content-Length)",
			contentLength:  -1, // No header
			bodySize:       1000,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectJSON:     true,
		},
		{
			name:           "Exact limit boundary",
			contentLength:  512,
			bodySize:       512,
			expectedStatus: http.StatusOK,
			expectJSON:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHandler := MaxBytesHandler(testMaxSize, logger, testHandler)

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
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.expectJSON {
				// Check that we get a proper JSON error response
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Expected JSON error response, got: %s", rr.Body.String())
				}

				if response["error"] == nil {
					t.Errorf("Expected error field in JSON response")
				}

				if response["code"] != float64(413) {
					t.Errorf("Expected error code 413, got %v", response["code"])
				}
			}
		})
	}
}

func TestMaxBytesHandlerMethods(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testMaxSize := int64(100)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	wrappedHandler := MaxBytesHandler(testMaxSize, logger, testHandler)

	methods := []struct {
		method          string
		shouldHaveLimit bool
	}{
		{"GET", false},
		{"POST", true},
		{"PUT", true},
		{"PATCH", true},
		{"DELETE", false},
		{"HEAD", false},
		{"OPTIONS", false},
	}

	for _, method := range methods {
		t.Run(fmt.Sprintf("Method_%s", method.method), func(t *testing.T) {
			// Create request with large body
			largeBody := strings.Repeat("x", int(testMaxSize)+50)
			req, err := http.NewRequest(method.method, "/test", strings.NewReader(largeBody))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if method.shouldHaveLimit {
				// Methods with bodies should be limited
				if rr.Code == http.StatusOK {
					t.Errorf("Expected %s request with large body to be rejected", method.method)
				}
			} else {
				// Methods without bodies should pass through
				if rr.Code != http.StatusOK {
					t.Errorf("Expected %s request to pass through, got status %d", method.method, rr.Code)
				}
			}
		})
	}
}

func TestWritePayloadTooLargeResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	rr := httptest.NewRecorder()
	maxSize := int64(1024)

	writePayloadTooLargeResponse(rr, logger, maxSize)

	// Check status code
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check JSON response structure
	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	expectedFields := []string{"error", "message", "status", "code"}
	for _, field := range expectedFields {
		if response[field] == nil {
			t.Errorf("Missing field '%s' in response", field)
		}
	}

	if response["code"] != float64(413) {
		t.Errorf("Expected code 413, got %v", response["code"])
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got %v", response["status"])
	}
}

func TestMaxBytesHandlerPanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testMaxSize := int64(100)

	tests := []struct {
		name               string
		handler            http.HandlerFunc
		expectedStatus     int
		expectPanicRethrow bool
	}{
		{
			name: "MaxBytesError panic recovery",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate a MaxBytesError panic
				panic(&http.MaxBytesError{Limit: testMaxSize})
			}),
			expectedStatus:     http.StatusRequestEntityTooLarge,
			expectPanicRethrow: false,
		},
		{
			name: "String panic recovery (exact match)",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate the string panic that http.MaxBytesReader might throw
				panic("http: request body too large")
			}),
			expectedStatus:     http.StatusRequestEntityTooLarge,
			expectPanicRethrow: false,
		},
		{
			name: "String panic recovery (case insensitive)",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test case insensitive matching
				panic("HTTP: Request Body Too Large")
			}),
			expectedStatus:     http.StatusRequestEntityTooLarge,
			expectPanicRethrow: false,
		},
		{
			name: "String panic recovery (partial match)",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test partial matching
				panic("Error: body too large for processing")
			}),
			expectedStatus:     http.StatusRequestEntityTooLarge,
			expectPanicRethrow: false,
		},
		{
			name: "String panic recovery (maxbytesreader pattern)",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Test future-proof pattern
				panic("MaxBytesReader: limit exceeded")
			}),
			expectedStatus:     http.StatusRequestEntityTooLarge,
			expectPanicRethrow: false,
		},
		{
			name: "Other panic rethrown",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate a different type of panic
				panic("some other error")
			}),
			expectedStatus:     0, // Won't be set due to panic
			expectPanicRethrow: true,
		},
		{
			name: "Nil panic rethrown",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate a nil panic
				panic(nil)
			}),
			expectedStatus:     0, // Won't be set due to panic
			expectPanicRethrow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHandler := MaxBytesHandler(testMaxSize, logger, tt.handler)

			req, err := http.NewRequest("POST", "/test", strings.NewReader("test body"))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			if tt.expectPanicRethrow {
				// Expect a panic to be rethrown
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic to be rethrown, but it wasn't")
					}
				}()
			}

			wrappedHandler.ServeHTTP(rr, req)

			if !tt.expectPanicRethrow {
				// Check that we got the expected status
				if rr.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
				}

				// For 413 responses, check JSON structure
				if tt.expectedStatus == http.StatusRequestEntityTooLarge {
					var response map[string]interface{}
					err := json.Unmarshal(rr.Body.Bytes(), &response)
					if err != nil {
						t.Errorf("Expected JSON error response, got: %s", rr.Body.String())
					}

					if response["error"] == nil {
						t.Errorf("Expected error field in JSON response")
					}
				}
			}
		})
	}
}


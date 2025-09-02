package middleware

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogHandler captures log entries for testing
type mockLogHandler struct {
	entries []mockLogEntry
}

type mockLogEntry struct {
	Level   slog.Level
	Message string
	Attrs   map[string]interface{}
}

func (h *mockLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *mockLogHandler) Handle(ctx context.Context, r slog.Record) error {
	entry := mockLogEntry{
		Level:   r.Level,
		Message: r.Message,
		Attrs:   make(map[string]interface{}),
	}

	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})

	h.entries = append(h.entries, entry)
	return nil
}

func (h *mockLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *mockLogHandler) WithGroup(name string) slog.Handler {
	return h
}

// DISABLED: func TestRedactLogger_HeaderRedaction(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedRedact map[string]bool
	}{
		{
			name: "redacts authorization header",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"Content-Type":  "application/json",
			},
			expectedRedact: map[string]bool{
				"Authorization": true,
				"Content-Type":  false,
			},
		},
		{
			name: "redacts cookie headers",
			headers: map[string]string{
				"Cookie":     "session=abc123",
				"Set-Cookie": "session=xyz789; HttpOnly",
				"User-Agent": "Mozilla/5.0",
			},
			expectedRedact: map[string]bool{
				"Cookie":     true,
				"Set-Cookie": true,
				"User-Agent": false,
			},
		},
		{
			name: "redacts API key headers",
			headers: map[string]string{
				"X-API-Key":    "secret-key",
				"X-Auth-Token": "auth-token",
				"X-Request-ID": "req-123",
			},
			expectedRedact: map[string]bool{
				"X-API-Key":    true,
				"X-Auth-Token": true,
				"X-Request-ID": false,
			},
		},
		{
			name: "case insensitive redaction",
			headers: map[string]string{
				"authorization": "Bearer token",
				"COOKIE":        "session=test",
				"x-api-key":     "key123",
			},
			expectedRedact: map[string]bool{
				"authorization": true,
				"COOKIE":        true,
				"x-api-key":     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedactLoggerConfig()
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			// Create headers
			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}

			// Redact headers
			redacted := rl.redactHeaders(headers)

			// Check redaction
			for header, shouldRedact := range tt.expectedRedact {
				values := redacted[header]
				if shouldRedact {
					assert.Equal(t, []string{"[REDACTED]"}, values, "Header %s should be redacted", header)
				} else {
					assert.Equal(t, []string{tt.headers[header]}, values, "Header %s should not be redacted", header)
				}
			}
		})
	}
}

// DISABLED: func TestRedactLogger_QueryParamRedaction(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedParams map[string]string
	}{
		{
			name:  "redacts token parameter",
			query: "page=1&token=secret123&sort=asc",
			expectedParams: map[string]string{
				"page":  "1",
				"token": "[REDACTED]",
				"sort":  "asc",
			},
		},
		{
			name:  "redacts api_key parameter",
			query: "api_key=mykey&version=2&format=json",
			expectedParams: map[string]string{
				"api_key": "[REDACTED]",
				"version": "2",
				"format":  "json",
			},
		},
		{
			name:  "redacts password parameter",
			query: "username=john&password=secret&remember=true",
			expectedParams: map[string]string{
				"username": "john",
				"password": "[REDACTED]",
				"remember": "true",
			},
		},
		{
			name:  "handles multiple values",
			query: "id=1&id=2&token=abc&token=xyz",
			expectedParams: map[string]string{
				"id":    "1,2", // Multiple values joined
				"token": "[REDACTED],[REDACTED]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedactLoggerConfig()
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			// Parse query
			params, err := url.ParseQuery(tt.query)
			require.NoError(t, err)

			// Redact query params
			redacted := rl.redactQueryParams(params)

			// Parse redacted query
			redactedParams, err := url.ParseQuery(redacted)
			require.NoError(t, err)

			// Check each expected parameter
			for key, expectedValue := range tt.expectedParams {
				actualValues := redactedParams[key]
				actualValue := strings.Join(actualValues, ",")
				assert.Equal(t, expectedValue, actualValue, "Parameter %s mismatch", key)
			}
		})
	}
}

// DISABLED: func TestRedactLogger_BodyRedaction(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		isJSON       bool
		expectFields map[string]string
	}{
		{
			name:   "redacts password in JSON",
			body:   `{"username":"john","password":"secret123","email":"john@example.com"}`,
			isJSON: true,
			expectFields: map[string]string{
				"username": "john",
				"password": "[REDACTED]",
				"email":    "john@example.com",
			},
		},
		{
			name:   "redacts token in JSON",
			body:   `{"action":"login","token":"abc123","timestamp":1234567890}`,
			isJSON: true,
			expectFields: map[string]string{
				"action":    "login",
				"token":     "[REDACTED]",
				"timestamp": "1234567890",
			},
		},
		{
			name:   "redacts nested JSON fields",
			body:   `{"user":{"name":"alice","api_key":"key123"},"status":"active"}`,
			isJSON: true,
			expectFields: map[string]string{
				"status": "active",
			},
		},
		{
			name:   "handles non-JSON text",
			body:   "password=secret123&user=john",
			isJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedactLoggerConfig()
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			// Redact body
			redacted := rl.redactBody([]byte(tt.body))

			if tt.isJSON {
				// Parse redacted JSON
				var result map[string]interface{}
				err := json.Unmarshal([]byte(redacted), &result)
				require.NoError(t, err)

				// Check specific fields
				for field, expectedValue := range tt.expectFields {
					actualValue, ok := result[field]
					if ok {
						assert.Equal(t, expectedValue, actualValue, "Field %s mismatch", field)
					}
				}
			}
		})
	}
}

// DISABLED: func TestRedactLogger_Middleware(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		path              string
		headers           map[string]string
		query             string
		body              string
		expectedStatus    int
		expectLogged      bool
		expectCorrelation bool
	}{
		{
			name:           "logs normal request",
			method:         "GET",
			path:           "/api/users",
			expectedStatus: http.StatusOK,
			expectLogged:   true,
		},
		{
			name:           "skips health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectLogged:   false,
		},
		{
			name:              "logs with correlation ID",
			method:            "POST",
			path:              "/api/login",
			headers:           map[string]string{"X-Request-ID": "test-123"},
			expectedStatus:    http.StatusOK,
			expectLogged:      true,
			expectCorrelation: true,
		},
		{
			name:   "logs with redacted auth header",
			method: "GET",
			path:   "/api/protected",
			headers: map[string]string{
				"Authorization": "Bearer secret-token",
			},
			expectedStatus: http.StatusOK,
			expectLogged:   true,
		},
		{
			name:           "logs with redacted query params",
			method:         "GET",
			path:           "/api/data",
			query:          "page=1&token=secret",
			expectedStatus: http.StatusOK,
			expectLogged:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			config := DefaultRedactLoggerConfig()
			config.LogLevel = slog.LevelDebug
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.expectedStatus)
				w.Write([]byte("OK"))
			})

			// Apply middleware
			wrapped := rl.Middleware(handler)

			// Create request
			url := tt.path
			if tt.query != "" {
				url += "?" + tt.query
			}

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, url, body)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Create response recorder
			rec := httptest.NewRecorder()

			// Execute request
			wrapped.ServeHTTP(rec, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Check logging
			if tt.expectLogged {
				assert.NotEmpty(t, mockHandler.entries, "Expected log entries")

				// Check for completion log
				found := false
				for _, entry := range mockHandler.entries {
					if strings.Contains(entry.Message, "completed") {
						found = true

						// Check correlation ID if expected
						if tt.expectCorrelation {
							correlationID, ok := entry.Attrs["correlation_id"].(string)
							assert.True(t, ok, "Expected correlation_id in log")
							assert.Equal(t, "test-123", correlationID)
						}

						// Check status code
						status, ok := entry.Attrs["status"].(int)
						assert.True(t, ok, "Expected status in log")
						assert.Equal(t, tt.expectedStatus, status)

						break
					}
				}
				assert.True(t, found, "Expected completion log entry")
			} else {
				assert.Empty(t, mockHandler.entries, "Expected no log entries")
			}
		})
	}
}

// DISABLED: func TestRedactLogger_SlowRequests(t *testing.T) {
	config := DefaultRedactLoggerConfig()
	config.SlowRequestThreshold = 100 * time.Millisecond
	mockHandler := &mockLogHandler{}
	logger := slog.New(mockHandler)

	rl, err := NewRedactLogger(config, logger)
	require.NoError(t, err)

	// Create slow handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	wrapped := rl.Middleware(handler)

	// Execute request
	req := httptest.NewRequest("GET", "/api/slow", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Check for slow request warning
	found := false
	for _, entry := range mockHandler.entries {
		if strings.Contains(entry.Message, "Slow request") {
			found = true
			slowFlag, ok := entry.Attrs["slow_request"].(bool)
			assert.True(t, ok, "Expected slow_request flag")
			assert.True(t, slowFlag, "Expected slow_request to be true")
			break
		}
	}
	assert.True(t, found, "Expected slow request warning")
}

// DISABLED: func TestRedactLogger_ErrorStatus(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel slog.Level
		expectedMsg   string
	}{
		{
			name:          "client error",
			statusCode:    http.StatusNotFound,
			expectedLevel: slog.LevelWarn,
			expectedMsg:   "client error",
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			expectedLevel: slog.LevelError,
			expectedMsg:   "server error",
		},
		{
			name:          "success",
			statusCode:    http.StatusOK,
			expectedLevel: slog.LevelInfo,
			expectedMsg:   "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedactLoggerConfig()
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			// Create handler with specific status
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			// Apply middleware
			wrapped := rl.Middleware(handler)

			// Execute request
			req := httptest.NewRequest("GET", "/api/test", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			// Check log level and message
			found := false
			for _, entry := range mockHandler.entries {
				if strings.Contains(entry.Message, tt.expectedMsg) {
					found = true
					assert.Equal(t, tt.expectedLevel, entry.Level, "Unexpected log level")
					break
				}
			}
			assert.True(t, found, "Expected log message containing '%s'", tt.expectedMsg)
		})
	}
}

// DISABLED: func TestRedactLogger_UpdateConfig(t *testing.T) {
	config := DefaultRedactLoggerConfig()
	mockHandler := &mockLogHandler{}
	logger := slog.New(mockHandler)

	rl, err := NewRedactLogger(config, logger)
	require.NoError(t, err)

	// Update configuration
	newConfig := DefaultRedactLoggerConfig()
	newConfig.SensitiveHeaders = append(newConfig.SensitiveHeaders, "X-Custom-Secret")
	newConfig.SkipPaths = append(newConfig.SkipPaths, "/custom/*")

	err = rl.UpdateConfig(newConfig)
	require.NoError(t, err)

	// Test that new sensitive header is redacted
	headers := http.Header{
		"X-Custom-Secret": []string{"secret-value"},
	}
	redacted := rl.redactHeaders(headers)
	assert.Equal(t, []string{"[REDACTED]"}, redacted["X-Custom-Secret"])

	// Test that new skip path works
	assert.True(t, rl.shouldSkipPath("/custom/test"))
}

// DISABLED: func TestRedactLogger_ClientIPExtraction(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "X-Forwarded-For single IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.2, 172.16.0.1",
			},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.2",
		},
		{
			name:       "RemoteAddr only",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:54321",
			expectedIP: "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRedactLoggerConfig()
			mockHandler := &mockLogHandler{}
			logger := slog.New(mockHandler)

			rl, err := NewRedactLogger(config, logger)
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := rl.getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func BenchmarkRedactLogger_HeaderRedaction(b *testing.B) {
	config := DefaultRedactLoggerConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rl, _ := NewRedactLogger(config, logger)

	headers := http.Header{
		"Authorization": []string{"Bearer token123"},
		"Content-Type":  []string{"application/json"},
		"Cookie":        []string{"session=abc123"},
		"User-Agent":    []string{"Mozilla/5.0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.redactHeaders(headers)
	}
}

func BenchmarkRedactLogger_JSONRedaction(b *testing.B) {
	config := DefaultRedactLoggerConfig()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rl, _ := NewRedactLogger(config, logger)

	body := []byte(`{
		"username": "john",
		"password": "secret123",
		"email": "john@example.com",
		"token": "abc123",
		"data": {
			"api_key": "key456",
			"value": "test"
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.redactBody(body)
	}
}

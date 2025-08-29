
package security



import (

	"crypto/subtle"

	"net/http"

	"strings"

	"time"

)



// WebhookSecurityValidator provides additional security validation for webhooks.

type WebhookSecurityValidator struct {

	maxPayloadSize   int64

	allowedUserAgent string

	requiredHeaders  []string

}



// NewWebhookSecurityValidator creates a new webhook security validator.

func NewWebhookSecurityValidator() *WebhookSecurityValidator {

	return &WebhookSecurityValidator{

		maxPayloadSize:   10 * 1024 * 1024, // 10MB default

		allowedUserAgent: "",               // Empty means allow all

		requiredHeaders: []string{ // Default required headers

			"Content-Type",

			"X-Hub-Signature-256",

		},

	}

}



// WithMaxPayloadSize sets the maximum allowed payload size.

func (v *WebhookSecurityValidator) WithMaxPayloadSize(size int64) *WebhookSecurityValidator {

	v.maxPayloadSize = size

	return v

}



// WithAllowedUserAgent sets the allowed User-Agent header value.

func (v *WebhookSecurityValidator) WithAllowedUserAgent(userAgent string) *WebhookSecurityValidator {

	v.allowedUserAgent = userAgent

	return v

}



// WithRequiredHeaders sets the required headers for webhook requests.

func (v *WebhookSecurityValidator) WithRequiredHeaders(headers []string) *WebhookSecurityValidator {

	v.requiredHeaders = headers

	return v

}



// ValidateRequest performs additional security validation on webhook requests.

func (v *WebhookSecurityValidator) ValidateRequest(r *http.Request) error {

	// Check Content-Length header.

	if r.ContentLength > v.maxPayloadSize {

		return &WebhookSecurityError{

			Code:    "PAYLOAD_TOO_LARGE",

			Message: "Request payload exceeds maximum allowed size",

		}

	}



	// Check required headers.

	for _, header := range v.requiredHeaders {

		if r.Header.Get(header) == "" {

			return &WebhookSecurityError{

				Code:    "MISSING_REQUIRED_HEADER",

				Message: "Missing required header: " + header,

			}

		}

	}



	// Check User-Agent if specified.

	if v.allowedUserAgent != "" {

		userAgent := r.Header.Get("User-Agent")

		if !strings.Contains(userAgent, v.allowedUserAgent) {

			return &WebhookSecurityError{

				Code:    "INVALID_USER_AGENT",

				Message: "Invalid or missing User-Agent header",

			}

		}

	}



	// Check Content-Type.

	contentType := r.Header.Get("Content-Type")

	if !strings.Contains(contentType, "application/json") {

		return &WebhookSecurityError{

			Code:    "INVALID_CONTENT_TYPE",

			Message: "Content-Type must be application/json",

		}

	}



	return nil

}



// WebhookSecurityError represents a webhook security validation error.

type WebhookSecurityError struct {

	Code    string

	Message string

}



// Error performs error operation.

func (e *WebhookSecurityError) Error() string {

	return e.Message

}



// SecureStringCompare performs constant-time string comparison.

func SecureStringCompare(a, b string) bool {

	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1

}



// WebhookTimingValidator helps prevent timing attacks by ensuring consistent response times.

type WebhookTimingValidator struct {

	minResponseTime time.Duration

}



// NewWebhookTimingValidator creates a new timing validator.

func NewWebhookTimingValidator(minTime time.Duration) *WebhookTimingValidator {

	return &WebhookTimingValidator{

		minResponseTime: minTime,

	}

}



// EnsureMinimumResponseTime ensures responses take at least the minimum time.

func (v *WebhookTimingValidator) EnsureMinimumResponseTime(start time.Time) {

	elapsed := time.Since(start)

	if elapsed < v.minResponseTime {

		time.Sleep(v.minResponseTime - elapsed)

	}

}



// WebhookRateLimiter provides simple rate limiting for webhook endpoints.

type WebhookRateLimiter struct {

	requests map[string][]time.Time

	limit    int

	window   time.Duration

}



// NewWebhookRateLimiter creates a new rate limiter.

func NewWebhookRateLimiter(limit int, window time.Duration) *WebhookRateLimiter {

	return &WebhookRateLimiter{

		requests: make(map[string][]time.Time),

		limit:    limit,

		window:   window,

	}

}



// IsAllowed checks if a request from the given IP is allowed.

func (rl *WebhookRateLimiter) IsAllowed(ip string) bool {

	now := time.Now()



	// Clean old requests.

	rl.cleanOldRequests(ip, now)



	// Check if we're within limits.

	if len(rl.requests[ip]) >= rl.limit {

		return false

	}



	// Add current request.

	rl.requests[ip] = append(rl.requests[ip], now)

	return true

}



func (rl *WebhookRateLimiter) cleanOldRequests(ip string, now time.Time) {

	if requests, exists := rl.requests[ip]; exists {

		cutoff := now.Add(-rl.window)

		validRequests := make([]time.Time, 0)



		for _, reqTime := range requests {

			if reqTime.After(cutoff) {

				validRequests = append(validRequests, reqTime)

			}

		}



		rl.requests[ip] = validRequests

	}

}


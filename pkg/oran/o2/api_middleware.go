package o2

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"

	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

// loggingMiddleware logs HTTP requests and responses
func (s *O2APIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create request context with correlation ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status and size
		wrappedWriter := &responseWriter{ResponseWriter: w}

		// Log request
		s.logger.Info("HTTP request started",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"),
		)

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Log response
		duration := time.Since(start)
		s.logger.Info("HTTP request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", wrappedWriter.statusCode,
			"response_size", wrappedWriter.size,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// metricsMiddleware records metrics for HTTP requests
func (s *O2APIServer) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture metrics
		wrappedWriter := &responseWriter{ResponseWriter: w}

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Record metrics
		duration := time.Since(start)
		endpoint := s.normalizeEndpoint(r.URL.Path)

		s.metrics.RecordRequest(
			r.Method,
			endpoint,
			wrappedWriter.statusCode,
			duration,
			int64(wrappedWriter.size),
		)

		// Record errors if status code indicates an error
		if wrappedWriter.statusCode >= 400 {
			errorType := "client_error"
			if wrappedWriter.statusCode >= 500 {
				errorType = "server_error"
			}
			s.metrics.RecordError(errorType, endpoint)
		}
	})
}

// recoveryMiddleware recovers from panics and returns 500 status
func (s *O2APIServer) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := r.Context().Value("request_id")
				s.logger.Error("panic recovered",
					"request_id", requestID,
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)

				s.metrics.RecordError("panic", s.normalizeEndpoint(r.URL.Path))

				w.Header().Set("Content-Type", ContentTypeProblemJSON)
				w.WriteHeader(StatusInternalServerError)
				w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500,"detail":"An unexpected error occurred"}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// createRateLimitMiddleware creates a rate limiting middleware
func (s *O2APIServer) createRateLimitMiddleware() http.Handler {
	config := s.config.SecurityConfig.RateLimitConfig

	// Create rate limiter based on configuration
	limiter := rate.NewLimiter(
		rate.Limit(float64(config.RequestsPerMin)/60.0), // requests per second
		config.BurstSize,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client identifier based on key function
		var clientKey string
		switch config.KeyFunc {
		case "ip":
			clientKey = getClientIP(r)
		case "user":
			if userID := r.Header.Get("X-User-ID"); userID != "" {
				clientKey = userID
			} else {
				clientKey = getClientIP(r)
			}
		case "token":
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				clientKey = authHeader
			} else {
				clientKey = getClientIP(r)
			}
		default:
			clientKey = getClientIP(r)
		}

		// Check rate limit
		if !limiter.Allow() {
			s.logger.Warn("rate limit exceeded",
				"client", clientKey,
				"method", r.Method,
				"path", r.URL.Path,
			)

			s.metrics.RecordError("rate_limit", s.normalizeEndpoint(r.URL.Path))

			w.Header().Set("Content-Type", ContentTypeProblemJSON)
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":"Rate limit exceeded"}`))
			return
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture response metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}

// normalizeEndpoint normalizes the endpoint path for metrics
func (s *O2APIServer) normalizeEndpoint(path string) string {
	// Replace path parameters with placeholders for consistent metrics
	// This helps avoid high cardinality issues with metrics

	// Use mux.CurrentRoute to get the route template if available
	// For now, implement basic normalization

	if strings.HasPrefix(path, "/ims/v1/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			endpoint := "/" + parts[1] + "/" + parts[2] + "/" + parts[3]

			// Replace IDs with placeholders
			if len(parts) > 4 {
				endpoint += "/{id}"
				if len(parts) > 5 {
					endpoint += "/" + strings.Join(parts[5:], "/")
				}
			}

			return endpoint
		}
	}

	return path
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in case of multiple
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	return r.RemoteAddr
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation using timestamp and random component
	// In production, consider using UUID or more sophisticated approach
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// requestSizeMiddleware limits the size of incoming requests
func (s *O2APIServer) requestSizeMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxSize {
				s.logger.Warn("request too large",
					"content_length", r.ContentLength,
					"max_size", maxSize,
					"method", r.Method,
					"path", r.URL.Path,
				)

				s.metrics.RecordError("request_too_large", s.normalizeEndpoint(r.URL.Path))

				w.Header().Set("Content-Type", ContentTypeProblemJSON)
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte(`{"type":"about:blank","title":"Request Entity Too Large","status":413,"detail":"Request body too large"}`))
				return
			}

			// Wrap the request body with a limited reader as additional protection
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			next.ServeHTTP(w, r)
		})
	}
}

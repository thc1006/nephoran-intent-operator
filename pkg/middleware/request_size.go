package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// RequestSizeLimiter creates middleware to enforce request body size limits
type RequestSizeLimiter struct {
	maxSize int64
	logger  *slog.Logger
}

// NewRequestSizeLimiter creates a new request size limiter middleware
func NewRequestSizeLimiter(maxSize int64, logger *slog.Logger) *RequestSizeLimiter {
	return &RequestSizeLimiter{
		maxSize: maxSize,
		logger:  logger,
	}
}

// Middleware returns the HTTP middleware function
func (rsl *RequestSizeLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply size limits to requests with bodies (POST, PUT, PATCH)
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			// Wrap the request body with MaxBytesReader
			r.Body = http.MaxBytesReader(w, r.Body, rsl.maxSize)
		}
		
		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

// Handler wraps individual handler functions with request size limits
func (rsl *RequestSizeLimiter) Handler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only apply size limits to requests with bodies (POST, PUT, PATCH)
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			// Wrap the request body with MaxBytesReader
			originalBody := r.Body
			r.Body = http.MaxBytesReader(w, r.Body, rsl.maxSize)
			
			// Log size limit enforcement
			rsl.logger.Debug("Request size limit enforced",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("max_size_bytes", rsl.maxSize),
			)
			
			// Defer cleanup (though MaxBytesReader handles this)
			defer func() {
				if originalBody != nil {
					originalBody.Close()
				}
			}()
		}
		
		// Call the wrapped handler
		handler(w, r)
	}
}

// writePayloadTooLargeResponse writes an HTTP 413 response
func writePayloadTooLargeResponse(w http.ResponseWriter, logger *slog.Logger, maxSize int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	
	response := fmt.Sprintf(`{
		"error": "Request payload too large",
		"message": "Request body exceeds maximum allowed size of %d bytes",
		"status": "error",
		"code": 413
	}`, maxSize)
	
	if _, err := w.Write([]byte(response)); err != nil {
		logger.Error("Failed to write payload too large response", slog.String("error", err.Error()))
	}
	
	logger.Warn("Request rejected due to size limit",
		slog.Int64("max_size_bytes", maxSize),
		slog.Int("status_code", http.StatusRequestEntityTooLarge),
	)
}

// MaxBytesHandler creates a handler that enforces size limits and returns proper 413 responses
func MaxBytesHandler(maxSize int64, logger *slog.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only apply size limits to requests with bodies
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			// Check if content-length header indicates oversized request
			if r.ContentLength > maxSize {
				logger.Warn("Request rejected due to Content-Length header",
					slog.Int64("content_length", r.ContentLength),
					slog.Int64("max_size_bytes", maxSize),
				)
				writePayloadTooLargeResponse(w, logger, maxSize)
				return
			}
			
			// Wrap the request body with MaxBytesReader
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			
			logger.Debug("Request size limit applied",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("max_size_bytes", maxSize),
				slog.Int64("content_length", r.ContentLength),
			)
		}
		
		// Call the handler and catch MaxBytesReader errors
		defer func() {
			if err := recover(); err != nil {
				// Check if this is a MaxBytesReader error or a known panic string
				if httpErr, ok := err.(*http.MaxBytesError); ok {
					logger.Warn("Request body size limit exceeded",
						slog.Int64("limit", httpErr.Limit),
						slog.String("path", r.URL.Path),
					)
					writePayloadTooLargeResponse(w, logger, maxSize)
					return
				}
				
				// Check for string-based panic from http.MaxBytesReader
				// Using multiple detection patterns to be resilient to standard library changes
				if errStr, ok := err.(string); ok {
					// Check for various patterns that indicate body size exceeded
					lowerErr := strings.ToLower(errStr)
					isSizeError := false
					
					// Current known pattern
					if errStr == "http: request body too large" {
						isSizeError = true
					} else if strings.Contains(lowerErr, "request body too large") {
						// More flexible pattern matching
						isSizeError = true
					} else if strings.Contains(lowerErr, "body too large") {
						// Even more flexible
						isSizeError = true
					} else if strings.Contains(lowerErr, "maxbytesreader") && strings.Contains(lowerErr, "limit") {
						// Pattern that might appear in future versions
						isSizeError = true
					}
					
					if isSizeError {
						logger.Warn("Request body size limit exceeded (string panic)",
							slog.Int64("max_size_bytes", maxSize),
							slog.String("path", r.URL.Path),
							slog.String("panic_message", errStr),
						)
						writePayloadTooLargeResponse(w, logger, maxSize)
						return
					}
				}
				
				// Log unexpected panic before re-panicking
				logger.Error("Unexpected panic in request size handler",
					slog.Any("error", err),
					slog.String("path", r.URL.Path),
				)
				
				// Re-panic if it's not a known MaxBytesReader error
				panic(err)
			}
		}()
		
		handler(w, r)
	}
}
package nephiowebui

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var (
	requestCounter = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Name: "nephio_webui_requests_total",

			Help: "Total number of HTTP requests",
		},

		[]string{"method", "path", "status"},
	)

	requestDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Name: "nephio_webui_request_duration_seconds",

			Help: "HTTP request latencies in seconds",

			Buckets: prometheus.DefBuckets,
		},

		[]string{"method", "path"},
	)
)

// AuthMiddleware handles JWT/OIDC token validation.

func AuthMiddleware(logger *zap.Logger, jwtVerifier func(token string) (*jwt.Claims, error)) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {

				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)

				return

			}

			parts := strings.Split(authHeader, " ")

			if len(parts) != 2 || parts[0] != "Bearer" {

				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)

				return

			}

			claims, err := jwtVerifier(parts[1])

			if err != nil {

				logger.Error("Token validation failed", zap.Error(err))

				http.Error(w, "Invalid token", http.StatusUnauthorized)

				return

			}

			// Add user claims to context.

			ctx := context.WithValue(r.Context(), "claims", claims)

			next.ServeHTTP(w, r.WithContext(ctx))

		})

	}

}

// RBACMiddleware handles role-based access control.

func RBACMiddleware(logger *zap.Logger, checkAccess func(claims *jwt.Claims, path, method string) bool) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			claims, ok := r.Context().Value("claims").(*jwt.Claims)

			if !ok {

				http.Error(w, "Missing authentication context", http.StatusInternalServerError)

				return

			}

			if !checkAccess(claims, r.URL.Path, r.Method) {

				http.Error(w, "Insufficient permissions", http.StatusForbidden)

				return

			}

			next.ServeHTTP(w, r)

		})

	}

}

// LoggingMiddleware provides structured logging for HTTP requests.

func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			correlationID := r.Header.Get("X-Correlation-ID")

			if correlationID == "" {

				correlationID = uuid.New().String()

			}

			ctx := context.WithValue(r.Context(), "correlation_id", correlationID)

			w.Header().Set("X-Correlation-ID", correlationID)

			// Use a custom ResponseWriter to capture status code.

			crw := &customResponseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(crw, r.WithContext(ctx))

			duration := time.Since(start)

			logger.Info("HTTP Request",

				zap.String("method", r.Method),

				zap.String("path", r.URL.Path),

				zap.Int("status", crw.status),

				zap.String("correlation_id", correlationID),

				zap.Duration("duration", duration),
			)

			// Prometheus metrics.

			requestCounter.WithLabelValues(r.Method, r.URL.Path, http.StatusText(crw.status)).Inc()

			requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())

		})

	}

}

// CORSMiddleware handles Cross-Origin Resource Sharing.

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			origin := r.Header.Get("Origin")

			allowOrigin := false

			for _, allowed := range allowedOrigins {

				if allowed == "*" || allowed == origin {

					allowOrigin = true

					break

				}

			}

			if allowOrigin {

				w.Header().Set("Access-Control-Allow-Origin", origin)

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Correlation-ID")

				w.Header().Set("Access-Control-Allow-Credentials", "true")

			}

			if r.Method == "OPTIONS" {

				w.WriteHeader(http.StatusOK)

				return

			}

			next.ServeHTTP(w, r)

		})

	}

}

// RateLimitMiddleware provides request rate limiting.

func RateLimitMiddleware(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {

	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if !limiter.Allow() {

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

				return

			}

			next.ServeHTTP(w, r)

		})

	}

}

// SecurityHeadersMiddleware adds security-related HTTP headers.

func SecurityHeadersMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("X-XSS-Protection", "1; mode=block")

		w.Header().Set("X-Content-Type-Options", "nosniff")

		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")

		next.ServeHTTP(w, r)

	})

}

// customResponseWriter wraps http.ResponseWriter to capture status code.

type customResponseWriter struct {
	http.ResponseWriter

	status int
}

// WriteHeader performs writeheader operation.

func (crw *customResponseWriter) WriteHeader(status int) {

	crw.status = status

	crw.ResponseWriter.WriteHeader(status)

}

//go:build go1.24

package generics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Handler represents a generic handler function.
type Handler[TRequest, TResponse any] func(context.Context, TRequest) Result[TResponse, error]

// Middleware represents a generic middleware function.
type Middleware[TRequest, TResponse any] func(Handler[TRequest, TResponse]) Handler[TRequest, TResponse]

// MiddlewareChain manages a chain of middlewares with type safety.
type MiddlewareChain[TRequest, TResponse any] struct {
	middlewares []Middleware[TRequest, TResponse]
}

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain[TRequest, TResponse any]() *MiddlewareChain[TRequest, TResponse] {
	return &MiddlewareChain[TRequest, TResponse]{
		middlewares: make([]Middleware[TRequest, TResponse], 0),
	}
}

// Add adds a middleware to the chain.
func (mc *MiddlewareChain[TRequest, TResponse]) Add(middleware Middleware[TRequest, TResponse]) *MiddlewareChain[TRequest, TResponse] {
	mc.middlewares = append(mc.middlewares, middleware)
	return mc
}

// Build builds the final handler with all middlewares applied.
func (mc *MiddlewareChain[TRequest, TResponse]) Build(finalHandler Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
	// Apply middlewares in reverse order (last middleware wraps first)
	handler := finalHandler
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		handler = mc.middlewares[i](handler)
	}
	return handler
}

// Execute executes the middleware chain with the given request.
func (mc *MiddlewareChain[TRequest, TResponse]) Execute(ctx context.Context, request TRequest, finalHandler Handler[TRequest, TResponse]) Result[TResponse, error] {
	handler := mc.Build(finalHandler)
	return handler(ctx, request)
}

// HTTP Middleware implementations

// HTTPRequest represents a generic HTTP request.
type HTTPRequest[T any] struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    T
}

// HTTPResponse represents a generic HTTP response.
type HTTPResponse[T any] struct {
	StatusCode int
	Headers    map[string]string
	Body       T
}

// CORSMiddleware provides CORS handling for HTTP requests.
func CORSMiddleware[TRequest, TResponse any](config CORSConfig) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Add CORS headers to context for later use
			ctx = context.WithValue(ctx, "cors_headers", config.Headers())
			return next(ctx, request)
		}
	}
}

// CORSConfig configures CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	MaxAge           time.Duration
	AllowCredentials bool
}

// Headers returns CORS headers.
func (c CORSConfig) Headers() map[string]string {
	headers := make(map[string]string)

	if len(c.AllowedOrigins) > 0 {
		headers["Access-Control-Allow-Origin"] = strings.Join(c.AllowedOrigins, ", ")
	}

	if len(c.AllowedMethods) > 0 {
		headers["Access-Control-Allow-Methods"] = strings.Join(c.AllowedMethods, ", ")
	}

	if len(c.AllowedHeaders) > 0 {
		headers["Access-Control-Allow-Headers"] = strings.Join(c.AllowedHeaders, ", ")
	}

	if len(c.ExposedHeaders) > 0 {
		headers["Access-Control-Expose-Headers"] = strings.Join(c.ExposedHeaders, ", ")
	}

	if c.MaxAge > 0 {
		headers["Access-Control-Max-Age"] = fmt.Sprintf("%.0f", c.MaxAge.Seconds())
	}

	if c.AllowCredentials {
		headers["Access-Control-Allow-Credentials"] = "true"
	}

	return headers
}

// AuthenticationMiddleware provides authentication for requests.
func AuthenticationMiddleware[TRequest, TResponse any](authenticator Authenticator[TRequest]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Authenticate the request
			authResult := authenticator.Authenticate(ctx, request)
			if authResult.IsErr() {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("authentication failed: %w", authResult.Error()))
			}

			// Add user info to context
			user := authResult.Value()
			ctx = context.WithValue(ctx, "authenticated_user", user)

			return next(ctx, request)
		}
	}
}

// Authenticator defines an interface for authentication.
type Authenticator[TRequest any] interface {
	Authenticate(ctx context.Context, request TRequest) Result[User, error]
}

// User represents an authenticated user.
type User struct {
	ID          string
	Username    string
	Email       string
	Roles       []string
	Permissions []string
	Metadata    map[string]any
}

// JWTAuthenticator implements JWT-based authentication.
type JWTAuthenticator[TRequest any] struct {
	tokenExtractor func(TRequest) Option[string]
	validator      TokenValidator
}

// TokenValidator validates JWT tokens.
type TokenValidator interface {
	Validate(ctx context.Context, token string) Result[User, error]
}

// NewJWTAuthenticator creates a new JWT authenticator.
func NewJWTAuthenticator[TRequest any](
	extractor func(TRequest) Option[string],
	validator TokenValidator,
) *JWTAuthenticator[TRequest] {
	return &JWTAuthenticator[TRequest]{
		tokenExtractor: extractor,
		validator:      validator,
	}
}

// Authenticate authenticates using JWT token.
func (ja *JWTAuthenticator[TRequest]) Authenticate(ctx context.Context, request TRequest) Result[User, error] {
	tokenOpt := ja.tokenExtractor(request)
	if tokenOpt.IsNone() {
		return Err[User, error](fmt.Errorf("no token found in request"))
	}

	token := tokenOpt.Value()
	return ja.validator.Validate(ctx, token)
}

// AuthorizationMiddleware provides authorization for requests.
func AuthorizationMiddleware[TRequest, TResponse any](authorizer Authorizer[TRequest]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Get user from context
			userVal := ctx.Value("authenticated_user")
			if userVal == nil {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("no authenticated user found"))
			}

			user, ok := userVal.(User)
			if !ok {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("invalid user type in context"))
			}

			// Check authorization
			authResult := authorizer.Authorize(ctx, user, request)
			if authResult.IsErr() {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("authorization failed: %w", authResult.Error()))
			}

			if !authResult.Value() {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("access denied"))
			}

			return next(ctx, request)
		}
	}
}

// Authorizer defines an interface for authorization.
type Authorizer[TRequest any] interface {
	Authorize(ctx context.Context, user User, request TRequest) Result[bool, error]
}

// RoleBasedAuthorizer implements role-based authorization.
type RoleBasedAuthorizer[TRequest any] struct {
	requiredRoles []string
	roleExtractor func(TRequest) []string // Extract required roles from request
}

// NewRoleBasedAuthorizer creates a new role-based authorizer.
func NewRoleBasedAuthorizer[TRequest any](
	requiredRoles []string,
	roleExtractor func(TRequest) []string,
) *RoleBasedAuthorizer[TRequest] {
	return &RoleBasedAuthorizer[TRequest]{
		requiredRoles: requiredRoles,
		roleExtractor: roleExtractor,
	}
}

// Authorize checks if user has required roles.
func (rba *RoleBasedAuthorizer[TRequest]) Authorize(ctx context.Context, user User, request TRequest) Result[bool, error] {
	// Get required roles from request if extractor is provided
	requiredRoles := rba.requiredRoles
	if rba.roleExtractor != nil {
		requiredRoles = rba.roleExtractor(request)
	}

	// Check if user has any of the required roles
	userRoleSet := NewSet(user.Roles...)
	for _, role := range requiredRoles {
		if userRoleSet.Contains(role) {
			return Ok[bool, error](true)
		}
	}

	return Ok[bool, error](false)
}

// RateLimitingMiddleware provides rate limiting for requests.
func RateLimitingMiddleware[TRequest, TResponse any](limiter RateLimiter[TRequest]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Check rate limit
			limitResult := limiter.Allow(ctx, request)
			if limitResult.IsErr() {
				var zero TResponse
				return Err[TResponse, error](limitResult.Error())
			}

			if !limitResult.Value() {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("rate limit exceeded"))
			}

			return next(ctx, request)
		}
	}
}

// RateLimiter defines an interface for rate limiting.
type RateLimiter[TRequest any] interface {
	Allow(ctx context.Context, request TRequest) Result[bool, error]
}

// TokenBucketRateLimiter implements token bucket rate limiting.
type TokenBucketRateLimiter[TRequest any] struct {
	buckets      *SafeMap[string, *TokenBucket]
	keyExtractor func(TRequest) string
	capacity     int
	refillRate   time.Duration
}

// TokenBucket represents a token bucket for rate limiting.
type TokenBucket struct {
	capacity   int
	tokens     int
	lastRefill time.Time
	refillRate time.Duration
	mu         sync.Mutex
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter.
func NewTokenBucketRateLimiter[TRequest any](
	keyExtractor func(TRequest) string,
	capacity int,
	refillRate time.Duration,
) *TokenBucketRateLimiter[TRequest] {
	return &TokenBucketRateLimiter[TRequest]{
		buckets:      NewSafeMap[string, *TokenBucket](),
		keyExtractor: keyExtractor,
		capacity:     capacity,
		refillRate:   refillRate,
	}
}

// Allow checks if the request is within rate limits.
func (tbrl *TokenBucketRateLimiter[TRequest]) Allow(ctx context.Context, request TRequest) Result[bool, error] {
	key := tbrl.keyExtractor(request)

	// Get or create token bucket
	bucketOpt := tbrl.buckets.Get(key)
	var bucket *TokenBucket

	if bucketOpt.IsNone() {
		bucket = &TokenBucket{
			capacity:   tbrl.capacity,
			tokens:     tbrl.capacity,
			lastRefill: time.Now(),
			refillRate: tbrl.refillRate,
		}
		tbrl.buckets.Set(key, bucket)
	} else {
		bucket = bucketOpt.Value()
	}

	return Ok[bool, error](bucket.takeToken())
}

// takeToken attempts to take a token from the bucket.
func (tb *TokenBucket) takeToken() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed / tb.refillRate)
	tb.tokens += tokensToAdd
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	// Check if token is available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// SafeMap provides a thread-safe generic map wrapper.
type SafeMap[K Comparable, V any] struct {
	data map[K]V
	mu   sync.RWMutex
}

// NewSafeMap creates a new thread-safe map.
func NewSafeMap[K Comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data: make(map[K]V),
	}
}

// Set sets a key-value pair thread-safely.
func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
}

// Get gets a value by key thread-safely.
func (sm *SafeMap[K, V]) Get(key K) Option[V] {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if value, exists := sm.data[key]; exists {
		return Some(value)
	}
	return None[V]()
}

// LoggingMiddleware provides logging for requests and responses.
func LoggingMiddleware[TRequest, TResponse any](logger Logger[TRequest, TResponse]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			start := time.Now()

			// Log request
			logger.LogRequest(ctx, request)

			// Process request
			result := next(ctx, request)

			// Log response
			duration := time.Since(start)
			if result.IsOk() {
				logger.LogResponse(ctx, request, result.Value(), duration, nil)
			} else {
				logger.LogResponse(ctx, request, *new(TResponse), duration, result.Error())
			}

			return result
		}
	}
}

// Logger defines an interface for logging middleware.
type Logger[TRequest, TResponse any] interface {
	LogRequest(ctx context.Context, request TRequest)
	LogResponse(ctx context.Context, request TRequest, response TResponse, duration time.Duration, err error)
}

// JSONLogger implements logging in JSON format.
type JSONLogger[TRequest, TResponse any] struct {
	writer func(string) // Where to write logs
}

// NewJSONLogger creates a new JSON logger.
func NewJSONLogger[TRequest, TResponse any](writer func(string)) *JSONLogger[TRequest, TResponse] {
	return &JSONLogger[TRequest, TResponse]{
		writer: writer,
	}
}

// LogRequest logs the request.
func (jl *JSONLogger[TRequest, TResponse]) LogRequest(ctx context.Context, request TRequest) {
	logEntry := map[string]any{
		"timestamp": time.Now(),
		"type":      "request",
		"request":   request,
	}

	if data, err := json.Marshal(logEntry); err == nil {
		jl.writer(string(data))
	}
}

// LogResponse logs the response.
func (jl *JSONLogger[TRequest, TResponse]) LogResponse(ctx context.Context, request TRequest, response TResponse, duration time.Duration, err error) {
	logEntry := map[string]any{
		"timestamp": time.Now(),
		"type":      "response",
		"request":   request,
		"response":  response,
		"duration":  duration.String(),
	}

	if err != nil {
		logEntry["error"] = err.Error()
	}

	if data, err := json.Marshal(logEntry); err == nil {
		jl.writer(string(data))
	}
}

// MetricsMiddleware provides metrics collection for requests.
func MetricsMiddleware[TRequest, TResponse any](collector MetricsCollector[TRequest, TResponse]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			start := time.Now()

			// Increment request counter
			collector.IncRequestCounter(ctx, request)

			// Process request
			result := next(ctx, request)

			// Record metrics
			duration := time.Since(start)
			collector.RecordLatency(ctx, request, duration)

			if result.IsErr() {
				collector.IncErrorCounter(ctx, request, result.Error())
			}

			return result
		}
	}
}

// MetricsCollector defines an interface for metrics collection.
type MetricsCollector[TRequest, TResponse any] interface {
	IncRequestCounter(ctx context.Context, request TRequest)
	IncErrorCounter(ctx context.Context, request TRequest, err error)
	RecordLatency(ctx context.Context, request TRequest, duration time.Duration)
}

// CircuitBreakerMiddleware provides circuit breaker functionality.
func CircuitBreakerMiddleware[TRequest, TResponse any](breaker CircuitBreaker[TRequest]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Check if circuit breaker allows the request
			if !breaker.Allow(ctx, request) {
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("circuit breaker is open"))
			}

			// Process request
			result := next(ctx, request)

			// Record success or failure
			if result.IsOk() {
				breaker.RecordSuccess(ctx, request)
			} else {
				breaker.RecordFailure(ctx, request, result.Error())
			}

			return result
		}
	}
}

// CircuitBreaker defines an interface for circuit breaker functionality.
type CircuitBreaker[TRequest any] interface {
	Allow(ctx context.Context, request TRequest) bool
	RecordSuccess(ctx context.Context, request TRequest)
	RecordFailure(ctx context.Context, request TRequest, err error)
}

// TimeoutMiddleware provides timeout functionality for requests.
func TimeoutMiddleware[TRequest, TResponse any](timeout time.Duration) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Channel to receive result
			resultChan := make(chan Result[TResponse, error], 1)

			// Process request in goroutine
			go func() {
				result := next(timeoutCtx, request)
				resultChan <- result
			}()

			// Wait for result or timeout
			select {
			case result := <-resultChan:
				return result
			case <-timeoutCtx.Done():
				var zero TResponse
				return Err[TResponse, error](fmt.Errorf("request timeout after %v", timeout))
			}
		}
	}
}

// CompressionMiddleware provides compression for responses.
func CompressionMiddleware[TRequest, TResponse any](compressor Compressor[TResponse]) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Process request
			result := next(ctx, request)

			if result.IsOk() {
				// Compress response
				compressedResult := compressor.Compress(ctx, result.Value())
				if compressedResult.IsOk() {
					return compressedResult
				}
				// Fall back to original response if compression fails
			}

			return result
		}
	}
}

// Compressor defines an interface for response compression.
type Compressor[TResponse any] interface {
	Compress(ctx context.Context, response TResponse) Result[TResponse, error]
}

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS.
func HTTPSRedirectMiddleware[TRequest, TResponse any](port int) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// This would typically check if the request is HTTP and redirect
			// For now, just proceed with the request
			return next(ctx, request)
		}
	}
}

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware[TRequest, TResponse any](headers map[string]string) Middleware[TRequest, TResponse] {
	defaultHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	// Merge with provided headers
	for k, v := range headers {
		defaultHeaders[k] = v
	}

	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			// Add security headers to context for later use
			ctx = context.WithValue(ctx, "security_headers", defaultHeaders)
			return next(ctx, request)
		}
	}
}

// ConditionalMiddleware applies middleware based on a condition.
func ConditionalMiddleware[TRequest, TResponse any](
	condition func(context.Context, TRequest) bool,
	middleware Middleware[TRequest, TResponse],
) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {
		middlewareHandler := middleware(next)

		return func(ctx context.Context, request TRequest) Result[TResponse, error] {
			if condition(ctx, request) {
				return middlewareHandler(ctx, request)
			}
			return next(ctx, request)
		}
	}
}

// MiddlewareGroup provides predefined middleware combinations.
type MiddlewareGroup[TRequest, TResponse any] struct {
	name        string
	middlewares []Middleware[TRequest, TResponse]
}

// NewMiddlewareGroup creates a new middleware group.
func NewMiddlewareGroup[TRequest, TResponse any](name string) *MiddlewareGroup[TRequest, TResponse] {
	return &MiddlewareGroup[TRequest, TResponse]{
		name:        name,
		middlewares: make([]Middleware[TRequest, TResponse], 0),
	}
}

// Add adds a middleware to the group.
func (mg *MiddlewareGroup[TRequest, TResponse]) Add(middleware Middleware[TRequest, TResponse]) *MiddlewareGroup[TRequest, TResponse] {
	mg.middlewares = append(mg.middlewares, middleware)
	return mg
}

// Build builds a middleware chain from the group.
func (mg *MiddlewareGroup[TRequest, TResponse]) Build() *MiddlewareChain[TRequest, TResponse] {
	chain := NewMiddlewareChain[TRequest, TResponse]()
	for _, middleware := range mg.middlewares {
		chain.Add(middleware)
	}
	return chain
}

// CommonMiddlewareGroups provides common middleware combinations

// WebAPIGroup provides common middleware for web APIs.
func WebAPIGroup[TRequest, TResponse any]() *MiddlewareGroup[TRequest, TResponse] {
	return NewMiddlewareGroup[TRequest, TResponse]("web-api").
		Add(LoggingMiddleware[TRequest, TResponse](NewJSONLogger[TRequest, TResponse](func(s string) {
			// Default logger implementation
		}))).
		Add(CORSMiddleware[TRequest, TResponse](CORSConfig{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"*"},
		})).
		Add(SecurityHeadersMiddleware[TRequest, TResponse](nil)).
		Add(TimeoutMiddleware[TRequest, TResponse](30 * time.Second))
}

// SecureAPIGroup provides middleware for secure APIs.
func SecureAPIGroup[TRequest, TResponse any](
	authenticator Authenticator[TRequest],
	authorizer Authorizer[TRequest],
) *MiddlewareGroup[TRequest, TResponse] {
	return NewMiddlewareGroup[TRequest, TResponse]("secure-api").
		Add(LoggingMiddleware[TRequest, TResponse](NewJSONLogger[TRequest, TResponse](func(s string) {
			// Default logger implementation
		}))).
		Add(SecurityHeadersMiddleware[TRequest, TResponse](nil)).
		Add(AuthenticationMiddleware[TRequest, TResponse](authenticator)).
		Add(AuthorizationMiddleware[TRequest, TResponse](authorizer)).
		Add(TimeoutMiddleware[TRequest, TResponse](30 * time.Second))
}

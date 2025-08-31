package shared

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Go 1.24 Interface Patterns and Error Handling

// Standard errors for interface implementations
var (
	ErrClientClosed       = errors.New("client is closed")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrRateLimited        = errors.New("rate limited")
)

// RequestOptions provides optional configuration for requests
type RequestOptions struct {
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	RateLimiter RateLimiterConfig      `json:"rate_limiter,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RateLimiterConfig configures rate limiting
type RateLimiterConfig struct {
	RequestsPerSecond float64       `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	Timeout           time.Duration `json:"timeout"`
}

// RequestOption represents functional options for requests (Go 1.24 pattern)
type RequestOption func(*RequestOptions)

// Functional option constructors
func WithTimeout(timeout time.Duration) RequestOption {
	return func(opts *RequestOptions) {
		opts.Timeout = timeout
	}
}

func WithRetryPolicy(policy *RetryPolicy) RequestOption {
	return func(opts *RequestOptions) {
		opts.RetryPolicy = policy
	}
}

func WithHeaders(headers map[string]string) RequestOption {
	return func(opts *RequestOptions) {
		opts.Headers = headers
	}
}

func WithRateLimiter(config RateLimiterConfig) RequestOption {
	return func(opts *RequestOptions) {
		opts.RateLimiter = config
	}
}

// HealthStatus provides detailed health information
type HealthStatus struct {
	Status      ClientStatus           `json:"status"`
	Latency     time.Duration          `json:"latency"`
	LastChecked time.Time              `json:"last_checked"`
	Errors      []string               `json:"errors,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ClientMetrics provides client performance metrics
type ClientMetrics struct {
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	TotalTokensUsed int64         `json:"total_tokens_used"`
	LastReset       time.Time     `json:"last_reset"`
}

// ClientError wraps client-specific errors with context (Go 1.24 error patterns)
type ClientError struct {
	Op     string // operation that failed
	Client string // client identifier
	Err    error  // underlying error
}

func (e *ClientError) Error() string {
	if e.Client != "" {
		return fmt.Sprintf("%s: client %s: %v", e.Op, e.Client, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ClientError) Unwrap() error { return e.Err }

// NewClientError creates a new client error with structured context
func NewClientError(op, client string, err error) *ClientError {
	return &ClientError{
		Op:     op,
		Client: client,
		Err:    err,
	}
}

// Result represents a generic result with error handling (Go 1.24 pattern)
type Result[T any] struct {
	Value T
	Err   error
}

// IsOK returns true if the result contains no error
func (r Result[T]) IsOK() bool {
	return r.Err == nil
}

// Unwrap returns the value and error
func (r Result[T]) Unwrap() (T, error) {
	return r.Value, r.Err
}

// Interface compliance validation helpers

// ValidateClientInterface checks if a client properly implements the interface
func ValidateClientInterface(client ClientInterface) error {
	if client == nil {
		return errors.New("client is nil")
	}

	// Check required methods exist by attempting to call them with safe parameters
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = client.HealthCheck(ctx)       // Allow this to fail, we're just checking method exists
	_ = client.GetStatus()            // Should not fail
	_ = client.GetModelCapabilities() // Should not fail
	endpoint := client.GetEndpoint()  // Should not fail

	if endpoint == "" {
		return errors.New("client GetEndpoint() returned empty string")
	}

	return nil
}

// EnhancedClientInterface extends ClientInterface with Go 1.24 patterns
type EnhancedClientInterface interface {
	ClientInterface

	// Process request with options (Go 1.24 functional options pattern)
	ProcessRequestWithOptions(ctx context.Context, request *LLMRequest, opts ...RequestOption) (*LLMResponse, error)

	// Batch processing with structured concurrency
	ProcessBatch(ctx context.Context, requests []*LLMRequest, opts ...RequestOption) ([]*LLMResponse, error)

	// Health check with detailed status
	HealthCheckDetailed(ctx context.Context) (*HealthStatus, error)

	// Metrics and observability
	GetMetrics(ctx context.Context) (*ClientMetrics, error)
}

// ClientFactory creates clients with compile-time type safety
type ClientFactory[T ClientInterface] interface {
	Create(config interface{}) (T, error)
	Validate(client T) error
}

// Interface compliance checks using Go 1.24 patterns
// These compile-time checks ensure proper interface implementation

// Compile-time checks for enhanced interface compliance
type compileTimeChecks struct{}

// This will cause a compilation error if ClientInterface is not properly implemented
func (*compileTimeChecks) validateInterfaces() {
	// var _ ClientInterface = (*YourClientType)(nil) // Example: Will fail if interface is not properly implemented
}

// Helper function to apply request options
func ApplyRequestOptions(base *RequestOptions, opts ...RequestOption) *RequestOptions {
	if base == nil {
		base = &RequestOptions{}
	}

	result := *base // copy
	for _, opt := range opts {
		opt(&result)
	}

	return &result
}

// Default configurations following Go 1.24 best practices
var (
	DefaultRequestOptions = RequestOptions{
		Timeout: 30 * time.Second,
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			BackoffBase:     time.Second,
			BackoffMax:      10 * time.Second,
			BackoffJitter:   true,
			RetryableErrors: nil,
		},
	}

	DefaultRateLimiter = RateLimiterConfig{
		RequestsPerSecond: 10.0,
		BurstSize:         5,
		Timeout:           5 * time.Second,
	}
)

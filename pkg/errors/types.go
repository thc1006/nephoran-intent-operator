package errors

import (
	"fmt"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// String returns the string representation of ErrorSeverity
func (es ErrorSeverity) String() string {
	return string(es)
}

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryBusiness   ErrorCategory = "business"
	CategorySystem     ErrorCategory = "system"
	CategorySecurity   ErrorCategory = "security"
	CategoryNetwork    ErrorCategory = "network"
	CategoryData       ErrorCategory = "data"
	CategoryValidation ErrorCategory = "validation"
	CategoryPermission ErrorCategory = "permission"
	CategoryResource   ErrorCategory = "resource"
	CategoryConfig     ErrorCategory = "config"
	CategoryInternal   ErrorCategory = "internal"
)

// RecoveryStrategy represents different error recovery strategies
type RecoveryStrategy string

const (
	StrategyRetry          RecoveryStrategy = "retry"
	StrategyBackoff        RecoveryStrategy = "backoff"
	StrategyExponential    RecoveryStrategy = "exponential"
	StrategyJittered       RecoveryStrategy = "jittered"
	StrategyCircuitBreaker RecoveryStrategy = "circuit_breaker"
	StrategyFallback       RecoveryStrategy = "fallback"
	StrategyBulkhead       RecoveryStrategy = "bulkhead"
	StrategyTimeout        RecoveryStrategy = "timeout"
	StrategyRateLimit      RecoveryStrategy = "rate_limit"
	StrategyDegradation    RecoveryStrategy = "degradation"
	StrategyComposite      RecoveryStrategy = "composite"
)

// ProcessingError represents an error that occurred during processing
type ProcessingError struct {
	// Basic error information
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`

	// Classification
	Type      ErrorType     `json:"type"`
	Category  ErrorCategory `json:"category"`
	Severity  ErrorSeverity `json:"severity"`
	Component string        `json:"component,omitempty"`
	Operation string        `json:"operation,omitempty"`
	Phase     string        `json:"phase,omitempty"`

	// Context and tracing
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Error chain
	Cause      error              `json:"-"`
	CauseChain []*ProcessingError `json:"cause_chain,omitempty"`
	StackTrace []StackFrame       `json:"stack_trace,omitempty"`

	// Recovery information
	Recoverable      bool             `json:"recoverable"`
	RetryCount       int              `json:"retry_count"`
	MaxRetries       int              `json:"max_retries"`
	BackoffStrategy  string           `json:"backoff_strategy,omitempty"`
	NextRetryTime    *time.Time       `json:"next_retry_time,omitempty"`
	RecoveryStrategy RecoveryStrategy `json:"recovery_strategy,omitempty"`
}

// Error implements the error interface
func (pe *ProcessingError) Error() string {
	if pe.Details != "" {
		return fmt.Sprintf("[%s:%s:%s] %s: %s", pe.Component, pe.Operation, pe.Phase, pe.Message, pe.Details)
	}
	return fmt.Sprintf("[%s:%s:%s] %s", pe.Component, pe.Operation, pe.Phase, pe.Message)
}

// Unwrap implements the error unwrapping interface
func (pe *ProcessingError) Unwrap() error {
	return pe.Cause
}

// This file contains type aliases and imports for backward compatibility

// ErrorType represents different types of errors for classification - using types from errors.go
// Additional constants for extended error types not in errors.go

// Note: Basic ErrorType constants are defined in errors.go to avoid conflicts

// ErrorImpact represents the impact level of an error
type ErrorImpact string

const (
	ImpactNone     ErrorImpact = "none"
	ImpactMinimal  ErrorImpact = "minimal"
	ImpactModerate ErrorImpact = "moderate"
	ImpactSevere   ErrorImpact = "severe"
	ImpactCritical ErrorImpact = "critical"
)

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
	Source   string `json:"source,omitempty"`
	Package  string `json:"package"`
}

// ErrorContextFunc is a function that can add context to an error
type ErrorContextFunc func(*ServiceError)

// ErrorPredicate is a function that tests an error condition
type ErrorPredicate func(*ServiceError) bool

// ErrorTransformer is a function that transforms one error into another
type ErrorTransformer func(*ServiceError) *ServiceError

// ErrorHandler is a function that handles an error
type ErrorHandler func(*ServiceError) error

// ErrorMetrics holds metrics about errors
type ErrorMetrics struct {
	TotalCount     int64            `json:"total_count"`
	CountByType    map[string]int64 `json:"count_by_type"`
	CountByCode    map[string]int64 `json:"count_by_code"`
	MeanLatency    float64          `json:"mean_latency"`
	P95Latency     float64          `json:"p95_latency"`
	P99Latency     float64          `json:"p99_latency"`
	LastOccurrence time.Time        `json:"last_occurrence"`
	RatePerSecond  float64          `json:"rate_per_second"`
}

// ErrorConfiguration holds configuration for error handling
type ErrorConfiguration struct {
	StackTraceEnabled     bool          `json:"stack_trace_enabled"`
	StackTraceDepth       int           `json:"stack_trace_depth"`
	SourceCodeEnabled     bool          `json:"source_code_enabled"`
	SourceCodeLines       int           `json:"source_code_lines"`
	RetryableTypes        []ErrorType   `json:"retryable_types"`
	TemporaryTypes        []ErrorType   `json:"temporary_types"`
	MaxCauseChainDepth    int           `json:"max_cause_chain_depth"`
	DefaultRetryAfter     time.Duration `json:"default_retry_after"`
	CircuitBreakerEnabled bool          `json:"circuit_breaker_enabled"`
	MetricsEnabled        bool          `json:"metrics_enabled"`
}

// DefaultErrorConfiguration returns sensible defaults
func DefaultErrorConfiguration() *ErrorConfiguration {
	return &ErrorConfiguration{
		StackTraceEnabled:     true,
		StackTraceDepth:       10,
		SourceCodeEnabled:     true,
		SourceCodeLines:       3,
		MaxCauseChainDepth:    5,
		DefaultRetryAfter:     time.Second * 30,
		CircuitBreakerEnabled: true,
		MetricsEnabled:        true,
		RetryableTypes: []ErrorType{
			ErrorTypeNetwork,
			ErrorTypeTimeout,
			ErrorTypeExternal,
			ErrorTypeRateLimit,
			ErrorTypeResource,
		},
		TemporaryTypes: []ErrorType{
			ErrorTypeTimeout,
			ErrorTypeRateLimit,
			ErrorTypeQuota,
			ErrorTypeResource,
		},
	}
}

package errors

import (
	"fmt"
	"time"
)

// ErrorSeverity represents the severity level of an error.

type ErrorSeverity string

const (

	// SeverityLow holds severitylow value.

	SeverityLow ErrorSeverity = "low"

	// SeverityMedium holds severitymedium value.

	SeverityMedium ErrorSeverity = "medium"

	// SeverityHigh holds severityhigh value.

	SeverityHigh ErrorSeverity = "high"

	// SeverityCritical holds severitycritical value.

	SeverityCritical ErrorSeverity = "critical"
)

// String returns the string representation of ErrorSeverity.

func (es ErrorSeverity) String() string {
	return string(es)
}

// ErrorCategory represents the category of an error.

type ErrorCategory string

const (

	// CategoryBusiness holds categorybusiness value.

	CategoryBusiness ErrorCategory = "business"

	// CategorySystem holds categorysystem value.

	CategorySystem ErrorCategory = "system"

	// CategorySecurity holds categorysecurity value.

	CategorySecurity ErrorCategory = "security"

	// CategoryNetwork holds categorynetwork value.

	CategoryNetwork ErrorCategory = "network"

	// CategoryData holds categorydata value.

	CategoryData ErrorCategory = "data"

	// CategoryValidation holds categoryvalidation value.

	CategoryValidation ErrorCategory = "validation"

	// CategoryPermission holds categorypermission value.

	CategoryPermission ErrorCategory = "permission"

	// CategoryResource holds categoryresource value.

	CategoryResource ErrorCategory = "resource"

	// CategoryConfig holds categoryconfig value.

	CategoryConfig ErrorCategory = "config"

	// CategoryInternal holds categoryinternal value.

	CategoryInternal ErrorCategory = "internal"

	// CategoryCapacity holds categorycapacity value.

	CategoryCapacity ErrorCategory = "capacity"

	// CategoryRateLimit holds categoryratelimit value.

	CategoryRateLimit ErrorCategory = "rate_limit"

	// CategoryCircuitBreaker holds categorycircuitbreaker value.

	CategoryCircuitBreaker ErrorCategory = "circuit_breaker"
)

// RecoveryStrategy represents different error recovery strategies.

type RecoveryStrategy string

const (

	// StrategyRetry holds strategyretry value.

	StrategyRetry RecoveryStrategy = "retry"

	// StrategyBackoff holds strategybackoff value.

	StrategyBackoff RecoveryStrategy = "backoff"

	// StrategyExponential holds strategyexponential value.

	StrategyExponential RecoveryStrategy = "exponential"

	// StrategyJittered holds strategyjittered value.

	StrategyJittered RecoveryStrategy = "jittered"

	// StrategyCircuitBreaker holds strategycircuitbreaker value.

	StrategyCircuitBreaker RecoveryStrategy = "circuit_breaker"

	// StrategyFallback holds strategyfallback value.

	StrategyFallback RecoveryStrategy = "fallback"

	// StrategyBulkhead holds strategybulkhead value.

	StrategyBulkhead RecoveryStrategy = "bulkhead"

	// StrategyTimeout holds strategytimeout value.

	StrategyTimeout RecoveryStrategy = "timeout"

	// StrategyRateLimit holds strategyratelimit value.

	StrategyRateLimit RecoveryStrategy = "rate_limit"

	// StrategyDegradation holds strategydegradation value.

	StrategyDegradation RecoveryStrategy = "degradation"

	// StrategyComposite holds strategycomposite value.

	StrategyComposite RecoveryStrategy = "composite"
)

// ProcessingError represents an error that occurred during processing.

type ProcessingError struct {
	// Basic error information.

	ID string `json:"id"`

	Code string `json:"code"`

	Message string `json:"message"`

	Details string `json:"details,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	// Classification.

	Type ErrorType `json:"type"`

	Category ErrorCategory `json:"category"`

	Severity ErrorSeverity `json:"severity"`

	Component string `json:"component,omitempty"`

	Operation string `json:"operation,omitempty"`

	Phase string `json:"phase,omitempty"`

	// Context and tracing.

	CorrelationID string `json:"correlation_id,omitempty"`

	TraceID string `json:"trace_id,omitempty"`

	SpanID string `json:"span_id,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Error chain.

	Cause error `json:"-"`

	CauseChain []*ProcessingError `json:"cause_chain,omitempty"`

	StackTrace []StackFrame `json:"stack_trace,omitempty"`

	// Recovery information.

	Recoverable bool `json:"recoverable"`

	RetryCount int `json:"retry_count"`

	MaxRetries int `json:"max_retries"`

	BackoffStrategy string `json:"backoff_strategy,omitempty"`

	NextRetryTime *time.Time `json:"next_retry_time,omitempty"`

	RecoveryStrategy RecoveryStrategy `json:"recovery_strategy,omitempty"`
}

// Error implements the error interface.

func (pe *ProcessingError) Error() string {
	if pe.Details != "" {
		return fmt.Sprintf("[%s:%s:%s] %s: %s", pe.Component, pe.Operation, pe.Phase, pe.Message, pe.Details)
	}

	return fmt.Sprintf("[%s:%s:%s] %s", pe.Component, pe.Operation, pe.Phase, pe.Message)
}

// Unwrap implements the error unwrapping interface.

func (pe *ProcessingError) Unwrap() error {
	return pe.Cause
}

// This file contains type aliases and imports for backward compatibility.

// ErrorType represents different types of errors for classification - using types from errors.go.

// Additional constants for extended error types not in errors.go.

// Note: Basic ErrorType constants are defined in errors.go to avoid conflicts.

// ErrorImpact represents the impact level of an error.

type ErrorImpact string

const (

	// ImpactNone holds impactnone value.

	ImpactNone ErrorImpact = "none"

	// ImpactMinimal holds impactminimal value.

	ImpactMinimal ErrorImpact = "minimal"

	// ImpactModerate holds impactmoderate value.

	ImpactModerate ErrorImpact = "moderate"

	// ImpactSevere holds impactsevere value.

	ImpactSevere ErrorImpact = "severe"

	// ImpactCritical holds impactcritical value.

	ImpactCritical ErrorImpact = "critical"
)

// StackFrame represents a single frame in the stack trace.

type StackFrame struct {
	File string `json:"file"`

	Line int `json:"line"`

	Function string `json:"function"`

	Source string `json:"source,omitempty"`

	Package string `json:"package"`
}

// String returns a formatted string representation of the stack frame.

func (sf StackFrame) String() string {
	return fmt.Sprintf("%s:%d %s", sf.File, sf.Line, sf.Function)
}

// ErrorContextFunc is a function that can add context to an error.

type ErrorContextFunc func(*ServiceError)

// ErrorPredicate is a function that tests an error condition.

type ErrorPredicate func(*ServiceError) bool

// ErrorTransformer is a function that transforms one error into another.

type ErrorTransformer func(*ServiceError) *ServiceError

// ErrorHandler is a function that handles an error.

type ErrorHandler func(*ServiceError) error

// ErrorMetrics holds metrics about errors.

type ErrorMetrics struct {
	TotalCount int64 `json:"total_count"`

	CountByType map[string]int64 `json:"count_by_type"`

	CountByCode map[string]int64 `json:"count_by_code"`

	MeanLatency float64 `json:"mean_latency"`

	P95Latency float64 `json:"p95_latency"`

	P99Latency float64 `json:"p99_latency"`

	LastOccurrence time.Time `json:"last_occurrence"`

	RatePerSecond float64 `json:"rate_per_second"`
}

// ErrorConfiguration holds configuration for error handling.

type ErrorConfiguration struct {
	StackTraceEnabled bool `json:"stack_trace_enabled"`

	StackTraceDepth int `json:"stack_trace_depth"`

	SourceCodeEnabled bool `json:"source_code_enabled"`

	SourceCodeLines int `json:"source_code_lines"`

	RetryableTypes []ErrorType `json:"retryable_types"`

	TemporaryTypes []ErrorType `json:"temporary_types"`

	MaxCauseChainDepth int `json:"max_cause_chain_depth"`

	DefaultRetryAfter time.Duration `json:"default_retry_after"`

	CircuitBreakerEnabled bool `json:"circuit_breaker_enabled"`

	MetricsEnabled bool `json:"metrics_enabled"`
}

// DefaultErrorConfiguration returns sensible defaults.

func DefaultErrorConfiguration() *ErrorConfiguration {
	return &ErrorConfiguration{
		StackTraceEnabled: true,

		StackTraceDepth: 10,

		SourceCodeEnabled: true,

		SourceCodeLines: 3,

		MaxCauseChainDepth: 5,

		DefaultRetryAfter: time.Second * 30,

		CircuitBreakerEnabled: true,

		MetricsEnabled: true,

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

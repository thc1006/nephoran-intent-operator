package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryUser        ErrorCategory = "user"
	CategorySystem      ErrorCategory = "system"
	CategoryInternal    ErrorCategory = "internal"
	CategoryExternal    ErrorCategory = "external"
	CategoryValidation  ErrorCategory = "validation"
	CategoryPermission  ErrorCategory = "permission"
	CategoryResource    ErrorCategory = "resource"
	CategoryNetwork     ErrorCategory = "network"
	CategoryTimeout     ErrorCategory = "timeout"
	CategoryConfig      ErrorCategory = "config"
)

// ErrorType represents different types of errors for classification
type ErrorType string

const (
	// Validation errors
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeRequired   ErrorType = "required"
	ErrorTypeInvalid    ErrorType = "invalid"
	ErrorTypeFormat     ErrorType = "format"
	ErrorTypeRange      ErrorType = "range"

	// Infrastructure errors  
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeExternal   ErrorType = "external"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeQuota      ErrorType = "quota"

	// Authentication/Authorization errors
	ErrorTypeAuth         ErrorType = "authentication"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"
	ErrorTypeExpired      ErrorType = "expired"

	// Business logic errors
	ErrorTypeBusiness    ErrorType = "business"
	ErrorTypeConflict    ErrorType = "conflict"
	ErrorTypeNotFound    ErrorType = "not_found"
	ErrorTypeDuplicate   ErrorType = "duplicate"
	ErrorTypePrecondition ErrorType = "precondition"

	// System errors
	ErrorTypeInternal ErrorType = "internal"
	ErrorTypeConfig   ErrorType = "configuration"
	ErrorTypeResource ErrorType = "resource"
	ErrorTypeDisk     ErrorType = "disk"
	ErrorTypeMemory   ErrorType = "memory"
	ErrorTypeCPU      ErrorType = "cpu"

	// O-RAN specific errors
	ErrorTypeORANA1     ErrorType = "oran_a1"
	ErrorTypeORANO1     ErrorType = "oran_o1"
	ErrorTypeORANO2     ErrorType = "oran_o2"
	ErrorTypeORANE2     ErrorType = "oran_e2"
	ErrorTypeRIC        ErrorType = "ric"
	ErrorTypeNetworkSlice ErrorType = "network_slice"

	// Kubernetes specific errors
	ErrorTypeK8sResource   ErrorType = "k8s_resource"
	ErrorTypeK8sAPI        ErrorType = "k8s_api"
	ErrorTypeK8sOperator   ErrorType = "k8s_operator"
	ErrorTypeK8sController ErrorType = "k8s_controller"

	// LLM/AI specific errors
	ErrorTypeLLM       ErrorType = "llm"
	ErrorTypeRAG       ErrorType = "rag"
	ErrorTypeEmbedding ErrorType = "embedding"
	ErrorTypeVector    ErrorType = "vector"
)

// ErrorImpact represents the impact level of an error
type ErrorImpact string

const (
	ImpactNone      ErrorImpact = "none"
	ImpactMinimal   ErrorImpact = "minimal"
	ImpactModerate  ErrorImpact = "moderate"
	ImpactSevere    ErrorImpact = "severe"
	ImpactCritical  ErrorImpact = "critical"
)

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
	Source   string `json:"source,omitempty"`
	Package  string `json:"package"`
}

// ServiceError represents a comprehensive error with rich context
type ServiceError struct {
	mu sync.RWMutex

	// Basic error information
	Type         ErrorType              `json:"type"`
	Code         string                 `json:"code"`
	Message      string                 `json:"message"`
	Details      string                 `json:"details,omitempty"`
	Service      string                 `json:"service"`
	Operation    string                 `json:"operation"`
	Component    string                 `json:"component,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	CorrelationID string                `json:"correlation_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`

	// Error classification
	Category  ErrorCategory `json:"category"`
	Severity  ErrorSeverity `json:"severity"`
	Impact    ErrorImpact   `json:"impact"`
	Retryable bool          `json:"retryable"`
	Temporary bool          `json:"temporary"`

	// HTTP information
	HTTPStatus   int    `json:"http_status,omitempty"`
	HTTPMethod   string `json:"http_method,omitempty"`
	HTTPPath     string `json:"http_path,omitempty"`
	RemoteAddr   string `json:"remote_addr,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`

	// Error chain and context
	Cause       error                  `json:"-"`
	CauseChain  []*ServiceError        `json:"cause_chain,omitempty"`
	StackTrace  []StackFrame           `json:"stack_trace,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`

	// Performance information
	Latency   time.Duration     `json:"latency,omitempty"`
	Resources map[string]string `json:"resources,omitempty"`

	// Recovery information
	RecoveryHints  []string          `json:"recovery_hints,omitempty"`
	RetryCount     int               `json:"retry_count,omitempty"`
	RetryAfter     time.Duration     `json:"retry_after,omitempty"`
	CircuitBreaker string            `json:"circuit_breaker,omitempty"`
	BackoffState   map[string]interface{} `json:"backoff_state,omitempty"`

	// Debugging information
	DebugInfo  map[string]interface{} `json:"debug_info,omitempty"`
	SourceCode string                 `json:"source_code,omitempty"`
	Hostname   string                 `json:"hostname,omitempty"`
	PID        int                    `json:"pid,omitempty"`
	GoroutineID string                `json:"goroutine_id,omitempty"`
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
	TotalCount    int64             `json:"total_count"`
	CountByType   map[string]int64  `json:"count_by_type"`
	CountByCode   map[string]int64  `json:"count_by_code"`
	MeanLatency   float64           `json:"mean_latency"`
	P95Latency    float64           `json:"p95_latency"`
	P99Latency    float64           `json:"p99_latency"`
	LastOccurrence time.Time        `json:"last_occurrence"`
	RatePerSecond float64           `json:"rate_per_second"`
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

// Error implements the error interface
func (e *ServiceError) Error() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.Details != "" {
		return fmt.Sprintf("[%s:%s:%s] %s: %s", e.Service, e.Component, e.Operation, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s:%s] %s", e.Service, e.Component, e.Operation, e.Message)
}

// String provides a detailed string representation
func (e *ServiceError) String() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return fmt.Sprintf(
		"ServiceError{Type:%s, Code:%s, Message:%s, Service:%s, Component:%s, Operation:%s, Severity:%s, Category:%s, Retryable:%t}",
		e.Type, e.Code, e.Message, e.Service, e.Component, e.Operation, e.Severity, e.Category, e.Retryable,
	)
}

// Unwrap implements the error unwrapping interface
func (e *ServiceError) Unwrap() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Cause
}

// Is implements error comparison
func (e *ServiceError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if se, ok := target.(*ServiceError); ok {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.Type == se.Type && e.Code == se.Code
	}
	
	e.mu.RLock()
	cause := e.Cause
	e.mu.RUnlock()
	
	if cause != nil {
		return fmt.Errorf("wrapped: %w", cause).Is(target)
	}
	return false
}

// As implements error type assertion
func (e *ServiceError) As(target interface{}) bool {
	if se, ok := target.(**ServiceError); ok {
		*se = e
		return true
	}
	
	e.mu.RLock()
	cause := e.Cause
	e.mu.RUnlock()
	
	if cause != nil {
		return fmt.Errorf("wrapped: %w", cause).As(target)
	}
	return false
}

// MarshalJSON implements json.Marshaler for safe concurrent access
func (e *ServiceError) MarshalJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a struct to avoid recursion
	type serviceErrorJSON ServiceError
	return json.Marshal((*serviceErrorJSON)(e))
}

// UnmarshalJSON implements json.Unmarshaler
func (e *ServiceError) UnmarshalJSON(data []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	type serviceErrorJSON ServiceError
	return json.Unmarshal(data, (*serviceErrorJSON)(e))
}

// Clone creates a deep copy of the error
func (e *ServiceError) Clone() *ServiceError {
	e.mu.RLock()
	defer e.mu.RUnlock()

	newErr := &ServiceError{
		Type:         e.Type,
		Code:         e.Code,
		Message:      e.Message,
		Details:      e.Details,
		Service:      e.Service,
		Operation:    e.Operation,
		Component:    e.Component,
		Timestamp:    e.Timestamp,
		CorrelationID: e.CorrelationID,
		RequestID:    e.RequestID,
		UserID:       e.UserID,
		SessionID:    e.SessionID,
		Category:     e.Category,
		Severity:     e.Severity,
		Impact:       e.Impact,
		Retryable:    e.Retryable,
		Temporary:    e.Temporary,
		HTTPStatus:   e.HTTPStatus,
		HTTPMethod:   e.HTTPMethod,
		HTTPPath:     e.HTTPPath,
		RemoteAddr:   e.RemoteAddr,
		UserAgent:    e.UserAgent,
		Cause:        e.Cause,
		Latency:      e.Latency,
		RetryCount:   e.RetryCount,
		RetryAfter:   e.RetryAfter,
		CircuitBreaker: e.CircuitBreaker,
		SourceCode:   e.SourceCode,
		Hostname:     e.Hostname,
		PID:          e.PID,
		GoroutineID:  e.GoroutineID,
	}

	// Deep copy slices and maps
	if e.CauseChain != nil {
		newErr.CauseChain = make([]*ServiceError, len(e.CauseChain))
		for i, cause := range e.CauseChain {
			newErr.CauseChain[i] = cause.Clone()
		}
	}

	if e.StackTrace != nil {
		newErr.StackTrace = make([]StackFrame, len(e.StackTrace))
		copy(newErr.StackTrace, e.StackTrace)
	}

	if e.Metadata != nil {
		newErr.Metadata = make(map[string]interface{}, len(e.Metadata))
		for k, v := range e.Metadata {
			newErr.Metadata[k] = v
		}
	}

	if e.Tags != nil {
		newErr.Tags = make([]string, len(e.Tags))
		copy(newErr.Tags, e.Tags)
	}

	if e.Resources != nil {
		newErr.Resources = make(map[string]string, len(e.Resources))
		for k, v := range e.Resources {
			newErr.Resources[k] = v
		}
	}

	if e.RecoveryHints != nil {
		newErr.RecoveryHints = make([]string, len(e.RecoveryHints))
		copy(newErr.RecoveryHints, e.RecoveryHints)
	}

	if e.BackoffState != nil {
		newErr.BackoffState = make(map[string]interface{}, len(e.BackoffState))
		for k, v := range e.BackoffState {
			newErr.BackoffState[k] = v
		}
	}

	if e.DebugInfo != nil {
		newErr.DebugInfo = make(map[string]interface{}, len(e.DebugInfo))
		for k, v := range e.DebugInfo {
			newErr.DebugInfo[k] = v
		}
	}

	return newErr
}

// GetSeverity returns the error severity level
func (e *ServiceError) GetSeverity() ErrorSeverity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Severity
}

// GetCategory returns the error category
func (e *ServiceError) GetCategory() ErrorCategory {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Category
}

// GetImpact returns the error impact level
func (e *ServiceError) GetImpact() ErrorImpact {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Impact
}

// IsRetryable returns whether this error is retryable
func (e *ServiceError) IsRetryable() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Retryable
}

// IsTemporary returns whether this error is temporary
func (e *ServiceError) IsTemporary() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Temporary
}

// GetHTTPStatus returns the appropriate HTTP status code for this error
func (e *ServiceError) GetHTTPStatus() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.HTTPStatus > 0 {
		return e.HTTPStatus
	}
	return getHTTPStatusForErrorType(e.Type)
}

// HasTag checks if the error has a specific tag
func (e *ServiceError) HasTag(tag string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the error (thread-safe)
func (e *ServiceError) AddTag(tag string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Check if tag already exists
	for _, t := range e.Tags {
		if t == tag {
			return
		}
	}
	
	e.Tags = append(e.Tags, tag)
}

// Helper functions for HTTP status mapping
func getHTTPStatusForErrorType(errType ErrorType) int {
	switch errType {
	case ErrorTypeValidation, ErrorTypeRequired, ErrorTypeInvalid, ErrorTypeFormat, ErrorTypeRange:
		return http.StatusBadRequest
	case ErrorTypeAuth:
		return http.StatusUnauthorized
	case ErrorTypeUnauthorized, ErrorTypeExpired:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict, ErrorTypeDuplicate:
		return http.StatusConflict
	case ErrorTypePrecondition:
		return http.StatusPreconditionFailed
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeTimeout:
		return http.StatusGatewayTimeout
	case ErrorTypeNetwork, ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeQuota, ErrorTypeResource:
		return http.StatusInsufficientStorage
	default:
		return http.StatusInternalServerError
	}
}

// Context extraction helpers
func extractContextFromHTTPRequest(r *http.Request) map[string]interface{} {
	ctx := make(map[string]interface{})
	if r != nil {
		ctx["http_method"] = r.Method
		ctx["http_path"] = r.URL.Path
		ctx["remote_addr"] = r.RemoteAddr
		ctx["user_agent"] = r.UserAgent()
		if r.Header.Get("X-Request-ID") != "" {
			ctx["request_id"] = r.Header.Get("X-Request-ID")
		}
		if r.Header.Get("X-Correlation-ID") != "" {
			ctx["correlation_id"] = r.Header.Get("X-Correlation-ID")
		}
	}
	return ctx
}

func extractContextFromContext(ctx context.Context) map[string]interface{} {
	ctxMap := make(map[string]interface{})
	
	// Extract common context values
	if reqID := ctx.Value("request_id"); reqID != nil {
		ctxMap["request_id"] = reqID
	}
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		ctxMap["correlation_id"] = corrID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		ctxMap["user_id"] = userID
	}
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		ctxMap["session_id"] = sessionID
	}
	
	return ctxMap
}

// Runtime information helpers
func getCurrentHostname() string {
	if hostname, err := runtime.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

func getCurrentPID() int {
	return runtime.Getpid()
}

func getCurrentGoroutineID() string {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// Parse goroutine ID from stack trace
	// Format: "goroutine 1 [running]:"
	for i, b := range buf {
		if b == ' ' {
			return string(buf[10:i]) // Skip "goroutine "
		}
	}
	return "unknown"
}

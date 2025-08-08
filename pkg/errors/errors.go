package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

// ErrorType represents different categories of errors
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
	ErrorTypeORANA1       ErrorType = "oran_a1"
	ErrorTypeORANO1       ErrorType = "oran_o1"
	ErrorTypeORANO2       ErrorType = "oran_o2"
	ErrorTypeORANE2       ErrorType = "oran_e2"
	ErrorTypeRIC          ErrorType = "ric"
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

// ServiceError represents a standardized error with context
type ServiceError struct {
	Type         ErrorType              `json:"type"`
	Code         string                 `json:"code"`
	Message      string                 `json:"message"`
	Details      string                 `json:"details,omitempty"`
	Service      string                 `json:"service"`
	Operation    string                 `json:"operation"`
	Timestamp    time.Time              `json:"timestamp"`
	RequestID    string                 `json:"request_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	Cause        error                  `json:"-"`
	StackTrace   []string               `json:"stack_trace,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Retryable    bool                   `json:"retryable"`
	HTTPStatus   int                    `json:"http_status,omitempty"`
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s: %s", e.Service, e.Operation, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Service, e.Operation, e.Message)
}

// Unwrap implements the error unwrapping interface
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *ServiceError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if se, ok := target.(*ServiceError); ok {
		return e.Type == se.Type && e.Code == se.Code
	}
	
	return errors.Is(e.Cause, target)
}

// ErrorBuilder helps build standardized errors
type ErrorBuilder struct {
	service   string
	operation string
	logger    *slog.Logger
}

// NewErrorBuilder creates a new error builder for a service
func NewErrorBuilder(service, operation string, logger *slog.Logger) *ErrorBuilder {
	return &ErrorBuilder{
		service:   service,
		operation: operation,
		logger:    logger,
	}
}

// ValidationError creates a validation error
func (eb *ErrorBuilder) ValidationError(code, message string) *ServiceError {
	return eb.newError(ErrorTypeValidation, code, message, 400, true)
}

// RequiredFieldError creates a required field error
func (eb *ErrorBuilder) RequiredFieldError(field string) *ServiceError {
	return eb.newError(ErrorTypeRequired, "required_field", 
		fmt.Sprintf("Required field '%s' is missing", field), 400, false)
}

// InvalidFieldError creates an invalid field error
func (eb *ErrorBuilder) InvalidFieldError(field, reason string) *ServiceError {
	return eb.newError(ErrorTypeInvalid, "invalid_field",
		fmt.Sprintf("Field '%s' is invalid: %s", field, reason), 400, false)
}

// NetworkError creates a network-related error
func (eb *ErrorBuilder) NetworkError(code, message string, cause error) *ServiceError {
	err := eb.newError(ErrorTypeNetwork, code, message, 502, true)
	err.Cause = cause
	return err
}

// TimeoutError creates a timeout error
func (eb *ErrorBuilder) TimeoutError(operation string, timeout time.Duration) *ServiceError {
	return eb.newError(ErrorTypeTimeout, "operation_timeout",
		fmt.Sprintf("Operation '%s' timed out after %v", operation, timeout), 504, true)
}

// NotFoundError creates a not found error
func (eb *ErrorBuilder) NotFoundError(resource, id string) *ServiceError {
	return eb.newError(ErrorTypeNotFound, "resource_not_found",
		fmt.Sprintf("%s with ID '%s' not found", resource, id), 404, false)
}

// InternalError creates an internal error
func (eb *ErrorBuilder) InternalError(message string, cause error) *ServiceError {
	err := eb.newError(ErrorTypeInternal, "internal_error", message, 500, false)
	err.Cause = cause
	err.StackTrace = getStackTrace(3)
	return err
}

// ConfigError creates a configuration error
func (eb *ErrorBuilder) ConfigError(setting, reason string) *ServiceError {
	return eb.newError(ErrorTypeConfig, "configuration_error",
		fmt.Sprintf("Configuration error for '%s': %s", setting, reason), 500, false)
}

// ExternalServiceError creates an external service error
func (eb *ErrorBuilder) ExternalServiceError(service string, cause error) *ServiceError {
	err := eb.newError(ErrorTypeExternal, "external_service_error",
		fmt.Sprintf("External service '%s' failed", service), 502, true)
	err.Cause = cause
	return err
}

// ContextCancelledError creates a context cancellation error
func (eb *ErrorBuilder) ContextCancelledError(ctx context.Context) *ServiceError {
	var message string
	if ctx.Err() == context.DeadlineExceeded {
		message = "Operation cancelled due to timeout"
	} else {
		message = "Operation cancelled by client"
	}
	
	err := eb.newError(ErrorTypeTimeout, "context_cancelled", message, 499, false)
	err.Cause = ctx.Err()
	return err
}

// WrapError wraps an external error with service context
func (eb *ErrorBuilder) WrapError(cause error, message string) *ServiceError {
	errorType := categorizeError(cause)
	httpStatus := getHTTPStatusForErrorType(errorType)
	retryable := isRetryableError(errorType, cause)
	
	err := eb.newError(errorType, "wrapped_error", message, httpStatus, retryable)
	err.Cause = cause
	
	if errorType == ErrorTypeInternal {
		err.StackTrace = getStackTrace(3)
	}
	
	return err
}

// newError creates a new ServiceError with common fields populated
func (eb *ErrorBuilder) newError(errType ErrorType, code, message string, httpStatus int, retryable bool) *ServiceError {
	err := &ServiceError{
		Type:       errType,
		Code:       code,
		Message:    message,
		Service:    eb.service,
		Operation:  eb.operation,
		Timestamp:  time.Now(),
		HTTPStatus: httpStatus,
		Retryable:  retryable,
		Metadata:   make(map[string]interface{}),
	}
	
	// Log the error
	if eb.logger != nil {
		logLevel := slog.LevelError
		if retryable || errType == ErrorTypeValidation {
			logLevel = slog.LevelWarn
		}
		
		eb.logger.Log(context.Background(), logLevel, "Service error created",
			"error_type", errType,
			"error_code", code,
			"message", message,
			"service", eb.service,
			"operation", eb.operation,
			"retryable", retryable,
		)
	}
	
	return err
}

// Helper functions

// categorizeError attempts to categorize an unknown error
func categorizeError(err error) ErrorType {
	if err == nil {
		return ErrorTypeInternal
	}
	
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorTypeTimeout
	case errors.Is(err, context.Canceled):
		return ErrorTypeTimeout
	default:
		return ErrorTypeInternal
	}
}

// getHTTPStatusForErrorType returns appropriate HTTP status for error type
func getHTTPStatusForErrorType(errType ErrorType) int {
	switch errType {
	case ErrorTypeValidation, ErrorTypeRequired, ErrorTypeInvalid:
		return 400
	case ErrorTypeAuth:
		return 401
	case ErrorTypeUnauthorized:
		return 401
	case ErrorTypeForbidden:
		return 403
	case ErrorTypeNotFound:
		return 404
	case ErrorTypeConflict:
		return 409
	case ErrorTypeTimeout:
		return 504
	case ErrorTypeNetwork, ErrorTypeExternal:
		return 502
	default:
		return 500
	}
}

// isRetryableError determines if an error type is generally retryable
func isRetryableError(errType ErrorType, cause error) bool {
	switch errType {
	case ErrorTypeNetwork, ErrorTypeExternal, ErrorTypeTimeout:
		return true
	case ErrorTypeDatabase:
		// Some database errors are retryable (connection issues)
		return true
	case ErrorTypeResource:
		// Resource exhaustion might be temporary
		return true
	default:
		return false
	}
}

// getStackTrace captures the current stack trace
func getStackTrace(skip int) []string {
	var stack []string
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}
	return stack
}

// WithRequestID adds a request ID to the error
func (e *ServiceError) WithRequestID(requestID string) *ServiceError {
	e.RequestID = requestID
	return e
}

// WithUserID adds a user ID to the error
func (e *ServiceError) WithUserID(userID string) *ServiceError {
	e.UserID = userID
	return e
}

// WithMetadata adds metadata to the error
func (e *ServiceError) WithMetadata(key string, value interface{}) *ServiceError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithDetails adds additional details to the error
func (e *ServiceError) WithDetails(details string) *ServiceError {
	e.Details = details
	return e
}

// ToLogAttributes converts the error to structured log attributes
func (e *ServiceError) ToLogAttributes() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("error_type", string(e.Type)),
		slog.String("error_code", e.Code),
		slog.String("service", e.Service),
		slog.String("operation", e.Operation),
		slog.Bool("retryable", e.Retryable),
	}
	
	if e.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", e.RequestID))
	}
	
	if e.UserID != "" {
		attrs = append(attrs, slog.String("user_id", e.UserID))
	}
	
	if e.Cause != nil {
		attrs = append(attrs, slog.String("cause", e.Cause.Error()))
	}
	
	return attrs
}

// IsRetryable returns whether this error is retryable
func (e *ServiceError) IsRetryable() bool {
	return e.Retryable
}

// GetHTTPStatus returns the appropriate HTTP status code for this error
func (e *ServiceError) GetHTTPStatus() int {
	if e.HTTPStatus > 0 {
		return e.HTTPStatus
	}
	return getHTTPStatusForErrorType(e.Type)
}
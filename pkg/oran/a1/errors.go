// Package a1 implements RFC 7807 compliant error handling for O-RAN A1 interfaces.

// This module provides structured error responses that comply with Problem Details for HTTP APIs.

package a1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// A1ErrorType represents the type URI for different A1 error categories.

type A1ErrorType string

const (

	// A1-P Policy Interface Error Types.

	ErrorTypePolicyTypeNotFound A1ErrorType = "urn:problem-type:a1:policy-type-not-found"

	// ErrorTypePolicyTypeAlreadyExists holds errortypepolicytypealreadyexists value.

	ErrorTypePolicyTypeAlreadyExists A1ErrorType = "urn:problem-type:a1:policy-type-already-exists"

	// ErrorTypePolicyTypeInvalidSchema holds errortypepolicytypeinvalidschema value.

	ErrorTypePolicyTypeInvalidSchema A1ErrorType = "urn:problem-type:a1:policy-type-invalid-schema"

	// ErrorTypePolicyInstanceNotFound holds errortypepolicyinstancenotfound value.

	ErrorTypePolicyInstanceNotFound A1ErrorType = "urn:problem-type:a1:policy-instance-not-found"

	// ErrorTypePolicyInstanceConflict holds errortypepolicyinstanceconflict value.

	ErrorTypePolicyInstanceConflict A1ErrorType = "urn:problem-type:a1:policy-instance-conflict"

	// ErrorTypePolicyValidationFailed holds errortypepolicyvalidationfailed value.

	ErrorTypePolicyValidationFailed A1ErrorType = "urn:problem-type:a1:policy-validation-failed"

	// ErrorTypePolicyEnforcementFailed holds errortypepolicyenforcementfailed value.

	ErrorTypePolicyEnforcementFailed A1ErrorType = "urn:problem-type:a1:policy-enforcement-failed"

	// A1-C Consumer Interface Error Types.

	ErrorTypeConsumerNotFound A1ErrorType = "urn:problem-type:a1:consumer-not-found"

	// ErrorTypeConsumerAlreadyExists holds errortypeconsumeralreadyexists value.

	ErrorTypeConsumerAlreadyExists A1ErrorType = "urn:problem-type:a1:consumer-already-exists"

	// ErrorTypeConsumerInvalidCallback holds errortypeconsumerinvalidcallback value.

	ErrorTypeConsumerInvalidCallback A1ErrorType = "urn:problem-type:a1:consumer-invalid-callback"

	// ErrorTypeConsumerNotificationFailed holds errortypeconsumernotificationfailed value.

	ErrorTypeConsumerNotificationFailed A1ErrorType = "urn:problem-type:a1:consumer-notification-failed"

	// A1-EI Enrichment Interface Error Types.

	ErrorTypeEITypeNotFound A1ErrorType = "urn:problem-type:a1:ei-type-not-found"

	// ErrorTypeEITypeAlreadyExists holds errortypeeitypealreadyexists value.

	ErrorTypeEITypeAlreadyExists A1ErrorType = "urn:problem-type:a1:ei-type-already-exists"

	// ErrorTypeEIJobNotFound holds errortypeeijobnotfound value.

	ErrorTypeEIJobNotFound A1ErrorType = "urn:problem-type:a1:ei-job-not-found"

	// ErrorTypeEIJobAlreadyExists holds errortypeeijobalreadyexists value.

	ErrorTypeEIJobAlreadyExists A1ErrorType = "urn:problem-type:a1:ei-job-already-exists"

	// ErrorTypeEIJobInvalidConfig holds errortypeeijobinvalidconfig value.

	ErrorTypeEIJobInvalidConfig A1ErrorType = "urn:problem-type:a1:ei-job-invalid-config"

	// ErrorTypeEIJobExecutionFailed holds errortypeeijobexecutionfailed value.

	ErrorTypeEIJobExecutionFailed A1ErrorType = "urn:problem-type:a1:ei-job-execution-failed"

	// Generic Error Types.

	ErrorTypeInvalidRequest A1ErrorType = "urn:problem-type:a1:invalid-request"

	// ErrorTypeInternalServerError holds errortypeinternalservererror value.

	ErrorTypeInternalServerError A1ErrorType = "urn:problem-type:a1:internal-server-error"

	// ErrorTypeServiceUnavailable holds errortypeserviceunavailable value.

	ErrorTypeServiceUnavailable A1ErrorType = "urn:problem-type:a1:service-unavailable"

	// ErrorTypeMethodNotAllowed holds errortypemethodnotallowed value.

	ErrorTypeMethodNotAllowed A1ErrorType = "urn:problem-type:a1:method-not-allowed"

	// ErrorTypeUnsupportedMediaType holds errortypeunsupportedmediatype value.

	ErrorTypeUnsupportedMediaType A1ErrorType = "urn:problem-type:a1:unsupported-media-type"

	// ErrorTypeRateLimitExceeded holds errortyperatelimitexceeded value.

	ErrorTypeRateLimitExceeded A1ErrorType = "urn:problem-type:a1:rate-limit-exceeded"

	// ErrorTypeAuthenticationRequired holds errortypeauthenticationrequired value.

	ErrorTypeAuthenticationRequired A1ErrorType = "urn:problem-type:a1:authentication-required"

	// ErrorTypeAuthorizationDenied holds errortypeauthorizationdenied value.

	ErrorTypeAuthorizationDenied A1ErrorType = "urn:problem-type:a1:authorization-denied"

	// ErrorTypeCircuitBreakerOpen holds errortypecircuitbreakeropen value.

	ErrorTypeCircuitBreakerOpen A1ErrorType = "urn:problem-type:a1:circuit-breaker-open"

	// ErrorTypeTimeout holds errortypetimeout value.

	ErrorTypeTimeout A1ErrorType = "urn:problem-type:a1:timeout"
)

// A1ProblemDetail represents RFC 7807 compliant problem details for A1 interfaces.

type A1ProblemDetail struct {
	Type string `json:"type" validate:"required,uri"`

	Title string `json:"title" validate:"required"`

	Status int `json:"status" validate:"required,min=100,max=599"`

	Detail string `json:"detail,omitempty"`

	Instance string `json:"instance,omitempty"`

	Timestamp time.Time `json:"timestamp,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	Extensions json.RawMessage `json:"-"`
}

// A1Error represents an A1-specific error with context information.

type A1Error struct {
	Type A1ErrorType `json:"type"`

	Title string `json:"title"`

	Status int `json:"status"`

	Detail string `json:"detail,omitempty"`

	Instance string `json:"instance,omitempty"`

	Cause error `json:"-"`

	Context json.RawMessage `json:"context,omitempty"`

	Retryable bool `json:"retryable,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	CorrelationID string `json:"correlation_id,omitempty"`
}

// Error implements the error interface.

func (e *A1Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("A1 Error [%s]: %s - %s", e.Type, e.Title, e.Detail)
	}

	return fmt.Sprintf("A1 Error [%s]: %s", e.Type, e.Title)
}

// Unwrap returns the underlying cause error.

func (e *A1Error) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error indicates a retryable condition.

func (e *A1Error) IsRetryable() bool {
	return e.Retryable
}

// ToProblemDetail converts A1Error to RFC 7807 compliant ProblemDetail.

func (e *A1Error) ToProblemDetail() *A1ProblemDetail {
	// Create extensions map
	extensions := make(map[string]interface{})
	
	// Add context as extensions if available
	if len(e.Context) > 0 {
		var contextMap map[string]interface{}
		if err := json.Unmarshal(e.Context, &contextMap); err == nil {
			for key, value := range contextMap {
				extensions[key] = value
			}
		}
	}

	if e.Retryable {
		extensions["retryable"] = true
	}
	
	// Marshal extensions to json.RawMessage
	extensionsBytes, err := json.Marshal(extensions)
	if err != nil {
		// If marshaling fails, use empty JSON object
		extensionsBytes = json.RawMessage("{}")
	}

	pd := &A1ProblemDetail{
		Type: string(e.Type),

		Title: e.Title,

		Status: e.Status,

		Detail: e.Detail,

		Instance: e.Instance,

		Timestamp: e.Timestamp,

		RequestID: e.CorrelationID,

		Extensions: extensionsBytes,
	}

	return pd
}

// MarshalJSON implements custom JSON marshaling to include extensions.

func (pd *A1ProblemDetail) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields including extensions.

	result := make(map[string]interface{})

	result["type"] = pd.Type

	result["title"] = pd.Title

	result["status"] = pd.Status

	if pd.Detail != "" {
		result["detail"] = pd.Detail
	}

	if pd.Instance != "" {
		result["instance"] = pd.Instance
	}

	if !pd.Timestamp.IsZero() {
		result["timestamp"] = pd.Timestamp.Format(time.RFC3339)
	}

	if pd.RequestID != "" {
		result["request_id"] = pd.RequestID
	}

	// Add extensions.
	if len(pd.Extensions) > 0 {
		var extensions map[string]interface{}
		if err := json.Unmarshal(pd.Extensions, &extensions); err == nil {
			for key, value := range extensions {
				result[key] = value
			}
		}
	}

	return json.Marshal(result)
}

// NewA1Error creates a new A1Error with the specified type and details.

func NewA1Error(errorType A1ErrorType, title string, status int, detail string) *A1Error {
	return &A1Error{
		Type: errorType,

		Title: title,

		Status: status,

		Detail: detail,

		Timestamp: time.Now(),

		Context: json.RawMessage("{}"),
	}
}

// NewA1ErrorWithCause creates a new A1Error with an underlying cause.

func NewA1ErrorWithCause(errorType A1ErrorType, title string, status int, detail string, cause error) *A1Error {
	err := NewA1Error(errorType, title, status, detail)

	err.Cause = cause

	return err
}

// WithContext adds context information to the error.

func (e *A1Error) WithContext(key string, value interface{}) *A1Error {
	// Parse existing context or create new one
	var contextMap map[string]interface{}
	if len(e.Context) > 0 {
		if err := json.Unmarshal(e.Context, &contextMap); err != nil {
			contextMap = make(map[string]interface{})
		}
	} else {
		contextMap = make(map[string]interface{})
	}

	contextMap[key] = value
	
	// Marshal back to json.RawMessage
	contextBytes, err := json.Marshal(contextMap)
	if err != nil {
		// If marshaling fails, keep the original context
		return e
	}
	e.Context = contextBytes

	return e
}

// WithInstance sets the instance URI for the error.

func (e *A1Error) WithInstance(instance string) *A1Error {
	e.Instance = instance

	return e
}

// WithCorrelationID sets the correlation ID for the error.

func (e *A1Error) WithCorrelationID(correlationID string) *A1Error {
	e.CorrelationID = correlationID

	return e
}

// AsRetryable marks the error as retryable.

func (e *A1Error) AsRetryable() *A1Error {
	e.Retryable = true

	return e
}

// Predefined A1 Errors for common scenarios.

// Policy Type Errors.

func NewPolicyTypeNotFoundError(policyTypeID int) *A1Error {
	return NewA1Error(

		ErrorTypePolicyTypeNotFound,

		"Policy Type Not Found",

		http.StatusNotFound,

		fmt.Sprintf("Policy type with ID %d does not exist", policyTypeID),
	).WithContext("policy_type_id", policyTypeID)
}

// NewPolicyTypeAlreadyExistsError performs newpolicytypealreadyexistserror operation.

func NewPolicyTypeAlreadyExistsError(policyTypeID int) *A1Error {
	return NewA1Error(

		ErrorTypePolicyTypeAlreadyExists,

		"Policy Type Already Exists",

		http.StatusConflict,

		fmt.Sprintf("Policy type with ID %d already exists", policyTypeID),
	).WithContext("policy_type_id", policyTypeID)
}

// NewPolicyTypeInvalidSchemaError performs newpolicytypeinvalidschemaerror operation.

func NewPolicyTypeInvalidSchemaError(policyTypeID int, validationError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypePolicyTypeInvalidSchema,

		"Policy Type Invalid Schema",

		http.StatusBadRequest,

		fmt.Sprintf("Policy type %d has invalid schema: %v", policyTypeID, validationError),

		validationError,
	).WithContext("policy_type_id", policyTypeID)
}

// Policy Instance Errors.

func NewPolicyInstanceNotFoundError(policyTypeID int, policyID string) *A1Error {
	return NewA1Error(

		ErrorTypePolicyInstanceNotFound,

		"Policy Instance Not Found",

		http.StatusNotFound,

		fmt.Sprintf("Policy instance %s for type %d does not exist", policyID, policyTypeID),
	).WithContext("policy_type_id", policyTypeID).WithContext("policy_id", policyID)
}

// NewPolicyInstanceConflictError performs newpolicyinstanceconflicterror operation.

func NewPolicyInstanceConflictError(policyTypeID int, policyID string) *A1Error {
	return NewA1Error(

		ErrorTypePolicyInstanceConflict,

		"Policy Instance Conflict",

		http.StatusConflict,

		fmt.Sprintf("Policy instance %s for type %d conflicts with existing instance", policyID, policyTypeID),
	).WithContext("policy_type_id", policyTypeID).WithContext("policy_id", policyID)
}

// NewPolicyValidationFailedError performs newpolicyvalidationfailederror operation.

func NewPolicyValidationFailedError(policyTypeID int, policyID string, validationError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypePolicyValidationFailed,

		"Policy Validation Failed",

		http.StatusBadRequest,

		fmt.Sprintf("Policy instance %s validation failed: %v", policyID, validationError),

		validationError,
	).WithContext("policy_type_id", policyTypeID).WithContext("policy_id", policyID)
}

// NewPolicyEnforcementFailedError performs newpolicyenforcementfailederror operation.

func NewPolicyEnforcementFailedError(policyTypeID int, policyID string, enforcementError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypePolicyEnforcementFailed,

		"Policy Enforcement Failed",

		http.StatusInternalServerError,

		fmt.Sprintf("Failed to enforce policy instance %s: %v", policyID, enforcementError),

		enforcementError,
	).WithContext("policy_type_id", policyTypeID).WithContext("policy_id", policyID).AsRetryable()
}

// Consumer Errors.

func NewConsumerNotFoundError(consumerID string) *A1Error {
	return NewA1Error(

		ErrorTypeConsumerNotFound,

		"Consumer Not Found",

		http.StatusNotFound,

		fmt.Sprintf("Consumer with ID %s does not exist", consumerID),
	).WithContext("consumer_id", consumerID)
}

// NewConsumerAlreadyExistsError performs newconsumeralreadyexistserror operation.

func NewConsumerAlreadyExistsError(consumerID string) *A1Error {
	return NewA1Error(

		ErrorTypeConsumerAlreadyExists,

		"Consumer Already Exists",

		http.StatusConflict,

		fmt.Sprintf("Consumer with ID %s already exists", consumerID),
	).WithContext("consumer_id", consumerID)
}

// NewConsumerInvalidCallbackError performs newconsumerinvalidcallbackerror operation.

func NewConsumerInvalidCallbackError(consumerID, callbackURL string, validationError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypeConsumerInvalidCallback,

		"Consumer Invalid Callback",

		http.StatusBadRequest,

		fmt.Sprintf("Consumer %s has invalid callback URL %s: %v", consumerID, callbackURL, validationError),

		validationError,
	).WithContext("consumer_id", consumerID).WithContext("callback_url", callbackURL)
}

// NewConsumerNotificationFailedError performs newconsumernotificationfailederror operation.

func NewConsumerNotificationFailedError(consumerID string, notificationError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypeConsumerNotificationFailed,

		"Consumer Notification Failed",

		http.StatusInternalServerError,

		fmt.Sprintf("Failed to notify consumer %s: %v", consumerID, notificationError),

		notificationError,
	).WithContext("consumer_id", consumerID).AsRetryable()
}

// Enrichment Information Errors.

func NewEITypeNotFoundError(eiTypeID string) *A1Error {
	return NewA1Error(

		ErrorTypeEITypeNotFound,

		"EI Type Not Found",

		http.StatusNotFound,

		fmt.Sprintf("Enrichment Information type %s does not exist", eiTypeID),
	).WithContext("ei_type_id", eiTypeID)
}

// NewEITypeAlreadyExistsError performs neweitypealreadyexistserror operation.

func NewEITypeAlreadyExistsError(eiTypeID string) *A1Error {
	return NewA1Error(

		ErrorTypeEITypeAlreadyExists,

		"EI Type Already Exists",

		http.StatusConflict,

		fmt.Sprintf("Enrichment Information type %s already exists", eiTypeID),
	).WithContext("ei_type_id", eiTypeID)
}

// NewEIJobNotFoundError performs neweijobnotfounderror operation.

func NewEIJobNotFoundError(eiJobID string) *A1Error {
	return NewA1Error(

		ErrorTypeEIJobNotFound,

		"EI Job Not Found",

		http.StatusNotFound,

		fmt.Sprintf("Enrichment Information job %s does not exist", eiJobID),
	).WithContext("ei_job_id", eiJobID)
}

// NewEIJobAlreadyExistsError performs neweijobalreadyexistserror operation.

func NewEIJobAlreadyExistsError(eiJobID string) *A1Error {
	return NewA1Error(

		ErrorTypeEIJobAlreadyExists,

		"EI Job Already Exists",

		http.StatusConflict,

		fmt.Sprintf("Enrichment Information job %s already exists", eiJobID),
	).WithContext("ei_job_id", eiJobID)
}

// NewEIJobInvalidConfigError performs neweijobinvalidconfigerror operation.

func NewEIJobInvalidConfigError(eiJobID string, configError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypeEIJobInvalidConfig,

		"EI Job Invalid Configuration",

		http.StatusBadRequest,

		fmt.Sprintf("Enrichment Information job %s has invalid configuration: %v", eiJobID, configError),

		configError,
	).WithContext("ei_job_id", eiJobID)
}

// NewEIJobExecutionFailedError performs neweijobexecutionfailederror operation.

func NewEIJobExecutionFailedError(eiJobID string, executionError error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypeEIJobExecutionFailed,

		"EI Job Execution Failed",

		http.StatusInternalServerError,

		fmt.Sprintf("Enrichment Information job %s execution failed: %v", eiJobID, executionError),

		executionError,
	).WithContext("ei_job_id", eiJobID).AsRetryable()
}

// Generic Errors.

func NewInvalidRequestError(detail string) *A1Error {
	return NewA1Error(

		ErrorTypeInvalidRequest,

		"Invalid Request",

		http.StatusBadRequest,

		detail,
	)
}

// NewInternalServerError performs newinternalservererror operation.

func NewInternalServerError(detail string, cause error) *A1Error {
	return NewA1ErrorWithCause(

		ErrorTypeInternalServerError,

		"Internal Server Error",

		http.StatusInternalServerError,

		detail,

		cause,
	).AsRetryable()
}

// NewServiceUnavailableError performs newserviceunavailableerror operation.

func NewServiceUnavailableError(detail string) *A1Error {
	return NewA1Error(

		ErrorTypeServiceUnavailable,

		"Service Unavailable",

		http.StatusServiceUnavailable,

		detail,
	).AsRetryable()
}

// NewMethodNotAllowedError performs newmethodnotallowederror operation.

func NewMethodNotAllowedError(method string, allowedMethods []string) *A1Error {
	return NewA1Error(

		ErrorTypeMethodNotAllowed,

		"Method Not Allowed",

		http.StatusMethodNotAllowed,

		fmt.Sprintf("Method %s is not allowed", method),
	).WithContext("method", method).WithContext("allowed_methods", allowedMethods)
}

// NewUnsupportedMediaTypeError performs newunsupportedmediatypeerror operation.

func NewUnsupportedMediaTypeError(contentType string, supportedTypes []string) *A1Error {
	return NewA1Error(

		ErrorTypeUnsupportedMediaType,

		"Unsupported Media Type",

		http.StatusUnsupportedMediaType,

		fmt.Sprintf("Content type %s is not supported", contentType),
	).WithContext("content_type", contentType).WithContext("supported_types", supportedTypes)
}

// NewRateLimitExceededError performs newratelimitexceedederror operation.

func NewRateLimitExceededError(limit int, windowSize time.Duration) *A1Error {
	return NewA1Error(

		ErrorTypeRateLimitExceeded,

		"Rate Limit Exceeded",

		http.StatusTooManyRequests,

		fmt.Sprintf("Rate limit of %d requests per %v exceeded", limit, windowSize),
	).WithContext("rate_limit", limit).WithContext("window_size", windowSize.String()).AsRetryable()
}

// NewAuthenticationRequiredError performs newauthenticationrequirederror operation.

func NewAuthenticationRequiredError() *A1Error {
	return NewA1Error(

		ErrorTypeAuthenticationRequired,

		"Authentication Required",

		http.StatusUnauthorized,

		"Authentication credentials are required to access this resource",
	)
}

// NewAuthorizationDeniedError performs newauthorizationdeniederror operation.

func NewAuthorizationDeniedError(resource string) *A1Error {
	return NewA1Error(

		ErrorTypeAuthorizationDenied,

		"Authorization Denied",

		http.StatusForbidden,

		fmt.Sprintf("Access to resource %s is denied", resource),
	).WithContext("resource", resource)
}

// NewCircuitBreakerOpenError performs newcircuitbreakeropenerror operation.

func NewCircuitBreakerOpenError(circuitName string) *A1Error {
	return NewA1Error(

		ErrorTypeCircuitBreakerOpen,

		"Circuit Breaker Open",

		http.StatusServiceUnavailable,

		fmt.Sprintf("Circuit breaker %s is open, requests are being rejected", circuitName),
	).WithContext("circuit_name", circuitName).AsRetryable()
}

// NewTimeoutError performs newtimeouterror operation.

func NewTimeoutError(operation string, timeout time.Duration) *A1Error {
	return NewA1Error(

		ErrorTypeTimeout,

		"Operation Timeout",

		http.StatusRequestTimeout,

		fmt.Sprintf("Operation %s timed out after %v", operation, timeout),
	).WithContext("operation", operation).WithContext("timeout", timeout.String()).AsRetryable()
}

// WriteA1Error writes an A1Error as an RFC 7807 compliant response.

func WriteA1Error(w http.ResponseWriter, err *A1Error) {
	w.Header().Set("Content-Type", ContentTypeProblemJSON)

	w.WriteHeader(err.Status)

	problemDetail := err.ToProblemDetail()

	if jsonData, jsonErr := json.MarshalIndent(problemDetail, "", "  "); jsonErr == nil {
		w.Write(jsonData)
	} else {
		// Fallback to simple error message if JSON marshaling fails.

		fmt.Fprintf(w, `{"type":"%s","title":"%s","status":%d}`, err.Type, err.Title, err.Status)
	}
}

// ExtractA1ErrorFromHTTPResponse extracts A1Error from an HTTP response.

func ExtractA1ErrorFromHTTPResponse(resp *http.Response) (*A1Error, error) {
	if resp.StatusCode < 400 {
		return nil, fmt.Errorf("response does not contain an error (status: %d)", resp.StatusCode)
	}

	var problemDetail A1ProblemDetail

	if err := json.NewDecoder(resp.Body).Decode(&problemDetail); err != nil {
		// Return generic error if response is not in problem+json format.

		return &A1Error{
			Type: ErrorTypeInternalServerError,

			Title: http.StatusText(resp.StatusCode),

			Status: resp.StatusCode,

			Timestamp: time.Now(),
		}, nil
	}

	a1Error := &A1Error{
		Type: A1ErrorType(problemDetail.Type),

		Title: problemDetail.Title,

		Status: problemDetail.Status,

		Detail: problemDetail.Detail,

		Instance: problemDetail.Instance,

		Timestamp: problemDetail.Timestamp,

		CorrelationID: problemDetail.RequestID,

		Context: problemDetail.Extensions,
	}

	// Check if error is marked as retryable.
	if len(problemDetail.Extensions) > 0 {
		var extensions map[string]interface{}
		if err := json.Unmarshal(problemDetail.Extensions, &extensions); err == nil {
			if retryable, ok := extensions["retryable"].(bool); ok {
				a1Error.Retryable = retryable
			}
		}
	}

	return a1Error, nil
}

// IsA1Error checks if an error is an A1Error.

func IsA1Error(err error) bool {
	var a1Err *A1Error

	return errors.As(err, &a1Err)
}

// GetA1Error extracts A1Error from an error, returning nil if not an A1Error.

func GetA1Error(err error) *A1Error {
	var a1Err *A1Error

	if errors.As(err, &a1Err) {
		return a1Err
	}

	return nil
}

// WrapError wraps a generic error as an A1 internal server error.

func WrapError(err error, detail string) *A1Error {
	return NewInternalServerError(detail, err)
}

// ErrorHandler is a middleware function type for handling A1 errors.

type ErrorHandler func(http.ResponseWriter, *http.Request, *A1Error)

// DefaultErrorHandler provides default error handling for A1 errors.

func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err *A1Error) {
	// Add request ID if available in context.

	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		err.CorrelationID = requestID
	}

	// Add instance URI based on request.

	if err.Instance == "" {
		err.Instance = r.URL.Path
	}

	WriteA1Error(w, err)
}

// ErrorMiddleware provides middleware for consistent error handling.

func ErrorMiddleware(errorHandler ErrorHandler) func(http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {

					var err *A1Error

					if a1Err, ok := rec.(*A1Error); ok {
						err = a1Err
					} else if genericErr, ok := rec.(error); ok {
						err = NewInternalServerError("Unexpected error occurred", genericErr)
					} else {
						err = NewInternalServerError(fmt.Sprintf("Panic: %v", rec), nil)
					}

					errorHandler(w, r, err)

				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

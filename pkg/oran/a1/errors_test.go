// Package a1 provides comprehensive unit tests for A1 error handling and RFC 7807 compliance
package a1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test RFC 7807 Error Structure

func TestA1Error_Structure(t *testing.T) {
	err := &A1Error{
		Type:     ErrorTypePolicyTypeNotFound,
		Title:    "Policy Type Not Found",
		Status:   http.StatusNotFound,
		Detail:   "The requested policy type with ID 123 was not found",
		Instance: "/A1-P/v2/policytypes/123",
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"policy_type_id": 123,
			"requested_by":   "test-client",
		},
	}

	assert.Equal(t, ErrorTypePolicyTypeNotFound, err.Type)
	assert.Equal(t, "Policy Type Not Found", err.Title)
	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.Equal(t, "The requested policy type with ID 123 was not found", err.Detail)
	assert.Equal(t, "/A1-P/v2/policytypes/123", err.Instance)
	assert.NotEmpty(t, err.Timestamp)
	assert.Equal(t, 123, err.Extensions["policy_type_id"])
}

func TestA1Error_JSON_Serialization(t *testing.T) {
	originalErr := &A1Error{
		Type:     ErrorTypeInvalidRequest,
		Title:    "Invalid Request",
		Status:   http.StatusBadRequest,
		Detail:   "The request body contains invalid JSON",
		Instance: "/A1-P/v2/policytypes/1",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Extensions: map[string]interface{}{
			"validation_errors": []string{"missing required field: schema"},
			"request_id":       "req-123",
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(originalErr)
	require.NoError(t, err)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	assert.Equal(t, string(ErrorTypeInvalidRequest), jsonMap["type"])
	assert.Equal(t, "Invalid Request", jsonMap["title"])
	assert.Equal(t, float64(http.StatusBadRequest), jsonMap["status"]) // JSON numbers are float64
	assert.Equal(t, "The request body contains invalid JSON", jsonMap["detail"])
	assert.Equal(t, "/A1-P/v2/policytypes/1", jsonMap["instance"])
	assert.Contains(t, jsonMap, "timestamp")
	assert.Contains(t, jsonMap, "validation_errors")
	assert.Contains(t, jsonMap, "request_id")

	// Deserialize back
	var deserializedErr A1Error
	err = json.Unmarshal(jsonData, &deserializedErr)
	require.NoError(t, err)

	assert.Equal(t, originalErr.Type, deserializedErr.Type)
	assert.Equal(t, originalErr.Title, deserializedErr.Title)
	assert.Equal(t, originalErr.Status, deserializedErr.Status)
	assert.Equal(t, originalErr.Detail, deserializedErr.Detail)
	assert.Equal(t, originalErr.Instance, deserializedErr.Instance)
}

func TestA1Error_Error_Method(t *testing.T) {
	err := &A1Error{
		Type:   ErrorTypePolicyInstanceNotFound,
		Title:  "Policy Instance Not Found", 
		Detail: "Policy instance 'test-policy-1' not found for policy type 123",
		Status: http.StatusNotFound,
	}

	errorString := err.Error()
	assert.Contains(t, errorString, "Policy Instance Not Found")
	assert.Contains(t, errorString, "Policy instance 'test-policy-1' not found for policy type 123")
}

// Test Error Constructors

func TestNewA1Error_Basic(t *testing.T) {
	err := NewA1Error(ErrorTypeInvalidRequest, "Invalid JSON syntax", http.StatusBadRequest, nil)

	assert.Equal(t, ErrorTypeInvalidRequest, err.Type)
	assert.Equal(t, "Invalid JSON syntax", err.Title)
	assert.Equal(t, "Invalid JSON syntax", err.Detail)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.NotZero(t, err.Timestamp)
	assert.Nil(t, err.Cause)
}

func TestNewA1Error_WithCause(t *testing.T) {
	cause := fmt.Errorf("JSON parsing failed")
	err := NewA1Error(ErrorTypeInvalidRequest, "Invalid request body", http.StatusBadRequest, cause)

	assert.Equal(t, ErrorTypeInvalidRequest, err.Type)
	assert.Equal(t, "Invalid request body", err.Title)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, cause, err.Cause)
}

func TestNewPolicyTypeNotFoundError(t *testing.T) {
	err := NewPolicyTypeNotFoundError(123)

	assert.Equal(t, ErrorTypePolicyTypeNotFound, err.Type)
	assert.Equal(t, "Policy Type Not Found", err.Title)
	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.Contains(t, err.Detail, "123")
	assert.Equal(t, 123, err.Extensions["policy_type_id"])
}

func TestNewPolicyInstanceNotFoundError(t *testing.T) {
	err := NewPolicyInstanceNotFoundError(123, "test-policy-1")

	assert.Equal(t, ErrorTypePolicyInstanceNotFound, err.Type)
	assert.Equal(t, "Policy Instance Not Found", err.Title)
	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.Contains(t, err.Detail, "test-policy-1")
	assert.Contains(t, err.Detail, "123")
	assert.Equal(t, 123, err.Extensions["policy_type_id"])
	assert.Equal(t, "test-policy-1", err.Extensions["policy_id"])
}

func TestNewPolicyValidationError(t *testing.T) {
	validationErrors := []ValidationDetail{
		{Field: "qos_class", Message: "must be between 1 and 9", Value: 15},
		{Field: "action", Message: "must be one of: allow, deny, redirect", Value: "invalid_action"},
	}

	err := NewPolicyValidationError("Policy data validation failed", validationErrors)

	assert.Equal(t, ErrorTypePolicyValidationFailed, err.Type)
	assert.Equal(t, "Policy Validation Failed", err.Title)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Contains(t, err.Detail, "Policy data validation failed")
	assert.Equal(t, validationErrors, err.Extensions["validation_errors"])
}

func TestNewCircuitBreakerOpenError(t *testing.T) {
	err := NewCircuitBreakerOpenError("ric-connection")

	assert.Equal(t, ErrorTypeCircuitBreakerOpen, err.Type)
	assert.Equal(t, "Circuit Breaker Open", err.Title)
	assert.Equal(t, http.StatusServiceUnavailable, err.Status)
	assert.Contains(t, err.Detail, "ric-connection")
	assert.Equal(t, "ric-connection", err.Extensions["circuit_breaker"])
}

// Test Error Type Categories

func TestErrorTypes_A1P_PolicyInterface(t *testing.T) {
	policyErrorTypes := []A1ErrorType{
		ErrorTypePolicyTypeNotFound,
		ErrorTypePolicyTypeAlreadyExists,
		ErrorTypePolicyTypeInvalidSchema,
		ErrorTypePolicyInstanceNotFound,
		ErrorTypePolicyInstanceConflict,
		ErrorTypePolicyValidationFailed,
		ErrorTypePolicyEnforcementFailed,
	}

	for _, errorType := range policyErrorTypes {
		t.Run(string(errorType), func(t *testing.T) {
			assert.Contains(t, string(errorType), "a1:policy")
			assert.True(t, strings.HasPrefix(string(errorType), "urn:problem-type:"))
		})
	}
}

func TestErrorTypes_A1C_ConsumerInterface(t *testing.T) {
	consumerErrorTypes := []A1ErrorType{
		ErrorTypeConsumerNotFound,
		ErrorTypeConsumerAlreadyExists,
		ErrorTypeConsumerInvalidCallback,
		ErrorTypeConsumerNotificationFailed,
	}

	for _, errorType := range consumerErrorTypes {
		t.Run(string(errorType), func(t *testing.T) {
			assert.Contains(t, string(errorType), "a1:consumer")
			assert.True(t, strings.HasPrefix(string(errorType), "urn:problem-type:"))
		})
	}
}

func TestErrorTypes_A1EI_EnrichmentInterface(t *testing.T) {
	enrichmentErrorTypes := []A1ErrorType{
		ErrorTypeEITypeNotFound,
		ErrorTypeEITypeAlreadyExists,
		ErrorTypeEIJobNotFound,
		ErrorTypeEIJobAlreadyExists,
		ErrorTypeEIJobInvalidConfig,
		ErrorTypeEIJobExecutionFailed,
	}

	for _, errorType := range enrichmentErrorTypes {
		t.Run(string(errorType), func(t *testing.T) {
			assert.Contains(t, string(errorType), "a1:ei")
			assert.True(t, strings.HasPrefix(string(errorType), "urn:problem-type:"))
		})
	}
}

func TestErrorTypes_Generic(t *testing.T) {
	genericErrorTypes := []A1ErrorType{
		ErrorTypeInvalidRequest,
		ErrorTypeInternalServerError,
		ErrorTypeServiceUnavailable,
		ErrorTypeMethodNotAllowed,
		ErrorTypeUnsupportedMediaType,
		ErrorTypeRateLimitExceeded,
		ErrorTypeAuthenticationRequired,
		ErrorTypeAuthorizationDenied,
		ErrorTypeCircuitBreakerOpen,
		ErrorTypeTimeout,
	}

	for _, errorType := range genericErrorTypes {
		t.Run(string(errorType), func(t *testing.T) {
			assert.True(t, strings.HasPrefix(string(errorType), "urn:problem-type:a1:"))
		})
	}
}

// Test Error Response Writing

func TestWriteA1Error_BasicError(t *testing.T) {
	err := NewA1Error(ErrorTypePolicyTypeNotFound, "Policy type not found", http.StatusNotFound, nil)
	err.Instance = "/A1-P/v2/policytypes/123"

	rr := httptest.NewRecorder()
	WriteA1Error(rr, err)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))
	assert.NotEmpty(t, rr.Header().Get("X-Error-ID"))

	var response map[string]interface{}
	err2 := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err2)

	assert.Equal(t, string(ErrorTypePolicyTypeNotFound), response["type"])
	assert.Equal(t, "Policy type not found", response["title"])
	assert.Equal(t, float64(http.StatusNotFound), response["status"])
	assert.Equal(t, "/A1-P/v2/policytypes/123", response["instance"])
}

func TestWriteA1Error_WithExtensions(t *testing.T) {
	err := &A1Error{
		Type:   ErrorTypePolicyValidationFailed,
		Title:  "Validation Failed",
		Status: http.StatusBadRequest,
		Detail: "Multiple validation errors",
		Extensions: map[string]interface{}{
			"validation_errors": []ValidationDetail{
				{Field: "qos_class", Message: "invalid range", Value: 15},
			},
			"policy_type_id": 123,
		},
	}

	rr := httptest.NewRecorder()
	WriteA1Error(rr, err)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var response map[string]interface{}
	err2 := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err2)

	assert.Contains(t, response, "validation_errors")
	assert.Contains(t, response, "policy_type_id")
	assert.Equal(t, float64(123), response["policy_type_id"])
}

func TestWriteA1Error_NonA1Error(t *testing.T) {
	genericErr := fmt.Errorf("generic error message")

	rr := httptest.NewRecorder()
	WriteA1Error(rr, genericErr)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, string(ErrorTypeInternalServerError), response["type"])
	assert.Equal(t, "Internal Server Error", response["title"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["status"])
	assert.Contains(t, response["detail"], "generic error message")
}

func TestWriteA1Error_NilError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteA1Error(rr, nil)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, string(ErrorTypeInternalServerError), response["type"])
}

// Test Error Middleware

func TestErrorMiddleware_HandlesA1Errors(t *testing.T) {
	handler := ErrorMiddleware(func(w http.ResponseWriter, r *http.Request, err error) {
		WriteA1Error(w, err)
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := NewPolicyTypeNotFoundError(123)
		WriteA1Error(w, err)
	})

	wrappedHandler := handler(testHandler)

	req := httptest.NewRequest("GET", "/A1-P/v2/policytypes/123", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))
}

func TestErrorMiddleware_HandlesNonA1Errors(t *testing.T) {
	handler := ErrorMiddleware(DefaultErrorHandler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected panic")
	})

	wrappedHandler := handler(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		wrappedHandler.ServeHTTP(rr, req)
	})
}

// Test Error Conversion

func TestConvertToA1Error_DatabaseError(t *testing.T) {
	dbErr := fmt.Errorf("database connection failed")
	a1Err := ConvertToA1Error(dbErr, "storage operation failed")

	assert.Equal(t, ErrorTypeInternalServerError, a1Err.Type)
	assert.Equal(t, "Internal Server Error", a1Err.Title)
	assert.Equal(t, http.StatusInternalServerError, a1Err.Status)
	assert.Contains(t, a1Err.Detail, "storage operation failed")
	assert.Equal(t, dbErr, a1Err.Cause)
}

func TestConvertToA1Error_ValidationError(t *testing.T) {
	validationErr := fmt.Errorf("field 'qos_class' is invalid")
	a1Err := ConvertValidationError(validationErr, "policy_data")

	assert.Equal(t, ErrorTypePolicyValidationFailed, a1Err.Type)
	assert.Equal(t, "Policy Validation Failed", a1Err.Title)
	assert.Equal(t, http.StatusBadRequest, a1Err.Status)
	assert.Contains(t, a1Err.Detail, "validation failed")
	assert.Equal(t, "policy_data", a1Err.Extensions["field"])
}

func TestConvertToA1Error_HTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectedType A1ErrorType
	}{
		{"Bad Request", http.StatusBadRequest, ErrorTypeInvalidRequest},
		{"Unauthorized", http.StatusUnauthorized, ErrorTypeAuthenticationRequired},
		{"Forbidden", http.StatusForbidden, ErrorTypeAuthorizationDenied},
		{"Not Found", http.StatusNotFound, ErrorTypeInternalServerError}, // Generic fallback
		{"Method Not Allowed", http.StatusMethodNotAllowed, ErrorTypeMethodNotAllowed},
		{"Unsupported Media Type", http.StatusUnsupportedMediaType, ErrorTypeUnsupportedMediaType},
		{"Too Many Requests", http.StatusTooManyRequests, ErrorTypeRateLimitExceeded},
		{"Internal Server Error", http.StatusInternalServerError, ErrorTypeInternalServerError},
		{"Service Unavailable", http.StatusServiceUnavailable, ErrorTypeServiceUnavailable},
		{"Gateway Timeout", http.StatusGatewayTimeout, ErrorTypeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConvertHTTPStatusToA1Error(tt.statusCode, "test message")
			assert.Equal(t, tt.expectedType, err.Type)
			assert.Equal(t, tt.statusCode, err.Status)
		})
	}
}

// Test Error Logging

func TestA1Error_LogFields(t *testing.T) {
	err := &A1Error{
		Type:     ErrorTypePolicyInstanceConflict,
		Title:    "Policy Instance Conflict",
		Status:   http.StatusConflict,
		Detail:   "Policy instance already exists",
		Instance: "/A1-P/v2/policytypes/1/policies/test-policy",
		Extensions: map[string]interface{}{
			"policy_type_id": 1,
			"policy_id":      "test-policy",
		},
	}

	logFields := err.LogFields()

	assert.Equal(t, string(ErrorTypePolicyInstanceConflict), logFields["error_type"])
	assert.Equal(t, "Policy Instance Conflict", logFields["error_title"])
	assert.Equal(t, http.StatusConflict, logFields["error_status"])
	assert.Equal(t, "Policy instance already exists", logFields["error_detail"])
	assert.Equal(t, "/A1-P/v2/policytypes/1/policies/test-policy", logFields["error_instance"])
	assert.Equal(t, 1, logFields["policy_type_id"])
	assert.Equal(t, "test-policy", logFields["policy_id"])
}

// Test Error Wrapping and Unwrapping

func TestA1Error_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause error")
	err := NewA1Error(ErrorTypeInternalServerError, "wrapper error", http.StatusInternalServerError, cause)

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestA1Error_Is(t *testing.T) {
	err1 := NewA1Error(ErrorTypePolicyTypeNotFound, "not found", http.StatusNotFound, nil)
	err2 := NewA1Error(ErrorTypePolicyTypeNotFound, "not found", http.StatusNotFound, nil)
	err3 := NewA1Error(ErrorTypeInvalidRequest, "bad request", http.StatusBadRequest, nil)

	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(fmt.Errorf("generic error")))
}

// Test Error Recovery

func TestRecoverFromPanic_A1Error(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			err := RecoverToA1Error(r, "panic occurred during request processing")
			
			assert.Equal(t, ErrorTypeInternalServerError, err.Type)
			assert.Equal(t, http.StatusInternalServerError, err.Status)
			assert.Contains(t, err.Detail, "panic occurred")
			assert.Contains(t, err.Extensions, "panic_value")
		}
	}()

	panic("test panic")
}

// Test Concurrent Error Handling

func TestA1Error_ConcurrentAccess(t *testing.T) {
	err := NewA1Error(ErrorTypeInternalServerError, "concurrent test", http.StatusInternalServerError, nil)

	// Test concurrent access to error fields
	done := make(chan bool)
	const numGoroutines = 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Access various fields
			_ = err.Error()
			_ = err.Type
			_ = err.Status
			
			// Modify extensions (should be safe if properly implemented)
			if err.Extensions == nil {
				err.Extensions = make(map[string]interface{})
			}
			err.Extensions["test"] = "value"
			
			// Serialize to JSON
			jsonData, _ := json.Marshal(err)
			var testErr A1Error
			json.Unmarshal(jsonData, &testErr)
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	assert.Equal(t, ErrorTypeInternalServerError, err.Type)
}

// Benchmarks

func BenchmarkWriteA1Error(b *testing.B) {
	err := NewPolicyTypeNotFoundError(123)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		WriteA1Error(rr, err)
	}
}

func BenchmarkA1Error_JSON_Marshal(b *testing.B) {
	err := &A1Error{
		Type:     ErrorTypePolicyValidationFailed,
		Title:    "Validation Failed",
		Status:   http.StatusBadRequest,
		Detail:   "Multiple validation errors occurred",
		Instance: "/A1-P/v2/policytypes/1/policies/test",
		Extensions: map[string]interface{}{
			"validation_errors": []ValidationDetail{
				{Field: "qos_class", Message: "invalid range", Value: 15},
				{Field: "action", Message: "invalid enum", Value: "invalid"},
			},
			"policy_type_id": 1,
			"policy_id":      "test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(err)
	}
}

func BenchmarkErrorMiddleware(b *testing.B) {
	handler := ErrorMiddleware(DefaultErrorHandler)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteA1Error(w, NewPolicyTypeNotFoundError(123))
	})
	wrappedHandler := handler(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}

// Additional types and functions for comprehensive error handling

type ValidationDetail struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// Enhanced A1Error with additional fields for comprehensive error handling
type A1Error struct {
	Type       A1ErrorType            `json:"type"`
	Title      string                 `json:"title"`
	Status     int                    `json:"status"`
	Detail     string                 `json:"detail"`
	Instance   string                 `json:"instance,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Extensions map[string]interface{} `json:"-"` // Flattened in JSON
	Cause      error                  `json:"-"` // Not serialized
}

func (e *A1Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Type, e.Title, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Title)
}

func (e *A1Error) Unwrap() error {
	return e.Cause
}

func (e *A1Error) Is(target error) bool {
	if other, ok := target.(*A1Error); ok {
		return e.Type == other.Type && e.Status == other.Status
	}
	return false
}

func (e *A1Error) MarshalJSON() ([]byte, error) {
	// Create a map with all fields
	result := map[string]interface{}{
		"type":      e.Type,
		"title":     e.Title,
		"status":    e.Status,
		"detail":    e.Detail,
		"timestamp": e.Timestamp,
	}

	if e.Instance != "" {
		result["instance"] = e.Instance
	}

	// Add extensions to the top level
	for k, v := range e.Extensions {
		result[k] = v
	}

	return json.Marshal(result)
}

func (e *A1Error) LogFields() map[string]interface{} {
	fields := map[string]interface{}{
		"error_type":   string(e.Type),
		"error_title":  e.Title,
		"error_status": e.Status,
		"error_detail": e.Detail,
	}

	if e.Instance != "" {
		fields["error_instance"] = e.Instance
	}

	// Add extensions
	for k, v := range e.Extensions {
		fields[k] = v
	}

	return fields
}

// Error constructors

func NewA1Error(errorType A1ErrorType, message string, statusCode int, cause error) *A1Error {
	return &A1Error{
		Type:       errorType,
		Title:      message,
		Detail:     message,
		Status:     statusCode,
		Timestamp:  time.Now(),
		Extensions: make(map[string]interface{}),
		Cause:      cause,
	}
}

func NewPolicyTypeNotFoundError(policyTypeID int) *A1Error {
	err := &A1Error{
		Type:      ErrorTypePolicyTypeNotFound,
		Title:     "Policy Type Not Found",
		Status:    http.StatusNotFound,
		Detail:    fmt.Sprintf("Policy type with ID %d was not found", policyTypeID),
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"policy_type_id": policyTypeID,
		},
	}
	return err
}

func NewPolicyInstanceNotFoundError(policyTypeID int, policyID string) *A1Error {
	err := &A1Error{
		Type:      ErrorTypePolicyInstanceNotFound,
		Title:     "Policy Instance Not Found",
		Status:    http.StatusNotFound,
		Detail:    fmt.Sprintf("Policy instance '%s' not found for policy type %d", policyID, policyTypeID),
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"policy_type_id": policyTypeID,
			"policy_id":      policyID,
		},
	}
	return err
}

func NewPolicyValidationError(message string, validationErrors []ValidationDetail) *A1Error {
	err := &A1Error{
		Type:      ErrorTypePolicyValidationFailed,
		Title:     "Policy Validation Failed",
		Status:    http.StatusBadRequest,
		Detail:    message,
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"validation_errors": validationErrors,
		},
	}
	return err
}

func NewCircuitBreakerOpenError(circuitBreakerName string) *A1Error {
	err := &A1Error{
		Type:      ErrorTypeCircuitBreakerOpen,
		Title:     "Circuit Breaker Open",
		Status:    http.StatusServiceUnavailable,
		Detail:    fmt.Sprintf("Circuit breaker '%s' is open due to repeated failures", circuitBreakerName),
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"circuit_breaker": circuitBreakerName,
		},
	}
	return err
}

// Error conversion functions

func ConvertToA1Error(err error, context string) *A1Error {
	if a1Err, ok := err.(*A1Error); ok {
		return a1Err
	}

	return &A1Error{
		Type:      ErrorTypeInternalServerError,
		Title:     "Internal Server Error",
		Status:    http.StatusInternalServerError,
		Detail:    fmt.Sprintf("%s: %s", context, err.Error()),
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"context": context,
		},
		Cause: err,
	}
}

func ConvertValidationError(err error, field string) *A1Error {
	return &A1Error{
		Type:      ErrorTypePolicyValidationFailed,
		Title:     "Policy Validation Failed",
		Status:    http.StatusBadRequest,
		Detail:    fmt.Sprintf("Validation failed for field '%s': %s", field, err.Error()),
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"field":            field,
			"validation_error": err.Error(),
		},
		Cause: err,
	}
}

func ConvertHTTPStatusToA1Error(statusCode int, message string) *A1Error {
	var errorType A1ErrorType
	var title string

	switch statusCode {
	case http.StatusBadRequest:
		errorType = ErrorTypeInvalidRequest
		title = "Bad Request"
	case http.StatusUnauthorized:
		errorType = ErrorTypeAuthenticationRequired
		title = "Authentication Required"
	case http.StatusForbidden:
		errorType = ErrorTypeAuthorizationDenied
		title = "Authorization Denied"
	case http.StatusMethodNotAllowed:
		errorType = ErrorTypeMethodNotAllowed
		title = "Method Not Allowed"
	case http.StatusUnsupportedMediaType:
		errorType = ErrorTypeUnsupportedMediaType
		title = "Unsupported Media Type"
	case http.StatusTooManyRequests:
		errorType = ErrorTypeRateLimitExceeded
		title = "Rate Limit Exceeded"
	case http.StatusServiceUnavailable:
		errorType = ErrorTypeServiceUnavailable
		title = "Service Unavailable"
	case http.StatusGatewayTimeout:
		errorType = ErrorTypeTimeout
		title = "Timeout"
	default:
		errorType = ErrorTypeInternalServerError
		title = "Internal Server Error"
	}

	return &A1Error{
		Type:      errorType,
		Title:     title,
		Status:    statusCode,
		Detail:    message,
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"http_status": statusCode,
		},
	}
}

func RecoverToA1Error(r interface{}, context string) *A1Error {
	var detail string
	if err, ok := r.(error); ok {
		detail = fmt.Sprintf("%s: %s", context, err.Error())
	} else {
		detail = fmt.Sprintf("%s: %v", context, r)
	}

	return &A1Error{
		Type:      ErrorTypeInternalServerError,
		Title:     "Internal Server Error",
		Status:    http.StatusInternalServerError,
		Detail:    detail,
		Timestamp: time.Now(),
		Extensions: map[string]interface{}{
			"panic_value": fmt.Sprintf("%v", r),
			"context":     context,
		},
	}
}

// Error response writing

func WriteA1Error(w http.ResponseWriter, err error) {
	var a1Err *A1Error

	if err == nil {
		a1Err = NewA1Error(ErrorTypeInternalServerError, "Unknown error occurred", http.StatusInternalServerError, nil)
	} else if a1Error, ok := err.(*A1Error); ok {
		a1Err = a1Error
	} else {
		a1Err = ConvertToA1Error(err, "unhandled error")
	}

	// Set headers
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("X-Error-ID", fmt.Sprintf("err_%d", time.Now().UnixNano()))
	w.WriteHeader(a1Err.Status)

	// Write JSON response
	if jsonData, jsonErr := json.Marshal(a1Err); jsonErr == nil {
		w.Write(jsonData)
	} else {
		// Fallback if JSON marshaling fails
		fallback := fmt.Sprintf(`{"type":"%s","title":"Error serialization failed","status":%d,"detail":"Failed to serialize error response"}`,
			ErrorTypeInternalServerError, http.StatusInternalServerError)
		w.Write([]byte(fallback))
	}
}

// Default error handler for middleware
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	WriteA1Error(w, err)
}

// Additional helper functions for error handling

func IsA1Error(err error) bool {
	_, ok := err.(*A1Error)
	return ok
}

func GetA1ErrorType(err error) A1ErrorType {
	if a1Err, ok := err.(*A1Error); ok {
		return a1Err.Type
	}
	return ErrorTypeInternalServerError
}

func GetA1ErrorStatus(err error) int {
	if a1Err, ok := err.(*A1Error); ok {
		return a1Err.Status
	}
	return http.StatusInternalServerError
}
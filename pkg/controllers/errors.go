// Package controllers implements Kubernetes controller logic for managing NetworkIntent
// and related custom resources. It provides the core reconciliation loops for the
// Nephoran Intent Operator in O-RAN/5G network orchestration.
package controllers

import (
	"fmt"
	"time"
)

// NetworkIntentError represents an error that occurred during NetworkIntent processing
type NetworkIntentError struct {
	Phase     string
	Operation string
	Cause     error
	Retries   int
	Timestamp time.Time
}

// Error implements the error interface
func (e *NetworkIntentError) Error() string {
	return fmt.Sprintf("NetworkIntent %s failed in %s: %v (retries: %d, time: %s)",
		e.Operation, e.Phase, e.Cause, e.Retries, e.Timestamp.Format(time.RFC3339))
}

// Unwrap returns the underlying cause
func (e *NetworkIntentError) Unwrap() error {
	return e.Cause
}

// IsRetryable determines if an error condition is retryable
func (e *NetworkIntentError) IsRetryable() bool {
	// Define retry logic based on error characteristics
	switch e.Phase {
	case "Processing":
		// LLM processing errors are usually retryable
		return true
	case "Deploying":
		// Git/deployment errors are usually retryable
		return true
	default:
		return false
	}
}

// NewNetworkIntentError creates a new NetworkIntentError
func NewNetworkIntentError(phase, operation string, cause error, retries int) *NetworkIntentError {
	return &NetworkIntentError{
		Phase:     phase,
		Operation: operation,
		Cause:     cause,
		Retries:   retries,
		Timestamp: time.Now(),
	}
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %s", e.Field, e.Value, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// DependencyError represents an error due to missing or invalid dependencies
type DependencyError struct {
	Dependency string
	Cause      error
}

// Error implements the error interface
func (e *DependencyError) Error() string {
	return fmt.Sprintf("dependency '%s' error: %v", e.Dependency, e.Cause)
}

// Unwrap returns the underlying cause
func (e *DependencyError) Unwrap() error {
	return e.Cause
}

// NewDependencyError creates a new DependencyError
func NewDependencyError(dependency string, cause error) *DependencyError {
	return &DependencyError{
		Dependency: dependency,
		Cause:      cause,
	}
}

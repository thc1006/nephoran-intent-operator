//go:build go1.24

package generics

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Validator represents a validation function for type T.

type Validator[T any] func(T) ValidationResult

// ValidationResult represents the result of a validation operation.

type ValidationResult struct {
	Valid bool

	Errors []ValidationError
}

// ValidationError represents a validation error.

type ValidationError struct {
	Field string

	Message string

	Code string

	Value any
}

// NewValidationResult creates a new validation result.

func NewValidationResult() ValidationResult {

	return ValidationResult{

		Valid: true,

		Errors: make([]ValidationError, 0),
	}

}

// AddError adds a validation error.

func (vr *ValidationResult) AddError(field, message, code string, value any) {

	vr.Valid = false

	vr.Errors = append(vr.Errors, ValidationError{

		Field: field,

		Message: message,

		Code: code,

		Value: value,
	})

}

// Combine combines multiple validation results.

func (vr *ValidationResult) Combine(other ValidationResult) {

	if !other.Valid {

		vr.Valid = false

		vr.Errors = append(vr.Errors, other.Errors...)

	}

}

// Error returns a string representation of all validation errors.

func (vr ValidationResult) Error() string {

	if vr.Valid {

		return ""

	}

	messages := make([]string, 0, len(vr.Errors))

	for _, err := range vr.Errors {

		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))

	}

	return strings.Join(messages, "; ")

}

// ValidationBuilder provides a fluent interface for building validators.

type ValidationBuilder[T any] struct {
	validators []Validator[T]
}

// NewValidationBuilder creates a new validation builder.

func NewValidationBuilder[T any]() *ValidationBuilder[T] {

	return &ValidationBuilder[T]{

		validators: make([]Validator[T], 0),
	}

}

// Add adds a validator to the builder.

func (vb *ValidationBuilder[T]) Add(validator Validator[T]) *ValidationBuilder[T] {

	vb.validators = append(vb.validators, validator)

	return vb

}

// Required adds a required field validator.

func (vb *ValidationBuilder[T]) Required(fieldName string, extractor func(T) any) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		if isZeroValue(value) {

			result := NewValidationResult()

			result.AddError(fieldName, "field is required", "required", value)

			return result

		}

		return NewValidationResult()

	}

	return vb.Add(validator)

}

// MinLength adds a minimum length validator for strings.

func (vb *ValidationBuilder[T]) MinLength(fieldName string, minLength int, extractor func(T) string) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if len(value) < minLength {

			result.AddError(fieldName, fmt.Sprintf("minimum length is %d", minLength), "min_length", value)

		}

		return result

	}

	return vb.Add(validator)

}

// MaxLength adds a maximum length validator for strings.

func (vb *ValidationBuilder[T]) MaxLength(fieldName string, maxLength int, extractor func(T) string) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if len(value) > maxLength {

			result.AddError(fieldName, fmt.Sprintf("maximum length is %d", maxLength), "max_length", value)

		}

		return result

	}

	return vb.Add(validator)

}

// Pattern adds a regex pattern validator for strings.

func (vb *ValidationBuilder[T]) Pattern(fieldName, pattern, message string, extractor func(T) string) *ValidationBuilder[T] {

	regex := regexp.MustCompile(pattern)

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if !regex.MatchString(value) {

			result.AddError(fieldName, message, "pattern", value)

		}

		return result

	}

	return vb.Add(validator)

}

// Email adds an email validation validator.

func (vb *ValidationBuilder[T]) Email(fieldName string, extractor func(T) string) *ValidationBuilder[T] {

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if value != "" && !emailRegex.MatchString(value) {

			result.AddError(fieldName, "invalid email format", "email", value)

		}

		return result

	}

	return vb.Add(validator)

}

// URL adds a URL validation validator.

func (vb *ValidationBuilder[T]) URL(fieldName string, extractor func(T) string) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if value != "" {

			if _, err := url.Parse(value); err != nil {

				result.AddError(fieldName, "invalid URL format", "url", value)

			}

		}

		return result

	}

	return vb.Add(validator)

}

// IP adds an IP address validation validator.

func (vb *ValidationBuilder[T]) IP(fieldName string, extractor func(T) string) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if value != "" && net.ParseIP(value) == nil {

			result.AddError(fieldName, "invalid IP address", "ip", value)

		}

		return result

	}

	return vb.Add(validator)

}

// Range adds a numeric range validator.

func (vb *ValidationBuilder[T]) Range(fieldName string, min, max int64, extractor func(T) int64) *ValidationBuilder[T] {

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if value < min || value > max {

			result.AddError(fieldName, fmt.Sprintf("value must be between %d and %d", min, max), "range", value)

		}

		return result

	}

	return vb.Add(validator)

}

// OneOf adds a validator that checks if value is one of the allowed values.

func (vb *ValidationBuilder[T]) OneOf(fieldName string, allowedValues []interface{}, extractor func(T) interface{}) *ValidationBuilder[T] {

	allowedSet := make(map[interface{}]bool)

	for _, v := range allowedValues {

		allowedSet[v] = true

	}

	validator := func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if !allowedSet[value] {

			result.AddError(fieldName, fmt.Sprintf("value must be one of: %v", allowedValues), "one_of", value)

		}

		return result

	}

	return vb.Add(validator)

}

// Custom adds a custom validator function.

func (vb *ValidationBuilder[T]) Custom(validator Validator[T]) *ValidationBuilder[T] {

	return vb.Add(validator)

}

// Conditional adds a conditional validator that only runs if condition is true.

func (vb *ValidationBuilder[T]) Conditional(condition func(T) bool, validator Validator[T]) *ValidationBuilder[T] {

	conditionalValidator := func(obj T) ValidationResult {

		if condition(obj) {

			return validator(obj)

		}

		return NewValidationResult()

	}

	return vb.Add(conditionalValidator)

}

// Build builds the final validator.

func (vb *ValidationBuilder[T]) Build() Validator[T] {

	return func(obj T) ValidationResult {

		result := NewValidationResult()

		for _, validator := range vb.validators {

			validationResult := validator(obj)

			result.Combine(validationResult)

		}

		return result

	}

}

// StructValidator provides validation for struct fields using reflection.

type StructValidator[T any] struct {
	fieldValidators map[string][]FieldValidator
}

// FieldValidator represents a validator for a specific field.

type FieldValidator struct {
	Name string

	Validator func(reflect.Value) ValidationResult
}

// NewStructValidator creates a new struct validator.

func NewStructValidator[T any]() *StructValidator[T] {

	return &StructValidator[T]{

		fieldValidators: make(map[string][]FieldValidator),
	}

}

// Field adds validators for a specific field.

func (sv *StructValidator[T]) Field(fieldName string, validators ...FieldValidator) *StructValidator[T] {

	sv.fieldValidators[fieldName] = append(sv.fieldValidators[fieldName], validators...)

	return sv

}

// Validate validates a struct using reflection.

func (sv *StructValidator[T]) Validate(obj T) ValidationResult {

	result := NewValidationResult()

	objValue := reflect.ValueOf(obj)

	objType := reflect.TypeOf(obj)

	// Handle pointer types.

	if objValue.Kind() == reflect.Ptr {

		objValue = objValue.Elem()

		objType = objType.Elem()

	}

	if objValue.Kind() != reflect.Struct {

		result.AddError("", "value must be a struct", "type_error", obj)

		return result

	}

	// Validate each field.

	for i := range objValue.NumField() {

		fieldType := objType.Field(i)

		fieldValue := objValue.Field(i)

		fieldName := fieldType.Name

		// Skip unexported fields.

		if !fieldValue.CanInterface() {

			continue

		}

		// Check for validation tag.

		if tag := fieldType.Tag.Get("validate"); tag != "" {

			tagResult := sv.validateTag(fieldName, fieldValue, tag)

			result.Combine(tagResult)

		}

		// Apply custom field validators.

		if validators, exists := sv.fieldValidators[fieldName]; exists {

			for _, validator := range validators {

				fieldResult := validator.Validator(fieldValue)

				result.Combine(fieldResult)

			}

		}

	}

	return result

}

// validateTag validates a field based on struct tags.

func (sv *StructValidator[T]) validateTag(fieldName string, fieldValue reflect.Value, tag string) ValidationResult {

	result := NewValidationResult()

	rules := strings.Split(tag, ",")

	for _, rule := range rules {

		rule = strings.TrimSpace(rule)

		parts := strings.Split(rule, "=")

		ruleName := parts[0]

		var ruleValue string

		if len(parts) > 1 {

			ruleValue = parts[1]

		}

		switch ruleName {

		case "required":

			if isZeroReflectValue(fieldValue) {

				result.AddError(fieldName, "field is required", "required", fieldValue.Interface())

			}

		case "min":

			if minVal, err := strconv.Atoi(ruleValue); err == nil {

				if !validateMinReflectValue(fieldValue, int64(minVal)) {

					result.AddError(fieldName, fmt.Sprintf("minimum value is %d", minVal), "min", fieldValue.Interface())

				}

			}

		case "max":

			if maxVal, err := strconv.Atoi(ruleValue); err == nil {

				if !validateMaxReflectValue(fieldValue, int64(maxVal)) {

					result.AddError(fieldName, fmt.Sprintf("maximum value is %d", maxVal), "max", fieldValue.Interface())

				}

			}

		case "email":

			if fieldValue.Kind() == reflect.String {

				emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

				if !emailRegex.MatchString(fieldValue.String()) {

					result.AddError(fieldName, "invalid email format", "email", fieldValue.Interface())

				}

			}

		case "url":

			if fieldValue.Kind() == reflect.String {

				if _, err := url.Parse(fieldValue.String()); err != nil {

					result.AddError(fieldName, "invalid URL format", "url", fieldValue.Interface())

				}

			}

		}

	}

	return result

}

// ValidationPipeline allows chaining multiple validators.

type ValidationPipeline[T any] struct {
	validators []Validator[T]

	stopOnFirstError bool
}

// NewValidationPipeline creates a new validation pipeline.

func NewValidationPipeline[T any]() *ValidationPipeline[T] {

	return &ValidationPipeline[T]{

		validators: make([]Validator[T], 0),
	}

}

// Add adds a validator to the pipeline.

func (vp *ValidationPipeline[T]) Add(validator Validator[T]) *ValidationPipeline[T] {

	vp.validators = append(vp.validators, validator)

	return vp

}

// StopOnFirstError configures the pipeline to stop on first error.

func (vp *ValidationPipeline[T]) StopOnFirstError(stop bool) *ValidationPipeline[T] {

	vp.stopOnFirstError = stop

	return vp

}

// Execute executes the validation pipeline.

func (vp *ValidationPipeline[T]) Execute(obj T) ValidationResult {

	result := NewValidationResult()

	for _, validator := range vp.validators {

		validationResult := validator(obj)

		result.Combine(validationResult)

		if vp.stopOnFirstError && !validationResult.Valid {

			break

		}

	}

	return result

}

// AsyncValidator provides asynchronous validation capabilities.

type AsyncValidator[T any] struct {
	validator Validator[T]
}

// NewAsyncValidator creates a new async validator.

func NewAsyncValidator[T any](validator Validator[T]) *AsyncValidator[T] {

	return &AsyncValidator[T]{

		validator: validator,
	}

}

// ValidateAsync validates asynchronously.

func (av *AsyncValidator[T]) ValidateAsync(ctx context.Context, obj T) <-chan ValidationResult {

	resultChan := make(chan ValidationResult, 1)

	go func() {

		defer close(resultChan)

		result := av.validator(obj)

		select {

		case resultChan <- result:

		case <-ctx.Done():

			// Context cancelled.

		}

	}()

	return resultChan

}

// BatchValidator validates multiple objects concurrently.

type BatchValidator[T any] struct {
	validator Validator[T]

	maxConcurrency int
}

// NewBatchValidator creates a new batch validator.

func NewBatchValidator[T any](validator Validator[T], maxConcurrency int) *BatchValidator[T] {

	if maxConcurrency <= 0 {

		maxConcurrency = 10

	}

	return &BatchValidator[T]{

		validator: validator,

		maxConcurrency: maxConcurrency,
	}

}

// ValidateBatch validates multiple objects concurrently.

func (bv *BatchValidator[T]) ValidateBatch(ctx context.Context, objects []T) <-chan BatchValidationResult[T] {

	resultChan := make(chan BatchValidationResult[T], len(objects))

	// Create semaphore for concurrency control.

	semaphore := make(chan struct{}, bv.maxConcurrency)

	go func() {

		defer close(resultChan)

		var wg sync.WaitGroup

		for i, obj := range objects {

			wg.Add(1)

			go func(index int, object T) {

				defer wg.Done()

				// Acquire semaphore.

				semaphore <- struct{}{}

				defer func() { <-semaphore }()

				result := bv.validator(object)

				batchResult := BatchValidationResult[T]{

					Index: index,

					Object: object,

					Result: result,
				}

				select {

				case resultChan <- batchResult:

				case <-ctx.Done():

					return

				}

			}(i, obj)

		}

		wg.Wait()

	}()

	return resultChan

}

// BatchValidationResult represents the result of a batch validation.

type BatchValidationResult[T any] struct {
	Index int

	Object T

	Result ValidationResult
}

// ConditionalValidator applies different validators based on conditions.

type ConditionalValidator[T any] struct {
	conditions []ConditionalRule[T]

	defaultValidator Validator[T]
}

// ConditionalRule represents a conditional validation rule.

type ConditionalRule[T any] struct {
	Condition func(T) bool

	Validator Validator[T]
}

// NewConditionalValidator creates a new conditional validator.

func NewConditionalValidator[T any]() *ConditionalValidator[T] {

	return &ConditionalValidator[T]{

		conditions: make([]ConditionalRule[T], 0),
	}

}

// When adds a conditional rule.

func (cv *ConditionalValidator[T]) When(condition func(T) bool, validator Validator[T]) *ConditionalValidator[T] {

	cv.conditions = append(cv.conditions, ConditionalRule[T]{

		Condition: condition,

		Validator: validator,
	})

	return cv

}

// Default sets the default validator when no conditions match.

func (cv *ConditionalValidator[T]) Default(validator Validator[T]) *ConditionalValidator[T] {

	cv.defaultValidator = validator

	return cv

}

// Validate validates using the first matching condition.

func (cv *ConditionalValidator[T]) Validate(obj T) ValidationResult {

	for _, rule := range cv.conditions {

		if rule.Condition(obj) {

			return rule.Validator(obj)

		}

	}

	if cv.defaultValidator != nil {

		return cv.defaultValidator(obj)

	}

	return NewValidationResult()

}

// Utility functions.

// isZeroValue checks if a value is the zero value.

func isZeroValue(value any) bool {

	if value == nil {

		return true

	}

	v := reflect.ValueOf(value)

	return v.IsZero()

}

// isZeroReflectValue checks if a reflect.Value is zero.

func isZeroReflectValue(value reflect.Value) bool {

	return value.IsZero()

}

// validateMinReflectValue validates minimum value for reflect.Value.

func validateMinReflectValue(value reflect.Value, min int64) bool {

	switch value.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return value.Int() >= min

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

		return int64(value.Uint()) >= min

	case reflect.Float32, reflect.Float64:

		return int64(value.Float()) >= min

	case reflect.String:

		return int64(len(value.String())) >= min

	case reflect.Slice, reflect.Array:

		return int64(value.Len()) >= min

	}

	return true

}

// validateMaxReflectValue validates maximum value for reflect.Value.

func validateMaxReflectValue(value reflect.Value, max int64) bool {

	switch value.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return value.Int() <= max

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:

		return int64(value.Uint()) <= max

	case reflect.Float32, reflect.Float64:

		return int64(value.Float()) <= max

	case reflect.String:

		return int64(len(value.String())) <= max

	case reflect.Slice, reflect.Array:

		return int64(value.Len()) <= max

	}

	return true

}

// Predefined validators.

// NotEmpty validates that a string is not empty.

func NotEmpty[T any](fieldName string, extractor func(T) string) Validator[T] {

	return func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if strings.TrimSpace(value) == "" {

			result.AddError(fieldName, "field cannot be empty", "not_empty", value)

		}

		return result

	}

}

// PositiveNumber validates that a number is positive.

func PositiveNumber[T any](fieldName string, extractor func(T) int64) Validator[T] {

	return func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if value <= 0 {

			result.AddError(fieldName, "value must be positive", "positive", value)

		}

		return result

	}

}

// InFuture validates that a time is in the future.

func InFuture[T any](fieldName string, extractor func(T) time.Time) Validator[T] {

	return func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if !value.After(time.Now()) {

			result.AddError(fieldName, "time must be in the future", "future", value)

		}

		return result

	}

}

// InPast validates that a time is in the past.

func InPast[T any](fieldName string, extractor func(T) time.Time) Validator[T] {

	return func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		if !value.Before(time.Now()) {

			result.AddError(fieldName, "time must be in the past", "past", value)

		}

		return result

	}

}

// ValidateUniqueSlice validates that slice elements are unique.

func ValidateUniqueSlice[T any, E Comparable](fieldName string, extractor func(T) []E) Validator[T] {

	return func(obj T) ValidationResult {

		values := extractor(obj)

		result := NewValidationResult()

		seen := NewSet[E]()

		for _, value := range values {

			if seen.Contains(value) {

				result.AddError(fieldName, "duplicate values are not allowed", "unique", values)

				break

			}

			seen.Add(value)

		}

		return result

	}

}

// ValidJSON validates that a string contains valid JSON.

func ValidJSON[T any](fieldName string, extractor func(T) string) Validator[T] {

	return func(obj T) ValidationResult {

		value := extractor(obj)

		result := NewValidationResult()

		var js json.RawMessage

		if err := json.Unmarshal([]byte(value), &js); err != nil {

			result.AddError(fieldName, "invalid JSON format", "json", value)

		}

		return result

	}

}

// CrossFieldValidator provides cross-field validation capabilities.

type CrossFieldValidator[T any] struct {
	validator func(T) ValidationResult
}

// NewCrossFieldValidator creates a new cross-field validator.

func NewCrossFieldValidator[T any](validator func(T) ValidationResult) *CrossFieldValidator[T] {

	return &CrossFieldValidator[T]{

		validator: validator,
	}

}

// Validate validates using cross-field logic.

func (cfv *CrossFieldValidator[T]) Validate(obj T) ValidationResult {

	return cfv.validator(obj)

}

// PasswordStrength validates password strength.

func PasswordStrength[T any](fieldName string, extractor func(T) string) Validator[T] {

	return func(obj T) ValidationResult {

		password := extractor(obj)

		result := NewValidationResult()

		if len(password) < 8 {

			result.AddError(fieldName, "password must be at least 8 characters long", "password_length", password)

		}

		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)

		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)

		hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

		if !hasUpper {

			result.AddError(fieldName, "password must contain at least one uppercase letter", "password_uppercase", password)

		}

		if !hasLower {

			result.AddError(fieldName, "password must contain at least one lowercase letter", "password_lowercase", password)

		}

		if !hasDigit {

			result.AddError(fieldName, "password must contain at least one digit", "password_digit", password)

		}

		if !hasSpecial {

			result.AddError(fieldName, "password must contain at least one special character", "password_special", password)

		}

		return result

	}

}

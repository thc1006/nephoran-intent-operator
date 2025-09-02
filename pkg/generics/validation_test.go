//go:build go1.24

package generics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutil"
)

// Test data structures
type TestUser struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"min=0,max=120"`
	Password string
}

func TestValidationBuilder_Required(t *testing.T) {
	ctx, cancel := testutil.ContextWithTimeout(t)
	defer cancel()

	builder := NewValidationBuilder[TestUser]()
	require.NotNil(t, builder, "validation builder should not be nil")

	validator := builder.
		Required("name", func(u TestUser) any { return u.Name }).
		Build()
	require.NotNil(t, validator, "validator should not be nil")

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "valid user with name",
			user:  TestUser{Name: "John Doe"},
			valid: true,
		},
		{
			name:  "invalid user without name",
			user:  TestUser{Name: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			select {
			case <-ctx.Done():
				t.Fatal("test timeout")
			default:
				result := validator(tt.user)
				require.NotNil(t, result, "validation result should not be nil")

				if result.Valid != tt.valid {
					t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
				}

				if !tt.valid && len(result.Errors) == 0 {
					t.Error("Expected validation errors for invalid case")
				}
			}
		})
	}
}

func TestValidationBuilder_MinMaxLength(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		MinLength("name", 2, func(u TestUser) string { return u.Name }).
		MaxLength("name", 50, func(u TestUser) string { return u.Name }).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "valid name length",
			user:  TestUser{Name: "John Doe"},
			valid: true,
		},
		{
			name:  "name too short",
			user:  TestUser{Name: "J"},
			valid: false,
		},
		{
			name:  "name too long",
			user:  TestUser{Name: strings.Repeat("a", 51)},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestValidationBuilder_Pattern(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		Pattern("name", `^[A-Za-z\s]+$`, "name must contain only letters and spaces",
			func(u TestUser) string { return u.Name }).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "valid name with letters and spaces",
			user:  TestUser{Name: "John Doe"},
			valid: true,
		},
		{
			name:  "invalid name with numbers",
			user:  TestUser{Name: "John123"},
			valid: false,
		},
		{
			name:  "invalid name with special characters",
			user:  TestUser{Name: "John@Doe"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestValidationBuilder_Email(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		Email("email", func(u TestUser) string { return u.Email }).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "valid email",
			user:  TestUser{Email: "john.doe@example.com"},
			valid: true,
		},
		{
			name:  "valid email with subdomain",
			user:  TestUser{Email: "user@mail.company.com"},
			valid: true,
		},
		{
			name:  "invalid email without @",
			user:  TestUser{Email: "johndoe.example.com"},
			valid: false,
		},
		{
			name:  "invalid email without domain",
			user:  TestUser{Email: "john@"},
			valid: false,
		},
		{
			name:  "empty email (should be valid for email validator)",
			user:  TestUser{Email: ""},
			valid: true, // Email validator allows empty strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v. Errors: %v", tt.valid, result.Valid, result.Errors)
			}
		})
	}
}

func TestValidationBuilder_Range(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		Range("age", 0, 120, func(u TestUser) int64 { return int64(u.Age) }).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "valid age",
			user:  TestUser{Age: 25},
			valid: true,
		},
		{
			name:  "minimum valid age",
			user:  TestUser{Age: 0},
			valid: true,
		},
		{
			name:  "maximum valid age",
			user:  TestUser{Age: 120},
			valid: true,
		},
		{
			name:  "age too low",
			user:  TestUser{Age: -1},
			valid: false,
		},
		{
			name:  "age too high",
			user:  TestUser{Age: 121},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestValidationBuilder_OneOf(t *testing.T) {
	type Status struct {
		Value string
	}

	builder := NewValidationBuilder[Status]()
	validator := builder.
		OneOf("status", []interface{}{"active", "inactive", "pending"},
			func(s Status) interface{} { return s.Value }).
		Build()

	tests := []struct {
		name   string
		status Status
		valid  bool
	}{
		{
			name:   "valid status",
			status: Status{Value: "active"},
			valid:  true,
		},
		{
			name:   "another valid status",
			status: Status{Value: "pending"},
			valid:  true,
		},
		{
			name:   "invalid status",
			status: Status{Value: "unknown"},
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.status)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestValidationBuilder_Custom(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		Custom(func(u TestUser) ValidationResult {
			result := NewValidationResult()
			if u.Name == "admin" && u.Age < 18 {
				result.AddError("user", "admin must be at least 18 years old", "admin_age", u)
			}
			return result
		}).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "adult admin",
			user:  TestUser{Name: "admin", Age: 25},
			valid: true,
		},
		{
			name:  "non-admin minor",
			user:  TestUser{Name: "user", Age: 16},
			valid: true,
		},
		{
			name:  "minor admin (invalid)",
			user:  TestUser{Name: "admin", Age: 16},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestValidationBuilder_Conditional(t *testing.T) {
	builder := NewValidationBuilder[TestUser]()
	validator := builder.
		Conditional(
			func(u TestUser) bool { return u.Age >= 18 },
			func(u TestUser) ValidationResult {
				result := NewValidationResult()
				if len(u.Password) < 8 {
					result.AddError("password", "adults must have password with at least 8 characters", "adult_password", u.Password)
				}
				return result
			}).
		Build()

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "adult with strong password",
			user:  TestUser{Age: 25, Password: "strongpassword"},
			valid: true,
		},
		{
			name:  "minor with weak password",
			user:  TestUser{Age: 16, Password: "weak"},
			valid: true, // Condition doesn't apply
		},
		{
			name:  "adult with weak password",
			user:  TestUser{Age: 25, Password: "weak"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestStructValidator(t *testing.T) {
	validator := NewStructValidator[TestUser]()

	user := TestUser{
		Name:     "",        // Should fail required
		Email:    "invalid", // Should fail email
		Age:      150,       // Should fail max
		Password: "test",
	}

	result := validator.Validate(user)

	if result.Valid {
		t.Error("Expected validation to fail")
	}

	// Check that we have multiple errors
	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}

	// Verify specific error types exist
	errorCodes := make(map[string]bool)
	for _, err := range result.Errors {
		errorCodes[err.Code] = true
	}

	expectedCodes := []string{"required", "email", "max"}
	for _, code := range expectedCodes {
		if !errorCodes[code] {
			t.Errorf("Expected error code '%s' not found", code)
		}
	}
}

func TestValidationPipeline(t *testing.T) {
	validator1 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Name == "" {
			result.AddError("name", "name is required", "required", u.Name)
		}
		return result
	}

	validator2 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Age < 0 {
			result.AddError("age", "age must be positive", "positive", u.Age)
		}
		return result
	}

	pipeline := NewValidationPipeline[TestUser]().
		Add(validator1).
		Add(validator2)

	tests := []struct {
		name       string
		user       TestUser
		valid      bool
		errorCount int
	}{
		{
			name:       "valid user",
			user:       TestUser{Name: "John", Age: 25},
			valid:      true,
			errorCount: 0,
		},
		{
			name:       "invalid user - missing name",
			user:       TestUser{Name: "", Age: 25},
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "invalid user - negative age",
			user:       TestUser{Name: "John", Age: -5},
			valid:      false,
			errorCount: 1,
		},
		{
			name:       "invalid user - multiple errors",
			user:       TestUser{Name: "", Age: -5},
			valid:      false,
			errorCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pipeline.Execute(tt.user)

			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}

			if len(result.Errors) != tt.errorCount {
				t.Errorf("Expected %d errors, got %d", tt.errorCount, len(result.Errors))
			}
		})
	}
}

func TestValidationPipeline_StopOnFirstError(t *testing.T) {
	validator1 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Name == "" {
			result.AddError("name", "name is required", "required", u.Name)
		}
		return result
	}

	validator2 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Age < 0 {
			result.AddError("age", "age must be positive", "positive", u.Age)
		}
		return result
	}

	pipeline := NewValidationPipeline[TestUser]().
		Add(validator1).
		Add(validator2).
		StopOnFirstError(true)

	user := TestUser{Name: "", Age: -5} // Both validators should fail
	result := pipeline.Execute(user)

	if result.Valid {
		t.Error("Expected validation to fail")
	}

	// Should only have 1 error because we stop on first error
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error (stop on first), got %d", len(result.Errors))
	}

	// Should be the name error
	if result.Errors[0].Field != "name" {
		t.Errorf("Expected first error to be 'name', got '%s'", result.Errors[0].Field)
	}
}

func TestAsyncValidator(t *testing.T) {
	validator := func(u TestUser) ValidationResult {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		result := NewValidationResult()
		if u.Name == "" {
			result.AddError("name", "name is required", "required", u.Name)
		}
		return result
	}

	asyncValidator := NewAsyncValidator(validator)

	user := TestUser{Name: ""}
	ctx := context.Background()

	resultChan := asyncValidator.ValidateAsync(ctx, user)

	select {
	case result := <-resultChan:
		if result.Valid {
			t.Error("Expected validation to fail")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
	case <-time.After(1 * time.Second):
		t.Error("Async validation timed out")
	}
}

func TestBatchValidator(t *testing.T) {
	validator := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Name == "" {
			result.AddError("name", "name is required", "required", u.Name)
		}
		return result
	}

	batchValidator := NewBatchValidator(validator, 2)

	users := []TestUser{
		{Name: "John"},
		{Name: ""},
		{Name: "Jane"},
		{Name: ""},
	}

	ctx := context.Background()
	resultChan := batchValidator.ValidateBatch(ctx, users)

	results := make([]BatchValidationResult[TestUser], 0, len(users))
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != len(users) {
		t.Errorf("Expected %d results, got %d", len(users), len(results))
	}

	// Count valid and invalid results
	validCount := 0
	invalidCount := 0

	for _, result := range results {
		if result.Result.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}

	if validCount != 2 {
		t.Errorf("Expected 2 valid results, got %d", validCount)
	}

	if invalidCount != 2 {
		t.Errorf("Expected 2 invalid results, got %d", invalidCount)
	}
}

func TestConditionalValidator(t *testing.T) {
	validator := NewConditionalValidator[TestUser]().
		When(
			func(u TestUser) bool { return u.Age >= 18 },
			func(u TestUser) ValidationResult {
				result := NewValidationResult()
				if len(u.Password) < 8 {
					result.AddError("password", "adult password must be at least 8 characters", "password_length", u.Password)
				}
				return result
			}).
		When(
			func(u TestUser) bool { return u.Age < 18 },
			func(u TestUser) ValidationResult {
				result := NewValidationResult()
				if u.Password != "" {
					result.AddError("password", "minors should not have passwords", "minor_password", u.Password)
				}
				return result
			}).
		Default(func(u TestUser) ValidationResult {
			return NewValidationResult() // Always valid for default case
		})

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "adult with strong password",
			user:  TestUser{Age: 25, Password: "strongpassword"},
			valid: true,
		},
		{
			name:  "adult with weak password",
			user:  TestUser{Age: 25, Password: "weak"},
			valid: false,
		},
		{
			name:  "minor without password",
			user:  TestUser{Age: 16, Password: ""},
			valid: true,
		},
		{
			name:  "minor with password",
			user:  TestUser{Age: 16, Password: "password"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v. Errors: %v", tt.valid, result.Valid, result.Errors)
			}
		})
	}
}

// Test predefined validators

func TestNotEmpty(t *testing.T) {
	validator := NotEmpty("name", func(u TestUser) string { return u.Name })

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "non-empty name",
			user:  TestUser{Name: "John"},
			valid: true,
		},
		{
			name:  "empty name",
			user:  TestUser{Name: ""},
			valid: false,
		},
		{
			name:  "whitespace name",
			user:  TestUser{Name: "   "},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestPositiveNumber(t *testing.T) {
	validator := PositiveNumber("age", func(u TestUser) int64 { return int64(u.Age) })

	tests := []struct {
		name  string
		user  TestUser
		valid bool
	}{
		{
			name:  "positive age",
			user:  TestUser{Age: 25},
			valid: true,
		},
		{
			name:  "zero age",
			user:  TestUser{Age: 0},
			valid: false,
		},
		{
			name:  "negative age",
			user:  TestUser{Age: -5},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestUniqueSliceValidator(t *testing.T) {
	type ListData struct {
		Items []int
	}

	validator := ValidateUniqueSlice("items", func(d ListData) []int { return d.Items })

	tests := []struct {
		name  string
		data  ListData
		valid bool
	}{
		{
			name:  "unique items",
			data:  ListData{Items: []int{1, 2, 3, 4}},
			valid: true,
		},
		{
			name:  "duplicate items",
			data:  ListData{Items: []int{1, 2, 2, 3}},
			valid: false,
		},
		{
			name:  "empty slice",
			data:  ListData{Items: []int{}},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.data)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	validator := PasswordStrength("password", func(u TestUser) string { return u.Password })

	tests := []struct {
		name      string
		user      TestUser
		valid     bool
		minErrors int
	}{
		{
			name:      "strong password",
			user:      TestUser{Password: "StrongPass123!"},
			valid:     true,
			minErrors: 0,
		},
		{
			name:      "weak password - too short",
			user:      TestUser{Password: "Weak1!"},
			valid:     false,
			minErrors: 1,
		},
		{
			name:      "weak password - no uppercase",
			user:      TestUser{Password: "weakpass123!"},
			valid:     false,
			minErrors: 1,
		},
		{
			name:      "weak password - no lowercase",
			user:      TestUser{Password: "WEAKPASS123!"},
			valid:     false,
			minErrors: 1,
		},
		{
			name:      "weak password - no digit",
			user:      TestUser{Password: "WeakPass!"},
			valid:     false,
			minErrors: 1,
		},
		{
			name:      "weak password - no special char",
			user:      TestUser{Password: "WeakPass123"},
			valid:     false,
			minErrors: 1,
		},
		{
			name:      "very weak password",
			user:      TestUser{Password: "weak"},
			valid:     false,
			minErrors: 4, // All requirements fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator(tt.user)
			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, result.Valid)
			}

			if len(result.Errors) < tt.minErrors {
				t.Errorf("Expected at least %d errors, got %d", tt.minErrors, len(result.Errors))
			}
		})
	}
}

// Benchmarks

func BenchmarkValidationBuilder_Simple(b *testing.B) {
	validator := NewValidationBuilder[TestUser]().
		Required("name", func(u TestUser) any { return u.Name }).
		Email("email", func(u TestUser) string { return u.Email }).
		Build()

	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator(user)
	}
}

func BenchmarkStructValidator(b *testing.B) {
	validator := NewStructValidator[TestUser]()
	user := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(user)
	}
}

func BenchmarkValidationPipeline(b *testing.B) {
	validator1 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Name == "" {
			result.AddError("name", "required", "required", u.Name)
		}
		return result
	}

	validator2 := func(u TestUser) ValidationResult {
		result := NewValidationResult()
		if u.Age < 0 {
			result.AddError("age", "positive", "positive", u.Age)
		}
		return result
	}

	pipeline := NewValidationPipeline[TestUser]().
		Add(validator1).
		Add(validator2)

	user := TestUser{Name: "John", Age: 25}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pipeline.Execute(user)
	}
}

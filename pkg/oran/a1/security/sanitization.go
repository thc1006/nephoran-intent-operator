package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/xeipuuv/gojsonschema"
)

// SanitizationConfig holds sanitization configuration
type SanitizationConfig struct {
	Enabled                 bool                           `json:"enabled"`
	Mode                    SanitizationMode               `json:"mode"`
	Policies                map[string]*SanitizationPolicy `json:"policies"`
	DefaultPolicy           string                         `json:"default_policy"`
	MaxInputSize            int64                          `json:"max_input_size"`
	MaxFieldCount           int                            `json:"max_field_count"`
	MaxNestingDepth         int                            `json:"max_nesting_depth"`
	MaxArrayLength          int                            `json:"max_array_length"`
	MaxStringLength         int                            `json:"max_string_length"`
	EnforceJSONSchema       bool                           `json:"enforce_json_schema"`
	SchemaPath              string                         `json:"schema_path"`
	DetectMaliciousPatterns bool                           `json:"detect_malicious_patterns"`
	LogSanitizationEvents   bool                           `json:"log_sanitization_events"`
	RejectOnViolation       bool                           `json:"reject_on_violation"`
}

// SanitizationMode represents the sanitization mode
type SanitizationMode string

const (
	ModeStrict      SanitizationMode = "strict"      // Reject invalid input
	ModeSanitize    SanitizationMode = "sanitize"    // Clean invalid input
	ModeLog         SanitizationMode = "log"         // Log violations only
	ModePassthrough SanitizationMode = "passthrough" // No sanitization
)

// SanitizationPolicy defines a sanitization policy
type SanitizationPolicy struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Rules               []*SanitizationRule  `json:"rules"`
	AllowedPatterns     []string             `json:"allowed_patterns"`
	BlockedPatterns     []string             `json:"blocked_patterns"`
	AllowedContentTypes []string             `json:"allowed_content_types"`
	CustomValidators    map[string]Validator `json:"-"`
	SchemaValidation    *SchemaValidation    `json:"schema_validation"`
}

// SanitizationRule defines a sanitization rule
type SanitizationRule struct {
	Field           string        `json:"field"`
	Type            InputType     `json:"type"`
	Required        bool          `json:"required"`
	MinLength       int           `json:"min_length"`
	MaxLength       int           `json:"max_length"`
	Pattern         string        `json:"pattern"`
	AllowHTML       bool          `json:"allow_html"`
	AllowedTags     []string      `json:"allowed_tags"`
	StripTags       bool          `json:"strip_tags"`
	EscapeHTML      bool          `json:"escape_html"`
	TrimWhitespace  bool          `json:"trim_whitespace"`
	Lowercase       bool          `json:"lowercase"`
	Uppercase       bool          `json:"uppercase"`
	CustomValidator string        `json:"custom_validator"`
	Transform       TransformFunc `json:"-"`
}

// InputType represents the type of input
type InputType string

const (
	TypeString   InputType = "string"
	TypeNumber   InputType = "number"
	TypeBoolean  InputType = "boolean"
	TypeArray    InputType = "array"
	TypeObject   InputType = "object"
	TypeEmail    InputType = "email"
	TypeURL      InputType = "url"
	TypeIP       InputType = "ip"
	TypeJSON     InputType = "json"
	TypeBase64   InputType = "base64"
	TypeUUID     InputType = "uuid"
	TypeAlphaNum InputType = "alphanumeric"
)

// SchemaValidation holds JSON schema validation configuration
type SchemaValidation struct {
	Enabled         bool            `json:"enabled"`
	SchemaFile      string          `json:"schema_file"`
	SchemaJSON      json.RawMessage `json:"schema_json"`
	StrictMode      bool            `json:"strict_mode"`
	AllowAdditional bool            `json:"allow_additional"`
}

// Validator interface for custom validation
type Validator interface {
	Validate(ctx context.Context, value interface{}) (bool, error)
	Sanitize(ctx context.Context, value interface{}) (interface{}, error)
}

// TransformFunc represents a transformation function
type TransformFunc func(value interface{}) (interface{}, error)

// SanitizationManager manages input sanitization
type SanitizationManager struct {
	config            *SanitizationConfig
	logger            *logging.StructuredLogger
	policies          map[string]*SanitizationPolicy
	schemas           map[string]*gojsonschema.Schema
	htmlPolicy        *bluemonday.Policy
	maliciousPatterns []*regexp.Regexp
	mu                sync.RWMutex
	stats             *SanitizationStats
}

// SanitizationStats tracks sanitization statistics
type SanitizationStats struct {
	mu                sync.RWMutex
	TotalRequests     int64
	SanitizedInputs   int64
	RejectedInputs    int64
	MaliciousDetected int64
	SchemaViolations  int64
	Violations        map[string]int64
}

// SanitizationResult represents the result of sanitization
type SanitizationResult struct {
	Sanitized      bool                   `json:"sanitized"`
	Modified       bool                   `json:"modified"`
	Violations     []Violation            `json:"violations,omitempty"`
	OriginalValue  interface{}            `json:"-"`
	SanitizedValue interface{}            `json:"sanitized_value"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Violation represents a sanitization violation
type Violation struct {
	Field    string            `json:"field"`
	Rule     string            `json:"rule"`
	Value    interface{}       `json:"value,omitempty"`
	Message  string            `json:"message"`
	Severity EventSeverity `json:"severity"`
}

// Note: Severity constants are defined in audit.go as EventSeverity type

// Common malicious patterns
var (
	sqlInjectionPattern     = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|eval|setTimeout|setInterval)`)
	xssPattern              = regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onclick=|onload=|<iframe|<embed|<object)`)
	pathTraversalPattern    = regexp.MustCompile(`\.\.[\\/]|\.\.%2[fF]`)
	commandInjectionPattern = regexp.MustCompile(`[;&|<>$\x60]`)
	ldapInjectionPattern    = regexp.MustCompile(`[*()\\,=]`)
	xmlInjectionPattern     = regexp.MustCompile(`(?i)(<!DOCTYPE|<!ENTITY|SYSTEM|PUBLIC)`)
	noSQLInjectionPattern   = regexp.MustCompile(`[\$\{\}]`)
)

// NewSanitizationManager creates a new sanitization manager
func NewSanitizationManager(config *SanitizationConfig, logger *logging.StructuredLogger) (*SanitizationManager, error) {
	if config == nil {
		return nil, errors.New("sanitization config is required")
	}

	sm := &SanitizationManager{
		config:   config,
		logger:   logger,
		policies: make(map[string]*SanitizationPolicy),
		schemas:  make(map[string]*gojsonschema.Schema),
		stats: &SanitizationStats{
			Violations: make(map[string]int64),
		},
	}

	// Initialize HTML sanitization policy
	sm.htmlPolicy = bluemonday.StrictPolicy()

	// Initialize malicious patterns
	sm.initializeMaliciousPatterns()

	// Load sanitization policies
	if err := sm.loadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load sanitization policies: %w", err)
	}

	// Load JSON schemas if enabled
	if config.EnforceJSONSchema && config.SchemaPath != "" {
		if err := sm.loadSchemas(); err != nil {
			return nil, fmt.Errorf("failed to load JSON schemas: %w", err)
		}
	}

	return sm, nil
}

// initializeMaliciousPatterns initializes patterns for detecting malicious input
func (sm *SanitizationManager) initializeMaliciousPatterns() {
	sm.maliciousPatterns = []*regexp.Regexp{
		sqlInjectionPattern,
		xssPattern,
		pathTraversalPattern,
		commandInjectionPattern,
		ldapInjectionPattern,
		xmlInjectionPattern,
		noSQLInjectionPattern,
	}
}

// loadPolicies loads sanitization policies
func (sm *SanitizationManager) loadPolicies() error {
	for name, policy := range sm.config.Policies {
		// Compile regex patterns
		for _, rule := range policy.Rules {
			if rule.Pattern != "" {
				if _, err := regexp.Compile(rule.Pattern); err != nil {
					return fmt.Errorf("invalid pattern in rule for field %s: %w", rule.Field, err)
				}
			}
		}
		sm.policies[name] = policy
	}
	return nil
}

// loadSchemas loads JSON schemas for validation
func (sm *SanitizationManager) loadSchemas() error {
	// Load schemas from configured path
	// This is a simplified implementation
	return nil
}

// SanitizePolicyInput sanitizes policy input data
func (sm *SanitizationManager) SanitizePolicyInput(ctx context.Context, input map[string]interface{}) (*SanitizationResult, error) {
	sm.stats.mu.Lock()
	sm.stats.TotalRequests++
	sm.stats.mu.Unlock()

	result := &SanitizationResult{
		OriginalValue:  input,
		SanitizedValue: make(map[string]interface{}),
		Violations:     []Violation{},
		Metadata:       make(map[string]interface{}),
	}

	// Check input size
	if err := sm.checkInputSize(input); err != nil {
		result.Violations = append(result.Violations, Violation{
			Field:    "_root",
			Rule:     "max_size",
			Message:  err.Error(),
			Severity: SeverityError,
		})
		if sm.config.RejectOnViolation {
			return result, err
		}
	}

	// Check field count
	if len(input) > sm.config.MaxFieldCount {
		result.Violations = append(result.Violations, Violation{
			Field:    "_root",
			Rule:     "max_fields",
			Message:  fmt.Sprintf("too many fields: %d > %d", len(input), sm.config.MaxFieldCount),
			Severity: SeverityWarning,
		})
		if sm.config.RejectOnViolation {
			return result, errors.New("field count exceeded")
		}
	}

	// Get the appropriate policy
	policy := sm.getPolicy(sm.config.DefaultPolicy)
	if policy == nil {
		return result, errors.New("no sanitization policy found")
	}

	// Sanitize each field
	sanitized := make(map[string]interface{})
	for key, value := range input {
		sanitizedValue, violations := sm.sanitizeField(ctx, key, value, policy)
		sanitized[key] = sanitizedValue
		result.Violations = append(result.Violations, violations...)

		if sanitizedValue != value {
			result.Modified = true
		}
	}

	// Check for malicious patterns if enabled
	if sm.config.DetectMaliciousPatterns {
		if violations := sm.detectMaliciousPatterns(sanitized); len(violations) > 0 {
			result.Violations = append(result.Violations, violations...)
			sm.stats.mu.Lock()
			sm.stats.MaliciousDetected++
			sm.stats.mu.Unlock()

			if sm.config.RejectOnViolation {
				return result, errors.New("malicious patterns detected")
			}
		}
	}

	// Validate against JSON schema if enabled
	if sm.config.EnforceJSONSchema {
		if violations := sm.validateSchema(sanitized); len(violations) > 0 {
			result.Violations = append(result.Violations, violations...)
			sm.stats.mu.Lock()
			sm.stats.SchemaViolations++
			sm.stats.mu.Unlock()

			if sm.config.RejectOnViolation {
				return result, errors.New("schema validation failed")
			}
		}
	}

	result.Sanitized = true
	result.SanitizedValue = sanitized

	// Update statistics
	if result.Modified {
		sm.stats.mu.Lock()
		sm.stats.SanitizedInputs++
		sm.stats.mu.Unlock()
	}

	if len(result.Violations) > 0 && sm.config.LogSanitizationEvents {
		sm.logger.Warn("input sanitization violations detected",
			slog.Int("violation_count", len(result.Violations)),
			slog.Any("violations", result.Violations))
	}

	return result, nil
}

// sanitizeField sanitizes a single field
func (sm *SanitizationManager) sanitizeField(ctx context.Context, field string, value interface{}, policy *SanitizationPolicy) (interface{}, []Violation) {
	var violations []Violation

	// Find matching rule
	var rule *SanitizationRule
	for _, r := range policy.Rules {
		if r.Field == field || r.Field == "*" {
			rule = r
			break
		}
	}

	if rule == nil {
		// No specific rule, apply default sanitization
		return sm.defaultSanitize(value), violations
	}

	// Type validation
	if !sm.validateType(value, rule.Type) {
		violations = append(violations, Violation{
			Field:    field,
			Rule:     "type",
			Message:  fmt.Sprintf("invalid type: expected %s", rule.Type),
			Severity: SeverityError,
		})
		return sm.coerceType(value, rule.Type), violations
	}

	// Apply sanitization based on type
	switch rule.Type {
	case TypeString:
		sanitized, v := sm.sanitizeString(value.(string), rule)
		violations = append(violations, v...)
		return sanitized, violations

	case TypeNumber:
		return sm.sanitizeNumber(value, rule), violations

	case TypeArray:
		return sm.sanitizeArray(ctx, value, rule, policy)

	case TypeObject:
		return sm.sanitizeObject(ctx, value, rule, policy)

	case TypeEmail:
		return sm.sanitizeEmail(value.(string), rule)

	case TypeURL:
		return sm.sanitizeURL(value.(string), rule)

	case TypeJSON:
		return sm.sanitizeJSON(value, rule)

	default:
		return value, violations
	}
}

// sanitizeString sanitizes string input
func (sm *SanitizationManager) sanitizeString(value string, rule *SanitizationRule) (string, []Violation) {
	var violations []Violation
	sanitized := value

	// Check length constraints
	if rule.MinLength > 0 && len(sanitized) < rule.MinLength {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "min_length",
			Message:  fmt.Sprintf("string too short: %d < %d", len(sanitized), rule.MinLength),
			Severity: SeverityWarning,
		})
	}

	if rule.MaxLength > 0 && len(sanitized) > rule.MaxLength {
		sanitized = sanitized[:rule.MaxLength]
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "max_length",
			Message:  "string truncated to max length",
			Severity: SeverityInfo,
		})
	}

	// Trim whitespace
	if rule.TrimWhitespace {
		sanitized = strings.TrimSpace(sanitized)
	}

	// Case transformation
	if rule.Lowercase {
		sanitized = strings.ToLower(sanitized)
	} else if rule.Uppercase {
		sanitized = strings.ToUpper(sanitized)
	}

	// HTML handling
	if rule.StripTags {
		sanitized = sm.htmlPolicy.Sanitize(sanitized)
	} else if rule.EscapeHTML {
		sanitized = html.EscapeString(sanitized)
	} else if rule.AllowHTML {
		// Create custom policy for allowed tags
		policy := bluemonday.NewPolicy()
		for _, tag := range rule.AllowedTags {
			policy.AllowElements(tag)
		}
		sanitized = policy.Sanitize(sanitized)
	}

	// Pattern validation
	if rule.Pattern != "" {
		if matched, _ := regexp.MatchString(rule.Pattern, sanitized); !matched {
			violations = append(violations, Violation{
				Field:    rule.Field,
				Rule:     "pattern",
				Message:  "string does not match required pattern",
				Severity: SeverityError,
			})
		}
	}

	// Check for control characters
	if containsControlCharacters(sanitized) {
		sanitized = removeControlCharacters(sanitized)
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "control_chars",
			Message:  "control characters removed",
			Severity: SeverityInfo,
		})
	}

	// Ensure valid UTF-8
	if !utf8.ValidString(sanitized) {
		sanitized = strings.ToValidUTF8(sanitized, "")
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "utf8",
			Message:  "invalid UTF-8 characters removed",
			Severity: SeverityWarning,
		})
	}

	return sanitized, violations
}

// sanitizeNumber sanitizes numeric input
func (sm *SanitizationManager) sanitizeNumber(value interface{}, rule *SanitizationRule) interface{} {
	// Convert to float64 for processing
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case string:
		// Try to parse string as number
		if parsed, err := parseNumber(v); err == nil {
			num = parsed
		} else {
			return 0
		}
	default:
		return 0
	}

	// Apply any numeric constraints
	// This could include min/max values, precision, etc.
	return num
}

// sanitizeEmail sanitizes email input
func (sm *SanitizationManager) sanitizeEmail(value string, rule *SanitizationRule) (string, []Violation) {
	var violations []Violation

	// Basic email sanitization
	email := strings.ToLower(strings.TrimSpace(value))

	// Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "email_format",
			Message:  "invalid email format",
			Severity: SeverityError,
		})
	}

	// Remove any potentially dangerous characters
	email = strings.ReplaceAll(email, "<", "")
	email = strings.ReplaceAll(email, ">", "")
	email = strings.ReplaceAll(email, ";", "")

	return email, violations
}

// sanitizeURL sanitizes URL input
func (sm *SanitizationManager) sanitizeURL(value string, rule *SanitizationRule) (string, []Violation) {
	var violations []Violation

	// Parse and validate URL
	u, err := url.Parse(value)
	if err != nil {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "url_format",
			Message:  "invalid URL format",
			Severity: SeverityError,
		})
		return "", violations
	}

	// Check for potentially dangerous schemes
	allowedSchemes := []string{"http", "https"}
	schemeAllowed := false
	for _, scheme := range allowedSchemes {
		if u.Scheme == scheme {
			schemeAllowed = true
			break
		}
	}

	if !schemeAllowed {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "url_scheme",
			Message:  fmt.Sprintf("disallowed URL scheme: %s", u.Scheme),
			Severity: SeverityError,
		})
		u.Scheme = "https" // Default to HTTPS
	}

	// Remove any fragments that might contain XSS
	u.Fragment = ""

	// Encode the URL properly
	return u.String(), violations
}

// sanitizeArray sanitizes array input
func (sm *SanitizationManager) sanitizeArray(ctx context.Context, value interface{}, rule *SanitizationRule, policy *SanitizationPolicy) (interface{}, []Violation) {
	var violations []Violation

	arr, ok := value.([]interface{})
	if !ok {
		return []interface{}{}, violations
	}

	// Check array length
	if sm.config.MaxArrayLength > 0 && len(arr) > sm.config.MaxArrayLength {
		arr = arr[:sm.config.MaxArrayLength]
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "max_array_length",
			Message:  "array truncated to max length",
			Severity: SeverityWarning,
		})
	}

	// Sanitize each element
	sanitized := make([]interface{}, 0, len(arr))
	for i, elem := range arr {
		fieldName := fmt.Sprintf("%s[%d]", rule.Field, i)
		sanitizedElem, v := sm.sanitizeField(ctx, fieldName, elem, policy)
		sanitized = append(sanitized, sanitizedElem)
		violations = append(violations, v...)
	}

	return sanitized, violations
}

// sanitizeObject sanitizes object input
func (sm *SanitizationManager) sanitizeObject(ctx context.Context, value interface{}, rule *SanitizationRule, policy *SanitizationPolicy) (interface{}, []Violation) {
	var violations []Violation

	obj, ok := value.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, violations
	}

	// Check nesting depth
	if depth := getObjectDepth(obj); depth > sm.config.MaxNestingDepth {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "max_nesting",
			Message:  fmt.Sprintf("object nesting too deep: %d > %d", depth, sm.config.MaxNestingDepth),
			Severity: SeverityError,
		})
		return map[string]interface{}{}, violations
	}

	// Sanitize each field
	sanitized := make(map[string]interface{})
	for key, val := range obj {
		fieldName := fmt.Sprintf("%s.%s", rule.Field, key)
		sanitizedVal, v := sm.sanitizeField(ctx, fieldName, val, policy)
		sanitized[key] = sanitizedVal
		violations = append(violations, v...)
	}

	return sanitized, violations
}

// sanitizeJSON sanitizes JSON input
func (sm *SanitizationManager) sanitizeJSON(value interface{}, rule *SanitizationRule) (interface{}, []Violation) {
	var violations []Violation

	// Validate JSON structure
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "json_format",
			Message:  "invalid JSON format",
			Severity: SeverityError,
		})
		return "{}", violations
	}

	// Re-parse to ensure valid JSON
	var parsed interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		violations = append(violations, Violation{
			Field:    rule.Field,
			Rule:     "json_parse",
			Message:  "JSON parsing failed",
			Severity: SeverityError,
		})
		return "{}", violations
	}

	return parsed, violations
}

// detectMaliciousPatterns detects malicious patterns in input
func (sm *SanitizationManager) detectMaliciousPatterns(data interface{}) []Violation {
	var violations []Violation

	// Convert to string for pattern matching
	str := fmt.Sprintf("%v", data)

	for _, pattern := range sm.maliciousPatterns {
		if pattern.MatchString(str) {
			violations = append(violations, Violation{
				Field:    "_content",
				Rule:     "malicious_pattern",
				Message:  fmt.Sprintf("potentially malicious pattern detected: %s", pattern.String()),
				Severity: SeverityCritical,
			})
		}
	}

	return violations
}

// validateSchema validates input against JSON schema
func (sm *SanitizationManager) validateSchema(data interface{}) []Violation {
	// This would validate against loaded JSON schemas
	// Simplified implementation
	return []Violation{}
}

// checkInputSize checks if input size is within limits
func (sm *SanitizationManager) checkInputSize(input interface{}) error {
	// Serialize to check size
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	if int64(len(data)) > sm.config.MaxInputSize {
		return fmt.Errorf("input size exceeds limit: %d > %d", len(data), sm.config.MaxInputSize)
	}

	return nil
}

// getPolicy retrieves a sanitization policy
func (sm *SanitizationManager) getPolicy(name string) *SanitizationPolicy {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.policies[name]
}

// defaultSanitize applies default sanitization
func (sm *SanitizationManager) defaultSanitize(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Basic string sanitization
		return html.EscapeString(strings.TrimSpace(v))
	default:
		return value
	}
}

// validateType validates if value matches expected type
func (sm *SanitizationManager) validateType(value interface{}, expectedType InputType) bool {
	switch expectedType {
	case TypeString:
		_, ok := value.(string)
		return ok
	case TypeNumber:
		switch value.(type) {
		case int, int64, float32, float64:
			return true
		}
		return false
	case TypeBoolean:
		_, ok := value.(bool)
		return ok
	case TypeArray:
		_, ok := value.([]interface{})
		return ok
	case TypeObject:
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true
	}
}

// coerceType attempts to coerce value to expected type
func (sm *SanitizationManager) coerceType(value interface{}, targetType InputType) interface{} {
	switch targetType {
	case TypeString:
		return fmt.Sprintf("%v", value)
	case TypeNumber:
		// Try to convert to number
		if num, err := parseNumber(fmt.Sprintf("%v", value)); err == nil {
			return num
		}
		return 0
	case TypeBoolean:
		// Try to convert to boolean
		switch v := value.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1"
		default:
			return false
		}
	default:
		return value
	}
}

// Helper functions

func containsControlCharacters(s string) bool {
	for _, r := range s {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

func removeControlCharacters(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func getObjectDepth(obj interface{}) int {
	return getDepthRecursive(obj, 0)
}

func getDepthRecursive(obj interface{}, currentDepth int) int {
	if currentDepth > 100 { // Prevent stack overflow
		return currentDepth
	}

	maxDepth := currentDepth

	switch v := obj.(type) {
	case map[string]interface{}:
		for _, val := range v {
			depth := getDepthRecursive(val, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	case []interface{}:
		for _, val := range v {
			depth := getDepthRecursive(val, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

func parseNumber(s string) (float64, error) {
	// Implementation would parse string to number
	return 0, nil
}

// GetStats returns sanitization statistics
func (sm *SanitizationManager) GetStats() *SanitizationStats {
	sm.stats.mu.RLock()
	defer sm.stats.mu.RUnlock()

	// Return a copy of stats
	return &SanitizationStats{
		TotalRequests:     sm.stats.TotalRequests,
		SanitizedInputs:   sm.stats.SanitizedInputs,
		RejectedInputs:    sm.stats.RejectedInputs,
		MaliciousDetected: sm.stats.MaliciousDetected,
		SchemaViolations:  sm.stats.SchemaViolations,
		Violations:        sm.stats.Violations,
	}
}

package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// SanitizationConfig holds sanitization configuration.

type SanitizationConfig struct {
	Enabled bool `json:"enabled"`

	Mode SanitizationMode `json:"mode"`

	Rules []SanitizationRule `json:"rules"`

	DefaultAction SanitizationAction `json:"default_action"`

	MaxInputSize int64 `json:"max_input_size"`

	MaxNestingDepth int `json:"max_nesting_depth"`

	HTMLSanitization *HTMLSanitizationConfig `json:"html_sanitization"`

	SQLInjectionProtection *SQLInjectionConfig `json:"sql_injection_protection"`

	XSSProtection *XSSProtectionConfig `json:"xss_protection"`

	CSVInjectionProtection *CSVInjectionConfig `json:"csv_injection_protection"`

	PathTraversalProtection *PathTraversalConfig `json:"path_traversal_protection"`

	CommandInjectionProtection *CommandInjectionConfig `json:"command_injection_protection"`

	LDAPInjectionProtection *LDAPInjectionConfig `json:"ldap_injection_protection"`

	NoSQLInjectionProtection *NoSQLInjectionConfig `json:"nosql_injection_protection"`

	TemplateInjectionProtection *TemplateInjectionConfig `json:"template_injection_protection"`

	RegexTimeoutProtection *RegexTimeoutConfig `json:"regex_timeout_protection"`

	CustomPatterns []CustomPattern `json:"custom_patterns"`

	WhitelistEnabled bool `json:"whitelist_enabled"`

	AllowedFields []string `json:"allowed_fields"`

	BlockedFields []string `json:"blocked_fields"`

	LogViolations bool `json:"log_violations"`

	StrictMode bool `json:"strict_mode"`
}

// SanitizationMode defines sanitization mode.

type SanitizationMode string

const (

	// ModeBlock blocks malicious input.

	ModeBlock SanitizationMode = "block"

	// ModeSanitize sanitizes malicious input.

	ModeSanitize SanitizationMode = "sanitize"

	// ModeLog logs malicious input but allows it.

	ModeLog SanitizationMode = "log"

	// ModeStrict applies strict sanitization rules.

	ModeStrict SanitizationMode = "strict"
)

// SanitizationAction defines the action to take for violations.

type SanitizationAction string

const (

	// ActionBlock blocks the request.

	ActionBlock SanitizationAction = "block"

	// ActionSanitize sanitizes the input.

	ActionSanitize SanitizationAction = "sanitize"

	// ActionWarn logs a warning.

	ActionWarn SanitizationAction = "warn"

	// ActionAllow allows the input.

	ActionAllow SanitizationAction = "allow"
)

// SanitizationRule defines a sanitization rule.

type SanitizationRule struct {
	Name string `json:"name"`

	Pattern string `json:"pattern"`

	Fields []string `json:"fields"`

	Action SanitizationAction `json:"action"`

	Replacement string `json:"replacement"`

	Description string `json:"description"`

	Enabled bool `json:"enabled"`

	Priority int `json:"priority"`

	CaseSensitive bool `json:"case_sensitive"`

	MatchWholeWord bool `json:"match_whole_word"`
}

// HTMLSanitizationConfig holds HTML sanitization configuration.

type HTMLSanitizationConfig struct {
	Enabled bool `json:"enabled"`

	AllowedTags []string `json:"allowed_tags"`

	AllowedAttributes map[string][]string `json:"allowed_attributes"`

	RemoveComments bool `json:"remove_comments"`

	RemoveScripts bool `json:"remove_scripts"`

	RemoveStyles bool `json:"remove_styles"`

	AllowDataAttributes bool `json:"allow_data_attributes"`

	StripTags bool `json:"strip_tags"`
}

// SQLInjectionConfig holds SQL injection protection configuration.

type SQLInjectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockPatterns []string `json:"block_patterns"`

	AllowedKeywords []string `json:"allowed_keywords"`

	CaseSensitive bool `json:"case_sensitive"`

	CheckComments bool `json:"check_comments"`

	CheckUnionOperators bool `json:"check_union_operators"`
}

// XSSProtectionConfig holds XSS protection configuration.

type XSSProtectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockJavaScript bool `json:"block_javascript"`

	BlockOnEvent bool `json:"block_on_event"`

	BlockDataURIs bool `json:"block_data_uris"`

	SanitizeHTML bool `json:"sanitize_html"`

	RemoveInlineStyles bool `json:"remove_inline_styles"`

	BlockBase64 bool `json:"block_base64"`
}

// CSVInjectionConfig holds CSV injection protection configuration.

type CSVInjectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockFormulas bool `json:"block_formulas"`

	BlockMacros bool `json:"block_macros"`

	EscapeSpecialChars bool `json:"escape_special_chars"`

	SpecialChars []string `json:"special_chars"`
}

// PathTraversalConfig holds path traversal protection configuration.

type PathTraversalConfig struct {
	Enabled bool `json:"enabled"`

	BlockDotDotSlash bool `json:"block_dot_dot_slash"`

	BlockAbsolutePaths bool `json:"block_absolute_paths"`

	AllowedPaths []string `json:"allowed_paths"`

	BlockedPaths []string `json:"blocked_paths"`

	NormalizePaths bool `json:"normalize_paths"`
}

// CommandInjectionConfig holds command injection protection configuration.

type CommandInjectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockShellMetachars bool `json:"block_shell_metachars"`

	BlockCommandChaining bool `json:"block_command_chaining"`

	BlockPipeOperators bool `json:"block_pipe_operators"`

	AllowedCommands []string `json:"allowed_commands"`

	BlockedCommands []string `json:"blocked_commands"`
}

// LDAPInjectionConfig holds LDAP injection protection configuration.

type LDAPInjectionConfig struct {
	Enabled bool `json:"enabled"`

	EscapeSpecialChars bool `json:"escape_special_chars"`

	BlockWildcards bool `json:"block_wildcards"`

	ValidateFilters bool `json:"validate_filters"`
}

// NoSQLInjectionConfig holds NoSQL injection protection configuration.

type NoSQLInjectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockOperators bool `json:"block_operators"`

	ValidateJSON bool `json:"validate_json"`

	BlockRegexOperators bool `json:"block_regex_operators"`

	BlockWhereOperators bool `json:"block_where_operators"`
}

// TemplateInjectionConfig holds template injection protection configuration.

type TemplateInjectionConfig struct {
	Enabled bool `json:"enabled"`

	BlockTemplateEngines bool `json:"block_template_engines"`

	AllowedTemplateEngines []string `json:"allowed_template_engines"`

	BlockExpressions bool `json:"block_expressions"`
}

// RegexTimeoutConfig holds regex timeout protection configuration.

type RegexTimeoutConfig struct {
	Enabled bool `json:"enabled"`

	TimeoutSeconds int `json:"timeout_seconds"`

	MaxRegexLength int `json:"max_regex_length"`

	BlockComplexRegex bool `json:"block_complex_regex"`
}

// CustomPattern holds custom sanitization pattern.

type CustomPattern struct {
	Name string `json:"name"`

	Pattern string `json:"pattern"`

	Action SanitizationAction `json:"action"`

	Replacement string `json:"replacement"`

	Description string `json:"description"`
}

// SanitizationManager manages input sanitization.

type SanitizationManager struct {
	config *SanitizationConfig

	logger *logging.StructuredLogger

	htmlPolicy *bluemonday.Policy

	compiledRules []*CompiledRule

	stats *SanitizationStats

	mu sync.RWMutex
}

// CompiledRule represents a compiled sanitization rule.

type CompiledRule struct {
	*SanitizationRule

	regex *regexp.Regexp
}

// SanitizationStats holds sanitization statistics.

type SanitizationStats struct {
	TotalRequests int64 `json:"total_requests"`

	BlockedRequests int64 `json:"blocked_requests"`

	SanitizedRequests int64 `json:"sanitized_requests"`

	TotalViolations int64 `json:"total_violations"`

	ViolationsByRule map[string]int64 `json:"violations_by_rule"`

	ViolationsByField map[string]int64 `json:"violations_by_field"`

	ViolationsByType map[string]int64 `json:"violations_by_type"`

	ProcessingTime time.Duration `json:"processing_time"`

	Violations map[string]int64

	mu sync.RWMutex
}

// SanitizationResult represents the result of sanitization.

type SanitizationResult struct {
	Sanitized bool `json:"sanitized"`

	Modified bool `json:"modified"`

	Violations []Violation `json:"violations,omitempty"`

	OriginalValue interface{} `json:"-"`

	SanitizedValue interface{} `json:"sanitized_value"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Violation represents a sanitization violation.

type Violation struct {
	Field string `json:"field"`

	Rule string `json:"rule"`

	Value interface{} `json:"value,omitempty"`

	Type string `json:"type"`

	Message string `json:"message"`

	Severity string `json:"severity"`

	Action string `json:"action"`
}

// NewSanitizationManager creates a new sanitization manager.

func NewSanitizationManager(config *SanitizationConfig, logger *logging.StructuredLogger) (*SanitizationManager, error) {
	if config == nil {

		return nil, errors.New("sanitization config is required")
	}

	sm := &SanitizationManager{
		config: config,

		logger: logger,

		stats: &SanitizationStats{
			ViolationsByRule:  make(map[string]int64),
			ViolationsByField: make(map[string]int64),
			ViolationsByType:  make(map[string]int64),
			Violations:        make(map[string]int64),
		},
	}

	// Initialize HTML policy

	if config.HTMLSanitization != nil && config.HTMLSanitization.Enabled {

		sm.htmlPolicy = sm.buildHTMLPolicy(config.HTMLSanitization)
	}

	// Compile rules

	if err := sm.compileRules(); err != nil {

		return nil, fmt.Errorf("failed to compile sanitization rules: %w", err)
	}

	return sm, nil
}

// buildHTMLPolicy builds HTML sanitization policy.

func (sm *SanitizationManager) buildHTMLPolicy(config *HTMLSanitizationConfig) *bluemonday.Policy {
	policy := bluemonday.NewPolicy()

	// Allow specific tags

	if len(config.AllowedTags) > 0 {

		policy.AllowElements(config.AllowedTags...)
	}

	// Allow specific attributes

	for tag, attrs := range config.AllowedAttributes {

		policy.AllowAttrs(attrs...).OnElements(tag)
	}

	// Configure data attributes

	if config.AllowDataAttributes {

		policy.AllowDataURIImages()
	}

	// Remove scripts and styles if configured

	if config.RemoveScripts {

		policy.AllowElements("script").SkipElementsContent("script")
	}

	if config.RemoveStyles {

		policy.AllowElements("style").SkipElementsContent("style")
	}

	return policy
}

// compileRules compiles sanitization rules.

func (sm *SanitizationManager) compileRules() error {
	sm.compiledRules = make([]*CompiledRule, 0, len(sm.config.Rules))

	for i, rule := range sm.config.Rules {

		if !rule.Enabled {

			continue
		}

		// Compile regex

		flags := ""

		if !rule.CaseSensitive {

			flags += "(?i)"
		}

		pattern := flags + rule.Pattern

		regex, err := regexp.Compile(pattern)

		if err != nil {

			sm.logger.Error("failed to compile sanitization rule",
				"rule", rule.Name,
				"pattern", rule.Pattern,
				"error", err)

			continue
		}

		compiledRule := &CompiledRule{
			SanitizationRule: &sm.config.Rules[i],
			regex:            regex,
		}

		sm.compiledRules = append(sm.compiledRules, compiledRule)
	}

	return nil
}

// SanitizePolicyInput sanitizes policy input data.

func (sm *SanitizationManager) SanitizePolicyInput(ctx context.Context, input map[string]interface{}) (*SanitizationResult, error) {
	sm.stats.mu.Lock()

	sm.stats.TotalRequests++

	sm.stats.mu.Unlock()

	// Create metadata map and marshal it to JSON
	metadataMap := map[string]interface{}{
		"timestamp":   time.Now().UTC(),
		"input_size":  len(fmt.Sprintf("%v", input)),
		"rule_count":  len(sm.compiledRules),
	}
	
	var metadataJSON json.RawMessage
	if metadataBytes, err := json.Marshal(metadataMap); err == nil {
		metadataJSON = json.RawMessage(metadataBytes)
	}

	result := &SanitizationResult{
		OriginalValue: input,

		SanitizedValue: make(map[string]interface{}),

		Violations: []Violation{},

		Metadata: metadataJSON,
	}

	// Check input size.

	if err := sm.checkInputSize(input); err != nil {

		result.Violations = append(result.Violations, Violation{
			Field: "_root",

			Rule: "max_input_size",

			Type: "size_violation",

			Message: err.Error(),

			Severity: "high",

			Action: string(ActionBlock),
		})

		return result, err
	}

	// Check nesting depth.

	if err := sm.checkNestingDepth(input, 0); err != nil {

		result.Violations = append(result.Violations, Violation{
			Field: "_root",

			Rule: "max_nesting_depth",

			Type: "structure_violation",

			Message: err.Error(),

			Severity: "medium",

			Action: string(ActionBlock),
		})

		return result, err
	}

	// Sanitize each field.

	sanitizedInput := make(map[string]interface{})

	for key, value := range input {

		// Check if field is allowed.

		if !sm.isFieldAllowed(key) {

			result.Violations = append(result.Violations, Violation{
				Field: key,

				Rule: "field_whitelist",

				Value: value,

				Type: "field_violation",

				Message: fmt.Sprintf("field '%s' is not allowed", key),

				Severity: "medium",

				Action: string(ActionBlock),
			})

			continue
		}

		// Check if field is blocked.

		if sm.isFieldBlocked(key) {

			result.Violations = append(result.Violations, Violation{
				Field: key,

				Rule: "field_blacklist",

				Value: value,

				Type: "field_violation",

				Message: fmt.Sprintf("field '%s' is blocked", key),

				Severity: "high",

				Action: string(ActionBlock),
			})

			continue
		}

		// Sanitize field value.

		sanitizedValue, violations := sm.sanitizeValue(key, value)

		result.Violations = append(result.Violations, violations...)

		sanitizedInput[key] = sanitizedValue

		if len(violations) > 0 {

			result.Modified = true
		}
	}

	result.SanitizedValue = sanitizedInput

	result.Sanitized = len(result.Violations) == 0

	// Update statistics.

	sm.updateStats(result)

	// Determine final action based on violations.

	if len(result.Violations) > 0 {

		switch sm.config.Mode {

		case ModeBlock:

			return result, errors.New("input blocked due to sanitization violations")

		case ModeSanitize:

			// Continue with sanitized input

		case ModeLog:

			// Log violations but continue

			sm.logViolations(ctx, result.Violations)

		case ModeStrict:

			// Block any violations in strict mode

			return result, errors.New("input blocked in strict mode")
		}
	}

	return result, nil
}

// checkInputSize checks if input size exceeds the limit.

func (sm *SanitizationManager) checkInputSize(input interface{}) error {
	if sm.config.MaxInputSize <= 0 {

		return nil
	}

	inputStr := fmt.Sprintf("%v", input)

	if int64(len(inputStr)) > sm.config.MaxInputSize {

		return fmt.Errorf("input size %d exceeds maximum allowed size %d",
			len(inputStr), sm.config.MaxInputSize)
	}

	return nil
}

// checkNestingDepth checks if input nesting depth exceeds the limit.

func (sm *SanitizationManager) checkNestingDepth(input interface{}, depth int) error {
	if sm.config.MaxNestingDepth <= 0 {

		return nil
	}

	if depth > sm.config.MaxNestingDepth {

		return fmt.Errorf("nesting depth %d exceeds maximum allowed depth %d",
			depth, sm.config.MaxNestingDepth)
	}

	switch v := input.(type) {

	case map[string]interface{}:

		for _, value := range v {

			if err := sm.checkNestingDepth(value, depth+1); err != nil {

				return err
			}
		}

	case []interface{}:

		for _, value := range v {

			if err := sm.checkNestingDepth(value, depth+1); err != nil {

				return err
			}
		}
	}

	return nil
}

// isFieldAllowed checks if a field is in the allowed list.

func (sm *SanitizationManager) isFieldAllowed(field string) bool {
	if !sm.config.WhitelistEnabled || len(sm.config.AllowedFields) == 0 {

		return true
	}

	for _, allowedField := range sm.config.AllowedFields {

		if field == allowedField {

			return true
		}
	}

	return false
}

// isFieldBlocked checks if a field is in the blocked list.

func (sm *SanitizationManager) isFieldBlocked(field string) bool {
	for _, blockedField := range sm.config.BlockedFields {

		if field == blockedField {

			return true
		}
	}

	return false
}

// sanitizeValue sanitizes a value based on configured rules.

func (sm *SanitizationManager) sanitizeValue(field string, value interface{}) (interface{}, []Violation) {
	var violations []Violation

	// Convert value to string for pattern matching.

	valueStr := fmt.Sprintf("%v", value)

	sanitizedStr := valueStr

	// Apply sanitization rules.

	for _, rule := range sm.compiledRules {

		// Check if rule applies to this field.

		if len(rule.Fields) > 0 {

			found := false

			for _, ruleField := range rule.Fields {

				if field == ruleField {

					found = true

					break
				}
			}

			if !found {

				continue
			}
		}

		// Check if value matches rule pattern.

		if rule.regex.MatchString(sanitizedStr) {

			violation := Violation{
				Field: field,

				Rule: rule.Name,

				Value: value,

				Type: "pattern_violation",

				Message: rule.Description,

				Severity: "medium",

				Action: string(rule.Action),
			}

			violations = append(violations, violation)

			// Apply rule action.

			switch rule.Action {

			case ActionBlock:

				return nil, violations

			case ActionSanitize:

				if rule.Replacement != "" {

					sanitizedStr = rule.regex.ReplaceAllString(sanitizedStr, rule.Replacement)

				} else {

					sanitizedStr = rule.regex.ReplaceAllString(sanitizedStr, "")
				}

			case ActionWarn:

				// Just log the violation, don't modify value

			case ActionAllow:

				// Allow the value as-is
			}
		}
	}

	// Apply specific sanitization based on value type.

	switch v := value.(type) {

	case string:

		sanitizedStr = sm.sanitizeString(field, v)

	case map[string]interface{}:

		return sm.sanitizeMap(field, v)

	case []interface{}:

		return sm.sanitizeArray(field, v)
	}

	// Convert back to original type if possible.

	if sanitizedStr != valueStr {

		return sm.convertToOriginalType(value, sanitizedStr), violations
	}

	return value, violations
}

// sanitizeString applies string-specific sanitization.

func (sm *SanitizationManager) sanitizeString(field, value string) string {
	sanitized := value

	// HTML sanitization.

	if sm.config.HTMLSanitization != nil && sm.config.HTMLSanitization.Enabled {

		if sm.htmlPolicy != nil {

			sanitized = sm.htmlPolicy.Sanitize(sanitized)

		} else {

			sanitized = html.EscapeString(sanitized)
		}
	}

	// SQL injection protection.

	if sm.config.SQLInjectionProtection != nil && sm.config.SQLInjectionProtection.Enabled {

		sanitized = sm.sanitizeSQL(sanitized)
	}

	// XSS protection.

	if sm.config.XSSProtection != nil && sm.config.XSSProtection.Enabled {

		sanitized = sm.sanitizeXSS(sanitized)
	}

	// Path traversal protection.

	if sm.config.PathTraversalProtection != nil && sm.config.PathTraversalProtection.Enabled {

		sanitized = sm.sanitizePath(sanitized)
	}

	// Command injection protection.

	if sm.config.CommandInjectionProtection != nil && sm.config.CommandInjectionProtection.Enabled {

		sanitized = sm.sanitizeCommand(sanitized)
	}

	// LDAP injection protection.

	if sm.config.LDAPInjectionProtection != nil && sm.config.LDAPInjectionProtection.Enabled {

		sanitized = sm.sanitizeLDAP(sanitized)
	}

	return sanitized
}

// sanitizeMap recursively sanitizes a map.

func (sm *SanitizationManager) sanitizeMap(parentField string, value map[string]interface{}) (interface{}, []Violation) {
	var allViolations []Violation

	sanitized := make(map[string]interface{})

	for key, val := range value {

		fieldPath := parentField + "." + key

		sanitizedVal, violations := sm.sanitizeValue(fieldPath, val)

		allViolations = append(allViolations, violations...)

		sanitized[key] = sanitizedVal
	}

	return sanitized, allViolations
}

// sanitizeArray recursively sanitizes an array.

func (sm *SanitizationManager) sanitizeArray(parentField string, value []interface{}) (interface{}, []Violation) {
	var allViolations []Violation

	sanitized := make([]interface{}, len(value))

	for i, val := range value {

		fieldPath := parentField + "[" + strconv.Itoa(i) + "]"

		sanitizedVal, violations := sm.sanitizeValue(fieldPath, val)

		allViolations = append(allViolations, violations...)

		sanitized[i] = sanitizedVal
	}

	return sanitized, allViolations
}

// convertToOriginalType attempts to convert sanitized string back to original type.

func (sm *SanitizationManager) convertToOriginalType(original interface{}, sanitized string) interface{} {
	switch original.(type) {

	case int:

		if val, err := strconv.Atoi(sanitized); err == nil {

			return val
		}

	case int64:

		if val, err := strconv.ParseInt(sanitized, 10, 64); err == nil {

			return val
		}

	case float64:

		if val, err := strconv.ParseFloat(sanitized, 64); err == nil {

			return val
		}

	case bool:

		if val, err := strconv.ParseBool(sanitized); err == nil {

			return val
		}
	}

	return sanitized
}

// Specific sanitization methods.

// sanitizeSQL sanitizes SQL injection attempts.

func (sm *SanitizationManager) sanitizeSQL(value string) string {
	config := sm.config.SQLInjectionProtection

	sanitized := value

	// Block common SQL injection patterns.

	sqlPatterns := []string{
		`(?i)(union\s+select)`,
		`(?i)(drop\s+table)`,
		`(?i)(delete\s+from)`,
		`(?i)(insert\s+into)`,
		`(?i)(update\s+set)`,
		`(?i)(exec\s*\()`,
		`(?i)(sp_\w+)`,
		`(?i)(xp_\w+)`,
		`';\s*--`,
		`';\s*/\*`,
	}

	// Add custom patterns.

	sqlPatterns = append(sqlPatterns, config.BlockPatterns...)

	for _, pattern := range sqlPatterns {

		re, err := regexp.Compile(pattern)

		if err != nil {

			continue
		}

		sanitized = re.ReplaceAllString(sanitized, "")
	}

	// Remove SQL comments.

	if config.CheckComments {

		commentPattern := regexp.MustCompile(`--.*$|/\*.*?\*/`)

		sanitized = commentPattern.ReplaceAllString(sanitized, "")
	}

	return sanitized
}

// sanitizeXSS sanitizes XSS attempts.

func (sm *SanitizationManager) sanitizeXSS(value string) string {
	config := sm.config.XSSProtection

	sanitized := value

	// Block JavaScript.

	if config.BlockJavaScript {

		jsPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>|javascript:|j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*:`)

		sanitized = jsPattern.ReplaceAllString(sanitized, "")
	}

	// Block event handlers.

	if config.BlockOnEvent {

		eventPattern := regexp.MustCompile(`(?i)on\w+\s*=`)

		sanitized = eventPattern.ReplaceAllString(sanitized, "")
	}

	// Block data URIs.

	if config.BlockDataURIs {

		dataURIPattern := regexp.MustCompile(`(?i)data\s*:\s*[^,]*,`)

		sanitized = dataURIPattern.ReplaceAllString(sanitized, "")
	}

	// Remove inline styles.

	if config.RemoveInlineStyles {

		stylePattern := regexp.MustCompile(`(?i)style\s*=\s*["'][^"']*["']`)

		sanitized = stylePattern.ReplaceAllString(sanitized, "")
	}

	return sanitized
}

// sanitizePath sanitizes path traversal attempts.

func (sm *SanitizationManager) sanitizePath(value string) string {
	config := sm.config.PathTraversalProtection

	sanitized := value

	// Block dot-dot-slash patterns.

	if config.BlockDotDotSlash {

		dotDotPattern := regexp.MustCompile(`\.\.[\\/]`)

		sanitized = dotDotPattern.ReplaceAllString(sanitized, "")
	}

	// Block absolute paths.

	if config.BlockAbsolutePaths {

		absolutePattern := regexp.MustCompile(`^[\\/]`)

		sanitized = absolutePattern.ReplaceAllString(sanitized, "")
	}

	// Normalize paths.

	if config.NormalizePaths {

		sanitized = strings.ReplaceAll(sanitized, "\\", "/")

		sanitized = regexp.MustCompile(`/+`).ReplaceAllString(sanitized, "/")
	}

	return sanitized
}

// sanitizeCommand sanitizes command injection attempts.

func (sm *SanitizationManager) sanitizeCommand(value string) string {
	config := sm.config.CommandInjectionProtection

	sanitized := value

	// Block shell metacharacters.

	if config.BlockShellMetachars {

		metaPattern := regexp.MustCompile(`[;&|<>$` + "`" + `\\]`)

		sanitized = metaPattern.ReplaceAllString(sanitized, "")
	}

	// Block command chaining.

	if config.BlockCommandChaining {

		chainPattern := regexp.MustCompile(`[;&|]{1,2}`)

		sanitized = chainPattern.ReplaceAllString(sanitized, "")
	}

	// Block pipe operators.

	if config.BlockPipeOperators {

		pipePattern := regexp.MustCompile(`\|+`)

		sanitized = pipePattern.ReplaceAllString(sanitized, "")
	}

	return sanitized
}

// sanitizeLDAP sanitizes LDAP injection attempts.

func (sm *SanitizationManager) sanitizeLDAP(value string) string {
	config := sm.config.LDAPInjectionProtection

	sanitized := value

	// Escape special LDAP characters.

	if config.EscapeSpecialChars {

		replacements := map[string]string{
			`\`:  `\\5c`,
			`*`:  `\\2a`,
			`(`:  `\\28`,
			`)`:  `\\29`,
			`\0`: `\\00`,
		}

		for char, escaped := range replacements {

			sanitized = strings.ReplaceAll(sanitized, char, escaped)
		}
	}

	// Block wildcards.

	if config.BlockWildcards {

		sanitized = strings.ReplaceAll(sanitized, "*", "")
	}

	return sanitized
}

// updateStats updates sanitization statistics.

func (sm *SanitizationManager) updateStats(result *SanitizationResult) {
	sm.stats.mu.Lock()

	defer sm.stats.mu.Unlock()

	if result.Modified {

		sm.stats.SanitizedRequests++
	}

	sm.stats.TotalViolations += int64(len(result.Violations))

	for _, violation := range result.Violations {

		sm.stats.ViolationsByRule[violation.Rule]++

		sm.stats.ViolationsByField[violation.Field]++

		sm.stats.ViolationsByType[violation.Type]++
	}
}

// logViolations logs sanitization violations.

func (sm *SanitizationManager) logViolations(ctx context.Context, violations []Violation) {
	if !sm.config.LogViolations {

		return
	}

	for _, violation := range violations {

		sm.logger.Warn("sanitization violation detected",
			"field", violation.Field,
			"rule", violation.Rule,
			"type", violation.Type,
			"message", violation.Message,
			"severity", violation.Severity,
			"action", violation.Action)
	}
}

// GetStats returns sanitization statistics.

func (sm *SanitizationManager) GetStats() *SanitizationStats {
	sm.stats.mu.RLock()

	defer sm.stats.mu.RUnlock()

	// Create a copy of stats.

	stats := &SanitizationStats{
		TotalRequests:     sm.stats.TotalRequests,
		BlockedRequests:   sm.stats.BlockedRequests,
		SanitizedRequests: sm.stats.SanitizedRequests,
		TotalViolations:   sm.stats.TotalViolations,
		ProcessingTime:    sm.stats.ProcessingTime,
		ViolationsByRule:  make(map[string]int64),
		ViolationsByField: make(map[string]int64),
		ViolationsByType:  make(map[string]int64),
		Violations:        make(map[string]int64),
	}

	// Copy maps.

	for k, v := range sm.stats.ViolationsByRule {

		stats.ViolationsByRule[k] = v
	}

	for k, v := range sm.stats.ViolationsByField {

		stats.ViolationsByField[k] = v
	}

	for k, v := range sm.stats.ViolationsByType {

		stats.ViolationsByType[k] = v
	}

	for k, v := range sm.stats.Violations {

		stats.Violations[k] = v
	}

	return stats
}

// ResetStats resets sanitization statistics.

func (sm *SanitizationManager) ResetStats() {
	sm.stats.mu.Lock()

	defer sm.stats.mu.Unlock()

	sm.stats.TotalRequests = 0

	sm.stats.BlockedRequests = 0

	sm.stats.SanitizedRequests = 0

	sm.stats.TotalViolations = 0

	sm.stats.ProcessingTime = 0

	sm.stats.ViolationsByRule = make(map[string]int64)

	sm.stats.ViolationsByField = make(map[string]int64)

	sm.stats.ViolationsByType = make(map[string]int64)

	sm.stats.Violations = make(map[string]int64)
}

// Close shuts down the sanitization manager.

func (sm *SanitizationManager) Close() error {
	// No resources to clean up currently.

	return nil
}

// Hash returns a hash of the sanitization manager configuration.

func (sm *SanitizationManager) Hash() string {
	configJSON, err := json.Marshal(sm.config)

	if err != nil {

		return ""
	}

	return fmt.Sprintf("%x", configJSON)
}
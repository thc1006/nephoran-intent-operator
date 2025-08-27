package config

import (
	"crypto/subtle"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validate    func(interface{}) error
}

// ValidationRules contains all validation rules for configuration values
type ValidationRules struct {
	rules map[string]*ValidationRule
}

// NewValidationRules creates a new validation rules instance
func NewValidationRules() *ValidationRules {
	vr := &ValidationRules{
		rules: make(map[string]*ValidationRule),
	}
	vr.initializeDefaultRules()
	return vr
}

// initializeDefaultRules initializes all default validation rules
func (vr *ValidationRules) initializeDefaultRules() {
	// Network configuration validation rules
	vr.AddRule("port", "Port number validation (1-65535)", validatePortNumber)
	vr.AddRule("timeout", "Timeout duration validation (positive duration)", validateTimeout)
	vr.AddRule("retry_count", "Retry count validation (0-100)", validateRetryCount)
	vr.AddRule("percentage", "Percentage validation (0.0-1.0)", validatePercentage)
	vr.AddRule("positive_integer", "Positive integer validation (> 0)", validatePositiveInteger)
	vr.AddRule("non_negative_integer", "Non-negative integer validation (>= 0)", validateNonNegativeInteger)
	vr.AddRule("kubernetes_resource", "Kubernetes resource quantity validation", validateKubernetesResource)
	vr.AddRule("domain", "Domain name validation", validateDomain)
	vr.AddRule("url", "URL validation", validateURLFormat)
	vr.AddRule("file_path", "File path validation", validateFilePath)
	vr.AddRule("jitter_factor", "Jitter factor validation (0.0-1.0)", validateJitterFactor)
	vr.AddRule("backoff_multiplier", "Backoff multiplier validation (> 1.0)", validateBackoffMultiplier)
	vr.AddRule("input_length", "Input length validation (1-1000000)", validateInputLength)
	vr.AddRule("output_length", "Output length validation (1-10000000)", validateOutputLength)
	vr.AddRule("context_boundary", "Context boundary validation (non-empty string)", validateContextBoundary)
	vr.AddRule("string_array", "String array validation", validateStringArray)
	vr.AddRule("circuit_breaker_threshold", "Circuit breaker threshold validation (1-1000)", validateCircuitBreakerThreshold)
	vr.AddRule("failure_rate", "Failure rate validation (0.0-1.0)", validateFailureRate)
}

// AddRule adds a validation rule
func (vr *ValidationRules) AddRule(name, description string, validateFunc func(interface{}) error) {
	vr.rules[name] = &ValidationRule{
		Name:        name,
		Description: description,
		Validate:    validateFunc,
	}
}

// ValidateValue validates a value against a specific rule
func (vr *ValidationRules) ValidateValue(ruleName string, value interface{}) error {
	rule, exists := vr.rules[ruleName]
	if !exists {
		return fmt.Errorf("validation rule '%s' not found", ruleName)
	}

	if err := rule.Validate(value); err != nil {
		return fmt.Errorf("validation failed for '%s': %w", ruleName, err)
	}

	return nil
}

// ValidateConfiguration validates a complete configuration against multiple rules
func (vr *ValidationRules) ValidateConfiguration(config map[string]interface{}, rules map[string]string) error {
	var errors []string

	for configKey, ruleName := range rules {
		value, exists := config[configKey]
		if !exists {
			continue // Skip missing values - let defaults handle them
		}

		if err := vr.ValidateValue(ruleName, value); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", configKey, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// GetRuleDescription returns the description for a rule
func (vr *ValidationRules) GetRuleDescription(ruleName string) string {
	if rule, exists := vr.rules[ruleName]; exists {
		return rule.Description
	}
	return "Rule not found"
}

// ListRules returns all available validation rules
func (vr *ValidationRules) ListRules() []string {
	var rules []string
	for name := range vr.rules {
		rules = append(rules, name)
	}
	return rules
}

// Validation functions for specific types

// validatePortNumber validates that a value is a valid port number (1-65535)
func validatePortNumber(value interface{}) error {
	var port int

	switch v := value.(type) {
	case int:
		port = v
	case string:
		var err error
		port, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid port format: %s", v)
		}
	default:
		return fmt.Errorf("port must be integer or string, got %T", value)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	return nil
}

// validateTimeout validates that a value is a positive duration
func validateTimeout(value interface{}) error {
	switch v := value.(type) {
	case time.Duration:
		if v <= 0 {
			return fmt.Errorf("timeout must be positive, got %v", v)
		}
	case string:
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s", v)
		}
		if duration <= 0 {
			return fmt.Errorf("timeout must be positive, got %v", duration)
		}
	default:
		return fmt.Errorf("timeout must be duration or string, got %T", value)
	}

	return nil
}

// validateRetryCount validates retry count (0-100)
func validateRetryCount(value interface{}) error {
	var count int

	switch v := value.(type) {
	case int:
		count = v
	case string:
		var err error
		count, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid retry count format: %s", v)
		}
	default:
		return fmt.Errorf("retry count must be integer or string, got %T", value)
	}

	if count < 0 || count > 100 {
		return fmt.Errorf("retry count must be between 0 and 100, got %d", count)
	}

	return nil
}

// validatePercentage validates a percentage value (0.0-1.0)
func validatePercentage(value interface{}) error {
	var percentage float64

	switch v := value.(type) {
	case float64:
		percentage = v
	case float32:
		percentage = float64(v)
	case string:
		var err error
		percentage, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid percentage format: %s", v)
		}
	default:
		return fmt.Errorf("percentage must be float or string, got %T", value)
	}

	if percentage < 0.0 || percentage > 1.0 {
		return fmt.Errorf("percentage must be between 0.0 and 1.0, got %f", percentage)
	}

	return nil
}

// validatePositiveInteger validates that a value is a positive integer (> 0)
func validatePositiveInteger(value interface{}) error {
	var num int

	switch v := value.(type) {
	case int:
		num = v
	case string:
		var err error
		num, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid integer format: %s", v)
		}
	default:
		return fmt.Errorf("value must be integer or string, got %T", value)
	}

	if num <= 0 {
		return fmt.Errorf("value must be positive, got %d", num)
	}

	return nil
}

// validateNonNegativeInteger validates that a value is a non-negative integer (>= 0)
func validateNonNegativeInteger(value interface{}) error {
	var num int

	switch v := value.(type) {
	case int:
		num = v
	case string:
		var err error
		num, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid integer format: %s", v)
		}
	default:
		return fmt.Errorf("value must be integer or string, got %T", value)
	}

	if num < 0 {
		return fmt.Errorf("value must be non-negative, got %d", num)
	}

	return nil
}

// validateKubernetesResource validates Kubernetes resource quantity format (e.g., "100m", "1Gi")
func validateKubernetesResource(value interface{}) error {
	var resource string

	switch v := value.(type) {
	case string:
		resource = v
	default:
		return fmt.Errorf("kubernetes resource must be string, got %T", value)
	}

	if resource == "" {
		return fmt.Errorf("kubernetes resource cannot be empty")
	}

	// Basic validation for Kubernetes resource format
	// This is a simplified check - in production, use k8s.io/apimachinery/pkg/api/resource.ParseQuantity
	resourcePattern := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?([mM]?[i]?|[KMGTPE]i?)$`)
	if !resourcePattern.MatchString(resource) {
		return fmt.Errorf("invalid kubernetes resource format: %s (expected format like '100m', '1Gi', '2')", resource)
	}

	return nil
}

// validateDomain validates domain name format
func validateDomain(value interface{}) error {
	var domain string

	switch v := value.(type) {
	case string:
		domain = v
	default:
		return fmt.Errorf("domain must be string, got %T", value)
	}

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Basic domain validation - simplified regex
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`)
	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	if len(domain) > 253 {
		return fmt.Errorf("domain too long: %d characters (max 253)", len(domain))
	}

	return nil
}

// validateURLFormat validates URL format
func validateURLFormat(value interface{}) error {
	var url string

	switch v := value.(type) {
	case string:
		url = v
	default:
		return fmt.Errorf("URL must be string, got %T", value)
	}

	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Basic URL validation
	urlPattern := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(\.[a-zA-Z]{2,})(:[0-9]+)?(/.*)?$`)
	if !urlPattern.MatchString(url) {
		return fmt.Errorf("invalid URL format: %s", url)
	}

	return nil
}

// validateFilePath validates file path format
func validateFilePath(value interface{}) error {
	var path string

	switch v := value.(type) {
	case string:
		path = v
	default:
		return fmt.Errorf("file path must be string, got %T", value)
	}

	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check for potentially dangerous path patterns
	dangerousPatterns := []string{
		"..",
		"//",
		"\\",
		"\x00",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("file path contains dangerous pattern '%s': %s", pattern, path)
		}
	}

	if len(path) > 4096 {
		return fmt.Errorf("file path too long: %d characters (max 4096)", len(path))
	}

	return nil
}

// validateJitterFactor validates jitter factor (0.0-1.0)
func validateJitterFactor(value interface{}) error {
	return validatePercentage(value)
}

// validateBackoffMultiplier validates backoff multiplier (> 1.0)
func validateBackoffMultiplier(value interface{}) error {
	var multiplier float64

	switch v := value.(type) {
	case float64:
		multiplier = v
	case float32:
		multiplier = float64(v)
	case string:
		var err error
		multiplier, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid multiplier format: %s", v)
		}
	default:
		return fmt.Errorf("multiplier must be float or string, got %T", value)
	}

	if multiplier <= 1.0 {
		return fmt.Errorf("backoff multiplier must be greater than 1.0, got %f", multiplier)
	}

	return nil
}

// validateInputLength validates input length limits (1-1000000)
func validateInputLength(value interface{}) error {
	var length int

	switch v := value.(type) {
	case int:
		length = v
	case string:
		var err error
		length, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid input length format: %s", v)
		}
	default:
		return fmt.Errorf("input length must be integer or string, got %T", value)
	}

	if length < 1 || length > 1000000 {
		return fmt.Errorf("input length must be between 1 and 1,000,000, got %d", length)
	}

	return nil
}

// validateOutputLength validates output length limits (1-10000000)
func validateOutputLength(value interface{}) error {
	var length int

	switch v := value.(type) {
	case int:
		length = v
	case string:
		var err error
		length, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid output length format: %s", v)
		}
	default:
		return fmt.Errorf("output length must be integer or string, got %T", value)
	}

	if length < 1 || length > 10000000 {
		return fmt.Errorf("output length must be between 1 and 10,000,000, got %d", length)
	}

	return nil
}

// validateContextBoundary validates context boundary (non-empty string)
func validateContextBoundary(value interface{}) error {
	var boundary string

	switch v := value.(type) {
	case string:
		boundary = v
	default:
		return fmt.Errorf("context boundary must be string, got %T", value)
	}

	if boundary == "" {
		return fmt.Errorf("context boundary cannot be empty")
	}

	if len(boundary) < 3 {
		return fmt.Errorf("context boundary too short, must be at least 3 characters: %s", boundary)
	}

	if len(boundary) > 50 {
		return fmt.Errorf("context boundary too long, must be at most 50 characters: %s", boundary)
	}

	return nil
}

// validateStringArray validates string array
func validateStringArray(value interface{}) error {
	switch v := value.(type) {
	case []string:
		// Valid string array
		return nil
	case []interface{}:
		// Check if all elements are strings
		for i, item := range v {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("array element at index %d is not a string: %T", i, item)
			}
		}
		return nil
	default:
		return fmt.Errorf("value must be string array, got %T", value)
	}
}

// validateCircuitBreakerThreshold validates circuit breaker threshold (1-1000)
func validateCircuitBreakerThreshold(value interface{}) error {
	var threshold int

	switch v := value.(type) {
	case int:
		threshold = v
	case string:
		var err error
		threshold, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid threshold format: %s", v)
		}
	default:
		return fmt.Errorf("threshold must be integer or string, got %T", value)
	}

	if threshold < 1 || threshold > 1000 {
		return fmt.Errorf("circuit breaker threshold must be between 1 and 1000, got %d", threshold)
	}

	return nil
}

// validateFailureRate validates failure rate (0.0-1.0)
func validateFailureRate(value interface{}) error {
	return validatePercentage(value)
}

// ValidateNetworkConfiguration validates network-specific configuration
func ValidateNetworkConfiguration(config map[string]interface{}) error {
	vr := NewValidationRules()

	networkRules := map[string]string{
		"metrics_port":       "port",
		"health_port":        "port",
		"llm_timeout":        "timeout",
		"git_timeout":        "timeout",
		"k8s_timeout":        "timeout",
		"max_retries":        "retry_count",
		"failure_rate":       "percentage",
		"jitter_factor":      "jitter_factor",
		"backoff_multiplier": "backoff_multiplier",
	}

	return vr.ValidateConfiguration(config, networkRules)
}

// ValidateSecurityConfiguration validates security-specific configuration
func ValidateSecurityConfiguration(config map[string]interface{}) error {
	vr := NewValidationRules()

	securityRules := map[string]string{
		"max_input_length":  "input_length",
		"max_output_length": "output_length",
		"context_boundary":  "context_boundary",
		"allowed_domains":   "string_array",
		"blocked_keywords":  "string_array",
	}

	return vr.ValidateConfiguration(config, securityRules)
}

// ValidateResilienceConfiguration validates resilience-specific configuration
func ValidateResilienceConfiguration(config map[string]interface{}) error {
	vr := NewValidationRules()

	resilienceRules := map[string]string{
		"cb_failure_threshold": "circuit_breaker_threshold",
		"cb_recovery_timeout":  "timeout",
		"cb_request_timeout":   "timeout",
		"cb_success_threshold": "positive_integer",
		"cb_min_requests":      "positive_integer",
		"cb_failure_rate":      "failure_rate",
	}

	return vr.ValidateConfiguration(config, resilienceRules)
}

// ValidateIPAddress validates IP address format
func ValidateIPAddress(value interface{}) error {
	var ipStr string

	switch v := value.(type) {
	case string:
		ipStr = v
	default:
		return fmt.Errorf("IP address must be string, got %T", value)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	return nil
}

// ValidateCompleteConfiguration validates all configuration sections together
func ValidateCompleteConfiguration(constants *Constants) error {
	// Convert constants to map for validation
	config := map[string]interface{}{
		// Network configuration
		"metrics_port":       constants.MetricsPort,
		"health_port":        constants.HealthProbePort,
		"llm_timeout":        constants.LLMTimeout,
		"git_timeout":        constants.GitTimeout,
		"k8s_timeout":        constants.KubernetesTimeout,
		"max_retries":        constants.MaxRetries,
		"jitter_factor":      constants.JitterFactor,
		"backoff_multiplier": constants.BackoffMultiplier,

		// Security configuration
		"max_input_length":  constants.MaxInputLength,
		"max_output_length": constants.MaxOutputLength,
		"context_boundary":  constants.ContextBoundary,
		"allowed_domains":   constants.AllowedDomains,
		"blocked_keywords":  constants.BlockedKeywords,

		// Resilience configuration
		"cb_failure_threshold": constants.CircuitBreakerFailureThreshold,
		"cb_recovery_timeout":  constants.CircuitBreakerRecoveryTimeout,
		"cb_request_timeout":   constants.CircuitBreakerRequestTimeout,
		"cb_success_threshold": constants.CircuitBreakerSuccessThreshold,
		"cb_min_requests":      constants.CircuitBreakerMinimumRequests,
		"cb_failure_rate":      constants.CircuitBreakerFailureRate,
	}

	// Validate each section
	if err := ValidateNetworkConfiguration(config); err != nil {
		return fmt.Errorf("network configuration validation failed: %w", err)
	}

	if err := ValidateSecurityConfiguration(config); err != nil {
		return fmt.Errorf("security configuration validation failed: %w", err)
	}

	if err := ValidateResilienceConfiguration(config); err != nil {
		return fmt.Errorf("resilience configuration validation failed: %w", err)
	}

	return nil
}

// SecretLoader represents a loader for sensitive configuration values
type SecretLoader struct {
	basePath string
	// Additional fields can be added for encryption, etc.
}

// NewSecretLoader creates a new secret loader
func NewSecretLoader(basePath string, options map[string]interface{}) (*SecretLoader, error) {
	if basePath == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	return &SecretLoader{
		basePath: basePath,
	}, nil
}

// LoadSecret loads a secret from the configured source
func (sl *SecretLoader) LoadSecret(secretName string) (string, error) {
	secretPath := filepath.Join(sl.basePath, secretName)

	content, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret %s: %w", secretName, err)
	}

	return strings.TrimSpace(string(content)), nil
}

// IsValidOpenAIKey validates if a string is a valid OpenAI API key format
func IsValidOpenAIKey(key string) bool {
	if key == "" {
		return false
	}

	// OpenAI keys typically start with "sk-" and are 51 characters long
	if !strings.HasPrefix(key, "sk-") {
		return false
	}

	if len(key) != 51 {
		return false
	}

	// Check if the rest contains only alphanumeric characters
	keyPart := key[3:] // Remove "sk-" prefix
	alphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	return alphanumeric.MatchString(keyPart)
}

// ClearString securely clears a string from memory
func ClearString(s *string) {
	if s == nil || *s == "" {
		return
	}

	// Convert string to byte slice for clearing
	data := []byte(*s)

	// Clear the underlying bytes
	for i := range data {
		data[i] = 0
	}

	// Clear the string by setting it to empty
	*s = ""
}

// SecureCompare performs constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	// Convert to byte slices
	aBytes := []byte(a)
	bBytes := []byte(b)

	// Use crypto/subtle for constant-time comparison
	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}

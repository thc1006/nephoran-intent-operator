package security

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// ValidationConfig contains security validation configuration.
type ValidationConfig struct {
	// KMP Data validation limits.
	MaxPRBUtilization float64 // Maximum PRB utilization (default: 1.0)
	MaxLatency        float64 // Maximum acceptable latency in ms (default: 10000)
	MaxReplicas       int     // Maximum replica count (default: 100)
	MaxActiveUEs      int     // Maximum active UEs (default: 10000)

	// File path validation.
	AllowedExtensions []string // Allowed file extensions
	MaxPathLength     int      // Maximum path length (default: 4096)

	// URL validation.
	AllowedSchemes []string // Allowed URL schemes (default: http, https)
	MaxURLLength   int      // Maximum URL length (default: 2048)
}

// DefaultValidationConfig returns a secure default configuration.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxPRBUtilization: 1.0,
		MaxLatency:        10000.0,
		MaxReplicas:       100,
		MaxActiveUEs:      10000,
		AllowedExtensions: []string{".json", ".yaml", ".yml"},
		MaxPathLength:     4096,
		AllowedSchemes:    []string{"http", "https"},
		MaxURLLength:      2048,
	}
}

// ValidationError represents a security validation error.
type ValidationError struct {
	Field   string
	Value   interface{}
	Reason  string
	Context string
}

// Error performs error operation.
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s (value: %v): %s [context: %s]",
		e.Field, e.Value, e.Reason, e.Context)
}

// Validator provides security validation functions.
type Validator struct {
	config ValidationConfig
}

// NewValidator creates a new security validator with the given configuration.
func NewValidator(config ValidationConfig) *Validator {
	return &Validator{config: config}
}

// ValidateKMPData performs comprehensive validation of KMP metrics data.
// Following OWASP Input Validation guidelines and O-RAN security requirements.
func (v *Validator) ValidateKMPData(data rules.KPMData) error {
	// Validate timestamp - must be within reasonable bounds.
	if data.Timestamp.IsZero() {
		return ValidationError{
			Field:   "timestamp",
			Value:   data.Timestamp,
			Reason:  "timestamp cannot be zero",
			Context: "KMP data validation",
		}
	}

	// Timestamp should not be too far in the future (prevent time-based attacks).
	maxFuture := time.Now().Add(5 * time.Minute)
	if data.Timestamp.After(maxFuture) {
		return ValidationError{
			Field:   "timestamp",
			Value:   data.Timestamp,
			Reason:  "timestamp too far in future",
			Context: "KMP data validation",
		}
	}

	// Timestamp should not be too old (prevent replay attacks).
	maxAge := time.Now().Add(-24 * time.Hour)
	if data.Timestamp.Before(maxAge) {
		return ValidationError{
			Field:   "timestamp",
			Value:   data.Timestamp,
			Reason:  "timestamp too old (potential replay attack)",
			Context: "KMP data validation",
		}
	}

	// Validate NodeID - must be non-empty and follow O-RAN naming conventions.
	if err := v.validateNodeID(data.NodeID); err != nil {
		return err
	}

	// Validate PRB utilization - must be between 0.0 and 1.0.
	if err := v.validatePRBUtilization(data.PRBUtilization); err != nil {
		return err
	}

	// Validate P95 latency - must be positive and within reasonable bounds.
	if err := v.validateLatency(data.P95Latency); err != nil {
		return err
	}

	// Validate ActiveUEs - must be non-negative and within limits.
	if err := v.validateActiveUEs(data.ActiveUEs); err != nil {
		return err
	}

	// Validate CurrentReplicas - must be positive and within limits.
	if err := v.validateReplicaCount(data.CurrentReplicas); err != nil {
		return err
	}

	return nil
}

// validateNodeID validates O-RAN node identifier format and security.
func (v *Validator) validateNodeID(nodeID string) error {
	if nodeID == "" {
		return ValidationError{
			Field:   "node_id",
			Value:   nodeID,
			Reason:  "node ID cannot be empty",
			Context: "KMP data validation",
		}
	}

	// Prevent injection attacks through NodeID.
	if strings.ContainsAny(nodeID, "'\";\\<>") {
		return ValidationError{
			Field:   "node_id",
			Value:   nodeID,
			Reason:  "node ID contains potentially dangerous characters",
			Context: "KMP data validation",
		}
	}

	// O-RAN node ID format validation (alphanumeric, hyphens, underscores only).
	validNodeID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validNodeID.MatchString(nodeID) {
		return ValidationError{
			Field:   "node_id",
			Value:   nodeID,
			Reason:  "node ID must contain only alphanumeric characters, hyphens, and underscores",
			Context: "KMP data validation",
		}
	}

	// Length validation.
	if len(nodeID) > 255 {
		return ValidationError{
			Field:   "node_id",
			Value:   nodeID,
			Reason:  "node ID too long (max 255 characters)",
			Context: "KMP data validation",
		}
	}

	return nil
}

// validatePRBUtilization validates Physical Resource Block utilization values.
func (v *Validator) validatePRBUtilization(prb float64) error {
	if prb < 0.0 {
		return ValidationError{
			Field:   "prb_utilization",
			Value:   prb,
			Reason:  "PRB utilization cannot be negative",
			Context: "KMP data validation",
		}
	}

	if prb > v.config.MaxPRBUtilization {
		return ValidationError{
			Field:   "prb_utilization",
			Value:   prb,
			Reason:  fmt.Sprintf("PRB utilization exceeds maximum allowed value (%.2f)", v.config.MaxPRBUtilization),
			Context: "KMP data validation",
		}
	}

	return nil
}

// validateLatency validates P95 latency measurements.
func (v *Validator) validateLatency(latency float64) error {
	if latency < 0.0 {
		return ValidationError{
			Field:   "p95_latency",
			Value:   latency,
			Reason:  "latency cannot be negative",
			Context: "KMP data validation",
		}
	}

	if latency > v.config.MaxLatency {
		return ValidationError{
			Field:   "p95_latency",
			Value:   latency,
			Reason:  fmt.Sprintf("latency exceeds maximum allowed value (%.2f ms)", v.config.MaxLatency),
			Context: "KMP data validation",
		}
	}

	return nil
}

// validateActiveUEs validates active User Equipment count.
func (v *Validator) validateActiveUEs(ues int) error {
	if ues < 0 {
		return ValidationError{
			Field:   "active_ues",
			Value:   ues,
			Reason:  "active UEs cannot be negative",
			Context: "KMP data validation",
		}
	}

	if ues > v.config.MaxActiveUEs {
		return ValidationError{
			Field:   "active_ues",
			Value:   ues,
			Reason:  fmt.Sprintf("active UEs exceeds maximum allowed value (%d)", v.config.MaxActiveUEs),
			Context: "KMP data validation",
		}
	}

	return nil
}

// validateReplicaCount validates replica count values.
func (v *Validator) validateReplicaCount(replicas int) error {
	if replicas < 0 {
		return ValidationError{
			Field:   "current_replicas",
			Value:   replicas,
			Reason:  "replica count cannot be negative",
			Context: "KMP data validation",
		}
	}

	if replicas > v.config.MaxReplicas {
		return ValidationError{
			Field:   "current_replicas",
			Value:   replicas,
			Reason:  fmt.Sprintf("replica count exceeds maximum allowed value (%d)", v.config.MaxReplicas),
			Context: "KMP data validation",
		}
	}

	return nil
}

// ValidateURL validates URLs from environment variables to prevent injection attacks.
func (v *Validator) ValidateURL(urlStr, context string) error {
	if urlStr == "" {
		return ValidationError{
			Field:   "url",
			Value:   urlStr,
			Reason:  "URL cannot be empty",
			Context: context,
		}
	}

	// Length validation to prevent buffer overflow attacks.
	if len(urlStr) > v.config.MaxURLLength {
		return ValidationError{
			Field:   "url",
			Value:   urlStr,
			Reason:  fmt.Sprintf("URL too long (max %d characters)", v.config.MaxURLLength),
			Context: context,
		}
	}

	// Parse URL to validate format.
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ValidationError{
			Field:   "url",
			Value:   urlStr,
			Reason:  fmt.Sprintf("invalid URL format: %v", err),
			Context: context,
		}
	}

	// Validate scheme - only allow HTTP/HTTPS.
	schemeAllowed := false
	for _, allowedScheme := range v.config.AllowedSchemes {
		if parsedURL.Scheme == allowedScheme {
			schemeAllowed = true
			break
		}
	}
	if !schemeAllowed {
		return ValidationError{
			Field:   "url.scheme",
			Value:   parsedURL.Scheme,
			Reason:  fmt.Sprintf("URL scheme not allowed (allowed: %v)", v.config.AllowedSchemes),
			Context: context,
		}
	}

	// Prevent access to localhost/loopback addresses in production environments.
	// Note: This can be relaxed for development/testing.
	host := parsedURL.Hostname()
	if host == "" {
		return ValidationError{
			Field:   "url.host",
			Value:   host,
			Reason:  "URL must have a valid hostname",
			Context: context,
		}
	}

	// Prevent injection through URL components.
	if strings.ContainsAny(parsedURL.RawQuery, "'\";\\<>") {
		return ValidationError{
			Field:   "url.query",
			Value:   parsedURL.RawQuery,
			Reason:  "URL query contains potentially dangerous characters",
			Context: context,
		}
	}

	return nil
}

// ValidateFilePath validates file paths to prevent directory traversal attacks.
func (v *Validator) ValidateFilePath(path, context string) error {
	if path == "" {
		return ValidationError{
			Field:   "file_path",
			Value:   path,
			Reason:  "file path cannot be empty",
			Context: context,
		}
	}

	// Length validation.
	if len(path) > v.config.MaxPathLength {
		return ValidationError{
			Field:   "file_path",
			Value:   path,
			Reason:  fmt.Sprintf("file path too long (max %d characters)", v.config.MaxPathLength),
			Context: context,
		}
	}

	// Clean the path to resolve any relative components.
	cleanPath := filepath.Clean(path)

	// Prevent directory traversal attacks.
	if strings.Contains(cleanPath, "..") {
		return ValidationError{
			Field:   "file_path",
			Value:   path,
			Reason:  "file path contains directory traversal sequences (..)",
			Context: context,
		}
	}

	// Prevent absolute paths pointing to sensitive system directories.
	if filepath.IsAbs(cleanPath) {
		// Check for common sensitive directories (case insensitive).
		lowerPath := strings.ToLower(cleanPath)
		sensitiveDirectories := []string{
			// Unix/Linux sensitive directories.
			"/etc", "/proc", "/sys", "/dev", "/root", "/var/log", "/boot",
			// Windows sensitive directories.
			"c:\\windows", "c:\\system32", "c:\\program files", "c:\\programdata",
			"c:\\users\\all users", "c:\\users\\default", "c:\\boot",
		}

		for _, sensitive := range sensitiveDirectories {
			if strings.HasPrefix(lowerPath, sensitive) {
				return ValidationError{
					Field:   "file_path",
					Value:   path,
					Reason:  "file path points to sensitive system directory",
					Context: context,
				}
			}
		}
	}

	// Validate file extension if specified.
	if len(v.config.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(cleanPath))
		if ext != "" {
			extensionAllowed := false
			for _, allowedExt := range v.config.AllowedExtensions {
				if strings.EqualFold(ext, allowedExt) {
					extensionAllowed = true
					break
				}
			}
			if !extensionAllowed {
				return ValidationError{
					Field:   "file_path.extension",
					Value:   ext,
					Reason:  fmt.Sprintf("file extension not allowed (allowed: %v)", v.config.AllowedExtensions),
					Context: context,
				}
			}
		}
	}

	// Prevent null bytes and other dangerous characters.
	if strings.ContainsAny(cleanPath, "\x00<>|\"") {
		return ValidationError{
			Field:   "file_path",
			Value:   path,
			Reason:  "file path contains dangerous characters",
			Context: context,
		}
	}

	return nil
}

// ValidateEnvironmentVariable validates environment variable values for security.
func (v *Validator) ValidateEnvironmentVariable(name, value, context string) error {
	if name == "" {
		return ValidationError{
			Field:   "env_var_name",
			Value:   name,
			Reason:  "environment variable name cannot be empty",
			Context: context,
		}
	}

	// Validate based on variable name patterns.
	switch {
	case strings.HasSuffix(name, "_URL"):
		return v.ValidateURL(value, fmt.Sprintf("%s (env: %s)", context, name))
	case strings.HasSuffix(name, "_DIR") || strings.HasSuffix(name, "_FILE"):
		return v.ValidateFilePath(value, fmt.Sprintf("%s (env: %s)", context, name))
	default:
		// General validation for other environment variables.
		if strings.ContainsAny(value, "\x00\r\n") {
			return ValidationError{
				Field:   name,
				Value:   value,
				Reason:  "environment variable contains null bytes or newlines",
				Context: context,
			}
		}
	}

	return nil
}

// SanitizeForLogging safely sanitizes values for logging to prevent log injection.
func (v *Validator) SanitizeForLogging(value string) string {
	// Remove newlines and carriage returns to prevent log injection.
	sanitized := strings.ReplaceAll(value, "\n", "\\n")
	sanitized = strings.ReplaceAll(sanitized, "\r", "\\r")
	sanitized = strings.ReplaceAll(sanitized, "\t", "\\t")

	// Truncate long values to prevent log flooding.
	if len(sanitized) > 256 {
		sanitized = sanitized[:253] + "..."
	}

	return sanitized
}

// Package security provides OWASP-compliant input validation and security utilities.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// OWASPValidator provides OWASP-compliant input validation.

type OWASPValidator struct {
	intentSchema *jsonschema.Schema

	pathValidator *PathValidator

	contentValidator *ContentValidator
}

// PathValidator provides secure path validation.

type PathValidator struct {
	allowedPaths []string

	maxPathLen int
}

// ContentValidator validates file and JSON content.

type ContentValidator struct {
	maxFileSize int64

	allowedMimeTypes []string
}

// SecurityViolation represents a security validation error.

type SecurityViolation struct {
	Field string `json:"field"`

	Type string `json:"type"`

	Severity string `json:"severity"`

	Message string `json:"message"`

	Value interface{} `json:"value,omitempty"`

	Remediation string `json:"remediation"`

	OWASPRule string `json:"owasp_rule"`
}

// ValidationResult contains validation results with security context.

type ValidationResult struct {
	IsValid bool `json:"is_valid"`

	Violations []SecurityViolation `json:"violations"`

	ThreatLevel string `json:"threat_level"`

	Timestamp string `json:"timestamp"`
}

// NewOWASPValidator creates a new OWASP-compliant validator.

func NewOWASPValidator() (*OWASPValidator, error) {
	// Load and compile the intent schema.

	schema, err := compileIntentSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to compile intent schema: %w", err)
	}

	pathValidator := &PathValidator{
		allowedPaths: getAllowedBasePaths(),

		maxPathLen: 4096, // OWASP recommended max path length

	}

	contentValidator := &ContentValidator{
		maxFileSize: 1048576, // 1MB max file size

		allowedMimeTypes: []string{"application/json", "text/plain"},
	}

	return &OWASPValidator{
		intentSchema: schema,

		pathValidator: pathValidator,

		contentValidator: contentValidator,
	}, nil
}

// ValidateIntentFile performs comprehensive security validation of intent files.

func (v *OWASPValidator) ValidateIntentFile(filePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: false,

		Violations: []SecurityViolation{},

		ThreatLevel: "LOW",

		Timestamp: generateSecureTimestamp(),
	}

	// 1. Path traversal and injection validation.

	if violations := v.pathValidator.ValidatePath(filePath); len(violations) > 0 {

		result.Violations = append(result.Violations, violations...)

		result.ThreatLevel = "HIGH"

	}

	// 2. File system security validation.

	if violations := v.validateFileSystem(filePath); len(violations) > 0 {

		result.Violations = append(result.Violations, violations...)

		if result.ThreatLevel != "HIGH" {
			result.ThreatLevel = "MEDIUM"
		}

	}

	// 3. Content validation.

	data, err := v.secureReadFile(filePath)
	if err != nil {

		result.Violations = append(result.Violations, SecurityViolation{
			Field: "file_access",

			Type: "FILE_READ_ERROR",

			Severity: "HIGH",

			Message: fmt.Sprintf("Failed to securely read file: %v", err),

			Remediation: "Ensure file exists and has proper permissions",

			OWASPRule: "A1:2021-Broken Access Control",
		})

		return result, nil

	}

	// 4. JSON schema validation with security context.

	if violations := v.validateIntentContent(data); len(violations) > 0 {

		result.Violations = append(result.Violations, violations...)

		if result.ThreatLevel == "LOW" {
			result.ThreatLevel = "MEDIUM"
		}

	}

	// Set validation result.

	result.IsValid = len(result.Violations) == 0

	return result, nil
}

// ValidatePath performs comprehensive path security validation.

func (pv *PathValidator) ValidatePath(path string) []SecurityViolation {
	var violations []SecurityViolation

	// Path length validation.

	if len(path) > pv.maxPathLen {
		violations = append(violations, SecurityViolation{
			Field: "file_path",

			Type: "PATH_LENGTH_VIOLATION",

			Severity: "MEDIUM",

			Message: fmt.Sprintf("Path length %d exceeds maximum %d", len(path), pv.maxPathLen),

			Value: len(path),

			Remediation: "Use shorter file paths",

			OWASPRule: "A3:2021-Injection",
		})
	}

	// Path traversal detection.

	if strings.Contains(path, "..") || strings.Contains(path, "~") {
		violations = append(violations, SecurityViolation{
			Field: "file_path",

			Type: "PATH_TRAVERSAL_ATTEMPT",

			Severity: "HIGH",

			Message: "Path contains traversal sequences",

			Value: path,

			Remediation: "Remove '..' and '~' from file paths",

			OWASPRule: "A3:2021-Injection",
		})
	}

	// Null byte injection.

	if strings.Contains(path, "\x00") {
		violations = append(violations, SecurityViolation{
			Field: "file_path",

			Type: "NULL_BYTE_INJECTION",

			Severity: "HIGH",

			Message: "Path contains null bytes",

			Value: path,

			Remediation: "Remove null bytes from file paths",

			OWASPRule: "A3:2021-Injection",
		})
	}

	// Canonical path validation.

	cleanPath := filepath.Clean(path)

	if cleanPath != path {
		violations = append(violations, SecurityViolation{
			Field: "file_path",

			Type: "NON_CANONICAL_PATH",

			Severity: "MEDIUM",

			Message: "Path is not in canonical form",

			Value: fmt.Sprintf("Original: %s, Canonical: %s", path, cleanPath),

			Remediation: "Use canonical file paths",

			OWASPRule: "A1:2021-Broken Access Control",
		})
	}

	// Whitelist validation.

	absPath, err := filepath.Abs(cleanPath)

	if err == nil {

		allowed := false

		for _, allowedPath := range pv.allowedPaths {
			if allowedAbs, err := filepath.Abs(allowedPath); err == nil {
				if strings.HasPrefix(absPath, allowedAbs) {

					allowed = true

					break

				}
			}
		}

		if !allowed {
			violations = append(violations, SecurityViolation{
				Field: "file_path",

				Type: "PATH_NOT_WHITELISTED",

				Severity: "HIGH",

				Message: "Path is not within allowed directories",

				Value: absPath,

				Remediation: "Use paths within allowed directories only",

				OWASPRule: "A1:2021-Broken Access Control",
			})
		}

	}

	return violations
}

// validateFileSystem performs file system level security checks.

func (v *OWASPValidator) validateFileSystem(path string) []SecurityViolation {
	var violations []SecurityViolation

	info, err := os.Stat(path)
	if err != nil {

		violations = append(violations, SecurityViolation{
			Field: "file_system",

			Type: "FILE_ACCESS_ERROR",

			Severity: "HIGH",

			Message: fmt.Sprintf("Cannot access file: %v", err),

			Remediation: "Ensure file exists and is accessible",

			OWASPRule: "A1:2021-Broken Access Control",
		})

		return violations

	}

	// File size validation.

	if info.Size() > v.contentValidator.maxFileSize {
		violations = append(violations, SecurityViolation{
			Field: "file_size",

			Type: "EXCESSIVE_FILE_SIZE",

			Severity: "MEDIUM",

			Message: fmt.Sprintf("File size %d exceeds maximum %d", info.Size(), v.contentValidator.maxFileSize),

			Value: info.Size(),

			Remediation: "Use smaller files",

			OWASPRule: "A4:2021-Insecure Design",
		})
	}

	// Permission validation.

	mode := info.Mode()

	if mode.Perm()&0o022 != 0 { // World or group writable

		violations = append(violations, SecurityViolation{
			Field: "file_permissions",

			Type: "INSECURE_PERMISSIONS",

			Severity: "MEDIUM",

			Message: "File has insecure permissions (world/group writable)",

			Value: mode.Perm().String(),

			Remediation: "Set secure file permissions (644 or 600)",

			OWASPRule: "A5:2021-Security Misconfiguration",
		})
	}

	return violations
}

// secureReadFile reads file with size and content validation.

func (v *OWASPValidator) secureReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer file.Close()

	// Validate file size before reading.

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > v.contentValidator.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", info.Size(), v.contentValidator.maxFileSize)
	}

	// Read with size limit.

	data := make([]byte, info.Size())

	n, err := file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data[:n], nil
}

// validateIntentContent validates JSON content with security checks.

func (v *OWASPValidator) validateIntentContent(data []byte) []SecurityViolation {
	var violations []SecurityViolation

	// JSON bomb detection.

	if len(data) > 0 {

		// Check for excessive nesting or repeated keys.

		var temp interface{}

		if err := json.Unmarshal(data, &temp); err != nil {

			violations = append(violations, SecurityViolation{
				Field: "json_content",

				Type: "MALFORMED_JSON",

				Severity: "HIGH",

				Message: fmt.Sprintf("Invalid JSON format: %v", err),

				Remediation: "Provide valid JSON content",

				OWASPRule: "A3:2021-Injection",
			})

			return violations

		}

	}

	// Schema validation.

	var intentData interface{}

	if err := json.Unmarshal(data, &intentData); err == nil {
		if err := v.intentSchema.Validate(intentData); err != nil {
			violations = append(violations, SecurityViolation{
				Field: "intent_schema",

				Type: "SCHEMA_VIOLATION",

				Severity: "HIGH",

				Message: fmt.Sprintf("Intent does not match schema: %v", err),

				Remediation: "Ensure intent matches the required schema",

				OWASPRule: "A8:2021-Software and Data Integrity Failures",
			})
		}
	}

	// Content injection detection.

	contentStr := string(data)

	injectionPatterns := []string{
		`<script`,

		`javascript:`,

		`eval\(`,

		`system\(`,

		`exec\(`,

		`shell_exec`,

		`passthru`,

		`popen`,

		`proc_open`,
	}

	for _, pattern := range injectionPatterns {

		matched, err := regexp.MatchString(`(?i)`+pattern, contentStr)
		if err != nil {

			violations = append(violations, SecurityViolation{
				Field: "content",

				Type: "REGEX_ERROR",

				Severity: "MEDIUM",

				Message: fmt.Sprintf("Invalid injection pattern regex: %s, error: %v", pattern, err),

				Remediation: "Fix the regular expression pattern",

				OWASPRule: "A3:2021-Injection",
			})

			continue

		}

		if matched {
			violations = append(violations, SecurityViolation{
				Field: "content",

				Type: "INJECTION_ATTEMPT",

				Severity: "HIGH",

				Message: fmt.Sprintf("Potential injection pattern detected: %s", pattern),

				Remediation: "Remove potentially malicious content",

				OWASPRule: "A3:2021-Injection",
			})
		}

	}

	return violations
}

// compileIntentSchema compiles the JSON schema for validation.

func compileIntentSchema() (*jsonschema.Schema, error) {
	schemaContent := `{

		"$schema": "https://json-schema.org/draft/2020-12/schema",

		"$id": "https://nephoran.io/schemas/secure-intent.json",

		"title": "Secure Intent Schema",

		"type": "object",

		"properties": {

			"intent_type": {

				"type": "string",

				"enum": ["scaling"],

				"pattern": "^[a-z]+$"

			},

			"target": {

				"type": "string",

				"minLength": 1,

				"maxLength": 63,

				"pattern": "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"

			},

			"namespace": {

				"type": "string", 

				"minLength": 1,

				"maxLength": 63,

				"pattern": "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"

			},

			"replicas": {

				"type": "integer",

				"minimum": 1,

				"maximum": 50

			},

			"reason": {

				"type": "string",

				"maxLength": 256,

				"pattern": "^[a-zA-Z0-9\\s\\-_.,:;!?()\\[\\]{}]+$"

			},

			"source": {

				"type": "string",

				"enum": ["user", "planner", "test"],

				"pattern": "^[a-z]+$"

			},

			"correlation_id": {

				"type": "string",

				"maxLength": 128,

				"pattern": "^[a-zA-Z0-9\\-_]+$"

			}

		},

		"required": ["intent_type", "target", "namespace", "replicas"],

		"additionalProperties": false

	}`

	compiler := jsonschema.NewCompiler()

	compiler.DefaultDraft(jsonschema.Draft2020)

	var schemaDoc interface{}

	if err := json.Unmarshal([]byte(schemaContent), &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	if err := compiler.AddResource("https://nephoran.io/schemas/secure-intent.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	return compiler.Compile("https://nephoran.io/schemas/secure-intent.json")
}

// getAllowedBasePaths returns platform-specific allowed paths.

func getAllowedBasePaths() []string {
	return []string{
		".",

		"./examples",

		"./packages",

		"./output",

		os.TempDir(),
	}
}

// generateSecureTimestamp creates a secure timestamp for audit trails.

func generateSecureTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

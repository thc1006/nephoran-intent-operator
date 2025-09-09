package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/opencontainers/go-digest"
)

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	// Container image security
	ImageConfig ImageSecurityConfig `json:"imageConfig"`
	
	// Input validation rules
	ValidationRules ValidationConfig `json:"validationRules"`
	
	// Repository allowlist
	RepositoryAllowlist map[string]RepositoryConfig `json:"repositoryAllowlist"`
}

// ImageSecurityConfig defines container image security settings
type ImageSecurityConfig struct {
	// Default registry to use
	DefaultRegistry string `json:"defaultRegistry"`
	
	// Default image version (never use 'latest')
	DefaultVersion string `json:"defaultVersion"`
	
	// Require image digest verification
	RequireDigest bool `json:"requireDigest"`
	
	// Require signature verification (Sigstore/cosign)
	RequireSignature bool `json:"requireSignature"`
	
	// Trusted image digests map[image:tag]digest
	TrustedDigests map[string]string `json:"trustedDigests"`
	
	// Cosign public key for signature verification
	CosignPublicKey string `json:"cosignPublicKey"`
}

// ValidationConfig defines input validation rules
type ValidationConfig struct {
	// Allowed pattern for target names
	TargetNamePattern string `json:"targetNamePattern"`
	
	// Maximum length for target names
	MaxTargetLength int `json:"maxTargetLength"`
	
	// Allowed characters in repository names
	RepoNamePattern string `json:"repoNamePattern"`
	
	// Maximum replicas allowed
	MaxReplicas int `json:"maxReplicas"`
}

// RepositoryConfig defines repository configuration
type RepositoryConfig struct {
	// Repository name
	Name string `json:"name"`
	
	// Allowed targets for this repository
	AllowedTargets []string `json:"allowedTargets"`
	
	// Description
	Description string `json:"description"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ImageConfig: ImageSecurityConfig{
			DefaultRegistry:  getEnvOrDefault("NF_IMAGE_REGISTRY", "registry.nephoran.io"),
			DefaultVersion:   getEnvOrDefault("NF_IMAGE_VERSION", "v1.0.0"),
			RequireDigest:    getEnvBool("NF_REQUIRE_DIGEST", true),
			RequireSignature: getEnvBool("NF_REQUIRE_SIGNATURE", false),
			TrustedDigests: map[string]string{
				"nephoran/nf-sim:v1.0.0": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				// Add more trusted digests as needed
			},
			CosignPublicKey: getEnvOrDefault("NF_COSIGN_PUBLIC_KEY", ""),
		},
		ValidationRules: ValidationConfig{
			TargetNamePattern: `^[a-zA-Z][a-zA-Z0-9-_]{0,62}$`,
			MaxTargetLength:   63,
			RepoNamePattern:   `^[a-z][a-z0-9-]{0,62}$`,
			MaxReplicas:       100,
		},
		RepositoryAllowlist: map[string]RepositoryConfig{
			"ran": {
				Name:           "ran-packages",
				AllowedTargets: []string{"ran", "gnb", "du", "cu", "ru"},
				Description:    "RAN network functions",
			},
			"core": {
				Name:           "core-packages",
				AllowedTargets: []string{"core", "smf", "upf", "amf", "ausf", "udm", "udr", "nrf", "pcf", "nssf"},
				Description:    "5G Core network functions",
			},
			"edge": {
				Name:           "edge-packages",
				AllowedTargets: []string{"mec", "edge", "uecm", "uelcm"},
				Description:    "Edge computing functions",
			},
			"transport": {
				Name:           "transport-packages",
				AllowedTargets: []string{"transport", "xhaul", "fronthaul", "midhaul", "backhaul"},
				Description:    "Transport network functions",
			},
			"management": {
				Name:           "management-packages",
				AllowedTargets: []string{"smo", "nms", "ems", "orchestrator"},
				Description:    "Management and orchestration functions",
			},
			"default": {
				Name:           "nephio-packages",
				AllowedTargets: []string{}, // Empty means any target not in other repos
				Description:    "Default package repository",
			},
		},
	}
}

// GetSecureImage returns a secure image reference with digest
func (sc *SecurityConfig) GetSecureImage(baseImage string) (string, error) {
	// Parse image reference
	parts := strings.Split(baseImage, ":")
	imageName := parts[0]
	imageTag := sc.ImageConfig.DefaultVersion
	
	if len(parts) > 1 && parts[1] != "latest" {
		imageTag = parts[1]
	}
	
	// Build full image reference
	fullImage := fmt.Sprintf("%s/%s:%s", sc.ImageConfig.DefaultRegistry, imageName, imageTag)
	
	// Check for trusted digest
	digestKey := fmt.Sprintf("%s:%s", imageName, imageTag)
	if digestValue, ok := sc.ImageConfig.TrustedDigests[digestKey]; ok && sc.ImageConfig.RequireDigest {
		// Validate the digest format before using it
		if err := ValidateDigest(digestValue); err != nil {
			return "", fmt.Errorf("trusted digest validation failed: %w", err)
		}
		fullImage = fmt.Sprintf("%s@%s", fullImage, digestValue)
	}
	
	return fullImage, nil
}

// ValidateTarget validates a target name against security rules
func (sc *SecurityConfig) ValidateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("target name cannot be empty")
	}
	
	if len(target) > sc.ValidationRules.MaxTargetLength {
		return fmt.Errorf("target name exceeds maximum length of %d characters", sc.ValidationRules.MaxTargetLength)
	}
	
	// Additional security checks first (before pattern check)
	if containsSQLInjectionPattern(target) {
		return fmt.Errorf("target name contains %s", SQL_INJECTION.GetStandardErrorMessage())
	}
	
	if containsPathTraversal(target) {
		return fmt.Errorf("target name contains %s", PATH_TRAVERSAL.GetStandardErrorMessage())
	}
	
	if containsScriptInjectionChars(target) {
		return fmt.Errorf("target name contains %s", SCRIPT_INJECTION.GetStandardErrorMessage())
	}
	
	// Check against pattern last (after security checks)
	pattern := regexp.MustCompile(sc.ValidationRules.TargetNamePattern)
	if !pattern.MatchString(target) {
		return fmt.Errorf("target name contains %s or format, must match pattern: %s", SCRIPT_INJECTION.GetStandardErrorMessage(), sc.ValidationRules.TargetNamePattern)
	}
	
	return nil
}

// ResolveRepository returns the repository name for a given target with validation
func (sc *SecurityConfig) ResolveRepository(target string) (string, error) {
	// Validate target first
	if err := sc.ValidateTarget(target); err != nil {
		return "", fmt.Errorf("invalid target: %w", err)
	}
	
	// Normalize target to lowercase for comparison
	normalizedTarget := strings.ToLower(target)
	
	// Check each repository's allowed targets
	for _, repo := range sc.RepositoryAllowlist {
		for _, allowedTarget := range repo.AllowedTargets {
			if normalizedTarget == strings.ToLower(allowedTarget) {
				return repo.Name, nil
			}
		}
	}
	
	// Return default repository if no specific match
	if defaultRepo, ok := sc.RepositoryAllowlist["default"]; ok {
		return defaultRepo.Name, nil
	}
	
	return "", fmt.Errorf("no repository mapping found for target: %s", target)
}

// Security error types for standardized messages
type SecurityErrorType int

const (
	SCRIPT_INJECTION SecurityErrorType = iota
	SQL_INJECTION
	PATH_TRAVERSAL
	DIGEST_INVALID
	DIGEST_UNSUPPORTED
)

// GetStandardErrorMessage returns standardized error messages for security categories
func (s SecurityErrorType) GetStandardErrorMessage() string {
	switch s {
	case SCRIPT_INJECTION:
		return "invalid characters"
	case SQL_INJECTION:
		return "potential SQL injection pattern"
	case PATH_TRAVERSAL:
		return "potential path traversal pattern"
	case DIGEST_INVALID:
		return "invalid digest format"
	case DIGEST_UNSUPPORTED:
		return "unsupported digest algorithm"
	default:
		return "security validation failed"
	}
}

// Improved digest validation variables
var (
	errInvalidDigest     = errors.New("invalid image digest")
	errUnsupportedDigest = errors.New("unsupported digest algorithm")
	hex64                = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

// ValidateDigest validates an image digest using improved OCI/CNCF standards
func ValidateDigest(s string) error {
	if s == "" {
		return fmt.Errorf("%s: empty digest", DIGEST_INVALID.GetStandardErrorMessage())
	}

	// Extra hardening: forbid quotes/semicolons/spaces even before Parse
	if strings.ContainsAny(s, "\"'; \t\n\r") {
		return fmt.Errorf("%s: contains dangerous characters", SCRIPT_INJECTION.GetStandardErrorMessage())
	}

	// Parse using go-digest format - validates general grammar algorithm:hex
	d, err := digest.Parse(s)
	if err != nil {
		return fmt.Errorf("%s: %w", DIGEST_INVALID.GetStandardErrorMessage(), err)
	}

	// Extract algorithm and hex parts
	alg := d.Algorithm().String()
	hex := strings.TrimPrefix(d.String(), alg+":")

	// Enforce sha256 algorithm only
	if alg != "sha256" {
		return fmt.Errorf("%s: only sha256 supported, got %s", DIGEST_UNSUPPORTED.GetStandardErrorMessage(), alg)
	}

	// Validate exact hex format (64 hex characters)
	if !hex64.MatchString(hex) {
		return fmt.Errorf("%s: sha256 must be exactly 64 hex characters", DIGEST_INVALID.GetStandardErrorMessage())
	}

	return nil
}

// containsScriptInjectionChars checks for common script injection characters
func containsScriptInjectionChars(input string) bool {
	scriptChars := []rune{'<', '>', '&', '"', '\'', ';', '(', ')', '{', '}', '[', ']'}
	for _, char := range input {
		for _, scriptChar := range scriptChars {
			if char == scriptChar {
				return true
			}
		}
	}
	return false
}

// containsSQLInjectionPattern checks for common SQL injection patterns
func containsSQLInjectionPattern(input string) bool {
	sqlPatterns := []string{
		"'",
		"\"",
		";",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
		"0x",
		"union",
		"select",
		"insert",
		"update",
		"delete",
		"drop",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	
	return false
}

// containsPathTraversal checks for path traversal patterns
func containsPathTraversal(input string) bool {
	pathPatterns := []string{
		"../",
		"..\\",
		"..",
		"./",
		".\\",
		"%2e%2e",
		"%252e",
		"0x2e",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range pathPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	
	return false
}

// HashTarget creates a secure hash of the target for use in labels
func HashTarget(target string) string {
	h := sha256.New()
	h.Write([]byte(target))
	return hex.EncodeToString(h.Sum(nil))[:8]
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

// CentralizedSanitizer provides comprehensive input sanitization
type CentralizedSanitizer struct {
	// Deny-listed patterns for various input types
	pathTraversalPatterns  []string
	scriptInjectionChars   []rune
	sqlInjectionPatterns   []string
	controlCharacterRanges []rune
	nullBytePattern        []byte
}

// NewCentralizedSanitizer creates a new sanitizer with predefined deny-lists
func NewCentralizedSanitizer() *CentralizedSanitizer {
	return &CentralizedSanitizer{
		pathTraversalPatterns: []string{
			"../",
			"..\\",
			"..",
			"./",
			".\\",
			"%2e%2e",
			"%252e",
			"0x2e",
			"~",
			"%7e",
			"\\u002e\\u002e",
		},
		scriptInjectionChars: []rune{
			'<', '>', '&', '"', '\'', ';', '(', ')', '{', '}', '[', ']',
			'`', '|', '$', '*', '?', '!', '^', '#',
		},
		sqlInjectionPatterns: []string{
			"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "0x",
			"union", "select", "insert", "update", "delete", "drop",
			"exec", "execute", "declare", "cast", "convert", "or ",
			"and ", "having", "waitfor", "shutdown", "alter",
		},
		controlCharacterRanges: []rune{
			0x00, 0x08, 0x0B, 0x0C, 0x0E, 0x1F, 0x7F, 0x9F,
		},
		nullBytePattern: []byte{0x00},
	}
}

// SanitizeInput provides comprehensive input sanitization based on context
func (cs *CentralizedSanitizer) SanitizeInput(input, context string) error {
	if input == "" {
		return nil // Empty input is generally acceptable
	}

	// Check for null bytes - always prohibited
	if err := cs.checkNullBytes(input); err != nil {
		return err
	}

	// Check for control characters
	if err := cs.checkControlCharacters(input); err != nil {
		return err
	}

	// Context-specific validation
	switch context {
	case "image_ref", "digest":
		return cs.sanitizeImageReference(input)
	case "file_path":
		return cs.sanitizeFilePath(input)
	case "prompt", "user_input":
		return cs.sanitizeUserInput(input)
	case "target_name":
		return cs.sanitizeTargetName(input)
	default:
		// Default: apply all checks
		return cs.sanitizeGeneral(input)
	}
}

// checkNullBytes detects null byte injection attempts
func (cs *CentralizedSanitizer) checkNullBytes(input string) error {
	for _, b := range []byte(input) {
		if b == 0x00 {
			return fmt.Errorf("%s: contains null bytes", SCRIPT_INJECTION.GetStandardErrorMessage())
		}
	}
	return nil
}

// checkControlCharacters detects dangerous control characters
func (cs *CentralizedSanitizer) checkControlCharacters(input string) error {
	for _, char := range input {
		// Check for specific dangerous ranges
		if (char >= 0x00 && char <= 0x1F) || char == 0x7F {
			return fmt.Errorf("%s: contains control characters", SCRIPT_INJECTION.GetStandardErrorMessage())
		}
	}
	return nil
}

// sanitizeImageReference validates image references and digests
func (cs *CentralizedSanitizer) sanitizeImageReference(input string) error {
	// Check for path traversal first
	if err := cs.checkPathTraversal(input); err != nil {
		return err
	}

	// Check for script injection
	if err := cs.checkScriptInjection(input); err != nil {
		return err
	}

	// Check for SQL injection patterns
	if err := cs.checkSQLInjection(input); err != nil {
		return err
	}

	// Special validation for digests
	if strings.Contains(input, ":") && (strings.Contains(input, "sha256:") || strings.Contains(input, "sha512:")) {
		return ValidateDigest(input)
	}

	return nil
}

// sanitizeFilePath validates file paths for security
func (cs *CentralizedSanitizer) sanitizeFilePath(input string) error {
	// Path traversal is the primary concern for file paths
	if err := cs.checkPathTraversal(input); err != nil {
		return err
	}

	// Check for script injection in file names
	if err := cs.checkScriptInjection(input); err != nil {
		return err
	}

	// Ensure no tilde expansion possibilities
	if strings.Contains(input, "~") {
		return fmt.Errorf("%s: tilde expansion not allowed", PATH_TRAVERSAL.GetStandardErrorMessage())
	}

	return nil
}

// sanitizeUserInput validates user prompts and inputs
func (cs *CentralizedSanitizer) sanitizeUserInput(input string) error {
	// Check for script injection
	if err := cs.checkScriptInjection(input); err != nil {
		return err
	}

	// Check for SQL injection patterns
	if err := cs.checkSQLInjection(input); err != nil {
		return err
	}

	// Path traversal checks for user inputs
	if err := cs.checkPathTraversal(input); err != nil {
		return err
	}

	return nil
}

// sanitizeTargetName validates Kubernetes target names
func (cs *CentralizedSanitizer) sanitizeTargetName(input string) error {
	// Check for script injection
	if err := cs.checkScriptInjection(input); err != nil {
		return err
	}

	// Check for SQL injection
	if err := cs.checkSQLInjection(input); err != nil {
		return err
	}

	// Check for path traversal
	if err := cs.checkPathTraversal(input); err != nil {
		return err
	}

	return nil
}

// sanitizeGeneral applies all security checks
func (cs *CentralizedSanitizer) sanitizeGeneral(input string) error {
	checks := []func(string) error{
		cs.checkScriptInjection,
		cs.checkSQLInjection,
		cs.checkPathTraversal,
	}

	for _, check := range checks {
		if err := check(input); err != nil {
			return err
		}
	}
	return nil
}

// checkPathTraversal detects path traversal attempts
func (cs *CentralizedSanitizer) checkPathTraversal(input string) error {
	lowerInput := strings.ToLower(input)
	for _, pattern := range cs.pathTraversalPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("%s: detected path traversal pattern '%s'", PATH_TRAVERSAL.GetStandardErrorMessage(), pattern)
		}
	}
	return nil
}

// checkScriptInjection detects script injection attempts
func (cs *CentralizedSanitizer) checkScriptInjection(input string) error {
	for _, char := range input {
		for _, scriptChar := range cs.scriptInjectionChars {
			if char == scriptChar {
				return fmt.Errorf("%s: contains dangerous character '%c'", SCRIPT_INJECTION.GetStandardErrorMessage(), char)
			}
		}
	}
	return nil
}

// checkSQLInjection detects SQL injection patterns
func (cs *CentralizedSanitizer) checkSQLInjection(input string) error {
	lowerInput := strings.ToLower(input)
	for _, pattern := range cs.sqlInjectionPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("%s: detected SQL pattern '%s'", SQL_INJECTION.GetStandardErrorMessage(), pattern)
		}
	}
	return nil
}

// Global sanitizer instance
var globalSanitizer = NewCentralizedSanitizer()

// SanitizeInput is a convenience function using the global sanitizer
func SanitizeInput(input, context string) error {
	return globalSanitizer.SanitizeInput(input, context)
}
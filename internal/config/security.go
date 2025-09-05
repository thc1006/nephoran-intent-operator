package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
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
	if digest, ok := sc.ImageConfig.TrustedDigests[digestKey]; ok && sc.ImageConfig.RequireDigest {
		fullImage = fmt.Sprintf("%s@%s", fullImage, digest)
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
		return fmt.Errorf("target name contains potential SQL injection pattern")
	}
	
	if containsPathTraversal(target) {
		return fmt.Errorf("target name contains potential path traversal pattern")
	}
	
	// Check against pattern last (after security checks)
	pattern := regexp.MustCompile(sc.ValidationRules.TargetNamePattern)
	if !pattern.MatchString(target) {
		return fmt.Errorf("target name contains invalid characters or format, must match pattern: %s", sc.ValidationRules.TargetNamePattern)
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
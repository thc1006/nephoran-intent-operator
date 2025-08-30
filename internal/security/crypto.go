package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CryptoSecureIdentifier provides OWASP-compliant unique identifier generation
type CryptoSecureIdentifier struct {
	entropy *EntropySource
	hasher  *SecureHasher
	encoder *SafeEncoder
}

// EntropySource provides cryptographically secure random number generation
type EntropySource struct {
	reader io.Reader
}

// SecureHasher provides collision-resistant hashing
type SecureHasher struct {
	salt []byte
}

// SafeEncoder provides secure base encoding
type SafeEncoder struct {
	encoding *base32.Encoding
}

// NewCryptoSecureIdentifier creates a new secure identifier generator
func NewCryptoSecureIdentifier() (*CryptoSecureIdentifier, error) {
	entropy := &EntropySource{reader: rand.Reader}

	// Generate cryptographically secure salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	hasher := &SecureHasher{salt: salt}

	// Use base32 without padding for safe filename encoding
	encoder := &SafeEncoder{
		encoding: base32.StdEncoding.WithPadding(base32.NoPadding),
	}

	return &CryptoSecureIdentifier{
		entropy: entropy,
		hasher:  hasher,
		encoder: encoder,
	}, nil
}

// GenerateSecurePackageName creates a collision-resistant package name
// following OWASP secure coding practices
func (c *CryptoSecureIdentifier) GenerateSecurePackageName(target string) (string, error) {
	// Input validation and sanitization
	if err := validateKubernetesName(target); err != nil {
		return "", fmt.Errorf("invalid target name: %w", err)
	}

	// Generate UUID v4 for guaranteed uniqueness
	packageUUID := uuid.New()

	// Create high-precision timestamp
	now := time.Now().UTC()
	timestamp := now.Format("20060102-150405")
	nanoseconds := fmt.Sprintf("%09d", now.Nanosecond())

	// Generate additional entropy
	entropy := make([]byte, 16)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Create compound identifier with multiple entropy sources
	compound := fmt.Sprintf("%s-%s-%s-%s",
		target,
		timestamp,
		nanoseconds,
		packageUUID.String())

	// Hash for collision resistance and length normalization
	hash := sha256.Sum256(append(c.hasher.salt, []byte(compound)...))
	hashString := hex.EncodeToString(hash[:12]) // Use first 12 bytes for reasonable length

	// Combine readable timestamp with secure hash
	securePackageName := fmt.Sprintf("%s-scaling-patch-%s-%s",
		sanitizeTarget(target),
		timestamp,
		hashString)

	// Final validation
	if err := validatePackageName(securePackageName); err != nil {
		return "", fmt.Errorf("generated package name validation failed: %w", err)
	}

	return securePackageName, nil
}

// GenerateCollisionResistantTimestamp creates RFC3339 timestamp with maximum entropy
func (c *CryptoSecureIdentifier) GenerateCollisionResistantTimestamp() (string, error) {
	now := time.Now().UTC()

	// Add cryptographic randomness to the timestamp
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create compound timestamp with entropy
	compound := fmt.Sprintf("%s-%s",
		now.Format(time.RFC3339Nano),
		hex.EncodeToString(randomBytes))

	return compound, nil
}

// validateKubernetesName validates Kubernetes resource naming conventions
func validateKubernetesName(name string) error {
	if len(name) == 0 || len(name) > 63 {
		return fmt.Errorf("name length must be 1-63 characters")
	}

	// Must start and end with alphanumeric
	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return fmt.Errorf("name must start and end with alphanumeric character")
	}

	// Check each character for valid Kubernetes naming
	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return fmt.Errorf("name contains invalid character: %c", char)
		}
	}

	return nil
}

// sanitizeTarget removes potential security risks from target name
func sanitizeTarget(target string) string {
	// Convert to lowercase
	target = strings.ToLower(target)

	// Replace invalid characters with hyphens
	var sanitized strings.Builder
	for _, char := range target {
		if isAlphaNumeric(byte(char)) {
			sanitized.WriteRune(char)
		} else {
			sanitized.WriteRune('-')
		}
	}

	result := sanitized.String()

	// Remove consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim hyphens from ends
	result = strings.Trim(result, "-")

	// Ensure minimum length
	if len(result) == 0 {
		result = "sanitized-target"
	}

	return result
}

// validatePackageName validates the generated package name
func validatePackageName(name string) error {
	if len(name) == 0 || len(name) > 253 {
		return fmt.Errorf("package name length must be 1-253 characters")
	}

	// Additional security checks
	if strings.Contains(name, "..") {
		return fmt.Errorf("package name contains path traversal sequence")
	}

	if strings.ContainsAny(name, `<>:"/\|?*`) {
		return fmt.Errorf("package name contains invalid characters")
	}

	return nil
}

// isAlphaNumeric checks if byte is lowercase alphanumeric
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

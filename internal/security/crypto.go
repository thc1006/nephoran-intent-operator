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

// CryptoSecureIdentifier provides OWASP-compliant unique identifier generation.

type CryptoSecureIdentifier struct {
	entropy *EntropySource

	hasher *SecureHasher

	encoder *SafeEncoder
}

// EntropySource provides cryptographically secure random number generation.

type EntropySource struct {
	reader io.Reader
}

// SecureHasher provides collision-resistant hashing.

type SecureHasher struct {
	salt []byte
}

// SafeEncoder provides safe base32 encoding without padding.

type SafeEncoder struct {
	encoding *base32.Encoding
}

// NewCryptoSecureIdentifier creates a new secure identifier generator.

func NewCryptoSecureIdentifier() *CryptoSecureIdentifier {
	entropy := &EntropySource{reader: rand.Reader}

	// Generate a secure salt for hashing.

	salt := make([]byte, 32)

	if _, err := rand.Read(salt); err != nil {
		panic("Failed to generate secure salt: " + err.Error())
	}

	hasher := &SecureHasher{salt: salt}

	encoder := &SafeEncoder{encoding: base32.StdEncoding.WithPadding(base32.NoPadding)}

	return &CryptoSecureIdentifier{
		entropy: entropy,

		hasher: hasher,

		encoder: encoder,
	}
}

// GenerateSecureUUID creates a cryptographically secure UUID v4.

func (c *CryptoSecureIdentifier) GenerateSecureUUID() string {
	id := uuid.New()

	return id.String()
}

// GenerateSecureToken creates a secure token for authentication/authorization.

func (c *CryptoSecureIdentifier) GenerateSecureToken(length int) (string, error) {
	if length < 16 {
		return "", fmt.Errorf("token length must be at least 16 bytes for security")
	}

	bytes := make([]byte, length)

	if _, err := c.entropy.reader.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}

	// Hash the random bytes with salt for additional security.

	hashedBytes := c.hasher.hash(bytes)

	// Encode using safe base32 (URL-safe, no padding).

	token := c.encoder.encoding.EncodeToString(hashedBytes)

	return token, nil
}

// GenerateSessionID creates a secure session identifier.

func (c *CryptoSecureIdentifier) GenerateSessionID() (string, error) {
	// Create a 32-byte secure session ID.

	sessionBytes := make([]byte, 32)

	if _, err := c.entropy.reader.Read(sessionBytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Add timestamp to ensure uniqueness.

	timestamp := time.Now().UnixNano()

	timestampBytes := make([]byte, 8)

	for i := range 8 {
		timestampBytes[i] = byte(timestamp >> (8 * i))
	}

	// Combine session bytes and timestamp.

	combined := append(sessionBytes, timestampBytes...)

	// Hash the combined data.

	hashedSession := c.hasher.hash(combined)

	// Encode as hex for session IDs (more readable in logs).

	sessionID := hex.EncodeToString(hashedSession)

	return sessionID, nil
}

// GenerateAPIKey creates a secure API key with metadata.

func (c *CryptoSecureIdentifier) GenerateAPIKey(prefix string) (string, error) {
	// Generate 32 bytes of entropy for the key.

	keyBytes := make([]byte, 32)

	if _, err := c.entropy.reader.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Add prefix for key identification.

	if prefix == "" {
		prefix = "neph" // Default prefix for Nephoran
	}

	// Hash the key bytes.

	hashedKey := c.hasher.hash(keyBytes)

	// Encode as base32 for API keys.

	encodedKey := c.encoder.encoding.EncodeToString(hashedKey)

	// Format: prefix_encodedkey.

	apiKey := fmt.Sprintf("%s_%s", prefix, encodedKey)

	return apiKey, nil
}

// ValidateTokenFormat checks if a token has a valid format.

func (c *CryptoSecureIdentifier) ValidateTokenFormat(token string) bool {
	// Basic validation: minimum length and character set.

	if len(token) < 32 {
		return false
	}

	// Check for valid base32 characters (A-Z, 2-7).

	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	for _, char := range token {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

// ValidateSessionIDFormat checks if a session ID has a valid format.

func (c *CryptoSecureIdentifier) ValidateSessionIDFormat(sessionID string) bool {
	// Session IDs should be 64 hex characters (32 bytes * 2).

	if len(sessionID) != 64 {
		return false
	}

	// Check for valid hex characters.

	for _, char := range sessionID {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}

// ValidateAPIKeyFormat checks if an API key has a valid format.

func (c *CryptoSecureIdentifier) ValidateAPIKeyFormat(apiKey string) bool {
	// API key format: prefix_base32encodedkey.

	parts := strings.Split(apiKey, "_")

	if len(parts) != 2 {
		return false
	}

	prefix, key := parts[0], parts[1]

	// Validate prefix (should be alphanumeric, 3-10 chars).

	if len(prefix) < 3 || len(prefix) > 10 {
		return false
	}

	for _, char := range prefix {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
			return false
		}
	}

	// Validate key part (should be valid base32).

	if len(key) < 32 {
		return false
	}

	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

	for _, char := range key {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}

	return true
}

// hash creates a secure hash with salt.

func (h *SecureHasher) hash(data []byte) []byte {
	hasher := sha256.New()

	hasher.Write(h.salt)

	hasher.Write(data)

	return hasher.Sum(nil)
}

// GenerateSecurePackageName creates a secure, collision-resistant package name.

func (c *CryptoSecureIdentifier) GenerateSecurePackageName(target string) (string, error) {
	// Validate target name format for Kubernetes compatibility
	if err := c.validateTargetName(target); err != nil {
		return "", fmt.Errorf("invalid target name: %w", err)
	}

	// Generate a secure timestamp.

	timestamp, err := c.GenerateCollisionResistantTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to generate timestamp: %w", err)
	}

	// Create unique entropy.

	entropy := make([]byte, 8)

	if _, err := c.entropy.reader.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Hash target + timestamp + entropy for collision resistance.

	combined := fmt.Sprintf("%s-%s-%x", target, timestamp, entropy)

	hashedName := c.hasher.hash([]byte(combined))

	// Use first 8 bytes as suffix for readability.

	suffix := hex.EncodeToString(hashedName[:4])

	// Format: target-scaling-patch-timestamp-suffix.

	packageName := fmt.Sprintf("%s-scaling-patch-%s-%s", target, timestamp, suffix)

	return packageName, nil
}

// GenerateCollisionResistantTimestamp creates a timestamp with nanosecond precision.

func (c *CryptoSecureIdentifier) GenerateCollisionResistantTimestamp() (string, error) {
	now := time.Now().UTC()

	// Add some entropy to prevent collisions in rapid succession.

	entropy := make([]byte, 2)

	if _, err := c.entropy.reader.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy for timestamp: %w", err)
	}

	// Format: YYYYMMDD-HHMMSS-NNNN (where NNNN is entropy-based).

	timestamp := fmt.Sprintf("%s-%04x",

		now.Format("20060102-150405"),

		entropy)

	return timestamp, nil
}

// validateTargetName validates that the target name meets Kubernetes naming requirements
func (c *CryptoSecureIdentifier) validateTargetName(target string) error {
	if target == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	// Check for invalid characters (must be lowercase alphanumeric with hyphens)
	for _, char := range target {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return fmt.Errorf("target name contains invalid character: %c", char)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(target, "../") || strings.Contains(target, "./") || strings.Contains(target, "/") {
		return fmt.Errorf("target name contains path traversal sequences")
	}

	return nil
}

// RegenerateSalt creates a new salt for the hasher (should be done periodically).

func (h *SecureHasher) RegenerateSalt() error {
	newSalt := make([]byte, 32)

	if _, err := rand.Read(newSalt); err != nil {
		return fmt.Errorf("failed to regenerate salt: %w", err)
	}

	h.salt = newSalt

	return nil
}

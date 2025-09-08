// Package security provides security validation utilities
package security

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	distdig "github.com/docker/distribution/digest"
)

var (
	// ErrInvalidDigest indicates the digest format is invalid
	ErrInvalidDigest = errors.New("invalid image digest")
	
	// ErrUnsupportedDigest indicates the digest algorithm is not supported
	ErrUnsupportedDigest = errors.New("unsupported digest algorithm")
	
	// hex64 validates SHA256 hex string (exactly 64 hex characters)
	// SECURITY FIX: Added missing $ anchor to prevent partial matches
	hex64 = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

// ValidateDigest performs strict validation of container image digests.
// Only allows sha256 algorithm with exactly 64 hex characters.
// Prevents injection attacks by rejecting dangerous characters early.
func ValidateDigest(s string) error {
	// SECURITY: Early validation to prevent injection attacks
	// Check for dangerous characters before any parsing
	if strings.ContainsAny(s, "\"'; \t\n\r\x00") {
		return fmt.Errorf("%w: contains forbidden characters", ErrInvalidDigest)
	}
	
	// Additional check for control characters
	for _, r := range s {
		if r < 32 || r > 126 { // Non-printable ASCII
			return fmt.Errorf("%w: contains control characters", ErrInvalidDigest)
		}
	}
	
	// Parse using the distribution library for format validation
	d, err := distdig.Parse(s)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDigest, err)
	}
	
	// Strictly enforce SHA256 only
	alg := d.Algorithm().String()
	if alg != "sha256" {
		return fmt.Errorf("%w: got %s, want sha256", ErrUnsupportedDigest, alg)
	}
	
	// Extract and validate the hex portion
	hex := strings.TrimPrefix(d.String(), alg+":")
	if !hex64.MatchString(hex) {
		return fmt.Errorf("%w: invalid sha256 hex format (must be 64 characters)", ErrInvalidDigest)
	}
	
	// Additional validation: ensure the original string matches expected format
	expectedFormat := fmt.Sprintf("sha256:%s", strings.ToLower(hex))
	if strings.ToLower(s) != expectedFormat {
		return fmt.Errorf("%w: format mismatch", ErrInvalidDigest)
	}
	
	return nil
}

// NormalizeDigest validates and returns a normalized (lowercase) digest
func NormalizeDigest(s string) (string, error) {
	if err := ValidateDigest(s); err != nil {
		return "", err
	}
	
	// Parse again to get normalized form
	d, _ := distdig.Parse(s) // Already validated above
	return strings.ToLower(d.String()), nil
}
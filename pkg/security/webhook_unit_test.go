package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "test-secret-key"
	config := &IncidentConfig{
		WebhookSecret: secret,
	}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ir.Close()

	tests := []struct {
		name        string
		payload     []byte
		secret      string
		signature   string
		expected    bool
		description string
	}{
		{
			name:        "valid_signature",
			payload:     []byte("test payload"),
			secret:      secret,
			signature:   generateTestSignature([]byte("test payload"), secret),
			expected:    true,
			description: "Should accept valid HMAC signature",
		},
		{
			name:        "invalid_signature",
			payload:     []byte("test payload"),
			secret:      secret,
			signature:   "sha256=invalid123",
			expected:    false,
			description: "Should reject invalid signature",
		},
		{
			name:        "wrong_secret",
			payload:     []byte("test payload"),
			secret:      secret,
			signature:   generateTestSignature([]byte("test payload"), "wrong-secret"),
			expected:    false,
			description: "Should reject signature with wrong secret",
		},
		{
			name:        "no_prefix",
			payload:     []byte("test payload"),
			secret:      secret,
			signature:   "invalid-without-prefix",
			expected:    false,
			description: "Should reject signature without sha256= prefix",
		},
		{
			name:        "empty_payload",
			payload:     []byte(""),
			secret:      secret,
			signature:   generateTestSignature([]byte(""), secret),
			expected:    true,
			description: "Should handle empty payload correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ir.verifyWebhookSignature(tt.payload, tt.signature)
			if result != tt.expected {
				t.Errorf("verifyWebhookSignature() = %v, expected %v\nDescription: %s",
					result, tt.expected, tt.description)
			}
		})
	}
}

func TestVerifyWebhookSignatureWithoutSecret(t *testing.T) {
	config := &IncidentConfig{
		WebhookSecret: "", // Empty secret
	}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ir.Close()

	payload := []byte("test payload")
	signature := "sha256=anysignature"

	result := ir.verifyWebhookSignature(payload, signature)
	if result {
		t.Error("Expected signature verification to fail when webhook secret is empty")
	}
}

func generateTestSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	return "sha256=" + signature
}

// TestHMACConsistency ensures HMAC generation is consistent
func TestHMACConsistency(t *testing.T) {
	secret := "test-secret"
	payload := []byte("consistent test payload")

	// Generate signature multiple times
	sig1 := generateTestSignature(payload, secret)
	sig2 := generateTestSignature(payload, secret)
	sig3 := generateTestSignature(payload, secret)

	if sig1 != sig2 || sig2 != sig3 {
		t.Error("HMAC signature generation is not consistent")
	}

	// Verify all generated signatures are valid
	config := &IncidentConfig{WebhookSecret: secret}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		t.Fatal(err)
	}
	defer ir.Close()

	if !ir.verifyWebhookSignature(payload, sig1) {
		t.Error("Generated signature should be valid")
	}
}

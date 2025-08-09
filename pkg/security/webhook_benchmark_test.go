package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// BenchmarkHMACVerification benchmarks the HMAC signature verification performance
func BenchmarkHMACVerification(b *testing.B) {
	secret := "test-webhook-secret-key-12345"
	payload := []byte(`{
		"type": "security_alert",
		"alert": {
			"title": "Performance Test Alert",
			"description": "This is a test alert for performance benchmarking",
			"severity": "High",
			"category": "performance_test"
		}
	}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Create incident response for testing
	config := &IncidentConfig{
		WebhookSecret: secret,
	}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		b.Fatal(err)
	}
	defer ir.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := ir.verifyWebhookSignature(payload, signature)
		if !result {
			b.Fatal("Expected signature verification to succeed")
		}
	}
}

// BenchmarkHMACVerificationLargePayload benchmarks with large payloads
func BenchmarkHMACVerificationLargePayload(b *testing.B) {
	secret := "test-webhook-secret-key-12345"

	// Create a 1MB payload
	largePayload := make([]byte, 1024*1024)
	for i := range largePayload {
		largePayload[i] = byte(i & 0xFF)
	}

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(largePayload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Create incident response for testing
	config := &IncidentConfig{
		WebhookSecret: secret,
	}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		b.Fatal(err)
	}
	defer ir.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := ir.verifyWebhookSignature(largePayload, signature)
		if !result {
			b.Fatal("Expected signature verification to succeed")
		}
	}
}

// BenchmarkHMACVerificationInvalid benchmarks invalid signature rejection
func BenchmarkHMACVerificationInvalid(b *testing.B) {
	secret := "test-webhook-secret-key-12345"
	payload := []byte(`{"type": "security_alert", "alert": {"title": "test"}}`)
	invalidSignature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	// Create incident response for testing
	config := &IncidentConfig{
		WebhookSecret: secret,
	}
	ir, err := NewIncidentResponse(config)
	if err != nil {
		b.Fatal(err)
	}
	defer ir.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := ir.verifyWebhookSignature(payload, invalidSignature)
		if result {
			b.Fatal("Expected signature verification to fail")
		}
	}
}

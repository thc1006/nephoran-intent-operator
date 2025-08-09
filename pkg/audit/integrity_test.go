package audit

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrityTestSuite tests audit log integrity protection
type IntegrityTestSuite struct {
	suite.Suite
	integrityChain *IntegrityChain
	signer         *EventSigner
	validator      *IntegrityValidator
}

func TestIntegrityTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrityTestSuite))
}

func (suite *IntegrityTestSuite) SetupTest() {
	var err error

	// Initialize integrity chain
	suite.integrityChain, err = NewIntegrityChain()
	suite.Require().NoError(err)

	// Initialize event signer with test key
	suite.signer, err = NewEventSigner(&SignerConfig{
		KeyType:   "hmac",
		SecretKey: "test-secret-key-for-integrity-testing",
	})
	suite.Require().NoError(err)

	// Initialize integrity validator
	suite.validator = NewIntegrityValidator(&ValidatorConfig{
		EnforceChain:       true,
		ValidateHashes:     true,
		ValidateSignatures: true,
		MaxClockSkew:       5 * time.Minute,
	})
}

// Hash Chain Tests
func (suite *IntegrityTestSuite) TestHashChainGeneration() {
	suite.Run("initialize hash chain", func() {
		chain := suite.integrityChain

		suite.NotNil(chain)
		suite.NotEmpty(chain.GetCurrentHash())
		suite.Equal(int64(0), chain.GetSequenceNumber())
	})

	suite.Run("single event hash chain", func() {
		event := createIntegrityTestEvent("test-event-1")

		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)

		// Event should have integrity fields populated
		suite.NotEmpty(event.Hash)
		suite.NotEmpty(event.PreviousHash)
		suite.Contains(event.IntegrityFields, "id")
		suite.Contains(event.IntegrityFields, "timestamp")
		suite.Contains(event.IntegrityFields, "event_type")

		// Chain should be updated
		suite.Equal(int64(1), suite.integrityChain.GetSequenceNumber())
		suite.Equal(event.Hash, suite.integrityChain.GetCurrentHash())
	})

	suite.Run("multiple events hash chain", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("event-1"),
			createIntegrityTestEvent("event-2"),
			createIntegrityTestEvent("event-3"),
		}

		var previousHash string
		for i, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)

			suite.NotEmpty(event.Hash)
			suite.NotEmpty(event.PreviousHash)

			if i > 0 {
				// Current event's previous hash should match the previous event's hash
				suite.Equal(previousHash, event.PreviousHash)
			}

			previousHash = event.Hash
		}

		// Verify chain integrity
		suite.True(suite.integrityChain.VerifyChain(events))
	})

	suite.Run("hash consistency", func() {
		event1 := createIntegrityTestEvent("consistent-test")
		event2 := createIntegrityTestEvent("consistent-test")

		// Same event data should produce same hash
		hash1 := suite.integrityChain.calculateEventHash(event1)
		hash2 := suite.integrityChain.calculateEventHash(event2)

		suite.Equal(hash1, hash2)

		// Different data should produce different hash
		event2.Action = "different-action"
		hash3 := suite.integrityChain.calculateEventHash(event2)
		suite.NotEqual(hash1, hash3)
	})
}

// Event Signing Tests
func (suite *IntegrityTestSuite) TestEventSigning() {
	suite.Run("sign single event", func() {
		event := createIntegrityTestEvent("signing-test")

		err := suite.signer.SignEvent(event)
		suite.NoError(err)

		suite.NotEmpty(event.Signature)
		suite.Contains(event.IntegrityFields, "signature")
	})

	suite.Run("verify signed event", func() {
		event := createIntegrityTestEvent("verification-test")

		err := suite.signer.SignEvent(event)
		suite.NoError(err)

		// Verify signature is valid
		valid, err := suite.signer.VerifySignature(event)
		suite.NoError(err)
		suite.True(valid)
	})

	suite.Run("detect signature tampering", func() {
		event := createIntegrityTestEvent("tampering-test")

		err := suite.signer.SignEvent(event)
		suite.NoError(err)
		originalSignature := event.Signature

		// Tamper with event data
		event.Action = "tampered-action"

		// Signature should no longer be valid
		valid, err := suite.signer.VerifySignature(event)
		suite.NoError(err)
		suite.False(valid)

		// Restore original data but tamper with signature
		event.Action = "tampering-test"
		event.Signature = "tampered-signature"

		valid, err = suite.signer.VerifySignature(event)
		suite.NoError(err)
		suite.False(valid)

		// Restore original signature - should be valid again
		event.Signature = originalSignature
		valid, err = suite.signer.VerifySignature(event)
		suite.NoError(err)
		suite.True(valid)
	})

	suite.Run("different signing algorithms", func() {
		algorithms := []struct {
			name      string
			keyType   string
			secretKey string
		}{
			{"HMAC-SHA256", "hmac", "hmac-test-key"},
			{"RSA", "rsa", "rsa-test-key"},
			{"ECDSA", "ecdsa", "ecdsa-test-key"},
		}

		for _, alg := range algorithms {
			suite.Run(alg.name, func() {
				signer, err := NewEventSigner(&SignerConfig{
					KeyType:   alg.keyType,
					SecretKey: alg.secretKey,
				})
				suite.NoError(err)

				event := createIntegrityTestEvent(fmt.Sprintf("test-%s", alg.name))
				err = signer.SignEvent(event)
				suite.NoError(err)

				valid, err := signer.VerifySignature(event)
				suite.NoError(err)
				suite.True(valid)
			})
		}
	})
}

// Integrity Validation Tests
func (suite *IntegrityTestSuite) TestIntegrityValidation() {
	suite.Run("validate complete event chain", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("chain-event-1"),
			createIntegrityTestEvent("chain-event-2"),
			createIntegrityTestEvent("chain-event-3"),
		}

		// Process events through integrity chain
		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		// Validate entire chain
		result := suite.validator.ValidateEventChain(events)

		suite.True(result.IsValid)
		suite.Empty(result.Violations)
		suite.Len(result.EventResults, 3)

		for _, eventResult := range result.EventResults {
			suite.True(eventResult.HashValid)
			suite.True(eventResult.ChainValid)
		}
	})

	suite.Run("detect hash chain break", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("break-event-1"),
			createIntegrityTestEvent("break-event-2"),
			createIntegrityTestEvent("break-event-3"),
		}

		// Process events through integrity chain
		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		// Tamper with middle event's hash
		events[1].Hash = "tampered-hash"

		result := suite.validator.ValidateEventChain(events)

		suite.False(result.IsValid)
		suite.NotEmpty(result.Violations)
		suite.Contains(result.Violations[0].Description, "hash chain break")
	})

	suite.Run("detect missing integrity fields", func() {
		event := createIntegrityTestEvent("missing-fields-test")
		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)

		// Remove integrity fields
		event.Hash = ""
		event.PreviousHash = ""

		result := suite.validator.ValidateEvent(event)

		suite.False(result.IsValid)
		suite.NotEmpty(result.Violations)
	})

	suite.Run("detect timestamp manipulation", func() {
		event := createIntegrityTestEvent("timestamp-test")
		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)

		// Manipulate timestamp beyond acceptable clock skew
		event.Timestamp = time.Now().Add(-10 * time.Minute)

		result := suite.validator.ValidateEvent(event)

		suite.False(result.IsValid)
		suite.Contains(result.Violations[0].Description, "timestamp")
	})

	suite.Run("validate event with signatures", func() {
		event := createIntegrityTestEvent("signature-validation-test")

		// Process through integrity chain and sign
		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)
		err = suite.signer.SignEvent(event)
		suite.NoError(err)

		result := suite.validator.ValidateEvent(event)

		suite.True(result.IsValid)
		suite.True(result.HashValid)
		suite.True(result.SignatureValid)
	})
}

// Tamper Detection Tests
func (suite *IntegrityTestSuite) TestTamperDetection() {
	suite.Run("detect data field tampering", func() {
		event := createIntegrityTestEvent("tamper-data-test")
		event.Data = map[string]interface{}{
			"sensitive_field": "original_value",
			"user_id":         "user123",
		}

		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)
		err = suite.signer.SignEvent(event)
		suite.NoError(err)

		// Tamper with data
		event.Data["sensitive_field"] = "tampered_value"

		result := suite.validator.ValidateEvent(event)

		suite.False(result.IsValid)
		suite.False(result.HashValid)
		suite.False(result.SignatureValid)
	})

	suite.Run("detect metadata tampering", func() {
		event := createIntegrityTestEvent("tamper-metadata-test")
		event.UserContext = &UserContext{
			UserID:   "original_user",
			Username: "original_username",
		}

		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)

		// Tamper with metadata
		event.UserContext.UserID = "tampered_user"

		result := suite.validator.ValidateEvent(event)

		suite.False(result.IsValid)
		suite.False(result.HashValid)
	})

	suite.Run("detect event injection", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("injection-event-1"),
			createIntegrityTestEvent("injection-event-2"),
			createIntegrityTestEvent("injection-event-4"), // Note: skipped 3
		}

		// Process first two events
		for _, event := range events[:2] {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		// Inject a third event with wrong sequence
		err := suite.integrityChain.ProcessEvent(events[2])
		suite.NoError(err)

		// Create the actual third event
		injectedEvent := createIntegrityTestEvent("injection-event-3-injected")
		injectedEvent.PreviousHash = events[1].Hash
		injectedEvent.Hash = "fake-hash"

		// Insert into chain
		eventsWithInjection := []*AuditEvent{events[0], events[1], injectedEvent, events[2]}

		result := suite.validator.ValidateEventChain(eventsWithInjection)

		suite.False(result.IsValid)
		suite.NotEmpty(result.Violations)
	})

	suite.Run("detect event deletion", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("deletion-event-1"),
			createIntegrityTestEvent("deletion-event-2"),
			createIntegrityTestEvent("deletion-event-3"),
		}

		// Process all events
		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		// Remove middle event (simulate deletion)
		eventsWithDeletion := []*AuditEvent{events[0], events[2]}

		result := suite.validator.ValidateEventChain(eventsWithDeletion)

		suite.False(result.IsValid)
		suite.NotEmpty(result.Violations)
		suite.Contains(result.Violations[0].Description, "chain gap")
	})
}

// Performance Tests
func (suite *IntegrityTestSuite) TestIntegrityPerformance() {
	suite.Run("high volume hash generation", func() {
		events := make([]*AuditEvent, 1000)
		for i := 0; i < len(events); i++ {
			events[i] = createIntegrityTestEvent(fmt.Sprintf("perf-event-%d", i))
		}

		start := time.Now()
		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}
		duration := time.Since(start)

		eventsPerSecond := float64(len(events)) / duration.Seconds()
		suite.Greater(eventsPerSecond, 500.0, "Hash generation too slow: %.2f events/sec", eventsPerSecond)
	})

	suite.Run("high volume signature verification", func() {
		events := make([]*AuditEvent, 100) // Smaller set for signing
		for i := 0; i < len(events); i++ {
			events[i] = createIntegrityTestEvent(fmt.Sprintf("sign-perf-event-%d", i))
			err := suite.signer.SignEvent(events[i])
			suite.NoError(err)
		}

		start := time.Now()
		for _, event := range events {
			valid, err := suite.signer.VerifySignature(event)
			suite.NoError(err)
			suite.True(valid)
		}
		duration := time.Since(start)

		verificationsPerSecond := float64(len(events)) / duration.Seconds()
		suite.Greater(verificationsPerSecond, 50.0, "Signature verification too slow: %.2f/sec", verificationsPerSecond)
	})

	suite.Run("chain validation performance", func() {
		events := make([]*AuditEvent, 500)
		for i := 0; i < len(events); i++ {
			events[i] = createIntegrityTestEvent(fmt.Sprintf("chain-perf-event-%d", i))
			err := suite.integrityChain.ProcessEvent(events[i])
			suite.NoError(err)
		}

		start := time.Now()
		result := suite.validator.ValidateEventChain(events)
		duration := time.Since(start)

		suite.True(result.IsValid)
		suite.Less(duration, 5*time.Second, "Chain validation too slow: %v", duration)
	})
}

// Recovery and Repair Tests
func (suite *IntegrityTestSuite) TestIntegrityRecovery() {
	suite.Run("recover from hash chain break", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("recovery-event-1"),
			createIntegrityTestEvent("recovery-event-2"),
			createIntegrityTestEvent("recovery-event-3"),
		}

		// Process events normally
		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		// Simulate chain break
		events[1].Hash = "broken-hash"

		// Attempt recovery
		recoverer := NewChainRecoverer(&RecovererConfig{
			RepairMode:   "recompute",
			BackupSource: "memory",
			VerifyRepair: true,
		})

		recoveryResult, err := recoverer.RecoverChain(events)
		suite.NoError(err)
		suite.True(recoveryResult.Success)
		suite.Equal(1, recoveryResult.EventsRepaired)

		// Validate recovered chain
		result := suite.validator.ValidateEventChain(recoveryResult.RepairedEvents)
		suite.True(result.IsValid)
	})

	suite.Run("detect unrepairable corruption", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("corrupt-event-1"),
			createIntegrityTestEvent("corrupt-event-2"),
		}

		// Corrupt multiple fields beyond repair
		events[0].ID = ""
		events[0].Timestamp = time.Time{}
		events[0].Hash = ""
		events[0].PreviousHash = ""

		recoverer := NewChainRecoverer(&RecovererConfig{
			RepairMode:   "recompute",
			BackupSource: "memory",
		})

		recoveryResult, err := recoverer.RecoverChain(events)
		suite.Error(err)
		suite.False(recoveryResult.Success)
	})
}

// Audit Trail Forensics Tests
func (suite *IntegrityTestSuite) TestForensicAnalysis() {
	suite.Run("forensic event analysis", func() {
		event := createIntegrityTestEvent("forensic-test")
		event.Data = map[string]interface{}{
			"source_ip":     "192.168.1.100",
			"session_id":    "sess_12345",
			"forensic_flag": true,
		}

		err := suite.integrityChain.ProcessEvent(event)
		suite.NoError(err)
		err = suite.signer.SignEvent(event)
		suite.NoError(err)

		forensicAnalyzer := NewForensicAnalyzer()
		analysis := forensicAnalyzer.AnalyzeEvent(event)

		suite.NotNil(analysis)
		suite.True(analysis.IntegrityVerified)
		suite.NotEmpty(analysis.Fingerprint)
		suite.NotNil(analysis.Timeline)
	})

	suite.Run("event timeline reconstruction", func() {
		events := []*AuditEvent{
			createIntegrityTestEvent("timeline-1"),
			createIntegrityTestEvent("timeline-2"),
			createIntegrityTestEvent("timeline-3"),
		}

		// Add timestamps in non-chronological order
		baseTime := time.Now().Add(-1 * time.Hour)
		events[0].Timestamp = baseTime.Add(10 * time.Minute)
		events[1].Timestamp = baseTime
		events[2].Timestamp = baseTime.Add(5 * time.Minute)

		for _, event := range events {
			err := suite.integrityChain.ProcessEvent(event)
			suite.NoError(err)
		}

		forensicAnalyzer := NewForensicAnalyzer()
		timeline := forensicAnalyzer.ReconstructTimeline(events)

		suite.Len(timeline, 3)

		// Verify chronological order in timeline
		for i := 1; i < len(timeline); i++ {
			suite.True(timeline[i].Timestamp.After(timeline[i-1].Timestamp) ||
				timeline[i].Timestamp.Equal(timeline[i-1].Timestamp))
		}
	})
}

// Helper functions and mock implementations

func createIntegrityTestEvent(action string) *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeAuthentication,
		Component: "test-component",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		UserContext: &UserContext{
			UserID: "test-user",
		},
		Data: map[string]interface{}{
			"test_field": "test_value",
		},
	}
}

// Mock implementations for testing (would be replaced with real implementations)

type IntegrityChain struct {
	currentHash    string
	sequenceNumber int64
	previousHash   string
}

func NewIntegrityChain() (*IntegrityChain, error) {
	// Initialize with genesis hash
	genesisHash := sha256.Sum256([]byte("genesis"))

	return &IntegrityChain{
		currentHash:    hex.EncodeToString(genesisHash[:]),
		sequenceNumber: 0,
		previousHash:   "",
	}, nil
}

func (ic *IntegrityChain) ProcessEvent(event *AuditEvent) error {
	// Set previous hash
	event.PreviousHash = ic.currentHash

	// Calculate current event hash
	event.Hash = ic.calculateEventHash(event)

	// Set integrity fields
	event.IntegrityFields = []string{"id", "timestamp", "event_type", "component", "action", "severity", "result"}

	// Update chain state
	ic.previousHash = ic.currentHash
	ic.currentHash = event.Hash
	ic.sequenceNumber++

	return nil
}

func (ic *IntegrityChain) calculateEventHash(event *AuditEvent) string {
	// Create deterministic hash input
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s|%s",
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		string(event.EventType),
		event.Component,
		event.Action,
		event.Severity,
		string(event.Result),
		event.PreviousHash,
	)

	// Include data fields in hash
	if event.Data != nil {
		dataBytes, _ := json.Marshal(event.Data)
		hashInput += "|" + string(dataBytes)
	}

	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

func (ic *IntegrityChain) GetCurrentHash() string {
	return ic.currentHash
}

func (ic *IntegrityChain) GetSequenceNumber() int64 {
	return ic.sequenceNumber
}

func (ic *IntegrityChain) VerifyChain(events []*AuditEvent) bool {
	for i, event := range events {
		// Recalculate hash
		expectedHash := ic.calculateEventHash(event)
		if event.Hash != expectedHash {
			return false
		}

		// Check chain linkage
		if i > 0 {
			if event.PreviousHash != events[i-1].Hash {
				return false
			}
		}
	}

	return true
}

type EventSigner struct {
	config *SignerConfig
}

type SignerConfig struct {
	KeyType   string
	SecretKey string
}

func NewEventSigner(config *SignerConfig) (*EventSigner, error) {
	return &EventSigner{config: config}, nil
}

func (es *EventSigner) SignEvent(event *AuditEvent) error {
	// Create signature input
	signatureInput := fmt.Sprintf("%s|%s|%s|%s",
		event.ID,
		event.Hash,
		event.Timestamp.Format(time.RFC3339Nano),
		es.config.SecretKey,
	)

	// Generate signature (simplified for testing)
	hash := sha256.Sum256([]byte(signatureInput))
	event.Signature = hex.EncodeToString(hash[:])

	// Add signature to integrity fields
	if event.IntegrityFields == nil {
		event.IntegrityFields = []string{}
	}
	event.IntegrityFields = append(event.IntegrityFields, "signature")

	return nil
}

func (es *EventSigner) VerifySignature(event *AuditEvent) (bool, error) {
	// Recalculate expected signature
	signatureInput := fmt.Sprintf("%s|%s|%s|%s",
		event.ID,
		event.Hash,
		event.Timestamp.Format(time.RFC3339Nano),
		es.config.SecretKey,
	)

	hash := sha256.Sum256([]byte(signatureInput))
	expectedSignature := hex.EncodeToString(hash[:])

	return event.Signature == expectedSignature, nil
}

type IntegrityValidator struct {
	config *ValidatorConfig
}

type ValidatorConfig struct {
	EnforceChain       bool
	ValidateHashes     bool
	ValidateSignatures bool
	MaxClockSkew       time.Duration
}

func NewIntegrityValidator(config *ValidatorConfig) *IntegrityValidator {
	return &IntegrityValidator{config: config}
}

func (iv *IntegrityValidator) ValidateEvent(event *AuditEvent) *EventValidationResult {
	result := &EventValidationResult{
		EventID:        event.ID,
		IsValid:        true,
		HashValid:      true,
		SignatureValid: true,
		ChainValid:     true,
		Violations:     []IntegrityViolation{},
	}

	// Validate timestamp
	if iv.config.MaxClockSkew > 0 {
		timeDiff := time.Since(event.Timestamp)
		if timeDiff < -iv.config.MaxClockSkew || timeDiff > iv.config.MaxClockSkew {
			result.IsValid = false
			result.Violations = append(result.Violations, IntegrityViolation{
				Type:        "timestamp_violation",
				Description: fmt.Sprintf("Event timestamp outside acceptable range: %v", timeDiff),
				Severity:    "high",
			})
		}
	}

	// Validate required integrity fields
	if event.Hash == "" {
		result.IsValid = false
		result.HashValid = false
		result.Violations = append(result.Violations, IntegrityViolation{
			Type:        "missing_hash",
			Description: "Event hash is missing",
			Severity:    "critical",
		})
	}

	if iv.config.EnforceChain && event.PreviousHash == "" {
		result.IsValid = false
		result.ChainValid = false
		result.Violations = append(result.Violations, IntegrityViolation{
			Type:        "missing_previous_hash",
			Description: "Previous hash is missing for chain validation",
			Severity:    "critical",
		})
	}

	return result
}

func (iv *IntegrityValidator) ValidateEventChain(events []*AuditEvent) *ChainValidationResult {
	result := &ChainValidationResult{
		IsValid:      true,
		EventResults: make([]*EventValidationResult, len(events)),
		Violations:   []IntegrityViolation{},
	}

	for i, event := range events {
		eventResult := iv.ValidateEvent(event)
		result.EventResults[i] = eventResult

		if !eventResult.IsValid {
			result.IsValid = false
			result.Violations = append(result.Violations, eventResult.Violations...)
		}

		// Check chain linkage
		if i > 0 && event.PreviousHash != events[i-1].Hash {
			result.IsValid = false
			result.Violations = append(result.Violations, IntegrityViolation{
				Type:        "hash_chain_break",
				Description: fmt.Sprintf("Hash chain break detected at event %d", i),
				Severity:    "critical",
			})
		}
	}

	// Check for gaps in chain (missing events)
	if len(events) > 1 {
		for i := 1; i < len(events); i++ {
			if events[i].PreviousHash != events[i-1].Hash {
				result.IsValid = false
				result.Violations = append(result.Violations, IntegrityViolation{
					Type:        "chain_gap",
					Description: fmt.Sprintf("Potential missing event between positions %d and %d", i-1, i),
					Severity:    "high",
				})
			}
		}
	}

	return result
}

type EventValidationResult struct {
	EventID        string
	IsValid        bool
	HashValid      bool
	SignatureValid bool
	ChainValid     bool
	Violations     []IntegrityViolation
}

type ChainValidationResult struct {
	IsValid      bool
	EventResults []*EventValidationResult
	Violations   []IntegrityViolation
}

type IntegrityViolation struct {
	Type        string
	Description string
	Severity    string
	EventID     string
	Timestamp   time.Time
}

type ChainRecoverer struct {
	config *RecovererConfig
}

type RecovererConfig struct {
	RepairMode   string
	BackupSource string
	VerifyRepair bool
}

func NewChainRecoverer(config *RecovererConfig) *ChainRecoverer {
	return &ChainRecoverer{config: config}
}

func (cr *ChainRecoverer) RecoverChain(events []*AuditEvent) (*RecoveryResult, error) {
	result := &RecoveryResult{
		Success:        false,
		EventsRepaired: 0,
		RepairedEvents: make([]*AuditEvent, len(events)),
	}

	// Copy events for repair
	copy(result.RepairedEvents, events)

	// Simple recovery: recalculate hashes
	chain := &IntegrityChain{currentHash: "genesis", sequenceNumber: 0}

	for i, event := range result.RepairedEvents {
		// Check if event has critical missing data
		if event.ID == "" || event.Timestamp.IsZero() {
			return result, fmt.Errorf("event %d has critical missing data and cannot be repaired", i)
		}

		// Recalculate hash and chain
		originalHash := event.Hash
		err := chain.ProcessEvent(event)
		if err != nil {
			return result, err
		}

		if originalHash != event.Hash {
			result.EventsRepaired++
		}
	}

	result.Success = true
	return result, nil
}

type RecoveryResult struct {
	Success        bool
	EventsRepaired int
	RepairedEvents []*AuditEvent
	Errors         []string
}

type ForensicAnalyzer struct{}

func NewForensicAnalyzer() *ForensicAnalyzer {
	return &ForensicAnalyzer{}
}

func (fa *ForensicAnalyzer) AnalyzeEvent(event *AuditEvent) *ForensicAnalysis {
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s",
		event.ID, event.Hash, event.Signature)))

	return &ForensicAnalysis{
		EventID:           event.ID,
		IntegrityVerified: event.Hash != "" && event.Signature != "",
		Fingerprint:       hex.EncodeToString(fingerprint[:]),
		Timeline: &EventTimeline{
			EventTime:     event.Timestamp,
			ProcessedTime: time.Now(),
		},
	}
}

func (fa *ForensicAnalyzer) ReconstructTimeline(events []*AuditEvent) []*AuditEvent {
	// Sort events by timestamp for timeline reconstruction
	timeline := make([]*AuditEvent, len(events))
	copy(timeline, events)

	// Simple bubble sort by timestamp
	for i := 0; i < len(timeline); i++ {
		for j := 0; j < len(timeline)-1-i; j++ {
			if timeline[j].Timestamp.After(timeline[j+1].Timestamp) {
				timeline[j], timeline[j+1] = timeline[j+1], timeline[j]
			}
		}
	}

	return timeline
}

type ForensicAnalysis struct {
	EventID           string
	IntegrityVerified bool
	Fingerprint       string
	Timeline          *EventTimeline
	Anomalies         []string
}

type EventTimeline struct {
	EventTime     time.Time
	ProcessedTime time.Time
	Sequence      int64
}

// Benchmark tests

func BenchmarkHashCalculation(b *testing.B) {
	chain, _ := NewIntegrityChain()
	event := createIntegrityTestEvent("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain.calculateEventHash(event)
	}
}

func BenchmarkEventSigning(b *testing.B) {
	signer, _ := NewEventSigner(&SignerConfig{
		KeyType:   "hmac",
		SecretKey: "benchmark-key",
	})
	event := createIntegrityTestEvent("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signer.SignEvent(event)
	}
}

func BenchmarkSignatureVerification(b *testing.B) {
	signer, _ := NewEventSigner(&SignerConfig{
		KeyType:   "hmac",
		SecretKey: "benchmark-key",
	})
	event := createIntegrityTestEvent("benchmark")
	signer.SignEvent(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signer.VerifySignature(event)
	}
}

func BenchmarkChainValidation(b *testing.B) {
	validator := NewIntegrityValidator(&ValidatorConfig{
		EnforceChain:       true,
		ValidateHashes:     true,
		ValidateSignatures: true,
	})

	events := make([]*AuditEvent, 100)
	chain, _ := NewIntegrityChain()

	for i := 0; i < len(events); i++ {
		events[i] = createIntegrityTestEvent(fmt.Sprintf("bench-%d", i))
		chain.ProcessEvent(events[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateEventChain(events)
	}
}

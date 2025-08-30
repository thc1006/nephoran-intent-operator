package audit

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (

	// IntegrityVersion defines the version of the integrity protection scheme.

	IntegrityVersion = "1.0"

	// HashAlgorithm defines the hash algorithm used for integrity protection.

	HashAlgorithm = "SHA256"

	// SignatureAlgorithm defines the signature algorithm.

	SignatureAlgorithm = "RSA-PSS"

	// MinKeySize defines the minimum RSA key size.

	MinKeySize = 2048

	// MaxChainLength defines maximum length of integrity chain to keep in memory.

	MaxChainLength = 10000
)

var (
	integrityOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{

		Name: "audit_integrity_operations_total",

		Help: "Total number of integrity operations",
	}, []string{"operation", "status"})

	integrityChainLength = promauto.NewGauge(prometheus.GaugeOpts{

		Name: "audit_integrity_chain_length",

		Help: "Current length of integrity chain",
	})

	integrityVerificationDuration = promauto.NewHistogram(prometheus.HistogramOpts{

		Name: "audit_integrity_verification_duration_seconds",

		Help: "Duration of integrity verification operations",
	})
)

// IntegrityChain maintains a cryptographic chain of audit events.

type IntegrityChain struct {
	mutex sync.RWMutex

	chain []IntegrityLink

	privateKey *rsa.PrivateKey

	publicKey *rsa.PublicKey

	logger logr.Logger

	keyID string

	lastHash string

	sequenceNum uint64

	enabled bool
}

// IntegrityLink represents a single link in the integrity chain.

type IntegrityLink struct {
	SequenceNumber uint64 `json:"sequence_number"`

	Timestamp time.Time `json:"timestamp"`

	EventID string `json:"event_id"`

	EventHash string `json:"event_hash"`

	PreviousHash string `json:"previous_hash"`

	ChainHash string `json:"chain_hash"`

	Signature string `json:"signature"`

	KeyID string `json:"key_id"`

	Version string `json:"version"`
}

// IntegrityConfig holds configuration for integrity protection.

type IntegrityConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	KeyPairPath string `json:"key_pair_path" yaml:"key_pair_path"`

	KeySize int `json:"key_size" yaml:"key_size"`

	AutoGenerateKeys bool `json:"auto_generate_keys" yaml:"auto_generate_keys"`

	ChainFile string `json:"chain_file" yaml:"chain_file"`

	VerificationMode string `json:"verification_mode" yaml:"verification_mode"`

	MaxChainLength int `json:"max_chain_length" yaml:"max_chain_length"`
}

// IntegrityReport contains the result of integrity verification.

type IntegrityReport struct {
	Valid bool `json:"valid"`

	TotalEvents int `json:"total_events"`

	VerifiedEvents int `json:"verified_events"`

	FailedEvents int `json:"failed_events"`

	MissingEvents []string `json:"missing_events"`

	TamperedEvents []string `json:"tampered_events"`

	ChainBreaks []IntegrityChainBreak `json:"chain_breaks"`

	VerificationTime time.Time `json:"verification_time"`

	VerificationDetails []IntegrityVerification `json:"verification_details"`
}

// IntegrityChainBreak represents a break in the integrity chain.

type IntegrityChainBreak struct {
	SequenceNumber uint64 `json:"sequence_number"`

	EventID string `json:"event_id"`

	Reason string `json:"reason"`

	Expected string `json:"expected"`

	Actual string `json:"actual"`
}

// IntegrityVerification contains details of a single event verification.

type IntegrityVerification struct {
	EventID string `json:"event_id"`

	SequenceNumber uint64 `json:"sequence_number"`

	Valid bool `json:"valid"`

	Error string `json:"error,omitempty"`
}

// NewIntegrityChain creates a new integrity chain.

func NewIntegrityChain() (*IntegrityChain, error) {

	config := DefaultIntegrityConfig()

	return NewIntegrityChainWithConfig(config)

}

// NewIntegrityChainWithConfig creates a new integrity chain with specific configuration.

func NewIntegrityChainWithConfig(config *IntegrityConfig) (*IntegrityChain, error) {

	if !config.Enabled {

		return &IntegrityChain{enabled: false}, nil

	}

	ic := &IntegrityChain{

		chain: make([]IntegrityLink, 0),

		logger: log.Log.WithName("integrity-chain"),

		enabled: true,

		sequenceNum: 0,
	}

	// Initialize cryptographic keys.

	if err := ic.initializeKeys(config); err != nil {

		return nil, fmt.Errorf("failed to initialize cryptographic keys: %w", err)

	}

	// Load existing chain if available.

	if config.ChainFile != "" {

		if err := ic.loadChain(config.ChainFile); err != nil {

			ic.logger.Error(err, "Failed to load existing chain", "file", config.ChainFile)

			// Continue with empty chain.

		}

	}

	ic.logger.Info("Integrity chain initialized",

		"key_id", ic.keyID,

		"chain_length", len(ic.chain),

		"last_sequence", ic.sequenceNum)

	return ic, nil

}

// ProcessEvent adds an audit event to the integrity chain.

func (ic *IntegrityChain) ProcessEvent(event *AuditEvent) error {

	if !ic.enabled {

		return nil

	}

	ic.mutex.Lock()

	defer ic.mutex.Unlock()

	// Calculate event hash.

	eventHash, err := ic.calculateEventHash(event)

	if err != nil {

		integrityOperationsTotal.WithLabelValues("hash_event", "error").Inc()

		return fmt.Errorf("failed to calculate event hash: %w", err)

	}

	// Create integrity link.

	ic.sequenceNum++

	link := IntegrityLink{

		SequenceNumber: ic.sequenceNum,

		Timestamp: time.Now().UTC(),

		EventID: event.ID,

		EventHash: eventHash,

		PreviousHash: ic.lastHash,

		Version: IntegrityVersion,

		KeyID: ic.keyID,
	}

	// Calculate chain hash.

	chainHash, err := ic.calculateChainHash(&link)

	if err != nil {

		integrityOperationsTotal.WithLabelValues("hash_chain", "error").Inc()

		return fmt.Errorf("failed to calculate chain hash: %w", err)

	}

	link.ChainHash = chainHash

	// Sign the link.

	signature, err := ic.signLink(&link)

	if err != nil {

		integrityOperationsTotal.WithLabelValues("sign_link", "error").Inc()

		return fmt.Errorf("failed to sign integrity link: %w", err)

	}

	link.Signature = signature

	// Add to chain.

	ic.chain = append(ic.chain, link)

	ic.lastHash = chainHash

	// Enforce maximum chain length.

	if len(ic.chain) > MaxChainLength {

		// Archive old entries (in production, these would be persisted).

		ic.chain = ic.chain[len(ic.chain)-MaxChainLength:]

	}

	// Update event with integrity information.

	event.Hash = eventHash

	event.PreviousHash = link.PreviousHash

	event.Signature = signature

	event.IntegrityFields = []string{"id", "timestamp", "event_type", "component", "action", "user_context", "result"}

	// Update metrics.

	integrityOperationsTotal.WithLabelValues("process_event", "success").Inc()

	integrityChainLength.Set(float64(len(ic.chain)))

	return nil

}

// VerifyEvent verifies the integrity of a single audit event.

func (ic *IntegrityChain) VerifyEvent(event *AuditEvent) error {

	if !ic.enabled {

		return nil

	}

	start := time.Now()

	defer func() {

		integrityVerificationDuration.Observe(time.Since(start).Seconds())

	}()

	// Find the corresponding link in the chain.

	ic.mutex.RLock()

	defer ic.mutex.RUnlock()

	var link *IntegrityLink

	for i := range ic.chain {

		if ic.chain[i].EventID == event.ID {

			link = &ic.chain[i]

			break

		}

	}

	if link == nil {

		integrityOperationsTotal.WithLabelValues("verify_event", "not_found").Inc()

		return fmt.Errorf("integrity link not found for event %s", event.ID)

	}

	// Verify event hash.

	expectedHash, err := ic.calculateEventHash(event)

	if err != nil {

		integrityOperationsTotal.WithLabelValues("verify_event", "hash_error").Inc()

		return fmt.Errorf("failed to calculate event hash: %w", err)

	}

	if expectedHash != link.EventHash {

		integrityOperationsTotal.WithLabelValues("verify_event", "hash_mismatch").Inc()

		return fmt.Errorf("event hash mismatch: expected %s, got %s", link.EventHash, expectedHash)

	}

	// Verify signature.

	if err := ic.verifyLinkSignature(link); err != nil {

		integrityOperationsTotal.WithLabelValues("verify_event", "signature_invalid").Inc()

		return fmt.Errorf("signature verification failed: %w", err)

	}

	integrityOperationsTotal.WithLabelValues("verify_event", "success").Inc()

	return nil

}

// VerifyChain verifies the integrity of the entire chain.

func (ic *IntegrityChain) VerifyChain() (*IntegrityReport, error) {

	if !ic.enabled {

		return &IntegrityReport{Valid: true}, nil

	}

	start := time.Now()

	defer func() {

		integrityVerificationDuration.Observe(time.Since(start).Seconds())

	}()

	ic.mutex.RLock()

	defer ic.mutex.RUnlock()

	report := &IntegrityReport{

		Valid: true,

		TotalEvents: len(ic.chain),

		VerificationTime: time.Now().UTC(),

		VerificationDetails: make([]IntegrityVerification, 0, len(ic.chain)),
	}

	var previousHash string

	for i, link := range ic.chain {

		verification := IntegrityVerification{

			EventID: link.EventID,

			SequenceNumber: link.SequenceNumber,

			Valid: true,
		}

		// Check sequence number.

		if link.SequenceNumber != uint64(i+1) {

			verification.Valid = false

			verification.Error = fmt.Sprintf("invalid sequence number: expected %d, got %d", i+1, link.SequenceNumber)

			report.Valid = false

			report.FailedEvents++

		}

		// Check previous hash.

		if link.PreviousHash != previousHash {

			verification.Valid = false

			verification.Error = fmt.Sprintf("chain break: expected previous hash %s, got %s", previousHash, link.PreviousHash)

			report.Valid = false

			report.FailedEvents++

			report.ChainBreaks = append(report.ChainBreaks, IntegrityChainBreak{

				SequenceNumber: link.SequenceNumber,

				EventID: link.EventID,

				Reason: "previous hash mismatch",

				Expected: previousHash,

				Actual: link.PreviousHash,
			})

		}

		// Verify signature.

		if err := ic.verifyLinkSignature(&link); err != nil {

			verification.Valid = false

			verification.Error = fmt.Sprintf("signature verification failed: %s", err.Error())

			report.Valid = false

			report.FailedEvents++

			report.TamperedEvents = append(report.TamperedEvents, link.EventID)

		}

		// Verify chain hash.

		expectedChainHash, err := ic.calculateChainHash(&link)

		if err != nil {

			verification.Valid = false

			verification.Error = fmt.Sprintf("failed to calculate chain hash: %s", err.Error())

		} else if expectedChainHash != link.ChainHash {

			verification.Valid = false

			verification.Error = fmt.Sprintf("chain hash mismatch: expected %s, got %s", expectedChainHash, link.ChainHash)

			report.Valid = false

			report.FailedEvents++

		}

		if verification.Valid {

			report.VerifiedEvents++

		}

		report.VerificationDetails = append(report.VerificationDetails, verification)

		previousHash = link.ChainHash

	}

	if report.Valid {

		integrityOperationsTotal.WithLabelValues("verify_chain", "success").Inc()

	} else {

		integrityOperationsTotal.WithLabelValues("verify_chain", "failed").Inc()

	}

	return report, nil

}

// GetChainInfo returns information about the integrity chain.

func (ic *IntegrityChain) GetChainInfo() map[string]interface{} {

	if !ic.enabled {

		return map[string]interface{}{

			"enabled": false,
		}

	}

	ic.mutex.RLock()

	defer ic.mutex.RUnlock()

	return map[string]interface{}{

		"enabled": true,

		"length": len(ic.chain),

		"last_sequence": ic.sequenceNum,

		"last_hash": ic.lastHash,

		"key_id": ic.keyID,

		"version": IntegrityVersion,

		"hash_algorithm": HashAlgorithm,

		"sign_algorithm": SignatureAlgorithm,
	}

}

// ExportChain exports the integrity chain for backup or transfer.

func (ic *IntegrityChain) ExportChain() ([]byte, error) {

	if !ic.enabled {

		return nil, fmt.Errorf("integrity chain is disabled")

	}

	ic.mutex.RLock()

	defer ic.mutex.RUnlock()

	export := struct {
		Version string `json:"version"`

		KeyID string `json:"key_id"`

		PublicKey string `json:"public_key"`

		Chain []IntegrityLink `json:"chain"`

		ExportTime time.Time `json:"export_time"`
	}{

		Version: IntegrityVersion,

		KeyID: ic.keyID,

		PublicKey: ic.exportPublicKey(),

		Chain: ic.chain,

		ExportTime: time.Now().UTC(),
	}

	return json.MarshalIndent(export, "", "  ")

}

// Helper methods.

func (ic *IntegrityChain) initializeKeys(config *IntegrityConfig) error {

	keySize := config.KeySize

	if keySize < MinKeySize {

		keySize = MinKeySize

	}

	if config.AutoGenerateKeys {

		// Generate new key pair.

		privateKey, err := rsa.GenerateKey(rand.Reader, keySize)

		if err != nil {

			return fmt.Errorf("failed to generate RSA key pair: %w", err)

		}

		ic.privateKey = privateKey

		ic.publicKey = &privateKey.PublicKey

		ic.keyID = ic.calculateKeyID(&privateKey.PublicKey)

	} else if config.KeyPairPath != "" {

		// Load existing key pair.

		if err := ic.loadKeyPair(config.KeyPairPath); err != nil {

			return fmt.Errorf("failed to load key pair: %w", err)

		}

	} else {

		return fmt.Errorf("either auto_generate_keys must be true or key_pair_path must be provided")

	}

	return nil

}

func (ic *IntegrityChain) calculateEventHash(event *AuditEvent) (string, error) {

	// Create a canonical representation of the event for hashing.

	hashData := struct {
		ID string `json:"id"`

		Timestamp time.Time `json:"timestamp"`

		EventType EventType `json:"event_type"`

		Component string `json:"component"`

		Action string `json:"action"`

		UserContext *UserContext `json:"user_context"`

		Result EventResult `json:"result"`

		Data map[string]interface{} `json:"data"`
	}{

		ID: event.ID,

		Timestamp: event.Timestamp,

		EventType: event.EventType,

		Component: event.Component,

		Action: event.Action,

		UserContext: event.UserContext,

		Result: event.Result,

		Data: event.Data,
	}

	// Sort data keys for deterministic hashing.

	if hashData.Data != nil {

		sortedData := make(map[string]interface{})

		keys := make([]string, 0, len(hashData.Data))

		for k := range hashData.Data {

			keys = append(keys, k)

		}

		sort.Strings(keys)

		for _, k := range keys {

			sortedData[k] = hashData.Data[k]

		}

		hashData.Data = sortedData

	}

	jsonBytes, err := json.Marshal(hashData)

	if err != nil {

		return "", fmt.Errorf("failed to marshal event for hashing: %w", err)

	}

	hash := sha256.Sum256(jsonBytes)

	return hex.EncodeToString(hash[:]), nil

}

func (ic *IntegrityChain) calculateChainHash(link *IntegrityLink) (string, error) {

	// Create canonical representation for chain hash.

	hashData := struct {
		SequenceNumber uint64 `json:"sequence_number"`

		EventID string `json:"event_id"`

		EventHash string `json:"event_hash"`

		PreviousHash string `json:"previous_hash"`

		KeyID string `json:"key_id"`
	}{

		SequenceNumber: link.SequenceNumber,

		EventID: link.EventID,

		EventHash: link.EventHash,

		PreviousHash: link.PreviousHash,

		KeyID: link.KeyID,
	}

	jsonBytes, err := json.Marshal(hashData)

	if err != nil {

		return "", fmt.Errorf("failed to marshal link for hashing: %w", err)

	}

	hash := sha256.Sum256(jsonBytes)

	return hex.EncodeToString(hash[:]), nil

}

func (ic *IntegrityChain) signLink(link *IntegrityLink) (string, error) {

	if ic.privateKey == nil {

		return "", fmt.Errorf("private key not available for signing")

	}

	// Create signature payload.

	payload := fmt.Sprintf("%d:%s:%s:%s", link.SequenceNumber, link.EventID, link.EventHash, link.ChainHash)

	payloadHash := sha256.Sum256([]byte(payload))

	// Sign using RSA-PSS.

	signature, err := rsa.SignPSS(rand.Reader, ic.privateKey, crypto.SHA256, payloadHash[:], nil)

	if err != nil {

		return "", fmt.Errorf("failed to sign link: %w", err)

	}

	return base64.StdEncoding.EncodeToString(signature), nil

}

func (ic *IntegrityChain) verifyLinkSignature(link *IntegrityLink) error {

	if ic.publicKey == nil {

		return fmt.Errorf("public key not available for verification")

	}

	// Decode signature.

	signature, err := base64.StdEncoding.DecodeString(link.Signature)

	if err != nil {

		return fmt.Errorf("failed to decode signature: %w", err)

	}

	// Create signature payload.

	payload := fmt.Sprintf("%d:%s:%s:%s", link.SequenceNumber, link.EventID, link.EventHash, link.ChainHash)

	payloadHash := sha256.Sum256([]byte(payload))

	// Verify using RSA-PSS.

	err = rsa.VerifyPSS(ic.publicKey, crypto.SHA256, payloadHash[:], signature, nil)

	if err != nil {

		return fmt.Errorf("signature verification failed: %w", err)

	}

	return nil

}

func (ic *IntegrityChain) calculateKeyID(publicKey *rsa.PublicKey) string {

	// Create key ID from public key hash.

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		// Return empty string on error, let caller handle appropriately
		return ""
	}

	hash := sha256.Sum256(publicKeyBytes)

	return hex.EncodeToString(hash[:])[:16] // Use first 16 characters

}

func (ic *IntegrityChain) exportPublicKey() string {

	if ic.publicKey == nil {

		return ""

	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(ic.publicKey)
	if err != nil {
		// Return empty string on error
		return ""
	}

	block := &pem.Block{

		Type: "PUBLIC KEY",

		Bytes: publicKeyBytes,
	}

	return string(pem.EncodeToMemory(block))

}

func (ic *IntegrityChain) loadKeyPair(keyPath string) error {

	// This is a placeholder implementation.

	// In practice, you would load keys from files or secure key management systems.

	return fmt.Errorf("key loading not implemented in this example")

}

func (ic *IntegrityChain) loadChain(chainFile string) error {

	// This is a placeholder implementation.

	// In practice, you would load the chain from a persistent store.

	return fmt.Errorf("chain loading not implemented in this example")

}

// DefaultIntegrityConfig returns a default integrity configuration.

func DefaultIntegrityConfig() *IntegrityConfig {

	return &IntegrityConfig{

		Enabled: true,

		KeySize: 2048,

		AutoGenerateKeys: true,

		VerificationMode: "strict",

		MaxChainLength: MaxChainLength,
	}

}

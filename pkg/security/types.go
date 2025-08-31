/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package security

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"
)

// Common errors.

var (

	// ErrKeyNotFound holds errkeynotfound value.

	ErrKeyNotFound = errors.New("key not found")
)

// KeyManager interface removed - using AdvancedKeyManager instead.

// StoredKey represents a stored cryptographic key.

type StoredKey struct {
	ID string `json:"id"`

	Type string `json:"type"` // "rsa", "ecdsa", etc.

	Bits int `json:"bits"` // Key size in bits

	PublicKey []byte `json:"publicKey"` // Public key bytes

	PrivateKey []byte `json:"privateKey"` // Encrypted private key bytes

	CreatedAt time.Time `json:"createdAt"`

	ExpiresAt time.Time `json:"expiresAt,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// RSAKey returns the RSA private key if this is an RSA key.

func (sk *StoredKey) RSAKey() (*rsa.PrivateKey, error) {

	// Implementation needed based on your crypto requirements.

	// This is a placeholder implementation.

	return nil, nil

}

// DefaultKeyManager provides a basic key manager implementation.

type DefaultKeyManager struct {
	keys map[string]*StoredKey
}

// NewDefaultKeyManager removed - use NewKeyManager from key_manager.go instead.

// GenerateKey generates a new key pair.

func (dkm *DefaultKeyManager) GenerateKey(keyType string, bits int) (*StoredKey, error) {

	// Placeholder implementation.

	key := &StoredKey{

		ID: generateKeyID(),

		Type: keyType,

		Bits: bits,

		CreatedAt: time.Now(),
	}

	return key, nil

}

// StoreKey stores a key securely.

func (dkm *DefaultKeyManager) StoreKey(key *StoredKey) error {

	dkm.keys[key.ID] = key

	return nil

}

// RetrieveKey retrieves a key by ID.

func (dkm *DefaultKeyManager) RetrieveKey(keyID string) (*StoredKey, error) {

	key, exists := dkm.keys[keyID]

	if !exists {

		return nil, ErrKeyNotFound

	}

	return key, nil

}

// RotateKey rotates an existing key.

func (dkm *DefaultKeyManager) RotateKey(keyID string) (*StoredKey, error) {

	oldKey, err := dkm.RetrieveKey(keyID)

	if err != nil {

		return nil, err

	}

	// Generate new key with same properties.

	newKey, err := dkm.GenerateKey(oldKey.Type, oldKey.Bits)

	if err != nil {

		return nil, err

	}

	// Store new key.

	err = dkm.StoreKey(newKey)

	if err != nil {

		return nil, err

	}

	return newKey, nil

}

// DeleteKey securely deletes a key.

func (dkm *DefaultKeyManager) DeleteKey(keyID string) error {

	delete(dkm.keys, keyID)

	return nil

}

// generateKeyID generates a unique key ID.

func generateKeyID() string {

	// Placeholder implementation - in production, use proper UUID generation.

	return "key-" + time.Now().Format("20060102150405")

}

// AdvancedKeyManager interface defines advanced key management operations.

type AdvancedKeyManager interface {

	// Basic key operations.

	GenerateKey(keyType string, bits int) (*StoredKey, error)

	StoreKey(key *StoredKey) error

	RetrieveKey(keyID string) (*StoredKey, error)

	RotateKey(keyID string) (*StoredKey, error)

	DeleteKey(keyID string) error

	// Advanced operations.

	GenerateMasterKey(keyType string, bits int) error

	DeriveKey(purpose string, version int) ([]byte, error)

	EscrowKey(keyID string, agents []EscrowAgent, threshold int) error

	SetupThresholdCrypto(keyID string, threshold, total int) error
}

// EscrowAgent represents a key escrow agent.

type EscrowAgent struct {
	ID string `json:"id"`

	Active bool `json:"active"`
}

// DetailedStoredKey represents a stored key with additional metadata.

type DetailedStoredKey struct {
	ID string `json:"id"`

	Version int `json:"version"`

	Key []byte `json:"key"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Created time.Time `json:"created"`

	Updated time.Time `json:"updated,omitempty"`
}

// KeyStore interface defines the storage backend for keys.

type KeyStore interface {
	Store(ctx context.Context, key *DetailedStoredKey) error

	Retrieve(ctx context.Context, keyID string) (*DetailedStoredKey, error)

	Delete(ctx context.Context, keyID string) error

	List(ctx context.Context) ([]*DetailedStoredKey, error)

	Rotate(ctx context.Context, keyID string, newKey *DetailedStoredKey) error
}

// DefaultAdvancedKeyManager implements AdvancedKeyManager.

type DefaultAdvancedKeyManager struct {
	keys map[string]*StoredKey

	store KeyStore

	master []byte
}

// NewKeyManager creates a new key manager with the given store.

func NewKeyManager(store KeyStore) AdvancedKeyManager {

	return &DefaultAdvancedKeyManager{

		keys: make(map[string]*StoredKey),

		store: store,
	}

}

// GenerateMasterKey generates a master key.

func (dkm *DefaultAdvancedKeyManager) GenerateMasterKey(keyType string, bits int) error {

	masterKey := make([]byte, bits)

	// In production, use crypto/rand.

	for i := range masterKey {

		masterKey[i] = byte(i % 256)

	}

	dkm.master = masterKey

	return nil

}

// DeriveKey derives a key for a specific purpose.

func (dkm *DefaultAdvancedKeyManager) DeriveKey(purpose string, version int) ([]byte, error) {

	if dkm.master == nil {

		return nil, fmt.Errorf("no master key available")

	}

	// Simple derivation for testing - in production use proper KDF.

	derived := make([]byte, 32)

	for i := range derived {

		derived[i] = dkm.master[i%len(dkm.master)] ^ byte(version)

	}

	return derived, nil

}

// EscrowKey implements key escrow.

func (dkm *DefaultAdvancedKeyManager) EscrowKey(keyID string, agents []EscrowAgent, threshold int) error {

	// Mock implementation for testing.

	if len(agents) < threshold {

		return fmt.Errorf("insufficient escrow agents")

	}

	return nil

}

// SetupThresholdCrypto sets up threshold cryptography.

func (dkm *DefaultAdvancedKeyManager) SetupThresholdCrypto(keyID string, threshold, total int) error {

	// Mock implementation for testing.

	if threshold > total {

		return fmt.Errorf("threshold cannot exceed total")

	}

	return nil

}

// Implement basic operations by delegating to DefaultKeyManager methods.

func (dkm *DefaultAdvancedKeyManager) GenerateKey(keyType string, bits int) (*StoredKey, error) {

	key := &StoredKey{

		ID: generateKeyID(),

		Type: keyType,

		Bits: bits,

		CreatedAt: time.Now(),
	}

	return key, nil

}

// StoreKey performs storekey operation.

func (dkm *DefaultAdvancedKeyManager) StoreKey(key *StoredKey) error {

	dkm.keys[key.ID] = key

	return nil

}

// RetrieveKey performs retrievekey operation.

func (dkm *DefaultAdvancedKeyManager) RetrieveKey(keyID string) (*StoredKey, error) {

	key, exists := dkm.keys[keyID]

	if !exists {

		return nil, ErrKeyNotFound

	}

	return key, nil

}

// RotateKey performs rotatekey operation.

func (dkm *DefaultAdvancedKeyManager) RotateKey(keyID string) (*StoredKey, error) {

	oldKey, err := dkm.RetrieveKey(keyID)

	if err != nil {

		return nil, err

	}

	// Generate new key with same properties.

	newKey, err := dkm.GenerateKey(oldKey.Type, oldKey.Bits)

	if err != nil {

		return nil, err

	}

	// Store new key.

	err = dkm.StoreKey(newKey)

	if err != nil {

		return nil, err

	}

	return newKey, nil

}

// DeleteKey performs deletekey operation.

func (dkm *DefaultAdvancedKeyManager) DeleteKey(keyID string) error {

	delete(dkm.keys, keyID)

	return nil

}

// Simple type aliases for backward compatibility.

type KeyManager = AdvancedKeyManager

// EncryptedSecret represents an encrypted secret with metadata following 2025 best practices
type EncryptedSecret struct {
	// Core identification and metadata
	ID      string `json:"id"`
	Name    string `json:"name"` // ADDED: Secret name field
	Type    string `json:"type"` // ADDED: Secret type field
	Version int    `json:"version"`

	// Encryption details
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"key_id"`
	KeyVersion int    `json:"key_version"` // For vault integration

	// Encrypted data and cryptographic components
	Data          []byte `json:"data,omitempty"` // Legacy field for compatibility
	IV            []byte `json:"iv,omitempty"`   // Legacy field for compatibility
	EncryptedData []byte `json:"encrypted_data"`
	Ciphertext    []byte `json:"ciphertext"` // ADDED: Ciphertext field
	Nonce         []byte `json:"nonce,omitempty"`
	Salt          []byte `json:"salt"` // ADDED: Salt field for key derivation

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Access tracking
	AccessCount  int64     `json:"access_count"`  // ADDED: Access count field
	LastAccessed time.Time `json:"last_accessed"` // ADDED: Last access time field

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SecretsBackend interface for secret storage implementations
type SecretsBackend interface {
	// Store stores an encrypted secret
	Store(ctx context.Context, key string, value *EncryptedSecret) error

	// Retrieve retrieves an encrypted secret
	Retrieve(ctx context.Context, key string) (*EncryptedSecret, error)

	// Delete deletes a secret
	Delete(ctx context.Context, key string) error

	// List lists all secret keys with optional prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Health checks backend health
	Health(ctx context.Context) error

	// Backup creates a backup of all secrets
	Backup(ctx context.Context) ([]byte, error)

	// Close closes the backend
	Close() error
}

// KeyVersion represents a versioned encryption key used in vault
type KeyVersion struct {
	Version   int       `json:"version"`
	Key       []byte    `json:"key"`
	Algorithm string    `json:"algorithm"` // ADDED: Algorithm field for key version
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

// VaultAuditEntry represents an audit log entry for vault operations
type VaultAuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Operation  string    `json:"operation"`
	Principal  string    `json:"principal"`   // User or service principal
	SecretName string    `json:"secret_name"` // ADDED: Secret name field
	User       string    `json:"user"`        // ADDED: User field for compatibility
	Resource   string    `json:"resource"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	IP         string    `json:"ip,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
}

// VaultStats represents statistics about vault operations following 2025 best practices
type VaultStats struct {
	// Health status
	VaultHealthy bool `json:"vault_healthy"`

	// Secret statistics
	TotalSecrets int64 `json:"total_secrets"`

	// Operation statistics
	TotalOperations      int64 `json:"total_operations"`
	SuccessfulOperations int64 `json:"successful_operations"`
	FailedOperations     int64 `json:"failed_operations"`

	// Key rotation statistics
	KeyRotations     int64     `json:"key_rotations"` // ADDED: Key rotation count
	LastRotation     time.Time `json:"last_rotation"` // ADDED: Last rotation time
	LastRotationTime time.Time `json:"last_rotation_time"`

	// Backup statistics
	BackupsCreated int64     `json:"backups_created"`
	LastBackup     time.Time `json:"last_backup"`
	LastBackupTime time.Time `json:"last_backup_time"`

	// System statistics
	CurrentKeyVersion   int     `json:"current_key_version"`
	ActiveConnections   int     `json:"active_connections"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	UptimeSeconds       int64   `json:"uptime_seconds"`
}

// SecretMetadata represents metadata associated with a secret following 2025 best practices
type SecretMetadata struct {
	// Core identification
	Name    string `json:"name"`
	Type    string `json:"type"`    // ADDED: Secret type field
	Version int    `json:"version"` // ADDED: Version field

	// Descriptive metadata
	Description string            `json:"description,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedBy   string            `json:"created_by,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Classification
	SecretType string `json:"secret_type"` // "tls", "password", "api_key", etc.
	Sensitive  bool   `json:"sensitive"`   // Whether this secret contains sensitive data

	// Extended metadata
	Metadata map[string]string `json:"metadata,omitempty"` // ADDED: Generic metadata field
}

// OPACompliancePolicyEngine represents Open Policy Agent compliance engine
type OPACompliancePolicyEngine struct {
	// Name of the OPA engine instance
	Name string `json:"name"`

	// Namespace where OPA is deployed
	Namespace string `json:"namespace"`

	// OPA server endpoint
	Endpoint string `json:"endpoint"`

	// Policy package name
	PolicyPackage string `json:"policyPackage"`

	// Policies loaded in the engine
	Policies []OPAPolicy `json:"policies,omitempty"`

	// Configuration for the OPA engine
	Config *OPAConfig `json:"config,omitempty"`

	// Status of the engine
	Status OPAEngineStatus `json:"status,omitempty"`
}

// OPAPolicy represents a single OPA policy
type OPAPolicy struct {
	// Name of the policy
	Name string `json:"name"`

	// Rego policy content
	Rego string `json:"rego"`

	// Package name for the policy
	Package string `json:"package"`

	// Version of the policy
	Version string `json:"version,omitempty"`

	// When the policy was created
	CreatedAt *time.Time `json:"createdAt,omitempty"`

	// When the policy was last updated
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// OPAConfig represents OPA configuration
type OPAConfig struct {
	// Bundles configuration
	Bundles map[string]OPABundle `json:"bundles,omitempty"`

	// Decision logs configuration
	DecisionLogs *OPADecisionLogsConfig `json:"decisionLogs,omitempty"`

	// Status configuration
	Status *OPAStatusConfig `json:"status,omitempty"`

	// Server configuration
	Server *OPAServerConfig `json:"server,omitempty"`
}

// OPABundle represents an OPA bundle configuration
type OPABundle struct {
	// Service name
	Service string `json:"service"`

	// Resource path
	Resource string `json:"resource,omitempty"`

	// Signing configuration
	Signing *OPABundleSigning `json:"signing,omitempty"`
}

// OPABundleSigning represents bundle signing configuration
type OPABundleSigning struct {
	// Public key for verification
	PublicKey string `json:"publicKey,omitempty"`

	// Key ID
	KeyID string `json:"keyId,omitempty"`

	// Exclude files from verification
	Exclude []string `json:"exclude,omitempty"`
}

// OPADecisionLogsConfig represents decision logs configuration
type OPADecisionLogsConfig struct {
	// Console logging
	Console bool `json:"console,omitempty"`

	// Service name for remote logging
	Service string `json:"service,omitempty"`

	// Reporting configuration
	Reporting *OPAReportingConfig `json:"reporting,omitempty"`
}

// OPAStatusConfig represents status reporting configuration
type OPAStatusConfig struct {
	// Service name for status reporting
	Service string `json:"service,omitempty"`

	// Trigger mode
	Trigger string `json:"trigger,omitempty"`
}

// OPAServerConfig represents OPA server configuration
type OPAServerConfig struct {
	// Encoding for server responses
	Encoding *OPAServerEncoding `json:"encoding,omitempty"`
}

// OPAServerEncoding represents server encoding configuration
type OPAServerEncoding struct {
	// GZIP compression
	GZIP *OPAGZIPConfig `json:"gzip,omitempty"`
}

// OPAGZIPConfig represents GZIP compression configuration
type OPAGZIPConfig struct {
	// Compression level
	Level int `json:"level,omitempty"`
}

// OPAReportingConfig represents reporting configuration
type OPAReportingConfig struct {
	// Minimum delay between reports
	MinDelaySeconds int `json:"minDelaySeconds,omitempty"`

	// Maximum delay between reports
	MaxDelaySeconds int `json:"maxDelaySeconds,omitempty"`

	// Upload size limit
	UploadSizeLimitBytes int64 `json:"uploadSizeLimitBytes,omitempty"`
}

// OPAEngineStatus represents the status of an OPA engine
type OPAEngineStatus struct {
	// Ready indicates if the engine is ready
	Ready bool `json:"ready"`

	// Healthy indicates if the engine is healthy
	Healthy bool `json:"healthy"`

	// Number of loaded policies
	PolicyCount int `json:"policyCount"`

	// Last evaluation timestamp
	LastEvaluation *time.Time `json:"lastEvaluation,omitempty"`

	// Error message if any
	Error string `json:"error,omitempty"`

	// Version of OPA
	Version string `json:"version,omitempty"`
}

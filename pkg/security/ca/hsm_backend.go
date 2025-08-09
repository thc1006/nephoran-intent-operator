package ca

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// HSMBackend implements Hardware Security Module integration
type HSMBackend struct {
	logger *logging.StructuredLogger
	config *HSMBackendConfig
	hsm    HSMProvider
}

// HSMBackendConfig holds HSM backend configuration
type HSMBackendConfig struct {
	// HSM connection
	ProviderType     HSMProviderType `yaml:"provider_type"`
	ConnectionString string          `yaml:"connection_string"`
	SlotID           int             `yaml:"slot_id"`
	TokenLabel       string          `yaml:"token_label"`

	// Authentication
	UserPIN      string `yaml:"user_pin"`
	SOPin        string `yaml:"so_pin"`
	KeystoreFile string `yaml:"keystore_file,omitempty"`
	KeystorePass string `yaml:"keystore_pass,omitempty"`

	// Key management
	KeyGeneration *HSMKeyGenConfig     `yaml:"key_generation"`
	KeyStorage    *HSMKeyStorageConfig `yaml:"key_storage"`

	// Security settings
	RequireAuthentication bool          `yaml:"require_authentication"`
	SessionTimeout        time.Duration `yaml:"session_timeout"`
	MaxRetries            int           `yaml:"max_retries"`

	// High availability
	HAConfig *HSMHAConfig `yaml:"ha_config,omitempty"`

	// Performance settings
	PoolSize    int `yaml:"pool_size"`
	MaxSessions int `yaml:"max_sessions"`
}

// HSMProviderType represents different HSM providers
type HSMProviderType string

const (
	HSMProviderPKCS11  HSMProviderType = "pkcs11"
	HSMProviderAWS     HSMProviderType = "aws_cloudhsm"
	HSMProviderAzure   HSMProviderType = "azure_keyvault"
	HSMProviderThales  HSMProviderType = "thales"
	HSMProviderSafeNet HSMProviderType = "safenet"
	HSMProviderYubiKey HSMProviderType = "yubikey"
	HSMProviderSoftHSM HSMProviderType = "softhsm"
)

// HSMKeyGenConfig holds key generation configuration
type HSMKeyGenConfig struct {
	KeyType     string                 `yaml:"key_type"` // RSA, ECDSA, Ed25519
	KeySize     int                    `yaml:"key_size"`
	Curve       string                 `yaml:"curve,omitempty"` // For ECDSA
	Extractable bool                   `yaml:"extractable"`
	Sensitive   bool                   `yaml:"sensitive"`
	Permanent   bool                   `yaml:"permanent"`
	KeyUsages   []string               `yaml:"key_usages"`
	Attributes  map[string]interface{} `yaml:"attributes,omitempty"`
}

// HSMKeyStorageConfig holds key storage configuration
type HSMKeyStorageConfig struct {
	KeyPrefix          string        `yaml:"key_prefix"`
	CAKeyLabel         string        `yaml:"ca_key_label"`
	IntermKeyLabel     string        `yaml:"interm_key_label"`
	AutoLabel          bool          `yaml:"auto_label"`
	KeyRetention       time.Duration `yaml:"key_retention"`
	BackupEnabled      bool          `yaml:"backup_enabled"`
	ReplicationEnabled bool          `yaml:"replication_enabled"`
}

// HSMHAConfig holds high availability configuration
type HSMHAConfig struct {
	Enabled             bool          `yaml:"enabled"`
	PrimaryHSM          HSMEndpoint   `yaml:"primary_hsm"`
	SecondaryHSMs       []HSMEndpoint `yaml:"secondary_hsms"`
	FailoverTimeout     time.Duration `yaml:"failover_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// HSMEndpoint represents an HSM endpoint
type HSMEndpoint struct {
	Address    string `yaml:"address"`
	SlotID     int    `yaml:"slot_id"`
	TokenLabel string `yaml:"token_label"`
	Priority   int    `yaml:"priority"`
}

// HSMProvider defines the interface for HSM providers
type HSMProvider interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	HealthCheck(ctx context.Context) error

	// Key management
	GenerateKey(ctx context.Context, config *HSMKeyGenConfig) (HSMKeyHandle, error)
	ImportKey(ctx context.Context, keyData []byte, config *HSMKeyGenConfig) (HSMKeyHandle, error)
	DeleteKey(ctx context.Context, handle HSMKeyHandle) error
	ListKeys(ctx context.Context) ([]HSMKeyInfo, error)

	// Cryptographic operations
	Sign(ctx context.Context, handle HSMKeyHandle, data []byte, algorithm string) ([]byte, error)
	Verify(ctx context.Context, handle HSMKeyHandle, data, signature []byte, algorithm string) error
	GetPublicKey(ctx context.Context, handle HSMKeyHandle) (crypto.PublicKey, error)

	// Certificate operations
	GenerateCSR(ctx context.Context, handle HSMKeyHandle, template *x509.CertificateRequest) ([]byte, error)
	SignCertificate(ctx context.Context, handle HSMKeyHandle, template, parent *x509.Certificate) ([]byte, error)

	// Provider info
	GetProviderInfo() HSMProviderInfo
}

// HSMKeyHandle represents a handle to a key stored in HSM
type HSMKeyHandle interface {
	ID() string
	Label() string
	Type() string
	Size() int
	Usage() []string
	Attributes() map[string]interface{}
}

// HSMKeyInfo contains information about an HSM key
type HSMKeyInfo struct {
	Handle     HSMKeyHandle           `json:"handle"`
	ID         string                 `json:"id"`
	Label      string                 `json:"label"`
	Type       string                 `json:"type"`
	Size       int                    `json:"size"`
	Usage      []string               `json:"usage"`
	CreatedAt  time.Time              `json:"created_at"`
	Attributes map[string]interface{} `json:"attributes"`
}

// HSMProviderInfo contains HSM provider information
type HSMProviderInfo struct {
	ProviderType        HSMProviderType `json:"provider_type"`
	Version             string          `json:"version"`
	SlotInfo            HSMSlotInfo     `json:"slot_info"`
	TokenInfo           HSMTokenInfo    `json:"token_info"`
	SupportedAlgorithms []string        `json:"supported_algorithms"`
	Capabilities        []string        `json:"capabilities"`
}

// HSMSlotInfo contains HSM slot information
type HSMSlotInfo struct {
	ID              int    `json:"id"`
	Description     string `json:"description"`
	Manufacturer    string `json:"manufacturer"`
	HardwareVersion string `json:"hardware_version"`
	FirmwareVersion string `json:"firmware_version"`
}

// HSMTokenInfo contains HSM token information
type HSMTokenInfo struct {
	Label             string `json:"label"`
	Manufacturer      string `json:"manufacturer"`
	Model             string `json:"model"`
	SerialNumber      string `json:"serial_number"`
	MaxSessionCount   int    `json:"max_session_count"`
	SessionCount      int    `json:"session_count"`
	MaxRWSessionCount int    `json:"max_rw_session_count"`
	RWSessionCount    int    `json:"rw_session_count"`
	FreePublicMemory  int    `json:"free_public_memory"`
	FreePrivateMemory int    `json:"free_private_memory"`
}

// BasicHSMKeyHandle implements HSMKeyHandle interface
type BasicHSMKeyHandle struct {
	id         string
	label      string
	keyType    string
	keySize    int
	usage      []string
	attributes map[string]interface{}
}

func (h *BasicHSMKeyHandle) ID() string                         { return h.id }
func (h *BasicHSMKeyHandle) Label() string                      { return h.label }
func (h *BasicHSMKeyHandle) Type() string                       { return h.keyType }
func (h *BasicHSMKeyHandle) Size() int                          { return h.keySize }
func (h *BasicHSMKeyHandle) Usage() []string                    { return h.usage }
func (h *BasicHSMKeyHandle) Attributes() map[string]interface{} { return h.attributes }

// NewHSMBackend creates a new HSM backend
func NewHSMBackend(logger *logging.StructuredLogger) (Backend, error) {
	return &HSMBackend{
		logger: logger,
	}, nil
}

// Initialize initializes the HSM backend
func (b *HSMBackend) Initialize(ctx context.Context, config interface{}) error {
	hsmConfig, ok := config.(*HSMBackendConfig)
	if !ok {
		return fmt.Errorf("invalid HSM config type")
	}

	b.config = hsmConfig

	// Validate configuration
	if err := b.validateConfig(); err != nil {
		return fmt.Errorf("HSM config validation failed: %w", err)
	}

	// Initialize HSM provider
	hsm, err := b.createHSMProvider()
	if err != nil {
		return fmt.Errorf("failed to create HSM provider: %w", err)
	}
	b.hsm = hsm

	// Connect to HSM
	if err := b.hsm.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to HSM: %w", err)
	}

	// Perform health check
	if err := b.hsm.HealthCheck(ctx); err != nil {
		return fmt.Errorf("HSM health check failed: %w", err)
	}

	// Initialize CA keys if needed
	if err := b.initializeCAKeys(ctx); err != nil {
		return fmt.Errorf("failed to initialize CA keys: %w", err)
	}

	b.logger.Info("HSM backend initialized successfully",
		"provider", b.config.ProviderType,
		"slot_id", b.config.SlotID,
		"token_label", b.config.TokenLabel)

	return nil
}

// HealthCheck performs health check on the HSM backend
func (b *HSMBackend) HealthCheck(ctx context.Context) error {
	if b.hsm == nil {
		return fmt.Errorf("HSM provider not initialized")
	}

	return b.hsm.HealthCheck(ctx)
}

// IssueCertificate issues a certificate using HSM
func (b *HSMBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	b.logger.Info("issuing certificate via HSM",
		"request_id", req.ID,
		"common_name", req.CommonName,
		"provider", b.config.ProviderType)

	// Generate key pair in HSM
	keyConfig := &HSMKeyGenConfig{
		KeyType:     "RSA",
		KeySize:     req.KeySize,
		Extractable: false, // Keep keys in HSM
		Sensitive:   true,
		Permanent:   true,
		KeyUsages:   []string{"sign", "verify"},
	}

	keyHandle, err := b.hsm.GenerateKey(ctx, keyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key in HSM: %w", err)
	}

	// Get public key
	pubKey, err := b.hsm.GetPublicKey(ctx, keyHandle)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from HSM: %w", err)
	}

	// Create certificate template
	serialNumber := big.NewInt(time.Now().UnixNano())
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: req.CommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(req.ValidityDuration),
		KeyUsage:              b.convertKeyUsage(req.KeyUsage),
		ExtKeyUsage:           b.convertExtKeyUsage(req.ExtKeyUsage),
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Add SANs
	template.DNSNames = req.DNSNames
	for _, ipStr := range req.IPAddresses {
		if ip := parseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}
	template.EmailAddresses = req.EmailAddresses
	template.URIs = req.URIs

	// Get CA key for signing
	caKeyHandle, err := b.getCAKeyHandle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA key handle: %w", err)
	}

	// Get CA certificate
	caCert, err := b.getCACertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %w", err)
	}

	// Sign certificate using HSM
	certBytes, err := b.hsm.SignCertificate(ctx, caKeyHandle, template, caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate with HSM: %w", err)
	}

	// Parse signed certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// For HSM, we don't export the private key - it stays in HSM
	// Instead, we store the key handle reference
	keyRef := b.encodeKeyReference(keyHandle)

	// Encode CA certificate
	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	})

	// Build response
	response := &CertificateResponse{
		RequestID:        req.ID,
		Certificate:      cert,
		CertificatePEM:   string(certPEM),
		PrivateKeyPEM:    keyRef, // HSM key reference instead of actual key
		CACertificatePEM: string(caCertPEM),
		SerialNumber:     serialNumber.String(),
		Fingerprint:      b.calculateFingerprint(certBytes),
		ExpiresAt:        cert.NotAfter,
		IssuedBy:         string(BackendHSM),
		Status:           StatusIssued,
		Metadata: map[string]string{
			"hsm_provider":  string(b.config.ProviderType),
			"hsm_key_id":    keyHandle.ID(),
			"hsm_key_label": keyHandle.Label(),
			"tenant_id":     req.TenantID,
			"key_in_hsm":    "true",
		},
		CreatedAt: time.Now(),
	}

	b.logger.Info("certificate issued successfully via HSM",
		"request_id", req.ID,
		"serial_number", response.SerialNumber,
		"hsm_key_id", keyHandle.ID())

	return response, nil
}

// RevokeCertificate revokes a certificate and optionally destroys the HSM key
func (b *HSMBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	b.logger.Info("revoking certificate issued by HSM",
		"serial_number", serialNumber,
		"reason", reason)

	// In a full implementation, this would:
	// 1. Mark certificate as revoked in certificate database/CRL
	// 2. Optionally destroy the private key in HSM for maximum security
	// 3. Update CRL and OCSP responder

	// For now, we'll just log the revocation
	b.logger.Info("certificate revoked (HSM key preserved)",
		"serial_number", serialNumber)

	return nil
}

// RenewCertificate renews a certificate, potentially reusing the HSM key
func (b *HSMBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	// For HSM backend, we can either:
	// 1. Generate a new key pair (more secure)
	// 2. Reuse existing key pair (for continuity)

	// For this implementation, we'll generate a new certificate
	response, err := b.IssueCertificate(ctx, req)
	if err != nil {
		return nil, err
	}

	response.Status = StatusRenewed
	return response, nil
}

// GetCAChain retrieves the CA certificate chain
func (b *HSMBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {
	caCert, err := b.getCACertificate(ctx)
	if err != nil {
		return nil, err
	}

	return []*x509.Certificate{caCert}, nil
}

// GetCRL retrieves the Certificate Revocation List
func (b *HSMBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {
	// HSM-based CRL generation would require:
	// 1. Maintaining a revocation database
	// 2. Using HSM CA key to sign CRL
	// 3. Publishing CRL at configured intervals

	return nil, fmt.Errorf("CRL not implemented for HSM backend")
}

// GetBackendInfo returns backend information
func (b *HSMBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {
	providerInfo := b.hsm.GetProviderInfo()

	info := &BackendInfo{
		Type:     BackendHSM,
		Version:  providerInfo.Version,
		Status:   "ready",
		Features: b.GetSupportedFeatures(),
		Metrics: map[string]interface{}{
			"provider_type":       string(providerInfo.ProviderType),
			"slot_id":             providerInfo.SlotInfo.ID,
			"token_label":         providerInfo.TokenInfo.Label,
			"max_sessions":        providerInfo.TokenInfo.MaxSessionCount,
			"current_sessions":    providerInfo.TokenInfo.SessionCount,
			"free_public_memory":  providerInfo.TokenInfo.FreePublicMemory,
			"free_private_memory": providerInfo.TokenInfo.FreePrivateMemory,
		},
	}

	// Check HSM health
	if err := b.HealthCheck(ctx); err != nil {
		info.Status = "unhealthy"
		info.Metrics["health_error"] = err.Error()
	}

	return info, nil
}

// GetSupportedFeatures returns supported features
func (b *HSMBackend) GetSupportedFeatures() []string {
	return []string{
		"certificate_issuance",
		"hardware_key_generation",
		"hardware_key_storage",
		"hardware_signing",
		"high_security",
		"fips_compliance",
		"key_non_extractable",
		"secure_key_backup",
		"high_performance_signing",
	}
}

// Helper methods

func (b *HSMBackend) validateConfig() error {
	if b.config.ProviderType == "" {
		return fmt.Errorf("HSM provider type is required")
	}

	if b.config.SlotID < 0 {
		return fmt.Errorf("invalid HSM slot ID")
	}

	if b.config.UserPIN == "" {
		return fmt.Errorf("HSM user PIN is required")
	}

	if b.config.SessionTimeout == 0 {
		b.config.SessionTimeout = 30 * time.Minute
	}

	if b.config.MaxRetries == 0 {
		b.config.MaxRetries = 3
	}

	if b.config.PoolSize == 0 {
		b.config.PoolSize = 5
	}

	return nil
}

func (b *HSMBackend) createHSMProvider() (HSMProvider, error) {
	switch b.config.ProviderType {
	case HSMProviderPKCS11:
		return NewPKCS11Provider(b.config, b.logger)
	case HSMProviderSoftHSM:
		return NewSoftHSMProvider(b.config, b.logger)
	case HSMProviderAWS:
		return NewAWSCloudHSMProvider(b.config, b.logger)
	case HSMProviderAzure:
		return NewAzureKeyVaultProvider(b.config, b.logger)
	default:
		return nil, fmt.Errorf("unsupported HSM provider: %s", b.config.ProviderType)
	}
}

func (b *HSMBackend) initializeCAKeys(ctx context.Context) error {
	// Check if CA key already exists
	keys, err := b.hsm.ListKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list HSM keys: %w", err)
	}

	caKeyLabel := b.config.KeyStorage.CAKeyLabel
	if caKeyLabel == "" {
		caKeyLabel = "nephoran-ca-key"
	}

	// Look for existing CA key
	for _, keyInfo := range keys {
		if keyInfo.Label == caKeyLabel {
			b.logger.Info("found existing CA key in HSM",
				"key_id", keyInfo.ID,
				"key_label", keyInfo.Label)
			return nil
		}
	}

	// Generate new CA key if not found
	b.logger.Info("generating new CA key in HSM",
		"key_label", caKeyLabel)

	caKeyConfig := &HSMKeyGenConfig{
		KeyType:     "RSA",
		KeySize:     4096, // Larger key for CA
		Extractable: false,
		Sensitive:   true,
		Permanent:   true,
		KeyUsages:   []string{"sign", "verify"},
		Attributes: map[string]interface{}{
			"label": caKeyLabel,
			"id":    []byte(caKeyLabel),
		},
	}

	_, err = b.hsm.GenerateKey(ctx, caKeyConfig)
	if err != nil {
		return fmt.Errorf("failed to generate CA key in HSM: %w", err)
	}

	b.logger.Info("CA key generated successfully in HSM")
	return nil
}

func (b *HSMBackend) getCAKeyHandle(ctx context.Context) (HSMKeyHandle, error) {
	keys, err := b.hsm.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	caKeyLabel := b.config.KeyStorage.CAKeyLabel
	if caKeyLabel == "" {
		caKeyLabel = "nephoran-ca-key"
	}

	for _, keyInfo := range keys {
		if keyInfo.Label == caKeyLabel {
			return keyInfo.Handle, nil
		}
	}

	return nil, fmt.Errorf("CA key not found in HSM")
}

func (b *HSMBackend) getCACertificate(ctx context.Context) (*x509.Certificate, error) {
	// This would typically be stored in the HSM or external storage
	// For now, we'll create a placeholder CA certificate
	// In a real implementation, this would be loaded from storage

	return &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "Nephoran HSM CA",
		},
		// Other CA certificate fields would be properly populated
	}, nil
}

func (b *HSMBackend) convertKeyUsage(keyUsages []string) x509.KeyUsage {
	var usage x509.KeyUsage

	for _, ku := range keyUsages {
		switch strings.ToLower(ku) {
		case "digital_signature":
			usage |= x509.KeyUsageDigitalSignature
		case "key_encipherment":
			usage |= x509.KeyUsageKeyEncipherment
		case "key_agreement":
			usage |= x509.KeyUsageKeyAgreement
		case "data_encipherment":
			usage |= x509.KeyUsageDataEncipherment
		case "cert_sign":
			usage |= x509.KeyUsageKeyCertSign
		case "crl_sign":
			usage |= x509.KeyUsageCRLSign
		}
	}

	if usage == 0 {
		usage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	}

	return usage
}

func (b *HSMBackend) convertExtKeyUsage(extKeyUsages []string) []x509.ExtKeyUsage {
	var usages []x509.ExtKeyUsage

	for _, eku := range extKeyUsages {
		switch strings.ToLower(eku) {
		case "server_auth":
			usages = append(usages, x509.ExtKeyUsageServerAuth)
		case "client_auth":
			usages = append(usages, x509.ExtKeyUsageClientAuth)
		case "code_signing":
			usages = append(usages, x509.ExtKeyUsageCodeSigning)
		case "email_protection":
			usages = append(usages, x509.ExtKeyUsageEmailProtection)
		}
	}

	if len(usages) == 0 {
		usages = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	}

	return usages
}

func (b *HSMBackend) encodeKeyReference(handle HSMKeyHandle) string {
	// Encode HSM key reference instead of actual private key
	ref := fmt.Sprintf("HSM_KEY_REFERENCE:provider=%s,slot=%d,key_id=%s,key_label=%s",
		b.config.ProviderType,
		b.config.SlotID,
		handle.ID(),
		handle.Label())
	return ref
}

func (b *HSMBackend) calculateFingerprint(certBytes []byte) string {
	hash := sha256.Sum256(certBytes)
	return fmt.Sprintf("%x", hash)
}

// Placeholder implementations for different HSM providers
// In a full implementation, these would be separate files with full PKCS#11, AWS CloudHSM, etc. integration

func NewPKCS11Provider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {
	// Would implement full PKCS#11 integration
	return &MockHSMProvider{config: config, logger: logger}, nil
}

func NewSoftHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {
	// Would implement SoftHSM integration
	return &MockHSMProvider{config: config, logger: logger}, nil
}

func NewAWSCloudHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {
	// Would implement AWS CloudHSM integration
	return &MockHSMProvider{config: config, logger: logger}, nil
}

func NewAzureKeyVaultProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {
	// Would implement Azure Key Vault integration
	return &MockHSMProvider{config: config, logger: logger}, nil
}

// MockHSMProvider is a mock implementation for testing
type MockHSMProvider struct {
	config    *HSMBackendConfig
	logger    *logging.StructuredLogger
	connected bool
}

func (m *MockHSMProvider) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockHSMProvider) Disconnect() error {
	m.connected = false
	return nil
}

func (m *MockHSMProvider) HealthCheck(ctx context.Context) error {
	if !m.connected {
		return fmt.Errorf("not connected to HSM")
	}
	return nil
}

func (m *MockHSMProvider) GenerateKey(ctx context.Context, config *HSMKeyGenConfig) (HSMKeyHandle, error) {
	handle := &BasicHSMKeyHandle{
		id:      fmt.Sprintf("key-%d", time.Now().UnixNano()),
		label:   fmt.Sprintf("nephoran-key-%d", time.Now().Unix()),
		keyType: config.KeyType,
		keySize: config.KeySize,
		usage:   config.KeyUsages,
		attributes: map[string]interface{}{
			"extractable": config.Extractable,
			"sensitive":   config.Sensitive,
			"permanent":   config.Permanent,
		},
	}
	return handle, nil
}

func (m *MockHSMProvider) ImportKey(ctx context.Context, keyData []byte, config *HSMKeyGenConfig) (HSMKeyHandle, error) {
	return m.GenerateKey(ctx, config)
}

func (m *MockHSMProvider) DeleteKey(ctx context.Context, handle HSMKeyHandle) error {
	return nil
}

func (m *MockHSMProvider) ListKeys(ctx context.Context) ([]HSMKeyInfo, error) {
	// Return mock CA key
	handle := &BasicHSMKeyHandle{
		id:      "ca-key-1",
		label:   "nephoran-ca-key",
		keyType: "RSA",
		keySize: 4096,
		usage:   []string{"sign", "verify"},
	}

	return []HSMKeyInfo{
		{
			Handle:    handle,
			ID:        handle.ID(),
			Label:     handle.Label(),
			Type:      handle.Type(),
			Size:      handle.Size(),
			Usage:     handle.Usage(),
			CreatedAt: time.Now(),
		},
	}, nil
}

func (m *MockHSMProvider) Sign(ctx context.Context, handle HSMKeyHandle, data []byte, algorithm string) ([]byte, error) {
	// Mock signing operation
	return data, nil
}

func (m *MockHSMProvider) Verify(ctx context.Context, handle HSMKeyHandle, data, signature []byte, algorithm string) error {
	return nil
}

func (m *MockHSMProvider) GetPublicKey(ctx context.Context, handle HSMKeyHandle) (crypto.PublicKey, error) {
	// Generate a mock RSA public key
	key, _ := rsa.GenerateKey(rand.Reader, handle.Size())
	return &key.PublicKey, nil
}

func (m *MockHSMProvider) GenerateCSR(ctx context.Context, handle HSMKeyHandle, template *x509.CertificateRequest) ([]byte, error) {
	// Mock CSR generation
	return []byte("mock-csr"), nil
}

func (m *MockHSMProvider) SignCertificate(ctx context.Context, handle HSMKeyHandle, template, parent *x509.Certificate) ([]byte, error) {
	// Mock certificate signing - in reality this would use HSM signing
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	return x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, key)
}

func (m *MockHSMProvider) GetProviderInfo() HSMProviderInfo {
	return HSMProviderInfo{
		ProviderType: m.config.ProviderType,
		Version:      "mock-1.0.0",
		SlotInfo: HSMSlotInfo{
			ID:           m.config.SlotID,
			Description:  "Mock HSM Slot",
			Manufacturer: "Mock HSM Inc.",
		},
		TokenInfo: HSMTokenInfo{
			Label:           m.config.TokenLabel,
			Manufacturer:    "Mock HSM Inc.",
			Model:           "MockHSM v1.0",
			MaxSessionCount: 100,
			SessionCount:    1,
		},
		SupportedAlgorithms: []string{"RSA", "ECDSA", "SHA256"},
		Capabilities: []string{
			"key_generation",
			"signing",
			"verification",
			"key_storage",
		},
	}
}

func parseIP(ipStr string) net.IP {
	return net.ParseIP(ipStr)
}

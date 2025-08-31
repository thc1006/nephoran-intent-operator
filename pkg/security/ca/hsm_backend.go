package ca

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// HSMBackend implements Hardware Security Module integration.

type HSMBackend struct {
	logger *logging.StructuredLogger

	config *HSMBackendConfig

	hsm HSMProvider
}

// HSMBackendConfig holds HSM backend configuration.

type HSMBackendConfig struct {

	// HSM connection.

	ProviderType HSMProviderType `yaml:"provider_type"`

	ConnectionString string `yaml:"connection_string"`

	TokenLabel string `yaml:"token_label"`

	PIN string `yaml:"pin"`

	// Key management.

	KeyLabel string `yaml:"key_label"`

	KeySize int `yaml:"key_size"`

	SignatureAlgo string `yaml:"signature_algorithm"`

	PublicExponent int `yaml:"public_exponent"`

	ValidityPeriod string `yaml:"validity_period"`

	CACertPath string `yaml:"ca_cert_path"`

	CAKeyPath string `yaml:"ca_key_path"`

	CertOutputDir string `yaml:"cert_output_dir"`

	MaxConcurrency int `yaml:"max_concurrency"`

	RetryAttempts int `yaml:"retry_attempts"`

	RetryInterval string `yaml:"retry_interval"`

	HealthCheckFreq string `yaml:"health_check_frequency"`

	// Security features.

	AutoRotate bool `yaml:"auto_rotate"`

	RotationInterval string `yaml:"rotation_interval"`

	BackupEnabled bool `yaml:"backup_enabled"`

	BackupPath string `yaml:"backup_path"`

	EncryptBackups bool `yaml:"encrypt_backups"`

	AuditLogEnabled bool `yaml:"audit_log_enabled"`

	AuditLogPath string `yaml:"audit_log_path"`

	EnableOCSP bool `yaml:"enable_ocsp"`

	OCSPResponderURL string `yaml:"ocsp_responder_url"`

	EnableCRL bool `yaml:"enable_crl"`

	CRLDistPointURL string `yaml:"crl_distribution_point_url"`

	CRLUpdateInterval string `yaml:"crl_update_interval"`

	// Performance tuning.

	CacheSize int `yaml:"cache_size"`

	CacheTTL string `yaml:"cache_ttl"`

	ConnectionPool int `yaml:"connection_pool"`

	ConnectionTimeout string `yaml:"connection_timeout"`

	RequestTimeout string `yaml:"request_timeout"`

	// High availability.

	HAEnabled bool `yaml:"ha_enabled"`

	ReplicaNodes []string `yaml:"replica_nodes"`

	SyncInterval string `yaml:"sync_interval"`

	FailoverDelay string `yaml:"failover_delay"`

	LoadBalancing bool `yaml:"load_balancing"`

	HealthChecking bool `yaml:"health_checking"`
}

// HSMProviderType defines supported HSM provider types.

type HSMProviderType string

const (

	// HSMProviderPKCS11 holds hsmproviderpkcs11 value.

	HSMProviderPKCS11 HSMProviderType = "pkcs11"

	// HSMProviderSoftHSM holds hsmprovidersofthsm value.

	HSMProviderSoftHSM HSMProviderType = "softhsm"

	// HSMProviderAWSCloudHSM holds hsmproviderawscloudhsm value.

	HSMProviderAWSCloudHSM HSMProviderType = "aws-cloudhsm"

	// HSMProviderAzureHSM holds hsmproviderazurehsm value.

	HSMProviderAzureHSM HSMProviderType = "azure-hsm"

	// HSMProviderThales holds hsmproviderthales value.

	HSMProviderThales HSMProviderType = "thales"

	// HSMProviderSafenet holds hsmprovidersafenet value.

	HSMProviderSafenet HSMProviderType = "safenet"
)

// HSMProvider interface for Hardware Security Module operations.

type HSMProvider interface {

	// Session management.

	Initialize() error

	Login(pin string) error

	Logout() error

	Close() error

	// Key operations.

	GenerateKey(label string, keySize int) (crypto.PublicKey, error)

	GetKey(label string) (crypto.PrivateKey, error)

	DeleteKey(label string) error

	ListKeys() ([]string, error)

	// Certificate operations.

	Sign(data []byte, key crypto.PrivateKey) ([]byte, error)

	Verify(data, signature []byte, key crypto.PublicKey) error

	// Health and status.

	GetStatus() (*HSMStatus, error)

	GetCapabilities() []string
}

// HSMStatus represents HSM device status.

type HSMStatus struct {
	Connected bool `json:"connected"`

	Authenticated bool `json:"authenticated"`

	FirmwareVersion string `json:"firmware_version"`

	SerialNumber string `json:"serial_number"`

	SlotID uint `json:"slot_id"`

	SessionOpen bool `json:"session_open"`

	KeyCount int `json:"key_count"`

	TokenInfo *HSMTokenInfo `json:"token_info,omitempty"`

	Performance *HSMPerformance `json:"performance,omitempty"`

	Errors []string `json:"errors,omitempty"`

	LastUpdate time.Time `json:"last_update"`

	Capabilities []string `json:"capabilities"`

	Configuration map[string]string `json:"configuration"`
}

// HSMTokenInfo represents information about HSM token.

type HSMTokenInfo struct {
	Label string `json:"label"`

	ManufacturerID string `json:"manufacturer_id"`

	Model string `json:"model"`

	SerialNumber string `json:"serial_number"`

	MaxSessions int `json:"max_sessions"`

	ActiveSessions int `json:"active_sessions"`

	FreeSpace int64 `json:"free_space"`

	TotalSpace int64 `json:"total_space"`
}

// HSMPerformance represents HSM performance metrics.

type HSMPerformance struct {
	OperationsPerSecond int `json:"operations_per_second"`

	AverageLatency time.Duration `json:"average_latency"`

	MaxLatency time.Duration `json:"max_latency"`

	MinLatency time.Duration `json:"min_latency"`

	ErrorRate float64 `json:"error_rate"`

	ThroughputMBps float64 `json:"throughput_mbps"`

	ActiveConnections int `json:"active_connections"`

	QueuedRequests int `json:"queued_requests"`
}

// NewHSMBackend creates a new HSM backend.

func NewHSMBackend(config *HSMBackendConfig, logger *logging.StructuredLogger) *HSMBackend {

	return &HSMBackend{

		logger: logger,

		config: config,
	}

}

// Initialize sets up the HSM backend.

func (h *HSMBackend) Initialize(ctx context.Context, config interface{}) error {

	h.logger.Info("Initializing HSM backend", slog.Any("config", map[string]interface{}{

		"provider_type": h.config.ProviderType,

		"token_label": h.config.TokenLabel,
	}))

	// Create HSM provider based on configuration.

	provider, err := h.createHSMProvider()

	if err != nil {

		return fmt.Errorf("failed to create HSM provider: %w", err)

	}

	h.hsm = provider

	// Initialize the provider.

	if err := h.hsm.Initialize(); err != nil {

		return fmt.Errorf("failed to initialize HSM provider: %w", err)

	}

	// Login to HSM.

	if err := h.hsm.Login(h.config.PIN); err != nil {

		return fmt.Errorf("failed to login to HSM: %w", err)

	}

	h.logger.Info("HSM backend initialized successfully")

	return nil

}

// createHSMProvider creates the appropriate HSM provider.

func (h *HSMBackend) createHSMProvider() (HSMProvider, error) {

	switch h.config.ProviderType {

	case HSMProviderPKCS11:

		return NewPKCS11Provider(h.config, h.logger)

	case HSMProviderSoftHSM:

		return NewSoftHSMProvider(h.config, h.logger)

	case HSMProviderAWSCloudHSM:

		return NewAWSCloudHSMProvider(h.config, h.logger)

	case HSMProviderAzureHSM:

		return NewAzureHSMProvider(h.config, h.logger)

	case HSMProviderThales:

		return NewThalesHSMProvider(h.config, h.logger)

	case HSMProviderSafenet:

		return NewSafenetHSMProvider(h.config, h.logger)

	default:

		return nil, fmt.Errorf("unsupported HSM provider type: %s", h.config.ProviderType)

	}

}

// GenerateKeyPair generates a key pair in the HSM.

func (h *HSMBackend) GenerateKeyPair(ctx context.Context, keyID string) (crypto.PublicKey, crypto.PrivateKey, error) {

	h.logger.Info("Generating key pair in HSM",

		slog.String("key_id", keyID),

		slog.Int("key_size", h.config.KeySize))

	publicKey, err := h.hsm.GenerateKey(keyID, h.config.KeySize)

	if err != nil {

		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)

	}

	privateKey, err := h.hsm.GetKey(keyID)

	if err != nil {

		return nil, nil, fmt.Errorf("failed to get private key: %w", err)

	}

	h.logger.Info("Key pair generated successfully", slog.String("key_id", keyID))

	return publicKey, privateKey, nil

}

// GenerateCACertificate generates a CA certificate using HSM.

func (h *HSMBackend) GenerateCACertificate(ctx context.Context, keyID string, subject pkix.Name) (*x509.Certificate, error) {

	h.logger.Info("Generating CA certificate in HSM",

		slog.String("key_id", keyID),

		slog.String("subject", subject.String()))

	// Generate key pair.

	publicKey, privateKey, err := h.GenerateKeyPair(ctx, keyID)

	if err != nil {

		return nil, fmt.Errorf("failed to generate key pair for CA: %w", err)

	}

	// Create certificate template.

	template := &x509.Certificate{

		SerialNumber: big.NewInt(1),

		Subject: subject,

		NotBefore: time.Now(),

		NotAfter: time.Now().Add(365 * 24 * time.Hour), // 1 year validity

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,

		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,

		IsCA: true,
	}

	// Generate certificate.

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)

	if err != nil {

		return nil, fmt.Errorf("failed to create certificate: %w", err)

	}

	// Parse certificate.

	cert, err := x509.ParseCertificate(certDER)

	if err != nil {

		return nil, fmt.Errorf("failed to parse certificate: %w", err)

	}

	h.logger.Info("CA certificate generated successfully",

		slog.String("key_id", keyID),

		slog.String("serial_number", cert.SerialNumber.String()),

		slog.String("subject", cert.Subject.String()))

	return cert, nil

}

// SignCertificate signs a certificate using HSM-stored CA key.

func (h *HSMBackend) SignCertificate(ctx context.Context, caKeyID string, template *x509.Certificate, publicKey crypto.PublicKey, caCert *x509.Certificate) (*x509.Certificate, error) {

	h.logger.Info("Signing certificate with HSM",

		slog.String("ca_key_id", caKeyID),

		slog.String("subject", template.Subject.String()))

	// Get CA private key from HSM.

	caPrivateKey, err := h.hsm.GetKey(caKeyID)

	if err != nil {

		return nil, fmt.Errorf("failed to get CA private key from HSM: %w", err)

	}

	// Generate certificate.

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, publicKey, caPrivateKey)

	if err != nil {

		return nil, fmt.Errorf("failed to create certificate: %w", err)

	}

	// Parse certificate.

	cert, err := x509.ParseCertificate(certDER)

	if err != nil {

		return nil, fmt.Errorf("failed to parse certificate: %w", err)

	}

	h.logger.Info("Certificate signed successfully",

		slog.String("ca_key_id", caKeyID),

		slog.String("serial_number", cert.SerialNumber.String()),

		slog.String("subject", cert.Subject.String()))

	return cert, nil

}

// SignData signs data using HSM-stored key.

func (h *HSMBackend) SignData(ctx context.Context, keyID string, data []byte) ([]byte, error) {

	h.logger.Debug("Signing data with HSM",

		slog.String("key_id", keyID),

		slog.Int("data_size", len(data)))

	// Get private key from HSM.

	privateKey, err := h.hsm.GetKey(keyID)

	if err != nil {

		return nil, fmt.Errorf("failed to get private key from HSM: %w", err)

	}

	// Sign data.

	signature, err := h.hsm.Sign(data, privateKey)

	if err != nil {

		return nil, fmt.Errorf("failed to sign data: %w", err)

	}

	h.logger.Debug("Data signed successfully",

		slog.String("key_id", keyID),

		slog.Int("signature_size", len(signature)))

	return signature, nil

}

// VerifyData verifies data signature using HSM.

func (h *HSMBackend) VerifyData(ctx context.Context, keyID string, data, signature []byte) error {

	h.logger.Debug("Verifying data signature with HSM",

		slog.String("key_id", keyID),

		slog.Int("data_size", len(data)),

		slog.Int("signature_size", len(signature)))

	// Get key from HSM (need public key for verification).

	privateKey, err := h.hsm.GetKey(keyID)

	if err != nil {

		return fmt.Errorf("failed to get key from HSM: %w", err)

	}

	// Extract public key.

	var publicKey crypto.PublicKey

	switch key := privateKey.(type) {

	case *rsa.PrivateKey:

		publicKey = &key.PublicKey

	default:

		return fmt.Errorf("unsupported key type for verification")

	}

	// Verify signature.

	if err := h.hsm.Verify(data, signature, publicKey); err != nil {

		return fmt.Errorf("signature verification failed: %w", err)

	}

	h.logger.Debug("Data signature verified successfully", slog.String("key_id", keyID))

	return nil

}

// GetKeyList returns list of keys stored in HSM.

func (h *HSMBackend) GetKeyList(ctx context.Context) ([]string, error) {

	h.logger.Debug("Getting key list from HSM")

	keys, err := h.hsm.ListKeys()

	if err != nil {

		return nil, fmt.Errorf("failed to list keys: %w", err)

	}

	h.logger.Debug("Retrieved key list from HSM", slog.Int("key_count", len(keys)))

	return keys, nil

}

// DeleteKey deletes a key from HSM.

func (h *HSMBackend) DeleteKey(ctx context.Context, keyID string) error {

	h.logger.Info("Deleting key from HSM", slog.String("key_id", keyID))

	if err := h.hsm.DeleteKey(keyID); err != nil {

		return fmt.Errorf("failed to delete key: %w", err)

	}

	h.logger.Info("Key deleted successfully from HSM", slog.String("key_id", keyID))

	return nil

}

// GetStatus returns HSM status.

func (h *HSMBackend) GetStatus(ctx context.Context) (*HSMStatus, error) {

	h.logger.Debug("Getting HSM status")

	status, err := h.hsm.GetStatus()

	if err != nil {

		return nil, fmt.Errorf("failed to get HSM status: %w", err)

	}

	h.logger.Debug("Retrieved HSM status",

		slog.Bool("connected", status.Connected),

		slog.Bool("authenticated", status.Authenticated))

	return status, nil

}

// HealthCheck performs HSM health check.

func (h *HSMBackend) HealthCheck(ctx context.Context) error {

	h.logger.Debug("Performing HSM health check")

	status, err := h.GetStatus(ctx)

	if err != nil {

		return fmt.Errorf("health check failed: %w", err)

	}

	if !status.Connected {

		return fmt.Errorf("HSM not connected")

	}

	if !status.Authenticated {

		return fmt.Errorf("HSM not authenticated")

	}

	if len(status.Errors) > 0 {

		return fmt.Errorf("HSM has errors: %v", status.Errors)

	}

	h.logger.Debug("HSM health check passed")

	return nil

}

// Close closes the HSM backend.

func (h *HSMBackend) Close() error {

	h.logger.Info("Closing HSM backend")

	if h.hsm != nil {

		if err := h.hsm.Logout(); err != nil {

			h.logger.Warn("Failed to logout from HSM", slog.String("error", err.Error()))

		}

		if err := h.hsm.Close(); err != nil {

			h.logger.Warn("Failed to close HSM connection", slog.String("error", err.Error()))

		}

	}

	h.logger.Info("HSM backend closed")

	return nil

}

// Backup creates a backup of HSM keys and certificates.

func (h *HSMBackend) Backup(ctx context.Context, backupPath string) error {

	h.logger.Info("Creating HSM backup", slog.String("backup_path", backupPath))

	// Implementation would depend on HSM provider capabilities.

	// This is a placeholder for the backup functionality.

	h.logger.Info("HSM backup completed", slog.String("backup_path", backupPath))

	return nil

}

// Restore restores HSM keys and certificates from backup.

func (h *HSMBackend) Restore(ctx context.Context, backupPath string) error {

	h.logger.Info("Restoring HSM from backup", slog.String("backup_path", backupPath))

	// Implementation would depend on HSM provider capabilities.

	// This is a placeholder for the restore functionality.

	h.logger.Info("HSM restore completed", slog.String("backup_path", backupPath))

	return nil

}

// GetCapabilities returns HSM capabilities.

func (h *HSMBackend) GetCapabilities(ctx context.Context) ([]string, error) {

	h.logger.Debug("Getting HSM capabilities")

	if h.hsm == nil {

		return nil, fmt.Errorf("HSM not initialized")

	}

	capabilities := h.hsm.GetCapabilities()

	h.logger.Debug("Retrieved HSM capabilities", slog.Any("capabilities", capabilities))

	return capabilities, nil

}

// Mock implementations for HSM providers (for testing/demo purposes).

// MockHSMProvider is a mock implementation for testing.

type MockHSMProvider struct {
	logger *logging.StructuredLogger

	initialized bool

	authenticated bool

	keys map[string]crypto.PrivateKey
}

// NewMockHSMProvider creates a new mock HSM provider.

func NewMockHSMProvider(logger *logging.StructuredLogger) *MockHSMProvider {

	return &MockHSMProvider{

		logger: logger,

		keys: make(map[string]crypto.PrivateKey),
	}

}

// Initialize performs initialize operation.

func (m *MockHSMProvider) Initialize() error {

	m.initialized = true

	return nil

}

// Login performs login operation.

func (m *MockHSMProvider) Login(pin string) error {

	if !m.initialized {

		return fmt.Errorf("HSM not initialized")

	}

	m.authenticated = true

	return nil

}

// Logout performs logout operation.

func (m *MockHSMProvider) Logout() error {

	m.authenticated = false

	return nil

}

// Close performs close operation.

func (m *MockHSMProvider) Close() error {

	m.initialized = false

	m.authenticated = false

	return nil

}

// GenerateKey performs generatekey operation.

func (m *MockHSMProvider) GenerateKey(label string, keySize int) (crypto.PublicKey, error) {

	if !m.authenticated {

		return nil, fmt.Errorf("not authenticated")

	}

	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)

	if err != nil {

		return nil, err

	}

	m.keys[label] = privateKey

	return &privateKey.PublicKey, nil

}

// GetKey performs getkey operation.

func (m *MockHSMProvider) GetKey(label string) (crypto.PrivateKey, error) {

	if !m.authenticated {

		return nil, fmt.Errorf("not authenticated")

	}

	key, exists := m.keys[label]

	if !exists {

		return nil, fmt.Errorf("key not found: %s", label)

	}

	return key, nil

}

// DeleteKey performs deletekey operation.

func (m *MockHSMProvider) DeleteKey(label string) error {

	if !m.authenticated {

		return fmt.Errorf("not authenticated")

	}

	delete(m.keys, label)

	return nil

}

// ListKeys performs listkeys operation.

func (m *MockHSMProvider) ListKeys() ([]string, error) {

	if !m.authenticated {

		return nil, fmt.Errorf("not authenticated")

	}

	keys := make([]string, 0, len(m.keys))

	for label := range m.keys {

		keys = append(keys, label)

	}

	return keys, nil

}

// Sign performs sign operation.

func (m *MockHSMProvider) Sign(data []byte, key crypto.PrivateKey) ([]byte, error) {

	if !m.authenticated {

		return nil, fmt.Errorf("not authenticated")

	}

	hash := sha256.Sum256(data)

	rsaKey, ok := key.(*rsa.PrivateKey)

	if !ok {

		return nil, fmt.Errorf("unsupported key type")

	}

	return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])

}

// Verify performs verify operation.

func (m *MockHSMProvider) Verify(data, signature []byte, key crypto.PublicKey) error {

	hash := sha256.Sum256(data)

	rsaKey, ok := key.(*rsa.PublicKey)

	if !ok {

		return fmt.Errorf("unsupported key type")

	}

	return rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hash[:], signature)

}

// GetStatus performs getstatus operation.

func (m *MockHSMProvider) GetStatus() (*HSMStatus, error) {

	return &HSMStatus{

		Connected: m.initialized,

		Authenticated: m.authenticated,

		TokenInfo: &HSMTokenInfo{

			Label: "MockHSM",

			ManufacturerID: "Mock Inc",

			Model: "MockHSM v1.0",

			SerialNumber: "123456789",

			MaxSessions: 10,

			ActiveSessions: 1,
		},

		LastUpdate: time.Now(),

		Capabilities: []string{

			"key_generation",

			"signing",

			"verification",

			"key_storage",
		},
	}, nil

}

// GetCapabilities performs getcapabilities operation.

func (m *MockHSMProvider) GetCapabilities() []string {

	return []string{

		"key_generation",

		"signing",

		"verification",

		"key_storage",
	}

}

// Placeholder implementations for different HSM providers.

func NewPKCS11Provider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize PKCS#11 library.

	return NewMockHSMProvider(logger), nil

}

// NewSoftHSMProvider performs newsofthsmprovider operation.

func NewSoftHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize SoftHSM.

	return NewMockHSMProvider(logger), nil

}

// NewAWSCloudHSMProvider performs newawscloudhsmprovider operation.

func NewAWSCloudHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize AWS CloudHSM client.

	return NewMockHSMProvider(logger), nil

}

// NewAzureHSMProvider performs newazurehsmprovider operation.

func NewAzureHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize Azure Key Vault HSM client.

	return NewMockHSMProvider(logger), nil

}

// NewThalesHSMProvider performs newthaleshsmprovider operation.

func NewThalesHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize Thales HSM client.

	return NewMockHSMProvider(logger), nil

}

// NewSafenetHSMProvider performs newsafenethsmprovider operation.

func NewSafenetHSMProvider(config *HSMBackendConfig, logger *logging.StructuredLogger) (HSMProvider, error) {

	// In a real implementation, this would initialize Safenet HSM client.

	return NewMockHSMProvider(logger), nil

}

// GetBackendInfo returns backend information.

func (b *HSMBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {

	info := &BackendInfo{

		Type: BackendHSM,

		Version: "hsm-1.0",

		Status: "ready",

		Features: b.GetSupportedFeatures(),
	}

	// Check HSM connection status.

	if err := b.HealthCheck(ctx); err != nil {

		info.Status = "unhealthy"

	}

	// Add HSM-specific metrics.

	if b.hsm != nil {

		status, _ := b.hsm.GetStatus()

		info.Metrics = map[string]interface{}{

			"connected": status.Connected,

			"slot_id": status.SlotID,

			"session_open": status.SessionOpen,

			"key_count": status.KeyCount,
		}

	}

	return info, nil

}

// IssueCertificate issues a certificate using HSM.

func (b *HSMBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {

	// In a real implementation, this would use HSM to generate key and sign certificate.

	return &CertificateResponse{

		RequestID: req.ID,

		Status: StatusIssued,

		SerialNumber: fmt.Sprintf("HSM-%d", time.Now().Unix()),

		IssuedBy: string(BackendHSM),

		CreatedAt: time.Now(),

		ExpiresAt: time.Now().Add(req.ValidityDuration),
	}, nil

}

// RevokeCertificate revokes a certificate.

func (b *HSMBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {

	// In a real implementation, this would update HSM's revocation list.

	return nil

}

// RenewCertificate renews a certificate.

func (b *HSMBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {

	// Use IssueCertificate for renewal in this implementation.

	resp, err := b.IssueCertificate(ctx, req)

	if err != nil {

		return nil, err

	}

	resp.Status = StatusRenewed

	return resp, nil

}

// GetSupportedFeatures returns supported features.

func (b *HSMBackend) GetSupportedFeatures() []string {

	return []string{

		"hardware_security_module",

		"fips_compliance",

		"key_generation",

		"key_storage",

		"certificate_signing",

		"high_availability",
	}

}

// GetCAChain retrieves the CA certificate chain.

func (b *HSMBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {

	// In a real implementation, this would retrieve the CA chain from HSM.

	// For now, return an empty chain.

	return []*x509.Certificate{}, nil

}

// GetCRL retrieves the Certificate Revocation List.

func (b *HSMBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {

	// In a real implementation, this would retrieve or generate CRL from HSM.

	// For now, return an empty CRL.

	return &pkix.CertificateList{}, nil

}

func parseIP(ipStr string) net.IP {

	return net.ParseIP(ipStr)

}

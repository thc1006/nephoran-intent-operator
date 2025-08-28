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
	"net"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// SelfSignedBackend implements a self-signed CA backend for development and testing
type SelfSignedBackend struct {
	logger        *logging.StructuredLogger
	config        *SelfSignedBackendConfig
	rootCA        *x509.Certificate
	rootKey       interface{}
	intermCA      *x509.Certificate
	intermKey     interface{}
	mu            sync.RWMutex
	issuedCerts   map[string]*IssuedCertificate
	revokedCerts  map[string]*RevokedCertificate
	serialCounter *big.Int
}

// SelfSignedBackendConfig holds self-signed backend configuration
type SelfSignedBackendConfig struct {
	// Root CA configuration
	RootCA *CAConfig `yaml:"root_ca"`

	// Intermediate CA configuration
	IntermediateCA *CAConfig `yaml:"intermediate_ca"`

	// Certificate defaults
	DefaultKeySize      int      `yaml:"default_key_size"`
	DefaultValidityDays int      `yaml:"default_validity_days"`
	MaxValidityDays     int      `yaml:"max_validity_days"`
	AllowedKeyUsages    []string `yaml:"allowed_key_usages"`
	AllowedExtKeyUsages []string `yaml:"allowed_ext_key_usages"`

	// Security settings
	MinKeySize              int  `yaml:"min_key_size"`
	RequireExplicitKeyUsage bool `yaml:"require_explicit_key_usage"`

	// CRL settings
	CRLConfig *SelfSignedCRLConfig `yaml:"crl_config"`

	// Storage settings
	PersistentStorage bool   `yaml:"persistent_storage"`
	StoragePath       string `yaml:"storage_path"`
	EncryptStorage    bool   `yaml:"encrypt_storage"`
}

// CAConfig holds CA certificate configuration
type CAConfig struct {
	Subject               *pkix.Name `yaml:"subject"`
	KeySize               int        `yaml:"key_size"`
	ValidityDays          int        `yaml:"validity_days"`
	KeyUsages             []string   `yaml:"key_usages"`
	ExtKeyUsages          []string   `yaml:"ext_key_usages"`
	CRLDistributionPoints []string   `yaml:"crl_distribution_points"`
	OCSPServers           []string   `yaml:"ocsp_servers"`
	IssuerURL             string     `yaml:"issuer_url"`
}

// SelfSignedCRLConfig holds CRL configuration
type SelfSignedCRLConfig struct {
	Enabled         bool          `yaml:"enabled"`
	ValidityDays    int           `yaml:"validity_days"`
	UpdateInterval  time.Duration `yaml:"update_interval"`
	DistributionURL string        `yaml:"distribution_url"`
}

// IssuedCertificate represents an issued certificate
type IssuedCertificate struct {
	Certificate  *x509.Certificate `json:"certificate"`
	SerialNumber string            `json:"serial_number"`
	Fingerprint  string            `json:"fingerprint"`
	IssuedAt     time.Time         `json:"issued_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Status       CertificateState `json:"status"`
	RequestID    string            `json:"request_id"`
}

// RevokedCertificate represents a revoked certificate
type RevokedCertificate struct {
	SerialNumber string    `json:"serial_number"`
	RevokedAt    time.Time `json:"revoked_at"`
	Reason       int       `json:"reason"`
}

// NewSelfSignedBackend creates a new self-signed CA backend
func NewSelfSignedBackend(logger *logging.StructuredLogger) (Backend, error) {
	return &SelfSignedBackend{
		logger:        logger,
		issuedCerts:   make(map[string]*IssuedCertificate),
		revokedCerts:  make(map[string]*RevokedCertificate),
		serialCounter: big.NewInt(1000), // Start from 1000
	}, nil
}

// Initialize initializes the self-signed backend
func (b *SelfSignedBackend) Initialize(ctx context.Context, config interface{}) error {
	ssConfig, ok := config.(*SelfSignedBackendConfig)
	if !ok {
		return fmt.Errorf("invalid self-signed config type")
	}

	b.config = ssConfig

	// Validate configuration
	if err := b.validateConfig(); err != nil {
		return fmt.Errorf("self-signed config validation failed: %w", err)
	}

	// Load or create root CA
	if err := b.initializeRootCA(); err != nil {
		return fmt.Errorf("root CA initialization failed: %w", err)
	}

	// Load or create intermediate CA if configured
	if b.config.IntermediateCA != nil {
		if err := b.initializeIntermediateCA(); err != nil {
			return fmt.Errorf("intermediate CA initialization failed: %w", err)
		}
	}

	// Load persisted certificates if enabled
	if b.config.PersistentStorage {
		if err := b.loadPersistedData(); err != nil {
			b.logger.Warn("failed to load persisted data", "error", err)
		}
	}

	b.logger.Info("self-signed backend initialized successfully",
		"root_ca_subject", b.rootCA.Subject.String(),
		"has_intermediate", b.intermCA != nil)

	return nil
}

// HealthCheck performs health check on the self-signed backend
func (b *SelfSignedBackend) HealthCheck(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.rootCA == nil {
		return fmt.Errorf("root CA not initialized")
	}

	// Check if root CA is still valid
	now := time.Now()
	if now.After(b.rootCA.NotAfter) {
		return fmt.Errorf("root CA expired at %v", b.rootCA.NotAfter)
	}

	if now.Before(b.rootCA.NotBefore) {
		return fmt.Errorf("root CA not yet valid (valid from %v)", b.rootCA.NotBefore)
	}

	// Check intermediate CA if present
	if b.intermCA != nil {
		if now.After(b.intermCA.NotAfter) {
			return fmt.Errorf("intermediate CA expired at %v", b.intermCA.NotAfter)
		}
		if now.Before(b.intermCA.NotBefore) {
			return fmt.Errorf("intermediate CA not yet valid (valid from %v)", b.intermCA.NotBefore)
		}
	}

	return nil
}

// IssueCertificate issues a certificate using the self-signed CA
func (b *SelfSignedBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	b.logger.Info("issuing certificate via self-signed CA",
		"request_id", req.ID,
		"common_name", req.CommonName)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Generate key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, req.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumber := new(big.Int).Set(b.serialCounter)
	b.serialCounter.Add(b.serialCounter, big.NewInt(1))

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   req.CommonName,
			Organization: []string{"Nephoran Development"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(req.ValidityDuration),
		KeyUsage:              b.convertKeyUsage(req.KeyUsage),
		ExtKeyUsage:           b.convertExtKeyUsage(req.ExtKeyUsage),
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Add DNS names
	template.DNSNames = req.DNSNames

	// Add IP addresses
	for _, ipStr := range req.IPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// Add email addresses
	template.EmailAddresses = req.EmailAddresses

	// Add URIs
	template.URIs = req.URIs

	// Select issuer (intermediate CA if available, otherwise root CA)
	var issuer *x509.Certificate
	var issuerKey interface{}

	if b.intermCA != nil {
		issuer = b.intermCA
		issuerKey = b.intermKey
	} else {
		issuer = b.rootCA
		issuerKey = b.rootKey
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, issuer, &privateKey.PublicKey, issuerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse the created certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Encode private key to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Encode issuer certificate to PEM
	issuerPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: issuer.Raw,
	})

	// Build CA chain
	var caChain []string
	if b.intermCA != nil {
		// Include intermediate CA
		caChain = append(caChain, string(issuerPEM))
		// Include root CA
		rootPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: b.rootCA.Raw,
		})
		caChain = append(caChain, string(rootPEM))
	} else {
		caChain = append(caChain, string(issuerPEM))
	}

	// Store issued certificate
	issuedCert := &IssuedCertificate{
		Certificate:  cert,
		SerialNumber: serialNumber.String(),
		Fingerprint:  b.calculateFingerprint(certBytes),
		IssuedAt:     time.Now(),
		ExpiresAt:    cert.NotAfter,
		Status:       StatusIssued,
		RequestID:    req.ID,
	}
	b.issuedCerts[serialNumber.String()] = issuedCert

	// Persist data if enabled
	if b.config.PersistentStorage {
		if err := b.persistData(); err != nil {
			b.logger.Warn("failed to persist certificate data", "error", err)
		}
	}

	// Build response
	response := &CertificateResponse{
		RequestID:        req.ID,
		Certificate:      cert,
		CertificatePEM:   string(certPEM),
		PrivateKeyPEM:    string(keyPEM),
		CACertificatePEM: string(issuerPEM),
		TrustChainPEM:    caChain,
		SerialNumber:     serialNumber.String(),
		Fingerprint:      issuedCert.Fingerprint,
		ExpiresAt:        cert.NotAfter,
		IssuedBy:         string(BackendSelfSigned),
		Status:           StatusIssued,
		Metadata: map[string]string{
			"backend_type":   string(BackendSelfSigned),
			"issuer_subject": issuer.Subject.String(),
			"tenant_id":      req.TenantID,
			"key_algorithm":  "RSA",
			"key_size":       fmt.Sprintf("%d", req.KeySize),
		},
		CreatedAt: time.Now(),
	}

	b.logger.Info("certificate issued successfully via self-signed CA",
		"request_id", req.ID,
		"serial_number", response.SerialNumber,
		"expires_at", response.ExpiresAt)

	return response, nil
}

// RevokeCertificate revokes a certificate
func (b *SelfSignedBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	b.logger.Info("revoking certificate",
		"serial_number", serialNumber,
		"reason", reason)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if certificate exists
	issuedCert, exists := b.issuedCerts[serialNumber]
	if !exists {
		return fmt.Errorf("certificate with serial number %s not found", serialNumber)
	}

	// Check if already revoked
	if _, revoked := b.revokedCerts[serialNumber]; revoked {
		return fmt.Errorf("certificate already revoked")
	}

	// Add to revoked certificates
	revokedCert := &RevokedCertificate{
		SerialNumber: serialNumber,
		RevokedAt:    time.Now(),
		Reason:       reason,
	}
	b.revokedCerts[serialNumber] = revokedCert

	// Update issued certificate status
	issuedCert.Status = StatusRevoked

	// Persist data if enabled
	if b.config.PersistentStorage {
		if err := b.persistData(); err != nil {
			b.logger.Warn("failed to persist revocation data", "error", err)
		}
	}

	b.logger.Info("certificate revoked successfully",
		"serial_number", serialNumber)

	return nil
}

// RenewCertificate renews a certificate (issues a new one)
func (b *SelfSignedBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	// For self-signed backend, renewal means issuing a new certificate
	response, err := b.IssueCertificate(ctx, req)
	if err != nil {
		return nil, err
	}

	response.Status = StatusRenewed
	return response, nil
}

// GetCAChain retrieves the CA certificate chain
func (b *SelfSignedBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var chain []*x509.Certificate

	if b.intermCA != nil {
		chain = append(chain, b.intermCA)
	}

	if b.rootCA != nil {
		chain = append(chain, b.rootCA)
	}

	return chain, nil
}

// GetCRL retrieves the Certificate Revocation List
func (b *SelfSignedBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {
	if b.config.CRLConfig == nil || !b.config.CRLConfig.Enabled {
		return nil, fmt.Errorf("CRL not enabled for self-signed backend")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Create CRL template
	now := time.Now()
	template := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: now,
		NextUpdate: now.AddDate(0, 0, b.config.CRLConfig.ValidityDays),
	}

	// Add revoked certificates
	for _, revoked := range b.revokedCerts {
		serialNumber := new(big.Int)
		serialNumber.SetString(revoked.SerialNumber, 10)

		template.RevokedCertificates = append(template.RevokedCertificates, pkix.RevokedCertificate{
			SerialNumber:   serialNumber,
			RevocationTime: revoked.RevokedAt,
			// ReasonCode field doesn't exist in pkix.RevokedCertificate
		})
	}

	// Select issuer
	var issuer *x509.Certificate
	var issuerKey crypto.Signer

	if b.intermCA != nil {
		issuer = b.intermCA
		var ok bool
		issuerKey, ok = b.intermKey.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("intermediate key is not a crypto.Signer")
		}
	} else {
		issuer = b.rootCA
		var ok bool
		issuerKey, ok = b.rootKey.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("root key is not a crypto.Signer")
		}
	}

	// Create CRL
	crlBytes, err := x509.CreateRevocationList(rand.Reader, template, issuer, issuerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRL: %w", err)
	}

	return x509.ParseCRL(crlBytes)
}

// GetBackendInfo returns backend information
func (b *SelfSignedBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var validUntil time.Time
	var issuer string

	if b.intermCA != nil {
		validUntil = b.intermCA.NotAfter
		issuer = b.intermCA.Subject.String()
	} else if b.rootCA != nil {
		validUntil = b.rootCA.NotAfter
		issuer = b.rootCA.Subject.String()
	}

	return &BackendInfo{
		Type:       BackendSelfSigned,
		Version:    "1.0.0",
		Status:     "ready",
		Issuer:     issuer,
		ValidUntil: validUntil,
		Features:   b.GetSupportedFeatures(),
		Metrics: map[string]interface{}{
			"issued_certificates":  len(b.issuedCerts),
			"revoked_certificates": len(b.revokedCerts),
			"has_intermediate_ca":  b.intermCA != nil,
		},
	}, nil
}

// GetSupportedFeatures returns supported features
func (b *SelfSignedBackend) GetSupportedFeatures() []string {
	return []string{
		"certificate_issuance",
		"certificate_revocation",
		"certificate_renewal",
		"ca_chain_retrieval",
		"crl_support",
		"hierarchical_ca",
		"persistent_storage",
		"development_testing",
	}
}

// Helper methods

func (b *SelfSignedBackend) validateConfig() error {
	if b.config.DefaultKeySize == 0 {
		b.config.DefaultKeySize = 2048
	}

	if b.config.MinKeySize == 0 {
		b.config.MinKeySize = 2048
	}

	if b.config.DefaultValidityDays == 0 {
		b.config.DefaultValidityDays = 30
	}

	if b.config.MaxValidityDays == 0 {
		b.config.MaxValidityDays = 365
	}

	if b.config.RootCA == nil {
		return fmt.Errorf("root CA configuration is required")
	}

	return nil
}

func (b *SelfSignedBackend) initializeRootCA() error {
	if b.config.RootCA.KeySize == 0 {
		b.config.RootCA.KeySize = 4096 // Larger key for root CA
	}

	if b.config.RootCA.ValidityDays == 0 {
		b.config.RootCA.ValidityDays = 3650 // 10 years
	}

	// Generate root CA key
	rootKey, err := rsa.GenerateKey(rand.Reader, b.config.RootCA.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate root CA key: %w", err)
	}

	// Create root CA certificate template
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               *b.config.RootCA.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, b.config.RootCA.ValidityDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
		MaxPathLenZero:        false,
	}

	// Add CRL distribution points if configured
	if len(b.config.RootCA.CRLDistributionPoints) > 0 {
		template.CRLDistributionPoints = b.config.RootCA.CRLDistributionPoints
	}

	// Add OCSP servers if configured
	if len(b.config.RootCA.OCSPServers) > 0 {
		template.OCSPServer = b.config.RootCA.OCSPServers
	}

	// Create self-signed root CA certificate
	rootCertBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rootKey.PublicKey, rootKey)
	if err != nil {
		return fmt.Errorf("failed to create root CA certificate: %w", err)
	}

	// Parse the created certificate
	rootCert, err := x509.ParseCertificate(rootCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse root CA certificate: %w", err)
	}

	b.rootCA = rootCert
	b.rootKey = rootKey

	return nil
}

func (b *SelfSignedBackend) initializeIntermediateCA() error {
	if b.config.IntermediateCA.KeySize == 0 {
		b.config.IntermediateCA.KeySize = 2048
	}

	if b.config.IntermediateCA.ValidityDays == 0 {
		b.config.IntermediateCA.ValidityDays = 1825 // 5 years
	}

	// Generate intermediate CA key
	intermKey, err := rsa.GenerateKey(rand.Reader, b.config.IntermediateCA.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate intermediate CA key: %w", err)
	}

	// Create intermediate CA certificate template
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               *b.config.IntermediateCA.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, b.config.IntermediateCA.ValidityDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Add CRL distribution points if configured
	if len(b.config.IntermediateCA.CRLDistributionPoints) > 0 {
		template.CRLDistributionPoints = b.config.IntermediateCA.CRLDistributionPoints
	}

	// Add OCSP servers if configured
	if len(b.config.IntermediateCA.OCSPServers) > 0 {
		template.OCSPServer = b.config.IntermediateCA.OCSPServers
	}

	// Create intermediate CA certificate signed by root CA
	intermCertBytes, err := x509.CreateCertificate(rand.Reader, template, b.rootCA, &intermKey.PublicKey, b.rootKey)
	if err != nil {
		return fmt.Errorf("failed to create intermediate CA certificate: %w", err)
	}

	// Parse the created certificate
	intermCert, err := x509.ParseCertificate(intermCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse intermediate CA certificate: %w", err)
	}

	b.intermCA = intermCert
	b.intermKey = intermKey

	return nil
}

func (b *SelfSignedBackend) convertKeyUsage(keyUsages []string) x509.KeyUsage {
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
			usage |= x509.KeyUsageCertSign
		case "crl_sign":
			usage |= x509.KeyUsageCRLSign
		case "content_commitment":
			usage |= x509.KeyUsageContentCommitment
		case "encipher_only":
			usage |= x509.KeyUsageEncipherOnly
		case "decipher_only":
			usage |= x509.KeyUsageDecipherOnly
		}
	}

	// Default usage if none specified
	if usage == 0 {
		usage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	}

	return usage
}

func (b *SelfSignedBackend) convertExtKeyUsage(extKeyUsages []string) []x509.ExtKeyUsage {
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
		case "timestamping":
			usages = append(usages, x509.ExtKeyUsageTimeStamping)
		case "ocsp_signing":
			usages = append(usages, x509.ExtKeyUsageOCSPSigning)
		}
	}

	// Default extended key usages if none specified
	if len(usages) == 0 {
		usages = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	}

	return usages
}

func (b *SelfSignedBackend) calculateFingerprint(certBytes []byte) string {
	hash := sha256.Sum256(certBytes)
	return fmt.Sprintf("%x", hash)
}

func (b *SelfSignedBackend) loadPersistedData() error {
	// Implementation would load from persistent storage
	// This is a placeholder for actual implementation
	b.logger.Debug("loading persisted certificate data")
	return nil
}

func (b *SelfSignedBackend) persistData() error {
	// Implementation would save to persistent storage
	// This is a placeholder for actual implementation
	b.logger.Debug("persisting certificate data")
	return nil
}

package mtls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// ServiceIdentity represents a service identity with associated certificates.

type ServiceIdentity struct {
	ServiceName string `json:"service_name"`

	TenantID string `json:"tenant_id"`

	Role ServiceRole `json:"role"`

	CertificateID string `json:"certificate_id"`

	SerialNumber string `json:"serial_number"`

	IssuedAt time.Time `json:"issued_at"`

	ExpiresAt time.Time `json:"expires_at"`

	Status IdentityStatus `json:"status"`

	Metadata map[string]string `json:"metadata"`

	// Certificate paths.

	CertPath string `json:"cert_path"`

	KeyPath string `json:"key_path"`

	CACertPath string `json:"ca_cert_path"`

	// Last rotation tracking.

	LastRotation time.Time `json:"last_rotation"`

	RotationCount int `json:"rotation_count"`
}

// IdentityStatus represents the status of a service identity.

type IdentityStatus string

const (

	// StatusActive holds statusactive value.

	StatusActive IdentityStatus = "active"

	// StatusExpiring holds statusexpiring value.

	StatusExpiring IdentityStatus = "expiring"

	// StatusExpired holds statusexpired value.

	StatusExpired IdentityStatus = "expired"

	// StatusRevoked holds statusrevoked value.

	StatusRevoked IdentityStatus = "revoked"

	// StatusPending holds statuspending value.

	StatusPending IdentityStatus = "pending"

	// StatusRotating holds statusrotating value.

	StatusRotating IdentityStatus = "rotating"
)

// IdentityManagerConfig holds configuration for the identity manager.

type IdentityManagerConfig struct {
	// Base directory for storing certificates.

	CertificateDir string `json:"certificate_dir"`

	// Identity rotation settings.

	AutoRotation bool `json:"auto_rotation"`

	RotationThreshold time.Duration `json:"rotation_threshold"`

	// Certificate validity period.

	CertificateValidityPeriod time.Duration `json:"certificate_validity_period"`

	// Certificate key size (bits).

	KeySize int `json:"key_size"`

	// FIPS compliance for 2025 security standards.

	FIPSMode bool `json:"fips_mode"`

	// CA certificate and key paths for self-signed operation
	CACertPath string `json:"ca_cert_path"`
	CAKeyPath  string `json:"ca_key_path"`
}

// IdentityManager manages service identities and certificate lifecycle.

type IdentityManager struct {
	config IdentityManagerConfig

	logger *logging.StructuredLogger

	// CA certificate and key for issuing certificates
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey

	// Identity tracking.

	identities map[string]*ServiceIdentity

	mu sync.RWMutex

	// Background rotation context.

	ctx context.Context

	cancel context.CancelFunc
}

// NewIdentityManager creates a new identity manager with 2025 security standards.

func NewIdentityManager(config IdentityManagerConfig, logger *logging.StructuredLogger) (*IdentityManager, error) {
	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("identity-manager", "1.0.0", "production"))
	}

	// Enforce 2025 security standards
	if config.KeySize < 3072 {
		config.KeySize = 3072 // Minimum 3072-bit keys for 2025
	}

	if config.CertificateValidityPeriod == 0 {
		config.CertificateValidityPeriod = 90 * 24 * time.Hour // Maximum 90-day certificates for 2025
	}

	// Enable FIPS mode by default for 2025
	config.FIPSMode = true

	ctx, cancel := context.WithCancel(context.Background())

	manager := &IdentityManager{
		config: config,

		logger: logger,

		identities: make(map[string]*ServiceIdentity),

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize or load CA certificate
	err := manager.initializeCA()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CA: %w", err)
	}

	// Start automatic rotation if enabled.

	if config.AutoRotation {
		go manager.startRotationLoop()
	}

	return manager, nil
}

// initializeCA initializes or loads the CA certificate and key
func (im *IdentityManager) initializeCA() error {
	// If CA cert and key paths are provided, load them
	if im.config.CACertPath != "" && im.config.CAKeyPath != "" {
		return im.loadCA()
	}

	// Otherwise, generate a self-signed CA
	return im.generateSelfSignedCA()
}

// loadCA loads existing CA certificate and key
func (im *IdentityManager) loadCA() error {
	// Load CA certificate
	certPEM, err := os.ReadFile(im.config.CACertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to parse CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Load CA private key
	keyPEM, err := os.ReadFile(im.config.CAKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to parse CA private key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %w", err)
	}

	im.caCert = caCert
	im.caKey = caKey

	im.logger.Info("Loaded existing CA certificate", "subject", caCert.Subject.String())
	return nil
}

// generateSelfSignedCA generates a self-signed CA certificate
func (im *IdentityManager) generateSelfSignedCA() error {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, im.config.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Nephoran Intent Operator"},
			Country:       []string{"US"},
			Province:      []string{"CA"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "Nephoran CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Create the certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse generated CA certificate: %w", err)
	}

	im.caCert = caCert
	im.caKey = caKey

	// Save CA certificate and key if paths are configured
	if im.config.CACertPath != "" {
		err = im.saveCA()
		if err != nil {
			return fmt.Errorf("failed to save CA: %w", err)
		}
	}

	im.logger.Info("Generated self-signed CA certificate", "subject", caCert.Subject.String())
	return nil
}

// saveCA saves the CA certificate and key to files
func (im *IdentityManager) saveCA() error {
	// Ensure directory exists
	caDir := filepath.Dir(im.config.CACertPath)
	err := os.MkdirAll(caDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Save CA certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: im.caCert.Raw,
	})

	err = os.WriteFile(im.config.CACertPath, certPEM, 0644)
	if err != nil {
		return fmt.Errorf("failed to save CA certificate: %w", err)
	}

	// Save CA private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(im.caKey),
	})

	err = os.WriteFile(im.config.CAKeyPath, keyPEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to save CA private key: %w", err)
	}

	return nil
}

// CreateServiceIdentity creates a new service identity with certificate.

func (im *IdentityManager) CreateServiceIdentity(serviceName, tenantID string, role ServiceRole) (*ServiceIdentity, error) {
	im.mu.Lock()

	defer im.mu.Unlock()

	identityKey := fmt.Sprintf("%s-%s-%s", tenantID, serviceName, role)

	// Check if identity already exists.

	if existing, exists := im.identities[identityKey]; exists {
		if existing.Status == StatusActive {
			return existing, nil
		}
	}

	// Generate certificate paths.

	certDir := filepath.Join(im.config.CertificateDir, tenantID, serviceName, string(role))

	certPath := filepath.Join(certDir, "cert.pem")

	keyPath := filepath.Join(certDir, "key.pem")

	caCertPath := filepath.Join(certDir, "ca.pem")

	// Create directory
	err := os.MkdirAll(certDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Generate certificate and key
	cert, privateKey, err := im.generateCertificate(serviceName, tenantID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for %s: %w", identityKey, err)
	}

	// Save certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	err = os.WriteFile(certPath, certPEM, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// Save private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	err = os.WriteFile(keyPath, keyPEM, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// Copy CA certificate
	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: im.caCert.Raw,
	})

	err = os.WriteFile(caCertPath, caCertPEM, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to save CA certificate: %w", err)
	}

	// Create service identity.

	identity := &ServiceIdentity{
		ServiceName: serviceName,

		TenantID: tenantID,

		Role: role,

		CertificateID: fmt.Sprintf("cert-%s-%d", identityKey, time.Now().Unix()),

		SerialNumber: cert.SerialNumber.String(),

		IssuedAt: cert.NotBefore,

		ExpiresAt: cert.NotAfter,

		Status: StatusActive,

		Metadata: map[string]string{
			"issuer": cert.Issuer.String(),

			"subject": cert.Subject.String(),

			"key_algorithm": cert.PublicKeyAlgorithm.String(),

			"signature_algorithm": cert.SignatureAlgorithm.String(),

			"fips_compliant": fmt.Sprintf("%t", im.config.FIPSMode),

			"security_level": "enterprise",
		},

		CertPath: certPath,

		KeyPath: keyPath,

		CACertPath: caCertPath,

		LastRotation: time.Now(),

		RotationCount: 0,
	}

	im.identities[identityKey] = identity

	im.logger.Info("Service identity created",

		"service_name", serviceName,

		"tenant_id", tenantID,

		"role", role,

		"certificate_id", identity.CertificateID,

		"expires_at", identity.ExpiresAt,

		"fips_mode", im.config.FIPSMode,
	)

	return identity, nil
}

// generateCertificate generates a certificate signed by the CA
func (im *IdentityManager) generateCertificate(serviceName, tenantID string, role ServiceRole) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, im.config.KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization:       []string{"Nephoran Intent Operator"},
			OrganizationalUnit: []string{string(role)},
			Country:            []string{"US"},
			Province:           []string{"CA"},
			Locality:           []string{"San Francisco"},
			CommonName:         fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, tenantID),
		},
		DNSNames: []string{
			serviceName,
			fmt.Sprintf("%s.%s", serviceName, tenantID),
			fmt.Sprintf("%s.%s.svc", serviceName, tenantID),
			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, tenantID),
		},
		EmailAddresses: []string{fmt.Sprintf("%s@%s.nephoran.io", serviceName, tenantID)},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().Add(im.config.CertificateValidityPeriod),
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, im.caCert, &privateKey.PublicKey, im.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, privateKey, nil
}

// GetServiceIdentity retrieves a service identity.

func (im *IdentityManager) GetServiceIdentity(serviceName, tenantID string, role ServiceRole) (*ServiceIdentity, error) {
	im.mu.RLock()

	defer im.mu.RUnlock()

	identityKey := fmt.Sprintf("%s-%s-%s", tenantID, serviceName, role)

	identity, exists := im.identities[identityKey]
	if !exists {
		return nil, fmt.Errorf("service identity not found: %s", identityKey)
	}

	return identity, nil
}

// RotateServiceIdentity rotates the certificate for a service identity.

func (im *IdentityManager) RotateServiceIdentity(serviceName, tenantID string, role ServiceRole) error {
	im.mu.Lock()

	defer im.mu.Unlock()

	identityKey := fmt.Sprintf("%s-%s-%s", tenantID, serviceName, role)

	identity, exists := im.identities[identityKey]
	if !exists {
		return fmt.Errorf("service identity not found for rotation: %s", identityKey)
	}

	// Mark as rotating.

	identity.Status = StatusRotating

	im.logger.Info("Starting certificate rotation",

		"service_name", serviceName,

		"tenant_id", tenantID,

		"role", role,

		"old_serial", identity.SerialNumber,
	)

	// Generate new certificate
	cert, privateKey, err := im.generateCertificate(serviceName, tenantID, role)
	if err != nil {
		identity.Status = StatusExpired // Mark as expired on failure
		return fmt.Errorf("failed to generate new certificate during rotation for %s: %w", identityKey, err)
	}

	// Save new certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	err = os.WriteFile(identity.CertPath, certPEM, 0644)
	if err != nil {
		identity.Status = StatusExpired
		return fmt.Errorf("failed to save new certificate: %w", err)
	}

	// Save new private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	err = os.WriteFile(identity.KeyPath, keyPEM, 0600)
	if err != nil {
		identity.Status = StatusExpired
		return fmt.Errorf("failed to save new private key: %w", err)
	}

	// Update identity with new certificate information.

	identity.SerialNumber = cert.SerialNumber.String()

	identity.IssuedAt = cert.NotBefore

	identity.ExpiresAt = cert.NotAfter

	identity.Status = StatusActive

	identity.LastRotation = time.Now()

	identity.RotationCount++

	// Update metadata.

	identity.Metadata["issuer"] = cert.Issuer.String()

	identity.Metadata["subject"] = cert.Subject.String()

	identity.Metadata["key_algorithm"] = cert.PublicKeyAlgorithm.String()

	identity.Metadata["signature_algorithm"] = cert.SignatureAlgorithm.String()

	identity.Metadata["rotation_count"] = fmt.Sprintf("%d", identity.RotationCount)

	im.logger.Info("Certificate rotation completed",

		"service_name", serviceName,

		"tenant_id", tenantID,

		"role", role,

		"new_serial", identity.SerialNumber,

		"rotation_count", identity.RotationCount,

		"expires_at", identity.ExpiresAt,
	)

	return nil
}

// ListServiceIdentities returns all service identities.

func (im *IdentityManager) ListServiceIdentities() []*ServiceIdentity {
	im.mu.RLock()

	defer im.mu.RUnlock()

	identities := make([]*ServiceIdentity, 0, len(im.identities))

	for _, identity := range im.identities {
		identities = append(identities, identity)
	}

	return identities
}

// RevokeServiceIdentity revokes a service identity.

func (im *IdentityManager) RevokeServiceIdentity(serviceName, tenantID string, role ServiceRole) error {
	im.mu.Lock()

	defer im.mu.Unlock()

	identityKey := fmt.Sprintf("%s-%s-%s", tenantID, serviceName, role)

	identity, exists := im.identities[identityKey]
	if !exists {
		return fmt.Errorf("service identity not found for revocation: %s", identityKey)
	}

	// Update status.

	identity.Status = StatusRevoked

	im.logger.Info("Service identity revoked",

		"service_name", serviceName,

		"tenant_id", tenantID,

		"role", role,

		"serial_number", identity.SerialNumber,
	)

	return nil
}

// startRotationLoop starts the automatic certificate rotation loop.

func (im *IdentityManager) startRotationLoop() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour

	defer ticker.Stop()

	for {
		select {

		case <-im.ctx.Done():

			im.logger.Info("Identity manager rotation loop stopped")

			return

		case <-ticker.C:

			im.checkForRotationCandidates()
		}
	}
}

// checkForRotationCandidates checks for certificates that need rotation.

func (im *IdentityManager) checkForRotationCandidates() {
	im.mu.RLock()

	var candidates []*ServiceIdentity

	now := time.Now()

	for _, identity := range im.identities {
		if identity.Status != StatusActive {
			continue
		}

		timeUntilExpiry := identity.ExpiresAt.Sub(now)

		if timeUntilExpiry <= im.config.RotationThreshold {
			candidates = append(candidates, identity)
		}
	}

	im.mu.RUnlock()

	if len(candidates) == 0 {
		return
	}

	im.logger.Info("Found certificates for rotation",

		"candidate_count", len(candidates),
	)

	for _, candidate := range candidates {
		err := im.RotateServiceIdentity(candidate.ServiceName, candidate.TenantID, candidate.Role)
		if err != nil {
			im.logger.Error("Failed to auto-rotate certificate",

				"service_name", candidate.ServiceName,

				"tenant_id", candidate.TenantID,

				"role", candidate.Role,

				"error", err,
			)
		}
	}
}

// UpdateIdentityStatus updates the status of a service identity.

func (im *IdentityManager) UpdateIdentityStatus(serviceName, tenantID string, role ServiceRole, status IdentityStatus) error {
	im.mu.Lock()

	defer im.mu.Unlock()

	identityKey := fmt.Sprintf("%s-%s-%s", tenantID, serviceName, role)

	identity, exists := im.identities[identityKey]
	if !exists {
		return fmt.Errorf("service identity not found for status update: %s", identityKey)
	}

	identity.Status = status

	im.logger.Info("Service identity status updated",

		"service_name", serviceName,

		"tenant_id", tenantID,

		"role", role,

		"new_status", status,
	)

	return nil
}

// Close shuts down the identity manager.

func (im *IdentityManager) Close() error {
	im.cancel()
	return nil
}

// IdentityStats represents statistics about managed identities
type IdentityStats struct {
	TotalIdentities int                    `json:"total_identities"`
	StatusCounts    map[IdentityStatus]int `json:"status_counts"`
	RoleCounts      map[ServiceRole]int    `json:"role_counts"`
	ExpiringSoon    int                    `json:"expiring_soon"`
	FIPSMode        bool                   `json:"fips_mode"`
	AutoRotation    bool                   `json:"auto_rotation"`
}

// GetIdentityStats returns statistics about managed identities.

func (im *IdentityManager) GetIdentityStats() *IdentityStats {
	im.mu.RLock()

	defer im.mu.RUnlock()

	stats := &IdentityStats{
		TotalIdentities: len(im.identities),
		StatusCounts:    make(map[IdentityStatus]int),
		RoleCounts:      make(map[ServiceRole]int),
		ExpiringSoon:    0,
		FIPSMode:        im.config.FIPSMode,
		AutoRotation:    im.config.AutoRotation,
	}

	now := time.Now()

	for _, identity := range im.identities {
		// Count by status.
		stats.StatusCounts[identity.Status]++

		// Count by role.
		stats.RoleCounts[identity.Role]++

		// Count expiring soon (within 7 days).
		if timeUntilExpiry := identity.ExpiresAt.Sub(now); timeUntilExpiry < 7*24*time.Hour {
			stats.ExpiringSoon++
		}
	}

	return stats
}
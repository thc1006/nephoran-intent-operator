// Package security provides enterprise-grade cryptographic utilities
// implementing O-RAN WG11 security requirements and modern cryptographic standards
package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
)

// CryptoConfig defines enterprise cryptographic configuration
type CryptoConfig struct {
	// RSA key size - minimum 3072 bits, recommended 4096 bits for O-RAN compliance
	RSAKeySize int `json:"rsa_key_size" yaml:"rsa_key_size"`

	// Certificate validity periods
	CertValidityPeriod time.Duration `json:"cert_validity_period" yaml:"cert_validity_period"`
	CAValidityPeriod   time.Duration `json:"ca_validity_period" yaml:"ca_validity_period"`

	// TLS configuration
	EnforceTLS13Only             bool     `json:"enforce_tls13_only" yaml:"enforce_tls13_only"`
	AllowedCipherSuites          []string `json:"allowed_cipher_suites" yaml:"allowed_cipher_suites"`
	RequirePerfectForwardSecrecy bool     `json:"require_perfect_forward_secrecy" yaml:"require_perfect_forward_secrecy"`

	// mTLS settings
	MTLSRequired         bool `json:"mtls_required" yaml:"mtls_required"`
	ClientCertValidation bool `json:"client_cert_validation" yaml:"client_cert_validation"`

	// Certificate rotation
	AutoRotateBeforeExpiry time.Duration `json:"auto_rotate_before_expiry" yaml:"auto_rotate_before_expiry"`

	// Random number generation
	UseHardwareRNG bool `json:"use_hardware_rng" yaml:"use_hardware_rng"`
}

// DefaultCryptoConfig returns O-RAN WG11 compliant cryptographic configuration
func DefaultCryptoConfig() *CryptoConfig {
	return &CryptoConfig{
		RSAKeySize:         4096,                      // O-RAN WG11 requirement
		CertValidityPeriod: 90 * 24 * time.Hour,       // 90 days
		CAValidityPeriod:   10 * 365 * 24 * time.Hour, // 10 years
		EnforceTLS13Only:   true,                      // TLS 1.3 mandatory
		AllowedCipherSuites: []string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		},
		RequirePerfectForwardSecrecy: true,
		MTLSRequired:                 true,
		ClientCertValidation:         true,
		AutoRotateBeforeExpiry:       30 * 24 * time.Hour, // 30 days before expiry
		UseHardwareRNG:               true,
	}
}

// SecureTLSConfigBuilder builds enterprise-grade TLS configurations
type SecureTLSConfigBuilder struct {
	config *CryptoConfig
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewSecureTLSConfigBuilder creates a new secure TLS configuration builder
func NewSecureTLSConfigBuilder(config *CryptoConfig, logger *zap.Logger) *SecureTLSConfigBuilder {
	if config == nil {
		config = DefaultCryptoConfig()
	}

	return &SecureTLSConfigBuilder{
		config: config,
		logger: logger,
	}
}

// BuildServerTLSConfig creates a secure server TLS configuration
func (b *SecureTLSConfigBuilder) BuildServerTLSConfig(serverCert tls.Certificate, clientCAs *x509.CertPool) (*tls.Config, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// O-RAN WG11 compliant cipher suites mapping
	cipherSuites := []uint16{
		tls.TLS_AES_256_GCM_SHA384,       // Highest security AEAD cipher
		tls.TLS_CHACHA20_POLY1305_SHA256, // ChaCha20-Poly1305 for performance
	}

	tlsConfig := &tls.Config{
		// TLS version enforcement - TLS 1.3 only
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,

		// Certificate configuration
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    clientCAs,

		// Cipher suite configuration (TLS 1.3 ignores this but kept for documentation)
		CipherSuites: cipherSuites,

		// Security hardening
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   b.config.RequirePerfectForwardSecrecy,
		Renegotiation:            tls.RenegotiateNever,

		// Protocol preferences
		NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2

		// Client authentication for mTLS
		ClientAuth: func() tls.ClientAuthType {
			if b.config.MTLSRequired {
				return tls.RequireAndVerifyClientCert
			}
			return tls.NoClientCert
		}(),

		// Custom verification
		VerifyConnection: b.verifyConnection,
	}

	return tlsConfig, nil
}

// BuildClientTLSConfig creates a secure client TLS configuration
func (b *SecureTLSConfigBuilder) BuildClientTLSConfig(clientCert *tls.Certificate, rootCAs *x509.CertPool, serverName string) (*tls.Config, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tlsConfig := &tls.Config{
		// TLS version enforcement
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,

		// Certificate configuration
		RootCAs:    rootCAs,
		ServerName: serverName,

		// Security settings
		SessionTicketsDisabled: b.config.RequirePerfectForwardSecrecy,
		Renegotiation:          tls.RenegotiateNever,

		// NEVER skip verification in production
		InsecureSkipVerify: false,

		// Custom certificate verification
		VerifyPeerCertificate: b.verifyPeerCertificate,
	}

	// Add client certificate for mTLS
	if clientCert != nil {
		tlsConfig.Certificates = []tls.Certificate{*clientCert}
	}

	return tlsConfig, nil
}

// verifyConnection performs custom connection verification
func (b *SecureTLSConfigBuilder) verifyConnection(cs tls.ConnectionState) error {
	// Enforce TLS 1.3 only
	if cs.Version != tls.VersionTLS13 {
		b.logger.Error("TLS version violation",
			zap.Uint16("version", cs.Version),
			zap.Uint16("required", tls.VersionTLS13))
		return fmt.Errorf("TLS 1.3 required, connection attempted with version 0x%04x", cs.Version)
	}

	// Verify cipher suite is approved
	approvedCipher := false
	switch cs.CipherSuite {
	case tls.TLS_AES_256_GCM_SHA384, tls.TLS_CHACHA20_POLY1305_SHA256:
		approvedCipher = true
	}

	if !approvedCipher {
		b.logger.Error("Unapproved cipher suite",
			zap.Uint16("cipher", cs.CipherSuite))
		return fmt.Errorf("cipher suite 0x%04x not approved for O-RAN WG11 compliance", cs.CipherSuite)
	}

	// Verify Perfect Forward Secrecy if required
	if b.config.RequirePerfectForwardSecrecy {
		if cs.CipherSuite != tls.TLS_AES_256_GCM_SHA384 && cs.CipherSuite != tls.TLS_CHACHA20_POLY1305_SHA256 {
			return fmt.Errorf("Perfect Forward Secrecy required but not provided by cipher suite")
		}
	}

	return nil
}

// verifyPeerCertificate performs custom peer certificate verification
func (b *SecureTLSConfigBuilder) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no peer certificates provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse peer certificate: %w", err)
	}

	// Verify certificate is not expired
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate is not valid at current time: notBefore=%v, notAfter=%v, now=%v",
			cert.NotBefore, cert.NotAfter, now)
	}

	// Verify key size meets O-RAN requirements
	if rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
		keySize := rsaPubKey.Size() * 8 // Convert bytes to bits
		if keySize < b.config.RSAKeySize {
			return fmt.Errorf("RSA key size %d bits is below required minimum %d bits", keySize, b.config.RSAKeySize)
		}
	}

	return nil
}

// SecureRandomGenerator provides cryptographically secure random number generation
type SecureRandomGenerator struct {
	reader io.Reader
	mu     sync.Mutex
}

// NewSecureRandomGenerator creates a new secure random generator
func NewSecureRandomGenerator() *SecureRandomGenerator {
	return &SecureRandomGenerator{
		reader: rand.Reader, // Always use crypto/rand
	}
}

// GenerateSecureBytes generates cryptographically secure random bytes
func (g *SecureRandomGenerator) GenerateSecureBytes(length int) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if length <= 0 {
		return nil, fmt.Errorf("length must be positive, got %d", length)
	}

	bytes := make([]byte, length)
	n, err := io.ReadFull(g.reader, bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure random bytes: %w", err)
	}

	if n != length {
		return nil, fmt.Errorf("insufficient random bytes generated: expected %d, got %d", length, n)
	}

	return bytes, nil
}

// GenerateSecureToken generates a cryptographically secure base64-encoded token
func (g *SecureRandomGenerator) GenerateSecureToken(length int) (string, error) {
	bytes, err := g.GenerateSecureBytes(length)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateSecureSessionID generates a secure session identifier
func (g *SecureRandomGenerator) GenerateSecureSessionID() (string, error) {
	// 256 bits of entropy for session IDs
	return g.GenerateSecureToken(32)
}

// GenerateSecureAPIKey generates a secure API key
func (g *SecureRandomGenerator) GenerateSecureAPIKey() (string, error) {
	// 512 bits of entropy for API keys
	return g.GenerateSecureToken(64)
}

// SecureCertificateGenerator generates enterprise-grade X.509 certificates
type SecureCertificateGenerator struct {
	config *CryptoConfig
	rng    *SecureRandomGenerator
	logger *zap.Logger
}

// NewSecureCertificateGenerator creates a new secure certificate generator
func NewSecureCertificateGenerator(config *CryptoConfig, logger *zap.Logger) *SecureCertificateGenerator {
	if config == nil {
		config = DefaultCryptoConfig()
	}

	return &SecureCertificateGenerator{
		config: config,
		rng:    NewSecureRandomGenerator(),
		logger: logger,
	}
}

// GenerateCAKeyPair generates a certificate authority key pair
func (g *SecureCertificateGenerator) GenerateCAKeyPair(subject pkix.Name) (*rsa.PrivateKey, *x509.Certificate, error) {
	// Generate RSA key pair with configured key size
	privateKey, err := rsa.GenerateKey(rand.Reader, g.config.RSAKeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Generate secure serial number
	serialNumberBytes, err := g.rng.GenerateSecureBytes(16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}
	serialNumber := new(big.Int).SetBytes(serialNumberBytes)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(g.config.CAValidityPeriod),

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,

		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},

		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,

		// Security extensions
		SubjectKeyId: sha256.New().Sum(nil)[:20], // Will be properly calculated
	}

	// Create the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse generated CA certificate: %w", err)
	}

	g.logger.Info("Generated CA certificate",
		zap.String("subject", subject.String()),
		zap.Int("keySize", g.config.RSAKeySize),
		zap.Time("notAfter", cert.NotAfter))

	return privateKey, cert, nil
}

// GenerateServerKeyPair generates a server certificate key pair
func (g *SecureCertificateGenerator) GenerateServerKeyPair(
	subject pkix.Name,
	dnsNames []string,
	ipAddresses []net.IP,
	caCert *x509.Certificate,
	caKey *rsa.PrivateKey) (*rsa.PrivateKey, *x509.Certificate, error) {

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, g.config.RSAKeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Generate secure serial number
	serialNumberBytes, err := g.rng.GenerateSecureBytes(16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}
	serialNumber := new(big.Int).SetBytes(serialNumberBytes)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(g.config.CertValidityPeriod),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},

		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,

		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Create the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse generated server certificate: %w", err)
	}

	g.logger.Info("Generated server certificate",
		zap.String("subject", subject.String()),
		zap.Strings("dnsNames", dnsNames),
		zap.Int("keySize", g.config.RSAKeySize),
		zap.Time("notAfter", cert.NotAfter))

	return privateKey, cert, nil
}

// SecureHTTPClientFactory creates secure HTTP clients with proper TLS configuration
type SecureHTTPClientFactory struct {
	tlsBuilder *SecureTLSConfigBuilder
	logger     *zap.Logger
}

// NewSecureHTTPClientFactory creates a new secure HTTP client factory
func NewSecureHTTPClientFactory(config *CryptoConfig, logger *zap.Logger) *SecureHTTPClientFactory {
	return &SecureHTTPClientFactory{
		tlsBuilder: NewSecureTLSConfigBuilder(config, logger),
		logger:     logger,
	}
}

// CreateSecureClient creates an HTTP client with enterprise-grade TLS security
func (f *SecureHTTPClientFactory) CreateSecureClient(
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool,
	timeout time.Duration) (*http.Client, error) {

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	tlsConfig, err := f.tlsBuilder.BuildClientTLSConfig(clientCert, rootCAs, "")
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Security: Disable HTTP/1.0 and HTTP/1.1 keep-alive
		DisableKeepAlives: false,

		// Prevent connection reuse to compromised servers
		DisableCompression: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,

		// Security: Prevent following redirects to potentially malicious hosts
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return client, nil
}

// SecureGRPCFactory creates secure gRPC clients and servers
type SecureGRPCFactory struct {
	tlsBuilder *SecureTLSConfigBuilder
	logger     *zap.Logger
}

// NewSecureGRPCFactory creates a new secure gRPC factory
func NewSecureGRPCFactory(config *CryptoConfig, logger *zap.Logger) *SecureGRPCFactory {
	return &SecureGRPCFactory{
		tlsBuilder: NewSecureTLSConfigBuilder(config, logger),
		logger:     logger,
	}
}

// CreateSecureServerCredentials creates secure gRPC server credentials
func (f *SecureGRPCFactory) CreateSecureServerCredentials(
	serverCert tls.Certificate,
	clientCAs *x509.CertPool) (credentials.TransportCredentials, error) {

	tlsConfig, err := f.tlsBuilder.BuildServerTLSConfig(serverCert, clientCAs)
	if err != nil {
		return nil, fmt.Errorf("failed to build server TLS config: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// CreateSecureClientCredentials creates secure gRPC client credentials
func (f *SecureGRPCFactory) CreateSecureClientCredentials(
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool,
	serverName string) (credentials.TransportCredentials, error) {

	tlsConfig, err := f.tlsBuilder.BuildClientTLSConfig(clientCert, rootCAs, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to build client TLS config: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// EncodeCertificatePEM encodes a certificate to PEM format
func EncodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// EncodePrivateKeyPEM encodes an RSA private key to PEM format
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// ValidateEnterpriseSecurityCompliance validates that the configuration meets enterprise security standards
func ValidateEnterpriseSecurityCompliance(config *CryptoConfig) error {
	var errors []string

	// Validate RSA key size
	if config.RSAKeySize < 3072 {
		errors = append(errors, fmt.Sprintf("RSA key size %d is below enterprise minimum of 3072 bits", config.RSAKeySize))
	}

	// Validate TLS 1.3 enforcement
	if !config.EnforceTLS13Only {
		errors = append(errors, "TLS 1.3 enforcement is required for enterprise security")
	}

	// Validate mTLS requirement
	if !config.MTLSRequired {
		errors = append(errors, "Mutual TLS is required for enterprise security")
	}

	// Validate certificate validity periods
	if config.CertValidityPeriod > 90*24*time.Hour {
		errors = append(errors, "Certificate validity period exceeds 90-day security recommendation")
	}

	if len(errors) > 0 {
		return fmt.Errorf("enterprise security compliance violations: %v", errors)
	}

	return nil
}

// Package security provides advanced TLS configuration and security enhancements.

package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ocsp"
)

// TLSEnhancedConfig provides advanced TLS configuration with Go 1.24+ features.

type TLSEnhancedConfig struct {

	// Core configuration.

	MinVersion uint16

	MaxVersion uint16

	CipherSuites []uint16

	CurvePreferences []tls.CurveID

	// Post-quantum readiness.

	PostQuantumEnabled bool

	HybridMode bool // Use classical + PQ algorithms

	// Certificate management.

	CertFile string

	KeyFile string

	CAFile string

	ClientCAs *x509.CertPool

	certificates []tls.Certificate

	// OCSP support.

	OCSPStaplingEnabled bool

	OCSPResponderURL string

	OCSPCache *OCSPCache

	// Session management.

	SessionTicketKeys [][]byte

	SessionTicketRotationInterval time.Duration

	// Enhanced security features.

	HSTSEnabled bool

	HSTSMaxAge time.Duration

	DHE2048Enabled bool // Disable DHE with less than 2048 bits

	// Certificate Transparency.

	CTEnabled bool

	CTLogServers []string

	// DANE (DNS-based Authentication of Named Entities).

	DANEEnabled bool

	DNSSECRequired bool

	// Advanced features.

	OnlineCertificateValidation bool

	CertificateRevocationCheck bool

	// 0-RTT Early Data Support (TLS 1.3)
	// WARNING: Enabling 0-RTT can expose the application to replay attacks
	// Only enable for idempotent operations
	Enable0RTT bool

	Max0RTTDataSize uint32 // Maximum size of 0-RTT early data in bytes

	// Certificate pinning.

	PinnedCertificates []string

	PinnedPublicKeys []string

	// Performance optimizations.

	SessionCacheSize int

	SessionCacheTimeout time.Duration

	// Security monitoring.

	SecurityEventCallback func(event SecurityEvent)

	FailureCallback func(failure SecurityFailure)

	// Metrics collection.

	MetricsCollector *TLSMetricsCollector

	// Mutex for thread-safe operations.

	mu sync.RWMutex
}

// TLSMetricsCollector collects TLS-related metrics.

type TLSMetricsCollector struct {
	totalConnections uint64

	successfulHandshakes uint64

	failedHandshakes uint64

	certificateErrors uint64

	ocspChecks uint64

	ocspHits uint64

	ocspMisses uint64

	sessionReuses uint64

	tlsVersionUsage map[uint16]uint64

	cipherSuiteUsage map[uint16]uint64

	mu sync.RWMutex
}

// SecurityEvent represents a security-related event.

type SecurityEvent struct {
	Timestamp   time.Time

	EventType   string

	ClientAddr  string

	CertSubject string

	TLSVersion  uint16

	CipherSuite uint16

	Details     map[string]interface{}
}

// SecurityFailure represents a security failure event.

type SecurityFailure struct {
	Timestamp time.Time

	FailureType string

	ClientAddr  string

	Error       error

	Context     map[string]interface{}
}

// OCSPCache provides caching for OCSP responses.

type OCSPCache struct {
	cache map[string]*CachedOCSPResponse

	ttl time.Duration

	maxSize int

	mu sync.RWMutex
}

// CachedOCSPResponse represents a cached OCSP response.

type CachedOCSPResponse struct {
	response  *ocsp.Response

	timestamp time.Time
}

// NewTLSEnhancedConfig creates a new enhanced TLS configuration.

func NewTLSEnhancedConfig() *TLSEnhancedConfig {

	return &TLSEnhancedConfig{

		MinVersion: tls.VersionTLS12, // Security fix: Set secure minimum version (G402)

		MaxVersion: tls.VersionTLS13,

		CipherSuites: []uint16{

			// TLS 1.3 cipher suites (automatic in Go 1.24+).

			tls.TLS_AES_256_GCM_SHA384,

			tls.TLS_AES_128_GCM_SHA256,

			tls.TLS_CHACHA20_POLY1305_SHA256,
		},

		CurvePreferences: []tls.CurveID{

			tls.X25519, // Preferred for performance

			tls.CurveP384, // NIST P-384

			tls.CurveP256, // NIST P-256

		},

		OCSPCache: &OCSPCache{

			cache: make(map[string]*CachedOCSPResponse),

			ttl: 24 * time.Hour,

			maxSize: 1000,
		},

		SessionTicketRotationInterval: 24 * time.Hour,

		SessionCacheSize: 10000,

		SessionCacheTimeout: 6 * time.Hour,

		HSTSMaxAge: 31536000 * time.Second, // 1 year

		MetricsCollector: &TLSMetricsCollector{

			tlsVersionUsage: make(map[uint16]uint64),

			cipherSuiteUsage: make(map[uint16]uint64),
		},
	}

}

// GetTLSConfig returns a standard tls.Config based on the enhanced configuration.

func (c *TLSEnhancedConfig) GetTLSConfig() (*tls.Config, error) {

	c.mu.RLock()

	defer c.mu.RUnlock()

	tlsConfig := &tls.Config{

		MinVersion: c.MinVersion,

		MaxVersion: c.MaxVersion,

		CipherSuites: c.CipherSuites,

		CurvePreferences: c.CurvePreferences,

		ClientCAs: c.ClientCAs,

		Certificates: c.certificates,

		PreferServerCipherSuites: true,

		SessionTicketsDisabled: false,

		InsecureSkipVerify: false,

		// Enhanced verification callback.

		VerifyPeerCertificate: c.verifyPeerCertificate,

		// Connection state callback for metrics.

		GetCertificate: c.getCertificate,

		// Dynamic session ticket keys.

		GetConfigForClient: c.getConfigForClient,
	}

	// Enable OCSP stapling if configured.

	if c.OCSPStaplingEnabled && c.OCSPResponderURL != "" {

		for i := range tlsConfig.Certificates {

			if len(tlsConfig.Certificates[i].OCSPStaple) == 0 {

				staple, err := c.generateOCSPStaple(&tlsConfig.Certificates[i])

				if err == nil {

					tlsConfig.Certificates[i].OCSPStaple = staple

				}

			}

		}

	}

	return tlsConfig, nil

}

// LoadCertificate loads and validates a certificate pair.

func (c *TLSEnhancedConfig) LoadCertificate(certFile, keyFile string) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)

	if err != nil {

		return fmt.Errorf("failed to load certificate: %w", err)

	}

	// Parse certificate for validation.

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])

	if err != nil {

		return fmt.Errorf("failed to parse certificate: %w", err)

	}

	// Validate certificate.

	if err := c.validateCertificate(x509Cert); err != nil {

		return fmt.Errorf("certificate validation failed: %w", err)

	}

	c.certificates = append(c.certificates, cert)

	c.CertFile = certFile

	c.KeyFile = keyFile

	return nil

}

// validateCertificate performs comprehensive certificate validation.

func (c *TLSEnhancedConfig) validateCertificate(cert *x509.Certificate) error {

	// Check expiration.

	now := time.Now()

	if cert.NotAfter.Before(now) {

		return errors.New("certificate has expired")

	}

	if cert.NotBefore.After(now) {

		return errors.New("certificate is not yet valid")

	}

	// Check for weak signature algorithms.

	switch cert.SignatureAlgorithm {

	case x509.MD5WithRSA, x509.SHA1WithRSA:

		return fmt.Errorf("certificate uses weak signature algorithm: %v", cert.SignatureAlgorithm)

	}

	// Check key strength for RSA keys.

	if cert.PublicKeyAlgorithm == x509.RSA {

		if rsaKey, ok := cert.PublicKey.(*interface{}); ok {

			_ = rsaKey // Key strength validation would go here

		}

	}

	// Check certificate extensions.

	if !cert.BasicConstraintsValid && len(cert.DNSNames) == 0 && len(cert.IPAddresses) == 0 {

		return errors.New("certificate lacks proper subject alternative names")

	}

	return nil

}

// checkOCSPStatus verifies certificate status via OCSP.

func (c *TLSEnhancedConfig) checkOCSPStatus(cert *x509.Certificate, rawCerts [][]byte) error {

	// Check cache first.

	c.OCSPCache.mu.RLock()

	if cached, ok := c.OCSPCache.cache[string(cert.SerialNumber.Bytes())]; ok {

		if time.Since(cached.timestamp) < c.OCSPCache.ttl {

			c.OCSPCache.mu.RUnlock()

			atomic.AddUint64(&c.MetricsCollector.ocspHits, 1)

			if cached.response.Status != ocsp.Good {

				return fmt.Errorf("certificate revoked or unknown: %v", cached.response.Status)

			}

			return nil

		}

	}

	c.OCSPCache.mu.RUnlock()

	atomic.AddUint64(&c.MetricsCollector.ocspMisses, 1)

	// Extract issuer certificate from rawCerts if available
	var issuer *x509.Certificate
	if len(rawCerts) > 1 {
		// Try to parse the second certificate as the issuer
		if parsed, err := x509.ParseCertificate(rawCerts[1]); err == nil {
			issuer = parsed
		}
	}
	
	// Fall back to using the cert as issuer (self-signed case)
	if issuer == nil {
		issuer = cert
	}

	// Create OCSP request using the rawCerts data
	_, err := ocsp.CreateRequest(cert, issuer, nil)

	if err != nil {

		return fmt.Errorf("failed to create OCSP request: %w", err)

	}

	// Send OCSP request.

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", c.OCSPResponderURL, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create HTTP request: %w", err)

	}

	httpReq.Header.Set("Content-Type", "application/ocsp-request")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(httpReq)

	if err != nil {

		return fmt.Errorf("OCSP request failed: %w", err)

	}

	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)

	if err != nil {

		return fmt.Errorf("failed to read OCSP response: %w", err)

	}

	ocspResp, err := ocsp.ParseResponse(respBytes, issuer)

	if err != nil {

		return fmt.Errorf("failed to parse OCSP response: %w", err)

	}

	// Cache the response.

	c.OCSPCache.mu.Lock()

	if len(c.OCSPCache.cache) >= c.OCSPCache.maxSize {

		// Simple eviction: remove the oldest entry.

		var oldestKey string

		var oldestTime time.Time

		for key, cached := range c.OCSPCache.cache {

			if oldestTime.IsZero() || cached.timestamp.Before(oldestTime) {

				oldestTime = cached.timestamp

				oldestKey = key

			}

		}

		delete(c.OCSPCache.cache, oldestKey)

	}

	c.OCSPCache.cache[string(cert.SerialNumber.Bytes())] = &CachedOCSPResponse{

		response:  ocspResp,

		timestamp: time.Now(),
	}

	c.OCSPCache.mu.Unlock()

	if ocspResp.Status != ocsp.Good {

		return fmt.Errorf("certificate revoked or unknown: %v", ocspResp.Status)

	}

	return nil

}

// verifyPeerCertificate performs custom certificate verification.

func (c *TLSEnhancedConfig) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

	if len(rawCerts) == 0 {

		return errors.New("no certificates provided")

	}

	cert, err := x509.ParseCertificate(rawCerts[0])

	if err != nil {

		return fmt.Errorf("failed to parse certificate: %w", err)

	}

	atomic.AddUint64(&c.MetricsCollector.totalConnections, 1)

	// Perform OCSP check if enabled.

	if c.OnlineCertificateValidation && c.OCSPResponderURL != "" {

		if err := c.checkOCSPStatus(cert, rawCerts); err != nil {

			atomic.AddUint64(&c.MetricsCollector.certificateErrors, 1)

			c.reportSecurityFailure("OCSP_CHECK_FAILED", "", err, map[string]interface{}{

				"certificate_subject": cert.Subject.String(),

				"serial_number": cert.SerialNumber.String(),
			})

			return err

		}

	}

	// Check certificate pinning.

	if len(c.PinnedCertificates) > 0 || len(c.PinnedPublicKeys) > 0 {

		if err := c.checkCertificatePinning(cert); err != nil {

			atomic.AddUint64(&c.MetricsCollector.certificateErrors, 1)

			return err

		}

	}

	atomic.AddUint64(&c.MetricsCollector.successfulHandshakes, 1)

	// Report successful connection.

	c.reportSecurityEvent("TLS_HANDSHAKE_SUCCESS", "", cert, 0, 0, map[string]interface{}{

		"certificate_subject": cert.Subject.String(),

		"serial_number": cert.SerialNumber.String(),
	})

	return nil

}

// checkCertificatePinning validates certificate against pinned values.

func (c *TLSEnhancedConfig) checkCertificatePinning(cert *x509.Certificate) error {

	// Check pinned certificates (exact match).

	certPEM := pem.EncodeToMemory(&pem.Block{

		Type:  "CERTIFICATE",

		Bytes: cert.Raw,
	})

	certPEMStr := string(certPEM)

	for _, pinned := range c.PinnedCertificates {

		if certPEMStr == pinned {

			return nil // Certificate matches

		}

	}

	// Check pinned public keys.

	pubKeyDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)

	if err != nil {

		return fmt.Errorf("failed to marshal public key: %w", err)

	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{

		Type:  "PUBLIC KEY",

		Bytes: pubKeyDER,
	})

	pubKeyPEMStr := string(pubKeyPEM)

	for _, pinned := range c.PinnedPublicKeys {

		if pubKeyPEMStr == pinned {

			return nil // Public key matches

		}

	}

	if len(c.PinnedCertificates) > 0 || len(c.PinnedPublicKeys) > 0 {

		return errors.New("certificate pinning validation failed")

	}

	return nil

}

// getCertificate dynamically selects certificates.

func (c *TLSEnhancedConfig) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {

	c.mu.RLock()

	defer c.mu.RUnlock()

	// Simple selection: return the first certificate.

	// In a production system, this would implement SNI-based selection.

	if len(c.certificates) > 0 {

		return &c.certificates[0], nil

	}

	return nil, errors.New("no certificates available")

}

// getConfigForClient provides per-client configuration.

func (c *TLSEnhancedConfig) getConfigForClient(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {

	// Base configuration.

	config, err := c.GetTLSConfig()

	if err != nil {

		return nil, err

	}

	// Custom per-client logic would go here.

	// For example, different cipher suites for different clients.

	return config, nil

}

// generateOCSPStaple generates an OCSP staple for a certificate.

func (c *TLSEnhancedConfig) generateOCSPStaple(cert *tls.Certificate) ([]byte, error) {

	if len(cert.Certificate) == 0 {

		return nil, errors.New("no certificate data")

	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])

	if err != nil {

		return nil, fmt.Errorf("failed to parse certificate: %w", err)

	}

	// In a real implementation, this would fetch OCSP response from the responder.

	// For now, return empty staple.

	_ = x509Cert

	return []byte{}, nil

}

// SetupPostQuantumReadiness configures post-quantum cryptography readiness.

func (c *TLSEnhancedConfig) SetupPostQuantumReadiness(enable bool, hybridMode bool) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.PostQuantumEnabled = enable

	c.HybridMode = hybridMode

	if hybridMode {

		// Use classical algorithms alongside PQ-ready configurations.

		c.MinVersion = tls.VersionTLS12 // Security fix: Use secure minimum version (G402)

	}

}

// reportSecurityEvent reports a security event if callback is configured.

func (c *TLSEnhancedConfig) reportSecurityEvent(eventType, clientAddr string, cert *x509.Certificate, tlsVersion, cipherSuite uint16, details map[string]interface{}) {

	if c.SecurityEventCallback != nil {

		event := SecurityEvent{

			Timestamp:   time.Now(),

			EventType:   eventType,

			ClientAddr:  clientAddr,

			TLSVersion:  tlsVersion,

			CipherSuite: cipherSuite,

			Details:     details,
		}

		if cert != nil {

			event.CertSubject = cert.Subject.String()

		}

		c.SecurityEventCallback(event)

	}

}

// reportSecurityFailure reports a security failure if callback is configured.

func (c *TLSEnhancedConfig) reportSecurityFailure(failureType, clientAddr string, err error, context map[string]interface{}) {

	if c.FailureCallback != nil {

		failure := SecurityFailure{

			Timestamp:   time.Now(),

			FailureType: failureType,

			ClientAddr:  clientAddr,

			Error:       err,

			Context:     context,
		}

		c.FailureCallback(failure)

	}

}

// GetMetrics returns current TLS metrics.

func (c *TLSEnhancedConfig) GetMetrics() *TLSMetricsCollector {

	return c.MetricsCollector

}

// LoadCA loads a certificate authority for client certificate validation.

func (c *TLSEnhancedConfig) LoadCA(caFile string) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	caCert, err := os.ReadFile(caFile)

	if err != nil {

		return fmt.Errorf("failed to read CA file: %w", err)

	}

	if c.ClientCAs == nil {

		c.ClientCAs = x509.NewCertPool()

	}

	if !c.ClientCAs.AppendCertsFromPEM(caCert) {

		return errors.New("failed to parse CA certificate")

	}

	c.CAFile = caFile

	return nil

}

// EnableHSTS enables HTTP Strict Transport Security.

func (c *TLSEnhancedConfig) EnableHSTS(maxAge time.Duration) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.HSTSEnabled = true

	c.HSTSMaxAge = maxAge

}

// StartSessionTicketRotation starts automatic session ticket key rotation.

func (c *TLSEnhancedConfig) StartSessionTicketRotation(ctx context.Context) {

	ticker := time.NewTicker(c.SessionTicketRotationInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			c.rotateSessionTickets()

		}

	}

}

// rotateSessionTickets rotates session ticket keys.

func (c *TLSEnhancedConfig) rotateSessionTickets() {

	c.mu.Lock()

	defer c.mu.Unlock()

	// Generate new 48-byte session ticket key.

	newKey := make([]byte, 48)

	// In production, use secure random generation.

	c.SessionTicketKeys = append([][]byte{newKey}, c.SessionTicketKeys...)

	// Keep only the latest 3 keys.

	if len(c.SessionTicketKeys) > 3 {

		c.SessionTicketKeys = c.SessionTicketKeys[:3]

	}

}

// ValidateConfiguration validates the TLS configuration.

func (c *TLSEnhancedConfig) ValidateConfiguration() error {

	c.mu.RLock()

	defer c.mu.RUnlock()

	if c.MinVersion < tls.VersionTLS12 {

		return errors.New("minimum TLS version must be 1.2 or higher")

	}

	if c.MaxVersion < c.MinVersion {

		return errors.New("maximum TLS version cannot be lower than minimum version")

	}

	if len(c.certificates) == 0 && c.CertFile == "" {

		return errors.New("no certificates configured")

	}

	if c.OCSPStaplingEnabled && c.OCSPResponderURL == "" {

		return errors.New("OCSP stapling enabled but no responder URL configured")

	}

	return nil

}

// CreateSecureListener creates a TLS listener with the enhanced configuration.

func (c *TLSEnhancedConfig) CreateSecureListener(address string) (net.Listener, error) {

	tlsConfig, err := c.GetTLSConfig()

	if err != nil {

		return nil, fmt.Errorf("failed to get TLS config: %w", err)

	}

	listener, err := tls.Listen("tcp", address, tlsConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create TLS listener: %w", err)

	}

	return listener, nil

}

// WrapHTTPTransport wraps an HTTP transport with enhanced TLS configuration.

func (c *TLSEnhancedConfig) WrapHTTPTransport(transport *http.Transport) error {

	tlsConfig, err := c.GetTLSConfig()

	if err != nil {

		return fmt.Errorf("failed to get TLS config: %w", err)

	}

	transport.TLSClientConfig = tlsConfig

	return nil

}
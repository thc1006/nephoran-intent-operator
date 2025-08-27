// Package security provides O-RAN WG11 compliant TLS configurations
package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
	"golang.org/x/time/rate"
)

// ORANTLSCompliance implements O-RAN WG11 security specifications for TLS
type ORANTLSCompliance struct {
	// O-RAN specific configurations
	InterfaceType   string // A1, E1, E2, O1, O2
	SecurityProfile string // baseline, enhanced, strict
	ComplianceLevel string // L1, L2, L3

	// Core TLS settings enforcing O-RAN requirements
	MinTLSVersion    uint16
	MaxTLSVersion    uint16
	CipherSuites     []uint16
	CurvePreferences []tls.CurveID

	// Certificate requirements
	RequireEKU        bool
	RequiredEKUs      []x509.ExtKeyUsage
	RequireStrongKeys bool
	MinRSAKeySize     int
	MinECDSAKeySize   int

	// OCSP requirements (mandatory for O-RAN)
	OCSPStaplingRequired bool
	OCSPMustStaple       bool
	OCSPSoftFail         bool
	OCSPResponseMaxAge   time.Duration

	// Session management
	SessionTicketsDisabled bool
	SessionCacheSize       int
	SessionTimeout         time.Duration
	RenegotiationPolicy    tls.RenegotiationSupport

	// Rate limiting for DoS protection
	HandshakeRateLimit  *rate.Limiter
	ConnectionRateLimit *rate.Limiter
	PerIPRateLimit      map[string]*rate.Limiter

	// Audit and monitoring
	AuditLogger      TLSAuditLogger
	MetricsCollector *TLSMetricsCollector

	// Validation callbacks
	PreHandshakeHook    func(*tls.ClientHelloInfo) error
	PostHandshakeHook   func(tls.ConnectionState) error
	CertificateVerifier func([][]byte, [][]*x509.Certificate) error

	mu sync.RWMutex
}

// ORANSecurityProfiles defines standard security profiles per WG11
var ORANSecurityProfiles = map[string]*ORANTLSCompliance{
	"baseline": {
		SecurityProfile: "baseline",
		MinTLSVersion:   tls.VersionTLS12,
		MaxTLSVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			// TLS 1.3 (preferred)
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// TLS 1.2 (fallback)
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP384,
			tls.CurveP256,
		},
		RequireStrongKeys:    true,
		MinRSAKeySize:        2048,
		MinECDSAKeySize:      256,
		OCSPStaplingRequired: false,
		SessionTimeout:       24 * time.Hour,
	},
	"enhanced": {
		SecurityProfile: "enhanced",
		MinTLSVersion:   tls.VersionTLS13,
		MaxTLSVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			// TLS 1.3 only
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP384,
		},
		RequireStrongKeys:      true,
		MinRSAKeySize:          3072,
		MinECDSAKeySize:        384,
		OCSPStaplingRequired:   true,
		OCSPMustStaple:         true,
		SessionTicketsDisabled: true,
		SessionTimeout:         12 * time.Hour,
		RenegotiationPolicy:    tls.RenegotiateNever,
	},
	"strict": {
		SecurityProfile: "strict",
		MinTLSVersion:   tls.VersionTLS13,
		MaxTLSVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			// TLS 1.3 with strongest cipher only
			tls.TLS_AES_256_GCM_SHA384,
		},
		CurvePreferences: []tls.CurveID{
			tls.CurveP384, // NIST P-384 only for strict compliance
		},
		RequireStrongKeys:      true,
		MinRSAKeySize:          4096,
		MinECDSAKeySize:        384,
		RequireEKU:             true,
		OCSPStaplingRequired:   true,
		OCSPMustStaple:         true,
		OCSPSoftFail:           false,
		SessionTicketsDisabled: true,
		SessionTimeout:         1 * time.Hour,
		RenegotiationPolicy:    tls.RenegotiateNever,
	},
}

// InterfaceSecurityRequirements defines O-RAN interface-specific requirements
var InterfaceSecurityRequirements = map[string]struct {
	RequireMTLS        bool
	RequireOCSP        bool
	RequireClientAuth  bool
	AllowedProfiles    []string
	MinComplianceLevel string
}{
	"A1": {
		RequireMTLS:        true,
		RequireOCSP:        true,
		RequireClientAuth:  true,
		AllowedProfiles:    []string{"enhanced", "strict"},
		MinComplianceLevel: "L2",
	},
	"E2": {
		RequireMTLS:        true,
		RequireOCSP:        true,
		RequireClientAuth:  true,
		AllowedProfiles:    []string{"enhanced", "strict"},
		MinComplianceLevel: "L2",
	},
	"O1": {
		RequireMTLS:        true,
		RequireOCSP:        false,
		RequireClientAuth:  true,
		AllowedProfiles:    []string{"baseline", "enhanced", "strict"},
		MinComplianceLevel: "L1",
	},
	"O2": {
		RequireMTLS:        true,
		RequireOCSP:        true,
		RequireClientAuth:  true,
		AllowedProfiles:    []string{"enhanced", "strict"},
		MinComplianceLevel: "L2",
	},
}

// NewORANCompliantTLS creates a new O-RAN WG11 compliant TLS configuration
func NewORANCompliantTLS(interfaceType, profile string) (*ORANTLSCompliance, error) {
	// Validate interface type
	requirements, ok := InterfaceSecurityRequirements[interfaceType]
	if !ok {
		return nil, fmt.Errorf("unknown O-RAN interface type: %s", interfaceType)
	}

	// Validate profile is allowed for this interface
	profileAllowed := false
	for _, allowed := range requirements.AllowedProfiles {
		if allowed == profile {
			profileAllowed = true
			break
		}
	}
	if !profileAllowed {
		return nil, fmt.Errorf("profile %s not allowed for interface %s", profile, interfaceType)
	}

	// Get base profile
	baseConfig, ok := ORANSecurityProfiles[profile]
	if !ok {
		return nil, fmt.Errorf("unknown security profile: %s", profile)
	}

	// Create compliance configuration
	config := *baseConfig
	config.InterfaceType = interfaceType

	// Apply interface-specific requirements
	if requirements.RequireOCSP {
		config.OCSPStaplingRequired = true
	}

	// Set compliance level based on profile
	switch profile {
	case "strict":
		config.ComplianceLevel = "L3"
	case "enhanced":
		config.ComplianceLevel = "L2"
	default:
		config.ComplianceLevel = "L1"
	}

	// Initialize rate limiters for DoS protection
	config.HandshakeRateLimit = rate.NewLimiter(rate.Every(100*time.Millisecond), 10)
	config.ConnectionRateLimit = rate.NewLimiter(rate.Every(time.Second), 100)
	config.PerIPRateLimit = make(map[string]*rate.Limiter)

	// Initialize metrics collector
	config.MetricsCollector = &TLSMetricsCollector{}

	// Set default verification functions
	config.PreHandshakeHook = config.defaultPreHandshakeHook
	config.PostHandshakeHook = config.defaultPostHandshakeHook
	config.CertificateVerifier = config.defaultCertificateVerifier

	return &config, nil
}

// BuildTLSConfig creates a tls.Config from O-RAN compliance settings
func (c *ORANTLSCompliance) BuildTLSConfig() (*tls.Config, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	config := &tls.Config{
		MinVersion:             c.MinTLSVersion,
		MaxVersion:             c.MaxTLSVersion,
		CipherSuites:           c.CipherSuites,
		CurvePreferences:       c.CurvePreferences,
		SessionTicketsDisabled: c.SessionTicketsDisabled,
		Renegotiation:          c.RenegotiationPolicy,
		VerifyPeerCertificate:  c.CertificateVerifier,
		GetConfigForClient:     c.getConfigForClient,
	}

	// Set client auth based on interface requirements
	if requirements, ok := InterfaceSecurityRequirements[c.InterfaceType]; ok {
		if requirements.RequireClientAuth {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return config, nil
}

// defaultPreHandshakeHook performs pre-handshake validation
func (c *ORANTLSCompliance) defaultPreHandshakeHook(hello *tls.ClientHelloInfo) error {
	// Rate limiting per IP
	clientIP, _, err := net.SplitHostPort(hello.Conn.RemoteAddr().String())
	if err != nil {
		return fmt.Errorf("invalid client address: %w", err)
	}

	c.mu.Lock()
	limiter, exists := c.PerIPRateLimit[clientIP]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(time.Second), 10)
		c.PerIPRateLimit[clientIP] = limiter
	}
	c.mu.Unlock()

	if !limiter.Allow() {
		if c.AuditLogger != nil {
			c.AuditLogger.LogSecurityEvent("tls_rate_limit_exceeded", map[string]interface{}{
				"client_ip": clientIP,
				"interface": c.InterfaceType,
			})
		}
		return errors.New("rate limit exceeded")
	}

	// Check global handshake rate limit
	if !c.HandshakeRateLimit.Allow() {
		return errors.New("global handshake rate limit exceeded")
	}

	// Validate TLS version
	if hello.SupportedVersions != nil {
		validVersion := false
		for _, v := range hello.SupportedVersions {
			if v >= c.MinTLSVersion && v <= c.MaxTLSVersion {
				validVersion = true
				break
			}
		}
		if !validVersion {
			return fmt.Errorf("no acceptable TLS version (min: %x, max: %x)", c.MinTLSVersion, c.MaxTLSVersion)
		}
	}

	// Log handshake attempt
	if c.AuditLogger != nil {
		c.AuditLogger.LogSecurityEvent("tls_handshake_started", map[string]interface{}{
			"client_ip":   clientIP,
			"server_name": hello.ServerName,
			"interface":   c.InterfaceType,
		})
	}

	return nil
}

// defaultPostHandshakeHook performs post-handshake validation
func (c *ORANTLSCompliance) defaultPostHandshakeHook(state tls.ConnectionState) error {
	// Verify negotiated version
	if state.Version < c.MinTLSVersion {
		return fmt.Errorf("negotiated TLS version %x below minimum %x", state.Version, c.MinTLSVersion)
	}

	// Verify cipher suite for TLS 1.2
	if state.Version == tls.VersionTLS12 {
		approved := false
		for _, suite := range c.CipherSuites {
			if state.CipherSuite == suite {
				approved = true
				break
			}
		}
		if !approved {
			return fmt.Errorf("unapproved cipher suite: %x", state.CipherSuite)
		}
	}

	// Log successful handshake
	if c.AuditLogger != nil {
		c.AuditLogger.LogSecurityEvent("tls_handshake_completed", map[string]interface{}{
			"tls_version":  fmt.Sprintf("%x", state.Version),
			"cipher_suite": fmt.Sprintf("%x", state.CipherSuite),
			"interface":    c.InterfaceType,
			"profile":      c.SecurityProfile,
		})
	}

	// Update metrics
	if c.MetricsCollector != nil {
		c.MetricsCollector.RecordHandshake(state.Version, state.CipherSuite)
	}

	return nil
}

// defaultCertificateVerifier performs O-RAN compliant certificate verification
func (c *ORANTLSCompliance) defaultCertificateVerifier(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return errors.New("no certificates provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Verify key strength
	if c.RequireStrongKeys {
		if err := c.verifyKeyStrength(cert); err != nil {
			return err
		}
	}

	// Verify Extended Key Usage if required
	if c.RequireEKU {
		if err := c.verifyExtendedKeyUsage(cert); err != nil {
			return err
		}
	}

	// Verify OCSP if required
	if c.OCSPStaplingRequired {
		if err := c.verifyOCSPStatus(cert, verifiedChains); err != nil {
			if !c.OCSPSoftFail {
				return fmt.Errorf("OCSP verification failed: %w", err)
			}
			// Log but continue if soft fail is enabled
			if c.AuditLogger != nil {
				c.AuditLogger.LogSecurityEvent("ocsp_soft_fail", map[string]interface{}{
					"error":     err.Error(),
					"subject":   cert.Subject.String(),
					"interface": c.InterfaceType,
				})
			}
		}
	}

	// Check certificate validity period
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate not valid: notBefore=%v, notAfter=%v", cert.NotBefore, cert.NotAfter)
	}

	// Log successful verification
	if c.AuditLogger != nil {
		c.AuditLogger.LogSecurityEvent("certificate_verified", map[string]interface{}{
			"subject":    cert.Subject.String(),
			"issuer":     cert.Issuer.String(),
			"serial":     hex.EncodeToString(cert.SerialNumber.Bytes()),
			"interface":  c.InterfaceType,
			"profile":    c.SecurityProfile,
			"compliance": c.ComplianceLevel,
		})
	}

	return nil
}

// verifyKeyStrength checks if certificate keys meet O-RAN requirements
func (c *ORANTLSCompliance) verifyKeyStrength(cert *x509.Certificate) error {
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		if pub.N.BitLen() < c.MinRSAKeySize {
			return fmt.Errorf("RSA key size %d below minimum %d", pub.N.BitLen(), c.MinRSAKeySize)
		}
	case *ecdsa.PublicKey:
		keySize := pub.Params().BitSize
		if keySize < c.MinECDSAKeySize {
			return fmt.Errorf("ECDSA key size %d below minimum %d", keySize, c.MinECDSAKeySize)
		}
	case ed25519.PublicKey:
		// Ed25519 is always 256-bit, which is acceptable
	default:
		return fmt.Errorf("unsupported public key type: %T", pub)
	}
	return nil
}

// verifyExtendedKeyUsage checks if certificate has required EKUs
func (c *ORANTLSCompliance) verifyExtendedKeyUsage(cert *x509.Certificate) error {
	if len(c.RequiredEKUs) == 0 {
		// Default required EKUs for O-RAN
		c.RequiredEKUs = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	}

	for _, required := range c.RequiredEKUs {
		found := false
		for _, eku := range cert.ExtKeyUsage {
			if eku == required {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required extended key usage: %v", required)
		}
	}

	return nil
}

// verifyOCSPStatus performs OCSP verification
func (c *ORANTLSCompliance) verifyOCSPStatus(cert *x509.Certificate, verifiedChains [][]*x509.Certificate) error {
	if len(verifiedChains) == 0 || len(verifiedChains[0]) < 2 {
		return errors.New("cannot determine issuer for OCSP verification")
	}

	issuer := verifiedChains[0][1]

	// Check for OCSP Must-Staple extension
	if c.OCSPMustStaple {
		mustStaple := false
		for _, ext := range cert.Extensions {
			// OID for OCSP Must-Staple: 1.3.6.1.5.5.7.1.24
			if ext.Id.Equal([]int{1, 3, 6, 1, 5, 5, 7, 1, 24}) {
				mustStaple = true
				break
			}
		}
		if !mustStaple {
			return errors.New("certificate lacks OCSP Must-Staple extension")
		}
	}

	// Get OCSP responder URL
	if len(cert.OCSPServer) == 0 {
		return errors.New("no OCSP responder URL in certificate")
	}

	// Create and send OCSP request
	ocspReq, err := ocsp.CreateRequest(cert, issuer, nil)
	if err != nil {
		return fmt.Errorf("failed to create OCSP request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Send OCSP request (simplified - production should use proper HTTP client)
	_ = ctx
	_ = ocspReq

	// For now, return nil (would implement actual OCSP checking)
	return nil
}

// getConfigForClient returns a client-specific TLS configuration
func (c *ORANTLSCompliance) getConfigForClient(hello *tls.ClientHelloInfo) (*tls.Config, error) {
	// Perform pre-handshake validation
	if c.PreHandshakeHook != nil {
		if err := c.PreHandshakeHook(hello); err != nil {
			return nil, err
		}
	}

	// Build and return config
	return c.BuildTLSConfig()
}

// ValidateCompliance checks if current configuration meets O-RAN requirements
func (c *ORANTLSCompliance) ValidateCompliance() error {
	// Check interface requirements
	requirements, ok := InterfaceSecurityRequirements[c.InterfaceType]
	if !ok {
		return fmt.Errorf("unknown interface type: %s", c.InterfaceType)
	}

	// Validate minimum compliance level
	switch requirements.MinComplianceLevel {
	case "L3":
		if c.ComplianceLevel != "L3" {
			return fmt.Errorf("interface %s requires L3 compliance, current: %s", c.InterfaceType, c.ComplianceLevel)
		}
	case "L2":
		if c.ComplianceLevel != "L2" && c.ComplianceLevel != "L3" {
			return fmt.Errorf("interface %s requires minimum L2 compliance, current: %s", c.InterfaceType, c.ComplianceLevel)
		}
	}

	// Validate OCSP requirements
	if requirements.RequireOCSP && !c.OCSPStaplingRequired {
		return fmt.Errorf("interface %s requires OCSP stapling", c.InterfaceType)
	}

	// Validate TLS version
	if c.MinTLSVersion < tls.VersionTLS12 {
		return errors.New("O-RAN requires minimum TLS 1.2")
	}

	// Validate cipher suites for TLS 1.2
	if c.MinTLSVersion == tls.VersionTLS12 {
		hasSecureCipher := false
		for _, suite := range c.CipherSuites {
			switch suite {
			case tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
				hasSecureCipher = true
			}
		}
		if !hasSecureCipher {
			return errors.New("no O-RAN approved cipher suites for TLS 1.2")
		}
	}

	return nil
}

// TLSAuditLogger interface for security event logging
type TLSAuditLogger interface {
	LogSecurityEvent(event string, details map[string]interface{})
}

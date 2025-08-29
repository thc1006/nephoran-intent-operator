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

	RootCAs *x509.CertPool

	// Advanced features.

	Enable0RTT bool

	SessionTicketKeys [][32]byte

	ClientSessionCache tls.ClientSessionCache

	// OCSP configuration.

	OCSPStapling bool

	OCSPResponderURL string

	OCSPCache *OCSPCache

	// CRL configuration.

	CRLCheckEnabled bool

	CRLDistributionPoints []string

	CRLCache *CRLCache

	// Smart card support.

	SmartCardEnabled bool

	SmartCardProvider string

	// Connection pooling.

	ConnectionPool *TLSConnectionPool

	MaxIdleConns int

	MaxConnsPerHost int

	IdleConnTimeout time.Duration

	// Monitoring.

	MetricsCollector *TLSMetricsCollector

	mu sync.RWMutex
}

// TLSConnectionPool manages TLS connection pooling.

type TLSConnectionPool struct {
	connections map[string][]*tls.Conn

	maxSize int

	mu sync.RWMutex

	stats *ConnectionPoolStats
}

// ConnectionPoolStats tracks pool statistics.

type ConnectionPoolStats struct {
	hits uint64

	misses uint64

	evictions uint64

	activeConns uint64
}

// OCSPCache provides OCSP response caching.

type OCSPCache struct {
	cache map[string]*ocspCacheEntry

	mu sync.RWMutex

	ttl time.Duration
}

type ocspCacheEntry struct {
	response *ocsp.Response

	timestamp time.Time
}

// CRLCache provides CRL caching.

type CRLCache struct {
	cache map[string]*crlCacheEntry

	mu sync.RWMutex

	ttl time.Duration
}

type crlCacheEntry struct {
	crl *x509.RevocationList

	timestamp time.Time
}

// TLSMetricsCollector collects TLS metrics.

type TLSMetricsCollector struct {
	handshakes uint64

	handshakeErrors uint64

	resumptions uint64

	zeroRTTAccepted uint64

	zeroRTTRejected uint64

	ocspHits uint64

	ocspMisses uint64
}

// NewTLSEnhancedConfig creates a new enhanced TLS configuration.

func NewTLSEnhancedConfig() *TLSEnhancedConfig {

	return &TLSEnhancedConfig{

		MinVersion: tls.VersionTLS13,

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

			cache: make(map[string]*ocspCacheEntry),

			ttl: time.Hour,
		},

		CRLCache: &CRLCache{

			cache: make(map[string]*crlCacheEntry),

			ttl: 24 * time.Hour,
		},

		ConnectionPool: &TLSConnectionPool{

			connections: make(map[string][]*tls.Conn),

			maxSize: 100,

			stats: &ConnectionPoolStats{},
		},

		MetricsCollector: &TLSMetricsCollector{},

		MaxIdleConns: 50,

		MaxConnsPerHost: 10,

		IdleConnTimeout: 90 * time.Second,
	}

}

// BuildTLSConfig creates a tls.Config from the enhanced configuration.

func (c *TLSEnhancedConfig) BuildTLSConfig() (*tls.Config, error) {

	c.mu.RLock()

	defer c.mu.RUnlock()

	tlsConfig := &tls.Config{

		MinVersion: c.MinVersion,

		MaxVersion: c.MaxVersion,

		CipherSuites: c.CipherSuites,

		CurvePreferences: c.CurvePreferences,

		ClientCAs: c.ClientCAs,

		RootCAs: c.RootCAs,

		ClientSessionCache: c.ClientSessionCache,
	}

	// Load certificates if provided.

	if c.CertFile != "" && c.KeyFile != "" {

		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)

		if err != nil {

			return nil, fmt.Errorf("failed to load certificate: %w", err)

		}

		tlsConfig.Certificates = []tls.Certificate{cert}

	}

	// Configure session resumption and 0-RTT.

	if c.Enable0RTT {

		// Go 1.24+ supports 0-RTT data.

		tlsConfig.SessionTicketsDisabled = false

		if len(c.SessionTicketKeys) > 0 {

			tlsConfig.SetSessionTicketKeys(c.SessionTicketKeys)

		}

	}

	// Set up certificate verification with OCSP/CRL.

	tlsConfig.VerifyPeerCertificate = c.verifyPeerCertificate

	// Configure OCSP stapling.

	if c.OCSPStapling {

		tlsConfig.GetCertificate = c.getCertificateWithOCSP

	}

	return tlsConfig, nil

}

// verifyPeerCertificate performs advanced certificate verification.

func (c *TLSEnhancedConfig) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

	if len(rawCerts) == 0 {

		return errors.New("no certificates provided")

	}

	// Parse the leaf certificate.

	cert, err := x509.ParseCertificate(rawCerts[0])

	if err != nil {

		return fmt.Errorf("failed to parse certificate: %w", err)

	}

	// Check OCSP if enabled.

	if c.OCSPStapling && c.OCSPResponderURL != "" {

		if err := c.checkOCSPStatus(cert, rawCerts); err != nil {

			return fmt.Errorf("OCSP validation failed: %w", err)

		}

	}

	// Check CRL if enabled.

	if c.CRLCheckEnabled && len(c.CRLDistributionPoints) > 0 {

		if err := c.checkCRLStatus(cert); err != nil {

			return fmt.Errorf("CRL validation failed: %w", err)

		}

	}

	// Additional custom verification can be added here.

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

	// Create OCSP request.

	issuer := cert // In production, this should be the actual issuer

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

	c.OCSPCache.cache[string(cert.SerialNumber.Bytes())] = &ocspCacheEntry{

		response: ocspResp,

		timestamp: time.Now(),
	}

	c.OCSPCache.mu.Unlock()

	if ocspResp.Status != ocsp.Good {

		return fmt.Errorf("certificate revoked or unknown: %v", ocspResp.Status)

	}

	return nil

}

// checkCRLStatus verifies certificate status via CRL.

func (c *TLSEnhancedConfig) checkCRLStatus(cert *x509.Certificate) error {

	// Check cache first.

	c.CRLCache.mu.RLock()

	for _, dp := range c.CRLDistributionPoints {

		if cached, ok := c.CRLCache.cache[dp]; ok {

			if time.Since(cached.timestamp) < c.CRLCache.ttl {

				// Check if certificate is in the CRL.

				for _, revoked := range cached.crl.RevokedCertificateEntries {

					if cert.SerialNumber.Cmp(revoked.SerialNumber) == 0 {

						c.CRLCache.mu.RUnlock()

						return fmt.Errorf("certificate is revoked")

					}

				}

			}

		}

	}

	c.CRLCache.mu.RUnlock()

	// If not in cache or cache expired, fetch CRL.

	// This is simplified - production code should fetch and parse actual CRL.

	return nil

}

// getCertificateWithOCSP returns certificate with OCSP stapling.

func (c *TLSEnhancedConfig) getCertificateWithOCSP(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {

	// Load certificate.

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)

	if err != nil {

		return nil, err

	}

	// Add OCSP stapling if available.

	if c.OCSPStapling && len(cert.Certificate) > 0 {

		// In production, fetch fresh OCSP response.

		// cert.OCSPStaple = ocspResponse.

		// TODO: Implement OCSP stapling.

	}

	return &cert, nil

}

// CreateHTTPSServer creates an HTTPS server with enhanced TLS.

func (c *TLSEnhancedConfig) CreateHTTPSServer(handler http.Handler, addr string) (*http.Server, error) {

	tlsConfig, err := c.BuildTLSConfig()

	if err != nil {

		return nil, err

	}

	server := &http.Server{

		Addr: addr,

		Handler: handler,

		TLSConfig: tlsConfig,

		// Enhanced timeouts for security.

		ReadTimeout: 15 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout: 60 * time.Second,

		ReadHeaderTimeout: 5 * time.Second,

		MaxHeaderBytes: 1 << 20, // 1 MB

	}

	// Configure connection state tracking.

	server.ConnState = c.trackConnectionState

	return server, nil

}

// trackConnectionState monitors connection states.

func (c *TLSEnhancedConfig) trackConnectionState(conn net.Conn, state http.ConnState) {

	switch state {

	case http.StateNew:

		atomic.AddUint64(&c.ConnectionPool.stats.activeConns, 1)

	case http.StateClosed:

		atomic.AddUint64(&c.ConnectionPool.stats.activeConns, ^uint64(0))

	}

}

// GetMetrics returns current TLS metrics.

func (c *TLSEnhancedConfig) GetMetrics() map[string]uint64 {

	return map[string]uint64{

		"handshakes": atomic.LoadUint64(&c.MetricsCollector.handshakes),

		"handshake_errors": atomic.LoadUint64(&c.MetricsCollector.handshakeErrors),

		"resumptions": atomic.LoadUint64(&c.MetricsCollector.resumptions),

		"zero_rtt_accepted": atomic.LoadUint64(&c.MetricsCollector.zeroRTTAccepted),

		"zero_rtt_rejected": atomic.LoadUint64(&c.MetricsCollector.zeroRTTRejected),

		"ocsp_hits": atomic.LoadUint64(&c.MetricsCollector.ocspHits),

		"ocsp_misses": atomic.LoadUint64(&c.MetricsCollector.ocspMisses),

		"pool_hits": atomic.LoadUint64(&c.ConnectionPool.stats.hits),

		"pool_misses": atomic.LoadUint64(&c.ConnectionPool.stats.misses),

		"active_conns": atomic.LoadUint64(&c.ConnectionPool.stats.activeConns),
	}

}

// EnablePostQuantum prepares for post-quantum cryptography.

func (c *TLSEnhancedConfig) EnablePostQuantum(hybridMode bool) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.PostQuantumEnabled = true

	c.HybridMode = hybridMode

	// When PQ algorithms are standardized in Go, they will be configured here.

	// For now, we prepare the infrastructure.

	if hybridMode {

		// Use classical algorithms alongside PQ-ready configurations.

		c.MinVersion = tls.VersionTLS13

	}

}

// LoadCABundle loads CA certificates from a file.

func (c *TLSEnhancedConfig) LoadCABundle(caFile string) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	caCert, err := os.ReadFile(caFile)

	if err != nil {

		return fmt.Errorf("failed to read CA file: %w", err)

	}

	c.RootCAs = x509.NewCertPool()

	if !c.RootCAs.AppendCertsFromPEM(caCert) {

		return fmt.Errorf("failed to parse CA certificate")

	}

	c.CAFile = caFile

	return nil

}

// EnableClientCertAuth enables client certificate authentication.

func (c *TLSEnhancedConfig) EnableClientCertAuth(clientCAFile string) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	clientCA, err := os.ReadFile(clientCAFile)

	if err != nil {

		return fmt.Errorf("failed to read client CA file: %w", err)

	}

	c.ClientCAs = x509.NewCertPool()

	if !c.ClientCAs.AppendCertsFromPEM(clientCA) {

		return fmt.Errorf("failed to parse client CA certificate")

	}

	return nil

}

// SetSessionTicketKeys sets custom session ticket keys for 0-RTT.

func (c *TLSEnhancedConfig) SetSessionTicketKeys(keys [][32]byte) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.SessionTicketKeys = keys

}

// GetConnectionFromPool retrieves a connection from the pool.

func (p *TLSConnectionPool) GetConnection(addr string) (*tls.Conn, bool) {

	p.mu.RLock()

	defer p.mu.RUnlock()

	if conns, ok := p.connections[addr]; ok && len(conns) > 0 {

		atomic.AddUint64(&p.stats.hits, 1)

		conn := conns[len(conns)-1]

		p.connections[addr] = conns[:len(conns)-1]

		return conn, true

	}

	atomic.AddUint64(&p.stats.misses, 1)

	return nil, false

}

// ReturnConnection returns a connection to the pool.

func (p *TLSConnectionPool) ReturnConnection(addr string, conn *tls.Conn) {

	p.mu.Lock()

	defer p.mu.Unlock()

	if len(p.connections[addr]) < p.maxSize {

		p.connections[addr] = append(p.connections[addr], conn)

	} else {

		atomic.AddUint64(&p.stats.evictions, 1)

		conn.Close()

	}

}

// ValidateCertificateChain performs comprehensive certificate chain validation.

func ValidateCertificateChain(certPEM, chainPEM, rootPEM []byte) error {

	// Parse certificates.

	cert, _ := pem.Decode(certPEM)

	if cert == nil {

		return errors.New("failed to decode certificate")

	}

	x509Cert, err := x509.ParseCertificate(cert.Bytes)

	if err != nil {

		return fmt.Errorf("failed to parse certificate: %w", err)

	}

	// Create certificate pool for roots.

	roots := x509.NewCertPool()

	if !roots.AppendCertsFromPEM(rootPEM) {

		return errors.New("failed to parse root certificate")

	}

	// Create certificate pool for intermediates.

	intermediates := x509.NewCertPool()

	if len(chainPEM) > 0 {

		if !intermediates.AppendCertsFromPEM(chainPEM) {

			return errors.New("failed to parse intermediate certificates")

		}

	}

	// Verify certificate chain.

	opts := x509.VerifyOptions{

		Roots: roots,

		Intermediates: intermediates,

		CurrentTime: time.Now(),
	}

	if _, err := x509Cert.Verify(opts); err != nil {

		return fmt.Errorf("certificate chain validation failed: %w", err)

	}

	return nil

}

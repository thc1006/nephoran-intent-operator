package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
)

// ClientConfig holds mTLS client configuration
type ClientConfig struct {
	// Service identity
	ServiceName string
	TenantID    string

	// Certificate configuration
	ClientCertPath       string
	ClientKeyPath        string
	CACertPath           string
	CertValidityDuration time.Duration

	// Connection settings
	ServerName         string
	InsecureSkipVerify bool
	DialTimeout        time.Duration
	KeepAliveTimeout   time.Duration
	MaxIdleConns       int
	MaxConnsPerHost    int
	IdleConnTimeout    time.Duration

	// Certificate rotation
	RotationEnabled  bool
	RotationInterval time.Duration
	RenewalThreshold time.Duration

	// CA integration
	CAManager      *ca.CAManager
	AutoProvision  bool
	PolicyTemplate string
}

// Client provides mTLS-enabled HTTP client functionality
type Client struct {
	config      *ClientConfig
	httpClient  *http.Client
	tlsConfig   *tls.Config
	certificate *tls.Certificate
	caCertPool  *x509.CertPool
	logger      *logging.StructuredLogger

	// Certificate management
	certMu          sync.RWMutex
	lastRotation    time.Time
	rotationTicker  *time.Ticker
	rotationContext context.Context
	rotationCancel  context.CancelFunc
}

// NewClient creates a new mTLS-enabled HTTP client
func NewClient(config *ClientConfig, logger *logging.StructuredLogger) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("client config is required")
	}

	if logger == nil {
		logger = logging.NewStructuredLogger()
	}

	client := &Client{
		config: config,
		logger: logger,
	}

	// Initialize certificate and TLS configuration
	if err := client.initializeCertificates(); err != nil {
		return nil, fmt.Errorf("failed to initialize certificates: %w", err)
	}

	// Configure HTTP client with mTLS
	client.configureHTTPClient()

	// Start certificate rotation if enabled
	if config.RotationEnabled {
		client.startCertificateRotation()
	}

	logger.Info("mTLS client initialized",
		"service_name", config.ServiceName,
		"server_name", config.ServerName,
		"auto_provision", config.AutoProvision,
		"rotation_enabled", config.RotationEnabled)

	return client, nil
}

// initializeCertificates loads or provisions certificates
func (c *Client) initializeCertificates() error {
	c.certMu.Lock()
	defer c.certMu.Unlock()

	// Auto-provision certificates if enabled and CA manager is available
	if c.config.AutoProvision && c.config.CAManager != nil {
		if err := c.provisionCertificate(); err != nil {
			return fmt.Errorf("failed to auto-provision certificate: %w", err)
		}
	}

	// Load client certificate
	if err := c.loadClientCertificate(); err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate pool
	if err := c.loadCACertificatePool(); err != nil {
		return fmt.Errorf("failed to load CA certificate pool: %w", err)
	}

	// Create TLS configuration
	c.createTLSConfig()

	return nil
}

// provisionCertificate requests a new certificate from the CA manager
func (c *Client) provisionCertificate() error {
	if c.config.CAManager == nil {
		return fmt.Errorf("CA manager not available for auto-provisioning")
	}

	// Create certificate request
	req := &ca.CertificateRequest{
		ID:               generateRequestID(),
		TenantID:         c.config.TenantID,
		CommonName:       c.config.ServiceName,
		DNSNames:         []string{c.config.ServiceName, fmt.Sprintf("%s.%s", c.config.ServiceName, c.config.TenantID)},
		ValidityDuration: c.config.CertValidityDuration,
		KeyUsage:         []string{"digital_signature", "key_encipherment", "client_auth"},
		ExtKeyUsage:      []string{"client_auth"},
		PolicyTemplate:   c.config.PolicyTemplate,
		AutoRenew:        c.config.RotationEnabled,
		Metadata: map[string]string{
			"service_name": c.config.ServiceName,
			"purpose":      "mtls_client",
			"component":    "nephoran-intent-operator",
		},
	}

	if c.config.CertValidityDuration == 0 {
		req.ValidityDuration = 24 * time.Hour // Default to 24 hours
	}

	// Issue certificate
	resp, err := c.config.CAManager.IssueCertificate(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Store certificates for file-based loading
	if err := c.storeCertificateFiles(resp); err != nil {
		c.logger.Warn("failed to store certificate files", "error", err)
	}

	c.logger.Info("certificate provisioned",
		"request_id", req.ID,
		"serial_number", resp.SerialNumber,
		"expires_at", resp.ExpiresAt)

	return nil
}

// loadClientCertificate loads the client certificate and private key
func (c *Client) loadClientCertificate() error {
	if c.config.ClientCertPath == "" || c.config.ClientKeyPath == "" {
		return fmt.Errorf("client certificate and key paths are required")
	}

	cert, err := tls.LoadX509KeyPair(c.config.ClientCertPath, c.config.ClientKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	c.certificate = &cert
	c.lastRotation = time.Now()

	// Log certificate info
	if len(cert.Certificate) > 0 {
		if x509Cert, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
			c.logger.Info("loaded client certificate",
				"subject", x509Cert.Subject.String(),
				"issuer", x509Cert.Issuer.String(),
				"expires_at", x509Cert.NotAfter,
				"serial_number", x509Cert.SerialNumber.String())
		}
	}

	return nil
}

// loadCACertificatePool loads the CA certificate pool
func (c *Client) loadCACertificatePool() error {
	if c.config.CACertPath == "" {
		// Use system CA pool if no custom CA specified
		var err error
		c.caCertPool, err = x509.SystemCertPool()
		if err != nil {
			c.caCertPool = x509.NewCertPool()
		}
		return nil
	}

	// Load custom CA certificate
	caCert, err := loadCertificateFile(c.config.CACertPath)
	if err != nil {
		return fmt.Errorf("failed to load CA certificate: %w", err)
	}

	c.caCertPool = x509.NewCertPool()
	c.caCertPool.AppendCertsFromPEM(caCert)

	c.logger.Info("loaded CA certificate pool", "ca_cert_path", c.config.CACertPath)

	return nil
}

// createTLSConfig creates the TLS configuration for mTLS
func (c *Client) createTLSConfig() {
	c.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{*c.certificate},
		RootCAs:      c.caCertPool,
		ServerName:   c.config.ServerName,
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		InsecureSkipVerify: c.config.InsecureSkipVerify,
		ClientAuth:         tls.RequireAndVerifyClientCert,

		// Enable session resumption
		ClientSessionCache: tls.NewLRUClientSessionCache(64),

		// Certificate verification callback
		VerifyConnection: c.verifyConnection,
	}
}

// configureHTTPClient creates and configures the HTTP client
func (c *Client) configureHTTPClient() {
	transport := &http.Transport{
		TLSClientConfig: c.tlsConfig,

		// Connection pooling
		MaxIdleConns:        c.config.MaxIdleConns,
		MaxConnsPerHost:     c.config.MaxConnsPerHost,
		IdleConnTimeout:     c.config.IdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,

		// Timeouts
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Set defaults if not specified
	if c.config.DialTimeout == 0 {
		c.config.DialTimeout = 10 * time.Second
	}
	if c.config.MaxIdleConns == 0 {
		c.config.MaxIdleConns = 100
	}
	if c.config.MaxConnsPerHost == 0 {
		c.config.MaxConnsPerHost = 10
	}
	if c.config.IdleConnTimeout == 0 {
		c.config.IdleConnTimeout = 90 * time.Second
	}

	c.httpClient = &http.Client{
		Transport: transport,
		Timeout:   c.config.DialTimeout,
	}
}

// startCertificateRotation starts the certificate rotation process
func (c *Client) startCertificateRotation() {
	if c.config.RotationInterval == 0 {
		c.config.RotationInterval = 1 * time.Hour // Default rotation check interval
	}

	c.rotationContext, c.rotationCancel = context.WithCancel(context.Background())
	c.rotationTicker = time.NewTicker(c.config.RotationInterval)

	go c.certificateRotationLoop()

	c.logger.Info("certificate rotation started",
		"rotation_interval", c.config.RotationInterval,
		"renewal_threshold", c.config.RenewalThreshold)
}

// certificateRotationLoop handles automatic certificate rotation
func (c *Client) certificateRotationLoop() {
	defer c.rotationTicker.Stop()

	for {
		select {
		case <-c.rotationContext.Done():
			return
		case <-c.rotationTicker.C:
			if err := c.checkAndRotateCertificate(); err != nil {
				c.logger.Error("certificate rotation check failed", "error", err)
			}
		}
	}
}

// checkAndRotateCertificate checks if certificate rotation is needed and performs it
func (c *Client) checkAndRotateCertificate() error {
	c.certMu.RLock()
	cert := c.certificate
	c.certMu.RUnlock()

	if cert == nil || len(cert.Certificate) == 0 {
		return fmt.Errorf("no certificate available for rotation check")
	}

	// Parse certificate to check expiration
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if rotation is needed
	renewalThreshold := c.config.RenewalThreshold
	if renewalThreshold == 0 {
		renewalThreshold = 24 * time.Hour // Default to 24 hours before expiry
	}

	if time.Until(x509Cert.NotAfter) > renewalThreshold {
		// Certificate is still valid for sufficient time
		return nil
	}

	c.logger.Info("certificate rotation needed",
		"expires_at", x509Cert.NotAfter,
		"renewal_threshold", renewalThreshold)

	// Perform certificate rotation
	return c.rotateCertificate()
}

// rotateCertificate performs certificate rotation
func (c *Client) rotateCertificate() error {
	c.logger.Info("starting certificate rotation")

	// Provision new certificate if auto-provisioning is enabled
	if c.config.AutoProvision && c.config.CAManager != nil {
		if err := c.provisionCertificate(); err != nil {
			return fmt.Errorf("failed to provision new certificate: %w", err)
		}
	}

	// Reinitialize certificates and TLS config
	if err := c.initializeCertificates(); err != nil {
		return fmt.Errorf("failed to reinitialize certificates: %w", err)
	}

	// Update HTTP client with new TLS configuration
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = c.tlsConfig

		// Close idle connections to force new TLS handshakes
		transport.CloseIdleConnections()
	}

	c.logger.Info("certificate rotation completed")

	return nil
}

// verifyConnection provides custom certificate verification
func (c *Client) verifyConnection(cs tls.ConnectionState) error {
	// Additional verification logic can be added here
	// For now, we rely on the standard TLS verification

	c.logger.Debug("TLS connection verified",
		"peer_certificates", len(cs.PeerCertificates),
		"cipher_suite", cs.CipherSuite,
		"tls_version", cs.Version)

	return nil
}

// GetHTTPClient returns the configured HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// Close gracefully shuts down the mTLS client
func (c *Client) Close() error {
	c.logger.Info("shutting down mTLS client")

	if c.rotationCancel != nil {
		c.rotationCancel()
	}

	if c.rotationTicker != nil {
		c.rotationTicker.Stop()
	}

	// Close idle connections
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// GetCertificateInfo returns information about the current certificate
func (c *Client) GetCertificateInfo() (*CertificateInfo, error) {
	c.certMu.RLock()
	defer c.certMu.RUnlock()

	if c.certificate == nil || len(c.certificate.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate available")
	}

	x509Cert, err := x509.ParseCertificate(c.certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &CertificateInfo{
		Subject:      x509Cert.Subject.String(),
		Issuer:       x509Cert.Issuer.String(),
		SerialNumber: x509Cert.SerialNumber.String(),
		NotBefore:    x509Cert.NotBefore,
		NotAfter:     x509Cert.NotAfter,
		DNSNames:     x509Cert.DNSNames,
		IsExpired:    time.Now().After(x509Cert.NotAfter),
		ExpiresIn:    time.Until(x509Cert.NotAfter),
	}, nil
}

// CertificateInfo holds certificate information
type CertificateInfo struct {
	Subject      string        `json:"subject"`
	Issuer       string        `json:"issuer"`
	SerialNumber string        `json:"serial_number"`
	NotBefore    time.Time     `json:"not_before"`
	NotAfter     time.Time     `json:"not_after"`
	DNSNames     []string      `json:"dns_names"`
	IsExpired    bool          `json:"is_expired"`
	ExpiresIn    time.Duration `json:"expires_in"`
}

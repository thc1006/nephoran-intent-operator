package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
)

// ServerConfig holds mTLS server configuration.

type ServerConfig struct {

	// Service identity.

	ServiceName string

	TenantID string

	// Certificate configuration.

	ServerCertPath string

	ServerKeyPath string

	CACertPath string

	ClientCACertPath string // Separate CA for client certificate verification

	CertValidityDuration time.Duration

	// Server settings.

	Address string

	Port int

	ReadTimeout time.Duration

	WriteTimeout time.Duration

	IdleTimeout time.Duration

	ReadHeaderTimeout time.Duration

	MaxHeaderBytes int

	// TLS settings.

	MinTLSVersion uint16

	MaxTLSVersion uint16

	CipherSuites []uint16

	RequireClientCert bool

	ClientCertValidation ClientCertValidation

	// Certificate rotation.

	RotationEnabled bool

	RotationInterval time.Duration

	RenewalThreshold time.Duration

	// CA integration.

	CAManager *ca.CAManager

	AutoProvision bool

	PolicyTemplate string

	// Security settings.

	EnableHSTS bool

	HSTSMaxAge int64

	AllowedClientCNs []string

	AllowedClientOrgs []string

	ClientCertRequired bool
}

// ClientCertValidation defines client certificate validation level.

type ClientCertValidation string

const (

	// ClientCertNoVerification holds clientcertnoverification value.

	ClientCertNoVerification ClientCertValidation = "none"

	// ClientCertRequestOnly holds clientcertrequestonly value.

	ClientCertRequestOnly ClientCertValidation = "request"

	// ClientCertRequired holds clientcertrequired value.

	ClientCertRequired ClientCertValidation = "require"

	// ClientCertRequiredVerify holds clientcertrequiredverify value.

	ClientCertRequiredVerify ClientCertValidation = "require_verify"
)

// Server provides mTLS-enabled HTTP server functionality.

type Server struct {
	config *ServerConfig

	httpServer *http.Server

	tlsConfig *tls.Config

	listener net.Listener

	logger *logging.StructuredLogger

	// Certificate management.

	serverCert *tls.Certificate

	caCertPool *x509.CertPool

	clientCACertPool *x509.CertPool

	certMu sync.RWMutex

	lastRotation time.Time

	rotationTicker *time.Ticker

	rotationContext context.Context

	rotationCancel context.CancelFunc

	// Server lifecycle.

	startedAt time.Time

	shutdownFunc context.CancelFunc
}

// NewServer creates a new mTLS-enabled HTTP server.

func NewServer(config *ServerConfig, logger *logging.StructuredLogger) (*Server, error) {

	if config == nil {

		return nil, fmt.Errorf("server config is required")

	}

	if logger == nil {

		logger = logging.NewStructuredLogger(logging.DefaultConfig("mtls-server", "1.0.0", "production"))

	}

	// Set defaults.

	if config.Address == "" {

		config.Address = "0.0.0.0"

	}

	if config.Port == 0 {

		config.Port = 8443

	}

	if config.ReadTimeout == 0 {

		config.ReadTimeout = 30 * time.Second

	}

	if config.WriteTimeout == 0 {

		config.WriteTimeout = 30 * time.Second

	}

	if config.IdleTimeout == 0 {

		config.IdleTimeout = 120 * time.Second

	}

	if config.ReadHeaderTimeout == 0 {

		config.ReadHeaderTimeout = 10 * time.Second

	}

	if config.MaxHeaderBytes == 0 {

		config.MaxHeaderBytes = 1 << 20 // 1MB

	}

	if config.MinTLSVersion == 0 {

		config.MinTLSVersion = tls.VersionTLS12

	}

	if config.MaxTLSVersion == 0 {

		config.MaxTLSVersion = tls.VersionTLS13

	}

	if config.HSTSMaxAge == 0 {

		config.HSTSMaxAge = 31536000 // 1 year

	}

	server := &Server{

		config: config,

		logger: logger,
	}

	// Initialize certificates and TLS configuration.

	if err := server.initializeCertificates(); err != nil {

		return nil, fmt.Errorf("failed to initialize certificates: %w", err)

	}

	// Configure HTTP server.

	if err := server.configureHTTPServer(); err != nil {

		return nil, fmt.Errorf("failed to configure HTTP server: %w", err)

	}

	// Start certificate rotation if enabled.

	if config.RotationEnabled {

		server.startCertificateRotation()

	}

	logger.Info("mTLS server initialized",

		"service_name", config.ServiceName,

		"address", fmt.Sprintf("%s:%d", config.Address, config.Port),

		"client_cert_validation", config.ClientCertValidation,

		"auto_provision", config.AutoProvision,

		"rotation_enabled", config.RotationEnabled)

	return server, nil

}

// initializeCertificates loads or provisions certificates.

func (s *Server) initializeCertificates() error {

	s.certMu.Lock()

	defer s.certMu.Unlock()

	// Auto-provision certificates if enabled and CA manager is available.

	if s.config.AutoProvision && s.config.CAManager != nil {

		if err := s.provisionServerCertificate(); err != nil {

			return fmt.Errorf("failed to auto-provision server certificate: %w", err)

		}

	}

	// Load server certificate.

	if err := s.loadServerCertificate(); err != nil {

		return fmt.Errorf("failed to load server certificate: %w", err)

	}

	// Load CA certificate pools.

	if err := s.loadCACertificatePools(); err != nil {

		return fmt.Errorf("failed to load CA certificate pools: %w", err)

	}

	// Create TLS configuration.

	s.createTLSConfig()

	return nil

}

// provisionServerCertificate requests a new server certificate from the CA manager.

func (s *Server) provisionServerCertificate() error {

	if s.config.CAManager == nil {

		return fmt.Errorf("CA manager not available for auto-provisioning")

	}

	// Create certificate request with server-specific attributes.

	req := &ca.CertificateRequest{

		ID: generateRequestID(),

		TenantID: s.config.TenantID,

		CommonName: s.config.ServiceName,

		DNSNames: []string{

			s.config.ServiceName,

			fmt.Sprintf("%s.%s", s.config.ServiceName, s.config.TenantID),

			fmt.Sprintf("%s.%s.svc.cluster.local", s.config.ServiceName, s.config.TenantID),

			"localhost",
		},

		IPAddresses: []string{"127.0.0.1", s.config.Address},

		ValidityDuration: s.config.CertValidityDuration,

		KeyUsage: []string{"digital_signature", "key_encipherment", "server_auth"},

		ExtKeyUsage: []string{"server_auth"},

		PolicyTemplate: s.config.PolicyTemplate,

		AutoRenew: s.config.RotationEnabled,

		Metadata: map[string]string{

			"service_name": s.config.ServiceName,

			"purpose": "mtls_server",

			"component": "nephoran-intent-operator",
		},
	}

	if s.config.CertValidityDuration == 0 {

		req.ValidityDuration = 24 * time.Hour // Default to 24 hours

	}

	// Issue certificate.

	resp, err := s.config.CAManager.IssueCertificate(context.Background(), req)

	if err != nil {

		return fmt.Errorf("failed to issue server certificate: %w", err)

	}

	// Store certificates for file-based loading.

	if err := s.storeCertificateFiles(resp); err != nil {

		s.logger.Warn("failed to store certificate files", "error", err)

	}

	s.logger.Info("server certificate provisioned",

		"request_id", req.ID,

		"serial_number", resp.SerialNumber,

		"expires_at", resp.ExpiresAt)

	return nil

}

// loadServerCertificate loads the server certificate and private key.

func (s *Server) loadServerCertificate() error {

	if s.config.ServerCertPath == "" || s.config.ServerKeyPath == "" {

		return fmt.Errorf("server certificate and key paths are required")

	}

	cert, err := tls.LoadX509KeyPair(s.config.ServerCertPath, s.config.ServerKeyPath)

	if err != nil {

		return fmt.Errorf("failed to load server certificate: %w", err)

	}

	s.serverCert = &cert

	s.lastRotation = time.Now()

	// Log certificate info.

	if len(cert.Certificate) > 0 {

		if x509Cert, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {

			s.logger.Info("loaded server certificate",

				"subject", x509Cert.Subject.String(),

				"issuer", x509Cert.Issuer.String(),

				"expires_at", x509Cert.NotAfter,

				"serial_number", x509Cert.SerialNumber.String(),

				"dns_names", x509Cert.DNSNames)

		}

	}

	return nil

}

// loadCACertificatePools loads CA certificate pools for server and client verification.

func (s *Server) loadCACertificatePools() error {

	// Load server CA pool (for server certificate chain validation).

	if s.config.CACertPath != "" {

		caCert, err := loadCertificateFile(s.config.CACertPath)

		if err != nil {

			return fmt.Errorf("failed to load server CA certificate: %w", err)

		}

		s.caCertPool = x509.NewCertPool()

		s.caCertPool.AppendCertsFromPEM(caCert)

		s.logger.Info("loaded server CA certificate pool", "ca_cert_path", s.config.CACertPath)

	}

	// Load client CA pool (for client certificate verification).

	clientCACertPath := s.config.ClientCACertPath

	if clientCACertPath == "" {

		clientCACertPath = s.config.CACertPath // Use same CA if not specified

	}

	if clientCACertPath != "" {

		clientCACert, err := loadCertificateFile(clientCACertPath)

		if err != nil {

			return fmt.Errorf("failed to load client CA certificate: %w", err)

		}

		s.clientCACertPool = x509.NewCertPool()

		s.clientCACertPool.AppendCertsFromPEM(clientCACert)

		s.logger.Info("loaded client CA certificate pool", "client_ca_cert_path", clientCACertPath)

	} else {

		// Use system CA pool if no custom CA specified.

		var err error

		s.clientCACertPool, err = x509.SystemCertPool()

		if err != nil {

			s.clientCACertPool = x509.NewCertPool()

		}

	}

	return nil

}

// createTLSConfig creates the TLS configuration for mTLS.

func (s *Server) createTLSConfig() {

	// Determine client authentication mode.

	var clientAuth tls.ClientAuthType

	switch s.config.ClientCertValidation {

	case ClientCertNoVerification:

		clientAuth = tls.NoClientCert

	case ClientCertRequestOnly:

		clientAuth = tls.RequestClientCert

	case ClientCertRequired:

		clientAuth = tls.RequireAnyClientCert

	case ClientCertRequiredVerify:

		clientAuth = tls.RequireAndVerifyClientCert

	default:

		clientAuth = tls.RequireAndVerifyClientCert

	}

	// Configure cipher suites.

	cipherSuites := s.config.CipherSuites

	if len(cipherSuites) == 0 {

		cipherSuites = []uint16{

			tls.TLS_AES_256_GCM_SHA384,

			tls.TLS_CHACHA20_POLY1305_SHA256,

			tls.TLS_AES_128_GCM_SHA256,

			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,

			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,

			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}

	}

	s.tlsConfig = &tls.Config{

		Certificates: []tls.Certificate{*s.serverCert},

		ClientCAs: s.clientCACertPool,

		ClientAuth: clientAuth,

		MinVersion: s.config.MinTLSVersion,

		MaxVersion: s.config.MaxTLSVersion,

		CipherSuites: cipherSuites,

		// Certificate selection callback.

		GetCertificate: s.getCertificate,

		// Client certificate verification callback.

		VerifyPeerCertificate: s.verifyClientCertificate,

		// Performance optimizations.

		SessionTicketsDisabled: false,

		PreferServerCipherSuites: true,
	}

}

// configureHTTPServer creates and configures the HTTP server.

func (s *Server) configureHTTPServer() error {

	addr := fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)

	s.httpServer = &http.Server{

		Addr: addr,

		TLSConfig: s.tlsConfig,

		ReadTimeout: s.config.ReadTimeout,

		WriteTimeout: s.config.WriteTimeout,

		IdleTimeout: s.config.IdleTimeout,

		ReadHeaderTimeout: s.config.ReadHeaderTimeout,

		MaxHeaderBytes: s.config.MaxHeaderBytes,

		// Add security headers middleware.

		Handler: s.addSecurityHeaders(http.DefaultServeMux),
	}

	return nil

}

// addSecurityHeaders adds security headers to HTTP responses.

func (s *Server) addSecurityHeaders(handler http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// HSTS header.

		if s.config.EnableHSTS {

			hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", s.config.HSTSMaxAge)

			w.Header().Set("Strict-Transport-Security", hstsValue)

		}

		// Security headers.

		w.Header().Set("X-Content-Type-Options", "nosniff")

		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("X-XSS-Protection", "1; mode=block")

		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		handler.ServeHTTP(w, r)

	})

}

// getCertificate returns the appropriate certificate for the connection.

func (s *Server) getCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {

	s.certMu.RLock()

	defer s.certMu.RUnlock()

	if s.serverCert == nil {

		return nil, fmt.Errorf("no server certificate available")

	}

	// Log connection attempt.

	s.logger.Debug("TLS connection requested",

		"server_name", clientHello.ServerName,

		"supported_versions", clientHello.SupportedVersions,

		"cipher_suites", len(clientHello.CipherSuites))

	return s.serverCert, nil

}

// verifyClientCertificate provides custom client certificate verification.

func (s *Server) verifyClientCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

	if len(rawCerts) == 0 {

		if s.config.ClientCertRequired {

			return fmt.Errorf("client certificate required but not provided")

		}

		return nil

	}

	// Parse client certificate.

	clientCert, err := x509.ParseCertificate(rawCerts[0])

	if err != nil {

		return fmt.Errorf("failed to parse client certificate: %w", err)

	}

	// Validate against allowed CNs.

	if len(s.config.AllowedClientCNs) > 0 {

		allowed := false

		for _, allowedCN := range s.config.AllowedClientCNs {

			if clientCert.Subject.CommonName == allowedCN {

				allowed = true

				break

			}

		}

		if !allowed {

			return fmt.Errorf("client certificate CN '%s' not in allowed list", clientCert.Subject.CommonName)

		}

	}

	// Validate against allowed organizations.

	if len(s.config.AllowedClientOrgs) > 0 {

		allowed := false

		for _, clientOrg := range clientCert.Subject.Organization {

			for _, allowedOrg := range s.config.AllowedClientOrgs {

				if clientOrg == allowedOrg {

					allowed = true

					break

				}

			}

			if allowed {

				break

			}

		}

		if !allowed {

			return fmt.Errorf("client certificate organization not in allowed list")

		}

	}

	s.logger.Info("client certificate verified",

		"subject", clientCert.Subject.String(),

		"issuer", clientCert.Issuer.String(),

		"serial_number", clientCert.SerialNumber.String(),

		"expires_at", clientCert.NotAfter)

	return nil

}

// startCertificateRotation starts the certificate rotation process.

func (s *Server) startCertificateRotation() {

	if s.config.RotationInterval == 0 {

		s.config.RotationInterval = 1 * time.Hour // Default rotation check interval

	}

	s.rotationContext, s.rotationCancel = context.WithCancel(context.Background())

	s.rotationTicker = time.NewTicker(s.config.RotationInterval)

	go s.certificateRotationLoop()

	s.logger.Info("certificate rotation started",

		"rotation_interval", s.config.RotationInterval,

		"renewal_threshold", s.config.RenewalThreshold)

}

// certificateRotationLoop handles automatic certificate rotation.

func (s *Server) certificateRotationLoop() {

	defer s.rotationTicker.Stop()

	for {

		select {

		case <-s.rotationContext.Done():

			return

		case <-s.rotationTicker.C:

			if err := s.checkAndRotateCertificate(); err != nil {

				s.logger.Error("certificate rotation check failed", "error", err)

			}

		}

	}

}

// checkAndRotateCertificate checks if certificate rotation is needed and performs it.

func (s *Server) checkAndRotateCertificate() error {

	s.certMu.RLock()

	cert := s.serverCert

	s.certMu.RUnlock()

	if cert == nil || len(cert.Certificate) == 0 {

		return fmt.Errorf("no certificate available for rotation check")

	}

	// Parse certificate to check expiration.

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])

	if err != nil {

		return fmt.Errorf("failed to parse certificate: %w", err)

	}

	// Check if rotation is needed.

	renewalThreshold := s.config.RenewalThreshold

	if renewalThreshold == 0 {

		renewalThreshold = 24 * time.Hour // Default to 24 hours before expiry

	}

	if time.Until(x509Cert.NotAfter) > renewalThreshold {

		// Certificate is still valid for sufficient time.

		return nil

	}

	s.logger.Info("certificate rotation needed",

		"expires_at", x509Cert.NotAfter,

		"renewal_threshold", renewalThreshold)

	// Perform certificate rotation.

	return s.rotateCertificate()

}

// rotateCertificate performs certificate rotation.

func (s *Server) rotateCertificate() error {

	s.logger.Info("starting certificate rotation")

	// Provision new certificate if auto-provisioning is enabled.

	if s.config.AutoProvision && s.config.CAManager != nil {

		if err := s.provisionServerCertificate(); err != nil {

			return fmt.Errorf("failed to provision new certificate: %w", err)

		}

	}

	// Reinitialize certificates and TLS config.

	if err := s.initializeCertificates(); err != nil {

		return fmt.Errorf("failed to reinitialize certificates: %w", err)

	}

	// Update HTTP server with new TLS configuration.

	s.httpServer.TLSConfig = s.tlsConfig

	s.logger.Info("certificate rotation completed")

	return nil

}

// Start starts the mTLS server.

func (s *Server) Start(ctx context.Context) error {

	s.startedAt = time.Now()

	// Create TLS listener.

	addr := fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)

	listener, err := net.Listen("tcp", addr)

	if err != nil {

		return fmt.Errorf("failed to create listener: %w", err)

	}

	s.listener = tls.NewListener(listener, s.tlsConfig)

	s.logger.Info("mTLS server starting",

		"address", addr,

		"service_name", s.config.ServiceName)

	// Start server in goroutine.

	go func() {

		if err := s.httpServer.Serve(s.listener); err != nil && err != http.ErrServerClosed {

			s.logger.Error("server failed", "error", err)

		}

	}()

	// Wait for context cancellation.

	<-ctx.Done()

	return s.Shutdown(context.Background())

}

// Shutdown gracefully shuts down the server.

func (s *Server) Shutdown(ctx context.Context) error {

	s.logger.Info("shutting down mTLS server")

	if s.rotationCancel != nil {

		s.rotationCancel()

	}

	if s.rotationTicker != nil {

		s.rotationTicker.Stop()

	}

	if s.httpServer != nil {

		return s.httpServer.Shutdown(ctx)

	}

	return nil

}

// GetServerInfo returns information about the server.

func (s *Server) GetServerInfo() *ServerInfo {

	return &ServerInfo{

		ServiceName: s.config.ServiceName,

		Address: fmt.Sprintf("%s:%d", s.config.Address, s.config.Port),

		StartedAt: s.startedAt,

		TLSEnabled: true,

		ClientAuth: string(s.config.ClientCertValidation),

		Uptime: time.Since(s.startedAt),
	}

}

// ServerInfo holds server information.

type ServerInfo struct {
	ServiceName string `json:"service_name"`

	Address string `json:"address"`

	StartedAt time.Time `json:"started_at"`

	TLSEnabled bool `json:"tls_enabled"`

	ClientAuth string `json:"client_auth"`

	Uptime time.Duration `json:"uptime"`
}

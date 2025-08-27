// Package security provides enterprise-grade mutual TLS (mTLS) implementation
// for the Nephoran Intent Operator, ensuring secure inter-service communication
// with full O-RAN WG11 compliance and automated certificate management
package security

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"k8s.io/client-go/kubernetes"
)

// MTLSConfig defines comprehensive mutual TLS configuration
type MTLSConfig struct {
	// Service identity
	ServiceName      string   `json:"service_name" yaml:"service_name"`
	ServiceNamespace string   `json:"service_namespace" yaml:"service_namespace"`
	AllowedServices  []string `json:"allowed_services" yaml:"allowed_services"`

	// Certificate configuration
	CertFile     string        `json:"cert_file" yaml:"cert_file"`
	KeyFile      string        `json:"key_file" yaml:"key_file"`
	CAFile       string        `json:"ca_file" yaml:"ca_file"`
	CertValidity time.Duration `json:"cert_validity" yaml:"cert_validity"`

	// Rotation configuration
	AutoRotateEnabled     bool          `json:"auto_rotate_enabled" yaml:"auto_rotate_enabled"`
	RotationThreshold     time.Duration `json:"rotation_threshold" yaml:"rotation_threshold"`
	RotationCheckInterval time.Duration `json:"rotation_check_interval" yaml:"rotation_check_interval"`

	// Security policies
	RequireClientCert      bool     `json:"require_client_cert" yaml:"require_client_cert"`
	ValidateCommonName     bool     `json:"validate_common_name" yaml:"validate_common_name"`
	ValidateSubjectAltName bool     `json:"validate_subject_alt_name" yaml:"validate_subject_alt_name"`
	AllowedCommonNames     []string `json:"allowed_common_names" yaml:"allowed_common_names"`
	AllowedSANs            []string `json:"allowed_sans" yaml:"allowed_sans"`

	// TLS configuration
	MinTLSVersion    uint16        `json:"min_tls_version" yaml:"min_tls_version"`
	MaxTLSVersion    uint16        `json:"max_tls_version" yaml:"max_tls_version"`
	CipherSuites     []uint16      `json:"cipher_suites" yaml:"cipher_suites"`
	CurvePreferences []tls.CurveID `json:"curve_preferences" yaml:"curve_preferences"`

	// OCSP and CRL configuration
	EnableOCSP         bool   `json:"enable_ocsp" yaml:"enable_ocsp"`
	OCSPServerURL      string `json:"ocsp_server_url" yaml:"ocsp_server_url"`
	EnableCRL          bool   `json:"enable_crl" yaml:"enable_crl"`
	CRLDistributionURL string `json:"crl_distribution_url" yaml:"crl_distribution_url"`
}

// DefaultMTLSConfig returns O-RAN WG11 compliant mTLS configuration
func DefaultMTLSConfig(serviceName string) *MTLSConfig {
	return &MTLSConfig{
		ServiceName:      serviceName,
		ServiceNamespace: "nephoran-system",
		AllowedServices:  []string{}, // Will be populated based on service mesh policy

		// Certificate validity - shorter for better security
		CertValidity: 90 * 24 * time.Hour, // 90 days max

		// Auto-rotation enabled with 30-day threshold
		AutoRotateEnabled:     true,
		RotationThreshold:     30 * 24 * time.Hour,
		RotationCheckInterval: 1 * time.Hour,

		// Strict security policies
		RequireClientCert:      true,
		ValidateCommonName:     true,
		ValidateSubjectAltName: true,

		// TLS 1.3 only with strongest configurations
		MinTLSVersion: tls.VersionTLS13,
		MaxTLSVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.CurveP521, // Highest security curve
			tls.CurveP384,
			tls.X25519,
		},

		// OCSP enabled for real-time certificate status checking
		EnableOCSP: true,
		EnableCRL:  true,
	}
}

// MTLSManager manages mutual TLS for inter-service communication
type MTLSManager struct {
	config     *MTLSConfig
	logger     *zap.Logger
	certGen    *SecureCertificateGenerator
	tlsBuilder *SecureTLSConfigBuilder

	// Certificate management
	serverCert tls.Certificate
	clientCert tls.Certificate
	caCert     *x509.Certificate
	caPool     *x509.CertPool

	// Kubernetes integration
	k8sClient kubernetes.Interface

	// Monitoring and metrics
	certExpiryTimer *time.Timer
	rotationCount   int64

	// Thread safety
	mu sync.RWMutex

	// Shutdown management
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewMTLSManager creates a new enterprise mTLS manager
func NewMTLSManager(config *MTLSConfig, k8sClient kubernetes.Interface, logger *zap.Logger) (*MTLSManager, error) {
	if config == nil {
		config = DefaultMTLSConfig("unknown-service")
	}

	// Validate configuration
	if err := validateMTLSConfig(config); err != nil {
		return nil, fmt.Errorf("invalid mTLS configuration: %w", err)
	}

	cryptoConfig := DefaultCryptoConfig()
	certGen := NewSecureCertificateGenerator(cryptoConfig, logger)
	tlsBuilder := NewSecureTLSConfigBuilder(cryptoConfig, logger)

	manager := &MTLSManager{
		config:     config,
		logger:     logger.With(zap.String("component", "mtls-manager")),
		certGen:    certGen,
		tlsBuilder: tlsBuilder,
		k8sClient:  k8sClient,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}

	// Initialize certificates
	if err := manager.initializeCertificates(); err != nil {
		return nil, fmt.Errorf("failed to initialize certificates: %w", err)
	}

	// Start certificate rotation monitoring
	if config.AutoRotateEnabled {
		go manager.certificateRotationLoop()
	}

	return manager, nil
}

// initializeCertificates initializes the CA and service certificates
func (m *MTLSManager) initializeCertificates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize CA certificate and key
	caCert, caKey, err := m.initializeCA()
	if err != nil {
		return fmt.Errorf("failed to initialize CA: %w", err)
	}

	m.caCert = caCert
	m.caPool = x509.NewCertPool()
	m.caPool.AddCert(caCert)

	// Generate server certificate
	serverSubject := pkix.Name{
		CommonName:   m.config.ServiceName,
		Organization: []string{"Nephoran Intent Operator"},
		Country:      []string{"US"},
		Province:     []string{""},
		Locality:     []string{""},
	}

	serverDNSNames := []string{
		m.config.ServiceName,
		fmt.Sprintf("%s.%s", m.config.ServiceName, m.config.ServiceNamespace),
		fmt.Sprintf("%s.%s.svc", m.config.ServiceName, m.config.ServiceNamespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", m.config.ServiceName, m.config.ServiceNamespace),
	}

	serverKey, serverCertX509, err := m.certGen.GenerateServerKeyPair(
		serverSubject,
		serverDNSNames,
		nil, // No IP addresses for now
		caCert,
		caKey,
	)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}

	m.serverCert = tls.Certificate{
		Certificate: [][]byte{serverCertX509.Raw, caCert.Raw},
		PrivateKey:  serverKey,
		Leaf:        serverCertX509,
	}

	// Generate client certificate (same as server for simplicity, but could be different)
	clientSubject := pkix.Name{
		CommonName:   fmt.Sprintf("%s-client", m.config.ServiceName),
		Organization: []string{"Nephoran Intent Operator Clients"},
		Country:      []string{"US"},
	}

	clientKey, clientCertX509, err := m.certGen.GenerateServerKeyPair(
		clientSubject,
		[]string{fmt.Sprintf("%s-client", m.config.ServiceName)},
		nil,
		caCert,
		caKey,
	)
	if err != nil {
		return fmt.Errorf("failed to generate client certificate: %w", err)
	}

	m.clientCert = tls.Certificate{
		Certificate: [][]byte{clientCertX509.Raw, caCert.Raw},
		PrivateKey:  clientKey,
		Leaf:        clientCertX509,
	}

	m.logger.Info("mTLS certificates initialized successfully",
		zap.String("service", m.config.ServiceName),
		zap.Time("server_cert_expires", serverCertX509.NotAfter),
		zap.Time("client_cert_expires", clientCertX509.NotAfter))

	// Schedule next rotation check
	m.scheduleNextRotationCheck()

	return nil
}

// initializeCA initializes or loads the Certificate Authority
func (m *MTLSManager) initializeCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	// In production, this would load from secure storage (K8s secrets, Vault, etc.)
	// For now, generate a self-signed CA

	caSubject := pkix.Name{
		CommonName:   "Nephoran Intent Operator CA",
		Organization: []string{"Nephoran"},
		Country:      []string{"US"},
	}

	caKey, caCert, err := m.certGen.GenerateCAKeyPair(caSubject)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	return caCert, caKey, nil
}

// GetServerTLSConfig returns a TLS configuration for servers
func (m *MTLSManager) GetServerTLSConfig() (*tls.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tlsConfig, err := m.tlsBuilder.BuildServerTLSConfig(m.serverCert, m.caPool)
	if err != nil {
		return nil, fmt.Errorf("failed to build server TLS config: %w", err)
	}

	// Add custom verification logic
	tlsConfig.VerifyConnection = m.verifyServerConnection
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	return tlsConfig, nil
}

// GetClientTLSConfig returns a TLS configuration for clients
func (m *MTLSManager) GetClientTLSConfig(serverName string) (*tls.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tlsConfig, err := m.tlsBuilder.BuildClientTLSConfig(&m.clientCert, m.caPool, serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to build client TLS config: %w", err)
	}

	// Add custom verification logic
	tlsConfig.VerifyPeerCertificate = m.verifyClientPeerCertificate

	return tlsConfig, nil
}

// verifyServerConnection performs custom server-side connection verification
func (m *MTLSManager) verifyServerConnection(cs tls.ConnectionState) error {
	// Standard TLS version and cipher validation
	if cs.Version != tls.VersionTLS13 {
		m.logger.Warn("Non-TLS 1.3 connection attempt blocked",
			zap.Uint16("version", cs.Version))
		return fmt.Errorf("TLS 1.3 required, got version 0x%04x", cs.Version)
	}

	// Verify client certificate is present for mTLS
	if len(cs.PeerCertificates) == 0 {
		m.logger.Warn("Client certificate missing for mTLS connection")
		return fmt.Errorf("client certificate required for mTLS")
	}

	clientCert := cs.PeerCertificates[0]

	// Validate certificate is not expired
	now := time.Now()
	if now.Before(clientCert.NotBefore) || now.After(clientCert.NotAfter) {
		m.logger.Warn("Client certificate time validity failed",
			zap.Time("not_before", clientCert.NotBefore),
			zap.Time("not_after", clientCert.NotAfter),
			zap.Time("now", now))
		return fmt.Errorf("client certificate is not valid at current time")
	}

	// Validate Common Name if required
	if m.config.ValidateCommonName && len(m.config.AllowedCommonNames) > 0 {
		allowed := false
		for _, cn := range m.config.AllowedCommonNames {
			if clientCert.Subject.CommonName == cn {
				allowed = true
				break
			}
		}
		if !allowed {
			m.logger.Warn("Client certificate CN not in allowed list",
				zap.String("client_cn", clientCert.Subject.CommonName),
				zap.Strings("allowed_cns", m.config.AllowedCommonNames))
			return fmt.Errorf("client certificate CN '%s' not authorized", clientCert.Subject.CommonName)
		}
	}

	// Validate Subject Alternative Names if required
	if m.config.ValidateSubjectAltName && len(m.config.AllowedSANs) > 0 {
		allowed := false
		for _, allowedSAN := range m.config.AllowedSANs {
			for _, clientSAN := range clientCert.DNSNames {
				if clientSAN == allowedSAN {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}
		if !allowed {
			m.logger.Warn("Client certificate SAN not in allowed list",
				zap.Strings("client_sans", clientCert.DNSNames),
				zap.Strings("allowed_sans", m.config.AllowedSANs))
			return fmt.Errorf("client certificate SANs not authorized")
		}
	}

	m.logger.Debug("Client mTLS authentication successful",
		zap.String("client_cn", clientCert.Subject.CommonName),
		zap.Strings("client_sans", clientCert.DNSNames),
		zap.Time("cert_expires", clientCert.NotAfter))

	return nil
}

// verifyClientPeerCertificate performs custom client-side peer certificate verification
func (m *MTLSManager) verifyClientPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no server certificates provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Validate certificate time validity
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("server certificate is not valid at current time: notBefore=%v, notAfter=%v, now=%v",
			cert.NotBefore, cert.NotAfter, now)
	}

	// Verify against our CA
	opts := x509.VerifyOptions{
		Roots:       m.caPool,
		CurrentTime: now,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	_, err = cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("server certificate verification failed: %w", err)
	}

	return nil
}

// CreateSecureHTTPClient creates an HTTP client with mTLS configuration
func (m *MTLSManager) CreateSecureHTTPClient(serverName string, timeout time.Duration) (*http.Client, error) {
	tlsConfig, err := m.GetClientTLSConfig(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client TLS config: %w", err)
	}

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

// CreateGRPCServerCredentials creates gRPC server credentials with mTLS
func (m *MTLSManager) CreateGRPCServerCredentials() (credentials.TransportCredentials, error) {
	tlsConfig, err := m.GetServerTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get server TLS config: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// CreateGRPCClientCredentials creates gRPC client credentials with mTLS
func (m *MTLSManager) CreateGRPCClientCredentials(serverName string) (credentials.TransportCredentials, error) {
	tlsConfig, err := m.GetClientTLSConfig(serverName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client TLS config: %w", err)
	}

	return credentials.NewTLS(tlsConfig), nil
}

// CreateGRPCServerOptions creates gRPC server options with mTLS and additional security
func (m *MTLSManager) CreateGRPCServerOptions() ([]grpc.ServerOption, error) {
	creds, err := m.CreateGRPCServerCredentials()
	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ConnectionTimeout(30 * time.Second),
		grpc.MaxConcurrentStreams(100),
		grpc.UnaryInterceptor(m.grpcUnaryServerInterceptor),
		grpc.StreamInterceptor(m.grpcStreamServerInterceptor),
	}, nil
}

// grpcUnaryServerInterceptor adds security logging for unary calls
func (m *MTLSManager) grpcUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Extract peer information
	if p, ok := peer.FromContext(ctx); ok {
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			m.logger.Debug("gRPC unary call with mTLS",
				zap.String("method", info.FullMethod),
				zap.String("client_cn", tlsInfo.State.PeerCertificates[0].Subject.CommonName),
				zap.String("remote_addr", p.Addr.String()))
		}
	}

	return handler(ctx, req)
}

// grpcStreamServerInterceptor adds security logging for streaming calls
func (m *MTLSManager) grpcStreamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Extract peer information
	ctx := ss.Context()
	if p, ok := peer.FromContext(ctx); ok {
		if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
			m.logger.Debug("gRPC stream call with mTLS",
				zap.String("method", info.FullMethod),
				zap.String("client_cn", tlsInfo.State.PeerCertificates[0].Subject.CommonName),
				zap.String("remote_addr", p.Addr.String()))
		}
	}

	return handler(srv, ss)
}

// certificateRotationLoop monitors certificate expiry and triggers rotation
func (m *MTLSManager) certificateRotationLoop() {
	defer close(m.doneCh)

	ticker := time.NewTicker(m.config.RotationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.checkAndRotateCertificates(); err != nil {
				m.logger.Error("Certificate rotation check failed", zap.Error(err))
			}
		case <-m.stopCh:
			return
		}
	}
}

// checkAndRotateCertificates checks if certificates need rotation and rotates them
func (m *MTLSManager) checkAndRotateCertificates() error {
	m.mu.RLock()
	serverExpiry := m.serverCert.Leaf.NotAfter
	clientExpiry := m.clientCert.Leaf.NotAfter
	m.mu.RUnlock()

	now := time.Now()
	threshold := m.config.RotationThreshold

	needsRotation := now.Add(threshold).After(serverExpiry) || now.Add(threshold).After(clientExpiry)

	if needsRotation {
		m.logger.Info("Certificate rotation triggered",
			zap.Time("server_expires", serverExpiry),
			zap.Time("client_expires", clientExpiry),
			zap.Duration("threshold", threshold))

		if err := m.rotateCertificates(); err != nil {
			return fmt.Errorf("certificate rotation failed: %w", err)
		}

		m.rotationCount++
		m.logger.Info("Certificate rotation completed successfully",
			zap.Int64("rotation_count", m.rotationCount))
	}

	return nil
}

// rotateCertificates generates new certificates
func (m *MTLSManager) rotateCertificates() error {
	// This would typically integrate with cert-manager or external CA
	// For now, regenerate with the same CA
	return m.initializeCertificates()
}

// scheduleNextRotationCheck schedules the next certificate rotation check
func (m *MTLSManager) scheduleNextRotationCheck() {
	m.mu.RLock()
	serverExpiry := m.serverCert.Leaf.NotAfter
	clientExpiry := m.clientCert.Leaf.NotAfter
	m.mu.RUnlock()

	// Schedule check for the earlier expiry minus threshold
	earliestExpiry := serverExpiry
	if clientExpiry.Before(serverExpiry) {
		earliestExpiry = clientExpiry
	}

	checkTime := earliestExpiry.Add(-m.config.RotationThreshold)
	duration := time.Until(checkTime)

	if duration < 0 {
		duration = time.Minute // Check immediately if overdue
	}

	if m.certExpiryTimer != nil {
		m.certExpiryTimer.Stop()
	}

	m.certExpiryTimer = time.AfterFunc(duration, func() {
		if err := m.checkAndRotateCertificates(); err != nil {
			m.logger.Error("Scheduled certificate rotation failed", zap.Error(err))
		}
	})
}

// Close shuts down the mTLS manager
func (m *MTLSManager) Close() error {
	close(m.stopCh)

	if m.certExpiryTimer != nil {
		m.certExpiryTimer.Stop()
	}

	// Wait for rotation loop to finish
	select {
	case <-m.doneCh:
	case <-time.After(5 * time.Second):
		m.logger.Warn("mTLS manager shutdown timeout")
	}

	return nil
}

// GetCertificateInfo returns information about current certificates
func (m *MTLSManager) GetCertificateInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"service_name":             m.config.ServiceName,
		"server_cert_expires":      m.serverCert.Leaf.NotAfter,
		"client_cert_expires":      m.clientCert.Leaf.NotAfter,
		"ca_cert_expires":          m.caCert.NotAfter,
		"rotation_count":           m.rotationCount,
		"auto_rotate_enabled":      m.config.AutoRotateEnabled,
		"rotation_threshold":       m.config.RotationThreshold,
		"days_until_server_expiry": time.Until(m.serverCert.Leaf.NotAfter).Hours() / 24,
		"days_until_client_expiry": time.Until(m.clientCert.Leaf.NotAfter).Hours() / 24,
	}
}

// Helper functions

// validateMTLSConfig validates the mTLS configuration
func validateMTLSConfig(config *MTLSConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if config.MinTLSVersion < tls.VersionTLS13 {
		return fmt.Errorf("minimum TLS version must be 1.3 or higher for O-RAN compliance")
	}

	if config.CertValidity > 90*24*time.Hour {
		return fmt.Errorf("certificate validity period exceeds 90-day security recommendation")
	}

	if config.RotationThreshold > config.CertValidity/2 {
		return fmt.Errorf("rotation threshold should be less than half of certificate validity period")
	}

	return nil
}

// MTLSHealthChecker provides health checking for mTLS connections
type MTLSHealthChecker struct {
	manager *MTLSManager
	logger  *zap.Logger
}

// NewMTLSHealthChecker creates a new mTLS health checker
func NewMTLSHealthChecker(manager *MTLSManager, logger *zap.Logger) *MTLSHealthChecker {
	return &MTLSHealthChecker{
		manager: manager,
		logger:  logger,
	}
}

// CheckHealth performs a comprehensive health check of mTLS configuration
func (hc *MTLSHealthChecker) CheckHealth() error {
	info := hc.manager.GetCertificateInfo()

	// Check certificate expiry
	serverExpiry, ok := info["server_cert_expires"].(time.Time)
	if !ok {
		return fmt.Errorf("unable to get server certificate expiry")
	}

	if time.Until(serverExpiry) < 24*time.Hour {
		return fmt.Errorf("server certificate expires in less than 24 hours")
	}

	clientExpiry, ok := info["client_cert_expires"].(time.Time)
	if !ok {
		return fmt.Errorf("unable to get client certificate expiry")
	}

	if time.Until(clientExpiry) < 24*time.Hour {
		return fmt.Errorf("client certificate expires in less than 24 hours")
	}

	// Test TLS configuration creation
	if _, err := hc.manager.GetServerTLSConfig(); err != nil {
		return fmt.Errorf("server TLS config creation failed: %w", err)
	}

	if _, err := hc.manager.GetClientTLSConfig("test-service"); err != nil {
		return fmt.Errorf("client TLS config creation failed: %w", err)
	}

	return nil
}

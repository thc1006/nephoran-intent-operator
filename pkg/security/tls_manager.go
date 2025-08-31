// Package security provides comprehensive TLS management and mTLS enforcement.

// for the Nephoran Intent Operator, implementing O-RAN WG11 security requirements.

package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (

	// Metrics for TLS operations.

	tlsCertificateExpiryDays = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Name: "nephoran_tls_certificate_expiry_days",

			Help: "Days until TLS certificate expiry",
		},

		[]string{"service", "type"},
	)

	tlsHandshakeErrors = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Name: "nephoran_tls_handshake_errors_total",

			Help: "Total number of TLS handshake errors",
		},

		[]string{"service", "reason"},
	)

	tlsVersionConnections = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Name: "nephoran_tls_version_connections_total",

			Help: "Total number of connections by TLS version",
		},

		[]string{"version"},
	)

	mtlsAuthenticationSuccess = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Name: "nephoran_mtls_authentication_success_total",

			Help: "Total successful mTLS authentications",
		},

		[]string{"client", "service"},
	)

	mtlsAuthenticationFailure = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Name: "nephoran_mtls_authentication_failure_total",

			Help: "Total failed mTLS authentications",
		},

		[]string{"reason", "service"},
	)
)

// TLSManagerConfig represents TLS configuration compliant with O-RAN security requirements.

type TLSManagerConfig struct {

	// Certificate paths.

	CertFile string

	KeyFile string

	CAFile string

	ClientCertFile string

	ClientKeyFile string

	// TLS settings.

	MinVersion uint16

	MaxVersion uint16

	CipherSuites []uint16

	MTLSEnabled bool

	ClientAuthType tls.ClientAuthType

	// Certificate validation.

	ValidateHostname bool

	AllowedCNs []string

	AllowedSANs []string

	// Rotation settings.

	RotationCheckInterval time.Duration

	RenewalThreshold time.Duration

	// Service identification.

	ServiceName string
}

// TLSManager manages TLS certificates and configurations.

type TLSManager struct {
	config *TLSManagerConfig

	logger *zap.Logger

	tlsConfig *tls.Config

	clientCreds credentials.TransportCredentials

	serverCreds credentials.TransportCredentials

	certPool *x509.CertPool

	mu sync.RWMutex

	watcher *fsnotify.Watcher

	stopCh chan struct{}
}

// NewTLSManager creates a new TLS manager with O-RAN compliant settings.

func NewTLSManager(config *TLSManagerConfig, logger *zap.Logger) (*TLSManager, error) {

	if config == nil {

		return nil, fmt.Errorf("TLS config cannot be nil")

	}

	// Set O-RAN compliant defaults.

	if config.MinVersion == 0 {

		config.MinVersion = tls.VersionTLS13

	}

	if config.MaxVersion == 0 {

		config.MaxVersion = tls.VersionTLS13

	}

	// O-RAN WG11 approved cipher suites for TLS 1.3.

	if len(config.CipherSuites) == 0 {

		config.CipherSuites = []uint16{

			tls.TLS_AES_256_GCM_SHA384,

			tls.TLS_CHACHA20_POLY1305_SHA256,
		}

	}

	// Default certificate rotation settings.

	if config.RotationCheckInterval == 0 {

		config.RotationCheckInterval = 1 * time.Hour

	}

	if config.RenewalThreshold == 0 {

		config.RenewalThreshold = 30 * 24 * time.Hour // 30 days

	}

	tm := &TLSManager{

		config: config,

		logger: logger,

		stopCh: make(chan struct{}),
	}

	// Initialize TLS configuration.

	if err := tm.loadCertificates(); err != nil {

		return nil, fmt.Errorf("failed to load certificates: %w", err)

	}

	// Setup file watcher for certificate rotation.

	if err := tm.setupFileWatcher(); err != nil {

		logger.Warn("Failed to setup certificate file watcher", zap.Error(err))

	}

	// Start certificate rotation monitor.

	go tm.monitorCertificateRotation()

	return tm, nil

}

// loadCertificates loads and validates TLS certificates.

func (tm *TLSManager) loadCertificates() error {

	tm.mu.Lock()

	defer tm.mu.Unlock()

	// Load CA certificate pool.

	caCert, err := os.ReadFile(tm.config.CAFile)

	if err != nil {

		return fmt.Errorf("failed to read CA certificate: %w", err)

	}

	tm.certPool = x509.NewCertPool()

	if !tm.certPool.AppendCertsFromPEM(caCert) {

		return fmt.Errorf("failed to parse CA certificate")

	}

	// Load server certificate.

	serverCert, err := tls.LoadX509KeyPair(tm.config.CertFile, tm.config.KeyFile)

	if err != nil {

		return fmt.Errorf("failed to load server certificate: %w", err)

	}

	// Parse certificate for validation and metrics.

	x509Cert, err := x509.ParseCertificate(serverCert.Certificate[0])

	if err != nil {

		return fmt.Errorf("failed to parse server certificate: %w", err)

	}

	// Validate certificate is not expired.

	if time.Now().After(x509Cert.NotAfter) {

		return fmt.Errorf("server certificate has expired")

	}

	// Update expiry metrics.

	daysUntilExpiry := time.Until(x509Cert.NotAfter).Hours() / 24

	tlsCertificateExpiryDays.WithLabelValues(tm.config.ServiceName, "server").Set(daysUntilExpiry)

	// Configure TLS settings.

	tm.tlsConfig = &tls.Config{

		Certificates: []tls.Certificate{serverCert},

		ClientCAs: tm.certPool,

		RootCAs: tm.certPool,

		MinVersion: tm.config.MinVersion,

		MaxVersion: tm.config.MaxVersion,

		CipherSuites: tm.config.CipherSuites,

		ClientAuth: tm.config.ClientAuthType,

		VerifyConnection: tm.verifyConnection,

		GetConfigForClient: tm.getConfigForClient,
	}

	// Set client authentication based on mTLS configuration.

	if tm.config.MTLSEnabled {

		tm.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		// Load client certificate for outbound connections.

		if tm.config.ClientCertFile != "" && tm.config.ClientKeyFile != "" {

			clientCert, err := tls.LoadX509KeyPair(tm.config.ClientCertFile, tm.config.ClientKeyFile)

			if err != nil {

				return fmt.Errorf("failed to load client certificate: %w", err)

			}

			tm.tlsConfig.Certificates = append(tm.tlsConfig.Certificates, clientCert)

			// Parse client certificate for metrics.

			x509ClientCert, err := x509.ParseCertificate(clientCert.Certificate[0])

			if err == nil {

				clientDaysUntilExpiry := time.Until(x509ClientCert.NotAfter).Hours() / 24

				tlsCertificateExpiryDays.WithLabelValues(tm.config.ServiceName, "client").Set(clientDaysUntilExpiry)

			}

		}

	}

	// Create gRPC credentials.

	tm.serverCreds = credentials.NewTLS(tm.tlsConfig)

	clientTLSConfig := tm.tlsConfig.Clone()

	clientTLSConfig.ServerName = tm.config.ServiceName

	tm.clientCreds = credentials.NewTLS(clientTLSConfig)

	tm.logger.Info("TLS certificates loaded successfully",

		zap.String("service", tm.config.ServiceName),

		zap.Bool("mTLS", tm.config.MTLSEnabled),

		zap.Float64("daysUntilExpiry", daysUntilExpiry))

	return nil

}

// verifyConnection performs custom verification during TLS handshake.

func (tm *TLSManager) verifyConnection(cs tls.ConnectionState) error {

	// Record TLS version metrics.

	switch cs.Version {

	case tls.VersionTLS13:

		tlsVersionConnections.WithLabelValues("1.3").Inc()

	case tls.VersionTLS12:

		tlsVersionConnections.WithLabelValues("1.2").Inc()

		// Reject TLS 1.2 if minimum version is 1.3.

		if tm.config.MinVersion >= tls.VersionTLS13 {

			tlsHandshakeErrors.WithLabelValues(tm.config.ServiceName, "tls_version_too_low").Inc()

			return fmt.Errorf("TLS 1.2 connections not allowed, minimum TLS 1.3 required")

		}

	default:

		tlsVersionConnections.WithLabelValues("other").Inc()

		tlsHandshakeErrors.WithLabelValues(tm.config.ServiceName, "unsupported_tls_version").Inc()

		return fmt.Errorf("unsupported TLS version: %x", cs.Version)

	}

	// Verify client certificate for mTLS.

	if tm.config.MTLSEnabled && len(cs.PeerCertificates) > 0 {

		clientCert := cs.PeerCertificates[0]

		// Validate certificate chain.

		opts := x509.VerifyOptions{

			Roots: tm.certPool,

			Intermediates: x509.NewCertPool(),

			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		for _, cert := range cs.PeerCertificates[1:] {

			opts.Intermediates.AddCert(cert)

		}

		if _, err := clientCert.Verify(opts); err != nil {

			mtlsAuthenticationFailure.WithLabelValues("certificate_validation_failed", tm.config.ServiceName).Inc()

			return fmt.Errorf("client certificate validation failed: %w", err)

		}

		// Validate allowed CNs.

		if len(tm.config.AllowedCNs) > 0 {

			allowed := false

			for _, cn := range tm.config.AllowedCNs {

				if clientCert.Subject.CommonName == cn {

					allowed = true

					break

				}

			}

			if !allowed {

				mtlsAuthenticationFailure.WithLabelValues("cn_not_allowed", tm.config.ServiceName).Inc()

				return fmt.Errorf("client CN %s not in allowed list", clientCert.Subject.CommonName)

			}

		}

		// Validate SANs.

		if len(tm.config.AllowedSANs) > 0 {

			allowed := false

			for _, san := range tm.config.AllowedSANs {

				for _, dnsName := range clientCert.DNSNames {

					if dnsName == san {

						allowed = true

						break

					}

				}

			}

			if !allowed {

				mtlsAuthenticationFailure.WithLabelValues("san_not_allowed", tm.config.ServiceName).Inc()

				return fmt.Errorf("client SANs not in allowed list")

			}

		}

		mtlsAuthenticationSuccess.WithLabelValues(clientCert.Subject.CommonName, tm.config.ServiceName).Inc()

		tm.logger.Debug("mTLS client authenticated",

			zap.String("clientCN", clientCert.Subject.CommonName),

			zap.Strings("clientSANs", clientCert.DNSNames))

	}

	return nil

}

// getConfigForClient returns a TLS configuration for a specific client.

func (tm *TLSManager) getConfigForClient(hello *tls.ClientHelloInfo) (*tls.Config, error) {

	tm.mu.RLock()

	defer tm.mu.RUnlock()

	// Return current configuration.

	// This allows for dynamic configuration updates.

	return tm.tlsConfig, nil

}

// setupFileWatcher sets up file watching for certificate changes.

func (tm *TLSManager) setupFileWatcher() error {

	watcher, err := fsnotify.NewWatcher()

	if err != nil {

		return fmt.Errorf("failed to create file watcher: %w", err)

	}

	tm.watcher = watcher

	// Watch certificate files.

	certDir := filepath.Dir(tm.config.CertFile)

	if err := watcher.Add(certDir); err != nil {

		return fmt.Errorf("failed to watch certificate directory: %w", err)

	}

	go func() {

		for {

			select {

			case event, ok := <-watcher.Events:

				if !ok {

					return

				}

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {

					tm.logger.Info("Certificate file changed, reloading", zap.String("file", event.Name))

					if err := tm.loadCertificates(); err != nil {

						tm.logger.Error("Failed to reload certificates", zap.Error(err))

					}

				}

			case err, ok := <-watcher.Errors:

				if !ok {

					return

				}

				tm.logger.Error("File watcher error", zap.Error(err))

			case <-tm.stopCh:

				return

			}

		}

	}()

	return nil

}

// monitorCertificateRotation monitors certificate expiry and triggers rotation.

func (tm *TLSManager) monitorCertificateRotation() {

	ticker := time.NewTicker(tm.config.RotationCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			tm.checkCertificateExpiry()

		case <-tm.stopCh:

			return

		}

	}

}

// checkCertificateExpiry checks if certificates need rotation.

func (tm *TLSManager) checkCertificateExpiry() {

	tm.mu.RLock()

	defer tm.mu.RUnlock()

	if tm.tlsConfig == nil || len(tm.tlsConfig.Certificates) == 0 {

		return

	}

	for i, cert := range tm.tlsConfig.Certificates {

		if len(cert.Certificate) == 0 {

			continue

		}

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])

		if err != nil {

			tm.logger.Error("Failed to parse certificate for expiry check", zap.Error(err))

			continue

		}

		timeUntilExpiry := time.Until(x509Cert.NotAfter)

		daysUntilExpiry := timeUntilExpiry.Hours() / 24

		certType := "server"

		if i > 0 {

			certType = "client"

		}

		tlsCertificateExpiryDays.WithLabelValues(tm.config.ServiceName, certType).Set(daysUntilExpiry)

		if timeUntilExpiry < tm.config.RenewalThreshold {

			tm.logger.Warn("Certificate approaching expiry",

				zap.String("subject", x509Cert.Subject.String()),

				zap.Time("expiryTime", x509Cert.NotAfter),

				zap.Float64("daysRemaining", daysUntilExpiry))

			// Trigger certificate renewal notification.

			// This would typically integrate with cert-manager or similar.

			tm.notifyCertificateRenewalRequired(x509Cert)

		}

	}

}

// notifyCertificateRenewalRequired sends notification for certificate renewal.

func (tm *TLSManager) notifyCertificateRenewalRequired(cert *x509.Certificate) {

	// This would integrate with cert-manager annotations or other renewal mechanisms.

	tm.logger.Info("Certificate renewal required",

		zap.String("subject", cert.Subject.String()),

		zap.Time("expiryTime", cert.NotAfter))

}

// GetTLSConfig returns the current TLS configuration.

func (tm *TLSManager) GetTLSConfig() *tls.Config {

	tm.mu.RLock()

	defer tm.mu.RUnlock()

	return tm.tlsConfig.Clone()

}

// GetServerCredentials returns gRPC server credentials.

func (tm *TLSManager) GetServerCredentials() credentials.TransportCredentials {

	tm.mu.RLock()

	defer tm.mu.RUnlock()

	return tm.serverCreds

}

// GetClientCredentials returns gRPC client credentials.

func (tm *TLSManager) GetClientCredentials() credentials.TransportCredentials {

	tm.mu.RLock()

	defer tm.mu.RUnlock()

	return tm.clientCreds

}

// CreateHTTPClient creates an HTTP client with mTLS configuration.

func (tm *TLSManager) CreateHTTPClient() *http.Client {

	return &http.Client{

		Transport: &http.Transport{

			TLSClientConfig: tm.GetTLSConfig(),

			ForceAttemptHTTP2: true,
		},

		Timeout: 30 * time.Second,
	}

}

// CreateGRPCServerOptions creates gRPC server options with TLS.

func (tm *TLSManager) CreateGRPCServerOptions() []grpc.ServerOption {

	return []grpc.ServerOption{

		grpc.Creds(tm.GetServerCredentials()),

		grpc.ConnectionTimeout(30 * time.Second),

		grpc.MaxConcurrentStreams(100),
	}

}

// CreateGRPCDialOptions creates gRPC dial options with TLS.

func (tm *TLSManager) CreateGRPCDialOptions() []grpc.DialOption {

	return []grpc.DialOption{

		grpc.WithTransportCredentials(tm.GetClientCredentials()),

		grpc.WithDefaultCallOptions(

			grpc.MaxCallRecvMsgSize(10*1024*1024), // 10MB

			grpc.MaxCallSendMsgSize(10*1024*1024), // 10MB

		),
	}

}

// ValidatePeerCertificate validates a peer certificate against configured policies.

func (tm *TLSManager) ValidatePeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

	if len(rawCerts) == 0 {

		if tm.config.MTLSEnabled {

			return fmt.Errorf("no client certificate provided for mTLS")

		}

		return nil

	}

	cert, err := x509.ParseCertificate(rawCerts[0])

	if err != nil {

		return fmt.Errorf("failed to parse peer certificate: %w", err)

	}

	// Check certificate validity.

	now := time.Now()

	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {

		return fmt.Errorf("certificate is not valid: notBefore=%v, notAfter=%v, now=%v",

			cert.NotBefore, cert.NotAfter, now)

	}

	// Verify against CA.

	opts := x509.VerifyOptions{

		Roots: tm.certPool,

		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {

		return fmt.Errorf("certificate verification failed: %w", err)

	}

	return nil

}

// Close stops the TLS manager and cleans up resources.

func (tm *TLSManager) Close() error {

	close(tm.stopCh)

	if tm.watcher != nil {

		return tm.watcher.Close()

	}

	return nil

}

// LoadTLSConfigFromEnv loads TLS configuration from environment variables.

func LoadTLSConfigFromEnv() *TLSManagerConfig {

	return &TLSManagerConfig{

		CertFile: os.Getenv("TLS_CERT_FILE"),

		KeyFile: os.Getenv("TLS_KEY_FILE"),

		CAFile: os.Getenv("TLS_CA_FILE"),

		ClientCertFile: os.Getenv("MTLS_CLIENT_CERT_FILE"),

		ClientKeyFile: os.Getenv("MTLS_CLIENT_KEY_FILE"),

		MTLSEnabled: os.Getenv("MTLS_ENABLED") == "true",

		ServiceName: os.Getenv("SERVICE_NAME"),
	}

}

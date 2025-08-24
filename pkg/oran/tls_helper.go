package oran

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// BuildTLSConfig builds a *tls.Config from the provided TLSConfig
// following O-RAN security requirements and best practices
func BuildTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if config == nil {
		return nil, fmt.Errorf("TLS configuration is required")
	}

	tlsConfig := &tls.Config{
		// CRITICAL SECURITY FIX: Enforce TLS 1.3 only - O-RAN WG11 compliance
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		
		// TLS 1.3 uses built-in secure cipher suites, but explicitly enforce the strongest
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (these are actually ignored in TLS 1.3, 
			// but kept for documentation and potential TLS 1.2 fallback scenarios)
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		
		// Security hardening
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       config.SkipVerify,
		
		// Additional security measures
		NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2
		
		// Renegotiation security
		Renegotiation: tls.RenegotiateNever,
		
		// Session tickets disabled for forward secrecy
		SessionTicketsDisabled: true,
	}

	// Load client certificate and key for mutual TLS
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate for server verification
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Enable client certificate authentication for mutual TLS
	if len(tlsConfig.Certificates) > 0 {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// ValidateTLSConfig validates the TLS configuration parameters
func ValidateTLSConfig(config *TLSConfig) error {
	if config == nil {
		return fmt.Errorf("TLS configuration cannot be nil")
	}

	// Check if certificate and key files exist when specified
	if config.CertFile != "" {
		if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file does not exist: %s", config.CertFile)
		}
	}

	if config.KeyFile != "" {
		if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("key file does not exist: %s", config.KeyFile)
		}
	}

	if config.CAFile != "" {
		if _, err := os.Stat(config.CAFile); os.IsNotExist(err) {
			return fmt.Errorf("CA file does not exist: %s", config.CAFile)
		}
	}

	// Ensure both cert and key are provided together
	if (config.CertFile != "" && config.KeyFile == "") || (config.CertFile == "" && config.KeyFile != "") {
		return fmt.Errorf("both certificate file and key file must be provided for mutual TLS")
	}

	return nil
}

// GetRecommendedTLSConfig returns a TLS configuration with O-RAN recommended security settings
func GetRecommendedTLSConfig(certFile, keyFile, caFile string) *TLSConfig {
	return &TLSConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		SkipVerify: false, // Always verify certificates in production
	}
}

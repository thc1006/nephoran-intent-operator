package mtls

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
)

// loadCertificateFile loads a certificate file and returns its contents
func loadCertificateFile(certPath string) ([]byte, error) {
	if certPath == "" {
		return nil, fmt.Errorf("certificate path is empty")
	}

	// Check if file exists
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate file does not exist: %s", certPath)
	}

	// Read certificate file
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file %s: %w", certPath, err)
	}

	if len(certData) == 0 {
		return nil, fmt.Errorf("certificate file is empty: %s", certPath)
	}

	return certData, nil
}

// generateRequestID generates a unique request ID for certificate requests
func generateRequestID() string {
	// Generate a random request ID
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("mtls-req-%x", randomBytes)
}

// storeCertificateFiles stores certificate response to files for subsequent loading
func (c *Client) storeCertificateFiles(resp *ca.CertificateResponse) error {
	if resp == nil {
		return fmt.Errorf("certificate response is nil")
	}

	// Ensure certificate directory exists
	certDir := filepath.Dir(c.config.ClientCertPath)
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Store certificate
	if err := ioutil.WriteFile(c.config.ClientCertPath, []byte(resp.CertificatePEM), 0640); err != nil {
		return fmt.Errorf("failed to store client certificate: %w", err)
	}

	// Store private key
	if err := ioutil.WriteFile(c.config.ClientKeyPath, []byte(resp.PrivateKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to store client private key: %w", err)
	}

	// Store CA certificate if provided
	if resp.CACertificatePEM != "" && c.config.CACertPath != "" {
		if err := ioutil.WriteFile(c.config.CACertPath, []byte(resp.CACertificatePEM), 0644); err != nil {
			return fmt.Errorf("failed to store CA certificate: %w", err)
		}
	}

	return nil
}

// storeCertificateFiles stores certificate response to files for subsequent loading (server version)
func (s *Server) storeCertificateFiles(resp *ca.CertificateResponse) error {
	if resp == nil {
		return fmt.Errorf("certificate response is nil")
	}

	// Ensure certificate directory exists
	certDir := filepath.Dir(s.config.ServerCertPath)
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Store certificate
	if err := ioutil.WriteFile(s.config.ServerCertPath, []byte(resp.CertificatePEM), 0640); err != nil {
		return fmt.Errorf("failed to store server certificate: %w", err)
	}

	// Store private key
	if err := ioutil.WriteFile(s.config.ServerKeyPath, []byte(resp.PrivateKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to store server private key: %w", err)
	}

	// Store CA certificate if provided
	if resp.CACertificatePEM != "" && s.config.CACertPath != "" {
		if err := ioutil.WriteFile(s.config.CACertPath, []byte(resp.CACertificatePEM), 0644); err != nil {
			return fmt.Errorf("failed to store CA certificate: %w", err)
		}
	}

	return nil
}

// ValidateClientConfig validates client configuration
func ValidateClientConfig(config *ClientConfig) error {
	if config == nil {
		return fmt.Errorf("client config is required")
	}

	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if config.AutoProvision {
		if config.CAManager == nil {
			return fmt.Errorf("CA manager is required for auto-provisioning")
		}
		if config.TenantID == "" {
			return fmt.Errorf("tenant ID is required for auto-provisioning")
		}
	} else {
		if config.ClientCertPath == "" {
			return fmt.Errorf("client certificate path is required when not auto-provisioning")
		}
		if config.ClientKeyPath == "" {
			return fmt.Errorf("client key path is required when not auto-provisioning")
		}
	}

	if config.RotationEnabled {
		if config.RotationInterval <= 0 {
			return fmt.Errorf("rotation interval must be positive when rotation is enabled")
		}
		if config.RenewalThreshold <= 0 {
			return fmt.Errorf("renewal threshold must be positive when rotation is enabled")
		}
	}

	return nil
}

// ValidateServerConfig validates server configuration
func ValidateServerConfig(config *ServerConfig) error {
	if config == nil {
		return fmt.Errorf("server config is required")
	}

	if config.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if config.AutoProvision {
		if config.CAManager == nil {
			return fmt.Errorf("CA manager is required for auto-provisioning")
		}
		if config.TenantID == "" {
			return fmt.Errorf("tenant ID is required for auto-provisioning")
		}
	} else {
		if config.ServerCertPath == "" {
			return fmt.Errorf("server certificate path is required when not auto-provisioning")
		}
		if config.ServerKeyPath == "" {
			return fmt.Errorf("server key path is required when not auto-provisioning")
		}
	}

	if config.RotationEnabled {
		if config.RotationInterval <= 0 {
			return fmt.Errorf("rotation interval must be positive when rotation is enabled")
		}
		if config.RenewalThreshold <= 0 {
			return fmt.Errorf("renewal threshold must be positive when rotation is enabled")
		}
	}

	return nil
}

// DefaultClientConfig returns a client configuration with sensible defaults
func DefaultClientConfig(serviceName string) *ClientConfig {
	return &ClientConfig{
		ServiceName:          serviceName,
		TenantID:             "default",
		CertValidityDuration: 24 * time.Hour,
		DialTimeout:          10 * time.Second,
		KeepAliveTimeout:     30 * time.Second,
		MaxIdleConns:         100,
		MaxConnsPerHost:      10,
		IdleConnTimeout:      90 * time.Second,
		RotationEnabled:      true,
		RotationInterval:     1 * time.Hour,
		RenewalThreshold:     6 * time.Hour,
		AutoProvision:        false,
		PolicyTemplate:       "client-auth",
	}
}

// DefaultServerConfig returns a server configuration with sensible defaults
func DefaultServerConfig(serviceName string) *ServerConfig {
	return &ServerConfig{
		ServiceName:          serviceName,
		TenantID:             "default",
		Address:              "0.0.0.0",
		Port:                 8443,
		CertValidityDuration: 24 * time.Hour,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		IdleTimeout:          120 * time.Second,
		ReadHeaderTimeout:    10 * time.Second,
		MaxHeaderBytes:       1 << 20, // 1MB
		MinTLSVersion:        0x0303,  // TLS 1.2
		MaxTLSVersion:        0x0304,  // TLS 1.3
		ClientCertValidation: ClientCertRequiredVerify,
		RotationEnabled:      true,
		RotationInterval:     1 * time.Hour,
		RenewalThreshold:     6 * time.Hour,
		AutoProvision:        false,
		PolicyTemplate:       "server-auth",
		EnableHSTS:           true,
		HSTSMaxAge:           31536000, // 1 year
		ClientCertRequired:   true,
	}
}

// ensureDirectory ensures that a directory exists with the specified permissions
func ensureDirectory(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, perm)
	}
	return nil
}

// writeSecureFile writes data to a file with specified permissions, ensuring parent directory exists
func writeSecureFile(filepath string, data []byte, perm os.FileMode) error {
	// Ensure parent directory exists
	dir := filepath[:len(filepath)-len(filepath[len(filepath)-1:])]
	if err := ensureDirectory(dir, 0750); err != nil {
		return err
	}

	// Write file with specified permissions
	return ioutil.WriteFile(filepath, data, perm)
}

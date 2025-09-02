package o1

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"
)

// O1Config represents the configuration for O1 interface
type O1Config struct {
	Address           string        `json:"address"`
	Port              int           `json:"port"`
	DefaultPort       int           `json:"defaultPort"`
	Username          string        `json:"username"`
	Password          string        `json:"password"`
	RetryInterval     time.Duration `json:"retryInterval"`
	MaxRetries        int           `json:"maxRetries"`
	TLSConfig         *TLSConfig    `json:"tlsConfig"`
	ConnectionTimeout time.Duration `json:"connectionTimeout"`
	ConnectTimeout    time.Duration `json:"connectTimeout"`
	RequestTimeout    time.Duration `json:"requestTimeout"`
}

// TLSConfig represents TLS configuration options
type TLSConfig struct {
	Enabled    bool   `json:"enabled"`
	CAFile     string `json:"caFile"`
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
	SkipVerify bool   `json:"skipVerify"`
	MinVersion string `json:"minVersion"`
}

// NewO1SecurityConfig creates a new security configuration
func NewO1SecurityConfig(config *O1Config) (*tls.Config, error) {
	if config.TLSConfig == nil || !config.TLSConfig.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSConfig.SkipVerify,
	}

	// Set minimum TLS version
	switch config.TLSConfig.MinVersion {
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	// Load CA certificate if provided
	if config.TLSConfig.CAFile != "" {
		caCert, err := ioutil.ReadFile(config.TLSConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if provided
	if config.TLSConfig.CertFile != "" && config.TLSConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSConfig.CertFile, config.TLSConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// NewO1SecurityManager creates a new security manager for O1 interface
func NewO1SecurityManager(config *O1Config) (*O1SecurityManager, error) {
	tlsConfig, err := NewO1SecurityConfig(config)
	if err != nil {
		return nil, err
	}

	return &O1SecurityManager{
		Config:    config,
		TLSConfig: tlsConfig,
	}, nil
}

// O1SecurityManager manages security configurations
type O1SecurityManager struct {
	Config    *O1Config
	TLSConfig *tls.Config
}

// Validate checks the security configuration
func (sm *O1SecurityManager) Validate() error {
	if sm.Config == nil {
		return fmt.Errorf("O1 configuration is nil")
	}

	if sm.Config.TLSConfig != nil && sm.Config.TLSConfig.Enabled {
		if sm.Config.TLSConfig.CAFile == "" && !sm.Config.TLSConfig.SkipVerify {
			return fmt.Errorf("CA file is required when skip verify is false")
		}
	}

	return nil
}

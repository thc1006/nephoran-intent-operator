package config

import (
	"crypto/x509"
	"time"
)

// APIKeyConfig represents configuration for an API key
type APIKeyConfig struct {
	Key         string            `json:"key,omitempty"`
	SecretName  string            `json:"secret_name,omitempty"`
	SecretKey   string            `json:"secret_key,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
}

// TLSConfig represents TLS configuration for O-RAN compliance
type TLSConfig struct {
	// Basic TLS settings
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	ServerName string `yaml:"server_name" json:"server_name"`

	// Certificate settings
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`

	// TLS version and cipher settings
	MinTLSVersion  string   `yaml:"min_tls_version" json:"min_tls_version"`
	MaxTLSVersion  string   `yaml:"max_tls_version" json:"max_tls_version"`
	AllowedCiphers []string `yaml:"allowed_ciphers" json:"allowed_ciphers"`
	NextProtos     []string `yaml:"next_protos" json:"next_protos"`

	// Client certificate settings
	RequireClientCert bool   `yaml:"require_client_cert" json:"require_client_cert"`
	ClientCertFile    string `yaml:"client_cert_file" json:"client_cert_file"`
	ClientKeyFile     string `yaml:"client_key_file" json:"client_key_file"`

	// Security and compliance settings
	ComplianceMode         string `yaml:"compliance_mode" json:"compliance_mode"`
	InterfaceType          string `yaml:"interface_type" json:"interface_type"`
	SessionTicketsDisabled bool   `yaml:"session_tickets_disabled" json:"session_tickets_disabled"`
	RenegotiationSupport   string `yaml:"renegotiation_support" json:"renegotiation_support"`
	FIPSMode               bool   `yaml:"fips_mode" json:"fips_mode"`

	// Connection settings
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	HandshakeTimeout   time.Duration `yaml:"handshake_timeout" json:"handshake_timeout"`

	// Certificate validation
	RootCAs      *x509.CertPool `yaml:"-" json:"-"`
	ClientCAs    *x509.CertPool `yaml:"-" json:"-"`
	Certificates []string       `yaml:"certificates" json:"certificates"`
}

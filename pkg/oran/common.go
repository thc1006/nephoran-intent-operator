package oran

// TLSConfig holds TLS configuration for O-RAN interfaces
type TLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	SkipVerify bool
}

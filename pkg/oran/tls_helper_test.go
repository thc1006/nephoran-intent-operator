package oran

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *TLSConfig
		expectError bool
		description string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			description: "should return error for nil config",
		},
		{
			name: "skip verify enabled",
			config: &TLSConfig{
				SkipVerify: true,
			},
			expectError: false,
			description: "should succeed with skip verify enabled",
		},
		{
			name: "empty config",
			config: &TLSConfig{
				SkipVerify: false,
			},
			expectError: false,
			description: "should succeed with empty config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := BuildTLSConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tlsConfig == nil {
				t.Error("expected TLS config but got nil")
				return
			}

			// Verify minimum TLS requirements
			if tlsConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("expected MinVersion TLS 1.2, got %v", tlsConfig.MinVersion)
			}

			if !tlsConfig.PreferServerCipherSuites {
				t.Error("expected PreferServerCipherSuites to be true")
			}

			if len(tlsConfig.CipherSuites) == 0 {
				t.Error("expected cipher suites to be configured")
			}

			if tt.config != nil && tlsConfig.InsecureSkipVerify != tt.config.SkipVerify {
				t.Errorf("expected InsecureSkipVerify %v, got %v",
					tt.config.SkipVerify, tlsConfig.InsecureSkipVerify)
			}
		})
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *TLSConfig
		expectError bool
		description string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			description: "should return error for nil config",
		},
		{
			name: "valid empty config",
			config: &TLSConfig{
				SkipVerify: true,
			},
			expectError: false,
			description: "should succeed with valid empty config",
		},
		{
			name: "cert without key",
			config: &TLSConfig{
				CertFile: "cert.pem",
			},
			expectError: true,
			description: "should return error when cert is provided without key",
		},
		{
			name: "key without cert",
			config: &TLSConfig{
				KeyFile: "key.pem",
			},
			expectError: true,
			description: "should return error when key is provided without cert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateTLSConfigWithFiles(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "tls_test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test certificate file
	certFile := filepath.Join(tempDir, "cert.pem")
	certContent := `-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQDKOGJQnQCAuTANBgkqhkiG9w0BAQsFADCBjDELMAkGA1UEBhMC
VVMxCzAJBgNVBAgMAlRYMQ8wDQYDVQQHDAZEYWxsYXMxDTALBgNVBAoMBFRlc3Qx
DTALBgNVBAsMBFRlc3QxEjAQBgNVBAMMCWxvY2FsaG9zdDEtMCsGCSqGSIb3DQEJ
ARYOY29udGFjdEB0ZXN0LmNvbTANBgkqhkiG9w0BAQsFAAOCAQEATest
-----END CERTIFICATE-----`
	if err := os.WriteFile(certFile, []byte(certContent), 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}

	// Create test key file
	keyFile := filepath.Join(tempDir, "key.pem")
	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDTest
-----END PRIVATE KEY-----`
	if err := os.WriteFile(keyFile, []byte(keyContent), 0644); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	// Create test CA file
	caFile := filepath.Join(tempDir, "ca.pem")
	if err := os.WriteFile(caFile, []byte(certContent), 0644); err != nil {
		t.Fatalf("failed to write CA file: %v", err)
	}

	tests := []struct {
		name        string
		config      *TLSConfig
		expectError bool
		description string
	}{
		{
			name: "valid cert and key files",
			config: &TLSConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
			},
			expectError: false,
			description: "should succeed with existing cert and key files",
		},
		{
			name: "valid cert, key, and CA files",
			config: &TLSConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   caFile,
			},
			expectError: false,
			description: "should succeed with all files present",
		},
		{
			name: "non-existent cert file",
			config: &TLSConfig{
				CertFile: filepath.Join(tempDir, "nonexistent.pem"),
				KeyFile:  keyFile,
			},
			expectError: true,
			description: "should return error for non-existent cert file",
		},
		{
			name: "non-existent key file",
			config: &TLSConfig{
				CertFile: certFile,
				KeyFile:  filepath.Join(tempDir, "nonexistent.pem"),
			},
			expectError: true,
			description: "should return error for non-existent key file",
		},
		{
			name: "non-existent CA file",
			config: &TLSConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
				CAFile:   filepath.Join(tempDir, "nonexistent.pem"),
			},
			expectError: true,
			description: "should return error for non-existent CA file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetRecommendedTLSConfig(t *testing.T) {
	certFile := "/path/to/cert.pem"
	keyFile := "/path/to/key.pem"
	caFile := "/path/to/ca.pem"

	config := GetRecommendedTLSConfig(certFile, keyFile, caFile)

	if config == nil {
		t.Fatal("expected TLS config but got nil")
	}

	if config.CertFile != certFile {
		t.Errorf("expected CertFile %s, got %s", certFile, config.CertFile)
	}

	if config.KeyFile != keyFile {
		t.Errorf("expected KeyFile %s, got %s", keyFile, config.KeyFile)
	}

	if config.CAFile != caFile {
		t.Errorf("expected CAFile %s, got %s", caFile, config.CAFile)
	}

	if config.SkipVerify {
		t.Error("expected SkipVerify to be false for recommended config")
	}
}

func TestTLSConfigSecurityFeatures(t *testing.T) {
	config := &TLSConfig{
		SkipVerify: false,
	}

	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test minimum TLS version
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		t.Errorf("TLS version too low, expected at least TLS 1.2, got %v", tlsConfig.MinVersion)
	}

	// Test cipher suites include strong encryption
	strongCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}

	foundStrong := false
	for _, configured := range tlsConfig.CipherSuites {
		for _, strong := range strongCiphers {
			if configured == strong {
				foundStrong = true
				break
			}
		}
		if foundStrong {
			break
		}
	}

	if !foundStrong {
		t.Error("no strong cipher suites found in configuration")
	}

	// Test server cipher suite preference
	if !tlsConfig.PreferServerCipherSuites {
		t.Error("expected server cipher suite preference to be enabled")
	}
}

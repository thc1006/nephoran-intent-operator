package o1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// generateTestCACert generates a self-signed CA certificate PEM for testing.
func generateTestCACert(t *testing.T) string {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test-ca",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create test certificate: %v", err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

func TestO1Config(t *testing.T) {
	// Generate a valid self-signed CA cert for the TLS test cases that need it.
	caPEM := generateTestCACert(t)

	tmpCA, err := os.CreateTemp("", "test-ca-*.crt")
	if err != nil {
		t.Fatalf("failed to create temp CA file: %v", err)
	}
	defer os.Remove(tmpCA.Name())
	if _, err := tmpCA.WriteString(caPEM); err != nil {
		t.Fatalf("failed to write temp CA file: %v", err)
	}
	tmpCA.Close()

	testCases := []struct {
		name          string
		config        *O1Config
		expectedError bool
		errorMsg      string
	}{
		{
			name: "Valid TLS Config with CA File",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:    true,
					CAFile:     tmpCA.Name(),
					SkipVerify: false,
				},
			},
			expectedError: false,
		},
		{
			name: "Valid TLS Config with Skip Verify",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:    true,
					SkipVerify: true,
					MinVersion: "1.3",
				},
			},
			expectedError: false,
		},
		{
			name: "Missing CA File without Skip Verify",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:    true,
					SkipVerify: false,
				},
			},
			expectedError: true,
			errorMsg:      "CA file is required when skip verify is false",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			securityManager, err := NewO1SecurityManager(tc.config)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, securityManager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, securityManager)

				// Validate specific TLS config
				if tc.config.TLSConfig != nil && tc.config.TLSConfig.Enabled {
					assert.NotNil(t, securityManager.TLSConfig)

					if tc.config.TLSConfig.MinVersion == "1.3" {
						assert.Equal(t, securityManager.TLSConfig.MinVersion, uint16(0x0304))
					}
				}
			}
		})
	}
}

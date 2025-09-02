/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

)

// TLSConfig represents TLS configuration for Docker registry authentication
type TLSConfig struct {
	CertFile           string `json:"certFile,omitempty" yaml:"certFile,omitempty"`
	KeyFile            string `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
	CAFile             string `json:"caFile,omitempty" yaml:"caFile,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty"`
	ServerName         string `json:"serverName,omitempty" yaml:"serverName,omitempty"`
}

// TLSLoginConfig combines Docker login configuration with TLS settings
type TLSLoginConfig struct {
	LoginConfig
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// TLSClient extends the basic Docker auth client with TLS capabilities
type TLSClient struct {
	*Client
	tlsConfig *tls.Config
}

// NewTLSClient creates a new Docker authentication client with TLS support
func NewTLSClient(store CredentialStore, tlsConfig *TLSConfig) (*TLSClient, error) {
	// Create base client
	baseClient := NewClient(store)

	// Configure TLS if provided
	var tlsConf *tls.Config
	if tlsConfig != nil {
		var err error
		tlsConf, err = buildTLSConfig(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
	}

	// Create HTTP client with TLS configuration
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
		},
	}

	// Update the auth client with TLS-enabled HTTP client
	baseClient.authClient.Client = httpClient

	return &TLSClient{
		Client:    baseClient,
		tlsConfig: tlsConf,
	}, nil
}

// LoginWithTLS authenticates with a Docker registry using TLS configuration
func (c *TLSClient) LoginWithTLS(ctx context.Context, config *TLSLoginConfig) error {
	if config == nil {
		return fmt.Errorf("TLS login config cannot be nil")
	}

	// If TLS config is provided, update the client
	if config.TLS != nil {
		tlsConf, err := buildTLSConfig(config.TLS)
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}

		// Update HTTP client with new TLS config
		c.authClient.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			},
		}
		c.tlsConfig = tlsConf
	}

	// Perform the login
	return c.Login(ctx, &config.LoginConfig)
}

// buildTLSConfig constructs a tls.Config from TLSConfig
func buildTLSConfig(config *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
		ServerName:         config.ServerName,
	}

	// Load client certificate if provided
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if config.CAFile != "" {
		caCert, err := ioutil.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// GetTLSConfig returns the current TLS configuration
func (c *TLSClient) GetTLSConfig() *tls.Config {
	return c.tlsConfig
}

// UpdateTLSConfig updates the TLS configuration for the client
func (c *TLSClient) UpdateTLSConfig(config *TLSConfig) error {
	if config == nil {
		return fmt.Errorf("TLS config cannot be nil")
	}

	tlsConf, err := buildTLSConfig(config)
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Update HTTP client with new TLS config
	c.authClient.Client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
		},
	}
	c.tlsConfig = tlsConf

	return nil
}

// ValidateTLSConnection validates the TLS connection to a registry
func (c *TLSClient) ValidateTLSConnection(ctx context.Context, registry string) error {
	// Create a simple GET request to test the connection
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/v2/", registry), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.authClient.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// We expect either 200 OK or 401 Unauthorized (which means the registry is responding)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
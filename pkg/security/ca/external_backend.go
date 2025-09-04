package ca

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// ExternalBackend implements integration with external PKI systems.

type ExternalBackend struct {
	logger *logging.StructuredLogger

	client *http.Client

	config *ExternalBackendConfig
}

// ExternalBackendConfig holds external backend configuration.

type ExternalBackendConfig struct {
	// Connection settings.

	BaseURL string `yaml:"base_url"`

	APIVersion string `yaml:"api_version"`

	Timeout time.Duration `yaml:"timeout"`

	RetryAttempts int `yaml:"retry_attempts"`

	RetryDelay time.Duration `yaml:"retry_delay"`

	// Authentication.

	AuthType AuthenticationType `yaml:"auth_type"`

	AuthConfig interface{} `yaml:"auth_config"`

	// TLS settings.

	TLSConfig *ExternalTLSConfig `yaml:"tls_config"`

	// API endpoints.

	Endpoints *APIEndpoints `yaml:"endpoints"`

	// Certificate profiles.

	Profiles map[string]*CertificateProfile `yaml:"profiles"`

	// Feature flags.

	Features *FeatureConfig `yaml:"features"`
}

// AuthenticationType represents different authentication types.

type AuthenticationType string

const (

	// AuthTypeNone holds authtypenone value.

	AuthTypeNone AuthenticationType = "none"

	// AuthTypeAPIKey holds authtypeapikey value.

	AuthTypeAPIKey AuthenticationType = "api_key"

	// AuthTypeBearer holds authtypebearer value.

	AuthTypeBearer AuthenticationType = "bearer"

	// AuthTypeBasic holds authtypebasic value.

	AuthTypeBasic AuthenticationType = "basic"

	// AuthTypeMTLS holds authtypemtls value.

	AuthTypeMTLS AuthenticationType = "mtls"

	// AuthTypeOAuth2 holds authtypeoauth2 value.

	AuthTypeOAuth2 AuthenticationType = "oauth2"

	// AuthTypeCustom holds authtypecustom value.

	AuthTypeCustom AuthenticationType = "custom"
)

// ExternalTLSConfig holds TLS configuration for external connections.

type ExternalTLSConfig struct {
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`

	CACertificates []string `yaml:"ca_certificates"`

	ClientCertificate string `yaml:"client_certificate"`

	ClientKey string `yaml:"client_key"`

	ServerName string `yaml:"server_name"`

	MinTLSVersion string `yaml:"min_tls_version"`

	CipherSuites []string `yaml:"cipher_suites"`
}

// APIEndpoints defines external API endpoints.

type APIEndpoints struct {
	Issue string `yaml:"issue"`

	Revoke string `yaml:"revoke"`

	Renew string `yaml:"renew"`

	GetCertificate string `yaml:"get_certificate"`

	GetCAChain string `yaml:"get_ca_chain"`

	GetCRL string `yaml:"get_crl"`

	Health string `yaml:"health"`

	Profiles string `yaml:"profiles"`
}

// CertificateProfile defines certificate issuance profiles.

type CertificateProfile struct {
	Name string `yaml:"name"`

	Template string `yaml:"template"`

	MaxValidityDays int `yaml:"max_validity_days"`

	KeyUsages []string `yaml:"key_usages"`

	ExtKeyUsages []string `yaml:"ext_key_usages"`

	AllowedSANTypes []string `yaml:"allowed_san_types"`

	RequireApproval bool `yaml:"require_approval"`

	CustomFields map[string]interface{} `yaml:"custom_fields"`
}

// FeatureConfig defines supported features.

type FeatureConfig struct {
	SupportsRevocation bool `yaml:"supports_revocation"`

	SupportsRenewal bool `yaml:"supports_renewal"`

	SupportsTemplates bool `yaml:"supports_templates"`

	SupportsProfiles bool `yaml:"supports_profiles"`

	SupportsCRL bool `yaml:"supports_crl"`

	SupportsOCSP bool `yaml:"supports_ocsp"`

	SupportsStreaming bool `yaml:"supports_streaming"`
}

// Authentication configurations.

type APIKeyAuth struct {
	Key string `yaml:"key"`

	Value string `yaml:"value"`

	Location string `yaml:"location"` // header, query
}

// BearerAuth represents a bearerauth.

type BearerAuth struct {
	Token string `yaml:"token"`
}

// BasicAuth represents a basicauth.

type BasicAuth struct {
	Username string `yaml:"username"`

	Password string `yaml:"password"`
}

// OAuth2Auth represents a oauth2auth.

type OAuth2Auth struct {
	ClientID string `yaml:"client_id"`

	ClientSecret string `yaml:"client_secret"`

	TokenURL string `yaml:"token_url"`

	Scopes []string `yaml:"scopes"`

	TokenCache bool `yaml:"token_cache"`
}

// CustomAuth represents a customauth.

type CustomAuth struct {
	Headers map[string]string `yaml:"headers"`

	Query map[string]string `yaml:"query"`

	Body interface{} `yaml:"body"`
}

// External API request/response structures.

type ExternalIssueRequest struct {
	Profile string `json:"profile,omitempty"`

	Template string `json:"template,omitempty"`

	Subject map[string]string `json:"subject"`

	SANs *SubjectAlternateNames `json:"sans,omitempty"`

	ValidityDays int `json:"validity_days,omitempty"`

	KeySize int `json:"key_size,omitempty"`

	KeyType string `json:"key_type,omitempty"`

	KeyUsages []string `json:"key_usages,omitempty"`

	ExtKeyUsages []string `json:"ext_key_usages,omitempty"`

	CustomFields json.RawMessage `json:"custom_fields,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// SubjectAlternateNames represents a subjectalternatenames.

type SubjectAlternateNames struct {
	DNS []string `json:"dns,omitempty"`

	IP []string `json:"ip,omitempty"`

	Email []string `json:"email,omitempty"`

	URI []string `json:"uri,omitempty"`
}

// ExternalIssueResponse represents a externalissueresponse.

type ExternalIssueResponse struct {
	RequestID string `json:"request_id"`

	Certificate string `json:"certificate"`

	PrivateKey string `json:"private_key,omitempty"`

	CertificateChain []string `json:"certificate_chain,omitempty"`

	SerialNumber string `json:"serial_number"`

	Fingerprint string `json:"fingerprint,omitempty"`

	ExpiresAt string `json:"expires_at"`

	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// ExternalRevokeRequest represents a externalrevokerequest.

type ExternalRevokeRequest struct {
	SerialNumber string `json:"serial_number,omitempty"`

	Certificate string `json:"certificate,omitempty"`

	Reason int `json:"reason"`

	ReasonText string `json:"reason_text,omitempty"`
}

// ExternalRevokeResponse represents a externalrevokeresponse.

type ExternalRevokeResponse struct {
	SerialNumber string `json:"serial_number"`

	RevokedAt string `json:"revoked_at"`

	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// ExternalHealthResponse represents a externalhealthresponse.

type ExternalHealthResponse struct {
	Status string `json:"status"`

	Version string `json:"version"`

	Timestamp string `json:"timestamp"`

	Components json.RawMessage `json:"components,omitempty"`

	Features []string `json:"features,omitempty"`

	Metrics json.RawMessage `json:"metrics,omitempty"`
}

// NewExternalBackend creates a new external PKI backend.

func NewExternalBackend(logger *logging.StructuredLogger) (Backend, error) {
	return &ExternalBackend{
		logger: logger,
	}, nil
}

// Initialize initializes the external backend.

func (b *ExternalBackend) Initialize(ctx context.Context, config interface{}) error {
	extConfig, ok := config.(*ExternalBackendConfig)

	if !ok {
		return fmt.Errorf("invalid external backend config type")
	}

	b.config = extConfig

	// Validate configuration.

	if err := b.validateConfig(); err != nil {
		return fmt.Errorf("external backend config validation failed: %w", err)
	}

	// Initialize HTTP client.

	if err := b.initializeClient(); err != nil {
		return fmt.Errorf("HTTP client initialization failed: %w", err)
	}

	// Test connectivity.

	if err := b.testConnectivity(ctx); err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	b.logger.Info("external backend initialized successfully",

		"base_url", b.config.BaseURL,

		"api_version", b.config.APIVersion,

		"auth_type", b.config.AuthType)

	return nil
}

// HealthCheck performs health check on the external backend.

func (b *ExternalBackend) HealthCheck(ctx context.Context) error {
	if b.config.Endpoints.Health == "" {
		// Fallback to basic connectivity test.

		return b.testConnectivity(ctx)
	}

	resp, err := b.makeRequest(ctx, "GET", b.config.Endpoints.Health, nil)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}

	var healthResp ExternalHealthResponse

	if err := json.Unmarshal(resp, &healthResp); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	if healthResp.Status != "healthy" && healthResp.Status != "ok" {
		return fmt.Errorf("external system unhealthy: %s", healthResp.Status)
	}

	return nil
}

// IssueCertificate issues a certificate using the external PKI system.

func (b *ExternalBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	b.logger.Info("issuing certificate via external PKI",

		"request_id", req.ID,

		"common_name", req.CommonName,

		"base_url", b.config.BaseURL)

	// Build external request.

	extReq := &ExternalIssueRequest{
		RequestID: req.ID,

		Subject: map[string]string{
			"CN": req.CommonName,
		},

		ValidityDays: int(req.ValidityDuration.Hours() / 24),

		KeySize: req.KeySize,

		KeyType: "RSA",

		KeyUsages: req.KeyUsage,

		ExtKeyUsages: req.ExtKeyUsage,

		Metadata: map[string]string{
			"tenant_id": req.TenantID,
		},
	}

	// Add SANs.

	if len(req.DNSNames) > 0 || len(req.IPAddresses) > 0 || len(req.EmailAddresses) > 0 {

		extReq.SANs = &SubjectAlternateNames{
			DNS: req.DNSNames,

			IP: req.IPAddresses,

			Email: req.EmailAddresses,
		}

		// Convert URIs to strings.

		if len(req.URIs) > 0 {

			uris := make([]string, len(req.URIs))

			for i, uri := range req.URIs {
				uris[i] = uri.String()
			}

			extReq.SANs.URI = uris

		}

	}

	// Set profile or template if specified.

	if req.PolicyTemplate != "" {
		if b.config.Features.SupportsProfiles {
			extReq.Profile = req.PolicyTemplate
		} else if b.config.Features.SupportsTemplates {
			extReq.Template = req.PolicyTemplate
		}
	}

	// Make request.

	respData, err := b.makeRequest(ctx, "POST", b.config.Endpoints.Issue, extReq)
	if err != nil {
		return nil, fmt.Errorf("certificate issuance request failed: %w", err)
	}

	// Parse response.

	var extResp ExternalIssueResponse

	if err := json.Unmarshal(respData, &extResp); err != nil {
		return nil, fmt.Errorf("failed to parse issuance response: %w", err)
	}

	// Handle async responses.

	if extResp.Status == "pending" || extResp.Status == "processing" {
		return b.waitForCertificate(ctx, extResp.RequestID, req)
	}

	if extResp.Status != "issued" && extResp.Status != "success" {
		return nil, fmt.Errorf("certificate issuance failed: %s - %s", extResp.Status, extResp.Message)
	}

	// Parse certificate.

	cert, err := b.parseCertificate(extResp.Certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issued certificate: %w", err)
	}

	// Build response.

	response := &CertificateResponse{
		RequestID: req.ID,

		Certificate: cert,

		CertificatePEM: extResp.Certificate,

		PrivateKeyPEM: extResp.PrivateKey,

		SerialNumber: extResp.SerialNumber,

		Fingerprint: extResp.Fingerprint,

		IssuedBy: string(BackendExternal),

		Status: StatusIssued,

		Metadata: map[string]string{
			"external_request_id": extResp.RequestID,

			"tenant_id": req.TenantID,
		},

		CreatedAt: time.Now(),
	}

	// Parse expiration.

	if expiry, err := time.Parse(time.RFC3339, extResp.ExpiresAt); err == nil {
		response.ExpiresAt = expiry
	} else {
		response.ExpiresAt = cert.NotAfter
	}

	// Add certificate chain.

	if len(extResp.CertificateChain) > 0 {

		response.TrustChainPEM = extResp.CertificateChain

		if len(extResp.CertificateChain) > 0 {
			response.CACertificatePEM = extResp.CertificateChain[0]
		}

	}

	// Merge metadata.

	for k, v := range extResp.Metadata {
		response.Metadata[k] = v
	}

	b.logger.Info("certificate issued successfully via external PKI",

		"request_id", req.ID,

		"serial_number", response.SerialNumber,

		"expires_at", response.ExpiresAt)

	return response, nil
}

// RevokeCertificate revokes a certificate using the external PKI system.

func (b *ExternalBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	if !b.config.Features.SupportsRevocation {
		return fmt.Errorf("external backend does not support certificate revocation")
	}

	b.logger.Info("revoking certificate via external PKI",

		"serial_number", serialNumber,

		"reason", reason)

	// Build revoke request.

	revokeReq := &ExternalRevokeRequest{
		SerialNumber: serialNumber,

		Reason: reason,

		ReasonText: b.getRevocationReasonText(reason),
	}

	// Make request.

	respData, err := b.makeRequest(ctx, "POST", b.config.Endpoints.Revoke, revokeReq)
	if err != nil {
		return fmt.Errorf("certificate revocation request failed: %w", err)
	}

	// Parse response.

	var revokeResp ExternalRevokeResponse

	if err := json.Unmarshal(respData, &revokeResp); err != nil {
		return fmt.Errorf("failed to parse revocation response: %w", err)
	}

	if revokeResp.Status != "revoked" && revokeResp.Status != "success" {
		return fmt.Errorf("certificate revocation failed: %s - %s", revokeResp.Status, revokeResp.Message)
	}

	b.logger.Info("certificate revoked successfully via external PKI",

		"serial_number", serialNumber)

	return nil
}

// RenewCertificate renews a certificate using the external PKI system.

func (b *ExternalBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	if !b.config.Features.SupportsRenewal && b.config.Endpoints.Renew != "" {
		// Use dedicated renewal endpoint.

		return b.renewViaDedicatedEndpoint(ctx, req)
	}

	// Fallback to issuing a new certificate.

	response, err := b.IssueCertificate(ctx, req)
	if err != nil {
		return nil, err
	}

	response.Status = StatusRenewed

	return response, nil
}

// GetCAChain retrieves the CA certificate chain.

func (b *ExternalBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {
	if b.config.Endpoints.GetCAChain == "" {
		return nil, fmt.Errorf("CA chain endpoint not configured")
	}

	respData, err := b.makeRequest(ctx, "GET", b.config.Endpoints.GetCAChain, nil)
	if err != nil {
		return nil, fmt.Errorf("CA chain request failed: %w", err)
	}

	// Parse response - could be PEM or JSON.

	if strings.HasPrefix(string(respData), "-----BEGIN CERTIFICATE-----") {
		// PEM format.

		return b.parseMultipleCertificates(string(respData))
	}

	// JSON format.

	var jsonResp map[string]interface{}

	if err := json.Unmarshal(respData, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to parse CA chain response: %w", err)
	}

	// Extract certificates from various possible JSON formats.

	var pemData string

	if chain, ok := jsonResp["ca_chain"].(string); ok {
		pemData = chain
	} else if cert, ok := jsonResp["certificate"].(string); ok {
		pemData = cert
	} else if certs, ok := jsonResp["certificates"].([]interface{}); ok {

		var certStrings []string

		for _, c := range certs {
			if certStr, ok := c.(string); ok {
				certStrings = append(certStrings, certStr)
			}
		}

		pemData = strings.Join(certStrings, "\n")

	}

	if pemData == "" {
		return nil, fmt.Errorf("no certificate data found in response")
	}

	return b.parseMultipleCertificates(pemData)
}

// GetCRL retrieves the Certificate Revocation List.

func (b *ExternalBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {
	if !b.config.Features.SupportsCRL {
		return nil, fmt.Errorf("external backend does not support CRL")
	}

	if b.config.Endpoints.GetCRL == "" {
		return nil, fmt.Errorf("CRL endpoint not configured")
	}

	respData, err := b.makeRequest(ctx, "GET", b.config.Endpoints.GetCRL, nil)
	if err != nil {
		return nil, fmt.Errorf("CRL request failed: %w", err)
	}

	// Parse CRL - could be PEM or DER.

	return b.parseCRL(string(respData))
}

// GetBackendInfo returns backend information.

func (b *ExternalBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {
	info := &BackendInfo{
		Type: BackendExternal,

		Version: "external",

		Status: "ready",

		Features: b.GetSupportedFeatures(),
	}

	// Try to get additional info from health endpoint.

	if err := b.HealthCheck(ctx); err == nil {
		if b.config.Endpoints.Health != "" {

			respData, err := b.makeRequest(ctx, "GET", b.config.Endpoints.Health, nil)

			if err == nil {

				var healthResp ExternalHealthResponse

				if err := json.Unmarshal(respData, &healthResp); err == nil {

					info.Version = healthResp.Version

					info.Status = healthResp.Status

					info.Metrics = healthResp.Metrics

				}

			}

		}
	} else {
		info.Status = "unhealthy"
	}

	return info, nil
}

// GetSupportedFeatures returns supported features.

func (b *ExternalBackend) GetSupportedFeatures() []string {
	features := []string{
		"certificate_issuance",

		"external_integration",

		"custom_profiles",
	}

	if b.config.Features.SupportsRevocation {
		features = append(features, "certificate_revocation")
	}

	if b.config.Features.SupportsRenewal {
		features = append(features, "certificate_renewal")
	}

	if b.config.Features.SupportsTemplates {
		features = append(features, "certificate_templates")
	}

	if b.config.Features.SupportsProfiles {
		features = append(features, "certificate_profiles")
	}

	if b.config.Features.SupportsCRL {
		features = append(features, "crl_support")
	}

	if b.config.Features.SupportsOCSP {
		features = append(features, "ocsp_support")
	}

	return features
}

// Helper methods.

func (b *ExternalBackend) validateConfig() error {
	if b.config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if b.config.Endpoints == nil {
		return fmt.Errorf("API endpoints configuration is required")
	}

	if b.config.Endpoints.Issue == "" {
		return fmt.Errorf("certificate issuance endpoint is required")
	}

	if b.config.Timeout == 0 {
		b.config.Timeout = 30 * time.Second
	}

	if b.config.RetryAttempts == 0 {
		b.config.RetryAttempts = 3
	}

	if b.config.RetryDelay == 0 {
		b.config.RetryDelay = 5 * time.Second
	}

	return nil
}

func (b *ExternalBackend) initializeClient() error {
	// Create HTTP client with custom transport.

	transport := &http.Transport{}

	// Configure TLS if specified.

	if b.config.TLSConfig != nil {

		tlsConfig := &tls.Config{
			InsecureSkipVerify: b.config.TLSConfig.InsecureSkipVerify,
		}

		// Load CA certificates.

		if len(b.config.TLSConfig.CACertificates) > 0 {

			caCertPool := x509.NewCertPool()

			for _, caCert := range b.config.TLSConfig.CACertificates {
				caCertPool.AppendCertsFromPEM([]byte(caCert))
			}

			tlsConfig.RootCAs = caCertPool

		}

		// Load client certificate.

		if b.config.TLSConfig.ClientCertificate != "" && b.config.TLSConfig.ClientKey != "" {

			cert, err := tls.LoadX509KeyPair(

				b.config.TLSConfig.ClientCertificate,

				b.config.TLSConfig.ClientKey)
			if err != nil {
				return fmt.Errorf("failed to load client certificate: %w", err)
			}

			tlsConfig.Certificates = []tls.Certificate{cert}

		}

		// Set server name.

		if b.config.TLSConfig.ServerName != "" {
			tlsConfig.ServerName = b.config.TLSConfig.ServerName
		}

		// Set minimum TLS version.

		if b.config.TLSConfig.MinTLSVersion != "" {
			switch b.config.TLSConfig.MinTLSVersion {

			case "1.2":

				tlsConfig.MinVersion = tls.VersionTLS12

			case "1.3":

				tlsConfig.MinVersion = tls.VersionTLS13

			}
		}

		transport.TLSClientConfig = tlsConfig

	}

	b.client = &http.Client{
		Transport: transport,

		Timeout: b.config.Timeout,
	}

	return nil
}

func (b *ExternalBackend) testConnectivity(ctx context.Context) error {
	// Make a simple request to test connectivity.

	url := b.config.BaseURL

	if b.config.Endpoints.Health != "" {
		url = b.buildURL(b.config.Endpoints.Health)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return err
	}

	b.addAuthentication(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode >= 400 {
		return fmt.Errorf("connectivity test failed with status %d", resp.StatusCode)
	}

	return nil
}

func (b *ExternalBackend) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader

	if body != nil {

		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		reqBody = strings.NewReader(string(jsonData))

	}

	url := b.buildURL(endpoint)

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	b.addAuthentication(req)

	// Retry logic.

	var resp *http.Response

	var lastErr error

	for attempt := 0; attempt <= b.config.RetryAttempts; attempt++ {

		if attempt > 0 {
			select {

			case <-ctx.Done():

				return nil, ctx.Err()

			case <-time.After(b.config.RetryDelay * time.Duration(attempt)):

			}
		}

		resp, lastErr = b.client.Do(req)

		if lastErr == nil && resp.StatusCode < 500 {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if lastErr == nil {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		}

	}

	if lastErr != nil {
		return nil, lastErr
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode >= 400 {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))

	}

	return io.ReadAll(resp.Body)
}

func (b *ExternalBackend) buildURL(endpoint string) string {
	baseURL := strings.TrimRight(b.config.BaseURL, "/")

	endpoint = strings.TrimLeft(endpoint, "/")

	if b.config.APIVersion != "" {
		return fmt.Sprintf("%s/%s/%s", baseURL, strings.Trim(b.config.APIVersion, "/"), endpoint)
	}

	return fmt.Sprintf("%s/%s", baseURL, endpoint)
}

func (b *ExternalBackend) addAuthentication(req *http.Request) {
	switch b.config.AuthType {

	case AuthTypeAPIKey:

		if auth, ok := b.config.AuthConfig.(*APIKeyAuth); ok {
			if auth.Location == "query" {

				q := req.URL.Query()

				q.Add(auth.Key, auth.Value)

				req.URL.RawQuery = q.Encode()

			} else {
				req.Header.Set(auth.Key, auth.Value)
			}
		}

	case AuthTypeBearer:

		if auth, ok := b.config.AuthConfig.(*BearerAuth); ok {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		}

	case AuthTypeBasic:

		if auth, ok := b.config.AuthConfig.(*BasicAuth); ok {
			req.SetBasicAuth(auth.Username, auth.Password)
		}

	case AuthTypeCustom:

		if auth, ok := b.config.AuthConfig.(*CustomAuth); ok {

			for key, value := range auth.Headers {
				req.Header.Set(key, value)
			}

			if len(auth.Query) > 0 {

				q := req.URL.Query()

				for key, value := range auth.Query {
					q.Add(key, value)
				}

				req.URL.RawQuery = q.Encode()

			}

		}

	}
}

func (b *ExternalBackend) waitForCertificate(ctx context.Context, requestID string, req *CertificateRequest) (*CertificateResponse, error) {
	timeout := time.NewTimer(5 * time.Minute)

	defer timeout.Stop()

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return nil, ctx.Err()

		case <-timeout.C:

			return nil, fmt.Errorf("timeout waiting for certificate issuance")

		case <-ticker.C:

			// Poll for certificate status.

			status, err := b.getCertificateStatus(ctx, requestID)
			if err != nil {

				b.logger.Warn("failed to check certificate status", "request_id", requestID, "error", err)

				continue

			}

			switch status.Status {

			case "issued", "success":

				return b.buildCertificateResponse(status, req)

			case "failed", "error":

				return nil, fmt.Errorf("certificate issuance failed: %s", status.Message)

			case "pending", "processing":

				// Continue waiting.

				continue

			default:

				return nil, fmt.Errorf("unknown certificate status: %s", status.Status)

			}

		}
	}
}

func (b *ExternalBackend) getCertificateStatus(ctx context.Context, requestID string) (*ExternalIssueResponse, error) {
	endpoint := fmt.Sprintf("%s/%s", b.config.Endpoints.GetCertificate, requestID)

	respData, err := b.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var status ExternalIssueResponse

	if err := json.Unmarshal(respData, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

func (b *ExternalBackend) buildCertificateResponse(extResp *ExternalIssueResponse, req *CertificateRequest) (*CertificateResponse, error) {
	cert, err := b.parseCertificate(extResp.Certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	response := &CertificateResponse{
		RequestID: req.ID,

		Certificate: cert,

		CertificatePEM: extResp.Certificate,

		PrivateKeyPEM: extResp.PrivateKey,

		SerialNumber: extResp.SerialNumber,

		Fingerprint: extResp.Fingerprint,

		ExpiresAt: cert.NotAfter,

		IssuedBy: string(BackendExternal),

		Status: StatusIssued,

		Metadata: map[string]string{
			"external_request_id": extResp.RequestID,

			"tenant_id": req.TenantID,
		},

		CreatedAt: time.Now(),
	}

	if expiry, err := time.Parse(time.RFC3339, extResp.ExpiresAt); err == nil {
		response.ExpiresAt = expiry
	}

	if len(extResp.CertificateChain) > 0 {

		response.TrustChainPEM = extResp.CertificateChain

		response.CACertificatePEM = extResp.CertificateChain[0]

	}

	for k, v := range extResp.Metadata {
		response.Metadata[k] = v
	}

	return response, nil
}

func (b *ExternalBackend) renewViaDedicatedEndpoint(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	// Implementation for dedicated renewal endpoint.

	// This would be similar to IssueCertificate but using the renewal endpoint.

	return b.IssueCertificate(ctx, req)
}

func (b *ExternalBackend) parseCertificate(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))

	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	return x509.ParseCertificate(block.Bytes)
}

func (b *ExternalBackend) parseMultipleCertificates(certsPEM string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	remaining := []byte(certsPEM)

	for {

		block, rest := pem.Decode(remaining)

		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		certs = append(certs, cert)

		remaining = rest

	}

	return certs, nil
}

func (b *ExternalBackend) parseCRL(crlData string) (*pkix.CertificateList, error) {
	// Try PEM first.

	if strings.Contains(crlData, "-----BEGIN") {

		block, _ := pem.Decode([]byte(crlData))

		if block != nil {
			return x509.ParseCRL(block.Bytes)
		}

	}

	// Try DER format.

	return x509.ParseCRL([]byte(crlData))
}

func (b *ExternalBackend) getRevocationReasonText(reason int) string {
	reasons := map[int]string{
		0: "unspecified",

		1: "keyCompromise",

		2: "caCompromise",

		3: "affiliationChanged",

		4: "superseded",

		5: "cessationOfOperation",

		6: "certificateHold",

		8: "removeFromCRL",

		9: "privilegeWithdrawn",

		10: "aaCompromise",
	}

	if text, ok := reasons[reason]; ok {
		return text
	}

	return "unspecified"
}

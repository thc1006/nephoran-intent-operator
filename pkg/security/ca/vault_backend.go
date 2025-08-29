
package ca



import (

	"context"

	"crypto/sha256"

	"crypto/x509"

	"crypto/x509/pkix"

	"encoding/json"

	"encoding/pem"

	"fmt"

	"strings"

	"time"



	"github.com/hashicorp/vault/api"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

)



// VaultBackend implements the HashiCorp Vault PKI backend.

type VaultBackend struct {

	logger *logging.StructuredLogger

	client *api.Client

	config *VaultBackendConfig

}



// VaultBackendConfig holds Vault PKI backend configuration.

type VaultBackendConfig struct {

	// Vault connection.

	Address       string `yaml:"address"`

	Token         string `yaml:"token"`

	CACert        string `yaml:"ca_cert"`

	CAPath        string `yaml:"ca_path"`

	ClientCert    string `yaml:"client_cert"`

	ClientKey     string `yaml:"client_key"`

	TLSSkipVerify bool   `yaml:"tls_skip_verify"`

	Namespace     string `yaml:"namespace"`



	// PKI configuration.

	PKIMount  string `yaml:"pki_mount"`

	Role      string `yaml:"role"`

	IssuerRef string `yaml:"issuer_ref"`



	// Authentication.

	AuthMethod VaultAuthMethod `yaml:"auth_method"`

	AuthConfig interface{}     `yaml:"auth_config"`



	// Advanced settings.

	MaxLeaseTTL         time.Duration `yaml:"max_lease_ttl"`

	DefaultTTL          time.Duration `yaml:"default_ttl"`

	AllowedDomains      []string      `yaml:"allowed_domains"`

	AllowSubdomains     bool          `yaml:"allow_subdomains"`

	AllowAnyName        bool          `yaml:"allow_any_name"`

	EnforceHostnames    bool          `yaml:"enforce_hostnames"`

	AllowIPSANs         bool          `yaml:"allow_ip_sans"`

	ServerFlag          bool          `yaml:"server_flag"`

	ClientFlag          bool          `yaml:"client_flag"`

	CodeSigningFlag     bool          `yaml:"code_signing_flag"`

	EmailProtectionFlag bool          `yaml:"email_protection_flag"`



	// CRL settings.

	CRLConfig *VaultCRLConfig `yaml:"crl_config"`



	// OCSP settings.

	OCSPConfig *VaultOCSPConfig `yaml:"ocsp_config"`

}



// VaultAuthMethod represents Vault authentication methods.

type VaultAuthMethod string



const (

	// AuthMethodToken holds authmethodtoken value.

	AuthMethodToken VaultAuthMethod = "token"

	// AuthMethodAppRole holds authmethodapprole value.

	AuthMethodAppRole VaultAuthMethod = "approle"

	// AuthMethodKubernetes holds authmethodkubernetes value.

	AuthMethodKubernetes VaultAuthMethod = "kubernetes"

	// AuthMethodAWS holds authmethodaws value.

	AuthMethodAWS VaultAuthMethod = "aws"

	// AuthMethodGCP holds authmethodgcp value.

	AuthMethodGCP VaultAuthMethod = "gcp"

	// AuthMethodAzure holds authmethodazure value.

	AuthMethodAzure VaultAuthMethod = "azure"

	// AuthMethodLDAP holds authmethodldap value.

	AuthMethodLDAP VaultAuthMethod = "ldap"

)



// VaultCRLConfig holds CRL configuration for Vault.

type VaultCRLConfig struct {

	Enabled          bool          `yaml:"enabled"`

	ExpiryBuffer     time.Duration `yaml:"expiry_buffer"`

	DisableExpiry    bool          `yaml:"disable_expiry"`

	AutoRebuild      bool          `yaml:"auto_rebuild"`

	AutoRebuildGrace time.Duration `yaml:"auto_rebuild_grace"`

}



// VaultOCSPConfig holds OCSP configuration for Vault.

type VaultOCSPConfig struct {

	Enabled       bool     `yaml:"enabled"`

	OCSPServers   []string `yaml:"ocsp_servers"`

	ResponderCert string   `yaml:"responder_cert"`

	ResponderKey  string   `yaml:"responder_key"`

}



// VaultAppRoleAuth holds AppRole authentication configuration.

type VaultAppRoleAuth struct {

	RoleID    string `yaml:"role_id"`

	SecretID  string `yaml:"secret_id"`

	MountPath string `yaml:"mount_path"`

}



// VaultKubernetesAuth holds Kubernetes authentication configuration.

type VaultKubernetesAuth struct {

	Role           string `yaml:"role"`

	JWT            string `yaml:"jwt"`

	MountPath      string `yaml:"mount_path"`

	ServiceAccount string `yaml:"service_account"`

}



// VaultIssueRequest represents a Vault certificate issue request.

type VaultIssueRequest struct {

	CommonName        string `json:"common_name"`

	AltNames          string `json:"alt_names,omitempty"`

	IPSans            string `json:"ip_sans,omitempty"`

	URISans           string `json:"uri_sans,omitempty"`

	OtherSans         string `json:"other_sans,omitempty"`

	TTL               string `json:"ttl,omitempty"`

	Format            string `json:"format"`

	PrivateKeyFormat  string `json:"private_key_format,omitempty"`

	ExcludeCNFromSans bool   `json:"exclude_cn_from_sans"`

	NotAfter          string `json:"not_after,omitempty"`

}



// VaultIssueResponse represents Vault certificate issue response.

type VaultIssueResponse struct {

	Certificate    string   `json:"certificate"`

	PrivateKey     string   `json:"private_key"`

	PrivateKeyType string   `json:"private_key_type"`

	SerialNumber   string   `json:"serial_number"`

	CAChain        []string `json:"ca_chain"`

	IssuingCA      string   `json:"issuing_ca"`

	Expiration     int64    `json:"expiration"`

}



// VaultRevokeRequest represents a Vault certificate revoke request.

type VaultRevokeRequest struct {

	SerialNumber string `json:"serial_number"`

}



// NewVaultBackend creates a new Vault PKI backend.

func NewVaultBackend(logger *logging.StructuredLogger) (Backend, error) {

	return &VaultBackend{

		logger: logger,

	}, nil

}



// Initialize initializes the Vault backend.

func (b *VaultBackend) Initialize(ctx context.Context, config interface{}) error {

	vaultConfig, ok := config.(*VaultBackendConfig)

	if !ok {

		return fmt.Errorf("invalid Vault config type")

	}



	b.config = vaultConfig



	// Validate configuration.

	if err := b.validateConfig(); err != nil {

		return fmt.Errorf("Vault config validation failed: %w", err)

	}



	// Initialize Vault client.

	if err := b.initializeClient(); err != nil {

		return fmt.Errorf("Vault client initialization failed: %w", err)

	}



	// Authenticate with Vault.

	if err := b.authenticate(ctx); err != nil {

		return fmt.Errorf("Vault authentication failed: %w", err)

	}



	// Verify PKI mount and role.

	if err := b.verifyPKISetup(ctx); err != nil {

		return fmt.Errorf("PKI setup verification failed: %w", err)

	}



	b.logger.Info("Vault backend initialized successfully",

		"address", b.config.Address,

		"pki_mount", b.config.PKIMount,

		"role", b.config.Role)



	return nil

}



// HealthCheck performs health check on the Vault backend.

func (b *VaultBackend) HealthCheck(ctx context.Context) error {

	// Check Vault health.

	resp, err := b.client.Sys().Health()

	if err != nil {

		return fmt.Errorf("Vault health check failed: %w", err)

	}



	if resp.Sealed {

		return fmt.Errorf("Vault is sealed")

	}



	// Check token validity.

	tokenInfo, err := b.client.Auth().Token().LookupSelf()

	if err != nil {

		return fmt.Errorf("token validation failed: %w", err)

	}



	if tokenInfo == nil || tokenInfo.Data == nil {

		return fmt.Errorf("invalid token")

	}



	// Check PKI mount accessibility.

	mounts, err := b.client.Sys().ListMounts()

	if err != nil {

		return fmt.Errorf("failed to list mounts: %w", err)

	}



	mountPath := b.config.PKIMount + "/"

	if _, exists := mounts[mountPath]; !exists {

		return fmt.Errorf("PKI mount %s not found", b.config.PKIMount)

	}



	return nil

}



// IssueCertificate issues a certificate using Vault PKI.

func (b *VaultBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {

	b.logger.Info("issuing certificate via Vault PKI",

		"request_id", req.ID,

		"common_name", req.CommonName,

		"role", b.config.Role)



	// Prepare Vault request.

	vaultReq := &VaultIssueRequest{

		CommonName:        req.CommonName,

		Format:            "pem",

		PrivateKeyFormat:  "pkcs8",

		ExcludeCNFromSans: false,

	}



	// Set alternative names.

	if len(req.DNSNames) > 0 {

		vaultReq.AltNames = strings.Join(req.DNSNames, ",")

	}



	// Set IP SANs.

	if len(req.IPAddresses) > 0 {

		vaultReq.IPSans = strings.Join(req.IPAddresses, ",")

	}



	// Set URI SANs.

	if len(req.URIs) > 0 {

		uris := make([]string, len(req.URIs))

		for i, uri := range req.URIs {

			uris[i] = uri.String()

		}

		vaultReq.URISans = strings.Join(uris, ",")

	}



	// Set TTL.

	if req.ValidityDuration > 0 {

		vaultReq.TTL = req.ValidityDuration.String()

	}



	// Issue certificate.

	path := fmt.Sprintf("%s/issue/%s", b.config.PKIMount, b.config.Role)

	resp, err := b.client.Logical().Write(path, b.structToMap(vaultReq))

	if err != nil {

		return nil, fmt.Errorf("certificate issuance failed: %w", err)

	}



	if resp == nil || resp.Data == nil {

		return nil, fmt.Errorf("empty response from Vault")

	}



	// Parse response.

	var vaultResp VaultIssueResponse

	if err := b.mapToStruct(resp.Data, &vaultResp); err != nil {

		return nil, fmt.Errorf("failed to parse Vault response: %w", err)

	}



	// Parse certificate.

	cert, err := b.parseCertificate(vaultResp.Certificate)

	if err != nil {

		return nil, fmt.Errorf("failed to parse certificate: %w", err)

	}



	// Build response.

	response := &CertificateResponse{

		RequestID:        req.ID,

		Certificate:      cert,

		CertificatePEM:   vaultResp.Certificate,

		PrivateKeyPEM:    vaultResp.PrivateKey,

		CACertificatePEM: vaultResp.IssuingCA,

		TrustChainPEM:    vaultResp.CAChain,

		SerialNumber:     vaultResp.SerialNumber,

		Fingerprint:      b.calculateFingerprint(cert.Raw),

		ExpiresAt:        time.Unix(vaultResp.Expiration, 0),

		IssuedBy:         string(BackendVault),

		Status:           StatusIssued,

		Metadata: map[string]string{

			"vault_address":   b.config.Address,

			"vault_pki_mount": b.config.PKIMount,

			"vault_role":      b.config.Role,

			"tenant_id":       req.TenantID,

		},

		CreatedAt: time.Now(),

	}



	b.logger.Info("certificate issued successfully via Vault PKI",

		"request_id", req.ID,

		"serial_number", response.SerialNumber,

		"expires_at", response.ExpiresAt)



	return response, nil

}



// RevokeCertificate revokes a certificate using Vault PKI.

func (b *VaultBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {

	b.logger.Info("revoking certificate via Vault PKI",

		"serial_number", serialNumber,

		"reason", reason)



	// Prepare revoke request.

	revokeReq := &VaultRevokeRequest{

		SerialNumber: serialNumber,

	}



	// Revoke certificate.

	path := fmt.Sprintf("%s/revoke", b.config.PKIMount)

	_, err := b.client.Logical().Write(path, b.structToMap(revokeReq))

	if err != nil {

		return fmt.Errorf("certificate revocation failed: %w", err)

	}



	b.logger.Info("certificate revoked successfully via Vault PKI",

		"serial_number", serialNumber)



	return nil

}



// RenewCertificate renews a certificate (Vault issues a new certificate).

func (b *VaultBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {

	// Vault doesn't have direct renewal - we issue a new certificate.

	response, err := b.IssueCertificate(ctx, req)

	if err != nil {

		return nil, err

	}



	response.Status = StatusRenewed

	return response, nil

}



// GetCAChain retrieves the CA certificate chain.

func (b *VaultBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {

	b.logger.Debug("retrieving CA chain from Vault PKI")



	// Get CA certificate.

	path := fmt.Sprintf("%s/ca", b.config.PKIMount)

	resp, err := b.client.Logical().Read(path)

	if err != nil {

		return nil, fmt.Errorf("failed to get CA certificate: %w", err)

	}



	if resp == nil || resp.Data == nil {

		return nil, fmt.Errorf("empty CA response from Vault")

	}



	caCertPEM, ok := resp.Data["certificate"].(string)

	if !ok {

		return nil, fmt.Errorf("invalid CA certificate format")

	}



	// Parse CA certificate.

	caCert, err := b.parseCertificate(caCertPEM)

	if err != nil {

		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)

	}



	// Get CA chain.

	chainPath := fmt.Sprintf("%s/ca_chain", b.config.PKIMount)

	chainResp, err := b.client.Logical().Read(chainPath)

	if err != nil {

		// Fall back to just the CA certificate.

		return []*x509.Certificate{caCert}, nil

	}



	var chain []*x509.Certificate

	chain = append(chain, caCert)



	if chainResp != nil && chainResp.Data != nil {

		if chainPEM, ok := chainResp.Data["ca_chain"].(string); ok {

			chainCerts, err := b.parseMultipleCertificates(chainPEM)

			if err == nil {

				chain = append(chain, chainCerts...)

			}

		}

	}



	return chain, nil

}



// GetCRL retrieves the Certificate Revocation List.

func (b *VaultBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {

	b.logger.Debug("retrieving CRL from Vault PKI")



	path := fmt.Sprintf("%s/crl", b.config.PKIMount)

	resp, err := b.client.Logical().Read(path)

	if err != nil {

		return nil, fmt.Errorf("failed to get CRL: %w", err)

	}



	if resp == nil || resp.Data == nil {

		return nil, fmt.Errorf("empty CRL response from Vault")

	}



	crlPEM, ok := resp.Data["crl"].(string)

	if !ok {

		return nil, fmt.Errorf("invalid CRL format")

	}



	return b.parseCRL(crlPEM)

}



// GetBackendInfo returns backend information.

func (b *VaultBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {

	health, err := b.client.Sys().Health()

	if err != nil {

		return nil, fmt.Errorf("failed to get Vault health: %w", err)

	}



	// Get CA info.

	caChain, err := b.GetCAChain(ctx)

	var validUntil time.Time

	var issuer string

	if err == nil && len(caChain) > 0 {

		validUntil = caChain[0].NotAfter

		issuer = caChain[0].Subject.String()

	}



	status := "ready"

	if health.Sealed {

		status = "sealed"

	} else if !health.Initialized {

		status = "uninitialized"

	}



	return &BackendInfo{

		Type:       BackendVault,

		Version:    health.Version,

		Status:     status,

		Issuer:     issuer,

		ValidUntil: validUntil,

		Features:   b.GetSupportedFeatures(),

		Metrics: map[string]interface{}{

			"sealed":       health.Sealed,

			"initialized":  health.Initialized,

			"cluster_id":   health.ClusterID,

			"cluster_name": health.ClusterName,

		},

	}, nil

}



// GetSupportedFeatures returns supported features.

func (b *VaultBackend) GetSupportedFeatures() []string {

	return []string{

		"certificate_issuance",

		"certificate_revocation",

		"certificate_renewal",

		"ca_chain_retrieval",

		"crl_support",

		"role_based_issuance",

		"template_support",

		"automatic_serial_management",

		"key_usage_control",

		"custom_extensions",

	}

}



// Helper methods.



func (b *VaultBackend) validateConfig() error {

	if b.config.Address == "" {

		return fmt.Errorf("Vault address is required")

	}



	if b.config.PKIMount == "" {

		b.config.PKIMount = "pki" // Default PKI mount

	}



	if b.config.Role == "" {

		return fmt.Errorf("Vault PKI role is required")

	}



	if b.config.AuthMethod == "" {

		b.config.AuthMethod = AuthMethodToken // Default to token auth

	}



	if b.config.DefaultTTL == 0 {

		b.config.DefaultTTL = 24 * 30 * time.Hour // 30 days

	}



	return nil

}



func (b *VaultBackend) initializeClient() error {

	config := api.DefaultConfig()

	config.Address = b.config.Address



	if b.config.CACert != "" {

		tlsConfig := &api.TLSConfig{

			CACert: b.config.CACert,

		}

		if b.config.ClientCert != "" {

			tlsConfig.ClientCert = b.config.ClientCert

			tlsConfig.ClientKey = b.config.ClientKey

		}

		tlsConfig.Insecure = b.config.TLSSkipVerify

		config.ConfigureTLS(tlsConfig)

	}



	client, err := api.NewClient(config)

	if err != nil {

		return fmt.Errorf("failed to create Vault client: %w", err)

	}



	if b.config.Namespace != "" {

		client.SetNamespace(b.config.Namespace)

	}



	b.client = client

	return nil

}



func (b *VaultBackend) authenticate(ctx context.Context) error {

	switch b.config.AuthMethod {

	case AuthMethodToken:

		if b.config.Token == "" {

			return fmt.Errorf("token is required for token authentication")

		}

		b.client.SetToken(b.config.Token)



	case AuthMethodAppRole:

		authConfig, ok := b.config.AuthConfig.(*VaultAppRoleAuth)

		if !ok {

			return fmt.Errorf("invalid AppRole auth config")

		}

		return b.authenticateAppRole(ctx, authConfig)



	case AuthMethodKubernetes:

		authConfig, ok := b.config.AuthConfig.(*VaultKubernetesAuth)

		if !ok {

			return fmt.Errorf("invalid Kubernetes auth config")

		}

		return b.authenticateKubernetes(ctx, authConfig)



	default:

		return fmt.Errorf("unsupported auth method: %s", b.config.AuthMethod)

	}



	return nil

}



func (b *VaultBackend) authenticateAppRole(ctx context.Context, authConfig *VaultAppRoleAuth) error {

	path := "auth/approle/login"

	if authConfig.MountPath != "" {

		path = fmt.Sprintf("auth/%s/login", authConfig.MountPath)

	}



	data := map[string]interface{}{

		"role_id":   authConfig.RoleID,

		"secret_id": authConfig.SecretID,

	}



	resp, err := b.client.Logical().Write(path, data)

	if err != nil {

		return fmt.Errorf("AppRole authentication failed: %w", err)

	}



	if resp == nil || resp.Auth == nil {

		return fmt.Errorf("invalid AppRole authentication response")

	}



	b.client.SetToken(resp.Auth.ClientToken)

	return nil

}



func (b *VaultBackend) authenticateKubernetes(ctx context.Context, authConfig *VaultKubernetesAuth) error {

	path := "auth/kubernetes/login"

	if authConfig.MountPath != "" {

		path = fmt.Sprintf("auth/%s/login", authConfig.MountPath)

	}



	data := map[string]interface{}{

		"role": authConfig.Role,

		"jwt":  authConfig.JWT,

	}



	resp, err := b.client.Logical().Write(path, data)

	if err != nil {

		return fmt.Errorf("Kubernetes authentication failed: %w", err)

	}



	if resp == nil || resp.Auth == nil {

		return fmt.Errorf("invalid Kubernetes authentication response")

	}



	b.client.SetToken(resp.Auth.ClientToken)

	return nil

}



func (b *VaultBackend) verifyPKISetup(ctx context.Context) error {

	// Check if PKI mount exists.

	mounts, err := b.client.Sys().ListMounts()

	if err != nil {

		return fmt.Errorf("failed to list mounts: %w", err)

	}



	mountPath := b.config.PKIMount + "/"

	mount, exists := mounts[mountPath]

	if !exists {

		return fmt.Errorf("PKI mount %s not found", b.config.PKIMount)

	}



	if mount.Type != "pki" {

		return fmt.Errorf("mount %s is not a PKI mount (type: %s)", b.config.PKIMount, mount.Type)

	}



	// Check if role exists.

	rolePath := fmt.Sprintf("%s/roles/%s", b.config.PKIMount, b.config.Role)

	_, err = b.client.Logical().Read(rolePath)

	if err != nil {

		return fmt.Errorf("PKI role %s not found or not accessible: %w", b.config.Role, err)

	}



	return nil

}



func (b *VaultBackend) parseCertificate(certPEM string) (*x509.Certificate, error) {

	block, _ := pem.Decode([]byte(certPEM))

	if block == nil {

		return nil, fmt.Errorf("failed to decode certificate PEM")

	}



	return x509.ParseCertificate(block.Bytes)

}



func (b *VaultBackend) parseMultipleCertificates(certsPEM string) ([]*x509.Certificate, error) {

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



func (b *VaultBackend) parseCRL(crlPEM string) (*pkix.CertificateList, error) {

	block, _ := pem.Decode([]byte(crlPEM))

	if block == nil {

		return nil, fmt.Errorf("failed to decode CRL PEM")

	}



	return x509.ParseCRL(block.Bytes)

}



func (b *VaultBackend) calculateFingerprint(certBytes []byte) string {

	return fmt.Sprintf("%x", sha256.Sum256(certBytes))

}



func (b *VaultBackend) structToMap(s interface{}) map[string]interface{} {

	data, _ := json.Marshal(s)

	var result map[string]interface{}

	json.Unmarshal(data, &result)

	return result

}



func (b *VaultBackend) mapToStruct(data map[string]interface{}, s interface{}) error {

	jsonData, err := json.Marshal(data)

	if err != nil {

		return err

	}

	return json.Unmarshal(jsonData, s)

}


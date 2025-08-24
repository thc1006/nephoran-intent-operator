package ca

import (
	"context"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	certmanagerclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertManagerBackend implements the cert-manager backend
type CertManagerBackend struct {
	logger            *logging.StructuredLogger
	client            client.Client
	certManagerClient certmanagerclientset.Interface
	config            *CertManagerConfig
}

// CertManagerConfig holds cert-manager specific configuration
type CertManagerConfig struct {
	// Issuer configuration
	IssuerName      string `yaml:"issuer_name"`
	IssuerKind      string `yaml:"issuer_kind"` // Issuer or ClusterIssuer
	IssuerNamespace string `yaml:"issuer_namespace"`

	// Certificate configuration
	CertificateNamespace string `yaml:"certificate_namespace"`
	SecretNamePrefix     string `yaml:"secret_name_prefix"`

	// Advanced settings
	EnableApproval  bool          `yaml:"enable_approval"`
	DefaultDuration time.Duration `yaml:"default_duration"`
	RenewBefore     time.Duration `yaml:"renew_before"`
	RevisionLimit   int32         `yaml:"revision_limit"`

	// ACME settings (if using ACME issuer)
	ACMEConfig *ACMEConfig `yaml:"acme_config,omitempty"`

	// CA issuer settings (if using CA issuer)
	CAConfig *CAIssuerConfig `yaml:"ca_config,omitempty"`

	// Vault issuer settings (if using Vault issuer)
	VaultConfig *VaultIssuerConfig `yaml:"vault_config,omitempty"`
}

// ACMEConfig holds ACME-specific configuration
type ACMEConfig struct {
	Server         string         `yaml:"server"`
	Email          string         `yaml:"email"`
	PreferredChain string         `yaml:"preferred_chain"`
	Solvers        []ACMESolver   `yaml:"solvers"`
	PrivateKey     ACMEPrivateKey `yaml:"private_key"`
}

// ACMESolver defines ACME challenge solvers
type ACMESolver struct {
	DNS01    *DNS01Solver    `yaml:"dns01,omitempty"`
	HTTP01   *HTTP01Solver   `yaml:"http01,omitempty"`
	Selector *SolverSelector `yaml:"selector,omitempty"`
}

// DNS01Solver defines DNS-01 challenge solver
type DNS01Solver struct {
	Provider string                 `yaml:"provider"`
	Config   map[string]interface{} `yaml:"config"`
}

// HTTP01Solver defines HTTP-01 challenge solver
type HTTP01Solver struct {
	Ingress *HTTP01IngressSolver `yaml:"ingress,omitempty"`
	Gateway *HTTP01GatewaySolver `yaml:"gateway,omitempty"`
}

// HTTP01IngressSolver defines ingress-based HTTP-01 solver
type HTTP01IngressSolver struct {
	Class           string            `yaml:"class"`
	ServiceType     string            `yaml:"service_type"`
	IngressTemplate map[string]string `yaml:"ingress_template"`
	PodTemplate     map[string]string `yaml:"pod_template"`
}

// HTTP01GatewaySolver defines gateway-based HTTP-01 solver
type HTTP01GatewaySolver struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// SolverSelector defines selector for ACME solvers
type SolverSelector struct {
	DNSNames    []string          `yaml:"dns_names"`
	DNSZones    []string          `yaml:"dns_zones"`
	MatchLabels map[string]string `yaml:"match_labels"`
}

// ACMEPrivateKey defines ACME private key configuration
type ACMEPrivateKey struct {
	SecretRef      SecretRef `yaml:"secret_ref"`
	Algorithm      string    `yaml:"algorithm"`
	Size           int       `yaml:"size"`
	RotationPolicy string    `yaml:"rotation_policy"`
}

// CAIssuerConfig holds CA issuer specific configuration
type CAIssuerConfig struct {
	SecretName            string   `yaml:"secret_name"`
	CRLDistributionPoints []string `yaml:"crl_distribution_points"`
	OCSPServers           []string `yaml:"ocsp_servers"`
}

// VaultIssuerConfig holds Vault issuer specific configuration
type VaultIssuerConfig struct {
	Server    string    `yaml:"server"`
	Path      string    `yaml:"path"`
	Role      string    `yaml:"role"`
	Auth      VaultAuth `yaml:"auth"`
	CABundle  string    `yaml:"ca_bundle"`
	Namespace string    `yaml:"namespace"`
}

// VaultAuth defines Vault authentication configuration
type VaultAuth struct {
	TokenSecretRef *SecretRef       `yaml:"token_secret_ref,omitempty"`
	AppRole        *VaultAppRole    `yaml:"app_role,omitempty"`
	Kubernetes     *VaultKubernetes `yaml:"kubernetes,omitempty"`
}

// VaultAppRole defines Vault AppRole authentication
type VaultAppRole struct {
	Path      string    `yaml:"path"`
	RoleID    string    `yaml:"role_id"`
	SecretRef SecretRef `yaml:"secret_ref"`
}

// VaultKubernetes defines Vault Kubernetes authentication
type VaultKubernetes struct {
	Path              string             `yaml:"path"`
	Role              string             `yaml:"role"`
	ServiceAccountRef *ServiceAccountRef `yaml:"service_account_ref,omitempty"`
}

// SecretRef references a Kubernetes secret
type SecretRef struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// ServiceAccountRef references a Kubernetes service account
type ServiceAccountRef struct {
	Name string `yaml:"name"`
}

// NewCertManagerBackend creates a new cert-manager backend
func NewCertManagerBackend(logger *logging.StructuredLogger, client client.Client) (Backend, error) {
	// Create cert-manager client
	certManagerClient, err := certmanagerclientset.NewForConfig(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert-manager client: %w", err)
	}

	return &CertManagerBackend{
		logger:            logger,
		client:            client,
		certManagerClient: certManagerClient,
	}, nil
}

// Initialize initializes the cert-manager backend
func (b *CertManagerBackend) Initialize(ctx context.Context, config interface{}) error {
	cmConfig, ok := config.(*CertManagerConfig)
	if !ok {
		return fmt.Errorf("invalid cert-manager config type")
	}

	b.config = cmConfig

	// Validate configuration
	if err := b.validateConfig(); err != nil {
		return fmt.Errorf("cert-manager config validation failed: %w", err)
	}

	// Check if issuer exists and is ready
	if err := b.verifyIssuer(ctx); err != nil {
		return fmt.Errorf("issuer verification failed: %w", err)
	}

	b.logger.Info("cert-manager backend initialized successfully",
		"issuer", b.config.IssuerName,
		"kind", b.config.IssuerKind,
		"namespace", b.config.IssuerNamespace)

	return nil
}

// HealthCheck performs health check on the cert-manager backend
func (b *CertManagerBackend) HealthCheck(ctx context.Context) error {
	// Check if issuer is ready
	return b.verifyIssuer(ctx)
}

// IssueCertificate issues a certificate using cert-manager
func (b *CertManagerBackend) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	b.logger.Info("issuing certificate via cert-manager",
		"request_id", req.ID,
		"common_name", req.CommonName,
		"issuer", b.config.IssuerName)

	// Create cert-manager Certificate resource
	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.generateCertificateName(req),
			Namespace: b.config.CertificateNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "nephoran-intent-operator",
				"app.kubernetes.io/component":  "certificate",
				"app.kubernetes.io/managed-by": "nephoran-ca-manager",
				"nephoran.io/request-id":       req.ID,
				"nephoran.io/tenant-id":        req.TenantID,
			},
			Annotations: map[string]string{
				"nephoran.io/request-id": req.ID,
				"nephoran.io/issued-by":  "cert-manager",
			},
		},
		Spec: certmanagerv1.CertificateSpec{
			CommonName:     req.CommonName,
			DNSNames:       req.DNSNames,
			IPAddresses:    req.IPAddresses,
			EmailAddresses: req.EmailAddresses,
			Duration:       &metav1.Duration{Duration: req.ValidityDuration},
			RenewBefore:    &metav1.Duration{Duration: b.config.RenewBefore},
			Subject: &certmanagerv1.X509Subject{
				Organizations: []string{"Nephoran"},
			},
			PrivateKey: &certmanagerv1.CertificatePrivateKey{
				Algorithm: certmanagerv1.RSAKeyAlgorithm,
				Size:      req.KeySize,
				Encoding:  certmanagerv1.PKCS1,
			},
			Usages:     b.convertKeyUsages(req.KeyUsage, req.ExtKeyUsage),
			SecretName: b.generateSecretName(req),
			IssuerRef: cmmetav1.ObjectReference{
				Name: b.config.IssuerName,
				Kind: b.config.IssuerKind,
			},
			RevisionHistoryLimit: &b.config.RevisionLimit,
		},
	}

	// Add URIs if specified
	if len(req.URIs) > 0 {
		uris := make([]string, len(req.URIs))
		for i, uri := range req.URIs {
			uris[i] = uri.String()
		}
		certificate.Spec.URIs = uris
	}

	// Set issuer namespace if specified
	if b.config.IssuerNamespace != "" {
		certificate.Spec.IssuerRef.Group = certmanagerv1.SchemeGroupVersion.Group
	}

	// Create the certificate
	createdCert, err := b.certManagerClient.CertmanagerV1().
		Certificates(b.config.CertificateNamespace).
		Create(ctx, certificate, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create cert-manager Certificate: %w", err)
	}

	// Wait for certificate to be ready
	response, err := b.waitForCertificateReady(ctx, createdCert.Name, req)
	if err != nil {
		return nil, fmt.Errorf("certificate issuance failed: %w", err)
	}

	b.logger.Info("certificate issued successfully via cert-manager",
		"request_id", req.ID,
		"certificate_name", createdCert.Name,
		"serial_number", response.SerialNumber)

	return response, nil
}

// RevokeCertificate revokes a certificate
func (b *CertManagerBackend) RevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	// cert-manager doesn't directly support certificate revocation
	// This would depend on the underlying issuer (ACME, Vault, etc.)
	b.logger.Warn("certificate revocation via cert-manager is issuer-dependent",
		"serial_number", serialNumber,
		"reason", reason)

	// For CA issuers, we might need to maintain our own CRL
	// For ACME issuers, revocation happens at the CA level
	// For Vault issuers, we can use Vault's revocation API

	return fmt.Errorf("revocation not implemented for cert-manager backend")
}

// RenewCertificate renews a certificate
func (b *CertManagerBackend) RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {
	b.logger.Info("renewing certificate via cert-manager",
		"request_id", req.ID,
		"common_name", req.CommonName)

	// Find existing certificate by metadata
	certName := b.generateCertificateName(req)
	existingCert, err := b.certManagerClient.CertmanagerV1().
		Certificates(b.config.CertificateNamespace).
		Get(ctx, certName, metav1.GetOptions{})
	if err != nil {
		// If certificate doesn't exist, issue a new one
		return b.IssueCertificate(ctx, req)
	}

	// Trigger renewal by updating the Certificate resource
	existingCert.Annotations["nephoran.io/force-renewal"] = time.Now().Format(time.RFC3339)

	_, err = b.certManagerClient.CertmanagerV1().
		Certificates(b.config.CertificateNamespace).
		Update(ctx, existingCert, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to trigger certificate renewal: %w", err)
	}

	// Wait for renewed certificate to be ready
	response, err := b.waitForCertificateReady(ctx, existingCert.Name, req)
	if err != nil {
		return nil, fmt.Errorf("certificate renewal failed: %w", err)
	}

	response.Status = StatusRenewed
	return response, nil
}

// GetCAChain retrieves the CA certificate chain
func (b *CertManagerBackend) GetCAChain(ctx context.Context) ([]*x509.Certificate, error) {
	// The CA chain depends on the issuer type
	// For CA issuers, we can get the CA certificate from the secret
	// For ACME issuers, the CA chain comes from the ACME server
	// This is a simplified implementation

	b.logger.Debug("retrieving CA chain for cert-manager backend")

	// This would be issuer-specific logic
	return nil, fmt.Errorf("CA chain retrieval not implemented for cert-manager backend")
}

// GetCRL retrieves the Certificate Revocation List
func (b *CertManagerBackend) GetCRL(ctx context.Context) (*pkix.CertificateList, error) {
	// CRL availability depends on the issuer
	return nil, fmt.Errorf("CRL retrieval not implemented for cert-manager backend")
}

// GetBackendInfo returns backend information
func (b *CertManagerBackend) GetBackendInfo(ctx context.Context) (*BackendInfo, error) {
	return &BackendInfo{
		Type:     BackendCertManager,
		Version:  "cert-manager v1.x",
		Status:   "ready",
		Issuer:   fmt.Sprintf("%s/%s", b.config.IssuerKind, b.config.IssuerName),
		Features: b.GetSupportedFeatures(),
	}, nil
}

// GetSupportedFeatures returns supported features
func (b *CertManagerBackend) GetSupportedFeatures() []string {
	return []string{
		"certificate_issuance",
		"certificate_renewal",
		"automatic_renewal",
		"multiple_issuers",
		"kubernetes_integration",
		"secret_management",
	}
}

// Helper methods

func (b *CertManagerBackend) validateConfig() error {
	if b.config.IssuerName == "" {
		return fmt.Errorf("issuer name is required")
	}

	if b.config.IssuerKind == "" {
		b.config.IssuerKind = "Issuer" // Default to Issuer
	}

	if b.config.IssuerKind != "Issuer" && b.config.IssuerKind != "ClusterIssuer" {
		return fmt.Errorf("issuer kind must be 'Issuer' or 'ClusterIssuer'")
	}

	if b.config.CertificateNamespace == "" {
		b.config.CertificateNamespace = "default"
	}

	if b.config.SecretNamePrefix == "" {
		b.config.SecretNamePrefix = "nephoran-cert"
	}

	if b.config.DefaultDuration == 0 {
		b.config.DefaultDuration = 24 * 30 * time.Hour // 30 days
	}

	if b.config.RenewBefore == 0 {
		b.config.RenewBefore = 24 * 7 * time.Hour // 7 days
	}

	if b.config.RevisionLimit == 0 {
		b.config.RevisionLimit = 3
	}

	return nil
}

func (b *CertManagerBackend) verifyIssuer(ctx context.Context) error {
	switch b.config.IssuerKind {
	case "Issuer":
		issuer, err := b.certManagerClient.CertmanagerV1().
			Issuers(b.config.IssuerNamespace).
			Get(ctx, b.config.IssuerName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("issuer not found: %w", err)
		}

		// Check if issuer is ready
		for _, condition := range issuer.Status.Conditions {
			if condition.Type == certmanagerv1.IssuerConditionReady {
				if condition.Status != certmanagerv1.ConditionTrue {
					return fmt.Errorf("issuer not ready: %s", condition.Message)
				}
				return nil
			}
		}
		return fmt.Errorf("issuer readiness status unknown")

	case "ClusterIssuer":
		clusterIssuer, err := b.certManagerClient.CertmanagerV1().
			ClusterIssuers().
			Get(ctx, b.config.IssuerName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("cluster issuer not found: %w", err)
		}

		// Check if cluster issuer is ready
		for _, condition := range clusterIssuer.Status.Conditions {
			if condition.Type == certmanagerv1.IssuerConditionReady {
				if condition.Status != certmanagerv1.ConditionTrue {
					return fmt.Errorf("cluster issuer not ready: %s", condition.Message)
				}
				return nil
			}
		}
		return fmt.Errorf("cluster issuer readiness status unknown")

	default:
		return fmt.Errorf("unknown issuer kind: %s", b.config.IssuerKind)
	}
}

func (b *CertManagerBackend) generateCertificateName(req *CertificateRequest) string {
	return fmt.Sprintf("%s-%s", b.config.SecretNamePrefix, req.ID)
}

func (b *CertManagerBackend) generateSecretName(req *CertificateRequest) string {
	return fmt.Sprintf("%s-%s-tls", b.config.SecretNamePrefix, req.ID)
}

func (b *CertManagerBackend) convertKeyUsages(keyUsage, extKeyUsage []string) []certmanagerv1.KeyUsage {
	var usages []certmanagerv1.KeyUsage

	// Convert standard key usages
	for _, usage := range keyUsage {
		switch usage {
		case "digital_signature":
			usages = append(usages, certmanagerv1.UsageDigitalSignature)
		case "key_encipherment":
			usages = append(usages, certmanagerv1.UsageKeyEncipherment)
		case "key_agreement":
			usages = append(usages, certmanagerv1.UsageKeyAgreement)
		case "data_encipherment":
			usages = append(usages, certmanagerv1.UsageDataEncipherment)
		case "cert_sign":
			usages = append(usages, certmanagerv1.UsageCertSign)
		case "crl_sign":
			usages = append(usages, certmanagerv1.UsageCRLSign)
		}
	}

	// Convert extended key usages
	for _, usage := range extKeyUsage {
		switch usage {
		case "server_auth":
			usages = append(usages, certmanagerv1.UsageServerAuth)
		case "client_auth":
			usages = append(usages, certmanagerv1.UsageClientAuth)
		case "code_signing":
			usages = append(usages, certmanagerv1.UsageCodeSigning)
		case "email_protection":
			usages = append(usages, certmanagerv1.UsageEmailProtection)
		case "timestamping":
			usages = append(usages, certmanagerv1.UsageTimestamping)
		case "ocsp_signing":
			usages = append(usages, certmanagerv1.UsageOCSPSigning)
		}
	}

	// Default usages if none specified
	if len(usages) == 0 {
		usages = []certmanagerv1.KeyUsage{
			certmanagerv1.UsageDigitalSignature,
			certmanagerv1.UsageKeyEncipherment,
			certmanagerv1.UsageServerAuth,
			certmanagerv1.UsageClientAuth,
		}
	}

	return usages
}

func (b *CertManagerBackend) waitForCertificateReady(ctx context.Context, certName string, req *CertificateRequest) (*CertificateResponse, error) {
	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, fmt.Errorf("timeout waiting for certificate to be ready")
		case <-ticker.C:
			cert, err := b.certManagerClient.CertmanagerV1().
				Certificates(b.config.CertificateNamespace).
				Get(ctx, certName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			// Check if certificate is ready
			for _, condition := range cert.Status.Conditions {
				if condition.Type == certmanagerv1.CertificateConditionReady {
					if condition.Status == certmanagerv1.ConditionTrue {
						// Certificate is ready, fetch the secret
						return b.buildCertificateResponse(ctx, cert, req)
					} else if condition.Status == certmanagerv1.ConditionFalse {
						return nil, fmt.Errorf("certificate failed: %s", condition.Message)
					}
				}
			}
		}
	}
}

func (b *CertManagerBackend) buildCertificateResponse(ctx context.Context, cert *certmanagerv1.Certificate, req *CertificateRequest) (*CertificateResponse, error) {
	// Fetch the certificate secret
	secret, err := b.client.Get(ctx, client.ObjectKey{
		Namespace: b.config.CertificateNamespace,
		Name:      cert.Spec.SecretName,
	}, &corev1.Secret{})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate secret: %w", err)
	}

	// Extract certificate data
	certPEM := secret.Data["tls.crt"]
	keyPEM := secret.Data["tls.key"]
	caPEM := secret.Data["ca.crt"]

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	response := &CertificateResponse{
		RequestID:        req.ID,
		Certificate:      parsedCert,
		CertificatePEM:   string(certPEM),
		PrivateKeyPEM:    string(keyPEM),
		CACertificatePEM: string(caPEM),
		SerialNumber:     parsedCert.SerialNumber.String(),
		Fingerprint:      fmt.Sprintf("%x", parsedCert.Fingerprint(crypto.SHA256)),
		ExpiresAt:        parsedCert.NotAfter,
		IssuedBy:         string(BackendCertManager),
		Status:           StatusIssued,
		Metadata: map[string]string{
			"cert_manager_certificate": cert.Name,
			"cert_manager_secret":      cert.Spec.SecretName,
			"tenant_id":                req.TenantID,
		},
		CreatedAt: time.Now(),
	}

	return response, nil
}


package ca



import (

	"context"

	"crypto/rand"

	"encoding/hex"

	"fmt"

	"time"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"



	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"



	"sigs.k8s.io/controller-runtime/pkg/client"

)



// CAIntegration provides integration between CA management and existing security components.

type CAIntegration struct {

	caManager  *CAManager

	tlsManager *security.TLSManager

	client     client.Client

	logger     *logging.StructuredLogger

	config     *IntegrationConfig

}



// IntegrationConfig configures CA integration with existing systems.

type IntegrationConfig struct {

	// TLS integration.

	TLSIntegration *TLSIntegrationConfig `yaml:"tls_integration"`



	// RBAC integration.

	RBACIntegration *RBACIntegrationConfig `yaml:"rbac_integration"`



	// Secret management.

	SecretIntegration *SecretIntegrationConfig `yaml:"secret_integration"`



	// Service mesh integration.

	ServiceMeshConfig *ServiceMeshIntegration `yaml:"service_mesh"`



	// Ingress integration.

	IngressConfig *IngressIntegration `yaml:"ingress"`



	// Operator integration.

	OperatorConfig *OperatorIntegration `yaml:"operator"`

}



// TLSIntegrationConfig configures TLS integration.

type TLSIntegrationConfig struct {

	Enabled               bool          `yaml:"enabled"`

	AutoUpdateTLSConfig   bool          `yaml:"auto_update_tls_config"`

	TLSConfigSecretName   string        `yaml:"tls_config_secret_name"`

	CertificateNamespaces []string      `yaml:"certificate_namespaces"`

	DefaultCertificateTTL time.Duration `yaml:"default_certificate_ttl"`

	HotReloadEnabled      bool          `yaml:"hot_reload_enabled"`

}



// RBACIntegrationConfig configures RBAC integration.

type RBACIntegrationConfig struct {

	Enabled                   bool   `yaml:"enabled"`

	CertificateBasedAuth      bool   `yaml:"certificate_based_auth"`

	ClientCertNamespace       string `yaml:"client_cert_namespace"`

	ServiceAccountIntegration bool   `yaml:"service_account_integration"`

}



// SecretIntegrationConfig configures secret management integration.

type SecretIntegrationConfig struct {

	Enabled            bool              `yaml:"enabled"`

	AutoCreateSecrets  bool              `yaml:"auto_create_secrets"`

	SecretNameTemplate string            `yaml:"secret_name_template"`

	SecretLabels       map[string]string `yaml:"secret_labels"`

	SecretAnnotations  map[string]string `yaml:"secret_annotations"`

	EncryptionEnabled  bool              `yaml:"encryption_enabled"`

}



// ServiceMeshIntegration configures service mesh integration.

type ServiceMeshIntegration struct {

	Enabled                bool              `yaml:"enabled"`

	MeshType               string            `yaml:"mesh_type"` // istio, linkerd, consul

	AutoInjectCertificates bool              `yaml:"auto_inject_certificates"`

	NamespaceSelector      map[string]string `yaml:"namespace_selector"`

	WorkloadSelector       map[string]string `yaml:"workload_selector"`

	CertificateTemplate    string            `yaml:"certificate_template"`

}



// IngressIntegration configures ingress integration.

type IngressIntegration struct {

	Enabled               bool   `yaml:"enabled"`

	IngressClass          string `yaml:"ingress_class"`

	AutoGenerateCerts     bool   `yaml:"auto_generate_certs"`

	CertificateAnnotation string `yaml:"certificate_annotation"`

	TLSSecretTemplate     string `yaml:"tls_secret_template"`

}



// OperatorIntegration configures operator-level integration.

type OperatorIntegration struct {

	Enabled             bool   `yaml:"enabled"`

	WebhookCertificates bool   `yaml:"webhook_certificates"`

	OperatorNamespace   string `yaml:"operator_namespace"`

	WebhookServiceName  string `yaml:"webhook_service_name"`

	CertRotationDays    int    `yaml:"cert_rotation_days"`

}



// CertificateRequest represents a certificate request from integration.

type IntegrationCertificateRequest struct {

	Name         string            `yaml:"name"`

	Namespace    string            `yaml:"namespace"`

	CommonName   string            `yaml:"common_name"`

	DNSNames     []string          `yaml:"dns_names"`

	IPAddresses  []string          `yaml:"ip_addresses"`

	TenantID     string            `yaml:"tenant_id"`

	SecretName   string            `yaml:"secret_name"`

	Labels       map[string]string `yaml:"labels"`

	Annotations  map[string]string `yaml:"annotations"`

	ValidityDays int               `yaml:"validity_days"`

	RenewBefore  int               `yaml:"renew_before"`

	Backend      string            `yaml:"backend"`

	Template     string            `yaml:"template"`

	AutoRenew    bool              `yaml:"auto_renew"`

}



// NewCAIntegration creates a new CA integration.

func NewCAIntegration(caManager *CAManager, client client.Client, logger *logging.StructuredLogger, config *IntegrationConfig) *CAIntegration {

	return &CAIntegration{

		caManager: caManager,

		client:    client,

		logger:    logger,

		config:    config,

	}

}



// Initialize initializes the CA integration.

func (i *CAIntegration) Initialize(ctx context.Context) error {

	i.logger.Info("initializing CA integration")



	// Initialize TLS integration.

	if i.config.TLSIntegration != nil && i.config.TLSIntegration.Enabled {

		if err := i.initializeTLSIntegration(ctx); err != nil {

			return fmt.Errorf("failed to initialize TLS integration: %w", err)

		}

	}



	// Initialize RBAC integration.

	if i.config.RBACIntegration != nil && i.config.RBACIntegration.Enabled {

		if err := i.initializeRBACIntegration(ctx); err != nil {

			return fmt.Errorf("failed to initialize RBAC integration: %w", err)

		}

	}



	// Initialize operator integration (for webhook certificates).

	if i.config.OperatorConfig != nil && i.config.OperatorConfig.Enabled {

		if err := i.initializeOperatorIntegration(ctx); err != nil {

			return fmt.Errorf("failed to initialize operator integration: %w", err)

		}

	}



	i.logger.Info("CA integration initialized successfully")

	return nil

}



// RequestCertificate requests a certificate through the integration layer.

func (i *CAIntegration) RequestCertificate(ctx context.Context, req *IntegrationCertificateRequest) (*CertificateResponse, error) {

	i.logger.Info("processing certificate request through integration",

		"name", req.Name,

		"namespace", req.Namespace,

		"common_name", req.CommonName)



	// Generate a unique request ID.

	randBytes := make([]byte, 16)

	if _, err := rand.Read(randBytes); err != nil {

		return nil, fmt.Errorf("failed to generate request ID: %w", err)

	}

	requestID := hex.EncodeToString(randBytes)



	// Convert integration request to CA manager request.

	caReq := &CertificateRequest{

		ID:               requestID,

		TenantID:         req.TenantID,

		CommonName:       req.CommonName,

		DNSNames:         req.DNSNames,

		IPAddresses:      req.IPAddresses,

		ValidityDuration: time.Duration(req.ValidityDays) * 24 * time.Hour,

		KeySize:          2048, // Default key size

		Backend:          CABackendType(req.Backend),

		PolicyTemplate:   req.Template,

		AutoRenew:        req.AutoRenew,

		Metadata: map[string]string{

			"integration_name":      req.Name,

			"integration_namespace": req.Namespace,

			"secret_name":           req.SecretName,

			"tenant_id":             req.TenantID,

		},

	}



	// Set default key usages for integration requests.

	caReq.KeyUsage = []string{"digital_signature", "key_encipherment"}

	caReq.ExtKeyUsage = []string{"server_auth", "client_auth"}



	// Issue certificate through CA manager.

	response, err := i.caManager.IssueCertificate(ctx, caReq)

	if err != nil {

		return nil, fmt.Errorf("certificate issuance failed: %w", err)

	}



	// Handle secret creation if enabled.

	if i.config.SecretIntegration != nil && i.config.SecretIntegration.AutoCreateSecrets {

		if err := i.createCertificateSecret(ctx, req, response); err != nil {

			i.logger.Warn("failed to create certificate secret",

				"error", err,

				"certificate", response.SerialNumber)

		}

	}



	// Update TLS configuration if enabled.

	if i.config.TLSIntegration != nil && i.config.TLSIntegration.AutoUpdateTLSConfig {

		if err := i.updateTLSConfiguration(ctx, response); err != nil {

			i.logger.Warn("failed to update TLS configuration",

				"error", err,

				"certificate", response.SerialNumber)

		}

	}



	i.logger.Info("certificate request completed successfully",

		"name", req.Name,

		"serial_number", response.SerialNumber,

		"expires_at", response.ExpiresAt)



	return response, nil

}



// RenewCertificate renews a certificate through the integration layer.

func (i *CAIntegration) RenewCertificate(ctx context.Context, serialNumber string) (*CertificateResponse, error) {

	i.logger.Info("renewing certificate through integration",

		"serial_number", serialNumber)



	response, err := i.caManager.RenewCertificate(ctx, serialNumber)

	if err != nil {

		return nil, fmt.Errorf("certificate renewal failed: %w", err)

	}



	// Update associated secrets.

	if i.config.SecretIntegration != nil && i.config.SecretIntegration.AutoCreateSecrets {

		if err := i.updateCertificateSecret(ctx, response); err != nil {

			i.logger.Warn("failed to update certificate secret",

				"error", err,

				"certificate", response.SerialNumber)

		}

	}



	i.logger.Info("certificate renewed successfully",

		"serial_number", response.SerialNumber,

		"new_expires_at", response.ExpiresAt)



	return response, nil

}



// GetOperatorWebhookCertificate generates/retrieves webhook certificates for the operator.

func (i *CAIntegration) GetOperatorWebhookCertificate(ctx context.Context) (*CertificateResponse, error) {

	if i.config.OperatorConfig == nil || !i.config.OperatorConfig.WebhookCertificates {

		return nil, fmt.Errorf("operator webhook certificates not enabled")

	}



	serviceName := i.config.OperatorConfig.WebhookServiceName

	namespace := i.config.OperatorConfig.OperatorNamespace

	if serviceName == "" {

		serviceName = "nephoran-webhook-service"

	}

	if namespace == "" {

		namespace = "nephoran-system"

	}



	req := &IntegrationCertificateRequest{

		Name:       "operator-webhook",

		Namespace:  namespace,

		CommonName: fmt.Sprintf("%s.%s.svc", serviceName, namespace),

		DNSNames: []string{

			serviceName,

			fmt.Sprintf("%s.%s", serviceName, namespace),

			fmt.Sprintf("%s.%s.svc", serviceName, namespace),

			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),

		},

		TenantID:     "operator",

		SecretName:   "nephoran-webhook-certs",

		ValidityDays: i.config.OperatorConfig.CertRotationDays,

		Template:     "webhook",

		AutoRenew:    true,

	}



	return i.RequestCertificate(ctx, req)

}



// Helper methods.



func (i *CAIntegration) initializeTLSIntegration(ctx context.Context) error {

	i.logger.Info("initializing TLS integration")



	// Check if TLS manager is available.

	if i.tlsManager == nil {

		i.logger.Warn("TLS manager not available, creating certificates without TLS integration")

		return nil

	}



	// Set up certificate refresh hooks if hot-reload is enabled.

	if i.config.TLSIntegration.HotReloadEnabled {

		// This would integrate with the TLS manager's hot-reload functionality.

		i.logger.Info("TLS hot-reload integration enabled")

	}



	return nil

}



func (i *CAIntegration) initializeRBACIntegration(ctx context.Context) error {

	i.logger.Info("initializing RBAC integration")



	// Set up certificate-based authentication if enabled.

	if i.config.RBACIntegration.CertificateBasedAuth {

		i.logger.Info("certificate-based authentication integration enabled")



		// This would configure client certificate authentication.

		// for Kubernetes API server access.

	}



	return nil

}



func (i *CAIntegration) initializeOperatorIntegration(ctx context.Context) error {

	i.logger.Info("initializing operator integration")



	// Generate webhook certificates if needed.

	if i.config.OperatorConfig.WebhookCertificates {

		i.logger.Info("webhook certificate generation enabled")



		// This would be called during operator startup to ensure.

		// webhook certificates are available.

	}



	return nil

}



func (i *CAIntegration) createCertificateSecret(ctx context.Context, req *IntegrationCertificateRequest, cert *CertificateResponse) error {

	secretName := req.SecretName

	if secretName == "" {

		// Generate secret name from template.

		if i.config.SecretIntegration.SecretNameTemplate != "" {

			secretName = fmt.Sprintf(i.config.SecretIntegration.SecretNameTemplate, req.Name)

		} else {

			secretName = fmt.Sprintf("cert-%s", req.Name)

		}

	}



	secret := &corev1.Secret{

		ObjectMeta: metav1.ObjectMeta{

			Name:        secretName,

			Namespace:   req.Namespace,

			Labels:      i.mergeLabels(req.Labels, i.config.SecretIntegration.SecretLabels),

			Annotations: i.mergeAnnotations(req.Annotations, i.config.SecretIntegration.SecretAnnotations),

		},

		Type: corev1.SecretTypeTLS,

		Data: map[string][]byte{

			"tls.crt": []byte(cert.CertificatePEM),

			"tls.key": []byte(cert.PrivateKeyPEM),

		},

	}



	// Add CA certificate if available.

	if cert.CACertificatePEM != "" {

		secret.Data["ca.crt"] = []byte(cert.CACertificatePEM)

	}



	// Add certificate metadata.

	if secret.Annotations == nil {

		secret.Annotations = make(map[string]string)

	}

	secret.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

	secret.Annotations["nephoran.io/expires-at"] = cert.ExpiresAt.Format(time.RFC3339)

	secret.Annotations["nephoran.io/issued-by"] = cert.IssuedBy

	secret.Annotations["nephoran.io/auto-renew"] = fmt.Sprintf("%v", req.AutoRenew)



	// Create or update secret.

	err := i.client.Create(ctx, secret)

	if err != nil {

		// Try update if create failed.

		existingSecret := &corev1.Secret{}

		if getErr := i.client.Get(ctx, types.NamespacedName{

			Name:      secretName,

			Namespace: req.Namespace,

		}, existingSecret); getErr == nil {

			secret.ResourceVersion = existingSecret.ResourceVersion

			err = i.client.Update(ctx, secret)

		}

	}



	if err != nil {

		return fmt.Errorf("failed to create/update certificate secret: %w", err)

	}



	i.logger.Info("certificate secret created/updated",

		"secret", secretName,

		"namespace", req.Namespace,

		"certificate", cert.SerialNumber)



	return nil

}



func (i *CAIntegration) updateCertificateSecret(ctx context.Context, cert *CertificateResponse) error {

	// Find secrets associated with this certificate.

	secretName := cert.Metadata["secret_name"]

	namespace := cert.Metadata["integration_namespace"]



	if secretName == "" || namespace == "" {

		i.logger.Debug("no secret information in certificate metadata, skipping update")

		return nil

	}



	secret := &corev1.Secret{}

	err := i.client.Get(ctx, types.NamespacedName{

		Name:      secretName,

		Namespace: namespace,

	}, secret)

	if err != nil {

		return fmt.Errorf("failed to get certificate secret: %w", err)

	}



	// Update secret data.

	secret.Data["tls.crt"] = []byte(cert.CertificatePEM)

	secret.Data["tls.key"] = []byte(cert.PrivateKeyPEM)

	if cert.CACertificatePEM != "" {

		secret.Data["ca.crt"] = []byte(cert.CACertificatePEM)

	}



	// Update annotations.

	if secret.Annotations == nil {

		secret.Annotations = make(map[string]string)

	}

	secret.Annotations["nephoran.io/certificate-serial"] = cert.SerialNumber

	secret.Annotations["nephoran.io/expires-at"] = cert.ExpiresAt.Format(time.RFC3339)

	secret.Annotations["nephoran.io/renewed-at"] = time.Now().Format(time.RFC3339)



	err = i.client.Update(ctx, secret)

	if err != nil {

		return fmt.Errorf("failed to update certificate secret: %w", err)

	}



	i.logger.Info("certificate secret updated",

		"secret", secretName,

		"namespace", namespace,

		"certificate", cert.SerialNumber)



	return nil

}



func (i *CAIntegration) updateTLSConfiguration(ctx context.Context, cert *CertificateResponse) error {

	if i.tlsManager == nil {

		return nil

	}



	// This would update the TLS manager configuration with new certificates.

	// Implementation depends on the TLS manager interface.

	i.logger.Info("updating TLS configuration",

		"certificate", cert.SerialNumber)



	return nil

}



func (i *CAIntegration) mergeLabels(reqLabels, configLabels map[string]string) map[string]string {

	merged := make(map[string]string)



	// Add config labels first.

	for k, v := range configLabels {

		merged[k] = v

	}



	// Add request labels (can override config labels).

	for k, v := range reqLabels {

		merged[k] = v

	}



	// Add standard labels.

	merged["app.kubernetes.io/managed-by"] = "nephoran-intent-operator"

	merged["nephoran.io/certificate"] = "true"



	return merged

}



func (i *CAIntegration) mergeAnnotations(reqAnnotations, configAnnotations map[string]string) map[string]string {

	merged := make(map[string]string)



	// Add config annotations first.

	for k, v := range configAnnotations {

		merged[k] = v

	}



	// Add request annotations (can override config annotations).

	for k, v := range reqAnnotations {

		merged[k] = v

	}



	return merged

}



// GetCertificateStatus returns the status of certificates managed by integration.

func (i *CAIntegration) GetCertificateStatus(ctx context.Context) (map[string]interface{}, error) {

	status := make(map[string]interface{})



	// Get certificate statistics from CA manager.

	healthStatus := i.caManager.HealthCheck(ctx)

	status["backend_health"] = healthStatus



	// Get certificate count by namespace.

	namespaces := i.config.TLSIntegration.CertificateNamespaces

	for _, ns := range namespaces {

		secrets := &corev1.SecretList{}

		listOpts := client.ListOptions{

			Namespace: ns,

		}



		if err := i.client.List(ctx, secrets, &listOpts); err != nil {

			continue

		}



		certCount := 0

		for _, secret := range secrets.Items {

			if secret.Type == corev1.SecretTypeTLS {

				if _, ok := secret.Annotations["nephoran.io/certificate-serial"]; ok {

					certCount++

				}

			}

		}



		status[fmt.Sprintf("certificates_%s", ns)] = certCount

	}



	return status, nil

}



package ca



import (

	"context"

	"crypto/rand"

	"crypto/x509"

	"crypto/x509/pkix"

	"fmt"

	"net/url"

	"sync"

	"time"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"



	"sigs.k8s.io/controller-runtime/pkg/client"

)



// CABackendType represents the type of CA backend.

type CABackendType string



const (

	// BackendCertManager holds backendcertmanager value.

	BackendCertManager CABackendType = "cert-manager"

	// BackendVault holds backendvault value.

	BackendVault CABackendType = "vault"

	// BackendExternal holds backendexternal value.

	BackendExternal CABackendType = "external"

	// BackendSelfSigned holds backendselfsigned value.

	BackendSelfSigned CABackendType = "self-signed"

	// BackendHSM holds backendhsm value.

	BackendHSM CABackendType = "hsm"

)



// CAManager manages Certificate Authorities and certificate lifecycle.

type CAManager struct {

	config          *Config

	logger          *logging.StructuredLogger

	client          client.Client

	backends        map[CABackendType]Backend

	certificatePool *CertificatePool

	policyEngine    *PolicyEngine

	distributor     *CertificateDistributor

	monitor         *CAMonitor

	mu              sync.RWMutex

	ctx             context.Context

	cancel          context.CancelFunc

}



// Config holds CA manager configuration.

type Config struct {

	// Global settings.

	DefaultBackend     CABackendType                 `yaml:"default_backend"`

	BackendConfigs     map[CABackendType]interface{} `yaml:"backend_configs"`

	CertificateStore   *CertificateStoreConfig       `yaml:"certificate_store"`

	DistributionConfig *DistributionConfig           `yaml:"distribution_config"`

	PolicyConfig       *PolicyConfig                 `yaml:"policy_config"`

	MonitoringConfig   *MonitoringConfig             `yaml:"monitoring_config"`



	// Lifecycle settings.

	DefaultValidityDuration time.Duration `yaml:"default_validity_duration"`

	RenewalThreshold        time.Duration `yaml:"renewal_threshold"`

	MaxRetryAttempts        int           `yaml:"max_retry_attempts"`

	RetryBackoff            time.Duration `yaml:"retry_backoff"`



	// Security settings.

	KeySize             int           `yaml:"key_size"`

	AllowedKeyTypes     []string      `yaml:"allowed_key_types"`

	MinValidityDuration time.Duration `yaml:"min_validity_duration"`

	MaxValidityDuration time.Duration `yaml:"max_validity_duration"`

	RequireApproval     bool          `yaml:"require_approval"`

	AutoRotationEnabled bool          `yaml:"auto_rotation_enabled"`



	// Multi-tenancy.

	TenantSupport bool                       `yaml:"tenant_support"`

	TenantConfigs map[string]*TenantCAConfig `yaml:"tenant_configs"`

	DefaultTenant string                     `yaml:"default_tenant"`

}



// CertificateStoreConfig configures certificate storage.

type CertificateStoreConfig struct {

	Type            string        `yaml:"type"` // kubernetes, vault, external

	Namespace       string        `yaml:"namespace"`

	SecretPrefix    string        `yaml:"secret_prefix"`

	EncryptionKey   string        `yaml:"encryption_key"`

	BackupEnabled   bool          `yaml:"backup_enabled"`

	BackupInterval  time.Duration `yaml:"backup_interval"`

	RetentionPeriod time.Duration `yaml:"retention_period"`

}



// DistributionConfig configures certificate distribution.

type DistributionConfig struct {

	Enabled            bool                `yaml:"enabled"`

	HotReloadEnabled   bool                `yaml:"hot_reload_enabled"`

	WatchIntervals     time.Duration       `yaml:"watch_intervals"`

	DistributionPaths  map[string]string   `yaml:"distribution_paths"`

	NotificationConfig *NotificationConfig `yaml:"notification_config"`

}



// NotificationConfig configures certificate notifications.

type NotificationConfig struct {

	Enabled     bool         `yaml:"enabled"`

	Webhooks    []string     `yaml:"webhooks"`

	EmailSMTP   *SMTPConfig  `yaml:"email_smtp"`

	SlackConfig *SlackConfig `yaml:"slack_config"`

}



// SMTPConfig configures SMTP notifications.

type SMTPConfig struct {

	Host     string   `yaml:"host"`

	Port     int      `yaml:"port"`

	Username string   `yaml:"username"`

	Password string   `yaml:"password"`

	From     string   `yaml:"from"`

	To       []string `yaml:"to"`

}



// SlackConfig configures Slack notifications.

type SlackConfig struct {

	WebhookURL string `yaml:"webhook_url"`

	Channel    string `yaml:"channel"`

}



// PolicyConfig configures certificate policies.

type PolicyConfig struct {

	Enabled          bool                       `yaml:"enabled"`

	PolicyTemplates  map[string]*PolicyTemplate `yaml:"policy_templates"`

	ValidationRules  []ValidationRule           `yaml:"validation_rules"`

	ApprovalRequired bool                       `yaml:"approval_required"`

	ApprovalWorkflow *ApprovalWorkflow          `yaml:"approval_workflow"`

}



// PolicyTemplate defines certificate policy templates.

type PolicyTemplate struct {

	Name             string            `yaml:"name"`

	KeyUsage         []string          `yaml:"key_usage"`

	ExtKeyUsage      []string          `yaml:"ext_key_usage"`

	ValidityDuration time.Duration     `yaml:"validity_duration"`

	KeySize          int               `yaml:"key_size"`

	AllowedSANTypes  []string          `yaml:"allowed_san_types"`

	RequiredFields   []string          `yaml:"required_fields"`

	CustomExtensions map[string]string `yaml:"custom_extensions"`

}



// ValidationRule defines certificate validation rules.

type ValidationRule struct {

	Name         string `yaml:"name"`

	Type         string `yaml:"type"` // subject, san, key_usage, etc.

	Pattern      string `yaml:"pattern"`

	Required     bool   `yaml:"required"`

	ErrorMessage string `yaml:"error_message"`

}



// ApprovalWorkflow defines certificate approval workflow.

type ApprovalWorkflow struct {

	Stages     []ApprovalStage   `yaml:"stages"`

	Timeout    time.Duration     `yaml:"timeout"`

	Escalation *EscalationConfig `yaml:"escalation"`

}



// ApprovalStage defines an approval stage.

type ApprovalStage struct {

	Name         string        `yaml:"name"`

	Approvers    []string      `yaml:"approvers"`

	MinApprovals int           `yaml:"min_approvals"`

	Timeout      time.Duration `yaml:"timeout"`

}



// EscalationConfig defines escalation settings.

type EscalationConfig struct {

	Enabled    bool          `yaml:"enabled"`

	Escalators []string      `yaml:"escalators"`

	Timeout    time.Duration `yaml:"timeout"`

}



// MonitoringConfig configures CA monitoring.

type MonitoringConfig struct {

	Enabled               bool          `yaml:"enabled"`

	HealthCheckInterval   time.Duration `yaml:"health_check_interval"`

	MetricsEnabled        bool          `yaml:"metrics_enabled"`

	AlertingEnabled       bool          `yaml:"alerting_enabled"`

	ExpirationWarningDays int           `yaml:"expiration_warning_days"`

}



// TenantCAConfig holds tenant-specific CA configuration.

type TenantCAConfig struct {

	TenantID       string        `yaml:"tenant_id"`

	Backend        CABackendType `yaml:"backend"`

	IssuerRef      string        `yaml:"issuer_ref"`

	PolicyTemplate string        `yaml:"policy_template"`

	Namespace      string        `yaml:"namespace"`

	CustomConfig   interface{}   `yaml:"custom_config"`

}



// CertificateRequest represents a certificate request.

type CertificateRequest struct {

	ID               string            `json:"id"`

	TenantID         string            `json:"tenant_id"`

	CommonName       string            `json:"common_name"`

	DNSNames         []string          `json:"dns_names"`

	IPAddresses      []string          `json:"ip_addresses"`

	EmailAddresses   []string          `json:"email_addresses"`

	URIs             []*url.URL        `json:"uris"`

	KeySize          int               `json:"key_size"`

	ValidityDuration time.Duration     `json:"validity_duration"`

	KeyUsage         []string          `json:"key_usage"`

	ExtKeyUsage      []string          `json:"ext_key_usage"`

	CustomExtensions map[string]string `json:"custom_extensions"`

	PolicyTemplate   string            `json:"policy_template"`

	Backend          CABackendType     `json:"backend"`

	AutoRenew        bool              `json:"auto_renew"`

	NotificationList []string          `json:"notification_list"`

	Metadata         map[string]string `json:"metadata"`

	CreatedAt        time.Time         `json:"created_at"`

	UpdatedAt        time.Time         `json:"updated_at"`

}



// CertificateResponse represents the response to a certificate request.

type CertificateResponse struct {

	RequestID        string            `json:"request_id"`

	Certificate      *x509.Certificate `json:"certificate"`

	CertificatePEM   string            `json:"certificate_pem"`

	PrivateKeyPEM    string            `json:"private_key_pem"`

	CACertificatePEM string            `json:"ca_certificate_pem"`

	TrustChainPEM    []string          `json:"trust_chain_pem"`

	SerialNumber     string            `json:"serial_number"`

	Fingerprint      string            `json:"fingerprint"`

	ExpiresAt        time.Time         `json:"expires_at"`

	IssuedBy         string            `json:"issued_by"`

	Status           CertificateStatus `json:"status"`

	Metadata         map[string]string `json:"metadata"`

	CreatedAt        time.Time         `json:"created_at"`

}



// CertificateStatus represents certificate status.

type CertificateStatus string



const (

	// CertStatusPending holds certstatuspending value.

	CertStatusPending CertificateStatus = "pending"

	// StatusIssued holds statusissued value.

	StatusIssued CertificateStatus = "issued"

	// StatusRenewed holds statusrenewed value.

	StatusRenewed CertificateStatus = "renewed"

	// StatusRevoked holds statusrevoked value.

	StatusRevoked CertificateStatus = "revoked"

	// StatusExpired holds statusexpired value.

	StatusExpired CertificateStatus = "expired"

	// CertStatusFailed holds certstatusfailed value.

	CertStatusFailed CertificateStatus = "failed"

)



// Backend defines the interface for CA backends.

type Backend interface {

	// Initialization.

	Initialize(ctx context.Context, config interface{}) error

	HealthCheck(ctx context.Context) error



	// Certificate operations.

	IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error)

	RevokeCertificate(ctx context.Context, serialNumber string, reason int) error

	RenewCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error)



	// CA operations.

	GetCAChain(ctx context.Context) ([]*x509.Certificate, error)

	GetCRL(ctx context.Context) (*pkix.CertificateList, error)



	// Metadata.

	GetBackendInfo(ctx context.Context) (*BackendInfo, error)

	GetSupportedFeatures() []string

}



// BackendInfo contains backend metadata.

type BackendInfo struct {

	Type       CABackendType          `json:"type"`

	Version    string                 `json:"version"`

	Status     string                 `json:"status"`

	Issuer     string                 `json:"issuer"`

	ValidUntil time.Time              `json:"valid_until"`

	Features   []string               `json:"features"`

	Metrics    map[string]interface{} `json:"metrics"`

}



// NewCAManager creates a new CA manager.

func NewCAManager(config *Config, logger *logging.StructuredLogger, client client.Client) (*CAManager, error) {

	if config == nil {

		return nil, fmt.Errorf("CA manager config is required")

	}



	ctx, cancel := context.WithCancel(context.Background())



	manager := &CAManager{

		config:   config,

		logger:   logger,

		client:   client,

		backends: make(map[CABackendType]Backend),

		ctx:      ctx,

		cancel:   cancel,

	}



	// Initialize certificate pool.

	pool, err := NewCertificatePool(config.CertificateStore, logger)

	if err != nil {

		cancel()

		return nil, fmt.Errorf("failed to initialize certificate pool: %w", err)

	}

	manager.certificatePool = pool



	// Initialize policy engine.

	if config.PolicyConfig != nil && config.PolicyConfig.Enabled {

		// Convert manager policy config to policy engine config.

		engineConfig := &PolicyEngineConfig{

			Enabled:                config.PolicyConfig.Enabled,

			Rules:                  convertValidationRulesToPolicyRules(config.PolicyConfig.ValidationRules),

			CertificatePinning:     false,                               // Default, can be configured separately

			AlgorithmStrengthCheck: true,                                // Default for security

			MinimumRSAKeySize:      2048,                                // Default

			AllowedECCurves:        []string{"P-256", "P-384", "P-521"}, // Default

		}

		policyEngine, err := NewPolicyEngine(engineConfig, logger)

		if err != nil {

			cancel()

			return nil, fmt.Errorf("failed to initialize policy engine: %w", err)

		}

		manager.policyEngine = policyEngine

	}



	// Initialize certificate distributor.

	if config.DistributionConfig != nil && config.DistributionConfig.Enabled {

		distributor, err := NewCertificateDistributor(config.DistributionConfig, logger, client)

		if err != nil {

			cancel()

			return nil, fmt.Errorf("failed to initialize certificate distributor: %w", err)

		}

		manager.distributor = distributor

	}



	// Initialize CA monitor.

	if config.MonitoringConfig != nil && config.MonitoringConfig.Enabled {

		monitor, err := NewCAMonitor(config.MonitoringConfig, logger)

		if err != nil {

			cancel()

			return nil, fmt.Errorf("failed to initialize CA monitor: %w", err)

		}

		manager.monitor = monitor

	}



	// Initialize backends.

	if err := manager.initializeBackends(); err != nil {

		cancel()

		return nil, fmt.Errorf("failed to initialize backends: %w", err)

	}



	// Start background processes.

	go manager.runCertificateLifecycleManager()

	if manager.monitor != nil {

		go manager.monitor.Start(ctx)

	}

	if manager.distributor != nil {

		go manager.distributor.Start(ctx)

	}



	return manager, nil

}



// initializeBackends initializes all configured backends.

func (m *CAManager) initializeBackends() error {

	for backendType, backendConfig := range m.config.BackendConfigs {

		var backend Backend

		var err error



		switch backendType {

		case BackendCertManager:

			backend, err = NewCertManagerBackend(m.logger, m.client)

		case BackendVault:

			backend, err = NewVaultBackend(m.logger)

		case BackendExternal:

			backend, err = NewExternalBackend(m.logger)

		case BackendSelfSigned:

			backend, err = NewSelfSignedBackend(m.logger)

		case BackendHSM:

			// Need HSM config - using default config for now.

			hsmConfig := &HSMBackendConfig{

				ProviderType: "mock", // Default to mock provider for testing

			}

			backend = NewHSMBackend(hsmConfig, m.logger)

			err = nil

		default:

			return fmt.Errorf("unsupported backend type: %s", backendType)

		}



		if err != nil {

			return fmt.Errorf("failed to create %s backend: %w", backendType, err)

		}



		if err := backend.Initialize(m.ctx, backendConfig); err != nil {

			return fmt.Errorf("failed to initialize %s backend: %w", backendType, err)

		}



		m.backends[backendType] = backend

		m.logger.Info("initialized CA backend",

			"type", backendType,

			"features", backend.GetSupportedFeatures())

	}



	return nil

}



// IssueCertificate issues a new certificate.

func (m *CAManager) IssueCertificate(ctx context.Context, req *CertificateRequest) (*CertificateResponse, error) {

	m.logger.Info("issuing certificate",

		"request_id", req.ID,

		"tenant_id", req.TenantID,

		"common_name", req.CommonName)



	// Validate request.

	if err := m.validateCertificateRequest(req); err != nil {

		return nil, fmt.Errorf("certificate request validation failed: %w", err)

	}



	// Apply policy validation if enabled (after certificate is issued).

	// Note: PolicyEngine validates certificates, not requests.



	// Select backend.

	backend, err := m.selectBackend(req)

	if err != nil {

		return nil, fmt.Errorf("backend selection failed: %w", err)

	}



	// Issue certificate.

	response, err := backend.IssueCertificate(ctx, req)

	if err != nil {

		m.logger.Error("certificate issuance failed",

			"request_id", req.ID,

			"backend", req.Backend,

			"error", err)

		return nil, fmt.Errorf("certificate issuance failed: %w", err)

	}



	// Store certificate.

	if err := m.certificatePool.StoreCertificate(response); err != nil {

		m.logger.Warn("failed to store certificate",

			"request_id", req.ID,

			"serial_number", response.SerialNumber,

			"error", err)

	}



	// Distribute certificate if enabled.

	if m.distributor != nil {

		if err := m.distributor.DistributeCertificate(response); err != nil {

			m.logger.Warn("certificate distribution failed",

				"request_id", req.ID,

				"error", err)

		}

	}



	m.logger.Info("certificate issued successfully",

		"request_id", req.ID,

		"serial_number", response.SerialNumber,

		"expires_at", response.ExpiresAt)



	return response, nil

}



// RevokeCertificate revokes an existing certificate.

func (m *CAManager) RevokeCertificate(ctx context.Context, serialNumber string, reason int, tenantID string) error {

	m.logger.Info("revoking certificate",

		"serial_number", serialNumber,

		"reason", reason,

		"tenant_id", tenantID)



	// Find certificate in pool.

	cert, err := m.certificatePool.GetCertificate(serialNumber)

	if err != nil {

		return fmt.Errorf("certificate not found: %w", err)

	}



	// Determine backend.

	var backend Backend

	if cert.IssuedBy != "" {

		if b, ok := m.backends[CABackendType(cert.IssuedBy)]; ok {

			backend = b

		}

	}

	if backend == nil {

		backend = m.backends[m.config.DefaultBackend]

	}



	// Revoke certificate.

	if err := backend.RevokeCertificate(ctx, serialNumber, reason); err != nil {

		return fmt.Errorf("certificate revocation failed: %w", err)

	}



	// Update certificate status.

	cert.Status = StatusRevoked

	if err := m.certificatePool.UpdateCertificate(cert); err != nil {

		m.logger.Warn("failed to update certificate status",

			"serial_number", serialNumber,

			"error", err)

	}



	m.logger.Info("certificate revoked successfully",

		"serial_number", serialNumber)



	return nil

}



// RenewCertificate renews an existing certificate.

func (m *CAManager) RenewCertificate(ctx context.Context, serialNumber string) (*CertificateResponse, error) {

	m.logger.Info("renewing certificate",

		"serial_number", serialNumber)



	// Get existing certificate.

	existingCert, err := m.certificatePool.GetCertificate(serialNumber)

	if err != nil {

		return nil, fmt.Errorf("existing certificate not found: %w", err)

	}



	// Create renewal request based on existing certificate.

	req := &CertificateRequest{

		ID:               generateManagerRequestID(),

		TenantID:         existingCert.Metadata["tenant_id"],

		CommonName:       existingCert.Certificate.Subject.CommonName,

		DNSNames:         existingCert.Certificate.DNSNames,

		ValidityDuration: m.config.DefaultValidityDuration,

		AutoRenew:        true,

	}



	// Extract IP addresses and other details from existing certificate.

	for _, ip := range existingCert.Certificate.IPAddresses {

		req.IPAddresses = append(req.IPAddresses, ip.String())

	}

	req.EmailAddresses = existingCert.Certificate.EmailAddresses

	req.URIs = existingCert.Certificate.URIs



	// Select backend.

	backend, err := m.selectBackend(req)

	if err != nil {

		return nil, fmt.Errorf("backend selection failed: %w", err)

	}



	// Renew certificate.

	response, err := backend.RenewCertificate(ctx, req)

	if err != nil {

		return nil, fmt.Errorf("certificate renewal failed: %w", err)

	}



	// Update status.

	response.Status = StatusRenewed



	// Store renewed certificate.

	if err := m.certificatePool.StoreCertificate(response); err != nil {

		m.logger.Warn("failed to store renewed certificate",

			"serial_number", response.SerialNumber,

			"error", err)

	}



	// Distribute certificate if enabled.

	if m.distributor != nil {

		if err := m.distributor.DistributeCertificate(response); err != nil {

			m.logger.Warn("certificate distribution failed",

				"serial_number", response.SerialNumber,

				"error", err)

		}

	}



	m.logger.Info("certificate renewed successfully",

		"old_serial_number", serialNumber,

		"new_serial_number", response.SerialNumber,

		"expires_at", response.ExpiresAt)



	return response, nil

}



// GetCertificate retrieves a certificate by serial number.

func (m *CAManager) GetCertificate(serialNumber string) (*CertificateResponse, error) {

	return m.certificatePool.GetCertificate(serialNumber)

}



// ListCertificates lists certificates with optional filters.

func (m *CAManager) ListCertificates(filters map[string]string) ([]*CertificateResponse, error) {

	return m.certificatePool.ListCertificates(filters)

}



// GetCAChain retrieves the CA certificate chain for a backend.

func (m *CAManager) GetCAChain(ctx context.Context, backendType CABackendType) ([]*x509.Certificate, error) {

	backend, ok := m.backends[backendType]

	if !ok {

		return nil, fmt.Errorf("backend %s not available", backendType)

	}



	return backend.GetCAChain(ctx)

}



// HealthCheck performs health check on all backends.

func (m *CAManager) HealthCheck(ctx context.Context) map[CABackendType]error {

	results := make(map[CABackendType]error)



	for backendType, backend := range m.backends {

		if err := backend.HealthCheck(ctx); err != nil {

			results[backendType] = err

		} else {

			results[backendType] = nil

		}

	}



	return results

}



// Close gracefully shuts down the CA manager.

func (m *CAManager) Close() error {

	m.logger.Info("shutting down CA manager")



	m.cancel()



	// Close distributor.

	if m.distributor != nil {

		m.distributor.Stop()

	}



	// Close certificate pool.

	if m.certificatePool != nil {

		m.certificatePool.Close()

	}



	return nil

}



// Helper methods.



func (m *CAManager) validateCertificateRequest(req *CertificateRequest) error {

	if req.CommonName == "" {

		return fmt.Errorf("common name is required")

	}



	if req.ValidityDuration < m.config.MinValidityDuration {

		return fmt.Errorf("validity duration too short: minimum %v", m.config.MinValidityDuration)

	}



	if req.ValidityDuration > m.config.MaxValidityDuration {

		return fmt.Errorf("validity duration too long: maximum %v", m.config.MaxValidityDuration)

	}



	if req.KeySize < 2048 {

		return fmt.Errorf("key size too small: minimum 2048 bits")

	}



	return nil

}



func (m *CAManager) selectBackend(req *CertificateRequest) (Backend, error) {

	// Use request-specific backend if specified.

	if req.Backend != "" {

		if backend, ok := m.backends[req.Backend]; ok {

			return backend, nil

		}

		return nil, fmt.Errorf("requested backend %s not available", req.Backend)

	}



	// Use tenant-specific backend if configured.

	if m.config.TenantSupport && req.TenantID != "" {

		if tenantConfig, ok := m.config.TenantConfigs[req.TenantID]; ok {

			if backend, ok := m.backends[tenantConfig.Backend]; ok {

				return backend, nil

			}

		}

	}



	// Use default backend.

	if backend, ok := m.backends[m.config.DefaultBackend]; ok {

		return backend, nil

	}



	return nil, fmt.Errorf("no available backends")

}



func (m *CAManager) runCertificateLifecycleManager() {

	ticker := time.NewTicker(time.Hour) // Check every hour

	defer ticker.Stop()



	for {

		select {

		case <-m.ctx.Done():

			return

		case <-ticker.C:

			m.processExpiringCertificates()

		}

	}

}



func (m *CAManager) processExpiringCertificates() {

	threshold := time.Now().Add(m.config.RenewalThreshold)

	expiringCerts, err := m.certificatePool.GetExpiringCertificates(threshold)

	if err != nil {

		m.logger.Error("failed to get expiring certificates", "error", err)

		return

	}



	for _, cert := range expiringCerts {

		if cert.Metadata["auto_renew"] == "true" {

			go func(serialNumber string) {

				_, err := m.RenewCertificate(m.ctx, serialNumber)

				if err != nil {

					m.logger.Error("automatic certificate renewal failed",

						"serial_number", serialNumber,

						"error", err)

				}

			}(cert.SerialNumber)

		}

	}

}



// generateManagerRequestID generates a unique request ID for manager operations.

func generateManagerRequestID() string {

	// Generate a random request ID.

	randomBytes := make([]byte, 8)

	if _, err := rand.Read(randomBytes); err != nil {

		// Fallback to time-based ID if random generation fails.

		return fmt.Sprintf("req-%x", time.Now().UnixNano())

	}

	return fmt.Sprintf("req-%x", randomBytes)

}



// convertValidationRulesToPolicyRules converts manager validation rules to policy engine rules.

func convertValidationRulesToPolicyRules(validationRules []ValidationRule) []PolicyRule {

	policyRules := make([]PolicyRule, len(validationRules))

	for i, rule := range validationRules {

		policyRules[i] = PolicyRule{

			Name:        rule.Name,

			Type:        rule.Type,

			Pattern:     rule.Pattern,

			Required:    rule.Required,

			Severity:    RuleSeverityError, // Default severity

			Description: rule.ErrorMessage,

		}

	}

	return policyRules

}


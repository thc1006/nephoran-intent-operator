package mtls

import (
	"context"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// IntegrationManager manages the integration of mTLS components with 2025 security best practices.

type IntegrationManager struct {
	config *IntegrationConfig

	logger *logging.StructuredLogger

	// Component managers.

	identityManager *IdentityManager

	monitor *MTLSMonitor

	// Lifecycle management.

	ctx context.Context

	cancel context.CancelFunc
}

// IntegrationConfig holds configuration for the integration manager.

type IntegrationConfig struct {
	// mTLS configuration.

	MTLSConfig *MTLSIntegrationConfig `json:"mtls_config"`

	// Monitoring configuration.

	MonitoringConfig *MonitoringIntegrationConfig `json:"monitoring_config"`

	// Security configuration.

	SecurityConfig *SecurityIntegrationConfig `json:"security_config"`
}

// MTLSIntegrationConfig holds mTLS-specific configuration.

type MTLSIntegrationConfig struct {
	Enabled bool `json:"enabled"`

	CertificateBaseDir string `json:"certificate_base_dir"`

	TenantID string `json:"tenant_id"`

	ValidityDuration time.Duration `json:"validity_duration"`

	RenewalThreshold time.Duration `json:"renewal_threshold"`

	RotationInterval time.Duration `json:"rotation_interval"`

	// FIPS compliance for 2025 security standards.

	FIPSMode bool `json:"fips_mode"`

	KeySize int `json:"key_size"`
}

// MonitoringIntegrationConfig holds monitoring-specific configuration.

type MonitoringIntegrationConfig struct {
	Enabled bool `json:"enabled"`

	MetricsInterval time.Duration `json:"metrics_interval"`

	AlertsEnabled bool `json:"alerts_enabled"`
}

// SecurityIntegrationConfig holds security-specific configuration.

type SecurityIntegrationConfig struct {
	AuditLogging bool `json:"audit_logging"`

	ThreatDetection bool `json:"threat_detection"`

	ComplianceReporting bool `json:"compliance_reporting"`
}

// NewIntegrationManager creates a new integration manager with enhanced 2025 security.

func NewIntegrationManager(config *IntegrationConfig, logger *logging.StructuredLogger) (*IntegrationManager, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("mtls-integration", "1.0.0", "production"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &IntegrationManager{
		config: config,

		logger: logger,

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize components.

	if err := manager.initializeIdentityManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize identity manager: %w", err)
	}

	if err := manager.initializeMonitoring(); err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	return manager, nil
}

// initializeIdentityManager initializes the identity manager.

func (m *IntegrationManager) initializeIdentityManager() error {
	if !m.config.MTLSConfig.Enabled {

		m.logger.Info("mTLS not enabled, skipping identity manager initialization")

		return nil

	}

	// Enforce 2025 security standards
	if m.config.MTLSConfig.KeySize == 0 {
		m.config.MTLSConfig.KeySize = 3072 // Minimum 3072-bit keys for 2025
	}

	if m.config.MTLSConfig.ValidityDuration == 0 {
		m.config.MTLSConfig.ValidityDuration = 90 * 24 * time.Hour // Maximum 90-day certificates for 2025
	}

	identityConfig := IdentityManagerConfig{
		CertificateDir: m.config.MTLSConfig.CertificateBaseDir,

		CertificateValidityPeriod: m.config.MTLSConfig.ValidityDuration,

		RotationThreshold: m.config.MTLSConfig.RenewalThreshold,

		AutoRotation: true,

		KeySize: m.config.MTLSConfig.KeySize,

		FIPSMode: m.config.MTLSConfig.FIPSMode,
	}

	var err error

	m.identityManager, err = NewIdentityManager(identityConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create identity manager: %w", err)
	}

	// Create service identities.

	if err := m.createServiceIdentities(); err != nil {
		return fmt.Errorf("failed to create service identities: %w", err)
	}

	return nil
}

// initializeMonitoring initializes the monitoring components.

func (m *IntegrationManager) initializeMonitoring() error {
	if !m.config.MonitoringConfig.Enabled {

		m.logger.Info("Monitoring not enabled, skipping monitoring initialization")

		return nil

	}

	m.monitor = NewMTLSMonitor(m.logger)

	// Add metric collectors.

	m.monitor.AddMetricCollector(&ConnectionMetricCollector{})

	m.monitor.AddMetricCollector(&CertificateMetricCollector{})

	m.monitor.AddMetricCollector(&SecurityMetricCollector{})

	m.monitor.AddMetricCollector(&PerformanceMetricCollector{})

	// Add alert rules.

	m.addDefaultAlertRules()

	// Start monitoring.

	go m.monitor.Start()

	return nil
}

// createServiceIdentities creates default service identities.

func (m *IntegrationManager) createServiceIdentities() error {
	if m.identityManager == nil {
		return nil
	}

	tenantID := m.config.MTLSConfig.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	// Create identities for common services with 2025 security roles.

	services := []struct {
		name string
		role ServiceRole
	}{
		{"controller", RoleController},
		{"llm-service", RoleLLMService},
		{"rag-service", RoleRAGService},
		{"git-client", RoleGitClient},
		{"database-client", RoleDatabaseClient},
		{"nephio-bridge", RoleNephioBridge},
		{"oran-adaptor", RoleORANAdaptor},
		{"monitoring", RoleMonitoring},
		{"api-server", ServiceRoleServer},
		{"webhook", ServiceRoleServer},
	}

	for _, service := range services {
		_, err := m.identityManager.CreateServiceIdentity(service.name, tenantID, service.role)
		if err != nil {
			m.logger.Error("Failed to create service identity",

				"service", service.name,

				"role", service.role,

				"error", err,
			)

			continue
		}

		m.logger.Info("Created service identity",

			"service", service.name,

			"role", service.role,

			"tenant", tenantID,
		)
	}

	return nil
}

// addDefaultAlertRules adds default alert rules for monitoring.

func (m *IntegrationManager) addDefaultAlertRules() {
	// Certificate expiry alert.

	m.monitor.AddAlertRule(AlertRule{
		Name: "certificate-expiring",

		Severity: AlertSeverityWarning,

		Condition: AlertCondition{
			Type: AlertTypeCertificateExpiring,

			Threshold: 7, // 7 days
		},

		Enabled: true,

		Description: "Alert when certificates are expiring within 7 days",
	})

	// Certificate expired alert.

	m.monitor.AddAlertRule(AlertRule{
		Name: "certificate-expired",

		Severity: AlertSeverityCritical,

		Condition: AlertCondition{
			Type: AlertTypeCertificateExpired,
		},

		Enabled: true,

		Description: "Alert when certificates have expired",
	})

	// High error rate alert.

	m.monitor.AddAlertRule(AlertRule{
		Name: "high-error-rate",

		Severity: AlertSeverityError,

		Condition: AlertCondition{
			Type: AlertTypeHighErrorRate,

			Threshold: 0.05, // 5% error rate
		},

		Enabled: true,

		Description: "Alert when error rate exceeds 5%",
	})

	// Old TLS version alert (2025 security requirement).

	m.monitor.AddAlertRule(AlertRule{
		Name: "old-tls-version",

		Severity: AlertSeverityCritical,

		Condition: AlertCondition{
			Type: AlertTypeOldTLSVersion,
		},

		Enabled: true,

		Description: "Alert when connections use TLS versions older than 1.3",
	})

	// Weak cipher alert (2025 security requirement).

	m.monitor.AddAlertRule(AlertRule{
		Name: "weak-cipher",

		Severity: AlertSeverityCritical,

		Condition: AlertCondition{
			Type: AlertTypeWeakCipher,
		},

		Enabled: true,

		Description: "Alert when connections use weak cipher suites",
	})
}

// Start starts the integration manager.

func (m *IntegrationManager) Start() error {
	m.logger.Info("Starting mTLS integration manager with 2025 security standards")

	// Start monitoring if enabled.

	if m.monitor != nil && m.config.MonitoringConfig.Enabled {
		go m.monitor.Start()
	}

	// Start certificate rotation monitoring.

	if m.identityManager != nil {
		// Identity manager handles its own background rotation
		m.logger.Info("Identity manager started with automatic rotation enabled")
	}

	return nil
}

// Stop stops the integration manager.

func (m *IntegrationManager) Stop() error {
	m.logger.Info("Stopping mTLS integration manager")

	m.cancel()

	// Stop monitoring.

	if m.monitor != nil {
		m.monitor.Stop()
	}

	// Close identity manager.

	if m.identityManager != nil {
		if err := m.identityManager.Close(); err != nil {
			m.logger.Error("Failed to close identity manager", "error", err)
		}
	}

	return nil
}

// GetServiceIdentity retrieves a service identity.

func (m *IntegrationManager) GetServiceIdentity(serviceName, tenantID string, role ServiceRole) (*ServiceIdentity, error) {
	if m.identityManager == nil {
		return nil, fmt.Errorf("identity manager not initialized")
	}

	return m.identityManager.GetServiceIdentity(serviceName, tenantID, role)
}

// RotateServiceCertificate rotates a service certificate.

func (m *IntegrationManager) RotateServiceCertificate(serviceName, tenantID string, role ServiceRole) error {
	if m.identityManager == nil {
		return fmt.Errorf("identity manager not initialized")
	}

	return m.identityManager.RotateServiceIdentity(serviceName, tenantID, role)
}

// GetMetrics retrieves current metrics.

func (m *IntegrationManager) GetMetrics() ([]*Metric, error) {
	if m.monitor == nil {
		return nil, fmt.Errorf("monitor not initialized")
	}

	return m.monitor.GetMetrics()
}

// GetConnectionStats returns connection statistics.

func (m *IntegrationManager) GetConnectionStats() *ConnectionStats {
	if m.monitor == nil {
		return nil
	}

	return m.monitor.GetConnectionStats()
}

// GetCertificateStats returns certificate statistics.

func (m *IntegrationManager) GetCertificateStats() *CertificateStats {
	if m.monitor == nil {
		return nil
	}

	return m.monitor.GetCertificateStats()
}

// GetIdentityStats returns identity statistics.

func (m *IntegrationManager) GetIdentityStats() *IdentityStats {
	if m.identityManager == nil {
		return nil
	}

	return m.identityManager.GetIdentityStats()
}

// TrackConnection tracks an mTLS connection.

func (m *IntegrationManager) TrackConnection(connID, serviceName, remoteAddr, localAddr string, tlsInfo *TLSConnectionInfo) error {
	if m.monitor == nil {
		return fmt.Errorf("monitor not initialized")
	}

	m.monitor.TrackConnection(connID, serviceName, remoteAddr, localAddr, tlsInfo)

	return nil
}

// UpdateConnectionActivity updates connection activity.

func (m *IntegrationManager) UpdateConnectionActivity(connID string) error {
	if m.monitor == nil {
		return fmt.Errorf("monitor not initialized")
	}

	m.monitor.UpdateConnectionActivity(connID)

	return nil
}

// RecordConnectionError records a connection error.

func (m *IntegrationManager) RecordConnectionError(connID string) error {
	if m.monitor == nil {
		return fmt.Errorf("monitor not initialized")
	}

	m.monitor.RecordConnectionError(connID)

	return nil
}

// GetHealthStatus returns the health status of the integration.

func (m *IntegrationManager) GetHealthStatus() map[string]interface{} {
	status := map[string]interface{}{
		"status": "healthy",

		"timestamp": time.Now(),

		"components": make(map[string]interface{}),
	}

	// Check identity manager status.

	if m.identityManager != nil {
		identityStats := m.identityManager.GetIdentityStats()

		status["components"].(map[string]interface{})["identity_manager"] = map[string]interface{}{
			"status": "healthy",

			"total_identities": identityStats.TotalIdentities,

			"expiring_soon": identityStats.ExpiringSoon,

			"fips_mode": identityStats.FIPSMode,

			"auto_rotation": identityStats.AutoRotation,
		}
	} else {
		status["components"].(map[string]interface{})["identity_manager"] = map[string]interface{}{
			"status": "disabled",
		}
	}

	// Check monitor status.

	if m.monitor != nil {
		connectionStats := m.monitor.GetConnectionStats()

		certificateStats := m.monitor.GetCertificateStats()

		status["components"].(map[string]interface{})["monitor"] = map[string]interface{}{
			"status": "healthy",

			"total_connections": connectionStats.TotalConnections,

			"total_certificates": certificateStats.TotalCertificates,

			"error_rate": connectionStats.ErrorRate,

			"expired_certificates": certificateStats.ExpiredCount,
		}
	} else {
		status["components"].(map[string]interface{})["monitor"] = map[string]interface{}{
			"status": "disabled",
		}
	}

	return status
}

// Close gracefully shuts down the integration manager.

func (m *IntegrationManager) Close() error {
	return m.Stop()
}

// ValidateConfiguration validates the integration configuration.

func ValidateConfiguration(config *IntegrationConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	if config.MTLSConfig == nil {
		return fmt.Errorf("mTLS configuration is required")
	}

	if config.MonitoringConfig == nil {
		return fmt.Errorf("monitoring configuration is required")
	}

	if config.SecurityConfig == nil {
		return fmt.Errorf("security configuration is required")
	}

	// Validate mTLS configuration.

	if config.MTLSConfig.Enabled {
		if config.MTLSConfig.CertificateBaseDir == "" {
			return fmt.Errorf("certificate base directory is required when mTLS is enabled")
		}

		// Enforce 2025 security standards
		if config.MTLSConfig.KeySize != 0 && config.MTLSConfig.KeySize < 3072 {
			return fmt.Errorf("key size must be at least 3072 bits for 2025 security standards")
		}

		if config.MTLSConfig.ValidityDuration > 90*24*time.Hour {
			return fmt.Errorf("certificate validity period exceeds 90-day maximum for 2025 security standards")
		}
	}

	return nil
}

// DefaultIntegrationConfig returns a default integration configuration with 2025 security standards.

func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		MTLSConfig: &MTLSIntegrationConfig{
			Enabled: true,

			CertificateBaseDir: "/etc/nephoran/certs",

			TenantID: "default",

			ValidityDuration: 90 * 24 * time.Hour, // 90 days maximum for 2025

			RenewalThreshold: 7 * 24 * time.Hour, // Renew 7 days before expiry

			RotationInterval: 1 * time.Hour, // Check every hour

			FIPSMode: true, // Enable FIPS mode by default for 2025

			KeySize: 3072, // 3072-bit keys minimum for 2025
		},

		MonitoringConfig: &MonitoringIntegrationConfig{
			Enabled: true,

			MetricsInterval: 30 * time.Second,

			AlertsEnabled: true,
		},

		SecurityConfig: &SecurityIntegrationConfig{
			AuditLogging: true,

			ThreatDetection: true,

			ComplianceReporting: true,
		},
	}
}
package mtls

import (
	"context"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
)

// IntegrationManager provides a comprehensive mTLS integration example
type IntegrationManager struct {
	config          *config.Config
	logger          *logging.StructuredLogger
	caManager       *ca.CAManager
	identityManager *IdentityManager
	monitor         *MTLSMonitor
	
	// Service clients and servers
	llmClient    *Client
	llmServer    *Server
	ragClient    *Client
	ragServer    *Server
	
	ctx    context.Context
	cancel context.CancelFunc
}

// IntegrationConfig holds configuration for the integration manager
type IntegrationConfig struct {
	Config    *config.Config
	Logger    *logging.StructuredLogger
	CAManager *ca.CAManager
}

// NewIntegrationManager creates a comprehensive mTLS integration manager
func NewIntegrationManager(config *IntegrationConfig) (*IntegrationManager, error) {
	if config.Logger == nil {
		config.Logger = logging.NewStructuredLogger()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &IntegrationManager{
		config: config.Config,
		logger: config.Logger,
		caManager: config.CAManager,
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Initialize components
	if err := manager.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize mTLS components: %w", err)
	}
	
	config.Logger.Info("mTLS integration manager initialized successfully")
	
	return manager, nil
}

// initializeComponents initializes all mTLS components
func (m *IntegrationManager) initializeComponents() error {
	// Initialize Identity Manager
	if err := m.initializeIdentityManager(); err != nil {
		return fmt.Errorf("failed to initialize identity manager: %w", err)
	}
	
	// Initialize Monitor
	if err := m.initializeMonitor(); err != nil {
		return fmt.Errorf("failed to initialize monitor: %w", err)
	}
	
	// Initialize Service Clients and Servers
	if err := m.initializeServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}
	
	return nil
}

// initializeIdentityManager initializes the identity manager
func (m *IntegrationManager) initializeIdentityManager() error {
	if !m.config.MTLSConfig.Enabled {
		m.logger.Info("mTLS not enabled, skipping identity manager initialization")
		return nil
	}
	
	identityConfig := &IdentityManagerConfig{
		CAManager:               m.caManager,
		BaseDir:                 m.config.MTLSConfig.CertificateBaseDir,
		DefaultTenantID:         m.config.MTLSConfig.TenantID,
		DefaultPolicyTemplate:   m.config.MTLSConfig.PolicyTemplate,
		DefaultValidityDuration: m.config.MTLSConfig.ValidityDuration,
		RenewalThreshold:        m.config.MTLSConfig.RenewalThreshold,
		RotationInterval:        m.config.MTLSConfig.RotationInterval,
		CleanupInterval:         1 * time.Hour,
		MaxIdentities:           1000,
		BackupEnabled:           true,
	}
	
	var err error
	m.identityManager, err = NewIdentityManager(identityConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create identity manager: %w", err)
	}
	
	// Create service identities
	if err := m.createServiceIdentities(); err != nil {
		return fmt.Errorf("failed to create service identities: %w", err)
	}
	
	return nil
}

// initializeMonitor initializes the mTLS monitor
func (m *IntegrationManager) initializeMonitor() error {
	m.monitor = NewMTLSMonitor(m.logger)
	
	// Track certificates if identity manager is available
	if m.identityManager != nil {
		identities := m.identityManager.ListServiceIdentities()
		for _, identity := range identities {
			if identity.Certificate != nil {
				m.monitor.TrackCertificate(
					identity.ServiceName,
					identity.Role,
					identity.CertPath,
					identity.Certificate,
				)
			}
		}
	}
	
	return nil
}

// initializeServices initializes mTLS-enabled services
func (m *IntegrationManager) initializeServices() error {
	if !m.config.MTLSConfig.Enabled {
		m.logger.Info("mTLS not enabled, skipping service initialization")
		return nil
	}
	
	// Initialize LLM service client and server
	if err := m.initializeLLMService(); err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}
	
	// Initialize RAG service client and server
	if err := m.initializeRAGService(); err != nil {
		return fmt.Errorf("failed to initialize RAG service: %w", err)
	}
	
	return nil
}

// initializeLLMService initializes the LLM service with mTLS
func (m *IntegrationManager) initializeLLMService() error {
	llmConfig := m.config.MTLSConfig.LLMProcessor
	if llmConfig == nil || !llmConfig.Enabled {
		return nil
	}
	
	// Create LLM client
	clientConfig := &ClientConfig{
		ServiceName:          llmConfig.ServiceName,
		TenantID:            m.config.MTLSConfig.TenantID,
		ClientCertPath:       llmConfig.ClientCertPath,
		ClientKeyPath:        llmConfig.ClientKeyPath,
		CACertPath:          llmConfig.CACertPath,
		ServerName:          llmConfig.ServerName,
		InsecureSkipVerify:  llmConfig.InsecureSkipVerify,
		DialTimeout:         llmConfig.DialTimeout,
		MaxIdleConns:        llmConfig.MaxIdleConns,
		MaxConnsPerHost:     llmConfig.MaxConnsPerHost,
		IdleConnTimeout:     llmConfig.IdleConnTimeout,
		CertValidityDuration: m.config.MTLSConfig.ValidityDuration,
		RotationEnabled:     m.config.MTLSConfig.RotationEnabled,
		RotationInterval:    m.config.MTLSConfig.RotationInterval,
		RenewalThreshold:    m.config.MTLSConfig.RenewalThreshold,
		CAManager:           m.caManager,
		AutoProvision:       m.config.MTLSConfig.AutoProvision,
		PolicyTemplate:      m.config.MTLSConfig.PolicyTemplate,
	}
	
	var err error
	m.llmClient, err = NewClient(clientConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	
	// Create LLM server
	serverConfig := &ServerConfig{
		ServiceName:          llmConfig.ServiceName,
		TenantID:            m.config.MTLSConfig.TenantID,
		Address:             "0.0.0.0",
		Port:                llmConfig.Port,
		ServerCertPath:      llmConfig.ServerCertPath,
		ServerKeyPath:       llmConfig.ServerKeyPath,
		CACertPath:         llmConfig.CACertPath,
		ClientCACertPath:   llmConfig.CACertPath,
		CertValidityDuration: m.config.MTLSConfig.ValidityDuration,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
		ClientCertValidation: ClientCertRequiredVerify,
		RotationEnabled:    m.config.MTLSConfig.RotationEnabled,
		RotationInterval:   m.config.MTLSConfig.RotationInterval,
		RenewalThreshold:   m.config.MTLSConfig.RenewalThreshold,
		CAManager:          m.caManager,
		AutoProvision:      m.config.MTLSConfig.AutoProvision,
		PolicyTemplate:     m.config.MTLSConfig.PolicyTemplate,
		EnableHSTS:         m.config.MTLSConfig.EnableHSTS,
		HSTSMaxAge:         m.config.MTLSConfig.HSTSMaxAge,
		AllowedClientCNs:   llmConfig.AllowedClientCNs,
		AllowedClientOrgs:  llmConfig.AllowedClientOrgs,
		ClientCertRequired: llmConfig.RequireClientCert,
	}
	
	m.llmServer, err = NewServer(serverConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create LLM server: %w", err)
	}
	
	// Start server in background
	go func() {
		if err := m.llmServer.Start(m.ctx); err != nil {
			m.logger.Error("LLM server failed", "error", err)
		}
	}()
	
	m.logger.Info("LLM service initialized with mTLS")
	
	return nil
}

// initializeRAGService initializes the RAG service with mTLS
func (m *IntegrationManager) initializeRAGService() error {
	ragConfig := m.config.MTLSConfig.RAGService
	if ragConfig == nil || !ragConfig.Enabled {
		return nil
	}
	
	// Create RAG client
	clientConfig := &ClientConfig{
		ServiceName:          ragConfig.ServiceName,
		TenantID:            m.config.MTLSConfig.TenantID,
		ClientCertPath:       ragConfig.ClientCertPath,
		ClientKeyPath:        ragConfig.ClientKeyPath,
		CACertPath:          ragConfig.CACertPath,
		ServerName:          ragConfig.ServerName,
		InsecureSkipVerify:  ragConfig.InsecureSkipVerify,
		DialTimeout:         ragConfig.DialTimeout,
		MaxIdleConns:        ragConfig.MaxIdleConns,
		MaxConnsPerHost:     ragConfig.MaxConnsPerHost,
		IdleConnTimeout:     ragConfig.IdleConnTimeout,
		CertValidityDuration: m.config.MTLSConfig.ValidityDuration,
		RotationEnabled:     m.config.MTLSConfig.RotationEnabled,
		RotationInterval:    m.config.MTLSConfig.RotationInterval,
		RenewalThreshold:    m.config.MTLSConfig.RenewalThreshold,
		CAManager:           m.caManager,
		AutoProvision:       m.config.MTLSConfig.AutoProvision,
		PolicyTemplate:      m.config.MTLSConfig.PolicyTemplate,
	}
	
	var err error
	m.ragClient, err = NewClient(clientConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create RAG client: %w", err)
	}
	
	// Create RAG server
	serverConfig := &ServerConfig{
		ServiceName:          ragConfig.ServiceName,
		TenantID:            m.config.MTLSConfig.TenantID,
		Address:             "0.0.0.0",
		Port:                ragConfig.Port,
		ServerCertPath:      ragConfig.ServerCertPath,
		ServerKeyPath:       ragConfig.ServerKeyPath,
		CACertPath:         ragConfig.CACertPath,
		ClientCACertPath:   ragConfig.CACertPath,
		CertValidityDuration: m.config.MTLSConfig.ValidityDuration,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
		ClientCertValidation: ClientCertRequiredVerify,
		RotationEnabled:    m.config.MTLSConfig.RotationEnabled,
		RotationInterval:   m.config.MTLSConfig.RotationInterval,
		RenewalThreshold:   m.config.MTLSConfig.RenewalThreshold,
		CAManager:          m.caManager,
		AutoProvision:      m.config.MTLSConfig.AutoProvision,
		PolicyTemplate:     m.config.MTLSConfig.PolicyTemplate,
		EnableHSTS:         m.config.MTLSConfig.EnableHSTS,
		HSTSMaxAge:         m.config.MTLSConfig.HSTSMaxAge,
		AllowedClientCNs:   ragConfig.AllowedClientCNs,
		AllowedClientOrgs:  ragConfig.AllowedClientOrgs,
		ClientCertRequired: ragConfig.RequireClientCert,
	}
	
	m.ragServer, err = NewServer(serverConfig, m.logger)
	if err != nil {
		return fmt.Errorf("failed to create RAG server: %w", err)
	}
	
	// Start server in background
	go func() {
		if err := m.ragServer.Start(m.ctx); err != nil {
			m.logger.Error("RAG server failed", "error", err)
		}
	}()
	
	m.logger.Info("RAG service initialized with mTLS")
	
	return nil
}

// createServiceIdentities creates service identities for all enabled services
func (m *IntegrationManager) createServiceIdentities() error {
	if m.identityManager == nil {
		return nil
	}
	
	// Create identities for all enabled services
	services := []struct {
		config *config.ServiceMTLSConfig
		role   ServiceRole
	}{
		{m.config.MTLSConfig.Controller, RoleController},
		{m.config.MTLSConfig.LLMProcessor, RoleLLMService},
		{m.config.MTLSConfig.RAGService, RoleRAGService},
		{m.config.MTLSConfig.GitClient, RoleGitClient},
		{m.config.MTLSConfig.Database, RoleDatabaseClient},
		{m.config.MTLSConfig.NephioBridge, RoleNephioBridge},
		{m.config.MTLSConfig.ORANAdaptor, RoleORANAdaptor},
		{m.config.MTLSConfig.Monitoring, RoleMonitoring},
	}
	
	for _, service := range services {
		if service.config != nil && service.config.Enabled {
			_, err := m.identityManager.CreateServiceIdentity(
				service.config.ServiceName,
				service.role,
				m.config.MTLSConfig.TenantID,
			)
			if err != nil {
				m.logger.Warn("failed to create service identity",
					"service_name", service.config.ServiceName,
					"role", service.role,
					"error", err)
			} else {
				m.logger.Info("service identity created",
					"service_name", service.config.ServiceName,
					"role", service.role)
			}
		}
	}
	
	return nil
}

// GetLLMClient returns the mTLS-enabled LLM client
func (m *IntegrationManager) GetLLMClient() *Client {
	return m.llmClient
}

// GetRAGClient returns the mTLS-enabled RAG client
func (m *IntegrationManager) GetRAGClient() *Client {
	return m.ragClient
}

// GetIdentityManager returns the identity manager
func (m *IntegrationManager) GetIdentityManager() *IdentityManager {
	return m.identityManager
}

// GetMonitor returns the mTLS monitor
func (m *IntegrationManager) GetMonitor() *MTLSMonitor {
	return m.monitor
}

// GetMetrics returns comprehensive mTLS metrics
func (m *IntegrationManager) GetMetrics() ([]*Metric, error) {
	if m.monitor == nil {
		return nil, fmt.Errorf("monitor not initialized")
	}
	
	return m.monitor.GetMetrics()
}

// GetConnectionStats returns connection statistics
func (m *IntegrationManager) GetConnectionStats() *ConnectionStats {
	if m.monitor == nil {
		return &ConnectionStats{}
	}
	
	return m.monitor.GetConnectionStats()
}

// GetCertificateStats returns certificate statistics
func (m *IntegrationManager) GetCertificateStats() *CertificateStats {
	if m.monitor == nil {
		return &CertificateStats{}
	}
	
	return m.monitor.GetCertificateStats()
}

// GetIdentityStats returns identity statistics
func (m *IntegrationManager) GetIdentityStats() *IdentityStats {
	if m.identityManager == nil {
		return &IdentityStats{}
	}
	
	return m.identityManager.GetStats()
}

// RotateAllCertificates rotates certificates for all services
func (m *IntegrationManager) RotateAllCertificates(ctx context.Context) error {
	if m.identityManager == nil {
		return fmt.Errorf("identity manager not initialized")
	}
	
	identities := m.identityManager.ListServiceIdentities()
	
	for _, identity := range identities {
		err := m.identityManager.RotateServiceIdentity(
			identity.ServiceName,
			identity.Role,
			identity.TenantID,
		)
		if err != nil {
			m.logger.Error("failed to rotate certificate",
				"service_name", identity.ServiceName,
				"role", identity.Role,
				"error", err)
		} else {
			m.logger.Info("certificate rotated successfully",
				"service_name", identity.ServiceName,
				"role", identity.Role)
		}
	}
	
	return nil
}

// Close gracefully shuts down the integration manager
func (m *IntegrationManager) Close() error {
	m.logger.Info("shutting down mTLS integration manager")
	
	// Stop services
	m.cancel()
	
	// Close clients
	if m.llmClient != nil {
		m.llmClient.Close()
	}
	if m.ragClient != nil {
		m.ragClient.Close()
	}
	
	// Close servers
	if m.llmServer != nil {
		m.llmServer.Shutdown(context.Background())
	}
	if m.ragServer != nil {
		m.ragServer.Shutdown(context.Background())
	}
	
	// Close managers
	if m.identityManager != nil {
		m.identityManager.Close()
	}
	if m.monitor != nil {
		m.monitor.Close()
	}
	
	return nil
}

// ValidateConfiguration validates the mTLS configuration
func (m *IntegrationManager) ValidateConfiguration() error {
	if m.config.MTLSConfig == nil {
		return fmt.Errorf("mTLS configuration is required")
	}
	
	if m.config.MTLSConfig.Enabled {
		// Validate required fields when mTLS is enabled
		if m.config.MTLSConfig.TenantID == "" {
			return fmt.Errorf("tenant ID is required when mTLS is enabled")
		}
		
		if m.config.MTLSConfig.CertificateBaseDir == "" {
			return fmt.Errorf("certificate base directory is required when mTLS is enabled")
		}
		
		if m.config.MTLSConfig.AutoProvision && m.caManager == nil {
			return fmt.Errorf("CA manager is required when auto-provisioning is enabled")
		}
	}
	
	return nil
}

// ExampleUsage demonstrates how to use the integration manager
func ExampleUsage() {
	// This is an example of how to integrate mTLS into your application
	
	// 1. Load configuration
	appConfig, err := config.LoadFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	
	// 2. Create logger
	logger := logging.NewStructuredLogger()
	
	// 3. Initialize CA manager (assuming you have one)
	var caManager *ca.CAManager
	// caManager = initializeCAManager() // Your CA manager initialization
	
	// 4. Create integration manager
	integrationConfig := &IntegrationConfig{
		Config:    appConfig,
		Logger:    logger,
		CAManager: caManager,
	}
	
	manager, err := NewIntegrationManager(integrationConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create integration manager: %v", err))
	}
	defer manager.Close()
	
	// 5. Use the manager to get mTLS-enabled clients
	llmClient := manager.GetLLMClient()
	if llmClient != nil {
		// Use the LLM client for secure communications
		logger.Info("LLM client ready for secure communications")
	}
	
	ragClient := manager.GetRAGClient()
	if ragClient != nil {
		// Use the RAG client for secure communications
		logger.Info("RAG client ready for secure communications")
	}
	
	// 6. Monitor mTLS health
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				// Get and log metrics
				metrics, err := manager.GetMetrics()
				if err != nil {
					logger.Error("Failed to get mTLS metrics", "error", err)
					continue
				}
				
				logger.Info("mTLS metrics collected",
					"metric_count", len(metrics),
					"connection_stats", manager.GetConnectionStats(),
					"certificate_stats", manager.GetCertificateStats())
			}
		}
	}()
	
	// Your application logic here...
	
	logger.Info("Application running with comprehensive mTLS security")
}
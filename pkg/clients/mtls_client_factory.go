
package clients



import (

	"context"

	"fmt"

	"net/http"

	"sync"



	"github.com/thc1006/nephoran-intent-operator/pkg/config"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/mtls"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"

)



// ServiceType represents the type of service client.

type ServiceType string



const (

	// ServiceTypeLLM holds servicetypellm value.

	ServiceTypeLLM ServiceType = "llm"

	// ServiceTypeRAG holds servicetyperag value.

	ServiceTypeRAG ServiceType = "rag"

	// ServiceTypeGit holds servicetypegit value.

	ServiceTypeGit ServiceType = "git"

	// ServiceTypeDatabase holds servicetypedatabase value.

	ServiceTypeDatabase ServiceType = "database"

	// ServiceTypeNephio holds servicetypenephio value.

	ServiceTypeNephio ServiceType = "nephio"

	// ServiceTypeORAN holds servicetypeoran value.

	ServiceTypeORAN ServiceType = "oran"

	// ServiceTypeMonitoring holds servicetypemonitoring value.

	ServiceTypeMonitoring ServiceType = "monitoring"

)



// MTLSClientFactory creates and manages mTLS-enabled service clients.

type MTLSClientFactory struct {

	config          *config.Config

	logger          *logging.StructuredLogger

	caManager       *ca.CAManager

	identityManager *mtls.IdentityManager



	// Client cache.

	clients map[ServiceType]interface{}

	mu      sync.RWMutex



	// Lifecycle management.

	ctx    context.Context

	cancel context.CancelFunc

}



// ClientFactoryConfig holds configuration for the client factory.

type ClientFactoryConfig struct {

	Config          *config.Config

	Logger          *logging.StructuredLogger

	CAManager       *ca.CAManager

	IdentityManager *mtls.IdentityManager

}



// NewMTLSClientFactory creates a new mTLS client factory.

func NewMTLSClientFactory(config *ClientFactoryConfig) (*MTLSClientFactory, error) {

	if config == nil {

		return nil, fmt.Errorf("client factory config is required")

	}



	if config.Config == nil {

		return nil, fmt.Errorf("application config is required")

	}



	if config.Logger == nil {

		config.Logger = logging.NewStructuredLogger(logging.DefaultConfig("mtls-client-factory", "1.0.0", "production"))

	}



	ctx, cancel := context.WithCancel(context.Background())



	factory := &MTLSClientFactory{

		config:          config.Config,

		logger:          config.Logger,

		caManager:       config.CAManager,

		identityManager: config.IdentityManager,

		clients:         make(map[ServiceType]interface{}),

		ctx:             ctx,

		cancel:          cancel,

	}



	config.Logger.Info("mTLS client factory initialized",

		"mtls_enabled", config.Config.MTLSConfig != nil && config.Config.MTLSConfig.Enabled,

		"auto_provision", config.Config.MTLSConfig != nil && config.Config.MTLSConfig.AutoProvision,

		"ca_manager_available", config.CAManager != nil)



	return factory, nil

}



// GetLLMClient returns an mTLS-enabled LLM processor client.

func (f *MTLSClientFactory) GetLLMClient() (shared.ClientInterface, error) {

	f.mu.RLock()

	if client, exists := f.clients[ServiceTypeLLM]; exists {

		f.mu.RUnlock()

		return client.(shared.ClientInterface), nil

	}

	f.mu.RUnlock()



	f.mu.Lock()

	defer f.mu.Unlock()



	// Double-check after acquiring write lock.

	if client, exists := f.clients[ServiceTypeLLM]; exists {

		return client.(shared.ClientInterface), nil

	}



	// Create mTLS-enabled HTTP client.

	httpClient, err := f.createMTLSHTTPClient(ServiceTypeLLM)

	if err != nil {

		return nil, fmt.Errorf("failed to create mTLS HTTP client for LLM service: %w", err)

	}



	// Create LLM client with mTLS HTTP client.

	llmClient := &MTLSLLMClient{

		baseURL:    f.getServiceURL(ServiceTypeLLM),

		httpClient: httpClient,

		logger:     f.logger,

	}



	f.clients[ServiceTypeLLM] = llmClient



	f.logger.Info("created mTLS LLM client",

		"service_url", llmClient.baseURL,

		"mtls_enabled", f.isMTLSEnabled(ServiceTypeLLM))



	return llmClient, nil

}



// GetRAGClient returns an mTLS-enabled RAG service client.

func (f *MTLSClientFactory) GetRAGClient() (*MTLSRAGClient, error) {

	f.mu.RLock()

	if client, exists := f.clients[ServiceTypeRAG]; exists {

		f.mu.RUnlock()

		return client.(*MTLSRAGClient), nil

	}

	f.mu.RUnlock()



	f.mu.Lock()

	defer f.mu.Unlock()



	// Double-check after acquiring write lock.

	if client, exists := f.clients[ServiceTypeRAG]; exists {

		return client.(*MTLSRAGClient), nil

	}



	// Create mTLS-enabled HTTP client.

	httpClient, err := f.createMTLSHTTPClient(ServiceTypeRAG)

	if err != nil {

		return nil, fmt.Errorf("failed to create mTLS HTTP client for RAG service: %w", err)

	}



	// Create RAG client with mTLS HTTP client.

	ragClient := &MTLSRAGClient{

		baseURL:    f.getServiceURL(ServiceTypeRAG),

		httpClient: httpClient,

		logger:     f.logger,

	}



	f.clients[ServiceTypeRAG] = ragClient



	f.logger.Info("created mTLS RAG client",

		"service_url", ragClient.baseURL,

		"mtls_enabled", f.isMTLSEnabled(ServiceTypeRAG))



	return ragClient, nil

}



// GetGitClient returns an mTLS-enabled Git client.

func (f *MTLSClientFactory) GetGitClient() (*MTLSGitClient, error) {

	f.mu.RLock()

	if client, exists := f.clients[ServiceTypeGit]; exists {

		f.mu.RUnlock()

		return client.(*MTLSGitClient), nil

	}

	f.mu.RUnlock()



	f.mu.Lock()

	defer f.mu.Unlock()



	// Double-check after acquiring write lock.

	if client, exists := f.clients[ServiceTypeGit]; exists {

		return client.(*MTLSGitClient), nil

	}



	// Create mTLS-enabled HTTP client.

	httpClient, err := f.createMTLSHTTPClient(ServiceTypeGit)

	if err != nil {

		return nil, fmt.Errorf("failed to create mTLS HTTP client for Git service: %w", err)

	}



	// Create Git client with mTLS HTTP client.

	gitClient := &MTLSGitClient{

		httpClient: httpClient,

		logger:     f.logger,

		config:     f.config,

	}



	f.clients[ServiceTypeGit] = gitClient



	f.logger.Info("created mTLS Git client",

		"mtls_enabled", f.isMTLSEnabled(ServiceTypeGit))



	return gitClient, nil

}



// GetMonitoringClient returns an mTLS-enabled monitoring client.

func (f *MTLSClientFactory) GetMonitoringClient() (*MTLSMonitoringClient, error) {

	f.mu.RLock()

	if client, exists := f.clients[ServiceTypeMonitoring]; exists {

		f.mu.RUnlock()

		return client.(*MTLSMonitoringClient), nil

	}

	f.mu.RUnlock()



	f.mu.Lock()

	defer f.mu.Unlock()



	// Double-check after acquiring write lock.

	if client, exists := f.clients[ServiceTypeMonitoring]; exists {

		return client.(*MTLSMonitoringClient), nil

	}



	// Create mTLS-enabled HTTP client.

	httpClient, err := f.createMTLSHTTPClient(ServiceTypeMonitoring)

	if err != nil {

		return nil, fmt.Errorf("failed to create mTLS HTTP client for monitoring service: %w", err)

	}



	// Create monitoring client with mTLS HTTP client.

	monitoringClient := &MTLSMonitoringClient{

		httpClient: httpClient,

		logger:     f.logger,

	}



	f.clients[ServiceTypeMonitoring] = monitoringClient



	f.logger.Info("created mTLS monitoring client",

		"mtls_enabled", f.isMTLSEnabled(ServiceTypeMonitoring))



	return monitoringClient, nil

}



// createMTLSHTTPClient creates an mTLS-enabled HTTP client for a service type.

func (f *MTLSClientFactory) createMTLSHTTPClient(serviceType ServiceType) (*http.Client, error) {

	// If mTLS is not enabled globally or for this service, return regular HTTP client.

	if !f.isMTLSEnabled(serviceType) {

		f.logger.Debug("mTLS not enabled, creating regular HTTP client", "service_type", serviceType)

		return &http.Client{}, nil

	}



	serviceCfg := f.getServiceMTLSConfig(serviceType)

	if serviceCfg == nil {

		return nil, fmt.Errorf("mTLS configuration not found for service type: %s", serviceType)

	}



	// Provision service identity if needed.

	if err := f.ensureServiceIdentity(serviceType); err != nil {

		return nil, fmt.Errorf("failed to ensure service identity: %w", err)

	}



	// Create mTLS client configuration.

	clientConfig := &mtls.ClientConfig{

		ServiceName:          serviceCfg.ServiceName,

		TenantID:             f.config.MTLSConfig.TenantID,

		ClientCertPath:       serviceCfg.ClientCertPath,

		ClientKeyPath:        serviceCfg.ClientKeyPath,

		CACertPath:           serviceCfg.CACertPath,

		ServerName:           serviceCfg.ServerName,

		InsecureSkipVerify:   serviceCfg.InsecureSkipVerify,

		DialTimeout:          serviceCfg.DialTimeout,

		KeepAliveTimeout:     serviceCfg.KeepAliveTimeout,

		MaxIdleConns:         serviceCfg.MaxIdleConns,

		MaxConnsPerHost:      serviceCfg.MaxConnsPerHost,

		IdleConnTimeout:      serviceCfg.IdleConnTimeout,

		CertValidityDuration: f.config.MTLSConfig.ValidityDuration,

		RotationEnabled:      f.config.MTLSConfig.RotationEnabled,

		RotationInterval:     f.config.MTLSConfig.RotationInterval,

		RenewalThreshold:     f.config.MTLSConfig.RenewalThreshold,

		CAManager:            f.caManager,

		AutoProvision:        f.config.MTLSConfig.AutoProvision,

		PolicyTemplate:       f.config.MTLSConfig.PolicyTemplate,

	}



	// Create mTLS client.

	mtlsClient, err := mtls.NewClient(clientConfig, f.logger)

	if err != nil {

		return nil, fmt.Errorf("failed to create mTLS client: %w", err)

	}



	return mtlsClient.GetHTTPClient(), nil

}



// ensureServiceIdentity ensures that a service identity exists.

func (f *MTLSClientFactory) ensureServiceIdentity(serviceType ServiceType) error {

	if f.identityManager == nil {

		return nil // Identity manager not available

	}



	serviceCfg := f.getServiceMTLSConfig(serviceType)

	if serviceCfg == nil {

		return fmt.Errorf("service mTLS configuration not found for type: %s", serviceType)

	}



	// Get service role based on service type.

	role := f.getServiceRole(serviceType)



	// Check if identity already exists.

	_, err := f.identityManager.GetServiceIdentity(serviceCfg.ServiceName, role, f.config.MTLSConfig.TenantID)

	if err == nil {

		return nil // Identity already exists

	}



	// Create service identity.

	_, err = f.identityManager.CreateServiceIdentity(serviceCfg.ServiceName, role, f.config.MTLSConfig.TenantID)

	if err != nil {

		return fmt.Errorf("failed to create service identity: %w", err)

	}



	return nil

}



// isMTLSEnabled checks if mTLS is enabled for a service type.

func (f *MTLSClientFactory) isMTLSEnabled(serviceType ServiceType) bool {

	if f.config.MTLSConfig == nil || !f.config.MTLSConfig.Enabled {

		return false

	}



	serviceCfg := f.getServiceMTLSConfig(serviceType)

	return serviceCfg != nil && serviceCfg.Enabled

}



// getServiceMTLSConfig returns mTLS configuration for a service type.

func (f *MTLSClientFactory) getServiceMTLSConfig(serviceType ServiceType) *config.ServiceMTLSConfig {

	if f.config.MTLSConfig == nil {

		return nil

	}



	switch serviceType {

	case ServiceTypeLLM:

		return f.config.MTLSConfig.LLMProcessor

	case ServiceTypeRAG:

		return f.config.MTLSConfig.RAGService

	case ServiceTypeGit:

		return f.config.MTLSConfig.GitClient

	case ServiceTypeDatabase:

		return f.config.MTLSConfig.Database

	case ServiceTypeNephio:

		return f.config.MTLSConfig.NephioBridge

	case ServiceTypeORAN:

		return f.config.MTLSConfig.ORANAdaptor

	case ServiceTypeMonitoring:

		return f.config.MTLSConfig.Monitoring

	default:

		return nil

	}

}



// getServiceRole returns the service role for a service type.

func (f *MTLSClientFactory) getServiceRole(serviceType ServiceType) mtls.ServiceRole {

	switch serviceType {

	case ServiceTypeLLM:

		return mtls.RoleLLMService

	case ServiceTypeRAG:

		return mtls.RoleRAGService

	case ServiceTypeGit:

		return mtls.RoleGitClient

	case ServiceTypeDatabase:

		return mtls.RoleDatabaseClient

	case ServiceTypeNephio:

		return mtls.RoleNephioBridge

	case ServiceTypeORAN:

		return mtls.RoleORANAdaptor

	case ServiceTypeMonitoring:

		return mtls.RoleMonitoring

	default:

		return mtls.RoleController // Default role

	}

}



// getServiceURL returns the service URL for a service type.

func (f *MTLSClientFactory) getServiceURL(serviceType ServiceType) string {

	switch serviceType {

	case ServiceTypeLLM:

		if f.isMTLSEnabled(serviceType) {

			serviceCfg := f.getServiceMTLSConfig(serviceType)

			if serviceCfg != nil && serviceCfg.Port > 0 {

				return fmt.Sprintf("https://%s:%d", serviceCfg.ServerName, serviceCfg.Port)

			}

		}

		return f.config.LLMProcessorURL



	case ServiceTypeRAG:

		if f.isMTLSEnabled(serviceType) {

			serviceCfg := f.getServiceMTLSConfig(serviceType)

			if serviceCfg != nil && serviceCfg.Port > 0 {

				return fmt.Sprintf("https://%s:%d", serviceCfg.ServerName, serviceCfg.Port)

			}

		}

		return f.config.GetRAGAPIURL(true)



	default:

		return ""

	}

}



// Close gracefully shuts down the client factory.

func (f *MTLSClientFactory) Close() error {

	f.logger.Info("shutting down mTLS client factory")



	f.cancel()



	f.mu.Lock()

	defer f.mu.Unlock()



	// Close all clients that implement closer interface.

	for serviceType, client := range f.clients {

		if closer, ok := client.(interface{ Close() error }); ok {

			if err := closer.Close(); err != nil {

				f.logger.Warn("failed to close client",

					"service_type", serviceType,

					"error", err)

			}

		}

	}



	f.clients = make(map[ServiceType]interface{})



	return nil

}



// GetClientStats returns statistics about managed clients.

func (f *MTLSClientFactory) GetClientStats() *ClientStats {

	f.mu.RLock()

	defer f.mu.RUnlock()



	stats := &ClientStats{

		TotalClients:  len(f.clients),

		ClientTypes:   make(map[ServiceType]bool),

		MTLSEnabled:   f.config.MTLSConfig != nil && f.config.MTLSConfig.Enabled,

		AutoProvision: f.config.MTLSConfig != nil && f.config.MTLSConfig.AutoProvision,

	}



	for serviceType := range f.clients {

		stats.ClientTypes[serviceType] = f.isMTLSEnabled(serviceType)

	}



	return stats

}



// ClientStats holds statistics about managed clients.

type ClientStats struct {

	TotalClients  int                  `json:"total_clients"`

	ClientTypes   map[ServiceType]bool `json:"client_types"`

	MTLSEnabled   bool                 `json:"mtls_enabled"`

	AutoProvision bool                 `json:"auto_provision"`

}


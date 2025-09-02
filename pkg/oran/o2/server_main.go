// Package o2 implements the main entry point for the O2 IMS API server.

package o2

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

// ServerManager manages the lifecycle of the O2 IMS API server.

type ServerManager struct {
	config *O2IMSConfig

	server *O2APIServer

	logger *logging.StructuredLogger

	ctx context.Context

	cancel context.CancelFunc
}

// NewServerManager creates a new server manager.

func NewServerManager(config *O2IMSConfig) (*ServerManager, error) {
	if config == nil {
		config = DefaultO2IMSConfig()
	}

	logger := config.Logger

	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("o2-ims-server", "1.0.0", "production"))
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ServerManager{
		config: config,

		logger: logger,

		ctx: ctx,

		cancel: cancel,
	}, nil
}

// Start starts the O2 IMS API server.

func (sm *ServerManager) Start() error {
	sm.logger.Info("starting O2 IMS API server")

	// Create and configure the API server.

	server, err := NewO2APIServerWithConfig(sm.config)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	sm.server = server

	// Set up signal handling for graceful shutdown.

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine.

	serverErrChan := make(chan error, 1)

	go func() {
		if err := server.Start(sm.ctx); err != nil {
			serverErrChan <- err
		}
	}()

	sm.logger.Info("O2 IMS API server started successfully",

		"address", fmt.Sprintf("%s:%d", sm.config.Host, sm.config.Port),

		"tls_enabled", sm.config.TLSEnabled)

	// Wait for shutdown signal or server error.

	select {

	case sig := <-signalChan:

		sm.logger.Info("received shutdown signal", "signal", sig)

		return sm.Shutdown()

	case err := <-serverErrChan:

		sm.logger.Error("server error", "error", err)

		return err

	}
}

// Shutdown gracefully shuts down the server.

func (sm *ServerManager) Shutdown() error {
	sm.logger.Info("shutting down O2 IMS API server")

	// Cancel context to signal shutdown to all components.

	sm.cancel()

	// If server is running, shut it down.

	if sm.server != nil {

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		defer cancel()

		if err := sm.server.Shutdown(shutdownCtx); err != nil {

			sm.logger.Error("error during server shutdown", "error", err)

			return err

		}

	}

	sm.logger.Info("O2 IMS API server shutdown completed")

	return nil
}

// RunServer is a convenience function to run the server with default configuration.

func RunServer() error {
	config := DefaultO2IMSConfig()

	// Override with environment variables if available.

	config = applyEnvironmentOverrides(config)

	manager, err := NewServerManager(config)
	if err != nil {
		return fmt.Errorf("failed to create server manager: %w", err)
	}

	return manager.Start()
}

// RunServerWithConfig runs the server with custom configuration.

func RunServerWithConfig(config *O2IMSConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	manager, err := NewServerManager(config)
	if err != nil {
		return fmt.Errorf("failed to create server manager: %w", err)
	}

	return manager.Start()
}

// applyEnvironmentOverrides applies environment variable overrides to configuration.

func applyEnvironmentOverrides(config *O2IMSConfig) *O2IMSConfig {
	// Port override.

	if port := os.Getenv("O2_IMS_PORT"); port != "" {
		if p, err := parseInt(port); err == nil {
			config.Port = p
		}
	}

	// Host override.

	if host := os.Getenv("O2_IMS_HOST"); host != "" {
		config.Host = host
	}

	// TLS configuration.

	if tlsEnabled := os.Getenv("O2_IMS_TLS_ENABLED"); tlsEnabled == "true" {

		config.TLSEnabled = true

		if certFile := os.Getenv("O2_IMS_CERT_FILE"); certFile != "" {
			config.CertFile = certFile
		}

		if keyFile := os.Getenv("O2_IMS_KEY_FILE"); keyFile != "" {
			config.KeyFile = keyFile
		}

	}

	// Database configuration.

	if dbType := os.Getenv("O2_IMS_DB_TYPE"); dbType != "" {
		config.DatabaseType = dbType
	}

	if dbURL := os.Getenv("O2_IMS_DB_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}

	// Authentication configuration.

	if authEnabled := os.Getenv("O2_IMS_AUTH_ENABLED"); authEnabled == "true" {

		if config.AuthenticationConfig == nil {
			config.AuthenticationConfig = &AuthenticationConfig{}
		}

		config.AuthenticationConfig.Enabled = true

		if jwtSecret := os.Getenv("O2_IMS_JWT_SECRET"); jwtSecret != "" {
			config.AuthenticationConfig.JWTSecret = jwtSecret
		}

	}

	// Metrics configuration.

	if metricsEnabled := os.Getenv("O2_IMS_METRICS_ENABLED"); metricsEnabled == "true" {

		if config.MetricsConfig == nil {
			config.MetricsConfig = &MetricsConfig{}
		}

		config.MetricsConfig.Enabled = true

	}

	return config
}

// Helper function to parse integer from string.

func parseInt(s string) (int, error) {
	var result int

	_, err := fmt.Sscanf(s, "%d", &result)

	return result, err
}

// Production deployment helper functions.

// CreateProductionConfig creates a production-ready configuration.

func CreateProductionConfig() *O2IMSConfig {
	config := DefaultO2IMSConfig()

	// Production defaults.

	config.TLSEnabled = true

	config.DatabaseType = "postgres"

	// Security hardening.

	config.SecurityConfig.EnableCSRF = true

	config.SecurityConfig.CORSEnabled = true

	config.SecurityConfig.CORSAllowedOrigins = []string{} // Specific origins in production

	config.SecurityConfig.RateLimitConfig.Enabled = true

	config.SecurityConfig.RateLimitConfig.RequestsPerMin = 600 // Stricter in production

	config.SecurityConfig.AuditLogging = true

	// Authentication enabled.

	config.AuthenticationConfig.Enabled = true

	// Enhanced monitoring.

	config.MetricsConfig.Enabled = true

	config.HealthCheckConfig.Enabled = true

	config.HealthCheckConfig.DeepHealthCheck = true

	// Notifications enabled.

	config.NotificationConfig.Enabled = true

	// Resource management tuning.

	config.ResourceConfig.MaxConcurrentOperations = 50

	config.ResourceConfig.StateReconcileInterval = 30 * time.Second

	config.ResourceConfig.AutoDiscoveryEnabled = true

	return config
}

// CreateDevelopmentConfig creates a development-friendly configuration.

func CreateDevelopmentConfig() *O2IMSConfig {
	config := DefaultO2IMSConfig()

	// Development defaults.

	config.TLSEnabled = false

	config.DatabaseType = "memory"

	// Relaxed security for development.

	config.SecurityConfig.CORSAllowedOrigins = []string{"*"}

	config.SecurityConfig.RateLimitConfig.Enabled = false

	config.SecurityConfig.AuditLogging = false

	// Authentication disabled for easier development.

	config.AuthenticationConfig.Enabled = false

	// Basic monitoring.

	config.MetricsConfig.Enabled = true

	config.HealthCheckConfig.Enabled = true

	config.HealthCheckConfig.DeepHealthCheck = false

	return config
}

// ValidateConfiguration validates the server configuration.

func ValidateConfiguration(config *O2IMSConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate port.

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	// Validate TLS configuration.

	if config.TLSEnabled {

		if config.CertFile == "" {
			return fmt.Errorf("TLS enabled but cert file not specified")
		}

		if config.KeyFile == "" {
			return fmt.Errorf("TLS enabled but key file not specified")
		}

		// Check if files exist.

		if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file does not exist: %s", config.CertFile)
		}

		if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("key file does not exist: %s", config.KeyFile)
		}

	}

	// Validate database configuration.

	if config.DatabaseType != "memory" && config.DatabaseURL == "" {
		return fmt.Errorf("database URL required for database type: %s", config.DatabaseType)
	}

	// Validate authentication configuration.

	if config.AuthenticationConfig != nil && config.AuthenticationConfig.Enabled {
		if config.AuthenticationConfig.JWTSecret == "" {
			return fmt.Errorf("JWT secret required when authentication is enabled")
		}
	}

	// Validate CORS configuration.

	if config.SecurityConfig != nil && config.SecurityConfig.CORSEnabled {
		if err := middleware.ValidateConfig(middleware.CORSConfig{
			AllowedOrigins: config.SecurityConfig.CORSAllowedOrigins,

			AllowedMethods: config.SecurityConfig.CORSAllowedMethods,

			AllowedHeaders: config.SecurityConfig.CORSAllowedHeaders,

			AllowCredentials: config.AuthenticationConfig != nil && config.AuthenticationConfig.Enabled,
		}); err != nil {
			return fmt.Errorf("invalid CORS configuration: %w", err)
		}
	}

	return nil
}

// SetupDefaultProviders sets up default cloud providers.

func SetupDefaultProviders(config *O2IMSConfig) {
	if config.CloudProviders == nil {
		config.CloudProviders = []string{"kubernetes"}
	}

	// Initialize CloudProviderConfigs if nil
	if config.CloudProviderConfigs == nil {
		config.CloudProviderConfigs = make(map[string]*CloudProviderConfig)
	}

	// Add default Kubernetes provider if not exists.

	if _, exists := config.CloudProviderConfigs["kubernetes"]; !exists {
		config.CloudProviderConfigs["kubernetes"] = &CloudProviderConfig{
			ProviderID: "kubernetes",

			Name: "Default Kubernetes Provider",

			Type: CloudProviderKubernetes,

			Description: "Default Kubernetes provider for local development",

			Endpoint: "",

			Enabled: true,

			Status: "ACTIVE",

			Metadata: json.RawMessage("{}"),

			CreatedAt: time.Now(),

			UpdatedAt: time.Now(),
		}
	}
}

// Example usage and testing helpers.

// ExampleUsage demonstrates how to use the O2 IMS API server.

func ExampleUsage() {
	// Create configuration.

	config := CreateDevelopmentConfig()

	// Setup default providers.

	SetupDefaultProviders(config)

	// Validate configuration.

	if err := ValidateConfiguration(config); err != nil {

		fmt.Printf("Configuration validation failed: %v\n", err)

		return

	}

	// Create and start server.

	manager, err := NewServerManager(config)
	if err != nil {

		fmt.Printf("Failed to create server manager: %v\n", err)

		return

	}

	fmt.Println("Starting O2 IMS API server...")

	if err := manager.Start(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// TestConfiguration creates a test configuration for unit tests.

func TestConfiguration() *O2IMSConfig {
	config := DefaultO2IMSConfig()

	// Test-specific settings.

	config.Port = 0 // Let the OS assign a port

	config.DatabaseType = "memory"

	config.TLSEnabled = false

	config.AuthenticationConfig.Enabled = false

	config.SecurityConfig.RateLimitConfig.Enabled = false

	config.MetricsConfig.Enabled = false

	config.HealthCheckConfig.Enabled = false

	config.NotificationConfig.Enabled = false

	return config
}

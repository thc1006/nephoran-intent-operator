package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// Config holds all configuration for the Nephoran Intent Operator.

type Config struct {

	// Controller configuration.

	MetricsAddr string

	ProbeAddr string

	EnableLeaderElection bool

	// Intent processing features - control core functionality.

	EnableNetworkIntent bool // Enable/disable NetworkIntent controller (default: true) - controls whether NetworkIntent CRDs are reconciled

	EnableLLMIntent bool // Enable/disable LLM Intent processing (default: false) - enables AI-powered natural language intent interpretation

	// LLM Processor configuration.

	LLMProcessorURL string

	LLMProcessorTimeout time.Duration

	LLMTimeout time.Duration // Timeout for individual LLM requests in seconds (default: 15s) - balances response time vs reliability

	LLMMaxRetries int // Maximum retry attempts for LLM requests (default: 2) - uses exponential backoff, 0-10 range

	LLMCacheMaxEntries int // Maximum entries in LLM response cache (default: 512) - LRU eviction, ~1-5KB per entry

	// RAG API configuration.

	RAGAPIURLInternal string

	RAGAPIURLExternal string

	RAGAPITimeout time.Duration

	// Git integration configuration.

	GitRepoURL string

	GitToken string

	GitTokenPath string // Path to file containing Git token

	GitBranch string

	GitConcurrentPushLimit int // Maximum concurrent git operations (default 4 if <= 0)

	// Weaviate configuration.

	WeaviateURL string

	WeaviateIndex string

	// OpenAI configuration.

	OpenAIAPIKey string

	OpenAIModel string

	OpenAIEmbeddingModel string

	// Kubernetes configuration.

	Namespace string

	CRDPath string

	// HTTP configuration.

	HTTPMaxBody int64 // Maximum HTTP request body size in bytes (default: 1MB) - prevents DoS attacks, range 1KB-100MB

	// Metrics configuration - observability and security.

	MetricsEnabled bool // Enable/disable Prometheus metrics endpoint (default: false) - exposes /metrics for monitoring

	MetricsAllowedIPs []string // Comma-separated IP addresses allowed to access metrics - security control, empty blocks all, "*" allows all

	// mTLS Configuration.

	MTLSConfig *MTLSConfig
}

// MTLSConfig holds mTLS-specific configuration.

type MTLSConfig struct {

	// Global mTLS settings.

	Enabled bool `yaml:"enabled"`

	RequireClientCerts bool `yaml:"require_client_certs"`

	TenantID string `yaml:"tenant_id"`

	// Certificate Authority settings.

	CAManagerEnabled bool `yaml:"ca_manager_enabled"`

	AutoProvision bool `yaml:"auto_provision"`

	PolicyTemplate string `yaml:"policy_template"`

	// Certificate paths and settings.

	CertificateBaseDir string `yaml:"certificate_base_dir"`

	ValidityDuration time.Duration `yaml:"validity_duration"`

	RenewalThreshold time.Duration `yaml:"renewal_threshold"`

	RotationEnabled bool `yaml:"rotation_enabled"`

	RotationInterval time.Duration `yaml:"rotation_interval"`

	// Service-specific configurations.

	Controller *ServiceMTLSConfig `yaml:"controller"`

	LLMProcessor *ServiceMTLSConfig `yaml:"llm_processor"`

	RAGService *ServiceMTLSConfig `yaml:"rag_service"`

	GitClient *ServiceMTLSConfig `yaml:"git_client"`

	Database *ServiceMTLSConfig `yaml:"database"`

	NephioBridge *ServiceMTLSConfig `yaml:"nephio_bridge"`

	ORANAdaptor *ServiceMTLSConfig `yaml:"oran_adaptor"`

	Monitoring *ServiceMTLSConfig `yaml:"monitoring"`

	// TLS settings.

	MinTLSVersion string `yaml:"min_tls_version"`

	MaxTLSVersion string `yaml:"max_tls_version"`

	CipherSuites []string `yaml:"cipher_suites"`

	// Security settings.

	AllowedClientCNs []string `yaml:"allowed_client_cns"`

	AllowedClientOrgs []string `yaml:"allowed_client_orgs"`

	EnableHSTS bool `yaml:"enable_hsts"`

	HSTSMaxAge int64 `yaml:"hsts_max_age"`
}

// ServiceMTLSConfig holds mTLS configuration for a specific service.

type ServiceMTLSConfig struct {
	Enabled bool `yaml:"enabled"`

	ServiceName string `yaml:"service_name"`

	ServerCertPath string `yaml:"server_cert_path"`

	ServerKeyPath string `yaml:"server_key_path"`

	ClientCertPath string `yaml:"client_cert_path"`

	ClientKeyPath string `yaml:"client_key_path"`

	CACertPath string `yaml:"ca_cert_path"`

	ServerName string `yaml:"server_name"`

	Port int `yaml:"port"`

	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`

	// Connection settings.

	DialTimeout time.Duration `yaml:"dial_timeout"`

	KeepAliveTimeout time.Duration `yaml:"keep_alive_timeout"`

	MaxIdleConns int `yaml:"max_idle_conns"`

	MaxConnsPerHost int `yaml:"max_conns_per_host"`

	IdleConnTimeout time.Duration `yaml:"idle_conn_timeout"`

	// Service-specific settings.

	RequireClientCert bool `yaml:"require_client_cert"`

	AllowedClientCNs []string `yaml:"allowed_client_cns"`

	AllowedClientOrgs []string `yaml:"allowed_client_orgs"`
}

// DefaultConfig returns a configuration with sensible defaults.

func DefaultConfig() *Config {

	return &Config{

		MetricsAddr: ":8080",

		ProbeAddr: ":8081",

		EnableLeaderElection: false,

		// Intent processing features.

		EnableNetworkIntent: true, // Default enabled

		EnableLLMIntent: false, // Default disabled

		// LLM configuration.

		LLMProcessorURL: "http://llm-processor.default.svc.cluster.local:8080",

		LLMProcessorTimeout: 30 * time.Second,

		LLMTimeout: 15 * time.Second, // Default 15s for individual requests

		LLMMaxRetries: 2, // Default 2 retries

		LLMCacheMaxEntries: 512, // Default 512 cache entries

		RAGAPIURLInternal: "http://rag-api.default.svc.cluster.local:5001",

		RAGAPIURLExternal: "http://localhost:5001",

		RAGAPITimeout: 30 * time.Second,

		GitBranch: "main",

		GitConcurrentPushLimit: 4, // Default value

		WeaviateURL: "http://weaviate.default.svc.cluster.local:8080",

		WeaviateIndex: "telecom_knowledge",

		OpenAIModel: "gpt-4o-mini",

		OpenAIEmbeddingModel: "text-embedding-3-large",

		Namespace: "default",

		CRDPath: "deployments/crds",

		// HTTP configuration.

		HTTPMaxBody: 1048576, // Default 1MB

		// Metrics configuration.

		MetricsEnabled: false, // Default disabled for security

		MetricsAllowedIPs: []string{}, // Empty by default

		MTLSConfig: DefaultMTLSConfig(),
	}

}

// DefaultMTLSConfig returns default mTLS configuration.

func DefaultMTLSConfig() *MTLSConfig {

	baseDir := "/etc/nephoran/certs"

	return &MTLSConfig{

		Enabled: false, // Disabled by default for backward compatibility

		RequireClientCerts: true,

		TenantID: "default",

		CAManagerEnabled: true,

		AutoProvision: false,

		PolicyTemplate: "service-auth",

		CertificateBaseDir: baseDir,

		ValidityDuration: 24 * time.Hour,

		RenewalThreshold: 6 * time.Hour,

		RotationEnabled: true,

		RotationInterval: 1 * time.Hour,

		Controller: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "nephoran-controller",

			ServerCertPath: fmt.Sprintf("%s/default/nephoran-controller/tls.crt", baseDir),

			ServerKeyPath: fmt.Sprintf("%s/default/nephoran-controller/tls.key", baseDir),

			ClientCertPath: fmt.Sprintf("%s/default/nephoran-controller/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/nephoran-controller/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/nephoran-controller/ca.crt", baseDir),

			Port: 8443,

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 100,

			MaxConnsPerHost: 10,

			IdleConnTimeout: 90 * time.Second,

			RequireClientCert: true,
		},

		LLMProcessor: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "llm-processor",

			ServerName: "llm-processor.default.svc.cluster.local",

			ServerCertPath: fmt.Sprintf("%s/default/llm-processor/tls.crt", baseDir),

			ServerKeyPath: fmt.Sprintf("%s/default/llm-processor/tls.key", baseDir),

			ClientCertPath: fmt.Sprintf("%s/default/llm-processor/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/llm-processor/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/llm-processor/ca.crt", baseDir),

			Port: 8443,

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 50,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 60 * time.Second,

			RequireClientCert: true,
		},

		RAGService: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "rag-api",

			ServerName: "rag-api.default.svc.cluster.local",

			ServerCertPath: fmt.Sprintf("%s/default/rag-api/tls.crt", baseDir),

			ServerKeyPath: fmt.Sprintf("%s/default/rag-api/tls.key", baseDir),

			ClientCertPath: fmt.Sprintf("%s/default/rag-api/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/rag-api/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/rag-api/ca.crt", baseDir),

			Port: 5443,

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 50,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 60 * time.Second,

			RequireClientCert: true,
		},

		GitClient: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "git-client",

			ClientCertPath: fmt.Sprintf("%s/default/git-client/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/git-client/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/git-client/ca.crt", baseDir),

			DialTimeout: 30 * time.Second,

			MaxIdleConns: 20,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 120 * time.Second,
		},

		Database: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "database-client",

			ClientCertPath: fmt.Sprintf("%s/default/database-client/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/database-client/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/database-client/ca.crt", baseDir),

			DialTimeout: 15 * time.Second,

			MaxIdleConns: 25,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 180 * time.Second,
		},

		NephioBridge: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "nephio-bridge",

			ServerName: "nephio-bridge.default.svc.cluster.local",

			ServerCertPath: fmt.Sprintf("%s/default/nephio-bridge/tls.crt", baseDir),

			ServerKeyPath: fmt.Sprintf("%s/default/nephio-bridge/tls.key", baseDir),

			ClientCertPath: fmt.Sprintf("%s/default/nephio-bridge/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/nephio-bridge/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/nephio-bridge/ca.crt", baseDir),

			Port: 8443,

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 30,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 90 * time.Second,

			RequireClientCert: true,
		},

		ORANAdaptor: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "oran-adaptor",

			ServerName: "oran-adaptor.default.svc.cluster.local",

			ServerCertPath: fmt.Sprintf("%s/default/oran-adaptor/tls.crt", baseDir),

			ServerKeyPath: fmt.Sprintf("%s/default/oran-adaptor/tls.key", baseDir),

			ClientCertPath: fmt.Sprintf("%s/default/oran-adaptor/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/oran-adaptor/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/oran-adaptor/ca.crt", baseDir),

			Port: 8443,

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 30,

			MaxConnsPerHost: 5,

			IdleConnTimeout: 90 * time.Second,

			RequireClientCert: true,
		},

		Monitoring: &ServiceMTLSConfig{

			Enabled: false,

			ServiceName: "monitoring-client",

			ClientCertPath: fmt.Sprintf("%s/default/monitoring-client/client.crt", baseDir),

			ClientKeyPath: fmt.Sprintf("%s/default/monitoring-client/client.key", baseDir),

			CACertPath: fmt.Sprintf("%s/default/monitoring-client/ca.crt", baseDir),

			DialTimeout: 10 * time.Second,

			MaxIdleConns: 20,

			MaxConnsPerHost: 3,

			IdleConnTimeout: 60 * time.Second,
		},

		MinTLSVersion: "1.2",

		MaxTLSVersion: "1.3",

		EnableHSTS: true,

		HSTSMaxAge: 31536000, // 1 year

		AllowedClientCNs: []string{}, // Empty means allow all valid certificates

		AllowedClientOrgs: []string{}, // Empty means allow all valid certificates

	}

}

// LoadFromEnv loads configuration from environment variables.

func LoadFromEnv() (*Config, error) {

	cfg := DefaultConfig()

	// Override defaults with environment variables if they exist.

	cfg.MetricsAddr = GetEnvOrDefault("METRICS_ADDR", cfg.MetricsAddr)

	cfg.ProbeAddr = GetEnvOrDefault("PROBE_ADDR", cfg.ProbeAddr)

	cfg.EnableLeaderElection = GetBoolEnv("ENABLE_LEADER_ELECTION", cfg.EnableLeaderElection)

	// Intent processing features.

	cfg.EnableNetworkIntent = GetBoolEnv("ENABLE_NETWORK_INTENT", cfg.EnableNetworkIntent)

	cfg.EnableLLMIntent = GetBoolEnv("ENABLE_LLM_INTENT", cfg.EnableLLMIntent)

	// LLM configuration.

	cfg.LLMProcessorURL = GetEnvOrDefault("LLM_PROCESSOR_URL", cfg.LLMProcessorURL)

	cfg.LLMProcessorTimeout = GetDurationEnv("LLM_PROCESSOR_TIMEOUT", cfg.LLMProcessorTimeout)

	// Handle LLM_TIMEOUT_SECS as integer seconds value for consistency with LLM implementations.

	if envTimeoutSecs := GetEnvOrDefault("LLM_TIMEOUT_SECS", ""); envTimeoutSecs != "" {

		if timeoutSecs, err := strconv.Atoi(envTimeoutSecs); err == nil && timeoutSecs > 0 {

			cfg.LLMTimeout = time.Duration(timeoutSecs) * time.Second

		} else {

			// Log parsing error but continue with default.

			log.Printf("Config warning: LLM_TIMEOUT_SECS parsing error: %v, using default %v", err, cfg.LLMTimeout)

		}

	}

	cfg.LLMMaxRetries = GetIntEnv("LLM_MAX_RETRIES", cfg.LLMMaxRetries)

	cfg.LLMCacheMaxEntries = GetIntEnv("LLM_CACHE_MAX_ENTRIES", cfg.LLMCacheMaxEntries)

	cfg.RAGAPIURLInternal = GetEnvOrDefault("RAG_API_URL", cfg.RAGAPIURLInternal)

	cfg.RAGAPIURLExternal = GetEnvOrDefault("RAG_API_URL_EXTERNAL", cfg.RAGAPIURLExternal)

	cfg.RAGAPITimeout = GetDurationEnv("RAG_API_TIMEOUT", cfg.RAGAPITimeout)

	cfg.GitRepoURL = GetEnvOrDefault("GIT_REPO_URL", cfg.GitRepoURL)

	// Check for token file path first.

	cfg.GitTokenPath = GetEnvOrDefault("GIT_TOKEN_PATH", cfg.GitTokenPath)

	if cfg.GitTokenPath != "" {

		// Try to read the token from file.

		if tokenData, err := os.ReadFile(cfg.GitTokenPath); err == nil {

			cfg.GitToken = strings.TrimSpace(string(tokenData))

		}

		// If file read fails, GitToken will remain empty and fallback will occur.

	}

	// Fallback to direct environment variable if no token loaded from file.

	if cfg.GitToken == "" {

		cfg.GitToken = GetEnvOrDefault("GIT_TOKEN", cfg.GitToken)

	}

	cfg.GitBranch = GetEnvOrDefault("GIT_BRANCH", cfg.GitBranch)

	// Use GetIntEnv with validation for positive values only.

	if limit := GetIntEnv("GIT_CONCURRENT_PUSH_LIMIT", cfg.GitConcurrentPushLimit); limit > 0 {

		cfg.GitConcurrentPushLimit = limit

	}

	cfg.WeaviateURL = GetEnvOrDefault("WEAVIATE_URL", cfg.WeaviateURL)

	cfg.WeaviateIndex = GetEnvOrDefault("WEAVIATE_INDEX", cfg.WeaviateIndex)

	cfg.OpenAIAPIKey = GetEnvOrDefault("OPENAI_API_KEY", cfg.OpenAIAPIKey)

	cfg.OpenAIModel = GetEnvOrDefault("OPENAI_MODEL", cfg.OpenAIModel)

	cfg.OpenAIEmbeddingModel = GetEnvOrDefault("OPENAI_EMBEDDING_MODEL", cfg.OpenAIEmbeddingModel)

	cfg.Namespace = GetEnvOrDefault("NAMESPACE", cfg.Namespace)

	cfg.CRDPath = GetEnvOrDefault("CRD_PATH", cfg.CRDPath)

	// HTTP configuration.

	cfg.HTTPMaxBody = GetInt64Env("HTTP_MAX_BODY", cfg.HTTPMaxBody)

	// Metrics configuration.

	cfg.MetricsEnabled = GetBoolEnv("METRICS_ENABLED", cfg.MetricsEnabled)

	cfg.MetricsAllowedIPs = GetStringSliceEnv("METRICS_ALLOWED_IPS", cfg.MetricsAllowedIPs)

	// Security validation for metrics configuration.

	if cfg.MetricsEnabled && len(cfg.MetricsAllowedIPs) == 0 {

		// Check if user explicitly set "*" to allow all access.

		if metricsIPsEnv := GetEnvOrDefault("METRICS_ALLOWED_IPS", ""); metricsIPsEnv == "*" {

			cfg.MetricsAllowedIPs = []string{"*"}

			log.Printf("Config warning: Metrics endpoint is enabled with unrestricted access (METRICS_ALLOWED_IPS=*). This may expose sensitive information.")

		} else {

			log.Printf("Config warning: Metrics endpoint is enabled but METRICS_ALLOWED_IPS is empty. Consider setting specific IP addresses or '*' for unrestricted access.")

		}

	}

	// Load mTLS configuration from environment.

	loadMTLSFromEnv(cfg)

	// Validate required configuration.

	if err := cfg.Validate(); err != nil {

		return nil, fmt.Errorf("configuration validation failed: %w", err)

	}

	return cfg, nil

}

// Validate checks that required configuration is present.

func (c *Config) Validate() error {

	var errors []string

	// Always required configuration.

	if c.OpenAIAPIKey == "" {

		errors = append(errors, "OPENAI_API_KEY is required")

	}

	// Git features validation - enabled when Git integration is intended.

	// Git features are considered enabled when GitRepoURL is configured or.

	// when other Git-related config suggests Git usage.

	if c.isGitFeatureEnabled() {

		if c.GitRepoURL == "" {

			errors = append(errors, "GIT_REPO_URL is required when Git features are enabled")

		}

	}

	// LLM processing validation - enabled when LLM processing is intended.

	// LLM processing is considered enabled when LLMProcessorURL is configured.

	if c.isLLMProcessingEnabled() {

		if c.LLMProcessorURL == "" {

			errors = append(errors, "LLM_PROCESSOR_URL is required when LLM processing is enabled")

		}

	}

	// RAG features validation - enabled when RAG features are intended.

	// RAG features are considered enabled when RAG API URL is configured or.

	// when RAG-related infrastructure is configured.

	if c.isRAGFeatureEnabled() {

		if c.RAGAPIURLInternal == "" {

			errors = append(errors, "RAG_API_URL_INTERNAL is required when RAG features are enabled")

		}

	}

	// Return validation errors if any.

	if len(errors) > 0 {

		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))

	}

	return nil

}

// isGitFeatureEnabled checks if Git features are enabled based on configuration.

func (c *Config) isGitFeatureEnabled() bool {

	// Git features are enabled when GitRepoURL is set or when Git token is provided.

	// (indicating intention to use Git even if URL is missing - validation error).

	return c.GitRepoURL != "" || c.GitToken != ""

}

// isLLMProcessingEnabled checks if LLM processing is enabled based on configuration.

func (c *Config) isLLMProcessingEnabled() bool {

	// LLM processing is enabled when LLMProcessorURL is set.

	// This follows the pattern seen in networkintent_constructor.go.

	return c.LLMProcessorURL != ""

}

// isRAGFeatureEnabled checks if RAG features are enabled based on configuration.

func (c *Config) isRAGFeatureEnabled() bool {

	// RAG features are enabled when RAG API URL is set or when Weaviate is configured.

	// (indicating intention to use RAG even if internal URL is missing - validation error).

	return c.RAGAPIURLInternal != "" || c.RAGAPIURLExternal != "" || c.WeaviateURL != ""

}

// GetRAGAPIURL returns the appropriate RAG API URL based on environment.

func (c *Config) GetRAGAPIURL(useInternal bool) string {

	if useInternal {

		return c.RAGAPIURLInternal

	}

	return c.RAGAPIURLExternal

}

// loadMTLSFromEnv loads mTLS configuration from environment variables.

func loadMTLSFromEnv(cfg *Config) {

	if cfg.MTLSConfig == nil {

		return

	}

	// Global mTLS settings.

	cfg.MTLSConfig.Enabled = GetBoolEnv("MTLS_ENABLED", cfg.MTLSConfig.Enabled)

	cfg.MTLSConfig.RequireClientCerts = GetBoolEnv("MTLS_REQUIRE_CLIENT_CERTS", cfg.MTLSConfig.RequireClientCerts)

	cfg.MTLSConfig.TenantID = GetEnvOrDefault("MTLS_TENANT_ID", cfg.MTLSConfig.TenantID)

	cfg.MTLSConfig.CAManagerEnabled = GetBoolEnv("MTLS_CA_MANAGER_ENABLED", cfg.MTLSConfig.CAManagerEnabled)

	cfg.MTLSConfig.AutoProvision = GetBoolEnv("MTLS_AUTO_PROVISION", cfg.MTLSConfig.AutoProvision)

	cfg.MTLSConfig.PolicyTemplate = GetEnvOrDefault("MTLS_POLICY_TEMPLATE", cfg.MTLSConfig.PolicyTemplate)

	cfg.MTLSConfig.CertificateBaseDir = GetEnvOrDefault("MTLS_CERTIFICATE_BASE_DIR", cfg.MTLSConfig.CertificateBaseDir)

	cfg.MTLSConfig.ValidityDuration = GetDurationEnv("MTLS_VALIDITY_DURATION", cfg.MTLSConfig.ValidityDuration)

	cfg.MTLSConfig.RenewalThreshold = GetDurationEnv("MTLS_RENEWAL_THRESHOLD", cfg.MTLSConfig.RenewalThreshold)

	cfg.MTLSConfig.RotationEnabled = GetBoolEnv("MTLS_ROTATION_ENABLED", cfg.MTLSConfig.RotationEnabled)

	cfg.MTLSConfig.RotationInterval = GetDurationEnv("MTLS_ROTATION_INTERVAL", cfg.MTLSConfig.RotationInterval)

	// TLS version settings.

	cfg.MTLSConfig.MinTLSVersion = GetEnvOrDefault("MTLS_MIN_TLS_VERSION", cfg.MTLSConfig.MinTLSVersion)

	cfg.MTLSConfig.MaxTLSVersion = GetEnvOrDefault("MTLS_MAX_TLS_VERSION", cfg.MTLSConfig.MaxTLSVersion)

	// Security settings.

	cfg.MTLSConfig.EnableHSTS = GetBoolEnv("MTLS_ENABLE_HSTS", cfg.MTLSConfig.EnableHSTS)

	cfg.MTLSConfig.HSTSMaxAge = GetInt64Env("MTLS_HSTS_MAX_AGE", cfg.MTLSConfig.HSTSMaxAge)

	cfg.MTLSConfig.AllowedClientCNs = GetStringSliceEnv("MTLS_ALLOWED_CLIENT_CNS", cfg.MTLSConfig.AllowedClientCNs)

	cfg.MTLSConfig.AllowedClientOrgs = GetStringSliceEnv("MTLS_ALLOWED_CLIENT_ORGS", cfg.MTLSConfig.AllowedClientOrgs)

	// Service-specific settings - Controller.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.Controller, "CONTROLLER")

	// Service-specific settings - LLM Processor.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.LLMProcessor, "LLM_PROCESSOR")

	// Service-specific settings - RAG Service.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.RAGService, "RAG_SERVICE")

	// Service-specific settings - Git Client.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.GitClient, "GIT_CLIENT")

	// Service-specific settings - Database.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.Database, "DATABASE")

	// Service-specific settings - Nephio Bridge.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.NephioBridge, "NEPHIO_BRIDGE")

	// Service-specific settings - ORAN Adaptor.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.ORANAdaptor, "ORAN_ADAPTOR")

	// Service-specific settings - Monitoring.

	loadServiceMTLSFromEnv(cfg.MTLSConfig.Monitoring, "MONITORING")

}

// loadServiceMTLSFromEnv loads mTLS configuration for a specific service.

func loadServiceMTLSFromEnv(serviceCfg *ServiceMTLSConfig, prefix string) {

	if serviceCfg == nil {

		return

	}

	serviceCfg.Enabled = GetBoolEnv(fmt.Sprintf("MTLS_%s_ENABLED", prefix), serviceCfg.Enabled)

	serviceCfg.ServiceName = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_SERVICE_NAME", prefix), serviceCfg.ServiceName)

	serviceCfg.ServerCertPath = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_SERVER_CERT_PATH", prefix), serviceCfg.ServerCertPath)

	serviceCfg.ServerKeyPath = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_SERVER_KEY_PATH", prefix), serviceCfg.ServerKeyPath)

	serviceCfg.ClientCertPath = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_CLIENT_CERT_PATH", prefix), serviceCfg.ClientCertPath)

	serviceCfg.ClientKeyPath = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_CLIENT_KEY_PATH", prefix), serviceCfg.ClientKeyPath)

	serviceCfg.CACertPath = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_CA_CERT_PATH", prefix), serviceCfg.CACertPath)

	serviceCfg.ServerName = GetEnvOrDefault(fmt.Sprintf("MTLS_%s_SERVER_NAME", prefix), serviceCfg.ServerName)

	serviceCfg.Port = GetIntEnv(fmt.Sprintf("MTLS_%s_PORT", prefix), serviceCfg.Port)

	serviceCfg.InsecureSkipVerify = GetBoolEnv(fmt.Sprintf("MTLS_%s_INSECURE_SKIP_VERIFY", prefix), serviceCfg.InsecureSkipVerify)

	serviceCfg.DialTimeout = GetDurationEnv(fmt.Sprintf("MTLS_%s_DIAL_TIMEOUT", prefix), serviceCfg.DialTimeout)

	serviceCfg.KeepAliveTimeout = GetDurationEnv(fmt.Sprintf("MTLS_%s_KEEP_ALIVE_TIMEOUT", prefix), serviceCfg.KeepAliveTimeout)

	serviceCfg.MaxIdleConns = GetIntEnv(fmt.Sprintf("MTLS_%s_MAX_IDLE_CONNS", prefix), serviceCfg.MaxIdleConns)

	serviceCfg.MaxConnsPerHost = GetIntEnv(fmt.Sprintf("MTLS_%s_MAX_CONNS_PER_HOST", prefix), serviceCfg.MaxConnsPerHost)

	serviceCfg.IdleConnTimeout = GetDurationEnv(fmt.Sprintf("MTLS_%s_IDLE_CONN_TIMEOUT", prefix), serviceCfg.IdleConnTimeout)

	serviceCfg.RequireClientCert = GetBoolEnv(fmt.Sprintf("MTLS_%s_REQUIRE_CLIENT_CERT", prefix), serviceCfg.RequireClientCert)

	serviceCfg.AllowedClientCNs = GetStringSliceEnv(fmt.Sprintf("MTLS_%s_ALLOWED_CLIENT_CNS", prefix), serviceCfg.AllowedClientCNs)

	serviceCfg.AllowedClientOrgs = GetStringSliceEnv(fmt.Sprintf("MTLS_%s_ALLOWED_CLIENT_ORGS", prefix), serviceCfg.AllowedClientOrgs)

}

// ConfigProvider interface methods.

// GetLLMProcessorURL returns the LLM processor URL.

func (c *Config) GetLLMProcessorURL() string {

	return c.LLMProcessorURL

}

// GetLLMProcessorTimeout returns the LLM processor timeout.

func (c *Config) GetLLMProcessorTimeout() time.Duration {

	return c.LLMProcessorTimeout

}

// GetGitRepoURL returns the Git repository URL.

func (c *Config) GetGitRepoURL() string {

	return c.GitRepoURL

}

// GetGitToken returns the Git token.

func (c *Config) GetGitToken() string {

	return c.GitToken

}

// GetGitBranch returns the Git branch.

func (c *Config) GetGitBranch() string {

	return c.GitBranch

}

// GetWeaviateURL returns the Weaviate URL.

func (c *Config) GetWeaviateURL() string {

	return c.WeaviateURL

}

// GetWeaviateIndex returns the Weaviate index name.

func (c *Config) GetWeaviateIndex() string {

	return c.WeaviateIndex

}

// GetOpenAIAPIKey returns the OpenAI API key.

func (c *Config) GetOpenAIAPIKey() string {

	return c.OpenAIAPIKey

}

// GetOpenAIModel returns the OpenAI model.

func (c *Config) GetOpenAIModel() string {

	return c.OpenAIModel

}

// GetOpenAIEmbeddingModel returns the OpenAI embedding model.

func (c *Config) GetOpenAIEmbeddingModel() string {

	return c.OpenAIEmbeddingModel

}

// GetNamespace returns the Kubernetes namespace.

func (c *Config) GetNamespace() string {

	return c.Namespace

}

// GetEnableNetworkIntent returns whether NetworkIntent controller is enabled.

func (c *Config) GetEnableNetworkIntent() bool {

	return c.EnableNetworkIntent

}

// GetEnableLLMIntent returns whether LLM Intent processing is enabled.

func (c *Config) GetEnableLLMIntent() bool {

	return c.EnableLLMIntent

}

// GetLLMTimeout returns the timeout for individual LLM requests.

func (c *Config) GetLLMTimeout() time.Duration {

	return c.LLMTimeout

}

// GetLLMMaxRetries returns the maximum retry attempts for LLM requests.

func (c *Config) GetLLMMaxRetries() int {

	return c.LLMMaxRetries

}

// GetLLMCacheMaxEntries returns the maximum entries in LLM cache.

func (c *Config) GetLLMCacheMaxEntries() int {

	return c.LLMCacheMaxEntries

}

// GetHTTPMaxBody returns the maximum HTTP request body size.

func (c *Config) GetHTTPMaxBody() int64 {

	return c.HTTPMaxBody

}

// GetMetricsEnabled returns whether metrics endpoint is enabled.

func (c *Config) GetMetricsEnabled() bool {

	return c.MetricsEnabled

}

// GetMetricsAllowedIPs returns the IP addresses allowed to access metrics.

func (c *Config) GetMetricsAllowedIPs() []string {

	return c.MetricsAllowedIPs

}

// Ensure Config implements interfaces.ConfigProvider.

var _ interfaces.ConfigProvider = (*Config)(nil)

// APIKeys provides type alias for disable_rag builds.

type APIKeys = interfaces.APIKeys

// SecretManager provides a concrete type for disable_rag builds.

type SecretManager struct {
	namespace string
}

// NewSecretManager creates a new secret manager for disable_rag builds.

func NewSecretManager(namespace string) (*SecretManager, error) {

	return &SecretManager{namespace: namespace}, nil

}

// LoadFileBasedAPIKeysWithValidation performs loadfilebasedapikeyswithvalidation operation.

func LoadFileBasedAPIKeysWithValidation() (*APIKeys, error) {

	return &APIKeys{}, nil

}

// GetSecretValue retrieves secret value for disable_rag builds (stub implementation).

func (sm *SecretManager) GetSecretValue(ctx context.Context, secretName, key, envVarName string) (string, error) {

	return "", fmt.Errorf("secret manager disabled with disable_rag build tag")

}

// CreateSecretFromEnvVars performs createsecretfromenvvars operation.

func (sm *SecretManager) CreateSecretFromEnvVars(ctx context.Context, secretName string, envVarMapping map[string]string) error {

	return fmt.Errorf("secret manager disabled with disable_rag build tag")

}

// UpdateSecret performs updatesecret operation.

func (sm *SecretManager) UpdateSecret(ctx context.Context, secretName string, data map[string][]byte) error {

	return fmt.Errorf("secret manager disabled with disable_rag build tag")

}

// SecretExists performs secretexists operation.

func (sm *SecretManager) SecretExists(ctx context.Context, secretName string) bool {

	return false

}

// RotateSecret performs rotatesecret operation.

func (sm *SecretManager) RotateSecret(ctx context.Context, secretName, secretKey, newValue string) error {

	return fmt.Errorf("secret manager disabled with disable_rag build tag")

}

// GetSecretRotationInfo performs getsecretrotationinfo operation.

func (sm *SecretManager) GetSecretRotationInfo(ctx context.Context, secretName string) (map[string]string, error) {

	return nil, fmt.Errorf("secret manager disabled with disable_rag build tag")

}

// GetAPIKeys performs getapikeys operation.

func (sm *SecretManager) GetAPIKeys(ctx context.Context) (*APIKeys, error) {

	return &APIKeys{}, nil

}

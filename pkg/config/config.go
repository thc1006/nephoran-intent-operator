package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// Config holds all configuration for the Nephoran Intent Operator
type Config struct {
	// Controller configuration
	MetricsAddr          string
	ProbeAddr            string
	EnableLeaderElection bool

	// LLM Processor configuration
	LLMProcessorURL     string
	LLMProcessorTimeout time.Duration

	// RAG API configuration
	RAGAPIURLInternal string
	RAGAPIURLExternal string
	RAGAPITimeout     time.Duration

	// Git integration configuration
	GitRepoURL  string
	GitToken    string
	GitTokenPath string  // Path to file containing Git token
	GitBranch   string
	GitConcurrentPushLimit int  // Maximum concurrent git operations (default 4 if <= 0)

	// Weaviate configuration
	WeaviateURL   string
	WeaviateIndex string

	// OpenAI configuration
	OpenAIAPIKey         string
	OpenAIModel          string
	OpenAIEmbeddingModel string

	// Kubernetes configuration
	Namespace string
	CRDPath   string

	// mTLS Configuration
	MTLSConfig *MTLSConfig
}

// MTLSConfig holds mTLS-specific configuration
type MTLSConfig struct {
	// Global mTLS settings
	Enabled             bool   `yaml:"enabled"`
	RequireClientCerts  bool   `yaml:"require_client_certs"`
	TenantID           string `yaml:"tenant_id"`
	
	// Certificate Authority settings
	CAManagerEnabled    bool   `yaml:"ca_manager_enabled"`
	AutoProvision      bool   `yaml:"auto_provision"`
	PolicyTemplate     string `yaml:"policy_template"`
	
	// Certificate paths and settings
	CertificateBaseDir string        `yaml:"certificate_base_dir"`
	ValidityDuration   time.Duration `yaml:"validity_duration"`
	RenewalThreshold   time.Duration `yaml:"renewal_threshold"`
	RotationEnabled    bool          `yaml:"rotation_enabled"`
	RotationInterval   time.Duration `yaml:"rotation_interval"`
	
	// Service-specific configurations
	Controller    *ServiceMTLSConfig `yaml:"controller"`
	LLMProcessor  *ServiceMTLSConfig `yaml:"llm_processor"`
	RAGService    *ServiceMTLSConfig `yaml:"rag_service"`
	GitClient     *ServiceMTLSConfig `yaml:"git_client"`
	Database      *ServiceMTLSConfig `yaml:"database"`
	NephioBridge  *ServiceMTLSConfig `yaml:"nephio_bridge"`
	ORANAdaptor   *ServiceMTLSConfig `yaml:"oran_adaptor"`
	Monitoring    *ServiceMTLSConfig `yaml:"monitoring"`
	
	// TLS settings
	MinTLSVersion    string   `yaml:"min_tls_version"`
	MaxTLSVersion    string   `yaml:"max_tls_version"`
	CipherSuites     []string `yaml:"cipher_suites"`
	
	// Security settings
	AllowedClientCNs  []string `yaml:"allowed_client_cns"`
	AllowedClientOrgs []string `yaml:"allowed_client_orgs"`
	EnableHSTS        bool     `yaml:"enable_hsts"`
	HSTSMaxAge        int64    `yaml:"hsts_max_age"`
}

// ServiceMTLSConfig holds mTLS configuration for a specific service
type ServiceMTLSConfig struct {
	Enabled            bool     `yaml:"enabled"`
	ServiceName        string   `yaml:"service_name"`
	ServerCertPath     string   `yaml:"server_cert_path"`
	ServerKeyPath      string   `yaml:"server_key_path"`
	ClientCertPath     string   `yaml:"client_cert_path"`
	ClientKeyPath      string   `yaml:"client_key_path"`
	CACertPath         string   `yaml:"ca_cert_path"`
	ServerName         string   `yaml:"server_name"`
	Port               int      `yaml:"port"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify"`
	
	// Connection settings
	DialTimeout         time.Duration `yaml:"dial_timeout"`
	KeepAliveTimeout    time.Duration `yaml:"keep_alive_timeout"`
	MaxIdleConns        int          `yaml:"max_idle_conns"`
	MaxConnsPerHost     int          `yaml:"max_conns_per_host"`
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"`
	
	// Service-specific settings
	RequireClientCert   bool     `yaml:"require_client_cert"`
	AllowedClientCNs    []string `yaml:"allowed_client_cns"`
	AllowedClientOrgs   []string `yaml:"allowed_client_orgs"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MetricsAddr:          ":8080",
		ProbeAddr:            ":8081",
		EnableLeaderElection: false,

		LLMProcessorURL:     "http://llm-processor.default.svc.cluster.local:8080",
		LLMProcessorTimeout: 30 * time.Second,

		RAGAPIURLInternal: "http://rag-api.default.svc.cluster.local:5001",
		RAGAPIURLExternal: "http://localhost:5001",
		RAGAPITimeout:     30 * time.Second,

		GitBranch: "main",
		GitConcurrentPushLimit: 4,  // Default value

		WeaviateURL:   "http://weaviate.default.svc.cluster.local:8080",
		WeaviateIndex: "telecom_knowledge",

		OpenAIModel:          "gpt-4o-mini",
		OpenAIEmbeddingModel: "text-embedding-3-large",

		Namespace: "default",
		CRDPath:   "deployments/crds",

		MTLSConfig: DefaultMTLSConfig(),
	}
}

// DefaultMTLSConfig returns default mTLS configuration
func DefaultMTLSConfig() *MTLSConfig {
	baseDir := "/etc/nephoran/certs"
	
	return &MTLSConfig{
		Enabled:             false, // Disabled by default for backward compatibility
		RequireClientCerts:  true,
		TenantID:           "default",
		CAManagerEnabled:   true,
		AutoProvision:      false,
		PolicyTemplate:     "service-auth",
		CertificateBaseDir: baseDir,
		ValidityDuration:   24 * time.Hour,
		RenewalThreshold:   6 * time.Hour,
		RotationEnabled:    true,
		RotationInterval:   1 * time.Hour,
		
		Controller: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "nephoran-controller",
			ServerCertPath:     fmt.Sprintf("%s/default/nephoran-controller/tls.crt", baseDir),
			ServerKeyPath:      fmt.Sprintf("%s/default/nephoran-controller/tls.key", baseDir),
			ClientCertPath:     fmt.Sprintf("%s/default/nephoran-controller/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/nephoran-controller/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/nephoran-controller/ca.crt", baseDir),
			Port:               8443,
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       100,
			MaxConnsPerHost:    10,
			IdleConnTimeout:    90 * time.Second,
			RequireClientCert:  true,
		},
		
		LLMProcessor: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "llm-processor",
			ServerName:         "llm-processor.default.svc.cluster.local",
			ServerCertPath:     fmt.Sprintf("%s/default/llm-processor/tls.crt", baseDir),
			ServerKeyPath:      fmt.Sprintf("%s/default/llm-processor/tls.key", baseDir),
			ClientCertPath:     fmt.Sprintf("%s/default/llm-processor/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/llm-processor/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/llm-processor/ca.crt", baseDir),
			Port:               8443,
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       50,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    60 * time.Second,
			RequireClientCert:  true,
		},
		
		RAGService: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "rag-api",
			ServerName:         "rag-api.default.svc.cluster.local",
			ServerCertPath:     fmt.Sprintf("%s/default/rag-api/tls.crt", baseDir),
			ServerKeyPath:      fmt.Sprintf("%s/default/rag-api/tls.key", baseDir),
			ClientCertPath:     fmt.Sprintf("%s/default/rag-api/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/rag-api/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/rag-api/ca.crt", baseDir),
			Port:               5443,
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       50,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    60 * time.Second,
			RequireClientCert:  true,
		},
		
		GitClient: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "git-client",
			ClientCertPath:     fmt.Sprintf("%s/default/git-client/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/git-client/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/git-client/ca.crt", baseDir),
			DialTimeout:        30 * time.Second,
			MaxIdleConns:       20,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    120 * time.Second,
		},
		
		Database: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "database-client",
			ClientCertPath:     fmt.Sprintf("%s/default/database-client/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/database-client/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/database-client/ca.crt", baseDir),
			DialTimeout:        15 * time.Second,
			MaxIdleConns:       25,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    180 * time.Second,
		},
		
		NephioBridge: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "nephio-bridge",
			ServerName:         "nephio-bridge.default.svc.cluster.local",
			ServerCertPath:     fmt.Sprintf("%s/default/nephio-bridge/tls.crt", baseDir),
			ServerKeyPath:      fmt.Sprintf("%s/default/nephio-bridge/tls.key", baseDir),
			ClientCertPath:     fmt.Sprintf("%s/default/nephio-bridge/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/nephio-bridge/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/nephio-bridge/ca.crt", baseDir),
			Port:               8443,
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       30,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    90 * time.Second,
			RequireClientCert:  true,
		},
		
		ORANAdaptor: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "oran-adaptor",
			ServerName:         "oran-adaptor.default.svc.cluster.local",
			ServerCertPath:     fmt.Sprintf("%s/default/oran-adaptor/tls.crt", baseDir),
			ServerKeyPath:      fmt.Sprintf("%s/default/oran-adaptor/tls.key", baseDir),
			ClientCertPath:     fmt.Sprintf("%s/default/oran-adaptor/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/oran-adaptor/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/oran-adaptor/ca.crt", baseDir),
			Port:               8443,
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       30,
			MaxConnsPerHost:    5,
			IdleConnTimeout:    90 * time.Second,
			RequireClientCert:  true,
		},
		
		Monitoring: &ServiceMTLSConfig{
			Enabled:            false,
			ServiceName:        "monitoring-client",
			ClientCertPath:     fmt.Sprintf("%s/default/monitoring-client/client.crt", baseDir),
			ClientKeyPath:      fmt.Sprintf("%s/default/monitoring-client/client.key", baseDir),
			CACertPath:         fmt.Sprintf("%s/default/monitoring-client/ca.crt", baseDir),
			DialTimeout:        10 * time.Second,
			MaxIdleConns:       20,
			MaxConnsPerHost:    3,
			IdleConnTimeout:    60 * time.Second,
		},
		
		MinTLSVersion:     "1.2",
		MaxTLSVersion:     "1.3",
		EnableHSTS:        true,
		HSTSMaxAge:        31536000, // 1 year
		AllowedClientCNs:  []string{}, // Empty means allow all valid certificates
		AllowedClientOrgs: []string{}, // Empty means allow all valid certificates
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Override defaults with environment variables if they exist
	if val := os.Getenv("METRICS_ADDR"); val != "" {
		cfg.MetricsAddr = val
	}

	if val := os.Getenv("PROBE_ADDR"); val != "" {
		cfg.ProbeAddr = val
	}

	if val := os.Getenv("ENABLE_LEADER_ELECTION"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.EnableLeaderElection = b
		}
	}

	if val := os.Getenv("LLM_PROCESSOR_URL"); val != "" {
		cfg.LLMProcessorURL = val
	}

	if val := os.Getenv("LLM_PROCESSOR_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.LLMProcessorTimeout = d
		}
	}

	if val := os.Getenv("RAG_API_URL"); val != "" {
		cfg.RAGAPIURLInternal = val
	}

	if val := os.Getenv("RAG_API_URL_EXTERNAL"); val != "" {
		cfg.RAGAPIURLExternal = val
	}

	if val := os.Getenv("RAG_API_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.RAGAPITimeout = d
		}
	}

	if val := os.Getenv("GIT_REPO_URL"); val != "" {
		cfg.GitRepoURL = val
	}

	// Check for token file path first
	if val := os.Getenv("GIT_TOKEN_PATH"); val != "" {
		cfg.GitTokenPath = val
		// Try to read the token from file
		if tokenData, err := os.ReadFile(val); err == nil {
			cfg.GitToken = strings.TrimSpace(string(tokenData))
		}
		// If file read fails, GitToken will remain empty and fallback will occur
	}

	// Fallback to direct environment variable if no token loaded from file
	if cfg.GitToken == "" {
		if val := os.Getenv("GIT_TOKEN"); val != "" {
			cfg.GitToken = val
		}
	}

	if val := os.Getenv("GIT_BRANCH"); val != "" {
		cfg.GitBranch = val
	}

	if val := os.Getenv("GIT_CONCURRENT_PUSH_LIMIT"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil && limit > 0 {
			cfg.GitConcurrentPushLimit = limit
		}
		// Ignore parse errors, will use default
	}

	if val := os.Getenv("WEAVIATE_URL"); val != "" {
		cfg.WeaviateURL = val
	}

	if val := os.Getenv("WEAVIATE_INDEX"); val != "" {
		cfg.WeaviateIndex = val
	}

	if val := os.Getenv("OPENAI_API_KEY"); val != "" {
		cfg.OpenAIAPIKey = val
	}

	if val := os.Getenv("OPENAI_MODEL"); val != "" {
		cfg.OpenAIModel = val
	}

	if val := os.Getenv("OPENAI_EMBEDDING_MODEL"); val != "" {
		cfg.OpenAIEmbeddingModel = val
	}

	if val := os.Getenv("NAMESPACE"); val != "" {
		cfg.Namespace = val
	}

	if val := os.Getenv("CRD_PATH"); val != "" {
		cfg.CRDPath = val
	}

	// Load mTLS configuration from environment
	loadMTLSFromEnv(cfg)

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	var errors []string

	// Always required configuration
	if c.OpenAIAPIKey == "" {
		errors = append(errors, "OPENAI_API_KEY is required")
	}

	// Git features validation - enabled when Git integration is intended
	// Git features are considered enabled when GitRepoURL is configured or
	// when other Git-related config suggests Git usage
	if c.isGitFeatureEnabled() {
		if c.GitRepoURL == "" {
			errors = append(errors, "GIT_REPO_URL is required when Git features are enabled")
		}
	}

	// LLM processing validation - enabled when LLM processing is intended
	// LLM processing is considered enabled when LLMProcessorURL is configured
	if c.isLLMProcessingEnabled() {
		if c.LLMProcessorURL == "" {
			errors = append(errors, "LLM_PROCESSOR_URL is required when LLM processing is enabled")
		}
	}

	// RAG features validation - enabled when RAG features are intended
	// RAG features are considered enabled when RAG API URL is configured or
	// when RAG-related infrastructure is configured
	if c.isRAGFeatureEnabled() {
		if c.RAGAPIURLInternal == "" {
			errors = append(errors, "RAG_API_URL_INTERNAL is required when RAG features are enabled")
		}
	}

	// Return validation errors if any
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// isGitFeatureEnabled checks if Git features are enabled based on configuration
func (c *Config) isGitFeatureEnabled() bool {
	// Git features are enabled when GitRepoURL is set or when Git token is provided
	// (indicating intention to use Git even if URL is missing - validation error)
	return c.GitRepoURL != "" || c.GitToken != ""
}

// isLLMProcessingEnabled checks if LLM processing is enabled based on configuration
func (c *Config) isLLMProcessingEnabled() bool {
	// LLM processing is enabled when LLMProcessorURL is set
	// This follows the pattern seen in networkintent_constructor.go
	return c.LLMProcessorURL != ""
}

// isRAGFeatureEnabled checks if RAG features are enabled based on configuration
func (c *Config) isRAGFeatureEnabled() bool {
	// RAG features are enabled when RAG API URL is set or when Weaviate is configured
	// (indicating intention to use RAG even if internal URL is missing - validation error)
	return c.RAGAPIURLInternal != "" || c.RAGAPIURLExternal != "" || c.WeaviateURL != ""
}

// GetRAGAPIURL returns the appropriate RAG API URL based on environment
func (c *Config) GetRAGAPIURL(useInternal bool) string {
	if useInternal {
		return c.RAGAPIURLInternal
	}
	return c.RAGAPIURLExternal
}

// loadMTLSFromEnv loads mTLS configuration from environment variables
func loadMTLSFromEnv(cfg *Config) {
	if cfg.MTLSConfig == nil {
		return
	}

	// Global mTLS settings
	if val := os.Getenv("MTLS_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.Enabled = b
		}
	}

	if val := os.Getenv("MTLS_REQUIRE_CLIENT_CERTS"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.RequireClientCerts = b
		}
	}

	if val := os.Getenv("MTLS_TENANT_ID"); val != "" {
		cfg.MTLSConfig.TenantID = val
	}

	if val := os.Getenv("MTLS_CA_MANAGER_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.CAManagerEnabled = b
		}
	}

	if val := os.Getenv("MTLS_AUTO_PROVISION"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.AutoProvision = b
		}
	}

	if val := os.Getenv("MTLS_POLICY_TEMPLATE"); val != "" {
		cfg.MTLSConfig.PolicyTemplate = val
	}

	if val := os.Getenv("MTLS_CERTIFICATE_BASE_DIR"); val != "" {
		cfg.MTLSConfig.CertificateBaseDir = val
	}

	if val := os.Getenv("MTLS_VALIDITY_DURATION"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.MTLSConfig.ValidityDuration = d
		}
	}

	if val := os.Getenv("MTLS_RENEWAL_THRESHOLD"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.MTLSConfig.RenewalThreshold = d
		}
	}

	if val := os.Getenv("MTLS_ROTATION_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.RotationEnabled = b
		}
	}

	if val := os.Getenv("MTLS_ROTATION_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.MTLSConfig.RotationInterval = d
		}
	}

	// TLS version settings
	if val := os.Getenv("MTLS_MIN_TLS_VERSION"); val != "" {
		cfg.MTLSConfig.MinTLSVersion = val
	}

	if val := os.Getenv("MTLS_MAX_TLS_VERSION"); val != "" {
		cfg.MTLSConfig.MaxTLSVersion = val
	}

	// Security settings
	if val := os.Getenv("MTLS_ENABLE_HSTS"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.MTLSConfig.EnableHSTS = b
		}
	}

	if val := os.Getenv("MTLS_HSTS_MAX_AGE"); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.MTLSConfig.HSTSMaxAge = i
		}
	}

	if val := os.Getenv("MTLS_ALLOWED_CLIENT_CNS"); val != "" {
		cfg.MTLSConfig.AllowedClientCNs = strings.Split(val, ",")
	}

	if val := os.Getenv("MTLS_ALLOWED_CLIENT_ORGS"); val != "" {
		cfg.MTLSConfig.AllowedClientOrgs = strings.Split(val, ",")
	}

	// Service-specific settings - Controller
	loadServiceMTLSFromEnv(cfg.MTLSConfig.Controller, "CONTROLLER")

	// Service-specific settings - LLM Processor
	loadServiceMTLSFromEnv(cfg.MTLSConfig.LLMProcessor, "LLM_PROCESSOR")

	// Service-specific settings - RAG Service
	loadServiceMTLSFromEnv(cfg.MTLSConfig.RAGService, "RAG_SERVICE")

	// Service-specific settings - Git Client
	loadServiceMTLSFromEnv(cfg.MTLSConfig.GitClient, "GIT_CLIENT")

	// Service-specific settings - Database
	loadServiceMTLSFromEnv(cfg.MTLSConfig.Database, "DATABASE")

	// Service-specific settings - Nephio Bridge
	loadServiceMTLSFromEnv(cfg.MTLSConfig.NephioBridge, "NEPHIO_BRIDGE")

	// Service-specific settings - ORAN Adaptor
	loadServiceMTLSFromEnv(cfg.MTLSConfig.ORANAdaptor, "ORAN_ADAPTOR")

	// Service-specific settings - Monitoring
	loadServiceMTLSFromEnv(cfg.MTLSConfig.Monitoring, "MONITORING")
}

// loadServiceMTLSFromEnv loads mTLS configuration for a specific service
func loadServiceMTLSFromEnv(serviceCfg *ServiceMTLSConfig, prefix string) {
	if serviceCfg == nil {
		return
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_ENABLED", prefix)); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			serviceCfg.Enabled = b
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_SERVICE_NAME", prefix)); val != "" {
		serviceCfg.ServiceName = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_SERVER_CERT_PATH", prefix)); val != "" {
		serviceCfg.ServerCertPath = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_SERVER_KEY_PATH", prefix)); val != "" {
		serviceCfg.ServerKeyPath = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_CLIENT_CERT_PATH", prefix)); val != "" {
		serviceCfg.ClientCertPath = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_CLIENT_KEY_PATH", prefix)); val != "" {
		serviceCfg.ClientKeyPath = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_CA_CERT_PATH", prefix)); val != "" {
		serviceCfg.CACertPath = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_SERVER_NAME", prefix)); val != "" {
		serviceCfg.ServerName = val
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_PORT", prefix)); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			serviceCfg.Port = i
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_INSECURE_SKIP_VERIFY", prefix)); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			serviceCfg.InsecureSkipVerify = b
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_DIAL_TIMEOUT", prefix)); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			serviceCfg.DialTimeout = d
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_KEEP_ALIVE_TIMEOUT", prefix)); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			serviceCfg.KeepAliveTimeout = d
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_MAX_IDLE_CONNS", prefix)); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			serviceCfg.MaxIdleConns = i
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_MAX_CONNS_PER_HOST", prefix)); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			serviceCfg.MaxConnsPerHost = i
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_IDLE_CONN_TIMEOUT", prefix)); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			serviceCfg.IdleConnTimeout = d
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_REQUIRE_CLIENT_CERT", prefix)); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			serviceCfg.RequireClientCert = b
		}
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_ALLOWED_CLIENT_CNS", prefix)); val != "" {
		serviceCfg.AllowedClientCNs = strings.Split(val, ",")
	}

	if val := os.Getenv(fmt.Sprintf("MTLS_%s_ALLOWED_CLIENT_ORGS", prefix)); val != "" {
		serviceCfg.AllowedClientOrgs = strings.Split(val, ",")
	}
}

// ConfigProvider interface methods

// GetLLMProcessorURL returns the LLM processor URL
func (c *Config) GetLLMProcessorURL() string {
	return c.LLMProcessorURL
}

// GetLLMProcessorTimeout returns the LLM processor timeout
func (c *Config) GetLLMProcessorTimeout() time.Duration {
	return c.LLMProcessorTimeout
}

// GetGitRepoURL returns the Git repository URL
func (c *Config) GetGitRepoURL() string {
	return c.GitRepoURL
}

// GetGitToken returns the Git token
func (c *Config) GetGitToken() string {
	return c.GitToken
}

// GetGitBranch returns the Git branch
func (c *Config) GetGitBranch() string {
	return c.GitBranch
}

// GetWeaviateURL returns the Weaviate URL
func (c *Config) GetWeaviateURL() string {
	return c.WeaviateURL
}

// GetWeaviateIndex returns the Weaviate index name
func (c *Config) GetWeaviateIndex() string {
	return c.WeaviateIndex
}

// GetOpenAIAPIKey returns the OpenAI API key
func (c *Config) GetOpenAIAPIKey() string {
	return c.OpenAIAPIKey
}

// GetOpenAIModel returns the OpenAI model
func (c *Config) GetOpenAIModel() string {
	return c.OpenAIModel
}

// GetOpenAIEmbeddingModel returns the OpenAI embedding model
func (c *Config) GetOpenAIEmbeddingModel() string {
	return c.OpenAIEmbeddingModel
}

// GetNamespace returns the Kubernetes namespace
func (c *Config) GetNamespace() string {
	return c.Namespace
}

// Ensure Config implements interfaces.ConfigProvider
var _ interfaces.ConfigProvider = (*Config)(nil)

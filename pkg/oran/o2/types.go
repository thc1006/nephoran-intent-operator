package o2

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// Additional missing types from various files
type SLAMonitor struct {
	Thresholds       map[string]float64 `json:"thresholds"`
	CheckInterval    time.Duration      `json:"checkInterval"`
	AlertCallbacks   []func(Alarm)      `json:"-"`
	EvaluationWindow time.Duration      `json:"evaluationWindow"`
}

type EventProcessor struct {
	Processors map[string]func(interface{}) error `json:"-"`
	Buffer     []interface{}                     `json:"-"`
	BatchSize  int                               `json:"batchSize"`
}

type NotificationConfig struct {
	Enabled         bool                  `json:"enabled"`
	Channels        []NotificationChannel `json:"channels"`
	Retries         int                   `json:"retries"`
	RetryDelay      time.Duration         `json:"retryDelay"`
	BatchSize       int                   `json:"batchSize"`
	FlushPeriod     time.Duration         `json:"flushPeriod"`
	DeadLetterQueue bool                  `json:"deadLetterQueue"`
}

type NotificationChannel struct {
	Type        string                 `json:"type"`        // "webhook", "email", "slack", etc.
	Endpoint    string                 `json:"endpoint"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
}

type InventoryConfig struct {
	SyncInterval      time.Duration `json:"syncInterval"`
	CacheEnabled      bool          `json:"cacheEnabled"`
	CacheTTL          time.Duration `json:"cacheTtl"`
	MaxItems          int           `json:"maxItems"`
	CompressionLevel  int           `json:"compressionLevel"`
	EncryptionEnabled bool          `json:"encryptionEnabled"`
	BackupEnabled     bool          `json:"backupEnabled"`
	BackupInterval    time.Duration `json:"backupInterval"`
}

// Missing types from models package that are referenced but not defined
type DeploymentManagerFilter struct {
	Names       []string `json:"names,omitempty"`
	Types       []string `json:"types,omitempty"`
	Statuses    []string `json:"statuses,omitempty"`
	Providers   []string `json:"providers,omitempty"`
	Locations   []string `json:"locations,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Offset      int      `json:"offset,omitempty"`
}

type DeploymentManager struct {
	DeploymentManagerID string                 `json:"deploymentManagerId"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	Type                string                 `json:"type"`
	Endpoint            string                 `json:"endpoint"`
	Status              string                 `json:"status"`
	Capabilities        []string               `json:"capabilities,omitempty"`
	SupportedTypes      []string               `json:"supportedTypes,omitempty"`
	Configuration       map[string]interface{} `json:"configuration,omitempty"`
	Credentials         map[string]string      `json:"credentials,omitempty"`
	CreatedAt           time.Time              `json:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt"`
}

// Model request types that might be missing (these are already defined in helper_types.go so we don't redefine them)

type CreateSubscriptionRequest struct {
	Callback    string                 `json:"callback"`
	ConsumerID  string                 `json:"consumerId,omitempty"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	EventTypes  []string               `json:"eventTypes,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retryPolicy,omitempty"`
}

// Additional configuration type that may be missing
type MonitoringIntegrationConfig struct {
	PrometheusEnabled   bool          `json:"prometheusEnabled"`
	PrometheusEndpoint  string        `json:"prometheusEndpoint,omitempty"`
	GrafanaEnabled      bool          `json:"grafanaEnabled"`
	GrafanaEndpoint     string        `json:"grafanaEndpoint,omitempty"`
	AlertManagerEnabled bool          `json:"alertManagerEnabled"`
	AlertManagerURL     string        `json:"alertManagerUrl,omitempty"`
	MetricsInterval     time.Duration `json:"metricsInterval"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
	LogLevel            string        `json:"logLevel"`
	EnableDetailedLogs  bool          `json:"enableDetailedLogs"`
}

// Stub implementations for models package types
func init() {
	// Initialize any required model types or configurations here
}

// Additional missing service types
type InventoryManagementService struct {
	Enabled          bool          `json:"enabled"`
	SyncInterval     time.Duration `json:"syncInterval"`
	CacheExpiration  time.Duration `json:"cacheExpiration"`
	MaxConcurrentOps int           `json:"maxConcurrentOps"`
	RetryAttempts    int           `json:"retryAttempts"`
	CompressionLevel int           `json:"compressionLevel"`
}

type MonitoringIntegrations struct {
	Prometheus   *PrometheusIntegration   `json:"prometheus,omitempty"`
	Grafana      *GrafanaIntegration      `json:"grafana,omitempty"`
	AlertManager *AlertManagerIntegration `json:"alertManager,omitempty"`
	Jaeger       *JaegerIntegration       `json:"jaeger,omitempty"`
	Elastic      *ElasticIntegration      `json:"elastic,omitempty"`
}

type PrometheusIntegration struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	PushGateway     string `json:"pushGateway,omitempty"`
	ScrapeInterval  string `json:"scrapeInterval"`
	RetentionPeriod string `json:"retentionPeriod"`
}

type GrafanaIntegration struct {
	Enabled     bool     `json:"enabled"`
	Endpoint    string   `json:"endpoint"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	OrgID       int      `json:"orgId,omitempty"`
	Dashboards  []string `json:"dashboards,omitempty"`
}

type AlertManagerIntegration struct {
	Enabled     bool     `json:"enabled"`
	Endpoint    string   `json:"endpoint"`
	WebhookURL  string   `json:"webhookUrl,omitempty"`
	Receivers   []string `json:"receivers,omitempty"`
}

type JaegerIntegration struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Agent    string `json:"agent,omitempty"`
}

type ElasticIntegration struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Index    string `json:"index"`
}

type ComponentHealthStatus struct {
	Name            string                 `json:"name"`
	Healthy         bool                   `json:"healthy"`
	Status          string                 `json:"status"`
	Message         string                 `json:"message,omitempty"`
	LastChecked     time.Time              `json:"lastChecked"`
	Uptime          time.Duration          `json:"uptime"`
	Version         string                 `json:"version,omitempty"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	Dependencies    []string               `json:"dependencies,omitempty"`
	Endpoints       []string               `json:"endpoints,omitempty"`
	LastError       string                 `json:"lastError,omitempty"`
	ErrorCount      int                    `json:"errorCount"`
	ResponseTime    time.Duration          `json:"responseTime"`
	ResourceUsage   *ResourceUsage         `json:"resourceUsage,omitempty"`
}

type ResourceUsage struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskUsage   float64 `json:"diskUsage"`
	NetworkIO   int64   `json:"networkIO"`
}


// Additional types for storage and api
type CloudProviderConfig struct {
	ProviderID   string            `json:"providerId"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Region       string            `json:"region,omitempty"`
	Endpoint     string            `json:"endpoint,omitempty"`
	Credentials  map[string]string `json:"credentials,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Enabled      bool              `json:"enabled"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	HealthStatus string            `json:"healthStatus,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
}

type DeploymentTemplateFilter struct {
	Names      []string `json:"names,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Versions   []string `json:"versions,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

type DeploymentFilter struct {
	Names           []string `json:"names,omitempty"`
	Statuses        []string `json:"statuses,omitempty"`
	TemplateIDs     []string `json:"templateIds,omitempty"`
	ResourcePoolIDs []string `json:"resourcePoolIds,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
}

type RequestContext struct {
	UserID      string            `json:"userId"`
	TenantID    string            `json:"tenantId"`
	TraceID     string            `json:"traceId"`
	RequestID   string            `json:"requestId"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	IPAddress   string            `json:"ipAddress,omitempty"`
	UserAgent   string            `json:"userAgent,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
}

type ResourceState struct {
	State       string                 `json:"state"`
	Phase       string                 `json:"phase"`
	Reason      string                 `json:"reason,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Conditions  []ResourceCondition    `json:"conditions,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// ResourceCondition is already defined in helper_types.go - no need to redeclare

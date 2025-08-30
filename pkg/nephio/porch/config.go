/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package porch

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Config defines the configuration for Porch integration.

type Config struct {

	// Kubernetes configuration.

	KubernetesConfig *KubernetesConfig `json:"kubernetesConfig,omitempty"`

	// Porch service configuration.

	PorchConfig *PorchServiceConfig `json:"porchConfig,omitempty"`

	// Repository configurations.

	Repositories map[string]*RepositoryConfig `json:"repositories,omitempty"`

	// Function configurations.

	Functions *FunctionRegistryConfig `json:"functions,omitempty"`

	// Cluster configurations.

	Clusters map[string]*ClusterConfig `json:"clusters,omitempty"`

	// Policy configurations.

	Policies *PolicyConfig `json:"policies,omitempty"`

	// Observability configuration.

	Observability *ObservabilityConfig `json:"observability,omitempty"`

	// Security configuration.

	Security *SecurityConfig `json:"security,omitempty"`

	// Performance configuration.

	Performance *PerformanceConfig `json:"performance,omitempty"`
}

// KubernetesConfig defines Kubernetes client configuration.

type KubernetesConfig struct {

	// Kubeconfig file path.

	KubeconfigPath string `json:"kubeconfigPath,omitempty"`

	// Context to use from kubeconfig.

	Context string `json:"context,omitempty"`

	// Master URL override.

	MasterURL string `json:"masterUrl,omitempty"`

	// Namespace to operate in.

	Namespace string `json:"namespace,omitempty"`

	// QPS for the Kubernetes client.

	QPS float32 `json:"qps,omitempty"`

	// Burst for the Kubernetes client.

	Burst int `json:"burst,omitempty"`

	// Timeout for requests.

	Timeout time.Duration `json:"timeout,omitempty"`

	// User agent.

	UserAgent string `json:"userAgent,omitempty"`

	// Additional headers.

	Headers map[string]string `json:"headers,omitempty"`
}

// PorchServiceConfig defines Porch service specific configuration.

type PorchServiceConfig struct {

	// Porch API server endpoint.

	Endpoint string `json:"endpoint"`

	// API version.

	APIVersion string `json:"apiVersion,omitempty"`

	// Timeout for operations.

	Timeout time.Duration `json:"timeout,omitempty"`

	// Retry configuration.

	Retry *RetryConfig `json:"retry,omitempty"`

	// Circuit breaker configuration.

	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`

	// TLS configuration.

	TLS *TLSConfig `json:"tls,omitempty"`

	// Authentication configuration.

	Auth *AuthenticationConfig `json:"auth,omitempty"`

	// Rate limiting.

	RateLimit *RateLimitConfig `json:"rateLimit,omitempty"`

	// Connection pooling.

	ConnectionPool *ConnectionPoolConfig `json:"connectionPool,omitempty"`

	// Enable experimental features.

	ExperimentalFeatures []string `json:"experimentalFeatures,omitempty"`
}

// FunctionRegistryConfig defines KRM function registry configuration.

type FunctionRegistryConfig struct {

	// Default registry for functions.

	DefaultRegistry string `json:"defaultRegistry,omitempty"`

	// Function registries.

	Registries map[string]*FunctionRegistrySpec `json:"registries,omitempty"`

	// Function execution configuration.

	Execution *FunctionExecutionConfig `json:"execution,omitempty"`

	// Cache configuration for function images.

	Cache *FunctionCacheConfig `json:"cache,omitempty"`

	// Security settings for function execution.

	Security *FunctionSecurityConfig `json:"security,omitempty"`

	// Resource limits for function execution.

	ResourceLimits *FunctionResourceLimits `json:"resourceLimits,omitempty"`
}

// ClusterConfig defines target cluster configuration.

type ClusterConfig struct {

	// Display name for the cluster.

	Name string `json:"name"`

	// Cluster endpoint.

	Endpoint string `json:"endpoint"`

	// Kubeconfig for the cluster.

	KubeconfigPath string `json:"kubeconfigPath,omitempty"`

	// Context name in kubeconfig.

	Context string `json:"context,omitempty"`

	// Namespace to deploy to.

	Namespace string `json:"namespace,omitempty"`

	// Labels for cluster identification.

	Labels map[string]string `json:"labels,omitempty"`

	// Annotations for cluster metadata.

	Annotations map[string]string `json:"annotations,omitempty"`

	// Cluster capabilities.

	Capabilities []string `json:"capabilities,omitempty"`

	// Cluster region/zone information.

	Location *ClusterLocation `json:"location,omitempty"`

	// Network configuration.

	Network *ClusterNetworkConfig `json:"network,omitempty"`

	// Security policies.

	SecurityPolicies []string `json:"securityPolicies,omitempty"`

	// Resource quotas.

	ResourceQuotas map[string]string `json:"resourceQuotas,omitempty"`

	// Health check configuration.

	HealthCheck *ClusterHealthConfig `json:"healthCheck,omitempty"`
}

// PolicyConfig defines policy and workflow configurations.

type PolicyConfig struct {

	// Default approval workflow.

	DefaultWorkflow string `json:"defaultWorkflow,omitempty"`

	// Approval workflows by package type.

	Workflows map[string]*WorkflowConfig `json:"workflows,omitempty"`

	// Validation policies.

	Validation *ValidationPolicyConfig `json:"validation,omitempty"`

	// Security policies.

	Security *SecurityPolicyConfig `json:"security,omitempty"`

	// Compliance policies.

	Compliance *CompliancePolicyConfig `json:"compliance,omitempty"`

	// RBAC policies.

	RBAC *RBACPolicyConfig `json:"rbac,omitempty"`

	// Audit policies.

	Audit *AuditPolicyConfig `json:"audit,omitempty"`
}

// ObservabilityConfig defines observability configuration.

type ObservabilityConfig struct {

	// Metrics configuration.

	Metrics *MetricsConfig `json:"metrics,omitempty"`

	// Logging configuration.

	Logging *LoggingConfig `json:"logging,omitempty"`

	// Tracing configuration.

	Tracing *TracingConfig `json:"tracing,omitempty"`

	// Health check configuration.

	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`

	// Alerting configuration.

	Alerting *AlertingConfig `json:"alerting,omitempty"`
}

// SecurityConfig defines security configuration.

type SecurityConfig struct {

	// Authentication configuration.

	Authentication *SecurityAuthConfig `json:"authentication,omitempty"`

	// Authorization configuration.

	Authorization *AuthorizationConfig `json:"authorization,omitempty"`

	// Encryption configuration.

	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// Network security configuration.

	NetworkSecurity *NetworkSecurityConfig `json:"networkSecurity,omitempty"`

	// Certificate management.

	Certificates *CertificateConfig `json:"certificates,omitempty"`

	// Secret management.

	Secrets *SecretManagementConfig `json:"secrets,omitempty"`

	// Vulnerability scanning.

	Scanning *ScanningConfig `json:"scanning,omitempty"`

	// Compliance scanning.

	Compliance *ComplianceScanConfig `json:"compliance,omitempty"`
}

// PerformanceConfig defines performance configuration.

type PerformanceConfig struct {

	// Client configuration.

	Client *ClientPerformanceConfig `json:"client,omitempty"`

	// Server configuration.

	Server *ServerPerformanceConfig `json:"server,omitempty"`

	// Caching configuration.

	Caching *CachingConfig `json:"caching,omitempty"`

	// Connection pooling.

	ConnectionPooling *ConnectionPoolingConfig `json:"connectionPooling,omitempty"`

	// Resource optimization.

	ResourceOptimization *ResourceOptimizationConfig `json:"resourceOptimization,omitempty"`

	// Performance monitoring.

	Monitoring *PerformanceMonitoringConfig `json:"monitoring,omitempty"`
}

// Supporting configuration types.

// RetryConfig defines retry behavior.

type RetryConfig struct {
	MaxRetries int `json:"maxRetries,omitempty"`

	InitialDelay time.Duration `json:"initialDelay,omitempty"`

	MaxDelay time.Duration `json:"maxDelay,omitempty"`

	BackoffFactor float64 `json:"backoffFactor,omitempty"`

	Jitter bool `json:"jitter,omitempty"`

	RetriableErrors []string `json:"retriableErrors,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker behavior.

type CircuitBreakerConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	FailureThreshold int `json:"failureThreshold,omitempty"`

	SuccessThreshold int `json:"successThreshold,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	HalfOpenMaxCalls int `json:"halfOpenMaxCalls,omitempty"`

	OpenStateTimeout time.Duration `json:"openStateTimeout,omitempty"`
}

// TLSConfig defines TLS configuration.

type TLSConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	CertFile string `json:"certFile,omitempty"`

	KeyFile string `json:"keyFile,omitempty"`

	CAFile string `json:"caFile,omitempty"`

	ServerName string `json:"serverName,omitempty"`

	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	CipherSuites []string `json:"cipherSuites,omitempty"`

	MinVersion string `json:"minVersion,omitempty"`

	MaxVersion string `json:"maxVersion,omitempty"`
}

// AuthenticationConfig defines authentication configuration.

type AuthenticationConfig struct {
	Type string `json:"type"` // bearer, basic, oauth2, oidc

	Token string `json:"token,omitempty"`

	Username string `json:"username,omitempty"`

	Password string `json:"password,omitempty"`

	ClientID string `json:"clientId,omitempty"`

	ClientSecret string `json:"clientSecret,omitempty"`

	TokenURL string `json:"tokenUrl,omitempty"`

	AuthURL string `json:"authUrl,omitempty"`

	Scopes []string `json:"scopes,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// RateLimitConfig defines rate limiting configuration.

type RateLimitConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	RequestsPerSecond float64 `json:"requestsPerSecond,omitempty"`

	Burst int `json:"burst,omitempty"`

	TimeWindow time.Duration `json:"timeWindow,omitempty"`

	Strategy string `json:"strategy,omitempty"` // token_bucket, sliding_window

}

// ConnectionPoolConfig defines connection pool configuration.

type ConnectionPoolConfig struct {
	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	MaxOpenConns int `json:"maxOpenConns,omitempty"`

	ConnMaxLifetime time.Duration `json:"connMaxLifetime,omitempty"`

	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime,omitempty"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval,omitempty"`
}

// Function configuration types.

// FunctionRegistrySpec defines a function registry specification.

type FunctionRegistrySpec struct {
	Name string `json:"name"`

	URL string `json:"url"`

	Type string `json:"type"` // docker, oci, git

	Auth *AuthConfig `json:"auth,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// FunctionExecutionConfig defines function execution configuration.

type FunctionExecutionConfig struct {
	Runtime string `json:"runtime,omitempty"` // docker, containerd, cri-o

	DefaultTimeout time.Duration `json:"defaultTimeout,omitempty"`

	MaxConcurrent int `json:"maxConcurrent,omitempty"`

	EnableNetworking bool `json:"enableNetworking,omitempty"`

	AllowPrivileged bool `json:"allowPrivileged,omitempty"`

	EnvironmentVars map[string]string `json:"environmentVars,omitempty"`

	VolumeBinds []string `json:"volumeBinds,omitempty"`

	SecurityContext *FunctionSecurityContext `json:"securityContext,omitempty"`

	ResourceLimits *FunctionResourceLimits `json:"resourceLimits,omitempty"`
}

// FunctionCacheConfig defines function cache configuration.

type FunctionCacheConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Size string `json:"size,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`

	CleanupInterval time.Duration `json:"cleanupInterval,omitempty"`

	Strategy string `json:"strategy,omitempty"` // LRU, LFU, FIFO

}

// FunctionSecurityConfig defines function security configuration.

type FunctionSecurityConfig struct {
	RunAsNonRoot bool `json:"runAsNonRoot,omitempty"`

	ReadOnlyRootFilesystem bool `json:"readOnlyRootFilesystem,omitempty"`

	AllowedCapabilities []string `json:"allowedCapabilities,omitempty"`

	DroppedCapabilities []string `json:"droppedCapabilities,omitempty"`

	AllowedSysctls []string `json:"allowedSysctls,omitempty"`

	SELinuxOptions *SELinuxOptions `json:"seLinuxOptions,omitempty"`

	AppArmorProfile string `json:"appArmorProfile,omitempty"`

	SeccompProfile string `json:"seccompProfile,omitempty"`
}

// FunctionResourceLimits defines resource limits for functions.

type FunctionResourceLimits struct {
	CPU string `json:"cpu,omitempty"`

	Memory string `json:"memory,omitempty"`

	Storage string `json:"storage,omitempty"`

	PIDs int64 `json:"pids,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	FileSize int64 `json:"fileSize,omitempty"`

	OpenFiles int64 `json:"openFiles,omitempty"`
}

// FunctionSecurityContext defines security context for functions.

type FunctionSecurityContext struct {
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	FSGroup *int64 `json:"fsGroup,omitempty"`

	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	SupplementalGroups []int64 `json:"supplementalGroups,omitempty"`

	SELinuxOptions *SELinuxOptions `json:"seLinuxOptions,omitempty"`

	WindowsOptions *WindowsOptions `json:"windowsOptions,omitempty"`
}

// Cluster configuration types.

// ClusterLocation defines cluster location information.

type ClusterLocation struct {
	Region string `json:"region,omitempty"`

	Zone string `json:"zone,omitempty"`

	Country string `json:"country,omitempty"`

	City string `json:"city,omitempty"`

	Datacenter string `json:"datacenter,omitempty"`

	Latitude float64 `json:"latitude,omitempty"`

	Longitude float64 `json:"longitude,omitempty"`
}

// ClusterNetworkConfig defines cluster network configuration.

type ClusterNetworkConfig struct {
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	PodCIDR string `json:"podCIDR,omitempty"`

	DNSServers []string `json:"dnsServers,omitempty"`

	MTU int `json:"mtu,omitempty"`

	NetworkPolicy bool `json:"networkPolicy,omitempty"`

	ServiceMesh string `json:"serviceMesh,omitempty"`

	LoadBalancer string `json:"loadBalancer,omitempty"`

	IngressController string `json:"ingressController,omitempty"`
}

// ClusterHealthConfig defines cluster health check configuration.

type ClusterHealthConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Interval time.Duration `json:"interval,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	FailureThreshold int `json:"failureThreshold,omitempty"`

	SuccessThreshold int `json:"successThreshold,omitempty"`

	Endpoints []string `json:"endpoints,omitempty"`
}

// Policy configuration types.

// WorkflowConfig defines workflow configuration.

type WorkflowConfig struct {
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Stages []WorkflowStageConfig `json:"stages"`

	Triggers []WorkflowTriggerConfig `json:"triggers,omitempty"`

	Approvers []ApproverConfig `json:"approvers,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	Notifications []NotificationConfig `json:"notifications,omitempty"`
}

// WorkflowStageConfig defines workflow stage configuration.

type WorkflowStageConfig struct {
	Name string `json:"name"`

	Type WorkflowStageType `json:"type"`

	Conditions []WorkflowConditionConfig `json:"conditions,omitempty"`

	Actions []WorkflowActionConfig `json:"actions"`

	Approvers []ApproverConfig `json:"approvers,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	OnSuccess []WorkflowActionConfig `json:"onSuccess,omitempty"`

	OnFailure []WorkflowActionConfig `json:"onFailure,omitempty"`
}

// WorkflowTriggerConfig defines workflow trigger configuration.

type WorkflowTriggerConfig struct {
	Type string `json:"type"`

	Condition map[string]interface{} `json:"condition"`

	Schedule string `json:"schedule,omitempty"`

	Events []string `json:"events,omitempty"`
}

// WorkflowConditionConfig defines workflow condition configuration.

type WorkflowConditionConfig struct {
	Type string `json:"type"`

	Condition map[string]interface{} `json:"condition"`

	Operator string `json:"operator,omitempty"`

	Values []interface{} `json:"values,omitempty"`
}

// WorkflowActionConfig defines workflow action configuration.

type WorkflowActionConfig struct {
	Type string `json:"type"`

	Config map[string]interface{} `json:"config"`

	Timeout time.Duration `json:"timeout,omitempty"`

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// ApproverConfig defines approver configuration.

type ApproverConfig struct {
	Type string `json:"type"` // user, group, service

	Name string `json:"name"`

	Roles []string `json:"roles,omitempty"`

	Stages []string `json:"stages,omitempty"`

	Permissions []string `json:"permissions,omitempty"`

	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// NotificationConfig defines notification configuration.

type NotificationConfig struct {
	Type string `json:"type"` // email, slack, webhook

	Target string `json:"target"`

	Template string `json:"template,omitempty"`

	Events []string `json:"events,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// ValidationPolicyConfig defines validation policy configuration.

type ValidationPolicyConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	DefaultValidators []string `json:"defaultValidators,omitempty"`

	CustomValidators map[string]ValidatorConfig `json:"customValidators,omitempty"`

	Strictness string `json:"strictness,omitempty"` // strict, moderate, lenient

	FailOnWarning bool `json:"failOnWarning,omitempty"`
}

// ValidatorConfig defines validator configuration.

type ValidatorConfig struct {
	Image string `json:"image"`

	Config map[string]interface{} `json:"config,omitempty"`

	Resources *FunctionResourceLimits `json:"resources,omitempty"`

	Enabled bool `json:"enabled,omitempty"`

	Severity string `json:"severity,omitempty"`

	Categories []string `json:"categories,omitempty"`
}

// SecurityPolicyConfig defines security policy configuration.

type SecurityPolicyConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	RequiredSecurityScanning bool `json:"requiredSecurityScanning,omitempty"`

	AllowedRegistries []string `json:"allowedRegistries,omitempty"`

	BlockedImages []string `json:"blockedImages,omitempty"`

	RequireSignedImages bool `json:"requireSignedImages,omitempty"`

	MaxSeverityLevel string `json:"maxSeverityLevel,omitempty"`

	EnforcePodSecurityStandards bool `json:"enforcePodSecurityStandards,omitempty"`

	NetworkPolicyRequired bool `json:"networkPolicyRequired,omitempty"`
}

// CompliancePolicyConfig defines compliance policy configuration.

type CompliancePolicyConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Frameworks []string `json:"frameworks,omitempty"` // CIS, NIST, SOC2, etc.

	Standards []ComplianceStandardConfig `json:"standards,omitempty"`

	Reporting *ComplianceReportingConfig `json:"reporting,omitempty"`

	Remediation *ComplianceRemediationConfig `json:"remediation,omitempty"`
}

// ComplianceStandardConfig defines compliance standard configuration.

type ComplianceStandardConfig struct {
	Name string `json:"name"`

	Version string `json:"version"`

	Controls []string `json:"controls,omitempty"`

	Severity string `json:"severity,omitempty"`

	Required bool `json:"required,omitempty"`

	Exemptions []string `json:"exemptions,omitempty"`
}

// ComplianceReportingConfig defines compliance reporting configuration.

type ComplianceReportingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Format []string `json:"format,omitempty"` // json, xml, pdf

	Schedule string `json:"schedule,omitempty"`

	Recipients []string `json:"recipients,omitempty"`

	Storage *StorageConfig `json:"storage,omitempty"`
}

// ComplianceRemediationConfig defines compliance remediation configuration.

type ComplianceRemediationConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	AutoRemediate bool `json:"autoRemediate,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	MaxAttempts int `json:"maxAttempts,omitempty"`

	NotifyOnFailure bool `json:"notifyOnFailure,omitempty"`
}

// RBACPolicyConfig defines RBAC policy configuration.

type RBACPolicyConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	DefaultRole string `json:"defaultRole,omitempty"`

	Roles map[string]RoleConfig `json:"roles,omitempty"`

	RoleBindings []RoleBindingConfig `json:"roleBindings,omitempty"`

	ServiceAccounts []ServiceAccountConfig `json:"serviceAccounts,omitempty"`
}

// RoleConfig defines role configuration.

type RoleConfig struct {
	Name string `json:"name"`

	Rules []PolicyRuleConfig `json:"rules"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
}

// PolicyRuleConfig defines policy rule configuration.

type PolicyRuleConfig struct {
	APIGroups []string `json:"apiGroups,omitempty"`

	Resources []string `json:"resources,omitempty"`

	ResourceNames []string `json:"resourceNames,omitempty"`

	Verbs []string `json:"verbs"`

	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
}

// RoleBindingConfig defines role binding configuration.

type RoleBindingConfig struct {
	Name string `json:"name"`

	RoleRef RoleRefConfig `json:"roleRef"`

	Subjects []SubjectConfig `json:"subjects"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
}

// RoleRefConfig defines role reference configuration.

type RoleRefConfig struct {
	APIGroup string `json:"apiGroup"`

	Kind string `json:"kind"`

	Name string `json:"name"`
}

// SubjectConfig defines subject configuration.

type SubjectConfig struct {
	Kind string `json:"kind"`

	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`

	APIGroup string `json:"apiGroup,omitempty"`
}

// ServiceAccountConfig defines service account configuration.

type ServiceAccountConfig struct {
	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Secrets []string `json:"secrets,omitempty"`

	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

// AuditPolicyConfig defines audit policy configuration.

type AuditPolicyConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Level string `json:"level,omitempty"` // Metadata, Request, RequestResponse

	Backend string `json:"backend,omitempty"` // file, webhook, elasticsearch

	Rules []AuditRuleConfig `json:"rules,omitempty"`

	Retention *AuditRetentionConfig `json:"retention,omitempty"`

	Destination *AuditDestinationConfig `json:"destination,omitempty"`
}

// AuditRuleConfig defines audit rule configuration.

type AuditRuleConfig struct {
	Level string `json:"level"`

	Namespaces []string `json:"namespaces,omitempty"`

	Verbs []string `json:"verbs,omitempty"`

	Resources []string `json:"resources,omitempty"`

	Users []string `json:"users,omitempty"`

	Groups []string `json:"groups,omitempty"`

	ServiceAccounts []string `json:"serviceAccounts,omitempty"`
}

// AuditRetentionConfig defines audit retention configuration.

type AuditRetentionConfig struct {
	Days int `json:"days,omitempty"`

	MaxSize string `json:"maxSize,omitempty"`

	MaxFiles int `json:"maxFiles,omitempty"`

	Compress bool `json:"compress,omitempty"`
}

// AuditDestinationConfig defines audit destination configuration.

type AuditDestinationConfig struct {
	Type string `json:"type"` // file, webhook, s3, elasticsearch

	Config map[string]interface{} `json:"config"`

	Format string `json:"format,omitempty"`

	BatchSize int `json:"batchSize,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`
}

// Observability configuration types.

// MetricsConfig defines metrics configuration.

type MetricsConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"` // prometheus, datadog, newrelic

	Endpoint string `json:"endpoint,omitempty"`

	Namespace string `json:"namespace,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Collectors []MetricsCollectorConfig `json:"collectors,omitempty"`

	Exporters []MetricsExporterConfig `json:"exporters,omitempty"`
}

// MetricsCollectorConfig defines metrics collector configuration.

type MetricsCollectorConfig struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Config map[string]interface{} `json:"config,omitempty"`

	Interval time.Duration `json:"interval,omitempty"`

	Enabled bool `json:"enabled,omitempty"`
}

// MetricsExporterConfig defines metrics exporter configuration.

type MetricsExporterConfig struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Endpoint string `json:"endpoint"`

	Config map[string]interface{} `json:"config,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	Enabled bool `json:"enabled,omitempty"`
}

// LoggingConfig defines logging configuration.

type LoggingConfig struct {
	Level string `json:"level,omitempty"`

	Format string `json:"format,omitempty"` // json, text

	Output []string `json:"output,omitempty"` // stdout, file, syslog

	File *LogFileConfig `json:"file,omitempty"`

	Fields map[string]interface{} `json:"fields,omitempty"`

	Filters []LogFilterConfig `json:"filters,omitempty"`

	Processors []LogProcessorConfig `json:"processors,omitempty"`
}

// LogFileConfig defines log file configuration.

type LogFileConfig struct {
	Path string `json:"path"`

	MaxSize string `json:"maxSize,omitempty"`

	MaxAge int `json:"maxAge,omitempty"`

	MaxBackups int `json:"maxBackups,omitempty"`

	Compress bool `json:"compress,omitempty"`
}

// LogFilterConfig defines log filter configuration.

type LogFilterConfig struct {
	Type string `json:"type"`

	Config map[string]interface{} `json:"config"`
}

// LogProcessorConfig defines log processor configuration.

type LogProcessorConfig struct {
	Type string `json:"type"`

	Config map[string]interface{} `json:"config"`
}

// TracingConfig defines tracing configuration.

type TracingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"` // jaeger, zipkin, otlp

	Endpoint string `json:"endpoint,omitempty"`

	SamplingRate float64 `json:"samplingRate,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	Exporters []TracingExporterConfig `json:"exporters,omitempty"`
}

// TracingExporterConfig defines tracing exporter configuration.

type TracingExporterConfig struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Endpoint string `json:"endpoint"`

	Config map[string]interface{} `json:"config,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	Enabled bool `json:"enabled,omitempty"`
}

// HealthCheckConfig defines health check configuration.

type HealthCheckConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Port int `json:"port,omitempty"`

	Path string `json:"path,omitempty"`

	Interval time.Duration `json:"interval,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	HealthyThreshold int `json:"healthyThreshold,omitempty"`

	UnhealthyThreshold int `json:"unhealthyThreshold,omitempty"`

	Checks []HealthCheckDefinition `json:"checks,omitempty"`
}

// HealthCheckDefinition defines individual health check.

type HealthCheckDefinition struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Config map[string]interface{} `json:"config,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	Critical bool `json:"critical,omitempty"`
}

// AlertingConfig defines alerting configuration.

type AlertingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"` // alertmanager, pagerduty

	Receivers []AlertReceiverConfig `json:"receivers,omitempty"`

	Routes []AlertRouteConfig `json:"routes,omitempty"`

	Rules []AlertRuleConfig `json:"rules,omitempty"`

	Templates map[string]string `json:"templates,omitempty"`
}

// AlertReceiverConfig defines alert receiver configuration.

type AlertReceiverConfig struct {
	Name string `json:"name"`

	Type string `json:"type"` // email, slack, webhook, pagerduty

	Config map[string]interface{} `json:"config"`

	Enabled bool `json:"enabled,omitempty"`
}

// AlertRouteConfig defines alert routing configuration.

type AlertRouteConfig struct {
	Match map[string]string `json:"match,omitempty"`

	MatchRE map[string]string `json:"matchRE,omitempty"`

	Receiver string `json:"receiver"`

	GroupBy []string `json:"groupBy,omitempty"`

	GroupWait time.Duration `json:"groupWait,omitempty"`

	GroupInterval time.Duration `json:"groupInterval,omitempty"`

	RepeatInterval time.Duration `json:"repeatInterval,omitempty"`
}

// AlertRuleConfig defines alert rule configuration.

type AlertRuleConfig struct {
	Name string `json:"name"`

	Query string `json:"query"`

	For time.Duration `json:"for,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Severity string `json:"severity,omitempty"`
}

// Security configuration types.

// SecurityAuthConfig defines security authentication configuration.

type SecurityAuthConfig struct {
	Methods []string `json:"methods,omitempty"` // cert, token, oidc

	OIDC *OIDCConfig `json:"oidc,omitempty"`

	Certificate *CertAuthConfig `json:"certificate,omitempty"`

	Token *TokenAuthConfig `json:"token,omitempty"`

	LDAP *LDAPAuthConfig `json:"ldap,omitempty"`
}

// OIDCConfig defines OIDC configuration.

type OIDCConfig struct {
	IssuerURL string `json:"issuerUrl"`

	ClientID string `json:"clientId"`

	ClientSecret string `json:"clientSecret"`

	RedirectURL string `json:"redirectUrl,omitempty"`

	Scopes []string `json:"scopes,omitempty"`

	ClaimsMapping map[string]string `json:"claimsMapping,omitempty"`
}

// CertAuthConfig defines certificate authentication configuration.

type CertAuthConfig struct {
	CAFile string `json:"caFile"`

	AllowedCNs []string `json:"allowedCNs,omitempty"`

	AllowedOUs []string `json:"allowedOUs,omitempty"`

	CRLFile string `json:"crlFile,omitempty"`

	VerifyChain bool `json:"verifyChain,omitempty"`
}

// TokenAuthConfig defines token authentication configuration.

type TokenAuthConfig struct {
	Type string `json:"type"` // jwt, opaque

	SigningKey string `json:"signingKey,omitempty"`

	SigningMethod string `json:"signingMethod,omitempty"`

	Issuer string `json:"issuer,omitempty"`

	Audience string `json:"audience,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`
}

// LDAPAuthConfig defines LDAP authentication configuration.

type LDAPAuthConfig struct {
	Host string `json:"host"`

	Port int `json:"port,omitempty"`

	UseSSL bool `json:"useSSL,omitempty"`

	BindDN string `json:"bindDN"`

	BindPassword string `json:"bindPassword"`

	BaseDN string `json:"baseDN"`

	UserFilter string `json:"userFilter"`

	GroupFilter string `json:"groupFilter,omitempty"`
}

// AuthorizationConfig defines authorization configuration.

type AuthorizationConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Mode string `json:"mode,omitempty"` // rbac, abac, webhook

	RBAC *RBACAuthzConfig `json:"rbac,omitempty"`

	ABAC *ABACAuthzConfig `json:"abac,omitempty"`

	Webhook *WebhookAuthzConfig `json:"webhook,omitempty"`
}

// RBACAuthzConfig defines RBAC authorization configuration.

type RBACAuthzConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	// Additional RBAC specific configuration.

}

// ABACAuthzConfig defines ABAC authorization configuration.

type ABACAuthzConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	PolicyFile string `json:"policyFile"`
}

// WebhookAuthzConfig defines webhook authorization configuration.

type WebhookAuthzConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	URL string `json:"url"`

	Timeout time.Duration `json:"timeout,omitempty"`
}

// EncryptionConfig defines encryption configuration.

type EncryptionConfig struct {
	AtRest *EncryptionAtRestConfig `json:"atRest,omitempty"`

	InTransit *EncryptionInTransitConfig `json:"inTransit,omitempty"`
}

// EncryptionAtRestConfig defines encryption at rest configuration.

type EncryptionAtRestConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"` // kms, vault, local

	KeyID string `json:"keyId,omitempty"`

	Algorithm string `json:"algorithm,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// EncryptionInTransitConfig defines encryption in transit configuration.

type EncryptionInTransitConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	MinTLSVersion string `json:"minTlsVersion,omitempty"`

	CipherSuites []string `json:"cipherSuites,omitempty"`

	Certificates *CertificateConfig `json:"certificates,omitempty"`
}

// NetworkSecurityConfig defines network security configuration.

type NetworkSecurityConfig struct {
	NetworkPolicies bool `json:"networkPolicies,omitempty"`

	ServiceMesh *ServiceMeshSecurityConfig `json:"serviceMesh,omitempty"`

	Firewall *FirewallConfig `json:"firewall,omitempty"`

	DDoSProtection *DDoSProtectionConfig `json:"ddosProtection,omitempty"`
}

// ServiceMeshSecurityConfig defines service mesh security configuration.

type ServiceMeshSecurityConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Type string `json:"type,omitempty"` // istio, linkerd, consul

	MTLS *mTLSConfig `json:"mtls,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// mTLSConfig defines mutual TLS configuration.

type mTLSConfig struct {
	Mode string `json:"mode,omitempty"` // strict, permissive

	AutoRotation bool `json:"autoRotation,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`
}

// FirewallConfig defines firewall configuration.

type FirewallConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Rules []FirewallRule `json:"rules,omitempty"`
}

// FirewallRule defines a firewall rule.

type FirewallRule struct {
	Name string `json:"name"`

	Direction string `json:"direction"` // ingress, egress

	Action string `json:"action"` // allow, deny

	Ports []string `json:"ports,omitempty"`

	Protocols []string `json:"protocols,omitempty"`

	Sources []string `json:"sources,omitempty"`

	Targets []string `json:"targets,omitempty"`
}

// DDoSProtectionConfig defines DDoS protection configuration.

type DDoSProtectionConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"`

	RateLimits []RateLimit `json:"rateLimits,omitempty"`

	BlacklistIPs []string `json:"blacklistIPs,omitempty"`

	WhitelistIPs []string `json:"whitelistIPs,omitempty"`

	DetectionThreshold int `json:"detectionThreshold,omitempty"`

	MitigationTimeout time.Duration `json:"mitigationTimeout,omitempty"`
}

// RateLimit defines rate limiting configuration.

type RateLimit struct {
	Path string `json:"path,omitempty"`

	Method string `json:"method,omitempty"`

	Limit int `json:"limit"`

	Window time.Duration `json:"window"`

	BurstLimit int `json:"burstLimit,omitempty"`
}

// CertificateConfig defines certificate configuration.

type CertificateConfig struct {
	CA *CertificateAuthority `json:"ca,omitempty"`

	AutoRotation bool `json:"autoRotation,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`

	Algorithms []string `json:"algorithms,omitempty"`

	KeySizes []int `json:"keySizes,omitempty"`
}

// CertificateAuthority defines CA configuration.

type CertificateAuthority struct {
	Type string `json:"type"` // internal, external, vault

	Config map[string]interface{} `json:"config,omitempty"`

	CertFile string `json:"certFile,omitempty"`

	KeyFile string `json:"keyFile,omitempty"`

	Issuer string `json:"issuer,omitempty"`
}

// SecretManagementConfig defines secret management configuration.

type SecretManagementConfig struct {
	Provider string `json:"provider"` // kubernetes, vault, aws-secrets-manager

	Config map[string]interface{} `json:"config,omitempty"`

	Encryption *SecretEncryptionConfig `json:"encryption,omitempty"`

	Rotation *SecretRotationConfig `json:"rotation,omitempty"`

	Access *SecretAccessConfig `json:"access,omitempty"`
}

// SecretEncryptionConfig defines secret encryption configuration.

type SecretEncryptionConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Algorithm string `json:"algorithm,omitempty"`

	KeyID string `json:"keyId,omitempty"`
}

// SecretRotationConfig defines secret rotation configuration.

type SecretRotationConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Interval time.Duration `json:"interval,omitempty"`

	Automatic bool `json:"automatic,omitempty"`
}

// SecretAccessConfig defines secret access configuration.

type SecretAccessConfig struct {
	Audit bool `json:"audit,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`

	MaxUses int `json:"maxUses,omitempty"`

	IPRestrict []string `json:"ipRestrict,omitempty"`
}

// ScanningConfig defines vulnerability scanning configuration.

type ScanningConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Provider string `json:"provider,omitempty"` // trivy, clair, snyk

	Schedule string `json:"schedule,omitempty"`

	Registries []string `json:"registries,omitempty"`

	Severities []string `json:"severities,omitempty"`

	FailOnSeverity string `json:"failOnSeverity,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// ComplianceScanConfig defines compliance scanning configuration.

type ComplianceScanConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Frameworks []string `json:"frameworks,omitempty"`

	Schedule string `json:"schedule,omitempty"`

	Benchmarks []ComplianceBenchmark `json:"benchmarks,omitempty"`
}

// ComplianceBenchmark defines compliance benchmark.

type ComplianceBenchmark struct {
	Name string `json:"name"`

	Version string `json:"version"`

	Controls []string `json:"controls,omitempty"`

	Enabled bool `json:"enabled,omitempty"`
}

// Performance configuration types.

// ClientPerformanceConfig defines client performance configuration.

type ClientPerformanceConfig struct {
	MaxConnections int `json:"maxConnections,omitempty"`

	ConnectionTimeout time.Duration `json:"connectionTimeout,omitempty"`

	RequestTimeout time.Duration `json:"requestTimeout,omitempty"`

	IdleConnTimeout time.Duration `json:"idleConnTimeout,omitempty"`

	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	MaxIdleConnsPerHost int `json:"maxIdleConnsPerHost,omitempty"`

	DisableKeepAlives bool `json:"disableKeepAlives,omitempty"`

	DisableCompression bool `json:"disableCompression,omitempty"`
}

// ServerPerformanceConfig defines server performance configuration.

type ServerPerformanceConfig struct {
	ReadTimeout time.Duration `json:"readTimeout,omitempty"`

	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout,omitempty"`

	WriteTimeout time.Duration `json:"writeTimeout,omitempty"`

	IdleTimeout time.Duration `json:"idleTimeout,omitempty"`

	MaxHeaderBytes int `json:"maxHeaderBytes,omitempty"`

	MaxRequestSize int64 `json:"maxRequestSize,omitempty"`

	WorkerPoolSize int `json:"workerPoolSize,omitempty"`
}

// CachingConfig defines caching configuration.

type CachingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Layers []CacheLayerConfig `json:"layers,omitempty"`

	Strategies []CacheStrategyConfig `json:"strategies,omitempty"`

	Invalidation *CacheInvalidationConfig `json:"invalidation,omitempty"`
}

// CacheLayerConfig defines cache layer configuration.

type CacheLayerConfig struct {
	Name string `json:"name"`

	Type string `json:"type"` // memory, redis, memcached

	Size string `json:"size,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`

	Enabled bool `json:"enabled,omitempty"`
}

// CacheStrategyConfig defines cache strategy configuration.

type CacheStrategyConfig struct {
	Name string `json:"name"`

	Type string `json:"type"` // lru, lfu, ttl

	Resources []string `json:"resources,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// CacheInvalidationConfig defines cache invalidation configuration.

type CacheInvalidationConfig struct {
	Strategy string `json:"strategy"` // manual, automatic, time-based

	Events []string `json:"events,omitempty"`

	TTL time.Duration `json:"ttl,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`
}

// ConnectionPoolingConfig defines connection pooling configuration.

type ConnectionPoolingConfig struct {
	Database *DatabasePoolConfig `json:"database,omitempty"`

	HTTP *HTTPPoolConfig `json:"http,omitempty"`

	GRPC *GRPCPoolConfig `json:"grpc,omitempty"`
}

// DatabasePoolConfig defines database connection pool configuration.

type DatabasePoolConfig struct {
	MaxOpenConns int `json:"maxOpenConns,omitempty"`

	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	ConnMaxLifetime time.Duration `json:"connMaxLifetime,omitempty"`

	ConnMaxIdleTime time.Duration `json:"connMaxIdleTime,omitempty"`
}

// HTTPPoolConfig defines HTTP connection pool configuration.

type HTTPPoolConfig struct {
	MaxConnsPerHost int `json:"maxConnsPerHost,omitempty"`

	MaxIdleConns int `json:"maxIdleConns,omitempty"`

	IdleConnTimeout time.Duration `json:"idleConnTimeout,omitempty"`

	DialTimeout time.Duration `json:"dialTimeout,omitempty"`

	KeepAlive time.Duration `json:"keepAlive,omitempty"`
}

// GRPCPoolConfig defines gRPC connection pool configuration.

type GRPCPoolConfig struct {
	MaxConnections int `json:"maxConnections,omitempty"`

	IdleTimeout time.Duration `json:"idleTimeout,omitempty"`

	MaxAge time.Duration `json:"maxAge,omitempty"`

	KeepAlive time.Duration `json:"keepAlive,omitempty"`

	KeepAliveTimeout time.Duration `json:"keepAliveTimeout,omitempty"`
}

// ResourceOptimizationConfig defines resource optimization configuration.

type ResourceOptimizationConfig struct {
	CPU *CPUOptimizationConfig `json:"cpu,omitempty"`

	Memory *MemoryOptimizationConfig `json:"memory,omitempty"`

	Disk *DiskOptimizationConfig `json:"disk,omitempty"`

	Network *NetworkOptimizationConfig `json:"network,omitempty"`
}

// CPUOptimizationConfig defines CPU optimization configuration.

type CPUOptimizationConfig struct {
	MaxWorkers int `json:"maxWorkers,omitempty"`

	GCPercent int `json:"gcPercent,omitempty"`

	MaxProcs int `json:"maxProcs,omitempty"`

	Affinity []int `json:"affinity,omitempty"`

	Scaling *AutoScalingConfig `json:"scaling,omitempty"`
}

// MemoryOptimizationConfig defines memory optimization configuration.

type MemoryOptimizationConfig struct {
	MaxHeapSize string `json:"maxHeapSize,omitempty"`

	GCPolicy string `json:"gcPolicy,omitempty"`

	BufferPools bool `json:"bufferPools,omitempty"`

	MemoryMapping bool `json:"memoryMapping,omitempty"`
}

// DiskOptimizationConfig defines disk optimization configuration.

type DiskOptimizationConfig struct {
	BufferSize int `json:"bufferSize,omitempty"`

	SyncWrites bool `json:"syncWrites,omitempty"`

	Compression bool `json:"compression,omitempty"`

	ReadAhead int `json:"readAhead,omitempty"`

	WriteCoalesce bool `json:"writeCoalesce,omitempty"`
}

// NetworkOptimizationConfig defines network optimization configuration.

type NetworkOptimizationConfig struct {
	BufferSizes *NetworkBufferConfig `json:"bufferSizes,omitempty"`

	Compression bool `json:"compression,omitempty"`

	KeepAlive bool `json:"keepAlive,omitempty"`

	NoDelay bool `json:"noDelay,omitempty"`

	Multiplexing bool `json:"multiplexing,omitempty"`
}

// NetworkBufferConfig defines network buffer configuration.

type NetworkBufferConfig struct {
	Send int `json:"send,omitempty"`

	Receive int `json:"receive,omitempty"`
}

// AutoScalingConfig defines auto-scaling configuration.

type AutoScalingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	MinWorkers int `json:"minWorkers,omitempty"`

	MaxWorkers int `json:"maxWorkers,omitempty"`

	TargetCPU float64 `json:"targetCPU,omitempty"`

	TargetMemory float64 `json:"targetMemory,omitempty"`
}

// PerformanceMonitoringConfig defines performance monitoring configuration.

type PerformanceMonitoringConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Profiling *ProfilingConfig `json:"profiling,omitempty"`

	Benchmarks *BenchmarkConfig `json:"benchmarks,omitempty"`

	Alerts []PerformanceAlertConfig `json:"alerts,omitempty"`
}

// ProfilingConfig defines profiling configuration.

type ProfilingConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	CPU bool `json:"cpu,omitempty"`

	Memory bool `json:"memory,omitempty"`

	Goroutine bool `json:"goroutine,omitempty"`

	Block bool `json:"block,omitempty"`

	Mutex bool `json:"mutex,omitempty"`

	Duration time.Duration `json:"duration,omitempty"`
}

// BenchmarkConfig defines benchmark configuration.

type BenchmarkConfig struct {
	Enabled bool `json:"enabled,omitempty"`

	Schedule string `json:"schedule,omitempty"`

	Duration time.Duration `json:"duration,omitempty"`

	Scenarios []string `json:"scenarios,omitempty"`
}

// PerformanceAlertConfig defines performance alert configuration.

type PerformanceAlertConfig struct {
	Name string `json:"name"`

	Metric string `json:"metric"`

	Threshold interface{} `json:"threshold"`

	Operator string `json:"operator"` // gt, lt, eq, ne

	Duration time.Duration `json:"duration,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

// Utility configuration types.

// StorageConfig defines storage configuration.

type StorageConfig struct {
	Type string `json:"type"` // file, s3, gcs, azure

	Path string `json:"path,omitempty"`

	Bucket string `json:"bucket,omitempty"`

	Prefix string `json:"prefix,omitempty"`

	Config map[string]interface{} `json:"config,omitempty"`

	Encryption *EncryptionAtRestConfig `json:"encryption,omitempty"`
}

// SELinuxOptions defines SELinux options.

type SELinuxOptions struct {
	User string `json:"user,omitempty"`

	Role string `json:"role,omitempty"`

	Type string `json:"type,omitempty"`

	Level string `json:"level,omitempty"`
}

// WindowsOptions defines Windows-specific options.

type WindowsOptions struct {
	GMSACredentialSpecName string `json:"gmsaCredentialSpecName,omitempty"`

	GMSACredentialSpec string `json:"gmsaCredentialSpec,omitempty"`

	RunAsUserName string `json:"runAsUserName,omitempty"`

	HostProcess *bool `json:"hostProcess,omitempty"`
}

// ConfigBuilder provides fluent API for building configurations.

type ConfigBuilder struct {
	config *Config

	logger logr.Logger
}

// NewConfigBuilder creates a new configuration builder.

func NewConfigBuilder() *ConfigBuilder {

	return &ConfigBuilder{

		config: &Config{

			Repositories: make(map[string]*RepositoryConfig),

			Clusters: make(map[string]*ClusterConfig),
		},

		logger: log.Log.WithName("porch-config"),
	}

}

// WithKubernetesConfig sets Kubernetes configuration.

func (cb *ConfigBuilder) WithKubernetesConfig(config *KubernetesConfig) *ConfigBuilder {

	cb.config.KubernetesConfig = config

	return cb

}

// WithPorchConfig sets Porch service configuration.

func (cb *ConfigBuilder) WithPorchConfig(config *PorchServiceConfig) *ConfigBuilder {

	cb.config.PorchConfig = config

	return cb

}

// AddRepository adds a repository configuration.

func (cb *ConfigBuilder) AddRepository(name string, config *RepositoryConfig) *ConfigBuilder {

	cb.config.Repositories[name] = config

	return cb

}

// AddCluster adds a cluster configuration.

func (cb *ConfigBuilder) AddCluster(name string, config *ClusterConfig) *ConfigBuilder {

	cb.config.Clusters[name] = config

	return cb

}

// WithFunctions sets function registry configuration.

func (cb *ConfigBuilder) WithFunctions(config *FunctionRegistryConfig) *ConfigBuilder {

	cb.config.Functions = config

	return cb

}

// WithPolicies sets policy configuration.

func (cb *ConfigBuilder) WithPolicies(config *PolicyConfig) *ConfigBuilder {

	cb.config.Policies = config

	return cb

}

// WithObservability sets observability configuration.

func (cb *ConfigBuilder) WithObservability(config *ObservabilityConfig) *ConfigBuilder {

	cb.config.Observability = config

	return cb

}

// WithSecurity sets security configuration.

func (cb *ConfigBuilder) WithSecurity(config *SecurityConfig) *ConfigBuilder {

	cb.config.Security = config

	return cb

}

// WithPerformance sets performance configuration.

func (cb *ConfigBuilder) WithPerformance(config *PerformanceConfig) *ConfigBuilder {

	cb.config.Performance = config

	return cb

}

// Build returns the built configuration.

func (cb *ConfigBuilder) Build() *Config {

	// Apply defaults.

	cb.applyDefaults()

	// Validate configuration.

	if err := cb.validate(); err != nil {

		cb.logger.Error(err, "Configuration validation failed")

		return nil

	}

	return cb.config

}

// applyDefaults applies default values to the configuration.

func (cb *ConfigBuilder) applyDefaults() {

	if cb.config.KubernetesConfig == nil {

		cb.config.KubernetesConfig = cb.getDefaultKubernetesConfig()

	}

	if cb.config.PorchConfig == nil {

		cb.config.PorchConfig = cb.getDefaultPorchConfig()

	}

	if cb.config.Functions == nil {

		cb.config.Functions = cb.getDefaultFunctionConfig()

	}

	if cb.config.Observability == nil {

		cb.config.Observability = cb.getDefaultObservabilityConfig()

	}

	if cb.config.Security == nil {

		cb.config.Security = cb.getDefaultSecurityConfig()

	}

	if cb.config.Performance == nil {

		cb.config.Performance = cb.getDefaultPerformanceConfig()

	}

}

// getDefaultKubernetesConfig returns default Kubernetes configuration.

func (cb *ConfigBuilder) getDefaultKubernetesConfig() *KubernetesConfig {

	var kubeconfig string

	if home := homedir.HomeDir(); home != "" {

		kubeconfig = filepath.Join(home, ".kube", "config")

	}

	return &KubernetesConfig{

		KubeconfigPath: kubeconfig,

		QPS: 50,

		Burst: 100,

		Timeout: 30 * time.Second,

		UserAgent: "nephoran-porch-client/v1.0.0",
	}

}

// getDefaultPorchConfig returns default Porch configuration.

func (cb *ConfigBuilder) getDefaultPorchConfig() *PorchServiceConfig {

	return &PorchServiceConfig{

		APIVersion: "porch.kpt.dev/v1alpha1",

		Timeout: 30 * time.Second,

		Retry: &RetryConfig{

			MaxRetries: 3,

			InitialDelay: 1 * time.Second,

			MaxDelay: 30 * time.Second,

			BackoffFactor: 2.0,

			Jitter: true,
		},

		CircuitBreaker: &CircuitBreakerConfig{

			Enabled: true,

			FailureThreshold: 5,

			SuccessThreshold: 3,

			Timeout: 60 * time.Second,

			HalfOpenMaxCalls: 3,

			OpenStateTimeout: 30 * time.Second,
		},

		RateLimit: &RateLimitConfig{

			Enabled: true,

			RequestsPerSecond: 10,

			Burst: 20,

			Strategy: "token_bucket",
		},

		ConnectionPool: &ConnectionPoolConfig{

			MaxIdleConns: 10,

			MaxOpenConns: 100,

			ConnMaxLifetime: 30 * time.Minute,

			ConnMaxIdleTime: 5 * time.Minute,

			HealthCheckInterval: 30 * time.Second,
		},
	}

}

// getDefaultFunctionConfig returns default function configuration.

func (cb *ConfigBuilder) getDefaultFunctionConfig() *FunctionRegistryConfig {

	return &FunctionRegistryConfig{

		DefaultRegistry: "gcr.io/kpt-fn",

		Execution: &FunctionExecutionConfig{

			Runtime: "docker",

			DefaultTimeout: 5 * time.Minute,

			MaxConcurrent: 5,

			EnableNetworking: false,

			AllowPrivileged: false,

			ResourceLimits: &FunctionResourceLimits{

				CPU: "1000m",

				Memory: "1Gi",

				Storage: "1Gi",

				Timeout: 5 * time.Minute,
			},
		},

		Cache: &FunctionCacheConfig{

			Enabled: true,

			Size: "1Gi",

			TTL: 24 * time.Hour,

			CleanupInterval: 1 * time.Hour,

			Strategy: "LRU",
		},
	}

}

// getDefaultObservabilityConfig returns default observability configuration.

func (cb *ConfigBuilder) getDefaultObservabilityConfig() *ObservabilityConfig {

	return &ObservabilityConfig{

		Metrics: &MetricsConfig{

			Enabled: true,

			Provider: "prometheus",

			Namespace: "porch",
		},

		Logging: &LoggingConfig{

			Level: "info",

			Format: "json",

			Output: []string{"stdout"},
		},

		Tracing: &TracingConfig{

			Enabled: false,

			Provider: "jaeger",

			SamplingRate: 0.1,
		},

		HealthCheck: &HealthCheckConfig{

			Enabled: true,

			Port: 8080,

			Path: "/healthz",

			Interval: 30 * time.Second,

			Timeout: 5 * time.Second,

			HealthyThreshold: 1,

			UnhealthyThreshold: 3,
		},
	}

}

// getDefaultSecurityConfig returns default security configuration.

func (cb *ConfigBuilder) getDefaultSecurityConfig() *SecurityConfig {

	return &SecurityConfig{

		Authentication: &SecurityAuthConfig{

			Methods: []string{"cert"},
		},

		Authorization: &AuthorizationConfig{

			Enabled: true,

			Mode: "rbac",
		},

		Encryption: &EncryptionConfig{

			InTransit: &EncryptionInTransitConfig{

				Enabled: true,

				MinTLSVersion: "1.3",
			},
		},

		Certificates: &CertificateConfig{

			AutoRotation: true,

			TTL: 24 * time.Hour,
		},
	}

}

// getDefaultPerformanceConfig returns default performance configuration.

func (cb *ConfigBuilder) getDefaultPerformanceConfig() *PerformanceConfig {

	return &PerformanceConfig{

		Client: &ClientPerformanceConfig{

			MaxConnections: 100,

			ConnectionTimeout: 30 * time.Second,

			RequestTimeout: 60 * time.Second,

			MaxIdleConns: 10,
		},

		Caching: &CachingConfig{

			Enabled: true,

			Layers: []CacheLayerConfig{

				{

					Name: "memory",

					Type: "memory",

					Size: "100Mi",

					TTL: 5 * time.Minute,

					Enabled: true,
				},
			},
		},
	}

}

// validate validates the configuration.

func (cb *ConfigBuilder) validate() error {

	if cb.config.PorchConfig == nil {

		return fmt.Errorf("porch configuration is required")

	}

	if cb.config.PorchConfig.Endpoint == "" {

		return fmt.Errorf("porch endpoint is required")

	}

	// Validate repositories.

	for name, repo := range cb.config.Repositories {

		if repo.URL == "" {

			return fmt.Errorf("repository %s: URL is required", name)

		}

		if repo.Type == "" {

			return fmt.Errorf("repository %s: type is required", name)

		}

	}

	// Validate clusters.

	for name, cluster := range cb.config.Clusters {

		if cluster.Endpoint == "" && cluster.KubeconfigPath == "" {

			return fmt.Errorf("cluster %s: either endpoint or kubeconfig path is required", name)

		}

	}

	return nil

}

// Configuration factory functions.

// NewConfig creates a new empty configuration.

func NewConfig() *Config {

	return &Config{

		Repositories: make(map[string]*RepositoryConfig),

		Clusters: make(map[string]*ClusterConfig),
	}

}

// WithDefaults adds default values to the configuration.

func (c *Config) WithDefaults() *Config {

	builder := NewConfigBuilder()

	builder.config = c

	builder.applyDefaults()

	return builder.config

}

// NewDefaultConfig creates a default configuration.

func NewDefaultConfig() *Config {

	return NewConfigBuilder().Build()

}

// NewConfigFromEnvironment creates configuration from environment variables.

func NewConfigFromEnvironment() (*Config, error) {

	builder := NewConfigBuilder()

	// Set Porch endpoint from environment.

	if endpoint := os.Getenv("PORCH_ENDPOINT"); endpoint != "" {

		builder.WithPorchConfig(&PorchServiceConfig{

			Endpoint: endpoint,
		})

	}

	// Set Kubernetes context from environment.

	if context := os.Getenv("KUBERNETES_CONTEXT"); context != "" {

		builder.WithKubernetesConfig(&KubernetesConfig{

			Context: context,
		})

	}

	// Set namespace from environment.

	if namespace := os.Getenv("PORCH_NAMESPACE"); namespace != "" {

		if builder.config.KubernetesConfig == nil {

			builder.config.KubernetesConfig = &KubernetesConfig{}

		}

		builder.config.KubernetesConfig.Namespace = namespace

	}

	config := builder.Build()

	if config == nil {

		return nil, fmt.Errorf("failed to build configuration from environment")

	}

	return config, nil

}

// LoadConfigFromFile loads configuration from a file.

func LoadConfigFromFile(filepath string) (*Config, error) {

	// Implementation would load from YAML/JSON file.

	// For now, return default config.

	return NewDefaultConfig(), nil

}

// GetKubernetesConfig creates Kubernetes REST config from Porch config.

func (c *Config) GetKubernetesConfig() (*rest.Config, error) {

	if c.KubernetesConfig == nil {

		return nil, fmt.Errorf("kubernetes configuration not specified")

	}

	var config *rest.Config

	var err error

	if c.KubernetesConfig.KubeconfigPath != "" {

		// Load from kubeconfig file.

		config, err = clientcmd.BuildConfigFromFlags(

			c.KubernetesConfig.MasterURL,

			c.KubernetesConfig.KubeconfigPath,
		)

	} else {

		// Use in-cluster config.

		config, err = rest.InClusterConfig()

	}

	if err != nil {

		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)

	}

	// Apply configuration overrides.

	if c.KubernetesConfig.QPS > 0 {

		config.QPS = c.KubernetesConfig.QPS

	}

	if c.KubernetesConfig.Burst > 0 {

		config.Burst = c.KubernetesConfig.Burst

	}

	if c.KubernetesConfig.Timeout > 0 {

		config.Timeout = c.KubernetesConfig.Timeout

	}

	if c.KubernetesConfig.UserAgent != "" {

		config.UserAgent = c.KubernetesConfig.UserAgent

	}

	// Set additional headers.

	if len(c.KubernetesConfig.Headers) > 0 {

		if config.WrapTransport == nil {

			config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {

				return &headerRoundTripper{

					headers: c.KubernetesConfig.Headers,

					rt: rt,
				}

			}

		}

	}

	return config, nil

}

// headerRoundTripper adds custom headers to requests.

type headerRoundTripper struct {
	headers map[string]string

	rt http.RoundTripper
}

// RoundTrip performs roundtrip operation.

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	for key, value := range h.headers {

		req.Header.Set(key, value)

	}

	return h.rt.RoundTrip(req)

}

// GetRepository returns repository configuration by name.

func (c *Config) GetRepository(name string) (*RepositoryConfig, error) {

	repo, exists := c.Repositories[name]

	if !exists {

		return nil, fmt.Errorf("repository %s not found", name)

	}

	return repo, nil

}

// GetCluster returns cluster configuration by name.

func (c *Config) GetCluster(name string) (*ClusterConfig, error) {

	cluster, exists := c.Clusters[name]

	if !exists {

		return nil, fmt.Errorf("cluster %s not found", name)

	}

	return cluster, nil

}

// IsFeatureEnabled checks if an experimental feature is enabled.

func (c *Config) IsFeatureEnabled(feature string) bool {

	if c.PorchConfig == nil {

		return false

	}

	for _, enabled := range c.PorchConfig.ExperimentalFeatures {

		if enabled == feature {

			return true

		}

	}

	return false

}

// GetDefaultWorkflow returns the default workflow configuration.

func (c *Config) GetDefaultWorkflow() (*WorkflowConfig, error) {

	if c.Policies == nil || c.Policies.DefaultWorkflow == "" {

		return nil, fmt.Errorf("no default workflow configured")

	}

	workflow, exists := c.Policies.Workflows[c.Policies.DefaultWorkflow]

	if !exists {

		return nil, fmt.Errorf("default workflow %s not found", c.Policies.DefaultWorkflow)

	}

	return workflow, nil

}

// GetWorkflowForPackage returns workflow configuration for a specific package type.

func (c *Config) GetWorkflowForPackage(packageType string) (*WorkflowConfig, error) {

	if c.Policies == nil {

		return c.GetDefaultWorkflow()

	}

	workflow, exists := c.Policies.Workflows[packageType]

	if !exists {

		return c.GetDefaultWorkflow()

	}

	return workflow, nil

}

// ValidateConfiguration validates the entire configuration.

func (c *Config) Validate() []error {

	var errors []error

	// Validate Kubernetes config.

	if c.KubernetesConfig != nil {

		if c.KubernetesConfig.QPS < 0 {

			errors = append(errors, fmt.Errorf("kubernetes QPS cannot be negative"))

		}

		if c.KubernetesConfig.Burst < 0 {

			errors = append(errors, fmt.Errorf("kubernetes Burst cannot be negative"))

		}

	}

	// Validate Porch config.

	if c.PorchConfig == nil {

		errors = append(errors, fmt.Errorf("porch configuration is required"))

	} else {

		if c.PorchConfig.Endpoint == "" {

			errors = append(errors, fmt.Errorf("porch endpoint is required"))

		}

		if c.PorchConfig.Timeout <= 0 {

			errors = append(errors, fmt.Errorf("porch timeout must be positive"))

		}

	}

	// Validate repository configurations.

	for name, repo := range c.Repositories {

		if repo.URL == "" {

			errors = append(errors, fmt.Errorf("repository %s: URL is required", name))

		}

		if repo.Type == "" {

			errors = append(errors, fmt.Errorf("repository %s: type is required", name))

		}

		if repo.Type != "git" && repo.Type != "oci" {

			errors = append(errors, fmt.Errorf("repository %s: unsupported type %s", name, repo.Type))

		}

	}

	// Validate cluster configurations.

	for name, cluster := range c.Clusters {

		if cluster.Endpoint == "" && cluster.KubeconfigPath == "" {

			errors = append(errors, fmt.Errorf("cluster %s: either endpoint or kubeconfig path is required", name))

		}

	}

	return errors

}

// DeepCopy creates a deep copy of the configuration.

func (c *Config) DeepCopy() *Config {

	// Implementation would perform deep copy of all fields.

	// For brevity, showing the pattern.

	copy := &Config{}

	if c.KubernetesConfig != nil {

		copy.KubernetesConfig = &KubernetesConfig{}

		*copy.KubernetesConfig = *c.KubernetesConfig

		if c.KubernetesConfig.Headers != nil {

			copy.KubernetesConfig.Headers = make(map[string]string)

			for k, v := range c.KubernetesConfig.Headers {

				copy.KubernetesConfig.Headers[k] = v

			}

		}

	}

	// Continue with other fields...

	return copy

}

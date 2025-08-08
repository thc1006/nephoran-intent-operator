package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// O1InterfaceSpec defines the desired state of O1Interface
type O1InterfaceSpec struct {
	// Host is the hostname or IP address of the managed element
	Host string `json:"host"`
	
	// Port is the NETCONF port (default: 830)
	// +optional
	// +kubebuilder:default=830
	Port int `json:"port,omitempty"`
	
	// Protocol defines the transport protocol (ssh, tls)
	// +optional
	// +kubebuilder:default=ssh
	// +kubebuilder:validation:Enum=ssh;tls
	Protocol string `json:"protocol,omitempty"`
	
	// Credentials for authentication
	Credentials O1Credentials `json:"credentials"`
	
	// FCAPS configuration
	FCAPS FCAPSConfig `json:"fcaps"`
	
	// SMO integration configuration
	// +optional
	SMOConfig *SMOConfig `json:"smoConfig,omitempty"`
	
	// Streaming configuration for real-time data
	// +optional
	StreamingConfig *StreamingConfig `json:"streamingConfig,omitempty"`
	
	// YANG models to load
	// +optional
	YANGModels []YANGModelRef `json:"yangModels,omitempty"`
	
	// HighAvailability configuration
	// +optional
	HighAvailability *HighAvailabilityConfig `json:"highAvailability,omitempty"`
	
	// Security configuration
	// +optional
	SecurityConfig *O1SecurityConfig `json:"securityConfig,omitempty"`
}

// O1Credentials defines authentication credentials
type O1Credentials struct {
	// Username reference
	// +optional
	UsernameRef *corev1.SecretKeySelector `json:"usernameRef,omitempty"`
	
	// Password reference
	// +optional
	PasswordRef *corev1.SecretKeySelector `json:"passwordRef,omitempty"`
	
	// SSH private key reference
	// +optional
	PrivateKeyRef *corev1.SecretKeySelector `json:"privateKeyRef,omitempty"`
	
	// Client certificate reference for mTLS
	// +optional
	ClientCertificateRef *corev1.SecretKeySelector `json:"clientCertificateRef,omitempty"`
	
	// Client key reference for mTLS
	// +optional
	ClientKeyRef *corev1.SecretKeySelector `json:"clientKeyRef,omitempty"`
	
	// CA certificate reference
	// +optional
	CACertificateRef *corev1.SecretKeySelector `json:"caCertificateRef,omitempty"`
}

// FCAPSConfig defines FCAPS management configuration
type FCAPSConfig struct {
	// Fault management configuration
	FaultManagement FaultManagementConfig `json:"faultManagement"`
	
	// Configuration management
	ConfigurationManagement ConfigManagementConfig `json:"configurationManagement"`
	
	// Accounting management
	AccountingManagement AccountingManagementConfig `json:"accountingManagement"`
	
	// Performance management
	PerformanceManagement PerformanceManagementConfig `json:"performanceManagement"`
	
	// Security management
	SecurityManagement SecurityManagementConfig `json:"securityManagement"`
}

// FaultManagementConfig defines fault management settings
type FaultManagementConfig struct {
	// Enable fault management
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// Alarm correlation enabled
	// +optional
	// +kubebuilder:default=true
	CorrelationEnabled bool `json:"correlationEnabled,omitempty"`
	
	// Root cause analysis enabled
	// +optional
	// +kubebuilder:default=false
	RootCauseAnalysis bool `json:"rootCauseAnalysis,omitempty"`
	
	// Alarm severity filter (CRITICAL, MAJOR, MINOR, WARNING)
	// +optional
	SeverityFilter []string `json:"severityFilter,omitempty"`
	
	// AlertManager integration
	// +optional
	AlertManagerConfig *AlertManagerConfig `json:"alertManagerConfig,omitempty"`
	
	// Alarm retention period in days
	// +optional
	// +kubebuilder:default=30
	RetentionDays int `json:"retentionDays,omitempty"`
}

// ConfigManagementConfig defines configuration management settings
type ConfigManagementConfig struct {
	// Enable configuration management
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// GitOps integration enabled
	// +optional
	// +kubebuilder:default=true
	GitOpsEnabled bool `json:"gitOpsEnabled,omitempty"`
	
	// Configuration versioning enabled
	// +optional
	// +kubebuilder:default=true
	VersioningEnabled bool `json:"versioningEnabled,omitempty"`
	
	// Maximum versions to retain
	// +optional
	// +kubebuilder:default=10
	MaxVersions int `json:"maxVersions,omitempty"`
	
	// Drift detection enabled
	// +optional
	// +kubebuilder:default=true
	DriftDetection bool `json:"driftDetection,omitempty"`
	
	// Drift check interval in seconds
	// +optional
	// +kubebuilder:default=300
	DriftCheckInterval int `json:"driftCheckInterval,omitempty"`
}

// AccountingManagementConfig defines accounting management settings
type AccountingManagementConfig struct {
	// Enable accounting management
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// Usage collection interval in seconds
	// +optional
	// +kubebuilder:default=60
	CollectionInterval int `json:"collectionInterval,omitempty"`
	
	// Billing integration enabled
	// +optional
	BillingEnabled bool `json:"billingEnabled,omitempty"`
	
	// Fraud detection enabled
	// +optional
	FraudDetection bool `json:"fraudDetection,omitempty"`
	
	// Data retention period in days
	// +optional
	// +kubebuilder:default=90
	RetentionDays int `json:"retentionDays,omitempty"`
}

// PerformanceManagementConfig defines performance management settings
type PerformanceManagementConfig struct {
	// Enable performance management
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// KPI collection interval in seconds
	// +optional
	// +kubebuilder:default=15
	CollectionInterval int `json:"collectionInterval,omitempty"`
	
	// Aggregation periods (5min, 15min, 1hour, 1day)
	// +optional
	AggregationPeriods []string `json:"aggregationPeriods,omitempty"`
	
	// Anomaly detection enabled
	// +optional
	// +kubebuilder:default=false
	AnomalyDetection bool `json:"anomalyDetection,omitempty"`
	
	// Prometheus integration
	// +optional
	PrometheusConfig *O1PrometheusConfig `json:"prometheusConfig,omitempty"`
	
	// Grafana integration
	// +optional
	GrafanaConfig *GrafanaConfig `json:"grafanaConfig,omitempty"`
}

// SecurityManagementConfig defines security management settings
type SecurityManagementConfig struct {
	// Enable security management
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// Certificate management enabled
	// +optional
	// +kubebuilder:default=true
	CertificateManagement bool `json:"certificateManagement,omitempty"`
	
	// Intrusion detection enabled
	// +optional
	// +kubebuilder:default=false
	IntrusionDetection bool `json:"intrusionDetection,omitempty"`
	
	// Compliance monitoring enabled
	// +optional
	// +kubebuilder:default=true
	ComplianceMonitoring bool `json:"complianceMonitoring,omitempty"`
	
	// Security audit interval in seconds
	// +optional
	// +kubebuilder:default=3600
	AuditInterval int `json:"auditInterval,omitempty"`
}

// SMOConfig defines SMO integration settings
type SMOConfig struct {
	// SMO endpoint URL
	Endpoint string `json:"endpoint"`
	
	// Registration enabled
	// +optional
	// +kubebuilder:default=true
	RegistrationEnabled bool `json:"registrationEnabled,omitempty"`
	
	// Hierarchical management enabled
	// +optional
	HierarchicalManagement bool `json:"hierarchicalManagement,omitempty"`
	
	// Service discovery configuration
	// +optional
	ServiceDiscovery *ServiceDiscoveryConfig `json:"serviceDiscovery,omitempty"`
}

// StreamingConfig defines real-time streaming settings
type StreamingConfig struct {
	// Enable streaming
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// WebSocket port
	// +optional
	// +kubebuilder:default=8080
	WebSocketPort int `json:"webSocketPort,omitempty"`
	
	// Stream types to enable
	// +optional
	StreamTypes []string `json:"streamTypes,omitempty"`
	
	// Quality of Service level (BestEffort, Reliable, Guaranteed)
	// +optional
	// +kubebuilder:default=Reliable
	QoSLevel string `json:"qosLevel,omitempty"`
	
	// Maximum connections
	// +optional
	// +kubebuilder:default=100
	MaxConnections int `json:"maxConnections,omitempty"`
	
	// Rate limiting configuration
	// +optional
	RateLimiting *RateLimitConfig `json:"rateLimiting,omitempty"`
}

// YANGModelRef references a YANG model to load
type YANGModelRef struct {
	// Model name
	Name string `json:"name"`
	
	// Model version
	// +optional
	Version string `json:"version,omitempty"`
	
	// Model source (file, configmap, url)
	// +optional
	Source string `json:"source,omitempty"`
	
	// ConfigMap reference for model data
	// +optional
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
}

// HighAvailabilityConfig defines HA settings
type HighAvailabilityConfig struct {
	// Enable high availability
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	
	// Number of replicas
	// +optional
	// +kubebuilder:default=2
	Replicas int `json:"replicas,omitempty"`
	
	// Failover timeout in seconds
	// +optional
	// +kubebuilder:default=30
	FailoverTimeout int `json:"failoverTimeout,omitempty"`
}

// O1SecurityConfig defines security settings
type O1SecurityConfig struct {
	// TLS configuration
	// +optional
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
	
	// OAuth2 configuration
	// +optional
	OAuth2Config *OAuth2Config `json:"oauth2Config,omitempty"`
	
	// RBAC rules
	// +optional
	RBACRules []RBACRule `json:"rbacRules,omitempty"`
}

// Helper configuration types

// AlertManagerConfig for Prometheus AlertManager integration
type AlertManagerConfig struct {
	URL string `json:"url"`
	// +optional
	AuthToken string `json:"authToken,omitempty"`
}

// PrometheusConfig for Prometheus integration
type O1PrometheusConfig struct {
	URL string `json:"url"`
	// +optional
	PushGatewayURL string `json:"pushGatewayURL,omitempty"`
}

// GrafanaConfig for Grafana integration
type GrafanaConfig struct {
	URL string `json:"url"`
	// +optional
	DashboardID string `json:"dashboardID,omitempty"`
}

// ServiceDiscoveryConfig for service discovery
type ServiceDiscoveryConfig struct {
	// +kubebuilder:validation:Enum=kubernetes;consul;etcd
	Type string `json:"type"`
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// RateLimitConfig for rate limiting
type RateLimitConfig struct {
	// Requests per second
	RequestsPerSecond int `json:"requestsPerSecond"`
	// Burst size
	BurstSize int `json:"burstSize"`
}

// TLSConfig for TLS settings
type TLSConfig struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	// +optional
	// +kubebuilder:validation:Enum=TLS1.2;TLS1.3
	MinVersion string `json:"minVersion,omitempty"`
	// +optional
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

// OAuth2Config for OAuth2 settings
type OAuth2Config struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
	Provider string `json:"provider"`
	ClientID string `json:"clientID"`
	// +optional
	ClientSecretRef *corev1.SecretKeySelector `json:"clientSecretRef,omitempty"`
}

// RBACRule defines an RBAC rule
type RBACRule struct {
	Role string `json:"role"`
	Permissions []string `json:"permissions"`
}

// O1InterfaceStatus defines the observed state of O1Interface
type O1InterfaceStatus struct {
	// Phase represents the current phase of the O1 interface
	// +kubebuilder:validation:Enum=Pending;Initializing;Connecting;Connected;Failed;Terminating
	Phase string `json:"phase,omitempty"`
	
	// Connection status
	ConnectionStatus ConnectionStatus `json:"connectionStatus,omitempty"`
	
	// FCAPS status
	FCAPSStatus FCAPSStatus `json:"fcapsStatus,omitempty"`
	
	// SMO registration status
	// +optional
	SMOStatus *SMOStatus `json:"smoStatus,omitempty"`
	
	// Streaming service status
	// +optional
	StreamingStatus *StreamingStatus `json:"streamingStatus,omitempty"`
	
	// Last sync time
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	
	// Error message if any
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`
	
	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ConnectionStatus represents NETCONF connection status
type ConnectionStatus struct {
	Connected bool `json:"connected"`
	// +optional
	SessionID string `json:"sessionID,omitempty"`
	// +optional
	ConnectedAt *metav1.Time `json:"connectedAt,omitempty"`
	// +optional
	Capabilities []string `json:"capabilities,omitempty"`
}

// FCAPSStatus represents FCAPS subsystems status
type FCAPSStatus struct {
	FaultManagementReady bool `json:"faultManagementReady"`
	ConfigManagementReady bool `json:"configManagementReady"`
	AccountingManagementReady bool `json:"accountingManagementReady"`
	PerformanceManagementReady bool `json:"performanceManagementReady"`
	SecurityManagementReady bool `json:"securityManagementReady"`
	// +optional
	ActiveAlarms int `json:"activeAlarms,omitempty"`
	// +optional
	ConfigVersion string `json:"configVersion,omitempty"`
	// +optional
	CollectedKPIs int `json:"collectedKPIs,omitempty"`
}

// SMOStatus represents SMO integration status
type SMOStatus struct {
	Registered bool `json:"registered"`
	// +optional
	RegistrationID string `json:"registrationID,omitempty"`
	// +optional
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`
}

// StreamingStatus represents streaming service status
type StreamingStatus struct {
	Active bool `json:"active"`
	// +optional
	ActiveConnections int `json:"activeConnections,omitempty"`
	// +optional
	StreamedEvents int64 `json:"streamedEvents,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=o1if
// +kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.host"
// +kubebuilder:printcolumn:name="Port",type="integer",JSONPath=".spec.port"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Connected",type="boolean",JSONPath=".status.connectionStatus.connected"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// O1Interface is the Schema for the o1interfaces API
type O1Interface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   O1InterfaceSpec   `json:"spec,omitempty"`
	Status O1InterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// O1InterfaceList contains a list of O1Interface
type O1InterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []O1Interface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&O1Interface{}, &O1InterfaceList{})
}
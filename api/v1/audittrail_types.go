package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AuditTrailSpec defines the desired state of AuditTrail
type AuditTrailSpec struct {
	// Enabled controls whether audit logging is active
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// LogLevel controls the minimum severity level for audit events
	// +kubebuilder:validation:Enum=emergency;alert;critical;error;warning;notice;info;debug
	// +kubebuilder:default:="info"
	LogLevel string `json:"logLevel,omitempty"`

	// BatchSize controls how many events to process in a batch
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// +kubebuilder:default:=100
	BatchSize int `json:"batchSize,omitempty"`

	// FlushInterval controls how often to flush batched events (in seconds)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default:=10
	FlushInterval int `json:"flushInterval,omitempty"`

	// MaxQueueSize controls the maximum number of events to queue
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=100000
	// +kubebuilder:default:=10000
	MaxQueueSize int `json:"maxQueueSize,omitempty"`

	// EnableIntegrity controls whether log integrity protection is enabled
	// +kubebuilder:default:=true
	EnableIntegrity bool `json:"enableIntegrity,omitempty"`

	// ComplianceMode controls additional compliance-specific features
	// +kubebuilder:validation:items:Enum=soc2;iso27001;pci_dss;hipaa;gdpr;ccpa;fisma;nist_csf
	ComplianceMode []string `json:"complianceMode,omitempty"`

	// Backends configuration for different output destinations
	Backends []AuditBackendConfig `json:"backends,omitempty"`

	// RetentionPolicy defines how long audit events should be retained
	RetentionPolicy *RetentionPolicySpec `json:"retentionPolicy,omitempty"`

	// IntegrityConfig defines integrity protection settings
	IntegrityConfig *IntegrityConfigSpec `json:"integrityConfig,omitempty"`

	// NotificationConfig defines notification settings for audit events
	NotificationConfig *NotificationConfigSpec `json:"notificationConfig,omitempty"`
}

// AuditBackendConfig defines configuration for audit backends
type AuditBackendConfig struct {
	// Type specifies the backend type
	// +kubebuilder:validation:Enum=file;elasticsearch;splunk;syslog;kafka;cloudwatch;stackdriver;azure_monitor;siem;webhook
	Type string `json:"type"`

	// Name is a unique identifier for this backend instance
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Enabled controls whether this backend is active
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// Settings contains backend-specific configuration
	// +kubebuilder:pruning:PreserveUnknownFields
	Settings runtime.RawExtension `json:"settings,omitempty"`

	// Format specifies the output format
	// +kubebuilder:validation:Enum=json;text;cef;leef;syslog
	// +kubebuilder:default:="json"
	Format string `json:"format,omitempty"`

	// Compression enables compression for the backend
	// +kubebuilder:default:=false
	Compression bool `json:"compression,omitempty"`

	// BufferSize controls the internal buffer size
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100000
	// +kubebuilder:default:=1000
	BufferSize int `json:"bufferSize,omitempty"`

	// Timeout for backend operations (in seconds)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default:=30
	Timeout int `json:"timeout,omitempty"`

	// RetryPolicy for failed operations
	RetryPolicy *RetryPolicySpec `json:"retryPolicy,omitempty"`

	// TLS configuration for secure connections
	TLS *TLSConfigSpec `json:"tls,omitempty"`

	// Filter configuration for this backend
	Filter *FilterConfigSpec `json:"filter,omitempty"`
}

// RetryPolicySpec defines retry behavior for failed operations
type RetryPolicySpec struct {
	// MaxRetries defines the maximum number of retry attempts
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default:=3
	MaxRetries int `json:"maxRetries,omitempty"`

	// InitialDelay defines the initial delay between retries (in seconds)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=60
	// +kubebuilder:default:=1
	InitialDelay int `json:"initialDelay,omitempty"`

	// MaxDelay defines the maximum delay between retries (in seconds)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default:=30
	MaxDelay int `json:"maxDelay,omitempty"`

	// BackoffFactor defines the multiplier for exponential backoff
	// +kubebuilder:default:=2.0
	BackoffFactor float64 `json:"backoffFactor,omitempty"`
}

// TLSConfigSpec defines TLS settings for secure connections
type TLSConfigSpec struct {
	// Enabled controls whether TLS is enabled
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`

	// CertFile path to the certificate file
	CertFile string `json:"certFile,omitempty"`

	// KeyFile path to the private key file
	KeyFile string `json:"keyFile,omitempty"`

	// CAFile path to the CA certificate file
	CAFile string `json:"caFile,omitempty"`

	// ServerName for SNI
	ServerName string `json:"serverName,omitempty"`

	// InsecureSkipVerify controls whether to skip certificate verification
	// +kubebuilder:default:=false
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// FilterConfigSpec defines event filtering for backends
type FilterConfigSpec struct {
	// MinSeverity defines the minimum severity level to process
	// +kubebuilder:validation:Enum=emergency;alert;critical;error;warning;notice;info;debug
	// +kubebuilder:default:="info"
	MinSeverity string `json:"minSeverity,omitempty"`

	// EventTypes defines which event types to include
	EventTypes []string `json:"eventTypes,omitempty"`

	// Components defines which components to include
	Components []string `json:"components,omitempty"`

	// ExcludeTypes defines event types to exclude
	ExcludeTypes []string `json:"excludeTypes,omitempty"`

	// IncludeFields defines which fields to include in output
	IncludeFields []string `json:"includeFields,omitempty"`

	// ExcludeFields defines which fields to exclude from output
	ExcludeFields []string `json:"excludeFields,omitempty"`
}

// RetentionPolicySpec defines audit event retention policies
type RetentionPolicySpec struct {
	// DefaultRetention defines the default retention period (in days)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3650
	// +kubebuilder:default:=365
	DefaultRetention int `json:"defaultRetention,omitempty"`

	// CheckInterval defines how often to check for expired events (in hours)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=168
	// +kubebuilder:default:=24
	CheckInterval int `json:"checkInterval,omitempty"`

	// ArchivalEnabled controls whether events are archived before deletion
	// +kubebuilder:default:=true
	ArchivalEnabled bool `json:"archivalEnabled,omitempty"`

	// CompressionEnabled controls whether archived events are compressed
	// +kubebuilder:default:=true
	CompressionEnabled bool `json:"compressionEnabled,omitempty"`

	// EncryptionEnabled controls whether archived events are encrypted
	// +kubebuilder:default:=true
	EncryptionEnabled bool `json:"encryptionEnabled,omitempty"`

	// Policies defines specific retention policies for different event types
	Policies []SpecificRetentionPolicy `json:"policies,omitempty"`
}

// SpecificRetentionPolicy defines retention settings for specific event types
type SpecificRetentionPolicy struct {
	// Name identifies this retention policy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Description provides a human-readable description
	Description string `json:"description,omitempty"`

	// RetentionPeriod defines how long events should be retained (in days)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=7300
	RetentionPeriod int `json:"retentionPeriod"`

	// EventTypes defines which event types this policy applies to
	EventTypes []string `json:"eventTypes,omitempty"`

	// Severity defines the minimum severity level this policy applies to
	// +kubebuilder:validation:Enum=emergency;alert;critical;error;warning;notice;info;debug
	Severity string `json:"severity,omitempty"`

	// Components defines which components this policy applies to
	Components []string `json:"components,omitempty"`

	// ComplianceStandard defines which compliance standard requires this retention
	// +kubebuilder:validation:Enum=soc2;iso27001;pci_dss;hipaa;gdpr;ccpa;fisma;nist_csf
	ComplianceStandard string `json:"complianceStandard,omitempty"`

	// ArchiveBeforeDelete controls whether events are archived before deletion
	// +kubebuilder:default:=true
	ArchiveBeforeDelete bool `json:"archiveBeforeDelete,omitempty"`

	// RequireApproval controls whether deletion requires manual approval
	// +kubebuilder:default:=false
	RequireApproval bool `json:"requireApproval,omitempty"`

	// LegalHoldExempt controls whether events can be deleted during legal holds
	// +kubebuilder:default:=false
	LegalHoldExempt bool `json:"legalHoldExempt,omitempty"`
}

// IntegrityConfigSpec defines integrity protection settings
type IntegrityConfigSpec struct {
	// Enabled controls whether integrity protection is active
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`

	// KeySize defines the RSA key size for signing
	// +kubebuilder:validation:Enum=2048;3072;4096
	// +kubebuilder:default:=2048
	KeySize int `json:"keySize,omitempty"`

	// AutoGenerateKeys controls whether to automatically generate key pairs
	// +kubebuilder:default:=true
	AutoGenerateKeys bool `json:"autoGenerateKeys,omitempty"`

	// KeyPairSecret defines the secret containing existing key pairs
	KeyPairSecret string `json:"keyPairSecret,omitempty"`

	// VerificationMode defines how strict verification should be
	// +kubebuilder:validation:Enum=strict;permissive;disabled
	// +kubebuilder:default:="strict"
	VerificationMode string `json:"verificationMode,omitempty"`

	// MaxChainLength defines maximum integrity chain length to keep in memory
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=100000
	// +kubebuilder:default:=10000
	MaxChainLength int `json:"maxChainLength,omitempty"`
}

// NotificationConfigSpec defines notification settings
type NotificationConfigSpec struct {
	// Enabled controls whether notifications are sent
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled,omitempty"`

	// Webhooks defines webhook endpoints for notifications
	Webhooks []WebhookNotification `json:"webhooks,omitempty"`

	// Email defines email notification settings
	Email *EmailNotificationConfig `json:"email,omitempty"`

	// Slack defines Slack notification settings
	Slack *SlackNotificationConfig `json:"slack,omitempty"`
}

// WebhookNotification defines webhook notification settings
type WebhookNotification struct {
	// Name identifies this webhook
	Name string `json:"name"`

	// URL defines the webhook endpoint
	URL string `json:"url"`

	// EventTypes defines which events trigger this webhook
	EventTypes []string `json:"eventTypes,omitempty"`

	// MinSeverity defines minimum severity to trigger notifications
	// +kubebuilder:validation:Enum=emergency;alert;critical;error;warning;notice;info;debug
	MinSeverity string `json:"minSeverity,omitempty"`

	// Headers defines custom headers to send with webhook
	Headers map[string]string `json:"headers,omitempty"`

	// SecretRef references a secret containing authentication credentials
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// EmailNotificationConfig defines email notification settings
type EmailNotificationConfig struct {
	// SMTPServer defines the SMTP server address
	SMTPServer string `json:"smtpServer"`

	// SMTPPort defines the SMTP server port
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default:=587
	SMTPPort int `json:"smtpPort,omitempty"`

	// From defines the sender email address
	From string `json:"from"`

	// To defines the recipient email addresses
	To []string `json:"to"`

	// Subject template for email notifications
	Subject string `json:"subject,omitempty"`

	// SecretRef references a secret containing SMTP credentials
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// SlackNotificationConfig defines Slack notification settings
type SlackNotificationConfig struct {
	// WebhookURL defines the Slack webhook URL
	WebhookURL string `json:"webhookUrl,omitempty"`

	// Channel defines the Slack channel to post to
	Channel string `json:"channel,omitempty"`

	// Username defines the bot username
	Username string `json:"username,omitempty"`

	// SecretRef references a secret containing Slack credentials
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// SecretReference references a Kubernetes secret
type SecretReference struct {
	// Name of the secret
	Name string `json:"name"`

	// Namespace of the secret (optional, defaults to same namespace as AuditTrail)
	Namespace string `json:"namespace,omitempty"`

	// Key within the secret
	Key string `json:"key,omitempty"`
}

// AuditTrailStatus defines the observed state of AuditTrail
type AuditTrailStatus struct {
	// Phase represents the current phase of the audit trail
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Stopping;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the audit trail's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdate timestamp of the last status update
	LastUpdate *metav1.Time `json:"lastUpdate,omitempty"`

	// Stats contains operational statistics
	Stats *AuditTrailStats `json:"stats,omitempty"`

	// BackendStatus contains status of individual backends
	BackendStatus []BackendStatus `json:"backendStatus,omitempty"`

	// IntegrityStatus contains integrity chain status
	IntegrityStatus *IntegrityStatus `json:"integrityStatus,omitempty"`

	// ComplianceStatus contains compliance tracking status
	ComplianceStatus *ComplianceStatus `json:"complianceStatus,omitempty"`
}

// AuditTrailStats contains operational statistics
type AuditTrailStats struct {
	// EventsReceived is the total number of events received
	EventsReceived int64 `json:"eventsReceived"`

	// EventsProcessed is the total number of events successfully processed
	EventsProcessed int64 `json:"eventsProcessed"`

	// EventsDropped is the total number of events dropped
	EventsDropped int64 `json:"eventsDropped"`

	// EventsArchived is the total number of events archived
	EventsArchived int64 `json:"eventsArchived"`

	// LastEventTime is the timestamp of the last processed event
	LastEventTime *metav1.Time `json:"lastEventTime,omitempty"`

	// QueueSize is the current size of the event queue
	QueueSize int `json:"queueSize"`

	// BackendCount is the number of active backends
	BackendCount int `json:"backendCount"`
}

// BackendStatus contains status of an individual backend
type BackendStatus struct {
	// Name of the backend
	Name string `json:"name"`

	// Type of the backend
	Type string `json:"type"`

	// Healthy indicates if the backend is healthy
	Healthy bool `json:"healthy"`

	// LastCheck is the timestamp of the last health check
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// Error contains the last error message if unhealthy
	Error string `json:"error,omitempty"`

	// EventsWritten is the number of events successfully written
	EventsWritten int64 `json:"eventsWritten"`

	// EventsFailed is the number of events that failed to write
	EventsFailed int64 `json:"eventsFailed"`

	// LastEventTime is the timestamp of the last event written
	LastEventTime *metav1.Time `json:"lastEventTime,omitempty"`
}

// IntegrityStatus contains integrity chain status
type IntegrityStatus struct {
	// Enabled indicates if integrity protection is enabled
	Enabled bool `json:"enabled"`

	// ChainLength is the current length of the integrity chain
	ChainLength int `json:"chainLength"`

	// LastSequence is the sequence number of the last event
	LastSequence int64 `json:"lastSequence"`

	// LastHash is the hash of the last event in the chain
	LastHash string `json:"lastHash,omitempty"`

	// KeyID is the ID of the current signing key
	KeyID string `json:"keyId,omitempty"`

	// LastVerification is the timestamp of the last chain verification
	LastVerification *metav1.Time `json:"lastVerification,omitempty"`

	// VerificationResult is the result of the last verification
	VerificationResult string `json:"verificationResult,omitempty"`
}

// ComplianceStatus contains compliance tracking status
type ComplianceStatus struct {
	// Standards is the list of active compliance standards
	Standards []string `json:"standards,omitempty"`

	// LastReport is the timestamp of the last compliance report
	LastReport *metav1.Time `json:"lastReport,omitempty"`

	// ViolationCount is the total number of violations detected
	ViolationCount int64 `json:"violationCount"`

	// RecentViolations is a list of recent violations
	RecentViolations []ComplianceViolationSummary `json:"recentViolations,omitempty"`
}

// ComplianceViolationSummary contains summary of a compliance violation
type ComplianceViolationSummary struct {
	// ViolationID is the unique identifier for the violation
	ViolationID string `json:"violationId"`

	// Standard is the compliance standard that was violated
	Standard string `json:"standard"`

	// ControlID is the specific control that was violated
	ControlID string `json:"controlId"`

	// Severity is the severity of the violation
	Severity string `json:"severity"`

	// DetectedAt is when the violation was detected
	DetectedAt *metav1.Time `json:"detectedAt"`

	// Status is the current remediation status
	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=security
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled"
// +kubebuilder:printcolumn:name="Backends",type="integer",JSONPath=".status.stats.backendCount"
// +kubebuilder:printcolumn:name="Events",type="integer",JSONPath=".status.stats.eventsReceived"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AuditTrail is the Schema for the audittrails API
type AuditTrail struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuditTrailSpec   `json:"spec,omitempty"`
	Status AuditTrailStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AuditTrailList contains a list of AuditTrail
type AuditTrailList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuditTrail `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuditTrail{}, &AuditTrailList{})
}

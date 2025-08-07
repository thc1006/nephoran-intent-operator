package audit

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Severity levels for audit events (aligned with syslog RFC 5424)
type Severity int

const (
	SeverityEmergency Severity = iota // System is unusable
	SeverityAlert                     // Action must be taken immediately
	SeverityCritical                  // Critical conditions
	SeverityError                     // Error conditions
	SeverityWarning                   // Warning conditions
	SeverityNotice                    // Normal but significant condition
	SeverityInfo                      // Informational messages
	SeverityDebug                     // Debug-level messages
)

// String returns the string representation of the severity
func (s Severity) String() string {
	switch s {
	case SeverityEmergency:
		return "emergency"
	case SeverityAlert:
		return "alert"
	case SeverityCritical:
		return "critical"
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityNotice:
		return "notice"
	case SeverityInfo:
		return "info"
	case SeverityDebug:
		return "debug"
	default:
		return "unknown"
	}
}

// EventType represents the type of audit event
type EventType string

const (
	// Authentication and Authorization Events
	EventTypeAuthentication       EventType = "authentication"
	EventTypeAuthenticationFailed EventType = "authentication_failed"
	EventTypeAuthenticationSuccess EventType = "authentication_success"
	EventTypeAuthorization        EventType = "authorization"
	EventTypeAuthorizationFailed  EventType = "authorization_failed"
	EventTypeAuthorizationSuccess EventType = "authorization_success"
	EventTypeSessionStart         EventType = "session_start"
	EventTypeSessionEnd           EventType = "session_end"
	EventTypePasswordChange       EventType = "password_change"
	EventTypeTokenIssuance        EventType = "token_issuance"
	EventTypeTokenRevocation      EventType = "token_revocation"
	
	// Data Access Events
	EventTypeDataAccess    EventType = "data_access"
	EventTypeDataCreate    EventType = "data_create"
	EventTypeDataRead      EventType = "data_read"
	EventTypeDataUpdate    EventType = "data_update"
	EventTypeDataDelete    EventType = "data_delete"
	EventTypeDataExport    EventType = "data_export"
	EventTypeDataImport    EventType = "data_import"
	EventTypeDataBackup    EventType = "data_backup"
	EventTypeDataRestore   EventType = "data_restore"
	
	// System Management Events
	EventTypeSystemChange     EventType = "system_change"
	EventTypeConfigChange     EventType = "config_change"
	EventTypeUserManagement   EventType = "user_management"
	EventTypeRoleManagement   EventType = "role_management"
	EventTypePolicyChange     EventType = "policy_change"
	EventTypeSystemStartup    EventType = "system_startup"
	EventTypeSystemShutdown   EventType = "system_shutdown"
	EventTypeServiceStart     EventType = "service_start"
	EventTypeServiceStop      EventType = "service_stop"
	EventTypeMaintenanceMode  EventType = "maintenance_mode"
	
	// Security Events
	EventTypeSecurityViolation EventType = "security_violation"
	EventTypeIntrusionAttempt  EventType = "intrusion_attempt"
	EventTypeMalwareDetection  EventType = "malware_detection"
	EventTypeAnomalyDetection  EventType = "anomaly_detection"
	EventTypeComplianceCheck   EventType = "compliance_check"
	EventTypeVulnerability     EventType = "vulnerability"
	EventTypeIncidentResponse  EventType = "incident_response"
	
	// Network and Infrastructure Events
	EventTypeNetworkAccess     EventType = "network_access"
	EventTypeFirewallRule      EventType = "firewall_rule"
	EventTypeNetworkAnomaly    EventType = "network_anomaly"
	EventTypeResourceAccess    EventType = "resource_access"
	EventTypeCapacityChange    EventType = "capacity_change"
	EventTypePerformanceAlert  EventType = "performance_alert"
	
	// Application-Specific Events
	EventTypeAPICall           EventType = "api_call"
	EventTypeWorkflowExecution EventType = "workflow_execution"
	EventTypeJobExecution      EventType = "job_execution"
	EventTypeDeployment        EventType = "deployment"
	EventTypeRollback          EventType = "rollback"
	EventTypeHealthCheck       EventType = "health_check"
	
	// O-RAN Specific Events
	EventTypeIntentProcessing  EventType = "intent_processing"
	EventTypeNetworkFunction   EventType = "network_function"
	EventTypeRICManagement     EventType = "ric_management"
	EventTypeA1Interface       EventType = "a1_interface"
	EventTypeO1Interface       EventType = "o1_interface"
	EventTypeE2Interface       EventType = "e2_interface"
	EventTypeSliceManagement   EventType = "slice_management"
)

// EventResult represents the outcome of an audited action
type EventResult string

const (
	ResultSuccess EventResult = "success"
	ResultFailure EventResult = "failure"
	ResultDenied  EventResult = "denied"
	ResultTimeout EventResult = "timeout"
	ResultError   EventResult = "error"
	ResultPartial EventResult = "partial"
)

// ComplianceStandard represents different compliance frameworks
type ComplianceStandard string

const (
	ComplianceSOC2     ComplianceStandard = "soc2"
	ComplianceISO27001 ComplianceStandard = "iso27001"
	CompliancePCIDSS   ComplianceStandard = "pci_dss"
	ComplianceHIPAA    ComplianceStandard = "hipaa"
	ComplianceGDPR     ComplianceStandard = "gdpr"
	ComplianceCCPA     ComplianceStandard = "ccpa"
	ComplianceFISMA    ComplianceStandard = "fisma"
	ComplianceNIST     ComplianceStandard = "nist_csf"
)

// UserContext contains information about the user performing the action
type UserContext struct {
	UserID       string            `json:"user_id"`
	Username     string            `json:"username,omitempty"`
	Email        string            `json:"email,omitempty"`
	Role         string            `json:"role,omitempty"`
	Groups       []string          `json:"groups,omitempty"`
	Permissions  []string          `json:"permissions,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
	ClientID     string            `json:"client_id,omitempty"`
	ServiceAccount bool            `json:"service_account"`
	AuthMethod   string            `json:"auth_method,omitempty"`
	AuthProvider string            `json:"auth_provider,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// NetworkContext contains network-related information
type NetworkContext struct {
	SourceIP        net.IP            `json:"source_ip,omitempty"`
	DestinationIP   net.IP            `json:"destination_ip,omitempty"`
	SourcePort      int               `json:"source_port,omitempty"`
	DestinationPort int               `json:"destination_port,omitempty"`
	Protocol        string            `json:"protocol,omitempty"`
	UserAgent       string            `json:"user_agent,omitempty"`
	Referrer        string            `json:"referrer,omitempty"`
	RequestID       string            `json:"request_id,omitempty"`
	Country         string            `json:"country,omitempty"`
	ASN             string            `json:"asn,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
}

// SystemContext contains system-related information
type SystemContext struct {
	Hostname    string `json:"hostname"`
	ProcessID   int    `json:"process_id"`
	ProcessName string `json:"process_name,omitempty"`
	ThreadID    int    `json:"thread_id"`
	ServiceName string `json:"service_name,omitempty"`
	Version     string `json:"version,omitempty"`
	Environment string `json:"environment,omitempty"`
	Cluster     string `json:"cluster,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	PodName     string `json:"pod_name,omitempty"`
	NodeName    string `json:"node_name,omitempty"`
}

// ResourceContext contains information about the resource being accessed
type ResourceContext struct {
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	ResourceName string            `json:"resource_name,omitempty"`
	Namespace    string            `json:"namespace,omitempty"`
	APIVersion   string            `json:"api_version,omitempty"`
	Kind         string            `json:"kind,omitempty"`
	Operation    string            `json:"operation"`
	Path         string            `json:"path,omitempty"`
	Query        map[string]string `json:"query,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// AuditEvent represents a complete audit log entry
type AuditEvent struct {
	// Core identification fields
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	
	// Event classification
	EventType   EventType   `json:"event_type"`
	Category    string      `json:"category,omitempty"`
	Severity    Severity    `json:"severity"`
	Result      EventResult `json:"result"`
	
	// Core event information
	Component   string `json:"component"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Message     string `json:"message,omitempty"`
	
	// Context information
	UserContext     *UserContext     `json:"user_context,omitempty"`
	NetworkContext  *NetworkContext  `json:"network_context,omitempty"`
	SystemContext   *SystemContext   `json:"system_context,omitempty"`
	ResourceContext *ResourceContext `json:"resource_context,omitempty"`
	
	// Event-specific data
	Data map[string]interface{} `json:"data,omitempty"`
	
	// Security and compliance
	RiskLevel           string                 `json:"risk_level,omitempty"`
	ComplianceMetadata  map[string]interface{} `json:"compliance_metadata,omitempty"`
	DataClassification  string                 `json:"data_classification,omitempty"`
	RetentionPeriod     string                 `json:"retention_period,omitempty"`
	
	// Error information
	Error      string            `json:"error,omitempty"`
	ErrorCode  string            `json:"error_code,omitempty"`
	StackTrace string            `json:"stack_trace,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
	
	// Performance metrics
	Duration     time.Duration     `json:"duration,omitempty"`
	ResponseSize int64            `json:"response_size,omitempty"`
	RequestSize  int64            `json:"request_size,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	
	// Correlation and tracing
	TraceID         string            `json:"trace_id,omitempty"`
	SpanID          string            `json:"span_id,omitempty"`
	ParentID        string            `json:"parent_id,omitempty"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	CausationID     string            `json:"causation_id,omitempty"`
	RelatedEvents   []string          `json:"related_events,omitempty"`
	
	// Integrity and forensics
	Signature       string `json:"signature,omitempty"`
	Hash            string `json:"hash,omitempty"`
	PreviousHash    string `json:"previous_hash,omitempty"`
	IntegrityFields []string `json:"integrity_fields,omitempty"`
}

// Validate checks if the audit event has all required fields
func (ae *AuditEvent) Validate() error {
	if ae.ID == "" {
		return fmt.Errorf("audit event ID is required")
	}
	
	if !isValidUUID(ae.ID) {
		return fmt.Errorf("audit event ID must be a valid UUID")
	}
	
	if ae.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	
	if ae.Component == "" {
		return fmt.Errorf("component is required")
	}
	
	if ae.Action == "" {
		return fmt.Errorf("action is required")
	}
	
	if ae.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	
	// Validate network context if present
	if ae.NetworkContext != nil {
		if err := ae.NetworkContext.Validate(); err != nil {
			return fmt.Errorf("invalid network context: %w", err)
		}
	}
	
	// Validate user context if present
	if ae.UserContext != nil {
		if err := ae.UserContext.Validate(); err != nil {
			return fmt.Errorf("invalid user context: %w", err)
		}
	}
	
	// Validate resource context if present
	if ae.ResourceContext != nil {
		if err := ae.ResourceContext.Validate(); err != nil {
			return fmt.Errorf("invalid resource context: %w", err)
		}
	}
	
	return nil
}

// Validate checks if the user context is valid
func (uc *UserContext) Validate() error {
	if uc.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	// Validate email format if present
	if uc.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(uc.Email) {
			return fmt.Errorf("invalid email format: %s", uc.Email)
		}
	}
	
	return nil
}

// Validate checks if the network context is valid
func (nc *NetworkContext) Validate() error {
	// Validate port ranges if present
	if nc.SourcePort < 0 || nc.SourcePort > 65535 {
		return fmt.Errorf("invalid source port: %d", nc.SourcePort)
	}
	
	if nc.DestinationPort < 0 || nc.DestinationPort > 65535 {
		return fmt.Errorf("invalid destination port: %d", nc.DestinationPort)
	}
	
	return nil
}

// Validate checks if the resource context is valid
func (rc *ResourceContext) Validate() error {
	if rc.ResourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	
	if rc.Operation == "" {
		return fmt.Errorf("operation is required")
	}
	
	return nil
}

// ToJSON converts the audit event to JSON
func (ae *AuditEvent) ToJSON() ([]byte, error) {
	return json.Marshal(ae)
}

// ToJSONIndent converts the audit event to indented JSON
func (ae *AuditEvent) ToJSONIndent() ([]byte, error) {
	return json.MarshalIndent(ae, "", "  ")
}

// FromJSON creates an audit event from JSON data
func FromJSON(data []byte) (*AuditEvent, error) {
	var event AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audit event: %w", err)
	}
	
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("invalid audit event: %w", err)
	}
	
	return &event, nil
}

// SetRiskLevel sets the risk level based on event type and severity
func (ae *AuditEvent) SetRiskLevel() {
	switch ae.EventType {
	case EventTypeAuthenticationFailed, EventTypeIntrusionAttempt, EventTypeMalwareDetection:
		ae.RiskLevel = "high"
	case EventTypeSecurityViolation, EventTypeAnomalyDetection:
		ae.RiskLevel = "medium"
	case EventTypeDataAccess, EventTypeSystemChange:
		if ae.Severity >= SeverityError {
			ae.RiskLevel = "high"
		} else if ae.Severity >= SeverityWarning {
			ae.RiskLevel = "medium"
		} else {
			ae.RiskLevel = "low"
		}
	default:
		ae.RiskLevel = "low"
	}
}

// SetRetentionPeriod sets the retention period based on compliance requirements and event type
func (ae *AuditEvent) SetRetentionPeriod(complianceMode []ComplianceStandard) {
	// Default retention periods based on compliance standards
	maxRetention := "1y" // Default 1 year
	
	for _, standard := range complianceMode {
		switch standard {
		case ComplianceSOC2:
			maxRetention = "7y" // SOC2 typically requires 7 years
		case ComplianceISO27001:
			maxRetention = "3y" // ISO 27001 typically requires 3 years
		case CompliancePCIDSS:
			maxRetention = "1y" // PCI DSS requires at least 1 year
		case ComplianceHIPAA:
			maxRetention = "6y" // HIPAA requires 6 years
		case ComplianceGDPR:
			// GDPR has specific retention requirements based on data type
			if ae.EventType == EventTypeDataAccess || ae.EventType == EventTypeDataProcessing {
				maxRetention = "3y"
			}
		}
	}
	
	// Security events may need longer retention
	switch ae.EventType {
	case EventTypeSecurityViolation, EventTypeIntrusionAttempt, EventTypeMalwareDetection:
		maxRetention = "7y"
	case EventTypeIncidentResponse:
		maxRetention = "10y"
	}
	
	ae.RetentionPeriod = maxRetention
}

// GetEventCategory returns a category for the event type
func (ae *AuditEvent) GetEventCategory() string {
	switch ae.EventType {
	case EventTypeAuthentication, EventTypeAuthenticationFailed, EventTypeAuthenticationSuccess,
		 EventTypeAuthorization, EventTypeAuthorizationFailed, EventTypeAuthorizationSuccess,
		 EventTypeSessionStart, EventTypeSessionEnd, EventTypePasswordChange,
		 EventTypeTokenIssuance, EventTypeTokenRevocation:
		return "authentication_authorization"
		
	case EventTypeDataAccess, EventTypeDataCreate, EventTypeDataRead, EventTypeDataUpdate,
		 EventTypeDataDelete, EventTypeDataExport, EventTypeDataImport, EventTypeDataBackup,
		 EventTypeDataRestore:
		return "data_access"
		
	case EventTypeSystemChange, EventTypeConfigChange, EventTypeUserManagement,
		 EventTypeRoleManagement, EventTypePolicyChange, EventTypeSystemStartup,
		 EventTypeSystemShutdown, EventTypeServiceStart, EventTypeServiceStop,
		 EventTypeMaintenanceMode:
		return "system_management"
		
	case EventTypeSecurityViolation, EventTypeIntrusionAttempt, EventTypeMalwareDetection,
		 EventTypeAnomalyDetection, EventTypeComplianceCheck, EventTypeVulnerability,
		 EventTypeIncidentResponse:
		return "security"
		
	case EventTypeNetworkAccess, EventTypeFirewallRule, EventTypeNetworkAnomaly,
		 EventTypeResourceAccess, EventTypeCapacityChange, EventTypePerformanceAlert:
		return "network_infrastructure"
		
	case EventTypeAPICall, EventTypeWorkflowExecution, EventTypeJobExecution,
		 EventTypeDeployment, EventTypeRollback, EventTypeHealthCheck:
		return "application"
		
	case EventTypeIntentProcessing, EventTypeNetworkFunction, EventTypeRICManagement,
		 EventTypeA1Interface, EventTypeO1Interface, EventTypeE2Interface,
		 EventTypeSliceManagement:
		return "oran_telecom"
		
	default:
		return "general"
	}
}

// Helper function to validate UUID format
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// EventBuilder provides a fluent interface for building audit events
type EventBuilder struct {
	event *AuditEvent
}

// NewEventBuilder creates a new event builder
func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &AuditEvent{
			ID:        uuid.New().String(),
			Version:   "1.0",
			Timestamp: time.Now().UTC(),
			Data:      make(map[string]interface{}),
		},
	}
}

// WithEventType sets the event type
func (eb *EventBuilder) WithEventType(eventType EventType) *EventBuilder {
	eb.event.EventType = eventType
	eb.event.Category = eb.event.GetEventCategory()
	return eb
}

// WithSeverity sets the severity
func (eb *EventBuilder) WithSeverity(severity Severity) *EventBuilder {
	eb.event.Severity = severity
	return eb
}

// WithComponent sets the component
func (eb *EventBuilder) WithComponent(component string) *EventBuilder {
	eb.event.Component = component
	return eb
}

// WithAction sets the action
func (eb *EventBuilder) WithAction(action string) *EventBuilder {
	eb.event.Action = action
	return eb
}

// WithDescription sets the description
func (eb *EventBuilder) WithDescription(description string) *EventBuilder {
	eb.event.Description = description
	return eb
}

// WithResult sets the result
func (eb *EventBuilder) WithResult(result EventResult) *EventBuilder {
	eb.event.Result = result
	return eb
}

// WithUser sets the user context
func (eb *EventBuilder) WithUser(userID, username string) *EventBuilder {
	if eb.event.UserContext == nil {
		eb.event.UserContext = &UserContext{}
	}
	eb.event.UserContext.UserID = userID
	eb.event.UserContext.Username = username
	return eb
}

// WithNetwork sets the network context
func (eb *EventBuilder) WithNetwork(sourceIP string, userAgent string) *EventBuilder {
	if eb.event.NetworkContext == nil {
		eb.event.NetworkContext = &NetworkContext{}
	}
	if sourceIP != "" {
		eb.event.NetworkContext.SourceIP = net.ParseIP(sourceIP)
	}
	eb.event.NetworkContext.UserAgent = userAgent
	return eb
}

// WithResource sets the resource context
func (eb *EventBuilder) WithResource(resourceType, resourceID, operation string) *EventBuilder {
	if eb.event.ResourceContext == nil {
		eb.event.ResourceContext = &ResourceContext{}
	}
	eb.event.ResourceContext.ResourceType = resourceType
	eb.event.ResourceContext.ResourceID = resourceID
	eb.event.ResourceContext.Operation = operation
	return eb
}

// WithData adds data to the event
func (eb *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	if eb.event.Data == nil {
		eb.event.Data = make(map[string]interface{})
	}
	eb.event.Data[key] = value
	return eb
}

// WithError sets error information
func (eb *EventBuilder) WithError(err error) *EventBuilder {
	if err != nil {
		eb.event.Error = err.Error()
		eb.event.Result = ResultError
		if eb.event.Severity < SeverityError {
			eb.event.Severity = SeverityError
		}
	}
	return eb
}

// WithTracing sets tracing information
func (eb *EventBuilder) WithTracing(traceID, spanID string) *EventBuilder {
	eb.event.TraceID = traceID
	eb.event.SpanID = spanID
	return eb
}

// Build creates the final audit event
func (eb *EventBuilder) Build() *AuditEvent {
	eb.event.SetRiskLevel()
	return eb.event
}

// Pre-defined event templates for common scenarios

// AuthenticationEvent creates an authentication event
func AuthenticationEvent(userID, provider string, success bool, err error) *AuditEvent {
	builder := NewEventBuilder().
		WithEventType(EventTypeAuthentication).
		WithComponent("authentication").
		WithAction("login").
		WithUser(userID, "").
		WithData("provider", provider)
	
	if success {
		builder.WithResult(ResultSuccess).WithSeverity(SeverityInfo)
	} else {
		builder.WithResult(ResultFailure).WithSeverity(SeverityWarning)
	}
	
	if err != nil {
		builder.WithError(err)
	}
	
	return builder.Build()
}

// DataAccessEvent creates a data access event
func DataAccessEvent(userID, resourceType, resourceID, operation string) *AuditEvent {
	return NewEventBuilder().
		WithEventType(EventTypeDataAccess).
		WithComponent("data_access").
		WithAction(operation).
		WithUser(userID, "").
		WithResource(resourceType, resourceID, operation).
		WithSeverity(SeverityInfo).
		WithResult(ResultSuccess).
		Build()
}

// SecurityViolationEvent creates a security violation event
func SecurityViolationEvent(userID, violationType, description string) *AuditEvent {
	return NewEventBuilder().
		WithEventType(EventTypeSecurityViolation).
		WithComponent("security").
		WithAction("violation_detected").
		WithDescription(description).
		WithUser(userID, "").
		WithData("violation_type", violationType).
		WithSeverity(SeverityCritical).
		WithResult(ResultFailure).
		Build()
}

// SystemChangeEvent creates a system change event
func SystemChangeEvent(userID, changeType, description string) *AuditEvent {
	return NewEventBuilder().
		WithEventType(EventTypeSystemChange).
		WithComponent("system").
		WithAction("change").
		WithDescription(description).
		WithUser(userID, "").
		WithData("change_type", changeType).
		WithSeverity(SeverityInfo).
		WithResult(ResultSuccess).
		Build()
}
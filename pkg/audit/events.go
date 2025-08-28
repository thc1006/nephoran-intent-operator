package audit

// Re-export types from the types package for backward compatibility
import (
	audittypes "github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// Type aliases for backward compatibility
type Severity = audittypes.Severity
type EventType = audittypes.EventType
type EventResult = audittypes.EventResult
type ComplianceStandard = audittypes.ComplianceStandard
type UserContext = audittypes.UserContext
type NetworkContext = audittypes.NetworkContext
type SystemContext = audittypes.SystemContext
type ResourceContext = audittypes.ResourceContext
type AuditEvent = audittypes.AuditEvent
type EventBuilder = audittypes.EventBuilder

// Constants re-exported for backward compatibility
const (
	SeverityEmergency = audittypes.SeverityEmergency
	SeverityAlert     = audittypes.SeverityAlert
	SeverityCritical  = audittypes.SeverityCritical
	SeverityError     = audittypes.SeverityError
	SeverityWarning   = audittypes.SeverityWarning
	SeverityNotice    = audittypes.SeverityNotice
	SeverityInfo      = audittypes.SeverityInfo
	SeverityDebug     = audittypes.SeverityDebug
)

// Event type constants
const (
	EventTypeDataCreate           = audittypes.EventTypeDataCreate
	EventTypeDataRead             = audittypes.EventTypeDataRead
	EventTypeDataUpdate           = audittypes.EventTypeDataUpdate
	EventTypeDataDelete           = audittypes.EventTypeDataDelete
	EventTypeDataAccess           = audittypes.EventTypeDataAccess
	EventTypeAPICall              = audittypes.EventTypeAPICall
	EventTypeAuthentication       = audittypes.EventTypeAuthentication
	EventTypeAuthenticationFailed = audittypes.EventTypeAuthenticationFailed
	EventTypeAuthorization        = audittypes.EventTypeAuthorization
	EventTypeSecurityViolation    = audittypes.EventTypeSecurityViolation
	EventTypeIntrusionAttempt     = audittypes.EventTypeIntrusionAttempt
	EventTypeMalwareDetection     = audittypes.EventTypeMalwareDetection
)

// Result constants
const (
	ResultSuccess = audittypes.ResultSuccess
	ResultFailure = audittypes.ResultFailure
	ResultDenied  = audittypes.ResultDenied
	ResultTimeout = audittypes.ResultTimeout
	ResultError   = audittypes.ResultError
	ResultPartial = audittypes.ResultPartial
)

// Compliance standards
const (
	ComplianceSOC2     = audittypes.ComplianceSOC2
	ComplianceISO27001 = audittypes.ComplianceISO27001
)

// NewEventBuilder creates a new event builder
func NewEventBuilder() *EventBuilder {
	return audittypes.NewEventBuilder()
}
package audit

// Re-export types from the types package for backward compatibility
import (
	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// Type aliases for backward compatibility
type Severity = types.Severity
type EventType = types.EventType
type EventResult = types.EventResult
type ComplianceStandard = types.ComplianceStandard
type UserContext = types.UserContext
type NetworkContext = types.NetworkContext
type SystemContext = types.SystemContext
type ResourceContext = types.ResourceContext
type AuditEvent = types.AuditEvent
type EventBuilder = types.EventBuilder

// Constants re-exported for backward compatibility
const (
	SeverityEmergency = types.SeverityEmergency
	SeverityAlert     = types.SeverityAlert
	SeverityCritical  = types.SeverityCritical
	SeverityError     = types.SeverityError
	SeverityWarning   = types.SeverityWarning
	SeverityNotice    = types.SeverityNotice
	SeverityInfo      = types.SeverityInfo
	SeverityDebug     = types.SeverityDebug

	EventTypeAuthentication        = types.EventTypeAuthentication
	EventTypeAuthenticationFailed  = types.EventTypeAuthenticationFailed
	EventTypeAuthenticationSuccess = types.EventTypeAuthenticationSuccess
	EventTypeAuthorization         = types.EventTypeAuthorization
	EventTypeAuthorizationFailed   = types.EventTypeAuthorizationFailed
	EventTypeAuthorizationSuccess  = types.EventTypeAuthorizationSuccess
	EventTypeSessionStart          = types.EventTypeSessionStart
	EventTypeSessionEnd            = types.EventTypeSessionEnd
	EventTypePasswordChange        = types.EventTypePasswordChange
	EventTypeTokenIssuance         = types.EventTypeTokenIssuance
	EventTypeTokenRevocation       = types.EventTypeTokenRevocation

	EventTypeDataAccess     = types.EventTypeDataAccess
	EventTypeDataCreate     = types.EventTypeDataCreate
	EventTypeDataRead       = types.EventTypeDataRead
	EventTypeDataUpdate     = types.EventTypeDataUpdate
	EventTypeDataDelete     = types.EventTypeDataDelete
	EventTypeDataExport     = types.EventTypeDataExport
	EventTypeDataImport     = types.EventTypeDataImport
	EventTypeDataBackup     = types.EventTypeDataBackup
	EventTypeDataRestore    = types.EventTypeDataRestore
	EventTypeDataProcessing = types.EventTypeDataProcessing

	EventTypeSystemChange    = types.EventTypeSystemChange
	EventTypeConfigChange    = types.EventTypeConfigChange
	EventTypeUserManagement  = types.EventTypeUserManagement
	EventTypeRoleManagement  = types.EventTypeRoleManagement
	EventTypePolicyChange    = types.EventTypePolicyChange
	EventTypeSystemStartup   = types.EventTypeSystemStartup
	EventTypeSystemShutdown  = types.EventTypeSystemShutdown
	EventTypeServiceStart    = types.EventTypeServiceStart
	EventTypeServiceStop     = types.EventTypeServiceStop
	EventTypeMaintenanceMode = types.EventTypeMaintenanceMode

	EventTypeSecurityViolation = types.EventTypeSecurityViolation
	EventTypeIntrusionAttempt  = types.EventTypeIntrusionAttempt
	EventTypeMalwareDetection  = types.EventTypeMalwareDetection
	EventTypeAnomalyDetection  = types.EventTypeAnomalyDetection
	EventTypeComplianceCheck   = types.EventTypeComplianceCheck
	EventTypeVulnerability     = types.EventTypeVulnerability
	EventTypeIncidentResponse  = types.EventTypeIncidentResponse

	EventTypeNetworkAccess    = types.EventTypeNetworkAccess
	EventTypeFirewallRule     = types.EventTypeFirewallRule
	EventTypeNetworkAnomaly   = types.EventTypeNetworkAnomaly
	EventTypeResourceAccess   = types.EventTypeResourceAccess
	EventTypeCapacityChange   = types.EventTypeCapacityChange
	EventTypePerformanceAlert = types.EventTypePerformanceAlert

	EventTypeAPICall           = types.EventTypeAPICall
	EventTypeWorkflowExecution = types.EventTypeWorkflowExecution
	EventTypeJobExecution      = types.EventTypeJobExecution
	EventTypeDeployment        = types.EventTypeDeployment
	EventTypeRollback          = types.EventTypeRollback
	EventTypeHealthCheck       = types.EventTypeHealthCheck

	EventTypeIntentProcessing = types.EventTypeIntentProcessing
	EventTypeNetworkFunction  = types.EventTypeNetworkFunction
	EventTypeRICManagement    = types.EventTypeRICManagement
	EventTypeA1Interface      = types.EventTypeA1Interface
	EventTypeO1Interface      = types.EventTypeO1Interface
	EventTypeE2Interface      = types.EventTypeE2Interface
	EventTypeSliceManagement  = types.EventTypeSliceManagement

	ResultSuccess = types.ResultSuccess
	ResultFailure = types.ResultFailure
	ResultDenied  = types.ResultDenied
	ResultTimeout = types.ResultTimeout
	ResultError   = types.ResultError
	ResultPartial = types.ResultPartial

	ComplianceSOC2     = types.ComplianceSOC2
	ComplianceISO27001 = types.ComplianceISO27001
	CompliancePCIDSS   = types.CompliancePCIDSS
	ComplianceHIPAA    = types.ComplianceHIPAA
	ComplianceGDPR     = types.ComplianceGDPR
	ComplianceCCPA     = types.ComplianceCCPA
	ComplianceFISMA    = types.ComplianceFISMA
	ComplianceNIST     = types.ComplianceNIST
)

// Function aliases for backward compatibility
var (
	NewEventBuilder        = types.NewEventBuilder
	AuthenticationEvent    = types.AuthenticationEvent
	DataAccessEvent        = types.DataAccessEvent
	SecurityViolationEvent = types.SecurityViolationEvent
	SystemChangeEvent      = types.SystemChangeEvent
	FromJSON               = types.FromJSON
)

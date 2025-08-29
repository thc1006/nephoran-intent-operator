
package audit



// Re-export types from the types package for backward compatibility.

import (

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"

)



// Type aliases for backward compatibility.

type (

	Severity = types.Severity

	// EventType represents a eventtype.

	EventType = types.EventType

	// EventResult represents a eventresult.

	EventResult = types.EventResult

	// ComplianceStandard represents a compliancestandard.

	ComplianceStandard = types.ComplianceStandard

	// UserContext represents a usercontext.

	UserContext = types.UserContext

	// NetworkContext represents a networkcontext.

	NetworkContext = types.NetworkContext

	// SystemContext represents a systemcontext.

	SystemContext = types.SystemContext

	// ResourceContext represents a resourcecontext.

	ResourceContext = types.ResourceContext

	// AuditEvent represents a auditevent.

	AuditEvent = types.AuditEvent

	// EventBuilder represents a eventbuilder.

	EventBuilder = types.EventBuilder

)



// Constants re-exported for backward compatibility.

const (

	// SeverityEmergency holds severityemergency value.

	SeverityEmergency = types.SeverityEmergency

	// SeverityAlert holds severityalert value.

	SeverityAlert = types.SeverityAlert

	// SeverityCritical holds severitycritical value.

	SeverityCritical = types.SeverityCritical

	// SeverityError holds severityerror value.

	SeverityError = types.SeverityError

	// SeverityWarning holds severitywarning value.

	SeverityWarning = types.SeverityWarning

	// SeverityNotice holds severitynotice value.

	SeverityNotice = types.SeverityNotice

	// SeverityInfo holds severityinfo value.

	SeverityInfo = types.SeverityInfo

	// SeverityDebug holds severitydebug value.

	SeverityDebug = types.SeverityDebug



	// EventTypeAuthentication holds eventtypeauthentication value.

	EventTypeAuthentication = types.EventTypeAuthentication

	// EventTypeAuthenticationFailed holds eventtypeauthenticationfailed value.

	EventTypeAuthenticationFailed = types.EventTypeAuthenticationFailed

	// EventTypeAuthenticationSuccess holds eventtypeauthenticationsuccess value.

	EventTypeAuthenticationSuccess = types.EventTypeAuthenticationSuccess

	// EventTypeAuthorization holds eventtypeauthorization value.

	EventTypeAuthorization = types.EventTypeAuthorization

	// EventTypeAuthorizationFailed holds eventtypeauthorizationfailed value.

	EventTypeAuthorizationFailed = types.EventTypeAuthorizationFailed

	// EventTypeAuthorizationSuccess holds eventtypeauthorizationsuccess value.

	EventTypeAuthorizationSuccess = types.EventTypeAuthorizationSuccess

	// EventTypeSessionStart holds eventtypesessionstart value.

	EventTypeSessionStart = types.EventTypeSessionStart

	// EventTypeSessionEnd holds eventtypesessionend value.

	EventTypeSessionEnd = types.EventTypeSessionEnd

	// EventTypePasswordChange holds eventtypepasswordchange value.

	EventTypePasswordChange = types.EventTypePasswordChange

	// EventTypeTokenIssuance holds eventtypetokenissuance value.

	EventTypeTokenIssuance = types.EventTypeTokenIssuance

	// EventTypeTokenRevocation holds eventtypetokenrevocation value.

	EventTypeTokenRevocation = types.EventTypeTokenRevocation



	// EventTypeDataAccess holds eventtypedataaccess value.

	EventTypeDataAccess = types.EventTypeDataAccess

	// EventTypeDataCreate holds eventtypedatacreate value.

	EventTypeDataCreate = types.EventTypeDataCreate

	// EventTypeDataRead holds eventtypedataread value.

	EventTypeDataRead = types.EventTypeDataRead

	// EventTypeDataUpdate holds eventtypedataupdate value.

	EventTypeDataUpdate = types.EventTypeDataUpdate

	// EventTypeDataDelete holds eventtypedatadelete value.

	EventTypeDataDelete = types.EventTypeDataDelete

	// EventTypeDataExport holds eventtypedataexport value.

	EventTypeDataExport = types.EventTypeDataExport

	// EventTypeDataImport holds eventtypedataimport value.

	EventTypeDataImport = types.EventTypeDataImport

	// EventTypeDataBackup holds eventtypedatabackup value.

	EventTypeDataBackup = types.EventTypeDataBackup

	// EventTypeDataRestore holds eventtypedatarestore value.

	EventTypeDataRestore = types.EventTypeDataRestore

	// EventTypeDataProcessing holds eventtypedataprocessing value.

	EventTypeDataProcessing = types.EventTypeDataProcessing



	// EventTypeSystemChange holds eventtypesystemchange value.

	EventTypeSystemChange = types.EventTypeSystemChange

	// EventTypeConfigChange holds eventtypeconfigchange value.

	EventTypeConfigChange = types.EventTypeConfigChange

	// EventTypeUserManagement holds eventtypeusermanagement value.

	EventTypeUserManagement = types.EventTypeUserManagement

	// EventTypeRoleManagement holds eventtyperolemanagement value.

	EventTypeRoleManagement = types.EventTypeRoleManagement

	// EventTypePolicyChange holds eventtypepolicychange value.

	EventTypePolicyChange = types.EventTypePolicyChange

	// EventTypeSystemStartup holds eventtypesystemstartup value.

	EventTypeSystemStartup = types.EventTypeSystemStartup

	// EventTypeSystemShutdown holds eventtypesystemshutdown value.

	EventTypeSystemShutdown = types.EventTypeSystemShutdown

	// EventTypeServiceStart holds eventtypeservicestart value.

	EventTypeServiceStart = types.EventTypeServiceStart

	// EventTypeServiceStop holds eventtypeservicestop value.

	EventTypeServiceStop = types.EventTypeServiceStop

	// EventTypeMaintenanceMode holds eventtypemaintenancemode value.

	EventTypeMaintenanceMode = types.EventTypeMaintenanceMode



	// EventTypeSecurityViolation holds eventtypesecurityviolation value.

	EventTypeSecurityViolation = types.EventTypeSecurityViolation

	// EventTypeIntrusionAttempt holds eventtypeintrusionattempt value.

	EventTypeIntrusionAttempt = types.EventTypeIntrusionAttempt

	// EventTypeMalwareDetection holds eventtypemalwaredetection value.

	EventTypeMalwareDetection = types.EventTypeMalwareDetection

	// EventTypeAnomalyDetection holds eventtypeanomalydetection value.

	EventTypeAnomalyDetection = types.EventTypeAnomalyDetection

	// EventTypeComplianceCheck holds eventtypecompliancecheck value.

	EventTypeComplianceCheck = types.EventTypeComplianceCheck

	// EventTypeVulnerability holds eventtypevulnerability value.

	EventTypeVulnerability = types.EventTypeVulnerability

	// EventTypeIncidentResponse holds eventtypeincidentresponse value.

	EventTypeIncidentResponse = types.EventTypeIncidentResponse



	// EventTypeNetworkAccess holds eventtypenetworkaccess value.

	EventTypeNetworkAccess = types.EventTypeNetworkAccess

	// EventTypeFirewallRule holds eventtypefirewallrule value.

	EventTypeFirewallRule = types.EventTypeFirewallRule

	// EventTypeNetworkAnomaly holds eventtypenetworkanomaly value.

	EventTypeNetworkAnomaly = types.EventTypeNetworkAnomaly

	// EventTypeResourceAccess holds eventtyperesourceaccess value.

	EventTypeResourceAccess = types.EventTypeResourceAccess

	// EventTypeCapacityChange holds eventtypecapacitychange value.

	EventTypeCapacityChange = types.EventTypeCapacityChange

	// EventTypePerformanceAlert holds eventtypeperformancealert value.

	EventTypePerformanceAlert = types.EventTypePerformanceAlert



	// EventTypeAPICall holds eventtypeapicall value.

	EventTypeAPICall = types.EventTypeAPICall

	// EventTypeWorkflowExecution holds eventtypeworkflowexecution value.

	EventTypeWorkflowExecution = types.EventTypeWorkflowExecution

	// EventTypeJobExecution holds eventtypejobexecution value.

	EventTypeJobExecution = types.EventTypeJobExecution

	// EventTypeDeployment holds eventtypedeployment value.

	EventTypeDeployment = types.EventTypeDeployment

	// EventTypeRollback holds eventtyperollback value.

	EventTypeRollback = types.EventTypeRollback

	// EventTypeHealthCheck holds eventtypehealthcheck value.

	EventTypeHealthCheck = types.EventTypeHealthCheck



	// EventTypeIntentProcessing holds eventtypeintentprocessing value.

	EventTypeIntentProcessing = types.EventTypeIntentProcessing

	// EventTypeNetworkFunction holds eventtypenetworkfunction value.

	EventTypeNetworkFunction = types.EventTypeNetworkFunction

	// EventTypeRICManagement holds eventtypericmanagement value.

	EventTypeRICManagement = types.EventTypeRICManagement

	// EventTypeA1Interface holds eventtypea1interface value.

	EventTypeA1Interface = types.EventTypeA1Interface

	// EventTypeO1Interface holds eventtypeo1interface value.

	EventTypeO1Interface = types.EventTypeO1Interface

	// EventTypeE2Interface holds eventtypee2interface value.

	EventTypeE2Interface = types.EventTypeE2Interface

	// EventTypeSliceManagement holds eventtypeslicemanagement value.

	EventTypeSliceManagement = types.EventTypeSliceManagement



	// ResultSuccess holds resultsuccess value.

	ResultSuccess = types.ResultSuccess

	// ResultFailure holds resultfailure value.

	ResultFailure = types.ResultFailure

	// ResultDenied holds resultdenied value.

	ResultDenied = types.ResultDenied

	// ResultTimeout holds resulttimeout value.

	ResultTimeout = types.ResultTimeout

	// ResultError holds resulterror value.

	ResultError = types.ResultError

	// ResultPartial holds resultpartial value.

	ResultPartial = types.ResultPartial



	// ComplianceSOC2 holds compliancesoc2 value.

	ComplianceSOC2 = types.ComplianceSOC2

	// ComplianceISO27001 holds complianceiso27001 value.

	ComplianceISO27001 = types.ComplianceISO27001

	// CompliancePCIDSS holds compliancepcidss value.

	CompliancePCIDSS = types.CompliancePCIDSS

	// ComplianceHIPAA holds compliancehipaa value.

	ComplianceHIPAA = types.ComplianceHIPAA

	// ComplianceGDPR holds compliancegdpr value.

	ComplianceGDPR = types.ComplianceGDPR

	// ComplianceCCPA holds complianceccpa value.

	ComplianceCCPA = types.ComplianceCCPA

	// ComplianceFISMA holds compliancefisma value.

	ComplianceFISMA = types.ComplianceFISMA

	// ComplianceNIST holds compliancenist value.

	ComplianceNIST = types.ComplianceNIST

)



// Function aliases for backward compatibility.

var (

	// NewEventBuilder holds neweventbuilder value.

	NewEventBuilder = types.NewEventBuilder

	// AuthenticationEvent holds authenticationevent value.

	AuthenticationEvent = types.AuthenticationEvent

	// DataAccessEvent holds dataaccessevent value.

	DataAccessEvent = types.DataAccessEvent

	// SecurityViolationEvent holds securityviolationevent value.

	SecurityViolationEvent = types.SecurityViolationEvent

	// SystemChangeEvent holds systemchangeevent value.

	SystemChangeEvent = types.SystemChangeEvent

	// FromJSON holds fromjson value.

	FromJSON = types.FromJSON

)


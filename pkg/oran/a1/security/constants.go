// Package security provides security constants and types shared across security modules.
package security

// ComplianceStandard represents compliance standards for audit and security.
type ComplianceStandard string

const (
	// Compliance Standards.
	ComplianceNone ComplianceStandard = "none"
	// ComplianceSOC2 holds compliancesoc2 value.
	ComplianceSOC2 ComplianceStandard = "soc2"
	// ComplianceISO27001 holds complianceiso27001 value.
	ComplianceISO27001 ComplianceStandard = "iso27001"
	// CompliancePCIDSS holds compliancepcidss value.
	CompliancePCIDSS ComplianceStandard = "pci-dss"
	// ComplianceHIPAA holds compliancehipaa value.
	ComplianceHIPAA ComplianceStandard = "hipaa"
	// ComplianceGDPR holds compliancegdpr value.
	ComplianceGDPR ComplianceStandard = "gdpr"
	// ComplianceNIST holds compliancenist value.
	ComplianceNIST ComplianceStandard = "nist"
	// ComplianceFIPS140 holds compliancefips140 value.
	ComplianceFIPS140 ComplianceStandard = "fips-140-2"
	// ComplianceORANWG11 holds complianceoranwg11 value.
	ComplianceORANWG11 ComplianceStandard = "oran-wg11"
)

// SecurityLevel represents security levels.
type SecurityLevel string

const (
	// SecurityLevelLow holds securitylevellow value.
	SecurityLevelLow SecurityLevel = "low"
	// SecurityLevelMedium holds securitylevelmedium value.
	SecurityLevelMedium SecurityLevel = "medium"
	// SecurityLevelHigh holds securitylevelhigh value.
	SecurityLevelHigh SecurityLevel = "high"
	// SecurityLevelCritical holds securitylevelcritical value.
	SecurityLevelCritical SecurityLevel = "critical"
)

// ThreatLevel represents threat assessment levels.
type ThreatLevel string

const (
	// ThreatLevelNone holds threatlevelnone value.
	ThreatLevelNone ThreatLevel = "none"
	// ThreatLevelLow holds threatlevellow value.
	ThreatLevelLow ThreatLevel = "low"
	// ThreatLevelModerate holds threatlevelmoderate value.
	ThreatLevelModerate ThreatLevel = "moderate"
	// ThreatLevelHigh holds threatlevelhigh value.
	ThreatLevelHigh ThreatLevel = "high"
	// ThreatLevelCritical holds threatlevelcritical value.
	ThreatLevelCritical ThreatLevel = "critical"
)

// ViolationType represents security violation types.
type ViolationType string

const (
	// ViolationTypeAuthFailure holds violationtypeauthfailure value.
	ViolationTypeAuthFailure ViolationType = "auth_failure"
	// ViolationTypeUnauthorizedAccess holds violationtypeunauthorizedaccess value.
	ViolationTypeUnauthorizedAccess ViolationType = "unauthorized_access"
	// ViolationTypeRateLimitExceeded holds violationtyperatelimitexceeded value.
	ViolationTypeRateLimitExceeded ViolationType = "rate_limit_exceeded"
	// ViolationTypeSQLInjection holds violationtypesqlinjection value.
	ViolationTypeSQLInjection ViolationType = "sql_injection"
	// ViolationTypeXSS holds violationtypexss value.
	ViolationTypeXSS ViolationType = "xss"
	// ViolationTypePathTraversal holds violationtypepathtraversal value.
	ViolationTypePathTraversal ViolationType = "path_traversal"
	// ViolationTypeMaliciousPayload holds violationtypemaliciouspayload value.
	ViolationTypeMaliciousPayload ViolationType = "malicious_payload"
	// ViolationTypeDataExfiltration holds violationtypedataexfiltration value.
	ViolationTypeDataExfiltration ViolationType = "data_exfiltration"
	// ViolationTypeBruteForce holds violationtypebruteforce value.
	ViolationTypeBruteForce ViolationType = "brute_force"
	// ViolationTypePolicyViolation holds violationtypepolicyviolation value.
	ViolationTypePolicyViolation ViolationType = "policy_violation"
)

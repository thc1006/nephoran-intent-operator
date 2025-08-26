// Package security provides security constants and types shared across security modules
package security

// ComplianceStandard represents compliance standards for audit and security
type ComplianceStandard string

const (
	// Compliance Standards
	ComplianceNone     ComplianceStandard = "none"
	ComplianceSOC2     ComplianceStandard = "soc2"
	ComplianceISO27001 ComplianceStandard = "iso27001"
	CompliancePCIDSS   ComplianceStandard = "pci-dss"
	ComplianceHIPAA    ComplianceStandard = "hipaa"
	ComplianceGDPR     ComplianceStandard = "gdpr"
	ComplianceNIST     ComplianceStandard = "nist"
	ComplianceFIPS140  ComplianceStandard = "fips-140-2"
	ComplianceORANWG11 ComplianceStandard = "oran-wg11"
)

// SecurityLevel represents security levels
type SecurityLevel string

const (
	SecurityLevelLow      SecurityLevel = "low"
	SecurityLevelMedium   SecurityLevel = "medium"
	SecurityLevelHigh     SecurityLevel = "high"
	SecurityLevelCritical SecurityLevel = "critical"
)

// ThreatLevel represents threat assessment levels
type ThreatLevel string

const (
	ThreatLevelNone     ThreatLevel = "none"
	ThreatLevelLow      ThreatLevel = "low"
	ThreatLevelModerate ThreatLevel = "moderate"
	ThreatLevelHigh     ThreatLevel = "high"
	ThreatLevelCritical ThreatLevel = "critical"
)

// ViolationType represents security violation types
type ViolationType string

const (
	ViolationTypeAuthFailure       ViolationType = "auth_failure"
	ViolationTypeUnauthorizedAccess ViolationType = "unauthorized_access"
	ViolationTypeRateLimitExceeded  ViolationType = "rate_limit_exceeded"
	ViolationTypeSQLInjection      ViolationType = "sql_injection"
	ViolationTypeXSS               ViolationType = "xss"
	ViolationTypePathTraversal     ViolationType = "path_traversal"
	ViolationTypeMaliciousPayload  ViolationType = "malicious_payload"
	ViolationTypeDataExfiltration  ViolationType = "data_exfiltration"
	ViolationTypeBruteForce        ViolationType = "brute_force"
	ViolationTypePolicyViolation   ViolationType = "policy_violation"
)
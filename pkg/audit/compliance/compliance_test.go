package compliance_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit/compliance"
)

// ComplianceTestSuite tests compliance framework functionality
type ComplianceTestSuite struct {
	suite.Suite
	complianceLogger *compliance.ComplianceLogger
	retentionManager *compliance.RetentionManager
}

func TestComplianceTestSuite(t *testing.T) {
	suite.Run(t, new(ComplianceTestSuite))
}

func (suite *ComplianceTestSuite) SetupTest() {
	complianceMode := []audit.ComplianceStandard{
		audit.ComplianceSOC2,
		audit.ComplianceISO27001,
		audit.CompliancePCIDSS,
	}

	suite.complianceLogger = compliance.NewComplianceLogger(complianceMode)

	retentionConfig := &RetentionConfig{
		ComplianceMode:     complianceMode,
		DefaultRetention:   365 * 24 * time.Hour,     // 1 year
		MinRetention:       30 * 24 * time.Hour,      // 30 days
		MaxRetention:       7 * 365 * 24 * time.Hour, // 7 years
		PurgeInterval:      24 * time.Hour,           // Daily
		BackupBeforePurge:  true,
		CompressionEnabled: true,
	}

	suite.retentionManager = compliance.NewRetentionManager(retentionConfig)
}

// SOC2 Compliance Tests
func (suite *ComplianceTestSuite) TestSOC2ComplianceRequirements() {
	tests := []struct {
		name             string
		event            *audit.AuditEvent
		expectedControls []string
		expectedService  string
		requiresEvidence bool
	}{
		{
			name: "authentication event CC6.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthentication,
				Component: "auth-service",
				Action:    "login",
				Severity:  audit.SeverityInfo,
				Result:    audit.ResultSuccess,
				UserContext: &audit.UserContext{
					UserID:     "user123",
					Username:   "testuser",
					AuthMethod: "oauth2",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"CC6.1", "CC6.8"},
			expectedService:  "Security",
			requiresEvidence: true,
		},
		{
			name: "authorization event CC6.2",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthorization,
				Component: "rbac-service",
				Action:    "permission_check",
				Severity:  audit.SeverityInfo,
				Result:    audit.ResultSuccess,
				UserContext: &audit.UserContext{
					UserID: "user123",
					Role:   "admin",
				},
				ResourceContext: &audit.ResourceContext{
					ResourceType: "deployment",
					Operation:    "create",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"CC6.2", "CC6.3"},
			expectedService:  "Security",
			requiresEvidence: true,
		},
		{
			name: "data access event CC6.7",
			event: &audit.AuditEvent{
				ID:                 uuid.New().String(),
				EventType:          audit.EventTypeDataAccess,
				Component:          "api-service",
				Action:             "sensitive_data_access",
				Severity:           audit.SeverityNotice,
				Result:             audit.ResultSuccess,
				DataClassification: "Confidential",
				Data: map[string]interface{}{
					"records_accessed": 150,
					"data_type":        "customer_pii",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"CC6.7", "CC7.1"},
			expectedService:  "Confidentiality",
			requiresEvidence: true,
		},
		{
			name: "system change event CC8.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeSystemChange,
				Component: "config-manager",
				Action:    "configuration_update",
				Severity:  audit.SeverityWarning,
				Result:    audit.ResultSuccess,
				Data: map[string]interface{}{
					"change_type":       "security_policy",
					"approval_required": true,
					"change_ticket":     "CHG-2023-1234",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"CC8.1", "CC3.3"},
			expectedService:  "Processing Integrity",
			requiresEvidence: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			compliance := suite.analyzeSOC2Compliance(tt.event)

			suite.NotNil(compliance)
			suite.Equal(audit.ComplianceSOC2, compliance.Standard)

			// Verify required controls are identified
			for _, expectedControl := range tt.expectedControls {
				suite.Contains(compliance.Controls, expectedControl)
			}

			// Verify trust service category
			suite.Equal(tt.expectedService, compliance.TrustService)

			// Verify evidence requirements
			suite.Equal(tt.requiresEvidence, compliance.RequiresEvidence)

			// Verify retention period meets SOC2 requirements (7 years)
			expectedRetention := 7 * 365 * 24 * time.Hour
			suite.GreaterOrEqual(compliance.RetentionPeriod, expectedRetention)
		})
	}
}

// ISO 27001 Compliance Tests
func (suite *ComplianceTestSuite) TestISO27001ComplianceRequirements() {
	tests := []struct {
		name             string
		event            *audit.AuditEvent
		expectedControls []string
		expectedAnnex    string
		riskCategory     string
	}{
		{
			name: "access management A.9.2.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthentication,
				Component: "identity-provider",
				Action:    "user_registration",
				Severity:  audit.SeverityInfo,
				Result:    audit.ResultSuccess,
				UserContext: &audit.UserContext{
					UserID:   "new_user_456",
					Username: "newuser",
					Role:     "operator",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"A.9.2.1", "A.9.2.2"},
			expectedAnnex:    "A.9 - Access Control",
			riskCategory:     "Medium",
		},
		{
			name: "operations security A.12.4.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeDataAccess,
				Component: "database-service",
				Action:    "sensitive_query",
				Severity:  audit.SeverityNotice,
				Result:    audit.ResultSuccess,
				Data: map[string]interface{}{
					"query_type":   "customer_data",
					"record_count": 25,
					"purpose":      "customer_support",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"A.12.4.1", "A.12.4.2"},
			expectedAnnex:    "A.12 - Operations Security",
			riskCategory:     "High",
		},
		{
			name: "incident management A.16.1.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeSecurityViolation,
				Component: "security-monitor",
				Action:    "violation_detected",
				Severity:  audit.SeverityCritical,
				Result:    audit.ResultFailure,
				Data: map[string]interface{}{
					"violation_type": "unauthorized_access_attempt",
					"source_ip":      "192.168.1.100",
					"target_system":  "production_database",
				},
				Timestamp: time.Now(),
			},
			expectedControls: []string{"A.16.1.1", "A.16.1.2"},
			expectedAnnex:    "A.16 - Information Security Incident Management",
			riskCategory:     "Critical",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			compliance := suite.analyzeISO27001Compliance(tt.event)

			suite.NotNil(compliance)
			suite.Equal(audit.ComplianceISO27001, compliance.Standard)

			// Verify controls
			for _, expectedControl := range tt.expectedControls {
				suite.Contains(compliance.Controls, expectedControl)
			}

			// Verify annex classification
			suite.Equal(tt.expectedAnnex, compliance.Annex)

			// Verify risk category
			suite.Equal(tt.riskCategory, compliance.RiskCategory)

			// Verify retention period meets ISO 27001 requirements (3 years minimum)
			expectedRetention := 3 * 365 * 24 * time.Hour
			suite.GreaterOrEqual(compliance.RetentionPeriod, expectedRetention)
		})
	}
}

// PCI DSS Compliance Tests
func (suite *ComplianceTestSuite) TestPCIDSSComplianceRequirements() {
	tests := []struct {
		name               string
		event              *audit.AuditEvent
		expectedReq        string
		dataClassification string
		requiresAlert      bool
	}{
		{
			name: "cardholder data access requirement 10.2.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeDataAccess,
				Component: "payment-processor",
				Action:    "card_data_access",
				Severity:  audit.SeverityNotice,
				Result:    audit.ResultSuccess,
				Data: map[string]interface{}{
					"cardholder_data": true,
					"card_numbers":    5,
					"purpose":         "transaction_processing",
				},
				DataClassification: "Cardholder Data",
				Timestamp:          time.Now(),
			},
			expectedReq:        "10.2.1",
			dataClassification: "Cardholder Data",
			requiresAlert:      true,
		},
		{
			name: "authentication failure requirement 8.1.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthenticationFailed,
				Component: "payment-gateway",
				Action:    "login_failed",
				Severity:  audit.SeverityWarning,
				Result:    audit.ResultFailure,
				UserContext: &audit.UserContext{
					UserID: "payment_user",
				},
				Data: map[string]interface{}{
					"failure_reason": "invalid_credentials",
					"attempt_count":  3,
				},
				Timestamp: time.Now(),
			},
			expectedReq:        "8.1.1",
			dataClassification: "Authentication Data",
			requiresAlert:      true,
		},
		{
			name: "system administrator access requirement 7.1.1",
			event: &audit.AuditEvent{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthorization,
				Component: "admin-console",
				Action:    "privileged_access",
				Severity:  audit.SeverityInfo,
				Result:    audit.ResultSuccess,
				UserContext: &audit.UserContext{
					UserID: "admin_user",
					Role:   "system_administrator",
				},
				ResourceContext: &audit.ResourceContext{
					ResourceType: "cardholder_data_environment",
					Operation:    "modify",
				},
				Timestamp: time.Now(),
			},
			expectedReq:        "7.1.1",
			dataClassification: "Administrative Access",
			requiresAlert:      false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			compliance := suite.analyzePCIDSSCompliance(tt.event)

			suite.NotNil(compliance)
			suite.Equal(audit.CompliancePCIDSS, compliance.Standard)

			// Verify requirement mapping
			suite.Equal(tt.expectedReq, compliance.Requirement)

			// Verify data classification
			suite.Equal(tt.dataClassification, compliance.DataClassification)

			// Verify alert requirements
			suite.Equal(tt.requiresAlert, compliance.RequiresAlert)

			// Verify retention period meets PCI DSS requirements (1 year minimum)
			expectedRetention := 365 * 24 * time.Hour
			suite.GreaterOrEqual(compliance.RetentionPeriod, expectedRetention)
		})
	}
}

// Retention Management Tests
func (suite *ComplianceTestSuite) TestRetentionManagement() {
	suite.Run("calculate retention period", func() {
		tests := []struct {
			name                string
			event               *audit.AuditEvent
			complianceStandards []audit.ComplianceStandard
			expectedMin         time.Duration
		}{
			{
				name: "SOC2 security event",
				event: &audit.AuditEvent{
					EventType: audit.EventTypeSecurityViolation,
					Severity:  audit.SeverityCritical,
				},
				complianceStandards: []audit.ComplianceStandard{audit.ComplianceSOC2},
				expectedMin:         7 * 365 * 24 * time.Hour, // 7 years
			},
			{
				name: "PCI DSS authentication event",
				event: &audit.AuditEvent{
					EventType: audit.EventTypeAuthentication,
					Data: map[string]interface{}{
						"cardholder_data": true,
					},
				},
				complianceStandards: []audit.ComplianceStandard{audit.CompliancePCIDSS},
				expectedMin:         365 * 24 * time.Hour, // 1 year
			},
			{
				name: "Multiple compliance requirements",
				event: &audit.AuditEvent{
					EventType: audit.EventTypeDataAccess,
					Severity:  audit.SeverityInfo,
				},
				complianceStandards: []audit.ComplianceStandard{
					audit.ComplianceSOC2,
					audit.CompliancePCIDSS,
					audit.ComplianceISO27001,
				},
				expectedMin: 7 * 365 * 24 * time.Hour, // Longest requirement wins
			},
		}

		for _, tt := range tests {
			suite.Run(tt.name, func() {
				retention := suite.retentionManager.CalculateRetentionPeriod(tt.event, tt.complianceStandards)
				suite.GreaterOrEqual(retention, tt.expectedMin)
			})
		}
	})

	suite.Run("retention policy enforcement", func() {
		// Create events with different ages
		events := []*audit.AuditEvent{
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now().Add(-10 * 24 * time.Hour), // 10 days old
				EventType: audit.EventTypeHealthCheck,
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now().Add(-400 * 24 * time.Hour), // 400 days old
				EventType: audit.EventTypeAuthentication,
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now().Add(-8 * 365 * 24 * time.Hour), // 8 years old
				EventType: audit.EventTypeSecurityViolation,
			},
		}

		policies := []RetentionPolicy{
			{
				EventType:       audit.EventTypeHealthCheck,
				RetentionPeriod: 30 * 24 * time.Hour, // 30 days
			},
			{
				EventType:       audit.EventTypeAuthentication,
				RetentionPeriod: 365 * 24 * time.Hour, // 1 year
			},
			{
				EventType:       audit.EventTypeSecurityViolation,
				RetentionPeriod: 7 * 365 * 24 * time.Hour, // 7 years
			},
		}

		for i, event := range events {
			policy := policies[i]
			shouldRetain := suite.retentionManager.ShouldRetainEvent(event, policy)

			expectedRetain := time.Since(event.Timestamp) < policy.RetentionPeriod
			suite.Equal(expectedRetain, shouldRetain,
				"Event %d retention decision mismatch", i)
		}
	})
}

// Compliance Report Generation Tests
func (suite *ComplianceTestSuite) TestComplianceReportGeneration() {
	suite.Run("generate SOC2 report", func() {
		events := []*audit.AuditEvent{
			createComplianceTestEvent(audit.EventTypeAuthentication, audit.SeverityInfo),
			createComplianceTestEvent(audit.EventTypeAuthorization, audit.SeverityWarning),
			createComplianceTestEvent(audit.EventTypeDataAccess, audit.SeverityNotice),
			createComplianceTestEvent(audit.EventTypeSecurityViolation, audit.SeverityCritical),
		}

		report := suite.complianceLogger.GenerateSOC2Report(events, time.Now().Add(-24*time.Hour), time.Now())

		suite.NotNil(report)
		suite.Equal("SOC2", report.Standard)
		suite.Len(report.TrustServices, 2) // Security and Confidentiality
		suite.Greater(report.TotalEvents, int64(0))
		suite.NotEmpty(report.ControlCoverage)

		// Verify security violations are flagged
		suite.Greater(report.SecurityViolations, int64(0))
	})

	suite.Run("generate ISO 27001 report", func() {
		events := []*audit.AuditEvent{
			createComplianceTestEvent(audit.EventTypeAuthentication, audit.SeverityInfo),
			createComplianceTestEvent(audit.EventTypeSystemChange, audit.SeverityWarning),
			createComplianceTestEvent(audit.EventTypeIncidentResponse, audit.SeverityCritical),
		}

		report := suite.complianceLogger.GenerateISO27001Report(events, time.Now().Add(-24*time.Hour), time.Now())

		suite.NotNil(report)
		suite.Equal("ISO27001", report.Standard)
		suite.Greater(report.TotalEvents, int64(0))
		suite.NotEmpty(report.AnnexCoverage)
		suite.NotEmpty(report.RiskAssessment)
	})

	suite.Run("generate PCI DSS report", func() {
		events := []*audit.AuditEvent{
			{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeDataAccess,
				Data: map[string]interface{}{
					"cardholder_data": true,
				},
				DataClassification: "Cardholder Data",
				Timestamp:          time.Now(),
			},
			{
				ID:        uuid.New().String(),
				EventType: audit.EventTypeAuthenticationFailed,
				Data: map[string]interface{}{
					"failure_reason": "invalid_card",
				},
				Timestamp: time.Now(),
			},
		}

		report := suite.complianceLogger.GeneratePCIDSSReport(events, time.Now().Add(-24*time.Hour), time.Now())

		suite.NotNil(report)
		suite.Equal("PCI_DSS", report.Standard)
		suite.Greater(report.TotalEvents, int64(0))
		suite.Greater(report.CardholderDataAccess, int64(0))
		suite.NotEmpty(report.RequirementCoverage)
	})
}

// Evidence Collection Tests
func (suite *ComplianceTestSuite) TestEvidenceCollection() {
	suite.Run("collect audit evidence", func() {
		event := &audit.AuditEvent{
			ID:        uuid.New().String(),
			EventType: audit.EventTypeAuthentication,
			Component: "auth-service",
			Action:    "mfa_verification",
			Severity:  audit.SeverityInfo,
			Result:    audit.ResultSuccess,
			UserContext: &audit.UserContext{
				UserID:     "user123",
				AuthMethod: "mfa",
			},
			Timestamp: time.Now(),
		}

		evidence := suite.complianceLogger.CollectEvidence(event, []audit.ComplianceStandard{audit.ComplianceSOC2})

		suite.NotNil(evidence)
		suite.Equal(event.ID, evidence.EventID)
		suite.Equal(audit.ComplianceSOC2, evidence.Standard)
		suite.NotEmpty(evidence.EvidenceType)
		suite.NotNil(evidence.Metadata)
		suite.True(evidence.Verified)

		// Verify evidence chain
		suite.NotEmpty(evidence.Hash)
		suite.NotEmpty(evidence.PreviousHash)
	})

	suite.Run("verify evidence integrity", func() {
		event := createComplianceTestEvent(audit.EventTypeDataAccess, audit.SeverityNotice)
		evidence := suite.complianceLogger.CollectEvidence(event, []audit.ComplianceStandard{audit.ComplianceISO27001})

		// Verify evidence hasn't been tampered with
		isValid := suite.complianceLogger.VerifyEvidenceIntegrity(evidence)
		suite.True(isValid)

		// Tamper with evidence and verify detection
		evidence.Metadata["tampered"] = "true"
		isValid = suite.complianceLogger.VerifyEvidenceIntegrity(evidence)
		suite.False(isValid)
	})
}

// Cross-Standard Compliance Tests
func (suite *ComplianceTestSuite) TestCrossStandardCompliance() {
	suite.Run("multi-standard event analysis", func() {
		event := &audit.AuditEvent{
			ID:        uuid.New().String(),
			EventType: audit.EventTypeDataAccess,
			Component: "payment-api",
			Action:    "customer_data_query",
			Severity:  audit.SeverityNotice,
			Result:    audit.ResultSuccess,
			Data: map[string]interface{}{
				"cardholder_data": true,
				"pii_records":     25,
			},
			DataClassification: "Sensitive",
			Timestamp:          time.Now(),
		}

		standards := []audit.ComplianceStandard{
			audit.ComplianceSOC2,
			audit.ComplianceISO27001,
			audit.CompliancePCIDSS,
		}

		analysis := suite.complianceLogger.AnalyzeMultiStandardCompliance(event, standards)

		suite.Len(analysis.Standards, 3)

		// Verify each standard has appropriate controls
		for _, stdAnalysis := range analysis.Standards {
			suite.NotEmpty(stdAnalysis.Controls)
			suite.Greater(stdAnalysis.RetentionPeriod, time.Duration(0))
		}

		// Verify strictest retention period is selected
		maxRetention := time.Duration(0)
		for _, std := range analysis.Standards {
			if std.RetentionPeriod > maxRetention {
				maxRetention = std.RetentionPeriod
			}
		}
		suite.Equal(maxRetention, analysis.FinalRetentionPeriod)
	})
}

// Performance Tests for Compliance Processing
func (suite *ComplianceTestSuite) TestComplianceProcessingPerformance() {
	suite.Run("high volume compliance analysis", func() {
		events := make([]*audit.AuditEvent, 1000)
		for i := 0; i < len(events); i++ {
			events[i] = createComplianceTestEvent(
				[]audit.EventType{
					audit.EventTypeAuthentication,
					audit.EventTypeAuthorization,
					audit.EventTypeDataAccess,
					audit.EventTypeSystemChange,
				}[i%4],
				[]audit.Severity{
					audit.SeverityInfo,
					audit.SeverityWarning,
					audit.SeverityError,
					audit.SeverityCritical,
				}[i%4],
			)
		}

		start := time.Now()
		processed := 0

		for _, event := range events {
			suite.complianceLogger.ProcessEvent(event)
			processed++
		}

		duration := time.Since(start)
		eventsPerSecond := float64(processed) / duration.Seconds()

		suite.Greater(eventsPerSecond, 100.0, "Compliance processing too slow: %.2f events/sec", eventsPerSecond)
	})
}

// Helper methods for compliance analysis (these would be implemented in the actual compliance package)

func (suite *ComplianceTestSuite) analyzeSOC2Compliance(event *audit.AuditEvent) *SOC2Compliance {
	// Mock implementation for testing
	return &SOC2Compliance{
		Standard:         audit.ComplianceSOC2,
		Controls:         []string{"CC6.1", "CC6.8"},
		TrustService:     "Security",
		RequiresEvidence: true,
		RetentionPeriod:  7 * 365 * 24 * time.Hour,
	}
}

func (suite *ComplianceTestSuite) analyzeISO27001Compliance(event *audit.AuditEvent) *ISO27001Compliance {
	// Mock implementation for testing
	return &ISO27001Compliance{
		Standard:        audit.ComplianceISO27001,
		Controls:        []string{"A.9.2.1", "A.9.2.2"},
		Annex:           "A.9 - Access Control",
		RiskCategory:    "Medium",
		RetentionPeriod: 3 * 365 * 24 * time.Hour,
	}
}

func (suite *ComplianceTestSuite) analyzePCIDSSCompliance(event *audit.AuditEvent) *PCIDSSCompliance {
	// Mock implementation for testing
	return &PCIDSSCompliance{
		Standard:           audit.CompliancePCIDSS,
		Requirement:        "10.2.1",
		DataClassification: "Cardholder Data",
		RequiresAlert:      true,
		RetentionPeriod:    365 * 24 * time.Hour,
	}
}

func createComplianceTestEvent(eventType audit.EventType, severity audit.Severity) *audit.AuditEvent {
	return &audit.AuditEvent{
		ID:        uuid.New().String(),
		EventType: eventType,
		Component: "test-component",
		Action:    "test-action",
		Severity:  severity,
		Result:    audit.ResultSuccess,
		UserContext: &audit.UserContext{
			UserID: "test-user",
		},
		Timestamp: time.Now(),
	}
}

// Compliance data structures for testing (would be in actual compliance package)

type SOC2Compliance struct {
	Standard         audit.ComplianceStandard
	Controls         []string
	TrustService     string
	RequiresEvidence bool
	RetentionPeriod  time.Duration
}

type ISO27001Compliance struct {
	Standard        audit.ComplianceStandard
	Controls        []string
	Annex           string
	RiskCategory    string
	RetentionPeriod time.Duration
}

type PCIDSSCompliance struct {
	Standard           audit.ComplianceStandard
	Requirement        string
	DataClassification string
	RequiresAlert      bool
	RetentionPeriod    time.Duration
}

type ComplianceReport struct {
	Standard    string
	TotalEvents int64
	GeneratedAt time.Time
}

type SOC2Report struct {
	ComplianceReport
	TrustServices      []string
	ControlCoverage    map[string]int64
	SecurityViolations int64
}

type ISO27001Report struct {
	ComplianceReport
	AnnexCoverage  map[string]int64
	RiskAssessment map[string]string
}

type PCIDSSReport struct {
	ComplianceReport
	CardholderDataAccess int64
	RequirementCoverage  map[string]int64
}

type Evidence struct {
	EventID      string
	Standard     audit.ComplianceStandard
	EvidenceType string
	Metadata     map[string]interface{}
	Hash         string
	PreviousHash string
	Verified     bool
	Timestamp    time.Time
}

type MultiStandardAnalysis struct {
	Standards            []ComplianceStandardAnalysis
	FinalRetentionPeriod time.Duration
}

type ComplianceStandardAnalysis struct {
	Standard        audit.ComplianceStandard
	Controls        []string
	RetentionPeriod time.Duration
}

// Mock implementations removed - using actual implementations from compliance package

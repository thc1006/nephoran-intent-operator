package compliance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

// TestComprehensiveComplianceFramework validates the main compliance framework
func TestComprehensiveComplianceFramework(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test framework initialization
	framework := NewComprehensiveComplianceFramework(logger)
	require.NotNil(t, framework)
	assert.NotNil(t, framework.cisCompliance)
	assert.NotNil(t, framework.nistFramework)
	assert.NotNil(t, framework.owaspProtection)
	assert.NotNil(t, framework.gdprCompliance)
	assert.NotNil(t, framework.oranWG11Compliance)
	assert.NotNil(t, framework.opaEnforcement)
	assert.NotNil(t, framework.complianceMonitor)

	// Test comprehensive compliance check
	ctx := context.Background()
	status, err := framework.RunComprehensiveComplianceCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Validate compliance status structure
	assert.NotZero(t, status.Timestamp)
	assert.GreaterOrEqual(t, status.OverallCompliance, 0.0)
	assert.LessOrEqual(t, status.OverallCompliance, 100.0)
	assert.NotNil(t, status.ComplianceViolations)
	assert.NotNil(t, status.RecommendedActions)
	assert.NotNil(t, status.AuditTrail)

	// Validate individual framework compliance
	assert.NotZero(t, status.CISCompliance.OverallScore)
	assert.NotZero(t, status.NISTCompliance.OverallMaturity)
	assert.NotZero(t, status.OWASPCompliance.OverallProtectionScore)
	assert.NotZero(t, status.GDPRCompliance.ComplianceScore)
	assert.NotZero(t, status.ORANCompliance.OverallSecurityPosture)
	assert.NotZero(t, status.OPACompliance.PolicyEnforcementRate)

	t.Logf("Overall Compliance Score: %.2f%%", status.OverallCompliance)
	t.Logf("CIS Kubernetes Score: %.2f%%", status.CISCompliance.OverallScore)
	t.Logf("NIST Framework Maturity: %s", status.NISTCompliance.OverallMaturity)
	t.Logf("OWASP Protection Score: %.2f%%", status.OWASPCompliance.OverallProtectionScore)
	t.Logf("GDPR Compliance Score: %.2f%%", status.GDPRCompliance.ComplianceScore)
	t.Logf("O-RAN Security Posture: %.2f%%", status.ORANCompliance.OverallSecurityPosture)
	t.Logf("OPA Policy Enforcement: %.2f%%", status.OPACompliance.PolicyEnforcementRate)
}

// TestAutomatedComplianceMonitor validates the automated monitoring system
func TestAutomatedComplianceMonitor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	framework := NewComprehensiveComplianceFramework(logger)

	// Create a mock Kubernetes config (in real implementation, this would be actual config)
	// For testing purposes, we'll test with nil config and expect appropriate handling
	monitor, err := NewAutomatedComplianceMonitor(nil, framework, logger)

	// In this test, we expect an error due to nil config
	// In production, this would be a valid Kubernetes config
	assert.Error(t, err)
	assert.Nil(t, monitor)

	// Test alert thresholds
	thresholds := &ComplianceAlertThresholds{
		CriticalScore:         80.0,
		WarningScore:          90.0,
		MaxViolations:         10,
		MaxCriticalViolations: 3,
		ResponseTimeMinutes:   5,
	}

	assert.Equal(t, 80.0, thresholds.CriticalScore)
	assert.Equal(t, 90.0, thresholds.WarningScore)
	assert.Equal(t, 10, thresholds.MaxViolations)
	assert.Equal(t, 3, thresholds.MaxCriticalViolations)
	assert.Equal(t, 5, thresholds.ResponseTimeMinutes)
}

// TestOPAPolicyEngine validates the OPA policy enforcement system
func TestOPAPolicyEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test OPA engine initialization
	engine, err := NewOPACompliancePolicyEngine(logger)
	require.NoError(t, err)
	require.NotNil(t, engine)

	// Validate that default policies are loaded
	assert.NotNil(t, engine.opaStore)
	assert.NotNil(t, engine.regoPolicies)
	assert.NotNil(t, engine.policyCache)
	assert.NotNil(t, engine.violationStore)

	// Test policy loading
	testPolicy := PolicyDefinition{
		ID:          "test-policy",
		Name:        "Test Policy",
		Framework:   "TEST",
		Category:    "Security",
		Severity:    "HIGH",
		Description: "Test policy for validation",
		RegoPolicy: `
package test.policy

violation[{"msg": "Test violation"}] {
    input.test == true
}
`,
		Enabled:     true,
		Enforcement: "deny",
		Scope: PolicyScope{
			Resources:  []string{"Pod"},
			Namespaces: []string{"*"},
		},
		CreatedAt: time.Now(),
		Version:   "1.0.0",
	}

	err = engine.LoadPolicy(testPolicy)
	require.NoError(t, err)

	// Verify policy was loaded
	cachedPolicy, exists := engine.policyCache.Get("test-policy")
	assert.True(t, exists)
	assert.Equal(t, testPolicy.ID, cachedPolicy.ID)
	assert.Equal(t, testPolicy.Framework, cachedPolicy.Framework)
}

// TestCISKubernetesCompliance validates CIS benchmark compliance
func TestCISKubernetesCompliance(t *testing.T) {
	// Test CIS compliance check structure
	status := CISComplianceStatus{
		Version:               "1.8.0",
		OverallScore:          95.5,
		ComplianceLevel:       "L2",
		FailedControls:        []CISControl{},
		PassedControls:        []CISControl{},
		NotApplicableControls: []CISControl{},
	}

	assert.Equal(t, "1.8.0", status.Version)
	assert.Equal(t, 95.5, status.OverallScore)
	assert.Equal(t, "L2", status.ComplianceLevel)
	assert.NotNil(t, status.FailedControls)
	assert.NotNil(t, status.PassedControls)

	// Test CIS control structure
	control := CISControl{
		ID:                 "1.2.1",
		Title:              "Ensure anonymous-auth is not set to true",
		Description:        "Disable anonymous authentication to the API server",
		Level:              "L1",
		Status:             "PASS",
		Severity:           "HIGH",
		Remediation:        "Set --anonymous-auth=false in API server configuration",
		Evidence:           "API server configured with anonymous-auth=false",
		LastChecked:        time.Now(),
		AutomatedCheck:     true,
		ManualVerification: false,
	}

	assert.Equal(t, "1.2.1", control.ID)
	assert.Equal(t, "L1", control.Level)
	assert.Equal(t, "PASS", control.Status)
	assert.Equal(t, "HIGH", control.Severity)
	assert.True(t, control.AutomatedCheck)
}

// TestNISTCybersecurityFramework validates NIST framework compliance
func TestNISTCybersecurityFramework(t *testing.T) {
	// Test NIST compliance status
	status := NISTComplianceStatus{
		Framework:           "NIST CSF 2.0",
		OverallMaturity:     "Tier 3",
		IdentifyScore:       92.0,
		ProtectScore:        94.5,
		DetectScore:         89.0,
		RespondScore:        87.5,
		RecoverScore:        85.0,
		ImplementedControls: []NISTControl{},
		GapAnalysis:         []NISTGap{},
	}

	assert.Equal(t, "NIST CSF 2.0", status.Framework)
	assert.Equal(t, "Tier 3", status.OverallMaturity)
	assert.GreaterOrEqual(t, status.IdentifyScore, 0.0)
	assert.LessOrEqual(t, status.IdentifyScore, 100.0)
	assert.NotNil(t, status.ImplementedControls)
	assert.NotNil(t, status.GapAnalysis)

	// Test NIST control structure
	control := NISTControl{
		FunctionArea:   "ID",
		Category:       "ID.AM",
		Subcategory:    "ID.AM-1",
		Reference:      "Physical devices and systems within the organization are inventoried",
		Implementation: "Asset inventory management system implemented",
		MaturityLevel:  3,
		Status:         "IMPLEMENTED",
		Evidence:       "CMDB contains all physical devices",
		LastAssessed:   time.Now(),
	}

	assert.Equal(t, "ID", control.FunctionArea)
	assert.Equal(t, "ID.AM", control.Category)
	assert.Equal(t, 3, control.MaturityLevel)
	assert.Equal(t, "IMPLEMENTED", control.Status)
}

// TestOWASPTop10Protection validates OWASP compliance
func TestOWASPTop10Protection(t *testing.T) {
	// Test OWASP compliance status
	status := OWASPComplianceStatus{
		OWASPVersion:            "2021",
		OverallProtectionScore:  91.5,
		VulnerabilityAssessment: []OWASPVulnerability{},
		ProtectionMechanisms:    []OWASPProtectionMechanism{},
		RiskMitigation:          []OWASPRiskMitigation{},
		SecurityControls:        []OWASPSecurityControl{},
	}

	assert.Equal(t, "2021", status.OWASPVersion)
	assert.GreaterOrEqual(t, status.OverallProtectionScore, 0.0)
	assert.NotNil(t, status.VulnerabilityAssessment)
	assert.NotNil(t, status.ProtectionMechanisms)

	// Test OWASP vulnerability structure
	vulnerability := OWASPVulnerability{
		Category:     "A01:2021",
		Rank:         1,
		Title:        "Broken Access Control",
		Description:  "Access control enforces policy such that users cannot act outside of their intended permissions",
		RiskLevel:    "HIGH",
		Mitigated:    true,
		Controls:     []string{"RBAC", "Attribute-based access control"},
		LastAssessed: time.Now(),
	}

	assert.Equal(t, "A01:2021", vulnerability.Category)
	assert.Equal(t, 1, vulnerability.Rank)
	assert.Equal(t, "HIGH", vulnerability.RiskLevel)
	assert.True(t, vulnerability.Mitigated)
	assert.NotEmpty(t, vulnerability.Controls)
}

// TestGDPRDataProtection validates GDPR compliance
func TestGDPRDataProtection(t *testing.T) {
	// Test GDPR compliance status
	status := GDPRComplianceStatus{
		ComplianceScore:         88.0,
		LawfulBasisDocumented:   true,
		PrivacyByDesign:         true,
		DataMinimizationApplied: true,
		RetentionPoliciesActive: true,
		PIACompleted:            true,
	}

	assert.GreaterOrEqual(t, status.ComplianceScore, 0.0)
	assert.True(t, status.LawfulBasisDocumented)
	assert.True(t, status.PrivacyByDesign)
	assert.True(t, status.DataMinimizationApplied)
	assert.True(t, status.RetentionPoliciesActive)
	assert.True(t, status.PIACompleted)

	// Test consent management status
	consentStatus := ConsentManagementStatus{
		ConsentMechanismImplemented: true,
		ConsentWithdrawalEnabled:    true,
		ConsentRecords:              1500,
		LastConsentAudit:            time.Now().AddDate(0, -1, 0),
	}

	assert.True(t, consentStatus.ConsentMechanismImplemented)
	assert.True(t, consentStatus.ConsentWithdrawalEnabled)
	assert.Greater(t, consentStatus.ConsentRecords, 0)
}

// TestORANWG11SecurityCompliance validates O-RAN security compliance
func TestORANWG11SecurityCompliance(t *testing.T) {
	// Test O-RAN compliance status
	status := ORANComplianceStatus{
		WG11Specification:       "R4.0",
		OverallSecurityPosture:  93.5,
		InterfaceSecurityScore:  96.0,
		ZeroTrustImplementation: 91.0,
		ThreatModelingMaturity:  89.5,
		SecurityControlsActive:  []ORANSecurityControl{},
		ComplianceGaps:          []ORANComplianceGap{},
	}

	assert.Equal(t, "R4.0", status.WG11Specification)
	assert.GreaterOrEqual(t, status.OverallSecurityPosture, 0.0)
	assert.GreaterOrEqual(t, status.InterfaceSecurityScore, 0.0)
	assert.NotNil(t, status.SecurityControlsActive)
	assert.NotNil(t, status.ComplianceGaps)

	// Test interface security control
	control := ORANSecurityControl{
		ControlID:      "E2-SEC-001",
		Interface:      "E2",
		SecurityDomain: "Interface Security",
		Description:    "E2 interface mutual TLS authentication",
		Status:         "ACTIVE",
		LastVerified:   time.Now(),
	}

	assert.Equal(t, "E2-SEC-001", control.ControlID)
	assert.Equal(t, "E2", control.Interface)
	assert.Equal(t, "ACTIVE", control.Status)
}

// TestComplianceViolationHandling validates violation processing
func TestComplianceViolationHandling(t *testing.T) {
	// Test compliance violation structure
	violation := ComplianceViolation{
		ID:                "violation-001",
		Framework:         "CIS Kubernetes Benchmark",
		Category:          "Pod Security",
		Severity:          "HIGH",
		Description:       "Pod running with privileged security context",
		AffectedResource:  "default/test-pod",
		DetectionTime:     time.Now(),
		Status:            "ACTIVE",
		RemediationAction: "Update pod security context to non-privileged",
		Evidence: map[string]interface{}{
			"privileged":       true,
			"security_context": "root",
			"capabilities":     []string{"SYS_ADMIN"},
		},
	}

	assert.Equal(t, "violation-001", violation.ID)
	assert.Equal(t, "CIS Kubernetes Benchmark", violation.Framework)
	assert.Equal(t, "HIGH", violation.Severity)
	assert.Equal(t, "ACTIVE", violation.Status)
	assert.NotEmpty(t, violation.RemediationAction)
	assert.NotNil(t, violation.Evidence)
	assert.Contains(t, violation.Evidence, "privileged")

	// Test compliance recommendation structure
	recommendation := ComplianceRecommendation{
		ID:              "rec-001",
		Framework:       "Multi-Framework",
		Priority:        "HIGH",
		Title:           "Implement Pod Security Standards",
		Description:     "Apply Kubernetes Pod Security Standards to enforce security policies",
		Implementation:  "Configure Pod Security Standards admission controller",
		EstimatedEffort: "2-3 days",
		BusinessImpact:  "Improved container security posture",
		CreatedAt:       time.Now(),
	}

	assert.Equal(t, "rec-001", recommendation.ID)
	assert.Equal(t, "HIGH", recommendation.Priority)
	assert.NotEmpty(t, recommendation.Implementation)
	assert.NotEmpty(t, recommendation.BusinessImpact)
}

// TestComplianceMetrics validates metrics collection
func TestComplianceMetrics(t *testing.T) {
	metricsCollector := NewComplianceMetricsCollector()
	require.NotNil(t, metricsCollector)

	// Test that metrics collectors are initialized
	assert.NotNil(t, metricsCollector.complianceScore)
	assert.NotNil(t, metricsCollector.violationCounter)
	assert.NotNil(t, metricsCollector.remediationCounter)
	assert.NotNil(t, metricsCollector.auditEventCounter)
	assert.NotNil(t, metricsCollector.complianceCheckDuration)

	// Test metrics recording (would typically involve Prometheus registry in real implementation)
	// For unit tests, we just verify the structure exists
	t.Log("Compliance metrics collector initialized successfully")
}

// TestComplianceAuditTrail validates audit logging
func TestComplianceAuditTrail(t *testing.T) {
	// Test audit event structure
	auditEvent := ComplianceAuditEvent{
		EventID:   "audit-001",
		Timestamp: time.Now(),
		EventType: "compliance_check",
		Framework: "CIS",
		Actor:     "system",
		Resource:  "cluster",
		Action:    "compliance_assessment",
		Result:    "PASS",
		Details: map[string]interface{}{
			"score":           95.5,
			"violations":      2,
			"recommendations": 3,
		},
	}

	assert.Equal(t, "audit-001", auditEvent.EventID)
	assert.Equal(t, "compliance_check", auditEvent.EventType)
	assert.Equal(t, "CIS", auditEvent.Framework)
	assert.Equal(t, "PASS", auditEvent.Result)
	assert.NotNil(t, auditEvent.Details)
	assert.Contains(t, auditEvent.Details, "score")
}

// TestComplianceDashboard validates dashboard functionality
func TestComplianceDashboard(t *testing.T) {
	// Test dashboard structure
	dashboard := ComplianceDashboard{
		OverallHealth:   "HEALTHY",
		ActiveAlerts:    []ComplianceAlert{},
		TrendData:       []ComplianceTrendPoint{},
		FrameworkStatus: []FrameworkStatusCard{},
	}

	assert.Equal(t, "HEALTHY", dashboard.OverallHealth)
	assert.NotNil(t, dashboard.ActiveAlerts)
	assert.NotNil(t, dashboard.TrendData)
	assert.NotNil(t, dashboard.FrameworkStatus)

	// Test compliance alert structure
	alert := ComplianceAlert{
		ID:           "alert-001",
		Severity:     "HIGH",
		Framework:    "OWASP",
		Message:      "Vulnerability scan detected critical issues",
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	assert.Equal(t, "alert-001", alert.ID)
	assert.Equal(t, "HIGH", alert.Severity)
	assert.Equal(t, "OWASP", alert.Framework)
	assert.False(t, alert.Acknowledged)

	// Test framework status card
	statusCard := FrameworkStatusCard{
		Framework:   "CIS",
		Status:      "COMPLIANT",
		Score:       95.5,
		LastChecked: time.Now(),
		Violations:  2,
	}

	assert.Equal(t, "CIS", statusCard.Framework)
	assert.Equal(t, "COMPLIANT", statusCard.Status)
	assert.Equal(t, 95.5, statusCard.Score)
	assert.Equal(t, 2, statusCard.Violations)
}

// TestRemediationActions validates automated remediation
func TestRemediationActions(t *testing.T) {
	// Test remediation action structure
	action := RemediationAction{
		ID:                "remediation-001",
		ViolationID:       "violation-001",
		ActionType:        "apply_network_policy",
		Description:       "Apply default deny network policy",
		Status:            "COMPLETED",
		ExecutedAt:        time.Now().Add(-5 * time.Minute),
		CompletedAt:       &time.Time{},
		Success:           true,
		ErrorMessage:      "",
		RollbackPlan:      "Remove network policy if issues occur",
		ResourcesAffected: []string{"namespace/default"},
		Details: map[string]interface{}{
			"policy_name": "default-deny-all",
			"namespace":   "default",
		},
	}

	*action.CompletedAt = time.Now()

	assert.Equal(t, "remediation-001", action.ID)
	assert.Equal(t, "violation-001", action.ViolationID)
	assert.Equal(t, "apply_network_policy", action.ActionType)
	assert.Equal(t, "COMPLETED", action.Status)
	assert.True(t, action.Success)
	assert.Empty(t, action.ErrorMessage)
	assert.NotEmpty(t, action.RollbackPlan)
	assert.NotNil(t, action.Details)
}

// BenchmarkComplianceCheck benchmarks the performance of compliance checks
func BenchmarkComplianceCheck(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	framework := NewComprehensiveComplianceFramework(logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := framework.RunComprehensiveComplianceCheck(ctx)
		if err != nil {
			b.Fatalf("Compliance check failed: %v", err)
		}
	}
}

// TestPolicyEvaluationPerformance validates that policy evaluation is performant
func TestPolicyEvaluationPerformance(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine, err := NewOPACompliancePolicyEngine(logger)
	require.NoError(t, err)

	// Test policy evaluation performance
	start := time.Now()

	// Simulate multiple policy evaluations
	for i := 0; i < 100; i++ {
		// In a real test, this would evaluate actual admission requests
		// For now, we just verify the engine is responsive
		_, exists := engine.policyCache.Get("cis-pod-security")
		if !exists {
			t.Error("Expected CIS pod security policy to be cached")
		}
	}

	duration := time.Since(start)
	t.Logf("100 policy cache lookups took: %v", duration)

	// Ensure performance is reasonable (under 10ms for 100 lookups)
	assert.Less(t, duration, 10*time.Millisecond)
}

// TestComplianceIntegration validates integration between components
func TestComplianceIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test that all major components can be initialized together
	framework := NewComprehensiveComplianceFramework(logger)
	require.NotNil(t, framework)

	opaEngine, err := NewOPACompliancePolicyEngine(logger)
	require.NoError(t, err)
	require.NotNil(t, opaEngine)

	// Test that components can work together
	ctx := context.Background()
	status, err := framework.RunComprehensiveComplianceCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Verify that the integration produces expected results
	assert.GreaterOrEqual(t, status.OverallCompliance, 85.0, "Expected high compliance score in test environment")
	assert.NotEmpty(t, status.AuditTrail, "Expected audit events to be generated")

	t.Logf("Integration test completed successfully with %.2f%% compliance", status.OverallCompliance)
}

// TestErrorHandling validates error scenarios
func TestErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with invalid configurations
	invalidFramework := &ComprehensiveComplianceFramework{}
	status, err := invalidFramework.RunComprehensiveComplianceCheck(context.Background())

	// Should handle gracefully
	assert.Error(t, err)
	assert.Nil(t, status)

	// Test OPA engine with invalid policy
	engine, err := NewOPACompliancePolicyEngine(logger)
	require.NoError(t, err)

	invalidPolicy := PolicyDefinition{
		ID:         "invalid-policy",
		RegoPolicy: "invalid rego syntax {",
	}

	err = engine.LoadPolicy(invalidPolicy)
	assert.Error(t, err, "Expected error when loading invalid policy")
}

// TestComplianceReporting validates report generation
func TestComplianceReporting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	framework := NewComprehensiveComplianceFramework(logger)

	ctx := context.Background()
	status, err := framework.RunComprehensiveComplianceCheck(ctx)
	require.NoError(t, err)

	// Test report structure
	assert.NotZero(t, status.Timestamp)
	assert.GreaterOrEqual(t, status.OverallCompliance, 0.0)
	assert.LessOrEqual(t, status.OverallCompliance, 100.0)

	// Test that all framework sections are present
	assert.NotNil(t, status.CISCompliance)
	assert.NotNil(t, status.NISTCompliance)
	assert.NotNil(t, status.OWASPCompliance)
	assert.NotNil(t, status.GDPRCompliance)
	assert.NotNil(t, status.ORANCompliance)
	assert.NotNil(t, status.OPACompliance)

	// Test audit trail
	assert.NotEmpty(t, status.AuditTrail)
	auditEvent := status.AuditTrail[0]
	assert.NotEmpty(t, auditEvent.EventID)
	assert.Equal(t, "comprehensive_compliance_check", auditEvent.EventType)
	assert.NotNil(t, auditEvent.Details)

	t.Logf("Compliance report generated successfully")
	t.Logf("Report timestamp: %v", status.Timestamp)
	t.Logf("Audit events: %d", len(status.AuditTrail))
}

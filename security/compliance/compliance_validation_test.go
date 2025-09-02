package compliance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"log/slog"
	"os"
)

// TestComprehensiveComplianceFramework validates the main compliance framework
func TestComprehensiveComplianceFramework(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with nil config to verify error handling
	framework, err := NewComprehensiveComplianceFramework(logger, nil)
	require.Error(t, err)
	require.Nil(t, framework)
	assert.Contains(t, err.Error(), "kubernetes config cannot be nil")

	// Create a config that will pass basic validation but fail to connect
	// This tests that we handle Kubernetes client creation properly
	config := &rest.Config{
		Host: "https://localhost:8443",
	}

	// The framework may or may not succeed depending on the test environment
	// In a test environment without a running Kubernetes cluster, it may still create the client
	framework, err = NewComprehensiveComplianceFramework(logger, config)

	// Either way, if we get a framework, it should be properly initialized
	if framework != nil {
		assert.NotNil(t, framework.logger)
		assert.NotNil(t, framework.kubeClient)
		assert.NotNil(t, framework.validator)
		assert.NotNil(t, framework.metricsCollector)
		assert.NotNil(t, framework.config)
	}
}

// TestAutomatedComplianceMonitor validates the automated monitoring system
func TestAutomatedComplianceMonitor(t *testing.T) {
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

// TestOPACompliancePolicyEngine validates the OPA policy engine type
func TestOPACompliancePolicyEngine(t *testing.T) {
	// Test OPA engine structure
	engine := &OPACompliancePolicyEngine{
		enabled:                 true,
		policyBundleURL:         "https://example.com/policies",
		policyDecisionTimeout:   5 * time.Second,
		evaluationCacheEnabled:  true,
		policyValidationMode:    "strict",
		realTimeEvaluation:      true,
		continuousMonitoring:    true,
		automaticRemediation:    false,
		policyViolationHandling: "block",
		auditLoggingEnabled:     true,
		decisionLogsRetention:   30 * 24 * time.Hour,
		auditStorageBackend:     "local",
	}

	assert.True(t, engine.enabled)
	assert.Equal(t, "https://example.com/policies", engine.policyBundleURL)
	assert.Equal(t, 5*time.Second, engine.policyDecisionTimeout)
	assert.True(t, engine.evaluationCacheEnabled)
	assert.Equal(t, "strict", engine.policyValidationMode)
	assert.True(t, engine.realTimeEvaluation)
	assert.Equal(t, "block", engine.policyViolationHandling)
}

// TestComplianceStatus validates compliance status structure
func TestComplianceStatus(t *testing.T) {
	// Test compliance status structure
	status := &ComplianceStatus{
		Timestamp:         time.Now(),
		OverallScore:      85.5,
		OverallCompliance: 85.5, // Both fields for compatibility
		FrameworkScores: map[string]float64{
			"CIS":   90.0,
			"NIST":  85.0,
			"OWASP": 80.0,
		},
		ComplianceViolations: []ComplianceViolation{},
		RecommendedActions:   []ComplianceRecommendation{},
		AuditTrail:           []ComplianceAuditEvent{},
		NextAuditDate:        time.Now().Add(24 * time.Hour),
	}

	assert.NotZero(t, status.Timestamp)
	assert.Equal(t, 85.5, status.OverallScore)
	assert.Equal(t, 85.5, status.OverallCompliance)
	assert.Equal(t, 3, len(status.FrameworkScores))
	assert.NotNil(t, status.ComplianceViolations)
	assert.NotNil(t, status.RecommendedActions)
	assert.NotNil(t, status.AuditTrail)
}

// TestComplianceViolation validates compliance violation structure
func TestComplianceViolation(t *testing.T) {
	// Test compliance violation structure
	violation := ComplianceViolation{
		ID:               "violation-001",
		Framework:        "CIS Kubernetes Benchmark",
		Category:         "Security", // Added category field
		RuleID:           "CIS-001",
		RuleName:         "Pod Security Context",
		Description:      "Pod running with privileged security context",
		Severity:         "HIGH",
		Resource:         "default/test-pod",
		AffectedResource: "default/test-pod", // Added affected resource field
		Namespace:        "default",
		DetectedAt:       time.Now(),
		Status:           "open",
		Impact:           "Medium impact on security posture",
	}

	assert.Equal(t, "violation-001", violation.ID)
	assert.Equal(t, "CIS Kubernetes Benchmark", violation.Framework)
	assert.Equal(t, "Security", violation.Category)
	assert.Equal(t, "HIGH", violation.Severity)
	assert.Equal(t, "default/test-pod", violation.AffectedResource)
	assert.Equal(t, "open", violation.Status)
}

// TestComplianceAlert validates compliance alert structure
func TestComplianceAlert(t *testing.T) {
	// Test compliance alert structure
	alert := ComplianceAlert{
		ID:           "alert-001",
		Timestamp:    time.Now(),
		Severity:     "HIGH",
		Framework:    "OWASP", // Added framework field
		Message:      "Vulnerability scan detected critical issues",
		Acknowledged: false, // Added acknowledged field
		Actions:      []string{"review", "remediate"},
		Metadata: map[string]interface{}{
			"scan_id": "scan-123",
			"issues":  5,
		},
	}

	assert.Equal(t, "alert-001", alert.ID)
	assert.Equal(t, "HIGH", alert.Severity)
	assert.Equal(t, "OWASP", alert.Framework)
	assert.False(t, alert.Acknowledged)
	assert.NotNil(t, alert.Actions)
	assert.NotNil(t, alert.Metadata)
}

// TestComplianceMetricsCollector validates metrics collection
func TestComplianceMetricsCollector(t *testing.T) {
	metricsCollector := NewComplianceMetricsCollector()
	require.NotNil(t, metricsCollector)

	// Test that metrics collectors are initialized
	assert.NotNil(t, metricsCollector.complianceScore)
	assert.NotNil(t, metricsCollector.violationCount)
	assert.NotNil(t, metricsCollector.violationCounter) // Added field
	assert.NotNil(t, metricsCollector.remediationCounter)
	assert.NotNil(t, metricsCollector.complianceCheckDuration)

	// Test MockMetric methods
	metric := metricsCollector.complianceScore
	labeledMetric := metric.WithLabelValues("test")
	assert.NotNil(t, labeledMetric)

	// Test metric operations (should not panic)
	metric.Set(85.5)
	metric.Inc()
	metric.Add(10.0)
	metric.Observe(2.5)

	t.Log("Compliance metrics collector initialized successfully")
}

// TestOPAPolicy validates OPA policy structure
func TestOPAPolicy(t *testing.T) {
	// Test OPA policy structure
	policy := OPAPolicy{
		PolicyID:        "test-policy-001",
		Name:            "Test Security Policy",
		Description:     "Test policy for security validation",
		Package:         "test.security",
		Rego:            "package test.security\n\nviolation[{\"msg\": \"test violation\"}] {\n    input.test == true\n}",
		Version:         "1.0.0",
		Framework:       "TEST",
		Category:        "Security",
		Severity:        "HIGH",
		Enabled:         true,
		CreatedAt:       &time.Time{},
		UpdatedAt:       &time.Time{},
		EvaluationCount: 100,
		ViolationCount:  5,
		Metadata: map[string]interface{}{
			"author": "test-suite",
			"tags":   []string{"security", "test"},
		},
		Dependencies: []string{"base-policy"},
	}

	*policy.CreatedAt = time.Now().Add(-24 * time.Hour)
	*policy.UpdatedAt = time.Now()

	assert.Equal(t, "test-policy-001", policy.PolicyID)
	assert.Equal(t, "Test Security Policy", policy.Name)
	assert.Equal(t, "1.0.0", policy.Version)
	assert.Equal(t, "TEST", policy.Framework)
	assert.Equal(t, "Security", policy.Category)
	assert.Equal(t, "HIGH", policy.Severity)
	assert.True(t, policy.Enabled)
	assert.Equal(t, int64(100), policy.EvaluationCount)
	assert.Equal(t, int64(5), policy.ViolationCount)
	assert.NotNil(t, policy.Metadata)
	assert.NotEmpty(t, policy.Dependencies)
}

// TestRemediationAction validates remediation action structure
func TestRemediationAction(t *testing.T) {
	// Test remediation action structure
	completedTime := time.Now()
	action := RemediationAction{
		ID:                "remediation-001",
		ViolationID:       "violation-001",
		ActionType:        "apply_network_policy",
		Description:       "Apply default deny network policy",
		Status:            "COMPLETED",
		ExecutedAt:        time.Now().Add(-5 * time.Minute),
		CompletedAt:       &completedTime,
		Success:           true,
		ErrorMessage:      "",
		RollbackPlan:      "Remove network policy if issues occur",
		ResourcesAffected: []string{"namespace/default"},
		Details: map[string]interface{}{
			"policy_name": "default-deny-all",
			"namespace":   "default",
		},
	}

	assert.Equal(t, "remediation-001", action.ID)
	assert.Equal(t, "violation-001", action.ViolationID)
	assert.Equal(t, "apply_network_policy", action.ActionType)
	assert.Equal(t, "COMPLETED", action.Status)
	assert.True(t, action.Success)
	assert.Empty(t, action.ErrorMessage)
	assert.NotEmpty(t, action.RollbackPlan)
	assert.NotNil(t, action.Details)
}

// TestPolicyTestCase validates policy test case structure
func TestPolicyTestCase(t *testing.T) {
	// Test policy test case structure
	testCase := PolicyTestCase{
		testID:      "test-001",
		name:        "Test Pod Security",
		description: "Test that pods without security context are flagged",
		input: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]string{"name": "test-pod"},
			"spec": map[string]interface{}{
				"securityContext": nil,
			},
		},
		expectedOutput:    "violation",
		expectedViolation: true,
	}

	assert.Equal(t, "test-001", testCase.testID)
	assert.Equal(t, "Test Pod Security", testCase.name)
	assert.NotEmpty(t, testCase.description)
	assert.NotNil(t, testCase.input)
	assert.Equal(t, "violation", testCase.expectedOutput)
	assert.True(t, testCase.expectedViolation)
}

// BenchmarkComplianceCheck benchmarks the performance of compliance status creation
func BenchmarkComplianceStatusCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := &ComplianceStatus{
			Timestamp:         time.Now(),
			OverallScore:      85.5,
			OverallCompliance: 85.5,
			FrameworkScores: map[string]float64{
				"CIS":   90.0,
				"NIST":  85.0,
				"OWASP": 80.0,
			},
			ComplianceViolations: []ComplianceViolation{},
			RecommendedActions:   []ComplianceRecommendation{},
			AuditTrail:           []ComplianceAuditEvent{},
			NextAuditDate:        time.Now().Add(24 * time.Hour),
		}
		if status == nil {
			b.Fatal("Failed to create compliance status")
		}
	}
}

// TestErrorHandling validates error scenarios
func TestErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with nil config - this should definitely error
	framework, err := NewComprehensiveComplianceFramework(logger, nil)
	assert.Error(t, err)
	assert.Nil(t, framework)
	assert.Contains(t, err.Error(), "kubernetes config cannot be nil")
}

// TestMockMetric validates mock metric implementation
func TestMockMetric(t *testing.T) {
	metric := &MockMetric{
		name:   "test_metric",
		labels: map[string]string{},
	}

	// Test that methods don't panic
	labeledMetric := metric.WithLabelValues("test", "value")
	assert.NotNil(t, labeledMetric)

	// Test all operations
	metric.Set(100.0)
	metric.Inc()
	metric.Add(50.0)
	metric.Observe(2.5)

	// Should not panic or error
	assert.Equal(t, "test_metric", metric.name)
}

// TestRunComprehensiveComplianceCheck validates the comprehensive compliance check method
func TestRunComprehensiveComplianceCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a mock config for testing
	config := &rest.Config{
		Host: "https://localhost:8443",
	}

	framework, err := NewComprehensiveComplianceFramework(logger, config)
	// Skip this test if we can't create a framework (no Kubernetes cluster available)
	if err != nil {
		t.Skip("Skipping test - no Kubernetes cluster available")
		return
	}

	require.NotNil(t, framework)

	// Test compliance check
	status, err := framework.RunComprehensiveComplianceCheck(nil)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Validate the status structure
	assert.NotZero(t, status.Timestamp)
	assert.Equal(t, 85.5, status.OverallScore)
	assert.Equal(t, 85.5, status.OverallCompliance)
	assert.NotNil(t, status.FrameworkScores)
	assert.NotNil(t, status.ComplianceViolations)
	assert.NotNil(t, status.RecommendedActions)
	assert.NotNil(t, status.AuditTrail)

	t.Logf("Compliance check completed with score: %.2f%%", status.OverallCompliance)
	t.Logf("Framework scores: %+v", status.FrameworkScores)
}

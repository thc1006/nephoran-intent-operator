package automation

import (
	"testing"
)

func TestAutomationTypeConstants(t *testing.T) {
	// Test that constants are defined and have expected values
	tests := []struct {
		name     string
		constant AutomationType
		expected string
	}{
		{"Provisioning", AutomationTypeProvisioning, "provisioning"},
		{"Renewal", AutomationTypeRenewal, "renewal"},
		{"Revocation", AutomationTypeRevocation, "revocation"},
		{"Discovery", AutomationTypeDiscovery, "discovery"},
		{"HealthCheck", AutomationTypeHealthCheck, "health_check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Expected %s = %q, got %q", tt.name, tt.expected, string(tt.constant))
			}
		})
	}
}

func TestAutomationStatusConstants(t *testing.T) {
	// Test that status constants are defined
	tests := []struct {
		name     string
		constant AutomationStatus
		expected string
	}{
		{"Pending", AutomationStatusPending, "pending"},
		{"Processing", AutomationStatusProcessing, "processing"},
		{"Completed", AutomationStatusCompleted, "completed"},
		{"Failed", AutomationStatusFailed, "failed"},
		{"Canceled", AutomationStatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Expected %s = %q, got %q", tt.name, tt.expected, string(tt.constant))
			}
		})
	}
}

func TestAutomationPriorityConstants(t *testing.T) {
	// Test that priority constants are defined and have expected order
	if AutomationPriorityLow >= AutomationPriorityNormal {
		t.Error("Expected Low priority to be less than Normal priority")
	}
	if AutomationPriorityNormal >= AutomationPriorityHigh {
		t.Error("Expected Normal priority to be less than High priority")
	}
	if AutomationPriorityHigh >= AutomationPriorityCritical {
		t.Error("Expected High priority to be less than Critical priority")
	}
}
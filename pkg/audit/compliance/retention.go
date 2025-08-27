package compliance

import (
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// RetentionManager manages audit log retention policies
type RetentionManager struct {
	config *RetentionConfig
}

// RetentionConfig holds retention configuration
type RetentionConfig struct {
	ComplianceMode     []types.ComplianceStandard
	DefaultRetention   time.Duration
	MinRetention       time.Duration
	MaxRetention       time.Duration
	PurgeInterval      time.Duration
	BackupBeforePurge  bool
	CompressionEnabled bool
}

// RetentionPolicy defines retention for specific event types
type RetentionPolicy struct {
	EventType       types.EventType
	RetentionPeriod time.Duration
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager(config *RetentionConfig) *RetentionManager {
	return &RetentionManager{config: config}
}

// CalculateRetentionPeriod calculates retention period based on compliance standards
func (rm *RetentionManager) CalculateRetentionPeriod(event *types.AuditEvent, standards []types.ComplianceStandard) time.Duration {
	maxRetention := rm.config.DefaultRetention

	for _, standard := range standards {
		var retention time.Duration
		switch standard {
		case types.ComplianceSOC2:
			retention = 7 * 365 * 24 * time.Hour
		case types.ComplianceISO27001:
			retention = 3 * 365 * 24 * time.Hour
		case types.CompliancePCIDSS:
			retention = 365 * 24 * time.Hour
		default:
			retention = rm.config.DefaultRetention
		}

		if retention > maxRetention {
			maxRetention = retention
		}
	}

	// Security events may need longer retention
	if event.EventType == types.EventTypeSecurityViolation {
		securityRetention := 7 * 365 * 24 * time.Hour
		if securityRetention > maxRetention {
			maxRetention = securityRetention
		}
	}

	return maxRetention
}

// ApplyRetentionPolicy applies retention policy to events
func (rm *RetentionManager) ApplyRetentionPolicy(events []*types.AuditEvent, standards []types.ComplianceStandard) []*types.AuditEvent {
	now := time.Now()
	var retained []*types.AuditEvent

	for _, event := range events {
		retentionPeriod := rm.CalculateRetentionPeriod(event, standards)
		if now.Sub(event.Timestamp) <= retentionPeriod {
			retained = append(retained, event)
		}
	}

	return retained
}

// GetPolicyRecommendations provides retention policy recommendations
func (rm *RetentionManager) GetPolicyRecommendations(standards []types.ComplianceStandard) []RetentionPolicy {
	var policies []RetentionPolicy

	// Base recommendations
	policies = append(policies, RetentionPolicy{
		EventType:       types.EventTypeDataAccess,
		RetentionPeriod: 90 * 24 * time.Hour,
	})

	policies = append(policies, RetentionPolicy{
		EventType:       types.EventTypeSystemChange,
		RetentionPeriod: 365 * 24 * time.Hour,
	})

	policies = append(policies, RetentionPolicy{
		EventType:       types.EventTypeSecurityViolation,
		RetentionPeriod: 7 * 365 * 24 * time.Hour,
	})

	// Adjust based on compliance standards
	for _, standard := range standards {
		switch standard {
		case types.ComplianceSOC2:
			// SOC2 requires longer retention
			for i := range policies {
				if policies[i].RetentionPeriod < 7*365*24*time.Hour {
					policies[i].RetentionPeriod = 7 * 365 * 24 * time.Hour
				}
			}
		case types.CompliancePCIDSS:
			// PCI DSS minimum 1 year
			for i := range policies {
				if policies[i].RetentionPeriod < 365*24*time.Hour {
					policies[i].RetentionPeriod = 365 * 24 * time.Hour
				}
			}
		}
	}

	return policies
}

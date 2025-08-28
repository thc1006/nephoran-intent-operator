package audit

import (
	"time"

	audittypes "github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// AuditConfig is an alias for controller compatibility
type AuditConfig = AuditSystemConfig

// Re-export for backward compatibility
type Event = audittypes.AuditEvent

// FilterCriteria defines criteria for filtering audit events
type FilterCriteria struct {
	StartTime     *time.Time                  `json:"start_time,omitempty"`
	EndTime       *time.Time                  `json:"end_time,omitempty"`
	UserID        string                      `json:"user_id,omitempty"`
	Component     string                      `json:"component,omitempty"`
	EventTypes    []audittypes.EventType      `json:"event_types,omitempty"`
	Severities    []audittypes.Severity       `json:"severities,omitempty"`
	MinSeverity   audittypes.Severity         `json:"min_severity,omitempty"`
	MaxResults    int                         `json:"max_results,omitempty"`
	SourceIP      string                      `json:"source_ip,omitempty"`
	ComplianceReq []audittypes.ComplianceStandard `json:"compliance_req,omitempty"`
	Tags          []string                    `json:"tags,omitempty"`
}

// AlertThresholds defines thresholds for different alert types
type AlertThresholds struct {
	CriticalEvents int `json:"critical_events" yaml:"critical_events"`
	ErrorRate      int `json:"error_rate" yaml:"error_rate"`
	QueueBacklog   int `json:"queue_backlog" yaml:"queue_backlog"`
	ResponseTime   int `json:"response_time" yaml:"response_time"` // milliseconds
}

// DefaultConfig returns a default audit system configuration
func DefaultConfig() *AuditSystemConfig {
	return &AuditSystemConfig{
		Enabled:           true,
		EnabledSources:    []string{"kubernetes", "oran", "nephio"},
		RetentionDays:     365,
		LogLevel:          audittypes.SeverityInfo,
		BatchSize:         DefaultBatchSize,
		FlushInterval:     DefaultFlushInterval,
		MaxQueueSize:      MaxAuditQueueSize,
		EnableIntegrity:   true,
		ComplianceMode:    []audittypes.ComplianceStandard{},
	}
}
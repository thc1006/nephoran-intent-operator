// Package types provides comprehensive audit interfaces and contracts

// for the Nephoran Intent Operator audit system.


package types



import (

	"context"

	"time"

)



// AuditLogger defines the interface for audit logging systems.

type AuditLogger interface {

	// LogEvent logs a single audit event.

	LogEvent(ctx context.Context, event *AuditEvent) error



	// LogEvents logs multiple audit events.

	LogEvents(ctx context.Context, events []*AuditEvent) error



	// Query searches for audit events.

	Query(ctx context.Context, query interface{}) ([]*AuditEvent, error)



	// Health checks the audit system health.

	Health(ctx context.Context) error



	// Close gracefully shuts down the audit logger.

	Close() error

}



// Backend represents an audit log destination.

type Backend interface {

	// Type returns the backend type identifier.

	Type() string



	// Initialize sets up the backend with configuration.

	Initialize(config interface{}) error



	// WriteEvent writes a single audit event.

	WriteEvent(ctx context.Context, event *AuditEvent) error



	// WriteEvents writes multiple audit events in a batch.

	WriteEvents(ctx context.Context, events []*AuditEvent) error



	// Query searches for audit events (optional for compliance reporting).

	Query(ctx context.Context, query interface{}) (interface{}, error)



	// Health checks the backend connectivity and status.

	Health(ctx context.Context) error



	// Close gracefully shuts down the backend.

	Close() error

}



// ComplianceTracker defines the interface for compliance tracking.

type ComplianceTracker interface {

	// ProcessEvent processes an audit event for compliance tracking.

	ProcessEvent(event *AuditEvent)



	// GetStatus returns current compliance status.

	GetStatus() interface{}



	// GenerateReport generates a compliance report.

	GenerateReport(ctx context.Context, startTime, endTime time.Time) (interface{}, error)



	// GetViolations returns compliance violations.

	GetViolations() []interface{}

}



// AuditEntry represents a generic audit entry interface.

type AuditEntry interface {

	// GetID returns the unique identifier.

	GetID() string



	// GetTimestamp returns the event timestamp.

	GetTimestamp() time.Time



	// GetEventType returns the event type.

	GetEventType() EventType



	// GetSeverity returns the severity level.

	GetSeverity() Severity



	// Validate checks if the entry is valid.

	Validate() error



	// ToJSON converts to JSON representation.

	ToJSON() ([]byte, error)

}



// Ensure AuditEvent implements AuditEntry.

var _ AuditEntry = (*AuditEvent)(nil)



// GetID returns the unique identifier.

func (ae *AuditEvent) GetID() string {

	return ae.ID

}



// GetTimestamp returns the event timestamp.

func (ae *AuditEvent) GetTimestamp() time.Time {

	return ae.Timestamp

}



// GetEventType returns the event type.

func (ae *AuditEvent) GetEventType() EventType {

	return ae.EventType

}



// GetSeverity returns the severity level.

func (ae *AuditEvent) GetSeverity() Severity {

	return ae.Severity

}


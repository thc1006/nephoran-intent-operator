package integration_tests

import (
	"time"
)

// AuditSystem mock for testing
type AuditSystem struct {
	events []AuditEvent
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID        string
	Timestamp time.Time
	Type      string
	Severity  string
	Result    string
	Message   string
	Component string
}

// Mock severity and result constants
const (
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
	ResultSuccess   = "success"
	ResultFailure   = "failure"
)

// NewMockAuditSystem creates a mock audit system for testing
func NewMockAuditSystem() *AuditSystem {
	return &AuditSystem{
		events: make([]AuditEvent, 0),
	}
}

// LogEvent logs an audit event
func (a *AuditSystem) LogEvent(event AuditEvent) {
	a.events = append(a.events, event)
}

// GetEvents returns all logged events
func (a *AuditSystem) GetEvents() []AuditEvent {
	return a.events
}
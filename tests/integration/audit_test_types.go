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
	ID                 string
	Timestamp          time.Time
	EventType          string
	Type               string
	Severity           string
	Result             string
	Message            string
	Component          string
	Action             string
	DataClassification string
	UserContext        *UserContext
	NetworkContext     *NetworkContext
	ResourceContext    *ResourceContext
	Data               map[string]interface{}
}

// Context types for audit events
type UserContext struct {
	UserID         string
	Username       string
	Role           string
	AuthMethod     string
	AuthProvider   string
	Groups         []string
	ServiceAccount bool
}

type NetworkContext struct {
	SourceIP   string
	SourcePort int
	UserAgent  string
	RequestID  string
}

type ResourceContext struct {
	ResourceType string
	ResourceID   string
	Operation    string
	Namespace    string
	APIVersion   string
}

// Mock severity and result constants
const (
	SeverityInfo     = "info"
	SeverityNotice   = "notice"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
	ResultSuccess    = "success"
	ResultFailure    = "failure"
	
	// Event types
	EventTypeAuthentication     = "authentication"
	EventTypeDataAccess         = "data_access"
	EventTypeSecurityViolation  = "security_violation"
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

// Stop terminates the audit system (mock implementation)
func (a *AuditSystem) Stop() error {
	return nil
}

// GetStats returns audit system statistics (mock implementation)
func (a *AuditSystem) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"events_received": len(a.events),
		"events_dropped":  0,
	}
}

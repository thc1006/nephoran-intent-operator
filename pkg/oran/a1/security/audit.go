package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	Enabled             bool               `json:"enabled"`
	Level               AuditLevel         `json:"level"`
	Destinations        []AuditDestination `json:"destinations"`
	BufferSize          int                `json:"buffer_size"`
	FlushInterval       time.Duration      `json:"flush_interval"`
	IncludeRequestBody  bool               `json:"include_request_body"`
	IncludeResponseBody bool               `json:"include_response_body"`
	MaskSensitiveData   bool               `json:"mask_sensitive_data"`
	SensitiveFields     []string           `json:"sensitive_fields"`
	RetentionDays       int                `json:"retention_days"`
	SignEvents          bool               `json:"sign_events"`
	EncryptEvents       bool               `json:"encrypt_events"`
	ComplianceMode      ComplianceStandard `json:"compliance_mode"`
	AlertOnViolation    bool               `json:"alert_on_violation"`
	AlertThresholds     *AlertThresholds   `json:"alert_thresholds"`
}

// AuditLevel represents the audit logging level
type AuditLevel string

const (
	AuditLevelNone     AuditLevel = "none"
	AuditLevelMinimal  AuditLevel = "minimal"
	AuditLevelStandard AuditLevel = "standard"
	AuditLevelDetailed AuditLevel = "detailed"
	AuditLevelVerbose  AuditLevel = "verbose"
)

// AuditDestination represents where audit logs are sent
type AuditDestination struct {
	Type           DestinationType   `json:"type"`
	Endpoint       string            `json:"endpoint"`
	Format         AuditFormat       `json:"format"`
	BatchSize      int               `json:"batch_size"`
	Timeout        time.Duration     `json:"timeout"`
	RetryAttempts  int               `json:"retry_attempts"`
	Authentication map[string]string `json:"authentication"`
}

// DestinationType represents the type of audit destination
type DestinationType string

const (
	DestinationFile       DestinationType = "file"
	DestinationSyslog     DestinationType = "syslog"
	DestinationHTTP       DestinationType = "http"
	DestinationKafka      DestinationType = "kafka"
	DestinationElastic    DestinationType = "elasticsearch"
	DestinationSplunk     DestinationType = "splunk"
	DestinationCloudWatch DestinationType = "cloudwatch"
)

// AuditFormat represents the format of audit logs
type AuditFormat string

const (
	FormatJSON   AuditFormat = "json"
	FormatCEF    AuditFormat = "cef"  // Common Event Format
	FormatLEEF   AuditFormat = "leef" // Log Event Extended Format
	FormatSyslog AuditFormat = "syslog"
)

// ComplianceStandard, SecurityLevel, ThreatLevel, and ViolationType are defined in constants.go

// AlertThresholds defines thresholds for security alerts
type AlertThresholds struct {
	FailedAuthAttempts  int           `json:"failed_auth_attempts"`
	TimeWindow          time.Duration `json:"time_window"`
	SuspiciousPatterns  int           `json:"suspicious_patterns"`
	RateLimitViolations int           `json:"rate_limit_violations"`
	UnauthorizedAccess  int           `json:"unauthorized_access"`
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	EventType       EventType              `json:"event_type"`
	Severity        EventSeverity          `json:"severity"`
	Actor           *Actor                 `json:"actor"`
	Action          *Action                `json:"action"`
	Resource        *Resource              `json:"resource"`
	Result          EventResult            `json:"result"`
	ErrorCode       string                 `json:"error_code,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	RequestID       string                 `json:"request_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
	ClientIP        string                 `json:"client_ip"`
	UserAgent       string                 `json:"user_agent,omitempty"`
	RequestMethod   string                 `json:"request_method,omitempty"`
	RequestPath     string                 `json:"request_path,omitempty"`
	RequestHeaders  map[string]string      `json:"request_headers,omitempty"`
	RequestBody     json.RawMessage        `json:"request_body,omitempty"`
	ResponseStatus  int                    `json:"response_status,omitempty"`
	ResponseHeaders map[string]string      `json:"response_headers,omitempty"`
	ResponseBody    json.RawMessage        `json:"response_body,omitempty"`
	Duration        time.Duration          `json:"duration,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	SecurityContext *SecurityContext       `json:"security_context,omitempty"`
	Signature       string                 `json:"signature,omitempty"`
}

// EventType represents the type of audit event
type EventType string

const (
	EventTypeAuthentication    EventType = "authentication"
	EventTypeAuthorization     EventType = "authorization"
	EventTypePolicyCreate      EventType = "policy_create"
	EventTypePolicyUpdate      EventType = "policy_update"
	EventTypePolicyDelete      EventType = "policy_delete"
	EventTypePolicyAccess      EventType = "policy_access"
	EventTypeDataAccess        EventType = "data_access"
	EventTypeDataModification  EventType = "data_modification"
	EventTypeConfigChange      EventType = "config_change"
	EventTypeSecurityViolation EventType = "security_violation"
	EventTypeRateLimitExceeded EventType = "rate_limit_exceeded"
	EventTypeMaliciousActivity EventType = "malicious_activity"
	EventTypeSystemError       EventType = "system_error"
	EventTypeAdminAction       EventType = "admin_action"
)

// EventSeverity represents the severity of an event
type EventSeverity string

const (
	SeverityDebug     EventSeverity = "debug"
	SeverityInfo      EventSeverity = "info"
	SeverityNotice    EventSeverity = "notice"
	SeverityWarning   EventSeverity = "warning"
	SeverityError     EventSeverity = "error"
	SeverityCritical  EventSeverity = "critical"
	SeverityAlert     EventSeverity = "alert"
	SeverityEmergency EventSeverity = "emergency"
)

// EventResult represents the result of an event
type EventResult string

const (
	ResultSuccess EventResult = "success"
	ResultFailure EventResult = "failure"
	ResultPartial EventResult = "partial"
	ResultDenied  EventResult = "denied"
	ResultError   EventResult = "error"
)

// Actor represents who performed the action
type Actor struct {
	Type        ActorType         `json:"type"`
	ID          string            `json:"id"`
	Username    string            `json:"username,omitempty"`
	Email       string            `json:"email,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// ActorType represents the type of actor
type ActorType string

const (
	ActorTypeUser    ActorType = "user"
	ActorTypeService ActorType = "service"
	ActorTypeSystem  ActorType = "system"
	ActorTypeUnknown ActorType = "unknown"
)

// Action represents what action was performed
type Action struct {
	Type       string                 `json:"type"`
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Resource represents what resource was affected
type Resource struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Name       string            `json:"name,omitempty"`
	Path       string            `json:"path,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SecurityContext contains security-related context
type SecurityContext struct {
	ThreatLevel      string          `json:"threat_level,omitempty"`
	RiskScore        float64         `json:"risk_score,omitempty"`
	Violations       []string        `json:"violations,omitempty"`
	ComplianceStatus map[string]bool `json:"compliance_status,omitempty"`
	Indicators       []string        `json:"indicators,omitempty"`
}

// AuditLogger manages security audit logging
type AuditLogger struct {
	config       *AuditConfig
	logger       *logging.StructuredLogger
	buffer       chan *AuditEvent
	destinations []AuditWriter
	encryptor    EventEncryptor
	signer       EventSigner
	mu           sync.RWMutex
	stats        *AuditStats
	alertManager *AlertManager
	shutdown     chan bool
	wg           sync.WaitGroup
}

// AuditWriter interface for writing audit events
type AuditWriter interface {
	Write(ctx context.Context, events []*AuditEvent) error
	Close() error
}

// EventEncryptor interface for encrypting audit events
type EventEncryptor interface {
	Encrypt(event *AuditEvent) (*AuditEvent, error)
	Decrypt(event *AuditEvent) (*AuditEvent, error)
}

// EventSigner interface for signing audit events
type EventSigner interface {
	Sign(event *AuditEvent) (string, error)
	Verify(event *AuditEvent, signature string) (bool, error)
}

// AuditStats tracks audit logging statistics
type AuditStats struct {
	mu               sync.RWMutex
	EventsLogged     int64
	EventsDropped    int64
	EventsByType     map[EventType]int64
	EventsBySeverity map[EventSeverity]int64
	FailedWrites     int64
	LastEventTime    time.Time
}

// AlertManager manages security alerts
type AlertManager struct {
	config     *AlertThresholds
	logger     *logging.StructuredLogger
	violations map[string]*ViolationTracker
	mu         sync.RWMutex
}

// ViolationTracker tracks security violations
type ViolationTracker struct {
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
	Alerted   bool
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditConfig, logger *logging.StructuredLogger) (*AuditLogger, error) {
	if config == nil {
		return nil, errors.New("audit config is required")
	}

	al := &AuditLogger{
		config:   config,
		logger:   logger,
		buffer:   make(chan *AuditEvent, config.BufferSize),
		shutdown: make(chan bool),
		stats: &AuditStats{
			EventsByType:     make(map[EventType]int64),
			EventsBySeverity: make(map[EventSeverity]int64),
		},
	}

	// Initialize destinations
	if err := al.initializeDestinations(); err != nil {
		return nil, fmt.Errorf("failed to initialize audit destinations: %w", err)
	}

	// Initialize encryption if enabled
	if config.EncryptEvents {
		al.encryptor = NewEventEncryptor()
	}

	// Initialize signing if enabled
	if config.SignEvents {
		al.signer = NewEventSigner()
	}

	// Initialize alert manager if enabled
	if config.AlertOnViolation && config.AlertThresholds != nil {
		al.alertManager = NewAlertManager(config.AlertThresholds, logger)
	}

	// Start background workers
	al.start()

	return al, nil
}

// initializeDestinations initializes audit destinations
func (al *AuditLogger) initializeDestinations() error {
	for _, dest := range al.config.Destinations {
		writer, err := al.createWriter(dest)
		if err != nil {
			return fmt.Errorf("failed to create writer for %s: %w", dest.Type, err)
		}
		al.destinations = append(al.destinations, writer)
	}
	return nil
}

// createWriter creates an audit writer based on destination type
func (al *AuditLogger) createWriter(dest AuditDestination) (AuditWriter, error) {
	switch dest.Type {
	case DestinationFile:
		return NewFileWriter(dest)
	case DestinationHTTP:
		return NewHTTPWriter(dest)
	case DestinationSyslog:
		return NewSyslogWriter(dest)
	default:
		return nil, fmt.Errorf("unsupported destination type: %s", dest.Type)
	}
}

// start starts background workers
func (al *AuditLogger) start() {
	// Start event processor
	al.wg.Add(1)
	go al.processEvents()

	// Start periodic flush
	al.wg.Add(1)
	go al.periodicFlush()
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) {
	if !al.config.Enabled {
		return
	}

	// Check audit level
	if !al.shouldLog(event) {
		return
	}

	// Generate event ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Extract context information
	al.enrichEventFromContext(ctx, event)

	// Mask sensitive data if configured
	if al.config.MaskSensitiveData {
		al.maskSensitiveData(event)
	}

	// Sign event if configured
	if al.config.SignEvents && al.signer != nil {
		signature, err := al.signer.Sign(event)
		if err != nil {
			al.logger.Error("failed to sign audit event", "error", err)
		} else {
			event.Signature = signature
		}
	}

	// Encrypt event if configured
	if al.config.EncryptEvents && al.encryptor != nil {
		encryptedEvent, err := al.encryptor.Encrypt(event)
		if err != nil {
			al.logger.Error("failed to encrypt audit event", "error", err)
		} else {
			event = encryptedEvent
		}
	}

	// Check for security violations
	if al.alertManager != nil {
		al.alertManager.CheckEvent(event)
	}

	// Send to buffer
	select {
	case al.buffer <- event:
		// Update statistics
		al.updateStats(event)
	default:
		// Buffer full, drop event
		al.stats.mu.Lock()
		al.stats.EventsDropped++
		al.stats.mu.Unlock()
		al.logger.Warn("audit event dropped due to full buffer",
			slog.String("event_id", event.ID),
			slog.String("event_type", string(event.EventType)))
	}
}

// LogAuthenticationEvent logs an authentication event
func (al *AuditLogger) LogAuthenticationEvent(ctx context.Context, username string, success bool, reason string) {
	result := ResultSuccess
	severity := SeverityInfo
	if !success {
		result = ResultFailure
		severity = SeverityWarning
	}

	event := &AuditEvent{
		EventType: EventTypeAuthentication,
		Severity:  severity,
		Actor: &Actor{
			Type:     ActorTypeUser,
			Username: username,
		},
		Action: &Action{
			Type:   "authenticate",
			Method: "login",
		},
		Result:       result,
		ErrorMessage: reason,
	}

	al.LogEvent(ctx, event)
}

// LogAuthorizationEvent logs an authorization event
func (al *AuditLogger) LogAuthorizationEvent(ctx context.Context, user *User, resource, action string, allowed bool) {
	result := ResultSuccess
	severity := SeverityInfo
	if !allowed {
		result = ResultDenied
		severity = SeverityWarning
	}

	event := &AuditEvent{
		EventType: EventTypeAuthorization,
		Severity:  severity,
		Actor: &Actor{
			Type:     ActorTypeUser,
			ID:       user.ID,
			Username: user.Username,
			Roles:    user.Roles,
		},
		Action: &Action{
			Type:   "authorize",
			Method: action,
		},
		Resource: &Resource{
			Type: "policy",
			Path: resource,
		},
		Result: result,
	}

	al.LogEvent(ctx, event)
}

// LogPolicyEvent logs a policy-related event
func (al *AuditLogger) LogPolicyEvent(ctx context.Context, eventType EventType, policyID string, actor *Actor, result EventResult) {
	event := &AuditEvent{
		EventType: eventType,
		Severity:  SeverityInfo,
		Actor:     actor,
		Action: &Action{
			Type: string(eventType),
		},
		Resource: &Resource{
			Type: "policy",
			ID:   policyID,
		},
		Result: result,
	}

	al.LogEvent(ctx, event)
}

// LogSecurityViolation logs a security violation
func (al *AuditLogger) LogSecurityViolation(ctx context.Context, violationType string, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: EventTypeSecurityViolation,
		Severity:  SeverityCritical,
		Action: &Action{
			Type:       "security_violation",
			Parameters: details,
		},
		Result: ResultFailure,
		SecurityContext: &SecurityContext{
			ThreatLevel: "high",
			Violations:  []string{violationType},
		},
	}

	al.LogEvent(ctx, event)
}

// processEvents processes buffered events
func (al *AuditLogger) processEvents() {
	defer al.wg.Done()

	batch := make([]*AuditEvent, 0, 100)
	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-al.buffer:
			batch = append(batch, event)
			if len(batch) >= 100 {
				al.writeBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				al.writeBatch(batch)
				batch = batch[:0]
			}

		case <-al.shutdown:
			// Flush remaining events
			if len(batch) > 0 {
				al.writeBatch(batch)
			}
			// Drain buffer
			for len(al.buffer) > 0 {
				batch = append(batch, <-al.buffer)
				if len(batch) >= 100 {
					al.writeBatch(batch)
					batch = batch[:0]
				}
			}
			if len(batch) > 0 {
				al.writeBatch(batch)
			}
			return
		}
	}
}

// writeBatch writes a batch of events to destinations
func (al *AuditLogger) writeBatch(events []*AuditEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, dest := range al.destinations {
		if err := dest.Write(ctx, events); err != nil {
			al.logger.Error("failed to write audit events",
				"error", err,
				"event_count", len(events))
			al.stats.mu.Lock()
			al.stats.FailedWrites++
			al.stats.mu.Unlock()
		}
	}
}

// periodicFlush performs periodic flush of events
func (al *AuditLogger) periodicFlush() {
	defer al.wg.Done()

	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Force flush logic if needed
		case <-al.shutdown:
			return
		}
	}
}

// shouldLog determines if an event should be logged based on audit level
func (al *AuditLogger) shouldLog(event *AuditEvent) bool {
	switch al.config.Level {
	case AuditLevelNone:
		return false
	case AuditLevelMinimal:
		return event.Severity >= SeverityWarning
	case AuditLevelStandard:
		return event.Severity >= SeverityInfo
	case AuditLevelDetailed:
		return event.Severity >= SeverityNotice
	case AuditLevelVerbose:
		return true
	default:
		return true
	}
}

// enrichEventFromContext enriches event with context information
func (al *AuditLogger) enrichEventFromContext(ctx context.Context, event *AuditEvent) {
	// Extract request ID from context
	if reqID := ctx.Value("request_id"); reqID != nil {
		event.RequestID = fmt.Sprintf("%v", reqID)
	}

	// Extract session ID from context
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		event.SessionID = fmt.Sprintf("%v", sessionID)
	}

	// Extract correlation ID from context
	if corrID := ctx.Value("correlation_id"); corrID != nil {
		event.CorrelationID = fmt.Sprintf("%v", corrID)
	}

	// Add runtime information
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["go_version"] = runtime.Version()
	event.Metadata["num_goroutines"] = runtime.NumGoroutine()
}

// maskSensitiveData masks sensitive data in the event
func (al *AuditLogger) maskSensitiveData(event *AuditEvent) {
	// Mask sensitive fields
	for _, field := range al.config.SensitiveFields {
		al.maskField(event, field)
	}

	// Mask common sensitive patterns
	if event.RequestBody != nil {
		masked := al.maskJSON(event.RequestBody)
		event.RequestBody = masked
	}

	if event.ResponseBody != nil {
		masked := al.maskJSON(event.ResponseBody)
		event.ResponseBody = masked
	}
}

// maskField masks a specific field in the event
func (al *AuditLogger) maskField(event *AuditEvent, field string) {
	// Implementation would mask specific fields
}

// maskJSON masks sensitive data in JSON
func (al *AuditLogger) maskJSON(data json.RawMessage) json.RawMessage {
	// Implementation would mask sensitive JSON fields
	return data
}

// updateStats updates audit statistics
func (al *AuditLogger) updateStats(event *AuditEvent) {
	al.stats.mu.Lock()
	defer al.stats.mu.Unlock()

	al.stats.EventsLogged++
	al.stats.EventsByType[event.EventType]++
	al.stats.EventsBySeverity[event.Severity]++
	al.stats.LastEventTime = event.Timestamp
}

// Close gracefully shuts down the audit logger
func (al *AuditLogger) Close() error {
	close(al.shutdown)
	al.wg.Wait()

	// Close all destinations
	for _, dest := range al.destinations {
		if err := dest.Close(); err != nil {
			al.logger.Error("failed to close audit destination", "error", err)
		}
	}

	return nil
}

// GetStats returns audit statistics
func (al *AuditLogger) GetStats() *AuditStats {
	al.stats.mu.RLock()
	defer al.stats.mu.RUnlock()

	// Return a copy of stats
	return &AuditStats{
		EventsLogged:     al.stats.EventsLogged,
		EventsDropped:    al.stats.EventsDropped,
		EventsByType:     al.stats.EventsByType,
		EventsBySeverity: al.stats.EventsBySeverity,
		FailedWrites:     al.stats.FailedWrites,
		LastEventTime:    al.stats.LastEventTime,
	}
}

// Implementation stubs for interfaces and helpers

func NewAlertManager(thresholds *AlertThresholds, logger *logging.StructuredLogger) *AlertManager {
	return &AlertManager{
		config:     thresholds,
		logger:     logger,
		violations: make(map[string]*ViolationTracker),
	}
}

func (am *AlertManager) CheckEvent(event *AuditEvent) {
	// Check for violations and raise alerts
}

func NewEventEncryptor() EventEncryptor {
	// Return default encryptor implementation
	return &defaultEventEncryptor{}
}

type defaultEventEncryptor struct{}

func (e *defaultEventEncryptor) Encrypt(event *AuditEvent) (*AuditEvent, error) {
	// Encryption implementation
	return event, nil
}

func (e *defaultEventEncryptor) Decrypt(event *AuditEvent) (*AuditEvent, error) {
	// Decryption implementation
	return event, nil
}

func NewEventSigner() EventSigner {
	// Return default signer implementation
	return &defaultEventSigner{}
}

type defaultEventSigner struct{}

func (s *defaultEventSigner) Sign(event *AuditEvent) (string, error) {
	// Create signature
	data, _ := json.Marshal(event)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (s *defaultEventSigner) Verify(event *AuditEvent, signature string) (bool, error) {
	// Verify signature
	return true, nil
}

// Audit writer implementations

func NewFileWriter(dest AuditDestination) (AuditWriter, error) {
	// File writer implementation
	return &fileWriter{dest: dest}, nil
}

type fileWriter struct {
	dest AuditDestination
}

func (fw *fileWriter) Write(ctx context.Context, events []*AuditEvent) error {
	// Write to file
	return nil
}

func (fw *fileWriter) Close() error {
	return nil
}

func NewHTTPWriter(dest AuditDestination) (AuditWriter, error) {
	// HTTP writer implementation
	return &httpWriter{dest: dest}, nil
}

type httpWriter struct {
	dest AuditDestination
}

func (hw *httpWriter) Write(ctx context.Context, events []*AuditEvent) error {
	// Send via HTTP
	return nil
}

func (hw *httpWriter) Close() error {
	return nil
}

func NewSyslogWriter(dest AuditDestination) (AuditWriter, error) {
	// Syslog writer implementation
	return &syslogWriter{dest: dest}, nil
}

type syslogWriter struct {
	dest AuditDestination
}

func (sw *syslogWriter) Write(ctx context.Context, events []*AuditEvent) error {
	// Write to syslog
	return nil
}

func (sw *syslogWriter) Close() error {
	return nil
}

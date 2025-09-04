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
)

// AuditConfig holds audit logging configuration.

type AuditConfig struct {
	Enabled bool `json:"enabled"`

	Level AuditLevel `json:"level"`

	Destinations []AuditDestination `json:"destinations"`

	BufferSize int `json:"buffer_size"`

	FlushInterval time.Duration `json:"flush_interval"`

	RetentionDays int `json:"retention_days"`

	CompressOldLogs bool `json:"compress_old_logs"`

	EncryptLogs bool `json:"encrypt_logs"`

	EncryptionKey string `json:"encryption_key,omitempty"`

	IncludeUserAgent bool `json:"include_user_agent"`

	IncludeSourceIP bool `json:"include_source_ip"`

	MaxEventSize int `json:"max_event_size"`

	SensitiveFields []string `json:"sensitive_fields"`
}

// AuditLevel defines the audit logging level.

type AuditLevel string

const (

	// AuditLevelNone does not log.

	AuditLevelNone AuditLevel = "none"

	// AuditLevelError only logs errors.

	AuditLevelError AuditLevel = "error"

	// AuditLevelWarn logs warnings and errors.

	AuditLevelWarn AuditLevel = "warn"

	// AuditLevelInfo logs info, warnings, and errors.

	AuditLevelInfo AuditLevel = "info"

	// AuditLevelDebug logs all events.

	AuditLevelDebug AuditLevel = "debug"
)

// AuditDestination defines where audit logs are sent.

type AuditDestination struct {
	Type DestinationType `json:"type"`

	Config json.RawMessage `json:"config"`

	Format string `json:"format"`

	Enabled bool `json:"enabled"`
}

// DestinationType defines the type of audit destination.

type DestinationType string

const (

	// DestinationTypeFile stores logs in files.

	DestinationTypeFile DestinationType = "file"

	// DestinationTypeSyslog sends logs to syslog.

	DestinationTypeSyslog DestinationType = "syslog"

	// DestinationTypeKafka sends logs to Kafka.

	DestinationTypeKafka DestinationType = "kafka"

	// DestinationTypeElasticsearch sends logs to Elasticsearch.

	DestinationTypeElasticsearch DestinationType = "elasticsearch"

	// DestinationTypeWebhook sends logs to a webhook.

	DestinationTypeWebhook DestinationType = "webhook"
)

// AuditLogger manages audit logging for security events.

type AuditLogger struct {
	config *AuditConfig

	buffer chan *AuditEvent

	destinations map[DestinationType]AuditWriter

	shutdown chan struct{}

	wg sync.WaitGroup

	mu sync.RWMutex

	logger *slog.Logger
}

// AuditWriter defines the interface for writing audit logs.

type AuditWriter interface {
	Write(ctx context.Context, event *AuditEvent) error

	Close() error
}

// AuditEvent represents a security audit event.

type AuditEvent struct {
	ID string `json:"id"`

	Timestamp time.Time `json:"timestamp"`

	EventType EventType `json:"event_type"`

	Severity Severity `json:"severity"`

	UserID string `json:"user_id,omitempty"`

	SessionID string `json:"session_id,omitempty"`

	CorrelationID string `json:"correlation_id,omitempty"`

	RemoteAddr string `json:"remote_addr,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	Action *Action `json:"action"`

	Resource *Resource `json:"resource"`

	Result Result `json:"result"`

	ResultMessage string `json:"result_message,omitempty"`

	Duration time.Duration `json:"duration,omitempty"`

	HTTPStatusCode int `json:"http_status_code,omitempty"`

	ResponseSize int64 `json:"response_size,omitempty"`

	ErrorCode string `json:"error_code,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`

	SecurityContext *SecurityContext `json:"security_context,omitempty"`
}

// EventType defines the type of audit event.

type EventType string

const (

	// EventTypeAuthentication represents authentication events.

	EventTypeAuthentication EventType = "authentication"

	// EventTypeAuthorization represents authorization events.

	EventTypeAuthorization EventType = "authorization"

	// EventTypeDataAccess represents data access events.

	EventTypeDataAccess EventType = "data_access"

	// EventTypeDataModification represents data modification events.

	EventTypeDataModification EventType = "data_modification"

	// EventTypeSystemAccess represents system access events.

	EventTypeSystemAccess EventType = "system_access"

	// EventTypeConfigurationChange represents configuration change events.

	EventTypeConfigurationChange EventType = "configuration_change"

	// EventTypeSecurityViolation represents security violation events.

	EventTypeSecurityViolation EventType = "security_violation"

	// EventTypeTokenOperation represents token operation events.

	EventTypeTokenOperation EventType = "token_operation"

	// EventTypeSessionManagement represents session management events.

	EventTypeSessionManagement EventType = "session_management"
)

// Severity defines the severity of an audit event.

type Severity string

const (

	// SeverityLow represents low severity events.

	SeverityLow Severity = "low"

	// SeverityMedium represents medium severity events.

	SeverityMedium Severity = "medium"

	// SeverityHigh represents high severity events.

	SeverityHigh Severity = "high"

	// SeverityCritical represents critical severity events.

	SeverityCritical Severity = "critical"
)

// Result defines the result of an audit event.

type Result string

const (

	// ResultSuccess represents a successful operation.

	ResultSuccess Result = "success"

	// ResultFailure represents a failed operation.

	ResultFailure Result = "failure"

	// ResultPartial represents a partially successful operation.

	ResultPartial Result = "partial"
)

// Action represents the action performed.

type Action struct {
	Type string `json:"type"`

	Method string `json:"method"`

	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// Resource represents what resource was affected.

type Resource struct {
	Type string `json:"type"`

	ID string `json:"id"`

	Name string `json:"name,omitempty"`

	Path string `json:"path,omitempty"`

	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// SecurityContext provides additional security information.

type SecurityContext struct {
	ThreatLevel string `json:"threat_level,omitempty"`

	RiskScore float64 `json:"risk_score,omitempty"`

	Classification string `json:"classification,omitempty"`

	GeoLocation string `json:"geo_location,omitempty"`

	DeviceFingerprint string `json:"device_fingerprint,omitempty"`

	Violations []string `json:"violations,omitempty"`

	MitigationActions []string `json:"mitigation_actions,omitempty"`
}

// NoOpAuditWriter is a simple audit writer that does nothing
type NoOpAuditWriter struct{}

// Write implements AuditWriter interface
func (w *NoOpAuditWriter) Write(ctx context.Context, event *AuditEvent) error {
	return nil
}

// Close implements AuditWriter interface
func (w *NoOpAuditWriter) Close() error {
	return nil
}

// NewAuditLogger creates a new audit logger.

func NewAuditLogger(config *AuditConfig) (*AuditLogger, error) {
	if config == nil {

		return nil, errors.New("audit config is required")
	}

	al := &AuditLogger{
		config:       config,
		buffer:       make(chan *AuditEvent, config.BufferSize),
		destinations: make(map[DestinationType]AuditWriter),
		shutdown:     make(chan struct{}),
		logger:       slog.Default(),
	}

	// Initialize destinations with no-op writers
	for _, dest := range config.Destinations {
		if dest.Enabled {
			al.destinations[dest.Type] = &NoOpAuditWriter{}
		}
	}

	// Start processing goroutine

	al.wg.Add(1)

	go al.processEvents()

	return al, nil
}

// LogEvent logs an audit event.

func (al *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) {
	if !al.config.Enabled {

		return
	}

	// Enrich event

	al.enrichEvent(ctx, event)

	// Mask sensitive data

	al.maskSensitiveData(event)

	// Check event size

	if al.exceedsMaxSize(event) {

		al.logger.Warn("audit event exceeds max size, truncating", "event_id", event.ID)

		al.truncateEvent(event)
	}

	// Send to buffer

	select {

	case al.buffer <- event:

	default:

		// Buffer is full, drop event

		al.logger.Warn("audit buffer full, dropping event", "event_id", event.ID)
	}
}

// LogAuthentication logs an authentication event.

func (al *AuditLogger) LogAuthentication(ctx context.Context, userID string, result Result, method string) {
	event := &AuditEvent{
		EventType: EventTypeAuthentication,

		Severity: SeverityMedium,

		UserID: userID,

		Action: &Action{
			Type:   "login",
			Method: method,
		},

		Result: result,
	}

	if result == ResultFailure {

		event.Severity = SeverityHigh
	}

	al.LogEvent(ctx, event)
}

// LogAuthorization logs an authorization event.

func (al *AuditLogger) LogAuthorization(ctx context.Context, userID string, resource *Resource, action string, result Result) {
	event := &AuditEvent{
		EventType: EventTypeAuthorization,

		Severity: SeverityMedium,

		UserID: userID,

		Action: &Action{
			Type:   action,
			Method: "RBAC",
		},

		Resource: resource,

		Result: result,
	}

	if result == ResultFailure {

		event.Severity = SeverityHigh
	}

	al.LogEvent(ctx, event)
}

// LogDataAccess logs a data access event.

func (al *AuditLogger) LogDataAccess(ctx context.Context, userID string, resource *Resource, action string) {
	event := &AuditEvent{
		EventType: EventTypeDataAccess,

		Severity: SeverityLow,

		UserID: userID,

		Action: &Action{
			Type: action,
		},

		Resource: resource,

		Result: ResultSuccess,
	}

	al.LogEvent(ctx, event)
}

// LogDataModification logs a data modification event.

func (al *AuditLogger) LogDataModification(ctx context.Context, userID string, resource *Resource, action string, oldValue, newValue interface{}) {
	// Marshal modification details to JSON
	modDetails := map[string]interface{}{
		"old_value": oldValue,
		"new_value": newValue,
	}
	
	var parametersJSON json.RawMessage
	if modDetailsJSON, err := json.Marshal(modDetails); err == nil {
		parametersJSON = json.RawMessage(modDetailsJSON)
	}

	event := &AuditEvent{
		EventType: EventTypeDataModification,

		Severity: SeverityMedium,

		UserID: userID,

		Action: &Action{
			Type:       action,
			Parameters: parametersJSON,
		},

		Resource: resource,

		Result: ResultSuccess,
	}

	al.LogEvent(ctx, event)
}

// LogConfigurationChange logs a configuration change event.

func (al *AuditLogger) LogConfigurationChange(ctx context.Context, userID string, configPath string, oldConfig, newConfig interface{}) {
	// Marshal configuration change details to JSON
	changeDetails := map[string]interface{}{
		"config_path": configPath,
		"old_config":  oldConfig,
		"new_config":  newConfig,
	}
	
	var parametersJSON json.RawMessage
	if changeDetailsJSON, err := json.Marshal(changeDetails); err == nil {
		parametersJSON = json.RawMessage(changeDetailsJSON)
	}

	event := &AuditEvent{
		EventType: EventTypeConfigurationChange,

		Severity: SeverityHigh,

		UserID: userID,

		Action: &Action{
			Type:       "config_change",
			Parameters: parametersJSON,
		},

		Resource: &Resource{
			Type: "configuration",
			Path: configPath,
		},

		Result: ResultSuccess,
	}

	al.LogEvent(ctx, event)
}

// LogTokenOperation logs a token operation event.

func (al *AuditLogger) LogTokenOperation(ctx context.Context, userID string, operation string, tokenType string) {
	// Marshal token operation details to JSON
	tokenDetails := map[string]interface{}{
		"operation":  operation,
		"token_type": tokenType,
	}
	
	var parametersJSON json.RawMessage
	if tokenDetailsJSON, err := json.Marshal(tokenDetails); err == nil {
		parametersJSON = json.RawMessage(tokenDetailsJSON)
	}

	event := &AuditEvent{
		EventType: EventTypeTokenOperation,

		Severity: SeverityMedium,

		UserID: userID,

		Action: &Action{
			Type:       operation,
			Parameters: parametersJSON,
		},

		Result: ResultSuccess,
	}

	al.LogEvent(ctx, event)
}

// LogSecurityViolation logs a security violation.

func (al *AuditLogger) LogSecurityViolation(ctx context.Context, violationType string, details map[string]interface{}) {
	// Marshal details to JSON for Parameters field
	var parametersJSON json.RawMessage
	if detailsJSON, err := json.Marshal(details); err == nil {
		parametersJSON = json.RawMessage(detailsJSON)
	}

	event := &AuditEvent{
		EventType: EventTypeSecurityViolation,

		Severity: SeverityCritical,

		Action: &Action{
			Type: "security_violation",
			Parameters: parametersJSON,
		},

		Result: ResultFailure,

		SecurityContext: &SecurityContext{
			ThreatLevel: "high",

			Violations: []string{violationType},
		},
	}

	al.LogEvent(ctx, event)
}

// processEvents processes buffered events.

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

			return
		}
	}
}

// writeBatch writes a batch of events to all destinations.

func (al *AuditLogger) writeBatch(events []*AuditEvent) {
	al.mu.RLock()

	defer al.mu.RUnlock()

	ctx := context.Background()

	for destType, writer := range al.destinations {

		for _, event := range events {

			if err := writer.Write(ctx, event); err != nil {

				al.logger.Error("failed to write audit event",
					"destination", destType,
					"event_id", event.ID,
					"error", err)
			}
		}
	}
}

// enrichEvent enriches an audit event with contextual information.

func (al *AuditLogger) enrichEvent(ctx context.Context, event *AuditEvent) {
	// Set ID and timestamp.

	if event.ID == "" {

		event.ID = uuid.New().String()
	}

	if event.Timestamp.IsZero() {

		event.Timestamp = time.Now().UTC()
	}

	// Extract user ID from context if not set.

	if event.UserID == "" {

		if userID := ctx.Value("user_id"); userID != nil {

			event.UserID = fmt.Sprintf("%v", userID)
		}
	}

	// Extract session ID from context.

	if sessionID := ctx.Value("session_id"); sessionID != nil {

		event.SessionID = fmt.Sprintf("%v", sessionID)
	}

	// Extract correlation ID from context.

	if corrID := ctx.Value("correlation_id"); corrID != nil {

		event.CorrelationID = fmt.Sprintf("%v", corrID)
	}

	// Add runtime information.
	metadataMap := make(map[string]interface{})
	if event.Metadata != nil {
		// Try to unmarshal existing metadata
		if err := json.Unmarshal(event.Metadata, &metadataMap); err != nil {
			// If unmarshal fails, start with empty map
			metadataMap = make(map[string]interface{})
		}
	}

	metadataMap["go_version"] = runtime.Version()
	metadataMap["num_goroutines"] = runtime.NumGoroutine()

	// Marshal back to JSON
	if metadataJSON, err := json.Marshal(metadataMap); err == nil {
		event.Metadata = json.RawMessage(metadataJSON)
	}
}

// maskSensitiveData masks sensitive data in the event.

func (al *AuditLogger) maskSensitiveData(event *AuditEvent) {
	// Mask sensitive fields.

	for _, field := range al.config.SensitiveFields {

		al.maskField(event, field)
	}
}

// maskField masks a specific field in the event.

func (al *AuditLogger) maskField(event *AuditEvent, field string) {
	// Implementation depends on field structure

	// This is a simplified version

	switch field {

	case "password":

		// Mask password in action parameters

		if event.Action != nil && event.Action.Parameters != nil {

			var params map[string]interface{}

			if err := json.Unmarshal(event.Action.Parameters, &params); err == nil {

				if _, exists := params["password"]; exists {

					params["password"] = "***MASKED***"

					if masked, err := json.Marshal(params); err == nil {

						event.Action.Parameters = json.RawMessage(masked)
					}
				}
			}
		}

	case "token":

		// Mask token in action parameters

		if event.Action != nil && event.Action.Parameters != nil {

			var params map[string]interface{}

			if err := json.Unmarshal(event.Action.Parameters, &params); err == nil {

				if _, exists := params["token"]; exists {

					params["token"] = "***MASKED***"

					if masked, err := json.Marshal(params); err == nil {

						event.Action.Parameters = json.RawMessage(masked)
					}
				}
			}
		}

	case "api_key":

		// Mask API key in action parameters

		if event.Action != nil && event.Action.Parameters != nil {

			var params map[string]interface{}

			if err := json.Unmarshal(event.Action.Parameters, &params); err == nil {

				if _, exists := params["api_key"]; exists {

					params["api_key"] = "***MASKED***"

					if masked, err := json.Marshal(params); err == nil {

						event.Action.Parameters = json.RawMessage(masked)
					}
				}
			}
		}
	}
}

// exceedsMaxSize checks if an event exceeds the maximum size.

func (al *AuditLogger) exceedsMaxSize(event *AuditEvent) bool {
	if al.config.MaxEventSize <= 0 {

		return false
	}

	eventJSON, err := json.Marshal(event)

	if err != nil {

		return false
	}

	return len(eventJSON) > al.config.MaxEventSize
}

// truncateEvent truncates an event to fit within the maximum size.

func (al *AuditLogger) truncateEvent(event *AuditEvent) {
	// Start by removing less critical fields

	// Remove user agent if present

	if len(event.UserAgent) > 0 {

		event.UserAgent = ""
	}

	// Truncate metadata if present

	if event.Metadata != nil {

		event.Metadata = json.RawMessage(`{"truncated": true}`)
	}

	// Truncate action parameters if present

	if event.Action != nil && event.Action.Parameters != nil {

		event.Action.Parameters = json.RawMessage(`{"truncated": true}`)
	}

	// Truncate resource attributes if present

	if event.Resource != nil && event.Resource.Attributes != nil {

		event.Resource.Attributes = json.RawMessage(`{"truncated": true}`)
	}
}

// Close shuts down the audit logger gracefully.

func (al *AuditLogger) Close() error {
	// Signal shutdown

	close(al.shutdown)

	// Wait for processing to complete

	al.wg.Wait()

	// Close all writers

	al.mu.Lock()

	defer al.mu.Unlock()

	var errs []error

	for destType, writer := range al.destinations {

		if err := writer.Close(); err != nil {

			errs = append(errs, fmt.Errorf("failed to close %s writer: %w", destType, err))
		}
	}

	// Return combined errors

	if len(errs) > 0 {

		return fmt.Errorf("errors closing audit writers: %v", errs)
	}

	return nil
}

// GetMetrics returns audit logger metrics.

func (al *AuditLogger) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"buffer_size":        cap(al.buffer),
		"buffer_used":        len(al.buffer),
		"destinations_count": len(al.destinations),
		"enabled":            al.config.Enabled,
	}
}

// Hash returns a SHA-256 hash of the audit logger configuration.

func (al *AuditLogger) Hash() string {
	configJSON, err := json.Marshal(al.config)

	if err != nil {

		return ""
	}

	hash := sha256.Sum256(configJSON)

	return hex.EncodeToString(hash[:])
}
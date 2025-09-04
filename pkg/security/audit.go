package security

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// AuditLogger provides secure audit logging capabilities.

type AuditLogger struct {
	logger logr.Logger

	enabled bool

	logFile *os.File

	minLevel interfaces.AuditLevel
}

// AuditEvent represents a security audit event.

type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`

	Level interfaces.AuditLevel `json:"level"`

	Event string `json:"event"`

	Component string `json:"component"`

	UserID string `json:"user_id,omitempty"`

	SessionID string `json:"session_id,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Data json.RawMessage `json:"data,omitempty"`

	Result string `json:"result"`

	Error string `json:"error,omitempty"`
}

// NewAuditLogger creates a new audit logger.

func NewAuditLogger(logFilePath string, minLevel interfaces.AuditLevel) (*AuditLogger, error) {
	logger := log.Log.WithName("audit-logger")

	al := &AuditLogger{
		logger: logger,

		enabled: true,

		minLevel: minLevel,
	}

	// If log file path is provided, open the file.

	if logFilePath != "" {

		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}

		al.logFile = file

	}

	return al, nil
}

// LogSecretAccess logs when secrets are accessed.

func (al *AuditLogger) LogSecretAccess(secretType, source, userID, sessionID string, success bool, err error) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: interfaces.AuditLevelInfo,

		Event: "secret_access",

		Component: "secret_manager",

		UserID: userID,

		SessionID: sessionID,

		Data: json.RawMessage(`{}`),

		Result: getResultString(success),
	}

	if err != nil {

		event.Level = interfaces.AuditLevelError

		event.Error = err.Error()

	}

	al.log(event)
}

// LogAuthenticationAttempt logs authentication attempts.

func (al *AuditLogger) LogAuthenticationAttempt(provider, userID, ipAddress, userAgent string, success bool, err error) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: interfaces.AuditLevelInfo,

		Event: "authentication_attempt",

		Component: "auth_middleware",

		UserID: userID,

		IPAddress: ipAddress,

		UserAgent: userAgent,

		Data: json.RawMessage(`{}`),

		Result: getResultString(success),
	}

	if err != nil {

		event.Level = interfaces.AuditLevelWarn

		event.Error = err.Error()

	}

	al.log(event)
}

// LogSecretRotation logs secret rotation events.

func (al *AuditLogger) LogSecretRotation(secretName, rotationType, userID string, success bool, err error) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: interfaces.AuditLevelInfo,

		Event: "secret_rotation",

		Component: "secret_manager",

		UserID: userID,

		Data: json.RawMessage(`{}`),

		Result: getResultString(success),
	}

	if err != nil {

		event.Level = interfaces.AuditLevelError

		event.Error = err.Error()

	}

	al.log(event)
}

// LogAPIKeyValidation logs API key validation events.

func (al *AuditLogger) LogAPIKeyValidation(keyType, provider string, success bool, err error) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: interfaces.AuditLevelInfo,

		Event: "api_key_validation",

		Component: "secret_loader",

		Data: json.RawMessage(`{}`),

		Result: getResultString(success),
	}

	if err != nil {

		event.Level = interfaces.AuditLevelWarn

		event.Error = err.Error()

	}

	al.log(event)
}

// LogUnauthorizedAccess logs unauthorized access attempts.

func (al *AuditLogger) LogUnauthorizedAccess(resource, userID, ipAddress, userAgent, reason string) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: interfaces.AuditLevelWarn,

		Event: "unauthorized_access",

		Component: "auth_middleware",

		UserID: userID,

		IPAddress: ipAddress,

		UserAgent: userAgent,

		Data: json.RawMessage(`{}`),

		Result: "denied",
	}

	al.log(event)
}

// LogSecurityViolation logs security violations.

func (al *AuditLogger) LogSecurityViolation(violationType, description, userID, ipAddress string, severity interfaces.AuditLevel) {
	event := &AuditEvent{
		Timestamp: time.Now().UTC(),

		Level: severity,

		Event: "security_violation",

		Component: "security_scanner",

		UserID: userID,

		IPAddress: ipAddress,

		Data: json.RawMessage(`{}`),

		Result: "violation_detected",
	}

	al.log(event)
}

// log writes the audit event to the configured outputs.

func (al *AuditLogger) log(event *AuditEvent) {
	if !al.enabled || event.Level < al.minLevel {
		return
	}

	// Log to structured logger.

	logFunc := al.logger.Info

	switch event.Level {

	case interfaces.AuditLevelWarn:

		logFunc = al.logger.Info // Use Info for warnings in structured logs

	case interfaces.AuditLevelError:

		logFunc = func(msg string, keysAndValues ...interface{}) {
			al.logger.Error(nil, msg, keysAndValues...)
		}

	case interfaces.AuditLevelCritical:

		logFunc = func(msg string, keysAndValues ...interface{}) {
			al.logger.Error(nil, msg, keysAndValues...)
		}

	}

	logFunc("AUDIT: "+event.Event,

		"component", event.Component,

		"user_id", event.UserID,

		"session_id", event.SessionID,

		"ip_address", event.IPAddress,

		"result", event.Result,

		"error", event.Error,

		"data", event.Data,
	)

	// Write to audit log file if configured.

	if al.logFile != nil {

		jsonData, err := json.Marshal(event)
		if err != nil {

			al.logger.Error(err, "Failed to marshal audit event to JSON")

			return

		}

		_, err = al.logFile.WriteString(string(jsonData) + "\n")
		if err != nil {
			al.logger.Error(err, "Failed to write audit event to file")
		}

	}
}

// Close closes the audit logger and any open files.

func (al *AuditLogger) Close() error {
	if al.logFile != nil {
		return al.logFile.Close()
	}

	return nil
}

// SetEnabled enables or disables audit logging.

func (al *AuditLogger) SetEnabled(enabled bool) {
	al.enabled = enabled
}

// IsEnabled returns whether audit logging is enabled.

func (al *AuditLogger) IsEnabled() bool {
	return al.enabled
}

// getResultString converts boolean success to string.

func getResultString(success bool) string {
	if success {
		return "success"
	}

	return "failure"
}

// GlobalAuditLogger provides global access to audit logging functionality.

var GlobalAuditLogger *AuditLogger

// InitGlobalAuditLogger initializes the global audit logger.

func InitGlobalAuditLogger(logFilePath string, minLevel interfaces.AuditLevel) error {
	var err error

	GlobalAuditLogger, err = NewAuditLogger(logFilePath, minLevel)

	return err
}

// AuditSecretAccess is a convenience function for logging secret access.

func AuditSecretAccess(secretType, source, userID, sessionID string, success bool, err error) {
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogSecretAccess(secretType, source, userID, sessionID, success, err)
	}
}

// AuditAuthenticationAttempt is a convenience function for logging auth attempts.

func AuditAuthenticationAttempt(provider, userID, ipAddress, userAgent string, success bool, err error) {
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogAuthenticationAttempt(provider, userID, ipAddress, userAgent, success, err)
	}
}

// Ensure AuditLogger implements interfaces.AuditLogger.

var _ interfaces.AuditLogger = (*AuditLogger)(nil)

package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"go.opentelemetry.io/otel/trace"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// StructuredLogger provides structured logging with correlation IDs and tracing
type StructuredLogger struct {
	logger       *slog.Logger
	serviceName  string
	environment  string
	version      string
	defaultAttrs []slog.Attr
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level       string `json:"level"`
	Format      string `json:"format"` // "json" or "text"
	ServiceName string `json:"service_name"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Output      string `json:"output"` // "stdout", "stderr", or file path
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:       "info",
		Format:      "json",
		ServiceName: "nephoran-intent-operator",
		Environment: shared.GetEnv("NEPHORAN_ENVIRONMENT", "production"),
		Version:     shared.GetEnv("NEPHORAN_VERSION", "1.0.0"),
		Output:      "stdout",
	}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) (*StructuredLogger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Configure log level
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure output
	var output *os.File
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		var err error
		output, err = os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	// Configure handler
	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize attribute formatting
				if a.Key == slog.TimeKey {
					return slog.Attr{
						Key:   "timestamp",
						Value: slog.StringValue(time.Now().UTC().Format(time.RFC3339Nano)),
					}
				}
				if a.Key == slog.LevelKey {
					return slog.Attr{
						Key:   "level",
						Value: slog.StringValue(a.Value.String()),
					}
				}
				if a.Key == slog.MessageKey {
					return slog.Attr{
						Key:   "message",
						Value: a.Value,
					}
				}
				return a
			},
		})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	logger := slog.New(handler)

	// Default attributes
	defaultAttrs := []slog.Attr{
		slog.String("service", config.ServiceName),
		slog.String("environment", config.Environment),
		slog.String("version", config.Version),
	}

	return &StructuredLogger{
		logger:       logger,
		serviceName:  config.ServiceName,
		environment:  config.Environment,
		version:      config.Version,
		defaultAttrs: defaultAttrs,
	}, nil
}

// ContextKey type for context keys
type ContextKey string

const (
	CorrelationIDKey ContextKey = "correlation_id"
	UserIDKey        ContextKey = "user_id"
	RequestIDKey     ContextKey = "request_id"
	OperationIDKey   ContextKey = "operation_id"
)

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithOperationID adds operation ID to context
func WithOperationID(ctx context.Context, operationID string) context.Context {
	if operationID == "" {
		operationID = uuid.New().String()
	}
	return context.WithValue(ctx, OperationIDKey, operationID)
}

// GetCorrelationID gets correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestID gets request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetOperationID gets operation ID from context
func GetOperationID(ctx context.Context) string {
	if id, ok := ctx.Value(OperationIDKey).(string); ok {
		return id
	}
	return ""
}

// extractContextAttributes extracts logging attributes from context
func (sl *StructuredLogger) extractContextAttributes(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	// Add correlation ID
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		attrs = append(attrs, slog.String("correlation_id", correlationID))
	}

	// Add request ID
	if requestID := GetRequestID(ctx); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	// Add operation ID
	if operationID := GetOperationID(ctx); operationID != "" {
		attrs = append(attrs, slog.String("operation_id", operationID))
	}

	// Add trace information if available
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	return attrs
}

// buildLogAttributes builds complete set of log attributes
func (sl *StructuredLogger) buildLogAttributes(ctx context.Context, attrs ...slog.Attr) []slog.Attr {
	allAttrs := make([]slog.Attr, 0, len(sl.defaultAttrs)+len(attrs)+5)

	// Add default attributes
	allAttrs = append(allAttrs, sl.defaultAttrs...)

	// Add context attributes
	allAttrs = append(allAttrs, sl.extractContextAttributes(ctx)...)

	// Add provided attributes
	allAttrs = append(allAttrs, attrs...)

	return allAttrs
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	allAttrs := sl.buildLogAttributes(ctx, attrs...)
	sl.logger.LogAttrs(ctx, slog.LevelDebug, msg, allAttrs...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	allAttrs := sl.buildLogAttributes(ctx, attrs...)
	sl.logger.LogAttrs(ctx, slog.LevelInfo, msg, allAttrs...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	allAttrs := sl.buildLogAttributes(ctx, attrs...)
	sl.logger.LogAttrs(ctx, slog.LevelWarn, msg, allAttrs...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	allAttrs := sl.buildLogAttributes(ctx, attrs...)
	if err != nil {
		allAttrs = append(allAttrs, slog.String("error", err.Error()))
		allAttrs = append(allAttrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}
	sl.logger.LogAttrs(ctx, slog.LevelError, msg, allAttrs...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	allAttrs := sl.buildLogAttributes(ctx, attrs...)
	if err != nil {
		allAttrs = append(allAttrs, slog.String("error", err.Error()))
		allAttrs = append(allAttrs, slog.String("error_type", fmt.Sprintf("%T", err)))
	}
	sl.logger.LogAttrs(ctx, slog.LevelError, msg, allAttrs...)
	os.Exit(1)
}

// WithFields returns a logger with additional fields
func (sl *StructuredLogger) WithFields(attrs ...slog.Attr) *StructuredLogger {
	newDefaultAttrs := make([]slog.Attr, 0, len(sl.defaultAttrs)+len(attrs))
	newDefaultAttrs = append(newDefaultAttrs, sl.defaultAttrs...)
	newDefaultAttrs = append(newDefaultAttrs, attrs...)

	return &StructuredLogger{
		logger:       sl.logger,
		serviceName:  sl.serviceName,
		environment:  sl.environment,
		version:      sl.version,
		defaultAttrs: newDefaultAttrs,
	}
}

// NetworkIntentLogger provides specialized logging for NetworkIntent operations
type NetworkIntentLogger struct {
	*StructuredLogger
}

// NewNetworkIntentLogger creates a NetworkIntent-specific logger
func NewNetworkIntentLogger(baseLogger *StructuredLogger) *NetworkIntentLogger {
	return &NetworkIntentLogger{
		StructuredLogger: baseLogger.WithFields(
			slog.String("component", "networkintent-controller"),
		),
	}
}

// LogReconciliationStart logs the start of reconciliation
func (nil *NetworkIntentLogger) LogReconciliationStart(ctx context.Context, name, namespace string) {
	nil.Info(ctx, "Starting NetworkIntent reconciliation",
		slog.String("intent_name", name),
		slog.String("intent_namespace", namespace),
		slog.String("operation", "reconciliation_start"),
	)
}

// LogReconciliationComplete logs successful reconciliation completion
func (nil *NetworkIntentLogger) LogReconciliationComplete(ctx context.Context, name, namespace string, duration time.Duration) {
	nil.Info(ctx, "NetworkIntent reconciliation completed successfully",
		slog.String("intent_name", name),
		slog.String("intent_namespace", namespace),
		slog.Duration("duration", duration),
		slog.String("operation", "reconciliation_complete"),
		slog.String("status", "success"),
	)
}

// LogReconciliationError logs reconciliation errors
func (nil *NetworkIntentLogger) LogReconciliationError(ctx context.Context, name, namespace string, err error, duration time.Duration) {
	nil.Error(ctx, "NetworkIntent reconciliation failed", err,
		slog.String("intent_name", name),
		slog.String("intent_namespace", namespace),
		slog.Duration("duration", duration),
		slog.String("operation", "reconciliation_error"),
		slog.String("status", "failed"),
	)
}

// LogLLMProcessing logs LLM processing events
func (nil *NetworkIntentLogger) LogLLMProcessing(ctx context.Context, name, namespace, model string, tokensUsed int, cost float64, duration time.Duration) {
	nil.Info(ctx, "LLM processing completed",
		slog.String("intent_name", name),
		slog.String("intent_namespace", namespace),
		slog.String("llm_model", model),
		slog.Int("tokens_used", tokensUsed),
		slog.Float64("cost_usd", cost),
		slog.Duration("duration", duration),
		slog.String("operation", "llm_processing"),
	)
}

// LogStatusTransition logs status transitions
func (nil *NetworkIntentLogger) LogStatusTransition(ctx context.Context, name, namespace, fromStatus, toStatus string) {
	nil.Info(ctx, "NetworkIntent status transition",
		slog.String("intent_name", name),
		slog.String("intent_namespace", namespace),
		slog.String("from_status", fromStatus),
		slog.String("to_status", toStatus),
		slog.String("operation", "status_transition"),
	)
}

// ORANInterfaceLogger provides specialized logging for O-RAN interface operations
type ORANInterfaceLogger struct {
	*StructuredLogger
}

// NewORANInterfaceLogger creates an O-RAN interface-specific logger
func NewORANInterfaceLogger(baseLogger *StructuredLogger) *ORANInterfaceLogger {
	return &ORANInterfaceLogger{
		StructuredLogger: baseLogger.WithFields(
			slog.String("component", "oran-interface"),
		),
	}
}

// LogInterfaceRequest logs O-RAN interface requests
func (oil *ORANInterfaceLogger) LogInterfaceRequest(ctx context.Context, interfaceType, operation, endpoint string, duration time.Duration, statusCode int) {
	level := slog.LevelInfo
	if statusCode >= 400 {
		level = slog.LevelWarn
	}
	if statusCode >= 500 {
		level = slog.LevelError
	}

	oil.logger.LogAttrs(ctx, level, "O-RAN interface request completed",
		oil.buildLogAttributes(ctx,
			slog.String("oran_interface", interfaceType),
			slog.String("operation", operation),
			slog.String("endpoint", endpoint),
			slog.Duration("duration", duration),
			slog.Int("status_code", statusCode),
			slog.String("operation_type", "interface_request"),
		)...,
	)
}

// LogConnectionStatus logs connection status changes
func (oil *ORANInterfaceLogger) LogConnectionStatus(ctx context.Context, interfaceType, endpoint string, connected bool) {
	status := "disconnected"
	level := slog.LevelError
	if connected {
		status = "connected"
		level = slog.LevelInfo
	}

	oil.logger.LogAttrs(ctx, level, "O-RAN interface connection status changed",
		oil.buildLogAttributes(ctx,
			slog.String("oran_interface", interfaceType),
			slog.String("endpoint", endpoint),
			slog.String("connection_status", status),
			slog.Bool("connected", connected),
			slog.String("operation_type", "connection_status"),
		)...,
	)
}

// LogPolicyOperation logs policy operations
func (oil *ORANInterfaceLogger) LogPolicyOperation(ctx context.Context, policyType, operation, policyID string, success bool, duration time.Duration) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}

	oil.logger.LogAttrs(ctx, level, "O-RAN policy operation completed",
		oil.buildLogAttributes(ctx,
			slog.String("policy_type", policyType),
			slog.String("operation", operation),
			slog.String("policy_id", policyID),
			slog.Bool("success", success),
			slog.Duration("duration", duration),
			slog.String("operation_type", "policy_operation"),
		)...,
	)
}

// SecurityLogger provides specialized logging for security events
type SecurityLogger struct {
	*StructuredLogger
}

// NewSecurityLogger creates a security-specific logger
func NewSecurityLogger(baseLogger *StructuredLogger) *SecurityLogger {
	return &SecurityLogger{
		StructuredLogger: baseLogger.WithFields(
			slog.String("component", "security"),
			slog.String("log_type", "security_event"),
		),
	}
}

// LogAuthenticationEvent logs authentication events
func (sl *SecurityLogger) LogAuthenticationEvent(ctx context.Context, userID, method string, success bool, clientIP string) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}

	sl.logger.LogAttrs(ctx, level, "Authentication event",
		sl.buildLogAttributes(ctx,
			slog.String("user_id", userID),
			slog.String("auth_method", method),
			slog.Bool("success", success),
			slog.String("client_ip", clientIP),
			slog.String("event_type", "authentication"),
		)...,
	)
}

// LogAuthorizationEvent logs authorization events
func (sl *SecurityLogger) LogAuthorizationEvent(ctx context.Context, userID, resource, action string, allowed bool) {
	level := slog.LevelInfo
	if !allowed {
		level = slog.LevelWarn
	}

	sl.logger.LogAttrs(ctx, level, "Authorization event",
		sl.buildLogAttributes(ctx,
			slog.String("user_id", userID),
			slog.String("resource", resource),
			slog.String("action", action),
			slog.Bool("allowed", allowed),
			slog.String("event_type", "authorization"),
		)...,
	)
}

// LogSecurityIncident logs security incidents
func (sl *SecurityLogger) LogSecurityIncident(ctx context.Context, incidentType, description string, severity string, sourceIP string) {
	level := slog.LevelError
	switch severity {
	case "low":
		level = slog.LevelWarn
	case "medium":
		level = slog.LevelWarn
	case "high":
		level = slog.LevelError
	case "critical":
		level = slog.LevelError
	}

	sl.logger.LogAttrs(ctx, level, "Security incident detected",
		sl.buildLogAttributes(ctx,
			slog.String("incident_type", incidentType),
			slog.String("description", description),
			slog.String("severity", severity),
			slog.String("source_ip", sourceIP),
			slog.String("event_type", "security_incident"),
		)...,
	)
}

// AuditLogger provides specialized logging for audit events
type AuditLogger struct {
	*StructuredLogger
}

// NewAuditLogger creates an audit-specific logger
func NewAuditLogger(baseLogger *StructuredLogger) *AuditLogger {
	return &AuditLogger{
		StructuredLogger: baseLogger.WithFields(
			slog.String("component", "audit"),
			slog.String("log_type", "audit_event"),
		),
	}
}

// LogResourceOperation logs resource operations for audit
func (al *AuditLogger) LogResourceOperation(ctx context.Context, userID, operation, resourceType, resourceName, namespace string, success bool) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelWarn
	}

	al.logger.LogAttrs(ctx, level, "Resource operation",
		al.buildLogAttributes(ctx,
			slog.String("user_id", userID),
			slog.String("operation", operation),
			slog.String("resource_type", resourceType),
			slog.String("resource_name", resourceName),
			slog.String("namespace", namespace),
			slog.Bool("success", success),
			slog.String("event_type", "resource_operation"),
		)...,
	)
}

// LogConfigurationChange logs configuration changes
func (al *AuditLogger) LogConfigurationChange(ctx context.Context, userID, configType, configName string, oldValue, newValue interface{}) {
	al.Info(ctx, "Configuration change",
		slog.String("user_id", userID),
		slog.String("config_type", configType),
		slog.String("config_name", configName),
		slog.Any("old_value", oldValue),
		slog.Any("new_value", newValue),
		slog.String("event_type", "configuration_change"),
	)
}

// Global logger instance
var DefaultLogger *StructuredLogger

// InitializeLogging initializes the global logger
func InitializeLogging(config *LogConfig) error {
	logger, err := NewStructuredLogger(config)
	if err != nil {
		return err
	}
	DefaultLogger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *StructuredLogger {
	return DefaultLogger
}

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// StructuredLogger provides enhanced structured logging capabilities
type StructuredLogger struct {
	*slog.Logger
	serviceName    string
	serviceVersion string
	environment    string
	component      string
	requestID      string
}

// Config holds configuration for the structured logger
type Config struct {
	Level       LogLevel `json:"level"`
	Format      string   `json:"format"` // "json" or "text"
	ServiceName string   `json:"service_name"`
	Version     string   `json:"version"`
	Environment string   `json:"environment"`
	Component   string   `json:"component"`
	AddSource   bool     `json:"add_source"`
	TimeFormat  string   `json:"time_format"`
}

// RequestContext holds request-specific logging context
type RequestContext struct {
	RequestID string
	UserID    string
	TraceID   string
	SpanID    string
	Method    string
	Path      string
	IP        string
	UserAgent string
}

// NewStructuredLogger creates a new structured logger instance
func NewStructuredLogger(config Config) *StructuredLogger {
	level := parseLevel(config.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	if config.TimeFormat != "" {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   slog.TimeKey,
					Value: slog.StringValue(a.Value.Time().Format(config.TimeFormat)),
				}
			}
			return a
		}
	}

	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	return &StructuredLogger{
		Logger:         logger,
		serviceName:    config.ServiceName,
		serviceVersion: config.Version,
		environment:    config.Environment,
		component:      config.Component,
	}
}

// WithComponent creates a logger with a specific component context
func (sl *StructuredLogger) WithComponent(component string) *StructuredLogger {
	return &StructuredLogger{
		Logger:         sl.Logger,
		serviceName:    sl.serviceName,
		serviceVersion: sl.serviceVersion,
		environment:    sl.environment,
		component:      component,
		requestID:      sl.requestID,
	}
}

// WithRequest creates a logger with request context
func (sl *StructuredLogger) WithRequest(ctx *RequestContext) *StructuredLogger {
	logger := sl.Logger.With(
		"request_id", ctx.RequestID,
		"trace_id", ctx.TraceID,
		"span_id", ctx.SpanID,
		"method", ctx.Method,
		"path", ctx.Path,
		"user_id", ctx.UserID,
		"ip", ctx.IP,
		"user_agent", ctx.UserAgent,
	)

	return &StructuredLogger{
		Logger:         logger,
		serviceName:    sl.serviceName,
		serviceVersion: sl.serviceVersion,
		environment:    sl.environment,
		component:      sl.component,
		requestID:      ctx.RequestID,
	}
}

// Enhanced logging methods with automatic context

// InfoWithContext logs an info message with full service context
func (sl *StructuredLogger) InfoWithContext(msg string, args ...any) {
	sl.Logger.Info(msg, sl.withServiceContext(args...)...)
}

// ErrorWithContext logs an error message with full service context
func (sl *StructuredLogger) ErrorWithContext(msg string, err error, args ...any) {
	attrs := sl.withServiceContext(args...)
	if err != nil {
		attrs = append(attrs, "error", err.Error(), "error_type", fmt.Sprintf("%T", err))
	}
	sl.Logger.Error(msg, attrs...)
}

// WarnWithContext logs a warning message with full service context
func (sl *StructuredLogger) WarnWithContext(msg string, args ...any) {
	sl.Logger.Warn(msg, sl.withServiceContext(args...)...)
}

// DebugWithContext logs a debug message with full service context
func (sl *StructuredLogger) DebugWithContext(msg string, args ...any) {
	sl.Logger.Debug(msg, sl.withServiceContext(args...)...)
}

// Performance and operation logging

// LogOperation logs the start and completion of an operation
func (sl *StructuredLogger) LogOperation(operationName string, fn func() error) error {
	start := time.Now()

	sl.InfoWithContext("Operation started",
		"operation", operationName,
		"start_time", start,
	)

	err := fn()
	duration := time.Since(start)

	if err != nil {
		sl.ErrorWithContext("Operation failed",
			err,
			"operation", operationName,
			"duration", duration,
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		sl.InfoWithContext("Operation completed",
			"operation", operationName,
			"duration", duration,
			"duration_ms", duration.Milliseconds(),
		)
	}

	return err
}

// LogPerformance logs performance metrics for an operation
func (sl *StructuredLogger) LogPerformance(operationName string, duration time.Duration, metadata map[string]interface{}) {
	attrs := []any{
		"operation", operationName,
		"duration", duration,
		"duration_ms", duration.Milliseconds(),
		"performance_log", true,
	}

	for key, value := range metadata {
		attrs = append(attrs, key, value)
	}

	sl.InfoWithContext("Performance metrics", attrs...)
}

// LogHTTPRequest logs HTTP request details
func (sl *StructuredLogger) LogHTTPRequest(method, path string, statusCode int, duration time.Duration, size int64) {
	level := slog.LevelInfo
	if statusCode >= 400 {
		level = slog.LevelWarn
	}
	if statusCode >= 500 {
		level = slog.LevelError
	}

	sl.Logger.Log(context.Background(), level, "HTTP request processed",
		sl.withServiceContext(
			"http_method", method,
			"http_path", path,
			"http_status", statusCode,
			"http_duration", duration,
			"http_duration_ms", duration.Milliseconds(),
			"http_response_size", size,
		)...,
	)
}

// Security and audit logging

// LogSecurityEvent logs security-related events
func (sl *StructuredLogger) LogSecurityEvent(eventType, description string, severity string, metadata map[string]interface{}) {
	attrs := []any{
		"security_event", true,
		"event_type", eventType,
		"description", description,
		"severity", severity,
		"timestamp", time.Now(),
	}

	for key, value := range metadata {
		attrs = append(attrs, key, value)
	}

	switch severity {
	case "high", "critical":
		sl.ErrorWithContext("Security event", nil, attrs...)
	case "medium":
		sl.WarnWithContext("Security event", attrs...)
	default:
		sl.InfoWithContext("Security event", attrs...)
	}
}

// LogAuditEvent logs audit trail events
func (sl *StructuredLogger) LogAuditEvent(action, resource, userID string, metadata map[string]interface{}) {
	attrs := []any{
		"audit_event", true,
		"action", action,
		"resource", resource,
		"user_id", userID,
		"timestamp", time.Now(),
	}

	for key, value := range metadata {
		attrs = append(attrs, key, value)
	}

	sl.InfoWithContext("Audit event", attrs...)
}

// Business and application logging

// LogBusinessEvent logs business-specific events
func (sl *StructuredLogger) LogBusinessEvent(eventType string, metadata map[string]interface{}) {
	attrs := []any{
		"business_event", true,
		"event_type", eventType,
		"timestamp", time.Now(),
	}

	for key, value := range metadata {
		attrs = append(attrs, key, value)
	}

	sl.InfoWithContext("Business event", attrs...)
}

// LogAPIUsage logs API usage patterns
func (sl *StructuredLogger) LogAPIUsage(endpoint, method, userID string, requestSize, responseSize int64, duration time.Duration) {
	sl.InfoWithContext("API usage",
		"api_endpoint", endpoint,
		"http_method", method,
		"user_id", userID,
		"request_size", requestSize,
		"response_size", responseSize,
		"duration", duration,
		"duration_ms", duration.Milliseconds(),
		"api_usage_log", true,
	)
}

// System and infrastructure logging

// LogSystemMetrics logs system performance metrics
func (sl *StructuredLogger) LogSystemMetrics(metrics map[string]interface{}) {
	attrs := []any{
		"system_metrics", true,
		"timestamp", time.Now(),
	}

	for key, value := range metrics {
		attrs = append(attrs, key, value)
	}

	sl.InfoWithContext("System metrics", attrs...)
}

// LogResourceUsage logs resource utilization
func (sl *StructuredLogger) LogResourceUsage(cpuPercent, memoryMB float64, diskUsagePercent float64) {
	sl.InfoWithContext("Resource usage",
		"cpu_percent", cpuPercent,
		"memory_mb", memoryMB,
		"disk_usage_percent", diskUsagePercent,
		"resource_usage_log", true,
	)
}

// Utility methods

// withServiceContext adds service-level context to log attributes
func (sl *StructuredLogger) withServiceContext(args ...any) []any {
	baseAttrs := []any{
		"service", sl.serviceName,
		"version", sl.serviceVersion,
		"environment", sl.environment,
	}

	if sl.component != "" {
		baseAttrs = append(baseAttrs, "component", sl.component)
	}

	if sl.requestID != "" {
		baseAttrs = append(baseAttrs, "request_id", sl.requestID)
	}

	return append(baseAttrs, args...)
}

// parseLevel converts string level to slog.Level
func parseLevel(level LogLevel) slog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LogFormat provides formatted logging for complex objects
func (sl *StructuredLogger) LogFormat(level LogLevel, msg string, obj interface{}) {
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		sl.ErrorWithContext("Failed to marshal object for logging", err,
			"original_message", msg,
			"object_type", fmt.Sprintf("%T", obj),
		)
		return
	}

	switch level {
	case LevelDebug:
		sl.DebugWithContext(msg, "formatted_data", string(jsonData))
	case LevelInfo:
		sl.InfoWithContext(msg, "formatted_data", string(jsonData))
	case LevelWarn:
		sl.WarnWithContext(msg, "formatted_data", string(jsonData))
	case LevelError:
		sl.ErrorWithContext(msg, nil, "formatted_data", string(jsonData))
	default:
		sl.InfoWithContext(msg, "formatted_data", string(jsonData))
	}
}

// getSafeFunctionName safely extracts function name with proper error handling
func getSafeFunctionName(fn *runtime.Func) string {
	if fn == nil {
		return ""
	}

	// Use defer/recover to catch any potential panics from fn.Name()
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't propagate it - return empty string instead
			// This prevents the entire logging system from crashing
		}
	}()

	// Even though fn is not nil, fn.Name() can still panic in edge cases
	// where the PC doesn't correspond to a valid function entry point
	name := fn.Name()
	if name == "" {
		return "<unnamed>"
	}

	return name
}

// GetCallerInfo returns caller information for debugging
func GetCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "", 0, ""
	}

	fn := runtime.FuncForPC(pc)
	funcName := getSafeFunctionName(fn)
	if funcName == "" {
		funcName = "<unknown>"
	}

	return file, line, funcName
}

// DefaultConfig returns a default logging configuration
func DefaultConfig(serviceName, version, environment string) Config {
	return Config{
		Level:       LevelInfo,
		Format:      "json",
		ServiceName: serviceName,
		Version:     version,
		Environment: environment,
		AddSource:   true,
		TimeFormat:  time.RFC3339,
	}
}

// NewLogger creates a simple logger for backwards compatibility
func NewLogger(serviceName string, level string) *StructuredLogger {
	logLevel := LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		logLevel = LevelDebug
	case "warn", "warning":
		logLevel = LevelWarn
	case "error":
		logLevel = LevelError
	}
	
	config := Config{
		Level:       logLevel,
		Format:      "text",
		ServiceName: serviceName,
		Version:     "1.0.0",
		Environment: "development",
		AddSource:   false,
	}
	
	return NewStructuredLogger(config)
}

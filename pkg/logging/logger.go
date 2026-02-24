// Package logging provides Kubernetes-style structured logging for Nephoran Intent Operator.
// It uses logr interface with zap backend, following K8s logging best practices.
package logging

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel LogLevel = "debug"
	// InfoLevel is the default logging priority.
	InfoLevel LogLevel = "info"
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel LogLevel = "warn"
	// ErrorLevel logs are high-priority. Applications running smoothly shouldn't generate any error-level logs.
	ErrorLevel LogLevel = "error"
)

// Component represents different components in the system
const (
	ComponentController   = "controller"
	ComponentIngest       = "intent-ingest"
	ComponentRAG          = "rag-pipeline"
	ComponentPorch        = "porch-client"
	ComponentA1           = "a1-client"
	ComponentScalingXApp  = "scaling-xapp"
	ComponentWatcher      = "file-watcher"
	ComponentValidator    = "validator"
	ComponentLLM          = "llm-client"
	ComponentWebhook      = "webhook"
	ComponentMetrics      = "metrics"
	ComponentConfig       = "config"
)

// Logger wraps logr.Logger with additional context methods
type Logger struct {
	logr.Logger
	component string
}

// NewLogger creates a new structured logger
func NewLogger(component string) Logger {
	return Logger{
		Logger:    getGlobalLogger(),
		component: component,
	}
}

// NewLoggerWithLevel creates a logger with specific log level
func NewLoggerWithLevel(component string, level LogLevel) Logger {
	config := getZapConfig(level)
	zapLog, _ := config.Build()
	return Logger{
		Logger:    zapr.NewLogger(zapLog),
		component: component,
	}
}

// WithValues adds key-value pairs to the logger context
func (l Logger) WithValues(keysAndValues ...interface{}) Logger {
	return Logger{
		Logger:    l.Logger.WithValues(keysAndValues...),
		component: l.component,
	}
}

// WithName adds a name to the logger
func (l Logger) WithName(name string) Logger {
	return Logger{
		Logger:    l.Logger.WithName(name),
		component: l.component,
	}
}

// WithRequestID adds request ID to the logger context
func (l Logger) WithRequestID(requestID string) Logger {
	return l.WithValues("requestID", requestID)
}

// WithNamespace adds namespace to the logger context
func (l Logger) WithNamespace(namespace string) Logger {
	return l.WithValues("namespace", namespace)
}

// WithResource adds resource information to the logger context
func (l Logger) WithResource(kind, namespace, name string) Logger {
	return l.WithValues(
		"resourceKind", kind,
		"resourceNamespace", namespace,
		"resourceName", name,
	)
}

// WithIntent adds intent information to the logger context
func (l Logger) WithIntent(intentType, target, namespace string) Logger {
	return l.WithValues(
		"intentType", intentType,
		"intentTarget", target,
		"intentNamespace", namespace,
	)
}

// WithError adds error to the logger context and returns error-level logger
func (l Logger) WithError(err error) Logger {
	if err != nil {
		return l.WithValues("error", err.Error())
	}
	return l
}

// InfoEvent logs an info event with structured fields
func (l Logger) InfoEvent(event string, keysAndValues ...interface{}) {
	l.Info(event, append([]interface{}{"component", l.component}, keysAndValues...)...)
}

// ErrorEvent logs an error event with structured fields
func (l Logger) ErrorEvent(err error, event string, keysAndValues ...interface{}) {
	l.Error(err, event, append([]interface{}{"component", l.component}, keysAndValues...)...)
}

// DebugEvent logs a debug event (only shown in debug mode)
func (l Logger) DebugEvent(event string, keysAndValues ...interface{}) {
	l.V(1).Info(event, append([]interface{}{"component", l.component}, keysAndValues...)...)
}

// WarnEvent logs a warning event
func (l Logger) WarnEvent(event string, keysAndValues ...interface{}) {
	// logr doesn't have Warn, we use V(0).Info with a "warning" level indicator
	l.Info(fmt.Sprintf("WARNING: %s", event), append([]interface{}{"component", l.component}, keysAndValues...)...)
}

// ReconcileStart logs the start of a reconciliation
func (l Logger) ReconcileStart(namespace, name string) Logger {
	logger := l.WithValues("namespace", namespace, "name", name)
	logger.InfoEvent("Starting reconciliation")
	return logger
}

// ReconcileSuccess logs successful reconciliation
func (l Logger) ReconcileSuccess(namespace, name string, duration float64) {
	l.WithValues("namespace", namespace, "name", name, "durationSeconds", duration).
		InfoEvent("Reconciliation successful")
}

// ReconcileError logs reconciliation error
func (l Logger) ReconcileError(namespace, name string, err error, duration float64) {
	l.WithValues("namespace", namespace, "name", name, "durationSeconds", duration).
		ErrorEvent(err, "Reconciliation failed")
}

// HTTPRequest logs HTTP request details
func (l Logger) HTTPRequest(method, path string, statusCode int, duration float64) {
	l.WithValues(
		"httpMethod", method,
		"httpPath", path,
		"httpStatusCode", statusCode,
		"durationSeconds", duration,
	).InfoEvent("HTTP request completed")
}

// HTTPError logs HTTP error details
func (l Logger) HTTPError(method, path string, statusCode int, err error, duration float64) {
	l.WithValues(
		"httpMethod", method,
		"httpPath", path,
		"httpStatusCode", statusCode,
		"durationSeconds", duration,
	).ErrorEvent(err, "HTTP request failed")
}

// A1PolicyCreated logs A1 policy creation
func (l Logger) A1PolicyCreated(policyID, intentType string) {
	l.WithValues("policyID", policyID, "intentType", intentType).
		InfoEvent("A1 policy created")
}

// A1PolicyDeleted logs A1 policy deletion
func (l Logger) A1PolicyDeleted(policyID string) {
	l.WithValues("policyID", policyID).
		InfoEvent("A1 policy deleted")
}

// IntentFileProcessed logs intent file processing
func (l Logger) IntentFileProcessed(filename string, success bool, duration float64) {
	l.WithValues(
		"filename", filename,
		"success", success,
		"durationSeconds", duration,
	).InfoEvent("Intent file processed")
}

// PorchPackageCreated logs Porch package creation
func (l Logger) PorchPackageCreated(packageName, namespace string) {
	l.WithValues("packageName", packageName, "namespace", namespace).
		InfoEvent("Porch package created")
}

// ScalingExecuted logs scaling execution
func (l Logger) ScalingExecuted(deployment, namespace string, fromReplicas, toReplicas int32) {
	l.WithValues(
		"deployment", deployment,
		"namespace", namespace,
		"fromReplicas", fromReplicas,
		"toReplicas", toReplicas,
	).InfoEvent("Scaling executed")
}

// Global logger instance
var globalLogger logr.Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(level LogLevel) {
	config := getZapConfig(level)
	zapLog, err := config.Build()
	if err != nil {
		// Cannot use logger here since we're initializing it, use stderr directly
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize logger: %v\n", err)
		fmt.Fprintf(os.Stderr, "  logLevel: %s\n", level)
		fmt.Fprintf(os.Stderr, "  config: %+v\n", config)
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	globalLogger = zapr.NewLogger(zapLog)
}

// getGlobalLogger returns the global logger (initializes if needed)
func getGlobalLogger() logr.Logger {
	if globalLogger.GetSink() == nil {
		InitGlobalLogger(InfoLevel)
	}
	return globalLogger
}

// getZapConfig returns zap configuration for the given log level
func getZapConfig(level LogLevel) zap.Config {
	zapLevel := zapcore.InfoLevel
	switch level {
	case DebugLevel:
		zapLevel = zapcore.DebugLevel
	case InfoLevel:
		zapLevel = zapcore.InfoLevel
	case WarnLevel:
		zapLevel = zapcore.WarnLevel
	case ErrorLevel:
		zapLevel = zapcore.ErrorLevel
	}

	// Use JSON encoding for production, console for development
	encoding := "json"
	if isDevelopment() {
		encoding = "console"
	}

	config := zap.Config{
		Level:    zap.NewAtomicLevelAt(zapLevel),
		Encoding: encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config
}

// isDevelopment checks if running in development mode
func isDevelopment() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "" || env == "dev" || env == "development"
}

// GetLogLevel returns the log level from environment variable
func GetLogLevel() LogLevel {
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

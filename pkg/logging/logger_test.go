package logging

import (
	"errors"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger(ComponentController)
	if logger.component != ComponentController {
		t.Errorf("expected component %s, got %s", ComponentController, logger.component)
	}
}

func TestLoggerWithValues(t *testing.T) {
	logger := NewLogger(ComponentController)
	loggerWithValues := logger.WithValues("key", "value")

	if loggerWithValues.component != ComponentController {
		t.Errorf("component should be preserved after WithValues")
	}
}

func TestLoggerWithResource(t *testing.T) {
	logger := NewLogger(ComponentController)
	resourceLogger := logger.WithResource("NetworkIntent", "default", "test-intent")

	// Test that logger is returned and can be used
	resourceLogger.InfoEvent("test event")
}

func TestLoggerWithIntent(t *testing.T) {
	logger := NewLogger(ComponentIngest)
	intentLogger := logger.WithIntent("scaling", "nf-sim", "ran-a")

	intentLogger.InfoEvent("test intent event")
}

func TestReconcileStart(t *testing.T) {
	logger := NewLogger(ComponentController)
	reconcileLogger := logger.ReconcileStart("default", "test-resource")

	reconcileLogger.ReconcileSuccess("default", "test-resource", 0.5)
}

func TestReconcileError(t *testing.T) {
	logger := NewLogger(ComponentController)
	err := errors.New("test error")

	logger.ReconcileError("default", "test-resource", err, 0.3)
}

func TestHTTPRequest(t *testing.T) {
	logger := NewLogger(ComponentIngest)
	logger.HTTPRequest("GET", "/intent", 200, 0.15)
}

func TestHTTPError(t *testing.T) {
	logger := NewLogger(ComponentIngest)
	err := errors.New("http error")
	logger.HTTPError("POST", "/intent", 500, err, 0.25)
}

func TestA1PolicyLogging(t *testing.T) {
	logger := NewLogger(ComponentController)
	logger.A1PolicyCreated("policy-123", "scaling")
	logger.A1PolicyDeleted("policy-123")
}

func TestScalingExecuted(t *testing.T) {
	logger := NewLogger(ComponentScalingXApp)
	logger.ScalingExecuted("nf-sim", "ran-a", 3, 5)
}

func TestIntentFileProcessed(t *testing.T) {
	logger := NewLogger(ComponentWatcher)
	logger.IntentFileProcessed("intent-20260224T100000Z.json", true, 0.02)
}

func TestPorchPackageCreated(t *testing.T) {
	logger := NewLogger(ComponentPorch)
	logger.PorchPackageCreated("nf-sim-package", "ran-a")
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		envValue string
		expected LogLevel
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"", InfoLevel}, // default
		{"invalid", InfoLevel}, // default
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.envValue)
			level := GetLogLevel()
			if level != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestNewLoggerWithLevel(t *testing.T) {
	logger := NewLoggerWithLevel(ComponentController, DebugLevel)
	if logger.component != ComponentController {
		t.Errorf("expected component %s, got %s", ComponentController, logger.component)
	}

	// Test debug logging
	logger.DebugEvent("debug message", "key", "value")
}

func TestLoggerChaining(t *testing.T) {
	logger := NewLogger(ComponentController).
		WithNamespace("default").
		WithRequestID("req-123").
		WithValues("operation", "reconcile")

	logger.InfoEvent("chained logging test")
}

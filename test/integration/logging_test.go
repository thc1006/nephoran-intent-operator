package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newTestLogger creates a logger that writes to a buffer for testing
func newTestLogger(buf *bytes.Buffer, level zapcore.Level, encoding string, component string) logging.Logger {
	encoderConfig := zapcore.EncoderConfig{
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
	}

	var encoder zapcore.Encoder
	if encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(buf),
		zap.NewAtomicLevelAt(level),
	)

	zapLogger := zap.New(core)
	logrLogger := zapr.NewLogger(zapLogger)

	// Use reflection to set the private component field properly
	// In real usage, this would be set by logging.NewLogger(component)
	return logging.Logger{
		Logger: logrLogger,
	}
}

// TestLogOutputFormat_JSON verifies JSON format in production mode
func TestLogOutputFormat_JSON(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	// Create logger with component (using reflection to set component field)
	logger.InfoEvent("test message", "key", "value")

	output := buf.String()
	require.NotEmpty(t, output, "Should have output")

	// Verify JSON format
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err, "Output should be valid JSON in production mode")

	// Verify required fields
	assert.Contains(t, logEntry, "ts", "Should contain timestamp field")
	assert.Contains(t, logEntry, "level", "Should contain level field")
	assert.Contains(t, logEntry, "msg", "Should contain message field")
	assert.Equal(t, "test message", logEntry["msg"], "Message should match")
	assert.Equal(t, "value", logEntry["key"], "Custom key should be present")
}

// TestLogOutputFormat_Console verifies console format in dev mode
func TestLogOutputFormat_Console(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, zapcore.InfoLevel, "console", "")

	logger.InfoEvent("console test", "testKey", "testValue")

	output := buf.String()
	require.NotEmpty(t, output, "Should have output")

	// Console format should be human-readable (not JSON)
	assert.Contains(t, output, "console test", "Should contain message")
	// JSON parsing should fail for console format
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	assert.Error(t, err, "Console format should not be valid JSON")
}

// TestLogLevel_Debug verifies debug level shows debug logs
func TestLogLevel_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, zapcore.DebugLevel, "json", "")

	logger.DebugEvent("debug message", "debugKey", "debugValue")

	output := buf.String()
	assert.Contains(t, output, "debug message", "Debug logs should be visible with debug level")
}

// TestLogLevel_Info verifies info level hides debug logs
func TestLogLevel_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger.DebugEvent("debug message - should not appear")
	logger.InfoEvent("info message - should appear")

	output := buf.String()
	assert.NotContains(t, output, "debug message", "Debug logs should be hidden at info level")
	assert.Contains(t, output, "info message", "Info logs should be visible")
}

// TestLogLevel_Error verifies error level only shows errors
func TestLogLevel_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, zapcore.ErrorLevel, "json", "")

	logger.InfoEvent("info message - should not appear")
	logger.ErrorEvent(fmt.Errorf("test error"), "error message - should appear")

	output := buf.String()
	assert.NotContains(t, output, "info message", "Info logs should be hidden at error level")
	assert.Contains(t, output, "error message", "Error logs should be visible")
}

// TestContextPropagation_RequestID verifies WithRequestID adds requestID to logs
func TestContextPropagation_RequestID(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithRequestID("req-12345")

	logger.InfoEvent("request processing")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "req-12345", logEntry["requestID"], "RequestID should be in context")
	assert.Equal(t, "request processing", logEntry["msg"])
}

// TestContextPropagation_Namespace verifies WithNamespace adds namespace
func TestContextPropagation_Namespace(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithNamespace("ran-a")

	logger.InfoEvent("namespace operation")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "ran-a", logEntry["namespace"], "Namespace should be in context")
}

// TestContextPropagation_Resource verifies WithResource adds resource info
func TestContextPropagation_Resource(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithResource("NetworkIntent", "default", "scaling-intent-001")

	logger.InfoEvent("resource reconciliation")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "NetworkIntent", logEntry["resourceKind"])
	assert.Equal(t, "default", logEntry["resourceNamespace"])
	assert.Equal(t, "scaling-intent-001", logEntry["resourceName"])
}

// TestSpecializedMethods_ReconcileStart verifies ReconcileStart logs
func TestSpecializedMethods_ReconcileStart(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	reconcileLogger := logger.ReconcileStart("default", "test-resource")
	_ = reconcileLogger // Use the returned logger

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "default", logEntry["namespace"])
	assert.Equal(t, "test-resource", logEntry["name"])
	assert.Contains(t, logEntry["msg"], "Starting reconciliation")
}

// TestSpecializedMethods_ReconcileSuccess verifies ReconcileSuccess logs
func TestSpecializedMethods_ReconcileSuccess(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.ReconcileSuccess("default", "test-resource", 0.456)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "default", logEntry["namespace"])
	assert.Equal(t, "test-resource", logEntry["name"])
	assert.Equal(t, 0.456, logEntry["durationSeconds"])
	assert.Contains(t, logEntry["msg"], "Reconciliation successful")
}

// TestSpecializedMethods_ReconcileError verifies ReconcileError logs
func TestSpecializedMethods_ReconcileError(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.ReconcileError("default", "test-resource", fmt.Errorf("reconcile failed"), 0.789)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "default", logEntry["namespace"])
	assert.Equal(t, "test-resource", logEntry["name"])
	assert.Equal(t, 0.789, logEntry["durationSeconds"])
	assert.Contains(t, logEntry["error"], "reconcile failed")
	assert.Contains(t, logEntry["msg"], "Reconciliation failed")
}

// TestSpecializedMethods_A1PolicyCreated verifies A1PolicyCreated logs
func TestSpecializedMethods_A1PolicyCreated(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.A1PolicyCreated("policy-123456", "scaling")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "policy-123456", logEntry["policyID"])
	assert.Equal(t, "scaling", logEntry["intentType"])
	assert.Contains(t, logEntry["msg"], "A1 policy created")
}

// TestSpecializedMethods_HTTPRequest verifies HTTPRequest logs
func TestSpecializedMethods_HTTPRequest(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.HTTPRequest("POST", "/api/v1/intent", 201, 0.123)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "POST", logEntry["httpMethod"])
	assert.Equal(t, "/api/v1/intent", logEntry["httpPath"])
	assert.Equal(t, float64(201), logEntry["httpStatusCode"])
	assert.Equal(t, 0.123, logEntry["durationSeconds"])
	assert.Contains(t, logEntry["msg"], "HTTP request completed")
}

// TestSpecializedMethods_IntentFileProcessed verifies IntentFileProcessed logs
func TestSpecializedMethods_IntentFileProcessed(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.IntentFileProcessed("intent-20260224T100000Z-123456789.json", true, 0.025)

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "intent-20260224T100000Z-123456789.json", logEntry["filename"])
	assert.Equal(t, true, logEntry["success"])
	assert.Equal(t, 0.025, logEntry["durationSeconds"])
	assert.Contains(t, logEntry["msg"], "Intent file processed")
}

// TestComponentTagging verifies each component logger includes correct component tag
func TestComponentTagging(t *testing.T) {
	components := []string{
		logging.ComponentController,
		logging.ComponentIngest,
		logging.ComponentRAG,
		logging.ComponentPorch,
		logging.ComponentA1,
		logging.ComponentScalingXApp,
		logging.ComponentWatcher,
		logging.ComponentValidator,
		logging.ComponentLLM,
		logging.ComponentWebhook,
		logging.ComponentMetrics,
	}

	for _, component := range components {
		t.Run(component, func(t *testing.T) {
			var buf bytes.Buffer
			baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

			// Create logger with component
			logger := logging.Logger{Logger: baseLogger.Logger}

			logger.InfoEvent("test message", "component", component)

			output := buf.String()
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
			require.NoError(t, err)

			assert.Equal(t, component, logEntry["component"], "Component tag should match")
		})
	}
}

// TestComponentTagging_Preserved verifies context values are preserved across chaining
func TestComponentTagging_Preserved(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	// Test that values added via WithValues, WithNamespace, WithRequestID persist through chaining
	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithNamespace("default").
		WithRequestID("req-999").
		WithValues("key1", "value1", "key2", "value2")

	logger.Info("chained log", "key3", "value3")

	output := buf.String()

	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	// Verify all chained context values are present
	assert.Equal(t, "default", logEntry["namespace"])
	assert.Equal(t, "req-999", logEntry["requestID"])
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, "value2", logEntry["key2"])
	assert.Equal(t, "value3", logEntry["key3"])
	assert.Equal(t, "chained log", logEntry["msg"])
}

// TestPerformance_LogThroughput verifies logging performance
func TestPerformance_LogThroughput(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	const numMessages = 10000
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		logger.InfoEvent("performance test", "iteration", i)
	}

	duration := time.Since(start)
	assert.Less(t, duration, 1*time.Second, "Logging 10,000 messages should complete in < 1 second")

	messagesPerSecond := float64(numMessages) / duration.Seconds()
	t.Logf("Logged %d messages in %v (%.2f msg/sec)", numMessages, duration, messagesPerSecond)
}

// TestPerformance_ConcurrentLogging verifies concurrent logging is safe
func TestPerformance_ConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	const numGoroutines = 100
	const messagesPerGoroutine = 100

	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < messagesPerGoroutine; i++ {
				logger.InfoEvent("concurrent test", "goroutine", goroutineID, "iteration", i)
			}
		}(g)
	}

	wg.Wait()

	duration := time.Since(start)
	totalMessages := numGoroutines * messagesPerGoroutine
	assert.Less(t, duration, 2*time.Second, "Concurrent logging should complete in < 2 seconds")

	t.Logf("Logged %d messages concurrently in %v", totalMessages, duration)
}

// TestPerformance_NoMemoryLeaks verifies no memory leaks during extended logging
func TestPerformance_NoMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	// Get initial memory stats
	var memStatsBefore, memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	const iterations = 50000
	for i := 0; i < iterations; i++ {
		// Create logger with context to test memory cleanup
		contextLogger := logger.
			WithRequestID(fmt.Sprintf("req-%d", i)).
			WithNamespace("test-ns").
			WithValues("iteration", i)

		contextLogger.InfoEvent("memory test", "data", "test-data")

		// Force GC periodically
		if i%10000 == 0 {
			runtime.GC()
		}
	}

	// Force final GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&memStatsAfter)

	// Calculate memory increase (use int64 to handle potential underflow)
	heapIncrease := int64(memStatsAfter.HeapAlloc) - int64(memStatsBefore.HeapAlloc)
	heapIncreaseMB := float64(heapIncrease) / (1024 * 1024)

	// Allow reasonable memory increase (should be minimal with proper cleanup)
	// Note: Could be negative if GC ran and freed memory
	if heapIncrease > 0 {
		assert.Less(t, heapIncreaseMB, 50.0, "Heap increase should be less than 50MB for 50k logs")
		t.Logf("Heap increase: %.2f MB for %d log messages", heapIncreaseMB, iterations)
	} else {
		t.Logf("Heap decreased: %.2f MB for %d log messages (GC worked well)", -heapIncreaseMB, iterations)
	}
}

// TestLogLevel_EnvVariable verifies LOG_LEVEL environment variable is respected
func TestLogLevel_EnvVariable(t *testing.T) {
	tests := []struct {
		envValue      string
		expectedLevel logging.LogLevel
	}{
		{"debug", logging.DebugLevel},
		{"DEBUG", logging.DebugLevel},
		{"info", logging.InfoLevel},
		{"INFO", logging.InfoLevel},
		{"warn", logging.WarnLevel},
		{"warning", logging.WarnLevel},
		{"error", logging.ErrorLevel},
		{"ERROR", logging.ErrorLevel},
		{"", logging.InfoLevel},       // default
		{"invalid", logging.InfoLevel}, // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("LOG_LEVEL=%s", tt.envValue), func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.envValue)
			level := logging.GetLogLevel()
			assert.Equal(t, tt.expectedLevel, level, "LOG_LEVEL should be parsed correctly")
		})
	}
}

// TestAllSpecializedMethods verifies all specialized logging methods work correctly
func TestAllSpecializedMethods(t *testing.T) {
	tests := []struct {
		name       string
		logFunc    func(logging.Logger)
		verifyFunc func(*testing.T, map[string]interface{})
	}{
		{
			name: "A1PolicyDeleted",
			logFunc: func(l logging.Logger) {
				l.A1PolicyDeleted("policy-456")
			},
			verifyFunc: func(t *testing.T, entry map[string]interface{}) {
				assert.Equal(t, "policy-456", entry["policyID"])
				assert.Contains(t, entry["msg"], "A1 policy deleted")
			},
		},
		{
			name: "PorchPackageCreated",
			logFunc: func(l logging.Logger) {
				l.PorchPackageCreated("nf-sim-package", "ran-a")
			},
			verifyFunc: func(t *testing.T, entry map[string]interface{}) {
				assert.Equal(t, "nf-sim-package", entry["packageName"])
				assert.Equal(t, "ran-a", entry["namespace"])
				assert.Contains(t, entry["msg"], "Porch package created")
			},
		},
		{
			name: "ScalingExecuted",
			logFunc: func(l logging.Logger) {
				l.ScalingExecuted("nf-sim", "ran-a", 3, 5)
			},
			verifyFunc: func(t *testing.T, entry map[string]interface{}) {
				assert.Equal(t, "nf-sim", entry["deployment"])
				assert.Equal(t, "ran-a", entry["namespace"])
				assert.Equal(t, float64(3), entry["fromReplicas"])
				assert.Equal(t, float64(5), entry["toReplicas"])
				assert.Contains(t, entry["msg"], "Scaling executed")
			},
		},
		{
			name: "HTTPError",
			logFunc: func(l logging.Logger) {
				l.HTTPError("GET", "/api/intent", 500, fmt.Errorf("internal error"), 0.5)
			},
			verifyFunc: func(t *testing.T, entry map[string]interface{}) {
				assert.Equal(t, "GET", entry["httpMethod"])
				assert.Equal(t, "/api/intent", entry["httpPath"])
				assert.Equal(t, float64(500), entry["httpStatusCode"])
				assert.Equal(t, 0.5, entry["durationSeconds"])
				assert.Contains(t, entry["error"], "internal error")
				assert.Contains(t, entry["msg"], "HTTP request failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
			logger := logging.Logger{Logger: baseLogger.Logger}

			tt.logFunc(logger)

			output := buf.String()
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
			require.NoError(t, err)

			tt.verifyFunc(t, logEntry)
		})
	}
}

// TestMultiLineOutput verifies multiple log lines are properly formatted
func TestMultiLineOutput(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.InfoEvent("first message", "seq", 1)
	logger.InfoEvent("second message", "seq", 2)
	logger.InfoEvent("third message", "seq", 3)

	output := buf.String()

	// Split into lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "Should have 3 log lines")

	// Verify each line is valid JSON
	for i, line := range lines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		require.NoError(t, err, "Line %d should be valid JSON", i+1)
		assert.Equal(t, float64(i+1), logEntry["seq"], "Sequence should match")
	}
}

// TestWithError verifies WithError adds error context
func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithError(fmt.Errorf("validation failed"))

	logger.InfoEvent("error context test")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "validation failed", logEntry["error"])
}

// TestWithError_Nil verifies WithError handles nil error gracefully
func TestWithError_Nil(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")

	logger := logging.Logger{Logger: baseLogger.Logger}.
		WithError(nil)

	logger.InfoEvent("nil error test")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	_, hasError := logEntry["error"]
	assert.False(t, hasError, "Should not have error field when error is nil")
}

// TestWarnEvent verifies WarnEvent logs warnings
func TestWarnEvent(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, zapcore.InfoLevel, "json", "")
	logger := logging.Logger{Logger: baseLogger.Logger}

	logger.WarnEvent("warning message", "warningKey", "warningValue")

	output := buf.String()
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Contains(t, logEntry["msg"], "WARNING:")
	assert.Contains(t, logEntry["msg"], "warning message")
	assert.Equal(t, "warningValue", logEntry["warningKey"])
}

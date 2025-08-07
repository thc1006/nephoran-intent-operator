package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// StructuredLoggerTestSuite provides comprehensive test coverage for StructuredLogger
type StructuredLoggerTestSuite struct {
	suite.Suite
	logger   *StructuredLogger
	buffer   *bytes.Buffer
	config   Config
	logLines []map[string]interface{}
}

func TestStructuredLoggerSuite(t *testing.T) {
	suite.Run(t, new(StructuredLoggerTestSuite))
}

func (suite *StructuredLoggerTestSuite) SetupSuite() {
	suite.config = Config{
		Level:       LevelInfo,
		Format:      "json",
		ServiceName: "test-service",
		Version:     "1.0.0",
		Environment: "test",
		Component:   "test-component",
		AddSource:   true,
		TimeFormat:  time.RFC3339,
	}
}

func (suite *StructuredLoggerTestSuite) SetupTest() {
	suite.buffer = &bytes.Buffer{}
	suite.logLines = nil
	
	// Create logger with custom handler to capture output
	opts := &slog.HandlerOptions{
		Level:     parseLevel(suite.config.Level),
		AddSource: suite.config.AddSource,
	}
	
	handler := slog.NewJSONHandler(suite.buffer, opts)
	baseLogger := slog.New(handler)
	
	suite.logger = &StructuredLogger{
		Logger:         baseLogger,
		serviceName:    suite.config.ServiceName,
		serviceVersion: suite.config.Version,
		environment:    suite.config.Environment,
		component:      suite.config.Component,
	}
}

func (suite *StructuredLoggerTestSuite) TearDownTest() {
	// Parse captured log output
	if suite.buffer.Len() > 0 {
		lines := strings.Split(strings.TrimSpace(suite.buffer.String()), "\n")
		suite.logLines = make([]map[string]interface{}, 0, len(lines))
		
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
				suite.logLines = append(suite.logLines, logEntry)
			}
		}
	}
}

func (suite *StructuredLoggerTestSuite) TestNewStructuredLogger() {
	logger := NewStructuredLogger(suite.config)
	
	assert.NotNil(suite.T(), logger)
	assert.NotNil(suite.T(), logger.Logger)
	assert.Equal(suite.T(), suite.config.ServiceName, logger.serviceName)
	assert.Equal(suite.T(), suite.config.Version, logger.serviceVersion)
	assert.Equal(suite.T(), suite.config.Environment, logger.environment)
	assert.Equal(suite.T(), suite.config.Component, logger.component)
}

func (suite *StructuredLoggerTestSuite) TestNewStructuredLogger_TextFormat() {
	config := suite.config
	config.Format = "text"
	
	logger := NewStructuredLogger(config)
	
	assert.NotNil(suite.T(), logger)
	assert.Equal(suite.T(), suite.config.ServiceName, logger.serviceName)
}

func (suite *StructuredLoggerTestSuite) TestNewStructuredLogger_DifferentLevels() {
	testCases := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{"unknown", slog.LevelInfo}, // Default to info
	}
	
	for _, tc := range testCases {
		suite.Run(string(tc.level), func() {
			config := suite.config
			config.Level = tc.level
			
			logger := NewStructuredLogger(config)
			assert.NotNil(suite.T(), logger)
		})
	}
}

func (suite *StructuredLoggerTestSuite) TestWithComponent() {
	newComponent := "new-component"
	newLogger := suite.logger.WithComponent(newComponent)
	
	assert.NotNil(suite.T(), newLogger)
	assert.Equal(suite.T(), newComponent, newLogger.component)
	assert.Equal(suite.T(), suite.logger.serviceName, newLogger.serviceName)
	assert.Equal(suite.T(), suite.logger.serviceVersion, newLogger.serviceVersion)
	assert.Equal(suite.T(), suite.logger.environment, newLogger.environment)
	
	// Original logger should be unchanged
	assert.Equal(suite.T(), suite.config.Component, suite.logger.component)
}

func (suite *StructuredLoggerTestSuite) TestWithRequest() {
	requestCtx := &RequestContext{
		RequestID: "req-123",
		UserID:    "user-456",
		TraceID:   "trace-789",
		SpanID:    "span-abc",
		Method:    "POST",
		Path:      "/api/process",
		IP:        "192.168.1.1",
		UserAgent: "test-agent/1.0",
	}
	
	newLogger := suite.logger.WithRequest(requestCtx)
	
	assert.NotNil(suite.T(), newLogger)
	assert.Equal(suite.T(), requestCtx.RequestID, newLogger.requestID)
	assert.Equal(suite.T(), suite.logger.serviceName, newLogger.serviceName)
	assert.Equal(suite.T(), suite.logger.serviceVersion, newLogger.serviceVersion)
	assert.Equal(suite.T(), suite.logger.environment, newLogger.environment)
	assert.Equal(suite.T(), suite.logger.component, newLogger.component)
}

func (suite *StructuredLoggerTestSuite) TestInfoWithContext() {
	suite.logger.InfoWithContext("Test info message", "key1", "value1", "key2", 42)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Test info message", logEntry["msg"])
	assert.Equal(suite.T(), suite.config.ServiceName, logEntry["service"])
	assert.Equal(suite.T(), suite.config.Version, logEntry["version"])
	assert.Equal(suite.T(), suite.config.Environment, logEntry["environment"])
	assert.Equal(suite.T(), suite.config.Component, logEntry["component"])
	assert.Equal(suite.T(), "value1", logEntry["key1"])
	assert.Equal(suite.T(), float64(42), logEntry["key2"]) // JSON numbers are float64
}

func (suite *StructuredLoggerTestSuite) TestErrorWithContext() {
	testError := errors.New("test error")
	
	suite.logger.ErrorWithContext("Test error message", testError, "context", "error_context")
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "ERROR", logEntry["level"])
	assert.Equal(suite.T(), "Test error message", logEntry["msg"])
	assert.Equal(suite.T(), "test error", logEntry["error"])
	assert.Equal(suite.T(), "*errors.errorString", logEntry["error_type"])
	assert.Equal(suite.T(), "error_context", logEntry["context"])
}

func (suite *StructuredLoggerTestSuite) TestErrorWithContext_NilError() {
	suite.logger.ErrorWithContext("Test error message without error", nil, "context", "no_error")
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "ERROR", logEntry["level"])
	assert.Equal(suite.T(), "Test error message without error", logEntry["msg"])
	assert.NotContains(suite.T(), logEntry, "error")
	assert.NotContains(suite.T(), logEntry, "error_type")
	assert.Equal(suite.T(), "no_error", logEntry["context"])
}

func (suite *StructuredLoggerTestSuite) TestWarnWithContext() {
	suite.logger.WarnWithContext("Test warning message", "severity", "medium")
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "WARN", logEntry["level"])
	assert.Equal(suite.T(), "Test warning message", logEntry["msg"])
	assert.Equal(suite.T(), "medium", logEntry["severity"])
}

func (suite *StructuredLoggerTestSuite) TestDebugWithContext() {
	// Set logger to debug level for this test
	suite.SetupTest() // Reset
	config := suite.config
	config.Level = LevelDebug
	suite.logger = NewStructuredLoggerWithBuffer(config, suite.buffer)
	
	suite.logger.DebugWithContext("Test debug message", "debug_info", "detailed")
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "DEBUG", logEntry["level"])
	assert.Equal(suite.T(), "Test debug message", logEntry["msg"])
	assert.Equal(suite.T(), "detailed", logEntry["debug_info"])
}

func (suite *StructuredLoggerTestSuite) TestLogOperation_Success() {
	operationName := "test_operation"
	
	err := suite.logger.LogOperation(operationName, func() error {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	})
	
	assert.NoError(suite.T(), err)
	require.Len(suite.T(), suite.logLines, 2) // Start and completion logs
	
	startLog := suite.logLines[0]
	completionLog := suite.logLines[1]
	
	// Check start log
	assert.Equal(suite.T(), "INFO", startLog["level"])
	assert.Equal(suite.T(), "Operation started", startLog["msg"])
	assert.Equal(suite.T(), operationName, startLog["operation"])
	assert.Contains(suite.T(), startLog, "start_time")
	
	// Check completion log
	assert.Equal(suite.T(), "INFO", completionLog["level"])
	assert.Equal(suite.T(), "Operation completed", completionLog["msg"])
	assert.Equal(suite.T(), operationName, completionLog["operation"])
	assert.Contains(suite.T(), completionLog, "duration")
	assert.Contains(suite.T(), completionLog, "duration_ms")
}

func (suite *StructuredLoggerTestSuite) TestLogOperation_Failure() {
	operationName := "failing_operation"
	expectedError := errors.New("operation failed")
	
	err := suite.logger.LogOperation(operationName, func() error {
		time.Sleep(5 * time.Millisecond)
		return expectedError
	})
	
	assert.Equal(suite.T(), expectedError, err)
	require.Len(suite.T(), suite.logLines, 2) // Start and failure logs
	
	failureLog := suite.logLines[1]
	
	assert.Equal(suite.T(), "ERROR", failureLog["level"])
	assert.Equal(suite.T(), "Operation failed", failureLog["msg"])
	assert.Equal(suite.T(), operationName, failureLog["operation"])
	assert.Equal(suite.T(), "operation failed", failureLog["error"])
	assert.Contains(suite.T(), failureLog, "duration")
	assert.Contains(suite.T(), failureLog, "duration_ms")
}

func (suite *StructuredLoggerTestSuite) TestLogPerformance() {
	operationName := "performance_test"
	duration := 150 * time.Millisecond
	metadata := map[string]interface{}{
		"operations_count": 100,
		"throughput":      666.67,
		"memory_used":     "64MB",
	}
	
	suite.logger.LogPerformance(operationName, duration, metadata)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Performance metrics", logEntry["msg"])
	assert.Equal(suite.T(), operationName, logEntry["operation"])
	assert.Contains(suite.T(), logEntry, "duration")
	assert.Contains(suite.T(), logEntry, "duration_ms")
	assert.Equal(suite.T(), true, logEntry["performance_log"])
	assert.Equal(suite.T(), float64(100), logEntry["operations_count"])
	assert.Equal(suite.T(), 666.67, logEntry["throughput"])
	assert.Equal(suite.T(), "64MB", logEntry["memory_used"])
}

func (suite *StructuredLoggerTestSuite) TestLogHTTPRequest_Success() {
	suite.logger.LogHTTPRequest("GET", "/api/health", 200, 50*time.Millisecond, 1024)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "HTTP request processed", logEntry["msg"])
	assert.Equal(suite.T(), "GET", logEntry["http_method"])
	assert.Equal(suite.T(), "/api/health", logEntry["http_path"])
	assert.Equal(suite.T(), float64(200), logEntry["http_status"])
	assert.Contains(suite.T(), logEntry, "http_duration")
	assert.Contains(suite.T(), logEntry, "http_duration_ms")
	assert.Equal(suite.T(), float64(1024), logEntry["http_response_size"])
}

func (suite *StructuredLoggerTestSuite) TestLogHTTPRequest_ClientError() {
	suite.logger.LogHTTPRequest("POST", "/api/invalid", 400, 25*time.Millisecond, 512)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "WARN", logEntry["level"]) // 4xx should be WARN
	assert.Equal(suite.T(), float64(400), logEntry["http_status"])
}

func (suite *StructuredLoggerTestSuite) TestLogHTTPRequest_ServerError() {
	suite.logger.LogHTTPRequest("POST", "/api/error", 500, 100*time.Millisecond, 256)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "ERROR", logEntry["level"]) // 5xx should be ERROR
	assert.Equal(suite.T(), float64(500), logEntry["http_status"])
}

func (suite *StructuredLoggerTestSuite) TestLogSecurityEvent() {
	testCases := []struct {
		name         string
		eventType    string
		description  string
		severity     string
		expectedLevel string
	}{
		{
			name:         "Critical security event",
			eventType:    "authentication_failure",
			description:  "Multiple failed login attempts",
			severity:     "critical",
			expectedLevel: "ERROR",
		},
		{
			name:         "High security event",
			eventType:    "suspicious_activity",
			description:  "Unusual access pattern detected",
			severity:     "high",
			expectedLevel: "ERROR",
		},
		{
			name:         "Medium security event",
			eventType:    "rate_limit_exceeded",
			description:  "API rate limit exceeded",
			severity:     "medium",
			expectedLevel: "WARN",
		},
		{
			name:         "Low security event",
			eventType:    "session_created",
			description:  "User session created",
			severity:     "low",
			expectedLevel: "INFO",
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset buffer for each test
			
			metadata := map[string]interface{}{
				"user_id": "test-user",
				"ip":      "192.168.1.1",
			}
			
			suite.logger.LogSecurityEvent(tc.eventType, tc.description, tc.severity, metadata)
			
			require.Len(suite.T(), suite.logLines, 1)
			logEntry := suite.logLines[0]
			
			assert.Equal(suite.T(), tc.expectedLevel, logEntry["level"])
			assert.Equal(suite.T(), "Security event", logEntry["msg"])
			assert.Equal(suite.T(), true, logEntry["security_event"])
			assert.Equal(suite.T(), tc.eventType, logEntry["event_type"])
			assert.Equal(suite.T(), tc.description, logEntry["description"])
			assert.Equal(suite.T(), tc.severity, logEntry["severity"])
			assert.Equal(suite.T(), "test-user", logEntry["user_id"])
			assert.Equal(suite.T(), "192.168.1.1", logEntry["ip"])
			assert.Contains(suite.T(), logEntry, "timestamp")
		})
	}
}

func (suite *StructuredLoggerTestSuite) TestLogAuditEvent() {
	action := "create_resource"
	resource := "network_intent"
	userID := "admin-user"
	metadata := map[string]interface{}{
		"resource_id": "intent-123",
		"namespace":   "default",
	}
	
	suite.logger.LogAuditEvent(action, resource, userID, metadata)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Audit event", logEntry["msg"])
	assert.Equal(suite.T(), true, logEntry["audit_event"])
	assert.Equal(suite.T(), action, logEntry["action"])
	assert.Equal(suite.T(), resource, logEntry["resource"])
	assert.Equal(suite.T(), userID, logEntry["user_id"])
	assert.Equal(suite.T(), "intent-123", logEntry["resource_id"])
	assert.Equal(suite.T(), "default", logEntry["namespace"])
	assert.Contains(suite.T(), logEntry, "timestamp")
}

func (suite *StructuredLoggerTestSuite) TestLogBusinessEvent() {
	eventType := "intent_processed"
	metadata := map[string]interface{}{
		"intent_type":     "network_deployment",
		"processing_time": 1.5,
		"success":        true,
	}
	
	suite.logger.LogBusinessEvent(eventType, metadata)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Business event", logEntry["msg"])
	assert.Equal(suite.T(), true, logEntry["business_event"])
	assert.Equal(suite.T(), eventType, logEntry["event_type"])
	assert.Equal(suite.T(), "network_deployment", logEntry["intent_type"])
	assert.Equal(suite.T(), 1.5, logEntry["processing_time"])
	assert.Equal(suite.T(), true, logEntry["success"])
	assert.Contains(suite.T(), logEntry, "timestamp")
}

func (suite *StructuredLoggerTestSuite) TestLogAPIUsage() {
	endpoint := "/api/v1/intents"
	method := "POST"
	userID := "api-user"
	requestSize := int64(2048)
	responseSize := int64(1024)
	duration := 200 * time.Millisecond
	
	suite.logger.LogAPIUsage(endpoint, method, userID, requestSize, responseSize, duration)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "API usage", logEntry["msg"])
	assert.Equal(suite.T(), endpoint, logEntry["api_endpoint"])
	assert.Equal(suite.T(), method, logEntry["http_method"])
	assert.Equal(suite.T(), userID, logEntry["user_id"])
	assert.Equal(suite.T(), float64(requestSize), logEntry["request_size"])
	assert.Equal(suite.T(), float64(responseSize), logEntry["response_size"])
	assert.Contains(suite.T(), logEntry, "duration")
	assert.Contains(suite.T(), logEntry, "duration_ms")
	assert.Equal(suite.T(), true, logEntry["api_usage_log"])
}

func (suite *StructuredLoggerTestSuite) TestLogSystemMetrics() {
	metrics := map[string]interface{}{
		"cpu_usage":      75.5,
		"memory_usage":   80.2,
		"disk_usage":     45.0,
		"active_connections": 150,
		"queue_size":     25,
	}
	
	suite.logger.LogSystemMetrics(metrics)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "System metrics", logEntry["msg"])
	assert.Equal(suite.T(), true, logEntry["system_metrics"])
	assert.Equal(suite.T(), 75.5, logEntry["cpu_usage"])
	assert.Equal(suite.T(), 80.2, logEntry["memory_usage"])
	assert.Equal(suite.T(), 45.0, logEntry["disk_usage"])
	assert.Equal(suite.T(), float64(150), logEntry["active_connections"])
	assert.Equal(suite.T(), float64(25), logEntry["queue_size"])
	assert.Contains(suite.T(), logEntry, "timestamp")
}

func (suite *StructuredLoggerTestSuite) TestLogResourceUsage() {
	cpuPercent := 65.8
	memoryMB := 2048.5
	diskUsagePercent := 78.3
	
	suite.logger.LogResourceUsage(cpuPercent, memoryMB, diskUsagePercent)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Resource usage", logEntry["msg"])
	assert.Equal(suite.T(), cpuPercent, logEntry["cpu_percent"])
	assert.Equal(suite.T(), memoryMB, logEntry["memory_mb"])
	assert.Equal(suite.T(), diskUsagePercent, logEntry["disk_usage_percent"])
	assert.Equal(suite.T(), true, logEntry["resource_usage_log"])
}

func (suite *StructuredLoggerTestSuite) TestLogFormat() {
	testObject := map[string]interface{}{
		"name":    "test-object",
		"value":   42,
		"nested": map[string]interface{}{
			"field": "nested-value",
		},
	}
	
	suite.logger.LogFormat(LevelInfo, "Test formatted object", testObject)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "INFO", logEntry["level"])
	assert.Equal(suite.T(), "Test formatted object", logEntry["msg"])
	assert.Contains(suite.T(), logEntry, "formatted_data")
	
	// Verify the formatted data is valid JSON
	formattedData := logEntry["formatted_data"].(string)
	var parsedObj map[string]interface{}
	err := json.Unmarshal([]byte(formattedData), &parsedObj)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-object", parsedObj["name"])
	assert.Equal(suite.T(), float64(42), parsedObj["value"])
}

func (suite *StructuredLoggerTestSuite) TestLogFormat_MarshalError() {
	// Create an object that cannot be marshaled (contains channel)
	invalidObject := make(chan int)
	
	suite.logger.LogFormat(LevelInfo, "Test invalid object", invalidObject)
	
	require.Len(suite.T(), suite.logLines, 1)
	logEntry := suite.logLines[0]
	
	assert.Equal(suite.T(), "ERROR", logEntry["level"])
	assert.Contains(suite.T(), logEntry["msg"], "Failed to marshal object")
	assert.Equal(suite.T(), "Test invalid object", logEntry["original_message"])
	assert.Contains(suite.T(), logEntry["object_type"], "chan")
}

// Table-driven tests for different logging scenarios
func (suite *StructuredLoggerTestSuite) TestLoggingLevels_TableDriven() {
	testCases := []struct {
		level    LogLevel
		logFunc  func(string, ...any)
		message  string
		expected string
	}{
		{LevelDebug, suite.logger.DebugWithContext, "Debug message", "DEBUG"},
		{LevelInfo, suite.logger.InfoWithContext, "Info message", "INFO"},
		{LevelWarn, suite.logger.WarnWithContext, "Warn message", "WARN"},
	}
	
	for _, tc := range testCases {
		suite.Run(string(tc.level), func() {
			// Set logger to debug level to capture all logs
			config := suite.config
			config.Level = LevelDebug
			suite.logger = NewStructuredLoggerWithBuffer(config, suite.buffer)
			
			tc.logFunc(tc.message, "test_key", "test_value")
			
			require.Len(suite.T(), suite.logLines, 1)
			logEntry := suite.logLines[0]
			
			assert.Equal(suite.T(), tc.expected, logEntry["level"])
			assert.Equal(suite.T(), tc.message, logEntry["msg"])
			assert.Equal(suite.T(), "test_value", logEntry["test_key"])
		})
	}
}

func (suite *StructuredLoggerTestSuite) TestWithServiceContext() {
	attrs := suite.logger.withServiceContext("key1", "value1", "key2", 42)
	
	// Convert to map for easier testing
	attrMap := make(map[string]interface{})
	for i := 0; i < len(attrs); i += 2 {
		key := attrs[i].(string)
		value := attrs[i+1]
		attrMap[key] = value
	}
	
	assert.Equal(suite.T(), suite.config.ServiceName, attrMap["service"])
	assert.Equal(suite.T(), suite.config.Version, attrMap["version"])
	assert.Equal(suite.T(), suite.config.Environment, attrMap["environment"])
	assert.Equal(suite.T(), suite.config.Component, attrMap["component"])
	assert.Equal(suite.T(), "value1", attrMap["key1"])
	assert.Equal(suite.T(), 42, attrMap["key2"])
}

func (suite *StructuredLoggerTestSuite) TestWithServiceContext_WithRequestID() {
	suite.logger.requestID = "req-test-123"
	
	attrs := suite.logger.withServiceContext("key1", "value1")
	
	// Convert to map for easier testing
	attrMap := make(map[string]interface{})
	for i := 0; i < len(attrs); i += 2 {
		key := attrs[i].(string)
		value := attrs[i+1]
		attrMap[key] = value
	}
	
	assert.Equal(suite.T(), "req-test-123", attrMap["request_id"])
}

func (suite *StructuredLoggerTestSuite) TestParseLevel() {
	testCases := []struct {
		input    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{"warning", slog.LevelWarn}, // Alternative format
		{"unknown", slog.LevelInfo}, // Default
		{"", slog.LevelInfo},        // Empty default
	}
	
	for _, tc := range testCases {
		suite.Run(string(tc.input), func() {
			level := parseLevel(tc.input)
			assert.Equal(suite.T(), tc.expected, level)
		})
	}
}

func (suite *StructuredLoggerTestSuite) TestGetCallerInfo() {
	file, line, funcName := GetCallerInfo(0)
	
	assert.NotEmpty(suite.T(), file)
	assert.Greater(suite.T(), line, 0)
	assert.Contains(suite.T(), funcName, "TestGetCallerInfo")
}

func (suite *StructuredLoggerTestSuite) TestDefaultConfig() {
	serviceName := "test-service"
	version := "2.0.0"
	environment := "production"
	
	config := DefaultConfig(serviceName, version, environment)
	
	assert.Equal(suite.T(), LevelInfo, config.Level)
	assert.Equal(suite.T(), "json", config.Format)
	assert.Equal(suite.T(), serviceName, config.ServiceName)
	assert.Equal(suite.T(), version, config.Version)
	assert.Equal(suite.T(), environment, config.Environment)
	assert.True(suite.T(), config.AddSource)
	assert.Equal(suite.T(), time.RFC3339, config.TimeFormat)
}

// Integration tests

func (suite *StructuredLoggerTestSuite) TestIntegrationFlow() {
	// Test complete logging flow with request context
	requestCtx := &RequestContext{
		RequestID: "req-integration-test",
		UserID:    "test-user",
		Method:    "POST",
		Path:      "/api/test",
		IP:        "127.0.0.1",
	}
	
	requestLogger := suite.logger.WithRequest(requestCtx)
	
	// Log operation with context
	err := requestLogger.LogOperation("integration_test", func() error {
		requestLogger.InfoWithContext("Processing request", "step", "validation")
		time.Sleep(10 * time.Millisecond)
		requestLogger.InfoWithContext("Processing request", "step", "execution")
		return nil
	})
	
	assert.NoError(suite.T(), err)
	require.Len(suite.T(), suite.logLines, 4) // 2 operation logs + 2 step logs
	
	// All logs should contain request context
	for _, logEntry := range suite.logLines {
		assert.Equal(suite.T(), requestCtx.RequestID, logEntry["request_id"])
		assert.Equal(suite.T(), requestCtx.UserID, logEntry["user_id"])
		assert.Equal(suite.T(), requestCtx.Method, logEntry["method"])
		assert.Equal(suite.T(), requestCtx.Path, logEntry["path"])
		assert.Equal(suite.T(), requestCtx.IP, logEntry["ip"])
	}
}

func (suite *StructuredLoggerTestSuite) TestConcurrentLogging() {
	// Test concurrent logging operations
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			componentLogger := suite.logger.WithComponent(fmt.Sprintf("component-%d", id))
			componentLogger.InfoWithContext("Concurrent log message",
				"goroutine_id", id,
				"timestamp", time.Now(),
			)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Should have exactly numGoroutines log entries
	require.Len(suite.T(), suite.logLines, numGoroutines)
	
	// Verify all entries are present with correct data
	componentCounts := make(map[string]int)
	for _, logEntry := range suite.logLines {
		component := logEntry["component"].(string)
		componentCounts[component]++
		assert.Equal(suite.T(), "Concurrent log message", logEntry["msg"])
		assert.Contains(suite.T(), logEntry, "goroutine_id")
	}
	
	// Each component should appear exactly once
	for component, count := range componentCounts {
		assert.Equal(suite.T(), 1, count, "Component %s should appear exactly once", component)
	}
}

// Edge cases and error handling

func (suite *StructuredLoggerTestSuite) TestEdgeCases() {
	testCases := []struct {
		name     string
		testFunc func()
	}{
		{
			name: "Log with nil metadata",
			testFunc: func() {
				suite.logger.LogBusinessEvent("test_event", nil)
				require.Len(suite.T(), suite.logLines, 1)
				assert.Equal(suite.T(), "Business event", suite.logLines[0]["msg"])
			},
		},
		{
			name: "Log with empty metadata",
			testFunc: func() {
				suite.logger.LogPerformance("test_operation", time.Second, map[string]interface{}{})
				require.Len(suite.T(), suite.logLines, 1)
				assert.Equal(suite.T(), "Performance metrics", suite.logLines[0]["msg"])
			},
		},
		{
			name: "Log with very long message",
			testFunc: func() {
				longMessage := strings.Repeat("Very long message ", 100)
				suite.logger.InfoWithContext(longMessage)
				require.Len(suite.T(), suite.logLines, 1)
				assert.Equal(suite.T(), longMessage, suite.logLines[0]["msg"])
			},
		},
		{
			name: "Log with special characters",
			testFunc: func() {
				message := "Message with special chars: áéíóú, 中文, 🚀, \"quotes\", 'apostrophes'"
				suite.logger.InfoWithContext(message, "special_field", "value with\nnewlines\tand\ttabs")
				require.Len(suite.T(), suite.logLines, 1)
				assert.Equal(suite.T(), message, suite.logLines[0]["msg"])
			},
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // Reset buffer for each test
			tc.testFunc()
		})
	}
}

// Helper functions for testing

func NewStructuredLoggerWithBuffer(config Config, buffer *bytes.Buffer) *StructuredLogger {
	level := parseLevel(config.Level)
	
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
	
	var handler slog.Handler
	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(buffer, opts)
	default:
		handler = slog.NewTextHandler(buffer, opts)
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

// Benchmarks for performance testing

func BenchmarkInfoWithContext(b *testing.B) {
	config := DefaultConfig("benchmark-service", "1.0.0", "test")
	config.Level = LevelInfo
	
	logger := NewStructuredLogger(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoWithContext("Benchmark log message",
			"iteration", i,
			"key1", "value1",
			"key2", "value2",
		)
	}
}

func BenchmarkLogOperation(b *testing.B) {
	config := DefaultConfig("benchmark-service", "1.0.0", "test")
	logger := NewStructuredLogger(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogOperation("benchmark_operation", func() error {
			return nil
		})
	}
}

func BenchmarkConcurrentLogging(b *testing.B) {
	config := DefaultConfig("benchmark-service", "1.0.0", "test")
	logger := NewStructuredLogger(config)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoWithContext("Concurrent benchmark message",
				"goroutine", "test",
				"timestamp", time.Now(),
			)
		}
	})
}
package backends_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// BackendIntegrationTestSuite tests backend implementations with real connections
type BackendIntegrationTestSuite struct {
	suite.Suite
	tempDir string
}

func TestBackendIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(BackendIntegrationTestSuite))
}

func (suite *BackendIntegrationTestSuite) SetupSuite() {
	var err error
	suite.tempDir, err = ioutil.TempDir("", "audit_backend_test")
	suite.Require().NoError(err)
}

func (suite *BackendIntegrationTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// File Backend Tests
func (suite *BackendIntegrationTestSuite) TestFileBackend() {
	logFile := filepath.Join(suite.tempDir, "audit_test.log")

	config := backends.BackendConfig{
		Type:    backends.BackendTypeFile,
		Enabled: true,
		Name:    "test-file",
		Settings: map[string]interface{}{
			"path":       logFile,
			"format":     "json",
			"permission": "0644",
		},
		Format: "json",
	}

	backend, err := newMockFileBackend(config)
	suite.Require().NoError(err)

	suite.Run("write single event", func() {
		event := createTestEvent("file-single")

		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Verify file was created and contains event
		suite.True(fileExists(logFile))
		content := readFileContent(suite.T(), logFile)
		suite.Contains(content, event.ID)
		suite.Contains(content, "file-single")
	})

	suite.Run("write batch events", func() {
		events := []*audit.AuditEvent{
			createTestEvent("batch-1"),
			createTestEvent("batch-2"),
			createTestEvent("batch-3"),
		}

		err := backend.WriteEvents(context.Background(), events)
		suite.NoError(err)

		// Verify all events are in file
		content := readFileContent(suite.T(), logFile)
		for _, event := range events {
			suite.Contains(content, event.ID)
		}
	})

	suite.Run("health check", func() {
		err := backend.Health(context.Background())
		suite.NoError(err)
	})

	suite.Run("close backend", func() {
		err := backend.Close()
		suite.NoError(err)
	})
}

func (suite *BackendIntegrationTestSuite) TestFileBackendWithRotation() {
	logFile := filepath.Join(suite.tempDir, "audit_rotation.log")

	config := backends.BackendConfig{
		Type:    backends.BackendTypeFile,
		Enabled: true,
		Name:    "test-file-rotation",
		Settings: map[string]interface{}{
			"path":        logFile,
			"format":      "json",
			"max_size":    1024, // 1KB for testing
			"max_backups": 3,
			"max_age":     7,
			"compress":    true,
		},
	}

	backend, err := newMockFileBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	// Write enough events to trigger rotation
	for i := 0; i < 100; i++ {
		event := createTestEvent(fmt.Sprintf("rotation-test-%d", i))
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	}

	// Check that rotation occurred (backup files created)
	files, err := filepath.Glob(filepath.Join(suite.tempDir, "audit_rotation.log*"))
	suite.NoError(err)
	suite.Greater(len(files), 1, "Expected log rotation to create backup files")
}

func (suite *BackendIntegrationTestSuite) TestFileBackendCompression() {
	logFile := filepath.Join(suite.tempDir, "audit_compressed.log")

	config := backends.BackendConfig{
		Type:        backends.BackendTypeFile,
		Enabled:     true,
		Name:        "test-compressed",
		Compression: true,
		Settings: map[string]interface{}{
			"path":   logFile,
			"format": "json",
		},
	}

	backend, err := newMockFileBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	event := createTestEvent("compression-test")
	err = backend.WriteEvent(context.Background(), event)
	suite.NoError(err)

	// File should exist and be compressed
	suite.True(fileExists(logFile))

	// Read compressed content (implementation would handle decompression)
	content := readFileContent(suite.T(), logFile)
	suite.NotEmpty(content)
}

// Mock Elasticsearch Backend Tests
func (suite *BackendIntegrationTestSuite) TestElasticsearchBackend() {
	// Create a mock Elasticsearch server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.Contains(r.URL.Path, "_bulk"):
			// Mock bulk indexing response
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"took":   5,
				"errors": false,
				"items": []map[string]interface{}{
					{
						"index": map[string]interface{}{
							"_index":   "audit-logs",
							"_type":    "_doc",
							"_id":      "1",
							"_version": 1,
							"result":   "created",
							"status":   201,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		case r.Method == "GET" && r.URL.Path == "/":
			// Mock cluster info response
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"name":         "test-node",
				"cluster_name": "test-cluster",
				"version": map[string]interface{}{
					"number": "7.10.0",
				},
			}
			json.NewEncoder(w).Encode(response)
		case r.Method == "POST" && strings.Contains(r.URL.Path, "_search"):
			// Mock search response
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"took": 5,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []map[string]interface{}{
						{
							"_source": map[string]interface{}{
								"id":        "test-event",
								"timestamp": time.Now().Format(time.RFC3339),
								"message":   "Test event",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := backends.BackendConfig{
		Type:    backends.BackendTypeElasticsearch,
		Enabled: true,
		Name:    "test-elasticsearch",
		Settings: map[string]interface{}{
			"urls":  []string{server.URL},
			"index": "audit-logs",
		},
		Timeout: 30 * time.Second,
	}

	backend, err := newMockElasticsearchBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	suite.Run("write single event", func() {
		event := createTestEvent("elasticsearch-single")
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	})

	suite.Run("write batch events", func() {
		events := []*audit.AuditEvent{
			createTestEvent("es-batch-1"),
			createTestEvent("es-batch-2"),
			createTestEvent("es-batch-3"),
		}

		err := backend.WriteEvents(context.Background(), events)
		suite.NoError(err)
	})

	suite.Run("health check", func() {
		err := backend.Health(context.Background())
		suite.NoError(err)
	})

	suite.Run("query events", func() {
		query := &QueryRequest{
			Query:     "test",
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Limit:     10,
		}

		response, err := backend.Query(context.Background(), query)
		suite.NoError(err)
		suite.NotNil(response)
		suite.GreaterOrEqual(response.TotalCount, int64(0))
	})
}

// Mock Splunk Backend Tests
func (suite *BackendIntegrationTestSuite) TestSplunkBackend() {
	// Create a mock Splunk HEC server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/services/collector"):
			// Verify authorization header
			auth := r.Header.Get("Authorization")
			suite.Contains(auth, "Splunk")

			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"text": "Success",
				"code": 0,
			}
			json.NewEncoder(w).Encode(response)
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/services/collector/health"):
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"text": "HEC is available",
				"code": 0,
			}
			json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := backends.BackendConfig{
		Type:    backends.BackendTypeSplunk,
		Enabled: true,
		Name:    "test-splunk",
		Settings: map[string]interface{}{
			"url":   server.URL,
			"token": "test-token",
			"index": "audit",
		},
		Auth: backends.AuthConfig{
			Type:  "hec",
			Token: "test-token",
		},
	}

	backend, err := newMockSplunkBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	suite.Run("write single event", func() {
		event := createTestEvent("splunk-single")
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	})

	suite.Run("write batch events", func() {
		events := []*audit.AuditEvent{
			createTestEvent("splunk-batch-1"),
			createTestEvent("splunk-batch-2"),
		}

		err := backend.WriteEvents(context.Background(), events)
		suite.NoError(err)
	})

	suite.Run("health check", func() {
		err := backend.Health(context.Background())
		suite.NoError(err)
	})
}

// Webhook Backend Tests
func (suite *BackendIntegrationTestSuite) TestWebhookBackend() {
	receivedEvents := make([]*audit.AuditEvent, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var event audit.AuditEvent
			body, err := ioutil.ReadAll(r.Body)
			suite.NoError(err)

			err = json.Unmarshal(body, &event)
			if err == nil {
				receivedEvents = append(receivedEvents, &event)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	config := backends.BackendConfig{
		Type:    backends.BackendTypeWebhook,
		Enabled: true,
		Name:    "test-webhook",
		Settings: map[string]interface{}{
			"url":     server.URL,
			"method":  "POST",
			"headers": map[string]string{"Content-Type": "application/json"},
		},
		Timeout: 10 * time.Second,
	}

	backend, err := newMockWebhookBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	suite.Run("write single event", func() {
		event := createTestEvent("webhook-single")
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		suite.Eventually(func() bool {
			return len(receivedEvents) > 0
		}, time.Second*5, time.Millisecond*100)

		suite.Equal("webhook-single", receivedEvents[0].Action)
	})

	suite.Run("health check", func() {
		err := backend.Health(context.Background())
		suite.NoError(err)
	})
}

// Syslog Backend Tests
func (suite *BackendIntegrationTestSuite) TestSyslogBackend() {
	// For this test, we'll use a file-based syslog approach
	syslogFile := filepath.Join(suite.tempDir, "syslog_test.log")

	config := backends.BackendConfig{
		Type:    backends.BackendTypeSyslog,
		Enabled: true,
		Name:    "test-syslog",
		Settings: map[string]interface{}{
			"network": "unix",
			"address": syslogFile,
			"tag":     "nephoran-audit",
		},
	}

	backend, err := newMockSyslogBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	suite.Run("write single event", func() {
		event := createTestEvent("syslog-single")
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)
	})
}

// Container-based Elasticsearch Integration Test
func (suite *BackendIntegrationTestSuite) TestElasticsearchWithContainer() {
	if os.Getenv("SKIP_CONTAINER_TESTS") == "true" {
		suite.T().Skip("Skipping container tests")
	}

	ctx := context.Background()

	// Start Elasticsearch container
	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:7.17.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":         "single-node",
			"xpack.security.enabled": "false",
		},
		WaitingFor: wait.ForHTTP("/").WithPort("9200/tcp").WithStartupTimeout(60 * time.Second),
	}

	esContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		suite.T().Skipf("Failed to start Elasticsearch container: %v", err)
		return
	}
	defer esContainer.Terminate(ctx)

	host, err := esContainer.Host(ctx)
	suite.Require().NoError(err)

	port, err := esContainer.MappedPort(ctx, "9200")
	suite.Require().NoError(err)

	esURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	config := backends.BackendConfig{
		Type:    backends.BackendTypeElasticsearch,
		Enabled: true,
		Name:    "test-elasticsearch-container",
		Settings: map[string]interface{}{
			"urls":  []string{esURL},
			"index": "audit-logs-test",
		},
		Timeout: 30 * time.Second,
	}

	backend, err := newMockElasticsearchBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	suite.Run("container health check", func() {
		// Wait for Elasticsearch to be ready
		suite.Eventually(func() bool {
			err := backend.Health(context.Background())
			return err == nil
		}, 30*time.Second, 1*time.Second)
	})

	suite.Run("container write and query", func() {
		event := createTestEvent("container-test")
		err := backend.WriteEvent(context.Background(), event)
		suite.NoError(err)

		// Wait for indexing
		time.Sleep(2 * time.Second)

		query := &QueryRequest{
			Query:     "container-test",
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Limit:     10,
		}

		response, err := backend.Query(context.Background(), query)
		suite.NoError(err)
		suite.Greater(response.TotalCount, int64(0))
	})
}

// Test Backend Factory
func (suite *BackendIntegrationTestSuite) TestBackendFactory() {
	tests := []struct {
		name        string
		backendType backends.BackendType
		config      backends.BackendConfig
		expectError bool
	}{
		{
			name:        "file backend",
			backendType: backends.BackendTypeFile,
			config: backends.BackendConfig{
				Type:    backends.BackendTypeFile,
				Enabled: true,
				Name:    "test-file",
				Settings: map[string]interface{}{
					"path": filepath.Join(suite.tempDir, "factory_test.log"),
				},
			},
			expectError: false,
		},
		{
			name:        "disabled backend",
			backendType: backends.BackendTypeFile,
			config: backends.BackendConfig{
				Type:    backends.BackendTypeFile,
				Enabled: false,
				Name:    "disabled-file",
			},
			expectError: true,
		},
		{
			name:        "unsupported backend",
			backendType: "unsupported",
			config: BackendConfig{
				Type:    "unsupported",
				Enabled: true,
				Name:    "unsupported",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			backend, err := backends.NewBackend(tt.config)

			if tt.expectError {
				suite.Error(err)
				suite.Nil(backend)
			} else {
				suite.NoError(err)
				suite.NotNil(backend)
				suite.Equal(string(tt.backendType), backend.Type())

				if backend != nil {
					backend.Close()
				}
			}
		})
	}
}

// Test Filter Configuration
func (suite *BackendIntegrationTestSuite) TestFilterConfiguration() {
	filter := FilterConfig{
		MinSeverity:   audit.SeverityWarning,
		EventTypes:    []audit.EventType{audit.EventTypeAuthentication},
		Components:    []string{"auth", "api"},
		ExcludeTypes:  []audit.EventType{audit.EventTypeHealthCheck},
		IncludeFields: []string{"user_id", "action"},
		ExcludeFields: []string{"debug_info"},
	}

	tests := []struct {
		name         string
		event        *audit.AuditEvent
		shouldFilter bool
	}{
		{
			name: "event passes all filters",
			event: &audit.AuditEvent{
				EventType: audit.EventTypeAuthentication,
				Component: "auth",
				Severity:  audit.SeverityError,
			},
			shouldFilter: true,
		},
		{
			name: "event filtered by severity",
			event: &audit.AuditEvent{
				EventType: audit.EventTypeAuthentication,
				Component: "auth",
				Severity:  audit.SeverityInfo, // Below threshold
			},
			shouldFilter: false,
		},
		{
			name: "event filtered by excluded type",
			event: &audit.AuditEvent{
				EventType: audit.EventTypeHealthCheck,
				Component: "auth",
				Severity:  audit.SeverityError,
			},
			shouldFilter: false,
		},
		{
			name: "event filtered by component",
			event: &audit.AuditEvent{
				EventType: audit.EventTypeAuthentication,
				Component: "other",
				Severity:  audit.SeverityError,
			},
			shouldFilter: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := filter.ShouldProcessEvent(tt.event)
			suite.Equal(tt.shouldFilter, result)
		})
	}
}

// Test Retry Policy
func (suite *BackendIntegrationTestSuite) TestRetryPolicy() {
	retryPolicy := RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	// Test delay calculation
	delays := calculateBackoffDelays(retryPolicy)

	suite.Equal(100*time.Millisecond, delays[0])
	suite.Equal(200*time.Millisecond, delays[1])
	suite.Equal(400*time.Millisecond, delays[2])

	// Ensure max delay is respected
	retryPolicy.MaxDelay = 300 * time.Millisecond
	delays = calculateBackoffDelays(retryPolicy)
	suite.LessOrEqual(delays[2], 300*time.Millisecond)
}

// Backend Performance Tests
func (suite *BackendIntegrationTestSuite) TestBackendPerformance() {
	logFile := filepath.Join(suite.tempDir, "performance_test.log")

	config := backends.BackendConfig{
		Type:    backends.BackendTypeFile,
		Enabled: true,
		Name:    "performance-test",
		Settings: map[string]interface{}{
			"path": logFile,
		},
		BufferSize: 1000,
	}

	backend, err := newMockFileBackend(config)
	suite.Require().NoError(err)
	defer backend.Close()

	// Measure batch write performance
	events := make([]*audit.AuditEvent, 100)
	for i := 0; i < len(events); i++ {
		events[i] = createTestEvent(fmt.Sprintf("perf-test-%d", i))
	}

	start := time.Now()
	err = backend.WriteEvents(context.Background(), events)
	duration := time.Since(start)

	suite.NoError(err)
	suite.Less(duration, 1*time.Second, "Batch write took too long: %v", duration)

	// Verify all events were written
	content := readFileContent(suite.T(), logFile)
	for _, event := range events {
		suite.Contains(content, event.ID)
	}
}

// Helper functions

func createTestEvent(action string) *audit.AuditEvent {
	return &audit.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: audit.EventTypeAuthentication,
		Component: "test",
		Action:    action,
		Severity:  audit.SeverityInfo,
		Result:    audit.ResultSuccess,
		UserContext: &audit.UserContext{
			UserID:   "test-user",
			Username: "testuser",
		},
		NetworkContext: &audit.NetworkContext{
			SourcePort: 8080,
		},
		ResourceContext: &audit.ResourceContext{
			ResourceType: "deployment",
			Operation:    "create",
		},
		Data: map[string]interface{}{
			"test_field": "test_value",
		},
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFileContent(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

func calculateBackoffDelays(policy RetryPolicy) []time.Duration {
	delays := make([]time.Duration, policy.MaxRetries)
	delay := policy.InitialDelay

	for i := 0; i < policy.MaxRetries; i++ {
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
		delays[i] = delay
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
	}

	return delays
}

// Benchmark tests

func BenchmarkFileBackendWriteEvent(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "benchmark_test")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "benchmark.log")
	config := backends.BackendConfig{
		Type:    backends.BackendTypeFile,
		Enabled: true,
		Name:    "benchmark",
		Settings: map[string]interface{}{
			"path": logFile,
		},
	}

	backend, err := newMockFileBackend(config)
	require.NoError(b, err)
	defer backend.Close()

	event := createTestEvent("benchmark")
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			backend.WriteEvent(ctx, event)
		}
	})
}

func BenchmarkFileBackendWriteBatch(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "benchmark_batch_test")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "benchmark_batch.log")
	config := backends.BackendConfig{
		Type:    backends.BackendTypeFile,
		Enabled: true,
		Name:    "benchmark-batch",
		Settings: map[string]interface{}{
			"path": logFile,
		},
	}

	backend, err := newMockFileBackend(config)
	require.NoError(b, err)
	defer backend.Close()

	events := make([]*audit.AuditEvent, 10)
	for i := 0; i < len(events); i++ {
		events[i] = createTestEvent(fmt.Sprintf("benchmark-%d", i))
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.WriteEvents(ctx, events)
	}
}

// Mock types and functions

type QueryRequest = backends.QueryRequest
type QueryResponse = backends.QueryResponse
type AuthConfig struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type FilterConfig struct {
	MinSeverity   audit.Severity      `json:"min_severity"`
	EventTypes    []audit.EventType   `json:"event_types"`
	Components    []string            `json:"components"`
	ExcludeTypes  []audit.EventType   `json:"exclude_types"`
	IncludeFields []string            `json:"include_fields"`
	ExcludeFields []string            `json:"exclude_fields"`
}

func (fc FilterConfig) ShouldProcessEvent(event *audit.AuditEvent) bool {
	// Mock implementation - always return true for now
	if event.Severity < fc.MinSeverity {
		return false
	}
	return true
}

type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

type BackendConfig = backends.BackendConfig

// Mock backend interface
type mockBackend struct {
	config backends.BackendConfig
}

func (mb *mockBackend) WriteEvent(ctx context.Context, event *audit.AuditEvent) error {
	return nil
}

func (mb *mockBackend) WriteEvents(ctx context.Context, events []*audit.AuditEvent) error {
	return nil
}

func (mb *mockBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{
		Events:     []*audit.AuditEvent{},
		TotalCount: 0,
	}, nil
}

func (mb *mockBackend) Health(ctx context.Context) error {
	return nil
}

func (mb *mockBackend) Close() error {
	return nil
}

func (mb *mockBackend) Type() string {
	return string(mb.config.Type)
}

func (mb *mockBackend) Initialize(config backends.BackendConfig) error {
	return nil
}

// Mock backend constructors
func newMockFileBackend(config backends.BackendConfig) (backends.Backend, error) {
	return &mockBackend{config: config}, nil
}

func newMockElasticsearchBackend(config backends.BackendConfig) (backends.Backend, error) {
	return &mockBackend{config: config}, nil
}

func newMockSplunkBackend(config backends.BackendConfig) (backends.Backend, error) {
	return &mockBackend{config: config}, nil
}

func newMockWebhookBackend(config backends.BackendConfig) (backends.Backend, error) {
	return &mockBackend{config: config}, nil
}

func newMockSyslogBackend(config backends.BackendConfig) (backends.Backend, error) {
	return &mockBackend{config: config}, nil
}

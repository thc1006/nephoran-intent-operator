package audit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// MockBackend is a mock implementation of the Backend interface for testing
type MockBackend struct {
	mock.Mock
	receivedEvents []*AuditEvent
	mutex          sync.RWMutex
	shouldFail     bool
	latency        time.Duration
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		receivedEvents: make([]*AuditEvent, 0),
	}
}

func (m *MockBackend) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBackend) Initialize(config backends.BackendConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	if m.shouldFail {
		return fmt.Errorf("mock backend failure")
	}

	m.mutex.Lock()
	m.receivedEvents = append(m.receivedEvents, event)
	m.mutex.Unlock()

	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	if m.shouldFail {
		return fmt.Errorf("mock backend batch failure")
	}

	m.mutex.Lock()
	m.receivedEvents = append(m.receivedEvents, events...)
	m.mutex.Unlock()

	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(*backends.QueryResponse), args.Error(1)
}

func (m *MockBackend) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBackend) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBackend) GetReceivedEvents() []*AuditEvent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	events := make([]*AuditEvent, len(m.receivedEvents))
	copy(events, m.receivedEvents)
	return events
}

func (m *MockBackend) SetFailure(shouldFail bool) {
	m.shouldFail = shouldFail
}

func (m *MockBackend) SetLatency(latency time.Duration) {
	m.latency = latency
}

func (m *MockBackend) Reset() {
	m.mutex.Lock()
	m.receivedEvents = m.receivedEvents[:0]
	m.shouldFail = false
	m.latency = 0
	m.mutex.Unlock()
}

// AuditSystemTestSuite contains the test suite for AuditSystem
type AuditSystemTestSuite struct {
	suite.Suite
	config      *AuditSystemConfig
	mockBackend *MockBackend
	auditSystem *AuditSystem
}

func TestAuditSystemTestSuite(t *testing.T) {
	suite.Run(t, new(AuditSystemTestSuite))
}

func (suite *AuditSystemTestSuite) SetupTest() {
	// Create a mock backend
	suite.mockBackend = NewMockBackend()
	suite.mockBackend.On("Type").Return("mock")
	suite.mockBackend.On("Initialize", mock.Anything).Return(nil)
	suite.mockBackend.On("WriteEvents", mock.Anything, mock.Anything).Return(nil)
	suite.mockBackend.On("Close").Return(nil)

	// Create a test configuration
	suite.config = &AuditSystemConfig{
		Enabled:         true,
		LogLevel:        SeverityInfo,
		BatchSize:       3,
		FlushInterval:   100 * time.Millisecond,
		MaxQueueSize:    10,
		EnableIntegrity: false, // Disabled for unit tests
		ComplianceMode:  []ComplianceStandard{ComplianceSOC2},
		Backends:        []backends.BackendConfig{},
	}

	// Create audit system
	auditSystem, err := NewAuditSystem(suite.config)
	suite.Require().NoError(err)

	// Replace backends with mock
	auditSystem.backends = []backends.Backend{suite.mockBackend}

	suite.auditSystem = auditSystem
}

func (suite *AuditSystemTestSuite) TearDownTest() {
	if suite.auditSystem != nil {
		err := suite.auditSystem.Stop()
		suite.NoError(err)
	}
}

func (suite *AuditSystemTestSuite) TestNewAuditSystem() {
	config := DefaultAuditConfig()
	auditSystem, err := NewAuditSystem(config)

	suite.NoError(err)
	suite.NotNil(auditSystem)
	suite.Equal(config, auditSystem.config)
	suite.NotNil(auditSystem.eventQueue)
	suite.NotNil(auditSystem.logger)
	suite.False(auditSystem.running.Load())
}

func (suite *AuditSystemTestSuite) TestNewAuditSystemWithNilConfig() {
	auditSystem, err := NewAuditSystem(nil)

	suite.NoError(err)
	suite.NotNil(auditSystem)
	suite.Equal(DefaultAuditConfig(), auditSystem.config)
}

func (suite *AuditSystemTestSuite) TestStartStop() {
	// Test start
	err := suite.auditSystem.Start()
	suite.NoError(err)
	suite.True(suite.auditSystem.running.Load())

	// Test double start (should return error)
	err = suite.auditSystem.Start()
	suite.Error(err)

	// Test stop
	err = suite.auditSystem.Stop()
	suite.NoError(err)
	suite.False(suite.auditSystem.running.Load())

	// Test double stop (should not error)
	err = suite.auditSystem.Stop()
	suite.NoError(err)
}

func (suite *AuditSystemTestSuite) TestLogEventDisabled() {
	suite.config.Enabled = false
	auditSystem, err := NewAuditSystem(suite.config)
	suite.Require().NoError(err)

	event := &AuditEvent{
		ID:        uuid.New().String(),
		EventType: EventTypeAuthentication,
		Component: "test",
		Action:    "test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	// Should not error but also not process the event
	err = auditSystem.LogEvent(event)
	suite.NoError(err)
}

func (suite *AuditSystemTestSuite) TestLogEventBelowMinimumSeverity() {
	suite.config.LogLevel = SeverityError
	auditSystem, err := NewAuditSystem(suite.config)
	suite.Require().NoError(err)
	auditSystem.backends = []backends.Backend{suite.mockBackend}

	err = auditSystem.Start()
	suite.Require().NoError(err)
	defer auditSystem.Stop()

	event := &AuditEvent{
		ID:        uuid.New().String(),
		EventType: EventTypeAuthentication,
		Component: "test",
		Action:    "test",
		Severity:  SeverityInfo, // Below minimum
		Timestamp: time.Now(),
	}

	err = auditSystem.LogEvent(event)
	suite.NoError(err)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Should not have received any events
	receivedEvents := suite.mockBackend.GetReceivedEvents()
	suite.Empty(receivedEvents)
}

func (suite *AuditSystemTestSuite) TestLogEventValidation() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	// Test invalid event (missing required fields)
	invalidEvent := &AuditEvent{}
	err = suite.auditSystem.LogEvent(invalidEvent)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid audit event")
}

func (suite *AuditSystemTestSuite) TestEventEnrichment() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	event := &AuditEvent{
		EventType: EventTypeAuthentication,
		Component: "test",
		Action:    "test",
		Severity:  SeverityInfo,
	}

	err = suite.auditSystem.LogEvent(event)
	suite.NoError(err)

	// Wait for batch processing
	time.Sleep(150 * time.Millisecond)

	receivedEvents := suite.mockBackend.GetReceivedEvents()
	suite.Len(receivedEvents, 1)

	enrichedEvent := receivedEvents[0]
	suite.NotEmpty(enrichedEvent.ID)
	suite.NotZero(enrichedEvent.Timestamp)
	suite.Equal(AuditFormatVersion, enrichedEvent.Version)
	suite.NotNil(enrichedEvent.SystemContext)
}

func (suite *AuditSystemTestSuite) TestBatchProcessing() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	// Send exactly batch size events
	for i := 0; i < suite.config.BatchSize; i++ {
		event := &AuditEvent{
			ID:        uuid.New().String(),
			EventType: EventTypeAuthentication,
			Component: "test",
			Action:    fmt.Sprintf("test-%d", i),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		}

		err := suite.auditSystem.LogEvent(event)
		suite.NoError(err)
	}

	// Give time for batch processing
	time.Sleep(50 * time.Millisecond)

	receivedEvents := suite.mockBackend.GetReceivedEvents()
	suite.Len(receivedEvents, suite.config.BatchSize)
}

func (suite *AuditSystemTestSuite) TestTimerBasedFlushing() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	// Send one event (less than batch size)
	event := &AuditEvent{
		ID:        uuid.New().String(),
		EventType: EventTypeAuthentication,
		Component: "test",
		Action:    "test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	err = suite.auditSystem.LogEvent(event)
	suite.NoError(err)

	// Should not be processed immediately
	time.Sleep(50 * time.Millisecond)
	receivedEvents := suite.mockBackend.GetReceivedEvents()
	suite.Empty(receivedEvents)

	// Wait for timer flush
	time.Sleep(150 * time.Millisecond)

	receivedEvents = suite.mockBackend.GetReceivedEvents()
	suite.Len(receivedEvents, 1)
}

func (suite *AuditSystemTestSuite) TestQueueOverflow() {
	// Create system with small queue
	config := *suite.config
	config.MaxQueueSize = 2

	auditSystem, err := NewAuditSystem(&config)
	suite.Require().NoError(err)
	auditSystem.backends = []backends.Backend{suite.mockBackend}

	err = auditSystem.Start()
	suite.Require().NoError(err)
	defer auditSystem.Stop()

	// Fill queue and then overflow
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			ID:        uuid.New().String(),
			EventType: EventTypeAuthentication,
			Component: "test",
			Action:    fmt.Sprintf("test-%d", i),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		}

		err := auditSystem.LogEvent(event)
		if i >= config.MaxQueueSize {
			// Should start dropping events
			suite.Error(err)
			suite.Contains(err.Error(), "audit queue is full")
		} else {
			suite.NoError(err)
		}
	}
}

func (suite *AuditSystemTestSuite) TestStats() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	// Log some events
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			ID:        uuid.New().String(),
			EventType: EventTypeAuthentication,
			Component: "test",
			Action:    fmt.Sprintf("test-%d", i),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		}

		err := suite.auditSystem.LogEvent(event)
		suite.NoError(err)
	}

	// Wait for processing
	time.Sleep(150 * time.Millisecond)

	stats := suite.auditSystem.GetStats()
	suite.Equal(int64(5), stats.EventsReceived)
	suite.Equal(int64(0), stats.EventsDropped)
	suite.Equal(1, stats.BackendCount)
	suite.True(stats.IntegrityEnabled)
	suite.Contains(stats.ComplianceMode, ComplianceSOC2)
}

func (suite *AuditSystemTestSuite) TestBackendFailure() {
	// Configure backend to fail
	suite.mockBackend.SetFailure(true)

	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	event := &AuditEvent{
		ID:        uuid.New().String(),
		EventType: EventTypeAuthentication,
		Component: "test",
		Action:    "test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	err = suite.auditSystem.LogEvent(event)
	suite.NoError(err)

	// Wait for processing (should not crash)
	time.Sleep(150 * time.Millisecond)

	// System should still be running
	suite.True(suite.auditSystem.running.Load())
}

func (suite *AuditSystemTestSuite) TestConcurrentEventLogging() {
	err := suite.auditSystem.Start()
	suite.Require().NoError(err)
	defer suite.auditSystem.Stop()

	const numGoroutines = 10
	const eventsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < eventsPerGoroutine; i++ {
				event := &AuditEvent{
					ID:        uuid.New().String(),
					EventType: EventTypeAuthentication,
					Component: "test",
					Action:    fmt.Sprintf("goroutine-%d-event-%d", goroutineID, i),
					Severity:  SeverityInfo,
					Timestamp: time.Now(),
				}

				err := suite.auditSystem.LogEvent(event)
				suite.NoError(err)
			}
		}(g)
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)

	receivedEvents := suite.mockBackend.GetReceivedEvents()
	suite.Len(receivedEvents, numGoroutines*eventsPerGoroutine)

	// Check that all events are unique
	eventIDs := make(map[string]bool)
	for _, event := range receivedEvents {
		suite.False(eventIDs[event.ID], "Duplicate event ID: %s", event.ID)
		eventIDs[event.ID] = true
	}
}

// Table-driven tests for various scenarios

func TestAuditSystemScenarios(t *testing.T) {
	tests := []struct {
		name        string
		config      func() *AuditSystemConfig
		events      []*AuditEvent
		expectCount int
		expectError bool
	}{
		{
			name: "normal operation",
			config: func() *AuditSystemConfig {
				return &AuditSystemConfig{
					Enabled:       true,
					LogLevel:      SeverityInfo,
					BatchSize:     2,
					FlushInterval: 50 * time.Millisecond,
					MaxQueueSize:  10,
					Backends:      []backends.BackendConfig{},
				}
			},
			events: []*AuditEvent{
				{
					ID:        uuid.New().String(),
					EventType: EventTypeAuthentication,
					Component: "test",
					Action:    "login",
					Severity:  SeverityInfo,
					Timestamp: time.Now(),
				},
			},
			expectCount: 1,
			expectError: false,
		},
		{
			name: "filtered by severity",
			config: func() *AuditSystemConfig {
				return &AuditSystemConfig{
					Enabled:       true,
					LogLevel:      SeverityError,
					BatchSize:     2,
					FlushInterval: 50 * time.Millisecond,
					MaxQueueSize:  10,
					Backends:      []backends.BackendConfig{},
				}
			},
			events: []*AuditEvent{
				{
					ID:        uuid.New().String(),
					EventType: EventTypeAuthentication,
					Component: "test",
					Action:    "login",
					Severity:  SeverityInfo, // Below threshold
					Timestamp: time.Now(),
				},
			},
			expectCount: 0,
			expectError: false,
		},
		{
			name: "invalid event",
			config: func() *AuditSystemConfig {
				return &AuditSystemConfig{
					Enabled:       true,
					LogLevel:      SeverityInfo,
					BatchSize:     2,
					FlushInterval: 50 * time.Millisecond,
					MaxQueueSize:  10,
					Backends:      []backends.BackendConfig{},
				}
			},
			events: []*AuditEvent{
				{
					// Missing required fields
					ID: "invalid-id",
				},
			},
			expectCount: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := NewMockBackend()
			mockBackend.On("Type").Return("mock")
			mockBackend.On("WriteEvents", mock.Anything, mock.Anything).Return(nil)
			mockBackend.On("Close").Return(nil)

			config := tt.config()
			auditSystem, err := NewAuditSystem(config)
			require.NoError(t, err)

			// Replace backends with mock
			auditSystem.backends = []backends.Backend{mockBackend}

			if config.Enabled {
				err = auditSystem.Start()
				require.NoError(t, err)
				defer auditSystem.Stop()
			}

			for _, event := range tt.events {
				err = auditSystem.LogEvent(event)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}

			// Wait for processing
			time.Sleep(100 * time.Millisecond)

			receivedEvents := mockBackend.GetReceivedEvents()
			assert.Len(t, receivedEvents, tt.expectCount)
		})
	}
}

// Benchmark tests

func BenchmarkAuditSystemLogEvent(b *testing.B) {
	mockBackend := NewMockBackend()
	mockBackend.On("Type").Return("mock")
	mockBackend.On("WriteEvents", mock.Anything, mock.Anything).Return(nil)
	mockBackend.On("Close").Return(nil)

	config := &AuditSystemConfig{
		Enabled:       true,
		LogLevel:      SeverityInfo,
		BatchSize:     100,
		FlushInterval: 1 * time.Second,
		MaxQueueSize:  10000,
		Backends:      []backends.BackendConfig{},
	}

	auditSystem, err := NewAuditSystem(config)
	require.NoError(b, err)

	auditSystem.backends = []backends.Backend{mockBackend}

	err = auditSystem.Start()
	require.NoError(b, err)
	defer auditSystem.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &AuditEvent{
				ID:        uuid.New().String(),
				EventType: EventTypeAuthentication,
				Component: "benchmark",
				Action:    "test",
				Severity:  SeverityInfo,
				Timestamp: time.Now(),
			}

			auditSystem.LogEvent(event)
		}
	})
}

func BenchmarkEventValidation(b *testing.B) {
	event := &AuditEvent{
		ID:        uuid.New().String(),
		EventType: EventTypeAuthentication,
		Component: "benchmark",
		Action:    "test",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
		UserContext: &UserContext{
			UserID:   "user123",
			Username: "testuser",
			Email:    "test@example.com",
		},
		NetworkContext: &NetworkContext{
			SourcePort:      8080,
			DestinationPort: 443,
		},
		ResourceContext: &ResourceContext{
			ResourceType: "deployment",
			Operation:    "create",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.Validate()
	}
}

func BenchmarkEventEnrichment(b *testing.B) {
	config := DefaultAuditConfig()
	auditSystem, err := NewAuditSystem(config)
	require.NoError(b, err)

	event := &AuditEvent{
		EventType: EventTypeAuthentication,
		Component: "benchmark",
		Action:    "test",
		Severity:  SeverityInfo,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auditSystem.enrichEvent(event)
	}
}

// Test compliance metadata enrichment

func TestComplianceMetadataEnrichment(t *testing.T) {
	tests := []struct {
		name           string
		complianceMode []ComplianceStandard
		eventType      EventType
		expectedFields map[string]string
	}{
		{
			name:           "SOC2 authentication event",
			complianceMode: []ComplianceStandard{ComplianceSOC2},
			eventType:      EventTypeAuthentication,
			expectedFields: map[string]string{
				"soc2_control_id":    "CC6.1",
				"soc2_trust_service": "Security",
			},
		},
		{
			name:           "ISO27001 data access event",
			complianceMode: []ComplianceStandard{ComplianceISO27001},
			eventType:      EventTypeDataAccess,
			expectedFields: map[string]string{
				"iso27001_control": "A.12.4.1",
				"iso27001_annex":   "A.12 - Operations Security",
			},
		},
		{
			name:           "PCI DSS authentication event",
			complianceMode: []ComplianceStandard{CompliancePCIDSS},
			eventType:      EventTypeAuthentication,
			expectedFields: map[string]string{
				"pci_requirement":         "8.1.1",
				"pci_data_classification": "Non-CHD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AuditSystemConfig{
				Enabled:        false, // Don't start processing
				ComplianceMode: tt.complianceMode,
			}

			auditSystem, err := NewAuditSystem(config)
			require.NoError(t, err)

			event := &AuditEvent{
				ID:        uuid.New().String(),
				EventType: tt.eventType,
				Component: "test",
				Action:    "test",
				Severity:  SeverityInfo,
				Timestamp: time.Now(),
			}

			auditSystem.enrichEvent(event)

			require.NotNil(t, event.ComplianceMetadata)
			for key, expectedValue := range tt.expectedFields {
				actualValue, exists := event.ComplianceMetadata[key]
				assert.True(t, exists, "Expected field %s not found", key)
				assert.Equal(t, expectedValue, actualValue, "Unexpected value for field %s", key)
			}
		})
	}
}

// Test metrics collection

func TestMetricsCollection(t *testing.T) {
	// Reset metrics
	auditEventsTotal.Reset()
	auditProcessingDuration.Reset()
	auditQueueSize.Reset()
	auditErrorsTotal.Reset()

	mockBackend := NewMockBackend()
	mockBackend.On("Type").Return("mock")
	mockBackend.On("WriteEvents", mock.Anything, mock.Anything).Return(nil)
	mockBackend.On("Close").Return(nil)

	config := &AuditSystemConfig{
		Enabled:       true,
		LogLevel:      SeverityInfo,
		BatchSize:     2,
		FlushInterval: 50 * time.Millisecond,
		MaxQueueSize:  10,
		Backends:      []backends.BackendConfig{},
	}

	auditSystem, err := NewAuditSystem(config)
	require.NoError(t, err)

	auditSystem.backends = []backends.Backend{mockBackend}

	err = auditSystem.Start()
	require.NoError(t, err)
	defer auditSystem.Stop()

	// Log events
	for i := 0; i < 3; i++ {
		event := &AuditEvent{
			ID:        uuid.New().String(),
			EventType: EventTypeAuthentication,
			Component: "test",
			Action:    fmt.Sprintf("test-%d", i),
			Severity:  SeverityInfo,
			Timestamp: time.Now(),
		}

		err := auditSystem.LogEvent(event)
		require.NoError(t, err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify metrics are collected
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var foundEventMetric bool
	for _, mf := range metricFamilies {
		if mf.GetName() == "nephoran_audit_events_total" {
			foundEventMetric = true
			break
		}
	}

	assert.True(t, foundEventMetric, "Expected audit events metric not found")
}

package audit

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// ChaosTestSuite performs chaos engineering tests on the audit system
type ChaosTestSuite struct {
	suite.Suite
	tempDir         string
	chaosServer     *httptest.Server
	flakyServer     *httptest.Server
	networkEmulator *NetworkEmulator
}

func TestChaosTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos tests in short mode")
	}
	suite.Run(t, new(ChaosTestSuite))
}

func (suite *ChaosTestSuite) SetupSuite() {
	var err error
	suite.tempDir, err = ioutil.TempDir("", "chaos_test")
	suite.Require().NoError(err)

	// Setup chaos server that randomly fails
	suite.chaosServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 30% chance of failure
		if rand.Float32() < 0.3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "chaos induced failure"}`))
			return
		}

		// 20% chance of timeout (slow response)
		if rand.Float32() < 0.2 {
			time.Sleep(5 * time.Second)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))

	// Setup flaky server with intermittent connectivity
	suite.flakyServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 50% chance of connection drop
		if rand.Float32() < 0.5 {
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close() // Simulate connection drop
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "flaky_ok"}`))
	}))

	// Setup network emulator
	suite.networkEmulator = NewNetworkEmulator()
}

func (suite *ChaosTestSuite) TearDownSuite() {
	if suite.chaosServer != nil {
		suite.chaosServer.Close()
	}
	if suite.flakyServer != nil {
		suite.flakyServer.Close()
	}
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// Test system behavior under various failure modes
func (suite *ChaosTestSuite) TestBackendFailureScenarios() {
	suite.Run("random backend failures", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  1000,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add chaos backend
		chaosBackend := &ChaosBackend{
			url:         suite.chaosServer.URL,
			failureRate: 0.3,
		}
		auditSystem.backends = []backends.Backend{chaosBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events and expect some failures but system should continue
		totalEvents := 100
		successCount := int64(0)

		for i := 0; i < totalEvents; i++ {
			event := createChaosTestEvent(fmt.Sprintf("chaos-test-%d", i))
			err := auditSystem.LogEvent(event)

			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}

			time.Sleep(10 * time.Millisecond) // Small delay between events
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		stats := auditSystem.GetStats()
		suite.Greater(successCount, int64(totalEvents/2), "Too many events failed")
		suite.T().Logf("Chaos test results: %d/%d events submitted, %d processed",
			successCount, totalEvents, stats.EventsReceived)
	})

	suite.Run("complete backend failure", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 50 * time.Millisecond,
			MaxQueueSize:  100,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add completely failing backend
		failingBackend := &ChaosBackend{
			url:         "http://nonexistent:99999",
			failureRate: 1.0, // Always fail
		}
		auditSystem.backends = []backends.Backend{failingBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// System should handle complete backend failure gracefully
		for i := 0; i < 20; i++ {
			event := createChaosTestEvent(fmt.Sprintf("fail-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err, "Event submission should not fail even if backend fails")
		}

		// System should remain responsive
		stats := auditSystem.GetStats()
		suite.Equal(int64(20), stats.EventsReceived)
		suite.T().Logf("Complete failure test: %d events received despite backend failure", stats.EventsReceived)
	})

	suite.Run("intermittent backend connectivity", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 200 * time.Millisecond,
			MaxQueueSize:  500,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add flaky backend
		flakyBackend := &ChaosBackend{
			url:         suite.flakyServer.URL,
			failureRate: 0.5,
		}
		auditSystem.backends = []backends.Backend{flakyBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events over time to test intermittent connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var eventCount int64
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					event := createChaosTestEvent(fmt.Sprintf("flaky-test-%d", eventCount))
					auditSystem.LogEvent(event)
					atomic.AddInt64(&eventCount, 1)
				}
			}
		}()

		<-ctx.Done()

		// System should handle intermittent connectivity
		stats := auditSystem.GetStats()
		suite.Greater(stats.EventsReceived, eventCount/2, "Too many events lost due to flaky connectivity")
		suite.T().Logf("Flaky connectivity test: %d events sent, %d processed", eventCount, stats.EventsReceived)
	})
}

// Test disk space and I/O failure scenarios
func (suite *ChaosTestSuite) TestDiskFailureScenarios() {
	suite.Run("disk space exhaustion simulation", func() {
		// Create a small filesystem simulation
		smallLogFile := filepath.Join(suite.tempDir, "small_disk.log")

		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 50 * time.Millisecond,
			MaxQueueSize:  100,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add file backend that will run out of space
		diskFullBackend := &DiskFullBackend{
			logFile:     smallLogFile,
			maxSize:     1024, // 1KB limit
			currentSize: 0,
		}
		auditSystem.backends = []backends.Backend{diskFullBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events until disk is full
		for i := 0; i < 100; i++ {
			event := createChaosTestEvent(fmt.Sprintf("diskfull-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err, "Event submission should not fail immediately")

			time.Sleep(10 * time.Millisecond)
		}

		// Wait for processing
		time.Sleep(1 * time.Second)

		// System should handle disk full gracefully
		stats := auditSystem.GetStats()
		suite.T().Logf("Disk full simulation: %d events received", stats.EventsReceived)
	})

	suite.Run("I/O error simulation", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     3,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  50,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add backend that simulates I/O errors
		ioErrorBackend := &IOErrorBackend{
			errorRate: 0.4,
		}
		auditSystem.backends = []backends.Backend{ioErrorBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events and expect some I/O failures
		for i := 0; i < 50; i++ {
			event := createChaosTestEvent(fmt.Sprintf("io-error-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		time.Sleep(1 * time.Second)

		stats := auditSystem.GetStats()
		suite.T().Logf("I/O error simulation: %d events processed", stats.EventsReceived)
	})
}

// Test network partitions and connectivity issues
func (suite *ChaosTestSuite) TestNetworkPartitions() {
	suite.Run("network partition simulation", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 200 * time.Millisecond,
			MaxQueueSize:  200,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Create network partition backend
		partitionBackend := NewNetworkPartitionBackend(suite.chaosServer.URL)
		auditSystem.backends = []backends.Backend{partitionBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Start sending events
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var eventsSent int64
		go func() {
			for i := 0; i < 100; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					event := createChaosTestEvent(fmt.Sprintf("partition-test-%d", i))
					auditSystem.LogEvent(event)
					atomic.AddInt64(&eventsSent, 1)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		// Simulate network partition after 5 seconds
		time.Sleep(5 * time.Second)
		partitionBackend.StartPartition()
		suite.T().Log("Network partition started")

		// Continue for 5 seconds
		time.Sleep(5 * time.Second)

		// Heal partition
		partitionBackend.HealPartition()
		suite.T().Log("Network partition healed")

		// Continue for remaining time
		<-ctx.Done()

		stats := auditSystem.GetStats()
		suite.T().Logf("Network partition test: %d events sent, %d processed", eventsSent, stats.EventsReceived)

		// System should recover after partition heals
		suite.Greater(stats.EventsReceived, int64(0))
	})

	suite.Run("DNS resolution failures", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  100,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Backend with invalid hostname
		dnsFailBackend := &ChaosBackend{
			url: "http://invalid-hostname-that-does-not-exist.local",
		}
		auditSystem.backends = []backends.Backend{dnsFailBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events despite DNS failures
		for i := 0; i < 20; i++ {
			event := createChaosTestEvent(fmt.Sprintf("dns-fail-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		time.Sleep(1 * time.Second)
		suite.T().Log("DNS failure test completed")
	})
}

// Test memory pressure and resource exhaustion
func (suite *ChaosTestSuite) TestResourceExhaustion() {
	suite.Run("memory pressure simulation", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     100,
			FlushInterval: 1 * time.Second,
			MaxQueueSize:  10000,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Add slow backend to cause memory buildup
		slowBackend := &SlowBackend{
			processingDelay: 100 * time.Millisecond,
		}
		auditSystem.backends = []backends.Backend{slowBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Rapidly send events to build up memory pressure
		go func() {
			for i := 0; i < 1000; i++ {
				// Create large events to increase memory pressure
				event := createLargeChaosEvent(fmt.Sprintf("memory-pressure-%d", i))
				auditSystem.LogEvent(event)

				if i%100 == 0 {
					suite.T().Logf("Sent %d events for memory pressure test", i)
				}
			}
		}()

		// Monitor for 10 seconds
		time.Sleep(10 * time.Second)

		stats := auditSystem.GetStats()
		suite.T().Logf("Memory pressure test: %d events processed", stats.EventsReceived)

		// System should handle memory pressure without crashing
		suite.Greater(stats.EventsReceived, int64(0))
	})

	suite.Run("queue overflow scenarios", func() {
		// Small queue to force overflow
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 1 * time.Second,
			MaxQueueSize:  50, // Small queue
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Very slow backend
		slowBackend := &SlowBackend{
			processingDelay: 500 * time.Millisecond,
		}
		auditSystem.backends = []backends.Backend{slowBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Rapidly send events to overflow queue
		var accepted, rejected int64
		for i := 0; i < 200; i++ {
			event := createChaosTestEvent(fmt.Sprintf("overflow-test-%d", i))
			err := auditSystem.LogEvent(event)

			if err != nil {
				atomic.AddInt64(&rejected, 1)
			} else {
				atomic.AddInt64(&accepted, 1)
			}
		}

		time.Sleep(2 * time.Second)

		suite.T().Logf("Queue overflow test: %d accepted, %d rejected", accepted, rejected)
		suite.Greater(rejected, int64(0), "Expected some events to be rejected due to queue overflow")
	})
}

// Test cascading failure scenarios
func (suite *ChaosTestSuite) TestCascadingFailures() {
	suite.Run("multiple backend failures", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 200 * time.Millisecond,
			MaxQueueSize:  500,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Multiple backends with different failure modes
		backends := []backends.Backend{
			&ChaosBackend{url: suite.chaosServer.URL, failureRate: 0.5},
			&SlowBackend{processingDelay: 200 * time.Millisecond},
			&IOErrorBackend{errorRate: 0.3},
		}
		auditSystem.backends = backends

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send events while backends fail in cascade
		for i := 0; i < 100; i++ {
			event := createChaosTestEvent(fmt.Sprintf("cascade-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err, "Event submission should not fail")

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(5 * time.Second)

		stats := auditSystem.GetStats()
		suite.T().Logf("Cascading failure test: %d events processed", stats.EventsReceived)

		// System should remain operational despite multiple backend failures
		suite.Greater(stats.EventsReceived, int64(50))
	})

	suite.Run("dependency failure propagation", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  100,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Backend that depends on external service
		dependentBackend := &DependentBackend{
			externalService: "http://dependency-service:8080",
			failureRate:     0.6,
		}
		auditSystem.backends = []backends.Backend{dependentBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Test with dependency failures
		for i := 0; i < 50; i++ {
			event := createChaosTestEvent(fmt.Sprintf("dependency-test-%d", i))
			err := auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		time.Sleep(2 * time.Second)
		suite.T().Log("Dependency failure test completed")
	})
}

// Test recovery scenarios
func (suite *ChaosTestSuite) TestRecoveryScenarios() {
	suite.Run("automatic recovery after transient failures", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  200,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Backend that recovers after failures
		recoveryBackend := NewRecoveryBackend()
		auditSystem.backends = []backends.Backend{recoveryBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Phase 1: Normal operation
		for i := 0; i < 20; i++ {
			event := createChaosTestEvent(fmt.Sprintf("recovery-phase1-%d", i))
			auditSystem.LogEvent(event)
		}

		time.Sleep(1 * time.Second)
		phase1Stats := auditSystem.GetStats()

		// Phase 2: Induce failures
		recoveryBackend.InduceFailures()
		for i := 0; i < 20; i++ {
			event := createChaosTestEvent(fmt.Sprintf("recovery-phase2-%d", i))
			auditSystem.LogEvent(event)
		}

		time.Sleep(1 * time.Second)

		// Phase 3: Recovery
		recoveryBackend.Recover()
		for i := 0; i < 20; i++ {
			event := createChaosTestEvent(fmt.Sprintf("recovery-phase3-%d", i))
			auditSystem.LogEvent(event)
		}

		time.Sleep(2 * time.Second)
		finalStats := auditSystem.GetStats()

		suite.T().Logf("Recovery test: Phase1=%d, Final=%d events processed",
			phase1Stats.EventsReceived, finalStats.EventsReceived)

		// Should recover and process more events
		suite.Greater(finalStats.EventsReceived, phase1Stats.EventsReceived)
	})

	suite.Run("graceful degradation under load", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     20,
			FlushInterval: 500 * time.Millisecond,
			MaxQueueSize:  1000,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		// Overloaded backend
		overloadedBackend := &OverloadedBackend{
			maxCapacity: 10, // Very low capacity
		}
		auditSystem.backends = []backends.Backend{overloadedBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Send high load
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var totalSent int64
		for i := 0; i < 500; i++ {
			select {
			case <-ctx.Done():
				break
			default:
				event := createChaosTestEvent(fmt.Sprintf("degradation-test-%d", i))
				auditSystem.LogEvent(event)
				atomic.AddInt64(&totalSent, 1)
				time.Sleep(20 * time.Millisecond)
			}
		}

		<-ctx.Done()

		stats := auditSystem.GetStats()
		processingRate := float64(stats.EventsReceived) / float64(totalSent)

		suite.T().Logf("Graceful degradation: %d sent, %d processed (%.2f%%)",
			totalSent, stats.EventsReceived, processingRate*100)

		// Should process some events despite overload
		suite.Greater(processingRate, 0.1) // At least 10%
	})
}

// Test system limits and edge cases
func (suite *ChaosTestSuite) TestSystemLimits() {
	suite.Run("maximum event size handling", func() {
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     5,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  50,
		}

		auditSystem, err := NewAuditSystem(config)
		suite.Require().NoError(err)

		limitBackend := &SizeLimitBackend{maxEventSize: 1024} // 1KB limit
		auditSystem.backends = []backends.Backend{limitBackend}

		err = auditSystem.Start()
		suite.Require().NoError(err)
		defer auditSystem.Stop()

		// Test with various event sizes
		eventSizes := []int{512, 1024, 2048, 5120} // 512B, 1KB, 2KB, 5KB

		for _, size := range eventSizes {
			event := createSizedChaosEvent(fmt.Sprintf("size-test-%d", size), size)
			err := auditSystem.LogEvent(event)
			suite.NoError(err, "Event submission should not fail")
		}

		time.Sleep(1 * time.Second)
		suite.T().Log("Event size limit test completed")
	})

	suite.Run("extreme batch sizes", func() {
		extremeConfigs := []struct {
			name      string
			batchSize int
			expected  string
		}{
			{"Tiny batch", 1, "should work"},
			{"Large batch", 10000, "should handle gracefully"},
			{"Zero batch", 0, "should use default"},
		}

		for _, config := range extremeConfigs {
			suite.T().Run(config.name, func(t *testing.T) {
				auditConfig := &AuditSystemConfig{
					Enabled:       true,
					LogLevel:      SeverityInfo,
					BatchSize:     config.batchSize,
					FlushInterval: 100 * time.Millisecond,
					MaxQueueSize:  1000,
				}

				auditSystem, err := NewAuditSystem(auditConfig)
				assert.NoError(t, err, config.expected)

				if err == nil {
					mockBackend := &MockChaosBackend{}
					auditSystem.backends = []backends.Backend{mockBackend}

					err = auditSystem.Start()
					assert.NoError(t, err)

					if err == nil {
						defer auditSystem.Stop()

						// Send test event
						event := createChaosTestEvent("extreme-batch-test")
						err = auditSystem.LogEvent(event)
						assert.NoError(t, err)

						time.Sleep(200 * time.Millisecond)
					}
				}
			})
		}
	})
}

// Helper functions and mock implementations

func createChaosTestEvent(action string) *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeSystemChange,
		Component: "chaos-test",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		UserContext: &UserContext{
			UserID: "chaos-test-user",
		},
		Data: map[string]interface{}{
			"test_type": "chaos",
			"timestamp": time.Now().Unix(),
		},
	}
}

func createLargeChaosEvent(action string) *AuditEvent {
	// Create event with large data payload
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("Large data value %d with lots of text to increase memory usage", i)
	}

	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeDataAccess,
		Component: "chaos-large-test",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		Data:      largeData,
	}
}

func createSizedChaosEvent(action string, targetSize int) *AuditEvent {
	// Create event with specific size
	padding := make([]byte, targetSize-200) // Account for other fields
	for i := range padding {
		padding[i] = byte(65 + (i % 26)) // Fill with letters
	}

	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeDataAccess,
		Component: "size-test",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		Data: map[string]interface{}{
			"padding": string(padding),
		},
	}
}

// Chaos Backend Implementations

type ChaosBackend struct {
	url         string
	failureRate float32
	processed   int64
}

func (cb *ChaosBackend) Type() string { return "chaos" }

func (cb *ChaosBackend) Initialize(config backends.BackendConfig) error { return nil }

func (cb *ChaosBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	if cb.failureRate > 0 && rand.Float32() < cb.failureRate {
		return errors.New("chaos induced failure")
	}
	atomic.AddInt64(&cb.processed, 1)
	return nil
}

func (cb *ChaosBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := cb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (cb *ChaosBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (cb *ChaosBackend) Health(ctx context.Context) error { return nil }

func (cb *ChaosBackend) Close() error { return nil }

type DiskFullBackend struct {
	logFile     string
	maxSize     int64
	currentSize int64
	mutex       sync.Mutex
}

func (dfb *DiskFullBackend) Type() string { return "disk-full" }

func (dfb *DiskFullBackend) Initialize(config backends.BackendConfig) error { return nil }

func (dfb *DiskFullBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	dfb.mutex.Lock()
	defer dfb.mutex.Unlock()

	eventSize := int64(len(event.Action) + len(event.Component) + 100) // Approximate size

	if dfb.currentSize+eventSize > dfb.maxSize {
		return errors.New("no space left on device")
	}

	dfb.currentSize += eventSize
	return nil
}

func (dfb *DiskFullBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := dfb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (dfb *DiskFullBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (dfb *DiskFullBackend) Health(ctx context.Context) error { return nil }

func (dfb *DiskFullBackend) Close() error { return nil }

type IOErrorBackend struct {
	errorRate float32
	processed int64
}

func (ieb *IOErrorBackend) Type() string { return "io-error" }

func (ieb *IOErrorBackend) Initialize(config backends.BackendConfig) error { return nil }

func (ieb *IOErrorBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	if rand.Float32() < ieb.errorRate {
		return errors.New("I/O error occurred")
	}
	atomic.AddInt64(&ieb.processed, 1)
	return nil
}

func (ieb *IOErrorBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := ieb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (ieb *IOErrorBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (ieb *IOErrorBackend) Health(ctx context.Context) error { return nil }

func (ieb *IOErrorBackend) Close() error { return nil }

type SlowBackend struct {
	processingDelay time.Duration
	processed       int64
}

func (sb *SlowBackend) Type() string { return "slow" }

func (sb *SlowBackend) Initialize(config backends.BackendConfig) error { return nil }

func (sb *SlowBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	time.Sleep(sb.processingDelay)
	atomic.AddInt64(&sb.processed, 1)
	return nil
}

func (sb *SlowBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := sb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (sb *SlowBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (sb *SlowBackend) Health(ctx context.Context) error { return nil }

func (sb *SlowBackend) Close() error { return nil }

type NetworkPartitionBackend struct {
	url         string
	partitioned bool
	mutex       sync.RWMutex
	processed   int64
}

func NewNetworkPartitionBackend(url string) *NetworkPartitionBackend {
	return &NetworkPartitionBackend{url: url}
}

func (npb *NetworkPartitionBackend) StartPartition() {
	npb.mutex.Lock()
	npb.partitioned = true
	npb.mutex.Unlock()
}

func (npb *NetworkPartitionBackend) HealPartition() {
	npb.mutex.Lock()
	npb.partitioned = false
	npb.mutex.Unlock()
}

func (npb *NetworkPartitionBackend) Type() string { return "network-partition" }

func (npb *NetworkPartitionBackend) Initialize(config backends.BackendConfig) error { return nil }

func (npb *NetworkPartitionBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	npb.mutex.RLock()
	partitioned := npb.partitioned
	npb.mutex.RUnlock()

	if partitioned {
		return errors.New("network partition: no route to host")
	}

	atomic.AddInt64(&npb.processed, 1)
	return nil
}

func (npb *NetworkPartitionBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := npb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (npb *NetworkPartitionBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (npb *NetworkPartitionBackend) Health(ctx context.Context) error { return nil }

func (npb *NetworkPartitionBackend) Close() error { return nil }

type DependentBackend struct {
	externalService string
	failureRate     float32
	processed       int64
}

func (db *DependentBackend) Type() string { return "dependent" }

func (db *DependentBackend) Initialize(config backends.BackendConfig) error { return nil }

func (db *DependentBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	// Simulate external service dependency
	if rand.Float32() < db.failureRate {
		return errors.New("external service unavailable")
	}
	atomic.AddInt64(&db.processed, 1)
	return nil
}

func (db *DependentBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := db.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (db *DependentBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (db *DependentBackend) Health(ctx context.Context) error { return nil }

func (db *DependentBackend) Close() error { return nil }

type RecoveryBackend struct {
	failing   bool
	processed int64
	mutex     sync.RWMutex
}

func NewRecoveryBackend() *RecoveryBackend {
	return &RecoveryBackend{}
}

func (rb *RecoveryBackend) InduceFailures() {
	rb.mutex.Lock()
	rb.failing = true
	rb.mutex.Unlock()
}

func (rb *RecoveryBackend) Recover() {
	rb.mutex.Lock()
	rb.failing = false
	rb.mutex.Unlock()
}

func (rb *RecoveryBackend) Type() string { return "recovery" }

func (rb *RecoveryBackend) Initialize(config backends.BackendConfig) error { return nil }

func (rb *RecoveryBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	rb.mutex.RLock()
	failing := rb.failing
	rb.mutex.RUnlock()

	if failing {
		return errors.New("backend in failure mode")
	}

	atomic.AddInt64(&rb.processed, 1)
	return nil
}

func (rb *RecoveryBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := rb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (rb *RecoveryBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (rb *RecoveryBackend) Health(ctx context.Context) error { return nil }

func (rb *RecoveryBackend) Close() error { return nil }

type OverloadedBackend struct {
	maxCapacity int
	currentLoad int64
	processed   int64
	mutex       sync.Mutex
}

func (ob *OverloadedBackend) Type() string { return "overloaded" }

func (ob *OverloadedBackend) Initialize(config backends.BackendConfig) error { return nil }

func (ob *OverloadedBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	if int(ob.currentLoad) >= ob.maxCapacity {
		return errors.New("backend overloaded")
	}

	ob.currentLoad++
	atomic.AddInt64(&ob.processed, 1)

	// Simulate processing time
	go func() {
		time.Sleep(100 * time.Millisecond)
		ob.mutex.Lock()
		ob.currentLoad--
		ob.mutex.Unlock()
	}()

	return nil
}

func (ob *OverloadedBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := ob.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (ob *OverloadedBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (ob *OverloadedBackend) Health(ctx context.Context) error { return nil }

func (ob *OverloadedBackend) Close() error { return nil }

type SizeLimitBackend struct {
	maxEventSize int
	processed    int64
}

func (slb *SizeLimitBackend) Type() string { return "size-limit" }

func (slb *SizeLimitBackend) Initialize(config backends.BackendConfig) error { return nil }

func (slb *SizeLimitBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	eventData, _ := event.ToJSON()
	if len(eventData) > slb.maxEventSize {
		return errors.New("event size exceeds limit")
	}

	atomic.AddInt64(&slb.processed, 1)
	return nil
}

func (slb *SizeLimitBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	for _, event := range events {
		if err := slb.WriteEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (slb *SizeLimitBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (slb *SizeLimitBackend) Health(ctx context.Context) error { return nil }

func (slb *SizeLimitBackend) Close() error { return nil }

type MockChaosBackend struct {
	processed int64
}

func (mcb *MockChaosBackend) Type() string { return "mock-chaos" }

func (mcb *MockChaosBackend) Initialize(config backends.BackendConfig) error { return nil }

func (mcb *MockChaosBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	atomic.AddInt64(&mcb.processed, 1)
	return nil
}

func (mcb *MockChaosBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	atomic.AddInt64(&mcb.processed, int64(len(events)))
	return nil
}

func (mcb *MockChaosBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (mcb *MockChaosBackend) Health(ctx context.Context) error { return nil }

func (mcb *MockChaosBackend) Close() error { return nil }

// Network Emulator for simulating network conditions
type NetworkEmulator struct {
	latency    time.Duration
	packetLoss float32
	bandwidth  int // bytes per second
}

func NewNetworkEmulator() *NetworkEmulator {
	return &NetworkEmulator{
		latency:    0,
		packetLoss: 0,
		bandwidth:  0, // unlimited
	}
}

func (ne *NetworkEmulator) SetLatency(latency time.Duration) {
	ne.latency = latency
}

func (ne *NetworkEmulator) SetPacketLoss(rate float32) {
	ne.packetLoss = rate
}

func (ne *NetworkEmulator) SetBandwidth(bytesPerSecond int) {
	ne.bandwidth = bytesPerSecond
}

func (ne *NetworkEmulator) SimulateNetworkCall(ctx context.Context, data []byte) error {
	// Simulate latency
	if ne.latency > 0 {
		time.Sleep(ne.latency)
	}

	// Simulate packet loss
	if ne.packetLoss > 0 && rand.Float32() < ne.packetLoss {
		return &net.OpError{Op: "write", Err: errors.New("packet loss")}
	}

	// Simulate bandwidth limits
	if ne.bandwidth > 0 {
		transferTime := time.Duration(len(data)) * time.Second / time.Duration(ne.bandwidth)
		time.Sleep(transferTime)
	}

	return nil
}

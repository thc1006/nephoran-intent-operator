package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// Performance test configurations
const (
	SmallDatasetSize  = 1000
	MediumDatasetSize = 10000
	LargeDatasetSize  = 100000

	// Concurrency levels for testing
	LowConcurrency    = 10
	MediumConcurrency = 50
	HighConcurrency   = 200

	// Performance thresholds
	MaxAcceptableLatencyMs    = 50   // 50ms max for event submission
	MinThroughputEventsPerSec = 1000 // Minimum 1000 events/sec
	MaxMemoryUsageMB          = 500  // Max 500MB memory usage

	// Test durations
	ShortTestDuration  = 30 * time.Second
	MediumTestDuration = 2 * time.Minute
	LongTestDuration   = 5 * time.Minute
)

// BenchmarkAuditSystemThroughput measures event processing throughput
func BenchmarkAuditSystemThroughput(b *testing.B) {
	benchmarks := []struct {
		name        string
		eventCount  int
		concurrency int
		batchSize   int
	}{
		{"Small_Sequential", SmallDatasetSize, 1, 10},
		{"Small_LowConcurrency", SmallDatasetSize, LowConcurrency, 10},
		{"Medium_Sequential", MediumDatasetSize, 1, 50},
		{"Medium_MediumConcurrency", MediumDatasetSize, MediumConcurrency, 50},
		{"Large_HighConcurrency", LargeDatasetSize, HighConcurrency, 100},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			auditSystem := setupBenchmarkAuditSystem(b, bm.batchSize)
			defer auditSystem.Stop()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				benchmarkEventSubmission(b, auditSystem, bm.eventCount, bm.concurrency)
			}

			// Report throughput
			stats := auditSystem.GetStats()
			throughput := float64(stats.EventsReceived) / b.Elapsed().Seconds()
			b.ReportMetric(throughput, "events/sec")
		})
	}
}

// BenchmarkAuditSystemLatency measures event processing latency
func BenchmarkAuditSystemLatency(b *testing.B) {
	auditSystem := setupBenchmarkAuditSystem(b, 10)
	defer auditSystem.Stop()

	event := createBenchmarkEvent("latency-test")

	b.ResetTimer()
	b.ReportAllocs()

	latencies := make([]time.Duration, b.N)

	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := auditSystem.LogEvent(event)
		latencies[i] = time.Since(start)

		if err != nil {
			b.Fatal(err)
		}

		// Refresh event ID to avoid caching effects
		event.ID = uuid.New().String()
		event.Timestamp = time.Now()
	}

	// Calculate and report latency metrics
	avgLatency, p95Latency, p99Latency := calculateLatencyMetrics(latencies)
	b.ReportMetric(float64(avgLatency.Nanoseconds())/1000000, "avg_latency_ms")
	b.ReportMetric(float64(p95Latency.Nanoseconds())/1000000, "p95_latency_ms")
	b.ReportMetric(float64(p99Latency.Nanoseconds())/1000000, "p99_latency_ms")
}

// BenchmarkBatchProcessing measures batch processing performance
func BenchmarkBatchProcessing(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			auditSystem := setupBenchmarkAuditSystem(b, batchSize)
			defer auditSystem.Stop()

			events := make([]*AuditEvent, batchSize)
			for i := 0; i < batchSize; i++ {
				events[i] = createBenchmarkEvent(fmt.Sprintf("batch-event-%d", i))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for _, event := range events {
					err := auditSystem.LogEvent(event)
					if err != nil {
						b.Fatal(err)
					}
				}
			}

			// Wait for batch processing to complete
			time.Sleep(100 * time.Millisecond)

			stats := auditSystem.GetStats()
			throughput := float64(stats.EventsReceived) / b.Elapsed().Seconds()
			b.ReportMetric(throughput, "events/sec")
		})
	}
}

// BenchmarkEventSerialization measures event serialization performance
func BenchmarkEventSerialization(b *testing.B) {
	eventSizes := []struct {
		name  string
		event *AuditEvent
	}{
		{"Small", createSmallEvent()},
		{"Medium", createMediumEvent()},
		{"Large", createLargeEvent()},
		{"ExtraLarge", createExtraLargeEvent()},
	}

	for _, es := range eventSizes {
		b.Run(es.name, func(b *testing.B) {
			b.ReportAllocs()

			var totalBytes int64

			for i := 0; i < b.N; i++ {
				jsonData, err := es.event.ToJSON()
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += int64(len(jsonData))
			}

			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes_per_event")
		})
	}
}

// BenchmarkBackendPerformance measures backend-specific performance
func BenchmarkBackendPerformance(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_backend")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	backendConfigs := []struct {
		name   string
		config backends.BackendConfig
	}{
		{
			"FileBackend",
			backends.BackendConfig{
				Type:     backends.BackendTypeFile,
				Enabled:  true,
				Name:     "benchmark-file",
				Settings: map[string]interface{}{},
			},
		},
		{
			"FileBackendCompressed",
			backends.BackendConfig{
				Type:        backends.BackendTypeFile,
				Enabled:     true,
				Name:        "benchmark-compressed",
				Compression: true,
				Settings:    map[string]interface{}{},
			},
		},
	}

	events := make([]*AuditEvent, 1000)
	for i := 0; i < len(events); i++ {
		events[i] = createBenchmarkEvent(fmt.Sprintf("backend-perf-%d", i))
	}

	for _, bc := range backendConfigs {
		b.Run(bc.name, func(b *testing.B) {
			// Note: In actual implementation, you would create the backend
			// For now, we'll simulate the performance characteristics

			b.ResetTimer()
			b.ReportAllocs()

			var totalProcessed int64

			for i := 0; i < b.N; i++ {
				start := time.Now()

				// Simulate backend processing
				for _, event := range events {
					// Mock serialization and write
					jsonData, _ := event.ToJSON()
					_ = jsonData // Use the data
					atomic.AddInt64(&totalProcessed, 1)
				}

				duration := time.Since(start)
				b.ReportMetric(float64(len(events))/duration.Seconds(), "events/sec")
			}
		})
	}
}

// BenchmarkConcurrentAccess measures performance under concurrent load
func BenchmarkConcurrentAccess(b *testing.B) {
	concurrencyLevels := []int{1, 10, 50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			auditSystem := setupBenchmarkAuditSystem(b, 100)
			defer auditSystem.Stop()

			b.ResetTimer()
			b.ReportAllocs()

			var totalEvents int64

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(concurrency)

				for g := 0; g < concurrency; g++ {
					go func(goroutineID int) {
						defer wg.Done()

						event := createBenchmarkEvent(fmt.Sprintf("concurrent-%d", goroutineID))
						err := auditSystem.LogEvent(event)
						if err == nil {
							atomic.AddInt64(&totalEvents, 1)
						}
					}(g)
				}

				wg.Wait()
			}

			throughput := float64(totalEvents) / b.Elapsed().Seconds()
			b.ReportMetric(throughput, "events/sec")
		})
	}
}

// BenchmarkMemoryUsage measures memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	auditSystem := setupBenchmarkAuditSystem(b, 100)
	defer auditSystem.Stop()

	b.ResetTimer()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < b.N; i++ {
		event := createBenchmarkEvent(fmt.Sprintf("memory-test-%d", i))
		auditSystem.LogEvent(event)

		// Force allocation tracking every 1000 events
		if i%1000 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	memoryUsed := m2.Alloc - m1.Alloc
	b.ReportMetric(float64(memoryUsed)/float64(b.N), "bytes_per_event")
	b.ReportMetric(float64(m2.Alloc)/1024/1024, "total_memory_mb")
}

// BenchmarkIntegrityProcessing measures integrity protection performance impact
func BenchmarkIntegrityProcessing(b *testing.B) {
	configs := []struct {
		name      string
		integrity bool
	}{
		{"WithoutIntegrity", false},
		{"WithIntegrity", true},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			auditConfig := &AuditSystemConfig{
				Enabled:         true,
				LogLevel:        SeverityInfo,
				BatchSize:       100,
				FlushInterval:   1 * time.Second,
				MaxQueueSize:    10000,
				EnableIntegrity: config.integrity,
			}

			auditSystem, err := NewAuditSystem(auditConfig)
			require.NoError(b, err)

			// Add mock backend
			mockBackend := &MockPerformanceBackend{}
			auditSystem.backends = []backends.Backend{mockBackend}

			err = auditSystem.Start()
			require.NoError(b, err)
			defer auditSystem.Stop()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				event := createBenchmarkEvent(fmt.Sprintf("integrity-test-%d", i))
				err := auditSystem.LogEvent(event)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// StressTestAuditSystem performs stress testing under various conditions
func TestAuditSystemStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	t.Run("HighVolumeStress", func(t *testing.T) {
		auditSystem := setupBenchmarkAuditSystem(t, 1000)
		defer auditSystem.Stop()

		const duration = 5 * time.Second
		const targetTPS = 5000 // 5000 transactions per second

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		var eventsSubmitted int64
		var eventsErrored int64

		// Start multiple goroutines generating events
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				ticker := time.NewTicker(time.Second / time.Duration(targetTPS/20))
				defer ticker.Stop()

				eventID := 0
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						event := createBenchmarkEvent(fmt.Sprintf("stress-%d-%d", workerID, eventID))
						err := auditSystem.LogEvent(event)
						if err != nil {
							atomic.AddInt64(&eventsErrored, 1)
						} else {
							atomic.AddInt64(&eventsSubmitted, 1)
						}
						eventID++
					}
				}
			}(i)
		}

		wg.Wait()

		// Wait for processing to complete
		time.Sleep(5 * time.Second)

		stats := auditSystem.GetStats()
		actualTPS := float64(eventsSubmitted) / duration.Seconds()
		errorRate := float64(eventsErrored) / float64(eventsSubmitted+eventsErrored)

		t.Logf("Stress test results:")
		t.Logf("  Target TPS: %d", targetTPS)
		t.Logf("  Actual TPS: %.2f", actualTPS)
		t.Logf("  Events submitted: %d", eventsSubmitted)
		t.Logf("  Events errored: %d", eventsErrored)
		t.Logf("  Error rate: %.2f%%", errorRate*100)
		t.Logf("  Events processed: %d", stats.EventsReceived)
		t.Logf("  Events dropped: %d", stats.EventsDropped)

		// Assertions
		require.Greater(t, actualTPS, float64(targetTPS)*0.8, "TPS too low")
		require.Less(t, errorRate, 0.05, "Error rate too high") // Less than 5% errors
		require.Less(t, float64(stats.EventsDropped)/float64(stats.EventsReceived), 0.1, "Too many dropped events")
	})

	t.Run("MemoryLeakTest", func(t *testing.T) {
		auditSystem := setupBenchmarkAuditSystem(t, 50)
		defer auditSystem.Stop()

		var m1, m2 runtime.MemStats

		// Baseline measurement
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Generate events for extended period
		for round := 0; round < 2; round++ {
			for i := 0; i < 10000; i++ {
				event := createBenchmarkEvent(fmt.Sprintf("leak-test-%d-%d", round, i))
				auditSystem.LogEvent(event)
			}

			// Wait for processing
			time.Sleep(1 * time.Second)

			// Force GC
			runtime.GC()
		}

		// Final measurement
		runtime.ReadMemStats(&m2)

		memoryGrowth := int64(m2.Alloc) - int64(m1.Alloc)
		memoryGrowthMB := float64(memoryGrowth) / 1024 / 1024

		t.Logf("Memory usage:")
		t.Logf("  Initial: %.2f MB", float64(m1.Alloc)/1024/1024)
		t.Logf("  Final: %.2f MB", float64(m2.Alloc)/1024/1024)
		t.Logf("  Growth: %.2f MB", memoryGrowthMB)

		// Memory growth should be reasonable (less than 100MB)
		require.Less(t, memoryGrowthMB, 100.0, "Excessive memory growth detected")
	})

	t.Run("BackpressureHandling", func(t *testing.T) {
		// Create system with very small queue to test backpressure
		config := &AuditSystemConfig{
			Enabled:       true,
			LogLevel:      SeverityInfo,
			BatchSize:     10,
			FlushInterval: 100 * time.Millisecond,
			MaxQueueSize:  100, // Small queue
		}

		auditSystem, err := NewAuditSystem(config)
		require.NoError(t, err)

		// Add slow backend to create backpressure
		slowBackend := &SlowMockBackend{latency: 10 * time.Millisecond}
		auditSystem.backends = []backends.Backend{slowBackend}

		err = auditSystem.Start()
		require.NoError(t, err)
		defer auditSystem.Stop()

		// Rapidly submit events to fill queue
		var submitted, errored int64
		for i := 0; i < 1000; i++ {
			event := createBenchmarkEvent(fmt.Sprintf("backpressure-%d", i))
			err := auditSystem.LogEvent(event)
			if err != nil {
				atomic.AddInt64(&errored, 1)
			} else {
				atomic.AddInt64(&submitted, 1)
			}
		}

		t.Logf("Backpressure results:")
		t.Logf("  Submitted: %d", submitted)
		t.Logf("  Errored: %d", errored)
		t.Logf("  Error rate: %.2f%%", float64(errored)/1000*100)

		// System should handle backpressure gracefully - with queue size 100,
		// some events should be accepted but many may be rejected
		require.Greater(t, submitted, int64(50), "Too many events rejected")
		require.Greater(t, errored, int64(0), "Expected some backpressure errors")
	})
}

// LoadTestAuditSystem performs realistic load testing
func TestAuditSystemLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	scenarios := []struct {
		name           string
		duration       time.Duration
		eventsPerSec   int
		concurrency    int
		expectedMinTPS float64
	}{
		{"LowLoad", 3 * time.Second, 100, 5, 90},
		{"MediumLoad", 6 * time.Second, 1000, 20, 900},
		{"HighLoad", 3 * time.Second, 5000, 50, 4000},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			auditSystem := setupBenchmarkAuditSystem(t, 100)
			defer auditSystem.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), scenario.duration)
			defer cancel()

			var totalEvents int64
			var wg sync.WaitGroup

			// Start worker goroutines
			for i := 0; i < scenario.concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					eventsPerWorker := scenario.eventsPerSec / scenario.concurrency
					ticker := time.NewTicker(time.Second / time.Duration(eventsPerWorker))
					defer ticker.Stop()

					eventID := 0
					for {
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							event := createBenchmarkEvent(fmt.Sprintf("load-%s-%d-%d", scenario.name, workerID, eventID))
							err := auditSystem.LogEvent(event)
							if err == nil {
								atomic.AddInt64(&totalEvents, 1)
							}
							eventID++
						}
					}
				}(i)
			}

			wg.Wait()

			// Allow time for processing
			time.Sleep(2 * time.Second)

			actualTPS := float64(totalEvents) / scenario.duration.Seconds()
			stats := auditSystem.GetStats()

			t.Logf("Load test %s results:", scenario.name)
			t.Logf("  Target TPS: %d", scenario.eventsPerSec)
			t.Logf("  Actual TPS: %.2f", actualTPS)
			t.Logf("  Events processed: %d", stats.EventsReceived)
			t.Logf("  Events dropped: %d", stats.EventsDropped)
			t.Logf("  Drop rate: %.2f%%", float64(stats.EventsDropped)/float64(stats.EventsReceived)*100)

			require.Greater(t, actualTPS, scenario.expectedMinTPS, "TPS below expected minimum")
		})
	}
}

// Helper functions for performance testing

func setupBenchmarkAuditSystem(t testing.TB, batchSize int) *AuditSystem {
	config := &AuditSystemConfig{
		Enabled:         true,
		LogLevel:        SeverityInfo,
		BatchSize:       batchSize,
		FlushInterval:   100 * time.Millisecond,
		MaxQueueSize:    10000,
		EnableIntegrity: false, // Disabled for performance
	}

	auditSystem, err := NewAuditSystem(config)
	require.NoError(t, err)

	// Add mock backend for performance testing
	mockBackend := &MockPerformanceBackend{}
	auditSystem.backends = []backends.Backend{mockBackend}

	err = auditSystem.Start()
	require.NoError(t, err)

	return auditSystem
}

func createBenchmarkEvent(action string) *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeAPICall,
		Component: "benchmark",
		Action:    action,
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		UserContext: &UserContext{
			UserID: "benchmark-user",
		},
	}
}

func createSmallEvent() *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeAPICall,
		Component: "small-test",
		Action:    "test",
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
	}
}

func createMediumEvent() *AuditEvent {
	return &AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: EventTypeDataAccess,
		Component: "medium-test",
		Action:    "access_data",
		Severity:  SeverityInfo,
		Result:    ResultSuccess,
		UserContext: &UserContext{
			UserID:   "medium-user",
			Username: "mediumuser",
			Role:     "operator",
			Groups:   []string{"group1", "group2"},
		},
		NetworkContext: &NetworkContext{
			SourcePort:      8080,
			DestinationPort: 443,
			Protocol:        "https",
			UserAgent:       "test-agent/1.0",
		},
		ResourceContext: &ResourceContext{
			ResourceType: "data",
			ResourceID:   "data123",
			Operation:    "read",
		},
		Data: map[string]interface{}{},
	}
}

func createLargeEvent() *AuditEvent {
	// Create large data payload
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_text_content", i)
	}

	return &AuditEvent{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		EventType:   EventTypeSystemChange,
		Component:   "large-test",
		Action:      "bulk_operation",
		Description: "This is a large event with extensive data payload for performance testing purposes",
		Severity:    SeverityInfo,
		Result:      ResultSuccess,
		UserContext: &UserContext{
			UserID:       "large-user",
			Username:     "largeuser",
			Role:         "administrator",
			Groups:       []string{"admin", "power-users", "auditors"},
			Permissions:  []string{"read", "write", "delete", "admin"},
			AuthMethod:   "oauth2",
			AuthProvider: "corporate-sso",
		},
		NetworkContext: &NetworkContext{
			SourcePort:      8080,
			DestinationPort: 443,
			Protocol:        "https",
			UserAgent:       "enterprise-client/2.1.0",
			Referrer:        "https://dashboard.company.com",
			Country:         "US",
			ASN:             "AS1234",
		},
		ResourceContext: &ResourceContext{
			ResourceType: "bulk-data",
			ResourceID:   "bulk-operation-12345",
			Operation:    "batch_update",
			Namespace:    "production",
			APIVersion:   "v1",
		},
		Data: largeData,
	}
}

func createExtraLargeEvent() *AuditEvent {
	// Create extra large data payload
	extraLargeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		extraLargeData[fmt.Sprintf("field_%d", i)] = map[string]interface{}{
			"payload":     json.RawMessage(`{}`),
			"nested_data": []string{"item1", "item2", "item3"},
			"timestamp":   time.Now(),
		}
	}

	return &AuditEvent{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		EventType:   EventTypeSystemChange,
		Component:   "extra-large-test",
		Action:      "massive_bulk_operation",
		Description: "This is an extra large event with massive data payload for stress testing the audit system under extreme conditions",
		Severity:    SeverityInfo,
		Result:      ResultSuccess,
		UserContext: &UserContext{
			UserID:   "xl-user",
			Username: "xlargeuser",
		},
		Data: extraLargeData,
	}
}

func benchmarkEventSubmission(t testing.TB, auditSystem *AuditSystem, eventCount, concurrency int) {
	if concurrency == 1 {
		// Sequential processing
		for i := 0; i < eventCount; i++ {
			event := createBenchmarkEvent(fmt.Sprintf("seq-event-%d", i))
			err := auditSystem.LogEvent(event)
			if err != nil {
				t.Fatal(err)
			}
		}
	} else {
		// Concurrent processing
		var wg sync.WaitGroup
		eventsPerGoroutine := eventCount / concurrency

		for g := 0; g < concurrency; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for i := 0; i < eventsPerGoroutine; i++ {
					event := createBenchmarkEvent(fmt.Sprintf("conc-event-%d-%d", goroutineID, i))
					err := auditSystem.LogEvent(event)
					if err != nil {
						// Don't fail the benchmark on individual errors in concurrent mode
						continue
					}
				}
			}(g)
		}

		wg.Wait()
	}

	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)
}

func calculateLatencyMetrics(latencies []time.Duration) (avg, p95, p99 time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Calculate average
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	avg = total / time.Duration(len(latencies))

	// Calculate percentiles
	p95Index := int(0.95 * float64(len(sorted)))
	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	p95 = sorted[p95Index]

	p99Index := int(0.99 * float64(len(sorted)))
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}
	p99 = sorted[p99Index]

	return avg, p95, p99
}

// Mock backends for performance testing

type MockPerformanceBackend struct {
	processedEvents int64
}

func (m *MockPerformanceBackend) Type() string {
	return "mock-performance"
}

func (m *MockPerformanceBackend) Initialize(config backends.BackendConfig) error {
	return nil
}

func (m *MockPerformanceBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	atomic.AddInt64(&m.processedEvents, 1)
	return nil
}

func (m *MockPerformanceBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	atomic.AddInt64(&m.processedEvents, int64(len(events)))
	return nil
}

func (m *MockPerformanceBackend) Query(ctx context.Context, query *backends.QueryRequest) (*backends.QueryResponse, error) {
	return &backends.QueryResponse{}, nil
}

func (m *MockPerformanceBackend) Health(ctx context.Context) error {
	return nil
}

func (m *MockPerformanceBackend) Close() error {
	return nil
}

type SlowMockBackend struct {
	MockPerformanceBackend
	latency time.Duration
}

func (s *SlowMockBackend) WriteEvent(ctx context.Context, event *AuditEvent) error {
	time.Sleep(s.latency)
	return s.MockPerformanceBackend.WriteEvent(ctx, event)
}

func (s *SlowMockBackend) WriteEvents(ctx context.Context, events []*AuditEvent) error {
	time.Sleep(s.latency * time.Duration(len(events)))
	return s.MockPerformanceBackend.WriteEvents(ctx, events)
}

// Performance test results analysis

type PerformanceResults struct {
	TestName         string
	EventsPerSecond  float64
	AverageLatencyMs float64
	P95LatencyMs     float64
	P99LatencyMs     float64
	MemoryUsageMB    float64
	ErrorRate        float64
	PassedThresholds bool
}

func analyzePerformanceResults(results *PerformanceResults) {
	fmt.Printf("Performance Analysis for %s:\n", results.TestName)
	fmt.Printf("  Throughput: %.2f events/sec (threshold: %d)\n", results.EventsPerSecond, MinThroughputEventsPerSec)
	fmt.Printf("  Average Latency: %.2f ms (threshold: %d ms)\n", results.AverageLatencyMs, MaxAcceptableLatencyMs)
	fmt.Printf("  P95 Latency: %.2f ms\n", results.P95LatencyMs)
	fmt.Printf("  P99 Latency: %.2f ms\n", results.P99LatencyMs)
	fmt.Printf("  Memory Usage: %.2f MB (threshold: %d MB)\n", results.MemoryUsageMB, MaxMemoryUsageMB)
	fmt.Printf("  Error Rate: %.2f%%\n", results.ErrorRate*100)

	// Check thresholds
	thresholdsPassed := true
	if results.EventsPerSecond < MinThroughputEventsPerSec {
		fmt.Printf("  ??Throughput below threshold\n")
		thresholdsPassed = false
	}
	if results.AverageLatencyMs > MaxAcceptableLatencyMs {
		fmt.Printf("  ??Average latency above threshold\n")
		thresholdsPassed = false
	}
	if results.MemoryUsageMB > MaxMemoryUsageMB {
		fmt.Printf("  ??Memory usage above threshold\n")
		thresholdsPassed = false
	}

	if thresholdsPassed {
		fmt.Printf("  ??All performance thresholds passed\n")
	}

	results.PassedThresholds = thresholdsPassed
}

// Additional specialized performance tests

func BenchmarkAuditEventValidation(b *testing.B) {
	validEvent := createBenchmarkEvent("validation-test")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := validEvent.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuditEventEnrichment(b *testing.B) {
	auditSystem := setupBenchmarkAuditSystem(b, 10)
	defer auditSystem.Stop()

	event := createBenchmarkEvent("enrichment-test")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		auditSystem.enrichEvent(event)
	}
}

func BenchmarkComplianceMetadataGeneration(b *testing.B) {
	auditSystem := setupBenchmarkAuditSystem(b, 10)
	defer auditSystem.Stop()

	// Enable compliance mode
	auditSystem.config.ComplianceMode = []ComplianceStandard{
		ComplianceSOC2,
		ComplianceISO27001,
		CompliancePCIDSS,
	}

	event := createBenchmarkEvent("compliance-test")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		auditSystem.enrichEvent(event)
	}
}

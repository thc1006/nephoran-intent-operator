package monitoring

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/testing/racetest"
)

// TestMetricsCollectorRaceConditions tests concurrent metrics collection
func TestMetricsCollectorRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 100,
		Iterations: 100,
		Timeout:    10 * time.Second,
	})

	collector := &metricsCollector{
		mu:          sync.RWMutex{},
		metrics:     make(map[string]*testMetric),
		aggregates:  &sync.Map{},
		updateCount: atomic.Int64{},
		errorCount:  atomic.Int64{},
	}

	runner.RunConcurrent(func(id int) error {
		metricName := fmt.Sprintf("metric-%d", id%20)
		value := float64(id)

		// Update metric
		collector.mu.Lock()
		if m, exists := collector.metrics[metricName]; exists {
			// Update existing metric
			atomic.AddInt64(&m.count, 1)
			// Use atomic for float64 (bit pattern)
			for {
				old := atomic.LoadUint64(&m.sumBits)
				new := math.Float64bits(math.Float64frombits(old) + value)
				if atomic.CompareAndSwapUint64(&m.sumBits, old, new) {
					break
				}
			}
		} else {
			// Create new metric
			collector.metrics[metricName] = &testMetric{
				name:    metricName,
				count:   1,
				sumBits: math.Float64bits(value),
			}
		}
		collector.updateCount.Add(1)
		collector.mu.Unlock()

		// Store in aggregates
		collector.aggregates.Store(metricName, value)

		// Read metrics periodically
		if id%10 == 0 {
			collector.mu.RLock()
			for _, m := range collector.metrics {
				count := atomic.LoadInt64(&m.count)
				sum := math.Float64frombits(atomic.LoadUint64(&m.sumBits))
				if count > 0 {
					_ = sum / float64(count) // Calculate average
				}
			}
			collector.mu.RUnlock()
		}

		return nil
	})

	t.Logf("Updates: %d, Errors: %d, Metrics: %d",
		collector.updateCount.Load(), collector.errorCount.Load(),
		len(collector.metrics))
}

// TestHealthCheckerDeadlockDetection tests for deadlocks in health checking
func TestHealthCheckerDeadlockDetection(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	healthChecker := &healthCheckSystem{
		mu:            sync.RWMutex{},
		checks:        make(map[string]*healthCheck),
		dependencies:  make(map[string][]string),
		statusCache:   &sync.Map{},
		checkCount:    atomic.Int64{},
		deadlockCount: atomic.Int64{},
	}

	// Setup circular dependency for deadlock detection
	healthChecker.dependencies["service-a"] = []string{"service-b"}
	healthChecker.dependencies["service-b"] = []string{"service-c"}
	healthChecker.dependencies["service-c"] = []string{"service-a"} // Circular!

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deadlock detector
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if detectDeadlock(healthChecker) {
					healthChecker.deadlockCount.Add(1)
					t.Log("Potential deadlock detected in health checks")
				}
			}
		}
	}()

	// Concurrent health checks
	for i := 0; i < 30; i++ {
		serviceID := fmt.Sprintf("service-%c", 'a'+i%3)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Check service health with dependency resolution
					checkServiceHealth(healthChecker, serviceID, make(map[string]bool))
					healthChecker.checkCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("Health checks: %d, Deadlocks detected: %d",
		healthChecker.checkCount.Load(), healthChecker.deadlockCount.Load())
}

// TestPrometheusExporterRace tests Prometheus metrics exporter concurrency
func TestPrometheusExporterRace(t *testing.T) {
	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 100,
		Timeout:    10 * time.Second,
	})

	exporter := &prometheusExporter{
		mu:         sync.RWMutex{},
		counters:   make(map[string]*atomic.Int64),
		gauges:     make(map[string]*atomic.Uint64),
		histograms: &sync.Map{},
		scraped:    atomic.Int64{},
	}

	runner.RunConcurrent(func(id int) error {
		metricType := id % 3
		metricName := fmt.Sprintf("metric_%d", id%10)

		switch metricType {
		case 0: // Counter
			exporter.mu.Lock()
			if _, exists := exporter.counters[metricName]; !exists {
				exporter.counters[metricName] = &atomic.Int64{}
			}
			counter := exporter.counters[metricName]
			exporter.mu.Unlock()
			counter.Add(1)

		case 1: // Gauge
			exporter.mu.Lock()
			if _, exists := exporter.gauges[metricName]; !exists {
				exporter.gauges[metricName] = &atomic.Uint64{}
			}
			gauge := exporter.gauges[metricName]
			exporter.mu.Unlock()
			gauge.Store(uint64(id))

		case 2: // Histogram
			observations := []float64{}
			if val, ok := exporter.histograms.Load(metricName); ok {
				observations = val.([]float64)
			}
			observations = append(observations, float64(id))
			exporter.histograms.Store(metricName, observations)
		}

		// Simulate scrape
		if id%20 == 0 {
			exporter.mu.RLock()
			for _, counter := range exporter.counters {
				_ = counter.Load()
			}
			for _, gauge := range exporter.gauges {
				_ = gauge.Load()
			}
			exporter.mu.RUnlock()

			exporter.histograms.Range(func(key, value interface{}) bool {
				// Process histogram
				return true
			})

			exporter.scraped.Add(1)
		}

		return nil
	})

	t.Logf("Counters: %d, Gauges: %d, Scrapes: %d",
		len(exporter.counters), len(exporter.gauges), exporter.scraped.Load())
}

// TestDistributedTracingRace tests distributed tracing with concurrent spans
func TestDistributedTracingRace(t *testing.T) {
	atomicTest := racetest.NewAtomicRaceTest(t)

	tracer := &distributedTracer{
		spans:       &sync.Map{},
		spanCounter: atomic.Int64{},
		traces:      make(map[string]*testTrace),
		traceMu:     sync.RWMutex{},
	}

	// Test atomic span counter
	atomicTest.TestCompareAndSwap(&tracer.spanCounter)

	runner := racetest.NewRunner(t, racetest.DefaultConfig())

	var created, finished atomic.Int64

	runner.RunConcurrent(func(id int) error {
		traceID := fmt.Sprintf("trace-%d", id%10)
		spanID := tracer.spanCounter.Add(1)

		// Create span
		span := &span{
			traceID:   traceID,
			spanID:    spanID,
			startTime: time.Now().UnixNano(),
			tags:      &sync.Map{},
		}

		tracer.spans.Store(spanID, span)
		created.Add(1)

		// Add to trace
		tracer.traceMu.Lock()
		if trace, exists := tracer.traces[traceID]; exists {
			trace.mu.Lock()
			trace.spans = append(trace.spans, spanID)
			trace.mu.Unlock()
		} else {
			tracer.traces[traceID] = &testTrace{
				id:    traceID,
				spans: []int64{spanID},
			}
		}
		tracer.traceMu.Unlock()

		// Add tags
		span.tags.Store("operation", fmt.Sprintf("op-%d", id))
		span.tags.Store("service", fmt.Sprintf("svc-%d", id%5))

		// Finish span
		atomic.StoreInt64(&span.endTime, time.Now().UnixNano())
		finished.Add(1)

		// Occasionally read trace
		if id%10 == 0 {
			tracer.traceMu.RLock()
			if trace := tracer.traces[traceID]; trace != nil {
				trace.mu.RLock()
				_ = len(trace.spans)
				trace.mu.RUnlock()
			}
			tracer.traceMu.RUnlock()
		}

		return nil
	})

	t.Logf("Created: %d, Finished: %d, Traces: %d",
		created.Load(), finished.Load(), len(tracer.traces))
}

// TestAlertManagerConcurrency tests alert manager with concurrent alerts
func TestAlertManagerConcurrency(t *testing.T) {
	channelTest := racetest.NewChannelRaceTest(t)

	alertChan := make(chan interface{}, 100)
	channelTest.TestConcurrentSendReceive(alertChan, 20, 10)

	alertManager := &alertManager{
		mu:           sync.RWMutex{},
		activeAlerts: make(map[string]*alert),
		alertQueue:   make(chan *alert, 100),
		processed:    atomic.Int64{},
		suppressed:   atomic.Int64{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Alert processor
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case alert := <-alertManager.alertQueue:
				alertManager.mu.Lock()
				if existing, ok := alertManager.activeAlerts[alert.id]; ok {
					// Update existing alert
					atomic.AddInt64(&existing.count, 1)
					alertManager.suppressed.Add(1)
				} else {
					// New alert
					alertManager.activeAlerts[alert.id] = alert
					alertManager.processed.Add(1)
				}
				alertManager.mu.Unlock()
			}
		}
	}()

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 100,
		Timeout:    5 * time.Second,
	})

	runner.RunConcurrent(func(id int) error {
		alertID := fmt.Sprintf("alert-%d", id%20)
		alert := &alert{
			id:        alertID,
			severity:  id % 3,
			timestamp: time.Now().UnixNano(),
			count:     1,
		}

		select {
		case alertManager.alertQueue <- alert:
			// Success
		default:
			// Queue full
		}

		// Check active alerts
		if id%10 == 0 {
			alertManager.mu.RLock()
			activeCount := len(alertManager.activeAlerts)
			alertManager.mu.RUnlock()
			_ = activeCount
		}

		return nil
	})

	cancel()
	time.Sleep(100 * time.Millisecond) // Let processor finish

	t.Logf("Processed: %d, Suppressed: %d, Active: %d",
		alertManager.processed.Load(), alertManager.suppressed.Load(),
		len(alertManager.activeAlerts))
}

// TestSLAMonitoringRace tests SLA monitoring with concurrent updates
func TestSLAMonitoringRace(t *testing.T) {
	mutexTest := racetest.NewMutexRaceTest(t)

	slaMonitor := &slaMonitor{
		mu:         sync.RWMutex{},
		metrics:    make(map[string]*slaMetric),
		violations: atomic.Int64{},
		checks:     atomic.Int64{},
	}

	// Create a simple map for testing critical sections
	testCounters := make(map[string]int)
	mutexTest.TestCriticalSection(&slaMonitor.mu, &testCounters)

	runner := racetest.NewRunner(t, racetest.DefaultConfig())

	runner.RunConcurrent(func(id int) error {
		service := fmt.Sprintf("service-%d", id%10)
		latency := time.Duration(id) * time.Millisecond
		threshold := 100 * time.Millisecond

		// Update SLA metric
		slaMonitor.mu.Lock()
		if metric, exists := slaMonitor.metrics[service]; exists {
			atomic.AddInt64(&metric.requests, 1)
			// Update latency using CAS
			for {
				old := atomic.LoadInt64(&metric.totalLatency)
				new := old + int64(latency)
				if atomic.CompareAndSwapInt64(&metric.totalLatency, old, new) {
					break
				}
			}
		} else {
			slaMonitor.metrics[service] = &slaMetric{
				service:      service,
				requests:     1,
				totalLatency: int64(latency),
				threshold:    int64(threshold),
			}
		}
		slaMonitor.mu.Unlock()

		// Check SLA violation
		slaMonitor.mu.RLock()
		if metric := slaMonitor.metrics[service]; metric != nil {
			requests := atomic.LoadInt64(&metric.requests)
			totalLatency := atomic.LoadInt64(&metric.totalLatency)
			if requests > 0 {
				avgLatency := time.Duration(totalLatency / requests)
				if avgLatency > threshold {
					slaMonitor.violations.Add(1)
				}
			}
		}
		slaMonitor.checks.Add(1)
		slaMonitor.mu.RUnlock()

		return nil
	})

	t.Logf("Checks: %d, Violations: %d, Services: %d",
		slaMonitor.checks.Load(), slaMonitor.violations.Load(),
		len(slaMonitor.metrics))
}

// BenchmarkMonitoringConcurrency benchmarks monitoring operations
func BenchmarkMonitoringConcurrency(b *testing.B) {
	metrics := &sync.Map{}
	var updates atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("metric-%d", updates.Load()%100)
			value := float64(updates.Add(1))

			// Update metric
			if current, ok := metrics.Load(key); ok {
				// Aggregate
				if m, ok := current.(*atomic.Uint64); ok {
					for {
						old := m.Load()
						new := math.Float64bits(math.Float64frombits(old) + value)
						if m.CompareAndSwap(old, new) {
							break
						}
					}
				}
			} else {
				m := &atomic.Uint64{}
				m.Store(math.Float64bits(value))
				metrics.Store(key, m)
			}
		}
	})

	b.Logf("Total updates: %d", updates.Load())
}

// Helper functions and types
type metricsCollector struct {
	mu          sync.RWMutex
	metrics     map[string]*testMetric
	aggregates  *sync.Map
	updateCount atomic.Int64
	errorCount  atomic.Int64
}

type testMetric struct {
	name    string
	count   int64
	sumBits uint64 // Float64 as bits for atomic operations
}

type healthCheckSystem struct {
	mu            sync.RWMutex
	checks        map[string]*healthCheck
	dependencies  map[string][]string
	statusCache   *sync.Map
	checkCount    atomic.Int64
	deadlockCount atomic.Int64
}

type healthCheck struct {
	service string
	status  int32 // 0=unknown, 1=healthy, 2=unhealthy
	lastRun int64
}

type prometheusExporter struct {
	mu         sync.RWMutex
	counters   map[string]*atomic.Int64
	gauges     map[string]*atomic.Uint64
	histograms *sync.Map
	scraped    atomic.Int64
}

type distributedTracer struct {
	spans       *sync.Map
	spanCounter atomic.Int64
	traces      map[string]*testTrace
	traceMu     sync.RWMutex
}

type span struct {
	traceID   string
	spanID    int64
	startTime int64
	endTime   int64
	tags      *sync.Map
}

type testTrace struct {
	id    string
	spans []int64
	mu    sync.RWMutex
}

type alertManager struct {
	mu           sync.RWMutex
	activeAlerts map[string]*alert
	alertQueue   chan *alert
	processed    atomic.Int64
	suppressed   atomic.Int64
}

type alert struct {
	id        string
	severity  int
	timestamp int64
	count     int64
}

type slaMonitor struct {
	mu         sync.RWMutex
	metrics    map[string]*slaMetric
	violations atomic.Int64
	checks     atomic.Int64
}

type slaMetric struct {
	service      string
	requests     int64
	totalLatency int64
	threshold    int64
}

func detectDeadlock(hc *healthCheckSystem) bool {
	// Simple cycle detection in dependency graph
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	hc.mu.RLock()
	defer hc.mu.RUnlock()

	for service := range hc.dependencies {
		if !visited[service] {
			if hasCycle(service, hc.dependencies, visited, recStack) {
				return true
			}
		}
	}
	return false
}

func hasCycle(node string, deps map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, dep := range deps[node] {
		if !visited[dep] {
			if hasCycle(dep, deps, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[node] = false
	return false
}

func checkServiceHealth(hc *healthCheckSystem, service string, checking map[string]bool) {
	if checking[service] {
		return // Circular dependency
	}
	checking[service] = true

	hc.mu.RLock()
	deps := hc.dependencies[service]
	hc.mu.RUnlock()

	// Check dependencies first
	for _, dep := range deps {
		checkServiceHealth(hc, dep, checking)
	}

	// Update health status
	hc.mu.Lock()
	if check, exists := hc.checks[service]; exists {
		atomic.StoreInt32(&check.status, 1) // Healthy
		atomic.StoreInt64(&check.lastRun, time.Now().UnixNano())
	} else {
		hc.checks[service] = &healthCheck{
			service: service,
			status:  1,
			lastRun: time.Now().UnixNano(),
		}
	}
	hc.mu.Unlock()

	// Cache result
	hc.statusCache.Store(service, true)
}

package performance_tests

import (
<<<<<<< HEAD
	
	"encoding/json"
"time"
=======
	"context"
	"encoding/json"
	"time"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
)

// ResourceUsageMetrics contains resource usage statistics
type ResourceUsageMetrics struct {
<<<<<<< HEAD
	CPUUsage    float64                `json:"cpu_usage"`
	MemoryUsage float64                `json:"memory_usage"`
	DiskIO      float64                `json:"disk_io"`
	NetworkIO   float64                `json:"network_io"`
	Timestamp   time.Time              `json:"timestamp"`
=======
	CPUUsage    float64         `json:"cpu_usage"`
	MemoryUsage float64         `json:"memory_usage"`
	DiskIO      float64         `json:"disk_io"`
	NetworkIO   float64         `json:"network_io"`
	Timestamp   time.Time       `json:"timestamp"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	Metadata    json.RawMessage `json:"metadata"`
}

// TimePointMetric represents a point in time with a value (avoiding redeclaration)
type TimePointMetric struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
<<<<<<< HEAD
=======

// BasicTestConfig holds basic configuration for simple performance tests
type BasicTestConfig struct {
	TestDuration       time.Duration `json:"test_duration"`
	ConcurrencyLevel   int           `json:"concurrency_level"`
	RequestsPerSecond  int           `json:"requests_per_second"`
	MetricsInterval    time.Duration `json:"metrics_interval"`
	ResourceMonitoring bool          `json:"resource_monitoring"`
}

// BasicLoadGenerator handles simple load generation for tests
type BasicLoadGenerator struct {
	config    *BasicTestConfig
	isRunning bool
}

func (lg *BasicLoadGenerator) Start() error {
	lg.isRunning = true
	return nil
}

func (lg *BasicLoadGenerator) Stop() error {
	lg.isRunning = false
	return nil
}

// BasicPerformanceProfiler handles simple performance profiling
type BasicPerformanceProfiler struct {
	isRunning bool
}

func (pp *BasicPerformanceProfiler) Start() error {
	pp.isRunning = true
	return nil
}

func (pp *BasicPerformanceProfiler) Stop() error {
	pp.isRunning = false
	return nil
}

// BasicResourceMonitor monitors system resources
type BasicResourceMonitor struct {
	isRunning bool
}

func (rm *BasicResourceMonitor) Start() error {
	rm.isRunning = true
	return nil
}

func (rm *BasicResourceMonitor) Stop() error {
	rm.isRunning = false
	return nil
}

// BasicTestResults holds simple test results
type BasicTestResults struct {
	Duration      time.Duration              `json:"duration"`
	TotalRequests int64                      `json:"total_requests"`
	SuccessRate   float64                    `json:"success_rate"`
	AvgLatency    time.Duration              `json:"avg_latency"`
	ResourceUsage []ResourceUsageMetrics     `json:"resource_usage"`
	Metrics       map[string]TimePointMetric `json:"metrics"`
}

// BasicRealtimeMetrics tracks real-time metrics
type BasicRealtimeMetrics struct {
	isRunning bool
}

func (rm *BasicRealtimeMetrics) Start(ctx context.Context) error {
	rm.isRunning = true
	return nil
}

func (rm *BasicRealtimeMetrics) Stop() error {
	rm.isRunning = false
	return nil
}

// BasicLatencyRecorder records latency metrics
type BasicLatencyRecorder struct {
	measurements []time.Duration
}

func NewBasicLatencyRecorder() *BasicLatencyRecorder {
	return &BasicLatencyRecorder{
		measurements: make([]time.Duration, 0),
	}
}

func (lr *BasicLatencyRecorder) Record(latency time.Duration) {
	lr.measurements = append(lr.measurements, latency)
}

func NewBasicRealtimeMetrics() *BasicRealtimeMetrics {
	return &BasicRealtimeMetrics{}
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

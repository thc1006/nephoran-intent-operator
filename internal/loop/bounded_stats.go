package loop

import (
	"sync"
	"sync/atomic"
	"time"
)

// BoundedStats implements memory-efficient statistics collection with rolling windows
type BoundedStats struct {
	mu sync.RWMutex

	// Counters (atomic for lock-free access)
	totalProcessed atomic.Int64
	successCount   atomic.Int64
	failureCount   atomic.Int64
	timeoutCount   atomic.Int64

	// Rolling windows for time-series data
	processingTimes  *RingBuffer
	queueDepths      *RingBuffer
	throughputWindow *ThroughputWindow

	// Aggregated metrics (updated periodically)
	aggregates      *AggregatedMetrics
	lastAggregation time.Time

	// Memory-efficient file tracking
	recentFiles  *BoundedFileList
	errorSummary *ErrorSummary

	// Resource metrics
	memorySnapshots *RingBuffer
	cpuSnapshots    *RingBuffer
}

// AggregatedMetrics holds pre-computed aggregate values
type AggregatedMetrics struct {
	AvgProcessingTime time.Duration
	P50ProcessingTime time.Duration
	P95ProcessingTime time.Duration
	P99ProcessingTime time.Duration
	AvgQueueDepth     float64
	MaxQueueDepth     int
	Throughput        float64 // items per second
	SuccessRate       float64
	LastUpdate        time.Time
}

// ThroughputWindow tracks throughput over a sliding window
type ThroughputWindow struct {
	mu         sync.Mutex
	buckets    []int64
	bucketSize time.Duration
	windowSize int
	currentIdx int
	lastUpdate time.Time
}

// BoundedFileList maintains a bounded list of recent files
type BoundedFileList struct {
	mu      sync.RWMutex
	files   []FileInfo
	maxSize int
	current int
}

// FileInfo stores minimal file information
type FileInfo struct {
	Name      string // Just filename, not full path
	Size      int64
	Timestamp time.Time
	Status    string
}

// ErrorSummary tracks error counts by type
type ErrorSummary struct {
	mu      sync.RWMutex
	counts  map[string]int64
	maxKeys int
}

// RingBuffer implements a fixed-size circular buffer for numeric values
type RingBuffer struct {
	mu       sync.RWMutex
	data     []float64
	size     int
	position int
	count    int
}

// BoundedStatsConfig configures the bounded statistics collector
type BoundedStatsConfig struct {
	ProcessingWindowSize int           // Number of samples for processing times
	QueueWindowSize      int           // Number of samples for queue depths
	ThroughputBuckets    int           // Number of buckets for throughput calculation
	ThroughputInterval   time.Duration // Time per bucket
	MaxRecentFiles       int           // Maximum recent files to track
	MaxErrorTypes        int           // Maximum error types to track
	AggregationInterval  time.Duration // How often to compute aggregates
}

// DefaultBoundedStatsConfig returns optimized default configuration
func DefaultBoundedStatsConfig() *BoundedStatsConfig {
	return &BoundedStatsConfig{
		ProcessingWindowSize: 1000,
		QueueWindowSize:      100,
		ThroughputBuckets:    60, // 60 seconds with 1-second buckets
		ThroughputInterval:   1 * time.Second,
		MaxRecentFiles:       100,
		MaxErrorTypes:        50,
		AggregationInterval:  5 * time.Second,
	}
}

// NewBoundedStats creates a new bounded statistics collector
func NewBoundedStats(config *BoundedStatsConfig) *BoundedStats {
	if config == nil {
		config = DefaultBoundedStatsConfig()
	}

	stats := &BoundedStats{
		processingTimes: NewRingBuffer(config.ProcessingWindowSize),
		queueDepths:     NewRingBuffer(config.QueueWindowSize),
		memorySnapshots: NewRingBuffer(60), // Last 60 samples
		cpuSnapshots:    NewRingBuffer(60),
		recentFiles:     NewBoundedFileList(config.MaxRecentFiles),
		errorSummary:    NewErrorSummary(config.MaxErrorTypes),
		aggregates:      &AggregatedMetrics{},
		lastAggregation: time.Now(),
	}

	// Initialize throughput window
	stats.throughputWindow = &ThroughputWindow{
		buckets:    make([]int64, config.ThroughputBuckets),
		bucketSize: config.ThroughputInterval,
		windowSize: config.ThroughputBuckets,
		lastUpdate: time.Now(),
	}

	// Start aggregation goroutine
	go stats.aggregationLoop(config.AggregationInterval)

	return stats
}

// RecordProcessing records a successful processing event
func (s *BoundedStats) RecordProcessing(filename string, size int64, duration time.Duration) {
	s.totalProcessed.Add(1)
	s.successCount.Add(1)

	// Add to processing times window
	s.processingTimes.Add(float64(duration.Milliseconds()))

	// Update throughput
	s.throughputWindow.Increment()

	// Add to recent files (memory-efficient)
	s.recentFiles.Add(FileInfo{
		Name:      extractFileName(filename),
		Size:      size,
		Timestamp: time.Now(),
		Status:    "success",
	})
}

// RecordFailure records a failed processing event
func (s *BoundedStats) RecordFailure(filename string, errorType string) {
	s.totalProcessed.Add(1)
	s.failureCount.Add(1)

	// Update error summary
	s.errorSummary.Increment(errorType)

	// Add to recent files
	s.recentFiles.Add(FileInfo{
		Name:      extractFileName(filename),
		Size:      0,
		Timestamp: time.Now(),
		Status:    "failed",
	})
}

// RecordTimeout records a timeout event
func (s *BoundedStats) RecordTimeout() {
	s.timeoutCount.Add(1)
}

// RecordQueueDepth records current queue depth
func (s *BoundedStats) RecordQueueDepth(depth int) {
	s.queueDepths.Add(float64(depth))
}

// RecordMemoryUsage records current memory usage
func (s *BoundedStats) RecordMemoryUsage(bytes int64) {
	s.memorySnapshots.Add(float64(bytes))
}

// RecordCPUUsage records current CPU usage percentage
func (s *BoundedStats) RecordCPUUsage(percentage float64) {
	s.cpuSnapshots.Add(percentage)
}

// GetSnapshot returns a lightweight snapshot of current statistics
func (s *BoundedStats) GetSnapshot() StatsSnapshot {
	// Get latest aggregates
	s.mu.RLock()
	aggregates := *s.aggregates
	s.mu.RUnlock()

	return StatsSnapshot{
		TotalProcessed:    s.totalProcessed.Load(),
		SuccessCount:      s.successCount.Load(),
		FailureCount:      s.failureCount.Load(),
		TimeoutCount:      s.timeoutCount.Load(),
		AvgProcessingTime: aggregates.AvgProcessingTime,
		P95ProcessingTime: aggregates.P95ProcessingTime,
		Throughput:        aggregates.Throughput,
		SuccessRate:       aggregates.SuccessRate,
		CurrentQueueDepth: int(s.queueDepths.Last()),
		AvgMemoryUsage:    s.memorySnapshots.Average(),
		AvgCPUUsage:       s.cpuSnapshots.Average(),
		RecentFiles:       s.recentFiles.GetRecent(10),
		TopErrors:         s.errorSummary.GetTop(5),
		LastUpdate:        aggregates.LastUpdate,
	}
}

// StatsSnapshot represents a point-in-time view of statistics
type StatsSnapshot struct {
	TotalProcessed    int64
	SuccessCount      int64
	FailureCount      int64
	TimeoutCount      int64
	AvgProcessingTime time.Duration
	P95ProcessingTime time.Duration
	Throughput        float64
	SuccessRate       float64
	CurrentQueueDepth int
	AvgMemoryUsage    float64
	AvgCPUUsage       float64
	RecentFiles       []FileInfo
	TopErrors         []ErrorCount
	LastUpdate        time.Time
}

// ErrorCount represents an error type and its count
type ErrorCount struct {
	Type  string
	Count int64
}

// aggregationLoop periodically computes aggregate metrics
func (s *BoundedStats) aggregationLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.computeAggregates()
	}
}

// computeAggregates computes aggregate metrics from raw data
func (s *BoundedStats) computeAggregates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate processing time percentiles
	times := s.processingTimes.GetAll()
	if len(times) > 0 {
		s.aggregates.AvgProcessingTime = time.Duration(average(times)) * time.Millisecond
		s.aggregates.P50ProcessingTime = time.Duration(percentile(times, 50)) * time.Millisecond
		s.aggregates.P95ProcessingTime = time.Duration(percentile(times, 95)) * time.Millisecond
		s.aggregates.P99ProcessingTime = time.Duration(percentile(times, 99)) * time.Millisecond
	}

	// Calculate queue depth metrics
	depths := s.queueDepths.GetAll()
	if len(depths) > 0 {
		s.aggregates.AvgQueueDepth = average(depths)
		s.aggregates.MaxQueueDepth = int(maximum(depths))
	}

	// Calculate throughput
	s.aggregates.Throughput = s.throughputWindow.GetThroughput()

	// Calculate success rate
	total := s.totalProcessed.Load()
	if total > 0 {
		s.aggregates.SuccessRate = float64(s.successCount.Load()) / float64(total)
	}

	s.aggregates.LastUpdate = time.Now()
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]float64, size),
		size: size,
	}
}

// Add adds a value to the ring buffer
func (r *RingBuffer) Add(value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[r.position] = value
	r.position = (r.position + 1) % r.size
	if r.count < r.size {
		r.count++
	}
}

// GetAll returns all values in the buffer
func (r *RingBuffer) GetAll() []float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return nil
	}

	result := make([]float64, r.count)
	if r.count < r.size {
		copy(result, r.data[:r.count])
	} else {
		// Buffer is full, need to reconstruct in order
		idx := 0
		for i := r.position; i < r.size; i++ {
			result[idx] = r.data[i]
			idx++
		}
		for i := 0; i < r.position; i++ {
			result[idx] = r.data[i]
			idx++
		}
	}

	return result
}

// Average returns the average of all values
func (r *RingBuffer) Average() float64 {
	values := r.GetAll()
	if len(values) == 0 {
		return 0
	}
	return average(values)
}

// Last returns the most recent value
func (r *RingBuffer) Last() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return 0
	}

	idx := r.position - 1
	if idx < 0 {
		idx = r.size - 1
	}
	return r.data[idx]
}

// NewBoundedFileList creates a new bounded file list
func NewBoundedFileList(maxSize int) *BoundedFileList {
	return &BoundedFileList{
		files:   make([]FileInfo, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a file to the list
func (l *BoundedFileList) Add(info FileInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.files[l.current] = info
	l.current = (l.current + 1) % l.maxSize
}

// GetRecent returns the N most recent files
func (l *BoundedFileList) GetRecent(n int) []FileInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n > l.maxSize {
		n = l.maxSize
	}

	result := make([]FileInfo, 0, n)
	idx := l.current - 1
	for i := 0; i < n && i < l.maxSize; i++ {
		if idx < 0 {
			idx = l.maxSize - 1
		}
		if l.files[idx].Name != "" {
			result = append(result, l.files[idx])
		}
		idx--
	}

	return result
}

// NewErrorSummary creates a new error summary
func NewErrorSummary(maxKeys int) *ErrorSummary {
	return &ErrorSummary{
		counts:  make(map[string]int64),
		maxKeys: maxKeys,
	}
}

// Increment increments the count for an error type
func (e *ErrorSummary) Increment(errorType string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.counts[errorType]++

	// Prune if too many keys
	if len(e.counts) > e.maxKeys {
		e.pruneLocked()
	}
}

// pruneLocked removes least frequent errors (must be called with lock held)
func (e *ErrorSummary) pruneLocked() {
	// Find minimum count
	minCount := int64(^uint64(0) >> 1) // Max int64
	minKey := ""
	for key, count := range e.counts {
		if count < minCount {
			minCount = count
			minKey = key
		}
	}

	// Remove minimum
	if minKey != "" {
		delete(e.counts, minKey)
	}
}

// GetTop returns the top N error types by count
func (e *ErrorSummary) GetTop(n int) []ErrorCount {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Convert to slice for sorting
	errors := make([]ErrorCount, 0, len(e.counts))
	for typ, count := range e.counts {
		errors = append(errors, ErrorCount{Type: typ, Count: count})
	}

	// Simple selection sort for top N (efficient for small N)
	for i := 0; i < n && i < len(errors); i++ {
		maxIdx := i
		for j := i + 1; j < len(errors); j++ {
			if errors[j].Count > errors[maxIdx].Count {
				maxIdx = j
			}
		}
		errors[i], errors[maxIdx] = errors[maxIdx], errors[i]
	}

	if len(errors) > n {
		errors = errors[:n]
	}

	return errors
}

// Increment increments the current throughput bucket
func (t *ThroughputWindow) Increment() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastUpdate)

	// Move to new bucket if needed
	bucketsToAdvance := int(elapsed / t.bucketSize)
	if bucketsToAdvance > 0 {
		// Clear old buckets
		for i := 0; i < bucketsToAdvance && i < t.windowSize; i++ {
			t.currentIdx = (t.currentIdx + 1) % t.windowSize
			t.buckets[t.currentIdx] = 0
		}
		t.lastUpdate = now
	}

	// Increment current bucket
	t.buckets[t.currentIdx]++
}

// GetThroughput returns the average throughput per second
func (t *ThroughputWindow) GetThroughput() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	sum := int64(0)
	for _, count := range t.buckets {
		sum += count
	}

	// Calculate average per second
	windowDuration := float64(t.windowSize) * t.bucketSize.Seconds()
	return float64(sum) / windowDuration
}

// Helper functions

func extractFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func maximum(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple implementation - for production use a proper sorting algorithm
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Bubble sort (ok for small datasets)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	idx := int(float64(len(sorted)-1) * p / 100.0)
	return sorted[idx]
}

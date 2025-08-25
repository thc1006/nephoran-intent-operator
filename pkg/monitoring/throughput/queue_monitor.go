package throughput

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// QueueMonitor provides comprehensive queue monitoring
type QueueMonitor struct {
	mu sync.RWMutex

	// Queue tracking
	queues map[string]*QueueTracker

	// Prometheus metrics
	queueDepthMetric *prometheus.GaugeVec
	waitTimeMetric   *prometheus.HistogramVec
	drainRateMetric  *prometheus.CounterVec
}

// QueueTracker monitors individual queue performance
type QueueTracker struct {
	name                 string
	depth                int
	maxDepth             int
	waitTimes            []time.Duration
	priorityDistribution map[Priority]int
}

// Priority represents queue priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// NewQueueMonitor creates a new queue monitor
func NewQueueMonitor() *QueueMonitor {
	return &QueueMonitor{
		queues: make(map[string]*QueueTracker),
		queueDepthMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_queue_depth",
			Help: "Current depth of processing queues",
		}, []string{"queue", "priority"}),
		waitTimeMetric: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_queue_wait_time_seconds",
			Help:    "Time spent in queue before processing",
			Buckets: prometheus.DefBuckets,
		}, []string{"queue", "priority"}),
		drainRateMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_queue_drain_rate",
			Help: "Rate of items being processed from queues",
		}, []string{"queue", "priority", "status"}),
	}
}

// RegisterQueue adds a new queue to monitoring
func (m *QueueMonitor) RegisterQueue(
	name string,
	maxDepth int,
) *QueueTracker {
	m.mu.Lock()
	defer m.mu.Unlock()

	tracker := &QueueTracker{
		name:                 name,
		maxDepth:             maxDepth,
		waitTimes:            make([]time.Duration, 0, 1000),
		priorityDistribution: make(map[Priority]int),
	}
	m.queues[name] = tracker
	return tracker
}

// Enqueue adds an item to the queue
func (m *QueueMonitor) Enqueue(
	queueName string,
	priority Priority,
	enqueuedAt time.Time,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return fmt.Errorf("queue %s not registered", queueName)
	}

	// Increment queue depth
	queue.depth++
	queue.priorityDistribution[priority]++

	// Update Prometheus metrics
	m.queueDepthMetric.WithLabelValues(queueName, priority.String()).Inc()

	// Check for queue overflow
	if queue.depth > queue.maxDepth {
		return fmt.Errorf("queue %s exceeded max depth of %d", queueName, queue.maxDepth)
	}

	return nil
}

// Dequeue removes an item from the queue
func (m *QueueMonitor) Dequeue(
	queueName string,
	priority Priority,
	enqueuedAt time.Time,
	status string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return fmt.Errorf("queue %s not registered", queueName)
	}

	// Decrement queue depth
	queue.depth--
	queue.priorityDistribution[priority]--

	// Calculate wait time
	waitTime := time.Since(enqueuedAt)
	queue.waitTimes = append(queue.waitTimes, waitTime)

	// Limit wait times history
	if len(queue.waitTimes) > 1000 {
		queue.waitTimes = queue.waitTimes[1:]
	}

	// Update Prometheus metrics
	m.queueDepthMetric.WithLabelValues(queueName, priority.String()).Dec()
	m.waitTimeMetric.WithLabelValues(queueName, priority.String()).Observe(waitTime.Seconds())
	m.drainRateMetric.WithLabelValues(queueName, priority.String(), status).Inc()

	return nil
}

// GetQueueStats retrieves queue performance statistics
func (m *QueueMonitor) GetQueueStats(queueName string) (*QueueStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return nil, fmt.Errorf("queue %s not registered", queueName)
	}

	// Calculate wait time statistics
	var totalWait time.Duration
	for _, wait := range queue.waitTimes {
		totalWait += wait
	}
	avgWaitTime := totalWait / time.Duration(len(queue.waitTimes))

	return &QueueStats{
		CurrentDepth:         queue.depth,
		MaxDepth:             queue.maxDepth,
		AverageWaitTime:      avgWaitTime,
		PriorityDistribution: queue.priorityDistribution,
	}, nil
}

// QueueStats represents detailed queue performance metrics
type QueueStats struct {
	CurrentDepth         int
	MaxDepth             int
	AverageWaitTime      time.Duration
	PriorityDistribution map[Priority]int
}

// String converts priority to readable string
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// BackpressureController manages queue flow control
type BackpressureController struct {
	mu sync.Mutex

	// Thresholds for backpressure activation
	maxQueueDepth      int
	backpressureActive bool

	// Metrics
	backpressureMetric *prometheus.GaugeVec
}

// NewBackpressureController creates a new backpressure controller
func NewBackpressureController(maxDepth int) *BackpressureController {
	return &BackpressureController{
		maxQueueDepth: maxDepth,
		backpressureMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_backpressure_active",
			Help: "Indicates if backpressure is currently active",
		}, []string{"queue"}),
	}
}

// ShouldApplyBackpressure checks if backpressure is needed
func (bc *BackpressureController) ShouldApplyBackpressure(
	currentDepth int,
	queueName string,
) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Apply backpressure if queue depth exceeds threshold
	shouldApply := currentDepth >= bc.maxQueueDepth

	if shouldApply && !bc.backpressureActive {
		bc.backpressureActive = true
		bc.backpressureMetric.WithLabelValues(queueName).Set(1)
	} else if !shouldApply && bc.backpressureActive {
		bc.backpressureActive = false
		bc.backpressureMetric.WithLabelValues(queueName).Set(0)
	}

	return shouldApply
}

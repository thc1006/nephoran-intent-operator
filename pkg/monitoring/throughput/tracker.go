package throughput

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// IntentType represents the different types of intents
type IntentType string

const (
	IntentTypeCore     IntentType = "core"
	IntentTypeRAN      IntentType = "ran"
	IntentTypeSlicing  IntentType = "slicing"
	IntentTypeDefault  IntentType = "default"
)

// ThroughputTracker provides real-time throughput monitoring
type ThroughputTracker struct {
	mu sync.RWMutex

	// Sliding window counters for different time windows
	windows map[time.Duration]*SlidingWindowCounter

	// Prometheus metrics
	intentProcessedTotal *prometheus.CounterVec
	intentProcessingTime *prometheus.HistogramVec
	queueDepth           *prometheus.GaugeVec

	// Burst detection
	burstDetector *BurstDetector

	// Geographic tracking
	regionCounters map[string]*SlidingWindowCounter
}

// NewThroughputTracker creates a new throughput tracker
func NewThroughputTracker() *ThroughputTracker {
	windows := map[time.Duration]*SlidingWindowCounter{
		time.Minute:   NewSlidingWindowCounter(time.Minute),
		time.Minute * 5: NewSlidingWindowCounter(time.Minute * 5),
		time.Minute * 15: NewSlidingWindowCounter(time.Minute * 15),
		time.Hour:    NewSlidingWindowCounter(time.Hour),
	}

	return &ThroughputTracker{
		windows: windows,
		intentProcessedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_intent_processed_total",
			Help: "Total number of intents processed",
		}, []string{"type", "status", "region"}),
		intentProcessingTime: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_intent_processing_duration_seconds",
			Help:    "Duration of intent processing",
			Buckets: prometheus.DefBuckets,
		}, []string{"type", "status"}),
		queueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_intent_queue_depth",
			Help: "Current depth of intent processing queues",
		}, []string{"queue"}),
		burstDetector: NewBurstDetector(),
		regionCounters: make(map[string]*SlidingWindowCounter),
	}
}

// RecordIntent records an intent processing event
func (t *ThroughputTracker) RecordIntent(
	intentType IntentType, 
	status string, 
	processingTime time.Duration, 
	region string,
) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Update sliding window counters
	for _, window := range t.windows {
		window.Increment()
	}

	// Update region-specific counter
	if _, exists := t.regionCounters[region]; !exists {
		t.regionCounters[region] = NewSlidingWindowCounter(time.Minute * 15)
	}
	t.regionCounters[region].Increment()

	// Prometheus metrics
	t.intentProcessedTotal.WithLabelValues(string(intentType), status, region).Inc()
	t.intentProcessingTime.WithLabelValues(string(intentType), status).Observe(processingTime.Seconds())

	// Burst detection
	t.burstDetector.Record()
}

// GetThroughputRates returns throughput rates for different windows
func (t *ThroughputTracker) GetThroughputRates() map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rates := make(map[string]float64)
	for duration, counter := range t.windows {
		rates[fmt.Sprintf("%d_min", duration/time.Minute)] = counter.Rate()
	}
	return rates
}

// GetRegionalThroughput returns throughput rates by region
func (t *ThroughputTracker) GetRegionalThroughput() map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	regionalRates := make(map[string]float64)
	for region, counter := range t.regionCounters {
		regionalRates[region] = counter.Rate()
	}
	return regionalRates
}

// SlidingWindowCounter tracks events in a sliding time window
type SlidingWindowCounter struct {
	mu        sync.Mutex
	window    time.Duration
	buckets   map[int64]int
	totalSum  int
	startTime int64
}

// NewSlidingWindowCounter creates a new sliding window counter
func NewSlidingWindowCounter(window time.Duration) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		window:    window,
		buckets:   make(map[int64]int),
		startTime: time.Now().Unix(),
	}
}

// Increment adds a new event to the counter
func (c *SlidingWindowCounter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()
	c.buckets[now]++
	c.totalSum++

	// Remove old buckets
	c.prune(now)
}

// Rate calculates the rate of events
func (c *SlidingWindowCounter) Rate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()
	c.prune(now)

	elapsed := math.Max(float64(now-c.startTime), 1)
	return float64(c.totalSum) / elapsed * 60 // Per minute rate
}

// prune removes old buckets outside the time window
func (c *SlidingWindowCounter) prune(now int64) {
	for bucket := range c.buckets {
		if now-bucket > int64(c.window.Seconds()) {
			c.totalSum -= c.buckets[bucket]
			delete(c.buckets, bucket)
		}
	}
}

// BurstDetector tracks burst events
type BurstDetector struct {
	mu        sync.Mutex
	timestamps []time.Time
	burstThreshold int
}

// NewBurstDetector creates a new burst detector
func NewBurstDetector() *BurstDetector {
	return &BurstDetector{
		timestamps:     make([]time.Time, 0),
		burstThreshold: 1000, // 1000 intents per second threshold
	}
}

// Record adds a new timestamp
func (b *BurstDetector) Record() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.timestamps = append(b.timestamps, now)
	b.prune(now)
}

// prune removes old timestamps and checks for bursts
func (b *BurstDetector) prune(now time.Time) {
	// Remove timestamps older than 1 second
	for len(b.timestamps) > 0 && now.Sub(b.timestamps[0]) > time.Second {
		b.timestamps = b.timestamps[1:]
	}
}

// IsBurstDetected checks if a burst has occurred
func (b *BurstDetector) IsBurstDetected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.timestamps) >= b.burstThreshold
}
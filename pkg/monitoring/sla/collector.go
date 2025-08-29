
package sla



import (

	"context"

	"fmt"

	"runtime"

	"strings"

	"sync"

	"sync/atomic"

	"time"



	"github.com/prometheus/client_golang/prometheus"

	"golang.org/x/time/rate"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

)



// Collector provides high-performance metrics collection with sub-100ms latency.

// and 100,000+ metrics/second throughput with intelligent sampling.

type Collector struct {

	config  *CollectorConfig

	logger  *logging.StructuredLogger

	started atomic.Bool



	// High-performance buffering.

	metricsBuffer  chan *MetricSample

	batchProcessor *BatchProcessor

	cardinalityMgr *CardinalityManager

	rateLimiter    *rate.Limiter



	// Connection pools for data sources.

	connPool *ConnectionPool



	// Performance metrics.

	metrics          *CollectorMetrics

	collectionCount  atomic.Uint64

	processingErrors atomic.Uint64

	sampledMetrics   atomic.Uint64

	droppedMetrics   atomic.Uint64



	// Worker management.

	workers        []*CollectorWorker

	workerCount    int

	processingRate atomic.Uint64



	// Synchronization.

	mu     sync.RWMutex

	stopCh chan struct{}

	wg     sync.WaitGroup

}



// CollectorConfig holds configuration for the metrics collector.

type CollectorConfig struct {

	// Buffer configuration for high throughput.

	BufferSize    int           `yaml:"buffer_size"`

	BatchSize     int           `yaml:"batch_size"`

	FlushInterval time.Duration `yaml:"flush_interval"`

	WorkerCount   int           `yaml:"worker_count"`



	// Cardinality management.

	MaxCardinality    int            `yaml:"max_cardinality"`

	CardinalityLimits map[string]int `yaml:"cardinality_limits"`



	// Sampling configuration.

	SamplingRate      float64 `yaml:"sampling_rate"`

	AdaptiveSampling  bool    `yaml:"adaptive_sampling"`

	SamplingThreshold int     `yaml:"sampling_threshold"`



	// Connection pooling.

	MaxConnections    int           `yaml:"max_connections"`

	ConnectionTimeout time.Duration `yaml:"connection_timeout"`

	IdleTimeout       time.Duration `yaml:"idle_timeout"`



	// Performance limits.

	MaxThroughput     int `yaml:"max_throughput"`

	BackpressureLimit int `yaml:"backpressure_limit"`

}



// DefaultCollectorConfig returns optimized default configuration.

func DefaultCollectorConfig() *CollectorConfig {

	return &CollectorConfig{

		BufferSize:    50000, // 50k metrics buffer

		BatchSize:     1000,  // Process in 1k batches

		FlushInterval: 100 * time.Millisecond,

		WorkerCount:   runtime.NumCPU(),



		MaxCardinality: 10000, // Prevent cardinality explosion

		CardinalityLimits: map[string]int{

			"intent_metrics":    5000,

			"component_metrics": 2000,

			"system_metrics":    1000,

		},



		SamplingRate:      1.0,   // Start with no sampling

		AdaptiveSampling:  true,  // Enable adaptive sampling under load

		SamplingThreshold: 75000, // Start sampling above 75k/sec



		MaxConnections:    20, // Connection pool size

		ConnectionTimeout: 5 * time.Second,

		IdleTimeout:       30 * time.Second,



		MaxThroughput:     100000, // 100k metrics/second limit

		BackpressureLimit: 40000,  // Apply backpressure at 40k buffer

	}

}



// MetricSample represents a single metric measurement.

type MetricSample struct {

	Name      string                 `json:"name"`

	Value     float64                `json:"value"`

	Labels    map[string]string      `json:"labels"`

	Timestamp time.Time              `json:"timestamp"`

	Type      MetricType             `json:"type"`

	Source    string                 `json:"source"`

	Priority  Priority               `json:"priority"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`

}



// MetricType defines the type of metric.

type MetricType string



const (

	// MetricTypeCounter holds metrictypecounter value.

	MetricTypeCounter MetricType = "counter"

	// MetricTypeGauge holds metrictypegauge value.

	MetricTypeGauge MetricType = "gauge"

	// MetricTypeHistogram holds metrictypehistogram value.

	MetricTypeHistogram MetricType = "histogram"

	// MetricTypeSummary holds metrictypesummary value.

	MetricTypeSummary MetricType = "summary"

)



// CollectorMetrics contains Prometheus metrics for the collector.

type CollectorMetrics struct {

	// Collection metrics.

	MetricsCollected  *prometheus.CounterVec

	CollectionLatency *prometheus.HistogramVec

	CollectionRate    prometheus.Gauge



	// Buffer metrics.

	BufferUtilization prometheus.Gauge

	BufferOverflows   prometheus.Counter

	DroppedMetrics    *prometheus.CounterVec



	// Processing metrics.

	BatchesProcessed  prometheus.Counter

	ProcessingLatency prometheus.Histogram

	ProcessingErrors  *prometheus.CounterVec



	// Cardinality metrics.

	CardinalityCount   *prometheus.GaugeVec

	CardinalityLimited *prometheus.CounterVec



	// Sampling metrics.

	SamplingRate   prometheus.Gauge

	SampledMetrics prometheus.Counter



	// Connection pool metrics.

	ActiveConnections prometheus.Gauge

	ConnectionErrors  *prometheus.CounterVec

	ConnectionLatency prometheus.Histogram

}



// BatchProcessor handles batch processing of metrics for efficiency.

type BatchProcessor struct {

	batchSize     int

	flushInterval time.Duration

	processor     func([]*MetricSample) error

	buffer        []*MetricSample

	mu            sync.Mutex

	lastFlush     time.Time

	metrics       *BatchProcessorMetrics

}



// BatchProcessorMetrics tracks batch processor performance.

type BatchProcessorMetrics struct {

	BatchesProcessed prometheus.Counter

	BatchSize        prometheus.Histogram

	ProcessingTime   prometheus.Histogram

	ProcessingErrors prometheus.Counter

}



// CardinalityManager prevents metric cardinality explosion.

type CardinalityManager struct {

	maxCardinality     int

	familyLimits       map[string]int

	currentCardinality map[string]int

	totalCardinality   atomic.Int64

	mu                 sync.RWMutex

	metrics            *CardinalityMetrics

}



// CardinalityMetrics tracks cardinality management.

type CardinalityMetrics struct {

	TotalCardinality  prometheus.Gauge

	FamilyCardinality *prometheus.GaugeVec

	DroppedMetrics    *prometheus.CounterVec

	LimitViolations   *prometheus.CounterVec

}



// CollectorWorker processes metrics in parallel.

type CollectorWorker struct {

	id        int

	collector *Collector

	tasks     chan *MetricSample

	quit      chan struct{}

	metrics   *WorkerMetrics

}



// ConnectionPool manages connections to data sources.

type ConnectionPool struct {

	maxConnections int

	connections    chan *Connection

	factory        func() (*Connection, error)

	metrics        *ConnectionPoolMetrics

}



// Connection represents a connection to a data source.

type Connection struct {

	ID        int

	CreatedAt time.Time

	LastUsed  time.Time

	InUse     atomic.Bool

}



// ConnectionPoolMetrics tracks connection pool performance.

type ConnectionPoolMetrics struct {

	ActiveConnections prometheus.Gauge

	TotalConnections  prometheus.Gauge

	ConnectionTime    prometheus.Histogram

	ConnectionErrors  *prometheus.CounterVec

}



// NewCollector creates a new high-performance metrics collector.

func NewCollector(config *CollectorConfig, logger *logging.StructuredLogger) (*Collector, error) {

	if config == nil {

		config = DefaultCollectorConfig()

	}



	if logger == nil {

		return nil, fmt.Errorf("logger is required")

	}



	// Initialize metrics.

	metrics := &CollectorMetrics{

		MetricsCollected: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_collector_metrics_collected_total",

			Help: "Total number of metrics collected",

		}, []string{"source", "type", "status"}),



		CollectionLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name:    "sla_collector_collection_latency_seconds",

			Help:    "Latency of metrics collection operations",

			Buckets: prometheus.ExponentialBuckets(0.00001, 2, 15), // 10μs to ~300ms

		}, []string{"source", "type"}),



		CollectionRate: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_collector_collection_rate",

			Help: "Current metrics collection rate per second",

		}),



		BufferUtilization: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_collector_buffer_utilization_percent",

			Help: "Buffer utilization percentage",

		}),



		BufferOverflows: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_collector_buffer_overflows_total",

			Help: "Total number of buffer overflows",

		}),



		DroppedMetrics: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_collector_dropped_metrics_total",

			Help: "Total number of dropped metrics",

		}, []string{"reason"}),



		BatchesProcessed: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_collector_batches_processed_total",

			Help: "Total number of metric batches processed",

		}),



		ProcessingLatency: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name:    "sla_collector_processing_latency_seconds",

			Help:    "Latency of batch processing operations",

			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to ~400ms

		}),



		ProcessingErrors: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_collector_processing_errors_total",

			Help: "Total number of processing errors",

		}, []string{"error_type"}),



		CardinalityCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "sla_collector_cardinality_count",

			Help: "Current cardinality count by metric family",

		}, []string{"family"}),



		CardinalityLimited: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_collector_cardinality_limited_total",

			Help: "Total number of metrics dropped due to cardinality limits",

		}, []string{"family", "reason"}),



		SamplingRate: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_collector_sampling_rate",

			Help: "Current sampling rate (0.0-1.0)",

		}),



		SampledMetrics: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_collector_sampled_metrics_total",

			Help: "Total number of metrics processed via sampling",

		}),



		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_collector_active_connections",

			Help: "Number of active connections in the pool",

		}),



		ConnectionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_collector_connection_errors_total",

			Help: "Total number of connection errors",

		}, []string{"error_type"}),



		ConnectionLatency: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name:    "sla_collector_connection_latency_seconds",

			Help:    "Connection establishment latency",

			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),

		}),

	}



	// Initialize cardinality manager.

	cardinalityMgr := &CardinalityManager{

		maxCardinality:     config.MaxCardinality,

		familyLimits:       config.CardinalityLimits,

		currentCardinality: make(map[string]int),

		metrics: &CardinalityMetrics{

			TotalCardinality: prometheus.NewGauge(prometheus.GaugeOpts{

				Name: "sla_collector_total_cardinality",

				Help: "Total metric cardinality across all families",

			}),

			FamilyCardinality: prometheus.NewGaugeVec(prometheus.GaugeOpts{

				Name: "sla_collector_family_cardinality",

				Help: "Cardinality by metric family",

			}, []string{"family"}),

			DroppedMetrics: prometheus.NewCounterVec(prometheus.CounterOpts{

				Name: "sla_collector_cardinality_dropped_total",

				Help: "Metrics dropped due to cardinality limits",

			}, []string{"family", "reason"}),

			LimitViolations: prometheus.NewCounterVec(prometheus.CounterOpts{

				Name: "sla_collector_cardinality_violations_total",

				Help: "Cardinality limit violations",

			}, []string{"family", "violation_type"}),

		},

	}



	// Initialize connection pool.

	connPool := &ConnectionPool{

		maxConnections: config.MaxConnections,

		connections:    make(chan *Connection, config.MaxConnections),

		factory: func() (*Connection, error) {

			return &Connection{

				ID:        int(time.Now().UnixNano()),

				CreatedAt: time.Now(),

				LastUsed:  time.Now(),

			}, nil

		},

		metrics: &ConnectionPoolMetrics{

			ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{

				Name: "sla_collector_pool_active_connections",

				Help: "Active connections in the pool",

			}),

			TotalConnections: prometheus.NewGauge(prometheus.GaugeOpts{

				Name: "sla_collector_pool_total_connections",

				Help: "Total connections created",

			}),

			ConnectionTime: prometheus.NewHistogram(prometheus.HistogramOpts{

				Name: "sla_collector_pool_connection_time_seconds",

				Help: "Time to acquire connection from pool",

			}),

			ConnectionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{

				Name: "sla_collector_pool_connection_errors_total",

				Help: "Connection pool errors",

			}, []string{"error_type"}),

		},

	}



	// Initialize batch processor.

	batchProcessor := &BatchProcessor{

		batchSize:     config.BatchSize,

		flushInterval: config.FlushInterval,

		buffer:        make([]*MetricSample, 0, config.BatchSize),

		lastFlush:     time.Now(),

		metrics: &BatchProcessorMetrics{

			BatchesProcessed: prometheus.NewCounter(prometheus.CounterOpts{

				Name: "sla_collector_batch_processor_batches_total",

				Help: "Total batches processed",

			}),

			BatchSize: prometheus.NewHistogram(prometheus.HistogramOpts{

				Name:    "sla_collector_batch_processor_batch_size",

				Help:    "Size of processed batches",

				Buckets: prometheus.LinearBuckets(0, 100, 20), // 0 to 2000

			}),

			ProcessingTime: prometheus.NewHistogram(prometheus.HistogramOpts{

				Name:    "sla_collector_batch_processor_processing_time_seconds",

				Help:    "Batch processing time",

				Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12),

			}),

			ProcessingErrors: prometheus.NewCounter(prometheus.CounterOpts{

				Name: "sla_collector_batch_processor_errors_total",

				Help: "Batch processing errors",

			}),

		},

	}



	// Initialize rate limiter.

	rateLimiter := rate.NewLimiter(rate.Limit(config.MaxThroughput), config.MaxThroughput/10)



	collector := &Collector{

		config:         config,

		logger:         logger.WithComponent("collector"),

		metricsBuffer:  make(chan *MetricSample, config.BufferSize),

		batchProcessor: batchProcessor,

		cardinalityMgr: cardinalityMgr,

		rateLimiter:    rateLimiter,

		connPool:       connPool,

		metrics:        metrics,

		workerCount:    config.WorkerCount,

		stopCh:         make(chan struct{}),

	}



	// Initialize workers.

	collector.workers = make([]*CollectorWorker, config.WorkerCount)

	for i := range config.WorkerCount {

		collector.workers[i] = &CollectorWorker{

			id:        i,

			collector: collector,

			tasks:     make(chan *MetricSample, 100),

			quit:      make(chan struct{}),

			metrics: &WorkerMetrics{

				TasksProcessed: prometheus.NewCounter(prometheus.CounterOpts{

					Name:        "sla_collector_worker_tasks_processed_total",

					Help:        "Tasks processed by worker",

					ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", i)},

				}),

				ProcessingTime: prometheus.NewHistogram(prometheus.HistogramOpts{

					Name:        "sla_collector_worker_processing_time_seconds",

					Help:        "Processing time by worker",

					ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", i)},

				}),

				Errors: prometheus.NewCounter(prometheus.CounterOpts{

					Name:        "sla_collector_worker_errors_total",

					Help:        "Errors by worker",

					ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", i)},

				}),

			},

		}

	}



	// Set processor function.

	batchProcessor.processor = collector.processBatch



	return collector, nil

}



// Start begins the metrics collection process.

func (c *Collector) Start(ctx context.Context) error {

	if c.started.Load() {

		return fmt.Errorf("collector already started")

	}



	c.logger.InfoWithContext("Starting metrics collector",

		"buffer_size", c.config.BufferSize,

		"batch_size", c.config.BatchSize,

		"worker_count", c.workerCount,

		"max_throughput", c.config.MaxThroughput,

	)



	// Initialize connection pool.

	if err := c.initializeConnectionPool(); err != nil {

		return fmt.Errorf("failed to initialize connection pool: %w", err)

	}



	// Start workers.

	for i, worker := range c.workers {

		c.wg.Add(1)

		go c.runWorker(ctx, worker, i)

	}



	// Start main collection loop.

	c.wg.Add(1)

	go c.runCollectionLoop(ctx)



	// Start batch processor.

	c.wg.Add(1)

	go c.runBatchProcessor(ctx)



	// Start metrics updater.

	c.wg.Add(1)

	go c.updateMetrics(ctx)



	c.started.Store(true)

	c.logger.InfoWithContext("Metrics collector started successfully")



	return nil

}



// Stop gracefully stops the metrics collector.

func (c *Collector) Stop(ctx context.Context) error {

	if !c.started.Load() {

		return nil

	}



	c.logger.InfoWithContext("Stopping metrics collector")



	// Signal stop.

	close(c.stopCh)



	// Stop workers.

	for _, worker := range c.workers {

		close(worker.quit)

	}



	// Wait for workers to finish.

	c.wg.Wait()



	// Close buffer.

	close(c.metricsBuffer)



	c.logger.InfoWithContext("Metrics collector stopped")



	return nil

}



// CollectMetric collects a single metric with sub-100ms latency.

func (c *Collector) CollectMetric(ctx context.Context, sample *MetricSample) error {

	start := time.Now()

	defer func() {

		latency := time.Since(start)

		c.metrics.CollectionLatency.WithLabelValues(sample.Source, string(sample.Type)).Observe(latency.Seconds())

	}()



	// Apply rate limiting.

	if !c.rateLimiter.Allow() {

		c.metrics.DroppedMetrics.WithLabelValues("rate_limited").Inc()

		c.droppedMetrics.Add(1)

		return fmt.Errorf("rate limit exceeded")

	}



	// Check cardinality limits.

	if !c.cardinalityMgr.ShouldAcceptMetric(sample.Name, sample.Labels) {

		c.metrics.DroppedMetrics.WithLabelValues("cardinality_limit").Inc()

		c.droppedMetrics.Add(1)

		return fmt.Errorf("cardinality limit exceeded")

	}



	// Apply adaptive sampling if enabled.

	if c.config.AdaptiveSampling {

		if !c.shouldSample() {

			c.metrics.SampledMetrics.Inc()

			c.sampledMetrics.Add(1)

			return nil // Sampled out

		}

	}



	// Add to buffer with timeout.

	select {

	case c.metricsBuffer <- sample:

		c.metrics.MetricsCollected.WithLabelValues(sample.Source, string(sample.Type), "success").Inc()

		c.collectionCount.Add(1)

		return nil

	case <-ctx.Done():

		return ctx.Err()

	default:

		// Buffer full, apply backpressure.

		c.metrics.BufferOverflows.Inc()

		c.metrics.DroppedMetrics.WithLabelValues("buffer_full").Inc()

		c.droppedMetrics.Add(1)

		return fmt.Errorf("collection buffer full")

	}

}



// CollectBatch collects multiple metrics efficiently.

func (c *Collector) CollectBatch(ctx context.Context, samples []*MetricSample) error {

	start := time.Now()

	defer func() {

		latency := time.Since(start)

		c.metrics.ProcessingLatency.Observe(latency.Seconds())

	}()



	successCount := 0

	for _, sample := range samples {

		if err := c.CollectMetric(ctx, sample); err != nil {

			c.logger.WarnWithContext("Failed to collect metric in batch",

				"metric", sample.Name,

				"error", err.Error(),

			)

		} else {

			successCount++

		}

	}



	c.logger.DebugWithContext("Batch collection completed",

		"total", len(samples),

		"successful", successCount,

		"failed", len(samples)-successCount,

	)



	return nil

}



// GetStats returns current collector statistics.

func (c *Collector) GetStats() CollectorStats {

	return CollectorStats{

		MetricsCollected:  c.collectionCount.Load(),

		ProcessingErrors:  c.processingErrors.Load(),

		SampledMetrics:    c.sampledMetrics.Load(),

		DroppedMetrics:    c.droppedMetrics.Load(),

		BufferUtilization: float64(len(c.metricsBuffer)) / float64(cap(c.metricsBuffer)) * 100,

		ProcessingRate:    c.processingRate.Load(),

		TotalCardinality:  c.cardinalityMgr.totalCardinality.Load(),

	}

}



// CollectorStats contains collector performance statistics.

type CollectorStats struct {

	MetricsCollected  uint64  `json:"metrics_collected"`

	ProcessingErrors  uint64  `json:"processing_errors"`

	SampledMetrics    uint64  `json:"sampled_metrics"`

	DroppedMetrics    uint64  `json:"dropped_metrics"`

	BufferUtilization float64 `json:"buffer_utilization_percent"`

	ProcessingRate    uint64  `json:"processing_rate"`

	TotalCardinality  int64   `json:"total_cardinality"`

}



// runCollectionLoop runs the main collection loop.

func (c *Collector) runCollectionLoop(ctx context.Context) {

	defer c.wg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-c.stopCh:

			return

		case sample := <-c.metricsBuffer:

			// Distribute to workers using round-robin.

			workerIndex := int(c.collectionCount.Load()) % c.workerCount

			worker := c.workers[workerIndex]



			select {

			case worker.tasks <- sample:

				// Task dispatched successfully.

			default:

				// Worker busy, process in batch processor.

				c.batchProcessor.AddSample(sample)

			}

		}

	}

}



// runWorker runs a collector worker goroutine.

func (c *Collector) runWorker(ctx context.Context, worker *CollectorWorker, workerID int) {

	defer c.wg.Done()



	c.logger.DebugWithContext("Starting collector worker", "worker_id", workerID)



	for {

		select {

		case <-ctx.Done():

			return

		case <-worker.quit:

			return

		case sample := <-worker.tasks:

			c.processMetricSample(ctx, worker, sample)

		}

	}

}



// processMetricSample processes a single metric sample.

func (c *Collector) processMetricSample(ctx context.Context, worker *CollectorWorker, sample *MetricSample) {

	start := time.Now()

	defer func() {

		duration := time.Since(start)

		worker.metrics.ProcessingTime.Observe(duration.Seconds())

		worker.metrics.TasksProcessed.Inc()

	}()



	// Process the metric sample.

	if err := c.storeSample(ctx, sample); err != nil {

		worker.metrics.Errors.Inc()

		c.processingErrors.Add(1)

		c.logger.ErrorWithContext("Failed to store metric sample",

			err,

			"worker_id", worker.id,

			"metric", sample.Name,

		)

		return

	}



	// Update cardinality tracking.

	c.cardinalityMgr.UpdateCardinality(sample.Name, sample.Labels)

	c.processingRate.Add(1)

}



// runBatchProcessor runs the batch processor.

func (c *Collector) runBatchProcessor(ctx context.Context) {

	defer c.wg.Done()



	ticker := time.NewTicker(c.config.FlushInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			// Flush remaining samples.

			c.batchProcessor.Flush()

			return

		case <-c.stopCh:

			c.batchProcessor.Flush()

			return

		case <-ticker.C:

			c.batchProcessor.FlushIfNeeded()

		}

	}

}



// updateMetrics updates collector performance metrics.

func (c *Collector) updateMetrics(ctx context.Context) {

	defer c.wg.Done()



	ticker := time.NewTicker(5 * time.Second)

	defer ticker.Stop()



	var lastCollected uint64

	lastUpdate := time.Now()



	for {

		select {

		case <-ctx.Done():

			return

		case <-c.stopCh:

			return

		case <-ticker.C:

			now := time.Now()

			duration := now.Sub(lastUpdate).Seconds()



			// Calculate collection rate.

			currentCollected := c.collectionCount.Load()

			rate := float64(currentCollected-lastCollected) / duration

			c.metrics.CollectionRate.Set(rate)



			// Update buffer utilization.

			utilization := float64(len(c.metricsBuffer)) / float64(cap(c.metricsBuffer)) * 100

			c.metrics.BufferUtilization.Set(utilization)



			// Update sampling rate.

			c.metrics.SamplingRate.Set(c.getCurrentSamplingRate())



			// Update cardinality metrics.

			c.cardinalityMgr.updateMetrics()



			lastCollected = currentCollected

			lastUpdate = now

		}

	}

}



// shouldSample determines if a metric should be sampled based on current load.

func (c *Collector) shouldSample() bool {

	if !c.config.AdaptiveSampling {

		return true

	}



	currentRate := c.processingRate.Load()

	if currentRate < uint64(c.config.SamplingThreshold) {

		return true

	}



	// Calculate adaptive sampling rate.

	overloadFactor := float64(currentRate) / float64(c.config.SamplingThreshold)

	samplingRate := 1.0 / overloadFactor



	// Ensure minimum sampling rate.

	if samplingRate < 0.1 {

		samplingRate = 0.1

	}



	return c.config.SamplingRate >= samplingRate

}



// getCurrentSamplingRate returns the current effective sampling rate.

func (c *Collector) getCurrentSamplingRate() float64 {

	if !c.config.AdaptiveSampling {

		return c.config.SamplingRate

	}



	currentRate := c.processingRate.Load()

	if currentRate < uint64(c.config.SamplingThreshold) {

		return 1.0

	}



	overloadFactor := float64(currentRate) / float64(c.config.SamplingThreshold)

	return 1.0 / overloadFactor

}



// initializeConnectionPool initializes the connection pool.

func (c *Collector) initializeConnectionPool() error {

	for i := range c.config.MaxConnections {

		conn, err := c.connPool.factory()

		if err != nil {

			return fmt.Errorf("failed to create connection %d: %w", i, err)

		}



		c.connPool.connections <- conn

		c.connPool.metrics.TotalConnections.Inc()

	}



	c.connPool.metrics.ActiveConnections.Set(float64(c.config.MaxConnections))



	return nil

}



// storeSample stores a metric sample (placeholder implementation).

func (c *Collector) storeSample(ctx context.Context, sample *MetricSample) error {

	// This would integrate with actual storage backend.

	// For now, we'll simulate storage operation.



	// Acquire connection from pool.

	conn, err := c.acquireConnection(ctx)

	if err != nil {

		return fmt.Errorf("failed to acquire connection: %w", err)

	}

	defer c.releaseConnection(conn)



	// Simulate storage operation.

	time.Sleep(time.Microsecond * 50) // 50μs simulated storage latency



	return nil

}



// acquireConnection acquires a connection from the pool.

func (c *Collector) acquireConnection(ctx context.Context) (*Connection, error) {

	start := time.Now()

	defer func() {

		c.connPool.metrics.ConnectionTime.Observe(time.Since(start).Seconds())

	}()



	select {

	case conn := <-c.connPool.connections:

		conn.LastUsed = time.Now()

		conn.InUse.Store(true)

		return conn, nil

	case <-ctx.Done():

		c.connPool.metrics.ConnectionErrors.WithLabelValues("timeout").Inc()

		return nil, ctx.Err()

	default:

		c.connPool.metrics.ConnectionErrors.WithLabelValues("pool_exhausted").Inc()

		return nil, fmt.Errorf("connection pool exhausted")

	}

}



// releaseConnection releases a connection back to the pool.

func (c *Collector) releaseConnection(conn *Connection) {

	conn.InUse.Store(false)



	select {

	case c.connPool.connections <- conn:

		// Connection returned to pool.

	default:

		// Pool full, connection will be garbage collected.

		c.connPool.metrics.TotalConnections.Dec()

	}

}



// processBatch processes a batch of metric samples.

func (c *Collector) processBatch(samples []*MetricSample) error {

	start := time.Now()

	defer func() {

		c.batchProcessor.metrics.ProcessingTime.Observe(time.Since(start).Seconds())

		c.batchProcessor.metrics.BatchesProcessed.Inc()

		c.batchProcessor.metrics.BatchSize.Observe(float64(len(samples)))

	}()



	// Process batch in parallel.

	errCh := make(chan error, len(samples))

	semaphore := make(chan struct{}, runtime.NumCPU())



	for _, sample := range samples {

		go func(s *MetricSample) {

			semaphore <- struct{}{}

			defer func() { <-semaphore }()



			if err := c.storeSample(context.Background(), s); err != nil {

				errCh <- err

			} else {

				errCh <- nil

			}

		}(sample)

	}



	// Collect results.

	errorCount := 0

	for range len(samples) {

		if err := <-errCh; err != nil {

			errorCount++

		}

	}



	if errorCount > 0 {

		c.batchProcessor.metrics.ProcessingErrors.Add(float64(errorCount))

		return fmt.Errorf("batch processing failed for %d/%d samples", errorCount, len(samples))

	}



	return nil

}



// AddSample adds a sample to the batch processor.

func (bp *BatchProcessor) AddSample(sample *MetricSample) {

	bp.mu.Lock()

	defer bp.mu.Unlock()



	bp.buffer = append(bp.buffer, sample)



	if len(bp.buffer) >= bp.batchSize {

		bp.flush()

	}

}



// FlushIfNeeded flushes the batch if needed.

func (bp *BatchProcessor) FlushIfNeeded() {

	bp.mu.Lock()

	defer bp.mu.Unlock()



	if len(bp.buffer) > 0 && time.Since(bp.lastFlush) >= bp.flushInterval {

		bp.flush()

	}

}



// Flush flushes all pending samples.

func (bp *BatchProcessor) Flush() {

	bp.mu.Lock()

	defer bp.mu.Unlock()

	bp.flush()

}



// flush internal flush implementation (must hold mutex).

func (bp *BatchProcessor) flush() {

	if len(bp.buffer) == 0 {

		return

	}



	// Process the batch.

	if bp.processor != nil {

		bp.processor(bp.buffer)

	}



	// Reset buffer.

	bp.buffer = bp.buffer[:0]

	bp.lastFlush = time.Now()

}



// ShouldAcceptMetric determines if a metric should be accepted based on cardinality.

func (cm *CardinalityManager) ShouldAcceptMetric(name string, labels map[string]string) bool {

	cm.mu.RLock()

	defer cm.mu.RUnlock()



	// Generate metric key.

	metricKey := cm.generateMetricKey(name, labels)



	// Check if metric already exists.

	if _, exists := cm.currentCardinality[metricKey]; exists {

		return true

	}



	// Check total cardinality limit.

	if cm.totalCardinality.Load() >= int64(cm.maxCardinality) {

		cm.metrics.DroppedMetrics.WithLabelValues(name, "total_limit").Inc()

		return false

	}



	// Check family-specific limits.

	if limit, exists := cm.familyLimits[name]; exists {

		familyCount := cm.getFamilyCardinality(name)

		if familyCount >= limit {

			cm.metrics.DroppedMetrics.WithLabelValues(name, "family_limit").Inc()

			cm.metrics.LimitViolations.WithLabelValues(name, "family_limit").Inc()

			return false

		}

	}



	return true

}



// UpdateCardinality updates cardinality tracking.

func (cm *CardinalityManager) UpdateCardinality(name string, labels map[string]string) {

	cm.mu.Lock()

	defer cm.mu.Unlock()



	metricKey := cm.generateMetricKey(name, labels)



	if _, exists := cm.currentCardinality[metricKey]; !exists {

		cm.currentCardinality[metricKey] = 1

		cm.totalCardinality.Add(1)

	}

}



// generateMetricKey generates a unique key for cardinality tracking.

func (cm *CardinalityManager) generateMetricKey(name string, labels map[string]string) string {

	key := name

	for k, v := range labels {

		key += fmt.Sprintf(",%s=%s", k, v)

	}

	return key

}



// getFamilyCardinality returns cardinality for a metric family.

func (cm *CardinalityManager) getFamilyCardinality(family string) int {

	count := 0

	for key := range cm.currentCardinality {

		if len(key) >= len(family) && key[:len(family)] == family {

			count++

		}

	}

	return count

}



// updateMetrics updates cardinality metrics.

func (cm *CardinalityManager) updateMetrics() {

	cm.mu.RLock()

	defer cm.mu.RUnlock()



	// Update total cardinality.

	cm.metrics.TotalCardinality.Set(float64(cm.totalCardinality.Load()))



	// Update per-family cardinality.

	familyCounts := make(map[string]int)

	for key := range cm.currentCardinality {

		// Extract family name (simplified).

		parts := strings.Split(key, ",")

		if len(parts) > 0 {

			family := parts[0]

			familyCounts[family]++

		}

	}



	for family, count := range familyCounts {

		cm.metrics.FamilyCardinality.WithLabelValues(family).Set(float64(count))

	}

}


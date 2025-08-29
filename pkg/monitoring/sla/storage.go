
package sla



import (

	"compress/gzip"

	"context"

	"encoding/binary"

	"fmt"

	"io"

	"os"

	"path/filepath"

	"sort"

	"sync"

	"sync/atomic"

	"time"



	"github.com/prometheus/client_golang/prometheus"



	"github.com/thc1006/nephoran-intent-operator/pkg/logging"

)



// StorageManager provides efficient time series data storage with compression and retention.

type StorageManager struct {

	config  *StorageConfig

	logger  *logging.StructuredLogger

	started atomic.Bool



	// Storage backends.

	memoryStore *MemoryStore

	diskStore   *DiskStore

	compressor  *Compressor



	// Data lifecycle management.

	retentionMgr  *RetentionManager

	compactionMgr *CompactionManager



	// Performance optimization.

	writeBuffer  *WriteBuffer

	readCache    *ReadCache

	indexManager *IndexManager



	// Background processing.

	flushTicker   *time.Ticker

	compactTicker *time.Ticker

	cleanupTicker *time.Ticker



	// Performance metrics.

	metrics          *StorageMetrics

	writesCount      atomic.Uint64

	readsCount       atomic.Uint64

	compactionsCount atomic.Uint64



	// Synchronization.

	mu     sync.RWMutex

	stopCh chan struct{}

	wg     sync.WaitGroup

}



// StorageConfig holds configuration for the storage manager.

type StorageConfig struct {

	// Data retention.

	RetentionPeriod    time.Duration `yaml:"retention_period"`

	CompactionInterval time.Duration `yaml:"compaction_interval"`

	MaxDiskUsageMB     int64         `yaml:"max_disk_usage_mb"`



	// Compression settings.

	CompressionEnabled bool    `yaml:"compression_enabled"`

	CompressionLevel   int     `yaml:"compression_level"`

	CompressionRatio   float64 `yaml:"compression_ratio"`



	// Performance tuning.

	WriteBufferSize int           `yaml:"write_buffer_size"`

	ReadCacheSize   int           `yaml:"read_cache_size"`

	FlushInterval   time.Duration `yaml:"flush_interval"`

	BatchSize       int           `yaml:"batch_size"`



	// Storage paths.

	DataDirectory   string `yaml:"data_directory"`

	IndexDirectory  string `yaml:"index_directory"`

	BackupDirectory string `yaml:"backup_directory"`



	// Recovery settings.

	BackupEnabled   bool          `yaml:"backup_enabled"`

	BackupInterval  time.Duration `yaml:"backup_interval"`

	RecoveryEnabled bool          `yaml:"recovery_enabled"`

}



// DefaultStorageConfig returns optimized default configuration.

func DefaultStorageConfig() *StorageConfig {

	return &StorageConfig{

		RetentionPeriod:    7 * 24 * time.Hour, // 7 days

		CompactionInterval: 1 * time.Hour,

		MaxDiskUsageMB:     1000, // 1GB



		CompressionEnabled: true,

		CompressionLevel:   6,   // Balanced compression

		CompressionRatio:   0.3, // Expected 70% compression



		WriteBufferSize: 10000,

		ReadCacheSize:   5000,

		FlushInterval:   30 * time.Second,

		BatchSize:       1000,



		DataDirectory:   "./data/sla-metrics",

		IndexDirectory:  "./data/sla-indexes",

		BackupDirectory: "./data/sla-backups",



		BackupEnabled:   true,

		BackupInterval:  6 * time.Hour,

		RecoveryEnabled: true,

	}

}



// DataPoint represents a single time series data point.

type DataPoint struct {

	Timestamp time.Time              `json:"timestamp"`

	Value     float64                `json:"value"`

	Labels    map[string]string      `json:"labels"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`

}



// TimeSeries represents a time series with metadata.

type TimeSeries struct {

	Name        string            `json:"name"`

	Labels      map[string]string `json:"labels"`

	Points      []*DataPoint      `json:"points"`

	Retention   time.Duration     `json:"retention"`

	LastUpdated time.Time         `json:"last_updated"`

}



// StorageMetrics contains Prometheus metrics for the storage manager.

type StorageMetrics struct {

	// Write metrics.

	WritesTotal     *prometheus.CounterVec

	WriteLatency    *prometheus.HistogramVec

	WriteBufferSize prometheus.Gauge

	WriteThroughput prometheus.Gauge



	// Read metrics.

	ReadsTotal      *prometheus.CounterVec

	ReadLatency     *prometheus.HistogramVec

	ReadCacheHits   prometheus.Counter

	ReadCacheMisses prometheus.Counter



	// Storage metrics.

	DiskUsageBytes   prometheus.Gauge

	DataPointsStored prometheus.Gauge

	TimeSeriesCount  prometheus.Gauge

	CompressionRatio prometheus.Gauge



	// Compaction metrics.

	CompactionsTotal  prometheus.Counter

	CompactionLatency prometheus.Histogram

	CompactionSavings prometheus.Gauge



	// Performance metrics.

	FlushOperations prometheus.Counter

	FlushLatency    prometheus.Histogram

	IndexOperations *prometheus.CounterVec

	IndexSize       prometheus.Gauge

}



// MemoryStore provides in-memory time series storage.

type MemoryStore struct {

	series  map[string]*TimeSeries

	maxSize int

	mu      sync.RWMutex

}



// DiskStore provides persistent disk-based storage.

type DiskStore struct {

	basePath    string

	partitions  map[string]*DiskPartition

	indexWriter *IndexWriter

	mu          sync.RWMutex

}



// DiskPartition represents a time-based partition on disk.

type DiskPartition struct {

	StartTime  time.Time

	EndTime    time.Time

	FilePath   string

	Compressed bool

	Size       int64

	PointCount int64

}



// Compressor handles data compression operations.

type Compressor struct {

	enabled          bool

	level            int

	compressionRatio atomic.Uint64 // Stored as percentage * 100

	mu               sync.RWMutex

}



// RetentionManager manages data retention policies.

type RetentionManager struct {

	policies     map[string]*RetentionPolicy

	cleanupQueue chan *CleanupTask

	mu           sync.RWMutex

}



// RetentionPolicy defines data retention rules.

type RetentionPolicy struct {

	Name     string        `json:"name"`

	Duration time.Duration `json:"duration"`

	Pattern  string        `json:"pattern"`

	Action   string        `json:"action"` // delete, compress, archive

	Priority int           `json:"priority"`

}



// CompactionManager handles data compaction operations.

type CompactionManager struct {

	strategy     CompactionStrategy

	thresholds   CompactionThresholds

	workers      []*CompactionWorker

	pendingTasks chan *CompactionTask

	mu           sync.RWMutex

}



// CompactionStrategy defines how data is compacted.

type CompactionStrategy struct {

	Type           string          `json:"type"` // time_based, size_based, hybrid

	TimeWindows    []time.Duration `json:"time_windows"`

	SizeThresholds []int64         `json:"size_thresholds"`

	MaxFiles       int             `json:"max_files"`

}



// CompactionThresholds defines when compaction is triggered.

type CompactionThresholds struct {

	MinFiles      int     `json:"min_files"`

	MaxFiles      int     `json:"max_files"`

	MaxSize       int64   `json:"max_size"`

	FragmentRatio float64 `json:"fragment_ratio"`

}



// WriteBuffer provides efficient batched writes.

type WriteBuffer struct {

	buffer    []*DataPoint

	maxSize   int

	flushSize int

	lastFlush time.Time

	mu        sync.Mutex

}



// ReadCache provides LRU caching for read operations.

type ReadCache struct {

	cache   map[string]*CacheEntry

	maxSize int

	hits    atomic.Uint64

	misses  atomic.Uint64

	mu      sync.RWMutex

}



// CacheEntry represents a cached read result.

type CacheEntry struct {

	Data      interface{}   `json:"data"`

	Timestamp time.Time     `json:"timestamp"`

	TTL       time.Duration `json:"ttl"`

	Size      int           `json:"size"`

}



// IndexManager manages time series indexes.

type IndexManager struct {

	indexes     map[string]*Index

	indexWriter *IndexWriter

	mu          sync.RWMutex

}



// Index represents a time series index.

type Index struct {

	Name        string                 `json:"name"`

	Type        string                 `json:"type"` // hash, btree, inverted

	Keys        []string               `json:"keys"`

	Entries     map[string]*IndexEntry `json:"entries"`

	LastUpdated time.Time              `json:"last_updated"`

}



// IndexEntry represents a single index entry.

type IndexEntry struct {

	Key       string    `json:"key"`

	SeriesID  string    `json:"series_id"`

	TimeRange TimeRange `json:"time_range"`

	Location  string    `json:"location"`

}



// TimeRange represents a time range.

type TimeRange struct {

	Start time.Time `json:"start"`

	End   time.Time `json:"end"`

}



// NewStorageManager creates a new time series storage manager.

func NewStorageManager(config *StorageConfig, logger *logging.StructuredLogger) (*StorageManager, error) {

	if config == nil {

		config = DefaultStorageConfig()

	}



	if logger == nil {

		return nil, fmt.Errorf("logger is required")

	}



	// Initialize metrics.

	metrics := &StorageMetrics{

		WritesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_storage_writes_total",

			Help: "Total number of write operations",

		}, []string{"series", "status"}),



		WriteLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name:    "sla_storage_write_latency_seconds",

			Help:    "Latency of write operations",

			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), // 0.1ms to ~3s

		}, []string{"series", "operation"}),



		WriteBufferSize: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_write_buffer_size",

			Help: "Current size of write buffer",

		}),



		WriteThroughput: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_write_throughput",

			Help: "Write throughput in points per second",

		}),



		ReadsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_storage_reads_total",

			Help: "Total number of read operations",

		}, []string{"series", "cache_status"}),



		ReadLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name:    "sla_storage_read_latency_seconds",

			Help:    "Latency of read operations",

			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 12), // 0.1ms to ~400ms

		}, []string{"series", "source"}),



		ReadCacheHits: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_storage_read_cache_hits_total",

			Help: "Total number of read cache hits",

		}),



		ReadCacheMisses: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_storage_read_cache_misses_total",

			Help: "Total number of read cache misses",

		}),



		DiskUsageBytes: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_disk_usage_bytes",

			Help: "Current disk usage in bytes",

		}),



		DataPointsStored: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_datapoints_stored",

			Help: "Total number of data points stored",

		}),



		TimeSeriesCount: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_timeseries_count",

			Help: "Number of time series being tracked",

		}),



		CompressionRatio: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_compression_ratio",

			Help: "Current compression ratio",

		}),



		CompactionsTotal: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_storage_compactions_total",

			Help: "Total number of compaction operations",

		}),



		CompactionLatency: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name:    "sla_storage_compaction_latency_seconds",

			Help:    "Latency of compaction operations",

			Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to ~68 minutes

		}),



		CompactionSavings: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_compaction_savings_bytes",

			Help: "Space saved by compaction operations",

		}),



		FlushOperations: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_storage_flush_operations_total",

			Help: "Total number of buffer flush operations",

		}),



		FlushLatency: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name:    "sla_storage_flush_latency_seconds",

			Help:    "Latency of buffer flush operations",

			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s

		}),



		IndexOperations: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_storage_index_operations_total",

			Help: "Total number of index operations",

		}, []string{"operation", "index_type"}),



		IndexSize: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_storage_index_size_bytes",

			Help: "Size of indexes in bytes",

		}),

	}



	// Create directories.

	if err := createDirectories(config); err != nil {

		return nil, fmt.Errorf("failed to create storage directories: %w", err)

	}



	// Initialize storage components.

	memoryStore := &MemoryStore{

		series:  make(map[string]*TimeSeries),

		maxSize: 10000, // Store up to 10k series in memory

	}



	diskStore := &DiskStore{

		basePath:    config.DataDirectory,

		partitions:  make(map[string]*DiskPartition),

		indexWriter: NewIndexWriter(config.IndexDirectory),

	}



	compressor := &Compressor{

		enabled: config.CompressionEnabled,

		level:   config.CompressionLevel,

	}



	retentionMgr := &RetentionManager{

		policies: map[string]*RetentionPolicy{

			"default": {

				Name:     "default",

				Duration: config.RetentionPeriod,

				Pattern:  "*",

				Action:   "delete",

				Priority: 1,

			},

		},

		cleanupQueue: make(chan *CleanupTask, 1000),

	}



	compactionMgr := &CompactionManager{

		strategy: CompactionStrategy{

			Type:        "time_based",

			TimeWindows: []time.Duration{1 * time.Hour, 6 * time.Hour, 24 * time.Hour},

			MaxFiles:    100,

		},

		thresholds: CompactionThresholds{

			MinFiles:      5,

			MaxFiles:      50,

			MaxSize:       100 * 1024 * 1024, // 100MB

			FragmentRatio: 0.3,

		},

		pendingTasks: make(chan *CompactionTask, 100),

	}



	writeBuffer := &WriteBuffer{

		buffer:    make([]*DataPoint, 0, config.WriteBufferSize),

		maxSize:   config.WriteBufferSize,

		flushSize: config.BatchSize,

		lastFlush: time.Now(),

	}



	readCache := &ReadCache{

		cache:   make(map[string]*CacheEntry),

		maxSize: config.ReadCacheSize,

	}



	indexManager := &IndexManager{

		indexes: map[string]*Index{

			"labels": {

				Name:    "labels",

				Type:    "hash",

				Keys:    []string{"__name__", "job", "instance"},

				Entries: make(map[string]*IndexEntry),

			},

			"time": {

				Name:    "time",

				Type:    "btree",

				Keys:    []string{"timestamp"},

				Entries: make(map[string]*IndexEntry),

			},

		},

		indexWriter: NewIndexWriter(config.IndexDirectory),

	}



	storageManager := &StorageManager{

		config:        config,

		logger:        logger.WithComponent("storage-manager"),

		memoryStore:   memoryStore,

		diskStore:     diskStore,

		compressor:    compressor,

		retentionMgr:  retentionMgr,

		compactionMgr: compactionMgr,

		writeBuffer:   writeBuffer,

		readCache:     readCache,

		indexManager:  indexManager,

		metrics:       metrics,

		stopCh:        make(chan struct{}),

	}



	return storageManager, nil

}



// Start begins the storage manager operations.

func (sm *StorageManager) Start(ctx context.Context) error {

	if sm.started.Load() {

		return fmt.Errorf("storage manager already started")

	}



	sm.logger.InfoWithContext("Starting storage manager",

		"data_directory", sm.config.DataDirectory,

		"retention_period", sm.config.RetentionPeriod,

		"compression_enabled", sm.config.CompressionEnabled,

		"max_disk_usage_mb", sm.config.MaxDiskUsageMB,

	)



	// Initialize tickers.

	sm.flushTicker = time.NewTicker(sm.config.FlushInterval)

	sm.compactTicker = time.NewTicker(sm.config.CompactionInterval)

	sm.cleanupTicker = time.NewTicker(1 * time.Hour)



	// Start background processes.

	sm.wg.Add(1)

	go sm.runFlushLoop(ctx)



	sm.wg.Add(1)

	go sm.runCompactionLoop(ctx)



	sm.wg.Add(1)

	go sm.runCleanupLoop(ctx)



	sm.wg.Add(1)

	go sm.runMetricsUpdater(ctx)



	// Start compaction workers.

	for i := range 2 {

		sm.wg.Add(1)

		go sm.runCompactionWorker(ctx, i)

	}



	// Start retention cleanup worker.

	sm.wg.Add(1)

	go sm.runRetentionWorker(ctx)



	sm.started.Store(true)

	sm.logger.InfoWithContext("Storage manager started successfully")



	return nil

}



// Stop gracefully stops the storage manager.

func (sm *StorageManager) Stop(ctx context.Context) error {

	if !sm.started.Load() {

		return nil

	}



	sm.logger.InfoWithContext("Stopping storage manager")



	// Stop tickers.

	if sm.flushTicker != nil {

		sm.flushTicker.Stop()

	}

	if sm.compactTicker != nil {

		sm.compactTicker.Stop()

	}

	if sm.cleanupTicker != nil {

		sm.cleanupTicker.Stop()

	}



	// Flush pending data.

	if err := sm.flushBuffer(); err != nil {

		sm.logger.Error("Failed to flush buffer during shutdown", "error", err)

	}



	// Signal stop.

	close(sm.stopCh)



	// Wait for background processes.

	sm.wg.Wait()



	sm.logger.InfoWithContext("Storage manager stopped")



	return nil

}



// WriteDataPoint writes a single data point to storage.

func (sm *StorageManager) WriteDataPoint(seriesName string, point *DataPoint) error {

	start := time.Now()

	defer func() {

		sm.metrics.WriteLatency.WithLabelValues(seriesName, "single").Observe(time.Since(start).Seconds())

		sm.writesCount.Add(1)

	}()



	// Add to write buffer.

	sm.writeBuffer.mu.Lock()

	defer sm.writeBuffer.mu.Unlock()



	sm.writeBuffer.buffer = append(sm.writeBuffer.buffer, point)

	sm.metrics.WriteBufferSize.Set(float64(len(sm.writeBuffer.buffer)))



	// Flush if buffer is full.

	if len(sm.writeBuffer.buffer) >= sm.writeBuffer.flushSize {

		if err := sm.flushBuffer(); err != nil {

			sm.logger.Error("Failed to flush buffer when full", "error", err)

		}

	}



	sm.metrics.WritesTotal.WithLabelValues(seriesName, "success").Inc()



	return nil

}



// WriteDataPoints writes multiple data points to storage.

func (sm *StorageManager) WriteDataPoints(seriesName string, points []*DataPoint) error {

	start := time.Now()

	defer func() {

		sm.metrics.WriteLatency.WithLabelValues(seriesName, "batch").Observe(time.Since(start).Seconds())

		sm.writesCount.Add(uint64(len(points)))

	}()



	sm.writeBuffer.mu.Lock()

	defer sm.writeBuffer.mu.Unlock()



	sm.writeBuffer.buffer = append(sm.writeBuffer.buffer, points...)

	sm.metrics.WriteBufferSize.Set(float64(len(sm.writeBuffer.buffer)))



	// Flush if buffer is approaching capacity.

	if len(sm.writeBuffer.buffer) >= sm.writeBuffer.maxSize-len(points) {

		if err := sm.flushBuffer(); err != nil {

			sm.logger.Error("Failed to flush buffer when approaching capacity", "error", err)

		}

	}



	sm.metrics.WritesTotal.WithLabelValues(seriesName, "success").Add(float64(len(points)))



	return nil

}



// ReadTimeSeries reads time series data within a time range.

func (sm *StorageManager) ReadTimeSeries(seriesName string, start, end time.Time) ([]*DataPoint, error) {

	readStart := time.Now()

	defer func() {

		sm.metrics.ReadLatency.WithLabelValues(seriesName, "combined").Observe(time.Since(readStart).Seconds())

		sm.readsCount.Add(1)

	}()



	// Generate cache key.

	cacheKey := fmt.Sprintf("%s:%d:%d", seriesName, start.Unix(), end.Unix())



	// Check cache first.

	if cachedData := sm.readCache.get(cacheKey); cachedData != nil {

		sm.metrics.ReadCacheHits.Inc()

		sm.metrics.ReadsTotal.WithLabelValues(seriesName, "cache_hit").Inc()

		return cachedData.([]*DataPoint), nil

	}



	sm.metrics.ReadCacheMisses.Inc()

	sm.metrics.ReadsTotal.WithLabelValues(seriesName, "cache_miss").Inc()



	var result []*DataPoint



	// Read from memory store.

	memoryPoints := sm.readFromMemory(seriesName, start, end)

	result = append(result, memoryPoints...)



	// Read from disk store.

	diskPoints := sm.readFromDisk(seriesName, start, end)

	result = append(result, diskPoints...)



	// Sort by timestamp.

	sort.Slice(result, func(i, j int) bool {

		return result[i].Timestamp.Before(result[j].Timestamp)

	})



	// Cache the result.

	sm.readCache.set(cacheKey, result, 5*time.Minute)



	return result, nil

}



// QueryTimeSeries performs a time series query with filters.

func (sm *StorageManager) QueryTimeSeries(query *TimeSeriesQuery) (*TimeSeriesResult, error) {

	start := time.Now()

	defer func() {

		sm.metrics.ReadLatency.WithLabelValues("query", "aggregated").Observe(time.Since(start).Seconds())

	}()



	// Use index to find matching series.

	matchingSeries := sm.indexManager.findMatchingSeries(query.LabelMatchers)



	result := &TimeSeriesResult{

		Query:     query,

		Series:    make(map[string][]*DataPoint),

		StartTime: query.Start,

		EndTime:   query.End,

		QueryTime: time.Since(start),

	}



	// Read data for each matching series.

	for _, seriesName := range matchingSeries {

		points, err := sm.ReadTimeSeries(seriesName, query.Start, query.End)

		if err != nil {

			sm.logger.WarnWithContext("Failed to read series data",

				"series", seriesName,

				"error", err.Error(),

			)

			continue

		}



		// Apply aggregation if specified.

		if query.Aggregation != "" {

			points = sm.aggregatePoints(points, query.Aggregation, query.Step)

		}



		result.Series[seriesName] = points

	}



	return result, nil

}



// flushBuffer flushes the write buffer to persistent storage.

func (sm *StorageManager) flushBuffer() error {

	start := time.Now()

	defer func() {

		sm.metrics.FlushLatency.Observe(time.Since(start).Seconds())

		sm.metrics.FlushOperations.Inc()

	}()



	if len(sm.writeBuffer.buffer) == 0 {

		return nil

	}



	// Group points by series.

	seriesPoints := make(map[string][]*DataPoint)

	for _, point := range sm.writeBuffer.buffer {

		seriesName := sm.generateSeriesName(point.Labels)

		seriesPoints[seriesName] = append(seriesPoints[seriesName], point)

	}



	// Write to memory store first.

	for seriesName, points := range seriesPoints {

		sm.writeToMemory(seriesName, points)

	}



	// Schedule disk writes.

	for seriesName, points := range seriesPoints {

		sm.scheduleWriteToDisk(seriesName, points)

	}



	// Update metrics.

	totalPoints := len(sm.writeBuffer.buffer)

	sm.metrics.DataPointsStored.Add(float64(totalPoints))



	// Clear buffer.

	sm.writeBuffer.buffer = sm.writeBuffer.buffer[:0]

	sm.writeBuffer.lastFlush = time.Now()

	sm.metrics.WriteBufferSize.Set(0)



	sm.logger.DebugWithContext("Buffer flushed",

		"points_written", totalPoints,

		"series_count", len(seriesPoints),

		"duration", time.Since(start),

	)



	return nil

}



// runFlushLoop runs the periodic buffer flush loop.

func (sm *StorageManager) runFlushLoop(ctx context.Context) {

	defer sm.wg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case <-sm.flushTicker.C:

			sm.writeBuffer.mu.Lock()

			if time.Since(sm.writeBuffer.lastFlush) >= sm.config.FlushInterval {

				if err := sm.flushBuffer(); err != nil {

					sm.logger.Error("Failed to flush buffer on timer", "error", err)

				}

			}

			sm.writeBuffer.mu.Unlock()

		}

	}

}



// runCompactionLoop runs the periodic compaction loop.

func (sm *StorageManager) runCompactionLoop(ctx context.Context) {

	defer sm.wg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case <-sm.compactTicker.C:

			sm.scheduleCompaction()

		}

	}

}



// runCleanupLoop runs the periodic cleanup loop.

func (sm *StorageManager) runCleanupLoop(ctx context.Context) {

	defer sm.wg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case <-sm.cleanupTicker.C:

			sm.performCleanup()

		}

	}

}



// runMetricsUpdater updates storage performance metrics.

func (sm *StorageManager) runMetricsUpdater(ctx context.Context) {

	defer sm.wg.Done()



	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case <-ticker.C:

			sm.updateStorageMetrics()

		}

	}

}



// updateStorageMetrics updates storage-related metrics.

func (sm *StorageManager) updateStorageMetrics() {

	// Update disk usage.

	diskUsage := sm.calculateDiskUsage()

	sm.metrics.DiskUsageBytes.Set(float64(diskUsage))



	// Update time series count.

	sm.memoryStore.mu.RLock()

	seriesCount := len(sm.memoryStore.series)

	sm.memoryStore.mu.RUnlock()



	sm.metrics.TimeSeriesCount.Set(float64(seriesCount))



	// Update compression ratio.

	if sm.compressor.enabled {

		ratio := float64(sm.compressor.compressionRatio.Load()) / 100.0

		sm.metrics.CompressionRatio.Set(ratio)

	}



	// Update cache hit ratio.

	hits := sm.readCache.hits.Load()

	misses := sm.readCache.misses.Load()

	total := hits + misses



	if total > 0 {

		hitRatio := float64(hits) / float64(total) * 100.0

		sm.logger.DebugWithContext("Cache performance",

			"hit_ratio", hitRatio,

			"total_requests", total,

		)

	}

}



// Helper methods and background processes.



func (sm *StorageManager) writeToMemory(seriesName string, points []*DataPoint) {

	sm.memoryStore.mu.Lock()

	defer sm.memoryStore.mu.Unlock()



	series, exists := sm.memoryStore.series[seriesName]

	if !exists {

		series = &TimeSeries{

			Name:        seriesName,

			Points:      make([]*DataPoint, 0),

			LastUpdated: time.Now(),

		}

		sm.memoryStore.series[seriesName] = series

	}



	series.Points = append(series.Points, points...)

	series.LastUpdated = time.Now()



	// Trim old points to maintain memory limits.

	sm.trimSeriesPoints(series)

}



func (sm *StorageManager) scheduleWriteToDisk(seriesName string, points []*DataPoint) {

	// This would schedule asynchronous writes to disk.

	go func() {

		if err := sm.writeToDisk(seriesName, points); err != nil {

			sm.logger.ErrorWithContext("Failed to write to disk",

				err,

				"series", seriesName,

				"points_count", len(points),

			)

		}

	}()

}



func (sm *StorageManager) writeToDisk(seriesName string, points []*DataPoint) error {

	// Create partition based on time.

	partition := sm.getOrCreatePartition(points[0].Timestamp)



	// Write data to partition file.

	if err := sm.writeToPartition(partition, seriesName, points); err != nil {

		return fmt.Errorf("failed to write to partition: %w", err)

	}



	// Update index.

	sm.indexManager.updateIndex(seriesName, points)



	return nil

}



func (sm *StorageManager) readFromMemory(seriesName string, start, end time.Time) []*DataPoint {

	sm.memoryStore.mu.RLock()

	defer sm.memoryStore.mu.RUnlock()



	series, exists := sm.memoryStore.series[seriesName]

	if !exists {

		return nil

	}



	var result []*DataPoint

	for _, point := range series.Points {

		if point.Timestamp.After(start) && point.Timestamp.Before(end) {

			result = append(result, point)

		}

	}



	return result

}



func (sm *StorageManager) readFromDisk(seriesName string, start, end time.Time) []*DataPoint {

	// Find relevant partitions.

	partitions := sm.findPartitions(start, end)



	var result []*DataPoint

	for _, partition := range partitions {

		points := sm.readFromPartition(partition, seriesName, start, end)

		result = append(result, points...)

	}



	return result

}



// Placeholder implementations for complex operations.



func (sm *StorageManager) calculateDiskUsage() int64 {

	// Calculate total disk usage.

	var totalSize int64



	sm.diskStore.mu.RLock()

	for _, partition := range sm.diskStore.partitions {

		totalSize += partition.Size

	}

	sm.diskStore.mu.RUnlock()



	return totalSize

}



func (sm *StorageManager) generateSeriesName(labels map[string]string) string {

	// Generate a series name from labels.

	name := labels["__name__"]

	if name == "" {

		name = "unknown"

	}

	return name

}



func (sm *StorageManager) trimSeriesPoints(series *TimeSeries) {

	// Trim old points to maintain memory efficiency.

	maxPoints := 1000 // Keep last 1000 points in memory

	if len(series.Points) > maxPoints {

		series.Points = series.Points[len(series.Points)-maxPoints:]

	}

}



func (sm *StorageManager) getOrCreatePartition(timestamp time.Time) *DiskPartition {

	// Create hourly partitions.

	partitionKey := timestamp.Format("2006010215")



	sm.diskStore.mu.Lock()

	defer sm.diskStore.mu.Unlock()



	partition, exists := sm.diskStore.partitions[partitionKey]

	if !exists {

		partition = &DiskPartition{

			StartTime: timestamp.Truncate(time.Hour),

			EndTime:   timestamp.Truncate(time.Hour).Add(time.Hour),

			FilePath:  filepath.Join(sm.diskStore.basePath, partitionKey+".data"),

		}

		sm.diskStore.partitions[partitionKey] = partition

	}



	return partition

}



func (sm *StorageManager) findPartitions(start, end time.Time) []*DiskPartition {

	sm.diskStore.mu.RLock()

	defer sm.diskStore.mu.RUnlock()



	var partitions []*DiskPartition

	for _, partition := range sm.diskStore.partitions {

		if partition.EndTime.After(start) && partition.StartTime.Before(end) {

			partitions = append(partitions, partition)

		}

	}



	return partitions

}



func (sm *StorageManager) writeToPartition(partition *DiskPartition, seriesName string, points []*DataPoint) error {

	// Open or create partition file.

	file, err := os.OpenFile(partition.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)

	if err != nil {

		return fmt.Errorf("failed to open partition file: %w", err)

	}

	defer file.Close()



	var writer io.Writer = file



	// Apply compression if enabled.

	if sm.compressor.enabled {

		gzipWriter, err := gzip.NewWriterLevel(file, sm.compressor.level)

		if err != nil {

			return fmt.Errorf("failed to create gzip writer: %w", err)

		}

		defer gzipWriter.Close()

		writer = gzipWriter

		partition.Compressed = true

	}



	// Write binary data.

	for _, point := range points {

		if err := sm.writeBinaryDataPoint(writer, point); err != nil {

			return fmt.Errorf("failed to write data point: %w", err)

		}

	}



	// Update partition metadata.

	partition.PointCount += int64(len(points))



	return nil

}



func (sm *StorageManager) writeBinaryDataPoint(writer io.Writer, point *DataPoint) error {

	// Write timestamp.

	timestamp := point.Timestamp.UnixNano()

	if err := binary.Write(writer, binary.LittleEndian, timestamp); err != nil {

		return err

	}



	// Write value.

	if err := binary.Write(writer, binary.LittleEndian, point.Value); err != nil {

		return err

	}



	// Write labels (simplified - in production would use more efficient encoding).

	labelCount := uint32(len(point.Labels))

	if err := binary.Write(writer, binary.LittleEndian, labelCount); err != nil {

		return err

	}



	for key, value := range point.Labels {

		keyLen := uint32(len(key))

		valueLen := uint32(len(value))



		if err := binary.Write(writer, binary.LittleEndian, keyLen); err != nil {

			return err

		}

		if _, err := writer.Write([]byte(key)); err != nil {

			return err

		}

		if err := binary.Write(writer, binary.LittleEndian, valueLen); err != nil {

			return err

		}

		if _, err := writer.Write([]byte(value)); err != nil {

			return err

		}

	}



	return nil

}



func (sm *StorageManager) readFromPartition(partition *DiskPartition, seriesName string, start, end time.Time) []*DataPoint {

	// Placeholder implementation.

	return []*DataPoint{}

}



func (sm *StorageManager) performCleanup() {

	// Remove expired data based on retention policies.

	cutoff := time.Now().Add(-sm.config.RetentionPeriod)



	// Clean up memory store.

	sm.memoryStore.mu.Lock()

	for name, series := range sm.memoryStore.series {

		filteredPoints := make([]*DataPoint, 0)

		for _, point := range series.Points {

			if point.Timestamp.After(cutoff) {

				filteredPoints = append(filteredPoints, point)

			}

		}

		series.Points = filteredPoints



		// Remove empty series.

		if len(series.Points) == 0 {

			delete(sm.memoryStore.series, name)

		}

	}

	sm.memoryStore.mu.Unlock()



	// Clean up disk partitions.

	sm.diskStore.mu.Lock()

	for key, partition := range sm.diskStore.partitions {

		if partition.EndTime.Before(cutoff) {

			// Remove partition file.

			os.Remove(partition.FilePath)

			delete(sm.diskStore.partitions, key)

		}

	}

	sm.diskStore.mu.Unlock()



	sm.logger.InfoWithContext("Cleanup completed", "cutoff_time", cutoff)

}



func (sm *StorageManager) scheduleCompaction() {

	// Schedule compaction based on thresholds.

	sm.diskStore.mu.RLock()

	partitionCount := len(sm.diskStore.partitions)

	sm.diskStore.mu.RUnlock()



	if partitionCount >= sm.compactionMgr.thresholds.MinFiles {

		task := &CompactionTask{

			Type:      "time_based",

			Priority:  1,

			CreatedAt: time.Now(),

		}



		select {

		case sm.compactionMgr.pendingTasks <- task:

			sm.logger.DebugWithContext("Compaction task scheduled",

				"partition_count", partitionCount,

			)

		default:

			sm.logger.WarnWithContext("Compaction queue full, skipping task")

		}

	}

}



func (sm *StorageManager) runCompactionWorker(ctx context.Context, workerID int) {

	defer sm.wg.Done()



	sm.logger.DebugWithContext("Starting compaction worker", "worker_id", workerID)



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case task := <-sm.compactionMgr.pendingTasks:

			sm.performCompaction(task)

		}

	}

}



func (sm *StorageManager) performCompaction(task *CompactionTask) {

	start := time.Now()

	defer func() {

		sm.metrics.CompactionLatency.Observe(time.Since(start).Seconds())

		sm.metrics.CompactionsTotal.Inc()

		sm.compactionsCount.Add(1)

	}()



	sm.logger.InfoWithContext("Performing compaction", "task_type", task.Type)



	// Simplified compaction - in production would merge multiple partitions.

	// into larger, more efficient files.



	sizeBefore := sm.calculateDiskUsage()



	// Perform actual compaction work here.

	// This is a placeholder for the complex compaction logic.



	sizeAfter := sm.calculateDiskUsage()

	savings := sizeBefore - sizeAfter



	sm.metrics.CompactionSavings.Set(float64(savings))



	sm.logger.InfoWithContext("Compaction completed",

		"duration", time.Since(start),

		"size_before", sizeBefore,

		"size_after", sizeAfter,

		"savings", savings,

	)

}



func (sm *StorageManager) runRetentionWorker(ctx context.Context) {

	defer sm.wg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-sm.stopCh:

			return

		case task := <-sm.retentionMgr.cleanupQueue:

			sm.processRetentionTask(task)

		}

	}

}



func (sm *StorageManager) processRetentionTask(task *CleanupTask) {

	// Process retention cleanup tasks.

	sm.logger.DebugWithContext("Processing retention task",

		"task_type", task.Type,

		"resource", task.Resource,

	)

}



func (sm *StorageManager) aggregatePoints(points []*DataPoint, aggregation string, step time.Duration) []*DataPoint {

	// Implement aggregation (sum, avg, max, min, etc.).

	// This is a simplified placeholder.

	return points

}



func (rc *ReadCache) get(key string) interface{} {

	rc.mu.RLock()

	defer rc.mu.RUnlock()



	entry, exists := rc.cache[key]

	if !exists {

		rc.misses.Add(1)

		return nil

	}



	// Check TTL.

	if time.Since(entry.Timestamp) > entry.TTL {

		delete(rc.cache, key)

		rc.misses.Add(1)

		return nil

	}



	rc.hits.Add(1)

	return entry.Data

}



func (rc *ReadCache) set(key string, data interface{}, ttl time.Duration) {

	rc.mu.Lock()

	defer rc.mu.Unlock()



	// Implement LRU eviction if cache is full.

	if len(rc.cache) >= rc.maxSize {

		rc.evictLRU()

	}



	rc.cache[key] = &CacheEntry{

		Data:      data,

		Timestamp: time.Now(),

		TTL:       ttl,

	}

}



func (rc *ReadCache) evictLRU() {

	// Simple eviction - remove oldest entry.

	var oldestKey string

	var oldestTime time.Time



	for key, entry := range rc.cache {

		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {

			oldestKey = key

			oldestTime = entry.Timestamp

		}

	}



	if oldestKey != "" {

		delete(rc.cache, oldestKey)

	}

}



// Helper functions.

func createDirectories(config *StorageConfig) error {

	dirs := []string{

		config.DataDirectory,

		config.IndexDirectory,

		config.BackupDirectory,

	}



	for _, dir := range dirs {

		if err := os.MkdirAll(dir, 0o755); err != nil {

			return fmt.Errorf("failed to create directory %s: %w", dir, err)

		}

	}



	return nil

}



// Placeholder types and functions.

type TimeSeriesQuery struct {

	LabelMatchers map[string]string

	Start         time.Time

	End           time.Time

	Step          time.Duration

	Aggregation   string

}



// TimeSeriesResult represents a timeseriesresult.

type TimeSeriesResult struct {

	Query     *TimeSeriesQuery

	Series    map[string][]*DataPoint

	StartTime time.Time

	EndTime   time.Time

	QueryTime time.Duration

}



// CleanupTask represents a cleanuptask.

type CleanupTask struct {

	Type     string

	Resource string

}



// CompactionTask represents a compactiontask.

type CompactionTask struct {

	Type      string

	Priority  int

	CreatedAt time.Time

}



// CompactionWorker represents a compactionworker.

type (

	CompactionWorker struct{}

	// IndexWriter represents a indexwriter.

	IndexWriter struct{}

)



// NewIndexWriter performs newindexwriter operation.

func NewIndexWriter(directory string) *IndexWriter { return &IndexWriter{} }



func (im *IndexManager) findMatchingSeries(matchers map[string]string) []string {

	// Placeholder implementation.

	return []string{}

}



func (im *IndexManager) updateIndex(seriesName string, points []*DataPoint) {

	// Placeholder implementation.

}


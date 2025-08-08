//go:build go1.24

package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/bytedance/sonic"
	"github.com/valyala/fastjson"
	"k8s.io/klog/v2"
)

// OptimizedJSONProcessor provides high-performance JSON operations with Go 1.24+ optimizations
type OptimizedJSONProcessor struct {
	schemaCache    *JSONSchemaCache
	encoderPool    *EncoderPool
	decoderPool    *DecoderPool
	streamPool     *StreamPool
	bufferPool     *BufferPool
	parserPool     *fastjson.ParserPool
	metrics       *JSONMetrics
	config        *JSONConfig
	workerPool    *JSONWorkerPool
	mu            sync.RWMutex
}

// JSONConfig contains JSON processing configuration
type JSONConfig struct {
	EnableSchemaOptimization bool
	EnableStreaming         bool
	EnableConcurrency       bool
	EnableSIMD              bool
	MaxConcurrentOperations int
	StreamingBufferSize     int
	SchemaValidation        bool
	CompressionEnabled      bool
	PoolSize                int
	MaxObjectSize           int64
	MetricsInterval         time.Duration
}

// JSONSchemaCache caches JSON schemas for optimization
type JSONSchemaCache struct {
	schemas        map[string]*CachedSchema
	mu             sync.RWMutex
	hitCount       int64
	missCount      int64
	validationTime int64
}

// CachedSchema represents a cached JSON schema with optimization metadata
type CachedSchema struct {
	Schema         interface{}
	FieldMap       map[string]FieldInfo
	OptimizedPath  []string
	LastUsed       time.Time
	ValidationFunc func([]byte) error
	SerializerFunc func(interface{}) ([]byte, error)
}

// FieldInfo contains metadata about JSON fields for optimization
type FieldInfo struct {
	Name        string
	Type        reflect.Type
	Offset      uintptr
	Size        uintptr
	IsPointer   bool
	IsRequired  bool
	Validation  func(interface{}) bool
	Serializer  func(interface{}) []byte
}

// EncoderPool manages a pool of JSON encoders
type EncoderPool struct {
	pool       sync.Pool
	created    int64
	reused     int64
	encodeTime int64
}

// DecoderPool manages a pool of JSON decoders
type DecoderPool struct {
	pool       sync.Pool
	created    int64
	reused     int64
	decodeTime int64
}

// StreamPool manages streaming JSON processors
type StreamPool struct {
	encoders   chan *json.Encoder
	decoders   chan *json.Decoder
	buffers    chan *bytes.Buffer
	maxSize    int
	created    int64
	reused     int64
}

// JSONWorkerPool manages concurrent JSON processing
type JSONWorkerPool struct {
	workers     chan chan *JSONTask
	workQueue   chan *JSONTask
	maxWorkers  int
	activeTasks int64
	completedTasks int64
	failedTasks int64
	shutdown    chan bool
	wg          sync.WaitGroup
}

// JSONTask represents a JSON processing task
type JSONTask struct {
	Operation   JSONOperation
	Data        []byte
	Target      interface{}
	Result      chan *JSONResult
	Context     context.Context
	StartTime   time.Time
	Schema      *CachedSchema
}

// JSONOperation defines the type of JSON operation
type JSONOperation int

const (
	JSONOperationMarshal JSONOperation = iota
	JSONOperationUnmarshal
	JSONOperationValidate
	JSONOperationStream
)

// JSONResult contains the result of a JSON operation
type JSONResult struct {
	Data      []byte
	Error     error
	Duration  time.Duration
	BytesProcessed int64
}

// JSONMetrics tracks JSON processing performance
type JSONMetrics struct {
	MarshalCount      int64
	UnmarshalCount    int64
	ValidationCount   int64
	StreamingCount    int64
	TotalProcessTime  int64 // nanoseconds
	ErrorCount        int64
	BytesProcessed    int64
	SchemaHitRate     float64
	PoolHitRate       float64
	ConcurrencyLevel  int64
	SIMDOperations    int64
	CompressionRatio  float64
}

// NewOptimizedJSONProcessor creates a new optimized JSON processor
func NewOptimizedJSONProcessor(config *JSONConfig) *OptimizedJSONProcessor {
	if config == nil {
		config = DefaultJSONConfig()
	}

	processor := &OptimizedJSONProcessor{
		schemaCache: NewJSONSchemaCache(),
		encoderPool: NewEncoderPool(),
		decoderPool: NewDecoderPool(),
		streamPool:  NewStreamPool(config.PoolSize),
		bufferPool:  NewBufferPool(),
		parserPool:  &fastjson.ParserPool{},
		metrics:    &JSONMetrics{},
		config:     config,
	}

	if config.EnableConcurrency {
		processor.workerPool = NewJSONWorkerPool(config.MaxConcurrentOperations)
	}

	// Start metrics collection
	processor.startMetricsCollection()

	return processor
}

// DefaultJSONConfig returns default JSON processing configuration
func DefaultJSONConfig() *JSONConfig {
	return &JSONConfig{
		EnableSchemaOptimization: true,
		EnableStreaming:         true,
		EnableConcurrency:       true,
		EnableSIMD:              true,
		MaxConcurrentOperations: runtime.NumCPU() * 2,
		StreamingBufferSize:     64 * 1024, // 64KB
		SchemaValidation:        true,
		CompressionEnabled:      false,
		PoolSize:                100,
		MaxObjectSize:           10 * 1024 * 1024, // 10MB
		MetricsInterval:         30 * time.Second,
	}
}

// NewJSONSchemaCache creates a new schema cache
func NewJSONSchemaCache() *JSONSchemaCache {
	return &JSONSchemaCache{
		schemas: make(map[string]*CachedSchema),
	}
}

// NewEncoderPool creates a new encoder pool
func NewEncoderPool() *EncoderPool {
	return &EncoderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return json.NewEncoder(nil)
			},
		},
	}
}

// NewDecoderPool creates a new decoder pool
func NewDecoderPool() *DecoderPool {
	return &DecoderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return json.NewDecoder(nil)
			},
		},
	}
}

// NewStreamPool creates a new streaming pool
func NewStreamPool(size int) *StreamPool {
	return &StreamPool{
		encoders: make(chan *json.Encoder, size),
		decoders: make(chan *json.Decoder, size),
		buffers:  make(chan *bytes.Buffer, size),
		maxSize:  size,
	}
}

// NewJSONWorkerPool creates a new worker pool for concurrent JSON processing
func NewJSONWorkerPool(maxWorkers int) *JSONWorkerPool {
	pool := &JSONWorkerPool{
		workers:    make(chan chan *JSONTask, maxWorkers),
		workQueue:  make(chan *JSONTask, maxWorkers*2),
		maxWorkers: maxWorkers,
		shutdown:   make(chan bool),
	}

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	// Start dispatcher
	go pool.dispatcher()

	return pool
}

// worker processes JSON tasks
func (jwp *JSONWorkerPool) worker() {
	defer jwp.wg.Done()

	// Create worker channel
	workChan := make(chan *JSONTask)

	for {
		select {
		case jwp.workers <- workChan:
			// Worker is available, wait for task
			select {
			case task := <-workChan:
				jwp.processTask(task)
			case <-jwp.shutdown:
				return
			}
		case <-jwp.shutdown:
			return
		}
	}
}

// dispatcher assigns tasks to available workers
func (jwp *JSONWorkerPool) dispatcher() {
	for {
		select {
		case task := <-jwp.workQueue:
			// Get available worker
			select {
			case workerChan := <-jwp.workers:
				atomic.AddInt64(&jwp.activeTasks, 1)
				workerChan <- task
			case <-time.After(5 * time.Second):
				// No workers available, handle timeout
				task.Result <- &JSONResult{
					Error: fmt.Errorf("worker pool timeout"),
				}
				atomic.AddInt64(&jwp.failedTasks, 1)
			}
		case <-jwp.shutdown:
			return
		}
	}
}

// processTask processes a single JSON task
func (jwp *JSONWorkerPool) processTask(task *JSONTask) {
	defer func() {
		atomic.AddInt64(&jwp.activeTasks, -1)
		atomic.AddInt64(&jwp.completedTasks, 1)
		if r := recover(); r != nil {
			task.Result <- &JSONResult{
				Error: fmt.Errorf("JSON processing panic: %v", r),
			}
			atomic.AddInt64(&jwp.failedTasks, 1)
		}
	}()

	start := time.Now()
	var result *JSONResult

	switch task.Operation {
	case JSONOperationMarshal:
		result = jwp.performMarshal(task)
	case JSONOperationUnmarshal:
		result = jwp.performUnmarshal(task)
	case JSONOperationValidate:
		result = jwp.performValidation(task)
	case JSONOperationStream:
		result = jwp.performStreaming(task)
	default:
		result = &JSONResult{
			Error: fmt.Errorf("unknown JSON operation: %v", task.Operation),
		}
	}

	result.Duration = time.Since(start)
	task.Result <- result
}

// performMarshal performs JSON marshaling
func (jwp *JSONWorkerPool) performMarshal(task *JSONTask) *JSONResult {
	// Use sonic for high-performance marshaling when available
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		data, err := sonic.Marshal(task.Target)
		if err != nil {
			return &JSONResult{Error: err}
		}
		return &JSONResult{
			Data:           data,
			BytesProcessed: int64(len(data)),
		}
	}

	// Fallback to standard JSON
	data, err := json.Marshal(task.Target)
	if err != nil {
		return &JSONResult{Error: err}
	}

	return &JSONResult{
		Data:           data,
		BytesProcessed: int64(len(data)),
	}
}

// performUnmarshal performs JSON unmarshaling
func (jwp *JSONWorkerPool) performUnmarshal(task *JSONTask) *JSONResult {
	// Use sonic for high-performance unmarshaling when available
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		err := sonic.Unmarshal(task.Data, task.Target)
		if err != nil {
			return &JSONResult{Error: err}
		}
		return &JSONResult{
			BytesProcessed: int64(len(task.Data)),
		}
	}

	// Fallback to standard JSON
	err := json.Unmarshal(task.Data, task.Target)
	if err != nil {
		return &JSONResult{Error: err}
	}

	return &JSONResult{
		BytesProcessed: int64(len(task.Data)),
	}
}

// performValidation performs JSON validation
func (jwp *JSONWorkerPool) performValidation(task *JSONTask) *JSONResult {
	if task.Schema != nil && task.Schema.ValidationFunc != nil {
		err := task.Schema.ValidationFunc(task.Data)
		return &JSONResult{
			Error:          err,
			BytesProcessed: int64(len(task.Data)),
		}
	}

	// Basic JSON validation
	if !json.Valid(task.Data) {
		return &JSONResult{
			Error: fmt.Errorf("invalid JSON"),
		}
	}

	return &JSONResult{
		BytesProcessed: int64(len(task.Data)),
	}
}

// performStreaming performs streaming JSON processing
func (jwp *JSONWorkerPool) performStreaming(task *JSONTask) *JSONResult {
	// Streaming implementation would depend on specific requirements
	// This is a placeholder
	return &JSONResult{
		BytesProcessed: int64(len(task.Data)),
	}
}

// MarshalOptimized performs optimized JSON marshaling
func (processor *OptimizedJSONProcessor) MarshalOptimized(ctx context.Context, obj interface{}) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&processor.metrics.MarshalCount, 1)
		atomic.AddInt64(&processor.metrics.TotalProcessTime, duration.Nanoseconds())
	}()

	if processor.config.EnableConcurrency && processor.workerPool != nil {
		return processor.marshalConcurrent(ctx, obj)
	}

	return processor.marshalSequential(obj)
}

// marshalSequential performs sequential marshaling
func (processor *OptimizedJSONProcessor) marshalSequential(obj interface{}) ([]byte, error) {
	// Check if we have a cached schema for optimization
	schemaKey := processor.getSchemaKey(obj)
	if schema := processor.schemaCache.Get(schemaKey); schema != nil {
		if schema.SerializerFunc != nil {
			return schema.SerializerFunc(obj)
		}
	}

	// Use sonic for better performance when available
	if processor.config.EnableSIMD && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		data, err := sonic.Marshal(obj)
		if err == nil {
			atomic.AddInt64(&processor.metrics.SIMDOperations, 1)
			atomic.AddInt64(&processor.metrics.BytesProcessed, int64(len(data)))
			return data, nil
		}
	}

	// Fallback to standard JSON with pooled encoder
	encoder := processor.encoderPool.Get()
	defer processor.encoderPool.Put(encoder)

	buffer := processor.bufferPool.GetBuffer(1024)
	defer processor.bufferPool.PutBuffer(buffer)

	buf := bytes.NewBuffer(buffer[:0])
	encoder.(*json.Encoder).Reset(buf)

	err := encoder.(*json.Encoder).Encode(obj)
	if err != nil {
		atomic.AddInt64(&processor.metrics.ErrorCount, 1)
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	atomic.AddInt64(&processor.metrics.BytesProcessed, int64(len(result)))

	return result, nil
}

// marshalConcurrent performs concurrent marshaling
func (processor *OptimizedJSONProcessor) marshalConcurrent(ctx context.Context, obj interface{}) ([]byte, error) {
	task := &JSONTask{
		Operation: JSONOperationMarshal,
		Target:    obj,
		Result:    make(chan *JSONResult, 1),
		Context:   ctx,
		StartTime: time.Now(),
	}

	// Add schema information if available
	schemaKey := processor.getSchemaKey(obj)
	task.Schema = processor.schemaCache.Get(schemaKey)

	select {
	case processor.workerPool.workQueue <- task:
		// Task queued successfully
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("marshal task queue timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for result
	select {
	case result := <-task.Result:
		if result.Error != nil {
			atomic.AddInt64(&processor.metrics.ErrorCount, 1)
			return nil, result.Error
		}
		atomic.AddInt64(&processor.metrics.BytesProcessed, result.BytesProcessed)
		return result.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UnmarshalOptimized performs optimized JSON unmarshaling
func (processor *OptimizedJSONProcessor) UnmarshalOptimized(ctx context.Context, data []byte, target interface{}) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&processor.metrics.UnmarshalCount, 1)
		atomic.AddInt64(&processor.metrics.TotalProcessTime, duration.Nanoseconds())
	}()

	if int64(len(data)) > processor.config.MaxObjectSize {
		return fmt.Errorf("JSON object too large: %d bytes", len(data))
	}

	if processor.config.EnableConcurrency && processor.workerPool != nil {
		return processor.unmarshalConcurrent(ctx, data, target)
	}

	return processor.unmarshalSequential(data, target)
}

// unmarshalSequential performs sequential unmarshaling
func (processor *OptimizedJSONProcessor) unmarshalSequential(data []byte, target interface{}) error {
	// Use sonic for better performance when available
	if processor.config.EnableSIMD && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		err := sonic.Unmarshal(data, target)
		if err == nil {
			atomic.AddInt64(&processor.metrics.SIMDOperations, 1)
			atomic.AddInt64(&processor.metrics.BytesProcessed, int64(len(data)))
			return nil
		}
	}

	// Fallback to standard JSON with pooled decoder
	decoder := processor.decoderPool.Get()
	defer processor.decoderPool.Put(decoder)

	buf := bytes.NewReader(data)
	decoder.(*json.Decoder).Reset(buf)

	err := decoder.(*json.Decoder).Decode(target)
	if err != nil {
		atomic.AddInt64(&processor.metrics.ErrorCount, 1)
		return err
	}

	atomic.AddInt64(&processor.metrics.BytesProcessed, int64(len(data)))
	return nil
}

// unmarshalConcurrent performs concurrent unmarshaling
func (processor *OptimizedJSONProcessor) unmarshalConcurrent(ctx context.Context, data []byte, target interface{}) error {
	task := &JSONTask{
		Operation: JSONOperationUnmarshal,
		Data:      data,
		Target:    target,
		Result:    make(chan *JSONResult, 1),
		Context:   ctx,
		StartTime: time.Now(),
	}

	select {
	case processor.workerPool.workQueue <- task:
		// Task queued successfully
	case <-time.After(5 * time.Second):
		return fmt.Errorf("unmarshal task queue timeout")
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for result
	select {
	case result := <-task.Result:
		if result.Error != nil {
			atomic.AddInt64(&processor.metrics.ErrorCount, 1)
			return result.Error
		}
		atomic.AddInt64(&processor.metrics.BytesProcessed, result.BytesProcessed)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StreamingUnmarshal performs streaming JSON unmarshaling
func (processor *OptimizedJSONProcessor) StreamingUnmarshal(ctx context.Context, reader io.Reader, callback func(interface{}) error) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&processor.metrics.StreamingCount, 1)
		atomic.AddInt64(&processor.metrics.TotalProcessTime, duration.Nanoseconds())
	}()

	decoder := json.NewDecoder(reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var obj interface{}
		err := decoder.Decode(&obj)
		if err == io.EOF {
			return nil // End of stream
		}
		if err != nil {
			atomic.AddInt64(&processor.metrics.ErrorCount, 1)
			return err
		}

		if err := callback(obj); err != nil {
			return err
		}
	}
}

// Get retrieves an encoder from the pool
func (ep *EncoderPool) Get() interface{} {
	atomic.AddInt64(&ep.reused, 1)
	return ep.pool.Get()
}

// Put returns an encoder to the pool
func (ep *EncoderPool) Put(encoder interface{}) {
	ep.pool.Put(encoder)
}

// Get retrieves a decoder from the pool
func (dp *DecoderPool) Get() interface{} {
	atomic.AddInt64(&dp.reused, 1)
	return dp.pool.Get()
}

// Put returns a decoder to the pool
func (dp *DecoderPool) Put(decoder interface{}) {
	dp.pool.Put(decoder)
}

// Get retrieves a schema from the cache
func (jsc *JSONSchemaCache) Get(key string) *CachedSchema {
	jsc.mu.RLock()
	defer jsc.mu.RUnlock()

	if schema, exists := jsc.schemas[key]; exists {
		atomic.AddInt64(&jsc.hitCount, 1)
		schema.LastUsed = time.Now()
		return schema
	}

	atomic.AddInt64(&jsc.missCount, 1)
	return nil
}

// Set stores a schema in the cache
func (jsc *JSONSchemaCache) Set(key string, schema *CachedSchema) {
	jsc.mu.Lock()
	defer jsc.mu.Unlock()
	jsc.schemas[key] = schema
}

// getSchemaKey generates a cache key for the given object type
func (processor *OptimizedJSONProcessor) getSchemaKey(obj interface{}) string {
	if obj == nil {
		return "nil"
	}
	
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	return t.String()
}

// startMetricsCollection starts background metrics collection
func (processor *OptimizedJSONProcessor) startMetricsCollection() {
	go func() {
		ticker := time.NewTicker(processor.config.MetricsInterval)
		defer ticker.Stop()

		for range ticker.C {
			processor.updateMetrics()
		}
	}()
}

// updateMetrics updates derived metrics
func (processor *OptimizedJSONProcessor) updateMetrics() {
	// Update schema hit rate
	hits := atomic.LoadInt64(&processor.schemaCache.hitCount)
	misses := atomic.LoadInt64(&processor.schemaCache.missCount)
	total := hits + misses
	if total > 0 {
		processor.metrics.SchemaHitRate = float64(hits) / float64(total)
	}

	// Update pool hit rate (simplified calculation)
	encoderReused := atomic.LoadInt64(&processor.encoderPool.reused)
	decoderReused := atomic.LoadInt64(&processor.decoderPool.reused)
	totalReused := encoderReused + decoderReused
	marshalCount := atomic.LoadInt64(&processor.metrics.MarshalCount)
	unmarshalCount := atomic.LoadInt64(&processor.metrics.UnmarshalCount)
	totalOps := marshalCount + unmarshalCount
	if totalOps > 0 {
		processor.metrics.PoolHitRate = float64(totalReused) / float64(totalOps)
	}

	// Update concurrency level
	if processor.workerPool != nil {
		atomic.StoreInt64(&processor.metrics.ConcurrencyLevel, atomic.LoadInt64(&processor.workerPool.activeTasks))
	}
}

// GetMetrics returns current JSON processing metrics
func (processor *OptimizedJSONProcessor) GetMetrics() JSONMetrics {
	return JSONMetrics{
		MarshalCount:     atomic.LoadInt64(&processor.metrics.MarshalCount),
		UnmarshalCount:   atomic.LoadInt64(&processor.metrics.UnmarshalCount),
		ValidationCount:  atomic.LoadInt64(&processor.metrics.ValidationCount),
		StreamingCount:   atomic.LoadInt64(&processor.metrics.StreamingCount),
		TotalProcessTime: atomic.LoadInt64(&processor.metrics.TotalProcessTime),
		ErrorCount:       atomic.LoadInt64(&processor.metrics.ErrorCount),
		BytesProcessed:   atomic.LoadInt64(&processor.metrics.BytesProcessed),
		SchemaHitRate:    processor.metrics.SchemaHitRate,
		PoolHitRate:      processor.metrics.PoolHitRate,
		ConcurrencyLevel: atomic.LoadInt64(&processor.metrics.ConcurrencyLevel),
		SIMDOperations:   atomic.LoadInt64(&processor.metrics.SIMDOperations),
		CompressionRatio: processor.metrics.CompressionRatio,
	}
}

// GetAverageProcessingTime returns the average processing time in microseconds
func (processor *OptimizedJSONProcessor) GetAverageProcessingTime() float64 {
	totalOps := atomic.LoadInt64(&processor.metrics.MarshalCount) +
		atomic.LoadInt64(&processor.metrics.UnmarshalCount) +
		atomic.LoadInt64(&processor.metrics.ValidationCount) +
		atomic.LoadInt64(&processor.metrics.StreamingCount)

	if totalOps == 0 {
		return 0
	}

	totalTime := atomic.LoadInt64(&processor.metrics.TotalProcessTime)
	return float64(totalTime) / float64(totalOps) / 1000 // Convert to microseconds
}

// Shutdown gracefully shuts down the JSON processor
func (processor *OptimizedJSONProcessor) Shutdown(ctx context.Context) error {
	// Stop worker pool
	if processor.workerPool != nil {
		close(processor.workerPool.shutdown)
		processor.workerPool.wg.Wait()
	}

	// Log final metrics
	metrics := processor.GetMetrics()
	klog.Infof("JSON processor shutdown - Total operations: %d, Avg processing time: %.2fμs, Error rate: %.2f%%",
		metrics.MarshalCount+metrics.UnmarshalCount+metrics.ValidationCount+metrics.StreamingCount,
		processor.GetAverageProcessingTime(),
		float64(metrics.ErrorCount)/float64(metrics.MarshalCount+metrics.UnmarshalCount)*100,
	)

	return nil
}

// ResetMetrics resets all performance metrics
func (processor *OptimizedJSONProcessor) ResetMetrics() {
	atomic.StoreInt64(&processor.metrics.MarshalCount, 0)
	atomic.StoreInt64(&processor.metrics.UnmarshalCount, 0)
	atomic.StoreInt64(&processor.metrics.ValidationCount, 0)
	atomic.StoreInt64(&processor.metrics.StreamingCount, 0)
	atomic.StoreInt64(&processor.metrics.TotalProcessTime, 0)
	atomic.StoreInt64(&processor.metrics.ErrorCount, 0)
	atomic.StoreInt64(&processor.metrics.BytesProcessed, 0)
	atomic.StoreInt64(&processor.metrics.ConcurrencyLevel, 0)
	atomic.StoreInt64(&processor.metrics.SIMDOperations, 0)
	processor.metrics.SchemaHitRate = 0
	processor.metrics.PoolHitRate = 0
	processor.metrics.CompressionRatio = 0

	// Reset cache metrics
	atomic.StoreInt64(&processor.schemaCache.hitCount, 0)
	atomic.StoreInt64(&processor.schemaCache.missCount, 0)

	// Reset pool metrics
	atomic.StoreInt64(&processor.encoderPool.reused, 0)
	atomic.StoreInt64(&processor.decoderPool.reused, 0)
}
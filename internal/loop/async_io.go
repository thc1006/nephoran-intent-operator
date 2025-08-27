package loop

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// AsyncIOManager handles asynchronous I/O operations with batching and buffering
type AsyncIOManager struct {
	// Write buffering
	writeBuffer      *BatchBuffer
	writeWorkers     int
	writeChan        chan *WriteRequest
	
	// Read caching
	readCache        *LRUCache
	readAheadSize    int
	
	// Metrics
	metrics          *IOMetrics
	
	// Lifecycle
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// WriteRequest represents an async write operation
type WriteRequest struct {
	Path      string
	Data      []byte
	Mode      os.FileMode
	Callback  func(error)
	Timestamp time.Time
}

// BatchBuffer accumulates writes for batch processing
type BatchBuffer struct {
	mu            sync.Mutex
	buffer        []*WriteRequest
	maxSize       int
	maxAge        time.Duration
	flushTimer    *time.Timer
	flushCallback func([]*WriteRequest)
}

// IOMetrics tracks I/O performance
type IOMetrics struct {
	mu              sync.RWMutex
	TotalReads      atomic.Int64
	TotalWrites     atomic.Int64
	CacheHits       atomic.Int64
	CacheMisses     atomic.Int64
	BytesRead       atomic.Int64
	BytesWritten    atomic.Int64
	ReadLatency     *RingBuffer
	WriteLatency    *RingBuffer
	BatchedWrites   atomic.Int64
	FailedWrites    atomic.Int64
}

// AsyncIOConfig configures the async I/O manager
type AsyncIOConfig struct {
	WriteWorkers    int           // Number of concurrent write workers
	WriteBatchSize  int           // Maximum writes to batch
	WriteBatchAge   time.Duration // Maximum age of batched writes
	ReadCacheSize   int           // Number of files to cache
	ReadAheadSize   int           // Bytes to read ahead
	BufferSize      int           // I/O buffer size
}

// DefaultAsyncIOConfig returns optimized default configuration
func DefaultAsyncIOConfig() *AsyncIOConfig {
	return &AsyncIOConfig{
		WriteWorkers:   4,
		WriteBatchSize: 10,
		WriteBatchAge:  100 * time.Millisecond,
		ReadCacheSize:  100,
		ReadAheadSize:  64 * 1024, // 64KB read-ahead
		BufferSize:     32 * 1024, // 32KB buffer
	}
}

// NewAsyncIOManager creates a new async I/O manager
func NewAsyncIOManager(config *AsyncIOConfig) *AsyncIOManager {
	if config == nil {
		config = DefaultAsyncIOConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &AsyncIOManager{
		writeWorkers:  config.WriteWorkers,
		writeChan:     make(chan *WriteRequest, config.WriteBatchSize*2),
		readCache:     NewLRUCache(config.ReadCacheSize),
		readAheadSize: config.ReadAheadSize,
		ctx:           ctx,
		cancel:        cancel,
		metrics: &IOMetrics{
			ReadLatency:  NewRingBuffer(1000),
			WriteLatency: NewRingBuffer(1000),
		},
	}
	
	// Initialize write buffer
	manager.writeBuffer = &BatchBuffer{
		buffer:  make([]*WriteRequest, 0, config.WriteBatchSize),
		maxSize: config.WriteBatchSize,
		maxAge:  config.WriteBatchAge,
		flushCallback: func(batch []*WriteRequest) {
			manager.processBatch(batch)
		},
	}
	
	// Start write workers
	for i := 0; i < config.WriteWorkers; i++ {
		manager.wg.Add(1)
		go manager.writeWorker(i)
	}
	
	// Start batch processor
	manager.wg.Add(1)
	go manager.batchProcessor()
	
	return manager
}

// ReadFileAsync reads a file asynchronously with caching
func (m *AsyncIOManager) ReadFileAsync(ctx context.Context, path string) ([]byte, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		m.metrics.ReadLatency.Add(float64(latency.Microseconds()))
	}()
	
	// Check cache first
	if cached, ok := m.readCache.Get(path); ok {
		m.metrics.CacheHits.Add(1)
		return cached.([]byte), nil
	}
	
	m.metrics.CacheMisses.Add(1)
	
	// Perform async read with buffering
	data, err := m.readWithBuffer(ctx, path)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	m.readCache.Put(path, data)
	
	m.metrics.TotalReads.Add(1)
	m.metrics.BytesRead.Add(int64(len(data)))
	
	return data, nil
}

// readWithBuffer reads a file with optimized buffering
func (m *AsyncIOManager) readWithBuffer(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Get file size for pre-allocation
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	// Pre-allocate buffer
	size := stat.Size()
	if size > int64(m.readAheadSize) {
		size = int64(m.readAheadSize)
	}
	
	buffer := make([]byte, 0, size)
	reader := bufio.NewReaderSize(file, 32*1024)
	
	// Read with context cancellation support
	done := make(chan struct{})
	var readErr error
	
	go func() {
		defer close(done)
		buffer, readErr = io.ReadAll(reader)
	}()
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return buffer, readErr
	}
}

// WriteFileAsync writes a file asynchronously with batching
func (m *AsyncIOManager) WriteFileAsync(path string, data []byte, mode os.FileMode, callback func(error)) {
	req := &WriteRequest{
		Path:      path,
		Data:      data,
		Mode:      mode,
		Callback:  callback,
		Timestamp: time.Now(),
	}
	
	// Try to add to batch
	if !m.writeBuffer.Add(req) {
		// Buffer full, send directly to worker
		select {
		case m.writeChan <- req:
		case <-m.ctx.Done():
			if callback != nil {
				callback(fmt.Errorf("manager shutting down"))
			}
		}
	}
}

// Add adds a write request to the batch buffer
func (b *BatchBuffer) Add(req *WriteRequest) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Check if buffer is full
	if len(b.buffer) >= b.maxSize {
		b.flushLocked()
	}
	
	b.buffer = append(b.buffer, req)
	
	// Start flush timer if this is the first item
	if len(b.buffer) == 1 {
		b.flushTimer = time.AfterFunc(b.maxAge, func() {
			b.mu.Lock()
			b.flushLocked()
			b.mu.Unlock()
		})
	}
	
	// Flush if buffer is now full
	if len(b.buffer) >= b.maxSize {
		b.flushLocked()
	}
	
	return true
}

// flushLocked flushes the buffer (must be called with lock held)
func (b *BatchBuffer) flushLocked() {
	if len(b.buffer) == 0 {
		return
	}
	
	// Stop timer if running
	if b.flushTimer != nil {
		b.flushTimer.Stop()
		b.flushTimer = nil
	}
	
	// Get batch and reset buffer
	batch := b.buffer
	b.buffer = make([]*WriteRequest, 0, b.maxSize)
	
	// Process batch
	if b.flushCallback != nil {
		go b.flushCallback(batch)
	}
}

// processBatch processes a batch of write requests
func (m *AsyncIOManager) processBatch(batch []*WriteRequest) {
	m.metrics.BatchedWrites.Add(int64(len(batch)))
	
	// Group writes by directory for better file system performance
	byDir := make(map[string][]*WriteRequest)
	for _, req := range batch {
		dir := getDir(req.Path)
		byDir[dir] = append(byDir[dir], req)
	}
	
	// Process each directory group
	for _, dirBatch := range byDir {
		for _, req := range dirBatch {
			select {
			case m.writeChan <- req:
			case <-m.ctx.Done():
				if req.Callback != nil {
					req.Callback(fmt.Errorf("manager shutting down"))
				}
			}
		}
	}
}

// writeWorker processes write requests
func (m *AsyncIOManager) writeWorker(id int) {
	defer m.wg.Done()
	
	for {
		select {
		case req := <-m.writeChan:
			m.processWrite(req)
			
		case <-m.ctx.Done():
			// Drain remaining writes
			for {
				select {
				case req := <-m.writeChan:
					m.processWrite(req)
				default:
					return
				}
			}
		}
	}
}

// processWrite performs the actual write operation
func (m *AsyncIOManager) processWrite(req *WriteRequest) {
	startTime := time.Now()
	
	err := m.writeWithBuffer(req.Path, req.Data, req.Mode)
	
	latency := time.Since(startTime)
	m.metrics.WriteLatency.Add(float64(latency.Microseconds()))
	
	if err != nil {
		m.metrics.FailedWrites.Add(1)
		log.Printf("Async write failed for %s: %v", req.Path, err)
	} else {
		m.metrics.TotalWrites.Add(1)
		m.metrics.BytesWritten.Add(int64(len(req.Data)))
	}
	
	if req.Callback != nil {
		req.Callback(err)
	}
}

// writeWithBuffer writes data with optimized buffering
func (m *AsyncIOManager) writeWithBuffer(path string, data []byte, mode os.FileMode) error {
	// Create temp file for atomic write
	tmpPath := path + ".tmp"
	
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	
	// Use buffered writer
	writer := bufio.NewWriterSize(file, 32*1024)
	
	_, err = writer.Write(data)
	if err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	
	// Flush buffer
	if err := writer.Flush(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	
	// Sync to disk
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	
	file.Close()
	
	// Atomic rename
	return os.Rename(tmpPath, path)
}

// batchProcessor periodically processes batched writes
func (m *AsyncIOManager) batchProcessor() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.writeBuffer.mu.Lock()
			if len(m.writeBuffer.buffer) > 0 {
				m.writeBuffer.flushLocked()
			}
			m.writeBuffer.mu.Unlock()
			
		case <-m.ctx.Done():
			// Final flush
			m.writeBuffer.mu.Lock()
			m.writeBuffer.flushLocked()
			m.writeBuffer.mu.Unlock()
			return
		}
	}
}

// GetMetrics returns current I/O metrics
func (m *AsyncIOManager) GetMetrics() *IOMetrics {
	metrics := &IOMetrics{
		TotalReads:    atomic.Int64{},
		TotalWrites:   atomic.Int64{},
		CacheHits:     atomic.Int64{},
		CacheMisses:   atomic.Int64{},
		BytesRead:     atomic.Int64{},
		BytesWritten:  atomic.Int64{},
		BatchedWrites: atomic.Int64{},
		FailedWrites:  atomic.Int64{},
	}
	
	metrics.TotalReads.Store(m.metrics.TotalReads.Load())
	metrics.TotalWrites.Store(m.metrics.TotalWrites.Load())
	metrics.CacheHits.Store(m.metrics.CacheHits.Load())
	metrics.CacheMisses.Store(m.metrics.CacheMisses.Load())
	metrics.BytesRead.Store(m.metrics.BytesRead.Load())
	metrics.BytesWritten.Store(m.metrics.BytesWritten.Load())
	metrics.BatchedWrites.Store(m.metrics.BatchedWrites.Load())
	metrics.FailedWrites.Store(m.metrics.FailedWrites.Load())
	
	m.metrics.mu.RLock()
	metrics.ReadLatency = m.metrics.ReadLatency
	metrics.WriteLatency = m.metrics.WriteLatency
	m.metrics.mu.RUnlock()
	
	return metrics
}

// Shutdown gracefully shuts down the I/O manager
func (m *AsyncIOManager) Shutdown(timeout time.Duration) error {
	// Cancel context
	m.cancel()
	
	// Final flush
	m.writeBuffer.mu.Lock()
	m.writeBuffer.flushLocked()
	m.writeBuffer.mu.Unlock()
	
	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

// getDir extracts directory from path
func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
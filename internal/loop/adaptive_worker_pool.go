
package loop



import (

	"context"

	"fmt"

	"log"

	"runtime"

	"sync"

	"sync/atomic"

	"time"

)



// AdaptiveWorkerPool implements a dynamic worker pool with backpressure control.

type AdaptiveWorkerPool struct {

	// Core configuration.

	minWorkers         int

	maxWorkers         int

	targetQueueDepth   int

	scaleUpThreshold   float64

	scaleDownThreshold float64



	// Work queue with bounded capacity for backpressure.

	workQueue     chan WorkItem

	priorityQueue chan WorkItem // High priority items



	// Worker management.

	workers         sync.WaitGroup

	activeWorkers   atomic.Int64

	processingItems atomic.Int64



	// Adaptive scaling.

	scalingMutex   sync.RWMutex

	currentWorkers int

	lastScaleTime  time.Time

	scaleCooldown  time.Duration



	// Metrics for scaling decisions.

	queueDepthHistory []int

	processingTimes   *RingBuffer



	// Lifecycle management.

	ctx         context.Context

	cancel      context.CancelFunc

	stopSignal  chan struct{}

	queueClosed sync.Once



	// Backpressure control.

	backpressure  atomic.Bool

	rejectedCount atomic.Int64



	// Performance metrics.

	metrics *PoolMetrics

}



// PoolMetrics tracks worker pool performance.

type PoolMetrics struct {

	mu                 sync.RWMutex

	TotalProcessed     int64

	TotalRejected      int64

	AverageWaitTime    time.Duration

	AverageProcessTime time.Duration

	CurrentWorkers     int

	QueueDepth         int

	Throughput         float64 // items/second

	lastReset          time.Time

}



// AdaptivePoolConfig configures the adaptive worker pool.

type AdaptivePoolConfig struct {

	MinWorkers         int           // Minimum number of workers

	MaxWorkers         int           // Maximum number of workers

	QueueSize          int           // Maximum queue size

	ScaleUpThreshold   float64       // Queue utilization threshold to scale up (0.7 = 70%)

	ScaleDownThreshold float64       // Queue utilization threshold to scale down (0.3 = 30%)

	ScaleCooldown      time.Duration // Minimum time between scaling operations

	MetricsWindow      int           // Size of metrics window for decisions

}



// DefaultAdaptiveConfig returns optimized default configuration.

func DefaultAdaptiveConfig() *AdaptivePoolConfig {

	cpuCount := runtime.NumCPU()



	return &AdaptivePoolConfig{

		MinWorkers:         max(2, cpuCount/2),

		MaxWorkers:         min(cpuCount*2, 32), // Cap at 32 to prevent resource exhaustion

		QueueSize:          1000,                // Bounded queue for backpressure

		ScaleUpThreshold:   0.7,

		ScaleDownThreshold: 0.3,

		ScaleCooldown:      5 * time.Second,

		MetricsWindow:      100,

	}

}



// NewAdaptiveWorkerPool creates a new adaptive worker pool.

func NewAdaptiveWorkerPool(config *AdaptivePoolConfig) *AdaptiveWorkerPool {

	if config == nil {

		config = DefaultAdaptiveConfig()

	}



	// Validate configuration.

	if config.MinWorkers < 1 {

		config.MinWorkers = 1

	}

	if config.MaxWorkers < config.MinWorkers {

		config.MaxWorkers = config.MinWorkers

	}

	if config.QueueSize < 100 {

		config.QueueSize = 100

	}



	ctx, cancel := context.WithCancel(context.Background())



	pool := &AdaptiveWorkerPool{

		minWorkers:         config.MinWorkers,

		maxWorkers:         config.MaxWorkers,

		targetQueueDepth:   config.QueueSize,

		scaleUpThreshold:   config.ScaleUpThreshold,

		scaleDownThreshold: config.ScaleDownThreshold,

		workQueue:          make(chan WorkItem, config.QueueSize),

		priorityQueue:      make(chan WorkItem, config.QueueSize/10), // 10% for priority

		currentWorkers:     config.MinWorkers,

		scaleCooldown:      config.ScaleCooldown,

		lastScaleTime:      time.Now(),

		queueDepthHistory:  make([]int, 0, config.MetricsWindow),

		processingTimes:    NewRingBuffer(config.MetricsWindow),

		ctx:                ctx,

		cancel:             cancel,

		stopSignal:         make(chan struct{}),

		metrics: &PoolMetrics{

			lastReset: time.Now(),

		},

	}



	// Start initial workers.

	for i := range config.MinWorkers {

		pool.startWorker(i)

	}



	// Start adaptive scaling monitor.

	go pool.scalingMonitor()



	// Start metrics collector.

	go pool.metricsCollector()



	return pool

}



// Submit adds a work item to the pool with backpressure control.

func (p *AdaptiveWorkerPool) Submit(item WorkItem) error {

	// Check if pool is shutting down.

	select {

	case <-p.ctx.Done():

		return fmt.Errorf("pool is shutting down")

	default:

	}



	// Try priority queue first if item has high priority.

	if p.isHighPriority(item) {

		select {

		case p.priorityQueue <- item:

			return nil

		default:

			// Priority queue full, try regular queue.

		}

	}



	// Try regular queue with timeout for backpressure.

	timeout := time.NewTimer(100 * time.Millisecond)

	defer timeout.Stop()



	select {

	case p.workQueue <- item:

		return nil

	case <-timeout.C:

		// Queue is full, apply backpressure.

		p.backpressure.Store(true)

		p.rejectedCount.Add(1)

		return fmt.Errorf("queue full, backpressure applied")

	case <-p.ctx.Done():

		return fmt.Errorf("pool is shutting down")

	}

}



// isHighPriority determines if an item should be prioritized.

func (p *AdaptiveWorkerPool) isHighPriority(item WorkItem) bool {

	// Priority based on retry attempts (failed items get priority).

	return item.Attempt > 0

}



// startWorker starts a new worker goroutine.

func (p *AdaptiveWorkerPool) startWorker(id int) {

	p.workers.Add(1)

	p.activeWorkers.Add(1)



	go func() {

		defer p.workers.Done()

		defer p.activeWorkers.Add(-1)



		log.Printf("Worker %d started", id)



		for {

			select {

			// Priority queue has precedence.

			case item := <-p.priorityQueue:

				p.processItem(id, item)



			case item := <-p.workQueue:

				p.processItem(id, item)



			case <-p.stopSignal:

				log.Printf("Worker %d stopping", id)

				return



			case <-p.ctx.Done():

				log.Printf("Worker %d context cancelled", id)

				return

			}



			// Clear backpressure if queue is below threshold.

			if len(p.workQueue) < p.targetQueueDepth/2 {

				p.backpressure.Store(false)

			}

		}

	}()

}



// processItem processes a single work item.

func (p *AdaptiveWorkerPool) processItem(_ int, item WorkItem) {

	startTime := time.Now()

	p.processingItems.Add(1)

	defer p.processingItems.Add(-1)



	// Process the item (placeholder - integrate with actual processor).

	// This would call the actual file processing logic.

	processingTime := time.Since(startTime)



	// Record metrics.

	p.processingTimes.Add(float64(processingTime.Milliseconds()))



	p.metrics.mu.Lock()

	p.metrics.TotalProcessed++

	p.metrics.AverageProcessTime = time.Duration(p.processingTimes.Average()) * time.Millisecond

	p.metrics.mu.Unlock()

}



// scalingMonitor monitors queue depth and scales workers accordingly.

func (p *AdaptiveWorkerPool) scalingMonitor() {

	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			p.evaluateScaling()



		case <-p.ctx.Done():

			return

		}

	}

}



// evaluateScaling determines if scaling is needed and performs it.

func (p *AdaptiveWorkerPool) evaluateScaling() {

	// Check cooldown period.

	if time.Since(p.lastScaleTime) < p.scaleCooldown {

		return

	}



	queueDepth := len(p.workQueue) + len(p.priorityQueue)



	// Record queue depth history.

	p.scalingMutex.Lock()

	p.queueDepthHistory = append(p.queueDepthHistory, queueDepth)

	if len(p.queueDepthHistory) > 10 {

		p.queueDepthHistory = p.queueDepthHistory[1:]

	}

	currentWorkers := p.currentWorkers

	p.scalingMutex.Unlock()



	// Calculate average queue utilization.

	avgUtilization := p.calculateAverageUtilization()



	// Scale up if needed.

	if avgUtilization > p.scaleUpThreshold && currentWorkers < p.maxWorkers {

		newWorkers := min(currentWorkers+2, p.maxWorkers)

		p.scaleWorkers(newWorkers - currentWorkers)

		log.Printf("Scaling up: %d -> %d workers (utilization: %.2f%%)",

			currentWorkers, newWorkers, avgUtilization*100)

	}



	// Scale down if needed.

	if avgUtilization < p.scaleDownThreshold && currentWorkers > p.minWorkers {

		newWorkers := max(currentWorkers-1, p.minWorkers)

		p.scaleWorkers(currentWorkers - newWorkers)

		log.Printf("Scaling down: %d -> %d workers (utilization: %.2f%%)",

			currentWorkers, newWorkers, avgUtilization*100)

	}

}



// calculateAverageUtilization calculates average queue utilization.

func (p *AdaptiveWorkerPool) calculateAverageUtilization() float64 {

	p.scalingMutex.RLock()

	defer p.scalingMutex.RUnlock()



	if len(p.queueDepthHistory) == 0 {

		return 0

	}



	sum := 0

	for _, depth := range p.queueDepthHistory {

		sum += depth

	}



	avgDepth := float64(sum) / float64(len(p.queueDepthHistory))

	return avgDepth / float64(p.targetQueueDepth)

}



// scaleWorkers adds or removes workers.

func (p *AdaptiveWorkerPool) scaleWorkers(delta int) {

	p.scalingMutex.Lock()

	defer p.scalingMutex.Unlock()



	if delta > 0 {

		// Add workers.

		for i := 0; i < delta; i++ {

			p.startWorker(p.currentWorkers + i)

		}

		p.currentWorkers += delta

	} else if delta < 0 {

		// Remove workers (signal them to stop).

		for i := 0; i < -delta; i++ {

			select {

			case p.stopSignal <- struct{}{}:

			default:

			}

		}

		p.currentWorkers += delta

	}



	p.lastScaleTime = time.Now()



	// Update metrics.

	p.metrics.mu.Lock()

	p.metrics.CurrentWorkers = p.currentWorkers

	p.metrics.mu.Unlock()

}



// metricsCollector collects performance metrics.

func (p *AdaptiveWorkerPool) metricsCollector() {

	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()



	lastProcessed := int64(0)



	for {

		select {

		case <-ticker.C:

			p.metrics.mu.Lock()



			// Calculate throughput.

			processed := p.metrics.TotalProcessed

			delta := processed - lastProcessed

			p.metrics.Throughput = float64(delta)

			lastProcessed = processed



			// Update queue depth.

			p.metrics.QueueDepth = len(p.workQueue) + len(p.priorityQueue)



			p.metrics.mu.Unlock()



		case <-p.ctx.Done():

			return

		}

	}

}



// GetMetrics returns current pool metrics.

func (p *AdaptiveWorkerPool) GetMetrics() *PoolMetrics {

	p.metrics.mu.RLock()

	defer p.metrics.mu.RUnlock()



	// Create a copy without the mutex to avoid copylocks.

	metricsCopy := &PoolMetrics{

		TotalProcessed:     p.metrics.TotalProcessed,

		TotalRejected:      p.metrics.TotalRejected,

		AverageWaitTime:    p.metrics.AverageWaitTime,

		AverageProcessTime: p.metrics.AverageProcessTime,

		CurrentWorkers:     p.metrics.CurrentWorkers,

		QueueDepth:         p.metrics.QueueDepth,

		Throughput:         p.metrics.Throughput,

		lastReset:          p.metrics.lastReset,

	}

	return metricsCopy

}



// Shutdown gracefully shuts down the pool.

func (p *AdaptiveWorkerPool) Shutdown(timeout time.Duration) error {

	// Cancel context to stop accepting new work.

	p.cancel()



	// Close queues.

	p.queueClosed.Do(func() {

		close(p.workQueue)

		close(p.priorityQueue)

	})



	// Wait for workers with timeout.

	done := make(chan struct{})

	go func() {

		p.workers.Wait()

		close(done)

	}()



	select {

	case <-done:

		return nil

	case <-time.After(timeout):

		return fmt.Errorf("shutdown timeout after %v", timeout)

	}

}



// IsBackpressured returns true if backpressure is applied.

func (p *AdaptiveWorkerPool) IsBackpressured() bool {

	return p.backpressure.Load()

}



// QueueDepth returns current queue depth.

func (p *AdaptiveWorkerPool) QueueDepth() int {

	return len(p.workQueue) + len(p.priorityQueue)

}



// ActiveWorkers returns the number of active workers.

func (p *AdaptiveWorkerPool) ActiveWorkers() int64 {

	return p.activeWorkers.Load()

}



// max returns the maximum of two integers.

func max(a, b int) int {

	if a > b {

		return a

	}

	return b

}



// min returns the minimum of two integers.

func min(a, b int) int {

	if a < b {

		return a

	}

	return b

}


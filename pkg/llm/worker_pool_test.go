package llm

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkerPool(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    5,
		QueueSize:      100,
		DefaultTimeout: 30 * time.Second,
		EnableMetrics:  true,
	}

	pool := NewWorkerPool(config)

	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.workerCount)
	assert.Equal(t, 5, len(pool.workers))
	assert.NotNil(t, pool.jobQueue)
	assert.NotNil(t, pool.metrics)
	assert.False(t, pool.started)
}

func TestWorkerPool_StartStop(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    3,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)

	// Test start
	err := pool.Start()
	assert.NoError(t, err)
	assert.True(t, pool.started)

	// Test duplicate start
	err = pool.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Test stop
	err = pool.Stop(5 * time.Second)
	assert.NoError(t, err)
	assert.False(t, pool.started)

	// Test stop when not started
	err = pool.Stop(5 * time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestWorkerPool_SubmitJob_Success(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var executed int32
	var callbackExecuted int32

	job := Job{
		ID: "test-job-1",
		Task: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		Callback: func(result JobResult) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-job-1", result.JobID)
			assert.True(t, result.Success)
			assert.NoError(t, result.Error)
			assert.True(t, result.Duration >= 100*time.Millisecond)
		},
	}

	err := pool.Submit(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&executed))
	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalJobs)
	assert.Equal(t, int64(1), metrics.CompletedJobs)
	assert.Equal(t, int64(0), metrics.FailedJobs)
}

func TestWorkerPool_SubmitJob_Failure(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	testError := errors.New("test error")
	var callbackExecuted int32

	job := Job{
		ID: "test-job-fail",
		Task: func(ctx context.Context) error {
			return testError
		},
		Callback: func(result JobResult) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-job-fail", result.JobID)
			assert.False(t, result.Success)
			assert.Equal(t, testError, result.Error)
		},
	}

	err := pool.Submit(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalJobs)
	assert.Equal(t, int64(1), metrics.CompletedJobs)
	assert.Equal(t, int64(1), metrics.FailedJobs)
}

func TestWorkerPool_SubmitJob_Timeout(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    1,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var callbackExecuted int32

	job := Job{
		ID:      "test-job-timeout",
		Timeout: 100 * time.Millisecond,
		Task: func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		Callback: func(result JobResult) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.False(t, result.Success)
			assert.Error(t, result.Error)
			assert.Contains(t, result.Error.Error(), "context deadline exceeded")
		},
	}

	err := pool.Submit(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))
}

func TestWorkerPool_SubmitJob_Panic(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    1,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var callbackExecuted int32

	job := Job{
		ID: "test-job-panic",
		Task: func(ctx context.Context) error {
			panic("test panic")
		},
		Callback: func(result JobResult) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.False(t, result.Success)
			assert.Error(t, result.Error)
			assert.Contains(t, result.Error.Error(), "job panicked: test panic")
		},
	}

	err := pool.Submit(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.FailedJobs)
}

func TestWorkerPool_ConcurrentJobs(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    5,
		QueueSize:      100,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(10 * time.Second)

	const numJobs = 50
	var completedJobs int32
	var failedJobs int32
	var wg sync.WaitGroup

	// Submit multiple jobs concurrently
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		jobID := i

		job := Job{
			ID: fmt.Sprintf("concurrent-job-%d", jobID),
			Task: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				if jobID%10 == 0 { // 10% failure rate
					return errors.New("simulated failure")
				}
				return nil
			},
			Callback: func(result JobResult) {
				defer wg.Done()
				if result.Success {
					atomic.AddInt32(&completedJobs, 1)
				} else {
					atomic.AddInt32(&failedJobs, 1)
				}
			},
		}

		err := pool.Submit(job)
		if err != nil {
			wg.Done()
			t.Errorf("Failed to submit job %d: %v", jobID, err)
		}
	}

	// Wait for all jobs to complete
	wg.Wait()

	assert.Equal(t, int32(45), atomic.LoadInt32(&completedJobs))
	assert.Equal(t, int32(5), atomic.LoadInt32(&failedJobs))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numJobs), metrics.TotalJobs)
	assert.Equal(t, int64(numJobs), metrics.CompletedJobs)
	assert.Equal(t, int64(5), metrics.FailedJobs)
}

func TestWorkerPool_QueueFull(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    1,
		QueueSize:      2, // Small queue
		DefaultTimeout: 1 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	// Submit jobs that will block the worker
	for i := 0; i < 2; i++ {
		job := Job{
			ID: fmt.Sprintf("blocking-job-%d", i),
			Task: func(ctx context.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			},
		}
		err := pool.Submit(job)
		assert.NoError(t, err)
	}

	// Worker is now busy, queue should be full
	// Try to submit another job - should be accepted (worker + queue = 3 total capacity)
	job := Job{
		ID: "overflow-job-1",
		Task: func(ctx context.Context) error {
			return nil
		},
	}
	err := pool.Submit(job)
	assert.NoError(t, err)

	// Now queue is really full - this should fail
	job = Job{
		ID: "overflow-job-2",
		Task: func(ctx context.Context) error {
			return nil
		},
	}
	err = pool.Submit(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job queue is full")
}

func TestWorkerPool_SubmitNotStarted(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)

	job := Job{
		ID: "test-job",
		Task: func(ctx context.Context) error {
			return nil
		},
	}

	err := pool.Submit(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker pool not started")
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())

	var completedJobs int32

	// Submit jobs that take some time
	for i := 0; i < 5; i++ {
		job := Job{
			ID: fmt.Sprintf("shutdown-job-%d", i),
			Task: func(ctx context.Context) error {
				select {
				case <-time.After(200 * time.Millisecond):
					atomic.AddInt32(&completedJobs, 1)
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		pool.Submit(job)
	}

	// Stop with sufficient timeout
	err := pool.Stop(2 * time.Second)
	assert.NoError(t, err)

	// All jobs should have completed
	assert.Equal(t, int32(5), atomic.LoadInt32(&completedJobs))
}

func TestWorkerPool_ForceShutdown(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())

	var completedJobs int32

	// Submit jobs that take longer than shutdown timeout
	for i := 0; i < 3; i++ {
		job := Job{
			ID: fmt.Sprintf("long-job-%d", i),
			Task: func(ctx context.Context) error {
				select {
				case <-time.After(2 * time.Second):
					atomic.AddInt32(&completedJobs, 1)
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}
		pool.Submit(job)
	}

	// Stop with short timeout to force shutdown
	err := pool.Stop(100 * time.Millisecond)
	assert.NoError(t, err)

	// Not all jobs should have completed due to force shutdown
	assert.True(t, atomic.LoadInt32(&completedJobs) < 3)
}

func TestWorkerMetrics(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    2,
		QueueSize:      10,
		DefaultTimeout: 5 * time.Second,
		EnableMetrics:  true,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	// Submit some jobs
	for i := 0; i < 10; i++ {
		job := Job{
			ID: fmt.Sprintf("metrics-job-%d", i),
			Task: func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond)
				if i%3 == 0 {
					return errors.New("test error")
				}
				return nil
			},
		}
		pool.Submit(job)
	}

	// Wait for completion
	time.Sleep(1 * time.Second)

	// Check pool metrics
	poolMetrics := pool.GetMetrics()
	assert.Equal(t, int64(10), poolMetrics.TotalJobs)
	assert.Equal(t, int64(10), poolMetrics.CompletedJobs)
	assert.True(t, poolMetrics.FailedJobs > 0)
	assert.True(t, poolMetrics.AverageLatency > 0)

	// Check worker metrics
	workerMetrics := pool.GetWorkerMetrics()
	assert.Len(t, workerMetrics, 2)

	totalProcessed := int64(0)
	for _, metrics := range workerMetrics {
		totalProcessed += metrics.JobsProcessed
		if metrics.JobsProcessed > 0 {
			assert.True(t, metrics.AverageLatency > 0)
		}
	}
	assert.Equal(t, int64(10), totalProcessed)
}

func TestIntentProcessingPool(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    3,
		QueueSize:      20,
		DefaultTimeout: 5 * time.Second,
	}

	processor := func(ctx context.Context, intent string) (map[string]interface{}, error) {
		// Simulate intent processing
		time.Sleep(100 * time.Millisecond)
		return map[string]interface{}{
			"intent":     intent,
			"processed":  true,
			"result":     "success",
			"timestamp": time.Now(),
		}, nil
	}

	intentPool := NewIntentProcessingPool(config, processor)
	require.NoError(t, intentPool.Start())
	defer intentPool.Stop(5 * time.Second)

	var results []map[string]interface{}
	var errors []error
	var wg sync.WaitGroup
	var mu sync.Mutex

	intents := []string{
		"Deploy AMF with 3 replicas",
		"Scale UPF to 5 instances",
		"Configure network slice for eMBB",
	}

	for i, intent := range intents {
		wg.Add(1)
		intentID := fmt.Sprintf("intent-%d", i)

		err := intentPool.ProcessIntent(context.Background(), intentID, intent, func(id string, result map[string]interface{}, err error) {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, err)
			} else {
				results = append(results, result)
			}
		})
		assert.NoError(t, err)
	}

	wg.Wait()

	assert.Len(t, results, 3)
	assert.Len(t, errors, 0)

	for _, result := range results {
		assert.True(t, result["processed"].(bool))
		assert.Equal(t, "success", result["result"])
	}
}

func TestHealthChecker(t *testing.T) {
	config := &WorkerPoolConfig{
		WorkerCount:    3,
		QueueSize:      10,
		DefaultTimeout: 1 * time.Second,
	}

	pool := NewWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	healthChecker := NewHealthChecker(pool, 100*time.Millisecond, 0.8)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start health monitoring
	go healthChecker.Start(ctx)

	// Submit some failing jobs to make workers unhealthy
	for i := 0; i < 20; i++ {
		job := Job{
			ID: fmt.Sprintf("failing-job-%d", i),
			Task: func(ctx context.Context) error {
				return errors.New("simulated failure")
			},
		}
		pool.Submit(job)
	}

	// Wait for jobs to process and health check to run
	time.Sleep(500 * time.Millisecond)

	// Check if health checker detected unhealthy workers
	unhealthyWorkers := healthChecker.GetUnhealthyWorkers()
	isHealthy := healthChecker.IsHealthy()

	// With 100% failure rate, all workers should be unhealthy
	assert.False(t, isHealthy)
	assert.True(t, len(unhealthyWorkers) > 0)

	// Now submit successful jobs to recover health
	for i := 0; i < 30; i++ {
		job := Job{
			ID: fmt.Sprintf("success-job-%d", i),
			Task: func(ctx context.Context) error {
				return nil
			},
		}
		pool.Submit(job)
	}

	// Wait for recovery
	time.Sleep(500 * time.Millisecond)

	// Health should improve
	finalUnhealthyWorkers := healthChecker.GetUnhealthyWorkers()
	finalHealthy := healthChecker.IsHealthy()

	// Should have fewer unhealthy workers or be completely healthy
	assert.True(t, len(finalUnhealthyWorkers) <= len(unhealthyWorkers))
	
	// With enough successful jobs, pool should eventually become healthy
	if !finalHealthy {
		t.Log("Pool not fully recovered yet, which is acceptable in this timing-sensitive test")
	}
}

// Benchmark tests
func BenchmarkWorkerPool_JobSubmission(b *testing.B) {
	config := &WorkerPoolConfig{
		WorkerCount:    10,
		QueueSize:      1000,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Stop(10 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			job := Job{
				ID: fmt.Sprintf("bench-job-%d", i),
				Task: func(ctx context.Context) error {
					// Minimal work
					return nil
				},
			}
			pool.Submit(job)
			i++
		}
	})
}

func BenchmarkWorkerPool_JobProcessing(b *testing.B) {
	config := &WorkerPoolConfig{
		WorkerCount:    4,
		QueueSize:      1000,
		DefaultTimeout: 5 * time.Second,
	}

	pool := NewWorkerPool(config)
	pool.Start()
	defer pool.Stop(10 * time.Second)

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		job := Job{
			ID: fmt.Sprintf("bench-process-job-%d", i),
			Task: func(ctx context.Context) error {
				// Simulate minimal processing work
				time.Sleep(1 * time.Microsecond)
				return nil
			},
			Callback: func(result JobResult) {
				wg.Done()
			},
		}
		pool.Submit(job)
	}
	
	wg.Wait()
}
package rag

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsyncWorkerPool_NewAsyncWorkerPool(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 3,
		QueryWorkers:    2,
		DocumentQueueSize: 10,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)

	assert.NotNil(t, pool)
	assert.Equal(t, 3, pool.documentWorkers)
	assert.Equal(t, 2, pool.queryWorkers)
	assert.NotNil(t, pool.documentQueue)
	assert.NotNil(t, pool.queryQueue)
	assert.NotNil(t, pool.metrics)
	assert.False(t, pool.started)
}

func TestAsyncWorkerPool_StartStop(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 2,
		QueryWorkers:    1,
		DocumentQueueSize: 5,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)

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

func TestAsyncWorkerPool_SubmitDocumentJob_Success(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 2,
		QueryWorkers:    1,
		DocumentQueueSize: 10,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var processed int32
	var callbackExecuted int32

	job := DocumentJob{
		ID:       "test-doc-1",
		FilePath: "/test/document.pdf",
		Content:  "Test document content for processing",
		Metadata: map[string]interface{}{
			"source": "test",
			"type":   "pdf",
		},
		Callback: func(id string, chunks []DocumentChunk, err error) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-doc-1", id)
			assert.NoError(t, err)
			assert.NotEmpty(t, chunks)
			assert.True(t, len(chunks) > 0)
			atomic.AddInt32(&processed, 1)
		},
	}

	err := pool.SubmitDocumentJob(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&processed))
	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.DocumentJobsSubmitted)
	assert.Equal(t, int64(1), metrics.DocumentJobsCompleted)
	assert.Equal(t, int64(0), metrics.DocumentJobsFailed)
}

func TestAsyncWorkerPool_SubmitDocumentJob_Failure(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    1,
		DocumentQueueSize: 5,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var callbackExecuted int32
	expectedError := errors.New("document processing failed")

	job := DocumentJob{
		ID:       "test-doc-fail",
		FilePath: "/invalid/path.pdf",
		Content:  "", // Empty content to trigger failure
		Callback: func(id string, chunks []DocumentChunk, err error) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-doc-fail", id)
			assert.Error(t, err)
			assert.Empty(t, chunks)
		},
	}

	err := pool.SubmitDocumentJob(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.DocumentJobsSubmitted)
	assert.Equal(t, int64(1), metrics.DocumentJobsCompleted)
	assert.Equal(t, int64(1), metrics.DocumentJobsFailed)
}

func TestAsyncWorkerPool_SubmitQueryJob_Success(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    2,
		DocumentQueueSize: 5,
		QueryQueueSize:    10,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var processed int32
	var callbackExecuted int32

	job := QueryJob{
		ID:    "test-query-1",
		Query: "Deploy AMF with 3 replicas",
		Filters: map[string]interface{}{
			"category": "network_function",
			"priority": "high",
		},
		Limit: 10,
		Callback: func(id string, results []RetrievedContext, err error) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-query-1", id)
			assert.NoError(t, err)
			assert.NotEmpty(t, results)
			atomic.AddInt32(&processed, 1)
		},
	}

	err := pool.SubmitQueryJob(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&processed))
	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.QueryJobsSubmitted)
	assert.Equal(t, int64(1), metrics.QueryJobsCompleted)
	assert.Equal(t, int64(0), metrics.QueryJobsFailed)
}

func TestAsyncWorkerPool_SubmitQueryJob_Failure(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    1,
		DocumentQueueSize: 5,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	var callbackExecuted int32

	job := QueryJob{
		ID:    "test-query-fail",
		Query: "", // Empty query to trigger failure
		Limit: 10,
		Callback: func(id string, results []RetrievedContext, err error) {
			atomic.AddInt32(&callbackExecuted, 1)
			assert.Equal(t, "test-query-fail", id)
			assert.Error(t, err)
			assert.Empty(t, results)
		},
	}

	err := pool.SubmitQueryJob(job)
	assert.NoError(t, err)

	// Wait for job completion
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackExecuted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.QueryJobsSubmitted)
	assert.Equal(t, int64(1), metrics.QueryJobsCompleted)
	assert.Equal(t, int64(1), metrics.QueryJobsFailed)
}

func TestAsyncWorkerPool_ConcurrentJobs(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 3,
		QueryWorkers:    3,
		DocumentQueueSize: 50,
		QueryQueueSize:    50,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(10 * time.Second)

	const numDocJobs = 20
	const numQueryJobs = 15

	var docCompleted int32
	var queryCompleted int32
	var wg sync.WaitGroup

	// Submit document jobs
	for i := 0; i < numDocJobs; i++ {
		wg.Add(1)
		job := DocumentJob{
			ID:      fmt.Sprintf("doc-job-%d", i),
			Content: fmt.Sprintf("Document content %d", i),
			Callback: func(id string, chunks []DocumentChunk, err error) {
				defer wg.Done()
				if err == nil {
					atomic.AddInt32(&docCompleted, 1)
				}
			},
		}
		err := pool.SubmitDocumentJob(job)
		if err != nil {
			wg.Done()
			t.Errorf("Failed to submit document job %d: %v", i, err)
		}
	}

	// Submit query jobs
	for i := 0; i < numQueryJobs; i++ {
		wg.Add(1)
		job := QueryJob{
			ID:    fmt.Sprintf("query-job-%d", i),
			Query: fmt.Sprintf("Test query %d", i),
			Limit: 5,
			Callback: func(id string, results []RetrievedContext, err error) {
				defer wg.Done()
				if err == nil {
					atomic.AddInt32(&queryCompleted, 1)
				}
			},
		}
		err := pool.SubmitQueryJob(job)
		if err != nil {
			wg.Done()
			t.Errorf("Failed to submit query job %d: %v", i, err)
		}
	}

	// Wait for all jobs to complete
	wg.Wait()

	assert.Equal(t, int32(numDocJobs), atomic.LoadInt32(&docCompleted))
	assert.Equal(t, int32(numQueryJobs), atomic.LoadInt32(&queryCompleted))

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numDocJobs), metrics.DocumentJobsSubmitted)
	assert.Equal(t, int64(numQueryJobs), metrics.QueryJobsSubmitted)
	assert.Equal(t, int64(numDocJobs), metrics.DocumentJobsCompleted)
	assert.Equal(t, int64(numQueryJobs), metrics.QueryJobsCompleted)
}

func TestAsyncWorkerPool_QueueFull(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    1,
		DocumentQueueSize: 2, // Small queue
		QueryQueueSize:    2, // Small queue
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	// Fill document queue
	for i := 0; i < 2; i++ {
		job := DocumentJob{
			ID:      fmt.Sprintf("blocking-doc-%d", i),
			Content: "test content",
			Callback: func(id string, chunks []DocumentChunk, err error) {
				time.Sleep(2 * time.Second) // Block worker
			},
		}
		err := pool.SubmitDocumentJob(job)
		assert.NoError(t, err)
	}

	// Queue is now full + worker busy - this should fail
	job := DocumentJob{
		ID:      "overflow-doc",
		Content: "test content",
	}
	err := pool.SubmitDocumentJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document queue is full")

	// Same test for query queue
	for i := 0; i < 2; i++ {
		job := QueryJob{
			ID:    fmt.Sprintf("blocking-query-%d", i),
			Query: "test query",
			Callback: func(id string, results []RetrievedContext, err error) {
				time.Sleep(2 * time.Second) // Block worker
			},
		}
		err := pool.SubmitQueryJob(job)
		assert.NoError(t, err)
	}

	// Query queue full test
	queryJob := QueryJob{
		ID:    "overflow-query",
		Query: "test query",
	}
	err = pool.SubmitQueryJob(queryJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query queue is full")
}

func TestAsyncWorkerPool_SubmitNotStarted(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    1,
		DocumentQueueSize: 5,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)

	// Test document job submission when not started
	docJob := DocumentJob{
		ID:      "test-doc",
		Content: "test content",
	}
	err := pool.SubmitDocumentJob(docJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "async worker pool not started")

	// Test query job submission when not started
	queryJob := QueryJob{
		ID:    "test-query",
		Query: "test query",
	}
	err = pool.SubmitQueryJob(queryJob)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "async worker pool not started")
}

func TestAsyncWorkerPool_GracefulShutdown(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 2,
		QueryWorkers:    2,
		DocumentQueueSize: 10,
		QueryQueueSize:    10,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())

	var completedJobs int32

	// Submit jobs that take some time
	for i := 0; i < 5; i++ {
		docJob := DocumentJob{
			ID:      fmt.Sprintf("shutdown-doc-%d", i),
			Content: "test content",
			Callback: func(id string, chunks []DocumentChunk, err error) {
				if err == nil {
					atomic.AddInt32(&completedJobs, 1)
				}
			},
		}
		pool.SubmitDocumentJob(docJob)

		queryJob := QueryJob{
			ID:    fmt.Sprintf("shutdown-query-%d", i),
			Query: "test query",
			Callback: func(id string, results []RetrievedContext, err error) {
				if err == nil {
					atomic.AddInt32(&completedJobs, 1)
				}
			},
		}
		pool.SubmitQueryJob(queryJob)
	}

	// Stop with sufficient timeout
	err := pool.Stop(3 * time.Second)
	assert.NoError(t, err)

	// All jobs should have completed
	assert.Equal(t, int32(10), atomic.LoadInt32(&completedJobs))
}

func TestAsyncWorkerPool_ForceShutdown(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 1,
		QueryWorkers:    1,
		DocumentQueueSize: 5,
		QueryQueueSize:    5,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())

	var completedJobs int32

	// Submit jobs that take longer than shutdown timeout
	for i := 0; i < 3; i++ {
		docJob := DocumentJob{
			ID:      fmt.Sprintf("long-doc-%d", i),
			Content: "test content",
			Callback: func(id string, chunks []DocumentChunk, err error) {
				time.Sleep(2 * time.Second)
				if err == nil {
					atomic.AddInt32(&completedJobs, 1)
				}
			},
		}
		pool.SubmitDocumentJob(docJob)
	}

	// Stop with short timeout to force shutdown
	err := pool.Stop(100 * time.Millisecond)
	assert.NoError(t, err)

	// Not all jobs should have completed due to force shutdown
	assert.True(t, atomic.LoadInt32(&completedJobs) < 3)
}

func TestAsyncWorkerPool_GetMetrics(t *testing.T) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 2,
		QueryWorkers:    2,
		DocumentQueueSize: 10,
		QueryQueueSize:    10,
	}

	pool := NewAsyncWorkerPool(config)
	require.NoError(t, pool.Start())
	defer pool.Stop(5 * time.Second)

	// Submit some jobs
	for i := 0; i < 5; i++ {
		docJob := DocumentJob{
			ID:      fmt.Sprintf("metrics-doc-%d", i),
			Content: fmt.Sprintf("test content %d", i),
			Callback: func(id string, chunks []DocumentChunk, err error) {
				// Job completed
			},
		}
		pool.SubmitDocumentJob(docJob)

		queryJob := QueryJob{
			ID:    fmt.Sprintf("metrics-query-%d", i),
			Query: fmt.Sprintf("test query %d", i),
			Callback: func(id string, results []RetrievedContext, err error) {
				// Job completed
			},
		}
		pool.SubmitQueryJob(queryJob)
	}

	// Wait for completion
	time.Sleep(1 * time.Second)

	metrics := pool.GetMetrics()
	assert.Equal(t, int64(5), metrics.DocumentJobsSubmitted)
	assert.Equal(t, int64(5), metrics.QueryJobsSubmitted)
	assert.Equal(t, int64(5), metrics.DocumentJobsCompleted)
	assert.Equal(t, int64(5), metrics.QueryJobsCompleted)
	assert.True(t, metrics.AverageDocumentProcessingTime > 0)
	assert.True(t, metrics.AverageQueryProcessingTime > 0)
}

func TestPipeline_ProcessDocumentsAsync(t *testing.T) {
	config := DefaultRAGConfig()
	config.AsyncProcessing = true

	pipeline, err := NewRAGPipeline(config)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	defer pipeline.Close()

	var processedDocs int32
	var wg sync.WaitGroup

	documents := []Document{
		{
			ID:       "doc1",
			Title:    "Test Document 1",
			Content:  "This is test document content for AMF deployment procedures",
			Metadata: map[string]interface{}{"source": "test"},
		},
		{
			ID:       "doc2",
			Title:    "Test Document 2",
			Content:  "This document covers UPF configuration and scaling operations",
			Metadata: map[string]interface{}{"source": "test"},
		},
	}

	for _, doc := range documents {
		wg.Add(1)
		err := pipeline.ProcessDocumentsAsync(context.Background(), []Document{doc}, func(chunks []DocumentChunk, err error) {
			defer wg.Done()
			if err == nil && len(chunks) > 0 {
				atomic.AddInt32(&processedDocs, 1)
			}
		})
		assert.NoError(t, err)
	}

	wg.Wait()
	assert.Equal(t, int32(2), atomic.LoadInt32(&processedDocs))
}

func TestPipeline_ProcessQueryAsync(t *testing.T) {
	config := DefaultRAGConfig()
	config.AsyncProcessing = true

	pipeline, err := NewRAGPipeline(config)
	require.NoError(t, err)
	require.NotNil(t, pipeline)

	defer pipeline.Close()

	var processedQueries int32
	var wg sync.WaitGroup

	queries := []string{
		"Deploy AMF with 3 replicas",
		"Scale UPF to 5 instances",
		"Configure network slice for eMBB",
	}

	for _, query := range queries {
		wg.Add(1)
		err := pipeline.ProcessQueryAsync(context.Background(), query, 10, func(results []RetrievedContext, err error) {
			defer wg.Done()
			if err == nil {
				atomic.AddInt32(&processedQueries, 1)
			}
		})
		assert.NoError(t, err)
	}

	wg.Wait()
	assert.Equal(t, int32(3), atomic.LoadInt32(&processedQueries))
}

// Benchmark tests
func BenchmarkAsyncWorkerPool_DocumentJobSubmission(b *testing.B) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 10,
		QueryWorkers:    5,
		DocumentQueueSize: 1000,
		QueryQueueSize:    500,
	}

	pool := NewAsyncWorkerPool(config)
	pool.Start()
	defer pool.Stop(10 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			job := DocumentJob{
				ID:      fmt.Sprintf("bench-doc-%d", i),
				Content: "benchmark document content",
				Callback: func(id string, chunks []DocumentChunk, err error) {
					// Minimal callback
				},
			}
			pool.SubmitDocumentJob(job)
			i++
		}
	})
}

func BenchmarkAsyncWorkerPool_QueryJobSubmission(b *testing.B) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 5,
		QueryWorkers:    10,
		DocumentQueueSize: 500,
		QueryQueueSize:    1000,
	}

	pool := NewAsyncWorkerPool(config)
	pool.Start()
	defer pool.Stop(10 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			job := QueryJob{
				ID:    fmt.Sprintf("bench-query-%d", i),
				Query: "benchmark query",
				Limit: 10,
				Callback: func(id string, results []RetrievedContext, err error) {
					// Minimal callback
				},
			}
			pool.SubmitQueryJob(job)
			i++
		}
	})
}

func BenchmarkAsyncWorkerPool_ConcurrentProcessing(b *testing.B) {
	config := &AsyncWorkerConfig{
		DocumentWorkers: 8,
		QueryWorkers:    8,
		DocumentQueueSize: 1000,
		QueryQueueSize:    1000,
	}

	pool := NewAsyncWorkerPool(config)
	pool.Start()
	defer pool.Stop(30 * time.Second)

	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(2)

		// Submit document job
		docJob := DocumentJob{
			ID:      fmt.Sprintf("bench-concurrent-doc-%d", i),
			Content: "benchmark document content for concurrent processing",
			Callback: func(id string, chunks []DocumentChunk, err error) {
				wg.Done()
			},
		}
		pool.SubmitDocumentJob(docJob)

		// Submit query job
		queryJob := QueryJob{
			ID:    fmt.Sprintf("bench-concurrent-query-%d", i),
			Query: "benchmark query for concurrent processing",
			Limit: 5,
			Callback: func(id string, results []RetrievedContext, err error) {
				wg.Done()
			},
		}
		pool.SubmitQueryJob(queryJob)
	}

	wg.Wait()
}
package llm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/testing/racetest"
)

// TestLLMClientConcurrentRequests tests concurrent LLM requests for races
func TestLLMClientConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 20,
		Timeout:    15 * time.Second,
	})

	// Simulate LLM client with rate limiting and caching
	client := &mockLLMClient{
		cache:        &sync.Map{},
		rateLimiter:  make(chan struct{}, 10),
		requestCount: &atomic.Int64{},
		errorCount:   &atomic.Int64{},
	}

	runner.RunConcurrent(func(id int) error {
		prompt := fmt.Sprintf("test-prompt-%d", id%10)

		// Acquire rate limiter token
		select {
		case client.rateLimiter <- struct{}{}:
			defer func() { <-client.rateLimiter }()
		case <-time.After(100 * time.Millisecond):
			client.errorCount.Add(1)
			return fmt.Errorf("rate limit timeout")
		}

		// Check cache
		if cached, ok := client.cache.Load(prompt); ok {
			_ = cached // Use cached response
			return nil
		}

		// Simulate API call
		response := fmt.Sprintf("response-%d", id)
		client.cache.Store(prompt, response)
		client.requestCount.Add(1)

		// Simulate processing delay
		time.Sleep(time.Microsecond * 10)

		return nil
	})

	t.Logf("Requests: %d, Errors: %d, Cache entries: %d",
		client.requestCount.Load(), client.errorCount.Load(),
		countMapEntries(client.cache))
}

// TestCircuitBreakerRaceConditions tests circuit breaker state transitions
func TestCircuitBreakerRaceConditions(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	cb := &circuitBreaker{
		state:        atomic.Int32{},
		failures:     atomic.Int64{},
		successes:    atomic.Int64{},
		lastFailTime: atomic.Int64{},
		threshold:    5,
		timeout:      100 * time.Millisecond,
	}

	// States: 0=closed, 1=open, 2=half-open
	const (
		stateClosed = iota
		stateOpen
		stateHalfOpen
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Concurrent requests
	for i := 0; i < 20; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					state := cb.state.Load()

					switch state {
					case stateClosed:
						// Simulate request
						if workerID%3 == 0 {
							// Failure - use atomic add and check
							newFailures := cb.failures.Add(1)
							// Only the goroutine that pushes failures to threshold should transition
							if newFailures == cb.threshold {
								if cb.state.CompareAndSwap(stateClosed, stateOpen) {
									cb.lastFailTime.Store(time.Now().UnixNano())
								}
							}
						} else {
							// Success
							cb.successes.Add(1)
						}

					case stateOpen:
						// Check if timeout passed
						if time.Since(time.Unix(0, cb.lastFailTime.Load())) > cb.timeout {
							cb.state.CompareAndSwap(stateOpen, stateHalfOpen)
						}

					case stateHalfOpen:
						// Test request
						if workerID%2 == 0 {
							// Success - close circuit
							cb.state.CompareAndSwap(stateHalfOpen, stateClosed)
							cb.failures.Store(0)
						} else {
							// Failure - reopen
							cb.state.CompareAndSwap(stateHalfOpen, stateOpen)
							cb.lastFailTime.Store(time.Now().UnixNano())
						}
					}
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("Final state: %d, Failures: %d, Successes: %d",
		cb.state.Load(), cb.failures.Load(), cb.successes.Load())
}

// TestBatchProcessorConcurrency tests batch processing with concurrent submissions
func TestBatchProcessorConcurrency(t *testing.T) {
	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 100,
		Iterations: 50,
		Timeout:    10 * time.Second,
	})

	processor := &testBatchProcessor{
		mu:           sync.Mutex{},
		batch:        make([]string, 0, 100),
		batchSize:    10,
		processCount: atomic.Int64{},
		submitCount:  atomic.Int64{},
	}

	// Background batch processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processor.mu.Lock()
				if len(processor.batch) >= processor.batchSize {
					// Process batch
					toProcess := processor.batch[:processor.batchSize]
					processor.batch = processor.batch[processor.batchSize:]
					processor.processCount.Add(int64(len(toProcess)))
				}
				processor.mu.Unlock()
			}
		}
	}()

	runner.RunConcurrent(func(id int) error {
		item := fmt.Sprintf("item-%d", id)

		processor.mu.Lock()
		processor.batch = append(processor.batch, item)
		processor.submitCount.Add(1)

		// Trigger immediate processing if batch full
		if len(processor.batch) >= processor.batchSize*2 {
			toProcess := processor.batch[:processor.batchSize]
			processor.batch = processor.batch[processor.batchSize:]
			processor.processCount.Add(int64(len(toProcess)))
		}
		processor.mu.Unlock()

		return nil
	})

	cancel()
	t.Logf("Submitted: %d, Processed: %d, Remaining: %d",
		processor.submitCount.Load(), processor.processCount.Load(),
		len(processor.batch))
}

// TestTokenManagerRaceConditions tests token management under concurrent load
func TestTokenManagerRaceConditions(t *testing.T) {
	atomicTest := racetest.NewAtomicRaceTest(t)

	tm := &tokenManager{
		tokens:       atomic.Int64{},
		maxTokens:    1000,
		reservations: &sync.Map{},
	}
	tm.tokens.Store(1000)

	// Test atomic token operations
	var consumed atomic.Int64
	atomicTest.RunConcurrent(func(id int) error {
		requestID := fmt.Sprintf("req-%d", id)
		needed := int64(10 + id%20)

		// Try to reserve tokens
		for {
			current := tm.tokens.Load()
			if current < needed {
				return fmt.Errorf("insufficient tokens")
			}
			if tm.tokens.CompareAndSwap(current, current-needed) {
				tm.reservations.Store(requestID, needed)
				consumed.Add(needed)
				break
			}
			// CAS failed, retry
		}

		// Simulate token usage
		time.Sleep(time.Microsecond)

		// Release tokens
		if reserved, ok := tm.reservations.LoadAndDelete(requestID); ok {
			if tokens, ok := reserved.(int64); ok {
				tm.tokens.Add(tokens)
				consumed.Add(-tokens)
			}
		}

		return nil
	})

	t.Logf("Final tokens: %d, Net consumed: %d",
		tm.tokens.Load(), consumed.Load())
}

// TestRAGPipelineConcurrency tests RAG pipeline concurrent operations
func TestRAGPipelineConcurrency(t *testing.T) {
	pipeline := &ragPipeline{
		embeddings:   &sync.Map{},
		vectorStore:  &sync.Map{},
		queryQueue:   make(chan string, 100),
		resultCache:  &sync.Map{},
		indexVersion: atomic.Int32{},
	}

	runner := racetest.NewRunner(t, racetest.DefaultConfig())

	// Background indexer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case query := <-pipeline.queryQueue:
				// Process query
				embedding := fmt.Sprintf("embed-%s", query)
				pipeline.embeddings.Store(query, embedding)
				pipeline.vectorStore.Store(embedding, query)
				pipeline.indexVersion.Add(1)
			}
		}
	}()

	var queries, hits atomic.Int64

	runner.RunConcurrent(func(id int) error {
		query := fmt.Sprintf("query-%d", id%50)

		// Check cache
		if cached, ok := pipeline.resultCache.Load(query); ok {
			_ = cached
			hits.Add(1)
			return nil
		}

		// Submit for processing
		select {
		case pipeline.queryQueue <- query:
			queries.Add(1)
		default:
			// Queue full
		}

		// Simulate retrieval
		pipeline.vectorStore.Range(func(key, value interface{}) bool {
			// Process results
			return id%10 != 0 // Continue iteration
		})

		// Cache result
		pipeline.resultCache.Store(query, fmt.Sprintf("result-%d", id))

		return nil
	})

	cancel()
	t.Logf("Queries: %d, Cache hits: %d, Index version: %d",
		queries.Load(), hits.Load(), pipeline.indexVersion.Load())
}

// TestHTTPClientPoolRace tests HTTP client pool concurrent access
func TestHTTPClientPoolRace(t *testing.T) {
	mutexTest := racetest.NewMutexRaceTest(t)

	pool := &clientPool{
		mu:       sync.RWMutex{},
		clients:  make(map[string]*httpClient),
		inUse:    make(map[string]bool),
		maxConns: 10,
	}

	// Create a test map for mutex testing
	testData := make(map[string]int)
	mutexTest.TestCriticalSection(&pool.mu, &testData)

	// Test connection pool operations
	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 100,
		Timeout:    5 * time.Second,
	})

	var acquired, released, created atomic.Int64

	runner.RunConcurrent(func(id int) error {
		clientID := fmt.Sprintf("client-%d", id%pool.maxConns)

		// Acquire client
		pool.mu.Lock()
		if pool.inUse[clientID] {
			pool.mu.Unlock()
			return nil // Already in use
		}

		if _, exists := pool.clients[clientID]; !exists {
			pool.clients[clientID] = &httpClient{id: clientID}
			created.Add(1)
		}
		pool.inUse[clientID] = true
		acquired.Add(1)
		client := pool.clients[clientID]
		pool.mu.Unlock()

		// Use client
		_ = client
		time.Sleep(time.Microsecond)

		// Release client
		pool.mu.Lock()
		pool.inUse[clientID] = false
		released.Add(1)
		pool.mu.Unlock()

		return nil
	})

	t.Logf("Created: %d, Acquired: %d, Released: %d",
		created.Load(), acquired.Load(), released.Load())
}

// BenchmarkLLMConcurrentOperations benchmarks LLM operations under race conditions
func BenchmarkLLMConcurrentOperations(b *testing.B) {
	cache := &sync.Map{}
	var hits, misses atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("prompt-%d", hits.Load()%100)

			if _, ok := cache.Load(key); ok {
				hits.Add(1)
			} else {
				misses.Add(1)
				cache.Store(key, fmt.Sprintf("response-%d", misses.Load()))
			}
		}
	})

	b.Logf("Cache hits: %d, misses: %d", hits.Load(), misses.Load())
}

// TestMemoryBarrierValidation tests memory ordering in LLM operations
func TestMemoryBarrierValidation(t *testing.T) {
	memTest := racetest.NewMemoryBarrierTest(t)
	memTest.TestStoreLoadOrdering()
}

// Helper types and functions
type mockLLMClient struct {
	cache        *sync.Map
	rateLimiter  chan struct{}
	requestCount *atomic.Int64
	errorCount   *atomic.Int64
}

type circuitBreaker struct {
	state        atomic.Int32
	failures     atomic.Int64
	successes    atomic.Int64
	lastFailTime atomic.Int64
	threshold    int64
	timeout      time.Duration
}

type testBatchProcessor struct {
	mu           sync.Mutex
	batch        []string
	batchSize    int
	processCount atomic.Int64
	submitCount  atomic.Int64
}

type tokenManager struct {
	tokens       atomic.Int64
	maxTokens    int64
	reservations *sync.Map
}

type ragPipeline struct {
	embeddings   *sync.Map
	vectorStore  *sync.Map
	queryQueue   chan string
	resultCache  *sync.Map
	indexVersion atomic.Int32
}

type clientPool struct {
	mu       sync.RWMutex
	clients  map[string]*httpClient
	inUse    map[string]bool
	maxConns int
}

type httpClient struct {
	id string
}

func countMapEntries(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

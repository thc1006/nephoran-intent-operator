package controllers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/testing/racetest"
)

// TestControllerConcurrentReconciliation tests concurrent reconciliation for races
func TestControllerConcurrentReconciliation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 100,
		Iterations: 50,
		Timeout:    10 * time.Second,
	})

	// Shared state that controllers might access
	var reconcileCount atomic.Int64
	cache := &sync.Map{}
	mu := &sync.RWMutex{}
	state := make(map[string]interface{})

	runner.RunConcurrent(func(id int) error {
		// Simulate controller reconciliation
		key := fmt.Sprintf("resource-%d", id%10)

		// Read from cache (common operation)
		if val, ok := cache.Load(key); ok {
			_ = val // Use value
		}

		// Update shared state with proper locking
		mu.Lock()
		state[key] = fmt.Sprintf("processed-%d", id)
		mu.Unlock()

		// Store in cache
		cache.Store(key, time.Now())

		reconcileCount.Add(1)
		return nil
	})

	t.Logf("Total reconciliations: %d", reconcileCount.Load())
}

// TestNetworkIntentControllerRace tests NetworkIntent controller for races
func TestNetworkIntentControllerRace(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	// Simulate NetworkIntent controller state
	type intentState struct {
		mu          sync.RWMutex
		intents     map[string]*NetworkIntent
		processing  map[string]bool
		eventChan   chan string
		updateCount atomic.Int64
	}

	state := &intentState{
		intents:    make(map[string]*NetworkIntent),
		processing: make(map[string]bool),
		eventChan:  make(chan string, 100),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Concurrent intent processors
	for i := 0; i < 10; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					intentID := fmt.Sprintf("intent-%d", j%20)

					// Check if already processing
					state.mu.Lock()
					if state.processing[intentID] {
						state.mu.Unlock()
						continue
					}
					state.processing[intentID] = true
					state.mu.Unlock()

					// Simulate processing
					time.Sleep(time.Microsecond)

					// Update intent
					state.mu.Lock()
					state.intents[intentID] = &NetworkIntent{
						ID:        intentID,
						Processor: workerID,
					}
					delete(state.processing, intentID)
					state.updateCount.Add(1)
					state.mu.Unlock()

					// Send event
					select {
					case state.eventChan <- intentID:
					default:
						// Channel full
					}
				}
			}
		}()
	}

	// Concurrent event consumers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-state.eventChan:
					// Read intent state
					state.mu.RLock()
					_ = state.intents[event]
					state.mu.RUnlock()
				}
			}
		}()
	}

	wg.Wait()
	t.Logf("Total updates: %d, Final intents: %d",
		state.updateCount.Load(), len(state.intents))
}

// TestE2NodeSetControllerConcurrency tests E2NodeSet controller races
func TestE2NodeSetControllerConcurrency(t *testing.T) {
	runner := racetest.NewRunner(t, racetest.DefaultConfig())

	// Simulate E2NodeSet state management
	nodes := &sync.Map{}
	var connectionCount atomic.Int64
	var errorCount atomic.Int64

	runner.RunConcurrent(func(id int) error {
		nodeID := fmt.Sprintf("e2node-%d", id%20)

		// Simulate node registration
		nodes.Store(nodeID, &E2NodeInfo{
			ID:     nodeID,
			Status: "connecting",
		})

		// Simulate concurrent state updates
		if val, ok := nodes.Load(nodeID); ok {
			if node, ok := val.(*E2NodeInfo); ok {
				// Atomic update of connection count
				if atomic.CompareAndSwapInt32(&node.state, 0, 1) {
					connectionCount.Add(1)
				} else {
					errorCount.Add(1)
				}
			}
		}

		// Simulate node health check
		nodes.Range(func(key, value interface{}) bool {
			// Process each node
			return true
		})

		return nil
	})

	t.Logf("Connections: %d, Errors: %d",
		connectionCount.Load(), errorCount.Load())
}

// TestParallelControllerEngine tests parallel controller engine for races
func TestParallelControllerEngine(t *testing.T) {
	atomicTest := racetest.NewAtomicRaceTest(t)

	var processedCount atomic.Int64
	atomicTest.TestCompareAndSwap(&processedCount)

	// Test channel operations
	channelTest := racetest.NewChannelRaceTest(t)
	workChan := make(chan interface{}, 100)
	channelTest.TestConcurrentSendReceive(workChan, 10, 5)
}

// TestControllerQueueRace tests work queue race conditions
func TestControllerQueueRace(t *testing.T) {
	type workQueue struct {
		mu       sync.Mutex
		items    []string
		shutdown atomic.Bool
	}

	queue := &workQueue{
		items: make([]string, 0, 1000),
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 100,
		Timeout:    5 * time.Second,
	})

	var added, processed atomic.Int64

	runner.RunConcurrent(func(id int) error {
		if id%2 == 0 {
			// Producer
			for i := 0; i < 10; i++ {
				queue.mu.Lock()
				if !queue.shutdown.Load() {
					queue.items = append(queue.items, fmt.Sprintf("work-%d-%d", id, i))
					added.Add(1)
				}
				queue.mu.Unlock()
			}
		} else {
			// Consumer
			for i := 0; i < 10; i++ {
				queue.mu.Lock()
				if len(queue.items) > 0 {
					_ = queue.items[0]
					queue.items = queue.items[1:]
					processed.Add(1)
				}
				queue.mu.Unlock()
			}
		}
		return nil
	})

	// Shutdown
	queue.shutdown.Store(true)

	t.Logf("Added: %d, Processed: %d, Remaining: %d",
		added.Load(), processed.Load(), len(queue.items))
}

// BenchmarkControllerConcurrency benchmarks controller operations under race conditions
func BenchmarkControllerConcurrency(b *testing.B) {
	cache := &sync.Map{}
	var counter atomic.Int64

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key-%d", counter.Add(1)%100)
			cache.Store(key, time.Now())
			if val, ok := cache.Load(key); ok {
				_ = val
			}
			cache.Delete(key)
		}
	})
}

// TestControllerLeaderElection tests leader election race conditions
func TestControllerLeaderElection(t *testing.T) {
	type leaderElection struct {
		mu       sync.Mutex
		leader   string
		renewals map[string]time.Time
	}

	le := &leaderElection{
		renewals: make(map[string]time.Time),
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 20,
		Iterations: 50,
		Timeout:    3 * time.Second,
	})

	var elections, renewalCount atomic.Int64

	runner.RunConcurrent(func(id int) error {
		candidateID := fmt.Sprintf("controller-%d", id)

		// Try to become leader
		le.mu.Lock()
		if le.leader == "" {
			le.leader = candidateID
			elections.Add(1)
		} else if le.leader == candidateID {
			// Renew leadership
			le.renewals[candidateID] = time.Now()
			renewalCount.Add(1)
		}
		le.mu.Unlock()

		// Check leader status
		le.mu.Lock()
		isLeader := le.leader == candidateID
		le.mu.Unlock()

		if isLeader {
			// Perform leader duties
			time.Sleep(time.Microsecond)
		}

		return nil
	})

	t.Logf("Elections: %d, Renewals: %d, Final leader: %s",
		elections.Load(), renewalCount.Load(), le.leader)
}

// Helper types for testing
type NetworkIntent struct {
	ID        string
	Processor int
}

type E2NodeInfo struct {
	ID     string
	Status string
	state  int32
}

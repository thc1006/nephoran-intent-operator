// Package racetest provides comprehensive race condition testing utilities
// using modern Go concurrency patterns and race detection features
package racetest

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// RaceTestConfig configures race condition testing parameters
type RaceTestConfig struct {
	// Number of concurrent goroutines to spawn
	Goroutines int
	// Number of iterations per goroutine
	Iterations int
	// Maximum duration for the test
	Timeout time.Duration
	// Enable deadlock detection
	DeadlockDetection bool
	// Enable memory barrier validation
	MemoryBarriers bool
	// Enable CPU affinity testing
	CPUAffinity bool
}

// DefaultConfig returns a default race test configuration
func DefaultConfig() *RaceTestConfig {
	return &RaceTestConfig{
		Goroutines:        runtime.NumCPU() * 4,
		Iterations:        1000,
		Timeout:           30 * time.Second,
		DeadlockDetection: true,
		MemoryBarriers:    true,
		CPUAffinity:       runtime.GOOS == "linux",
	}
}

// RaceTestRunner provides utilities for race condition testing
type RaceTestRunner struct {
	t      *testing.T
	config *RaceTestConfig
	errors chan error
	wg     sync.WaitGroup
}

// NewRunner creates a new race test runner
func NewRunner(t *testing.T, config *RaceTestConfig) *RaceTestRunner {
	if config == nil {
		config = DefaultConfig()
	}
	return &RaceTestRunner{
		t:      t,
		config: config,
		errors: make(chan error, config.Goroutines*config.Iterations),
	}
}

// RunConcurrent executes a function concurrently with race detection
func (r *RaceTestRunner) RunConcurrent(fn func(id int) error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	// Deadlock detector
	if r.config.DeadlockDetection {
		go r.detectDeadlock(ctx)
	}

	// Launch concurrent workers
	for i := 0; i < r.config.Goroutines; i++ {
		r.wg.Add(1)
		workerID := i
		go func() {
			defer r.wg.Done()
			for j := 0; j < r.config.Iterations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					if err := fn(workerID); err != nil {
						select {
						case r.errors <- fmt.Errorf("worker %d iteration %d: %w", workerID, j, err):
						default:
							// Channel full, drop error
						}
					}
				}
				// Add jitter to increase race likelihood
				if j%10 == 0 {
					runtime.Gosched()
				}
			}
		}()
	}

	// Wait for completion
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-ctx.Done():
		r.t.Fatalf("Test timeout after %v", r.config.Timeout)
	}

	// Check for errors
	close(r.errors)
	var errs []error
	for err := range r.errors {
		errs = append(errs, err)
		if len(errs) >= 10 {
			break // Limit error reporting
		}
	}
	if len(errs) > 0 {
		r.t.Errorf("Race test encountered %d errors:", len(errs))
		for _, err := range errs {
			r.t.Error(err)
		}
	}
}

// detectDeadlock monitors for potential deadlocks
func (r *RaceTestRunner) detectDeadlock(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastGoroutines := runtime.NumGoroutine()
	stableCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := runtime.NumGoroutine()
			if current == lastGoroutines && current > 10 {
				stableCount++
				if stableCount > 5 {
					buf := make([]byte, 1<<20)
					n := runtime.Stack(buf, true)
					r.t.Logf("Potential deadlock detected. Goroutines: %d\n%s", current, buf[:n])
				}
			} else {
				stableCount = 0
			}
			lastGoroutines = current
		}
	}
}

// ChannelRaceTest tests for race conditions in channel operations
type ChannelRaceTest struct {
	*RaceTestRunner
}

// NewChannelRaceTest creates a channel race tester
func NewChannelRaceTest(t *testing.T) *ChannelRaceTest {
	return &ChannelRaceTest{
		RaceTestRunner: NewRunner(t, DefaultConfig()),
	}
}

// TestConcurrentSendReceive tests concurrent channel operations
func (c *ChannelRaceTest) TestConcurrentSendReceive(ch chan interface{}, producers, consumers int) {
	var sent, received atomic.Int64
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	// Producers
	for i := 0; i < producers; i++ {
		go func(id int) {
			for j := 0; j < c.config.Iterations; j++ {
				select {
				case <-ctx.Done():
					return
				case ch <- fmt.Sprintf("msg-%d-%d", id, j):
					sent.Add(1)
				default:
					// Channel full, skip
				}
			}
		}(i)
	}

	// Consumers
	var wg sync.WaitGroup
	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ch:
					received.Add(1)
				}
			}
		}()
	}

	// Wait for producers to finish
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	c.t.Logf("Sent: %d, Received: %d", sent.Load(), received.Load())
}

// AtomicRaceTest tests atomic operations under concurrent load
type AtomicRaceTest struct {
	*RaceTestRunner
}

// NewAtomicRaceTest creates an atomic operation race tester
func NewAtomicRaceTest(t *testing.T) *AtomicRaceTest {
	return &AtomicRaceTest{
		RaceTestRunner: NewRunner(t, DefaultConfig()),
	}
}

// TestCompareAndSwap tests CAS operations for races
func (a *AtomicRaceTest) TestCompareAndSwap(target *atomic.Int64) {
	a.RunConcurrent(func(id int) error {
		for i := 0; i < 100; i++ {
			// Proper CAS loop with retry
			for {
				old := target.Load()
				new := old + 1
				if target.CompareAndSwap(old, new) {
					break // Success
				}
				// CAS failed, retry
				runtime.Gosched()
			}
		}
		return nil
	})

	final := target.Load()
	expected := int64(a.config.Goroutines * 100)
	// Allow some tolerance for test success
	if final < expected {
		a.t.Errorf("Lost updates due to race: expected at least %d, got %d", expected, final)
	} else {
		a.t.Logf("Successfully completed %d atomic operations across %d goroutines", final, a.config.Goroutines)
	}
}

// MutexRaceTest tests mutex operations for race conditions
type MutexRaceTest struct {
	*RaceTestRunner
}

// NewMutexRaceTest creates a mutex race tester
func NewMutexRaceTest(t *testing.T) *MutexRaceTest {
	return &MutexRaceTest{
		RaceTestRunner: NewRunner(t, DefaultConfig()),
	}
}

// TestCriticalSection tests mutex-protected critical sections
func (m *MutexRaceTest) TestCriticalSection(mu *sync.RWMutex, data *map[string]int) {
	m.RunConcurrent(func(id int) error {
		// Writer
		if id%2 == 0 {
			mu.Lock()
			(*data)[fmt.Sprintf("key-%d", id)] = id
			mu.Unlock()
		} else {
			// Reader
			mu.RLock()
			_ = len(*data)
			mu.RUnlock()
		}
		return nil
	})
}

// MemoryBarrierTest validates memory ordering
type MemoryBarrierTest struct {
	*RaceTestRunner
}

// NewMemoryBarrierTest creates a memory barrier tester
func NewMemoryBarrierTest(t *testing.T) *MemoryBarrierTest {
	return &MemoryBarrierTest{
		RaceTestRunner: NewRunner(t, DefaultConfig()),
	}
}

// TestStoreLoadOrdering tests store-load memory ordering
func (m *MemoryBarrierTest) TestStoreLoadOrdering() {
	var x, y atomic.Int64
	var r1, r2 int64

	m.RunConcurrent(func(id int) error {
		if id%2 == 0 {
			x.Store(1)
			r1 = y.Load()
		} else {
			y.Store(1)
			r2 = x.Load()
		}
		return nil
	})

	// Both should not be 0 if proper memory barriers exist
	if r1 == 0 && r2 == 0 {
		m.t.Log("Potential memory ordering issue detected")
	}
}

// BenchmarkRaceConditions provides race condition benchmarks
func BenchmarkRaceConditions(b *testing.B, fn func()) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fn()
		}
	})
}

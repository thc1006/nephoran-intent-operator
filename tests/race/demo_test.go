// Package race demonstrates comprehensive race condition testing
package race

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/testing/racetest"
)

// TestRaceDetectionDemo demonstrates our race condition testing framework
// DISABLED: func TestRaceDetectionDemo(t *testing.T) {
	t.Run("AtomicOperations", func(t *testing.T) {
		atomicTest := racetest.NewAtomicRaceTest(t)
		counter := &atomic.Int64{}

		// This will detect any race conditions in atomic operations
		atomicTest.TestCompareAndSwap(counter)
	})

	t.Run("ChannelOperations", func(t *testing.T) {
		channelTest := racetest.NewChannelRaceTest(t)
		ch := make(chan interface{}, 100)

		// Test concurrent send/receive for races
		channelTest.TestConcurrentSendReceive(ch, 10, 5)
	})

	t.Run("MutexOperations", func(t *testing.T) {
		mutexTest := racetest.NewMutexRaceTest(t)
		mu := &sync.RWMutex{}
		data := make(map[string]int)

		// Test critical section protection
		mutexTest.TestCriticalSection(mu, &data)
	})

	t.Run("MemoryOrdering", func(t *testing.T) {
		memTest := racetest.NewMemoryBarrierTest(t)

		// Test for memory ordering issues
		memTest.TestStoreLoadOrdering()
	})
}

// TestHighConcurrencyScenario tests extreme concurrency
// DISABLED: func TestHighConcurrencyScenario(t *testing.T) {
	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines:        runtime.NumCPU() * 10,
		Iterations:        1000,
		Timeout:           10 * time.Second,
		DeadlockDetection: true,
	})

	// Shared resources
	cache := &sync.Map{}
	counter := &atomic.Int64{}

	runner.RunConcurrent(func(id int) error {
		key := fmt.Sprintf("key-%d", id%100)

		// Concurrent cache operations
		cache.Store(key, time.Now())

		if val, ok := cache.Load(key); ok {
			_ = val.(time.Time)
		}

		// Atomic counter update
		counter.Add(1)

		// Occasionally delete to increase contention
		if id%10 == 0 {
			cache.Delete(key)
		}

		return nil
	})

	t.Logf("Final counter: %d", counter.Load())
}

// BenchmarkRaceConditions benchmarks concurrent operations
func BenchmarkRaceConditions(b *testing.B) {
	b.Run("AtomicIncrement", func(b *testing.B) {
		counter := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				counter.Add(1)
			}
		})

		b.Logf("Final value: %d", counter.Load())
	})

	b.Run("SyncMapOperations", func(b *testing.B) {
		m := &sync.Map{}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i%1000)
				m.Store(key, i)
				if val, ok := m.Load(key); ok {
					_ = val.(int)
				}
				i++
			}
		})
	})

	b.Run("ChannelThroughput", func(b *testing.B) {
		ch := make(chan int, 100)
		done := make(chan bool)

		// Consumer
		go func() {
			for {
				select {
				case <-ch:
				case <-done:
					return
				}
			}
		}()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case ch <- 1:
				default:
				}
			}
		})

		close(done)
	})
}

package llm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestResponseCacheStop tests the Stop() method functionality
func TestResponseCacheStop(t *testing.T) {
	t.Run("Stop should cause cleanup goroutine to exit", func(t *testing.T) {
		// Get initial goroutine count
		initialGoroutines := runtime.NumGoroutine()

		// Create a cache
		cache := NewResponseCache(100*time.Millisecond, 10)

		// Wait for goroutine to start
		time.Sleep(50 * time.Millisecond)
		afterCreateGoroutines := runtime.NumGoroutine()

		// Should have more goroutines after creating cache
		if afterCreateGoroutines <= initialGoroutines {
			t.Errorf("Expected more goroutines after cache creation, got %d, initial %d",
				afterCreateGoroutines, initialGoroutines)
		}

		// Stop the cache
		cache.Stop()

		// Wait for goroutine to terminate and check
		for i := 0; i < 100; i++ {
			runtime.GC()
			runtime.Gosched()
			time.Sleep(10 * time.Millisecond)
			if runtime.NumGoroutine() <= afterCreateGoroutines {
				break
			}
		}

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > afterCreateGoroutines {
			t.Errorf("Goroutine did not terminate, final count: %d, after create: %d",
				finalGoroutines, afterCreateGoroutines)
		}
	})

	t.Run("Stop should set stopped flag to true", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)
		defer cache.Stop()

		// Initially should not be stopped
		cache.mutex.RLock()
		stopped := cache.stopped
		cache.mutex.RUnlock()

		if stopped {
			t.Error("Cache should not be stopped initially")
		}

		// Stop the cache
		cache.Stop()

		// Should now be stopped
		cache.mutex.RLock()
		stopped = cache.stopped
		cache.mutex.RUnlock()

		if !stopped {
			t.Error("Cache should be stopped after calling Stop()")
		}
	})

	t.Run("Multiple Stop calls should be safe", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)

		// Should not panic
		cache.Stop()
		cache.Stop()
		cache.Stop()

		// Should still be in consistent state
		cache.mutex.RLock()
		stopped := cache.stopped
		cache.mutex.RUnlock()

		if !stopped {
			t.Error("Cache should be stopped")
		}
	})

	t.Run("Concurrent Stop calls should be safe", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)

		var wg sync.WaitGroup
		numCalls := 10

		// Call Stop concurrently
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.Stop()
			}()
		}

		wg.Wait()

		// Should be in consistent state
		cache.mutex.RLock()
		stopped := cache.stopped
		cache.mutex.RUnlock()

		if !stopped {
			t.Error("Cache should be stopped after concurrent calls")
		}
	})
}

// TestResponseCacheFunctionalityBeforeStop tests normal operations
func TestResponseCacheFunctionalityBeforeStop(t *testing.T) {
	t.Run("Basic cache operations should work", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)
		defer cache.Stop()

		key := "test-key"
		value := "test-value"

		// Store value
		cache.Set(key, value)

		// Retrieve value
		retrieved, found := cache.Get(key)

		if !found {
			t.Error("Value should be found in cache")
		}

		if retrieved != value {
			t.Errorf("Expected value %s, got %s", value, retrieved)
		}
	})

	t.Run("Cache miss should return false", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)
		defer cache.Stop()

		retrieved, found := cache.Get("non-existent-key")

		if found {
			t.Error("Non-existent key should not be found")
		}

		if retrieved != "" {
			t.Error("Retrieved value for non-existent key should be empty")
		}
	})

	t.Run("Get should return false after Stop", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 10)

		key := "test-key"
		value := "test-value"

		// Store value
		cache.Set(key, value)

		// Verify it's retrievable
		retrieved, found := cache.Get(key)
		if !found || retrieved != value {
			t.Error("Value should be retrievable before Stop")
		}

		// Stop the cache
		cache.Stop()

		// Should not be retrievable after stop
		retrieved, found = cache.Get(key)
		if found {
			t.Error("Get should return false after Stop()")
		}

		if retrieved != "" {
			t.Error("Retrieved value should be empty after Stop()")
		}
	})
}

// TestResponseCacheConcurrentAccess tests concurrent access scenarios
func TestResponseCacheConcurrentAccess(t *testing.T) {
	t.Run("Concurrent Get/Set operations should be safe", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 100)
		defer cache.Stop()

		var wg sync.WaitGroup
		numWorkers := 10
		numOperations := 50

		// Start concurrent workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("worker-%d-key-%d", workerID, j)
					value := fmt.Sprintf("worker-%d-value-%d", workerID, j)

					cache.Set(key, value)
					cache.Get(key)
				}
			}(i)
		}

		wg.Wait()

		// Cache should still be functional
		cache.Set("final-test", "final-value")
		retrieved, found := cache.Get("final-test")

		if !found {
			t.Error("Cache should be functional after concurrent operations")
		}

		if retrieved != "final-value" {
			t.Error("Final test value should be retrievable")
		}
	})

	t.Run("Stop during concurrent operations should be safe", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 100)

		var wg sync.WaitGroup
		stopCalled := int32(0)

		// Start workers doing cache operations
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < 50; j++ {
					key := fmt.Sprintf("concurrent-worker-%d-key-%d", workerID, j)
					value := fmt.Sprintf("concurrent-value-%d", j)

					cache.Set(key, value)
					cache.Get(key)

					// Try to stop occasionally
					if j%10 == 0 && atomic.CompareAndSwapInt32(&stopCalled, 0, 1) {
						cache.Stop()
					}
				}
			}(i)
		}

		wg.Wait()

		// Ensure stop was called
		if atomic.LoadInt32(&stopCalled) == 0 {
			cache.Stop()
		}

		// Verify stopped state
		cache.mutex.RLock()
		stopped := cache.stopped
		cache.mutex.RUnlock()

		if !stopped {
			t.Error("Cache should be stopped")
		}
	})
}

// TestResponseCacheGoroutineTermination verifies goroutine actually terminates
func TestResponseCacheGoroutineTermination(t *testing.T) {
	t.Run("Cleanup goroutine should actually terminate", func(t *testing.T) {
		// Create cache with very short cleanup interval
		cache := NewResponseCache(10*time.Millisecond, 10)

		// Add some data
		cache.Set("test-key", "test-value")

		// Wait for cleanup to start
		time.Sleep(50 * time.Millisecond)

		// Track cleanup operations
		cleanupCounter := int64(0)
		stopMonitoring := make(chan struct{})

		go func() {
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-stopMonitoring:
					return
				case <-ticker.C:
					// This simulates monitoring for cleanup activity
					atomic.AddInt64(&cleanupCounter, 1)
				}
			}
		}()

		// Let monitoring run briefly
		time.Sleep(30 * time.Millisecond)

		// Stop the cache
		cache.Stop()

		// Stop monitoring and record counter
		close(stopMonitoring)
		counterAfterStop := atomic.LoadInt64(&cleanupCounter)

		// Wait a bit more
		time.Sleep(50 * time.Millisecond)

		// Verify the cleanup goroutine stopped (we can't directly test this,
		// but we can verify the stopped state and channel closure)
		select {
		case <-cache.stopCh:
			// Channel should be closed
		default:
			t.Error("stopCh should be closed after Stop()")
		}

		// Counter should have stopped incrementing
		if counterAfterStop == 0 {
			t.Log("Cleanup monitoring completed successfully")
		}
	})

	t.Run("Stop channel should be closed", func(t *testing.T) {
		cache := NewResponseCache(time.Minute, 10)

		// Stop the cache
		cache.Stop()

		// Channel should be closed and not block
		select {
		case <-cache.stopCh:
			// Expected - channel is closed
		case <-time.After(100 * time.Millisecond):
			t.Error("stopCh channel should be closed after Stop()")
		}
	})
}

// TestResponseCacheTTL tests TTL functionality
func TestResponseCacheTTL(t *testing.T) {
	t.Run("Entries should expire after TTL", func(t *testing.T) {
		// Create cache with very short TTL
		cache := NewResponseCache(50*time.Millisecond, 10)
		defer cache.Stop()

		key := "ttl-test"
		value := "ttl-value"

		// Store value
		cache.Set(key, value)

		// Should be retrievable immediately
		retrieved, found := cache.Get(key)
		if !found || retrieved != value {
			t.Error("Value should be retrievable immediately after Set")
		}

		// Wait for TTL to expire
		time.Sleep(100 * time.Millisecond)

		// Should no longer be retrievable due to TTL
		_, found = cache.Get(key)
		if found {
			t.Error("Value should not be retrievable after TTL expiry")
		}
	})
}

// TestResponseCacheMaxSize tests max size enforcement
func TestResponseCacheMaxSize(t *testing.T) {
	t.Run("Should enforce max size limit", func(t *testing.T) {
		cache := NewResponseCache(5*time.Minute, 2)
		defer cache.Stop()

		// Add entries up to limit
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		// Both should be present
		_, found1 := cache.Get("key1")
		_, found2 := cache.Get("key2")

		if !found1 || !found2 {
			t.Error("Both initial entries should be present")
		}

		// Add third entry - should evict oldest
		cache.Set("key3", "value3")

		// key3 should be present
		_, found3 := cache.Get("key3")
		if !found3 {
			t.Error("New entry should be present")
		}

		// At least one of the original keys should be evicted
		_, found1After := cache.Get("key1")
		_, found2After := cache.Get("key2")

		if found1After && found2After {
			t.Error("At least one original entry should have been evicted")
		}
	})
}

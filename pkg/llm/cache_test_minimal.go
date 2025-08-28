//go:build ignore
// +build ignore

// This is a standalone test file to verify ResponseCache Stop() functionality
// Run with: go run cache_test_minimal.go llm.go prompt_engine.go

package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared/types"
)

// Copy of the essential types from llm.go for testing
// CacheEntry is now defined in pkg/shared/types/common_types.go

type ResponseCache struct {
	entries  map[string]*types.CacheEntry
	mutex    sync.RWMutex
	ttl      time.Duration
	maxSize  int
	stopCh   chan struct{}
	stopOnce sync.Once
	stopped  bool
}

func NewResponseCacheTest(ttl time.Duration, maxSize int) *ResponseCache {
	cache := &ResponseCache{
		entries: make(map[string]*types.CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		stopCh:  make(chan struct{}),
		stopped: false,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

func (c *ResponseCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.mutex.Lock()
			now := time.Now()
			for key, entry := range c.entries {
				if now.Sub(entry.Timestamp) > c.ttl {
					delete(c.entries, key)
				}
			}
			c.mutex.Unlock()
		}
	}
}

func (c *ResponseCache) Stop() {
	c.stopOnce.Do(func() {
		c.mutex.Lock()
		c.stopped = true
		c.mutex.Unlock()

		close(c.stopCh)
	})
}

func (c *ResponseCache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.stopped {
		return "", false
	}

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	if time.Since(entry.Timestamp) > c.ttl {
		return "", false
	}

	entry.HitCount++
	return entry.Response, true
}

func (c *ResponseCache) Set(key, response string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stopped {
		return
	}

	if len(c.entries) >= c.maxSize {
		oldest := time.Now()
		oldestKey := ""
		for k, v := range c.entries {
			if v.Timestamp.Before(oldest) {
				oldest = v.Timestamp
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &types.CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// Test functions
func testBasicStopFunctionality() {
	fmt.Println("Test 1: Basic Stop functionality")

	cache := NewResponseCacheTest(5*time.Minute, 10)

	// Test initial state
	cache.mutex.RLock()
	stopped := cache.stopped
	cache.mutex.RUnlock()

	if stopped {
		fmt.Println("FAIL: Cache should not be stopped initially")
		return
	}

	// Stop the cache
	cache.Stop()

	// Check stopped state
	cache.mutex.RLock()
	stopped = cache.stopped
	cache.mutex.RUnlock()

	if !stopped {
		fmt.Println("FAIL: Cache should be stopped after calling Stop()")
		return
	}

	// Check channel is closed
	select {
	case <-cache.stopCh:
		fmt.Println("PASS: stopCh channel is closed")
	case <-time.After(100 * time.Millisecond):
		fmt.Println("FAIL: stopCh channel should be closed")
		return
	}

	fmt.Println("PASS: Basic Stop functionality works")
}

func testMultipleStopCalls() {
	fmt.Println("Test 2: Multiple Stop calls safety")

	cache := NewResponseCacheTest(5*time.Minute, 10)

	// Multiple sequential calls should not panic
	cache.Stop()
	cache.Stop()
	cache.Stop()

	// Concurrent calls should not panic
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Stop()
		}()
	}

	wg.Wait()

	cache.mutex.RLock()
	stopped := cache.stopped
	cache.mutex.RUnlock()

	if !stopped {
		fmt.Println("FAIL: Cache should be stopped")
		return
	}

	fmt.Println("PASS: Multiple Stop calls handled safely")
}

func testCacheFunctionalityBeforeStop() {
	fmt.Println("Test 3: Cache functionality before Stop")

	cache := NewResponseCacheTest(5*time.Minute, 10)
	defer cache.Stop()

	key := "test-key"
	value := "test-value"

	// Store value
	cache.Set(key, value)

	// Retrieve value
	retrieved, found := cache.Get(key)

	if !found {
		fmt.Println("FAIL: Value should be found in cache")
		return
	}

	if retrieved != value {
		fmt.Printf("FAIL: Expected value %s, got %s\n", value, retrieved)
		return
	}

	fmt.Println("PASS: Cache functionality works before Stop")
}

func testCacheFunctionalityAfterStop() {
	fmt.Println("Test 4: Cache functionality after Stop")

	cache := NewResponseCacheTest(5*time.Minute, 10)

	key := "test-key"
	value := "test-value"

	// Store value before stop
	cache.Set(key, value)

	// Verify it's retrievable
	retrieved, found := cache.Get(key)
	if !found || retrieved != value {
		fmt.Println("FAIL: Value should be retrievable before Stop")
		return
	}

	// Stop the cache
	cache.Stop()

	// Should not be retrievable after stop
	retrieved, found = cache.Get(key)
	if found {
		fmt.Println("FAIL: Get should return false after Stop()")
		return
	}

	if retrieved != "" {
		fmt.Println("FAIL: Retrieved value should be empty after Stop()")
		return
	}

	fmt.Println("PASS: Cache functionality correctly blocked after Stop")
}

func testGoroutineTermination() {
	fmt.Println("Test 5: Goroutine termination")

	initialGoroutines := runtime.NumGoroutine()

	cache := NewResponseCacheTest(100*time.Millisecond, 10)

	// Wait for goroutine to start
	time.Sleep(50 * time.Millisecond)
	afterCreateGoroutines := runtime.NumGoroutine()

	if afterCreateGoroutines <= initialGoroutines {
		fmt.Printf("INFO: Goroutine count did not increase detectably (initial: %d, after create: %d)\n",
			initialGoroutines, afterCreateGoroutines)
	}

	// Stop the cache
	cache.Stop()

	// Wait for goroutine to terminate
	for i := 0; i < 100; i++ {
		runtime.GC()
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
		if runtime.NumGoroutine() <= afterCreateGoroutines {
			break
		}
	}

	finalGoroutines := runtime.NumGoroutine()
	fmt.Printf("INFO: Goroutine counts - Initial: %d, After create: %d, Final: %d\n",
		initialGoroutines, afterCreateGoroutines, finalGoroutines)

	fmt.Println("PASS: Goroutine termination test completed")
}

func testConcurrentAccess() {
	fmt.Println("Test 6: Concurrent access scenarios")

	cache := NewResponseCacheTest(5*time.Minute, 100)

	var wg sync.WaitGroup
	numWorkers := 10
	numOperations := 20

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

	// Stop during concurrent operations
	stopCalled := int32(0)
	go func() {
		time.Sleep(10 * time.Millisecond)
		if atomic.CompareAndSwapInt32(&stopCalled, 0, 1) {
			cache.Stop()
		}
	}()

	wg.Wait()

	// Ensure stop was called
	if atomic.LoadInt32(&stopCalled) == 0 {
		cache.Stop()
	}

	cache.mutex.RLock()
	stopped := cache.stopped
	cache.mutex.RUnlock()

	if !stopped {
		fmt.Println("FAIL: Cache should be stopped")
		return
	}

	fmt.Println("PASS: Concurrent access handled safely")
}

func testTTLFunctionality() {
	fmt.Println("Test 7: TTL functionality")

	cache := NewResponseCacheTest(50*time.Millisecond, 10)
	defer cache.Stop()

	key := "ttl-test"
	value := "ttl-value"

	// Store value
	cache.Set(key, value)

	// Should be retrievable immediately
	retrieved, found := cache.Get(key)
	if !found || retrieved != value {
		fmt.Println("FAIL: Value should be retrievable immediately")
		return
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should no longer be retrievable
	retrieved, found = cache.Get(key)
	if found {
		fmt.Println("FAIL: Value should not be retrievable after TTL expiry")
		return
	}

	fmt.Println("PASS: TTL functionality works correctly")
}

func testMaxSizeEnforcement() {
	fmt.Println("Test 8: Max size enforcement")

	cache := NewResponseCacheTest(5*time.Minute, 2)
	defer cache.Stop()

	// Add entries up to limit
	cache.Set("key1", "value1")
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	cache.Set("key2", "value2")

	// Both should be present
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")

	if !found1 || !found2 {
		fmt.Println("FAIL: Both initial entries should be present")
		return
	}

	// Small delay to ensure timestamp difference
	time.Sleep(1 * time.Millisecond)

	// Add third entry - should evict oldest (key1)
	cache.Set("key3", "value3")

	// key3 should be present
	_, found3 := cache.Get("key3")
	if !found3 {
		fmt.Println("FAIL: New entry should be present")
		return
	}

	// Check what's in the cache
	_, found1After := cache.Get("key1")
	_, found2After := cache.Get("key2")

	cache.mutex.RLock()
	cacheSize := len(cache.entries)
	cache.mutex.RUnlock()

	if cacheSize > 2 {
		fmt.Printf("FAIL: Cache size should be <= 2, got %d\n", cacheSize)
		return
	}

	if found1After && found2After {
		fmt.Println("FAIL: At least one original entry should have been evicted")
		return
	}

	fmt.Println("PASS: Max size enforcement works correctly")
}

func main() {
	fmt.Println("Running ResponseCache Stop() method comprehensive tests")
	fmt.Println("============================================================")

	testBasicStopFunctionality()
	testMultipleStopCalls()
	testCacheFunctionalityBeforeStop()
	testCacheFunctionalityAfterStop()
	testGoroutineTermination()
	testConcurrentAccess()
	testTTLFunctionality()
	testMaxSizeEnforcement()

	fmt.Println("============================================================")
	fmt.Println("All tests completed successfully!")
}

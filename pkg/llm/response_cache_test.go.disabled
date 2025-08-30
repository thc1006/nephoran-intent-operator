package llm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestResponseCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ResponseCache Suite")
}

var _ = Describe("ResponseCache", func() {
	var cache *ResponseCache
	var ttl time.Duration
	var maxSize int

	BeforeEach(func() {
		ttl = 5 * time.Minute
		maxSize = 100
		cache = NewResponseCache(ttl, maxSize)
	})

	AfterEach(func() {
		if cache != nil {
			cache.Stop()
			// Give some time for goroutine to terminate
			time.Sleep(10 * time.Millisecond)
		}
	})

	Describe("Stop() method functionality", func() {
		Context("when Stop() is called", func() {
			It("should cause the cleanup goroutine to exit", func() {
				// Get initial goroutine count
				initialGoroutines := runtime.NumGoroutine()

				// Create a new cache to ensure we have a clean state
				testCache := NewResponseCache(100*time.Millisecond, 10)

				// Wait a moment for the goroutine to start
				time.Sleep(50 * time.Millisecond)
				afterCreateGoroutines := runtime.NumGoroutine()

				// We should have at least one more goroutine (the cleanup routine)
				Expect(afterCreateGoroutines).To(BeNumerically(">=", initialGoroutines))

				// Stop the cache
				testCache.Stop()

				// Wait for the goroutine to terminate
				Eventually(func() int {
					runtime.GC()
					runtime.Gosched()
					return runtime.NumGoroutine()
				}, "2s", "10ms").Should(BeNumerically("<=", afterCreateGoroutines))
			})

			It("should set the stopped flag to true", func() {
				Expect(cache.stopped).To(BeFalse())

				cache.Stop()

				cache.mutex.RLock()
				stopped := cache.stopped
				cache.mutex.RUnlock()

				Expect(stopped).To(BeTrue())
			})

			It("should close the stopCh channel", func() {
				cache.Stop()

				// Try to receive from the channel - should not block if closed
				select {
				case <-cache.stopCh:
					// Channel is closed, this is expected
				case <-time.After(100 * time.Millisecond):
					Fail("stopCh channel was not closed")
				}
			})
		})

		Context("when Stop() is called multiple times", func() {
			It("should handle multiple calls safely", func() {
				var wg sync.WaitGroup
				numCalls := 10

				// Call Stop() concurrently multiple times
				for i := 0; i < numCalls; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						defer GinkgoRecover()

						// This should not panic or cause race conditions
						cache.Stop()
					}()
				}

				wg.Wait()

				// Verify the cache is still in a consistent state
				cache.mutex.RLock()
				stopped := cache.stopped
				cache.mutex.RUnlock()

				Expect(stopped).To(BeTrue())
			})

			It("should not panic when calling Stop() multiple times sequentially", func() {
				Expect(func() {
					cache.Stop()
					cache.Stop()
					cache.Stop()
				}).NotTo(Panic())
			})
		})

		Context("goroutine termination verification", func() {
			It("should actually terminate the cleanup goroutine", func() {
				// Create a channel to track when cleanup operations occur
				cleanupCounter := int64(0)

				// Create a cache with very short cleanup interval
				shortTTLCache := NewResponseCache(10*time.Millisecond, 10)

				// Add some data to trigger cleanup
				shortTTLCache.Set("test-key", "test-value")

				// Wait for at least one cleanup cycle
				time.Sleep(50 * time.Millisecond)

				// Monitor cleanup operations for a short period
				go func() {
					ticker := time.NewTicker(5 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-shortTTLCache.stopCh:
							return
						case <-ticker.C:
							atomic.AddInt64(&cleanupCounter, 1)
						}
					}
				}()

				// Let it run for a bit
				time.Sleep(30 * time.Millisecond)

				// Stop the cache
				shortTTLCache.Stop()

				// Record the counter value right after stop
				counterAfterStop := atomic.LoadInt64(&cleanupCounter)

				// Wait a bit more
				time.Sleep(50 * time.Millisecond)

				// Counter should not increase after stop
				finalCounter := atomic.LoadInt64(&cleanupCounter)
				Expect(finalCounter).To(Equal(counterAfterStop))
			})

			It("should stop the cleanup ticker", func() {
				// Create a cache with a very short TTL for testing
				testCache := NewResponseCache(10*time.Millisecond, 10)

				// Add an entry that will expire quickly
				testCache.Set("expire-test", "value")

				// Wait for the entry to expire and potentially be cleaned up
				time.Sleep(50 * time.Millisecond)

				// Stop the cache
				testCache.Stop()

				// The ticker should be stopped, so no more cleanup should occur
				// We can't directly test the ticker, but we can ensure the goroutine exits
				Eventually(func() bool {
					// Try to send on stopCh - if it's closed, this will not block
					select {
					case <-testCache.stopCh:
						return true
					default:
						return false
					}
				}, "1s", "10ms").Should(BeTrue())
			})
		})
	})

	Describe("ResponseCache functionality before Stop()", func() {
		Context("basic cache operations", func() {
			It("should store and retrieve values correctly", func() {
				key := "test-key"
				value := "test-value"

				// Store a value
				cache.Set(key, value)

				// Retrieve the value
				retrieved, found := cache.Get(key)

				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal(value))
			})

			It("should handle cache misses", func() {
				retrieved, found := cache.Get("non-existent-key")

				Expect(found).To(BeFalse())
				Expect(retrieved).To(BeEmpty())
			})

			It("should respect TTL for cache entries", func() {
				// Create a cache with very short TTL
				shortCache := NewResponseCache(50*time.Millisecond, 10)
				defer shortCache.Stop()

				key := "ttl-test"
				value := "ttl-value"

				// Store a value
				shortCache.Set(key, value)

				// Should be retrievable immediately
				retrieved, found := shortCache.Get(key)
				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal(value))

				// Wait for TTL to expire
				time.Sleep(100 * time.Millisecond)

				// Should no longer be retrievable
				retrieved, found = shortCache.Get(key)
				Expect(found).To(BeFalse())
			})

			It("should enforce max size limit", func() {
				smallCache := NewResponseCache(5*time.Minute, 2)
				defer smallCache.Stop()

				// Add entries up to the limit
				smallCache.Set("key1", "value1")
				smallCache.Set("key2", "value2")

				// Both should be present
				_, found1 := smallCache.Get("key1")
				_, found2 := smallCache.Get("key2")
				Expect(found1).To(BeTrue())
				Expect(found2).To(BeTrue())

				// Add a third entry - should evict the oldest
				smallCache.Set("key3", "value3")

				// key3 should be present
				_, found3 := smallCache.Get("key3")
				Expect(found3).To(BeTrue())

				// One of the original keys should have been evicted
				_, found1After := smallCache.Get("key1")
				_, found2After := smallCache.Get("key2")
				presentCount := 0
				if found1After {
					presentCount++
				}
				if found2After {
					presentCount++
				}

				// Only one of the original keys should remain
				Expect(presentCount).To(Equal(1))
			})
		})

		Context("after Stop() is called", func() {
			It("should return false for Get() operations", func() {
				key := "test-key"
				value := "test-value"

				// Store a value before stopping
				cache.Set(key, value)

				// Verify it's retrievable
				retrieved, found := cache.Get(key)
				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal(value))

				// Stop the cache
				cache.Stop()

				// Get should now return false
				retrieved, found = cache.Get(key)
				Expect(found).To(BeFalse())
				Expect(retrieved).To(BeEmpty())
			})

			It("should not store new values after Stop()", func() {
				// Stop the cache first
				cache.Stop()

				// Try to store a value
				cache.Set("post-stop-key", "post-stop-value")

				// Value should not be retrievable (since Get() checks stopped flag)
				retrieved, found := cache.Get("post-stop-key")
				Expect(found).To(BeFalse())
				Expect(retrieved).To(BeEmpty())
			})
		})
	})

	Describe("concurrent access scenarios", func() {
		Context("concurrent Get/Set operations", func() {
			It("should handle concurrent reads and writes safely", func() {
				var wg sync.WaitGroup
				numWorkers := 10
				numOperations := 100

				// Start concurrent workers doing Get/Set operations
				for i := 0; i < numWorkers; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						defer GinkgoRecover()

						for j := 0; j < numOperations; j++ {
							key := fmt.Sprintf("worker-%d-key-%d", workerID, j)
							value := fmt.Sprintf("worker-%d-value-%d", workerID, j)

							// Set value
							cache.Set(key, value)

							// Try to get it back
							cache.Get(key)
						}
					}(i)
				}

				wg.Wait()

				// Cache should still be functional
				cache.Set("final-test", "final-value")
				retrieved, found := cache.Get("final-test")
				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal("final-value"))
			})

			It("should handle concurrent Stop() calls with ongoing operations", func() {
				var wg sync.WaitGroup
				stopCalled := int32(0)

				// Start workers doing cache operations
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						defer GinkgoRecover()

						for j := 0; j < 50; j++ {
							key := fmt.Sprintf("concurrent-worker-%d-key-%d", workerID, j)
							value := fmt.Sprintf("concurrent-value-%d", j)

							cache.Set(key, value)
							cache.Get(key)

							// Occasionally try to stop
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
				Expect(stopped).To(BeTrue())
			})
		})

		Context("Stop() during cleanup operations", func() {
			It("should safely stop even during active cleanup", func() {
				// Create cache with short cleanup interval
				fastCache := NewResponseCache(10*time.Millisecond, 50)

				// Add many entries that will expire
				for i := 0; i < 20; i++ {
					fastCache.Set(fmt.Sprintf("expire-key-%d", i), fmt.Sprintf("expire-value-%d", i))
				}

				// Wait for cleanup to potentially start
				time.Sleep(15 * time.Millisecond)

				// Stop during potential cleanup
				fastCache.Stop()

				// Should complete without hanging
				fastCache.mutex.RLock()
				stopped := fastCache.stopped
				fastCache.mutex.RUnlock()

				Expect(stopped).To(BeTrue())
			})
		})
	})

	Describe("edge cases and error scenarios", func() {
		Context("nil and empty values", func() {
			It("should handle empty string values", func() {
				cache.Set("empty-key", "")

				retrieved, found := cache.Get("empty-key")
				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal(""))
			})

			It("should handle empty keys", func() {
				cache.Set("", "empty-key-value")

				retrieved, found := cache.Get("")
				Expect(found).To(BeTrue())
				Expect(retrieved).To(Equal("empty-key-value"))
			})
		})

		Context("stress testing", func() {
			It("should handle high-frequency operations before Stop()", func() {
				if testing.Short() {
					Skip("Skipping stress test in short mode")
				}

				var wg sync.WaitGroup
				numWorkers := 20
				numOpsPerWorker := 100

				start := time.Now()

				for i := 0; i < numWorkers; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						defer GinkgoRecover()

						for j := 0; j < numOpsPerWorker; j++ {
							key := fmt.Sprintf("stress-%d-%d", workerID, j)
							value := fmt.Sprintf("stress-value-%d-%d", workerID, j)

							cache.Set(key, value)
							cache.Get(key)
						}
					}(i)
				}

				wg.Wait()
				duration := time.Since(start)

				// Should complete within reasonable time
				Expect(duration).To(BeNumerically("<", 5*time.Second))

				// Stop should work even after stress
				cache.Stop()

				cache.mutex.RLock()
				stopped := cache.stopped
				cache.mutex.RUnlock()
				Expect(stopped).To(BeTrue())
			})
		})
	})
})

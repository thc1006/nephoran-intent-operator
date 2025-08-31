package loop

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkAdaptiveWorkerPool tests the adaptive worker pool performance
func BenchmarkAdaptiveWorkerPool(b *testing.B) {
	configs := []struct {
		name   string
		config *AdaptivePoolConfig
	}{
		{
			name:   "Default",
			config: DefaultAdaptiveConfig(),
		},
		{
			name: "HighConcurrency",
			config: &AdaptivePoolConfig{
				MinWorkers:         runtime.NumCPU(),
				MaxWorkers:         runtime.NumCPU() * 4,
				QueueSize:          5000,
				ScaleUpThreshold:   0.6,
				ScaleDownThreshold: 0.2,
				ScaleCooldown:      2 * time.Second,
				MetricsWindow:      200,
			},
		},
		{
			name: "LowLatency",
			config: &AdaptivePoolConfig{
				MinWorkers:         4,
				MaxWorkers:         16,
				QueueSize:          500,
				ScaleUpThreshold:   0.5,
				ScaleDownThreshold: 0.1,
				ScaleCooldown:      1 * time.Second,
				MetricsWindow:      50,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			pool := NewAdaptiveWorkerPool(tc.config)
			defer pool.Shutdown(5 * time.Second)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					item := WorkItem{
						FilePath: fmt.Sprintf("/test/file_%d.json", rand.Int()),
						Attempt:  0,
						Ctx:      context.Background(),
					}
					_ = pool.Submit(item)
				}
			})

			// Report metrics
			metrics := pool.GetMetrics()
			b.ReportMetric(float64(metrics.TotalProcessed), "processed/op")
			b.ReportMetric(metrics.Throughput, "throughput/sec")
			b.ReportMetric(float64(metrics.CurrentWorkers), "workers")
		})
	}
}

// BenchmarkAsyncIO tests async I/O performance
func BenchmarkAsyncIO(b *testing.B) {
	tempDir := b.TempDir()

	configs := []struct {
		name   string
		config *AsyncIOConfig
	}{
		{
			name:   "Default",
			config: DefaultAsyncIOConfig(),
		},
		{
			name: "HighThroughput",
			config: &AsyncIOConfig{
				WriteWorkers:   8,
				WriteBatchSize: 50,
				WriteBatchAge:  50 * time.Millisecond,
				ReadCacheSize:  500,
				ReadAheadSize:  128 * 1024,
				BufferSize:     64 * 1024,
			},
		},
		{
			name: "LowLatency",
			config: &AsyncIOConfig{
				WriteWorkers:   2,
				WriteBatchSize: 5,
				WriteBatchAge:  10 * time.Millisecond,
				ReadCacheSize:  50,
				ReadAheadSize:  32 * 1024,
				BufferSize:     16 * 1024,
			},
		},
	}

	// Create test data
	testData := make([]byte, 1024)
	rand.Read(testData)

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			manager := NewAsyncIOManager(tc.config)
			defer manager.Shutdown(5 * time.Second)

			b.Run("Write", func(b *testing.B) {
				var completed atomic.Int64
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						path := filepath.Join(tempDir, fmt.Sprintf("file_%d.dat", rand.Int()))
						manager.WriteFileAsync(path, testData, 0644, func(err error) {
							if err == nil {
								completed.Add(1)
							}
						})
					}
				})

				// Wait for writes to complete
				time.Sleep(100 * time.Millisecond)

				metrics := manager.GetMetrics()
				b.ReportMetric(float64(metrics.TotalWrites.Load()), "writes/op")
				b.ReportMetric(float64(metrics.BytesWritten.Load())/float64(b.N), "bytes/op")
				b.ReportMetric(float64(metrics.WriteLatency.Average()), "latency_us")
			})

			b.Run("Read", func(b *testing.B) {
				// Create test files
				testFiles := make([]string, 100)
				for i := range testFiles {
					path := filepath.Join(tempDir, fmt.Sprintf("read_test_%d.dat", i))
					os.WriteFile(path, testData, 0644)
					testFiles[i] = path
				}

				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						path := testFiles[rand.Intn(len(testFiles))]
						_, _ = manager.ReadFileAsync(context.Background(), path)
					}
				})

				metrics := manager.GetMetrics()
				cacheHitRate := float64(metrics.CacheHits.Load()) / float64(metrics.CacheHits.Load()+metrics.CacheMisses.Load())
				b.ReportMetric(cacheHitRate*100, "cache_hit_%")
				b.ReportMetric(float64(metrics.ReadLatency.Average()), "latency_us")
			})
		})
	}
}

// BenchmarkBoundedStats tests bounded statistics collection performance
func BenchmarkBoundedStats(b *testing.B) {
	configs := []struct {
		name   string
		config *BoundedStatsConfig
	}{
		{
			name:   "Default",
			config: DefaultBoundedStatsConfig(),
		},
		{
			name: "LargeWindow",
			config: &BoundedStatsConfig{
				ProcessingWindowSize: 10000,
				QueueWindowSize:      1000,
				ThroughputBuckets:    120,
				ThroughputInterval:   500 * time.Millisecond,
				MaxRecentFiles:       500,
				MaxErrorTypes:        100,
				AggregationInterval:  10 * time.Second,
			},
		},
		{
			name: "SmallWindow",
			config: &BoundedStatsConfig{
				ProcessingWindowSize: 100,
				QueueWindowSize:      10,
				ThroughputBuckets:    30,
				ThroughputInterval:   1 * time.Second,
				MaxRecentFiles:       10,
				MaxErrorTypes:        10,
				AggregationInterval:  1 * time.Second,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			stats := NewBoundedStats(tc.config)

			b.Run("RecordProcessing", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						stats.RecordProcessing(
							fmt.Sprintf("/test/file_%d.json", rand.Int()),
							int64(rand.Intn(10000)),
							time.Duration(rand.Intn(100))*time.Millisecond,
						)
					}
				})
			})

			b.Run("GetSnapshot", func(b *testing.B) {
				// Pre-populate with data
				for i := 0; i < 1000; i++ {
					stats.RecordProcessing(
						fmt.Sprintf("/test/file_%d.json", i),
						int64(rand.Intn(10000)),
						time.Duration(rand.Intn(100))*time.Millisecond,
					)
				}

				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = stats.GetSnapshot()
					}
				})
			})
		})
	}
}

// BenchmarkMemoryUsage compares memory usage between implementations
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("BoundedStats", func(b *testing.B) {
		stats := NewBoundedStats(DefaultBoundedStatsConfig())

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Simulate heavy usage
		for i := 0; i < 10000; i++ {
			stats.RecordProcessing(
				fmt.Sprintf("/very/long/path/to/file/that/could/consume/memory_%d.json", i),
				int64(rand.Intn(1000000)),
				time.Duration(rand.Intn(1000))*time.Millisecond,
			)
			if i%100 == 0 {
				stats.RecordFailure(
					fmt.Sprintf("/failed/file_%d.json", i),
					fmt.Sprintf("error_type_%d", rand.Intn(20)),
				)
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		memUsed := m2.Alloc - m1.Alloc
		b.ReportMetric(float64(memUsed)/1024/1024, "MB_allocated")
		b.ReportMetric(float64(memUsed)/10000, "bytes/record")
	})
}

// BenchmarkConcurrentLoad tests performance under concurrent load
func BenchmarkConcurrentLoad(b *testing.B) {
	tempDir := b.TempDir()

	// Create test infrastructure
	pool := NewAdaptiveWorkerPool(DefaultAdaptiveConfig())
	ioManager := NewAsyncIOManager(DefaultAsyncIOConfig())
	stats := NewBoundedStats(DefaultBoundedStatsConfig())

	defer pool.Shutdown(5 * time.Second)
	defer ioManager.Shutdown(5 * time.Second)

	// Prepare test data
	testData := make([]byte, 4096)
	rand.Read(testData)

	b.ResetTimer()

	// Simulate realistic concurrent workload
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	itemsPerWorker := b.N / workers

	start := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < itemsPerWorker; i++ {
				// Submit work to pool
				item := WorkItem{
					FilePath: filepath.Join(tempDir, fmt.Sprintf("file_%d_%d.json", workerID, i)),
					Attempt:  0,
					Ctx:      context.Background(),
				}

				if err := pool.Submit(item); err != nil {
					stats.RecordFailure(item.FilePath, "pool_submit_failed")
					continue
				}

				// Async write
				ioManager.WriteFileAsync(item.FilePath, testData, 0644, func(err error) {
					if err != nil {
						stats.RecordFailure(item.FilePath, "write_failed")
					} else {
						stats.RecordProcessing(item.FilePath, int64(len(testData)), 10*time.Millisecond)
					}
				})

				// Record queue depth
				stats.RecordQueueDepth(pool.QueueDepth())
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Get final metrics
	poolMetrics := pool.GetMetrics()
	ioMetrics := ioManager.GetMetrics()
	statsSnapshot := stats.GetSnapshot()

	// Report comprehensive metrics
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "ops/sec")
	b.ReportMetric(poolMetrics.Throughput, "pool_throughput/sec")
	b.ReportMetric(float64(poolMetrics.CurrentWorkers), "final_workers")
	b.ReportMetric(float64(ioMetrics.TotalWrites.Load()), "total_writes")
	b.ReportMetric(statsSnapshot.SuccessRate*100, "success_rate_%")
	b.ReportMetric(float64(statsSnapshot.AvgProcessingTime.Milliseconds()), "avg_latency_ms")

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.Alloc)/1024/1024, "MB_in_use")
}

// BenchmarkBackpressure tests backpressure handling
func BenchmarkBackpressure(b *testing.B) {
	config := &AdaptivePoolConfig{
		MinWorkers:         2,
		MaxWorkers:         4,
		QueueSize:          100, // Small queue to trigger backpressure
		ScaleUpThreshold:   0.8,
		ScaleDownThreshold: 0.2,
		ScaleCooldown:      1 * time.Second,
		MetricsWindow:      50,
	}

	pool := NewAdaptiveWorkerPool(config)
	defer pool.Shutdown(5 * time.Second)

	var rejected atomic.Int64
	var accepted atomic.Int64

	b.ResetTimer()

	// Generate high load to trigger backpressure
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			item := WorkItem{
				FilePath: fmt.Sprintf("/test/file_%d.json", rand.Int()),
				Attempt:  0,
				Ctx:      context.Background(),
			}

			if err := pool.Submit(item); err != nil {
				rejected.Add(1)
			} else {
				accepted.Add(1)
			}
		}
	})

	rejectionRate := float64(rejected.Load()) / float64(b.N) * 100
	b.ReportMetric(rejectionRate, "rejection_rate_%")
	b.ReportMetric(float64(accepted.Load()), "accepted")
	b.ReportMetric(float64(rejected.Load()), "rejected")
}

// BenchmarkLRUCache tests LRU cache performance
func BenchmarkLRUCache(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			cache := NewLRUCache(size)

			// Prepare test data
			keys := make([]string, size*2)
			for i := range keys {
				keys[i] = fmt.Sprintf("key_%d", i)
			}

			b.Run("Put", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						cache.Put(keys[i%len(keys)], i)
						i++
					}
				})
			})

			b.Run("Get", func(b *testing.B) {
				// Pre-populate cache
				for i := 0; i < size; i++ {
					cache.Put(keys[i], i)
				}

				b.ResetTimer()

				var hits atomic.Int64
				var misses atomic.Int64

				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if _, ok := cache.Get(keys[i%len(keys)]); ok {
							hits.Add(1)
						} else {
							misses.Add(1)
						}
						i++
					}
				})

				hitRate := float64(hits.Load()) / float64(hits.Load()+misses.Load()) * 100
				b.ReportMetric(hitRate, "hit_rate_%")
			})
		})
	}
}

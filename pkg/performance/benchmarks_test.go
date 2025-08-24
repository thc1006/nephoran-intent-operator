//go:build go1.24

package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
	"github.com/nephoran/intent-operator/pkg/testutil"
)

// BenchmarkOptimizedHTTPClient tests HTTP client performance
func BenchmarkOptimizedHTTPClient(b *testing.B) {
	testutil.SkipIfShort(b)
	
	// Create mock HTTP server instead of using external service
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test": "data", "benchmark": true}`))
	}))
	defer mockServer.Close()

	config := DefaultHTTPConfig()
	require.NotNil(b, config, "HTTP config should not be nil")
	
	client := NewOptimizedHTTPClient(config)
	require.NotNil(b, client, "HTTP client should not be nil")
	
	ctx, cancel := testutil.ContextWithDeadline(b, 30*time.Second)
	defer cancel()
	defer client.Shutdown(ctx)

	b.Run("ConcurrentRequests", func(b *testing.B) {
		b.SetParallelism(runtime.NumCPU())
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, err := http.NewRequest("GET", mockServer.URL, nil)
				if err != nil {
					b.Fatalf("Failed to create request: %v", err)
				}
				
				requestCtx, requestCancel := context.WithTimeout(ctx, 5*time.Second)
				resp, err := client.DoWithOptimizations(requestCtx, req)
				requestCancel()
				
				if err != nil {
					b.Errorf("HTTP request failed: %v", err)
					continue
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
		})
	})

	b.Run("ConnectionPooling", func(b *testing.B) {
		require.NotNil(b, client.connectionPool, "connection pool should not be nil")
		
		for i := 0; i < b.N; i++ {
			clientFromPool := client.connectionPool.GetClient()
			if clientFromPool == nil {
				b.Error("Failed to get client from pool")
				continue
			}
			client.connectionPool.ReturnClient(clientFromPool)
		}
	})

	b.Run("BufferPooling", func(b *testing.B) {
		require.NotNil(b, client.bufferPool, "buffer pool should not be nil")
		
		for i := 0; i < b.N; i++ {
			buffer := client.bufferPool.GetBuffer(1024)
			require.NotNil(b, buffer, "buffer should not be nil")
			client.bufferPool.PutBuffer(buffer)
		}
	})
}

// BenchmarkMemoryPoolManager tests memory pool performance
func BenchmarkMemoryPoolManager(b *testing.B) {
	testutil.SkipIfShort(b)
	
	config := DefaultMemoryConfig()
	require.NotNil(b, config, "memory config should not be nil")
	
	mpm := NewMemoryPoolManager(config)
	require.NotNil(b, mpm, "memory pool manager should not be nil")
	
	ctx, cancel := testutil.ContextWithDeadline(b, 30*time.Second)
	defer cancel()
	defer mpm.Shutdown()

	b.Run("ObjectPool", func(b *testing.B) {
		// Test with string pool
		stringPool := NewObjectPool[string](
			"test_strings",
			func() string { return "" },
			func(s string) { 
				// Safe reset operation - avoid potential panic on slice bounds
				if len(s) > 0 {
					_ = s[:0] 
				}
			},
		)
		require.NotNil(b, stringPool, "string pool should not be nil")

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case <-ctx.Done():
					return
				default:
					obj := stringPool.Get()
					_ = obj + "test"
					stringPool.Put(obj)
				}
			}
		})
	})

	b.Run("RingBuffer", func(b *testing.B) {
		ringBuffer := NewRingBuffer(1024)
		require.NotNil(b, ringBuffer, "ring buffer should not be nil")
		
		mpm.RegisterRingBuffer("test_ring", ringBuffer)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case <-ctx.Done():
					return
				default:
					data := "test data"
					if !ringBuffer.Push(unsafe.Pointer(&data)) {
						b.Error("Failed to push to ring buffer")
						continue
					}
					if _, ok := ringBuffer.Pop(); !ok {
						b.Error("Failed to pop from ring buffer")
						continue
					}
				}
			}
		})
	})

	b.Run("GCOptimization", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				mpm.ForceGC()
				metrics := mpm.GetMemoryStats()
				require.NotNil(b, metrics, "memory metrics should not be nil")
				if metrics.GCCount == 0 {
					b.Error("GC not triggered")
				}
			}
		}
	})
}

// BenchmarkOptimizedJSONProcessor tests JSON processing performance
func BenchmarkOptimizedJSONProcessor(b *testing.B) {
	testutil.SkipIfShort(b)
	
	config := DefaultJSONConfig()
	require.NotNil(b, config, "JSON config should not be nil")
	
	processor := NewOptimizedJSONProcessor(config)
	require.NotNil(b, processor, "JSON processor should not be nil")
	
	ctx, cancel := testutil.ContextWithDeadline(b, 30*time.Second)
	defer cancel()
	defer processor.Shutdown(ctx)

	testData := struct {
		Name     string            `json:"name"`
		Age      int               `json:"age"`
		Email    string            `json:"email"`
		Metadata map[string]string `json:"metadata"`
		Items    []string          `json:"items"`
	}{
		Name:  "Test User",
		Age:   30,
		Email: "test@example.com",
		Metadata: map[string]string{
			"department": "engineering",
			"level":      "senior",
			"location":   "remote",
		},
		Items: []string{"item1", "item2", "item3", "item4", "item5"},
	}

	// Pre-marshal test data
	testJSON, _ := json.Marshal(testData)

	b.Run("MarshalOptimized", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case <-ctx.Done():
					return
				default:
					marshalCtx, marshalCancel := context.WithTimeout(ctx, 5*time.Second)
					_, err := processor.MarshalOptimized(marshalCtx, testData)
					marshalCancel()
					if err != nil {
						b.Errorf("Marshal failed: %v", err)
					}
				}
			}
		})
	})

	b.Run("UnmarshalOptimized", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case <-ctx.Done():
					return
				default:
					var target struct {
						Name     string            `json:"name"`
						Age      int               `json:"age"`
						Email    string            `json:"email"`
						Metadata map[string]string `json:"metadata"`
						Items    []string          `json:"items"`
					}
					unmarshalCtx, unmarshalCancel := context.WithTimeout(ctx, 5*time.Second)
					err := processor.UnmarshalOptimized(unmarshalCtx, testJSON, &target)
					unmarshalCancel()
					if err != nil {
						b.Errorf("Unmarshal failed: %v", err)
					}
				}
			}
		})
	})

	b.Run("StandardJSONComparison", func(b *testing.B) {
		b.Run("StandardMarshal", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := json.Marshal(testData)
				if err != nil {
					b.Errorf("Standard marshal failed: %v", err)
				}
			}
		})

		b.Run("StandardUnmarshal", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var target struct {
					Name     string            `json:"name"`
					Age      int               `json:"age"`
					Email    string            `json:"email"`
					Metadata map[string]string `json:"metadata"`
					Items    []string          `json:"items"`
				}
				err := json.Unmarshal(testJSON, &target)
				if err != nil {
					b.Errorf("Standard unmarshal failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkEnhancedGoroutinePool tests goroutine pool performance
func BenchmarkEnhancedGoroutinePool(b *testing.B) {
	testutil.SkipIfShort(b)
	
	config := DefaultPoolConfig()
	require.NotNil(b, config, "pool config should not be nil")
	
	pool := NewEnhancedGoroutinePool(config)
	require.NotNil(b, pool, "goroutine pool should not be nil")
	
	ctx, cancel := testutil.ContextWithDeadline(b, 30*time.Second)
	defer cancel()
	defer pool.Shutdown(ctx)

	b.Run("TaskSubmission", func(b *testing.B) {
		if b.N <= 0 {
			b.Skip("invalid benchmark N value")
			return
		}
		
		tasks := make([]*Task, b.N)
		for i := 0; i < b.N; i++ {
			tasks[i] = &Task{
				ID: uint64(i),
				Function: func() error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Microsecond):
						return nil
					}
				},
				Priority: PriorityNormal,
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				// Bounds check
				if i >= len(tasks) {
					b.Fatalf("task index %d out of bounds for tasks slice of length %d", i, len(tasks))
				}
				task := tasks[i]
				require.NotNil(b, task, "task should not be nil")
				
				err := pool.SubmitTask(task)
				if err != nil {
					b.Errorf("Task submission failed: %v", err)
				}
			}
		}
	})

	b.Run("ConcurrentTaskSubmission", func(b *testing.B) {
		b.SetParallelism(runtime.NumCPU())
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				select {
				case <-ctx.Done():
					return
				default:
					task := &Task{
						ID: uint64(time.Now().UnixNano()),
						Function: func() error {
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								runtime.Gosched()
								return nil
							}
						},
						Priority: PriorityNormal,
					}
					err := pool.SubmitTask(task)
					if err != nil {
						b.Errorf("Concurrent task submission failed: %v", err)
					}
				}
			}
		})
	})

	b.Run("WorkStealing", func(b *testing.B) {
		// Fill queues unevenly to test work stealing
		for i := 0; i < 100; i++ {
			task := &Task{
				ID: uint64(i),
				Function: func() error {
					time.Sleep(time.Millisecond)
					return nil
				},
				Priority: PriorityNormal,
			}
			pool.SubmitTask(task)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			metrics := pool.GetMetrics()
			if metrics.StolenTasks == 0 {
				// Work stealing should have occurred
				b.Log("No work stealing detected")
			}
		}
	})
}

// BenchmarkOptimizedCache tests cache performance
func BenchmarkOptimizedCache(b *testing.B) {
	config := DefaultCacheConfig()
	cache := NewOptimizedCache[string, any](config)
	defer cache.Shutdown(context.Background())

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := map[string]any{
			"id":   i,
			"name": fmt.Sprintf("Item %d", i),
			"data": make([]byte, 100),
		}
		cache.Set(key, value)
	}

	b.Run("CacheGet", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key_%d", b.N%1000)
				_, found := cache.Get(key)
				if !found {
					b.Error("Cache miss for existing key")
				}
			}
		})
	})

	b.Run("CacheSet", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("new_key_%d", i)
				value := map[string]any{
					"id":   i,
					"name": fmt.Sprintf("New Item %d", i),
					"data": make([]byte, 100),
				}
				err := cache.Set(key, value)
				if err != nil {
					b.Errorf("Cache set failed: %v", err)
				}
				i++
			}
		})
	})

	b.Run("CacheGetOrSet", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("getorset_key_%d", i%500)
				_, _ = cache.GetOrSet(key, func() any {
					return map[string]any{
						"id":   i,
						"name": fmt.Sprintf("GetOrSet Item %d", i),
						"data": make([]byte, 50),
					}
				})
				i++
			}
		})
	})

	b.Run("EvictionPerformance", func(b *testing.B) {
		smallCache := NewOptimizedCache[string, []byte](&CacheConfig{
			MaxSize:        100, // Small cache to trigger evictions
			EvictionPolicy: EvictionLRU,
			ShardCount:     4,
		})
		defer smallCache.Shutdown(context.Background())

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("evict_key_%d", i)
			value := make([]byte, 1024)
			smallCache.Set(key, value)
		}

		metrics := smallCache.GetMetrics()
		if metrics.Evictions == 0 {
			b.Error("No evictions occurred")
		}
	})
}

// BenchmarkPerformanceIntegration tests integrated performance optimizations
func BenchmarkPerformanceIntegration(b *testing.B) {
	// Create integrated system with all optimizations
	httpConfig := DefaultHTTPConfig()
	httpClient := NewOptimizedHTTPClient(httpConfig)
	defer httpClient.Shutdown(context.Background())

	memConfig := DefaultMemoryConfig()
	memManager := NewMemoryPoolManager(memConfig)
	defer memManager.Shutdown()

	jsonConfig := DefaultJSONConfig()
	jsonProcessor := NewOptimizedJSONProcessor(jsonConfig)
	defer jsonProcessor.Shutdown(context.Background())

	poolConfig := DefaultPoolConfig()
	goroutinePool := NewEnhancedGoroutinePool(poolConfig)
	defer goroutinePool.Shutdown(context.Background())

	cacheConfig := DefaultCacheConfig()
	cache := NewOptimizedCache[string, any](cacheConfig)
	defer cache.Shutdown(context.Background())

	b.Run("IntegratedWorkload", func(b *testing.B) {
		b.SetParallelism(runtime.NumCPU())
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate realistic workload
				task := &Task{
					ID: uint64(time.Now().UnixNano()),
					Function: func() error {
						// JSON processing
						testData := map[string]any{
							"timestamp": time.Now(),
							"data":      "test payload",
							"metrics":   []int{1, 2, 3, 4, 5},
						}

						// Marshal JSON
						jsonData, err := jsonProcessor.MarshalOptimized(context.Background(), testData)
						if err != nil {
							return err
						}

						// Cache result
						cacheKey := fmt.Sprintf("json_%d", time.Now().UnixNano())
						cache.Set(cacheKey, string(jsonData))

						// Memory operations
						stringPool := NewObjectPool[[]byte](
							"temp_pool",
							func() []byte { return make([]byte, 0, 1024) },
							func(b []byte) { b = b[:0] },
						)
						buffer := stringPool.Get()
						buffer = append(buffer, jsonData...)
						stringPool.Put(buffer)

						return nil
					},
					Priority: PriorityNormal,
				}

				err := goroutinePool.SubmitTask(task)
				if err != nil {
					b.Errorf("Integrated workload failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkMemoryEfficiency tests memory usage optimization
func BenchmarkMemoryEfficiency(b *testing.B) {
	b.Run("MemoryAllocation", func(b *testing.B) {
		memConfig := DefaultMemoryConfig()
		memManager := NewMemoryPoolManager(memConfig)
		defer memManager.Shutdown()

		b.ResetTimer()
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		for i := 0; i < b.N; i++ {
			// Use object pools instead of direct allocation
			boolPool := NewObjectPool[[]bool](
				"bool_slice",
				func() []bool { return make([]bool, 100) },
				func(slice []bool) { slice = slice[:0] },
			)

			slice := boolPool.Get()
			for j := 0; j < 100; j++ {
				slice = append(slice, j%2 == 0)
			}
			boolPool.Put(slice)
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
		b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
	})
}

// BenchmarkLatencyOptimization tests latency improvements
func BenchmarkLatencyOptimization(b *testing.B) {
	b.Run("ResponseTime", func(b *testing.B) {
		cache := NewOptimizedCache[string, string](DefaultCacheConfig())
		defer cache.Shutdown(context.Background())

		// Pre-populate cache
		for i := 0; i < 1000; i++ {
			cache.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}

		start := time.Now()
		for i := 0; i < b.N; i++ {
			_, _ = cache.Get(fmt.Sprintf("key_%d", i%1000))
		}
		duration := time.Since(start)

		latencyPerOp := duration / time.Duration(b.N)
		b.ReportMetric(float64(latencyPerOp.Nanoseconds()), "ns/op")

		if latencyPerOp > time.Microsecond {
			b.Errorf("Latency too high: %v (expected < 1μs)", latencyPerOp)
		}
	})
}

// BenchmarkThroughputOptimization tests throughput improvements
func BenchmarkThroughputOptimization(b *testing.B) {
	b.Run("ConcurrentOps", func(b *testing.B) {
		pool := NewEnhancedGoroutinePool(DefaultPoolConfig())
		defer pool.Shutdown(context.Background())

		var completed int64
		start := time.Now()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				task := &Task{
					Function: func() error {
						// Simulate work
						for i := 0; i < 100; i++ {
							_ = i * i
						}
						atomic.AddInt64(&completed, 1)
						return nil
					},
					Priority: PriorityNormal,
				}
				pool.SubmitTask(task)
			}
		})

		// Wait for completion
		for atomic.LoadInt64(&completed) < int64(b.N) {
			time.Sleep(time.Millisecond)
		}

		duration := time.Since(start)
		throughput := float64(b.N) / duration.Seconds()
		b.ReportMetric(throughput, "ops/sec")

		if throughput < 10000 {
			b.Errorf("Throughput too low: %.2f ops/sec (expected > 10k)", throughput)
		}
	})
}

// Benchmark comparison with standard library implementations
func BenchmarkStandardComparison(b *testing.B) {
	testData := struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		ID:    123,
		Name:  "Test User",
		Email: "test@example.com",
	}

	b.Run("JSONMarshal/Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(testData)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("JSONMarshal/Optimized", func(b *testing.B) {
		processor := NewOptimizedJSONProcessor(DefaultJSONConfig())
		defer processor.Shutdown(context.Background())

		for i := 0; i < b.N; i++ {
			_, err := processor.MarshalOptimized(context.Background(), testData)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("MapAccess/Standard", func(b *testing.B) {
		m := make(map[string]any)
		for i := 0; i < 1000; i++ {
			m[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = m[fmt.Sprintf("key_%d", i%1000)]
		}
	})

	b.Run("MapAccess/OptimizedCache", func(b *testing.B) {
		cache := NewOptimizedCache[string, string](DefaultCacheConfig())
		defer cache.Shutdown(context.Background())

		for i := 0; i < 1000; i++ {
			cache.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.Get(fmt.Sprintf("key_%d", i%1000))
		}
	})
}

// Helper function to verify performance improvements
func verifyPerformanceGains(b *testing.B, standardTime, optimizedTime time.Duration, expectedImprovement float64) {
	if standardTime == 0 {
		b.Error("Standard time cannot be zero")
		return
	}

	improvementRatio := float64(standardTime-optimizedTime) / float64(standardTime)
	if improvementRatio < expectedImprovement {
		b.Errorf("Performance improvement insufficient: %.2f%% (expected >= %.2f%%)",
			improvementRatio*100, expectedImprovement*100)
	} else {
		b.Logf("Performance improvement: %.2f%%", improvementRatio*100)
	}
}

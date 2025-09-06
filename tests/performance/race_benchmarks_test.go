// Package performance_tests provides comprehensive race condition benchmarks
// for all critical system components using Go 1.24+ features
package performance_tests

import (
	"fmt"
	mathrand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

// BenchmarkControllerRaceConditions benchmarks controller operations under race conditions
func BenchmarkControllerRaceConditions(b *testing.B) {
	scenarios := []struct {
		name       string
		goroutines int
		workload   func(b *testing.B, id int)
	}{
		{
			name:       "ReconcileLoop",
			goroutines: runtime.NumCPU() * 2,
			workload:   benchmarkReconcileLoop,
		},
		{
			name:       "CacheOperations",
			goroutines: runtime.NumCPU() * 4,
			workload:   benchmarkCacheOperations,
		},
		{
			name:       "QueueProcessing",
			goroutines: runtime.NumCPU(),
			workload:   benchmarkQueueProcessing,
		},
		{
			name:       "LeaderElection",
			goroutines: 10,
			workload:   benchmarkLeaderElection,
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.SetParallelism(scenario.goroutines)
			b.RunParallel(func(pb *testing.PB) {
				id := mathrand.Intn(scenario.goroutines)
				for pb.Next() {
					scenario.workload(b, id)
				}
			})
		})
	}
}

func benchmarkReconcileLoop(b *testing.B, id int) {
	// Simulate controller reconciliation
	cache := &sync.Map{}
	key := fmt.Sprintf("resource-%d", id%100)

	// Write to cache
	cache.Store(key, time.Now())

	// Read from cache
	if val, ok := cache.Load(key); ok {
		_ = val.(time.Time)
	}

	// Update metrics
	reconcileCounter.Add(1)
}

func benchmarkCacheOperations(b *testing.B, id int) {
	key := fmt.Sprintf("cache-key-%d", id%1000)
	value := fmt.Sprintf("value-%d", id)

	// Concurrent cache operations
	globalCache.Store(key, value)

	if val, ok := globalCache.Load(key); ok {
		_ = val.(string)
		cacheHits.Add(1)
	} else {
		cacheMisses.Add(1)
	}

	// Occasionally delete
	if id%10 == 0 {
		globalCache.Delete(key)
	}
}

func benchmarkQueueProcessing(b *testing.B, id int) {
	item := fmt.Sprintf("work-item-%d", id)

	select {
	case workQueue <- item:
		enqueued.Add(1)
	default:
		dropped.Add(1)
	}

	select {
	case <-workQueue:
		processed.Add(1)
	default:
		// Queue empty
	}
}

func benchmarkLeaderElection(b *testing.B, id int) {
	candidate := fmt.Sprintf("node-%d", id)

	leaderMu.Lock()
	if currentLeader == "" {
		currentLeader = candidate
		elections.Add(1)
	} else if currentLeader == candidate {
		renewals.Add(1)
	}
	leaderMu.Unlock()
}

// BenchmarkLLMRaceConditions benchmarks LLM operations under concurrent load
func BenchmarkLLMRaceConditions(b *testing.B) {
	b.Run("TokenManagement", func(b *testing.B) {
		tokens := &atomic.Int64{}
		tokens.Store(10000)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				needed := int64(mathrand.Intn(100) + 1)

				// Try to acquire tokens
				for {
					current := tokens.Load()
					if current < needed {
						break
					}
					if tokens.CompareAndSwap(current, current-needed) {
						// Use tokens
						time.Sleep(time.Nanosecond)
						// Release tokens
						tokens.Add(needed)
						break
					}
				}
			}
		})
	})

	b.Run("CircuitBreaker", func(b *testing.B) {
		state := &atomic.Int32{} // 0=closed, 1=open, 2=half-open
		failures := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				currentState := state.Load()

				switch currentState {
				case 0: // Closed
					if mathrand.Float32() < 0.1 { // 10% failure rate
						if failures.Add(1) >= 5 {
							state.CompareAndSwap(0, 1)
						}
					}
				case 1: // Open
					// Check timeout and transition to half-open
					state.CompareAndSwap(1, 2)
				case 2: // Half-open
					if mathrand.Float32() < 0.5 {
						state.CompareAndSwap(2, 0)
						failures.Store(0)
					} else {
						state.CompareAndSwap(2, 1)
					}
				}
			}
		})
	})

	b.Run("BatchProcessing", func(b *testing.B) {
		batch := make([]string, 0, 100)
		batchMu := &sync.Mutex{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				item := fmt.Sprintf("item-%d", mathrand.Int())

				batchMu.Lock()
				batch = append(batch, item)
				if len(batch) >= 100 {
					// Process batch
					batch = batch[:0]
					batchesProcessed.Add(1)
				}
				batchMu.Unlock()
			}
		})
	})
}

// BenchmarkSecurityRaceConditions benchmarks security operations
func BenchmarkSecurityRaceConditions(b *testing.B) {
	b.Run("CertificateRotation", func(b *testing.B) {
		certs := &sync.Map{}
		rotations := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				certID := fmt.Sprintf("cert-%d", mathrand.Intn(100))

				// Check if rotation needed
				if val, ok := certs.Load(certID); ok {
					if cert := val.(*mockCert); time.Since(cert.created) > time.Hour {
						// Rotate
						certs.Store(certID, &mockCert{created: time.Now()})
						rotations.Add(1)
					}
				} else {
					// Create new cert
					certs.Store(certID, &mockCert{created: time.Now()})
				}
			}
		})

		b.Logf("Certificate rotations: %d", rotations.Load())
	})

	b.Run("RBACAuthorization", func(b *testing.B) {
		permissions := &sync.Map{}
		authorized := &atomic.Int64{}
		denied := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				user := fmt.Sprintf("user-%d", mathrand.Intn(10))
				resource := fmt.Sprintf("resource-%d", mathrand.Intn(100))
				action := []string{"read", "write", "delete"}[mathrand.Intn(3)]

				key := fmt.Sprintf("%s:%s:%s", user, resource, action)

				// Check cached permission
				if val, ok := permissions.Load(key); ok {
					if val.(bool) {
						authorized.Add(1)
					} else {
						denied.Add(1)
					}
				} else {
					// Simulate permission check
					allowed := mathrand.Float32() > 0.3
					permissions.Store(key, allowed)
					if allowed {
						authorized.Add(1)
					} else {
						denied.Add(1)
					}
				}
			}
		})

		b.Logf("Authorized: %d, Denied: %d", authorized.Load(), denied.Load())
	})

	b.Run("SecretManagement", func(b *testing.B) {
		secrets := &sync.Map{}
		version := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				secretID := fmt.Sprintf("secret-%d", mathrand.Intn(50))

				if mathrand.Float32() < 0.1 { // 10% rotation rate
					// Rotate secret
					secrets.Store(secretID, fmt.Sprintf("value-v%d", version.Add(1)))
				} else {
					// Read secret
					if val, ok := secrets.Load(secretID); ok {
						_ = val.(string)
					} else {
						secrets.Store(secretID, fmt.Sprintf("value-v%d", version.Load()))
					}
				}
			}
		})
	})
}

// BenchmarkMonitoringRaceConditions benchmarks monitoring operations
func BenchmarkMonitoringRaceConditions(b *testing.B) {
	b.Run("MetricsCollection", func(b *testing.B) {
		metrics := &sync.Map{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				metricName := fmt.Sprintf("metric-%d", mathrand.Intn(100))
				value := mathrand.Float64() * 100

				// Update metric
				if current, ok := metrics.Load(metricName); ok {
					m := current.(*metricData)
					atomic.AddInt64(&m.count, 1)
					// Atomic float64 update
					for {
						old := atomic.LoadUint64(&m.sumBits)
						new := floatToUint64(uint64ToFloat(old) + value)
						if atomic.CompareAndSwapUint64(&m.sumBits, old, new) {
							break
						}
					}
				} else {
					metrics.Store(metricName, &metricData{
						count:   1,
						sumBits: floatToUint64(value),
					})
				}
			}
		})
	})

	b.Run("AlertProcessing", func(b *testing.B) {
		alerts := make(chan *mockAlert, 1000)
		active := &sync.Map{}
		processed := &atomic.Int64{}

		// Alert processor
		go func() {
			for alert := range alerts {
				if _, loaded := active.LoadOrStore(alert.id, alert); loaded {
					// Duplicate alert
				} else {
					processed.Add(1)
				}
			}
		}()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				alert := &mockAlert{
					id:       fmt.Sprintf("alert-%d", mathrand.Intn(50)),
					severity: mathrand.Intn(3),
				}

				select {
				case alerts <- alert:
					// Sent
				default:
					// Queue full
				}
			}
		})

		close(alerts)
		b.Logf("Alerts processed: %d", processed.Load())
	})

	b.Run("DistributedTracing", func(b *testing.B) {
		traces := &sync.Map{}
		spanCounter := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				traceID := fmt.Sprintf("trace-%d", mathrand.Intn(100))
				spanID := spanCounter.Add(1)

				// Add span to trace
				if val, ok := traces.Load(traceID); ok {
					spans := val.([]int64)
					traces.Store(traceID, append(spans, spanID))
				} else {
					traces.Store(traceID, []int64{spanID})
				}
			}
		})

		b.Logf("Total spans: %d", spanCounter.Load())
	})
}

// BenchmarkHighContentionScenarios tests extreme contention scenarios
func BenchmarkHighContentionScenarios(b *testing.B) {
	b.Run("SingleResourceContention", func(b *testing.B) {
		resource := &atomic.Int64{}

		b.SetParallelism(runtime.NumCPU() * 8)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// All goroutines compete for same resource
				for {
					old := resource.Load()
					if resource.CompareAndSwap(old, old+1) {
						break
					}
					runtime.Gosched()
				}
			}
		})

		b.Logf("Final value: %d", resource.Load())
	})

	b.Run("HotPathContention", func(b *testing.B) {
		hotPath := &sync.Mutex{}
		counter := int64(0)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				hotPath.Lock()
				counter++
				hotPath.Unlock()
			}
		})

		b.Logf("Counter: %d", counter)
	})

	b.Run("ChannelStress", func(b *testing.B) {
		ch := make(chan int, 100)
		sent := &atomic.Int64{}
		received := &atomic.Int64{}

		// Multiple producers
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				for j := 0; j < b.N/runtime.NumCPU(); j++ {
					select {
					case ch <- j:
						sent.Add(1)
					default:
					}
				}
			}()
		}

		// Multiple consumers
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				for {
					select {
					case <-ch:
						received.Add(1)
					default:
						return
					}
				}
			}()
		}

		time.Sleep(100 * time.Millisecond)
		b.Logf("Sent: %d, Received: %d", sent.Load(), received.Load())
	})
}

// BenchmarkMemoryOrdering tests memory ordering and barriers
func BenchmarkMemoryOrdering(b *testing.B) {
	b.Run("StoreLoadOrdering", func(b *testing.B) {
		var x, y atomic.Int64
		var r1, r2 int64
		violations := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Reset
				x.Store(0)
				y.Store(0)

				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					x.Store(1)
					r1 = y.Load()
				}()

				go func() {
					defer wg.Done()
					y.Store(1)
					r2 = x.Load()
				}()

				wg.Wait()

				// Check for memory ordering violation
				if r1 == 0 && r2 == 0 {
					violations.Add(1)
				}
			}
		})

		b.Logf("Memory ordering violations: %d", violations.Load())
	})

	b.Run("AcquireRelease", func(b *testing.B) {
		var flag atomic.Bool
		var data int64
		errors := &atomic.Int64{}

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				if counter%2 == 0 {
					// Writer
					data = int64(mathrand.Int())
					flag.Store(true) // Release
				} else {
					// Reader
					if flag.Load() { // Acquire
						if data == 0 {
							errors.Add(1)
						}
					}
				}
				counter++
			}
		})

		b.Logf("Errors: %d", errors.Load())
	})
}

// Global variables for benchmarks
var (
	globalCache      = &sync.Map{}
	workQueue        = make(chan string, 1000)
	leaderMu         sync.Mutex
	currentLeader    string
	reconcileCounter = &atomic.Int64{}
	cacheHits        = &atomic.Int64{}
	cacheMisses      = &atomic.Int64{}
	enqueued         = &atomic.Int64{}
	dropped          = &atomic.Int64{}
	processed        = &atomic.Int64{}
	elections        = &atomic.Int64{}
	renewals         = &atomic.Int64{}
	batchesProcessed = &atomic.Int64{}
)

// Helper types
type mockCert struct {
	created time.Time
}

type mockAlert struct {
	id       string
	severity int
}

type metricData struct {
	count   int64
	sumBits uint64
}

// Helper functions
func floatToUint64(f float64) uint64 {
	return *(*uint64)(unsafe.Pointer(&f))
}

func uint64ToFloat(u uint64) float64 {
	return *(*float64)(unsafe.Pointer(&u))
}

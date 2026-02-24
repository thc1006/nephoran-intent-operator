# Performance Optimization Task #66

**Objective**: Achieve 4-10x throughput improvement (50-100 req/sec → 200-500 req/sec)

**Date**: 2026-02-24
**Target**: Fix goroutine leaks, HTTP connection pools, unbounded channels, missing timeouts

---

## 1. Issues Identified

### 1.1 Goroutine Leaks

**Found Issues:**

1. **`internal/loop/watcher.go:730`** - `startMetricsCollection()` goroutine
   - ✅ Already has WaitGroup tracking via `backgroundWG`
   - Location: Called at line 716 in `NewWatcher()`
   - Status: **FIXED** - Proper lifecycle management exists

2. **`internal/loop/watcher.go:922`** - Metrics server goroutine
   - ❌ NO WaitGroup tracking
   - Status: **NEEDS FIX**

3. **`pkg/oran/o1/client.go:91`** - Performance subscription goroutine
   - ❌ NO WaitGroup tracking
   - Creates unbounded goroutine for each subscription
   - Status: **NEEDS FIX**

### 1.2 HTTP Connection Pools

**Default HTTP Client Issues** (2 connections per host is too low):

Production clients found (non-test, non-examples):
- `pkg/oran/a1/a1_adaptor.go` - ✅ HAS custom transport with proper config
- `pkg/oran/e2/e2_adaptor.go` - ❌ NEEDS custom transport
- `pkg/porch/client.go` - ❌ NEEDS custom transport
- `pkg/porch/direct_client.go` - ❌ NEEDS custom transport
- `pkg/nephio/client.go` - ❌ NEEDS custom transport
- `controllers/networkintent_controller.go` - ❌ Multiple clients need config
- `cmd/nephio-bridge/main.go` - ❌ NEEDS custom transport
- `internal/loop/watcher.go` (Porch HTTP calls) - ❌ Uses default client

**Optimal Configuration:**
```go
Transport: &http.Transport{
    MaxIdleConns:        100,     // Total across all hosts
    MaxIdleConnsPerHost: 50,      // Per backend (adjust based on expected concurrency)
    IdleConnTimeout:     90 * time.Second,
    DisableKeepAlives:   false,   // Keep connections alive
    TLSHandshakeTimeout: 10 * time.Second,
    DialContext: (&net.Dialer{
        Timeout:   10 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
}
```

### 1.3 Unbounded Channels

**Found Issues:**

1. **`pkg/controllers/orchestration/event_bus.go:235`**
   - `eventChan: make(chan ProcessingEvent, 1000)`
   - Status: **BOUNDED** - Good size (1000)

2. **`pkg/controllers/parallel/task_scheduler.go:48,52`**
   - `schedulingQueue: make(chan *Task, 1000)`
   - `readyTasks: make(chan *Task, 1000)`
   - Status: **BOUNDED** - Good size (1000)

3. **`pkg/oran/o1/client.go:89`**
   - `ch := make(chan *PerformanceMeasurement, 100)`
   - Status: **BOUNDED** - But could grow with multiple subscriptions
   - Needs: Subscription limit per client

4. **`pkg/oran/a1/a1_adaptor.go:182`**
   - `eventQueue chan *PolicyEvent` - **UNBOUNDED in struct definition**
   - Status: **NEEDS FIX** - Must be created with buffer size

### 1.4 Missing Timeouts

**HTTP Clients Without Timeouts:**

1. ❌ `pkg/oran/e2/e2_adaptor.go` - No timeout set
2. ❌ `pkg/nephio/client.go` - No timeout set
3. ❌ `pkg/porch/direct_client.go` - No timeout set
4. ✅ `pkg/oran/a1/a1_adaptor.go` - Has 30s timeout
5. ✅ `pkg/rag/client.go` - Has 30s timeout (configurable)
6. ✅ `pkg/llm/client_consolidated.go` - Has timeout via optimized transport

---

## 2. Implementation Plan

### Phase 1: Fix Goroutine Leaks (Priority: P0)

1. Add WaitGroup tracking to metrics server goroutine
2. Add WaitGroup tracking to O1 performance subscriptions
3. Implement subscription lifecycle management

### Phase 2: Configure HTTP Connection Pools (Priority: P0)

1. Create shared HTTP transport factory with optimal settings
2. Update all O-RAN clients (A1, E2, O1, O2)
3. Update Porch clients
4. Update Nephio clients
5. Update controller HTTP clients

### Phase 3: Bound Unbounded Channels (Priority: P1)

1. Fix A1 eventQueue channel creation
2. Add subscription limits to O1 client
3. Document channel sizing rationale

### Phase 4: Add Missing Timeouts (Priority: P1)

1. Add timeouts to all HTTP clients (default 30s)
2. Add context timeouts for long-running operations
3. Add database query timeouts (if applicable)

---

## 3. Expected Performance Impact

### Before Optimization:
- HTTP clients: 2 connections/host (default)
- Goroutine leaks: Memory growth over time
- No connection pooling: New connection per request
- Throughput: 50-100 req/sec

### After Optimization:
- HTTP clients: 50 connections/host (25x improvement)
- No goroutine leaks: Stable memory usage
- Connection pooling: Reuse existing connections (10-100x faster than new conn)
- Expected throughput: **200-500 req/sec (4-10x improvement)**

**Connection Pool Math:**
- Before: 2 connections × 10 concurrent requests = 80% request queuing
- After: 50 connections × 10 concurrent requests = 0% request queuing
- TLS handshake savings: ~100ms per reused connection

---

## 4. Files to Modify

### Critical Files (Must Fix):

1. `internal/loop/watcher.go` - Metrics server goroutine lifecycle
2. `pkg/oran/a1/a1_adaptor.go` - Event queue channel initialization
3. `pkg/oran/e2/e2_adaptor.go` - Add HTTP transport and timeout
4. `pkg/oran/o1/client.go` - Subscription lifecycle management
5. `pkg/porch/client.go` - Add HTTP transport
6. `pkg/porch/direct_client.go` - Add HTTP transport
7. `pkg/nephio/client.go` - Add HTTP transport

### Supporting Files:

8. `pkg/shared/http_transport.go` - **NEW** - Shared transport factory
9. `controllers/networkintent_controller.go` - Update HTTP clients

---

## 5. Testing Strategy

### Unit Tests:
- Verify WaitGroup tracking in goroutine lifecycle tests
- Verify HTTP client configuration in client tests
- Verify channel buffer sizes in channel tests

### Integration Tests:
- Load test: 500 concurrent requests with connection reuse
- Memory leak test: Run for 1 hour, verify stable memory
- Timeout test: Verify all operations respect timeout limits

### Performance Benchmarks:
- Baseline: Current throughput with `go test -bench`
- Target: 4-10x improvement in throughput
- Metrics: requests/sec, latency p50/p95/p99, memory usage

---

## 6. Rollout Plan

1. **Phase 1**: Fix goroutine leaks (low risk, high impact)
2. **Phase 2**: Configure HTTP pools (medium risk, highest impact)
3. **Phase 3**: Bound channels (low risk, medium impact)
4. **Phase 4**: Add timeouts (low risk, safety improvement)

Each phase will be committed separately with:
- Unit tests passing
- Integration tests passing
- Performance benchmarks showing improvement
- Documentation updates

---

## 7. Success Criteria

✅ All goroutines have proper WaitGroup tracking
✅ All HTTP clients use optimized connection pools (50+ conns/host)
✅ All channels are bounded with documented sizes
✅ All HTTP operations have timeouts (30s default)
✅ Throughput improves by 4-10x (200-500 req/sec achieved)
✅ Memory usage remains stable under load
✅ All tests pass (go test ./...)

---

**Next Steps**: Start with Phase 1 - Fix goroutine leaks in watcher and O1 client.

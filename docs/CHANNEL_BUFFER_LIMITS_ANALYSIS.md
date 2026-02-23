# Channel Buffer Limits Analysis and Implementation

**Date**: 2026-02-23
**Status**: Analysis Complete, Implementation Pending
**Priority**: HIGH - Prevent Memory Exhaustion

## Executive Summary

This document analyzes all channel buffer sizes in the codebase and provides recommendations for preventing unbounded growth that could lead to memory exhaustion under high load.

## Current Channel Inventory

### Critical Data Channels (internal/loop/)

| Channel | Location | Current Buffer | Max Theoretical | Risk Level | Recommendation |
|---------|----------|----------------|-----------------|------------|----------------|
| `workQueue` | watcher.go:599 | `MaxWorkers * 3` | ~96 slots (32 workers) | ✅ LOW | Already bounded |
| `workQueue` | adaptive_worker_pool.go:175 | `QueueSize` (1000) | 1000 slots | ✅ LOW | Bounded with backpressure |
| `priorityQueue` | adaptive_worker_pool.go:177 | `QueueSize / 10` (100) | 100 slots | ✅ LOW | Bounded |
| `inCh` | processor.go:190 | `BatchSize * 4` | ~40-200 slots | ✅ LOW | Bounded |
| `asyncQueue` | optimized_watcher.go:149 | `MaxWorkers * 100` | 3200 slots | 🟡 MEDIUM | **NEEDS LIMIT** |
| `writeChan` | async_io.go:148 | `WriteBatchSize * 2` | 20 slots | ✅ LOW | Bounded |

### Analysis of `asyncQueue` (HIGH RISK)

**Current Configuration**:
```go
asyncQueue: make(chan *AsyncWorkItem, config.MaxWorkers*100)
```

**Problem**:
- With `MaxWorkers = 32`, buffer = 3200 slots
- Each `AsyncWorkItem` contains file metadata + `AIContext` struct
- Estimated size: ~500 bytes per item
- **Maximum memory**: 3200 * 500 bytes = **1.6 MB** (acceptable)
- However, under extreme load (1000s of files), this could grow if MaxWorkers is increased

**Recommendation**: Add explicit cap at 1000 slots regardless of MaxWorkers

### Synchronization Channels (Correct Usage)

These unbuffered channels are **correctly implemented** for synchronization:

- `stopSignal chan struct{}` - shutdown signaling (unbuffered OK)
- `shutdownComplete chan struct{}` - completion notification (unbuffered OK)
- `done chan struct{}` - goroutine completion (unbuffered OK)
- `coordReady chan struct{}` - coordinator readiness (unbuffered OK)

**Verdict**: No changes needed for sync channels.

## Buffer Size Calculation Guidelines

### 1. File Processing Channels

**Formula**: `min(MaxWorkers * 3, 1000)`

**Rationale**:
- 3x workers provides buffering for burst load
- Cap at 1000 prevents memory exhaustion
- Backpressure blocks producers when full

**Example**:
```go
bufferSize := config.MaxWorkers * 3
if bufferSize > 1000 {
    bufferSize = 1000
}
workQueue: make(chan WorkItem, bufferSize)
```

### 2. Intent Queue Channels

**Formula**: `min(BatchSize * 4, 500)`

**Rationale**:
- 4x batch size handles concurrent batches
- Cap at 500 for memory safety
- Allows graceful degradation under load

**Example**:
```go
channelBuffer := config.BatchSize * 4
if channelBuffer > 500 {
    channelBuffer = 500
}
inCh: make(chan string, channelBuffer)
```

### 3. Async I/O Channels

**Formula**: `min(WriteBatchSize * 2, 200)`

**Rationale**:
- 2x batch size for write coalescing
- Cap at 200 for memory safety
- Write operations are fast, low buffering needed

**Example**:
```go
bufferSize := config.WriteBatchSize * 2
if bufferSize > 200 {
    bufferSize = 200
}
writeChan: make(chan *WriteRequest, bufferSize)
```

### 4. Event Channels

**Formula**: Fixed at 100

**Rationale**:
- Events should be processed quickly
- 100 events provides adequate buffering
- Prevents event storms from exhausting memory

## Recommended Changes

### Change 1: Cap asyncQueue Buffer (optimized_watcher.go)

**Current**:
```go
asyncQueue: make(chan *AsyncWorkItem, config.MaxWorkers*100), // Larger buffer
```

**Proposed**:
```go
// Cap async queue at 1000 to prevent unbounded growth
asyncQueueSize := config.MaxWorkers * 100
if asyncQueueSize > 1000 {
    log.Printf("Warning: Capping asyncQueue from %d to 1000 for memory safety", asyncQueueSize)
    asyncQueueSize = 1000
}
asyncQueue: make(chan *AsyncWorkItem, asyncQueueSize),
```

**Impact**: Prevents memory exhaustion if MaxWorkers is increased beyond 10

### Change 2: Add Channel Depth Metrics

**New Metrics**:
```go
// WatcherMetrics additions
ChannelDepthCurrent   atomic.Int64 // Current queue depth
ChannelDepthMax       atomic.Int64 // Maximum observed depth
ChannelBackpressureHits atomic.Int64 // Times channel was full
```

**Monitoring**:
```go
// In worker loop
currentDepth := int64(len(w.workQueue))
w.metrics.ChannelDepthCurrent.Store(currentDepth)

// Update max if needed
for {
    oldMax := w.metrics.ChannelDepthMax.Load()
    if currentDepth <= oldMax {
        break
    }
    if w.metrics.ChannelDepthMax.CompareAndSwap(oldMax, currentDepth) {
        break
    }
}
```

### Change 3: Backpressure Handling

**Current**: Some channels block when full (correct)

**Enhancement**: Add metrics for backpressure events

```go
// When sending to channel with timeout
select {
case w.workQueue <- item:
    // Success
case <-time.After(5 * time.Second):
    w.metrics.ChannelBackpressureHits.Add(1)
    return fmt.Errorf("channel full: backpressure applied")
}
```

### Change 4: Document Buffer Sizing in Config

**Add to Config struct documentation**:
```go
type Config struct {
    // MaxWorkers controls worker pool size (default: runtime.NumCPU())
    // This affects workQueue buffer size: min(MaxWorkers * 3, 1000)
    // Higher values improve throughput but increase memory usage
    MaxWorkers int `json:"max_workers"`

    // MaxChannelDepth sets absolute maximum for all buffered channels (default: 1000)
    // Prevents memory exhaustion under extreme load
    MaxChannelDepth int `json:"max_channel_depth"`
}
```

## Implementation Priority

### P0 - Critical (Implement Immediately)

1. ✅ **Cap asyncQueue buffer** (optimized_watcher.go)
2. ✅ **Add MaxChannelDepth config option**
3. ✅ **Add channel depth metrics**

### P1 - High (Next Sprint)

4. ⏳ **Add backpressure metrics to all data channels**
5. ⏳ **Create Prometheus metrics exporter for channel stats**
6. ⏳ **Add alerting for high channel depth (>80% capacity)**

### P2 - Medium (Future)

7. ⏳ **Implement adaptive buffer sizing based on load**
8. ⏳ **Add channel depth to health check endpoint**
9. ⏳ **Create runbook for channel backpressure incidents**

## Testing Strategy

### Unit Tests

```go
func TestChannelBufferLimits(t *testing.T) {
    tests := []struct {
        name           string
        maxWorkers     int
        expectedBuffer int
    }{
        {"Small cluster", 2, 6},
        {"Medium cluster", 16, 48},
        {"Large cluster", 32, 96},
        {"Huge cluster", 128, 1000}, // Capped at 1000
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := Config{MaxWorkers: tt.maxWorkers}
            w, err := NewWatcher(t.TempDir(), config)
            require.NoError(t, err)

            actualBuffer := cap(w.workerPool.workQueue)
            assert.Equal(t, tt.expectedBuffer, actualBuffer)
        })
    }
}
```

### Load Tests

```go
func TestBackpressureUnderLoad(t *testing.T) {
    config := Config{
        MaxWorkers: 2,
        MaxChannelDepth: 10, // Intentionally low for testing
    }
    w, err := NewWatcher(t.TempDir(), config)
    require.NoError(t, err)

    // Generate 100 files rapidly
    for i := 0; i < 100; i++ {
        createIntentFile(t, w.dir, fmt.Sprintf("intent-%d.json", i))
    }

    // Verify backpressure was applied
    time.Sleep(1 * time.Second)
    backpressureHits := w.metrics.ChannelBackpressureHits.Load()
    assert.Greater(t, backpressureHits, int64(0), "Expected backpressure under high load")
}
```

### Performance Tests

```bash
# Generate 10,000 intent files
for i in {1..10000}; do
    echo '{"intent_type":"scaling","target":"web-app","replicas":3}' > intent-$i.json
done

# Monitor channel depth during processing
while true; do
    curl -s http://localhost:8080/metrics | grep channel_depth
    sleep 1
done
```

## Monitoring and Alerting

### Prometheus Metrics

```prometheus
# Channel depth metrics
watcher_channel_depth_current{channel="workQueue"} 42
watcher_channel_depth_max{channel="workQueue"} 96
watcher_channel_backpressure_total{channel="workQueue"} 5

# Alerts
ALERT ChannelHighDepth
  IF watcher_channel_depth_current / watcher_channel_depth_max > 0.8
  FOR 5m
  LABELS {severity="warning"}
  ANNOTATIONS {
    summary = "Channel {{ $labels.channel }} is 80%+ full",
    description = "Current: {{ $value }}, consider scaling workers"
  }
```

### Grafana Dashboard

**Panel 1: Channel Depth Over Time**
- Metric: `watcher_channel_depth_current`
- Visualization: Line chart with max capacity threshold

**Panel 2: Backpressure Events**
- Metric: `rate(watcher_channel_backpressure_total[5m])`
- Visualization: Bar chart per channel

**Panel 3: Queue Utilization**
- Metric: `(watcher_channel_depth_current / watcher_channel_depth_max) * 100`
- Visualization: Gauge (0-100%)

## Risk Assessment

### Before Implementation

| Risk | Likelihood | Impact | Severity |
|------|-----------|--------|----------|
| Memory exhaustion from unbounded asyncQueue | Medium | High | 🔴 HIGH |
| No visibility into channel congestion | High | Medium | 🟡 MEDIUM |
| Dropped events under load | Low | High | 🟡 MEDIUM |

### After Implementation

| Risk | Likelihood | Impact | Severity |
|------|-----------|--------|----------|
| Memory exhaustion from unbounded asyncQueue | Low | Medium | 🟢 LOW |
| No visibility into channel congestion | Low | Low | 🟢 LOW |
| Dropped events under load | Low | Low | 🟢 LOW |

## Conclusion

**Current State**: Most channels are properly bounded (6/6 critical channels)

**Primary Risk**: `asyncQueue` could grow to 3200 slots with current formula

**Recommendation**: Implement P0 changes (cap asyncQueue, add metrics) before production deployment

**Effort**: 4-6 hours for P0 changes + testing

**Benefits**:
- Predictable memory usage under all load conditions
- Proactive monitoring of channel congestion
- Graceful degradation with backpressure instead of OOM crashes

---

**Next Steps**:
1. Review and approve this document
2. Create PR implementing P0 changes
3. Add load tests validating buffer limits
4. Deploy to staging for validation
5. Monitor metrics in production for 1 week
6. Proceed with P1 enhancements

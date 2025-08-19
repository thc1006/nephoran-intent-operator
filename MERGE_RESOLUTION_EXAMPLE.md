# Merge Resolution: Unified Watcher Implementation

This document demonstrates how the unified watcher resolves the merge conflict between the Config-based approach (HEAD) and IntentProcessor approach (origin/integrate/mvp).

## Overview

The resolved implementation provides **dual constructor patterns** that support both approaches:

1. **Legacy Config-based approach** - Original HEAD implementation with enhanced security, metrics, and worker pools
2. **New IntentProcessor approach** - Clean abstraction from origin/integrate/mvp for batching and validation
3. **Hybrid processing** - Automatically selects the right processing pattern based on which constructor is used

## Constructor Patterns

### 1. Legacy Config-based Approach
```go
// Backward compatible - uses Config struct with all security enhancements
watcher, err := loop.NewWatcher(dir, loop.Config{
    PorchPath:    "/path/to/porch",
    Mode:        "direct",
    MaxWorkers:  4,
    MetricsPort: 8080,
    // ... other config options
})
```

### 2. IntentProcessor Approach
```go
// New pattern - uses IntentProcessor for clean abstraction
processor := loop.NewProcessor(processorConfig, validator, submitFunc)
watcher, err := loop.NewWatcherWithProcessor(dir, processor)
```

### 3. Low-level Unified Approach
```go
// Unified constructor - supports both patterns
watcher, err := loop.NewWatcherWithConfig(dir, config, processor)
// If processor is nil, uses Config-based legacy approach
// If processor is provided, uses IntentProcessor approach
```

## Key Features Preserved

### From HEAD (Config-based)
- ✅ **Enhanced Security**: Input validation, path traversal prevention, size limits
- ✅ **Advanced Metrics**: Prometheus endpoints, latency tracking, worker utilization
- ✅ **Worker Pool Management**: Concurrent processing with backpressure control
- ✅ **File State Tracking**: Debouncing, duplicate prevention, atomic operations
- ✅ **Comprehensive Validation**: JSON schema validation, intent structure verification

### From origin/integrate/mvp (IntentProcessor)
- ✅ **Clean Abstraction**: Processor interface for validation and submission
- ✅ **Batch Processing**: Configurable batching with flush intervals
- ✅ **Idempotency**: Automatic duplicate detection across restarts
- ✅ **Error Handling**: Dedicated error directory and retry logic

## Migration Path

### Immediate Use (Backward Compatible)
Existing code using `NewWatcher(dir, config)` continues to work unchanged.

### Gradual Migration
```go
// Step 1: Continue using Config approach
watcher, err := loop.NewWatcher(dir, config)

// Step 2: Switch to IntentProcessor when ready
processor := createProcessor() // your processor setup
watcher, err := loop.NewWatcherWithProcessor(dir, processor)
```

## Command Line Usage

The main application now supports both approaches:

### Legacy Mode (Default)
```bash
./conductor-loop --handoff ./handoff --mode direct --porch /path/to/porch
```

### IntentProcessor Mode
```bash
./conductor-loop --use-processor --handoff-dir ./handoff --porch-mode structured --batch-size 10
```

## Hybrid Processing Logic

The unified implementation uses this decision tree:

```go
func (w *Watcher) handleIntentFile(event fsnotify.Event) {
    if w.processor != nil {
        // Use IntentProcessor approach - clean, batched processing
        return w.processor.ProcessFile(event.Name)
    }
    
    // Fall back to enhanced legacy processing
    w.handleIntentFileWithEnhancedDebounce(event.Name, event.Op)
}
```

## Benefits

1. **Smooth Migration**: No breaking changes for existing users
2. **Best of Both Worlds**: Security enhancements + clean abstraction
3. **Flexibility**: Choose the right approach for your use case
4. **Future-Proof**: Easy to deprecate legacy approach when ready

## Testing

Both patterns are fully tested:

```bash
# Test legacy Config approach
go test ./internal/loop -run TestConfig
go test ./internal/loop -run TestNewWatcher

# Test IntentProcessor approach  
go test ./internal/loop -run TestProcessor
go test ./internal/loop -run TestProcessExistingFiles
```

## Summary

This unified implementation successfully resolves the merge conflict by:

- ✅ **Preserving backward compatibility** with the Config-based approach
- ✅ **Supporting the new IntentProcessor pattern** for clean abstraction  
- ✅ **Keeping all security enhancements** from HEAD
- ✅ **Maintaining all metrics and monitoring** capabilities
- ✅ **Enabling smooth migration** between patterns
- ✅ **Providing hybrid processing** that automatically selects the right approach

Both existing and new users can benefit from this unified implementation without breaking changes.
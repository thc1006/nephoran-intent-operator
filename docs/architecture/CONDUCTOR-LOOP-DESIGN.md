# Conductor-Loop Design Document

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Status Contract Documentation](#status-contract-documentation)
4. [Once-Mode Exit Rules](#once-mode-exit-rules)
5. [Windows Timing and Backoff Strategy](#windows-timing-and-backoff-strategy)
6. [Cross-Platform Compatibility](#cross-platform-compatibility)
7. [Concurrency Model](#concurrency-model)
8. [Error Handling and Resilience](#error-handling-and-resilience)
9. [Performance Characteristics](#performance-characteristics)
10. [Security Considerations](#security-considerations)
11. [Operational Guidelines](#operational-guidelines)
12. [Troubleshooting Guide](#troubleshooting-guide)

---

## Executive Summary

The Conductor-Loop system is a robust, cross-platform file watcher and processor designed to handle intent files in a distributed O-RAN/Nephio environment. This design document captures the critical architectural decisions made to ensure reliable operation across Windows, Linux, and macOS platforms, with particular emphasis on handling Windows-specific file system race conditions and graceful shutdown semantics.

### Key Design Principles

- **Reliability First**: Every operation has retry logic with exponential backoff
- **Platform Agnostic**: Abstractions handle OS-specific behaviors transparently
- **Graceful Degradation**: Distinguishes between real failures and shutdown-induced failures
- **Observable**: Comprehensive metrics and structured logging for debugging
- **Idempotent**: Safe to retry any operation without side effects

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Conductor-Loop System                    │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌──────────────┐    ┌──────────────┐  │
│  │   Watcher    │────>│   Processor  │───>│   Executor   │  │
│  │  (fsnotify)  │     │  (Validator) │    │    (Porch)   │  │
│  └──────────────┘     └──────────────┘    └──────────────┘  │
│         │                     │                    │         │
│         v                     v                    v         │
│  ┌──────────────┐     ┌──────────────┐    ┌──────────────┐  │
│  │File Manager  │     │Worker Pool   │    │Status Writer │  │
│  │(Atomic Ops)  │     │(Concurrency) │    │  (Atomic)    │  │
│  └──────────────┘     └──────────────┘    └──────────────┘  │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    Platform Abstraction Layer            │ │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐            │ │
│  │  │ Windows │    │  Linux  │    │  macOS  │            │ │
│  │  │  Fsync  │    │  Fsync  │    │  Fsync  │            │ │
│  │  └─────────┘    └─────────┘    └─────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **File Detection**: Watcher monitors handoff directory for `.json` files
2. **Debouncing**: Events are debounced to handle rapid file changes
3. **Validation**: Files are validated against JSON schema
4. **Processing**: Valid intents are sent to Porch for execution
5. **Status Recording**: Results are atomically written to status files
6. **Cleanup**: Processed files are moved to appropriate directories

---

## Status Contract Documentation

### Status Types and Semantics

The system uses a tri-state status model for tracking file processing:

```go
type ProcessingStatus string

const (
    StatusUnspecified ProcessingStatus = "UNSPECIFIED"  // Initial or unknown state
    StatusSuccess     ProcessingStatus = "SUCCESS"      // Successfully processed
    StatusFailed      ProcessingStatus = "FAILED"       // Processing failed
)
```

### Status Computation Logic

The status is computed based on multiple factors:

1. **Validation Result**
   - Invalid JSON → `FAILED`
   - Schema validation failure → `FAILED`
   - Valid intent → Continue to processing

2. **Processing Result**
   - Porch execution success → `SUCCESS`
   - Porch execution failure → `FAILED`
   - Timeout during processing → `FAILED`
   - Context cancellation → `UNSPECIFIED` (graceful shutdown)

3. **Partial Initialization Scenarios**

When the system encounters partially initialized states:

```go
// Status determination for partial states
func computeStatus(validationPassed bool, processingStarted bool, 
                   processingCompleted bool, error error) ProcessingStatus {
    // Not even validated
    if !validationPassed {
        return StatusFailed
    }
    
    // Validated but processing not started (queue full/shutdown)
    if !processingStarted {
        if isShutdownError(error) {
            return StatusUnspecified  // Don't penalize for shutdown
        }
        return StatusFailed
    }
    
    // Processing started but not completed
    if !processingCompleted {
        if isShutdownError(error) {
            return StatusUnspecified  // Graceful shutdown in progress
        }
        return StatusFailed
    }
    
    // Fully processed
    if error != nil {
        return StatusFailed
    }
    return StatusSuccess
}
```

### Status File Format

Status files follow a versioned naming convention and contain structured metadata:

**Filename Format**: `<intent-filename.json>-YYYYMMDD-HHMMSS.status`

**Content Structure**:
```json
{
    "intent_file": "intent-scale.json",
    "status": "SUCCESS",
    "message": "Processing completed successfully",
    "timestamp": "2025-08-21T10:30:45Z",
    "processed_by": "conductor-loop",
    "mode": "direct",
    "porch_path": "/usr/local/bin/porch",
    "worker_id": "worker-3",
    "processing_duration_ms": 1234,
    "retry_count": 0
}
```

---

## Once-Mode Exit Rules

### Exit Code Semantics

The system uses specific exit codes to communicate different failure scenarios:

| Exit Code | Meaning | Description |
|-----------|---------|-------------|
| 0 | Success | All files processed successfully OR only shutdown-induced failures |
| 1 | General Error | Watcher initialization or runtime error |
| 2 | Stats Error | Unable to retrieve processing statistics (infrastructure issue) |
| 3 | Expected Shutdown | Stats unavailable due to expected shutdown conditions |
| 8 | Real Failures | One or more files had genuine processing failures |

### Failure Classification

The system distinguishes between two types of failures:

#### Real Failures (RealFailedCount)
Genuine processing errors that indicate problems with:
- Intent file format or content
- Schema validation failures
- Porch execution errors
- Network connectivity issues
- Permission problems

#### Shutdown Failures (ShutdownFailedCount)
Failures caused by graceful shutdown:
- Context cancellation
- Process termination signals
- Worker pool drainage
- In-flight request cancellations

### Once-Mode Completion Logic

```go
func determineExitCode(stats ProcessingStats) int {
    // Infrastructure issues preventing stats collection
    if stats.Error != nil {
        if isExpectedShutdownError(stats.Error) {
            return 3  // Expected during shutdown
        }
        return 2  // Unexpected infrastructure issue
    }
    
    // Real failures take precedence
    if stats.RealFailedCount > 0 {
        return 8  // Actual processing failures
    }
    
    // Only shutdown failures - acceptable
    if stats.ShutdownFailedCount > 0 {
        return 0  // Success (shutdown is graceful)
    }
    
    // All files processed successfully
    return 0
}
```

### Work Completion Detection

Once-mode considers work complete when:

1. **All files processed**: `ProcessedCount + FailedCount >= ExpectedFileCount`
2. **Timeout reached**: Configurable timeout expires (default: 30s)
3. **Shutdown signal**: SIGTERM/SIGINT received
4. **No new files**: No files detected in initial scan

---

## Windows Timing and Backoff Strategy

### File System Race Conditions

Windows file systems exhibit unique race conditions due to:
- Antivirus software scanning newly created files
- File indexing services locking files temporarily
- SMB/network drive synchronization delays
- Handle cleanup delays after process termination

### Retry Logic Implementation

```go
const (
    maxFileRetries    = 10                    // Maximum retry attempts
    baseRetryDelay    = 5 * time.Millisecond  // Initial retry delay
    maxRetryDelay     = 500 * time.Millisecond // Maximum backoff delay
    fileSyncTimeout   = 2 * time.Second       // Total timeout for operations
)

func retryWithExponentialBackoff(operation func() error) error {
    delay := baseRetryDelay
    
    for attempt := 1; attempt <= maxFileRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        if !isRetryableError(err) {
            return err  // Non-retryable error
        }
        
        if attempt == maxFileRetries {
            return fmt.Errorf("failed after %d retries: %w", maxFileRetries, err)
        }
        
        time.Sleep(delay)
        delay = min(delay * 2, maxRetryDelay)  // Exponential backoff with cap
    }
}
```

### Windows-Specific Error Detection

```go
func isRetryableError(err error) bool {
    // Windows error codes that indicate temporary conditions
    const (
        ERROR_SHARING_VIOLATION = 32  // File in use
        ERROR_LOCK_VIOLATION    = 33  // File locked
        ERROR_ACCESS_DENIED     = 5   // Temporary permission issue
    )
    
    if errno, ok := extractWindowsError(err); ok {
        switch errno {
        case ERROR_SHARING_VIOLATION,
             ERROR_LOCK_VIOLATION,
             ERROR_ACCESS_DENIED:
            return true
        }
    }
    
    // Pattern matching for error messages
    patterns := []string{
        "sharing violation",
        "being used by another process",
        "access is denied",
    }
    
    errorStr := strings.ToLower(err.Error())
    for _, pattern := range patterns {
        if strings.Contains(errorStr, pattern) {
            return true
        }
    }
    
    return false
}
```

### Atomic Operation Sequence

The system ensures atomicity through a carefully orchestrated sequence:

1. **Write Phase**
   ```go
   // 1. Write to temporary file
   tempFile := targetFile + ".tmp"
   writeFileWithSync(tempFile, data)
   
   // 2. Ensure data is flushed to disk
   file.Sync()  // Force OS buffer flush
   
   // 3. Atomic rename (retry on Windows)
   renameFileWithRetry(tempFile, targetFile)
   ```

2. **Move Phase**
   ```go
   // 1. Try atomic rename (same volume)
   if err := os.Rename(src, dst); err == nil {
       return nil
   }
   
   // 2. Fall back to copy+delete (cross-volume)
   copyFileWithSync(src, dst)
   removeFileWithRetry(src)
   ```

3. **Record Phase**
   ```go
   // 1. Update in-memory state
   fileState.recordProcessed(filename)
   
   // 2. Write status file atomically
   writeStatusFileAtomic(filename, status, message)
   
   // 3. Update persistent state
   updateProcessedList(filename)
   ```

---

## Cross-Platform Compatibility

### Platform Abstraction Layer

The system uses build tags to provide platform-specific implementations:

```go
// fsync_windows.go
// +build windows

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
    // Windows-specific implementation with retry logic
}

// fsync_unix.go
// +build !windows

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
    // Unix implementation using standard rename semantics
}
```

### Batch File Execution (Windows)

Windows requires special handling for executable scripts:

```go
func executePorchCommand(ctx context.Context, porchPath string, args []string) error {
    if runtime.GOOS == "windows" {
        if strings.HasSuffix(porchPath, ".bat") || strings.HasSuffix(porchPath, ".cmd") {
            // Windows batch files require cmd.exe
            cmd := exec.CommandContext(ctx, "cmd.exe", "/C", porchPath)
            cmd.Args = append(cmd.Args, args...)
            return cmd.Run()
        }
    }
    
    // Standard execution for executables
    return exec.CommandContext(ctx, porchPath, args...).Run()
}
```

### Path Validation and Normalization

```go
func validateAndNormalizePath(path string) (string, error) {
    // Clean path (handles ../ and ./ sequences)
    cleanPath := filepath.Clean(path)
    
    // Convert to absolute path
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        return "", fmt.Errorf("invalid path: %w", err)
    }
    
    // Validate existence and permissions
    info, err := os.Stat(absPath)
    if err != nil {
        if os.IsNotExist(err) {
            // Check if parent exists for creation
            parent := filepath.Dir(absPath)
            if _, err := os.Stat(parent); err != nil {
                return "", fmt.Errorf("parent directory does not exist: %s", parent)
            }
            return absPath, nil  // Can be created
        }
        return "", err
    }
    
    // Verify it's a directory if it exists
    if !info.IsDir() {
        return "", fmt.Errorf("path exists but is not a directory: %s", absPath)
    }
    
    return absPath, nil
}
```

### Platform-Specific Error Handling

```go
type PlatformError struct {
    OS       string
    Original error
    Retryable bool
}

func wrapPlatformError(err error) *PlatformError {
    pe := &PlatformError{
        OS:       runtime.GOOS,
        Original: err,
    }
    
    switch runtime.GOOS {
    case "windows":
        pe.Retryable = isWindowsRetryableError(err)
    case "linux", "darwin":
        pe.Retryable = isUnixRetryableError(err)
    default:
        pe.Retryable = false
    }
    
    return pe
}
```

---

## Concurrency Model

### Worker Pool Design

The system uses a bounded worker pool to control resource usage:

```go
type WorkerPool struct {
    workQueue       chan WorkItem
    workers         sync.WaitGroup
    maxWorkers      int
    activeWorkers   int64  // atomic counter
    processingItems int64  // atomic counter
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.maxWorkers; i++ {
        wp.workers.Add(1)
        go wp.worker(i)
    }
}

func (wp *WorkerPool) worker(id int) {
    defer wp.workers.Done()
    defer atomic.AddInt64(&wp.activeWorkers, -1)
    
    atomic.AddInt64(&wp.activeWorkers, 1)
    
    for item := range wp.workQueue {
        atomic.AddInt64(&wp.processingItems, 1)
        wp.processItem(item)
        atomic.AddInt64(&wp.processingItems, -1)
    }
}
```

### State Management Synchronization

The system uses fine-grained locking to minimize contention:

```go
type FileProcessingState struct {
    processing   map[string]*sync.Mutex  // Per-file locks
    recentEvents map[string]time.Time    // Deduplication cache
    mu           sync.RWMutex            // Protects maps
}

func (fps *FileProcessingState) TryProcess(filename string) bool {
    fps.mu.Lock()
    defer fps.mu.Unlock()
    
    // Check if already processing
    if _, exists := fps.processing[filename]; exists {
        return false  // Already being processed
    }
    
    // Check recent events (deduplication)
    if lastTime, exists := fps.recentEvents[filename]; exists {
        if time.Since(lastTime) < 100*time.Millisecond {
            return false  // Too recent, likely duplicate
        }
    }
    
    // Mark as processing
    fps.processing[filename] = &sync.Mutex{}
    fps.recentEvents[filename] = time.Now()
    return true
}
```

### Channel-Based Shutdown Coordination

```go
type ShutdownCoordinator struct {
    ctx           context.Context
    cancel        context.CancelFunc
    shutdownChan  chan struct{}
    workers       sync.WaitGroup
    drainTimeout  time.Duration
}

func (sc *ShutdownCoordinator) Shutdown() error {
    // Signal shutdown
    close(sc.shutdownChan)
    sc.cancel()
    
    // Wait for workers with timeout
    done := make(chan struct{})
    go func() {
        sc.workers.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil  // Clean shutdown
    case <-time.After(sc.drainTimeout):
        return fmt.Errorf("shutdown timeout after %v", sc.drainTimeout)
    }
}
```

### Backpressure Management

The system implements backpressure to prevent resource exhaustion:

```go
func (w *Watcher) handleFileEvent(filename string) {
    // Try to queue immediately
    select {
    case w.workQueue <- WorkItem{FilePath: filename}:
        return  // Successfully queued
    default:
        // Queue is full, implement backpressure
    }
    
    // Record backpressure event
    atomic.AddInt64(&w.metrics.BackpressureEventsTotal, 1)
    
    // Try with timeout
    ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
    defer cancel()
    
    select {
    case w.workQueue <- WorkItem{FilePath: filename, Ctx: ctx}:
        log.Printf("Queued after backpressure: %s", filename)
    case <-ctx.Done():
        log.Printf("Dropped due to backpressure: %s", filename)
        w.recordDropped(filename)
    }
}
```

---

## Error Handling and Resilience

### Error Classification

The system classifies errors into categories for appropriate handling:

```go
type ErrorCategory int

const (
    ErrorTransient ErrorCategory = iota  // Retry with backoff
    ErrorValidation                       // Don't retry, move to failed
    ErrorInfrastructure                   // Alert and retry
    ErrorShutdown                         // Expected during shutdown
    ErrorPermanent                        // Don't retry, log and skip
)

func classifyError(err error) ErrorCategory {
    if err == context.Canceled {
        return ErrorShutdown
    }
    
    errStr := strings.ToLower(err.Error())
    
    // Validation errors
    if strings.Contains(errStr, "json") || 
       strings.Contains(errStr, "schema") ||
       strings.Contains(errStr, "validation") {
        return ErrorValidation
    }
    
    // Infrastructure errors
    if strings.Contains(errStr, "connection refused") ||
       strings.Contains(errStr, "network") ||
       strings.Contains(errStr, "timeout") {
        return ErrorInfrastructure
    }
    
    // Transient errors
    if strings.Contains(errStr, "sharing violation") ||
       strings.Contains(errStr, "locked") ||
       strings.Contains(errStr, "busy") {
        return ErrorTransient
    }
    
    return ErrorPermanent
}
```

### Recovery Strategies

Different error categories trigger different recovery strategies:

```go
func handleError(err error, filename string) {
    category := classifyError(err)
    
    switch category {
    case ErrorTransient:
        // Retry with exponential backoff
        retryWithBackoff(func() error {
            return processFile(filename)
        })
        
    case ErrorValidation:
        // Move to failed directory, don't retry
        moveToFailed(filename, err.Error())
        
    case ErrorInfrastructure:
        // Alert monitoring, retry with longer backoff
        alertMonitoring("Infrastructure error", err)
        retryWithLongerBackoff(func() error {
            return processFile(filename)
        })
        
    case ErrorShutdown:
        // Expected, mark as shutdown failure
        markAsShutdownFailure(filename)
        
    case ErrorPermanent:
        // Log and move to failed
        log.Printf("Permanent error for %s: %v", filename, err)
        moveToFailed(filename, err.Error())
    }
}
```

---

## Performance Characteristics

### Throughput Metrics

Expected performance under different conditions:

| Scenario | Files/Second | Latency (p99) | CPU Usage | Memory |
|----------|--------------|---------------|-----------|---------|
| Idle | 0 | N/A | <1% | 50MB |
| Light (10 files/min) | 0.17 | 100ms | 2% | 75MB |
| Normal (100 files/min) | 1.67 | 200ms | 5% | 100MB |
| Heavy (1000 files/min) | 16.67 | 500ms | 15% | 200MB |
| Burst (100 files at once) | 20-30 | 2s | 25% | 300MB |

### Optimization Strategies

1. **Debouncing**: Reduces duplicate processing for rapidly changing files
2. **Worker Pool**: Limits concurrent operations to prevent resource exhaustion
3. **Batch Processing**: Groups operations for efficiency
4. **Caching**: Recent events cached to prevent reprocessing
5. **Lazy Initialization**: Resources allocated only when needed

### Resource Limits

```go
const (
    MaxWorkers       = 32           // Maximum concurrent workers
    MaxQueueDepth    = 1000         // Maximum queued items
    MaxFileSize      = 10 * 1024 * 1024  // 10MB max file size
    MaxMessageSize   = 64 * 1024    // 64KB max status message
    MaxStatusSize    = 1024 * 1024  // 1MB max status file
)
```

---

## Security Considerations

### Input Validation

All inputs are validated and sanitized:

```go
func sanitizeFilename(filename string) string {
    // Remove path traversal attempts
    filename = strings.ReplaceAll(filename, "..", "")
    filename = strings.ReplaceAll(filename, "~", "")
    
    // Clean path
    filename = filepath.Clean(filename)
    
    // Ensure it's under allowed directory
    if !strings.HasPrefix(filename, allowedDir) {
        return ""  // Reject
    }
    
    return filename
}
```

### Process Isolation

The system runs with minimal privileges:
- No network access except to Porch
- Limited file system access (handoff directory only)
- No shell execution except configured Porch command
- Dropped privileges after initialization

---

## Operational Guidelines

### Deployment Checklist

1. **Pre-deployment**
   - [ ] Verify Porch executable path
   - [ ] Create handoff directory with proper permissions
   - [ ] Configure monitoring endpoints
   - [ ] Set up log rotation

2. **Configuration**
   - [ ] Set appropriate worker count for hardware
   - [ ] Configure debounce duration for environment
   - [ ] Set timeout values based on network latency
   - [ ] Enable metrics collection

3. **Monitoring**
   - [ ] Configure alerts for high failure rates
   - [ ] Set up dashboard for queue depth
   - [ ] Monitor disk space in handoff directory
   - [ ] Track processing latency percentiles

### Graceful Shutdown Procedure

1. Send SIGTERM to process
2. System stops accepting new files
3. Completes in-flight processing
4. Writes final status files
5. Reports statistics
6. Exits with appropriate code

### Maintenance Operations

**Log Rotation**:
```bash
# Rotate logs daily, keep 7 days
logrotate -f /etc/logrotate.d/conductor-loop
```

**Clean Old Status Files**:
```bash
# Remove status files older than 30 days
find /var/conductor/status -name "*.status" -mtime +30 -delete
```

**Monitor Queue Depth**:
```bash
# Check current queue depth
curl http://localhost:9090/metrics | grep queue_depth
```

---

## Troubleshooting Guide

### Common Issues and Solutions

#### High CPU Usage
**Symptoms**: CPU consistently above 50%
**Causes**: 
- Too many workers for available cores
- Debounce duration too low
- File system events storm

**Solutions**:
1. Reduce worker count to `NumCPU * 2`
2. Increase debounce duration to 500ms+
3. Check for applications rapidly modifying files

#### Files Stuck in Processing
**Symptoms**: Files remain in handoff directory
**Causes**:
- Porch command hanging
- Network issues
- Invalid file format

**Solutions**:
1. Check Porch logs for errors
2. Verify network connectivity
3. Validate file against schema manually
4. Check for file locks (Windows)

#### Exit Code 8 in Once Mode
**Symptoms**: Process exits with code 8
**Causes**: Real processing failures occurred

**Solutions**:
1. Check failed directory for files
2. Review error logs for failure reasons
3. Validate failed files manually
4. Fix issues and reprocess

#### Windows "Access Denied" Errors
**Symptoms**: Frequent "access denied" errors on Windows
**Causes**:
- Antivirus scanning
- Windows Search indexing
- Other processes accessing files

**Solutions**:
1. Add handoff directory to antivirus exclusions
2. Disable Windows Search for directory
3. Increase retry count and delays
4. Use dedicated directory not accessed by other processes

### Debug Commands

**Enable Debug Logging**:
```bash
conductor-loop -debug -handoff ./handoff
```

**Test File Processing**:
```bash
echo '{"test": "intent"}' > ./handoff/test.json
tail -f conductor-loop.log
```

**Check Status Files**:
```bash
ls -la ./handoff/status/*.status
cat ./handoff/status/test.json-*.status | jq .
```

**Monitor Metrics**:
```bash
watch -n 1 'curl -s localhost:9090/metrics | grep -E "(processed|failed|queue)"'
```

---

## Appendices

### A. Error Codes Reference

| Code | Constant | Description |
|------|----------|-------------|
| 0 | EXIT_SUCCESS | Normal completion |
| 1 | EXIT_ERROR | General error |
| 2 | EXIT_STATS_ERROR | Statistics unavailable |
| 3 | EXIT_SHUTDOWN_STATS | Expected shutdown stats error |
| 8 | EXIT_REAL_FAILURES | Processing failures detected |

### B. Metrics Reference

| Metric | Type | Description |
|--------|------|-------------|
| files_processed_total | Counter | Total files successfully processed |
| files_failed_total | Counter | Total files that failed processing |
| processing_duration_seconds | Histogram | Time to process each file |
| queue_depth_current | Gauge | Current number of queued files |
| workers_active | Gauge | Currently active workers |
| backpressure_events_total | Counter | Times queue was full |

### C. Configuration Parameters

| Parameter | Default | Range | Description |
|-----------|---------|-------|-------------|
| debounce | 500ms | 10ms-5s | Event debouncing duration |
| workers | NumCPU | 1-32 | Concurrent workers |
| period | 2s | 100ms-1h | Directory scan period |
| timeout | 30s | 1s-5m | Processing timeout |
| retry | 3 | 1-10 | Retry attempts |

### D. File System Layout

```
handoff/
├── *.json                 # Incoming intent files
├── processed/            # Successfully processed files
│   └── *.json
├── failed/               # Failed processing
│   ├── *.json
│   └── *.json.error.log
├── status/               # Processing status files
│   └── *.status
└── .processed            # Idempotency tracking
```

---

*Document Version: 1.0.0*  
*Last Updated: 2025-08-21*  
*Authors: Conductor-Loop Development Team*
# Race Condition Fix - ClientAdapter

## Issue
A race condition was detected at `pkg/nephio/blueprint/generator.go:122` where multiple goroutines were concurrently accessing the `ClientAdapter.client` field without proper synchronization.

## Root Cause
The `ClientAdapter` struct was accessing its `client` field directly without any mutex protection, leading to potential data races when multiple goroutines called `ProcessIntent` simultaneously.

## Solution
Added mutex protection to the `ClientAdapter` struct:

1. **Added sync.RWMutex field** to `ClientAdapter` struct (line 106)
2. **Protected client access** with read lock in `ProcessIntent` method (lines 116-118)
3. **Used local copy** of client pointer after acquiring lock to avoid holding lock during RPC call

## Code Changes

### Before:
```go
type ClientAdapter struct {
    client *llm.Client
}

func (ca *ClientAdapter) ProcessIntent(...) {
    // Direct access without protection
    response, err := ca.client.ProcessIntent(timeoutCtx, request.Intent)
}
```

### After:
```go
type ClientAdapter struct {
    mu     sync.RWMutex
    client *llm.Client
}

func (ca *ClientAdapter) ProcessIntent(...) {
    // Protected access with read lock
    ca.mu.RLock()
    client := ca.client
    ca.mu.RUnlock()
    
    response, err := client.ProcessIntent(timeoutCtx, request.Intent)
}
```

## Verification
Created comprehensive tests to verify the fix:
- `generator_race_test.go`: Tests concurrent access patterns
- `generator_nil_test.go`: Tests nil client handling

All tests pass successfully without race conditions.

## Performance Impact
Minimal - using RWMutex with read locks allows multiple concurrent readers, only blocking during the brief moment of copying the client pointer.
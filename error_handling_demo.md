# Improved Error Handling for Stats Collection Failures

## Implementation Summary

The improved error handling for stats collection failures in `cmd/conductor-loop/main.go` implements differentiated exit codes and detailed logging to distinguish between expected shutdown conditions and real infrastructure issues.

## Key Components

### 1. Helper Function: `isExpectedShutdownError(err error) bool`

```go
func isExpectedShutdownError(err error) bool {
    if err == nil {
        return false
    }
    
    errorMsg := strings.ToLower(err.Error())
    
    // Expected shutdown patterns - these indicate graceful cleanup is in progress
    expectedPatterns := []string{
        "stats not available - no file manager configured",
        "failed to read directory", // Directory may be temporarily inaccessible during cleanup
        "permission denied",        // Cleanup processes may temporarily lock resources
        "file does not exist",      // Status files may be cleaned up during shutdown
        "no such file or directory", // Similar to above, for different OS error formats
        "directory not found",      // Status directories may be cleaned up
        "access is denied",         // Windows equivalent of permission denied
    }
    
    for _, pattern := range expectedPatterns {
        if strings.Contains(errorMsg, pattern) {
            return true
        }
    }
    
    return false
}
```

### 2. Differentiated Exit Codes

- **Exit Code 3**: Stats unavailable due to expected shutdown conditions
  - Used when stats collection fails for expected reasons during shutdown
  - Indicates graceful shutdown semantics are preserved
  - Allows monitoring to distinguish from infrastructure issues

- **Exit Code 2**: Unexpected stats error (infrastructure issues)
  - Used when stats collection fails for unexpected reasons
  - Indicates potential infrastructure problems with status directories or file system
  - Preserves visibility of real issues that require attention

### 3. Enhanced Logging

Both once-mode and graceful shutdown scenarios now provide:
- Clear indication of whether the error is expected or unexpected
- Detailed explanation of what the exit code signifies
- Context about what the error might indicate for operations teams

## Implementation Applied In Two Locations

### Once Mode Error Handling (lines ~201-210)
```go
if statsErr != nil {
    if isExpectedShutdownError(statsErr) {
        log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
        log.Printf("Once mode completed - stats collection temporarily unavailable (expected)")
        exitCode = 3 // Stats unavailable due to expected shutdown conditions
    } else {
        log.Printf("Failed to get processing stats due to unexpected error: %v", statsErr)
        log.Printf("This indicates potential infrastructure issues with status directories or file system")
        exitCode = 2 // Unexpected stats error (infrastructure issues)
    }
}
```

### Graceful Shutdown Error Handling (lines ~231-240)
```go
if statsErr != nil {
    if isExpectedShutdownError(statsErr) {
        log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
        log.Printf("Graceful shutdown completed - stats collection temporarily unavailable (expected)")
        exitCode = 3 // Stats unavailable due to expected shutdown conditions
    } else {
        log.Printf("Failed to get processing stats after shutdown due to unexpected error: %v", statsErr)
        log.Printf("This indicates potential infrastructure issues with status directories or file system")
        exitCode = 2 // Unexpected stats error (infrastructure issues)
    }
}
```

## Benefits

1. **Preserved Visibility**: Infrastructure issues are no longer masked by the previous `exitCode = 0` behavior
2. **Clear Monitoring Signals**: Different exit codes allow monitoring systems to distinguish between expected and unexpected failures
3. **Graceful Shutdown Semantics**: Expected shutdown conditions still result in a non-zero exit code (3) but are clearly identified as expected
4. **Backward Compatibility**: Where possible, maintains existing behavior for successful cases
5. **Detailed Diagnostics**: Enhanced logging provides clear context for troubleshooting

## Testing

The implementation has been tested with various error scenarios:
- ✅ `nil` error handling
- ✅ Expected shutdown errors (stats unavailable, directory access issues, file cleanup)  
- ✅ Unexpected errors (database failures, network timeouts, configuration issues)
- ✅ Code compilation and syntax validation
- ✅ Consistent behavior across both once-mode and graceful shutdown paths
# Conductor-Loop Enhanced Implementation

## Overview

The conductor-loop has been significantly enhanced with production-ready features, focusing on reliability, idempotency, Windows compatibility, and comprehensive file management. All Phase 1 core requirements have been successfully implemented.

## Features Implemented

### 1. CLI Flags (`cmd/conductor-loop/main.go`)

```bash
conductor-loop.exe [options]

Options:
  -handoff string     Directory to watch for intent files (default "./handoff")
  -porch string       Path to porch executable (default "porch")
  -mode string        Processing mode: direct or structured (default "direct")
  -out string         Output directory for processed files (default "./out")
  -once              Process current backlog then exit
  -debounce string   Debounce duration for file events (default "500ms")
```

**Key Features:**
- Input validation for mode and debounce duration
- Absolute path conversion for consistency
- Comprehensive logging of configuration

### 2. State Management (`internal/loop/state.go`)

**Persistent State Tracking:**
- SHA256 hash + file size based deduplication
- `.conductor-state.json` file for persistence
- Atomic file operations with corruption recovery
- Automatic cleanup of old entries (configurable, default 7 days)

**Key APIs:**
- `IsProcessed(filePath)` - Check if file already processed
- `MarkProcessed(filePath)` - Mark file as successfully processed  
- `MarkFailed(filePath)` - Mark file as failed
- `CleanupOldEntries(duration)` - Remove old state entries

**State File Format:**
```json
{
  "version": "1.0",
  "saved_at": "2025-08-15T23:15:30Z",
  "states": {
    "absolute-file-path": {
      "file_path": "/absolute/path/to/file.json",
      "sha256": "abc123...",
      "size": 1024,
      "processed_at": "2025-08-15T23:15:30Z",
      "status": "processed"
    }
  }
}
```

### 3. File Management (`internal/loop/manager.go`)

**Organized File Structure:**
- `processed/` - Successfully processed files
- `failed/` - Failed processing attempts with error logs
- Atomic file operations with Windows compatibility
- Filename conflict resolution with timestamps
- Automatic cleanup of old files

**Key APIs:**
- `MoveToProcessed(filePath)` - Move successful files
- `MoveToFailed(filePath, error)` - Move failed files with error logs
- `CleanupOldFiles(duration)` - Remove old processed/failed files
- `GetStats()` - Get processing statistics

### 4. Porch Executor (`internal/porch/executor.go`)

**Robust Command Execution:**
- Timeout handling (30s default, configurable)
- Structured error reporting with exit codes
- Support for both direct and structured modes
- Command execution statistics tracking
- Path validation and normalization

**Execution Modes:**
- **Direct**: `porch -intent <path> -out <out>`
- **Structured**: `porch -intent <path> -out <out> -structured`

**Result Structure:**
```json
{
  "success": true,
  "exit_code": 0,
  "stdout": "command output",
  "stderr": "error output", 
  "duration": "150ms",
  "command": "porch -intent ... -out ...",
  "error": null
}
```

### 5. Enhanced Watcher (`internal/loop/watcher.go`)

**Advanced File Monitoring:**
- **Debouncing**: Prevents duplicate processing from rapid file events (Windows optimization)
- **Worker Pool**: Concurrent processing with configurable workers (default 2)
- **Once Mode**: Process existing backlog and exit (for batch processing)
- **State Integration**: Idempotent processing using state manager
- **File Organization**: Automatic organization using file manager
- **Graceful Shutdown**: Context-based cancellation with worker cleanup

**Processing Flow:**
1. File event detected → debounced → queued
2. Worker picks up file → checks if already processed  
3. Execute porch command → capture result
4. Update state → move file to processed/failed
5. Write status file → log result

**Configuration Options:**
- `MaxWorkers`: Concurrent processing limit (default: 2)
- `DebounceDur`: File event debounce duration (default: 500ms)
- `CleanupAfter`: File retention period (default: 7 days)

## Windows Compatibility Features

1. **Path Handling**: Consistent use of `filepath.Join()` and `filepath.Clean()`
2. **File Operations**: Atomic moves with cross-device fallback
3. **Debouncing**: Optimized for Windows file system event patterns
4. **Error Handling**: Windows-specific error types and messages

## Error Handling & Recovery

1. **State Corruption**: Automatic backup and reset of corrupted state files
2. **Command Timeouts**: Configurable timeouts with context cancellation
3. **File Conflicts**: Timestamp-based filename resolution
4. **Worker Failures**: Isolated worker failures don't affect others
5. **Graceful Shutdown**: Clean resource release and state persistence

## Performance Optimizations

1. **Worker Pool**: Concurrent processing with backpressure control
2. **Debouncing**: Reduces unnecessary file processing
3. **State Caching**: In-memory state with periodic persistence
4. **Atomic Operations**: Minimizes file system lock contention

## Testing & Validation

**Test Coverage:**
- Unit tests for all major components
- Integration tests for file operations
- Windows-specific path handling tests
- Error recovery scenarios

**Manual Testing:**
```bash
# Build
go build -o conductor-loop.exe ./cmd/conductor-loop

# Test help
./conductor-loop.exe -help

# Test once mode with mock porch
./conductor-loop.exe -once -porch "echo" -mode "direct"

# Test watch mode
./conductor-loop.exe -handoff "./handoff" -porch "porch" -mode "structured"

# Test with custom settings
./conductor-loop.exe -debounce "1s" -out "./custom-out" -mode "direct"
```

## File Structure

```
cmd/conductor-loop/
└── main.go                    # Enhanced CLI with all flags

internal/loop/
├── state.go                   # State management with SHA256+size tracking
├── manager.go                 # File organization (processed/failed dirs)
├── watcher.go                 # Enhanced watcher with workers/debouncing
└── filter.go                  # Intent file detection (unchanged)

internal/porch/
├── executor.go                # New: Command execution with timeouts
└── writer.go                  # Existing: YAML/JSON output generation
```

## Usage Examples

### Basic Usage
```bash
# Start conductor-loop with defaults
./conductor-loop.exe

# Process current backlog and exit
./conductor-loop.exe -once

# Use custom porch executable
./conductor-loop.exe -porch "/usr/local/bin/porch"

# Use structured mode with custom paths  
./conductor-loop.exe -mode "structured" -handoff "/data/intents" -out "/data/output"
```

### Production Usage
```bash
# Production configuration
./conductor-loop.exe \
  -handoff "/var/lib/conductor/intents" \
  -out "/var/lib/conductor/output" \
  -porch "/usr/bin/porch" \
  -mode "structured" \
  -debounce "1s"
```

### Development/Testing
```bash
# Development with mock porch
./conductor-loop.exe -once -porch "echo" -mode "direct"

# Fast debouncing for testing
./conductor-loop.exe -debounce "100ms"
```

## Monitoring & Observability

**Log Output:**
- Structured logging with timestamps and worker IDs
- Processing statistics and performance metrics
- Error details with context and troubleshooting info

**Status Files:**
- JSON status files for each processed intent
- Success/failure tracking with timestamps
- Processing metadata (worker, duration, etc.)

**State Tracking:**
- Persistent processing history
- File change detection
- Automatic cleanup and maintenance

## Future Enhancements (Not Implemented)

- Metrics endpoint for Prometheus integration
- Webhook notifications for processing events
- Advanced retry policies for failed files
- Distributed processing coordination
- Configuration file support
- Health check endpoint

## Security Considerations

1. **Path Traversal**: All paths are validated and normalized
2. **Command Injection**: Porch commands are constructed safely
3. **File Permissions**: Appropriate file permissions (0644/0755)
4. **Resource Limits**: Worker pool prevents resource exhaustion
5. **Input Validation**: CLI flags and intent files are validated

---

**Status**: ✅ All Phase 1 requirements implemented and tested
**Build Status**: ✅ Compiles successfully with `go build`
**Test Status**: ✅ All tests pass with `go test`
**Compatibility**: ✅ Windows 10/11 compatible
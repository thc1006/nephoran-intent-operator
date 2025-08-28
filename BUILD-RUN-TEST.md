# BUILD-RUN-TEST - Conductor Loop

## Overview
Polling-based conductor that periodically scans a handoff directory for intent-*.json files, processes them with idempotent logic, and outputs KRM patches or calls porch directly.

## Requirements
- Go 1.24+
- Windows/Linux/macOS
- Optional: `porch` executable in PATH for actual processing

## Build

```powershell
# Windows PowerShell
go build -o conductor-loop.exe ./cmd/conductor-loop

# Linux/macOS
go build -o conductor-loop ./cmd/conductor-loop
```

## Run

### Basic Usage (Polling Mode)
```powershell
# Start with default 2s polling period
./conductor-loop.exe --handoff ./handoff --out ./out --period 2s

# Custom polling period
./conductor-loop.exe --handoff ./handoff --out ./out --period 5s

# Process once and exit
./conductor-loop.exe --handoff ./handoff --out ./out --once

# With Porch URL for direct API calls
./conductor-loop.exe --handoff ./handoff --porch-url http://localhost:8080 --period 2s
```

### Flags
- `--handoff`: Directory to watch for intent files (default: `./handoff`)
- `--out`: Output directory for processed files (default: `./out`)
- `--period`: Polling period for scanning directory (default: `2s`)
- `--porch`: Path to porch executable (default: `porch`)
- `--porch-url`: Porch HTTP URL for direct API calls (optional)
- `--mode`: Processing mode: `direct` or `structured` (default: `direct`)
- `--once`: Process current backlog then exit
- `--debounce`: Debounce duration for file events (default: `500ms`)

### Log Format
The conductor uses structured logging with prefixes:
- `LOOP:START` - Conductor starting
- `LOOP:SCAN` - Scanning directory
- `LOOP:PROCESS` - Processing a file
- `LOOP:SKIP_DUP` - Skipping duplicate (same SHA256)
- `LOOP:DONE` - Processing completed
- `LOOP:ERROR` - Error occurred
- `LOOP:WARNING` - Non-critical warning

## Test

### Unit Tests
```powershell
# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/loop/... -v

# With coverage
go test ./... -v -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Test
```powershell
# Create test intent file
mkdir -p handoff
cat > handoff/intent-test.json << 'EOF'
{
  "apiVersion": "network.nephio.io/v1alpha1",
  "kind": "NetworkIntent",
  "metadata": {
    "name": "test-intent",
    "namespace": "default"
  },
  "spec": {
    "action": "scale",
    "target": "ran-du-01",
    "parameters": {
      "replicas": 3
    }
  }
}
EOF

# Run conductor in once mode
./conductor-loop.exe --handoff handoff --out out --once

# Check state file
cat handoff/.conductor-state.json

# Check processed/failed directories
ls handoff/processed/
ls handoff/failed/
```

### Mock Porch Server (for testing)
```powershell
# Create mock porch script (Windows)
cat > porch.bat << 'EOF'
@echo off
echo Mock porch processing: %*
exit /b 0
EOF

# Create mock porch script (Linux/macOS)
cat > porch << 'EOF'
#!/bin/bash
echo "Mock porch processing: $@"
exit 0
EOF
chmod +x porch

# Add to PATH
export PATH=".:$PATH"  # Linux/macOS
set PATH=%CD%;%PATH%   # Windows
```

## State Management
The conductor maintains state in `.conductor-state.json` to avoid reprocessing:
```json
{
  "version": "1.0",
  "saved_at": "2025-08-17T01:00:00Z",
  "states": {
    "C:\\handoff\\intent-test.json": {
      "file_path": "C:\\handoff\\intent-test.json",
      "sha256": "abc123...",
      "size": 256,
      "processed_at": "2025-08-17T01:00:00Z",
      "status": "processed"
    }
  }
}
```

## Directory Structure
```
handoff/
├── .conductor-state.json    # Processing state
├── intent-*.json           # Pending files
├── processed/              # Successfully processed
│   └── intent-*.json
├── failed/                 # Failed processing
│   ├── intent-*.json
│   └── intent-*.json.error # Error details
└── status/                 # Processing status
    └── intent-*.json.status

out/                        # KRM patches output
└── intent-*.yaml
```

## Monitoring

### Health Check
```powershell
# Check if running
ps | grep conductor-loop

# Check logs (with timestamps)
./conductor-loop.exe 2>&1 | tee conductor.log

# Monitor state changes
watch -n 1 cat handoff/.conductor-state.json
```

### Metrics
The conductor tracks:
- Files processed/failed count
- Processing duration
- Queue depth
- Worker utilization

## Troubleshooting

### Common Issues

1. **"porch: executable file not found"**
   - Install porch or use mock script
   - Specify full path with `--porch /path/to/porch`

2. **"work queue full"**
   - Increase worker count in code
   - Reduce polling period
   - Process backlog with `--once`

3. **Files not being processed**
   - Check file naming: must match `intent-*.json`
   - Verify permissions on handoff directory
   - Check `.conductor-state.json` for duplicates

4. **State file corrupted**
   - Delete `.conductor-state.json` to reset
   - Backup created automatically as `.conductor-state.json.backup.TIMESTAMP`

## Performance Tuning

```powershell
# Fast polling for high throughput
./conductor-loop.exe --period 500ms --debounce 100ms

# Slow polling for resource conservation
./conductor-loop.exe --period 10s --debounce 2s

# Batch processing
./conductor-loop.exe --once  # Process backlog
./conductor-loop.exe --period 30s  # Then normal operation
```

## Security Considerations
- State file contains SHA256 hashes only (no sensitive data)
- Failed files isolated in separate directory
- Atomic file operations prevent corruption
- Windows-specific path handling for cross-platform compatibility
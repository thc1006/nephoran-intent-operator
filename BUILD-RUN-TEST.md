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
=======
# BUILD-RUN-TEST Guide for porch-direct

## Overview

The `porch-direct` CLI tool reads an intent JSON file and interacts with the Porch API to create/update KRM packages for network function scaling.

## Prerequisites

- Go 1.24+ installed
- Access to a Porch API endpoint (or use dry-run mode)
- Valid intent JSON file (conforming to `docs/contracts/intent.schema.json`)

## Building

### Windows PowerShell
```powershell
# Build the CLI tool
go build -o porch-direct.exe ./cmd/porch-direct

# Or with optimizations
go build -ldflags="-s -w" -o porch-direct.exe ./cmd/porch-direct
```

### Linux/Mac
```bash
# Build the CLI tool
go build -o porch-direct ./cmd/porch-direct

# Or with optimizations
go build -ldflags="-s -w" -o porch-direct ./cmd/porch-direct
```

## Running

### Basic Usage

```bash
# Generate KRM package to filesystem (default mode)
./porch-direct --intent sample-intent.json

# Dry-run mode (writes to ./out/)
./porch-direct --intent sample-intent.json --dry-run

# Minimal package (Deployment + Kptfile only)
./porch-direct --intent sample-intent.json --minimal

# Submit to Porch API
./porch-direct --intent sample-intent.json \
  --porch http://localhost:9443 \
  --repo ran-packages \
  --package gnb-scaling \
  --workspace default \
  --namespace ran-sim

# Submit with auto-approval
./porch-direct --intent sample-intent.json \
  --porch http://localhost:9443 \
  --auto-approve
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--intent` | (required) | Path to intent JSON file |
| `--out` | `examples/packages/direct` | Output directory for generated KRM |
| `--dry-run` | `false` | Dry-run mode (no files written) |
| `--minimal` | `false` | Generate minimal package |
| `--repo` | (auto-resolved) | Target Porch repository |
| `--package` | (auto-resolved) | Target package name |
| `--workspace` | `default` | Workspace identifier |
| `--namespace` | `default` | Target namespace |
| `--porch` | `http://localhost:9443` | Porch API base URL |
| `--auto-approve` | `false` | Auto-approve package proposals |

### Sample Intent Files

Create a test intent file:

```json
{
  "intent_type": "scaling",
  "target": "gnb",
  "namespace": "ran-sim",
  "replicas": 3,
  "reason": "Increased traffic load",
  "source": "planner",
  "correlation_id": "test-001"
}
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./cmd/porch-direct
go test ./pkg/porch
go test ./internal/generator

# Run with verbose output
go test -v ./...
```

### Integration Tests

```bash
# Test dry-run mode
./porch-direct --intent test-intent.json --dry-run

# Verify output files
ls -la ./out/
cat ./out/porch-package-request.json
cat ./out/overlays/deployment.yaml
```

### E2E Test with Local Porch

```bash
# 1. Start local Porch server (if available)
# kubectl port-forward -n porch-system svc/porch-server 9443:443

# 2. Create test intent
cat > test-intent.json <<EOF
{
  "intent_type": "scaling",
  "target": "gnb",
  "namespace": "ran-sim",
  "replicas": 5,
  "source": "test"
}
EOF

# 3. Submit to Porch
./porch-direct --intent test-intent.json \
  --porch http://localhost:9443 \
  --repo test-repo \
  --package test-package

# 4. Verify package creation
# Check Porch UI or use kpt CLI
```
>>>>>>> origin/integrate/mvp

## Troubleshooting

### Common Issues

<<<<<<< HEAD
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
=======
1. **"go.mod not found" error**
   - Ensure you're running from the project root
   - Check that go.mod exists in the repository

2. **"intent validation failed" error**
   - Verify intent JSON conforms to schema
   - Check required fields: intent_type, target, namespace, replicas

3. **"failed to connect to Porch" error**
   - Verify Porch URL is correct
   - Check network connectivity
   - Use --dry-run mode for testing without Porch

4. **"package not created" error**
   - Check Porch repository exists
   - Verify permissions to create packages
   - Review Porch server logs

### Debug Mode

```bash
# Enable verbose logging
export DEBUG=true
./porch-direct --intent sample-intent.json --dry-run

# Check generated files
find ./out -type f -exec echo {} \; -exec cat {} \;
```

## Development Workflow

1. **Make changes** to the code
2. **Build** the binary: `go build -o porch-direct ./cmd/porch-direct`
3. **Test** with sample intent: `./porch-direct --intent sample-intent.json --dry-run`
4. **Run tests**: `go test ./...`
5. **Submit PR** with test results

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test porch-direct
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: go build ./cmd/porch-direct
      - run: go test -cover ./...
      - run: ./porch-direct --intent sample-intent.json --dry-run
```

## Performance Testing

```bash
# Generate multiple intents
for i in {1..10}; do
  echo '{"intent_type":"scaling","target":"gnb'$i'","namespace":"test","replicas":'$i'}' > intent-$i.json
done

# Time execution
time for i in {1..10}; do
  ./porch-direct --intent intent-$i.json --dry-run
done
```

## Security Considerations

- Never commit Porch API credentials
- Use environment variables for sensitive data
- Validate all intent JSON inputs
- Run with minimal permissions
- Review generated KRM for security implications
>>>>>>> origin/integrate/mvp

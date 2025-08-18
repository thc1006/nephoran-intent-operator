# BUILD-RUN-TEST.windows.md

# Conductor Watch - Build, Run, and Test Guide for Windows

## Prerequisites

- Go 1.24+ installed
- PowerShell 5.0+
- Git Bash (optional, for Unix-like commands)

## Build Instructions

### 1. Install Dependencies

```powershell
# Navigate to project root
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-conductor-watch

# Download dependencies
go mod download

# Verify dependencies
go mod verify
```

### 2. Build the Binary

```powershell
# Build the conductor-watch executable
go build -o conductor-watch.exe .\cmd\conductor-watch

# Verify build
if (Test-Path .\conductor-watch.exe) {
    Write-Host "✓ Build successful" -ForegroundColor Green
    (Get-Item .\conductor-watch.exe).Length
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
}
```

## Run Tests

### 1. Unit Tests

```powershell
# Run all tests with verbose output
go test -v ./internal/watch/...

# Run tests with coverage
go test -v -cover ./internal/watch/...

# Run specific test
go test -v -run TestValidator ./internal/watch
```

Expected output:
```
=== RUN   TestValidator
=== RUN   TestValidator/valid_intent
=== RUN   TestValidator/missing_required_field
=== RUN   TestValidator/wrong_intent_type
--- PASS: TestValidator (0.XX s)
    --- PASS: TestValidator/valid_intent (0.00s)
    --- PASS: TestValidator/missing_required_field (0.00s)
    --- PASS: TestValidator/wrong_intent_type (0.00s)
PASS
```

### 2. Integration Tests

```powershell
# Run main package tests
go test -v ./cmd/conductor-watch/...
```

## Setup Test Environment

### 1. Create Directory Structure

```powershell
# Create handoff directory
New-Item -ItemType Directory -Force -Path .\handoff | Out-Null

# Verify directory creation
Get-ChildItem -Path . -Directory | Where-Object Name -eq "handoff"
```

### 2. Create Test Intent Files

```powershell
# Valid intent file
@'
{
  "intent_type": "scaling",
  "target": "my-deployment",
  "namespace": "default",
  "replicas": 3,
  "reason": "Testing conductor-watch",
  "source": "test",
  "correlation_id": "test-001"
}
'@ | Out-File -FilePath .\handoff\intent-valid-001.json -Encoding utf8

# Invalid intent (wrong type)
@'
{
  "intent_type": "update",
  "target": "my-deployment",
  "namespace": "default",
  "replicas": 3
}
'@ | Out-File -FilePath .\handoff\intent-invalid-type.json -Encoding utf8

# Invalid intent (missing field)
@'
{
  "intent_type": "scaling",
  "target": "my-deployment",
  "replicas": 3
}
'@ | Out-File -FilePath .\handoff\intent-invalid-missing.json -Encoding utf8

# Verify files created
Get-ChildItem .\handoff\intent-*.json | Select-Object Name, Length
```

## Run the Application

### 1. Basic Run (Default Settings)

```powershell
.\conductor-watch.exe --handoff .\handoff
```

Expected output:
```
[conductor-watch] 2025/08/17 12:00:00.123456 Starting conductor-watch:
[conductor-watch] 2025/08/17 12:00:00.123456   Watching: C:\Users\tingy\dev\_worktrees\nephoran\feat-conductor-watch\handoff
[conductor-watch] 2025/08/17 12:00:00.123456   Schema: C:\Users\tingy\dev\_worktrees\nephoran\feat-conductor-watch\docs\contracts\intent.schema.json
[conductor-watch] 2025/08/17 12:00:00.123456   Debounce: 300ms
[conductor-watch] 2025/08/17 12:00:00.234567 WATCH:OK intent-valid-001.json - type=scaling target=my-deployment namespace=default replicas=3
[conductor-watch] 2025/08/17 12:00:00.234567 WATCH:INVALID intent-invalid-type.json - schema validation failed: value must be 'scaling'
[conductor-watch] 2025/08/17 12:00:00.234567 WATCH:INVALID intent-invalid-missing.json - schema validation failed: missing property 'namespace'
[conductor-watch] 2025/08/17 12:00:00.234567 Processed 3 existing intent files on startup
```

### 2. Run with HTTP POST Endpoint

```powershell
# Start a test HTTP server (in another terminal)
# Using Python:
python -m http.server 8080

# Or using netcat (if available):
# nc -l -p 8080

# Run conductor-watch with POST URL
.\conductor-watch.exe --handoff .\handoff --post-url "http://localhost:8080/intents"
```

Expected additional output:
```
[conductor-watch] 2025/08/17 12:00:00.123456   POST URL: http://localhost:8080/intents
[conductor-watch] 2025/08/17 12:00:00.345678 WATCH:POST_OK intent-valid-001.json - Status=200 Response=...
```

### 3. Run with Custom Debounce

```powershell
# Use 500ms debounce for slower file systems
.\conductor-watch.exe --handoff .\handoff --debounce-ms 500
```

## Live Testing

### 1. Test File Watch (New File)

In Terminal 1, start the watcher:
```powershell
.\conductor-watch.exe --handoff .\handoff
```

In Terminal 2, create a new intent file:
```powershell
# Generate timestamp
$timestamp = Get-Date -Format "yyyyMMddHHmmss"

# Create new valid intent
@"
{
  "intent_type": "scaling",
  "target": "new-app-$timestamp",
  "namespace": "production",
  "replicas": 5,
  "source": "test",
  "correlation_id": "live-test-$timestamp"
}
"@ | Out-File -FilePath ".\handoff\intent-$timestamp.json" -Encoding utf8

Write-Host "Created intent-$timestamp.json" -ForegroundColor Green
```

Expected output in Terminal 1:
```
[conductor-watch] 2025/08/17 12:01:00.123456 WATCH:NEW intent-20250817120100.json
[conductor-watch] 2025/08/17 12:01:00.423456 WATCH:OK intent-20250817120100.json - type=scaling target=new-app-20250817120100 namespace=production replicas=5
```

### 2. Test Debouncing (Partial Write Simulation)

```powershell
# Create a large file that simulates slow write
$largeIntent = @{
    intent_type = "scaling"
    target = "large-deployment"
    namespace = "default"
    replicas = 10
    reason = "x" * 10000  # Large reason field
} | ConvertTo-Json

# Write in chunks to simulate partial write
$filePath = ".\handoff\intent-partial-$(Get-Date -Format 'HHmmss').json"
$bytes = [System.Text.Encoding]::UTF8.GetBytes($largeIntent)

# Write first half
$stream = [System.IO.File]::OpenWrite($filePath)
$stream.Write($bytes, 0, $bytes.Length / 2)
$stream.Flush()

# Small delay
Start-Sleep -Milliseconds 100

# Write second half
$stream.Write($bytes, $bytes.Length / 2, $bytes.Length - $bytes.Length / 2)
$stream.Close()

Write-Host "Partial write test completed" -ForegroundColor Yellow
```

The watcher should only process the file once after debounce delay (300ms default).

### 3. Test Invalid JSON Handling

```powershell
# Create malformed JSON
@'
{
  "intent_type": "scaling",
  "target": "broken-json
  "namespace": "default"
'@ | Out-File -FilePath ".\handoff\intent-broken.json" -Encoding utf8
```

Expected output:
```
[conductor-watch] 2025/08/17 12:02:00.123456 WATCH:NEW intent-broken.json
[conductor-watch] 2025/08/17 12:02:00.423456 WATCH:INVALID intent-broken.json - invalid JSON: unexpected end of JSON input
```

## Verify Logs are Greppable

### PowerShell

```powershell
# Redirect output to file
.\conductor-watch.exe --handoff .\handoff 2>&1 | Tee-Object -FilePath conductor-watch.log

# In another terminal, filter logs
Select-String "WATCH:NEW" conductor-watch.log
Select-String "WATCH:OK" conductor-watch.log
Select-String "WATCH:INVALID" conductor-watch.log
Select-String "WATCH:POST" conductor-watch.log
```

### Git Bash

```bash
# Run and grep in real-time
./conductor-watch.exe --handoff ./handoff 2>&1 | grep --line-buffered "WATCH:"

# Filter specific patterns
./conductor-watch.exe --handoff ./handoff 2>&1 | grep -E "WATCH:(NEW|OK|INVALID)"
```

## Graceful Shutdown

Press `Ctrl+C` in the terminal running conductor-watch.

Expected output:
```
[conductor-watch] 2025/08/17 12:03:00.123456 Received signal interrupt, shutting down gracefully
[conductor-watch] 2025/08/17 12:03:00.123456 Conductor-watch stopped
```

## Troubleshooting

### 1. Schema File Not Found

Error:
```
Schema file not found at C:\...\docs\contracts\intent.schema.json
```

Solution:
```powershell
# Verify schema exists
Test-Path .\docs\contracts\intent.schema.json

# If missing, specify custom path
.\conductor-watch.exe --schema .\path\to\schema.json --handoff .\handoff
```

### 2. Permission Denied

Error:
```
Failed to create handoff directory: Access is denied
```

Solution:
```powershell
# Run PowerShell as Administrator or use a user-writable directory
.\conductor-watch.exe --handoff "$env:TEMP\handoff"
```

### 3. File Already in Use (Windows specific)

Error:
```
The process cannot access the file because it is being used by another process
```

Solution:
- Increase debounce delay: `--debounce-ms 500`
- Ensure no antivirus is scanning the directory
- Close any file explorers showing the handoff directory

### 4. POST Endpoint Connection Refused

Error:
```
WATCH:POST_ERROR Failed to POST: dial tcp: connection refused
```

Solution:
```powershell
# Test endpoint is reachable
Test-NetConnection -ComputerName localhost -Port 8080

# Use curl to test manually
curl -X POST http://localhost:8080/intents -H "Content-Type: application/json" -d "{}"
```

## Performance Testing

### Batch File Creation

```powershell
# Create 100 intent files rapidly
1..100 | ForEach-Object {
    $intent = @{
        intent_type = "scaling"
        target = "app-$_"
        namespace = "perf-test"
        replicas = $_ % 10 + 1
    } | ConvertTo-Json
    
    $intent | Out-File -FilePath ".\handoff\intent-perf-$_.json" -Encoding utf8
}

Write-Host "Created 100 test files" -ForegroundColor Green
```

The watcher should process all files with proper debouncing.

## Cleanup

```powershell
# Remove test files
Remove-Item .\handoff\intent-*.json -Force

# Remove binary
Remove-Item .\conductor-watch.exe -Force

# Remove log file if created
if (Test-Path .\conductor-watch.log) {
    Remove-Item .\conductor-watch.log -Force
}

Write-Host "Cleanup completed" -ForegroundColor Green
```

## Expected Log Patterns

### Successful Processing
```
WATCH:NEW <filename>          # New file detected
WATCH:OK <filename> - type=scaling target=<target> namespace=<ns> replicas=<n>
```

### Validation Failures
```
WATCH:INVALID <filename> - invalid JSON: <error>
WATCH:INVALID <filename> - schema validation failed: <error>
```

### HTTP POST Results
```
WATCH:POST_OK <filename> - Status=200 Response=<body>
WATCH:POST_FAILED <filename> - Status=<code> Response=<body>
WATCH:POST_ERROR <filename> - Failed to POST: <error>
```

## Success Criteria Verification

✓ Watches ./handoff directory for intent-*.json files
✓ Validates against docs/contracts/intent.schema.json
✓ Logs with WATCH:NEW/OK/INVALID prefixes
✓ Supports --handoff and --post-url flags
✓ Implements 300ms debouncing (configurable)
✓ Windows path compatible (absolute paths)
✓ Graceful shutdown on Ctrl+C
✓ Processes existing files on startup
✓ Concurrent-safe validation
✓ HTTP POST for valid intents (optional)
# BUILD-RUN-TEST.windows.md

## Windows PowerShell Build, Run, and Test Guide

### Prerequisites
- Go 1.24+ installed
- PowerShell 5.1+ or PowerShell Core 7+
- Windows 10/11 or Windows Server 2019+

### Build Steps

```powershell
# Navigate to project root
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-conductor-loop

# Clean previous builds
Remove-Item -Path ".\conductor-loop.exe" -ErrorAction SilentlyContinue

# Build the conductor-loop executable
go build -o conductor-loop.exe .\cmd\conductor-loop

# Verify build
if (Test-Path ".\conductor-loop.exe") {
    Write-Host "✓ Build successful" -ForegroundColor Green
    .\conductor-loop.exe --help
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
}
```

### Directory Setup

```powershell
# Create required directories
New-Item -ItemType Directory -Force -Path .\handoff, .\out | Out-Null

# Verify directories
Get-ChildItem -Directory | Where-Object { $_.Name -in @('handoff', 'out') } | 
    ForEach-Object { Write-Host "✓ Directory created: $($_.FullName)" -ForegroundColor Green }
```

### Test Data Preparation

```powershell
# Create a valid intent JSON file
$intentJson = @'
{
  "apiVersion": "network.nephio.io/v1alpha1",
  "kind": "NetworkIntent",
  "metadata": {
    "name": "scale-intent-001",
    "namespace": "default",
    "labels": {
      "app": "ran-du",
      "environment": "test"
    }
  },
  "spec": {
    "action": "scale",
    "target": "ran-du-01",
    "parameters": {
      "replicas": 3,
      "cpu": "2000m",
      "memory": "4Gi"
    },
    "priority": "high",
    "timestamp": "2025-01-17T10:00:00Z"
  }
}
'@

# Write test intent file
$intentJson | Out-File -FilePath ".\handoff\intent-001.json" -Encoding UTF8

# Create additional test files for duplicate detection
$intentJson | Out-File -FilePath ".\handoff\intent-002.json" -Encoding UTF8

# Create a different intent for processing
$intentJson2 = $intentJson -replace '"scale-intent-001"', '"scale-intent-002"' -replace '"replicas": 3', '"replicas": 5'
$intentJson2 | Out-File -FilePath ".\handoff\intent-003.json" -Encoding UTF8

Write-Host "✓ Test files created in .\handoff\" -ForegroundColor Green
```

### Run Conductor Loop

#### Basic Polling Mode (2s period)
```powershell
# Run with default 2-second polling
.\conductor-loop.exe --handoff .\handoff --out .\out --period 2s

# Expected initial output:
# [conductor-loop] 2025/01/17 10:00:00.000000 LOOP:START - Starting watcher on directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:00.000001 LOOP:SCAN - Scanning directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:00.000002 LOOP:PROCESS - Processing file: intent-001.json
# [conductor-loop] 2025/01/17 10:00:00.000003 LOOP:PROCESS - Processing file: intent-002.json (duplicate content)
# [conductor-loop] 2025/01/17 10:00:00.000004 LOOP:SKIP_DUP - File intent-002.json unchanged (SHA: abc123...), skipping
# [conductor-loop] 2025/01/17 10:00:00.000005 LOOP:PROCESS - Processing file: intent-003.json
# [conductor-loop] 2025/01/17 10:00:02.000000 LOOP:SCAN - Scanning directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:02.000001 LOOP:SKIP_DUP - File intent-001.json already processed (SHA: abc123...), skipping
# [conductor-loop] 2025/01/17 10:00:02.000002 LOOP:SKIP_DUP - File intent-003.json already processed (SHA: def456...), skipping
```

#### With Porch URL (HTTP POST mode)
```powershell
# Run with HTTP POST to porch-url
.\conductor-loop.exe --handoff .\handoff --porch-url http://localhost:8080/api/v1/intent --period 2s

# Expected output includes HTTP POST operations:
# [conductor-loop] 2025/01/17 10:00:00.000000 LOOP:PROCESS - POSTing to http://localhost:8080/api/v1/intent
```

#### Process Once and Exit
```powershell
# Process all files once and exit (no continuous polling)
.\conductor-loop.exe --handoff .\handoff --out .\out --once

# Expected output:
# [conductor-loop] 2025/01/17 10:00:00.000000 LOOP:SCAN - Scanning directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:00.000001 LOOP:PROCESS - Processing file: intent-001.json
# [conductor-loop] 2025/01/17 10:00:00.000002 LOOP:DONE - All files processed successfully
```

### Verify Output

```powershell
# Check generated YAML files
Get-ChildItem .\out\*.yaml | ForEach-Object {
    Write-Host "`n✓ Generated: $($_.Name)" -ForegroundColor Green
    Get-Content $_.FullName | Select-Object -First 10
}

# Check state file
if (Test-Path ".\handoff\.conductor-state.json") {
    Write-Host "`n✓ State file contents:" -ForegroundColor Green
    Get-Content ".\handoff\.conductor-state.json" | ConvertFrom-Json | ConvertTo-Json -Depth 10
}
```

### Test Duplicate Detection

```powershell
# Add a duplicate file (same content)
Copy-Item ".\handoff\intent-001.json" ".\handoff\intent-001-copy.json"

# Run conductor loop
.\conductor-loop.exe --handoff .\handoff --out .\out --period 2s

# Expected log showing duplicate detection:
# [conductor-loop] 2025/01/17 10:00:00.000000 LOOP:SCAN - Scanning directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:00.000001 LOOP:SKIP_DUP - File intent-001-copy.json unchanged (SHA: abc123...), skipping
```

### Test File Modification

```powershell
# Modify an existing file
$modified = Get-Content ".\handoff\intent-001.json" | ConvertFrom-Json
$modified.spec.parameters.replicas = 7
$modified | ConvertTo-Json -Depth 10 | Out-File ".\handoff\intent-001.json" -Encoding UTF8

# Run conductor loop
.\conductor-loop.exe --handoff .\handoff --out .\out --period 2s

# Expected log showing reprocessing:
# [conductor-loop] 2025/01/17 10:00:00.000000 LOOP:SCAN - Scanning directory: C:\...\handoff
# [conductor-loop] 2025/01/17 10:00:00.000001 LOOP:PROCESS - Processing file: intent-001.json (modified)
```

### Cleanup

```powershell
# Stop conductor loop (Ctrl+C or close window)

# Clean test data
Remove-Item -Path ".\handoff\intent-*.json" -ErrorAction SilentlyContinue
Remove-Item -Path ".\out\*.yaml" -ErrorAction SilentlyContinue
Remove-Item -Path ".\handoff\.conductor-state.json" -ErrorAction SilentlyContinue

Write-Host "✓ Cleanup complete" -ForegroundColor Green
```

### Expected Log Patterns

| Log Prefix | Meaning | Example |
|------------|---------|---------|
| `LOOP:START` | Conductor starting | `LOOP:START - Starting watcher on directory: C:\...\handoff` |
| `LOOP:SCAN` | Directory scan initiated | `LOOP:SCAN - Scanning directory: C:\...\handoff` |
| `LOOP:PROCESS` | File being processed | `LOOP:PROCESS - Processing file: intent-001.json` |
| `LOOP:SKIP_DUP` | Duplicate detected (same SHA) | `LOOP:SKIP_DUP - File intent-001.json unchanged (SHA: abc123...), skipping` |
| `LOOP:DONE` | Processing completed | `LOOP:DONE - Successfully processed intent-001.json` |
| `LOOP:ERROR` | Error occurred | `LOOP:ERROR - Failed to process intent-001.json: invalid JSON` |
| `LOOP:WARNING` | Non-critical issue | `LOOP:WARNING - Work queue full, skipping file intent-002.json` |

### Performance Testing

```powershell
# Generate multiple intent files for load testing
1..100 | ForEach-Object {
    $intent = @{
        apiVersion = "network.nephio.io/v1alpha1"
        kind = "NetworkIntent"
        metadata = @{
            name = "intent-$_"
            namespace = "default"
        }
        spec = @{
            action = "scale"
            target = "ran-du-$_"
            parameters = @{
                replicas = Get-Random -Minimum 1 -Maximum 10
            }
        }
    }
    $intent | ConvertTo-Json -Depth 10 | Out-File -FilePath ".\handoff\intent-$_.json" -Encoding UTF8
}

# Measure processing time
$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
.\conductor-loop.exe --handoff .\handoff --out .\out --once
$stopwatch.Stop()
Write-Host "Processed 100 files in $($stopwatch.Elapsed.TotalSeconds) seconds" -ForegroundColor Cyan
```

### Troubleshooting

```powershell
# Enable verbose logging
$env:LOG_LEVEL = "DEBUG"
.\conductor-loop.exe --handoff .\handoff --out .\out --period 2s

# Check for file locks
Get-Process | Where-Object { $_.Name -like "*conductor*" } | Stop-Process -Force

# Verify JSON validity
Get-ChildItem .\handoff\*.json | ForEach-Object {
    try {
        $null = Get-Content $_.FullName | ConvertFrom-Json
        Write-Host "✓ Valid JSON: $($_.Name)" -ForegroundColor Green
    } catch {
        Write-Host "✗ Invalid JSON: $($_.Name) - $_" -ForegroundColor Red
    }
}

# Monitor state file changes
Get-Content ".\handoff\.conductor-state.json" -Wait | ConvertFrom-Json
```

## Success Criteria Verification

✅ **Polling works**: Scans `.\handoff` directory every 2 seconds  
✅ **Duplicate detection**: SHA256-based deduplication via `.conductor-state.json`  
✅ **YAML output**: Generates `.\out\*.yaml` files for each unique intent  
✅ **Idempotent**: Subsequent runs show `LOOP:SKIP_DUP` for unchanged files  
✅ **Windows-safe**: Proper path handling for Windows file systems  
✅ **HTTP POST option**: Supports `--porch-url` for API integration
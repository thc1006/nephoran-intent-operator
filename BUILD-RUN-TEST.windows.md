<<<<<<< HEAD
<<<<<<< HEAD
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
=======
=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
# BUILD-RUN-TEST Guide for Windows

This guide covers building, running, and testing multiple components of the Nephoran Intent Operator on Windows.

## Components Covered

<<<<<<< HEAD
1. [Intent Ingest Service](#intent-ingest-service) - HTTP service for intent ingestion with LLM support
2. [Porch Direct CLI](#porch-direct-cli) - Intent-driven KRM package generator
3. [FCAPS Simulator](#fcaps-simulator) - VES event simulator with burst detection  
4. [Conductor Watch](#conductor-watch) - File watcher for intent processing
5. [Admission Webhook](#admission-webhook) - Kubernetes admission control

---

# Intent Ingest Service

## Overview

The `intent-ingest` service accepts intents via HTTP POST and saves them to a handoff directory for processing by other components. It supports both plain text (with rules or LLM parsing) and JSON input.

## Build

```powershell
# Build the service
go build -o intent-ingest.exe .\cmd\intent-ingest

# Create handoff directory
New-Item -ItemType Directory -Force -Path .\handoff | Out-Null
```

## Run

### Default Mode (Rules)

```powershell
# Run with default settings
.\intent-ingest.exe --addr :8080 --handoff .\handoff

# Or with environment variable
$env:MODE="rules"
.\intent-ingest.exe
```

### LLM Mode with Mock Provider

```powershell
# Using environment variables
$env:MODE="llm"
$env:PROVIDER="mock"
.\intent-ingest.exe --addr :8080 --handoff .\handoff

# Or using command-line flags
.\intent-ingest.exe --mode llm --provider mock --addr :8080 --handoff .\handoff
```

## Test Examples

### Health Check

```powershell
curl.exe -X GET http://localhost:8080/healthz
```

### Plain Text Intent (Rules Mode)

```powershell
# Scale intent
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale nf-sim to 5 in ns ran-a"

# Expected response:
# {
#   "status": "accepted",
#   "saved": "handoff\\intent-20250816T120000Z.json",
#   "preview": {
#     "intent_type": "scaling",
#     "target": "nf-sim",
#     "namespace": "ran-a",
#     "replicas": 5,
#     "source": "user"
#   }
# }
```

### JSON Intent

```powershell
# Send JSON directly
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: application/json" `
  -d '{\"intent_type\":\"scaling\",\"target\":\"api-gateway\",\"namespace\":\"production\",\"replicas\":10,\"source\":\"user\"}'
```

### LLM Mode Test

```powershell
# Start server in LLM mode
$env:MODE="llm"
$env:PROVIDER="mock"
.\intent-ingest.exe --addr :8080 --handoff .\handoff

# In another terminal
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale database to 3 in ns backend"

# Mock provider returns same result as rules mode
```

### Alternative Plain Text Formats

```powershell
# Simple scale (default namespace)
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale my-app to 3"

# Scale out
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale out web-server by 2 in ns production"

# Scale in
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale in cache by 1"
```

## Verify Output

```powershell
# Check handoff files were created
Get-ChildItem .\handoff\intent-*.json

# View a specific file
Get-Content .\handoff\intent-*.json | Select-Object -Last 1 | ConvertFrom-Json | ConvertTo-Json -Depth 3
```

## Test Script

```powershell
# test-intent.ps1
$baseUrl = "http://localhost:8080"

Write-Host "Testing intent-ingest service..." -ForegroundColor Cyan

# Health check
$health = Invoke-RestMethod -Uri "$baseUrl/healthz" -Method Get
Write-Host "Health: $health" -ForegroundColor Green

# Test plain text
$response = Invoke-RestMethod -Uri "$baseUrl/intent" -Method Post `
    -ContentType "text/plain" `
    -Body "scale test-app to 3 in ns testing"
Write-Host "Plain text response:" -ForegroundColor Yellow
$response | ConvertTo-Json -Depth 3

# Verify file was created
$savedFile = $response.saved
if (Test-Path $savedFile) {
    Write-Host "✓ File created: $savedFile" -ForegroundColor Green
} else {
    Write-Host "✗ File not found: $savedFile" -ForegroundColor Red
}

# Test JSON
$jsonBody = @{
    intent_type = "scaling"
    target = "json-app"
    namespace = "production"
    replicas = 5
    source = "test"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "$baseUrl/intent" -Method Post `
    -ContentType "application/json" `
    -Body $jsonBody
Write-Host "JSON response:" -ForegroundColor Yellow
$response | ConvertTo-Json -Depth 3
```

## Error Cases

```powershell
# Invalid text format
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "invalid text"
# Returns: HTTP 400

# Invalid JSON schema (replicas > 100)
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: application/json" `
  -d '{\"intent_type\":\"scaling\",\"target\":\"test\",\"namespace\":\"default\",\"replicas\":200}'
# Returns: HTTP 400 - schema validation failed

# Missing required field
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: application/json" `
  -d '{\"intent_type\":\"scaling\",\"target\":\"test\",\"replicas\":5}'
# Returns: HTTP 400 - missing namespace
```

## Environment Variables

- `MODE`: Intent parsing mode (`rules` or `llm`)
- `PROVIDER`: LLM provider name (`mock` for LLM mode)

Command-line flags override environment variables.
=======
1. [Porch Direct CLI](#porch-direct-cli) - Intent-driven KRM package generator
2. [FCAPS Simulator](#fcaps-simulator) - VES event simulator with burst detection  
3. [Conductor Watch](#conductor-watch) - File watcher for intent processing
4. [Admission Webhook](#admission-webhook) - Kubernetes admission control
>>>>>>> 6835433495e87288b95961af7173d866977175ff

---

# Porch Direct CLI

## Overview

The `porch-direct` CLI tool reads an intent JSON file and interacts with the Porch API to create/update KRM packages for network function scaling. It supports both dry-run mode (writes to `.\out\`) and live mode (calls Porch API).

## Prerequisites

- Windows 10/11 with PowerShell 5.1+
- Go 1.24+ installed
- Git for Windows

## Build Instructions

### PowerShell Commands

```powershell
# Clone and navigate to the repository
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-porch-direct

# Build the CLI tool
go build -o porch-direct.exe .\cmd\porch-direct

# Verify the build
.\porch-direct.exe --help
```

## Test Intent Files

### Create Sample Intent (examples\intent.json)

```powershell
# Create examples directory if it doesn't exist
New-Item -ItemType Directory -Force -Path .\examples | Out-Null

# Create a sample intent file
@'
{
  "intent_type": "scaling",
  "target": "nf-sim",
  "namespace": "ran-a",
  "replicas": 3,
  "reason": "Load increase detected",
  "source": "planner",
  "correlation_id": "test-001"
}
'@ | Out-File -FilePath .\examples\intent.json -Encoding UTF8
```

### Alternative Intent Examples

```powershell
# Minimal intent
@'
{
  "intent_type": "scaling",
  "target": "gnb",
  "namespace": "ran-sim",
  "replicas": 5
}
'@ | Out-File -FilePath .\examples\intent-minimal.json -Encoding UTF8

# Scale-down intent
@'
{
  "intent_type": "scaling",
  "target": "upf",
  "namespace": "core",
  "replicas": 1,
  "reason": "Night-time scale down"
}
'@ | Out-File -FilePath .\examples\intent-scaledown.json -Encoding UTF8
```

## Dry-Run Mode Testing

### Basic Dry-Run

```powershell
# Create output directory
New-Item -ItemType Directory -Force -Path .\out | Out-Null

# Run in dry-run mode (writes to filesystem)
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --dry-run

# Check generated files
<<<<<<< HEAD
Get-ChildItem .\out\packages\*
```

### Output Structure

The dry-run creates the following structure:
```
.\out\
  packages\
    nf-sim-scaling-<timestamp>\
      Kptfile
      README.md
      deployment-patch.yaml
```

### Verify Generated Content

```powershell
# View the generated deployment patch
Get-Content .\out\packages\nf-sim-scaling-*\deployment-patch.yaml

# View the Kptfile
Get-Content .\out\packages\nf-sim-scaling-*\Kptfile

# Check all generated files
Get-ChildItem -Recurse .\out\packages | Select-Object FullName
```

## Live Mode Testing (Against Porch API)

### Prerequisites for Live Mode

```powershell
# Set Porch API endpoint (adjust to your environment)
$env:PORCH_API_URL = "http://localhost:8080"

# Or use Kubernetes port-forward
kubectl port-forward -n porch-system svc/porch-server 8080:80
```

### Run Against Porch API

```powershell
# Without dry-run flag, it calls the Porch API
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --porch-url $env:PORCH_API_URL `
  --repo-name "my-packages" `
  --namespace "default"

# With verbose output
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --porch-url $env:PORCH_API_URL `
  --repo-name "my-packages" `
  --namespace "default" `
  --verbose
```

### Batch Processing

```powershell
# Process multiple intent files
Get-ChildItem .\examples\intent*.json | ForEach-Object {
    Write-Host "Processing $($_.Name)..." -ForegroundColor Cyan
    .\porch-direct.exe `
      --intent $_.FullName `
      --dry-run
}
```

## Error Testing

### Invalid Intent Files

```powershell
# Missing required field
@'
{
  "intent_type": "scaling",
  "target": "test-app"
}
'@ | Out-File -FilePath .\examples\invalid-intent.json -Encoding UTF8

# This should fail with validation error
.\porch-direct.exe --intent .\examples\invalid-intent.json --dry-run
```

### Invalid Replica Count

```powershell
# Replicas out of range
@'
{
  "intent_type": "scaling",
  "target": "test-app",
  "namespace": "default",
  "replicas": 200
}
'@ | Out-File -FilePath .\examples\invalid-replicas.json -Encoding UTF8

.\porch-direct.exe --intent .\examples\invalid-replicas.json --dry-run
# Expected: Error - replicas must be between 1 and 100
```

## Integration with Intent-Ingest

```powershell
# Step 1: Start intent-ingest service
Start-Process -FilePath ".\intent-ingest.exe" -ArgumentList "--addr :8080 --handoff .\handoff"

# Step 2: Send an intent
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale nf-sim to 5 in ns ran-a"

# Step 3: Process the saved intent with porch-direct
$latestIntent = Get-ChildItem .\handoff\intent-*.json | Sort-Object LastWriteTime -Descending | Select-Object -First 1
.\porch-direct.exe --intent $latestIntent.FullName --dry-run

# Step 4: Verify the generated package
Get-ChildItem .\out\packages\*
=======
Get-ChildItem -Path .\examples\packages\direct\nf-sim-scaling\
```

### Dry-Run with Porch Parameters

```powershell
# Run with full Porch parameters (still dry-run)
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --repo my-repo `
  --package nf-sim `
  --workspace ws-001 `
  --namespace ran-a `
  --porch http://localhost:9443 `
  --dry-run

# With Porch URL + dry-run, it writes structured output to .\out\
Get-ChildItem -Path .\out\ -Recurse
```

### Expected Dry-Run Output

When using `--porch` with `--dry-run`, the tool creates:

```
.\out\
├── porch-package-request.json     # Full Porch API request payload
├── overlays\
│   ├── deployment.yaml            # KRM Deployment overlay
│   └── configmap.yaml            # ConfigMap with intent
```

#### Sample porch-package-request.json

```json
{
  "repository": "my-repo",
  "package": "nf-sim",
  "workspace": "ws-001",
  "namespace": "ran-a",
  "intent": {
    "intent_type": "scaling",
    "target": "nf-sim",
    "namespace": "ran-a",
    "replicas": 3,
    "reason": "Load increase detected",
    "source": "planner",
    "correlation_id": "test-001"
  },
  "files": {
    "Kptfile": "...",
    "deployment.yaml": "...",
    "service.yaml": "...",
    "README.md": "..."
  }
}
```

#### Sample overlays\deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nf-sim
  namespace: default
spec:
  replicas: 3
```

#### Sample overlays\configmap.yaml

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nf-sim-intent
  namespace: default
data:
  intent.json: |
    {
      "intent_type": "scaling",
      "target": "nf-sim",
      "namespace": "ran-a",
      "replicas": 3,
      "reason": "Load increase detected",
      "source": "planner",
      "correlation_id": "test-001"
    }
```

## Live Mode Testing (Requires Porch Server)

### Start Local Porch Server (if available)

```powershell
# Port-forward to Porch server (if running in Kubernetes)
kubectl port-forward -n porch-system svc/porch-server 9443:443
```

### Live API Call

```powershell
# Create/update package via Porch API
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --repo gitops-repo `
  --package nf-sim-scaling `
  --workspace default `
  --namespace ran-a `
  --porch http://localhost:9443

# With auto-approval
.\porch-direct.exe `
  --intent .\examples\intent.json `
  --repo gitops-repo `
  --package nf-sim-scaling `
  --workspace default `
  --namespace ran-a `
  --porch http://localhost:9443 `
  --auto-approve
```

## Validation and Testing

### Run Unit Tests

```powershell
# Run tests for the CLI
go test .\cmd\porch-direct\...

# Run tests for the Porch client
go test .\pkg\porch\...

# Run all tests with coverage
go test -cover ./...
```

### Validate Intent JSON

```powershell
# Check if intent file is valid JSON
$intent = Get-Content .\examples\intent.json -Raw | ConvertFrom-Json
$intent | Format-List

# Validate required fields
if ($intent.intent_type -ne "scaling") {
    Write-Error "Invalid intent_type"
}
if ($intent.replicas -lt 1 -or $intent.replicas -gt 100) {
    Write-Error "Replicas out of range (1-100)"
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
```

---

# FCAPS Simulator

<<<<<<< HEAD
## Overview

The FCAPS simulator generates VES (VNF Event Streaming) events simulating network function metrics and events. It includes burst detection and various event patterns.

## Build

```powershell
# Build the simulator
go build -o fcaps-sim.exe .\cmd\fcaps-sim

# Verify build
.\fcaps-sim.exe --help
```

## Basic Usage

### Generate Single Event

```powershell
# Generate a single measurement event
.\fcaps-sim.exe `
  --mode single `
  --event-type measurement `
  --output .\events

# Check generated event
Get-Content .\events\*.json | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

### Continuous Event Generation

```powershell
# Generate events every 5 seconds
.\fcaps-sim.exe `
  --mode continuous `
  --interval 5s `
  --output .\events

# Stop with Ctrl+C
```

### Burst Mode

```powershell
# Generate burst of events (simulating high load)
.\fcaps-sim.exe `
  --mode burst `
  --burst-count 10 `
  --burst-interval 100ms `
  --output .\events

# List generated events
Get-ChildItem .\events\*.json | Select-Object Name, Length, LastWriteTime
```

## Event Types

### Measurement Events

```powershell
# CPU/Memory metrics
.\fcaps-sim.exe `
  --event-type measurement `
  --nf-name "gnb-001" `
  --mode single
```

### Fault Events

```powershell
# Simulated fault
.\fcaps-sim.exe `
  --event-type fault `
  --severity MAJOR `
  --nf-name "upf-002" `
  --mode single
```

### Notification Events

```powershell
# State change notification
.\fcaps-sim.exe `
  --event-type notification `
  --change-type "configuration" `
  --nf-name "amf-001" `
  --mode single
```

## Integration with Load Detection

```powershell
# Simulate load increase pattern
$script = @'
# Generate normal load
for ($i = 0; $i -lt 5; $i++) {
    & .\fcaps-sim.exe --mode single --event-type measurement --cpu-util 30
    Start-Sleep -Seconds 2
}

# Generate high load burst
Write-Host "Simulating high load..." -ForegroundColor Red
for ($i = 0; $i -lt 10; $i++) {
    & .\fcaps-sim.exe --mode single --event-type measurement --cpu-util 85
    Start-Sleep -Milliseconds 500
}

# Back to normal
Write-Host "Returning to normal..." -ForegroundColor Green
for ($i = 0; $i -lt 5; $i++) {
    & .\fcaps-sim.exe --mode single --event-type measurement --cpu-util 35
    Start-Sleep -Seconds 2
}
'@

$script | Out-File -FilePath .\simulate-load.ps1 -Encoding UTF8
.\simulate-load.ps1
```

## Output Analysis

```powershell
# Count events by type
$events = Get-ChildItem .\events\*.json | ForEach-Object {
    Get-Content $_ | ConvertFrom-Json
}

$events | Group-Object -Property {$_.event.commonEventHeader.domain} | 
    Select-Object Name, Count

# Find high CPU events
$highCpu = $events | Where-Object {
    $_.event.measurementFields.cpuUsageArray.percentUsage -gt 80
}

$highCpu | Select-Object @{
    Name='Time'; Expression={$_.event.commonEventHeader.startEpochMicrosec}
}, @{
    Name='NF'; Expression={$_.event.commonEventHeader.sourceName}
}, @{
    Name='CPU%'; Expression={$_.event.measurementFields.cpuUsageArray.percentUsage}
}
```
=======
## Quick Start

```powershell
# Build
go build .\cmd\fcaps-sim

# Run with local reducer (burst detection)
.\fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff .\handoff

# Check for generated intents
Get-ChildItem .\handoff\intent-*.json
```

## Overview

The FCAPS simulator (`fcaps-sim`) emits VES (VNF Event Stream) JSON events to a configurable HTTP collector and triggers scaling intents based on event patterns. The reducer (`fcaps-reducer`) detects event bursts and generates scaling intents.

## Prerequisites

- Go 1.21+ installed
- Git Bash or PowerShell
- Network access for HTTP communication

## Build Instructions

### 1. Build the FCAPS Simulator

```powershell
# From repository root
go build -o fcaps-sim.exe ./cmd/fcaps-sim
```

### 2. Build the FCAPS Reducer

```powershell
# From repository root
go build -o fcaps-reducer.exe ./cmd/fcaps-reducer
```

### 3. Build the Intent Ingest Service (Optional)

```powershell
# From repository root
go build -o intent-ingest.exe ./cmd/intent-ingest
```

## Run Instructions

### Component 1: FCAPS Reducer (VES Collector)

Start the reducer first to collect VES events and detect bursts:

```powershell
# Basic startup
./fcaps-reducer.exe

# With custom configuration
./fcaps-reducer.exe --listen :9999 --handoff ./handoff --burst 3 --window 60 --verbose
```

**Flags:**
- `--listen`: HTTP listen address (default: `:9999`)
- `--handoff`: Directory for intent files (default: `./handoff`)
- `--burst`: Number of critical events to trigger scaling (default: 3)
- `--window`: Time window in seconds for burst detection (default: 60)
- `--verbose`: Enable detailed logging

### Component 2: Intent Ingest Service (Optional)

If you want to process intents directly:

```powershell
# Start intent ingest service
./intent-ingest.exe --out ./handoff
```

### Component 3: FCAPS Simulator with Local Reducer

Run the simulator with integrated reducer:

```powershell
# Local reducer mode (no external dependencies)
./fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff ./handoff

# With verbose logging
./fcaps-sim.exe --delay 1 --burst 3 --nf-name nf-sim --out-handoff ./handoff --verbose

# Custom VES events
./fcaps-sim.exe `
  --input docs/contracts/fcaps.ves.examples.json `
  --delay 2 `
  --burst 2 `
  --nf-name nf-sim `
  --out-handoff ./handoff `
  --verbose
```

**Flags:**
- `--input`: Path to VES events JSON file (default: `docs/contracts/fcaps.ves.examples.json`)
- `--delay`: Seconds between events (default: 5)
- `--burst`: Critical event threshold for burst detection (default: 1)
- `--target`: Target deployment name (default: `nf-sim`)
- `--namespace`: Target namespace (default: `ran-a`)
- `--nf-name`: Network Function name (default: `nf-sim`)
- `--out-handoff`: Local handoff directory for reducer mode (enables local reducer)
- `--collector-url`: VES collector URL (optional, default: `http://localhost:9999/eventListener/v7`)
- `--intent-url`: Intent service URL (optional, default: `http://localhost:8080/intent`)
- `--verbose`: Enable verbose logging
>>>>>>> 6835433495e87288b95961af7173d866977175ff

---

# Conductor Watch

<<<<<<< HEAD
## Overview

`conductor-watch` monitors a directory for new intent files and processes them through the orchestration pipeline.

## Build

```powershell
# Build conductor-watch
go build -o conductor-watch.exe .\cmd\conductor-watch

# Verify
.\conductor-watch.exe --help
```

## Basic Operation

### Start Watching

```powershell
# Watch handoff directory
.\conductor-watch.exe `
  --watch-dir .\handoff `
  --output-dir .\processed `
  --poll-interval 2s

# With verbose logging
.\conductor-watch.exe `
  --watch-dir .\handoff `
  --output-dir .\processed `
  --poll-interval 2s `
  --verbose
```

### Test File Processing

```powershell
# In another terminal, create a test intent file
@'
{
  "intent_type": "scaling",
  "target": "test-app",
  "namespace": "default",
  "replicas": 3
}
'@ | Out-File -FilePath .\handoff\intent-test.json -Encoding UTF8

# Watch the logs in conductor-watch terminal
# File should be moved to processed directory
```

## Integration Pipeline

```powershell
# Terminal 1: Start intent-ingest
.\intent-ingest.exe --addr :8080 --handoff .\handoff

# Terminal 2: Start conductor-watch
.\conductor-watch.exe `
  --watch-dir .\handoff `
  --output-dir .\processed `
  --porch-command ".\porch-direct.exe"

# Terminal 3: Send test intent
curl.exe -X POST http://localhost:8080/intent `
  -H "Content-Type: text/plain" `
  -d "scale my-app to 5 in ns production"

# Verify processing
Get-ChildItem .\processed\*.json
Get-ChildItem .\out\packages\*
```

## Error Handling

```powershell
# Test with invalid intent
@'
{
  "invalid": "data"
}
'@ | Out-File -FilePath .\handoff\bad-intent.json -Encoding UTF8

# Check error directory
Get-ChildItem .\handoff\errors\*.json
Get-Content .\handoff\errors\*.error
=======
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

### 2. Run with HTTP POST Endpoint

```powershell
# Start a test HTTP server (in another terminal)
# Using Python:
python -m http.server 8080

# Run conductor-watch with POST URL
.\conductor-watch.exe --handoff .\handoff --post-url "http://localhost:8080/intents"
```

### 3. Run with Custom Debounce

```powershell
# Use 500ms debounce for slower file systems
.\conductor-watch.exe --handoff .\handoff --debounce-ms 500
>>>>>>> 6835433495e87288b95961af7173d866977175ff
```

---

# Admission Webhook

<<<<<<< HEAD
## Overview

The admission webhook validates and potentially mutates Kubernetes resources based on intent policies.

## Build

```powershell
# Build webhook
go build -o webhook.exe .\cmd\webhook-manager

# Generate certificates (for testing)
.\scripts\generate-certs.ps1
```

## Local Testing with Mock Server

```powershell
# Start webhook in test mode
.\webhook.exe `
  --port 8443 `
  --cert-file .\certs\tls.crt `
  --key-file .\certs\tls.key `
  --test-mode

# Test validation endpoint
$admissionReview = @{
    apiVersion = "admission.k8s.io/v1"
    kind = "AdmissionReview"
    request = @{
        uid = "test-123"
        kind = @{
            group = "apps"
            version = "v1"
            kind = "Deployment"
        }
        operation = "CREATE"
        object = @{
            metadata = @{
                name = "test-app"
                namespace = "default"
            }
            spec = @{
                replicas = 3
            }
        }
    }
} | ConvertTo-Json -Depth 10

# Send test request
Invoke-RestMethod `
  -Uri https://localhost:8443/validate `
  -Method Post `
  -ContentType "application/json" `
  -Body $admissionReview `
  -SkipCertificateCheck
```

## Deploy to Kubernetes

```powershell
# Build container (requires Docker Desktop)
docker build -t nephoran/webhook:latest -f Dockerfile.webhook .

# Deploy to cluster
kubectl apply -f deployments/webhook/

# Verify deployment
kubectl get pods -n nephoran-system
kubectl logs -n nephoran-system deployment/admission-webhook
```

## Test Webhook in Cluster

```powershell
# Create test deployment that triggers webhook
kubectl apply -f - @"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-scaling
  namespace: default
spec:
  replicas: 10
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
"@

# Check if webhook modified/rejected it
kubectl get deployment test-scaling -o yaml
=======
## Prerequisites
```powershell
# Verify Go 1.24
go version
# go version go1.24.5 windows/amd64

# Verify kubectl 
kubectl version --client
# Client Version: v1.29.0

# Verify kind installed
kind version
# kind v0.20.0 go1.20.4 windows/amd64
```

## Step 1: Install controller-gen
```powershell
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.15.0

# Verify installation
controller-gen --version
# Version: v0.15.0
```

## Step 2: Generate DeepCopy and CRDs
```powershell
# Generate DeepCopy methods
controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/intent/v1alpha1"

# Expected: Creates api/intent/v1alpha1/zz_generated.deepcopy.go

# Generate CRDs
controller-gen crd paths="./api/intent/v1alpha1" output:crd:artifacts:config="./deployments/crds"

# Expected: Creates deployments/crds/intent.nephoran.io_networkintents.yaml

# Verify files exist
ls api/intent/v1alpha1/zz_generated.deepcopy.go
ls deployments/crds/intent.nephoran.io_networkintents.yaml
```

## Step 3: Build webhook manager
```powershell
# Build the webhook manager binary
go build -o webhook-manager.exe ./cmd/webhook-manager

# Expected: webhook-manager.exe created

# Test binary
./webhook-manager.exe --help
# Expected output:
# Usage of webhook-manager.exe:
#   -cert-dir string
#   -health-probe-bind-address string (default ":8081")
#   -metrics-bind-address string (default ":8080")
#   -webhook-port int (default 9443)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
```

---

<<<<<<< HEAD
# Complete Integration Test

## Full Pipeline Test

```powershell
# setup-test.ps1
Write-Host "Setting up complete integration test..." -ForegroundColor Cyan

# Create directories
@("handoff", "processed", "out", "events", "logs") | ForEach-Object {
    New-Item -ItemType Directory -Force -Path $_ | Out-Null
}

# Start all components
$processes = @()

# 1. Intent Ingest Service
$processes += Start-Process -FilePath ".\intent-ingest.exe" `
  -ArgumentList "--addr :8080 --handoff .\handoff" `
  -RedirectStandardOutput ".\logs\intent-ingest.log" `
  -PassThru

Write-Host "Started intent-ingest on :8080" -ForegroundColor Green

# 2. Conductor Watch
$processes += Start-Process -FilePath ".\conductor-watch.exe" `
  -ArgumentList "--watch-dir .\handoff --output-dir .\processed" `
  -RedirectStandardOutput ".\logs\conductor-watch.log" `
  -PassThru

Write-Host "Started conductor-watch" -ForegroundColor Green

# 3. FCAPS Simulator
$processes += Start-Process -FilePath ".\fcaps-sim.exe" `
  -ArgumentList "--mode continuous --interval 5s --output .\events" `
  -RedirectStandardOutput ".\logs\fcaps-sim.log" `
  -PassThru

Write-Host "Started FCAPS simulator" -ForegroundColor Green

# Wait for services to start
Start-Sleep -Seconds 3

# Send test intents
Write-Host "`nSending test intents..." -ForegroundColor Yellow

$intents = @(
    "scale nf-sim to 5 in ns ran-a",
    "scale gnb to 3 in ns ran-b",
    "scale upf to 2 in ns core"
)

foreach ($intent in $intents) {
    curl.exe -X POST http://localhost:8080/intent `
      -H "Content-Type: text/plain" `
      -d $intent
    Write-Host "Sent: $intent" -ForegroundColor Cyan
    Start-Sleep -Seconds 1
}

# Monitor results
Write-Host "`nMonitoring processing..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Check results
Write-Host "`nResults:" -ForegroundColor Green
Write-Host "Handoff files:" -ForegroundColor Yellow
Get-ChildItem .\handoff\*.json | Select-Object Name

Write-Host "`nProcessed files:" -ForegroundColor Yellow
Get-ChildItem .\processed\*.json | Select-Object Name

Write-Host "`nGenerated packages:" -ForegroundColor Yellow
Get-ChildItem .\out\packages\* | Select-Object Name

Write-Host "`nVES Events:" -ForegroundColor Yellow
Get-ChildItem .\events\*.json | Select-Object -Last 5 Name

# Cleanup
Write-Host "`nPress any key to stop all services..." -ForegroundColor Red
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

$processes | ForEach-Object { Stop-Process -Id $_.Id -Force }
Write-Host "All services stopped" -ForegroundColor Green
```

## Run Complete Test

```powershell
# Execute the test script
.\setup-test.ps1

# Check logs for any errors
Get-Content .\logs\*.log | Select-String -Pattern "error" -Context 2
```

## Troubleshooting

### Port Already in Use

```powershell
# Find process using port 8080
netstat -ano | findstr :8080

# Kill process (replace PID with actual process ID)
Stop-Process -Id <PID> -Force
```

### Permission Issues

```powershell
# Run PowerShell as Administrator
# Or adjust folder permissions
icacls .\handoff /grant "${env:USERNAME}:(OI)(CI)F"
```

### Certificate Issues (Webhook)

```powershell
# Generate self-signed cert for testing
New-SelfSignedCertificate `
  -DnsName "localhost" `
  -CertStoreLocation "Cert:\CurrentUser\My" `
  -KeyExportPolicy Exportable `
  -KeySpec Signature `
  -KeyAlgorithm RSA `
  -KeyLength 2048
```
=======
# Common Troubleshooting
>>>>>>> 6835433495e87288b95961af7173d866977175ff

## Performance Testing

```powershell
<<<<<<< HEAD
# Load test intent-ingest
for ($i = 1; $i -le 100; $i++) {
    $intent = "scale app-$i to $($i % 10 + 1) in ns test"
    Invoke-RestMethod -Uri "http://localhost:8080/intent" `
      -Method Post `
      -ContentType "text/plain" `
      -Body $intent
    
    if ($i % 10 -eq 0) {
        Write-Host "Sent $i intents..." -ForegroundColor Cyan
    }
}

# Check processing rate
$processed = Get-ChildItem .\processed\*.json
Write-Host "Processed $($processed.Count) intents" -ForegroundColor Green
```

This completes the Windows build, run, and test guide for all major components of the Nephoran Intent Operator.
>>>>>>> origin/integrate/mvp
=======
# Generate multiple intents
1..10 | ForEach-Object {
    @{
        intent_type = "scaling"
        target = "nf-sim-$_"
        namespace = "test"
        replicas = $_
    } | ConvertTo-Json | Out-File -FilePath ".\examples\intent-$_.json"
}

# Measure dry-run performance
Measure-Command {
    1..10 | ForEach-Object {
        .\porch-direct.exe --intent ".\examples\intent-$_.json" --dry-run
    }
}

# Clean up test files
Remove-Item -Path .\examples\intent-*.json -Force
```

## Port Already in Use

```powershell
# Find process using port 9999
netstat -ano | findstr :9999

# Kill the process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

## Permission Issues

```powershell
# Run as Administrator or ensure write permissions for handoff directory
mkdir handoff
icacls handoff /grant Everyone:F
```

## Build Errors

```powershell
# Clean module cache
go clean -modcache

# Download dependencies
go mod download

# Verify dependencies
go mod verify
```

## Cleanup

```powershell
# Remove generated files
Remove-Item -Path .\out -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path .\examples\packages\direct -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item ./handoff/intent-*.json -Force
Remove-Item *.exe -Force
Remove-Item test-*.json -Force

# Delete test resources
kubectl delete networkintent --all -n default

# Delete webhook deployment
kubectl delete -k ./config/webhook/

# Delete namespace
kubectl delete namespace nephoran-system

# Delete kind cluster
kind delete cluster --name webhook-test
```

## Security Considerations

- Never commit Porch API credentials to the repository
- Use environment variables for sensitive configuration
- Validate all intent JSON inputs before processing
- Run with minimal Windows permissions
- Review generated KRM for security implications
>>>>>>> 6835433495e87288b95961af7173d866977175ff

# BUILD-RUN-TEST Guide for Windows

This guide covers building, running, and testing multiple components of the Nephoran Intent Operator on Windows.

## Components Covered

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
```

---

# FCAPS Simulator

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

---

# Conductor Watch

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
```

---

# Admission Webhook

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
```

---

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

## Performance Testing

```powershell
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
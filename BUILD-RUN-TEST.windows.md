# BUILD-RUN-TEST Guide for Windows

This guide covers building, running, and testing multiple components of the Nephoran Intent Operator on Windows.

## Components Covered

1. [Porch Direct CLI](#porch-direct-cli) - Intent-driven KRM package generator
2. [FCAPS Simulator](#fcaps-simulator) - VES event simulator with burst detection  
3. [Conductor Watch](#conductor-watch) - File watcher for intent processing
4. [Admission Webhook](#admission-webhook) - Kubernetes admission control

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
```

---

# FCAPS Simulator

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

---

# Conductor Watch

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
```

---

# Admission Webhook

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
```

---

# Common Troubleshooting

## Performance Testing

```powershell
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
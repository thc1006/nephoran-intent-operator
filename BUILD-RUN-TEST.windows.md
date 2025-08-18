# BUILD-RUN-TEST Guide for Windows (porch-direct)

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

### Clean Up

```powershell
# Remove generated files
Remove-Item -Path .\out -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path .\examples\packages\direct -Recurse -Force -ErrorAction SilentlyContinue

# Remove binary
Remove-Item -Path .\porch-direct.exe -Force -ErrorAction SilentlyContinue
```

## Troubleshooting

### Common Issues

1. **"go.mod not found" error**
   ```powershell
   # Ensure you're in the project root
   cd C:\Users\tingy\dev\_worktrees\nephoran\feat-porch-direct
   ```

2. **"intent validation failed" error**
   - Check that `intent_type` is "scaling"
   - Ensure `replicas` is between 1 and 100
   - Verify all required fields are present

3. **"failed to create output directory" error**
   ```powershell
   # Check permissions
   icacls .\out
   # Take ownership if needed
   takeown /f .\out /r
   ```

4. **Porch API connection error**
   - Verify Porch server is running
   - Check port-forward is active
   - Test with curl: `curl http://localhost:9443/api/v1/repositories`

### Debug Mode

```powershell
# Enable verbose logging
$env:DEBUG = "true"
.\porch-direct.exe --intent .\examples\intent.json --dry-run

# Check Windows file paths
[System.IO.Path]::GetFullPath(".\out")
```

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

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Test porch-direct Windows
on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: go build -o porch-direct.exe .\cmd\porch-direct
      - run: |
          New-Item -ItemType Directory -Force -Path .\out
          .\porch-direct.exe --intent .\examples\intent.json --dry-run
        shell: powershell
      - run: go test -cover ./...
```

## Security Considerations

- Never commit Porch API credentials to the repository
- Use environment variables for sensitive configuration
- Validate all intent JSON inputs before processing
- Run with minimal Windows permissions
- Review generated KRM for security implications
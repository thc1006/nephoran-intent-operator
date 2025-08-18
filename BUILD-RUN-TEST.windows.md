# Intent Ingest Service - Windows Build, Run & Test Guide

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
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

---

# Admission Webhook - Build, Run, and Test Guide for Windows

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

## Step 4: Create kind cluster
```powershell
# Create kind cluster
kind create cluster --name webhook-test

# Expected:
# Creating cluster "webhook-test" ...
# ✓ Ensuring node image (kindest/node:v1.27.3)
# ✓ Preparing nodes
# ✓ Writing configuration
# ✓ Starting control-plane
# ✓ Installing CNI
# ✓ Installing StorageClass
# Set kubectl context to "kind-webhook-test"

# Verify cluster
kubectl cluster-info --context kind-webhook-test
```

## Step 5: Build and load Docker image
```powershell
# Build Docker image
docker build -t nephoran/webhook-manager:latest -f- . @"
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-manager ./cmd/webhook-manager

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/webhook-manager .
ENTRYPOINT ["/webhook-manager"]
"@

# Expected: Successfully built <image-id>
# Successfully tagged nephoran/webhook-manager:latest

# Load image into kind
kind load docker-image nephoran/webhook-manager:latest --name webhook-test

# Expected: Image: "nephoran/webhook-manager:latest" with ID "sha256:..." not yet present on node "webhook-test-control-plane", loading...
```

## Step 6: Create namespace and deploy CRDs
```powershell
# Create namespace
kubectl create namespace nephoran-system

# Expected: namespace/nephoran-system created

# Apply CRDs
kubectl apply -f deployments/crds/

# Expected: customresourcedefinition.apiextensions.k8s.io/networkintents.intent.nephoran.io created
```

## Step 7: Generate self-signed certificates
```powershell
# Create certificate for webhook (using OpenSSL or generate in-cluster)
@"
apiVersion: v1
kind: Secret
metadata:
  name: webhook-server-cert
  namespace: nephoran-system
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURQekNDQWllZ0F3SUJBZ0lVS1VpRGdJdUdEemRUS2tRanJQK0ZXSktPSEhrd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0R6RU5NQXNHQTFVRUF3d0VkR1Z6ZERBZUZ3MHlNakEzTVRFd05qVXlNakJhRncwek1qQTNNRGd3TmpVeQpNakJhTUE4eERUQUxCZ05WQkFNTUJIUmxjM1F3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURFd0NpYnpGMFo0MjZSM0xxRXdNOGtkaHRWQ3lIUStQZUlRbzBKM3hEaHJ0NEl2bklIQzJQenBhaE0KZ3FGUnRMWlk0L3RYYVhqdWxWTlhSUFhFOGlNR2VKT2g2cm9odHlCNURoOTBqRzBLaE5SWUlQOTRrNWlMaFZOdwpaU1o3bENUK2JVQUxtTzFEVGJOcER6SFBXMVhwVXBRRnJqVUxjbHNKRERJdk0ybUxJUnB2VkViWHY1akE0WnJUClA1bDRzMzRiL1ZsZ01sOGsxRmhGc1VmeWJxV1dzWDRJWmZHaVEwRWxBZUZRUEhwMEtJOGNPbGNYeUcyS2tVcFYKVTRSYWJtNEVkTExmSGdOZG5rOTJudEZQdlh0SFhKejA3bXRBenNicUp1ZHU0RUpnMG80eEJPeHBvQllOeDJRTwpJN0RkS2Q0di9GOURnSWpBelUyOUczQnB5aTVaQWdNQkFBR2pVekJSTUIwR0ExVWREZ1FXQkJRbjdBeDlEa2pVCkNQa2l0Uld2SUdSL0pTNEpCekFmQmdOVkhTTUVHREFXZ0JRbjdBeDlEa2pVQ1BraXRSV3ZJR1IvSlM0SkJ6QVAKCQVVER1RRUJBd0lCQmpBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQWh3Q1ZESmVxOFl5Q3JMOCtMZXJxb3FGRwp1MEFyZXBwNWx1YkJzK3lTS0FNNDNGQzRHQjBDQ0draDR5NnhSTGRqVVB1S2tJaXJUTGswUzl5UG5EY3lOWFNtCkpxQnRaL2Z0UGR6NWo0TndPLzRid2xVeWw5MXBTelJtWTJpT2MyaUZLd05abjhqQmFKcFZzRnV0cnE2N0xJZmQKQjRiZEdad0xmWWh6KzJRdVJTTjd3a005c2kyaGpMNkJQTWVuM0JLbHRqQ2ZOcFJJN25mRmJEU0NXZGRBbEdDQwo3MEdZY3dQNGFQcDlwZXBwMkJqbU5ydVh0aEhLRmtIS3Y0TjZudEZZejYwQ3V5OGJMQjdXU3dqaGJHN08xL0prCmRJSktEakJQUEtBc2lROG5KenhrL0c1dnFwUVYvUjFqL0xrckR1akJqamNWdUNsNlRCZHRpemZlL3F1YW93PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBeE1Bb204eGRHZU51a2R5NmhNRFBKSFliVlFzaDBQajNpRUtOQ2Q4UTRhN2VDTDV5CkJ3dGo4NldvVElLaFViUzJXT1A3VjJsNDdwVlRWMFQxeFBJakJuaVRvZXE2SWJjZ2VRNGZkSXh0Q29UVVdDRC8KZUpPWWk0VlRjR1VtZTVRay9tMUFDNWp0UTAyemFROHh6MXRWNlZLVUJhNDFDM0piQ1F3eUx6TnBpeUVhYjFSRwoxNytZd09HYTB6K1plTE4rRy8xWllESmZKTlJZUmJGSDhtNmxsckYrQ0dYeG9rTkJKUUhoVUR4NmRDaVBIRHBYCkY4aHRpcEZLVlZPRVdtNXVCSFN5M3g0RFhaNVBkcDdSVDcxN1IxeWM5TzVyUU03RzZpYm5idUJDWU5LT01RVHMKYUFXRGN0a0RpT3czU25lTC94ZlE0Q0l3TTFOZFJ0d2Fjb3VXUUlEQVFBQkFvSUJBQk1HS1NTSGtlT2tQQjJvagpGRzhybUhDSUFhRnFlc3JJN2F5ZjRZVGJGOGdIOVZUTXdVcHJQYkJPOFJxMW81Ym1vOHVoSS9YY0F1Z0x2NjA1CjJIRWJxaUtiZ3lWdnNvV1FhY3pMdEY0cElwTEFhaU9JcVBNdCtwWm9YL29kMjJYVlp6aE05TVRIaTlReUNJdHQKWGlFMEtneGJHWGN0a0xFL1JGR2hsd3hqMjNGVEpuaXBrYUtjSUErcnpmQzZQRnp1amw5SzJGRzlWVVRDRGZ1ZQoyUHNYUHlvQU9mVGRYUHRsZDFUb3JQSG9MZWpWaGdkL0RrbEtzVjN2YzN6Q3g5MnVJQjlOQjdBTGNES3BiZGF1CjVCN0FiZGVWUjJCMXI0VnpYN2hLdVhEYXpLNzhLQUhjMHBGaUpONkVEVm9jd1Y1SDdJRXFFY0k0UmtXMmQ4eG4KczQ4R1VnRUNnWUVBOHNGSVFodmtpb05OVGMveG5RdFF6MDlIQ2hWSVIwUU5oOFBSYVZBU08yU0NhTHBEL1dCRApWcTlKdlowQkxIWkZJSUdPR2xMTGN5Y0R6MUxxMVRCN0xvOWdTQitQVEFMMzA3SkJHVzVYRyt3bUVkUzF4OWx2CjBVQXNRRkh5eDlRbUtGcTJZOXdXTjdXbHFKUU5UWGJBNWhVTDFUQzRjUHJjN1VRckplMTB5UUVDZ1lFQXo5T0cKTGdGbGFKMVNRNkJDbUJiQlNQNzBzUGJCMWxYWGgxb3g5Y1Z4VnZ0UG9DK0VBb0JhcEl2M0xkRGluZGdhcC9oTwo1cGp1cDlUR01YRGpsMEJJQ0VYUGd0MlBKcXRoZHQ3Sk91R0ZwK08xZ3V2L1l0OU5sczR0UFJzekFjcjAzdUg5CnR2TlN0RTBJdWRRQ0lJdmJJc0xwN1hNcklJRjNQOFRaVytIUmJoa0NnWUVBNFg0S1FQcGhQRGFhdEtKUGV3cHcKN2Y4c1JySUtHczNweGl1b1NqRVpQQ3pMZHA3d0FRL3YvMUtkMDdqdEJJN1lQZTZIM3h3VGtGZXBJNTZlL0dtTwo0ekZpTUFmTE0vdU5jU3VYT3ZiSUZCT3d1N1haRGF5aDNkcnEvaFI0d2llTlNXdXNLRkFldGRKN3pUa1VQNXlECkhyN2RTSzZ2ejlpQ3JySkd5STRyQVFFQ2dZRUF3Qm1mTGdTb1lwVkptMFg3TVpyL09vRnhLME5YYzBCQ0JxdGQKNy9ENHN6em9tMFN0Uk01aGExRW5KMmJMbFRyOVJRRkR0ZmhIT0R3dDZPaEZUWHhQbEVnRks1NnJYZ0xzd2JVaQpCT2k1U0hGZ0lOaUQzRlFnR0hYaW9aeG16SW1IMVBXL1lCNGhUR2VLOGJTdHZLNTFJa3BSVzg0OGNGZzJodHVUClU1bjU3WWtDZ1lBa3ZJQkwwNFVIcTdRZ1ZJRE84bWpWOVJBZTRBTHRjeWtIaHFrbHBhaDVnbFFBOElEL2liTUsKN0dBNVZWUVJvQ3h1U3o1UU1xdktNdFBuMXRFSTRsT25OZ0lnV0ZQOXFJOGlRM1Zpb3I5cEFDUDBaQmxhS1JOYQowdlBUdXpuV1dPaU9xZXhQVnRpWkJzRGRJQ3UrRmhXUjREd1l5akxGZ0pFaFJzSk9PVzJaYWc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
"@ | kubectl apply -f -

# Expected: secret/webhook-server-cert created
```

## Step 8: Deploy webhook deployment
```powershell
# Create deployment.yaml
@"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webhook-manager
  namespace: nephoran-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: webhook-manager
  template:
    metadata:
      labels:
        app: webhook-manager
    spec:
      containers:
      - name: webhook
        image: nephoran/webhook-manager:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
        volumeMounts:
        - name: webhook-certs
          mountPath: /tmp/k8s-webhook-server/serving-certs
          readOnly: true
        env:
        - name: WEBHOOK_CERT_DIR
          value: /tmp/k8s-webhook-server/serving-certs
      volumes:
      - name: webhook-certs
        secret:
          secretName: webhook-server-cert
"@ | Set-Content config/webhook/deployment.yaml

# Deploy webhook using kustomize
kubectl apply -k ./config/webhook/

# Expected:
# service/webhook-service created
# deployment.apps/webhook-manager created
# mutatingwebhookconfiguration.admissionregistration.k8s.io/networkintent-mutating-webhook created
# validatingwebhookconfiguration.admissionregistration.k8s.io/networkintent-validating-webhook created

# Verify deployment
kubectl get pods -n nephoran-system

# Expected:
# NAME                              READY   STATUS    RESTARTS   AGE
# webhook-manager-xxxxxxxxx-xxxxx   1/1     Running   0          10s
```

## Step 9: Test valid NetworkIntent (should succeed)
```powershell
# Apply valid NetworkIntent
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: valid-intent
  namespace: default
spec:
  intentType: scaling
  target: nf-sim
  namespace: default
  replicas: 3
"@

# Expected: networkintent.intent.nephoran.io/valid-intent created

# Verify defaulting worked (source should be "user")
kubectl get networkintent valid-intent -o jsonpath='{.spec.source}'

# Expected output: user
```

## Step 10: Test invalid NetworkIntent (should be rejected)
```powershell
# Test with negative replicas
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-replicas
  namespace: default
spec:
  intentType: scaling
  target: nf-sim
  namespace: default
  replicas: -1
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.replicas): invalid value: -1, Details: must be >= 0

# Test with empty target
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-target
  namespace: default
spec:
  intentType: scaling
  target: ""
  namespace: default
  replicas: 3
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.target): invalid value: , Details: must be non-empty

# Test with invalid intentType
kubectl apply -f - @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-type
  namespace: default
spec:
  intentType: provisioning
  target: nf-sim
  namespace: default
  replicas: 3
"@

# Expected error:
# error validating data: ValidationError(NetworkIntent.spec.intentType): invalid value: provisioning, Details: only 'scaling' supported
```

## Cleanup
```powershell
# Delete test resources
kubectl delete networkintent --all -n default

# Delete webhook deployment
kubectl delete -k ./config/webhook/

# Delete namespace
kubectl delete namespace nephoran-system

# Delete kind cluster
kind delete cluster --name webhook-test
```

## Verification Summary
✅ **Build successful**: `webhook-manager.exe` created
✅ **CRDs generated**: `deployments/crds/intent.nephoran.io_networkintents.yaml`
✅ **DeepCopy generated**: `api/intent/v1alpha1/zz_generated.deepcopy.go`
✅ **Deployment successful**: Webhook pod running in `nephoran-system`
✅ **Defaulting works**: Empty source field defaulted to "user"
✅ **Validation works**: Invalid CRs rejected with clear error messages
  - Negative replicas: "must be >= 0"
  - Empty target: "must be non-empty"
  - Invalid intentType: "only 'scaling' supported"

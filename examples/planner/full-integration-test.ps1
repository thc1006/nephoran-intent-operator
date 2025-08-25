#!/usr/bin/env pwsh
# Full Integration Test: Planner → Intent → Conductor → NetworkIntent CRD

Write-Host "`n=== Nephoran Planner Full Integration Test ===" -ForegroundColor Cyan
Write-Host "Testing: KPM Metrics → Planner → Intent → Conductor → CRD" -ForegroundColor Gray
Write-Host ""

# Configuration
$CONDUCTOR_URL = "http://localhost:8080/intent"
$HANDOFF_DIR = "./handoff"

# Function to check service
function Test-Service {
    param($Url, $ServiceName)
    try {
        $response = Invoke-WebRequest -Uri $Url -Method Head -TimeoutSec 2 -ErrorAction Stop
        Write-Host "✓ $ServiceName is running" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "✗ $ServiceName is not responding at $Url" -ForegroundColor Red
        return $false
    }
}

# Step 1: Check conductor service
Write-Host "1. Checking conductor service..." -ForegroundColor Yellow
$conductorRunning = Test-Service -Url "http://localhost:8080" -ServiceName "Conductor"

if (-not $conductorRunning) {
    Write-Host "   Starting conductor is recommended. Run in separate terminal:" -ForegroundColor Yellow
    Write-Host "   cd ../feat-ingest-tests" -ForegroundColor Gray
    Write-Host "   go run ./cmd/intent-ingest" -ForegroundColor Gray
    Write-Host ""
}

# Step 2: Create test intent
Write-Host "2. Creating test intent..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path $HANDOFF_DIR | Out-Null

$intent = @{
    intent_type = "scaling"
    target = "e2-node-001"
    namespace = "default"
    replicas = 3
    reason = "High load detected: PRB=92%, P95 Latency=135ms"
    source = "planner"
    correlation_id = "test-$(Get-Date -Format 'yyyyMMddHHmmss')"
}

$intentJson = $intent | ConvertTo-Json -Depth 10
$intentFile = "$HANDOFF_DIR/intent-integration-test.json"
$intentJson | Set-Content $intentFile

Write-Host "   Intent created: $intentFile" -ForegroundColor Gray
Write-Host "   Content:" -ForegroundColor Gray
$intent | Format-Table | Out-String | Write-Host

# Step 3: POST intent to conductor
if ($conductorRunning) {
    Write-Host "3. POSTing intent to conductor..." -ForegroundColor Yellow
    
    try {
        $response = Invoke-RestMethod -Uri $CONDUCTOR_URL `
            -Method Post `
            -ContentType "application/json" `
            -Body $intentJson
        
        Write-Host "   ✓ Intent accepted by conductor" -ForegroundColor Green
        Write-Host "   Response:" -ForegroundColor Gray
        $response | ConvertTo-Json -Depth 10 | Write-Host
        
        if ($response.saved) {
            Write-Host "   Intent saved to: $($response.saved)" -ForegroundColor Gray
        }
        
        # Step 4: Check for NetworkIntent CRD (if kubectl available)
        Write-Host ""
        Write-Host "4. Checking for NetworkIntent CRD..." -ForegroundColor Yellow
        
        if (Get-Command kubectl -ErrorAction SilentlyContinue) {
            Start-Sleep -Seconds 2
            $crds = kubectl get networkintents -o json 2>$null | ConvertFrom-Json
            
            if ($crds.items.Count -gt 0) {
                Write-Host "   ✓ NetworkIntent CRDs found:" -ForegroundColor Green
                foreach ($crd in $crds.items) {
                    Write-Host "     - $($crd.metadata.name): replicas=$($crd.spec.replicas)" -ForegroundColor Gray
                }
            } else {
                Write-Host "   No NetworkIntent CRDs found (may need to wait or conductor may not be connected to K8s)" -ForegroundColor Yellow
            }
        } else {
            Write-Host "   kubectl not available - skipping CRD check" -ForegroundColor Yellow
        }
        
    } catch {
        Write-Host "   ✗ Failed to POST intent: $_" -ForegroundColor Red
    }
} else {
    Write-Host "3. Skipping POST (conductor not running)" -ForegroundColor Yellow
    Write-Host "   To test the full flow, start the conductor:" -ForegroundColor Gray
    Write-Host "   cd ../feat-ingest-tests && go run ./cmd/intent-ingest" -ForegroundColor Gray
}

# Step 5: Test with curl command
Write-Host ""
Write-Host "5. Alternative: Test with curl..." -ForegroundColor Yellow
Write-Host "   You can also POST the intent using curl:" -ForegroundColor Gray
Write-Host "   curl -X POST $CONDUCTOR_URL -H 'Content-Type: application/json' -d @$intentFile" -ForegroundColor Cyan

# Step 6: Summary
Write-Host ""
Write-Host "=== Test Summary ===" -ForegroundColor Cyan
Write-Host "✓ Intent file created: $intentFile" -ForegroundColor Green

if ($conductorRunning -and $response) {
    Write-Host "✓ Intent accepted by conductor" -ForegroundColor Green
    Write-Host "✓ Intent saved for processing" -ForegroundColor Green
} else {
    Write-Host "⚠ Conductor not tested (service not running)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Monitor conductor logs for processing" -ForegroundColor Gray
Write-Host "2. Check NetworkIntent CRDs: kubectl get networkintents" -ForegroundColor Gray
Write-Host "3. Verify Porch packages: kubectl get packagerevisions" -ForegroundColor Gray
Write-Host "4. Check deployment scaling: kubectl get deployments -w" -ForegroundColor Gray

Write-Host ""
Write-Host "Full MVP Flow:" -ForegroundColor Cyan
Write-Host "KPM Metrics → Planner → Intent JSON → Conductor → NetworkIntent CRD → Porch → Deployment Scale" -ForegroundColor Gray
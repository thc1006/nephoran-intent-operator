#!/usr/bin/env pwsh
# Quick test for planner with simulation mode

Write-Host "`n=== Quick Planner Test ===" -ForegroundColor Cyan

# Setup
$HANDOFF_DIR = "./handoff"
New-Item -ItemType Directory -Force -Path $HANDOFF_DIR | Out-Null
Remove-Item "$HANDOFF_DIR/intent-*.json" -Force -ErrorAction SilentlyContinue

# Create high-load sample data
$sampleData = @{
    timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    node_id = "e2-node-001"
    prb_utilization = 0.92
    p95_latency = 135.0
    active_ues = 200
    current_replicas = 2
} | ConvertTo-Json

$sampleData | Set-Content "examples/planner/kpm-sample.json"

# Build planner
Write-Host "Building planner..." -ForegroundColor Yellow
go build -o planner.exe ./planner/cmd/planner
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Run planner in simulation mode (single evaluation)
Write-Host "Running planner in simulation mode..." -ForegroundColor Yellow
$env:PLANNER_SIM_MODE = "true"
$env:PLANNER_OUTPUT_DIR = $HANDOFF_DIR

# Start planner and let it run for one cycle
$proc = Start-Process -FilePath ".\planner.exe" `
    -ArgumentList "-config", "planner/config/config.yaml" `
    -PassThru -WindowStyle Hidden

Write-Host "Waiting for planner to process (35 seconds)..." -ForegroundColor Yellow
Start-Sleep -Seconds 35

# Check for intent
$intentFile = Get-ChildItem "$HANDOFF_DIR/intent-*.json" -ErrorAction SilentlyContinue | 
    Sort-Object LastWriteTime -Descending | Select-Object -First 1

if ($intentFile) {
    Write-Host "`n✓ Intent generated successfully!" -ForegroundColor Green
    Write-Host "Content:" -ForegroundColor Cyan
    Get-Content $intentFile.FullName | ConvertFrom-Json | ConvertTo-Json -Depth 10
} else {
    Write-Host "`n✗ No intent generated" -ForegroundColor Red
    Write-Host "Note: The rule engine requires 3 data points in the evaluation window." -ForegroundColor Yellow
    Write-Host "For testing, you may need to:" -ForegroundColor Yellow
    Write-Host "  1. Reduce DataPoints requirement in engine.go" -ForegroundColor Gray
    Write-Host "  2. Or run multiple polling cycles to accumulate history" -ForegroundColor Gray
}

# Cleanup
Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue

Write-Host "`nTest complete!" -ForegroundColor Cyan
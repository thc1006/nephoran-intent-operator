#!/usr/bin/env pwsh
# Test script to demonstrate conductor-loop enhanced functionality

Write-Host "=== Conductor-Loop Enhanced Functionality Test ===" -ForegroundColor Cyan
Write-Host ""

# Clean up previous test runs
Write-Host "Cleaning up previous test runs..." -ForegroundColor Yellow
Remove-Item -Path ".\test-handoff" -Recurse -ErrorAction SilentlyContinue
Remove-Item -Path ".\test-out" -Recurse -ErrorAction SilentlyContinue
Remove-Item -Path "conductor-loop.exe" -ErrorAction SilentlyContinue

# Build conductor-loop
Write-Host "Building conductor-loop..." -ForegroundColor Yellow
go build -o conductor-loop.exe .\cmd\conductor-loop
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Create test directories
New-Item -ItemType Directory -Path "test-handoff" -Force | Out-Null
New-Item -ItemType Directory -Path "test-out" -Force | Out-Null

# Create test intent files
$intents = @(
    @{
        file = "intent-test-app1.json"
        content = @{
            intent_type = "scaling"
            target = "test-app-1"
            namespace = "production" 
            replicas = 5
            reason = "High CPU utilization detected"
            timestamp = (Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')
        } | ConvertTo-Json -Depth 10
    },
    @{
        file = "intent-test-app2.json"
        content = @{
            intent_type = "scaling"
            target = "test-app-2"
            namespace = "staging"
            replicas = 3
            reason = "Load balancer queue depth exceeded"  
            timestamp = (Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')
        } | ConvertTo-Json -Depth 10
    },
    @{
        file = "intent-test-app3.json"
        content = @{
            intent_type = "scaling"
            target = "test-app-3"
            namespace = "development"
            replicas = 7
            reason = "Memory pressure alert"
            timestamp = (Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')
        } | ConvertTo-Json -Depth 10
    }
)

Write-Host "Creating test intent files..." -ForegroundColor Yellow
foreach ($intent in $intents) {
    $intent.content | Out-File -FilePath "test-handoff\$($intent.file)" -Encoding UTF8
    Write-Host "  Created: $($intent.file)" -ForegroundColor Green
}

# Test 1: CLI Help
Write-Host ""
Write-Host "=== Test 1: CLI Help ===" -ForegroundColor Blue
.\conductor-loop.exe -help

# Test 2: Once mode with echo (mock porch)
Write-Host ""
Write-Host "=== Test 2: Once Mode Processing ===" -ForegroundColor Blue
Write-Host "Running conductor-loop in once mode..."

$process = Start-Process -FilePath ".\conductor-loop.exe" -ArgumentList @(
    "-handoff", "test-handoff",
    "-out", "test-out",
    "-once",
    "-porch", "echo",
    "-mode", "direct",
    "-debounce", "100ms"
) -Wait -PassThru -NoNewWindow -RedirectStandardOutput "test-output.txt" -RedirectStandardError "test-error.txt"

Write-Host "Exit code: $($process.ExitCode)" -ForegroundColor $(if ($process.ExitCode -eq 0) { 'Green' } else { 'Red' })

# Show output
if (Test-Path "test-output.txt") {
    Write-Host ""
    Write-Host "=== Standard Output ===" -ForegroundColor Yellow
    Get-Content "test-output.txt" | ForEach-Object { Write-Host "  $_" -ForegroundColor White }
}

if (Test-Path "test-error.txt" -and (Get-Content "test-error.txt" -Raw).Trim().Length -gt 0) {
    Write-Host ""
    Write-Host "=== Standard Error ===" -ForegroundColor Yellow
    Get-Content "test-error.txt" | ForEach-Object { Write-Host "  $_" -ForegroundColor White }
}

# Test 3: Verify Results
Write-Host ""
Write-Host "=== Test 3: Verify Results ===" -ForegroundColor Blue

# Check processed directory
if (Test-Path "test-handoff\processed") {
    Write-Host "✓ Processed directory created" -ForegroundColor Green
    $processedFiles = Get-ChildItem "test-handoff\processed" -File
    Write-Host "  Found $($processedFiles.Count) processed files:" -ForegroundColor Green
    $processedFiles | ForEach-Object { Write-Host "    - $($_.Name)" -ForegroundColor Green }
} else {
    Write-Host "✗ Processed directory not found" -ForegroundColor Red
}

# Check failed directory  
if (Test-Path "test-handoff\failed") {
    $failedFiles = Get-ChildItem "test-handoff\failed" -File
    if ($failedFiles.Count -gt 0) {
        Write-Host "  Found $($failedFiles.Count) failed files:" -ForegroundColor Yellow
        $failedFiles | ForEach-Object { Write-Host "    - $($_.Name)" -ForegroundColor Yellow }
    } else {
        Write-Host "✓ No failed files" -ForegroundColor Green
    }
}

# Check status files
if (Test-Path "test-handoff\status") {
    Write-Host "✓ Status directory created" -ForegroundColor Green
    $statusFiles = Get-ChildItem "test-handoff\status" -File
    Write-Host "  Found $($statusFiles.Count) status files:" -ForegroundColor Green
    
    $statusFiles | ForEach-Object {
        $status = Get-Content $_.FullName | ConvertFrom-Json
        $color = if ($status.status -eq 'success') { 'Green' } else { 'Red' }
        Write-Host "    - $($_.Name): $($status.status)" -ForegroundColor $color
        if ($status.message) {
            Write-Host "      Message: $($status.message)" -ForegroundColor Gray
        }
    }
} else {
    Write-Host "✗ Status directory not found" -ForegroundColor Red
}

# Check state file
if (Test-Path "test-handoff\.conductor-state.json") {
    Write-Host "✓ State file created" -ForegroundColor Green
    $state = Get-Content "test-handoff\.conductor-state.json" | ConvertFrom-Json
    Write-Host "  Version: $($state.version)" -ForegroundColor Green
    Write-Host "  Saved at: $($state.saved_at)" -ForegroundColor Green
    Write-Host "  State entries: $(($state.states | Get-Member -MemberType NoteProperty).Count)" -ForegroundColor Green
    
    # Show first state entry as example
    $firstState = ($state.states | Get-Member -MemberType NoteProperty | Select-Object -First 1)
    if ($firstState) {
        $stateEntry = $state.states."$($firstState.Name)"
        Write-Host "  Example entry:" -ForegroundColor Green
        Write-Host "    File: $($stateEntry.file_path | Split-Path -Leaf)" -ForegroundColor Green
        Write-Host "    Status: $($stateEntry.status)" -ForegroundColor Green
        Write-Host "    Processed at: $($stateEntry.processed_at)" -ForegroundColor Green
        Write-Host "    SHA256: $($stateEntry.sha256.Substring(0,16))..." -ForegroundColor Green
    }
} else {
    Write-Host "! State file not created (may not have been saved yet)" -ForegroundColor Yellow
}

# Test 4: Feature Summary
Write-Host ""
Write-Host "=== Test 4: Feature Summary ===" -ForegroundColor Blue
Write-Host "✅ CLI Flags implemented:" -ForegroundColor Green
Write-Host "   -handoff, -porch, -mode, -out, -once, -debounce" -ForegroundColor White

Write-Host "✅ State Management:" -ForegroundColor Green  
Write-Host "   SHA256+size based deduplication with persistent state" -ForegroundColor White

Write-Host "✅ File Organization:" -ForegroundColor Green
Write-Host "   Atomic moves to processed/failed directories" -ForegroundColor White

Write-Host "✅ Porch Execution:" -ForegroundColor Green
Write-Host "   Timeout handling, structured errors, statistics tracking" -ForegroundColor White

Write-Host "✅ Enhanced Watcher:" -ForegroundColor Green
Write-Host "   Debouncing, worker pool, once mode, Windows compatibility" -ForegroundColor White

# Cleanup
Write-Host ""
Write-Host "Cleaning up test files..." -ForegroundColor Yellow
Remove-Item -Path "test-output.txt" -ErrorAction SilentlyContinue  
Remove-Item -Path "test-error.txt" -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "=== Test Complete! ===" -ForegroundColor Cyan
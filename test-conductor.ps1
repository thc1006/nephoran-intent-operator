#!/usr/bin/env pwsh

# Test script for conductor-loop

Write-Host "Testing conductor-loop enhanced implementation..."

# Build conductor-loop
Write-Host "Building conductor-loop..."
go build -o conductor-loop.exe ./cmd/conductor-loop
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# Create test intent file
$testIntent = @"
{
  "intent_type": "scaling",
  "target": "test-app",
  "namespace": "test-namespace",
  "replicas": 5,
  "reason": "Testing conductor-loop",
  "timestamp": "$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
}
"@

# Clean up any previous test
Remove-Item -Path ".\handoff\intent-test-*.json" -ErrorAction SilentlyContinue
Remove-Item -Path ".\handoff\processed" -Recurse -ErrorAction SilentlyContinue
Remove-Item -Path ".\handoff\failed" -Recurse -ErrorAction SilentlyContinue
Remove-Item -Path ".\handoff\.conductor-state.json" -ErrorAction SilentlyContinue

# Create test intent file
$testFile = ".\handoff\intent-test-$(Get-Date -Format 'yyyyMMddTHHmmss').json"
$testIntent | Out-File -FilePath $testFile -Encoding utf8

Write-Host "Created test file: $testFile"
Write-Host "Contents:"
Get-Content $testFile

# Test 1: Once mode with mock porch
Write-Host "`nTest 1: Running conductor-loop in once mode with mock porch..."
Start-Process -FilePath ".\conductor-loop.exe" -ArgumentList @(
    "-once",
    "-porch", "echo",
    "-mode", "direct",
    "-handoff", ".\handoff",
    "-out", ".\out"
) -Wait -NoNewWindow

# Check results
Write-Host "`nChecking results..."
if (Test-Path ".\handoff\processed") {
    Write-Host "Processed directory created: ✓" -ForegroundColor Green
    Get-ChildItem ".\handoff\processed" | ForEach-Object { 
        Write-Host "  Processed file: $($_.Name)" -ForegroundColor Green
    }
} else {
    Write-Host "Processed directory not found: ✗" -ForegroundColor Red
}

if (Test-Path ".\handoff\status") {
    Write-Host "Status directory created: ✓" -ForegroundColor Green
    Get-ChildItem ".\handoff\status\*.status" | ForEach-Object { 
        Write-Host "  Status file: $($_.Name)" -ForegroundColor Green
        $status = Get-Content $_.FullName | ConvertFrom-Json
        Write-Host "    Status: $($status.status)" -ForegroundColor $(if ($status.status -eq 'success') { 'Green' } else { 'Red' })
    }
} else {
    Write-Host "Status directory not found: ✗" -ForegroundColor Red
}

if (Test-Path ".\handoff\.conductor-state.json") {
    Write-Host "State file created: ✓" -ForegroundColor Green
    $state = Get-Content ".\handoff\.conductor-state.json" | ConvertFrom-Json
    Write-Host "  State entries: $($state.states.Count)" -ForegroundColor Green
} else {
    Write-Host "State file not found: ✗" -ForegroundColor Red
}

Write-Host "`nTest completed!" -ForegroundColor Cyan
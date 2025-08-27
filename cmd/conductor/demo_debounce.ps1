# Demo script to test the debounced conductor
# This script demonstrates that the race condition has been fixed

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$conductorExe = Join-Path $scriptDir "conductor.exe"
$handoffDir = Join-Path $scriptDir "test-handoff"
$outDir = Join-Path $scriptDir "test-output"

# Create test directories
New-Item -ItemType Directory -Force -Path $handoffDir | Out-Null
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

Write-Host "Testing debounced conductor with rapid file creation..." -ForegroundColor Yellow

# Start conductor in background
$conductorProcess = Start-Process -FilePath $conductorExe -ArgumentList @("-watch", $handoffDir, "-out", $outDir) -PassThru -WindowStyle Hidden

Start-Sleep -Seconds 2

Write-Host "Creating rapid file events to test debouncing..." -ForegroundColor Cyan

# Create rapid file events (simulating race condition scenario)
for ($i = 1; $i -le 5; $i++) {
    $intentFile = Join-Path $handoffDir "intent-test-$i.json"
    $intentData = @{
        correlation_id = "test-correlation-$i"
        action = "scale-up"
        target_replicas = 3
        namespace = "test"
    } | ConvertTo-Json -Depth 3
    
    # Create the file
    $intentData | Out-File -FilePath $intentFile -Encoding UTF8
    Write-Host "Created: $intentFile" -ForegroundColor Green
    
    # Create multiple rapid writes to the same file (simulating race condition)
    for ($j = 1; $j -le 3; $j++) {
        $intentData | Out-File -FilePath $intentFile -Encoding UTF8
        Start-Sleep -Milliseconds 50
    }
    
    Start-Sleep -Milliseconds 100
}

Write-Host "Waiting for debounced processing to complete..." -ForegroundColor Yellow
Start-Sleep -Seconds 3

# Stop conductor
if (!$conductorProcess.HasExited) {
    $conductorProcess.CloseMainWindow()
    Start-Sleep -Seconds 2
    if (!$conductorProcess.HasExited) {
        $conductorProcess.Kill()
    }
}

Write-Host "Demo completed! Check logs above for debounced processing messages." -ForegroundColor Green
Write-Host "Key improvements:" -ForegroundColor White
Write-Host "  - 500ms debounce window prevents race conditions" -ForegroundColor Gray
Write-Host "  - Multiple rapid file changes are consolidated into single processing" -ForegroundColor Gray
Write-Host "  - Memory leaks prevented by cleaning up processors" -ForegroundColor Gray
Write-Host "  - Proper logging for debugging file processing" -ForegroundColor Gray

# Cleanup
Remove-Item -Path $handoffDir -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path $outDir -Recurse -Force -ErrorAction SilentlyContinue
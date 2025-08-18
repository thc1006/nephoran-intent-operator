# verify-conductor-watch.ps1
# Quick verification script for conductor-watch

Write-Host "`n=== Conductor Watch Verification ===" -ForegroundColor Cyan

# 1. Build
Write-Host "`n[1/4] Building conductor-watch..." -ForegroundColor Yellow
go build -o conductor-watch.exe ./cmd/conductor-watch
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build successful" -ForegroundColor Green
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}

# 2. Run tests
Write-Host "`n[2/4] Running tests..." -ForegroundColor Yellow
go test ./internal/watch/...
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Tests passed" -ForegroundColor Green
} else {
    Write-Host "✗ Tests failed" -ForegroundColor Red
    exit 1
}

# 3. Create test environment
Write-Host "`n[3/4] Setting up test environment..." -ForegroundColor Yellow
New-Item -ItemType Directory -Force -Path .\handoff | Out-Null

# Create valid intent
@'
{
  "intent_type": "scaling",
  "target": "verify-deployment",
  "namespace": "default",
  "replicas": 5,
  "source": "test"
}
'@ | Out-File -FilePath .\handoff\intent-verify-valid.json -Encoding utf8

# Create invalid intent
@'
{
  "intent_type": "invalid",
  "target": "verify-deployment",
  "namespace": "default",
  "replicas": 5
}
'@ | Out-File -FilePath .\handoff\intent-verify-invalid.json -Encoding utf8

Write-Host "✓ Test files created" -ForegroundColor Green

# 4. Run conductor-watch briefly
Write-Host "`n[4/4] Testing conductor-watch..." -ForegroundColor Yellow
$job = Start-Job -ScriptBlock {
    param($path)
    Set-Location $path
    .\conductor-watch.exe --handoff .\handoff 2>&1
} -ArgumentList (Get-Location)

Start-Sleep -Seconds 2
$output = Receive-Job $job
Stop-Job $job
Remove-Job $job

# Check for expected patterns
$hasWatchOK = $output -match "WATCH:OK"
$hasWatchInvalid = $output -match "WATCH:INVALID"
$hasProcessed = $output -match "Processed \d+ existing intent files"

Write-Host "`nOutput sample:" -ForegroundColor Gray
$output | Select-Object -First 10 | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkGray }

if ($hasWatchOK -and $hasWatchInvalid -and $hasProcessed) {
    Write-Host "`n✓ Verification PASSED" -ForegroundColor Green
    Write-Host "  - WATCH:OK found" -ForegroundColor Green
    Write-Host "  - WATCH:INVALID found" -ForegroundColor Green
    Write-Host "  - Startup processing found" -ForegroundColor Green
} else {
    Write-Host "`n✗ Verification FAILED" -ForegroundColor Red
    if (-not $hasWatchOK) { Write-Host "  - WATCH:OK not found" -ForegroundColor Red }
    if (-not $hasWatchInvalid) { Write-Host "  - WATCH:INVALID not found" -ForegroundColor Red }
    if (-not $hasProcessed) { Write-Host "  - Startup processing not found" -ForegroundColor Red }
}

Write-Host "`n=== Verification Complete ===" -ForegroundColor Cyan
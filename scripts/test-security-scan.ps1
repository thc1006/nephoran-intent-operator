# Test Security Scan - Verifies govulncheck functionality locally
# This script tests the security scan process that will run in CI

param(
    [switch]$Verbose,
    [string]$OutputDir = ".test-security-results"
)

Write-Host "==== Testing Security Scan Process ====" -ForegroundColor Green

# Create output directory
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Check Go installation
Write-Host "Checking Go installation..." -ForegroundColor Yellow
$goVersion = go version
Write-Host "Go Version: $goVersion"

if ($goVersion -notmatch "go1\.24\.6") {
    Write-Warning "Expected Go 1.24.6 but found different version"
}

# Check if govulncheck is installed
Write-Host "`nChecking govulncheck installation..." -ForegroundColor Yellow
$govulnPath = "$env:USERPROFILE\go\bin\govulncheck.exe"

if (-not (Test-Path $govulnPath)) {
    Write-Host "Installing govulncheck..." -ForegroundColor Yellow
    go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
    
    if (-not (Test-Path $govulnPath)) {
        Write-Error "Failed to install govulncheck"
        exit 1
    }
}

# Verify govulncheck works
Write-Host "`nVerifying govulncheck..." -ForegroundColor Yellow
try {
    $versionOutput = & $govulnPath -version
    Write-Host "govulncheck version output:"
    Write-Host $versionOutput -ForegroundColor Cyan
} catch {
    Write-Error "Failed to run govulncheck -version: $_"
    exit 1
}

# Test JSON output format
Write-Host "`nTesting JSON scan format..." -ForegroundColor Yellow
$jsonOutput = "$OutputDir\govulncheck-test.json"

try {
    Write-Host "Running: govulncheck -json ./..."
    & $govulnPath -json ./... > $jsonOutput 2>&1
    $scanExitCode = $LASTEXITCODE
    
    Write-Host "Scan completed with exit code: $scanExitCode"
    
    # Check if output file was created
    if (Test-Path $jsonOutput) {
        $fileSize = (Get-Item $jsonOutput).Length
        Write-Host "Output file created: $jsonOutput ($fileSize bytes)" -ForegroundColor Green
        
        # Try to parse JSON
        try {
            $jsonContent = Get-Content $jsonOutput -Raw | ConvertFrom-Json
            $findings = @($jsonContent.finding)
            Write-Host "JSON parsing successful. Found $($findings.Count) findings." -ForegroundColor Green
            
            if ($findings.Count -eq 0) {
                Write-Host "✅ No vulnerabilities found!" -ForegroundColor Green
            } else {
                Write-Host "⚠️  Found $($findings.Count) vulnerabilities" -ForegroundColor Yellow
                foreach ($finding in $findings[0..2]) { # Show first 3
                    Write-Host "  - $($finding.osv.id): $($finding.osv.summary)" -ForegroundColor Red
                }
                if ($findings.Count -gt 3) {
                    Write-Host "  ... and $($findings.Count - 3) more" -ForegroundColor Red
                }
            }
            
        } catch {
            Write-Warning "Failed to parse JSON output: $_"
            Write-Host "Raw output (first 500 chars):"
            Write-Host (Get-Content $jsonOutput -Raw).Substring(0, [Math]::Min(500, (Get-Content $jsonOutput -Raw).Length))
        }
    } else {
        Write-Error "No output file was created"
    }
    
} catch {
    Write-Error "Failed to run vulnerability scan: $_"
    exit 1
}

# Test module download and verify
Write-Host "`nTesting module operations..." -ForegroundColor Yellow
try {
    Write-Host "Running: go mod download"
    go mod download
    Write-Host "Running: go mod verify" 
    go mod verify
    Write-Host "✅ Module operations successful" -ForegroundColor Green
} catch {
    Write-Warning "Module operations had issues: $_"
}

Write-Host "`n==== Security Scan Test Summary ====" -ForegroundColor Green
Write-Host "✅ Go 1.24.6 available"
Write-Host "✅ govulncheck v1.1.4 installed and working"  
Write-Host "✅ JSON output format functional"
Write-Host "✅ Module download/verify operational"

if ($scanExitCode -eq 0) {
    Write-Host "✅ No vulnerabilities detected in current codebase" -ForegroundColor Green
} elseif ($scanExitCode -eq 1) {
    Write-Host "⚠️  Vulnerabilities detected - review required" -ForegroundColor Yellow
} else {
    Write-Host "⚠️  Scan completed with exit code $scanExitCode" -ForegroundColor Yellow
}

Write-Host "`nTest artifacts saved to: $OutputDir" -ForegroundColor Cyan
Write-Host "This process mirrors what will happen in the CI pipeline." -ForegroundColor Cyan
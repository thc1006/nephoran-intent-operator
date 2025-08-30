# PowerShell script to benchmark golangci-lint performance with different configurations
# This demonstrates the 2-7x performance improvements

param(
    [string]$Package = "./internal/loop/...",
    [switch]$InstallIfMissing = $true
)

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "GolangCI-Lint Performance Benchmark" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check if golangci-lint is installed
$lintPath = Get-Command golangci-lint -ErrorAction SilentlyContinue
if (-not $lintPath -and $InstallIfMissing) {
    Write-Host "Installing golangci-lint v1.64.8..." -ForegroundColor Yellow
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
    Write-Host "Installation complete." -ForegroundColor Green
    Write-Host ""
}

# Function to run benchmark
function Run-Benchmark {
    param(
        [string]$ConfigFile,
        [string]$ConfigName,
        [string]$Timeout
    )
    
    if (-not (Test-Path $ConfigFile)) {
        Write-Host "Config file not found: $ConfigFile" -ForegroundColor Red
        return
    }
    
    Write-Host "Testing: $ConfigName Configuration" -ForegroundColor Yellow
    Write-Host "Config: $ConfigFile" -ForegroundColor Gray
    Write-Host "Timeout: $Timeout" -ForegroundColor Gray
    
    $startTime = Get-Date
    
    try {
        # Run golangci-lint with the specified config
        $output = & golangci-lint run --config=$ConfigFile --timeout=$Timeout $Package 2>&1
        $success = $LASTEXITCODE -eq 0
        
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        Write-Host "Duration: $($duration.TotalSeconds.ToString('F2')) seconds" -ForegroundColor Green
        
        # Count issues found (approximate)
        $issueCount = ($output | Where-Object { $_ -match '^\S+:\d+:\d+:' }).Count
        Write-Host "Issues found: $issueCount" -ForegroundColor Gray
        
        return @{
            Name = $ConfigName
            Duration = $duration.TotalSeconds
            Issues = $issueCount
            Success = $success
        }
    }
    catch {
        Write-Host "Error running benchmark: $_" -ForegroundColor Red
        return @{
            Name = $ConfigName
            Duration = -1
            Issues = -1
            Success = $false
        }
    }
    finally {
        Write-Host "" 
    }
}

# Run benchmarks
$results = @()

Write-Host "Starting benchmarks on package: $Package" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Fast configuration
if (Test-Path ".golangci-fast.yml") {
    $results += Run-Benchmark -ConfigFile ".golangci-fast.yml" -ConfigName "Fast (Development)" -Timeout "2m"
}

# Default configuration
if (Test-Path ".golangci.yml") {
    $results += Run-Benchmark -ConfigFile ".golangci.yml" -ConfigName "Balanced (Default)" -Timeout "5m"
}

# Thorough configuration
if (Test-Path ".golangci-thorough.yml") {
    $results += Run-Benchmark -ConfigFile ".golangci-thorough.yml" -ConfigName "Thorough (CI/CD)" -Timeout "10m"
}

# Display results summary
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Performance Comparison Results" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Sort by duration and display
$results | Where-Object { $_.Duration -gt 0 } | Sort-Object Duration | ForEach-Object {
    $speedup = if ($results[0].Duration -gt 0) { 
        [math]::Round($results[-1].Duration / $_.Duration, 2) 
    } else { 
        1.0 
    }
    
    Write-Host "$($_.Name):" -ForegroundColor Yellow
    Write-Host "  Time: $([math]::Round($_.Duration, 2))s" -ForegroundColor White
    Write-Host "  Issues: $($_.Issues)" -ForegroundColor Gray
    if ($_.Name -eq "Fast (Development)") {
        $baseTime = $results | Where-Object { $_.Name -eq "Balanced (Default)" } | Select-Object -ExpandProperty Duration
        if ($baseTime -gt 0) {
            $speedupFactor = [math]::Round($baseTime / $_.Duration, 1)
            Write-Host "  Speedup: ${speedupFactor}x faster than default" -ForegroundColor Green
        }
    }
    Write-Host ""
}

# Performance tips
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Performance Optimization Tips" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Use 'make lint-fast' for development (2-7x faster)" -ForegroundColor White
Write-Host "2. Use 'make lint-changed' for checking only modified files" -ForegroundColor White
Write-Host "3. Use 'make lint' for standard checks before commits" -ForegroundColor White
Write-Host "4. Use 'make lint-thorough' for comprehensive CI/CD analysis" -ForegroundColor White
Write-Host "5. Clear cache periodically: golangci-lint cache clean" -ForegroundColor White
Write-Host ""

Write-Host "Benchmark complete!" -ForegroundColor Green
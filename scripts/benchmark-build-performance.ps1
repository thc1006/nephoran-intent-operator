# PowerShell script to benchmark build performance improvements
# Measures and compares build times between standard and ultra-fast modes

param(
    [switch]$Baseline = $false,
    [switch]$Compare = $false,
    [string]$OutputFile = "build-benchmark-results.json"
)

$ErrorActionPreference = "Stop"

# Results storage
$results = @{
    timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    system = @{
        os = $env:OS
        processor = $env:PROCESSOR_IDENTIFIER
        cores = $env:NUMBER_OF_PROCESSORS
        memory = (Get-CimInstance Win32_PhysicalMemory | Measure-Object -Property capacity -Sum).Sum / 1gb
    }
    benchmarks = @{}
}

function Measure-BuildTime {
    param(
        [string]$Name,
        [scriptblock]$Command
    )
    
    Write-Host "Running benchmark: $Name" -ForegroundColor Cyan
    
    # Clean before measurement
    if (Test-Path "bin") {
        Remove-Item -Path "bin" -Recurse -Force 2>$null
    }
    
    # Measure cold start (no cache)
    go clean -cache 2>$null
    $coldStart = Measure-Command {
        & $Command 2>&1 | Out-Null
    }
    
    # Measure warm start (with cache)
    $warmStart = Measure-Command {
        & $Command 2>&1 | Out-Null
    }
    
    # Measure incremental build (minor change)
    if (Test-Path "cmd/conductor-loop/main.go") {
        # Add a comment to trigger rebuild
        Add-Content -Path "cmd/conductor-loop/main.go" -Value "// benchmark test"
        $incremental = Measure-Command {
            & $Command 2>&1 | Out-Null
        }
        # Remove the comment
        $content = Get-Content "cmd/conductor-loop/main.go" | Where-Object { $_ -notmatch "// benchmark test" }
        Set-Content -Path "cmd/conductor-loop/main.go" -Value $content
    } else {
        $incremental = $warmStart
    }
    
    $result = @{
        cold_start_seconds = [math]::Round($coldStart.TotalSeconds, 2)
        warm_start_seconds = [math]::Round($warmStart.TotalSeconds, 2)
        incremental_seconds = [math]::Round($incremental.TotalSeconds, 2)
    }
    
    Write-Host "  Cold start: $($result.cold_start_seconds)s" -ForegroundColor Gray
    Write-Host "  Warm start: $($result.warm_start_seconds)s" -ForegroundColor Gray
    Write-Host "  Incremental: $($result.incremental_seconds)s" -ForegroundColor Gray
    
    return $result
}

function Run-Benchmarks {
    Write-Host "Starting build performance benchmarks..." -ForegroundColor Green
    Write-Host "This will take several minutes..." -ForegroundColor Yellow
    
    # Benchmark 1: Standard build
    $results.benchmarks.standard_build = Measure-BuildTime -Name "Standard Build" -Command {
        make build
    }
    
    # Benchmark 2: Fast build
    $results.benchmarks.fast_build = Measure-BuildTime -Name "Fast Build" -Command {
        make build-fast
    }
    
    # Benchmark 3: Ultra build
    $results.benchmarks.ultra_build = Measure-BuildTime -Name "Ultra Build" -Command {
        make build-ultra
    }
    
    # Benchmark 4: Standard test
    Write-Host "Running test benchmarks..." -ForegroundColor Cyan
    $testStandard = Measure-Command {
        make test 2>&1 | Out-Null
    }
    $results.benchmarks.standard_test = @{
        seconds = [math]::Round($testStandard.TotalSeconds, 2)
    }
    Write-Host "  Standard test: $($results.benchmarks.standard_test.seconds)s" -ForegroundColor Gray
    
    # Benchmark 5: Ultra test
    $testUltra = Measure-Command {
        make test-ultra 2>&1 | Out-Null
    }
    $results.benchmarks.ultra_test = @{
        seconds = [math]::Round($testUltra.TotalSeconds, 2)
    }
    Write-Host "  Ultra test: $($results.benchmarks.ultra_test.seconds)s" -ForegroundColor Gray
    
    # Benchmark 6: Docker build (if Docker is available)
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        Write-Host "Running Docker benchmarks..." -ForegroundColor Cyan
        
        # Standard Docker build
        $dockerStandard = Measure-Command {
            docker build -f Dockerfile --build-arg SERVICE=conductor-loop -t test:standard . 2>&1 | Out-Null
        }
        $results.benchmarks.docker_standard = @{
            seconds = [math]::Round($dockerStandard.TotalSeconds, 2)
        }
        Write-Host "  Standard Docker: $($results.benchmarks.docker_standard.seconds)s" -ForegroundColor Gray
        
        # Ultra Docker build
        $dockerUltra = Measure-Command {
            docker build -f Dockerfile.ultra-fast --build-arg SERVICE=conductor-loop -t test:ultra . 2>&1 | Out-Null
        }
        $results.benchmarks.docker_ultra = @{
            seconds = [math]::Round($dockerUltra.TotalSeconds, 2)
        }
        Write-Host "  Ultra Docker: $($results.benchmarks.docker_ultra.seconds)s" -ForegroundColor Gray
    }
    
    # Calculate improvements
    $results.improvements = @{
        build_speedup = [math]::Round($results.benchmarks.standard_build.warm_start_seconds / $results.benchmarks.ultra_build.warm_start_seconds, 2)
        test_speedup = [math]::Round($results.benchmarks.standard_test.seconds / $results.benchmarks.ultra_test.seconds, 2)
    }
    
    if ($results.benchmarks.docker_standard) {
        $results.improvements.docker_speedup = [math]::Round($results.benchmarks.docker_standard.seconds / $results.benchmarks.docker_ultra.seconds, 2)
    }
}

function Show-Results {
    Write-Host "`nBenchmark Results" -ForegroundColor Cyan
    Write-Host "=================" -ForegroundColor Cyan
    
    Write-Host "`nBuild Performance:" -ForegroundColor Yellow
    Write-Host "  Standard build (warm): $($results.benchmarks.standard_build.warm_start_seconds)s"
    Write-Host "  Fast build (warm):     $($results.benchmarks.fast_build.warm_start_seconds)s"
    Write-Host "  Ultra build (warm):    $($results.benchmarks.ultra_build.warm_start_seconds)s" -ForegroundColor Green
    
    Write-Host "`nTest Performance:" -ForegroundColor Yellow
    Write-Host "  Standard test: $($results.benchmarks.standard_test.seconds)s"
    Write-Host "  Ultra test:    $($results.benchmarks.ultra_test.seconds)s" -ForegroundColor Green
    
    if ($results.benchmarks.docker_standard) {
        Write-Host "`nDocker Performance:" -ForegroundColor Yellow
        Write-Host "  Standard Docker: $($results.benchmarks.docker_standard.seconds)s"
        Write-Host "  Ultra Docker:    $($results.benchmarks.docker_ultra.seconds)s" -ForegroundColor Green
    }
    
    Write-Host "`nPerformance Improvements:" -ForegroundColor Cyan
    Write-Host "  Build speedup: $($results.improvements.build_speedup)x faster" -ForegroundColor Green
    Write-Host "  Test speedup:  $($results.improvements.test_speedup)x faster" -ForegroundColor Green
    if ($results.improvements.docker_speedup) {
        Write-Host "  Docker speedup: $($results.improvements.docker_speedup)x faster" -ForegroundColor Green
    }
    
    # Calculate total time saved
    $standardTotal = $results.benchmarks.standard_build.warm_start_seconds + $results.benchmarks.standard_test.seconds
    $ultraTotal = $results.benchmarks.ultra_build.warm_start_seconds + $results.benchmarks.ultra_test.seconds
    $timeSaved = $standardTotal - $ultraTotal
    $percentImprovement = [math]::Round((($standardTotal - $ultraTotal) / $standardTotal) * 100, 1)
    
    Write-Host "`nTotal Time Comparison:" -ForegroundColor Cyan
    Write-Host "  Standard workflow: $([math]::Round($standardTotal, 1))s"
    Write-Host "  Ultra workflow:    $([math]::Round($ultraTotal, 1))s" -ForegroundColor Green
    Write-Host "  Time saved:        $([math]::Round($timeSaved, 1))s ($percentImprovement% improvement)" -ForegroundColor Green
}

function Save-Results {
    $results | ConvertTo-Json -Depth 10 | Set-Content -Path $OutputFile
    Write-Host "`nResults saved to: $OutputFile" -ForegroundColor Cyan
}

function Compare-Results {
    if (-not (Test-Path $OutputFile)) {
        Write-Host "No baseline results found. Run with -Baseline first." -ForegroundColor Red
        exit 1
    }
    
    $baseline = Get-Content $OutputFile | ConvertFrom-Json
    Run-Benchmarks
    
    Write-Host "`nComparison with Baseline" -ForegroundColor Cyan
    Write-Host "========================" -ForegroundColor Cyan
    
    # Compare build times
    $baselineBuild = $baseline.benchmarks.ultra_build.warm_start_seconds
    $currentBuild = $results.benchmarks.ultra_build.warm_start_seconds
    $buildDiff = $currentBuild - $baselineBuild
    $buildPercent = [math]::Round(($buildDiff / $baselineBuild) * 100, 1)
    
    Write-Host "`nUltra Build Performance:"
    Write-Host "  Baseline: $baselineBuild s"
    Write-Host "  Current:  $currentBuild s"
    if ($buildDiff -lt 0) {
        Write-Host "  Change:   $([math]::Abs($buildDiff))s faster ($([math]::Abs($buildPercent))% improvement)" -ForegroundColor Green
    } elseif ($buildDiff -gt 0) {
        Write-Host "  Change:   $buildDiff s slower ($buildPercent% regression)" -ForegroundColor Red
    } else {
        Write-Host "  Change:   No change" -ForegroundColor Yellow
    }
    
    # Compare test times
    $baselineTest = $baseline.benchmarks.ultra_test.seconds
    $currentTest = $results.benchmarks.ultra_test.seconds
    $testDiff = $currentTest - $baselineTest
    $testPercent = [math]::Round(($testDiff / $baselineTest) * 100, 1)
    
    Write-Host "`nUltra Test Performance:"
    Write-Host "  Baseline: $baselineTest s"
    Write-Host "  Current:  $currentTest s"
    if ($testDiff -lt 0) {
        Write-Host "  Change:   $([math]::Abs($testDiff))s faster ($([math]::Abs($testPercent))% improvement)" -ForegroundColor Green
    } elseif ($testDiff -gt 0) {
        Write-Host "  Change:   $testDiff s slower ($testPercent% regression)" -ForegroundColor Red
    } else {
        Write-Host "  Change:   No change" -ForegroundColor Yellow
    }
}

# Main logic
if ($Baseline) {
    Run-Benchmarks
    Show-Results
    Save-Results
    Write-Host "`nBaseline established. Run with -Compare to track improvements." -ForegroundColor Green
} elseif ($Compare) {
    Compare-Results
} else {
    Run-Benchmarks
    Show-Results
    Save-Results
}

Write-Host "`nOptimization Tips:" -ForegroundColor Cyan
Write-Host "1. Enable build caching: .\scripts\enable-build-cache.ps1" -ForegroundColor Gray
Write-Host "2. Use SSD for GOCACHE and GOMODCACHE" -ForegroundColor Gray
Write-Host "3. Close unnecessary applications to free up RAM" -ForegroundColor Gray
Write-Host "4. Use 'make ultra-fast' for the fastest workflow" -ForegroundColor Gray
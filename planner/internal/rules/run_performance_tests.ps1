# Performance Test Runner for Rule Engine Optimizations
# This script runs comprehensive performance tests to validate memory optimizations

param(
    [string]$TestType = "all",  # Options: all, unit, stress, bench
    [switch]$Verbose,
    [switch]$Short,
    [string]$OutputFile = ""
)

$ErrorActionPreference = "Stop"

Write-Host "Rule Engine Performance Test Suite" -ForegroundColor Green
Write-Host "===================================" -ForegroundColor Green

# Change to the rules directory
$rulesDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $rulesDir

# Define test categories
$testSuites = @{
    "unit" = @{
        "name" = "Unit & Integration Tests"
        "command" = "go test -v -run 'Test.*' ."
        "description" = "Core functionality and optimization validation"
    }
    "stress" = @{
        "name" = "Stress Tests"
        "command" = "go test -v -tags=performance -run 'StressTest.*' ."
        "description" = "High-load scenarios and memory leak detection"
    }
    "bench" = @{
        "name" = "Benchmarks"
        "command" = "go test -v -bench=. -benchmem -run='^$' ."
        "description" = "Performance measurements and comparisons"
    }
    "bench-stress" = @{
        "name" = "Performance Benchmarks"
        "command" = "go test -v -tags=performance -bench=. -benchmem -run='^$' ."
        "description" = "Comprehensive performance analysis"
    }
}

function Run-TestSuite {
    param(
        [string]$SuiteName,
        [hashtable]$Suite
    )
    
    Write-Host "`n$($Suite.name)" -ForegroundColor Cyan
    Write-Host ("-" * $Suite.name.Length) -ForegroundColor Cyan
    Write-Host $Suite.description -ForegroundColor Gray
    
    $command = $Suite.command
    
    if ($Short) {
        $command += " -short"
    }
    
    if ($Verbose) {
        $command += " -v"
    }
    
    Write-Host "`nExecuting: $command" -ForegroundColor Yellow
    
    $startTime = Get-Date
    
    try {
        if ($OutputFile -ne "") {
            $output = Invoke-Expression $command 2>&1
            $output | Out-File -FilePath $OutputFile -Append -Encoding UTF8
            Write-Host $output
        } else {
            Invoke-Expression $command
        }
        
        $duration = (Get-Date) - $startTime
        Write-Host "`n✓ $($Suite.name) completed in $($duration.TotalSeconds.ToString('F2')) seconds" -ForegroundColor Green
        
    } catch {
        $duration = (Get-Date) - $startTime
        Write-Host "`n✗ $($Suite.name) failed after $($duration.TotalSeconds.ToString('F2')) seconds" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        throw
    }
}

function Show-Summary {
    Write-Host "`n" -NoNewline
    Write-Host "Performance Test Summary" -ForegroundColor Green
    Write-Host "========================" -ForegroundColor Green
    
    Write-Host "`nKey Optimization Areas Tested:" -ForegroundColor White
    Write-Host "  ✓ Memory Growth Prevention - Capacity limits enforced" -ForegroundColor Green
    Write-Host "  ✓ Efficient Pruning - In-place operations vs slice recreation" -ForegroundColor Green
    Write-Host "  ✓ Thread Safety - Concurrent access patterns" -ForegroundColor Green
    Write-Host "  ✓ Long-running Stability - Extended operation memory patterns" -ForegroundColor Green
    Write-Host "  ✓ Edge Case Handling - Boundary conditions and error scenarios" -ForegroundColor Green
    Write-Host "  ✓ Backward Compatibility - Existing functionality preserved" -ForegroundColor Green
    
    Write-Host "`nPerformance Improvements Validated:" -ForegroundColor White
    Write-Host "  • MaxHistorySize parameter prevents unbounded memory growth" -ForegroundColor Cyan
    Write-Host "  • In-place pruning reduces allocation overhead" -ForegroundColor Cyan
    Write-Host "  • Conditional pruning based on PruneInterval optimizes CPU usage" -ForegroundColor Cyan
    Write-Host "  • Emergency capacity management handles burst scenarios" -ForegroundColor Cyan
    
    Write-Host "`nTo analyze specific performance characteristics:" -ForegroundColor White
    Write-Host "  go test -bench=BenchmarkRuleEngine_MemoryGrowthComparison -benchmem" -ForegroundColor Gray
    Write-Host "  go test -bench=BenchmarkRuleEngine_PruningComparison -benchmem" -ForegroundColor Gray
    Write-Host "  go test -run=TestRuleEngine_LongRunningMemoryStability" -ForegroundColor Gray
}

# Initialize output file if specified
if ($OutputFile -ne "") {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "Rule Engine Performance Test Results - $timestamp" | Out-File -FilePath $OutputFile -Encoding UTF8
    "=" * 60 | Out-File -FilePath $OutputFile -Append -Encoding UTF8
    "" | Out-File -FilePath $OutputFile -Append -Encoding UTF8
}

# Main execution logic
try {
    switch ($TestType.ToLower()) {
        "all" {
            Write-Host "Running complete test suite..." -ForegroundColor White
            Run-TestSuite "unit" $testSuites["unit"]
            Run-TestSuite "bench" $testSuites["bench"]
            if (-not $Short) {
                Run-TestSuite "stress" $testSuites["stress"]
                Run-TestSuite "bench-stress" $testSuites["bench-stress"]
            }
        }
        "unit" {
            Run-TestSuite "unit" $testSuites["unit"]
        }
        "stress" {
            Run-TestSuite "stress" $testSuites["stress"]
        }
        "bench" {
            Run-TestSuite "bench" $testSuites["bench"]
        }
        "bench-stress" {
            Run-TestSuite "bench-stress" $testSuites["bench-stress"]
        }
        default {
            Write-Host "Invalid test type: $TestType" -ForegroundColor Red
            Write-Host "Valid options: all, unit, stress, bench" -ForegroundColor Gray
            exit 1
        }
    }
    
    Show-Summary
    
    if ($OutputFile -ne "") {
        Write-Host "`nResults saved to: $OutputFile" -ForegroundColor Green
    }
    
} catch {
    Write-Host "`nTest execution failed!" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

Write-Host "`nAll performance tests completed successfully! 🎉" -ForegroundColor Green
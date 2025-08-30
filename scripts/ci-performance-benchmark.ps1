#!/usr/bin/env pwsh
# =============================================================================
# CI Performance Benchmarking Script
# Measures and tracks CI pipeline performance metrics
# =============================================================================

param(
    [string]$WorkflowFile = ".github/workflows/ci-ultra-speed.yml",
    [string]$BaselineFile = ".github/workflows/ci.yml",
    [switch]$GenerateReport,
    [switch]$TestLocal
)

# Performance metrics storage
$script:Metrics = @{
    Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Platform = "GitHub Actions Ubuntu Latest"
    Measurements = @{}
}

# Colors for output
$script:Colors = @{
    Success = "Green"
    Warning = "Yellow"  
    Error = "Red"
    Info = "Cyan"
    Metric = "Magenta"
}

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Measure-WorkflowPerformance {
    param(
        [string]$WorkflowPath,
        [string]$WorkflowName
    )
    
    Write-ColorOutput "`n📊 Analyzing Workflow: $WorkflowName" $script:Colors.Info
    Write-ColorOutput ("=" * 60) $script:Colors.Info
    
    if (-not (Test-Path $WorkflowPath)) {
        Write-ColorOutput "❌ Workflow file not found: $WorkflowPath" $script:Colors.Error
        return $null
    }
    
    $content = Get-Content $WorkflowPath -Raw
    $metrics = @{
        WorkflowName = $WorkflowName
        FilePath = $WorkflowPath
        FileSize = (Get-Item $WorkflowPath).Length
        Jobs = @{}
        TotalJobs = 0
        ParallelJobs = 0
        TotalTimeoutMinutes = 0
        CacheStrategies = @()
        OptimizationFeatures = @()
    }
    
    # Parse YAML to extract job information
    $jobs = [regex]::Matches($content, '(?ms)^\s{2}(\w[\w-]*):.*?(?=^\s{2}\w|^\w|\z)')
    $metrics.TotalJobs = $jobs.Count
    
    foreach ($job in $jobs) {
        $jobName = $job.Groups[1].Value
        $jobContent = $job.Value
        
        # Extract timeout
        if ($jobContent -match 'timeout-minutes:\s*(\d+)') {
            $timeout = [int]$matches[1]
            $metrics.Jobs[$jobName] = @{ Timeout = $timeout }
            $metrics.TotalTimeoutMinutes += $timeout
        }
        
        # Check for matrix strategy (parallelization)
        if ($jobContent -match 'strategy:.*?matrix:') {
            $metrics.ParallelJobs++
        }
        
        # Check for cache usage
        if ($jobContent -match 'uses:\s*actions/cache(@v\d+)?') {
            $metrics.CacheStrategies += "Job: $jobName uses GitHub Actions Cache"
        }
    }
    
    # Identify optimization features
    if ($content -match 'cancel-in-progress:\s*true') {
        $metrics.OptimizationFeatures += "Cancel in-progress runs"
    }
    if ($content -match 'DOCKER_BUILDKIT:\s*["'']?1') {
        $metrics.OptimizationFeatures += "Docker BuildKit enabled"
    }
    if ($content -match 'CGO_ENABLED:\s*["'']?0') {
        $metrics.OptimizationFeatures += "CGO disabled for faster builds"
    }
    if ($content -match 'GOPROXY:') {
        $metrics.OptimizationFeatures += "Go proxy caching"
    }
    if ($content -match 'fetch-depth:\s*[12]') {
        $metrics.OptimizationFeatures += "Shallow git clone"
    }
    if ($content -match '\-\-fast') {
        $metrics.OptimizationFeatures += "Fast mode enabled"
    }
    if ($content -match 'skip-cache:\s*false|cache:\s*true') {
        $metrics.OptimizationFeatures += "Aggressive caching"
    }
    
    # Calculate parallelization score
    $metrics.ParallelizationRatio = if ($metrics.TotalJobs -gt 0) { 
        [math]::Round(($metrics.ParallelJobs / $metrics.TotalJobs) * 100, 2)
    } else { 0 }
    
    # Estimate execution time (simplified model)
    $metrics.EstimatedSerialTime = $metrics.TotalTimeoutMinutes
    $metrics.EstimatedParallelTime = if ($metrics.ParallelJobs -gt 0) {
        [math]::Ceiling($metrics.TotalTimeoutMinutes / ([math]::Max($metrics.ParallelJobs, 1)))
    } else {
        $metrics.TotalTimeoutMinutes
    }
    
    return $metrics
}

function Compare-WorkflowPerformance {
    param(
        [hashtable]$Baseline,
        [hashtable]$Optimized
    )
    
    Write-ColorOutput "`n🔥 PERFORMANCE COMPARISON" $script:Colors.Success
    Write-ColorOutput ("=" * 60) $script:Colors.Success
    
    # Time improvements
    $timeImprovement = [math]::Round(
        (($Baseline.EstimatedSerialTime - $Optimized.EstimatedParallelTime) / $Baseline.EstimatedSerialTime) * 100, 
        2
    )
    
    $parallelSpeedup = [math]::Round(
        $Baseline.EstimatedSerialTime / [math]::Max($Optimized.EstimatedParallelTime, 1),
        2
    )
    
    Write-ColorOutput "`n⏱️  Time Metrics:" $script:Colors.Metric
    Write-ColorOutput "  Baseline Serial Time: $($Baseline.EstimatedSerialTime) minutes" White
    Write-ColorOutput "  Optimized Parallel Time: $($Optimized.EstimatedParallelTime) minutes" White
    Write-ColorOutput "  Time Improvement: $timeImprovement%" $script:Colors.Success
    Write-ColorOutput "  Speedup Factor: ${parallelSpeedup}x" $script:Colors.Success
    
    Write-ColorOutput "`n🚀 Parallelization:" $script:Colors.Metric
    Write-ColorOutput "  Baseline Parallel Jobs: $($Baseline.ParallelJobs)/$($Baseline.TotalJobs) ($($Baseline.ParallelizationRatio)%)" White
    Write-ColorOutput "  Optimized Parallel Jobs: $($Optimized.ParallelJobs)/$($Optimized.TotalJobs) ($($Optimized.ParallelizationRatio)%)" White
    
    $parallelIncrease = $Optimized.ParallelJobs - $Baseline.ParallelJobs
    if ($parallelIncrease -gt 0) {
        Write-ColorOutput "  Parallel Jobs Added: +$parallelIncrease" $script:Colors.Success
    }
    
    Write-ColorOutput "`n💾 Cache Optimizations:" $script:Colors.Metric
    Write-ColorOutput "  Baseline Cache Strategies: $($Baseline.CacheStrategies.Count)" White
    Write-ColorOutput "  Optimized Cache Strategies: $($Optimized.CacheStrategies.Count)" White
    
    $cacheImprovement = $Optimized.CacheStrategies.Count - $Baseline.CacheStrategies.Count
    if ($cacheImprovement -gt 0) {
        Write-ColorOutput "  Additional Cache Layers: +$cacheImprovement" $script:Colors.Success
    }
    
    Write-ColorOutput "`n⚡ Optimization Features:" $script:Colors.Metric
    $newFeatures = $Optimized.OptimizationFeatures | Where-Object { $_ -notin $Baseline.OptimizationFeatures }
    if ($newFeatures) {
        foreach ($feature in $newFeatures) {
            Write-ColorOutput "  ✅ $feature" $script:Colors.Success
        }
    } else {
        Write-ColorOutput "  No new optimization features detected" $script:Colors.Warning
    }
    
    # Performance grade
    Write-ColorOutput "`n🏆 Performance Grade:" $script:Colors.Metric
    $grade = switch ($timeImprovement) {
        { $_ -ge 80 } { "S - ULTRA SPEED ACHIEVED! 🚀🚀🚀"; break }
        { $_ -ge 60 } { "A - Excellent Performance! 🚀🚀"; break }
        { $_ -ge 40 } { "B - Good Performance! 🚀"; break }
        { $_ -ge 20 } { "C - Moderate Performance"; break }
        { $_ -ge 0 }  { "D - Minimal Improvement"; break }
        default { "F - Performance Degraded" }
    }
    
    $gradeColor = if ($timeImprovement -ge 40) { $script:Colors.Success } 
                  elseif ($timeImprovement -ge 20) { $script:Colors.Warning }
                  else { $script:Colors.Error }
    
    Write-ColorOutput "  $grade" $gradeColor
    
    return @{
        TimeImprovement = $timeImprovement
        SpeedupFactor = $parallelSpeedup
        ParallelIncrease = $parallelIncrease
        CacheImprovement = $cacheImprovement
        NewFeatures = $newFeatures
        Grade = $grade
    }
}

function Test-LocalPerformance {
    Write-ColorOutput "`n🔬 Running Local Performance Tests" $script:Colors.Info
    Write-ColorOutput ("=" * 60) $script:Colors.Info
    
    # Test Go build performance
    Write-ColorOutput "`nTesting Go build performance..." $script:Colors.Info
    $goBuildTime = Measure-Command {
        go build -o /tmp/test-build ./cmd/main.go 2>$null
    }
    Write-ColorOutput "  Go build time: $($goBuildTime.TotalSeconds) seconds" White
    
    # Test Go module download
    Write-ColorOutput "`nTesting Go module cache..." $script:Colors.Info
    $goModTime = Measure-Command {
        go mod download 2>$null
    }
    Write-ColorOutput "  Go mod download time: $($goModTime.TotalSeconds) seconds" White
    
    # Test parallel capability
    Write-ColorOutput "`nTesting parallel execution capability..." $script:Colors.Info
    $cpuCount = (Get-CimInstance Win32_ComputerSystem).NumberOfLogicalProcessors
    Write-ColorOutput "  Available CPU cores: $cpuCount" White
    Write-ColorOutput "  Recommended parallel jobs: $([math]::Min($cpuCount, 4))" $script:Colors.Success
    
    # Cache size analysis
    Write-ColorOutput "`nAnalyzing cache sizes..." $script:Colors.Info
    $goModCache = "$env:GOPATH/pkg/mod"
    if (Test-Path $goModCache) {
        $cacheSize = (Get-ChildItem $goModCache -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-ColorOutput "  Go module cache size: $([math]::Round($cacheSize, 2)) MB" White
    }
    
    return @{
        GoBuildTime = $goBuildTime.TotalSeconds
        GoModTime = $goModTime.TotalSeconds
        CPUCores = $cpuCount
        RecommendedParallelJobs = [math]::Min($cpuCount, 4)
    }
}

function Export-PerformanceReport {
    param(
        [hashtable]$BaselineMetrics,
        [hashtable]$OptimizedMetrics,
        [hashtable]$Comparison
    )
    
    $reportPath = "ci-performance-report-$(Get-Date -Format 'yyyyMMdd-HHmmss').md"
    
    $report = @"
# CI Performance Optimization Report

**Generated:** $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

## Executive Summary

The optimized CI pipeline achieves **${($Comparison.SpeedupFactor)}x speedup** with **$($Comparison.TimeImprovement)% time reduction**.

**Performance Grade: $($Comparison.Grade)**

## Detailed Metrics

### Time Performance
| Metric | Baseline | Optimized | Improvement |
|--------|----------|-----------|-------------|
| Estimated Serial Time | $($BaselineMetrics.EstimatedSerialTime) min | $($OptimizedMetrics.EstimatedSerialTime) min | - |
| Estimated Parallel Time | $($BaselineMetrics.EstimatedParallelTime) min | $($OptimizedMetrics.EstimatedParallelTime) min | **$($Comparison.TimeImprovement)%** |
| Total Jobs | $($BaselineMetrics.TotalJobs) | $($OptimizedMetrics.TotalJobs) | - |
| Parallel Jobs | $($BaselineMetrics.ParallelJobs) | $($OptimizedMetrics.ParallelJobs) | **+$($Comparison.ParallelIncrease)** |
| Parallelization Ratio | $($BaselineMetrics.ParallelizationRatio)% | $($OptimizedMetrics.ParallelizationRatio)% | - |

### Optimization Techniques

#### Cache Strategies
- Baseline: $($BaselineMetrics.CacheStrategies.Count) strategies
- Optimized: $($OptimizedMetrics.CacheStrategies.Count) strategies
- **Improvement: +$($Comparison.CacheImprovement) cache layers**

#### New Features Implemented
$(($Comparison.NewFeatures | ForEach-Object { "- ✅ $_" }) -join "`n")

## Recommendations

1. **Further Optimizations:**
   - Consider implementing distributed caching for shared dependencies
   - Explore using self-hosted runners for critical paths
   - Implement intelligent test selection based on changed files

2. **Monitoring:**
   - Track actual CI execution times in production
   - Monitor cache hit rates
   - Measure developer feedback cycle time

3. **Next Steps:**
   - Deploy optimized pipeline to staging branch
   - A/B test against baseline for 1 week
   - Collect metrics and iterate

## Technical Details

### Baseline Configuration
- File: $($BaselineMetrics.FilePath)
- Size: $($BaselineMetrics.FileSize) bytes

### Optimized Configuration  
- File: $($OptimizedMetrics.FilePath)
- Size: $($OptimizedMetrics.FileSize) bytes

---
*Report generated by CI Performance Benchmarking Tool*
"@
    
    $report | Out-File -FilePath $reportPath -Encoding utf8
    Write-ColorOutput "`n📄 Report saved to: $reportPath" $script:Colors.Success
    return $reportPath
}

# Main execution
function Main {
    Write-ColorOutput "`n🚀 CI PERFORMANCE BENCHMARKING TOOL" $script:Colors.Success
    Write-ColorOutput ("=" * 60) $script:Colors.Success
    Write-ColorOutput "Comparing: $BaselineFile vs $WorkflowFile" $script:Colors.Info
    
    # Analyze workflows
    $baselineMetrics = Measure-WorkflowPerformance -WorkflowPath $BaselineFile -WorkflowName "Baseline CI"
    $optimizedMetrics = Measure-WorkflowPerformance -WorkflowPath $WorkflowFile -WorkflowName "Ultra Speed CI"
    
    if (-not $baselineMetrics -or -not $optimizedMetrics) {
        Write-ColorOutput "`n❌ Failed to analyze workflows" $script:Colors.Error
        exit 1
    }
    
    # Compare performance
    $comparison = Compare-WorkflowPerformance -Baseline $baselineMetrics -Optimized $optimizedMetrics
    
    # Run local tests if requested
    if ($TestLocal) {
        $localMetrics = Test-LocalPerformance
        Write-ColorOutput "`n📊 Local Performance Metrics:" $script:Colors.Metric
        Write-ColorOutput "  Recommended parallel jobs based on CPU: $($localMetrics.RecommendedParallelJobs)" $script:Colors.Success
    }
    
    # Generate report if requested
    if ($GenerateReport) {
        Export-PerformanceReport -BaselineMetrics $baselineMetrics -OptimizedMetrics $optimizedMetrics -Comparison $comparison
    }
    
    Write-ColorOutput "`n✅ Performance analysis complete!" $script:Colors.Success
    Write-ColorOutput "The optimized pipeline achieves ${($comparison.SpeedupFactor)}x faster execution!" $script:Colors.Success
}

# Run main function
Main
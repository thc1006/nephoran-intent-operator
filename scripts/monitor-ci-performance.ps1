# monitor-ci-performance.ps1
# Monitors golangci-lint and CI performance for Nephoran project
# Provides performance metrics, success/failure tracking, and alerting

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("run", "report", "analyze", "alert", "help")]
    [string]$Action = "run",
    
    [Parameter()]
    [string]$ConfigFile = ".golangci.yml",
    
    [Parameter()]
    [string]$ReportPath = "ci-performance-report.json",
    
    [Parameter()]
    [string]$HistoryPath = "ci-performance-history.json",
    
    [Parameter()]
    [int]$TimeoutMinutes = 10,
    
    [Parameter()]
    [switch]$Verbose,
    
    [Parameter()]
    [switch]$SaveMetrics,
    
    [Parameter()]
    [string]$AlertThreshold = "300" # seconds
)

# Script constants
$script:ProjectRoot = Split-Path -Parent $PSScriptRoot
$script:StartTime = Get-Date
$script:ReportFile = Join-Path $script:ProjectRoot $ReportPath
$script:HistoryFile = Join-Path $script:ProjectRoot $HistoryPath

# Performance metrics structure
class CIPerformanceMetrics {
    [datetime]$Timestamp
    [string]$GitBranch
    [string]$GitCommit
    [string]$ConfigFile
    [double]$LintExecutionTime
    [int]$TotalFiles
    [int]$IssuesFound
    [string]$Status
    [string]$ErrorMessage
    [hashtable]$LinterTimings
    [hashtable]$ModuleTimings
    [double]$MemoryUsageMB
    [double]$CPUUsagePercent
}

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $(Get-Date -Format 'HH:mm:ss') $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $(Get-Date -Format 'HH:mm:ss') $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $(Get-Date -Format 'HH:mm:ss') $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $(Get-Date -Format 'HH:mm:ss') $Message" -ForegroundColor Red
}

function Write-Verbose {
    param([string]$Message)
    if ($Verbose -or $VerbosePreference -eq 'Continue') {
        Write-Host "[VERBOSE] $(Get-Date -Format 'HH:mm:ss') $Message" -ForegroundColor Gray
    }
}

# Get git information
function Get-GitInfo {
    try {
        $branch = & git rev-parse --abbrev-ref HEAD 2>$null
        $commit = & git rev-parse --short HEAD 2>$null
        return @{
            Branch = if ($branch) { $branch } else { "unknown" }
            Commit = if ($commit) { $commit } else { "unknown" }
        }
    } catch {
        return @{
            Branch = "unknown"
            Commit = "unknown"
        }
    }
}

# Check if golangci-lint is available
function Test-GolangCILint {
    try {
        $version = & golangci-lint version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Verbose "Found golangci-lint: $($version -split "`n" | Select-Object -First 1)"
            return $true
        }
    } catch {}
    
    Write-Warning "golangci-lint not found in PATH. Installing..."
    return Install-GolangCILint
}

# Install golangci-lint
function Install-GolangCILint {
    try {
        Write-Info "Installing golangci-lint..."
        
        # Download and install golangci-lint
        $installScript = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
        $version = "v1.63.4"
        
        if ($IsWindows -or $env:OS -eq "Windows_NT") {
            # Windows installation
            $gopath = $env:GOPATH
            if (-not $gopath) {
                $gopath = & go env GOPATH 2>$null
            }
            if (-not $gopath) {
                throw "GOPATH not found"
            }
            
            $binPath = Join-Path $gopath "bin"
            if (-not (Test-Path $binPath)) {
                New-Item -ItemType Directory -Path $binPath -Force | Out-Null
            }
            
            # Use go install as fallback for Windows
            Write-Info "Using go install method for Windows..."
            & go install github.com/golangci/golangci-lint/cmd/golangci-lint@$version
            
            if ($LASTEXITCODE -eq 0) {
                Write-Success "golangci-lint installed successfully"
                return $true
            } else {
                throw "go install failed with exit code $LASTEXITCODE"
            }
        } else {
            # Unix-like systems
            & bash -c "curl -sSfL $installScript | sh -s -- -b `$(go env GOPATH)/bin $version"
            if ($LASTEXITCODE -eq 0) {
                Write-Success "golangci-lint installed successfully"
                return $true
            } else {
                throw "Installation script failed with exit code $LASTEXITCODE"
            }
        }
    } catch {
        Write-Error "Failed to install golangci-lint: $_"
        return $false
    }
}

# Run performance monitoring
function Start-PerformanceMonitoring {
    param(
        [string]$ConfigFile,
        [int]$TimeoutMinutes
    )
    
    Write-Info "Starting CI performance monitoring..."
    Write-Verbose "Config file: $ConfigFile"
    Write-Verbose "Timeout: $TimeoutMinutes minutes"
    
    # Verify golangci-lint is available
    if (-not (Test-GolangCILint)) {
        throw "golangci-lint is not available and could not be installed"
    }
    
    # Get git information
    $gitInfo = Get-GitInfo
    Write-Info "Branch: $($gitInfo.Branch), Commit: $($gitInfo.Commit)"
    
    # Initialize metrics
    $metrics = [CIPerformanceMetrics]::new()
    $metrics.Timestamp = Get-Date
    $metrics.GitBranch = $gitInfo.Branch
    $metrics.GitCommit = $gitInfo.Commit
    $metrics.ConfigFile = $ConfigFile
    $metrics.LinterTimings = @{}
    $metrics.ModuleTimings = @{}
    
    try {
        # Start performance counters
        $process = $null
        $startTime = Get-Date
        
        Write-Info "Running golangci-lint with performance monitoring..."
        
        # Create performance monitoring script block
        $monitoringJob = Start-Job -ScriptBlock {
            param($ProjectRoot, $ConfigFile, $TimeoutSeconds)
            
            Set-Location $ProjectRoot
            $env:GOGC = "20" # Aggressive garbage collection for memory monitoring
            
            $startTime = Get-Date
            $timeoutTime = $startTime.AddSeconds($TimeoutSeconds)
            
            # Run golangci-lint with detailed output
            $lintArgs = @(
                "run"
                "--config=$ConfigFile"
                "--timeout=$($TimeoutSeconds)s"
                "--print-resources-usage"
                "--verbose"
                "./..."
            )
            
            try {
                $output = & golangci-lint $lintArgs 2>&1
                $exitCode = $LASTEXITCODE
                $endTime = Get-Date
                
                return @{
                    ExitCode = $exitCode
                    Output = $output
                    Duration = ($endTime - $startTime).TotalSeconds
                    Success = $exitCode -eq 0
                }
            } catch {
                $endTime = Get-Date
                return @{
                    ExitCode = -1
                    Output = $_.Exception.Message
                    Duration = ($endTime - $startTime).TotalSeconds
                    Success = $false
                }
            }
        } -ArgumentList $script:ProjectRoot, $ConfigFile, ($TimeoutMinutes * 60)
        
        # Wait for job completion with progress
        $timeout = New-TimeSpan -Minutes $TimeoutMinutes
        $elapsed = [System.Diagnostics.Stopwatch]::StartNew()
        
        Write-Info "Waiting for lint execution to complete (timeout: $TimeoutMinutes minutes)..."
        
        while ($monitoringJob.State -eq "Running" -and $elapsed.Elapsed -lt $timeout) {
            Start-Sleep -Seconds 5
            $progress = [math]::Round(($elapsed.Elapsed.TotalSeconds / $timeout.TotalSeconds) * 100, 1)
            Write-Progress -Activity "Running golangci-lint" -Status "Elapsed: $($elapsed.Elapsed.ToString('mm\:ss'))" -PercentComplete $progress
        }
        
        Write-Progress -Activity "Running golangci-lint" -Completed
        
        # Get results
        if ($monitoringJob.State -eq "Running") {
            Write-Warning "Lint execution timed out, stopping job..."
            Stop-Job $monitoringJob
            $metrics.Status = "timeout"
            $metrics.ErrorMessage = "Execution timed out after $TimeoutMinutes minutes"
            $metrics.LintExecutionTime = $TimeoutMinutes * 60
        } else {
            $result = Receive-Job $monitoringJob
            Remove-Job $monitoringJob
            
            $metrics.LintExecutionTime = $result.Duration
            $metrics.Status = if ($result.Success) { "success" } else { "failed" }
            
            if (-not $result.Success) {
                $metrics.ErrorMessage = $result.Output -join "`n"
                Write-Error "Lint execution failed: $($metrics.ErrorMessage)"
            } else {
                Write-Success "Lint execution completed successfully"
                
                # Parse output for additional metrics
                $output = $result.Output -join "`n"
                
                # Extract file count
                if ($output -match "(\d+) files") {
                    $metrics.TotalFiles = [int]$matches[1]
                }
                
                # Extract issue count
                if ($output -match "(\d+) issues") {
                    $metrics.IssuesFound = [int]$matches[1]
                } else {
                    $metrics.IssuesFound = 0
                }
                
                Write-Info "Files processed: $($metrics.TotalFiles), Issues found: $($metrics.IssuesFound)"
            }
        }
        
        # Add resource usage estimates (simplified for cross-platform compatibility)
        $metrics.MemoryUsageMB = [math]::Round((Get-Process -Id $PID).WorkingSet64 / 1MB, 2)
        $metrics.CPUUsagePercent = 0 # Placeholder - accurate CPU monitoring requires platform-specific code
        
    } catch {
        $metrics.Status = "error"
        $metrics.ErrorMessage = $_.Exception.Message
        $metrics.LintExecutionTime = (Get-Date - $startTime).TotalSeconds
        Write-Error "Performance monitoring failed: $_"
    }
    
    # Save metrics if requested
    if ($SaveMetrics) {
        Save-PerformanceMetrics -Metrics $metrics
    }
    
    # Display results
    Show-PerformanceResults -Metrics $metrics
    
    return $metrics
}

# Save performance metrics
function Save-PerformanceMetrics {
    param([CIPerformanceMetrics]$Metrics)
    
    try {
        Write-Verbose "Saving performance metrics..."
        
        # Load existing history
        $history = @()
        if (Test-Path $script:HistoryFile) {
            $historyJson = Get-Content $script:HistoryFile -Raw -ErrorAction SilentlyContinue
            if ($historyJson) {
                $history = $historyJson | ConvertFrom-Json
            }
        }
        
        # Add current metrics
        $history += $Metrics
        
        # Keep only last 100 entries to prevent file from growing too large
        if ($history.Count -gt 100) {
            $history = $history | Select-Object -Last 100
        }
        
        # Save updated history
        $history | ConvertTo-Json -Depth 10 | Set-Content $script:HistoryFile -Encoding UTF8
        
        Write-Verbose "Metrics saved to: $script:HistoryFile"
    } catch {
        Write-Warning "Failed to save performance metrics: $_"
    }
}

# Show performance results
function Show-PerformanceResults {
    param([CIPerformanceMetrics]$Metrics)
    
    Write-Host ""
    Write-Host "=================================="
    Write-Host "  CI PERFORMANCE MONITORING REPORT"
    Write-Host "=================================="
    Write-Host "Timestamp: $($Metrics.Timestamp.ToString('yyyy-MM-dd HH:mm:ss'))"
    Write-Host "Branch: $($Metrics.GitBranch)"
    Write-Host "Commit: $($Metrics.GitCommit)"
    Write-Host "Config: $($Metrics.ConfigFile)"
    Write-Host ""
    Write-Host "PERFORMANCE METRICS:"
    Write-Host "  Execution Time: $([math]::Round($Metrics.LintExecutionTime, 2)) seconds"
    Write-Host "  Files Processed: $($Metrics.TotalFiles)"
    Write-Host "  Issues Found: $($Metrics.IssuesFound)"
    Write-Host "  Memory Usage: $($Metrics.MemoryUsageMB) MB"
    Write-Host "  Status: $($Metrics.Status.ToUpper())"
    
    if ($Metrics.ErrorMessage) {
        Write-Host ""
        Write-Host "ERROR DETAILS:"
        Write-Host $Metrics.ErrorMessage
    }
    
    Write-Host ""
    
    # Performance assessment
    $executionTime = $Metrics.LintExecutionTime
    if ($executionTime -lt 60) {
        Write-Success "⚡ Excellent performance (< 1 minute)"
    } elseif ($executionTime -lt 180) {
        Write-Info "✓ Good performance (< 3 minutes)"
    } elseif ($executionTime -lt 300) {
        Write-Warning "⚠ Slow performance (< 5 minutes)"
    } else {
        Write-Error "🐌 Very slow performance (> 5 minutes)"
    }
}

# Generate performance report
function New-PerformanceReport {
    Write-Info "Generating performance report..."
    
    if (-not (Test-Path $script:HistoryFile)) {
        Write-Warning "No performance history found. Run monitoring first."
        return
    }
    
    try {
        $historyJson = Get-Content $script:HistoryFile -Raw
        $history = $historyJson | ConvertFrom-Json
        
        if ($history.Count -eq 0) {
            Write-Warning "Performance history is empty."
            return
        }
        
        # Calculate statistics
        $successfulRuns = $history | Where-Object { $_.Status -eq "success" }
        $failedRuns = $history | Where-Object { $_.Status -ne "success" }
        
        $stats = @{
            TotalRuns = $history.Count
            SuccessfulRuns = $successfulRuns.Count
            FailedRuns = $failedRuns.Count
            SuccessRate = if ($history.Count -gt 0) { [math]::Round(($successfulRuns.Count / $history.Count) * 100, 2) } else { 0 }
            AverageExecutionTime = if ($successfulRuns.Count -gt 0) { [math]::Round(($successfulRuns | Measure-Object -Property LintExecutionTime -Average).Average, 2) } else { 0 }
            MinExecutionTime = if ($successfulRuns.Count -gt 0) { [math]::Round(($successfulRuns | Measure-Object -Property LintExecutionTime -Minimum).Minimum, 2) } else { 0 }
            MaxExecutionTime = if ($successfulRuns.Count -gt 0) { [math]::Round(($successfulRuns | Measure-Object -Property LintExecutionTime -Maximum).Maximum, 2) } else { 0 }
            LastRun = $history | Select-Object -Last 1
        }
        
        # Generate report
        Write-Host ""
        Write-Host "=================================="
        Write-Host "  PERFORMANCE ANALYSIS REPORT"
        Write-Host "=================================="
        Write-Host "Total Runs: $($stats.TotalRuns)"
        Write-Host "Successful: $($stats.SuccessfulRuns)"
        Write-Host "Failed: $($stats.FailedRuns)"
        Write-Host "Success Rate: $($stats.SuccessRate)%"
        Write-Host ""
        Write-Host "EXECUTION TIME STATISTICS:"
        Write-Host "  Average: $($stats.AverageExecutionTime) seconds"
        Write-Host "  Minimum: $($stats.MinExecutionTime) seconds"
        Write-Host "  Maximum: $($stats.MaxExecutionTime) seconds"
        Write-Host ""
        
        # Recent trend analysis
        $recentRuns = $history | Select-Object -Last 10
        $recentSuccessRate = if ($recentRuns.Count -gt 0) { [math]::Round((($recentRuns | Where-Object { $_.Status -eq "success" }).Count / $recentRuns.Count) * 100, 2) } else { 0 }
        
        Write-Host "RECENT TRENDS (Last 10 runs):"
        Write-Host "  Success Rate: $recentSuccessRate%"
        
        if ($recentSuccessRate -lt 80) {
            Write-Warning "⚠ Recent success rate is below 80%"
        } else {
            Write-Success "✓ Recent performance looks good"
        }
        
        # Save detailed report
        $reportData = @{
            GeneratedAt = Get-Date
            Statistics = $stats
            RecentTrends = @{
                Last10Runs = $recentRuns
                SuccessRate = $recentSuccessRate
            }
            History = $history
        }
        
        $reportData | ConvertTo-Json -Depth 10 | Set-Content $script:ReportFile -Encoding UTF8
        Write-Info "Detailed report saved to: $script:ReportFile"
        
    } catch {
        Write-Error "Failed to generate performance report: $_"
    }
}

# Check for performance alerts
function Test-PerformanceAlerts {
    param([double]$ThresholdSeconds = 300)
    
    Write-Info "Checking for performance alerts (threshold: $ThresholdSeconds seconds)..."
    
    if (-not (Test-Path $script:HistoryFile)) {
        Write-Warning "No performance history found for alerts."
        return
    }
    
    try {
        $historyJson = Get-Content $script:HistoryFile -Raw
        $history = $historyJson | ConvertFrom-Json
        
        if ($history.Count -eq 0) {
            return
        }
        
        $lastRun = $history | Select-Object -Last 1
        $recentRuns = $history | Select-Object -Last 5
        
        $alerts = @()
        
        # Check last run performance
        if ($lastRun.LintExecutionTime -gt $ThresholdSeconds) {
            $alerts += "🚨 Last run exceeded threshold: $([math]::Round($lastRun.LintExecutionTime, 2))s > ${ThresholdSeconds}s"
        }
        
        # Check recent failure rate
        $recentFailures = ($recentRuns | Where-Object { $_.Status -ne "success" }).Count
        if ($recentFailures -gt ($recentRuns.Count / 2)) {
            $alerts += "🚨 High failure rate in recent runs: $recentFailures/$($recentRuns.Count)"
        }
        
        # Check performance degradation
        if ($recentRuns.Count -ge 3) {
            $oldAvg = ($recentRuns | Select-Object -First 2 | Measure-Object -Property LintExecutionTime -Average).Average
            $newAvg = ($recentRuns | Select-Object -Last 2 | Measure-Object -Property LintExecutionTime -Average).Average
            
            if ($newAvg -gt ($oldAvg * 1.5)) {
                $alerts += "🚨 Performance degradation detected: $([math]::Round($newAvg, 2))s vs $([math]::Round($oldAvg, 2))s average"
            }
        }
        
        # Display alerts
        if ($alerts.Count -gt 0) {
            Write-Host ""
            Write-Host "PERFORMANCE ALERTS:" -ForegroundColor Red
            foreach ($alert in $alerts) {
                Write-Warning $alert
            }
            Write-Host ""
        } else {
            Write-Success "✓ No performance alerts detected"
        }
        
        return $alerts.Count
        
    } catch {
        Write-Error "Failed to check performance alerts: $_"
        return -1
    }
}

# Show help
function Show-Help {
    Write-Host @"
CI Performance Monitoring Tool for Nephoran
==========================================

USAGE:
    .\monitor-ci-performance.ps1 [ACTION] [OPTIONS]

ACTIONS:
    run         Monitor a live CI run (default)
    report      Generate performance analysis report
    analyze     Analyze historical performance data
    alert       Check for performance alerts
    help        Show this help message

OPTIONS:
    -ConfigFile <path>          golangci-lint config file (default: .golangci.yml)
    -ReportPath <path>          Performance report output path
    -HistoryPath <path>         Performance history file path
    -TimeoutMinutes <int>       Execution timeout in minutes (default: 10)
    -AlertThreshold <seconds>   Performance alert threshold (default: 300)
    -SaveMetrics               Save metrics to history file
    -Verbose                   Enable verbose output

EXAMPLES:
    # Monitor current CI run with metrics saving
    .\monitor-ci-performance.ps1 run -SaveMetrics

    # Generate performance report
    .\monitor-ci-performance.ps1 report

    # Check for performance alerts with custom threshold
    .\monitor-ci-performance.ps1 alert -AlertThreshold 180

    # Monitor with custom config and timeout
    .\monitor-ci-performance.ps1 run -ConfigFile .golangci-custom.yml -TimeoutMinutes 15 -SaveMetrics

PERFORMANCE THRESHOLDS:
    < 60s     : Excellent performance ⚡
    < 180s    : Good performance ✓
    < 300s    : Acceptable performance ⚠
    > 300s    : Slow performance 🐌

OUTPUT FILES:
    ci-performance-history.json : Historical performance data
    ci-performance-report.json  : Detailed analysis report
"@
}

# Main execution
try {
    Push-Location $script:ProjectRoot
    
    switch ($Action.ToLower()) {
        "run" {
            $metrics = Start-PerformanceMonitoring -ConfigFile $ConfigFile -TimeoutMinutes $TimeoutMinutes
            
            # Check for alerts if metrics were saved
            if ($SaveMetrics -and $metrics.Status -eq "success") {
                Test-PerformanceAlerts -ThresholdSeconds ([double]$AlertThreshold) | Out-Null
            }
        }
        
        "report" {
            New-PerformanceReport
        }
        
        "analyze" {
            New-PerformanceReport
            Test-PerformanceAlerts -ThresholdSeconds ([double]$AlertThreshold) | Out-Null
        }
        
        "alert" {
            $alertCount = Test-PerformanceAlerts -ThresholdSeconds ([double]$AlertThreshold)
            if ($alertCount -gt 0) {
                exit 1
            }
        }
        
        "help" {
            Show-Help
        }
        
        default {
            Write-Error "Unknown action: $Action. Use 'help' for usage information."
            exit 1
        }
    }
    
} catch {
    Write-Error "Script execution failed: $_"
    exit 1
} finally {
    Pop-Location
}
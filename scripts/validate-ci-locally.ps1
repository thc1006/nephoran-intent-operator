# validate-ci-locally.ps1
# Run CI validation locally before pushing to avoid CI failures
# Usage: .\scripts\validate-ci-locally.ps1

param(
    [switch]$Quick,  # Skip long-running tests
    [switch]$FixOnly,  # Only run build, no tests
    [switch]$Verbose  # Show detailed output
)

$ErrorActionPreference = "Stop"
$script:hasErrors = $false
$startTime = Get-Date

function Write-Status {
    param([string]$Message, [string]$Type = "INFO")
    
    $color = switch ($Type) {
        "SUCCESS" { "Green" }
        "ERROR" { "Red" }
        "WARNING" { "Yellow" }
        "INFO" { "Cyan" }
        default { "White" }
    }
    
    $timestamp = Get-Date -Format "HH:mm:ss"
    Write-Host "[$timestamp] " -NoNewline -ForegroundColor Gray
    Write-Host "$Type: " -NoNewline -ForegroundColor $color
    Write-Host $Message
}

function Test-Command {
    param([string]$Command)
    try {
        if (Get-Command $Command -ErrorAction SilentlyContinue) {
            return $true
        }
        return $false
    } catch {
        return $false
    }
}

function Run-Command {
    param(
        [string]$Command,
        [string]$Description,
        [int]$TimeoutSeconds = 120
    )
    
    Write-Status "Running: $Description" "INFO"
    if ($Verbose) {
        Write-Host "  Command: $Command" -ForegroundColor Gray
    }
    
    $job = Start-Job -ScriptBlock {
        param($cmd)
        Invoke-Expression $cmd 2>&1
    } -ArgumentList $Command
    
    $result = Wait-Job $job -Timeout $TimeoutSeconds
    
    if ($null -eq $result) {
        Stop-Job $job
        Remove-Job $job
        Write-Status "Timeout after ${TimeoutSeconds}s: $Description" "ERROR"
        $script:hasErrors = $true
        return $false
    }
    
    $output = Receive-Job $job
    Remove-Job $job
    
    if ($job.State -eq "Failed" -or $LASTEXITCODE -ne 0) {
        Write-Status "Failed: $Description" "ERROR"
        if ($Verbose -and $output) {
            Write-Host $output -ForegroundColor Red
        }
        $script:hasErrors = $true
        return $false
    }
    
    Write-Status "Success: $Description" "SUCCESS"
    if ($Verbose -and $output) {
        Write-Host $output -ForegroundColor Gray
    }
    return $true
}

function Run-BuildCheck {
    Write-Host "`n========================================" -ForegroundColor Blue
    Write-Host "  CI BUILD VALIDATION" -ForegroundColor Blue
    Write-Host "========================================`n" -ForegroundColor Blue
    
    # Check Go version
    Write-Status "Checking Go version..." "INFO"
    $goVersion = go version
    Write-Host "  $goVersion" -ForegroundColor Gray
    
    # Download dependencies
    if (-not $Quick) {
        Run-Command "go mod download" "Download dependencies" 60
        Run-Command "go mod verify" "Verify dependencies" 30
    }
    
    # Build all modules
    Write-Status "Building all modules..." "INFO"
    $buildResult = Run-Command "go build ./..." "Build all packages" 180
    
    if (-not $buildResult) {
        Write-Status "Build failed! Attempting module-by-module build..." "WARNING"
        
        # Try building each major module separately
        $modules = @(
            "pkg/performance",
            "pkg/oran/e2",
            "pkg/oran/o2", 
            "pkg/clients",
            "pkg/controllers/parallel",
            "pkg/controllers/orchestration",
            "pkg/auth",
            "pkg/git",
            "pkg/llm",
            "pkg/rag"
        )
        
        foreach ($module in $modules) {
            $cmd = "go build ./$module/..."
            Run-Command $cmd "Build $module" 60
        }
    }
    
    # Build main binaries
    Write-Status "Building main binaries..." "INFO"
    Run-Command "go build -o bin/manager.exe cmd/main.go" "Build manager binary" 60
    
    if (Test-Path "cmd/porch-publisher/main.go") {
        Run-Command "go build -o bin/porch-publisher.exe cmd/porch-publisher/main.go" "Build porch-publisher binary" 60
    }
}

function Run-Tests {
    if ($FixOnly) {
        Write-Status "Skipping tests (FixOnly mode)" "INFO"
        return
    }
    
    Write-Host "`n========================================" -ForegroundColor Blue
    Write-Host "  RUNNING TESTS" -ForegroundColor Blue
    Write-Host "========================================`n" -ForegroundColor Blue
    
    if ($Quick) {
        # Run only unit tests
        Run-Command "go test -short -timeout 30s ./pkg/..." "Run quick unit tests" 60
    } else {
        # Run full test suite
        Run-Command "go test -timeout 5m ./..." "Run all tests" 300
    }
}

function Run-Linting {
    if ($FixOnly -or $Quick) {
        Write-Status "Skipping linting checks" "INFO"
        return
    }
    
    Write-Host "`n========================================" -ForegroundColor Blue
    Write-Host "  LINTING CHECKS" -ForegroundColor Blue  
    Write-Host "========================================`n" -ForegroundColor Blue
    
    if (Test-Command "golangci-lint") {
        Run-Command "golangci-lint run --timeout=5m" "Run golangci-lint" 300
    } else {
        Write-Status "golangci-lint not installed, skipping" "WARNING"
    }
}

function Run-SecurityCheck {
    if ($FixOnly -or $Quick) {
        Write-Status "Skipping security checks" "INFO"
        return
    }
    
    Write-Host "`n========================================" -ForegroundColor Blue
    Write-Host "  SECURITY CHECKS" -ForegroundColor Blue
    Write-Host "========================================`n" -ForegroundColor Blue
    
    if (Test-Command "gosec") {
        Run-Command "gosec -fmt text ./..." "Run security scan" 120
    } else {
        Write-Status "gosec not installed, skipping" "WARNING"
    }
    
    if (Test-Command "govulncheck") {
        Run-Command "govulncheck ./..." "Run vulnerability check" 60
    } else {
        Write-Status "govulncheck not installed, skipping" "WARNING"
    }
}

function Update-FixLoop {
    # Update CI_BUILD_FIX_LOOP.md with results
    $fixLoopFile = "CI_BUILD_FIX_LOOP.md"
    if (Test-Path $fixLoopFile) {
        $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        $status = if ($script:hasErrors) { "❌ FAILED" } else { "✅ PASSED" }
        
        $entry = "`n| $timestamp | Local Validation | - | $status |"
        Add-Content -Path $fixLoopFile -Value $entry
        
        Write-Status "Updated $fixLoopFile with validation result" "INFO"
    }
}

function Show-Summary {
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    Write-Host "`n========================================" -ForegroundColor Blue
    Write-Host "  VALIDATION SUMMARY" -ForegroundColor Blue
    Write-Host "========================================`n" -ForegroundColor Blue
    
    Write-Host "Duration: $($duration.ToString('mm\:ss'))" -ForegroundColor Gray
    
    if ($script:hasErrors) {
        Write-Host "`n❌ VALIDATION FAILED" -ForegroundColor Red
        Write-Host "Please fix the errors before pushing to CI" -ForegroundColor Yellow
        exit 1
    } else {
        Write-Host "`n✅ VALIDATION PASSED" -ForegroundColor Green
        Write-Host "Safe to push to CI!" -ForegroundColor Green
        exit 0
    }
}

# Main execution
try {
    Write-Host "Starting CI Validation..." -ForegroundColor Cyan
    Write-Host "Mode: $(if ($Quick) { 'Quick' } elseif ($FixOnly) { 'Fix Only' } else { 'Full' })" -ForegroundColor Gray
    
    Run-BuildCheck
    Run-Tests
    Run-Linting
    Run-SecurityCheck
    Update-FixLoop
    
} catch {
    Write-Status "Unexpected error: $_" "ERROR"
    $script:hasErrors = $true
} finally {
    Show-Summary
}
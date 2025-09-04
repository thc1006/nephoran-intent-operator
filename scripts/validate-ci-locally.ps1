<<<<<<< HEAD
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
=======
#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Replicate CI/Lint and Ubuntu CI/golangci-lint jobs locally to prevent push-fail cycles

.DESCRIPTION  
    This script replicates the exact CI validation steps locally to catch all issues
    before pushing to GitHub. It runs:
    1. CI/Lint job validation (from ci.yml)
    2. Ubuntu CI/Code Quality golangci-lint v2 (from ubuntu-ci.yml)

.PARAMETER Verbose
    Show verbose output from linting tools

.PARAMETER SkipSetup
    Skip environment setup steps (assumes tools are installed)

.PARAMETER OnlyLint
    Run only linting checks, skip tests and builds

.EXAMPLE
    .\scripts\validate-ci-locally.ps1
    Run full CI validation

.EXAMPLE  
    .\scripts\validate-ci-locally.ps1 -Verbose -OnlyLint
    Run only linting with verbose output
#>

param(
    [switch]$Verbose,
    [switch]$SkipSetup,
    [switch]$OnlyLint
)

$ErrorActionPreference = "Stop"
$OriginalLocation = Get-Location

# Color output functions
function Write-Success { param($Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Error { param($Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warning { param($Message) Write-Host "⚠️ $Message" -ForegroundColor Yellow }
function Write-Info { param($Message) Write-Host "ℹ️ $Message" -ForegroundColor Cyan }
function Write-Phase { param($Message) Write-Host "`n=== $Message ===" -ForegroundColor Yellow }

try {
    Write-Host "🚀 Starting Complete CI Validation Locally..." -ForegroundColor Cyan
    Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""

    # =============================================================================
    # Phase 0: Environment Setup
    # =============================================================================
    if (-not $SkipSetup) {
        Write-Phase "Phase 0: Environment Setup & Validation"
        
        # Check Go installation
        try {
            $goVersion = go version
            Write-Success "Go installed: $goVersion"
        } catch {
            Write-Error "Go is not installed or not in PATH"
            Write-Host "Please install Go 1.24.x and add to PATH"
            exit 1
        }

        # Check if golangci-lint is installed
        try {
            $lintVersion = golangci-lint version
            Write-Success "golangci-lint available: $lintVersion"
        } catch {
            Write-Warning "golangci-lint not found, attempting to install..."
            
            # Install golangci-lint v1.61.0 (same as CI)
            try {
                if ($IsWindows) {
                    # Windows installation
                    Invoke-WebRequest -Uri "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" -OutFile "install-golangci.sh"
                    & wsl bash install-golangci.sh -b "$(go env GOPATH)/bin" v1.61.0
                    Remove-Item "install-golangci.sh" -Force
                } else {
                    # Unix-like systems
                    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b (go env GOPATH)/bin v1.61.0
                }
                
                # Add Go bin to PATH for this session
                $env:PATH = "$(go env GOPATH)/bin;$env:PATH"
                
                $lintVersion = golangci-lint version
                Write-Success "golangci-lint installed: $lintVersion"
            } catch {
                Write-Error "Failed to install golangci-lint automatically"
                Write-Host "Please install manually: https://golangci-lint.run/usage/install/"
                exit 1
            }
        }

        # Verify we're in a Go module
        if (-not (Test-Path "go.mod")) {
            Write-Error "go.mod not found. Please run from the project root directory."
            exit 1
        }

        Write-Success "Environment setup complete"
    }

    # =============================================================================
    # Phase 1: Go Environment Validation (Replicate CI checks)
    # =============================================================================
    Write-Phase "Phase 1: Go Environment Validation"
    
    Write-Info "Go environment:"
    go env
    Write-Host ""

    # Verify Go version matches CI expectation (1.24.x)
    $goVersionOutput = go version
    if ($goVersionOutput -match "go(\d+\.\d+)") {
        $goVer = $matches[1]
        if ($goVer -like "1.24*") {
            Write-Success "Go version $goVer matches CI environment"
        } else {
            Write-Warning "Go version $goVer differs from CI (expects 1.24.x)"
        }
    }

    # =============================================================================
    # Phase 2: Dependency Management (Exact CI Replication)
    # =============================================================================
    Write-Phase "Phase 2: Dependency Management"
    
    Write-Info "Running go mod tidy..."
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go mod tidy failed"
        exit 1
    }

    Write-Info "Downloading dependencies..."
    go mod download
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go mod download failed"
        exit 1
    }

    Write-Info "Verifying dependencies..."
    go mod verify
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go mod verify failed"
        exit 1
    }

    Write-Success "Dependencies validated"

    # =============================================================================
    # Phase 3: CI/Lint Pre-build Validation Gate 
    # =============================================================================
    Write-Phase "Phase 3: CI/Lint Pre-build Validation Gate"

    # Step 1: Build all packages (exact CI replication)
    Write-Info "Step 1: Building all packages..."
    go build ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed - compilation errors detected"
        Write-Host "This typically indicates symbol collisions or missing dependencies"
        exit 1
    }
    Write-Success "[BUILD OK] ✅ All packages built successfully"

    # Step 2: Run go vet (exact CI replication)
    Write-Info "Step 2: Running go vet..."
    go vet ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Go vet failed - potential issues detected"
        exit 1
    }
    Write-Success "[VET OK] ✅ Go vet passed successfully"

    # Step 3: Compile test binaries per package (exact CI replication)
    Write-Info "Step 3: Compiling test binaries per package..."
    
    # Clean and create cache directory
    if (Test-Path ".cache/tests") {
        Remove-Item ".cache/tests" -Recurse -Force
    }
    New-Item -ItemType Directory -Path ".cache/tests" -Force | Out-Null
    
    # Counter for compiled binaries
    $compiledCount = 0
    
    # Get all Go packages
    $packages = go list ./...
    
    foreach ($pkg in $packages) {
        # Generate safe filename from package path
        $safeName = $pkg -replace '[^A-Za-z0-9]', '_'
        $outputFile = ".cache/tests/${safeName}.test"
        
        # Try to compile test binary
        go test -c $pkg -o $outputFile 2>$null
        if ($LASTEXITCODE -eq 0) {
            $compiledCount++
            Write-Host "  ✓ Compiled: $pkg → ${safeName}.test"
        } else {
            # Some packages may not have tests, which is okay
            Write-Host "  ○ Skipped: $pkg (no tests or compilation error)"
        }
    }
    
    Write-Host ""
    Write-Host "Test binary compilation complete:"
    Write-Host "  Total packages: $($packages.Count)"
    Write-Host "  Compiled binaries: $compiledCount"
    
    # List generated files for debugging
    if (Test-Path ".cache/tests" -PathType Container) {
        $testFiles = Get-ChildItem ".cache/tests" -File
        if ($testFiles.Count -gt 0) {
            Write-Host ""
            Write-Host "Generated test binaries in .cache/tests/:"
            foreach ($file in $testFiles) {
                Write-Host "  $($file.Name) ($($file.Length) bytes)"
            }
            Write-Host "[TEST BINARIES BUILT: $($testFiles.Count) files]"
        } else {
            Write-Host "[COMPILE-ONLY TESTS OK] No test binaries generated (packages may lack tests)"
        }
    }

    # Step 4: Guard against stub files without proper build tags
    Write-Host ""
    Write-Info "Step 4: Checking for stub files with incorrect build tags..."
    
    $stubFiles = Get-ChildItem -Recurse -Include "*missing_types*.go", "*stub*.go" | Where-Object { $_.DirectoryName -notlike "*vendor*" }
    
    foreach ($stubFile in $stubFiles) {
        $firstThreeLines = Get-Content $stubFile.FullName -TotalCount 3 -ErrorAction SilentlyContinue
        $hasBuildTag = $firstThreeLines | Where-Object { $_ -match "//go:build.*stub|// \+build.*stub" }
        
        if (-not $hasBuildTag) {
            Write-Error "Stub file $($stubFile.FullName) lacks proper build tag constraint"
            Write-Host "Add '//go:build <tag>_stub' to prevent default compilation"
            exit 1
        }
    }
    Write-Success "✅ All stub files have proper build constraints"
    
    Write-Success "=== ✅ Build/vet gate passed - proceeding with lint ==="

    # =============================================================================
    # Phase 4: Ubuntu CI - golangci-lint v2 Configuration Verification
    # =============================================================================
    Write-Phase "Phase 4: Ubuntu CI - golangci-lint v2 Configuration Verification"
    
    # Check if .golangci.yml exists
    if (Test-Path ".golangci.yml") {
        Write-Info "Verifying golangci-lint config (v2 schema)..."
        golangci-lint config verify --config=.golangci.yml
        if ($LASTEXITCODE -ne 0) {
            Write-Error "golangci-lint configuration verification failed"
            exit 1
        }
        Write-Success "golangci-lint configuration verified"
    } else {
        Write-Warning "No .golangci.yml found - using default configuration"
    }

    # =============================================================================
    # Phase 5: golangci-lint Execution (Both CI Jobs)
    # =============================================================================
    Write-Phase "Phase 5: golangci-lint Execution (Replicating Both CI Jobs)"
    
    # Job 1: CI/Lint (from ci.yml) - with 10m timeout, github-actions format
    Write-Info "Running golangci-lint (CI/Lint job replication)..."
    if ($Verbose) {
        golangci-lint run --timeout=10m --out-format=github-actions --verbose
    } else {
        golangci-lint run --timeout=10m --out-format=github-actions
    }
    
    $ciLintResult = $LASTEXITCODE
    
    # Job 2: Ubuntu CI/Code Quality (from ubuntu-ci.yml) - with 5m timeout
    Write-Info "Running golangci-lint (Ubuntu CI job replication)..."
    if ($Verbose) {
        golangci-lint run --timeout=5m --verbose
    } else {
        golangci-lint run --timeout=5m
    }
    
    $ubuntuLintResult = $LASTEXITCODE
    
    # Check both results
    if ($ciLintResult -ne 0) {
        Write-Error "CI/Lint job would FAIL - golangci-lint issues detected (10m timeout)"
        $hasLintErrors = $true
    } else {
        Write-Success "CI/Lint job would PASS ✅"
    }
    
    if ($ubuntuLintResult -ne 0) {
        Write-Error "Ubuntu CI/Code Quality job would FAIL - golangci-lint issues detected (5m timeout)"
        $hasLintErrors = $true
    } else {
        Write-Success "Ubuntu CI/Code Quality job would PASS ✅"
    }

    if ($hasLintErrors) {
        Write-Host ""
        Write-Error "LINT VALIDATION FAILED - Issues detected that will cause CI to fail"
        Write-Warning "Fix the above linting issues before pushing to avoid CI failures"
        exit 1
    }

    Write-Success "✅ Both golangci-lint jobs would PASS"

    # =============================================================================
    # Phase 6: Additional Validations (if not OnlyLint)
    # =============================================================================
    if (-not $OnlyLint) {
        Write-Phase "Phase 6: Additional Build & Test Validations"
        
        # Test execution
        Write-Info "Running tests with coverage..."
        if (Test-Path "test-results") {
            Remove-Item "test-results" -Recurse -Force
        }
        New-Item -ItemType Directory -Path "test-results" -Force | Out-Null
        
        go test -v -race -coverprofile=test-results/coverage.out -covermode=atomic ./...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Tests failed"
            exit 1
        }
        Write-Success "Tests passed"

        # Generate coverage report
        if (Test-Path "test-results/coverage.out") {
            Write-Info "Generating coverage report..."
            go tool cover -html=test-results/coverage.out -o test-results/coverage.html
            $coverage = go tool cover -func=test-results/coverage.out | Select-String "total:"
            Write-Success "Coverage report generated: $coverage"
        }

        # Final build verification
        Write-Info "Final build verification..."
        if (Test-Path "bin") {
            Remove-Item "bin" -Recurse -Force
        }
        New-Item -ItemType Directory -Path "bin" -Force | Out-Null
        
        go build -o bin/ ./cmd/...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Final build failed"
            exit 1
        }
        
        # Verify executables
        $executables = Get-ChildItem "bin" -File
        foreach ($exe in $executables) {
            if ($exe.Length -gt 0) {
                Write-Success "$($exe.Name) is executable ($($exe.Length) bytes)"
            } else {
                Write-Error "$($exe.Name) has zero size"
                exit 1
            }
        }
    }

    # =============================================================================
    # Final Success Report
    # =============================================================================
    Write-Host ""
    Write-Host "🎉 ALL CI VALIDATION PASSED LOCALLY! 🎉" -ForegroundColor Green
    Write-Host "=" * 50 -ForegroundColor Green
    Write-Host ""
    Write-Success "✅ CI/Lint job would PASS"
    Write-Success "✅ Ubuntu CI/Code Quality (golangci-lint v2) job would PASS"
    if (-not $OnlyLint) {
        Write-Success "✅ Tests would PASS" 
        Write-Success "✅ Build would PASS"
    }
    Write-Host ""
    Write-Host "🚀 SAFE TO PUSH! All validation checks completed successfully." -ForegroundColor Cyan
    Write-Host "Completed at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray

} catch {
    Write-Host ""
    Write-Error "Validation failed with error: $($_.Exception.Message)"
    Write-Host "Stack trace:" -ForegroundColor Red
    Write-Host $_.ScriptStackTrace -ForegroundColor Red
    exit 1
} finally {
    Set-Location $OriginalLocation
>>>>>>> origin/integrate/mvp
}
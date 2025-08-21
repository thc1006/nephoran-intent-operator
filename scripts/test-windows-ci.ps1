#!/usr/bin/env pwsh
# test-windows-ci.ps1 - Local Windows CI testing script
# Simulates the enhanced Windows CI pipeline locally

param(
    [switch]$SkipBuild,
    [switch]$SkipTests,
    [switch]$DebugMode,
    [int]$TimeoutMinutes = 30,
    [string]$TestSuite = "all"  # all, core, loop, security, integration
)

# Script configuration
$ErrorActionPreference = "Continue"
$ProgressPreference = "SilentlyContinue"

# Colors for output
$Colors = @{
    Header = "Cyan"
    Success = "Green"
    Warning = "Yellow"
    Error = "Red"
    Info = "White"
}

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "Info")
    Write-Host $Message -ForegroundColor $Colors[$Color]
}

function Test-Prerequisites {
    Write-ColorOutput "=== Checking Prerequisites ===" "Header"
    
    # Check Go installation
    try {
        $goVersion = go version
        Write-ColorOutput "✅ Go: $goVersion" "Success"
    } catch {
        Write-ColorOutput "❌ Go not found. Please install Go 1.24+" "Error"
        return $false
    }
    
    # Check PowerShell version
    $psVersion = $PSVersionTable.PSVersion
    if ($psVersion.Major -ge 5) {
        Write-ColorOutput "✅ PowerShell: $psVersion" "Success"
    } else {
        Write-ColorOutput "❌ PowerShell 5.1+ required. Current: $psVersion" "Error"
        return $false
    }
    
    # Check project structure
    $requiredFiles = @("go.mod", "go.sum", "cmd\conductor-loop\main.go")
    foreach ($file in $requiredFiles) {
        if (Test-Path $file) {
            Write-ColorOutput "✅ Found: $file" "Success"
        } else {
            Write-ColorOutput "❌ Missing: $file" "Error"
            return $false
        }
    }
    
    return $true
}

function Setup-Environment {
    Write-ColorOutput "=== Setting Up Environment ===" "Header"
    
    # Create temp directories (similar to CI)
    $script:TempBase = Join-Path $env:TEMP "nephoran-local-ci-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    $script:TempGo = Join-Path $TempBase "go"
    $script:TempTest = Join-Path $TempBase "test"
    $script:TempCache = Join-Path $TempBase "cache"
    
    @($TempBase, $TempGo, $TempTest, $TempCache) | ForEach-Object {
        New-Item -ItemType Directory -Force -Path $_ | Out-Null
        Write-ColorOutput "Created: $_" "Info"
    }
    
    # Set environment variables
    $env:TMPDIR = $TempGo
    $env:GOCACHE = $TempCache
    $env:GOTMPDIR = $TempGo
    $env:TEST_TEMP_DIR = $TempTest
    $env:CGO_ENABLED = "0"
    $env:GOMAXPROCS = "4"
    $env:GOTRACEBACK = "all"
    $env:GO_TEST_TIMEOUT_SCALE = "2"
    
    Write-ColorOutput "Environment configured:" "Info"
    Write-ColorOutput "  Temp Base: $TempBase" "Info"
    Write-ColorOutput "  Go Cache: $TempCache" "Info"
    Write-ColorOutput "  Test Dir: $TempTest" "Info"
    
    return $true
}

function Test-Dependencies {
    Write-ColorOutput "=== Testing Dependencies ===" "Header"
    
    try {
        Write-ColorOutput "Downloading dependencies..." "Info"
        go mod download
        go mod verify
        Write-ColorOutput "✅ Dependencies verified" "Success"
        return $true
    } catch {
        Write-ColorOutput "❌ Dependency verification failed: $_" "Error"
        return $false
    }
}

function Build-Executables {
    if ($SkipBuild) {
        Write-ColorOutput "=== Skipping Build (--SkipBuild specified) ===" "Warning"
        return $true
    }
    
    Write-ColorOutput "=== Building Windows Executables ===" "Header"
    
    $buildTargets = @(
        @{name="conductor-loop"; path="./cmd/conductor-loop"; binary="conductor-loop.exe"},
        @{name="conductor-watch"; path="./cmd/conductor-watch"; binary="conductor-watch.exe"},
        @{name="intent-ingest"; path="./cmd/intent-ingest"; binary="intent-ingest.exe"}
    )
    
    $allBuildsSucceeded = $true
    
    foreach ($target in $buildTargets) {
        Write-ColorOutput "Building $($target.name)..." "Info"
        $buildStart = Get-Date
        
        try {
            # Build with optimizations
            go build -v -ldflags="-s -w" -o $target.binary $target.path 2>&1 | 
                Tee-Object -FilePath "build-$($target.name).log"
            
            if (Test-Path $target.binary) {
                $buildEnd = Get-Date
                $buildTime = ($buildEnd - $buildStart).TotalSeconds
                $fileSize = [math]::Round((Get-Item $target.binary).Length / 1MB, 2)
                
                Write-ColorOutput "✅ $($target.name): $fileSize MB in $([math]::Round($buildTime, 2))s" "Success"
                
                # Test executable
                $helpOutput = & ".\$($target.binary)" --help 2>&1 | Select-Object -First 3
                Write-ColorOutput "  Help output: $($helpOutput -join '; ')" "Info"
            } else {
                throw "Binary not found after build"
            }
        } catch {
            Write-ColorOutput "❌ Build failed for $($target.name): $_" "Error"
            $allBuildsSucceeded = $false
        }
    }
    
    return $allBuildsSucceeded
}

function Run-TestSuite {
    param([string]$SuiteName, [hashtable]$SuiteConfig)
    
    Write-ColorOutput "=== Testing $SuiteName ===" "Header"
    
    $maxRetries = 3
    $attempt = 1
    $success = $false
    
    while ($attempt -le $maxRetries -and -not $success) {
        Write-ColorOutput "Attempt $attempt of $maxRetries for $SuiteName..." "Info"
        
        try {
            # Create isolated test directory
            $testDir = Join-Path $env:TEST_TEMP_DIR $SuiteName
            New-Item -ItemType Directory -Force -Path $testDir | Out-Null
            
            # Set test environment
            $oldTmpDir = $env:TMPDIR
            $env:TMPDIR = $testDir
            
            $coverageFile = "coverage-$SuiteName.out"
            $testLogFile = "test-$SuiteName-$attempt.log"
            
            Write-ColorOutput "Running: go test $($SuiteConfig.flags) $($SuiteConfig.packages)" "Info"
            
            # Run tests with timeout
            $testProcess = Start-Process -FilePath "go" -ArgumentList @(
                "test", "-v", "-count=1", "-timeout=$($SuiteConfig.timeout)",
                "-coverprofile=$coverageFile"
            ) + $SuiteConfig.flags.Split() + $SuiteConfig.packages.Split() -NoNewWindow -Wait -PassThru -RedirectStandardOutput $testLogFile -RedirectStandardError "error-$SuiteName-$attempt.log"
            
            if ($testProcess.ExitCode -eq 0) {
                Write-ColorOutput "✅ $SuiteName tests passed on attempt $attempt" "Success"
                $success = $true
                
                # Generate coverage report
                if (Test-Path $coverageFile) {
                    $coverage = go tool cover -func=$coverageFile | Select-String "total:" | ForEach-Object { $_.ToString().Split()[-1] }
                    Write-ColorOutput "  Coverage: $coverage" "Info"
                    go tool cover -html=$coverageFile -o "coverage-$SuiteName.html"
                }
            } else {
                throw "Tests failed with exit code $($testProcess.ExitCode)"
            }
            
            # Restore environment
            $env:TMPDIR = $oldTmpDir
            
        } catch {
            Write-ColorOutput "❌ $SuiteName test attempt $attempt failed: $_" "Error"
            if ($attempt -eq $maxRetries) {
                Write-ColorOutput "❌ All attempts failed for $SuiteName" "Error"
                return $false
            } else {
                Write-ColorOutput "Retrying in 5 seconds..." "Warning"
                Start-Sleep -Seconds 5
            }
        }
        $attempt++
    }
    
    return $success
}

function Run-Tests {
    if ($SkipTests) {
        Write-ColorOutput "=== Skipping Tests (--SkipTests specified) ===" "Warning"
        return $true
    }
    
    Write-ColorOutput "=== Running Test Suites ===" "Header"
    
    # Define test suites (matching CI configuration)
    $testSuites = @{
        "core" = @{
            packages = "./api/... ./pkg/config/... ./pkg/auth/... ./internal/ingest/..."
            flags = "-short"
            timeout = "10m"
        }
        "loop" = @{
            packages = "./internal/loop/... ./cmd/conductor-loop/... ./cmd/conductor-watch/..."
            flags = ""
            timeout = "20m"
        }
        "security" = @{
            packages = "./pkg/security/... ./internal/security/..."
            flags = "-short"
            timeout = "10m"
        }
        "integration" = @{
            packages = "./cmd/... -run=TestIntegration"
            flags = ""
            timeout = "15m"
        }
    }
    
    # Generate deepcopy files if needed
    if (-not (Test-Path "api\v1\zz_generated.deepcopy.go")) {
        Write-ColorOutput "Installing controller-gen..." "Info"
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
        
        Write-ColorOutput "Generating deepcopy methods..." "Info"
        controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1"
    }
    
    $allTestsSucceeded = $true
    
    # Run specified test suite(s)
    if ($TestSuite -eq "all") {
        foreach ($suiteName in $testSuites.Keys) {
            $success = Run-TestSuite -SuiteName $suiteName -SuiteConfig $testSuites[$suiteName]
            if (-not $success) {
                $allTestsSucceeded = $false
            }
        }
    } elseif ($testSuites.ContainsKey($TestSuite)) {
        $allTestsSucceeded = Run-TestSuite -SuiteName $TestSuite -SuiteConfig $testSuites[$TestSuite]
    } else {
        Write-ColorOutput "❌ Unknown test suite: $TestSuite" "Error"
        Write-ColorOutput "Available suites: $($testSuites.Keys -join ', ')" "Info"
        return $false
    }
    
    return $allTestsSucceeded
}

function Run-E2ETests {
    Write-ColorOutput "=== Running E2E Tests ===" "Header"
    
    # Create E2E test directories
    $e2eDir = Join-Path $env:TEST_TEMP_DIR "e2e"
    @("handoff", "out", "porch-output") | ForEach-Object {
        New-Item -ItemType Directory -Force -Path (Join-Path $e2eDir $_) | Out-Null
    }
    
    # Create test intent file
    $testIntent = @{
        apiVersion = "network.nephio.io/v1alpha1"
        kind = "NetworkIntent"
        metadata = @{
            name = "local-e2e-test"
            namespace = "default"
        }
        spec = @{
            action = "scale"
            target = "ran-du-local"
            parameters = @{
                replicas = 3
                cpu = "1000m"
                memory = "2Gi"
            }
        }
    }
    
    $testIntent | ConvertTo-Json -Depth 10 | Out-File -FilePath (Join-Path $e2eDir "handoff\test-intent.json") -Encoding UTF8
    
    # Test conductor-loop if it exists
    if (Test-Path "conductor-loop.exe") {
        Write-ColorOutput "Testing conductor-loop..." "Info"
        
        try {
            $process = Start-Process -FilePath ".\conductor-loop.exe" -ArgumentList @(
                "--handoff", (Join-Path $e2eDir "handoff"),
                "--out", (Join-Path $e2eDir "out"),
                "--once"
            ) -NoNewWindow -Wait -PassThru -RedirectStandardOutput "e2e-once.log" -RedirectStandardError "e2e-once-error.log"
            
            if ($process.ExitCode -eq 0) {
                Write-ColorOutput "✅ E2E test passed" "Success"
                
                # Check output files
                $outputFiles = Get-ChildItem (Join-Path $e2eDir "out") -Filter "*.yaml" -ErrorAction SilentlyContinue
                if ($outputFiles.Count -gt 0) {
                    Write-ColorOutput "✅ Generated $($outputFiles.Count) output files" "Success"
                } else {
                    Write-ColorOutput "⚠️ No output files generated" "Warning"
                }
                return $true
            } else {
                Write-ColorOutput "❌ E2E test failed with exit code $($process.ExitCode)" "Error"
                return $false
            }
        } catch {
            Write-ColorOutput "❌ E2E test failed: $_" "Error"
            return $false
        }
    } else {
        Write-ColorOutput "⚠️ conductor-loop.exe not found, skipping E2E test" "Warning"
        return $true
    }
}

function Generate-Report {
    Write-ColorOutput "=== Generating Test Report ===" "Header"
    
    $reportFile = "windows-ci-test-report.md"
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    
    $report = @"
# Windows CI Test Report

**Generated:** $timestamp  
**PowerShell Version:** $($PSVersionTable.PSVersion)  
**Go Version:** $(go version)  
**Test Suite:** $TestSuite  

## Environment
- **Temp Base:** $TempBase
- **CGO Enabled:** $env:CGO_ENABLED
- **Max Procs:** $env:GOMAXPROCS
- **Test Timeout Scale:** $env:GO_TEST_TIMEOUT_SCALE

## Build Artifacts
"@
    
    Get-ChildItem "*.exe" -ErrorAction SilentlyContinue | ForEach-Object {
        $size = [math]::Round($_.Length / 1MB, 2)
        $report += "`n- **$($_.Name):** $size MB"
    }
    
    $report += "`n`n## Test Results"
    Get-ChildItem "test-*.log" -ErrorAction SilentlyContinue | ForEach-Object {
        $lines = (Get-Content $_.FullName | Measure-Object -Line).Lines
        $report += "`n- **$($_.Name):** $lines lines"
    }
    
    $report += "`n`n## Coverage Reports"
    Get-ChildItem "coverage-*.out" -ErrorAction SilentlyContinue | ForEach-Object {
        if (Test-Path $_.FullName) {
            try {
                $coverage = go tool cover -func=$_.FullName | Select-String "total:" | ForEach-Object { $_.ToString().Split()[-1] }
                $report += "`n- **$($_.BaseName):** $coverage"
            } catch {
                $report += "`n- **$($_.BaseName):** Error reading coverage"
            }
        }
    }
    
    $report += "`n`n## Log Files"
    Get-ChildItem "*.log" -ErrorAction SilentlyContinue | ForEach-Object {
        $report += "`n- $($_.Name)"
    }
    
    $report | Out-File -FilePath $reportFile -Encoding UTF8
    Write-ColorOutput "Report saved to: $reportFile" "Success"
}

function Cleanup {
    Write-ColorOutput "=== Cleanup ===" "Header"
    
    if (Test-Path $TempBase) {
        try {
            Remove-Item -Path $TempBase -Recurse -Force -ErrorAction SilentlyContinue
            Write-ColorOutput "✅ Cleaned up temp directory: $TempBase" "Success"
        } catch {
            Write-ColorOutput "⚠️ Could not clean up temp directory: $_" "Warning"
        }
    }
    
    # Clean up any leftover processes
    Get-Process | Where-Object { $_.Name -match "conductor|intent" } | ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
            Write-ColorOutput "Stopped process: $($_.Name) ($($_.Id))" "Info"
        } catch {
            # Ignore cleanup errors
        }
    }
}

# Main execution
function Main {
    $startTime = Get-Date
    
    Write-ColorOutput "🪟 Windows CI Local Testing Script" "Header"
    Write-ColorOutput "Started at: $startTime" "Info"
    Write-ColorOutput "Parameters: SkipBuild=$SkipBuild, SkipTests=$SkipTests, TestSuite=$TestSuite" "Info"
    Write-ColorOutput "" "Info"
    
    try {
        # Run all phases
        $phases = @(
            @{name="Prerequisites"; func={Test-Prerequisites}},
            @{name="Environment"; func={Setup-Environment}},
            @{name="Dependencies"; func={Test-Dependencies}},
            @{name="Build"; func={Build-Executables}},
            @{name="Tests"; func={Run-Tests}},
            @{name="E2E"; func={Run-E2ETests}}
        )
        
        $allSucceeded = $true
        foreach ($phase in $phases) {
            $phaseStart = Get-Date
            $success = & $phase.func
            $phaseEnd = Get-Date
            $phaseTime = ($phaseEnd - $phaseStart).TotalSeconds
            
            if ($success) {
                Write-ColorOutput "✅ $($phase.name) completed in $([math]::Round($phaseTime, 2))s" "Success"
            } else {
                Write-ColorOutput "❌ $($phase.name) failed after $([math]::Round($phaseTime, 2))s" "Error"
                $allSucceeded = $false
                if (-not $DebugMode) {
                    break
                }
            }
            Write-ColorOutput "" "Info"
        }
        
        # Generate report
        Generate-Report
        
        # Final summary
        $endTime = Get-Date
        $totalTime = ($endTime - $startTime).TotalMinutes
        
        Write-ColorOutput "=== Final Summary ===" "Header"
        Write-ColorOutput "Total time: $([math]::Round($totalTime, 2)) minutes" "Info"
        
        if ($allSucceeded) {
            Write-ColorOutput "✅ All phases completed successfully!" "Success"
            exit 0
        } else {
            Write-ColorOutput "❌ Some phases failed. Check logs for details." "Error"
            exit 1
        }
        
    } finally {
        Cleanup
    }
}

# Script entry point
if ($MyInvocation.InvocationName -ne '.') {
    Main
}
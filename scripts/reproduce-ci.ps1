# Comprehensive CI Reproduction Script
# Replicates the exact CI pipeline locally with all checks and optimizations

param(
    [ValidateSet("dependency-security", "build-and-quality", "testing", "linting", "all")]
    [string]$Job = "all",
    [switch]$Fast = $false,
    [switch]$SkipSecurity = $false,
    [switch]$Parallel = $false,
    [switch]$ActMode = $false,
    [string]$GoVersion = "1.24.6",
    [string]$LogDir = "ci-results"
)

$ErrorActionPreference = "Continue"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Cyan = "Cyan"
$Blue = "Blue"
$Magenta = "Magenta"

# CI Environment Variables (matching CI)
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"
$env:GOVULNCHECK_VERSION = "latest"
$env:CONTROLLER_GEN_VERSION = "v0.19.0"
$env:GOLANGCI_LINT_VERSION = "v1.64.3"
$env:ENVTEST_K8S_VERSION = "1.31.0"

# Fast mode overrides
if ($Fast) {
    $env:FAST_MODE = "true"
    Write-Host "⚡ Fast mode enabled - skipping slow operations" -ForegroundColor $Yellow
}

if ($SkipSecurity) {
    $env:SKIP_SECURITY = "true"
    Write-Host "🔓 Security checks disabled" -ForegroundColor $Yellow
}

# Create results directory
if (-not (Test-Path $LogDir)) {
    New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
}

$startTime = Get-Date
$results = @{}

function Write-StepHeader {
    param($Title, $Color = $Cyan)
    
    $border = "=" * 60
    Write-Host ""
    Write-Host $border -ForegroundColor $Color
    Write-Host "🎯 $Title" -ForegroundColor $Color
    Write-Host $border -ForegroundColor $Color
    Write-Host ""
}

function Write-JobResult {
    param($JobName, $Success, $Duration, $Output = "")
    
    $status = if ($Success) { "✅ PASSED" } else { "❌ FAILED" }
    $color = if ($Success) { $Green } else { $Red }
    
    Write-Host ""
    Write-Host "🏁 $JobName - $status ($($Duration.TotalSeconds.ToString('F2'))s)" -ForegroundColor $color
    
    $results[$JobName] = @{
        Success = $Success
        Duration = $Duration
        Output = $Output
    }
    
    # Log to file
    $logFile = Join-Path $LogDir "$($JobName.ToLower() -replace '[^a-z0-9]', '-').log"
    @"
=== $JobName Results ===
Status: $(if ($Success) { "PASSED" } else { "FAILED" })
Duration: $($Duration.TotalSeconds.ToString('F2'))s
Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')

$Output
"@ | Out-File -FilePath $logFile -Encoding utf8
}

function Invoke-JobStep {
    param(
        [string]$Name,
        [scriptblock]$ScriptBlock,
        [switch]$Required = $true
    )
    
    Write-Host "🔄 $Name..." -ForegroundColor $Blue
    $stepStart = Get-Date
    $success = $false
    $output = ""
    
    try {
        $output = & $ScriptBlock 2>&1 | Out-String
        $success = $LASTEXITCODE -eq 0 -or $LASTEXITCODE -eq $null
    } catch {
        $output = $_.Exception.Message
        $success = $false
    }
    
    $stepEnd = Get-Date
    $stepDuration = $stepEnd - $stepStart
    
    if ($success) {
        Write-Host "  ✅ $Name completed ($($stepDuration.TotalSeconds.ToString('F2'))s)" -ForegroundColor $Green
    } else {
        Write-Host "  ❌ $Name failed ($($stepDuration.TotalSeconds.ToString('F2'))s)" -ForegroundColor $Red
        if ($output) {
            Write-Host "  Error: $($output.Trim())" -ForegroundColor $Red
        }
        
        if ($Required) {
            throw "$Name failed and is required"
        }
    }
    
    return @{ Success = $success; Output = $output; Duration = $stepDuration }
}

function Job-DependencySecurity {
    Write-StepHeader "DEPENDENCY SECURITY & INTEGRITY"
    $jobStart = Get-Date
    $allStepsSucceeded = $true
    $jobOutput = ""
    
    try {
        # Setup Go
        $result = Invoke-JobStep "Setup Go $GoVersion" {
            go version
            if ((go version) -notmatch $GoVersion.Split('.')[0..1] -join '.') {
                throw "Go version mismatch. Expected: $GoVersion, Found: $(go version)"
            }
        }
        $jobOutput += "Go Setup: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Download dependencies
        $result = Invoke-JobStep "Download Go dependencies" {
            Write-Host "📦 Downloading and caching all dependencies..."
            go mod download -x
            go mod verify
        }
        $jobOutput += "Dependencies: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Vulnerability scanning
        if ($env:SKIP_SECURITY -ne "true") {
            $result = Invoke-JobStep "Vulnerability scanning" {
                if (-not (Get-Command govulncheck -ErrorAction SilentlyContinue)) {
                    Write-Host "📥 Installing govulncheck..."
                    go install "golang.org/x/vuln/cmd/govulncheck@$env:GOVULNCHECK_VERSION"
                }
                Write-Host "🛡️ Running vulnerability scan..."
                govulncheck ./...
            } -Required:$false
            $jobOutput += "Vulnerability Scan: $($result.Output)`n"
            # Don't fail job for vulnerability warnings
        } else {
            Write-Host "🔓 Vulnerability scanning skipped" -ForegroundColor $Yellow
            $jobOutput += "Vulnerability Scan: Skipped`n"
        }
        
    } catch {
        $allStepsSucceeded = $false
        $jobOutput += "Error: $($_.Exception.Message)`n"
    }
    
    $jobDuration = (Get-Date) - $jobStart
    Write-JobResult "Dependency Security" $allStepsSucceeded $jobDuration $jobOutput
    return $allStepsSucceeded
}

function Job-BuildAndQuality {
    Write-StepHeader "BUILD & CODE QUALITY"
    $jobStart = Get-Date
    $allStepsSucceeded = $true
    $jobOutput = ""
    
    try {
        # Install build tools
        $result = Invoke-JobStep "Install build tools" {
            if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" -Force }
            
            # Install controller-gen
            if (-not (Test-Path "bin/controller-gen.exe")) {
                Write-Host "📥 Installing controller-gen $env:CONTROLLER_GEN_VERSION..."
                $env:GOBIN = (Resolve-Path "bin").Path
                go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$env:CONTROLLER_GEN_VERSION"
            }
            
            if (Test-Path "bin/controller-gen.exe") {
                & "bin/controller-gen.exe" --version
            } else {
                throw "controller-gen installation failed"
            }
        }
        $jobOutput += "Build Tools: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Code generation
        $result = Invoke-JobStep "Code generation and verification" {
            Write-Host "🏗️ Generating code and manifests..."
            
            # Generate code
            & "bin/controller-gen.exe" object:headerFile="hack/boilerplate.go.txt" paths="./..."
            & "bin/controller-gen.exe" rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
            
            # Check for changes
            $gitStatus = git status --porcelain 2>$null
            if ($gitStatus) {
                Write-Host "Generated files changed:"
                $gitStatus | ForEach-Object { Write-Host "  $_" }
                throw "Generated files are not up to date. Run 'make generate manifests' locally."
            }
        }
        $jobOutput += "Code Generation: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Code formatting
        $result = Invoke-JobStep "Code formatting verification" {
            Write-Host "📝 Verifying code formatting..."
            $beforeHash = (Get-ChildItem -Recurse -Include "*.go" -Exclude "vendor/*" | Get-FileHash | Measure-Object -Property Hash -Sum).Sum
            go fmt ./...
            $afterHash = (Get-ChildItem -Recurse -Include "*.go" -Exclude "vendor/*" | Get-FileHash | Measure-Object -Property Hash -Sum).Sum
            
            if ($beforeHash -ne $afterHash) {
                throw "Code is not properly formatted. Run 'go fmt ./...' locally."
            }
        }
        $jobOutput += "Formatting: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Static analysis
        $result = Invoke-JobStep "Static analysis (go vet)" {
            go vet ./...
        }
        $jobOutput += "Static Analysis: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Build verification
        $result = Invoke-JobStep "Build verification" {
            Write-Host "🏗️ Building all packages..."
            go build -v ./...
            
            # Build main executables
            if (Test-Path "cmd") {
                Get-ChildItem "cmd" -Directory | ForEach-Object {
                    $cmdDir = $_.FullName
                    $cmdName = $_.Name
                    if (Test-Path "$cmdDir/main.go") {
                        Write-Host "🔄 Building $cmdName..."
                        go build -o "tmp/build-test-$cmdName" "./$($_.Name)"
                    }
                }
            }
        }
        $jobOutput += "Build: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
    } catch {
        $allStepsSucceeded = $false
        $jobOutput += "Error: $($_.Exception.Message)`n"
    }
    
    $jobDuration = (Get-Date) - $jobStart
    Write-JobResult "Build and Quality" $allStepsSucceeded $jobDuration $jobOutput
    return $allStepsSucceeded
}

function Job-Testing {
    Write-StepHeader "COMPREHENSIVE TESTING"
    $jobStart = Get-Date
    $allStepsSucceeded = $true
    $jobOutput = ""
    
    try {
        # Setup envtest
        $result = Invoke-JobStep "Setup envtest binaries" {
            if (-not (Get-Command setup-envtest -ErrorAction SilentlyContinue)) {
                go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
            }
            
            $kubebinPath = & setup-envtest use $env:ENVTEST_K8S_VERSION --arch=amd64 --os=windows -p path
            $env:KUBEBUILDER_ASSETS = $kubebinPath
            Write-Host "KUBEBUILDER_ASSETS=$kubebinPath"
        } -Required:$false
        $jobOutput += "Envtest Setup: $($result.Output)`n"
        
        # Run tests
        $result = Invoke-JobStep "Execute test suite" {
            $env:GOMAXPROCS = "2"
            
            if ($env:FAST_MODE -eq "true") {
                Write-Host "⚡ Fast mode: Essential tests only"
                go test -short -timeout=10m -parallel=4 ./api/... ./controllers/... ./pkg/...
            } else {
                Write-Host "🔍 Full test suite with coverage"
                if (-not (Test-Path "test-results")) { New-Item -ItemType Directory -Path "test-results" -Force }
                go test -v -race -vet=off -timeout=25m -parallel=2 -coverprofile=test-results/coverage.out -covermode=atomic ./...
                
                # Generate coverage report
                if (Test-Path "test-results/coverage.out") {
                    go tool cover -html=test-results/coverage.out -o test-results/coverage.html
                    $coverage = go tool cover -func=test-results/coverage.out | Select-String "total:" | ForEach-Object { $_.ToString() }
                    Write-Host "📈 Coverage: $coverage"
                }
            }
        }
        $jobOutput += "Tests: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Benchmark tests (optional)
        $result = Invoke-JobStep "Benchmark tests" {
            $benchmarks = go test -list=Benchmark ./... 2>$null
            if ($benchmarks -and $benchmarks.Count -gt 0) {
                Write-Host "🏃 Running benchmarks..."
                go test -bench=. -benchmem -timeout=10m ./... | Tee-Object -FilePath "test-results/benchmarks.txt"
            } else {
                Write-Host "ℹ️ No benchmark tests found"
            }
        } -Required:$false
        $jobOutput += "Benchmarks: $($result.Output)`n"
        
    } catch {
        $allStepsSucceeded = $false
        $jobOutput += "Error: $($_.Exception.Message)`n"
    }
    
    $jobDuration = (Get-Date) - $jobStart
    Write-JobResult "Testing" $allStepsSucceeded $jobDuration $jobOutput
    return $allStepsSucceeded
}

function Job-Linting {
    Write-StepHeader "ADVANCED LINTING"
    $jobStart = Get-Date
    $allStepsSucceeded = $true
    $jobOutput = ""
    
    try {
        # Check golangci-lint installation
        $result = Invoke-JobStep "Verify golangci-lint installation" {
            $version = golangci-lint version 2>$null
            if (-not $version) {
                throw "golangci-lint not found. Run: .\scripts\install-golangci-lint.ps1"
            }
            Write-Host "Using: $version"
        }
        $jobOutput += "Tool Check: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
        # Run linting
        $configFile = if ($Fast) { ".golangci-fast.yml" } else { ".golangci.yml" }
        $result = Invoke-JobStep "Execute golangci-lint" {
            $timeout = if ($Fast) { "10m" } else { "15m" }
            Write-Host "🔍 Running linting with $configFile (timeout: $timeout)..."
            golangci-lint run --config=$configFile --timeout=$timeout --verbose --out-format=colored-line-number --print-issued-lines --print-linter-name --sort-results
        }
        $jobOutput += "Linting: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
    } catch {
        $allStepsSucceeded = $false
        $jobOutput += "Error: $($_.Exception.Message)`n"
    }
    
    $jobDuration = (Get-Date) - $jobStart
    Write-JobResult "Linting" $allStepsSucceeded $jobDuration $jobOutput
    return $allStepsSucceeded
}

function Job-ActMode {
    Write-StepHeader "GITHUB ACTIONS SIMULATION (act)"
    $jobStart = Get-Date
    
    if (-not (Get-Command act -ErrorAction SilentlyContinue)) {
        Write-Host "❌ act is not installed. Run: .\scripts\install-act.ps1" -ForegroundColor $Red
        return $false
    }
    
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "❌ Docker is not available. act requires Docker to run." -ForegroundColor $Red
        return $false
    }
    
    $allStepsSucceeded = $true
    $jobOutput = ""
    
    try {
        # List available workflows
        Write-Host "📋 Available workflows:" -ForegroundColor $Blue
        $workflows = act -l 2>&1 | Out-String
        Write-Host $workflows
        $jobOutput += "Workflows: $workflows`n"
        
        # Run the main CI workflow
        $result = Invoke-JobStep "Run main-ci workflow with act" {
            Write-Host "🎭 Running GitHub Actions workflow locally..."
            $actArgs = @(
                "-W", ".github/workflows/main-ci.yml"
                "-P", "ubuntu-latest=catthehacker/ubuntu:act-latest"
                "--container-architecture", "linux/amd64"
                "--artifact-server-path", $LogDir
            )
            
            if ($Job -ne "all") {
                $actArgs += "--job"
                $actArgs += $Job
            }
            
            Write-Host "🚀 Executing: act $($actArgs -join ' ')"
            act @actArgs
        } -Required:$true
        $jobOutput += "Act Execution: $($result.Output)`n"
        $allStepsSucceeded = $allStepsSucceeded -and $result.Success
        
    } catch {
        $allStepsSucceeded = $false
        $jobOutput += "Error: $($_.Exception.Message)`n"
    }
    
    $jobDuration = (Get-Date) - $jobStart
    Write-JobResult "GitHub Actions Simulation" $allStepsSucceeded $jobDuration $jobOutput
    return $allStepsSucceeded
}

# Main Execution
Write-StepHeader "🚀 LOCAL CI REPRODUCTION STARTED" $Magenta

Write-Host "Configuration:" -ForegroundColor $Cyan
Write-Host "  Job: $Job"
Write-Host "  Fast Mode: $Fast"
Write-Host "  Skip Security: $SkipSecurity"
Write-Host "  Parallel: $Parallel"
Write-Host "  Act Mode: $ActMode"
Write-Host "  Go Version: $GoVersion"
Write-Host "  Working Directory: $(Get-Location)"
Write-Host "  Log Directory: $LogDir"
Write-Host ""

# Clean previous results
if (Test-Path "tmp") { Remove-Item "tmp" -Recurse -Force -ErrorAction SilentlyContinue }
New-Item -ItemType Directory -Path "tmp" -Force | Out-Null

$overallSuccess = $true

if ($ActMode) {
    # Use act to run GitHub Actions locally
    $success = Job-ActMode
    $overallSuccess = $overallSuccess -and $success
} else {
    # Run individual jobs
    $jobsToRun = if ($Job -eq "all") {
        @("dependency-security", "build-and-quality", "testing", "linting")
    } else {
        @($Job)
    }
    
    foreach ($jobName in $jobsToRun) {
        $success = switch ($jobName) {
            "dependency-security" { Job-DependencySecurity }
            "build-and-quality" { Job-BuildAndQuality }
            "testing" { Job-Testing }
            "linting" { Job-Linting }
            default { $false }
        }
        
        $overallSuccess = $overallSuccess -and $success
        
        if (-not $success -and -not $Parallel) {
            Write-Host "⚠️ Job $jobName failed. Stopping pipeline." -ForegroundColor $Yellow
            break
        }
    }
}

# Final Results
$totalDuration = (Get-Date) - $startTime
Write-StepHeader "🏁 CI REPRODUCTION RESULTS" $Magenta

Write-Host "📊 SUMMARY:" -ForegroundColor $Cyan
Write-Host "  Total Duration: $($totalDuration.TotalMinutes.ToString('F2')) minutes"
Write-Host "  Jobs Run: $(($results.Keys).Count)"
Write-Host "  Overall Success: $(if ($overallSuccess) { '✅ PASSED' } else { '❌ FAILED' })"

$results.GetEnumerator() | ForEach-Object {
    $status = if ($_.Value.Success) { "✅" } else { "❌" }
    $duration = $_.Value.Duration.TotalSeconds.ToString('F2')
    Write-Host "  $status $($_.Key): ${duration}s"
}

Write-Host ""
Write-Host "📁 Detailed logs saved to: $LogDir" -ForegroundColor $Blue

if (Test-Path "test-results/coverage.html") {
    Write-Host "📊 Coverage report: test-results/coverage.html" -ForegroundColor $Blue
}

Write-Host ""
if ($overallSuccess) {
    Write-Host "🎉 All CI checks passed! Ready for deployment." -ForegroundColor $Green
    exit 0
} else {
    Write-Host "❌ Some CI checks failed. Please review the logs and fix issues." -ForegroundColor $Red
    exit 1
}
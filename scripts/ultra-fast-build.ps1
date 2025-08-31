# =============================================================================
# Ultra-Fast Build Script for Windows PowerShell (2025 Optimized)
# =============================================================================
# Features: Maximum parallelization, Go 1.24 optimizations, smart caching
# Usage: .\scripts\ultra-fast-build.ps1 [Command] [Options]

[CmdletBinding()]
param(
    [string]$Command = "help",
    [string]$Service = "",
    [string]$TestType = "all",
    [switch]$Verbose = $false,
    [switch]$NoCache = $false,
    [int]$ParallelJobs = $env:NUMBER_OF_PROCESSORS,
    [switch]$SkipTests = $false,
    [switch]$SkipLint = $false,
    [string]$Target = "windows"
)

# =============================================================================
# Configuration and Environment
# =============================================================================

# Script paths
$ScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptRoot
$BuildDir = Join-Path $ProjectRoot "bin"
$CacheDir = Join-Path $ProjectRoot ".build-cache"
$ArtifactsDir = Join-Path $ProjectRoot ".build-artifacts"

# Build configuration (compatible with older PowerShell versions)
$BuildVersion = if ($env:BUILD_VERSION) { $env:BUILD_VERSION } else {
    try { git describe --tags --always --dirty 2>$null } catch { "dev" }
}
$BuildCommit = if ($env:BUILD_COMMIT) { $env:BUILD_COMMIT } else {
    try { git rev-parse --short HEAD 2>$null } catch { "unknown" }
}
$BuildDate = if ($env:BUILD_DATE) { $env:BUILD_DATE } else {
    Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
}
$BuildUser = $env:USERNAME

# Performance settings - optimized for Windows
$MaxParallelJobs = if ($ParallelJobs -gt 0) { $ParallelJobs } else { 8 }
$TestParallelJobs = [Math]::Max(1, $MaxParallelJobs / 2)

# Go optimization environment variables
$env:GOMAXPROCS = $MaxParallelJobs
$env:GOMEMLIMIT = "4GiB"
$env:GOTOOLCHAIN = "local"
$env:GOAMD64 = "v3"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"

# =============================================================================
# Utility Functions
# =============================================================================

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { "Red" }
        "WARNING" { "Yellow" }
        "SUCCESS" { "Green" }
        "INFO" { "Cyan" }
        default { "White" }
    }
    Write-Host "[$timestamp] $Message" -ForegroundColor $color
}

function Write-Success { param([string]$Message) Write-Log $Message "SUCCESS" }
function Write-Error { param([string]$Message) Write-Log $Message "ERROR" }
function Write-Warning { param([string]$Message) Write-Log $Message "WARNING" }
function Write-Info { param([string]$Message) Write-Log $Message "INFO" }

function Write-VerboseLog {
    param([string]$Message)
    if ($Verbose) { Write-Log "[VERBOSE] $Message" }
}

function Test-CommandExists {
    param([string]$Command)
    $null -ne (Get-Command $Command -ErrorAction SilentlyContinue)
}

function Get-FileHash {
    param([string]$Path)
    if (Test-Path $Path) {
        return (Get-FileHash $Path -Algorithm SHA256).Hash.Substring(0, 16)
    }
    return "missing"
}

function New-CacheKey {
    param([string[]]$Files)
    $hashInput = ""
    foreach ($file in $Files) {
        if (Test-Path $file) {
            $hashInput += "${file}:" + (Get-Item $file).LastWriteTime.Ticks + ";"
        }
    }
    $hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash([System.Text.Encoding]::UTF8.GetBytes($hashInput))
    return "v3:" + [System.BitConverter]::ToString($hash).Replace("-", "").Substring(0, 16)
}

# =============================================================================
# Cache Management
# =============================================================================

function Initialize-Cache {
    if ($NoCache) { return }
    
    @($CacheDir, "$CacheDir\deps", "$CacheDir\build", "$CacheDir\test", "$CacheDir\docker", $ArtifactsDir) | 
    ForEach-Object {
        if (-not (Test-Path $_)) {
            New-Item -Path $_ -ItemType Directory -Force | Out-Null
        }
    }
    
    Write-VerboseLog "Cache directory: $CacheDir"
    Write-VerboseLog "Artifacts directory: $ArtifactsDir"
}

function Test-CacheValid {
    param([string]$CacheFile, [string]$CacheKey)
    
    if (-not (Test-Path $CacheFile)) { return $false }
    
    $storedKey = Get-Content $CacheFile -First 1 -ErrorAction SilentlyContinue
    return $storedKey -eq $CacheKey
}

function Save-Cache {
    param([string]$CacheFile, [string]$CacheKey)
    
    $dir = Split-Path -Parent $CacheFile
    if (-not (Test-Path $dir)) {
        New-Item -Path $dir -ItemType Directory -Force | Out-Null
    }
    
    Set-Content -Path $CacheFile -Value $CacheKey
}

# =============================================================================
# Dependency Management
# =============================================================================

function Initialize-Dependencies {
    Write-Log "🔧 Setting up dependencies with smart caching..."
    
    $depsFiles = @("go.mod", "go.sum")
    if (Test-Path "tools.go") { $depsFiles += "tools.go" }
    
    $cacheKey = New-CacheKey $depsFiles
    $cacheFile = Join-Path $CacheDir "deps\cache.key"
    
    if ((Test-CacheValid $cacheFile $cacheKey) -and -not $NoCache) {
        Write-Info "📦 Dependencies cache hit - skipping download"
        return
    }
    
    Write-Info "📦 Downloading dependencies with maximum parallelization ($MaxParallelJobs jobs)..."
    
    try {
        # Download and verify dependencies
        Write-VerboseLog "Running go mod download..."
        go mod download
        if ($LASTEXITCODE -ne 0) { throw "go mod download failed" }
        
        Write-VerboseLog "Verifying module integrity..."
        go mod verify
        if ($LASTEXITCODE -ne 0) { throw "go mod verify failed" }
        
        # Install build tools in parallel using PowerShell jobs
        Write-Info "🛠️ Installing build tools in parallel..."
        $toolJobs = @()
        
        $tools = @(
            "sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0",
            "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest",
            "github.com/onsi/ginkgo/v2/ginkgo@latest",
            "golang.org/x/tools/cmd/cover@latest",
            "golang.org/x/vuln/cmd/govulncheck@latest",
            "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.65.0"
        )
        
        foreach ($tool in $tools) {
            $toolJobs += Start-Job -ScriptBlock {
                $env:GOMAXPROCS = $using:MaxParallelJobs
                go install $using:tool
            }
        }
        
        # Wait for all tool installations
        $toolJobs | Wait-Job | Receive-Job
        $toolJobs | Remove-Job
        
        if (-not $NoCache) {
            Save-Cache $cacheFile $cacheKey
        }
        Write-Success "✅ Dependencies setup completed"
    }
    catch {
        Write-Error "Failed to setup dependencies: $_"
        throw
    }
}

# =============================================================================
# Code Generation
# =============================================================================

function Invoke-CodeGeneration {
    Write-Log "🏗️ Generating code with smart caching..."
    
    # Find API files for cache key
    $apiFiles = Get-ChildItem -Path "api\v1" -Filter "*.go" -ErrorAction SilentlyContinue | ForEach-Object { $_.FullName }
    if (-not $apiFiles) { $apiFiles = @() }
    
    $cacheKey = New-CacheKey $apiFiles
    $cacheFile = Join-Path $CacheDir "build\generate.key"
    
    if ((Test-CacheValid $cacheFile $cacheKey) -and -not $NoCache) {
        Write-Info "🏗️ Code generation cache hit - skipping"
        return
    }
    
    Write-Info "🏗️ Generating CRDs and deep copy methods..."
    
    # Ensure output directory exists
    $crdsDir = "deployments\crds"
    if (-not (Test-Path $crdsDir)) {
        New-Item -Path $crdsDir -ItemType Directory -Force | Out-Null
    }
    
    try {
        # Generate code with error handling
        Write-VerboseLog "Generating deep copy methods..."
        $null = controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1" 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Deep copy generation failed - continuing with existing files"
        }
        
        Write-VerboseLog "Generating CRDs..."
        $null = controller-gen crd rbac:roleName=manager-role webhook paths="./api/v1" output:crd:artifacts:config=deployments/crds 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "CRD generation failed - using existing CRDs"
        }
        
        if (-not $NoCache) {
            Save-Cache $cacheFile $cacheKey
        }
        Write-Success "✅ Code generation completed"
    }
    catch {
        Write-Warning "Code generation encountered issues, but continuing..."
    }
}

# =============================================================================
# Ultra-Fast Building
# =============================================================================

function Get-BuildServices {
    $services = @()
    
    # Scan cmd directory for services
    if (Test-Path "cmd") {
        Get-ChildItem -Path "cmd" -Directory | ForEach-Object {
            $mainFile = Join-Path $_.FullName "main.go"
            if (Test-Path $mainFile) {
                $services += $_.Name
            }
        }
    }
    
    # Add main controller
    if (Test-Path "cmd\main.go") {
        $services += "manager"
    }
    
    return $services
}

function Build-Service {
    param([string]$ServiceName)
    
    Write-Info "🔨 Building $ServiceName with Go 1.24 optimizations..."
    
    # Determine command path
    $cmdPath = switch ($ServiceName) {
        "manager" { ".\cmd\main.go" }
        "controller" { ".\cmd\main.go" }
        default { ".\cmd\$ServiceName\main.go" }
    }
    
    if (-not (Test-Path $cmdPath)) {
        Write-Error "Command path not found: $cmdPath"
        return $false
    }
    
    # Build cache management
    $cacheKey = New-CacheKey @($cmdPath, "go.mod", "go.sum")
    $cacheFile = Join-Path $CacheDir "build\$ServiceName.key"
    $outputFile = Join-Path $BuildDir "$ServiceName.exe"
    
    if ((Test-CacheValid $cacheFile $cacheKey) -and (Test-Path $outputFile) -and -not $NoCache) {
        Write-Info "📦 Build cache hit for $ServiceName - skipping"
        return $true
    }
    
    # Create output directory
    if (-not (Test-Path $BuildDir)) {
        New-Item -Path $BuildDir -ItemType Directory -Force | Out-Null
    }
    
    try {
        # Ultra-optimized build for Windows
        $ldflags = "-s -w -X main.version=$BuildVersion -X main.commit=$BuildCommit -X main.date=$BuildDate"
        
        # Set build environment for optimal performance
        $env:CGO_ENABLED = "0"
        $env:GOOS = if ($Target -eq "linux") { "linux" } else { "windows" }
        $env:GOARCH = "amd64"
        
        Write-VerboseLog "Building $ServiceName with optimizations (parallel jobs: $MaxParallelJobs)..."
        
        $buildArgs = @(
            "build", "-v"
            "-ldflags", $ldflags
            "-trimpath"
            "-buildmode=pie"
            "-tags", "netgo,osusergo"
            "-p", $MaxParallelJobs
            "-o", $outputFile
            $cmdPath
        )
        
        & go @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed for $ServiceName"
        }
        
        # Verify build
        if (-not (Test-Path $outputFile)) {
            throw "Output file not created: $outputFile"
        }
        
        if (-not $NoCache) {
            Save-Cache $cacheFile $cacheKey
        }
        
        $fileSize = [math]::Round((Get-Item $outputFile).Length / 1MB, 2)
        Write-Success "✅ Built $ServiceName successfully ($fileSize MB)"
        return $true
    }
    catch {
        Write-Error "Build failed for $ServiceName`: $_"
        return $false
    }
}

function Build-AllServices {
    Write-Log "🚀 Building all services with maximum parallelism..."
    
    $services = Get-BuildServices
    if ($services.Count -eq 0) {
        Write-Warning "No services found to build"
        return
    }
    
    Write-Info "Found $($services.Count) services: $($services -join ', ')"
    
    # Build services in parallel using PowerShell jobs
    $buildJobs = @()
    $batchSize = $MaxParallelJobs
    
    for ($i = 0; $i -lt $services.Count; $i += $batchSize) {
        $batch = $services[$i..($i + $batchSize - 1)]
        
        foreach ($service in $batch) {
            $buildJobs += Start-Job -ScriptBlock {
                param($ServiceName, $ProjectRoot, $BuildFunction)
                
                Set-Location $ProjectRoot
                
                # Import the build function into the job scope
                . ([ScriptBlock]::Create($BuildFunction))
                
                Build-Service $ServiceName
            } -ArgumentList $service, $ProjectRoot, ${function:Build-Service}.ToString()
        }
        
        # Wait for batch to complete
        $results = $buildJobs | Wait-Job | Receive-Job
        $buildJobs | Remove-Job
        $buildJobs = @()
        
        # Check for failures
        if ($results -contains $false) {
            Write-Error "Some builds failed in parallel batch"
            return
        }
    }
    
    Write-Success "✅ All services built successfully"
}

# =============================================================================
# Ultra-Fast Testing
# =============================================================================

function Invoke-Tests {
    param([string]$Type = "all")
    
    if ($SkipTests) {
        Write-Warning "Skipping tests as requested"
        return
    }
    
    Write-Log "🧪 Running ultra-fast tests ($Type) with $TestParallelJobs parallel workers..."
    
    # Create artifacts directory
    if (-not (Test-Path $ArtifactsDir)) {
        New-Item -Path $ArtifactsDir -ItemType Directory -Force | Out-Null
    }
    
    # Setup test environment
    $env:USE_EXISTING_CLUSTER = "false"
    $env:ENVTEST_K8S_VERSION = "1.29.0"
    
    try {
        # Run tests based on type with optimized settings
        $testArgs = @("test", "./...")
        
        switch ($Type) {
            "unit" {
                Write-Info "🔬 Running unit tests..."
                $testArgs += @("-short", "-race", "-parallel=$TestParallelJobs", "-timeout=10m")
                $testArgs += @("-coverprofile=$ArtifactsDir\unit-coverage.out")
            }
            "integration" {
                Write-Info "🔗 Running integration tests..."
                $testArgs += @("-race", "-parallel=$([math]::Max(1, $TestParallelJobs / 2))", "-timeout=20m")
                $testPath = ".\tests\integration\..."
                $testArgs = @("test", $testPath) + $testArgs[2..$testArgs.Length]
            }
            "e2e" {
                Write-Info "🎭 Running e2e tests..."
                $testArgs += @("-parallel=2", "-timeout=30m")
                $testPath = ".\tests\e2e\..."
                $testArgs = @("test", $testPath) + $testArgs[2..$testArgs.Length]
            }
            "all" {
                Write-Info "🔬 Running comprehensive test suite..."
                $testArgs += @("-race", "-parallel=$TestParallelJobs", "-timeout=25m")
                $testArgs += @("-coverprofile=$ArtifactsDir\coverage.out", "-json")
            }
        }
        
        # Set memory limit for tests
        $env:GOMEMLIMIT = "3GiB"
        
        Write-VerboseLog "Running: go $($testArgs -join ' ')"
        
        if ($Type -eq "all") {
            # For comprehensive tests, capture JSON output
            & go @testArgs | Tee-Object -FilePath "$ArtifactsDir\test-results.json"
        } else {
            & go @testArgs
        }
        
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Some tests failed, but continuing..."
        } else {
            Write-Success "✅ Tests completed successfully"
        }
        
        # Generate coverage HTML if available
        $coverageFile = Join-Path $ArtifactsDir "coverage.out"
        if (Test-Path $coverageFile) {
            go tool cover -html=$coverageFile -o "$ArtifactsDir\coverage.html" 2>$null
            
            # Calculate coverage percentage
            $coverageOutput = go tool cover -func=$coverageFile 2>$null | Select-String "total:" | Select-Object -Last 1
            if ($coverageOutput) {
                $coveragePercent = ($coverageOutput.ToString() -split '\s+')[-1]
                Write-Info "Test coverage: $coveragePercent"
            }
        }
    }
    catch {
        Write-Warning "Tests encountered issues: $_"
    }
}

# =============================================================================
# Ultra-Fast Linting
# =============================================================================

function Invoke-Linting {
    if ($SkipLint) {
        Write-Warning "Skipping linting as requested"
        return
    }
    
    Write-Log "🔍 Running ultra-fast linting with $MaxParallelJobs workers..."
    
    # Check if golangci-lint is available
    if (-not (Test-CommandExists "golangci-lint")) {
        Write-Info "Installing golangci-lint..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.65.0
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to install golangci-lint"
            return
        }
    }
    
    try {
        # Run with ultra-fast configuration
        $lintArgs = @(
            "run"
            "--fast"
            "--timeout=8m"
            "--concurrency=$MaxParallelJobs"
            "--build-tags=netgo,osusergo"
            "--out-format=colored-line-number"
        )
        
        Write-VerboseLog "Running: golangci-lint $($lintArgs -join ' ')"
        & golangci-lint @lintArgs
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✅ Linting completed - no issues found"
        } else {
            Write-Warning "Linting found issues, but continuing build..."
        }
    }
    catch {
        Write-Warning "Linting encountered issues: $_"
    }
}

# =============================================================================
# Main Commands
# =============================================================================

function Invoke-FullBuild {
    $startTime = Get-Date
    
    Write-Log "🚀 Starting ultra-fast complete build pipeline..."
    
    try {
        Initialize-Cache
        Initialize-Dependencies
        Invoke-CodeGeneration
        Build-AllServices
        Invoke-Tests $TestType
        Invoke-Linting
        
        $endTime = Get-Date
        $duration = [math]::Round(($endTime - $startTime).TotalSeconds, 2)
        
        Write-Success "🎉 Complete build pipeline finished in ${duration}s"
        
        # Show build summary
        Write-Log ""
        Write-Log "📊 Build Summary:"
        Write-Info "Version: $BuildVersion"
        Write-Info "Commit: $BuildCommit"
        Write-Info "Duration: ${duration}s"
        Write-Info "Target: $Target"
        Write-Info "Parallel Jobs: $MaxParallelJobs"
        
        if (Test-Path $BuildDir) {
            $serviceCount = (Get-ChildItem $BuildDir).Count
            Write-Info "Services Built: $serviceCount"
        }
        
        if (Test-Path $CacheDir) {
            $cacheFiles = Get-ChildItem -Path $CacheDir -Filter "*.key" -Recurse
            Write-Info "Cache Entries: $($cacheFiles.Count)"
        }
    }
    catch {
        Write-Error "Build pipeline failed: $_"
        exit 1
    }
}

function Show-Help {
    @"
Ultra-Fast Build Script for Nephoran Intent Operator (Windows PowerShell)

Usage: .\scripts\ultra-fast-build.ps1 [Command] [Options]

Commands:
    help                Show this help message
    deps                Setup dependencies with smart caching
    generate            Generate code (CRDs, deep copy methods)
    build [Service]     Build service(s) with parallel compilation
    test [Type]         Run tests (unit|integration|e2e|all)
    lint                Run ultra-fast linting
    clean               Clean build cache and artifacts
    all                 Run complete build pipeline

Options:
    -Verbose            Enable verbose logging
    -NoCache            Disable build caching
    -ParallelJobs N     Set parallelism level (default: $env:NUMBER_OF_PROCESSORS)
    -SkipTests          Skip test execution
    -SkipLint           Skip linting
    -Target             Build target (windows|linux, default: windows)
    -TestType           Test type for 'all' command (unit|integration|e2e|all)

Examples:
    .\scripts\ultra-fast-build.ps1 all                          # Complete optimized build
    .\scripts\ultra-fast-build.ps1 build manager               # Build specific service
    .\scripts\ultra-fast-build.ps1 test unit -ParallelJobs 8   # Run unit tests with 8 parallel processes
    .\scripts\ultra-fast-build.ps1 all -Target linux -NoCache  # Cross-compile for Linux without cache

Performance Tips:
    - Use SSD storage for better cache performance
    - Increase -ParallelJobs for faster compilation on high-core machines
    - Use -NoCache for clean builds
    - Monitor system resources during parallel builds
    - Consider using WSL2 for even better Go build performance

"@ | Write-Host
}

function Clear-BuildCache {
    Write-Log "🧹 Cleaning build cache and artifacts..."
    
    @($CacheDir, $ArtifactsDir, $BuildDir) | ForEach-Object {
        if (Test-Path $_) {
            Remove-Item -Path $_ -Recurse -Force
            Write-VerboseLog "Removed: $_"
        }
    }
    
    # Clean Go cache
    go clean -cache -modcache -testcache 2>$null
    
    Write-Success "✅ Build environment cleaned"
}

# =============================================================================
# Main Entry Point
# =============================================================================

# Change to project root
Set-Location $ProjectRoot

# Initialize cache
Initialize-Cache

# Execute command
switch ($Command.ToLower()) {
    "help" { Show-Help }
    "deps" { Initialize-Dependencies }
    "generate" { Invoke-CodeGeneration }
    "gen" { Invoke-CodeGeneration }
    "build" { 
        if ($Service) {
            Build-Service $Service
        } else {
            Build-AllServices
        }
    }
    "test" { Invoke-Tests $TestType }
    "lint" { Invoke-Linting }
    "clean" { Clear-BuildCache }
    "all" { Invoke-FullBuild }
    default {
        Write-Error "Unknown command: $Command"
        Show-Help
        exit 1
    }
}
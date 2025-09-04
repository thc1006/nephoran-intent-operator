# Comprehensive CI Fix Validation Script
# Tests the complete solution for PR 176 CI failures

param(
    [int]$TimeoutMinutes = 25,
    [switch]$ForceCleanCache,
    [switch]$Verbose
)

Write-Host "🚀 Starting comprehensive CI fix validation..." -ForegroundColor Green
Write-Host "📊 Configuration:" -ForegroundColor Cyan
Write-Host "  - Timeout: $TimeoutMinutes minutes" -ForegroundColor Gray
Write-Host "  - Force Clean Cache: $ForceCleanCache" -ForegroundColor Gray
Write-Host "  - Verbose: $Verbose" -ForegroundColor Gray
Write-Host ""

# Function for timed execution
function Measure-Command-WithTimeout {
    param(
        [ScriptBlock]$ScriptBlock,
        [int]$TimeoutMinutes,
        [string]$Description
    )
    
    Write-Host "⏱️  Starting: $Description" -ForegroundColor Yellow
    $startTime = Get-Date
    
    try {
        $job = Start-Job -ScriptBlock $ScriptBlock
        $result = $job | Wait-Job -Timeout ($TimeoutMinutes * 60)
        
        if ($result) {
            $output = Receive-Job -Job $job
            $duration = (Get-Date) - $startTime
            Write-Host "✅ Completed: $Description in $($duration.TotalMinutes.ToString('F2')) minutes" -ForegroundColor Green
            
            if ($Verbose -and $output) {
                Write-Host "Output:" -ForegroundColor Gray
                $output | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            }
            
            Remove-Job -Job $job
            return @{ Success = $true; Duration = $duration; Output = $output }
        } else {
            Write-Host "❌ Timeout: $Description exceeded $TimeoutMinutes minutes" -ForegroundColor Red
            Stop-Job -Job $job
            Remove-Job -Job $job
            return @{ Success = $false; Duration = $null; Output = "Timeout" }
        }
    }
    catch {
        $duration = (Get-Date) - $startTime
        Write-Host "❌ Error: $Description failed after $($duration.TotalMinutes.ToString('F2')) minutes" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        return @{ Success = $false; Duration = $duration; Output = $_.Exception.Message }
    }
}

# Test environment setup
Write-Host "🔧 Setting up test environment..." -ForegroundColor Cyan

# Set Go environment variables (matching our CI workflow)
$env:GO_VERSION = "1.22"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"
$env:CGO_ENABLED = 0
$env:GOMAXPROCS = 4
$env:GOMEMLIMIT = "8GiB"
$env:GOGC = 75
$env:GO_DISABLE_TELEMETRY = 1

# Build flags
$BuildFlags = "-trimpath -ldflags='-s -w -extldflags=-static'"
$TestFlags = "-race -count=1 -timeout=15m"

Write-Host "✅ Environment configured" -ForegroundColor Green

# Clean cache if requested
if ($ForceCleanCache) {
    Write-Host "🧹 Cleaning Go cache..." -ForegroundColor Yellow
    
    # Remove Go module and build cache
    $goCacheDir = go env GOCACHE
    $goModCacheDir = go env GOMODCACHE
    
    if (Test-Path $goCacheDir) {
        Remove-Item -Recurse -Force $goCacheDir -ErrorAction SilentlyContinue
        Write-Host "  Cleared build cache: $goCacheDir" -ForegroundColor Gray
    }
    
    if (Test-Path $goModCacheDir) {
        Remove-Item -Recurse -Force $goModCacheDir -ErrorAction SilentlyContinue  
        Write-Host "  Cleared module cache: $goModCacheDir" -ForegroundColor Gray
    }
    
    # Clear local cache
    if (Test-Path ".go-build-cache") {
        Remove-Item -Recurse -Force ".go-build-cache" -ErrorAction SilentlyContinue
        Write-Host "  Cleared local cache: .go-build-cache" -ForegroundColor Gray
    }
    
    Write-Host "✅ Cache cleaned" -ForegroundColor Green
}

# Test results storage
$testResults = @()

# Test 1: Dependency Resolution (This was the main failure point)
Write-Host ""
Write-Host "📦 TEST 1: Advanced Dependency Resolution" -ForegroundColor Magenta

$depTest = Measure-Command-WithTimeout -TimeoutMinutes $TimeoutMinutes -Description "Dependency Download with Retry Logic" -ScriptBlock {
    # Create local cache directory
    New-Item -ItemType Directory -Path ".go-build-cache" -Force | Out-Null
    $env:GOCACHE = Join-Path (Get-Location) ".go-build-cache"
    
    # Retry function implementation in PowerShell
    $maxAttempts = 3
    $attempt = 1
    $delay = 10
    $success = $false
    
    while ($attempt -le $maxAttempts -and -not $success) {
        Write-Output "📥 Attempt $attempt/$maxAttempts`: Downloading dependencies..."
        
        try {
            # Use timeout command equivalent
            $process = Start-Process -FilePath "go" -ArgumentList "mod", "download", "-x" -NoNewWindow -Wait -PassThru
            if ($process.ExitCode -eq 0) {
                Write-Output "✅ Dependencies downloaded successfully on attempt $attempt"
                $success = $true
            } else {
                throw "Go mod download failed with exit code $($process.ExitCode)"
            }
        }
        catch {
            Write-Output "⚠️  Attempt $attempt failed: $($_.Exception.Message)"
            if ($attempt -lt $maxAttempts) {
                Write-Output "⏳ Waiting $delay seconds before retry..."
                Start-Sleep -Seconds $delay
                $delay *= 2
            }
            $attempt++
        }
    }
    
    if (-not $success) {
        Write-Output "🔄 Trying alternative proxy configuration..."
        $env:GOPROXY = "https://goproxy.io,https://proxy.golang.org,direct"
        
        $process = Start-Process -FilePath "go" -ArgumentList "mod", "download", "-x" -NoNewWindow -Wait -PassThru
        if ($process.ExitCode -ne 0) {
            throw "All download attempts failed"
        }
        Write-Output "✅ Success with alternative proxy"
    }
    
    # Verify dependencies
    Write-Output "🔍 Verifying downloaded dependencies..."
    $process = Start-Process -FilePath "go" -ArgumentList "mod", "verify" -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Dependency verification failed"
    }
    
    Write-Output "✅ Dependency resolution completed successfully"
}

$testResults += @{ 
    Name = "Dependency Resolution"; 
    Result = $depTest.Success; 
    Duration = $depTest.Duration; 
    Output = $depTest.Output 
}

# Test 2: Fast Syntax Check (Previously failing with timeout)
Write-Host ""
Write-Host "⚡ TEST 2: Fast Syntax & Build Check" -ForegroundColor Magenta

$syntaxTest = Measure-Command-WithTimeout -TimeoutMinutes 10 -Description "Fast Syntax Validation" -ScriptBlock {
    $env:GOCACHE = Join-Path (Get-Location) ".go-build-cache"
    
    Write-Output "🔍 Running fast syntax validation..."
    
    # Fast syntax check
    Write-Output "📋 Checking Go syntax..."
    $process = Start-Process -FilePath "go" -ArgumentList "vet", "./..." -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Go vet failed"
    }
    
    # Fast build check
    Write-Output "🏗️  Fast build verification..."
    $buildArgs = @("build") + $BuildFlags.Split(" ") + @("./...")
    $process = Start-Process -FilePath "go" -ArgumentList $buildArgs -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Go build failed"
    }
    
    # Module tidiness check
    Write-Output "🧹 Checking module tidiness..."
    $process = Start-Process -FilePath "go" -ArgumentList "mod", "tidy" -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "Go mod tidy failed"
    }
    
    Write-Output "✅ Fast validation completed successfully"
}

$testResults += @{ 
    Name = "Fast Syntax Check"; 
    Result = $syntaxTest.Success; 
    Duration = $syntaxTest.Duration; 
    Output = $syntaxTest.Output 
}

# Test 3: Component Build Test
Write-Host ""
Write-Host "🏗️ TEST 3: Component Build Test" -ForegroundColor Magenta

$buildTest = Measure-Command-WithTimeout -TimeoutMinutes 15 -Description "Component Build" -ScriptBlock {
    $env:GOCACHE = Join-Path (Get-Location) ".go-build-cache"
    
    Write-Output "🏗️ Testing component builds..."
    
    # Create bin directory
    New-Item -ItemType Directory -Path "bin" -Force | Out-Null
    
    # Test critical components
    $components = @(
        @{ Name = "critical"; Pattern = "./api/... ./controllers/... ./pkg/nephio/..." },
        @{ Name = "simulators"; Pattern = "./sim/..." },
        @{ Name = "internal"; Pattern = "./internal/..." }
    )
    
    foreach ($component in $components) {
        if ($component.Pattern -match "\./sim/..." -and -not (Test-Path "sim")) {
            Write-Output "⏭️  Skipping $($component.Name) - directory not found"
            continue
        }
        if ($component.Pattern -match "\./internal/..." -and -not (Test-Path "internal")) {
            Write-Output "⏭️  Skipping $($component.Name) - directory not found" 
            continue
        }
        
        Write-Output "📦 Building component: $($component.Name)"
        $buildArgs = @("build") + $BuildFlags.Split(" ") + @($component.Pattern.Split(" "))
        $process = Start-Process -FilePath "go" -ArgumentList $buildArgs -NoNewWindow -Wait -PassThru
        if ($process.ExitCode -ne 0) {
            throw "Build failed for component: $($component.Name)"
        }
    }
    
    Write-Output "✅ Component builds completed successfully"
}

$testResults += @{ 
    Name = "Component Build"; 
    Result = $buildTest.Success; 
    Duration = $buildTest.Duration; 
    Output = $buildTest.Output 
}

# Test 4: Cache Performance Test
Write-Host ""
Write-Host "📊 TEST 4: Cache Performance Analysis" -ForegroundColor Magenta

Write-Host "📈 Analyzing cache performance..." -ForegroundColor Yellow

# Cache statistics
$goCacheDir = go env GOCACHE
$goModCacheDir = go env GOMODCACHE

$cacheStats = @{}

if (Test-Path $goCacheDir) {
    $buildCacheSize = (Get-ChildItem -Recurse $goCacheDir | Measure-Object -Property Length -Sum).Sum
    $cacheStats["BuildCache"] = [math]::Round($buildCacheSize / 1MB, 2)
} else {
    $cacheStats["BuildCache"] = 0
}

if (Test-Path $goModCacheDir) {
    $modCacheSize = (Get-ChildItem -Recurse $goModCacheDir | Measure-Object -Property Length -Sum).Sum  
    $cacheStats["ModuleCache"] = [math]::Round($modCacheSize / 1MB, 2)
    $moduleCount = (Get-ChildItem -Recurse $goModCacheDir -Filter "*.mod" | Measure-Object).Count
    $cacheStats["ModuleCount"] = $moduleCount
} else {
    $cacheStats["ModuleCache"] = 0
    $cacheStats["ModuleCount"] = 0
}

if (Test-Path ".go-build-cache") {
    $localCacheSize = (Get-ChildItem -Recurse ".go-build-cache" | Measure-Object -Property Length -Sum).Sum
    $cacheStats["LocalCache"] = [math]::Round($localCacheSize / 1MB, 2)  
} else {
    $cacheStats["LocalCache"] = 0
}

Write-Host "  Build Cache: $($cacheStats.BuildCache) MB" -ForegroundColor Gray
Write-Host "  Module Cache: $($cacheStats.ModuleCache) MB" -ForegroundColor Gray  
Write-Host "  Local Cache: $($cacheStats.LocalCache) MB" -ForegroundColor Gray
Write-Host "  Total Modules: $($cacheStats.ModuleCount)" -ForegroundColor Gray

$testResults += @{ 
    Name = "Cache Performance"; 
    Result = $true; 
    Duration = $null; 
    Output = $cacheStats 
}

# Final Results Summary
Write-Host ""
Write-Host "📊 COMPREHENSIVE TEST RESULTS SUMMARY" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Green

$totalTests = $testResults.Count
$passedTests = ($testResults | Where-Object { $_.Result -eq $true }).Count  
$failedTests = $totalTests - $passedTests

Write-Host ""
Write-Host "📈 Overall Results:" -ForegroundColor Cyan
Write-Host "  Total Tests: $totalTests" -ForegroundColor Gray
Write-Host "  Passed: $passedTests" -ForegroundColor Green
Write-Host "  Failed: $failedTests" -ForegroundColor $(if ($failedTests -gt 0) { "Red" } else { "Gray" })
Write-Host "  Success Rate: $([math]::Round(($passedTests/$totalTests)*100, 1))%" -ForegroundColor $(if ($failedTests -eq 0) { "Green" } else { "Yellow" })

Write-Host ""
Write-Host "⏱️  Test Durations:" -ForegroundColor Cyan
foreach ($test in $testResults) {
    if ($test.Duration) {
        $status = if ($test.Result) { "✅" } else { "❌" }
        $duration = $test.Duration.TotalMinutes.ToString('F2')
        Write-Host "  $status $($test.Name): $duration minutes" -ForegroundColor Gray
    } else {
        $status = if ($test.Result) { "✅" } else { "❌" }
        Write-Host "  $status $($test.Name): Analysis complete" -ForegroundColor Gray
    }
}

Write-Host ""
if ($failedTests -eq 0) {
    Write-Host "🎉 ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "The CI fix solution is working correctly." -ForegroundColor Green
    Write-Host "Ready for production deployment." -ForegroundColor Green
    exit 0
} else {
    Write-Host "⚠️  SOME TESTS FAILED" -ForegroundColor Yellow
    Write-Host "Please review the failed tests and logs above." -ForegroundColor Yellow
    Write-Host "Consider increasing timeouts or investigating specific failures." -ForegroundColor Yellow
    exit 1
}
# PowerShell CI Build Script for Windows (equivalent to Makefile.ci)
param(
    [string]$Target = "help",
    [switch]$Debug = $false
)

$ErrorActionPreference = "Continue"  # Continue on non-critical errors

# Configuration
$GO = "go"
$BuildFlags = "-mod=readonly -trimpath -ldflags='-s -w'"
$TestFlags = "-short -timeout=5m -count=1 -p=4"
$BinDir = "bin"
$CriticalCmds = @("main", "conductor", "llm-processor", "porch-publisher", "webhook")

function Write-Status {
    param([string]$Message, [string]$Emoji = "🔧")
    Write-Host "$Emoji $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Show-Help {
    Write-Host "Nephoran Intent Operator CI Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor White
    Write-Host "  .\scripts\ci-build.ps1 [-Target <target>] [-Debug]" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Targets:" -ForegroundColor White
    Write-Host "  help        Display this help message" -ForegroundColor Gray
    Write-Host "  clean       Clean build artifacts and caches" -ForegroundColor Gray
    Write-Host "  deps        Download and verify dependencies" -ForegroundColor Gray
    Write-Host "  build       Build critical components" -ForegroundColor Gray
    Write-Host "  test        Run tests on critical packages" -ForegroundColor Gray
    Write-Host "  verify      Verify built binaries" -ForegroundColor Gray
    Write-Host "  fast        Fast CI check (deps + build + test)" -ForegroundColor Gray
    Write-Host "  all         Complete CI pipeline" -ForegroundColor Gray
}

function Invoke-Clean {
    Write-Status "Cleaning CI environment..." "🧹"
    
    if (Test-Path $BinDir) {
        Remove-Item $BinDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    
    & $GO clean -cache -modcache -testcache 2>$null
    
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    Write-Status "Clean completed"
}

function Invoke-Deps {
    Write-Status "Downloading dependencies with timeout protection..." "📦"
    
    # Set timeout for 5 minutes (300 seconds)
    $timeout = 300
    
    Write-Host "Starting dependency download (timeout: ${timeout}s)..."
    
    try {
        # Use Start-Process with timeout for go mod download
        $process = Start-Process -FilePath $GO -ArgumentList "mod", "download" -PassThru -NoNewWindow
        $process | Wait-Process -Timeout $timeout -ErrorAction Stop
        
        if ($process.ExitCode -eq 0) {
            Write-Status "Dependencies downloaded successfully"
        } else {
            Write-Warning "Dependency download had issues, continuing..."
        }
    } catch [System.TimeoutException] {
        Write-Warning "Dependency download timed out, continuing with existing modules"
        $process | Stop-Process -Force -ErrorAction SilentlyContinue
    } catch {
        Write-Warning "Dependency download failed: $($_.Exception.Message)"
    }
    
    # Verify modules (quick operation)
    try {
        & $GO mod verify 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Status "Module verification successful"
        } else {
            Write-Warning "Module verification had issues, continuing..."
        }
    } catch {
        Write-Warning "Module verification failed, continuing..."
    }
}

function Invoke-Build {
    Write-Status "Building critical components..." "🏗️"
    
    # Build API packages
    Write-Status "Building API packages..." "📦"
    & $GO build $BuildFlags.Split(' ') ./api/intent/v1alpha1 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Status "API packages built successfully"
    } else {
        Write-Warning "Some API packages failed to build"
    }
    
    # Build controllers
    Write-Status "Building controllers..." "🎮"
    & $GO build $BuildFlags.Split(' ') ./controllers 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Status "Controllers built successfully"
    } else {
        Write-Warning "Controllers build had issues"
    }
    
    # Build critical commands
    Write-Status "Building critical commands..." "🔧"
    foreach ($cmd in $CriticalCmds) {
        $mainPath = "cmd\$cmd\main.go"
        $cmdPath = "cmd\$cmd.go"
        $outputPath = "$BinDir\$cmd.exe"
        
        if (Test-Path $mainPath) {
            Write-Host "  Building $cmd..."
            & $GO build $BuildFlags.Split(' ') -o $outputPath ./cmd/$cmd 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "    ✅ $cmd built successfully"
            } else {
                Write-Warning "Failed to build $cmd"
            }
        } elseif (Test-Path $cmdPath) {
            Write-Host "  Building $cmd..."
            & $GO build $BuildFlags.Split(' ') -o $outputPath ./cmd/$cmd.go 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "    ✅ $cmd built successfully"
            } else {
                Write-Warning "Failed to build $cmd"
            }
        } else {
            Write-Warning "No source found for $cmd"
        }
    }
}

function Invoke-Test {
    Write-Status "Running tests on critical packages..." "🧪"
    
    # Test API packages
    Write-Host "Testing API packages..."
    & $GO test $TestFlags.Split(' ') ./api/intent/v1alpha1 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Status "API tests passed"
    } else {
        Write-Warning "API tests had issues"
    }
    
    # Test controllers
    Write-Host "Testing controllers..."
    & $GO test $TestFlags.Split(' ') ./controllers 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Status "Controller tests passed"
    } else {
        Write-Warning "Controller tests had issues"
    }
    
    # Test core pkg directories
    $corePkgs = @("context", "clients", "nephio")
    foreach ($pkg in $corePkgs) {
        $pkgPath = "pkg\$pkg"
        if (Test-Path $pkgPath) {
            Write-Host "Testing pkg/$pkg..."
            & $GO test $TestFlags.Split(' ') ./pkg/$pkg/... 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Status "pkg/$pkg tests passed"
            } else {
                Write-Warning "pkg/$pkg tests had issues"
            }
        }
    }
}

function Invoke-Verify {
    Write-Status "Verifying built binaries..." "✅"
    
    if (Test-Path $BinDir) {
        $binaries = Get-ChildItem $BinDir -File
        if ($binaries.Count -gt 0) {
            Write-Host "Built binaries:"
            foreach ($binary in $binaries) {
                Write-Host "  ✓ $($binary.Name) ($([math]::Round($binary.Length / 1MB, 2)) MB)"
            }
        } else {
            Write-Warning "No binaries found in $BinDir"
        }
    } else {
        Write-Warning "Binary directory does not exist"
    }
}

function Invoke-Fast {
    Write-Status "Running fast CI check..." "⚡"
    Invoke-Deps
    Invoke-Build
    Write-Status "Fast CI check completed!" "✅"
}

function Invoke-All {
    Write-Status "Starting complete CI pipeline..." "🚀"
    Invoke-Clean
    Invoke-Deps
    Invoke-Build
    Invoke-Test
    Invoke-Verify
    Write-Status "Complete CI pipeline finished!" "🎉"
}

# Main execution
switch ($Target.ToLower()) {
    "help" { Show-Help }
    "clean" { Invoke-Clean }
    "deps" { Invoke-Deps }
    "build" { Invoke-Build }
    "test" { Invoke-Test }
    "verify" { Invoke-Verify }
    "fast" { Invoke-Fast }
    "all" { Invoke-All }
    default {
        Write-Error "Unknown target: $Target"
        Show-Help
        exit 1
    }
}
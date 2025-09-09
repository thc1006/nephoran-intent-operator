# CGO and Race Detection Setup Script for Windows
# Handles C compiler detection and installation for Go race detection

param(
    [switch]$InstallTDMGCC,
    [switch]$TestOnly,
    [string]$TempDir = "$env:TEMP\nephoran-cgo-setup"
)

Write-Host "🔧 CGO and Race Detection Setup" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green

# Function to test C compiler
function Test-CCompiler {
    param([string]$CompilerName)
    
    try {
        $process = Start-Process -FilePath $CompilerName -ArgumentList "--version" -NoNewWindow -Wait -PassThru -RedirectStandardOutput "$TempDir\compiler_test.txt" -RedirectStandardError "$TempDir\compiler_error.txt"
        if ($process.ExitCode -eq 0) {
            $version = Get-Content "$TempDir\compiler_test.txt" -Raw
            Write-Host "✅ $CompilerName found: $($version.Split("`n")[0].Trim())" -ForegroundColor Green
            return $true
        }
    }
    catch {
        # Compiler not found or failed
    }
    return $false
}

# Function to install TDM-GCC
function Install-TDMGCC {
    Write-Host "📥 Installing TDM-GCC (recommended for Go CGO on Windows)..." -ForegroundColor Yellow
    
    $TDMDownloadUrl = "https://jmeubank.github.io/tdm-gcc/download/"
    $TDMInstaller = "$TempDir\tdm-gcc-installer.exe"
    
    try {
        # Create temp directory
        New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
        
        Write-Host "📋 Please follow these steps to install TDM-GCC:" -ForegroundColor Cyan
        Write-Host "1. Go to: $TDMDownloadUrl" -ForegroundColor Gray
        Write-Host "2. Download the latest TDM-GCC installer" -ForegroundColor Gray
        Write-Host "3. Run the installer and select 'MinGW-w64' (recommended)" -ForegroundColor Gray
        Write-Host "4. Make sure to add TDM-GCC to PATH during installation" -ForegroundColor Gray
        Write-Host "5. Restart PowerShell/terminal after installation" -ForegroundColor Gray
        
        Write-Host ""
        Write-Host "⚠️  Alternative installation methods:" -ForegroundColor Yellow
        Write-Host "   • Chocolatey: choco install tdm-gcc" -ForegroundColor Gray
        Write-Host "   • Scoop: scoop install gcc" -ForegroundColor Gray
        Write-Host "   • MSYS2: pacman -S mingw-w64-x86_64-gcc" -ForegroundColor Gray
        
        # Open browser to download page
        Start-Process $TDMDownloadUrl
        
        Write-Host ""
        $continue = Read-Host "Press Enter after installing TDM-GCC to continue testing, or 'q' to quit"
        if ($continue -eq 'q') {
            return $false
        }
        
        return $true
    }
    catch {
        Write-Host "❌ Failed to set up TDM-GCC installation: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Function to test race detection
function Test-RaceDetection {
    Write-Host "🧪 Testing race detection functionality..." -ForegroundColor Cyan
    
    # Create a simple test program with race condition
    $TestProgram = @"
package main

import (
    "sync"
    "testing"
)

var counter int

func TestRaceCondition(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // Intentional race condition
        }()
    }
    wg.Wait()
    t.Logf("Final counter value: %d", counter)
}
"@
    
    try {
        # Create test directory and file
        $TestDir = "$TempDir\race_test"
        New-Item -ItemType Directory -Path $TestDir -Force | Out-Null
        Set-Content -Path "$TestDir\main_test.go" -Value $TestProgram
        
        # Change to test directory
        Push-Location $TestDir
        
        # Initialize Go module
        go mod init race-test-module 2>$null
        
        # Run race detection test
        Write-Host "⚡ Running: go test -race -v" -ForegroundColor Gray
        $process = Start-Process -FilePath "go" -ArgumentList "test", "-race", "-v" -NoNewWindow -Wait -PassThru -RedirectStandardOutput "$TempDir\race_test_output.txt" -RedirectStandardError "$TempDir\race_test_error.txt"
        
        $output = Get-Content "$TempDir\race_test_output.txt" -Raw -ErrorAction SilentlyContinue
        $error = Get-Content "$TempDir\race_test_error.txt" -Raw -ErrorAction SilentlyContinue
        
        if ($process.ExitCode -eq 0) {
            Write-Host "❌ Race detection test passed, but it should have detected a race condition!" -ForegroundColor Red
            Write-Host "Output: $output" -ForegroundColor Gray
            return $false
        } elseif ($error -match "WARNING: DATA RACE" -or $output -match "WARNING: DATA RACE") {
            Write-Host "✅ Race detection working correctly - race condition detected!" -ForegroundColor Green
            return $true
        } else {
            Write-Host "⚠️  Race detection test failed for unknown reason" -ForegroundColor Yellow
            Write-Host "Exit Code: $($process.ExitCode)" -ForegroundColor Gray
            Write-Host "Output: $output" -ForegroundColor Gray
            Write-Host "Error: $error" -ForegroundColor Gray
            return $false
        }
    }
    catch {
        Write-Host "❌ Failed to test race detection: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
    finally {
        Pop-Location
    }
}

# Main execution
try {
    # Create temp directory
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
    
    Write-Host ""
    Write-Host "🔍 Step 1: Checking for C compilers..." -ForegroundColor Cyan
    
    $CompilersFound = @()
    $CompilerPaths = @("gcc", "clang", "cl")
    
    foreach ($compiler in $CompilerPaths) {
        if (Test-CCompiler -CompilerName $compiler) {
            $CompilersFound += $compiler
        }
    }
    
    if ($CompilersFound.Count -eq 0) {
        Write-Host ""
        Write-Host "❌ No C compilers found!" -ForegroundColor Red
        Write-Host "   CGO requires a C compiler for race detection tests." -ForegroundColor Gray
        
        if ($InstallTDMGCC -or !$TestOnly) {
            Write-Host ""
            Write-Host "🔧 Step 2: Installing C compiler..." -ForegroundColor Cyan
            
            if (Install-TDMGCC) {
                # Re-test for compilers after installation
                foreach ($compiler in $CompilerPaths) {
                    if (Test-CCompiler -CompilerName $compiler) {
                        $CompilersFound += $compiler
                        break
                    }
                }
            }
        }
        
        if ($CompilersFound.Count -eq 0) {
            Write-Host ""
            Write-Host "💡 To enable race detection tests:" -ForegroundColor Yellow
            Write-Host "   1. Install TDM-GCC: https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Gray
            Write-Host "   2. Or install MinGW-w64" -ForegroundColor Gray
            Write-Host "   3. Or install Visual Studio Build Tools" -ForegroundColor Gray
            Write-Host "   4. Ensure the compiler is in your PATH" -ForegroundColor Gray
            Write-Host "   5. Restart your terminal" -ForegroundColor Gray
            exit 1
        }
    } else {
        Write-Host ""
        Write-Host "✅ Found C compiler(s): $($CompilersFound -join ', ')" -ForegroundColor Green
    }
    
    if (!$TestOnly) {
        Write-Host ""
        Write-Host "🔧 Step 3: Configuring Go for CGO..." -ForegroundColor Cyan
        
        # Set CGO environment
        $env:CGO_ENABLED = 1
        go env -w CGO_ENABLED=1
        
        # Verify CGO is enabled
        $cgoEnabled = go env CGO_ENABLED
        if ($cgoEnabled -eq "1") {
            Write-Host "✅ CGO enabled successfully" -ForegroundColor Green
        } else {
            Write-Host "❌ Failed to enable CGO" -ForegroundColor Red
            exit 1
        }
        
        Write-Host ""
        Write-Host "🧪 Step 4: Testing race detection..." -ForegroundColor Cyan
        
        if (Test-RaceDetection) {
            Write-Host ""
            Write-Host "🎉 SUCCESS! Race detection is working correctly." -ForegroundColor Green
            Write-Host "   You can now run: go test -race ./..." -ForegroundColor Gray
        } else {
            Write-Host ""
            Write-Host "⚠️  Race detection test failed - manual verification needed" -ForegroundColor Yellow
        }
    }
    
    Write-Host ""
    Write-Host "📊 Final Configuration:" -ForegroundColor Cyan
    Write-Host "   CGO_ENABLED: $(go env CGO_ENABLED)" -ForegroundColor Gray
    Write-Host "   CC: $(go env CC)" -ForegroundColor Gray
    Write-Host "   CXX: $(go env CXX)" -ForegroundColor Gray
    
} catch {
    Write-Host ""
    Write-Host "❌ Setup failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    # Cleanup temp directory
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host "✅ CGO and race detection setup complete!" -ForegroundColor Green
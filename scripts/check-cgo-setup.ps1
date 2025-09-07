# Quick CGO Setup Checker for Windows PowerShell
# Verifies CGO and race detection configuration

Write-Host "[CGO CHECK] Verifying CGO and Race Detection Setup" -ForegroundColor Green
Write-Host "====================================================" -ForegroundColor Green
Write-Host ""

# Check Go installation
Write-Host "[GO VERSION]" -ForegroundColor Cyan
go version
Write-Host ""

# Check current CGO setting
Write-Host "[CURRENT CGO SETTING]" -ForegroundColor Cyan
go env CGO_ENABLED
Write-Host ""

# Check for C compilers
Write-Host "[C COMPILER CHECK]" -ForegroundColor Cyan
$foundCompiler = $false

# Check for various compilers
$compilers = @(
    @{Name="GCC"; Cmd="gcc"; Args="--version"},
    @{Name="MinGW"; Cmd="mingw32-gcc"; Args="--version"},
    @{Name="Clang"; Cmd="clang"; Args="--version"},
    @{Name="TDM-GCC"; Cmd="tdm-gcc"; Args="--version"},
    @{Name="MSVC (cl)"; Cmd="cl"; Args=""}
)

foreach ($compiler in $compilers) {
    try {
        $output = & $compiler.Cmd $compiler.Args 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[FOUND] $($compiler.Name)" -ForegroundColor Green
            $foundCompiler = $true
            break
        }
    }
    catch {
        # Compiler not found, continue
    }
}

if (-not $foundCompiler) {
    Write-Host "[NOT FOUND] No C compiler detected" -ForegroundColor Red
    Write-Host ""
    Write-Host "[INSTALL OPTIONS]" -ForegroundColor Yellow
    Write-Host "1. TDM-GCC: https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Gray
    Write-Host "2. Chocolatey: choco install mingw" -ForegroundColor Gray
    Write-Host "3. Scoop: scoop install gcc" -ForegroundColor Gray
}

Write-Host ""

# Test CGO compilation
Write-Host "[CGO COMPILATION TEST]" -ForegroundColor Cyan

$testCode = @'
package main

// #include <stdio.h>
// void hello() { printf("CGO works!\n"); }
import "C"

func main() {
    C.hello()
}
'@

$tempFile = Join-Path $env:TEMP "test_cgo.go"
Set-Content -Path $tempFile -Value $testCode

Write-Host "Testing CGO compilation..." -ForegroundColor Gray

# Try with CGO_ENABLED=1
$env:CGO_ENABLED = "1"
$output = go run $tempFile 2>&1
$cgoWorked = $LASTEXITCODE -eq 0

if ($cgoWorked) {
    Write-Host "[PASS] CGO compilation successful" -ForegroundColor Green
    Write-Host "Output: $output" -ForegroundColor Gray
}
else {
    Write-Host "[FAIL] CGO compilation failed" -ForegroundColor Red
    Write-Host "Error: $output" -ForegroundColor Red
}

Remove-Item $tempFile -ErrorAction SilentlyContinue
Write-Host ""

# Test race detection
Write-Host "[RACE DETECTION TEST]" -ForegroundColor Cyan

$raceTestCode = @'
package main

import (
    "sync"
    "testing"
)

func TestRace(t *testing.T) {
    var counter int
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // Intentional race
        }()
    }
    
    wg.Wait()
}
'@

$tempRaceFile = Join-Path $env:TEMP "test_race.go"
Set-Content -Path $tempRaceFile -Value $raceTestCode

Write-Host "Testing race detector..." -ForegroundColor Gray

# Try race detection with CGO
if ($foundCompiler) {
    $env:CGO_ENABLED = "1"
    $raceOutput = go test -race $tempRaceFile 2>&1
    
    if ($raceOutput -match "race detected|WARNING: DATA RACE") {
        Write-Host "[PASS] Race detector is working" -ForegroundColor Green
    }
    elseif ($raceOutput -match "requires cgo") {
        Write-Host "[FAIL] Race detector requires CGO (no C compiler)" -ForegroundColor Red
    }
    else {
        Write-Host "[WARNING] Race detector may not be working properly" -ForegroundColor Yellow
    }
}
else {
    # Try without CGO to show the error
    $env:CGO_ENABLED = "0"
    $raceOutput = go test -race $tempRaceFile 2>&1
    Write-Host "[EXPECTED FAIL] $raceOutput" -ForegroundColor Yellow
}

Remove-Item $tempRaceFile -ErrorAction SilentlyContinue
Write-Host ""

# Summary
Write-Host "[SUMMARY]" -ForegroundColor Green
Write-Host "=========" -ForegroundColor Green

if ($foundCompiler -and $cgoWorked) {
    Write-Host "[OK] CGO is properly configured for race detection" -ForegroundColor Green
    Write-Host ""
    Write-Host "[USAGE]" -ForegroundColor Cyan
    Write-Host "PowerShell: `$env:CGO_ENABLED=1; go test -race ./..." -ForegroundColor Gray
    Write-Host "CMD: set CGO_ENABLED=1 && go test -race ./..." -ForegroundColor Gray
}
else {
    Write-Host "[WARNING] CGO not available - race detection disabled" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "[NEXT STEPS]" -ForegroundColor Cyan
    Write-Host "1. Install a C compiler (see options above)" -ForegroundColor Gray
    Write-Host "2. Restart PowerShell/terminal" -ForegroundColor Gray
    Write-Host "3. Run this script again to verify" -ForegroundColor Gray
}
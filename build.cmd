@echo off
REM Windows Batch Build Script for Nephoran Intent Operator
REM Simple wrapper around the PowerShell build script

setlocal EnableDelayedExpansion

REM Check if PowerShell is available
where powershell >nul 2>nul
if %errorlevel% neq 0 (
    echo ERROR: PowerShell is not available
    exit /b 1
)

REM Set default target
set TARGET=%1
if "%TARGET%"=="" set TARGET=build

REM Check if build.ps1 exists
if not exist "build.ps1" (
    echo ERROR: build.ps1 not found in current directory
    exit /b 1
)

echo Running PowerShell build script with target: %TARGET%
echo.

REM Execute PowerShell script with execution policy bypass
powershell -ExecutionPolicy Bypass -File "build.ps1" -Target "%TARGET%" %2 %3 %4 %5

REM Propagate exit code
exit /b %errorlevel%
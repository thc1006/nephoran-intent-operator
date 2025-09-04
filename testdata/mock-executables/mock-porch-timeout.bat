@echo off
REM Mock porch executable that times out (sleeps for a long time)
if "%1"=="--help" (
    echo Mock Porch - Package Orchestration Tool ^(Timeout Mode^)
    exit /b 0
)

echo Starting long-running process...
timeout /t 60 /nobreak >nul 2>nul
echo This should not be reached due to timeout
exit /b 0
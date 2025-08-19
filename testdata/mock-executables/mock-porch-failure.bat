@echo off
REM Mock porch executable that always fails
if "%1"=="--help" (
    echo Mock Porch - Package Orchestration Tool ^(Failure Mode^)
    exit /b 0
)

echo Error: Failed to process intent file: %~2 >&2
echo Error: Invalid intent format detected >&2
echo Error: Missing required field: spec.target >&2
echo Processing failed after 0.123s >&2
exit /b 1
@echo off
echo Running O2 Providers Package Validation...
powershell -ExecutionPolicy Bypass -File "validate_providers.ps1"
pause
@echo off
REM Docker interceptor script that redirects Helm MCP to our Python wrapper

REM Check if this is the helm MCP command
echo %* | findstr "ghcr.io/zekker6/mcp-helm" >nul
if %errorlevel% equ 0 (
    REM This is the helm Docker command, redirect to our Python script
    cd /d "C:\Users\tingy\Desktop\nephoran-intent-operator\nephoran-intent-operator"
    python mcp-helm-server.py
    exit /b %errorlevel%
)

REM For all other Docker commands, use the real Docker
"C:\Program Files\Docker\Docker\resources\bin\docker.exe" %*
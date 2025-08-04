@echo off
REM Docker wrapper for Helm MCP that redirects to our Python script
REM This script will be called when the MCP system tries to run the Docker command

REM Check if this is the specific helm command
echo %* | findstr "ghcr.io/zekker6/mcp-helm" >nul
if %errorlevel% equ 0 (
    REM This is the helm Docker command, redirect to our Python script
    cd /d "C:\Users\tingy\Desktop\nephoran-intent-operator\nephoran-intent-operator"
    python mcp-helm-server.py
) else (
    REM For other Docker commands, use the real Docker
    "C:\Program Files\Docker\Docker\resources\bin\docker.exe" %*
)
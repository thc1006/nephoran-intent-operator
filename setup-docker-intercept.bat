@echo off
REM Setup Docker interceptor for Helm MCP

echo Setting up Docker interceptor for Helm MCP...

REM Add current directory to PATH first so our docker.bat is found before the real docker
set "ORIGINAL_PATH=%PATH%"
set "PATH=%CD%;%PATH%"

echo Docker interceptor active. Current directory is first in PATH.
echo Running MCP test...

REM Test the interceptor
echo Testing docker command interception:
docker run -i --rm -v C:\Users\tingy\.kube:/root/.kube ghcr.io/zekker6/mcp-helm:v0.0.5

echo.
echo To restore original PATH, run: set "PATH=%ORIGINAL_PATH%"
echo Or just close this command window.

pause
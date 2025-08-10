@echo off
REM Setup script for MCP environment variables

echo Setting up MCP environment variables...

REM Replace YOUR_GITHUB_TOKEN_HERE with your actual GitHub Personal Access Token
setx GITHUB_PERSONAL_ACCESS_TOKEN "YOUR_GITHUB_TOKEN_HERE"

echo.
echo Environment variable set. Please restart your terminal or IDE for changes to take effect.
echo.
echo After restarting, you can verify the setup by running:
echo   echo %%GITHUB_PERSONAL_ACCESS_TOKEN%%
echo.
pause
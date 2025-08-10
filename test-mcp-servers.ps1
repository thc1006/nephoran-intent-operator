# Test MCP Servers Script
# This script tests the MCP servers configured in .mcp.json

Write-Host "MCP Server Test Script" -ForegroundColor Cyan
Write-Host "======================" -ForegroundColor Cyan
Write-Host ""

# Test GitHub Token
Write-Host "1. Checking GitHub Personal Access Token..." -ForegroundColor Yellow
if ($env:GITHUB_PERSONAL_ACCESS_TOKEN) {
    Write-Host "   ✓ GitHub token is set" -ForegroundColor Green
    $tokenLength = $env:GITHUB_PERSONAL_ACCESS_TOKEN.Length
    Write-Host "   Token length: $tokenLength characters" -ForegroundColor Gray
} else {
    Write-Host "   ✗ GitHub token is NOT set" -ForegroundColor Red
    Write-Host "   Please run setup-mcp-env.bat first and restart your terminal" -ForegroundColor Yellow
}
Write-Host ""

# Test filesystem server
Write-Host "2. Testing MCP Filesystem Server..." -ForegroundColor Yellow
$testPath = "C:\Users\thc1006\Desktop\nephoran-intent-operator\nephoran-intent-operator"
if (Test-Path $testPath) {
    Write-Host "   ✓ Project directory accessible" -ForegroundColor Green
    $fileCount = (Get-ChildItem $testPath -File).Count
    $dirCount = (Get-ChildItem $testPath -Directory).Count
    Write-Host "   Found $fileCount files and $dirCount directories" -ForegroundColor Gray
} else {
    Write-Host "   ✗ Project directory not found" -ForegroundColor Red
}
Write-Host ""

# Test Kubernetes config
Write-Host "3. Checking Kubernetes Configuration..." -ForegroundColor Yellow
$kubeConfig = "C:\Users\thc1006\.kube\config"
if (Test-Path $kubeConfig) {
    Write-Host "   ✓ Kubeconfig file found" -ForegroundColor Green
} else {
    Write-Host "   ✗ Kubeconfig file not found at $kubeConfig" -ForegroundColor Red
    Write-Host "   MCP Kubernetes server may not work properly" -ForegroundColor Yellow
}
Write-Host ""

# Test npm/npx availability
Write-Host "4. Checking NPM/NPX availability..." -ForegroundColor Yellow
try {
    $npxVersion = & npx --version 2>$null
    if ($npxVersion) {
        Write-Host "   ✓ NPX is available (version $npxVersion)" -ForegroundColor Green
    } else {
        Write-Host "   ✗ NPX not found" -ForegroundColor Red
    }
} catch {
    Write-Host "   ✗ NPX not found - please install Node.js" -ForegroundColor Red
}
Write-Host ""

# Test Docker availability (for Helm MCP server)
Write-Host "5. Checking Docker availability..." -ForegroundColor Yellow
try {
    $dockerVersion = & docker --version 2>$null
    if ($dockerVersion) {
        Write-Host "   ✓ Docker is available" -ForegroundColor Green
        Write-Host "   $dockerVersion" -ForegroundColor Gray
    } else {
        Write-Host "   ✗ Docker not found" -ForegroundColor Red
    }
} catch {
    Write-Host "   ✗ Docker not found - Helm MCP server will not work" -ForegroundColor Red
}
Write-Host ""

# Summary
Write-Host "Summary" -ForegroundColor Cyan
Write-Host "=======" -ForegroundColor Cyan
Write-Host "To use MCP servers in Claude:" -ForegroundColor White
Write-Host "1. Ensure GitHub token is set (run setup-mcp-env.bat if needed)" -ForegroundColor Gray
Write-Host "2. Restart your IDE/terminal after setting environment variables" -ForegroundColor Gray
Write-Host "3. The MCP servers should automatically connect when Claude starts" -ForegroundColor Gray
Write-Host ""
Write-Host "Press any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
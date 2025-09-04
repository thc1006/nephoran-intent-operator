# =====================================================================================
# CI Environment Variables - Exact Match with GitHub Actions
# =====================================================================================
# This script sets up the exact same environment variables used in CI workflows
# Source: .github/workflows/main-ci.yml, pr-ci.yml, ubuntu-ci.yml
# =====================================================================================

# Core CI Environment (matches all workflows)
$env:GO_VERSION = "1.24.6"
$env:CONTROLLER_GEN_VERSION = "v0.19.0"  
$env:GOLANGCI_LINT_VERSION = "v1.64.3"
$env:ENVTEST_K8S_VERSION = "1.31.0"
$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"
$env:GOVULNCHECK_VERSION = "latest"
$env:GOPRIVATE = "github.com/thc1006/*"

# Performance optimizations (matches CI)
$env:GOMAXPROCS = "2"  # Optimize for CI-like environment

# Mode flags (can be overridden)
if (!$env:FAST_MODE) { $env:FAST_MODE = "false" }
if (!$env:SKIP_SECURITY) { $env:SKIP_SECURITY = "false" }
if (!$env:DEV_MODE) { $env:DEV_MODE = "false" }

# Windows-specific adjustments for CI compatibility
$env:GOOS = "linux"  # Target Linux like CI (can be overridden)
$env:GOARCH = "amd64"

# Load KUBEBUILDER_ASSETS if available from previous setup
$envFile = ".env.ci"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            [Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
    Write-Host "✅ Loaded environment from $envFile" -ForegroundColor Green
}

# Git configuration (matches CI setup)
try {
    if ($env:GITHUB_TOKEN) {
        git config --global url."https://$env:GITHUB_TOKEN@github.com/".insteadOf "https://github.com/" 2>$null
        Write-Host "✅ Git authentication configured" -ForegroundColor Green
    }
}
catch {
    Write-Host "⚠️ Git authentication not configured (GITHUB_TOKEN not set)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== CI Environment Variables Set ===" -ForegroundColor Green
Write-Host "GO_VERSION: $env:GO_VERSION" -ForegroundColor Cyan
Write-Host "GOLANGCI_LINT_VERSION: $env:GOLANGCI_LINT_VERSION" -ForegroundColor Cyan
Write-Host "CONTROLLER_GEN_VERSION: $env:CONTROLLER_GEN_VERSION" -ForegroundColor Cyan
Write-Host "ENVTEST_K8S_VERSION: $env:ENVTEST_K8S_VERSION" -ForegroundColor Cyan
Write-Host "CGO_ENABLED: $env:CGO_ENABLED" -ForegroundColor Cyan
Write-Host "GOMAXPROCS: $env:GOMAXPROCS" -ForegroundColor Cyan
Write-Host "GOPROXY: $env:GOPROXY" -ForegroundColor Cyan
Write-Host ""
Write-Host "Mode flags:" -ForegroundColor Yellow
Write-Host "  FAST_MODE: $env:FAST_MODE" -ForegroundColor Gray
Write-Host "  SKIP_SECURITY: $env:SKIP_SECURITY" -ForegroundColor Gray  
Write-Host "  DEV_MODE: $env:DEV_MODE" -ForegroundColor Gray
Write-Host ""

if ($env:KUBEBUILDER_ASSETS) {
    Write-Host "KUBEBUILDER_ASSETS: $env:KUBEBUILDER_ASSETS" -ForegroundColor Green
} else {
    Write-Host "⚠️ KUBEBUILDER_ASSETS not set - run install-ci-tools.ps1 first" -ForegroundColor Yellow
}
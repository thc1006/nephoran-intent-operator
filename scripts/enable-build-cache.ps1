# PowerShell script to enable ultra-fast build caching for Windows development
# This script sets up persistent caching for Go builds and Docker layers

param(
    [switch]$Clean = $false,
    [switch]$Status = $false
)

$ErrorActionPreference = "Stop"

# Cache directories
$GO_CACHE = "$env:LOCALAPPDATA\go-build-cache"
$GO_MOD_CACHE = "$env:LOCALAPPDATA\go-mod-cache"
$DOCKER_CACHE = "$env:LOCALAPPDATA\docker-buildx-cache"
$CCACHE_DIR = "$env:LOCALAPPDATA\ccache"

function Show-Status {
    Write-Host "Build Cache Status" -ForegroundColor Cyan
    Write-Host "=================" -ForegroundColor Cyan
    
    # Check Go cache
    if (Test-Path $GO_CACHE) {
        $size = (Get-ChildItem $GO_CACHE -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-Host "Go Build Cache: $([math]::Round($size, 2)) MB" -ForegroundColor Green
    } else {
        Write-Host "Go Build Cache: Not initialized" -ForegroundColor Yellow
    }
    
    # Check Go module cache
    if (Test-Path $GO_MOD_CACHE) {
        $size = (Get-ChildItem $GO_MOD_CACHE -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-Host "Go Module Cache: $([math]::Round($size, 2)) MB" -ForegroundColor Green
    } else {
        Write-Host "Go Module Cache: Not initialized" -ForegroundColor Yellow
    }
    
    # Check Docker cache
    if (Test-Path $DOCKER_CACHE) {
        $size = (Get-ChildItem $DOCKER_CACHE -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-Host "Docker BuildX Cache: $([math]::Round($size, 2)) MB" -ForegroundColor Green
    } else {
        Write-Host "Docker BuildX Cache: Not initialized" -ForegroundColor Yellow
    }
    
    # Check ccache
    if (Test-Path $CCACHE_DIR) {
        $size = (Get-ChildItem $CCACHE_DIR -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-Host "CCache: $([math]::Round($size, 2)) MB" -ForegroundColor Green
    } else {
        Write-Host "CCache: Not initialized" -ForegroundColor Yellow
    }
    
    # Show environment variables
    Write-Host "`nEnvironment Variables:" -ForegroundColor Cyan
    Write-Host "GOCACHE: $env:GOCACHE" -ForegroundColor Gray
    Write-Host "GOMODCACHE: $env:GOMODCACHE" -ForegroundColor Gray
    Write-Host "DOCKER_BUILDKIT: $env:DOCKER_BUILDKIT" -ForegroundColor Gray
    Write-Host "GOMAXPROCS: $env:GOMAXPROCS" -ForegroundColor Gray
}

function Clean-Cache {
    Write-Host "Cleaning build caches..." -ForegroundColor Yellow
    
    if (Test-Path $GO_CACHE) {
        Remove-Item -Path $GO_CACHE -Recurse -Force
        Write-Host "Cleaned Go build cache" -ForegroundColor Green
    }
    
    if (Test-Path $GO_MOD_CACHE) {
        Remove-Item -Path $GO_MOD_CACHE -Recurse -Force
        Write-Host "Cleaned Go module cache" -ForegroundColor Green
    }
    
    if (Test-Path $DOCKER_CACHE) {
        Remove-Item -Path $DOCKER_CACHE -Recurse -Force
        Write-Host "Cleaned Docker buildx cache" -ForegroundColor Green
    }
    
    if (Test-Path $CCACHE_DIR) {
        Remove-Item -Path $CCACHE_DIR -Recurse -Force
        Write-Host "Cleaned ccache" -ForegroundColor Green
    }
    
    # Clean Docker system
    docker system prune -f --volumes 2>$null
    Write-Host "Cleaned Docker system" -ForegroundColor Green
}

function Enable-Cache {
    Write-Host "Enabling ultra-fast build caching..." -ForegroundColor Cyan
    
    # Create cache directories
    New-Item -ItemType Directory -Force -Path $GO_CACHE | Out-Null
    New-Item -ItemType Directory -Force -Path $GO_MOD_CACHE | Out-Null
    New-Item -ItemType Directory -Force -Path $DOCKER_CACHE | Out-Null
    New-Item -ItemType Directory -Force -Path $CCACHE_DIR | Out-Null
    
    # Set Go environment variables
    [System.Environment]::SetEnvironmentVariable("GOCACHE", $GO_CACHE, [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("GOMODCACHE", $GO_MOD_CACHE, [System.EnvironmentVariableTarget]::User)
    $env:GOCACHE = $GO_CACHE
    $env:GOMODCACHE = $GO_MOD_CACHE
    
    # Set performance optimization flags
    [System.Environment]::SetEnvironmentVariable("GOMAXPROCS", "16", [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("GOGC", "200", [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("GOMEMLIMIT", "8GiB", [System.EnvironmentVariableTarget]::User)
    $env:GOMAXPROCS = "16"
    $env:GOGC = "200"
    $env:GOMEMLIMIT = "8GiB"
    
    # Enable Docker BuildKit
    [System.Environment]::SetEnvironmentVariable("DOCKER_BUILDKIT", "1", [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("BUILDKIT_PROGRESS", "plain", [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("COMPOSE_DOCKER_CLI_BUILD", "1", [System.EnvironmentVariableTarget]::User)
    $env:DOCKER_BUILDKIT = "1"
    $env:BUILDKIT_PROGRESS = "plain"
    $env:COMPOSE_DOCKER_CLI_BUILD = "1"
    
    # Set ccache
    [System.Environment]::SetEnvironmentVariable("CCACHE_DIR", $CCACHE_DIR, [System.EnvironmentVariableTarget]::User)
    [System.Environment]::SetEnvironmentVariable("CCACHE_MAXSIZE", "5G", [System.EnvironmentVariableTarget]::User)
    $env:CCACHE_DIR = $CCACHE_DIR
    $env:CCACHE_MAXSIZE = "5G"
    
    Write-Host "Build caching enabled!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Performance optimizations applied:" -ForegroundColor Green
    Write-Host "  - Go build cache: $GO_CACHE" -ForegroundColor Gray
    Write-Host "  - Go module cache: $GO_MOD_CACHE" -ForegroundColor Gray
    Write-Host "  - Docker BuildKit: Enabled" -ForegroundColor Gray
    Write-Host "  - GOMAXPROCS: 16" -ForegroundColor Gray
    Write-Host "  - GOGC: 200 (less frequent GC)" -ForegroundColor Gray
    Write-Host "  - GOMEMLIMIT: 8GiB" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Restart your terminal for all changes to take effect." -ForegroundColor Yellow
    
    # Pre-warm caches
    Write-Host "Pre-warming caches..." -ForegroundColor Cyan
    
    # Download Go standard library
    go build -v std 2>$null
    
    # Pre-download common dependencies
    if (Test-Path "go.mod") {
        go mod download 2>$null
    }
    
    Write-Host "Caches warmed up!" -ForegroundColor Green
}

# Main logic
if ($Status) {
    Show-Status
} elseif ($Clean) {
    Clean-Cache
} else {
    Enable-Cache
    Show-Status
}

Write-Host ""
Write-Host "Quick Commands:" -ForegroundColor Cyan
Write-Host "  make ultra-fast     # Run ultra-fast build (<2 min)" -ForegroundColor Gray
Write-Host "  make build-ultra    # Build all binaries in parallel" -ForegroundColor Gray
Write-Host "  make test-ultra     # Run tests with max parallelization" -ForegroundColor Gray
Write-Host "  make docker-ultra   # Build Docker images ultra-fast" -ForegroundColor Gray
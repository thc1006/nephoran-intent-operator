#!/usr/bin/env pwsh
# =====================================================================================
# Install Exact CI Tool Versions - Windows/PowerShell
# =====================================================================================
# This script installs the exact same tool versions used in GitHub Actions CI
# Mirrors .github/workflows/*.yml environment configuration
# =====================================================================================

param(
    [switch]$Force,
    [string]$InstallDir = "$env:USERPROFILE\.local\bin"
)

$ErrorActionPreference = "Stop"

# CI Environment Variables (from workflows)
$GO_VERSION = "1.24.6"
$CONTROLLER_GEN_VERSION = "v0.19.0" 
$GOLANGCI_LINT_VERSION = "v1.64.3"
$ENVTEST_K8S_VERSION = "1.31.0"
$GOVULNCHECK_VERSION = "latest"

Write-Host "=== Installing Exact CI Tool Versions ===" -ForegroundColor Green
Write-Host "Go Version: $GO_VERSION" -ForegroundColor Cyan
Write-Host "golangci-lint: $GOLANGCI_LINT_VERSION" -ForegroundColor Cyan 
Write-Host "controller-gen: $CONTROLLER_GEN_VERSION" -ForegroundColor Cyan
Write-Host "envtest K8s: $ENVTEST_K8S_VERSION" -ForegroundColor Cyan
Write-Host ""

# Create install directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "Created install directory: $InstallDir" -ForegroundColor Yellow
}

# Add to PATH if not already there
$currentPath = $env:PATH
if ($currentPath -notlike "*$InstallDir*") {
    $env:PATH = "$InstallDir;$currentPath"
    Write-Host "Added $InstallDir to PATH for this session" -ForegroundColor Yellow
}

# Function to check if tool needs installation
function Test-ToolVersion {
    param($Command, $ExpectedVersion, $VersionArg = "--version")
    
    try {
        $output = & $Command $VersionArg 2>&1 | Out-String
        if ($output -match $ExpectedVersion.Replace("v", "")) {
            return $true
        }
    }
    catch {
        return $false
    }
    return $false
}

# 1. Verify Go Version
Write-Host "[1/5] Checking Go version..." -ForegroundColor Blue
try {
    $goOutput = go version 2>&1 | Out-String
    if ($goOutput -match "go$GO_VERSION") {
        Write-Host "✅ Go $GO_VERSION already installed" -ForegroundColor Green
    } else {
        Write-Host "❌ Go version mismatch. Expected: $GO_VERSION" -ForegroundColor Red
        Write-Host "Current: $goOutput" -ForegroundColor Gray
        Write-Host "Please install Go $GO_VERSION from https://golang.org/dl/" -ForegroundColor Yellow
        exit 1
    }
}
catch {
    Write-Host "❌ Go not found. Please install Go $GO_VERSION" -ForegroundColor Red
    exit 1
}

# 2. Install golangci-lint
Write-Host "[2/5] Installing golangci-lint $GOLANGCI_LINT_VERSION..." -ForegroundColor Blue
$golangciPath = Join-Path $InstallDir "golangci-lint.exe"

if ((Test-Path $golangciPath) -and !(Test-ToolVersion "golangci-lint" $GOLANGCI_LINT_VERSION) -or $Force) {
    Write-Host "Installing golangci-lint $GOLANGCI_LINT_VERSION..." -ForegroundColor Yellow
    
    # Download Windows binary
    $downloadUrl = "https://github.com/golangci/golangci-lint/releases/download/$GOLANGCI_LINT_VERSION/golangci-lint-${GOLANGCI_LINT_VERSION}-windows-amd64.zip"
    $tempZip = "$env:TEMP\golangci-lint.zip"
    $tempDir = "$env:TEMP\golangci-lint-extract"
    
    try {
        Write-Host "Downloading from: $downloadUrl" -ForegroundColor Gray
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip -UseBasicParsing
        
        if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
        Expand-Archive -Path $tempZip -DestinationPath $tempDir
        
        $extractedExe = Get-ChildItem -Path $tempDir -Recurse -Name "golangci-lint.exe" | Select-Object -First 1
        if ($extractedExe) {
            $sourcePath = Join-Path $tempDir (Split-Path $extractedExe -Parent) "golangci-lint.exe"
            Copy-Item $sourcePath $golangciPath -Force
            Write-Host "✅ golangci-lint installed to $golangciPath" -ForegroundColor Green
        } else {
            throw "golangci-lint.exe not found in archive"
        }
        
        # Cleanup
        Remove-Item $tempZip -ErrorAction SilentlyContinue
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        
        # Verify installation
        $version = & $golangciPath version 2>&1 | Out-String
        Write-Host "Installed version: $($version.Trim())" -ForegroundColor Gray
        
    }
    catch {
        Write-Host "❌ Failed to install golangci-lint: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✅ golangci-lint $GOLANGCI_LINT_VERSION already installed" -ForegroundColor Green
}

# 3. Install controller-gen
Write-Host "[3/5] Installing controller-gen $CONTROLLER_GEN_VERSION..." -ForegroundColor Blue
$env:GOBIN = $InstallDir

try {
    if ($Force -or !(Test-ToolVersion "controller-gen" $CONTROLLER_GEN_VERSION.Replace("v", ""))) {
        Write-Host "Installing controller-gen $CONTROLLER_GEN_VERSION..." -ForegroundColor Yellow
        go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$CONTROLLER_GEN_VERSION"
        Write-Host "✅ controller-gen installed" -ForegroundColor Green
    } else {
        Write-Host "✅ controller-gen $CONTROLLER_GEN_VERSION already installed" -ForegroundColor Green  
    }
}
catch {
    Write-Host "❌ Failed to install controller-gen: $_" -ForegroundColor Red
    exit 1
}

# 4. Install setup-envtest
Write-Host "[4/5] Installing setup-envtest..." -ForegroundColor Blue
try {
    if ($Force -or !(Get-Command "setup-envtest" -ErrorAction SilentlyContinue)) {
        Write-Host "Installing setup-envtest..." -ForegroundColor Yellow
        go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
        Write-Host "✅ setup-envtest installed" -ForegroundColor Green
    } else {
        Write-Host "✅ setup-envtest already installed" -ForegroundColor Green
    }
}
catch {
    Write-Host "❌ Failed to install setup-envtest: $_" -ForegroundColor Red
    exit 1
}

# 5. Install govulncheck
Write-Host "[5/5] Installing govulncheck..." -ForegroundColor Blue
try {
    if ($Force -or !(Get-Command "govulncheck" -ErrorAction SilentlyContinue)) {
        Write-Host "Installing govulncheck $GOVULNCHECK_VERSION..." -ForegroundColor Yellow
        go install "golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION"
        Write-Host "✅ govulncheck installed" -ForegroundColor Green
    } else {
        Write-Host "✅ govulncheck already installed" -ForegroundColor Green
    }
}
catch {
    Write-Host "❌ Failed to install govulncheck: $_" -ForegroundColor Red
    exit 1
}

# Setup envtest binaries
Write-Host ""
Write-Host "=== Setting up envtest binaries ===" -ForegroundColor Green
try {
    Write-Host "Downloading Kubernetes $ENVTEST_K8S_VERSION test binaries..." -ForegroundColor Blue
    $env:KUBEBUILDER_ASSETS = & setup-envtest use $ENVTEST_K8S_VERSION --arch=amd64 --os=windows -p path
    Write-Host "✅ KUBEBUILDER_ASSETS set to: $env:KUBEBUILDER_ASSETS" -ForegroundColor Green
    
    # Save to environment file for later use
    $envFile = Join-Path (Get-Location) ".env.ci"
    "KUBEBUILDER_ASSETS=$env:KUBEBUILDER_ASSETS" | Out-File -FilePath $envFile -Encoding UTF8
    Write-Host "✅ Environment saved to $envFile" -ForegroundColor Green
}
catch {
    Write-Host "⚠️ Failed to setup envtest binaries: $_" -ForegroundColor Yellow
    Write-Host "This may be needed for controller tests" -ForegroundColor Gray
}

Write-Host ""
Write-Host "=== Installation Summary ===" -ForegroundColor Green
Write-Host "Tools installed in: $InstallDir" -ForegroundColor Cyan
Write-Host "Add this to your permanent PATH:" -ForegroundColor Yellow
Write-Host "  [Environment]::SetEnvironmentVariable('PATH', '$InstallDir;' + [Environment]::GetEnvironmentVariable('PATH', 'User'), 'User')" -ForegroundColor Gray

# Verification
Write-Host ""
Write-Host "=== Verification ===" -ForegroundColor Green
$tools = @(
    @{Name="go"; Command="go version"}
    @{Name="golangci-lint"; Command="golangci-lint version"}  
    @{Name="controller-gen"; Command="controller-gen --version"}
    @{Name="setup-envtest"; Command="setup-envtest --help"}
    @{Name="govulncheck"; Command="govulncheck -version"}
)

foreach ($tool in $tools) {
    try {
        $output = Invoke-Expression $tool.Command 2>&1 | Out-String
        $firstLine = ($output -split "`n")[0].Trim()
        Write-Host "✅ $($tool.Name): $firstLine" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ $($tool.Name): Not available" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "🎉 CI tool installation complete!" -ForegroundColor Green
Write-Host "Run './scripts/run-ci-locally.ps1' to execute CI jobs locally" -ForegroundColor Cyan
#!/usr/bin/env pwsh
# Script: 01-install-porch.ps1
# Purpose: Install and verify Porch (porchctl/kpt) with specific versions

param(
    [string]$KptVersion = "v1.0.0-beta.54",
    [string]$PorchVersion = "v0.0.21",
    [switch]$SkipInstall = $false
)

$ErrorActionPreference = "Stop"

Write-Host "==== MVP Demo: Install Porch Components ====" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray

# Function to check if command exists
function Test-Command {
    param([string]$Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Function to download and install binary
function Install-Binary {
    param(
        [string]$Name,
        [string]$Version,
        [string]$Url
    )
    
    Write-Host "`nInstalling $Name $Version..." -ForegroundColor Yellow
    
    $tempDir = Join-Path $env:TEMP "mvp-install"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    $fileName = if ($IsWindows) { "$Name.exe" } else { $Name }
    $downloadPath = Join-Path $tempDir $fileName
    
    try {
        Write-Host "Downloading from: $Url"
        Invoke-WebRequest -Uri $Url -OutFile $downloadPath -UseBasicParsing
        
        if ($IsWindows) {
            # Move to user's local bin directory
            $binDir = Join-Path $env:USERPROFILE "bin"
            New-Item -ItemType Directory -Path $binDir -Force | Out-Null
            Move-Item -Path $downloadPath -Destination (Join-Path $binDir $fileName) -Force
            
            # Add to PATH if not already there
            $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
            if ($currentPath -notlike "*$binDir*") {
                [Environment]::SetEnvironmentVariable("Path", "$currentPath;$binDir", "User")
                $env:Path = "$env:Path;$binDir"
                Write-Host "Added $binDir to PATH"
            }
        } else {
            # For Linux/Mac
            chmod +x $downloadPath
            sudo mv $downloadPath /usr/local/bin/$Name
        }
        
        Write-Host "$Name installed successfully!" -ForegroundColor Green
    } catch {
        Write-Error "Failed to install ${Name}: $_"
        exit 1
    } finally {
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Check and install kpt
if (-not $SkipInstall) {
    if (Test-Command "kpt") {
        Write-Host "`nkpt is already installed" -ForegroundColor Green
    } else {
        $os = if ($IsWindows) { "windows" } elseif ($IsMacOS) { "darwin" } else { "linux" }
        $arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
        $kptUrl = "https://github.com/kptdev/kpt/releases/download/$KptVersion/kpt_${os}_${arch}"
        if ($IsWindows) { $kptUrl += ".exe" }
        
        Install-Binary -Name "kpt" -Version $KptVersion -Url $kptUrl
    }
    
    # Check and install porchctl
    if (Test-Command "porchctl") {
        Write-Host "`nporchctl is already installed" -ForegroundColor Green
    } else {
        $os = if ($IsWindows) { "windows" } elseif ($IsMacOS) { "darwin" } else { "linux" }
        $arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
        $porchUrl = "https://github.com/nephio-project/porch/releases/download/$PorchVersion/porchctl_${os}_${arch}"
        if ($IsWindows) { $porchUrl += ".exe" }
        
        Install-Binary -Name "porchctl" -Version $PorchVersion -Url $porchUrl
    }
}

# Verify installations
Write-Host "`n==== Verifying Installations ====" -ForegroundColor Cyan

# Check kpt
if (Test-Command "kpt") {
    Write-Host "`nkpt version:" -ForegroundColor Yellow
    & kpt version
} else {
    Write-Error "kpt is not installed or not in PATH"
    exit 1
}

# Check porchctl
if (Test-Command "porchctl") {
    Write-Host "`nporchctl version:" -ForegroundColor Yellow
    & porchctl version 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Note: porchctl version command may require connection to Porch server" -ForegroundColor Gray
    }
} else {
    Write-Error "porchctl is not installed or not in PATH"
    exit 1
}

# Check kubectl (required for Porch operations)
if (Test-Command "kubectl") {
    Write-Host "`nkubectl version:" -ForegroundColor Yellow
    & kubectl version --client --short 2>$null
} else {
    Write-Warning "kubectl is not installed. It will be required for Porch operations."
    Write-Host "Install kubectl from: https://kubernetes.io/docs/tasks/tools/" -ForegroundColor Yellow
}

# Check if Porch is deployed in cluster
Write-Host "`n==== Checking Porch Deployment ====" -ForegroundColor Cyan
if (Test-Command "kubectl") {
    $porchNamespace = "porch-system"
    $porchDeployment = kubectl get deployment -n $porchNamespace porch-server 2>$null
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Porch server is deployed in namespace: $porchNamespace" -ForegroundColor Green
        kubectl get pods -n $porchNamespace 2>$null | Select-String "porch"
    } else {
        Write-Warning "Porch server is not deployed in the cluster"
        Write-Host "To deploy Porch, run:" -ForegroundColor Yellow
        Write-Host "  kubectl apply -f https://github.com/nephio-project/porch/releases/download/$PorchVersion/porch-server.yaml"
    }
}

Write-Host "`n==== Installation Summary ====" -ForegroundColor Green
Write-Host "✓ kpt: $(if (Test-Command 'kpt') { 'Installed' } else { 'Not found' })"
Write-Host "✓ porchctl: $(if (Test-Command 'porchctl') { 'Installed' } else { 'Not found' })"
Write-Host "✓ kubectl: $(if (Test-Command 'kubectl') { 'Installed' } else { 'Not found' })"

Write-Host "`nInstallation complete! Next step: Run 02-prepare-nf-sim.ps1" -ForegroundColor Cyan
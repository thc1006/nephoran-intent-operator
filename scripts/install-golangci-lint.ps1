# Install golangci-lint v1.64.3 locally
# This script ensures the exact version used in CI is installed locally

param(
    [string]$Version = "v1.64.3",
    [string]$InstallDir = "$env:USERPROFILE\go\bin"
)

$ErrorActionPreference = "Stop"

Write-Host "🔧 Installing golangci-lint $Version for local CI reproduction..." -ForegroundColor Green

# Ensure Go is installed and GOPATH is set
if (-not $env:GOPATH) {
    $env:GOPATH = "$env:USERPROFILE\go"
}

# Create install directory if it doesn't exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "✅ Created directory: $InstallDir"
}

# Add to PATH if not already present
$currentPath = [Environment]::GetEnvironmentVariable("PATH", [System.EnvironmentVariableTarget]::User)
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host "📝 Adding $InstallDir to PATH..."
    [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallDir", [System.EnvironmentVariableTarget]::User)
    $env:PATH = "$env:PATH;$InstallDir"
    Write-Host "✅ Added to PATH (restart shell or use refreshenv)"
}

# Check if already installed with correct version
$existingVersion = $null
try {
    $existingVersion = & golangci-lint version 2>$null | Select-String "golangci-lint has version (\S+)" | ForEach-Object { $_.Matches[0].Groups[1].Value }
} catch {
    # Command not found or error - will install
}

if ($existingVersion -eq $Version.TrimStart('v')) {
    Write-Host "✅ golangci-lint $Version is already installed at correct version" -ForegroundColor Green
    exit 0
}

Write-Host "📥 Downloading and installing golangci-lint $Version..."

# Method 1: Try Go install (fastest for Go 1.24+)
try {
    Write-Host "🔄 Attempting installation via 'go install'..."
    $env:CGO_ENABLED = "0"
    & go install "github.com/golangci/golangci-lint/cmd/golangci-lint@$Version"
    
    # Verify installation
    $newVersion = & "$InstallDir\golangci-lint.exe" version 2>$null | Select-String "golangci-lint has version (\S+)" | ForEach-Object { $_.Matches[0].Groups[1].Value }
    if ($newVersion -eq $Version.TrimStart('v')) {
        Write-Host "✅ Successfully installed golangci-lint $Version via go install" -ForegroundColor Green
        Write-Host "📍 Installed at: $InstallDir\golangci-lint.exe"
        & "$InstallDir\golangci-lint.exe" version
        exit 0
    }
} catch {
    Write-Host "⚠️ Go install method failed, trying direct download..." -ForegroundColor Yellow
}

# Method 2: Direct download (Windows binary)
try {
    Write-Host "🔄 Downloading Windows binary from GitHub releases..."
    
    # Determine architecture
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $downloadUrl = "https://github.com/golangci/golangci-lint/releases/download/$Version/golangci-lint-$($Version.TrimStart('v'))-windows-$arch.zip"
    
    Write-Host "🌐 Download URL: $downloadUrl"
    
    # Download to temp location
    $tempDir = [System.IO.Path]::GetTempPath()
    $zipFile = Join-Path $tempDir "golangci-lint-$Version.zip"
    $extractDir = Join-Path $tempDir "golangci-lint-$Version"
    
    # Clean up any existing files
    if (Test-Path $zipFile) { Remove-Item $zipFile -Force }
    if (Test-Path $extractDir) { Remove-Item $extractDir -Recurse -Force }
    
    # Download with proper TLS settings
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $webClient = New-Object System.Net.WebClient
    $webClient.Headers.Add("User-Agent", "golangci-lint-installer")
    $webClient.DownloadFile($downloadUrl, $zipFile)
    
    Write-Host "✅ Downloaded to: $zipFile"
    
    # Extract
    Write-Host "📦 Extracting archive..."
    Expand-Archive -Path $zipFile -DestinationPath $extractDir -Force
    
    # Find the executable
    $exePath = Get-ChildItem -Path $extractDir -Name "golangci-lint.exe" -Recurse | Select-Object -First 1
    if (-not $exePath) {
        throw "golangci-lint.exe not found in extracted archive"
    }
    
    $fullExePath = Join-Path (Split-Path $exePath.FullName) "golangci-lint.exe"
    
    # Copy to install directory
    Write-Host "📋 Installing to: $InstallDir\golangci-lint.exe"
    Copy-Item -Path $fullExePath -Destination "$InstallDir\golangci-lint.exe" -Force
    
    # Clean up temporary files
    Remove-Item $zipFile -Force -ErrorAction SilentlyContinue
    Remove-Item $extractDir -Recurse -Force -ErrorAction SilentlyContinue
    
    # Verify installation
    $finalVersion = & "$InstallDir\golangci-lint.exe" version 2>$null | Select-String "golangci-lint has version (\S+)" | ForEach-Object { $_.Matches[0].Groups[1].Value }
    if ($finalVersion -eq $Version.TrimStart('v')) {
        Write-Host "✅ Successfully installed golangci-lint $Version via direct download" -ForegroundColor Green
        Write-Host "📍 Installed at: $InstallDir\golangci-lint.exe"
        & "$InstallDir\golangci-lint.exe" version
        exit 0
    } else {
        throw "Version verification failed. Expected: $($Version.TrimStart('v')), Got: $finalVersion"
    }
    
} catch {
    Write-Host "❌ Direct download method failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Method 3: Use chocolatey if available
if (Get-Command choco -ErrorAction SilentlyContinue) {
    try {
        Write-Host "🔄 Attempting installation via Chocolatey..."
        & choco install golangci-lint --version="$($Version.TrimStart('v'))" --yes --no-progress
        
        # Verify
        $chocoVersion = & golangci-lint version 2>$null | Select-String "golangci-lint has version (\S+)" | ForEach-Object { $_.Matches[0].Groups[1].Value }
        if ($chocoVersion -eq $Version.TrimStart('v')) {
            Write-Host "✅ Successfully installed golangci-lint $Version via Chocolatey" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "⚠️ Chocolatey method failed" -ForegroundColor Yellow
    }
}

Write-Host "❌ All installation methods failed. Please install manually:" -ForegroundColor Red
Write-Host "   1. Download from: https://github.com/golangci/golangci-lint/releases/tag/$Version"
Write-Host "   2. Extract golangci-lint.exe to: $InstallDir"
Write-Host "   3. Add $InstallDir to your PATH"
exit 1
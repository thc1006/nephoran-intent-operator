# Install nektos/act for local GitHub Actions simulation
# This allows running GitHub Actions workflows locally

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:USERPROFILE\bin",
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"

Write-Host "🎭 Installing nektos/act for local GitHub Actions simulation..." -ForegroundColor Green

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

# Check if already installed
$actPath = "$InstallDir\act.exe"
if ((Test-Path $actPath) -and -not $Force) {
    try {
        $existingVersion = & $actPath --version 2>$null
        Write-Host "✅ act is already installed: $existingVersion" -ForegroundColor Green
        Write-Host "Use -Force to reinstall"
        exit 0
    } catch {
        Write-Host "⚠️ Existing installation appears corrupted, reinstalling..." -ForegroundColor Yellow
    }
}

# Method 1: Try winget (Windows Package Manager)
if (Get-Command winget -ErrorAction SilentlyContinue) {
    try {
        Write-Host "🔄 Attempting installation via winget..."
        & winget install nektos.act
        
        # Verify installation
        $newVersion = & act --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Successfully installed act via winget: $newVersion" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "⚠️ winget method failed, trying direct download..." -ForegroundColor Yellow
    }
}

# Method 2: Try chocolatey
if (Get-Command choco -ErrorAction SilentlyContinue) {
    try {
        Write-Host "🔄 Attempting installation via Chocolatey..."
        & choco install act-cli --yes --no-progress
        
        # Verify installation
        $newVersion = & act --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Successfully installed act via Chocolatey: $newVersion" -ForegroundColor Green
            exit 0
        }
    } catch {
        Write-Host "⚠️ Chocolatey method failed, trying direct download..." -ForegroundColor Yellow
    }
}

# Method 3: Direct download from GitHub
try {
    Write-Host "🔄 Downloading act from GitHub releases..."
    
    # Get latest release info
    if ($Version -eq "latest") {
        Write-Host "🔍 Getting latest release info..."
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $apiResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/nektos/act/releases/latest" -Headers @{"User-Agent"="act-installer"}
        $Version = $apiResponse.tag_name
        Write-Host "📋 Latest version: $Version"
    }
    
    # Determine architecture
    $arch = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i386" }
    $downloadUrl = "https://github.com/nektos/act/releases/download/$Version/act_Windows_$arch.zip"
    
    Write-Host "🌐 Download URL: $downloadUrl"
    
    # Download to temp location
    $tempDir = [System.IO.Path]::GetTempPath()
    $zipFile = Join-Path $tempDir "act-$Version.zip"
    $extractDir = Join-Path $tempDir "act-$Version"
    
    # Clean up any existing files
    if (Test-Path $zipFile) { Remove-Item $zipFile -Force }
    if (Test-Path $extractDir) { Remove-Item $extractDir -Recurse -Force }
    
    # Download
    $webClient = New-Object System.Net.WebClient
    $webClient.Headers.Add("User-Agent", "act-installer")
    $webClient.DownloadFile($downloadUrl, $zipFile)
    
    Write-Host "✅ Downloaded to: $zipFile"
    
    # Extract
    Write-Host "📦 Extracting archive..."
    Expand-Archive -Path $zipFile -DestinationPath $extractDir -Force
    
    # Find the executable
    $exePath = Get-ChildItem -Path $extractDir -Name "act.exe" -Recurse | Select-Object -First 1
    if (-not $exePath) {
        throw "act.exe not found in extracted archive"
    }
    
    # Copy to install directory
    Write-Host "📋 Installing to: $actPath"
    Copy-Item -Path $exePath.FullName -Destination $actPath -Force
    
    # Clean up temporary files
    Remove-Item $zipFile -Force -ErrorAction SilentlyContinue
    Remove-Item $extractDir -Recurse -Force -ErrorAction SilentlyContinue
    
    # Verify installation
    $finalVersion = & $actPath --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Successfully installed act via direct download: $finalVersion" -ForegroundColor Green
        
        # Check Docker dependency
        Write-Host ""
        Write-Host "🐳 Checking Docker dependency..."
        try {
            $dockerVersion = & docker --version 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✅ Docker found: $dockerVersion" -ForegroundColor Green
            } else {
                throw "Docker not found"
            }
        } catch {
            Write-Host "⚠️ Docker not found. act requires Docker to run GitHub Actions locally." -ForegroundColor Yellow
            Write-Host "📋 Please install Docker Desktop from: https://www.docker.com/products/docker-desktop/"
            Write-Host "📋 Or install via winget: winget install Docker.DockerDesktop"
        }
        
        Write-Host ""
        Write-Host "💡 USAGE EXAMPLES:"
        Write-Host "  # List available workflows"
        Write-Host "  act -l"
        Write-Host ""
        Write-Host "  # Run CI workflow"
        Write-Host "  act -W .github/workflows/main-ci.yml"
        Write-Host ""
        Write-Host "  # Run specific job"
        Write-Host "  act -j linting"
        Write-Host ""
        Write-Host "  # Run with specific runner (use ubuntu-latest image)"
        Write-Host "  act -P ubuntu-latest=catthehacker/ubuntu:act-latest"
        
        exit 0
    } else {
        throw "Installation verification failed"
    }
    
} catch {
    Write-Host "❌ Direct download method failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "❌ All installation methods failed. Please install manually:" -ForegroundColor Red
Write-Host "   1. Download from: https://github.com/nektos/act/releases"
Write-Host "   2. Extract act.exe to: $InstallDir"
Write-Host "   3. Install Docker Desktop if not already installed"
Write-Host "   4. Add $InstallDir to your PATH"
exit 1
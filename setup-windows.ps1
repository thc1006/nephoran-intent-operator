# Windows Development Environment Setup for Nephoran Intent Operator
# This script sets up the development environment on Windows 11 with proper path handling

param(
    [switch]$Force,
    [switch]$SkipKubernetes,
    [switch]$Help
)

if ($Help) {
    Write-Host "Nephoran Intent Operator - Windows Development Setup"
    Write-Host ""
    Write-Host "Usage: .\setup-windows.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Force           Force reinstall even if components exist"
    Write-Host "  -SkipKubernetes  Skip Kubernetes cluster setup"
    Write-Host "  -Help            Show this help message"
    Write-Host ""
    exit 0
}

# Function to check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Function to install Chocolatey if not present
function Install-Chocolatey {
    if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
        Write-Host "Installing Chocolatey package manager..." -ForegroundColor Yellow
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    } else {
        Write-Host "Chocolatey already installed" -ForegroundColor Green
    }
}

# Function to install or verify Go
function Install-Go {
    $goVersion = "1.24"
    if ((Get-Command go -ErrorAction SilentlyContinue) -and !$Force) {
        $currentVersion = (go version).Split(" ")[2].Replace("go", "")
        if ($currentVersion -ge $goVersion) {
            Write-Host "Go $currentVersion is already installed" -ForegroundColor Green
            return
        }
    }
    
    Write-Host "Installing Go $goVersion..." -ForegroundColor Yellow
    choco install golang --version=$goVersion -y
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# Function to install or verify Python
function Install-Python {
    if ((Get-Command python -ErrorAction SilentlyContinue) -and !$Force) {
        Write-Host "Python is already installed" -ForegroundColor Green
        return
    }
    
    Write-Host "Installing Python..." -ForegroundColor Yellow
    choco install python3 -y
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# Function to install Docker Desktop
function Install-Docker {
    if ((Get-Command docker -ErrorAction SilentlyContinue) -and !$Force) {
        Write-Host "Docker is already installed" -ForegroundColor Green
        return
    }
    
    Write-Host "Installing Docker Desktop..." -ForegroundColor Yellow
    choco install docker-desktop -y
    Write-Host "Please restart your computer after Docker installation completes" -ForegroundColor Red
}

# Function to install kubectl
function Install-Kubectl {
    if ((Get-Command kubectl -ErrorAction SilentlyContinue) -and !$Force) {
        Write-Host "kubectl is already installed" -ForegroundColor Green
        return
    }
    
    Write-Host "Installing kubectl..." -ForegroundColor Yellow
    choco install kubernetes-cli -y
}

# Function to install kind (Kubernetes in Docker)
function Install-Kind {
    if ((Get-Command kind -ErrorAction SilentlyContinue) -and !$Force) {
        Write-Host "kind is already installed" -ForegroundColor Green
        return
    }
    
    Write-Host "Installing kind..." -ForegroundColor Yellow
    choco install kind -y
}

# Function to install Git
function Install-Git {
    if ((Get-Command git -ErrorAction SilentlyContinue) -and !$Force) {
        Write-Host "Git is already installed" -ForegroundColor Green
        return
    }
    
    Write-Host "Installing Git..." -ForegroundColor Yellow
    choco install git -y
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# Function to setup Python environment without pyenv
function Setup-PythonEnvironment {
    Write-Host "Setting up Python environment..." -ForegroundColor Yellow
    
    # Create virtual environment if it doesn't exist
    if (!(Test-Path "venv")) {
        python -m venv venv
    }
    
    # Activate virtual environment
    & ".\venv\Scripts\Activate.ps1"
    
    # Install requirements
    if (Test-Path "requirements-rag.txt") {
        python -m pip install --upgrade pip
        python -m pip install -r requirements-rag.txt
        Write-Host "Python dependencies installed successfully" -ForegroundColor Green
    } else {
        Write-Host "requirements-rag.txt not found" -ForegroundColor Red
    }
}

# Function to create Windows-compatible environment script
function Create-WindowsEnvScript {
    $envScript = @"
@echo off
REM Windows Environment Setup for Nephoran Intent Operator
REM This script replaces Unix-specific commands with Windows equivalents

REM Set GOPATH if not set
if "%GOPATH%"=="" (
    for /f "tokens=*" %%i in ('go env GOPATH') do set GOPATH=%%i
)

REM Add Go bin to PATH if not present
echo %PATH% | find /i "%GOPATH%\bin" >nul
if errorlevel 1 (
    set PATH=%PATH%;%GOPATH%\bin
)

REM Function equivalent to 'cygpath' for path conversion
set "UNIX_PATH_CONVERTER=powershell -Command ""(Get-Location).Path.Replace('\', '/')"""

REM Set common development variables
set GOOS=windows
set GOARCH=amd64

echo Windows development environment configured
echo GOPATH: %GOPATH%
echo PATH includes Go bin: %GOPATH%\bin
"@
    
    $envScript | Out-File -FilePath "windows-env.bat" -Encoding ASCII
    Write-Host "Created windows-env.bat for environment setup" -ForegroundColor Green
}

# Function to setup local Kubernetes cluster
function Setup-KubernetesCluster {
    if ($SkipKubernetes) {
        Write-Host "Skipping Kubernetes cluster setup" -ForegroundColor Yellow
        return
    }
    
    Write-Host "Setting up local Kubernetes cluster with kind..." -ForegroundColor Yellow
    
    # Check if Docker is running
    try {
        docker info | Out-Null
    } catch {
        Write-Host "Docker is not running. Please start Docker Desktop and run this script again." -ForegroundColor Red
        return
    }
    
    # Create kind cluster if it doesn't exist
    $clusters = kind get clusters 2>$null
    if ($clusters -notcontains "nephoran-dev") {
        Write-Host "Creating kind cluster 'nephoran-dev'..." -ForegroundColor Yellow
        
        $kindConfig = @"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: nephoran-dev
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
"@
        
        $kindConfig | Out-File -FilePath "kind-config.yaml" -Encoding UTF8
        kind create cluster --config kind-config.yaml
        Remove-Item "kind-config.yaml"
        
        Write-Host "Kind cluster created successfully" -ForegroundColor Green
    } else {
        Write-Host "Kind cluster 'nephoran-dev' already exists" -ForegroundColor Green
    }
    
    # Set kubectl context
    kubectl cluster-info --context kind-nephoran-dev
}

# Function to verify setup
function Verify-Setup {
    Write-Host "`nVerifying installation..." -ForegroundColor Yellow
    
    $tools = @(
        @{Name="Go"; Command="go version"},
        @{Name="Python"; Command="python --version"},
        @{Name="Docker"; Command="docker --version"},
        @{Name="kubectl"; Command="kubectl version --client"},
        @{Name="kind"; Command="kind version"},
        @{Name="Git"; Command="git --version"}
    )
    
    foreach ($tool in $tools) {
        try {
            $output = Invoke-Expression $tool.Command 2>$null
            Write-Host "✓ $($tool.Name): $output" -ForegroundColor Green
        } catch {
            Write-Host "✗ $($tool.Name): Not installed or not in PATH" -ForegroundColor Red
        }
    }
    
    # Check if cluster is running
    if (!$SkipKubernetes) {
        try {
            kubectl cluster-info --context kind-nephoran-dev 2>$null | Out-Null
            Write-Host "✓ Kubernetes cluster: Running" -ForegroundColor Green
        } catch {
            Write-Host "✗ Kubernetes cluster: Not running" -ForegroundColor Red
        }
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - Windows Development Setup" -ForegroundColor Blue
Write-Host "======================================================" -ForegroundColor Blue

if (!(Test-Administrator)) {
    Write-Host "This script requires administrator privileges. Please run as administrator." -ForegroundColor Red
    exit 1
}

try {
    Install-Chocolatey
    Install-Go
    Install-Python
    Install-Docker
    Install-Kubectl
    Install-Kind
    Install-Git
    
    # Refresh PATH
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    
    Setup-PythonEnvironment
    Create-WindowsEnvScript
    Setup-KubernetesCluster
    
    Verify-Setup
    
    Write-Host "`nSetup completed successfully!" -ForegroundColor Green
    Write-Host "To activate the Python virtual environment, run: .\venv\Scripts\Activate.ps1" -ForegroundColor Yellow
    Write-Host "To setup environment variables, run: .\windows-env.bat" -ForegroundColor Yellow
    Write-Host "To build the project, run: make setup-dev && make build-all" -ForegroundColor Yellow
    
} catch {
    Write-Host "Setup failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
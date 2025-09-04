# Windows Setup Guide for E2E Testing

This guide provides detailed setup instructions for running Nephoran Intent Operator E2E tests on Windows environments.

## Prerequisites

### System Requirements

- **Windows 10/11** or **Windows Server 2019/2022**
- **PowerShell 5.1+** (PowerShell 7+ recommended)
- **Git for Windows** (includes Git Bash)
- **Docker Desktop** for Windows
- **Administrative privileges** for tool installation

### Required Tools Installation

#### 1. Docker Desktop

1. Download from: https://www.docker.com/products/docker-desktop/
2. Install with default settings
3. Ensure WSL 2 integration is enabled
4. Start Docker Desktop and verify:
```powershell
docker --version
docker run hello-world
```

#### 2. Chocolatey (Package Manager)

Install Chocolatey for easy tool management:

```powershell
# Run as Administrator
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Verify installation
choco --version
```

#### 3. Kind (Kubernetes in Docker)

```powershell
# Install via Chocolatey
choco install kind

# Or download manually
$kindVersion = "v0.20.0"
Invoke-WebRequest -Uri "https://kind.sigs.k8s.io/dl/$kindVersion/kind-windows-amd64" -OutFile "kind.exe"
Move-Item .\kind.exe $env:USERPROFILE\go\bin\kind.exe

# Verify installation
kind --version
```

#### 4. kubectl (Kubernetes CLI)

```powershell
# Install via Chocolatey
choco install kubernetes-cli

# Or download manually
$kubectlVersion = (Invoke-WebRequest -Uri "https://dl.k8s.io/release/stable.txt").Content.Trim()
Invoke-WebRequest -Uri "https://dl.k8s.io/release/$kubectlVersion/bin/windows/amd64/kubectl.exe" -OutFile "kubectl.exe"
Move-Item .\kubectl.exe $env:USERPROFILE\go\bin\kubectl.exe

# Verify installation
kubectl version --client
```

#### 5. Go (Programming Language)

```powershell
# Install via Chocolatey
choco install golang

# Or download from: https://golang.org/dl/
# Choose the Windows .msi installer

# Verify installation
go version

# Set up Go workspace
$env:GOPATH = "$env:USERPROFILE\go"
$env:PATH += ";$env:GOPATH\bin"
```

#### 6. Kustomize (Optional)

```powershell
# Install via Chocolatey
choco install kustomize

# Or download manually
$kustomizeVersion = "v5.1.1"
Invoke-WebRequest -Uri "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$kustomizeVersion/kustomize_$kustomizeVersion_windows_amd64.tar.gz" -OutFile "kustomize.tar.gz"
# Extract and move to PATH

# Verify installation
kustomize version
```

#### 7. Git for Windows

1. Download from: https://git-scm.com/download/win
2. Install with default settings
3. Ensure Git Bash is included
4. Verify installation:
```powershell
git --version
bash --version
```

## Environment Setup

### PowerShell Configuration

1. **Set Execution Policy**:
```powershell
# Run as Administrator
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

2. **Configure PowerShell Profile**:
```powershell
# Edit profile
notepad $PROFILE

# Add these lines to your profile:
$env:GOPATH = "$env:USERPROFILE\go"
$env:PATH += ";$env:GOPATH\bin"
$env:PATH += ";C:\ProgramData\chocolatey\bin"

# Enable useful features
Set-PSReadlineOption -BellStyle None
Set-PSReadlineKeyHandler -Key Tab -Function Complete
```

3. **Install PowerShell Modules** (optional):
```powershell
# Install useful modules
Install-Module -Name PSReadLine -Force
Install-Module -Name posh-git -Force
```

### PATH Configuration

Ensure these directories are in your PATH:

```powershell
# Check current PATH
$env:PATH -split ';'

# Add missing directories
$pathDirs = @(
    "$env:USERPROFILE\go\bin",
    "C:\ProgramData\chocolatey\bin",
    "C:\Program Files\Docker\Docker\resources\bin",
    "C:\Program Files\Git\bin"
)

foreach ($dir in $pathDirs) {
    if (Test-Path $dir) {
        if ($env:PATH -notlike "*$dir*") {
            $env:PATH += ";$dir"
        }
    }
}

# Make permanent (run as Administrator)
[Environment]::SetEnvironmentVariable("PATH", $env:PATH, [EnvironmentVariableTarget]::Machine)
```

## Running E2E Tests

### Option 1: Native PowerShell (Recommended)

```powershell
# Navigate to project root
cd C:\path\to\nephoran-intent-operator

# Run with default settings
.\hack\run-e2e.ps1

# Run with custom parameters
.\hack\run-e2e.ps1 -ClusterName "win-test" -Namespace "test-system" -SkipCleanup

# Run with verbose output
.\hack\run-e2e.ps1 -Verbose

# Show available parameters
Get-Help .\hack\run-e2e.ps1 -Detailed
```

### Option 2: Git Bash (Fallback)

```powershell
# Use Git Bash for bash script
.\hack\run-e2e.ps1 -UseBash

# Or run bash script directly
bash .\hack\run-e2e.sh
```

### Option 3: WSL (Advanced)

If you have WSL installed:

```powershell
# Run in WSL environment
wsl bash ./hack/run-e2e.sh
```

## Troubleshooting

### Common Issues

#### 1. "kind: command not found"

**Solution**:
```powershell
# Check if kind is installed
Get-Command kind -ErrorAction SilentlyContinue

# If not found, reinstall
choco uninstall kind
choco install kind

# Refresh PATH
refreshenv

# Or add manually to current session
$env:PATH += ";C:\ProgramData\chocolatey\bin"
```

#### 2. "kubectl: command not found"

**Solution**:
```powershell
# Check kubectl installation
Get-Command kubectl -ErrorAction SilentlyContinue

# Reinstall if needed
choco uninstall kubernetes-cli
choco install kubernetes-cli

# Test kubectl
kubectl version --client
```

#### 3. "Docker is not running"

**Solution**:
1. Start Docker Desktop
2. Wait for Docker to fully start (check system tray)
3. Test Docker:
```powershell
docker info
docker run hello-world
```

#### 4. "PowerShell execution policy error"

**Solution**:
```powershell
# Check current policy
Get-ExecutionPolicy

# Set to RemoteSigned (run as Administrator)
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser

# Or bypass for current session
Set-ExecutionPolicy Bypass -Scope Process
```

#### 5. "Kind cluster creation fails"

**Solution**:
```powershell
# Check Docker is running
docker ps

# Check available resources
docker system df

# Try creating cluster manually
kind create cluster --name test-cluster

# Check kind logs
kind get clusters
kind get kubeconfig --name test-cluster
```

#### 6. "Go build fails"

**Solution**:
```powershell
# Check Go installation
go version
go env

# Verify GOPATH
$env:GOPATH

# Clean Go modules
go clean -modcache

# Rebuild
go build ./...
```

### Windows-Specific Considerations

#### File Path Issues

Windows uses backslashes (`\`) while many tools expect forward slashes (`/`):

```powershell
# Convert paths when needed
$unixPath = $windowsPath -replace '\\', '/'

# Use Join-Path for cross-platform compatibility
$configPath = Join-Path $projectRoot "config\webhook"
```

#### Line Ending Issues

Git may convert line endings:

```powershell
# Check Git configuration
git config core.autocrlf

# Set to false to preserve line endings
git config --global core.autocrlf false

# Convert files if needed
dos2unix ./hack/run-e2e.sh
```

#### Permission Issues

Some operations require elevated privileges:

```powershell
# Run PowerShell as Administrator when needed
Start-Process PowerShell -Verb RunAs

# Check if running as Administrator
([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
```

## Performance Optimization

### Docker Desktop Settings

1. **Resource Allocation**:
   - Memory: 8GB+ recommended
   - CPU: 4+ cores recommended
   - Swap: 1GB+

2. **WSL 2 Backend**:
   - Enable WSL 2 for better performance
   - Allocate sufficient resources in `.wslconfig`

3. **File Sharing**:
   - Only share necessary directories
   - Use Docker volumes for better performance

### PowerShell Performance

```powershell
# Enable faster cmdlet discovery
$PSDefaultParameterValues = @{
    '*:Encoding' = 'UTF8'
    'Out-File:Encoding' = 'UTF8'
    'Export-*:Encoding' = 'UTF8'
}

# Use native commands when possible
function kubectl { & kubectl.exe $args }
function kind { & kind.exe $args }
function go { & go.exe $args }
```

## Development Environment

### IDE Setup

#### Visual Studio Code

1. Install VS Code: https://code.visualstudio.com/
2. Install extensions:
   - Go extension
   - Kubernetes extension
   - PowerShell extension
   - YAML extension

3. Configure settings:
```json
{
    "go.gopath": "%USERPROFILE%\\go",
    "go.goroot": "C:\\Program Files\\Go",
    "terminal.integrated.shell.windows": "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
}
```

#### GoLand (JetBrains)

1. Install GoLand: https://www.jetbrains.com/go/
2. Configure Go SDK path
3. Enable Kubernetes plugin
4. Set terminal to PowerShell

### Git Configuration

```powershell
# Configure Git for Windows
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
git config --global core.autocrlf false
git config --global core.longpaths true

# Configure line endings
git config --global core.eol lf
git config --global core.autocrlf input
```

## Testing in CI/CD

### GitHub Actions (Windows Runner)

```yaml
name: E2E Tests (Windows)
on: [push, pull_request]

jobs:
  test-windows:
    runs-on: windows-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Install tools
      run: |
        choco install kind kubectl -y
        refreshenv
    
    - name: Run E2E Tests
      run: .\hack\run-e2e.ps1
      env:
        CLUSTER_NAME: "win-ci-${{ github.run_id }}"
    
    - name: Upload logs
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: windows-e2e-logs
        path: C:\temp\nephoran-logs\
```

### Azure DevOps

```yaml
trigger:
- main

pool:
  vmImage: 'windows-latest'

steps:
- task: GoTool@0
  inputs:
    version: '1.21'

- powershell: |
    choco install kind kubectl -y
    refreshenv
  displayName: 'Install tools'

- powershell: |
    .\hack\run-e2e.ps1
  displayName: 'Run E2E tests'
  env:
    CLUSTER_NAME: "azure-ci-$(Build.BuildId)"
```

## Support

### Getting Help

1. **Project Issues**: https://github.com/nephoran/intent-operator/issues
2. **Kind Documentation**: https://kind.sigs.k8s.io/
3. **Kubectl Documentation**: https://kubernetes.io/docs/reference/kubectl/
4. **PowerShell Documentation**: https://docs.microsoft.com/en-us/powershell/

### Community Resources

- **Kubernetes Slack**: https://kubernetes.slack.com/
- **Kind GitHub**: https://github.com/kubernetes-sigs/kind
- **Stack Overflow**: Tag questions with `kubernetes`, `kind`, `windows`
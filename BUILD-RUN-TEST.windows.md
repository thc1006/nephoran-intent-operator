# Windows Development Guide for Nephoran E2E Testing

## Table of Contents

1. [Prerequisites and Tool Installation](#prerequisites-and-tool-installation)
2. [Git Bash Setup and Configuration](#git-bash-setup-and-configuration)
3. [Step-by-Step Execution](#step-by-step-execution)
4. [PowerShell Alternatives](#powershell-alternatives)
5. [Windows-Specific Troubleshooting](#windows-specific-troubleshooting)
6. [Development Workflow](#development-workflow)
7. [IDE Integration](#ide-integration)

## Prerequisites and Tool Installation

### System Requirements

**Minimum Requirements**:
- Windows 10 Version 1903+ or Windows 11
- 8 GB RAM (16 GB recommended)
- 20 GB free disk space
- Virtualization enabled in BIOS (for Docker)
- Administrator privileges for installation

### Essential Tools Installation

#### 1. Package Manager - Chocolatey

Open PowerShell as Administrator and install Chocolatey:

```powershell
# Install Chocolatey
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Verify installation
choco --version
```

#### 2. Docker Desktop

```powershell
# Install Docker Desktop
choco install docker-desktop -y

# After installation, restart your computer
# Start Docker Desktop from Start Menu
# Wait for Docker to be ready (system tray icon turns green)

# Verify Docker installation
docker --version
docker run hello-world
```

**Docker Desktop Settings**:
1. Open Docker Desktop Settings
2. General → Enable "Use the WSL 2 based engine" (recommended)
3. Resources → Advanced:
   - CPUs: 4+ (50% of available)
   - Memory: 4-8 GB
   - Disk image size: 50+ GB
4. Kubernetes → Enable Kubernetes (optional, for comparison testing)

#### 3. Kubernetes Tools

```powershell
# Install kubectl
choco install kubernetes-cli -y

# Install kind (Kubernetes in Docker)
choco install kind -y

# Install kustomize (optional but recommended)
choco install kustomize -y

# Verify installations
kubectl version --client
kind --version
kustomize version
```

#### 4. Go Programming Language

```powershell
# Install Go
choco install golang -y

# Verify installation
go version

# Set GOPATH (if not automatically set)
[Environment]::SetEnvironmentVariable("GOPATH", "$env:USERPROFILE\go", "User")
[Environment]::SetEnvironmentVariable("PATH", "$env:PATH;$env:USERPROFILE\go\bin", "User")

# Restart PowerShell to apply changes
```

#### 5. Git and Git Bash

```powershell
# Install Git (includes Git Bash)
choco install git -y

# Configure Git
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
git config --global core.autocrlf true  # Important for Windows
git config --global core.longpaths true  # Handle long paths

# Verify installation
git --version
```

#### 6. Additional Development Tools

```powershell
# Install useful utilities
choco install curl wget jq make -y

# Install VS Code (recommended IDE)
choco install vscode -y

# Install Windows Terminal (better terminal experience)
choco install microsoft-windows-terminal -y
```

### Manual Installation Alternative

If you prefer manual installation:

1. **Docker Desktop**: https://www.docker.com/products/docker-desktop/
2. **kubectl**: https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/
3. **kind**: https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries
4. **Go**: https://go.dev/dl/
5. **Git for Windows**: https://gitforwindows.org/

## Git Bash Setup and Configuration

### Initial Git Bash Configuration

1. **Launch Git Bash**:
   - Right-click in any folder → "Git Bash Here"
   - Or from Start Menu → Git → Git Bash

2. **Configure Bash Profile**:

Create/edit `~/.bashrc`:

```bash
# ~/.bashrc

# Set default editor
export EDITOR=code  # or vim, nano, notepad

# Aliases for common commands
alias ll='ls -la'
alias k='kubectl'
alias d='docker'
alias g='git'
alias gst='git status'
alias gco='git checkout'
alias gpl='git pull'
alias gps='git push'

# Kubernetes shortcuts
alias kgp='kubectl get pods'
alias kgs='kubectl get svc'
alias kgn='kubectl get nodes'
alias kaf='kubectl apply -f'
alias kdel='kubectl delete'
alias klog='kubectl logs'

# Project shortcuts
alias cdp='cd /c/Users/$USER/dev/nephoran'
alias e2e='./hack/run-e2e.sh'
alias e2eprod='./hack/run-production-e2e.sh'

# Color output
export CLICOLOR=1
export LSCOLORS=ExFxBxDxCxegedabagacad

# Go environment
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Docker Desktop for Windows fix
export DOCKER_HOST=tcp://localhost:2375

# Kind cluster name
export KIND_CLUSTER=nephoran-e2e

# Prompt customization
PS1='\[\033[32m\]\u@\h \[\033[33m\]\w\[\033[36m\]$(__git_ps1)\[\033[0m\]\n$ '

# Auto-completion for kubectl
if [ -f /usr/share/bash-completion/bash_completion ]; then
    . /usr/share/bash-completion/bash_completion
fi
source <(kubectl completion bash)
complete -F __start_kubectl k

# Function to quickly create and switch to kind cluster
kind-create() {
    kind create cluster --name ${1:-nephoran-e2e} --wait 2m
    kubectl cluster-info
}

# Function to delete kind cluster
kind-delete() {
    kind delete cluster --name ${1:-nephoran-e2e}
}

# Function to run E2E tests with logging
run-e2e() {
    echo "Starting E2E tests at $(date)"
    time ./hack/run-e2e.sh "$@" 2>&1 | tee e2e-$(date +%Y%m%d-%H%M%S).log
    echo "E2E tests completed at $(date)"
}
```

3. **Apply Configuration**:

```bash
# Reload bashrc
source ~/.bashrc

# Test aliases
k version --client
d version
```

### Path Configuration for Windows

Git Bash needs proper PATH configuration to find Windows executables:

```bash
# Add to ~/.bashrc
# Windows Program Files
export PATH="$PATH:/c/Program Files/Docker/Docker/resources/bin"
export PATH="$PATH:/c/ProgramData/DockerDesktop/version-bin"

# User-installed tools
export PATH="$PATH:/c/ProgramData/chocolatey/bin"
export PATH="$PATH:$HOME/.local/bin"
export PATH="$PATH:$HOME/go/bin"

# Fix for spaces in paths
export PATH="$PATH:/c/Program\ Files/Git/cmd"
```

### Handling Line Endings

Configure Git to handle Windows line endings properly:

```bash
# Global settings
git config --global core.autocrlf true
git config --global core.eol lf

# Project-specific settings (in project root)
echo "* text=auto eol=lf" > .gitattributes
echo "*.sh text eol=lf" >> .gitattributes
echo "*.bash text eol=lf" >> .gitattributes
echo "*.yaml text eol=lf" >> .gitattributes
echo "*.yml text eol=lf" >> .gitattributes
echo "*.json text eol=lf" >> .gitattributes
echo "*.go text eol=lf" >> .gitattributes
```

## Step-by-Step Execution

### Complete E2E Test Execution Walkthrough

#### Step 1: Clone and Navigate to Repository

```bash
# Open Git Bash
# Clone repository (if not already done)
git clone https://github.com/thc1006/nephoran-intent-operator.git
cd nephoran-intent-operator

# Or navigate to existing clone
cd /c/Users/$USER/dev/nephoran-intent-operator
```

#### Step 2: Verify Environment

```bash
# Check all tools are available
./hack/check-tools.sh

# Expected output:
# ✅ Docker: 24.0.0
# ✅ kind: v0.20.0
# ✅ kubectl: v1.30.0
# ✅ go: 1.24.0
# ✅ make: 4.3
# ✅ curl: 8.0.0
```

If any tools are missing:

```bash
# Install missing tools via Chocolatey (in PowerShell as Admin)
choco install <tool-name> -y

# Or download manually and add to PATH
```

#### Step 3: Build Components (Optional)

```bash
# Build all Go components
make build

# Or build specific components
go build -o bin/intent-ingest.exe ./cmd/intent-ingest
go build -o bin/conductor-loop.exe ./cmd/conductor-loop
go build -o bin/webhook-manager.exe ./cmd/webhook-manager
```

#### Step 4: Run Basic E2E Tests

```bash
# Simple execution
./hack/run-e2e.sh

# With options
CLUSTER_NAME=my-test SKIP_CLEANUP=true ./hack/run-e2e.sh

# Verbose mode with extended timeout
VERBOSE=true TIMEOUT=600 ./hack/run-e2e.sh
```

#### Step 5: Monitor Test Progress

Open a second Git Bash window for monitoring:

```bash
# Watch cluster status
watch kubectl get all -A

# Monitor pods in test namespace
kubectl get pods -n nephoran-system -w

# Check logs
kubectl logs -n nephoran-system -f deployment/webhook-manager
```

#### Step 6: Run Production Scenarios

```bash
# Run full production test suite
./hack/run-production-e2e.sh

# Run specific scenario
TEST_SCENARIO=security ./hack/run-production-e2e.sh

# Run with custom configuration
PARALLEL_TESTS=8 TEST_TIMEOUT=45m ./hack/run-production-e2e.sh
```

#### Step 7: Analyze Results

```bash
# View test summary
cat test-reports/e2e-summary.txt

# Check for failures
grep FAIL test-reports/*.log

# View generated artifacts
ls -la handoff/
ls -la test-reports/
```

#### Step 8: Debug Failed Tests

```bash
# Keep cluster for investigation
SKIP_CLEANUP=true ./hack/run-e2e.sh

# Inspect resources
kubectl get networkintents -A
kubectl describe networkintent <name> -n nephoran-system

# Check events
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'

# View controller logs
kubectl logs -n nephoran-system deployment/webhook-manager --tail=100
```

#### Step 9: Cleanup

```bash
# Manual cleanup if SKIP_CLEANUP was used
kind delete cluster --name nephoran-e2e

# Clean test artifacts
rm -rf handoff/*.json handoff/*.yaml
rm -rf test-reports/*

# Clean build artifacts
make clean
```

### Quick Test Commands

```bash
# Smoke test (minimal validation)
./hack/run-e2e-simple.sh

# Idempotent test (can run multiple times)
./hack/run-e2e-idempotent.sh

# CI validation
./hack/validate-e2e.sh

# Full test suite
make test-e2e
```

## PowerShell Alternatives

### Native PowerShell E2E Execution

#### Using PowerShell Script

```powershell
# Direct PowerShell execution
.\hack\run-e2e.ps1

# With parameters
.\hack\run-e2e.ps1 `
    -ClusterName "ps-test" `
    -Namespace "test-ns" `
    -SkipCleanup `
    -Verbose

# Using splatting for cleaner syntax
$params = @{
    ClusterName = "ps-test"
    Namespace = "test-ns"
    IntentIngestMode = "local"
    ConductorMode = "local"
    SkipCleanup = $true
    Verbose = $true
}
.\hack\run-e2e.ps1 @params
```

#### PowerShell Functions for E2E Testing

Add to your PowerShell profile (`$PROFILE`):

```powershell
# PowerShell Profile Functions

function Run-E2ETest {
    param(
        [string]$ClusterName = "nephoran-e2e",
        [switch]$SkipCleanup,
        [switch]$Verbose
    )
    
    Write-Host "Starting E2E Test..." -ForegroundColor Cyan
    $startTime = Get-Date
    
    & "$PSScriptRoot\hack\run-e2e.ps1" `
        -ClusterName $ClusterName `
        -SkipCleanup:$SkipCleanup `
        -Verbose:$Verbose
    
    $duration = (Get-Date) - $startTime
    Write-Host "Test completed in $($duration.TotalMinutes) minutes" -ForegroundColor Green
}

function Get-E2EStatus {
    Write-Host "Cluster Status:" -ForegroundColor Cyan
    kind get clusters
    
    Write-Host "`nPods Status:" -ForegroundColor Cyan
    kubectl get pods -A | Select-String "nephoran"
    
    Write-Host "`nNetworkIntents:" -ForegroundColor Cyan
    kubectl get networkintents -A
}

function Clean-E2E {
    param(
        [string]$ClusterName = "nephoran-e2e"
    )
    
    Write-Host "Cleaning up E2E resources..." -ForegroundColor Yellow
    
    # Delete cluster
    kind delete cluster --name $ClusterName
    
    # Clean artifacts
    Remove-Item -Path "handoff\*" -Force -ErrorAction SilentlyContinue
    Remove-Item -Path "test-reports\*" -Force -ErrorAction SilentlyContinue
    
    Write-Host "Cleanup complete" -ForegroundColor Green
}

function Watch-E2E {
    param(
        [int]$IntervalSeconds = 5
    )
    
    while ($true) {
        Clear-Host
        Write-Host "E2E Monitoring - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
        Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
        Write-Host ""
        
        Get-E2EStatus
        
        Start-Sleep -Seconds $IntervalSeconds
    }
}

# Aliases
Set-Alias e2e Run-E2ETest
Set-Alias e2e-status Get-E2EStatus
Set-Alias e2e-clean Clean-E2E
Set-Alias e2e-watch Watch-E2E
```

#### PowerShell Debugging Commands

```powershell
# Enable debug output
$DebugPreference = "Continue"
$VerbosePreference = "Continue"

# Trace execution
Set-PSDebug -Trace 1

# Run with transcript
Start-Transcript -Path "e2e-debug-$(Get-Date -Format 'yyyyMMdd-HHmmss').txt"
.\hack\run-e2e.ps1 -Verbose
Stop-Transcript

# Check execution policy
Get-ExecutionPolicy -List

# If scripts are blocked
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### WSL2 Alternative

If you prefer a Linux environment on Windows:

```powershell
# Install WSL2
wsl --install

# Install Ubuntu
wsl --install -d Ubuntu

# Enter WSL2
wsl

# Inside WSL2, install tools
sudo apt update
sudo apt install -y docker.io kubectl golang make curl

# Run E2E tests in WSL2
./hack/run-e2e.sh
```

## Windows-Specific Troubleshooting

### Common Windows Issues and Solutions

#### Issue 1: Docker Desktop Not Starting

**Problem**: Docker Desktop fails to start or shows "Docker Desktop is starting..."

**Solutions**:

```powershell
# 1. Enable virtualization in BIOS
# Restart and enter BIOS, enable Intel VT-x or AMD-V

# 2. Enable Hyper-V and WSL2
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart

# 3. Reset Docker Desktop
Stop-Process -Name "Docker Desktop" -Force -ErrorAction SilentlyContinue
Remove-Item "$env:APPDATA\Docker" -Recurse -Force
# Reinstall Docker Desktop

# 4. Check Windows version
winver  # Should be 1903 or higher
```

#### Issue 2: Path Too Long Errors

**Problem**: "The filename or extension is too long"

**Solutions**:

```powershell
# Enable long path support in Windows
New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" `
    -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force

# Enable in Git
git config --global core.longpaths true

# Use shorter paths
# Instead of: C:\Users\Username\Documents\Projects\nephoran-intent-operator
# Use: C:\dev\nephoran
```

#### Issue 3: Line Ending Issues

**Problem**: Scripts fail with "bad interpreter" or syntax errors

**Solutions**:

```bash
# In Git Bash
# Convert line endings for all shell scripts
find . -name "*.sh" -exec dos2unix {} \;

# Or use Git to fix
git config core.autocrlf true
git rm --cached -r .
git reset --hard

# Manual fix for single file
sed -i 's/\r$//' hack/run-e2e.sh
```

#### Issue 4: Permission Denied Errors

**Problem**: Cannot execute scripts or access files

**Solutions**:

```bash
# In Git Bash
# Make scripts executable
chmod +x hack/*.sh
chmod +x hack/*.ps1

# Fix ownership issues
# Run Git Bash as Administrator

# Or in PowerShell (as Administrator)
icacls "C:\dev\nephoran" /grant "${env:USERNAME}:(OI)(CI)F" /T
```

#### Issue 5: Network/Firewall Issues

**Problem**: Cannot connect to cluster or pull images

**Solutions**:

```powershell
# Check Windows Firewall
Get-NetFirewallRule | Where DisplayName -like "*Docker*"
Get-NetFirewallRule | Where DisplayName -like "*kubectl*"

# Add firewall rules
New-NetFirewallRule -DisplayName "Docker" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 2375,2376,6443
New-NetFirewallRule -DisplayName "Kubernetes" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 6443,10250-10252

# Check proxy settings
[System.Net.WebRequest]::DefaultWebProxy
$env:HTTP_PROXY
$env:HTTPS_PROXY
$env:NO_PROXY

# Bypass proxy for local
$env:NO_PROXY = "localhost,127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,.local"
```

#### Issue 6: Kind Cluster Issues

**Problem**: Kind cluster creation fails

**Solutions**:

```bash
# Check Docker is running
docker info

# Clean up existing clusters
kind delete clusters --all

# Try with explicit config
cat > kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
  - containerPort: 443
    hostPort: 443
EOF

kind create cluster --config kind-config.yaml --name nephoran-e2e

# Use specific image
kind create cluster --image kindest/node:v1.30.0 --name nephoran-e2e
```

### Performance Optimization on Windows

```powershell
# Allocate more resources to Docker Desktop
# Docker Desktop → Settings → Resources:
# - CPUs: 6-8 (for 12-16 core system)
# - Memory: 8-12 GB
# - Swap: 2-4 GB
# - Disk: 100+ GB

# Disable Windows Defender for development folders (temporary)
Add-MpPreference -ExclusionPath "C:\dev"
Add-MpPreference -ExclusionProcess "docker.exe","dockerd.exe","kubectl.exe","kind.exe"

# Use SSD for development
# Move Docker images to SSD
# Docker Desktop → Settings → Advanced → Disk image location

# Disable indexing for development folders
# Right-click folder → Properties → Advanced → Uncheck "Allow files to be indexed"
```

## Development Workflow

### Typical Development Cycle

```bash
# 1. Start your day - Update and verify environment
git pull origin main
./hack/check-tools.sh
docker info
kind get clusters

# 2. Create feature branch
git checkout -b feature/my-feature

# 3. Make changes
code .  # Open in VS Code

# 4. Test locally
make build
./hack/run-e2e-simple.sh  # Quick validation

# 5. Run full E2E tests
./hack/run-e2e.sh

# 6. Debug if needed
SKIP_CLEANUP=true VERBOSE=true ./hack/run-e2e.sh
kubectl get all -n nephoran-system
kubectl logs -n nephoran-system deployment/webhook-manager

# 7. Clean and retest
kind delete cluster --name nephoran-e2e
./hack/run-e2e.sh

# 8. Commit and push
git add .
git commit -m "feat: add new feature"
git push origin feature/my-feature

# 9. Create PR
# Use GitHub CLI or web interface
```

### Debugging Workflow

```bash
# 1. Enable maximum verbosity
export VERBOSE=true
export DEBUG=1
export LOG_LEVEL=debug

# 2. Run with extended timeout and skip cleanup
TIMEOUT=900 SKIP_CLEANUP=true ./hack/run-e2e.sh 2>&1 | tee debug.log

# 3. Inspect cluster state
kubectl get all -A
kubectl get events -A --sort-by='.lastTimestamp' | tail -20
kubectl describe pods -n nephoran-system

# 4. Check component logs
kubectl logs -n nephoran-system deployment/webhook-manager --tail=100
kubectl logs -n nephoran-system -l app=intent-ingest --tail=100

# 5. Test individual components
# Test webhook
kubectl apply -f tests/e2e/samples/basic-scale-intent.yaml --dry-run=server

# Test intent processing
curl -X POST http://localhost:8080/intent -H "Content-Type: application/json" \
  -d '{"intent_type":"scaling","target":"test","replicas":3}'

# 6. Clean up after debugging
kind delete cluster --name nephoran-e2e
rm -rf handoff/* test-reports/*
```

### Performance Testing Workflow

```bash
# 1. Baseline test
time ./hack/run-e2e.sh

# 2. Parallel execution test
PARALLEL_TESTS=8 time ./hack/run-production-e2e.sh

# 3. Stress test
for i in {1..10}; do
  ./hack/run-e2e-simple.sh &
done
wait

# 4. Monitor resources
# In another terminal
watch -n 2 'docker stats --no-stream'
```

## IDE Integration

### Visual Studio Code

#### Recommended Extensions

```powershell
# Install VS Code extensions via command line
code --install-extension golang.go
code --install-extension ms-kubernetes-tools.vscode-kubernetes-tools
code --install-extension ms-azuretools.vscode-docker
code --install-extension redhat.vscode-yaml
code --install-extension hashicorp.terraform
code --install-extension streetsidesoftware.code-spell-checker
code --install-extension eamodio.gitlens
code --install-extension ms-vscode.powershell
code --install-extension timonwong.shellcheck
```

#### VS Code Settings

Create `.vscode/settings.json`:

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "file",
  "go.formatTool": "goimports",
  "go.testFlags": ["-v", "-race"],
  "go.testTimeout": "10m",
  
  "files.eol": "\n",
  "files.trimTrailingWhitespace": true,
  "files.insertFinalNewline": true,
  
  "terminal.integrated.defaultProfile.windows": "Git Bash",
  "terminal.integrated.profiles.windows": {
    "Git Bash": {
      "path": "C:\\Program Files\\Git\\bin\\bash.exe",
      "icon": "terminal-bash"
    },
    "PowerShell": {
      "source": "PowerShell",
      "icon": "terminal-powershell"
    }
  },
  
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  },
  
  "[yaml]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "redhat.vscode-yaml"
  },
  
  "[json]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "vscode.json-language-features"
  },
  
  "[shellscript]": {
    "editor.defaultFormatter": "foxundermoon.shell-format"
  },
  
  "kubernetes.kubectl.path": "C:\\ProgramData\\chocolatey\\bin\\kubectl.exe",
  "docker.dockerPath": "C:\\Program Files\\Docker\\Docker\\resources\\bin\\docker.exe"
}
```

#### Launch Configurations

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Run E2E Tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/tests/e2e",
      "args": [
        "-test.v",
        "-test.timeout=30m",
        "-ginkgo.v"
      ],
      "env": {
        "KUBECONFIG": "${workspaceFolder}/.kube/config",
        "GO_ENV": "test"
      }
    },
    {
      "name": "Debug Intent Ingest",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/intent-ingest",
      "env": {
        "LOG_LEVEL": "debug",
        "PORT": "8080"
      }
    },
    {
      "name": "Debug Webhook Manager",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/webhook-manager",
      "env": {
        "LOG_LEVEL": "debug"
      }
    }
  ]
}
```

#### Task Runner

Create `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Run E2E Tests",
      "type": "shell",
      "command": "./hack/run-e2e.sh",
      "windows": {
        "command": ".\\hack\\run-e2e.ps1"
      },
      "group": {
        "kind": "test",
        "isDefault": true
      },
      "presentation": {
        "reveal": "always",
        "panel": "new"
      }
    },
    {
      "label": "Build All",
      "type": "shell",
      "command": "make build",
      "group": {
        "kind": "build",
        "isDefault": true
      }
    },
    {
      "label": "Create Kind Cluster",
      "type": "shell",
      "command": "kind create cluster --name nephoran-e2e",
      "presentation": {
        "reveal": "always"
      }
    },
    {
      "label": "Delete Kind Cluster",
      "type": "shell",
      "command": "kind delete cluster --name nephoran-e2e",
      "presentation": {
        "reveal": "always"
      }
    },
    {
      "label": "Check Tools",
      "type": "shell",
      "command": "./hack/check-tools.sh",
      "windows": {
        "command": "bash ./hack/check-tools.sh"
      },
      "presentation": {
        "reveal": "always"
      }
    }
  ]
}
```

### JetBrains GoLand

#### Run Configurations

1. **E2E Test Configuration**:
   - Type: Go Test
   - Package: `github.com/thc1006/nephoran-intent-operator/tests/e2e`
   - Working directory: Project root
   - Environment: `KUBECONFIG=$PROJECT_DIR$/.kube/config;GO_ENV=test`
   - Program arguments: `-v -timeout=30m`

2. **Shell Script Configuration**:
   - Type: Shell Script
   - Script path: `$PROJECT_DIR$/hack/run-e2e.sh`
   - Interpreter: Git Bash (`C:\Program Files\Git\bin\bash.exe`)
   - Working directory: Project root

#### External Tools

Configure external tools for quick access:

1. **Run E2E Tests**:
   - Program: `C:\Program Files\Git\bin\bash.exe`
   - Arguments: `hack/run-e2e.sh`
   - Working directory: `$ProjectFileDir$`

2. **Check Tools**:
   - Program: `C:\Program Files\Git\bin\bash.exe`
   - Arguments: `hack/check-tools.sh`
   - Working directory: `$ProjectFileDir$`

### Windows Terminal Configuration

Create a profile for the project in Windows Terminal settings:

```json
{
  "profiles": {
    "list": [
      {
        "guid": "{your-guid-here}",
        "name": "Nephoran E2E",
        "commandline": "C:\\Program Files\\Git\\bin\\bash.exe",
        "startingDirectory": "C:\\dev\\nephoran-intent-operator",
        "icon": "🚀",
        "colorScheme": "One Half Dark",
        "fontFace": "Cascadia Code",
        "fontSize": 11,
        "acrylicOpacity": 0.9,
        "useAcrylic": true,
        "environment": {
          "SKIP_CLEANUP": "true",
          "VERBOSE": "true"
        }
      }
    ]
  }
}
```

## Tips and Best Practices

### Windows-Specific Best Practices

1. **Use Git Bash for scripts**: Bash scripts work better in Git Bash than PowerShell
2. **Keep paths short**: Windows has path length limitations
3. **Handle line endings**: Always configure Git for proper line ending handling
4. **Run as Administrator**: When installing tools or dealing with permissions
5. **Use WSL2 for complex scenarios**: Consider WSL2 for better Linux compatibility
6. **Regular Docker cleanup**: Windows Docker can consume significant disk space
7. **Antivirus exclusions**: Add development folders to antivirus exclusions
8. **SSD usage**: Use SSD for better Docker and build performance

### Productivity Tips

1. **Use aliases**: Set up aliases for common commands
2. **Terminal multiplexer**: Use Windows Terminal tabs or tmux in Git Bash
3. **Script everything**: Create scripts for repetitive tasks
4. **Version control settings**: Commit your IDE settings and configurations
5. **Documentation**: Keep notes on Windows-specific issues and solutions
6. **Backup**: Regular backup of your development environment
7. **Virtualization**: Consider using VMs for isolated testing environments

### Common Gotchas

1. **Case sensitivity**: Windows is case-insensitive, but Git and Linux are not
2. **Path separators**: Use forward slashes (/) in Git Bash, backslashes (\) in PowerShell
3. **File permissions**: Windows doesn't have Unix-style permissions
4. **Symbolic links**: Require admin privileges on Windows
5. **Process management**: Different process model than Unix
6. **Network interfaces**: Different network interface names
7. **Environment variables**: Different syntax between PowerShell and Bash

## Quick Reference

### Essential Commands

```bash
# E2E Testing
./hack/run-e2e.sh                    # Run basic E2E tests
./hack/run-production-e2e.sh         # Run production scenarios
./hack/validate-e2e.sh               # CI validation
./hack/check-tools.sh                # Check tool availability

# Cluster Management
kind create cluster --name test      # Create cluster
kind delete cluster --name test      # Delete cluster
kind get clusters                    # List clusters
kubectl cluster-info                 # Cluster information

# Debugging
kubectl logs -f <pod>                # Follow logs
kubectl describe <resource>          # Detailed info
kubectl get events --sort-by='.lastTimestamp'  # Recent events
kubectl exec -it <pod> -- bash       # Shell into pod

# Git Operations
git status                           # Check status
git checkout -b feature/name         # Create branch
git add .                            # Stage changes
git commit -m "message"              # Commit
git push origin branch               # Push changes

# Docker Commands
docker ps                            # List containers
docker images                        # List images
docker system prune -a               # Clean up
docker stats                         # Resource usage
```

### Environment Variables

```bash
# Test Control
CLUSTER_NAME=name                    # Cluster name
NAMESPACE=ns                         # Test namespace
SKIP_CLEANUP=true                    # Keep resources
VERBOSE=true                         # Verbose output
TIMEOUT=600                          # Timeout in seconds

# Component Modes
INTENT_INGEST_MODE=local            # local or sidecar
CONDUCTOR_MODE=local                # local or in-cluster
PORCH_MODE=structured-patch         # structured-patch or direct

# Test Configuration
PARALLEL_TESTS=4                    # Parallel execution
TEST_SCENARIO=all                   # Test scenario
TEST_TIMEOUT=30m                    # Test timeout
REPORT_DIR=./test-reports          # Report directory
```

### File Locations

```
C:\Users\%USERNAME%\
├── .bashrc                         # Bash configuration
├── .gitconfig                      # Git configuration
├── go\                            # Go workspace
│   ├── bin\                       # Go binaries
│   └── pkg\                       # Go packages
└── dev\                           # Development projects
    └── nephoran-intent-operator\  # Project root
        ├── hack\                  # Scripts
        ├── tests\e2e\            # E2E tests
        ├── handoff\              # Generated artifacts
        └── test-reports\         # Test results
```

## Conclusion

This guide provides comprehensive coverage of Windows-specific development and testing workflows for the Nephoran E2E testing framework. By following these guidelines and best practices, Windows developers can effectively contribute to and test the Nephoran Intent Operator project with the same efficiency as Unix-based developers.

Remember that while Windows presents some unique challenges, modern tools like Git Bash, WSL2, and Docker Desktop have made cross-platform development much more seamless. The key is proper configuration and understanding of the Windows-specific nuances covered in this guide.

For additional support or Windows-specific issues not covered here, please consult the project's issue tracker or reach out to the development team.

---

*Document Version: 1.0.0*  
*Last Updated: 2025-08-16*  
*Platform: Windows 10/11 with Git Bash*
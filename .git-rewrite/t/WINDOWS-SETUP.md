# Windows Development Setup Guide
## Nephoran Intent Operator on Windows 11

This guide provides comprehensive instructions for setting up the Nephoran Intent Operator development environment on Windows 11, addressing common path and environment issues.

## Quick Start

### 1. Automated Setup (Recommended)

Run the automated setup script as Administrator:

```powershell
# Run PowerShell as Administrator
.\setup-windows.ps1
```

This script will:
- Install all required development tools via Chocolatey
- Setup Python virtual environment (avoiding pyenv-win issues)
- Configure local Kubernetes cluster with kind
- Create Windows-compatible environment scripts

### 2. Manual Verification

After automated setup, verify your environment:

```powershell
# Check tool versions
go version
python --version
docker --version
kubectl version --client
kind version

# Verify cluster
kubectl cluster-info --context kind-nephoran-dev
```

## Issue-Specific Solutions

### Issue 1: Build System Path Problems

**Problem**: `/c/Users/.../C:/MinGW/bin/make.exe: No such file or directory`

**Solution**: Updated Makefile with Windows path handling

The Makefile now includes Windows-specific logic:
- Cross-platform path separators (`\` vs `/`)
- Windows executable extensions (`.exe`)
- Proper environment variable handling

**Usage**:
```cmd
# Use the updated Makefile
make setup-dev
make build-all
```

### Issue 2: Python Environment Issues

**Problem**: `cygpath: command not found` in pyenv

**Solutions Provided**:

1. **Virtual Environment (Recommended)**:
   ```powershell
   # Create and activate virtual environment
   python -m venv venv
   .\venv\Scripts\Activate.ps1
   
   # Install dependencies
   python -m pip install -r requirements-rag.txt
   ```

2. **Windows Environment Script**:
   ```cmd
   # Use the generated environment script
   .\windows-env.bat
   ```

3. **Alternative to pyenv-win**:
   - Use Python from Microsoft Store or official installer
   - Avoid pyenv-win complexity with virtual environments

### Issue 3: Kubernetes Cluster Setup

**Problem**: `dial tcp 127.0.0.1:52458: connectex: No connection could be made`

**Solutions**:

1. **Automated Cluster Setup**:
   ```powershell
   # Deploy with automatic cluster creation
   .\deploy-windows.ps1 -Environment local
   ```

2. **Manual kind Setup**:
   ```powershell
   # Create kind cluster manually
   kind create cluster --name nephoran-dev
   kubectl cluster-info --context kind-nephoran-dev
   ```

3. **Alternative: Docker Desktop Kubernetes**:
   ```powershell
   # Enable Kubernetes in Docker Desktop settings
   # Then verify:
   kubectl cluster-info --context docker-desktop
   ```

## Development Workflow

### Building the Project

```powershell
# Setup development environment
make setup-dev

# Build all components (Windows-compatible)
make build-all

# Build specific components
make build-llm-processor    # Creates bin\llm-processor.exe
make build-nephio-bridge    # Creates bin\nephio-bridge.exe
make build-oran-adaptor     # Creates bin\oran-adaptor.exe
```

### Docker Operations

```powershell
# Build Docker images
make docker-build

# Deploy locally with image loading
.\deploy-windows.ps1 -Environment local

# Deploy to remote registry
.\deploy-windows.ps1 -Environment remote
```

### Testing

```powershell
# Run integration tests (Windows-compatible)
make test-integration

# Run linting
make lint

# Manual testing
kubectl apply -f my-first-intent.yaml
kubectl get networkintents
kubectl describe networkintent <name>
```

## Windows-Specific Configuration

### Environment Variables

Set these in PowerShell or through System Properties:

```powershell
# Required for LLM processing
$env:OPENAI_API_KEY = "your-openai-api-key"

# Optional: Custom configuration
$env:LLM_PROCESSOR_URL = "http://llm-processor:8080"
$env:WEAVIATE_URL = "http://weaviate:8080"
$env:RAG_API_URL = "http://rag-api:5001"
```

### Path Configuration

The setup scripts handle path configuration automatically, but manual setup:

```powershell
# Add Go tools to PATH
$goPath = go env GOPATH
$env:PATH += ";$goPath\bin"

# Verify PATH
echo $env:PATH | Select-String "go"
```

### Docker Desktop Configuration

Ensure Docker Desktop is configured for:
- WSL 2 backend (recommended)
- Kubernetes enabled (if not using kind)
- Resource allocation: 4GB+ RAM, 2+ CPUs

## Troubleshooting

### Common Windows Issues

1. **PowerShell Execution Policy**:
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

2. **Long Path Support**:
   ```powershell
   # Enable long paths in Windows
   New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
   ```

3. **Windows Defender Exclusions**:
   Add to Windows Defender exclusions:
   - Go workspace directory
   - Docker Desktop
   - WSL 2 file system

### Build Issues

1. **Go Module Issues**:
   ```powershell
   go clean -modcache
   go mod download
   go mod tidy
   ```

2. **Docker Build Issues**:
   ```powershell
   docker system prune -f
   docker builder prune -f
   ```

3. **Kubernetes Issues**:
   ```powershell
   # Reset kind cluster
   kind delete cluster --name nephoran-dev
   .\setup-windows.ps1 -Force
   
   # Reset Docker Desktop Kubernetes
   # Docker Desktop > Settings > Kubernetes > Reset Kubernetes Cluster
   ```

### Debugging Commands

```powershell
# Check cluster connectivity
kubectl cluster-info
kubectl get nodes

# Check pod status
kubectl get pods -A
kubectl logs -f deployment/nephio-bridge

# Check CRD registration
kubectl get crd | Select-String "nephoran"
kubectl api-resources | Select-String "nephoran"

# Test RAG API
kubectl port-forward svc/rag-api 5001:5001
# In another terminal:
curl http://localhost:5001/healthz
```

## Integration with Windows Tools

### Visual Studio Code

Recommended extensions:
- Go extension
- Kubernetes extension
- Docker extension
- PowerShell extension

### Windows Terminal

Recommended configuration for multiple environments:
```json
{
    "name": "Development",
    "commandline": "powershell.exe -NoExit -Command \"& { . .\\venv\\Scripts\\Activate.ps1; . .\\windows-env.bat }\"",
    "startingDirectory": "C:\\Users\\thc1006\\Desktop\\nephoran-intent-operator\\nephoran-intent-operator"
}
```

### Git Bash Alternative

If preferring Git Bash over PowerShell:
```bash
# Use Unix-style paths in Git Bash
export GOPATH=$(go env GOPATH | tr '\\' '/')
export PATH="$PATH:$GOPATH/bin"

# Convert Windows paths when needed
winpath() {
    echo "$1" | sed 's|/c/|C:/|g' | tr '/' '\\'
}
```

## Performance Optimization

### Windows-Specific Optimizations

1. **WSL 2 Integration**:
   - Move project to WSL 2 file system for better performance
   - Use Docker Desktop WSL 2 backend

2. **Exclusions**:
   - Add project directory to Windows Defender exclusions
   - Exclude Go module cache and Docker directories

3. **Resource Allocation**:
   - Increase Docker Desktop memory limit (4GB+)
   - Allocate sufficient CPU cores for kind cluster

## Next Steps

After successful setup:

1. **Verify End-to-End Workflow**:
   ```powershell
   # Deploy and test
   .\deploy-windows.ps1 -Environment local
   kubectl apply -f my-first-intent.yaml
   kubectl get networkintents
   ```

2. **Populate Knowledge Base**:
   ```powershell
   # Setup vector database
   make populate-kb
   ```

3. **Development Iteration**:
   ```powershell
   # Make changes, then:
   make lint
   make test-integration
   make docker-build
   .\deploy-windows.ps1 -Environment local
   ```

This Windows setup guide ensures you can develop and deploy the Nephoran Intent Operator effectively on Windows 11, with solutions for all major platform-specific issues.
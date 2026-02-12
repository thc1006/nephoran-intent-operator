# Complete Local Kubernetes Development Environment Setup
## Nephoran Intent Operator on Windows 11

This guide provides step-by-step instructions for setting up a complete local Kubernetes development environment for the Nephoran Intent Operator project on Windows 11.

## 🎯 Quick Start (Automated Setup)

For immediate setup, run these commands in PowerShell as Administrator:

```powershell
# 1. Automated environment setup
.\local-k8s-setup.ps1 -WithRegistry

# 2. Build and deploy
make setup-dev
make build-all
.\local-deploy.ps1 -UseLocalRegistry

# 3. Validate everything works
.\validate-environment.ps1
.\test-crds.ps1 -CleanupAfter
```

## 📋 Prerequisites

- **Windows 11** (tested and optimized)
- **PowerShell 5.1+** or **PowerShell Core 7+**
- **Administrator privileges** (for initial setup)
- **Internet connection** (for downloading tools and images)

## 🛠️ Step-by-Step Setup

### Step 1: Environment Preparation

1. **Run PowerShell as Administrator**
   ```powershell
   # Right-click PowerShell -> "Run as Administrator"
   ```

2. **Set Execution Policy** (if needed)
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

3. **Navigate to Project Directory**
   ```powershell
   cd C:\Users\thc1006\Desktop\nephoran-intent-operator\nephoran-intent-operator
   ```

### Step 2: Tool Installation and Cluster Setup

1. **Run Automated Setup**
   ```powershell
   # Full setup with local registry
   .\local-k8s-setup.ps1 -WithRegistry -ClusterType kind
   
   # Alternative: Minikube setup
   .\local-k8s-setup.ps1 -WithRegistry -ClusterType minikube
   
   # Alternative: Docker Desktop (manual Kubernetes enable required)
   .\local-k8s-setup.ps1 -WithRegistry -ClusterType docker-desktop
   ```

2. **Verify Cluster Setup**
   ```powershell
   kubectl cluster-info
   kubectl get nodes
   kubectl get crd | Select-String "nephoran"
   ```

### Step 3: Development Environment Setup

1. **Setup Go and Python Dependencies**
   ```powershell
   make setup-dev
   ```

2. **Set Required Environment Variables**
   ```powershell
   # Required for LLM processing
   $env:OPENAI_API_KEY = "your-openai-api-key"
   
   # Optional: Set in permanent environment
   [Environment]::SetEnvironmentVariable("OPENAI_API_KEY", "your-key", "User")
   ```

3. **Activate Python Virtual Environment** (if using)
   ```powershell
   .\venv\Scripts\Activate.ps1
   ```

### Step 4: Build and Deploy Application

1. **Build All Components**
   ```powershell
   # Build Go binaries
   make build-all
   
   # Build Docker images
   make docker-build
   ```

2. **Deploy to Local Cluster**
   ```powershell
   # With local registry
   .\local-deploy.ps1 -UseLocalRegistry
   
   # Without local registry (direct image loading)
   .\local-deploy.ps1
   
   # Force rebuild and deploy
   .\local-deploy.ps1 -Force -UseLocalRegistry
   ```

3. **Monitor Deployment**
   ```powershell
   # Watch deployment progress
   kubectl get pods -n nephoran-system -w
   
   # Check deployment status
   kubectl get deployments -n nephoran-system
   ```

### Step 5: Validation and Testing

1. **Validate Complete Environment**
   ```powershell
   .\validate-environment.ps1 -Detailed
   ```

2. **Test CRD Functionality**
   ```powershell
   # Basic CRD testing
   .\test-crds.ps1
   
   # Extended testing with controller validation
   .\test-crds.ps1 -WaitForControllers -Verbose -CleanupAfter
   ```

3. **Test Sample Network Intent**
   ```powershell
   # Apply sample intent
   kubectl apply -f archive/my-first-intent.yaml
   
   # Check processing
   kubectl get networkintents -n nephoran-system
   kubectl describe networkintent <intent-name> -n nephoran-system
   ```

## 🔧 Development Workflow

### Daily Development Cycle

1. **Make Code Changes**
   - Edit Go controllers, Python services, or configuration files

2. **Quick Rebuild and Deploy**
   ```powershell
   # Fast iteration (skip tests for speed)
   make build-all
   .\local-deploy.ps1 -Force
   ```

3. **Test Changes**
   ```powershell
   # Apply test resources
   kubectl apply -f test-resources/
   
   # Watch logs
   kubectl logs -f deployment/nephio-bridge -n nephoran-system
   ```

4. **Debug Issues**
   ```powershell
   # Check pod status
   kubectl get pods -n nephoran-system
   
   # Get detailed logs
   kubectl logs deployment/llm-processor -n nephoran-system --previous
   
   # Port forward for direct testing
   kubectl port-forward svc/rag-api 5001:5001 -n nephoran-system
   ```

### Advanced Development Tasks

1. **Update CRDs**
   ```powershell
   # After modifying CRD definitions
   make generate
   kubectl apply -f deployments/crds/
   ```

2. **Reset Environment**
   ```powershell
   # Delete and recreate cluster
   kind delete cluster --name nephoran-dev
   .\local-k8s-setup.ps1 -WithRegistry -Force
   ```

3. **Test with Local Registry**
   ```powershell
   # Push to local registry
   .\local-deploy.ps1 -UseLocalRegistry
   
   # Verify registry contents
   curl http://localhost:5001/v2/_catalog
   ```

## 🐛 Troubleshooting

### Common Issues and Solutions

1. **Docker Not Running**
   ```
   Error: Docker daemon not responding
   Solution: Start Docker Desktop and wait for it to fully initialize
   ```

2. **Cluster Not Accessible**
   ```
   Error: Cannot connect to Kubernetes cluster
   Solution: Run .\local-k8s-setup.ps1 to recreate cluster
   ```

3. **CRDs Not Established**
   ```
   Error: CRD exists but not established
   Solution: kubectl delete crd <crd-name> && kubectl apply -f deployments/crds/
   ```

4. **Images Not Found**
   ```
   Error: Failed to pull image
   Solution: Check image names and run make docker-build
   ```

5. **Pods Stuck in Pending**
   ```
   Error: Pods not scheduling
   Solution: Check node resources with kubectl describe nodes
   ```

### Debugging Commands

```powershell
# Comprehensive system check
.\validate-environment.ps1 -Detailed

# Check cluster resources
kubectl top nodes
kubectl top pods -n nephoran-system

# Examine failed pods
kubectl describe pod <pod-name> -n nephoran-system

# Check events
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'

# Network debugging
kubectl run debug --image=busybox:1.28 --rm -it --restart=Never -- /bin/sh
```

## 📊 Environment Validation Checklist

Use this checklist to verify your environment is properly configured:

### ✅ Tools Installation
- [ ] Docker Desktop running
- [ ] kubectl installed and configured
- [ ] kind/minikube installed
- [ ] Go 1.20+ installed
- [ ] Python 3.8+ with pip

### ✅ Kubernetes Cluster
- [ ] Cluster accessible (`kubectl cluster-info`)
- [ ] Nodes ready (`kubectl get nodes`)
- [ ] Namespace created (`kubectl get ns nephoran-system`)
- [ ] CRDs deployed and established
- [ ] Local registry running (if enabled)

### ✅ Application Deployment
- [ ] All deployments ready
- [ ] All pods running
- [ ] Services accessible
- [ ] Container images present

### ✅ Functionality Testing
- [ ] CRD creation works
- [ ] Controller processes intents
- [ ] Logs show no critical errors
- [ ] Sample resources can be created

## 🔄 Maintenance and Updates

### Regular Maintenance Tasks

1. **Update Dependencies**
   ```powershell
   # Update Go modules
   go mod tidy
   go mod download
   
   # Update Python packages
   pip install -r requirements-rag.txt --upgrade
   ```

2. **Clean Up Resources**
   ```powershell
   # Remove old images
   docker image prune -f
   
   # Clean up test resources
   kubectl delete networkintents --all -n nephoran-system
   ```

3. **Backup Configuration**
   ```powershell
   # Export current kubeconfig
   kubectl config view --raw > backup-kubeconfig.yaml
   
   # Save current environment variables
   Get-ChildItem env: | Out-File environment-backup.txt
   ```

### Updating the Application

1. **Pull Latest Changes**
   ```powershell
   git pull origin main
   ```

2. **Rebuild and Redeploy**
   ```powershell
   make setup-dev
   make build-all
   .\local-deploy.ps1 -Force -UseLocalRegistry
   ```

3. **Validate Updates**
   ```powershell
   .\validate-environment.ps1
   .\test-crds.ps1 -CleanupAfter
   ```

## 🎯 Performance Optimization

### Windows-Specific Optimizations

1. **WSL 2 Integration**
   - Enable WSL 2 backend in Docker Desktop
   - Move project to WSL 2 filesystem for better performance

2. **Resource Allocation**
   - Increase Docker Desktop memory to 8GB+
   - Allocate 4+ CPU cores to Docker

3. **Windows Defender Exclusions**
   - Add project directory to exclusions
   - Add Docker Desktop to exclusions
   - Add Go workspace to exclusions

### Cluster Optimization

1. **Kind Configuration**
   ```powershell
   # Use more powerful kind cluster
   .\local-k8s-setup.ps1 -WithRegistry -Force
   # Creates 3-node cluster with proper resource allocation
   ```

2. **Registry Performance**
   - Use local registry for faster image pulls
   - Pre-pull base images when possible

## 📚 Additional Resources

### Documentation
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Docker Desktop Documentation](https://docs.docker.com/desktop/)

### Project-Specific
- `CLAUDE.md` - Complete project documentation
- `WINDOWS-SETUP.md` - Windows-specific setup guide
- `Makefile` - Build system reference

### Scripts Reference
- `local-k8s-setup.ps1` - Cluster setup and configuration
- `local-deploy.ps1` - Application deployment
- `validate-environment.ps1` - Environment validation
- `test-crds.ps1` - CRD and controller testing

This complete setup guide ensures you have a fully functional local Kubernetes development environment for the Nephoran Intent Operator on Windows 11.
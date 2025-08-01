# Complete Local Kubernetes Development Environment Setup
# Nephoran Intent Operator - Windows 11 Compatible
# This script sets up a complete development environment with local registry

param(
    [ValidateSet("kind", "minikube", "docker-desktop")]
    [string]$ClusterType = "kind",
    
    [switch]$WithRegistry,
    [switch]$Force,
    [switch]$SkipValidation,
    [switch]$Help
)

if ($Help) {
    Write-Host @"
Nephoran Intent Operator - Local Kubernetes Setup

DESCRIPTION:
  Sets up a complete local Kubernetes development environment on Windows 11
  with support for CRD deployment, testing, and local container registry.

USAGE:
  .\local-k8s-setup.ps1 [options]

OPTIONS:
  -ClusterType      Kubernetes cluster type (kind, minikube, docker-desktop)
  -WithRegistry     Setup local container registry for development
  -Force            Force recreation of existing cluster
  -SkipValidation   Skip validation steps
  -Help             Show this help message

EXAMPLES:
  .\local-k8s-setup.ps1                          # Default kind cluster
  .\local-k8s-setup.ps1 -ClusterType minikube    # Use minikube
  .\local-k8s-setup.ps1 -WithRegistry -Force     # With local registry, force recreate

REQUIREMENTS:
  - Docker Desktop for Windows
  - kubectl CLI
  - kind or minikube (installed automatically if missing)

"@
    exit 0
}

# Configuration
$CLUSTER_NAME = "nephoran-dev"
$REGISTRY_NAME = "nephoran-registry"
$REGISTRY_PORT = "5001"
$KUBECTL_VERSION = "v1.29.0"

# Color functions for better output
function Write-Step($Message) { Write-Host "🔧 $Message" -ForegroundColor Cyan }
function Write-Success($Message) { Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Warning($Message) { Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error($Message) { Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Info($Message) { Write-Host "ℹ️  $Message" -ForegroundColor Blue }

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Install-RequiredTools {
    Write-Step "Checking and installing required tools..."
    
    # Check if Chocolatey is installed
    if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
        Write-Warning "Chocolatey not found. Installing..."
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    }
    
    # Install kubectl if not present
    if (!(Get-Command kubectl -ErrorAction SilentlyContinue)) {
        Write-Step "Installing kubectl..."
        choco install kubernetes-cli -y
    }
    
    # Install cluster-specific tools
    switch ($ClusterType) {
        "kind" {
            if (!(Get-Command kind -ErrorAction SilentlyContinue)) {
                Write-Step "Installing kind..."
                choco install kind -y
            }
        }
        "minikube" {
            if (!(Get-Command minikube -ErrorAction SilentlyContinue)) {
                Write-Step "Installing minikube..."
                choco install minikube -y
            }
        }
        "docker-desktop" {
            if (!(Get-Command docker -ErrorAction SilentlyContinue)) {
                Write-Error "Docker Desktop is required but not found. Please install Docker Desktop for Windows."
                exit 1
            }
        }
    }
    
    # Install kustomize if not present
    if (!(Get-Command kustomize -ErrorAction SilentlyContinue)) {
        Write-Step "Installing kustomize..."
        choco install kustomize -y
    }
    
    # Refresh PATH
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    
    Write-Success "All required tools installed"
}

function Test-DockerRunning {
    Write-Step "Checking Docker status..."
    try {
        docker info 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Docker is running"
            return $true
        }
    } catch {}
    
    Write-Error "Docker is not running. Please start Docker Desktop and try again."
    Write-Info "To start Docker Desktop:"
    Write-Info "1. Open Docker Desktop from Start Menu"
    Write-Info "2. Wait for Docker to start (whale icon in system tray)"
    Write-Info "3. Re-run this script"
    return $false
}

function Setup-LocalRegistry {
    if (!$WithRegistry) {
        Write-Info "Skipping local registry setup (use -WithRegistry to enable)"
        return
    }
    
    Write-Step "Setting up local container registry..."
    
    # Check if registry is already running
    $existingRegistry = docker ps --filter "name=$REGISTRY_NAME" --format "{{.Names}}" 2>$null
    if ($existingRegistry -eq $REGISTRY_NAME) {
        if ($Force) {
            Write-Step "Removing existing registry..."
            docker stop $REGISTRY_NAME 2>$null | Out-Null
            docker rm $REGISTRY_NAME 2>$null | Out-Null
        } else {
            Write-Success "Local registry is already running on port $REGISTRY_PORT"
            return
        }
    }
    
    # Start local registry
    Write-Step "Starting local container registry..."
    docker run -d --restart=always -p "${REGISTRY_PORT}:5000" --name $REGISTRY_NAME registry:2
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Local registry started on localhost:$REGISTRY_PORT"
        
        # Test registry
        Start-Sleep -Seconds 3
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:$REGISTRY_PORT/v2/" -UseBasicParsing
            if ($response.StatusCode -eq 200) {
                Write-Success "Registry is responding correctly"
            }
        } catch {
            Write-Warning "Registry may still be starting up"
        }
    } else {
        Write-Error "Failed to start local registry"
        exit 1
    }
}

function Setup-KindCluster {
    Write-Step "Setting up Kind cluster..."
    
    # Check if cluster exists
    $existingClusters = kind get clusters 2>$null
    if ($existingClusters -contains $CLUSTER_NAME) {
        if ($Force) {
            Write-Step "Deleting existing Kind cluster..."
            kind delete cluster --name $CLUSTER_NAME
        } else {
            Write-Success "Kind cluster '$CLUSTER_NAME' already exists"
            kind get kubeconfig --name $CLUSTER_NAME | Set-Content -Path "$env:USERPROFILE\.kube\config"
            return
        }
    }
    
    # Create Kind configuration
    $kindConfig = @"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: $CLUSTER_NAME
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
  - containerPort: 30000
    hostPort: 30000
    protocol: TCP
  - containerPort: 30001
    hostPort: 30001
    protocol: TCP
- role: worker
  labels:
    nephoran.com/node-type: "worker"
- role: worker
  labels:
    nephoran.com/node-type: "worker"
"@

    if ($WithRegistry) {
        # Add registry configuration for Kind
        $kindConfig += @"

containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:$REGISTRY_PORT"]
    endpoint = ["http://host.docker.internal:$REGISTRY_PORT"]
"@
    }
    
    $kindConfig | Out-File -FilePath "kind-config.yaml" -Encoding UTF8
    
    Write-Step "Creating Kind cluster with 3 nodes..."
    kind create cluster --config kind-config.yaml
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Kind cluster created successfully"
        Remove-Item "kind-config.yaml"
        
        # Connect registry to Kind network if registry is running
        if ($WithRegistry) {
            Write-Step "Connecting registry to Kind network..."
            docker network connect "kind" $REGISTRY_NAME 2>$null
        }
    } else {
        Write-Error "Failed to create Kind cluster"
        Remove-Item "kind-config.yaml" -ErrorAction SilentlyContinue
        exit 1
    }
}

function Setup-MinikubeCluster {
    Write-Step "Setting up Minikube cluster..."
    
    # Check if cluster exists
    $status = minikube status -p $CLUSTER_NAME 2>$null
    if ($status -like "*Running*" -and !$Force) {
        Write-Success "Minikube cluster '$CLUSTER_NAME' is already running"
        minikube update-context -p $CLUSTER_NAME
        return
    }
    
    if ($Force) {
        Write-Step "Deleting existing Minikube cluster..."
        minikube delete -p $CLUSTER_NAME 2>$null
    }
    
    Write-Step "Starting Minikube cluster..."
    minikube start -p $CLUSTER_NAME `
        --driver=hyperv `
        --cpus=4 `
        --memory=8192 `
        --disk-size=50g `
        --kubernetes-version=$KUBECTL_VERSION `
        --addons=ingress,registry,metrics-server
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Minikube cluster created successfully"
        minikube update-context -p $CLUSTER_NAME
        
        if ($WithRegistry) {
            Write-Info "Minikube has built-in registry addon enabled"
            Write-Info "Access it via: kubectl port-forward -n kube-system service/registry 5000:80"
        }
    } else {
        Write-Error "Failed to create Minikube cluster"
        exit 1
    }
}

function Setup-DockerDesktopKubernetes {
    Write-Step "Configuring Docker Desktop Kubernetes..."
    
    Write-Info "Please enable Kubernetes in Docker Desktop:"
    Write-Info "1. Open Docker Desktop"
    Write-Info "2. Go to Settings > Kubernetes"
    Write-Info "3. Check 'Enable Kubernetes'"
    Write-Info "4. Click 'Apply & Restart'"
    Write-Info "5. Wait for Kubernetes to start"
    
    # Wait for user confirmation
    Read-Host "Press Enter when Kubernetes is enabled in Docker Desktop"
    
    # Verify Kubernetes is running
    $maxAttempts = 30
    $attempt = 0
    
    do {
        $attempt++
        Write-Step "Checking Kubernetes status (attempt $attempt/$maxAttempts)..."
        
        try {
            kubectl cluster-info --context docker-desktop 2>$null | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Docker Desktop Kubernetes is running"
                kubectl config use-context docker-desktop
                return
            }
        } catch {}
        
        Start-Sleep -Seconds 5
    } while ($attempt -lt $maxAttempts)
    
    Write-Error "Docker Desktop Kubernetes is not responding after $maxAttempts attempts"
    exit 1
}

function Deploy-CRDs {
    Write-Step "Deploying Custom Resource Definitions..."
    
    $crdPath = "deployments\crds"
    if (!(Test-Path $crdPath)) {
        Write-Error "CRD directory not found: $crdPath"
        Write-Info "Please run this script from the project root directory"
        exit 1
    }
    
    # Apply CRDs
    kubectl apply -f $crdPath
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to apply CRDs"
        exit 1
    }
    
    Write-Step "Waiting for CRDs to be established..."
    Start-Sleep -Seconds 5
    
    # Verify CRDs
    $crds = @("e2nodesets.nephoran.com", "networkintents.nephoran.com", "managedelements.nephoran.com")
    foreach ($crd in $crds) {
        $status = kubectl get crd $crd -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>$null
        if ($status -eq "True") {
            Write-Success "CRD $crd is established"
        } else {
            Write-Warning "CRD $crd is not yet established"
        }
    }
}

function Setup-Namespace {
    Write-Step "Setting up development namespace..."
    
    $namespace = "nephoran-system"
    
    # Create namespace if it doesn't exist
    kubectl get namespace $namespace 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        kubectl create namespace $namespace
        Write-Success "Created namespace: $namespace"
    } else {
        Write-Success "Namespace already exists: $namespace"
    }
    
    # Set as default namespace for convenience
    kubectl config set-context --current --namespace=$namespace
    Write-Info "Set default namespace to: $namespace"
}

function Create-DevelopmentSecrets {
    Write-Step "Creating development secrets..."
    
    # Create OpenAI API key secret if not exists
    kubectl get secret openai-api-key 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Info "Creating placeholder OpenAI API key secret"
        Write-Warning "Please update this secret with your actual OpenAI API key:"
        Write-Info "kubectl create secret generic openai-api-key --from-literal=api-key=your-openai-api-key"
        
        kubectl create secret generic openai-api-key --from-literal=api-key=placeholder-key
        Write-Success "Created placeholder OpenAI API key secret"
    }
    
    # Create registry secret if using local registry
    if ($WithRegistry) {
        kubectl get secret regcred 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Step "Creating registry credentials secret..."
            kubectl create secret docker-registry regcred `
                --docker-server="localhost:$REGISTRY_PORT" `
                --docker-username=admin `
                --docker-password=admin `
                --docker-email=admin@nephoran.dev
            Write-Success "Created registry credentials secret"
        }
    }
}

function Validate-Setup {
    if ($SkipValidation) {
        Write-Info "Skipping validation steps"
        return
    }
    
    Write-Step "Validating cluster setup..."
    
    # Test cluster connectivity
    try {
        $clusterInfo = kubectl cluster-info 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Cluster connectivity: OK"
        } else {
            Write-Error "Cluster connectivity: FAILED"
            return
        }
    } catch {
        Write-Error "Failed to get cluster info"
        return
    }
    
    # Check nodes
    $nodes = kubectl get nodes --no-headers 2>$null | Measure-Object | Select-Object -ExpandProperty Count
    Write-Success "Nodes available: $nodes"
    
    # Check CRDs
    $establishedCrds = kubectl get crd -l app.kubernetes.io/part-of=nephoran-intent-operator --no-headers 2>$null | Measure-Object | Select-Object -ExpandProperty Count
    Write-Success "CRDs established: $establishedCrds"
    
    # Check namespace
    $currentNamespace = kubectl config view --minify --output 'jsonpath={.contexts[0].context.namespace}' 2>$null
    Write-Success "Current namespace: $currentNamespace"
    
    # Test registry if enabled
    if ($WithRegistry) {
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:$REGISTRY_PORT/v2/" -UseBasicParsing
            if ($response.StatusCode -eq 200) {
                Write-Success "Local registry: OK"
            }
        } catch {
            Write-Warning "Local registry: Not responding"
        }
    }
    
    # Test sample CRD creation
    Write-Step "Testing CRD functionality..."
    $testIntent = @"
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: test-intent
  namespace: $((kubectl config view --minify --output 'jsonpath={.contexts[0].context.namespace}' 2>$null))
spec:
  intent: "Test intent for validation"
  parameters: {}
"@
    
    $testIntent | kubectl apply -f - 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "CRD creation: OK"
        kubectl delete networkintent test-intent 2>$null | Out-Null
    } else {
        Write-Warning "CRD creation: FAILED"
    }
}

function Show-Summary {
    Write-Host "`n" -NoNewline
    Write-Host "🎉 " -NoNewline -ForegroundColor Green
    Write-Host "Local Kubernetes Development Environment Setup Complete!" -ForegroundColor Green
    Write-Host "================================================================" -ForegroundColor Blue
    
    $context = kubectl config current-context 2>$null
    $namespace = kubectl config view --minify --output 'jsonpath={.contexts[0].context.namespace}' 2>$null
    
    Write-Host "`nCluster Information:" -ForegroundColor Cyan
    Write-Host "  Type: $ClusterType" -ForegroundColor White
    Write-Host "  Name: $CLUSTER_NAME" -ForegroundColor White
    Write-Host "  Context: $context" -ForegroundColor White
    Write-Host "  Namespace: $namespace" -ForegroundColor White
    
    if ($WithRegistry) {
        Write-Host "`nLocal Registry:" -ForegroundColor Cyan
        Write-Host "  URL: localhost:$REGISTRY_PORT" -ForegroundColor White
        Write-Host "  Status: Running" -ForegroundColor Green
    }
    
    Write-Host "`nNext Steps:" -ForegroundColor Cyan
    Write-Host "  1. Set your OpenAI API key:" -ForegroundColor White
    Write-Host "     kubectl create secret generic openai-api-key --from-literal=api-key=your-actual-key --dry-run=client -o yaml | kubectl apply -f -" -ForegroundColor Gray
    
    Write-Host "  2. Build and deploy the application:" -ForegroundColor White
    Write-Host "     make setup-dev" -ForegroundColor Gray
    Write-Host "     make build-all" -ForegroundColor Gray
    
    if ($WithRegistry) {
        Write-Host "     .\local-deploy.ps1 -UseLocalRegistry" -ForegroundColor Gray
    } else {
        Write-Host "     .\deploy-cross-platform.ps1 local" -ForegroundColor Gray
    }
    
    Write-Host "  3. Test the deployment:" -ForegroundColor White
    Write-Host "     kubectl get pods" -ForegroundColor Gray
    Write-Host "     kubectl apply -f my-first-intent.yaml" -ForegroundColor Gray
    
    Write-Host "`nUseful Commands:" -ForegroundColor Cyan
    Write-Host "  Cluster info:     kubectl cluster-info" -ForegroundColor White
    Write-Host "  View nodes:       kubectl get nodes" -ForegroundColor White
    Write-Host "  View CRDs:        kubectl get crd | Select-String nephoran" -ForegroundColor White
    Write-Host "  View pods:        kubectl get pods -A" -ForegroundColor White
    Write-Host "  Switch namespace: kubectl config set-context --current --namespace=<namespace>" -ForegroundColor White
    
    if ($ClusterType -eq "kind") {
        Write-Host "`nKind-specific commands:" -ForegroundColor Cyan
        Write-Host "  Load image:       kind load docker-image <image>:tag --name $CLUSTER_NAME" -ForegroundColor White
        Write-Host "  Delete cluster:   kind delete cluster --name $CLUSTER_NAME" -ForegroundColor White
    } elseif ($ClusterType -eq "minikube") {
        Write-Host "`nMinikube-specific commands:" -ForegroundColor Cyan
        Write-Host "  Dashboard:        minikube dashboard -p $CLUSTER_NAME" -ForegroundColor White
        Write-Host "  Stop cluster:     minikube stop -p $CLUSTER_NAME" -ForegroundColor White
        Write-Host "  Delete cluster:   minikube delete -p $CLUSTER_NAME" -ForegroundColor White
    }
    
    if ($WithRegistry) {
        Write-Host "`nRegistry commands:" -ForegroundColor Cyan
        Write-Host "  Stop registry:    docker stop $REGISTRY_NAME" -ForegroundColor White
        Write-Host "  Start registry:   docker start $REGISTRY_NAME" -ForegroundColor White
        Write-Host "  View images:      curl http://localhost:$REGISTRY_PORT/v2/_catalog" -ForegroundColor White
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - Local Kubernetes Setup" -ForegroundColor Blue
Write-Host "Cluster Type: $ClusterType" -ForegroundColor Blue
Write-Host "Local Registry: $($WithRegistry ? 'Enabled' : 'Disabled')" -ForegroundColor Blue
Write-Host "=================================================" -ForegroundColor Blue

# Check if running as administrator for some operations
if ($ClusterType -eq "minikube" -and !(Test-Administrator)) {
    Write-Warning "Minikube setup may require administrator privileges"
    Write-Info "Consider running as administrator if you encounter permission issues"
}

try {
    Install-RequiredTools
    
    if (!(Test-DockerRunning)) {
        exit 1
    }
    
    Setup-LocalRegistry
    
    switch ($ClusterType) {
        "kind" { Setup-KindCluster }
        "minikube" { Setup-MinikubeCluster }
        "docker-desktop" { Setup-DockerDesktopKubernetes }
    }
    
    Deploy-CRDs
    Setup-Namespace
    Create-DevelopmentSecrets
    Validate-Setup
    Show-Summary
    
} catch {
    Write-Error "Setup failed: $($_.Exception.Message)"
    Write-Info "Stack trace: $($_.ScriptStackTrace)"
    exit 1
}
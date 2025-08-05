# Windows Deployment Script for Nephoran Intent Operator
# Compatible with Windows 11, PowerShell, and local Kubernetes clusters

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("local", "remote")]
    [string]$Environment,
    
    [switch]$SkipBuild,
    [switch]$Force,
    [switch]$Help
)

if ($Help) {
    Write-Host "Nephoran Intent Operator - Windows Deployment Script"
    Write-Host ""
    Write-Host "Usage: .\deploy-windows.ps1 -Environment <local|remote> [options]"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -Environment     Target environment (local or remote)"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -SkipBuild       Skip Docker image building"
    Write-Host "  -Force           Force rebuild and redeploy"
    Write-Host "  -Help            Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\deploy-windows.ps1 -Environment local"
    Write-Host "  .\deploy-windows.ps1 -Environment remote -SkipBuild"
    Write-Host ""
    exit 0
}

# Configuration
$REGISTRY = "us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran"
$APP_NAME = "telecom-llm-automation"

# Function to get Git version
function Get-GitVersion {
    try {
        return (git describe --tags --always --dirty 2>$null) -replace "`n", ""
    } catch {
        return "dev"
    }
}

# Function to check prerequisites
function Test-Prerequisites {
    Write-Host "Checking prerequisites..." -ForegroundColor Yellow
    
    $requiredTools = @("docker", "kubectl", "kustomize")
    $missing = @()
    
    foreach ($tool in $requiredTools) {
        if (!(Get-Command $tool -ErrorAction SilentlyContinue)) {
            $missing += $tool
        }
    }
    
    if ($missing.Count -gt 0) {
        Write-Host "Missing required tools: $($missing -join ', ')" -ForegroundColor Red
        Write-Host "Please run .\setup-windows.ps1 first" -ForegroundColor Yellow
        exit 1
    }
    
    # Check if Docker is running
    try {
        docker info | Out-Null
    } catch {
        Write-Host "Docker is not running. Please start Docker Desktop." -ForegroundColor Red
        exit 1
    }
    
    Write-Host "Prerequisites check passed" -ForegroundColor Green
}

# Function to detect cluster type
function Get-ClusterType {
    try {
        $context = kubectl config current-context 2>$null
        if ($context -like "*kind*") {
            return "kind"
        } elseif ($context -like "*minikube*") {
            return "minikube"
        } elseif ($context -like "*gke*") {
            return "gke"
        } else {
            return "unknown"
        }
    } catch {
        return "none"
    }
}

# Function to setup local cluster if needed
function Setup-LocalCluster {
    $clusterType = Get-ClusterType
    
    if ($clusterType -eq "none") {
        Write-Host "No Kubernetes cluster found. Setting up kind cluster..." -ForegroundColor Yellow
        
        # Check if kind is available
        if (!(Get-Command kind -ErrorAction SilentlyContinue)) {
            Write-Host "kind not found. Installing via Chocolatey..." -ForegroundColor Yellow
            choco install kind -y
            $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
        }
        
        # Create kind cluster
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
- role: worker
"@
        
        $kindConfig | Out-File -FilePath "kind-config.yaml" -Encoding UTF8
        kind create cluster --config kind-config.yaml
        Remove-Item "kind-config.yaml"
        
        Write-Host "Kind cluster created successfully" -ForegroundColor Green
    } else {
        Write-Host "Using existing cluster: $clusterType" -ForegroundColor Green
    }
}

# Function to build Docker images
function Build-DockerImages {
    if ($SkipBuild) {
        Write-Host "Skipping Docker image build" -ForegroundColor Yellow
        return
    }
    
    $version = Get-GitVersion
    Write-Host "Building Docker images with version: $version" -ForegroundColor Yellow
    
    $images = @(
        @{Name="llm-processor"; Dockerfile="cmd/llm-processor/Dockerfile"},
        @{Name="nephio-bridge"; Dockerfile="cmd/nephio-bridge/Dockerfile"},
        @{Name="oran-adaptor"; Dockerfile="cmd/oran-adaptor/Dockerfile"},
        @{Name="rag-api"; Dockerfile="pkg/rag/Dockerfile"}
    )
    
    foreach ($image in $images) {
        $tag = "$REGISTRY/$($image.Name):$version"
        Write-Host "Building $tag..." -ForegroundColor Blue
        
        docker build -t $tag -f $image.Dockerfile .
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to build $tag" -ForegroundColor Red
            exit 1
        }
    }
    
    Write-Host "All images built successfully" -ForegroundColor Green
}

# Function to load images into local cluster
function Load-ImagesLocal {
    $clusterType = Get-ClusterType
    $version = Get-GitVersion
    
    Write-Host "Loading images into $clusterType cluster..." -ForegroundColor Yellow
    
    $images = @("llm-processor", "nephio-bridge", "oran-adaptor", "rag-api")
    
    foreach ($imageName in $images) {
        $tag = "$REGISTRY/$imageName`:$version"
        Write-Host "Loading $tag..." -ForegroundColor Blue
        
        switch ($clusterType) {
            "kind" {
                kind load docker-image $tag --name nephoran-dev
            }
            "minikube" {
                minikube image load $tag
            }
            default {
                Write-Host "Unknown cluster type: $clusterType" -ForegroundColor Yellow
            }
        }
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to load $tag" -ForegroundColor Red
            exit 1
        }
    }
    
    Write-Host "All images loaded successfully" -ForegroundColor Green
}

# Function to push images to remote registry
function Push-ImagesRemote {
    $version = Get-GitVersion
    Write-Host "Pushing images to remote registry..." -ForegroundColor Yellow
    
    # Authenticate with GCP if needed
    try {
        gcloud auth configure-docker us-central1-docker.pkg.dev 2>$null
    } catch {
        Write-Host "GCP authentication failed. Please run 'gcloud auth login' first" -ForegroundColor Red
        exit 1
    }
    
    $images = @("llm-processor", "nephio-bridge", "oran-adaptor", "rag-api")
    
    foreach ($imageName in $images) {
        $tag = "$REGISTRY/$imageName`:$version"
        Write-Host "Pushing $tag..." -ForegroundColor Blue
        
        docker push $tag
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Failed to push $tag" -ForegroundColor Red
            exit 1
        }
    }
    
    Write-Host "All images pushed successfully" -ForegroundColor Green
}

# Function to deploy to Kubernetes
function Deploy-ToKubernetes {
    Write-Host "Deploying to Kubernetes..." -ForegroundColor Yellow
    
    # Apply CRDs first
    Write-Host "Applying Custom Resource Definitions..." -ForegroundColor Blue
    kubectl apply -f deployments/crds/
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed to apply CRDs" -ForegroundColor Red
        exit 1
    }
    
    # Wait for CRDs to be established
    Start-Sleep -Seconds 5
    
    # Deploy using Kustomize
    $overlay = if ($Environment -eq "local") { "local" } else { "remote" }
    Write-Host "Deploying with $overlay overlay..." -ForegroundColor Blue
    
    kustomize build "deployments/kustomize/overlays/$overlay" | kubectl apply -f -
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed to deploy with Kustomize" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "Deployment completed successfully" -ForegroundColor Green
}

# Function to verify deployment
function Verify-Deployment {
    Write-Host "Verifying deployment..." -ForegroundColor Yellow
    
    # Check pods
    Write-Host "Checking pod status..." -ForegroundColor Blue
    kubectl get pods -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check services
    Write-Host "Checking services..." -ForegroundColor Blue
    kubectl get services -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check CRDs
    Write-Host "Checking Custom Resource Definitions..." -ForegroundColor Blue
    kubectl get crd | Select-String "nephoran.com"
    
    # Wait for pods to be ready
    Write-Host "Waiting for pods to be ready..." -ForegroundColor Blue
    kubectl wait --for=condition=Ready pods -l app.kubernetes.io/part-of=nephoran-intent-operator --timeout=300s
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "All pods are ready" -ForegroundColor Green
    } else {
        Write-Host "Some pods are not ready. Check logs with:" -ForegroundColor Yellow
        Write-Host "kubectl logs -l app.kubernetes.io/part-of=nephoran-intent-operator" -ForegroundColor Yellow
    }
}

# Function to provide post-deployment instructions
function Show-PostDeploymentInstructions {
    Write-Host "`nDeployment Summary:" -ForegroundColor Blue
    Write-Host "===================" -ForegroundColor Blue
    
    Write-Host "Environment: $Environment" -ForegroundColor White
    Write-Host "Cluster Type: $(Get-ClusterType)" -ForegroundColor White
    Write-Host "Version: $(Get-GitVersion)" -ForegroundColor White
    
    Write-Host "`nUseful Commands:" -ForegroundColor Blue
    Write-Host "=================" -ForegroundColor Blue
    Write-Host "Check pod status:     kubectl get pods" -ForegroundColor White
    Write-Host "View logs:            kubectl logs -f deployment/nephio-bridge" -ForegroundColor White
    Write-Host "Check CRDs:           kubectl get crd | Select-String nephoran" -ForegroundColor White
    Write-Host "Test NetworkIntent:   kubectl apply -f archive/my-first-intent.yaml" -ForegroundColor White
    Write-Host "Port forward RAG API: kubectl port-forward svc/rag-api 5001:5001" -ForegroundColor White
    
    if ($Environment -eq "local") {
        Write-Host "`nLocal Development:" -ForegroundColor Blue
        Write-Host "==================" -ForegroundColor Blue
        Write-Host "Access cluster:       kubectl cluster-info" -ForegroundColor White
        Write-Host "Delete cluster:       kind delete cluster --name nephoran-dev" -ForegroundColor White
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - Windows Deployment" -ForegroundColor Blue
Write-Host "Environment: $Environment" -ForegroundColor Blue
Write-Host "=============================================" -ForegroundColor Blue

try {
    Test-Prerequisites
    
    if ($Environment -eq "local") {
        Setup-LocalCluster
    }
    
    Build-DockerImages
    
    if ($Environment -eq "local") {
        Load-ImagesLocal
    } else {
        Push-ImagesRemote
    }
    
    Deploy-ToKubernetes
    Verify-Deployment
    Show-PostDeploymentInstructions
    
    Write-Host "`nDeployment completed successfully!" -ForegroundColor Green
    
} catch {
    Write-Host "Deployment failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Stack trace: $($_.ScriptStackTrace)" -ForegroundColor Red
    exit 1
}
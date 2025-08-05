# Local Development Deployment Script for Nephoran Intent Operator
# Optimized for Windows 11 Kubernetes environments with local registry support

param(
    [switch]$UseLocalRegistry,
    [switch]$SkipBuild,
    [switch]$WatchLogs,
    [switch]$Force,
    [string]$Namespace = "nephoran-system",
    [switch]$Help
)

if ($Help) {
    Write-Host @"
Nephoran Intent Operator - Local Development Deployment

DESCRIPTION:
  Deploys the Nephoran Intent Operator to a local Kubernetes cluster
  with optimizations for development workflow and testing.

USAGE:
  .\local-deploy.ps1 [options]

OPTIONS:
  -UseLocalRegistry    Use local container registry (localhost:5001)
  -SkipBuild          Skip Docker image building
  -WatchLogs          Follow logs after deployment
  -Force              Force rebuild and redeploy
  -Namespace          Target namespace (default: nephoran-system)
  -Help               Show this help message

EXAMPLES:
  .\local-deploy.ps1                              # Standard local deployment
  .\local-deploy.ps1 -UseLocalRegistry            # With local registry
  .\local-deploy.ps1 -SkipBuild -WatchLogs        # Quick redeploy with log watching
  .\local-deploy.ps1 -Force -UseLocalRegistry     # Force complete rebuild

PREREQUISITES:
  - Local Kubernetes cluster (kind, minikube, or docker-desktop)
  - Docker running
  - kubectl configured for local cluster
  - CRDs already deployed (use .\local-k8s-setup.ps1)

"@
    exit 0
}

# Configuration
$LOCAL_REGISTRY = "localhost:5001"
$REGISTRY_PREFIX = if ($UseLocalRegistry) { $LOCAL_REGISTRY } else { "" }
$IMAGE_PULL_POLICY = if ($UseLocalRegistry) { "Always" } else { "Never" }

# Image definitions with local registry support
$IMAGES = @{
    "llm-processor" = @{
        "dockerfile" = "cmd/llm-processor/Dockerfile"
        "context" = "."
    }
    "nephio-bridge" = @{
        "dockerfile" = "cmd/nephio-bridge/Dockerfile"  
        "context" = "."
    }
    "oran-adaptor" = @{
        "dockerfile" = "cmd/oran-adaptor/Dockerfile"
        "context" = "."
    }
    "rag-api" = @{
        "dockerfile" = "pkg/rag/Dockerfile"
        "context" = "."
    }
}

# Color functions
function Write-Step($Message) { Write-Host "🔧 $Message" -ForegroundColor Cyan }
function Write-Success($Message) { Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Warning($Message) { Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error($Message) { Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Info($Message) { Write-Host "ℹ️  $Message" -ForegroundColor Blue }

function Get-GitVersion {
    try {
        $version = (git rev-parse --short HEAD 2>$null) -replace "`n", ""
        if ([string]::IsNullOrEmpty($version)) {
            return "dev-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        }
        return $version
    } catch {
        return "dev-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    }
}

function Test-Prerequisites {
    Write-Step "Checking prerequisites..."
    
    # Check kubectl
    if (!(Get-Command kubectl -ErrorAction SilentlyContinue)) {
        Write-Error "kubectl not found. Please install kubectl."
        exit 1
    }
    
    # Check Docker
    if (!(Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Error "Docker not found. Please install Docker."
        exit 1
    }
    
    # Test Docker daemon
    try {
        docker info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Docker not running"
        }
    } catch {
        Write-Error "Docker is not running. Please start Docker and try again."
        exit 1
    }
    
    # Test Kubernetes connectivity
    try {
        kubectl cluster-info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Kubernetes not accessible"
        }
    } catch {
        Write-Error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        Write-Info "Tip: Run .\local-k8s-setup.ps1 to setup a local cluster"
        exit 1
    }
    
    # Test local registry if enabled
    if ($UseLocalRegistry) {
        try {
            $response = Invoke-WebRequest -Uri "http://$LOCAL_REGISTRY/v2/" -UseBasicParsing -TimeoutSec 5
            if ($response.StatusCode -ne 200) {
                throw "Registry not responding"
            }
        } catch {
            Write-Error "Local registry at $LOCAL_REGISTRY is not accessible."
            Write-Info "Tip: Run .\local-k8s-setup.ps1 -WithRegistry to setup local registry"
            exit 1
        }
    }
    
    # Check if namespace exists
    kubectl get namespace $Namespace 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Namespace '$Namespace' does not exist. Creating it..."
        kubectl create namespace $Namespace
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to create namespace '$Namespace'"
            exit 1
        }
    }
    
    # Check CRDs
    $requiredCrds = @("networkintents.nephoran.com", "e2nodesets.nephoran.com", "managedelements.nephoran.com")
    foreach ($crd in $requiredCrds) {
        kubectl get crd $crd 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "CRD '$crd' not found. CRDs should be deployed first."
            Write-Info "Tip: Run .\local-k8s-setup.ps1 to deploy CRDs"
        }
    }
    
    Write-Success "Prerequisites check passed"
}

function Get-ClusterType {
    try {
        $context = kubectl config current-context 2>$null
        if ($context -like "*kind*") {
            return "kind"
        } elseif ($context -like "*minikube*") {
            return "minikube"
        } elseif ($context -like "*docker-desktop*") {
            return "docker-desktop"
        } else {
            return "unknown"
        }
    } catch {
        return "unknown"
    }
}

function Build-Images {
    if ($SkipBuild) {
        Write-Info "Skipping image build"
        return
    }
    
    $imageTag = Get-GitVersion
    $clusterType = Get-ClusterType
    
    Write-Step "Building images with tag: $imageTag"
    Write-Info "Target cluster type: $clusterType"
    
    foreach ($imageName in $IMAGES.Keys) {
        $imageConfig = $IMAGES[$imageName]
        $localTag = "${imageName}:${imageTag}"
        $registryTag = if ($UseLocalRegistry) { "${LOCAL_REGISTRY}/${imageName}:${imageTag}" } else { $localTag }
        
        Write-Step "Building $imageName..."
        
        # Build image
        docker build -t $localTag -f $imageConfig.dockerfile $imageConfig.context
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to build $imageName"
            exit 1
        }
        
        # Tag for registry if using local registry
        if ($UseLocalRegistry) {
            docker tag $localTag $registryTag
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to tag $imageName for registry"
                exit 1
            }
        }
        
        Write-Success "Built $imageName"
    }
    
    Write-Success "All images built successfully"
}

function Push-ImagesLocalRegistry {
    if (!$UseLocalRegistry) {
        return
    }
    
    $imageTag = Get-GitVersion
    Write-Step "Pushing images to local registry..."
    
    foreach ($imageName in $IMAGES.Keys) {
        $registryTag = "${LOCAL_REGISTRY}/${imageName}:${imageTag}"
        
        Write-Step "Pushing $registryTag..."
        docker push $registryTag
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to push $registryTag"
            exit 1
        }
        
        Write-Success "Pushed $imageName"
    }
    
    Write-Success "All images pushed to local registry"
}

function Load-ImagesIntoCluster {
    if ($UseLocalRegistry) {
        return  # Images will be pulled from registry
    }
    
    $imageTag = Get-GitVersion
    $clusterType = Get-ClusterType
    
    Write-Step "Loading images into $clusterType cluster..."
    
    foreach ($imageName in $IMAGES.Keys) {
        $localTag = "${imageName}:${imageTag}"
        
        Write-Step "Loading $localTag..."
        
        switch ($clusterType) {
            "kind" {
                # Extract cluster name from context
                $context = kubectl config current-context
                $clusterName = if ($context -like "kind-*") {
                    $context -replace "kind-", ""
                } else {
                    "nephoran-dev"  # default cluster name
                }
                
                kind load docker-image $localTag --name $clusterName
                if ($LASTEXITCODE -ne 0) {
                    Write-Error "Failed to load $localTag into kind cluster"
                    exit 1
                }
            }
            "minikube" {
                # Extract profile name if possible
                $profile = "nephoran-dev"  # default profile
                minikube image load $localTag -p $profile
                if ($LASTEXITCODE -ne 0) {
                    Write-Error "Failed to load $localTag into minikube"
                    exit 1
                }
            }
            "docker-desktop" {
                Write-Info "Docker Desktop shares Docker daemon - images are automatically available"
            }
            default {
                Write-Warning "Unknown cluster type '$clusterType'. Images may not be available in cluster."
            }
        }
        
        Write-Success "Loaded $imageName"
    }
    
    Write-Success "All images loaded into cluster"
}

function Generate-LocalManifests {
    $imageTag = Get-GitVersion
    Write-Step "Generating local deployment manifests..."
    
    # Create temporary directory for generated manifests
    $tempDir = "temp-manifests"
    if (Test-Path $tempDir) {
        Remove-Item $tempDir -Recurse -Force
    }
    New-Item -ItemType Directory $tempDir | Out-Null
    
    try {
        # Copy base kustomization
        $baseDir = "deployments\kustomize\base"
        $localOverlay = "deployments\kustomize\overlays\local"
        
        if (!(Test-Path $localOverlay)) {
            Write-Error "Local overlay not found: $localOverlay"
            exit 1
        }
        
        # Generate kustomization with local settings
        Push-Location $tempDir
        
        # Create kustomization.yaml for local development
        $kustomization = @"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: $Namespace

resources:
- ../$localOverlay

images:
"@

        foreach ($imageName in $IMAGES.Keys) {
            $imageRef = if ($UseLocalRegistry) {
                "${LOCAL_REGISTRY}/${imageName}:${imageTag}"
            } else {
                "${imageName}:${imageTag}"
            }
            
            $kustomization += @"

- name: $imageName
  newTag: $imageTag
"@
            if ($UseLocalRegistry) {
                $kustomization += @"

  newName: ${LOCAL_REGISTRY}/${imageName}
"@
            }
        }
        
        # Add patches for local development
        $kustomization += @"

patchesStrategicMerge:
- local-patches.yaml
"@
        
        $kustomization | Out-File -FilePath "kustomization.yaml" -Encoding UTF8
        
        # Create local patches
        $patches = @"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
spec:
  template:
    spec:
      containers:
      - name: llm-processor
        imagePullPolicy: $IMAGE_PULL_POLICY
        env:
        - name: ENVIRONMENT
          value: "local"
        - name: LOG_LEVEL
          value: "debug"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephio-bridge
spec:
  template:
    spec:
      containers:
      - name: nephio-bridge
        imagePullPolicy: $IMAGE_PULL_POLICY
        env:
        - name: ENVIRONMENT
          value: "local"
        - name: LOG_LEVEL
          value: "debug"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-adaptor
spec:
  template:
    spec:
      containers:
      - name: oran-adaptor
        imagePullPolicy: $IMAGE_PULL_POLICY
        env:
        - name: ENVIRONMENT
          value: "local"
        - name: LOG_LEVEL
          value: "debug"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rag-api
spec:
  template:
    spec:
      containers:
      - name: rag-api
        imagePullPolicy: $IMAGE_PULL_POLICY
        env:
        - name: ENVIRONMENT
          value: "local"
        - name: LOG_LEVEL
          value: "debug"
"@
        
        $patches | Out-File -FilePath "local-patches.yaml" -Encoding UTF8
        
        Write-Success "Generated local deployment manifests"
        
    } finally {
        Pop-Location
    }
    
    return $tempDir
}

function Deploy-ToCluster {
    $manifestDir = Generate-LocalManifests
    
    try {
        Write-Step "Deploying to Kubernetes cluster..."
        Write-Info "Target namespace: $Namespace"
        
        # Apply the generated manifests
        Push-Location $manifestDir
        kubectl apply -k .
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to apply Kubernetes manifests"
            exit 1
        }
        
        Pop-Location
        
        # Wait for rollout if forcing deployment
        if ($Force) {
            Write-Step "Waiting for deployments to roll out..."
            foreach ($imageName in $IMAGES.Keys) {
                kubectl rollout status deployment/$imageName -n $Namespace --timeout=300s
                if ($LASTEXITCODE -ne 0) {
                    Write-Warning "Deployment $imageName rollout may have issues"
                }
            }
        }
        
        Write-Success "Deployment completed"
        
    } finally {
        # Cleanup temp directory
        if (Test-Path $manifestDir) {
            Remove-Item $manifestDir -Recurse -Force
        }
    }
}

function Verify-Deployment {
    Write-Step "Verifying deployment..."
    
    # Check pods
    Write-Info "Pod Status:"
    kubectl get pods -n $Namespace -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check services
    Write-Info "Service Status:"
    kubectl get services -n $Namespace -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check deployments
    Write-Info "Deployment Status:"
    kubectl get deployments -n $Namespace
    
    # Wait for pods to be ready
    Write-Step "Waiting for pods to be ready..."
    $readyPods = 0
    $totalPods = $IMAGES.Keys.Count
    $maxWaitTime = 300  # 5 minutes
    $elapsed = 0
    
    while ($readyPods -lt $totalPods -and $elapsed -lt $maxWaitTime) {
        Start-Sleep -Seconds 10
        $elapsed += 10
        
        $podStatus = kubectl get pods -n $Namespace -l app.kubernetes.io/part-of=nephoran-intent-operator -o jsonpath='{.items[*].status.phase}' 2>$null
        if ($podStatus) {
            $runningPods = ($podStatus.Split(" ") | Where-Object { $_ -eq "Running" }).Count
            $readyPods = $runningPods
            Write-Info "Ready pods: $readyPods/$totalPods (waited ${elapsed}s)"
        }
    }
    
    if ($readyPods -eq $totalPods) {
        Write-Success "All pods are ready"
    } else {
        Write-Warning "Some pods may not be ready. Check logs for issues."
    }
}

function Watch-Logs {
    if (!$WatchLogs) {
        return
    }
    
    Write-Step "Starting log monitoring..."
    Write-Info "Press Ctrl+C to stop log monitoring"
    
    # Follow logs from all deployments
    $jobs = @()
    foreach ($imageName in $IMAGES.Keys) {
        $job = Start-Job -ScriptBlock {
            param($imageName, $namespace)
            kubectl logs -f deployment/$imageName -n $namespace
        } -ArgumentList $imageName, $Namespace
        $jobs += $job
    }
    
    try {
        # Wait for user to stop
        while ($true) {
            Start-Sleep -Seconds 1
            
            # Check if any jobs failed
            foreach ($job in $jobs) {
                if ($job.State -eq "Failed") {
                    Write-Warning "Log monitoring failed for one or more deployments"
                    break
                }
            }
        }
    } finally {
        # Cleanup jobs
        $jobs | Stop-Job
        $jobs | Remove-Job
    }
}

function Show-PostDeploymentInfo {
    Write-Host "`n🎉 Local Deployment Complete!" -ForegroundColor Green
    Write-Host "================================" -ForegroundColor Blue
    
    $imageTag = Get-GitVersion
    $clusterType = Get-ClusterType
    
    Write-Host "`nDeployment Information:" -ForegroundColor Cyan
    Write-Host "  Cluster Type: $clusterType" -ForegroundColor White
    Write-Host "  Namespace: $Namespace" -ForegroundColor White
    Write-Host "  Image Tag: $imageTag" -ForegroundColor White
    Write-Host "  Registry: $($UseLocalRegistry ? $LOCAL_REGISTRY : 'Local Docker')" -ForegroundColor White
    Write-Host "  Pull Policy: $IMAGE_PULL_POLICY" -ForegroundColor White
    
    Write-Host "`nUseful Commands:" -ForegroundColor Cyan
    Write-Host "  View pods:        kubectl get pods -n $Namespace" -ForegroundColor White
    Write-Host "  View services:    kubectl get svc -n $Namespace" -ForegroundColor White
    Write-Host "  View logs:        kubectl logs -f deployment/<name> -n $Namespace" -ForegroundColor White
    Write-Host "  Port forward:     kubectl port-forward svc/<service> <local-port>:<remote-port> -n $Namespace" -ForegroundColor White
    Write-Host "  Delete deployment: kubectl delete -k deployments/kustomize/overlays/local" -ForegroundColor White
    
    Write-Host "`nTesting:" -ForegroundColor Cyan
    Write-Host "  Apply test CRD:   kubectl apply -f archive/my-first-intent.yaml" -ForegroundColor White
    Write-Host "  View CRDs:        kubectl get networkintents -n $Namespace" -ForegroundColor White
    Write-Host "  Describe CRD:     kubectl describe networkintent <name> -n $Namespace" -ForegroundColor White
    
    Write-Host "`nDevelopment Workflow:" -ForegroundColor Cyan
    Write-Host "  1. Make code changes" -ForegroundColor White
    Write-Host "  2. Run: .\local-deploy.ps1 $(if($UseLocalRegistry){'-UseLocalRegistry '})-Force" -ForegroundColor White
    Write-Host "  3. Test your changes" -ForegroundColor White
    Write-Host "  4. Repeat" -ForegroundColor White
    
    if ($UseLocalRegistry) {
        Write-Host "`nLocal Registry:" -ForegroundColor Cyan
        Write-Host "  URL: $LOCAL_REGISTRY" -ForegroundColor White
        Write-Host "  Catalog: curl http://$LOCAL_REGISTRY/v2/_catalog" -ForegroundColor White
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - Local Development Deployment" -ForegroundColor Blue
Write-Host "Namespace: $Namespace" -ForegroundColor Blue
Write-Host "Local Registry: $($UseLocalRegistry ? 'Enabled' : 'Disabled')" -ForegroundColor Blue
Write-Host "=========================================================" -ForegroundColor Blue

try {
    Test-Prerequisites
    Build-Images
    Push-ImagesLocalRegistry
    Load-ImagesIntoCluster
    Deploy-ToCluster
    Verify-Deployment
    Show-PostDeploymentInfo
    Watch-Logs
    
} catch {
    Write-Error "Deployment failed: $($_.Exception.Message)"
    Write-Info "Stack trace: $($_.ScriptStackTrace)"
    exit 1
}
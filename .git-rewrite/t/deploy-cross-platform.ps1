# Cross-Platform Deployment Script for Nephoran Intent Operator
# Works on Windows PowerShell, PowerShell Core, and can be adapted for bash
# Maintains compatibility with existing deploy.sh functionality

param(
    [Parameter(Mandatory=$true, Position=0)]
    [ValidateSet("local", "remote")]
    [string]$Environment,
    
    [switch]$SkipBuild,
    [switch]$Force,
    [switch]$Verbose,
    [switch]$Help
)

if ($Help) {
    Write-Host @"
Nephoran Intent Operator - Cross-Platform Deployment Script

USAGE:
  .\deploy-cross-platform.ps1 <environment> [options]

ARGUMENTS:
  local   - Builds images and deploys them to the current Kubernetes context
            using the 'Never' imagePullPolicy. Ideal for local development
            with Minikube, Kind, or Docker Desktop.
  remote  - Builds and pushes images to Google Artifact Registry, then deploys
            to the current Kubernetes context. Requires GCP authentication.

OPTIONS:
  -SkipBuild      Skip Docker image building
  -Force          Force rebuild and redeploy
  -Verbose        Enable verbose output
  -Help           Show this help message

REQUIREMENTS:
  - docker, kubectl, kustomize CLI tools
  - For 'remote': gcloud CLI and GCP authentication
  - Git for version tagging

EXAMPLES:
  .\deploy-cross-platform.ps1 local
  .\deploy-cross-platform.ps1 remote -Verbose
  .\deploy-cross-platform.ps1 local -SkipBuild

"@
    exit 0
}

# --- Configuration (matches deploy.sh) ---
$GCP_PROJECT_ID = "poised-elf-466913-q2"
$GCP_REGION = "us-central1"
$AR_REPO = "nephoran"

# Cross-platform path handling
$PathSeparator = if ($IsWindows -or $env:OS -eq "Windows_NT") { "\" } else { "/" }

# Image definitions (matches deploy.sh)
$IMAGES = @{
    "llm-processor" = "cmd${PathSeparator}llm-processor"
    "nephio-bridge" = "cmd${PathSeparator}nephio-bridge"
    "oran-adaptor" = "cmd${PathSeparator}oran-adaptor"
    "rag-api" = "pkg${PathSeparator}rag"
}

# --- Functions ---

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { "Red" }
        "WARN" { "Yellow" }
        "SUCCESS" { "Green" }
        default { "White" }
    }
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Get-GitVersion {
    try {
        $version = (git rev-parse --short HEAD 2>$null) -replace "`n", ""
        if ([string]::IsNullOrEmpty($version)) {
            return "dev"
        }
        return $version
    } catch {
        Write-Log "Git not available, using 'dev' tag" "WARN"
        return "dev"
    }
}

function Test-Prerequisites {
    Write-Log "Checking prerequisites..."
    
    $requiredTools = @("docker", "kubectl", "kustomize")
    if ($Environment -eq "remote") {
        $requiredTools += "gcloud"
    }
    
    $missingTools = @()
    foreach ($tool in $requiredTools) {
        if (!(Get-Command $tool -ErrorAction SilentlyContinue)) {
            $missingTools += $tool
        }
    }
    
    if ($missingTools.Count -gt 0) {
        Write-Log "Missing required tools: $($missingTools -join ', ')" "ERROR"
        Write-Log "Please install missing tools and ensure they are in PATH" "ERROR"
        exit 1
    }
    
    # Check if Docker is running
    try {
        docker info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Docker not responding"
        }
    } catch {
        Write-Log "Docker is not running. Please start Docker Desktop or Docker daemon." "ERROR"
        exit 1
    }
    
    # Check Kubernetes connectivity
    try {
        kubectl cluster-info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Kubernetes not accessible"
        }
    } catch {
        Write-Log "Cannot connect to Kubernetes cluster. Please check your kubeconfig." "ERROR"
        exit 1
    }
    
    Write-Log "Prerequisites check passed" "SUCCESS"
}

function Build-Images {
    if ($SkipBuild) {
        Write-Log "Skipping Docker image build" "WARN"
        return
    }
    
    $imageTag = Get-GitVersion
    Write-Log "Building container images with tag: $imageTag"
    
    foreach ($imageName in $IMAGES.Keys) {
        $dockerPath = $IMAGES[$imageName]
        $fullImageName = "${imageName}:${imageTag}"
        $dockerfilePath = "${dockerPath}${PathSeparator}Dockerfile"
        
        Write-Log "Building $fullImageName from $dockerPath..."
        
        if (!(Test-Path $dockerfilePath)) {
            Write-Log "Dockerfile not found: $dockerfilePath" "ERROR"
            exit 1
        }
        
        docker build -t $fullImageName -f $dockerfilePath .
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Failed to build $fullImageName" "ERROR"
            exit 1
        }
        
        if ($Verbose) {
            Write-Log "Successfully built $fullImageName" "SUCCESS"
        }
    }
    
    Write-Log "All images built successfully" "SUCCESS"
}

function Get-ClusterType {
    try {
        $context = kubectl config current-context 2>$null
        if ($LASTEXITCODE -ne 0) {
            return "unknown"
        }
        
        if ($context -like "*minikube*") {
            return "minikube"
        } elseif ($context -like "*kind*") {
            return "kind"
        } elseif ($context -like "*docker-desktop*") {
            return "docker-desktop"
        } elseif ($context -like "*gke*") {
            return "gke"
        } else {
            return "unknown"
        }
    } catch {
        return "unknown"
    }
}

function Load-ImagesIntoLocalCluster {
    Write-Log "Loading images into local Kubernetes cluster..."
    
    $clusterType = Get-ClusterType
    $imageTag = Get-GitVersion
    
    Write-Log "Detected cluster type: $clusterType"
    
    foreach ($imageName in $IMAGES.Keys) {
        $fullImageName = "${imageName}:${imageTag}"
        Write-Log "Loading $fullImageName..."
        
        switch ($clusterType) {
            "minikube" {
                minikube image load $fullImageName
                if ($LASTEXITCODE -ne 0) {
                    Write-Log "Failed to load $fullImageName into minikube" "ERROR"
                    exit 1
                }
            }
            "kind" {
                # Extract cluster name from context (usually kind-<name>)
                $context = kubectl config current-context
                $clusterName = if ($context -like "kind-*") {
                    $context -replace "kind-", ""
                } else {
                    "kind"  # default cluster name
                }
                
                kind load docker-image $fullImageName --name $clusterName
                if ($LASTEXITCODE -ne 0) {
                    Write-Log "Failed to load $fullImageName into kind cluster" "ERROR"
                    exit 1
                }
            }
            "docker-desktop" {
                # Docker Desktop shares the Docker daemon, so images are already available
                Write-Log "Docker Desktop detected - images are automatically available"
            }
            default {
                Write-Log "Unknown or unsupported local cluster type: $clusterType" "WARN"
                Write-Log "Your Docker environment may be shared with Kubernetes" "WARN"
                Write-Log "If you see 'ErrImageNeverPull', please load the image manually:" "WARN"
                Write-Log "  - For Minikube: minikube image load $fullImageName" "WARN"
                Write-Log "  - For Kind: kind load docker-image $fullImageName" "WARN"
            }
        }
        
        if ($Verbose) {
            Write-Log "Successfully loaded $fullImageName" "SUCCESS"
        }
    }
    
    Write-Log "Image loading completed" "SUCCESS"
}

function Push-Images {
    Write-Log "Pushing images to Artifact Registry..."
    
    $imageTag = Get-GitVersion
    $arUrl = "${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPO}"
    $kustomizeDir = "deployments${PathSeparator}kustomize${PathSeparator}overlays${PathSeparator}${Environment}"
    
    # Authenticate Docker to Artifact Registry
    Write-Log "Authenticating Docker with GCP Artifact Registry..."
    gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev"
    if ($LASTEXITCODE -ne 0) {
        Write-Log "Failed to authenticate with GCP. Please run 'gcloud auth login' first" "ERROR"
        exit 1
    }
    
    foreach ($imageName in $IMAGES.Keys) {
        $localImage = "${imageName}:${imageTag}"
        $remoteImage = "${arUrl}/${imageName}:${imageTag}"
        
        Write-Log "Tagging and pushing $remoteImage"
        
        docker tag $localImage $remoteImage
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Failed to tag $localImage as $remoteImage" "ERROR"
            exit 1
        }
        
        docker push $remoteImage
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Failed to push $remoteImage" "ERROR"
            exit 1
        }
        
        # Update kustomization with the new image tag
        $imageInKustomization = "${arUrl}/${imageName}"
        Write-Log "Updating kustomization for $imageName..."
        
        Push-Location $kustomizeDir
        try {
            kustomize edit set image "${imageInKustomization}=${remoteImage}"
            if ($LASTEXITCODE -ne 0) {
                Write-Log "Failed to update kustomization for $imageName" "ERROR"
                exit 1
            }
        } finally {
            Pop-Location
        }
        
        if ($Verbose) {
            Write-Log "Successfully pushed and updated $imageName" "SUCCESS"
        }
    }
    
    Write-Log "All images pushed successfully" "SUCCESS"
}

function Update-LocalKustomization {
    $imageTag = Get-GitVersion
    $kustomizeDir = "deployments${PathSeparator}kustomize${PathSeparator}overlays${PathSeparator}${Environment}"
    
    Write-Log "Updating local kustomization with image tag: $imageTag"
    
    foreach ($imageName in $IMAGES.Keys) {
        Push-Location $kustomizeDir
        try {
            kustomize edit set image "${imageName}=${imageName}:${imageTag}"
            if ($LASTEXITCODE -ne 0) {
                Write-Log "Failed to update kustomization for $imageName" "ERROR"
                exit 1
            }
        } finally {
            Pop-Location
        }
        
        if ($Verbose) {
            Write-Log "Updated kustomization for $imageName" "SUCCESS"
        }
    }
}

function Deploy-ToKubernetes {
    $kustomizeDir = "deployments${PathSeparator}kustomize${PathSeparator}overlays${PathSeparator}${Environment}"
    
    Write-Log "Deploying to Kubernetes using kustomize overlay: $kustomizeDir"
    
    # Apply CRDs first
    $crdDir = "deployments${PathSeparator}crds"
    if (Test-Path $crdDir) {
        Write-Log "Applying Custom Resource Definitions..."
        kubectl apply -f $crdDir
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Failed to apply CRDs" "ERROR"
            exit 1
        }
        
        # Wait for CRDs to be established
        Start-Sleep -Seconds 3
    }
    
    # Apply main deployment
    kubectl apply -k $kustomizeDir
    if ($LASTEXITCODE -ne 0) {
        Write-Log "Failed to apply Kubernetes manifests" "ERROR"
        exit 1
    }
    
    Write-Log "Restarting deployments to apply changes..."
    foreach ($imageName in $IMAGES.Keys) {
        kubectl rollout restart deployment $imageName 2>$null
        # Don't fail if deployment doesn't exist yet
        if ($Verbose) {
            Write-Log "Restarted deployment: $imageName"
        }
    }
    
    Write-Log "Deployment complete" "SUCCESS"
}

function Verify-Deployment {
    Write-Log "Verifying deployment..."
    
    # Check pods
    Write-Log "Checking pod status..."
    kubectl get pods -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check services
    Write-Log "Checking services..."
    kubectl get services -l app.kubernetes.io/part-of=nephoran-intent-operator
    
    # Check CRDs
    Write-Log "Checking Custom Resource Definitions..."
    kubectl get crd | Select-String "nephoran.com"
    
    Write-Log "Deployment verification completed"
}

# --- Main Execution ---

Write-Log "Starting deployment for environment: $Environment"
Write-Log "Cluster type: $(Get-ClusterType)"

try {
    Test-Prerequisites
    Build-Images
    
    if ($Environment -eq "remote") {
        Push-Images
    } elseif ($Environment -eq "local") {
        Load-ImagesIntoLocalCluster
        Update-LocalKustomization
        Write-Log "Skipping image push for local deployment"
    } else {
        Write-Log "Invalid environment '$Environment'. Use 'local' or 'remote'." "ERROR"
        exit 1
    }
    
    Deploy-ToKubernetes
    Verify-Deployment
    
    Write-Log "Deployment completed successfully!" "SUCCESS"
    Write-Log "To monitor the deployment:"
    Write-Log "  kubectl get pods -w"
    Write-Log "  kubectl logs -f deployment/nephio-bridge"
    
} catch {
    Write-Log "Deployment failed: $($_.Exception.Message)" "ERROR"
    if ($Verbose) {
        Write-Log "Stack trace: $($_.ScriptStackTrace)" "ERROR"
    }
    exit 1
}
# Environment Validation Script for Nephoran Intent Operator
# Comprehensive validation of local development environment

param(
    [switch]$SkipClusterTests,
    [switch]$SkipImageTests,
    [switch]$SkipControllerTests,
    [switch]$Detailed,
    [switch]$Fix,
    [switch]$Help
)

if ($Help) {
    Write-Host @"
Nephoran Intent Operator - Environment Validation

DESCRIPTION:
  Validates the complete local development environment including
  Kubernetes cluster, container images, CRDs, and controller deployments.

USAGE:
  .\validate-environment.ps1 [options]

OPTIONS:
  -SkipClusterTests     Skip Kubernetes cluster validation
  -SkipImageTests       Skip container image validation
  -SkipControllerTests  Skip controller deployment validation
  -Detailed             Show detailed validation information
  -Fix                  Attempt to fix common issues automatically
  -Help                 Show this help message

EXAMPLES:
  .\validate-environment.ps1                    # Full validation
  .\validate-environment.ps1 -Detailed          # Detailed output
  .\validate-environment.ps1 -Fix               # Fix issues automatically
  .\validate-environment.ps1 -SkipImageTests    # Skip image validation

VALIDATION AREAS:
  - Tool installation and versions
  - Kubernetes cluster connectivity and health
  - CRD registration and status
  - Container images and registry
  - Controller deployments and logs
  - Network connectivity and services

"@
    exit 0
}

# Validation configuration
$REQUIRED_TOOLS = @{
    "docker" = @{
        "minVersion" = "20.0.0"
        "versionCommand" = "docker --version"
        "installCommand" = "choco install docker-desktop -y"
    }
    "kubectl" = @{
        "minVersion" = "1.27.0"
        "versionCommand" = "kubectl version --client --short"
        "installCommand" = "choco install kubernetes-cli -y"
    }
    "kind" = @{
        "minVersion" = "0.17.0"
        "versionCommand" = "kind version"
        "installCommand" = "choco install kind -y"
    }
    "kustomize" = @{
        "minVersion" = "4.5.0"
        "versionCommand" = "kustomize version --short"
        "installCommand" = "choco install kustomize -y"
    }
    "go" = @{
        "minVersion" = "1.20.0"
        "versionCommand" = "go version"
        "installCommand" = "choco install golang -y"
    }
}

$EXPECTED_CRDS = @(
    "networkintents.nephoran.com",
    "e2nodesets.nephoran.com", 
    "managedelements.nephoran.com"
)

$EXPECTED_DEPLOYMENTS = @(
    "nephio-bridge",
    "llm-processor",
    "oran-adaptor",
    "rag-api"
)

$EXPECTED_SERVICES = @(
    "nephio-bridge",
    "llm-processor", 
    "rag-api"
)

# Global validation results
$VALIDATION_RESULTS = @{
    "Tools" = @{}
    "Cluster" = @{}
    "CRDs" = @{}
    "Images" = @{}
    "Deployments" = @{}
    "Services" = @{}
    "Overall" = @{
        "Passed" = 0
        "Failed" = 0
        "Warnings" = 0
    }
}

# Color functions
function Write-Pass($Message) { Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Fail($Message) { Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warn($Message) { Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Info($Message) { Write-Host "ℹ️  $Message" -ForegroundColor Blue }
function Write-Step($Message) { Write-Host "🔧 $Message" -ForegroundColor Cyan }
function Write-Detail($Message) { if ($Detailed) { Write-Host "   $Message" -ForegroundColor Gray } }

function Add-Result {
    param([string]$Category, [string]$Test, [string]$Status, [string]$Message, [string]$Fix = "")
    
    $VALIDATION_RESULTS[$Category][$Test] = @{
        "Status" = $Status
        "Message" = $Message
        "Fix" = $Fix
    }
    
    switch ($Status) {
        "PASS" { $VALIDATION_RESULTS.Overall.Passed++ }
        "FAIL" { $VALIDATION_RESULTS.Overall.Failed++ }
        "WARN" { $VALIDATION_RESULTS.Overall.Warnings++ }
    }
}

function Parse-Version {
    param([string]$VersionString)
    
    # Extract version numbers using regex
    if ($VersionString -match '(\d+)\.(\d+)\.(\d+)') {
        return [Version]"$($Matches[1]).$($Matches[2]).$($Matches[3])"
    }
    return [Version]"0.0.0"
}

function Test-ToolVersion {
    param([string]$Tool, [hashtable]$Config)
    
    Write-Step "Validating $Tool..."
    
    # Check if tool exists
    if (!(Get-Command $Tool -ErrorAction SilentlyContinue)) {
        Write-Fail "$Tool not found"
        Add-Result "Tools" $Tool "FAIL" "Tool not installed" $Config.installCommand
        
        if ($Fix) {
            Write-Info "Attempting to install $Tool..."
            try {
                Invoke-Expression $Config.installCommand
                Write-Pass "Installed $Tool"
            } catch {
                Write-Fail "Failed to install $Tool"
            }
        }
        return
    }
    
    # Check version
    try {
        $versionOutput = Invoke-Expression $Config.versionCommand 2>$null
        $currentVersion = Parse-Version $versionOutput
        $minVersion = Parse-Version $Config.minVersion
        
        if ($currentVersion -ge $minVersion) {
            Write-Pass "$Tool version $currentVersion (>= $minVersion)"
            Add-Result "Tools" $Tool "PASS" "Version $currentVersion"
            Write-Detail "Command: $($Config.versionCommand)"
            Write-Detail "Output: $versionOutput"
        } else {
            Write-Warn "$Tool version $currentVersion (< $minVersion required)"
            Add-Result "Tools" $Tool "WARN" "Version $currentVersion is below minimum $minVersion" $Config.installCommand
        }
    } catch {
        Write-Warn "$Tool found but version check failed"
        Add-Result "Tools" $Tool "WARN" "Version check failed" ""
    }
}

function Test-AllTools {
    Write-Step "Validating required tools..."
    
    foreach ($tool in $REQUIRED_TOOLS.Keys) {
        Test-ToolVersion $tool $REQUIRED_TOOLS[$tool]
    }
}

function Test-DockerRunning {
    Write-Step "Testing Docker daemon..."
    
    try {
        docker info 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Pass "Docker daemon is running"
            Add-Result "Cluster" "DockerDaemon" "PASS" "Docker daemon running"
            
            # Get Docker info
            if ($Detailed) {
                $dockerInfo = docker info --format "{{.ServerVersion}}" 2>$null
                Write-Detail "Docker version: $dockerInfo"
                
                $containers = docker ps -q | Measure-Object | Select-Object -ExpandProperty Count
                Write-Detail "Running containers: $containers"
            }
        } else {
            throw "Docker daemon not responding"
        }
    } catch {
        Write-Fail "Docker daemon is not running"
        Add-Result "Cluster" "DockerDaemon" "FAIL" "Docker daemon not running" "Start Docker Desktop"
        
        if ($Fix) {
            Write-Info "Please start Docker Desktop manually"
            Write-Info "Waiting for Docker to start..."
            
            $attempts = 0
            $maxAttempts = 30
            
            while ($attempts -lt $maxAttempts) {
                Start-Sleep -Seconds 5
                $attempts++
                
                try {
                    docker info 2>$null | Out-Null
                    if ($LASTEXITCODE -eq 0) {
                        Write-Pass "Docker is now running"
                        break
                    }
                } catch {}
                
                Write-Info "Still waiting... ($attempts/$maxAttempts)"
            }
        }
    }
}

function Test-KubernetesCluster {
    if ($SkipClusterTests) {
        Write-Info "Skipping cluster tests"
        return
    }
    
    Write-Step "Testing Kubernetes cluster connectivity..."
    
    try {
        # Test basic connectivity
        kubectl cluster-info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Cluster not accessible"
        }
        
        Write-Pass "Kubernetes cluster is accessible"
        Add-Result "Cluster" "Connectivity" "PASS" "Cluster accessible"
        
        # Get cluster info
        $context = kubectl config current-context 2>$null
        $clusterType = if ($context -like "*kind*") { "kind" } elseif ($context -like "*minikube*") { "minikube" } elseif ($context -like "*docker-desktop*") { "docker-desktop" } else { "unknown" }
        
        Write-Detail "Context: $context"
        Write-Detail "Cluster type: $clusterType"
        
        # Test nodes
        $nodes = kubectl get nodes --no-headers 2>$null | Measure-Object | Select-Object -ExpandProperty Count
        if ($nodes -gt 0) {
            Write-Pass "Cluster has $nodes node(s)"
            Add-Result "Cluster" "Nodes" "PASS" "$nodes nodes available"
            
            if ($Detailed) {
                $nodeInfo = kubectl get nodes -o wide --no-headers 2>$null
                Write-Detail "Node details:"
                $nodeInfo.Split("`n") | ForEach-Object { Write-Detail "  $_" }
            }
        } else {
            Write-Warn "No nodes found in cluster"
            Add-Result "Cluster" "Nodes" "WARN" "No nodes found"
        }
        
        # Test API server
        $apiVersion = kubectl version --short 2>$null | Select-String "Server Version"
        if ($apiVersion) {
            Write-Pass "API server responding: $apiVersion"
            Add-Result "Cluster" "APIServer" "PASS" "API server responding"
        }
        
    } catch {
        Write-Fail "Cannot connect to Kubernetes cluster"
        Add-Result "Cluster" "Connectivity" "FAIL" "Cluster not accessible" "Run .\local-k8s-setup.ps1"
        
        if ($Fix) {
            Write-Info "Attempting to setup local cluster..."
            Write-Info "Please run: .\local-k8s-setup.ps1"
        }
    }
}

function Test-CRDs {
    Write-Step "Testing Custom Resource Definitions..."
    
    foreach ($crdName in $EXPECTED_CRDS) {
        Write-Step "Testing CRD: $crdName"
        
        # Check if CRD exists
        kubectl get crd $crdName 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            # Check if established
            $established = kubectl get crd $crdName -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>$null
            if ($established -eq "True") {
                Write-Pass "CRD $crdName is established"
                Add-Result "CRDs" $crdName "PASS" "Established"
                
                if ($Detailed) {
                    $version = kubectl get crd $crdName -o jsonpath='{.spec.versions[0].name}' 2>$null
                    $group = kubectl get crd $crdName -o jsonpath='{.spec.group}' 2>$null
                    Write-Detail "Version: $version"
                    Write-Detail "Group: $group"
                }
            } else {
                Write-Warn "CRD $crdName exists but not established"
                Add-Result "CRDs" $crdName "WARN" "Not established"
            }
        } else {
            Write-Fail "CRD $crdName not found"
            Add-Result "CRDs" $crdName "FAIL" "Not found" "Apply CRDs from deployments/crds/"
        }
    }
}

function Test-Images {
    if ($SkipImageTests) {
        Write-Info "Skipping image tests"
        return
    }
    
    Write-Step "Testing container images..."
    
    $imageTag = try { 
        (git rev-parse --short HEAD 2>$null) -replace "`n", ""
    } catch { 
        "latest" 
    }
    
    $expectedImages = @(
        "llm-processor:$imageTag",
        "nephio-bridge:$imageTag", 
        "oran-adaptor:$imageTag",
        "rag-api:$imageTag"
    )
    
    foreach ($image in $expectedImages) {
        Write-Step "Testing image: $image"
        
        # Check if image exists locally
        docker image inspect $image 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Pass "Image $image exists locally"
            Add-Result "Images" $image "PASS" "Available locally"
            
            if ($Detailed) {
                $size = docker image inspect $image --format='{{.Size}}' 2>$null
                $sizeMB = [math]::Round($size / 1MB, 2)
                Write-Detail "Size: ${sizeMB}MB"
                
                $created = docker image inspect $image --format='{{.Created}}' 2>$null
                Write-Detail "Created: $created"
            }
        } else {
            Write-Warn "Image $image not found locally"
            Add-Result "Images" $image "WARN" "Not found locally" "Run make docker-build"
        }
    }
    
    # Test local registry if enabled
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:5001/v2/" -UseBasicParsing -TimeoutSec 5 2>$null
        if ($response.StatusCode -eq 200) {
            Write-Pass "Local registry is accessible"
            Add-Result "Images" "LocalRegistry" "PASS" "Registry responding"
            
            if ($Detailed) {
                try {
                    $catalog = Invoke-WebRequest -Uri "http://localhost:5001/v2/_catalog" -UseBasicParsing 2>$null | ConvertFrom-Json
                    Write-Detail "Registry catalog: $($catalog.repositories -join ', ')"
                } catch {
                    Write-Detail "Could not retrieve registry catalog"
                }
            }
        }
    } catch {
        Write-Info "Local registry not running (optional)"
        Add-Result "Images" "LocalRegistry" "WARN" "Not running" "Run .\local-k8s-setup.ps1 -WithRegistry"
    }
}

function Test-Deployments {
    if ($SkipControllerTests) {
        Write-Info "Skipping controller tests"
        return
    }
    
    Write-Step "Testing deployments..."
    
    # Test namespace
    $namespace = "nephoran-system"
    kubectl get namespace $namespace 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Pass "Namespace $namespace exists"
        Add-Result "Deployments" "Namespace" "PASS" "Namespace exists"
    } else {
        Write-Warn "Namespace $namespace not found"
        Add-Result "Deployments" "Namespace" "WARN" "Namespace missing" "kubectl create namespace $namespace"
    }
    
    foreach ($deployment in $EXPECTED_DEPLOYMENTS) {
        Write-Step "Testing deployment: $deployment"
        
        # Check if deployment exists
        kubectl get deployment $deployment -n $namespace 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            # Check deployment status
            $ready = kubectl get deployment $deployment -n $namespace -o jsonpath='{.status.readyReplicas}' 2>$null
            $desired = kubectl get deployment $deployment -n $namespace -o jsonpath='{.spec.replicas}' 2>$null
            
            if ($ready -eq $desired -and $ready -gt 0) {
                Write-Pass "Deployment $deployment is ready ($ready/$desired)"
                Add-Result "Deployments" $deployment "PASS" "Ready ($ready/$desired)"
                
                if ($Detailed) {
                    $image = kubectl get deployment $deployment -n $namespace -o jsonpath='{.spec.template.spec.containers[0].image}' 2>$null
                    Write-Detail "Image: $image"
                    
                    $conditions = kubectl get deployment $deployment -n $namespace -o jsonpath='{.status.conditions[*].type}' 2>$null
                    Write-Detail "Conditions: $conditions"
                }
            } else {
                Write-Warn "Deployment $deployment not ready ($ready/$desired)"
                Add-Result "Deployments" $deployment "WARN" "Not ready ($ready/$desired)"
            }
        } else {
            Write-Warn "Deployment $deployment not found"
            Add-Result "Deployments" $deployment "WARN" "Not found" "Run .\local-deploy.ps1"
        }
    }
}

function Test-Services {
    Write-Step "Testing services..."
    
    $namespace = "nephoran-system"
    
    foreach ($service in $EXPECTED_SERVICES) {
        Write-Step "Testing service: $service"
        
        # Check if service exists
        kubectl get service $service -n $namespace 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Pass "Service $service exists"
            Add-Result "Services" $service "PASS" "Service exists"
            
            if ($Detailed) {
                $type = kubectl get service $service -n $namespace -o jsonpath='{.spec.type}' 2>$null
                $ports = kubectl get service $service -n $namespace -o jsonpath='{.spec.ports[*].port}' 2>$null
                Write-Detail "Type: $type"
                Write-Detail "Ports: $ports"
                
                # Test service connectivity (basic)
                $clusterIP = kubectl get service $service -n $namespace -o jsonpath='{.spec.clusterIP}' 2>$null
                if ($clusterIP -and $clusterIP -ne "None") {
                    Write-Detail "Cluster IP: $clusterIP"
                }
            }
        } else {
            Write-Warn "Service $service not found"
            Add-Result "Services" $service "WARN" "Not found"
        }
    }
}

function Test-NetworkConnectivity {
    Write-Step "Testing network connectivity..."
    
    # Test DNS resolution in cluster
    try {
        $dnsTest = kubectl run dns-test --image=busybox:1.28 --rm -it --restart=Never -- nslookup kubernetes.default 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Pass "DNS resolution working"
            Add-Result "Cluster" "DNS" "PASS" "DNS working"
        } else {
            Write-Warn "DNS resolution test failed"
            Add-Result "Cluster" "DNS" "WARN" "DNS issues"
        }
    } catch {
        Write-Warn "Could not test DNS resolution"
        Add-Result "Cluster" "DNS" "WARN" "DNS test failed"
    }
    
    # Test service mesh / ingress if present
    $ingressControllers = kubectl get pods -A -l app.kubernetes.io/component=controller 2>$null | Select-String "ingress"
    if ($ingressControllers) {
        Write-Pass "Ingress controller detected"
        Add-Result "Cluster" "Ingress" "PASS" "Ingress available"
    } else {
        Write-Info "No ingress controller detected (optional)"
        Add-Result "Cluster" "Ingress" "WARN" "No ingress"
    }
}

function Generate-ValidationReport {
    Write-Host "`n📊 Environment Validation Report" -ForegroundColor Blue
    Write-Host "=================================" -ForegroundColor Blue
    
    $categories = @("Tools", "Cluster", "CRDs", "Images", "Deployments", "Services")
    
    foreach ($category in $categories) {
        if ($VALIDATION_RESULTS[$category].Count -eq 0) { continue }
        
        Write-Host "`n$category`:" -ForegroundColor Cyan
        
        foreach ($test in $VALIDATION_RESULTS[$category].Keys) {
            $result = $VALIDATION_RESULTS[$category][$test]
            $icon = switch ($result.Status) {
                "PASS" { "✅" }
                "FAIL" { "❌" }
                "WARN" { "⚠️ " }
                default { "❓" }
            }
            
            Write-Host "  $icon $test`: $($result.Message)" -ForegroundColor White
            
            if ($result.Status -eq "FAIL" -and $result.Fix) {
                Write-Host "    💡 Fix: $($result.Fix)" -ForegroundColor Gray
            }
        }
    }
    
    # Overall summary
    $total = $VALIDATION_RESULTS.Overall.Passed + $VALIDATION_RESULTS.Overall.Failed + $VALIDATION_RESULTS.Overall.Warnings
    $passRate = if ($total -gt 0) { [math]::Round(($VALIDATION_RESULTS.Overall.Passed / $total) * 100, 1) } else { 0 }
    
    Write-Host "`nSummary:" -ForegroundColor Blue
    Write-Host "  Total Tests: $total" -ForegroundColor White
    Write-Host "  Passed: $($VALIDATION_RESULTS.Overall.Passed) ($passRate%)" -ForegroundColor Green
    Write-Host "  Failed: $($VALIDATION_RESULTS.Overall.Failed)" -ForegroundColor Red
    Write-Host "  Warnings: $($VALIDATION_RESULTS.Overall.Warnings)" -ForegroundColor Yellow
    
    # Overall status
    if ($VALIDATION_RESULTS.Overall.Failed -eq 0) {
        if ($VALIDATION_RESULTS.Overall.Warnings -eq 0) {
            Write-Host "`n🎉 Environment is fully validated!" -ForegroundColor Green
        } else {
            Write-Host "`n⚠️  Environment is mostly ready (some warnings)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "`n❌ Environment has issues that need attention" -ForegroundColor Red
    }
    
    # Next steps
    if ($VALIDATION_RESULTS.Overall.Failed -gt 0 -or $VALIDATION_RESULTS.Overall.Warnings -gt 0) {
        Write-Host "`nRecommended Actions:" -ForegroundColor Cyan
        
        if ($VALIDATION_RESULTS.Tools.Values | Where-Object { $_.Status -eq "FAIL" }) {
            Write-Host "  1. Install missing tools" -ForegroundColor White
        }
        
        if ($VALIDATION_RESULTS.Cluster.Values | Where-Object { $_.Status -eq "FAIL" }) {
            Write-Host "  2. Setup local Kubernetes cluster: .\local-k8s-setup.ps1" -ForegroundColor White
        }
        
        if ($VALIDATION_RESULTS.CRDs.Values | Where-Object { $_.Status -eq "FAIL" }) {
            Write-Host "  3. Deploy CRDs: kubectl apply -f deployments/crds/" -ForegroundColor White
        }
        
        if ($VALIDATION_RESULTS.Images.Values | Where-Object { $_.Status -ne "PASS" }) {
            Write-Host "  4. Build images: make docker-build" -ForegroundColor White
        }
        
        if ($VALIDATION_RESULTS.Deployments.Values | Where-Object { $_.Status -ne "PASS" }) {
            Write-Host "  5. Deploy application: .\local-deploy.ps1" -ForegroundColor White
        }
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - Environment Validation" -ForegroundColor Blue
Write-Host "Cluster Tests: $($SkipClusterTests ? 'Disabled' : 'Enabled')" -ForegroundColor Blue
Write-Host "Image Tests: $($SkipImageTests ? 'Disabled' : 'Enabled')" -ForegroundColor Blue
Write-Host "Controller Tests: $($SkipControllerTests ? 'Disabled' : 'Enabled')" -ForegroundColor Blue
Write-Host "=================================================" -ForegroundColor Blue

try {
    Test-AllTools
    Test-DockerRunning
    Test-KubernetesCluster
    Test-CRDs
    Test-Images
    Test-Deployments
    Test-Services
    Test-NetworkConnectivity
    
    Generate-ValidationReport
    
} catch {
    Write-Fail "Validation failed: $($_.Exception.Message)"
    if ($Detailed) {
        Write-Info "Stack trace: $($_.ScriptStackTrace)"
    }
    exit 1
}
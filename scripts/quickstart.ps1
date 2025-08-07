# Nephoran Intent Operator - Quick Start Script for Windows
# This script automates the 15-minute quickstart tutorial on Windows
# Usage: .\scripts\quickstart.ps1 [-Cleanup] [-SkipPrereq] [-Production]

param(
    [switch]$Cleanup,
    [switch]$SkipPrereq,
    [switch]$Production,
    [switch]$Help
)

# Configuration
$ClusterName = "nephoran-quickstart"
$Namespace = "nephoran-system"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$DemoMode = -not $Production

# Colors for output
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Blue }
function Write-Success { Write-Host "[SUCCESS] $args" -ForegroundColor Green }
function Write-Warning { Write-Host "[WARNING] $args" -ForegroundColor Yellow }
function Write-Error { Write-Host "[ERROR] $args" -ForegroundColor Red }
function Write-Step { 
    Write-Host "`n" -NoNewline
    Write-Host ("=" * 60) -ForegroundColor Magenta
    Write-Host "STEP: $args" -ForegroundColor White
    Write-Host ("=" * 60) -ForegroundColor Magenta
    Write-Host ""
}

# Show help
if ($Help) {
    Write-Host @"
Nephoran Intent Operator - Quick Start Script for Windows

Usage: .\scripts\quickstart.ps1 [OPTIONS]

Options:
  -Cleanup      Clean up all resources and exit
  -SkipPrereq   Skip prerequisite checks
  -Production   Use production configuration (requires API keys)
  -Help         Show this help message

Examples:
  .\scripts\quickstart.ps1                    # Run quickstart in demo mode
  .\scripts\quickstart.ps1 -Production        # Run with real API keys
  .\scripts\quickstart.ps1 -Cleanup           # Clean up all resources
"@
    exit 0
}

# Function to check if a command exists
function Test-Command {
    param($Command)
    
    try {
        if (Get-Command $Command -ErrorAction Stop) {
            Write-Success "$Command is installed ✓"
            return $true
        }
    } catch {
        Write-Error "$Command is not installed ✗"
        return $false
    }
}

# Check prerequisites
function Test-Prerequisites {
    Write-Step "Checking Prerequisites (2 minutes)"
    
    $allGood = $true
    
    Write-Info "Checking required tools..."
    
    # Check Docker
    if (-not (Test-Command "docker")) {
        $allGood = $false
        Write-Host "  Install Docker Desktop from: https://www.docker.com/products/docker-desktop" -ForegroundColor Cyan
    }
    
    # Check kubectl
    if (-not (Test-Command "kubectl")) {
        $allGood = $false
        Write-Host "  Install kubectl with: choco install kubernetes-cli" -ForegroundColor Cyan
    }
    
    # Check git
    if (-not (Test-Command "git")) {
        $allGood = $false
        Write-Host "  Install git from: https://git-scm.com/download/win" -ForegroundColor Cyan
    }
    
    # Check kind
    if (-not (Test-Command "kind")) {
        $allGood = $false
        Write-Host "  Install kind with: choco install kind" -ForegroundColor Cyan
    }
    
    if (-not $allGood) {
        Write-Error "Some prerequisites are missing. Please install them first."
        Write-Host ""
        Write-Host "Quick installation with Chocolatey:" -ForegroundColor Yellow
        Write-Host "  choco install docker-desktop kubernetes-cli kind git -y" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Or with winget:" -ForegroundColor Yellow
        Write-Host "  winget install Docker.DockerDesktop" -ForegroundColor Cyan
        Write-Host "  winget install Kubernetes.kubectl" -ForegroundColor Cyan
        exit 1
    }
    
    # Check Docker daemon
    try {
        docker info | Out-Null
    } catch {
        Write-Error "Docker daemon is not running. Please start Docker Desktop."
        exit 1
    }
    
    Write-Success "All prerequisites met! ✓"
}

# Clean up resources
function Remove-Resources {
    Write-Step "Cleaning up resources"
    
    # Delete network intent
    kubectl delete networkintent deploy-amf-quickstart --ignore-not-found=true 2>$null
    
    # Delete namespace
    kubectl delete namespace $Namespace --ignore-not-found=true 2>$null
    
    # Delete CRDs
    kubectl delete crd networkintents.nephoran.com --ignore-not-found=true 2>$null
    kubectl delete crd managedelements.nephoran.com --ignore-not-found=true 2>$null
    kubectl delete crd e2nodesets.nephoran.com --ignore-not-found=true 2>$null
    
    # Delete Kind cluster
    $clusters = kind get clusters 2>$null
    if ($clusters -contains $ClusterName) {
        Write-Info "Deleting Kind cluster: $ClusterName"
        kind delete cluster --name $ClusterName
        Write-Success "Cluster deleted successfully"
    } else {
        Write-Info "Cluster $ClusterName not found, skipping deletion"
    }
    
    # Clean up temporary files
    Remove-Item -Path "kind-config.yaml", "my-first-intent.yaml", "validate-quickstart.ps1" -ErrorAction SilentlyContinue
    
    Write-Success "Cleanup completed!"
}

# Setup Kubernetes cluster
function New-Cluster {
    Write-Step "Setting up Kubernetes Cluster (5 minutes)"
    
    # Check if cluster already exists
    $clusters = kind get clusters 2>$null
    if ($clusters -contains $ClusterName) {
        Write-Warning "Cluster $ClusterName already exists"
        $response = Read-Host "Delete and recreate? (y/n)"
        if ($response -eq 'y') {
            kind delete cluster --name $ClusterName
        } else {
            Write-Info "Using existing cluster"
            return
        }
    }
    
    Write-Info "Creating Kind cluster configuration..."
    @"
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: $ClusterName
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
  - containerPort: 8080
    hostPort: 8080
    protocol: TCP
- role: worker
- role: worker
"@ | Out-File -FilePath "kind-config.yaml" -Encoding UTF8
    
    Write-Info "Creating Kind cluster: $ClusterName"
    kind create cluster --config=kind-config.yaml
    
    Write-Info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
    
    Write-Success "Cluster created successfully!"
    kubectl get nodes
}

# Install CRDs
function Install-CRDs {
    Write-Info "Installing Custom Resource Definitions..."
    
    # Create CRDs
    $crdContent = @"
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: networkintents.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: NetworkIntent
    listKind: NetworkIntentList
    plural: networkintents
    singular: networkintent
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              intent:
                type: string
          status:
            type: object
            properties:
              phase:
                type: string
              message:
                type: string
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: managedelements.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: ManagedElement
    listKind: ManagedElementList
    plural: managedelements
    singular: managedelement
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: e2nodesets.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: E2NodeSet
    listKind: E2NodeSetList
    plural: e2nodesets
    singular: e2nodeset
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
"@
    
    $crdContent | kubectl apply -f -
    
    Write-Info "Verifying CRD installation..."
    kubectl get crds | Select-String nephoran
    
    Write-Success "CRDs installed successfully!"
}

# Deploy controller
function Deploy-Controller {
    Write-Step "Deploying Nephoran Controller"
    
    Write-Info "Creating namespace: $Namespace"
    kubectl create namespace $Namespace --dry-run=client -o yaml | kubectl apply -f -
    
    if ($DemoMode) {
        Write-Info "Deploying in DEMO mode (no API keys required)..."
        
        # Create mock secrets
        kubectl create secret generic llm-secrets `
            --from-literal=openai-api-key=demo-key-not-for-production `
            -n $Namespace --dry-run=client -o yaml | kubectl apply -f -
    } else {
        Write-Info "Deploying in PRODUCTION mode..."
        
        # Check for API key
        if (-not $env:OPENAI_API_KEY) {
            Write-Error "OPENAI_API_KEY environment variable not set"
            Write-Host "Please set: `$env:OPENAI_API_KEY = 'your-key-here'" -ForegroundColor Cyan
            exit 1
        }
        
        kubectl create secret generic llm-secrets `
            --from-literal=openai-api-key="$env:OPENAI_API_KEY" `
            -n $Namespace --dry-run=client -o yaml | kubectl apply -f -
    }
    
    Write-Info "Creating controller configuration..."
    $configMode = if ($DemoMode) { "demo" } else { "production" }
    $llmProvider = if ($DemoMode) { "mock" } else { "openai" }
    $mockResponses = if ($DemoMode) { "true" } else { "false" }
    
    @"
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-config
  namespace: $Namespace
data:
  config.yaml: |
    mode: $configMode
    llm:
      provider: $llmProvider
      mockResponses: $mockResponses
      model: gpt-4o-mini
    rag:
      enabled: false
    telemetry:
      enabled: true
      prometheus:
        port: 9090
    logging:
      level: info
      format: json
"@ | kubectl apply -f -
    
    Write-Info "Deploying controller..."
    $demoValue = if ($DemoMode) { "true" } else { "false" }
    
    @"
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nephoran-controller
  namespace: $Namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-controller
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["nephoran.com"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nephoran-controller
subjects:
- kind: ServiceAccount
  name: nephoran-controller
  namespace: $Namespace
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-controller
  namespace: $Namespace
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nephoran-controller
  template:
    metadata:
      labels:
        app: nephoran-controller
    spec:
      serviceAccountName: nephoran-controller
      containers:
      - name: controller
        image: ghcr.io/thc1006/nephoran-intent-operator:latest
        imagePullPolicy: Always
        env:
        - name: DEMO_MODE
          value: "$demoValue"
        - name: LOG_LEVEL
          value: "info"
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 9090
          name: health
        volumeMounts:
        - name: config
          mountPath: /etc/nephoran
        - name: secrets
          mountPath: /etc/secrets
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: nephoran-config
      - name: secrets
        secret:
          secretName: llm-secrets
---
apiVersion: v1
kind: Service
metadata:
  name: nephoran-controller
  namespace: $Namespace
spec:
  selector:
    app: nephoran-controller
  ports:
  - name: metrics
    port: 8080
    targetPort: metrics
  - name: health
    port: 9090
    targetPort: health
"@ | kubectl apply -f -
    
    Write-Info "Waiting for controller to be ready..."
    try {
        kubectl wait --for=condition=available --timeout=120s `
            deployment/nephoran-controller -n $Namespace
        Write-Success "Controller deployed successfully!"
    } catch {
        Write-Error "Controller failed to start. Checking logs..."
        kubectl logs -n $Namespace deployment/nephoran-controller --tail=50
        exit 1
    }
}

# Deploy first intent
function Deploy-FirstIntent {
    Write-Step "Deploying Your First Intent (5 minutes)"
    
    Write-Info "Creating NetworkIntent for AMF deployment..."
    @"
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: deploy-amf-quickstart
  namespace: default
  labels:
    tutorial: quickstart
    component: amf
    generated-from: quickstart-script
spec:
  intent: |
    Deploy a production-ready AMF (Access and Mobility Management Function) 
    for a 5G core network with the following requirements:
    - High availability with 3 replicas
    - Auto-scaling enabled (min: 3, max: 10 pods)
    - Resource limits: 2 CPU cores, 4GB memory per pod
    - Enable prometheus monitoring on port 9090
    - Configure for urban area with expected 100k UE connections
    - Set up with standard 3GPP interfaces (N1, N2, N11)
    - Include health checks and readiness probes
    - Deploy in namespace: default
"@ | Out-File -FilePath "my-first-intent.yaml" -Encoding UTF8
    
    Write-Info "Applying the NetworkIntent..."
    kubectl apply -f my-first-intent.yaml
    
    Write-Info "Waiting for intent processing..."
    
    # Monitor intent status
    for ($i = 1; $i -le 30; $i++) {
        $status = kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.phase}' 2>$null
        if (-not $status) { $status = "Unknown" }
        
        switch ($status) {
            { $_ -in "Deployed", "Ready", "Completed" } {
                Write-Success "Intent processed successfully! Status: $status"
                return
            }
            { $_ -in "Failed", "Error" } {
                Write-Error "Intent processing failed! Status: $status"
                kubectl describe networkintent deploy-amf-quickstart
                exit 1
            }
            default {
                Write-Host -NoNewline "`rProcessing intent... Status: $status ($i/30)" -ForegroundColor Cyan
                Start-Sleep -Seconds 2
            }
        }
    }
    Write-Host ""
    
    Write-Info "Checking generated resources..."
    kubectl get all -l generated-from=deploy-amf-quickstart 2>$null
    
    Write-Success "First intent deployed successfully!"
}

# Validate deployment
function Test-Deployment {
    Write-Step "Validating Deployment (2 minutes)"
    
    $validationScript = @'
$Errors = 0
$Warnings = 0

Write-Host "🔍 Running Nephoran Quickstart Validation..." -ForegroundColor Cyan
Write-Host "==========================================="

# Check CRDs
Write-Host "`n📋 Checking CRDs..." -ForegroundColor Yellow
@("networkintents", "managedelements", "e2nodesets") | ForEach-Object {
    try {
        kubectl get crd "$_.nephoran.com" 2>&1 | Out-Null
        Write-Host "✓ $_.nephoran.com installed" -ForegroundColor Green
    } catch {
        Write-Host "✗ $_.nephoran.com missing" -ForegroundColor Red
        $Errors++
    }
}

# Check controller
Write-Host "`n🎮 Checking Controller..." -ForegroundColor Yellow
try {
    $ready = kubectl get deployment nephoran-controller -n nephoran-system -o jsonpath='{.status.readyReplicas}' 2>$null
    $desired = kubectl get deployment nephoran-controller -n nephoran-system -o jsonpath='{.spec.replicas}' 2>$null
    
    if ($ready -eq $desired) {
        Write-Host "✓ Controller running ($ready/$desired replicas ready)" -ForegroundColor Green
    } else {
        Write-Host "⚠ Controller partially ready ($ready/$desired replicas)" -ForegroundColor Yellow
        $Warnings++
    }
} catch {
    Write-Host "✗ Controller deployment not found" -ForegroundColor Red
    $Errors++
}

# Check intent
Write-Host "`n📝 Checking Network Intent..." -ForegroundColor Yellow
try {
    $status = kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.phase}' 2>$null
    $message = kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.message}' 2>$null
    
    Write-Host "✓ Intent found - Status: $status" -ForegroundColor Green
    if ($message) {
        Write-Host "   Message: $message"
    }
} catch {
    Write-Host "✗ Intent not found" -ForegroundColor Red
    $Errors++
}

# Summary
Write-Host "`n==========================================="
Write-Host "📊 VALIDATION SUMMARY" -ForegroundColor Cyan
Write-Host "==========================================="

if ($Errors -eq 0) {
    if ($Warnings -eq 0) {
        Write-Host "🎉 PERFECT! All checks passed!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Your Nephoran Intent Operator quickstart is complete!"
        Write-Host ""
        Write-Host "Next steps:"
        Write-Host "  1. View controller logs: kubectl logs -n nephoran-system deployment/nephoran-controller"
        Write-Host "  2. Try more intents: kubectl apply -f examples/networkintent-example.yaml"
        Write-Host "  3. Access metrics: kubectl port-forward -n nephoran-system svc/nephoran-controller 8080:8080"
    } else {
        Write-Host "✅ SUCCESS with $Warnings warnings" -ForegroundColor Yellow
        Write-Host "The system is functional but may need minor adjustments."
    }
} else {
    Write-Host "❌ ISSUES FOUND: $Errors errors, $Warnings warnings" -ForegroundColor Red
    Write-Host ""
    Write-Host "Troubleshooting tips:"
    Write-Host "  1. Check controller logs: kubectl logs -n nephoran-system deployment/nephoran-controller"
    Write-Host "  2. Describe the intent: kubectl describe networkintent deploy-amf-quickstart"
    Write-Host "  3. Check events: kubectl get events --sort-by='.lastTimestamp'"
}

exit $Errors
'@
    
    $validationScript | Out-File -FilePath "validate-quickstart.ps1" -Encoding UTF8
    & .\validate-quickstart.ps1
}

# Show summary
function Show-Summary {
    Write-Step "Quick Start Complete! 🎉"
    
    Write-Host ("=" * 60) -ForegroundColor Green
    Write-Host "QUICKSTART SUMMARY" -ForegroundColor White
    Write-Host ("=" * 60) -ForegroundColor Green
    Write-Host ""
    Write-Host "✅ Cluster Name: $ClusterName"
    Write-Host "✅ Namespace: $Namespace"
    Write-Host "✅ Mode: $(if ($DemoMode) { 'Demo' } else { 'Production' })"
    Write-Host ""
    Write-Host "Useful Commands:" -ForegroundColor Cyan
    Write-Host "  # View controller logs"
    Write-Host "  kubectl logs -n $Namespace deployment/nephoran-controller -f"
    Write-Host ""
    Write-Host "  # Watch intent status"
    Write-Host "  kubectl get networkintents -w"
    Write-Host ""
    Write-Host "  # Port-forward to access metrics"
    Write-Host "  kubectl port-forward -n $Namespace svc/nephoran-controller 8080:8080"
    Write-Host ""
    Write-Host "  # Deploy more examples"
    Write-Host "  kubectl apply -f examples/networkintent-example.yaml"
    Write-Host ""
    Write-Host "  # Clean up everything"
    Write-Host "  .\scripts\quickstart.ps1 -Cleanup"
    Write-Host ""
    Write-Host ("=" * 60) -ForegroundColor Green
    Write-Host ""
    Write-Host "📖 Documentation: https://github.com/thc1006/nephoran-intent-operator"
    Write-Host "🐛 Issues: https://github.com/thc1006/nephoran-intent-operator/issues"
    Write-Host ""
    Write-Host "Thank you for trying Nephoran Intent Operator!" -ForegroundColor Green
}

# Main execution
function Main {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║     Nephoran Intent Operator - Quick Start Script      ║" -ForegroundColor Cyan
    Write-Host "║           15-Minute Setup & Deployment                 ║" -ForegroundColor Yellow
    Write-Host "╚════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    # Start timer
    $StartTime = Get-Date
    
    # Handle cleanup mode
    if ($Cleanup) {
        Remove-Resources
        exit 0
    }
    
    # Run quickstart steps
    if (-not $SkipPrereq) {
        Test-Prerequisites
    }
    
    New-Cluster
    Install-CRDs
    Deploy-Controller
    Deploy-FirstIntent
    Test-Deployment
    
    # Calculate elapsed time
    $EndTime = Get-Date
    $Elapsed = $EndTime - $StartTime
    $Minutes = [math]::Floor($Elapsed.TotalMinutes)
    $Seconds = $Elapsed.Seconds
    
    Show-Summary
    
    Write-Host "Total time: $Minutes minutes $Seconds seconds" -ForegroundColor Green
    
    if ($Minutes -le 15) {
        Write-Host "✅ Completed within the 15-minute target!" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Took longer than 15 minutes, but that's okay!" -ForegroundColor Yellow
    }
}

# Run main function
Main
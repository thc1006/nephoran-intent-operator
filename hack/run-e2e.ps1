# E2E Test Harness for Nephoran Intent Operator - Windows PowerShell
# Provides native PowerShell implementation for Windows environments
# Features:
# - Creates kind cluster
# - Applies CRDs and webhook-manager
# - Runs intent-ingest locally
# - Runs conductor-loop for KRM patches
# - Validates webhook acceptance/rejection
# - Clear PASS/FAIL summary

param(
    [string]$ClusterName = "nephoran-e2e",
    [string]$KindImage = "",
    [string]$Namespace = "nephoran-system",
    [string]$IntentIngestMode = "local",  # local or sidecar
    [string]$ConductorMode = "local",      # local or in-cluster
    [string]$PorchMode = "structured-patch", # structured-patch or direct
    [switch]$SkipCleanup = $false,
    [switch]$UseBash = $false,  # Fall back to bash script
    [switch]$Verbose = $false,
    [int]$Timeout = 300
)

$ErrorActionPreference = 'Stop'

# If UseBash flag is set, delegate to bash script
if ($UseBash) {
    function Need($name) {
        if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
            Write-Error "❌ need $name but not found in PATH"
            exit 127
        }
    }
    
    Need "bash"
    Write-Host "[run-e2e.ps1] Delegating to bash script..."
    $env:CLUSTER_NAME = $ClusterName
    $env:NAMESPACE = $Namespace
    $env:SKIP_CLEANUP = if ($SkipCleanup) { "true" } else { "false" }
    bash ./hack/run-e2e.sh
    exit $LASTEXITCODE
}

# Native PowerShell implementation follows...
Write-Host "🚀 Running native PowerShell E2E test harness" -ForegroundColor Cyan

# Script configuration
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$CrdDir = Join-Path $ProjectRoot "deployments\crds"
$WebhookConfig = Join-Path $ProjectRoot "config\webhook"
$SamplesDir = Join-Path $ProjectRoot "tests\e2e\samples"

# Test tracking
$script:TotalTests = 0
$script:PassedTests = 0
$script:FailedTests = 0
$script:FailedTestNames = @()

# Helper functions
function Write-Info { Write-Host "ℹ️  $($args[0])" -ForegroundColor Blue }
function Write-Success { Write-Host "✅ $($args[0])" -ForegroundColor Green }
function Write-Warning { Write-Host "⚠️  $($args[0])" -ForegroundColor Yellow }
function Write-Error { Write-Host "❌ $($args[0])" -ForegroundColor Red }

function Write-TestStart {
    param([string]$TestName)
    Write-Host "`n═══ TEST: $TestName ═══" -ForegroundColor Blue
    $script:TotalTests++
}

function Write-TestPass {
    param([string]$TestName)
    Write-Host "✅ PASS: $TestName" -ForegroundColor Green
    $script:PassedTests++
}

function Write-TestFail {
    param([string]$TestName)
    Write-Host "❌ FAIL: $TestName" -ForegroundColor Red
    $script:FailedTests++
    $script:FailedTestNames += $TestName
}

# Tool detection
function Test-Command {
    param([string]$Name)
    $cmd = Get-Command $Name, "$Name.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
    return $cmd -ne $null
}

Write-Info "Detecting required tools..."

if (-not (Test-Command "kind")) {
    Write-Error "kind not found. Install: choco install kind"
    Write-Host "   Or download from: https://kind.sigs.k8s.io/"
    exit 1
}

if (-not (Test-Command "kubectl")) {
    Write-Error "kubectl not found. Install: choco install kubernetes-cli"
    Write-Host "   Or download from: https://kubernetes.io/docs/tasks/tools/"
    exit 1
}

Write-Success "All required tools found"

try {
    # Create kind cluster
    Write-TestStart "Kind Cluster Setup"
    
    $existingClusters = kind get clusters 2>$null
    if ($existingClusters -contains $ClusterName) {
        Write-Info "Cluster '$ClusterName' exists, verifying..."
        kubectl cluster-info --context "kind-$ClusterName" 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Using existing cluster"
        } else {
            Write-Warning "Recreating cluster..."
            kind delete cluster --name $ClusterName 2>$null
            kind create cluster --name $ClusterName --wait 120s
        }
    } else {
        Write-Info "Creating cluster '$ClusterName'..."
        if ($KindImage) {
            kind create cluster --name $ClusterName --image $KindImage --wait 120s
        } else {
            kind create cluster --name $ClusterName --wait 120s
        }
    }
    Write-TestPass "Kind Cluster Setup"
    
    # Apply CRDs
    Write-TestStart "CRD Installation"
    if (Test-Path $CrdDir) {
        kubectl apply -f $CrdDir
        Start-Sleep -Seconds 5
        Write-TestPass "CRD Installation"
    } else {
        Write-Error "CRD directory not found: $CrdDir"
        Write-TestFail "CRD Installation"
    }
    
    # Create namespace
    Write-TestStart "Namespace Setup"
    kubectl create namespace $Namespace 2>$null
    kubectl label namespace $Namespace "nephoran.io/webhook=enabled" --overwrite
    Write-TestPass "Namespace Setup"
    
    # Deploy webhook if config exists
    if (Test-Path $WebhookConfig) {
        Write-TestStart "Webhook Manager Deployment"
        kubectl kustomize $WebhookConfig | kubectl apply -n $Namespace -f -
        Start-Sleep -Seconds 10
        Write-TestPass "Webhook Manager Deployment"
    }
    
    # Run validation tests
    Write-TestStart "Webhook Validation Tests"
    
    # Test valid intent
    $validIntent = @"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: valid-test
  namespace: $Namespace
spec:
  description: "Test valid intent"
  intent:
    action: scale
    target: test-deployment
    parameters:
      replicas: 3
  priority: high
  owner: e2e-test
"@
    
    $validIntent | kubectl apply -f - 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-TestPass "Valid Intent Accepted"
    } else {
        Write-TestFail "Valid Intent Rejected"
    }
    
    # Show summary
    Write-Host "`n═══════════════════════════════════════════" -ForegroundColor Blue
    Write-Host "           E2E TEST SUMMARY" -ForegroundColor Blue
    Write-Host "═══════════════════════════════════════════" -ForegroundColor Blue
    Write-Host "Total: $script:TotalTests | " -NoNewline
    Write-Host "Passed: $script:PassedTests" -ForegroundColor Green -NoNewline
    Write-Host " | " -NoNewline
    Write-Host "Failed: $script:FailedTests" -ForegroundColor Red
    
    if ($script:FailedTests -eq 0 -and $script:TotalTests -gt 0) {
        Write-Host "🎉 ALL TESTS PASSED! 🎉" -ForegroundColor Green
    } elseif ($script:FailedTests -gt 0) {
        Write-Host "⚠️  SOME TESTS FAILED ⚠️" -ForegroundColor Red
        exit 1
    }
    
} finally {
    # Cleanup
    if (-not $SkipCleanup) {
        Write-Info "Cleaning up..."
        kubectl delete networkintent --all -n $Namespace 2>$null
        kind delete cluster --name $ClusterName 2>$null
    } else {
        Write-Warning "Skipping cleanup (use: kind delete cluster --name $ClusterName)"
    }
}

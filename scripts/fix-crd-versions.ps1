# Fix CRD Version Issues for Nephoran Intent Operator
# This script resolves CRD version conflicts and ensures clean deployment

param(
    [switch]$Force,
    [switch]$Backup,
    [switch]$Help
)

if ($Help) {
    Write-Host @"
CRD Version Fix Script

DESCRIPTION:
  Fixes CRD version conflicts by cleaning up existing CRDs and redeploying
  them with consistent versioning.

USAGE:
  .\fix-crd-versions.ps1 [options]

OPTIONS:
  -Force    Force deletion of existing CRDs without confirmation
  -Backup   Create backup of existing CRD data before deletion
  -Help     Show this help message

EXAMPLES:
  .\fix-crd-versions.ps1                    # Interactive fix
  .\fix-crd-versions.ps1 -Force             # Automatic fix
  .\fix-crd-versions.ps1 -Backup -Force     # Fix with backup

"@
    exit 0
}

# Color functions
function Write-Step($Message) { Write-Host "🔧 $Message" -ForegroundColor Cyan }
function Write-Success($Message) { Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Warning($Message) { Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error($Message) { Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Info($Message) { Write-Host "ℹ️  $Message" -ForegroundColor Blue }

$NEPHORAN_CRDS = @(
    "networkintents.nephoran.com",
    "e2nodesets.nephoran.com", 
    "managedelements.nephoran.com"
)

function Test-KubernetesConnection {
    Write-Step "Testing Kubernetes connection..."
    
    try {
        kubectl cluster-info 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Connected to Kubernetes cluster"
            return $true
        }
    } catch {}
    
    Write-Error "Cannot connect to Kubernetes cluster"
    Write-Info "Please ensure your cluster is running and kubectl is configured"
    return $false
}

function Backup-ExistingCRDs {
    if (!$Backup) {
        return
    }
    
    Write-Step "Creating backup of existing CRD data..."
    
    $backupDir = "crd-backup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    New-Item -ItemType Directory $backupDir -Force | Out-Null
    
    foreach ($crd in $NEPHORAN_CRDS) {
        Write-Info "Backing up $crd..."
        
        # Backup CRD definition
        kubectl get crd $crd -o yaml > "$backupDir\$crd-definition.yaml" 2>$null
        
        # Backup CRD instances
        $resourceType = $crd.Split('.')[0]  # e.g., "networkintents" from "networkintents.nephoran.com"
        kubectl get $resourceType -A -o yaml > "$backupDir\$resourceType-instances.yaml" 2>$null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Backed up $crd"
        } else {
            Write-Warning "Could not backup $crd (may not exist)"
        }
    }
    
    Write-Success "Backup completed in: $backupDir"
}

function Remove-ExistingCRDs {
    Write-Step "Checking for existing CRDs..."
    
    $existingCrds = @()
    foreach ($crd in $NEPHORAN_CRDS) {
        kubectl get crd $crd 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $existingCrds += $crd
        }
    }
    
    if ($existingCrds.Count -eq 0) {
        Write-Info "No existing Nephoran CRDs found"
        return
    }
    
    Write-Warning "Found existing CRDs: $($existingCrds -join ', ')"
    
    if (!$Force) {
        $confirmation = Read-Host "Delete existing CRDs? This will remove all related resources (y/N)"
        if ($confirmation -notlike "y*") {
            Write-Info "Cancelled by user"
            exit 0
        }
    }
    
    foreach ($crd in $existingCrds) {
        Write-Step "Deleting CRD: $crd"
        
        # First delete all instances of the CRD
        $resourceType = $crd.Split('.')[0]
        Write-Info "Deleting all $resourceType instances..."
        kubectl delete $resourceType --all -A 2>$null
        
        # Then delete the CRD itself
        kubectl delete crd $crd --timeout=60s
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Deleted $crd"
        } else {
            Write-Error "Failed to delete $crd"
            
            # Try force deletion
            Write-Step "Attempting force deletion..."
            kubectl patch crd $crd -p '{"metadata":{"finalizers":[]}}' --type=merge 2>$null
            kubectl delete crd $crd --force --grace-period=0 2>$null
        }
    }
    
    # Wait for CRDs to be fully removed
    Write-Step "Waiting for CRDs to be fully removed..."
    $maxWait = 60
    $waited = 0
    
    while ($waited -lt $maxWait) {
        $remaining = @()
        foreach ($crd in $existingCrds) {
            kubectl get crd $crd 2>$null | Out-Null
            if ($LASTEXITCODE -eq 0) {
                $remaining += $crd
            }
        }
        
        if ($remaining.Count -eq 0) {
            Write-Success "All CRDs removed successfully"
            break
        }
        
        Write-Info "Still waiting for: $($remaining -join ', ') (${waited}s/${maxWait}s)"
        Start-Sleep -Seconds 5
        $waited += 5
    }
    
    if ($waited -ge $maxWait) {
        Write-Warning "Some CRDs may still be terminating. Continuing anyway..."
    }
}

function Deploy-CleanCRDs {
    Write-Step "Deploying clean CRDs..."
    
    $crdPath = "deployments\crds"
    if (!(Test-Path $crdPath)) {
        Write-Error "CRD directory not found: $crdPath"
        Write-Info "Please run this script from the project root directory"
        exit 1
    }
    
    # Apply CRDs one by one to get better error reporting
    $crdFiles = Get-ChildItem $crdPath -Filter "*.yaml" | Where-Object { $_.Name -like "*nephoran.com*" }
    
    foreach ($crdFile in $crdFiles) {
        Write-Step "Applying $($crdFile.Name)..."
        
        kubectl apply -f $crdFile.FullName
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Applied $($crdFile.Name)"
        } else {
            Write-Error "Failed to apply $($crdFile.Name)"
            exit 1
        }
    }
    
    # Wait for CRDs to be established
    Write-Step "Waiting for CRDs to be established..."
    Start-Sleep -Seconds 5
    
    foreach ($crd in $NEPHORAN_CRDS) {
        $maxWait = 60
        $waited = 0
        
        while ($waited -lt $maxWait) {
            $status = kubectl get crd $crd -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>$null
            if ($status -eq "True") {
                Write-Success "CRD $crd is established"
                break
            }
            
            Start-Sleep -Seconds 2
            $waited += 2
        }
        
        if ($waited -ge $maxWait) {
            Write-Warning "CRD $crd may not be fully established"
        }
    }
}

function Verify-CRDDeployment {
    Write-Step "Verifying CRD deployment..."
    
    $allGood = $true
    
    foreach ($crd in $NEPHORAN_CRDS) {
        # Check if CRD exists
        kubectl get crd $crd 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Error "CRD $crd not found"
            $allGood = $false
            continue
        }
        
        # Check if established
        $established = kubectl get crd $crd -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>$null
        if ($established -ne "True") {
            Write-Warning "CRD $crd exists but not established"
            $allGood = $false
            continue
        }
        
        # Check version
        $version = kubectl get crd $crd -o jsonpath='{.spec.versions[0].name}' 2>$null
        Write-Success "CRD $crd is ready (version: $version)"
    }
    
    if ($allGood) {
        Write-Success "All CRDs are properly deployed and established"
    } else {
        Write-Error "Some CRDs have issues"
        exit 1
    }
}

function Test-CRDFunctionality {
    Write-Step "Testing CRD functionality..."
    
    # Create a test NetworkIntent
    $testIntent = @"
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: crd-test-intent
  namespace: default
spec:
  intent: "Test intent for CRD validation"
  parameters: {}
"@
    
    Write-Info "Creating test NetworkIntent..."
    $testIntent | kubectl apply -f - 2>$null
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Test NetworkIntent created successfully"
        
        # Clean up test resource
        kubectl delete networkintent crd-test-intent 2>$null
        Write-Info "Test resource cleaned up"
    } else {
        Write-Warning "Failed to create test NetworkIntent"
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - CRD Version Fix" -ForegroundColor Blue
Write-Host "=========================================" -ForegroundColor Blue

try {
    if (!(Test-KubernetesConnection)) {
        exit 1
    }
    
    Backup-ExistingCRDs
    Remove-ExistingCRDs
    Deploy-CleanCRDs
    Verify-CRDDeployment
    Test-CRDFunctionality
    
    Write-Host "`n🎉 CRD version issues fixed successfully!" -ForegroundColor Green
    Write-Host "You can now continue with your deployment:" -ForegroundColor Blue
    Write-Host "  .\local-deploy.ps1 -UseLocalRegistry" -ForegroundColor White
    
} catch {
    Write-Error "CRD fix failed: $($_.Exception.Message)"
    exit 1
}
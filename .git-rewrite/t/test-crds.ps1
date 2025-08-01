# CRD Testing and Validation Script for Nephoran Intent Operator
# Comprehensive testing of Custom Resource Definitions and controllers

param(
    [string]$Namespace = "nephoran-system",
    [switch]$CleanupAfter,
    [switch]$Verbose,
    [switch]$WaitForControllers,
    [int]$TimeoutSeconds = 300,
    [switch]$Help
)

if ($Help) {
    Write-Host @"
Nephoran Intent Operator - CRD Testing Script

DESCRIPTION:
  Tests Custom Resource Definitions (CRDs) and controller functionality
  by creating sample resources and validating their behavior.

USAGE:
  .\test-crds.ps1 [options]

OPTIONS:
  -Namespace         Target namespace (default: nephoran-system)
  -CleanupAfter      Remove test resources after completion
  -Verbose           Enable verbose output and detailed logging
  -WaitForControllers Wait for controllers to process resources
  -TimeoutSeconds    Timeout for controller operations (default: 300)
  -Help              Show this help message

EXAMPLES:
  .\test-crds.ps1                                    # Basic CRD testing
  .\test-crds.ps1 -Verbose -CleanupAfter             # Full test with cleanup
  .\test-crds.ps1 -WaitForControllers -TimeoutSeconds 600  # Extended timeout

PREREQUISITES:
  - Kubernetes cluster with CRDs deployed
  - kubectl configured and accessible
  - Controllers deployed and running (optional for CRD validation)

"@
    exit 0
}

# Test resource definitions
$TEST_RESOURCES = @{
    "NetworkIntent" = @{
        "apiVersion" = "nephoran.com/v1alpha1"
        "kind" = "NetworkIntent"
        "metadata" = @{
            "name" = "test-network-intent"
            "namespace" = $Namespace
        }
        "spec" = @{
            "intent" = "Deploy 5G network slice with high throughput for video streaming"
            "parameters" = @{}
        }
    }
    "E2NodeSet" = @{
        "apiVersion" = "nephoran.com/v1alpha1"
        "kind" = "E2NodeSet"
        "metadata" = @{
            "name" = "test-e2nodeset"
            "namespace" = $Namespace
        }
        "spec" = @{
            "replicas" = 3
            "template" = @{
                "metadata" = @{
                    "labels" = @{
                        "app" = "e2-node"
                        "version" = "v1.0.0"
                    }
                }
                "spec" = @{
                    "nodeType" = "gNB"
                    "configuration" = @{
                        "plmnId" = "00101"
                        "nrCellId" = "12345"
                    }
                }
            }
        }
    }
    "ManagedElement" = @{
        "apiVersion" = "nephoran.com/v1alpha1"
        "kind" = "ManagedElement"
        "metadata" = @{
            "name" = "test-managed-element"
            "namespace" = $Namespace
        }
        "spec" = @{
            "elementType" = "O-RAN-DU"
            "managementEndpoint" = "http://oran-du.nephoran.local:8080"
            "configuration" = @{
                "vendor" = "TestVendor"
                "version" = "1.0.0"
                "capabilities" = @("5G-NR", "O-RAN-7.2")
            }
        }
    }
}

# Color functions
function Write-Step($Message) { Write-Host "🔧 $Message" -ForegroundColor Cyan }
function Write-Success($Message) { Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Warning($Message) { Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error($Message) { Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Info($Message) { Write-Host "ℹ️  $Message" -ForegroundColor Blue }
function Write-Test($Message) { Write-Host "🧪 $Message" -ForegroundColor Magenta }

function Test-Prerequisites {
    Write-Step "Checking prerequisites..."
    
    # Test kubectl
    if (!(Get-Command kubectl -ErrorAction SilentlyContinue)) {
        Write-Error "kubectl not found. Please install kubectl."
        exit 1
    }
    
    # Test cluster connectivity
    try {
        kubectl cluster-info 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Cluster not accessible"
        }
    } catch {
        Write-Error "Cannot connect to Kubernetes cluster."
        exit 1
    }
    
    # Test namespace
    kubectl get namespace $Namespace 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Namespace '$Namespace' does not exist. Creating it..."
        kubectl create namespace $Namespace
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to create namespace '$Namespace'"
            exit 1
        }
    }
    
    Write-Success "Prerequisites check passed"
}

function Test-CRDExists {
    param([string]$CrdName)
    
    kubectl get crd $CrdName 2>$null | Out-Null
    return $LASTEXITCODE -eq 0
}

function Test-CRDEstablished {
    param([string]$CrdName)
    
    $status = kubectl get crd $CrdName -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' 2>$null
    return $status -eq "True"
}

function Get-CRDVersion {
    param([string]$CrdName)
    
    return kubectl get crd $CrdName -o jsonpath='{.spec.versions[0].name}' 2>$null
}

function Test-AllCRDs {
    Write-Step "Testing CRD registration and status..."
    
    $crdMapping = @{
        "NetworkIntent" = "networkintents.nephoran.com"
        "E2NodeSet" = "e2nodesets.nephoran.com"
        "ManagedElement" = "managedelements.nephoran.com"
    }
    
    $crdResults = @{}
    
    foreach ($resourceType in $crdMapping.Keys) {
        $crdName = $crdMapping[$resourceType]
        Write-Test "Testing CRD: $crdName"
        
        $result = @{
            "Exists" = Test-CRDExists $crdName
            "Established" = $false
            "Version" = ""
        }
        
        if ($result.Exists) {
            Write-Success "  ✓ CRD exists"
            $result.Established = Test-CRDEstablished $crdName
            $result.Version = Get-CRDVersion $crdName
            
            if ($result.Established) {
                Write-Success "  ✓ CRD is established"
                Write-Info "  ℹ️ Version: $($result.Version)"
            } else {
                Write-Warning "  ⚠️ CRD exists but is not established"
            }
        } else {
            Write-Error "  ❌ CRD does not exist"
        }
        
        $crdResults[$resourceType] = $result
    }
    
    return $crdResults
}

function ConvertTo-Yaml {
    param([hashtable]$Object)
    
    # Simple hashtable to YAML conversion
    # For more complex scenarios, consider using a proper YAML library
    function Convert-Level {
        param([hashtable]$Hash, [int]$Indent = 0)
        
        $yaml = ""
        $spaces = " " * $Indent
        
        foreach ($key in $Hash.Keys) {
            $value = $Hash[$key]
            
            if ($value -is [hashtable]) {
                $yaml += "${spaces}${key}:`n"
                $yaml += Convert-Level $value ($Indent + 2)
            } elseif ($value -is [array]) {
                $yaml += "${spaces}${key}:`n"
                foreach ($item in $value) {
                    if ($item -is [hashtable]) {
                        $yaml += "${spaces}- `n"
                        $yaml += Convert-Level $item ($Indent + 4)
                    } else {
                        $yaml += "${spaces}- $item`n"
                    }
                }
            } else {
                $yaml += "${spaces}${key}: $value`n"
            }
        }
        
        return $yaml
    }
    
    return Convert-Level $Object
}

function Create-TestResource {
    param([string]$ResourceType, [hashtable]$ResourceDef)
    
    Write-Test "Creating test $ResourceType..."
    
    # Convert to YAML and create temp file
    $yaml = ConvertTo-Yaml $ResourceDef
    $tempFile = "temp-$($ResourceDef.metadata.name).yaml"
    
    try {
        $yaml | Out-File -FilePath $tempFile -Encoding UTF8
        
        # Apply the resource
        kubectl apply -f $tempFile -n $Namespace
        if ($LASTEXITCODE -eq 0) {
            Write-Success "  ✓ $ResourceType created successfully"
            return $true
        } else {
            Write-Error "  ❌ Failed to create $ResourceType"
            return $false
        }
    } finally {
        # Cleanup temp file
        if (Test-Path $tempFile) {
            Remove-Item $tempFile
        }
    }
}

function Get-ResourceStatus {
    param([string]$ResourceType, [string]$ResourceName)
    
    $kind = $ResourceType.ToLower() + "s"  # Pluralize
    
    try {
        $resource = kubectl get $kind $ResourceName -n $Namespace -o json 2>$null | ConvertFrom-Json
        if ($LASTEXITCODE -eq 0) {
            return @{
                "Exists" = $true
                "Resource" = $resource
                "Status" = $resource.status
                "Phase" = $resource.status.phase
                "Conditions" = $resource.status.conditions
            }
        }
    } catch {}
    
    return @{
        "Exists" = $false
        "Resource" = $null
        "Status" = $null
        "Phase" = "Unknown"
        "Conditions" = @()
    }
}

function Wait-ForController {
    param([string]$ResourceType, [string]$ResourceName, [int]$TimeoutSec)
    
    Write-Step "Waiting for controller to process $ResourceType/$ResourceName..."
    
    $elapsed = 0
    $interval = 10
    
    while ($elapsed -lt $TimeoutSec) {
        $status = Get-ResourceStatus $ResourceType $ResourceName
        
        if ($status.Exists) {
            if ($Verbose) {
                Write-Info "  Current status: $($status.Phase)"
                if ($status.Conditions) {
                    foreach ($condition in $status.Conditions) {
                        Write-Info "    Condition: $($condition.type) = $($condition.status)"
                    }
                }
            }
            
            # Check for completion indicators
            if ($status.Status) {
                $observedGeneration = $status.Status.observedGeneration
                $generation = $status.Resource.metadata.generation
                
                if ($observedGeneration -eq $generation) {
                    Write-Success "  ✓ Controller has processed the resource"
                    return $true
                }
            }
        }
        
        Start-Sleep -Seconds $interval
        $elapsed += $interval
        Write-Info "  Waiting... (${elapsed}s/${TimeoutSec}s)"
    }
    
    Write-Warning "  ⚠️ Timeout waiting for controller (${TimeoutSec}s)"
    return $false
}

function Test-ResourceCreation {
    param([hashtable]$CrdResults)
    
    Write-Step "Testing resource creation..."
    
    $creationResults = @{}
    
    foreach ($resourceType in $TEST_RESOURCES.Keys) {
        $resourceDef = $TEST_RESOURCES[$resourceType]
        
        # Skip if CRD is not available
        if (!$CrdResults[$resourceType].Established) {
            Write-Warning "Skipping $resourceType test - CRD not established"
            $creationResults[$resourceType] = @{
                "Created" = $false
                "Reason" = "CRD not established"
            }
            continue
        }
        
        $created = Create-TestResource $resourceType $resourceDef
        $result = @{
            "Created" = $created
            "Reason" = if ($created) { "Success" } else { "Creation failed" }
        }
        
        if ($created -and $WaitForControllers) {
            $processed = Wait-ForController $resourceType $resourceDef.metadata.name $TimeoutSeconds
            $result["ControllerProcessed"] = $processed
        }
        
        $creationResults[$resourceType] = $result
    }
    
    return $creationResults
}

function Test-ResourceValidation {
    Write-Step "Testing resource validation and schema enforcement..."
    
    # Test invalid NetworkIntent (missing required fields)
    $invalidIntent = @{
        "apiVersion" = "nephoran.com/v1alpha1"
        "kind" = "NetworkIntent"
        "metadata" = @{
            "name" = "invalid-intent"
            "namespace" = $Namespace
        }
        "spec" = @{
            # Missing required 'intent' field
            "parameters" = @{}
        }
    }
    
    Write-Test "Testing schema validation with invalid resource..."
    
    $yaml = ConvertTo-Yaml $invalidIntent
    $tempFile = "temp-invalid-intent.yaml"
    
    try {
        $yaml | Out-File -FilePath $tempFile -Encoding UTF8
        
        # This should fail validation
        kubectl apply -f $tempFile -n $Namespace 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Success "  ✓ Schema validation correctly rejected invalid resource"
            return $true
        } else {
            Write-Warning "  ⚠️ Schema validation did not reject invalid resource"
            # Clean up if it was somehow created
            kubectl delete -f $tempFile -n $Namespace 2>$null
            return $false
        }
    } finally {
        if (Test-Path $tempFile) {
            Remove-Item $tempFile
        }
    }
}

function Get-ControllerLogs {
    param([string]$ControllerName)
    
    try {
        $logs = kubectl logs deployment/$ControllerName -n $Namespace --tail=50 2>$null
        return $logs
    } catch {
        return "Unable to retrieve logs"
    }
}

function Test-ControllerBehavior {
    param([hashtable]$CreationResults)
    
    if (!$WaitForControllers) {
        Write-Info "Skipping controller behavior tests (use -WaitForControllers to enable)"
        return
    }
    
    Write-Step "Testing controller behavior..."
    
    # Check if controllers are running
    $controllers = @("nephio-bridge", "llm-processor", "oran-adaptor")
    
    foreach ($controller in $controllers) {
        Write-Test "Checking controller: $controller"
        
        $pods = kubectl get pods -n $Namespace -l app=$controller -o jsonpath='{.items[*].metadata.name}' 2>$null
        if ($pods) {
            $podList = $pods.Split(" ") | Where-Object { $_ -ne "" }
            $runningPods = 0
            
            foreach ($pod in $podList) {
                $status = kubectl get pod $pod -n $Namespace -o jsonpath='{.status.phase}' 2>$null
                if ($status -eq "Running") {
                    $runningPods++
                }
            }
            
            if ($runningPods -gt 0) {
                Write-Success "  ✓ $controller is running ($runningPods pods)"
                
                if ($Verbose) {
                    Write-Info "  Recent logs:"
                    $logs = Get-ControllerLogs $controller
                    $logs.Split("`n") | Select-Object -Last 10 | ForEach-Object {
                        Write-Info "    $_"
                    }
                }
            } else {
                Write-Warning "  ⚠️ $controller has no running pods"
            }
        } else {
            Write-Warning "  ⚠️ $controller pods not found"
        }
    }
}

function Cleanup-TestResources {
    if (!$CleanupAfter) {
        Write-Info "Skipping cleanup (use -CleanupAfter to enable)"
        return
    }
    
    Write-Step "Cleaning up test resources..."
    
    foreach ($resourceType in $TEST_RESOURCES.Keys) {
        $resourceDef = $TEST_RESOURCES[$resourceType]
        $resourceName = $resourceDef.metadata.name
        $kind = $resourceType.ToLower() + "s"  # Pluralize
        
        Write-Info "Deleting $resourceType/$resourceName..."
        kubectl delete $kind $resourceName -n $Namespace 2>$null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "  ✓ Deleted $resourceType/$resourceName"
        } else {
            Write-Warning "  ⚠️ Failed to delete $resourceType/$resourceName (may not exist)"
        }
    }
    
    Write-Success "Cleanup completed"
}

function Generate-TestReport {
    param([hashtable]$CrdResults, [hashtable]$CreationResults)
    
    Write-Host "`n📊 Test Report" -ForegroundColor Blue
    Write-Host "==============" -ForegroundColor Blue
    
    Write-Host "`nCRD Status:" -ForegroundColor Cyan
    foreach ($resourceType in $CrdResults.Keys) {
        $result = $CrdResults[$resourceType]
        $status = if ($result.Established) { "✅ OK" } elseif ($result.Exists) { "⚠️ Not Established" } else { "❌ Missing" }
        Write-Host "  $resourceType`: $status" -ForegroundColor White
    }
    
    Write-Host "`nResource Creation:" -ForegroundColor Cyan
    foreach ($resourceType in $CreationResults.Keys) {
        $result = $CreationResults[$resourceType]
        $status = if ($result.Created) { "✅ Success" } else { "❌ Failed" }
        Write-Host "  $resourceType`: $status" -ForegroundColor White
        
        if ($result.ContainsKey("ControllerProcessed")) {
            $controllerStatus = if ($result.ControllerProcessed) { "✅ Processed" } else { "⚠️ Timeout" }
            Write-Host "    Controller: $controllerStatus" -ForegroundColor Gray
        }
    }
    
    # Overall assessment
    $crdOk = ($CrdResults.Values | Where-Object { $_.Established }).Count
    $crdTotal = $CrdResults.Count
    $resourceOk = ($CreationResults.Values | Where-Object { $_.Created }).Count
    $resourceTotal = $CreationResults.Count
    
    Write-Host "`nSummary:" -ForegroundColor Cyan
    Write-Host "  CRDs: $crdOk/$crdTotal established" -ForegroundColor White
    Write-Host "  Resources: $resourceOk/$resourceTotal created successfully" -ForegroundColor White
    
    if ($crdOk -eq $crdTotal -and $resourceOk -eq $resourceTotal) {
        Write-Host "`n🎉 All tests passed!" -ForegroundColor Green
    } elseif ($crdOk -eq $crdTotal) {
        Write-Host "`n⚠️  CRDs are working, but some resource creation failed" -ForegroundColor Yellow
    } else {
        Write-Host "`n❌ Some CRDs are not properly established" -ForegroundColor Red
    }
}

# Main execution
Write-Host "Nephoran Intent Operator - CRD Testing" -ForegroundColor Blue
Write-Host "Namespace: $Namespace" -ForegroundColor Blue
Write-Host "Wait for Controllers: $($WaitForControllers ? 'Yes' : 'No')" -ForegroundColor Blue
Write-Host "=====================================" -ForegroundColor Blue

try {
    Test-Prerequisites
    
    $crdResults = Test-AllCRDs
    $creationResults = Test-ResourceCreation $crdResults
    
    Test-ResourceValidation
    Test-ControllerBehavior $creationResults
    
    Generate-TestReport $crdResults $creationResults
    
    Cleanup-TestResources
    
    Write-Host "`n✅ CRD testing completed" -ForegroundColor Green
    
} catch {
    Write-Error "Testing failed: $($_.Exception.Message)"
    if ($Verbose) {
        Write-Info "Stack trace: $($_.ScriptStackTrace)"
    }
    exit 1
}
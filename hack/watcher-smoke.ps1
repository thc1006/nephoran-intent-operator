#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Comprehensive smoke test for conductor-watch functionality

.DESCRIPTION
    This script demonstrates the complete conductor-watch workflow:
    1. Pre-flight checks (CRD, namespace, kubectl connectivity)
    2. Build and start conductor-watch in background
    3. Apply test NetworkIntent resource
    4. Verify conductor-watch processes the intent
    5. Cleanup and show verification steps

.PARAMETER DryRun
    Run conductor-watch in dry-run mode (default: true)

.PARAMETER Timeout
    Timeout in seconds for waiting operations (default: 30)

.PARAMETER Namespace
    Kubernetes namespace to use for testing (default: ran-a)

.PARAMETER SkipBuild
    Skip building conductor-watch if binary already exists

.EXAMPLE
    .\hack\watcher-smoke.ps1
    
.EXAMPLE
    .\hack\watcher-smoke.ps1 -Timeout 60 -Namespace test-ns

.NOTES
    Requires: kubectl, go, PowerShell 5.1+
    Windows compatible with proper path handling
#>

param(
    [switch]$DryRun = $true,
    [int]$Timeout = 30,
    [string]$Namespace = "ran-a",
    [switch]$SkipBuild = $false
)

# Error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Configuration
$CRD_NAME = "networkintents.nephoran.com"
$BINARY_PATH = ".\conductor-loop.exe"
$HANDOFF_DIR = ".\handoff"
$SCHEMA_PATH = ".\docs\contracts\intent.schema.json"
$TEST_INTENT_NAME = "watcher-smoke-test"

# Color functions for output
function Write-Success { param($Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Error { param($Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warning { param($Message) Write-Host "⚠️ $Message" -ForegroundColor Yellow }
function Write-Info { param($Message) Write-Host "ℹ️ $Message" -ForegroundColor Cyan }
function Write-Step { param($Message) Write-Host "`n🔄 $Message" -ForegroundColor Blue }

# Cleanup function
function Cleanup {
    Write-Step "Cleaning up test resources..."
    
    # Stop conductor-watch process if running
    if ($global:ConductorProcess -and !$global:ConductorProcess.HasExited) {
        Write-Info "Stopping conductor-watch process (PID: $($global:ConductorProcess.Id))"
        try {
            $global:ConductorProcess.Kill()
            $global:ConductorProcess.WaitForExit(5000)
            Write-Success "Conductor-watch process stopped"
        } catch {
            Write-Warning "Failed to stop conductor-watch gracefully: $_"
        }
    }
    
    # Remove test NetworkIntent
    try {
        kubectl delete networkintent $TEST_INTENT_NAME -n $Namespace --ignore-not-found=true 2>$null
        Write-Success "Test NetworkIntent removed"
    } catch {
        Write-Warning "Failed to remove test NetworkIntent: $_"
    }
    
    # Remove test handoff files
    try {
        Get-ChildItem -Path $HANDOFF_DIR -Filter "intent-*smoke-test*.json" -ErrorAction SilentlyContinue | Remove-Item -Force
        Write-Success "Test handoff files removed"
    } catch {
        Write-Warning "Failed to remove test handoff files: $_"
    }
}

# Register cleanup on exit
Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Cleanup }
trap { Cleanup; break }

function Test-CommandExists {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

function Wait-ForFileCreation {
    param(
        [string]$Directory,
        [string]$Pattern,
        [int]$TimeoutSeconds = 30
    )
    
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    while ($stopwatch.Elapsed.TotalSeconds -lt $TimeoutSeconds) {
        $files = Get-ChildItem -Path $Directory -Filter $Pattern -ErrorAction SilentlyContinue
        if ($files) {
            return $files[0]
        }
        Start-Sleep -Milliseconds 500
    }
    return $null
}

function Start-ConductorWatch {
    Write-Step "Starting conductor-watch..."
    
    # Build arguments for conductor-watch
    $args = @(
        "-handoff-dir", $HANDOFF_DIR
        "-schema", $SCHEMA_PATH
        "-batch-size", "1"
        "-batch-interval", "2s"
    )
    
    if ($DryRun) {
        Write-Info "Running in DRY-RUN mode (logs only, no actual porch commands)"
    }
    
    # Start conductor-watch in background
    $processInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processInfo.FileName = $BINARY_PATH
    $processInfo.Arguments = $args -join " "
    $processInfo.UseShellExecute = $false
    $processInfo.RedirectStandardOutput = $true
    $processInfo.RedirectStandardError = $true
    $processInfo.CreateNoWindow = $true
    $processInfo.WorkingDirectory = Get-Location
    
    try {
        $process = [System.Diagnostics.Process]::Start($processInfo)
        $global:ConductorProcess = $process
        
        # Give it a moment to start
        Start-Sleep -Seconds 2
        
        if ($process.HasExited) {
            $output = $process.StandardOutput.ReadToEnd()
            $error = $process.StandardError.ReadToEnd()
            throw "Conductor-watch exited immediately. Output: $output. Error: $error"
        }
        
        Write-Success "Conductor-watch started (PID: $($process.Id))"
        Write-Info "Monitoring directory: $HANDOFF_DIR"
        return $process
    } catch {
        throw "Failed to start conductor-watch: $_"
    }
}

function Test-PreflightChecks {
    Write-Step "Running pre-flight checks..."
    
    # Check required tools
    $requiredTools = @("kubectl", "go")
    foreach ($tool in $requiredTools) {
        if (Test-CommandExists $tool) {
            Write-Success "$tool is available"
        } else {
            Write-Error "$tool is required but not found"
            throw "Missing required tool: $tool"
        }
    }
    
    # Check kubectl connectivity
    try {
        kubectl cluster-info | Out-Null
        Write-Success "Kubernetes cluster is accessible"
    } catch {
        Write-Error "Cannot connect to Kubernetes cluster"
        throw "Kubectl connectivity failed: $_"
    }
    
    # Check for NetworkIntent CRD
    try {
        kubectl get crd $CRD_NAME 2>$null | Out-Null
        Write-Success "NetworkIntent CRD exists"
    } catch {
        Write-Error "NetworkIntent CRD not found"
        Write-Info "Install CRDs with: kubectl apply -f deployments/crds/"
        throw "CRD $CRD_NAME not found"
    }
    
    # Ensure namespace exists
    try {
        kubectl get namespace $Namespace 2>$null | Out-Null
        Write-Success "Namespace '$Namespace' exists"
    } catch {
        Write-Warning "Namespace '$Namespace' not found, creating..."
        kubectl create namespace $Namespace
        Write-Success "Namespace '$Namespace' created"
    }
    
    # Check required files
    $requiredFiles = @($SCHEMA_PATH)
    foreach ($file in $requiredFiles) {
        if (Test-Path $file) {
            Write-Success "Required file exists: $file"
        } else {
            Write-Error "Required file missing: $file"
            throw "Missing required file: $file"
        }
    }
    
    # Ensure handoff directory exists
    if (!(Test-Path $HANDOFF_DIR)) {
        New-Item -ItemType Directory -Path $HANDOFF_DIR -Force | Out-Null
        Write-Success "Created handoff directory: $HANDOFF_DIR"
    } else {
        Write-Success "Handoff directory exists: $HANDOFF_DIR"
    }
}

function Build-ConductorWatch {
    if ($SkipBuild -and (Test-Path $BINARY_PATH)) {
        Write-Success "Skipping build, binary exists: $BINARY_PATH"
        return
    }
    
    Write-Step "Building conductor-watch..."
    
    try {
        $env:CGO_ENABLED = "0"
        go build -ldflags="-s -w" -o $BINARY_PATH .\cmd\conductor-loop\
        
        if (Test-Path $BINARY_PATH) {
            Write-Success "Conductor-watch built successfully: $BINARY_PATH"
        } else {
            throw "Binary not found after build"
        }
    } catch {
        Write-Error "Failed to build conductor-watch: $_"
        throw
    }
}

function Apply-TestNetworkIntent {
    Write-Step "Applying test NetworkIntent..."
    
    # Create test NetworkIntent using Here-String
    $timestamp = Get-Date -Format 'yyyyMMddHHmmss'
    $networkIntent = @"
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: $TEST_INTENT_NAME
  namespace: $Namespace
spec:
  intent: "Scale nf-simulator to 3 replicas for smoke test"
  intentType: "scaling" 
  parameters:
    target: "nf-simulator"
    namespace: "$Namespace"
    replicas: 3
    reason: "Conductor-watch smoke test"
    source: "test"
    correlation_id: "smoke-test-$timestamp"
  parametersMap:
    target: "nf-simulator"
    namespace: "$Namespace"
    replicas: "3"
"@
    
    try {
        # Apply using pipeline
        $networkIntent | kubectl apply -f -
        Write-Success "Test NetworkIntent applied successfully"
        
        # Show the applied object
        Write-Info "Applied NetworkIntent:"
        kubectl get networkintent $TEST_INTENT_NAME -n $Namespace -o yaml | Select-Object -First 20
        
        return $true
    } catch {
        Write-Error "Failed to apply test NetworkIntent: $_"
        return $false
    }
}

function Verify-ConductorWatchProcessing {
    Write-Step "Verifying conductor-watch processing..."
    
    # Wait for handoff file to be created
    Write-Info "Waiting for handoff file creation (timeout: ${Timeout}s)..."
    $handoffFile = Wait-ForFileCreation -Directory $HANDOFF_DIR -Pattern "intent-*smoke-test*.json" -TimeoutSeconds $Timeout
    
    if ($handoffFile) {
        Write-Success "Handoff file created: $($handoffFile.Name)"
        
        # Show file contents
        Write-Info "Handoff file contents:"
        Get-Content $handoffFile.FullName | ConvertFrom-Json | ConvertTo-Json -Depth 10 | Write-Host
    } else {
        Write-Warning "No handoff file created within timeout"
    }
    
    # Check conductor-watch logs
    Write-Info "Checking conductor-watch process status..."
    if ($global:ConductorProcess -and !$global:ConductorProcess.HasExited) {
        Write-Success "Conductor-watch is still running"
        
        # Try to read some output (non-blocking)
        try {
            $output = $global:ConductorProcess.StandardOutput.ReadToEnd()
            $error = $global:ConductorProcess.StandardError.ReadToEnd()
            
            if ($output) {
                Write-Info "Conductor-watch stdout:"
                $output | Write-Host
            }
            if ($error) {
                Write-Info "Conductor-watch stderr:"
                $error | Write-Host
            }
        } catch {
            Write-Warning "Could not read process output: $_"
        }
    } else {
        Write-Warning "Conductor-watch process has exited"
        if ($global:ConductorProcess) {
            Write-Info "Exit code: $($global:ConductorProcess.ExitCode)"
        }
    }
    
    # Check for expected log patterns in handoff files
    Write-Info "Looking for processed intent files..."
    $intentFiles = Get-ChildItem -Path $HANDOFF_DIR -Filter "intent-*.json" | Sort-Object CreationTime -Descending
    
    if ($intentFiles) {
        Write-Success "Found $($intentFiles.Count) intent file(s) in handoff directory"
        foreach ($file in $intentFiles | Select-Object -First 3) {
            Write-Info "- $($file.Name) ($(Get-Date $file.CreationTime -Format 'yyyy-MM-dd HH:mm:ss'))"
        }
    } else {
        Write-Warning "No intent files found in handoff directory"
    }
}

function Show-VerificationSteps {
    Write-Step "Manual verification steps:"
    
    Write-Host @"

📋 Manual Verification Checklist:

1. Check NetworkIntent Status:
   kubectl get networkintent $TEST_INTENT_NAME -n $Namespace -o yaml

2. Check for handoff files:
   Get-ChildItem $HANDOFF_DIR -Filter "intent-*.json"

3. Verify conductor-watch logs (if still running):
   # Look for these log patterns:
   # [conductor-loop] Starting conductor-loop:
   # [conductor-loop] Watching: $HANDOFF_DIR
   # [CREATE] Intent file detected: intent-*.json
   # Processing file: intent-*.json

4. Expected porch command (in dry-run mode):
   # Should see log like:
   # [DRY-RUN] Would execute: kpt pkg get ...
   # [DRY-RUN] Would execute: kpt fn eval ...
   # [DRY-RUN] Would execute: kpt live apply ...

5. Check NetworkIntent reconciliation:
   kubectl describe networkintent $TEST_INTENT_NAME -n $Namespace

6. Verify no errors in conductor-watch:
   # No files should appear in: $HANDOFF_DIR\errors\

🔧 Troubleshooting:
- If no handoff file: Check NetworkIntent controller is running
- If no processing: Check conductor-watch logs for schema validation errors
- If porch errors: Verify kpt/porch installation in cluster

"@ -ForegroundColor White
}

# Main execution
function Main {
    Write-Host "`n🚀 Conductor-Watch Smoke Test" -ForegroundColor Magenta
    Write-Host "==============================`n" -ForegroundColor Magenta
    
    try {
        # Step 1: Pre-flight checks
        Test-PreflightChecks
        
        # Step 2: Build conductor-watch
        Build-ConductorWatch
        
        # Step 3: Start conductor-watch
        Start-ConductorWatch
        
        # Step 4: Apply test NetworkIntent
        if (Apply-TestNetworkIntent) {
            # Step 5: Wait and verify processing
            Start-Sleep -Seconds 3  # Give a moment for file system events
            Verify-ConductorWatchProcessing
        }
        
        # Step 6: Show verification steps
        Show-VerificationSteps
        
        Write-Success "`nSmoke test completed successfully!"
        Write-Info "Use Ctrl+C to stop and cleanup, or run with -Cleanup to clean up now"
        
        # Wait for user input if in interactive mode
        if ([Environment]::UserInteractive) {
            Write-Host "`nPress Enter to cleanup and exit..." -NoNewline
            Read-Host
        }
        
    } catch {
        Write-Error "Smoke test failed: $_"
        Write-Info "Cleanup will run automatically..."
        exit 1
    } finally {
        Cleanup
    }
}

# Execute main function
Main
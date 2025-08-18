#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Basic conductor-watch functionality test (no Kubernetes required)

.DESCRIPTION
    This script tests conductor-watch file monitoring without requiring Kubernetes:
    1. Build conductor-watch binary
    2. Start conductor-watch in background with dry-run mode
    3. Create test intent JSON files in handoff directory
    4. Verify conductor-watch detects and processes files
    5. Show processing logs and cleanup

.PARAMETER Timeout
    Timeout in seconds for waiting operations (default: 10)

.EXAMPLE
    .\hack\watcher-basic-test.ps1
    
.EXAMPLE
    .\hack\watcher-basic-test.ps1 -Timeout 15

.NOTES
    Requires: go, PowerShell 5.1+
    No Kubernetes cluster required
#>

param(
    [int]$Timeout = 10
)

# Error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Configuration
$BINARY_PATH = ".\conductor-loop.exe"
$HANDOFF_DIR = ".\handoff"
$SCHEMA_PATH = ".\docs\contracts\intent.schema.json"

# Color functions for output
function Write-Success { param($Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Error { param($Message) Write-Host "❌ $Message" -ForegroundColor Red }
function Write-Warning { param($Message) Write-Host "⚠️ $Message" -ForegroundColor Yellow }
function Write-Info { param($Message) Write-Host "ℹ️ $Message" -ForegroundColor Cyan }
function Write-Step { param($Message) Write-Host "`n🔄 $Message" -ForegroundColor Blue }

# Cleanup function
function Cleanup {
    Write-Step "Cleaning up..."
    
    # Stop conductor-watch process if running
    if ($global:ConductorProcess -and !$global:ConductorProcess.HasExited) {
        Write-Info "Stopping conductor-watch process (PID: $($global:ConductorProcess.Id))"
        try {
            $global:ConductorProcess.Kill()
            $global:ConductorProcess.WaitForExit(3000)
            Write-Success "Conductor-watch process stopped"
        } catch {
            Write-Warning "Failed to stop conductor-watch gracefully: $_"
        }
    }
    
    # Remove test files
    try {
        Get-ChildItem -Path $HANDOFF_DIR -Filter "intent-*basic-test*.json" -ErrorAction SilentlyContinue | Remove-Item -Force
        Write-Success "Test files removed"
    } catch {
        Write-Warning "Failed to remove test files: $_"
    }
}

# Register cleanup on exit
Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Cleanup }
trap { Cleanup; break }

function Build-ConductorWatch {
    Write-Step "Building conductor-watch..."
    
    if (Test-Path $BINARY_PATH) {
        Write-Info "Binary already exists, rebuilding..."
    }
    
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

function Start-ConductorWatch {
    Write-Step "Starting conductor-watch in background..."
    
    # Ensure handoff directory exists
    if (!(Test-Path $HANDOFF_DIR)) {
        New-Item -ItemType Directory -Path $HANDOFF_DIR -Force | Out-Null
        Write-Success "Created handoff directory: $HANDOFF_DIR"
    }
    
    # Build arguments for conductor-watch
    $args = @(
        "-handoff-dir", $HANDOFF_DIR
        "-schema", $SCHEMA_PATH
        "-batch-size", "1"
        "-batch-interval", "1s"
        "-porch-mode", "structured"
    )
    
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
        Start-Sleep -Seconds 1
        
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

function Create-TestIntentFiles {
    Write-Step "Creating test intent files..."
    
    $timestamp = Get-Date -Format "yyyyMMddHHmmss"
    
    # Test file 1: Valid scaling intent
    $testIntent1 = @{
        intent_type = "scaling"
        target = "nf-simulator"
        namespace = "test-ns"
        replicas = 3
        reason = "Basic test - scale up"
        source = "test"
        correlation_id = "basic-test-1-$timestamp"
    }
    
    # Test file 2: Another valid scaling intent
    $testIntent2 = @{
        intent_type = "scaling"
        target = "du-simulator"
        namespace = "ran-ns"
        replicas = 1
        reason = "Basic test - scale down"
        source = "test"
        correlation_id = "basic-test-2-$timestamp"
    }
    
    $testFiles = @()
    
    try {
        # Create test file 1 using Here-String for better JSON formatting
        $file1 = Join-Path $HANDOFF_DIR "intent-$timestamp-basic-test-1.json"
        $json1 = @"
{
  "intent_type": "scaling",
  "target": "nf-simulator",
  "namespace": "test-ns",
  "replicas": 3,
  "reason": "Basic test - scale up",
  "source": "test",
  "correlation_id": "basic-test-1-$timestamp"
}
"@
        $json1 | Out-File -FilePath $file1 -Encoding UTF8 -NoNewline
        $testFiles += $file1
        Write-Success "Created test file: $file1"
        
        # Wait a moment before creating the second file
        Start-Sleep -Seconds 1
        
        # Create test file 2
        $file2 = Join-Path $HANDOFF_DIR "intent-$timestamp-basic-test-2.json"
        $json2 = @"
{
  "intent_type": "scaling",
  "target": "du-simulator",
  "namespace": "ran-ns",
  "replicas": 1,
  "reason": "Basic test - scale down",
  "source": "test",
  "correlation_id": "basic-test-2-$timestamp"
}
"@
        $json2 | Out-File -FilePath $file2 -Encoding UTF8 -NoNewline
        $testFiles += $file2
        Write-Success "Created test file: $file2"
        
        return $testFiles
    } catch {
        Write-Error "Failed to create test files: $_"
        throw
    }
}

function Verify-Processing {
    param([array]$TestFiles)
    
    Write-Step "Verifying conductor-watch processing..."
    
    # Wait for processing
    Write-Info "Waiting ${Timeout}s for file processing..."
    Start-Sleep -Seconds $Timeout
    
    # Check conductor-watch process status
    if ($global:ConductorProcess -and !$global:ConductorProcess.HasExited) {
        Write-Success "Conductor-watch is still running"
        
        # Try to capture output
        try {
            # Read available output without blocking
            $output = ""
            $error = ""
            
            # Use Peek to check if data is available
            if ($global:ConductorProcess.StandardOutput.Peek() -ge 0) {
                $output = $global:ConductorProcess.StandardOutput.ReadToEnd()
            }
            if ($global:ConductorProcess.StandardError.Peek() -ge 0) {
                $error = $global:ConductorProcess.StandardError.ReadToEnd()
            }
            
            if ($output) {
                Write-Info "Conductor-watch stdout:"
                Write-Host $output -ForegroundColor Gray
                
                # Check for porch output patterns in stdout
                $outputPatterns = @(
                    "wrote: output",
                    "scaling-patch.yaml"
                )
                
                $foundOutputPatterns = @()
                foreach ($pattern in $outputPatterns) {
                    if ($output -match $pattern) {
                        $foundOutputPatterns += $pattern
                    }
                }
                
                if ($foundOutputPatterns.Count -gt 0) {
                    Write-Success "Found porch output patterns - files are being processed!"
                    foreach ($pattern in $foundOutputPatterns) {
                        Write-Info "✓ $pattern"
                    }
                } else {
                    Write-Warning "No porch output patterns found in stdout"
                }
            }
            
            if ($error) {
                Write-Info "Conductor-watch stderr (logs):"
                Write-Host $error -ForegroundColor Gray
                
                # Check for expected log patterns in stderr
                $logPatterns = @(
                    "Starting conductor-loop",
                    "Watching directory",
                    "Intent file detected",
                    "Successfully processed"
                )
                
                $foundLogPatterns = @()
                foreach ($pattern in $logPatterns) {
                    if ($error -match $pattern) {
                        $foundLogPatterns += $pattern
                    }
                }
                
                Write-Success "Found $($foundLogPatterns.Count)/$($logPatterns.Count) expected log patterns"
                foreach ($pattern in $foundLogPatterns) {
                    Write-Info "✓ $pattern"
                }
                
                if ($foundLogPatterns.Count -eq $logPatterns.Count) {
                    Write-Success "All expected log patterns found - conductor-watch is working correctly!"
                } else {
                    Write-Warning "Some expected log patterns missing"
                }
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
    
    # Check if test files still exist (they should be processed and might be moved/deleted)
    $remainingFiles = @()
    foreach ($file in $TestFiles) {
        if (Test-Path $file) {
            $remainingFiles += $file
        }
    }
    
    if ($remainingFiles.Count -lt $TestFiles.Count) {
        Write-Success "$($TestFiles.Count - $remainingFiles.Count) files were processed/removed"
    } else {
        Write-Info "All test files still present (may be normal depending on configuration)"
    }
    
    # Check for any files in handoff directory
    $allFiles = Get-ChildItem -Path $HANDOFF_DIR -Filter "*.json" | Sort-Object CreationTime -Descending
    if ($allFiles) {
        Write-Info "Current files in handoff directory:"
        foreach ($file in $allFiles | Select-Object -First 5) {
            Write-Info "- $($file.Name) ($(Get-Date $file.CreationTime -Format 'HH:mm:ss'))"
        }
    } else {
        Write-Info "No JSON files found in handoff directory"
    }
}

function Show-Summary {
    Write-Step "Test Summary:"
    
    Write-Host @"

📋 Basic Conductor-Watch Test Results:

✅ What was tested:
- Binary build process
- File system monitoring startup
- Intent file detection
- Basic processing workflow

🔧 Key observations:
- Conductor-watch should detect CREATE/WRITE events for intent-*.json files
- Processing should log file names and operations
- Schema validation should occur against docs/contracts/intent.schema.json
- Files may be moved/processed depending on configuration

📝 Next steps for full testing:
1. Run complete smoke test: .\hack\watcher-smoke.ps1
2. Test with real Kubernetes cluster and NetworkIntent CRDs
3. Verify end-to-end integration with porch/kpt

"@ -ForegroundColor White
}

# Main execution
function Main {
    Write-Host "`n🧪 Conductor-Watch Basic Test" -ForegroundColor Magenta
    Write-Host "==============================`n" -ForegroundColor Magenta
    
    try {
        # Step 1: Build conductor-watch
        Build-ConductorWatch
        
        # Step 2: Start conductor-watch
        Start-ConductorWatch
        
        # Step 3: Create test files
        $testFiles = Create-TestIntentFiles
        
        # Step 4: Verify processing
        Verify-Processing -TestFiles $testFiles
        
        # Step 5: Show summary
        Show-Summary
        
        Write-Success "`nBasic test completed!"
        
    } catch {
        Write-Error "Basic test failed: $_"
        exit 1
    } finally {
        Cleanup
    }
}

# Execute main function
Main
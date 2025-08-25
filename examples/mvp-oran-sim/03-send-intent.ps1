#!/usr/bin/env pwsh
# Script: 03-send-intent.ps1
# Purpose: Send scaling intent via HTTP POST or drop to handoff directory

param(
    [string]$Method = "handoff",  # "http" or "handoff"
    [string]$IntentUrl = "http://localhost:8080/intent",
    [string]$HandoffDir = "../../handoff",
    [string]$Target = "nf-sim",
    [string]$Namespace = "mvp-demo",
    [int]$Replicas = 3,
    [string]$Source = "test",
    [string]$Reason = "MVP demo scaling test"
)

$ErrorActionPreference = "Stop"

Write-Host "==== MVP Demo: Send Scaling Intent ====" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray
Write-Host "Method: $Method" -ForegroundColor Gray

# Generate correlation ID
$correlationId = "mvp-demo-$(Get-Date -Format 'yyyyMMddHHmmss')"

# Create intent JSON according to schema
$intentJson = @{
    intent_type = "scaling"
    target = $Target
    namespace = $Namespace
    replicas = $Replicas
    reason = $Reason
    source = $Source
    correlation_id = $correlationId
} | ConvertTo-Json -Depth 10

Write-Host "`nIntent JSON:" -ForegroundColor Yellow
Write-Host $intentJson

# Validate JSON against schema if jq is available
$jqAvailable = Get-Command jq -ErrorAction SilentlyContinue
if ($jqAvailable) {
    Write-Host "`nValidating intent structure..." -ForegroundColor Yellow
    $tempFile = Join-Path $env:TEMP "intent-validate.json"
    Set-Content -Path $tempFile -Value $intentJson -Encoding UTF8
    
    $validation = echo $intentJson | jq -e '.intent_type == "scaling" and .target and .namespace and .replicas' 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Intent structure is valid" -ForegroundColor Green
    } else {
        Write-Warning "Intent structure validation failed, but continuing..."
    }
    Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue
}

# Send intent based on method
if ($Method -eq "http") {
    # Send via HTTP POST
    Write-Host "`nSending intent via HTTP POST to: $IntentUrl" -ForegroundColor Yellow
    
    try {
        $headers = @{
            "Content-Type" = "application/json"
            "X-Correlation-Id" = $correlationId
        }
        
        $response = Invoke-RestMethod -Uri $IntentUrl -Method Post -Body $intentJson -Headers $headers -ContentType "application/json"
        
        Write-Host "Intent sent successfully!" -ForegroundColor Green
        Write-Host "Response:" -ForegroundColor Gray
        $response | ConvertTo-Json -Depth 10
    } catch {
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
            Write-Warning "HTTP request failed with status code: $statusCode"
            
            # Try to get error details
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $errorBody = $reader.ReadToEnd()
                Write-Host "Error details: $errorBody" -ForegroundColor Red
            } catch {
                Write-Host "Could not read error response body" -ForegroundColor Red
            }
        } else {
            Write-Warning "Could not connect to intent endpoint: $_"
            Write-Host "Make sure the intent-ingest service is running" -ForegroundColor Yellow
        }
        
        # Fallback to handoff method
        Write-Host "`nFalling back to handoff directory method..." -ForegroundColor Yellow
        $Method = "handoff"
    }
}

if ($Method -eq "handoff") {
    # Drop to handoff directory
    $scriptDir = Split-Path -Parent $PSScriptRoot
    $handoffPath = Join-Path $scriptDir $HandoffDir
    
    # Resolve to absolute path
    $handoffPath = (Resolve-Path $handoffPath -ErrorAction SilentlyContinue).Path
    if (-not $handoffPath) {
        # Create handoff directory if it doesn't exist
        $handoffPath = Join-Path (Get-Location).Path $HandoffDir
        New-Item -ItemType Directory -Path $handoffPath -Force | Out-Null
    }
    
    Write-Host "`nDropping intent to handoff directory: $handoffPath" -ForegroundColor Yellow
    
    # Generate filename with timestamp
    $timestamp = Get-Date -Format "yyyyMMddTHHmmssZ"
    $fileName = "intent-$timestamp.json"
    $filePath = Join-Path $handoffPath $fileName
    
    # Write intent to file
    Set-Content -Path $filePath -Value $intentJson -Encoding UTF8
    
    if (Test-Path $filePath) {
        Write-Host "Intent written successfully!" -ForegroundColor Green
        Write-Host "File: $fileName" -ForegroundColor Gray
        Write-Host "Full path: $filePath" -ForegroundColor Gray
        
        # Verify file contents
        Write-Host "`nVerifying file contents:" -ForegroundColor Yellow
        Get-Content $filePath | Write-Host
    } else {
        Write-Error "Failed to write intent file"
        exit 1
    }
}

# Check if conductor is running (optional)
Write-Host "`nChecking for conductor process..." -ForegroundColor Yellow
$conductorProc = Get-Process -Name "conductor" -ErrorAction SilentlyContinue
if ($conductorProc) {
    Write-Host "Conductor process is running (PID: $($conductorProc.Id))" -ForegroundColor Green
} else {
    Write-Host "Conductor process not found. Make sure it's running to process the intent." -ForegroundColor Yellow
    Write-Host "Start conductor with: go run ./cmd/conductor" -ForegroundColor Gray
}

# Summary
Write-Host "`n==== Intent Summary ====" -ForegroundColor Green
Write-Host "✓ Target: $Target"
Write-Host "✓ Namespace: $Namespace"
Write-Host "✓ Desired replicas: $Replicas"
Write-Host "✓ Correlation ID: $correlationId"
Write-Host "✓ Method: $Method"

Write-Host "`nIntent sent! Next step: Run 04-porch-apply.ps1" -ForegroundColor Cyan
Write-Host "Monitor the conductor logs for processing status" -ForegroundColor Gray
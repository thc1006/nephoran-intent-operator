#!/usr/bin/env pwsh
# populate-knowledge-base.ps1 - Windows PowerShell script to populate Weaviate vector store

param(
    [string]$WeaviateUrl = "http://localhost:8080",
    [switch]$ClearExisting,
    [switch]$UseClusterUrl
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Nephoran Knowledge Base Population" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Check prerequisites
if (-not $env:OPENAI_API_KEY) {
    Write-Host "ERROR: OPENAI_API_KEY environment variable not set" -ForegroundColor Red
    Write-Host "Please set: `$env:OPENAI_API_KEY = 'your-api-key'" -ForegroundColor Yellow
    exit 1
}

# Check Python installation
try {
    $pythonVersion = python --version 2>&1
    Write-Host "✓ Python found: $pythonVersion" -ForegroundColor Green
} catch {
    Write-Host "ERROR: Python not found. Please install Python 3.8+" -ForegroundColor Red
    exit 1
}

# Set paths
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$knowledgeBasePath = Join-Path $scriptDir "knowledge_base"
$populateScript = Join-Path $scriptDir "scripts\populate_vector_store_enhanced.py"

# Check if knowledge base exists
if (-not (Test-Path $knowledgeBasePath)) {
    Write-Host "ERROR: Knowledge base directory not found: $knowledgeBasePath" -ForegroundColor Red
    exit 1
}

# Check if populate script exists
if (-not (Test-Path $populateScript)) {
    Write-Host "ERROR: Population script not found: $populateScript" -ForegroundColor Red
    exit 1
}

# List files to be processed
Write-Host "`nFiles to be processed:" -ForegroundColor Yellow
Get-ChildItem -Path $knowledgeBasePath -File | ForEach-Object {
    Write-Host "  • $($_.Name)" -ForegroundColor White
}

# Set Weaviate URL based on environment
if ($UseClusterUrl) {
    $env:WEAVIATE_URL = "http://weaviate.nephoran-system.svc.cluster.local:8080"
    Write-Host "`nUsing cluster Weaviate URL: $($env:WEAVIATE_URL)" -ForegroundColor Yellow
} else {
    $env:WEAVIATE_URL = $WeaviateUrl
    Write-Host "`nUsing Weaviate URL: $($env:WEAVIATE_URL)" -ForegroundColor Yellow
}

# Check if we need to port-forward
if ($WeaviateUrl -eq "http://localhost:8080") {
    Write-Host "`nChecking for Weaviate deployment in cluster..." -ForegroundColor Yellow
    $weaviateDeployment = kubectl get deployment -n nephoran-system weaviate 2>$null
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Found Weaviate deployment. Setting up port-forward..." -ForegroundColor Green
        
        # Kill any existing port-forward
        Get-Process kubectl -ErrorAction SilentlyContinue | Where-Object {
            $_.CommandLine -like "*port-forward*weaviate*"
        } | Stop-Process -Force
        
        # Start port-forward in background
        $portForwardJob = Start-Job -ScriptBlock {
            kubectl port-forward -n nephoran-system svc/weaviate 8080:8080
        }
        
        Write-Host "Waiting for port-forward to be ready..." -ForegroundColor Yellow
        Start-Sleep -Seconds 3
        
        # Test connection
        try {
            $response = Invoke-WebRequest -Uri "$WeaviateUrl/v1/.well-known/ready" -Method Get -TimeoutSec 5
            if ($response.StatusCode -eq 200) {
                Write-Host "✓ Weaviate connection successful" -ForegroundColor Green
            }
        } catch {
            Write-Host "WARNING: Could not connect to Weaviate. Make sure it's running." -ForegroundColor Yellow
        }
    }
}

# Install Python dependencies if needed
Write-Host "`nChecking Python dependencies..." -ForegroundColor Yellow
$requirementsFile = Join-Path $scriptDir "requirements-rag.txt"
if (Test-Path $requirementsFile) {
    Write-Host "Installing dependencies from requirements-rag.txt..." -ForegroundColor White
    python -m pip install -q -r $requirementsFile
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Dependencies installed successfully" -ForegroundColor Green
    } else {
        Write-Host "WARNING: Some dependencies may have failed to install" -ForegroundColor Yellow
    }
}

# Build command arguments
$pythonArgs = @($populateScript, $knowledgeBasePath, "--log-level", "INFO")
if ($ClearExisting) {
    $pythonArgs += "--clear-existing"
    Write-Host "`nWARNING: Will clear existing data before populating!" -ForegroundColor Yellow
}

# Run the population script
Write-Host "`nPopulating vector store..." -ForegroundColor Cyan
Write-Host "Command: python $($pythonArgs -join ' ')" -ForegroundColor Gray

$populateProcess = Start-Process -FilePath "python" -ArgumentList $pythonArgs -NoNewWindow -PassThru -Wait

if ($populateProcess.ExitCode -eq 0) {
    Write-Host "`n✓ Knowledge base population completed successfully!" -ForegroundColor Green
} else {
    Write-Host "`n✗ Knowledge base population failed with exit code: $($populateProcess.ExitCode)" -ForegroundColor Red
}

# Clean up port-forward if we started it
if ($portForwardJob) {
    Write-Host "`nCleaning up port-forward..." -ForegroundColor Yellow
    Stop-Job -Job $portForwardJob
    Remove-Job -Job $portForwardJob
}

Write-Host "`nDone!" -ForegroundColor Cyan
exit $populateProcess.ExitCode
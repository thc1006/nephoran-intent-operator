#!/usr/bin/env pwsh
# Script: 04-porch-apply.ps1
# Purpose: Apply package via porchctl or kpt live apply

param(
    [string]$PackageName = "nf-sim-package",
    [string]$PackageDir = "",
    [string]$Namespace = "mvp-demo",
    [string]$Method = "direct",  # "porch" or "direct"
    [int]$WaitSeconds = 30
)

$ErrorActionPreference = "Stop"

Write-Host "==== MVP Demo: Apply Package with Porch/KPT ====" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray
Write-Host "Method: $Method" -ForegroundColor Gray

# Set package directory if not provided
if (-not $PackageDir) {
    $PackageDir = Join-Path $PSScriptRoot "package-$PackageName"
}

# Verify package directory exists
if (-not (Test-Path $PackageDir)) {
    Write-Error "Package directory not found: $PackageDir"
    Write-Host "Run 02-prepare-nf-sim.ps1 first to create the package" -ForegroundColor Yellow
    exit 1
}

Write-Host "Package directory: $PackageDir" -ForegroundColor Gray

if ($Method -eq "porch") {
    # Apply via Porch
    Write-Host "`nApplying package via Porch..." -ForegroundColor Yellow
    
    # Check if Porch is available
    $porchAvailable = kubectl get crd packagerevisions.porch.kpt.dev 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Porch CRDs not found. Falling back to direct apply method."
        $Method = "direct"
    } else {
        # Create PackageRevision
        Write-Host "Creating PackageRevision for $PackageName..." -ForegroundColor Yellow
        
        $packageRevisionYaml = @"
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: $PackageName-v1
  namespace: default
spec:
  packageName: $PackageName
  revision: v1
  repository: mvp-packages
  lifecycle: Published
"@
        
        $prFile = Join-Path $env:TEMP "package-revision-$PackageName.yaml"
        Set-Content -Path $prFile -Value $packageRevisionYaml -Encoding UTF8
        
        kubectl apply -f $prFile
        if ($LASTEXITCODE -eq 0) {
            Write-Host "PackageRevision created successfully" -ForegroundColor Green
            
            # Apply the package
            Write-Host "Applying package with porchctl..." -ForegroundColor Yellow
            porchctl rpkg approve default/$PackageName-v1 2>$null
            porchctl rpkg propose-delete default/$PackageName-v1 2>$null
        } else {
            Write-Warning "Could not create PackageRevision. Falling back to direct apply."
            $Method = "direct"
        }
        
        Remove-Item -Path $prFile -Force -ErrorAction SilentlyContinue
    }
}

if ($Method -eq "direct") {
    # Direct kubectl/kpt apply
    Write-Host "`nApplying package directly with kubectl..." -ForegroundColor Yellow
    
    # Check if kpt is available for live apply
    $kptAvailable = Get-Command kpt -ErrorAction SilentlyContinue
    
    if ($kptAvailable) {
        Write-Host "Using kpt live apply..." -ForegroundColor Yellow
        
        Push-Location $PackageDir
        try {
            # Initialize kpt live if not already done
            kpt live init . 2>$null
            
            # Apply the package
            kpt live apply . --reconcile-timeout=2m
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Package applied successfully with kpt!" -ForegroundColor Green
            } else {
                Write-Warning "kpt live apply failed, trying kubectl apply..."
                kubectl apply -f . --namespace=$Namespace
            }
        } finally {
            Pop-Location
        }
    } else {
        # Fall back to kubectl apply
        Write-Host "Using kubectl apply..." -ForegroundColor Yellow
        
        $yamlFiles = Get-ChildItem -Path $PackageDir -Filter "*.yaml" | Where-Object { $_.Name -ne "Kptfile" }
        
        foreach ($file in $yamlFiles) {
            Write-Host "Applying $($file.Name)..." -ForegroundColor Gray
            kubectl apply -f $file.FullName
            
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to apply $($file.Name)"
                exit 1
            }
        }
        
        Write-Host "All resources applied successfully!" -ForegroundColor Green
    }
}

# Wait for deployment to be ready
Write-Host "`nWaiting for deployment to be ready (max $WaitSeconds seconds)..." -ForegroundColor Yellow

$startTime = Get-Date
$timeout = $startTime.AddSeconds($WaitSeconds)

while ((Get-Date) -lt $timeout) {
    $deployment = kubectl get deployment nf-sim -n $Namespace -o json 2>$null | ConvertFrom-Json
    
    if ($deployment) {
        $ready = $deployment.status.readyReplicas
        $desired = $deployment.spec.replicas
        
        Write-Host "Status: $ready/$desired replicas ready" -ForegroundColor Gray
        
        if ($ready -eq $desired -and $ready -gt 0) {
            Write-Host "Deployment is ready!" -ForegroundColor Green
            break
        }
    }
    
    Start-Sleep -Seconds 2
}

# Show final deployment status
Write-Host "`n==== Deployment Status ====" -ForegroundColor Cyan
kubectl get deployment nf-sim -n $Namespace
Write-Host ""
kubectl get pods -n $Namespace -l app=nf-sim

# Show service status
Write-Host "`n==== Service Status ====" -ForegroundColor Cyan
kubectl get service nf-sim -n $Namespace

# Summary
Write-Host "`n==== Apply Summary ====" -ForegroundColor Green
Write-Host "✓ Package: $PackageName"
Write-Host "✓ Namespace: $Namespace"
Write-Host "✓ Method: $Method"

$deployment = kubectl get deployment nf-sim -n $Namespace -o json 2>$null | ConvertFrom-Json
if ($deployment) {
    Write-Host "✓ Current replicas: $($deployment.status.replicas)"
    Write-Host "✓ Ready replicas: $($deployment.status.readyReplicas)"
}

Write-Host "`nPackage applied! Next step: Run 05-validate.ps1" -ForegroundColor Cyan
Write-Host "To scale, send another intent with 03-send-intent.ps1" -ForegroundColor Gray
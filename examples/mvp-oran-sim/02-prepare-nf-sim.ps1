#!/usr/bin/env pwsh
# Script: 02-prepare-nf-sim.ps1
# Purpose: Create a minimal KRM package for NF simulator deployment

param(
    [string]$Namespace = "mvp-demo",
    [string]$PackageName = "nf-sim-package",
    [string]$PackageRepo = "mvp-packages",
    [switch]$SkipNamespace = $false
)

$ErrorActionPreference = "Stop"

Write-Host "==== MVP Demo: Prepare NF Simulator Package ====" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray

# Create namespace if needed
if (-not $SkipNamespace) {
    Write-Host "`nCreating namespace: $Namespace" -ForegroundColor Yellow
    
    $nsExists = kubectl get namespace $Namespace 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Namespace $Namespace already exists" -ForegroundColor Green
    } else {
        kubectl create namespace $Namespace
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to create namespace $Namespace"
            exit 1
        }
        Write-Host "Namespace $Namespace created successfully" -ForegroundColor Green
    }
}

# Create local package directory structure
$packageDir = Join-Path $PSScriptRoot "package-$PackageName"
Write-Host "`nCreating package directory: $packageDir" -ForegroundColor Yellow

if (Test-Path $packageDir) {
    Write-Host "Removing existing package directory..."
    Remove-Item -Path $packageDir -Recurse -Force
}

New-Item -ItemType Directory -Path $packageDir -Force | Out-Null

# Create Kptfile
$kptfileContent = @"
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: $PackageName
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: NF Simulator package for MVP demo
  keywords:
  - nf-sim
  - mvp
  - oran
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-namespace:v0.4.1
    configMap:
      namespace: $Namespace
"@

$kptfilePath = Join-Path $packageDir "Kptfile"
Set-Content -Path $kptfilePath -Value $kptfileContent -Encoding UTF8
Write-Host "Created Kptfile" -ForegroundColor Green

# Copy deployment YAML
$deploymentSource = Join-Path $PSScriptRoot "nf-sim-deployment.yaml"
$deploymentDest = Join-Path $packageDir "nf-sim-deployment.yaml"

if (Test-Path $deploymentSource) {
    Copy-Item -Path $deploymentSource -Destination $deploymentDest
    Write-Host "Copied nf-sim-deployment.yaml" -ForegroundColor Green
} else {
    # Create deployment YAML if it doesn't exist
    $deploymentContent = @"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nf-sim
  namespace: $Namespace
  labels:
    app: nf-sim
    component: cnf-simulator
    orchestrator: porch
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nf-sim
  template:
    metadata:
      labels:
        app: nf-sim
        component: cnf-simulator
    spec:
      containers:
      - name: simulator
        image: nginx:alpine
        ports:
        - containerPort: 80
          name: http
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: nf-sim
  namespace: $Namespace
  labels:
    app: nf-sim
spec:
  selector:
    app: nf-sim
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  type: ClusterIP
"@
    Set-Content -Path $deploymentDest -Value $deploymentContent -Encoding UTF8
    Write-Host "Created nf-sim-deployment.yaml" -ForegroundColor Green
}

# Create package README
$readmeContent = @"
# NF Simulator Package

This package deploys a Network Function (NF) simulator for the MVP demo.

## Contents
- `nf-sim-deployment.yaml`: Kubernetes Deployment and Service for the NF simulator
- `Kptfile`: KPT package metadata and pipeline configuration

## Usage
This package is managed by Porch and can be scaled using NetworkIntent CRDs.

## Default Configuration
- Namespace: $Namespace
- Initial replicas: 1
- Container: nginx:alpine (lightweight simulator)
"@

$readmePath = Join-Path $packageDir "README.md"
Set-Content -Path $readmePath -Value $readmeContent -Encoding UTF8
Write-Host "Created README.md" -ForegroundColor Green

# Initialize kpt package
Write-Host "`nInitializing kpt package..." -ForegroundColor Yellow
Push-Location $packageDir
try {
    kpt pkg init . 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "kpt pkg init returned non-zero exit code, but package may still be valid"
    }
} finally {
    Pop-Location
}

# If Porch is available, register the package repository
Write-Host "`nChecking Porch availability..." -ForegroundColor Yellow
$porchAvailable = kubectl get crd repositories.porch.kpt.dev 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "Porch CRDs are available" -ForegroundColor Green
    
    # Check if repository exists
    $repoExists = kubectl get repository $PackageRepo -n default 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "`nCreating Porch repository: $PackageRepo" -ForegroundColor Yellow
        
        $repoYaml = @"
apiVersion: porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: $PackageRepo
  namespace: default
spec:
  type: git
  content: Package
  deployment: false
  git:
    repo: https://github.com/example/mvp-packages
    branch: main
    directory: /
"@
        $repoFile = Join-Path $env:TEMP "repo-$PackageRepo.yaml"
        Set-Content -Path $repoFile -Value $repoYaml -Encoding UTF8
        
        kubectl apply -f $repoFile
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Repository $PackageRepo created" -ForegroundColor Green
        } else {
            Write-Warning "Could not create repository. Manual setup may be required."
        }
        Remove-Item -Path $repoFile -Force
    } else {
        Write-Host "Repository $PackageRepo already exists" -ForegroundColor Green
    }
} else {
    Write-Warning "Porch is not installed. Package created locally but not registered with Porch."
    Write-Host "Install Porch first using 01-install-porch.ps1" -ForegroundColor Yellow
}

# Summary
Write-Host "`n==== Package Preparation Summary ====" -ForegroundColor Green
Write-Host "✓ Namespace: $Namespace"
Write-Host "✓ Package directory: $packageDir"
Write-Host "✓ Package contents:"
Get-ChildItem -Path $packageDir | ForEach-Object {
    Write-Host "  - $($_.Name)"
}

Write-Host "`nPackage prepared! Next step: Run 03-send-intent.ps1" -ForegroundColor Cyan
Write-Host "Package location: $packageDir" -ForegroundColor Gray
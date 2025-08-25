#!/usr/bin/env pwsh
# Script: 05-validate.ps1
# Purpose: Validate deployment status and replica count

param(
    [string]$Namespace = "mvp-demo",
    [string]$DeploymentName = "nf-sim",
    [int]$ExpectedReplicas = 0,
    [switch]$Continuous = $false,
    [int]$Interval = 5
)

$ErrorActionPreference = "Stop"

Write-Host "==== MVP Demo: Validate Deployment ====" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray

function Get-DeploymentStatus {
    param([string]$Name, [string]$Namespace)
    
    $deployment = kubectl get deployment $Name -n $Namespace -o json 2>$null | ConvertFrom-Json
    
    if ($deployment) {
        return @{
            Name = $deployment.metadata.name
            Namespace = $deployment.metadata.namespace
            Replicas = $deployment.spec.replicas
            ReadyReplicas = $deployment.status.readyReplicas
            UpdatedReplicas = $deployment.status.updatedReplicas
            AvailableReplicas = $deployment.status.availableReplicas
            Conditions = $deployment.status.conditions
            CreationTime = $deployment.metadata.creationTimestamp
            Labels = $deployment.metadata.labels
        }
    }
    return $null
}

function Show-ValidationResults {
    param($Status, [int]$Expected)
    
    Write-Host "`n==== Deployment Validation Results ====" -ForegroundColor Cyan
    
    if (-not $Status) {
        Write-Host "❌ Deployment not found: $DeploymentName in namespace $Namespace" -ForegroundColor Red
        return $false
    }
    
    Write-Host "Deployment: $($Status.Name)" -ForegroundColor Yellow
    Write-Host "Namespace: $($Status.Namespace)" -ForegroundColor Gray
    Write-Host "Created: $($Status.CreationTime)" -ForegroundColor Gray
    
    Write-Host "`nReplica Status:" -ForegroundColor Yellow
    Write-Host "  Desired:   $($Status.Replicas)" -ForegroundColor Gray
    Write-Host "  Ready:     $($Status.ReadyReplicas)" -ForegroundColor Gray
    Write-Host "  Updated:   $($Status.UpdatedReplicas)" -ForegroundColor Gray
    Write-Host "  Available: $($Status.AvailableReplicas)" -ForegroundColor Gray
    
    # Check if deployment is healthy
    $isHealthy = $Status.ReadyReplicas -eq $Status.Replicas -and $Status.Replicas -gt 0
    
    if ($isHealthy) {
        Write-Host "`n✅ Deployment is healthy" -ForegroundColor Green
    } else {
        Write-Host "`n⚠️ Deployment is not fully ready" -ForegroundColor Yellow
    }
    
    # Check expected replicas if specified
    if ($Expected -gt 0) {
        if ($Status.Replicas -eq $Expected) {
            Write-Host "✅ Replica count matches expected: $Expected" -ForegroundColor Green
        } else {
            Write-Host "❌ Replica count mismatch - Expected: $Expected, Actual: $($Status.Replicas)" -ForegroundColor Red
        }
    }
    
    # Show conditions
    if ($Status.Conditions) {
        Write-Host "`nConditions:" -ForegroundColor Yellow
        foreach ($condition in $Status.Conditions) {
            $symbol = if ($condition.status -eq "True") { "✓" } else { "✗" }
            $color = if ($condition.status -eq "True") { "Green" } else { "Red" }
            Write-Host "  $symbol $($condition.type): $($condition.status)" -ForegroundColor $color
            if ($condition.message) {
                Write-Host "    Message: $($condition.message)" -ForegroundColor Gray
            }
        }
    }
    
    return $isHealthy
}

# Main validation loop
do {
    Clear-Host
    Write-Host "==== MVP Demo: Validate Deployment ====" -ForegroundColor Cyan
    Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')" -ForegroundColor Gray
    
    # Get deployment status
    $status = Get-DeploymentStatus -Name $DeploymentName -Namespace $Namespace
    
    # Show validation results
    $isHealthy = Show-ValidationResults -Status $status -Expected $ExpectedReplicas
    
    # Show pod details
    Write-Host "`n==== Pod Details ====" -ForegroundColor Cyan
    kubectl get pods -n $Namespace -l app=$DeploymentName --no-headers | ForEach-Object {
        $fields = $_ -split '\s+'
        $podName = $fields[0]
        $ready = $fields[1]
        $status = $fields[2]
        $restarts = $fields[3]
        $age = $fields[4]
        
        $symbol = if ($status -eq "Running") { "✓" } else { "✗" }
        $color = if ($status -eq "Running") { "Green" } else { "Yellow" }
        
        Write-Host "  $symbol $podName - Status: $status, Ready: $ready, Restarts: $restarts, Age: $age" -ForegroundColor $color
    }
    
    # Show service endpoint
    Write-Host "`n==== Service Endpoint ====" -ForegroundColor Cyan
    $service = kubectl get service $DeploymentName -n $Namespace -o json 2>$null | ConvertFrom-Json
    if ($service) {
        Write-Host "Service: $($service.metadata.name)" -ForegroundColor Yellow
        Write-Host "Type: $($service.spec.type)" -ForegroundColor Gray
        Write-Host "Cluster IP: $($service.spec.clusterIP)" -ForegroundColor Gray
        Write-Host "Port: $($service.spec.ports[0].port)" -ForegroundColor Gray
        
        # Test service connectivity if possible
        if ($service.spec.type -eq "LoadBalancer" -and $service.status.loadBalancer.ingress) {
            $endpoint = $service.status.loadBalancer.ingress[0].ip
            if (-not $endpoint) {
                $endpoint = $service.status.loadBalancer.ingress[0].hostname
            }
            Write-Host "External Endpoint: $endpoint" -ForegroundColor Green
        }
    } else {
        Write-Host "Service not found" -ForegroundColor Red
    }
    
    # Show recent events
    Write-Host "`n==== Recent Events ====" -ForegroundColor Cyan
    $events = kubectl get events -n $Namespace --field-selector involvedObject.name=$DeploymentName --sort-by='.lastTimestamp' 2>$null
    if ($events) {
        $eventLines = $events -split "`n" | Select-Object -Last 5
        foreach ($line in $eventLines) {
            if ($line -and $line -notmatch "^LAST SEEN") {
                Write-Host "  $line" -ForegroundColor Gray
            }
        }
    } else {
        Write-Host "  No recent events" -ForegroundColor Gray
    }
    
    # Summary
    Write-Host "`n==== Validation Summary ====" -ForegroundColor Green
    if ($isHealthy) {
        Write-Host "✅ Deployment validation PASSED" -ForegroundColor Green
        Write-Host "  - Deployment is running with $($status.ReadyReplicas) ready replicas" -ForegroundColor Gray
    } else {
        Write-Host "⚠️ Deployment validation WARNINGS" -ForegroundColor Yellow
        if (-not $status) {
            Write-Host "  - Deployment not found" -ForegroundColor Red
        } elseif ($status.ReadyReplicas -ne $status.Replicas) {
            Write-Host "  - Not all replicas are ready ($($status.ReadyReplicas)/$($status.Replicas))" -ForegroundColor Yellow
        }
    }
    
    if ($Continuous) {
        Write-Host "`nRefreshing in $Interval seconds... (Press Ctrl+C to stop)" -ForegroundColor Gray
        Start-Sleep -Seconds $Interval
    }
} while ($Continuous)

# Final message
if (-not $Continuous) {
    Write-Host "`n==== Next Steps ====" -ForegroundColor Cyan
    Write-Host "1. To scale up: Run 03-send-intent.ps1 -Replicas 5" -ForegroundColor Gray
    Write-Host "2. To scale down: Run 03-send-intent.ps1 -Replicas 1" -ForegroundColor Gray
    Write-Host "3. To monitor continuously: Run 05-validate.ps1 -Continuous" -ForegroundColor Gray
    Write-Host "4. To clean up: Run 'make mvp-clean' or 'kubectl delete namespace $Namespace'" -ForegroundColor Gray
}
# Security Hardening Validation Script for Nephoran Intent Operator
# Validates container and Kubernetes security configurations
# Performs comprehensive security checks and compliance verification

[CmdletBinding()]
param(
    [string]$Namespace = "nephoran-system",
    [string]$KubeConfig = $env:KUBECONFIG,
    [string[]]$Services = @("network-intent-controller", "llm-processor", "rag-api", "nephio-bridge", "oran-adaptor"),
    [string]$OutputDir = "security-validation-results",
    [ValidateSet("strict", "moderate", "baseline")]
    [string]$ValidationLevel = "strict",
    [switch]$CisCompliance,
    [switch]$NistCompliance,
    [switch]$GenerateReport,
    [string]$SlackWebhook = $env:SLACK_WEBHOOK_URL
)

# Configuration
$ErrorActionPreference = "Stop"
$ProgressPreference = "Continue"

# Colors for output
$ColorRed = "Red"
$ColorYellow = "Yellow"
$ColorGreen = "Green"
$ColorCyan = "Cyan"
$ColorMagenta = "Magenta"

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

function Test-KubernetesAccess {
    Write-ColorOutput "🔧 Testing Kubernetes access..." $ColorCyan
    
    try {
        $null = kubectl get nodes 2>$null
        Write-ColorOutput "✅ Kubernetes access verified" $ColorGreen
        
        # Check if namespace exists
        $namespaceExists = kubectl get namespace $Namespace 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✅ Namespace '$Namespace' exists" $ColorGreen
        }
        else {
            Write-ColorOutput "❌ Namespace '$Namespace' does not exist" $ColorRed
            return $false
        }
        
        return $true
    }
    catch {
        Write-ColorOutput "❌ Cannot access Kubernetes cluster: $_" $ColorRed
        return $false
    }
}

function Test-PodSecurityContext {
    param([string]$ServiceName)
    
    Write-ColorOutput "🔒 Validating Pod Security Context for $ServiceName..." $ColorCyan
    
    $validationResults = @{
        Service = $ServiceName
        Tests = @()
        OverallStatus = "PASS"
    }
    
    try {
        # Get pod details
        $podJson = kubectl get pods -n $Namespace -l app=$ServiceName -o json | ConvertFrom-Json
        
        if (-not $podJson.items -or $podJson.items.Count -eq 0) {
            Write-ColorOutput "⚠️ No pods found for service $ServiceName" $ColorYellow
            return $validationResults
        }
        
        foreach ($pod in $podJson.items) {
            $podName = $pod.metadata.name
            
            # Test 1: Non-root user
            $runAsNonRoot = $pod.spec.securityContext.runAsNonRoot
            $test1 = @{
                Name = "runAsNonRoot"
                Status = if ($runAsNonRoot -eq $true) { "PASS" } else { "FAIL" }
                Details = "runAsNonRoot: $runAsNonRoot"
                Severity = if ($runAsNonRoot -eq $true) { "INFO" } else { "HIGH" }
            }
            $validationResults.Tests += $test1
            
            # Test 2: User ID validation (should not be 0)
            $runAsUser = $pod.spec.securityContext.runAsUser
            $test2 = @{
                Name = "runAsUser"
                Status = if ($runAsUser -and $runAsUser -ne 0) { "PASS" } else { "FAIL" }
                Details = "runAsUser: $runAsUser"
                Severity = if ($runAsUser -and $runAsUser -ne 0) { "INFO" } else { "CRITICAL" }
            }
            $validationResults.Tests += $test2
            
            # Test 3: Read-only root filesystem
            foreach ($container in $pod.spec.containers) {
                $readOnlyRootFs = $container.securityContext.readOnlyRootFilesystem
                $test3 = @{
                    Name = "readOnlyRootFilesystem-$($container.name)"
                    Status = if ($readOnlyRootFs -eq $true) { "PASS" } else { "FAIL" }
                    Details = "Container $($container.name) readOnlyRootFilesystem: $readOnlyRootFs"
                    Severity = if ($readOnlyRootFs -eq $true) { "INFO" } else { "MEDIUM" }
                }
                $validationResults.Tests += $test3
            }
            
            # Test 4: Capabilities dropped
            foreach ($container in $pod.spec.containers) {
                $capabilities = $container.securityContext.capabilities
                $allDropped = $capabilities -and $capabilities.drop -contains "ALL"
                $test4 = @{
                    Name = "capabilities-$($container.name)"
                    Status = if ($allDropped) { "PASS" } else { "FAIL" }
                    Details = "Container $($container.name) capabilities.drop: $($capabilities.drop -join ',')"
                    Severity = if ($allDropped) { "INFO" } else { "HIGH" }
                }
                $validationResults.Tests += $test4
            }
            
            # Test 5: Privilege escalation prevention
            foreach ($container in $pod.spec.containers) {
                $allowPrivilegeEscalation = $container.securityContext.allowPrivilegeEscalation
                $test5 = @{
                    Name = "allowPrivilegeEscalation-$($container.name)"
                    Status = if ($allowPrivilegeEscalation -eq $false) { "PASS" } else { "FAIL" }
                    Details = "Container $($container.name) allowPrivilegeEscalation: $allowPrivilegeEscalation"
                    Severity = if ($allowPrivilegeEscalation -eq $false) { "INFO" } else { "CRITICAL" }
                }
                $validationResults.Tests += $test5
            }
            
            # Test 6: Seccomp profile
            $seccompProfile = $pod.spec.securityContext.seccompProfile
            $test6 = @{
                Name = "seccompProfile"
                Status = if ($seccompProfile -and $seccompProfile.type -eq "RuntimeDefault") { "PASS" } else { "FAIL" }
                Details = "seccompProfile.type: $($seccompProfile.type)"
                Severity = if ($seccompProfile -and $seccompProfile.type -eq "RuntimeDefault") { "INFO" } else { "MEDIUM" }
            }
            $validationResults.Tests += $test6
        }
    }
    catch {
        Write-ColorOutput "❌ Error validating security context for $ServiceName`: $_" $ColorRed
        $validationResults.OverallStatus = "ERROR"
    }
    
    # Determine overall status
    $failedTests = $validationResults.Tests | Where-Object { $_.Status -eq "FAIL" }
    $criticalFailed = $failedTests | Where-Object { $_.Severity -eq "CRITICAL" }
    $highFailed = $failedTests | Where-Object { $_.Severity -eq "HIGH" }
    
    if ($criticalFailed.Count -gt 0) {
        $validationResults.OverallStatus = "CRITICAL"
        Write-ColorOutput "❌ CRITICAL security issues found for $ServiceName" $ColorRed
    }
    elseif ($highFailed.Count -gt 0) {
        $validationResults.OverallStatus = "HIGH"
        Write-ColorOutput "⚠️ HIGH severity security issues found for $ServiceName" $ColorYellow
    }
    elseif ($failedTests.Count -gt 0) {
        $validationResults.OverallStatus = "MEDIUM"
        Write-ColorOutput "⚠️ MEDIUM severity security issues found for $ServiceName" $ColorYellow
    }
    else {
        Write-ColorOutput "✅ All security context tests passed for $ServiceName" $ColorGreen
    }
    
    return $validationResults
}

function Test-NetworkPolicies {
    Write-ColorOutput "🌐 Validating Network Policies..." $ColorCyan
    
    $networkResults = @{
        Tests = @()
        OverallStatus = "PASS"
    }
    
    try {
        # Test 1: Default deny-all policy exists
        $denyAllPolicy = kubectl get networkpolicy -n $Namespace nephoran-default-deny-all 2>$null
        $test1 = @{
            Name = "default-deny-all-policy"
            Status = if ($LASTEXITCODE -eq 0) { "PASS" } else { "FAIL" }
            Details = "Default deny-all NetworkPolicy existence"
            Severity = if ($LASTEXITCODE -eq 0) { "INFO" } else { "CRITICAL" }
        }
        $networkResults.Tests += $test1
        
        # Test 2: Service-specific network policies
        foreach ($service in $Services) {
            $servicePolicy = kubectl get networkpolicy -n $Namespace "$service-policy" 2>$null
            $test = @{
                Name = "$service-network-policy"
                Status = if ($LASTEXITCODE -eq 0) { "PASS" } else { "FAIL" }
                Details = "NetworkPolicy for $service exists"
                Severity = if ($LASTEXITCODE -eq 0) { "INFO" } else { "HIGH" }
            }
            $networkResults.Tests += $test
        }
        
        # Test 3: Pod Security Standards enforcement
        $namespaceJson = kubectl get namespace $Namespace -o json | ConvertFrom-Json
        $pssEnforce = $namespaceJson.metadata.labels.'pod-security.kubernetes.io/enforce'
        $test3 = @{
            Name = "pod-security-standards-enforce"
            Status = if ($pssEnforce -eq "restricted") { "PASS" } else { "FAIL" }
            Details = "Pod Security Standards enforce level: $pssEnforce"
            Severity = if ($pssEnforce -eq "restricted") { "INFO" } else { "HIGH" }
        }
        $networkResults.Tests += $test3
    }
    catch {
        Write-ColorOutput "❌ Error validating network policies: $_" $ColorRed
        $networkResults.OverallStatus = "ERROR"
    }
    
    # Determine overall status
    $failedTests = $networkResults.Tests | Where-Object { $_.Status -eq "FAIL" }
    $criticalFailed = $failedTests | Where-Object { $_.Severity -eq "CRITICAL" }
    
    if ($criticalFailed.Count -gt 0) {
        $networkResults.OverallStatus = "CRITICAL"
        Write-ColorOutput "❌ CRITICAL network security issues found" $ColorRed
    }
    elseif ($failedTests.Count -gt 0) {
        $networkResults.OverallStatus = "HIGH"
        Write-ColorOutput "⚠️ Network security issues found" $ColorYellow
    }
    else {
        Write-ColorOutput "✅ All network policy tests passed" $ColorGreen
    }
    
    return $networkResults
}

function Test-ResourceLimits {
    Write-ColorOutput "📊 Validating Resource Limits and Quotas..." $ColorCyan
    
    $resourceResults = @{
        Tests = @()
        OverallStatus = "PASS"
    }
    
    try {
        # Test 1: ResourceQuota exists
        $resourceQuota = kubectl get resourcequota -n $Namespace nephoran-security-quota 2>$null
        $test1 = @{
            Name = "resource-quota-exists"
            Status = if ($LASTEXITCODE -eq 0) { "PASS" } else { "FAIL" }
            Details = "ResourceQuota nephoran-security-quota exists"
            Severity = if ($LASTEXITCODE -eq 0) { "INFO" } else { "MEDIUM" }
        }
        $resourceResults.Tests += $test1
        
        # Test 2: LimitRange exists
        $limitRange = kubectl get limitrange -n $Namespace nephoran-security-limits 2>$null
        $test2 = @{
            Name = "limit-range-exists"
            Status = if ($LASTEXITCODE -eq 0) { "PASS" } else { "FAIL" }
            Details = "LimitRange nephoran-security-limits exists"
            Severity = if ($LASTEXITCODE -eq 0) { "INFO" } else { "MEDIUM" }
        }
        $resourceResults.Tests += $test2
        
        # Test 3: Pod resource limits
        foreach ($service in $Services) {
            $podJson = kubectl get pods -n $Namespace -l app=$service -o json | ConvertFrom-Json
            
            foreach ($pod in $podJson.items) {
                foreach ($container in $pod.spec.containers) {
                    $hasLimits = $container.resources -and $container.resources.limits
                    $hasRequests = $container.resources -and $container.resources.requests
                    
                    $test = @{
                        Name = "resource-limits-$service-$($container.name)"
                        Status = if ($hasLimits -and $hasRequests) { "PASS" } else { "FAIL" }
                        Details = "Container $($container.name) has resource limits: $([bool]$hasLimits), requests: $([bool]$hasRequests)"
                        Severity = if ($hasLimits -and $hasRequests) { "INFO" } else { "MEDIUM" }
                    }
                    $resourceResults.Tests += $test
                }
            }
        }
    }
    catch {
        Write-ColorOutput "❌ Error validating resource limits: $_" $ColorRed
        $resourceResults.OverallStatus = "ERROR"
    }
    
    # Determine overall status
    $failedTests = $resourceResults.Tests | Where-Object { $_.Status -eq "FAIL" }
    if ($failedTests.Count -gt 0) {
        $resourceResults.OverallStatus = "MEDIUM"
        Write-ColorOutput "⚠️ Resource limit issues found" $ColorYellow
    }
    else {
        Write-ColorOutput "✅ All resource limit tests passed" $ColorGreen
    }
    
    return $resourceResults
}

function Test-CISCompliance {
    Write-ColorOutput "🛡️ Running CIS Kubernetes Benchmark checks..." $ColorCyan
    
    $cisResults = @{
        Tests = @()
        OverallStatus = "PASS"
        Benchmark = "CIS Kubernetes Benchmark v1.8.0"
    }
    
    # CIS 5.1.1 - Minimize the admission of privileged containers
    $privilegedPods = kubectl get pods -n $Namespace -o json | ConvertFrom-Json |
        ForEach-Object { $_.items } |
        Where-Object { 
            $_.spec.containers | Where-Object { $_.securityContext.privileged -eq $true }
        }
    
    $test1 = @{
        Name = "CIS-5.1.1-no-privileged-containers"
        Status = if (-not $privilegedPods) { "PASS" } else { "FAIL" }
        Details = "Found $($privilegedPods.Count) privileged containers"
        Severity = if (-not $privilegedPods) { "INFO" } else { "CRITICAL" }
        Reference = "CIS 5.1.1"
    }
    $cisResults.Tests += $test1
    
    # CIS 5.1.3 - Minimize the admission of containers with allowPrivilegeEscalation
    $privilegeEscalationPods = kubectl get pods -n $Namespace -o json | ConvertFrom-Json |
        ForEach-Object { $_.items } |
        Where-Object { 
            $_.spec.containers | Where-Object { $_.securityContext.allowPrivilegeEscalation -ne $false }
        }
    
    $test2 = @{
        Name = "CIS-5.1.3-no-privilege-escalation"
        Status = if (-not $privilegeEscalationPods) { "PASS" } else { "FAIL" }
        Details = "Found $($privilegeEscalationPods.Count) containers allowing privilege escalation"
        Severity = if (-not $privilegeEscalationPods) { "INFO" } else { "HIGH" }
        Reference = "CIS 5.1.3"
    }
    $cisResults.Tests += $test2
    
    # CIS 5.1.4 - Minimize the admission of containers with capabilities
    $capabilityPods = kubectl get pods -n $Namespace -o json | ConvertFrom-Json |
        ForEach-Object { $_.items } |
        Where-Object { 
            $_.spec.containers | Where-Object { 
                -not ($_.securityContext.capabilities.drop -contains "ALL")
            }
        }
    
    $test3 = @{
        Name = "CIS-5.1.4-drop-all-capabilities"
        Status = if (-not $capabilityPods) { "PASS" } else { "FAIL" }
        Details = "Found $($capabilityPods.Count) containers not dropping ALL capabilities"
        Severity = if (-not $capabilityPods) { "INFO" } else { "HIGH" }
        Reference = "CIS 5.1.4"
    }
    $cisResults.Tests += $test3
    
    # CIS 5.2.2 - Minimize the admission of containers with readOnlyRootFilesystem set to false
    $writableRootPods = kubectl get pods -n $Namespace -o json | ConvertFrom-Json |
        ForEach-Object { $_.items } |
        Where-Object { 
            $_.spec.containers | Where-Object { $_.securityContext.readOnlyRootFilesystem -ne $true }
        }
    
    $test4 = @{
        Name = "CIS-5.2.2-read-only-root-filesystem"
        Status = if (-not $writableRootPods) { "PASS" } else { "FAIL" }
        Details = "Found $($writableRootPods.Count) containers with writable root filesystem"
        Severity = if (-not $writableRootPods) { "INFO" } else { "MEDIUM" }
        Reference = "CIS 5.2.2"
    }
    $cisResults.Tests += $test4
    
    # CIS 5.2.3 - Minimize the admission of containers with runAsUser of 0
    $rootUserPods = kubectl get pods -n $Namespace -o json | ConvertFrom-Json |
        ForEach-Object { $_.items } |
        Where-Object { 
            $_.spec.securityContext.runAsUser -eq 0 -or
            ($_.spec.containers | Where-Object { $_.securityContext.runAsUser -eq 0 })
        }
    
    $test5 = @{
        Name = "CIS-5.2.3-no-root-user"
        Status = if (-not $rootUserPods) { "PASS" } else { "FAIL" }
        Details = "Found $($rootUserPods.Count) containers running as root (UID 0)"
        Severity = if (-not $rootUserPods) { "INFO" } else { "CRITICAL" }
        Reference = "CIS 5.2.3"
    }
    $cisResults.Tests += $test5
    
    # Determine overall CIS compliance status
    $failedTests = $cisResults.Tests | Where-Object { $_.Status -eq "FAIL" }
    $criticalFailed = $failedTests | Where-Object { $_.Severity -eq "CRITICAL" }
    
    if ($criticalFailed.Count -gt 0) {
        $cisResults.OverallStatus = "CRITICAL"
        Write-ColorOutput "❌ CRITICAL CIS compliance failures found" $ColorRed
    }
    elseif ($failedTests.Count -gt 0) {
        $cisResults.OverallStatus = "NON_COMPLIANT"
        Write-ColorOutput "⚠️ CIS compliance issues found" $ColorYellow
    }
    else {
        $cisResults.OverallStatus = "COMPLIANT"
        Write-ColorOutput "✅ CIS Kubernetes Benchmark compliance verified" $ColorGreen
    }
    
    return $cisResults
}

function Generate-SecurityValidationReport {
    param(
        [array]$SecurityResults,
        [hashtable]$NetworkResults,
        [hashtable]$ResourceResults,
        [hashtable]$CisResults
    )
    
    Write-ColorOutput "📄 Generating security validation report..." $ColorCyan
    
    if (!(Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }
    
    $reportFile = Join-Path $OutputDir "security-validation-report.json"
    $htmlReportFile = Join-Path $OutputDir "security-validation-report.html"
    
    # Calculate overall statistics
    $totalTests = 0
    $passedTests = 0
    $failedTests = 0
    $criticalIssues = 0
    $highIssues = 0
    
    # Count security context tests
    foreach ($result in $SecurityResults) {
        $totalTests += $result.Tests.Count
        $passedTests += ($result.Tests | Where-Object { $_.Status -eq "PASS" }).Count
        $failedTests += ($result.Tests | Where-Object { $_.Status -eq "FAIL" }).Count
        $criticalIssues += ($result.Tests | Where-Object { $_.Severity -eq "CRITICAL" -and $_.Status -eq "FAIL" }).Count
        $highIssues += ($result.Tests | Where-Object { $_.Severity -eq "HIGH" -and $_.Status -eq "FAIL" }).Count
    }
    
    # Count network policy tests
    $totalTests += $NetworkResults.Tests.Count
    $passedTests += ($NetworkResults.Tests | Where-Object { $_.Status -eq "PASS" }).Count
    $failedTests += ($NetworkResults.Tests | Where-Object { $_.Status -eq "FAIL" }).Count
    $criticalIssues += ($NetworkResults.Tests | Where-Object { $_.Severity -eq "CRITICAL" -and $_.Status -eq "FAIL" }).Count
    $highIssues += ($NetworkResults.Tests | Where-Object { $_.Severity -eq "HIGH" -and $_.Status -eq "FAIL" }).Count
    
    # Count resource limit tests
    $totalTests += $ResourceResults.Tests.Count
    $passedTests += ($ResourceResults.Tests | Where-Object { $_.Status -eq "PASS" }).Count
    $failedTests += ($ResourceResults.Tests | Where-Object { $_.Status -eq "FAIL" }).Count
    
    # Count CIS compliance tests
    if ($CisResults) {
        $totalTests += $CisResults.Tests.Count
        $passedTests += ($CisResults.Tests | Where-Object { $_.Status -eq "PASS" }).Count
        $failedTests += ($CisResults.Tests | Where-Object { $_.Status -eq "FAIL" }).Count
        $criticalIssues += ($CisResults.Tests | Where-Object { $_.Severity -eq "CRITICAL" -and $_.Status -eq "FAIL" }).Count
        $highIssues += ($CisResults.Tests | Where-Object { $_.Severity -eq "HIGH" -and $_.Status -eq "FAIL" }).Count
    }
    
    # Determine overall compliance status
    $overallStatus = "COMPLIANT"
    if ($criticalIssues -gt 0) {
        $overallStatus = "CRITICAL_ISSUES"
    }
    elseif ($highIssues -gt 5) {
        $overallStatus = "HIGH_RISK"
    }
    elseif ($failedTests -gt 0) {
        $overallStatus = "NON_COMPLIANT"
    }
    
    $report = @{
        GeneratedAt = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
        Namespace = $Namespace
        ValidationLevel = $ValidationLevel
        OverallStatus = $overallStatus
        Summary = @{
            TotalTests = $totalTests
            PassedTests = $passedTests
            FailedTests = $failedTests
            CriticalIssues = $criticalIssues
            HighIssues = $highIssues
            CompliancePercentage = if ($totalTests -gt 0) { [math]::Round(($passedTests / $totalTests) * 100, 2) } else { 0 }
        }
        SecurityContextResults = $SecurityResults
        NetworkPolicyResults = $NetworkResults
        ResourceLimitResults = $ResourceResults
        CISComplianceResults = if ($CisResults) { $CisResults } else { $null }
        Recommendations = @()
    }
    
    # Add recommendations based on findings
    if ($criticalIssues -gt 0) {
        $report.Recommendations += "🚨 IMMEDIATE ACTION REQUIRED: $criticalIssues critical security issues found"
    }
    if ($highIssues -gt 0) {
        $report.Recommendations += "⚠️ Address $highIssues high-severity security issues"
    }
    if ($report.Summary.CompliancePercentage -lt 90) {
        $report.Recommendations += "📈 Improve security compliance - currently at $($report.Summary.CompliancePercentage)%"
    }
    
    # Save JSON report
    $report | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFile -Encoding UTF8
    Write-ColorOutput "✅ JSON report saved: $reportFile" $ColorGreen
    
    # Generate HTML report (simplified version)
    $htmlContent = @"
<!DOCTYPE html>
<html>
<head>
    <title>Nephoran Security Validation Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .status-critical { color: #d32f2f; font-weight: bold; }
        .status-compliant { color: #388e3c; font-weight: bold; }
        .status-non-compliant { color: #f57c00; font-weight: bold; }
        .summary { display: flex; gap: 20px; margin: 20px 0; }
        .metric { background: #e8f4fd; padding: 15px; border-radius: 5px; flex: 1; text-align: center; }
        .recommendations { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🛡️ Nephoran Security Validation Report</h1>
        <p><strong>Generated:</strong> $(Get-Date -Format "yyyy-MM-dd HH:mm:ss UTC")</p>
        <p><strong>Namespace:</strong> $Namespace</p>
        <p><strong>Validation Level:</strong> $ValidationLevel</p>
        <p class="status-$($overallStatus.ToLower())"><strong>Overall Status:</strong> $overallStatus</p>
    </div>
    
    <div class="summary">
        <div class="metric">
            <h3>$($report.Summary.TotalTests)</h3>
            <p>Total Tests</p>
        </div>
        <div class="metric">
            <h3>$($report.Summary.PassedTests)</h3>
            <p>Passed Tests</p>
        </div>
        <div class="metric">
            <h3>$($report.Summary.FailedTests)</h3>
            <p>Failed Tests</p>
        </div>
        <div class="metric">
            <h3>$($report.Summary.CompliancePercentage)%</h3>
            <p>Compliance</p>
        </div>
    </div>
    
    <div class="recommendations">
        <h3>🎯 Recommendations</h3>
        <ul>
"@
    foreach ($recommendation in $report.Recommendations) {
        $htmlContent += "<li>$recommendation</li>`n"
    }
    
    $htmlContent += @"
        </ul>
    </div>
</body>
</html>
"@
    
    $htmlContent | Out-File -FilePath $htmlReportFile -Encoding UTF8
    Write-ColorOutput "✅ HTML report saved: $htmlReportFile" $ColorGreen
    
    return $report
}

# Main execution
function Main {
    try {
        Write-ColorOutput "🚀 Starting Nephoran Security Hardening Validation" $ColorCyan
        Write-ColorOutput "Namespace: $Namespace" $ColorCyan
        Write-ColorOutput "Validation Level: $ValidationLevel" $ColorCyan
        Write-ColorOutput "Services: $($Services -join ', ')" $ColorCyan
        
        # Test Kubernetes access
        if (-not (Test-KubernetesAccess)) {
            exit 1
        }
        
        # Run security context validation for each service
        $securityResults = @()
        foreach ($service in $Services) {
            $result = Test-PodSecurityContext -ServiceName $service
            $securityResults += $result
        }
        
        # Test network policies
        $networkResults = Test-NetworkPolicies
        
        # Test resource limits
        $resourceResults = Test-ResourceLimits
        
        # Run CIS compliance checks if requested
        $cisResults = $null
        if ($CisCompliance) {
            $cisResults = Test-CISCompliance
        }
        
        # Generate comprehensive report
        if ($GenerateReport) {
            $report = Generate-SecurityValidationReport -SecurityResults $securityResults -NetworkResults $networkResults -ResourceResults $resourceResults -CisResults $cisResults
        }
        
        # Summary output
        Write-ColorOutput "`n📊 Security Validation Summary:" $ColorCyan
        
        $totalCritical = 0
        $totalHigh = 0
        $totalFailed = 0
        
        foreach ($result in $securityResults) {
            $criticalCount = ($result.Tests | Where-Object { $_.Severity -eq "CRITICAL" -and $_.Status -eq "FAIL" }).Count
            $highCount = ($result.Tests | Where-Object { $_.Severity -eq "HIGH" -and $_.Status -eq "FAIL" }).Count
            $failedCount = ($result.Tests | Where-Object { $_.Status -eq "FAIL" }).Count
            
            $totalCritical += $criticalCount
            $totalHigh += $highCount
            $totalFailed += $failedCount
            
            Write-ColorOutput "$($result.Service): $($result.OverallStatus)" $(
                switch ($result.OverallStatus) {
                    "CRITICAL" { $ColorRed }
                    "HIGH" { $ColorYellow }
                    "MEDIUM" { $ColorYellow }
                    default { $ColorGreen }
                }
            )
        }
        
        Write-ColorOutput "`nOverall Results:" $ColorCyan
        Write-ColorOutput "Critical Issues: $totalCritical" $(if ($totalCritical -gt 0) { $ColorRed } else { $ColorGreen })
        Write-ColorOutput "High Issues: $totalHigh" $(if ($totalHigh -gt 0) { $ColorYellow } else { $ColorGreen })
        Write-ColorOutput "Total Failed Tests: $totalFailed" $(if ($totalFailed -gt 0) { $ColorYellow } else { $ColorGreen })
        
        # Exit with appropriate code
        if ($totalCritical -gt 0) {
            Write-ColorOutput "`n❌ CRITICAL security issues found. Deployment should be blocked." $ColorRed
            exit 1
        }
        elseif ($totalHigh -gt 5) {
            Write-ColorOutput "`n⚠️ Too many HIGH severity issues found." $ColorYellow
            exit 2
        }
        elseif ($totalFailed -gt 10) {
            Write-ColorOutput "`n⚠️ Too many failed security tests." $ColorYellow
            exit 3
        }
        else {
            Write-ColorOutput "`n✅ Security validation completed successfully." $ColorGreen
            exit 0
        }
    }
    catch {
        Write-ColorOutput "❌ Validation failed: $_" $ColorRed
        Write-ColorOutput $_.Exception.StackTrace $ColorRed
        exit 1
    }
}

# Execute main function
Main
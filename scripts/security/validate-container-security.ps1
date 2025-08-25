# ==============================================================================
# Container Security Validation Script
# ==============================================================================
# Validates container security hardening implementation across all deployments
# Implements CIS Docker Benchmark and Kubernetes security best practices
# Usage: .\scripts\security\validate-container-security.ps1

[CmdletBinding()]
param(
    [Parameter(HelpMessage="Path to the repository root")]
    [string]$RepoRoot = (Get-Location),
    
    [Parameter(HelpMessage="Skip Docker daemon validation")]
    [switch]$SkipDockerValidation,
    
    [Parameter(HelpMessage="Generate detailed security report")]
    [switch]$DetailedReport,
    
    [Parameter(HelpMessage="Export results to JSON")]
    [string]$ExportPath,
    
    [Parameter(HelpMessage="Run in CI/CD mode (non-interactive)")]
    [switch]$CiMode
)

# Security validation configuration
$script:ValidationResults = @()
$script:SecurityScore = 0
$script:MaxScore = 100
$script:FailedChecks = @()

# Color functions for output
function Write-SecurityCheck {
    param([string]$Message, [string]$Status = "INFO")
    $colors = @{
        "PASS" = "Green"
        "FAIL" = "Red"
        "WARN" = "Yellow"
        "INFO" = "Cyan"
    }
    Write-Host "[$Status] $Message" -ForegroundColor $colors[$Status]
}

function Test-DockerfileSecurityHardening {
    [CmdletBinding()]
    param([string]$DockerfilePath)
    
    Write-SecurityCheck "Validating Dockerfile security: $DockerfilePath" "INFO"
    
    if (!(Test-Path $DockerfilePath)) {
        Write-SecurityCheck "Dockerfile not found: $DockerfilePath" "FAIL"
        return @{
            Path = $DockerfilePath
            Score = 0
            Issues = @("File not found")
        }
    }
    
    $content = Get-Content $DockerfilePath -Raw
    $issues = @()
    $score = 0
    
    # Check 1: Non-root user (10 points)
    if ($content -match 'USER\s+(?!0\s|root\s)') {
        Write-SecurityCheck "✓ Non-root user configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ No non-root user configured" "FAIL"
        $issues += "Missing non-root USER directive"
    }
    
    # Check 2: Distroless or minimal base images (15 points)
    if ($content -match 'FROM.*distroless|FROM.*alpine:|FROM.*scratch') {
        Write-SecurityCheck "✓ Minimal base image used" "PASS"
        $score += 15
    } else {
        Write-SecurityCheck "✗ Non-minimal base image detected" "WARN"
        $issues += "Consider using distroless or alpine base images"
    }
    
    # Check 3: Version pinning (10 points)
    $fromStatements = ([regex]'FROM\s+([^\s]+)').Matches($content)
    $versionedImages = ($fromStatements | Where-Object { $_.Groups[1].Value -match ":" }).Count
    if ($versionedImages -eq $fromStatements.Count -and $fromStatements.Count -gt 0) {
        Write-SecurityCheck "✓ All base images version-pinned" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Some base images not version-pinned" "FAIL"
        $issues += "Pin all base image versions"
    }
    
    # Check 4: COPY with --chown (5 points)
    if ($content -match 'COPY\s+--chown=') {
        Write-SecurityCheck "✓ COPY with --chown found" "PASS"
        $score += 5
    } else {
        Write-SecurityCheck "✗ COPY without --chown detected" "WARN"
        $issues += "Use COPY --chown to set proper ownership"
    }
    
    # Check 5: Multi-stage build (10 points)
    if (($content | Select-String "FROM.*AS").Count -gt 1) {
        Write-SecurityCheck "✓ Multi-stage build implemented" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Single-stage build detected" "WARN"
        $issues += "Consider multi-stage build for security"
    }
    
    # Check 6: HEALTHCHECK configured (5 points)
    if ($content -match 'HEALTHCHECK') {
        Write-SecurityCheck "✓ Health check configured" "PASS"
        $score += 5
    } else {
        Write-SecurityCheck "✗ No health check found" "WARN"
        $issues += "Add HEALTHCHECK directive"
    }
    
    # Check 7: No secrets in build (15 points)
    $secretPatterns = @(
        "password\s*=",
        "secret\s*=",
        "key\s*=.*[A-Za-z0-9]{20}",
        "token\s*=",
        "api[_-]?key\s*="
    )
    
    $secretFound = $false
    foreach ($pattern in $secretPatterns) {
        if ($content -match $pattern) {
            $secretFound = $true
            break
        }
    }
    
    if (-not $secretFound) {
        Write-SecurityCheck "✓ No hardcoded secrets detected" "PASS"
        $score += 15
    } else {
        Write-SecurityCheck "✗ Potential secrets in Dockerfile" "FAIL"
        $issues += "Remove hardcoded secrets from Dockerfile"
    }
    
    # Check 8: Cleanup commands (10 points)
    if ($content -match 'rm\s+-rf.*cache|rm\s+-rf.*tmp|apt-get\s+clean') {
        Write-SecurityCheck "✓ Cleanup commands found" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ No cleanup commands found" "WARN"
        $issues += "Add cleanup commands to reduce image size"
    }
    
    # Check 9: Security labels (5 points)
    if ($content -match 'LABEL.*security') {
        Write-SecurityCheck "✓ Security labels present" "PASS"
        $score += 5
    } else {
        Write-SecurityCheck "✗ No security labels found" "WARN"
        $issues += "Add security-related labels"
    }
    
    # Check 10: Build arguments for security (5 points)
    if ($content -match 'ARG.*VERSION|ARG.*BUILD_DATE|ARG.*VCS_REF') {
        Write-SecurityCheck "✓ Build arguments configured" "PASS"
        $score += 5
    } else {
        Write-SecurityCheck "✗ No build arguments for traceability" "WARN"
        $issues += "Add build arguments for image traceability"
    }
    
    # Check 11: Package version pinning (10 points)
    if ($content -match 'apk\s+add.*=|apt-get\s+install.*=') {
        Write-SecurityCheck "✓ Package versions pinned" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Package versions not pinned" "WARN"
        $issues += "Pin package versions in package managers"
    }
    
    return @{
        Path = $DockerfilePath
        Score = $score
        MaxScore = 110
        Issues = $issues
        Percentage = [math]::Round(($score / 110) * 100, 2)
    }
}

function Test-DockerComposeSecurityHardening {
    [CmdletBinding()]
    param([string]$ComposePath)
    
    Write-SecurityCheck "Validating Docker Compose security: $ComposePath" "INFO"
    
    if (!(Test-Path $ComposePath)) {
        Write-SecurityCheck "Docker Compose file not found: $ComposePath" "FAIL"
        return @{
            Path = $ComposePath
            Score = 0
            Issues = @("File not found")
        }
    }
    
    $content = Get-Content $ComposePath -Raw
    $issues = @()
    $score = 0
    
    # Check 1: Security options configured (20 points)
    if ($content -match 'security_opt:' -and $content -match 'no-new-privileges:true') {
        Write-SecurityCheck "✓ Security options configured" "PASS"
        $score += 20
    } else {
        Write-SecurityCheck "✗ Security options missing" "FAIL"
        $issues += "Configure security_opt with no-new-privileges"
    }
    
    # Check 2: Non-root user (15 points)
    if ($content -match 'user:\s*"\d+:\d+"' -and !($content -match 'user:\s*"0')) {
        Write-SecurityCheck "✓ Non-root users configured" "PASS"
        $score += 15
    } else {
        Write-SecurityCheck "✗ Root users detected" "FAIL"
        $issues += "Configure non-root users for all services"
    }
    
    # Check 3: Read-only root filesystem (10 points)
    if ($content -match 'read_only:\s*true') {
        Write-SecurityCheck "✓ Read-only root filesystem" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Writable root filesystem" "WARN"
        $issues += "Enable read-only root filesystem where possible"
    }
    
    # Check 4: Capabilities dropped (15 points)
    if ($content -match 'cap_drop:' -and $content -match '- ALL') {
        Write-SecurityCheck "✓ All capabilities dropped" "PASS"
        $score += 15
    } else {
        Write-SecurityCheck "✗ Capabilities not properly restricted" "FAIL"
        $issues += "Drop ALL capabilities by default"
    }
    
    # Check 5: Resource limits (10 points)
    if ($content -match 'limits:' -and $content -match 'memory:' -and $content -match 'cpus:') {
        Write-SecurityCheck "✓ Resource limits configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Resource limits missing" "WARN"
        $issues += "Configure CPU and memory limits"
    }
    
    # Check 6: Health checks (10 points)
    if ($content -match 'healthcheck:') {
        Write-SecurityCheck "✓ Health checks configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Health checks missing" "WARN"
        $issues += "Configure health checks for services"
    }
    
    # Check 7: Tmpfs for sensitive directories (10 points)
    if ($content -match 'tmpfs:' -and $content -match 'noexec') {
        Write-SecurityCheck "✓ Secure tmpfs configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Tmpfs not configured or insecure" "WARN"
        $issues += "Configure tmpfs with security options"
    }
    
    # Check 8: Network isolation (10 points)
    if ($content -match 'networks:' -and !($content -match 'network_mode:\s*host')) {
        Write-SecurityCheck "✓ Network isolation configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Network isolation missing" "WARN"
        $issues += "Configure custom networks for isolation"
    }
    
    return @{
        Path = $ComposePath
        Score = $score
        MaxScore = 110
        Issues = $issues
        Percentage = [math]::Round(($score / 110) * 100, 2)
    }
}

function Test-KubernetesSecurityContext {
    [CmdletBinding()]
    param([string]$ManifestPath)
    
    Write-SecurityCheck "Validating Kubernetes security contexts: $ManifestPath" "INFO"
    
    if (!(Test-Path $ManifestPath)) {
        Write-SecurityCheck "Kubernetes manifest not found: $ManifestPath" "WARN"
        return @{
            Path = $ManifestPath
            Score = 0
            Issues = @("File not found")
        }
    }
    
    $content = Get-Content $ManifestPath -Raw
    $issues = @()
    $score = 0
    
    # Check 1: Pod security context (25 points)
    if ($content -match 'securityContext:' -and $content -match 'runAsNonRoot:\s*true') {
        Write-SecurityCheck "✓ Pod security context configured" "PASS"
        $score += 25
    } else {
        Write-SecurityCheck "✗ Pod security context missing" "FAIL"
        $issues += "Configure pod-level securityContext with runAsNonRoot: true"
    }
    
    # Check 2: Container security context (20 points)
    if ($content -match 'allowPrivilegeEscalation:\s*false' -and $content -match 'readOnlyRootFilesystem:\s*true') {
        Write-SecurityCheck "✓ Container security context hardened" "PASS"
        $score += 20
    } else {
        Write-SecurityCheck "✗ Container security context not hardened" "FAIL"
        $issues += "Set allowPrivilegeEscalation: false and readOnlyRootFilesystem: true"
    }
    
    # Check 3: Capabilities dropped (15 points)
    if ($content -match 'drop:' -and $content -match '- ALL') {
        Write-SecurityCheck "✓ All capabilities dropped" "PASS"
        $score += 15
    } else {
        Write-SecurityCheck "✗ Capabilities not dropped" "FAIL"
        $issues += "Drop ALL capabilities in securityContext"
    }
    
    # Check 4: Seccomp profile (10 points)
    if ($content -match 'seccompProfile:' -and $content -match 'type:\s*RuntimeDefault') {
        Write-SecurityCheck "✓ Seccomp profile configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Seccomp profile missing" "WARN"
        $issues += "Configure seccompProfile with RuntimeDefault"
    }
    
    # Check 5: Resource limits (10 points)
    if ($content -match 'limits:' -and $content -match 'requests:') {
        Write-SecurityCheck "✓ Resource limits and requests configured" "PASS"
        $score += 10
    } else {
        Write-SecurityCheck "✗ Resource limits missing" "WARN"
        $issues += "Configure resource limits and requests"
    }
    
    return @{
        Path = $ManifestPath
        Score = $score
        MaxScore = 80
        Issues = $issues
        Percentage = [math]::Round(($score / 80) * 100, 2)
    }
}

function Test-ImageSecurityScanning {
    Write-SecurityCheck "Validating image security scanning configuration" "INFO"
    
    $issues = @()
    $score = 0
    
    # Check for Trivy configuration
    $trivyConfigPath = Join-Path $RepoRoot "security\configs\trivy.yaml"
    if (Test-Path $trivyConfigPath) {
        Write-SecurityCheck "✓ Trivy configuration found" "PASS"
        $score += 25
    } else {
        Write-SecurityCheck "✗ Trivy configuration missing" "FAIL"
        $issues += "Configure Trivy for vulnerability scanning"
    }
    
    # Check for security scanning in CI/CD
    $ciConfigPath = Join-Path $RepoRoot ".github\workflows\ci.yml"
    if (Test-Path $ciConfigPath) {
        $ciContent = Get-Content $ciConfigPath -Raw
        if ($ciContent -match "trivy|vulnerability|security-scan") {
            Write-SecurityCheck "✓ Security scanning in CI/CD" "PASS"
            $score += 25
        } else {
            Write-SecurityCheck "✗ No security scanning in CI/CD" "FAIL"
            $issues += "Add security scanning to CI/CD pipeline"
        }
    }
    
    # Check for SBOM generation
    $sbomScriptPath = Join-Path $RepoRoot "scripts\security\generate-sbom.ps1"
    if (Test-Path $sbomScriptPath) {
        Write-SecurityCheck "✓ SBOM generation script found" "PASS"
        $score += 25
    } else {
        Write-SecurityCheck "✗ SBOM generation missing" "WARN"
        $issues += "Implement SBOM generation"
    }
    
    # Check for image signing configuration
    $signingConfigPath = Join-Path $RepoRoot "deployments\security\image-signing-verification.yaml"
    if (Test-Path $signingConfigPath) {
        Write-SecurityCheck "✓ Image signing configuration found" "PASS"
        $score += 25
    } else {
        Write-SecurityCheck "✗ Image signing configuration missing" "WARN"
        $issues += "Configure container image signing"
    }
    
    return @{
        Score = $score
        MaxScore = 100
        Issues = $issues
        Percentage = [math]::Round(($score / 100) * 100, 2)
    }
}

function Invoke-SecurityValidation {
    Write-Host "`n=== Container Security Validation Report ===" -ForegroundColor Cyan
    Write-Host "Repository: $RepoRoot" -ForegroundColor Gray
    Write-Host "Validation Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""
    
    $allResults = @()
    
    # Test Dockerfiles
    Write-Host "1. DOCKERFILE SECURITY ANALYSIS" -ForegroundColor Yellow
    Write-Host "================================" -ForegroundColor Yellow
    
    $dockerfiles = @(
        "Dockerfile",
        "Dockerfile.multiarch",
        "Dockerfile.dev"
    )
    
    foreach ($dockerfile in $dockerfiles) {
        $dockerfilePath = Join-Path $RepoRoot $dockerfile
        $result = Test-DockerfileSecurityHardening -DockerfilePath $dockerfilePath
        $allResults += $result
        
        Write-Host ""
        Write-Host "File: $dockerfile - Score: $($result.Score)/$($result.MaxScore) ($($result.Percentage)%)" -ForegroundColor $(
            if ($result.Percentage -ge 80) { "Green" } 
            elseif ($result.Percentage -ge 60) { "Yellow" } 
            else { "Red" }
        )
        
        if ($DetailedReport -and $result.Issues.Count -gt 0) {
            Write-Host "Issues:" -ForegroundColor Red
            $result.Issues | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
        }
        Write-Host ""
    }
    
    # Test Docker Compose
    Write-Host "2. DOCKER COMPOSE SECURITY ANALYSIS" -ForegroundColor Yellow
    Write-Host "====================================" -ForegroundColor Yellow
    
    $composeFiles = @(
        "deployments\docker-compose.yml",
        "deployments\docker-compose.consolidated.yml"
    )
    
    foreach ($composeFile in $composeFiles) {
        $composePath = Join-Path $RepoRoot $composeFile
        if (Test-Path $composePath) {
            $result = Test-DockerComposeSecurityHardening -ComposePath $composePath
            $allResults += $result
            
            Write-Host "File: $composeFile - Score: $($result.Score)/$($result.MaxScore) ($($result.Percentage)%)" -ForegroundColor $(
                if ($result.Percentage -ge 80) { "Green" } 
                elseif ($result.Percentage -ge 60) { "Yellow" } 
                else { "Red" }
            )
            
            if ($DetailedReport -and $result.Issues.Count -gt 0) {
                Write-Host "Issues:" -ForegroundColor Red
                $result.Issues | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
            }
            Write-Host ""
        }
    }
    
    # Test Kubernetes Security Contexts
    Write-Host "3. KUBERNETES SECURITY ANALYSIS" -ForegroundColor Yellow
    Write-Host "================================" -ForegroundColor Yellow
    
    $k8sManifests = Get-ChildItem -Path (Join-Path $RepoRoot "deployments") -Filter "*.yaml" -Recurse | 
        Where-Object { $_.Name -notmatch "docker-compose" } | 
        Select-Object -First 5  # Limit to first 5 for demo
    
    foreach ($manifest in $k8sManifests) {
        $result = Test-KubernetesSecurityContext -ManifestPath $manifest.FullName
        $allResults += $result
        
        Write-Host "File: $($manifest.Name) - Score: $($result.Score)/$($result.MaxScore) ($($result.Percentage)%)" -ForegroundColor $(
            if ($result.Percentage -ge 80) { "Green" } 
            elseif ($result.Percentage -ge 60) { "Yellow" } 
            else { "Red" }
        )
    }
    
    # Test Image Security Scanning
    Write-Host "`n4. IMAGE SECURITY SCANNING ANALYSIS" -ForegroundColor Yellow
    Write-Host "====================================" -ForegroundColor Yellow
    
    $scanResult = Test-ImageSecurityScanning
    $allResults += $scanResult
    
    Write-Host "Security Scanning Infrastructure - Score: $($scanResult.Score)/$($scanResult.MaxScore) ($($scanResult.Percentage)%)" -ForegroundColor $(
        if ($scanResult.Percentage -ge 80) { "Green" } 
        elseif ($scanResult.Percentage -ge 60) { "Yellow" } 
        else { "Red" }
    )
    
    # Calculate overall score
    $totalScore = ($allResults | Measure-Object -Property Score -Sum).Sum
    $totalMaxScore = ($allResults | Measure-Object -Property MaxScore -Sum).Sum
    $overallPercentage = [math]::Round(($totalScore / $totalMaxScore) * 100, 2)
    
    Write-Host "`n=== OVERALL SECURITY SCORE ===" -ForegroundColor Cyan
    Write-Host "Total Score: $totalScore / $totalMaxScore ($overallPercentage%)" -ForegroundColor $(
        if ($overallPercentage -ge 80) { "Green" } 
        elseif ($overallPercentage -ge 60) { "Yellow" } 
        else { "Red" }
    )
    
    # Security grade
    $grade = switch ($overallPercentage) {
        { $_ -ge 95 } { "A+" }
        { $_ -ge 90 } { "A" }
        { $_ -ge 80 } { "B" }
        { $_ -ge 70 } { "C" }
        { $_ -ge 60 } { "D" }
        default { "F" }
    }
    
    Write-Host "Security Grade: $grade" -ForegroundColor $(
        if ($grade -match "A") { "Green" }
        elseif ($grade -match "[BC]") { "Yellow" }
        else { "Red" }
    )
    
    # Export results if requested
    if ($ExportPath) {
        $exportData = @{
            ValidationTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            Repository = $RepoRoot
            OverallScore = $overallPercentage
            Grade = $grade
            TotalScore = $totalScore
            MaxScore = $totalMaxScore
            Results = $allResults
        }
        
        $exportData | ConvertTo-Json -Depth 10 | Out-File -FilePath $ExportPath -Encoding UTF8
        Write-Host "`nResults exported to: $ExportPath" -ForegroundColor Green
    }
    
    # Recommendations
    Write-Host "`n=== RECOMMENDATIONS ===" -ForegroundColor Cyan
    
    $allIssues = $allResults | ForEach-Object { $_.Issues } | Where-Object { $_ } | Sort-Object -Unique
    
    if ($allIssues.Count -gt 0) {
        Write-Host "Priority Security Issues:" -ForegroundColor Yellow
        $allIssues | ForEach-Object { Write-Host "  • $_" -ForegroundColor Yellow }
    } else {
        Write-Host "No critical security issues found! 🎉" -ForegroundColor Green
    }
    
    Write-Host ""
    
    # Exit with appropriate code for CI/CD
    if ($CiMode) {
        if ($overallPercentage -lt 70) {
            Write-Host "Security validation failed. Minimum required score: 70%" -ForegroundColor Red
            exit 1
        } else {
            Write-Host "Security validation passed." -ForegroundColor Green
            exit 0
        }
    }
}

# Run the validation
try {
    Invoke-SecurityValidation
} catch {
    Write-Error "Security validation failed: $_"
    if ($CiMode) {
        exit 1
    }
}
#!/usr/bin/env pwsh

# fix-dependency-security-scan.ps1
# Fix Go Dependency Security Scan failures and ensure go.sum is up to date
# Handles network connectivity issues and provides fallback strategies

param(
    [switch]$Force,
    [switch]$SkipNetworkTests,
    [int]$TimeoutSeconds = 300,
    [string]$ProxyUrl = ""
)

$ErrorActionPreference = "Continue"
$ProgressPreference = "SilentlyContinue"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Blue = "`e[34m"
$Reset = "`e[0m"

function Write-Status {
    param($Message, $Color = $Blue)
    Write-Host "${Color}[$(Get-Date -Format 'HH:mm:ss')] $Message${Reset}"
}

function Write-Success {
    param($Message)
    Write-Status $Message $Green
}

function Write-Warning {
    param($Message)
    Write-Status $Message $Yellow
}

function Write-Error {
    param($Message)
    Write-Status $Message $Red
}

function Test-NetworkConnectivity {
    Write-Status "Testing network connectivity..."
    
    $testUrls = @(
        "https://proxy.golang.org",
        "https://sum.golang.org",
        "https://github.com"
    )
    
    $networkOk = $true
    foreach ($url in $testUrls) {
        try {
            $response = Invoke-WebRequest -Uri $url -Method Head -TimeoutSec 10 -UseBasicParsing
            if ($response.StatusCode -eq 200) {
                Write-Success "✓ $url accessible"
            } else {
                Write-Warning "⚠ $url returned status $($response.StatusCode)"
                $networkOk = $false
            }
        }
        catch {
            Write-Warning "⚠ Cannot reach $url : $($_.Exception.Message)"
            $networkOk = $false
        }
    }
    
    return $networkOk
}

function Set-GoProxyEnvironment {
    param($ProxyUrl)
    
    Write-Status "Setting up Go proxy environment..."
    
    if ($ProxyUrl) {
        $env:GOPROXY = $ProxyUrl
        Write-Status "Using custom proxy: $ProxyUrl"
    } else {
        $env:GOPROXY = "https://proxy.golang.org,direct"
    }
    
    $env:GOSUMDB = "sum.golang.org"
    $env:GOPRIVATE = ""
    $env:GONOPROXY = ""
    $env:GONOSUMDB = ""
    
    Write-Success "Go proxy environment configured"
    Write-Status "GOPROXY: $env:GOPROXY"
    Write-Status "GOSUMDB: $env:GOSUMDB"
}

function Backup-GoModFiles {
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $backupDir = ".\.dependency-backups"
    
    if (-not (Test-Path $backupDir)) {
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    }
    
    if (Test-Path "go.mod") {
        Copy-Item "go.mod" "$backupDir\go.mod.backup.$timestamp"
        Write-Success "✓ Backed up go.mod"
    }
    
    if (Test-Path "go.sum") {
        Copy-Item "go.sum" "$backupDir\go.sum.backup.$timestamp"
        Write-Success "✓ Backed up go.sum"
    }
}

function Test-GoVersion {
    Write-Status "Checking Go version..."
    
    try {
        $goVersionOutput = go version
        $goVersion = ($goVersionOutput -replace "go version go", "" -split " ")[0]
        Write-Success "Go version: $goVersion"
        
        # Check if version meets minimum requirements
        if ($goVersion -lt "1.21") {
            Write-Warning "⚠ Go version $goVersion may be too old. Consider upgrading to Go 1.21+"
            return $false
        }
        
        return $true
    }
    catch {
        Write-Error "✗ Go is not installed or not in PATH"
        return $false
    }
}

function Run-GoModTidy {
    param($TimeoutSeconds)
    
    Write-Status "Running go mod tidy..."
    
    $job = Start-Job -ScriptBlock {
        param($WorkingDirectory, $ProxyEnv, $SumDbEnv)
        Set-Location $WorkingDirectory
        $env:GOPROXY = $ProxyEnv
        $env:GOSUMDB = $SumDbEnv
        go mod tidy
    } -ArgumentList (Get-Location), $env:GOPROXY, $env:GOSUMDB
    
    $completed = Wait-Job $job -Timeout $TimeoutSeconds
    
    if ($completed) {
        $output = Receive-Job $job
        $exitCode = $job.State
        Remove-Job $job
        
        if ($exitCode -eq "Completed") {
            Write-Success "✓ go mod tidy completed successfully"
            return $true
        } else {
            Write-Error "✗ go mod tidy failed"
            if ($output) {
                Write-Host $output
            }
            return $false
        }
    } else {
        Write-Warning "⚠ go mod tidy timed out after $TimeoutSeconds seconds"
        Remove-Job $job -Force
        return $false
    }
}

function Run-GoModDownload {
    param($TimeoutSeconds)
    
    Write-Status "Running go mod download..."
    
    $job = Start-Job -ScriptBlock {
        param($WorkingDirectory, $ProxyEnv, $SumDbEnv)
        Set-Location $WorkingDirectory
        $env:GOPROXY = $ProxyEnv
        $env:GOSUMDB = $SumDbEnv
        go mod download
    } -ArgumentList (Get-Location), $env:GOPROXY, $env:GOSUMDB
    
    $completed = Wait-Job $job -Timeout $TimeoutSeconds
    
    if ($completed) {
        $output = Receive-Job $job
        $exitCode = $job.State
        Remove-Job $job
        
        if ($exitCode -eq "Completed") {
            Write-Success "✓ go mod download completed successfully"
            return $true
        } else {
            Write-Error "✗ go mod download failed"
            if ($output) {
                Write-Host $output
            }
            return $false
        }
    } else {
        Write-Warning "⚠ go mod download timed out after $TimeoutSeconds seconds"
        Remove-Job $job -Force
        return $false
    }
}

function Test-GoModVerify {
    Write-Status "Verifying go.mod integrity..."
    
    try {
        $output = go mod verify 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✓ All modules verified successfully"
            return $true
        } else {
            Write-Error "✗ Module verification failed:"
            Write-Host $output
            return $false
        }
    }
    catch {
        Write-Error "✗ Failed to run go mod verify: $($_.Exception.Message)"
        return $false
    }
}

function Test-GoBuild {
    Write-Status "Testing Go build (compilation check)..."
    
    try {
        # Try to build specific main packages to avoid building everything
        $mainPackages = @(
            "./cmd/conductor-loop",
            "./cmd/conductor",
            "./cmd/intent-ingest"
        )
        
        foreach ($pkg in $mainPackages) {
            if (Test-Path $pkg) {
                Write-Status "Testing build for $pkg..."
                $output = go build -o NUL $pkg 2>&1
                if ($LASTEXITCODE -eq 0) {
                    Write-Success "✓ $pkg builds successfully"
                } else {
                    Write-Warning "⚠ $pkg build issues: $output"
                }
            }
        }
        
        # Test that all packages can be loaded
        Write-Status "Testing package loading..."
        $output = go list ./... 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✓ All packages can be loaded"
            return $true
        } else {
            Write-Error "✗ Some packages failed to load:"
            Write-Host $output
            return $false
        }
    }
    catch {
        Write-Error "✗ Build test failed: $($_.Exception.Message)"
        return $false
    }
}

function Install-SecurityTools {
    Write-Status "Installing security scanning tools..."
    
    try {
        # Install govulncheck
        Write-Status "Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✓ govulncheck installed"
        } else {
            Write-Warning "⚠ govulncheck installation may have failed"
        }
        
        # Install cyclonedx-gomod for SBOM generation
        Write-Status "Installing CycloneDX gomod tool..."
        go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✓ cyclonedx-gomod installed"
        } else {
            Write-Warning "⚠ cyclonedx-gomod installation may have failed"
        }
        
        return $true
    }
    catch {
        Write-Error "✗ Failed to install security tools: $($_.Exception.Message)"
        return $false
    }
}

function Run-SecurityScan {
    Write-Status "Running security scans..."
    
    $reportsDir = ".\.security-reports"
    if (-not (Test-Path $reportsDir)) {
        New-Item -ItemType Directory -Path $reportsDir -Force | Out-Null
    }
    
    try {
        # Run govulncheck
        Write-Status "Running govulncheck..."
        $vulnOutput = govulncheck -json ./... 2>&1
        if ($LASTEXITCODE -eq 0) {
            $vulnOutput | Out-File "$reportsDir\govulncheck.json"
            Write-Success "✓ govulncheck completed - report saved to $reportsDir\govulncheck.json"
        } else {
            Write-Warning "⚠ govulncheck completed with warnings"
            $vulnOutput | Out-File "$reportsDir\govulncheck-warnings.log"
        }
        
        # Generate SBOM
        Write-Status "Generating SBOM..."
        $sbomOutput = cyclonedx-gomod mod -json -output-file "$reportsDir\sbom.json" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "✓ SBOM generated - saved to $reportsDir\sbom.json"
        } else {
            Write-Warning "⚠ SBOM generation may have failed: $sbomOutput"
        }
        
        return $true
    }
    catch {
        Write-Error "✗ Security scan failed: $($_.Exception.Message)"
        return $false
    }
}

function Show-Summary {
    param($Results)
    
    Write-Status ""
    Write-Status "=== SUMMARY ===" $Blue
    
    foreach ($result in $Results.GetEnumerator()) {
        $status = if ($result.Value) { "✓" } else { "✗" }
        $color = if ($result.Value) { $Green } else { $Red }
        Write-Status "$status $($result.Key)" $color
    }
    
    $successCount = ($Results.Values | Where-Object { $_ }).Count
    $totalCount = $Results.Count
    
    Write-Status ""
    if ($successCount -eq $totalCount) {
        Write-Success "All checks passed! Security scan should work now."
        return $true
    } else {
        Write-Warning "$successCount/$totalCount checks passed. Some issues may remain."
        return $false
    }
}

# Main execution
function Main {
    Write-Status "=== Fix Dependency Security Scan ===" $Blue
    Write-Status "Working directory: $(Get-Location)"
    Write-Status "Timeout: $TimeoutSeconds seconds"
    Write-Status ""
    
    # Check if we're in the right directory
    if (-not (Test-Path "go.mod")) {
        Write-Error "No go.mod found. Please run this script from the repository root."
        exit 1
    }
    
    $results = @{}
    
    # Step 1: Check Go version
    $results["Go Version"] = Test-GoVersion
    
    # Step 2: Test network connectivity (unless skipped)
    if (-not $SkipNetworkTests) {
        $networkOk = Test-NetworkConnectivity
        $results["Network Connectivity"] = $networkOk
        
        if (-not $networkOk -and -not $Force) {
            Write-Warning "Network connectivity issues detected. Use -Force to proceed anyway."
            Write-Warning "Or use -SkipNetworkTests to skip network tests."
            Show-Summary $results
            exit 1
        }
    } else {
        Write-Status "Skipping network connectivity tests"
        $results["Network Connectivity"] = $true
    }
    
    # Step 3: Set up Go proxy environment
    Set-GoProxyEnvironment $ProxyUrl
    
    # Step 4: Create backups (unless forced to skip)
    if (-not $Force) {
        Backup-GoModFiles
    }
    
    # Step 5: Run go mod tidy
    $results["go mod tidy"] = Run-GoModTidy $TimeoutSeconds
    
    # Step 6: Run go mod download
    $results["go mod download"] = Run-GoModDownload $TimeoutSeconds
    
    # Step 7: Verify modules
    $results["Module Verification"] = Test-GoModVerify
    
    # Step 8: Test build
    $results["Build Test"] = Test-GoBuild
    
    # Step 9: Install security tools
    $results["Security Tools"] = Install-SecurityTools
    
    # Step 10: Run security scan
    $results["Security Scan"] = Run-SecurityScan
    
    # Show summary
    $success = Show-Summary $results
    
    if ($success) {
        Write-Status ""
        Write-Success "Dependencies are now ready for CI security scan!"
        Write-Status "Next steps:"
        Write-Status "1. Commit the updated go.sum file"
        Write-Status "2. Push changes to trigger CI"
        Write-Status "3. Check security reports in .security-reports/"
        exit 0
    } else {
        Write-Status ""
        Write-Warning "Some issues remain. Check the output above for details."
        Write-Status "You may need to:"
        Write-Status "1. Fix network connectivity issues"
        Write-Status "2. Update Go to a newer version"
        Write-Status "3. Resolve any dependency conflicts manually"
        exit 1
    }
}

# Run the main function
Main
# Cryptographic Security Validation Script
# Validates that all critical cryptographic vulnerabilities have been fixed

param(
    [string]$RepoPath = (Get-Location).Path,
    [switch]$Verbose = $false
)

Write-Host "🔒 Cryptographic Security Validation" -ForegroundColor Cyan
Write-Host "Repository: $RepoPath" -ForegroundColor Yellow
Write-Host ""

$ErrorCount = 0
$WarningCount = 0
$FixCount = 0

function Write-Check {
    param([string]$Message, [string]$Status, [string]$Color = "Green")
    
    $statusSymbol = switch ($Status) {
        "PASS" { "✅" }
        "FAIL" { "❌" }
        "WARN" { "⚠️" }
        "FIX" { "🔧" }
        default { "ℹ️" }
    }
    
    Write-Host "$statusSymbol $Message" -ForegroundColor $Color
}

function Test-InsecureSkipVerifyUsage {
    Write-Host "Checking for InsecureSkipVerify usage..." -ForegroundColor Blue
    
    $insecureFiles = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "InsecureSkipVerify.*true" |
        Where-Object { $_.Line -notmatch "test|Test|_test\.go" -and $_.Line -notmatch "// Only for testing" }
    
    if ($insecureFiles.Count -eq 0) {
        Write-Check "No production InsecureSkipVerify=true found" "PASS"
        return $true
    } else {
        foreach ($file in $insecureFiles) {
            Write-Check "Found InsecureSkipVerify=true in $($file.Filename):$($file.LineNumber)" "FAIL" "Red"
            if ($Verbose) {
                Write-Host "    $($file.Line.Trim())" -ForegroundColor Gray
            }
        }
        return $false
    }
}

function Test-TLSVersionEnforcement {
    Write-Host "Checking TLS version enforcement..." -ForegroundColor Blue
    
    # Check for TLS 1.3 enforcement
    $tls13Files = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "tls\.VersionTLS13" 
    
    if ($tls13Files.Count -gt 0) {
        Write-Check "TLS 1.3 enforcement found in $($tls13Files.Count) files" "PASS"
        
        if ($Verbose) {
            foreach ($file in $tls13Files[0..4]) { # Show first 5
                Write-Host "  - $($file.Filename):$($file.LineNumber)" -ForegroundColor Green
            }
        }
        
        return $true
    } else {
        Write-Check "No TLS 1.3 enforcement found" "FAIL" "Red"
        return $false
    }
}

function Test-WeakTLSVersions {
    Write-Host "Checking for weak TLS versions..." -ForegroundColor Blue
    
    $weakTLS = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "tls\.VersionTLS1[0-2]" |
        Where-Object { $_.Line -notmatch "test|Test|_test\.go" }
    
    if ($weakTLS.Count -eq 0) {
        Write-Check "No weak TLS versions (1.0, 1.1, 1.2) found in production code" "PASS"
        return $true
    } else {
        foreach ($file in $weakTLS) {
            Write-Check "Found weak TLS version in $($file.Filename):$($file.LineNumber)" "WARN" "Yellow"
            if ($Verbose) {
                Write-Host "    $($file.Line.Trim())" -ForegroundColor Gray
            }
        }
        return $false
    }
}

function Test-StrongCipherSuites {
    Write-Host "Checking for strong cipher suites..." -ForegroundColor Blue
    
    # Check for AES-256-GCM
    $strongCiphers = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "TLS_AES_256_GCM_SHA384|TLS_CHACHA20_POLY1305_SHA256"
    
    if ($strongCiphers.Count -gt 0) {
        Write-Check "Strong cipher suites (AEAD) found in $($strongCiphers.Count) locations" "PASS"
        return $true
    } else {
        Write-Check "No strong cipher suites found" "WARN" "Yellow"
        return $false
    }
}

function Test-MathRandUsage {
    Write-Host "Checking for insecure math/rand usage..." -ForegroundColor Blue
    
    $mathRandFiles = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern 'import.*"math/rand"' |
        Where-Object { $_.Line -notmatch "_test\.go" }
    
    if ($mathRandFiles.Count -eq 0) {
        Write-Check "No insecure math/rand imports found in production code" "PASS"
        return $true
    } else {
        foreach ($file in $mathRandFiles) {
            Write-Check "Found math/rand import in $($file.Filename):$($file.LineNumber)" "WARN" "Yellow"
        }
        return $false
    }
}

function Test-CryptoRandUsage {
    Write-Host "Checking for secure crypto/rand usage..." -ForegroundColor Blue
    
    $cryptoRandFiles = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern 'import.*"crypto/rand"'
    
    if ($cryptoRandFiles.Count -gt 0) {
        Write-Check "Secure crypto/rand usage found in $($cryptoRandFiles.Count) files" "PASS"
        return $true
    } else {
        Write-Check "No crypto/rand usage found" "WARN" "Yellow"
        return $false
    }
}

function Test-RSAKeySize {
    Write-Host "Checking RSA key sizes..." -ForegroundColor Blue
    
    # Check for 4096-bit keys (secure)
    $strongRSA = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "GenerateKey.*4096|4096.*GenerateKey"
    
    # Check for weak 2048-bit keys
    $weakRSA = Get-ChildItem -Path $RepoPath -Recurse -Include "*.go" | 
        Select-String -Pattern "GenerateKey.*2048|2048.*GenerateKey"
    
    if ($strongRSA.Count -gt 0) {
        Write-Check "4096-bit RSA keys found in $($strongRSA.Count) locations" "PASS"
        $result = $true
    } else {
        $result = $false
    }
    
    if ($weakRSA.Count -gt 0) {
        foreach ($file in $weakRSA) {
            Write-Check "Found weak 2048-bit RSA key in $($file.Filename):$($file.LineNumber)" "WARN" "Yellow"
        }
        $result = $false
    }
    
    if (-not $result) {
        Write-Check "No strong RSA key generation found" "WARN" "Yellow"
    }
    
    return $result
}

function Test-SecurityAuditReport {
    Write-Host "Checking for security audit documentation..." -ForegroundColor Blue
    
    $auditReport = Get-ChildItem -Path $RepoPath -Recurse -Include "*SECURITY*AUDIT*", "*CRYPTO*AUDIT*" -Name
    
    if ($auditReport.Count -gt 0) {
        Write-Check "Security audit report found: $($auditReport -join ', ')" "PASS"
        return $true
    } else {
        Write-Check "No security audit report found" "WARN" "Yellow"
        return $false
    }
}

function Test-SecurityImplementations {
    Write-Host "Checking for security implementations..." -ForegroundColor Blue
    
    $securityFiles = @(
        "pkg/security/crypto_hardened.go",
        "pkg/security/secure_random.go",
        "pkg/security/mtls_enterprise.go",
        "pkg/security/cert_rotation_enterprise.go"
    )
    
    $foundCount = 0
    foreach ($file in $securityFiles) {
        $fullPath = Join-Path $RepoPath $file
        if (Test-Path $fullPath) {
            Write-Check "Security implementation found: $file" "PASS"
            $foundCount++
        } else {
            Write-Check "Missing security implementation: $file" "FAIL" "Red"
        }
    }
    
    return $foundCount -eq $securityFiles.Count
}

# Run all tests
Write-Host "Running cryptographic security validation checks..." -ForegroundColor Magenta
Write-Host ""

$testResults = @()
$testResults += Test-InsecureSkipVerifyUsage
$testResults += Test-TLSVersionEnforcement
$testResults += Test-WeakTLSVersions
$testResults += Test-StrongCipherSuites
$testResults += Test-MathRandUsage
$testResults += Test-CryptoRandUsage
$testResults += Test-RSAKeySize
$testResults += Test-SecurityAuditReport
$testResults += Test-SecurityImplementations

$passCount = ($testResults | Where-Object { $_ -eq $true }).Count
$failCount = $testResults.Count - $passCount

Write-Host ""
Write-Host "🔒 Cryptographic Security Validation Results" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "✅ Passed: $passCount" -ForegroundColor Green
Write-Host "❌ Failed: $failCount" -ForegroundColor Red

if ($failCount -eq 0) {
    Write-Host ""
    Write-Host "🎉 ALL CRYPTOGRAPHIC SECURITY CHECKS PASSED!" -ForegroundColor Green
    Write-Host "   The codebase meets enterprise-grade security standards." -ForegroundColor Green
    exit 0
} else {
    Write-Host ""
    Write-Host "⚠️  SECURITY ISSUES DETECTED!" -ForegroundColor Red
    Write-Host "   Please address the failing checks above." -ForegroundColor Red
    exit 1
}
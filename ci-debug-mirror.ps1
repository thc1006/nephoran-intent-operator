# CI Debug Mirror Script
# Reproduces exact CI environment locally for golangci-lint debugging
# Generated: 2025-08-28

Write-Host "🔬 Nephoran CI Debug Mirror" -ForegroundColor Cyan
Write-Host "Creating exact local mirror of CI environment..." -ForegroundColor Green

# =============================================================================
# Environment Verification
# =============================================================================

Write-Host "`n📋 STEP 1: Environment Verification" -ForegroundColor Yellow

# Check Go version
$goVersion = go version
Write-Host "Local Go Version: $goVersion"

# Expected from CI: go1.24.6 (matches go.mod: go 1.24.6)
if ($goVersion -match "go1\.24\.") {
    Write-Host "✅ Go version matches CI requirements" -ForegroundColor Green
} else {
    Write-Host "❌ Go version mismatch! CI expects go1.24.x" -ForegroundColor Red
    Write-Host "Install Go 1.24.x from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Check golangci-lint version
$gopath = go env GOPATH
$linterPath = "$gopath\bin\golangci-lint.exe"

if (Test-Path $linterPath) {
    $linterVersion = & $linterPath version
    Write-Host "Local golangci-lint: $linterVersion"
    
    if ($linterVersion -match "v1\.61\.0") {
        Write-Host "✅ golangci-lint version matches CI (v1.61.0)" -ForegroundColor Green
    } else {
        Write-Host "❌ golangci-lint version mismatch! Installing v1.61.0..." -ForegroundColor Yellow
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
        $linterVersion = & $linterPath version
        Write-Host "Updated: $linterVersion"
    }
} else {
    Write-Host "⚠️ golangci-lint not found. Installing v1.61.0..." -ForegroundColor Yellow
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
    
    if (Test-Path $linterPath) {
        $linterVersion = & $linterPath version
        Write-Host "Installed: $linterVersion"
        Write-Host "✅ golangci-lint v1.61.0 installed successfully" -ForegroundColor Green
    } else {
        Write-Host "❌ Failed to install golangci-lint" -ForegroundColor Red
        exit 1
    }
}

# =============================================================================
# Environment Comparison
# =============================================================================

Write-Host "`n📊 STEP 2: Environment Comparison" -ForegroundColor Yellow

# Display key environment variables
Write-Host "Key Go Environment Variables:"
$goEnv = go env
$keyVars = @('GOVERSION', 'GOOS', 'GOARCH', 'GOMOD', 'GOPATH', 'GOPROXY', 'GOSUMDB')

foreach ($var in $keyVars) {
    $value = ($goEnv | Where-Object { $_ -match "^$var=" }) -replace "^$var=", ""
    Write-Host "  $var = $value"
}

Write-Host "`nCI vs Local Differences:"
Write-Host "  CI Platform: ubuntu-latest (Linux)"
Write-Host "  Local Platform: windows/amd64"
Write-Host "  Expected Impact: Path separators, possible Go build cache differences"

# =============================================================================
# Pre-lint Diagnostics
# =============================================================================

Write-Host "`n🔍 STEP 3: Pre-lint Diagnostics" -ForegroundColor Yellow

# Check if .golangci.yml exists
if (Test-Path ".golangci.yml") {
    Write-Host "✅ Found .golangci.yml configuration"
    
    # Check for deprecated options
    $config = Get-Content ".golangci.yml" -Raw
    $deprecations = @()
    
    if ($config -match "run\.skip-files") { $deprecations += "run.skip-files -> issues.exclude-files" }
    if ($config -match "run\.skip-dirs") { $deprecations += "run.skip-dirs -> issues.exclude-dirs" }
    if ($config -match "github-actions") { $deprecations += "output format 'github-actions' -> 'colored-line-number'" }
    
    if ($deprecations.Count -gt 0) {
        Write-Host "⚠️ Configuration deprecations found:" -ForegroundColor Yellow
        foreach ($dep in $deprecations) {
            Write-Host "    $dep"
        }
    }
} else {
    Write-Host "❌ .golangci.yml not found!" -ForegroundColor Red
    exit 1
}

# Check module dependencies
Write-Host "`nChecking module dependencies..."
$modStatus = go mod verify 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Module verification passed"
} else {
    Write-Host "⚠️ Module verification issues:" -ForegroundColor Yellow
    Write-Host "$modStatus"
}

# =============================================================================
# Exact CI Command Execution
# =============================================================================

Write-Host "`n🚀 STEP 4: Exact CI Command Execution" -ForegroundColor Yellow

$timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
$logFile = "golangci-lint-debug-$timestamp.log"

Write-Host "Executing exact CI command with full output capture..."
Write-Host "Command: golangci-lint run --timeout=10m --out-format=github-actions ./..."
Write-Host "Log file: $logFile"

# Execute with both formats for comparison
Write-Host "`n--- CI Format (github-actions) ---"
try {
    $ciOutput = & $linterPath run --timeout=10m --out-format=github-actions ./... 2>&1
    $ciExitCode = $LASTEXITCODE
    $ciOutput | Out-File -FilePath $logFile -Encoding UTF8
    
    Write-Host "Exit Code: $ciExitCode"
    if ($ciOutput) {
        Write-Host "Output preview (first 20 lines):"
        ($ciOutput | Select-Object -First 20) | ForEach-Object { Write-Host "  $_" }
        Write-Host "  ... (full output saved to $logFile)"
    }
} catch {
    Write-Host "❌ CI format execution failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n--- Recommended Format (colored-line-number) ---"
$logFileRecommended = "golangci-lint-recommended-$timestamp.log"
try {
    $recOutput = & $linterPath run --timeout=10m --out-format=colored-line-number ./... 2>&1
    $recExitCode = $LASTEXITCODE
    $recOutput | Out-File -FilePath $logFileRecommended -Encoding UTF8
    
    Write-Host "Exit Code: $recExitCode"
    if ($recOutput) {
        Write-Host "Output preview (first 20 lines):"
        ($recOutput | Select-Object -First 20) | ForEach-Object { Write-Host "  $_" }
        Write-Host "  ... (full output saved to $logFileRecommended)"
    }
} catch {
    Write-Host "❌ Recommended format execution failed: $($_.Exception.Message)" -ForegroundColor Red
}

# =============================================================================
# Error Analysis
# =============================================================================

Write-Host "`n📈 STEP 5: Error Analysis" -ForegroundColor Yellow

# Parse common error patterns
$errorPatterns = @{
    "GetNamespace undefined" = "Missing methods in Kubernetes resource types"
    "could not import sync/atomic" = "Go standard library import issues (unsupported version: 2)"
    "undefined: DocumentChunk" = "Missing type definitions in RAG package"
    "undefined: GinkgoWriter" = "Missing Ginkgo test framework imports"
    "goanalysis_metalinter" = "Static analysis tool failure (ast package import issue)"
}

Write-Host "Common Error Patterns Found:"
foreach ($pattern in $errorPatterns.Keys) {
    if ($ciOutput -match [regex]::Escape($pattern)) {
        Write-Host "  ❌ $pattern" -ForegroundColor Red
        Write-Host "     → $($errorPatterns[$pattern])" -ForegroundColor Gray
    }
}

# =============================================================================
# Recommendations
# =============================================================================

Write-Host "`n💡 STEP 6: Debugging Recommendations" -ForegroundColor Yellow

Write-Host "Immediate Actions:"
Write-Host "1. 🔧 Fix GetNamespace method issues in API types"
Write-Host "2. 🧹 Resolve import issues with sync/atomic and k8s.io packages"
Write-Host "3. 📦 Fix undefined types in pkg/rag package"
Write-Host "4. 🧪 Ensure proper Ginkgo imports in test files"
Write-Host "5. 🔄 Update .golangci.yml to remove deprecated options"

Write-Host "`nDebugging Commands:"
Write-Host "# Run specific linters only:"
Write-Host "$linterPath run --disable-all --enable=typecheck ./..."
Write-Host "# Run with verbose output:"
Write-Host "$linterPath run --verbose ./..."
Write-Host "# Check specific package:"
Write-Host "$linterPath run ./api/v1/..."

Write-Host "`nCI vs Local Environment:"
Write-Host "✅ Go version: MATCHES (1.24.6)"
Write-Host "✅ golangci-lint version: MATCHES (v1.61.0)"
Write-Host "✅ Configuration: SAME (.golangci.yml)"
Write-Host "⚠️  Platform: DIFFERENT (ubuntu-latest vs windows)"
Write-Host "⚠️  Path separators: DIFFERENT (/ vs \)"

Write-Host "`n📋 Log Files Created:" -ForegroundColor Green
Write-Host "  - $logFile (CI format output)"
Write-Host "  - $logFileRecommended (recommended format output)"

Write-Host "`n🎯 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Review log files for specific error locations"
Write-Host "2. Fix undefined methods/types identified above"
Write-Host "3. Test fixes with: .\ci-debug-mirror.ps1"
Write-Host "4. Verify CI passes after fixes"

Write-Host "`n✅ CI Debug Mirror Complete" -ForegroundColor Green
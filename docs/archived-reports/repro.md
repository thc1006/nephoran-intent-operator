# CI Reproduction Guide

## Prerequisites
- Go 1.24.1 or compatible
- Git repository cloned locally
- Windows PowerShell or bash terminal

## Exact Commands to Reproduce CI Locally

### Step 1: Environment Setup
```powershell
# Clean any cached data
go clean -testcache
go clean -modcache

# Verify Go environment
go env | findstr -E "(GOVERSION|GOOS|GOARCH)"
```

**Expected Output**:
```
GOVERSION="go1.24.1"
GOOS="windows"
GOARCH="amd64"
```

### Step 2: Build Validation (Critical Path)
```powershell
# Step 2a: Build all packages (catches compile-time errors)
Write-Host "=== Step 1: Building all packages ===" -ForegroundColor Yellow
$buildResult = go build ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed - compilation errors detected" -ForegroundColor Red
    exit 1
} else {
    Write-Host "✅ All packages built successfully" -ForegroundColor Green
}
```

### Step 3: Static Analysis
```powershell
# Step 3a: Run go vet
Write-Host "=== Step 2: Running go vet ===" -ForegroundColor Yellow
$vetResult = go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Go vet failed - potential issues detected" -ForegroundColor Red
    exit 1
} else {
    Write-Host "✅ Go vet passed successfully" -ForegroundColor Green
}

# Step 3b: Compile test binaries per package
Write-Host "=== Step 3: Compiling test binaries ===" -ForegroundColor Yellow
Remove-Item -Recurse -Force -ErrorAction SilentlyContinue .cache/tests
New-Item -ItemType Directory -Force .cache/tests | Out-Null

$compiledCount = 0
$packages = go list ./...
foreach ($pkg in $packages) {
    $safeName = $pkg -replace '[^A-Za-z0-9]', '_'
    $outputFile = ".cache/tests/${safeName}.test"
    
    if (go test -c $pkg -o $outputFile 2>$null) {
        $compiledCount++
        Write-Host "  ✓ Compiled: $pkg → ${safeName}.test" -ForegroundColor Gray
    } else {
        Write-Host "  ○ Skipped: $pkg (no tests or compilation error)" -ForegroundColor DarkGray
    }
}

Write-Host "Test binary compilation complete: $compiledCount binaries" -ForegroundColor Green
```

### Step 4: Lint Checks (golangci-lint)
```powershell
# Step 4a: Install golangci-lint (if not available)
if (!(Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
    Write-Host "Installing golangci-lint..." -ForegroundColor Yellow
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
}

# Step 4b: Run golangci-lint exactly as CI does
Write-Host "=== Step 4: Running golangci-lint ===" -ForegroundColor Yellow
golangci-lint run --timeout=10m --only-new-issues=false
```

### Step 5: Unit Tests (Core Packages)
```powershell
# Step 5a: Test core fixed packages
Write-Host "=== Step 5: Running unit tests on core packages ===" -ForegroundColor Yellow
go test -short -v ./pkg/shared ./api/v1 ./pkg/controllers/interfaces

# Step 5b: Test additional critical packages
go test -short -v ./pkg/nephio/dependencies ./pkg/optimization ./pkg/oran/e2
```

### Step 6: Security Scan (govulncheck)
```powershell
# Step 6a: Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@v1.1.4

# Step 6b: Run vulnerability scan
Write-Host "=== Step 6: Running vulnerability scan ===" -ForegroundColor Yellow
govulncheck ./...
```

---

## Expected Outputs

### ✅ Successful Build Output
```
# After applying all fixes, you should see:
=== Step 1: Building all packages ===
✅ All packages built successfully

=== Step 2: Running go vet ===
✅ Go vet passed successfully  

=== Step 3: Compiling test binaries ===
Test binary compilation complete: 25+ binaries

=== Step 4: Running golangci-lint ===
[No output = success, or only minor warnings]

=== Step 5: Running unit tests ===
ok  	github.com/thc1006/nephoran-intent-operator/pkg/shared	0.167s
?   	github.com/thc1006/nephoran-intent-operator/api/v1	[no test files]
?   	github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces	[no test files]
```

### ❌ Original Failure Output (Before Fixes)
```
# Before fixes, you would see:
# github.com/thc1006/nephoran-intent-operator/pkg/optimization  
##[error]pkg/optimization/telecom_optimizer.go:21:2: "fmt" imported and not used
##[error]pkg/optimization/telecom_optimizer.go:22:2: "math" imported and not used

# github.com/thc1006/nephoran-intent-operator/pkg/nephio
##[error]pkg/nephio/workflow_engine.go:340:26: cannot use *promauto.NewCounterVec(...) as *prometheus.CounterVec value

# [... many more errors leading to:]
❌ Build failed - compilation errors detected
This typically indicates symbol collisions or missing dependencies
```

---

## Troubleshooting

### If Build Still Fails:
1. **Check Go version**: Must be 1.24.x or compatible
2. **Clean modules**: `go clean -modcache && go mod download`
3. **Verify patches applied**: Check that all files in `patch.diff` were modified
4. **Check for conflicts**: `git status` should show modified files without conflicts

### If Tests Fail:
1. **Timeout issues**: Add `-timeout=5m` to go test commands
2. **Race conditions**: Remove `-race` flag if running on single CPU
3. **Missing test deps**: Run `go mod download` and `go mod tidy`

### If Lint Fails:
1. **Check golangci-lint version**: Should be v1.61.0
2. **Configuration**: Ensure `.golangci.yml` exists at repo root
3. **Incremental**: Add `--new-from-rev=HEAD~1` for incremental checking

---

## CI Workflow Mapping

This local reproduction matches the GitHub Actions workflow in `.github/workflows/ci.yml`:

- **hygiene** job → Environment setup
- **build-validation** → Steps 2-3 (build, vet, test compilation)  
- **lint** job → Step 4 (golangci-lint)
- **unit-tests** job → Step 5 (go test)
- **security** job → Step 6 (govulncheck)

**Time expectation**: 5-15 minutes locally vs 20+ minutes in CI
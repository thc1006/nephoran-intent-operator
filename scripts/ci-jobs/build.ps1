#!/usr/bin/env pwsh
# =====================================================================================
# CI Job: Build & Code Quality  
# =====================================================================================
# Mirrors: .github/workflows/main-ci.yml -> build-and-quality job
#         .github/workflows/ubuntu-ci.yml -> build job
# This script reproduces the exact build and quality checks from CI
# =====================================================================================

param(
    [switch]$SkipCodegen,
    [switch]$SkipFormat,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Load CI environment
. "scripts\ci-env.ps1"

Write-Host "=== Build & Code Quality ===" -ForegroundColor Green
Write-Host "Skip Codegen: $SkipCodegen" -ForegroundColor Cyan
Write-Host "Skip Format: $SkipFormat" -ForegroundColor Cyan
Write-Host ""

# Step 1: Install build tools (mirrors CI line 229-257)
Write-Host "[1/6] Installing build tools..." -ForegroundColor Blue
try {
    # Create bin directory
    if (!(Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" -Force | Out-Null
    }
    
    # Check for controller-gen
    $controllerGenPath = "bin\controller-gen.exe"
    $controllerGenInstalled = $false
    
    if (Test-Path $controllerGenPath) {
        try {
            $version = & $controllerGenPath --version 2>&1 | Out-String
            if ($version -match $env:CONTROLLER_GEN_VERSION.Replace("v", "")) {
                Write-Host "✅ controller-gen $env:CONTROLLER_GEN_VERSION already installed" -ForegroundColor Green
                $controllerGenInstalled = $true
            }
        }
        catch {
            Write-Host "⚠️ Existing controller-gen version check failed" -ForegroundColor Yellow
        }
    }
    
    if (!$controllerGenInstalled) {
        Write-Host "📥 Installing controller-gen $env:CONTROLLER_GEN_VERSION..." -ForegroundColor Yellow
        
        # Install to local bin directory (matches CI pattern)
        $env:GOBIN = (Resolve-Path "bin").Path
        & go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$env:CONTROLLER_GEN_VERSION"
        
        # Verify installation
        if (Test-Path $controllerGenPath) {
            $version = & $controllerGenPath --version 2>&1 | Out-String
            Write-Host "✅ controller-gen installed successfully" -ForegroundColor Green
            Write-Host "Version: $($version.Trim())" -ForegroundColor Gray
        } else {
            Write-Host "❌ controller-gen installation failed" -ForegroundColor Red
            Get-ChildItem "bin" -ErrorAction SilentlyContinue | ForEach-Object {
                Write-Host "  Found: $($_.Name)" -ForegroundColor Gray
            }
            exit 1
        }
    }
}
catch {
    Write-Host "❌ Build tools installation failed: $_" -ForegroundColor Red
    exit 1
}

# Step 2: Code generation and verification (mirrors CI line 259-294)
if (!$SkipCodegen) {
    Write-Host "[2/6] Code generation and verification..." -ForegroundColor Blue
    try {
        Write-Host "🏗️ Generating code and manifests..." -ForegroundColor Yellow
        
        # Store checksums before generation (matches CI)
        Write-Host "📋 Storing checksums before generation..." -ForegroundColor Gray
        $beforeHashes = @{}
        
        # Get checksums of generated files
        $generatedFiles = @()
        if (Test-Path "api") { $generatedFiles += Get-ChildItem "api" -Recurse -Filter "*.go" }
        if (Test-Path "controllers") { $generatedFiles += Get-ChildItem "controllers" -Recurse -Filter "*.go" }
        if (Test-Path "config\crd\bases") { $generatedFiles += Get-ChildItem "config\crd\bases" -Recurse -Filter "*.yaml" }
        
        foreach ($file in $generatedFiles) {
            $beforeHashes[$file.FullName] = (Get-FileHash $file.FullName).Hash
        }
        
        # Generate code (matches CI exact commands)
        Write-Host "🔄 Running object code generation..." -ForegroundColor Gray
        $headerFile = if (Test-Path "hack\boilerplate.go.txt") { "hack\boilerplate.go.txt" } else { "" }
        $codegenArgs = @("object:headerFile=`"$headerFile`"", "paths=./...")
        if ($headerFile) {
            & "bin\controller-gen.exe" @codegenArgs
        } else {
            & "bin\controller-gen.exe" object paths=./...
        }
        
        Write-Host "🔄 Running CRD and RBAC generation..." -ForegroundColor Gray
        $manifestArgs = @(
            "rbac:roleName=manager-role"
            "crd"
            "webhook" 
            "paths=./..."
            "output:crd:artifacts:config=config/crd/bases"
        )
        & "bin\controller-gen.exe" @manifestArgs
        
        # Check for changes (matches CI)
        Write-Host "🔍 Checking for uncommitted generated code..." -ForegroundColor Gray
        
        # Use git to check for changes
        $gitStatus = git status --porcelain 2>&1 | Out-String
        if ($gitStatus.Trim()) {
            Write-Host "❌ Generated files are not up to date" -ForegroundColor Red
            Write-Host "📋 Files that changed:" -ForegroundColor Yellow
            git status --porcelain | ForEach-Object {
                Write-Host "  $_" -ForegroundColor Red
            }
            Write-Host ""
            Write-Host "📋 Detailed changes:" -ForegroundColor Yellow
            git diff --name-only | ForEach-Object {
                Write-Host "Changed file: $_" -ForegroundColor Red
                git diff $_ | Select-Object -First 20 | ForEach-Object {
                    Write-Host "  $_" -ForegroundColor Gray
                }
            }
            Write-Host ""
            Write-Host "🔧 Please run 'make generate manifests' locally and commit the changes" -ForegroundColor Yellow
            exit 1
        }
        
        Write-Host "✅ All generated files are up to date" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Code generation failed: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[2/6] Code generation skipped" -ForegroundColor Blue
}

# Step 3: Code formatting verification (mirrors CI line 296-324)
if (!$SkipFormat) {
    Write-Host "[3/6] Code formatting verification..." -ForegroundColor Blue
    try {
        Write-Host "📝 Verifying code formatting..." -ForegroundColor Yellow
        
        # Store original state (matches CI pattern)
        Write-Host "📋 Storing original formatting state..." -ForegroundColor Gray
        $beforeFiles = Get-ChildItem -Path . -Recurse -Filter "*.go" | Where-Object { $_.FullName -notmatch "vendor|\.git" }
        $beforeHashes = @{}
        foreach ($file in $beforeFiles) {
            $beforeHashes[$file.FullName] = (Get-FileHash $file.FullName).Hash
        }
        
        # Run go fmt (matches CI)
        Write-Host "🔄 Running go fmt..." -ForegroundColor Gray
        $fmtOutput = go fmt ./... 2>&1 | Out-String
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Code formatting failed" -ForegroundColor Red
            Write-Host $fmtOutput -ForegroundColor Red
            exit 1
        }
        
        # Check for changes
        Write-Host "🔍 Checking for formatting changes..." -ForegroundColor Gray
        $afterFiles = Get-ChildItem -Path . -Recurse -Filter "*.go" | Where-Object { $_.FullName -notmatch "vendor|\.git" }
        $hasChanges = $false
        
        foreach ($file in $afterFiles) {
            $afterHash = (Get-FileHash $file.FullName).Hash
            if ($beforeHashes[$file.FullName] -ne $afterHash) {
                $hasChanges = $true
                Write-Host "  Changed: $($file.Name)" -ForegroundColor Red
            }
        }
        
        if ($hasChanges) {
            Write-Host "❌ Code is not properly formatted" -ForegroundColor Red
            Write-Host "📋 Files that would change:" -ForegroundColor Yellow
            git diff --name-only | ForEach-Object {
                Write-Host "  $_" -ForegroundColor Red
            }
            Write-Host ""
            Write-Host "🔧 Please run 'go fmt ./...' locally and commit the changes" -ForegroundColor Yellow
            exit 1
        }
        
        Write-Host "✅ Code is properly formatted" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Code formatting check failed: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[3/6] Code formatting check skipped" -ForegroundColor Blue
}

# Step 4: Static analysis (mirrors CI line 326-337) 
Write-Host "[4/6] Static analysis with go vet..." -ForegroundColor Blue
try {
    Write-Host "🔍 Running static analysis..." -ForegroundColor Yellow
    
    $vetOutput = go vet ./... 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Static analysis found issues" -ForegroundColor Red
        Write-Host $vetOutput -ForegroundColor Red
        Write-Host "📋 Please fix the reported issues" -ForegroundColor Yellow
        exit 1
    }
    
    Write-Host "✅ Static analysis passed" -ForegroundColor Green
}
catch {
    Write-Host "❌ Static analysis failed: $_" -ForegroundColor Red
    exit 1
}

# Step 5: Build verification (mirrors CI line 339-368)
Write-Host "[5/6] Build verification..." -ForegroundColor Blue
try {
    Write-Host "🏗️ Building all packages..." -ForegroundColor Yellow
    
    if ($Verbose) {
        $buildOutput = go build -v ./... 2>&1 | Out-String
        Write-Host $buildOutput -ForegroundColor Gray
    } else {
        go build ./... 2>&1 | Out-Null
    }
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed" -ForegroundColor Red
        Write-Host "📋 Build errors detected" -ForegroundColor Yellow
        exit 1
    }
    
    # Build main executables if they exist (matches CI)
    Write-Host "🔄 Building main executables..." -ForegroundColor Gray
    if (Test-Path "cmd") {
        $builtExecutables = 0
        Get-ChildItem "cmd" -Directory | ForEach-Object {
            $cmdDir = $_.FullName
            $mainFile = Join-Path $cmdDir "main.go"
            
            if (Test-Path $mainFile) {
                $cmdName = $_.Name
                $outputPath = "bin\$cmdName.exe"
                
                Write-Host "🔄 Building $cmdName..." -ForegroundColor Gray
                go build -o $outputPath ".\cmd\$cmdName" 2>&1 | Out-Null
                
                if ($LASTEXITCODE -eq 0 -and (Test-Path $outputPath)) {
                    Write-Host "✅ Built $cmdName successfully" -ForegroundColor Green
                    $builtExecutables++
                } else {
                    Write-Host "❌ Failed to build $cmdName" -ForegroundColor Red
                    exit 1
                }
            }
        }
        
        if ($builtExecutables -gt 0) {
            Write-Host "Built $builtExecutables executable(s) in bin/" -ForegroundColor Green
        } else {
            Write-Host "No main.go files found in cmd directories" -ForegroundColor Gray
        }
    }
    
    Write-Host "✅ All builds completed successfully" -ForegroundColor Green
}
catch {
    Write-Host "❌ Build verification failed: $_" -ForegroundColor Red
    exit 1
}

# Step 6: Verify executables (additional check)
Write-Host "[6/6] Verifying built executables..." -ForegroundColor Blue
if (Test-Path "bin") {
    $executables = Get-ChildItem "bin" -Filter "*.exe"
    if ($executables) {
        Write-Host "Verifying executables:" -ForegroundColor Gray
        foreach ($exe in $executables) {
            if (Test-Path $exe.FullName) {
                $size = [math]::Round($exe.Length / 1MB, 2)
                Write-Host "✅ $($exe.Name) ($size MB)" -ForegroundColor Green
            } else {
                Write-Host "❌ $($exe.Name) - not accessible" -ForegroundColor Red
            }
        }
    } else {
        Write-Host "No executables found in bin/" -ForegroundColor Gray
    }
} else {
    Write-Host "No bin directory found" -ForegroundColor Gray
}

Write-Host ""
Write-Host "🎉 Build & Code Quality checks completed successfully!" -ForegroundColor Green
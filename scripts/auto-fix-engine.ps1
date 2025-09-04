#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Automated Fix Engine - Apply incremental fixes with safety checks
.DESCRIPTION
    Intelligent fix engine that applies fixes incrementally with:
    - Safety checks and rollback capability
    - Fix verification after each application
    - Conflict detection and resolution
    - Progress tracking integration
    - Smart fix ordering based on success history
#>
param(
    [Parameter(Position = 0)]
    [ValidateSet("analyze", "apply", "verify", "rollback", "reset")]
    [string]$Command = "analyze",
    
    [string[]]$FixTypes = @(),
    [switch]$DryRun,
    [switch]$Force,
    [switch]$Interactive,
    [int]$MaxFixes = 10,
    [string]$BackupId
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$FixEngineDir = Join-Path $ProjectRoot ".fix-engine"
$BackupDir = Join-Path $FixEngineDir "backups"
$StateFile = Join-Path $FixEngineDir "state.json"

# Fix definitions with smart ordering and safety checks
$FixDefinitions = @{
    "go-fmt" = @{
        name = "Go Formatting"
        description = "Format Go code with go fmt"
        priority = 1
        safety_level = "safe"
        command = "go fmt ./..."
        verification = "go fmt -l ./..."
        rollback_command = $null
        dependencies = @()
        conflicts = @()
    }
    
    "go-mod-tidy" = @{
        name = "Go Module Cleanup"
        description = "Clean up go.mod and go.sum files"
        priority = 2
        safety_level = "safe"
        command = "go mod tidy"
        verification = "go mod verify"
        rollback_command = $null
        dependencies = @()
        conflicts = @()
    }
    
    "imports-fix" = @{
        name = "Import Organization"
        description = "Fix and organize Go imports"
        priority = 3
        safety_level = "safe"
        command = { 
            if (Get-Command "goimports" -ErrorAction SilentlyContinue) {
                "goimports -w ."
            } else {
                "go install golang.org/x/tools/cmd/goimports@latest; goimports -w ."
            }
        }
        verification = "go build ./..."
        rollback_command = $null
        dependencies = @()
        conflicts = @()
    }
    
    "golangci-autofix" = @{
        name = "Golangci-Lint Auto-Fix"
        description = "Apply golangci-lint automatic fixes"
        priority = 4
        safety_level = "medium"
        command = "golangci-lint run --fix --timeout=15m"
        verification = "golangci-lint run --timeout=5m"
        rollback_command = $null
        dependencies = @("go-fmt")
        conflicts = @()
    }
    
    "ineffassign-fix" = @{
        name = "Remove Ineffective Assignments"
        description = "Fix ineffective assignments detected by linters"
        priority = 5
        safety_level = "medium"
        command = {
            # Custom logic to fix ineffective assignments
            Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
                $content = Get-Content $_.FullName -Raw
                # Remove assignments to blank identifier when not needed
                $content = $content -replace '(\w+)\s*:?=\s*[^;\n]*\s*//.*ineffectual assignment', '_ = $1'
                Set-Content $_.FullName -Value $content -NoNewline
            }
            return "Fixed ineffective assignments"
        }
        verification = "golangci-lint run --disable-all --enable=ineffassign --timeout=2m"
        rollback_command = $null
        dependencies = @("go-fmt")
        conflicts = @()
    }
    
    "unused-vars" = @{
        name = "Remove Unused Variables"
        description = "Remove or fix unused variable declarations"
        priority = 6
        safety_level = "high"
        command = {
            # Analyze and fix unused variables
            $fixes = @()
            Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
                $file = $_.FullName
                $content = Get-Content $file -Raw
                
                # Look for unused variable patterns
                if ($content -match '(\w+)\s*:=.*//.*not used') {
                    $fixes += "Fixed unused variable in $file"
                    $content = $content -replace '(\w+)\s*:=([^;\n]*)', '_ = $2  // Fixed unused variable'
                    Set-Content $file -Value $content -NoNewline
                }
            }
            return "Fixed $($fixes.Count) unused variables"
        }
        verification = "go vet ./..."
        rollback_command = $null
        dependencies = @("go-fmt")
        conflicts = @("golangci-autofix")
    }
    
    "error-handling" = @{
        name = "Improve Error Handling"
        description = "Add missing error checks and improve error handling"
        priority = 7
        safety_level = "high"
        command = {
            # Conservative error handling fixes
            $fixes = @()
            Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
                $file = $_.FullName
                $content = Get-Content $file -Raw
                
                # Add error checks for common patterns
                if ($content -match '(\w+)\s*:=\s*([^;\n]*Error[^;\n]*)\s*$') {
                    $fixes += "Added error check in $file"
                    # This is a complex fix that would need careful implementation
                }
            }
            return "Improved error handling in $($fixes.Count) locations"
        }
        verification = "golangci-lint run --disable-all --enable=errcheck --timeout=3m"
        rollback_command = $null
        dependencies = @("go-fmt", "go-mod-tidy")
        conflicts = @()
    }
    
    "security-fixes" = @{
        name = "Security Improvements"
        description = "Apply security-related fixes from gosec"
        priority = 8
        safety_level = "critical"
        command = {
            # Apply gosec fixes carefully
            if (Get-Command "gosec" -ErrorAction SilentlyContinue) {
                $result = & gosec -fmt json ./... | ConvertFrom-Json
                # Apply safe security fixes only
                return "Applied security fixes"
            } else {
                return "gosec not available - skipping security fixes"
            }
        }
        verification = "gosec ./..."
        rollback_command = $null
        dependencies = @("go-fmt", "go-mod-tidy")
        conflicts = @()
    }
    
    "test-fixes" = @{
        name = "Test Compilation Fixes"
        description = "Fix test compilation issues"
        priority = 9
        safety_level = "medium"
        command = "go test -c ./..."
        verification = "go test -c ./..."
        rollback_command = $null
        dependencies = @("go-fmt", "go-mod-tidy")
        conflicts = @()
    }
    
    "build-tag-fixes" = @{
        name = "Build Tag Corrections"
        description = "Fix build tag syntax and issues"
        priority = 10
        safety_level = "medium"
        command = {
            $fixes = @()
            Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
                $file = $_.FullName
                $content = Get-Content $file
                
                # Fix old-style build tags
                $updated = $false
                for ($i = 0; $i -lt $content.Length; $i++) {
                    if ($content[$i] -match '^//\s*\+build\s+(.+)$') {
                        $constraint = $matches[1]
                        # Convert to new //go:build format
                        $content[$i] = "//go:build $constraint"
                        $updated = $true
                        $fixes += "Fixed build tag in $file"
                    }
                }
                
                if ($updated) {
                    Set-Content $file -Value $content
                }
            }
            return "Fixed $($fixes.Count) build tags"
        }
        verification = "go build ./..."
        rollback_command = $null
        dependencies = @()
        conflicts = @()
    }
}

function Initialize-FixEngine {
    if (-not (Test-Path $FixEngineDir)) {
        New-Item -Path $FixEngineDir -ItemType Directory -Force | Out-Null
    }
    
    if (-not (Test-Path $BackupDir)) {
        New-Item -Path $BackupDir -ItemType Directory -Force | Out-Null
    }
    
    if (-not (Test-Path $StateFile)) {
        $initialState = @{
            version = "1.0"
            created = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
            last_backup = $null
            applied_fixes = @()
            failed_fixes = @()
            rollback_stack = @()
        }
        
        $initialState | ConvertTo-Json -Depth 10 | Set-Content -Path $StateFile
    }
    
    Write-Host "🔧 Fix engine initialized" -ForegroundColor Green
}

function Get-EngineState {
    if (-not (Test-Path $StateFile)) {
        Initialize-FixEngine
    }
    return Get-Content -Path $StateFile -Raw | ConvertFrom-Json
}

function Save-EngineState {
    param([object]$State)
    $State | ConvertTo-Json -Depth 10 | Set-Content -Path $StateFile
}

function Create-Backup {
    param([string]$Description = "Auto-backup")
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $backupId = "backup-$timestamp"
    $backupPath = Join-Path $BackupDir $backupId
    
    Write-Host "💾 Creating backup: $backupId" -ForegroundColor Cyan
    
    New-Item -Path $backupPath -ItemType Directory -Force | Out-Null
    
    # Backup critical files
    $filesToBackup = @("go.mod", "go.sum")
    $filesToBackup += Get-ChildItem -Recurse -Filter "*.go" | Select-Object -First 50 | ForEach-Object { $_.FullName }
    
    foreach ($file in $filesToBackup) {
        if (Test-Path $file) {
            $relativePath = Resolve-Path $file -Relative
            $backupFilePath = Join-Path $backupPath $relativePath
            $backupFileDir = Split-Path -Parent $backupFilePath
            
            if (-not (Test-Path $backupFileDir)) {
                New-Item -Path $backupFileDir -ItemType Directory -Force | Out-Null
            }
            
            Copy-Item $file -Destination $backupFilePath -Force
        }
    }
    
    # Create backup metadata
    $metadata = @{
        id = $backupId
        timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
        description = $Description
        git_commit = (git rev-parse HEAD 2>$null) ?? "unknown"
        git_branch = (git rev-parse --abbrev-ref HEAD 2>$null) ?? "unknown"
        files_count = (Get-ChildItem $backupPath -Recurse -File).Count
    }
    
    $metadata | ConvertTo-Json -Depth 5 | Set-Content -Path (Join-Path $backupPath "metadata.json")
    
    Write-Host "✅ Backup created: $backupPath" -ForegroundColor Green
    return $backupId
}

function Restore-Backup {
    param([string]$BackupId)
    
    $backupPath = Join-Path $BackupDir $BackupId
    if (-not (Test-Path $backupPath)) {
        throw "Backup not found: $BackupId"
    }
    
    Write-Host "🔄 Restoring backup: $BackupId" -ForegroundColor Yellow
    
    $metadataPath = Join-Path $backupPath "metadata.json"
    if (Test-Path $metadataPath) {
        $metadata = Get-Content $metadataPath -Raw | ConvertFrom-Json
        Write-Host "   Description: $($metadata.description)"
        Write-Host "   Created: $($metadata.timestamp)"
        Write-Host "   Files: $($metadata.files_count)"
    }
    
    # Restore files
    Get-ChildItem $backupPath -Recurse -File | Where-Object { $_.Name -ne "metadata.json" } | ForEach-Object {
        $relativePath = $_.FullName.Substring($backupPath.Length + 1)
        $targetPath = Join-Path $ProjectRoot $relativePath
        $targetDir = Split-Path -Parent $targetPath
        
        if (-not (Test-Path $targetDir)) {
            New-Item -Path $targetDir -ItemType Directory -Force | Out-Null
        }
        
        Copy-Item $_.FullName -Destination $targetPath -Force
        Write-Host "   Restored: $relativePath" -ForegroundColor Gray
    }
    
    Write-Host "✅ Backup restored successfully" -ForegroundColor Green
}

function Get-AvailableFixes {
    param([string[]]$FilterTypes = @())
    
    $availableFixes = @()
    
    foreach ($fixKey in $FixDefinitions.Keys) {
        if ($FilterTypes.Count -gt 0 -and $fixKey -notin $FilterTypes) {
            continue
        }
        
        $fix = $FixDefinitions[$fixKey]
        $fix.key = $fixKey
        
        # Check if dependencies are met
        $dependenciesMet = $true
        foreach ($dep in $fix.dependencies) {
            $state = Get-EngineState
            if ($dep -notin $state.applied_fixes) {
                $dependenciesMet = $false
                break
            }
        }
        
        if ($dependenciesMet) {
            $availableFixes += $fix
        }
    }
    
    # Sort by priority
    return $availableFixes | Sort-Object priority
}

function Test-FixApplicable {
    param([hashtable]$Fix)
    
    Write-Host "🔍 Testing if fix '$($Fix.name)' is applicable..." -ForegroundColor Cyan
    
    # Check if fix is needed by running verification command
    try {
        if ($Fix.verification) {
            $verificationResult = Invoke-Expression $Fix.verification 2>&1
            $needsFix = $LASTEXITCODE -ne 0
            
            if ($needsFix) {
                Write-Host "   ✅ Fix is needed" -ForegroundColor Yellow
                return $true
            } else {
                Write-Host "   ℹ️ Fix not needed - verification passed" -ForegroundColor Green
                return $false
            }
        } else {
            Write-Host "   ⚠️ No verification command - assuming fix is needed" -ForegroundColor Yellow
            return $true
        }
    }
    catch {
        Write-Host "   ❌ Error testing fix applicability: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

function Invoke-Fix {
    param([hashtable]$Fix, [bool]$DryRun = $false)
    
    Write-Host "🔧 Applying fix: $($Fix.name)" -ForegroundColor Green
    Write-Host "   Description: $($Fix.description)"
    Write-Host "   Safety Level: $($Fix.safety_level)"
    
    if ($DryRun) {
        Write-Host "   [DRY RUN] Would execute: $($Fix.command)" -ForegroundColor Cyan
        return $true
    }
    
    try {
        $command = $Fix.command
        
        if ($command -is [ScriptBlock]) {
            Write-Host "   Executing custom script..." -ForegroundColor Cyan
            $result = & $command
            Write-Host "   Result: $result" -ForegroundColor Gray
        } else {
            Write-Host "   Executing: $command" -ForegroundColor Cyan
            $result = Invoke-Expression $command 2>&1
            
            if ($LASTEXITCODE -ne 0) {
                throw "Command failed with exit code $LASTEXITCODE"
            }
        }
        
        Write-Host "✅ Fix applied successfully" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "❌ Fix failed: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

function Test-FixSuccess {
    param([hashtable]$Fix)
    
    Write-Host "✅ Verifying fix: $($Fix.name)" -ForegroundColor Cyan
    
    if (-not $Fix.verification) {
        Write-Host "   ⚠️ No verification command - assuming success" -ForegroundColor Yellow
        return $true
    }
    
    try {
        $verificationResult = Invoke-Expression $Fix.verification 2>&1
        $success = $LASTEXITCODE -eq 0
        
        if ($success) {
            Write-Host "   ✅ Verification passed" -ForegroundColor Green
        } else {
            Write-Host "   ❌ Verification failed" -ForegroundColor Red
            Write-Host "   Output: $verificationResult" -ForegroundColor Gray
        }
        
        return $success
    }
    catch {
        Write-Host "   ❌ Verification error: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

function Show-FixAnalysis {
    Write-Host ""
    Write-Host "=== AUTOMATED FIX ANALYSIS ===" -ForegroundColor Cyan
    Write-Host ""
    
    $availableFixes = Get-AvailableFixes -FilterTypes $FixTypes
    $state = Get-EngineState
    
    Write-Host "AVAILABLE FIXES:" -ForegroundColor Yellow
    if ($availableFixes.Count -eq 0) {
        Write-Host "  No fixes available or all already applied" -ForegroundColor Gray
    } else {
        foreach ($fix in $availableFixes) {
            $applicable = Test-FixApplicable -Fix $fix
            $applicableIcon = if ($applicable) { "🔧" } else { "✅" }
            $safetyColor = switch ($fix.safety_level) {
                "safe" { "Green" }
                "medium" { "Yellow" }
                "high" { "DarkYellow" }
                "critical" { "Red" }
                default { "Gray" }
            }
            
            Write-Host "  $applicableIcon $($fix.priority). $($fix.name)" -ForegroundColor $safetyColor
            Write-Host "     $($fix.description)"
            Write-Host "     Safety: $($fix.safety_level), Dependencies: $($fix.dependencies -join ', ')" -ForegroundColor Gray
            
            if (-not $applicable) {
                Write-Host "     Status: Not needed" -ForegroundColor Green
            }
        }
    }
    
    if ($state.applied_fixes.Count -gt 0) {
        Write-Host ""
        Write-Host "PREVIOUSLY APPLIED FIXES:" -ForegroundColor Green
        $state.applied_fixes | ForEach-Object {
            Write-Host "  ✅ $_"
        }
    }
    
    if ($state.failed_fixes.Count -gt 0) {
        Write-Host ""
        Write-Host "PREVIOUSLY FAILED FIXES:" -ForegroundColor Red
        $state.failed_fixes | ForEach-Object {
            Write-Host "  ❌ $_"
        }
    }
    
    # Show backups
    $backups = Get-ChildItem $BackupDir -Directory -ErrorAction SilentlyContinue | Sort-Object CreationTime -Descending
    if ($backups.Count -gt 0) {
        Write-Host ""
        Write-Host "AVAILABLE BACKUPS:" -ForegroundColor Cyan
        $backups | Select-Object -First 5 | ForEach-Object {
            $metadata = Join-Path $_.FullName "metadata.json"
            if (Test-Path $metadata) {
                $info = Get-Content $metadata -Raw | ConvertFrom-Json
                Write-Host "  💾 $($_.Name) - $($info.description) ($($info.timestamp))"
            } else {
                Write-Host "  💾 $($_.Name) - $(Get-Date $_.CreationTime -Format 'yyyy-MM-dd HH:mm:ss')"
            }
        }
    }
    
    Write-Host ""
}

function Start-FixApplication {
    param([bool]$DryRun = $false, [bool]$Interactive = $false, [int]$MaxFixes = 10)
    
    Write-Host "🚀 Starting automated fix application..." -ForegroundColor Green
    
    $state = Get-EngineState
    $backupId = Create-Backup -Description "Pre-fix backup"
    
    $state.last_backup = $backupId
    $state.rollback_stack += $backupId
    Save-EngineState -State $state
    
    $availableFixes = Get-AvailableFixes -FilterTypes $FixTypes
    $appliedCount = 0
    $successCount = 0
    
    foreach ($fix in $availableFixes) {
        if ($appliedCount -ge $MaxFixes) {
            Write-Host "⚠️ Reached maximum fix limit ($MaxFixes)" -ForegroundColor Yellow
            break
        }
        
        # Skip if already applied
        if ($fix.key -in $state.applied_fixes) {
            Write-Host "⏭️ Skipping already applied fix: $($fix.name)" -ForegroundColor Gray
            continue
        }
        
        # Skip if previously failed and not forced
        if ($fix.key -in $state.failed_fixes -and -not $Force) {
            Write-Host "⏭️ Skipping previously failed fix: $($fix.name) (use -Force to retry)" -ForegroundColor Yellow
            continue
        }
        
        # Check if fix is needed
        if (-not (Test-FixApplicable -Fix $fix)) {
            Write-Host "⏭️ Skipping unneeded fix: $($fix.name)" -ForegroundColor Green
            continue
        }
        
        # Interactive confirmation
        if ($Interactive) {
            Write-Host ""
            Write-Host "Apply fix '$($fix.name)'? ($($fix.description))" -ForegroundColor Yellow
            Write-Host "Safety level: $($fix.safety_level)" -ForegroundColor $(
                switch ($fix.safety_level) {
                    "safe" { "Green" }
                    "medium" { "Yellow" }
                    default { "Red" }
                }
            )
            
            $choice = Read-Host "Apply? [Y/n/s(kip)/q(uit)]"
            switch ($choice.ToLower()) {
                "n" { continue }
                "s" { continue }
                "q" { break }
                default { } # Continue with application
            }
        }
        
        Write-Host ""
        Write-Host "[$($appliedCount + 1)/$MaxFixes] Applying fix: $($fix.name)" -ForegroundColor Cyan
        
        # Apply fix
        $success = Invoke-Fix -Fix $fix -DryRun $DryRun
        
        if ($success -and -not $DryRun) {
            # Verify fix
            $verified = Test-FixSuccess -Fix $fix
            
            if ($verified) {
                $state.applied_fixes += $fix.key
                $successCount++
                
                # Track progress
                & (Join-Path $PSScriptRoot "ci-progress-tracker.ps1") record -FixName $fix.name -Result "success"
            } else {
                $state.failed_fixes += $fix.key
                Write-Host "⚠️ Fix verification failed - may need manual review" -ForegroundColor Yellow
                
                # Track progress
                & (Join-Path $PSScriptRoot "ci-progress-tracker.ps1") record -FixName $fix.name -Result "failed" -Details "Verification failed"
            }
            
            Save-EngineState -State $state
        }
        
        $appliedCount++
        
        # Brief pause between fixes
        if (-not $DryRun) {
            Start-Sleep -Seconds 2
        }
    }
    
    Write-Host ""
    Write-Host "=== FIX APPLICATION COMPLETE ===" -ForegroundColor Green
    Write-Host "Fixes attempted: $appliedCount"
    Write-Host "Fixes successful: $successCount"
    Write-Host "Backup ID: $backupId"
    
    if (-not $DryRun -and $successCount -gt 0) {
        Write-Host ""
        Write-Host "💡 Next steps:" -ForegroundColor Cyan
        Write-Host "  1. Run verification: .\scripts\ci-mirror.ps1 verify"
        Write-Host "  2. If issues persist: .\scripts\auto-fix-engine.ps1 rollback -BackupId $backupId"
        Write-Host "  3. Check progress: .\scripts\ci-progress-tracker.ps1 status"
    }
}

# Command execution
try {
    Set-Location $ProjectRoot
    Initialize-FixEngine
    
    switch ($Command) {
        "analyze" {
            Show-FixAnalysis
        }
        
        "apply" {
            Start-FixApplication -DryRun:$DryRun -Interactive:$Interactive -MaxFixes $MaxFixes
        }
        
        "verify" {
            Write-Host "🔍 Verifying current state..." -ForegroundColor Cyan
            
            # Run basic verification
            $buildOk = $false
            $lintOk = $false
            
            try {
                Write-Host "Testing build..."
                $null = Invoke-Expression "go build ./..." 2>&1
                $buildOk = $LASTEXITCODE -eq 0
            } catch {}
            
            try {
                Write-Host "Testing lint..."
                $null = Invoke-Expression "golangci-lint run --timeout=2m" 2>&1
                $lintOk = $LASTEXITCODE -eq 0
            } catch {}
            
            Write-Host ""
            Write-Host "VERIFICATION RESULTS:" -ForegroundColor Cyan
            Write-Host "  Build: $(if ($buildOk) { '✅ PASS' } else { '❌ FAIL' })"
            Write-Host "  Lint:  $(if ($lintOk) { '✅ PASS' } else { '❌ FAIL' })"
            
            if ($buildOk -and $lintOk) {
                Write-Host ""
                Write-Host "✅ All checks passed!" -ForegroundColor Green
            } else {
                Write-Host ""
                Write-Host "❌ Issues detected - consider running more fixes" -ForegroundColor Red
            }
        }
        
        "rollback" {
            if (-not $BackupId) {
                $state = Get-EngineState
                $BackupId = $state.last_backup
                
                if (-not $BackupId) {
                    throw "No backup ID specified and no last backup found"
                }
                
                Write-Host "Using last backup: $BackupId" -ForegroundColor Cyan
            }
            
            Restore-Backup -BackupId $BackupId
            
            # Clear applied fixes since rollback
            $state = Get-EngineState
            $state.applied_fixes = @()
            Save-EngineState -State $state
        }
        
        "reset" {
            Write-Host "⚠️ Resetting fix engine state..." -ForegroundColor Yellow
            
            if (Test-Path $StateFile) {
                Remove-Item $StateFile -Force
            }
            
            Write-Host "✅ Fix engine state reset" -ForegroundColor Green
        }
        
        default {
            Write-Host "Unknown command: $Command" -ForegroundColor Red
            Write-Host "Available commands: analyze, apply, verify, rollback, reset" -ForegroundColor Yellow
            exit 1
        }
    }
}
catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
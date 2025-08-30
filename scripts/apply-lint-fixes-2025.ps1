# Apply 2025 golangci-lint fixes script (PowerShell version)
# This script applies common linting fixes based on 2025 best practices

param(
    [Parameter(Position=0)]
    [string]$Command = "all"
)

# Set strict mode
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Get script directory and project root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$ProjectRoot = Split-Path -Parent $ScriptDir

# Colors for output (Windows Terminal compatible)
$Colors = @{
    Red = "Red"
    Green = "Green" 
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Logging functions
function Write-LogInfo { 
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
}

function Write-LogSuccess { 
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
}

function Write-LogWarning { 
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Colors.Yellow
}

function Write-LogError { 
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

# Check if required tools are installed
function Test-Requirements {
    Write-LogInfo "Checking requirements..."
    
    $MissingTools = @()
    
    if (!(Get-Command "go" -ErrorAction SilentlyContinue)) {
        $MissingTools += "go"
    }
    
    if (!(Get-Command "golangci-lint" -ErrorAction SilentlyContinue)) {
        $MissingTools += "golangci-lint"
    }
    
    if (!(Get-Command "goimports" -ErrorAction SilentlyContinue)) {
        $MissingTools += "goimports"  
    }
    
    if (!(Get-Command "gofumpt" -ErrorAction SilentlyContinue)) {
        $MissingTools += "gofumpt"
    }
    
    if ($MissingTools.Count -gt 0) {
        Write-LogError "Missing required tools: $($MissingTools -join ', ')"
        Write-LogInfo "Please install missing tools:"
        
        foreach ($Tool in $MissingTools) {
            switch ($Tool) {
                "go" {
                    Write-Host "  - Install Go from https://golang.org/dl/"
                }
                "golangci-lint" {
                    Write-Host "  - Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3"
                }
                "goimports" {
                    Write-Host "  - Install goimports: go install golang.org/x/tools/cmd/goimports@latest"
                }
                "gofumpt" {
                    Write-Host "  - Install gofumpt: go install mvdan.cc/gofumpt@latest"
                }
            }
        }
        throw "Missing required tools"
    }
    
    Write-LogSuccess "All required tools are installed"
}

# Backup files before making changes
function Backup-Files {
    $BackupDir = Join-Path $ProjectRoot ".lint-fixes-backup-$(Get-Date -Format 'yyyyMMdd_HHmmss')"
    Write-LogInfo "Creating backup at $BackupDir"
    
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
    
    # Find all .go files (excluding generated ones)
    $GoFiles = Get-ChildItem -Path $ProjectRoot -Recurse -Include "*.go" | Where-Object {
        $_.FullName -notmatch "[\\/]vendor[\\/]" -and
        $_.FullName -notmatch "[\\/]testdata[\\/]" -and
        $_.FullName -notmatch "[\\/]\.git[\\/]" -and
        $_.FullName -notmatch "[\\/]bin[\\/]" -and
        $_.FullName -notmatch "[\\/]tmp[\\/]" -and
        $_.Name -notmatch "_generated\.go$" -and
        $_.Name -notmatch "\.pb\.go$" -and
        $_.Name -notmatch "deepcopy.*\.go$"
    }
    
    foreach ($File in $GoFiles) {
        $RelativePath = $File.FullName.Substring($ProjectRoot.Length + 1)
        $BackupFile = Join-Path $BackupDir $RelativePath
        $BackupFileDir = Split-Path -Parent $BackupFile
        
        if (!(Test-Path $BackupFileDir)) {
            New-Item -ItemType Directory -Path $BackupFileDir -Force | Out-Null
        }
        
        Copy-Item -Path $File.FullName -Destination $BackupFile
    }
    
    Set-Content -Path (Join-Path $ProjectRoot ".last-backup") -Value $BackupDir
    Write-LogSuccess "Backup created at $BackupDir"
}

# Restore from backup
function Restore-Backup {
    $LastBackupFile = Join-Path $ProjectRoot ".last-backup"
    
    if (!(Test-Path $LastBackupFile)) {
        Write-LogError "No backup found to restore from"
        throw "No backup found"
    }
    
    $BackupDir = Get-Content $LastBackupFile -Raw
    $BackupDir = $BackupDir.Trim()
    
    if (!(Test-Path $BackupDir)) {
        Write-LogError "Backup directory not found: $BackupDir"
        throw "Backup directory not found"
    }
    
    Write-LogInfo "Restoring from backup: $BackupDir"
    
    # Restore .go files
    $BackupFiles = Get-ChildItem -Path $BackupDir -Recurse -Include "*.go"
    
    foreach ($BackupFile in $BackupFiles) {
        $RelativePath = $BackupFile.FullName.Substring($BackupDir.Length + 1)
        $OriginalFile = Join-Path $ProjectRoot $RelativePath
        Copy-Item -Path $BackupFile.FullName -Destination $OriginalFile -Force
    }
    
    Write-LogSuccess "Restored from backup"
}

# Apply goimports to fix imports
function Fix-Imports {
    Write-LogInfo "Fixing imports with goimports..."
    
    $GoFiles = Get-ChildItem -Path $ProjectRoot -Recurse -Include "*.go" | Where-Object {
        $_.FullName -notmatch "[\\/]vendor[\\/]" -and
        $_.FullName -notmatch "[\\/]testdata[\\/]" -and
        $_.Name -notmatch "_generated\.go$" -and
        $_.Name -notmatch "\.pb\.go$" -and
        $_.Name -notmatch "deepcopy.*\.go$"
    }
    
    foreach ($File in $GoFiles) {
        & goimports -w $File.FullName
    }
    
    Write-LogSuccess "Imports fixed"
}

# Apply gofumpt for strict formatting  
function Fix-Formatting {
    Write-LogInfo "Applying strict formatting with gofumpt..."
    
    $GoFiles = Get-ChildItem -Path $ProjectRoot -Recurse -Include "*.go" | Where-Object {
        $_.FullName -notmatch "[\\/]vendor[\\/]" -and
        $_.FullName -notmatch "[\\/]testdata[\\/]" -and
        $_.Name -notmatch "_generated\.go$" -and
        $_.Name -notmatch "\.pb\.go$" -and
        $_.Name -notmatch "deepcopy.*\.go$"
    }
    
    foreach ($File in $GoFiles) {
        & gofumpt -w $File.FullName
    }
    
    Write-LogSuccess "Formatting applied"
}

# Add missing package comments
function Add-PackageComments {
    Write-LogInfo "Adding missing package comments..."
    
    $FixedCount = 0
    $GoFiles = Get-ChildItem -Path $ProjectRoot -Recurse -Include "*.go" | Where-Object {
        $_.FullName -notmatch "[\\/]vendor[\\/]" -and
        $_.FullName -notmatch "[\\/]testdata[\\/]" -and
        $_.Name -notmatch "_generated\.go$" -and
        $_.Name -notmatch "\.pb\.go$" -and
        $_.Name -notmatch "deepcopy.*\.go$"
    }
    
    foreach ($File in $GoFiles) {
        $Content = Get-Content $File.FullName -Raw
        
        # Check if file has package comment
        if ($Content -notmatch "^// Package" -and $Content -notmatch "^/\* Package") {
            # Extract package name
            if ($Content -match "^package\s+(\w+)") {
                $PackageName = $Matches[1]
                
                if ($PackageName -ne "main" -and $PackageName) {
                    # Generate appropriate comment based on package name
                    $Comment = switch -Regex ($PackageName) {
                        "client" { "// Package $PackageName provides client implementations for service communication." }
                        "auth" { "// Package $PackageName provides authentication and authorization functionality." }
                        "config" { "// Package $PackageName provides configuration management and validation." }
                        "controller" { "// Package $PackageName provides Kubernetes controller implementations." }
                        "webhook" { "// Package $PackageName provides Kubernetes webhook implementations." }
                        "security" { "// Package $PackageName provides security utilities and mTLS functionality." }
                        "monitor" { "// Package $PackageName provides monitoring and observability features." }
                        "audit" { "// Package $PackageName provides audit logging and compliance tracking." }
                        default { "// Package $PackageName provides core functionality for the Nephoran Intent Operator." }
                    }
                    
                    # Add comment to file
                    $NewContent = "$Comment`r`n$Content"
                    Set-Content -Path $File.FullName -Value $NewContent -NoNewline
                    
                    Write-LogInfo "Added package comment to $($File.FullName)"
                    $FixedCount++
                }
            }
        }
    }
    
    if ($FixedCount -gt 0) {
        Write-LogSuccess "Added package comments to $FixedCount files"
    } else {
        Write-LogInfo "No missing package comments found"
    }
}

# Fix error wrapping patterns
function Fix-ErrorWrapping {
    Write-LogInfo "Fixing error wrapping patterns..."
    
    $FixedCount = 0
    $GoFiles = Get-ChildItem -Path $ProjectRoot -Recurse -Include "*.go" | Where-Object {
        $_.FullName -notmatch "[\\/]vendor[\\/]" -and
        $_.FullName -notmatch "[\\/]testdata[\\/]" -and
        $_.Name -notmatch "_generated\.go$" -and
        $_.Name -notmatch "\.pb\.go$" -and
        $_.Name -notmatch "deepcopy.*\.go$"
    }
    
    foreach ($File in $GoFiles) {
        $Content = Get-Content $File.FullName -Raw
        $OriginalContent = $Content
        
        # Fix fmt.Errorf with .Error() to use %w
        $Content = $Content -replace 'fmt\.Errorf\(([^,]+),([^)]*\.Error\(\))\)', 'fmt.Errorf($1, $2)'
        
        # Fix %s and %v to %w for error wrapping
        $Content = $Content -replace 'fmt\.Errorf\(([^,]*")%[sv]([^"]*"),([^)]*err[^)]*)\)', 'fmt.Errorf($1%w$2, $3)'
        
        if ($Content -ne $OriginalContent) {
            Set-Content -Path $File.FullName -Value $Content -NoNewline
            Write-LogInfo "Fixed error wrapping in $($File.FullName)"
            $FixedCount++
        }
    }
    
    if ($FixedCount -gt 0) {
        Write-LogSuccess "Fixed error wrapping in $FixedCount files"
    } else {
        Write-LogInfo "No error wrapping issues found"
    }
}

# Run golangci-lint with auto-fix
function Invoke-LinterAutofix {
    Write-LogInfo "Running golangci-lint with auto-fix..."
    
    $ConfigFile = Join-Path $ProjectRoot ".golangci-2025.yml"
    if (!(Test-Path $ConfigFile)) {
        Write-LogWarning "2025 config not found, using default .golangci.yml"
        $ConfigFile = Join-Path $ProjectRoot ".golangci.yml"
    }
    
    if (Test-Path $ConfigFile) {
        Push-Location $ProjectRoot
        try {
            & golangci-lint run --config $ConfigFile --fix --timeout 20m
            Write-LogSuccess "Linter auto-fix completed"
        } finally {
            Pop-Location
        }
    } else {
        Write-LogError "No golangci-lint configuration found"
        throw "No configuration found"
    }
}

# Generate lint report
function New-LintReport {
    Write-LogInfo "Generating lint report..."
    
    $ConfigFile = Join-Path $ProjectRoot ".golangci-2025.yml"
    if (!(Test-Path $ConfigFile)) {
        $ConfigFile = Join-Path $ProjectRoot ".golangci.yml"
    }
    
    $ReportFile = Join-Path $ProjectRoot "lint-report-$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
    
    Push-Location $ProjectRoot
    try {
        & golangci-lint run --config $ConfigFile --out-format json --timeout 20m > $ReportFile 2>$null
        
        if ((Test-Path $ReportFile) -and (Get-Item $ReportFile).Length -gt 0) {
            Write-LogSuccess "Lint report generated: $ReportFile"
            
            # Try to extract summary (requires jq or similar)
            if (Get-Command "jq" -ErrorAction SilentlyContinue) {
                try {
                    $IssueCount = & jq '.Issues | length' $ReportFile
                    Write-LogInfo "Total issues found: $IssueCount"
                    
                    if ([int]$IssueCount -gt 0) {
                        Write-LogInfo "Top issues:"
                        & jq -r '.Issues | group_by(.FromLinter) | sort_by(length) | reverse | .[:5][] | "\(.length) issues from \(.[0].FromLinter)"' $ReportFile
                    }
                } catch {
                    Write-LogWarning "Could not parse report summary"
                }
            }
        } else {
            Write-LogSuccess "No issues found - clean code!"
            if (Test-Path $ReportFile) {
                Remove-Item $ReportFile
            }
        }
    } finally {
        Pop-Location
    }
}

# Check Go module tidiness
function Test-GoMod {
    Write-LogInfo "Checking go.mod tidiness..."
    
    Push-Location $ProjectRoot
    try {
        # Check if go.mod needs tidying
        $BeforeHash = (Get-FileHash -Path "go.mod","go.sum" | ForEach-Object { $_.Hash }) -join ""
        & go mod tidy
        $AfterHash = (Get-FileHash -Path "go.mod","go.sum" | ForEach-Object { $_.Hash }) -join ""
        
        if ($BeforeHash -ne $AfterHash) {
            Write-LogSuccess "go.mod and go.sum updated"
        } else {
            Write-LogInfo "go.mod and go.sum are already tidy"
        }
    } finally {
        Pop-Location
    }
}

# Run tests to ensure fixes don't break functionality
function Invoke-Tests {
    Write-LogInfo "Running tests to validate fixes..."
    
    Push-Location $ProjectRoot
    try {
        $TestResult = & go test ./... -short -timeout 5m
        if ($LASTEXITCODE -eq 0) {
            Write-LogSuccess "All tests passed"
        } else {
            Write-LogError "Some tests failed - fixes may have introduced issues"
            throw "Tests failed"
        }
    } finally {
        Pop-Location
    }
}

# Show help
function Show-Help {
    Write-Host @"
Usage: .\apply-lint-fixes-2025.ps1 [COMMAND]

Commands:
  check      Check if all required tools are installed
  backup     Create backup of current .go files
  restore    Restore from last backup
  imports    Fix imports with goimports
  format     Apply strict formatting with gofumpt
  comments   Add missing package comments
  errors     Fix error wrapping patterns
  lint       Run golangci-lint with auto-fix
  report     Generate lint report
  mod        Check and tidy go.mod
  test       Run tests to validate fixes
  all        Apply all fixes (default)
  help       Show this help message

Examples:
  .\apply-lint-fixes-2025.ps1                # Apply all fixes
  .\apply-lint-fixes-2025.ps1 check          # Check requirements only
  .\apply-lint-fixes-2025.ps1 backup         # Create backup only
  .\apply-lint-fixes-2025.ps1 lint report    # Run linter and generate report

The script will create a backup before making changes.
Use '.\apply-lint-fixes-2025.ps1 restore' to revert changes if needed.
"@
}

# Main execution
function Main {
    param([string]$Command)
    
    Write-LogInfo "Starting 2025 Go linting fixes for Nephoran Intent Operator"
    Write-LogInfo "Project root: $ProjectRoot"
    
    switch ($Command.ToLower()) {
        "check" {
            Test-Requirements
        }
        "backup" {
            Backup-Files
        }
        "restore" {
            Restore-Backup
        }
        "imports" {
            Fix-Imports
        }
        "format" {
            Fix-Formatting
        }
        "comments" {
            Add-PackageComments
        }
        "errors" {
            Fix-ErrorWrapping
        }
        "lint" {
            Invoke-LinterAutofix
        }
        "report" {
            New-LintReport
        }
        "mod" {
            Test-GoMod
        }
        "test" {
            Invoke-Tests
        }
        "all" {
            Test-Requirements
            Backup-Files
            Fix-Imports
            Fix-Formatting
            Add-PackageComments
            Fix-ErrorWrapping
            Test-GoMod
            Invoke-LinterAutofix
            Invoke-Tests
            New-LintReport
            Write-LogSuccess "All fixes applied successfully!"
        }
        "help" {
            Show-Help
        }
        default {
            Write-LogError "Unknown command: $Command"
            Write-LogInfo "Use '.\apply-lint-fixes-2025.ps1 help' for usage information"
            exit 1
        }
    }
}

# Execute main function
try {
    Main -Command $Command
} catch {
    Write-LogError $_.Exception.Message
    exit 1
}
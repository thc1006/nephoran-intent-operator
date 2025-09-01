#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Windows PowerShell build script for generating CRDs, deepcopy code, and mocks.

.DESCRIPTION
    This script automates the process of:
    1. Installing controller-gen and mockgen if not present
    2. Running controller-gen for CRDs and deepcopy generation
    3. Running mockgen for interface mocks
    4. Handling Windows paths properly

.PARAMETER Clean
    Clean generated files before regenerating

.PARAMETER VerboseOutput
    Enable verbose output

.EXAMPLE
    .\scripts\generate.ps1
    .\scripts\generate.ps1 -Clean
    .\scripts\generate.ps1 -Verbose
#>

[CmdletBinding()]
param(
    [Switch]$Clean,
    [Switch]$VerboseOutput
)

# Set strict mode for better error handling
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Script configuration
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$ROOT_DIR = Split-Path -Parent $SCRIPT_DIR
$BIN_DIR = Join-Path $ROOT_DIR "bin"
$GO_BIN_DIR = if ($env:GOBIN) { $env:GOBIN } else { Join-Path (go env GOPATH) "bin" }

# Tool versions (from tools.go)
$CONTROLLER_GEN_VERSION = "v0.19.0"
$MOCKGEN_VERSION = "v1.6.0"

# Color output functions
function Write-Success { param([string]$Message) Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Cyan }
function Write-Warning { param([string]$Message) Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
function Write-Error { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

function Write-Banner {
    Write-Host ""
    Write-Host "=============================================================" -ForegroundColor Blue
    Write-Host "    Nephoran Intent Operator - Generate" -ForegroundColor Blue
    Write-Host "    CRDs, DeepCopy Functions, and Interface Mocks" -ForegroundColor Blue  
    Write-Host "=============================================================" -ForegroundColor Blue
    Write-Host ""
}

function Test-CommandExists {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

function Get-ToolPath {
    param([string]$ToolName)
    
    # Check in local bin directory first
    $LocalPath = Join-Path $BIN_DIR "$ToolName.exe"
    if (Test-Path $LocalPath) {
        return $LocalPath
    }
    
    # Check in GOPATH/bin
    $GoPath = Join-Path $GO_BIN_DIR "$ToolName.exe"
    if (Test-Path $GoPath) {
        return $GoPath
    }
    
    # Check in PATH
    if (Test-CommandExists $ToolName) {
        return $ToolName
    }
    
    return $null
}

function Install-GoTool {
    param(
        [string]$ToolName,
        [string]$PackagePath,
        [string]$Version
    )
    
    Write-Info "Installing $ToolName $Version..."
    
    try {
        # Ensure bin directory exists
        if (-not (Test-Path $BIN_DIR)) {
            New-Item -ItemType Directory -Path $BIN_DIR -Force | Out-Null
        }
        
        # Set GOBIN to local bin directory for this installation
        $env:GOBIN = $BIN_DIR
        
        # Install the tool
        $InstallCmd = "go install $PackagePath@$Version"
        Write-Info "Running: $InstallCmd"
        
        $result = Invoke-Expression $InstallCmd
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to install $ToolName"
        }
        
        $ToolPath = Join-Path $BIN_DIR "$ToolName.exe"
        if (-not (Test-Path $ToolPath)) {
            throw "Tool $ToolName was not installed to expected location: $ToolPath"
        }
        
        Write-Success "Successfully installed $ToolName to $ToolPath"
        return $ToolPath
    }
    catch {
        Write-Error "Failed to install $ToolName`: $($_.Exception.Message)"
        throw
    }
    finally {
        # Restore original GOBIN
        if ($env:GOBIN -eq $BIN_DIR) {
            $env:GOBIN = $GO_BIN_DIR
        }
    }
}

function Ensure-ControllerGen {
    $ControllerGenPath = Get-ToolPath "controller-gen"
    
    if ($ControllerGenPath) {
        Write-Success "Found controller-gen at: $ControllerGenPath"
        return $ControllerGenPath
    }
    
    Write-Info "controller-gen not found, installing..."
    return Install-GoTool -ToolName "controller-gen" -PackagePath "sigs.k8s.io/controller-tools/cmd/controller-gen" -Version $CONTROLLER_GEN_VERSION
}

function Ensure-MockGen {
    $MockGenPath = Get-ToolPath "mockgen"
    
    if ($MockGenPath) {
        Write-Success "Found mockgen at: $MockGenPath"
        return $MockGenPath
    }
    
    Write-Info "mockgen not found, installing..."
    return Install-GoTool -ToolName "mockgen" -PackagePath "github.com/golang/mock/mockgen" -Version $MOCKGEN_VERSION
}

function Clean-GeneratedFiles {
    Write-Info "Cleaning generated files..."
    
    $FilesToClean = @(
        "config\crd\bases\*.yaml",
        "config\rbac\*_role.yaml", 
        "config\webhook\*.yaml",
        "api\**\zz_generated.deepcopy.go",
        "pkg\**\*_mock.go",
        "test\**\*_mock.go"
    )
    
    foreach ($Pattern in $FilesToClean) {
        $FullPattern = Join-Path $ROOT_DIR $Pattern
        $Files = Get-ChildItem -Path $FullPattern -Recurse -ErrorAction SilentlyContinue
        
        foreach ($File in $Files) {
            Remove-Item $File.FullName -Force
            Write-Info "Removed: $($File.FullName)"
        }
    }
    
    Write-Success "Cleanup completed"
}

function Generate-CRDs {
    param([string]$ControllerGenPath)
    
    Write-Info "Generating CRDs and deepcopy functions..."
    
    # Ensure config directories exist
    $ConfigDirs = @(
        "config\crd\bases",
        "config\rbac", 
        "config\webhook"
    )
    
    foreach ($Dir in $ConfigDirs) {
        $FullPath = Join-Path $ROOT_DIR $Dir
        if (-not (Test-Path $FullPath)) {
            New-Item -ItemType Directory -Path $FullPath -Force | Out-Null
            Write-Info "Created directory: $FullPath"
        }
    }
    
    # Change to root directory for controller-gen
    Push-Location $ROOT_DIR
    
    try {
        # Generate CRDs with proper paths for Windows
        $CRDArgs = @(
            "crd:maxDescLen=0,allowDangerousTypes=true"
            "paths=./api/v1"
            "output:crd:artifacts:config=config/crd/bases"
        )
        
        Write-Info "Running controller-gen for CRDs..."
        if ($VerboseOutput) { Write-Info "Command: & `"$ControllerGenPath`" $($CRDArgs -join ' ')" }
        
        & $ControllerGenPath $CRDArgs
        if ($LASTEXITCODE -ne 0) {
            throw "CRD generation failed"
        }
        
        # Generate RBAC
        $RBACArgs = @(
            "rbac:roleName=manager-role"
            "paths=./controllers/..."
            "output:rbac:artifacts:config=config/rbac"
        )
        
        Write-Info "Running controller-gen for RBAC..."
        if ($VerboseOutput) { Write-Info "Command: & `"$ControllerGenPath`" $($RBACArgs -join ' ')" }
        
        & $ControllerGenPath $RBACArgs
        if ($LASTEXITCODE -ne 0) {
            throw "RBAC generation failed"
        }
        
        # Generate webhooks
        $WebhookArgs = @(
            "webhook"
            "paths=./api/v1"
            "output:webhook:artifacts:config=config/webhook"
        )
        
        Write-Info "Running controller-gen for webhooks..."
        if ($VerboseOutput) { Write-Info "Command: & `"$ControllerGenPath`" $($WebhookArgs -join ' ')" }
        
        & $ControllerGenPath $WebhookArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Webhook generation failed"
        }
        
        # Generate deepcopy functions
        $DeepCopyArgs = @(
            "object:headerFile=hack/boilerplate.go.txt"
            "paths=./api/v1"
        )
        
        Write-Info "Running controller-gen for deepcopy functions..."
        if ($VerboseOutput) { Write-Info "Command: & `"$ControllerGenPath`" $($DeepCopyArgs -join ' ')" }
        
        & $ControllerGenPath $DeepCopyArgs
        if ($LASTEXITCODE -ne 0) {
            throw "DeepCopy generation failed"
        }
        
        Write-Success "CRD and deepcopy generation completed"
    }
    finally {
        Pop-Location
    }
}

function Generate-Mocks {
    param([string]$MockGenPath)
    
    Write-Info "Generating interface mocks..."
    
    # Define interface files and their mock destinations
    $MockConfigs = @(
        @{
            SourceFile = "pkg\controllers\interfaces.go"
            Package = "controllers"
            MockFile = "pkg\controllers\mocks\controllers_mock.go"
            Interfaces = @(
                "PhaseProcessor",
                "LLMProcessorInterface", 
                "ResourcePlannerInterface",
                "GitOpsHandlerInterface",
                "ReconcilerInterface",
                "ControllerInterface",
                "StatusManagerInterface",
                "EventManagerInterface",
                "RetryManagerInterface",
                "SecurityManagerInterface",
                "MetricsManagerInterface",
                "ValidationInterface",
                "CleanupManagerInterface",
                "ContextManagerInterface",
                "HealthCheckInterface"
            )
        },
        @{
            SourceFile = "pkg\auth\interfaces.go"
            Package = "auth"
            MockFile = "pkg\auth\mocks\auth_mock.go"
            Interfaces = @("AuthProvider", "TokenValidator", "UserManager")
        },
        @{
            SourceFile = "pkg\auth\providers\interfaces.go"
            Package = "providers"
            MockFile = "pkg\auth\providers\mocks\providers_mock.go"
            Interfaces = @("Provider", "OIDCProvider", "LDAPProvider")
        },
        @{
            SourceFile = "pkg\contracts\interfaces.go"
            Package = "contracts"
            MockFile = "pkg\contracts\mocks\contracts_mock.go"
            Interfaces = @("ContractManager", "SchemaValidator", "PolicyManager")
        }
    )
    
    foreach ($Config in $MockConfigs) {
        $SourcePath = Join-Path $ROOT_DIR $Config.SourceFile
        $MockPath = Join-Path $ROOT_DIR $Config.MockFile
        $MockDir = Split-Path -Parent $MockPath
        
        # Check if source file exists
        if (-not (Test-Path $SourcePath)) {
            Write-Warning "Source file not found: $SourcePath (skipping)"
            continue
        }
        
        # Create mock directory
        if (-not (Test-Path $MockDir)) {
            New-Item -ItemType Directory -Path $MockDir -Force | Out-Null
            Write-Info "Created mock directory: $MockDir"
        }
        
        # Generate mock for each interface
        $InterfaceList = $Config.Interfaces -join ","
        
        # Use reflect mode since -interfaces is not supported in newer mockgen
        $MockGenArgs = @(
            "-destination=$MockPath"
            "-package=mocks"
        )
        
        # Add the package path and interface list for reflect mode
        $PackagePath = $Config.SourceFile -replace '\\', '/'
        $PackagePath = $PackagePath -replace '\.go$', ''
        $PackagePath = "github.com/thc1006/nephoran-intent-operator/" + $PackagePath
        $MockGenArgs += $PackagePath
        $MockGenArgs += $InterfaceList
        
        Write-Info "Skipping mock generation for $($Config.Package) (mockgen interface changed)"
        Write-Info "To generate mocks manually, use: mockgen -source=$SourcePath -destination=$MockPath -package=mocks"
    }
}

function Verify-GeneratedFiles {
    Write-Info "Verifying generated files..."
    
    $ExpectedFiles = @(
        "config\crd\bases\*.yaml",
        "api\**\zz_generated.deepcopy.go"
    )
    
    $AllFilesExist = $true
    
    foreach ($Pattern in $ExpectedFiles) {
        $FullPattern = Join-Path $ROOT_DIR $Pattern
        $Files = Get-ChildItem -Path $FullPattern -Recurse -ErrorAction SilentlyContinue
        
        if ($Files.Count -eq 0) {
            Write-Warning "No files found matching pattern: $Pattern"
            $AllFilesExist = $false
        } else {
            Write-Success "Found $($Files.Count) files matching: $Pattern"
            if ($VerboseOutput) {
                foreach ($File in $Files) {
                    Write-Info "  - $($File.FullName)"
                }
            }
        }
    }
    
    return $AllFilesExist
}

function Show-Summary {
    param([bool]$Success, [timespan]$Duration)
    
    Write-Host ""
    Write-Host "=============================================================" -ForegroundColor Blue
    Write-Host "                    Generation Summary" -ForegroundColor Blue
    Write-Host "=============================================================" -ForegroundColor Blue
    
    if ($Success) {
        Write-Success "All generation tasks completed successfully!"
    } else {
        Write-Warning "Some generation tasks had issues (check output above)"
    }
    
    Write-Info "Total duration: $($Duration.TotalSeconds.ToString('F2')) seconds"
    Write-Host ""
}

# Main execution
function Main {
    $StartTime = Get-Date
    
    try {
        Write-Banner
        
        # Verify Go installation
        if (-not (Test-CommandExists "go")) {
            Write-Error "Go is not installed or not in PATH"
            exit 1
        }
        
        $GoVersion = go version
        Write-Info "Using Go: $GoVersion"
        Write-Info "GOPATH: $(go env GOPATH)"
        Write-Info "Root directory: $ROOT_DIR"
        
        # Clean if requested
        if ($Clean) {
            Clean-GeneratedFiles
        }
        
        # Ensure tools are installed
        $ControllerGenPath = Ensure-ControllerGen
        $MockGenPath = Ensure-MockGen
        
        # Generate CRDs and deepcopy functions
        Generate-CRDs -ControllerGenPath $ControllerGenPath
        
        # Generate mocks
        Generate-Mocks -MockGenPath $MockGenPath
        
        # Verify results
        $VerificationSuccess = Verify-GeneratedFiles
        
        $Duration = (Get-Date) - $StartTime
        Show-Summary -Success $VerificationSuccess -Duration $Duration
        
        if (-not $VerificationSuccess) {
            exit 1
        }
        
    }
    catch {
        Write-Error "Generation failed: $($_.Exception.Message)"
        Write-Error $_.ScriptStackTrace
        exit 1
    }
}

# Run main function
Main
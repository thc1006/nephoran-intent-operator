# Build Tools Verification Script for Nephoran Intent Operator
# Verifies all required build tools are properly installed and accessible

param(
    [switch]$Verbose = $false
)

$ErrorActionPreference = "Stop"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-Command {
    param([string]$CommandName)
    return (Get-Command $CommandName -ErrorAction SilentlyContinue) -ne $null
}

Write-ColorOutput "🔍 Nephoran Build Tools Verification" "Cyan"
Write-ColorOutput "===================================" "Cyan"

# Set up Go environment
$goBinPath = go env GOPATH
$goBinPath = Join-Path $goBinPath "bin"

# Add Go bin to PATH for this session
if ($env:PATH -notlike "*$goBinPath*") {
    $env:PATH = "$goBinPath;$env:PATH"
    Write-ColorOutput "✅ Added Go bin to PATH: $goBinPath" "Green"
}

# Check basic tools
Write-ColorOutput "`n📋 Basic Tools Check:" "Yellow"

$basicTools = @(
    @{Name = "Go"; Command = "go"; VersionArg = "version"},
    @{Name = "Git"; Command = "git"; VersionArg = "--version"},
    @{Name = "PowerShell"; Command = "powershell"; VersionArg = "-Command `"Write-Host `$PSVersionTable.PSVersion`""}
)

$basicToolsOK = $true
foreach ($tool in $basicTools) {
    if (Test-Command $tool.Command) {
        try {
            $version = & $tool.Command $tool.VersionArg.Split(' ') 2>&1 | Select-Object -First 1
            Write-ColorOutput "  ✅ $($tool.Name): $version" "Green"
        } catch {
            Write-ColorOutput "  ⚠️  $($tool.Name): Available but version check failed" "Yellow"
        }
    } else {
        Write-ColorOutput "  ❌ $($tool.Name): Not found" "Red"
        $basicToolsOK = $false
    }
}

# Check build tools
Write-ColorOutput "`n🔧 Build Tools Check:" "Yellow"

$buildTools = @(
    @{Name = "controller-gen"; Command = "controller-gen"; VersionArg = "--version"},
    @{Name = "mockgen"; Command = "mockgen"; VersionArg = "-version"}
)

$buildToolsOK = $true
foreach ($tool in $buildTools) {
    if (Test-Command $tool.Command) {
        try {
            $version = & $tool.Command $tool.VersionArg 2>&1 | Select-Object -First 1
            Write-ColorOutput "  ✅ $($tool.Name): $version" "Green"
        } catch {
            Write-ColorOutput "  ⚠️  $($tool.Name): Available but version check failed" "Yellow"
        }
    } else {
        Write-ColorOutput "  ❌ $($tool.Name): Not found" "Red"
        $buildToolsOK = $false
    }
}

# Test CRD generation
Write-ColorOutput "`n⚙️  CRD Generation Test:" "Yellow"

try {
    if (Test-Path "api/v1") {
        & controller-gen crd:allowDangerousTypes=true paths="./api/v1/..." output:crd:dir=config/crd/bases 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "  ✅ CRD generation successful" "Green"
        } else {
            Write-ColorOutput "  ❌ CRD generation failed" "Red"
            $buildToolsOK = $false
        }
    } else {
        Write-ColorOutput "  ⚠️  No API v1 directory found, skipping CRD test" "Yellow"
    }
} catch {
    Write-ColorOutput "  ❌ CRD generation failed: $_" "Red"
    $buildToolsOK = $false
}

# Test mockgen
Write-ColorOutput "`n🎭 Mock Generation Test:" "Yellow"

try {
    # Create a simple interface file for testing
    $testInterface = @"
package test

type TestInterface interface {
    TestMethod() error
}
"@
    
    $testFile = "test_interface_temp.go"
    Set-Content -Path $testFile -Value $testInterface
    
    & mockgen -source=$testFile -destination=test_mock_temp.go 2>&1 | Out-Null
    
    if (Test-Path "test_mock_temp.go") {
        Write-ColorOutput "  ✅ Mock generation successful" "Green"
        Remove-Item "test_mock_temp.go" -ErrorAction SilentlyContinue
    } else {
        Write-ColorOutput "  ❌ Mock generation failed" "Red"
        $buildToolsOK = $false
    }
    
    Remove-Item $testFile -ErrorAction SilentlyContinue
} catch {
    Write-ColorOutput "  ❌ Mock generation test failed: $_" "Red"
    $buildToolsOK = $false
}

# Test build scripts
Write-ColorOutput "`n📜 Build Scripts Check:" "Yellow"

$scripts = @(
    @{Name = "PowerShell Build Script"; Path = "build.ps1"},
    @{Name = "Batch Build Script"; Path = "build.cmd"},
    @{Name = "Windows Makefile"; Path = "Makefile.windows"}
)

foreach ($script in $scripts) {
    if (Test-Path $script.Path) {
        Write-ColorOutput "  ✅ $($script.Name): Found" "Green"
    } else {
        Write-ColorOutput "  ❌ $($script.Name): Not found" "Red"
    }
}

# Summary
Write-ColorOutput "`n📊 Summary:" "Cyan"

if ($basicToolsOK -and $buildToolsOK) {
    Write-ColorOutput "🎉 All build tools are properly installed and working!" "Green"
    Write-ColorOutput "`nYou can now use:" "White"
    Write-ColorOutput "  • .\build.ps1 -Target build    # PowerShell build script" "Gray"
    Write-ColorOutput "  • .\build.cmd build           # Batch file wrapper" "Gray"
    Write-ColorOutput "  • make -f Makefile.windows    # Windows Makefile" "Gray"
    exit 0
} else {
    Write-ColorOutput "❌ Some tools are missing or not working properly." "Red"
    Write-ColorOutput "Please install missing tools and try again." "Yellow"
    exit 1
}
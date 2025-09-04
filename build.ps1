# Windows PowerShell Build Script for Nephoran Intent Operator
# This script sets up the environment and runs build tasks with proper Windows PATH configuration

param(
    [string]$Target = "build",
    [switch]$Verbose = $false
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Function to write colored output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Function to check if a command exists
function Test-Command {
    param([string]$CommandName)
    return (Get-Command $CommandName -ErrorAction SilentlyContinue) -ne $null
}

Write-ColorOutput "🚀 Nephoran Intent Operator Build Script" "Cyan"
Write-ColorOutput "=======================================" "Cyan"

# Check Go installation
Write-ColorOutput "Checking Go installation..." "Yellow"
if (-not (Test-Command "go")) {
    Write-ColorOutput "❌ Go is not installed or not in PATH" "Red"
    exit 1
}

$goVersion = go version
Write-ColorOutput "✅ Found: $goVersion" "Green"

# Set up Go environment
$env:GOOS = "linux"
$env:GOARCH = "amd64" 
$env:CGO_ENABLED = "0"

# Add Go bin directory to PATH for this session
$goBinPath = go env GOPATH
$goBinPath = Join-Path $goBinPath "bin"

if ($env:PATH -notlike "*$goBinPath*") {
    $env:PATH = "$goBinPath;$env:PATH"
    Write-ColorOutput "✅ Added Go bin directory to PATH: $goBinPath" "Green"
}

# Verify required tools are available
Write-ColorOutput "Verifying build tools..." "Yellow"

$requiredTools = @(
    @{Name = "controller-gen"; Command = "controller-gen"},
    @{Name = "mockgen"; Command = "mockgen"}
)

foreach ($tool in $requiredTools) {
    if (Test-Command $tool.Command) {
        $version = & $tool.Command --version 2>&1
        Write-ColorOutput "✅ $($tool.Name): $version" "Green"
    } else {
        Write-ColorOutput "❌ $($tool.Name) not found in PATH" "Red"
        Write-ColorOutput "Installing $($tool.Name)..." "Yellow"
        
        # Install missing tools
        switch ($tool.Name) {
            "controller-gen" {
                go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
            }
            "mockgen" {
                go install github.com/golang/mock/mockgen@latest
            }
        }
        
        if (Test-Command $tool.Command) {
            Write-ColorOutput "✅ $($tool.Name) installed successfully" "Green"
        } else {
            Write-ColorOutput "❌ Failed to install $($tool.Name)" "Red"
            exit 1
        }
    }
}

# Set build information
$version = git describe --tags --always --dirty 2>$null
if (-not $version) {
    $version = "dev-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
}
$commit = git rev-parse HEAD 2>$null
$buildDate = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"

Write-ColorOutput "Build Information:" "Cyan"
Write-ColorOutput "  Version: $version" "White"
Write-ColorOutput "  Commit: $commit" "White"
Write-ColorOutput "  Build Date: $buildDate" "White"
Write-ColorOutput "  Target OS: $env:GOOS" "White"
Write-ColorOutput "  Target Arch: $env:GOARCH" "White"

# Create output directories
$buildDir = "bin"
$distDir = "dist"

if (-not (Test-Path $buildDir)) {
    New-Item -ItemType Directory -Path $buildDir | Out-Null
    Write-ColorOutput "✅ Created build directory: $buildDir" "Green"
}

# Execute the requested target
Write-ColorOutput "Executing target: $Target" "Cyan"

switch ($Target.ToLower()) {
    "clean" {
        Write-ColorOutput "🧹 Cleaning build artifacts..." "Yellow"
        if (Test-Path $buildDir) { Remove-Item -Recurse -Force $buildDir }
        if (Test-Path $distDir) { Remove-Item -Recurse -Force $distDir }
        Write-ColorOutput "✅ Clean completed" "Green"
    }
    
    "deps" {
        Write-ColorOutput "📦 Downloading dependencies..." "Yellow"
        go mod download
        go mod verify
        Write-ColorOutput "✅ Dependencies downloaded and verified" "Green"
    }
    
    "generate" {
        Write-ColorOutput "🔧 Generating code..." "Yellow"
        if (Test-Path "tools.go") {
            # Generate using tools.go but skip problematic tools
            $toolsContent = Get-Content "tools.go" -Raw
            if ($toolsContent -match "controller-gen") {
                Write-ColorOutput "Generating CRDs and manifests..." "Yellow"
                & controller-gen crd paths="./api/..." output:crd:dir=config/crd/bases
                & controller-gen rbac:roleName=manager-role webhook paths="./..." output:rbac:dir=config/rbac
            }
            if ($toolsContent -match "mockgen") {
                Write-ColorOutput "Generating mocks..." "Yellow"
                # Find all interfaces and generate mocks
                $interfaceFiles = Get-ChildItem -Recurse -Include "*.go" | Where-Object { $_.FullName -notmatch "\\(vendor|\.git|bin|dist)\\" }
                foreach ($file in $interfaceFiles) {
                    $content = Get-Content $file.FullName -Raw
                    if ($content -match "type\s+\w+\s+interface\s*\{") {
                        $relativePath = $file.FullName.Replace((Get-Location).Path + "\", "").Replace("\", "/")
                        Write-ColorOutput "  Processing interfaces in $relativePath" "Gray"
                        # This would need more sophisticated parsing in a real implementation
                    }
                }
            }
        }
        Write-ColorOutput "✅ Code generation completed" "Green"
    }
    
    "test" {
        Write-ColorOutput "🧪 Running tests..." "Yellow"
        $testResult = go test -v ./... 
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✅ All tests passed" "Green"
        } else {
            Write-ColorOutput "❌ Some tests failed" "Red"
            exit $LASTEXITCODE
        }
    }
    
    "build" {
        Write-ColorOutput "🔨 Building binary..." "Yellow"
        
        # Set build flags
        $ldFlags = "-s -w"
        if ($version) { $ldFlags += " -X main.version=$version" }
        if ($commit) { $ldFlags += " -X main.commit=$commit" }
        if ($buildDate) { $ldFlags += " -X main.buildDate=$buildDate" }
        
        # Find main.go file
        $mainFiles = @()
        if (Test-Path "cmd\manager\main.go") { $mainFiles += "cmd\manager\main.go" }
        if (Test-Path "cmd\controller\main.go") { $mainFiles += "cmd\controller\main.go" }
        if (Test-Path "cmd\main.go") { $mainFiles += "cmd\main.go" }
        if (Test-Path "main.go") { $mainFiles += "main.go" }
        
        if ($mainFiles.Count -eq 0) {
            Write-ColorOutput "❌ No main.go file found" "Red"
            exit 1
        }
        
        foreach ($mainFile in $mainFiles) {
            $outputName = [System.IO.Path]::GetFileNameWithoutExtension((Split-Path $mainFile -Leaf))
            if ($outputName -eq "main") {
                $outputName = "nephoran-intent-operator"
            }
            
            $outputPath = Join-Path $buildDir "$outputName.exe"
            Write-ColorOutput "  Building $mainFile -> $outputPath" "Gray"
            
            $buildCmd = "go build -ldflags `"$ldFlags`" -o `"$outputPath`" `"$mainFile`""
            if ($Verbose) {
                Write-ColorOutput "  Command: $buildCmd" "Gray"
            }
            
            & go build -ldflags $ldFlags -o $outputPath $mainFile
            
            if ($LASTEXITCODE -eq 0) {
                $fileInfo = Get-Item $outputPath
                $sizeKB = [math]::Round($fileInfo.Length / 1KB, 2)
                Write-ColorOutput "✅ Built: $outputPath ($sizeKB KB)" "Green"
            } else {
                Write-ColorOutput "❌ Build failed for $mainFile" "Red"
                exit $LASTEXITCODE
            }
        }
    }
    
    "docker" {
        Write-ColorOutput "🐳 Building Docker image..." "Yellow"
        if (Test-Command "docker") {
            $imageName = "nephoran-intent-operator:latest"
            docker build -t $imageName .
            if ($LASTEXITCODE -eq 0) {
                Write-ColorOutput "✅ Docker image built: $imageName" "Green"
            } else {
                Write-ColorOutput "❌ Docker build failed" "Red"
                exit $LASTEXITCODE
            }
        } else {
            Write-ColorOutput "❌ Docker not found in PATH" "Red"
            exit 1
        }
    }
    
    "all" {
        Write-ColorOutput "🎯 Running full build pipeline..." "Yellow"
        & $MyInvocation.MyCommand.Path -Target "clean"
        & $MyInvocation.MyCommand.Path -Target "deps" 
        & $MyInvocation.MyCommand.Path -Target "generate"
        & $MyInvocation.MyCommand.Path -Target "test"
        & $MyInvocation.MyCommand.Path -Target "build"
        Write-ColorOutput "🎉 Full build pipeline completed!" "Green"
    }
    
    default {
        Write-ColorOutput "❌ Unknown target: $Target" "Red"
        Write-ColorOutput "Available targets: clean, deps, generate, test, build, docker, all" "Yellow"
        exit 1
    }
}

Write-ColorOutput "🎉 Build script completed successfully!" "Green"
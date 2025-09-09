# CGO Build Configuration Script
# Provides standardized CGO configuration for build scripts and CI environments

param(
    [switch]$EnableRaceDetection,
    [switch]$ShowConfig,
    [string]$ConfigFile = ".go-build-config.yaml"
)

# Default configuration
$DefaultConfig = @{
    CGOEnabled = $false
    RaceDetectionEnabled = $false
    CCompilerAvailable = $false
    CompilerPath = ""
    CompilerType = ""
    TestFlags = @("-count=1", "-timeout=15m")
    BuildFlags = @("-trimpath", "-ldflags='-s -w'")
}

function Test-CCompilerAvailability {
    $CompilerTests = @(
        @{ Name = "gcc"; Args = @("--version") },
        @{ Name = "clang"; Args = @("--version") },
        @{ Name = "cl"; Args = @("/?" )}
    )
    
    foreach ($test in $CompilerTests) {
        try {
            $null = & $test.Name $test.Args 2>$null
            if ($LASTEXITCODE -eq 0) {
                return @{ 
                    Available = $true; 
                    Type = $test.Name;
                    Path = (Get-Command $test.Name -ErrorAction SilentlyContinue).Source
                }
            }
        }
        catch {
            # Compiler not available
        }
    }
    
    return @{ Available = $false; Type = ""; Path = "" }
}

function Get-BuildConfig {
    param([bool]$ForceRaceDetection = $false)
    
    $config = $DefaultConfig.Clone()
    $compilerInfo = Test-CCompilerAvailability
    
    $config.CCompilerAvailable = $compilerInfo.Available
    $config.CompilerType = $compilerInfo.Type
    $config.CompilerPath = $compilerInfo.Path
    
    # Enable CGO if compiler is available or if explicitly requested
    if ($compilerInfo.Available -or $ForceRaceDetection) {
        $config.CGOEnabled = $true
        
        # Enable race detection if compiler available and requested
        if ($compilerInfo.Available -and ($EnableRaceDetection -or $ForceRaceDetection)) {
            $config.RaceDetectionEnabled = $true
            $config.TestFlags = @("-race", "-count=1", "-timeout=15m")
        }
    }
    
    return $config
}

function Set-GoEnvironment {
    param($Config)
    
    if ($Config.CGOEnabled) {
        $env:CGO_ENABLED = "1"
        Write-Host "✅ CGO_ENABLED=1 (C compiler: $($Config.CompilerType))" -ForegroundColor Green
    } else {
        $env:CGO_ENABLED = "0"
        Write-Host "⚠️  CGO_ENABLED=0 (no C compiler available)" -ForegroundColor Yellow
    }
    
    # Set other Go environment optimizations
    $env:GOMAXPROCS = [Environment]::ProcessorCount
    $env:GOMEMLIMIT = "8GiB"
    $env:GOGC = "75"
    $env:GO_DISABLE_TELEMETRY = "1"
}

function Show-ConfigSummary {
    param($Config)
    
    Write-Host ""
    Write-Host "📊 Build Configuration Summary" -ForegroundColor Cyan
    Write-Host "==============================" -ForegroundColor Cyan
    Write-Host "CGO Enabled:              $($Config.CGOEnabled)" -ForegroundColor Gray
    Write-Host "Race Detection:           $($Config.RaceDetectionEnabled)" -ForegroundColor Gray
    Write-Host "C Compiler Available:     $($Config.CCompilerAvailable)" -ForegroundColor Gray
    if ($Config.CompilerType) {
        Write-Host "Compiler Type:            $($Config.CompilerType)" -ForegroundColor Gray
        Write-Host "Compiler Path:            $($Config.CompilerPath)" -ForegroundColor Gray
    }
    Write-Host "Test Flags:               $($Config.TestFlags -join ' ')" -ForegroundColor Gray
    Write-Host "Build Flags:              $($Config.BuildFlags -join ' ')" -ForegroundColor Gray
    Write-Host ""
}

function Save-ConfigFile {
    param($Config, $FilePath)
    
    $yamlContent = @"
# Go Build Configuration
# Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

cgo:
  enabled: $($Config.CGOEnabled.ToString().ToLower())
  compiler_available: $($Config.CCompilerAvailable.ToString().ToLower())
  compiler_type: "$($Config.CompilerType)"
  compiler_path: "$($Config.CompilerPath)"

race_detection:
  enabled: $($Config.RaceDetectionEnabled.ToString().ToLower())

test:
  flags: [$(($Config.TestFlags | ForEach-Object { "`"$_`"" }) -join ", ")]

build:
  flags: [$(($Config.BuildFlags | ForEach-Object { "`"$_`"" }) -join ", ")]

environment:
  GOMAXPROCS: "$([Environment]::ProcessorCount)"
  GOMEMLIMIT: "8GiB"
  GOGC: "75"
"@
    
    Set-Content -Path $FilePath -Value $yamlContent
    Write-Host "💾 Configuration saved to: $FilePath" -ForegroundColor Green
}

# Main execution
try {
    Write-Host "🔧 CGO Build Configuration" -ForegroundColor Green
    
    # Get build configuration
    $buildConfig = Get-BuildConfig -ForceRaceDetection:$EnableRaceDetection
    
    # Set Go environment variables
    Set-GoEnvironment -Config $buildConfig
    
    # Show configuration if requested or if showing detailed info
    if ($ShowConfig -or $VerbosePreference -eq "Continue") {
        Show-ConfigSummary -Config $buildConfig
    }
    
    # Save configuration file if specified
    if ($ConfigFile) {
        Save-ConfigFile -Config $buildConfig -FilePath $ConfigFile
    }
    
    # Export configuration for use by calling scripts
    $global:BuildConfig = $buildConfig
    
    # Return configuration object for use in other scripts
    return $buildConfig
    
} catch {
    Write-Host "❌ Failed to configure build environment: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
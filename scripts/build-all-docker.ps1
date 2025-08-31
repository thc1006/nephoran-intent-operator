# Build all Docker services for Nephoran Intent Operator
# Production-ready PowerShell build script with parallel builds and error handling

[CmdletBinding()]
param(
    [Parameter(Position=0)]
    [ValidateSet("parallel", "sequential", "multiarch")]
    [string]$BuildMode = "parallel",
    
    [string]$Registry = $env:REGISTRY ?? "nephoran",
    [string]$Version = $env:VERSION ?? (git describe --tags --always --dirty 2>$null ?? "dev"),
    [int]$ParallelBuilds = $env:PARALLEL_BUILDS ?? 4,
    [switch]$NoCache,
    [switch]$Push,
    [switch]$Help
)

# Configuration
$BUILD_DATE = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ").ToUniversalTime()
$VCS_REF = git rev-parse HEAD 2>$null ?? "unknown"
$env:DOCKER_BUILDKIT = "1"

# Services to build
$SERVICES = @(
    "intent-ingest",
    "llm-processor", 
    "nephio-bridge",
    "oran-adaptor",
    "conductor",
    "conductor-loop",
    "porch-publisher",
    "a1-sim",
    "e2-kpm-sim",
    "fcaps-sim",
    "o1-ves-sim"
)

# Logging functions
function Write-Info($Message) {
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success($Message) {
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning($Message) {
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error($Message) {
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Show help
if ($Help) {
    @"
Usage: .\build-all-docker.ps1 [BuildMode] [Options]

BuildMode:
  parallel   - Build services in parallel (default)
  sequential - Build services one by one  
  multiarch  - Build multi-architecture images (requires docker buildx)

Options:
  -Registry <string>     - Docker registry prefix (default: nephoran)
  -Version <string>      - Image version tag (default: git describe)
  -ParallelBuilds <int>  - Max concurrent builds (default: 4)
  -NoCache              - Build without cache
  -Push                 - Push images after building
  -Help                 - Show this help

Environment Variables:
  REGISTRY      - Docker registry prefix
  VERSION       - Image version tag
  PARALLEL_BUILDS - Max concurrent builds

Examples:
  .\build-all-docker.ps1                               # Parallel build
  .\build-all-docker.ps1 sequential                   # Sequential build  
  .\build-all-docker.ps1 multiarch -Push             # Multi-arch build and push
  .\build-all-docker.ps1 -Registry myregistry -Version v1.0.0  # Custom registry and version

Prerequisites:
  - Docker
  - Git
  - Optional: trivy (for security scanning)
  - Optional: docker buildx (for multi-arch builds)
"@
    exit 0
}

# Build function for a single service
function Build-Service {
    param(
        [string]$Service,
        [hashtable]$BuildArgs
    )
    
    $dockerfile = "cmd/$Service/Dockerfile"
    $imageTag = "$Registry/${Service}:$Version"
    $latestTag = "$Registry/${Service}:latest"
    
    Write-Info "Building $Service..."
    
    if (-not (Test-Path $dockerfile)) {
        Write-Error "Dockerfile not found: $dockerfile"
        return $false
    }
    
    try {
        $dockerArgs = @(
            "build"
            "--file", $dockerfile
            "--tag", $imageTag
            "--tag", $latestTag
            "--target", "runtime"
        )
        
        # Add build arguments
        foreach ($arg in $BuildArgs.GetEnumerator()) {
            $dockerArgs += "--build-arg", "$($arg.Key)=$($arg.Value)"
        }
        
        # Add optional flags
        if ($NoCache) { $dockerArgs += "--no-cache" }
        
        $dockerArgs += "."
        
        # Execute docker build
        $process = Start-Process -FilePath "docker" -ArgumentList $dockerArgs -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -ne 0) {
            Write-Error "Failed to build $Service (exit code: $($process.ExitCode))"
            return $false
        }
        
        Write-Success "Built $Service -> $imageTag"
        
        # Security: Scan the image if trivy is available
        if (Get-Command trivy -ErrorAction SilentlyContinue) {
            Write-Info "Scanning $Service for vulnerabilities..."
            $scanResult = trivy image --exit-code 0 --severity HIGH,CRITICAL $imageTag 2>$null
            if ($LASTEXITCODE -ne 0) {
                Write-Warning "Security scan found issues in $Service"
            }
        }
        
        # Push if requested
        if ($Push) {
            Write-Info "Pushing $imageTag..."
            docker push $imageTag
            docker push $latestTag
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Pushed $imageTag"
            } else {
                Write-Error "Failed to push $imageTag"
                return $false
            }
        }
        
        return $true
    }
    catch {
        Write-Error "Exception building ${Service}: $($_.Exception.Message)"
        return $false
    }
}

# Build all services in parallel
function Build-AllParallel {
    $buildArgs = @{
        VERSION = $Version
        BUILD_DATE = $BUILD_DATE
        VCS_REF = $VCS_REF
        CGO_ENABLED = "0"
    }
    
    Write-Info "Starting parallel build of $($SERVICES.Count) services (max $ParallelBuilds concurrent)..."
    
    $jobs = @()
    $failedServices = @()
    
    foreach ($service in $SERVICES) {
        # Wait if we've reached max parallel builds
        while ($jobs.Count -ge $ParallelBuilds) {
            $completedJobs = $jobs | Where-Object { $_.State -ne "Running" }
            foreach ($job in $completedJobs) {
                $result = Receive-Job -Job $job
                if (-not $result) {
                    $failedServices += $job.Name
                }
                Remove-Job -Job $job
                $jobs = $jobs | Where-Object { $_.Id -ne $job.Id }
            }
            if ($jobs.Count -ge $ParallelBuilds) {
                Start-Sleep -Seconds 1
            }
        }
        
        # Start build job
        $job = Start-Job -Name $service -ScriptBlock {
            param($Service, $BuildArgs, $Registry, $Version, $NoCache, $Push)
            
            # Re-import functions in job context
            function Write-Info($Message) { Write-Host "[INFO] $Message" -ForegroundColor Blue }
            function Write-Success($Message) { Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
            function Write-Warning($Message) { Write-Host "[WARN] $Message" -ForegroundColor Yellow }
            function Write-Error($Message) { Write-Host "[ERROR] $Message" -ForegroundColor Red }
            
            $dockerfile = "cmd/$Service/Dockerfile"
            $imageTag = "$Registry/${Service}:$Version"
            $latestTag = "$Registry/${Service}:latest"
            
            Write-Info "Building $Service..."
            
            if (-not (Test-Path $dockerfile)) {
                Write-Error "Dockerfile not found: $dockerfile"
                return $false
            }
            
            try {
                $dockerArgs = @(
                    "build"
                    "--file", $dockerfile
                    "--tag", $imageTag  
                    "--tag", $latestTag
                    "--target", "runtime"
                )
                
                foreach ($arg in $BuildArgs.GetEnumerator()) {
                    $dockerArgs += "--build-arg", "$($arg.Key)=$($arg.Value)"
                }
                
                if ($NoCache) { $dockerArgs += "--no-cache" }
                $dockerArgs += "."
                
                $process = Start-Process -FilePath "docker" -ArgumentList $dockerArgs -Wait -PassThru -NoNewWindow
                
                if ($process.ExitCode -ne 0) {
                    Write-Error "Failed to build $Service"
                    return $false
                }
                
                Write-Success "Built $Service -> $imageTag"
                
                if ($Push) {
                    docker push $imageTag
                    docker push $latestTag
                    if ($LASTEXITCODE -eq 0) {
                        Write-Success "Pushed $imageTag"
                    } else {
                        Write-Error "Failed to push $imageTag"
                        return $false
                    }
                }
                
                return $true
            }
            catch {
                Write-Error "Exception building ${Service}: $($_.Exception.Message)"
                return $false
            }
        } -ArgumentList $service, $buildArgs, $Registry, $Version, $NoCache, $Push
        
        $jobs += $job
    }
    
    # Wait for remaining jobs
    foreach ($job in $jobs) {
        $result = Receive-Job -Job $job -Wait
        if (-not $result) {
            $failedServices += $job.Name
        }
        Remove-Job -Job $job
    }
    
    if ($failedServices.Count -gt 0) {
        Write-Error "Failed to build services: $($failedServices -join ', ')"
        return $false
    }
    
    Write-Success "All services built successfully!"
    return $true
}

# Build all services sequentially
function Build-AllSequential {
    $buildArgs = @{
        VERSION = $Version
        BUILD_DATE = $BUILD_DATE
        VCS_REF = $VCS_REF
        CGO_ENABLED = "0"
    }
    
    Write-Info "Starting sequential build of $($SERVICES.Count) services..."
    
    $failedServices = @()
    
    foreach ($service in $SERVICES) {
        if (-not (Build-Service -Service $service -BuildArgs $buildArgs)) {
            $failedServices += $service
        }
    }
    
    if ($failedServices.Count -gt 0) {
        Write-Error "Failed to build services: $($failedServices -join ', ')"
        return $false
    }
    
    Write-Success "All services built successfully!"
    return $true
}

# Multi-architecture build function
function Build-MultiArch {
    $platforms = "linux/amd64,linux/arm64"
    Write-Info "Building multi-architecture images for platforms: $platforms"
    
    # Create/use buildx builder
    $builderExists = docker buildx inspect nephoran-builder 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Info "Creating buildx builder..."
        docker buildx create --name nephoran-builder --use
    } else {
        docker buildx use nephoran-builder
    }
    
    $failedServices = @()
    
    foreach ($service in $SERVICES) {
        $dockerfile = "cmd/$service/Dockerfile"
        $imageTag = "$Registry/${service}:$Version"
        
        if (-not (Test-Path $dockerfile)) {
            Write-Error "Dockerfile not found: $dockerfile"
            $failedServices += $service
            continue
        }
        
        Write-Info "Building $service for multiple architectures..."
        
        $dockerArgs = @(
            "buildx", "build"
            "--platform", $platforms
            "--file", $dockerfile
            "--tag", $imageTag
            "--build-arg", "VERSION=$Version"
            "--build-arg", "BUILD_DATE=$BUILD_DATE"
            "--build-arg", "VCS_REF=$VCS_REF"
            "--build-arg", "CGO_ENABLED=0"
            "--target", "runtime"
        )
        
        if ($Push) { $dockerArgs += "--push" }
        if ($NoCache) { $dockerArgs += "--no-cache" }
        
        $dockerArgs += "."
        
        $process = Start-Process -FilePath "docker" -ArgumentList $dockerArgs -Wait -PassThru -NoNewWindow
        
        if ($process.ExitCode -ne 0) {
            Write-Error "Failed to build $service for multiple architectures"
            $failedServices += $service
            continue
        }
        
        Write-Success "Built $service for multiple architectures"
    }
    
    if ($failedServices.Count -gt 0) {
        Write-Error "Failed to build services: $($failedServices -join ', ')"
        return $false
    }
    
    Write-Success "All services built successfully!"
    return $true
}

# Cleanup function
function Invoke-Cleanup {
    Write-Info "Cleaning up build artifacts..."
    docker system prune -f --filter "label=stage=builder" 2>$null | Out-Null
}

# Main execution
try {
    # Cleanup on exit
    trap { Invoke-Cleanup }
    
    Write-Info "=== Nephoran Docker Build Script ==="
    Write-Info "Registry: $Registry"
    Write-Info "Version: $Version"
    Write-Info "Build Date: $BUILD_DATE"
    Write-Info "VCS Ref: $VCS_REF"
    Write-Info "Services: $($SERVICES -join ', ')"
    Write-Info "Build Mode: $BuildMode"
    
    # Check prerequisites
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Error "Docker is not installed or not in PATH"
        exit 1
    }
    
    $dockerInfo = docker info 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Docker daemon is not running"
        exit 1
    }
    
    # Execute based on build mode
    $success = switch ($BuildMode) {
        "sequential" { Build-AllSequential }
        "multiarch" { Build-MultiArch }
        "parallel" { Build-AllParallel }
        default { Build-AllParallel }
    }
    
    if ($success) {
        Write-Success "=== Build completed successfully! ==="
        Write-Info "Images built with tag: $Version"
        Write-Info "To run all services: docker-compose -f docker-compose.services.yml up -d"
    } else {
        Write-Error "=== Build failed! ==="
        exit 1
    }
}
catch {
    Write-Error "Unexpected error: $($_.Exception.Message)"
    exit 1
}
finally {
    Invoke-Cleanup
}
# Docker Build Performance Benchmark Script
# Compares old vs new Dockerfile performance

param(
    [string]$Service = "llm-processor",
    [string]$Platform = "linux/amd64",
    [switch]$CleanCache,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Docker Build Performance Benchmark" -ForegroundColor Cyan
Write-Host "Service: $Service | Platform: $Platform" -ForegroundColor Yellow

# Functions
function Measure-DockerBuild {
    param(
        [string]$DockerFile,
        [string]$Tag,
        [string]$Description
    )
    
    Write-Host "`n📊 Testing: $Description" -ForegroundColor Green
    Write-Host "Dockerfile: $DockerFile" -ForegroundColor Gray
    
    $startTime = Get-Date
    
    try {
        $buildCmd = @"
docker buildx build 
--platform $Platform 
--build-arg SERVICE=$Service 
--build-arg VERSION=benchmark-$(Get-Date -Format "yyyyMMdd-HHmmss")
--build-arg BUILD_DATE=$(Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") 
--build-arg VCS_REF=$(git rev-parse --short HEAD) 
-f $DockerFile 
-t $Tag 
--load 
.
"@
        
        if ($Verbose) {
            Write-Host "Command: $buildCmd" -ForegroundColor Gray
        }
        
        Invoke-Expression $buildCmd
        
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        # Get image size
        $imageInfo = docker images $Tag --format "{{.Size}}" | Select-Object -First 1
        
        Write-Host "✅ Build completed successfully" -ForegroundColor Green
        Write-Host "Duration: $($duration.TotalMinutes.ToString("F2")) minutes" -ForegroundColor Cyan
        Write-Host "Image size: $imageInfo" -ForegroundColor Cyan
        
        return @{
            Success = $true
            Duration = $duration
            ImageSize = $imageInfo
            Tag = $Tag
        }
    }
    catch {
        $endTime = Get-Date
        $duration = $endTime - $startTime
        
        Write-Host "❌ Build failed after $($duration.TotalMinutes.ToString("F2")) minutes" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        
        return @{
            Success = $false
            Duration = $duration
            Error = $_.Exception.Message
            Tag = $Tag
        }
    }
}

function Clear-BuildCache {
    Write-Host "🧹 Clearing Docker build cache..." -ForegroundColor Yellow
    docker buildx prune -f
    docker system prune -f
    Write-Host "Cache cleared" -ForegroundColor Green
}

function Show-Results {
    param($OldResult, $NewResult)
    
    Write-Host "`n📈 PERFORMANCE COMPARISON" -ForegroundColor Magenta
    Write-Host "=" * 50 -ForegroundColor Gray
    
    if ($OldResult.Success -and $NewResult.Success) {
        $oldMinutes = $OldResult.Duration.TotalMinutes
        $newMinutes = $NewResult.Duration.TotalMinutes
        $improvement = (($oldMinutes - $newMinutes) / $oldMinutes) * 100
        
        Write-Host "Old Dockerfile (multiarch):" -ForegroundColor Yellow
        Write-Host "  Duration: $($oldMinutes.ToString("F2")) minutes" 
        Write-Host "  Image size: $($OldResult.ImageSize)"
        
        Write-Host "`nNew Dockerfile (fast-2025):" -ForegroundColor Green
        Write-Host "  Duration: $($newMinutes.ToString("F2")) minutes"
        Write-Host "  Image size: $($NewResult.ImageSize)"
        
        Write-Host "`nIMPROVEMENT:" -ForegroundColor Cyan
        Write-Host "  Time saved: $($improvement.ToString("F1"))%" -ForegroundColor Green
        Write-Host "  Absolute time: $((($oldMinutes - $newMinutes) * 60).ToString("F0")) seconds"
        
        if ($improvement -gt 50) {
            Write-Host "🎉 EXCELLENT: >50% improvement!" -ForegroundColor Green
        } elseif ($improvement -gt 25) {
            Write-Host "👍 GOOD: >25% improvement" -ForegroundColor Yellow
        } else {
            Write-Host "⚠️  MODERATE: <25% improvement" -ForegroundColor Red
        }
    }
    else {
        if (!$OldResult.Success) {
            Write-Host "❌ Old build failed: $($OldResult.Error)" -ForegroundColor Red
        }
        if (!$NewResult.Success) {
            Write-Host "❌ New build failed: $($NewResult.Error)" -ForegroundColor Red
        }
    }
}

function Test-BuildCache {
    param($DockerFile, $Tag)
    
    Write-Host "`n🔄 Testing cache efficiency..." -ForegroundColor Yellow
    
    # First build (cold)
    Write-Host "Cold build (no cache):"
    $coldResult = Measure-DockerBuild -DockerFile $DockerFile -Tag "$Tag-cold" -Description "Cold build"
    
    # Second build (should hit cache)
    Write-Host "`nWarm build (with cache):"
    $warmResult = Measure-DockerBuild -DockerFile $DockerFile -Tag "$Tag-warm" -Description "Warm build"
    
    if ($coldResult.Success -and $warmResult.Success) {
        $cacheEfficiency = (1 - ($warmResult.Duration.TotalSeconds / $coldResult.Duration.TotalSeconds)) * 100
        Write-Host "Cache efficiency: $($cacheEfficiency.ToString("F1"))%" -ForegroundColor Cyan
        
        if ($cacheEfficiency -gt 80) {
            Write-Host "🚀 EXCELLENT cache efficiency!" -ForegroundColor Green
        }
    }
}

# Main execution
try {
    # Validate prerequisites
    if (!(Test-Path "Dockerfile.multiarch")) {
        throw "Dockerfile.multiarch not found"
    }
    
    if (!(Test-Path "Dockerfile.fast-2025")) {
        throw "Dockerfile.fast-2025 not found"
    }
    
    # Clear cache if requested
    if ($CleanCache) {
        Clear-BuildCache
    }
    
    # Test old Dockerfile
    Write-Host "`n🏃‍♂️ Phase 1: Testing current Dockerfile.multiarch"
    $oldResult = Measure-DockerBuild -DockerFile "Dockerfile.multiarch" -Tag "nephoran/$Service-old" -Description "Current multiarch build"
    
    # Test new optimized Dockerfile
    Write-Host "`n🚀 Phase 2: Testing optimized Dockerfile.fast-2025"
    $newResult = Measure-DockerBuild -DockerFile "Dockerfile.fast-2025" -Tag "nephoran/$Service-new" -Description "Optimized fast build"
    
    # Show comparison
    Show-Results -OldResult $oldResult -NewResult $newResult
    
    # Test cache efficiency on new Dockerfile
    if ($newResult.Success) {
        Test-BuildCache -DockerFile "Dockerfile.fast-2025" -Tag "nephoran/$Service-cache"
    }
    
    # Cleanup test images
    Write-Host "`n🧹 Cleaning up test images..."
    docker rmi -f "nephoran/$Service-old" "nephoran/$Service-new" "nephoran/$Service-cache-cold" "nephoran/$Service-cache-warm" 2>$null
    
    Write-Host "`n✅ Benchmark completed successfully!" -ForegroundColor Green
    
    if ($newResult.Success -and $oldResult.Success) {
        $improvement = (($oldResult.Duration.TotalMinutes - $newResult.Duration.TotalMinutes) / $oldResult.Duration.TotalMinutes) * 100
        if ($improvement -gt 40) {
            Write-Host "🎯 RECOMMENDATION: Deploy the optimized Dockerfile immediately!" -ForegroundColor Green
        }
    }
}
catch {
    Write-Host "❌ Benchmark failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host "`n📝 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Review the performance improvements above"
Write-Host "2. If satisfied, update your CI/CD pipeline"
Write-Host "3. Replace Dockerfile.multiarch with Dockerfile.fast-2025"
Write-Host "4. Update GitHub Actions to use the new workflow"
Write-Host "5. Monitor build times in production"
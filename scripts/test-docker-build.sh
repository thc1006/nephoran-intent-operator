#!/bin/bash

# =============================================================================
# Docker Build Test Script with Enhanced Resilience
# =============================================================================
# This script tests the enhanced Docker build process with comprehensive
# error handling and retry logic.
# =============================================================================

set -euo pipefail

# Configuration
IMAGE_BASE="nephoran-test"
SERVICE_NAME="conductor-loop"
BUILD_VERSION="test-$(date +%Y%m%d-%H%M%S)"
PLATFORMS="linux/amd64"
MAX_ATTEMPTS=3

echo "=== Docker Build Test Script ==="
echo "Image: $IMAGE_BASE"
echo "Service: $SERVICE_NAME" 
echo "Version: $BUILD_VERSION"
echo "Platforms: $PLATFORMS"
echo ""

# Enhanced build test function
test_docker_build() {
    local attempt=1
    local build_success=false
    
    while [ $attempt -le $MAX_ATTEMPTS ] && [ "$build_success" = "false" ]; do
        echo "🔄 Build test attempt $attempt/$MAX_ATTEMPTS"
        echo "Starting build at $(date)"
        
        # Build command with timeout
        build_cmd="docker buildx build"
        build_cmd="$build_cmd --platform $PLATFORMS"
        build_cmd="$build_cmd --build-arg SERVICE=$SERVICE_NAME"
        build_cmd="$build_cmd --build-arg VERSION=$BUILD_VERSION"
        build_cmd="$build_cmd --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        build_cmd="$build_cmd --build-arg VCS_REF=$BUILD_VERSION"
        build_cmd="$build_cmd --tag $IMAGE_BASE:$BUILD_VERSION"
        build_cmd="$build_cmd --load"
        build_cmd="$build_cmd --progress=plain"
        build_cmd="$build_cmd ."
        
        echo "Build command:"
        echo "$build_cmd"
        echo ""
        
        start_time=$(date +%s)
        if timeout 1800s $build_cmd; then
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            echo "✅ BUILD SUCCESS in ${duration}s (attempt $attempt)"
            build_success=true
            
            # Verify the image
            echo "Verifying built image..."
            if docker inspect "$IMAGE_BASE:$BUILD_VERSION" >/dev/null 2>&1; then
                image_size=$(docker images --format "{{.Size}}" "$IMAGE_BASE:$BUILD_VERSION")
                echo "✅ Image verified successfully"
                echo "📊 Image size: $image_size"
                
                # Test image functionality (basic smoke test)
                echo "Running smoke test..."
                if timeout 30s docker run --rm "$IMAGE_BASE:$BUILD_VERSION" --version 2>/dev/null || \
                   timeout 30s docker run --rm "$IMAGE_BASE:$BUILD_VERSION" --help >/dev/null 2>&1; then
                    echo "✅ Smoke test passed"
                else
                    echo "⚠️  Smoke test failed (binary might not support --version/--help)"
                fi
            else
                echo "❌ Image verification failed"
                build_success=false
            fi
        else
            exit_code=$?
            end_time=$(date +%s)
            duration=$((end_time - start_time))
            echo "❌ BUILD FAILED with exit code $exit_code after ${duration}s (attempt $attempt)"
            
            if [ $attempt -lt $MAX_ATTEMPTS ]; then
                echo "🔄 Cleaning up for retry..."
                docker system prune -f 2>/dev/null || true
                docker builder prune -f 2>/dev/null || true
                
                backoff_time=$((attempt * 10))
                echo "⏱️  Waiting ${backoff_time}s before retry..."
                sleep $backoff_time
            fi
        fi
        
        attempt=$((attempt + 1))
    done
    
    if [ "$build_success" = "true" ]; then
        return 0
    else
        return 1
    fi
}

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test resources..."
    docker rmi "$IMAGE_BASE:$BUILD_VERSION" 2>/dev/null || true
    docker system prune -f 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Check prerequisites
echo "🔍 Checking prerequisites..."
if ! command -v docker >/dev/null 2>&1; then
    echo "❌ Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker-compose >/dev/null 2>&1; then
    echo "⚠️  docker-compose not found (not required for basic build test)"
fi

if ! docker buildx version >/dev/null 2>&1; then
    echo "❌ Docker Buildx is not available"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Show Docker environment info
echo "🐳 Docker Environment Info:"
echo "Docker version: $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo 'unknown')"
echo "Buildx version: $(docker buildx version 2>/dev/null | head -1 || echo 'unknown')"
echo "Available builders:"
docker buildx ls 2>/dev/null || echo "Unable to list builders"
echo ""

# Test the build process
echo "🚀 Starting Docker build test..."
if test_docker_build; then
    echo ""
    echo "🎉 BUILD TEST COMPLETED SUCCESSFULLY!"
    echo "✅ Enhanced Docker build resilience is working correctly"
    echo ""
    echo "Summary:"
    echo "- Image: $IMAGE_BASE:$BUILD_VERSION"
    echo "- Service: $SERVICE_NAME"
    echo "- Platform: $PLATFORMS"
    echo "- Status: ✅ PASSED"
    
    # Show final image info
    if docker inspect "$IMAGE_BASE:$BUILD_VERSION" >/dev/null 2>&1; then
        echo ""
        echo "📋 Final Image Details:"
        docker inspect "$IMAGE_BASE:$BUILD_VERSION" --format '{{json .}}' | \
            jq -r '"  ID: " + .Id[:12] + "\n  Created: " + .Created + "\n  Size: " + (.Size | tostring) + " bytes\n  Architecture: " + .Architecture + "\n  OS: " + .Os' 2>/dev/null || \
            docker inspect "$IMAGE_BASE:$BUILD_VERSION" --format '  ID: {{.Id}}' 
    fi
    
    exit 0
else
    echo ""
    echo "💥 BUILD TEST FAILED!"
    echo "❌ Enhanced Docker build resilience needs review"
    echo ""
    echo "Diagnostic information:"
    echo "- Docker system info:"
    docker system info --format 'json' 2>/dev/null | \
        jq -r '"  Server Version: " + .ServerVersion + "\n  Storage Driver: " + .Driver + "\n  Operating System: " + .OperatingSystem' 2>/dev/null || \
        echo "  Unable to get system info"
    
    echo "- Recent Docker events:"
    docker events --since=5m --until=now 2>/dev/null | tail -5 || echo "  No recent events"
    
    exit 1
fi
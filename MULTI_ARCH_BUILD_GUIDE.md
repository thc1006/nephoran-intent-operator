# Multi-Architecture Build Guide

This document outlines the multi-architecture build support implemented for the Nephoran Intent Operator project.

## Overview

All Dockerfiles in the project have been updated to support multi-architecture builds for both `linux/amd64` and `linux/arm64` platforms using Docker Buildx.

## Updated Dockerfiles

### 1. Main Service Dockerfile (`/Dockerfile`)
- **Location**: Root directory
- **Services**: llm-processor, nephio-bridge, oran-adaptor (multi-service)
- **Enhancements**:
  - Added `BUILDPLATFORM` and `TARGETPLATFORM` arguments
  - Cross-compilation support with `TARGETOS` and `TARGETARCH`
  - Platform-specific optimizations (UPX compression for amd64)
  - Dynamic base image selection

### 2. LLM Processor (`/cmd/llm-processor/Dockerfile`)
- **Service**: AI-powered network intent processing
- **Enhancements**:
  - Multi-arch Go cross-compilation
  - Platform-aware optimization stage
  - Architecture-specific binary compression

### 3. Nephio Bridge (`/cmd/nephio-bridge/Dockerfile`)
- **Service**: GitOps integration bridge
- **Enhancements**:
  - Cross-platform Go builds
  - Multi-arch distroless runtime
  - Platform-specific labels and metadata

### 4. O-RAN Adaptor (`/cmd/oran-adaptor/Dockerfile`)
- **Service**: O-RAN interface management
- **Enhancements**:
  - Multi-architecture binary builds
  - Platform-aware optimizations
  - Architecture-specific runtime configuration

### 5. RAG API (`/pkg/rag/Dockerfile`)
- **Service**: Python-based Retrieval-Augmented Generation API
- **Enhancements**:
  - Multi-arch Python package compilation
  - Platform-specific build tools (gcc cross-compilers)
  - Architecture-aware pip caching
  - Cross-platform runtime optimization

## Key Features Implemented

### 1. Build Platform Arguments
```dockerfile
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
```

### 2. Cross-Compilation Support
```dockerfile
# Go services
CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}

# Python services (for native extensions)
if [ "$TARGETARCH" = "arm64" ]; then
    export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++
fi
```

### 3. Platform-Specific Optimizations
```dockerfile
# UPX compression only for amd64 (due to arm64 compatibility)
if [ "$TARGETARCH" = "amd64" ] && command -v upx >/dev/null 2>&1; then
    upx --best --lzma /tmp/binary
fi
```

### 4. Dynamic Base Image Selection
```dockerfile
# Use platform-agnostic distroless images
FROM gcr.io/distroless/static:nonroot AS runtime-base
```

### 5. Enhanced Labels and Metadata
```dockerfile
LABEL build.architecture="${TARGETARCH:-amd64}" \
      build.platform="${TARGETPLATFORM}" \
      build.multi-arch="true"
```

## Build Cache Optimizations

### 1. Registry-Based Caching
The Makefile includes registry-based caching for multi-arch builds:

```makefile
CACHE_FROM="--cache-from type=registry,ref=$(CACHE_REGISTRY)/$(IMG):cache"
CACHE_TO="--cache-to type=registry,ref=$(CACHE_REGISTRY)/$(IMG):cache,mode=max"
```

### 2. Python Package Caching
```dockerfile
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir --user -r requirements-rag.txt
```

## Security Enhancements

### 1. Minimal Attack Surface
- Distroless base images for Go services
- Minimal Python slim base for RAG API
- Non-root user execution across all services

### 2. Binary Hardening
- Static linking for Go binaries
- Symbol stripping with `binutils`
- Optional UPX compression for space optimization

### 3. Dependency Verification
```dockerfile
RUN go mod download && \
    go mod verify && \
    go mod tidy
```

## Platform-Specific Considerations

### AMD64 (x86_64)
- Full optimization support including UPX compression
- Native compilation performance
- Extensive toolchain availability

### ARM64 (aarch64)
- Cross-compilation from AMD64 build hosts
- Skip UPX compression due to compatibility
- Platform-specific build tools for Python native extensions

## Build Commands

### Using Makefile Targets
```bash
# Setup buildx environment
make setup-buildx

# Build all services for multiple architectures
make build-multiarch

# Complete build with SBOM and security scanning
make build-complete
```

### Direct Docker Buildx Commands
```bash
# Build specific service for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v2.0.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse --short HEAD) \
  --tag registry/service:latest \
  --push \
  -f cmd/service/Dockerfile .
```

## Performance Optimizations

### 1. Parallel Builds
- Multi-stage builds for efficient caching
- Parallel buildx execution for multiple platforms
- Optimized layer ordering for cache efficiency

### 2. Binary Size Optimization
- Go build flags: `-ldflags="-w -s"`
- Strip unnecessary symbols
- UPX compression for amd64 binaries

### 3. Runtime Efficiency
- Distroless images for minimal overhead
- Optimized environment variables
- Platform-specific runtime configurations

## Verification and Testing

### 1. Multi-Platform Verification
```bash
# Verify images support multiple platforms
docker buildx imagetools inspect registry/service:latest

# Test platform-specific functionality
make build-verify
```

### 2. Security Scanning
```bash
# Run vulnerability scans on multi-arch images
make build-vulnerability-scan
```

## Integration with CI/CD

The enhanced build system integrates seamlessly with CI/CD pipelines:

```bash
# CI/CD build pipeline
make ci-setup      # Setup build environment
make ci-build      # Build with optimizations
make ci-push       # Push multi-arch images
make ci-security   # Security scanning
make ci-sign       # Image signing (optional)
```

## Troubleshooting

### Common Issues

1. **ARM64 Build Failures**
   - Ensure buildx driver supports multi-platform
   - Check cross-compilation toolchain availability

2. **Cache Miss Issues**
   - Verify registry credentials for cache access
   - Check cache key consistency across builds

3. **UPX Compression Errors**
   - UPX only applies to amd64 builds
   - ARM64 builds skip compression automatically

### Debug Build Information
Each Dockerfile includes debug output showing:
- Build platform information
- Target architecture details
- Cross-compilation settings

## Best Practices

1. **Always use buildx for multi-arch builds**
2. **Implement platform-specific optimizations carefully**
3. **Test both architectures in CI/CD**
4. **Use registry caching for faster builds**
5. **Monitor image sizes across platforms**
6. **Verify functionality on target architectures**

## Future Enhancements

1. **Additional Architecture Support**
   - Potential support for linux/s390x, linux/ppc64le
   - Platform-specific optimization strategies

2. **Advanced Caching**
   - Cross-platform cache sharing
   - Intelligent cache invalidation

3. **Security Hardening**
   - Platform-specific security policies
   - Architecture-aware vulnerability scanning

This multi-architecture build system provides a robust foundation for deploying the Nephoran Intent Operator across diverse Kubernetes environments while maintaining security, performance, and reliability standards.
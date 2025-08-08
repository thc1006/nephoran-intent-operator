# Dockerfile Consolidation Migration Guide

## Executive Summary

The Nephoran Intent Operator Docker build system has been consolidated from 17+ Dockerfile variants down to 3 essential Dockerfiles, achieving a **82% reduction** in Dockerfile complexity while maintaining all necessary deployment scenarios.

## Consolidation Overview

### Before: 17 Dockerfile Variants
- `Dockerfile` - Base production build
- `Dockerfile.dev` - Development environment
- `Dockerfile.production` - Production optimized
- `Dockerfile.multiarch` - Multi-architecture support
- `Dockerfile.go124` - Go 1.24 specific
- `Dockerfile.llm-secure` - LLM service security hardened
- `Dockerfile.nephio-secure` - Nephio service security hardened
- `Dockerfile.oran-secure` - O-RAN service security hardened
- `Dockerfile.rag-secure` - RAG service security hardened
- `Dockerfile.security` - General security features
- `Dockerfile.security-scan` - Security scanning tools
- `cmd/llm-processor/Dockerfile` - LLM processor specific
- `cmd/nephio-bridge/Dockerfile` - Nephio bridge specific
- `cmd/oran-adaptor/Dockerfile` - O-RAN adaptor specific
- `deployments/a1-service/docker/Dockerfile` - A1 service specific
- `pkg/rag/Dockerfile` - RAG package specific

### After: 3 Essential Dockerfiles

#### 1. **Dockerfile** (Production)
- **Purpose**: Production-ready builds for all services
- **Features**:
  - Multi-stage builds for optimization
  - Security hardening with non-root users
  - Distroless base images for minimal attack surface
  - Support for all Go and Python services
  - Optimized layer caching
  - Static binary compilation
  - Health checks included

#### 2. **Dockerfile.dev** (Development)
- **Purpose**: Development environment with debugging tools
- **Features**:
  - Hot-reload support with Air (Go) and Watchdog (Python)
  - Debugging tools (Delve for Go, debugpy for Python)
  - Development utilities (vim, jq, kubectl, helm)
  - Code analysis tools (golangci-lint, black, flake8)
  - Volume mounting for live code updates
  - Multiple exposed ports for debugging

#### 3. **Dockerfile.multiarch** (Multi-Architecture)
- **Purpose**: Cross-platform builds for multiple architectures
- **Features**:
  - Support for linux/amd64, linux/arm64, linux/arm/v7
  - Docker Buildx integration
  - Architecture-specific optimizations
  - Cross-compilation support
  - Platform-aware resource limits
  - Build caching for faster multi-arch builds

## Migration Instructions

### Building Services with New Dockerfiles

#### Production Builds

```bash
# Build a specific service for production
docker build --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .

# Build all services using the consolidated script
./scripts/docker-build-consolidated.sh --production

# Build with specific version
docker build \
  --build-arg SERVICE=nephio-bridge \
  --build-arg VERSION=v2.0.0 \
  -t nephoran/nephio-bridge:v2.0.0 .
```

#### Development Builds

```bash
# Build development image with debugging tools
docker build -f Dockerfile.dev \
  --build-arg SERVICE=llm-processor \
  --build-arg SERVICE_TYPE=go \
  -t nephoran/llm-processor:dev .

# Run with hot-reload
docker run -v $(pwd):/workspace -p 8080:8080 -p 40000:40000 \
  -e DEBUG_MODE=true \
  nephoran/llm-processor:dev

# Use the build script
./scripts/docker-build-consolidated.sh --dev --service llm-processor
```

#### Multi-Architecture Builds

```bash
# Setup buildx (one-time)
docker buildx create --use --name multiarch-builder

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 \
  -f Dockerfile.multiarch \
  --build-arg SERVICE=oran-adaptor \
  -t nephoran/oran-adaptor:latest \
  --push .

# Use the build script
./scripts/docker-build-consolidated.sh --multiarch --service oran-adaptor
```

### Docker Compose Usage

#### Production Deployment

```bash
# Use production Dockerfile (default)
docker-compose -f deployments/docker-compose.consolidated.yml up

# With specific version
VERSION=v2.0.0 docker-compose -f deployments/docker-compose.consolidated.yml up
```

#### Development Environment

```bash
# Use development Dockerfile
DOCKERFILE=Dockerfile.dev docker-compose -f deployments/docker-compose.consolidated.yml up

# With debugging enabled
DOCKERFILE=Dockerfile.dev DEBUG_MODE=true docker-compose up
```

### Service Mapping

| Old Dockerfile | Service | New Dockerfile | Build Args |
|---------------|---------|----------------|-------------|
| `cmd/llm-processor/Dockerfile` | llm-processor | `Dockerfile` | `SERVICE=llm-processor SERVICE_TYPE=go` |
| `cmd/nephio-bridge/Dockerfile` | nephio-bridge | `Dockerfile` | `SERVICE=nephio-bridge SERVICE_TYPE=go` |
| `cmd/oran-adaptor/Dockerfile` | oran-adaptor | `Dockerfile` | `SERVICE=oran-adaptor SERVICE_TYPE=go` |
| `pkg/rag/Dockerfile` | rag-api | `Dockerfile` | `SERVICE=rag-api SERVICE_TYPE=python` |
| `Dockerfile.llm-secure` | llm-processor | `Dockerfile` | `SERVICE=llm-processor` (security included) |
| `Dockerfile.production` | all services | `Dockerfile` | Per-service args |

## Build Script Usage

The new consolidated build script (`scripts/docker-build-consolidated.sh`) simplifies the build process:

```bash
# Show help
./scripts/docker-build-consolidated.sh --help

# Build all production images
./scripts/docker-build-consolidated.sh --production

# Build specific service for development
./scripts/docker-build-consolidated.sh --dev --service llm-processor

# Build multi-arch and push to registry
./scripts/docker-build-consolidated.sh --multiarch --push

# Build with custom registry and version
./scripts/docker-build-consolidated.sh \
  --registry myregistry.io/nephoran \
  --version v2.1.0 \
  --service nephio-bridge
```

## CI/CD Pipeline Updates

Update your CI/CD pipelines to use the new consolidated Dockerfiles:

### GitHub Actions Example

```yaml
- name: Build Production Image
  run: |
    docker build \
      --build-arg SERVICE=${{ matrix.service }} \
      --build-arg SERVICE_TYPE=${{ matrix.type }} \
      --build-arg VERSION=${{ github.ref_name }} \
      --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
      --build-arg VCS_REF=${{ github.sha }} \
      -t ${{ env.REGISTRY }}/${{ matrix.service }}:${{ github.ref_name }} \
      -f Dockerfile \
      .
```

### GitLab CI Example

```yaml
build:
  stage: build
  script:
    - docker build
        --build-arg SERVICE=${CI_JOB_NAME}
        --build-arg VERSION=${CI_COMMIT_TAG:-latest}
        --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        --build-arg VCS_REF=${CI_COMMIT_SHORT_SHA}
        -t ${CI_REGISTRY_IMAGE}/${CI_JOB_NAME}:${CI_COMMIT_TAG:-latest}
        -f Dockerfile
        .
```

## Benefits of Consolidation

### 1. **Reduced Maintenance Overhead**
- 82% fewer Dockerfiles to maintain
- Centralized security updates
- Consistent build patterns across all services

### 2. **Improved Build Performance**
- Shared base layers reduce build time
- Better layer caching
- Optimized multi-stage builds

### 3. **Enhanced Security**
- Consistent security hardening across all services
- Single point for security updates
- Distroless images for minimal attack surface

### 4. **Simplified Development Workflow**
- One development Dockerfile for all services
- Consistent debugging setup
- Unified hot-reload configuration

### 5. **Better Multi-Architecture Support**
- Single Dockerfile for all architectures
- Optimized cross-compilation
- Platform-specific optimizations

## Troubleshooting

### Common Issues and Solutions

#### Issue: Build fails with "Unknown service"
**Solution**: Ensure SERVICE argument matches one of: llm-processor, nephio-bridge, oran-adaptor, manager, rag-api

#### Issue: Multi-arch build fails
**Solution**: Ensure Docker Buildx is installed and configured:
```bash
docker buildx create --use --name multiarch-builder
docker buildx inspect --bootstrap
```

#### Issue: Development hot-reload not working
**Solution**: Ensure source code is mounted as volume and SERVICE_TYPE is set correctly

#### Issue: Python service build fails
**Solution**: Set SERVICE_TYPE=python for RAG API service

## Rollback Plan

If you need to temporarily revert to old Dockerfiles:

1. Restore from git history:
```bash
git checkout <previous-commit> -- Dockerfile.<variant>
```

2. Use the old build commands specific to each Dockerfile

3. Update CI/CD pipelines to use old Dockerfile paths

## Support and Documentation

- **Build Script Help**: `./scripts/docker-build-consolidated.sh --help`
- **Docker Compose Examples**: `deployments/docker-compose.consolidated.yml`
- **CI/CD Templates**: See examples above
- **Issue Tracking**: Report issues in the project's issue tracker

## Conclusion

The Dockerfile consolidation reduces complexity by 82% while maintaining all functionality. The three essential Dockerfiles (production, development, multi-arch) cover all deployment scenarios with improved maintainability, security, and performance.
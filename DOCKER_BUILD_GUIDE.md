# Docker Build Guide - Nephoran Intent Operator

This guide provides comprehensive instructions for building, testing, and deploying all Nephoran services using Docker.

## Overview

The Nephoran Intent Operator consists of 11 microservices, each with production-ready Docker configurations:

### Core Services
- **intent-ingest** - Intent ingestion API (Port: 8081)
- **llm-processor** - LLM processing with RAG support (Port: 8083)
- **nephio-bridge** - Nephio Porch integration (Port: 8082)
- **oran-adaptor** - O-RAN interface adaptor (Port: 8084)
- **conductor** - Main orchestration service (Port: 8085)
- **conductor-loop** - File watcher and loop controller
- **porch-publisher** - Package publishing service (Port: 8086)

### Simulation Services
- **a1-sim** - A1 interface simulator (Port: 9001)
- **e2-kpm-sim** - E2 KPM interface simulator (Port: 9002)
- **fcaps-sim** - FCAPS/VES event simulator (Port: 9003)
- **o1-ves-sim** - O1 VES event simulator (Port: 9004)

## Quick Start

### Prerequisites

- Docker Engine 20.10+ or Docker Desktop
- Git
- Optional: Trivy (for security scanning)
- Optional: Docker Buildx (for multi-arch builds)

### Build All Services

```bash
# Unix/Linux/macOS
./scripts/build-all-docker.sh

# Windows PowerShell
.\scripts\build-all-docker.ps1
```

### Run All Services

```bash
docker-compose -f docker-compose.services.yml up -d
```

## Build Options

### Build Modes

1. **Parallel Build (Default)** - Fastest option
   ```bash
   ./scripts/build-all-docker.sh parallel
   ```

2. **Sequential Build** - More reliable on resource-constrained systems
   ```bash
   ./scripts/build-all-docker.sh sequential
   ```

3. **Multi-Architecture Build** - For ARM64 and AMD64 platforms
   ```bash
   ./scripts/build-all-docker.sh multiarch
   ```

### Environment Variables

```bash
export REGISTRY="your-registry"          # Default: nephoran
export VERSION="v1.0.0"                  # Default: git describe
export PARALLEL_BUILDS=8                 # Default: 4
export DOCKER_BUILDKIT=1                 # Enable BuildKit
```

### PowerShell Options

```powershell
# Custom registry and version
.\scripts\build-all-docker.ps1 -Registry myregistry -Version v1.0.0

# Build and push to registry
.\scripts\build-all-docker.ps1 multiarch -Push

# Build without cache
.\scripts\build-all-docker.ps1 -NoCache
```

## Docker Compose Configurations

### Full Stack (docker-compose.services.yml)
- All 11 services
- Infrastructure (Weaviate, Redis)
- Monitoring (Prometheus, Grafana)
- Production-ready networking

### Development (docker-compose.yml)
- Core services only
- Development configurations
- Local volume mounts

## Individual Service Builds

### Build Single Service

```bash
# Build intent-ingest service
docker build -f cmd/intent-ingest/Dockerfile \
  --tag nephoran/intent-ingest:latest \
  --build-arg VERSION=v1.0.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse HEAD) \
  --target runtime \
  .
```

### Development Build

```bash
# Build with development stage
docker build -f cmd/intent-ingest/Dockerfile \
  --tag nephoran/intent-ingest:dev \
  --target dev \
  .
```

## Security Features

### Multi-Stage Builds
- **Builder stage**: Go compilation with security flags
- **Runtime stage**: Distroless base image
- **Dev stage**: Development tools and hot reload

### Security Hardening
- Non-root user (UID 65532)
- Read-only root filesystem
- No privilege escalation
- Minimal attack surface (distroless)
- Static compilation with security flags

### Vulnerability Scanning
```bash
# Automatic scanning with Trivy (if installed)
trivy image nephoran/intent-ingest:latest

# Manual security scan
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy:latest image nephoran/intent-ingest:latest
```

## Performance Optimizations

### Build Performance
- Docker BuildKit enabled
- Multi-stage builds for caching
- Parallel builds (up to 4 concurrent)
- .dockerignore to reduce context

### Runtime Performance
- Static binaries (no dynamic linking)
- Optimized Go build flags
- Resource limits defined
- Health checks configured

## Monitoring and Observability

### Built-in Metrics
- Prometheus metrics on port 9090
- Health checks on port 8080
- Structured JSON logging

### Monitoring Stack
```bash
# Start monitoring stack
docker-compose -f docker-compose.services.yml up -d prometheus grafana

# Access monitoring
open http://localhost:3000  # Grafana
open http://localhost:9090  # Prometheus
```

## Troubleshooting

### Common Issues

1. **Build Fails - Out of Memory**
   ```bash
   # Use sequential builds
   ./scripts/build-all-docker.sh sequential
   ```

2. **Permission Denied on Linux**
   ```bash
   # Make script executable
   chmod +x scripts/build-all-docker.sh
   ```

3. **Docker Daemon Not Running**
   ```bash
   # Start Docker daemon
   sudo systemctl start docker  # Linux
   # Or restart Docker Desktop
   ```

### Debug Build Issues

```bash
# Build with debug output
DOCKER_BUILDKIT=0 docker build --no-cache --progress=plain -f cmd/SERVICE/Dockerfile .

# Check build context size
docker build --dry-run -f cmd/SERVICE/Dockerfile . 2>&1 | grep "build context"

# Inspect failed build
docker build --target builder -f cmd/SERVICE/Dockerfile .
docker run -it LAST_LAYER /bin/sh
```

### Verify Service Health

```bash
# Check container status
docker-compose -f docker-compose.services.yml ps

# Check service health
curl http://localhost:8081/health  # intent-ingest
curl http://localhost:8083/health  # llm-processor

# Check logs
docker-compose -f docker-compose.services.yml logs intent-ingest
```

## CI/CD Integration

### GitHub Actions
```yaml
name: Build Docker Images
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Build images
      run: ./scripts/build-all-docker.sh parallel
    - name: Run security scan
      run: |
        docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
          aquasec/trivy:latest image nephoran/intent-ingest:latest
```

### Registry Push
```bash
# Login to registry
echo $REGISTRY_PASSWORD | docker login -u $REGISTRY_USER --password-stdin

# Build and push
REGISTRY=your-registry VERSION=v1.0.0 ./scripts/build-all-docker.sh multiarch
```

## Best Practices

### Production Deployments
1. Use specific version tags, not `latest`
2. Enable security scanning in CI/CD
3. Use multi-arch builds for compatibility
4. Set resource limits and requests
5. Enable health checks and readiness probes

### Development Workflow
1. Use development Docker targets for hot reload
2. Mount source code as volumes for development
3. Use docker-compose override files for local config
4. Leverage Docker BuildKit for faster builds

### Security Best Practices
1. Regularly update base images
2. Scan images for vulnerabilities
3. Use distroless images in production
4. Enable read-only root filesystem
5. Run containers as non-root user
6. Use specific version tags

## Resource Requirements

### Minimum System Requirements
- CPU: 4 cores
- RAM: 8GB
- Disk: 20GB free space
- Docker: 20.10+

### Production Requirements
- CPU: 8+ cores
- RAM: 16GB+
- Disk: 100GB+ SSD
- Network: 1Gbps+

### Per-Service Resources

| Service | Memory Limit | CPU Limit | Storage |
|---------|-------------|-----------|---------|
| intent-ingest | 512Mi | 0.5 | 1GB |
| llm-processor | 2Gi | 2.0 | 5GB |
| nephio-bridge | 1Gi | 1.0 | 1GB |
| oran-adaptor | 1Gi | 1.0 | 1GB |
| conductor | 512Mi | 0.5 | 1GB |
| conductor-loop | 512Mi | 1.0 | 2GB |
| porch-publisher | 512Mi | 0.5 | 1GB |
| Simulators (each) | 256Mi | 0.25 | 0.5GB |

## Support

For issues with Docker builds:

1. Check the build logs for specific error messages
2. Verify Docker daemon is running and accessible
3. Ensure sufficient system resources
4. Check network connectivity for base image pulls
5. Validate Dockerfile syntax and build context

For production deployments, consider:
- Using a container orchestration platform (Kubernetes)
- Implementing proper logging and monitoring
- Setting up automated backups and disaster recovery
- Configuring proper network security and firewalls
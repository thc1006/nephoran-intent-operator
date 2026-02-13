# Resilient Docker Build System

## Overview

The Nephoran Intent Operator uses an advanced, infrastructure-aware Docker build system designed to handle external service failures, registry outages, and network connectivity issues. This system automatically adapts its build strategy based on the current state of external dependencies.

## Problem Statement

Traditional Docker builds can fail when external services experience outages:

- **GitHub Actions cache service** returns HTML error pages instead of API responses
- **Container registries** (gcr.io, docker.io) may be temporarily unavailable
- **Go proxy services** may experience downtime
- **Base image pulls** can timeout or fail

## Solution Architecture

### 1. Infrastructure Health Monitoring

Before starting any build, the system performs comprehensive health checks:

```bash
# Example health check output
✅ GHA cache service responsive
⚠️ docker.io may be experiencing issues
✅ gcr.io responsive
✅ ghcr.io responsive
✅ 2/3 Go proxies responsive
✅ DNS resolution working
✅ Internet connectivity confirmed

Infrastructure health score: 85/100 (85%)
```

### 2. Intelligent Strategy Selection

Based on infrastructure health, the system selects the optimal build strategy:

| Health Score | Strategy | Dockerfile | Cache Strategy | Timeout |
|-------------|----------|------------|----------------|---------|
| 80-100% | **Optimal** | Standard | GHA + Registry | 40min |
| 60-79% | **Resilient** | Resilient | Local + Fallback | 30min |
| 40-59% | **Conservative** | Resilient | Local Only | 20min |
| 0-39% | **Emergency** | Resilient | No Cache | 15min |

### 3. Multi-Source Dockerfile Strategies

#### Standard Dockerfile (`Dockerfile`)
- Uses primary registries (gcr.io/distroless)
- Relies on external cache services
- Optimal for healthy infrastructure

#### Resilient Dockerfile (`Dockerfile.resilient`)
- Multiple base image sources with fallback chain
- Alternative package repositories
- Multi-proxy Go module downloads
- Enhanced retry logic with exponential backoff

```dockerfile
# Primary distroless base image
FROM gcr.io/distroless/static:${DISTROLESS_VERSION} AS distroless-gcr

# Secondary base image (different registry)
FROM gcr.io/distroless/static-debian12:${DISTROLESS_VERSION} AS distroless-alt

# Alpine fallback for maximum reliability
FROM alpine:${ALPINE_VERSION} AS alpine-fallback
RUN set -ex; \
    # Multiple repository sources with retry logic
    repos=("dl-cdn.alpinelinux.org" "mirror.lzu.edu.cn" "mirrors.aliyun.com"); \
    # ... retry logic implementation
```

### 4. Cache Fallback Strategies

The system implements multiple cache strategies to handle service outages:

#### GHA Full Strategy (Optimal)
```bash
--cache-from=type=local,src=/tmp/cache 
--cache-from=type=gha 
--cache-from=type=registry,ref=$IMAGE_NAME:cache
--cache-to=type=local,dest=/tmp/cache,mode=max 
--cache-to=type=registry,ref=$IMAGE_NAME:cache,mode=max,ignore-error=true
```

#### Local-Only Strategy (Conservative)
```bash
--cache-from=type=local,src=/tmp/cache
--cache-to=type=local,dest=/tmp/cache,mode=max
```

#### No-Cache Strategy (Emergency)
```bash
--no-cache
```

## Usage

### Automatic Usage (CI)

The CI system automatically uses the resilient build system:

```yaml
- name: Pre-build cache and registry health check
  run: |
    # Automatic infrastructure health assessment
    # Sets GHA_CACHE_AVAILABLE and AVAILABLE_REGISTRIES

- name: Build and push container image with smart infrastructure-aware strategy
  run: |
    ./scripts/deploy/smart-docker-build.sh \
      "$SERVICE_NAME" \
      "$IMAGE_NAME" \
      "$BUILD_VERSION" \
      "$PLATFORMS" \
      "$SHOULD_PUSH"
```

### Manual Usage

```bash
# Basic usage with automatic strategy selection
./scripts/deploy/smart-docker-build.sh conductor-loop nephoran/conductor-loop v1.0.0

# Multi-platform build with push
./scripts/deploy/smart-docker-build.sh \
  conductor-loop \
  ghcr.io/myorg/nephoran \
  v1.2.3 \
  "linux/amd64,linux/arm64" \
  true
```

### Direct Docker Usage

For manual builds during outages:

```bash
# Use resilient Dockerfile during outages
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg SERVICE=conductor-loop \
  --cache-from=type=local,src=/tmp/cache \
  --cache-to=type=local,dest=/tmp/cache,mode=max \
  -f Dockerfile.resilient \
  -t nephoran/conductor-loop:latest \
  .
```

## Registry Mirrors and Alternatives

The system includes pre-configured registry mirrors for enhanced reliability:

### Container Registries
- **docker.io** → mirror.gcr.io, dockerhub.azk8s.cn
- **gcr.io** → mirror.gcr.io, gcr.azk8s.cn  
- **quay.io** → quay.azk8s.cn
- **registry.k8s.io** → k8s.gcr.io, registry.aliyuncs.com/google_containers

### Go Proxy Mirrors
- proxy.golang.org (primary)
- goproxy.cn (China mirror)
- goproxy.io (alternative)
- athens.azurefd.net (Microsoft-hosted)

### Package Repository Mirrors
- dl-cdn.alpinelinux.org (primary)
- mirror.lzu.edu.cn (China mirror)
- mirrors.aliyun.com (Alibaba mirror)
- mirrors.tuna.tsinghua.edu.cn (Tsinghua mirror)

## Error Handling and Recovery

### Build Failure Recovery
1. **Automatic retry** with exponential backoff (3 attempts max)
2. **Strategy degradation** (optimal → resilient → conservative → emergency)
3. **Cache cleanup** between attempts
4. **Detailed failure analysis** with infrastructure diagnostics

### Service Outage Detection
The system detects various types of service outages:

```bash
# GHA cache service HTML error pages
"failed to parse error response 400: <h2>Our services aren't available right now</h2>"

# Registry connectivity issues
"dial tcp: lookup gcr.io: no such host"

# Go proxy timeouts
"timeout awaiting response from goproxy.cn"
```

### Fallback Mechanisms
- **Base image fallbacks**: distroless → debian → alpine → ubuntu
- **Package repository rotation** for Alpine/Debian packages
- **Go proxy rotation** with different providers
- **Cache strategy degradation** from remote to local
- **Build timeout adjustment** based on strategy

## Monitoring and Observability

### Build Summary Reports
Each build generates comprehensive summary reports:

```markdown
## Build Results

**Binaries built successfully**

| Architecture | Size | Status |
|-------------|------|--------|
| linux/amd64 | 45.2 MB | Built |
| linux/arm64 | 42.8 MB | Built |

**Container image built successfully**

**Image Details:**
- Digest: `sha256:abc123...`
- Platforms: linux/amd64,linux/arm64
- Registry: `ghcr.io/myorg/nephoran-intent-operator`

**Infrastructure Health During Build:**
- GHA Cache: true
- Available Registries: gcr.io docker.io ghcr.io
```

### Diagnostic Information
On failures, the system provides detailed diagnostic information:

```bash
=== DIAGNOSTIC INFORMATION ===
Docker version: 24.0.7
Buildx version: v0.12.1
Available builders: default
Infrastructure health: 45%
Healthy registries: 1/3
Healthy Go proxies: 2/3
Failed strategy: optimal
Last error: timeout waiting for registry response
```

## Best Practices

### Development
1. **Test with network issues**: Use network simulation tools
2. **Test offline builds**: Verify local cache effectiveness
3. **Monitor build times**: Track strategy performance
4. **Update mirrors regularly**: Keep alternative sources current

### Operations
1. **Monitor infrastructure health**: Set up external service monitoring
2. **Cache pre-warming**: Populate local caches during healthy periods
3. **Strategy tuning**: Adjust thresholds based on service reliability
4. **Incident response**: Use emergency strategy during outages

### Security
1. **Mirror verification**: Ensure mirror integrity
2. **Registry authentication**: Secure alternative registries
3. **Cache isolation**: Separate cache by environment
4. **Base image scanning**: Verify all fallback images

## Configuration

### Environment Variables

```bash
# Build configuration
DOCKER_BUILDKIT=1
BUILDX_NO_DEFAULT_ATTESTATIONS=1
BUILDKIT_PROGRESS=plain

# Infrastructure health thresholds
HEALTH_CHECK_TIMEOUT=30        # Seconds for each health check
MAX_RETRY_ATTEMPTS=3           # Maximum build retry attempts
BUILD_TIMEOUT_OPTIMAL=2400     # 40 minutes for optimal strategy
BUILD_TIMEOUT_EMERGENCY=900    # 15 minutes for emergency strategy

# Cache configuration
CACHE_LOCAL_PATH=/tmp/cache    # Local cache directory
CACHE_REGISTRY_REF=myorg/cache # Registry cache reference
```

### Strategy Customization

```bash
# Custom health score thresholds
THRESHOLD_OPTIMAL=80    # Use optimal strategy above this score
THRESHOLD_RESILIENT=60  # Use resilient strategy above this score
THRESHOLD_CONSERVATIVE=40  # Use conservative strategy above this score
```

## Troubleshooting

### Common Issues

#### "Our services aren't available right now"
**Cause**: GitHub Actions cache service outage
**Solution**: System automatically switches to local cache strategy

#### "timeout awaiting registry response"
**Cause**: Container registry connectivity issues
**Solution**: System tries alternative registries and mirrors

#### "failed to resolve Go proxy"
**Cause**: Go proxy service outage
**Solution**: System rotates through multiple proxy providers

#### Build fails with "no space left on device"
**Cause**: Insufficient disk space for cache
**Solution**: 
```bash
# Clean up build cache
docker system prune -af
docker builder prune -af
rm -rf /tmp/cache/*
```

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
export DEBUG=1
export BUILDKIT_PROGRESS=plain
./scripts/deploy/smart-docker-build.sh conductor-loop test latest
```

## Future Enhancements

### Planned Features
1. **Adaptive cache sizing** based on available disk space
2. **Regional mirror selection** based on geographic location
3. **Build performance analytics** and optimization recommendations
4. **Integration with service status APIs** for proactive health checking
5. **Custom mirror configuration** via environment variables

### Contribution Guidelines
1. Test new mirrors thoroughly before adding
2. Ensure fallback strategies maintain security standards
3. Add comprehensive error handling for new failure modes
4. Update documentation for new configuration options

## References

- [Docker Buildx Documentation](https://docs.docker.com/buildx/)
- [GitHub Actions Cache](https://docs.github.com/en/actions/using-workflows/caching-dependencies)
- [Go Module Proxy Protocol](https://go.dev/ref/mod#module-proxy)
- [Container Registry Alternatives](https://docs.docker.com/docker-hub/mirror/)
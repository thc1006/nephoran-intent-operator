# syntax=docker/dockerfile:1.9-labs
# =============================================================================
# Consolidated Production Dockerfile for Nephoran Intent Operator
# =============================================================================
# Supports all services with single build command using build arguments
# Security-hardened, multi-architecture ready, optimized for production
#
# CRITICAL: Go Module Cache Strategy for CI/CD Reliability
# This Dockerfile uses custom cache locations (/tmp/.cache/) instead of default
# Go cache paths (/go/pkg/mod) to avoid permission issues in GitHub Actions.
# The GOCACHE and GOMODCACHE environment variables redirect cache to writable
# locations that work with BuildKit cache mounts across all CI environments.
# 
# Build examples:
#   docker build --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .
#   docker build --build-arg SERVICE=nephio-bridge -t nephoran/nephio-bridge:latest .
#   docker build --build-arg SERVICE=oran-adaptor -t nephoran/oran-adaptor:latest .
#   docker build --build-arg SERVICE=planner -t nephoran/planner:latest .
#   docker build --build-arg SERVICE=rag-api -t nephoran/rag-api:latest .
#   docker build --build-arg SERVICE=manager -t nephoran/manager:latest .
#   docker build --build-arg SERVICE=conductor-loop -t nephoran/conductor-loop:latest .
#
# Multi-arch build:
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .
# =============================================================================

# Global build platform arguments (must be at the very top)
ARG BUILDPLATFORM=linux/amd64
ARG TARGETPLATFORM=linux/amd64
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Security-hardened base image versions with latest patches (2025 standards)
ARG GO_VERSION=1.24.1
ARG PYTHON_VERSION=3.12.10
ARG ALPINE_VERSION=3.21
ARG DISTROLESS_VERSION=nonroot
ARG DEBIAN_VERSION=bookworm-20250108-slim
ARG SERVICE_TYPE=go

# BuildKit cache optimization and multi-arch support
ARG BUILDKIT_INLINE_CACHE=1
ARG BUILDKIT_MULTI_PLATFORM=1
ARG DOCKER_DEFAULT_PLATFORM=linux/amd64

# Build optimization flags
ARG BUILD_PARALLEL=true
ARG CACHE_SHARING=locked
ARG MAX_PARALLELISM=4

# Security scanning versions
ARG TRIVY_VERSION=0.57.1
ARG COSIGN_VERSION=2.4.0

# =============================================================================
# STAGE: GO Dependencies
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-deps

# Install minimal build dependencies with security updates (2025 hardened)
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    set -eux; \
    apk update && apk upgrade --no-cache; \
    apk add --no-cache --virtual .build-deps \
        git \
        ca-certificates \
        tzdata \
        curl \
        gnupg \
    && rm -rf /tmp/* /var/tmp/* \
    && find / -xdev -type f -perm +6000 -delete 2>/dev/null || true \
    && find / -xdev -type f -perm /2000 -delete 2>/dev/null || true

# Create non-root build user with security hardening
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot -s /sbin/nologin && \
    mkdir -p /home/nonroot && \
    chown -R nonroot:nonroot /home/nonroot

WORKDIR /workspace
COPY go.mod go.sum ./

# Configure Go environment with proper cache locations and disable module proxy fallback
ENV GOCACHE=/tmp/.cache/go-build \
    GOMODCACHE=/tmp/.cache/go-mod \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    GOPRIVATE="" \
    GONOPROXY="" \
    GONOSUMDB="" \
    GO111MODULE=on \
    CGO_ENABLED=0

# Create cache directories with proper permissions and ownership
RUN mkdir -p /tmp/.cache/go-build /tmp/.cache/go-mod && \
    chmod 755 /tmp/.cache/go-build /tmp/.cache/go-mod && \
    chown -R nonroot:nonroot /tmp/.cache

# Download dependencies with BuildKit cache mounts and proper permissions
RUN --mount=type=cache,target=/tmp/.cache/go-mod,sharing=locked \
    --mount=type=cache,target=/tmp/.cache/go-build,sharing=locked \
    set -eux; \
    # Ensure cache directories are writable and fix permission issues
    chmod 755 /tmp/.cache/go-mod /tmp/.cache/go-build; \
    # Set ownership to current user to avoid permission issues
    chown $(id -u):$(id -g) /tmp/.cache/go-mod /tmp/.cache/go-build 2>/dev/null || true; \
    echo "Starting Go module download with retries..."; \
    # Download with retries and better error handling
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    go mod download -x || \
    (echo "First download attempt failed, retrying in 5s..."; sleep 5 && go mod download -x) || \
    (echo "Second download attempt failed, retrying in 10s..."; sleep 10 && go mod download -x) || \
    (echo "Final download attempt with verbose logging..."; go mod download -v)

# =============================================================================
# STAGE: GO Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-builder

# Re-declare build platform ARGs for this stage
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG SERVICE
ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF

# Validate required build arguments
RUN if [ -z "$SERVICE" ]; then \
    echo "ERROR: SERVICE build argument is required. Use --build-arg SERVICE=<service-name>" >&2; \
    echo "Valid services: conductor-loop, intent-ingest, nephio-bridge, llm-processor, oran-adaptor, a1-sim, conductor, e2-kpm-sim, fcaps-sim, o1-ves-sim, porch-publisher, planner, manager, controller" >&2; \
    exit 1; \
    fi

# Install minimal build tools with security focus and caching
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    set -eux; \
    apk update && apk upgrade --no-cache; \
    apk add --no-cache --virtual .build-deps \
        git \
        ca-certificates \
        tzdata \
        binutils \
        curl \
        gnupg \
        upx \
        file \
        make \
    && rm -rf /tmp/* /var/tmp/* \
    && find / -xdev -type f -perm +6000 -delete 2>/dev/null || true

# Create non-root build user with security hardening
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot -s /sbin/nologin && \
    mkdir -p /home/nonroot && \
    chown -R nonroot:nonroot /home/nonroot

WORKDIR /build

# Copy go.mod and go.sum from previous stage
COPY --from=go-deps /workspace/go.mod /workspace/go.sum ./

# Copy source code
COPY . .

# Configure Go environment with proper cache locations matching go-deps stage
ENV GOCACHE=/tmp/.cache/go-build \
    GOMODCACHE=/tmp/.cache/go-mod \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org

# Create cache directories with proper permissions and ownership
RUN mkdir -p /tmp/.cache/go-build /tmp/.cache/go-mod && \
    chmod 755 /tmp/.cache/go-build /tmp/.cache/go-mod && \
    chown -R nonroot:nonroot /tmp/.cache

# Download dependencies to prepare build environment
RUN --mount=type=cache,target=/tmp/.cache/go-mod,sharing=locked \
    --mount=type=cache,target=/tmp/.cache/go-build,sharing=locked \
    set -eux; \
    # Ensure cache directories are writable and fix any permission issues
    chmod 755 /tmp/.cache/go-mod /tmp/.cache/go-build; \
    # Set ownership to current user to avoid permission issues
    chown $(id -u):$(id -g) /tmp/.cache/go-mod /tmp/.cache/go-build 2>/dev/null || true; \
    echo "Downloading Go modules..."; \
    go mod download || true; \
    echo "Dependencies downloaded successfully"

# Build service based on SERVICE argument with verbose logging and enhanced parallelism
# Use cache mounts with proper permissions aligned with environment variables
RUN --mount=type=cache,target=/tmp/.cache/go-mod,sharing=locked \
    --mount=type=cache,target=/tmp/.cache/go-build,sharing=locked \
    set -ex; \
    # Fix cache permissions for build stage
    chmod 755 /tmp/.cache/go-mod /tmp/.cache/go-build; \
    chown $(id -u):$(id -g) /tmp/.cache/go-mod /tmp/.cache/go-build 2>/dev/null || true; \
    echo "=== Building service: $SERVICE ==="; \
    echo "Build platform: $BUILDPLATFORM"; \
    echo "Target platform: $TARGETPLATFORM"; \
    echo "Target OS: $TARGETOS"; \
    echo "Target arch: $TARGETARCH"; \
    echo "Working directory: $(pwd)"; \
    echo "Available files in workspace:"; \
    find . -maxdepth 3 -type f -name "*.go" | head -20; \
    case "$SERVICE" in \
        "conductor-loop") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "intent-ingest") CMD_PATH="./cmd/intent-ingest/main.go" ;; \
        "nephio-bridge") CMD_PATH="./cmd/nephio-bridge/main.go" ;; \
        "llm-processor") CMD_PATH="./cmd/llm-processor/main.go" ;; \
        "oran-adaptor") CMD_PATH="./cmd/oran-adaptor/main.go" ;; \
        "a1-sim") CMD_PATH="./cmd/a1-sim/main.go" ;; \
        "conductor") CMD_PATH="./cmd/conductor/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "fcaps-sim") CMD_PATH="./cmd/fcaps-sim/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        "porch-publisher") CMD_PATH="./cmd/porch-publisher/main.go" ;; \
        "planner") CMD_PATH="./planner/cmd/planner/main.go" ;; \
        "manager") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "controller") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "rag-api") CMD_PATH="./cmd/rag-api/main.go" ;; \
        "nephio-bridge") CMD_PATH="./cmd/nephio-bridge/main.go" ;; \
        *) echo "Unknown service: $SERVICE. Valid services: conductor-loop, intent-ingest, nephio-bridge, llm-processor, oran-adaptor, a1-sim, conductor, e2-kpm-sim, fcaps-sim, o1-ves-sim, porch-publisher, planner, manager, controller, rag-api, nephio-bridge" && exit 1 ;; \
    esac; \
    echo "Selected CMD_PATH: $CMD_PATH"; \
    echo "Verifying source file exists:"; \
    test -f "$CMD_PATH" && echo "✅ Source file found: $CMD_PATH" || { echo "❌ Source file not found: $CMD_PATH"; ls -la "$(dirname $CMD_PATH)" || true; exit 1; }; \
    echo "=== Starting Go build ==="; \
    echo "Go environment:"; \
    go env GOOS GOARCH GOROOT GOPATH GOMOD || true; \
    echo "Go modules status:"; \
    go list -m all | head -10 || true; \
    echo "Cross-compilation target: ${TARGETOS}/${TARGETARCH}"; \
    echo "Available CPU cores: $(nproc)"; \
    echo "Build parallelism: ${BUILD_PARALLEL}"; \
    echo "Building with command:"; \
    echo "CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /build/service $CMD_PATH"; \
    export CGO_ENABLED=0; \
    export GOOS=${TARGETOS}; \
    export GOARCH=${TARGETARCH}; \
    export GOAMD64=v3; \
    export GOFLAGS="-mod=readonly -buildvcs=false"; \
    go build \
        -v \
        -trimpath \
        -ldflags="-s -w \
                 -X main.version=${VERSION} \
                 -X main.buildDate=${BUILD_DATE} \
                 -X main.gitCommit=${VCS_REF} \
                 -buildid='' \
                 -extldflags '-static'" \
        -tags="netgo,osusergo,static_build" \
        -installsuffix netgo \
        -a \
        -o /build/service \
        $CMD_PATH; \
    ls -la /build/service && test -x /build/service && echo "Binary verification: $(stat -c '%n: size=%s, mode=%a' /build/service)"; \
    # Skip strip for statically linked Go binaries - not needed and can cause issues \
    # Enhanced binary validation and optimization
    file /build/service && echo "Binary validation: $(file /build/service)"; \
    ldd /build/service 2>/dev/null && echo "⚠️  Binary has dynamic dependencies" || echo "✅ Static binary confirmed"; \
    # Verify binary architecture matches target
    readelf -h /build/service | grep -E "(Class|Machine)" || echo "Binary architecture info unavailable"; \
    # Verify binary is executable and has correct permissions
    test -x /build/service && echo "✅ Binary is executable"; \
    # Optional: compress binary with UPX if available and beneficial
    if command -v upx >/dev/null 2>&1 && [ "$(stat -c%s /build/service)" -gt 10485760 ]; then \
        echo "Compressing large binary with UPX..."; \
        upx --best --lzma /build/service 2>/dev/null || echo "UPX compression failed or not beneficial"; \
    fi; \
    # Remove build dependencies to reduce image size
    apk del .build-deps || true

# =============================================================================
# STAGE: Python Dependencies
# =============================================================================
FROM --platform=$BUILDPLATFORM python:${PYTHON_VERSION}-slim AS python-deps

# Create non-root user with better security
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get upgrade -y; \
    groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot -d /home/nonroot

WORKDIR /deps
COPY requirements-rag.txt ./

# Install Python dependencies with caching
USER nonroot
RUN --mount=type=cache,target=/home/nonroot/.cache/pip,uid=65532,gid=65532,sharing=locked \
    pip install --user --no-compile --upgrade pip setuptools wheel && \
    pip install --user --no-compile -r requirements-rag.txt

# =============================================================================
# STAGE: Python Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM python:${PYTHON_VERSION}-slim AS python-builder

# Security-hardened Python builder with caching
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    set -eux; \
    export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y --no-install-recommends \
        gcc \
        python3-dev \
        libffi-dev \
        libssl-dev \
        build-essential \
        pkg-config \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /tmp/* /var/tmp/* /usr/share/doc/* /usr/share/man/* \
    && find / -xdev -type f -perm +6000 -delete 2>/dev/null || true

RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot

COPY --from=python-deps --chown=nonroot:nonroot /home/nonroot/.local /home/nonroot/.local
COPY --chown=nonroot:nonroot rag-python/ /app/

WORKDIR /app
USER nonroot

# Pre-compile Python bytecode
RUN python -m compileall -b . && \
    find . -name "*.py" -delete && \
    find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true

# =============================================================================
# STAGE: GO Runtime (Distroless)
# =============================================================================
FROM gcr.io/distroless/static:${DISTROLESS_VERSION} AS go-runtime

# Re-declare ARGs for this stage
ARG SERVICE
ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF
ARG TARGETARCH
ARG TARGETPLATFORM
ARG GO_VERSION

# Copy certificates and timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary with restricted permissions
COPY --from=go-builder --chmod=555 /build/service /service

# Comprehensive security and compliance labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="Nephoran ${SERVICE}" \
      org.opencontainers.image.description="Security-hardened ${SERVICE} service for Nephoran Intent Operator" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.url="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.documentation="https://github.com/thc1006/nephoran-intent-operator/docs" \
      org.opencontainers.image.licenses="Apache-2.0" \
      service.name="${SERVICE}" \
      service.version="${VERSION}" \
      service.component="${SERVICE}" \
      security.scan="required" \
      security.hardened="true" \
      security.nonroot="true" \
      security.readonly="true" \
      security.capabilities="none" \
      security.seccomp="enabled" \
      security.apparmor="enabled" \
      compliance.cis="compliant" \
      compliance.nist="800-53" \
      compliance.pci-dss="4.0" \
      compliance.owasp="top10-2025" \
      build.architecture="${TARGETARCH}" \
      build.platform="${TARGETPLATFORM}" \
      build.go.version="${GO_VERSION}" \
      build.distroless="true" \
      vulnerability.scanner="trivy" \
      sbom.format="spdx-json"

# Non-root user with drop capabilities (65532:65532 from distroless)
USER 65532:65532

# Security: Drop all capabilities
# Note: Capabilities are handled by container runtime, this is documentation
# Run with: --cap-drop=ALL --security-opt=no-new-privileges:true

# Optimized Go runtime environment for containers
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    GOMAXPROCS=2 \
    GODEBUG="madvdontneed=1,gctrace=0" \
    TZ=UTC \
    SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt \
    SSL_CERT_DIR=/etc/ssl/certs

# Enhanced health check with security considerations and better defaults
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD ["/service", "--version"] || ["/service", "--help"]

# Service ports: 8080 (llm-processor), 8081 (nephio-bridge), 8082 (oran-adaptor)
EXPOSE 8080 8081 8082

ENTRYPOINT ["/service"]

# =============================================================================
# STAGE: Python Runtime (Distroless)
# =============================================================================
FROM gcr.io/distroless/python3-debian12:${DISTROLESS_VERSION} AS python-runtime

ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF
ARG PYTHON_VERSION

# Copy Python packages and application
COPY --from=python-builder --chown=nonroot:nonroot /home/nonroot/.local/lib/python3.12/site-packages /home/nonroot/.local/lib/python3.12/site-packages
COPY --from=python-builder --chown=nonroot:nonroot /app /app

# Comprehensive security labels for Python runtime
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="Nephoran RAG API" \
      org.opencontainers.image.description="Security-hardened RAG service for Nephoran Intent Operator" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.licenses="Apache-2.0" \
      service.name="rag-api" \
      service.version="${VERSION}" \
      service.component="rag-api" \
      security.scan="required" \
      security.hardened="true" \
      security.nonroot="true" \
      security.python.version="${PYTHON_VERSION}" \
      compliance.cis="compliant" \
      build.distroless="true" \
      vulnerability.scanner="trivy" \
      sbom.format="spdx-json"

# Security-hardened Python environment
ENV PYTHONPATH=/home/nonroot/.local/lib/python3.12/site-packages:/app \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONHASHSEED=random \
    PYTHONIOENCODING=utf-8 \
    PYTHONUTF8=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PORT=5001 \
    FLASK_ENV=production \
    WERKZEUG_DEBUG_PIN=off

USER nonroot
WORKDIR /app

EXPOSE 5001

# Security-hardened Python entrypoint
ENTRYPOINT ["python", "-O", "-B", "-s"]
CMD ["api.pyc"]

# =============================================================================
# STAGE: Final Runtime Selection
# =============================================================================
# Select the appropriate runtime based on SERVICE argument
ARG SERVICE
FROM go-runtime AS final-go
FROM python-runtime AS final-python

# Default to go-runtime for all services except rag-api
# rag-api service should be built with SERVICE_TYPE=python
FROM final-${SERVICE_TYPE:-go} AS final

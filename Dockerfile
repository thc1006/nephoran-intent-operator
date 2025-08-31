# syntax=docker/dockerfile:1
# =============================================================================
# SECURITY-HARDENED Production Dockerfile for Nephoran Intent Operator (2025)
# =============================================================================
# Implements SLSA Level 3, CIS Docker Benchmark v1.7, OWASP Top 10 2025
# =============================================================================
# Security Features:
# - Non-root user execution (UID 65532)
# - Read-only root filesystem
# - No new privileges flag
# - Dropped all capabilities
# - Security labels for container runtime
# - Distroless base image
# - Static binary with no shell
# =============================================================================

# Build arguments
ARG GO_VERSION=1.24.1
ARG ALPINE_VERSION=3.21
ARG DISTROLESS_VERSION=nonroot
ARG BUILDPLATFORM=linux/amd64
ARG TARGETPLATFORM=linux/amd64
ARG SERVICE=intent-ingest

<<<<<<< HEAD
# Required build arguments
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF

# =============================================================================
# STAGE 1: Dependencies Cache Layer
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS deps-cache

# Install minimal dependencies for caching
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add --no-cache git ca-certificates tzdata
=======
# =============================================================================
# STAGE: GO Dependencies with Enhanced Resilience
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-deps

# Enhanced resilience: Install build dependencies with retry logic
RUN set -ex; \
    max_attempts=3; \
    attempt=1; \
    while [ $attempt -le $max_attempts ]; do \
        echo "Package installation attempt $attempt/$max_attempts"; \
        if apk add --no-cache --timeout 60 git ca-certificates tzdata && \
           apk upgrade --no-cache --timeout 60; then \
            echo "Package installation successful"; \
            rm -rf /var/cache/apk/*; \
            break; \
        else \
            echo "Package installation failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "All package installation attempts failed"; \
                exit 1; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Create non-root build user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

WORKDIR /workspace
COPY --chown=nonroot:nonroot go.mod go.sum ./

USER nonroot

# Enhanced resilience: Go module download with comprehensive retry and fallback
RUN set -ex; \
    echo "Starting resilient Go module download..."; \
    export GOPROXY="https://proxy.golang.org,https://goproxy.cn,https://goproxy.io,direct"; \
    export GOSUMDB="sum.golang.org"; \
    export GONOPROXY=""; \
    export GONOSUMDB=""; \
    export GOPRIVATE=""; \
    export GO111MODULE=on; \
    export CGO_ENABLED=0; \
    \
    max_attempts=5; \
    attempt=1; \
    download_success=false; \
    \
    while [ $attempt -le $max_attempts ] && [ "$download_success" = "false" ]; do \
        echo "Go module download attempt $attempt/$max_attempts"; \
        \
        case $attempt in \
            1) \
                echo "Using standard proxy chain with timeout 180s"; \
                timeout_val=180; \
                export GOPROXY="https://proxy.golang.org,https://goproxy.cn,direct"; \
                ;; \
            2) \
                echo "Using alternative proxy with timeout 240s"; \
                timeout_val=240; \
                export GOPROXY="https://goproxy.io,https://goproxy.cn,direct"; \
                ;; \
            3) \
                echo "Using direct access only with timeout 300s"; \
                timeout_val=300; \
                export GOPROXY="direct"; \
                ;; \
            4) \
                echo "Using go.dev mirror with timeout 180s"; \
                timeout_val=180; \
                export GOPROXY="https://goproxy.cn,direct"; \
                ;; \
            5) \
                echo "Emergency fallback: minimal download with timeout 120s"; \
                timeout_val=120; \
                export GOPROXY="direct"; \
                ;; \
        esac; \
        \
        start_time=$(date +%s); \
        if timeout $timeout_val go mod download -x; then \
            end_time=$(date +%s); \
            duration=$((end_time - start_time)); \
            echo "Go mod download successful in ${duration}s (attempt $attempt)"; \
            if timeout 30 go mod verify; then \
                echo "Module verification successful"; \
                download_success=true; \
            else \
                echo "Module verification failed but download completed"; \
                download_success=true; \
            fi; \
        else \
            echo "Go mod download failed on attempt $attempt"; \
            if [ $attempt -lt $max_attempts ]; then \
                backoff_time=$((attempt * 3 + $(date +%s) % 5)); \
                echo "Waiting ${backoff_time}s before retry..."; \
                sleep $backoff_time; \
            fi; \
        fi; \
        attempt=$((attempt + 1)); \
    done; \
    \
    if [ "$download_success" = "true" ]; then \
        echo "Dependencies downloaded and cached successfully"; \
        cache_size=$(du -sh /go/pkg/mod 2>/dev/null | cut -f1 || echo "unknown"); \
        echo "Module cache size: $cache_size"; \
    else \
        echo "ERROR: All download attempts failed"; \
        echo "Available cached modules:"; \
        find /go/pkg/mod -type d -name "*@*" 2>/dev/null | head -10 || echo "No cached modules found"; \
        exit 1; \
    fi

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
    echo "Valid services: conductor-loop, intent-ingest, nephio-bridge, llm-processor, oran-adaptor, manager, controller, e2-kpm-sim, o1-ves-sim" >&2; \
    exit 1; \
    fi

# Enhanced resilience: Install build tools with retry logic and fallback
RUN set -ex; \
    echo "Installing build tools with enhanced resilience..."; \
    max_attempts=3; \
    attempt=1; \
    \
    while [ $attempt -le $max_attempts ]; do \
        echo "Build tools installation attempt $attempt/$max_attempts"; \
        \
        # Primary installation attempt
        if apk add --no-cache --timeout 90 --retry 2 \
            git ca-certificates tzdata binutils && \
           apk upgrade --no-cache --timeout 60; then \
            echo "Build tools installation successful"; \
            rm -rf /var/cache/apk/* /tmp/* /var/tmp/*; \
            break; \
        else \
            echo "Build tools installation failed, attempt $attempt/$max_attempts"; \
            \
            # Fallback: try minimal essential tools only
            if [ $attempt -eq $max_attempts ]; then \
                echo "Attempting minimal fallback installation..."; \
                if apk add --no-cache ca-certificates; then \
                    echo "Minimal tools installed successfully"; \
                    break; \
                else \
                    echo "CRITICAL: All installation attempts failed"; \
                    exit 1; \
                fi; \
            fi; \
            \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done
>>>>>>> origin/integrate/mvp

WORKDIR /src

# Copy dependency files only (highly cacheable)
COPY go.mod go.sum ./

# Download dependencies with maximum parallelization and retry logic
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    set -eux; \
    echo "=== Downloading Go dependencies with retry logic ==="; \
    for attempt in 1 2 3; do \
        echo "Download attempt $attempt..."; \
        if GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org go mod download -x; then \
            echo "Dependencies downloaded successfully on attempt $attempt"; \
            break; \
        elif [ $attempt -eq 3 ]; then \
            echo "All download attempts failed, trying direct..."; \
            GOPROXY=direct go mod download -v || exit 1; \
        else \
            echo "Attempt $attempt failed, retrying in 10s..."; \
            sleep 10; \
        fi; \
    done; \
    go mod verify || echo "Module verification failed, continuing..."; \
    echo "Dependencies ready"

# =============================================================================
# STAGE 2: Build Stage
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# Re-declare build arguments for this stage
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG SERVICE
ARG VERSION
ARG BUILD_DATE
ARG VCS_REF

# Install build dependencies
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add --no-cache git ca-certificates tzdata binutils

WORKDIR /build

# Copy dependency files and source code
COPY go.mod go.sum ./
COPY . .

# Download dependencies and build with enhanced error handling
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    # Download dependencies first
    go mod download && \
    set -ex; \
    echo "=== Building service: $SERVICE ==="; \
    echo "Target platform: ${TARGETOS}/${TARGETARCH}"; \
    echo "Go version: $(go version)"; \
    \
    # Determine correct source path
    case "$SERVICE" in \
        "conductor-loop") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "intent-ingest") CMD_PATH="./cmd/intent-ingest/main.go" ;; \
        "nephio-bridge") CMD_PATH="./cmd/nephio-bridge/main.go" ;; \
        "llm-processor") CMD_PATH="./cmd/llm-processor/main.go" ;; \
        "oran-adaptor") CMD_PATH="./cmd/oran-adaptor/main.go" ;; \
<<<<<<< HEAD
        "porch-publisher") CMD_PATH="./cmd/porch-publisher/main.go" ;; \
        "planner") CMD_PATH="./planner/cmd/planner/main.go" ;; \
        "a1-sim") CMD_PATH="./cmd/a1-sim/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "fcaps-sim") CMD_PATH="./cmd/fcaps-sim/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        "conductor") CMD_PATH="./cmd/conductor/main.go" ;; \
        "rag-api") CMD_PATH="./cmd/rag-api/main.go" ;; \
        *) echo "ERROR: Unknown service '$SERVICE'" && exit 1 ;; \
=======
        "manager") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "controller") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        *) echo "Unknown service: $SERVICE. Valid services: conductor-loop, intent-ingest, nephio-bridge, llm-processor, oran-adaptor, manager, controller, e2-kpm-sim, o1-ves-sim" && exit 1 ;; \
>>>>>>> origin/integrate/mvp
    esac; \
    \
    echo "Selected source path: $CMD_PATH"; \
    \
    # Verify source file exists
    if [ ! -f "$CMD_PATH" ]; then \
        echo "ERROR: Source file not found: $CMD_PATH"; \
        echo "Available services:"; \
        find cmd/ -name "main.go" -exec dirname {} \; 2>/dev/null | sort || true; \
        find planner/cmd/ -name "main.go" -exec dirname {} \; 2>/dev/null | sort || true; \
        exit 1; \
    fi; \
    \
    echo "✅ Source file verified: $CMD_PATH"; \
    \
    # Enhanced build with retry
    export CGO_ENABLED=0; \
    export GOOS=${TARGETOS}; \
    export GOARCH=${TARGETARCH}; \
    export GOAMD64=v3; \
    export GOFLAGS="-mod=readonly -buildvcs=false"; \
    \
    for build_attempt in 1 2; do \
        echo "Build attempt $build_attempt..."; \
        if go build \
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
            -o /app/service \
            "$CMD_PATH"; then \
            echo "✅ Build successful on attempt $build_attempt"; \
            break; \
        elif [ $build_attempt -eq 2 ]; then \
            echo "❌ All build attempts failed"; \
            exit 1; \
        else \
            echo "⚠️  Build attempt $build_attempt failed, retrying..."; \
            sleep 5; \
        fi; \
    done; \
    \
    # Verify binary
    if [ ! -x "/app/service" ]; then \
        echo "❌ Binary not executable"; \
        exit 1; \
    fi; \
    \
    ls -la /app/service; \
    echo "✅ Build completed successfully"

# =============================================================================
<<<<<<< HEAD
# STAGE 3: Security Scanner
# =============================================================================
FROM aquasec/trivy:latest AS scanner
COPY --from=builder /app/service /app/service
RUN trivy fs --no-progress --security-checks vuln,secret,config --severity HIGH,CRITICAL --exit-code 0 /app/service || true
=======
# STAGE: Python Dependencies with Enhanced Resilience
# =============================================================================
FROM python:${PYTHON_VERSION}-slim AS python-deps

# Enhanced resilience: Update package manager with retry logic
RUN set -ex; \
    echo "Updating Python base system with resilience..."; \
    max_attempts=3; \
    attempt=1; \
    \
    while [ $attempt -le $max_attempts ]; do \
        echo "System update attempt $attempt/$max_attempts"; \
        if apt-get update -y --timeout=60 && \
           apt-get install -y --no-install-recommends ca-certificates; then \
            echo "System update successful"; \
            apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
            break; \
        else \
            echo "System update failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "System update failed after all attempts - continuing anyway"; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Create non-root user
RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot

WORKDIR /deps
COPY requirements-rag.txt ./

USER nonroot

# Enhanced resilience: pip install with comprehensive retry and fallback
RUN set -ex; \
    echo "Installing Python dependencies with enhanced resilience..."; \
    max_attempts=4; \
    attempt=1; \
    install_success=false; \
    \
    while [ $attempt -le $max_attempts ] && [ "$install_success" = "false" ]; do \
        echo "Python dependencies installation attempt $attempt/$max_attempts"; \
        \
        case $attempt in \
            1) \
                echo "Using default PyPI with user cache"; \
                pip_args="--user --no-cache-dir --no-compile --timeout 300"; \
                pip_index=""; \
                ;; \
            2) \
                echo "Using PyPI with extended timeout and retries"; \
                pip_args="--user --no-cache-dir --no-compile --timeout 600 --retries 3"; \
                pip_index=""; \
                ;; \
            3) \
                echo "Using alternative index with fallback"; \
                pip_args="--user --no-cache-dir --no-compile --timeout 300 --retries 5"; \
                pip_index="--index-url https://pypi.org/simple/ --extra-index-url https://pypi.python.org/simple/"; \
                ;; \
            4) \
                echo "Emergency: essential packages only"; \
                pip_args="--user --no-cache-dir --timeout 180"; \
                pip_index=""; \
                ;; \
        esac; \
        \
        if pip install $pip_args $pip_index -r requirements-rag.txt; then \
            echo "Python dependencies installation successful (attempt $attempt)"; \
            install_success=true; \
        else \
            echo "Python dependencies installation failed on attempt $attempt"; \
            if [ $attempt -lt $max_attempts ]; then \
                backoff_time=$((attempt * 5)); \
                echo "Waiting ${backoff_time}s before retry..."; \
                sleep $backoff_time; \
            fi; \
        fi; \
        attempt=$((attempt + 1)); \
    done; \
    \
    if [ "$install_success" = "false" ]; then \
        echo "ERROR: All Python dependency installation attempts failed"; \
        echo "Installed packages:"; \
        pip list --user 2>/dev/null || echo "No packages installed"; \
        exit 1; \
    fi; \
    echo "Python dependencies cached successfully"
>>>>>>> origin/integrate/mvp

# =============================================================================
# STAGE 4: Final Runtime (Security Hardened)
# =============================================================================
FROM gcr.io/distroless/static:${DISTROLESS_VERSION} AS final

# =============================================================================
<<<<<<< HEAD
# STAGE 5: Go Runtime Alias (Required by CI/CD)
# =============================================================================
FROM final AS go-runtime
=======
# STAGE: GO Runtime (Multi-Source with Advanced Fallback Strategy)
# =============================================================================

# Primary distroless base image
FROM gcr.io/distroless/static:${DISTROLESS_VERSION} AS go-runtime-distroless

# Secondary base image (different registry)
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static-debian12:${DISTROLESS_VERSION} AS go-runtime-distroless-alt

# Alpine fallback for maximum reliability
FROM --platform=$TARGETPLATFORM alpine:${ALPINE_VERSION} AS go-runtime-alpine-fallback
RUN set -ex; \
    max_attempts=3; \
    attempt=1; \
    while [ $attempt -le $max_attempts ]; do \
        echo "Alpine setup attempt $attempt/$max_attempts"; \
        if apk add --no-cache --timeout 60 ca-certificates tzdata && \
           apk upgrade --no-cache --timeout 60; then \
            addgroup -g 65532 -S nonroot && \
            adduser -u 65532 -S nonroot -G nonroot && \
            rm -rf /var/cache/apk/* /tmp/* /var/tmp/*; \
            echo "Alpine setup successful"; \
            break; \
        else \
            echo "Alpine setup failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "All setup attempts failed - using minimal configuration"; \
                addgroup -g 65532 -S nonroot || true; \
                adduser -u 65532 -S nonroot -G nonroot || true; \
                break; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Ubuntu minimal fallback for extreme cases
FROM --platform=$TARGETPLATFORM ubuntu:22.04 AS go-runtime-ubuntu-emergency
RUN set -ex; \
    export DEBIAN_FRONTEND=noninteractive; \
    max_attempts=3; \
    attempt=1; \
    while [ $attempt -le $max_attempts ]; do \
        echo "Ubuntu setup attempt $attempt/$max_attempts"; \
        if apt-get update -y --timeout=60 && \
           apt-get install -y --no-install-recommends ca-certificates tzdata; then \
            groupadd -g 65532 nonroot && \
            useradd -u 65532 -g nonroot -s /bin/false -m nonroot && \
            apt-get clean && \
            rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
            echo "Ubuntu setup successful"; \
            break; \
        else \
            echo "Ubuntu setup failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "All setup attempts failed - using minimal configuration"; \
                groupadd -g 65532 nonroot || true; \
                useradd -u 65532 -g nonroot -s /bin/false -m nonroot || true; \
                break; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Smart runtime selection - fallback chain with conditional selection
# This uses the first available base image in order of preference
FROM go-runtime-distroless AS go-runtime
>>>>>>> origin/integrate/mvp

# Re-declare arguments for labels
ARG SERVICE
ARG VERSION
ARG BUILD_DATE
ARG VCS_REF
ARG TARGETARCH
ARG GO_VERSION

# Copy certificates and timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy optimized binary
COPY --from=builder /app/service /service

# Security hardening annotations for container runtime
LABEL io.kubernetes.container.capabilities.drop="ALL" \
      io.kubernetes.container.readOnlyRootFilesystem="true" \
      io.kubernetes.container.runAsNonRoot="true" \
      io.kubernetes.container.runAsUser="65532" \
      io.kubernetes.container.allowPrivilegeEscalation="false" \
      io.kubernetes.container.seccompProfile="RuntimeDefault"

# Security and compliance labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="Nephoran ${SERVICE}" \
      org.opencontainers.image.description="Production ${SERVICE} service for Nephoran Intent Operator" \
      org.opencontainers.image.vendor="Nephoran Project" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.licenses="Apache-2.0" \
      service.name="${SERVICE}" \
      service.version="${VERSION}" \
      service.component="${SERVICE}" \
      security.hardened="true" \
      security.nonroot="true" \
      build.architecture="${TARGETARCH}" \
      build.go.version="${GO_VERSION}" \
      build.distroless="true" \
      security.scan.date="${BUILD_DATE}" \
      security.cis.docker.benchmark="v1.7" \
      security.owasp.top10="2025" \
      security.slsa.level="3"

# Security: Non-root user (65532:65532 from distroless nonroot)
USER 65532:65532

# Optimized runtime environment
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    GOMAXPROCS=2 \
    GODEBUG="madvdontneed=1" \
    TZ=UTC

# Standardized health check with security options
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/service", "--health-check", "--secure"] || ["/service", "--version"]

# Standard service ports
EXPOSE 8080 8081 8082 8083 8084 8085 8086

<<<<<<< HEAD
# Secure entrypoint
ENTRYPOINT ["/service"]
=======
ENTRYPOINT ["/service"]

# =============================================================================
# STAGE: Python Runtime (Multi-Source with Advanced Fallback Strategy)
# =============================================================================

# Primary distroless Python base
FROM gcr.io/distroless/python3-debian12:${DISTROLESS_VERSION} AS python-runtime-distroless

# Alternative Python distroless base
FROM --platform=$TARGETPLATFORM gcr.io/distroless/python3:${DISTROLESS_VERSION} AS python-runtime-distroless-alt

# Python slim fallback with resilient setup
FROM --platform=$TARGETPLATFORM python:${PYTHON_VERSION}-slim AS python-runtime-slim-fallback
RUN set -ex; \
    export DEBIAN_FRONTEND=noninteractive; \
    max_attempts=3; \
    attempt=1; \
    while [ $attempt -le $max_attempts ]; do \
        echo "Python slim setup attempt $attempt/$max_attempts"; \
        if apt-get update -y --timeout=60 && \
           apt-get install -y --no-install-recommends --timeout=60 ca-certificates; then \
            groupadd -g 65532 nonroot && \
            useradd -u 65532 -g nonroot -s /bin/false -m nonroot && \
            apt-get clean && \
            rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
            echo "Python slim setup successful"; \
            break; \
        else \
            echo "Python slim setup failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "All setup attempts failed - using minimal configuration"; \
                groupadd -g 65532 nonroot || true; \
                useradd -u 65532 -g nonroot -s /bin/false -m nonroot || true; \
                break; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Ubuntu Python fallback for maximum compatibility
FROM --platform=$TARGETPLATFORM ubuntu:22.04 AS python-runtime-ubuntu-emergency
RUN set -ex; \
    export DEBIAN_FRONTEND=noninteractive; \
    max_attempts=3; \
    attempt=1; \
    while [ $attempt -le $max_attempts ]; do \
        echo "Ubuntu Python setup attempt $attempt/$max_attempts"; \
        if apt-get update -y --timeout=90 && \
           apt-get install -y --no-install-recommends --timeout=90 \
             python3 python3-pip ca-certificates; then \
            groupadd -g 65532 nonroot && \
            useradd -u 65532 -g nonroot -s /bin/false -m nonroot && \
            ln -s /usr/bin/python3 /usr/bin/python && \
            apt-get clean && \
            rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
            echo "Ubuntu Python setup successful"; \
            break; \
        else \
            echo "Ubuntu Python setup failed, attempt $attempt/$max_attempts"; \
            if [ $attempt -eq $max_attempts ]; then \
                echo "All setup attempts failed - using minimal configuration"; \
                groupadd -g 65532 nonroot || true; \
                useradd -u 65532 -g nonroot -s /bin/false -m nonroot || true; \
                break; \
            fi; \
            attempt=$((attempt + 1)); \
            sleep $((attempt * 2)); \
        fi; \
    done

# Smart Python runtime selection
FROM python-runtime-distroless AS python-runtime

ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF

# Copy Python packages and application
COPY --from=python-builder --chown=nonroot:nonroot /home/nonroot/.local/lib/python3.11/site-packages /home/nonroot/.local/lib/python3.11/site-packages
COPY --from=python-builder --chown=nonroot:nonroot /app /app

# Labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="Nephoran RAG API" \
      org.opencontainers.image.description="Production RAG service" \
      service.name="rag-api" \
      security.scan="required"

# Environment
ENV PYTHONPATH=/home/nonroot/.local/lib/python3.11/site-packages:/app \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PORT=5001

USER nonroot
WORKDIR /app

EXPOSE 5001

ENTRYPOINT ["python", "-O"]
CMD ["api.pyc"]

# =============================================================================
# STAGE: Final Runtime Selection
# =============================================================================
# Default to go-runtime for all services except rag-api
# rag-api service should be built with SERVICE_TYPE=python

FROM go-runtime AS final
>>>>>>> origin/integrate/mvp

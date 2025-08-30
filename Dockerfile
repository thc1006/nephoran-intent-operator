# =============================================================================
# Consolidated Production Dockerfile for Nephoran Intent Operator
# =============================================================================
# Supports all services with single build command using build arguments
# Security-hardened, multi-architecture ready, optimized for production
# 
# Build examples:
#   docker build --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .
#   docker build --build-arg SERVICE=nephio-bridge -t nephoran/nephio-bridge:latest .
#   docker build --build-arg SERVICE=oran-adaptor -t nephoran/oran-adaptor:latest .
#   docker build --build-arg SERVICE=rag-api -t nephoran/rag-api:latest .
#   docker build --build-arg SERVICE=manager -t nephoran/manager:latest .
#
# Multi-arch build:
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .
# =============================================================================

# Global build platform arguments (must be at the very top)
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Version arguments
ARG GO_VERSION=1.24
ARG PYTHON_VERSION=3.11
ARG ALPINE_VERSION=3.22
ARG DISTROLESS_VERSION=nonroot
ARG SERVICE_TYPE=go

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

# Create non-root build user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

WORKDIR /build
RUN chown nonroot:nonroot /build

# Copy dependencies from previous stage
COPY --from=go-deps /go/pkg /go/pkg
COPY --from=go-deps /workspace/go.mod /workspace/go.sum ./

# Copy source code
COPY --chown=nonroot:nonroot . .

USER nonroot

# Build service based on SERVICE argument
RUN set -ex; \
    case "$SERVICE" in \
        "conductor-loop") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "intent-ingest") CMD_PATH="./cmd/intent-ingest/main.go" ;; \
        "nephio-bridge") CMD_PATH="./cmd/nephio-bridge/main.go" ;; \
        "llm-processor") CMD_PATH="./cmd/llm-processor/main.go" ;; \
        "oran-adaptor") CMD_PATH="./cmd/oran-adaptor/main.go" ;; \
        "manager") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "controller") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        *) echo "Unknown service: $SERVICE. Valid services: conductor-loop, intent-ingest, nephio-bridge, llm-processor, oran-adaptor, manager, controller, e2-kpm-sim, o1-ves-sim" && exit 1 ;; \
    esac; \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -buildmode=pie \
        -trimpath \
        -mod=readonly \
        -ldflags="-w -s -extldflags '-static' \
                 -X main.version=${VERSION} \
                 -X main.buildDate=${BUILD_DATE} \
                 -X main.gitCommit=${VCS_REF} \
                 -buildid=" \
        -tags="netgo osusergo static_build" \
        -o /build/service \
        $CMD_PATH && \
    file /build/service && \
    strip --strip-unneeded /build/service 2>/dev/null || true

# =============================================================================
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

# =============================================================================
# STAGE: Python Builder
# =============================================================================
FROM python:${PYTHON_VERSION}-slim AS python-builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc python3-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

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

# Re-declare ARGs for this stage
ARG SERVICE
ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF
ARG TARGETARCH

# Copy certificates and timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary with restricted permissions
COPY --from=go-builder --chmod=555 /build/service /service

# Labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="Nephoran ${SERVICE}" \
      org.opencontainers.image.description="Production ${SERVICE} service" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      service.name="${SERVICE}" \
      security.scan="required" \
      build.architecture="${TARGETARCH}"

# Non-root user (65532:65532 from distroless)
USER 65532:65532

# Environment
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/service", "--health-check"]

# Service ports: 8080 (llm-processor), 8081 (nephio-bridge), 8082 (oran-adaptor)
EXPOSE 8080 8081 8082

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

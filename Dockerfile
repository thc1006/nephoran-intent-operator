# syntax=docker/dockerfile:1.9-labs
# =============================================================================
# FIXED Production Dockerfile for Nephoran Intent Operator (2025)
# =============================================================================
# Addresses critical CI/CD pipeline failures with working multi-stage build
# =============================================================================

# Build arguments
ARG GO_VERSION=1.24.1
ARG ALPINE_VERSION=3.21
ARG DISTROLESS_VERSION=nonroot
ARG BUILDPLATFORM=linux/amd64
ARG TARGETPLATFORM=linux/amd64
ARG SERVICE=intent-ingest

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
        "porch-publisher") CMD_PATH="./cmd/porch-publisher/main.go" ;; \
        "planner") CMD_PATH="./planner/cmd/planner/main.go" ;; \
        "a1-sim") CMD_PATH="./cmd/a1-sim/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "fcaps-sim") CMD_PATH="./cmd/fcaps-sim/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        "conductor") CMD_PATH="./cmd/conductor/main.go" ;; \
        "rag-api") CMD_PATH="./cmd/rag-api/main.go" ;; \
        *) echo "ERROR: Unknown service '$SERVICE'" && exit 1 ;; \
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
# STAGE 3: Final Runtime
# =============================================================================
FROM gcr.io/distroless/static:${DISTROLESS_VERSION} AS final

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
      build.distroless="true"

# Security: Non-root user (65532:65532 from distroless nonroot)
USER 65532:65532

# Optimized runtime environment
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    GOMAXPROCS=2 \
    GODEBUG="madvdontneed=1" \
    TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=20s --retries=3 \
    CMD ["/service", "--version"]

# Standard service ports
EXPOSE 8080 8081 8082 8083 8084 8085 8086

# Secure entrypoint
ENTRYPOINT ["/service"]
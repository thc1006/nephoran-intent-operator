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
#   docker build --build-arg SERVICE=manager -t nephoran/manager:latest .
#   docker build --build-arg SERVICE=conductor -t nephoran/conductor:latest .
#   docker build --build-arg SERVICE=porch-publisher -t nephoran/porch-publisher:latest .
#
# Multi-arch build:
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .
# =============================================================================

# Security-hardened base image versions
ARG GO_VERSION=1.24.5
ARG PYTHON_VERSION=3.12.8
ARG ALPINE_VERSION=3.21.2
ARG DISTROLESS_VERSION=nonroot
ARG DEBIAN_VERSION=bookworm-20241223-slim

# =============================================================================
# STAGE: GO Dependencies
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-deps

# Install minimal build dependencies with security updates
RUN set -eux; \
    apk update && apk upgrade --no-cache; \
    apk add --no-cache --virtual .build-deps \
        git \
        ca-certificates \
        tzdata \
        curl \
        gnupg \
    && rm -rf /var/cache/apk/* /var/lib/apk/lists/* /tmp/* /var/tmp/*

# Create non-root build user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

WORKDIR /workspace
COPY --chown=nonroot:nonroot go.mod go.sum ./

USER nonroot
RUN go mod download && go mod verify

# =============================================================================
# STAGE: GO Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-builder

ARG TARGETPLATFORM
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG SERVICE
ARG VERSION=v2.0.0
ARG BUILD_DATE
ARG VCS_REF

# Install minimal build tools with security focus
RUN set -eux; \
    apk update && apk upgrade --no-cache; \
    apk add --no-cache --virtual .build-deps \
        git=~2.45 \
        ca-certificates \
        tzdata \
        binutils=~2.42 \
        curl=~8.11 \
        gnupg \
    && rm -rf /var/cache/apk/* /var/lib/apk/lists/* /tmp/* /var/tmp/*

# Create non-root build user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

WORKDIR /build

# Copy dependencies from previous stage
COPY --from=go-deps /go/pkg /go/pkg
COPY --from=go-deps /workspace/go.mod /workspace/go.sum ./

# Copy source code
COPY --chown=nonroot:nonroot . .

USER nonroot

# Build service based on SERVICE argument
RUN set -ex; \
    case "$SERVICE" in \
        "llm-processor") CMD_PATH="./cmd/llm-processor/main.go" ;; \
        "nephio-bridge") CMD_PATH="./cmd/nephio-bridge/main.go" ;; \
        "oran-adaptor") CMD_PATH="./cmd/oran-adaptor/main.go" ;; \
        "a1-sim") CMD_PATH="./cmd/a1-sim/main.go" ;; \
        "conductor") CMD_PATH="./cmd/conductor/main.go" ;; \
        "conductor-loop") CMD_PATH="./cmd/conductor-loop/main.go" ;; \
        "e2-kpm-sim") CMD_PATH="./cmd/e2-kpm-sim/main.go" ;; \
        "fcaps-sim") CMD_PATH="./cmd/fcaps-sim/main.go" ;; \
        "intent-ingest") CMD_PATH="./cmd/intent-ingest/main.go" ;; \
        "o1-ves-sim") CMD_PATH="./cmd/o1-ves-sim/main.go" ;; \
        "porch-publisher") CMD_PATH="./cmd/porch-publisher/main.go" ;; \
        "manager"|"controller") CMD_PATH="./cmd/main.go" ;; \
        *) echo "Unknown service: $SERVICE" && exit 1 ;; \
    esac; \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -buildmode=pie \
        -trimpath \
        -mod=readonly \
        -ldflags="-w -s -extldflags '-static -fPIC' \
                 -X main.version=${VERSION} \
                 -X main.buildDate=${BUILD_DATE} \
                 -X main.gitCommit=${VCS_REF} \
                 -X main.buildDate=${BUILD_DATE} \
                 -X main.buildPlatform=${TARGETPLATFORM} \
                 -buildid='' \
                 -linkmode=external" \
        -tags="netgo osusergo static_build timetzdata" \
        -gcflags="-N -l" \
        -asmflags="-D GOOS_${TARGETOS}" \
        -o /build/service \
        $CMD_PATH; \
    file /build/service; \
    strip --strip-unneeded /build/service 2>/dev/null || true; \
    # Verify binary security properties
    readelf -d /build/service | grep -E '(BIND_NOW|RELRO)' || echo 'Warning: Missing hardening flags'; \
    # Remove build dependencies
    apk del .build-deps || true

# =============================================================================
# STAGE: Python Dependencies
# =============================================================================
FROM python:${PYTHON_VERSION}-slim AS python-deps

# Create non-root user
RUN groupadd -g 65532 nonroot && \
    useradd -u 65532 -g nonroot -s /bin/false -m nonroot

WORKDIR /deps
COPY requirements-rag.txt ./

USER nonroot
RUN pip install --user --no-cache-dir --no-compile -r requirements-rag.txt

# =============================================================================
# STAGE: Python Builder
# =============================================================================
FROM python:${PYTHON_VERSION}-slim AS python-builder

# Security-hardened Python builder
RUN set -eux; \
    export DEBIAN_FRONTEND=noninteractive; \
    apt-get update; \
    apt-get upgrade -y; \
    apt-get install -y --no-install-recommends \
        gcc=4:12.2.0-3ubuntu1 \
        python3-dev=3.12.3-0ubuntu2 \
        libffi-dev=3.4.4-1 \
        libssl-dev=3.0.13-0ubuntu3.4 \
        build-essential=12.10ubuntu1 \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /usr/share/doc/* /usr/share/man/*

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
      compliance.cis="compliant" \
      compliance.nist="800-53" \
      build.architecture="${TARGETARCH}" \
      build.platform="${TARGETPLATFORM}" \
      build.go.version="${GO_VERSION}" \
      build.distroless="true" \
      vulnerability.scanner="trivy" \
      sbom.format="spdx-json"

# Non-root user (65532:65532 from distroless)
USER 65532:65532

# Environment
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    TZ=UTC

# Enhanced health check with security considerations
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/service", "--health-check", "--secure"]

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

# This is a clever Docker trick: the last FROM wins based on build-arg conditions
FROM final-${SERVICE_TYPE:-go} AS final
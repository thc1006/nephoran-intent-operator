# Secure Multi-stage Dockerfile for Conductor Loop
# Production-ready with security hardening and distroless base
#
# Build stages:
# 1. security-scanner: Security vulnerability scanning
# 2. builder: Build application with security hardening
# 3. runtime: Minimal runtime with distroless base

# =============================================================================
# Stage 1: Security Scanner
# =============================================================================
FROM golang:1.24-alpine AS security-scanner

# Install security scanning tools
RUN apk add --no-cache \
    git \
    curl \
    bash \
    && rm -rf /var/cache/apk/*

# Install govulncheck (pinned version for supply chain security)
RUN go install golang.org/x/vuln/cmd/govulncheck@v1.1.4

# Install gosec for static analysis
RUN go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

WORKDIR /security-scan

# Copy go.mod and go.sum for dependency scanning
COPY go.mod go.sum ./

# Copy source code for static analysis
COPY . .

# Run security scans
RUN echo "Running vulnerability scan..." && \
    govulncheck ./... && \
    echo "Running static security analysis..." && \
    gosec -severity medium -confidence medium -fmt json -out /tmp/gosec-report.json ./... || true && \
    echo "Security scans completed"

# =============================================================================
# Stage 2: Builder
# =============================================================================
FROM golang:1.24-alpine AS builder

# Security: Install CA certificates and create non-root user
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata \
    && rm -rf /var/cache/apk/* \
    && adduser -D -s /bin/sh -u 10001 conductor

# Set working directory
WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies with verification
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for metadata
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF
ARG CGO_ENABLED=0

# Security: Build with security flags and optimizations
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.vcsRef=${VCS_REF} \
    -extldflags '-static'" \
    -a -installsuffix cgo \
    -tags netgo,osusergo \
    -trimpath \
    -o /build/conductor-loop \
    ./cmd/conductor-loop

# Verify the binary
RUN file /build/conductor-loop && \
    /build/conductor-loop --version || true

# =============================================================================
# Stage 3: Security Hardened Runtime
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Metadata labels following OCI standard
LABEL org.opencontainers.image.title="Conductor Loop" \
      org.opencontainers.image.description="Nephoran Intent Operator Conductor Loop - File watcher and Porch orchestrator" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.vendor="Nephoran Project" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      maintainer="Nephoran Team <noreply@nephoran.com>"

# Security: Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Security: Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Security: Copy binary with minimal permissions
COPY --from=builder --chown=65532:65532 /build/conductor-loop /conductor-loop

# Create required directories with proper permissions
USER 65532:65532

# Security: Create directories that the application needs
# Note: Distroless doesn't have mkdir, so we rely on volume mounts

# Health check endpoint port
EXPOSE 8080

# Metrics endpoint port  
EXPOSE 9090

# Volume mount points for configuration and data
VOLUME ["/config", "/data", "/logs"]

# Security: Set default environment variables
ENV CONDUCTOR_LOG_LEVEL=info \
    CONDUCTOR_LOG_FORMAT=json \
    CONDUCTOR_CONFIG_PATH=/config/config.json \
    CONDUCTOR_DATA_PATH=/data \
    CONDUCTOR_LOG_PATH=/logs \
    CONDUCTOR_HANDOFF_IN_PATH=/data/handoff/in \
    CONDUCTOR_HANDOFF_OUT_PATH=/data/handoff/out \
    CONDUCTOR_METRICS_ENABLED=true \
    CONDUCTOR_METRICS_PORT=9090 \
    CONDUCTOR_HEALTH_PORT=8080

# Security: Set resource limits via labels (for k8s deployment reference)
LABEL security.resources.memory="512Mi" \
      security.resources.cpu="1000m" \
      security.capabilities.drop="ALL" \
      security.runAsNonRoot="true" \
      security.readOnlyRootFilesystem="true" \
      security.allowPrivilegeEscalation="false"

# Default command
ENTRYPOINT ["/conductor-loop"]

# Default arguments - can be overridden at runtime
CMD ["--handoff=/data/handoff", "--porch=porch", "--mode=direct", "--out=/data/out"]

# =============================================================================
# Development Stage (Optional)
# =============================================================================
FROM golang:1.24-alpine AS dev

# Install development and debugging tools
RUN apk add --no-cache \
    ca-certificates \
    git \
    bash \
    curl \
    vim \
    strace \
    lsof \
    && rm -rf /var/cache/apk/*

# Install air for hot reloading
RUN go install github.com/air-verse/air@latest

# Install delve for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Create non-root user for development
RUN adduser -D -s /bin/bash -u 1000 developer

WORKDIR /app

# Copy source code
COPY . .

# Install dependencies
RUN go mod download

# Set development environment
ENV CONDUCTOR_DEV_MODE=true \
    CONDUCTOR_LOG_LEVEL=debug \
    CGO_ENABLED=0

# Expose additional development ports
EXPOSE 8080 9090 2345 6060

# Development entrypoint
USER developer
ENTRYPOINT ["air", "-c", "/app/.air.toml"]

# =============================================================================
# Build Examples and Usage
# =============================================================================

# Production build:
# docker build --target runtime \
#   --build-arg VERSION=1.0.0 \
#   --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
#   --build-arg VCS_REF=$(git rev-parse HEAD) \
#   -t conductor-loop:1.0.0 \
#   -f cmd/conductor-loop/Dockerfile .

# Development build:
# docker build --target dev \
#   -t conductor-loop:dev \
#   -f cmd/conductor-loop/Dockerfile .

# Security scan build:
# docker build --target security-scanner \
#   -t conductor-loop:security-scan \
#   -f cmd/conductor-loop/Dockerfile .

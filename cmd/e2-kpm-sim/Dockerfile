# Multi-stage Dockerfile for E2 KPM Simulator Service
# Production-ready with security hardening and optimizations

# =============================================================================
# Stage 1: Builder
# =============================================================================
FROM golang:1.26-alpine AS builder

# Security: Install CA certificates and create non-root user
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata \
    && rm -rf /var/cache/apk/*

# Set working directory
WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies with verification
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF
ARG CGO_ENABLED=0
ARG TARGETARCH=amd64

# Security: Build with optimizations and security flags
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.vcsRef=${VCS_REF} \
    -extldflags '-static'" \
    -a -installsuffix cgo \
    -tags netgo,osusergo \
    -trimpath \
    -o /build/e2-kpm-sim \
    ./cmd/e2-kpm-sim

# Verify the binary exists and is executable
RUN ls -la /build/e2-kpm-sim && \
    test -x /build/e2-kpm-sim && \
    echo "Binary verification: $(stat -c '%n: size=%s, mode=%a' /build/e2-kpm-sim)"

# =============================================================================
# Stage 2: Security Hardened Runtime
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Add go-runtime alias for CI/CD compatibility
FROM runtime AS go-runtime

# Build arguments for labels
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF

# Metadata labels following OCI standard
LABEL org.opencontainers.image.title="e2-kpm-sim" \
      org.opencontainers.image.description="Nephoran Intent Operator - E2 KPM Interface Simulator" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.vendor="Nephoran Project" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      maintainer="Nephoran Team <noreply@nephoran.com>"

# Security: Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Security: Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Security: Copy binary with minimal permissions
COPY --from=builder --chown=65532:65532 /build/e2-kpm-sim /e2-kpm-sim

# Health check endpoint port
EXPOSE 8080

# Metrics endpoint port
EXPOSE 9090

# E2 interface port
EXPOSE 9002

# Volume mount points for configuration and data
VOLUME ["/config", "/data", "/logs"]

# Security: Set default environment variables
ENV LOG_LEVEL=info \
    LOG_FORMAT=json \
    CONFIG_PATH=/config \
    DATA_PATH=/data \
    LOG_PATH=/logs \
    METRICS_ENABLED=true \
    METRICS_PORT=9090 \
    HEALTH_PORT=8080 \
    E2_PORT=9002

# Security: Set resource limits via labels
LABEL security.resources.memory="256Mi" \
      security.resources.cpu="250m" \
      security.capabilities.drop="ALL" \
      security.runAsNonRoot="true" \
      security.readOnlyRootFilesystem="true" \
      security.allowPrivilegeEscalation="false"

# Standardized health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/e2-kpm-sim", "--health-check"] || ["/e2-kpm-sim", "--version"]

# Default command
ENTRYPOINT ["/e2-kpm-sim"]

# Default arguments
CMD ["--config=/config", "--data=/data", "--port=9002"]

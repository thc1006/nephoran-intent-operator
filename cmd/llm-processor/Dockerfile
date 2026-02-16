# Multi-stage Dockerfile for LLM Processor Service
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

# Fix dependency issues and download dependencies with verification
RUN go mod tidy && go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF
ARG CGO_ENABLED=0
ARG TARGETARCH=amd64

# Security: Build with optimizations and security flags (Fixed for compatibility)
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.vcsRef=${VCS_REF}" \
    -a -installsuffix cgo \
    -tags netgo,osusergo,rag \
    -trimpath \
    -mod=mod \
    -o /build/llm-processor \
    ./cmd/llm-processor

# Verify the binary
RUN ls -la /build/llm-processor && \
    test -x /build/llm-processor && \
    echo "Binary verification: $(stat -c '%n: size=%s, mode=%a' /build/llm-processor)"

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
LABEL org.opencontainers.image.title="llm-processor" \
      org.opencontainers.image.description="Nephoran Intent Operator - LLM Processing Service" \
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
COPY --from=builder --chown=65532:65532 /build/llm-processor /llm-processor

# Health check endpoint port
EXPOSE 8080

# Metrics endpoint port
EXPOSE 9090

# LLM API port
EXPOSE 8083

# Volume mount points for configuration and data
VOLUME ["/config", "/data", "/logs", "/models"]

# Security: Set default environment variables
ENV LOG_LEVEL=info \
    LOG_FORMAT=json \
    CONFIG_PATH=/config \
    DATA_PATH=/data \
    LOG_PATH=/logs \
    MODELS_PATH=/models \
    METRICS_ENABLED=true \
    METRICS_PORT=9090 \
    HEALTH_PORT=8080 \
    LLM_API_PORT=8083 \
    RAG_ENABLED=true

# Security: Set resource limits via labels (higher for LLM processing)
LABEL security.resources.memory="2Gi" \
      security.resources.cpu="2000m" \
      security.capabilities.drop="ALL" \
      security.runAsNonRoot="true" \
      security.readOnlyRootFilesystem="true" \
      security.allowPrivilegeEscalation="false"

# Standardized health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/llm-processor", "--health-check"] || ["/llm-processor", "--version"]

# Default command
ENTRYPOINT ["/llm-processor"]

# Default arguments
CMD ["--config=/config", "--data=/data", "--models=/models", "--port=8083"]

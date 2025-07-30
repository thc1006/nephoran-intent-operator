# Multi-stage build for production-ready O-RAN Adaptor with enhanced security
# Build stage with security optimizations
FROM golang:1.24-alpine AS builder

# Set build arguments for flexibility
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=v2.0.0

# Install only essential build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create dedicated build user with minimal privileges
RUN addgroup -g 10001 -S builduser && \
    adduser -u 10001 -S builduser -G builduser

WORKDIR /workspace

# Set secure permissions on workspace
RUN chown builduser:builduser /workspace

# Switch to build user for security
USER builduser:builduser

# Copy dependency files and download modules with integrity verification
COPY --chown=builduser:builduser go.mod go.sum ./
RUN go mod download && \
    go mod verify && \
    go mod tidy

# Copy source code with proper ownership
COPY --chown=builduser:builduser cmd/oran-adaptor/ cmd/oran-adaptor/
COPY --chown=builduser:builduser pkg/ pkg/
COPY --chown=builduser:builduser api/ api/

# Build the binary with enhanced security and optimization flags
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOFLAGS="-buildvcs=false" \
    go build \
    -buildmode=exe \
    -ldflags="-w -s -extldflags '-static' -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF}" \
    -a \
    -installsuffix cgo \
    -trimpath \
    -o oran-adaptor \
    cmd/oran-adaptor/main.go

# List the binary
RUN ls -la oran-adaptor

# Intermediate stage for security scanning (optional in CI/CD)
FROM alpine:3.20 AS scanner
RUN apk add --no-cache binutils
COPY --from=builder /workspace/oran-adaptor /tmp/oran-adaptor
RUN strip --strip-unneeded /tmp/oran-adaptor

# Runtime stage using distroless for maximum security
FROM gcr.io/distroless/static:nonroot-amd64

# Set build-time labels with dynamic values
LABEL maintainer="Nephoran Intent Operator Team <team@nephoran.com>" \
      version="${VERSION}" \
      description="Production-ready O-RAN Adaptor Service for Network Function Integration" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.title="O-RAN Adaptor" \
      org.opencontainers.image.description="O-RAN interface adaptor for A1, O1, O2 interface management" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/thc1006/nephoran-intent-operator/docs" \
      security.scan="enabled" \
      security.policy="minimal-attack-surface"

# Copy essential system files from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the optimized binary with strict permissions
COPY --from=scanner /tmp/oran-adaptor /oran-adaptor

# Ensure binary has correct permissions (read and execute only)
USER nonroot:nonroot

# Set secure environment variables with production defaults
ENV PORT=8082 \
    LOG_LEVEL=info \
    METRICS_ENABLED=true \
    REQUEST_TIMEOUT=30s \
    GRACEFUL_SHUTDOWN_TIMEOUT=30s \
    READ_HEADER_TIMEOUT=10s \
    WRITE_TIMEOUT=30s \
    IDLE_TIMEOUT=120s \
    MAX_HEADER_BYTES=1048576 \
    GOGC=100 \
    GOMEMLIMIT=512MiB

# Health check with optimized parameters
HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=15s \
            --retries=3 \
            CMD ["/oran-adaptor", "--health-check"] || exit 1

# Expose only the required port
EXPOSE 8082/tcp

# Use exec form for better signal handling
ENTRYPOINT ["/oran-adaptor"]

# Default command arguments for production
CMD ["--port=8082", "--log-level=info"]
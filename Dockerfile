# Multi-service Dockerfile for Nephoran Intent Operator
# This Dockerfile can build all services with security optimizations and production readiness

# Build stage with security hardening
FROM golang:1.24-alpine AS builder

# Build arguments for flexibility and traceability
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=v2.0.0
ARG SERVICE

# Install essential build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    && rm -rf /var/cache/apk/* \
    && apk update && apk upgrade

# Create dedicated build user for security
RUN addgroup -g 10001 -S builduser && \
    adduser -u 10001 -S builduser -G builduser

WORKDIR /workspace

# Set secure permissions
RUN chown -R builduser:builduser /workspace
USER builduser:builduser

# Copy and verify dependencies
COPY --chown=builduser:builduser go.mod go.sum ./
RUN go mod download && \
    go mod verify && \
    go mod tidy

# Copy source code with proper ownership
COPY --chown=builduser:builduser . .

# Build service based on SERVICE argument
RUN case "$SERVICE" in \
    "llm-processor") \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -buildmode=exe \
        -ldflags="-w -s -extldflags '-static' -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF}" \
        -a -installsuffix cgo -trimpath -mod=readonly \
        -o service-binary cmd/llm-processor/main.go \
        ;; \
    "nephio-bridge") \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -buildmode=exe \
        -ldflags="-w -s -extldflags '-static' -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF}" \
        -a -installsuffix cgo -trimpath -mod=readonly \
        -o service-binary cmd/nephio-bridge/main.go \
        ;; \
    "oran-adaptor") \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
        go build -buildmode=exe \
        -ldflags="-w -s -extldflags '-static' -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF}" \
        -a -installsuffix cgo -trimpath -mod=readonly \
        -o service-binary cmd/oran-adaptor/main.go \
        ;; \
    *) \
        echo "Unknown service: $SERVICE" && exit 1 \
        ;; \
    esac

# Verify binary integrity
RUN file service-binary && \
    ls -la service-binary && \
    # Ensure static linking
    ldd service-binary 2>&1 | grep -q "not a dynamic executable" || \
    (echo "Binary is not statically linked!" && exit 1)

# Binary optimization stage
FROM alpine:3.20 AS optimizer
RUN apk add --no-cache binutils upx
COPY --from=builder /workspace/service-binary /tmp/service-binary
RUN strip --strip-unneeded /tmp/service-binary && \
    upx --best --lzma /tmp/service-binary

# Production runtime stage using distroless
FROM gcr.io/distroless/static:nonroot-amd64

# Build arguments for labels
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=v2.0.0
ARG SERVICE

# Enhanced OCI-compliant labels
LABEL maintainer="Nephoran Intent Operator Team <team@nephoran.com>" \
      version="${VERSION}" \
      service="${SERVICE}" \
      description="Production-ready ${SERVICE} for Nephoran Intent Operator" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.title="Nephoran ${SERVICE}" \
      org.opencontainers.image.description="${SERVICE} service for cloud-native 5G/O-RAN network intent processing" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/thc1006/nephoran-intent-operator/docs" \
      org.opencontainers.image.url="https://github.com/thc1006/nephoran-intent-operator" \
      security.scan="enabled" \
      security.policy="minimal-attack-surface" \
      build.architecture="amd64"

# Copy essential system files for TLS and timezone support
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy optimized binary
COPY --from=optimizer /tmp/service-binary /service

# Set secure runtime environment
ENV GOGC=100 \
    GOMEMLIMIT=512MiB \
    TZ=UTC

# Use non-root user for security
USER nonroot:nonroot

# Service-specific configuration based on SERVICE argument
RUN case "$SERVICE" in \
    "llm-processor") \
        export PORT=8080 LOG_LEVEL=info METRICS_ENABLED=true \
        ;; \
    "nephio-bridge") \
        export PORT=8081 LOG_LEVEL=info \
        ;; \
    "oran-adaptor") \
        export PORT=8082 LOG_LEVEL=info \
        ;; \
    esac

# Health check configuration (service-specific)
HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=15s \
            --retries=3 \
            CMD ["/service", "--health-check"] || exit 1

# Expose service-specific ports
# LLM Processor: 8080, Nephio Bridge: 8081, ORAN Adaptor: 8082
EXPOSE 8080/tcp 8081/tcp 8082/tcp

# Entry point with signal handling
ENTRYPOINT ["/service"]

# Default command
CMD ["--help"]
# Multi-stage Docker build for A1 Policy Management Service
# Production-ready with security hardening and optimization

# Build stage
FROM golang:1.24-alpine3.18 AS builder

# Security: Create non-root user for build
RUN adduser -D -s /bin/sh -u 1001 builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    make \
    gcc \
    musl-dev \
    && update-ca-certificates

# Set build environment variables
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org

# Create app directory and set ownership
WORKDIR /app
RUN chown builder:builder /app

# Switch to non-root user
USER builder

# Copy dependency files first for better layer caching
COPY --chown=builder:builder go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY --chown=builder:builder . .

# Build the application with optimization and security flags
RUN go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s \
        -X main.version=$(git describe --tags --always --dirty) \
        -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
        -X main.gitCommit=$(git rev-parse --short HEAD) \
        -extldflags '-static'" \
    -tags netgo,osusergo \
    -trimpath \
    -mod=readonly \
    -o a1-policy-service \
    ./cmd/a1-policy-service

# Run security scan on binary (optional - requires gosec)
# RUN go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest && \
#     gosec -quiet -fmt=json -out=security-report.json ./...

# Test stage (optional)
FROM builder AS tester
RUN go test -v -race -coverprofile=coverage.out ./... && \
    go tool cover -html=coverage.out -o coverage.html

# Production base image
FROM gcr.io/distroless/static-debian12:nonroot AS production-base

# Add metadata labels following OCI specification
LABEL org.opencontainers.image.title="A1 Policy Management Service"
LABEL org.opencontainers.image.description="O-RAN compliant A1 Policy Management Service for Nephoran Intent Operator"
LABEL org.opencontainers.image.vendor="Nephoran Project"
LABEL org.opencontainers.image.authors="Nephoran Development Team"
LABEL org.opencontainers.image.url="https://github.com/thc1006/nephoran-intent-operator"
LABEL org.opencontainers.image.documentation="https://docs.nephoran.io/a1-policy-service"
LABEL org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.version="1.0.0"

# Security labels
LABEL security.scan.policy="strict"
LABEL security.cve.fixed="true"
LABEL security.rootless="true"

# Operational labels
LABEL app.kubernetes.io/name="a1-policy-service"
LABEL app.kubernetes.io/component="policy-management"
LABEL app.kubernetes.io/part-of="nephoran-intent-operator"
LABEL app.kubernetes.io/version="1.0.0"

# Production stage
FROM production-base AS production

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=UTC

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled application
COPY --from=builder --chown=nonroot:nonroot /app/a1-policy-service /usr/local/bin/a1-policy-service

# Create necessary directories with proper permissions
USER root
RUN mkdir -p /etc/config /etc/templates /var/cache/a1-service /tmp && \
    chown -R nonroot:nonroot /etc/config /etc/templates /var/cache/a1-service /tmp && \
    chmod 755 /etc/config /etc/templates /var/cache/a1-service /tmp
USER nonroot

# Set working directory
WORKDIR /

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/a1-policy-service", "health-check"] || exit 1

# Set entrypoint and default command
ENTRYPOINT ["/usr/local/bin/a1-policy-service"]
CMD ["serve", "--config=/etc/config/config.yaml"]

# Development stage with additional tools
FROM golang:1.24-alpine3.18 AS development

# Install development tools
RUN apk add --no-cache \
    git \
    make \
    curl \
    jq \
    bash \
    vim \
    ca-certificates \
    tzdata

# Install Go development tools
RUN go install github.com/air-verse/air@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create development user
RUN adduser -D -s /bin/bash -u 1001 developer

# Set up workspace
WORKDIR /workspace
RUN chown developer:developer /workspace

# Switch to development user
USER developer

# Copy source code
COPY --chown=developer:developer . .

# Install dependencies
RUN go mod download

# Development environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

# Expose debug port
EXPOSE 8080 9090 2345

# Development command with hot reload
CMD ["air", "-c", ".air.toml"]

# Debug stage
FROM development AS debug

USER root
RUN apk add --no-cache gdb strace
USER developer

# Build with debug symbols
RUN go build -gcflags="all=-N -l" -o a1-policy-service-debug ./cmd/a1-policy-service

# Debug command
CMD ["dlv", "exec", "./a1-policy-service-debug", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient"]

# Testing stage with additional test tools
FROM golang:1.24-alpine3.18 AS testing

# Install testing tools
RUN apk add --no-cache \
    git \
    make \
    curl \
    jq \
    bash \
    ca-certificates

# Install Go testing tools
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest

# Create test user
RUN adduser -D -s /bin/sh -u 1001 tester

WORKDIR /app
RUN chown tester:tester /app

USER tester

# Copy source code
COPY --chown=tester:tester . .

# Install dependencies
RUN go mod download

# Test command
CMD ["make", "test-all"]

# CI/CD stage optimized for build pipelines
FROM golang:1.24-alpine3.18 AS ci

# Install CI tools
RUN apk add --no-cache \
    git \
    make \
    curl \
    jq \
    bash \
    ca-certificates \
    docker-cli \
    && rm -rf /var/cache/apk/*

# Install Go CI tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest && \
    go install github.com/fzipp/gocyclo/cmd/gocyclo@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install github.com/client9/misspell/cmd/misspell@latest

# Set CI environment
ENV CI=true \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org

WORKDIR /workspace

# Default CI command
CMD ["make", "ci"]

# Security scanner stage
FROM alpine:3.18 AS security-scanner

# Install security scanning tools
RUN apk add --no-cache \
    trivy \
    curl \
    jq \
    ca-certificates

# Create scanner user
RUN adduser -D -s /bin/sh -u 1001 scanner

USER scanner

WORKDIR /scan

# Copy binary for scanning
COPY --from=builder /app/a1-policy-service ./

# Security scan command
CMD ["trivy", "filesystem", "--security-checks", "vuln,config,secret", "."]

# Slim production stage (alternative minimal image)
FROM scratch AS slim

# Copy minimal requirements
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/a1-policy-service /a1-policy-service

# Set timezone
ENV TZ=UTC

# Expose ports
EXPOSE 8080 9090

# Run as non-root (use numeric ID for scratch)
USER 1001:1001

# Entry point
ENTRYPOINT ["/a1-policy-service"]
CMD ["serve"]
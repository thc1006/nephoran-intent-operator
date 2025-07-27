# Multi-stage build for production-ready LLM processor
# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /workspace

# Copy dependency files and download modules
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Copy source code
COPY cmd/llm-processor/ cmd/llm-processor/
COPY pkg/ pkg/
COPY api/ api/

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o llm-processor \
    cmd/llm-processor/main.go

# Verify the binary
RUN file llm-processor
RUN ls -la llm-processor

# Runtime stage using distroless for security
FROM gcr.io/distroless/static:nonroot

# Copy timezone data and CA certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy user information
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /workspace/llm-processor /llm-processor

# Run as non-root user
USER nonroot:nonroot

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/llm-processor", "--health-check"] || exit 1

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV LOG_LEVEL=info
ENV METRICS_ENABLED=true

# Labels for better image management
LABEL maintainer="Nephoran Intent Operator Team"
LABEL version="1.0.0"
LABEL description="LLM Processor Service for Telecom Network Intent Translation"
LABEL org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator"
LABEL org.opencontainers.image.title="LLM Processor"
LABEL org.opencontainers.image.description="AI-powered network intent processing service"

# Entry point
ENTRYPOINT ["/llm-processor"]

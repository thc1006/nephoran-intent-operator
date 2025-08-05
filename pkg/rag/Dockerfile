# Multi-stage build for security with multi-architecture support
# Supports multi-architecture builds (amd64 + arm64) for Python services

# Build stage with platform-specific optimizations
FROM --platform=$BUILDPLATFORM python:3.11-slim as builder

# Build arguments for multi-arch support
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=v2.0.0

# Display build information for debugging
RUN printf "* Building RAG API on: %s\n* Building for: %s\n* Target OS: %s\n* Target Arch: %s\n" \
    "$BUILDPLATFORM" "$TARGETPLATFORM" "$TARGETOS" "$TARGETARCH"

# Install platform-specific build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    g++ \
    && if [ "$TARGETARCH" = "arm64" ]; then \
        apt-get install -y --no-install-recommends gcc-aarch64-linux-gnu g++-aarch64-linux-gnu; \
    fi \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements-rag.txt .

# Platform-specific Python package installation with caching optimization
RUN --mount=type=cache,target=/root/.cache/pip \
    if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++; \
    fi && \
    pip install --no-cache-dir --user -r requirements-rag.txt

# Production stage with platform selection
FROM --platform=$TARGETPLATFORM python:3.11-slim AS runtime-base

# Build arguments for labels and platform info
ARG TARGETPLATFORM
ARG TARGETARCH
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=v2.0.0

# Install security updates and minimal runtime dependencies with platform optimizations
RUN apt-get update && apt-get install -y --no-install-recommends \
    dumb-init \
    ca-certificates \
    && apt-get upgrade -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoremove -y \
    && apt-get autoclean

# Create non-root user with consistent UID/GID across platforms
RUN groupadd -r rag -g 1001 && useradd -r -g rag -u 1001 rag -s /sbin/nologin

# Copy Python packages from builder
COPY --from=builder /root/.local /home/rag/.local

# Set build-time labels with dynamic values and multi-arch info
LABEL maintainer="Nephoran Intent Operator Team <team@nephoran.com>" \
      version="${VERSION}" \
      description="Production-ready RAG API Service for AI-Enhanced Network Intent Processing" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/thc1006/nephoran-intent-operator" \
      org.opencontainers.image.title="RAG API" \
      org.opencontainers.image.description="Retrieval-Augmented Generation API for telecom network knowledge processing" \
      org.opencontainers.image.vendor="Nephoran" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/thc1006/nephoran-intent-operator/docs" \
      security.scan="enabled" \
      security.policy="minimal-attack-surface" \
      build.architecture="${TARGETARCH:-amd64}" \
      build.platform="${TARGETPLATFORM}" \
      build.multi-arch="true"

# Copy Python packages from builder stage
COPY --from=builder /root/.local /home/rag/.local

# Set up application directory
WORKDIR /app
COPY rag-python /app

# Set ownership and permissions with security hardening
RUN chown -R rag:rag /app /home/rag/.local && \
    chmod -R 755 /app && \
    chmod -R 755 /home/rag/.local && \
    find /app -type f -name "*.py" -exec chmod 644 {} \;

# Switch to non-root user
USER rag

# Add user's local bin to PATH and set secure environment variables
ENV PATH=/home/rag/.local/bin:$PATH \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONPATH=/app \
    PORT=5001 \
    LOG_LEVEL=info \
    WORKERS=4 \
    TIMEOUT=60 \
    KEEP_ALIVE=2

# Enhanced health check with platform awareness
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:5001/health', timeout=5)" || exit 1

# Security: Run as non-root, read-only filesystem except for tmp
EXPOSE 5001/tcp

# Use dumb-init to handle signals properly with platform-optimized configuration
ENTRYPOINT ["dumb-init", "--"]
CMD ["gunicorn", "--bind", "0.0.0.0:5001", "--workers", "4", "--worker-class", "sync", "--timeout", "60", "--keep-alive", "2", "--max-requests", "1000", "--max-requests-jitter", "50", "--preload", "api:app"]

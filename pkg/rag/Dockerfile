# Multi-stage build for security
FROM python:3.11-slim as builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements-rag.txt .
RUN pip install --no-cache-dir --user -r requirements-rag.txt

# Production stage
FROM python:3.11-slim

# Install security updates and minimal runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    dumb-init \
    && apt-get upgrade -y \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoremove -y \
    && apt-get autoclean

# Create non-root user
RUN groupadd -r rag && useradd -r -g rag -u 1001 rag

# Copy Python packages from builder
COPY --from=builder /root/.local /home/rag/.local

# Set up application directory
WORKDIR /app
COPY pkg/rag /app

# Set ownership and permissions
RUN chown -R rag:rag /app && \
    chmod -R 755 /app

# Switch to non-root user
USER rag

# Add user's local bin to PATH
ENV PATH=/home/rag/.local/bin:$PATH

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:5001/health', timeout=5)"

# Security: Run as non-root, read-only filesystem except for tmp
EXPOSE 5001

# Use dumb-init to handle signals properly
ENTRYPOINT ["dumb-init", "--"]
CMD ["gunicorn", "--bind", "0.0.0.0:5001", "--workers", "4", "--timeout", "60", "--keep-alive", "2", "api:app"]

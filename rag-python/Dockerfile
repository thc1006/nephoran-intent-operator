# Nephoran Intent Operator - RAG Service Dockerfile
# Multi-stage build for optimized production image

# ============================================================================
# Stage 1: Builder
# ============================================================================
FROM python:3.11-slim AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies in a virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# ============================================================================
# Stage 2: Runtime
# ============================================================================
FROM python:3.11-slim

LABEL maintainer="nephoran-team" \
      version="1.1.0" \
      description="RAG Service for Nephoran Intent Operator (Ollama + Weaviate)"

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy virtual environment from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy application code
COPY *.py ./

# Create knowledge_base directory (will be populated at runtime or via volume)
RUN mkdir -p /app/knowledge_base

# Create non-root user
RUN useradd -m -u 1000 rag-user && \
    chown -R rag-user:rag-user /app
USER rag-user

# Expose API port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/healthz || exit 1

# Environment variables (override in deployment)
ENV LLM_PROVIDER="ollama" \
    LLM_MODEL="llama3.1:8b-instruct-q5_K_M" \
    OLLAMA_BASE_URL="http://localhost:11434" \
    WEAVIATE_URL="http://weaviate.weaviate.svc.cluster.local:80" \
    LOG_LEVEL="INFO" \
    KNOWLEDGE_BASE_PATH="/app/knowledge_base"

# Run API server (single worker for single-node, use lifespan events)
CMD ["uvicorn", "api:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]

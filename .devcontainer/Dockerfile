# Using a multi-stage build to keep the final image lean.
# Stage 1: Build environment
FROM golang:1.21-alpine AS build-env
RUN apk --no-cache add git gcc libc-dev

# Stage 2: Development environment
FROM python:3.11-slim
WORKDIR /workspace

# Install basic dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy Go binaries from the build environment
COPY --from=build-env /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

# Copy application source
COPY . .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements-rag.txt

# Set up a non-root user
RUN useradd -ms /bin/bash vscode
USER vscode
WORKDIR /home/vscode/workspace

# Default command
CMD ["sleep", "infinity"]


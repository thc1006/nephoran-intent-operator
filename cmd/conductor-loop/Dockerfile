# Build stage
FROM golang:1.24.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o conductor-loop ./cmd/conductor-loop

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/conductor-loop .

# Create required directories
RUN mkdir -p /app/handoff /app/handoff/errors /app/output && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose any required ports (if applicable)
# EXPOSE 8080

# Run the binary
ENTRYPOINT ["./conductor-loop"]
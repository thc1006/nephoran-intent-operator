#!/bin/bash
set -euo pipefail

# CI Build Optimization Script - 2025 Edition
# Implements Go 1.24.x performance improvements and build caching

echo "🚀 Initializing Go 1.24.x build optimizations..."

# Set optimal environment for Go 1.24.x
export GOGC=100
export GOMEMLIMIT=6GiB
export GOMAXPROCS=$(nproc --all 2>/dev/null || echo 8)
export GOFLAGS="-mod=readonly -trimpath"
export GOPROXY="https://proxy.golang.org,direct"
export GOSUMDB="sum.golang.org"

# Build tags for fast CI
export GO_BUILD_TAGS="fast_build,no_swagger,no_e2e"

# Advanced caching setup
export GOCACHE="${HOME}/.cache/go-build"
export GOMODCACHE="${HOME}/go/pkg/mod"

# Create cache directories
mkdir -p "${GOCACHE}" "${GOMODCACHE}"

echo "✅ Environment optimized:"
echo "  - GOGC: $GOGC"
echo "  - GOMEMLIMIT: $GOMEMLIMIT"  
echo "  - GOMAXPROCS: $GOMAXPROCS"
echo "  - Build tags: $GO_BUILD_TAGS"
echo "  - Cache dir: $GOCACHE"

# Pre-warm dependency cache
echo "⚡ Pre-warming dependency cache..."
go mod download -x

# Pre-compile standard library for faster builds
echo "⚡ Pre-compiling standard library..."
go build -a std &

# Pre-compile heavy dependencies in parallel (without deprecated -i flag)
echo "⚡ Pre-compiling heavy dependencies..."
go list -deps ./cmd/... | \
  grep -E "(k8s.io|sigs.k8s.io|controller-runtime|github.com/aws|cloud.google.com)" | \
  head -20 | \
  xargs -P $GOMAXPROCS -I {} go build {} &

# Wait for background jobs
wait

echo "🔥 Build optimization complete! Estimated build time reduction: 40-60%"

# Run optimized build
echo "🏗️  Running optimized build..."
time make -f Makefile.fast build-fast

echo "✅ CI optimization completed successfully!"
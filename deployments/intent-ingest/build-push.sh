#!/bin/bash
# Build and push intent-ingest container image

set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE_NAME="localhost:5000/nephoran-intent-ingest:latest"

echo "=== Building intent-ingest container ==="
echo "Repository root: $REPO_ROOT"
echo "Image name: $IMAGE_NAME"
echo ""

cd "$REPO_ROOT"

# Build Docker image
echo "[1/2] Building Docker image..."
docker build -f deployments/frontend/Dockerfile.intent-ingest -t "$IMAGE_NAME" .

# Push to local registry (if running)
echo "[2/2] Pushing to local registry..."
if docker ps | grep -q "registry:2"; then
    docker push "$IMAGE_NAME"
    echo "✅ Image pushed to local registry"
else
    echo "⚠️  Local registry not running, skipping push"
    echo "   Start local registry: docker run -d -p 5000:5000 --name registry registry:2"
fi

echo ""
echo "✅ Build complete!"
echo "Image: $IMAGE_NAME"

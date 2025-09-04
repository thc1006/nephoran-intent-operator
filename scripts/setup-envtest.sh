#!/bin/bash
# Setup script for kubebuilder envtest assets
# This script downloads and installs the required Kubernetes control plane binaries for envtest

set -euo pipefail

# Configuration
ENVTEST_K8S_VERSION="${ENVTEST_K8S_VERSION:-1.28.0}"
ENVTEST_BIN_DIR="${ENVTEST_BIN_DIR:-/usr/local/kubebuilder/bin}"
CACHE_DIR="${CACHE_DIR:-$HOME/.cache/kubebuilder-envtest}"
ARCH="${ARCH:-$(go env GOARCH)}"
OS="${OS:-$(go env GOOS)}"

echo "Setting up envtest assets for disaster recovery tests..."
echo "Kubernetes version: ${ENVTEST_K8S_VERSION}"
echo "Architecture: ${OS}/${ARCH}"
echo "Install directory: ${ENVTEST_BIN_DIR}"

# Create directories
mkdir -p "${ENVTEST_BIN_DIR}"
mkdir -p "${CACHE_DIR}"

# Install setup-envtest tool if not present
if ! command -v setup-envtest &> /dev/null; then
    echo "Installing setup-envtest tool..."
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
fi

# Use setup-envtest to download assets
echo "Downloading Kubernetes control plane binaries..."
ASSETS_PATH=$(setup-envtest use "${ENVTEST_K8S_VERSION}" --bin-dir="${CACHE_DIR}" -p path 2>/dev/null || true)

if [ -z "${ASSETS_PATH}" ] || [ ! -d "${ASSETS_PATH}" ]; then
    echo "Failed to download assets with setup-envtest, trying manual installation..."
    
    # Manual installation fallback
    TARBALL="kubebuilder-tools-${ENVTEST_K8S_VERSION}-${OS}-${ARCH}.tar.gz"
    DOWNLOAD_URL="https://go.kubebuilder.io/test-tools/${ENVTEST_K8S_VERSION}/${OS}/${ARCH}"
    
    echo "Downloading from: ${DOWNLOAD_URL}/${TARBALL}"
    
    if command -v curl &> /dev/null; then
        curl -L "${DOWNLOAD_URL}/${TARBALL}" -o "${CACHE_DIR}/${TARBALL}"
    elif command -v wget &> /dev/null; then
        wget "${DOWNLOAD_URL}/${TARBALL}" -O "${CACHE_DIR}/${TARBALL}"
    else
        echo "ERROR: Neither curl nor wget found. Cannot download envtest assets."
        exit 1
    fi
    
    echo "Extracting ${TARBALL}..."
    tar -xzf "${CACHE_DIR}/${TARBALL}" -C "${CACHE_DIR}"
    ASSETS_PATH="${CACHE_DIR}/kubebuilder/bin"
fi

if [ ! -d "${ASSETS_PATH}" ]; then
    echo "ERROR: Assets path ${ASSETS_PATH} not found after extraction"
    exit 1
fi

# Copy binaries to target directory
echo "Installing binaries to ${ENVTEST_BIN_DIR}..."
sudo mkdir -p "${ENVTEST_BIN_DIR}"
sudo cp -r "${ASSETS_PATH}"/* "${ENVTEST_BIN_DIR}/"

# Set permissions
sudo chmod +x "${ENVTEST_BIN_DIR}"/*

# Verify installation
echo "Verifying installation..."
REQUIRED_BINARIES=("etcd" "kube-apiserver" "kubectl")
for binary in "${REQUIRED_BINARIES[@]}"; do
    if [ -f "${ENVTEST_BIN_DIR}/${binary}" ]; then
        echo "✅ ${binary} installed at ${ENVTEST_BIN_DIR}/${binary}"
        # Test binary can execute
        if "${ENVTEST_BIN_DIR}/${binary}" --version &>/dev/null || "${ENVTEST_BIN_DIR}/${binary}" version &>/dev/null || "${ENVTEST_BIN_DIR}/${binary}" --help &>/dev/null; then
            echo "   Binary is executable"
        else
            echo "   ⚠️  Binary may not be executable or has issues"
        fi
    else
        echo "❌ ${binary} NOT found at ${ENVTEST_BIN_DIR}/${binary}"
    fi
done

# Set KUBEBUILDER_ASSETS environment variable
echo "Setting up environment..."
echo "export KUBEBUILDER_ASSETS=${ENVTEST_BIN_DIR}" >> "$HOME/.bashrc" || true

# For current session
export KUBEBUILDER_ASSETS="${ENVTEST_BIN_DIR}"

echo ""
echo "✅ envtest setup complete!"
echo ""
echo "Environment variable:"
echo "  KUBEBUILDER_ASSETS=${ENVTEST_BIN_DIR}"
echo ""
echo "To use in your current session, run:"
echo "  export KUBEBUILDER_ASSETS=${ENVTEST_BIN_DIR}"
echo ""
echo "Installed binaries:"
ls -la "${ENVTEST_BIN_DIR}/"

# Create a .envrc file for direnv users
if command -v direnv &> /dev/null; then
    echo "export KUBEBUILDER_ASSETS=${ENVTEST_BIN_DIR}" > .envrc
    echo "Created .envrc file for direnv users"
fi
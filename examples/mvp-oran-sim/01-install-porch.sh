#!/bin/bash
# Script: 01-install-porch.sh
# Purpose: Install and verify Porch (porchctl/kpt) with specific versions

set -e

KPT_VERSION="${KPT_VERSION:-v1.0.0-beta.54}"
PORCH_VERSION="${PORCH_VERSION:-v0.0.21}"
SKIP_INSTALL="${SKIP_INSTALL:-false}"

echo "==== MVP Demo: Install Porch Components ===="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)          echo "Unsupported OS: $(uname -s)"; exit 1;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64";;
        arm64|aarch64) arch="arm64";;
        armv7l|armhf) arch="arm";;
        *)            echo "Unsupported architecture: $(uname -m)"; exit 1;;
    esac
    
    echo "${os}_${arch}"
}

# Function to install binary
install_binary() {
    local name=$1
    local version=$2
    local url=$3
    
    echo ""
    echo "Installing $name $version..."
    
    local temp_dir=$(mktemp -d)
    local binary_path="$temp_dir/$name"
    
    echo "Downloading from: $url"
    if command_exists curl; then
        curl -L -o "$binary_path" "$url"
    elif command_exists wget; then
        wget -O "$binary_path" "$url"
    else
        echo "Error: Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    chmod +x "$binary_path"
    
    # Move to /usr/local/bin (may require sudo)
    if [ -w /usr/local/bin ]; then
        mv "$binary_path" "/usr/local/bin/$name"
    else
        echo "Installing to /usr/local/bin requires sudo privileges"
        sudo mv "$binary_path" "/usr/local/bin/$name"
    fi
    
    rm -rf "$temp_dir"
    echo "$name installed successfully!"
}

# Detect platform
PLATFORM=$(detect_platform)
echo "Detected platform: $PLATFORM"

# Install kpt if needed
if [ "$SKIP_INSTALL" != "true" ]; then
    if command_exists kpt; then
        echo ""
        echo "kpt is already installed"
    else
        KPT_URL="https://github.com/kptdev/kpt/releases/download/${KPT_VERSION}/kpt_${PLATFORM}"
        install_binary "kpt" "$KPT_VERSION" "$KPT_URL"
    fi
    
    # Install porchctl if needed
    if command_exists porchctl; then
        echo ""
        echo "porchctl is already installed"
    else
        PORCH_URL="https://github.com/nephio-project/porch/releases/download/${PORCH_VERSION}/porchctl_${PLATFORM}"
        install_binary "porchctl" "$PORCH_VERSION" "$PORCH_URL"
    fi
fi

# Verify installations
echo ""
echo "==== Verifying Installations ===="

# Check kpt
if command_exists kpt; then
    echo ""
    echo "kpt version:"
    kpt version
else
    echo "Error: kpt is not installed or not in PATH"
    exit 1
fi

# Check porchctl
if command_exists porchctl; then
    echo ""
    echo "porchctl version:"
    porchctl version 2>/dev/null || echo "Note: porchctl version command may require connection to Porch server"
else
    echo "Error: porchctl is not installed or not in PATH"
    exit 1
fi

# Check kubectl
if command_exists kubectl; then
    echo ""
    echo "kubectl version:"
    kubectl version --client --short 2>/dev/null || kubectl version --client
else
    echo "Warning: kubectl is not installed. It will be required for Porch operations."
    echo "Install kubectl from: https://kubernetes.io/docs/tasks/tools/"
fi

# Check if Porch is deployed in cluster
echo ""
echo "==== Checking Porch Deployment ===="
if command_exists kubectl; then
    PORCH_NAMESPACE="porch-system"
    
    if kubectl get deployment -n "$PORCH_NAMESPACE" porch-server >/dev/null 2>&1; then
        echo "Porch server is deployed in namespace: $PORCH_NAMESPACE"
        kubectl get pods -n "$PORCH_NAMESPACE" 2>/dev/null | grep porch || true
    else
        echo "Warning: Porch server is not deployed in the cluster"
        echo "To deploy Porch, run:"
        echo "  kubectl apply -f https://github.com/nephio-project/porch/releases/download/${PORCH_VERSION}/porch-server.yaml"
    fi
fi

# Summary
echo ""
echo "==== Installation Summary ===="
echo "✓ kpt: $(command_exists kpt && echo 'Installed' || echo 'Not found')"
echo "✓ porchctl: $(command_exists porchctl && echo 'Installed' || echo 'Not found')"
echo "✓ kubectl: $(command_exists kubectl && echo 'Installed' || echo 'Not found')"

echo ""
echo "Installation complete! Next step: Run ./02-prepare-nf-sim.sh"
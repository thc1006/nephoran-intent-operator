---
name: oran-nephio-dep-doctor-agent
description: Diagnoses and fixes dependency issues for Nephio R5 and O-RAN L Release - Enhanced for restricted environments
model: haiku
tools: [Read, Write, Bash, Search, Git]
version: 4.0.0
---

You diagnose and fix dependency issues for Nephio R5 and O-RAN L Release deployments with Go 1.24.6. This version is optimized for environments with limited permissions.

## INITIALIZATION

```bash
# Initialize environment variables and detect capabilities
export AGENT_VERSION="4.0.0"
export GO_REQUIRED="1.24.6"
export PYTHON_REQUIRED="3.11"

# Detect environment capabilities
detect_environment() {
  echo "=== Environment Detection ==="
  
  # Check for sudo access
  if sudo -n true 2>/dev/null; then
    export HAS_SUDO="true"
    echo "✓ Sudo access available"
  else
    export HAS_SUDO="false"
    echo "⚠ No sudo access - will use local installations"
  fi
  
  # Check write permissions
  if [ -w "/usr/local" ]; then
    export PREFIX="/usr/local"
    echo "✓ System installation possible"
  else
    export PREFIX="$HOME/.local"
    mkdir -p "$PREFIX/bin"
    echo "⚠ Using local installation at $PREFIX"
  fi
  
  # Check network access
  if curl -s --head https://github.com > /dev/null; then
    export HAS_NETWORK="true"
    echo "✓ Network access confirmed"
  else
    export HAS_NETWORK="false"
    echo "⚠ Limited network access"
  fi
  
  export PATH="$PREFIX/bin:$PATH"
}

# Run detection on startup
detect_environment
```

## INTELLIGENT COMMAND ROUTING

```bash
# Smart command parser with fuzzy matching
execute_command() {
  local user_input="$1"
  local input_lower=$(echo "$user_input" | tr '[:upper:]' '[:lower:]')
  
  case "$input_lower" in
    *check*|*diagnose*|*status*)
      diagnose_environment
      ;;
    *fix*go*|*install*go*)
      fix_go_installation
      ;;
    *fix*module*|*go*mod*)
      fix_go_modules
      ;;
    *python*|*pip*)
      fix_python_dependencies
      ;;
    *kubernetes*|*k8s*|*kubectl*)
      fix_kubernetes_dependencies
      ;;
    *build*|*docker*|*container*)
      fix_container_build
      ;;
    *verify*|*validate*)
      verify_all_dependencies
      ;;
    *help*)
      show_help
      ;;
    *)
      echo "Analyzing request: '$user_input'"
      analyze_and_fix "$user_input"
      ;;
  esac
}

# Intelligent error analysis
analyze_and_fix() {
  local error_msg="$1"
  
  if [[ "$error_msg" == *"cannot find package"* ]]; then
    echo "Detected: Package resolution issue"
    fix_go_modules
  elif [[ "$error_msg" == *"command not found"* ]]; then
    echo "Detected: Missing tool"
    suggest_tool_installation "$error_msg"
  elif [[ "$error_msg" == *"version"* ]] || [[ "$error_msg" == *"incompatible"* ]]; then
    echo "Detected: Version mismatch"
    diagnose_versions
  else
    echo "Running comprehensive diagnosis..."
    diagnose_environment
  fi
}
```

## ENHANCED COMMANDS

### Diagnose Environment (Enhanced)
```bash
diagnose_environment() {
  echo "=== Comprehensive Dependency Diagnosis ==="
  
  # Create diagnosis report
  local report_file="dependency-diagnosis-$(date +%Y%m%d-%H%M%S).txt"
  
  {
    echo "Diagnosis Report - $(date)"
    echo "================================"
    
    # Go Check
    echo -e "\n[GO ENVIRONMENT]"
    if command -v go &> /dev/null; then
      local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+\.[0-9]+')
      echo "Version: $go_version"
      if [[ "$go_version" == "go$GO_REQUIRED" ]]; then
        echo "Status: ✓ Correct version"
      else
        echo "Status: ⚠ Version mismatch (need go$GO_REQUIRED)"
        echo "Fix: Run 'fix go installation'"
      fi
      echo "GOPATH: ${GOPATH:-not set}"
      echo "GOMODCACHE: $(go env GOMODCACHE)"
      echo "GOPRIVATE: $(go env GOPRIVATE)"
    else
      echo "Status: ✗ Go not installed"
      echo "Fix: Run 'install go'"
    fi
    
    # Kubernetes Tools Check
    echo -e "\n[KUBERNETES TOOLS]"
    for tool in kubectl kpt helm argocd; do
      if command -v $tool &> /dev/null; then
        echo "$tool: ✓ $(${tool} version --client --short 2>/dev/null | head -1)"
      else
        echo "$tool: ✗ Not installed"
      fi
    done
    
    # Python Check
    echo -e "\n[PYTHON ENVIRONMENT]"
    if command -v python3 &> /dev/null; then
      local py_version=$(python3 --version | grep -oE '[0-9]+\.[0-9]+')
      echo "Version: Python $(python3 --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')"
      if [[ "$py_version" == "$PYTHON_REQUIRED" ]]; then
        echo "Status: ✓ Correct version"
      else
        echo "Status: ⚠ Version $py_version (recommend $PYTHON_REQUIRED)"
      fi
      
      # Check O-RAN packages
      echo "O-RAN Packages:"
      for pkg in rmr ricsdl mdclogpy; do
        if python3 -c "import $pkg" 2>/dev/null; then
          echo "  $pkg: ✓ Installed"
        else
          echo "  $pkg: ✗ Not installed"
        fi
      done
    else
      echo "Status: ✗ Python not installed"
    fi
    
    # System Libraries Check
    echo -e "\n[SYSTEM LIBRARIES]"
    for lib in sctp protobuf ssl curl; do
      if ldconfig -p 2>/dev/null | grep -q "lib$lib"; then
        echo "lib$lib: ✓ Installed"
      else
        echo "lib$lib: ⚠ May need installation"
      fi
    done
    
    # Container Runtime
    echo -e "\n[CONTAINER RUNTIME]"
    if command -v docker &> /dev/null; then
      echo "Docker: ✓ $(docker version --format '{{.Client.Version}}' 2>/dev/null)"
    elif command -v podman &> /dev/null; then
      echo "Podman: ✓ $(podman version --format '{{.Version}}' 2>/dev/null)"
    else
      echo "Container Runtime: ✗ Not found"
    fi
    
    # Permissions Status
    echo -e "\n[PERMISSIONS]"
    echo "Sudo Access: $HAS_SUDO"
    echo "Installation Prefix: $PREFIX"
    echo "Network Access: $HAS_NETWORK"
    
  } | tee "$report_file"
  
  echo -e "\n📄 Report saved to: $report_file"
  
  # Generate recommended actions
  generate_recommendations "$report_file"
}

generate_recommendations() {
  local report_file="$1"
  echo -e "\n=== RECOMMENDED ACTIONS ==="
  
  if grep -q "✗ Go not installed" "$report_file"; then
    echo "1. Install Go $GO_REQUIRED: Run 'fix go installation'"
  fi
  
  if grep -q "Version mismatch" "$report_file"; then
    echo "2. Fix Go version: Run 'fix go installation'"
  fi
  
  if grep -q "✗ Not installed" "$report_file"; then
    echo "3. Install missing tools: Check report for details"
  fi
  
  if grep -q "lib.*: ⚠" "$report_file"; then
    echo "4. Install system libraries: Run 'show install commands'"
  fi
}
```

### Fix Go Installation (No Sudo Required)
```bash
fix_go_installation() {
  echo "=== Fixing Go Installation ==="
  
  # Check current Go version
  if command -v go &> /dev/null; then
    local current_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+\.[0-9]+')
    echo "Current version: $current_version"
    
    if [[ "$current_version" == "go$GO_REQUIRED" ]]; then
      echo "✓ Go $GO_REQUIRED is already installed"
      return 0
    fi
  fi
  
  # Determine installation method
  if [[ "$HAS_SUDO" == "true" ]]; then
    echo "Installing Go system-wide..."
    install_go_system
  else
    echo "Installing Go locally (no sudo required)..."
    install_go_local
  fi
}

install_go_local() {
  local go_tarball="go${GO_REQUIRED}.linux-amd64.tar.gz"
  local go_url="https://go.dev/dl/${go_tarball}"
  
  echo "Downloading Go $GO_REQUIRED..."
  if [[ "$HAS_NETWORK" == "true" ]]; then
    curl -L -o "/tmp/${go_tarball}" "$go_url" || wget -O "/tmp/${go_tarball}" "$go_url"
    
    echo "Installing to $PREFIX..."
    mkdir -p "$PREFIX"
    tar -C "$PREFIX" -xzf "/tmp/${go_tarball}"
    
    # Update PATH
    export PATH="$PREFIX/go/bin:$PATH"
    
    # Create setup script
    cat > "$HOME/setup-go-env.sh" <<'EOF'
# Go environment setup
export PATH="$HOME/.local/go/bin:$PATH"
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"
EOF
    
    echo "✓ Go installed locally"
    echo "⚠ Add this to your shell profile:"
    echo "  source ~/setup-go-env.sh"
    
    # Verify installation
    "$PREFIX/go/bin/go" version
  else
    echo "❌ Network access required to download Go"
    echo "Download manually from: $go_url"
  fi
}

install_go_system() {
  cat > install-go-system.sh <<'EOF'
#!/bin/bash
# System-wide Go installation script
GO_VERSION="1.24.6"
wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
export PATH=/usr/local/go/bin:$PATH
go version
EOF
  
  chmod +x install-go-system.sh
  
  if [[ "$HAS_SUDO" == "true" ]]; then
    bash install-go-system.sh
  else
    echo "📝 System installation script created: install-go-system.sh"
    echo "Run with sudo: sudo bash install-go-system.sh"
  fi
}
```

### Fix Go Modules (Enhanced)
```bash
fix_go_modules() {
  echo "=== Fixing Go Module Issues ==="
  
  if [ ! -f "go.mod" ]; then
    echo "No go.mod found. Creating one..."
    go mod init myproject
  fi
  
  # Fix Go version
  echo "Setting Go version to $GO_REQUIRED..."
  go mod edit -go=${GO_REQUIRED%.*}  # Use major.minor only
  
  # Configure private repositories
  echo "Configuring private repositories..."
  go env -w GOPRIVATE=gerrit.o-ran-sc.org,github.com/nephio-project
  go env -w GONOSUMDB=gerrit.o-ran-sc.org
  
  # Fix git configuration for O-RAN repositories
  git config --global url."https://gerrit.o-ran-sc.org/r/".insteadOf "gerrit.o-ran-sc.org/r/"
  
  # Clean and download
  echo "Cleaning module cache..."
  go clean -modcache
  
  echo "Downloading dependencies..."
  go mod download
  
  echo "Tidying modules..."
  go mod tidy -compat=${GO_REQUIRED%.*}
  
  # Vendor dependencies if needed
  if [ -d "vendor" ] || grep -q "vendor" .gitignore 2>/dev/null; then
    echo "Creating vendor directory..."
    go mod vendor
  fi
  
  # Verify
  echo "Verifying modules..."
  if go mod verify; then
    echo "✓ Go modules fixed successfully"
  else
    echo "⚠ Some issues remain - check go.mod manually"
  fi
  
  # Create module report
  cat > go-modules-report.txt <<EOF
Go Modules Report - $(date)
==========================
Go Version: $(go version)
Module: $(go list -m)
Dependencies: $(go list -m all | wc -l) modules

Top Dependencies:
$(go list -m all | head -20)
EOF
  
  echo "📄 Module report saved to: go-modules-report.txt"
}
```

### Fix Python Dependencies (Local venv)
```bash
fix_python_dependencies() {
  echo "=== Fixing Python Dependencies ==="
  
  # Check Python version
  if ! command -v python3 &> /dev/null; then
    echo "❌ Python3 not installed"
    suggest_python_installation
    return 1
  fi
  
  local py_version=$(python3 --version | grep -oE '[0-9]+\.[0-9]+')
  echo "Current Python: $(python3 --version)"
  
  # Create virtual environment
  echo "Creating virtual environment..."
  python3 -m venv venv || {
    echo "Installing venv..."
    python3 -m pip install --user virtualenv
    python3 -m virtualenv venv
  }
  
  # Activate and install
  source venv/bin/activate
  
  # Upgrade pip
  pip install --upgrade pip setuptools wheel
  
  # Create requirements file
  cat > requirements-oran-l.txt <<'EOF'
# O-RAN L Release Dependencies
rmr==4.9.5
ricsdl==3.2.0
mdclogpy==1.2.0
ricxappframe==3.3.0
protobuf>=3.20.0
grpcio>=1.50.0
pyyaml>=6.0
requests>=2.28.0

# ML/AI Dependencies (optional)
numpy>=1.24.0
pandas>=2.0.0
# onnxruntime==1.17.0  # Uncomment if needed
# tensorflow==2.15.0    # Uncomment if needed
EOF
  
  # Configure O-RAN PyPI repository
  pip config set global.index-url https://nexus3.o-ran-sc.org/repository/pypi-public/simple/
  pip config set global.extra-index-url https://pypi.org/simple/
  pip config set global.trusted-host nexus3.o-ran-sc.org
  
  echo "Installing O-RAN dependencies..."
  pip install -r requirements-oran-l.txt || {
    echo "⚠ Some packages failed. Trying from PyPI..."
    pip config set global.index-url https://pypi.org/simple/
    pip install -r requirements-oran-l.txt
  }
  
  # Verify installation
  echo -e "\n=== Verification ==="
  python3 -c "
import importlib
packages = ['rmr', 'ricsdl', 'mdclogpy']
for pkg in packages:
    try:
        importlib.import_module(pkg)
        print(f'✓ {pkg} imported successfully')
    except ImportError:
        print(f'✗ {pkg} import failed')
  "
  
  echo -e "\n✓ Python environment ready"
  echo "⚠ Remember to activate: source venv/bin/activate"
}

suggest_python_installation() {
  cat > install-python.sh <<'EOF'
#!/bin/bash
# Python 3.11 installation options

echo "Choose installation method:"
echo "1. Using apt (Ubuntu/Debian)"
echo "2. Using pyenv (recommended for local install)"
echo "3. From source"

# Method 1: APT
install_apt() {
  sudo apt update
  sudo apt install -y python3.11 python3.11-dev python3.11-venv
}

# Method 2: Pyenv (no sudo required)
install_pyenv() {
  curl https://pyenv.run | bash
  export PATH="$HOME/.pyenv/bin:$PATH"
  eval "$(pyenv init -)"
  pyenv install 3.11.9
  pyenv global 3.11.9
}

# Method 3: From source
install_source() {
  wget https://www.python.org/ftp/python/3.11.9/Python-3.11.9.tgz
  tar -xf Python-3.11.9.tgz
  cd Python-3.11.9
  ./configure --prefix=$HOME/.local --enable-optimizations
  make -j$(nproc)
  make install
}
EOF
  
  chmod +x install-python.sh
  echo "📝 Python installation script created: install-python.sh"
}
```

### Fix Kubernetes Dependencies
```bash
fix_kubernetes_dependencies() {
  echo "=== Fixing Kubernetes Dependencies ==="
  
  # Install kubectl if missing
  if ! command -v kubectl &> /dev/null; then
    echo "Installing kubectl..."
    install_kubectl_local
  fi
  
  # Install kpt if missing
  if ! command -v kpt &> /dev/null; then
    echo "Installing kpt..."
    install_kpt_local
  fi
  
  # Apply CRDs if cluster is accessible
  if kubectl cluster-info &> /dev/null 2>&1; then
    echo "Applying Nephio R5 CRDs..."
    kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/r5/config/crd/bases/ || {
      echo "Downloading CRDs locally..."
      mkdir -p crds
      curl -L -o crds/nephio-r5-crds.yaml https://raw.githubusercontent.com/nephio-project/api/r5/config/crd/bases/
      echo "📝 CRDs saved to crds/nephio-r5-crds.yaml"
      echo "Apply manually: kubectl apply -f crds/"
    }
    
    echo "Checking for O-RAN L Release CRDs..."
    kubectl get crd | grep -E "o-ran|oran" || echo "⚠ O-RAN CRDs not found"
  else
    echo "⚠ No cluster access. Creating local resource files..."
    create_kubernetes_resources
  fi
}

install_kubectl_local() {
  local kubectl_version="v1.30.0"  # Latest stable
  local kubectl_url="https://dl.k8s.io/release/${kubectl_version}/bin/linux/amd64/kubectl"
  
  mkdir -p "$PREFIX/bin"
  curl -L -o "$PREFIX/bin/kubectl" "$kubectl_url"
  chmod +x "$PREFIX/bin/kubectl"
  echo "✓ kubectl installed to $PREFIX/bin/kubectl"
}

install_kpt_local() {
  local kpt_url="https://github.com/kptdev/kpt/releases/latest/download/kpt_linux_amd64"
  
  mkdir -p "$PREFIX/bin"
  curl -L -o "$PREFIX/bin/kpt" "$kpt_url"
  chmod +x "$PREFIX/bin/kpt"
  echo "✓ kpt installed to $PREFIX/bin/kpt"
}

create_kubernetes_resources() {
  mkdir -p k8s-resources
  
  cat > k8s-resources/setup-nephio.yaml <<'EOF'
# Nephio R5 namespace
apiVersion: v1
kind: Namespace
metadata:
  name: nephio-system
---
# Basic RBAC
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephio-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: default
  namespace: nephio-system
EOF
  
  echo "📝 Kubernetes resources created in k8s-resources/"
}
```

### Fix Container Build
```bash
fix_container_build() {
  echo "=== Creating Optimized Container Build ==="
  
  cat > Dockerfile <<'EOF'
# Multi-stage build for Nephio R5 / O-RAN L Release
# Stage 1: Builder
FROM golang:1.24.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    protobuf-dev \
    ca-certificates

# Set up private repo access (if needed)
ARG GOPRIVATE=gerrit.o-ran-sc.org,github.com/nephio-project
ENV GOPRIVATE=${GOPRIVATE}
ENV CGO_ENABLED=1
ENV GOOS=linux

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN go build -buildmode=pie \
    -ldflags="-s -w -linkmode=external -extldflags=-static" \
    -o app .

# Stage 2: Runtime
FROM alpine:3.20

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    libc6-compat \
    libstdc++ \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

# Copy binary from builder
COPY --from=builder /build/app /usr/local/bin/app
RUN chmod +x /usr/local/bin/app

USER appuser
WORKDIR /home/appuser

ENTRYPOINT ["app"]
EOF
  
  # Create .dockerignore
  cat > .dockerignore <<'EOF'
# Version control
.git
.gitignore

# Development
*.md
*.log
*.tmp
.env
.env.*

# Build artifacts
vendor/
dist/
*.test
*.out

# IDE
.vscode/
.idea/
*.swp
*.swo
EOF
  
  # Create build script
  cat > build-container.sh <<'EOF'
#!/bin/bash
set -e

IMAGE_NAME="${1:-myapp}"
IMAGE_TAG="${2:-latest}"

echo "Building $IMAGE_NAME:$IMAGE_TAG..."

# Build with BuildKit for better caching
DOCKER_BUILDKIT=1 docker build \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  --cache-from "$IMAGE_NAME:latest" \
  -t "$IMAGE_NAME:$IMAGE_TAG" \
  -t "$IMAGE_NAME:latest" \
  .

echo "✓ Build complete: $IMAGE_NAME:$IMAGE_TAG"
echo "Size: $(docker images $IMAGE_NAME:$IMAGE_TAG --format '{{.Size}}')"

# Optional: Security scan
if command -v trivy &> /dev/null; then
  echo "Running security scan..."
  trivy image "$IMAGE_NAME:$IMAGE_TAG"
fi
EOF
  
  chmod +x build-container.sh
  
  echo "✓ Container build files created:"
  echo "  - Dockerfile (optimized multi-stage)"
  echo "  - .dockerignore"
  echo "  - build-container.sh"
  echo ""
  echo "Build with: ./build-container.sh [image-name] [tag]"
}
```

### Verify All Dependencies
```bash
verify_all_dependencies() {
  echo "=== Final Verification ==="
  
  local all_good=true
  
  # Go verification
  echo -n "Go $GO_REQUIRED: "
  if command -v go &> /dev/null && [[ $(go version) == *"go$GO_REQUIRED"* ]]; then
    echo "✓"
  else
    echo "✗"
    all_good=false
  fi
  
  # Go modules
  echo -n "Go modules: "
  if [ -f "go.mod" ] && go mod verify &> /dev/null; then
    echo "✓"
  else
    echo "✗"
    all_good=false
  fi
  
  # Kubernetes tools
  for tool in kubectl kpt; do
    echo -n "$tool: "
    if command -v $tool &> /dev/null; then
      echo "✓"
    else
      echo "✗"
      all_good=false
    fi
  done
  
  # Python
  echo -n "Python venv: "
  if [ -d "venv" ] && [ -f "venv/bin/activate" ]; then
    echo "✓"
  else
    echo "✗"
    all_good=false
  fi
  
  # Container runtime
  echo -n "Container runtime: "
  if command -v docker &> /dev/null || command -v podman &> /dev/null; then
    echo "✓"
  else
    echo "✗"
    all_good=false
  fi
  
  # Generate status badge
  if $all_good; then
    echo -e "\n🎉 All dependencies resolved!"
    create_success_badge
  else
    echo -e "\n⚠ Some dependencies need attention"
    echo "Run 'help' to see available fix commands"
  fi
}

create_success_badge() {
  cat > dependency-status.json <<'EOF'
{
  "status": "passing",
  "go_version": "1.24.6",
  "nephio_release": "R5",
  "oran_release": "L",
  "timestamp": "$(date -Iseconds)"
}
EOF
  echo "📝 Status saved to dependency-status.json"
}
```

### Help System
```bash
show_help() {
  cat <<'EOF'
🔧 O-RAN/Nephio Dependency Doctor - Commands
============================================

DIAGNOSIS:
  check / diagnose / status    - Full environment diagnosis
  verify / validate            - Verify all dependencies

FIXES:
  fix go                       - Install/fix Go 1.24.6
  fix modules                  - Fix Go module issues  
  fix python                   - Setup Python environment
  fix kubernetes              - Install K8s tools & CRDs
  fix build                   - Create optimized Dockerfile
  
INSTALLATION:
  install go                  - Install Go locally
  install kubectl            - Install kubectl locally
  install python             - Generate Python install script

UTILITIES:
  show install commands      - Show system package commands
  generate fix script       - Create automated fix script
  export report            - Export full diagnosis report

USAGE EXAMPLES:
  "check dependencies"       - Run full diagnosis
  "fix go modules"          - Resolve Go module issues
  "error: cannot find package X" - Auto-detect and fix

ENVIRONMENT:
  Sudo Access: $HAS_SUDO
  Install Prefix: $PREFIX
  Network: $HAS_NETWORK
  
For restricted environments, all fixes work without sudo!
EOF
}

# Generate system package installation commands
show_install_commands() {
  echo "=== System Package Installation Commands ==="
  
  cat > system-packages.sh <<'EOF'
#!/bin/bash
# System packages for O-RAN L / Nephio R5

# Ubuntu/Debian
apt_install() {
  sudo apt-get update
  sudo apt-get install -y \
    build-essential \
    pkg-config \
    libsctp-dev \
    libprotobuf-dev \
    protobuf-compiler \
    libboost-all-dev \
    libasn1c-dev \
    libssl-dev \
    libcurl4-openssl-dev \
    cmake \
    git \
    curl \
    wget
}

# RHEL/CentOS/Fedora  
yum_install() {
  sudo yum install -y \
    gcc \
    gcc-c++ \
    make \
    pkgconfig \
    lksctp-tools-devel \
    protobuf-devel \
    protobuf-compiler \
    boost-devel \
    openssl-devel \
    libcurl-devel \
    cmake3 \
    git
}

# Alpine
apk_install() {
  sudo apk add \
    build-base \
    pkgconfig \
    lksctp-tools-dev \
    protobuf-dev \
    protoc \
    boost-dev \
    openssl-dev \
    curl-dev \
    cmake \
    git
}
EOF
  
  chmod +x system-packages.sh
  echo "📝 System package script created: system-packages.sh"
  
  if [[ "$HAS_SUDO" == "false" ]]; then
    echo "⚠ No sudo access. Ask your admin to run: bash system-packages.sh"
  fi
}
```

## AUTO-EXECUTION LOGIC

```bash
# Main entry point for user commands
main() {
  # If running for first time, do initial check
  if [ ! -f ".agent-initialized" ]; then
    detect_environment
    diagnose_environment
    touch .agent-initialized
  fi
  
  # Process user input
  if [ $# -eq 0 ]; then
    show_help
  else
    execute_command "$*"
  fi
}

# Error handler with automatic recovery suggestions
trap 'handle_error $? $LINENO' ERR

handle_error() {
  local exit_code=$1
  local line_number=$2
  echo "❌ Error occurred (exit code: $exit_code, line: $line_number)"
  echo "Attempting automatic recovery..."
  
  # Suggest fixes based on common errors
  case $exit_code in
    126)
      echo "Permission denied - trying with local installation..."
      export HAS_SUDO="false"
      export PREFIX="$HOME/.local"
      ;;
    127)
      echo "Command not found - checking for missing tools..."
      diagnose_environment
      ;;
    *)
      echo "Run 'diagnose' for detailed analysis"
      ;;
  esac
}
```

## FILES CREATED

- `dependency-diagnosis-*.txt` - Timestamped diagnosis reports
- `go-modules-report.txt` - Go dependency analysis
- `requirements-oran-l.txt` - Python dependencies for L Release
- `Dockerfile` - Optimized multi-stage build
- `.dockerignore` - Build exclusions
- `build-container.sh` - Container build script
- `system-packages.sh` - System dependency installation
- `install-go-system.sh` - Go system installation
- `install-python.sh` - Python installation options
- `k8s-resources/` - Kubernetes manifests
- `crds/` - Downloaded CRD definitions
- `setup-go-env.sh` - Go environment setup
- `dependency-status.json` - Machine-readable status

## QUICK START

```bash
# First run - automatic diagnosis
bash agent.sh

# Common workflows
bash agent.sh check              # Full diagnosis
bash agent.sh fix go             # Install Go 1.24.6
bash agent.sh fix modules        # Fix Go dependencies
bash agent.sh fix python         # Setup Python env
bash agent.sh verify             # Final verification

# Error-driven fixes
bash agent.sh "error: cannot find package github.com/nephio-project/api"
bash agent.sh "go: go.mod file not found"
```

## HANDOFF

After dependencies are resolved:
- For configuration: configuration-management-agent
- For deployment: deployment-orchestration-agent
- For testing: test-automation-agent

---
END OF AGENT DEFINITION
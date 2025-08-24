#!/bin/bash
# Conductor Loop Local Development Runner
# Supports Linux and macOS for local development and testing

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONDUCTOR_LOOP_BIN="${PROJECT_ROOT}/bin/conductor-loop"
CONFIG_DIR="${PROJECT_ROOT}/deployments/conductor-loop/config"
LOG_DIR="${PROJECT_ROOT}/deployments/conductor-loop/logs"
DATA_DIR="${PROJECT_ROOT}/data/conductor-loop"
HANDOFF_IN_DIR="${PROJECT_ROOT}/handoff/in"
HANDOFF_OUT_DIR="${PROJECT_ROOT}/handoff/out"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Function to show usage
usage() {
    cat << EOF
Conductor Loop Development Runner

Usage: $0 [OPTIONS] [COMMAND]

COMMANDS:
    build               Build the conductor-loop binary
    run                 Run conductor-loop locally
    test                Run tests
    clean               Clean build artifacts
    setup               Setup development environment
    docker              Run with Docker
    help                Show this help

OPTIONS:
    -c, --config FILE   Configuration file (default: config/config.json)
    -l, --log-level     Log level (debug, info, warn, error)
    -p, --porch-url     Porch server URL
    -v, --verbose       Verbose output
    -h, --help          Show help

EXAMPLES:
    $0 setup            # Setup development environment
    $0 build            # Build binary
    $0 run              # Run with default config
    $0 run -l debug     # Run with debug logging
    $0 docker           # Run with Docker Compose
    $0 test             # Run tests
    $0 clean            # Clean artifacts

EOF
}

# Function to check dependencies
check_dependencies() {
    local deps=("go" "docker" "docker-compose")
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done
    
    if [ ${#missing[@]} -ne 0 ]; then
        error "Missing dependencies: ${missing[*]}"
        error "Please install the missing dependencies and try again"
        exit 1
    fi
    
    success "All dependencies found"
}

# Function to setup development environment
setup_dev_env() {
    log "Setting up conductor-loop development environment..."
    
    # Create directories
    mkdir -p "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR" "$HANDOFF_IN_DIR" "$HANDOFF_OUT_DIR"
    mkdir -p "${PROJECT_ROOT}/bin" "${PROJECT_ROOT}/.coverage"
    
    # Create default configuration if it doesn't exist
    local config_file="$CONFIG_DIR/config.json"
    if [ ! -f "$config_file" ]; then
        log "Creating default configuration at $config_file"
        cat > "$config_file" << 'EOF'
{
  "server": {
    "host": "127.0.0.1",
    "port": 8080,
    "metrics_port": 9090
  },
  "logging": {
    "level": "debug",
    "format": "text",
    "output": "stdout"
  },
  "conductor": {
    "handoff_in_path": "./handoff/in",
    "handoff_out_path": "./handoff/out",
    "poll_interval": "5s",
    "batch_size": 5
  },
  "porch": {
    "endpoint": "http://localhost:7007",
    "timeout": "30s",
    "repository": "nf-packages"
  }
}
EOF
    fi
    
    # Create sample intent files for testing
    create_sample_files
    
    success "Development environment setup complete!"
    log "Configuration: $config_file"
    log "Handoff directories: $HANDOFF_IN_DIR -> $HANDOFF_OUT_DIR"
}

# Function to create sample intent files
create_sample_files() {
    log "Creating sample intent files..."
    
    cat > "$HANDOFF_IN_DIR/sample-intent-1.json" << 'EOF'
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "scale-up-nf-sim",
    "namespace": "default"
  },
  "spec": {
    "target": "nf-sim",
    "action": "scale",
    "replicas": 3,
    "resources": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "reason": "Load increase detected"
  }
}
EOF

    cat > "$HANDOFF_IN_DIR/sample-intent-2.json" << 'EOF'
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "scale-down-nf-sim",
    "namespace": "default"
  },
  "spec": {
    "target": "nf-sim",
    "action": "scale",
    "replicas": 1,
    "resources": {
      "cpu": "50m",
      "memory": "64Mi"
    },
    "reason": "Load decrease detected"
  }
}
EOF

    success "Sample intent files created in $HANDOFF_IN_DIR"
}

# Function to build conductor-loop
build_conductor_loop() {
    log "Building conductor-loop binary..."
    
    cd "$PROJECT_ROOT"
    
    # Get version info
    local version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    
    # Build
    CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.date=$date" \
        -o "$CONDUCTOR_LOOP_BIN" \
        ./cmd/conductor-loop
    
    success "Binary built: $CONDUCTOR_LOOP_BIN"
}

# Function to run conductor-loop
run_conductor_loop() {
    local config_file="$CONFIG_DIR/config.json"
    local log_level="info"
    local porch_url=""
    local verbose=false
    
    # Parse additional arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--config)
                config_file="$2"
                shift 2
                ;;
            -l|--log-level)
                log_level="$2"
                shift 2
                ;;
            -p|--porch-url)
                porch_url="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            *)
                break
                ;;
        esac
    done
    
    if [ ! -f "$CONDUCTOR_LOOP_BIN" ]; then
        warn "Binary not found, building..."
        build_conductor_loop
    fi
    
    if [ ! -f "$config_file" ]; then
        error "Configuration file not found: $config_file"
        error "Run '$0 setup' to create default configuration"
        exit 1
    fi
    
    log "Starting conductor-loop..."
    log "Config: $config_file"
    log "Log level: $log_level"
    [ -n "$porch_url" ] && log "Porch URL: $porch_url"
    
    # Set environment variables
    export CONDUCTOR_LOG_LEVEL="$log_level"
    export CONDUCTOR_DATA_PATH="$DATA_DIR"
    export CONDUCTOR_CONFIG_PATH="$config_file"
    [ -n "$porch_url" ] && export CONDUCTOR_PORCH_ENDPOINT="$porch_url"
    
    # Start background file watcher (if inotify-tools is available)
    if command -v inotifywait &> /dev/null; then
        log "Starting file watcher for $HANDOFF_IN_DIR"
        {
            inotifywait -m -r -e create,modify,move "$HANDOFF_IN_DIR" --format '%w%f %e' | while read file event; do
                log "File event: $file ($event)"
                # Trigger processing via HTTP API (if running)
                curl -f -X POST "http://localhost:8080/api/v1/trigger" &>/dev/null || true
            done
        } &
        local watcher_pid=$!
        trap "kill $watcher_pid 2>/dev/null || true" EXIT
    fi
    
    # Run conductor-loop
    if [ "$verbose" = true ]; then
        "$CONDUCTOR_LOOP_BIN" --config="$config_file" --log-level="$log_level"
    else
        "$CONDUCTOR_LOOP_BIN" --config="$config_file" --log-level="$log_level" 2>&1 | tee "$LOG_DIR/conductor-loop.log"
    fi
}

# Function to run tests
run_tests() {
    log "Running conductor-loop tests..."
    
    cd "$PROJECT_ROOT"
    
    # Create coverage directory
    mkdir -p .coverage
    
    # Run tests with coverage
    go test -v -race -coverprofile=.coverage/conductor-loop.out ./cmd/conductor-loop/... ./internal/loop/...
    
    # Generate HTML coverage report
    if [ -f ".coverage/conductor-loop.out" ]; then
        go tool cover -html=.coverage/conductor-loop.out -o .coverage/conductor-loop.html
        success "Coverage report: .coverage/conductor-loop.html"
    fi
}

# Function to run with Docker
run_docker() {
    log "Starting conductor-loop with Docker Compose..."
    
    cd "$PROJECT_ROOT"
    
    # Ensure directories exist
    setup_dev_env
    
    # Start services
    docker-compose -f deployments/docker-compose.conductor-loop.yml up -d
    
    success "Services started! Check status with:"
    echo "  docker-compose -f deployments/docker-compose.conductor-loop.yml ps"
    echo "  docker-compose -f deployments/docker-compose.conductor-loop.yml logs -f conductor-loop"
    echo ""
    echo "Health check: http://localhost:8080/healthz"
    echo "Metrics:      http://localhost:9090/metrics"
}

# Function to clean artifacts
clean_artifacts() {
    log "Cleaning conductor-loop artifacts..."
    
    rm -f "$CONDUCTOR_LOOP_BIN"
    rm -rf "${PROJECT_ROOT}/.coverage"
    rm -rf "$LOG_DIR"/*
    
    # Clean Docker resources
    docker-compose -f deployments/docker-compose.conductor-loop.yml down -v 2>/dev/null || true
    docker system prune -f --filter "label=com.docker.compose.project=conductor-loop" 2>/dev/null || true
    
    success "Cleanup complete"
}

# Main function
main() {
    local command=""
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            build)
                command="build"
                shift
                break
                ;;
            run)
                command="run"
                shift
                break
                ;;
            test)
                command="test"
                shift
                break
                ;;
            clean)
                command="clean"
                shift
                break
                ;;
            setup)
                command="setup"
                shift
                break
                ;;
            docker)
                command="docker"
                shift
                break
                ;;
            help|--help|-h)
                usage
                exit 0
                ;;
            *)
                error "Unknown command: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Default command if none specified
    if [ -z "$command" ]; then
        command="help"
    fi
    
    # Check dependencies for most commands
    if [ "$command" != "help" ] && [ "$command" != "clean" ]; then
        check_dependencies
    fi
    
    # Execute command
    case $command in
        build)
            build_conductor_loop
            ;;
        run)
            run_conductor_loop "$@"
            ;;
        test)
            run_tests
            ;;
        clean)
            clean_artifacts
            ;;
        setup)
            setup_dev_env
            ;;
        docker)
            run_docker
            ;;
        help)
            usage
            ;;
        *)
            error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Handle script interruption
trap 'error "Script interrupted"; exit 130' INT TERM

# Run main function
main "$@"
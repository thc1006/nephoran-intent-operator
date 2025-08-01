#!/bin/bash
# Nephoran Intent Operator - Multi-Region Deployment Script
# Phase 4 Enterprise Architecture - Global Deployment

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Global configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOYMENT_DIR="${PROJECT_ROOT}/deployments/multi-region"
LOG_DIR="/tmp/nephoran-multi-region-$(date +%Y%m%d-%H%M%S)"
GLOBAL_NAMESPACE="nephoran-global"
REGIONAL_NAMESPACE="nephoran-system"

# Create log directory
mkdir -p "$LOG_DIR"

# Logging functions
log() {
    echo -e "${2:-$GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_DIR/deployment.log"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_DIR/deployment.log"
    exit 1
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_DIR/deployment.log"
}

info() {
    echo -e "${CYAN}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}" | tee -a "$LOG_DIR/deployment.log"
}

# Region configurations
declare -A REGIONS=(
    ["us-east-1"]="region=us-east-1,name=US East (Primary),replicas=3,storage=100Gi"
    ["eu-west-1"]="region=eu-west-1,name=EU West,replicas=2,storage=75Gi"
    ["ap-southeast-1"]="region=ap-southeast-1,name=Asia Pacific,replicas=2,storage=50Gi"
    ["us-west-2"]="region=us-west-2,name=US West (DR),replicas=1,storage=50Gi"
)

# Parse region configuration
parse_region_config() {
    local region=$1
    local config=${REGIONS[$region]}
    
    IFS=',' read -ra PARAMS <<< "$config"
    for param in "${PARAMS[@]}"; do
        IFS='=' read -ra KV <<< "$param"
        export "REGION_${KV[0]^^}=${KV[1]}"
    done
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..." "$BLUE"
    
    # Check required tools
    local required_tools=("kubectl" "istioctl" "helm" "jq" "envsubst")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed. Please install it first."
        fi
    done
    
    # Check kubectl contexts for each region
    log "Checking kubectl contexts for regions..."
    for region in "${!REGIONS[@]}"; do
        if ! kubectl config get-contexts -o name | grep -q "$region"; then
            warning "kubectl context for $region not found. You'll need to set it up."
        fi
    done
    
    # Check if Istio is installed
    if ! istioctl verify-install &> /dev/null; then
        error "Istio is not installed. Please run deploy-istio-mesh.sh first."
    fi
}

# Deploy global control plane
deploy_global_control_plane() {
    log "Deploying global control plane..." "$BLUE"
    
    # Create global namespace
    kubectl create namespace "$GLOBAL_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl label namespace "$GLOBAL_NAMESPACE" istio-injection=enabled --overwrite
    
    # Deploy global architecture
    log "Applying global architecture configuration..."
    kubectl apply -f "$DEPLOYMENT_DIR/multi-region-architecture.yaml"
    
    # Wait for CockroachDB to be ready
    log "Waiting for global state store to be ready..."
    kubectl wait --for=condition=ready pod -l app=cockroachdb -n "$GLOBAL_NAMESPACE" --timeout=300s || {
        warning "CockroachDB not ready, checking status..."
        kubectl get pods -n "$GLOBAL_NAMESPACE" -l app=cockroachdb
    }
    
    # Initialize CockroachDB
    log "Initializing global state store..."
    kubectl exec -n "$GLOBAL_NAMESPACE" cockroachdb-0 -- \
        /cockroach/cockroach init --certs-dir=/cockroach/cockroach-certs || true
}

# Deploy regional instance
deploy_region() {
    local region=$1
    parse_region_config "$region"
    
    log "Deploying region: $region ($REGION_NAME)" "$BLUE"
    
    # Switch to region context
    if kubectl config get-contexts -o name | grep -q "$region"; then
        log "Switching to kubectl context: $region"
        kubectl config use-context "$region"
    else
        warning "Context $region not found, using current context"
    fi
    
    # Create region-specific values
    cat > "$LOG_DIR/${region}-values.env" <<EOF
REGION=$region
REGION_NAME=$REGION_NAME
VERSION=${VERSION:-latest}
REGISTRY=${REGISTRY:-docker.io}

# Capacity settings
MAX_CONCURRENT_INTENTS=100
MAX_DEPLOYMENTS_PER_MINUTE=50
MAX_LLM_RPS=10
MAX_VECTOR_CONNECTIONS=100

# Regional features
EDGE_ENABLED=${EDGE_ENABLED:-false}
LOCAL_ML_ENABLED=${LOCAL_ML_ENABLED:-true}
DATA_RESIDENCY=${DATA_RESIDENCY:-false}
COMPLIANCE_MODE=${COMPLIANCE_MODE:-standard}

# Failover regions
FAILOVER_REGION_1=${FAILOVER_REGION_1:-us-west-2}
FAILOVER_REGION_2=${FAILOVER_REGION_2:-eu-west-1}

# LLM settings
LLM_REPLICAS=${REGION_REPLICAS:-2}
LLM_CPU_REQUEST=1000m
LLM_MEMORY_REQUEST=2Gi
LLM_CPU_LIMIT=2000m
LLM_MEMORY_LIMIT=4Gi
REGIONAL_MODEL_ENDPOINT=https://api-${region}.openai.com

# Weaviate settings
WEAVIATE_REPLICAS=${REGION_REPLICAS:-2}
WEAVIATE_VERSION=1.25.0
WEAVIATE_CPU_REQUEST=1000m
WEAVIATE_MEMORY_REQUEST=4Gi
WEAVIATE_CPU_LIMIT=2000m
WEAVIATE_MEMORY_LIMIT=8Gi
WEAVIATE_STORAGE_SIZE=${REGION_STORAGE}
WEAVIATE_REPLICATION_FACTOR=2
STORAGE_CLASS=fast-ssd
AWS_REGION=${region}

# HPA settings
HPA_MIN_REPLICAS=2
HPA_MAX_REPLICAS=10
HPA_CPU_TARGET=70
HPA_MEMORY_TARGET=80
HPA_RPS_TARGET=50
HPA_SCALE_DOWN_WINDOW=300
HPA_SCALE_UP_WINDOW=60
EOF
    
    # Deploy regional template with substitution
    log "Deploying regional components..."
    envsubst < "$DEPLOYMENT_DIR/region-deployment-template.yaml" > "$LOG_DIR/${region}-deployment.yaml"
    kubectl apply -f "$LOG_DIR/${region}-deployment.yaml"
    
    # Configure cross-region connectivity
    configure_cross_region_connectivity "$region"
    
    # Verify deployment
    verify_regional_deployment "$region"
}

# Configure cross-region connectivity
configure_cross_region_connectivity() {
    local region=$1
    
    log "Configuring cross-region connectivity for $region..." "$CYAN"
    
    # Create multi-cluster secret
    istioctl create-remote-secret \
        --name="$region" \
        --namespace="$REGIONAL_NAMESPACE" > "$LOG_DIR/${region}-remote-secret.yaml"
    
    # Apply to other regions
    for other_region in "${!REGIONS[@]}"; do
        if [[ "$other_region" != "$region" ]]; then
            if kubectl config get-contexts -o name | grep -q "$other_region"; then
                log "Applying remote secret to $other_region"
                kubectl config use-context "$other_region"
                kubectl apply -f "$LOG_DIR/${region}-remote-secret.yaml"
            fi
        fi
    done
    
    # Switch back to original region
    kubectl config use-context "$region"
}

# Verify regional deployment
verify_regional_deployment() {
    local region=$1
    
    log "Verifying deployment in region $region..." "$CYAN"
    
    # Check pod status
    kubectl get pods -n "$REGIONAL_NAMESPACE" -o wide
    
    # Check service mesh connectivity
    istioctl proxy-status -n "$REGIONAL_NAMESPACE"
    
    # Test cross-region connectivity
    if [[ "$region" != "us-east-1" ]]; then
        log "Testing connectivity to primary region..."
        kubectl exec deployment/llm-processor-regional -n "$REGIONAL_NAMESPACE" -- \
            curl -s -o /dev/null -w "%{http_code}" https://us-east-1.nephoran.io/health || true
    fi
}

# Deploy global traffic management
deploy_global_traffic_management() {
    log "Deploying global traffic management..." "$BLUE"
    
    # Apply global traffic policies
    kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: global-traffic-routing
  namespace: ${GLOBAL_NAMESPACE}
spec:
  hosts:
  - global.nephoran.io
  gateways:
  - nephoran-global-gateway
  http:
  - match:
    - headers:
        x-region:
          exact: us-east-1
    route:
    - destination:
        host: us-east-1.nephoran.local
  - match:
    - headers:
        x-region:
          exact: eu-west-1
    route:
    - destination:
        host: eu-west-1.nephoran.local
  - match:
    - headers:
        x-region:
          exact: ap-southeast-1
    route:
    - destination:
        host: ap-southeast-1.nephoran.local
  - route:
    - destination:
        host: us-east-1.nephoran.local
        subset: primary
      weight: 40
    - destination:
        host: eu-west-1.nephoran.local
        subset: secondary
      weight: 30
    - destination:
        host: ap-southeast-1.nephoran.local
        subset: secondary
      weight: 20
    - destination:
        host: us-west-2.nephoran.local
        subset: dr
      weight: 10
EOF
}

# Test multi-region deployment
test_multi_region() {
    log "Testing multi-region deployment..." "$BLUE"
    
    # Create test intent
    kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: multi-region-test
  namespace: ${REGIONAL_NAMESPACE}
spec:
  description: "Deploy AMF across multiple regions for global coverage"
  priority: high
  annotations:
    nephoran.io/multi-region: "true"
    nephoran.io/regions: "us-east-1,eu-west-1,ap-southeast-1"
EOF
    
    # Monitor propagation
    log "Monitoring intent propagation across regions..."
    for region in "${!REGIONS[@]}"; do
        if kubectl config get-contexts -o name | grep -q "$region"; then
            kubectl config use-context "$region"
            kubectl get networkintent multi-region-test -n "$REGIONAL_NAMESPACE" || true
        fi
    done
}

# Generate deployment report
generate_deployment_report() {
    log "Generating multi-region deployment report..." "$CYAN"
    
    cat > "$LOG_DIR/multi-region-deployment-report.txt" <<EOF
Multi-Region Deployment Report
==============================
Date: $(date)
Regions Deployed: ${!REGIONS[@]}

Global Control Plane:
$(kubectl get pods -n "$GLOBAL_NAMESPACE" 2>/dev/null || echo "Not accessible")

Regional Deployments:
EOF
    
    for region in "${!REGIONS[@]}"; do
        if kubectl config get-contexts -o name | grep -q "$region"; then
            kubectl config use-context "$region"
            echo -e "\n$region:" >> "$LOG_DIR/multi-region-deployment-report.txt"
            kubectl get pods -n "$REGIONAL_NAMESPACE" --no-headers 2>/dev/null | wc -l >> "$LOG_DIR/multi-region-deployment-report.txt"
        fi
    done
    
    log "Report saved to: $LOG_DIR/multi-region-deployment-report.txt" "$GREEN"
}

# Main deployment function
main() {
    log "Starting Nephoran Multi-Region Deployment" "$BLUE"
    log "==========================================" "$BLUE"
    
    # Parse command line arguments
    case "${1:-all}" in
        "global")
            check_prerequisites
            deploy_global_control_plane
            ;;
        "regions")
            check_prerequisites
            for region in "${!REGIONS[@]}"; do
                deploy_region "$region"
            done
            deploy_global_traffic_management
            ;;
        "test")
            test_multi_region
            ;;
        "all")
            check_prerequisites
            deploy_global_control_plane
            for region in "${!REGIONS[@]}"; do
                deploy_region "$region"
            done
            deploy_global_traffic_management
            test_multi_region
            generate_deployment_report
            ;;
        *)
            if [[ -n "${REGIONS[$1]:-}" ]]; then
                check_prerequisites
                deploy_region "$1"
            else
                error "Unknown command or region: $1"
            fi
            ;;
    esac
    
    log "==========================================" "$GREEN"
    log "Multi-region deployment completed!" "$GREEN"
    log "Logs saved to: $LOG_DIR" "$GREEN"
}

# Execute main function
main "$@"
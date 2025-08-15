#!/bin/bash
# Nephoran Intent Operator - Istio Service Mesh Deployment Script
# Phase 4 Enterprise Architecture Implementation

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ISTIO_VERSION=${ISTIO_VERSION:-"1.20.0"}
NAMESPACE="nephoran-system"
ISTIO_NAMESPACE="istio-system"
LOG_FILE="/tmp/istio-deployment-$(date +%Y%m%d-%H%M%S).log"

# Logging function
log() {
    echo -e "${2:-$GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
    exit 1
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_FILE"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..." "$BLUE"
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed. Please install kubectl first."
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    fi
    
    # Check if istioctl is installed
    if ! command -v istioctl &> /dev/null; then
        log "istioctl not found. Installing istioctl ${ISTIO_VERSION}..." "$YELLOW"
        curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh -
        export PATH=$PWD/istio-${ISTIO_VERSION}/bin:$PATH
    fi
    
    # Verify istioctl version
    INSTALLED_VERSION=$(istioctl version --remote=false 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    log "istioctl version: ${INSTALLED_VERSION}"
    
    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log "Creating namespace $NAMESPACE..."
        kubectl create namespace "$NAMESPACE"
        kubectl label namespace "$NAMESPACE" istio-injection=enabled
    else
        log "Namespace $NAMESPACE already exists"
        # Ensure istio injection is enabled
        kubectl label namespace "$NAMESPACE" istio-injection=enabled --overwrite
    fi
}

# Install Istio control plane
install_istio() {
    log "Installing Istio control plane..." "$BLUE"
    
    # Check if Istio is already installed
    if kubectl get namespace "$ISTIO_NAMESPACE" &> /dev/null; then
        warning "Istio system namespace already exists. Checking installation status..."
        if istioctl verify-install &> /dev/null; then
            log "Istio is already installed and healthy"
            return 0
        fi
    fi
    
    # Apply Istio operator configuration
    log "Applying Istio operator configuration..."
    kubectl apply -f deployments/istio/istio-service-mesh-config.yaml
    
    # Wait for Istio to be ready
    log "Waiting for Istio control plane to be ready..."
    kubectl wait --for=condition=Ready pod -l app=istiod -n "$ISTIO_NAMESPACE" --timeout=300s
    
    # Verify installation
    if istioctl verify-install; then
        log "Istio installation verified successfully" "$GREEN"
    else
        error "Istio installation verification failed"
    fi
}

# Deploy Istio resources
deploy_istio_resources() {
    log "Deploying Istio resources for Nephoran..." "$BLUE"
    
    # Deploy Virtual Services
    log "Deploying Virtual Services..."
    kubectl apply -f deployments/istio/virtual-services.yaml
    
    # Deploy Destination Rules
    log "Deploying Destination Rules..."
    kubectl apply -f deployments/istio/destination-rules.yaml
    
    # Deploy Security Policies
    log "Deploying Security Policies..."
    kubectl apply -f deployments/istio/security-policies.yaml
    
    # Deploy Gateway
    log "Deploying Gateway configuration..."
    kubectl apply -f deployments/istio/gateway.yaml
    
    # Wait for configurations to propagate
    log "Waiting for Istio configurations to propagate..."
    sleep 10
}

# Enable sidecar injection for existing deployments
enable_sidecar_injection() {
    log "Enabling sidecar injection for existing deployments..." "$BLUE"
    
    # Get all deployments in namespace
    DEPLOYMENTS=$(kubectl get deployments -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
    
    for deployment in $DEPLOYMENTS; do
        log "Restarting deployment $deployment to inject sidecar..."
        kubectl rollout restart deployment/"$deployment" -n "$NAMESPACE"
    done
    
    # Wait for all deployments to be ready
    log "Waiting for all deployments to be ready with sidecars..."
    kubectl wait --for=condition=Available deployment --all -n "$NAMESPACE" --timeout=300s
}

# Verify service mesh configuration
verify_mesh_configuration() {
    log "Verifying service mesh configuration..." "$BLUE"
    
    # Check sidecar injection
    log "Checking sidecar injection status..."
    PODS_WITHOUT_SIDECAR=$(kubectl get pods -n "$NAMESPACE" -o json | jq -r '.items[] | select(.spec.containers | length == 1) | .metadata.name' | wc -l)
    
    if [ "$PODS_WITHOUT_SIDECAR" -gt 0 ]; then
        warning "Found $PODS_WITHOUT_SIDECAR pods without sidecar injection"
    else
        log "All pods have sidecar injection enabled" "$GREEN"
    fi
    
    # Check mTLS status
    log "Checking mTLS configuration..."
    istioctl authn tls-check -n "$NAMESPACE" | grep -E "STATUS|OK" | head -20
    
    # Check authorization policies
    log "Checking authorization policies..."
    kubectl get authorizationpolicies -n "$NAMESPACE" -o wide
    
    # Check virtual services
    log "Checking virtual services..."
    kubectl get virtualservices -n "$NAMESPACE" -o wide
    
    # Check destination rules
    log "Checking destination rules..."
    kubectl get destinationrules -n "$NAMESPACE" -o wide
}

# Configure observability
configure_observability() {
    log "Configuring Istio observability..." "$BLUE"
    
    # Enable telemetry v2
    kubectl apply -f - <<EOF
apiVersion: v1
data:
  mesh: |
    defaultConfig:
      proxyStatsMatcher:
        inclusionRegexps:
        - ".*outlier_detection.*"
        - ".*circuit_breakers.*"
        - ".*upstream_rq_retry.*"
        - ".*upstream_rq_pending.*"
        - ".*nephoran.*"
      telemetry:
        v2:
          prometheus:
            configOverride:
              inboundSidecar:
                disable_host_header_fallback: false
              outboundSidecar:
                disable_host_header_fallback: false
              gateway:
                disable_host_header_fallback: false
kind: ConfigMap
metadata:
  name: istio
  namespace: ${ISTIO_NAMESPACE}
EOF
    
    # Restart Istio components to pick up configuration
    kubectl rollout restart deployment/istiod -n "$ISTIO_NAMESPACE"
}

# Test service mesh functionality
test_mesh_functionality() {
    log "Testing service mesh functionality..." "$BLUE"
    
    # Create test pod
    log "Creating test pod..."
    kubectl run test-mesh-pod --image=curlimages/curl:latest -n "$NAMESPACE" --rm -i --restart=Never -- \
        curl -s -o /dev/null -w "%{http_code}" http://llm-processor:8080/health || true
    
    # Test mTLS between services
    log "Testing mTLS communication..."
    kubectl exec deployment/nephio-bridge -n "$NAMESPACE" -c istio-proxy -- \
        openssl s_client -connect llm-processor:8080 -servername llm-processor -showcerts </dev/null 2>/dev/null | \
        grep -E "subject|issuer" | head -4 || true
    
    # Check metrics
    log "Checking Istio metrics..."
    kubectl exec deployment/prometheus -n "$ISTIO_NAMESPACE" -- \
        promtool query instant 'istio_request_total{destination_service_namespace="'$NAMESPACE'"}' 2>/dev/null | head -10 || true
}

# Generate summary report
generate_report() {
    log "Generating deployment report..." "$BLUE"
    
    cat > "/tmp/istio-deployment-report-$(date +%Y%m%d-%H%M%S).txt" <<EOF
Istio Service Mesh Deployment Report
====================================
Date: $(date)
Istio Version: ${ISTIO_VERSION}
Namespace: ${NAMESPACE}

Components Deployed:
- Virtual Services: $(kubectl get virtualservices -n "$NAMESPACE" --no-headers | wc -l)
- Destination Rules: $(kubectl get destinationrules -n "$NAMESPACE" --no-headers | wc -l)
- Authorization Policies: $(kubectl get authorizationpolicies -n "$NAMESPACE" --no-headers | wc -l)
- Gateways: $(kubectl get gateways -n "$NAMESPACE" --no-headers | wc -l)

Sidecar Injection Status:
$(kubectl get pods -n "$NAMESPACE" -o json | jq -r '.items[] | "\(.metadata.name): \(.spec.containers | length) containers"')

Service Mesh Health:
$(istioctl proxy-status -n "$NAMESPACE" 2>/dev/null || echo "Unable to fetch proxy status")

Log file: ${LOG_FILE}
EOF
    
    log "Deployment report saved to: /tmp/istio-deployment-report-$(date +%Y%m%d-%H%M%S).txt" "$GREEN"
}

# Main execution
main() {
    log "Starting Istio Service Mesh deployment for Nephoran Intent Operator" "$BLUE"
    log "============================================================" "$BLUE"
    
    check_prerequisites
    install_istio
    deploy_istio_resources
    enable_sidecar_injection
    verify_mesh_configuration
    configure_observability
    test_mesh_functionality
    generate_report
    
    log "============================================================" "$GREEN"
    log "Istio Service Mesh deployment completed successfully!" "$GREEN"
    log "Log file: ${LOG_FILE}" "$GREEN"
}

# Execute main function
main "$@"
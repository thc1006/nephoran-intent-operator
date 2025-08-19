#!/bin/bash

# End-to-end test script for Nephoran Intent Operator webhook
# This script sets up a test environment and runs webhook validation tests

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-nephoran-system}"
WEBHOOK_IMAGE="${WEBHOOK_IMAGE:-nephoran/webhook-manager:latest}"
TIMEOUT="${TIMEOUT:-300s}"
CLEANUP="${CLEANUP:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

cleanup() {
    if [[ "${CLEANUP}" == "true" ]]; then
        log "Cleaning up test resources..."
        kubectl delete -f examples/intent-scaling-up.json --ignore-not-found=true >/dev/null 2>&1 || true
        kubectl delete -f examples/intent-scaling-down.json --ignore-not-found=true >/dev/null 2>&1 || true
        kubectl delete networkintent test-webhook-intent --ignore-not-found=true >/dev/null 2>&1 || true
    fi
}

trap cleanup EXIT

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl >/dev/null 2>&1; then
        error "kubectl is required but not installed"
        exit 1
    fi
    
    # Check if kustomize is available
    if ! command -v kustomize >/dev/null 2>&1; then
        warn "kustomize not found, installing..."
        go install sigs.k8s.io/kustomize/kustomize/v5@latest
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info >/dev/null 2>&1; then
        error "Cannot access Kubernetes cluster"
        exit 1
    fi
    
    # Check if cert-manager is installed
    if ! kubectl get crd certificates.cert-manager.io >/dev/null 2>&1; then
        error "cert-manager is required but not installed"
        error "Install with: kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml"
        exit 1
    fi
    
    log "Prerequisites check passed"
}

deploy_webhook() {
    log "Deploying webhook using kustomize..."
    
    # Apply the kustomize configuration
    kustomize build config/default | kubectl apply -f -
    
    log "Waiting for certificate to be ready..."
    kubectl wait --for=condition=Ready certificate/webhook-serving-cert -n "${NAMESPACE}" --timeout="${TIMEOUT}" || {
        error "Certificate not ready after ${TIMEOUT}"
        kubectl describe certificate webhook-serving-cert -n "${NAMESPACE}"
        exit 1
    }
    
    log "Waiting for webhook deployment to be ready..."
    kubectl wait --for=condition=Available deployment/webhook-manager -n "${NAMESPACE}" --timeout="${TIMEOUT}" || {
        error "Webhook deployment not ready after ${TIMEOUT}"
        kubectl describe deployment webhook-manager -n "${NAMESPACE}"
        kubectl logs -n "${NAMESPACE}" -l app=webhook-manager --tail=50
        exit 1
    }
    
    log "Webhook deployment ready"
}

test_webhook_validation() {
    log "Testing webhook validation..."
    
    # Create a test NetworkIntent with invalid data (should be rejected)
    cat <<EOF | kubectl apply -f - && {
        error "Webhook should have rejected invalid NetworkIntent"
        exit 1
    } || log "✅ Invalid NetworkIntent correctly rejected by webhook"
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: test-invalid-intent
  namespace: default
spec:
  intentType: "invalid-type"
  replicas: -1
  target: ""
  namespace: ""
EOF

    # Create a valid NetworkIntent (should be accepted and defaulted)
    cat <<EOF | kubectl apply -f -
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: test-valid-intent
  namespace: default
spec:
  intentType: "scaling"
  replicas: 3
  target: "test-deployment"
  namespace: "default"
EOF

    log "✅ Valid NetworkIntent correctly accepted by webhook"
    
    # Check if the defaulter worked (source should be set to "user")
    SOURCE=$(kubectl get networkintent test-valid-intent -o jsonpath='{.spec.source}')
    if [[ "${SOURCE}" == "user" ]]; then
        log "✅ Defaulter webhook correctly set source to 'user'"
    else
        error "Defaulter webhook failed to set source field"
        exit 1
    fi
    
    # Clean up test resources
    kubectl delete networkintent test-valid-intent --ignore-not-found=true
    
    log "Webhook validation tests passed"
}

verify_webhook_configuration() {
    log "Verifying webhook configuration..."
    
    # Check MutatingWebhookConfiguration
    if kubectl get mutatingwebhookconfiguration nephoran-networkintent-mutating >/dev/null 2>&1; then
        log "✅ MutatingWebhookConfiguration found"
        
        # Verify CA bundle is injected
        CA_BUNDLE=$(kubectl get mutatingwebhookconfiguration nephoran-networkintent-mutating -o jsonpath='{.webhooks[0].clientConfig.caBundle}')
        if [[ -n "${CA_BUNDLE}" ]]; then
            log "✅ CA bundle properly injected by cert-manager"
        else
            error "CA bundle not injected"
            exit 1
        fi
    else
        error "MutatingWebhookConfiguration not found"
        exit 1
    fi
    
    # Check ValidatingWebhookConfiguration
    if kubectl get validatingwebhookconfiguration nephoran-networkintent-validating >/dev/null 2>&1; then
        log "✅ ValidatingWebhookConfiguration found"
        
        # Verify CA bundle is injected
        CA_BUNDLE=$(kubectl get validatingwebhookconfiguration nephoran-networkintent-validating -o jsonpath='{.webhooks[0].clientConfig.caBundle}')
        if [[ -n "${CA_BUNDLE}" ]]; then
            log "✅ CA bundle properly injected by cert-manager"
        else
            error "CA bundle not injected"
            exit 1
        fi
    else
        error "ValidatingWebhookConfiguration not found"
        exit 1
    fi
    
    log "Webhook configuration verification passed"
}

show_status() {
    log "Webhook deployment status:"
    echo ""
    echo "Namespace: ${NAMESPACE}"
    kubectl get namespace "${NAMESPACE}" 2>/dev/null || echo "Namespace not found"
    echo ""
    echo "Certificates:"
    kubectl get certificates -n "${NAMESPACE}" 2>/dev/null || echo "No certificates found"
    echo ""
    echo "Webhook Deployment:"
    kubectl get deployment webhook-manager -n "${NAMESPACE}" 2>/dev/null || echo "Deployment not found"
    echo ""
    echo "Webhook Service:"
    kubectl get service webhook-manager -n "${NAMESPACE}" 2>/dev/null || echo "Service not found"
    echo ""
    echo "Webhook Configurations:"
    kubectl get mutatingwebhookconfiguration nephoran-networkintent-mutating 2>/dev/null || echo "MutatingWebhookConfiguration not found"
    kubectl get validatingwebhookconfiguration nephoran-networkintent-validating 2>/dev/null || echo "ValidatingWebhookConfiguration not found"
}

main() {
    log "Starting Nephoran Intent Operator webhook E2E tests"
    log "Namespace: ${NAMESPACE}"
    log "Webhook Image: ${WEBHOOK_IMAGE}"
    log "Timeout: ${TIMEOUT}"
    
    check_prerequisites
    deploy_webhook
    verify_webhook_configuration
    test_webhook_validation
    show_status
    
    log "🎉 All E2E tests passed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h          Show this help message"
        echo "  --no-cleanup        Don't clean up test resources"
        echo ""
        echo "Environment variables:"
        echo "  NAMESPACE           Target namespace (default: nephoran-system)"
        echo "  WEBHOOK_IMAGE       Webhook container image (default: nephoran/webhook-manager:latest)"
        echo "  TIMEOUT            Wait timeout (default: 300s)"
        echo "  CLEANUP            Clean up after test (default: true)"
        exit 0
        ;;
    --no-cleanup)
        CLEANUP="false"
        main
        ;;
    "")
        main
        ;;
    *)
        error "Unknown option: $1"
        error "Use --help for usage information"
        exit 1
        ;;
esac
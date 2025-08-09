#!/bin/bash
# Nephoran Intent Operator Webhook Deployment Script
# This script deploys the NetworkIntent validation webhook with proper certificates

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="nephoran-intent-operator-system"
WEBHOOK_NAME="nephoran-intent-operator-webhook"
CERT_ISSUER="selfsigned-cluster-issuer"
DRY_RUN=false
VERBOSE=false

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy the Nephoran Intent Operator NetworkIntent validation webhook.

OPTIONS:
    -n, --namespace NAMESPACE    Kubernetes namespace (default: ${NAMESPACE})
    -w, --webhook-name NAME      Webhook name prefix (default: ${WEBHOOK_NAME})
    -i, --issuer NAME           Certificate issuer name (default: ${CERT_ISSUER})
    -d, --dry-run               Show what would be deployed without applying
    -v, --verbose               Enable verbose output
    -h, --help                  Show this help message

EXAMPLES:
    $0                          Deploy with default settings
    $0 -n my-namespace          Deploy to custom namespace
    $0 -d                       Dry run - show what would be deployed
    $0 -v                       Enable verbose output

PREREQUISITES:
    - Kubernetes cluster with admission webhooks enabled
    - cert-manager installed and running
    - kubectl configured and accessible
    - Sufficient RBAC permissions

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -w|--webhook-name)
            WEBHOOK_NAME="$2"
            shift 2
            ;;
        -i|--issuer)
            CERT_ISSUER="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed or not in PATH"
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
    fi
    
    # Check for cert-manager
    if ! kubectl get crd certificates.cert-manager.io &> /dev/null; then
        print_error "cert-manager CRDs not found. Please install cert-manager first."
    fi
    
    # Check cert-manager is running
    if ! kubectl get pods -n cert-manager -l app=cert-manager --field-selector=status.phase=Running &> /dev/null; then
        print_warning "cert-manager pods may not be running. Certificate generation might fail."
    fi
    
    print_success "Prerequisites check completed"
}

# Function to create namespace
create_namespace() {
    print_status "Creating namespace ${NAMESPACE}..."
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_status "Namespace ${NAMESPACE} already exists"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            print_status "[DRY RUN] Would create namespace: ${NAMESPACE}"
        else
            kubectl create namespace "$NAMESPACE"
            print_success "Created namespace: ${NAMESPACE}"
        fi
    fi
}

# Function to deploy webhook manifests
deploy_webhook() {
    print_status "Deploying webhook manifests..."
    
    local manifest_file="config/webhook/manifests.yaml"
    
    if [[ ! -f "$manifest_file" ]]; then
        print_error "Webhook manifest file not found: ${manifest_file}"
    fi
    
    # Update manifests with custom values
    local temp_manifest="/tmp/webhook-manifests-${RANDOM}.yaml"
    sed -e "s/nephoran-intent-operator-system/${NAMESPACE}/g" \
        -e "s/selfsigned-cluster-issuer/${CERT_ISSUER}/g" \
        "$manifest_file" > "$temp_manifest"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "[DRY RUN] Would apply webhook manifests:"
        if [[ "$VERBOSE" == "true" ]]; then
            cat "$temp_manifest"
        fi
    else
        kubectl apply -f "$temp_manifest"
        print_success "Applied webhook manifests"
    fi
    
    rm -f "$temp_manifest"
}

# Function to wait for certificate readiness
wait_for_certificate() {
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "[DRY RUN] Would wait for certificate readiness"
        return
    fi
    
    print_status "Waiting for certificate to be ready..."
    
    local timeout=300  # 5 minutes
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if kubectl get certificate nephoran-intent-operator-serving-cert -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' | grep -q "True"; then
            print_success "Certificate is ready"
            return
        fi
        
        sleep 5
        elapsed=$((elapsed + 5))
        print_status "Waiting for certificate... (${elapsed}/${timeout}s)"
    done
    
    print_error "Certificate failed to become ready within ${timeout} seconds"
}

# Function to verify webhook deployment
verify_webhook() {
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "[DRY RUN] Would verify webhook deployment"
        return
    fi
    
    print_status "Verifying webhook deployment..."
    
    # Check ValidatingWebhookConfiguration
    if kubectl get validatingwebhookconfiguration nephoran-intent-operator-validating-webhook-configuration &> /dev/null; then
        print_success "ValidatingWebhookConfiguration is deployed"
    else
        print_error "ValidatingWebhookConfiguration not found"
    fi
    
    # Check webhook service
    if kubectl get service nephoran-intent-operator-webhook-service -n "$NAMESPACE" &> /dev/null; then
        print_success "Webhook service is deployed"
    else
        print_error "Webhook service not found"
    fi
    
    # Check certificate
    if kubectl get certificate nephoran-intent-operator-serving-cert -n "$NAMESPACE" &> /dev/null; then
        print_success "Certificate is deployed"
        
        # Check certificate status
        local cert_status=$(kubectl get certificate nephoran-intent-operator-serving-cert -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')
        if [[ "$cert_status" == "True" ]]; then
            print_success "Certificate is ready"
        else
            print_warning "Certificate is not ready yet"
        fi
    else
        print_error "Certificate not found"
    fi
}

# Function to test webhook functionality
test_webhook() {
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "[DRY RUN] Would test webhook functionality"
        return
    fi
    
    print_status "Testing webhook functionality..."
    
    # Create a temporary valid NetworkIntent
    local valid_intent="/tmp/valid-intent-${RANDOM}.yaml"
    cat > "$valid_intent" << EOF
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: webhook-test-valid
  namespace: ${NAMESPACE}
spec:
  intent: "Deploy AMF with high availability for production cluster testing"
EOF
    
    # Create a temporary invalid NetworkIntent
    local invalid_intent="/tmp/invalid-intent-${RANDOM}.yaml"
    cat > "$invalid_intent" << EOF
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: webhook-test-invalid
  namespace: ${NAMESPACE}
spec:
  intent: "Make me coffee and send emails"
EOF
    
    # Test valid intent (should succeed)
    print_status "Testing valid NetworkIntent (should succeed)..."
    if kubectl apply -f "$valid_intent" --dry-run=server &> /dev/null; then
        print_success "Valid NetworkIntent passed webhook validation"
    else
        print_warning "Valid NetworkIntent failed webhook validation (unexpected)"
    fi
    
    # Test invalid intent (should fail)
    print_status "Testing invalid NetworkIntent (should fail)..."
    if kubectl apply -f "$invalid_intent" --dry-run=server &> /dev/null; then
        print_warning "Invalid NetworkIntent passed webhook validation (unexpected)"
    else
        print_success "Invalid NetworkIntent correctly rejected by webhook"
    fi
    
    # Clean up temporary files
    rm -f "$valid_intent" "$invalid_intent"
}

# Function to show deployment status
show_status() {
    print_status "Webhook deployment status:"
    
    echo ""
    echo "=== ValidatingWebhookConfiguration ==="
    kubectl get validatingwebhookconfiguration nephoran-intent-operator-validating-webhook-configuration -o wide 2>/dev/null || echo "Not found"
    
    echo ""
    echo "=== Webhook Service ==="
    kubectl get service nephoran-intent-operator-webhook-service -n "$NAMESPACE" -o wide 2>/dev/null || echo "Not found"
    
    echo ""
    echo "=== Certificate ==="
    kubectl get certificate nephoran-intent-operator-serving-cert -n "$NAMESPACE" -o wide 2>/dev/null || echo "Not found"
    
    echo ""
    echo "=== Certificate Status ==="
    kubectl describe certificate nephoran-intent-operator-serving-cert -n "$NAMESPACE" 2>/dev/null || echo "Not found"
    
    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
        echo "=== Webhook Configuration Details ==="
        kubectl describe validatingwebhookconfiguration nephoran-intent-operator-validating-webhook-configuration 2>/dev/null || echo "Not found"
    fi
}

# Main deployment flow
main() {
    print_status "Starting Nephoran Intent Operator Webhook Deployment"
    print_status "Namespace: ${NAMESPACE}"
    print_status "Webhook Name: ${WEBHOOK_NAME}"
    print_status "Certificate Issuer: ${CERT_ISSUER}"
    print_status "Dry Run: ${DRY_RUN}"
    
    echo ""
    
    # Execute deployment steps
    check_prerequisites
    create_namespace
    deploy_webhook
    
    if [[ "$DRY_RUN" == "false" ]]; then
        wait_for_certificate
        verify_webhook
        test_webhook
        show_status
        
        echo ""
        print_success "Webhook deployment completed successfully!"
        print_status "The NetworkIntent validation webhook is now active."
        print_status "Try creating NetworkIntent resources to test validation."
    else
        print_status "Dry run completed. Use --dry-run=false to actually deploy."
    fi
}

# Run main function
main "$@"
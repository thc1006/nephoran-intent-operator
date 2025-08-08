#!/bin/bash

# deploy-performance-monitoring.sh - Deploy comprehensive performance monitoring stack for Nephoran Intent Operator
# This script deploys the complete performance benchmarking and monitoring infrastructure

set -euo pipefail

# Configuration
NAMESPACE="nephoran-monitoring"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MONITORING_DIR="$PROJECT_ROOT/deployments/monitoring"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if running in correct directory
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        log_error "Script must be run from project root directory"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create namespace
create_namespace() {
    log_info "Creating namespace: $NAMESPACE"
    
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Label namespace for monitoring
    kubectl label namespace "$NAMESPACE" name=nephoran-monitoring --overwrite
    kubectl label namespace "$NAMESPACE" monitoring=enabled --overwrite
    
    log_success "Namespace $NAMESPACE created/updated"
}

# Deploy Prometheus for performance benchmarking
deploy_prometheus() {
    log_info "Deploying Prometheus performance benchmarking stack..."
    
    # Create ServiceAccount if it doesn't exist
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus-benchmarking
rules:
- apiGroups: [""]
  resources: ["nodes", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions", "networking.k8s.io"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus-benchmarking
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus-benchmarking
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: $NAMESPACE
EOF

    # Deploy Prometheus configuration and deployment
    kubectl apply -f "$MONITORING_DIR/performance-benchmarking-prometheus.yaml"
    
    # Wait for Prometheus to be ready
    log_info "Waiting for Prometheus to be ready..."
    kubectl wait --for=condition=Ready pod -l app=performance-benchmarking-prometheus -n "$NAMESPACE" --timeout=300s
    
    log_success "Prometheus performance benchmarking stack deployed"
}

# Deploy Grafana with performance dashboards
deploy_grafana() {
    log_info "Deploying Grafana with performance benchmarking dashboards..."
    
    # Deploy Grafana configuration and dashboards
    kubectl apply -f "$MONITORING_DIR/performance-benchmarking-grafana.yaml"
    
    # Wait for Grafana to be ready
    log_info "Waiting for Grafana to be ready..."
    kubectl wait --for=condition=Ready pod -l app=performance-benchmarking-grafana -n "$NAMESPACE" --timeout=300s
    
    log_success "Grafana performance benchmarking dashboards deployed"
}

# Deploy additional monitoring components
deploy_monitoring_components() {
    log_info "Deploying additional monitoring components..."
    
    # Deploy ServiceMonitors for automatic service discovery
    kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: performance-benchmarking
  namespace: $NAMESPACE
  labels:
    app: performance-benchmarking
spec:
  selector:
    matchLabels:
      app: performance-test-runner
  endpoints:
  - port: metrics
    interval: 1s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-components
  namespace: $NAMESPACE
  labels:
    app: nephoran-monitoring
spec:
  selector:
    matchLabels:
      component: nephoran
  namespaceSelector:
    any: true
  endpoints:
  - port: metrics
    interval: 5s
    path: /metrics
EOF

    log_success "Additional monitoring components deployed"
}

# Create port-forward services for easy access
create_access_services() {
    log_info "Creating access services..."
    
    # Create NodePort services for external access
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: prometheus-benchmarking-nodeport
  namespace: $NAMESPACE
  labels:
    app: prometheus-benchmarking-external
spec:
  type: NodePort
  ports:
  - port: 9090
    targetPort: 9090
    protocol: TCP
    name: web
    nodePort: 30090
  selector:
    app: performance-benchmarking-prometheus
---
apiVersion: v1
kind: Service
metadata:
  name: grafana-benchmarking-nodeport
  namespace: $NAMESPACE
  labels:
    app: grafana-benchmarking-external
spec:
  type: NodePort
  ports:
  - port: 3000
    targetPort: 3000
    protocol: TCP
    name: web
    nodePort: 30030
  selector:
    app: performance-benchmarking-grafana
EOF

    log_success "Access services created"
}

# Validate deployment
validate_deployment() {
    log_info "Validating deployment..."
    
    # Check if all pods are running
    local failed=false
    
    if ! kubectl get pods -n "$NAMESPACE" -l app=performance-benchmarking-prometheus | grep -q "Running"; then
        log_error "Prometheus pods not running"
        failed=true
    fi
    
    if ! kubectl get pods -n "$NAMESPACE" -l app=performance-benchmarking-grafana | grep -q "Running"; then
        log_error "Grafana pods not running"
        failed=true
    fi
    
    if [ "$failed" = true ]; then
        log_error "Deployment validation failed"
        exit 1
    fi
    
    log_success "Deployment validation passed"
}

# Display access information
display_access_info() {
    log_info "Performance monitoring stack deployed successfully!"
    echo
    echo "=== ACCESS INFORMATION ==="
    echo
    
    # Get cluster IP for instructions
    CLUSTER_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "localhost")
    
    echo "Prometheus (Performance Benchmarking):"
    echo "  - URL: http://$CLUSTER_IP:30090"
    echo "  - Port-forward: kubectl port-forward -n $NAMESPACE svc/performance-benchmarking-prometheus 9090:9090"
    echo
    
    echo "Grafana (Performance Dashboards):"
    echo "  - URL: http://$CLUSTER_IP:30030"
    echo "  - Port-forward: kubectl port-forward -n $NAMESPACE svc/performance-benchmarking-grafana 3000:3000"
    echo "  - Username: admin"
    echo "  - Password: nephoran-benchmarking-2024"
    echo
    
    echo "Key Dashboards:"
    echo "  1. Performance Benchmarking - Real-time Validation"
    echo "     - Validates all 6 performance claims in real-time"
    echo "     - Overall performance score aggregation"
    echo "  2. Statistical Validation Dashboard"
    echo "     - Confidence intervals and statistical significance"
    echo "     - Effect size analysis"
    echo "  3. Performance Regression Analysis"
    echo "     - Trend analysis and regression detection"
    echo "     - Performance stability scoring"
    echo
    
    echo "Performance Claims Monitored:"
    echo "  ✓ CLAIM 1: Sub-2-second P95 latency for intent processing"
    echo "  ✓ CLAIM 2: 200+ concurrent user capacity"
    echo "  ✓ CLAIM 3: 45 intents per minute throughput"
    echo "  ✓ CLAIM 4: 99.95% availability SLA"
    echo "  ✓ CLAIM 5: Sub-200ms P95 RAG retrieval latency"
    echo "  ✓ CLAIM 6: 87% cache hit rate"
    echo
    
    echo "Monitoring Features:"
    echo "  • Real-time performance metrics validation"
    echo "  • Statistical significance testing with 95% confidence"
    echo "  • Automated regression detection"
    echo "  • Load testing scenario results"
    echo "  • Performance trend analysis"
    echo "  • Comprehensive alerting on SLA violations"
    echo
}

# Cleanup function
cleanup() {
    log_warn "Cleaning up resources..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
    log_success "Cleanup completed"
}

# Main execution
main() {
    log_info "Starting Nephoran Performance Monitoring deployment..."
    
    # Parse command line arguments
    case "${1:-deploy}" in
        "deploy")
            check_prerequisites
            create_namespace
            deploy_prometheus
            deploy_grafana
            deploy_monitoring_components
            create_access_services
            validate_deployment
            display_access_info
            ;;
        "cleanup")
            cleanup
            ;;
        "validate")
            validate_deployment
            ;;
        *)
            echo "Usage: $0 [deploy|cleanup|validate]"
            echo
            echo "Commands:"
            echo "  deploy   - Deploy the complete performance monitoring stack (default)"
            echo "  cleanup  - Remove all monitoring components"
            echo "  validate - Validate existing deployment"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
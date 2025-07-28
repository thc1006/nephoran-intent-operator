#!/bin/bash
set -euo pipefail

# Comprehensive Weaviate Deployment Script for Nephoran Intent Operator
# This script handles the complete deployment of production-ready Weaviate cluster

# Configuration
NAMESPACE="nephoran-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/tmp/weaviate-deploy-$(date +%Y%m%d-%H%M%S).log"
TIMEOUT_SECONDS=600
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

log_info() {
    log "INFO" "${BLUE}$*${NC}"
}

log_warn() {
    log "WARN" "${YELLOW}$*${NC}"
}

log_error() {
    log "ERROR" "${RED}$*${NC}"
}

log_success() {
    log "SUCCESS" "${GREEN}$*${NC}"
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Deployment failed with exit code: $exit_code"
    log_error "Check the log file: $LOG_FILE"
    cleanup_on_failure
    exit $exit_code
}

trap handle_error ERR

# Cleanup function for failed deployments
cleanup_on_failure() {
    log_warn "Cleaning up failed deployment resources..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true || true
    log_warn "Cleanup completed"
}

# Pre-flight checks
preflight_checks() {
    log_info "Running pre-flight checks..."
    
    # Check required tools
    local required_tools=("kubectl" "curl" "python3" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check OpenAI API key
    if [[ -z "$OPENAI_API_KEY" ]]; then
        log_error "OPENAI_API_KEY environment variable is required"
        exit 1
    fi
    
    # Validate OpenAI API key
    if ! curl -s -H "Authorization: Bearer $OPENAI_API_KEY" \
         "https://api.openai.com/v1/models" > /dev/null; then
        log_error "Invalid OpenAI API key"
        exit 1
    fi
    
    # Check cluster resources
    local nodes_ready=$(kubectl get nodes --no-headers | grep -c "Ready" || echo "0")
    if [[ "$nodes_ready" -lt 2 ]]; then
        log_warn "Less than 2 nodes available. High availability may be compromised."
    fi
    
    # Check storage classes
    if ! kubectl get storageclass fast-ssd &> /dev/null; then
        log_warn "fast-ssd storage class not found. Using default storage class."
        # Update deployment to use default storage class
        sed -i 's/storageClassName: "fast-ssd"/storageClassName: ""/g' \
            "$SCRIPT_DIR/weaviate-deployment.yaml"
    fi
    
    log_success "Pre-flight checks completed"
}

# Create namespace and basic resources
create_namespace() {
    log_info "Creating namespace and basic resources..."
    
    # Create namespace with labels
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | \
    kubectl apply -f - || true
    
    # Label namespace for monitoring and policies
    kubectl label namespace "$NAMESPACE" \
        name="$NAMESPACE" \
        tier="database" \
        component="rag-system" \
        --overwrite
    
    # Create OpenAI API key secret
    kubectl create secret generic openai-api-key \
        --from-literal=api-key="$OPENAI_API_KEY" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Namespace and basic resources created"
}

# Deploy RBAC and security policies
deploy_rbac() {
    log_info "Deploying RBAC and security policies..."
    
    if [[ -f "$SCRIPT_DIR/rbac.yaml" ]]; then
        kubectl apply -f "$SCRIPT_DIR/rbac.yaml"
        log_success "RBAC policies deployed"
    else
        log_error "RBAC file not found: $SCRIPT_DIR/rbac.yaml"
        exit 1
    fi
}

# Deploy network policies
deploy_network_policies() {
    log_info "Deploying network policies..."
    
    if [[ -f "$SCRIPT_DIR/network-policy.yaml" ]]; then
        kubectl apply -f "$SCRIPT_DIR/network-policy.yaml"
        log_success "Network policies deployed"
    else
        log_warn "Network policy file not found, skipping..."
    fi
}

# Deploy Weaviate cluster
deploy_weaviate() {
    log_info "Deploying Weaviate cluster..."
    
    if [[ -f "$SCRIPT_DIR/weaviate-deployment.yaml" ]]; then
        kubectl apply -f "$SCRIPT_DIR/weaviate-deployment.yaml"
        log_success "Weaviate deployment manifest applied"
    else
        log_error "Weaviate deployment file not found: $SCRIPT_DIR/weaviate-deployment.yaml"
        exit 1
    fi
}

# Wait for deployment readiness
wait_for_deployment() {
    log_info "Waiting for Weaviate deployment to be ready..."
    
    # Wait for deployment to be available
    if kubectl wait --for=condition=available \
        deployment/weaviate \
        --namespace="$NAMESPACE" \
        --timeout="${TIMEOUT_SECONDS}s"; then
        log_success "Weaviate deployment is available"
    else
        log_error "Weaviate deployment failed to become ready within ${TIMEOUT_SECONDS}s"
        kubectl describe deployment weaviate -n "$NAMESPACE"
        kubectl logs -l app=weaviate -n "$NAMESPACE" --tail=50
        exit 1
    fi
    
    # Wait for all pods to be ready
    local ready_pods=0
    local total_pods=0
    local max_attempts=60
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        ready_pods=$(kubectl get pods -l app=weaviate -n "$NAMESPACE" \
            -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | \
            grep -o "True" | wc -l || echo "0")
        total_pods=$(kubectl get pods -l app=weaviate -n "$NAMESPACE" \
            --no-headers | wc -l || echo "0")
        
        if [[ "$ready_pods" -eq "$total_pods" ]] && [[ "$total_pods" -gt 0 ]]; then
            log_success "All $total_pods Weaviate pods are ready"
            break
        fi
        
        log_info "Waiting for pods to be ready: $ready_pods/$total_pods"
        sleep 10
        ((attempt++))
    done
    
    if [[ $attempt -eq $max_attempts ]]; then
        log_error "Pods failed to become ready within timeout"
        kubectl describe pods -l app=weaviate -n "$NAMESPACE"
        exit 1
    fi
}

# Deploy monitoring and observability
deploy_monitoring() {
    log_info "Deploying monitoring and observability..."
    
    if [[ -f "$SCRIPT_DIR/monitoring.yaml" ]]; then
        kubectl apply -f "$SCRIPT_DIR/monitoring.yaml"
        log_success "Monitoring resources deployed"
    else
        log_warn "Monitoring file not found, skipping..."
    fi
}

# Deploy backup automation
deploy_backup() {
    log_info "Deploying backup automation..."
    
    if [[ -f "$SCRIPT_DIR/backup-cronjob.yaml" ]]; then
        kubectl apply -f "$SCRIPT_DIR/backup-cronjob.yaml"
        log_success "Backup automation deployed"
    else
        log_warn "Backup file not found, skipping..."
    fi
}

# Initialize telecom schemas
initialize_schemas() {
    log_info "Initializing telecom-specific schemas..."
    
    if [[ -f "$SCRIPT_DIR/telecom-schema.py" ]]; then
        # Create a job to run the schema initialization
        cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: weaviate-schema-init
  namespace: $NAMESPACE
  labels:
    app: weaviate
    component: schema-init
spec:
  backoffLimit: 3
  activeDeadlineSeconds: 1800
  template:
    spec:
      serviceAccountName: weaviate
      restartPolicy: Never
      containers:
      - name: schema-init
        image: python:3.11-slim
        env:
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: WEAVIATE_API_KEY
          valueFrom:
            secretKeyRef:
              name: weaviate-api-key
              key: api-key
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-api-key
              key: api-key
        command: ["/bin/bash"]
        args:
        - -c
        - |
          set -e
          pip install weaviate-client==3.25.3 requests
          python3 /scripts/telecom-schema.py
        volumeMounts:
        - name: schema-script
          mountPath: /scripts
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: schema-script
        configMap:
          name: telecom-schema-script
          defaultMode: 0755
EOF
        
        # Create ConfigMap with the schema script
        kubectl create configmap telecom-schema-script \
            --from-file="telecom-schema.py=$SCRIPT_DIR/telecom-schema.py" \
            --namespace="$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -
        
        # Wait for schema initialization job to complete
        if kubectl wait --for=condition=complete \
            job/weaviate-schema-init \
            --namespace="$NAMESPACE" \
            --timeout=1800s; then
            log_success "Schema initialization completed"
        else
            log_error "Schema initialization failed"
            kubectl logs -l job-name=weaviate-schema-init -n "$NAMESPACE"
            exit 1
        fi
        
        # Clean up the job
        kubectl delete job weaviate-schema-init -n "$NAMESPACE"
        
    else
        log_warn "Schema initialization script not found, skipping..."
    fi
}

# Health and connectivity checks
run_health_checks() {
    log_info "Running comprehensive health checks..."
    
    # Get Weaviate service endpoint
    local weaviate_service=$(kubectl get service weaviate -n "$NAMESPACE" \
        -o jsonpath='{.spec.clusterIP}')
    
    # Test basic connectivity
    if kubectl run weaviate-health-check \
        --image=curlimages/curl:8.5.0 \
        --rm -i --restart=Never \
        --namespace="$NAMESPACE" \
        -- curl -s "http://$weaviate_service:8080/v1/.well-known/ready" > /dev/null; then
        log_success "Weaviate health check passed"
    else
        log_error "Weaviate health check failed"
        exit 1
    fi
    
    # Test schema availability
    local schema_classes=$(kubectl run weaviate-schema-check \
        --image=curlimages/curl:8.5.0 \
        --rm -i --restart=Never \
        --namespace="$NAMESPACE" \
        -- curl -s "http://$weaviate_service:8080/v1/schema" | \
        jq -r '.classes[].class' 2>/dev/null || echo "")
    
    if echo "$schema_classes" | grep -q "TelecomKnowledge"; then
        log_success "Telecom schemas are available"
    else
        log_warn "Telecom schemas may not be properly initialized"
    fi
    
    # Test HPA status
    if kubectl get hpa weaviate-hpa -n "$NAMESPACE" &> /dev/null; then
        local hpa_status=$(kubectl get hpa weaviate-hpa -n "$NAMESPACE" \
            -o jsonpath='{.status.conditions[?(@.type=="ScalingActive")].status}')
        if [[ "$hpa_status" == "True" ]]; then
            log_success "HPA is active and functional"
        else
            log_warn "HPA may not be functioning properly"
        fi
    fi
    
    # Test backup job
    if kubectl get cronjob weaviate-backup -n "$NAMESPACE" &> /dev/null; then
        log_success "Backup automation is configured"
    else
        log_warn "Backup automation is not configured"
    fi
}

# Generate deployment summary
generate_summary() {
    log_info "Generating deployment summary..."
    
    local summary_file="/tmp/weaviate-deployment-summary-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$summary_file" <<EOF
====================================================================
Weaviate Deployment Summary - $(date)
====================================================================

Namespace: $NAMESPACE
Environment: $ENVIRONMENT

DEPLOYMENT STATUS:
$(kubectl get deployments -n "$NAMESPACE" -o wide)

PODS:
$(kubectl get pods -n "$NAMESPACE" -o wide)

SERVICES:
$(kubectl get services -n "$NAMESPACE" -o wide)

PERSISTENT VOLUMES:
$(kubectl get pvc -n "$NAMESPACE" -o wide)

HPA STATUS:
$(kubectl get hpa -n "$NAMESPACE" -o wide 2>/dev/null || echo "HPA not found")

BACKUP JOBS:
$(kubectl get cronjobs -n "$NAMESPACE" -o wide 2>/dev/null || echo "Backup jobs not found")

RESOURCE USAGE:
$(kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Metrics not available")

CONNECTION ENDPOINTS:
- Weaviate API: http://weaviate.$NAMESPACE.svc.cluster.local:8080
- Metrics: http://weaviate.$NAMESPACE.svc.cluster.local:2112/metrics

NEXT STEPS:
1. Verify schema initialization: kubectl logs -l component=schema-init -n $NAMESPACE
2. Test API connectivity: kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE
3. Monitor deployment: kubectl logs -f deployment/weaviate -n $NAMESPACE
4. Check backup status: kubectl get jobs -l component=backup -n $NAMESPACE

TROUBLESHOOTING:
- Deployment logs: $LOG_FILE
- Pod logs: kubectl logs -l app=weaviate -n $NAMESPACE
- Events: kubectl get events -n $NAMESPACE --sort-by='.firstTimestamp'

====================================================================
EOF
    
    cat "$summary_file"
    log_success "Deployment summary saved to: $summary_file"
}

# Main deployment function
main() {
    log_info "Starting Weaviate deployment for Nephoran Intent Operator"
    log_info "Deployment log: $LOG_FILE"
    
    preflight_checks
    create_namespace
    deploy_rbac
    deploy_network_policies
    deploy_weaviate
    wait_for_deployment
    deploy_monitoring
    deploy_backup
    initialize_schemas
    run_health_checks
    generate_summary
    
    log_success "Weaviate deployment completed successfully!"
    log_info "Access Weaviate: kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE"
    log_info "View logs: kubectl logs -f deployment/weaviate -n $NAMESPACE"
}

# Script usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy production-ready Weaviate cluster for Nephoran Intent Operator

OPTIONS:
    -h, --help          Show this help message
    -n, --namespace     Kubernetes namespace (default: nephoran-system)
    -e, --environment   Environment type (default: production)
    --dry-run          Show what would be deployed without executing
    --cleanup          Remove existing deployment before installing
    --skip-checks      Skip pre-flight checks (not recommended)

ENVIRONMENT VARIABLES:
    OPENAI_API_KEY     Required: OpenAI API key for embeddings
    ENVIRONMENT        Deployment environment (development, staging, production)

EXAMPLES:
    $0                              # Deploy with defaults
    $0 -n my-namespace             # Deploy to custom namespace
    $0 --environment staging       # Deploy staging configuration
    $0 --cleanup                   # Clean install

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --dry-run)
            log_info "Dry run mode - no changes will be made"
            set -n  # No execution mode
            shift
            ;;
        --cleanup)
            log_warn "Cleaning up existing deployment..."
            kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
            log_success "Cleanup completed"
            shift
            ;;
        --skip-checks)
            log_warn "Skipping pre-flight checks"
            preflight_checks() { log_warn "Pre-flight checks skipped"; }
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main deployment
main
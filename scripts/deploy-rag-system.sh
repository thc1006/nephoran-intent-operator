#!/bin/bash

# Comprehensive RAG System Deployment Script for Nephoran Intent Operator
# Deploys Weaviate vector database and enhanced RAG pipeline

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-default}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
WEAVIATE_API_KEY="${WEAVIATE_API_KEY:-nephoran-rag-key}"
KNOWLEDGE_BASE_INIT="${KNOWLEDGE_BASE_INIT:-true}"

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

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check required commands
    local required_commands=("kubectl" "kustomize" "docker")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "$cmd is required but not installed"
            exit 1
        fi
    done
    
    # Check kubectl connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "kubectl cannot connect to cluster"
        exit 1
    fi
    
    # Check OpenAI API key
    if [[ -z "$OPENAI_API_KEY" ]]; then
        log_error "OPENAI_API_KEY environment variable is required"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create secrets
create_secrets() {
    log_info "Creating secrets..."
    
    # Create OpenAI API key secret
    kubectl create secret generic openai-api-key \
        --from-literal=api-key="$OPENAI_API_KEY" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create Weaviate API key secret
    kubectl create secret generic weaviate-api-key \
        --from-literal=api-key="$WEAVIATE_API_KEY" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Secrets created successfully"
}

# Deploy Weaviate vector database
deploy_weaviate() {
    log_info "Deploying Weaviate vector database..."
    
    # Apply Weaviate deployment
    kubectl apply -f deployments/weaviate/ --namespace="$NAMESPACE"
    
    # Wait for Weaviate to be ready
    log_info "Waiting for Weaviate to be ready..."
    kubectl wait --for=condition=ready pod -l app=weaviate --timeout=300s --namespace="$NAMESPACE"
    
    # Verify Weaviate health
    local weaviate_pod=$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}' --namespace="$NAMESPACE")
    if kubectl exec "$weaviate_pod" --namespace="$NAMESPACE" -- wget -qO- http://localhost:8080/v1/.well-known/ready | grep -q "true"; then
        log_success "Weaviate deployed and ready"
    else
        log_error "Weaviate health check failed"
        exit 1
    fi
}

# Update RAG API deployment
update_rag_api_deployment() {
    log_info "Updating RAG API deployment configuration..."
    
    # Update ConfigMap with enhanced configuration
    kubectl create configmap rag-api-config \
        --from-literal=WEAVIATE_URL="http://weaviate:8080" \
        --from-literal=OPENAI_MODEL="gpt-4o-mini" \
        --from-literal=CACHE_MAX_SIZE="1000" \
        --from-literal=CACHE_TTL_SECONDS="3600" \
        --from-literal=CHUNK_SIZE="1000" \
        --from-literal=CHUNK_OVERLAP="200" \
        --from-literal=MAX_CONCURRENT_FILES="5" \
        --from-literal=KNOWLEDGE_BASE_PATH="/app/knowledge_base" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create knowledge base volume from existing directory
    if [[ -d "knowledge_base" ]]; then
        log_info "Creating knowledge base ConfigMap from local directory..."
        kubectl create configmap knowledge-base-docs \
            --from-file=knowledge_base/ \
            --namespace="$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    # Apply updated RAG API deployment
    kubectl apply -f deployments/kustomize/base/rag-api/ --namespace="$NAMESPACE"
    
    log_success "RAG API deployment updated"
}

# Populate knowledge base
populate_knowledge_base() {
    if [[ "$KNOWLEDGE_BASE_INIT" == "true" ]]; then
        log_info "Initializing knowledge base..."
        
        # Wait for RAG API to be ready
        kubectl wait --for=condition=ready pod -l app=rag-api --timeout=300s --namespace="$NAMESPACE"
        
        # Get RAG API pod name
        local rag_pod=$(kubectl get pods -l app=rag-api -o jsonpath='{.items[0].metadata.name}' --namespace="$NAMESPACE")
        
        # Trigger knowledge base population
        if kubectl exec "$rag_pod" --namespace="$NAMESPACE" -- \
            curl -s -X POST http://localhost:5001/knowledge/populate \
            -H "Content-Type: application/json" \
            -d '{}' | grep -q "success"; then
            log_success "Knowledge base populated successfully"
        else
            log_warning "Knowledge base population failed or no documents found"
        fi
    else
        log_info "Skipping knowledge base initialization (KNOWLEDGE_BASE_INIT=false)"
    fi
}

# Restart LLM processor to pick up new RAG API
restart_llm_processor() {
    log_info "Restarting LLM processor to use enhanced RAG API..."
    
    # Restart LLM processor deployment
    kubectl rollout restart deployment/llm-processor --namespace="$NAMESPACE"
    kubectl rollout status deployment/llm-processor --namespace="$NAMESPACE" --timeout=300s
    
    log_success "LLM processor restarted successfully"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying RAG system deployment..."
    
    # Check all pods are running
    local components=("weaviate" "rag-api" "llm-processor")
    for component in "${components[@]}"; do
        local ready_replicas=$(kubectl get deployment "$component" --namespace="$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired_replicas=$(kubectl get deployment "$component" --namespace="$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
        
        if [[ "$ready_replicas" == "$desired_replicas" ]] && [[ "$ready_replicas" != "0" ]]; then
            log_success "$component is ready ($ready_replicas/$desired_replicas replicas)"
        else
            log_error "$component is not ready ($ready_replicas/$desired_replicas replicas)"
            return 1
        fi
    done
    
    # Test RAG API endpoints
    local rag_pod=$(kubectl get pods -l app=rag-api -o jsonpath='{.items[0].metadata.name}' --namespace="$NAMESPACE")
    
    # Test health endpoint
    if kubectl exec "$rag_pod" --namespace="$NAMESPACE" -- \
        curl -s http://localhost:5001/healthz | grep -q "ok"; then
        log_success "RAG API health check passed"
    else
        log_error "RAG API health check failed"
        return 1
    fi
    
    # Test readiness endpoint
    if kubectl exec "$rag_pod" --namespace="$NAMESPACE" -- \
        curl -s http://localhost:5001/readyz | grep -q "ready"; then
        log_success "RAG API readiness check passed"
    else
        log_warning "RAG API readiness check failed (may need knowledge base population)"
    fi
    
    # Test stats endpoint
    if kubectl exec "$rag_pod" --namespace="$NAMESPACE" -- \
        curl -s http://localhost:5001/stats | grep -q "timestamp"; then
        log_success "RAG API stats endpoint working"
    else
        log_warning "RAG API stats endpoint not responding"
    fi
    
    log_success "RAG system deployment verification completed"
}

# Display connection information
show_connection_info() {
    log_info "RAG System Connection Information:"
    echo ""
    echo "Weaviate Vector Database:"
    echo "  Internal URL: http://weaviate:8080"
    echo "  Port Forward: kubectl port-forward svc/weaviate 8080:8080 --namespace=$NAMESPACE"
    echo ""
    echo "RAG API Service:"
    echo "  Internal URL: http://rag-api:5001"
    echo "  Port Forward: kubectl port-forward svc/rag-api 5001:5001 --namespace=$NAMESPACE"
    echo ""
    echo "Useful Commands:"
    echo "  Check logs: kubectl logs -f deployment/rag-api --namespace=$NAMESPACE"
    echo "  Test endpoint: curl http://localhost:5001/healthz (after port-forward)"
    echo "  Upload docs: curl -X POST -F 'files=@doc.pdf' http://localhost:5001/knowledge/upload"
    echo "  Get stats: curl http://localhost:5001/stats"
    echo ""
}

# Cleanup function
cleanup() {
    log_info "Cleaning up existing RAG system resources..."
    
    # Delete existing resources (ignore errors)
    kubectl delete -f deployments/weaviate/ --namespace="$NAMESPACE" --ignore-not-found=true
    kubectl delete configmap rag-api-config --namespace="$NAMESPACE" --ignore-not-found=true
    kubectl delete configmap knowledge-base-docs --namespace="$NAMESPACE" --ignore-not-found=true
    
    log_success "Cleanup completed"
}

# Main deployment function
deploy_rag_system() {
    log_info "Starting RAG system deployment..."
    
    check_prerequisites
    create_secrets
    deploy_weaviate
    update_rag_api_deployment
    populate_knowledge_base
    restart_llm_processor
    verify_deployment
    show_connection_info
    
    log_success "RAG system deployment completed successfully!"
}

# Help function
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Deploy the comprehensive RAG system for Nephoran Intent Operator"
    echo ""
    echo "Options:"
    echo "  deploy          Deploy the complete RAG system (default)"
    echo "  cleanup         Remove existing RAG system resources"
    echo "  verify          Verify existing deployment"
    echo "  help            Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  NAMESPACE                 Kubernetes namespace (default: default)"
    echo "  OPENAI_API_KEY           OpenAI API key (required)"
    echo "  WEAVIATE_API_KEY         Weaviate API key (default: nephoran-rag-key)"
    echo "  KNOWLEDGE_BASE_INIT      Initialize knowledge base (default: true)"
    echo ""
    echo "Examples:"
    echo "  OPENAI_API_KEY=sk-... $0 deploy"
    echo "  NAMESPACE=nephoran $0 cleanup"
    echo "  $0 verify"
}

# Main script logic
case "${1:-deploy}" in
    "deploy")
        deploy_rag_system
        ;;
    "cleanup")
        cleanup
        ;;
    "verify")
        verify_deployment
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
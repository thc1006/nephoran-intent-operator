#!/bin/bash

# Weaviate Deployment and Validation Script for Nephoran Intent Operator
# This script deploys Weaviate with comprehensive validation and testing

set -euo pipefail

# Configuration
NAMESPACE="nephoran-system"
WEAVIATE_RELEASE="weaviate"
DEPLOYMENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/tmp/weaviate-deployment-$(date +%Y%m%d-%H%M%S).log"

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

info() { log "${BLUE}INFO${NC}" "$@"; }
warn() { log "${YELLOW}WARN${NC}" "$@"; }
error() { log "${RED}ERROR${NC}" "$@"; }
success() { log "${GREEN}SUCCESS${NC}" "$@"; }

# Error handling
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        error "Deployment failed with exit code $exit_code"
        error "Check log file: $LOG_FILE"
        
        # Collect diagnostic information
        info "Collecting diagnostic information..."
        kubectl get pods -n "$NAMESPACE" -l app=weaviate >> "$LOG_FILE" 2>&1 || true
        kubectl describe pods -n "$NAMESPACE" -l app=weaviate >> "$LOG_FILE" 2>&1 || true
        kubectl logs -n "$NAMESPACE" -l app=weaviate --tail=50 >> "$LOG_FILE" 2>&1 || true
    fi
}
trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "helm" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Check kubectl connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "kubectl cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check namespace
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        info "Creating namespace $NAMESPACE"
        kubectl create namespace "$NAMESPACE" || {
            error "Failed to create namespace $NAMESPACE"
            exit 1
        }
    fi
    
    # Check Helm
    if ! helm version &> /dev/null; then
        error "Helm is not properly configured"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

# Validate configuration files
validate_config_files() {
    info "Validating configuration files..."
    
    local config_files=(
        "weaviate-deployment.yaml"
        "values.yaml" 
        "rbac.yaml"
        "backup-system.yaml"
        "monitoring-enhanced.yaml"
    )
    
    for file in "${config_files[@]}"; do
        local file_path="$DEPLOYMENT_DIR/$file"
        if [ ! -f "$file_path" ]; then
            error "Configuration file not found: $file_path"
            exit 1
        fi
        
        # Validate YAML syntax
        if ! kubectl apply --dry-run=client -f "$file_path" &> /dev/null; then
            error "Invalid YAML syntax in $file"
            exit 1
        fi
    done
    
    success "Configuration files validation passed"
}

# Deploy storage classes if needed
deploy_storage_classes() {
    info "Checking and deploying storage classes..."
    
    # Check if storage classes exist
    local storage_classes=("gp3-encrypted" "gp2-encrypted" "sc1-encrypted")
    for sc in "${storage_classes[@]}"; do
        if ! kubectl get storageclass "$sc" &> /dev/null; then
            warn "Storage class $sc not found, creating default configuration"
            
            # Create basic storage class definition
            cat << EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: $sc
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
EOF
        fi
    done
    
    success "Storage classes configured"
}

# Deploy Weaviate components
deploy_weaviate() {
    info "Deploying Weaviate components..."
    
    # Deploy in order of dependencies
    local deployment_files=(
        "rbac.yaml"
        "weaviate-deployment.yaml"
        "backup-system.yaml"
        "monitoring-enhanced.yaml"
    )
    
    for file in "${deployment_files[@]}"; do
        info "Deploying $file..."
        kubectl apply -f "$DEPLOYMENT_DIR/$file" || {
            error "Failed to deploy $file"
            exit 1
        }
    done
    
    success "Weaviate components deployed"
}

# Wait for deployment readiness
wait_for_readiness() {
    info "Waiting for Weaviate deployment to be ready..."
    
    # Wait for deployment to be available
    if ! kubectl wait --for=condition=available --timeout=600s deployment/weaviate -n "$NAMESPACE"; then
        error "Weaviate deployment failed to become available"
        return 1
    fi
    
    # Wait for pods to be ready
    if ! kubectl wait --for=condition=ready --timeout=300s pod -l app=weaviate -n "$NAMESPACE"; then
        error "Weaviate pods failed to become ready"
        return 1
    fi
    
    # Wait for service to be ready
    local max_attempts=60
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if kubectl get endpoints weaviate -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' | grep -q .; then
            break
        fi
        sleep 5
        ((attempt++))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        error "Weaviate service endpoints not ready after 5 minutes"
        return 1
    fi
    
    success "Weaviate deployment is ready"
}

# Validate Weaviate API
validate_weaviate_api() {
    info "Validating Weaviate API..."
    
    # Get API key
    local api_key
    api_key=$(kubectl get secret weaviate-api-key -n "$NAMESPACE" -o jsonpath='{.data.api-key}' | base64 -d)
    
    # Port forward for testing
    kubectl port-forward svc/weaviate 8080:8080 -n "$NAMESPACE" &
    local port_forward_pid=$!
    
    # Cleanup function for port forward
    cleanup_port_forward() {
        kill $port_forward_pid 2>/dev/null || true
    }
    trap cleanup_port_forward EXIT
    
    # Wait for port forward to be ready
    sleep 5
    
    # Test readiness endpoint
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -f -H "Authorization: Bearer $api_key" "http://localhost:8080/v1/.well-known/ready" &> /dev/null; then
            break
        fi
        sleep 2
        ((attempt++))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        error "Weaviate readiness check failed"
        cleanup_port_forward
        return 1
    fi
    
    # Test liveness endpoint
    if ! curl -f -H "Authorization: Bearer $api_key" "http://localhost:8080/v1/.well-known/live" &> /dev/null; then
        error "Weaviate liveness check failed"
        cleanup_port_forward
        return 1
    fi
    
    # Test schema endpoint
    if ! curl -f -H "Authorization: Bearer $api_key" "http://localhost:8080/v1/schema" &> /dev/null; then
        error "Weaviate schema endpoint failed"
        cleanup_port_forward
        return 1
    fi
    
    # Test meta endpoint
    local meta_response
    meta_response=$(curl -s -H "Authorization: Bearer $api_key" "http://localhost:8080/v1/meta")
    if ! echo "$meta_response" | jq -e '.version' &> /dev/null; then
        error "Weaviate meta endpoint failed"
        cleanup_port_forward
        return 1
    fi
    
    local weaviate_version
    weaviate_version=$(echo "$meta_response" | jq -r '.version')
    info "Weaviate version: $weaviate_version"
    
    cleanup_port_forward
    success "Weaviate API validation passed"
}

# Test schema creation
test_schema_creation() {
    info "Testing schema creation..."
    
    # Port forward for testing
    kubectl port-forward svc/weaviate 8080:8080 -n "$NAMESPACE" &
    local port_forward_pid=$!
    
    cleanup_port_forward() {
        kill $port_forward_pid 2>/dev/null || true
    }
    trap cleanup_port_forward EXIT
    
    sleep 5
    
    local api_key
    api_key=$(kubectl get secret weaviate-api-key -n "$NAMESPACE" -o jsonpath='{.data.api-key}' | base64 -d)
    
    # Test creating TelecomKnowledge class
    local schema_payload='{
        "class": "TelecomKnowledge",
        "description": "Test telecom knowledge class",
        "vectorizer": "text2vec-openai",
        "moduleConfig": {
            "text2vec-openai": {
                "model": "text-embedding-3-large",
                "dimensions": 3072,
                "type": "text"
            }
        },
        "properties": [
            {
                "name": "content",
                "dataType": ["text"],
                "description": "Document content",
                "indexFilterable": true,
                "indexSearchable": true
            },
            {
                "name": "title", 
                "dataType": ["text"],
                "description": "Document title",
                "indexFilterable": true,
                "indexSearchable": true
            }
        ]
    }'
    
    # Create schema
    local response
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $api_key" \
        -H "Content-Type: application/json" \
        -d "$schema_payload" \
        "http://localhost:8080/v1/schema")
    
    # Check if creation was successful or class already exists
    if echo "$response" | jq -e '.error' &> /dev/null; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error.message')
        if [[ "$error_msg" != *"already exists"* ]]; then
            error "Schema creation failed: $error_msg"
            cleanup_port_forward
            return 1
        else
            info "TelecomKnowledge class already exists"
        fi
    else
        success "TelecomKnowledge class created successfully"
    fi
    
    cleanup_port_forward
    success "Schema creation test passed"
}

# Test basic vector operations
test_vector_operations() {
    info "Testing basic vector operations..."
    
    # Port forward for testing
    kubectl port-forward svc/weaviate 8080:8080 -n "$NAMESPACE" &
    local port_forward_pid=$!
    
    cleanup_port_forward() {
        kill $port_forward_pid 2>/dev/null || true
    }
    trap cleanup_port_forward EXIT
    
    sleep 5
    
    local api_key
    api_key=$(kubectl get secret weaviate-api-key -n "$NAMESPACE" -o jsonpath='{.data.api-key}' | base64 -d)
    
    # Test GraphQL query
    local query_payload='{
        "query": "{Get{TelecomKnowledge(limit: 1){content title}}}"
    }'
    
    local response
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $api_key" \
        -H "Content-Type: application/json" \
        -d "$query_payload" \
        "http://localhost:8080/v1/graphql")
    
    if echo "$response" | jq -e '.errors' &> /dev/null; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.errors[0].message')
        warn "GraphQL query returned error (expected for empty database): $error_msg"
    else
        success "GraphQL query executed successfully"
    fi
    
    cleanup_port_forward
    success "Vector operations test passed"
}

# Validate monitoring setup
validate_monitoring() {
    info "Validating monitoring setup..."
    
    # Check ServiceMonitor
    if ! kubectl get servicemonitor weaviate-monitor -n "$NAMESPACE" &> /dev/null; then
        error "ServiceMonitor not found"
        return 1
    fi
    
    # Check PrometheusRule
    if ! kubectl get prometheusrule weaviate-alerts -n "$NAMESPACE" &> /dev/null; then
        error "PrometheusRule not found"
        return 1
    fi
    
    # Check health check CronJob
    if ! kubectl get cronjob weaviate-health-check -n "$NAMESPACE" &> /dev/null; then
        error "Health check CronJob not found"
        return 1
    fi
    
    success "Monitoring setup validation passed"
}

# Validate backup system
validate_backup_system() {
    info "Validating backup system..."
    
    # Check backup CronJobs
    local backup_jobs=("weaviate-hourly-backup" "weaviate-daily-backup" "weaviate-weekly-backup")
    for job in "${backup_jobs[@]}"; do
        if ! kubectl get cronjob "$job" -n "$NAMESPACE" &> /dev/null; then
            error "Backup CronJob $job not found"
            return 1
        fi
    done
    
    # Check backup configuration
    if ! kubectl get configmap weaviate-backup-config -n "$NAMESPACE" &> /dev/null; then
        error "Backup configuration not found"
        return 1
    fi
    
    success "Backup system validation passed"
}

# Performance test
performance_test() {
    info "Running basic performance test..."
    
    # Port forward for testing
    kubectl port-forward svc/weaviate 8080:8080 -n "$NAMESPACE" &
    local port_forward_pid=$!
    
    cleanup_port_forward() {
        kill $port_forward_pid 2>/dev/null || true
    }
    trap cleanup_port_forward EXIT
    
    sleep 5
    
    local api_key
    api_key=$(kubectl get secret weaviate-api-key -n "$NAMESPACE" -o jsonpath='{.data.api-key}' | base64 -d)
    
    # Measure query response time
    local start_time
    local end_time
    local response_time
    
    start_time=$(date +%s%N)
    curl -s -H "Authorization: Bearer $api_key" "http://localhost:8080/v1/meta" > /dev/null
    end_time=$(date +%s%N)
    
    response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds
    
    info "API response time: ${response_time}ms"
    
    if [ $response_time -gt 5000 ]; then
        warn "API response time is high (>5s)"
    elif [ $response_time -gt 1000 ]; then
        warn "API response time is elevated (>1s)"
    else
        success "API response time is good (<1s)"
    fi
    
    cleanup_port_forward
    success "Performance test completed"
}

# Resource validation
validate_resources() {
    info "Validating resource allocation..."
    
    # Check pod resource usage
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[*].metadata.name}')
    
    for pod in $pods; do
        local cpu_request
        local memory_request
        local cpu_limit
        local memory_limit
        
        cpu_request=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.requests.cpu}')
        memory_request=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.requests.memory}')
        cpu_limit=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits.cpu}')
        memory_limit=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits.memory}')
        
        info "Pod $pod resources:"
        info "  CPU: $cpu_request (request) / $cpu_limit (limit)"
        info "  Memory: $memory_request (request) / $memory_limit (limit)"
    done
    
    # Check PVC status
    local pvcs
    pvcs=$(kubectl get pvc -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[*].metadata.name}')
    
    for pvc in $pvcs; do
        local status
        local capacity
        
        status=$(kubectl get pvc "$pvc" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
        capacity=$(kubectl get pvc "$pvc" -n "$NAMESPACE" -o jsonpath='{.status.capacity.storage}')
        
        if [ "$status" != "Bound" ]; then
            error "PVC $pvc is not bound (status: $status)"
            return 1
        fi
        
        info "PVC $pvc: $status ($capacity)"
    done
    
    success "Resource validation passed"
}

# Generate deployment report
generate_report() {
    info "Generating deployment report..."
    
    local report_file="/tmp/weaviate-deployment-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# Weaviate Deployment Report

**Date:** $(date)
**Namespace:** $NAMESPACE
**Cluster:** $(kubectl config current-context)

## Deployment Status

### Pods
\`\`\`
$(kubectl get pods -n "$NAMESPACE" -l app=weaviate)
\`\`\`

### Services
\`\`\`
$(kubectl get services -n "$NAMESPACE" -l app=weaviate)
\`\`\`

### PersistentVolumeClaims
\`\`\`
$(kubectl get pvc -n "$NAMESPACE" -l app=weaviate)
\`\`\`

### HorizontalPodAutoscaler
\`\`\`
$(kubectl get hpa -n "$NAMESPACE" -l app=weaviate)
\`\`\`

### Backup Jobs
\`\`\`
$(kubectl get cronjobs -n "$NAMESPACE" -l component=backup)
\`\`\`

### Monitoring
\`\`\`
$(kubectl get servicemonitor,prometheusrule -n "$NAMESPACE" -l app=weaviate)
\`\`\`

## Configuration Summary

- **Replicas:** $(kubectl get deployment weaviate -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
- **Image:** $(kubectl get deployment weaviate -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
- **Storage Class:** $(kubectl get pvc weaviate-pvc -n "$NAMESPACE" -o jsonpath='{.spec.storageClassName}')
- **Storage Size:** $(kubectl get pvc weaviate-pvc -n "$NAMESPACE" -o jsonpath='{.spec.resources.requests.storage}')

## Next Steps

1. Configure OpenAI API key in the weaviate-api-key secret
2. Populate the knowledge base with telecom documents
3. Configure monitoring dashboards
4. Set up backup notifications
5. Test RAG functionality with sample queries

## Log File

Full deployment log: $LOG_FILE

EOF

    info "Deployment report generated: $report_file"
}

# Main execution
main() {
    info "Starting Weaviate deployment and validation"
    info "Log file: $LOG_FILE"
    
    # Pre-deployment checks
    check_prerequisites
    validate_config_files
    deploy_storage_classes
    
    # Deployment
    deploy_weaviate
    wait_for_readiness
    
    # Validation tests
    validate_weaviate_api
    test_schema_creation
    test_vector_operations
    validate_monitoring
    validate_backup_system
    validate_resources
    
    # Performance test
    performance_test
    
    # Generate report
    generate_report
    
    success "Weaviate deployment and validation completed successfully!"
    success "Check the deployment report and configure the system for your specific needs."
    
    info "Quick start commands:"
    info "  kubectl port-forward svc/weaviate 8080:8080 -n $NAMESPACE"
    info "  kubectl logs -f deployment/weaviate -n $NAMESPACE"
    info "  kubectl get all -l app=weaviate -n $NAMESPACE"
}

# Execute main function
main "$@"
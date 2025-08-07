#!/bin/bash

# O2 Infrastructure Management Service Production Readiness Checklist
# Validates O2 IMS deployment for production readiness
# Usage: ./o2-production-readiness-check.sh [namespace] [endpoint]

set -euo pipefail

# Configuration
NAMESPACE="${1:-nephoran-system}"
O2_ENDPOINT="${2:-https://o2ims.nephoran.svc.cluster.local:8080}"
TIMEOUT=30
RETRY_COUNT=3
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_section() {
    echo -e "\n${BLUE}==================== $1 ====================${NC}"
}

# Test results tracking
declare -A test_results
total_tests=0
passed_tests=0
failed_tests=0
warning_tests=0

# Function to record test result
record_test() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    test_results["$test_name"]="$status:$message"
    total_tests=$((total_tests + 1))
    
    case $status in
        "PASS")
            log_success "$test_name: $message"
            passed_tests=$((passed_tests + 1))
            ;;
        "FAIL")
            log_error "$test_name: $message"
            failed_tests=$((failed_tests + 1))
            ;;
        "WARN")
            log_warning "$test_name: $message"
            warning_tests=$((warning_tests + 1))
            ;;
    esac
}

# Function to check if command exists
check_command() {
    local cmd="$1"
    if ! command -v "$cmd" &> /dev/null; then
        log_error "$cmd is not installed or not in PATH"
        exit 1
    fi
}

# Function to make HTTP request with retry
http_request() {
    local url="$1"
    local method="${2:-GET}"
    local expected_status="${3:-200}"
    local max_attempts="$RETRY_COUNT"
    
    for ((i=1; i<=max_attempts; i++)); do
        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Attempt $i/$max_attempts: $method $url"
        fi
        
        local response
        local status_code
        
        response=$(curl -s -w "%{http_code}" -X "$method" \
            -H "Accept: application/json" \
            -H "Content-Type: application/json" \
            --connect-timeout "$TIMEOUT" \
            --max-time "$TIMEOUT" \
            "$url" 2>/dev/null || echo "000")
        
        status_code="${response: -3}"
        
        if [[ "$status_code" == "$expected_status" ]]; then
            echo "${response%???}"  # Remove last 3 characters (status code)
            return 0
        fi
        
        if [[ $i -lt $max_attempts ]]; then
            sleep 2
        fi
    done
    
    return 1
}

# Function to check Kubernetes resource
check_k8s_resource() {
    local resource_type="$1"
    local resource_name="$2"
    local namespace="$3"
    
    kubectl get "$resource_type" "$resource_name" -n "$namespace" &> /dev/null
}

log_section "O2 IMS Production Readiness Assessment"
log_info "Namespace: $NAMESPACE"
log_info "Endpoint: $O2_ENDPOINT"
log_info "Starting comprehensive production readiness validation..."

# Check prerequisites
log_section "Prerequisites Check"

check_command "kubectl"
check_command "curl"
check_command "jq"

# Check kubectl connectivity
if kubectl cluster-info &> /dev/null; then
    record_test "Kubernetes Connectivity" "PASS" "kubectl can connect to cluster"
else
    record_test "Kubernetes Connectivity" "FAIL" "Cannot connect to Kubernetes cluster"
    exit 1
fi

# Check namespace existence
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    record_test "Namespace Existence" "PASS" "Namespace $NAMESPACE exists"
else
    record_test "Namespace Existence" "FAIL" "Namespace $NAMESPACE does not exist"
    exit 1
fi

# Deployment and Pod Health Check
log_section "Deployment Health Validation"

# Check O2 IMS deployment
if check_k8s_resource "deployment" "o2ims-api" "$NAMESPACE"; then
    # Get deployment status
    ready_replicas=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    desired_replicas=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
    
    if [[ "$ready_replicas" == "$desired_replicas" ]] && [[ "$ready_replicas" -gt 0 ]]; then
        record_test "O2 IMS Deployment" "PASS" "All $ready_replicas/$desired_replicas replicas ready"
    else
        record_test "O2 IMS Deployment" "FAIL" "Only $ready_replicas/$desired_replicas replicas ready"
    fi
else
    record_test "O2 IMS Deployment" "FAIL" "O2 IMS deployment not found"
fi

# Check pod health
o2_pods=$(kubectl get pods -n "$NAMESPACE" -l app=o2ims-api --no-headers 2>/dev/null || echo "")
if [[ -n "$o2_pods" ]]; then
    running_pods=$(echo "$o2_pods" | grep -c "Running" || echo "0")
    total_pods=$(echo "$o2_pods" | wc -l)
    
    if [[ "$running_pods" == "$total_pods" ]] && [[ "$running_pods" -gt 0 ]]; then
        record_test "Pod Health" "PASS" "All $running_pods/$total_pods pods running"
    else
        record_test "Pod Health" "FAIL" "Only $running_pods/$total_pods pods running"
        
        # Show pod details for debugging
        if [[ "$VERBOSE" == "true" ]]; then
            kubectl get pods -n "$NAMESPACE" -l app=o2ims-api
        fi
    fi
else
    record_test "Pod Health" "FAIL" "No O2 IMS pods found"
fi

# Service and Network Connectivity
log_section "Network Connectivity Validation"

# Check service existence
if check_k8s_resource "service" "o2ims-service" "$NAMESPACE"; then
    record_test "Service Existence" "PASS" "O2 IMS service exists"
    
    # Get service details
    service_type=$(kubectl get service o2ims-service -n "$NAMESPACE" -o jsonpath='{.spec.type}' 2>/dev/null || echo "Unknown")
    cluster_ip=$(kubectl get service o2ims-service -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "None")
    
    record_test "Service Configuration" "PASS" "Type: $service_type, ClusterIP: $cluster_ip"
else
    record_test "Service Existence" "FAIL" "O2 IMS service not found"
fi

# Check ingress (if exists)
if check_k8s_resource "ingress" "o2ims-ingress" "$NAMESPACE"; then
    hosts=$(kubectl get ingress o2ims-ingress -n "$NAMESPACE" -o jsonpath='{.spec.rules[*].host}' 2>/dev/null || echo "")
    record_test "Ingress Configuration" "PASS" "Ingress exists with hosts: $hosts"
else
    record_test "Ingress Configuration" "WARN" "No ingress found - external access may be limited"
fi

# API Health and Functionality Check
log_section "API Health and Functionality"

# Test service information endpoint
if service_info=$(http_request "$O2_ENDPOINT/o2ims/v1/" "GET" "200"); then
    api_version=$(echo "$service_info" | jq -r '.apiVersion // "unknown"' 2>/dev/null || echo "unknown")
    specification=$(echo "$service_info" | jq -r '.specification // "unknown"' 2>/dev/null || echo "unknown")
    
    record_test "Service Information API" "PASS" "API Version: $api_version"
    
    if [[ "$specification" == *"O-RAN"* ]]; then
        record_test "O-RAN Specification" "PASS" "Specification: $specification"
    else
        record_test "O-RAN Specification" "WARN" "Specification not clearly O-RAN compliant: $specification"
    fi
else
    record_test "Service Information API" "FAIL" "Cannot access service information endpoint"
fi

# Test health endpoint
if http_request "$O2_ENDPOINT/health" "GET" "200" > /dev/null; then
    record_test "Health Endpoint" "PASS" "Health endpoint responding"
else
    record_test "Health Endpoint" "FAIL" "Health endpoint not responding"
fi

# Test readiness endpoint
if http_request "$O2_ENDPOINT/ready" "GET" "200" > /dev/null; then
    record_test "Readiness Endpoint" "PASS" "Readiness endpoint responding"
else
    record_test "Readiness Endpoint" "FAIL" "Readiness endpoint not responding"
fi

# Test core API endpoints
api_endpoints=(
    "resourcePools:GET"
    "resourceTypes:GET"
    "resources:GET"
    "deployments:GET"
    "subscriptions:GET"
    "alarms:GET"
)

for endpoint_spec in "${api_endpoints[@]}"; do
    endpoint="${endpoint_spec%:*}"
    method="${endpoint_spec#*:}"
    
    if http_request "$O2_ENDPOINT/o2ims/v1/$endpoint" "$method" "200" > /dev/null; then
        record_test "API Endpoint $endpoint" "PASS" "$method request successful"
    else
        record_test "API Endpoint $endpoint" "FAIL" "$method request failed"
    fi
done

# Configuration and Security Validation
log_section "Configuration and Security"

# Check ConfigMaps
config_maps=("o2ims-config" "o2ims-provider-config")
for cm in "${config_maps[@]}"; do
    if check_k8s_resource "configmap" "$cm" "$NAMESPACE"; then
        record_test "ConfigMap $cm" "PASS" "ConfigMap exists"
    else
        record_test "ConfigMap $cm" "WARN" "ConfigMap $cm not found - may use defaults"
    fi
done

# Check Secrets
secrets=("o2ims-oauth-config" "o2ims-database-credentials" "o2ims-tls-certs")
for secret in "${secrets[@]}"; do
    if check_k8s_resource "secret" "$secret" "$NAMESPACE"; then
        record_test "Secret $secret" "PASS" "Secret exists"
    else
        record_test "Secret $secret" "WARN" "Secret $secret not found"
    fi
done

# Check RBAC
rbac_resources=("serviceaccount/o2ims-service-account" "clusterrole/o2ims-cluster-role" "clusterrolebinding/o2ims-cluster-role-binding")
for rbac in "${rbac_resources[@]}"; do
    resource_type="${rbac%/*}"
    resource_name="${rbac#*/}"
    
    if [[ "$resource_type" == "clusterrole" ]] || [[ "$resource_type" == "clusterrolebinding" ]]; then
        # Cluster-scoped resources
        if kubectl get "$resource_type" "$resource_name" &> /dev/null; then
            record_test "RBAC $resource_name" "PASS" "$resource_type exists"
        else
            record_test "RBAC $resource_name" "WARN" "$resource_type not found"
        fi
    else
        # Namespaced resources
        if check_k8s_resource "$resource_type" "$resource_name" "$NAMESPACE"; then
            record_test "RBAC $resource_name" "PASS" "$resource_type exists"
        else
            record_test "RBAC $resource_name" "WARN" "$resource_type not found"
        fi
    fi
done

# Resource Requirements and Limits
log_section "Resource Management"

# Check resource requests and limits
deployment_resources=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o json 2>/dev/null | jq '.spec.template.spec.containers[0].resources // {}' 2>/dev/null || echo "{}")

if [[ "$deployment_resources" != "{}" ]]; then
    requests=$(echo "$deployment_resources" | jq '.requests // {}')
    limits=$(echo "$deployment_resources" | jq '.limits // {}')
    
    if [[ "$requests" != "{}" ]]; then
        cpu_request=$(echo "$requests" | jq -r '.cpu // "not set"')
        memory_request=$(echo "$requests" | jq -r '.memory // "not set"')
        record_test "Resource Requests" "PASS" "CPU: $cpu_request, Memory: $memory_request"
    else
        record_test "Resource Requests" "WARN" "No resource requests configured"
    fi
    
    if [[ "$limits" != "{}" ]]; then
        cpu_limit=$(echo "$limits" | jq -r '.cpu // "not set"')
        memory_limit=$(echo "$limits" | jq -r '.memory // "not set"')
        record_test "Resource Limits" "PASS" "CPU: $cpu_limit, Memory: $memory_limit"
    else
        record_test "Resource Limits" "WARN" "No resource limits configured"
    fi
else
    record_test "Resource Configuration" "WARN" "No resource configuration found"
fi

# Check Horizontal Pod Autoscaler
if check_k8s_resource "hpa" "o2ims-hpa" "$NAMESPACE"; then
    min_replicas=$(kubectl get hpa o2ims-hpa -n "$NAMESPACE" -o jsonpath='{.spec.minReplicas}' 2>/dev/null || echo "1")
    max_replicas=$(kubectl get hpa o2ims-hpa -n "$NAMESPACE" -o jsonpath='{.spec.maxReplicas}' 2>/dev/null || echo "unknown")
    record_test "Auto-scaling (HPA)" "PASS" "Min: $min_replicas, Max: $max_replicas"
else
    record_test "Auto-scaling (HPA)" "WARN" "No HPA configured - manual scaling only"
fi

# Monitoring and Observability
log_section "Monitoring and Observability"

# Check if metrics endpoint is accessible
if http_request "$O2_ENDPOINT/metrics" "GET" "200" > /dev/null; then
    record_test "Metrics Endpoint" "PASS" "Prometheus metrics available"
else
    record_test "Metrics Endpoint" "WARN" "Metrics endpoint not accessible"
fi

# Check ServiceMonitor (Prometheus)
if check_k8s_resource "servicemonitor" "o2ims-service-monitor" "$NAMESPACE"; then
    record_test "ServiceMonitor" "PASS" "Prometheus ServiceMonitor configured"
else
    record_test "ServiceMonitor" "WARN" "No ServiceMonitor found - metrics may not be scraped"
fi

# Check if Jaeger tracing is configured
jaeger_pods=$(kubectl get pods -n observability -l app=jaeger --no-headers 2>/dev/null | wc -l || echo "0")
if [[ "$jaeger_pods" -gt 0 ]]; then
    record_test "Distributed Tracing" "PASS" "Jaeger pods found in observability namespace"
else
    record_test "Distributed Tracing" "WARN" "No Jaeger pods found - tracing may not be available"
fi

# Persistence and Data Storage
log_section "Data Persistence"

# Check PersistentVolumeClaims
pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | grep -c "Bound" || echo "0")
if [[ "$pvcs" -gt 0 ]]; then
    record_test "Persistent Storage" "PASS" "$pvcs PVC(s) bound"
else
    record_test "Persistent Storage" "WARN" "No PVCs found - using ephemeral storage"
fi

# Check database connectivity (if external database is used)
if check_k8s_resource "secret" "o2ims-database-credentials" "$NAMESPACE"; then
    # Database credentials exist, likely using external database
    record_test "Database Configuration" "PASS" "Database credentials configured"
else
    record_test "Database Configuration" "WARN" "No database credentials found - using in-memory storage"
fi

# Backup and Recovery
log_section "Backup and Recovery"

# Check if backup CronJob exists
if check_k8s_resource "cronjob" "o2ims-backup" "$NAMESPACE"; then
    schedule=$(kubectl get cronjob o2ims-backup -n "$NAMESPACE" -o jsonpath='{.spec.schedule}' 2>/dev/null || echo "unknown")
    record_test "Backup CronJob" "PASS" "Backup scheduled: $schedule"
else
    record_test "Backup CronJob" "WARN" "No backup CronJob configured"
fi

# Check if VolumeSnapshot capability exists
if kubectl get crd volumesnapshots.snapshot.storage.k8s.io &> /dev/null; then
    record_test "Volume Snapshots" "PASS" "VolumeSnapshot CRD available"
else
    record_test "Volume Snapshots" "WARN" "VolumeSnapshot CRD not available"
fi

# Performance and Scalability
log_section "Performance Validation"

# Check pod resource usage
pod_metrics=$(kubectl top pods -n "$NAMESPACE" -l app=o2ims-api --no-headers 2>/dev/null || echo "")
if [[ -n "$pod_metrics" ]]; then
    record_test "Resource Monitoring" "PASS" "Pod metrics available via kubectl top"
else
    record_test "Resource Monitoring" "WARN" "Pod metrics not available - metrics-server may not be installed"
fi

# Test API response time
start_time=$(date +%s%3N)
if http_request "$O2_ENDPOINT/o2ims/v1/" "GET" "200" > /dev/null; then
    end_time=$(date +%s%3N)
    response_time=$((end_time - start_time))
    
    if [[ $response_time -lt 1000 ]]; then
        record_test "API Response Time" "PASS" "${response_time}ms (< 1000ms)"
    elif [[ $response_time -lt 3000 ]]; then
        record_test "API Response Time" "WARN" "${response_time}ms (slow but acceptable)"
    else
        record_test "API Response Time" "FAIL" "${response_time}ms (too slow)"
    fi
else
    record_test "API Response Time" "FAIL" "Cannot measure - API not responding"
fi

# High Availability
log_section "High Availability"

# Check if multiple replicas are configured
replicas=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
if [[ "$replicas" -gt 1 ]]; then
    record_test "High Availability" "PASS" "$replicas replicas configured"
else
    record_test "High Availability" "WARN" "Only $replicas replica - no HA"
fi

# Check PodDisruptionBudget
if check_k8s_resource "poddisruptionbudget" "o2ims-pdb" "$NAMESPACE"; then
    record_test "Pod Disruption Budget" "PASS" "PDB configured for controlled disruptions"
else
    record_test "Pod Disruption Budget" "WARN" "No PDB - disruptions may affect availability"
fi

# Check anti-affinity rules
affinity=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o json 2>/dev/null | jq '.spec.template.spec.affinity // {}' 2>/dev/null || echo "{}")
if [[ "$affinity" != "{}" ]]; then
    record_test "Pod Anti-Affinity" "PASS" "Affinity rules configured"
else
    record_test "Pod Anti-Affinity" "WARN" "No affinity rules - pods may be scheduled on same node"
fi

# Security Hardening
log_section "Security Validation"

# Check security context
security_context=$(kubectl get deployment o2ims-api -n "$NAMESPACE" -o json 2>/dev/null | jq '.spec.template.spec.securityContext // {}' 2>/dev/null || echo "{}")
if [[ "$security_context" != "{}" ]]; then
    run_as_non_root=$(echo "$security_context" | jq -r '.runAsNonRoot // false')
    if [[ "$run_as_non_root" == "true" ]]; then
        record_test "Security Context" "PASS" "Running as non-root user"
    else
        record_test "Security Context" "WARN" "Not explicitly running as non-root"
    fi
else
    record_test "Security Context" "WARN" "No security context configured"
fi

# Check NetworkPolicies
network_policies=$(kubectl get networkpolicy -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")
if [[ "$network_policies" -gt 0 ]]; then
    record_test "Network Policies" "PASS" "$network_policies network policy(ies) configured"
else
    record_test "Network Policies" "WARN" "No network policies - network access not restricted"
fi

# Check if TLS is enabled
tls_secret=$(kubectl get secrets -n "$NAMESPACE" --no-headers 2>/dev/null | grep -c "tls" || echo "0")
if [[ "$tls_secret" -gt 0 ]]; then
    record_test "TLS Configuration" "PASS" "TLS secret(s) found"
else
    record_test "TLS Configuration" "WARN" "No TLS secrets found"
fi

# Generate Final Report
log_section "Production Readiness Report"

echo -e "\n${BLUE}Summary:${NC}"
echo "Total Tests: $total_tests"
echo -e "${GREEN}Passed: $passed_tests${NC}"
echo -e "${YELLOW}Warnings: $warning_tests${NC}"
echo -e "${RED}Failed: $failed_tests${NC}"

# Calculate readiness score
readiness_score=$(( (passed_tests * 100) / total_tests ))
warning_penalty=$(( warning_tests * 5 ))
adjusted_score=$(( readiness_score - warning_penalty ))

if [[ $adjusted_score -lt 0 ]]; then
    adjusted_score=0
fi

echo -e "\n${BLUE}Production Readiness Score: ${NC}${adjusted_score}%"

# Determine readiness level
if [[ $failed_tests -eq 0 ]] && [[ $adjusted_score -ge 90 ]]; then
    echo -e "${GREEN}✓ PRODUCTION READY${NC}"
    echo "The O2 IMS deployment meets production readiness standards."
elif [[ $failed_tests -eq 0 ]] && [[ $adjusted_score -ge 70 ]]; then
    echo -e "${YELLOW}⚠ PRODUCTION READY WITH WARNINGS${NC}"
    echo "The deployment is production ready but has areas for improvement."
elif [[ $failed_tests -le 2 ]] && [[ $adjusted_score -ge 60 ]]; then
    echo -e "${YELLOW}⚠ NEEDS ATTENTION${NC}"
    echo "Address the failed tests before production deployment."
else
    echo -e "${RED}✗ NOT PRODUCTION READY${NC}"
    echo "Significant issues must be resolved before production deployment."
fi

# Recommendations
echo -e "\n${BLUE}Recommendations:${NC}"

if [[ $warning_tests -gt 0 ]]; then
    echo "• Address warnings to improve production readiness"
fi

if [[ $failed_tests -gt 0 ]]; then
    echo "• Fix all failed tests before production deployment"
fi

echo "• Regularly run this assessment to maintain production readiness"
echo "• Consider implementing additional monitoring and alerting"
echo "• Establish backup and disaster recovery procedures"
echo "• Document operational procedures and runbooks"

# Exit code based on readiness
if [[ $failed_tests -eq 0 ]] && [[ $adjusted_score -ge 70 ]]; then
    exit 0
else
    exit 1
fi
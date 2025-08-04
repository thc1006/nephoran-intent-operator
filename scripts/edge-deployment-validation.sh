#!/bin/bash
# Edge Computing Deployment Validation Script
# Nephoran Intent Operator - Phase 4 Enterprise Architecture

set -e

EDGE_NAMESPACE="nephoran-edge"
VALIDATION_DATE=$(date +%Y%m%d-%H%M%S)
VALIDATION_LOG="/tmp/edge-validation-${VALIDATION_DATE}.log"

echo "=== Nephoran Edge Computing Deployment Validation ==="
echo "Validation ID: EDGE-VAL-${VALIDATION_DATE}"
echo "Logging to: ${VALIDATION_LOG}"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${VALIDATION_LOG}"
}

# Function to check command success
check_success() {
    if [ $? -eq 0 ]; then
        log "✅ $1 - SUCCESS"
        return 0
    else
        log "❌ $1 - FAILED"
        return 1
    fi
}

# Phase 1: Prerequisites Validation
log "Phase 1: Prerequisites Validation"

# Check Kubernetes cluster access
log "Checking Kubernetes cluster access..."
kubectl cluster-info > /dev/null 2>&1
check_success "Kubernetes cluster connectivity"

# Check required storage classes
log "Checking storage classes..."
kubectl get storageclass fast-ssd > /dev/null 2>&1
check_success "Storage class 'fast-ssd' availability"

# Check node labels for edge nodes
log "Checking edge node labels..."
EDGE_NODES=$(kubectl get nodes -l nephoran.io/node-type=edge --no-headers | wc -l)
if [ "$EDGE_NODES" -gt 0 ]; then
    log "✅ Found ${EDGE_NODES} edge nodes"
else
    log "❌ No edge nodes found with label nephoran.io/node-type=edge"
fi

# Phase 2: Namespace and RBAC Validation
log "Phase 2: Namespace and RBAC Validation"

# Check namespace
kubectl get namespace ${EDGE_NAMESPACE} > /dev/null 2>&1
check_success "Edge namespace '${EDGE_NAMESPACE}' exists"

# Check service accounts
REQUIRED_SA=("edge-discovery" "edge-ml" "edge-cache" "edge-local-ric" "edge-cloud-sync" "edge-metrics-collector" "edge-intent-processor")
for sa in "${REQUIRED_SA[@]}"; do
    kubectl get serviceaccount "${sa}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1
    check_success "Service account '${sa}' exists"
done

# Check RBAC
kubectl get clusterrole edge-controller > /dev/null 2>&1
check_success "ClusterRole 'edge-controller' exists"

kubectl get clusterrolebinding edge-controller > /dev/null 2>&1
check_success "ClusterRoleBinding 'edge-controller' exists"

# Phase 3: Core Services Validation
log "Phase 3: Core Services Validation"

# Check deployments
REQUIRED_DEPLOYMENTS=("edge-discovery-service" "edge-ml-service" "edge-cache-service" "edge-local-ric" "edge-cloud-sync" "edge-intent-processor")
for deployment in "${REQUIRED_DEPLOYMENTS[@]}"; do
    kubectl get deployment "${deployment}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1
    if check_success "Deployment '${deployment}' exists"; then
        # Check if deployment is ready
        READY_REPLICAS=$(kubectl get deployment "${deployment}" -n ${EDGE_NAMESPACE} -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        DESIRED_REPLICAS=$(kubectl get deployment "${deployment}" -n ${EDGE_NAMESPACE} -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "1")
        
        if [ "${READY_REPLICAS}" = "${DESIRED_REPLICAS}" ]; then
            log "✅ Deployment '${deployment}' is ready (${READY_REPLICAS}/${DESIRED_REPLICAS})"
        else
            log "⚠️ Deployment '${deployment}' not fully ready (${READY_REPLICAS}/${DESIRED_REPLICAS})"
        fi
    fi
done

# Check DaemonSet
kubectl get daemonset edge-metrics-collector -n ${EDGE_NAMESPACE} > /dev/null 2>&1
if check_success "DaemonSet 'edge-metrics-collector' exists"; then
    DESIRED_DS=$(kubectl get daemonset edge-metrics-collector -n ${EDGE_NAMESPACE} -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
    READY_DS=$(kubectl get daemonset edge-metrics-collector -n ${EDGE_NAMESPACE} -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
    
    if [ "${READY_DS}" = "${DESIRED_DS}" ]; then
        log "✅ DaemonSet 'edge-metrics-collector' is ready (${READY_DS}/${DESIRED_DS})"
    else
        log "⚠️ DaemonSet 'edge-metrics-collector' not fully ready (${READY_DS}/${DESIRED_DS})"
    fi
fi

# Phase 4: Services and Networking Validation
log "Phase 4: Services and Networking Validation"

# Check services
REQUIRED_SERVICES=("edge-discovery-service" "edge-ml-service" "edge-cache-service" "edge-local-ric" "edge-cloud-sync" "edge-intent-processor")
for service in "${REQUIRED_SERVICES[@]}"; do
    kubectl get service "${service}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1
    check_success "Service '${service}' exists"
done

# Check service endpoints
for service in "${REQUIRED_SERVICES[@]}"; do
    ENDPOINTS=$(kubectl get endpoints "${service}" -n ${EDGE_NAMESPACE} -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w)
    if [ "${ENDPOINTS}" -gt 0 ]; then
        log "✅ Service '${service}' has ${ENDPOINTS} endpoint(s)"
    else
        log "⚠️ Service '${service}' has no endpoints"
    fi
done

# Phase 5: Configuration Validation
log "Phase 5: Configuration Validation"

# Check ConfigMaps
REQUIRED_CONFIGMAPS=("edge-computing-config" "edge-cloud-sync-config" "edge-monitoring-config" "edge-recording-rules" "edge-alert-rules")
for cm in "${REQUIRED_CONFIGMAPS[@]}"; do
    kubectl get configmap "${cm}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1
    check_success "ConfigMap '${cm}' exists"
done

# Check PVCs
REQUIRED_PVCS=("edge-ml-models" "edge-cache-data" "edge-ric-data" "edge-sync-data" "edge-intent-data")
for pvc in "${REQUIRED_PVCS[@]}"; do
    kubectl get pvc "${pvc}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1
    if check_success "PVC '${pvc}' exists"; then
        PVC_STATUS=$(kubectl get pvc "${pvc}" -n ${EDGE_NAMESPACE} -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "${PVC_STATUS}" = "Bound" ]; then
            log "✅ PVC '${pvc}' is bound"
        else
            log "⚠️ PVC '${pvc}' status: ${PVC_STATUS}"
        fi
    fi
done

# Phase 6: Health Endpoint Validation
log "Phase 6: Health Endpoint Validation"

# Function to check service health
check_service_health() {
    local service_name=$1
    local port=$2
    local path=$3
    
    log "Checking health of ${service_name}..."
    
    # Port-forward and check health
    kubectl port-forward "service/${service_name}" "${port}:${port}" -n ${EDGE_NAMESPACE} > /dev/null 2>&1 &
    PF_PID=$!
    sleep 3
    
    # Check health endpoint
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}${path}" 2>/dev/null || echo "000")
    
    # Kill port-forward
    kill $PF_PID > /dev/null 2>&1
    
    if [ "${HTTP_STATUS}" = "200" ]; then
        log "✅ ${service_name} health check passed (HTTP ${HTTP_STATUS})"
        return 0
    else
        log "⚠️ ${service_name} health check failed (HTTP ${HTTP_STATUS})"
        return 1
    fi
}

# Check health endpoints
check_service_health "edge-discovery-service" "8080" "/healthz"
check_service_health "edge-cloud-sync" "8080" "/healthz"
check_service_health "edge-intent-processor" "8080" "/healthz"

# Phase 7: O-RAN Function Validation
log "Phase 7: O-RAN Function Validation"

# Check O-RAN Local RIC
log "Checking O-RAN Local RIC functionality..."
RIC_POD=$(kubectl get pods -l app=edge-local-ric -n ${EDGE_NAMESPACE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -n "${RIC_POD}" ]; then
    log "✅ Found RIC pod: ${RIC_POD}"
    
    # Check RIC process
    kubectl exec "${RIC_POD}" -n ${EDGE_NAMESPACE} -- ps aux | grep -i ric > /dev/null 2>&1
    check_success "RIC process running in pod"
    
    # Check E2 interface port
    kubectl exec "${RIC_POD}" -n ${EDGE_NAMESPACE} -- netstat -ln | grep :38472 > /dev/null 2>&1
    check_success "E2 interface port (38472) listening"
    
    # Check A1 interface port
    kubectl exec "${RIC_POD}" -n ${EDGE_NAMESPACE} -- netstat -ln | grep :9999 > /dev/null 2>&1
    check_success "A1 interface port (9999) listening"
else
    log "❌ No RIC pod found"
fi

# Phase 8: ML Inference Validation
log "Phase 8: ML Inference Validation"

# Check ML service pods
ML_PODS=$(kubectl get pods -l app=edge-ml -n ${EDGE_NAMESPACE} --no-headers 2>/dev/null | wc -l)
if [ "${ML_PODS}" -gt 0 ]; then
    log "✅ Found ${ML_PODS} ML inference pod(s)"
    
    # Check TensorFlow Serving
    ML_POD=$(kubectl get pods -l app=edge-ml -n ${EDGE_NAMESPACE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "${ML_POD}" ]; then
        kubectl exec "${ML_POD}" -n ${EDGE_NAMESPACE} -c tensorflow-serving -- curl -s http://localhost:8501/v1/models > /dev/null 2>&1
        check_success "TensorFlow Serving API responding"
    fi
else
    log "❌ No ML inference pods found"
fi

# Phase 9: Cache Performance Validation
log "Phase 9: Cache Performance Validation"

# Check Redis cache
CACHE_PODS=$(kubectl get pods -l app=edge-cache -n ${EDGE_NAMESPACE} --no-headers 2>/dev/null | wc -l)
if [ "${CACHE_PODS}" -gt 0 ]; then
    log "✅ Found ${CACHE_PODS} cache pod(s)"
    
    # Check Redis connectivity
    CACHE_POD=$(kubectl get pods -l app=edge-cache -n ${EDGE_NAMESPACE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "${CACHE_POD}" ]; then
        kubectl exec "${CACHE_POD}" -n ${EDGE_NAMESPACE} -c redis-cache -- redis-cli ping | grep PONG > /dev/null 2>&1
        check_success "Redis cache responding to ping"
        
        # Check cache info
        kubectl exec "${CACHE_POD}" -n ${EDGE_NAMESPACE} -c redis-cache -- redis-cli info server | grep redis_version > /dev/null 2>&1
        check_success "Redis cache version info available"
    fi
else
    log "❌ No cache pods found"
fi

# Phase 10: Metrics and Monitoring Validation
log "Phase 10: Metrics and Monitoring Validation"

# Check if metrics collector is running on edge nodes
METRICS_PODS=$(kubectl get pods -l app=edge-metrics-collector -n ${EDGE_NAMESPACE} --no-headers 2>/dev/null | wc -l)
if [ "${METRICS_PODS}" -gt 0 ]; then
    log "✅ Found ${METRICS_PODS} metrics collector pod(s)"
    
    # Check metrics endpoint
    METRICS_POD=$(kubectl get pods -l app=edge-metrics-collector -n ${EDGE_NAMESPACE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "${METRICS_POD}" ]; then
        kubectl port-forward "${METRICS_POD}" 8080:8080 -n ${EDGE_NAMESPACE} > /dev/null 2>&1 &
        PF_PID=$!
        sleep 3
        
        curl -s http://localhost:8080/metrics | grep edge_ > /dev/null 2>&1
        if check_success "Edge metrics endpoint responding"; then
            METRICS_COUNT=$(curl -s http://localhost:8080/metrics | grep -c "^edge_" || echo "0")
            log "✅ Found ${METRICS_COUNT} edge-specific metrics"
        fi
        
        kill $PF_PID > /dev/null 2>&1
    fi
else
    log "❌ No metrics collector pods found"
fi

# Phase 11: End-to-End Connectivity Test
log "Phase 11: End-to-End Connectivity Test"

# Test edge-to-cloud sync connectivity
SYNC_POD=$(kubectl get pods -l app=edge-cloud-sync -n ${EDGE_NAMESPACE} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "${SYNC_POD}" ]; then
    log "✅ Found sync pod: ${SYNC_POD}"
    
    # Check sync service logs for successful connections
    kubectl logs "${SYNC_POD}" -n ${EDGE_NAMESPACE} --tail=50 | grep -i "connected\|sync.*success" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        log "✅ Edge-cloud sync showing successful connections"
    else
        log "⚠️ No recent successful sync connections found in logs"
    fi
else
    log "❌ No sync pod found"
fi

# Phase 12: Generate Validation Report
log "Phase 12: Generating Validation Report"

# Count successes and warnings
SUCCESSES=$(grep -c "✅" "${VALIDATION_LOG}")
WARNINGS=$(grep -c "⚠️" "${VALIDATION_LOG}")
FAILURES=$(grep -c "❌" "${VALIDATION_LOG}")

# Generate summary report
REPORT_FILE="/tmp/edge-validation-report-${VALIDATION_DATE}.json"
cat > "${REPORT_FILE}" <<EOF
{
  "validation_id": "EDGE-VAL-${VALIDATION_DATE}",
  "timestamp": "$(date -Iseconds)",
  "cluster": "$(kubectl config current-context)",
  "edge_namespace": "${EDGE_NAMESPACE}",
  "summary": {
    "total_checks": $((SUCCESSES + WARNINGS + FAILURES)),
    "successes": ${SUCCESSES},
    "warnings": ${WARNINGS},
    "failures": ${FAILURES},
    "success_rate": $(echo "scale=2; ${SUCCESSES} * 100 / (${SUCCESSES} + ${WARNINGS} + ${FAILURES})" | bc -l 2>/dev/null || echo "0")
  },
  "edge_nodes_found": ${EDGE_NODES},
  "metrics_pods": ${METRICS_PODS},
  "ml_pods": ${ML_PODS},
  "cache_pods": ${CACHE_PODS},
  "status": "$([ ${FAILURES} -eq 0 ] && echo "PASSED" || echo "FAILED")",
  "recommendations": [
    $([ ${WARNINGS} -gt 0 ] && echo '"Review warning items for optimal performance",' || echo '')
    $([ ${FAILURES} -gt 0 ] && echo '"Address failed checks before production deployment",' || echo '')
    "Monitor edge metrics and health endpoints regularly",
    "Validate O-RAN function performance under load",
    "Test autonomous operation during connectivity loss"
  ]
}
EOF

# Final summary
log "=== Edge Computing Validation Summary ==="
log "Validation ID: EDGE-VAL-${VALIDATION_DATE}"
log "Total Checks: $((SUCCESSES + WARNINGS + FAILURES))"
log "✅ Successes: ${SUCCESSES}"
log "⚠️ Warnings: ${WARNINGS}"
log "❌ Failures: ${FAILURES}"
log "Success Rate: $(echo "scale=1; ${SUCCESSES} * 100 / (${SUCCESSES} + ${WARNINGS} + ${FAILURES})" | bc -l 2>/dev/null || echo "0")%"

if [ ${FAILURES} -eq 0 ]; then
    log "🎉 Edge Computing Deployment Validation: PASSED"
    log "✅ Edge computing integration is ready for production use"
else
    log "❌ Edge Computing Deployment Validation: FAILED"
    log "⚠️ Please address failed checks before production deployment"
fi

log "Detailed log: ${VALIDATION_LOG}"
log "Summary report: ${REPORT_FILE}"
echo ""
echo "Edge Computing Validation Complete"
echo "Check reports at:"
echo "  - Detailed log: ${VALIDATION_LOG}"
echo "  - Summary report: ${REPORT_FILE}"
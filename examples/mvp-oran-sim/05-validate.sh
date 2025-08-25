#!/bin/bash
# Script: 05-validate.sh
# Purpose: Validate deployment status and replica count

set -e

NAMESPACE="${NAMESPACE:-mvp-demo}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-nf-sim}"
EXPECTED_REPLICAS="${EXPECTED_REPLICAS:-0}"
CONTINUOUS="${CONTINUOUS:-false}"
INTERVAL="${INTERVAL:-5}"

echo "==== MVP Demo: Validate Deployment ===="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

get_deployment_status() {
    local name=$1
    local namespace=$2
    
    if kubectl get deployment "$name" -n "$namespace" >/dev/null 2>&1; then
        echo "found"
    else
        echo "notfound"
    fi
}

show_validation_results() {
    local namespace=$1
    local deployment=$2
    local expected=$3
    
    echo ""
    echo "==== Deployment Validation Results ===="
    
    if [ "$(get_deployment_status "$deployment" "$namespace")" = "notfound" ]; then
        echo "❌ Deployment not found: $deployment in namespace $namespace"
        return 1
    fi
    
    # Get deployment details
    local replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    local ready=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local updated=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.updatedReplicas}' 2>/dev/null || echo "0")
    local available=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "0")
    local created=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.metadata.creationTimestamp}' 2>/dev/null)
    
    echo "Deployment: $deployment"
    echo "Namespace: $namespace"
    echo "Created: $created"
    
    echo ""
    echo "Replica Status:"
    echo "  Desired:   $replicas"
    echo "  Ready:     $ready"
    echo "  Updated:   $updated"
    echo "  Available: $available"
    
    # Check if deployment is healthy
    if [ "$ready" = "$replicas" ] && [ "$replicas" -gt 0 ]; then
        echo ""
        echo "✅ Deployment is healthy"
        local is_healthy=true
    else
        echo ""
        echo "⚠️ Deployment is not fully ready"
        local is_healthy=false
    fi
    
    # Check expected replicas if specified
    if [ "$expected" -gt 0 ]; then
        if [ "$replicas" = "$expected" ]; then
            echo "✅ Replica count matches expected: $expected"
        else
            echo "❌ Replica count mismatch - Expected: $expected, Actual: $replicas"
        fi
    fi
    
    # Show conditions
    echo ""
    echo "Conditions:"
    kubectl get deployment "$deployment" -n "$namespace" -o json | \
        jq -r '.status.conditions[]? | "  \(if .status == "True" then "✓" else "✗" end) \(.type): \(.status)"'
    
    if [ "$is_healthy" = "true" ]; then
        return 0
    else
        return 1
    fi
}

validate_once() {
    # Get deployment status
    show_validation_results "$NAMESPACE" "$DEPLOYMENT_NAME" "$EXPECTED_REPLICAS" || true
    
    # Show pod details
    echo ""
    echo "==== Pod Details ===="
    kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT_NAME" --no-headers 2>/dev/null | while read -r line; do
        pod_name=$(echo "$line" | awk '{print $1}')
        ready=$(echo "$line" | awk '{print $2}')
        status=$(echo "$line" | awk '{print $3}')
        restarts=$(echo "$line" | awk '{print $4}')
        age=$(echo "$line" | awk '{print $5}')
        
        if [ "$status" = "Running" ]; then
            symbol="✓"
        else
            symbol="✗"
        fi
        
        echo "  $symbol $pod_name - Status: $status, Ready: $ready, Restarts: $restarts, Age: $age"
    done
    
    # Show service endpoint
    echo ""
    echo "==== Service Endpoint ===="
    if kubectl get service "$DEPLOYMENT_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
        service_info=$(kubectl get service "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o json)
        service_type=$(echo "$service_info" | jq -r '.spec.type')
        cluster_ip=$(echo "$service_info" | jq -r '.spec.clusterIP')
        port=$(echo "$service_info" | jq -r '.spec.ports[0].port')
        
        echo "Service: $DEPLOYMENT_NAME"
        echo "Type: $service_type"
        echo "Cluster IP: $cluster_ip"
        echo "Port: $port"
        
        if [ "$service_type" = "LoadBalancer" ]; then
            external_ip=$(echo "$service_info" | jq -r '.status.loadBalancer.ingress[0].ip // .status.loadBalancer.ingress[0].hostname // "pending"')
            echo "External Endpoint: $external_ip"
        fi
    else
        echo "Service not found"
    fi
    
    # Show recent events
    echo ""
    echo "==== Recent Events ===="
    kubectl get events -n "$NAMESPACE" --field-selector "involvedObject.name=$DEPLOYMENT_NAME" \
        --sort-by='.lastTimestamp' 2>/dev/null | tail -n 5 | grep -v "^LAST SEEN" || echo "  No recent events"
    
    # Summary
    echo ""
    echo "==== Validation Summary ===="
    if [ "$(get_deployment_status "$DEPLOYMENT_NAME" "$NAMESPACE")" = "found" ]; then
        local ready=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        
        if [ "$ready" = "$desired" ] && [ "$ready" -gt 0 ]; then
            echo "✅ Deployment validation PASSED"
            echo "  - Deployment is running with $ready ready replicas"
        else
            echo "⚠️ Deployment validation WARNINGS"
            echo "  - Not all replicas are ready ($ready/$desired)"
        fi
    else
        echo "❌ Deployment validation FAILED"
        echo "  - Deployment not found"
    fi
}

# Main validation loop
if [ "$CONTINUOUS" = "true" ]; then
    while true; do
        clear
        echo "==== MVP Demo: Validate Deployment (Continuous Mode) ===="
        echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
        validate_once
        echo ""
        echo "Refreshing in $INTERVAL seconds... (Press Ctrl+C to stop)"
        sleep "$INTERVAL"
    done
else
    validate_once
    
    # Final message
    echo ""
    echo "==== Next Steps ===="
    echo "1. To scale up: Run ./03-send-intent.sh with REPLICAS=5"
    echo "2. To scale down: Run ./03-send-intent.sh with REPLICAS=1"
    echo "3. To monitor continuously: Run CONTINUOUS=true ./05-validate.sh"
    echo "4. To clean up: Run 'make mvp-clean' or 'kubectl delete namespace $NAMESPACE'"
fi
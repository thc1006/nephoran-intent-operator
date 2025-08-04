#!/bin/bash
# Security Hardening Validation Script for Nephoran Intent Operator

set -euo pipefail

NAMESPACE="${NAMESPACE:-nephoran-system}"
FAILED_CHECKS=0
PASSED_CHECKS=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED_CHECKS++))
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED_CHECKS++))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

echo "=== Nephoran Intent Operator Security Validation ==="
echo "Namespace: $NAMESPACE"
echo ""

# Check 1: Namespace Pod Security Standards
echo "1. Checking Pod Security Standards..."
PSS_LABELS=$(kubectl get namespace $NAMESPACE -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}')
if [[ "$PSS_LABELS" == "restricted" ]]; then
    check_pass "Pod Security Standards enforced (restricted)"
else
    check_fail "Pod Security Standards not properly configured (expected: restricted, got: $PSS_LABELS)"
fi

# Check 2: RBAC Permissions
echo ""
echo "2. Checking RBAC Permissions..."
# Check for wildcard permissions
WILDCARD_PERMS=$(kubectl get clusterroles,roles -A -o json | jq -r '.items[] | select(.rules[]? | select(.apiGroups[]? == "*" or .resources[]? == "*" or .verbs[]? == "*")) | .metadata.name' | grep -i nephoran || true)
if [[ -z "$WILDCARD_PERMS" ]]; then
    check_pass "No wildcard RBAC permissions found"
else
    check_fail "Wildcard RBAC permissions found in: $WILDCARD_PERMS"
fi

# Check 3: Service Account Token Mounting
echo ""
echo "3. Checking Service Account Token Mounting..."
DEPLOYMENTS=$(kubectl get deployments -n $NAMESPACE -o name)
for deployment in $DEPLOYMENTS; do
    AUTO_MOUNT=$(kubectl get $deployment -n $NAMESPACE -o jsonpath='{.spec.template.spec.automountServiceAccountToken}')
    if [[ "$AUTO_MOUNT" == "false" ]] || [[ -z "$AUTO_MOUNT" ]]; then
        check_pass "$deployment: Service account token auto-mounting disabled or not set"
    else
        check_warn "$deployment: Service account token auto-mounting enabled"
    fi
done

# Check 4: Container Security Contexts
echo ""
echo "4. Checking Container Security Contexts..."
for deployment in $DEPLOYMENTS; do
    CONTAINERS=$(kubectl get $deployment -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[*].name}')
    for container in $CONTAINERS; do
        # Check runAsNonRoot
        RUN_AS_NONROOT=$(kubectl get $deployment -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$container')].securityContext.runAsNonRoot}")
        if [[ "$RUN_AS_NONROOT" == "true" ]]; then
            check_pass "$deployment/$container: runAsNonRoot enabled"
        else
            check_fail "$deployment/$container: runAsNonRoot not enabled"
        fi
        
        # Check readOnlyRootFilesystem
        READONLY_FS=$(kubectl get $deployment -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$container')].securityContext.readOnlyRootFilesystem}")
        if [[ "$READONLY_FS" == "true" ]]; then
            check_pass "$deployment/$container: readOnlyRootFilesystem enabled"
        else
            check_warn "$deployment/$container: readOnlyRootFilesystem not enabled"
        fi
        
        # Check capabilities dropped
        CAPS_DROPPED=$(kubectl get $deployment -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$container')].securityContext.capabilities.drop[0]}")
        if [[ "$CAPS_DROPPED" == "ALL" ]]; then
            check_pass "$deployment/$container: All capabilities dropped"
        else
            check_fail "$deployment/$container: Not all capabilities dropped"
        fi
    done
done

# Check 5: Network Policies
echo ""
echo "5. Checking Network Policies..."
NETPOL_COUNT=$(kubectl get networkpolicies -n $NAMESPACE --no-headers | wc -l)
POD_COUNT=$(kubectl get pods -n $NAMESPACE --no-headers | wc -l)
if [[ $NETPOL_COUNT -gt 0 ]]; then
    check_pass "Network policies found: $NETPOL_COUNT"
    # Check for default deny
    DEFAULT_DENY=$(kubectl get networkpolicies -n $NAMESPACE -o json | jq -r '.items[] | select(.metadata.name == "default-deny-all")')
    if [[ -n "$DEFAULT_DENY" ]]; then
        check_pass "Default deny-all network policy exists"
    else
        check_warn "No default deny-all network policy found"
    fi
else
    check_fail "No network policies found"
fi

# Check 6: Secrets Management
echo ""
echo "6. Checking Secrets Management..."
# Check if secrets are mounted as files
for deployment in $DEPLOYMENTS; do
    SECRET_MOUNTS=$(kubectl get $deployment -n $NAMESPACE -o json | jq -r '.spec.template.spec.volumes[]? | select(.secret) | .name')
    if [[ -n "$SECRET_MOUNTS" ]]; then
        check_pass "$deployment: Uses secret volume mounts"
    else
        # Check for secret env vars (not recommended)
        SECRET_ENVS=$(kubectl get $deployment -n $NAMESPACE -o json | jq -r '.spec.template.spec.containers[].env[]? | select(.valueFrom.secretKeyRef) | .name')
        if [[ -n "$SECRET_ENVS" ]]; then
            check_warn "$deployment: Uses secret environment variables (consider file mounts instead)"
        fi
    fi
done

# Check 7: Resource Limits
echo ""
echo "7. Checking Resource Limits..."
for deployment in $DEPLOYMENTS; do
    CONTAINERS=$(kubectl get $deployment -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[*].name}')
    for container in $CONTAINERS; do
        CPU_LIMIT=$(kubectl get $deployment -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$container')].resources.limits.cpu}")
        MEM_LIMIT=$(kubectl get $deployment -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$container')].resources.limits.memory}")
        if [[ -n "$CPU_LIMIT" ]] && [[ -n "$MEM_LIMIT" ]]; then
            check_pass "$deployment/$container: Resource limits configured"
        else
            check_fail "$deployment/$container: Resource limits not configured"
        fi
    done
done

# Check 8: Admission Webhooks
echo ""
echo "8. Checking Admission Webhooks..."
WEBHOOKS=$(kubectl get validatingwebhookconfigurations -o json | jq -r '.items[] | select(.metadata.name | contains("nephoran")) | .metadata.name')
if [[ -n "$WEBHOOKS" ]]; then
    check_pass "Validating webhooks found: $WEBHOOKS"
    # Check failure policy
    FAILURE_POLICY=$(kubectl get validatingwebhookconfigurations -o json | jq -r '.items[] | select(.metadata.name | contains("nephoran")) | .webhooks[].failurePolicy')
    if [[ "$FAILURE_POLICY" == "Fail" ]]; then
        check_pass "Webhook failure policy set to Fail"
    else
        check_warn "Webhook failure policy not set to Fail"
    fi
else
    check_fail "No validating webhooks found"
fi

# Check 9: Image Pull Policy
echo ""
echo "9. Checking Image Pull Policies..."
for deployment in $DEPLOYMENTS; do
    PULL_POLICIES=$(kubectl get $deployment -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[*].imagePullPolicy}')
    for policy in $PULL_POLICIES; do
        if [[ "$policy" == "Always" ]]; then
            check_pass "$deployment: Image pull policy set to Always"
        else
            check_warn "$deployment: Image pull policy not set to Always (got: $policy)"
        fi
    done
done

# Check 10: TLS Configuration
echo ""
echo "10. Checking TLS Configuration..."
SERVICES=$(kubectl get services -n $NAMESPACE -o name)
for service in $SERVICES; do
    # Check if service has TLS annotations or uses HTTPS
    TLS_ENABLED=$(kubectl get $service -n $NAMESPACE -o jsonpath='{.metadata.annotations.service\.beta\.kubernetes\.io/aws-load-balancer-ssl-cert}')
    if [[ -n "$TLS_ENABLED" ]]; then
        check_pass "$service: TLS configured"
    else
        # Check if it's an internal service (doesn't need external TLS)
        SERVICE_TYPE=$(kubectl get $service -n $NAMESPACE -o jsonpath='{.spec.type}')
        if [[ "$SERVICE_TYPE" == "ClusterIP" ]]; then
            check_pass "$service: Internal service (ClusterIP)"
        else
            check_warn "$service: External service without TLS configuration"
        fi
    fi
done

# Summary
echo ""
echo "=== Security Validation Summary ==="
echo -e "Passed checks: ${GREEN}$PASSED_CHECKS${NC}"
echo -e "Failed checks: ${RED}$FAILED_CHECKS${NC}"

if [[ $FAILED_CHECKS -eq 0 ]]; then
    echo -e "${GREEN}✓ All security checks passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some security checks failed. Please review and fix the issues.${NC}"
    exit 1
fi
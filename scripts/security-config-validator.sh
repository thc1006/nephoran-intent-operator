#!/bin/bash
# Security Configuration Validator for Nephoran Intent Operator
# Validates security configuration compliance and best practices
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-nephoran-system}"
REPORT_DIR="${REPORT_DIR:-/tmp/nephoran-security-reports}"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

# Initialize report directory
mkdir -p "$REPORT_DIR"

validate_pod_security_standards() {
    log "Validating Pod Security Standards compliance..."
    
    local violations=0
    local report_file="$REPORT_DIR/pod-security-standards-$TIMESTAMP.json"
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": []}" > "$report_file"
    
    # Check for runAsNonRoot
    local non_root_violations=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[] | select(.spec.securityContext.runAsNonRoot != true) | .metadata.name' | wc -l)
    
    if [ "$non_root_violations" -gt 0 ]; then
        error "Found $non_root_violations pods not running as non-root user"
        violations=$((violations + non_root_violations))
        
        kubectl get pods -n $NAMESPACE -o json | \
            jq -r '.items[] | select(.spec.securityContext.runAsNonRoot != true) | .metadata.name' | \
            while read pod; do
                jq --arg pod "$pod" '.violations += [{"type": "runAsNonRoot", "pod": $pod, "severity": "high"}]' \
                    "$report_file" > "${report_file}.tmp" && mv "${report_file}.tmp" "$report_file"
            done
    else
        success "All pods configured to run as non-root user"
    fi
    
    # Check for readOnlyRootFilesystem
    local readonly_violations=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[] | select(.spec.containers[]? | .securityContext.readOnlyRootFilesystem != true) | .metadata.name' | wc -l)
    
    if [ "$readonly_violations" -gt 0 ]; then
        warn "Found $readonly_violations pods without read-only root filesystem"
        violations=$((violations + readonly_violations))
    else
        success "All pods configured with read-only root filesystem"
    fi
    
    # Check for allowPrivilegeEscalation
    local privilege_violations=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[] | select(.spec.containers[]? | .securityContext.allowPrivilegeEscalation != false) | .metadata.name' | wc -l)
    
    if [ "$privilege_violations" -gt 0 ]; then
        error "Found $privilege_violations pods allowing privilege escalation"
        violations=$((violations + privilege_violations))
    else
        success "All pods configured to deny privilege escalation"
    fi
    
    echo "Pod Security Standards validation complete: $violations violations found"
    return $violations
}

validate_network_policies() {
    log "Validating Network Policy configuration..."
    
    local violations=0
    local report_file="$REPORT_DIR/network-policies-$TIMESTAMP.json"
    
    # Check if NetworkPolicies exist
    local policy_count=$(kubectl get networkpolicies -n $NAMESPACE --no-headers 2>/dev/null | wc -l)
    
    if [ "$policy_count" -eq 0 ]; then
        error "No NetworkPolicies found in namespace $NAMESPACE"
        violations=$((violations + 1))
        echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": [{\"type\": \"missing_network_policies\", \"severity\": \"critical\"}]}" > "$report_file"
    else
        success "Found $policy_count NetworkPolicies in namespace"
        
        # Check for default deny policy
        local default_deny=$(kubectl get networkpolicies -n $NAMESPACE -o json | \
            jq -r '.items[] | select(.spec.podSelector == {}) | .metadata.name' | wc -l)
        
        if [ "$default_deny" -eq 0 ]; then
            warn "No default deny NetworkPolicy found"
            violations=$((violations + 1))
        else
            success "Default deny NetworkPolicy configured"
        fi
        
        # Check for ingress and egress rules
        local policies_with_rules=$(kubectl get networkpolicies -n $NAMESPACE -o json | \
            jq -r '.items[] | select(.spec.ingress or .spec.egress) | .metadata.name' | wc -l)
        
        if [ "$policies_with_rules" -eq 0 ]; then
            warn "No NetworkPolicies with specific ingress/egress rules found"
            violations=$((violations + 1))
        else
            success "NetworkPolicies with specific rules configured"
        fi
    fi
    
    echo "Network Policy validation complete: $violations violations found"
    return $violations
}

validate_rbac_configuration() {
    log "Validating RBAC configuration..."
    
    local violations=0
    local report_file="$REPORT_DIR/rbac-validation-$TIMESTAMP.json"
    
    # Check for overprivileged service accounts
    local sa_list=$(kubectl get serviceaccounts -n $NAMESPACE -o json | jq -r '.items[].metadata.name')
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": []}" > "$report_file"
    
    for sa in $sa_list; do
        # Check if service account can create pods (should be restricted)
        if kubectl auth can-i create pods --as=system:serviceaccount:$NAMESPACE:$sa -n $NAMESPACE 2>/dev/null; then
            if [ "$sa" != "nephio-bridge" ] && [ "$sa" != "oran-adaptor" ]; then
                warn "Service account $sa has pod creation permissions"
                violations=$((violations + 1))
                jq --arg sa "$sa" '.violations += [{"type": "overprivileged_sa", "service_account": $sa, "permission": "create_pods", "severity": "medium"}]' \
                    "$report_file" > "${report_file}.tmp" && mv "${report_file}.tmp" "$report_file"
            fi
        fi
        
        # Check if service account can list secrets (should be restricted)
        if kubectl auth can-i list secrets --as=system:serviceaccount:$NAMESPACE:$sa -n $NAMESPACE 2>/dev/null; then
            if [ "$sa" != "nephio-bridge" ]; then
                warn "Service account $sa has secrets listing permissions"
                violations=$((violations + 1))
                jq --arg sa "$sa" '.violations += [{"type": "overprivileged_sa", "service_account": $sa, "permission": "list_secrets", "severity": "high"}]' \
                    "$report_file" > "${report_file}.tmp" && mv "${report_file}.tmp" "$report_file"
            fi
        fi
    done
    
    # Check for cluster-admin bindings
    local cluster_admin_bindings=$(kubectl get clusterrolebindings -o json | \
        jq -r ".items[] | select(.roleRef.name == \"cluster-admin\" and (.subjects[]? | .namespace == \"$NAMESPACE\")) | .metadata.name" | wc -l)
    
    if [ "$cluster_admin_bindings" -gt 0 ]; then
        error "Found cluster-admin bindings for namespace $NAMESPACE"
        violations=$((violations + cluster_admin_bindings))
    else
        success "No cluster-admin bindings found for namespace"
    fi
    
    echo "RBAC validation complete: $violations violations found"
    return $violations
}

validate_secret_management() {
    log "Validating secret management practices..."
    
    local violations=0
    local report_file="$REPORT_DIR/secret-management-$TIMESTAMP.json"
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": []}" > "$report_file"
    
    # Check for secrets in environment variables
    local env_secret_count=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[].spec.containers[].env[]? | select(.value and (.name | test("(?i)(password|token|key|secret)"))) | .name' | wc -l)
    
    if [ "$env_secret_count" -gt 0 ]; then
        error "Found $env_secret_count potential secrets in environment variables"
        violations=$((violations + env_secret_count))
        jq --argjson count "$env_secret_count" '.violations += [{"type": "secrets_in_env", "count": $count, "severity": "critical"}]' \
            "$report_file" > "${report_file}.tmp" && mv "${report_file}.tmp" "$report_file"
    else
        success "No secrets found in environment variables"
    fi
    
    # Check for proper secret mounting
    local secret_volume_count=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[].spec.volumes[]? | select(.secret) | .name' | wc -l)
    
    if [ "$secret_volume_count" -gt 0 ]; then
        success "Found $secret_volume_count properly mounted secret volumes"
    else
        warn "No secret volumes found - verify secret management strategy"
    fi
    
    # Check secret age (secrets older than 90 days should be rotated)
    local old_secrets=$(kubectl get secrets -n $NAMESPACE -o json | \
        jq -r --arg date "$(date -d '90 days ago' -Iseconds)" '.items[] | select(.metadata.creationTimestamp < $date) | .metadata.name' | wc -l)
    
    if [ "$old_secrets" -gt 0 ]; then
        warn "Found $old_secrets secrets older than 90 days - consider rotation"
        violations=$((violations + 1))
    else
        success "All secrets are within rotation window"
    fi
    
    echo "Secret management validation complete: $violations violations found"
    return $violations
}

validate_container_security() {
    log "Validating container security configuration..."
    
    local violations=0
    local report_file="$REPORT_DIR/container-security-$TIMESTAMP.json"
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": []}" > "$report_file"
    
    # Check image pull policy
    local always_pull_violations=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[].spec.containers[] | select(.imagePullPolicy != "Always") | .name' | wc -l)
    
    if [ "$always_pull_violations" -gt 0 ]; then
        warn "Found $always_pull_violations containers not using 'Always' pull policy"
        violations=$((violations + always_pull_violations))
    else
        success "All containers configured with 'Always' pull policy"
    fi
    
    # Check for resource limits
    local no_limits_count=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[].spec.containers[] | select(.resources.limits == null) | .name' | wc -l)
    
    if [ "$no_limits_count" -gt 0 ]; then
        error "Found $no_limits_count containers without resource limits"
        violations=$((violations + no_limits_count))
    else
        success "All containers have resource limits configured"
    fi
    
    # Check for capabilities
    local privileged_containers=$(kubectl get pods -n $NAMESPACE -o json | \
        jq -r '.items[].spec.containers[] | select(.securityContext.privileged == true) | .name' | wc -l)
    
    if [ "$privileged_containers" -gt 0 ]; then
        error "Found $privileged_containers privileged containers"
        violations=$((violations + privileged_containers))
    else
        success "No privileged containers found"
    fi
    
    echo "Container security validation complete: $violations violations found"
    return $violations
}

validate_tls_configuration() {
    log "Validating TLS configuration..."
    
    local violations=0
    local report_file="$REPORT_DIR/tls-configuration-$TIMESTAMP.json"
    
    echo "{\"timestamp\": \"$(date -Iseconds)\", \"violations\": []}" > "$report_file"
    
    # Check for TLS secrets
    local tls_secret_count=$(kubectl get secrets -n $NAMESPACE -o json | \
        jq -r '.items[] | select(.type == "kubernetes.io/tls") | .metadata.name' | wc -l)
    
    if [ "$tls_secret_count" -eq 0 ]; then
        warn "No TLS secrets found in namespace"
        violations=$((violations + 1))
    else
        success "Found $tls_secret_count TLS secrets"
        
        # Check TLS secret expiration
        kubectl get secrets -n $NAMESPACE -o json | \
            jq -r '.items[] | select(.type == "kubernetes.io/tls") | .metadata.name' | \
            while read secret; do
                local cert_data=$(kubectl get secret "$secret" -n $NAMESPACE -o json | jq -r '.data."tls.crt"' | base64 -d)
                local expiry=$(echo "$cert_data" | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
                if [ -n "$expiry" ]; then
                    local days_until_expiry=$(( ($(date -d "$expiry" +%s) - $(date +%s)) / 86400 ))
                    if [ "$days_until_expiry" -lt 30 ]; then
                        warn "TLS certificate in secret $secret expires in $days_until_expiry days"
                        violations=$((violations + 1))
                    fi
                fi
            done
    fi
    
    # Check ingress TLS configuration
    local ingress_tls_count=$(kubectl get ingress -n $NAMESPACE -o json 2>/dev/null | \
        jq -r '.items[] | select(.spec.tls) | .metadata.name' | wc -l || echo "0")
    
    if [ "$ingress_tls_count" -gt 0 ]; then
        success "Found $ingress_tls_count ingresses with TLS configuration"
    else
        warn "No ingresses with TLS configuration found"
    fi
    
    echo "TLS configuration validation complete: $violations violations found"
    return $violations
}

generate_security_report() {
    log "Generating comprehensive security report..."
    
    local total_violations=0
    local report_file="$REPORT_DIR/security-assessment-$TIMESTAMP.json"
    
    # Run all validation functions
    validate_pod_security_standards
    local pod_violations=$?
    total_violations=$((total_violations + pod_violations))
    
    validate_network_policies
    local network_violations=$?
    total_violations=$((total_violations + network_violations))
    
    validate_rbac_configuration
    local rbac_violations=$?
    total_violations=$((total_violations + rbac_violations))
    
    validate_secret_management
    local secret_violations=$?
    total_violations=$((total_violations + secret_violations))
    
    validate_container_security
    local container_violations=$?
    total_violations=$((total_violations + container_violations))
    
    validate_tls_configuration
    local tls_violations=$?
    total_violations=$((total_violations + tls_violations))
    
    # Calculate security score
    local max_score=100
    local deduction_per_violation=2
    local security_score=$((max_score - (total_violations * deduction_per_violation)))
    if [ "$security_score" -lt 0 ]; then
        security_score=0
    fi
    
    # Generate final report
    cat > "$report_file" <<EOF
{
  "timestamp": "$(date -Iseconds)",
  "namespace": "$NAMESPACE",
  "assessment_summary": {
    "total_violations": $total_violations,
    "security_score": $security_score,
    "max_score": $max_score,
    "status": "$([ $security_score -ge 85 ] && echo "PASS" || echo "FAIL")"
  },
  "category_results": {
    "pod_security_standards": {
      "violations": $pod_violations,
      "status": "$([ $pod_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "network_policies": {
      "violations": $network_violations,
      "status": "$([ $network_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "rbac_configuration": {
      "violations": $rbac_violations,
      "status": "$([ $rbac_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "secret_management": {
      "violations": $secret_violations,
      "status": "$([ $secret_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "container_security": {
      "violations": $container_violations,
      "status": "$([ $container_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "tls_configuration": {
      "violations": $tls_violations,
      "status": "$([ $tls_violations -eq 0 ] && echo "PASS" || echo "FAIL")"
    }
  },
  "recommendations": [
    $([ $pod_violations -gt 0 ] && echo "\"Review and fix Pod Security Standards violations\",")
    $([ $network_violations -gt 0 ] && echo "\"Implement proper NetworkPolicies for micro-segmentation\",")
    $([ $rbac_violations -gt 0 ] && echo "\"Review RBAC configuration for least privilege access\",")
    $([ $secret_violations -gt 0 ] && echo "\"Improve secret management practices and rotation\",")
    $([ $container_violations -gt 0 ] && echo "\"Harden container security configuration\",")
    $([ $tls_violations -gt 0 ] && echo "\"Ensure proper TLS configuration and certificate management\",")
    "\"Regular security assessments and continuous monitoring\""
  ]
}
EOF
    
    log "Security assessment complete:"
    log "  Total violations: $total_violations"
    log "  Security score: $security_score/$max_score"
    log "  Status: $([ $security_score -ge 85 ] && echo "PASS" || echo "FAIL")"
    log "  Report saved to: $report_file"
    
    if [ $security_score -ge 85 ]; then
        success "Security assessment PASSED with score $security_score/100"
        return 0
    else
        error "Security assessment FAILED with score $security_score/100"
        return 1
    fi
}

main() {
    log "Starting Nephoran Intent Operator Security Configuration Validation"
    log "Namespace: $NAMESPACE"
    log "Report directory: $REPORT_DIR"
    
    # Ensure namespace exists
    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        error "Namespace $NAMESPACE does not exist"
        return 1
    fi
    
    # Run security validation
    generate_security_report
    local result=$?
    
    log "Security configuration validation complete"
    return $result
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
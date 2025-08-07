#!/bin/bash

# Compliance Validation Script for Nephoran Intent Operator
# This script performs continuous compliance monitoring for security standards

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NAMESPACE:-nephoran-system}"
COMPLIANCE_REPORT_DIR="${COMPLIANCE_REPORT_DIR:-./compliance-reports}"
SECURITY_POLICY_FILE="${SECURITY_POLICY_FILE:-.github/security-policy.yml}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${COMPLIANCE_REPORT_DIR}/compliance_report_${TIMESTAMP}.json"

# Create report directory
mkdir -p "${COMPLIANCE_REPORT_DIR}"

# Initialize report
echo "{" > "${REPORT_FILE}"
echo "  \"timestamp\": \"${TIMESTAMP}\"," >> "${REPORT_FILE}"
echo "  \"compliance_checks\": {" >> "${REPORT_FILE}"

# Function to log messages
log() {
    echo -e "${1}${2}${NC}"
}

# Function to add check result to report
add_to_report() {
    local check_name=$1
    local status=$2
    local details=$3
    
    echo "    \"${check_name}\": {" >> "${REPORT_FILE}"
    echo "      \"status\": \"${status}\"," >> "${REPORT_FILE}"
    echo "      \"details\": \"${details}\"" >> "${REPORT_FILE}"
    echo "    }," >> "${REPORT_FILE}"
}

# Function to check RBAC compliance
check_rbac_compliance() {
    log "${BLUE}" "Checking RBAC Compliance..."
    
    local violations=0
    local details=""
    
    # Check for wildcard permissions in roles
    local wildcard_roles=$(kubectl get roles,clusterroles -A -o json | \
        jq -r '.items[] | select(.rules[]? | .apiGroups[]? == "*" or .resources[]? == "*" or .verbs[]? == "*") | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${wildcard_roles}" ]; then
        violations=$((violations + 1))
        details="${details}Wildcard permissions found in: ${wildcard_roles}. "
        log "${RED}" "  ✗ Wildcard permissions detected"
    else
        log "${GREEN}" "  ✓ No wildcard permissions found"
    fi
    
    # Check for least privilege
    local admin_count=$(kubectl get clusterrolebindings -o json | \
        jq -r '.items[] | select(.roleRef.name == "cluster-admin") | .subjects[]?.name' | \
        wc -l)
    
    if [ "${admin_count}" -gt 3 ]; then
        violations=$((violations + 1))
        details="${details}Too many cluster-admin bindings: ${admin_count}. "
        log "${YELLOW}" "  ⚠ High number of cluster-admin bindings: ${admin_count}"
    else
        log "${GREEN}" "  ✓ Cluster-admin bindings within limit"
    fi
    
    # Check service account token automounting
    local auto_mount=$(kubectl get serviceaccounts -n "${NAMESPACE}" -o json | \
        jq -r '.items[] | select(.automountServiceAccountToken != false) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${auto_mount}" ]; then
        violations=$((violations + 1))
        details="${details}Service accounts with automount enabled: ${auto_mount}. "
        log "${YELLOW}" "  ⚠ Service accounts with automount token: ${auto_mount}"
    else
        log "${GREEN}" "  ✓ Service account token automounting disabled"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "rbac_compliance" "PASS" "All RBAC checks passed"
        return 0
    else
        add_to_report "rbac_compliance" "FAIL" "${details}"
        return 1
    fi
}

# Function to check network policies
check_network_policies() {
    log "${BLUE}" "Checking Network Policies..."
    
    local violations=0
    local details=""
    
    # Check for default deny policy
    local default_deny=$(kubectl get networkpolicies -n "${NAMESPACE}" -o json | \
        jq -r '.items[] | select(.spec.podSelector == {} and .spec.policyTypes[]? == "Ingress" and .spec.policyTypes[]? == "Egress") | .metadata.name')
    
    if [ -z "${default_deny}" ]; then
        violations=$((violations + 1))
        details="${details}No default deny network policy found. "
        log "${RED}" "  ✗ Default deny network policy missing"
    else
        log "${GREEN}" "  ✓ Default deny network policy exists"
    fi
    
    # Check for egress restrictions
    local unrestricted_egress=$(kubectl get networkpolicies -n "${NAMESPACE}" -o json | \
        jq -r '.items[] | select(.spec.egress[]?.to == null) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${unrestricted_egress}" ]; then
        violations=$((violations + 1))
        details="${details}Unrestricted egress in policies: ${unrestricted_egress}. "
        log "${YELLOW}" "  ⚠ Unrestricted egress policies: ${unrestricted_egress}"
    else
        log "${GREEN}" "  ✓ All egress policies have restrictions"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "network_policies" "PASS" "All network policy checks passed"
        return 0
    else
        add_to_report "network_policies" "FAIL" "${details}"
        return 1
    fi
}

# Function to check container security
check_container_security() {
    log "${BLUE}" "Checking Container Security..."
    
    local violations=0
    local details=""
    
    # Check for non-root containers
    local root_containers=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[] | select(.securityContext.runAsUser == 0 or .securityContext.runAsNonRoot != true) | .name' | \
        tr '\n' ' ')
    
    if [ -n "${root_containers}" ]; then
        violations=$((violations + 1))
        details="${details}Containers running as root: ${root_containers}. "
        log "${RED}" "  ✗ Containers running as root detected"
    else
        log "${GREEN}" "  ✓ All containers running as non-root"
    fi
    
    # Check for read-only root filesystem
    local writable_fs=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[] | select(.securityContext.readOnlyRootFilesystem != true) | .name' | \
        tr '\n' ' ')
    
    if [ -n "${writable_fs}" ]; then
        violations=$((violations + 1))
        details="${details}Containers with writable root filesystem: ${writable_fs}. "
        log "${YELLOW}" "  ⚠ Writable root filesystems: ${writable_fs}"
    else
        log "${GREEN}" "  ✓ All containers have read-only root filesystem"
    fi
    
    # Check for privileged containers
    local privileged=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[] | select(.securityContext.privileged == true or .securityContext.allowPrivilegeEscalation == true) | .name' | \
        tr '\n' ' ')
    
    if [ -n "${privileged}" ]; then
        violations=$((violations + 1))
        details="${details}Privileged containers: ${privileged}. "
        log "${RED}" "  ✗ Privileged containers detected"
    else
        log "${GREEN}" "  ✓ No privileged containers"
    fi
    
    # Check for capabilities
    local capabilities=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[] | select(.securityContext.capabilities.add != null and (.securityContext.capabilities.add | length) > 0) | .name' | \
        tr '\n' ' ')
    
    if [ -n "${capabilities}" ]; then
        violations=$((violations + 1))
        details="${details}Containers with added capabilities: ${capabilities}. "
        log "${YELLOW}" "  ⚠ Containers with additional capabilities: ${capabilities}"
    else
        log "${GREEN}" "  ✓ No additional capabilities granted"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "container_security" "PASS" "All container security checks passed"
        return 0
    else
        add_to_report "container_security" "FAIL" "${details}"
        return 1
    fi
}

# Function to check secrets management
check_secrets_management() {
    log "${BLUE}" "Checking Secrets Management..."
    
    local violations=0
    local details=""
    
    # Check for unencrypted secrets
    local unencrypted=$(kubectl get secrets -n "${NAMESPACE}" -o json | \
        jq -r '.items[] | select(.metadata.annotations["encryption.nephoran.io/algorithm"] == null) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${unencrypted}" ]; then
        violations=$((violations + 1))
        details="${details}Unencrypted secrets: ${unencrypted}. "
        log "${RED}" "  ✗ Unencrypted secrets detected"
    else
        log "${GREEN}" "  ✓ All secrets are encrypted"
    fi
    
    # Check for old secrets needing rotation
    local old_secrets=$(kubectl get secrets -n "${NAMESPACE}" -o json | \
        jq -r --arg date "$(date -d '90 days ago' --iso-8601)" \
        '.items[] | select(.metadata.annotations["rotation.nephoran.io/last-rotated"] < $date) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${old_secrets}" ]; then
        violations=$((violations + 1))
        details="${details}Secrets needing rotation: ${old_secrets}. "
        log "${YELLOW}" "  ⚠ Secrets older than 90 days: ${old_secrets}"
    else
        log "${GREEN}" "  ✓ All secrets within rotation period"
    fi
    
    # Check for secrets in environment variables
    local env_secrets=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[].env[]? | select(.name | test("PASSWORD|SECRET|KEY|TOKEN"; "i")) | .name' | \
        tr '\n' ' ')
    
    if [ -n "${env_secrets}" ]; then
        violations=$((violations + 1))
        details="${details}Potential secrets in environment: ${env_secrets}. "
        log "${YELLOW}" "  ⚠ Potential secrets in environment variables"
    else
        log "${GREEN}" "  ✓ No secrets in environment variables"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "secrets_management" "PASS" "All secrets management checks passed"
        return 0
    else
        add_to_report "secrets_management" "FAIL" "${details}"
        return 1
    fi
}

# Function to check TLS configuration
check_tls_configuration() {
    log "${BLUE}" "Checking TLS Configuration..."
    
    local violations=0
    local details=""
    
    # Check for TLS version in ingress
    local old_tls=$(kubectl get ingress -n "${NAMESPACE}" -o json | \
        jq -r '.items[] | select(.metadata.annotations["nginx.ingress.kubernetes.io/ssl-protocols"] != "TLSv1.3") | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${old_tls}" ]; then
        violations=$((violations + 1))
        details="${details}Ingresses not using TLS 1.3: ${old_tls}. "
        log "${YELLOW}" "  ⚠ Ingresses not enforcing TLS 1.3: ${old_tls}"
    else
        log "${GREEN}" "  ✓ All ingresses using TLS 1.3"
    fi
    
    # Check for certificate expiration
    local expiring_certs=$(kubectl get certificates -n "${NAMESPACE}" -o json 2>/dev/null | \
        jq -r --arg date "$(date -d '30 days' --iso-8601)" \
        '.items[] | select(.status.notAfter < $date) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${expiring_certs}" ]; then
        violations=$((violations + 1))
        details="${details}Certificates expiring soon: ${expiring_certs}. "
        log "${YELLOW}" "  ⚠ Certificates expiring within 30 days: ${expiring_certs}"
    else
        log "${GREEN}" "  ✓ All certificates valid for >30 days"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "tls_configuration" "PASS" "All TLS configuration checks passed"
        return 0
    else
        add_to_report "tls_configuration" "FAIL" "${details}"
        return 1
    fi
}

# Function to check O-RAN compliance
check_oran_compliance() {
    log "${BLUE}" "Checking O-RAN Security Compliance..."
    
    local violations=0
    local details=""
    
    # Check for A1 interface security
    local a1_mtls=$(kubectl get deployments -n "${NAMESPACE}" -l "oran-interface=a1" -o json | \
        jq -r '.items[] | select(.spec.template.metadata.annotations["security.istio.io/tlsMode"] != "STRICT") | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${a1_mtls}" ]; then
        violations=$((violations + 1))
        details="${details}A1 interface without mTLS: ${a1_mtls}. "
        log "${RED}" "  ✗ A1 interface missing mTLS"
    else
        log "${GREEN}" "  ✓ A1 interface has mTLS enabled"
    fi
    
    # Check for O1 interface NETCONF/SSH
    local o1_ssh=$(kubectl get services -n "${NAMESPACE}" -l "oran-interface=o1" -o json | \
        jq -r '.items[] | select(.spec.ports[]?.port == 830 and .metadata.annotations["netconf.enabled"] != "true") | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${o1_ssh}" ]; then
        violations=$((violations + 1))
        details="${details}O1 interface without NETCONF/SSH: ${o1_ssh}. "
        log "${YELLOW}" "  ⚠ O1 interface missing NETCONF/SSH"
    else
        log "${GREEN}" "  ✓ O1 interface properly configured"
    fi
    
    # Check for xApp sandboxing
    local xapp_sandbox=$(kubectl get deployments -n "${NAMESPACE}" -l "component=xapp" -o json | \
        jq -r '.items[] | select(.spec.template.spec.securityContext.runAsNonRoot != true) | .metadata.name' | \
        tr '\n' ' ')
    
    if [ -n "${xapp_sandbox}" ]; then
        violations=$((violations + 1))
        details="${details}xApps without sandboxing: ${xapp_sandbox}. "
        log "${YELLOW}" "  ⚠ xApps missing sandboxing"
    else
        log "${GREEN}" "  ✓ xApps properly sandboxed"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "oran_compliance" "PASS" "All O-RAN security checks passed"
        return 0
    else
        add_to_report "oran_compliance" "FAIL" "${details}"
        return 1
    fi
}

# Function to check audit logging
check_audit_logging() {
    log "${BLUE}" "Checking Audit Logging..."
    
    local violations=0
    local details=""
    
    # Check if audit logging is enabled
    local audit_enabled=$(kubectl get configmap -n kube-system audit-config -o json 2>/dev/null | \
        jq -r '.data.enabled')
    
    if [ "${audit_enabled}" != "true" ]; then
        violations=$((violations + 1))
        details="${details}Audit logging not enabled. "
        log "${RED}" "  ✗ Audit logging disabled"
    else
        log "${GREEN}" "  ✓ Audit logging enabled"
    fi
    
    # Check for audit webhook configuration
    local audit_webhook=$(kubectl get configmap -n kube-system audit-config -o json 2>/dev/null | \
        jq -r '.data.webhook')
    
    if [ -z "${audit_webhook}" ]; then
        violations=$((violations + 1))
        details="${details}No audit webhook configured. "
        log "${YELLOW}" "  ⚠ Audit webhook not configured"
    else
        log "${GREEN}" "  ✓ Audit webhook configured"
    fi
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "audit_logging" "PASS" "All audit logging checks passed"
        return 0
    else
        add_to_report "audit_logging" "FAIL" "${details}"
        return 1
    fi
}

# Function to run vulnerability scanning
run_vulnerability_scan() {
    log "${BLUE}" "Running Vulnerability Scan..."
    
    local violations=0
    local details=""
    
    # Check if Trivy is available
    if ! command -v trivy &> /dev/null; then
        log "${YELLOW}" "  ⚠ Trivy not installed, skipping image scanning"
        add_to_report "vulnerability_scan" "SKIP" "Trivy not available"
        return 0
    fi
    
    # Scan images in the namespace
    local images=$(kubectl get pods -n "${NAMESPACE}" -o json | \
        jq -r '.items[].spec.containers[].image' | sort -u)
    
    for image in ${images}; do
        log "${BLUE}" "  Scanning ${image}..."
        
        local scan_result=$(trivy image --severity CRITICAL,HIGH --quiet --format json "${image}" 2>/dev/null)
        local critical_count=$(echo "${scan_result}" | jq -r '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length')
        local high_count=$(echo "${scan_result}" | jq -r '[.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH")] | length')
        
        if [ "${critical_count}" -gt 0 ]; then
            violations=$((violations + critical_count))
            details="${details}${image}: ${critical_count} CRITICAL vulnerabilities. "
            log "${RED}" "    ✗ ${critical_count} CRITICAL vulnerabilities"
        fi
        
        if [ "${high_count}" -gt 0 ]; then
            violations=$((violations + high_count))
            details="${details}${image}: ${high_count} HIGH vulnerabilities. "
            log "${YELLOW}" "    ⚠ ${high_count} HIGH vulnerabilities"
        fi
        
        if [ "${critical_count}" -eq 0 ] && [ "${high_count}" -eq 0 ]; then
            log "${GREEN}" "    ✓ No critical or high vulnerabilities"
        fi
    done
    
    if [ ${violations} -eq 0 ]; then
        add_to_report "vulnerability_scan" "PASS" "No critical or high vulnerabilities found"
        return 0
    else
        add_to_report "vulnerability_scan" "FAIL" "${details}"
        return 1
    fi
}

# Function to check compliance with specific standards
check_compliance_standards() {
    log "${BLUE}" "Checking Compliance Standards..."
    
    local soc2_pass=true
    local iso27001_pass=true
    local pcidss_pass=true
    local gdpr_pass=true
    
    # SOC2 checks
    log "${BLUE}" "  SOC2 Type 2 Compliance:"
    if check_rbac_compliance &>/dev/null && check_audit_logging &>/dev/null; then
        log "${GREEN}" "    ✓ Access controls and monitoring in place"
    else
        soc2_pass=false
        log "${RED}" "    ✗ SOC2 requirements not met"
    fi
    
    # ISO 27001 checks
    log "${BLUE}" "  ISO 27001 Compliance:"
    if check_container_security &>/dev/null && check_network_policies &>/dev/null; then
        log "${GREEN}" "    ✓ Security controls implemented"
    else
        iso27001_pass=false
        log "${RED}" "    ✗ ISO 27001 requirements not met"
    fi
    
    # PCI-DSS checks
    log "${BLUE}" "  PCI-DSS v4 Compliance:"
    if check_tls_configuration &>/dev/null && check_secrets_management &>/dev/null; then
        log "${GREEN}" "    ✓ Data protection controls in place"
    else
        pcidss_pass=false
        log "${RED}" "    ✗ PCI-DSS requirements not met"
    fi
    
    # GDPR checks
    log "${BLUE}" "  GDPR Compliance:"
    local pii_encryption=$(kubectl get configmap -n "${NAMESPACE}" gdpr-config -o json 2>/dev/null | \
        jq -r '.data.encryption_enabled')
    
    if [ "${pii_encryption}" == "true" ]; then
        log "${GREEN}" "    ✓ PII encryption enabled"
    else
        gdpr_pass=false
        log "${RED}" "    ✗ GDPR requirements not met"
    fi
    
    local compliance_summary=""
    if ${soc2_pass} && ${iso27001_pass} && ${pcidss_pass} && ${gdpr_pass}; then
        compliance_summary="All compliance standards met"
        add_to_report "compliance_standards" "PASS" "${compliance_summary}"
        return 0
    else
        compliance_summary="Some compliance standards not met"
        add_to_report "compliance_standards" "FAIL" "${compliance_summary}"
        return 1
    fi
}

# Function to generate summary
generate_summary() {
    # Close the JSON structure
    echo "  }," >> "${REPORT_FILE}"
    
    # Add summary
    echo "  \"summary\": {" >> "${REPORT_FILE}"
    echo "    \"total_checks\": 9," >> "${REPORT_FILE}"
    echo "    \"passed\": $(grep -c '"PASS"' "${REPORT_FILE}")," >> "${REPORT_FILE}"
    echo "    \"failed\": $(grep -c '"FAIL"' "${REPORT_FILE}")," >> "${REPORT_FILE}"
    echo "    \"skipped\": $(grep -c '"SKIP"' "${REPORT_FILE}")" >> "${REPORT_FILE}"
    echo "  }" >> "${REPORT_FILE}"
    echo "}" >> "${REPORT_FILE}"
}

# Main execution
main() {
    log "${GREEN}" "==============================================="
    log "${GREEN}" "Nephoran Intent Operator Compliance Validation"
    log "${GREEN}" "==============================================="
    log "${BLUE}" "Timestamp: ${TIMESTAMP}"
    log "${BLUE}" "Namespace: ${NAMESPACE}"
    echo ""
    
    local total_failures=0
    
    # Run all compliance checks
    check_rbac_compliance || ((total_failures++))
    echo ""
    
    check_network_policies || ((total_failures++))
    echo ""
    
    check_container_security || ((total_failures++))
    echo ""
    
    check_secrets_management || ((total_failures++))
    echo ""
    
    check_tls_configuration || ((total_failures++))
    echo ""
    
    check_oran_compliance || ((total_failures++))
    echo ""
    
    check_audit_logging || ((total_failures++))
    echo ""
    
    run_vulnerability_scan || ((total_failures++))
    echo ""
    
    check_compliance_standards || ((total_failures++))
    echo ""
    
    # Generate summary
    generate_summary
    
    log "${GREEN}" "==============================================="
    if [ ${total_failures} -eq 0 ]; then
        log "${GREEN}" "✓ All compliance checks passed!"
    else
        log "${RED}" "✗ ${total_failures} compliance check(s) failed"
        log "${YELLOW}" "Review the report at: ${REPORT_FILE}"
    fi
    log "${GREEN}" "==============================================="
    
    # Upload report if configured
    if [ -n "${COMPLIANCE_REPORT_UPLOAD_URL:-}" ]; then
        log "${BLUE}" "Uploading compliance report..."
        curl -X POST -H "Content-Type: application/json" \
            -d @"${REPORT_FILE}" \
            "${COMPLIANCE_REPORT_UPLOAD_URL}" || \
            log "${YELLOW}" "Failed to upload compliance report"
    fi
    
    # Exit with appropriate code
    exit ${total_failures}
}

# Run main function
main "$@"
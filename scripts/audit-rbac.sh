#!/bin/bash

# Nephoran Intent Operator RBAC Audit Script
# Purpose: Audit RBAC permissions, identify unused permissions, and validate least-privilege
# Security: Implements O-RAN security requirements and zero-trust principles

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-nephoran-system}"
AUDIT_DIR="${AUDIT_DIR:-./rbac-audit-$(date +%Y%m%d-%H%M%S)}"
KUBECTL="${KUBECTL:-kubectl}"
AUDIT_DAYS="${AUDIT_DAYS:-30}"  # Days to look back for audit logs

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create audit directory
mkdir -p "${AUDIT_DIR}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Nephoran RBAC Security Audit${NC}"
echo -e "${BLUE}========================================${NC}"
echo "Namespace: ${NAMESPACE}"
echo "Audit Directory: ${AUDIT_DIR}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to log messages
log() {
    echo -e "$1" | tee -a "${AUDIT_DIR}/audit.log"
}

# Function to check expired bindings
check_expired_bindings() {
    log "${YELLOW}Checking for expired role bindings...${NC}"
    
    # Get all role bindings with expiry annotations
    ${KUBECTL} get rolebindings,clusterrolebindings -A \
        -o json | jq -r '.items[] | 
        select(.metadata.annotations."security.nephoran.io/expires" != null) | 
        {
            kind: .kind,
            name: .metadata.name,
            namespace: .metadata.namespace,
            expires: .metadata.annotations."security.nephoran.io/expires"
        }' > "${AUDIT_DIR}/bindings-with-expiry.json"
    
    # Check for expired bindings
    current_date=$(date +%Y-%m-%d)
    expired_count=0
    
    while IFS= read -r binding; do
        expires=$(echo "$binding" | jq -r '.expires')
        name=$(echo "$binding" | jq -r '.name')
        kind=$(echo "$binding" | jq -r '.kind')
        
        if [[ "$expires" < "$current_date" ]]; then
            log "${RED}EXPIRED: ${kind} ${name} (expired: ${expires})${NC}"
            echo "$binding" >> "${AUDIT_DIR}/expired-bindings.json"
            ((expired_count++))
        fi
    done < <(cat "${AUDIT_DIR}/bindings-with-expiry.json" | jq -c '.')
    
    if [ $expired_count -eq 0 ]; then
        log "${GREEN}✓ No expired role bindings found${NC}"
    else
        log "${RED}✗ Found ${expired_count} expired role bindings${NC}"
    fi
    echo ""
}

# Function to audit service account usage
audit_service_accounts() {
    log "${YELLOW}Auditing service account usage...${NC}"
    
    # Get all service accounts
    ${KUBECTL} get serviceaccounts -n ${NAMESPACE} -o json > "${AUDIT_DIR}/service-accounts.json"
    
    # Check which service accounts are actually used by pods
    ${KUBECTL} get pods -n ${NAMESPACE} -o json | \
        jq -r '.items[].spec.serviceAccountName' | \
        sort | uniq > "${AUDIT_DIR}/used-service-accounts.txt"
    
    # Find unused service accounts
    ${KUBECTL} get serviceaccounts -n ${NAMESPACE} -o json | \
        jq -r '.items[].metadata.name' | \
        while read -r sa; do
            if ! grep -q "^${sa}$" "${AUDIT_DIR}/used-service-accounts.txt"; then
                if [[ "$sa" != "default" ]]; then
                    log "${YELLOW}⚠ Potentially unused service account: ${sa}${NC}"
                    echo "$sa" >> "${AUDIT_DIR}/unused-service-accounts.txt"
                fi
            fi
        done
    
    echo ""
}

# Function to check for overly permissive roles
check_permissive_roles() {
    log "${YELLOW}Checking for overly permissive roles...${NC}"
    
    # Check for wildcard permissions
    ${KUBECTL} get roles,clusterroles -A -o json | \
        jq -r '.items[] | 
        select(.rules[]? | 
            (.apiGroups[]? == "*") or 
            (.resources[]? == "*") or 
            (.verbs[]? == "*")
        ) | {
            kind: .kind,
            name: .metadata.name,
            namespace: .metadata.namespace
        }' > "${AUDIT_DIR}/wildcard-permissions.json"
    
    if [ -s "${AUDIT_DIR}/wildcard-permissions.json" ]; then
        log "${RED}✗ Found roles with wildcard permissions:${NC}"
        cat "${AUDIT_DIR}/wildcard-permissions.json" | \
            jq -r '"  - \(.kind): \(.name) (namespace: \(.namespace // "cluster-wide"))"' | \
            while read -r line; do
                log "$line"
            done
    else
        log "${GREEN}✓ No wildcard permissions found${NC}"
    fi
    
    # Check for cluster-admin bindings
    ${KUBECTL} get clusterrolebindings -o json | \
        jq -r '.items[] | 
        select(.roleRef.name == "cluster-admin") | 
        {
            name: .metadata.name,
            subjects: .subjects
        }' > "${AUDIT_DIR}/cluster-admin-bindings.json"
    
    if [ -s "${AUDIT_DIR}/cluster-admin-bindings.json" ] && \
       [ "$(jq -r '.subjects[]?' "${AUDIT_DIR}/cluster-admin-bindings.json" | wc -l)" -gt 0 ]; then
        log "${RED}✗ Found cluster-admin bindings:${NC}"
        cat "${AUDIT_DIR}/cluster-admin-bindings.json" | \
            jq -r '"  - Binding: \(.name)"' | \
            while read -r line; do
                log "$line"
            done
    else
        log "${GREEN}✓ No inappropriate cluster-admin bindings found${NC}"
    fi
    
    echo ""
}

# Function to analyze actual API calls vs granted permissions
analyze_api_usage() {
    log "${YELLOW}Analyzing API usage vs granted permissions...${NC}"
    
    # This requires audit logs to be enabled
    if ${KUBECTL} get --raw /api/v1/namespaces/kube-system/services/kube-apiserver/proxy/logs/audit > /dev/null 2>&1; then
        # Fetch recent audit logs
        ${KUBECTL} logs -n kube-system -l component=kube-apiserver \
            --since=${AUDIT_DAYS}d --prefix=true 2>/dev/null | \
            grep -E "audit.*nephoran" > "${AUDIT_DIR}/api-audit.log" || true
        
        if [ -s "${AUDIT_DIR}/api-audit.log" ]; then
            # Extract API calls by service account
            grep -oP 'user\.username="system:serviceaccount:[^:]+:\K[^"]+' \
                "${AUDIT_DIR}/api-audit.log" | \
                sort | uniq -c | sort -rn > "${AUDIT_DIR}/api-calls-by-sa.txt"
            
            log "${GREEN}✓ API usage analysis saved to ${AUDIT_DIR}/api-calls-by-sa.txt${NC}"
        else
            log "${YELLOW}⚠ No audit logs found for Nephoran components${NC}"
        fi
    else
        log "${YELLOW}⚠ Audit logs not accessible - enable audit logging for detailed analysis${NC}"
    fi
    
    echo ""
}

# Function to check for default service account usage
check_default_sa_usage() {
    log "${YELLOW}Checking for default service account usage...${NC}"
    
    default_usage=$(${KUBECTL} get pods -n ${NAMESPACE} -o json | \
        jq -r '.items[] | 
        select(.spec.serviceAccountName == "default" or .spec.serviceAccountName == null) | 
        .metadata.name' | wc -l)
    
    if [ "$default_usage" -gt 0 ]; then
        log "${RED}✗ Found ${default_usage} pods using default service account${NC}"
        ${KUBECTL} get pods -n ${NAMESPACE} -o json | \
            jq -r '.items[] | 
            select(.spec.serviceAccountName == "default" or .spec.serviceAccountName == null) | 
            .metadata.name' > "${AUDIT_DIR}/pods-using-default-sa.txt"
    else
        log "${GREEN}✓ No pods using default service account${NC}"
    fi
    
    echo ""
}

# Function to validate RBAC best practices
validate_best_practices() {
    log "${YELLOW}Validating RBAC best practices...${NC}"
    
    # Check for automountServiceAccountToken
    ${KUBECTL} get serviceaccounts -n ${NAMESPACE} -o json | \
        jq -r '.items[] | 
        select(.automountServiceAccountToken != false) | 
        .metadata.name' > "${AUDIT_DIR}/sa-with-automount.txt"
    
    if [ -s "${AUDIT_DIR}/sa-with-automount.txt" ]; then
        log "${YELLOW}⚠ Service accounts with automountServiceAccountToken enabled:${NC}"
        while read -r sa; do
            log "  - ${sa}"
        done < "${AUDIT_DIR}/sa-with-automount.txt"
    fi
    
    # Check for missing MFA requirements on admin roles
    ${KUBECTL} get rolebindings,clusterrolebindings -A -o json | \
        jq -r '.items[] | 
        select(.metadata.name | contains("admin")) | 
        select(.metadata.annotations."security.nephoran.io/require-mfa" != "true") | 
        .metadata.name' > "${AUDIT_DIR}/admin-without-mfa.txt"
    
    if [ -s "${AUDIT_DIR}/admin-without-mfa.txt" ]; then
        log "${RED}✗ Admin role bindings without MFA requirement:${NC}"
        while read -r binding; do
            log "  - ${binding}"
        done < "${AUDIT_DIR}/admin-without-mfa.txt"
    else
        log "${GREEN}✓ All admin bindings require MFA${NC}"
    fi
    
    echo ""
}

# Function to generate permission matrix
generate_permission_matrix() {
    log "${YELLOW}Generating permission matrix...${NC}"
    
    # Create a matrix of service accounts and their permissions
    echo "Service Account,Role,API Groups,Resources,Verbs" > "${AUDIT_DIR}/permission-matrix.csv"
    
    ${KUBECTL} get rolebindings -n ${NAMESPACE} -o json | \
        jq -r '.items[] | 
        {
            sa: .subjects[]? | select(.kind == "ServiceAccount") | .name,
            role: .roleRef.name
        } | "\(.sa),\(.role)"' | \
        while IFS=',' read -r sa role; do
            # Get role details
            ${KUBECTL} get role "${role}" -n ${NAMESPACE} -o json 2>/dev/null | \
                jq -r --arg sa "$sa" --arg role "$role" '.rules[]? | 
                "\($sa),\($role),\(.apiGroups | join(";")),\(.resources | join(";")),\(.verbs | join(";"))"' \
                >> "${AUDIT_DIR}/permission-matrix.csv"
        done
    
    log "${GREEN}✓ Permission matrix saved to ${AUDIT_DIR}/permission-matrix.csv${NC}"
    echo ""
}

# Function to check O-RAN specific RBAC
check_oran_rbac() {
    log "${YELLOW}Checking O-RAN specific RBAC configuration...${NC}"
    
    # Check for RIC operator roles
    ric_roles=$(${KUBECTL} get clusterroles -o json | \
        jq -r '.items[] | select(.metadata.name | contains("ric")) | .metadata.name' | wc -l)
    
    if [ "$ric_roles" -gt 0 ]; then
        log "${GREEN}✓ Found ${ric_roles} RIC-related roles${NC}"
    else
        log "${YELLOW}⚠ No RIC-specific roles found${NC}"
    fi
    
    # Check for network function operator roles
    nf_roles=$(${KUBECTL} get clusterroles -o json | \
        jq -r '.items[] | select(.metadata.name | contains("nf-operator")) | .metadata.name' | wc -l)
    
    if [ "$nf_roles" -gt 0 ]; then
        log "${GREEN}✓ Found ${nf_roles} network function operator roles${NC}"
    else
        log "${YELLOW}⚠ No network function operator roles found${NC}"
    fi
    
    # Check for O-RAN interface permissions
    oran_perms=$(${KUBECTL} get roles,clusterroles -A -o json | \
        jq -r '.items[].rules[]? | 
        select(.apiGroups[]? | contains("oran")) | 
        .resources[]' | sort | uniq | wc -l)
    
    log "${BLUE}ℹ Found permissions for ${oran_perms} O-RAN resource types${NC}"
    
    echo ""
}

# Function to generate recommendations
generate_recommendations() {
    log "${YELLOW}Generating security recommendations...${NC}"
    
    {
        echo "# RBAC Security Audit Recommendations"
        echo ""
        echo "## Summary"
        echo "Audit Date: $(date)"
        echo "Namespace: ${NAMESPACE}"
        echo ""
        
        echo "## Critical Findings"
        if [ -s "${AUDIT_DIR}/expired-bindings.json" ]; then
            echo "- **CRITICAL**: Found expired role bindings that should be renewed or removed"
        fi
        if [ -s "${AUDIT_DIR}/cluster-admin-bindings.json" ]; then
            echo "- **CRITICAL**: Found cluster-admin bindings - review for necessity"
        fi
        if [ -s "${AUDIT_DIR}/pods-using-default-sa.txt" ]; then
            echo "- **HIGH**: Pods using default service account - assign specific service accounts"
        fi
        
        echo ""
        echo "## Recommendations"
        echo ""
        echo "### Immediate Actions"
        echo "1. Remove or renew expired role bindings"
        echo "2. Replace default service account usage with dedicated service accounts"
        echo "3. Review and remove unnecessary wildcard permissions"
        echo "4. Enable MFA for all administrative role bindings"
        echo ""
        
        echo "### Medium-term Actions"
        echo "1. Implement automated RBAC expiry monitoring"
        echo "2. Create fine-grained roles for each component"
        echo "3. Implement audit logging for all API access"
        echo "4. Regular RBAC reviews (monthly recommended)"
        echo ""
        
        echo "### Long-term Strategy"
        echo "1. Implement zero-trust network policies alongside RBAC"
        echo "2. Use external identity providers with OIDC"
        echo "3. Implement privileged access management (PAM)"
        echo "4. Automate compliance validation"
        echo ""
        
        echo "## O-RAN Compliance"
        echo "- Ensure RIC components have dedicated service accounts"
        echo "- Implement interface-specific permissions (A1, O1, O2, E2)"
        echo "- Follow O-RAN WG11 security specifications"
        echo "- Maintain audit trails for compliance"
        
    } > "${AUDIT_DIR}/recommendations.md"
    
    log "${GREEN}✓ Recommendations saved to ${AUDIT_DIR}/recommendations.md${NC}"
    echo ""
}

# Function to generate audit report
generate_report() {
    log "${YELLOW}Generating final audit report...${NC}"
    
    {
        echo "# Nephoran RBAC Audit Report"
        echo ""
        echo "## Executive Summary"
        echo "Date: $(date)"
        echo "Namespace: ${NAMESPACE}"
        echo ""
        
        echo "## Findings"
        echo ""
        
        echo "### Service Accounts"
        sa_total=$(${KUBECTL} get serviceaccounts -n ${NAMESPACE} --no-headers | wc -l)
        echo "- Total service accounts: ${sa_total}"
        if [ -s "${AUDIT_DIR}/unused-service-accounts.txt" ]; then
            unused_count=$(wc -l < "${AUDIT_DIR}/unused-service-accounts.txt")
            echo "- Unused service accounts: ${unused_count}"
        fi
        echo ""
        
        echo "### Role Bindings"
        rb_total=$(${KUBECTL} get rolebindings -n ${NAMESPACE} --no-headers | wc -l)
        crb_total=$(${KUBECTL} get clusterrolebindings --no-headers | wc -l)
        echo "- Role bindings: ${rb_total}"
        echo "- Cluster role bindings: ${crb_total}"
        if [ -s "${AUDIT_DIR}/expired-bindings.json" ]; then
            expired_count=$(jq -s 'length' "${AUDIT_DIR}/expired-bindings.json")
            echo "- Expired bindings: ${expired_count}"
        fi
        echo ""
        
        echo "### Security Issues"
        if [ -s "${AUDIT_DIR}/wildcard-permissions.json" ]; then
            wildcard_count=$(jq -s 'length' "${AUDIT_DIR}/wildcard-permissions.json")
            echo "- Roles with wildcard permissions: ${wildcard_count}"
        fi
        if [ -s "${AUDIT_DIR}/cluster-admin-bindings.json" ]; then
            ca_count=$(jq -s 'length' "${AUDIT_DIR}/cluster-admin-bindings.json")
            echo "- Cluster-admin bindings: ${ca_count}"
        fi
        echo ""
        
        echo "## Files Generated"
        echo "- audit.log - Main audit log"
        echo "- permission-matrix.csv - Service account permission matrix"
        echo "- recommendations.md - Security recommendations"
        echo "- expired-bindings.json - List of expired bindings"
        echo "- wildcard-permissions.json - Roles with wildcard permissions"
        
    } > "${AUDIT_DIR}/report.md"
    
    log "${GREEN}✓ Audit report saved to ${AUDIT_DIR}/report.md${NC}"
}

# Main execution
main() {
    log "${BLUE}Starting RBAC audit...${NC}"
    echo ""
    
    # Check prerequisites
    if ! command_exists jq; then
        log "${RED}Error: jq is required but not installed${NC}"
        exit 1
    fi
    
    # Run audit checks
    check_expired_bindings
    audit_service_accounts
    check_permissive_roles
    analyze_api_usage
    check_default_sa_usage
    validate_best_practices
    generate_permission_matrix
    check_oran_rbac
    generate_recommendations
    generate_report
    
    log "${GREEN}========================================${NC}"
    log "${GREEN}RBAC audit completed successfully!${NC}"
    log "${GREEN}Results saved to: ${AUDIT_DIR}${NC}"
    log "${GREEN}========================================${NC}"
    
    # Return non-zero if critical issues found
    if [ -s "${AUDIT_DIR}/expired-bindings.json" ] || \
       [ -s "${AUDIT_DIR}/cluster-admin-bindings.json" ]; then
        exit 1
    fi
}

# Run main function
main "$@"
#!/bin/bash
# Execute comprehensive security audit for Nephoran Intent Operator
# Phase 3 Production Excellence - Security Penetration Testing
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-nephoran-system}"
REPORT_DIR="${REPORT_DIR:-/tmp/nephoran-security-audit}"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
AUDIT_ID="SEC-AUDIT-$TIMESTAMP"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

info() {
    echo -e "${PURPLE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Initialize audit environment
initialize_audit_environment() {
    log "Initializing security audit environment..."
    
    # Create report directory structure
    mkdir -p "$REPORT_DIR"/{config,vulnerabilities,penetration,compliance,summary}
    
    # Create audit metadata
    cat > "$REPORT_DIR/audit-metadata.json" <<EOF
{
    "audit_id": "$AUDIT_ID",
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "auditor": "$(whoami)",
    "hostname": "$(hostname)",
    "kubernetes_version": "$(kubectl version --short --client | grep Client | cut -d' ' -f3)",
    "cluster_info": {
        "nodes": $(kubectl get nodes --no-headers | wc -l),
        "namespaces": $(kubectl get namespaces --no-headers | wc -l),
        "pods": $(kubectl get pods --all-namespaces --no-headers | wc -l)
    }
}
EOF
    
    success "Audit environment initialized with ID: $AUDIT_ID"
}

# Execute security configuration validation
execute_config_validation() {
    log "Executing security configuration validation..."
    
    local config_report="$REPORT_DIR/config/security-config-validation.json"
    
    # Run security config validator
    if [[ -f "$SCRIPT_DIR/security-config-validator.sh" ]]; then
        REPORT_DIR="$REPORT_DIR/config" "$SCRIPT_DIR/security-config-validator.sh" > "$REPORT_DIR/config/validation.log" 2>&1
        local config_result=$?
        
        log "Security configuration validation completed with exit code: $config_result"
        
        # Parse results and create summary
        if [[ -f "$REPORT_DIR/config/security-assessment-"*".json" ]]; then
            local latest_config=$(ls -t "$REPORT_DIR/config/security-assessment-"*".json" | head -1)
            cp "$latest_config" "$config_report"
            
            local config_score=$(jq -r '.assessment_summary.security_score' "$config_report")
            log "Configuration security score: $config_score/100"
        else
            warn "Security configuration report not found"
            config_result=1
        fi
    else
        error "Security config validator script not found"
        config_result=1
    fi
    
    return $config_result
}

# Execute vulnerability scanning
execute_vulnerability_scan() {
    log "Executing comprehensive vulnerability scanning..."
    
    local vuln_report="$REPORT_DIR/vulnerabilities/vulnerability-assessment.json"
    
    # Run vulnerability scanner
    if [[ -f "$SCRIPT_DIR/vulnerability-scanner.sh" ]]; then
        REPORT_DIR="$REPORT_DIR/vulnerabilities" "$SCRIPT_DIR/vulnerability-scanner.sh" > "$REPORT_DIR/vulnerabilities/scan.log" 2>&1
        local vuln_result=$?
        
        log "Vulnerability scanning completed with exit code: $vuln_result"
        
        # Parse results and create summary
        if [[ -f "$REPORT_DIR/vulnerabilities/vulnerability-assessment-"*".json" ]]; then
            local latest_vuln=$(ls -t "$REPORT_DIR/vulnerabilities/vulnerability-assessment-"*".json" | head -1)
            cp "$latest_vuln" "$vuln_report"
            
            local vuln_score=$(jq -r '.vulnerability_assessment.vulnerability_score' "$vuln_report")
            local total_issues=$(jq -r '.vulnerability_assessment.total_issues' "$vuln_report")
            log "Vulnerability score: $vuln_score/100 (Total issues: $total_issues)"
        else
            warn "Vulnerability assessment report not found"
            vuln_result=1
        fi
    else
        error "Vulnerability scanner script not found"
        vuln_result=1
    fi
    
    return $vuln_result
}

# Execute penetration testing
execute_penetration_testing() {
    log "Executing security penetration testing..."
    
    local pen_report="$REPORT_DIR/penetration/penetration-test-results.json"
    
    # Run main penetration test
    if [[ -f "$SCRIPT_DIR/security-penetration-test.sh" ]]; then
        REPORT_DIR="$REPORT_DIR/penetration" "$SCRIPT_DIR/security-penetration-test.sh" > "$REPORT_DIR/penetration/pentest.log" 2>&1
        local pen_result=$?
        
        log "Penetration testing completed with exit code: $pen_result"
        
        # Parse results and create summary
        if [[ -f "$REPORT_DIR/penetration/penetration-test-"*".json" ]]; then
            local latest_pen=$(ls -t "$REPORT_DIR/penetration/penetration-test-"*".json" | head -1)
            cp "$latest_pen" "$pen_report"
            
            local pen_score=$(jq -r '.penetration_test_summary.security_score' "$pen_report")
            local vulnerabilities_found=$(jq -r '.penetration_test_summary.vulnerabilities_found' "$pen_report")
            log "Penetration test score: $pen_score/100 (Vulnerabilities found: $vulnerabilities_found)"
        else
            warn "Penetration test report not found"
            pen_result=1
        fi
    else
        error "Penetration test script not found"
        pen_result=1
    fi
    
    return $pen_result
}

# Execute compliance validation
execute_compliance_validation() {
    log "Executing security compliance validation..."
    
    local compliance_report="$REPORT_DIR/compliance/compliance-assessment.json"
    local compliance_score=0
    local compliance_issues=0
    
    # Check for required security policies
    local required_policies=(
        "networkpolicy"
        "podsecuritypolicy"
        "limitrange"
        "resourcequota"
    )
    
    local policy_compliance=0
    for policy in "${required_policies[@]}"; do
        local policy_count=$(kubectl get "$policy" -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
        if [[ $policy_count -gt 0 ]]; then
            policy_compliance=$((policy_compliance + 25))
            success "Found $policy_count $policy resources"
        else
            warn "No $policy resources found"
            compliance_issues=$((compliance_issues + 1))
        fi
    done
    
    # Check for security annotations and labels
    local security_annotations=0
    local pods_with_security=$(kubectl get pods -n "$NAMESPACE" -o json | \
        jq -r '.items[] | select(.metadata.annotations."security.nephoran.com/scanned" == "true") | .metadata.name' | wc -l)
    
    if [[ $pods_with_security -gt 0 ]]; then
        security_annotations=25
        success "Found $pods_with_security pods with security annotations"
    else
        warn "No pods with security annotations found"
        compliance_issues=$((compliance_issues + 1))
    fi
    
    # Check for monitoring integration
    local monitoring_compliance=0
    local security_monitors=$(kubectl get servicemonitor -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    
    if [[ $security_monitors -gt 0 ]]; then
        monitoring_compliance=25
        success "Found $security_monitors security monitoring configurations"
    else
        warn "No security monitoring configurations found"
        compliance_issues=$((compliance_issues + 1))
    fi
    
    # Check for backup and disaster recovery
    local backup_compliance=0
    local backup_configs=$(kubectl get cronjob -n "$NAMESPACE" -l "app.kubernetes.io/component=backup" --no-headers 2>/dev/null | wc -l)
    
    if [[ $backup_configs -gt 0 ]]; then
        backup_compliance=25
        success "Found $backup_configs backup configurations"
    else
        warn "No backup configurations found"
        compliance_issues=$((compliance_issues + 1))
    fi
    
    compliance_score=$((policy_compliance + security_annotations + monitoring_compliance + backup_compliance))
    
    # Generate compliance report
    cat > "$compliance_report" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "compliance_assessment": {
        "total_score": $compliance_score,
        "max_score": 100,
        "compliance_issues": $compliance_issues,
        "status": "$([ $compliance_score -ge 80 ] && echo "COMPLIANT" || echo "NON_COMPLIANT")"
    },
    "policy_compliance": {
        "score": $policy_compliance,
        "required_policies": $(printf '%s\n' "${required_policies[@]}" | jq -R . | jq -s .)
    },
    "security_annotations": {
        "score": $security_annotations,
        "pods_with_annotations": $pods_with_security
    },
    "monitoring_compliance": {
        "score": $monitoring_compliance,
        "security_monitors": $security_monitors
    },
    "backup_compliance": {
        "score": $backup_compliance,
        "backup_configs": $backup_configs
    }
}
EOF
    
    log "Compliance validation completed - Score: $compliance_score/100"
    
    if [[ $compliance_score -ge 80 ]]; then
        return 0
    else
        return 1
    fi
}

# Generate comprehensive security audit report
generate_audit_report() {
    log "Generating comprehensive security audit report..."
    
    local final_report="$REPORT_DIR/summary/security-audit-$AUDIT_ID.json"
    local executive_summary="$REPORT_DIR/summary/executive-summary-$AUDIT_ID.md"
    
    # Collect scores from individual assessments
    local config_score=0
    local vuln_score=0
    local pen_score=0
    local compliance_score=0
    
    # Extract configuration score
    if [[ -f "$REPORT_DIR/config/security-config-validation.json" ]]; then
        config_score=$(jq -r '.assessment_summary.security_score // 0' "$REPORT_DIR/config/security-config-validation.json")
    fi
    
    # Extract vulnerability score
    if [[ -f "$REPORT_DIR/vulnerabilities/vulnerability-assessment.json" ]]; then
        vuln_score=$(jq -r '.vulnerability_assessment.vulnerability_score // 0' "$REPORT_DIR/vulnerabilities/vulnerability-assessment.json")
    fi
    
    # Extract penetration test score
    if [[ -f "$REPORT_DIR/penetration/penetration-test-results.json" ]]; then
        pen_score=$(jq -r '.penetration_test_summary.security_score // 0' "$REPORT_DIR/penetration/penetration-test-results.json")
    fi
    
    # Extract compliance score
    if [[ -f "$REPORT_DIR/compliance/compliance-assessment.json" ]]; then
        compliance_score=$(jq -r '.compliance_assessment.total_score // 0' "$REPORT_DIR/compliance/compliance-assessment.json")
    fi
    
    # Calculate overall security score (weighted average)
    local overall_score=$(echo "scale=2; ($config_score * 0.3 + $vuln_score * 0.3 + $pen_score * 0.25 + $compliance_score * 0.15)" | bc)
    local rounded_score=$(printf "%.0f" "$overall_score")
    
    # Determine security status
    local security_status="FAIL"
    if [[ $rounded_score -ge 95 ]]; then
        security_status="EXCELLENT"
    elif [[ $rounded_score -ge 85 ]]; then
        security_status="GOOD"
    elif [[ $rounded_score -ge 70 ]]; then
        security_status="ACCEPTABLE"
    elif [[ $rounded_score -ge 50 ]]; then
        security_status="POOR"
    fi
    
    # Generate final JSON report
    cat > "$final_report" <<EOF
{
    "audit_metadata": $(cat "$REPORT_DIR/audit-metadata.json"),
    "overall_assessment": {
        "security_score": $rounded_score,
        "max_score": 100,
        "security_status": "$security_status",
        "passed": $([ $rounded_score -ge 85 ] && echo "true" || echo "false")
    },
    "component_scores": {
        "configuration_security": {
            "score": $config_score,
            "weight": 30,
            "contribution": $(echo "scale=2; $config_score * 0.3" | bc)
        },
        "vulnerability_assessment": {
            "score": $vuln_score,
            "weight": 30,
            "contribution": $(echo "scale=2; $vuln_score * 0.3" | bc)
        },
        "penetration_testing": {
            "score": $pen_score,
            "weight": 25,
            "contribution": $(echo "scale=2; $pen_score * 0.25" | bc)
        },
        "compliance_validation": {
            "score": $compliance_score,
            "weight": 15,
            "contribution": $(echo "scale=2; $compliance_score * 0.15" | bc)
        }
    },
    "recommendations": [
        $([ $config_score -lt 85 ] && echo "\"Improve security configuration and hardening\",")
        $([ $vuln_score -lt 85 ] && echo "\"Address identified vulnerabilities and update dependencies\",")
        $([ $pen_score -lt 85 ] && echo "\"Strengthen security controls against penetration attempts\",")
        $([ $compliance_score -lt 85 ] && echo "\"Implement missing compliance requirements\",")
        "\"Continue regular security assessments and monitoring\",",
        "\"Implement automated security testing in CI/CD pipeline\",",
        "\"Establish incident response procedures for security events\""
    ],
    "next_audit_recommended": "$(date -d '+90 days' -Iseconds)"
}
EOF
    
    # Generate executive summary
    cat > "$executive_summary" <<EOF
# Nephoran Intent Operator Security Audit Report
**Audit ID:** $AUDIT_ID  
**Date:** $(date +'%Y-%m-%d %H:%M:%S')  
**Namespace:** $NAMESPACE  

## Executive Summary

The comprehensive security audit of the Nephoran Intent Operator has been completed with an overall security score of **$rounded_score/100** classified as **$security_status**.

### Component Assessment Results

| Component | Score | Status |
|-----------|-------|--------|
| Configuration Security | $config_score/100 | $([ $config_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") |
| Vulnerability Assessment | $vuln_score/100 | $([ $vuln_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") |
| Penetration Testing | $pen_score/100 | $([ $pen_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") |
| Compliance Validation | $compliance_score/100 | $([ $compliance_score -ge 85 ] && echo "✅ PASS" || echo "❌ FAIL") |

### Key Findings

$([ $rounded_score -ge 95 ] && echo "🎉 **EXCELLENT SECURITY POSTURE** - The system demonstrates exceptional security controls and practices.")
$([ $rounded_score -ge 85 ] && [ $rounded_score -lt 95 ] && echo "✅ **GOOD SECURITY POSTURE** - The system meets production security requirements with minor improvements needed.")
$([ $rounded_score -ge 70 ] && [ $rounded_score -lt 85 ] && echo "⚠️ **ACCEPTABLE SECURITY** - The system has adequate security but requires attention to reach production standards.")
$([ $rounded_score -lt 70 ] && echo "🚨 **SECURITY CONCERNS** - The system requires immediate attention to address significant security gaps.")

### Recommendations

1. **Immediate Actions** (if score < 85):
   - Address critical vulnerabilities identified in the vulnerability assessment
   - Implement missing security configurations and policies
   - Strengthen access controls and network segmentation

2. **Short-term Improvements** (1-3 months):
   - Implement automated security scanning in CI/CD pipeline
   - Enhance monitoring and alerting for security events
   - Conduct security training for development and operations teams

3. **Long-term Strategy** (3-12 months):
   - Establish regular penetration testing schedule
   - Implement zero-trust security architecture
   - Develop incident response and disaster recovery procedures

### Next Steps

- **Next Audit Recommended:** $(date -d '+90 days' +'%Y-%m-%d')
- **Immediate Remediation Required:** $([ $rounded_score -ge 85 ] && echo "No" || echo "Yes")
- **Production Deployment Ready:** $([ $rounded_score -ge 85 ] && echo "Yes" || echo "No")

---
*Report generated by Nephoran Security Audit Framework v3.0*
EOF
    
    success "Comprehensive security audit report generated:"
    success "  Overall Score: $rounded_score/100 ($security_status)"
    success "  JSON Report: $final_report"
    success "  Executive Summary: $executive_summary"
    
    # Return success if score >= 85
    if [[ $rounded_score -ge 85 ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution function
main() {
    log "=========================================="
    log "Nephoran Intent Operator Security Audit"
    log "Phase 3 Production Excellence - Security Testing"
    log "=========================================="
    
    # Initialize audit environment
    initialize_audit_environment
    
    # Execute security assessments
    local overall_result=0
    
    log "Step 1: Security Configuration Validation"
    if ! execute_config_validation; then
        error "Security configuration validation failed"
        overall_result=1
    fi
    
    log "Step 2: Vulnerability Assessment"
    if ! execute_vulnerability_scan; then
        error "Vulnerability assessment failed"
        overall_result=1
    fi
    
    log "Step 3: Penetration Testing"
    if ! execute_penetration_testing; then
        error "Penetration testing failed"
        overall_result=1
    fi
    
    log "Step 4: Compliance Validation"
    if ! execute_compliance_validation; then
        error "Compliance validation failed"
        overall_result=1
    fi
    
    log "Step 5: Generating Comprehensive Report"
    if ! generate_audit_report; then
        error "Security audit failed overall assessment"
        overall_result=1
    fi
    
    # Final status
    if [[ $overall_result -eq 0 ]]; then
        success "=========================================="
        success "Security audit PASSED - Production Ready"
        success "=========================================="
    else
        error "=========================================="
        error "Security audit FAILED - Remediation Required"
        error "=========================================="
    fi
    
    log "Audit complete. Reports available in: $REPORT_DIR"
    return $overall_result
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
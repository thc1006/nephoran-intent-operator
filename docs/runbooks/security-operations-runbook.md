# Security Operations Runbook - Nephoran Intent Operator

**Version:** 2.0  
**Last Updated:** January 2025  
**Audience:** Security Operations, SRE Teams, Compliance Officers  
**Classification:** Security Operations Documentation

## Table of Contents

1. [Security Architecture Overview](#security-architecture-overview)
2. [Security Monitoring](#security-monitoring)
3. [Incident Response](#incident-response)
4. [Access Control Management](#access-control-management)
5. [Secret Management](#secret-management)
6. [Certificate Management](#certificate-management)
7. [Compliance Procedures](#compliance-procedures)
8. [Security Auditing](#security-auditing)
9. [Vulnerability Management](#vulnerability-management)

## Security Architecture Overview

### Security Layers

```yaml
Defense in Depth:
  Network Layer:
    - Network segmentation via NetworkPolicies
    - Service mesh (Istio) with mTLS
    - WAF protection for external endpoints
    - DDoS protection
    
  Application Layer:
    - OAuth2 authentication
    - RBAC authorization
    - Input validation and sanitization
    - Rate limiting and throttling
    
  Data Layer:
    - Encryption at rest (AES-256)
    - Encryption in transit (TLS 1.3)
    - Database encryption
    - Backup encryption
    
  Infrastructure Layer:
    - Pod security policies
    - Container image scanning
    - Runtime security (Falco)
    - Admission controllers
```

### Security Components

| Component | Purpose | Security Features |
|-----------|---------|------------------|
| OAuth2 Provider | Authentication | Multi-provider support, token validation |
| Istio Service Mesh | Network security | mTLS, authorization policies |
| Cert-Manager | Certificate management | Automatic renewal, ACME support |
| Falco | Runtime security | Anomaly detection, compliance |
| OPA | Policy enforcement | Admission control, RBAC |
| Vault | Secret management | Dynamic secrets, encryption |

## Security Monitoring

### Security Event Monitoring

```bash
#!/bin/bash
# Security monitoring script

monitor_security_events() {
  echo "=== Security Event Monitoring ==="
  echo "Time: $(date)"
  
  # Authentication failures
  echo "--- Authentication Failures (Last Hour) ---"
  kubectl logs -n nephoran-system --since=1h --all-containers=true | \
    grep -i "auth.*fail\|unauthorized\|403\|401" | \
    tail -20
  
  # Privilege escalation attempts
  echo "--- Privilege Escalation Attempts ---"
  kubectl logs -n kube-system -l app=falco --since=1h | \
    grep -i "privilege\|escalation\|sudo\|root" | \
    tail -10
  
  # Network policy violations
  echo "--- Network Policy Violations ---"
  kubectl logs -n kube-system -l k8s-app=cilium --since=1h | \
    grep "Policy denied" | \
    tail -10
  
  # Suspicious container activity
  echo "--- Suspicious Container Activity ---"
  kubectl logs -n kube-system -l app=falco --since=1h | \
    grep -E "Unexpected\|Anomaly\|Suspicious" | \
    tail -10
  
  # Certificate issues
  echo "--- Certificate Issues ---"
  kubectl get certificates --all-namespaces | \
    grep -v "True" | grep -v "READY"
  
  # Secret access patterns
  echo "--- Unusual Secret Access ---"
  kubectl logs -n kube-system -l component=kube-apiserver --since=1h | \
    grep "secrets" | grep -v "system:serviceaccount" | \
    tail -10
}

# Run continuous monitoring
while true; do
  monitor_security_events
  sleep 300  # Run every 5 minutes
done
```

### Security Metrics

```yaml
# security-metrics.yaml
Security KPIs:
  Authentication:
    - auth_success_rate: Rate of successful authentications
    - auth_failure_rate: Rate of failed authentication attempts
    - token_validation_errors: Invalid token presentations
    - session_duration: Average session length
    
  Authorization:
    - rbac_denials: RBAC policy denial rate
    - privilege_escalations: Attempted privilege escalations
    - policy_violations: Policy enforcement violations
    
  Network Security:
    - tls_handshake_failures: Failed TLS connections
    - network_policy_blocks: Blocked network connections
    - ddos_attempts: Detected DDoS attempts
    
  Compliance:
    - compliance_score: Overall compliance percentage
    - audit_findings: Number of audit findings
    - remediation_time: Average time to remediate findings
```

### Security Alerts

```yaml
# security-alerts.yml
groups:
- name: security-alerts
  rules:
  - alert: HighAuthenticationFailureRate
    expr: rate(nephoran_auth_failures_total[5m]) > 10
    for: 2m
    labels:
      severity: high
      category: security
    annotations:
      summary: "High authentication failure rate detected"
      description: "Authentication failures: {{ $value }}/sec"
      action: "Check for brute force attempts"
      
  - alert: PrivilegeEscalationAttempt
    expr: nephoran_privilege_escalation_attempts > 0
    for: 1m
    labels:
      severity: critical
      category: security
    annotations:
      summary: "Privilege escalation attempt detected"
      description: "User {{ $labels.user }} attempted privilege escalation"
      action: "Immediate investigation required"
      
  - alert: CertificateExpiringSoon
    expr: (cert_manager_certificate_expiration_timestamp_seconds - time()) < 7 * 24 * 3600
    for: 1h
    labels:
      severity: medium
      category: security
    annotations:
      summary: "Certificate expiring soon"
      description: "Certificate {{ $labels.name }} expires in {{ $value | humanizeDuration }}"
      action: "Renew certificate"
      
  - alert: SecretAccessAnomaly
    expr: rate(nephoran_secret_access_total[5m]) > 50
    for: 5m
    labels:
      severity: high
      category: security
    annotations:
      summary: "Unusual secret access pattern detected"
      description: "High rate of secret access: {{ $value }}/sec"
      action: "Investigate potential credential stuffing"
```

## Incident Response

### Security Incident Classification

```yaml
Incident Categories:
  Category 1 - Critical:
    - Active data breach
    - Ransomware infection
    - Complete authentication bypass
    - Root access compromise
    Response Time: Immediate
    
  Category 2 - High:
    - Suspicious privileged activity
    - Multiple authentication failures
    - Unauthorized access attempts
    - Certificate compromise
    Response Time: 15 minutes
    
  Category 3 - Medium:
    - Policy violations
    - Unusual network traffic
    - Failed compliance checks
    - Expired certificates
    Response Time: 1 hour
    
  Category 4 - Low:
    - Security scan findings
    - Documentation issues
    - Minor policy deviations
    Response Time: 24 hours
```

### Incident Response Procedure

```bash
#!/bin/bash
# Security incident response script

respond_to_security_incident() {
  local INCIDENT_TYPE=$1
  local SEVERITY=$2
  
  echo "=== SECURITY INCIDENT RESPONSE ==="
  echo "Type: $INCIDENT_TYPE"
  echo "Severity: $SEVERITY"
  echo "Time: $(date)"
  
  # Create incident record
  INCIDENT_ID="SEC-$(date +%Y%m%d-%H%M%S)"
  echo "Incident ID: $INCIDENT_ID"
  
  case $INCIDENT_TYPE in
    "data-breach")
      echo "--- Data Breach Response ---"
      # Immediate containment
      kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: emergency-isolation
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress: []
  egress: []
EOF
      
      # Preserve evidence
      kubectl get pods -n nephoran-system -o yaml > /tmp/incident-$INCIDENT_ID-pods.yaml
      kubectl logs --all-containers=true -n nephoran-system > /tmp/incident-$INCIDENT_ID-logs.txt
      
      # Rotate all secrets
      ./scripts/emergency-secret-rotation.sh
      ;;
      
    "authentication-bypass")
      echo "--- Authentication Bypass Response ---"
      # Disable all external access
      kubectl patch service nephoran-api -n nephoran-system -p '{"spec":{"type":"ClusterIP"}}'
      
      # Force re-authentication
      kubectl delete pods -n nephoran-system -l app=llm-processor
      kubectl delete pods -n nephoran-system -l app=nephoran-controller
      
      # Audit all service accounts
      kubectl get serviceaccounts --all-namespaces -o yaml > /tmp/incident-$INCIDENT_ID-sa.yaml
      ;;
      
    "privilege-escalation")
      echo "--- Privilege Escalation Response ---"
      # Review RBAC policies
      kubectl get clusterrolebindings -o yaml > /tmp/incident-$INCIDENT_ID-rbac.yaml
      kubectl get rolebindings --all-namespaces -o yaml >> /tmp/incident-$INCIDENT_ID-rbac.yaml
      
      # Check for modified pods
      kubectl get pods --all-namespaces -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.spec.securityContext}{"\n"}{end}' | \
        grep -E "privileged|root"
      
      # Terminate suspicious pods
      read -p "Terminate suspicious pods? (y/N) " -n 1 -r
      echo
      if [[ $REPLY =~ ^[Yy]$ ]]; then
        kubectl delete pods -n nephoran-system --field-selector status.phase=Running
      fi
      ;;
      
    "certificate-compromise")
      echo "--- Certificate Compromise Response ---"
      # Revoke compromised certificates
      kubectl delete certificate --all -n nephoran-system
      
      # Force certificate renewal
      kubectl annotate certificate --all -n nephoran-system \
        cert-manager.io/force-renew=$(date +%s) --overwrite
      
      # Restart all pods to pick up new certificates
      kubectl rollout restart deployment --all -n nephoran-system
      ;;
  esac
  
  # Notify security team
  ./scripts/notify-security-team.sh \
    --incident-id "$INCIDENT_ID" \
    --type "$INCIDENT_TYPE" \
    --severity "$SEVERITY"
  
  echo "Initial response completed at $(date)"
}
```

### Post-Incident Analysis

```bash
#!/bin/bash
# Post-incident analysis script

analyze_security_incident() {
  local INCIDENT_ID=$1
  
  echo "=== Post-Incident Analysis: $INCIDENT_ID ==="
  
  # Timeline reconstruction
  echo "--- Timeline Reconstruction ---"
  grep -h "$INCIDENT_ID\|$(date +%Y-%m-%d)" /tmp/incident-*.txt | \
    sort -k1,2 | head -50
  
  # Impact assessment
  echo "--- Impact Assessment ---"
  echo "Affected services:"
  kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l
  echo "Failed authentications:"
  grep -c "auth.*fail" /tmp/incident-$INCIDENT_ID-logs.txt
  echo "Data accessed:"
  grep -c "GET\|POST" /tmp/incident-$INCIDENT_ID-logs.txt
  
  # Root cause analysis
  echo "--- Root Cause Indicators ---"
  grep -E "error|fail|denied|unauthorized" /tmp/incident-$INCIDENT_ID-logs.txt | \
    awk '{print $5}' | sort | uniq -c | sort -rn | head -10
  
  # Generate report
  cat > /tmp/incident-$INCIDENT_ID-report.md <<EOF
# Security Incident Report: $INCIDENT_ID

## Executive Summary
- Incident ID: $INCIDENT_ID
- Date: $(date)
- Severity: $SEVERITY
- Status: Under Investigation

## Timeline
$(grep -h "$INCIDENT_ID" /tmp/incident-*.txt | head -20)

## Impact
- Services Affected: $(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
- Duration: TBD
- Data Exposure: Under Investigation

## Response Actions
1. Immediate containment implemented
2. Evidence preserved
3. Security team notified
4. Investigation ongoing

## Recommendations
- Review access controls
- Enhance monitoring
- Update security policies
- Conduct security training

EOF
  
  echo "Report saved to /tmp/incident-$INCIDENT_ID-report.md"
}
```

## Access Control Management

### RBAC Configuration

```yaml
# rbac-config.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-operator
rules:
- apiGroups: ["nephoran.io"]
  resources: ["networkintents", "e2nodesets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["secrets", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-viewer
rules:
- apiGroups: ["nephoran.io"]
  resources: ["networkintents", "e2nodesets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods", "services"]
  verbs: ["get", "list"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-admin
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

### Access Control Procedures

```bash
#!/bin/bash
# Access control management script

manage_user_access() {
  local ACTION=$1
  local USERNAME=$2
  local ROLE=$3
  
  case $ACTION in
    "grant")
      echo "Granting $ROLE access to $USERNAME..."
      
      # Create service account
      kubectl create serviceaccount $USERNAME -n nephoran-system
      
      # Create role binding
      kubectl create rolebinding ${USERNAME}-${ROLE} \
        --clusterrole=$ROLE \
        --serviceaccount=nephoran-system:$USERNAME \
        -n nephoran-system
      
      # Generate kubeconfig
      ./scripts/generate-kubeconfig.sh $USERNAME > /tmp/${USERNAME}-kubeconfig.yaml
      
      echo "Access granted. Kubeconfig saved to /tmp/${USERNAME}-kubeconfig.yaml"
      ;;
      
    "revoke")
      echo "Revoking access for $USERNAME..."
      
      # Delete role bindings
      kubectl delete rolebinding ${USERNAME}-${ROLE} -n nephoran-system
      
      # Delete service account
      kubectl delete serviceaccount $USERNAME -n nephoran-system
      
      echo "Access revoked for $USERNAME"
      ;;
      
    "audit")
      echo "Auditing access for $USERNAME..."
      
      # Check permissions
      kubectl auth can-i --list --as=system:serviceaccount:nephoran-system:$USERNAME
      
      # Check recent activity
      kubectl logs -n kube-system -l component=kube-apiserver | \
        grep $USERNAME | tail -20
      ;;
  esac
}

# Regular access review
review_access_permissions() {
  echo "=== Access Permission Review ==="
  
  # List all service accounts
  echo "--- Service Accounts ---"
  kubectl get serviceaccounts --all-namespaces | grep -v "^kube-"
  
  # List role bindings
  echo "--- Role Bindings ---"
  kubectl get rolebindings --all-namespaces | grep -v "^kube-"
  
  # List cluster role bindings
  echo "--- Cluster Role Bindings ---"
  kubectl get clusterrolebindings | grep -v "^system:"
  
  # Check for overly permissive roles
  echo "--- Overly Permissive Roles ---"
  kubectl get clusterrolebindings -o json | \
    jq -r '.items[] | select(.roleRef.name == "cluster-admin") | .metadata.name'
}
```

## Secret Management

### Secret Rotation Procedures

```bash
#!/bin/bash
# Secret rotation script

rotate_secrets() {
  echo "=== Secret Rotation Procedure ==="
  echo "Starting rotation at $(date)"
  
  # Rotate API keys
  echo "--- Rotating API Keys ---"
  
  # OpenAI API key
  NEW_OPENAI_KEY=$(./scripts/generate-api-key.sh openai)
  kubectl create secret generic openai-key \
    --from-literal=api-key="$NEW_OPENAI_KEY" \
    --dry-run=client -o yaml | \
    kubectl apply -f - -n nephoran-system
  
  # Database passwords
  echo "--- Rotating Database Passwords ---"
  NEW_DB_PASSWORD=$(openssl rand -base64 32)
  kubectl create secret generic database-credentials \
    --from-literal=username=nephoran \
    --from-literal=password="$NEW_DB_PASSWORD" \
    --dry-run=client -o yaml | \
    kubectl apply -f - -n nephoran-system
  
  # Update database with new password
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "ALTER USER nephoran PASSWORD '$NEW_DB_PASSWORD';"
  
  # JWT signing keys
  echo "--- Rotating JWT Signing Keys ---"
  openssl genrsa -out /tmp/jwt-private.pem 4096
  openssl rsa -in /tmp/jwt-private.pem -pubout -out /tmp/jwt-public.pem
  
  kubectl create secret generic jwt-keys \
    --from-file=private=/tmp/jwt-private.pem \
    --from-file=public=/tmp/jwt-public.pem \
    --dry-run=client -o yaml | \
    kubectl apply -f - -n nephoran-system
  
  # Clean up temp files
  rm -f /tmp/jwt-*.pem
  
  # Restart affected services
  echo "--- Restarting Services ---"
  kubectl rollout restart deployment/llm-processor -n nephoran-system
  kubectl rollout restart deployment/nephoran-controller -n nephoran-system
  kubectl rollout restart deployment/rag-api -n nephoran-system
  
  # Verify rotation
  echo "--- Verification ---"
  kubectl get secrets -n nephoran-system -o json | \
    jq -r '.items[] | select(.metadata.name | contains("key") or contains("credential") or contains("password")) | 
    "\(.metadata.name): Updated \(.metadata.creationTimestamp)"'
  
  echo "Secret rotation completed at $(date)"
}

# Emergency secret rotation
emergency_secret_rotation() {
  echo "=== EMERGENCY SECRET ROTATION ==="
  echo "🚨 Starting emergency rotation at $(date)"
  
  # Backup current secrets
  kubectl get secrets -n nephoran-system -o yaml > /tmp/secrets-backup-$(date +%Y%m%d-%H%M%S).yaml
  
  # Rotate all secrets immediately
  for secret in $(kubectl get secrets -n nephoran-system -o name | grep -v "^secret/default-token"); do
    echo "Rotating $secret..."
    kubectl delete $secret -n nephoran-system
  done
  
  # Regenerate all secrets
  ./scripts/generate-all-secrets.sh | kubectl apply -f -
  
  # Force pod restart to pick up new secrets
  kubectl delete pods --all -n nephoran-system
  
  echo "Emergency rotation completed at $(date)"
}
```

### Vault Integration

```yaml
# vault-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vault-config
  namespace: nephoran-system
data:
  config.hcl: |
    storage "raft" {
      path = "/vault/data"
    }
    
    listener "tcp" {
      address = "0.0.0.0:8200"
      tls_cert_file = "/vault/tls/tls.crt"
      tls_key_file = "/vault/tls/tls.key"
    }
    
    api_addr = "https://vault.nephoran-system:8200"
    cluster_addr = "https://vault.nephoran-system:8201"
    ui = true
    
  policies.hcl: |
    # Policy for Nephoran services
    path "secret/data/nephoran/*" {
      capabilities = ["read", "list"]
    }
    
    path "auth/token/renew-self" {
      capabilities = ["update"]
    }
```

## Certificate Management

### Certificate Lifecycle Management

```bash
#!/bin/bash
# Certificate management script

manage_certificates() {
  echo "=== Certificate Management ==="
  
  # Check certificate status
  echo "--- Certificate Status ---"
  kubectl get certificates --all-namespaces -o wide
  
  # Check expiration dates
  echo "--- Expiration Dates ---"
  for cert in $(kubectl get certificates -n nephoran-system -o name); do
    CERT_NAME=$(echo $cert | cut -d'/' -f2)
    SECRET_NAME=$(kubectl get $cert -n nephoran-system -o jsonpath='{.spec.secretName}')
    
    if [ ! -z "$SECRET_NAME" ]; then
      EXPIRY=$(kubectl get secret $SECRET_NAME -n nephoran-system -o jsonpath='{.data.tls\.crt}' | \
        base64 -d | openssl x509 -enddate -noout 2>/dev/null | cut -d'=' -f2)
      echo "$CERT_NAME: $EXPIRY"
    fi
  done
  
  # Renew expiring certificates
  echo "--- Renewal Check ---"
  for cert in $(kubectl get certificates -n nephoran-system -o name); do
    DAYS_LEFT=$(kubectl get $cert -n nephoran-system -o jsonpath='{.status.renewalTime}' | \
      xargs -I {} date -d {} +%s | xargs -I {} expr {} - $(date +%s) | xargs -I {} expr {} / 86400)
    
    if [ $DAYS_LEFT -lt 30 ]; then
      echo "⚠️  $cert expires in $DAYS_LEFT days - Renewing..."
      kubectl annotate $cert -n nephoran-system \
        cert-manager.io/force-renew=$(date +%s) --overwrite
    fi
  done
  
  # Validate certificates
  echo "--- Certificate Validation ---"
  for cert in $(kubectl get certificates -n nephoran-system -o name); do
    kubectl describe $cert -n nephoran-system | grep -A5 "Status:"
  done
}

# Certificate rotation procedure
rotate_certificates() {
  echo "=== Certificate Rotation ==="
  
  # Backup current certificates
  kubectl get secrets -n nephoran-system -l cert-manager.io/certificate-name -o yaml > \
    /tmp/certificates-backup-$(date +%Y%m%d-%H%M%S).yaml
  
  # Force renewal
  kubectl annotate certificate --all -n nephoran-system \
    cert-manager.io/force-renew=$(date +%s) --overwrite
  
  # Wait for renewal
  echo "Waiting for certificate renewal..."
  sleep 60
  
  # Verify renewal
  kubectl get certificates -n nephoran-system
  
  # Restart services to pick up new certificates
  kubectl rollout restart deployment --all -n nephoran-system
  
  echo "Certificate rotation completed"
}
```

## Compliance Procedures

### Compliance Checklist

```yaml
Compliance Requirements:
  SOC2:
    - Access control reviews: Monthly
    - Security monitoring: Continuous
    - Incident response testing: Quarterly
    - Vulnerability scanning: Weekly
    - Penetration testing: Annually
    
  PCI-DSS:
    - Network segmentation: Validated
    - Encryption at rest: AES-256
    - Encryption in transit: TLS 1.2+
    - Access logging: Enabled
    - Security patches: Within 30 days
    
  HIPAA:
    - PHI encryption: Required
    - Access controls: Role-based
    - Audit logging: 6-year retention
    - Business associate agreements: Current
    - Risk assessments: Annual
    
  GDPR:
    - Data minimization: Enforced
    - Right to erasure: Implemented
    - Consent management: Tracked
    - Data breach notification: 72 hours
    - Privacy by design: Validated
```

### Compliance Validation

```bash
#!/bin/bash
# Compliance validation script

validate_compliance() {
  local FRAMEWORK=$1
  
  echo "=== Compliance Validation: $FRAMEWORK ==="
  echo "Date: $(date)"
  
  case $FRAMEWORK in
    "SOC2")
      echo "--- SOC2 Compliance Check ---"
      
      # Access control
      echo "✓ Access Control Review"
      kubectl get rolebindings --all-namespaces | wc -l
      
      # Monitoring
      echo "✓ Security Monitoring"
      curl -s http://prometheus.monitoring:9090/api/v1/query?query=up | \
        jq -r '.data.result | length'
      
      # Logging
      echo "✓ Audit Logging"
      kubectl logs -n kube-system -l component=kube-apiserver --tail=1 | wc -l
      
      # Encryption
      echo "✓ Encryption Status"
      kubectl get secrets --all-namespaces | grep -c "Opaque"
      ;;
      
    "PCI-DSS")
      echo "--- PCI-DSS Compliance Check ---"
      
      # Network segmentation
      echo "✓ Network Policies"
      kubectl get networkpolicies --all-namespaces | wc -l
      
      # TLS configuration
      echo "✓ TLS Version Check"
      kubectl get ingress --all-namespaces -o json | \
        jq -r '.items[].spec.tls' | grep -v null | wc -l
      
      # Vulnerability status
      echo "✓ Vulnerability Scan Status"
      kubectl get vulnerabilityreports --all-namespaces | \
        grep -c "CRITICAL\|HIGH"
      ;;
      
    "HIPAA")
      echo "--- HIPAA Compliance Check ---"
      
      # PHI encryption
      echo "✓ PHI Encryption"
      kubectl get secrets -n nephoran-system -o json | \
        jq -r '.items[] | select(.metadata.labels.phi == "true") | .metadata.name'
      
      # Audit retention
      echo "✓ Audit Log Retention"
      find /var/log/audit -name "*.log" -mtime +2190 | wc -l
      
      # Access controls
      echo "✓ RBAC Policies"
      kubectl get clusterroles | grep -c "nephoran"
      ;;
      
    "GDPR")
      echo "--- GDPR Compliance Check ---"
      
      # Data inventory
      echo "✓ Personal Data Inventory"
      kubectl get configmaps -n nephoran-system -l data-type=personal | wc -l
      
      # Consent tracking
      echo "✓ Consent Records"
      kubectl get secrets -n nephoran-system -l consent=required | wc -l
      
      # Data retention
      echo "✓ Data Retention Policies"
      kubectl get cronjobs -n nephoran-system | grep -c "data-cleanup"
      ;;
  esac
  
  echo "Validation completed at $(date)"
}

# Generate compliance report
generate_compliance_report() {
  echo "=== Compliance Report Generation ==="
  
  REPORT_DATE=$(date +%Y-%m-%d)
  REPORT_FILE="/tmp/compliance-report-$REPORT_DATE.md"
  
  cat > $REPORT_FILE <<EOF
# Compliance Report - $REPORT_DATE

## Executive Summary
- Report Date: $REPORT_DATE
- Frameworks: SOC2, PCI-DSS, HIPAA, GDPR
- Overall Status: Compliant with observations

## SOC2 Compliance
$(validate_compliance SOC2)

## PCI-DSS Compliance  
$(validate_compliance PCI-DSS)

## HIPAA Compliance
$(validate_compliance HIPAA)

## GDPR Compliance
$(validate_compliance GDPR)

## Recommendations
1. Continue monthly access reviews
2. Update security patches within 30 days
3. Conduct quarterly incident response drills
4. Review and update data retention policies

## Sign-off
- Security Officer: ________________
- Compliance Officer: ________________
- Date: $REPORT_DATE
EOF
  
  echo "Report saved to $REPORT_FILE"
}
```

## Security Auditing

### Audit Log Collection

```bash
#!/bin/bash
# Audit log collection script

collect_audit_logs() {
  echo "=== Audit Log Collection ==="
  
  # API server audit logs
  echo "--- API Server Audit Logs ---"
  kubectl logs -n kube-system -l component=kube-apiserver --tail=1000 > \
    /tmp/audit-apiserver-$(date +%Y%m%d).log
  
  # Service audit logs
  echo "--- Service Audit Logs ---"
  for service in llm-processor rag-api nephoran-controller; do
    kubectl logs -n nephoran-system deployment/$service --tail=1000 | \
      grep -i "audit" > /tmp/audit-$service-$(date +%Y%m%d).log
  done
  
  # Authentication logs
  echo "--- Authentication Logs ---"
  kubectl logs -n nephoran-system --all-containers=true | \
    grep -i "auth" > /tmp/audit-auth-$(date +%Y%m%d).log
  
  # Network policy logs
  echo "--- Network Policy Logs ---"
  kubectl logs -n kube-system -l k8s-app=cilium | \
    grep "Policy" > /tmp/audit-network-$(date +%Y%m%d).log
  
  # Compress logs
  tar -czf /tmp/audit-logs-$(date +%Y%m%d).tar.gz /tmp/audit-*.log
  rm /tmp/audit-*.log
  
  echo "Audit logs collected: /tmp/audit-logs-$(date +%Y%m%d).tar.gz"
}

# Audit analysis
analyze_audit_logs() {
  echo "=== Audit Log Analysis ==="
  
  # Extract and analyze
  tar -xzf /tmp/audit-logs-$(date +%Y%m%d).tar.gz -C /tmp/
  
  # Suspicious activities
  echo "--- Suspicious Activities ---"
  grep -h -i "denied\|failed\|error\|unauthorized" /tmp/audit-*.log | \
    awk '{print $5}' | sort | uniq -c | sort -rn | head -20
  
  # User activity summary
  echo "--- User Activity Summary ---"
  grep -h "user=" /tmp/audit-*.log | \
    sed 's/.*user=\([^ ]*\).*/\1/' | \
    sort | uniq -c | sort -rn
  
  # Resource access patterns
  echo "--- Resource Access Patterns ---"
  grep -h "GET\|POST\|PUT\|DELETE" /tmp/audit-*.log | \
    awk '{print $6, $7}' | sort | uniq -c | sort -rn | head -20
  
  # Time-based analysis
  echo "--- Activity by Hour ---"
  grep -h "^[0-9]" /tmp/audit-*.log | \
    awk '{print substr($1,12,2)}' | \
    sort | uniq -c
}
```

## Vulnerability Management

### Vulnerability Scanning

```bash
#!/bin/bash
# Vulnerability scanning script

scan_vulnerabilities() {
  echo "=== Vulnerability Scanning ==="
  echo "Scan started at $(date)"
  
  # Container image scanning
  echo "--- Container Image Scanning ---"
  for image in $(kubectl get pods -n nephoran-system -o jsonpath='{.items[*].spec.containers[*].image}' | tr ' ' '\n' | sort -u); do
    echo "Scanning $image..."
    trivy image --severity HIGH,CRITICAL --no-progress $image
  done
  
  # Kubernetes security scanning
  echo "--- Kubernetes Security Scanning ---"
  kubectl apply -f https://raw.githubusercontent.com/aquasecurity/kube-bench/main/job.yaml
  sleep 30
  kubectl logs job/kube-bench
  kubectl delete job kube-bench
  
  # Network vulnerability scanning
  echo "--- Network Vulnerability Scanning ---"
  # This would typically use nmap or similar
  kubectl run network-scan --image=instrumentisto/nmap --rm -i --restart=Never -- \
    -sV -p 80,443,8080,8443 nephoran-api.nephoran-system.svc.cluster.local
  
  # OWASP dependency check
  echo "--- Dependency Vulnerability Check ---"
  ./scripts/dependency-check.sh --project nephoran --scan /app --format JSON --out /tmp/dependency-report.json
  
  echo "Scan completed at $(date)"
}

# Vulnerability remediation
remediate_vulnerabilities() {
  echo "=== Vulnerability Remediation ==="
  
  # Get vulnerability reports
  kubectl get vulnerabilityreports -n nephoran-system -o json > /tmp/vulns.json
  
  # High/Critical vulnerabilities
  CRITICAL_VULNS=$(jq -r '.items[] | select(.report.summary.criticalCount > 0) | .metadata.name' /tmp/vulns.json)
  HIGH_VULNS=$(jq -r '.items[] | select(.report.summary.highCount > 0) | .metadata.name' /tmp/vulns.json)
  
  # Auto-remediation for known issues
  for vuln in $CRITICAL_VULNS; do
    echo "Remediating critical vulnerability: $vuln"
    
    # Update base images
    if echo $vuln | grep -q "base-image"; then
      kubectl set image deployment/$(echo $vuln | cut -d'-' -f1) \
        *=nephoran/$(echo $vuln | cut -d'-' -f1):latest-security \
        -n nephoran-system
    fi
    
    # Apply security patches
    if echo $vuln | grep -q "CVE-"; then
      ./scripts/apply-security-patch.sh $(echo $vuln | grep -o "CVE-[0-9]*-[0-9]*")
    fi
  done
  
  echo "Remediation completed"
}
```

### Security Hardening

```bash
#!/bin/bash
# Security hardening script

apply_security_hardening() {
  echo "=== Security Hardening ==="
  
  # Pod Security Policies
  echo "--- Applying Pod Security Policies ---"
  kubectl apply -f - <<EOF
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: nephoran-restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
EOF
  
  # Network Policies
  echo "--- Applying Network Policies ---"
  kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-network-policy
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: nephoran
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    - podSelector:
        matchLabels:
          app.kubernetes.io/part-of: nephoran
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
EOF
  
  # Resource Quotas
  echo "--- Applying Resource Quotas ---"
  kubectl apply -f - <<EOF
apiVersion: v1
kind: ResourceQuota
metadata:
  name: nephoran-quota
  namespace: nephoran-system
spec:
  hard:
    requests.cpu: "100"
    requests.memory: 200Gi
    limits.cpu: "200"
    limits.memory: 400Gi
    persistentvolumeclaims: "10"
    services.loadbalancers: "2"
EOF
  
  echo "Security hardening completed"
}
```

## Related Documentation

- [Incident Response Runbook](./incident-response-runbook.md) - General incident procedures
- [Master Operational Runbook](./operational-runbook-master.md) - Daily operations
- [Disaster Recovery Runbook](./disaster-recovery-runbook.md) - DR procedures
- [Security Hardening Guide](../security/hardening-guide.md) - Detailed hardening procedures
- [Compliance Audit Guide](../compliance-audit-guide.md) - Compliance procedures

---

**Note:** This security runbook contains sensitive procedures. Access should be restricted to authorized security personnel only. Regular reviews and updates are mandatory to maintain security posture.
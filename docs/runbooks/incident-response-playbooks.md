# Nephoran Intent Operator - Incident Response Playbooks
## TRL 9 Production-Ready 24/7 Operations

### Document Overview
**Version:** 2.1  
**Last Updated:** 2024-08-08  
**Classification:** Internal Operations  
**Review Cycle:** Monthly  

This document provides comprehensive incident response procedures for the Nephoran Intent Operator platform, ensuring rapid resolution of production issues while maintaining 99.95% availability SLA.

---

## 1. Incident Response Framework

### 1.1 Severity Classification

| Severity | Definition | Response Time | Escalation |
|----------|------------|---------------|------------|
| **Critical (P1)** | System down, data loss, security breach | 15 minutes | Immediate |
| **High (P2)** | Major functionality impaired, SLA breach | 1 hour | 2 hours |
| **Medium (P3)** | Minor functionality impaired, degraded performance | 4 hours | 1 business day |
| **Low (P4)** | Cosmetic issues, feature requests | 1 business day | 1 week |

### 1.2 Response Team Structure

**Incident Commander:** Senior Operations Engineer  
**Technical Lead:** Platform Architect  
**Subject Matter Experts:**
- LLM/RAG Specialist
- O-RAN Interface Expert
- Kubernetes/Infrastructure Engineer
- Security Engineer
- Network Specialist

### 1.3 Communication Channels

- **Primary:** PagerDuty alerts
- **War Room:** Slack #nephoran-incidents
- **Executive:** Teams #nephoran-executive
- **Customer:** Status page updates
- **Engineering:** Discord #engineering-ops

---

## 2. P1 Critical Incident Playbooks

### 2.1 System Down - Complete Service Outage

**Alert:** `NephoranSystemDown`  
**Symptoms:** No intent processing, health checks failing  
**Business Impact:** Complete service unavailability  

#### Immediate Response (0-5 minutes)

1. **Acknowledge Alert**
   ```bash
   # Check overall system status
   kubectl get pods -n nephoran-system
   kubectl get nodes
   curl -f http://nephoran-api/health
   ```

2. **Establish War Room**
   ```bash
   # Send incident notification
   ./scripts/incident/notify-war-room.sh P1 "System Down" "Complete service outage"
   ```

3. **Quick Health Assessment**
   ```bash
   # Check critical components
   kubectl describe deployment nephoran-intent-controller -n nephoran-system
   kubectl logs -l app=nephoran-intent-controller -n nephoran-system --tail=100
   ```

#### Detailed Investigation (5-15 minutes)

4. **Infrastructure Check**
   ```bash
   # Kubernetes cluster health
   kubectl get componentstatuses
   kubectl top nodes
   kubectl get events --sort-by='.lastTimestamp' | tail -20
   
   # Database connectivity
   kubectl exec -it deployment/nephoran-db -n nephoran-system -- pg_isready
   
   # Network connectivity
   kubectl exec -it deployment/test-pod -n nephoran-system -- nslookup kubernetes.default.svc.cluster.local
   ```

5. **Application Health**
   ```bash
   # Check application logs
   kubectl logs deployment/nephoran-intent-controller -n nephoran-system --since=30m
   kubectl logs deployment/llm-processor -n nephoran-system --since=30m
   kubectl logs deployment/rag-api -n nephoran-system --since=30m
   
   # Check for recent configuration changes
   kubectl get configmaps -n nephoran-system -o yaml | grep -A5 -B5 "lastUpdate"
   ```

#### Resolution Actions

6. **Common Fixes**
   ```bash
   # Restart stuck controllers
   kubectl rollout restart deployment/nephoran-intent-controller -n nephoran-system
   
   # Scale up if resource constrained
   kubectl scale deployment/nephoran-intent-controller --replicas=5 -n nephoran-system
   
   # Clear stuck resources
   kubectl delete pod -l app=nephoran-intent-controller --field-selector=status.phase=Failed -n nephoran-system
   ```

7. **Emergency Rollback**
   ```bash
   # Rollback to previous version
   kubectl rollout undo deployment/nephoran-intent-controller -n nephoran-system
   kubectl rollout status deployment/nephoran-intent-controller -n nephoran-system
   ```

#### Post-Resolution (Within 1 hour)

8. **Verify Recovery**
   ```bash
   # Test end-to-end functionality
   ./scripts/e2e/verify-intent-processing.sh
   
   # Check metrics recovery
   curl -s http://prometheus:9090/api/v1/query?query=up{job="nephoran-intent-controller"}
   ```

9. **Customer Communication**
   ```bash
   # Update status page
   ./scripts/incident/update-status-page.sh "Resolved" "Service fully restored"
   
   # Send resolution notice
   ./scripts/incident/send-customer-update.sh "resolved" "incident_12345"
   ```

### 2.2 Data Loss or Corruption

**Alert:** `DataCorruptionDetected` or `BackupFailure`  
**Symptoms:** Data inconsistencies, failed integrity checks  
**Business Impact:** Data loss, regulatory compliance risk  

#### Immediate Response (0-10 minutes)

1. **Stop All Writes**
   ```bash
   # Scale down intent controller to prevent further corruption
   kubectl scale deployment/nephoran-intent-controller --replicas=0 -n nephoran-system
   
   # Enable read-only mode
   kubectl patch configmap/nephoran-config -n nephoran-system -p '{"data":{"read_only":"true"}}'
   ```

2. **Assess Scope**
   ```bash
   # Check database integrity
   kubectl exec -it deployment/nephoran-db -n nephoran-system -- \
     psql -U nephoran -d nephoran -c "SELECT pg_check_integrity();"
   
   # Verify backup availability
   ./scripts/backup/list-available-backups.sh
   ```

#### Recovery Actions

3. **Point-in-Time Recovery**
   ```bash
   # Restore from latest clean backup
   ./scripts/backup/restore-database.sh --timestamp="$(date -d '1 hour ago' -Iseconds)"
   
   # Verify data integrity post-restore
   ./scripts/verification/check-data-integrity.sh
   ```

4. **Resume Operations**
   ```bash
   # Disable read-only mode
   kubectl patch configmap/nephoran-config -n nephoran-system -p '{"data":{"read_only":"false"}}'
   
   # Scale up controllers
   kubectl scale deployment/nephoran-intent-controller --replicas=3 -n nephoran-system
   ```

### 2.3 Security Breach

**Alert:** `SecurityPolicyViolation` or `UnauthorizedAccess`  
**Symptoms:** Unusual access patterns, policy violations  
**Business Impact:** Data confidentiality breach, compliance violation  

#### Immediate Response (0-5 minutes)

1. **Isolate Affected Systems**
   ```bash
   # Block suspicious IPs
   kubectl apply -f security/emergency-network-policies.yaml
   
   # Rotate all API keys
   ./scripts/security/emergency-key-rotation.sh
   ```

2. **Preserve Evidence**
   ```bash
   # Capture current state
   kubectl logs -l app=nephoran-api -n nephoran-system --since=1h > /tmp/security-incident-logs.txt
   
   # Export audit logs
   ./scripts/audit/export-audit-logs.sh --since="1 hour ago" --output=/tmp/audit-export.json
   ```

#### Investigation and Containment

3. **Threat Assessment**
   ```bash
   # Check for privilege escalation
   kubectl auth can-i --list --as=system:serviceaccount:default:suspicious-account
   
   # Review recent RBAC changes
   kubectl get rolebindings,clusterrolebindings -A -o yaml | grep -A10 -B10 "$(date -d '1 day ago' +%Y-%m-%d)"
   ```

4. **Containment Actions**
   ```bash
   # Disable affected service accounts
   kubectl patch serviceaccount/suspicious-account -n nephoran-system -p '{"metadata":{"labels":{"disabled":"true"}}}'
   
   # Enable enhanced monitoring
   kubectl apply -f monitoring/security-enhanced-config.yaml
   ```

---

## 3. P2 High Severity Playbooks

### 3.1 High Latency Performance Degradation

**Alert:** `IntentProcessingHighLatency`  
**Symptoms:** P95 latency > 2s, user complaints  
**Business Impact:** SLA breach, poor user experience  

#### Investigation Steps

1. **Performance Analysis**
   ```bash
   # Check current performance metrics
   curl -s "http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(nephoran_intent_processing_duration_seconds_bucket[5m]))by(le))"
   
   # Identify bottlenecks
   kubectl top pods -n nephoran-system --sort-by=cpu
   kubectl top pods -n nephoran-system --sort-by=memory
   ```

2. **Resource Optimization**
   ```bash
   # Scale up processing capacity
   kubectl scale deployment/llm-processor --replicas=8 -n nephoran-system
   kubectl scale deployment/rag-api --replicas=6 -n nephoran-system
   
   # Optimize resource limits
   kubectl patch deployment/nephoran-intent-controller -n nephoran-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"controller","resources":{"limits":{"cpu":"2000m","memory":"4Gi"}}}]}}}}'
   ```

### 3.2 Circuit Breaker Activation

**Alert:** `LLMCircuitBreakerOpen`  
**Symptoms:** Circuit breaker protecting LLM services  
**Business Impact:** Reduced functionality, fallback mode  

#### Recovery Actions

1. **Assess LLM Health**
   ```bash
   # Check LLM service status
   curl -f http://llm-processor:8080/health
   
   # Review error rates
   curl -s "http://prometheus:9090/api/v1/query?query=rate(llm_request_duration_seconds_count{status_code!~'2..'}[5m])"
   ```

2. **Reset Circuit Breaker**
   ```bash
   # Force circuit breaker reset (use carefully)
   curl -X POST http://llm-processor:8080/admin/circuit-breaker/reset
   
   # Gradually increase load
   kubectl patch deployment/llm-processor -n nephoran-system -p '{"spec":{"replicas":2}}'
   sleep 60
   kubectl patch deployment/llm-processor -n nephoran-system -p '{"spec":{"replicas":4}}'
   ```

---

## 4. O-RAN Interface Incident Playbooks

### 4.1 A1 Interface Policy Violation

**Alert:** `A1InterfacePolicyViolation`  
**Symptoms:** Policy enforcement failures, RIC communication issues  
**Business Impact:** Network optimization disabled  

#### Resolution Steps

1. **Policy Analysis**
   ```bash
   # Check current policies
   kubectl get a1policies -A
   kubectl describe a1policy/traffic-steering-policy -n nephoran-system
   
   # Validate policy syntax
   ./scripts/oran/validate-a1-policies.sh
   ```

2. **RIC Connectivity**
   ```bash
   # Test Near-RT RIC connection
   curl -f http://near-rt-ric:8080/a1-p/policies
   
   # Check A1 interface logs
   kubectl logs -l component=a1-interface -n nephoran-system --tail=200
   ```

### 4.2 E2 Interface Subscription Failure

**Alert:** `E2InterfaceSubscriptionFailure`  
**Symptoms:** Loss of real-time RAN data  
**Business Impact:** Reduced network intelligence  

#### Resolution Steps

1. **Subscription Recovery**
   ```bash
   # List current subscriptions
   kubectl get e2subscriptions -A
   
   # Recreate failed subscriptions
   kubectl delete e2subscription/kpm-subscription -n nephoran-system
   kubectl apply -f manifests/e2/kmp-subscription.yaml
   ```

2. **E2 Node Health**
   ```bash
   # Check E2 node connectivity
   kubectl get e2nodes -A
   kubectl describe e2node/ran-simulator-1 -n nephoran-system
   ```

---

## 5. NWDAF Analytics Incident Playbooks

### 5.1 Data Ingestion Failure

**Alert:** `NWDAFDataIngestionFailure`  
**Symptoms:** No new analytics data, stale predictions  
**Business Impact:** Degraded network optimization  

#### Resolution Steps

1. **Data Pipeline Check**
   ```bash
   # Check ingestion status
   kubectl logs -l app=nwdaf-collector -n nwdaf-system --tail=100
   
   # Verify data sources
   ./scripts/nwdaf/check-data-sources.sh
   ```

2. **Pipeline Recovery**
   ```bash
   # Restart ingestion pipeline
   kubectl rollout restart deployment/nwdaf-collector -n nwdaf-system
   
   # Clear stuck messages
   kubectl exec -it deployment/kafka -n nwdaf-system -- kafka-topics --bootstrap-server localhost:9092 --delete --topic ingestion-errors
   ```

### 5.2 ML Model Accuracy Degradation

**Alert:** `NWDAFMLModelAccuracyLow`  
**Symptoms:** Poor prediction accuracy, increased false positives  
**Business Impact:** Suboptimal network decisions  

#### Resolution Steps

1. **Model Evaluation**
   ```bash
   # Check model performance metrics
   curl -s "http://nwdaf-ml:8080/metrics/model-accuracy"
   
   # Review training data quality
   ./scripts/nwdaf/analyze-training-data.sh --days=7
   ```

2. **Model Retraining**
   ```bash
   # Trigger model retraining
   kubectl create job nwdaf-retrain-$(date +%s) --from=cronjob/nwdaf-model-training -n nwdaf-system
   
   # Monitor training progress
   kubectl logs -f job/nwdaf-retrain-$(date +%s) -n nwdaf-system
   ```

---

## 6. Escalation Procedures

### 6.1 Executive Escalation Matrix

| Time Window | Role | Contact Method |
|-------------|------|----------------|
| 0-30 minutes | On-Call Engineer | PagerDuty |
| 30-60 minutes | Senior Operations Manager | Phone + Slack |
| 1-2 hours | VP Engineering | Phone + Teams |
| 2+ hours | CTO | All channels |

### 6.2 Customer Communication

**P1 Incidents:**
- Initial notification: 15 minutes
- Hourly updates until resolved
- Post-incident report: 24 hours

**P2 Incidents:**
- Initial notification: 1 hour
- Updates every 4 hours
- Post-incident report: 48 hours

### 6.3 External Escalation

**Vendor Support:**
- OpenAI API issues: Submit ticket within 30 minutes
- Cloud provider outages: Activate support plan
- Kubernetes issues: Engage Red Hat/SUSE support

---

## 7. Post-Incident Procedures

### 7.1 Immediate Actions (Within 4 hours)

1. **Service Verification**
   ```bash
   # Full system health check
   ./scripts/health/comprehensive-check.sh
   
   # Performance baseline verification
   ./scripts/monitoring/verify-sla-metrics.sh
   ```

2. **Documentation**
   ```bash
   # Generate incident timeline
   ./scripts/incident/generate-timeline.sh --incident-id=INC-2024-001
   
   # Export metrics during incident
   ./scripts/monitoring/export-incident-metrics.sh --start="2024-08-08T10:00:00Z" --end="2024-08-08T12:00:00Z"
   ```

### 7.2 Post-Incident Review (Within 72 hours)

**Required Attendees:**
- Incident Commander
- Technical responders
- Engineering Manager
- Product Owner

**Review Template:**
```markdown
## Incident Post-Mortem: INC-2024-001

### Incident Summary
- Start Time: 2024-08-08 10:15 UTC
- End Time: 2024-08-08 11:45 UTC
- Duration: 1h 30m
- Severity: P1
- Impact: Complete service outage

### Root Cause
[Detailed root cause analysis]

### Timeline
[Detailed timeline of events]

### What Went Well
- Quick detection (5 minutes)
- Effective team coordination
- Rapid communication to customers

### What Could Be Improved
- Monitoring gaps identified
- Documentation updates needed
- Process refinements

### Action Items
1. [ ] Update monitoring alerts (Owner: SRE Team, Due: 2024-08-15)
2. [ ] Improve runbook documentation (Owner: Platform Team, Due: 2024-08-20)
3. [ ] Implement additional safeguards (Owner: Dev Team, Due: 2024-08-30)
```

### 7.3 Follow-up Actions

**Immediate (1-3 days):**
- Update monitoring alerts
- Patch critical vulnerabilities
- Improve documentation

**Short-term (1-2 weeks):**
- Implement preventive measures
- Update training materials
- Refine runbooks

**Long-term (1-3 months):**
- Architectural improvements
- Process optimization
- Tool enhancements

---

## 8. Emergency Contact Information

### Internal Contacts
- **Operations Manager:** +1-555-0101 (Primary)
- **Engineering On-Call:** +1-555-0102 (PagerDuty)
- **Security Team:** security-oncall@company.com
- **Executive Escalation:** exec-escalation@company.com

### External Contacts
- **Cloud Provider Support:** Enterprise Support Plan
- **OpenAI Support:** API Enterprise Support
- **Kubernetes Support:** Red Hat Premium Support

### Communication Channels
- **Slack War Room:** #nephoran-incidents
- **Teams Executive:** #nephoran-executive
- **Status Page:** https://status.nephoran.io
- **Customer Updates:** notify@nephoran.io

---

**Document Control:**
- **Next Review:** 2024-09-08
- **Owner:** Operations Team
- **Approved By:** VP Engineering
- **Distribution:** All Operations Staff, Engineering Leadership
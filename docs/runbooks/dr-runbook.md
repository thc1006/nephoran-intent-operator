# Disaster Recovery Runbook
## Nephoran Intent Operator - Production Operations Guide

**Document Version:** 1.0  
**Last Updated:** 2024-12-07  
**Document Owner:** DevOps Team  
**Review Schedule:** Quarterly  

### Executive Summary

This runbook provides comprehensive disaster recovery procedures for the Nephoran Intent Operator, ensuring business continuity with a Recovery Time Objective (RTO) of less than 5 minutes and Recovery Point Objective (RPO) of less than 30 minutes. The procedures are designed for production environments and have been validated through regular disaster recovery testing.

---

## Table of Contents

1. [Overview and Architecture](#overview-and-architecture)
2. [RTO/RPO Requirements](#rtorpo-requirements)
3. [Emergency Contacts](#emergency-contacts)
4. [Disaster Scenarios](#disaster-scenarios)
5. [Pre-Disaster Preparation](#pre-disaster-preparation)
6. [Disaster Detection](#disaster-detection)
7. [Failover Procedures](#failover-procedures)
8. [Recovery Validation](#recovery-validation)
9. [Post-Incident Activities](#post-incident-activities)
10. [Testing Procedures](#testing-procedures)
11. [Troubleshooting Guide](#troubleshooting-guide)
12. [Appendices](#appendices)

---

## Overview and Architecture

### System Architecture

The Nephoran Intent Operator disaster recovery solution consists of:

- **Primary Production Cluster**: Main operational cluster running Nephoran components
- **Secondary DR Cluster**: Pre-provisioned standby cluster ready for immediate activation
- **Backup Storage**: S3-compatible storage for backup data and configurations
- **Monitoring System**: Health monitoring with automatic failover triggers
- **DNS/Load Balancer**: Traffic routing capability for seamless failover

### Key Components

| Component | Primary Location | Secondary Location | Backup Frequency |
|-----------|------------------|-------------------|------------------|
| Nephoran Operator | Primary K8s Cluster | Secondary K8s Cluster | Hourly (config) / Daily (full) |
| LLM Processor | Primary K8s Cluster | Secondary K8s Cluster | Hourly (config) / Daily (full) |
| RAG API | Primary K8s Cluster | Secondary K8s Cluster | Hourly (config) / Daily (full) |
| Weaviate Database | Primary K8s Cluster | Secondary K8s Cluster | 6 hourly / Daily (full) |
| Vector Data | Persistent Volumes | Volume Snapshots | Daily |
| Configurations | ConfigMaps/Secrets | Velero Backups | Hourly |
| Network Intents | Custom Resources | Velero Backups | Hourly |

---

## RTO/RPO Requirements

### Service Level Objectives

| Metric | Target | Maximum Acceptable | Measurement Method |
|--------|--------|-------------------|-------------------|
| **RTO (Recovery Time Objective)** | < 5 minutes | 15 minutes | Time from disaster detection to service restoration |
| **RPO (Recovery Point Objective)** | < 30 minutes | 2 hours | Maximum acceptable data loss |
| **Service Availability** | 99.95% | 99.9% | Monthly uptime measurement |
| **Backup Success Rate** | 100% | 99% | Daily backup completion rate |

### Business Impact

- **Critical Impact**: RTO > 15 minutes, RPO > 2 hours
- **High Impact**: RTO 5-15 minutes, RPO 30 minutes - 2 hours  
- **Medium Impact**: RTO < 5 minutes, RPO < 30 minutes

---

## Emergency Contacts

### Primary Contacts

| Role | Name | Phone | Email | Availability |
|------|------|-------|-------|--------------|
| **Incident Commander** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 24/7 |
| **DevOps Lead** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 24/7 |
| **Platform Engineer** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | Business Hours |
| **Network Operations** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 24/7 |

### Escalation Chain

1. **Level 1**: On-call DevOps Engineer (0-15 minutes)
2. **Level 2**: DevOps Lead (15-30 minutes)
3. **Level 3**: Incident Commander (30-45 minutes)
4. **Level 4**: Engineering Director (45+ minutes)

### External Contacts

| Service | Contact | Phone | Email | SLA |
|---------|---------|-------|-------|-----|
| **Cloud Provider** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 1 hour |
| **DNS Provider** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 30 minutes |
| **Monitoring Vendor** | [TO BE FILLED] | [TO BE FILLED] | [TO BE FILLED] | 2 hours |

---

## Disaster Scenarios

### Scenario 1: Primary Cluster Complete Failure

**Symptoms:**
- Primary Kubernetes API unreachable
- All pod endpoints returning 5xx errors
- Node hardware failures
- Network partitioning

**Impact:** CRITICAL - Complete service outage
**Estimated RTO:** 3-5 minutes
**Recovery Procedure:** [Full Cluster Failover](#full-cluster-failover)

### Scenario 2: Partial Service Degradation

**Symptoms:**
- Some components responding, others failing
- Intermittent 5xx errors
- High latency responses
- Partial node failures

**Impact:** HIGH - Degraded service performance
**Estimated RTO:** 2-3 minutes  
**Recovery Procedure:** [Partial Service Recovery](#partial-service-recovery)

### Scenario 3: Data Corruption/Loss

**Symptoms:**
- Vector database corruption
- Persistent volume failures
- Network intent data inconsistencies
- Configuration drift

**Impact:** HIGH - Data integrity issues
**Estimated RTO:** 10-15 minutes
**Recovery Procedure:** [Data Recovery](#data-recovery-procedure)

### Scenario 4: Network/DNS Failures

**Symptoms:**
- DNS resolution failures
- Load balancer unreachable
- Certificate validation errors
- Cross-cluster communication failures

**Impact:** MEDIUM-HIGH - Service unreachable
**Estimated RTO:** 1-2 minutes
**Recovery Procedure:** [Network Recovery](#network-recovery-procedure)

---

## Pre-Disaster Preparation

### Daily Checklist

**Backup Verification (Automated)**
```bash
# Check backup completion status
kubectl get backups -n velero --sort-by=.metadata.creationTimestamp
kubectl get schedules -n velero -o wide

# Verify backup storage accessibility
velero backup get --output table
```

**Secondary Cluster Health Check**
```bash
# Switch to secondary cluster context
kubectl config use-context nephoran-dr-secondary

# Verify cluster health
kubectl get nodes
kubectl get pods --all-namespaces | grep -v Running
kubectl top nodes
```

**Monitoring System Status**
```bash
# Check Prometheus targets
kubectl port-forward -n monitoring svc/prometheus 9090:9090 &
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health != "up")'

# Verify alert manager
kubectl get pods -n monitoring -l app.kubernetes.io/name=alertmanager
```

### Weekly Checklist

**Disaster Recovery Test**
```bash
# Run DR test suite
./scripts/test-disaster-recovery.sh --quick
```

**Backup Storage Audit**
```bash
# Check backup retention and cleanup
aws s3 ls s3://nephoran-disaster-recovery-backups/production-cluster/ | head -20
aws s3 ls s3://nephoran-disaster-recovery-backups/production-cluster/ | tail -20
```

**Secondary Cluster Sync**
```bash
# Verify secondary cluster image currency
./scripts/setup-secondary-cluster.sh --validate-images
```

### Monthly Checklist

**Full DR Test**
```bash
# Complete disaster recovery simulation
./scripts/test-disaster-recovery.sh --full
```

**Runbook Review**
- Review and update contact information
- Validate procedures against current architecture
- Update RTO/RPO measurements
- Review lessons learned from incidents

---

## Disaster Detection

### Automated Monitoring

**Health Check Endpoints**
- Primary cluster API: `https://nephoran-api.example.com/health`
- LLM Processor: `https://nephoran-api.example.com/llm/health`
- RAG API: `https://nephoran-api.example.com/rag/health`
- Weaviate: `https://nephoran-api.example.com/vector/health`

**Alert Conditions**

| Alert | Threshold | Duration | Action |
|-------|-----------|----------|---------|
| ClusterDown | API unreachable | 2 minutes | Auto-failover trigger |
| PodCrashLoop | >3 restarts in 5min | 5 minutes | Investigation required |
| HighErrorRate | >5% error rate | 3 minutes | Prepare for failover |
| BackupFailure | Backup incomplete | 1 occurrence | Immediate investigation |
| StorageFull | >90% usage | 1 minute | Scale storage/cleanup |

**Monitoring Commands**
```bash
# Check cluster health
kubectl get nodes --no-headers | grep -v Ready

# Check critical pods
kubectl get pods -n nephoran-system --no-headers | grep -v Running

# Check recent events
kubectl get events --sort-by=.metadata.creationTimestamp -n nephoran-system | tail -20
```

### Manual Detection

**Performance Indicators**
- API response time > 5 seconds
- Intent processing time > 30 seconds
- Vector search latency > 2 seconds
- Backup duration > 2 hours

**User Reports**
- Network intent creation failures
- LLM processing timeouts
- RAG query failures
- UI/API unavailability

---

## Failover Procedures

### Immediate Actions (First 60 seconds)

1. **Assess Situation**
   ```bash
   # Quick health check
   ./scripts/failover-to-secondary.sh --check-primary
   ./scripts/failover-to-secondary.sh --check-secondary
   ```

2. **Notify Stakeholders**
   - Alert incident commander
   - Notify operations team
   - Update status page

3. **Preserve Evidence**
   ```bash
   # Capture logs before failover
   kubectl logs -n nephoran-system --all-containers=true --tail=1000 > /tmp/primary-logs-$(date +%Y%m%d-%H%M%S).log
   kubectl get events --all-namespaces --sort-by=.metadata.creationTimestamp > /tmp/primary-events-$(date +%Y%m%d-%H%M%S).log
   ```

### Full Cluster Failover

**Step 1: Initiate Failover (0-2 minutes)**
```bash
# Execute automatic failover
./scripts/failover-to-secondary.sh --execute

# Monitor failover progress
tail -f /tmp/failover-$(date +%Y%m%d)*.log
```

**Step 2: Verify Secondary Cluster (2-3 minutes)**
```bash
# Switch to secondary cluster
kubectl config use-context nephoran-dr-secondary

# Check cluster status
kubectl get nodes
kubectl get pods -n nephoran-system -o wide
```

**Step 3: Update DNS/Load Balancer (3-4 minutes)**
```bash
# DNS updates (if not automated)
# Update A records to point to secondary cluster
# Example for Cloudflare:
curl -X PUT "https://api.cloudflare.com/client/v4/zones/ZONE_ID/dns_records/RECORD_ID" \
  -H "Authorization: Bearer $CLOUDFLARE_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"type":"A","name":"nephoran-api.example.com","content":"SECONDARY_CLUSTER_IP"}'
```

**Step 4: Validate Services (4-5 minutes)**
```bash
# Test API endpoints
curl -s https://nephoran-api.example.com/health
curl -s https://nephoran-api.example.com/llm/health
curl -s https://nephoran-api.example.com/rag/health

# Test network intent creation
kubectl apply -f - << EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: dr-test-intent
  namespace: nephoran-system
spec:
  intent: "Deploy test network function for DR validation"
EOF
```

### Partial Service Recovery

**For Individual Component Failures:**

1. **Identify Failed Component**
   ```bash
   kubectl get pods -n nephoran-system | grep -v Running
   kubectl describe pod FAILING_POD -n nephoran-system
   ```

2. **Attempt Restart**
   ```bash
   kubectl rollout restart deployment/COMPONENT_NAME -n nephoran-system
   kubectl rollout status deployment/COMPONENT_NAME -n nephoran-system
   ```

3. **Restore from Backup if Needed**
   ```bash
   # Create targeted restore
   velero create restore component-restore-$(date +%Y%m%d-%H%M%S) \
     --from-backup LATEST_BACKUP_NAME \
     --include-resources deployments,configmaps,secrets \
     --selector app.kubernetes.io/name=COMPONENT_NAME
   ```

### Data Recovery Procedure

**Vector Database Recovery:**
```bash
# Stop Weaviate pods
kubectl scale deployment/weaviate --replicas=0 -n nephoran-system

# Restore from latest backup
velero create restore weaviate-restore-$(date +%Y%m%d-%H%M%S) \
  --from-backup LATEST_BACKUP_NAME \
  --include-resources persistentvolumeclaims \
  --selector app.kubernetes.io/name=weaviate

# Wait for PVC restore
kubectl wait --for=condition=Bound pvc/weaviate-data -n nephoran-system --timeout=300s

# Restart Weaviate
kubectl scale deployment/weaviate --replicas=3 -n nephoran-system
kubectl rollout status deployment/weaviate -n nephoran-system
```

**Configuration Recovery:**
```bash
# Restore configurations
velero create restore config-restore-$(date +%Y%m%d-%H%M%S) \
  --from-backup LATEST_BACKUP_NAME \
  --include-resources configmaps,secrets \
  --include-namespaces nephoran-system,nephoran-production

# Restart affected components
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout restart deployment/rag-api -n nephoran-system
```

### Network Recovery Procedure

**DNS Failover:**
```bash
# Manual DNS update (if automation fails)
# Update DNS records to point to secondary cluster
# Verify DNS propagation
dig nephoran-api.example.com +short
nslookup nephoran-api.example.com 8.8.8.8
```

**Certificate Issues:**
```bash
# Check certificate status
kubectl get certificates -n nephoran-system
kubectl describe certificate nephoran-tls -n nephoran-system

# Force certificate renewal
kubectl delete certificate nephoran-tls -n nephoran-system
kubectl apply -f deployments/cert-manager/certificates.yaml
```

---

## Recovery Validation

### Automated Validation

**Health Check Script:**
```bash
#!/bin/bash
# validate-recovery.sh

echo "Validating disaster recovery..."

# API Health Checks
if curl -s -f https://nephoran-api.example.com/health > /dev/null; then
    echo "✓ Main API healthy"
else
    echo "✗ Main API unhealthy"
    exit 1
fi

# Component Health Checks
components=("llm" "rag" "vector")
for component in "${components[@]}"; do
    if curl -s -f https://nephoran-api.example.com/$component/health > /dev/null; then
        echo "✓ $component API healthy"
    else
        echo "✗ $component API unhealthy"
        exit 1
    fi
done

# Kubernetes Health
if kubectl get nodes | grep -q "Ready"; then
    echo "✓ Kubernetes cluster healthy"
else
    echo "✗ Kubernetes cluster unhealthy"
    exit 1
fi

# Pod Health
failing_pods=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
if [ "$failing_pods" -eq 0 ]; then
    echo "✓ All pods running"
else
    echo "✗ $failing_pods pods not running"
    exit 1
fi

echo "Recovery validation successful!"
```

### Manual Validation

**Functional Testing:**

1. **Create Test Network Intent**
   ```bash
   kubectl apply -f - << EOF
   apiVersion: nephoran.com/v1
   kind: NetworkIntent
   metadata:
     name: dr-validation-intent
     namespace: nephoran-system
   spec:
     intent: "Deploy test AMF with high availability for DR validation"
   EOF
   ```

2. **Monitor Intent Processing**
   ```bash
   kubectl get networkintent dr-validation-intent -n nephoran-system -w
   kubectl logs -f deployment/nephoran-operator -n nephoran-system
   ```

3. **Verify RAG Functionality**
   ```bash
   curl -X POST https://nephoran-api.example.com/rag/query \
     -H "Content-Type: application/json" \
     -d '{"query": "What are the requirements for AMF deployment?"}'
   ```

4. **Test LLM Processing**
   ```bash
   curl -X POST https://nephoran-api.example.com/llm/process \
     -H "Content-Type: application/json" \
     -d '{"intent": "Deploy high availability UPF with auto-scaling"}'
   ```

### Performance Validation

**Response Time Testing:**
```bash
# API response times
time curl -s https://nephoran-api.example.com/health
time curl -s https://nephoran-api.example.com/llm/health
time curl -s https://nephoran-api.example.com/rag/health

# Should be < 2 seconds each
```

**Throughput Testing:**
```bash
# Concurrent request testing
for i in {1..10}; do
    curl -s https://nephoran-api.example.com/health &
done
wait

# All requests should complete successfully
```

---

## Post-Incident Activities

### Immediate Post-Recovery (Within 1 hour)

1. **Status Communication**
   - Update status page to "Operational"
   - Notify stakeholders of recovery
   - Send all-clear communications

2. **System Stabilization**
   ```bash
   # Monitor for 30 minutes post-recovery
   watch kubectl get pods -n nephoran-system
   
   # Check error rates
   kubectl logs deployment/nephoran-operator -n nephoran-system --tail=100
   ```

3. **Initial Incident Report**
   - Document timeline of events
   - Capture key metrics (RTO/RPO achieved)
   - Note any deviations from procedures

### Short-term Activities (Within 24 hours)

1. **Primary Cluster Assessment**
   - Determine root cause of failure
   - Assess recovery feasibility
   - Plan primary cluster restoration

2. **Backup Validation**
   ```bash
   # Verify backup integrity
   velero backup describe BACKUP_USED_FOR_RECOVERY
   
   # Check for any backup gaps
   velero backup get --output table
   ```

3. **Performance Monitoring**
   - Monitor system performance on secondary cluster
   - Check for any degraded functionality
   - Validate all automated processes

### Long-term Activities (Within 1 week)

1. **Incident Post-Mortem**
   - Conduct blameless post-mortem meeting
   - Document lessons learned
   - Identify improvement actions

2. **Primary Cluster Recovery**
   ```bash
   # Plan primary cluster restoration
   # This may involve:
   # - Hardware replacement
   # - Infrastructure rebuilding
   # - Data synchronization
   # - Testing before cutback
   ```

3. **Process Improvements**
   - Update runbook based on lessons learned
   - Improve monitoring/alerting
   - Enhance automation
   - Schedule additional training

### Documentation Updates

**Incident Report Template:**
```markdown
# Incident Report: [Date] - [Brief Description]

## Summary
- **Start Time:** [Timestamp]
- **End Time:** [Timestamp] 
- **Duration:** [Minutes]
- **RTO Achieved:** [Minutes]
- **RPO Achieved:** [Minutes]

## Timeline
- [Timestamp]: [Event description]
- [Timestamp]: [Action taken]
- [Timestamp]: [Result]

## Root Cause
[Detailed analysis of what caused the incident]

## Actions Taken
[Step-by-step recovery actions]

## Lessons Learned
[What went well, what could be improved]

## Action Items
- [ ] [Improvement action 1]
- [ ] [Improvement action 2]
- [ ] [Process update]
```

---

## Testing Procedures

### Monthly DR Tests

**Complete Test Procedure:**
```bash
# 1. Schedule test (communicate to stakeholders)
echo "Starting monthly DR test at $(date)"

# 2. Run comprehensive test
./scripts/test-disaster-recovery.sh --full

# 3. Validate results
if [ $? -eq 0 ]; then
    echo "DR test passed"
else
    echo "DR test failed - investigate issues"
fi

# 4. Document results
# Generate test report and file with monthly reports
```

**Test Scenarios to Cover:**
1. Primary cluster complete failure
2. Individual component failures
3. Database corruption scenarios
4. Network connectivity issues
5. DNS/Load balancer failures
6. Backup restoration validation

### Quarterly DR Exercises

**Tabletop Exercises:**
- Simulate different disaster scenarios
- Walk through procedures with team
- Identify gaps in knowledge/procedures
- Update contact information

**Full Failover Tests:**
- Actual failover to secondary cluster
- Run production traffic simulation
- Measure RTO/RPO compliance
- Test rollback procedures

### Annual DR Audit

**Comprehensive Review:**
- Architecture validation
- Procedure accuracy verification
- Team competency assessment
- Technology currency review
- Compliance validation

---

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue: Backup Fails to Complete

**Symptoms:**
- Velero backup stuck in "InProgress" state
- Backup timeout errors
- S3 connectivity issues

**Diagnosis:**
```bash
# Check backup status
kubectl describe backup BACKUP_NAME -n velero

# Check Velero logs  
kubectl logs deployment/velero -n velero --tail=100

# Test S3 connectivity
kubectl exec deployment/velero -n velero -- aws s3 ls s3://nephoran-disaster-recovery-backups/
```

**Resolution:**
```bash
# Delete stuck backup
kubectl delete backup BACKUP_NAME -n velero

# Check storage credentials
kubectl get secret cloud-credentials -n velero -o yaml

# Restart Velero if needed
kubectl rollout restart deployment/velero -n velero
```

#### Issue: Restore Fails with PVC Errors

**Symptoms:**
- Restore stuck waiting for PVCs
- PVC in "Pending" state
- Volume mount failures

**Diagnosis:**
```bash
# Check PVC status
kubectl get pvc -n TARGET_NAMESPACE
kubectl describe pvc FAILING_PVC -n TARGET_NAMESPACE

# Check storage class
kubectl get storageclass
kubectl describe storageclass default
```

**Resolution:**
```bash
# Check available storage
kubectl top nodes

# If using dynamic provisioning, ensure storage class exists
kubectl get storageclass

# For static PVs, check PV availability
kubectl get pv
```

#### Issue: Secondary Cluster Not Ready

**Symptoms:**
- Secondary cluster health checks fail
- Missing essential components
- Image pull failures

**Diagnosis:**
```bash
# Switch to secondary cluster
kubectl config use-context nephoran-dr-secondary

# Check node status
kubectl get nodes -o wide

# Check pod status
kubectl get pods --all-namespaces | grep -v Running

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp | tail -20
```

**Resolution:**
```bash
# Re-run secondary cluster setup
./scripts/setup-secondary-cluster.sh --force

# Pull required images
kubectl get pods -n nephoran-system -o jsonpath='{range .items[*]}{.spec.containers[*].image}{"\n"}{end}' | sort | uniq | xargs -I {} docker pull {}
```

#### Issue: DNS/Load Balancer Failover Issues

**Symptoms:**
- DNS not resolving to secondary cluster
- SSL certificate errors
- Load balancer health check failures

**Diagnosis:**
```bash
# Check DNS resolution
dig nephoran-api.example.com +short
nslookup nephoran-api.example.com 8.8.8.8

# Check certificate status
kubectl get certificates -n nephoran-system
openssl s_client -connect nephoran-api.example.com:443 -servername nephoran-api.example.com
```

**Resolution:**
```bash
# Manual DNS update
# (Update DNS records to point to secondary cluster)

# Force certificate renewal
kubectl delete certificate nephoran-tls -n nephoran-system
kubectl apply -f deployments/cert-manager/certificates.yaml

# Restart ingress controller
kubectl rollout restart deployment/ingress-nginx-controller -n ingress-nginx
```

### Escalation Procedures

**Level 1: Automated Resolution**
- Automated health checks and restart procedures
- Self-healing mechanisms
- Circuit breaker activation

**Level 2: DevOps Engineer Response**
- Manual troubleshooting using runbook
- Log analysis and diagnostics
- Initial recovery attempts

**Level 3: DevOps Lead Involvement**
- Complex issue resolution
- Cross-team coordination
- Decision on manual failover

**Level 4: Incident Commander Escalation**
- Major incident management
- Executive communication
- External vendor coordination

---

## Appendices

### Appendix A: Command Reference

**Essential Commands:**
```bash
# Cluster health check
kubectl cluster-info
kubectl get nodes
kubectl get pods --all-namespaces | grep -v Running

# Backup operations
velero backup get
velero backup describe BACKUP_NAME
velero backup create emergency-backup --include-namespaces nephoran-system

# Restore operations
velero restore get
velero restore describe RESTORE_NAME
velero restore create emergency-restore --from-backup BACKUP_NAME

# Failover operations
./scripts/failover-to-secondary.sh --check-primary
./scripts/failover-to-secondary.sh --check-secondary
./scripts/failover-to-secondary.sh --execute

# Testing
./scripts/test-disaster-recovery.sh --quick
./scripts/test-disaster-recovery.sh --full
```

### Appendix B: Configuration Files

**Backup Schedule Configuration:**
- Location: `deployments/velero/backup-schedule.yaml`
- Contains: Daily, weekly, and critical backup schedules
- Review frequency: Monthly

**Secondary Cluster Configuration:**
- Location: `scripts/setup-secondary-cluster.sh`
- Contains: K3d cluster configuration and setup procedures
- Review frequency: Quarterly

**Monitoring Configuration:**
- Location: `deployments/monitoring/`
- Contains: Prometheus alerts and Grafana dashboards
- Review frequency: Monthly

### Appendix C: Network Diagrams

**Primary Architecture:**
```
[Users] -> [Load Balancer] -> [Primary K8s Cluster]
                                    |
                              [Backup Storage]
                                    |
                            [Secondary K8s Cluster]
```

**Failover Architecture:**
```
[Users] -> [Load Balancer] -> [Secondary K8s Cluster]
                                    |
                              [Backup Storage]
                                    |
                            [Primary K8s Cluster] (Failed)
```

### Appendix D: Change Log

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024-12-07 | DevOps Team | Initial version |

### Appendix E: Related Documentation

- [Deployment Guide](./operations/01-production-deployment-guide.md)
- [Monitoring Guide](./operations/02-monitoring-alerting-runbooks.md)
- [Security Guidelines](./SECURITY.md)
- [Performance Tuning](./operations/PERFORMANCE-TUNING.md)

---

## Document Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| **Author** | DevOps Team | [TO BE SIGNED] | [DATE] |
| **Reviewer** | Platform Lead | [TO BE SIGNED] | [DATE] |
| **Approver** | Engineering Director | [TO BE SIGNED] | [DATE] |

---

**End of Document**

*This runbook is a living document and should be updated regularly based on system changes, incident learnings, and test results. For questions or suggestions, please contact the DevOps team.*
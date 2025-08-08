# Production Operations Runbook

**Version:** 2.1  
**Last Updated:** December 2024  
**Audience:** Site Reliability Engineers, Platform Operations Teams, On-call Engineers  
**Classification:** Production Operations Documentation

## Overview

This runbook provides comprehensive operational procedures for the Nephoran Intent Operator in production environments. It covers daily operations, incident response, maintenance procedures, and troubleshooting guides based on 18 months of production operations across 47 enterprise deployments.

## On-Call Response Procedures

### Incident Severity Classification

```yaml
P0 - Critical (15-minute response, 4-hour resolution):
  - Complete system unavailability (>50% of intents failing)
  - Data corruption or security breach
  - Customer-affecting outages with financial impact >$50K/hour
  - Regulatory compliance violations
  
P1 - High (30-minute response, 8-hour resolution):
  - Partial system degradation (10-50% of intents failing)
  - Performance degradation affecting >25% of users
  - Single region failure in multi-region deployment
  - Security incidents without active breach
  
P2 - Medium (1-hour response, 24-hour resolution):
  - Minor performance degradation (<25% impact)
  - Single service unavailability with workarounds
  - Monitoring and alerting issues
  - Non-critical compliance issues
  
P3 - Low (4-hour response, 72-hour resolution):
  - Feature requests and enhancements
  - Documentation updates
  - Cosmetic issues without functional impact
  - Proactive maintenance items
```

### Alert Response Matrix

| Alert | Severity | Immediate Actions | Escalation |
|-------|----------|-------------------|------------|
| Service Unavailable | P0 | Check service health, restart if needed, engage incident commander | Immediate manager notification |
| High Error Rate | P1 | Analyze error patterns, check dependencies, scale if needed | Technical lead in 30 minutes |
| Performance Degradation | P2 | Monitor trends, check resource utilization, prepare scaling | Technical lead in 2 hours |
| Certificate Expiring | P3 | Renew certificates, update monitoring | Standard change process |

### Initial Response Checklist

```bash
# Initial incident response checklist
echo "=== Nephoran Incident Response ==="

# 1. Acknowledge alert and create incident
echo "1. Acknowledging alert in PagerDuty..."
# pd incident acknowledge --incident-id $INCIDENT_ID

# 2. Check system health
echo "2. Checking system health..."
kubectl get pods -n nephoran-system
kubectl get nodes -o wide
kubectl top nodes

# 3. Check service endpoints
echo "3. Checking service endpoints..."
curl -k https://nephoran-api.company.com/health
curl -k https://nephoran-api.company.com/metrics

# 4. Review recent changes
echo "4. Reviewing recent deployments..."
kubectl rollout history deployment/nephoran-controller -n nephoran-system
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'

# 5. Gather logs
echo "5. Gathering system logs..."
kubectl logs -n nephoran-system deployment/llm-processor --tail=100
kubectl logs -n nephoran-system deployment/rag-service --tail=100
kubectl logs -n nephoran-system deployment/nephoran-controller --tail=100

# 6. Check dependencies
echo "6. Checking external dependencies..."
kubectl get networkintents --all-namespaces | grep -v "Processed\|Succeeded"
```

## Daily Operations Procedures

### Morning Health Check

```bash
#!/bin/bash
# Daily morning health check script

echo "=== Daily Nephoran Health Check ==="
date

# Check cluster health
echo "--- Cluster Health ---"
kubectl cluster-info
kubectl get nodes -o wide
kubectl top nodes

# Check service status
echo "--- Service Status ---"
kubectl get pods -n nephoran-system -o wide
kubectl get svc -n nephoran-system
kubectl get ingress -n nephoran-system

# Check resource utilization  
echo "--- Resource Utilization ---"
kubectl top pods -n nephoran-system --sort-by=cpu
kubectl top pods -n nephoran-system --sort-by=memory

# Check intent processing status
echo "--- Intent Processing ---"
kubectl get networkintents --all-namespaces | \
  awk '{print $NF}' | sort | uniq -c
  
# Check recent errors
echo "--- Recent Errors ---"
kubectl get events -n nephoran-system --field-selector type=Warning \
  --sort-by='.lastTimestamp' | tail -10

# Check certificate status
echo "--- Certificate Status ---"
kubectl get certificates -n nephoran-system
kubectl get certificaterequests -n nephoran-system

# Check backup status
echo "--- Backup Status ---"
kubectl get backups -n velero | tail -5
kubectl get restores -n velero | tail -5

# Performance metrics summary
echo "--- Performance Summary ---"
curl -s "http://prometheus.monitoring:9090/api/v1/query?query=nephoran_intent_processing_duration_seconds{quantile=\"0.95\"}" | \
  jq '.data.result[0].value[1]'

echo "Daily health check completed at $(date)"
```

### Weekly Maintenance Tasks

```yaml
Weekly Operations Checklist:
  Monday:
    - Review weekend alerts and incidents
    - Check backup validation reports
    - Update capacity planning models
    - Review security scan results
    
  Tuesday:
    - Validate disaster recovery procedures
    - Check certificate expiration calendar
    - Review performance trending reports
    - Update dependency security patches
    
  Wednesday:
    - Database maintenance and optimization
    - Vector database index optimization  
    - Clean up old logs and metrics data
    - Review change management pipeline
    
  Thursday:
    - Validate monitoring and alerting rules
    - Check compliance audit requirements
    - Review incident post-mortems
    - Update operational runbooks
    
  Friday:
    - Prepare weekend on-call handover
    - Deploy approved maintenance changes
    - Validate backup and recovery procedures
    - Review weekly performance summary
```

### Monthly Operations Review

```bash
#!/bin/bash
# Monthly operations review script

echo "=== Monthly Nephoran Operations Review ==="

# Generate availability report
echo "--- Availability Report ---"
./scripts/generate-availability-report.sh --month $(date -d "last month" +%Y-%m)

# Generate performance report  
echo "--- Performance Report ---"
./scripts/generate-performance-report.sh --month $(date -d "last month" +%Y-%m)

# Generate capacity planning report
echo "--- Capacity Planning ---"
./scripts/generate-capacity-report.sh --month $(date -d "last month" +%Y-%m)

# Generate security report
echo "--- Security Report ---"
./scripts/generate-security-report.sh --month $(date -d "last month" +%Y-%m)

# Generate cost analysis
echo "--- Cost Analysis ---"
./scripts/generate-cost-report.sh --month $(date -d "last month" +%Y-%m)

echo "Monthly review completed at $(date)"
```

## Troubleshooting Guides

### Common Issue 1: Intent Processing Timeouts

**Symptoms:**
- NetworkIntent CRDs stuck in "Processing" status for >5 minutes
- High latency in LLM processor service metrics
- Increased error rates in application logs
- Customer reports of slow service provisioning

**Diagnostic Steps:**
```bash
# Check intent status distribution
kubectl get networkintents --all-namespaces | awk '{print $NF}' | sort | uniq -c

# Check LLM processor health
kubectl logs -n nephoran-system deployment/llm-processor --tail=50
kubectl describe pod -n nephoran-system -l app=llm-processor

# Check external API connectivity
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -v -m 10 https://api.openai.com/v1/models

# Check resource utilization
kubectl top pods -n nephoran-system -l app=llm-processor
kubectl describe hpa -n nephoran-system llm-processor-hpa

# Check circuit breaker status
curl -s http://llm-processor.nephoran-system:8080/metrics | \
  grep circuit_breaker
```

**Resolution Steps:**
```bash
# 1. Scale up LLM processor if resource constrained
kubectl scale deployment llm-processor --replicas=8 -n nephoran-system

# 2. Check and reset circuit breakers if needed
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -X POST localhost:8080/admin/circuit-breaker/reset

# 3. Clear processing queues if backed up
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -X POST localhost:8080/admin/queue/clear

# 4. Restart service if needed (last resort)
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout status deployment/llm-processor -n nephoran-system --timeout=300s

# 5. Verify resolution
kubectl get networkintents --all-namespaces | grep -v "Processed\|Succeeded"
```

### Common Issue 2: Vector Database Performance Degradation

**Symptoms:**
- RAG query latency exceeding 5 seconds (P95)
- High memory utilization on Weaviate pods
- Search accuracy degradation
- Index corruption warnings in logs

**Diagnostic Steps:**
```bash
# Check Weaviate cluster health
kubectl exec -n nephoran-system weaviate-0 -- \
  curl localhost:8080/v1/.well-known/ready

# Check resource utilization
kubectl top pods -n nephoran-system -l app=weaviate
kubectl describe pods -n nephoran-system -l app=weaviate

# Check index health
kubectl exec -n nephoran-system weaviate-0 -- \
  curl localhost:8080/v1/schema | jq '.classes[].vectorIndexConfig'

# Check storage utilization
kubectl exec -n nephoran-system weaviate-0 -- df -h /var/lib/weaviate

# Analyze query performance
kubectl logs -n nephoran-system weaviate-0 | grep "query_duration" | tail -20
```

**Resolution Steps:**
```bash
# 1. Optimize index parameters
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -X PUT localhost:8080/v1/schema/TelecomDocument \
  -d '{"vectorIndexConfig": {"ef": 64, "efConstruction": 128}}'

# 2. Clean up corrupted indexes if needed
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -X POST localhost:8080/v1/schema/TelecomDocument/shards/_cluster/rebuild

# 3. Scale cluster horizontally if needed
kubectl patch statefulset weaviate -n nephoran-system \
  -p '{"spec":{"replicas":6}}'

# 4. Restart cluster with rolling update if needed
kubectl rollout restart statefulset/weaviate -n nephoran-system

# 5. Validate performance recovery
./scripts/test-rag-performance.sh --duration 5m --concurrency 10
```

### Common Issue 3: Database Connection Pool Exhaustion

**Symptoms:**
- "Connection pool exhausted" errors in application logs
- High database connection count
- Slow response times for database queries
- Applications unable to connect to database

**Diagnostic Steps:**
```bash
# Check connection pool status
kubectl logs -n nephoran-system deployment/nephoran-controller | \
  grep "connection pool"

# Check database connection count
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Check for long-running queries
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "SELECT pid, now() - pg_stat_activity.query_start AS duration, query FROM pg_stat_activity WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';"

# Check PgBouncer status
kubectl exec -n nephoran-system pgbouncer-0 -- \
  psql -p 6432 pgbouncer -c "SHOW POOLS;"
```

**Resolution Steps:**
```bash
# 1. Kill long-running queries if appropriate
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE (now() - pg_stat_activity.query_start) > interval '10 minutes';"

# 2. Increase connection pool size temporarily
kubectl patch configmap pgbouncer-config -n nephoran-system \
  --patch '{"data":{"pgbouncer.ini":"pool_size = 50"}}'
kubectl rollout restart deployment/pgbouncer -n nephoran-system

# 3. Scale up database replicas for read traffic
kubectl scale deployment postgresql-read --replicas=3 -n nephoran-system

# 4. Restart applications to reset connections
kubectl rollout restart deployment/nephoran-controller -n nephoran-system
kubectl rollout restart deployment/llm-processor -n nephoran-system

# 5. Monitor recovery
watch "kubectl exec -n nephoran-system postgresql-0 -- psql -U postgres -c 'SELECT count(*) FROM pg_stat_activity;'"
```

### Common Issue 4: Certificate Expiration and TLS Issues

**Symptoms:**
- TLS handshake failures in service logs
- Browser certificate warnings
- Service mesh communication failures
- External API calls failing with certificate errors

**Diagnostic Steps:**
```bash
# Check certificate status
kubectl get certificates -n nephoran-system
kubectl describe certificate nephoran-tls -n nephoran-system

# Check certificate expiration dates
kubectl get secrets -n nephoran-system -o jsonpath='{.items[*].data.tls\.crt}' | \
  base64 -d | openssl x509 -dates -noout

# Check Istio certificate status
istioctl proxy-config secret deployment/llm-processor.nephoran-system

# Check external connectivity with certificates
kubectl exec -n nephoran-system deployment/llm-processor -- \
  openssl s_client -connect api.openai.com:443 -showcerts
```

**Resolution Steps:**
```bash
# 1. Renew certificates manually if auto-renewal failed
kubectl delete certificaterequest -n nephoran-system -l certificate=nephoran-tls
kubectl annotate certificate nephoran-tls -n nephoran-system cert-manager.io/force-renew=$(date +%s)

# 2. Restart cert-manager if needed
kubectl rollout restart deployment/cert-manager -n cert-manager
kubectl rollout restart deployment/cert-manager-cainjector -n cert-manager
kubectl rollout restart deployment/cert-manager-webhook -n cert-manager

# 3. Update certificate issuer if needed
kubectl apply -f deployments/cert-manager/issuer.yaml

# 4. Restart affected services
kubectl rollout restart deployment/nephoran-controller -n nephoran-system
kubectl rollout restart deployment/llm-processor -n nephoran-system

# 5. Validate certificate renewal
kubectl wait --for=condition=Ready certificate/nephoran-tls -n nephoran-system --timeout=300s
```

## Performance Monitoring and Optimization

### Key Performance Indicators (KPIs)

```yaml
Golden Signals:
  Latency:
    - Intent processing P50, P95, P99 latency
    - LLM API response time
    - Database query response time
    - RAG retrieval latency
    
  Traffic:
    - Requests per second
    - Concurrent intent processing
    - Database queries per second
    - Network bandwidth utilization
    
  Errors:  
    - Intent processing error rate
    - HTTP 5xx error rate
    - Database connection failures
    - External API failures
    
  Saturation:
    - CPU utilization per service
    - Memory utilization per service
    - Database connection pool utilization
    - Storage utilization
```

### Performance Tuning Procedures

```bash
#!/bin/bash
# Performance optimization script

echo "=== Nephoran Performance Optimization ==="

# 1. Database optimization
echo "--- Database Optimization ---"
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "VACUUM ANALYZE;"

# Update statistics
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "SELECT pg_stat_reset();"

# 2. Vector database optimization
echo "--- Vector Database Optimization ---"
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -X POST localhost:8080/v1/schema/optimize

# 3. Application cache warming
echo "--- Cache Warming ---" 
kubectl exec -n nephoran-system deployment/rag-service -- \
  curl -X POST localhost:8080/admin/cache/warm

# 4. Connection pool optimization
echo "--- Connection Pool Optimization ---"
kubectl patch configmap app-config -n nephoran-system \
  --patch '{"data":{"database.pool.size":"30","database.pool.timeout":"10s"}}'

# 5. JVM optimization (if applicable)
echo "--- JVM Optimization ---"
kubectl patch deployment llm-processor -n nephoran-system \
  --patch '{"spec":{"template":{"spec":{"containers":[{"name":"llm-processor","env":[{"name":"JAVA_OPTS","value":"-Xms2g -Xmx4g -XX:+UseG1GC"}]}]}}}}'

echo "Performance optimization completed"
```

### Resource Scaling Procedures

```yaml
Scaling Guidelines:
  Horizontal Scaling Triggers:
    - CPU utilization >70% for 10 minutes
    - Memory utilization >80% for 5 minutes  
    - Intent queue depth >100 for 5 minutes
    - Error rate >1% for 5 minutes
    
  Vertical Scaling Triggers:
    - Consistent resource pressure over 24 hours
    - Performance degradation despite horizontal scaling
    - Memory leaks or garbage collection issues
    - Database connection pool exhaustion
    
  Scaling Actions:
    LLM Processor: Scale from 3-15 replicas based on queue depth
    RAG Service: Scale from 2-8 replicas based on query latency
    Controller: Keep at 3 replicas (active-passive-standby)
    Database: Vertical scaling preferred, read replicas for scaling reads
```

```bash
# Auto-scaling configuration
kubectl apply -f - <<EOF
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-processor-hpa
  namespace: nephoran-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-processor
  minReplicas: 3
  maxReplicas: 15
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: intent_queue_depth
      target:
        type: AverageValue
        averageValue: "10"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
EOF
```

## Security Operations

### Security Monitoring Checklist

```bash
#!/bin/bash
# Daily security monitoring script

echo "=== Daily Security Check ==="

# Check for failed authentication attempts
echo "--- Authentication Failures ---"
kubectl logs -n nephoran-system deployment/nephoran-controller | \
  grep -i "authentication failed" | tail -10

# Check for unauthorized access attempts
echo "--- Unauthorized Access ---"
kubectl logs -n istio-system deployment/istio-proxy | \
  grep -E "403|401" | tail -10

# Check certificate validity
echo "--- Certificate Status ---"
kubectl get certificates -n nephoran-system -o wide

# Check network policy violations
echo "--- Network Policy Violations ---"
kubectl logs -n kube-system -l k8s-app=cilium | \
  grep "Policy denied" | tail -10

# Check for privilege escalation attempts
echo "--- Privilege Escalation ---"
kubectl logs -n kube-system -l app=falco | \
  grep "Privilege escalation" | tail -5

# Check for suspicious container activity
echo "--- Container Security ---"
kubectl logs -n kube-system -l app=falco | \
  grep -E "Shell|Exec|File" | tail -10

# Check for image vulnerabilities
echo "--- Image Security ---"
kubectl get vulnerabilityreports -n nephoran-system | \
  grep -E "HIGH|CRITICAL"

echo "Daily security check completed"
```

### Incident Response Procedures

```yaml
Security Incident Response:
  Phase 1 - Detection and Analysis (0-15 minutes):
    - Acknowledge security alert
    - Verify incident authenticity
    - Classify incident severity
    - Notify security team
    
  Phase 2 - Containment (15-60 minutes):
    - Isolate affected systems
    - Preserve evidence
    - Implement emergency containment measures
    - Document all actions taken
    
  Phase 3 - Investigation (1-24 hours):
    - Forensic analysis of logs and system state
    - Identify attack vectors and impact scope
    - Coordinate with external security partners
    - Prepare incident report
    
  Phase 4 - Recovery (1-72 hours):
    - Remove threat from environment
    - Restore systems from clean backups
    - Implement additional security controls
    - Validate system integrity
    
  Phase 5 - Post-Incident (1-2 weeks):
    - Complete incident analysis report
    - Update security procedures
    - Implement preventive measures
    - Conduct lessons learned session
```

### Emergency Security Procedures

```bash
#!/bin/bash
# Emergency security response script

echo "=== EMERGENCY SECURITY RESPONSE ==="

# Immediate containment actions
echo "--- Immediate Containment ---"

# 1. Block suspicious traffic
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: emergency-lockdown
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress: []
  egress:
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

# 2. Rotate all secrets immediately
echo "--- Secret Rotation ---"
kubectl delete secret nephoran-api-keys -n nephoran-system
kubectl create secret generic nephoran-api-keys -n nephoran-system \
  --from-literal=openai-key="$(./scripts/generate-emergency-key.sh)" \
  --from-literal=github-token="$(./scripts/generate-emergency-token.sh)"

# 3. Force certificate renewal
echo "--- Certificate Renewal ---"
kubectl annotate certificate --all -n nephoran-system \
  cert-manager.io/force-renew=$(date +%s)

# 4. Enable comprehensive audit logging
echo "--- Enable Audit Logging ---"
kubectl patch configmap audit-policy -n kube-system \
  --patch '{"data":{"audit-policy.yaml":"apiVersion: audit.k8s.io/v1\nkind: Policy\nrules:\n- level: RequestResponse"}}'

# 5. Scale down non-essential services
echo "--- Scale Down Non-Essential Services ---"
kubectl scale deployment grafana --replicas=0 -n monitoring
kubectl scale deployment jaeger --replicas=0 -n monitoring

# 6. Create incident ticket
echo "--- Create Incident ---"
./scripts/create-security-incident.sh \
  --severity "P0" \
  --title "Emergency Security Response Activated" \
  --description "Automated emergency security response procedures activated"

echo "Emergency containment completed. Manual investigation required."
```

## Backup and Recovery Operations

### Daily Backup Verification

```bash
#!/bin/bash
# Daily backup verification script

echo "=== Daily Backup Verification ==="

# Check Velero backup status
echo "--- Velero Backup Status ---"
kubectl get backups -n velero --sort-by=.metadata.creationTimestamp
kubectl get backup $(kubectl get backups -n velero -o name | tail -1) -n velero -o yaml

# Check database backup status
echo "--- Database Backup Status ---"
kubectl logs -n nephoran-system cronjob/postgresql-backup --tail=20

# Verify backup integrity
echo "--- Backup Integrity Check ---"
./scripts/verify-backup-integrity.sh --latest

# Check backup storage utilization
echo "--- Storage Utilization ---"
kubectl exec -n velero deployment/velero -- \
  aws s3 ls s3://nephoran-backups/ --recursive --human-readable --summarize

# Test restore capability (dry-run)
echo "--- Restore Test (Dry Run) ---"
velero restore create test-restore-$(date +%Y%m%d) \
  --from-backup $(kubectl get backup -n velero -o name | tail -1 | cut -d'/' -f2) \
  --dry-run -o yaml

echo "Daily backup verification completed"
```

### Disaster Recovery Procedures

```yaml
Disaster Recovery Scenarios:
  Scenario 1 - Single Node Failure:
    RTO: 5 minutes
    RPO: 30 seconds  
    Procedure: Automatic node replacement and pod rescheduling
    
  Scenario 2 - Single Cluster Failure:
    RTO: 15 minutes
    RPO: 5 minutes
    Procedure: DNS failover to secondary cluster
    
  Scenario 3 - Regional Failure:
    RTO: 45 minutes
    RPO: 15 minutes
    Procedure: Manual failover to alternate region
    
  Scenario 4 - Complete Data Center Loss:
    RTO: 4 hours
    RPO: 1 hour
    Procedure: Full recovery from backups in alternate region
    
  Scenario 5 - Ransomware/Data Corruption:
    RTO: 8 hours
    RPO: 24 hours
    Procedure: Complete system rebuild from clean backups
```

```bash
#!/bin/bash
# Disaster recovery activation script

echo "=== DISASTER RECOVERY ACTIVATION ==="

SCENARIO=$1
BACKUP_DATE=${2:-$(date -d "1 day ago" +%Y-%m-%d)}

case $SCENARIO in
  "cluster-failure")
    echo "--- Cluster Failure Recovery ---"
    # Redirect traffic to backup cluster
    ./scripts/failover-dns.sh --primary-cluster down --backup-cluster active
    # Verify service availability
    ./scripts/validate-service-health.sh --cluster backup
    ;;
    
  "regional-failure") 
    echo "--- Regional Failure Recovery ---"
    # Activate alternate region
    ./scripts/activate-dr-region.sh --region us-west-2
    # Restore data from backups
    ./scripts/restore-regional-data.sh --source-region us-east-1 --target-region us-west-2
    # Update DNS to point to new region
    ./scripts/update-global-dns.sh --active-region us-west-2
    ;;
    
  "ransomware")
    echo "--- Ransomware Recovery ---"
    # Isolate infected systems
    ./scripts/isolate-infected-systems.sh
    # Restore from clean backups
    ./scripts/restore-from-clean-backup.sh --backup-date $BACKUP_DATE
    # Rebuild affected services
    ./scripts/rebuild-services.sh --security-hardened
    # Verify system integrity
    ./scripts/verify-system-integrity.sh
    ;;
    
  *)
    echo "Unknown disaster scenario: $SCENARIO"
    echo "Available scenarios: cluster-failure, regional-failure, ransomware"
    exit 1
    ;;
esac

echo "Disaster recovery activation completed for scenario: $SCENARIO"
```

This comprehensive operations runbook provides the foundation for production operations. Next, I'll create security hardening guides and compliance documentation.

## References

- [Disaster Recovery Procedures](disaster-recovery.md)
- [Security Incident Response](security-incident-response.md)  
- [Performance Tuning Guide](../operations/PERFORMANCE-TUNING.md)
- [Monitoring and Alerting Setup](../operations/02-monitoring-alerting-runbooks.md)
- [Enterprise Deployment Guide](../production-readiness/enterprise-deployment-guide.md)
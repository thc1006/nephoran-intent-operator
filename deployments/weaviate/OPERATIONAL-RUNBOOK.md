# Weaviate Operational Runbook for Nephoran Intent Operator

## Overview

This runbook provides comprehensive operational procedures for managing the Weaviate vector database deployment in the Nephoran Intent Operator production environment.

## Table of Contents

1. [Deployment](#deployment)
2. [Daily Operations](#daily-operations)
3. [Monitoring and Alerting](#monitoring-and-alerting)
4. [Backup and Recovery](#backup-and-recovery)
5. [Troubleshooting](#troubleshooting)
6. [Scaling](#scaling)
7. [Maintenance](#maintenance)
8. [Emergency Procedures](#emergency-procedures)

## Deployment

### Initial Deployment

1. **Prerequisites**
   ```bash
   # Ensure kubectl is configured
   kubectl cluster-info
   
   # Verify namespace exists
   kubectl get namespace nephoran-system
   
   # Check storage classes
   kubectl get storageclass gp3-encrypted gp2-encrypted sc1-encrypted
   ```

2. **Deploy Weaviate**
   ```bash
   cd deployments/weaviate
   chmod +x deploy-and-validate.sh
   ./deploy-and-validate.sh
   ```

3. **Verify Deployment**
   ```bash
   kubectl get pods -n nephoran-system -l app=weaviate
   kubectl get svc -n nephoran-system -l app=weaviate
   kubectl get pvc -n nephoran-system -l app=weaviate
   ```

### Configuration

1. **Set OpenAI API Key**
   ```bash
   kubectl create secret generic openai-api-key \
     --from-literal=api-key=YOUR_OPENAI_API_KEY \
     -n nephoran-system
   ```

2. **Configure Backup Notifications**
   ```bash
   kubectl patch secret backup-notifications -n nephoran-system \
     --type merge -p='{"data":{"webhook-url":"BASE64_ENCODED_URL"}}'
   ```

## Daily Operations

### Health Checks

1. **Pod Health**
   ```bash
   kubectl get pods -n nephoran-system -l app=weaviate
   kubectl top pods -n nephoran-system -l app=weaviate
   ```

2. **Service Connectivity**
   ```bash
   kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
   curl -H "Authorization: Bearer $(kubectl get secret weaviate-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d)" \
     http://localhost:8080/v1/.well-known/ready
   ```

3. **Storage Usage**
   ```bash
   kubectl get pvc -n nephoran-system -l app=weaviate
   kubectl describe pvc weaviate-pvc -n nephoran-system
   ```

### Performance Monitoring

1. **Query Metrics**
   ```bash
   # Access Grafana dashboard
   kubectl port-forward svc/grafana 3000:3000 -n monitoring
   # Navigate to Weaviate Overview dashboard
   ```

2. **Resource Usage**
   ```bash
   kubectl top pods -n nephoran-system -l app=weaviate
   kubectl describe hpa weaviate-hpa -n nephoran-system
   ```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Availability Metrics**
   - Pod readiness and liveness
   - Service endpoint availability
   - Cluster quorum status

2. **Performance Metrics**
   - Query latency (p95, p99)
   - Query throughput
   - Memory and CPU usage
   - Disk I/O patterns

3. **Business Metrics**
   - Document count in knowledge base
   - Embedding generation rate
   - RAG query success rate
   - Telecom accuracy score

### Alert Response Procedures

#### Critical Alerts

**WeaviateDown**
1. Check pod status: `kubectl get pods -n nephoran-system -l app=weaviate`
2. Review pod logs: `kubectl logs -l app=weaviate -n nephoran-system --tail=100`
3. Check node resources: `kubectl describe nodes`
4. Restart if necessary: `kubectl rollout restart deployment/weaviate -n nephoran-system`

**WeaviateClusterUnhealthy**
1. Identify unhealthy nodes: `kubectl get pods -n nephoran-system -l app=weaviate -o wide`
2. Check networking: `kubectl exec -it <pod-name> -n nephoran-system -- ping <other-pod-ip>`
3. Review cluster formation logs
4. Consider scaling up: `kubectl scale deployment weaviate --replicas=5 -n nephoran-system`

**WeaviateCriticalMemoryUsage**
1. Immediate scale-up: `kubectl scale deployment weaviate --replicas=<current+2> -n nephoran-system`
2. Check memory leaks in logs
3. Review query patterns for inefficiencies
4. Consider vertical scaling of memory limits

#### Warning Alerts

**WeaviateHighMemoryUsage**
1. Monitor trend over 30 minutes
2. Check for memory-intensive queries
3. Review vector index growth
4. Plan for capacity increase if trend continues

**WeaviateQueryLatencyHigh**
1. Check current query load
2. Review slow query logs
3. Analyze vector index fragmentation
4. Consider adding read replicas

**WeaviateDiskSpaceLow**
1. Clean up old backup files
2. Review log retention policies
3. Plan storage expansion
4. Consider archiving old data

## Backup and Recovery

### Backup Verification

1. **Check Backup Jobs**
   ```bash
   kubectl get cronjobs -n nephoran-system -l component=backup
   kubectl get jobs -n nephoran-system -l component=backup --sort-by=.metadata.creationTimestamp
   ```

2. **Verify Backup Files**
   ```bash
   # Check S3 bucket (replace with your bucket)
   aws s3 ls s3://nephoran-weaviate-backups/production/daily/ --recursive
   ```

3. **Test Restore Procedure**
   ```bash
   # Run backup script with list option
   kubectl exec -it <backup-pod> -n nephoran-system -- /scripts/restore-script.sh --list-backups
   ```

### Recovery Procedures

#### Full Cluster Recovery

1. **Prepare Recovery Environment**
   ```bash
   # Scale down current deployment
   kubectl scale deployment weaviate --replicas=0 -n nephoran-system
   
   # Wait for pods to terminate
   kubectl wait --for=delete pod -l app=weaviate -n nephoran-system --timeout=300s
   ```

2. **Restore from Backup**
   ```bash
   # Get latest backup ID
   BACKUP_ID=$(aws s3 ls s3://nephoran-weaviate-backups/production/daily/ | sort | tail -1 | awk '{print $4}' | sed 's/.tar.gz//')
   
   # Create restore job
   kubectl create job weaviate-restore --from=cronjob/weaviate-daily-backup -n nephoran-system
   kubectl set env job/weaviate-restore BACKUP_ID=$BACKUP_ID -n nephoran-system
   ```

3. **Verify Recovery**
   ```bash
   # Scale up deployment
   kubectl scale deployment weaviate --replicas=3 -n nephoran-system
   
   # Wait for readiness
   kubectl wait --for=condition=available deployment/weaviate -n nephoran-system --timeout=600s
   
   # Verify data integrity
   kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
   curl -H "Authorization: Bearer $(kubectl get secret weaviate-api-key -n nephoran-system -o jsonpath='{.data.api-key}' | base64 -d)" \
     -H "Content-Type: application/json" \
     -d '{"query": "{Aggregate{TelecomKnowledge{meta{count}}}}"}' \
     http://localhost:8080/v1/graphql
   ```

#### Point-in-Time Recovery

1. **Identify Recovery Point**
   ```bash
   aws s3 ls s3://nephoran-weaviate-backups/production/hourly/ --recursive | grep "$(date -d '2 hours ago' +%Y%m%d)"
   ```

2. **Execute Recovery**
   ```bash
   # Use specific backup ID for recovery
   kubectl exec -it <backup-pod> -n nephoran-system -- /scripts/restore-script.sh --backup-id <specific-backup-id>
   ```

## Scaling

### Horizontal Scaling

1. **Manual Scaling**
   ```bash
   # Scale up
   kubectl scale deployment weaviate --replicas=5 -n nephoran-system
   
   # Verify scaling
   kubectl get pods -n nephoran-system -l app=weaviate
   kubectl rollout status deployment/weaviate -n nephoran-system
   ```

2. **Auto-scaling Configuration**
   ```bash
   # Check HPA status
   kubectl describe hpa weaviate-hpa -n nephoran-system
   
   # Modify HPA targets
   kubectl patch hpa weaviate-hpa -n nephoran-system --type merge -p='{"spec":{"minReplicas":5,"maxReplicas":15}}'
   ```

### Vertical Scaling

1. **Update Resource Limits**
   ```bash
   kubectl patch deployment weaviate -n nephoran-system --type merge -p='{
     "spec": {
       "template": {
         "spec": {
           "containers": [{
             "name": "weaviate",
             "resources": {
               "requests": {"memory": "16Gi", "cpu": "4000m"},
               "limits": {"memory": "48Gi", "cpu": "12000m"}
             }
           }]
         }
       }
     }
   }'
   ```

2. **Storage Expansion**
   ```bash
   # Expand PVC (if storage class supports it)
   kubectl patch pvc weaviate-pvc -n nephoran-system --type merge -p='{"spec":{"resources":{"requests":{"storage":"1Ti"}}}}'
   ```

## Troubleshooting

### Common Issues

#### 1. Pod Stuck in Pending State

**Symptoms**: Pods remain in Pending status
**Diagnosis**:
```bash
kubectl describe pod <pod-name> -n nephoran-system
kubectl get events -n nephoran-system --sort-by='.metadata.creationTimestamp'
```

**Resolution**:
- Check node resources: `kubectl describe nodes`
- Verify PVC availability: `kubectl get pvc -n nephoran-system`
- Review node affinity rules
- Check storage class provisioner

#### 2. High Memory Usage

**Symptoms**: Memory usage exceeding 85% consistently
**Diagnosis**:
```bash
kubectl top pods -n nephoran-system -l app=weaviate
kubectl exec -it <pod-name> -n nephoran-system -- free -h
```

**Resolution**:
- Scale horizontally: Add more replicas
- Optimize vector cache settings
- Review query patterns for inefficiencies
- Consider vertical scaling

#### 3. Query Latency Issues

**Symptoms**: Query response times > 1 second
**Diagnosis**:
```bash
# Check current load
kubectl port-forward svc/weaviate 8080:8080 -n nephoran-system &
curl -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/meta

# Review logs for slow queries
kubectl logs -l app=weaviate -n nephoran-system | grep "slow"
```

**Resolution**:
- Review vector index configuration
- Optimize query structure
- Add read replicas
- Check network latency between pods

#### 4. Backup Failures

**Symptoms**: Backup jobs failing consistently
**Diagnosis**:
```bash
kubectl logs job/<backup-job-name> -n nephoran-system
kubectl describe cronjob <backup-cronjob-name> -n nephoran-system
```

**Resolution**:
- Check S3 credentials and permissions
- Verify network connectivity to S3
- Review disk space on backup volumes
- Check Weaviate API connectivity

### Debugging Commands

```bash
# Pod debugging
kubectl exec -it <pod-name> -n nephoran-system -- /bin/bash
kubectl logs <pod-name> -n nephoran-system --previous

# Network debugging
kubectl exec -it <pod-name> -n nephoran-system -- netstat -tuln
kubectl exec -it <pod-name> -n nephoran-system -- nslookup weaviate.nephoran-system.svc.cluster.local

# Storage debugging
kubectl exec -it <pod-name> -n nephoran-system -- df -h
kubectl exec -it <pod-name> -n nephoran-system -- ls -la /var/lib/weaviate

# Configuration debugging
kubectl get configmap weaviate-config -n nephoran-system -o yaml
kubectl get secret weaviate-api-key -n nephoran-system -o yaml
```

## Maintenance

### Regular Maintenance Tasks

#### Weekly
- Review backup job status and success rates
- Check storage usage trends
- Review performance metrics and identify optimization opportunities
- Update security patches if available

#### Monthly
- Perform backup restore test
- Review and optimize vector indexes
- Update Weaviate to latest stable version (if needed)
- Review and clean up old backup files

#### Quarterly
- Conduct disaster recovery drill
- Review and update capacity planning
- Performance benchmark against baseline
- Security audit and vulnerability assessment

### Upgrade Procedures

1. **Pre-upgrade Checklist**
   - Create full backup
   - Review release notes for breaking changes
   - Test upgrade in staging environment
   - Plan rollback procedure

2. **Upgrade Process**
   ```bash
   # Create backup
   kubectl create job weaviate-pre-upgrade-backup --from=cronjob/weaviate-daily-backup -n nephoran-system
   
   # Update deployment image
   kubectl set image deployment/weaviate weaviate=semitechnologies/weaviate:NEW_VERSION -n nephoran-system
   
   # Monitor rollout
   kubectl rollout status deployment/weaviate -n nephoran-system
   
   # Verify functionality
   ./deploy-and-validate.sh
   ```

3. **Rollback Procedure**
   ```bash
   # Rollback deployment
   kubectl rollout undo deployment/weaviate -n nephoran-system
   
   # Wait for rollback completion
   kubectl rollout status deployment/weaviate -n nephoran-system
   
   # Verify system health
   kubectl get pods -n nephoran-system -l app=weaviate
   ```

## Emergency Procedures

### Data Loss Incident

1. **Immediate Response**
   - Stop all write operations
   - Assess scope of data loss
   - Notify stakeholders
   - Begin recovery procedures

2. **Recovery Steps**
   - Identify latest valid backup
   - Execute point-in-time recovery
   - Validate data integrity
   - Resume normal operations

### Security Incident

1. **Immediate Response**
   - Isolate affected components
   - Rotate all API keys and secrets
   - Review access logs
   - Notify security team

2. **Recovery Steps**
   - Apply security patches
   - Update RBAC policies
   - Audit all configurations
   - Implement additional monitoring

### Complete Cluster Failure

1. **Assessment**
   - Determine cause of failure
   - Assess data integrity
   - Evaluate recovery options

2. **Recovery**
   - Deploy fresh cluster
   - Restore from latest backup
   - Validate all services
   - Update DNS/load balancer endpoints

## Contact Information

- **Platform Team**: platform-team@nephoran.com
- **AI/ML Team**: ai-ml-team@nephoran.com
- **On-Call**: +1-555-ON-CALL (665-2255)
- **Escalation Manager**: escalation@nephoran.com

## References

- [Weaviate Documentation](https://weaviate.io/developers/weaviate)
- [Kubernetes Troubleshooting Guide](https://kubernetes.io/docs/tasks/debug-application-cluster/)
- [Nephoran Intent Operator Documentation](../docs/)
- [Monitoring Dashboards](https://grafana.nephoran.com/dashboards)

---

**Document Version**: 1.0  
**Last Updated**: $(date)  
**Next Review**: $(date -d '+3 months')
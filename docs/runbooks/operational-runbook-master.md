# Master Operational Runbook - Nephoran Intent Operator

**Version:** 3.0  
**Last Updated:** January 2025  
**Audience:** Site Reliability Engineers, Platform Operations Teams  
**Scope:** Comprehensive daily, weekly, and monthly operational procedures

## Table of Contents

1. [System Overview](#system-overview)
2. [Daily Operations](#daily-operations)
3. [Weekly Operations](#weekly-operations)
4. [Monthly Operations](#monthly-operations)
5. [Service Management](#service-management)
6. [Resource Management](#resource-management)
7. [Maintenance Procedures](#maintenance-procedures)
8. [Health Monitoring](#health-monitoring)
9. [Performance Optimization](#performance-optimization)

## System Overview

### Core Components

| Component | Purpose | Port | Health Endpoint | SLA Target |
|-----------|---------|------|-----------------|------------|
| LLM Processor | Natural language intent processing | 8080 | /healthz, /readyz | 99.9% |
| RAG API | Vector database and knowledge retrieval | 5001 | /health, /ready | 99.9% |
| Nephio Bridge | NetworkIntent CRD controller | N/A | Controller metrics | 99.9% |
| O-RAN Adaptor | O-RAN interface implementations | 8082 | /healthz | 99.5% |
| Weaviate | Vector database storage | 8080 | /v1/.well-known/ready | 99.9% |

### Critical Dependencies

- OpenAI API (or configured LLM provider)
- Kubernetes cluster (1.28+)
- Weaviate vector database
- Redis (for caching)
- PostgreSQL (for state management)
- Git repository (for GitOps)

## Daily Operations

### Morning Health Check (08:00 UTC)

```bash
#!/bin/bash
# Daily morning health check - comprehensive system validation

echo "=== Nephoran Daily Health Check - $(date) ==="

# 1. Cluster Health
echo "--- Kubernetes Cluster Health ---"
kubectl cluster-info
kubectl get nodes -o wide
kubectl top nodes
echo ""

# 2. Service Status
echo "--- Service Status ---"
kubectl get pods -n nephoran-system -o wide
kubectl get deployments -n nephoran-system
kubectl get statefulsets -n nephoran-system
echo ""

# 3. Resource Utilization
echo "--- Resource Utilization ---"
kubectl top pods -n nephoran-system --sort-by=cpu
kubectl top pods -n nephoran-system --sort-by=memory
echo ""

# 4. Intent Processing Status
echo "--- Intent Processing Status (Last 24h) ---"
kubectl get networkintents --all-namespaces | \
  awk '{print $NF}' | sort | uniq -c
  
# Check for stuck intents
STUCK_INTENTS=$(kubectl get networkintents --all-namespaces | grep -E "Processing|Pending" | wc -l)
if [ $STUCK_INTENTS -gt 0 ]; then
  echo "⚠️  WARNING: $STUCK_INTENTS intents stuck in processing"
  kubectl get networkintents --all-namespaces | grep -E "Processing|Pending"
fi
echo ""

# 5. Error Analysis
echo "--- Recent Errors (Last Hour) ---"
kubectl get events -n nephoran-system --field-selector type=Warning \
  --sort-by='.lastTimestamp' | head -10
echo ""

# 6. Certificate Status
echo "--- Certificate Status ---"
kubectl get certificates -n nephoran-system -o wide
for cert in $(kubectl get certificates -n nephoran-system -o name); do
  kubectl get $cert -n nephoran-system -o jsonpath='{.status.renewalTime}' 2>/dev/null
  echo " - $cert"
done
echo ""

# 7. Storage Status
echo "--- Storage Status ---"
kubectl get pvc -n nephoran-system
kubectl exec -n nephoran-system deployment/weaviate -- df -h /var/lib/weaviate 2>/dev/null || echo "Weaviate storage check failed"
echo ""

# 8. External Dependencies
echo "--- External Dependencies ---"
# Check OpenAI API
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -s -o /dev/null -w "OpenAI API: %{http_code}\n" https://api.openai.com/v1/models 2>/dev/null || echo "OpenAI API check failed"

# Check Git connectivity
kubectl exec -n nephoran-system deployment/nephoran-controller -- \
  git ls-remote --heads origin 2>/dev/null > /dev/null && echo "Git: OK" || echo "Git: FAILED"
echo ""

# 9. Performance Metrics
echo "--- Performance Summary (Last Hour) ---"
# This would typically query Prometheus
echo "Intent Processing P95 Latency: $(curl -s 'http://prometheus.monitoring:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_intent_processing_duration_seconds_bucket[1h]))' 2>/dev/null | jq -r '.data.result[0].value[1]' 2>/dev/null || echo 'N/A')s"
echo "Success Rate: $(curl -s 'http://prometheus.monitoring:9090/api/v1/query?query=rate(nephoran_networkintent_success_total[1h])/rate(nephoran_networkintent_total[1h])' 2>/dev/null | jq -r '.data.result[0].value[1]' 2>/dev/null || echo 'N/A')"
echo ""

# 10. Summary
echo "=== Health Check Summary ==="
UNHEALTHY_PODS=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
if [ $UNHEALTHY_PODS -eq 0 ] && [ $STUCK_INTENTS -eq 0 ]; then
  echo "✅ System Status: HEALTHY"
else
  echo "⚠️  System Status: DEGRADED"
  echo "   - Unhealthy Pods: $UNHEALTHY_PODS"
  echo "   - Stuck Intents: $STUCK_INTENTS"
fi
echo "Completed at $(date)"
```

### Afternoon Check (14:00 UTC)

```bash
#!/bin/bash
# Afternoon performance and capacity check

echo "=== Afternoon Performance Check - $(date) ==="

# Check auto-scaling status
echo "--- Auto-scaling Status ---"
kubectl get hpa -n nephoran-system
echo ""

# Check queue depths
echo "--- Processing Queue Status ---"
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -s localhost:8080/metrics | grep queue_depth 2>/dev/null || echo "Queue metrics unavailable"
echo ""

# Check cache performance
echo "--- Cache Performance ---"
kubectl exec -n nephoran-system deployment/rag-api -- \
  curl -s localhost:5001/cache/stats 2>/dev/null | jq '.' || echo "Cache stats unavailable"
echo ""

# Check for capacity issues
echo "--- Capacity Alerts ---"
kubectl top nodes | awk '$3>80 || $5>80 {print "WARNING: Node " $1 " - CPU: " $3 " Memory: " $5}'
kubectl top pods -n nephoran-system | awk '$2>80 || $3>80 {print "WARNING: Pod " $1 " - CPU: " $2 " Memory: " $3}'
```

### End of Day Report (17:00 UTC)

```bash
#!/bin/bash
# Generate daily summary report

echo "=== Daily Operations Summary - $(date +%Y-%m-%d) ==="

# Intent processing statistics
echo "--- Intent Processing Statistics ---"
echo "Total Intents Processed: $(kubectl get networkintents --all-namespaces --no-headers | wc -l)"
echo "Successful: $(kubectl get networkintents --all-namespaces --no-headers | grep Succeeded | wc -l)"
echo "Failed: $(kubectl get networkintents --all-namespaces --no-headers | grep Failed | wc -l)"
echo "In Progress: $(kubectl get networkintents --all-namespaces --no-headers | grep -E 'Processing|Pending' | wc -l)"
echo ""

# Service availability
echo "--- Service Availability ---"
for deployment in llm-processor rag-api nephoran-controller; do
  READY=$(kubectl get deployment $deployment -n nephoran-system -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
  DESIRED=$(kubectl get deployment $deployment -n nephoran-system -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)
  echo "$deployment: $READY/$DESIRED replicas ready"
done
echo ""

# Incident summary
echo "--- Incident Summary ---"
kubectl get events -n nephoran-system --field-selector type=Warning | wc -l | xargs echo "Warning Events:"
kubectl get events -n nephoran-system --field-selector type=Normal | wc -l | xargs echo "Normal Events:"
echo ""

# Tomorrow's maintenance
echo "--- Scheduled Maintenance ---"
# Check for scheduled maintenance windows
echo "No scheduled maintenance" # This would typically query a maintenance calendar
```

## Weekly Operations

### Monday: Review and Planning

```bash
#!/bin/bash
# Monday weekly review and planning

echo "=== Weekly Review - Week of $(date +%Y-%m-%d) ==="

# Review last week's incidents
echo "--- Last Week's Incidents ---"
kubectl get events -n nephoran-system --field-selector type=Warning \
  --since=168h | awk '{print $1, $4, $6}' | sort | uniq -c | sort -rn | head -20

# Check backup status
echo "--- Backup Status ---"
kubectl get backups -n velero --sort-by=.metadata.creationTimestamp | tail -10

# Review security scan results
echo "--- Security Scan Summary ---"
kubectl get vulnerabilityreports -n nephoran-system -o wide | grep -E "HIGH|CRITICAL" | wc -l | xargs echo "High/Critical vulnerabilities:"

# Capacity planning
echo "--- Capacity Trends ---"
echo "Average CPU usage last week: $(calculate_weekly_avg cpu)"
echo "Average Memory usage last week: $(calculate_weekly_avg memory)"
echo "Intent volume trend: $(calculate_intent_trend)"
```

### Wednesday: Maintenance Window

```bash
#!/bin/bash
# Wednesday maintenance procedures

echo "=== Weekly Maintenance - $(date) ==="

# 1. Database maintenance
echo "--- Database Maintenance ---"
kubectl exec -n nephoran-system postgresql-0 -- \
  psql -U postgres -c "VACUUM ANALYZE;" 2>/dev/null && echo "✅ Database vacuum completed" || echo "❌ Database vacuum failed"

# 2. Log rotation
echo "--- Log Rotation ---"
for pod in $(kubectl get pods -n nephoran-system -o name); do
  kubectl exec -n nephoran-system $pod -- sh -c 'find /var/log -name "*.log" -mtime +7 -delete' 2>/dev/null
done
echo "✅ Log rotation completed"

# 3. Cache cleanup
echo "--- Cache Cleanup ---"
kubectl exec -n nephoran-system deployment/rag-api -- \
  curl -X POST localhost:5001/cache/cleanup 2>/dev/null && echo "✅ Cache cleanup completed" || echo "❌ Cache cleanup failed"

# 4. Vector database optimization
echo "--- Vector Database Optimization ---"
kubectl exec -n nephoran-system weaviate-0 -- \
  curl -X POST localhost:8080/v1/schema/TelecomDocument/shards/optimize 2>/dev/null && echo "✅ Vector DB optimized" || echo "❌ Vector DB optimization failed"

# 5. Certificate renewal check
echo "--- Certificate Renewal Check ---"
for cert in $(kubectl get certificates -n nephoran-system -o name); do
  DAYS_LEFT=$(kubectl get $cert -n nephoran-system -o jsonpath='{.status.renewalTime}' | xargs -I {} date -d {} +%s | xargs -I {} expr {} - $(date +%s) | xargs -I {} expr {} / 86400)
  if [ $DAYS_LEFT -lt 30 ]; then
    echo "⚠️  $cert expires in $DAYS_LEFT days"
  fi
done
```

### Friday: Backup Verification

```bash
#!/bin/bash
# Friday backup verification and testing

echo "=== Weekly Backup Verification - $(date) ==="

# Verify latest backups
echo "--- Backup Inventory ---"
velero backup get --output table

# Test restore capability
echo "--- Restore Test (Dry Run) ---"
LATEST_BACKUP=$(velero backup get -o json | jq -r '.items[-1].metadata.name')
velero restore create test-restore-$(date +%Y%m%d) \
  --from-backup $LATEST_BACKUP \
  --namespace-mappings nephoran-system:nephoran-test \
  --dry-run -o yaml > /tmp/restore-test.yaml
echo "✅ Restore dry run completed successfully"

# Verify backup integrity
echo "--- Backup Integrity Check ---"
./scripts/verify-backup-integrity.sh --backup $LATEST_BACKUP
```

## Monthly Operations

### Monthly Performance Review

```bash
#!/bin/bash
# Monthly performance analysis and reporting

echo "=== Monthly Performance Review - $(date +%B-%Y) ==="

# Generate performance report
./scripts/generate-performance-report.sh --month $(date +%Y-%m)

# Capacity planning analysis
echo "--- Capacity Planning ---"
./scripts/analyze-capacity-trends.sh --lookback 30d --forecast 90d

# Cost analysis
echo "--- Cost Analysis ---"
./scripts/generate-cost-report.sh --month $(date +%Y-%m)

# SLA compliance
echo "--- SLA Compliance ---"
./scripts/check-sla-compliance.sh --month $(date +%Y-%m)
```

### Monthly Security Audit

```bash
#!/bin/bash
# Monthly security audit procedures

echo "=== Monthly Security Audit - $(date) ==="

# Container image scanning
echo "--- Container Image Security ---"
for image in $(kubectl get pods -n nephoran-system -o jsonpath='{.items[*].spec.containers[*].image}' | tr ' ' '\n' | sort -u); do
  echo "Scanning $image..."
  trivy image --severity HIGH,CRITICAL $image
done

# RBAC audit
echo "--- RBAC Audit ---"
kubectl auth can-i --list --as=system:serviceaccount:nephoran-system:default

# Network policy review
echo "--- Network Policy Review ---"
kubectl get networkpolicies -n nephoran-system -o yaml | kubectl neat

# Secret rotation status
echo "--- Secret Rotation Status ---"
for secret in $(kubectl get secrets -n nephoran-system -o name); do
  AGE=$(kubectl get $secret -n nephoran-system -o jsonpath='{.metadata.creationTimestamp}' | xargs -I {} date -d {} +%s | xargs -I {} expr $(date +%s) - {} | xargs -I {} expr {} / 86400)
  if [ $AGE -gt 90 ]; then
    echo "⚠️  $secret is $AGE days old - consider rotation"
  fi
done
```

## Service Management

### Service Health Checks

```bash
#!/bin/bash
# Comprehensive service health validation

function check_service_health() {
  local service=$1
  local namespace=$2
  local port=$3
  local endpoint=$4
  
  echo "Checking $service..."
  
  # Check pod status
  kubectl get pods -n $namespace -l app=$service --no-headers | grep -v Running && echo "⚠️  Unhealthy pods found" || echo "✅ All pods running"
  
  # Check endpoint
  kubectl exec -n $namespace deployment/$service -- curl -s -f localhost:$port$endpoint > /dev/null 2>&1 && echo "✅ Health endpoint responsive" || echo "❌ Health endpoint not responding"
  
  # Check recent restarts
  RESTARTS=$(kubectl get pods -n $namespace -l app=$service -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}' | tr ' ' '+' | bc)
  [ $RESTARTS -gt 0 ] && echo "⚠️  Total restarts: $RESTARTS" || echo "✅ No recent restarts"
  
  echo ""
}

# Check all services
check_service_health "llm-processor" "nephoran-system" "8080" "/healthz"
check_service_health "rag-api" "nephoran-system" "5001" "/health"
check_service_health "nephoran-controller" "nephoran-system" "8080" "/metrics"
check_service_health "weaviate" "nephoran-system" "8080" "/v1/.well-known/ready"
```

### Service Scaling Operations

```bash
#!/bin/bash
# Service scaling procedures

function scale_service() {
  local service=$1
  local replicas=$2
  local namespace="nephoran-system"
  
  echo "Scaling $service to $replicas replicas..."
  
  # Record current state
  CURRENT=$(kubectl get deployment $service -n $namespace -o jsonpath='{.spec.replicas}')
  echo "Current replicas: $CURRENT"
  
  # Apply scaling
  kubectl scale deployment $service -n $namespace --replicas=$replicas
  
  # Wait for rollout
  kubectl rollout status deployment/$service -n $namespace --timeout=300s
  
  # Verify
  ACTUAL=$(kubectl get deployment $service -n $namespace -o jsonpath='{.status.readyReplicas}')
  [ "$ACTUAL" == "$replicas" ] && echo "✅ Scaling completed" || echo "❌ Scaling failed - only $ACTUAL/$replicas ready"
}

# Emergency scaling procedures
scale_up_all() {
  echo "=== Emergency Scale Up ==="
  scale_service "llm-processor" 10
  scale_service "rag-api" 5
  scale_service "nephoran-controller" 3
}

scale_down_all() {
  echo "=== Controlled Scale Down ==="
  scale_service "llm-processor" 3
  scale_service "rag-api" 2
  scale_service "nephoran-controller" 2
}
```

## Resource Management

### Cache Management

```bash
#!/bin/bash
# Cache management procedures

# Warm cache with frequently used data
warm_cache() {
  echo "=== Cache Warming ==="
  
  # Warm LLM processor cache
  kubectl exec -n nephoran-system deployment/llm-processor -- \
    curl -X POST localhost:8080/cache/warm \
    -H "Content-Type: application/json" \
    -d '{
      "strategy": "popular_queries",
      "limit": 100,
      "categories": ["network_deployment", "scaling_operations", "policy_management"]
    }' 2>/dev/null && echo "✅ LLM cache warmed" || echo "❌ LLM cache warming failed"
  
  # Warm RAG cache
  kubectl exec -n nephoran-system deployment/rag-api -- \
    curl -X POST localhost:5001/cache/warm 2>/dev/null && echo "✅ RAG cache warmed" || echo "❌ RAG cache warming failed"
}

# Clear cache (use with caution)
clear_cache() {
  echo "=== Cache Clearing ==="
  read -p "Are you sure you want to clear all caches? (y/N) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl exec -n nephoran-system deployment/llm-processor -- \
      curl -X DELETE localhost:8080/cache/clear 2>/dev/null && echo "✅ LLM cache cleared" || echo "❌ LLM cache clear failed"
    
    kubectl exec -n nephoran-system deployment/rag-api -- \
      curl -X DELETE localhost:5001/cache/clear 2>/dev/null && echo "✅ RAG cache cleared" || echo "❌ RAG cache clear failed"
  fi
}

# Cache statistics
cache_stats() {
  echo "=== Cache Statistics ==="
  
  echo "--- LLM Processor Cache ---"
  kubectl exec -n nephoran-system deployment/llm-processor -- \
    curl -s localhost:8080/cache/stats 2>/dev/null | jq '.' || echo "Stats unavailable"
  
  echo "--- RAG API Cache ---"
  kubectl exec -n nephoran-system deployment/rag-api -- \
    curl -s localhost:5001/cache/stats 2>/dev/null | jq '.' || echo "Stats unavailable"
}
```

### Knowledge Base Management

```bash
#!/bin/bash
# Knowledge base update and management

update_knowledge_base() {
  echo "=== Knowledge Base Update ==="
  
  # Upload new documents
  local DOCS_PATH=${1:-"/path/to/documents"}
  
  if [ -d "$DOCS_PATH" ]; then
    echo "Uploading documents from $DOCS_PATH..."
    ./scripts/update-knowledge-base.sh "$DOCS_PATH"
  else
    echo "❌ Documents path not found: $DOCS_PATH"
    return 1
  fi
  
  # Verify indexing
  echo "--- Verifying Indexing ---"
  kubectl exec -n nephoran-system deployment/rag-api -- \
    curl -s localhost:5001/knowledge/stats 2>/dev/null | jq '.' || echo "Stats unavailable"
  
  # Trigger reindexing if needed
  read -p "Trigger full reindexing? (y/N) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl exec -n nephoran-system deployment/rag-api -- \
      curl -X POST localhost:5001/knowledge/reindex 2>/dev/null && echo "✅ Reindexing started" || echo "❌ Reindexing failed"
  fi
}

# Knowledge base statistics
knowledge_base_stats() {
  echo "=== Knowledge Base Statistics ==="
  
  kubectl exec -n nephoran-system deployment/rag-api -- \
    curl -s localhost:5001/knowledge/stats 2>/dev/null | jq '{
      total_documents: .total_documents,
      total_chunks: .total_chunks,
      index_size: .index_size,
      last_updated: .last_updated,
      categories: .categories
    }' || echo "Stats unavailable"
}
```

## Maintenance Procedures

### Zero-Downtime Updates

```bash
#!/bin/bash
# Zero-downtime update procedure

perform_rolling_update() {
  local component=$1
  local new_image=$2
  
  echo "=== Rolling Update: $component ==="
  
  # Pre-update checks
  echo "--- Pre-update Checks ---"
  kubectl get deployment $component -n nephoran-system
  
  # Create canary deployment (optional)
  echo "--- Creating Canary ---"
  kubectl get deployment $component -n nephoran-system -o yaml | \
    sed "s/name: $component/name: $component-canary/" | \
    sed "s/replicas: .*/replicas: 1/" | \
    kubectl apply -f -
  
  # Update image
  echo "--- Updating Image ---"
  kubectl set image deployment/$component $component=$new_image -n nephoran-system
  
  # Monitor rollout
  echo "--- Monitoring Rollout ---"
  kubectl rollout status deployment/$component -n nephoran-system --timeout=600s
  
  # Verify
  echo "--- Verification ---"
  kubectl get pods -n nephoran-system -l app=$component
  
  # Clean up canary
  kubectl delete deployment $component-canary -n nephoran-system 2>/dev/null
  
  echo "✅ Rolling update completed"
}
```

### Database Maintenance

```bash
#!/bin/bash
# Database maintenance procedures

database_maintenance() {
  echo "=== Database Maintenance ==="
  
  # Check connection pool
  echo "--- Connection Pool Status ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "SELECT count(*) as connections FROM pg_stat_activity;" 2>/dev/null || echo "Connection check failed"
  
  # Vacuum and analyze
  echo "--- Vacuum and Analyze ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "VACUUM ANALYZE;" 2>/dev/null && echo "✅ Vacuum completed" || echo "❌ Vacuum failed"
  
  # Check for long-running queries
  echo "--- Long Running Queries ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "SELECT pid, now() - pg_stat_activity.query_start AS duration, query FROM pg_stat_activity WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';" 2>/dev/null || echo "Query check failed"
  
  # Update statistics
  echo "--- Update Statistics ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "ANALYZE;" 2>/dev/null && echo "✅ Statistics updated" || echo "❌ Statistics update failed"
}
```

## Health Monitoring

### Comprehensive Health Dashboard

```bash
#!/bin/bash
# Generate comprehensive health dashboard

generate_health_dashboard() {
  echo "==================================="
  echo "  Nephoran System Health Dashboard"
  echo "  Generated: $(date)"
  echo "==================================="
  echo ""
  
  # Overall Status
  echo "📊 OVERALL STATUS"
  echo "-----------------"
  UNHEALTHY=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
  if [ $UNHEALTHY -eq 0 ]; then
    echo "🟢 System Status: HEALTHY"
  else
    echo "🔴 System Status: UNHEALTHY ($UNHEALTHY pods not running)"
  fi
  echo ""
  
  # Service Status
  echo "🔧 SERVICE STATUS"
  echo "-----------------"
  for svc in llm-processor rag-api nephoran-controller weaviate; do
    READY=$(kubectl get deployment $svc -n nephoran-system -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    DESIRED=$(kubectl get deployment $svc -n nephoran-system -o jsonpath='{.spec.replicas}' 2>/dev/null || kubectl get statefulset $svc -n nephoran-system -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)
    if [ "$READY" == "$DESIRED" ] && [ "$READY" != "0" ]; then
      echo "🟢 $svc: $READY/$DESIRED"
    else
      echo "🔴 $svc: $READY/$DESIRED"
    fi
  done
  echo ""
  
  # Resource Usage
  echo "💾 RESOURCE USAGE"
  echo "-----------------"
  kubectl top nodes | head -5
  echo ""
  
  # Intent Processing
  echo "📝 INTENT PROCESSING"
  echo "--------------------"
  TOTAL=$(kubectl get networkintents --all-namespaces --no-headers | wc -l)
  SUCCESS=$(kubectl get networkintents --all-namespaces --no-headers | grep Succeeded | wc -l)
  FAILED=$(kubectl get networkintents --all-namespaces --no-headers | grep Failed | wc -l)
  PROCESSING=$(kubectl get networkintents --all-namespaces --no-headers | grep -E 'Processing|Pending' | wc -l)
  echo "Total: $TOTAL | Success: $SUCCESS | Failed: $FAILED | Processing: $PROCESSING"
  echo ""
  
  # Recent Issues
  echo "⚠️  RECENT ISSUES"
  echo "-----------------"
  kubectl get events -n nephoran-system --field-selector type=Warning --sort-by='.lastTimestamp' | head -5
  echo ""
  
  echo "==================================="
}

# Run the dashboard
generate_health_dashboard
```

## Performance Optimization

### Performance Tuning Checklist

```bash
#!/bin/bash
# Performance optimization procedures

optimize_performance() {
  echo "=== Performance Optimization ==="
  
  # 1. JVM Tuning (if applicable)
  echo "--- JVM Optimization ---"
  kubectl patch deployment llm-processor -n nephoran-system --type merge -p \
    '{"spec":{"template":{"spec":{"containers":[{"name":"llm-processor","env":[{"name":"JAVA_OPTS","value":"-Xms2g -Xmx4g -XX:+UseG1GC -XX:MaxGCPauseMillis=100"}]}]}}}}'
  
  # 2. Connection Pool Optimization
  echo "--- Connection Pool Optimization ---"
  kubectl patch configmap app-config -n nephoran-system --type merge -p \
    '{"data":{"database.pool.size":"50","database.pool.timeout":"10s","redis.pool.size":"100"}}'
  
  # 3. Resource Limits Adjustment
  echo "--- Resource Limits Adjustment ---"
  kubectl patch deployment llm-processor -n nephoran-system --type merge -p \
    '{"spec":{"template":{"spec":{"containers":[{"name":"llm-processor","resources":{"requests":{"cpu":"2","memory":"4Gi"},"limits":{"cpu":"4","memory":"8Gi"}}}]}}}}'
  
  # 4. HPA Configuration
  echo "--- HPA Configuration ---"
  kubectl patch hpa llm-processor-hpa -n nephoran-system --type merge -p \
    '{"spec":{"targetCPUUtilizationPercentage":70,"minReplicas":3,"maxReplicas":15}}'
  
  # 5. Vector Database Optimization
  echo "--- Vector Database Optimization ---"
  kubectl exec -n nephoran-system weaviate-0 -- \
    curl -X PUT localhost:8080/v1/schema/TelecomDocument \
    -H "Content-Type: application/json" \
    -d '{"vectorIndexConfig": {"ef": 256, "efConstruction": 512, "maxConnections": 32}}' 2>/dev/null
  
  echo "✅ Performance optimization completed"
}
```

## Troubleshooting Quick Reference

### Common Issues and Quick Fixes

| Issue | Quick Check | Quick Fix |
|-------|------------|-----------|
| High latency | `kubectl top pods -n nephoran-system` | Scale up: `kubectl scale deployment llm-processor --replicas=10` |
| Stuck intents | `kubectl get networkintents --all-namespaces \| grep Processing` | Restart controller: `kubectl rollout restart deployment/nephoran-controller` |
| Memory issues | `kubectl describe pods -n nephoran-system \| grep OOMKilled` | Increase limits or scale horizontally |
| Connection errors | `kubectl logs -n nephoran-system deployment/llm-processor \| grep connection` | Check and reset connection pools |
| Certificate issues | `kubectl get certificates -n nephoran-system` | Force renewal: `kubectl annotate certificate --all cert-manager.io/force-renew=$(date +%s)` |

## Related Runbooks

- [Monitoring & Alerting Runbook](./monitoring-alerting-runbook.md) - Detailed monitoring procedures
- [Incident Response Runbook](./incident-response-runbook.md) - Emergency response procedures
- [Disaster Recovery Runbook](./disaster-recovery-runbook.md) - DR procedures
- [Security Operations Runbook](./security-operations-runbook.md) - Security procedures
- [Weaviate Operations](../../deployments/weaviate/OPERATIONAL-RUNBOOK.md) - Weaviate-specific procedures

---

**Note:** This is a living document. Update procedures based on operational experience and system changes.
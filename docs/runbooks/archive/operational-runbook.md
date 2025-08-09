# Nephoran Intent Operator - Production Monitoring Operational Runbook

## Overview

This runbook provides step-by-step procedures for operating and maintaining the comprehensive monitoring, alerting, and observability system for the Nephoran Intent Operator in production environments.

## Quick Reference

### Emergency Contacts
- **On-Call Engineer**: Use PagerDuty escalation
- **Slack Channels**: 
  - `#nephoran-alerts` - General alerts
  - `#nephoran-critical` - Critical incidents
  - `#nephoran-on-call` - On-call coordination
- **Email**: `oncall@nephoran.com`

### Key URLs
- **Grafana**: `https://grafana.nephoran.local`
- **Prometheus**: `https://prometheus.nephoran.local`
- **Jaeger**: `https://jaeger.nephoran.local`
- **AlertManager**: `https://alertmanager.nephoran.local`

## System Architecture

### Monitoring Stack Components
- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Jaeger**: Distributed tracing
- **AlertManager**: Alert routing and management
- **Custom Health Checks**: Component health monitoring

### Monitored Components
- **NetworkIntent Controller**: Intent processing and reconciliation
- **E2NodeSet Controller**: Node set management
- **LLM Processor**: Language model processing service
- **RAG API**: Retrieval-augmented generation service
- **Weaviate**: Vector database
- **O-RAN Interfaces**: A1, O1, O2 interface adaptors

## Daily Operations

### Morning Health Check (Run Every Day)

1. **Access Monitoring Operations Script**:
   ```bash
   cd /workspaces/nephoran-intent-operator
   ./scripts/monitoring-operations.sh health-check
   ```

2. **Review System Overview Dashboard**:
   - Open Grafana → Nephoran → System Overview
   - Check all components show "Healthy" status
   - Verify intent processing rate is normal
   - Confirm no active alerts

3. **Check Critical Metrics**:
   - System uptime > 99.9%
   - Intent success rate > 95%
   - LLM response time P95 < 2s
   - No critical alerts firing

### Weekly Operations (Run Every Monday)

1. **Performance Review**:
   ```bash
   ./scripts/monitoring-operations.sh analyze-performance
   ```

2. **Collect Weekly Metrics**:
   ```bash
   ./scripts/monitoring-operations.sh collect-metrics
   ```

3. **Review Dashboards**:
   - Business KPIs dashboard for trends
   - Infrastructure dashboard for resource usage
   - LLM & RAG Performance for cost analysis

## Alert Response Procedures

### Severity Levels

#### Critical (P1) - Response Time: < 5 minutes
- System completely down
- >50% of intents failing
- Security breach detected
- Data loss occurring

#### High (P2) - Response Time: < 15 minutes
- Significant degradation
- 25-50% of intents failing
- Important component unavailable
- Performance severely impacted

#### Medium (P3) - Response Time: < 1 hour
- Minor degradation
- 10-25% of intents failing
- Non-critical component issues
- Performance moderately impacted

#### Low (P4) - Response Time: < 4 hours
- Informational alerts
- <10% of intents failing
- Minor performance issues
- Capacity warnings

### Critical Alert Response (P1)

#### 1. System Down Alert
**Symptoms**: All components unreachable, no metrics flowing

**Immediate Actions** (0-5 minutes):
```bash
# Check cluster health
kubectl get nodes
kubectl get pods -n nephoran-system
kubectl get pods -n nephoran-monitoring

# Run auto-repair
./scripts/monitoring-operations.sh auto-repair

# Check for recent changes
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | head -20
```

**Investigation Steps**:
1. Check Kubernetes cluster status
2. Verify network connectivity
3. Check resource utilization
4. Review recent deployments

**Escalation**: If not resolved in 15 minutes, escalate to engineering team

#### 2. High Intent Failure Rate
**Symptoms**: >50% of NetworkIntent processing failing

**Immediate Actions**:
```bash
# Check controller status
kubectl logs -n nephoran-system deployment/nephio-bridge --tail=100

# Check LLM processor
kubectl logs -n nephoran-system deployment/llm-processor --tail=100

# Restart services if needed
./scripts/monitoring-operations.sh restart-service llm-processor
./scripts/monitoring-operations.sh restart-service rag-api
```

**Investigation Steps**:
1. Check LLM API connectivity and quotas
2. Verify Weaviate database health
3. Review error patterns in logs
4. Check resource constraints

#### 3. Component Unavailable
**Symptoms**: Specific component failing health checks

**Immediate Actions**:
```bash
# Identify failed component
./scripts/monitoring-operations.sh health-check

# Restart specific component
./scripts/monitoring-operations.sh restart-component <component-name>

# Check component-specific logs
kubectl logs -n nephoran-monitoring deployment/<component> --tail=100
```

### High Priority Alert Response (P2)

#### 1. Performance Degradation
**Symptoms**: P95 response time > 5 seconds, success rate 75-95%

**Actions**:
```bash
# Analyze performance
./scripts/monitoring-operations.sh analyze-performance

# Check resource usage
kubectl top pods -n nephoran-system
kubectl top nodes

# Scale up if needed
kubectl scale deployment/llm-processor --replicas=3 -n nephoran-system
```

#### 2. Resource Exhaustion
**Symptoms**: High CPU/memory usage, pods being evicted

**Actions**:
```bash
# Check resource usage
kubectl describe nodes | grep -A 5 "Allocated resources"

# Scale components
./scripts/monitoring-operations.sh scale-component prometheus 2

# Check for memory leaks
kubectl top pods -n nephoran-system --sort-by=memory
```

### Medium Priority Alert Response (P3)

#### 1. Cache Miss Rate High
**Symptoms**: RAG cache hit rate < 70%

**Actions**:
1. Check cache configuration in Grafana
2. Review query patterns for optimization
3. Consider increasing cache size
4. Check for cache invalidation issues

#### 2. Moderate Error Rate
**Symptoms**: 10-25% of operations failing

**Actions**:
1. Review error patterns in logs
2. Check for intermittent connectivity issues
3. Monitor for pattern resolution
4. Document findings for trend analysis

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. Prometheus Not Collecting Metrics

**Symptoms**:
- Empty or missing data in Grafana
- "No data" messages in dashboards

**Diagnosis**:
```bash
# Check Prometheus targets
./scripts/monitoring-operations.sh check-prometheus-targets

# Check ServiceMonitor configuration
kubectl get servicemonitor -n nephoran-monitoring -o yaml

# Verify service endpoints
kubectl get endpoints -n nephoran-system
```

**Solutions**:
```bash
# Restart Prometheus
./scripts/monitoring-operations.sh restart-component prometheus

# Fix Prometheus configuration
kubectl apply -f deployments/monitoring/prometheus-config.yaml

# Verify service labels match ServiceMonitor selector
kubectl get services -n nephoran-system --show-labels
```

#### 2. Grafana Dashboards Not Loading

**Symptoms**:
- Blank dashboards
- "Dashboard not found" errors

**Diagnosis**:
```bash
# Check Grafana logs
kubectl logs -n nephoran-monitoring deployment/grafana

# Verify dashboard configmap
kubectl get configmap nephoran-production-dashboards -n nephoran-system -o yaml
```

**Solutions**:
```bash
# Restart Grafana
./scripts/monitoring-operations.sh restart-component grafana

# Reimport dashboards
kubectl delete configmap nephoran-production-dashboards -n nephoran-system
kubectl apply -f deployments/monitoring/production-grafana-dashboards.yaml
```

#### 3. Jaeger Traces Missing

**Symptoms**:
- No traces appearing in Jaeger UI
- Incomplete trace data

**Diagnosis**:
```bash
# Check Jaeger collector
kubectl logs -n nephoran-monitoring deployment/jaeger | grep collector

# Verify OpenTelemetry configuration
kubectl get pods -n nephoran-system -o yaml | grep -A 10 -B 10 OTEL
```

**Solutions**:
```bash
# Restart Jaeger
./scripts/monitoring-operations.sh restart-component jaeger

# Check service instrumentation
# Verify OTEL_EXPORTER_JAEGER_ENDPOINT is set correctly
```

#### 4. High Memory Usage

**Symptoms**:
- Pods being OOMKilled
- High memory alerts firing

**Diagnosis**:
```bash
# Check memory usage
kubectl top pods -n nephoran-system --sort-by=memory

# Check for memory leaks
./scripts/monitoring-operations.sh analyze-performance
```

**Solutions**:
```bash
# Increase memory limits
kubectl patch deployment llm-processor -n nephoran-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-processor","resources":{"limits":{"memory":"4Gi"}}}]}}}}'

# Scale horizontally
kubectl scale deployment/llm-processor --replicas=3 -n nephoran-system
```

### Health Check Failures

#### LLM Processor Health Check Failing

**Common Causes**:
1. OpenAI API key issues
2. Network connectivity problems
3. Resource constraints

**Resolution Steps**:
```bash
# Check service logs
kubectl logs -n nephoran-system deployment/llm-processor --tail=50

# Verify API key
kubectl get secret openai-api-key -n nephoran-system -o yaml

# Test connectivity
kubectl exec -n nephoran-system deployment/llm-processor -- curl -I https://api.openai.com
```

#### RAG API Health Check Failing

**Common Causes**:
1. Weaviate connection issues
2. Python dependencies problems
3. Configuration errors

**Resolution Steps**:
```bash
# Check RAG API logs
kubectl logs -n nephoran-system deployment/rag-api --tail=50

# Test Weaviate connection
kubectl exec -n nephoran-system deployment/rag-api -- curl http://weaviate:8080/v1/.well-known/ready

# Restart if needed
./scripts/monitoring-operations.sh restart-service rag-api
```

## Maintenance Procedures

### Monthly Maintenance

#### 1. Update Monitoring Stack
```bash
# Update Prometheus
kubectl set image deployment/prometheus prometheus=prom/prometheus:v2.46.0 -n nephoran-monitoring

# Update Grafana
kubectl set image deployment/grafana grafana=grafana/grafana:10.1.0 -n nephoran-monitoring

# Update Jaeger
kubectl set image deployment/jaeger jaeger=jaegertracing/all-in-one:1.50 -n nephoran-monitoring
```

#### 2. Clean Up Old Data
```bash
# Clean up old metrics (if using persistent storage)
# This would typically be automated by Prometheus retention policies

# Clean up old traces
# Jaeger automatically handles trace retention based on configuration
```

#### 3. Review and Update Dashboards
1. Review dashboard usage analytics
2. Update queries based on new metrics
3. Add new panels for new features
4. Remove obsolete metrics

### Quarterly Maintenance

#### 1. Performance Optimization Review
1. Analyze 3-month trends
2. Optimize slow queries
3. Review and adjust resource allocations
4. Update monitoring targets and SLAs

#### 2. Alert Tuning
1. Review alert frequency and accuracy
2. Adjust thresholds based on historical data
3. Remove or modify noisy alerts
4. Add new alerts for new components

#### 3. Documentation Updates
1. Update this runbook with new procedures
2. Document new alert types
3. Update contact information
4. Review and update escalation procedures

## Performance Tuning

### Prometheus Optimization

#### 1. Storage Optimization
```yaml
# Adjust retention policy
--storage.tsdb.retention.time=30d
--storage.tsdb.retention.size=50GB
```

#### 2. Scrape Interval Optimization
```yaml
# High-frequency metrics (every 15s)
- job_name: 'nephoran-critical'
  scrape_interval: 15s

# Normal metrics (every 30s)
- job_name: 'nephoran-standard'
  scrape_interval: 30s

# Low-frequency metrics (every 60s)
- job_name: 'nephoran-infrastructure'
  scrape_interval: 60s
```

### Grafana Optimization

#### 1. Dashboard Performance
- Use appropriate time ranges
- Limit number of series in graphs
- Use recording rules for complex queries
- Implement dashboard caching

#### 2. Query Optimization
```promql
# Use recording rules for expensive queries
rule_groups:
  - name: nephoran_performance
    rules:
    - record: nephoran:intent_success_rate
      expr: rate(nephoran_networkintent_reconciliations_total{result="success"}[5m]) / rate(nephoran_networkintent_reconciliations_total[5m])
```

### Jaeger Optimization

#### 1. Sampling Configuration
```yaml
# Probabilistic sampling (10%)
JAEGER_SAMPLER_TYPE: probabilistic
JAEGER_SAMPLER_PARAM: 0.1

# Rate limiting sampling (1000 traces/second)
JAEGER_SAMPLER_TYPE: ratelimiting
JAEGER_SAMPLER_PARAM: 1000
```

## Disaster Recovery

### Backup Procedures

#### 1. Monitoring Configuration Backup
```bash
# Backup Prometheus configuration
kubectl get configmap prometheus-config -n nephoran-monitoring -o yaml > prometheus-config-backup.yaml

# Backup Grafana dashboards
kubectl get configmap nephoran-production-dashboards -n nephoran-system -o yaml > grafana-dashboards-backup.yaml

# Backup AlertManager configuration
kubectl get configmap alertmanager-config -n nephoran-monitoring -o yaml > alertmanager-config-backup.yaml
```

#### 2. Metrics Data Backup
```bash
# If using persistent storage for Prometheus
# Create volume snapshot or backup
kubectl get pvc -n nephoran-monitoring
```

### Recovery Procedures

#### 1. Complete Stack Recovery
```bash
# Deploy monitoring stack
kubectl apply -f deployments/monitoring/complete-monitoring-stack.yaml

# Apply configurations
kubectl apply -f prometheus-config-backup.yaml
kubectl apply -f grafana-dashboards-backup.yaml
kubectl apply -f alertmanager-config-backup.yaml

# Verify recovery
./scripts/monitoring-operations.sh health-check
```

#### 2. Component-Specific Recovery
```bash
# Recover specific component
kubectl delete deployment <component> -n nephoran-monitoring
kubectl apply -f deployments/monitoring/complete-monitoring-stack.yaml

# Wait for recovery
kubectl rollout status deployment/<component> -n nephoran-monitoring
```

## Contact Information

### On-Call Escalation
1. **Level 1**: Automated alerts to on-call engineer
2. **Level 2**: After 30 minutes, escalate to engineering manager
3. **Level 3**: After 1 hour, escalate to engineering director

### External Dependencies
- **OpenAI API Support**: Use OpenAI dashboard for API issues
- **Kubernetes Support**: Contact cloud provider support for cluster issues
- **Network Issues**: Contact network team for connectivity problems

### Documentation Links
- **Architecture Documentation**: `/docs/architecture/`
- **API Documentation**: `/docs/api/`
- **Development Guide**: `/docs/development/`
- **Security Procedures**: `/docs/security/`

---

**Last Updated**: $(date)
**Version**: 1.0.0
**Maintainer**: Nephoran Operations Team
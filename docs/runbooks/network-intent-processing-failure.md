# NetworkIntent Processing Failure Recovery Runbook

## Table of Contents
1. [Overview](#overview)
2. [Severity Classification](#severity-classification)
3. [Diagnostic Procedures](#diagnostic-procedures)
4. [Common Failure Scenarios](#common-failure-scenarios)
5. [Recovery Actions](#recovery-actions)
6. [Escalation Procedures](#escalation-procedures)
7. [Post-Incident Actions](#post-incident-actions)

## Overview

This runbook provides comprehensive procedures for diagnosing and recovering from NetworkIntent processing failures in the Nephoran Intent Operator. NetworkIntent failures can range from simple LLM timeouts to complex multi-component cascading failures.

**SLA Targets:**
- P1: Recovery < 5 minutes
- P2: Recovery < 15 minutes
- P3: Recovery < 1 hour
- P4: Recovery < 4 hours

## Severity Classification

### P1 - Critical (Service Down)
- **Definition**: Complete NetworkIntent processing failure affecting all users
- **Indicators**:
  - No NetworkIntents are being processed
  - All NetworkIntent resources stuck in "Processing" state for > 5 minutes
  - Controller not responding to new NetworkIntent creates
  - Critical component completely unavailable (LLM Processor, RAG API)

**Example Alert:**
```yaml
Alert: NetworkIntentProcessingDown
Severity: Critical
Description: No NetworkIntent processed in last 5 minutes
```

### P2 - High (Degraded Service)
- **Definition**: Significant reduction in processing capability or high error rate
- **Indicators**:
  - Error rate > 25% for NetworkIntent processing
  - Processing time > 5x baseline (>10 seconds)
  - Partial component failures affecting >50% of requests
  - Memory or CPU usage > 90%

**Example Alert:**
```yaml
Alert: NetworkIntentHighErrorRate
Severity: High
Description: NetworkIntent error rate 25% over last 10 minutes
```

### P3 - Medium (Limited Impact)
- **Definition**: Intermittent failures or performance degradation
- **Indicators**:
  - Error rate 10-25% for NetworkIntent processing
  - Processing time 2-5x baseline (2-10 seconds)
  - Occasional component timeouts
  - Resource usage 70-90%

### P4 - Low (Minimal Impact)
- **Definition**: Minor issues not affecting core functionality
- **Indicators**:
  - Error rate < 10%
  - Processing time 1.5-2x baseline
  - Non-critical feature failures
  - Resource usage 50-70%

## Diagnostic Procedures

### Step 1: Initial Triage (30 seconds)

```bash
# Check NetworkIntent Controller status
kubectl get pods -n nephoran-system -l app=nephoran-controller
kubectl logs -n nephoran-system -l app=nephoran-controller --tail=50

# Check current NetworkIntent status
kubectl get networkintents -A -o wide
```

**Expected Output:**
```
NAME                           READY   STATUS    AGE
nephoran-controller-xyz123     1/1     Running   2d

NAME              INTENT                           STATUS        AGE
test-intent-1     Deploy high-availability AMF     Processing    5m
test-intent-2     Scale UPF to 3 replicas         Completed     2m
```

### Step 2: Component Health Check (60 seconds)

```bash
# Check all Nephoran components
kubectl get pods -n nephoran-system --show-labels
kubectl get services -n nephoran-system

# Check component readiness
kubectl get pods -n nephoran-system -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\t"}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}'
```

**Expected Output:**
```
nephoran-controller-abc123     Running    True
llm-processor-def456          Running    True
rag-api-ghi789               Running    True
weaviate-jkl012              Running    True
```

### Step 3: NetworkIntent Status Analysis (90 seconds)

```bash
# Get detailed NetworkIntent status
kubectl get networkintents -A -o yaml | grep -A 10 -B 5 "status:"

# Check specific NetworkIntent details
kubectl describe networkintent <FAILED_INTENT_NAME> -n <NAMESPACE>

# Check NetworkIntent events
kubectl get events -n <NAMESPACE> --field-selector involvedObject.name=<INTENT_NAME>
```

### Step 4: Controller Logs Analysis (2 minutes)

```bash
# Get controller logs with error filtering
kubectl logs -n nephoran-system -l app=nephoran-controller --since=10m | grep -E "(ERROR|FATAL|panic)"

# Get structured logs for specific NetworkIntent
kubectl logs -n nephoran-system -l app=nephoran-controller --since=10m | grep "<INTENT_NAME>"

# Check controller metrics
kubectl port-forward -n nephoran-system svc/nephoran-controller-metrics 8080:8080 &
curl http://localhost:8080/metrics | grep networkintent
```

**Key Metrics to Check:**
- `networkintent_processing_duration_seconds`
- `networkintent_processing_errors_total`
- `networkintent_active_count`

## Common Failure Scenarios

### Scenario 1: LLM Timeout Failures

**Symptoms:**
- NetworkIntents stuck in "Processing" state
- Log entries: "LLM request timeout" or "context deadline exceeded"
- LLM Processor service responding slowly

**Diagnostic Commands:**
```bash
# Check LLM Processor status and logs
kubectl logs -n nephoran-system -l app=llm-processor --tail=100
kubectl get pods -n nephoran-system -l app=llm-processor -o wide

# Check LLM Processor metrics
kubectl port-forward -n nephoran-system svc/llm-processor 8081:8080 &
curl http://localhost:8081/metrics | grep -E "(llm_request_duration|llm_timeout)"

# Test LLM Processor directly
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=llm-processor -o jsonpath='{.items[0].metadata.name}') -- curl -X POST http://localhost:8080/health
```

**Recovery Actions:**
```bash
# 1. Restart LLM Processor if unhealthy
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout status deployment/llm-processor -n nephoran-system

# 2. Verify LLM API configuration
kubectl get configmap llm-processor-config -n nephoran-system -o yaml

# 3. Test LLM connectivity
kubectl run llm-test --rm -i --tty --image=curlimages/curl -- curl -X POST "https://api.openai.com/v1/chat/completions" \
  -H "Authorization: Bearer $(kubectl get secret llm-secrets -n nephoran-system -o jsonpath='{.data.openai-api-key}' | base64 -d)" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"test"}],"max_tokens":10}'

# 4. Requeue failed NetworkIntents
kubectl patch networkintent <INTENT_NAME> -n <NAMESPACE> --type='merge' -p='{"metadata":{"annotations":{"nephoran.com/requeue":"true"}}}'
```

### Scenario 2: RAG Service Unavailable

**Symptoms:**
- NetworkIntents failing with "RAG service unavailable"
- Context retrieval failures
- Weaviate connection errors

**Diagnostic Commands:**
```bash
# Check RAG API status
kubectl logs -n nephoran-system -l app=rag-api --tail=100
kubectl get pods -n nephoran-system -l app=rag-api -o wide

# Check Weaviate status
kubectl logs -n nephoran-system -l app=weaviate --tail=50
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=weaviate -o jsonpath='{.items[0].metadata.name}') -- wget -qO- http://localhost:8080/v1/meta

# Test RAG API connectivity
kubectl port-forward -n nephoran-system svc/rag-api 8082:8080 &
curl -X GET http://localhost:8082/health
curl -X POST http://localhost:8082/v1/query -H "Content-Type: application/json" -d '{"query":"test query","top_k":3}'
```

**Recovery Actions:**
```bash
# 1. Restart RAG API if unhealthy
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout status deployment/rag-api -n nephoran-system

# 2. Check and restart Weaviate if needed
kubectl rollout restart statefulset/weaviate -n nephoran-system
kubectl rollout status statefulset/weaviate -n nephoran-system

# 3. Verify Weaviate schema
kubectl exec -n nephoran-system -it $(kubectl get pods -n nephoran-system -l app=weaviate -o jsonpath='{.items[0].metadata.name}') -- \
  wget -qO- http://localhost:8080/v1/schema | jq '.classes[] | .class'

# 4. Test vector database connectivity
kubectl run weaviate-test --rm -i --tty --image=curlimages/curl -- \
  curl -X GET "http://weaviate.nephoran-system.svc.cluster.local:8080/v1/meta"
```

### Scenario 3: Nephio Integration Errors

**Symptoms:**
- NetworkIntents processed but no packages generated
- GitOps repository not updated
- Nephio Porch connection failures

**Diagnostic Commands:**
```bash
# Check Nephio Bridge status
kubectl logs -n nephoran-system -l app=nephio-bridge --tail=100
kubectl get pods -n nephoran-system -l app=nephio-bridge -o wide

# Check Nephio Porch connectivity
kubectl get packagerepositories -A
kubectl get packagerevisions -A

# Check GitOps repository access
kubectl get secrets -n nephoran-system git-credentials -o yaml
kubectl logs -n nephoran-system -l app=nephio-bridge | grep -i "git\|porch\|package"
```

**Recovery Actions:**
```bash
# 1. Restart Nephio Bridge
kubectl rollout restart deployment/nephio-bridge -n nephoran-system
kubectl rollout status deployment/nephio-bridge -n nephoran-system

# 2. Verify Porch connection
kubectl port-forward -n porch-system svc/porch-server 9443:9443 &
curl -k https://localhost:9443/api/porch/v1alpha1/packagerevisions

# 3. Test GitOps repository access
kubectl run git-test --rm -i --tty --image=alpine/git -- \
  git ls-remote https://$(kubectl get secret git-credentials -n nephoran-system -o jsonpath='{.data.token}' | base64 -d)@github.com/your-org/nephio-packages.git

# 4. Force package regeneration
kubectl annotate networkintent <INTENT_NAME> -n <NAMESPACE> nephoran.com/force-regenerate=true --overwrite
```

## Recovery Actions

### Automated Recovery Procedures

**Circuit Breaker Reset:**
```bash
# Reset LLM Processor circuit breaker
kubectl exec -n nephoran-system $(kubectl get pods -n nephoran-system -l app=llm-processor -o jsonpath='{.items[0].metadata.name}') -- \
  curl -X POST http://localhost:8080/admin/circuit-breaker/reset

# Reset RAG API circuit breaker
kubectl exec -n nephoran-system $(kubectl get pods -n nephoran-system -l app=rag-api -o jsonpath='{.items[0].metadata.name}') -- \
  curl -X POST http://localhost:8080/admin/circuit-breaker/reset
```

**Cache Clearing:**
```bash
# Clear LLM response cache
kubectl exec -n nephoran-system $(kubectl get pods -n nephoran-system -l app=llm-processor -o jsonpath='{.items[0].metadata.name}') -- \
  curl -X POST http://localhost:8080/admin/cache/clear

# Clear RAG cache
kubectl exec -n nephoran-system $(kubectl get pods -n nephoran-system -l app=rag-api -o jsonpath='{.items[0].metadata.name}') -- \
  curl -X DELETE http://localhost:8080/admin/cache
```

### Manual Recovery Steps

**Step 1: Identify Stuck NetworkIntents**
```bash
# Find NetworkIntents stuck in Processing state for > 5 minutes
kubectl get networkintents -A -o json | jq -r '
.items[] | 
select(.status.phase == "Processing" and (.metadata.creationTimestamp | fromdateiso8601) < (now - 300)) |
"\(.metadata.namespace)/\(.metadata.name) - \(.status.phase) - \(.metadata.creationTimestamp)"'
```

**Step 2: Force Requeue Stuck Intents**
```bash
# Requeue stuck NetworkIntents
for intent in $(kubectl get networkintents -A -o json | jq -r '.items[] | select(.status.phase == "Processing") | "\(.metadata.namespace)/\(.metadata.name)"'); do
  IFS='/' read -r namespace name <<< "$intent"
  kubectl patch networkintent "$name" -n "$namespace" --type='merge' -p='{"metadata":{"annotations":{"nephoran.com/requeue":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}}}'
done
```

**Step 3: Component Health Recovery**
```bash
# Full system restart (use with caution)
kubectl rollout restart deployment/nephoran-controller -n nephoran-system
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout restart deployment/nephio-bridge -n nephoran-system

# Wait for all deployments to be ready
kubectl wait --for=condition=available --timeout=300s deployment --all -n nephoran-system
```

**Step 4: Verification**
```bash
# Verify recovery by creating test NetworkIntent
cat <<EOF | kubectl apply -f -
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: recovery-test-$(date +%s)
  namespace: default
spec:
  intent: "Deploy a simple AMF instance for testing recovery"
  priority: high
  scope: cluster
EOF

# Monitor test intent processing
kubectl get networkintents recovery-test-* -w
```

## Escalation Procedures

### Level 1: Automated Recovery (0-5 minutes)
- **Trigger**: Monitoring alerts
- **Actions**: Automated restart procedures, circuit breaker resets
- **Success Criteria**: Service restored within 5 minutes
- **Escalation**: If not resolved, escalate to Level 2

### Level 2: On-Call Engineer (5-15 minutes)
- **Trigger**: Level 1 failure or P1/P2 alerts
- **Contact**: Primary on-call engineer
- **Actions**: Manual diagnostic procedures, component restarts, configuration fixes
- **Success Criteria**: Root cause identified and service restored
- **Escalation**: If not resolved in 15 minutes for P1 or 1 hour for P2, escalate to Level 3

### Level 3: Subject Matter Expert (15+ minutes for P1)
- **Trigger**: Level 2 escalation
- **Contact**: Team lead or senior engineer
- **Actions**: Deep system analysis, emergency hotfixes, architectural decisions
- **Authority**: Can approve emergency deployment bypassing normal procedures

### Level 4: Vendor/External Support
- **Trigger**: Infrastructure or external dependency failures
- **Contact**: Cloud provider support, vendor support teams
- **Actions**: Infrastructure investigation, external service troubleshooting

### Emergency Contacts

```yaml
Primary On-Call:
  - Slack: #nephoran-ops-alerts
  - PagerDuty: +1-xxx-xxx-xxxx
  - Email: oncall-nephoran@company.com

Team Lead:
  - Phone: +1-xxx-xxx-xxxx
  - Email: nephoran-lead@company.com

Infrastructure Team:
  - Slack: #infrastructure-alerts
  - Phone: +1-xxx-xxx-xxxx
```

## Post-Incident Actions

### Immediate Actions (within 1 hour of resolution)

1. **Update Status Page**
   ```bash
   curl -X PATCH "https://api.statuspage.io/v1/pages/{page_id}/incidents/{incident_id}" \
     -H "Authorization: OAuth {token}" \
     -H "Content-Type: application/json" \
     -d '{"incident": {"status": "resolved", "message": "NetworkIntent processing fully restored"}}'
   ```

2. **Document Timeline**
   - Incident start time
   - Detection time
   - Response time
   - Resolution time
   - Key actions taken

3. **Verify System Health**
   ```bash
   # Run comprehensive health check
   ./scripts/health-check-comprehensive.sh
   
   # Verify all NetworkIntents processing
   kubectl get networkintents -A --sort-by=.metadata.creationTimestamp
   ```

### Follow-up Actions (within 24 hours)

1. **Create Incident Report**
   - Root cause analysis
   - Timeline of events
   - Impact assessment
   - Recovery actions effectiveness

2. **Review Monitoring and Alerting**
   ```bash
   # Check if alerts fired appropriately
   curl -X GET "http://prometheus.monitoring.svc.cluster.local:9090/api/v1/query" \
     --data-urlencode 'query=ALERTS{alertname=~"NetworkIntent.*"}[1h]'
   ```

3. **Update Runbooks**
   - Document any new recovery procedures
   - Update escalation contacts if needed
   - Add new diagnostic commands discovered

### Long-term Actions (within 1 week)

1. **Implement Preventive Measures**
   - Enhanced monitoring
   - Improved error handling
   - Additional redundancy

2. **Conduct Post-Incident Review**
   - Team retrospective meeting
   - Process improvement identification
   - Training needs assessment

3. **Update Documentation**
   - Architecture diagrams
   - Operational procedures
   - Emergency response contacts

### Metrics and KPIs

Track the following metrics for continuous improvement:

```yaml
Incident Response Metrics:
  - Mean Time to Detect (MTTD): Target < 2 minutes
  - Mean Time to Respond (MTTR): Target < 5 minutes for P1
  - Mean Time to Resolve (MTTR): Target < 15 minutes for P1
  - False Positive Rate: Target < 5%

System Health Metrics:
  - NetworkIntent Success Rate: Target > 99%
  - Processing Time P95: Target < 5 seconds
  - Component Availability: Target > 99.9%
  - API Error Rate: Target < 0.1%
```

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-07  
**Next Review**: 2025-02-07  
**Owner**: Nephoran Operations Team
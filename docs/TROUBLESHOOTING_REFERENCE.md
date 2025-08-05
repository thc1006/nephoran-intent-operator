# Nephoran Intent Operator - Troubleshooting Reference

## Overview

This comprehensive troubleshooting guide provides solutions for common issues, diagnostic procedures, and maintenance recommendations for the Nephoran Intent Operator system. Use this reference to quickly identify and resolve problems across all components.

## Quick Diagnostic Commands

### System Health Check
```bash
# Run comprehensive health diagnostics
./diagnose_cluster.sh

# Validate development environment
./validate-environment.ps1

# Check all Nephoran services
kubectl get pods -l app.kubernetes.io/part-of=nephoran

# Service-specific health checks
curl http://localhost:8080/healthz  # LLM Processor (port-forward required)
curl http://localhost:5001/readyz   # RAG API (port-forward required)
```

### Log Analysis
```bash
# Check all service logs
kubectl logs -f deployment/llm-processor
kubectl logs -f deployment/nephio-bridge  
kubectl logs -f deployment/oran-adaptor
kubectl logs -f deployment/rag-api

# Search for errors across all pods
kubectl logs -l app.kubernetes.io/part-of=nephoran --tail=100 | grep -i "error\|failed\|timeout"

# Follow logs with timestamps
kubectl logs -f deployment/llm-processor --timestamps=true
```

## Component-Specific Troubleshooting

### 1. LLM Processor Service Issues

#### Problem: LLM Processor Not Starting
**Symptoms**:
- Pod in `CrashLoopBackOff` state
- HTTP 503 responses on health checks
- Startup errors in logs

**Diagnostic Commands**:
```bash
# Check pod status and events
kubectl describe pod -l app=llm-processor

# Examine startup logs
kubectl logs deployment/llm-processor --previous

# Verify configuration
kubectl get configmap llm-processor-config -o yaml
kubectl get secret llm-processor-secrets -o yaml
```

**Common Causes and Solutions**:

1. **Missing OpenAI API Key**:
   ```bash
   # Verify secret exists
   kubectl get secret llm-processor-secrets
   
   # Create missing secret
   kubectl create secret generic llm-processor-secrets \
     --from-literal=openai-api-key="your-api-key-here"
   
   # Restart deployment
   kubectl rollout restart deployment/llm-processor
   ```

2. **RAG API Connection Failure**:
   ```bash
   # Test RAG API connectivity
   kubectl exec -it deployment/llm-processor -- curl http://rag-api:5001/healthz
   
   # Check service and endpoints
   kubectl get service rag-api
   kubectl get endpoints rag-api
   
   # Verify network policies
   kubectl get networkpolicies
   ```

3. **Memory/CPU Resource Constraints**:
   ```bash
   # Check resource usage
   kubectl top pod -l app=llm-processor
   
   # Increase resource limits
   kubectl patch deployment llm-processor -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "resources": {
                 "limits": {
                   "cpu": "2000m",
                   "memory": "4Gi"
                 }
               }
             }
           ]
         }
       }
     }
   }'
   ```

#### Problem: High Latency or Timeouts
**Symptoms**:
- Slow response times (>30 seconds)
- Timeout errors in application logs
- Circuit breaker opening frequently

**Diagnostic Commands**:
```bash
# Check circuit breaker status
kubectl port-forward svc/llm-processor 8080:8080 &
curl http://localhost:8080/circuit-breaker/status

# Monitor performance metrics
kubectl port-forward svc/prometheus 9090:9090 &
# Access Prometheus at http://localhost:9090
```

**Solutions**:

1. **Reset Circuit Breaker**:
   ```bash
   curl -X POST http://localhost:8080/circuit-breaker/control \
     -H "Content-Type: application/json" \
     -d '{"action": "reset", "circuit_name": "llm-processor"}'
   ```

2. **Optimize Cache Settings**:
   ```bash
   # Check cache statistics
   curl http://localhost:8080/cache/stats
   
   # Clear cache if needed
   curl -X DELETE http://localhost:8080/cache/clear
   ```

3. **Scale Horizontally**:
   ```bash
   # Increase replica count
   kubectl scale deployment llm-processor --replicas=3
   
   # Verify scaling
   kubectl get pods -l app=llm-processor
   ```

### 2. RAG API Service Issues

#### Problem: Vector Database Connection Failures
**Symptoms**:
- RAG API returning 503 errors
- "Weaviate connection failed" in logs
- Knowledge base queries failing

**Diagnostic Commands**:
```bash
# Check Weaviate status
kubectl get pods -l app=weaviate
kubectl logs -f deployment/weaviate

# Test Weaviate connectivity
kubectl port-forward svc/weaviate 8080:8080 &
curl http://localhost:8080/v1/meta

# Check RAG API connectivity to Weaviate
kubectl exec -it deployment/rag-api -- curl http://weaviate:8080/v1/meta
```

**Solutions**:

1. **Restart Weaviate**:
   ```bash
   kubectl rollout restart deployment/weaviate
   
   # Wait for readiness
   kubectl wait --for=condition=ready pod -l app=weaviate --timeout=300s
   ```

2. **Check Storage Issues**:
   ```bash
   # Verify PVC status
   kubectl get pvc | grep weaviate
   kubectl describe pvc weaviate-data
   
   # Check storage class
   kubectl get storageclass
   ```

3. **Verify Network Connectivity**:
   ```bash
   # Test with debug pod
   kubectl run debug --image=curlimages/curl --rm -it --restart=Never -- sh
   # From inside pod: curl http://weaviate:8080/v1/meta
   ```

#### Problem: Knowledge Base Empty or Outdated
**Symptoms**:
- RAG queries returning no results
- Low confidence scores in responses
- Missing telecom documentation

**Diagnostic Commands**:
```bash
# Check knowledge base statistics
kubectl port-forward svc/rag-api 5001:5001 &
curl http://localhost:5001/knowledge/stats

# Verify document count
curl http://localhost:5001/v1/query \
  -H "Content-Type: application/json" \
  -d '{"query": "5G AMF", "limit": 1}'
```

**Solutions**:

1. **Repopulate Knowledge Base**:
   ```bash
   # Run knowledge base population script
   ./populate-knowledge-base.ps1
   
   # Or use make target
   make populate-kb-enhanced
   ```

2. **Upload Specific Documents**:
   ```bash
   # Upload individual documents
   curl -X POST http://localhost:5001/knowledge/upload \
     -F "file=@3gpp-ts-23501.pdf" \
     -F "metadata={\"category\":\"5G Core\",\"source\":\"3GPP\"}"
   ```

3. **Verify Document Processing**:
   ```bash
   # Check processing logs
   kubectl logs deployment/rag-api | grep -i "document\|processing\|embedding"
   ```

### 3. Kubernetes Controller Issues

#### Problem: NetworkIntent Not Processing
**Symptoms**:
- NetworkIntent stuck in "Pending" phase
- No controller logs for specific intents
- Status conditions not updating

**Diagnostic Commands**:
```bash
# Check specific NetworkIntent status
kubectl describe networkintent <intent-name>

# Verify controller is running
kubectl get pods -l app=nephio-bridge
kubectl logs -f deployment/nephio-bridge

# Check RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:default:nephio-bridge
```

**Solutions**:

1. **Restart Controller**:
   ```bash
   kubectl rollout restart deployment/nephio-bridge
   
   # Verify restart
   kubectl get pods -l app=nephio-bridge
   ```

2. **Fix RBAC Issues**:
   ```bash
   # Reapply RBAC configuration
   kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml
   
   # Verify permissions
   kubectl auth can-i create configmaps --as=system:serviceaccount:default:nephio-bridge
   kubectl auth can-i patch networkintents/status --as=system:serviceaccount:default:nephio-bridge
   ```

3. **Check CRD Installation**:
   ```bash
   # Verify CRDs are installed
   kubectl get crd | grep nephoran
   
   # Reinstall if missing
   kubectl apply -f deployments/crds/
   ```

#### Problem: E2NodeSet Scaling Failures
**Symptoms**:
- E2NodeSet replicas not matching desired count
- ConfigMaps not being created
- Scaling operations timing out

**Diagnostic Commands**:
```bash
# Check E2NodeSet status
kubectl get e2nodesets
kubectl describe e2nodeset <nodeset-name>

# Verify ConfigMaps
kubectl get configmaps -l e2nodeset=<nodeset-name>

# Check controller logs for E2NodeSet operations
kubectl logs deployment/nephio-bridge | grep -i "e2nodeset\|configmap"
```

**Solutions**:

1. **Manual ConfigMap Creation Test**:
   ```bash
   # Test ConfigMap creation permissions
   kubectl create configmap test-cm --from-literal=test=value
   kubectl delete configmap test-cm
   ```

2. **Fix Resource Quotas**:
   ```bash
   # Check resource quotas
   kubectl get resourcequota
   kubectl describe resourcequota
   
   # Increase quotas if needed
   kubectl patch resourcequota <quota-name> -p '{"spec":{"hard":{"configmaps":"100"}}}'
   ```

3. **Scale Gradually**:
   ```bash
   # Scale in smaller increments
   kubectl patch e2nodeset <nodeset-name> --type merge -p '{"spec":{"replicas":3}}'
   
   # Wait for stabilization before scaling further
   kubectl wait --for=condition=ScalingComplete e2nodeset/<nodeset-name> --timeout=300s
   ```

### 4. Network and Connectivity Issues

#### Problem: Inter-Service Communication Failures
**Symptoms**:
- Services unable to reach each other
- DNS resolution failures
- Connection timeouts

**Diagnostic Commands**:
```bash
# Test DNS resolution
kubectl exec -it deployment/llm-processor -- nslookup rag-api

# Check service endpoints
kubectl get endpoints
kubectl describe service rag-api

# Test connectivity with debug pod
kubectl run debug --image=curlimages/curl --rm -it --restart=Never -- sh
# From inside: curl http://rag-api:5001/healthz
```

**Solutions**:

1. **Fix Service Configuration**:
   ```bash
   # Check service selectors
   kubectl describe service rag-api
   kubectl get pods -l app=rag-api --show-labels
   
   # Recreate service if selectors don't match
   kubectl delete service rag-api
   kubectl apply -f deployments/kustomize/base/rag-api/service.yaml
   ```

2. **Network Policy Issues**:
   ```bash
   # List network policies
   kubectl get networkpolicies
   
   # Temporarily remove policies for testing
   kubectl delete networkpolicy --all
   
   # Test connectivity and reapply policies
   kubectl apply -f deployments/security/security-network-policies.yaml
   ```

3. **CoreDNS Issues**:
   ```bash
   # Check CoreDNS pods
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   
   # Restart CoreDNS if needed
   kubectl rollout restart deployment/coredns -n kube-system
   ```

### 5. Performance and Resource Issues

#### Problem: High Memory Usage
**Symptoms**:
- Pods being OOMKilled
- High memory utilization in monitoring
- Slow response times

**Diagnostic Commands**:
```bash
# Check memory usage
kubectl top pods --sort-by=memory
kubectl describe node

# Check pod resource limits
kubectl describe pod <pod-name> | grep -A 5 "Limits:\|Requests:"

# Monitor memory over time
kubectl port-forward svc/prometheus 9090:9090 &
# Query: container_memory_usage_bytes{pod=~"llm-processor.*"}
```

**Solutions**:

1. **Increase Memory Limits**:
   ```bash
   # Update deployment with higher memory limits
   kubectl patch deployment llm-processor -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "resources": {
                 "limits": {
                   "memory": "8Gi"
                 },
                 "requests": {
                   "memory": "4Gi"
                 }
               }
             }
           ]
         }
       }
     }
   }'
   ```

2. **Optimize Cache Settings**:
   ```bash
   # Reduce cache size
   kubectl patch configmap llm-processor-config -p '
   {
     "data": {
       "CACHE_MAX_SIZE": "500",
       "L1_CACHE_SIZE": "100"
     }
   }'
   
   # Restart to apply changes
   kubectl rollout restart deployment/llm-processor
   ```

3. **Enable Memory Optimization**:
   ```bash
   # Trigger manual optimization
   kubectl port-forward svc/llm-processor 8080:8080 &
   curl -X POST http://localhost:8080/performance/optimize \
     -H "Content-Type: application/json" \
     -d '{"target_metrics": ["memory_usage"], "optimization_level": "aggressive"}'
   ```

#### Problem: CPU Throttling
**Symptoms**:
- High CPU utilization
- Request queuing
- Increased response times

**Diagnostic Commands**:
```bash
# Check CPU usage
kubectl top pods --sort-by=cpu
kubectl top nodes

# Monitor CPU throttling
kubectl port-forward svc/prometheus 9090:9090 &
# Query: rate(container_cpu_cfs_throttled_seconds_total[5m])
```

**Solutions**:

1. **Increase CPU Limits**:
   ```bash
   kubectl patch deployment llm-processor -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "resources": {
                 "limits": {
                   "cpu": "4000m"
                 },
                 "requests": {
                   "cpu": "2000m"
                 }
               }
             }
           ]
         }
       }
     }
   }'
   ```

2. **Horizontal Scaling**:
   ```bash
   # Scale to multiple replicas
   kubectl scale deployment llm-processor --replicas=3
   
   # Configure HPA
   kubectl apply -f deployments/kustomize/base/llm-processor/hpa.yaml
   ```

## Security and Authentication Issues

### Problem: OAuth2 Authentication Failures
**Symptoms**:
- 401 Unauthorized responses
- Token validation errors
- Redirect loop issues

**Diagnostic Commands**:
```bash
# Check OAuth2 configuration
kubectl get configmap oauth2-config -o yaml

# Verify provider connectivity
kubectl exec -it deployment/llm-processor -- curl https://login.microsoftonline.com/common/v2.0/.well-known/openid_configuration

# Check logs for auth errors
kubectl logs deployment/llm-processor | grep -i "auth\|oauth\|token"
```

**Solutions**:

1. **Verify OAuth2 Configuration**:
   ```bash
   # Update OAuth2 settings
   kubectl patch configmap oauth2-config -p '
   {
     "data": {
       "AZURE_CLIENT_ID": "your-client-id",
       "AZURE_TENANT_ID": "your-tenant-id",
       "REDIRECT_URL": "http://llm-processor:8080/auth/callback/azure"
     }
   }'
   
   kubectl rollout restart deployment/llm-processor
   ```

2. **Check Network Connectivity to Auth Providers**:
   ```bash
   # Test external connectivity
   kubectl exec -it deployment/llm-processor -- curl -I https://login.microsoftonline.com
   
   # Check egress network policies
   kubectl get networkpolicy | grep egress
   ```

3. **Validate JWT Tokens**:
   ```bash
   # Test token endpoint directly
   curl -X POST http://localhost:8080/auth/refresh \
     -H "Content-Type: application/json" \
     -d '{"refresh_token": "your-refresh-token"}'
   ```

### Problem: RBAC Permission Denied
**Symptoms**:
- 403 Forbidden errors
- "cannot create/update/delete" messages
- Controller reconciliation failures

**Diagnostic Commands**:
```bash
# Check service account permissions
kubectl auth can-i --list --as=system:serviceaccount:default:nephio-bridge

# Verify role bindings
kubectl get rolebinding,clusterrolebinding | grep nephoran

# Check specific permissions
kubectl auth can-i create networkintents --as=system:serviceaccount:default:nephio-bridge
kubectl auth can-i patch e2nodesets/status --as=system:serviceaccount:default:nephio-bridge
```

**Solutions**:

1. **Reapply RBAC Configuration**:
   ```bash
   kubectl apply -f deployments/kubernetes/nephio-bridge-rbac.yaml
   kubectl apply -f deployments/security/security-rbac.yaml
   ```

2. **Debug Specific Permission Issues**:
   ```bash
   # Check which permissions are missing
   kubectl auth can-i create configmaps --as=system:serviceaccount:default:nephio-bridge -v=8
   
   # Add missing permissions manually
   kubectl create clusterrole nephoran-additional --verb=create,update,patch --resource=configmaps
   kubectl create clusterrolebinding nephoran-additional --clusterrole=nephoran-additional --serviceaccount=default:nephio-bridge
   ```

## Monitoring and Observability Issues

### Problem: Metrics Not Appearing in Prometheus
**Symptoms**:
- Missing metrics in Prometheus UI
- ServiceMonitor not detecting targets
- Scrape failures in Prometheus

**Diagnostic Commands**:
```bash
# Check ServiceMonitor configuration
kubectl get servicemonitor -l app.kubernetes.io/part-of=nephoran

# Verify metrics endpoints
kubectl port-forward svc/llm-processor 8080:8080 &
curl http://localhost:8080/metrics

# Check Prometheus targets
kubectl port-forward svc/prometheus 9090:9090 &
# Go to http://localhost:9090/targets
```

**Solutions**:

1. **Fix ServiceMonitor Labels**:
   ```bash
   # Update ServiceMonitor with correct labels
   kubectl patch servicemonitor llm-processor -p '
   {
     "spec": {
       "selector": {
         "matchLabels": {
           "app": "llm-processor"
         }
       }
     }
   }'
   ```

2. **Verify Metrics Endpoint**:
   ```bash
   # Test metrics endpoint directly
   kubectl exec -it deployment/llm-processor -- curl localhost:8080/metrics
   
   # Check service port configuration
   kubectl describe service llm-processor
   ```

3. **Restart Prometheus**:
   ```bash
   kubectl rollout restart deployment/prometheus
   
   # Check Prometheus configuration
   kubectl get configmap prometheus-config -o yaml
   ```

### Problem: Logs Not Appearing in Centralized System
**Symptoms**:
- Missing logs in Kibana/log aggregation
- Log shipping failures
- Incomplete log entries

**Diagnostic Commands**:
```bash
# Check log shipping pods
kubectl get pods -n logging

# Verify log format
kubectl logs deployment/llm-processor --tail=10

# Check Elasticsearch connectivity
kubectl port-forward svc/elasticsearch 9200:9200 &
curl http://localhost:9200/_cluster/health
```

**Solutions**:

1. **Fix Log Format**:
   ```bash
   # Ensure structured logging is enabled
   kubectl patch deployment llm-processor -p '
   {
     "spec": {
       "template": {
         "spec": {
           "containers": [
             {
               "name": "llm-processor",
               "env": [
                 {
                   "name": "LOG_FORMAT",
                   "value": "json"
                 }
               ]
             }
           ]
         }
       }
     }
   }'
   ```

2. **Restart Log Shipping**:
   ```bash
   kubectl rollout restart daemonset/fluent-bit -n logging
   kubectl rollout restart deployment/logstash -n logging
   ```

## Disaster Recovery Procedures

### System Backup
```bash
# Backup all Nephoran resources
kubectl get all,configmaps,secrets,pvc -l app.kubernetes.io/part-of=nephoran -o yaml > nephoran-backup-$(date +%Y%m%d).yaml

# Backup Weaviate data
./deployments/weaviate/backup-validation.sh

# Backup custom resources
kubectl get networkintents,e2nodesets -A -o yaml > nephoran-crds-backup-$(date +%Y%m%d).yaml
```

### Emergency Recovery
```bash
# Emergency system validation
./diagnose_cluster.sh

# Run disaster recovery automation
./scripts/disaster-recovery.sh

# Restore from backup if needed
kubectl apply -f nephoran-backup-YYYYMMDD.yaml

# Verify system health after recovery
make validate-all
```

### Complete System Reset
```bash
# Warning: This will delete all Nephoran resources
kubectl delete all,configmaps,secrets,pvc -l app.kubernetes.io/part-of=nephoran

# Clean up CRDs (this will delete all custom resources)
kubectl delete crd networkintents.nephoran.com
kubectl delete crd e2nodesets.nephoran.com
kubectl delete crd managedelements.nephoran.com

# Redeploy system
./deploy.sh local
```

## Performance Tuning

### Optimization Checklist
```bash
# 1. Enable performance monitoring
kubectl port-forward svc/llm-processor 8080:8080 &
curl http://localhost:8080/performance/metrics

# 2. Optimize cache settings
curl -X POST http://localhost:8080/cache/warm \
  -H "Content-Type: application/json" \
  -d '{"strategy": "popular_queries", "limit": 100}'

# 3. Check resource utilization
kubectl top pods --sort-by=memory
kubectl top pods --sort-by=cpu

# 4. Run performance tests
./scripts/performance-benchmark-suite.sh

# 5. Tune JVM/Go runtime (if applicable)
kubectl patch deployment llm-processor -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "llm-processor",
            "env": [
              {
                "name": "GOGC",
                "value": "100"
              },
              {
                "name": "GOMEMLIMIT",
                "value": "3GiB"
              }
            ]
          }
        ]
      }
    }
  }
}'
```

## Support and Escalation

### Gathering Support Information
```bash
# Collect comprehensive system information
./scripts/collect-support-info.sh

# Generate health report
kubectl get pods,services,endpoints,configmaps -l app.kubernetes.io/part-of=nephoran -o wide > system-state.txt

# Export recent logs
kubectl logs deployment/llm-processor --since=1h > llm-processor-logs.txt
kubectl logs deployment/rag-api --since=1h > rag-api-logs.txt
kubectl logs deployment/nephio-bridge --since=1h > nephio-bridge-logs.txt
```

### Emergency Contacts and Procedures
1. **Check Documentation**: Review CLAUDE.md, DEVELOPER_GUIDE.md, and API_REFERENCE.md
2. **Run Diagnostics**: Execute `./diagnose_cluster.sh` and `./validate-environment.ps1`
3. **Check Recent Changes**: Review git history and recent deployments
4. **Collect Evidence**: Gather logs, metrics, and system state information
5. **Contact Support**: Include all collected information with support request

This troubleshooting reference covers the most common issues and their solutions. For complex problems not covered here, refer to the developer documentation or contact the support team with detailed diagnostic information.
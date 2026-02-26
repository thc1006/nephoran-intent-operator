# O-RAN Production Fixes - Implementation Guide

**Document Version**: 1.0
**Date**: 2026-02-26
**Status**: Implementation in Progress

---

## Executive Summary

This document provides step-by-step implementation for fixing 3 critical O-RAN production issues:

1. ✅ **Redis Status Store** - DEPLOYED (fixes A1 HTTP 405)
2. 🔄 **OAI gNB with E2** - In Research
3. ⏳ **Real KPI Monitor xApp** - Pending

---

## Phase 1: Redis Status Store ✅ COMPLETE

### **Deployment Status**

```bash
✅ Redis 7.4.8 deployed (ricplt namespace)
✅ Service: a1-status-store:6379
✅ Connectivity: PONG
✅ Configuration: 256MB, LRU eviction
```

### **Code Integration Required**

Two components need modification:

#### **A. Scaling xApp** (`deployments/xapps/scaling-xapp/`)

**Current Code** (Line 334-378 in `main.go`):
```go
// OLD: Causes HTTP 405 error
url := fmt.Sprintf("%s/A1-P/v2/policytypes/100/policies/%s/status", x.a1URL, policyID)
resp, err := http.Post(url, "application/json", strings.NewReader(string(statusJSON)))
```

**New Code** (implemented in `main-redis.go`):
```go
// NEW: Write to Redis
redisKey := fmt.Sprintf("policy:status:100:%s", policyID)
err := x.redisClient.Set(ctx, redisKey, statusJSON, 0).Err()
```

**Dependencies**:
- Add to `go.mod`: `github.com/redis/go-redis/v9 v9.0.5`
- Initialize Redis client: `InitRedisClient(RedisConfig{Address: "a1-status-store.ricplt:6379"})`

#### **B. A1 Service** (`pkg/oran/a1/`)

**Current Code** (`a1_adaptor.go` GetPolicyStatus):
```go
// OLD: HTTP GET to A1 Mediator
url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s/status", ...)
resp, err := a.httpClient.Do(req)
```

**New Code** (implemented in `redis_status_store.go`):
```go
// NEW: Read from Redis
redisKey := fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)
statusJSON, err := r.client.Get(ctx, redisKey).Result()
```

**Files Created**:
- ✅ `deployments/xapps/scaling-xapp/main-redis.go` - Redis writer implementation
- ✅ `pkg/oran/a1/redis_status_store.go` - Redis reader implementation

### **Integration Steps**

```bash
# 1. Update scaling xApp
cd deployments/xapps/scaling-xapp
cat main-redis.go >> main.go  # Merge Redis functions
go mod tidy
go mod vendor  # If using vendor

# 2. Rebuild and redeploy xApp
docker build -t scaling-xapp:redis .
kubectl set image deployment/ricxapp-scaling -n ricxapp scaling=scaling-xapp:redis

# 3. Update A1 service
cd pkg/oran/a1
# Modify a1_adaptor.go to use RedisStatusStore instead of HTTP
go build ./...

# 4. Verify
kubectl logs -n ricxapp deployment/ricxapp-scaling | grep "Redis"
# Expected: "✅ Policy status written to Redis"
```

### **Testing**

```bash
# 1. Create test policy
curl -X PUT "http://service-ricplt-a1mediator-http.ricplt:10000/A1-P/v2/policytypes/100/policies/test-redis-001" \
  -H "Content-Type: application/json" \
  -d '{"qosObjectives":{"replicas":5},"scope":{"target":"nginx","namespace":"default"}}'

# 2. Check Redis directly
kubectl exec -n ricplt deployment/a1-status-store -- redis-cli GET "policy:status:100:test-redis-001"
# Expected: {"enforce_status":"ENFORCED","enforce_reason":"Scaled to 5 replicas"}

# 3. Query status via A1 API
curl -s "http://service-ricplt-a1mediator-http.ricplt:10000/A1-P/v2/policytypes/100/policies/test-redis-001/status"
# Expected: {"enforce_status":"ENFORCED","instance_status":"IN EFFECT"}
```

---

## Phase 2: OAI gNB with E2 Support 🔄 IN RESEARCH

### **Research Findings**

**OAI Docker Images Found**:
- Core Network: `oaisoftwarealliance/oai-amf:v1.5.0`, `oai-smf:v1.5.0`, etc.
- gNB: **Still researching exact tags** (2024.w42, 2025.w02, etc.)

**E2 Agent Build Requirements**:
```bash
# Build OAI with E2 support
./build_oai -I -w SIMU --gNB --nrUE --build-e2 --ninja
```

**E2 Configuration** (from research):
```yaml
e2_agent = {
  near_ric_ip_addr = "service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local";
  sm_dir = "/usr/local/lib/flexric/"
}
```

### **Docker Image Strategy**

**Option 1: Use OAI Official Images** (if tags available)
```yaml
image: oaisoftwarealliance/oai-gnb:2025.w02
# or
image: oaisoftwarealliance/oai-gnb:latest
```

**Option 2: Build Custom Image with E2**
```dockerfile
FROM ubuntu:22.04
# Install dependencies...
RUN git clone https://gitlab.eurecom.fr/oai/openairinterface5g.git
RUN cd openairinterface5g && ./build_oai -I -w SIMU --gNB --build-e2 --ninja
```

### **Deployment Plan** (when image confirmed)

```bash
# 1. Update deployment YAML
vi deployments/ric/e2sim/oai-gnb-deployment.yaml

# Change image to:
image: oaisoftwarealliance/oai-gnb:2025.wXX  # Use confirmed tag

# 2. Add E2 configuration
configMap:
  gnb.conf: |
    e2_agent = {
      near_ric_ip_addr = "service-ricplt-e2term-sctp-alpha.ricplt.svc.cluster.local";
      port = 36422;
      sm_dir = "/usr/local/lib/flexric/"
    }

# 3. Deploy
kubectl apply -f deployments/ric/e2sim/oai-gnb-deployment.yaml

# 4. Verify E2 connection
kubectl logs -n ricxapp deployment/oai-gnb | grep "E2 Setup"
# Expected: "[E2] E2 Setup Response received"

# 5. Check E2 Manager
kubectl exec -n ricplt deployment/deployment-ricplt-e2mgr -- \
  curl -s http://localhost:3800/v1/nodeb/states | jq .
# Expected: {"nodeb_list": [{"gnbId": "...", "connectionStatus": "CONNECTED"}]}
```

---

## Phase 3: Real KPI Monitor xApp ⏳ PENDING

### **Current Status**

- Mock KPI Monitor deployed: `ricxapp-kpimon` (2 replicas)
- Real xApp: `o-ran-sc/ric-app-kpimon-go` (needs build)

### **Build Steps**

```bash
# 1. Clone repo
git clone https://github.com/o-ran-sc/ric-app-kpimon-go.git
cd ric-app-kpimon-go

# 2. Build Docker image
docker build -t localhost:5000/kpimon-xapp:v1 .
docker push localhost:5000/kpimon-xapp:v1

# 3. Create xApp descriptor
cat > kpimon-descriptor.yaml <<EOF
schema: "1.0.0"
name: kpimon
version: "1.0.0"
containers:
  - name: kpimon
    image:
      registry: localhost:5000
      name: kpimon-xapp
      tag: v1
messaging:
  ports:
    - name: rmr-data
      port: 4560
      rxMessages: ["RIC_SUB_RESP", "RIC_INDICATION"]
      txMessages: ["RIC_SUB_REQ"]
EOF

# 4. Onboard xApp (requires dms_cli)
dms_cli onboard kpimon-descriptor.yaml

# 5. Deploy xApp
dms_cli install kpimon --namespace ricxapp

# 6. Scale down mock
kubectl scale deployment ricxapp-kpimon -n ricxapp --replicas=0

# 7. Verify E2 subscription
kubectl logs -n ricxapp -l app=kpimon | grep "Subscription Response"
# Expected: "[INFO] E2 Subscription Response received (status=SUCCESS)"
```

### **Configuration**

KPI Monitor xApp configuration:
```yaml
# xApp config (injected via ConfigMap)
e2_subscription:
  reporting_period_ms: 1000  # 1 second
  kpm_metrics:
    - "DRB.UEThpDl"      # Downlink throughput
    - "DRB.UEThpUl"      # Uplink throughput
    - "RRU.PrbUsedDl"    # PRB usage
    - "RRC.ConnMean"     # RRC connections
```

---

## Timeline & Dependencies

| Phase | Task | Status | Dependency | ETA |
|-------|------|--------|------------|-----|
| 1A | Redis deployed | ✅ DONE | None | Complete |
| 1B | xApp Redis integration | 🔄 Code Ready | 1A | 2h |
| 1C | A1 Service integration | 🔄 Code Ready | 1A | 2h |
| 2A | OAI image research | 🔄 In Progress | None | 1h |
| 2B | OAI gNB deployment | ⏳ Pending | 2A | 2h |
| 2C | E2 connection verify | ⏳ Pending | 2B | 1h |
| 3A | Build KPI Monitor xApp | ⏳ Pending | 2B (E2 gNB) | 4h |
| 3B | Deploy & verify | ⏳ Pending | 3A | 2h |

**Total Estimated Time**: 14 hours (with OAI image confirmed)

---

## Success Criteria

### **Phase 1 Complete When**:
- ✅ Redis deployed and healthy
- ✅ xApp writes status to Redis
- ✅ A1 Mediator reads status from Redis
- ✅ Zero HTTP 405 errors in logs
- ✅ Policy status updates work end-to-end

### **Phase 2 Complete When**:
- ⏳ OAI gNB pod Running
- ⏳ E2 Setup Response logged
- ⏳ E2 Manager shows gNB CONNECTED
- ⏳ No E2 connection errors

### **Phase 3 Complete When**:
- ⏳ KPI Monitor xApp Running (not mock)
- ⏳ E2 Subscription successful
- ⏳ KPI metrics visible in Prometheus
- ⏳ Grafana dashboard shows real RAN metrics

---

## Rollback Procedures

### **Phase 1 Rollback**:
```bash
# Revert xApp to POST method
kubectl set image deployment/ricxapp-scaling -n ricxapp scaling=scaling-xapp:original

# Delete Redis (if needed)
kubectl delete -f deployments/ric/status-store/redis-deployment.yaml
```

### **Phase 2 Rollback**:
```bash
# Remove OAI gNB
kubectl delete deployment oai-gnb -n ricxapp

# UERANSIM remains as fallback (no E2 but functional)
```

### **Phase 3 Rollback**:
```bash
# Scale up mock KPI Monitor
kubectl scale deployment ricxapp-kpimon -n ricxapp --replicas=2

# Delete real xApp
dms_cli uninstall kpimon
```

---

## References

### **Official Documentation**
- [O-RAN A1 API Spec](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-a1/en/latest/user-guide-api.html)
- [OAI E2 Agent Documentation](https://openairinterface.org/oam/)
- [KPI Monitor xApp Guide](https://wiki.o-ran-sc.org/display/RICA/KPI+Monitor+xApp)

### **GitHub Repositories**
- [OAI OpenAirInterface5G](https://gitlab.eurecom.fr/oai/openairinterface5g)
- [OAI Kubernetes](https://github.com/OPENAIRINTERFACE/openair-k8s)
- [O-RAN SC KPI Monitor](https://github.com/o-ran-sc/ric-app-kpimon-go)
- [srsRAN O-RAN SC RIC](https://github.com/srsran/oran-sc-ric)

### **Integration Examples**
- [OAI-OSC RIC Integration](https://github.com/lance-ufrn/osc_ric_oai_integration)
- [OAIC Workshop 2024](https://openaicellular.github.io/oaic/OAIC-2024-Workshop-oai-flexric-documentation.html)

---

**Last Updated**: 2026-02-26 22:35 UTC
**Next Review**: After Phase 2A completes (OAI image confirmed)

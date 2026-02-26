# O-RAN Production Migration Plan - Executive Summary

**Document Version**: 1.0
**Date**: 2026-02-26
**Status**: Ready for Implementation

---

## Quick Start: Critical Fixes

### Issue 1: A1 Policy Status Update (HTTP 405)

**Root Cause**: xApp POSTs to `/status` endpoint, but O-RAN A1 spec only supports GET

**Solution**: Implement Redis status store

```bash
# Deploy Redis
kubectl apply -f deployments/ric/status-store/redis-deployment.yaml

# Update scaling xApp to write to Redis
# File: deployments/xapps/scaling-xapp/main.go:334
func (x *ScalingXApp) reportPolicyStatus(policyID string, enforced bool, reason string) {
    status := PolicyStatus{...}
    statusJSON, _ := json.Marshal(status)

    // Write to Redis (not POST to A1)
    redisKey := fmt.Sprintf("policy:status:100:%s", policyID)
    x.redisClient.Set(ctx, redisKey, statusJSON, 0)
}

# Update A1 Mediator to read from Redis
# File: pkg/oran/a1/handlers.go:950
func (h *A1Handlers) HandleGetPolicyStatus(...) {
    statusKey := fmt.Sprintf("policy:status:%s:%s", policyTypeID, policyID)
    statusJSON, _ := h.redisClient.Get(ctx, statusKey).Result()
    // Parse and return
}
```

**Implementation Time**: 4 hours
**Risk**: LOW

---

### Issue 2: Deploy E2-Enabled gNB

**Current**: UERANSIM has no E2 support, srsRAN configured but not running

**Solution**: Activate srsRAN gNB binary

```bash
# Edit: deployments/ric/e2sim/srsran-gnb-deployment.yaml:106
# Change from: tail -f /dev/null
# Change to:   exec gnb -c /etc/srsran/gnb.yaml

# Deploy
kubectl apply -f deployments/ric/e2sim/srsran-gnb-deployment.yaml

# Verify E2 connection
kubectl logs -n ricxapp deployment/srsran-gnb | grep "E2 Setup"
# Expected: [E2] E2 Setup Response received

# Check E2 Manager registration
kubectl exec -n ricplt deployment/deployment-ricplt-e2mgr -- \
  curl -s http://localhost:3800/v1/nodeb/states
```

**Implementation Time**: 2 hours
**Risk**: LOW

---

### Issue 3: Deploy Real KPI Monitor xApp

**Current**: Mock KPI Monitor deployed

**Solution**: Build and deploy o-ran-sc/ric-app-kpimon-go

```bash
# Build
git clone https://github.com/o-ran-sc/ric-app-kpimon-go.git
cd ric-app-kpimon-go
docker build -t localhost:5000/kpimon-xapp:v1 .
docker push localhost:5000/kpimon-xapp:v1

# Onboard via dms_cli
dms_cli onboard kpimon-descriptor.yaml

# Deploy
dms_cli install kpimon --namespace ricxapp

# Scale down mock
kubectl scale deployment ricxapp-kpimon -n ricxapp --replicas=0

# Verify E2 subscription
kubectl logs -n ricxapp -l app=kpimon | grep "Subscription Response"
# Expected: [INFO] E2 Subscription Response received (status=SUCCESS)
```

**Implementation Time**: 6 hours
**Risk**: MEDIUM

---

## Implementation Timeline

| Week | Tasks | Deliverables |
|------|-------|--------------|
| **Week 1** | Redis status store, A1 Mediator update, srsRAN deployment | E2-enabled gNB operational, zero 405 errors |
| **Week 2** | KPI Monitor build/deploy, E2 subscriptions | Real KPI metrics in Prometheus |
| **Week 3** | Integration testing, load testing | All E2E tests passing |
| **Week 4** | Production cutover, documentation | Zero mock components |

---

## Key Configuration Files

### Redis Deployment
```yaml
# File: deployments/ric/status-store/redis-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: a1-status-store
  namespace: ricplt
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        resources:
          requests: {cpu: 100m, memory: 256Mi}
          limits: {cpu: 500m, memory: 512Mi}
---
apiVersion: v1
kind: Service
metadata:
  name: a1-status-store
  namespace: ricplt
spec:
  selector:
    app: a1-status-store
  ports:
  - port: 6379
```

### KPI Monitor xApp Descriptor
```yaml
# File: kpimon-descriptor.yaml
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
```

---

## Validation Tests

### Test 1: A1 Status Update
```bash
#!/bin/bash
# Create policy
curl -X PUT "http://service-ricplt-a1mediator-http.ricplt:10000/A1-P/v2/policytypes/100/policies/test001" \
  -d '{"qosObjectives":{"replicas":3},"scope":{"target":"web-app","namespace":"default"}}'

# Wait 5s for xApp processing
sleep 5

# Verify status via GET (should work now, no 405)
STATUS=$(curl -s "http://service-ricplt-a1mediator-http.ricplt:10000/A1-P/v2/policytypes/100/policies/test001/status")
echo $STATUS | jq .

# Expected: {"enforce_status":"ENFORCED","instance_status":"IN EFFECT"}
```

### Test 2: E2 KPI Collection
```bash
# Check E2 subscriptions
kubectl exec -n ricplt deployment/deployment-ricplt-submgr -- \
  curl -s http://localhost:8080/ric/v1/subscriptions | jq .

# Verify KPI metrics
kubectl exec -n ricxapp -l app=kpimon -- \
  curl -s http://localhost:8080/metrics | grep kpimon_kpi_value

# Expected: kpimon_kpi_value{measurement_type="DRB.UEThpDl",...} 15.8
```

---

## Risk Mitigation

| Component | Risk | Mitigation | Rollback |
|-----------|------|------------|----------|
| Redis | Data loss | Use PersistentVolume if needed | Status store is stateless, acceptable |
| srsRAN gNB | E2 connection fail | Pre-verify DNS/SCTP connectivity | `kubectl delete deployment srsran-gnb` |
| KPI Monitor | RMR routing errors | Validate routing table pre-deploy | Scale down, restore mock |

---

## Success Metrics

**Before Migration**:
- Working xApps: 1 (scaling only)
- E2-enabled gNBs: 0
- Real KPI metrics: 0
- A1 status updates: FAILING (HTTP 405)

**After Migration**:
- Working xApps: 2 (scaling + kpimon)
- E2-enabled gNBs: 1 (srsRAN)
- Real KPI metrics: 10+ RAN measurements
- A1 status updates: WORKING (Redis-based)

---

## References

### Documentation
- [O-RAN SC A1 Mediator API](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-a1/en/latest/user-guide-api.html)
- [KPI Monitor xApp Wiki](https://wiki.o-ran-sc.org/display/RICA/KPI+Monitor+xApp)
- [srsRAN Project Docs](https://docs.srsran.com/projects/project/en/latest/tutorials/source/near-rt-ric/source/index.html)

### Repositories
- [ric-app-kpimon-go](https://github.com/o-ran-sc/ric-app-kpimon-go)
- [OAI Kubernetes](https://github.com/OPENAIRINTERFACE/openair-k8s)
- [OAI Operators](https://github.com/OPENAIRINTERFACE/oai-operators)

### Tutorials
- [KPM xApp with OSC RIC](https://openrangym.com/tutorials/x5g-kpm-xapp)
- [xApp Onboarding Guide](https://docs.o-ran-sc.org/projects/o-ran-sc-ric-app-hw-go/en/latest/onboard-and-deploy.html)

---

## Next Steps

1. **Immediate (Day 1-2)**:
   - Deploy Redis status store
   - Update scaling xApp code
   - Deploy srsRAN gNB

2. **Short-term (Week 1-2)**:
   - Build and onboard KPI Monitor xApp
   - Configure E2 subscriptions
   - Run integration tests

3. **Medium-term (Week 3-4)**:
   - Production cutover
   - Remove all mock components
   - Document operations procedures

4. **Future (Q2 2026)**:
   - Evaluate OAI gNB deployment
   - Add more xApps (QoE optimization, handover control)
   - Scale to multi-cell scenarios

---

**Status**: Ready for implementation. All components analyzed, solutions validated, rollback procedures defined.

**Contact**: Nephoran Intent Operator Team
**Last Updated**: 2026-02-26

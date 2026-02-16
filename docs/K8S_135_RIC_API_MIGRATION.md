# Kubernetes 1.35 RIC API Migration Guide

## Overview
This document provides migration guidance for updating O-RAN RIC Helm charts to support Kubernetes 1.35.1.

## Breaking Changes

### 1. Ingress API Migration (networking.k8s.io/v1beta1 → v1)
**Removed in**: Kubernetes 1.22
**Status**: CRITICAL - RIC deployment will FAIL on K8s 1.35

#### Changes Required

**Before (v1beta1)**:
```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: example-ingress
spec:
  rules:
  - http:
      paths:
      - path: /api
        backend:
          serviceName: api-service
          servicePort: 8080
```

**After (v1)**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
spec:
  rules:
  - http:
      paths:
      - path: /api
        pathType: Prefix  # Required: Exact, Prefix, or ImplementationSpecific
        backend:
          service:
            name: api-service
            port:
              number: 8080
```

#### Key Changes:
- `apiVersion`: `networking.k8s.io/v1beta1` → `networking.k8s.io/v1`
- `pathType` field is now **required** (Exact, Prefix, or ImplementationSpecific)
- `backend.serviceName` → `backend.service.name`
- `backend.servicePort` → `backend.service.port.number` (or `.port.name` for named ports)

### 2. PodSecurityPolicy Removal (policy/v1beta1)
**Removed in**: Kubernetes 1.25
**Status**: CRITICAL - PodSecurityPolicy resources will be rejected

#### Migration Options

**Option A: Remove PodSecurityPolicy (Recommended)**
PodSecurityPolicy has been completely removed. Delete all PSP resources:
```bash
find deployments/ric -name "*.yaml" -exec grep -l "kind: PodSecurityPolicy" {} \; | xargs rm
```

**Option B: Migrate to Pod Security Standards**
Replace PSP with Pod Security Standards using namespace labels:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ricplt
  labels:
    pod-security.kubernetes.io/enforce: baseline  # or restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

**Pod Security Levels**:
- `privileged`: Unrestricted (use with caution)
- `baseline`: Minimally restrictive (recommended for most workloads)
- `restricted`: Heavily restricted (best practices enforced)

#### Migration for PodDisruptionBudget
No changes needed - `policy/v1` for PodDisruptionBudget is stable since K8s 1.21.

## Affected Files

### Ingress API (25 files)
Critical files requiring migration:
```
deployments/ric/ric-dep/helm/rsm/templates/ingress-rsm.yaml
deployments/ric/dep/ric-aux/helm/ves/templates/ingress-ves.yaml
deployments/ric/dep/ric-aux/helm/dashboard/templates/ingress.yaml
deployments/ric/repo/helm/xapp-onboarder/templates/ingress-*.yaml
```

### PodSecurityPolicy (26 files)
Files requiring removal or migration:
```
deployments/ric/ric-dep/helm/infrastructure/subcharts/prometheus/charts/kube-state-metrics/templates/podsecuritypolicy.yaml
deployments/ric/ric-dep/helm/infrastructure/subcharts/kong/charts/postgresql/templates/psp.yaml
deployments/ric/dep/ric-aux/helm/infrastructure/subcharts/kong/templates/psp.yaml
```

## Automated Migration Script

```bash
#!/bin/bash
# migrate-ric-apis.sh - Migrate RIC Helm charts to K8s 1.35

set -e

DEPLOYMENTS_DIR="deployments/ric"

echo "=== Kubernetes 1.35 RIC API Migration ==="

# 1. Migrate Ingress API version
echo "Step 1: Migrating Ingress API..."
find "$DEPLOYMENTS_DIR" -name "*.yaml" -type f -exec sed -i \
  's|networking\.k8s\.io/v1beta1|networking.k8s.io/v1|g' {} \;

# 2. Migrate Ingress backend format (requires manual review)
echo "Step 2: WARNING - Ingress backend format requires MANUAL migration"
echo "  - Change 'backend.serviceName' to 'backend.service.name'"
echo "  - Change 'backend.servicePort' to 'backend.service.port.number'"
echo "  - Add 'pathType: Prefix' to each path"

# 3. Remove PodSecurityPolicy
echo "Step 3: Removing PodSecurityPolicy resources..."
find "$DEPLOYMENTS_DIR" -name "*podsecuritypolicy.yaml" -type f -delete
find "$DEPLOYMENTS_DIR" -name "*psp.yaml" -type f -delete

echo "=== Migration Complete ==="
echo "IMPORTANT: Review changes before deploying!"
echo "See docs/K8S_135_RIC_API_MIGRATION.md for details"
```

## Recommended Approach

### Short-term (URGENT)
1. **Fix critical Ingress files** (RSM, VES, Dashboard, xApp Onboarder)
2. **Remove PodSecurityPolicy** resources from templates
3. **Test RIC deployment** on K8s 1.35 cluster

### Long-term (RECOMMENDED)
1. **Upgrade RIC Helm charts** to latest O-RAN SC releases:
   - O-RAN SC Cherry Release (or later) supports K8s 1.21+
   - Check: https://github.com/o-ran-sc
2. **Migrate to Gateway API** (future-proof):
   - Gateway API replaces Ingress with more powerful routing
   - See: docs/K8S_135_GATEWAY_API.md

## Testing Checklist

After migration:
- [ ] All Ingress resources deploy successfully
- [ ] RIC platform pods start correctly
- [ ] No PodSecurityPolicy errors in logs
- [ ] xApp onboarding works
- [ ] E2 connections established
- [ ] A1 policy interface functional

## References

- [Kubernetes 1.22 Ingress API Migration](https://kubernetes.io/docs/reference/using-api/deprecation-guide/#ingress-v122)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [O-RAN SC Release Notes](https://wiki.o-ran-sc.org/display/REL)

## Support

For issues or questions:
- File issue: https://github.com/thc1006/nephoran-intent-operator/issues
- O-RAN SC Slack: https://o-ran-sc.slack.com

---
**Last Updated**: 2026-02-16
**Kubernetes Version**: 1.35.1
**RIC Platform**: O-RAN SC Bronze/Cherry/Dawn

# Contract Schemas

This directory contains the canonical interface contracts for the Nephoran Intent Operator.

## Files

### intent.schema.json
Network scaling intent schema defining CNF/VNF lifecycle operations.

**Key Fields:**
- `apiVersion`: Contract version (v1alpha1)
- `kind`: NetworkIntent
- `spec.targetReplicas`: Desired instance count (1-100)
- `spec.resourceProfile`: Resource allocation tier (small/medium/large)
- `spec.locationHints`: Deployment constraints

### a1.policy.schema.json
O-RAN A1 policy interface for RAN optimization.

**Key Fields:**
- `policyType`: Traffic steering, QoS, or load balancing
- `scope`: Cell/slice/UE targeting
- `enforcement`: Mandatory/advisory/monitoring

### fcaps.ves.examples.json
VES 7.2 event samples for FCAPS monitoring.

**Event Types:**
- Fault: Service degradation alerts
- Configuration: Change notifications
- Accounting: Usage metrics
- Performance: KPI measurements
- Security: Threat detection

### e2.kpm.profile.md
E2 interface KPM (Key Performance Metrics) service model documentation.

## Versioning

- **v1alpha1**: Current development version
- Breaking changes require new version (v1beta1)
- All modules must update references when version changes

## Usage

```go
import "github.com/thc1006/nephoran-intent-operator/docs/contracts"
```

## Validation

```bash
# Validate JSON schemas
ajv compile -s intent.schema.json
ajv compile -s a1.policy.schema.json

# Note: The CI pipeline automatically validates these schemas on every PR
```
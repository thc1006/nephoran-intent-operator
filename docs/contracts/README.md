# Nephoran Intent Operator - Contract Definitions

This directory contains the formal interface contracts for the Nephoran Intent Operator MVP. These contracts define the boundaries between modules and ensure consistent integration across the system.

## Version Policy

All contracts follow semantic versioning:
- **Major**: Breaking changes requiring consumer updates
- **Minor**: Backward-compatible additions
- **Patch**: Documentation or non-functional changes

Current contract version: **1.0.0** (MVP baseline)

## Contract Files

### 1. intent.schema.json

**Purpose**: Defines the structure of scaling intents accepted by the system.

**Schema Version**: JSON Schema Draft 2020-12

**Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `intent_type` | string (const) | Yes | Must be "scaling" for MVP |
| `target` | string | Yes | Kubernetes Deployment name to scale |
| `namespace` | string | Yes | Kubernetes namespace of the target |
| `replicas` | integer | Yes | Desired replica count (1-100) |
| `reason` | string | No | Human-readable reason for scaling (max 512 chars) |
| `source` | enum | No | Origin of intent: "user", "planner", or "test" |
| `correlation_id` | string | No | Tracing identifier for request correlation |

**Validation Notes**:
- Enforced by admission webhook
- Replica limits may be further constrained by ConfigMap settings
- Empty strings not allowed for required fields

### 2. a1.policy.schema.json

**Purpose**: Models A1 policy instances for autonomous scaling decisions.

**Schema Version**: JSON Schema Draft 2020-12

**Fields**:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `policyTypeId` | string (const) | Yes | Must be "oran.sim.scaling.v1" |
| `scope` | object | Yes | Policy application scope |
| `scope.namespace` | string | Yes | Target Kubernetes namespace |
| `scope.target` | string | Yes | Target deployment name |
| `rules` | object | Yes | Scaling rules and thresholds |
| `rules.metric` | enum | Yes | Metric to monitor (see below) |
| `rules.scale_out_threshold` | number | Yes | Threshold to trigger scale-out |
| `rules.scale_in_threshold` | number | Yes | Threshold to trigger scale-in |
| `rules.cooldown_seconds` | integer | Yes | Minimum time between scaling actions |
| `rules.min_replicas` | integer | Yes | Minimum allowed replicas (≥1) |
| `rules.max_replicas` | integer | Yes | Maximum allowed replicas (≥1) |
| `notes` | string | No | Optional policy description |

**Supported Metrics**:
- `kpm.p95_latency_ms`: 95th percentile latency in milliseconds
- `kpm.prb_utilization`: Physical Resource Block utilization (0-1)
- `kpm.ue_count`: Number of connected User Equipment

**Integration**: Pushed via A1 NBI to Near-RT RIC for policy enforcement.

### 3. fcaps.ves.examples.json

**Purpose**: Example VES (VNF Event Stream) messages for FCAPS integration.

**Structure**: Contains three example event types:

#### Fault Event
- **Domain**: `fault`
- **Key Fields**:
  - `alarmCondition`: Fault type (e.g., "LINK_DOWN")
  - `eventSeverity`: Impact level (e.g., "CRITICAL")
  - `specificProblem`: Detailed description
  - `alarmInterfaceA`: Affected interface

#### Measurement Event
- **Domain**: `measurementsForVfScaling`
- **Key Fields**:
  - `vNicUsageArray`: Network interface statistics
  - `additionalFields`: KPM metrics matching A1 policy metrics

#### Heartbeat Event
- **Domain**: `heartbeat`
- **Key Fields**:
  - `heartbeatInterval`: Expected interval in seconds

**Version**: VES 4.1 / Fault Fields 4.0 / Measurements 1.1 / Heartbeat 3.0

### 4. e2.kpm.profile.md

**Purpose**: Documents the E2SM-KPM (Key Performance Measurement) profile for RIC integration.

**Key Specifications**:

| Aspect | Value |
|--------|-------|
| Sampling Window | 30 seconds |
| Max Data Staleness | 90 seconds |
| Collection Path | E2 Node → E2 Termination → KPM xApp → REST API |

**Metrics Collected**:
- `kpm.p95_latency_ms`: P95 end-to-end latency (UL+DL)
- `kpm.prb_utilization`: PRB utilization ratio [0..1]
- `kpm.ue_count`: Connected UE count

**Missing Data Policy**: No scaling action when data unavailable

### 5. meta-2020-12.json

**Purpose**: JSON Schema 2020-12 meta-schema for validation tooling.

**Usage**: Reference schema for validating our contract schemas with ajv-cli.

## Validation

All JSON schemas can be validated using ajv-cli:

```bash
# Install ajv-cli globally
npm install -g ajv-cli

# Validate intent schema structure
ajv compile --spec=draft2020 -s docs/contracts/intent.schema.json

# Validate A1 policy schema structure
ajv compile --spec=draft2020 -s docs/contracts/a1.policy.schema.json

# Validate an intent instance against schema
ajv validate --spec=draft2020 -s docs/contracts/intent.schema.json -d examples/intent-sample.json

# Validate a policy instance against schema
ajv validate --spec=draft2020 -s docs/contracts/a1.policy.schema.json -d examples/policy-sample.json
```

## Contract Changes

### Process
1. Changes to contracts must be proposed in a separate PR to `integrate/mvp`
2. Update version numbers according to semantic versioning
3. Update examples to match new schemas
4. Announce breaking changes in PR description
5. Update this README with field changes

### Backward Compatibility
- MVP contracts (v1.0.0) establish the baseline
- Minor versions may add optional fields
- Major versions indicate breaking changes requiring code updates

## Integration Points

| Contract | Producer | Consumer | Transport |
|----------|----------|----------|-----------|
| intent.schema | REST API clients | Intent Controller | HTTP POST /api/v1/intent |
| a1.policy.schema | Planner Service | A1 Mediator/RIC | A1 REST API |
| fcaps.ves | Network Functions | VES Collector | HTTP/HTTPS |
| e2.kpm.profile | E2 Nodes/xApps | Planner Service | REST readout |

## Testing Contracts

Example test data is provided in the `examples/` directory:
- `examples/intent-scaling-up.json`: Valid scale-up intent
- `examples/intent-scaling-down.json`: Valid scale-down intent
- `examples/policy-latency-based.json`: Latency-triggered scaling policy
- `examples/policy-prb-based.json`: PRB utilization scaling policy

## References

- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core.html)
- [O-RAN A1 Interface Specification](https://www.o-ran.org/specifications)
- [VES Event Listener 7.2](https://docs.onap.org/projects/onap-dcaegen2/en/latest/sections/services/ves-http/index.html)
- [E2SM-KPM Service Model](https://wiki.o-ran-sc.org/display/RICNR/E2SM-KPM)

## Ownership

These contracts are maintained by the Project Conductor role. Feature teams must not modify contracts directly in their branches. Contract changes require coordination through the `integrate/mvp` branch.

For questions or clarifications, please open an issue with the `contract` label.
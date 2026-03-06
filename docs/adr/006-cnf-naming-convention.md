# ADR-006: CNF Naming Convention

**Status:** Accepted | **Date:** 2025-03-06

## Context
When CNFs are scaled out, new pod instances must have predictable, distinguishable names. Need to differentiate base pods from scaled replicas and associate with slice type.

## Decision
Pattern: `{component}-{slice-type}-{instance-id}[-scaled-{ordinal}]`

- `component`: odu | ocu-cp | ocu-up | amf | smf | upf | nrf | nssf
- `slice-type`: embb | urllc | mmtc | custom-<n>
- `instance-id`: zero-padded 3 digits (e.g., 001)
- `-scaled-{ordinal}`: ONLY for scale-out replicas; zero-padded 2 digits

### Required Kubernetes Labels
```yaml
app.kubernetes.io/component: "odu"
nephio.org/slice-type: "embb"
nephio.org/instance-id: "001"
nephio.org/scaled-ordinal: "02"    # only on scaled replicas
```

### Examples
| Scenario | Pod Name |
|---|---|
| Base O-DU for eMBB | `odu-embb-001` |
| Scaled-out O-DU #2 | `odu-embb-001-scaled-02` |
| Base AMF | `amf-embb-001` |

## Enforcement
- OPA Rego policy (`opa_policies/naming_convention.rego`)
- Config Validator pre-commit check
- LLM prompt templates include naming rules as constraints

# ADR-008: Config Validation Strategy

**Status:** Accepted | **Date:** 2025-03-06

## Context
LLM-generated YAML may contain errors, security issues, or non-compliant configurations. Need multi-layer validation before cluster deployment.

## Decision
Five validation layers (executed in order):
1. **YAML syntax**: Parse with `ruamel.yaml`; reject malformed
2. **Kubernetes schema**: Validate against K8s OpenAPI spec using `kubeconform`
3. **OPA/Gatekeeper policies**: Rego policies for naming, resource limits, network policies
4. **Naming convention**: Specific check per ADR-006
5. **LLM semantic review** (optional): Send YAML back to LLM for semantic correctness

## Consequences
- Layers 1-4 deterministic and fast; layer 5 adds latency but catches semantic issues
- Results: `error` (blocks commit) / `warning` (logged)
- All policies in `opa_policies/`, version-controlled

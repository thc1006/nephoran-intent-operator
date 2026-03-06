# ADR-0004: LLM must not directly operate clusters; only propose changes via PRs

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- LLM outputs can be incorrect or unsafe.
- Direct cluster mutation reduces auditability and increases blast radius.
- Research prototype must be safe-by-design.

## Decision
- LLM produces **IntentPlan.json** only.
- All changes become PRs with CI validation and (optionally) human approval.
- Closed-loop actions are PR-based (Act phase = propose PR), never imperative apply.

## Consequences
- Pros: safety, governance, reproducibility.
- Cons: slower feedback loop; requires strong CI and good diff summaries.

## Alternatives considered
- Allow LLM to run kubectl in a sandbox namespace.
- Allow LLM to patch only Deployments replicas directly.


## References
- Porch GitOps rationale: https://docs.nephio.org/docs/porch/

# ADR-010: Closed-Loop Architecture

**Status:** Accepted | **Date:** 2025-03-06

## Context
System must auto-detect performance issues and scale CNFs via Observe-Analyze-Act with LLM-in-the-loop.

## Decision
- **Observe**: Prometheus + KPIMON xApp metrics, 15s scrape interval
- **Analyze**: Metrics summary -> LLM Gateway `/analyze` -> evaluate against intent SLA
- **Act**: Proposed actions -> Intent Engine -> GitOps pipeline (validate -> commit -> sync -> deploy)

### Guard Rails
- Max scale-out: configurable per component (default 5 replicas)
- Cooldown: minimum 5 minutes between same-component actions
- Human approval gate: configurable (default enabled for scale-out > 3)
- Auto rollback if new replicas fail health checks within 2 minutes

## Consequences
- Loop latency (observe -> deploy) is minutes, not seconds — acceptable for RAN scaling
- False positive scaling recommendations mitigated by guard rails and cooldown
- All closed-loop actions auditable via Git history

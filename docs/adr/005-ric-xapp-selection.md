# ADR-005: RIC & xApp Selection

**Status:** Accepted | **Date:** 2025-03-06

## Context
Closed-loop automation requires a Near-RT RIC with xApps for KPI monitoring and traffic steering. Must be mature, officially maintained.

## Decision
- **Near-RT RIC Platform**: O-RAN SC official release (k-release or later), deployed via Helm
  - Includes: E2 termination, A1 mediator, Subscription Manager, Routing Manager
- **KPIMON xApp**: O-RAN SC `ric-app-kpimon` — collects RAN KPIs via E2SM-KPM
- **Traffic Steering xApp**: O-RAN SC `ric-app-ts` — A1 policy-based traffic steering

## Consequences
- O-RAN SC RIC requires its own namespace and Helm-based deployment
- E2 connectivity between OAI gNB and RIC must be validated
- KPIMON metrics feed into Prometheus for closed-loop Observe phase

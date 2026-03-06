# ADR-009: E2 Simulator & Traffic Simulation

**Status:** Accepted | **Date:** 2025-03-06

## Context
To validate CNF scale-out/in resolves traffic issues for specific network slices, need realistic traffic simulation via E2 interface.

## Decision
- Use O-RAN SC `e2sim` as E2 simulator
- Custom traffic profiles:
  - eMBB: high throughput, moderate UE count
  - URLLC: low latency, burst traffic
  - mMTC: high UE count, low per-UE throughput
- E2sim reports metrics via E2SM-KPM -> KPIMON xApp -> Prometheus
- Profiles configurable via ConfigMap

## Consequences
- May need e2sim patches for custom E2SM-KPM indicators
- Traffic profiles must generate enough load to trigger scaling thresholds
- E2sim runs as K8s deployment alongside RIC

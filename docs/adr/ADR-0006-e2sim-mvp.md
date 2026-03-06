# ADR-0006: MVP uses O-RAN SC E2 simulator interface for E2 traffic/indication scaffolding

- **Status**: Proposed
- **Date**: 2026-03-06

## Context
- To validate xApps and closed-loop behavior early, we need an E2-facing simulator/driver before full RAN integration.
- O-RAN SC provides simulator interface components to scaffold E2 interactions.

## Decision
- Use `o-ran-sc/sim-e2-interface` for MVP scaffolding.
- Provide a `packages/base/components/sim-e2/` placeholder package and runbook steps.
- In later milestones, replace or complement with real RAN E2 agents and traffic generation.

## Consequences
- Pros: early integration testing; decouples from full RAN bring-up.
- Cons: simulation fidelity limitations; must be clearly labeled as scaffolding.

## Alternatives considered
- Directly start with real E2 agent + OAI RAN.
- Use alternative E2SIM projects not aligned with O-RAN SC.


## References
- sim-e2-interface: https://github.com/o-ran-sc/sim-e2-interface

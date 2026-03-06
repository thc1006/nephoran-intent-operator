# ADR-0005: Use mature O-RAN SC upstream xApps (KPIMON + Traffic Steering)

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- We want credible validation with minimal re-implementation.
- O-RAN SC maintains upstream xApps and documentation for near-RT RIC use cases.

## Decision
- Adopt upstream xApps:
  - KPI Monitoring: `o-ran-sc/ric-app-kpimon-go`
  - Traffic Steering: `o-ran-sc/ric-app-ts`
- Integrate them as components/packages in `packages/base/components/` with clear onboarding steps and assumptions.

## Consequences
- Pros: alignment with O-RAN SC ecosystem; easier to compare/extend.
- Cons: integration complexity (SDL, onboarder/app manager, dependencies).

## Alternatives considered
- Build our own simplified xApps.
- Use third-party forks without upstream alignment.

## References
- KPIMON repo: https://github.com/o-ran-sc/ric-app-kpimon-go
- Traffic Steering user guide: https://docs.o-ran-sc.org/projects/o-ran-sc-ric-app-ts/en/latest/user-guide.html

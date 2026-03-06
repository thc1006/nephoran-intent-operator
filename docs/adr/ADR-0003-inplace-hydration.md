# ADR-0003: Prefer in-place hydration (kpt packages) for Day-2 operations

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- We need repeatable composition of base packages with environment/site parameters.
- In-place hydration is often easier to maintain for Day-2 modifications than over-parameterized templating.

## Decision
- Use kpt-style packages with setters/patches and in-place hydration by default.
- Allow out-of-place only when needed for generating derived packages for specific workflows.

## Consequences
- Pros: clearer diffs, easier review, fewer hidden templating behaviors.
- Cons: may require more explicit overlays and package organization.

## Alternatives considered
- Use Helm templating for everything.
- Use only out-of-place generation and treat outputs as canonical.

## References
- Nephio glossary (Hydration): https://docs.nephio.org/docs/glossary-abbreviations/

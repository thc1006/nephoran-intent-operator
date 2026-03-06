# ADR-0007: Stable identity uses resource names + labels; StatefulSet only when needed

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- Deployment-managed Pods have non-stable generated suffixes.
- Closed-loop automation needs stable correlation across revisions and scale events.

## Decision
- Use deterministic resource naming: `<domain>-<component>-<site>-<slice>-<instance>`.
- Use mandatory labels for correlation (intent-id, site, slice, component).
- If stable pod identity is required, use StatefulSet with ordinal suffixes.

## Consequences
- Pros: deterministic tracking, clear diffs and audit.
- Cons: requires discipline and validations in generator.

## Alternatives considered
- Rely on pod names.
- Use annotations only.
- Force custom pod naming via mutating webhooks.

## References
- Internal project standard; see `CLAUDE.md` section 'Naming + Labeling'.

# ADR-0001: Northbound Interface uses TM Forum TMF921 (Intent Management API) subset

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- We need a standardized intent interface for interoperability and research reproducibility.
- A custom API would be fast, but would reduce portability and hinder alignment with OSS/BSS practices.

## Decision
- Implement a **minimal TMF921 v5.0 compatible subset** in `intentd`.
- Maintain an internal canonical `IntentPlan` model, and map TMF921 Intent payloads to/from it.
- MVP endpoints: create intent, get intent, get status/observations (subset).

## Consequences
- Pros: standard-aligned, easier to extend later.
- Cons: mapping layer adds complexity; full TMF921 coverage is out-of-scope for MVP.

## Alternatives considered
- Custom REST/gRPC API.
- CLI-only MVP without any API.
- Implement older TMF921 versions.

## References
- TMF921 v5.0 page: https://www.tmforum.org/open-digital-architecture/open-apis/intent-management-api-TMF921/v5.0
- TMF921 User Guide v5.0.0: https://www.tmforum.org/resources/specifications/tmf921-intent-management-api-user-guide-v5-0-0/

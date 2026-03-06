# ADR-0002: Configuration lifecycle is anchored on Porch (kpt-as-a-service) + GitOps

- **Status**: Accepted
- **Date**: 2026-03-06

## Context
- The system must be auditable, reviewable, and reversible.
- Nephio/Porch provides opinionated KRM package lifecycle management and supports controller-driven automation.
- GitOps provides a well-understood operational model for safe changes.

## Decision
- All desired state changes are expressed as **KRM/kpt packages**.
- Packages are managed through Porch lifecycle (draft → proposed → published) and delivered via ConfigSync.
- No direct `kubectl apply` for operational changes.

## Consequences
- Pros: reproducible, traceable, supports approvals and promotion gates.
- Cons: adds PR/approval latency; requires CI checks and promotion controls.

## Alternatives considered
- Use Helm-only + ArgoCD without Porch.
- Direct kubectl/imperative changes.
- Terraform-only approach for K8s resources.

## References
- Porch overview: https://docs.nephio.org/docs/porch/
- Porch lifecycle guidance: https://docs.nephio.org/docs/guides/user-guides/

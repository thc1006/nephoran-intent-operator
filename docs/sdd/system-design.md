# System Design (SDD)
This is a lightweight SDD extension. The canonical spec is in `CLAUDE.md`.

## Key interfaces
- TMF921 subset endpoints in `docs/api/openapi.yaml`
- Canonical plan schema in `schemas/intent-plan.schema.json`

## Key invariants
- All changes PR-based
- Plan schema-constrained
- Package generation deterministic

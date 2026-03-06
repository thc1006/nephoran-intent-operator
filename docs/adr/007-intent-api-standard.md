# ADR-007: Intent API Standard

**Status:** Accepted | **Date:** 2025-03-06

## Context
The Intent Engine needs an API standard aligned with telecom industry practices for receiving natural-language intents.

## Decision
- Align with TM Forum TMF 921 Intent Management API concepts
- Core resources: Intent, IntentReport, IntentPlan
- Endpoints:
  - `POST /intents` — submit new intent
  - `GET /intents/{id}` — retrieve status and plan
  - `GET /intents/{id}/report` — execution report
  - `DELETE /intents/{id}` — cancel/rollback
- Request body: `original_text` (NL) + optional structured hints
- Internal contract: `schemas/intent-plan.schema.json`

## Consequences
- Full TMF 921 compliance is non-goal; adopt conceptual model and key attributes

# ADR-003: Nephio Porch Integration Strategy

**Status:** Accepted | **Date:** 2025-03-06

## Context
Nephio provides Porch (Package Orchestration) for managing KRM packages through a lifecycle (draft -> proposed -> published). LLM-generated configs must integrate into this lifecycle, leveraging Nephio's template hydration and specialization.

## Decision
- Use Porch REST API for programmatic package management
- LLM Gateway generates base KRM packages; Nephio handles hydration (injectors) and specialization
- PackageVariantSet CRDs for multi-cluster deployment with per-cluster specialization
- Intent Engine submits packages in "draft" state; validation promotes to "proposed"; approval publishes

## Consequences
- Requires Nephio R3+ for stable Porch API
- LLM output must conform to KRM package structure (Kptfile + resources)
- Hydration functions (set-namespace, set-labels, inject-resource-limits) are standard Nephio functions

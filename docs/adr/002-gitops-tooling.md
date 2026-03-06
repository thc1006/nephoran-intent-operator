# ADR-002: GitOps Tooling Selection

**Status:** Accepted | **Date:** 2025-03-06

## Context
LLM-generated configurations must flow through a GitOps pipeline before reaching Nephio/Kubernetes clusters. Need a GitOps controller that integrates well with Nephio Porch.

## Decision
- **Primary**: FluxCD — lighter footprint, native Kustomize support, good Nephio alignment
- **Alternative**: Argo CD — supported for users preferring its UI
- Config Sync (Google) rejected due to GKE coupling

## Consequences
- FluxCD `GitRepository` and `Kustomization` CRDs manage sync from Git to cluster
- Nephio Porch operates upstream: Porch manages package lifecycle, FluxCD handles cluster sync
- Both FluxCD and Argo CD manifests maintained in `gitops/` directory

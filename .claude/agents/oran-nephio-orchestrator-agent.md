---
name: oran-nephio-orchestrator-agent
description: Advanced orchestration agent for O-RAN L Release and Nephio R5 deployments. Use PROACTIVELY for complex multi-cluster orchestration, GitOps workflows, and cross-domain service deployment. MUST BE USED for coordinating O-RAN network functions across distributed edge infrastructure with comprehensive automation.
model: opus
tools: Read, Write, Bash, Search, Git
version: 2.0.0
last_updated: August 20, 2025
dependencies:
  - go: 1.24.6
  - kubernetes: 1.32+
  - argocd: 3.1.0+
  - kpt: v1.0.0-beta.27
  - helm: 3.14+
  - nephio: r5
  - porch: 1.0.0+
  - cluster-api: 1.6.0+
  - metal3: 1.6.0+
  - crossplane: 1.15.0+
  - flux: 2.2+
  - terraform: 1.7+
  - ansible: 9.2+
  - istio: 1.20+
compatibility:
  nephio: r5
  oran: l-release
  go: 1.24.6
  kubernetes: 1.32+
  os: linux/amd64, linux/arm64
  cloud_providers: [aws, azure, gcp, on-premise]
validation_status: tested
maintainer:
  name: O-RAN Orchestration Team
  email: oran-orchestration@nephio-oran.io
  slack: "#oran-orchestration"
  github: "@nephio-oran/oran-orchestration"
---

## Version Compatibility Matrix

### Orchestration Platform

| Component | Required Version | O-RAN L Release | Nephio R5 | Notes |
|-----------|------------------|-----------------|-----------|-------|
| **Go** | 1.24.6 | ✅ Compatible | ✅ Compatible | Cloud-native development |
| **Kubernetes** | 1.32+ | ✅ Compatible | ✅ Compatible | Container orchestration |
| **ArgoCD** | 3.1.0+ | ✅ Compatible | ✅ Compatible | GitOps deployment engine |
| **Kpt** | 1.0.0-beta.27+ | ✅ Compatible | ✅ Compatible | Package management |
| **Helm** | 3.14+ | ✅ Compatible | ✅ Compatible | Chart management |

### GitOps Stack

| Component | Required Version | O-RAN L Release | Nephio R5 | Notes |
|-----------|------------------|-----------------|-----------|-------|
| **Git** | 2.40+ | ✅ Compatible | ✅ Compatible | Version control |
| **Kustomize** | 5.3+ | ✅ Compatible | ✅ Compatible | Configuration management |
| **Jsonnet** | 0.20+ | ✅ Compatible | ✅ Compatible | Configuration templating |
| **ConfigSync** | 1.17+ | ✅ Compatible | ✅ Compatible | Alternative to ArgoCD |
| **Flux** | 2.2+ | ✅ Compatible | ✅ Compatible | GitOps toolkit |

### Cloud Infrastructure

| Component | Required Version | O-RAN L Release | Nephio R5 | Notes |
|-----------|------------------|-----------------|-----------|-------|
| **Terraform** | 1.7+ | ✅ Compatible | ✅ Compatible | Infrastructure as Code |
| **Cluster API** | 1.6+ | ✅ Compatible | ✅ Compatible | Kubernetes cluster lifecycle |
| **Metal3** | 0.5+ | ✅ Compatible | ✅ Compatible | Bare metal provisioning |
| **Crossplane** | 1.15+ | ✅ Compatible | ✅ Compatible | Cloud resource management |
| **Istio** | 1.20+ | ✅ Compatible | ✅ Compatible | Service mesh |


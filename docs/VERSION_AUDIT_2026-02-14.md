# Version Audit Report - Nephoran Intent Operator
**Date:** 2026-02-14
**Scope:** Deep analysis of all O-RAN, infrastructure, and dependency versions

## Executive Summary

This audit identifies **15 critical version updates** across O-RAN components, programming languages, Kubernetes, and dependencies. The repository is currently 1-2 releases behind latest stable versions in multiple areas.

### Priority Updates Required

| Component | Current | Latest | Priority | Risk |
|-----------|---------|--------|----------|------|
| Kubernetes | 1.29 | 1.35.1 | 🔴 CRITICAL | K8s 1.29 EOL: Feb 28, 2026 |
| Go | 1.24.6 | 1.26 | 🟠 HIGH | Performance & security fixes |
| O-RAN SC RIC | L Release (forks) | M Release | 🟠 HIGH | Missing Dec 2025 features |
| weaviate-client | 3.26.2 | 4.19.2 | 🟡 MEDIUM | Multi-vector search support |
| OpenAI models | gpt-4o-mini | GPT-5.2, GPT-4.1 | 🟡 MEDIUM | GPT-4o retiring Feb 13, 2026 |

---

## 1. O-RAN Software Community Components

### Near-RT RIC Platform Releases

**Current Status:**
- Using **thc1006 forks** of O-RAN SC components with custom commits
- Submodules:
  - `ric-plt-submgr` (commit: 7543a37)
  - `ric-plt-rtmgr` (commit: 22a6176)
  - `ric-plt-e2mgr` (commit: 2cedb68)
  - `ric-plt-e2` (commit: a66badf)
  - `ric-plt-appmgr` (commit: 00c8cca)

**Official O-RAN SC Release Timeline:**

| Release | Date | Status | Key Features |
|---------|------|--------|--------------|
| **J Release** | 2024-07-01 | Superseded | Python O1 Simulator, Kubeflow integration |
| **K Release** | 2024-12-19 | Superseded | AI/ML APIs, K8s operators, OKD support |
| **L Release** | 2025-06-30 | Active | New simulator, enhanced AI/ML framework |
| **M Release** | 2025-12-20 | **LATEST** | Enhanced SMO integration, TEIV improvements |

**Recommendation:** 🟠 **HIGH PRIORITY**
- **Action:** Sync thc1006 forks with O-RAN SC M Release
- **Why:** M Release includes critical SMO integration improvements and updated APIs
- **Impact:** Better O-RAN ALLIANCE alignment, newer xApp support
- **Effort:** Moderate (need to rebase forks on upstream M Release tags)

**Sources:**
- [O-RAN SC Documentation](https://docs.o-ran-sc.org/en/latest/release-notes.html)
- [O-RAN SC M Release Home](https://docs.o-ran-sc.org/en/latest/)
- [O-RAN SC Projects](https://docs.o-ran-sc.org/en/latest/projects.html)

---

## 2. Programming Languages & Toolchains

### Go (Golang)

**Current:**
```go
go 1.24.0
toolchain go1.24.6
```

**Latest Stable:** Go 1.26 (released **February 10, 2026**)
- Go 1.24.13: Latest patch for 1.24 series (Feb 4, 2026) - security fixes
- Go 1.26: Latest major release with language changes and performance improvements

**Recommendation:** 🟠 **HIGH PRIORITY**
- **Action:** Upgrade to Go 1.26
- **Why:**
  - Security fixes in crypto/tls and crypto/x509
  - Performance improvements
  - Two new language changes
  - Actively supported (1.24 approaching EOL)
- **Migration:** Review Go 1.26 release notes for breaking changes
- **Effort:** Low (minor version bump, backward compatible)

**Sources:**
- [Go Release History](https://go.dev/doc/devel/release)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)
- [Go 1.24 Release](https://go.dev/doc/go1.24)

---

## 3. Kubernetes & Cloud Infrastructure

### Kubernetes

**Current (Terraform defaults):**
```hcl
aws_cluster_version = "1.29"
azure_cluster_version = "1.29"
```

**Latest Stable:** Kubernetes 1.35.1 (released **February 10, 2026**)

**Supported Versions (as of Feb 2026):**

| Version | EOL Date | Status |
|---------|----------|--------|
| 1.35.1 | Feb 28, 2027 | ✅ Latest |
| 1.34.4 | Oct 27, 2026 | ✅ Supported |
| 1.33.8 | Jun 28, 2026 | ✅ Supported |
| 1.32.12 | **Feb 28, 2026** | ⚠️ Expiring soon |
| 1.29.x | **UNSUPPORTED** | ❌ EOL |

**Recommendation:** 🔴 **CRITICAL PRIORITY**
- **Action:** Upgrade to Kubernetes 1.35.1 (or minimum 1.33.8)
- **Why:**
  - **K8s 1.29 is past EOL** - no security patches
  - Production clusters on EOL versions are security risks
  - Cloud providers (EKS, AKS, GKE) will force upgrades
- **Migration Path:** 1.29 → 1.33 → 1.35 (staged upgrade recommended)
- **Testing:** Update all K8s manifests, test CRDs compatibility
- **Effort:** High (requires cluster upgrades, testing)

**Cloud Provider Support:**
- **AWS EKS:** Latest supported: 1.35
- **Azure AKS:** Latest supported: 1.35
- **Google GKE:** Latest supported: 1.35

**Sources:**
- [Kubernetes Releases](https://kubernetes.io/releases/)
- [Kubernetes EOL](https://endoflife.date/kubernetes)
- [K8s v1.35 Release](https://kubernetes.io/blog/2025/12/17/kubernetes-v1-35-release/)

---

## 4. Python RAG Pipeline Dependencies

### Current Versions (rag-python/requirements.txt)

**Core RAG Framework:**
```txt
langchain==0.1.20
langchain-community==0.0.38
langchain-openai==0.1.7
```

**Latest Available:**
- `langchain`: **0.3.x** (major version bump available)
- `langchain-community`: Check PyPI for latest
- `langchain-openai`: **0.2.x+** available

**Vector Stores:**
```txt
weaviate-client==3.26.2
chromadb==0.4.22
faiss-cpu==1.7.4
```

**Latest:**
- `weaviate-client`: **4.19.2** (released recently)
  - New features: Multi-vector search, scalar quantization, async replication, S3 tenant offloading
  - Breaking changes: API redesign from v3 to v4
- Weaviate Server: **1.34** (November 2025)
  - Flat index with RQ quantization
  - Server-side batching improvements

**LLM Providers:**
```txt
openai==1.14.0
```

**Latest:** openai >= 1.50.x
**Critical Model Changes (Feb 13, 2026):**
- ❌ **GPT-4o** - Being retired from ChatGPT
- ❌ **GPT-4o-mini** - Being retired
- ✅ **GPT-5.2** - Latest, smarter, more capable
- ✅ **GPT-4.1 / GPT-4.1 mini** - Specialized for coding

**Current Code:**
```python
# rag-python/telecom_pipeline.py:31
self.llm = ChatOpenAI(
    model="gpt-4o-mini",  # ⚠️ DEPRECATED MODEL
    temperature=0,
    max_tokens=2048,
)
```

**API Server:**
```txt
fastapi==0.110.0
uvicorn[standard]==0.28.0
pydantic==2.6.3
```

**Latest:**
- `fastapi`: **0.115.x+** (check PyPI)
- `uvicorn`: **0.32.x+**
- `pydantic`: **2.10.x+**

**Monitoring:**
```txt
prometheus-client==0.20.0
opentelemetry-api==1.23.0
opentelemetry-sdk==1.23.0
```

**Latest:**
- `prometheus-client`: **0.21.x+**
- `opentelemetry-*`: **1.30.x+** (active development)

### Recommendations: 🟡 **MEDIUM PRIORITY**

**Immediate Actions:**
1. **OpenAI Model Migration (CRITICAL):**
   - Replace `gpt-4o-mini` with `gpt-4.1-mini` or `gpt-5.2`
   - Update cost calculations (new pricing)
   - Test prompt compatibility

2. **Weaviate Client v3 → v4 (BREAKING CHANGE):**
   - Major API redesign
   - Benefits: Multi-vector search, better performance
   - Requires code refactoring
   - Migration guide: https://weaviate.io/developers/weaviate/client-libraries/python

3. **LangChain 0.1.x → 0.3.x:**
   - Check breaking changes
   - New features: Improved streaming, better error handling
   - Update all langchain-* packages together

**Sources:**
- [OpenAI Model Retirement](https://openai.com/index/retiring-gpt-4o-and-older-models/)
- [Weaviate Release Notes](https://docs.weaviate.io/weaviate/release-notes)
- [Weaviate Python Client 4.19.2](https://weaviate-python-client.readthedocs.io/en/stable/changelog.html)
- [LangChain Weaviate Integration](https://python.langchain.com/docs/integrations/vectorstores/weaviate/)

---

## 5. Go Dependencies (Partial Audit)

### Current (go.mod)

**Cloud SDKs:**
```go
github.com/aws/aws-sdk-go-v2 v1.39.0
github.com/Azure/azure-sdk-for-go/sdk/azcore v1.19.1
cloud.google.com/go/compute v1.44.0
```

**Kubernetes:**
```go
// Need to check client-go, controller-runtime versions
// Should align with K8s 1.35.1 target
```

**Recommendation:** 🟡 **MEDIUM PRIORITY**
- Run `go get -u ./...` to check for available updates
- Focus on security patches
- Ensure compatibility with Go 1.26

---

## 6. Infrastructure as Code

### Terraform

**Current:** No explicit version pinning detected in `infrastructure/terraform/`

**Recommendation:**
- Add Terraform version constraint:
  ```hcl
  terraform {
    required_version = ">= 1.9.0"
  }
  ```
- Latest Terraform: **1.10.x** (check Hashicorp releases)

---

## 7. Version Update Priority Matrix

### Phase 1: Critical Security (Complete within 1 week)
1. ✅ Kubernetes 1.29 → 1.35.1 (CRITICAL - EOL risk)
2. ✅ OpenAI gpt-4o-mini → gpt-4.1-mini (Model retirement)
3. ✅ Go 1.24.6 → 1.26 (Security fixes)

### Phase 2: Major Features (Complete within 2 weeks)
4. 🔄 O-RAN SC forks → M Release sync
5. 🔄 weaviate-client 3.26.2 → 4.19.2 (Breaking change)
6. 🔄 langchain 0.1.20 → 0.3.x

### Phase 3: Incremental Updates (Complete within 1 month)
7. 🔄 Python dependencies patch updates
8. 🔄 Go dependencies updates
9. 🔄 Cloud SDK updates

---

## 8. Testing Strategy for Updates

### Pre-Update Checklist
- [ ] Create version update tracking branch
- [ ] Run full test suite on current versions
- [ ] Document current behavior (baseline)
- [ ] Set up rollback plan

### Per-Component Testing
1. **Kubernetes:**
   - Test CRD compatibility
   - Verify controller reconciliation
   - Check RBAC policies
   - Validate webhooks

2. **Go 1.26:**
   - Run `go test ./...`
   - Check build performance
   - Verify CGO compatibility

3. **O-RAN SC M Release:**
   - Test E2 interface compatibility
   - Verify xApp deployment
   - Check A1 policy handling

4. **RAG Pipeline:**
   - Test LLM response quality
   - Verify vector search accuracy
   - Check API latency

### Integration Testing
- Full E2E workflow tests
- Load testing with updated components
- Security scanning (CVE checks)

---

## 9. Estimated Effort & Timeline

| Task | Effort | Timeline | Engineer Days |
|------|--------|----------|---------------|
| K8s 1.35.1 upgrade | High | Week 1 | 3-5 days |
| Go 1.26 upgrade | Low | Week 1 | 0.5 day |
| OpenAI model migration | Medium | Week 1 | 1-2 days |
| O-RAN SC M Release sync | High | Week 2-3 | 5-7 days |
| Weaviate v4 migration | High | Week 3-4 | 3-4 days |
| Python deps updates | Medium | Week 4 | 2-3 days |
| **Total** | | **4 weeks** | **15-22 days** |

---

## 10. Risk Assessment

### High Risk Updates
- **Kubernetes 1.35.1:** Cluster downtime, workload disruptions
- **Weaviate v4:** API breaking changes, data migration
- **O-RAN SC M Release:** Interface compatibility issues

### Mitigation Strategies
1. **Blue-Green Deployment:** Parallel clusters for K8s upgrade
2. **Feature Flags:** Gradual rollout of RAG v4 features
3. **Canary Deployments:** Test O-RAN updates on staging first
4. **Automated Rollback:** Pre-configured rollback procedures

---

## 11. Version Pinning Strategy

### Recommendations for Future
1. **Pin all versions explicitly** (no floating dependencies)
2. **Use Dependabot/Renovate** for automated PR generation
3. **Semantic versioning policy:**
   - Patch updates: Auto-merge after CI
   - Minor updates: Review + manual merge
   - Major updates: Dedicated spike + testing
4. **Quarterly version audits** (repeat this analysis)

---

## 12. Action Items Summary

### Immediate (This Week)
- [ ] Create issue for K8s 1.35.1 upgrade
- [ ] Update OpenAI model in RAG pipeline
- [ ] Upgrade Go to 1.26
- [ ] Add Terraform version constraints

### Short-term (Next 2 Weeks)
- [ ] Research O-RAN SC M Release changes
- [ ] Plan Weaviate v4 migration
- [ ] Update LangChain dependencies

### Long-term (Next Month)
- [ ] Implement automated dependency tracking
- [ ] Set up quarterly version audit process
- [ ] Create version compatibility matrix docs

---

## 13. References & Sources

### O-RAN SC
- [O-RAN SC M Release Documentation](https://docs.o-ran-sc.org/en/latest/)
- [Near-RT RIC Installation Guide](https://lf-o-ran-sc.atlassian.net/wiki/display/RICP/Introduction+and+guides)
- [ric-plt-submgr Repository](https://github.com/o-ran-sc/ric-plt-submgr)

### Kubernetes
- [Kubernetes Releases](https://kubernetes.io/releases/)
- [Kubernetes v1.35 Announcement](https://kubernetes.io/blog/2025/12/17/kubernetes-v1-35-release/)
- [Kubernetes EOL Policy](https://endoflife.date/kubernetes)

### Go
- [Go Release History](https://go.dev/doc/devel/release)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)

### Python/RAG
- [OpenAI Model Retirement Notice](https://openai.com/index/retiring-gpt-4o-and-older-models/)
- [Weaviate v4 Migration Guide](https://weaviate.io/developers/weaviate/client-libraries/python)
- [LangChain Weaviate Integration](https://python.langchain.com/docs/integrations/vectorstores/weaviate/)

---

**Report Generated:** 2026-02-14 by Claude Code
**Next Audit Due:** 2026-05-14 (Quarterly review)

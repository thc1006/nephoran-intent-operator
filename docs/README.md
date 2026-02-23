# Documentation Directory Structure

This directory contains all project documentation organized by category for easy navigation and maintenance.

## Directory Structure

```
docs/
├── README.md                           # This file - documentation overview
├── adr/                               # Architecture Decision Records
│   ├── ADR-001-controller-runtime.md
│   ├── ADR-002-llm-integration.md
│   ├── ADR-003-rag-pipeline.md
│   ├── ADR-004-oran-interfaces.md
│   ├── ADR-005-security-architecture.md
│   └── ADR-006-adoption-of-go-1.24.md
├── api/                               # API documentation
│   ├── openapi.yaml
│   └── examples/
├── design/                            # Technical design documents
│   └── o2-interface.md                # O2 Interface Design
├── operations/                        # Operational documentation
│   └── runbook.md
├── reports/                           # Analysis and performance reports
│   ├── chaos-testing.md               # Chaos engineering results
│   ├── excellence-achievement.md      # Project excellence metrics
│   ├── performance-optimization.md    # Performance tuning results
│   └── resilience-validation.md       # System resilience analysis
├── security/                          # Security documentation
│   ├── docker-security-audit.md       # Container security audit
│   ├── hardening-guide.md             # Security hardening procedures
│   ├── implementation-summary.md      # Overall security implementation
│   ├── scanning-implementation.md     # Security scanning setup
│   ├── security-implementation-summary.md # Comprehensive security report
│   └── supply-chain-security.md       # Supply chain security measures
└── workflows/                         # CI/CD workflow documentation
    ├── cleanup-guide.md               # Workflow cleanup procedures
    ├── cleanup-report.md              # Cleanup implementation report
    ├── fixes-report.md                # Workflow fixes and improvements
    └── scripts/
        └── cleanup-workflows.sh       # Automated cleanup script
```

## Quick Navigation

### For Developers
- **Getting Started**: See [../QUICKSTART.md](../QUICKSTART.md) in project root
- **API Reference**: `api/` directory contains OpenAPI specifications
- **Architecture Decisions**: `adr/` directory for design rationale
- **Planner Integration Guide**: [`development/PLANNER-INTEGRATION-GUIDE.md`](development/PLANNER-INTEGRATION-GUIDE.md)
- **Root Cleanup Policy**: [`development/ROOT-CLEANUP-POLICY.md`](development/ROOT-CLEANUP-POLICY.md) - Repository structure guidelines

### For Operations Teams
- **Security Documentation**: `security/` directory for security policies and procedures
- **Workflow Management**: `workflows/` directory for CI/CD documentation
- **System Operations**: `operations/` directory for runbooks and procedures
- **LLM Provider Runbook**: [`operations/LLM_PROVIDER_RUNBOOK.md`](operations/LLM_PROVIDER_RUNBOOK.md)

### For Project Managers
- **Performance Reports**: `reports/` directory for metrics and analysis
- **Architecture Overview**: `architecture/` directory for technical designs
- **Security Compliance**: `security/` directory for compliance documentation
- **Conductor Loop Design**: [`architecture/CONDUCTOR-LOOP-DESIGN.md`](architecture/CONDUCTOR-LOOP-DESIGN.md)

## Documentation Categories

### Architecture Decision Records (ADRs)
Documents that capture important architectural decisions made during the project lifecycle. These provide context for why certain technical choices were made and serve as reference for future development.

#### Status tags
- **Active**: current source of truth.
- **Draft**: under development; verify before use.
- **Archive**: retained for history only; do not follow without review.

> 建議：新增或修改文件時，於開頭標註狀態（Active/Draft/Archive），過期內容移至 `docs/archive/` 並在此 README 保持索引。

### API Documentation
Complete API reference materials including OpenAPI specifications, examples, and integration guides. Essential for developers integrating with the Nephoran Intent Operator.

### Design Documents
Technical specifications and design documents that detail system architecture, interface specifications, and implementation approaches.

### Operations Documentation
Day-to-day operational procedures, runbooks, troubleshooting guides, and maintenance procedures for production deployments.

### Reports and Analysis
Performance benchmarks, security assessments, resilience testing results, and other analytical reports that document system capabilities and characteristics.

### Security Documentation
Comprehensive security documentation including implementation summaries, hardening guides, audit reports, and compliance materials. Critical for production security posture validation.

### Workflow Documentation
CI/CD pipeline documentation, automation scripts, and workflow management procedures. Essential for maintaining the development and deployment processes.

### Archive (history/legacy)
Use `docs/archive/`存放過期報告、一次性方案或被取代的流程，並在對應的主分類連結至最新版本；已移動：whitepapers、reports、presentations、knowledge-base。

## Document Maintenance

### Review Schedule
- **Security Documents**: Monthly review and updates
- **API Documentation**: Updated with each release
- **ADRs**: Added as needed, never modified once published
- **Reports**: Updated quarterly or after significant changes
- **Operations**: Updated as procedures change

### Contributing
When adding new documentation:
1. Place documents in the appropriate category directory
2. Update this README if adding new categories
3. Follow existing naming conventions (lowercase, hyphens for spaces)
4. Include frontmatter with metadata where appropriate
5. **Avoid adding files to repository root** - see [Root Cleanup Policy](development/ROOT-CLEANUP-POLICY.md) for guidelines

### Standards
- Use Markdown format for all documentation
- Include table of contents for documents > 1000 words
- Reference related documents using relative links
- Keep line length reasonable for readability (80-120 characters)
- Use semantic versioning for API documentation

## External References

### Project Root Documentation
- [CLAUDE.md](../CLAUDE.md) - Project overview and technical architecture
- [README.md](../README.md) - Main project README
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [SECURITY.md](../SECURITY.md) - Security policy and reporting
- [GitHub Releases](https://github.com/thc1006/nephoran-intent-operator/releases) - Version history (auto-generated)

### Related Resources
- [O-RAN Alliance Specifications](https://www.o-ran.org/specifications)
- [Nephio Documentation](https://nephio.org/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [OpenAPI Specification](https://swagger.io/specification/)

---

**Last Updated**: December 2024  
**Maintained By**: Nephoran Intent Operator Team  
**Review Frequency**: Quarterly

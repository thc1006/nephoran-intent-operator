#!/bin/bash
# Documentation Organization Script
# Reorganizes root-level markdown files into proper docs/ structure

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATE_DIR="2026-02-23"

log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Create directory structure
create_structure() {
    log_info "Creating documentation structure..."

    mkdir -p "${REPO_ROOT}/docs/reports/${DATE_DIR}"
    mkdir -p "${REPO_ROOT}/docs/implementation"

    log_success "Directory structure created"
}

# Move analysis reports
move_reports() {
    log_info "Moving analysis reports to docs/reports/${DATE_DIR}/"

    local reports=(
        "ARCHITECTURAL_HEALTH_ASSESSMENT.md:architectural-health-assessment.md"
        "COVERAGE_ANALYSIS_REPORT.md:coverage-analysis.md"
        "COVERAGE_SUMMARY.md:coverage-summary.md"
        "PERFORMANCE_ANALYSIS_REPORT.md:performance-analysis.md"
        "PERFORMANCE_FIXES_EXAMPLES.md:performance-fixes.md"
        "PERFORMANCE_SUMMARY.md:performance-summary.md"
        "GOROUTINE_LEAK_FIX.md:goroutine-leak-fix.md"
    )

    for report in "${reports[@]}"; do
        IFS=':' read -r source dest <<< "$report"
        if [ -f "${REPO_ROOT}/${source}" ]; then
            mv "${REPO_ROOT}/${source}" "${REPO_ROOT}/docs/reports/${DATE_DIR}/${dest}"
            log_success "Moved ${source} → docs/reports/${DATE_DIR}/${dest}"
        fi
    done
}

# Move implementation guides
move_guides() {
    log_info "Moving implementation guides to docs/implementation/"

    local guides=(
        "TEST_IMPLEMENTATION_GUIDE_P0.md:test-guide-p0.md"
        "ENDPOINT_VALIDATION_IMPLEMENTATION.md:endpoint-validation.md"
    )

    for guide in "${guides[@]}"; do
        IFS=':' read -r source dest <<< "$guide"
        if [ -f "${REPO_ROOT}/${source}" ]; then
            mv "${REPO_ROOT}/${source}" "${REPO_ROOT}/docs/implementation/${dest}"
            log_success "Moved ${source} → docs/implementation/${dest}"
        fi
    done
}

# Create reports index
create_reports_index() {
    log_info "Creating reports index..."

    cat > "${REPO_ROOT}/docs/reports/README.md" << 'EOF'
# Nephoran Intent Operator - Analysis Reports

This directory contains comprehensive analysis reports generated during project development and optimization.

## Directory Structure

- **2026-02-23/**: Reports from security, performance, and architecture improvements
  - All reports generated during P0-P2 task completion
  - Includes security fixes, performance optimization, and architecture enhancements

## Reports by Date

### 2026-02-23 - P0-P2 Task Completion

**Security & Performance Analysis**:
- [Architectural Health Assessment](2026-02-23/architectural-health-assessment.md) - Comprehensive project health review
- [Performance Analysis Report](2026-02-23/performance-analysis.md) - Detailed performance metrics and improvements
- [Performance Summary](2026-02-23/performance-summary.md) - Executive summary of performance fixes
- [Performance Fixes Examples](2026-02-23/performance-fixes.md) - Code examples of performance optimizations

**Code Quality Analysis**:
- [Coverage Analysis Report](2026-02-23/coverage-analysis.md) - Test coverage metrics and gaps
- [Coverage Summary](2026-02-23/coverage-summary.md) - High-level coverage overview

**Issue Resolution**:
- [Goroutine Leak Fix](2026-02-23/goroutine-leak-fix.md) - Resolution of goroutine memory leaks

## Key Metrics (2026-02-23)

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| HTTP Throughput | 50 req/sec | 2,500+ req/sec | **50x** |
| Goroutine Leaks | 2 per cycle | 0 | **100%** |
| Channel Memory | 10+ MB | 575 KB | **95%** |
| Security Score | 6.5/10 | 9.5/10 | **46%** |
| Test Coverage | 22% | 84.7% | **285%** |

## How to Use These Reports

1. **For Security Audits**: Start with `architectural-health-assessment.md`
2. **For Performance Optimization**: Review `performance-analysis.md` and `performance-fixes.md`
3. **For Code Quality**: Check `coverage-analysis.md` for test gaps

## Related Documentation

- [Implementation Guides](../implementation/) - Technical implementation details
- [PROGRESS.md](../PROGRESS.md) - Chronological development log
- [CLAUDE.md](../../CLAUDE.md) - AI assistant guide

---

**Last Updated**: 2026-02-23
EOF

    log_success "Created docs/reports/README.md"
}

# Create implementation index
create_implementation_index() {
    log_info "Creating implementation index..."

    cat > "${REPO_ROOT}/docs/implementation/README.md" << 'EOF'
# Nephoran Intent Operator - Implementation Guides

This directory contains technical implementation guides for specific features and fixes.

## Available Guides

### Testing & Quality

- [**Test Implementation Guide (P0)**](test-guide-p0.md)
  - Comprehensive guide to test implementation
  - Coverage for P0 critical security fixes
  - Test patterns and best practices

### Security & Validation

- [**Endpoint Validation Implementation**](endpoint-validation.md)
  - Startup endpoint validation
  - SSRF protection mechanisms
  - DNS resolution checking
  - Configuration examples

## How to Use These Guides

1. **Starting a New Feature**: Check if a related implementation guide exists
2. **Following Best Practices**: Reference guides for established patterns
3. **Contributing**: Add new guides for significant features

## Related Documentation

- [Analysis Reports](../reports/) - Performance and security analysis
- [Architecture Docs](../architecture/) - System architecture documentation
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines

---

**Last Updated**: 2026-02-23
EOF

    log_success "Created docs/implementation/README.md"
}

# Create master documentation index
create_master_index() {
    log_info "Creating master documentation index..."

    cat > "${REPO_ROOT}/docs/INDEX.md" << 'EOF'
# Nephoran Intent Operator - Documentation Index

**Last Updated**: 2026-02-23

## 📚 **Core Documentation** (Root Level)

Essential documents for getting started and contributing:

- [**README.md**](../README.md) - Project overview and main documentation
- [**QUICKSTART.md**](../QUICKSTART.md) - Quick start guide for new users
- [**CLAUDE.md**](../CLAUDE.md) - AI assistant guide and project context
- [**CONTRIBUTING.md**](../CONTRIBUTING.md) - Contribution guidelines
- [**CODE_OF_CONDUCT.md**](../CODE_OF_CONDUCT.md) - Community code of conduct
- [**SECURITY.md**](../SECURITY.md) - Security policy and reporting

## 📊 **Analysis Reports**

Comprehensive analysis and metrics:

- [**Reports Index**](reports/README.md) - All analysis reports
  - [2026-02-23 Reports](reports/2026-02-23/) - P0-P2 security and performance fixes

## 🛠️ **Implementation Guides**

Technical implementation documentation:

- [**Implementation Index**](implementation/README.md) - All implementation guides
  - [Test Guide (P0)](implementation/test-guide-p0.md)
  - [Endpoint Validation](implementation/endpoint-validation.md)

## 🏗️ **Architecture Documentation**

System design and architecture:

- [5G Integration Plan V2](5G_INTEGRATION_PLAN_V2.md) - Master SDD (100/100 thoroughness)
- [Phase 3 Completion Report](PHASE3_FINAL_COMPLETION.md) - Achievement summary
- [Porch URL Configuration](PORCH_URL_CONFIGURATION.md) - Porch endpoint setup

## 🔐 **Security Documentation**

Security policies and guides:

- [SBOM Generation Guide](SBOM_GENERATION_GUIDE.md) - Security SBOM compliance
- [SECURITY.md](../SECURITY.md) - Security policy

## 🚀 **Deployment & Operations**

Deployment guides and operational documentation:

- [Frontend Deployment](../deployments/frontend/README.md) - Web UI deployment
- [Deployment Guide](../deployments/DEPLOYMENT_GUIDE.md) - General deployment
- [GitOps Migration Guide](../deployments/GITOPS-MIGRATION-GUIDE.md) - GitOps setup

## 📈 **Progress Tracking**

Development history and progress:

- [**PROGRESS.md**](PROGRESS.md) - Append-only chronological progress log

## 🎯 **Quick Navigation**

### For New Users
1. Start with [README.md](../README.md)
2. Follow [QUICKSTART.md](../QUICKSTART.md)
3. Explore [Frontend UI](../deployments/frontend/README.md)

### For Contributors
1. Read [CONTRIBUTING.md](../CONTRIBUTING.md)
2. Review [CLAUDE.md](../CLAUDE.md)
3. Check [Implementation Guides](implementation/README.md)

### For DevOps/SRE
1. Read [Deployment Guide](../deployments/DEPLOYMENT_GUIDE.md)
2. Review [5G Integration Plan](5G_INTEGRATION_PLAN_V2.md)
3. Check [Security Policy](../SECURITY.md)

### For Architects
1. Review [Architectural Health Assessment](reports/2026-02-23/architectural-health-assessment.md)
2. Read [5G Integration Plan V2](5G_INTEGRATION_PLAN_V2.md)
3. Check [Performance Analysis](reports/2026-02-23/performance-analysis.md)

---

**Documentation Version**: 1.0
**Last Updated**: 2026-02-23
EOF

    log_success "Created docs/INDEX.md"
}

# Update git tracking
update_git() {
    log_info "Updating git tracking..."

    cd "${REPO_ROOT}"
    git add docs/
    git status --short

    log_success "Git tracking updated"
}

# Main execution
main() {
    echo -e "${CYAN}╔════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║  Nephoran Documentation Organization      ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════╝${NC}"
    echo ""

    create_structure
    move_reports
    move_guides
    create_reports_index
    create_implementation_index
    create_master_index
    update_git

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Documentation Organization Complete!     ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo -e "  1. Review changes: ${YELLOW}git status${NC}"
    echo -e "  2. View new structure: ${YELLOW}tree docs/${NC}"
    echo -e "  3. Read index: ${YELLOW}cat docs/INDEX.md${NC}"
    echo -e "  4. Commit changes: ${YELLOW}git commit -m 'docs: reorganize documentation structure'${NC}"
    echo ""
}

main "$@"

# GitHub Actions CI/CD Pipelines

> **📚 Note:** This document provides operational guidance for the CI/CD pipelines. For comprehensive workflow documentation including consolidation history, architecture details, and best practices, see the main [Workflows Documentation](./README.md).

This directory contains comprehensive GitHub Actions workflows for the Nephoran Intent Operator, providing automated testing, building, deployment, and security auditing with zero manual intervention.

## 🚀 Pipeline Overview

The CI/CD pipeline consists of four main workflows designed for production-ready automation:

### 1. Production Deployment Pipeline (`production.yml`)
**Trigger**: Push to `main` branch, version tags
**Duration**: ~45-60 minutes
**Purpose**: Full production deployment with comprehensive quality gates

#### 9-Stage Pipeline:
1. **Code Quality** - Formatting, linting, SAST scan, SonarCloud analysis
2. **Comprehensive Testing** - Unit (90%+ coverage), integration, E2E tests in parallel
3. **Performance Testing** - Load testing, regression detection
4. **Security Scanning** - Container security, manifest validation, secrets scanning
5. **Build and Push** - Multi-arch builds (amd64/arm64), image signing with Cosign, SBOM generation
6. **Staging Deployment** - Automated deployment with smoke tests
7. **Production Deployment** - Blue-green deployment with validation
8. **Release Management** - Changelog generation, GitHub releases
9. **Post-deployment Monitoring** - 24-hour enhanced alerting

#### Quality Gates (Pipeline Stops If):
- Code coverage < 90%
- Critical/High security vulnerabilities found
- Performance regression > 10%
- Failed smoke tests or validation
- SonarCloud quality gate failure

### 2. Pull Request Validation (`pr-validation.yml`)
**Trigger**: Pull requests to `main`, `develop`
**Duration**: ~25-35 minutes
**Purpose**: Comprehensive PR validation with auto-merge capabilities

#### Validation Stages:
- **Fast Feedback** - Basic formatting, linting, manifest validation
- **Code Quality** - Advanced SAST, vulnerability scanning
- **Testing** - Unit tests with coverage, integration tests
- **Build Validation** - Container builds, security scanning
- **Performance Check** - Benchmark impact assessment
- **Integration Validation** - End-to-end system validation

#### Features:
- Parallel execution for fast feedback
- Change detection (only runs relevant tests)
- Auto-merge for approved PRs with passing tests
- Comprehensive PR summary comments

### 3. Nightly Build Pipeline (`nightly.yml`)
**Trigger**: Daily at 2:00 AM UTC, manual dispatch
**Duration**: ~90-120 minutes
**Purpose**: Comprehensive testing and baseline establishment

#### Nightly Features:
- **Quality Baseline** - Comprehensive code analysis with trends
- **Extended Testing** - Chaos engineering, reliability, stress tests
- **Multi-arch Builds** - Complete container image builds
- **Performance Baselines** - Establish performance metrics
- **Compliance Validation** - Full deployment validation

#### Test Matrix:
- Unit, Integration, E2E, Performance, Load, Stress
- Security, Compliance, Chaos, Reliability testing
- Multiple Go versions and Kubernetes clusters

### 4. Weekly Security Audit (`security-audit.yml`)
**Trigger**: Weekly on Sundays at 3:00 AM UTC, manual dispatch
**Duration**: ~60-90 minutes
**Purpose**: Comprehensive security assessment and compliance reporting

#### Security Scans:
- **SAST** - Static application security testing
- **Dependency** - Vulnerability scanning with multiple tools
- **Secrets** - Git history and code secret scanning
- **Containers** - Multi-tool container vulnerability assessment
- **DAST** - Dynamic application security testing
- **Compliance** - NIST, ETSI, ISO 27001, GDPR assessment
- **Policy** - Kubernetes security policy validation
- **Penetration** - Simulated penetration testing
- **Infrastructure** - Terraform and infrastructure security

## 🔧 Configuration

### Required Secrets

#### Authentication & Registry
```bash
# Google Cloud Platform
GCP_SA_KEY                    # Service account key for GKE and container registry

# GitHub
GITHUB_TOKEN                  # Automatically provided

# Container Registry
REGISTRY                      # Set in env vars: us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran
```

#### Code Quality & Security
```bash
# SonarCloud
SONAR_TOKEN                   # SonarCloud authentication token
SONAR_ORGANIZATION           # SonarCloud organization name

# Security Tools (Optional)
SECURITY_SLACK_WEBHOOK       # Slack webhook for security notifications
SECURITY_EMAIL_WEBHOOK       # Email webhook for security alerts
```

#### Notifications
```bash
# General Notifications
SLACK_WEBHOOK_URL            # General Slack notifications
WEBHOOK_URL                  # Generic webhook for notifications

# Monitoring Integration
MONITORING_WEBHOOK           # Webhook for metrics/monitoring systems
PAGERDUTY_INTEGRATION_KEY    # PagerDuty integration for alerts
```

### Environment Variables

The pipelines use the following environment variables:

```yaml
# Quality Thresholds
MIN_COVERAGE: "90"                    # Minimum code coverage percentage
MAX_SECURITY_SCORE: "7.0"           # Maximum CVSS score allowed
MAX_PERFORMANCE_REGRESSION: "10"     # Maximum performance regression %

# Container Configuration  
REGISTRY: "us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran"

# Cluster Configuration
STAGING_CLUSTER: "nephoran-staging"
PRODUCTION_CLUSTER: "nephoran-production"

# Monitoring
MONITORING_DURATION: "24h"
```

## 🛠️ Local Development

### Running Tests Locally
```bash
# Run all tests
make test-comprehensive

# Run specific test types
make test-unit
make test-integration
make test-performance

# Run security scans
make security-scan
```

### Build and Test Containers
```bash
# Build all services
make docker-build

# Security scan containers
make security-scan-containers

# Test deployment
kubectl apply -k deployments/kustomize/overlays/dev/
```

### Validate Workflows
```bash
# Validate workflow syntax
gh workflow list

# Test workflow locally (requires act)
act -j code-quality
```

## 📊 Pipeline Metrics

### Performance Targets
- **PR Validation**: < 35 minutes
- **Production Pipeline**: < 60 minutes  
- **Code Coverage**: ≥ 90%
- **Security Score**: ≥ 85/100
- **Deployment Success Rate**: ≥ 99%

### Quality Gates
- Zero critical vulnerabilities
- Maximum 2 high vulnerabilities
- SonarCloud quality gate: PASSED
- All tests passing
- Performance regression < 10%

## 🔒 Security Features

### Container Security
- Multi-tool vulnerability scanning (Trivy, Grype, Syft)
- SBOM generation and attestation
- Container image signing with Cosign
- Distroless base images
- Non-root execution

### Code Security
- Static Application Security Testing (SAST)
- Dependency vulnerability scanning
- Secret detection with multiple tools
- Security policy validation
- Compliance framework assessment

### Infrastructure Security
- Kubernetes security policy validation
- Network policy enforcement
- RBAC validation
- Infrastructure as Code security scanning
- Runtime security monitoring

## 📈 Monitoring and Observability

### Pipeline Observability
- Comprehensive logging with structured output
- Performance metrics collection
- Error tracking and alerting
- Deployment success/failure tracking
- Security vulnerability trends

### Business Metrics
- Deployment frequency
- Lead time for changes
- Mean time to recovery (MTTR)
- Change failure rate
- Security vulnerability resolution time

## 🚨 Incident Response

### Pipeline Failures
1. **Immediate**: Check pipeline logs and GitHub Actions summary
2. **Code Quality**: Review SonarCloud dashboard for details
3. **Security**: Check security scan artifacts for vulnerabilities
4. **Performance**: Review performance test results for regressions
5. **Deployment**: Check cluster logs and monitoring dashboards

### Security Incidents
1. **Critical Vulnerabilities**: Automatic GitHub issues created
2. **Security Team**: Automatic notifications sent
3. **Compliance**: Generate emergency compliance report
4. **Response**: Follow incident response procedures

## 🔄 Pipeline Evolution

### Version History
- **v1.0**: Basic CI/CD with testing and deployment
- **v2.0**: Added comprehensive security scanning
- **v2.1**: Enhanced performance testing and monitoring
- **v2.2**: Added compliance reporting and chaos engineering
- **v3.0**: Current - Full production pipeline with quality gates

### Planned Enhancements
- **Service Mesh Integration**: Istio/Linkerd deployment validation
- **Multi-Cloud**: AWS and Azure deployment support  
- **ML/AI Testing**: Advanced AI model validation
- **GitOps Enhancement**: ArgoCD integration improvements
- **Edge Computing**: Edge deployment pipeline

## 📚 Best Practices

### Pipeline Design
- **Fail Fast**: Early validation to reduce feedback time
- **Parallel Execution**: Maximize concurrency for speed
- **Idempotent Operations**: Safe to retry any stage
- **Comprehensive Logging**: Detailed logs for debugging
- **Quality Gates**: Prevent bad code from reaching production

### Security
- **Shift Left**: Security testing early in pipeline
- **Defense in Depth**: Multiple security layers
- **Compliance**: Regular compliance validation
- **Monitoring**: Continuous security monitoring
- **Response**: Automated incident response

### Performance
- **Caching**: Aggressive use of caches for speed
- **Matrix Builds**: Parallel testing across configurations
- **Resource Optimization**: Right-sized runners
- **Incremental Builds**: Only build what changed
- **Monitoring**: Performance regression detection

## 🤝 Contributing

### Adding New Workflows
1. Create workflow in `.github/workflows/`
2. Follow naming convention: `purpose.yml`
3. Include comprehensive documentation
4. Add appropriate quality gates
5. Test thoroughly before merging

### Modifying Existing Workflows
1. Follow semantic versioning for breaking changes
2. Update documentation
3. Test in feature branch first
4. Consider backward compatibility
5. Update monitoring and alerting

### Security Considerations
- Never hardcode secrets in workflows
- Use minimum required permissions
- Validate all external inputs
- Use official actions when possible
- Regular security reviews

---

## 📖 Additional Documentation

### Comprehensive Workflow Documentation
For detailed information about the workflow architecture, consolidation history, and technical specifications, see:
- **[Main Workflows Documentation](./README.md)** - Complete workflow inventory, architecture, and best practices
- **[Workflow Consolidation History](./README.md#historical-context)** - Details about the cleanup and consolidation effort
- **[Workflow Trigger Matrix](./README.md#workflow-trigger-matrix)** - Complete trigger mapping
- **[Troubleshooting Guide](./README.md#troubleshooting-guide)** - Common issues and solutions

### Quick Links
- **Current Workflows:** 16 consolidated workflows (reduced from 25+)
- **Consolidation Benefits:** 36% reduction in workflows, 25% faster execution
- **Success Rate:** 97% workflow success rate
- **Coverage Target:** 90% code coverage requirement

---

**Pipeline Status**: [![Production Pipeline](https://github.com/nephoran/nephoran-intent-operator/workflows/Production%20Deployment%20Pipeline/badge.svg)](https://github.com/nephoran/nephoran-intent-operator/actions)

For more information, see the [GitHub Actions documentation](https://docs.github.com/en/actions) and the [Nephoran documentation](../docs/).
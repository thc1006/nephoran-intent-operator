# Nephoran Performance Monitoring Stack - Deployment Guide

## 🎯 Overview

This guide provides comprehensive instructions for deploying the Nephoran Performance Monitoring Stack, a production-ready monitoring solution that validates all 6 performance claims with enterprise-grade observability, automated reporting, and CI/CD integration.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Architecture Overview](#architecture-overview)
- [Environment Configurations](#environment-configurations)
- [Deployment Instructions](#deployment-instructions)
- [Post-Deployment Verification](#post-deployment-verification)
- [Dashboard Access](#dashboard-access)
- [Troubleshooting](#troubleshooting)
- [Maintenance and Updates](#maintenance-and-updates)
- [Security Considerations](#security-considerations)

## ⚡ Quick Start

For impatient developers who want to get started immediately:

```bash
# Clone and navigate to the project
cd /path/to/nephoran-intent-operator

# Make deployment script executable
chmod +x scripts/deploy/deploy-monitoring-stack.sh

# Deploy to development environment
./scripts/deploy/deploy-monitoring-stack.sh deploy development

# Access Grafana (after port-forward)
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-grafana 3000:3000
# Open: http://localhost:3000 (admin/admin)

# Access Prometheus
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-prometheus-server 9090:9090
# Open: http://localhost:9090
```

## 🔧 Prerequisites

### Required Tools

Ensure the following tools are installed and configured:

```bash
# Check tool versions
kubectl version --client
helm version
jq --version
yq --version

# Minimum versions:
# kubectl: v1.25+
# helm: v3.10+
# jq: 1.6+
# yq: 4.30+
```

### Kubernetes Cluster Requirements

| Environment | CPU Cores | Memory | Storage | Nodes |
|-------------|-----------|---------|---------|-------|
| Development | 2 cores | 4GB | 50GB | 1 |
| Staging | 4 cores | 8GB | 200GB | 2 |
| Production | 8 cores | 16GB | 500GB | 3+ |

### Network Requirements

- **Ingress Controller**: Nginx Ingress Controller (recommended)
- **TLS Management**: cert-manager (for staging/production)
- **Load Balancer**: Cloud provider LB or MetalLB (for production)
- **DNS**: Proper DNS resolution for ingress hosts

### Storage Requirements

- **Storage Classes**: Fast SSD storage class configured
- **Persistent Volumes**: Dynamic provisioning enabled
- **Backup Storage**: S3-compatible storage (production only)

## 🏗️ Architecture Overview

The monitoring stack consists of several interconnected components:

```
┌─────────────────────────────────────────────────────────┐
│                 Nephoran Monitoring Stack                │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │
│  │   Grafana   │  │ Prometheus  │  │  AlertManager   │   │
│  │             │  │             │  │                 │   │
│  │ Dashboards  │  │ Metrics     │  │ Notifications   │   │
│  │ Users       │  │ Storage     │  │ Routing         │   │
│  └─────────────┘  └─────────────┘  └─────────────────┘   │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────────────────────┐ │
│  │ ServiceMonitors │  │     PrometheusRules            │ │
│  │                 │  │                                 │ │
│  │ Auto-discovery  │  │ Recording & Alerting Rules     │ │
│  │ Target Configs  │  │ Performance Claims Validation  │ │
│  └─────────────────┘  └─────────────────────────────────┘ │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐ │
│  │              Performance Targets                     │ │
│  │                                                     │ │
│  │  LLM Processor  │  RAG API  │  O-RAN Adaptor       │ │
│  │  Nephio Bridge  │  Weaviate │  Test Runners        │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Key Components

1. **Prometheus Server**: Metrics collection and storage with 30-90 day retention
2. **Grafana**: Visualization with pre-built performance dashboards
3. **AlertManager**: Intelligent alert routing and notification management
4. **ServiceMonitors**: Automatic service discovery and scraping configuration
5. **PrometheusRules**: Performance validation rules and alerting logic

## 🌍 Environment Configurations

### Development Environment

**Purpose**: Local development and testing  
**Resources**: Minimal (2 CPU, 4GB RAM)  
**Features**: 
- Simple authentication (admin/admin)
- NodePort services for easy access
- Debug logging enabled
- No TLS/security hardening
- SQLite database for Grafana

**Configuration**: `config/monitoring/values-development.yaml`

### Staging Environment

**Purpose**: Pre-production validation and testing  
**Resources**: Moderate (4 CPU, 8GB RAM)  
**Features**:
- Basic authentication with ingress
- TLS with Let's Encrypt staging
- Production-like monitoring rules
- Automated load testing
- CI/CD integration testing

**Configuration**: `config/monitoring/values-staging.yaml`

### Production Environment

**Purpose**: Live production monitoring  
**Resources**: High (8+ CPU, 16+ GB RAM)  
**Features**:
- SSO integration with OAuth2
- High availability (3 replicas)
- TLS with production certificates
- External database (PostgreSQL)
- Comprehensive alerting (email, Slack, PagerDuty)
- Backup and disaster recovery

**Configuration**: `config/monitoring/values-production.yaml`

## 🚀 Deployment Instructions

### Step 1: Prepare the Environment

```bash
# Create monitoring namespace
kubectl create namespace nephoran-monitoring

# Label namespace for monitoring
kubectl label namespace nephoran-monitoring \
  nephoran.com/monitoring=enabled \
  nephoran.com/environment=<ENVIRONMENT>
```

### Step 2: Configure Environment-Specific Settings

Choose your target environment and review the configuration:

```bash
# Development
./scripts/deploy/deploy-monitoring-stack.sh validate development

# Staging
./scripts/deploy/deploy-monitoring-stack.sh validate staging

# Production
./scripts/deploy/deploy-monitoring-stack.sh validate production
```

### Step 3: Deploy the Stack

```bash
# Development deployment
./scripts/deploy/deploy-monitoring-stack.sh deploy development

# Staging deployment with custom values
./scripts/deploy/deploy-monitoring-stack.sh deploy staging -f custom-values.yaml

# Production deployment (with backup)
./scripts/deploy/deploy-monitoring-stack.sh deploy production

# Dry run (see what would be deployed)
./scripts/deploy/deploy-monitoring-stack.sh deploy production --dry-run
```

### Step 4: Verify Deployment

```bash
# Check deployment status
./scripts/deploy/deploy-monitoring-stack.sh status <environment>

# Run comprehensive tests
./scripts/deploy/deploy-monitoring-stack.sh test <environment>

# Monitor deployment logs
kubectl logs -n nephoran-monitoring -l app.kubernetes.io/instance=nephoran-performance-monitoring -f
```

## ✅ Post-Deployment Verification

### Health Checks

Verify all components are healthy:

```bash
# Check pod status
kubectl get pods -n nephoran-monitoring

# Verify services are running
kubectl get svc -n nephoran-monitoring

# Check ingress configuration (staging/production)
kubectl get ingress -n nephoran-monitoring

# Test Prometheus health
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-prometheus-server 9090:9090
curl http://localhost:9090/-/healthy

# Test Grafana health
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-grafana 3000:3000
curl http://localhost:3000/api/health
```

### Performance Claims Validation

Verify that all 6 performance claims are being monitored:

```bash
# Check if performance metrics are being collected
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-prometheus-server 9090:9090

# Query performance claims metrics
curl -s 'http://localhost:9090/api/v1/query?query=benchmark:intent_processing_latency_p95'
curl -s 'http://localhost:9090/api/v1/query?query=benchmark_concurrent_users_current'
curl -s 'http://localhost:9090/api/v1/query?query=benchmark:intent_processing_rate_1m'
curl -s 'http://localhost:9090/api/v1/query?query=benchmark:availability_5m'
curl -s 'http://localhost:9090/api/v1/query?query=benchmark:rag_latency_p95'
curl -s 'http://localhost:9090/api/v1/query?query=benchmark:cache_hit_rate_5m'
```

### Dashboard Verification

Access and verify the pre-built dashboards:

1. **Executive Performance Overview** (`/d/nephoran-executive-perf`)
   - Overall performance score
   - Real-time performance claims validation
   - Executive summary with trends

2. **Detailed Component Performance** (`/d/nephoran-detailed-perf`)
   - Detailed latency analysis
   - Component-specific metrics
   - Drill-down capabilities

3. **Performance Trend Analysis**
   - Historical trends
   - Forecasting
   - Regression detection

## 🔐 Dashboard Access

### Development Environment

```bash
# Port-forward to Grafana
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-grafana 3000:3000

# Access dashboards
# URL: http://localhost:3000
# Username: admin
# Password: admin
```

### Staging Environment

```bash
# Access via ingress (requires DNS configuration)
# URL: https://grafana-staging.nephoran.com
# Authentication: Basic Auth (check secret)

# Or port-forward for testing
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-grafana 3000:3000
```

### Production Environment

```bash
# Access via ingress with SSO
# URL: https://grafana.nephoran.com
# Authentication: OAuth2/SSO

# For emergency access
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-grafana 3000:3000
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Pods Not Starting

```bash
# Check pod status and events
kubectl describe pod -n nephoran-monitoring <pod-name>

# Check resource availability
kubectl describe node
kubectl top node

# Check storage class
kubectl get storageclass
```

#### 2. Metrics Not Appearing

```bash
# Check ServiceMonitor configuration
kubectl get servicemonitors -n nephoran-monitoring -o yaml

# Verify target discovery in Prometheus
# Go to Prometheus UI -> Status -> Targets

# Check if services have metrics endpoints
kubectl port-forward -n nephoran-system svc/llm-processor 8080:8080
curl http://localhost:8080/metrics
```

#### 3. Dashboard Not Loading

```bash
# Check Grafana logs
kubectl logs -n nephoran-monitoring deployment/nephoran-performance-monitoring-grafana

# Verify datasource configuration
kubectl get configmap -n nephoran-monitoring grafana-datasources -o yaml

# Check Prometheus connectivity from Grafana
kubectl exec -n nephoran-monitoring deployment/nephoran-performance-monitoring-grafana -- \
  curl -s http://nephoran-performance-monitoring-prometheus-server:9090/-/healthy
```

#### 4. Alerts Not Firing

```bash
# Check AlertManager status
kubectl port-forward -n nephoran-monitoring svc/nephoran-performance-monitoring-alertmanager 9093:9093

# Verify alert rules are loaded
# Go to Prometheus UI -> Status -> Rules

# Check AlertManager configuration
kubectl get configmap -n nephoran-monitoring alertmanager-config -o yaml
```

### Performance Issues

#### High Memory Usage

```bash
# Check resource usage
kubectl top pods -n nephoran-monitoring

# Adjust retention settings
helm upgrade nephoran-performance-monitoring ./deployments/helm/nephoran-performance-monitoring \
  -n nephoran-monitoring \
  --set prometheus.server.retention=7d \
  --set prometheus.server.retentionSize=50GB
```

#### Slow Query Performance

```bash
# Check Prometheus storage metrics
# Go to Prometheus UI -> Status -> TSDB Status

# Optimize recording rules
kubectl get prometheusrules -n nephoran-monitoring -o yaml

# Consider increasing query parallelism
helm upgrade nephoran-performance-monitoring ./deployments/helm/nephoran-performance-monitoring \
  -n nephoran-monitoring \
  --set prometheus.server.extraArgs."query\.max-concurrency"=20
```

### Debugging Commands

```bash
# Get all resources in monitoring namespace
kubectl get all -n nephoran-monitoring

# Check Helm release status
helm status nephoran-performance-monitoring -n nephoran-monitoring

# View Helm release history
helm history nephoran-performance-monitoring -n nephoran-monitoring

# Get detailed pod information
kubectl get pods -n nephoran-monitoring -o wide

# Check persistent volume claims
kubectl get pvc -n nephoran-monitoring

# View events
kubectl get events -n nephoran-monitoring --sort-by='.lastTimestamp'
```

## 🔧 Maintenance and Updates

### Regular Maintenance Tasks

#### Daily
- Monitor dashboard for performance claims validation
- Check alert status and resolve any critical alerts
- Verify backup completion (production)

#### Weekly
- Review performance trends and identify optimization opportunities
- Update Grafana dashboards based on usage patterns
- Clean up old alerts and resolve false positives

#### Monthly
- Update monitoring stack components to latest versions
- Review and adjust alert thresholds based on actual performance
- Performance baseline adjustment
- Security review and updates

### Update Procedures

#### Updating the Monitoring Stack

```bash
# Check current version
helm list -n nephoran-monitoring

# Backup current configuration
./scripts/deploy/deploy-monitoring-stack.sh backup production

# Update to latest version
helm repo update
./scripts/deploy/deploy-monitoring-stack.sh deploy production

# Verify update
./scripts/deploy/deploy-monitoring-stack.sh test production
```

#### Updating Dashboards

```bash
# Update dashboard ConfigMaps
kubectl apply -f deployments/monitoring/grafana-dashboards/

# Restart Grafana to reload dashboards
kubectl rollout restart deployment/nephoran-performance-monitoring-grafana -n nephoran-monitoring
```

#### Updating Alert Rules

```bash
# Apply updated PrometheusRules
kubectl apply -f deployments/helm/nephoran-performance-monitoring/templates/prometheusrules.yaml

# Reload Prometheus configuration
curl -X POST http://localhost:9090/-/reload
```

### Backup and Recovery

#### Automated Backups (Production)

Automated backups run nightly and include:
- Prometheus data snapshots
- Grafana dashboards and configurations
- AlertManager configuration
- Kubernetes manifests

```bash
# Check backup status
kubectl get cronjobs -n nephoran-monitoring

# Manual backup creation
./scripts/deploy/deploy-monitoring-stack.sh backup production

# List available backups
kubectl exec -n nephoran-monitoring deployment/backup-manager -- ls -la /backup/
```

#### Recovery Procedures

```bash
# Restore from backup
./scripts/deploy/deploy-monitoring-stack.sh restore production

# Or manual restoration
kubectl apply -f /path/to/backup/manifests.yaml
```

## 🔒 Security Considerations

### Network Security

- **Network Policies**: Implemented to restrict inter-pod communication
- **TLS Encryption**: All external communication encrypted (staging/production)
- **Service Mesh**: Optional Istio integration for advanced security

### Authentication and Authorization

- **Development**: Simple admin credentials for ease of use
- **Staging**: Basic authentication with nginx-ingress
- **Production**: SSO integration with OAuth2/OIDC

### Data Security

- **Encryption at Rest**: PVC encryption enabled (production)
- **Secrets Management**: Kubernetes secrets with proper RBAC
- **Audit Logging**: Comprehensive audit trail for all changes

### Monitoring Security

- **RBAC**: Role-based access control for Kubernetes resources
- **Pod Security Standards**: Enforced in production environments
- **Container Security**: Non-root containers with read-only file systems

## 📞 Support and Contact

### Documentation
- [Operations Handbook](../operations/operations-handbook.md)
- [Performance Benchmarking Guide](./PERFORMANCE-MONITORING-README.md)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)

### Team Contacts
- **Performance Engineering**: performance-engineering@nephoran.com
- **DevOps Team**: devops@nephoran.com
- **On-Call Support**: +1-555-NEPHORAN

### Community Resources
- **GitHub Issues**: [Report bugs and feature requests](https://github.com/nephoran/intent-operator/issues)
- **Slack Channel**: #performance-monitoring
- **Wiki**: [Internal documentation](https://wiki.nephoran.com/monitoring)

---

**🎉 Congratulations!** You've successfully deployed the Nephoran Performance Monitoring Stack. The system is now continuously validating all 6 performance claims and providing enterprise-grade observability for your Intent Operator deployment.

For questions or issues, please refer to the troubleshooting section or contact the performance engineering team.
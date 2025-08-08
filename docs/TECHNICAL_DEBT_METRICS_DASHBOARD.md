# Technical Debt Metrics Dashboard
## Real-time Monitoring of Code Quality and Maintenance Health

### Dashboard Overview

This document defines the metrics, visualizations, and alerts for monitoring technical debt and maintenance health in the Nephoran Intent Operator platform.

---

## Core Metrics Definitions

### 1. Technical Debt Metrics

```yaml
Technical Debt Ratio (TDR):
  Formula: (Estimated Remediation Time / Development Time) × 100
  Target: ≤5%
  Alert Threshold: >8%
  Critical Threshold: >12%
  
Cyclomatic Complexity:
  Formula: Number of linearly independent paths through code
  Target: ≤10 per function
  Alert Threshold: >10 per function
  Critical Threshold: >20 per function
  
Code Duplication Percentage:
  Formula: (Duplicate Lines / Total Lines) × 100
  Target: ≤7%
  Alert Threshold: >10%
  Critical Threshold: >15%
  
Test Coverage:
  Formula: (Tested Lines / Total Lines) × 100
  Target: ≥90%
  Alert Threshold: <85%
  Critical Threshold: <80%
```

### 2. Security Metrics

```yaml
High Severity Vulnerabilities:
  Target: 0
  Alert Threshold: ≥1
  Critical Threshold: ≥3
  
Medium Severity Vulnerabilities:
  Target: ≤2
  Alert Threshold: >5
  Critical Threshold: >10
  
Days Since Last Security Scan:
  Target: ≤1 day
  Alert Threshold: >1 day
  Critical Threshold: >3 days
  
SBOM Generation Success Rate:
  Target: 100%
  Alert Threshold: <95%
  Critical Threshold: <90%
```

### 3. Performance Metrics

```yaml
Build Time:
  Target: ≤5 minutes
  Alert Threshold: >7 minutes
  Critical Threshold: >10 minutes
  
Container Size:
  Target: ≤700MB
  Alert Threshold: >800MB
  Critical Threshold: >1GB
  
Test Execution Time:
  Target: ≤10 minutes
  Alert Threshold: >15 minutes
  Critical Threshold: >20 minutes
  
Memory Usage (Runtime):
  Target: ≤500MB
  Alert Threshold: >750MB
  Critical Threshold: >1GB
```

### 4. Operational Metrics

```yaml
CI/CD Success Rate:
  Target: ≥95%
  Alert Threshold: <90%
  Critical Threshold: <85%
  
Deployment Frequency:
  Target: ≥1 per day
  Alert Threshold: <3 per week
  Critical Threshold: <1 per week
  
Mean Time to Recovery (MTTR):
  Target: ≤30 minutes
  Alert Threshold: >1 hour
  Critical Threshold: >4 hours
  
Documentation Coverage:
  Target: ≥80%
  Alert Threshold: <70%
  Critical Threshold: <60%
```

---

## Dashboard Layout Specification

### Panel 1: Health Overview (Top Row)
```json
{
  "panels": [
    {
      "title": "Technical Debt Health Score",
      "type": "stat",
      "targets": "technical_debt_health_score",
      "thresholds": [5, 8, 12],
      "colors": ["green", "yellow", "red"]
    },
    {
      "title": "Security Posture Score", 
      "type": "stat",
      "targets": "security_posture_score",
      "thresholds": [90, 80, 70],
      "colors": ["green", "yellow", "red"]
    },
    {
      "title": "Build Performance Index",
      "type": "stat", 
      "targets": "build_performance_index",
      "thresholds": [100, 80, 60],
      "colors": ["green", "yellow", "red"]
    },
    {
      "title": "Code Quality Grade",
      "type": "stat",
      "targets": "code_quality_grade",
      "displayMode": "basic",
      "colorMode": "background"
    }
  ]
}
```

### Panel 2: Technical Debt Trends (Second Row)
```json
{
  "panels": [
    {
      "title": "Technical Debt Ratio Trend",
      "type": "graph",
      "targets": "technical_debt_ratio_7d",
      "yAxis": {"max": 15, "unit": "percent"},
      "thresholds": [
        {"value": 5, "color": "green"},
        {"value": 8, "color": "yellow"}, 
        {"value": 12, "color": "red"}
      ]
    },
    {
      "title": "Cyclomatic Complexity Distribution",
      "type": "histogram",
      "targets": "cyclomatic_complexity_distribution",
      "buckets": [1, 5, 10, 15, 20, 25]
    },
    {
      "title": "Code Coverage by Package",
      "type": "table",
      "targets": "test_coverage_by_package",
      "columns": ["Package", "Coverage %", "Trend", "Status"]
    }
  ]
}
```

### Panel 3: Security Monitoring (Third Row)
```json
{
  "panels": [
    {
      "title": "Vulnerability Count by Severity",
      "type": "bar",
      "targets": [
        "high_severity_vulnerabilities",
        "medium_severity_vulnerabilities", 
        "low_severity_vulnerabilities"
      ],
      "colors": ["red", "orange", "yellow"]
    },
    {
      "title": "Security Scan Results Timeline",
      "type": "graph",
      "targets": "security_scan_results_7d",
      "legend": ["Pass", "Fail", "Warning"]
    },
    {
      "title": "Dependency Security Health",
      "type": "heatmap",
      "targets": "dependency_security_matrix",
      "xAxis": "Package",
      "yAxis": "Severity Level"
    }
  ]
}
```

### Panel 4: Performance Monitoring (Fourth Row)
```json
{
  "panels": [
    {
      "title": "Build Time Trend",
      "type": "graph",
      "targets": "build_time_7d",
      "yAxis": {"unit": "minutes"},
      "thresholds": [
        {"value": 5, "color": "green"},
        {"value": 7, "color": "yellow"},
        {"value": 10, "color": "red"}
      ]
    },
    {
      "title": "Container Size Evolution",
      "type": "graph",
      "targets": "container_size_7d",
      "yAxis": {"unit": "MB"},
      "thresholds": [
        {"value": 700, "color": "green"},
        {"value": 800, "color": "yellow"},
        {"value": 1000, "color": "red"}
      ]
    },
    {
      "title": "Test Execution Performance",
      "type": "table",
      "targets": "test_execution_metrics",
      "columns": ["Test Suite", "Duration", "Trend", "Status"]
    }
  ]
}
```

### Panel 5: Operational Excellence (Fifth Row)
```json
{
  "panels": [
    {
      "title": "CI/CD Pipeline Success Rate",
      "type": "graph",
      "targets": "cicd_success_rate_7d",
      "yAxis": {"max": 100, "unit": "percent"},
      "thresholds": [95, 90, 85]
    },
    {
      "title": "Deployment Frequency", 
      "type": "stat",
      "targets": "deployment_frequency_7d",
      "unit": "deployments/day"
    },
    {
      "title": "MTTR Trend",
      "type": "graph",
      "targets": "mttr_7d",
      "yAxis": {"unit": "minutes"},
      "thresholds": [30, 60, 240]
    },
    {
      "title": "Documentation Health",
      "type": "gauge",
      "targets": "documentation_coverage",
      "min": 0,
      "max": 100,
      "thresholds": [80, 70, 60]
    }
  ]
}
```

---

## Metric Collection Implementation

### Prometheus Metrics Configuration

```yaml
# prometheus-technical-debt.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "technical_debt_rules.yml"

scrape_configs:
  - job_name: 'nephoran-quality-metrics'
    static_configs:
      - targets: ['quality-exporter:8080']
    scrape_interval: 30s
    metrics_path: '/metrics'
    
  - job_name: 'nephoran-build-metrics'
    static_configs:
      - targets: ['build-metrics:8080']
    scrape_interval: 60s
    
  - job_name: 'nephoran-security-metrics'
    static_configs:
      - targets: ['security-scanner:8080']
    scrape_interval: 300s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Custom Metrics Exporter

```go
// pkg/monitoring/technical_debt_exporter.go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    technicalDebtRatio = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "nephoran_technical_debt_ratio",
            Help: "Technical debt ratio percentage",
        },
        []string{"package", "component"},
    )
    
    cyclomaticComplexity = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nephoran_cyclomatic_complexity",
            Help: "Cyclomatic complexity distribution",
            Buckets: []float64{1, 5, 10, 15, 20, 25, 30},
        },
        []string{"package", "function"},
    )
    
    codeCoverage = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "nephoran_code_coverage_percent", 
            Help: "Test coverage percentage by package",
        },
        []string{"package", "test_type"},
    )
    
    securityVulnerabilities = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "nephoran_security_vulnerabilities",
            Help: "Number of security vulnerabilities by severity",
        },
        []string{"severity", "component"},
    )
    
    buildDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "nephoran_build_duration_seconds",
            Help: "Build duration in seconds",
            Buckets: prometheus.LinearBuckets(60, 60, 10), // 1min to 10min
        },
        []string{"stage", "result"},
    )
)

func RecordTechnicalDebtMetrics(packageName, component string, ratio float64) {
    technicalDebtRatio.WithLabelValues(packageName, component).Set(ratio)
}

func RecordComplexityMetrics(packageName, functionName string, complexity float64) {
    cyclomaticComplexity.WithLabelValues(packageName, functionName).Observe(complexity)
}

func RecordCoverageMetrics(packageName, testType string, coverage float64) {
    codeCoverage.WithLabelValues(packageName, testType).Set(coverage)
}

func RecordSecurityMetrics(severity, component string, count float64) {
    securityVulnerabilities.WithLabelValues(severity, component).Set(count)
}

func RecordBuildMetrics(stage, result string, duration float64) {
    buildDuration.WithLabelValues(stage, result).Observe(duration)
}
```

### Alert Rules Configuration

```yaml
# technical_debt_rules.yml
groups:
  - name: technical_debt_alerts
    rules:
    
    # Technical Debt Alerts
    - alert: HighTechnicalDebtRatio
      expr: nephoran_technical_debt_ratio > 8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High technical debt ratio detected"
        description: "Package {{ $labels.package }} has technical debt ratio of {{ $value }}%"
        
    - alert: CriticalTechnicalDebtRatio  
      expr: nephoran_technical_debt_ratio > 12
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Critical technical debt ratio"
        description: "Package {{ $labels.package }} has critical debt ratio of {{ $value }}%"
    
    # Code Quality Alerts
    - alert: LowTestCoverage
      expr: nephoran_code_coverage_percent < 85
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Test coverage below threshold"
        description: "Package {{ $labels.package }} coverage is {{ $value }}%"
        
    - alert: HighComplexity
      expr: histogram_quantile(0.95, nephoran_cyclomatic_complexity) > 15
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High cyclomatic complexity detected"
        description: "95th percentile complexity is {{ $value }}"
    
    # Security Alerts  
    - alert: HighSecurityVulnerabilities
      expr: sum(nephoran_security_vulnerabilities{severity="high"}) > 0
      for: 0m
      labels:
        severity: critical
      annotations:
        summary: "High severity vulnerabilities detected"
        description: "Found {{ $value }} high severity vulnerabilities"
        
    - alert: MediumSecurityVulnerabilities
      expr: sum(nephoran_security_vulnerabilities{severity="medium"}) > 5
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Multiple medium severity vulnerabilities"
        description: "Found {{ $value }} medium severity vulnerabilities"
    
    # Performance Alerts
    - alert: SlowBuildTime
      expr: histogram_quantile(0.95, nephoran_build_duration_seconds) > 420  # 7 minutes
      for: 3m
      labels:
        severity: warning
      annotations:
        summary: "Build time degradation"
        description: "95th percentile build time is {{ $value }}s"
        
    - alert: CriticalBuildTime
      expr: histogram_quantile(0.95, nephoran_build_duration_seconds) > 600  # 10 minutes
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Critical build time degradation"
        description: "Build time has reached {{ $value }}s"
```

---

## Automated Reporting System

### Daily Quality Report

```bash
#!/bin/bash
# scripts/generate-daily-quality-report.sh

REPORT_DATE=$(date +%Y-%m-%d)
REPORT_FILE="reports/daily-quality-report-${REPORT_DATE}.md"

cat > "${REPORT_FILE}" << EOF
# Daily Quality Report - ${REPORT_DATE}

## Technical Debt Summary
- Technical Debt Ratio: $(curl -s 'http://prometheus:9090/api/v1/query?query=avg(nephoran_technical_debt_ratio)' | jq -r '.data.result[0].value[1]')%
- Average Complexity: $(curl -s 'http://prometheus:9090/api/v1/query?query=avg(nephoran_cyclomatic_complexity)' | jq -r '.data.result[0].value[1]')
- Test Coverage: $(curl -s 'http://prometheus:9090/api/v1/query?query=avg(nephoran_code_coverage_percent)' | jq -r '.data.result[0].value[1]')%

## Security Status
- High Vulnerabilities: $(curl -s 'http://prometheus:9090/api/v1/query?query=sum(nephoran_security_vulnerabilities{severity="high"})' | jq -r '.data.result[0].value[1] // 0')
- Medium Vulnerabilities: $(curl -s 'http://prometheus:9090/api/v1/query?query=sum(nephoran_security_vulnerabilities{severity="medium"})' | jq -r '.data.result[0].value[1] // 0')

## Performance Metrics  
- Build Time (95th): $(curl -s 'http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95, nephoran_build_duration_seconds)' | jq -r '.data.result[0].value[1]')s
- Container Size: $(docker images nephoran/intent-operator:latest --format "table {{.Size}}" | tail -1)

## Recommendations
$(generate_recommendations)

EOF

echo "Daily quality report generated: ${REPORT_FILE}"
```

### Weekly Trend Analysis

```python
#!/usr/bin/env python3
# scripts/weekly-trend-analysis.py

import requests
import json
import datetime
import matplotlib.pyplot as plt
from datetime import datetime, timedelta

def query_prometheus(query, start_time, end_time):
    """Query Prometheus for range data"""
    url = "http://prometheus:9090/api/v1/query_range"
    params = {
        'query': query,
        'start': start_time.isoformat() + 'Z',
        'end': end_time.isoformat() + 'Z', 
        'step': '1h'
    }
    response = requests.get(url, params=params)
    return response.json()

def generate_trend_report():
    """Generate weekly trend analysis"""
    end_time = datetime.now()
    start_time = end_time - timedelta(days=7)
    
    # Technical debt trend
    debt_data = query_prometheus(
        'avg(nephoran_technical_debt_ratio)', 
        start_time, 
        end_time
    )
    
    # Coverage trend
    coverage_data = query_prometheus(
        'avg(nephoran_code_coverage_percent)', 
        start_time, 
        end_time
    )
    
    # Build time trend
    build_data = query_prometheus(
        'histogram_quantile(0.95, nephoran_build_duration_seconds)', 
        start_time, 
        end_time
    )
    
    # Generate plots
    create_trend_charts(debt_data, coverage_data, build_data)
    
    # Generate summary report
    create_summary_report(debt_data, coverage_data, build_data)

def create_trend_charts(debt_data, coverage_data, build_data):
    """Create trend visualization charts"""
    fig, (ax1, ax2, ax3) = plt.subplots(3, 1, figsize=(12, 10))
    
    # Technical debt trend
    if debt_data['data']['result']:
        timestamps = [datetime.fromtimestamp(float(x[0])) for x in debt_data['data']['result'][0]['values']]
        values = [float(x[1]) for x in debt_data['data']['result'][0]['values']]
        ax1.plot(timestamps, values, 'r-', label='Technical Debt %')
        ax1.axhline(y=5, color='g', linestyle='--', label='Target (5%)')
        ax1.axhline(y=8, color='y', linestyle='--', label='Warning (8%)')
        ax1.set_title('Technical Debt Ratio Trend')
        ax1.set_ylabel('Percentage')
        ax1.legend()
        ax1.grid(True)
    
    # Similar plots for coverage and build time...
    
    plt.tight_layout()
    plt.savefig(f'reports/weekly-trends-{datetime.now().strftime("%Y-%m-%d")}.png')
    plt.close()

if __name__ == "__main__":
    generate_trend_report()
```

---

## Integration with CI/CD Pipeline

### GitHub Actions Integration

```yaml
# .github/workflows/quality-dashboard.yml
name: Update Quality Dashboard

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  collect-metrics:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
        
    - name: Run code analysis
      run: |
        make lint
        make test
        make coverage
        
    - name: Collect technical debt metrics
      run: |
        go run scripts/collect-metrics.go
        
    - name: Security scan
      run: |
        make security-scan
        
    - name: Update metrics
      run: |
        curl -X POST http://pushgateway:9091/metrics/job/nephoran-quality \
          --data-binary @metrics/quality-metrics.txt
          
    - name: Generate reports
      run: |
        make quality-report
        
    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: quality-reports
        path: reports/
```

### Makefile Integration

```makefile
# Quality dashboard targets

.PHONY: dashboard-setup
dashboard-setup: ## Setup quality metrics dashboard
	@echo "Setting up quality metrics dashboard..."
	kubectl apply -f monitoring/prometheus-technical-debt.yaml
	kubectl apply -f monitoring/grafana-technical-debt-dashboard.yaml
	@echo "Dashboard available at: http://grafana.localhost/d/technical-debt"

.PHONY: collect-metrics
collect-metrics: ## Collect all quality metrics
	@echo "Collecting quality metrics..."
	mkdir -p reports/metrics
	go run scripts/collect-metrics.go > reports/metrics/quality-metrics.json
	go run scripts/security-metrics.go > reports/metrics/security-metrics.json
	go run scripts/performance-metrics.go > reports/metrics/performance-metrics.json

.PHONY: update-dashboard
update-dashboard: collect-metrics ## Update dashboard with latest metrics
	@echo "Updating dashboard metrics..."
	curl -X POST http://pushgateway:9091/metrics/job/nephoran-quality \
		--data-binary @reports/metrics/quality-metrics.txt

.PHONY: generate-report
generate-report: ## Generate comprehensive quality report
	@echo "Generating quality report..."
	mkdir -p reports
	scripts/generate-daily-quality-report.sh
	python3 scripts/weekly-trend-analysis.py
	@echo "Reports available in reports/ directory"

.PHONY: dashboard-status
dashboard-status: ## Check dashboard status
	@echo "Quality Dashboard Status"
	@echo "======================="
	@curl -s http://prometheus:9090/api/v1/targets | \
		jq -r '.data.activeTargets[] | select(.job | contains("quality")) | "\(.job): \(.health)"'
	@curl -s http://grafana:3000/api/health | jq '.'
```

---

## Dashboard Access and Permissions

### Role-Based Access Control

```yaml
# rbac-dashboard.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: quality-dashboard-viewer
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["*"]
  verbs: ["get", "list"]
  
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole  
metadata:
  name: quality-dashboard-admin
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]

---  
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: quality-dashboard-viewers
subjects:
- kind: User
  name: team-lead
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: developers
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: quality-dashboard-viewer
  apiGroup: rbac.authorization.k8s.io
```

### Grafana Dashboard Configuration

```json
{
  "dashboard": {
    "id": null,
    "title": "Nephoran Technical Debt Dashboard",
    "tags": ["nephoran", "technical-debt", "quality"],
    "timezone": "browser",
    "schemaVersion": 30,
    "version": 1,
    "time": {
      "from": "now-7d",
      "to": "now"
    },
    "timepicker": {
      "refresh_intervals": ["5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d"]
    },
    "templating": {
      "list": [
        {
          "name": "package",
          "type": "query",
          "query": "label_values(nephoran_technical_debt_ratio, package)",
          "refresh": 1,
          "multi": true,
          "includeAll": true,
          "allValue": ".*"
        }
      ]
    },
    "panels": [
      {
        "id": 1,
        "title": "Technical Debt Health Score",
        "type": "stat",
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "thresholds": {
              "steps": [
                {"color": "green", "value": null},
                {"color": "yellow", "value": 5},
                {"color": "red", "value": 8}
              ]
            }
          }
        },
        "targets": [
          {
            "expr": "avg(nephoran_technical_debt_ratio)",
            "refId": "A"
          }
        ]
      }
    ]
  }
}
```

---

## Maintenance and Updates

### Monthly Dashboard Review

```yaml
Monthly Tasks:
  1. Review metric definitions and thresholds
  2. Update alert rules based on team feedback
  3. Add new metrics for emerging concerns
  4. Optimize query performance
  5. Update documentation and training materials
  
Quarterly Tasks:
  1. Full dashboard redesign review
  2. Integration with new tools
  3. Performance optimization
  4. User experience improvements
  5. Technology stack updates
```

### Dashboard Evolution Strategy

```yaml
Phase 1 (Current):
  - Basic technical debt metrics
  - Security vulnerability tracking
  - Build performance monitoring
  - Test coverage visualization
  
Phase 2 (Next Quarter):
  - Predictive analytics
  - AI-driven insights
  - Automated recommendations
  - Custom alerting per team
  
Phase 3 (Future):
  - ML-based anomaly detection
  - Automated remediation triggers
  - Cross-project comparisons
  - ROI tracking and reporting
```

---

## Conclusion

This Technical Debt Metrics Dashboard provides comprehensive visibility into code quality, security posture, and maintenance health. With real-time monitoring, automated alerting, and trend analysis, teams can proactively manage technical debt and maintain high-quality codebases.

The dashboard serves as both a diagnostic tool for identifying issues and a strategic asset for making data-driven decisions about technical investments and priorities.

---

**Document Version**: 1.0.0  
**Last Updated**: January 2025  
**Dashboard URL**: http://grafana.nephoran.local/d/technical-debt  
**Maintenance**: Platform Engineering Team
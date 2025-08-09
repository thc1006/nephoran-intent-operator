# Monitoring Operations Runbook - Procedures and Troubleshooting Guide

## Executive Summary

This operations runbook provides comprehensive procedures for managing, maintaining, and troubleshooting the Nephoran Intent Operator monitoring and SLA validation systems. It serves as the definitive operational reference for ensuring continuous monitoring effectiveness, rapid incident response, and optimal system performance.

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Incident Response Procedures](#incident-response-procedures)
3. [Troubleshooting Guide](#troubleshooting-guide)
4. [Performance Tuning](#performance-tuning)
5. [Capacity Planning](#capacity-planning)
6. [Maintenance Procedures](#maintenance-procedures)
7. [Disaster Recovery](#disaster-recovery)
8. [Alert Management](#alert-management)
9. [Escalation Procedures](#escalation-procedures)
10. [Knowledge Base](#knowledge-base)

## Daily Operations

### Daily Monitoring Checklist

```yaml
daily_checklist:
  morning_procedures:
    - name: "system_health_check"
      time: "08:00 UTC"
      duration: "15 minutes"
      tasks:
        - verify_all_monitoring_components_healthy
        - check_overnight_alert_summary
        - validate_sla_metrics_current_status
        - review_capacity_utilization
        - verify_backup_completion_status
      success_criteria:
        - all_components_green_status
        - no_critical_alerts_outstanding
        - sla_metrics_within_target_ranges
        - capacity_utilization_below_80_percent
        - all_backups_completed_successfully
        
    - name: "performance_review"
      time: "08:30 UTC"
      duration: "15 minutes"  
      tasks:
        - review_24h_availability_report
        - analyze_latency_trends
        - check_throughput_patterns
        - identify_performance_anomalies
        - validate_baseline_drift
      deliverables:
        - daily_performance_summary
        - anomaly_investigation_tickets
        - trend_analysis_report
        
  midday_procedures:
    - name: "capacity_monitoring"
      time: "12:00 UTC"
      duration: "10 minutes"
      tasks:
        - check_storage_usage_trends
        - monitor_memory_consumption
        - validate_cpu_utilization_patterns
        - review_network_bandwidth_usage
        - assess_scaling_requirements
        
    - name: "alert_management_review"
      time: "12:15 UTC"
      duration: "15 minutes"
      tasks:
        - review_alert_effectiveness
        - update_alert_thresholds_if_needed
        - process_alert_feedback
        - optimize_notification_routing
        
  evening_procedures:
    - name: "daily_summary_preparation"
      time: "17:00 UTC"
      duration: "20 minutes"
      tasks:
        - generate_daily_sla_report
        - update_executive_dashboard
        - prepare_overnight_monitoring_brief
        - schedule_maintenance_windows
        - update_incident_tracking
      deliverables:
        - daily_sla_compliance_report
        - executive_summary_email
        - overnight_operations_brief
```

### Standard Operating Procedures

#### SLA Metrics Verification

```bash
#!/bin/bash
# Daily SLA metrics verification script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/monitoring/daily_sla_check.log"
PROMETHEUS_URL="http://prometheus.monitoring.svc.cluster.local:9090"
ALERT_WEBHOOK="https://alerts.nephoran.io/webhook"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_FILE}"
}

check_availability() {
    local query="sla:availability:5m"
    local threshold=99.95
    
    log "Checking availability SLA..."
    
    local result=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=${query}" | \
        jq -r '.data.result[0].value[1]' 2>/dev/null)
    
    if [[ -z "${result}" || "${result}" == "null" ]]; then
        log "ERROR: Could not retrieve availability metric"
        send_alert "availability_metric_missing" "critical"
        return 1
    fi
    
    local availability=$(echo "${result}" | cut -d. -f1-2)
    log "Current availability: ${availability}%"
    
    if (( $(echo "${availability} < ${threshold}" | bc -l) )); then
        log "WARNING: Availability ${availability}% below threshold ${threshold}%"
        send_alert "availability_sla_violation" "warning" \
            "Availability: ${availability}%, Target: ${threshold}%"
        return 1
    fi
    
    log "Availability SLA: PASS (${availability}% >= ${threshold}%)"
    return 0
}

check_latency() {
    local query="sla:latency:p95:5m"
    local threshold=2000  # 2 seconds in milliseconds
    
    log "Checking P95 latency SLA..."
    
    local result=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=${query}" | \
        jq -r '.data.result[0].value[1]' 2>/dev/null)
    
    if [[ -z "${result}" || "${result}" == "null" ]]; then
        log "ERROR: Could not retrieve latency metric"
        send_alert "latency_metric_missing" "critical"
        return 1
    fi
    
    local latency_ms=$(echo "${result} * 1000" | bc -l | cut -d. -f1)
    log "Current P95 latency: ${latency_ms}ms"
    
    if (( latency_ms > threshold )); then
        log "WARNING: P95 latency ${latency_ms}ms exceeds threshold ${threshold}ms"
        send_alert "latency_sla_violation" "warning" \
            "P95 Latency: ${latency_ms}ms, Target: <${threshold}ms"
        return 1
    fi
    
    log "Latency SLA: PASS (${latency_ms}ms < ${threshold}ms)"
    return 0
}

check_throughput() {
    local query="sla:throughput:1m"
    local threshold=45  # intents per minute
    
    log "Checking throughput SLA..."
    
    local result=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=${query}" | \
        jq -r '.data.result[0].value[1]' 2>/dev/null)
    
    if [[ -z "${result}" || "${result}" == "null" ]]; then
        log "ERROR: Could not retrieve throughput metric"
        send_alert "throughput_metric_missing" "critical"
        return 1
    fi
    
    local throughput=$(echo "${result}" | cut -d. -f1)
    log "Current throughput: ${throughput} intents/min"
    
    if (( throughput < threshold )); then
        log "WARNING: Throughput ${throughput} below threshold ${threshold}"
        send_alert "throughput_sla_violation" "warning" \
            "Throughput: ${throughput} intents/min, Target: >=${threshold} intents/min"
        return 1
    fi
    
    log "Throughput SLA: PASS (${throughput} >= ${threshold} intents/min)"
    return 0
}

send_alert() {
    local alert_type="$1"
    local severity="$2"
    local message="${3:-}"
    
    local payload=$(jq -n \
        --arg type "${alert_type}" \
        --arg severity "${severity}" \
        --arg message "${message}" \
        --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)" \
        '{
            "alert_type": $type,
            "severity": $severity,
            "message": $message,
            "timestamp": $timestamp,
            "source": "daily_sla_check"
        }')
    
    curl -X POST "${ALERT_WEBHOOK}" \
        -H "Content-Type: application/json" \
        -d "${payload}" \
        --max-time 10 || log "Failed to send alert: ${alert_type}"
}

generate_daily_report() {
    local report_date=$(date '+%Y-%m-%d')
    local report_file="/var/reports/daily_sla_${report_date}.json"
    
    log "Generating daily SLA report..."
    
    # Query for detailed metrics over the last 24 hours
    local availability_24h=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=avg_over_time(sla:availability:5m[24h])" | \
        jq -r '.data.result[0].value[1]')
    
    local latency_p95_24h=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=max_over_time(sla:latency:p95:5m[24h]) * 1000" | \
        jq -r '.data.result[0].value[1]')
    
    local throughput_avg_24h=$(curl -s "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=avg_over_time(sla:throughput:1m[24h])" | \
        jq -r '.data.result[0].value[1]')
    
    # Generate report
    jq -n \
        --arg date "${report_date}" \
        --arg availability "${availability_24h}" \
        --arg latency "${latency_p95_24h}" \
        --arg throughput "${throughput_avg_24h}" \
        --argjson availability_target 99.95 \
        --argjson latency_target 2000 \
        --argjson throughput_target 45 \
        '{
            "report_date": $date,
            "sla_metrics": {
                "availability": {
                    "measured": ($availability | tonumber | . * 100 | round / 100),
                    "target": $availability_target,
                    "compliant": (($availability | tonumber) >= ($availability_target / 100))
                },
                "latency_p95_ms": {
                    "measured": ($latency | tonumber | round),
                    "target": $latency_target,
                    "compliant": (($latency | tonumber) <= $latency_target)
                },
                "throughput_per_min": {
                    "measured": ($throughput | tonumber | round),
                    "target": $throughput_target,
                    "compliant": (($throughput | tonumber) >= $throughput_target)
                }
            },
            "overall_compliance": (
                (($availability | tonumber) >= ($availability_target / 100)) and
                (($latency | tonumber) <= $latency_target) and
                (($throughput | tonumber) >= $throughput_target)
            ),
            "generation_timestamp": now
        }' > "${report_file}"
    
    log "Daily report generated: ${report_file}"
}

main() {
    log "Starting daily SLA metrics verification"
    
    local exit_code=0
    
    check_availability || exit_code=1
    check_latency || exit_code=1
    check_throughput || exit_code=1
    
    generate_daily_report
    
    if [[ ${exit_code} -eq 0 ]]; then
        log "Daily SLA check: ALL CHECKS PASSED"
    else
        log "Daily SLA check: SOME CHECKS FAILED"
    fi
    
    return ${exit_code}
}

# Execute main function
main "$@"
```

#### Component Health Monitoring

```python
#!/usr/bin/env python3
"""
Component health monitoring script for Nephoran monitoring infrastructure
"""

import asyncio
import aiohttp
import logging
import json
import time
from datetime import datetime, timedelta
from dataclasses import dataclass
from typing import List, Dict, Optional
import sys

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/monitoring/component_health.log'),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

@dataclass
class HealthCheck:
    name: str
    url: str
    timeout: int
    expected_status: int = 200
    critical: bool = True
    
@dataclass 
class HealthResult:
    name: str
    healthy: bool
    response_time_ms: float
    status_code: Optional[int]
    error_message: Optional[str]
    timestamp: datetime

class ComponentHealthMonitor:
    def __init__(self, config_file='/etc/monitoring/health_checks.json'):
        self.config_file = config_file
        self.health_checks = []
        self.session = None
        self.load_config()
        
    def load_config(self):
        """Load health check configuration"""
        try:
            with open(self.config_file, 'r') as f:
                config = json.load(f)
                
            self.health_checks = [
                HealthCheck(**check) for check in config['health_checks']
            ]
            
            logger.info(f"Loaded {len(self.health_checks)} health checks")
            
        except Exception as e:
            logger.error(f"Failed to load config: {e}")
            # Use default configuration
            self.health_checks = self.get_default_health_checks()
            
    def get_default_health_checks(self) -> List[HealthCheck]:
        """Default health check configuration"""
        return [
            HealthCheck("prometheus", "http://prometheus:9090/-/healthy", 5, critical=True),
            HealthCheck("grafana", "http://grafana:3000/api/health", 5, critical=True),
            HealthCheck("jaeger", "http://jaeger:16687/", 5, critical=True),
            HealthCheck("loki", "http://loki:3100/ready", 5, critical=True),
            HealthCheck("nephoran-controller", "http://nephoran-controller:8080/healthz", 5, critical=True),
            HealthCheck("llm-processor", "http://llm-processor:8000/health", 10, critical=True),
            HealthCheck("alertmanager", "http://alertmanager:9093/-/healthy", 5, critical=False),
        ]
        
    async def check_component_health(self, health_check: HealthCheck) -> HealthResult:
        """Perform individual health check"""
        start_time = time.time()
        
        try:
            async with self.session.get(
                health_check.url,
                timeout=aiohttp.ClientTimeout(total=health_check.timeout)
            ) as response:
                response_time = (time.time() - start_time) * 1000
                
                healthy = response.status == health_check.expected_status
                
                return HealthResult(
                    name=health_check.name,
                    healthy=healthy,
                    response_time_ms=response_time,
                    status_code=response.status,
                    error_message=None if healthy else f"HTTP {response.status}",
                    timestamp=datetime.utcnow()
                )
                
        except asyncio.TimeoutError:
            return HealthResult(
                name=health_check.name,
                healthy=False,
                response_time_ms=(time.time() - start_time) * 1000,
                status_code=None,
                error_message="Timeout",
                timestamp=datetime.utcnow()
            )
            
        except Exception as e:
            return HealthResult(
                name=health_check.name,
                healthy=False,
                response_time_ms=(time.time() - start_time) * 1000,
                status_code=None,
                error_message=str(e),
                timestamp=datetime.utcnow()
            )
    
    async def perform_all_health_checks(self) -> List[HealthResult]:
        """Perform all health checks concurrently"""
        tasks = [
            self.check_component_health(check) 
            for check in self.health_checks
        ]
        
        results = await asyncio.gather(*tasks)
        return results
    
    def analyze_health_results(self, results: List[HealthResult]) -> Dict:
        """Analyze health check results and generate summary"""
        total_checks = len(results)
        healthy_checks = sum(1 for r in results if r.healthy)
        unhealthy_checks = [r for r in results if not r.healthy]
        
        critical_failures = [
            r for r in unhealthy_checks 
            if any(hc.critical for hc in self.health_checks if hc.name == r.name)
        ]
        
        overall_health = "HEALTHY" if len(critical_failures) == 0 else "UNHEALTHY"
        
        average_response_time = sum(r.response_time_ms for r in results) / total_checks
        
        summary = {
            "overall_health": overall_health,
            "total_components": total_checks,
            "healthy_components": healthy_checks,
            "unhealthy_components": len(unhealthy_checks),
            "critical_failures": len(critical_failures),
            "average_response_time_ms": round(average_response_time, 2),
            "check_timestamp": datetime.utcnow().isoformat(),
            "component_details": [
                {
                    "name": r.name,
                    "status": "healthy" if r.healthy else "unhealthy",
                    "response_time_ms": round(r.response_time_ms, 2),
                    "status_code": r.status_code,
                    "error": r.error_message
                }
                for r in results
            ],
            "failures": [
                {
                    "component": r.name,
                    "error": r.error_message,
                    "response_time_ms": round(r.response_time_ms, 2),
                    "critical": any(hc.critical for hc in self.health_checks if hc.name == r.name)
                }
                for r in unhealthy_checks
            ]
        }
        
        return summary
    
    async def send_alerts_for_failures(self, failures: List[Dict]):
        """Send alerts for component failures"""
        if not failures:
            return
            
        for failure in failures:
            severity = "critical" if failure["critical"] else "warning"
            await self.send_alert(
                alert_type="component_health_failure",
                severity=severity,
                message=f"Component {failure['component']} failed health check: {failure['error']}",
                metadata=failure
            )
    
    async def send_alert(self, alert_type: str, severity: str, message: str, metadata: Dict):
        """Send alert to webhook"""
        alert_webhook = "https://alerts.nephoran.io/webhook"
        
        payload = {
            "alert_type": alert_type,
            "severity": severity,
            "message": message,
            "metadata": metadata,
            "timestamp": datetime.utcnow().isoformat(),
            "source": "component_health_monitor"
        }
        
        try:
            async with self.session.post(
                alert_webhook,
                json=payload,
                timeout=aiohttp.ClientTimeout(total=10)
            ) as response:
                if response.status == 200:
                    logger.info(f"Alert sent successfully: {alert_type}")
                else:
                    logger.error(f"Failed to send alert: HTTP {response.status}")
                    
        except Exception as e:
            logger.error(f"Error sending alert: {e}")
    
    def save_health_report(self, summary: Dict):
        """Save health check summary to file"""
        timestamp = datetime.utcnow().strftime('%Y%m%d_%H%M%S')
        filename = f"/var/reports/component_health_{timestamp}.json"
        
        try:
            with open(filename, 'w') as f:
                json.dump(summary, f, indent=2)
            logger.info(f"Health report saved: {filename}")
        except Exception as e:
            logger.error(f"Failed to save health report: {e}")
    
    async def run_health_monitoring(self):
        """Main health monitoring execution"""
        logger.info("Starting component health monitoring")
        
        connector = aiohttp.TCPConnector(limit=20, limit_per_host=5)
        self.session = aiohttp.ClientSession(connector=connector)
        
        try:
            # Perform health checks
            results = await self.perform_all_health_checks()
            
            # Analyze results
            summary = self.analyze_health_results(results)
            
            # Log summary
            logger.info(
                f"Health check complete: {summary['overall_health']} - "
                f"{summary['healthy_components']}/{summary['total_components']} healthy"
            )
            
            # Send alerts for failures
            if summary['failures']:
                logger.warning(f"Found {len(summary['failures'])} component failures")
                await self.send_alerts_for_failures(summary['failures'])
            
            # Save report
            self.save_health_report(summary)
            
            # Print summary to stdout
            print(json.dumps(summary, indent=2))
            
            return summary['overall_health'] == "HEALTHY"
            
        finally:
            await self.session.close()

async def main():
    monitor = ComponentHealthMonitor()
    success = await monitor.run_health_monitoring()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    asyncio.run(main())
```

## Incident Response Procedures

### SLA Violation Response

```yaml
sla_violation_response:
  severity_levels:
    critical:
      definition: "SLA metric below target for >5 minutes"
      response_time: "5 minutes"
      escalation_time: "15 minutes"
      actions:
        - immediate_page_oncall_engineer
        - activate_incident_command_center
        - begin_automated_mitigation
        - notify_executive_stakeholders
        
    warning:
      definition: "SLA metric trending toward violation"
      response_time: "15 minutes"
      escalation_time: "1 hour"
      actions:
        - alert_operations_team
        - begin_proactive_investigation
        - prepare_mitigation_options
        - increase_monitoring_frequency
        
    info:
      definition: "SLA metric recovered or minor deviation"
      response_time: "1 hour"
      escalation_time: "4 hours"
      actions:
        - log_for_trend_analysis
        - update_monitoring_dashboard
        - conduct_brief_investigation

incident_command_structure:
  incident_commander:
    responsibilities:
      - overall_incident_coordination
      - stakeholder_communication
      - decision_making_authority
      - post_incident_review_scheduling
    selection_criteria: "senior_sre_or_principal_engineer"
    
  technical_lead:
    responsibilities:
      - technical_investigation_coordination
      - mitigation_strategy_development
      - system_recovery_execution
      - root_cause_analysis_initiation
    selection_criteria: "expert_in_affected_system"
    
  communications_lead:
    responsibilities:
      - internal_stakeholder_updates
      - customer_communication
      - status_page_management
      - documentation_maintenance
    selection_criteria: "technical_writer_or_product_manager"

response_workflow:
  detection_to_acknowledgment: "< 5 minutes"
  acknowledgment_to_initial_response: "< 10 minutes"
  initial_response_to_mitigation: "< 30 minutes"
  mitigation_to_resolution: "< 2 hours"
  resolution_to_post_mortem: "< 24 hours"
```

### Automated Response Procedures

```go
// Automated incident response system
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/nephoran/monitoring/pkg/alerts"
    "github.com/nephoran/monitoring/pkg/incidents"
    "github.com/nephoran/monitoring/pkg/mitigations"
)

type IncidentResponseSystem struct {
    alertManager      *alerts.Manager
    incidentManager   *incidents.Manager
    mitigationEngine  *mitigations.Engine
    
    // Configuration
    autoMitigationEnabled bool
    escalationTimeouts    map[string]time.Duration
}

type SLAViolation struct {
    Type        string    `json:"type"`        // availability, latency, throughput
    Severity    string    `json:"severity"`    // critical, warning, info
    CurrentValue float64   `json:"current_value"`
    TargetValue  float64   `json:"target_value"`
    Timestamp    time.Time `json:"timestamp"`
    Component    string    `json:"component"`
    Details      map[string]interface{} `json:"details"`
}

func (irs *IncidentResponseSystem) HandleSLAViolation(ctx context.Context, violation *SLAViolation) error {
    log.Printf("Handling SLA violation: %s %s", violation.Type, violation.Severity)
    
    // Create incident record
    incident, err := irs.incidentManager.CreateIncident(ctx, &incidents.Incident{
        Title:       fmt.Sprintf("%s SLA Violation - %s", violation.Type, violation.Component),
        Description: fmt.Sprintf("SLA violation detected: %s at %v, target %v", 
            violation.Type, violation.CurrentValue, violation.TargetValue),
        Severity:    violation.Severity,
        Component:   violation.Component,
        Timestamp:   violation.Timestamp,
        Metadata:    violation.Details,
    })
    
    if err != nil {
        return fmt.Errorf("failed to create incident: %w", err)
    }
    
    // Send immediate notifications
    if err := irs.sendImmediateNotifications(ctx, incident, violation); err != nil {
        log.Printf("Failed to send notifications: %v", err)
    }
    
    // Start automated mitigation if enabled and appropriate
    if irs.autoMitigationEnabled && violation.Severity == "critical" {
        go irs.attemptAutoMitigation(ctx, incident, violation)
    }
    
    // Schedule escalation if not resolved
    go irs.scheduleEscalation(ctx, incident, violation)
    
    return nil
}

func (irs *IncidentResponseSystem) sendImmediateNotifications(ctx context.Context, incident *incidents.Incident, violation *SLAViolation) error {
    notifications := make([]alerts.Notification, 0)
    
    switch violation.Severity {
    case "critical":
        // Page on-call engineer
        notifications = append(notifications, alerts.Notification{
            Type:     "page",
            Target:   "oncall-sre",
            Subject:  fmt.Sprintf("CRITICAL: %s", incident.Title),
            Message:  irs.formatIncidentMessage(incident, violation),
            Priority: "high",
        })
        
        // Slack critical channel
        notifications = append(notifications, alerts.Notification{
            Type:     "slack",
            Target:   "#incidents-critical",
            Subject:  incident.Title,
            Message:  irs.formatIncidentMessage(incident, violation),
            Priority: "high",
        })
        
        // Email executives for prolonged incidents
        if time.Since(violation.Timestamp) > 15*time.Minute {
            notifications = append(notifications, alerts.Notification{
                Type:     "email",
                Target:   "executives@nephoran.io",
                Subject:  fmt.Sprintf("Extended SLA Violation: %s", incident.Title),
                Message:  irs.formatExecutiveMessage(incident, violation),
                Priority: "high",
            })
        }
        
    case "warning":
        // Slack operations channel
        notifications = append(notifications, alerts.Notification{
            Type:     "slack",
            Target:   "#operations",
            Subject:  incident.Title,
            Message:  irs.formatIncidentMessage(incident, violation),
            Priority: "medium",
        })
        
    case "info":
        // Log only for trend analysis
        log.Printf("INFO: %s", incident.Title)
        return nil
    }
    
    // Send all notifications
    for _, notification := range notifications {
        if err := irs.alertManager.Send(ctx, &notification); err != nil {
            log.Printf("Failed to send notification %s: %v", notification.Type, err)
        }
    }
    
    return nil
}

func (irs *IncidentResponseSystem) attemptAutoMitigation(ctx context.Context, incident *incidents.Incident, violation *SLAViolation) {
    log.Printf("Attempting automated mitigation for incident %s", incident.ID)
    
    // Select appropriate mitigation strategy
    strategy := irs.mitigationEngine.SelectStrategy(violation.Type, violation.Component)
    if strategy == nil {
        log.Printf("No automated mitigation available for %s", violation.Type)
        return
    }
    
    // Execute mitigation
    result, err := irs.mitigationEngine.ExecuteStrategy(ctx, strategy, violation)
    if err != nil {
        log.Printf("Automated mitigation failed: %v", err)
        irs.incidentManager.AddNote(ctx, incident.ID, 
            fmt.Sprintf("Automated mitigation failed: %v", err))
        return
    }
    
    // Update incident with mitigation results
    irs.incidentManager.AddNote(ctx, incident.ID,
        fmt.Sprintf("Automated mitigation executed: %s", result.Description))
    
    if result.Success {
        log.Printf("Automated mitigation successful for incident %s", incident.ID)
        irs.incidentManager.UpdateStatus(ctx, incident.ID, "mitigating")
        
        // Schedule validation check
        go irs.scheduleMitigationValidation(ctx, incident, violation, 5*time.Minute)
    }
}

func (irs *IncidentResponseSystem) scheduleMitigationValidation(ctx context.Context, incident *incidents.Incident, violation *SLAViolation, delay time.Duration) {
    time.Sleep(delay)
    
    // Check if SLA has recovered
    currentValue := irs.getCurrentSLAValue(violation.Type, violation.Component)
    
    if irs.isSLACompliant(violation.Type, currentValue, violation.TargetValue) {
        log.Printf("SLA recovered after mitigation for incident %s", incident.ID)
        irs.incidentManager.ResolveIncident(ctx, incident.ID, "Automated mitigation successful")
        
        // Send recovery notification
        irs.alertManager.Send(ctx, &alerts.Notification{
            Type:     "slack",
            Target:   "#operations",
            Subject:  fmt.Sprintf("RESOLVED: %s", incident.Title),
            Message:  fmt.Sprintf("SLA recovered after automated mitigation. Current value: %v", currentValue),
            Priority: "medium",
        })
    } else {
        log.Printf("SLA has not recovered after mitigation for incident %s", incident.ID)
        irs.incidentManager.AddNote(ctx, incident.ID, 
            fmt.Sprintf("Mitigation validation failed. Current value: %v, Target: %v", currentValue, violation.TargetValue))
        
        // Escalate to human intervention
        irs.escalateToHuman(ctx, incident, violation)
    }
}
```

### Manual Response Procedures

```yaml
manual_response_procedures:
  availability_violations:
    immediate_actions:
      - name: "identify_failed_components"
        command: "kubectl get pods -n nephoran-system --field-selector=status.phase!=Running"
        timeout: "30s"
        
      - name: "check_resource_constraints"
        command: "kubectl top nodes && kubectl top pods -n nephoran-system"
        timeout: "30s"
        
      - name: "verify_network_connectivity"
        command: "curl -s http://nephoran-controller:8080/healthz"
        timeout: "10s"
        
      - name: "check_recent_deployments"
        command: "kubectl rollout history deployment/nephoran-controller -n nephoran-system"
        timeout: "30s"
        
    investigation_steps:
      - examine_system_logs_last_30_minutes
      - analyze_resource_utilization_trends
      - review_recent_configuration_changes
      - check_dependency_service_status
      - validate_network_policies_and_ingress
      
    mitigation_options:
      - restart_failed_pods
      - scale_up_replicas
      - rollback_recent_deployment
      - increase_resource_limits
      - bypass_failing_component
      
  latency_violations:
    immediate_actions:
      - name: "identify_slow_components"
        command: "curl -s 'http://prometheus:9090/api/v1/query?query=rate(nephoran_intent_processing_duration_seconds_sum[5m])/rate(nephoran_intent_processing_duration_seconds_count[5m])'"
        timeout: "10s"
        
      - name: "check_database_performance"
        command: "kubectl exec -it deployment/postgresql -- psql -c 'SELECT * FROM pg_stat_activity WHERE state = \"active\";'"
        timeout: "30s"
        
      - name: "analyze_llm_service_latency"
        command: "curl -s http://llm-processor:8000/metrics | grep latency"
        timeout: "10s"
        
    investigation_steps:
      - analyze_distributed_traces_in_jaeger
      - examine_database_slow_query_log
      - check_llm_service_response_times
      - review_resource_contention_metrics
      - investigate_network_latency_issues
      
    mitigation_options:
      - optimize_database_queries
      - scale_llm_service_horizontally
      - increase_processing_timeouts
      - enable_request_caching
      - redistribute_load_across_regions
      
  throughput_violations:
    immediate_actions:
      - name: "check_queue_depth"
        command: "kubectl exec -it deployment/nephoran-controller -- curl localhost:8080/metrics | grep queue_depth"
        timeout: "10s"
        
      - name: "verify_worker_scaling"
        command: "kubectl get hpa nephoran-controller -n nephoran-system"
        timeout: "10s"
        
      - name: "check_rate_limiting"
        command: "kubectl logs deployment/api-gateway -n nephoran-system | grep -i 'rate limit'"
        timeout: "30s"
        
    investigation_steps:
      - analyze_processing_pipeline_bottlenecks
      - check_horizontal_pod_autoscaler_behavior
      - examine_rate_limiting_configuration
      - review_resource_quota_constraints
      - investigate_external_dependency_limits
      
    mitigation_options:
      - increase_max_replicas_in_hpa
      - adjust_rate_limiting_thresholds
      - scale_database_connections
      - optimize_intent_processing_algorithm
      - implement_load_shedding_mechanisms
```

## Troubleshooting Guide

### Common Issues and Solutions

#### High Memory Usage in Prometheus

**Symptoms:**
- Prometheus pods being OOM killed
- Query performance degradation
- Missing metrics data

**Investigation Steps:**
```bash
# Check memory usage
kubectl top pods -n monitoring | grep prometheus

# Check Prometheus metrics
curl -s http://prometheus:9090/api/v1/query?query=prometheus_tsdb_symbol_table_size_bytes

# Check cardinality
curl -s http://prometheus:9090/api/v1/label/__name__/values | jq '. | length'

# Identify high-cardinality metrics
curl -s http://prometheus:9090/api/v1/query?query=topk\(10,count\ by\ \(__name__\)\(\{__name__=~\".+\"\}\)\)
```

**Solutions:**
1. **Reduce metric cardinality:**
```yaml
# Add to Prometheus configuration
metric_relabel_configs:
  - source_labels: [__name__]
    regex: 'go_.*|process_.*'
    action: drop
  - source_labels: [instance]
    regex: '(.+):[0-9]+'
    target_label: instance
    replacement: '${1}'
```

2. **Increase memory limits:**
```yaml
resources:
  limits:
    memory: "32Gi"
  requests:
    memory: "16Gi"
```

3. **Implement recording rules:**
```yaml
groups:
  - name: nephoran_recording_rules
    interval: 30s
    rules:
      - record: nephoran:availability:5m
        expr: |
          sum(rate(nephoran_health_check_success_total[5m])) /
          sum(rate(nephoran_health_check_total[5m])) * 100
```

#### Jaeger Query Performance Issues

**Symptoms:**
- Slow trace queries
- UI timeouts
- High CPU usage in Jaeger query service

**Investigation Steps:**
```bash
# Check Jaeger query service logs
kubectl logs deployment/jaeger-query -n monitoring

# Check Elasticsearch cluster health
kubectl exec -it elasticsearch-0 -- curl localhost:9200/_cluster/health

# Check index sizes
kubectl exec -it elasticsearch-0 -- curl localhost:9200/_cat/indices?v
```

**Solutions:**
1. **Optimize Elasticsearch indices:**
```bash
# Delete old indices
kubectl exec -it elasticsearch-0 -- curl -X DELETE localhost:9200/jaeger-span-2024.01.*

# Optimize index templates
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: jaeger-es-index-template
data:
  template.json: |
    {
      "index_patterns": ["jaeger-span-*"],
      "settings": {
        "number_of_shards": 3,
        "number_of_replicas": 1,
        "index.refresh_interval": "30s"
      }
    }
EOF
```

2. **Adjust sampling rates:**
```yaml
sampling:
  default_strategy:
    type: adaptive
    max_traces_per_second: 50
    sampling_store:
      type: memory
      max_buckets: 5
```

#### Grafana Dashboard Loading Issues

**Symptoms:**
- Dashboards loading slowly
- Panel timeouts
- "Query timeout" errors

**Investigation Steps:**
```bash
# Check Grafana logs
kubectl logs deployment/grafana -n monitoring

# Test Prometheus query performance
curl -w "@curl-format.txt" -s "http://prometheus:9090/api/v1/query?query=up"

# Check Grafana database
kubectl exec -it deployment/grafana -- sqlite3 /var/lib/grafana/grafana.db ".tables"
```

**Solutions:**
1. **Optimize queries:**
```yaml
# Use recording rules for complex queries
expr: nephoran:availability:5m  # Instead of complex aggregation

# Limit time ranges
time_range: "1h"  # Instead of "24h" for detailed views
```

2. **Increase timeouts:**
```yaml
# grafana.ini
[database]
query_timeout = 0

[dataproxy]
timeout = 300
```

### Performance Optimization Procedures

#### Database Optimization

```sql
-- PostgreSQL optimization for monitoring data
-- Create indices for time-series queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_metrics_timestamp 
  ON metrics (timestamp DESC) 
  WHERE timestamp > NOW() - INTERVAL '7 days';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_metrics_component_timestamp
  ON metrics (component, timestamp DESC)
  WHERE timestamp > NOW() - INTERVAL '7 days';

-- Partition large tables by time
CREATE TABLE metrics_2024_01 PARTITION OF metrics
  FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Auto-vacuum configuration
ALTER TABLE metrics SET (
  autovacuum_vacuum_scale_factor = 0.02,
  autovacuum_analyze_scale_factor = 0.01,
  autovacuum_vacuum_cost_limit = 1000
);

-- Statistics collection
ANALYZE metrics;

-- Query performance monitoring
SELECT 
  query,
  mean_time,
  calls,
  total_time,
  mean_time / calls as avg_per_call
FROM pg_stat_statements 
WHERE query LIKE '%metrics%'
ORDER BY mean_time DESC 
LIMIT 10;
```

#### Storage Optimization

```bash
#!/bin/bash
# Storage optimization script

# Compress old Prometheus data
find /prometheus -name "*.tsdb" -mtime +7 -exec gzip {} \;

# Clean up old log files
find /var/log/monitoring -name "*.log" -mtime +30 -delete

# Optimize filesystem
tune2fs -c 0 -i 0 /dev/sdb1  # Disable forced checks

# Configure log rotation
cat > /etc/logrotate.d/monitoring << 'EOF'
/var/log/monitoring/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    postrotate
        systemctl reload rsyslog
    endscript
}
EOF

# Monitor disk usage
df -h | awk '$5 > 80 { print "WARNING: " $1 " is " $5 " full" }'
```

## Capacity Planning

### Resource Forecasting

```python
#!/usr/bin/env python3
"""
Capacity planning and forecasting for monitoring infrastructure
"""

import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression
from sklearn.metrics import mean_squared_error, r2_score
import matplotlib.pyplot as plt
from datetime import datetime, timedelta
import json

class CapacityPlanner:
    def __init__(self, prometheus_client):
        self.prometheus = prometheus_client
        self.forecasting_periods = [30, 90, 180, 365]  # days
        
    def collect_resource_metrics(self, days_back=90):
        """Collect historical resource utilization data"""
        
        metrics_queries = {
            'cpu_usage': 'avg(rate(container_cpu_usage_seconds_total[5m]))',
            'memory_usage': 'avg(container_memory_working_set_bytes)',
            'disk_usage': 'avg(node_filesystem_size_bytes - node_filesystem_free_bytes)',
            'network_rx': 'avg(rate(node_network_receive_bytes_total[5m]))',
            'network_tx': 'avg(rate(node_network_transmit_bytes_total[5m]))',
            'metrics_ingestion_rate': 'rate(prometheus_tsdb_samples_appended_total[5m])',
            'query_rate': 'rate(prometheus_http_requests_total[5m])',
        }
        
        end_time = datetime.now()
        start_time = end_time - timedelta(days=days_back)
        
        historical_data = {}
        
        for metric_name, query in metrics_queries.items():
            data = self.prometheus.query_range(
                query=query,
                start_time=start_time,
                end_time=end_time,
                step='1h'
            )
            
            historical_data[metric_name] = data
            
        return historical_data
    
    def forecast_capacity_requirements(self, historical_data):
        """Generate capacity forecasts using linear regression"""
        
        forecasts = {}
        
        for metric_name, data in historical_data.items():
            if not data:
                continue
                
            # Prepare data for modeling
            df = pd.DataFrame(data)
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            df['days'] = (df['timestamp'] - df['timestamp'].min()).dt.days
            
            X = df[['days']].values
            y = df['value'].values
            
            # Fit linear regression model
            model = LinearRegression()
            model.fit(X, y)
            
            # Calculate model performance
            y_pred = model.predict(X)
            mse = mean_squared_error(y, y_pred)
            r2 = r2_score(y, y_pred)
            
            # Generate forecasts
            metric_forecasts = {}
            current_days = X[-1][0]
            
            for period in self.forecasting_periods:
                future_days = current_days + period
                predicted_value = model.predict([[future_days]])[0]
                
                # Calculate confidence intervals (simple approach)
                residuals = y - y_pred
                std_residual = np.std(residuals)
                confidence_interval = 1.96 * std_residual  # 95% CI
                
                metric_forecasts[f"{period}_days"] = {
                    "predicted_value": predicted_value,
                    "confidence_lower": predicted_value - confidence_interval,
                    "confidence_upper": predicted_value + confidence_interval,
                    "trend_slope": model.coef_[0],
                    "model_r2": r2
                }
            
            forecasts[metric_name] = {
                "current_value": y[-1],
                "model_performance": {"mse": mse, "r2": r2},
                "forecasts": metric_forecasts
            }
            
        return forecasts
    
    def generate_scaling_recommendations(self, forecasts):
        """Generate scaling recommendations based on forecasts"""
        
        recommendations = {}
        
        # Define capacity thresholds
        thresholds = {
            'cpu_usage': 0.7,           # 70% CPU utilization
            'memory_usage': 0.8,        # 80% memory utilization  
            'disk_usage': 0.85,         # 85% disk utilization
            'metrics_ingestion_rate': 100000,  # 100k samples/sec
        }
        
        for metric_name, forecast_data in forecasts.items():
            if metric_name not in thresholds:
                continue
                
            threshold = thresholds[metric_name]
            current_value = forecast_data['current_value']
            
            recommendations[metric_name] = {
                "current_utilization": current_value,
                "threshold": threshold,
                "recommendations": []
            }
            
            for period_name, forecast in forecast_data['forecasts'].items():
                predicted_value = forecast['predicted_value']
                days = int(period_name.split('_')[0])
                
                if predicted_value > threshold:
                    urgency = "immediate" if days <= 30 else "planned"
                    scaling_factor = predicted_value / current_value
                    
                    recommendation = {
                        "timeframe": f"{days} days",
                        "urgency": urgency,
                        "predicted_utilization": predicted_value,
                        "scaling_factor": scaling_factor,
                        "action": self.generate_scaling_action(metric_name, scaling_factor)
                    }
                    
                    recommendations[metric_name]["recommendations"].append(recommendation)
        
        return recommendations
    
    def generate_scaling_action(self, metric_name, scaling_factor):
        """Generate specific scaling actions"""
        
        actions = {
            'cpu_usage': f"Increase CPU limits by {int((scaling_factor - 1) * 100 + 20)}%",
            'memory_usage': f"Increase memory limits by {int((scaling_factor - 1) * 100 + 20)}%",
            'disk_usage': f"Increase storage capacity by {int((scaling_factor - 1) * 100 + 30)}%",
            'metrics_ingestion_rate': f"Scale Prometheus horizontally to {int(scaling_factor) + 1} replicas"
        }
        
        return actions.get(metric_name, f"Scale resources by factor of {scaling_factor:.2f}")
    
    def generate_capacity_report(self):
        """Generate comprehensive capacity planning report"""
        
        # Collect historical data
        historical_data = self.collect_resource_metrics()
        
        # Generate forecasts
        forecasts = self.forecast_capacity_requirements(historical_data)
        
        # Generate recommendations
        recommendations = self.generate_scaling_recommendations(forecasts)
        
        # Compile report
        report = {
            "generation_timestamp": datetime.utcnow().isoformat(),
            "analysis_period_days": 90,
            "forecasting_periods": self.forecasting_periods,
            "forecasts": forecasts,
            "scaling_recommendations": recommendations,
            "summary": self.generate_executive_summary(recommendations)
        }
        
        # Save report
        timestamp = datetime.utcnow().strftime('%Y%m%d_%H%M%S')
        filename = f"/var/reports/capacity_planning_{timestamp}.json"
        
        with open(filename, 'w') as f:
            json.dump(report, f, indent=2)
            
        return report
    
    def generate_executive_summary(self, recommendations):
        """Generate executive summary of capacity planning"""
        
        immediate_actions = 0
        planned_actions = 0
        
        for metric_recs in recommendations.values():
            for rec in metric_recs.get('recommendations', []):
                if rec['urgency'] == 'immediate':
                    immediate_actions += 1
                else:
                    planned_actions += 1
        
        return {
            "total_metrics_analyzed": len(recommendations),
            "immediate_actions_required": immediate_actions,
            "planned_actions_required": planned_actions,
            "overall_status": "action_required" if immediate_actions > 0 else "planning_mode",
            "next_review_date": (datetime.utcnow() + timedelta(days=30)).isoformat()
        }
```

### Storage Planning

```yaml
storage_planning:
  current_usage:
    prometheus: "2TB"
    jaeger: "800GB" 
    loki: "1.5TB"
    grafana: "50GB"
    total: "4.35TB"
    
  growth_projections:
    monthly_growth_rate: "15%"
    prometheus_retention: "30 days"
    jaeger_retention: "14 days"
    loki_retention: "30 days"
    
  6_month_projection:
    prometheus: "4TB"
    jaeger: "1.5TB"
    loki: "3TB"
    grafana: "100GB"
    total: "8.6TB"
    
  12_month_projection:
    prometheus: "8TB"
    jaeger: "3TB"
    loki: "6TB"
    grafana: "200GB"
    total: "17.2TB"
    
  storage_optimization:
    compression_ratio: "10:1"
    archival_storage: "s3_glacier"
    hot_storage_ssd: "30_days"
    warm_storage_hdd: "90_days"
    cold_storage_s3: "365_days"
    
scaling_triggers:
  storage_usage_warning: "80%"
  storage_usage_critical: "90%"
  ingest_rate_scaling: ">100k_samples_per_second"
  query_latency_scaling: ">5s_p95"
```

## Maintenance Procedures

### Scheduled Maintenance

```yaml
maintenance_schedule:
  daily:
    - name: "log_rotation"
      time: "02:00 UTC"
      duration: "10 minutes"
      impact: "none"
      
    - name: "metric_cleanup"
      time: "03:00 UTC"
      duration: "30 minutes"
      impact: "minimal"
      
    - name: "backup_verification"
      time: "04:00 UTC"
      duration: "15 minutes"
      impact: "none"
      
  weekly:
    - name: "certificate_renewal"
      day: "Sunday"
      time: "01:00 UTC"
      duration: "45 minutes"
      impact: "brief_connectivity_interruption"
      
    - name: "database_optimization"
      day: "Sunday"
      time: "02:00 UTC"  
      duration: "2 hours"
      impact: "performance_improvement"
      
  monthly:
    - name: "security_updates"
      week: "first_sunday"
      time: "01:00 UTC"
      duration: "4 hours"
      impact: "rolling_restart_required"
      
    - name: "capacity_planning_review"
      week: "second_monday"
      time: "09:00 UTC"
      duration: "2 hours"
      impact: "none"
      
  quarterly:
    - name: "disaster_recovery_test"
      month: "march_june_september_december"
      time: "planned_maintenance_window"
      duration: "8 hours"
      impact: "full_system_unavailable"
```

### Component Update Procedures

```bash
#!/bin/bash
# Safe component update procedure

set -euo pipefail

COMPONENT="${1:-}"
TARGET_VERSION="${2:-}"
MAINTENANCE_WINDOW="${3:-4h}"

if [[ -z "$COMPONENT" || -z "$TARGET_VERSION" ]]; then
    echo "Usage: $0 <component> <target_version> [maintenance_window]"
    exit 1
fi

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# Pre-update checks
perform_pre_update_checks() {
    log "Performing pre-update checks for $COMPONENT"
    
    # Check system health
    if ! kubectl get pods -n monitoring | grep -E "(Running|Completed)" > /dev/null; then
        log "ERROR: System not healthy, aborting update"
        exit 1
    fi
    
    # Verify backup completion
    if ! check_recent_backup; then
        log "ERROR: No recent backup found, aborting update"
        exit 1
    fi
    
    # Check available resources
    if ! check_resource_availability; then
        log "ERROR: Insufficient resources for update, aborting"
        exit 1
    fi
    
    log "Pre-update checks passed"
}

# Create backup before update
create_pre_update_backup() {
    log "Creating pre-update backup for $COMPONENT"
    
    kubectl create job "backup-${COMPONENT}-$(date +%Y%m%d%H%M)" \
        --from=cronjob/backup-${COMPONENT} -n monitoring
    
    # Wait for backup completion
    kubectl wait --for=condition=complete --timeout=30m \
        job/backup-${COMPONENT}-$(date +%Y%m%d%H%M) -n monitoring
        
    log "Pre-update backup completed"
}

# Perform rolling update
perform_rolling_update() {
    log "Starting rolling update of $COMPONENT to version $TARGET_VERSION"
    
    # Update image version
    kubectl set image deployment/${COMPONENT} \
        ${COMPONENT}=${COMPONENT}:${TARGET_VERSION} \
        -n monitoring
    
    # Wait for rollout to complete
    kubectl rollout status deployment/${COMPONENT} \
        --timeout=${MAINTENANCE_WINDOW} -n monitoring
        
    log "Rolling update completed"
}

# Perform post-update validation
perform_post_update_validation() {
    log "Performing post-update validation for $COMPONENT"
    
    # Wait for pods to be ready
    kubectl wait --for=condition=ready pod \
        -l app=${COMPONENT} --timeout=300s -n monitoring
    
    # Perform health checks
    case $COMPONENT in
        "prometheus")
            validate_prometheus_health
            ;;
        "grafana") 
            validate_grafana_health
            ;;
        "jaeger")
            validate_jaeger_health
            ;;
        *)
            validate_generic_health $COMPONENT
            ;;
    esac
    
    # Check SLA metrics
    if ! check_sla_metrics; then
        log "WARNING: SLA metrics degraded after update"
        return 1
    fi
    
    log "Post-update validation completed successfully"
}

validate_prometheus_health() {
    local endpoint="http://prometheus.monitoring.svc.cluster.local:9090"
    
    # Check API health
    if ! curl -f "${endpoint}/-/healthy" >/dev/null 2>&1; then
        log "ERROR: Prometheus health check failed"
        return 1
    fi
    
    # Check metrics ingestion
    local samples=$(curl -s "${endpoint}/api/v1/query?query=prometheus_tsdb_samples_appended_total" | \
        jq -r '.data.result[0].value[1]')
        
    if [[ "$samples" == "null" || "$samples" -lt 1 ]]; then
        log "ERROR: Prometheus not ingesting samples"
        return 1
    fi
    
    log "Prometheus health validation passed"
}

validate_grafana_health() {
    local endpoint="http://grafana.monitoring.svc.cluster.local:3000"
    
    # Check API health
    if ! curl -f "${endpoint}/api/health" >/dev/null 2>&1; then
        log "ERROR: Grafana health check failed"  
        return 1
    fi
    
    # Check database connection
    local db_status=$(curl -s "${endpoint}/api/health" | jq -r '.database')
    if [[ "$db_status" != "ok" ]]; then
        log "ERROR: Grafana database connection failed"
        return 1
    fi
    
    log "Grafana health validation passed"
}

# Rollback procedure
perform_rollback() {
    log "Performing rollback for $COMPONENT"
    
    kubectl rollout undo deployment/${COMPONENT} -n monitoring
    kubectl rollout status deployment/${COMPONENT} --timeout=600s -n monitoring
    
    # Validate rollback
    if perform_post_update_validation; then
        log "Rollback completed successfully"
    else
        log "ERROR: Rollback validation failed"
        exit 1
    fi
}

# Main execution
main() {
    log "Starting maintenance update for $COMPONENT to version $TARGET_VERSION"
    
    # Trap for cleanup on exit
    trap 'log "Update procedure interrupted"' INT TERM
    
    perform_pre_update_checks
    create_pre_update_backup
    
    if perform_rolling_update && perform_post_update_validation; then
        log "Update completed successfully"
        
        # Clean up old ReplicaSets
        kubectl delete rs -l app=${COMPONENT} \
            --field-selector='status.replicas==0' -n monitoring
            
    else
        log "Update failed, initiating rollback"
        perform_rollback
    fi
    
    log "Maintenance procedure completed"
}

# Execute main function
main "$@"
```

## Disaster Recovery

### Backup and Recovery Procedures

```yaml
disaster_recovery_plan:
  backup_strategy:
    frequency: "hourly"
    retention: 
      hourly: "24 hours"
      daily: "30 days"
      weekly: "12 weeks"
      monthly: "12 months"
      yearly: "7 years"
      
  backup_components:
    prometheus_data:
      method: "snapshot"
      location: "s3://nephoran-backups/prometheus/"
      encryption: "aes256"
      compression: "gzip"
      
    grafana_config:
      method: "export"
      location: "s3://nephoran-backups/grafana/"
      includes:
        - "dashboards"
        - "data_sources"
        - "organizations"
        - "users"
        
    jaeger_data:
      method: "elasticsearch_snapshot"
      location: "s3://nephoran-backups/jaeger/"
      indices_pattern: "jaeger-*"
      
    loki_data:
      method: "filesystem_backup"
      location: "s3://nephoran-backups/loki/"
      includes: "chunks_and_index"
      
    configuration_data:
      method: "git_repository"
      location: "git://github.com/nephoran/monitoring-config.git"
      branch: "backup"

  recovery_procedures:
    rto: "4 hours"  # Recovery Time Objective
    rpo: "1 hour"   # Recovery Point Objective
    
    disaster_scenarios:
      - name: "complete_cluster_loss"
        recovery_time: "4 hours"
        procedure: "full_cluster_restore"
        
      - name: "prometheus_data_corruption"
        recovery_time: "1 hour"
        procedure: "prometheus_restore_from_snapshot"
        
      - name: "monitoring_namespace_deletion"
        recovery_time: "30 minutes"
        procedure: "namespace_and_config_restore"
```

### Disaster Recovery Scripts

```bash
#!/bin/bash
# Disaster recovery automation script

set -euo pipefail

DISASTER_TYPE="${1:-}"
RECOVERY_TIMESTAMP="${2:-latest}"
BACKUP_LOCATION="${3:-s3://nephoran-backups}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] DISASTER_RECOVERY: $*"
}

declare_disaster() {
    local disaster_type="$1"
    
    log "DECLARING DISASTER: $disaster_type"
    
    # Send emergency notifications
    curl -X POST "${EMERGENCY_WEBHOOK}" \
        -H "Content-Type: application/json" \
        -d "{
            \"alert_type\": \"disaster_declared\",
            \"disaster_type\": \"$disaster_type\",
            \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)\",
            \"recovery_initiated\": true
        }"
    
    # Update status page
    update_status_page "major_outage" "Disaster recovery in progress"
    
    # Activate incident command center
    activate_incident_command_center "$disaster_type"
}

restore_prometheus_from_backup() {
    local timestamp="$1"
    
    log "Restoring Prometheus data from backup (timestamp: $timestamp)"
    
    # Scale down Prometheus
    kubectl scale deployment prometheus --replicas=0 -n monitoring
    
    # Download backup
    aws s3 cp "${BACKUP_LOCATION}/prometheus/${timestamp}/" \
        /tmp/prometheus-restore/ --recursive
    
    # Extract data
    tar -xzf /tmp/prometheus-restore/data.tar.gz -C /tmp/prometheus-restore/
    
    # Restore to persistent volume
    kubectl exec -it prometheus-0 -- rm -rf /prometheus/*
    kubectl cp /tmp/prometheus-restore/data prometheus-0:/prometheus/ -n monitoring
    
    # Scale up Prometheus
    kubectl scale deployment prometheus --replicas=3 -n monitoring
    
    # Wait for ready
    kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=300s
    
    log "Prometheus restore completed"
}

restore_grafana_from_backup() {
    local timestamp="$1"
    
    log "Restoring Grafana configuration from backup (timestamp: $timestamp)"
    
    # Download backup
    aws s3 cp "${BACKUP_LOCATION}/grafana/${timestamp}/dashboards.json" \
        /tmp/grafana-restore/
    aws s3 cp "${BACKUP_LOCATION}/grafana/${timestamp}/datasources.json" \
        /tmp/grafana-restore/
    
    # Restore via API
    local grafana_url="http://admin:admin@grafana.monitoring.svc.cluster.local:3000"
    
    # Restore data sources
    while IFS= read -r datasource; do
        curl -X POST "${grafana_url}/api/datasources" \
            -H "Content-Type: application/json" \
            -d "$datasource"
    done < <(jq -c '.[]' /tmp/grafana-restore/datasources.json)
    
    # Restore dashboards
    while IFS= read -r dashboard; do
        curl -X POST "${grafana_url}/api/dashboards/db" \
            -H "Content-Type: application/json" \
            -d "$dashboard"
    done < <(jq -c '.[]' /tmp/grafana-restore/dashboards.json)
    
    log "Grafana restore completed"
}

rebuild_monitoring_cluster() {
    log "Rebuilding monitoring cluster from scratch"
    
    # Create namespace
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply all configurations
    kubectl apply -k monitoring-config/base/ -n monitoring
    
    # Wait for core components
    kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=600s
    kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=300s
    
    # Restore data
    restore_prometheus_from_backup "$RECOVERY_TIMESTAMP"
    restore_grafana_from_backup "$RECOVERY_TIMESTAMP"
    
    # Verify system health
    perform_post_recovery_validation
    
    log "Monitoring cluster rebuild completed"
}

perform_post_recovery_validation() {
    log "Performing post-recovery validation"
    
    local validation_errors=0
    
    # Check Prometheus
    if ! curl -f http://prometheus.monitoring.svc.cluster.local:9090/-/healthy >/dev/null 2>&1; then
        log "ERROR: Prometheus health check failed"
        ((validation_errors++))
    fi
    
    # Check Grafana
    if ! curl -f http://grafana.monitoring.svc.cluster.local:3000/api/health >/dev/null 2>&1; then
        log "ERROR: Grafana health check failed"
        ((validation_errors++))
    fi
    
    # Check metrics ingestion
    local samples=$(curl -s http://prometheus:9090/api/v1/query?query=prometheus_tsdb_samples_appended_total | \
        jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
        
    if [[ "$samples" -lt 1 ]]; then
        log "ERROR: Metrics ingestion not working"
        ((validation_errors++))
    fi
    
    # Check SLA metrics
    if ! /usr/local/bin/check_sla_metrics.sh; then
        log "WARNING: SLA metrics validation failed"
        ((validation_errors++))
    fi
    
    if [[ $validation_errors -eq 0 ]]; then
        log "Post-recovery validation: PASSED"
        return 0
    else
        log "Post-recovery validation: FAILED ($validation_errors errors)"
        return 1
    fi
}

complete_recovery() {
    log "Completing disaster recovery"
    
    # Update status page
    update_status_page "operational" "Systems restored and operational"
    
    # Send recovery notification
    curl -X POST "${EMERGENCY_WEBHOOK}" \
        -H "Content-Type: application/json" \
        -d "{
            \"alert_type\": \"disaster_recovery_complete\",
            \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)\",
            \"recovery_duration\": \"$((SECONDS / 60)) minutes\"
        }"
    
    # Schedule post-mortem
    schedule_post_mortem "$DISASTER_TYPE"
    
    log "DISASTER RECOVERY COMPLETED"
}

main() {
    case "$DISASTER_TYPE" in
        "complete_cluster_loss")
            declare_disaster "$DISASTER_TYPE"
            rebuild_monitoring_cluster
            ;;
        "prometheus_corruption")
            declare_disaster "$DISASTER_TYPE"
            restore_prometheus_from_backup "$RECOVERY_TIMESTAMP"
            ;;
        "namespace_deletion")
            declare_disaster "$DISASTER_TYPE"
            rebuild_monitoring_cluster
            ;;
        *)
            echo "Unknown disaster type: $DISASTER_TYPE"
            echo "Supported types: complete_cluster_loss, prometheus_corruption, namespace_deletion"
            exit 1
            ;;
    esac
    
    if perform_post_recovery_validation; then
        complete_recovery
    else
        log "RECOVERY VALIDATION FAILED - MANUAL INTERVENTION REQUIRED"
        exit 1
    fi
}

# Execute main function
main "$@"
```

## Alert Management

### Alert Configuration

```yaml
alert_rules:
  sla_violations:
    - alert: "AvailabilitySLAViolation"
      expr: "sla:availability:5m < 99.95"
      for: "5m"
      labels:
        severity: "critical"
        component: "availability"
      annotations:
        summary: "Availability SLA violation"
        description: "Availability {{ $value }}% is below target 99.95%"
        runbook_url: "https://runbook.nephoran.io/sla/availability"
        
    - alert: "LatencySLAViolation"
      expr: "sla:latency:p95:5m > 2000"
      for: "3m"
      labels:
        severity: "critical"
        component: "latency"
      annotations:
        summary: "P95 latency SLA violation"
        description: "P95 latency {{ $value }}ms exceeds target 2000ms"
        runbook_url: "https://runbook.nephoran.io/sla/latency"
        
    - alert: "ThroughputSLAViolation"
      expr: "sla:throughput:1m < 45"
      for: "2m"
      labels:
        severity: "critical"
        component: "throughput"
      annotations:
        summary: "Throughput SLA violation"
        description: "Throughput {{ $value }} intents/min below target 45"
        runbook_url: "https://runbook.nephoran.io/sla/throughput"

  component_health:
    - alert: "PrometheusDown"
      expr: "up{job='prometheus'} == 0"
      for: "1m"
      labels:
        severity: "critical"
        component: "prometheus"
      annotations:
        summary: "Prometheus instance down"
        description: "Prometheus instance {{ $labels.instance }} is down"
        
    - alert: "GrafanaDown"
      expr: "up{job='grafana'} == 0"
      for: "2m"
      labels:
        severity: "warning"
        component: "grafana"
      annotations:
        summary: "Grafana instance down"
        description: "Grafana instance {{ $labels.instance }} is down"

  resource_usage:
    - alert: "HighMemoryUsage"
      expr: "container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9"
      for: "5m"
      labels:
        severity: "warning"
        component: "resource"
      annotations:
        summary: "High memory usage"
        description: "Container {{ $labels.container }} memory usage > 90%"
        
    - alert: "DiskSpaceWarning"
      expr: "node_filesystem_free_bytes / node_filesystem_size_bytes < 0.2"
      for: "10m"
      labels:
        severity: "warning"
        component: "storage"
      annotations:
        summary: "Low disk space"
        description: "Disk space on {{ $labels.instance }} < 20%"
```

### Notification Routing

```yaml
alertmanager_config:
  global:
    smtp_smarthost: 'localhost:587'
    smtp_from: 'alerts@nephoran.io'
    
  route:
    group_by: ['alertname', 'component']
    group_wait: 30s
    group_interval: 5m
    repeat_interval: 4h
    receiver: 'default'
    
    routes:
      # Critical SLA violations
      - match:
          severity: critical
          component: availability|latency|throughput
        receiver: 'sla-critical'
        group_wait: 10s
        group_interval: 1m
        repeat_interval: 15m
        
      # Component failures
      - match:
          severity: critical
          component: prometheus|grafana|jaeger
        receiver: 'component-critical'
        
      # Resource warnings
      - match:
          severity: warning
          component: resource|storage
        receiver: 'resource-warnings'
        repeat_interval: 2h

  receivers:
    - name: 'default'
      slack_configs:
        - api_url: '{{ .SlackWebhookURL }}'
          channel: '#monitoring-alerts'
          
    - name: 'sla-critical'
      pagerduty_configs:
        - routing_key: '{{ .PagerDutyIntegrationKey }}'
          description: 'SLA Violation: {{ .GroupLabels.alertname }}'
          severity: 'critical'
      slack_configs:
        - api_url: '{{ .SlackWebhookURL }}'
          channel: '#incidents-critical'
          title: 'CRITICAL SLA VIOLATION'
          color: 'danger'
      email_configs:
        - to: 'sre-team@nephoran.io'
          subject: 'CRITICAL: SLA Violation - {{ .GroupLabels.alertname }}'
          
    - name: 'component-critical'
      slack_configs:
        - api_url: '{{ .SlackWebhookURL }}'
          channel: '#operations'
          color: 'danger'
      email_configs:
        - to: 'operations-team@nephoran.io'
          
    - name: 'resource-warnings'
      slack_configs:
        - api_url: '{{ .SlackWebhookURL }}'
          channel: '#resource-alerts'
          color: 'warning'

  inhibit_rules:
    # Inhibit duplicate alerts
    - source_match:
        severity: 'critical'
      target_match:
        severity: 'warning'
      equal: ['alertname', 'component']
```

## Escalation Procedures

### Escalation Matrix

```yaml
escalation_matrix:
  sla_violations:
    level_1:
      timeframe: "immediate"
      contacts:
        - role: "oncall_sre"
          method: "pagerduty"
        - role: "monitoring_team_lead"
          method: "slack"
      actions:
        - investigate_root_cause
        - attempt_immediate_mitigation
        - update_incident_channel
        
    level_2:
      timeframe: "15 minutes"
      contacts:
        - role: "engineering_manager"
          method: "phone_call"
        - role: "principal_architect"
          method: "slack"
      actions:
        - activate_incident_command_center
        - consider_architectural_solutions
        - prepare_customer_communication
        
    level_3:
      timeframe: "30 minutes"
      contacts:
        - role: "cto"
          method: "phone_call"
        - role: "ceo"
          method: "email"
      actions:
        - executive_decision_making
        - public_communication_strategy
        - resource_allocation_decisions
        
  component_failures:
    level_1:
      timeframe: "immediate"
      contacts:
        - role: "platform_team"
          method: "slack"
      actions:
        - restart_failed_components
        - check_dependencies
        - monitor_recovery
        
    level_2:
      timeframe: "30 minutes"
      contacts:
        - role: "infrastructure_team"
          method: "pagerduty"
      actions:
        - deep_system_investigation
        - consider_scaling_solutions
        - prepare_rollback_plan
```

### Contact Information

```yaml
contact_directory:
  teams:
    sre_team:
      primary_oncall: "+1-555-SRE-TEAM"
      backup_oncall: "+1-555-SRE-BACK"
      slack_channel: "#sre-team"
      email: "sre-team@nephoran.io"
      
    platform_team:
      lead: "+1-555-PLATFORM"
      slack_channel: "#platform"
      email: "platform-team@nephoran.io"
      
    executive_team:
      cto: "+1-555-CTO-EXEC"
      ceo: "+1-555-CEO-EXEC"
      vp_engineering: "+1-555-VP-ENG"
      
  external_contacts:
    cloud_provider_support:
      aws: "+1-800-AWS-SUPPORT"
      azure: "+1-800-AZURE"
      gcp: "+1-800-GCP-HELP"
      
    vendor_support:
      prometheus: "support@prometheus.io"
      grafana: "support@grafana.com"
      jaegertracing: "support@jaegertracing.io"
```

## Knowledge Base

### Common Scenarios and Solutions

#### Scenario: Sudden Spike in Alert Volume

**Symptoms:**
- AlertManager receiving 100+ alerts per minute
- Notification channels overwhelmed
- Difficulty identifying root cause

**Investigation:**
```bash
# Check alert volume by type
curl -s http://alertmanager:9093/api/v1/alerts | \
    jq '.data | group_by(.labels.alertname) | map({name: .[0].labels.alertname, count: length}) | sort_by(.count) | reverse'

# Identify alert storm patterns  
curl -s http://prometheus:9090/api/v1/query?query=increase\(alertmanager_alerts_received_total[5m]\)

# Check for infrastructure issues
kubectl get events --sort-by='.firstTimestamp' -n monitoring
```

**Solution:**
1. Implement alert grouping and throttling
2. Identify and silence noisy alerts
3. Fix underlying infrastructure issues
4. Review alert threshold configuration

#### Scenario: Monitoring Data Gaps

**Symptoms:**
- Missing data points in dashboards
- Gaps in time series data
- SLA calculation inconsistencies

**Investigation:**
```bash
# Check Prometheus targets
curl -s http://prometheus:9090/api/v1/targets | jq '.data.activeTargets[] | select(.health!="up")'

# Check scrape intervals
curl -s http://prometheus:9090/api/v1/query?query=up\{job=\"nephoran-operator\"\}

# Verify network connectivity
kubectl exec -it prometheus-0 -- wget -qO- http://nephoran-controller:8080/metrics
```

**Solution:**
1. Fix service discovery configuration
2. Resolve network connectivity issues
3. Adjust scrape intervals and timeouts
4. Implement data backfill procedures

### Troubleshooting Decision Tree

```
Alert Received
    │
    ├─ SLA Violation?
    │   ├─ Yes → Follow SLA Violation Procedure
    │   └─ No → Continue to Component Check
    │
    ├─ Component Down?
    │   ├─ Yes → Restart Component → Verify Recovery
    │   └─ No → Continue to Resource Check
    │
    ├─ Resource Exhaustion?
    │   ├─ Yes → Scale Resources → Monitor Impact  
    │   └─ No → Continue to Configuration Check
    │
    ├─ Configuration Issue?
    │   ├─ Yes → Review Recent Changes → Rollback if Needed
    │   └─ No → Escalate to Level 2 Support
    │
    └─ Unknown Issue → Document → Create Investigation Ticket
```

## Conclusion

This comprehensive operations runbook provides the foundation for maintaining optimal monitoring system performance and ensuring rapid, effective response to incidents. Regular review and updates of these procedures ensure continued operational excellence and SLA compliance for the Nephoran Intent Operator monitoring infrastructure.
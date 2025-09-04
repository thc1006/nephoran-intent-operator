#!/usr/bin/env python3
"""
Security Automation Suite for Nephoran Intent Operator

This comprehensive security automation framework provides:
- Automated vulnerability assessment
- Compliance monitoring and validation
- Incident response automation
- Security metrics collection and analysis
- Remediation workflow automation

Author: Security Team
Version: 2.0.0
License: Apache 2.0
"""

import asyncio
import json
import logging
import os
import subprocess
import sys
import time
import uuid
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any
from dataclasses import dataclass, asdict
from enum import Enum
import yaml
import requests
import kubernetes
from kubernetes import client, config
from kubernetes.client.rest import ApiException

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('security-automation.log'),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

class SecuritySeverity(Enum):
    """Security severity levels"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"

class IncidentStatus(Enum):
    """Incident response status"""
    DETECTED = "detected"
    INVESTIGATING = "investigating"
    CONTAINED = "contained"
    ERADICATED = "eradicated"
    RECOVERED = "recovered"
    CLOSED = "closed"

@dataclass
class SecurityFinding:
    """Security finding data structure"""
    id: str
    severity: SecuritySeverity
    category: str
    title: str
    description: str
    affected_resource: str
    remediation: str
    timestamp: datetime
    tool: str
    cve_id: Optional[str] = None
    cvss_score: Optional[float] = None
    references: Optional[List[str]] = None
    
class SecurityAutomationSuite:
    """Main security automation orchestrator"""
    
    def __init__(self, config_path: str = "security/configs/automation-config.yaml"):
        """Initialize the security automation suite"""
        self.config = self._load_config(config_path)
        self.findings: List[SecurityFinding] = []
        self.incidents: Dict[str, Dict] = {}
        self.metrics: Dict[str, Any] = {}
        
        # Initialize Kubernetes client
        try:
            config.load_incluster_config()
        except:
            try:
                config.load_kube_config()
            except:
                logger.warning("Could not load Kubernetes configuration")
        
        self.k8s_client = client.ApiClient()
        self.v1 = client.CoreV1Api()
        self.apps_v1 = client.AppsV1Api()
        self.networking_v1 = client.NetworkingV1Api()
        
    def _load_config(self, config_path: str) -> Dict:
        """Load automation configuration"""
        try:
            with open(config_path, 'r') as f:
                config = yaml.safe_load(f)
            logger.info(f"Loaded configuration from {config_path}")
            return config
        except FileNotFoundError:
            logger.warning(f"Configuration file {config_path} not found, using defaults")
            return self._get_default_config()
        except Exception as e:
            logger.error(f"Error loading configuration: {e}")
            return self._get_default_config()
    
    def _get_default_config(self) -> Dict:
        """Get default configuration"""
        return {
            "scanning": {
                "enabled_tools": ["gosec", "trivy", "gitleaks"],
                "scan_frequency": "daily",
                "severity_threshold": "medium"
            },
            "compliance": {
                "frameworks": ["nist", "cis", "owasp"],
                "check_frequency": "weekly"
            },
            "incident_response": {
                "auto_containment": True,
                "notification_channels": ["slack", "email"],
                "escalation_timeout": 3600
            },
            "remediation": {
                "auto_remediation": False,
                "require_approval": True,
                "backup_before_change": True
            }
        }

    async def run_comprehensive_security_scan(self) -> Dict[str, Any]:
        """Run comprehensive security scanning"""
        logger.info("Starting comprehensive security scan")
        
        scan_results = {
            "scan_id": str(uuid.uuid4()),
            "timestamp": datetime.now().isoformat(),
            "tools_executed": [],
            "findings_summary": {},
            "compliance_status": {},
            "remediation_recommendations": []
        }
        
        # Run SAST (Static Application Security Testing)
        if "gosec" in self.config["scanning"]["enabled_tools"]:
            sast_results = await self._run_sast_scan()
            scan_results["tools_executed"].append("gosec")
            scan_results["findings_summary"]["sast"] = sast_results
        
        # Run container vulnerability scanning
        if "trivy" in self.config["scanning"]["enabled_tools"]:
            container_results = await self._run_container_scan()
            scan_results["tools_executed"].append("trivy")
            scan_results["findings_summary"]["container_security"] = container_results
        
        # Run secret detection
        if "gitleaks" in self.config["scanning"]["enabled_tools"]:
            secret_results = await self._run_secret_scan()
            scan_results["tools_executed"].append("gitleaks")
            scan_results["findings_summary"]["secrets"] = secret_results
        
        # Run infrastructure security scan
        infra_results = await self._run_infrastructure_scan()
        scan_results["tools_executed"].append("kube-bench")
        scan_results["findings_summary"]["infrastructure"] = infra_results
        
        # Run compliance checks
        compliance_results = await self._run_compliance_checks()
        scan_results["compliance_status"] = compliance_results
        
        # Generate remediation recommendations
        recommendations = self._generate_remediation_recommendations()
        scan_results["remediation_recommendations"] = recommendations
        
        # Save results
        await self._save_scan_results(scan_results)
        
        # Generate alerts for critical findings
        await self._process_critical_findings(scan_results)
        
        logger.info(f"Comprehensive security scan completed: {scan_results['scan_id']}")
        return scan_results

    async def _run_sast_scan(self) -> Dict[str, Any]:
        """Run Static Application Security Testing"""
        logger.info("Running SAST scan with GoSec")
        
        try:
            # Run gosec with SARIF output for better parsing
            cmd = [
                "gosec",
                "-fmt", "sarif",
                "-out", "/tmp/gosec-results.sarif",
                "-severity", self.config["scanning"]["severity_threshold"],
                "./..."
            ]
            
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd="."
            )
            
            stdout, stderr = await process.communicate()
            
            # Parse SARIF results
            sarif_results = self._parse_sarif_results("/tmp/gosec-results.sarif")
            
            # Convert to standardized findings
            findings = []
            for result in sarif_results.get("runs", [{}])[0].get("results", []):
                finding = SecurityFinding(
                    id=str(uuid.uuid4()),
                    severity=self._map_sarif_severity(result.get("level", "warning")),
                    category="static_analysis",
                    title=result.get("message", {}).get("text", "Unknown issue"),
                    description=result.get("message", {}).get("text", ""),
                    affected_resource=result.get("locations", [{}])[0].get("physicalLocation", {}).get("artifactLocation", {}).get("uri", "unknown"),
                    remediation=self._get_gosec_remediation(result.get("ruleId", "")),
                    timestamp=datetime.now(),
                    tool="gosec",
                    cve_id=None,
                    references=result.get("help", {}).get("uri", [])}
                )
                findings.append(finding)
            
            self.findings.extend(findings)
            
            return {
                "tool": "gosec",
                "status": "completed",
                "findings_count": len(findings),
                "severity_breakdown": self._get_severity_breakdown(findings),
                "execution_time": time.time()
            }
            
        except Exception as e:
            logger.error(f"SAST scan failed: {e}")
            return {
                "tool": "gosec",
                "status": "failed",
                "error": str(e),
                "findings_count": 0
            }

    async def _run_container_scan(self) -> Dict[str, Any]:
        """Run container vulnerability scanning"""
        logger.info("Running container vulnerability scan with Trivy")
        
        try:
            # Get all container images in use
            images = await self._get_running_container_images()
            
            scan_results = {
                "tool": "trivy",
                "status": "completed",
                "images_scanned": len(images),
                "total_vulnerabilities": 0,
                "severity_breakdown": {
                    "critical": 0,
                    "high": 0,
                    "medium": 0,
                    "low": 0
                },
                "image_results": []
            }
            
            for image in images:
                logger.info(f"Scanning container image: {image}")
                
                cmd = [
                    "trivy",
                    "image",
                    "--format", "json",
                    "--severity", "CRITICAL,HIGH,MEDIUM,LOW",
                    image
                ]
                
                process = await asyncio.create_subprocess_exec(
                    *cmd,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE
                )
                
                stdout, stderr = await process.communicate()
                
                if process.returncode == 0:
                    try:
                        trivy_results = json.loads(stdout.decode())
                        image_findings = self._process_trivy_results(trivy_results, image)
                        
                        self.findings.extend(image_findings)
                        
                        image_result = {
                            "image": image,
                            "vulnerabilities": len(image_findings),
                            "severity_breakdown": self._get_severity_breakdown(image_findings)
                        }
                        
                        scan_results["image_results"].append(image_result)
                        scan_results["total_vulnerabilities"] += len(image_findings)
                        
                        # Update overall severity breakdown
                        for severity in scan_results["severity_breakdown"]:
                            scan_results["severity_breakdown"][severity] += image_result["severity_breakdown"].get(severity, 0)
                        
                    except json.JSONDecodeError as e:
                        logger.error(f"Failed to parse Trivy results for {image}: {e}")
                else:
                    logger.error(f"Trivy scan failed for {image}: {stderr.decode()}")
            
            return scan_results
            
        except Exception as e:
            logger.error(f"Container scan failed: {e}")
            return {
                "tool": "trivy",
                "status": "failed",
                "error": str(e)
            }

    async def _run_secret_scan(self) -> Dict[str, Any]:
        """Run secrets detection scan"""
        logger.info("Running secrets detection with Gitleaks")
        
        try:
            cmd = [
                "gitleaks",
                "detect",
                "--source", ".",
                "--report-format", "json",
                "--report-path", "/tmp/gitleaks-results.json",
                "--config", "security/configs/gitleaks.toml"
            ]
            
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            
            stdout, stderr = await process.communicate()
            
            # Gitleaks returns non-zero exit code when secrets are found
            if os.path.exists("/tmp/gitleaks-results.json"):
                with open("/tmp/gitleaks-results.json", 'r') as f:
                    gitleaks_results = json.load(f)
                
                findings = []
                for result in gitleaks_results:
                    finding = SecurityFinding(
                        id=str(uuid.uuid4()),
                        severity=SecuritySeverity.CRITICAL,  # All secrets are critical
                        category="secrets",
                        title=f"Secret detected: {result.get('RuleID', 'Unknown')}",
                        description=result.get("Description", "Potential secret detected"),
                        affected_resource=f"{result.get('File', 'unknown')}:{result.get('StartLine', 0)}",
                        remediation="Remove or encrypt the secret, rotate if already committed",
                        timestamp=datetime.now(),
                        tool="gitleaks"
                    )
                    findings.append(finding)
                
                self.findings.extend(findings)
                
                # Immediately trigger incident for any secrets found
                if findings:
                    await self._trigger_security_incident(
                        "secrets_detected",
                        f"Gitleaks detected {len(findings)} potential secrets in the codebase",
                        SecuritySeverity.CRITICAL
                    )
                
                return {
                    "tool": "gitleaks",
                    "status": "completed",
                    "secrets_found": len(findings),
                    "files_affected": len(set(f.affected_resource.split(':')[0] for f in findings))
                }
            else:
                return {
                    "tool": "gitleaks",
                    "status": "completed",
                    "secrets_found": 0
                }
                
        except Exception as e:
            logger.error(f"Secrets scan failed: {e}")
            return {
                "tool": "gitleaks",
                "status": "failed",
                "error": str(e)
            }

    async def _run_infrastructure_scan(self) -> Dict[str, Any]:
        """Run infrastructure security scan"""
        logger.info("Running infrastructure security scan")
        
        try:
            # Run kube-bench for CIS Kubernetes Benchmark
            kube_bench_results = await self._run_kube_bench()
            
            # Check Pod Security Standards compliance
            psp_results = await self._check_pod_security_standards()
            
            # Network policy analysis
            network_results = await self._analyze_network_policies()
            
            # RBAC analysis
            rbac_results = await self._analyze_rbac_configuration()
            
            return {
                "tool": "infrastructure_scan",
                "status": "completed",
                "components": {
                    "kube_bench": kube_bench_results,
                    "pod_security_standards": psp_results,
                    "network_policies": network_results,
                    "rbac": rbac_results
                }
            }
            
        except Exception as e:
            logger.error(f"Infrastructure scan failed: {e}")
            return {
                "tool": "infrastructure_scan",
                "status": "failed",
                "error": str(e)
            }

    async def _run_compliance_checks(self) -> Dict[str, Any]:
        """Run compliance framework checks"""
        logger.info("Running compliance checks")
        
        compliance_status = {}
        
        # NIST CSF 2.0 compliance
        if "nist" in self.config["compliance"]["frameworks"]:
            compliance_status["nist_csf"] = await self._check_nist_compliance()
        
        # CIS Kubernetes Benchmark
        if "cis" in self.config["compliance"]["frameworks"]:
            compliance_status["cis_kubernetes"] = await self._check_cis_compliance()
        
        # OWASP compliance
        if "owasp" in self.config["compliance"]["frameworks"]:
            compliance_status["owasp"] = await self._check_owasp_compliance()
        
        return compliance_status

    async def _trigger_security_incident(self, incident_type: str, description: str, severity: SecuritySeverity):
        """Trigger security incident response"""
        incident_id = str(uuid.uuid4())
        
        incident = {
            "id": incident_id,
            "type": incident_type,
            "description": description,
            "severity": severity.value,
            "status": IncidentStatus.DETECTED.value,
            "created_at": datetime.now().isoformat(),
            "affected_systems": [],
            "timeline": [],
            "response_actions": []
        }
        
        self.incidents[incident_id] = incident
        
        logger.critical(f"Security incident triggered: {incident_id} - {description}")
        
        # Execute automated response actions
        if self.config["incident_response"]["auto_containment"]:
            await self._execute_automated_response(incident)
        
        # Send notifications
        await self._send_incident_notifications(incident)
        
        return incident_id

    async def _execute_automated_response(self, incident: Dict):
        """Execute automated incident response actions"""
        logger.info(f"Executing automated response for incident {incident['id']}")
        
        incident_type = incident["type"]
        severity = SecuritySeverity(incident["severity"])
        
        response_actions = []
        
        try:
            if incident_type == "secrets_detected":
                # Immediately revoke any found credentials
                actions = await self._revoke_exposed_credentials(incident)
                response_actions.extend(actions)
                
            elif incident_type == "malware_detected":
                # Quarantine affected pods
                actions = await self._quarantine_infected_pods(incident)
                response_actions.extend(actions)
                
            elif incident_type == "unauthorized_access":
                # Lock down accounts and audit access
                actions = await self._lockdown_compromised_access(incident)
                response_actions.extend(actions)
                
            elif incident_type == "vulnerability_exploitation":
                # Apply emergency patches or mitigations
                actions = await self._apply_emergency_mitigations(incident)
                response_actions.extend(actions)
            
            # Update incident with response actions
            incident["response_actions"] = response_actions
            incident["timeline"].append({
                "timestamp": datetime.now().isoformat(),
                "action": "automated_response_executed",
                "details": f"Executed {len(response_actions)} automated response actions"
            })
            
            # Update incident status
            if response_actions:
                incident["status"] = IncidentStatus.CONTAINED.value
                
        except Exception as e:
            logger.error(f"Automated response failed for incident {incident['id']}: {e}")
            incident["timeline"].append({
                "timestamp": datetime.now().isoformat(),
                "action": "automated_response_failed",
                "error": str(e)
            })

    async def _quarantine_infected_pods(self, incident: Dict) -> List[str]:
        """Quarantine pods suspected of malware infection"""
        actions = []
        
        try:
            # Get list of suspicious pods from incident context
            suspicious_pods = incident.get("affected_systems", [])
            
            for pod_info in suspicious_pods:
                if pod_info.get("type") == "pod":
                    namespace = pod_info.get("namespace", "default")
                    pod_name = pod_info.get("name")
                    
                    if pod_name:
                        # Label pod for quarantine
                        body = {
                            "metadata": {
                                "labels": {
                                    "quarantine": "true",
                                    "security.nephoran.io/incident-id": incident["id"]
                                },
                                "annotations": {
                                    "security.nephoran.io/quarantine-reason": "malware_detected",
                                    "security.nephoran.io/quarantine-timestamp": datetime.now().isoformat()
                                }
                            }
                        }
                        
                        await self._patch_pod(namespace, pod_name, body)
                        
                        # Apply network isolation
                        await self._apply_quarantine_network_policy(namespace, pod_name)
                        
                        actions.append(f"Quarantined pod {namespace}/{pod_name}")
                        logger.info(f"Quarantined pod {namespace}/{pod_name}")
            
        except Exception as e:
            logger.error(f"Failed to quarantine pods: {e}")
            actions.append(f"Failed to quarantine pods: {e}")
        
        return actions

    async def _apply_quarantine_network_policy(self, namespace: str, pod_name: str):
        """Apply network policy to isolate quarantined pod"""
        policy_name = f"quarantine-{pod_name}-{int(time.time())}"
        
        network_policy = client.V1NetworkPolicy(
            api_version="networking.k8s.io/v1",
            kind="NetworkPolicy",
            metadata=client.V1ObjectMeta(
                name=policy_name,
                namespace=namespace,
                labels={
                    "app.kubernetes.io/name": "quarantine-policy",
                    "security.nephoran.io/purpose": "quarantine"
                }
            ),
            spec=client.V1NetworkPolicySpec(
                pod_selector=client.V1LabelSelector(
                    match_labels={
                        "quarantine": "true"
                    }
                ),
                policy_types=["Ingress", "Egress"],
                ingress=[],  # Deny all ingress
                egress=[
                    # Allow DNS only
                    client.V1NetworkPolicyEgressRule(
                        to=[],
                        ports=[
                            client.V1NetworkPolicyPort(
                                protocol="UDP",
                                port=53
                            )
                        ]
                    )
                ]
            )
        )
        
        try:
            self.networking_v1.create_namespaced_network_policy(
                namespace=namespace,
                body=network_policy
            )
            logger.info(f"Applied quarantine network policy {policy_name} in namespace {namespace}")
        except ApiException as e:
            logger.error(f"Failed to create quarantine network policy: {e}")

    async def generate_security_metrics_dashboard(self) -> Dict[str, Any]:
        """Generate comprehensive security metrics dashboard"""
        logger.info("Generating security metrics dashboard")
        
        dashboard_data = {
            "generated_at": datetime.now().isoformat(),
            "summary": {
                "total_findings": len(self.findings),
                "active_incidents": len([i for i in self.incidents.values() if i["status"] not in ["closed", "resolved"]]),
                "critical_findings": len([f for f in self.findings if f.severity == SecuritySeverity.CRITICAL]),
                "high_findings": len([f for f in self.findings if f.severity == SecuritySeverity.HIGH])
            },
            "findings_by_category": self._get_findings_by_category(),
            "findings_by_tool": self._get_findings_by_tool(),
            "severity_distribution": self._get_severity_breakdown(self.findings),
            "compliance_scores": await self._calculate_compliance_scores(),
            "security_trends": await self._calculate_security_trends(),
            "remediation_metrics": self._calculate_remediation_metrics(),
            "incident_statistics": self._calculate_incident_statistics()
        }
        
        # Save dashboard data
        dashboard_path = "security/reports/security-metrics-dashboard.json"
        os.makedirs(os.path.dirname(dashboard_path), exist_ok=True)
        
        with open(dashboard_path, 'w') as f:
            json.dump(dashboard_data, f, indent=2, default=str)
        
        logger.info(f"Security metrics dashboard saved to {dashboard_path}")
        return dashboard_data

    async def _send_incident_notifications(self, incident: Dict):
        """Send incident notifications to configured channels"""
        try:
            # Slack notification
            if "slack" in self.config["incident_response"]["notification_channels"]:
                await self._send_slack_notification(incident)
            
            # Email notification
            if "email" in self.config["incident_response"]["notification_channels"]:
                await self._send_email_notification(incident)
            
            # PagerDuty integration (if configured)
            if "pagerduty" in self.config["incident_response"]["notification_channels"]:
                await self._send_pagerduty_alert(incident)
                
        except Exception as e:
            logger.error(f"Failed to send notifications for incident {incident['id']}: {e}")

    def _get_severity_breakdown(self, findings: List[SecurityFinding]) -> Dict[str, int]:
        """Get severity breakdown of findings"""
        breakdown = {
            "critical": 0,
            "high": 0,
            "medium": 0,
            "low": 0,
            "info": 0
        }
        
        for finding in findings:
            if finding.severity.value in breakdown:
                breakdown[finding.severity.value] += 1
        
        return breakdown

    def _get_findings_by_category(self) -> Dict[str, int]:
        """Get findings grouped by category"""
        categories = {}
        for finding in self.findings:
            categories[finding.category] = categories.get(finding.category, 0) + 1
        return categories

    def _get_findings_by_tool(self) -> Dict[str, int]:
        """Get findings grouped by scanning tool"""
        tools = {}
        for finding in self.findings:
            tools[finding.tool] = tools.get(finding.tool, 0) + 1
        return tools

    async def _calculate_compliance_scores(self) -> Dict[str, float]:
        """Calculate compliance scores for different frameworks"""
        # This would integrate with actual compliance checking tools
        # For now, returning mock scores
        return {
            "nist_csf": 85.5,
            "cis_kubernetes": 78.2,
            "owasp_top_10": 92.1,
            "pci_dss": 88.7,
            "iso_27001": 91.3
        }

    # Additional utility methods would go here...
    
    async def run_automated_remediation(self, findings: List[SecurityFinding]) -> Dict[str, Any]:
        """Run automated remediation for applicable findings"""
        logger.info(f"Starting automated remediation for {len(findings)} findings")
        
        remediation_results = {
            "total_findings": len(findings),
            "attempted_fixes": 0,
            "successful_fixes": 0,
            "failed_fixes": 0,
            "skipped_fixes": 0,
            "results": []
        }
        
        for finding in findings:
            try:
                # Only attempt automated remediation if configured and safe
                if (self.config["remediation"]["auto_remediation"] and 
                    finding.severity in [SecuritySeverity.LOW, SecuritySeverity.MEDIUM]):
                    
                    remediation_results["attempted_fixes"] += 1
                    
                    # Backup before making changes if configured
                    if self.config["remediation"]["backup_before_change"]:
                        await self._create_remediation_backup(finding)
                    
                    # Apply remediation based on finding type
                    success = await self._apply_automated_remediation(finding)
                    
                    if success:
                        remediation_results["successful_fixes"] += 1
                        result = {
                            "finding_id": finding.id,
                            "status": "success",
                            "action": "automated_fix_applied"
                        }
                    else:
                        remediation_results["failed_fixes"] += 1
                        result = {
                            "finding_id": finding.id,
                            "status": "failed",
                            "action": "automated_fix_failed"
                        }
                else:
                    remediation_results["skipped_fixes"] += 1
                    result = {
                        "finding_id": finding.id,
                        "status": "skipped",
                        "reason": "requires_manual_intervention"
                    }
                
                remediation_results["results"].append(result)
                
            except Exception as e:
                logger.error(f"Remediation failed for finding {finding.id}: {e}")
                remediation_results["failed_fixes"] += 1
                remediation_results["results"].append({
                    "finding_id": finding.id,
                    "status": "error",
                    "error": str(e)
                })
        
        logger.info(f"Automated remediation completed. Success: {remediation_results['successful_fixes']}, Failed: {remediation_results['failed_fixes']}")
        return remediation_results

async def main():
    """Main entry point for the security automation suite"""
    automation_suite = SecurityAutomationSuite()
    
    # Run comprehensive security scan
    scan_results = await automation_suite.run_comprehensive_security_scan()
    
    # Generate security metrics dashboard
    dashboard = await automation_suite.generate_security_metrics_dashboard()
    
    # Print summary
    print("\n" + "="*60)
    print("SECURITY AUTOMATION SUITE - EXECUTION SUMMARY")
    print("="*60)
    print(f"Scan ID: {scan_results['scan_id']}")
    print(f"Tools Executed: {', '.join(scan_results['tools_executed'])}")
    print(f"Total Findings: {dashboard['summary']['total_findings']}")
    print(f"Critical Findings: {dashboard['summary']['critical_findings']}")
    print(f"High Severity Findings: {dashboard['summary']['high_findings']}")
    print(f"Active Incidents: {dashboard['summary']['active_incidents']}")
    print("="*60)
    
    return scan_results

if __name__ == "__main__":
    asyncio.run(main())
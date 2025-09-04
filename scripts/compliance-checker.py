#!/usr/bin/env python3
"""
Telecommunications Security Compliance Checker

This script validates the Nephoran Intent Operator against various
telecommunications security frameworks and standards.
"""

import json
import os
import sys
import argparse
import yaml
from typing import Dict, List, Any
from pathlib import Path

class TelecomComplianceChecker:
    def __init__(self, project_root: str):
        self.project_root = Path(project_root)
        self.compliance_results = {}
    
    def check_nist_csf_compliance(self) -> Dict[str, Any]:
        """Check compliance with NIST Cybersecurity Framework."""
        results = {
            'framework': 'NIST Cybersecurity Framework',
            'version': '1.1',
            'categories': {},
            'overall_score': 0,
            'recommendations': []
        }
        
        # IDENTIFY - Asset Management, Risk Assessment
        identify_score = 0
        identify_checks = []
        
        # Check for asset inventory and documentation
        if (self.project_root / 'README.md').exists():
            identify_score += 20
            identify_checks.append("✓ Project documentation exists")
        else:
            identify_checks.append("✗ Missing project documentation")
            results['recommendations'].append("Create comprehensive project documentation")
        
        # Check for dependency management
        if (self.project_root / 'go.mod').exists():
            identify_score += 20
            identify_checks.append("✓ Dependency management in place")
        else:
            identify_checks.append("✗ Missing dependency management")
        
        # Check for security documentation
        security_docs = list(self.project_root.glob('**/SECURITY*.md'))
        if security_docs:
            identify_score += 20
            identify_checks.append("✓ Security documentation found")
        else:
            identify_checks.append("✗ Missing security documentation")
            results['recommendations'].append("Create security documentation and incident response procedures")
        
        # Check for risk assessment artifacts
        if list(self.project_root.glob('**/security/**')):
            identify_score += 20
            identify_checks.append("✓ Security configurations found")
        else:
            identify_checks.append("✗ No security configurations found")
        
        results['categories']['IDENTIFY'] = {
            'score': min(identify_score, 100),
            'checks': identify_checks,
            'description': 'Asset inventory, risk assessment, and governance'
        }
        
        # PROTECT - Access Control, Data Security
        protect_score = 0
        protect_checks = []
        
        # Check for RBAC configurations
        rbac_files = list(self.project_root.glob('**/rbac*.yaml')) + list(self.project_root.glob('**/rbac*.yml'))
        if rbac_files:
            protect_score += 25
            protect_checks.append(f"✓ RBAC configurations found ({len(rbac_files)} files)")
        else:
            protect_checks.append("✗ Missing RBAC configurations")
            results['recommendations'].append("Implement Role-Based Access Control (RBAC)")
        
        # Check for network policies
        netpol_files = list(self.project_root.glob('**/*network*policy*.yaml'))
        if netpol_files:
            protect_score += 25
            protect_checks.append(f"✓ Network policies found ({len(netpol_files)} files)")
        else:
            protect_checks.append("✗ Missing network policies")
            results['recommendations'].append("Implement Kubernetes Network Policies for micro-segmentation")
        
        # Check for TLS/encryption usage
        tls_files = self._find_files_with_content(['tls', 'ssl', 'encrypt', 'crypto'])
        if tls_files:
            protect_score += 25
            protect_checks.append(f"✓ Encryption/TLS implementations found ({len(tls_files)} files)")
        else:
            protect_checks.append("✗ Limited encryption implementation")
            results['recommendations'].append("Implement comprehensive TLS/mTLS for all communications")
        
        # Check for security contexts in deployments
        if self._check_security_contexts():
            protect_score += 25
            protect_checks.append("✓ Security contexts configured")
        else:
            protect_checks.append("✗ Missing security contexts")
            results['recommendations'].append("Configure restrictive security contexts for all containers")
        
        results['categories']['PROTECT'] = {
            'score': min(protect_score, 100),
            'checks': protect_checks,
            'description': 'Access control, data protection, and maintenance'
        }
        
        # DETECT - Monitoring and Anomalies
        detect_score = 0
        detect_checks = []
        
        # Check for monitoring configurations
        monitoring_files = list(self.project_root.glob('**/monitoring/**/*.yaml'))
        if monitoring_files:
            detect_score += 30
            detect_checks.append(f"✓ Monitoring configurations found ({len(monitoring_files)} files)")
        else:
            detect_checks.append("✗ Missing monitoring configurations")
            results['recommendations'].append("Implement comprehensive monitoring with Prometheus/Grafana")
        
        # Check for ServiceMonitor resources
        servicemonitor_files = list(self.project_root.glob('**/*servicemonitor*.yaml'))
        if servicemonitor_files:
            detect_score += 25
            detect_checks.append(f"✓ ServiceMonitor resources found ({len(servicemonitor_files)} files)")
        else:
            detect_checks.append("✗ Missing ServiceMonitor resources")
        
        # Check for logging configuration
        logging_files = self._find_files_with_content(['log', 'logger', 'logging'])
        if logging_files:
            detect_score += 25
            detect_checks.append(f"✓ Logging implementations found ({len(logging_files)} files)")
        else:
            detect_checks.append("✗ Limited logging implementation")
        
        # Check for alerting rules
        alert_files = list(self.project_root.glob('**/*alert*.yaml')) + list(self.project_root.glob('**/*prometheus*rule*.yaml'))
        if alert_files:
            detect_score += 20
            detect_checks.append(f"✓ Alerting rules found ({len(alert_files)} files)")
        else:
            detect_checks.append("✗ Missing alerting rules")
            results['recommendations'].append("Configure alerting rules for security events")
        
        results['categories']['DETECT'] = {
            'score': min(detect_score, 100),
            'checks': detect_checks,
            'description': 'Continuous monitoring and anomaly detection'
        }
        
        # RESPOND - Incident Response
        respond_score = 0
        respond_checks = []
        
        # Check for incident response procedures
        if (self.project_root / 'INCIDENT_RESPONSE.md').exists():
            respond_score += 40
            respond_checks.append("✓ Incident response documentation exists")
        else:
            respond_checks.append("✗ Missing incident response procedures")
            results['recommendations'].append("Create incident response procedures and runbooks")
        
        # Check for deployment rollback capabilities
        if list(self.project_root.glob('**/rollback*')):
            respond_score += 30
            respond_checks.append("✓ Rollback procedures found")
        else:
            respond_checks.append("✗ Missing rollback procedures")
        
        # Check for automation scripts
        automation_scripts = list(self.project_root.glob('scripts/*.sh'))
        if automation_scripts:
            respond_score += 30
            respond_checks.append(f"✓ Automation scripts found ({len(automation_scripts)} scripts)")
        else:
            respond_checks.append("✗ Limited automation")
        
        results['categories']['RESPOND'] = {
            'score': min(respond_score, 100),
            'checks': respond_checks,
            'description': 'Incident response and communications'
        }
        
        # RECOVER - Business Continuity
        recover_score = 0
        recover_checks = []
        
        # Check for backup procedures
        backup_files = list(self.project_root.glob('**/backup*')) + list(self.project_root.glob('**/disaster*'))
        if backup_files:
            recover_score += 40
            recover_checks.append(f"✓ Backup/DR configurations found ({len(backup_files)} files)")
        else:
            recover_checks.append("✗ Missing backup procedures")
            results['recommendations'].append("Implement backup and disaster recovery procedures")
        
        # Check for multi-region deployment capability
        if list(self.project_root.glob('**/multi-region/**')):
            recover_score += 30
            recover_checks.append("✓ Multi-region deployment support")
        else:
            recover_checks.append("✗ No multi-region deployment")
        
        # Check for health checks
        if self._check_health_checks():
            recover_score += 30
            recover_checks.append("✓ Health checks configured")
        else:
            recover_checks.append("✗ Missing health checks")
            results['recommendations'].append("Implement comprehensive health checks")
        
        results['categories']['RECOVER'] = {
            'score': min(recover_score, 100),
            'checks': recover_checks,
            'description': 'Recovery planning and communications'
        }
        
        # Calculate overall score
        category_scores = [cat['score'] for cat in results['categories'].values()]
        results['overall_score'] = sum(category_scores) // len(category_scores)
        
        return results
    
    def check_etsi_nfv_sec_compliance(self) -> Dict[str, Any]:
        """Check compliance with ETSI NFV Security (NFV-SEC) specifications."""
        results = {
            'framework': 'ETSI NFV Security',
            'version': 'NFV-SEC 013 v3.1.1',
            'domains': {},
            'overall_score': 0,
            'recommendations': []
        }
        
        # Security Zones and Trust Domains
        zones_score = 0
        zones_checks = []
        
        # Check for network segmentation
        netpol_files = list(self.project_root.glob('**/*network*policy*.yaml'))
        if netpol_files:
            zones_score += 30
            zones_checks.append(f"✓ Network policies for segmentation ({len(netpol_files)} files)")
        else:
            zones_checks.append("✗ Missing network segmentation")
            results['recommendations'].append("Implement network segmentation with NetworkPolicies")
        
        # Check for namespace isolation
        namespace_files = list(self.project_root.glob('**/namespace*.yaml'))
        if namespace_files:
            zones_score += 25
            zones_checks.append(f"✓ Namespace isolation ({len(namespace_files)} files)")
        else:
            zones_checks.append("✗ Limited namespace isolation")
        
        # Check for security contexts
        if self._check_security_contexts():
            zones_score += 25
            zones_checks.append("✓ Container security contexts")
        else:
            zones_checks.append("✗ Missing security contexts")
        
        # Check for pod security standards
        psp_files = list(self.project_root.glob('**/*pod*security*.yaml'))
        if psp_files:
            zones_score += 20
            zones_checks.append(f"✓ Pod security policies ({len(psp_files)} files)")
        else:
            zones_checks.append("✗ Missing pod security policies")
            results['recommendations'].append("Implement Pod Security Standards")
        
        results['domains']['Security_Zones'] = {
            'score': min(zones_score, 100),
            'checks': zones_checks,
            'description': 'Network segmentation and trust domains'
        }
        
        # VNF Security Controls
        vnf_score = 0
        vnf_checks = []
        
        # Check for resource quotas
        quota_files = list(self.project_root.glob('**/resource*quota*.yaml'))
        if quota_files:
            vnf_score += 25
            vnf_checks.append(f"✓ Resource quotas configured ({len(quota_files)} files)")
        else:
            vnf_checks.append("✗ Missing resource quotas")
            results['recommendations'].append("Configure resource quotas for workload isolation")
        
        # Check for admission controllers/webhooks
        webhook_files = list(self.project_root.glob('**/*webhook*.yaml'))
        if webhook_files:
            vnf_score += 25
            vnf_checks.append(f"✓ Admission webhooks ({len(webhook_files)} files)")
        else:
            vnf_checks.append("✗ Missing admission controls")
        
        # Check for RBAC
        rbac_files = list(self.project_root.glob('**/rbac*.yaml'))
        if rbac_files:
            vnf_score += 25
            vnf_checks.append(f"✓ RBAC configurations ({len(rbac_files)} files)")
        else:
            vnf_checks.append("✗ Missing RBAC")
        
        # Check for secrets management
        secret_files = list(self.project_root.glob('**/secret*.yaml'))
        if secret_files:
            vnf_score += 25
            vnf_checks.append(f"✓ Secrets management ({len(secret_files)} files)")
        else:
            vnf_checks.append("✗ Limited secrets management")
            results['recommendations'].append("Implement proper secrets management")
        
        results['domains']['VNF_Security'] = {
            'score': min(vnf_score, 100),
            'checks': vnf_checks,
            'description': 'Virtual Network Function security controls'
        }
        
        # NFVI Security
        nfvi_score = 0
        nfvi_checks = []
        
        # Check for infrastructure security configurations
        security_files = list(self.project_root.glob('**/security/**/*.yaml'))
        if security_files:
            nfvi_score += 40
            nfvi_checks.append(f"✓ Infrastructure security configs ({len(security_files)} files)")
        else:
            nfvi_checks.append("✗ Limited infrastructure security")
        
        # Check for monitoring and observability
        monitoring_files = list(self.project_root.glob('**/monitoring/**/*.yaml'))
        if monitoring_files:
            nfvi_score += 30
            nfvi_checks.append(f"✓ Infrastructure monitoring ({len(monitoring_files)} files)")
        else:
            nfvi_checks.append("✗ Missing infrastructure monitoring")
        
        # Check for compliance with cloud-native security
        if (self.project_root / 'deployments/istio').exists():
            nfvi_score += 30
            nfvi_checks.append("✓ Service mesh security (Istio)")
        else:
            nfvi_checks.append("✗ No service mesh security")
            results['recommendations'].append("Consider implementing service mesh for mTLS and traffic policies")
        
        results['domains']['NFVI_Security'] = {
            'score': min(nfvi_score, 100),
            'checks': nfvi_checks,
            'description': 'NFV Infrastructure security'
        }
        
        # Security Management
        mgmt_score = 0
        mgmt_checks = []
        
        # Check for security automation
        automation_files = list(self.project_root.glob('**/security*.sh')) + list(self.project_root.glob('**/security*.py'))
        if automation_files:
            mgmt_score += 35
            mgmt_checks.append(f"✓ Security automation ({len(automation_files)} files)")
        else:
            mgmt_checks.append("✗ Limited security automation")
        
        # Check for compliance validation
        if (self.project_root / '.github/workflows/security-scan.yml').exists():
            mgmt_score += 35
            mgmt_checks.append("✓ Automated security scanning")
        else:
            mgmt_checks.append("✗ No automated security scanning")
        
        # Check for security documentation
        security_docs = list(self.project_root.glob('**/SECURITY*.md'))
        if security_docs:
            mgmt_score += 30
            mgmt_checks.append(f"✓ Security documentation ({len(security_docs)} files)")
        else:
            mgmt_checks.append("✗ Missing security documentation")
        
        results['domains']['Security_Management'] = {
            'score': min(mgmt_score, 100),
            'checks': mgmt_checks,
            'description': 'Security management and governance'
        }
        
        # Calculate overall score
        domain_scores = [domain['score'] for domain in results['domains'].values()]
        results['overall_score'] = sum(domain_scores) // len(domain_scores)
        
        return results
    
    def check_oran_security_compliance(self) -> Dict[str, Any]:
        """Check compliance with O-RAN Security specifications."""
        results = {
            'framework': 'O-RAN Security',
            'version': 'O-RAN.WG11.Security-v01.00',
            'interfaces': {},
            'overall_score': 0,
            'recommendations': []
        }
        
        # A1 Interface Security
        a1_score = 0
        a1_checks = []
        
        # Check for A1 policy management security
        a1_files = self._find_files_with_content(['a1', 'policy', 'near-rt-ric'])
        if a1_files:
            a1_score += 30
            a1_checks.append(f"✓ A1 interface implementation ({len(a1_files)} files)")
        else:
            a1_checks.append("✗ No A1 interface implementation")
        
        # Check for authentication in A1
        auth_files = self._find_files_with_content(['oauth', 'jwt', 'authentication'])
        if auth_files:
            a1_score += 35
            a1_checks.append(f"✓ Authentication mechanisms ({len(auth_files)} files)")
        else:
            a1_checks.append("✗ Limited authentication")
            results['recommendations'].append("Implement OAuth2/JWT authentication for O-RAN interfaces")
        
        # Check for TLS/mTLS
        tls_files = self._find_files_with_content(['tls', 'mtls', 'certificate'])
        if tls_files:
            a1_score += 35
            a1_checks.append(f"✓ TLS/mTLS implementation ({len(tls_files)} files)")
        else:
            a1_checks.append("✗ Missing TLS implementation")
            results['recommendations'].append("Implement mTLS for all O-RAN interface communications")
        
        results['interfaces']['A1_Interface'] = {
            'score': min(a1_score, 100),
            'checks': a1_checks,
            'description': 'A1 interface security (Non-RT RIC to Near-RT RIC)'
        }
        
        # O1 Interface Security (FCAPS)
        o1_score = 0
        o1_checks = []
        
        # Check for O1 FCAPS implementation
        o1_files = self._find_files_with_content(['o1', 'fcaps', 'netconf', 'yang'])
        if o1_files:
            o1_score += 40
            o1_checks.append(f"✓ O1 interface implementation ({len(o1_files)} files)")
        else:
            o1_checks.append("✗ No O1 interface implementation")
        
        # Check for monitoring and management security
        monitoring_files = list(self.project_root.glob('**/monitoring/**/*.yaml'))
        if monitoring_files:
            o1_score += 30
            o1_checks.append(f"✓ Secure monitoring ({len(monitoring_files)} files)")
        else:
            o1_checks.append("✗ Limited secure monitoring")
        
        # Check for configuration security
        if self._check_security_contexts():
            o1_score += 30
            o1_checks.append("✓ Configuration security controls")
        else:
            o1_checks.append("✗ Missing configuration security")
        
        results['interfaces']['O1_Interface'] = {
            'score': min(o1_score, 100),
            'checks': o1_checks,
            'description': 'O1 interface security (FCAPS management)'
        }
        
        # O2 Interface Security (Cloud Infrastructure)
        o2_score = 0
        o2_checks = []
        
        # Check for cloud infrastructure security
        if (self.project_root / 'deployments').exists():
            o2_score += 30
            o2_checks.append("✓ Cloud deployment configurations")
        else:
            o2_checks.append("✗ No cloud deployment configs")
        
        # Check for container security
        if self._check_security_contexts():
            o2_score += 35
            o2_checks.append("✓ Container security configurations")
        else:
            o2_checks.append("✗ Missing container security")
        
        # Check for network security
        netpol_files = list(self.project_root.glob('**/*network*policy*.yaml'))
        if netpol_files:
            o2_score += 35
            o2_checks.append(f"✓ Network security policies ({len(netpol_files)} files)")
        else:
            o2_checks.append("✗ Missing network security policies")
        
        results['interfaces']['O2_Interface'] = {
            'score': min(o2_score, 100),
            'checks': o2_checks,
            'description': 'O2 interface security (Cloud infrastructure management)'
        }
        
        # E2 Interface Security
        e2_score = 0
        e2_checks = []
        
        # Check for E2 implementation
        e2_files = self._find_files_with_content(['e2', 'e2node', 'ran'])
        if e2_files:
            e2_score += 40
            e2_checks.append(f"✓ E2 interface implementation ({len(e2_files)} files)")
        else:
            e2_checks.append("✗ No E2 interface implementation")
        
        # Check for RAN security
        ran_security = self._find_files_with_content(['ran', 'radio', 'baseband'])
        if ran_security:
            e2_score += 30
            e2_checks.append(f"✓ RAN security considerations ({len(ran_security)} files)")
        else:
            e2_checks.append("✗ Limited RAN security")
        
        # Check for real-time security
        if list(self.project_root.glob('**/monitoring/**')):
            e2_score += 30
            e2_checks.append("✓ Real-time monitoring capabilities")
        else:
            e2_checks.append("✗ No real-time monitoring")
            results['recommendations'].append("Implement real-time security monitoring for RAN functions")
        
        results['interfaces']['E2_Interface'] = {
            'score': min(e2_score, 100),
            'checks': e2_checks,
            'description': 'E2 interface security (Near-RT RIC to E2 Nodes)'
        }
        
        # Calculate overall score
        interface_scores = [interface['score'] for interface in results['interfaces'].values()]
        results['overall_score'] = sum(interface_scores) // len(interface_scores) if interface_scores else 0
        
        return results
    
    def _find_files_with_content(self, patterns: List[str]) -> List[Path]:
        """Find files containing any of the specified patterns."""
        found_files = []
        
        # Search in Go files
        go_files = list(self.project_root.glob('**/*.go'))
        for go_file in go_files:
            try:
                with open(go_file, 'r', encoding='utf-8', errors='ignore') as f:
                    content = f.read().lower()
                    if any(pattern in content for pattern in patterns):
                        found_files.append(go_file)
            except (OSError, UnicodeError) as e:
                # Log specific errors rather than silently continuing
                print(f"Warning: Could not read {go_file}: {e}", file=sys.stderr)
                continue
        
        # Search in YAML files
        yaml_files = list(self.project_root.glob('**/*.yaml')) + list(self.project_root.glob('**/*.yml'))
        for yaml_file in yaml_files:
            try:
                with open(yaml_file, 'r', encoding='utf-8', errors='ignore') as f:
                    content = f.read().lower()
                    if any(pattern in content for pattern in patterns):
                        found_files.append(yaml_file)
            except (OSError, UnicodeError) as e:
                # Log specific errors rather than silently continuing
                print(f"Warning: Could not read {yaml_file}: {e}", file=sys.stderr)
                continue
        
        return found_files
    
    def _check_security_contexts(self) -> bool:
        """Check if security contexts are properly configured."""
        deployment_files = list(self.project_root.glob('**/deployment*.yaml')) + list(self.project_root.glob('**/deployment*.yml'))
        
        security_contexts_found = 0
        for deploy_file in deployment_files:
            try:
                with open(deploy_file, 'r') as f:
                    content = f.read().lower()
                    if 'securitycontext' in content and ('runasnonroot' in content or 'runasuser' in content):
                        security_contexts_found += 1
            except (OSError, UnicodeError) as e:
                # Log specific errors rather than silently continuing
                print(f"Warning: Could not read {deploy_file}: {e}", file=sys.stderr)
                continue
        
        return security_contexts_found > 0
    
    def _check_health_checks(self) -> bool:
        """Check if health checks are configured."""
        deployment_files = list(self.project_root.glob('**/deployment*.yaml'))
        
        health_checks_found = 0
        for deploy_file in deployment_files:
            try:
                with open(deploy_file, 'r') as f:
                    content = f.read().lower()
                    if 'livenessprobe' in content or 'readinessprobe' in content:
                        health_checks_found += 1
            except (OSError, UnicodeError) as e:
                # Log specific errors rather than silently continuing
                print(f"Warning: Could not read {deploy_file}: {e}", file=sys.stderr)
                continue
        
        return health_checks_found > 0

def main():
    parser = argparse.ArgumentParser(description='Check telecommunications security compliance')
    parser.add_argument('--framework', choices=['nist-csf', 'etsi-nfv-sec', 'oran-security'], 
                       required=True, help='Compliance framework to check')
    parser.add_argument('--profile', choices=['telecom', 'cloud-native', 'standard'], 
                       default='telecom', help='Compliance profile')
    parser.add_argument('--output', help='Output file for results (JSON)')
    parser.add_argument('--project-root', default='.', help='Project root directory')
    
    args = parser.parse_args()
    
    if not os.path.exists(args.project_root):
        print(f"Error: Project root '{args.project_root}' does not exist")
        sys.exit(1)
    
    checker = TelecomComplianceChecker(args.project_root)
    
    # Run compliance check
    if args.framework == 'nist-csf':
        results = checker.check_nist_csf_compliance()
    elif args.framework == 'etsi-nfv-sec':
        results = checker.check_etsi_nfv_sec_compliance()
    elif args.framework == 'oran-security':
        results = checker.check_oran_security_compliance()
    
    # Print results
    print(f"=" * 60)
    print(f"TELECOMMUNICATIONS SECURITY COMPLIANCE CHECK")
    print(f"Framework: {results['framework']}")
    print(f"Overall Score: {results['overall_score']}/100")
    print(f"=" * 60)
    
    # Print category/domain/interface results
    categories_key = 'categories' if 'categories' in results else 'domains' if 'domains' in results else 'interfaces'
    
    for name, details in results.get(categories_key, {}).items():
        print(f"\n{name.replace('_', ' ').title()}: {details['score']}/100")
        print(f"  Description: {details['description']}")
        for check in details['checks']:
            print(f"    {check}")
    
    if results['recommendations']:
        print(f"\nRECOMMENDATIONS:")
        for i, rec in enumerate(results['recommendations'], 1):
            print(f"  {i}. {rec}")
    
    # Determine compliance status
    if results['overall_score'] >= 80:
        status = "COMPLIANT"
        exit_code = 0
    elif results['overall_score'] >= 60:
        status = "PARTIALLY COMPLIANT"
        exit_code = 0
    else:
        status = "NON-COMPLIANT"
        exit_code = 1
    
    print(f"\nCOMPLIANCE STATUS: {status}")
    
    # Save results if requested
    if args.output:
        results['compliance_status'] = status
        results['compliance_score'] = results['overall_score']
        results['check_timestamp'] = os.path.getctime(args.project_root)
        
        with open(args.output, 'w') as f:
            json.dump(results, f, indent=2, default=str)
        
        print(f"Results saved to: {args.output}")
    
    sys.exit(exit_code)

if __name__ == '__main__':
    main()
#!/usr/bin/env python3
"""
Critical Vulnerability Checker for Nephoran Intent Operator

This script analyzes security scan results to identify critical vulnerabilities
that require immediate attention and can block CI/CD pipelines.
"""

import json
import os
import sys
import argparse
from typing import Dict, List, Any
from pathlib import Path

class VulnerabilityChecker:
    def __init__(self, critical_threshold: float = 7.0, max_critical: int = 0, max_high: int = 5):
        self.critical_threshold = critical_threshold
        self.max_critical = max_critical
        self.max_high = max_high
        self.critical_vulns = []
        self.high_vulns = []
        self.blocking_issues = []
    
    def check_gosec_report(self, report_path: str) -> Dict[str, Any]:
        """Check gosec report for critical security issues."""
        try:
            with open(report_path, 'r') as f:
                data = json.load(f)
            
            issues = data.get('Issues', [])
            critical_issues = []
            high_issues = []
            
            for issue in issues:
                severity = issue.get('severity', '').upper()
                rule_id = issue.get('rule_id', '')
                file_path = issue.get('file', '')
                line = issue.get('line', '')
                
                issue_detail = {
                    'tool': 'gosec',
                    'rule_id': rule_id,
                    'file': file_path,
                    'line': line,
                    'description': issue.get('details', ''),
                    'severity': severity
                }
                
                # Critical security patterns
                critical_patterns = [
                    'G101',  # Look for hardcoded credentials
                    'G102',  # Bind to all interfaces
                    'G103',  # Audit the use of unsafe block
                    'G104',  # Audit errors not checked
                    'G107',  # Url provided to HTTP request as taint input
                    'G301',  # Poor file permissions used when creating a directory
                    'G302',  # Poor file permissions used with chmod
                    'G303',  # Creating tempfile using a predictable path
                    'G304',  # File path provided as taint input
                    'G305',  # File traversal when extracting zip/tar archive
                    'G306',  # Poor file permissions used when writing to a new file
                    'G401',  # Detect the usage of DES, RC4, MD5 or SHA1
                    'G402',  # Look for bad TLS connection settings
                    'G403',  # Ensure minimum RSA key length of 2048 bits
                    'G404',  # Insecure random number source (rand)
                    'G501',  # Import blacklist: crypto/md5
                    'G502',  # Import blacklist: crypto/des
                    'G503',  # Import blacklist: crypto/rc4
                    'G504',  # Import blacklist: net/http/cgi
                    'G505',  # Import blacklist: crypto/sha1
                    'G601',  # Implicit memory aliasing in RangeStmt
                ]
                
                if rule_id in critical_patterns or severity == 'HIGH':
                    if rule_id in ['G101', 'G102', 'G401', 'G402', 'G404']:
                        critical_issues.append(issue_detail)
                        self.critical_vulns.append(issue_detail)
                    else:
                        high_issues.append(issue_detail)
                        self.high_vulns.append(issue_detail)
            
            return {
                'tool': 'gosec',
                'critical_count': len(critical_issues),
                'high_count': len(high_issues),
                'critical_issues': critical_issues,
                'high_issues': high_issues
            }
            
        except Exception as e:
            return {'tool': 'gosec', 'error': str(e)}
    
    def check_trivy_report(self, report_path: str) -> Dict[str, Any]:
        """Check Trivy report for critical vulnerabilities."""
        try:
            with open(report_path, 'r') as f:
                data = json.load(f)
            
            results = data.get('Results', [])
            critical_vulns = []
            high_vulns = []
            
            for result in results:
                target = result.get('Target', '')
                vulnerabilities = result.get('Vulnerabilities', [])
                
                for vuln in vulnerabilities:
                    severity = vuln.get('Severity', '').upper()
                    cvss_score = 0
                    
                    # Extract CVSS score
                    if 'CVSS' in vuln:
                        cvss_data = vuln['CVSS']
                        if isinstance(cvss_data, dict):
                            for version, data in cvss_data.items():
                                if isinstance(data, dict) and 'Score' in data:
                                    cvss_score = max(cvss_score, data['Score'])
                    
                    vuln_detail = {
                        'tool': 'trivy',
                        'target': target,
                        'vulnerability_id': vuln.get('VulnerabilityID', ''),
                        'package_name': vuln.get('PkgName', ''),
                        'installed_version': vuln.get('InstalledVersion', ''),
                        'fixed_version': vuln.get('FixedVersion', ''),
                        'title': vuln.get('Title', ''),
                        'description': vuln.get('Description', '')[:200] + '...' if len(vuln.get('Description', '')) > 200 else vuln.get('Description', ''),
                        'severity': severity,
                        'cvss_score': cvss_score,
                        'references': vuln.get('References', [])[:3]  # Limit references
                    }
                    
                    if severity == 'CRITICAL' or cvss_score >= 9.0:
                        critical_vulns.append(vuln_detail)
                        self.critical_vulns.append(vuln_detail)
                    elif severity == 'HIGH' or cvss_score >= 7.0:
                        high_vulns.append(vuln_detail)
                        self.high_vulns.append(vuln_detail)
            
            return {
                'tool': 'trivy',
                'target': data.get('ArtifactName', 'unknown'),
                'critical_count': len(critical_vulns),
                'high_count': len(high_vulns),
                'critical_vulnerabilities': critical_vulns,
                'high_vulnerabilities': high_vulns
            }
            
        except Exception as e:
            return {'tool': 'trivy', 'error': str(e)}
    
    def check_gitleaks_report(self, report_path: str) -> Dict[str, Any]:
        """Check gitleaks report for exposed secrets."""
        try:
            with open(report_path, 'r') as f:
                content = f.read().strip()
                if not content:
                    return {'tool': 'gitleaks', 'critical_count': 0, 'high_count': 0, 'secrets': []}
                
                secrets = []
                for line in content.split('\n'):
                    if line.strip():
                        secrets.append(json.loads(line))
            
            critical_secrets = []
            high_secrets = []
            
            # Patterns that are considered critical
            critical_patterns = [
                'private-key', 'api-key', 'aws-access-token', 'github-pat',
                'slack-access-token', 'stripe-access-token', 'password'
            ]
            
            for secret in secrets:
                rule_id = secret.get('RuleID', '').lower()
                secret_detail = {
                    'tool': 'gitleaks',
                    'rule_id': secret.get('RuleID', ''),
                    'file': secret.get('File', ''),
                    'line_number': secret.get('StartLine', 0),
                    'commit': secret.get('Commit', ''),
                    'description': secret.get('Description', ''),
                    'match': secret.get('Match', '')[:100] + '...' if len(secret.get('Match', '')) > 100 else secret.get('Match', '')
                }
                
                if any(pattern in rule_id for pattern in critical_patterns):
                    critical_secrets.append(secret_detail)
                    self.critical_vulns.append(secret_detail)
                else:
                    high_secrets.append(secret_detail)
                    self.high_vulns.append(secret_detail)
            
            return {
                'tool': 'gitleaks',
                'critical_count': len(critical_secrets),
                'high_count': len(high_secrets),
                'critical_secrets': critical_secrets,
                'high_secrets': high_secrets
            }
            
        except Exception as e:
            return {'tool': 'gitleaks', 'error': str(e)}
    
    def check_govulncheck_report(self, report_path: str) -> Dict[str, Any]:
        """Check govulncheck report for Go-specific vulnerabilities."""
        try:
            with open(report_path, 'r') as f:
                content = f.read().strip()
                if not content:
                    return {'tool': 'govulncheck', 'critical_count': 0, 'high_count': 0}
                
                data = json.loads(content)
            
            vulns = data.get('Vulns', [])
            critical_vulns = []
            high_vulns = []
            
            for vuln in vulns:
                osv = vuln.get('OSV', {})
                vuln_detail = {
                    'tool': 'govulncheck',
                    'vulnerability_id': osv.get('id', ''),
                    'summary': osv.get('summary', ''),
                    'details': osv.get('details', '')[:200] + '...' if len(osv.get('details', '')) > 200 else osv.get('details', ''),
                    'affected_packages': [pkg.get('package', {}).get('name', '') for pkg in osv.get('affected', [])],
                    'severity': osv.get('database_specific', {}).get('severity', 'UNKNOWN')
                }
                
                # Check if this is a critical vulnerability
                severity = vuln_detail['severity'].upper()
                if severity in ['CRITICAL', 'HIGH'] or 'critical' in vuln_detail['summary'].lower():
                    if 'critical' in vuln_detail['summary'].lower() or severity == 'CRITICAL':
                        critical_vulns.append(vuln_detail)
                        self.critical_vulns.append(vuln_detail)
                    else:
                        high_vulns.append(vuln_detail)
                        self.high_vulns.append(vuln_detail)
            
            return {
                'tool': 'govulncheck',
                'critical_count': len(critical_vulns),
                'high_count': len(high_vulns),
                'critical_vulnerabilities': critical_vulns,
                'high_vulnerabilities': high_vulns
            }
            
        except Exception as e:
            return {'tool': 'govulncheck', 'error': str(e)}
    
    def analyze_reports(self, input_dir: str) -> Dict[str, Any]:
        """Analyze all available security reports."""
        results = {}
        
        # Report checkers mapping
        report_checkers = {
            'gosec-report.json': self.check_gosec_report,
            'gitleaks-report.json': self.check_gitleaks_report,
            'govulncheck-report.json': self.check_govulncheck_report,
        }
        
        # Check Trivy reports for each service
        services = ['llm-processor', 'nephio-bridge', 'oran-adaptor', 'rag-api']
        for service in services:
            trivy_report = f'trivy-{service}.json'
            trivy_path = os.path.join(input_dir, trivy_report)
            if os.path.exists(trivy_path):
                results[f'trivy-{service}'] = self.check_trivy_report(trivy_path)
        
        # Check other reports
        for report_file, checker_func in report_checkers.items():
            report_path = os.path.join(input_dir, report_file)
            if os.path.exists(report_path):
                results[report_file.replace('.json', '')] = checker_func(report_path)
        
        return results
    
    def generate_blocking_summary(self, results: Dict[str, Any]) -> Dict[str, Any]:
        """Generate summary of blocking issues."""
        total_critical = len(self.critical_vulns)
        total_high = len(self.high_vulns)
        
        # Determine if build should be blocked
        block_build = False
        block_reasons = []
        
        if total_critical > self.max_critical:
            block_build = True
            block_reasons.append(f"Found {total_critical} critical vulnerabilities (max allowed: {self.max_critical})")
        
        if total_high > self.max_high:
            block_build = True
            block_reasons.append(f"Found {total_high} high vulnerabilities (max allowed: {self.max_high})")
        
        # Check for specific blocking patterns
        for vuln in self.critical_vulns:
            if vuln['tool'] == 'gitleaks' and 'private-key' in vuln.get('rule_id', '').lower():
                block_build = True
                block_reasons.append("Private key found in repository")
                break
        
        for vuln in self.critical_vulns:
            if vuln['tool'] == 'gosec' and vuln.get('rule_id') == 'G101':
                block_build = True
                block_reasons.append("Hardcoded credentials detected in code")
                break
        
        return {
            'block_build': block_build,
            'block_reasons': block_reasons,
            'total_critical': total_critical,
            'total_high': total_high,
            'critical_vulnerabilities': self.critical_vulns[:10],  # Show top 10
            'high_vulnerabilities': self.high_vulns[:10],  # Show top 10
            'summary_by_tool': {
                tool: {
                    'critical': result.get('critical_count', 0),
                    'high': result.get('high_count', 0),
                    'has_error': 'error' in result
                }
                for tool, result in results.items()
            }
        }
    
    def print_summary(self, summary: Dict[str, Any]) -> None:
        """Print vulnerability summary to console."""
        print("=" * 60)
        print("CRITICAL VULNERABILITY ASSESSMENT")
        print("=" * 60)
        
        print(f"Total Critical Vulnerabilities: {summary['total_critical']}")
        print(f"Total High Vulnerabilities: {summary['total_high']}")
        print(f"Build Blocked: {'YES' if summary['block_build'] else 'NO'}")
        
        if summary['block_reasons']:
            print("\nBLOCKING REASONS:")
            for reason in summary['block_reasons']:
                print(f"  • {reason}")
        
        print("\nSUMMARY BY TOOL:")
        for tool, stats in summary['summary_by_tool'].items():
            status = "ERROR" if stats['has_error'] else "OK"
            print(f"  {tool:20} Critical: {stats['critical']:2} High: {stats['high']:2} [{status}]")
        
        if summary['critical_vulnerabilities']:
            print(f"\nTOP CRITICAL VULNERABILITIES:")
            for i, vuln in enumerate(summary['critical_vulnerabilities'][:5], 1):
                tool = vuln['tool'].upper()
                if vuln['tool'] == 'trivy':
                    desc = f"{vuln.get('vulnerability_id', '')}: {vuln.get('title', '')}"
                elif vuln['tool'] == 'gosec':
                    desc = f"{vuln.get('rule_id', '')}: {vuln.get('description', '')}"
                elif vuln['tool'] == 'gitleaks':
                    desc = f"{vuln.get('rule_id', '')}: {vuln.get('description', '')}"
                else:
                    desc = vuln.get('summary', vuln.get('description', 'Unknown'))
                
                print(f"  {i}. [{tool}] {desc[:80]}...")
        
        print("=" * 60)

def main():
    parser = argparse.ArgumentParser(description='Check for critical vulnerabilities that block deployment')
    parser.add_argument('--input-dir', required=True, help='Directory containing security scan reports')
    parser.add_argument('--fail-on-critical', action='store_true', help='Exit with code 1 if critical vulnerabilities found')
    parser.add_argument('--max-critical', type=int, default=0, help='Maximum allowed critical vulnerabilities')
    parser.add_argument('--max-high', type=int, default=5, help='Maximum allowed high vulnerabilities')
    parser.add_argument('--output', help='Output file for detailed results (JSON)')
    
    args = parser.parse_args()
    
    if not os.path.exists(args.input_dir):
        print(f"Error: Input directory '{args.input_dir}' does not exist")
        sys.exit(1)
    
    # Initialize checker
    checker = VulnerabilityChecker(
        max_critical=args.max_critical,
        max_high=args.max_high
    )
    
    # Analyze reports
    results = checker.analyze_reports(args.input_dir)
    summary = checker.generate_blocking_summary(results)
    
    # Print summary
    checker.print_summary(summary)
    
    # Save detailed results if requested
    if args.output:
        detailed_results = {
            'summary': summary,
            'detailed_results': results,
            'analysis_timestamp': os.path.getctime(args.input_dir),
            'input_directory': args.input_dir
        }
        
        with open(args.output, 'w') as f:
            json.dump(detailed_results, f, indent=2, default=str)
        
        print(f"\nDetailed results saved to: {args.output}")
    
    # Exit with appropriate code
    if summary['block_build'] and args.fail_on_critical:
        print("\nEXITING WITH ERROR CODE 1 DUE TO CRITICAL VULNERABILITIES")
        sys.exit(1)
    elif summary['block_build']:
        print("\nWARNING: Critical vulnerabilities found but not failing build (use --fail-on-critical)")
        sys.exit(0)
    else:
        print("\nSUCCESS: No blocking vulnerabilities found")
        sys.exit(0)

if __name__ == '__main__':
    main()
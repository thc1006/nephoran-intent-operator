#!/usr/bin/env python3
"""
Security Report Generator for Nephoran Intent Operator

This script aggregates security scan results from multiple tools and generates
comprehensive HTML and JSON reports with telecommunications security compliance.
"""

import json
import os
import sys
import argparse
import datetime
from pathlib import Path
from typing import Dict, List, Any
# Use defusedxml for secure XML parsing to prevent XXE attacks
try:
    from defusedxml.ElementTree import parse as safe_parse
    import defusedxml.ElementTree as ET
except ImportError:
    # Fallback to standard library with warning
    import xml.etree.ElementTree as ET
    import warnings
    warnings.warn("defusedxml not available. Using xml.etree.ElementTree with potential security risks. Install defusedxml for secure XML parsing.", 
                  category=UserWarning)
    safe_parse = ET.parse

def parse_gosec_report(report_path: str) -> Dict[str, Any]:
    """Parse gosec JSON report."""
    try:
        with open(report_path, 'r') as f:
            data = json.load(f)
        
        issues = data.get('Issues', [])
        stats = data.get('Stats', {})
        
        return {
            'tool': 'gosec',
            'total_issues': len(issues),
            'critical_issues': len([i for i in issues if i.get('severity', '').upper() == 'HIGH']),
            'high_issues': len([i for i in issues if i.get('severity', '').upper() == 'MEDIUM']),
            'medium_issues': len([i for i in issues if i.get('severity', '').upper() == 'LOW']),
            'files_scanned': stats.get('files', 0),
            'lines_scanned': stats.get('lines', 0),
            'issues': issues[:10]  # Top 10 issues for summary
        }
    except Exception as e:
        print(f"Error parsing gosec report: {e}")
        return {'tool': 'gosec', 'error': str(e)}

def parse_trivy_report(report_path: str) -> Dict[str, Any]:
    """Parse Trivy JSON report."""
    try:
        with open(report_path, 'r') as f:
            data = json.load(f)
        
        results = data.get('Results', [])
        critical_count = 0
        high_count = 0
        medium_count = 0
        low_count = 0
        
        for result in results:
            vulnerabilities = result.get('Vulnerabilities', [])
            for vuln in vulnerabilities:
                severity = vuln.get('Severity', '').upper()
                if severity == 'CRITICAL':
                    critical_count += 1
                elif severity == 'HIGH':
                    high_count += 1
                elif severity == 'MEDIUM':
                    medium_count += 1
                elif severity == 'LOW':
                    low_count += 1
        
        return {
            'tool': 'trivy',
            'total_issues': critical_count + high_count + medium_count + low_count,
            'critical_issues': critical_count,
            'high_issues': high_count,
            'medium_issues': medium_count,
            'low_issues': low_count,
            'results': results[:5]  # Top 5 results for summary
        }
    except Exception as e:
        print(f"Error parsing Trivy report: {e}")
        return {'tool': 'trivy', 'error': str(e)}

def parse_nancy_report(report_path: str) -> Dict[str, Any]:
    """Parse Nancy JSON report."""
    try:
        with open(report_path, 'r') as f:
            data = json.load(f)
        
        vulnerable_packages = []
        if isinstance(data, list):
            vulnerable_packages = data
        elif isinstance(data, dict) and 'vulnerablePackages' in data:
            vulnerable_packages = data['vulnerablePackages']
        
        critical_count = 0
        high_count = 0
        medium_count = 0
        
        for package in vulnerable_packages:
            vulnerabilities = package.get('vulnerabilities', [])
            for vuln in vulnerabilities:
                cvss_score = vuln.get('cvssScore', 0)
                if cvss_score >= 9.0:
                    critical_count += 1
                elif cvss_score >= 7.0:
                    high_count += 1
                elif cvss_score >= 4.0:
                    medium_count += 1
        
        return {
            'tool': 'nancy',
            'total_issues': len(vulnerable_packages),
            'critical_issues': critical_count,
            'high_issues': high_count,
            'medium_issues': medium_count,
            'vulnerable_packages': vulnerable_packages[:10]
        }
    except Exception as e:
        print(f"Error parsing Nancy report: {e}")
        return {'tool': 'nancy', 'error': str(e)}

def parse_gitleaks_report(report_path: str) -> Dict[str, Any]:
    """Parse gitleaks JSON report."""
    try:
        with open(report_path, 'r') as f:
            content = f.read().strip()
            if not content:
                return {'tool': 'gitleaks', 'total_issues': 0, 'secrets': []}
            
            # Parse line by line as gitleaks outputs JSONL
            secrets = []
            for line in content.split('\n'):
                if line.strip():
                    secrets.append(json.loads(line))
        
        return {
            'tool': 'gitleaks',
            'total_issues': len(secrets),
            'critical_issues': len([s for s in secrets if 'password' in s.get('RuleID', '').lower()]),
            'high_issues': len([s for s in secrets if 'token' in s.get('RuleID', '').lower()]),
            'secrets': secrets[:10]
        }
    except Exception as e:
        print(f"Error parsing gitleaks report: {e}")
        return {'tool': 'gitleaks', 'error': str(e)}

def parse_govulncheck_report(report_path: str) -> Dict[str, Any]:
    """Parse govulncheck JSON report."""
    try:
        with open(report_path, 'r') as f:
            content = f.read().strip()
            if not content:
                return {'tool': 'govulncheck', 'total_issues': 0, 'vulnerabilities': []}
            
            data = json.loads(content)
        
        vulns = data.get('Vulns', [])
        
        return {
            'tool': 'govulncheck',
            'total_issues': len(vulns),
            'critical_issues': len([v for v in vulns if v.get('OSV', {}).get('database_specific', {}).get('severity') == 'HIGH']),
            'high_issues': len([v for v in vulns if v.get('OSV', {}).get('database_specific', {}).get('severity') == 'MODERATE']),
            'vulnerabilities': vulns[:10]
        }
    except Exception as e:
        print(f"Error parsing govulncheck report: {e}")
        return {'tool': 'govulncheck', 'error': str(e)}

def generate_html_report(scan_results: Dict[str, Any], output_path: str) -> None:
    """Generate comprehensive HTML security report."""
    
    html_template = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nephoran Intent Operator - Security Scan Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
        }
        .header .subtitle {
            margin: 10px 0 0 0;
            opacity: 0.9;
            font-size: 1.1em;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .metric-card {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
            border-left: 4px solid #2196F3;
        }
        .metric-card.critical {
            border-left-color: #f44336;
        }
        .metric-card.high {
            border-left-color: #ff9800;
        }
        .metric-card.medium {
            border-left-color: #ffeb3b;
        }
        .metric-card.low {
            border-left-color: #4caf50;
        }
        .metric-value {
            font-size: 3em;
            font-weight: bold;
            margin: 0;
        }
        .metric-label {
            color: #666;
            font-size: 1.1em;
            margin: 5px 0 0 0;
        }
        .section {
            background: white;
            margin-bottom: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .section-header {
            background: #f8f9fa;
            padding: 20px;
            border-bottom: 1px solid #e9ecef;
        }
        .section-header h2 {
            margin: 0;
            color: #495057;
        }
        .section-content {
            padding: 20px;
        }
        .tool-result {
            margin-bottom: 25px;
            padding: 15px;
            border-left: 3px solid #dee2e6;
            background: #f8f9fa;
        }
        .tool-name {
            font-weight: bold;
            font-size: 1.2em;
            color: #495057;
            margin-bottom: 10px;
        }
        .issue-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
            gap: 10px;
            margin-top: 10px;
        }
        .issue-count {
            text-align: center;
            padding: 8px;
            border-radius: 4px;
            font-weight: bold;
        }
        .critical { background: #ffebee; color: #c62828; }
        .high { background: #fff3e0; color: #ef6c00; }
        .medium { background: #fffde7; color: #f57f17; }
        .low { background: #e8f5e8; color: #2e7d32; }
        .compliance-status {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .compliance-item {
            padding: 15px;
            border: 1px solid #dee2e6;
            border-radius: 6px;
        }
        .compliance-item.passed {
            border-left: 4px solid #28a745;
            background: #d4edda;
        }
        .compliance-item.warning {
            border-left: 4px solid #ffc107;
            background: #fff3cd;
        }
        .compliance-item.failed {
            border-left: 4px solid #dc3545;
            background: #f8d7da;
        }
        .recommendations {
            list-style: none;
            padding: 0;
        }
        .recommendations li {
            padding: 10px;
            margin: 10px 0;
            border-left: 4px solid #2196F3;
            background: #e3f2fd;
        }
        .recommendations li.critical {
            border-left-color: #f44336;
            background: #ffebee;
        }
        .recommendations li.high {
            border-left-color: #ff9800;
            background: #fff3e0;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 15px;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
        }
        .footer {
            margin-top: 40px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
            text-align: center;
            color: #666;
        }
        @media (max-width: 768px) {
            .summary {
                grid-template-columns: 1fr;
            }
            body {
                padding: 10px;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Security Scan Report</h1>
        <div class="subtitle">Nephoran Intent Operator - Comprehensive Vulnerability Assessment</div>
        <div class="subtitle">Generated: {scan_date}</div>
    </div>

    <div class="summary">
        <div class="metric-card critical">
            <div class="metric-value">{total_critical}</div>
            <div class="metric-label">Critical Issues</div>
        </div>
        <div class="metric-card high">
            <div class="metric-value">{total_high}</div>
            <div class="metric-label">High Issues</div>
        </div>
        <div class="metric-card medium">
            <div class="metric-value">{total_medium}</div>
            <div class="metric-label">Medium Issues</div>
        </div>
        <div class="metric-card low">
            <div class="metric-value">{total_low}</div>
            <div class="metric-label">Low Issues</div>
        </div>
    </div>

    <div class="section">
        <div class="section-header">
            <h2>Executive Summary</h2>
        </div>
        <div class="section-content">
            <p>This security assessment analyzed the Nephoran Intent Operator codebase and container images using multiple industry-standard security scanning tools. The scan included static application security testing (SAST), dependency vulnerability analysis, container image scanning, and telecommunications security compliance checks.</p>
            
            <h3>Risk Assessment</h3>
            <p>Overall Risk Level: <strong>{risk_level}</strong></p>
            
            <h3>Key Findings</h3>
            <ul>
                {key_findings}
            </ul>
        </div>
    </div>

    <div class="section">
        <div class="section-header">
            <h2>Detailed Scan Results</h2>
        </div>
        <div class="section-content">
            {tool_results}
        </div>
    </div>

    <div class="section">
        <div class="section-header">
            <h2>Telecommunications Security Compliance</h2>
        </div>
        <div class="section-content">
            <div class="compliance-status">
                <div class="compliance-item {nist_status}">
                    <h4>NIST Cybersecurity Framework</h4>
                    <p>Assessment of compliance with NIST CSF requirements for telecommunications systems.</p>
                    <p><strong>Status:</strong> {nist_compliance}</p>
                </div>
                <div class="compliance-item {etsi_status}">
                    <h4>ETSI NFV Security</h4>
                    <p>Validation against ETSI NFV-SEC security specifications for network functions.</p>
                    <p><strong>Status:</strong> {etsi_compliance}</p>
                </div>
                <div class="compliance-item {oran_status}">
                    <h4>O-RAN Security</h4>
                    <p>Compliance with O-RAN Alliance security requirements and specifications.</p>
                    <p><strong>Status:</strong> {oran_compliance}</p>
                </div>
            </div>
        </div>
    </div>

    <div class="section">
        <div class="section-header">
            <h2>Recommendations</h2>
        </div>
        <div class="section-content">
            <ul class="recommendations">
                {recommendations}
            </ul>
        </div>
    </div>

    <div class="footer">
        <p>Generated by Nephoran Security Scanner | {scan_date}</p>
        <p>For questions or assistance, contact the security team</p>
    </div>
</body>
</html>
"""

    # Calculate totals
    total_critical = scan_results['summary']['critical_issues']
    total_high = scan_results['summary']['high_issues']
    total_medium = scan_results['summary']['medium_issues']
    total_low = scan_results['summary']['low_issues']
    
    # Determine risk level
    if total_critical > 0:
        risk_level = "CRITICAL"
    elif total_high > 10:
        risk_level = "HIGH"
    elif total_high > 0:
        risk_level = "MEDIUM"
    else:
        risk_level = "LOW"
    
    # Generate key findings
    key_findings = []
    if total_critical > 0:
        key_findings.append(f"<li><strong>Critical:</strong> {total_critical} critical vulnerabilities require immediate attention</li>")
    if total_high > 0:
        key_findings.append(f"<li><strong>High:</strong> {total_high} high-severity issues should be addressed within 7 days</li>")
    
    tools_with_issues = [tool for tool, results in scan_results['tools'].items() if not results.get('error') and results.get('total_issues', 0) > 0]
    if tools_with_issues:
        key_findings.append(f"<li><strong>Scope:</strong> Issues found by {', '.join(tools_with_issues)}</li>")
    
    # Generate tool results
    tool_results_html = ""
    for tool_name, results in scan_results['tools'].items():
        if results.get('error'):
            tool_results_html += f"""
            <div class="tool-result">
                <div class="tool-name">{tool_name.upper()}</div>
                <p><strong>Error:</strong> {results['error']}</p>
            </div>
            """
        else:
            tool_results_html += f"""
            <div class="tool-result">
                <div class="tool-name">{tool_name.upper()}</div>
                <div class="issue-grid">
                    <div class="issue-count critical">Critical: {results.get('critical_issues', 0)}</div>
                    <div class="issue-count high">High: {results.get('high_issues', 0)}</div>
                    <div class="issue-count medium">Medium: {results.get('medium_issues', 0)}</div>
                    <div class="issue-count low">Low: {results.get('low_issues', 0)}</div>
                </div>
            </div>
            """
    
    # Generate recommendations
    recommendations_html = ""
    if total_critical > 0:
        recommendations_html += '<li class="critical">URGENT: Address all critical vulnerabilities before production deployment</li>'
    if total_high > 5:
        recommendations_html += '<li class="high">Prioritize remediation of high-severity vulnerabilities</li>'
    
    recommendations_html += """
        <li>Implement automated security scanning in CI/CD pipeline</li>
        <li>Establish regular security review cycles (monthly)</li>
        <li>Monitor CVE databases for new vulnerabilities affecting dependencies</li>
        <li>Consider implementing runtime security monitoring</li>
        <li>Review and update container base images regularly</li>
        <li>Enhance secret management practices</li>
        <li>Implement network segmentation and Zero Trust architecture</li>
    """
    
    # Compliance status (placeholder - would be populated from actual compliance checks)
    compliance = scan_results.get('compliance', {})
    nist_compliance = compliance.get('nist_csf', 'Not Checked')
    etsi_compliance = compliance.get('etsi_nfv_sec', 'Not Checked')
    oran_compliance = compliance.get('oran_security', 'Not Checked')
    
    def get_status_class(status):
        if status.lower() in ['passed', 'compliant']:
            return 'passed'
        elif status.lower() in ['warning', 'partial']:
            return 'warning'
        else:
            return 'failed'
    
    # Generate HTML
    html_content = html_template.format(
        scan_date=datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S UTC'),
        total_critical=total_critical,
        total_high=total_high,
        total_medium=total_medium,
        total_low=total_low,
        risk_level=risk_level,
        key_findings=''.join(key_findings) if key_findings else '<li>No significant security issues detected</li>',
        tool_results=tool_results_html,
        nist_status=get_status_class(nist_compliance),
        nist_compliance=nist_compliance,
        etsi_status=get_status_class(etsi_compliance),
        etsi_compliance=etsi_compliance,
        oran_status=get_status_class(oran_compliance),
        oran_compliance=oran_compliance,
        recommendations=recommendations_html
    )
    
    with open(output_path, 'w') as f:
        f.write(html_content)

def generate_json_report(scan_results: Dict[str, Any], output_path: str) -> None:
    """Generate JSON security report."""
    report = {
        'scan_metadata': {
            'timestamp': datetime.datetime.now().isoformat(),
            'project': 'nephoran-intent-operator',
            'version': '1.0.0',
            'scanner_version': 'nephoran-security-scanner-1.0'
        },
        'summary': scan_results['summary'],
        'tools': scan_results['tools'],
        'compliance': scan_results.get('compliance', {}),
        'recommendations': [
            'Address all critical vulnerabilities immediately',
            'Implement automated security scanning in CI/CD',
            'Establish regular security review cycles',
            'Monitor CVE databases for new vulnerabilities',
            'Update container base images regularly',
            'Enhance secret management practices'
        ]
    }
    
    with open(output_path, 'w') as f:
        json.dump(report, f, indent=2, default=str)

def main():
    parser = argparse.ArgumentParser(description='Generate comprehensive security report from scan results')
    parser.add_argument('--input-dir', required=True, help='Directory containing security scan reports')
    parser.add_argument('--output', required=True, help='Output file path')
    parser.add_argument('--format', choices=['html', 'json'], default='html', help='Output format')
    
    args = parser.parse_args()
    
    if not os.path.exists(args.input_dir):
        print(f"Error: Input directory '{args.input_dir}' does not exist")
        sys.exit(1)
    
    # Initialize results structure
    scan_results = {
        'summary': {
            'critical_issues': 0,
            'high_issues': 0,
            'medium_issues': 0,
            'low_issues': 0,
            'total_issues': 0
        },
        'tools': {},
        'compliance': {}
    }
    
    # Parse available reports
    report_files = {
        'gosec-report.json': parse_gosec_report,
        'trivy-llm-processor.json': parse_trivy_report,
        'trivy-nephio-bridge.json': parse_trivy_report,
        'trivy-oran-adaptor.json': parse_trivy_report,
        'trivy-rag-api.json': parse_trivy_report,
        'nancy-report.json': parse_nancy_report,
        'gitleaks-report.json': parse_gitleaks_report,
        'govulncheck-report.json': parse_govulncheck_report
    }
    
    for report_file, parser_func in report_files.items():
        report_path = os.path.join(args.input_dir, report_file)
        if os.path.exists(report_path):
            try:
                result = parser_func(report_path)
                tool_name = result['tool']
                if 'trivy' in report_file:
                    service_name = report_file.replace('trivy-', '').replace('.json', '')
                    tool_name = f"trivy-{service_name}"
                
                scan_results['tools'][tool_name] = result
                
                # Aggregate summary
                scan_results['summary']['critical_issues'] += result.get('critical_issues', 0)
                scan_results['summary']['high_issues'] += result.get('high_issues', 0)
                scan_results['summary']['medium_issues'] += result.get('medium_issues', 0)
                scan_results['summary']['low_issues'] += result.get('low_issues', 0)
                scan_results['summary']['total_issues'] += result.get('total_issues', 0)
                
            except Exception as e:
                print(f"Warning: Failed to parse {report_file}: {e}")
    
    # Check for compliance reports
    compliance_files = {
        'nist-csf-compliance.txt': 'nist_csf',
        'etsi-nfv-sec-compliance.txt': 'etsi_nfv_sec',
        'oran-security-compliance.txt': 'oran_security'
    }
    
    for compliance_file, compliance_key in compliance_files.items():
        compliance_path = os.path.join(args.input_dir, compliance_file)
        if os.path.exists(compliance_path):
            scan_results['compliance'][compliance_key] = 'Checked'
        else:
            scan_results['compliance'][compliance_key] = 'Not Checked'
    
    # Generate report
    if args.format == 'html':
        generate_html_report(scan_results, args.output)
    else:
        generate_json_report(scan_results, args.output)
    
    print(f"Security report generated: {args.output}")
    print(f"Summary: {scan_results['summary']['critical_issues']} critical, {scan_results['summary']['high_issues']} high, {scan_results['summary']['medium_issues']} medium, {scan_results['summary']['low_issues']} low issues")

if __name__ == '__main__':
    main()
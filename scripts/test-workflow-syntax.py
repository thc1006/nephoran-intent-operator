#!/usr/bin/env python3
"""
GitHub Actions Workflow Syntax Validator
==========================================
Comprehensive validation tool for GitHub Actions YAML workflows with detailed
analysis of syntax, structure, security, and best practices.
"""

import yaml
import json
import os
import sys
from pathlib import Path
from typing import Dict, List, Any, Tuple
import re


class WorkflowValidator:
    def __init__(self, project_root: str):
        self.project_root = Path(project_root)
        self.workflows_dir = self.project_root / ".github" / "workflows"
        self.validation_results = []
        self.errors = []
        self.warnings = []
        
    def log_result(self, test_name: str, status: str, message: str, file_name: str = ""):
        """Log a test result"""
        result = {
            "test": test_name,
            "status": status,  # PASS, FAIL, WARN
            "message": message,
            "file": file_name
        }
        self.validation_results.append(result)
        
        status_icon = {
            "PASS": "✅",
            "FAIL": "❌", 
            "WARN": "⚠️"
        }.get(status, "❓")
        
        print(f"{status_icon} {test_name}: {message}")
        
        if status == "FAIL":
            self.errors.append(result)
        elif status == "WARN":
            self.warnings.append(result)
    
    def validate_yaml_syntax(self, file_path: Path) -> bool:
        """Validate YAML syntax of a workflow file"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                yaml.safe_load(f)
            self.log_result(f"YAML-Syntax-{file_path.name}", "PASS", 
                           "Valid YAML syntax", file_path.name)
            return True
        except yaml.YAMLError as e:
            self.log_result(f"YAML-Syntax-{file_path.name}", "FAIL",
                           f"YAML syntax error: {str(e)}", file_path.name)
            return False
        except Exception as e:
            self.log_result(f"YAML-Syntax-{file_path.name}", "FAIL",
                           f"File read error: {str(e)}", file_path.name)
            return False
    
    def validate_workflow_structure(self, file_path: Path, workflow_data: Dict) -> bool:
        """Validate GitHub Actions workflow structure"""
        required_fields = ["name", "on", "jobs"]
        missing_fields = []
        
        for field in required_fields:
            if field not in workflow_data:
                missing_fields.append(field)
        
        if missing_fields:
            self.log_result(f"Structure-{file_path.name}", "FAIL",
                           f"Missing required fields: {', '.join(missing_fields)}", 
                           file_path.name)
            return False
        else:
            self.log_result(f"Structure-{file_path.name}", "PASS",
                           "All required workflow fields present", file_path.name)
            return True
    
    def validate_security_practices(self, file_path: Path, workflow_data: Dict) -> bool:
        """Validate security best practices in workflow"""
        security_issues = []
        security_warnings = []
        
        # Check for appropriate permissions
        if "permissions" in workflow_data:
            permissions = workflow_data["permissions"]
            
            # Check for overly broad permissions
            if permissions == "write-all" or permissions == "read-all":
                security_issues.append("Overly broad permissions detected")
            
            # Check for specific required permissions for GHCR
            required_ghcr_perms = ["contents", "packages", "id-token"]
            for perm in required_ghcr_perms:
                if perm not in permissions:
                    security_warnings.append(f"Missing {perm} permission for GHCR")
        else:
            security_warnings.append("No explicit permissions defined")
        
        # Check for secrets handling
        workflow_str = str(workflow_data)
        if "secrets." in workflow_str:
            # Look for potential secret misuse
            if "secrets.GITHUB_TOKEN" in workflow_str:
                self.log_result(f"Security-{file_path.name}-Secrets", "PASS",
                               "Proper GITHUB_TOKEN usage detected", file_path.name)
            else:
                security_warnings.append("Custom secrets detected - ensure proper usage")
        
        # Check for hardcoded values that should be secrets
        hardcoded_patterns = [
            r"password:\s*['\"][^'\"]*['\"]",
            r"token:\s*['\"][^'\"]*['\"]",
            r"key:\s*['\"][^'\"]*['\"]"
        ]
        
        for pattern in hardcoded_patterns:
            if re.search(pattern, str(workflow_data), re.IGNORECASE):
                security_issues.append("Potential hardcoded credentials detected")
        
        # Report results
        if security_issues:
            self.log_result(f"Security-{file_path.name}", "FAIL",
                           f"Security issues: {'; '.join(security_issues)}", file_path.name)
            return False
        elif security_warnings:
            self.log_result(f"Security-{file_path.name}", "WARN",
                           f"Security warnings: {'; '.join(security_warnings)}", file_path.name)
        else:
            self.log_result(f"Security-{file_path.name}", "PASS",
                           "Security practices validated", file_path.name)
        
        return True
    
    def validate_ci_best_practices(self, file_path: Path, workflow_data: Dict) -> bool:
        """Validate CI/CD best practices"""
        issues = []
        warnings = []
        
        # Check for concurrency controls
        if "concurrency" not in workflow_data:
            warnings.append("No concurrency control - may have overlapping runs")
        else:
            concurrency = workflow_data["concurrency"]
            if "group" not in concurrency:
                warnings.append("Concurrency group not specified")
            if "cancel-in-progress" not in concurrency:
                warnings.append("Cancel-in-progress not configured")
        
        # Check for timeout configurations
        jobs = workflow_data.get("jobs", {})
        jobs_without_timeouts = []
        
        for job_name, job_config in jobs.items():
            if "timeout-minutes" not in job_config:
                jobs_without_timeouts.append(job_name)
        
        if jobs_without_timeouts:
            warnings.append(f"Jobs without timeouts: {', '.join(jobs_without_timeouts)}")
        
        # Check for caching
        has_caching = False
        for job_name, job_config in jobs.items():
            steps = job_config.get("steps", [])
            for step in steps:
                if isinstance(step, dict) and "uses" in step:
                    if "cache" in step["uses"]:
                        has_caching = True
                        break
        
        if not has_caching:
            warnings.append("No caching detected - builds may be slow")
        
        # Check for conditional job execution
        conditional_jobs = 0
        for job_name, job_config in jobs.items():
            if "if" in job_config or "needs" in job_config:
                conditional_jobs += 1
        
        if conditional_jobs == 0:
            warnings.append("No conditional job execution - may waste resources")
        
        # Report results
        if issues:
            self.log_result(f"BestPractices-{file_path.name}", "FAIL",
                           f"Issues: {'; '.join(issues)}", file_path.name)
            return False
        elif warnings:
            self.log_result(f"BestPractices-{file_path.name}", "WARN",
                           f"Warnings: {'; '.join(warnings)}", file_path.name)
        else:
            self.log_result(f"BestPractices-{file_path.name}", "PASS",
                           "Best practices followed", file_path.name)
        
        return True
    
    def validate_action_versions(self, file_path: Path, workflow_data: Dict) -> bool:
        """Validate GitHub Actions versions are current and secure"""
        recommended_actions = {
            "actions/checkout": "v4",
            "actions/setup-go": "v5",
            "actions/cache": "v4",
            "actions/upload-artifact": "v4",
            "docker/setup-buildx-action": "v3",
            "docker/login-action": "v3",
            "docker/metadata-action": "v5"
        }
        
        outdated_actions = []
        jobs = workflow_data.get("jobs", {})
        
        for job_name, job_config in jobs.items():
            steps = job_config.get("steps", [])
            for step in steps:
                if isinstance(step, dict) and "uses" in step:
                    action_ref = step["uses"]
                    
                    # Extract action name and version
                    if "@" in action_ref:
                        action_name, version = action_ref.rsplit("@", 1)
                        
                        if action_name in recommended_actions:
                            recommended_version = recommended_actions[action_name]
                            if version != recommended_version:
                                outdated_actions.append(f"{action_name}@{version} (recommended: @{recommended_version})")
        
        if outdated_actions:
            self.log_result(f"ActionVersions-{file_path.name}", "WARN",
                           f"Outdated actions: {'; '.join(outdated_actions)}", file_path.name)
        else:
            self.log_result(f"ActionVersions-{file_path.name}", "PASS",
                           "Action versions are current", file_path.name)
        
        return len(outdated_actions) == 0
    
    def validate_single_workflow(self, file_path: Path) -> bool:
        """Validate a single workflow file comprehensively"""
        print(f"\n🔍 Validating workflow: {file_path.name}")
        
        # Test 1: YAML syntax
        if not self.validate_yaml_syntax(file_path):
            return False
        
        # Load workflow data
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                workflow_data = yaml.safe_load(f)
        except Exception as e:
            self.log_result(f"Load-{file_path.name}", "FAIL",
                           f"Failed to load workflow: {str(e)}", file_path.name)
            return False
        
        # Test 2: Workflow structure
        structure_valid = self.validate_workflow_structure(file_path, workflow_data)
        
        # Test 3: Security practices
        security_valid = self.validate_security_practices(file_path, workflow_data)
        
        # Test 4: CI best practices
        practices_valid = self.validate_ci_best_practices(file_path, workflow_data)
        
        # Test 5: Action versions
        versions_valid = self.validate_action_versions(file_path, workflow_data)
        
        return structure_valid and security_valid
    
    def validate_all_workflows(self) -> bool:
        """Validate all workflow files in the project"""
        print("🚀 Starting GitHub Actions Workflow Validation")
        print(f"📁 Workflows directory: {self.workflows_dir}")
        
        if not self.workflows_dir.exists():
            self.log_result("Workflows-Directory", "FAIL", "Workflows directory does not exist")
            return False
        
        # Find all workflow files
        workflow_files = list(self.workflows_dir.glob("*.yml")) + list(self.workflows_dir.glob("*.yaml"))
        
        if not workflow_files:
            self.log_result("Workflows-Files", "FAIL", "No workflow files found")
            return False
        
        print(f"📋 Found {len(workflow_files)} workflow files to validate")
        
        all_valid = True
        for workflow_file in workflow_files:
            if not self.validate_single_workflow(workflow_file):
                all_valid = False
        
        return all_valid
    
    def generate_report(self) -> str:
        """Generate a detailed validation report"""
        total_tests = len(self.validation_results)
        passed_tests = len([r for r in self.validation_results if r["status"] == "PASS"])
        failed_tests = len(self.errors)
        warning_tests = len(self.warnings)
        
        success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
        
        report = [
            "=" * 80,
            "GITHUB ACTIONS WORKFLOW VALIDATION REPORT",
            "=" * 80,
            f"Generated: {os.environ.get('DATE', 'unknown')}",
            "",
            "SUMMARY:",
            f"  Total Tests: {total_tests}",
            f"  Passed: {passed_tests}",
            f"  Failed: {failed_tests}",
            f"  Warnings: {warning_tests}",
            f"  Success Rate: {success_rate:.1f}%",
            ""
        ]
        
        if self.errors:
            report.extend([
                "FAILURES:",
                "----------"
            ])
            for error in self.errors:
                report.append(f"❌ {error['test']}: {error['message']} ({error['file']})")
            report.append("")
        
        if self.warnings:
            report.extend([
                "WARNINGS:",
                "----------"
            ])
            for warning in self.warnings:
                report.append(f"⚠️ {warning['test']}: {warning['message']} ({warning['file']})")
            report.append("")
        
        # Overall status
        if failed_tests == 0:
            if warning_tests == 0:
                report.append("🎉 ALL WORKFLOWS VALIDATED SUCCESSFULLY!")
                report.append("Status: READY FOR PRODUCTION ✅")
            else:
                report.append("✅ WORKFLOWS VALIDATED WITH MINOR WARNINGS")
                report.append("Status: READY FOR PRODUCTION (with recommendations) ⚠️")
        else:
            report.append("❌ WORKFLOW VALIDATION FAILED")
            report.append("Status: REQUIRES FIXES BEFORE PRODUCTION ❌")
        
        return "\n".join(report)
    
    def run_validation(self) -> bool:
        """Run the complete validation suite"""
        try:
            success = self.validate_all_workflows()
            
            # Generate and save report
            report = self.generate_report()
            print("\n" + report)
            
            # Save detailed results
            results_file = self.project_root / "workflow-validation-results.json"
            with open(results_file, 'w') as f:
                json.dump({
                    "summary": {
                        "total_tests": len(self.validation_results),
                        "passed": len([r for r in self.validation_results if r["status"] == "PASS"]),
                        "failed": len(self.errors),
                        "warnings": len(self.warnings),
                        "success_rate": (len([r for r in self.validation_results if r["status"] == "PASS"]) / len(self.validation_results) * 100) if self.validation_results else 0
                    },
                    "results": self.validation_results,
                    "errors": self.errors,
                    "warnings": self.warnings
                }, f, indent=2)
            
            print(f"\n📊 Detailed results saved to: {results_file}")
            return success and len(self.errors) == 0
            
        except Exception as e:
            print(f"❌ Validation failed with exception: {str(e)}")
            return False


def main():
    """Main execution function"""
    if len(sys.argv) > 1:
        project_root = sys.argv[1]
    else:
        # Assume script is in scripts/ directory
        project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    
    print("GitHub Actions Workflow Validator")
    print(f"Project root: {project_root}")
    
    validator = WorkflowValidator(project_root)
    success = validator.run_validation()
    
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
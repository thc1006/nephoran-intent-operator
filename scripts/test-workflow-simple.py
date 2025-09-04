#!/usr/bin/env python3
"""
Simple GitHub Actions Workflow Validator
Windows-compatible version without Unicode characters
"""

import yaml
import json
import os
import sys
from pathlib import Path

def validate_workflow_file(file_path):
    """Validate a single workflow file"""
    results = []
    
    try:
        # Test YAML syntax
        with open(file_path, 'r', encoding='utf-8') as f:
            workflow_data = yaml.safe_load(f)
        
        results.append(("YAML-Syntax", "PASS", "Valid YAML syntax"))
        
        # Test required fields
        required_fields = ["name", "on", "jobs"]
        missing_fields = [field for field in required_fields if field not in workflow_data]
        
        if missing_fields:
            results.append(("Structure", "FAIL", f"Missing fields: {', '.join(missing_fields)}"))
        else:
            results.append(("Structure", "PASS", "All required fields present"))
        
        # Test permissions
        if "permissions" in workflow_data:
            permissions = workflow_data["permissions"]
            required_perms = ["contents", "packages", "security-events", "id-token"]
            
            missing_perms = [perm for perm in required_perms if perm not in permissions]
            if missing_perms:
                results.append(("Permissions", "WARN", f"Missing permissions: {', '.join(missing_perms)}"))
            else:
                results.append(("Permissions", "PASS", "Required permissions present"))
        else:
            results.append(("Permissions", "WARN", "No permissions defined"))
        
        # Test concurrency
        if "concurrency" in workflow_data:
            results.append(("Concurrency", "PASS", "Concurrency control configured"))
        else:
            results.append(("Concurrency", "WARN", "No concurrency control"))
        
        return results
        
    except yaml.YAMLError as e:
        return [("YAML-Syntax", "FAIL", f"YAML error: {str(e)}")]
    except Exception as e:
        return [("File-Access", "FAIL", f"File error: {str(e)}")]

def main():
    """Main function"""
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    workflows_dir = os.path.join(project_root, ".github", "workflows")
    
    print("GitHub Actions Workflow Validator")
    print(f"Project root: {project_root}")
    print(f"Workflows directory: {workflows_dir}")
    print()
    
    if not os.path.exists(workflows_dir):
        print("[FAIL] Workflows directory not found")
        sys.exit(1)
    
    # Find workflow files
    workflow_files = []
    for ext in ["*.yml", "*.yaml"]:
        workflow_files.extend(Path(workflows_dir).glob(ext))
    
    if not workflow_files:
        print("[FAIL] No workflow files found")
        sys.exit(1)
    
    print(f"Found {len(workflow_files)} workflow files")
    print()
    
    # Validate each file
    total_tests = 0
    passed_tests = 0
    failed_tests = 0
    warning_tests = 0
    
    for workflow_file in workflow_files:
        print(f"Validating: {workflow_file.name}")
        results = validate_workflow_file(workflow_file)
        
        for test_name, status, message in results:
            total_tests += 1
            status_symbol = {"PASS": "[PASS]", "FAIL": "[FAIL]", "WARN": "[WARN]"}[status]
            print(f"  {status_symbol} {test_name}: {message}")
            
            if status == "PASS":
                passed_tests += 1
            elif status == "FAIL":
                failed_tests += 1
            elif status == "WARN":
                warning_tests += 1
        
        print()
    
    # Summary
    print("=" * 60)
    print("VALIDATION SUMMARY")
    print("=" * 60)
    print(f"Total tests: {total_tests}")
    print(f"Passed: {passed_tests}")
    print(f"Failed: {failed_tests}")
    print(f"Warnings: {warning_tests}")
    
    if total_tests > 0:
        success_rate = (passed_tests / total_tests) * 100
        print(f"Success rate: {success_rate:.1f}%")
    
    print()
    if failed_tests == 0:
        if warning_tests == 0:
            print("ALL WORKFLOWS VALIDATED SUCCESSFULLY!")
            print("Status: READY FOR PRODUCTION")
        else:
            print("WORKFLOWS VALIDATED WITH WARNINGS")
            print("Status: READY FOR PRODUCTION (address warnings)")
        sys.exit(0)
    else:
        print("WORKFLOW VALIDATION FAILED")
        print("Status: REQUIRES FIXES BEFORE PRODUCTION")
        sys.exit(1)

if __name__ == "__main__":
    main()
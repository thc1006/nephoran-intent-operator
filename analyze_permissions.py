#!/usr/bin/env python3
import os
import yaml
import json
from pathlib import Path

def analyze_workflow_permissions():
    workflows_dir = Path(".github/workflows")
    results = []
    
    for workflow_file in workflows_dir.glob("*.yml"):
        if workflow_file.suffix != ".yml":
            continue
            
        try:
            with open(workflow_file, 'r', encoding='utf-8') as f:
                content = f.read()
                workflow = yaml.safe_load(content)
                
            if not workflow:
                continue
                
            result = {
                "file": workflow_file.name,
                "name": workflow.get("name", "unnamed"),
                "on": list(workflow.get("on", {}).keys()) if isinstance(workflow.get("on"), dict) else workflow.get("on"),
                "permissions": workflow.get("permissions", "NOT DEFINED"),
                "jobs": {}
            }
            
            # Check for job-level permissions
            if "jobs" in workflow:
                for job_name, job_config in workflow.get("jobs", {}).items():
                    if "permissions" in job_config:
                        result["jobs"][job_name] = job_config["permissions"]
                        
            # Check for specific concerns
            concerns = []
            
            # Check if permissions are defined
            if result["permissions"] == "NOT DEFINED":
                concerns.append("NO PERMISSIONS DEFINED - defaults to full write")
                
            # Check for excessive permissions
            elif isinstance(result["permissions"], dict):
                if "write-all" in str(result["permissions"]):
                    concerns.append("WRITE-ALL permission granted")
                if result["permissions"].get("contents") == "write":
                    concerns.append("Has write access to repository contents")
                if result["permissions"].get("packages") == "write":
                    concerns.append("Has write access to packages")
                if result["permissions"].get("administration") == "write":
                    concerns.append("Has ADMIN access to repository")
                if result["permissions"].get("actions") == "write":
                    concerns.append("Can modify GitHub Actions")
                    
            # Check for pull_request events
            if isinstance(result["on"], list) and "pull_request" in result["on"]:
                if result["permissions"] != "NOT DEFINED" and isinstance(result["permissions"], dict):
                    if result["permissions"].get("contents") == "write":
                        concerns.append("PR workflow has write access - potential security risk")
                        
            result["concerns"] = concerns
            results.append(result)
            
        except Exception as e:
            results.append({
                "file": workflow_file.name,
                "error": str(e)
            })
    
    return results

# Run analysis
results = analyze_workflow_permissions()

# Print report
print("=" * 80)
print("GitHub Workflow Permissions Security Audit Report")
print("=" * 80)
print()

# Summary statistics
total_workflows = len(results)
no_permissions = sum(1 for r in results if r.get("permissions") == "NOT DEFINED")
with_concerns = sum(1 for r in results if r.get("concerns"))

print(f"Total workflows analyzed: {total_workflows}")
print(f"Workflows without permissions defined: {no_permissions}")
print(f"Workflows with security concerns: {with_concerns}")
print()

# Detailed analysis
print("=" * 80)
print("DETAILED ANALYSIS")
print("=" * 80)

for result in sorted(results, key=lambda x: len(x.get("concerns", [])), reverse=True):
    if "error" in result:
        print(f"\n[ERROR] {result['file']}: {result['error']}")
        continue
        
    print(f"\n{'='*60}")
    print(f"Workflow: {result['file']}")
    print(f"Name: {result['name']}")
    print(f"Triggers: {result['on']}")
    print(f"Permissions: {json.dumps(result['permissions'], indent=2) if result['permissions'] != 'NOT DEFINED' else 'NOT DEFINED'}")
    
    if result["jobs"]:
        print(f"Job-level permissions:")
        for job, perms in result["jobs"].items():
            print(f"  - {job}: {perms}")
    
    if result["concerns"]:
        print(f"SECURITY CONCERNS:")
        for concern in result["concerns"]:
            print(f"  WARNING: {concern}")
    else:
        print("OK: No major security concerns identified")

print("\n" + "=" * 80)
print("RECOMMENDATIONS")
print("=" * 80)

print("""
1. CRITICAL: Add explicit permissions blocks to ALL workflows
2. Use minimal permissions principle - only grant what's needed
3. For PR workflows, use read-only permissions unless absolutely necessary
4. Avoid 'write-all' or 'administration: write' permissions
5. Consider using GITHUB_TOKEN with restricted scopes
6. Implement job-level permissions for granular control
7. Regularly audit and review workflow permissions
""")
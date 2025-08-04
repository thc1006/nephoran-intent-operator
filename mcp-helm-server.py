#!/usr/bin/env python3
"""
Alternative MCP Helm server for troubleshooting configuration caching issues.
This is identical to helm-local-wrapper.py but with a different name.
"""

import asyncio
import json
import sys
import subprocess
import os
from typing import Any, Dict, List, Optional

class MCPHelmServer:
    def __init__(self):
        # Use the local helm.exe we downloaded
        current_dir = os.path.dirname(os.path.abspath(__file__))
        self.helm_binary = os.path.join(current_dir, "helm.exe")
        self.kubeconfig_path = os.path.expanduser(os.environ.get("KUBECONFIG", "~/.kube/config"))
        
    async def run_helm_command(self, args: List[str]) -> Dict[str, Any]:
        """Run a Helm command using local binary"""
        try:
            cmd = [self.helm_binary] + args
            
            env = os.environ.copy()
            env["KUBECONFIG"] = self.kubeconfig_path
            
            result = subprocess.run(
                cmd, 
                capture_output=True, 
                text=True, 
                timeout=60,
                env=env
            )
            
            return {
                "success": result.returncode == 0,
                "stdout": result.stdout,
                "stderr": result.stderr,
                "returncode": result.returncode
            }
        except subprocess.TimeoutExpired:
            return {
                "success": False,
                "stdout": "",
                "stderr": "Command timed out",
                "returncode": -1
            }
        except Exception as e:
            return {
                "success": False,
                "stdout": "",
                "stderr": str(e),
                "returncode": -1
            }

    async def handle_request(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """Handle MCP requests"""
        method = request.get("method", "")
        params = request.get("params", {})
        
        if method == "tools/list":
            return {
                "jsonrpc": "2.0",
                "id": request.get("id"),
                "result": {
                    "tools": [
                        {
                            "name": "helm_version",
                            "description": "Get Helm version information"
                        },
                        {
                            "name": "helm_list",
                            "description": "List Helm releases",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "namespace": {"type": "string"},
                                    "all_namespaces": {"type": "boolean"}
                                }
                            }
                        },
                        {
                            "name": "helm_install",
                            "description": "Install a Helm chart",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "release_name": {"type": "string"},
                                    "chart": {"type": "string"},
                                    "namespace": {"type": "string"},
                                    "values_file": {"type": "string"}
                                },
                                "required": ["release_name", "chart"]
                            }
                        },
                        {
                            "name": "helm_uninstall",
                            "description": "Uninstall a Helm release",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "release_name": {"type": "string"},
                                    "namespace": {"type": "string"}
                                },
                                "required": ["release_name"]
                            }
                        },
                        {
                            "name": "helm_status",
                            "description": "Get status of a Helm release",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "release_name": {"type": "string"},
                                    "namespace": {"type": "string"}
                                },
                                "required": ["release_name"]
                            }
                        },
                        {
                            "name": "helm_repo_add",
                            "description": "Add a Helm repository",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "repo_name": {"type": "string"},
                                    "repo_url": {"type": "string"}
                                },
                                "required": ["repo_name", "repo_url"]
                            }
                        }
                    ]
                }
            }
        
        elif method == "tools/call":
            tool_name = params.get("name", "")
            arguments = params.get("arguments", {})
            
            if tool_name == "helm_version":
                result = await self.run_helm_command(["version"])
                
            elif tool_name == "helm_list":
                cmd = ["list"]
                if arguments.get("all_namespaces"):
                    cmd.append("--all-namespaces")
                elif arguments.get("namespace"):
                    cmd.extend(["-n", arguments["namespace"]])
                cmd.append("--output=table")
                result = await self.run_helm_command(cmd)
                
            elif tool_name == "helm_install":
                cmd = ["install", arguments["release_name"], arguments["chart"]]
                if arguments.get("namespace"):
                    cmd.extend(["-n", arguments["namespace"], "--create-namespace"])
                if arguments.get("values_file"):
                    cmd.extend(["-f", arguments["values_file"]])
                result = await self.run_helm_command(cmd)
                
            elif tool_name == "helm_uninstall":
                cmd = ["uninstall", arguments["release_name"]]
                if arguments.get("namespace"):
                    cmd.extend(["-n", arguments["namespace"]])
                result = await self.run_helm_command(cmd)
                
            elif tool_name == "helm_status":
                cmd = ["status", arguments["release_name"]]
                if arguments.get("namespace"):
                    cmd.extend(["-n", arguments["namespace"]])
                result = await self.run_helm_command(cmd)
                
            elif tool_name == "helm_repo_add":
                cmd = ["repo", "add", arguments["repo_name"], arguments["repo_url"]]
                result = await self.run_helm_command(cmd)
                
            else:
                result = {
                    "success": False,
                    "stdout": "",
                    "stderr": f"Unknown tool: {tool_name}",
                    "returncode": 1
                }
            
            return {
                "jsonrpc": "2.0",
                "id": request.get("id"),
                "result": {
                    "content": [
                        {
                            "type": "text",
                            "text": f"Command executed: {result['success']}\n"
                                   f"{'Output' if result['success'] else 'Error'}: {result['stdout'] if result['success'] else result['stderr']}"
                        }
                    ]
                }
            }
        
        else:
            return {
                "jsonrpc": "2.0",
                "id": request.get("id"),
                "error": {
                    "code": -32601,
                    "message": f"Method not found: {method}"
                }
            }

async def main():
    """Main MCP server loop"""
    server = MCPHelmServer()
    
    # Simple stdio-based MCP server
    while True:
        try:
            line = input()
            if not line:
                continue
                
            request = json.loads(line)
            response = await server.handle_request(request)
            print(json.dumps(response))
            sys.stdout.flush()
            
        except EOFError:
            break
        except json.JSONDecodeError:
            continue
        except Exception as e:
            error_response = {
                "jsonrpc": "2.0",
                "id": None,
                "error": {
                    "code": -32603,
                    "message": f"Internal error: {str(e)}"
                }
            }
            print(json.dumps(error_response))
            sys.stdout.flush()

if __name__ == "__main__":
    asyncio.run(main())
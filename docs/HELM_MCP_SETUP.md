# Helm MCP Server Setup

## Overview
This directory contains a custom Helm MCP (Model Context Protocol) server implementation that provides Helm functionality when the official Helm MCP server is not accessible.

## Files
- `helm-local-wrapper.py` - Local Helm MCP wrapper using the Helm binary
- `mcp-helm-server.py` - Alternative MCP Helm server (currently active)
- `helm.exe` - Local Helm v3.17.1 binary for Windows
- `.mcp.json` - Updated MCP configuration

## Issue Resolution
**Problem**: The MCP system was caching the old Docker-based configuration (`ghcr.io/zekker6/mcp-helm:v0.0.5`) even after updating `.mcp.json`.

**Solution Applied**:
1. Cleared MCP cache directories
2. Created alternative script `mcp-helm-server.py` 
3. Updated `.mcp.json` to use the new script
4. Both wrapper scripts are functionally identical and provide the same Helm tools

## Features
The Helm MCP wrapper provides the following tools:
- `helm_version` - Get Helm version information
- `helm_list` - List Helm releases (supports namespace filtering)
- `helm_install` - Install Helm charts
- `helm_uninstall` - Uninstall Helm releases
- `helm_status` - Get release status
- `helm_repo_add` - Add Helm repositories

## Configuration
The MCP configuration uses:
- Local Python wrapper script
- Local Helm binary (helm.exe)
- Current kubeconfig context (kind-nephoran-dev)

## Usage
The Helm MCP server is automatically started when you connect to MCP services. You can use Helm commands through the MCP interface.

## Troubleshooting
- Ensure Docker is running for Kind cluster
- Verify kubectl context is set to `kind-nephoran-dev`
- Check that helm.exe is in the project directory
- Ensure KUBECONFIG environment variable is correctly set

## Alternative Solutions
If needed, you can also use:
1. Direct Helm commands: `./helm.exe <command>`
2. Docker-based Helm: `docker run --rm --network=host -v "C:\Users\tingy\.kube:/root/.kube" alpine/helm:latest <command>`
# Conductor Loop

The Conductor Loop is a secure, production-ready file-watching orchestration component of the Nephoran Intent Operator that monitors intent files and triggers Porch package deployments automatically with enterprise-grade security controls.

## Overview

The conductor-loop watches a specified directory for intent files (matching pattern `intent-*.json`) and automatically processes them by calling the Porch CLI tool. It provides:

- **Secure File System Watching**: Real-time monitoring with path traversal protection using fsnotify
- **Debounced Processing**: Prevents duplicate processing with configurable debounce timing
- **Worker Pool**: Concurrent processing with configurable worker count and resource limits
- **State Management**: Tracks processed files with SHA256 integrity checking to prevent reprocessing
- **File Organization**: Automatically organizes processed and failed files with atomic operations
- **Status Reporting**: Generates cryptographically signed status files for each processed intent
- **Graceful Shutdown**: Clean shutdown with proper resource cleanup and data consistency
- **Security Hardening**: Input validation, command injection prevention, and resource exhaustion protection
- **Audit Trail**: Comprehensive logging with correlation IDs for security forensics
- **Production Monitoring**: Health checks, metrics collection, and performance monitoring

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Intent Files  │────│  Conductor Loop  │────│  Porch Package  │
│   (handoff/)    │    │   File Watcher   │    │   Generation    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  File Manager    │
                       │  - processed/    │
                       │  - failed/       │
                       │  - status/       │
                       └──────────────────┘
```

### Component Flow

1. **File Watcher**: Monitors handoff directory for new/modified intent files
2. **Debouncer**: Prevents duplicate processing of rapidly changing files
3. **Worker Pool**: Processes intent files concurrently
4. **Porch Executor**: Executes porch CLI commands for each intent
5. **State Manager**: Tracks file processing state with SHA256 checksums
6. **File Manager**: Organizes processed/failed files and creates error logs
7. **Status Writer**: Generates JSON status files for each processed intent

## Quick Start

### Prerequisites

- Go 1.24+ (for building from source)
- Porch CLI tool installed and accessible in PATH
- Read/write access to handoff directory

### Installation

#### Option 1: Build from Source
```powershell
# Windows PowerShell
cd cmd/conductor-loop
go build -o conductor-loop.exe .
```

```bash
# Linux/macOS
cd cmd/conductor-loop
go build -o conductor-loop .
```

#### Option 2: Use Docker
```bash
docker run --rm -v $(pwd)/handoff:/handoff ghcr.io/thc1006/conductor-loop:latest
```

### Basic Usage

#### Windows PowerShell
```powershell
# Monitor current directory handoff folder
.\conductor-loop.exe

# Custom configuration
.\conductor-loop.exe `
    -handoff "C:\data\intents" `
    -porch "C:\tools\porch.exe" `
    -mode "structured" `
    -out "C:\data\output" `
    -debounce "1s"
```

#### Linux/macOS
```bash
# Monitor current directory handoff folder
./conductor-loop

# Custom configuration
./conductor-loop \
    --handoff /data/intents \
    --porch /usr/local/bin/porch \
    --mode structured \
    --out /data/output \
    --debounce 1s
```

#### One-time Processing
```bash
# Process existing files once and exit
./conductor-loop --once
```

## Configuration Options

| Flag | Default | Description |
|------|---------|-------------|
| `-handoff` | `./handoff` | Directory to watch for intent files |
| `-porch` | `porch` | Path to porch executable |
| `-mode` | `direct` | Processing mode: `direct` or `structured` |
| `-out` | `./out` | Output directory for processed files |
| `-once` | `false` | Process current backlog then exit |
| `-debounce` | `500ms` | Debounce duration for file events |

### Processing Modes

#### Direct Mode
```bash
porch -intent /path/to/intent.json -out /output/dir
```

#### Structured Mode
```bash
porch -intent /path/to/intent.json -out /output/dir -structured
```

## Intent File Format

Intent files must:
- Start with `intent-` prefix
- End with `.json` extension
- Follow the intent schema (see [docs/contracts/intent.schema.json](../../docs/contracts/intent.schema.json))

Example: `intent-scale-up-001.json`
```json
{
  "intent_type": "scaling",
  "target": "cnf-simulator",
  "namespace": "oran-sim",
  "replicas": 3,
  "reason": "High load detected",
  "source": "planner",
  "correlation_id": "scale-2024-001"
}
```

## Directory Structure

```
handoff/
├── intent-*.json           # Input intent files
├── processed/              # Successfully processed files
├── failed/                 # Failed processing attempts
│   └── *.error.log        # Error logs for failed files
├── status/                 # Status files
│   └── intent-*.json.status
└── .conductor-state.json   # Internal state tracking
```

## Status Files

Each processed intent generates a status file:

```json
{
  "intent_file": "intent-scale-up-001.json",
  "status": "success",
  "message": "Processed by worker 0 in 2.5s",
  "timestamp": "2024-08-15T10:30:00Z",
  "processed_by": "conductor-loop",
  "mode": "direct",
  "porch_path": "porch"
}
```

## Troubleshooting

### Common Issues

#### 1. Porch Command Not Found
```
ERROR: Failed to create watcher: porch validation failed: exec: "porch": executable file not found in %PATH%
```

**Solution**: Install Porch CLI or specify full path:
```bash
conductor-loop -porch /path/to/porch
```

#### 2. Permission Denied
```
ERROR: Failed to create handoff directory: mkdir handoff: permission denied
```

**Solution**: Ensure write permissions to the handoff directory or run with appropriate permissions.

#### 3. High Memory Usage
**Symptoms**: Process memory grows over time
**Solution**: Enable cleanup or reduce cleanup interval:
- Files older than 7 days are automatically cleaned up
- State entries are pruned daily
- Check logs for cleanup activity

#### 4. File Processing Stuck
**Symptoms**: Files remain in handoff directory without processing
**Solutions**:
1. Check porch executable permissions and functionality
2. Verify intent file format matches schema
3. Check available disk space in output directory
4. Review logs for specific error messages

### Logging

Enable detailed logging:
```bash
# Set log level for detailed output
export CONDUCTOR_LOG_LEVEL=debug
./conductor-loop
```

Log messages include:
- File watcher events
- Processing worker activity
- Porch command execution details
- State management operations
- Error details and stack traces

### Performance Considerations

#### Memory Usage
- **Baseline**: ~10-20 MB
- **Per tracked file**: ~200 bytes
- **Large backlogs**: Increase available memory

#### CPU Usage
- **Idle**: <1% CPU
- **Processing**: Depends on Porch execution time
- **High throughput**: Consider increasing worker count

#### Disk I/O
- **File watching**: Minimal overhead
- **State persistence**: JSON writes on state changes
- **File organization**: Move operations (fast on same filesystem)

### Scaling Recommendations

#### Small Deployments (< 100 intents/day)
```bash
conductor-loop -debounce 500ms  # Default settings
```

#### Medium Deployments (100-1000 intents/day)
```bash
conductor-loop -debounce 200ms
# Consider increasing max workers in code (default: 2)
```

#### Large Deployments (> 1000 intents/day)
```bash
conductor-loop -debounce 100ms
# Deploy multiple instances with different handoff directories
# Use external load balancer or file distributor
```

## Docker Deployment

### Basic Container
```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY conductor-loop /usr/local/bin/
COPY porch /usr/local/bin/
ENTRYPOINT ["conductor-loop"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  conductor-loop:
    image: ghcr.io/thc1006/conductor-loop:latest
    volumes:
      - ./handoff:/handoff
      - ./output:/output
    environment:
      - CONDUCTOR_LOG_LEVEL=info
    command: [
      "-handoff", "/handoff",
      "-out", "/output",
      "-mode", "structured"
    ]
```

## Kubernetes Deployment

See [deployments/k8s/conductor-loop/](../../deployments/k8s/conductor-loop/) for complete Kubernetes manifests including:

- Deployment with rolling updates
- ConfigMap for configuration
- PersistentVolumeClaims for data persistence
- ServiceAccount with required permissions
- NetworkPolicy for security
- HorizontalPodAutoscaler for scaling
- Monitoring integration

### Quick Deploy
```bash
kubectl apply -k deployments/k8s/conductor-loop/
```

## Health Monitoring

### Health Check Endpoints
The conductor-loop exposes health endpoints when running in Kubernetes mode:

- `/healthz` - Basic health check
- `/readyz` - Readiness check (can accept new files)
- `/livez` - Liveness check (process is responsive)

### Metrics
Prometheus metrics available at `/metrics`:
- `conductor_files_processed_total` - Total files processed
- `conductor_files_failed_total` - Total files failed
- `conductor_processing_duration_seconds` - Processing time histogram
- `conductor_active_workers` - Current active worker count
- `conductor_queue_size` - Current work queue size

## Security Architecture

### Defense-in-Depth Security

The conductor-loop implements multiple layers of security controls:

#### 1. Input Validation & Sanitization
- **Schema Validation**: All intent files validated against strict JSON schema
- **Content Sanitization**: Path traversal protection prevents `../` attacks
- **Size Limits**: Maximum file size enforcement (configurable, default 10MB)
- **Character Encoding**: UTF-8 validation with suspicious character detection
- **Rate Limiting**: Configurable file processing rate limits to prevent DoS

#### 2. File System Security
- **Principle of Least Privilege**: Runs with minimal required permissions (UID > 10000)
- **Chroot Isolation**: Optional chroot jail for enhanced isolation
- **Path Validation**: Absolute path validation prevents directory traversal
- **Atomic Operations**: File moves use atomic operations to prevent TOCTOU attacks
- **Secure Permissions**: Files created with restrictive permissions (0644/0755)

#### 3. Process Security  
- **Non-Root Execution**: Never runs as root user
- **Resource Limits**: Memory, CPU, and file descriptor limits enforced
- **Capability Dropping**: All Linux capabilities dropped in containers
- **Seccomp Profiles**: Restricted system call access via seccomp
- **Read-Only Root FS**: Container root filesystem mounted read-only

#### 4. Command Execution Security
- **Command Sanitization**: Porch path validation prevents command injection
- **Argument Escaping**: All command arguments properly escaped
- **Environment Isolation**: Clean environment variables for child processes
- **Timeout Protection**: Configurable execution timeouts prevent hangs
- **Output Sanitization**: Command output sanitized before logging

#### 5. Network Security
- **No Network Exposure**: Default mode exposes no network ports
- **TLS-Only Communication**: When enabled, only TLS 1.3+ connections allowed
- **Network Policies**: Kubernetes NetworkPolicy integration for traffic control
- **Egress Filtering**: Optional egress traffic restrictions

#### 6. Data Protection
- **Encryption at Rest**: State files encrypted with AES-256-GCM
- **Integrity Checking**: SHA256 checksums for all processed files
- **Secure State Management**: State persistence with atomic writes
- **Data Sanitization**: Sensitive data scrubbed from logs and status files
- **Backup Encryption**: Automated backups encrypted before storage

### Compliance & Standards

- **O-RAN Security**: Compliant with O-RAN Alliance WG11 security specifications
- **Container Security**: CIS Kubernetes Benchmark compliance
- **NIST Cybersecurity Framework**: Implements Identify, Protect, Detect, Respond, Recover
- **OWASP**: Follows OWASP secure coding practices
- **Supply Chain Security**: SLSA Level 3 compliance for builds

### Security Monitoring

#### Real-Time Detection
- **Anomaly Detection**: Statistical analysis of processing patterns
- **Threat Detection**: Suspicious file patterns and access attempts
- **Performance Monitoring**: Resource exhaustion attack detection
- **Integrity Monitoring**: File system integrity checks

#### Audit & Compliance
- **Comprehensive Logging**: All operations logged with correlation IDs
- **Tamper-Evident Logs**: Cryptographically signed log entries
- **Compliance Reporting**: Automated compliance status reports
- **Forensic Analysis**: Detailed audit trails for incident investigation

### Security Configuration

#### Environment Variables
```bash
# Security settings
export CONDUCTOR_SECURITY_ENABLED="true"
export CONDUCTOR_MAX_FILE_SIZE="10485760"  # 10MB
export CONDUCTOR_CHROOT_ENABLED="false"
export CONDUCTOR_ENCRYPTION_ENABLED="true"
export CONDUCTOR_AUDIT_ENABLED="true"
```

#### Kubernetes Security Context
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  runAsGroup: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

For comprehensive security documentation, see [docs/conductor-loop/SECURITY.md](../../docs/conductor-loop/SECURITY.md).

## Integration Examples

### CI/CD Pipeline
```yaml
# GitHub Actions example
- name: Deploy Intent
  run: |
    echo '${{ secrets.SCALING_INTENT }}' > handoff/intent-ci-deploy.json
    # conductor-loop will automatically process the file
    
- name: Wait for Processing
  run: |
    timeout 60 bash -c 'until [ -f handoff/status/intent-ci-deploy.json.status ]; do sleep 1; done'
    cat handoff/status/intent-ci-deploy.json.status
```

### Monitoring Integration
```bash
# Prometheus scrape config
scrape_configs:
  - job_name: 'conductor-loop'
    static_configs:
      - targets: ['conductor-loop:9090']
```

### Log Aggregation
```yaml
# Fluentd config for log collection
<source>
  @type tail
  path /var/log/conductor-loop.log
  pos_file /var/log/fluentd-conductor-loop.log.pos
  tag conductor-loop
  format json
</source>
```

## Migration Guide

### From Manual Porch Execution
1. Create handoff directory structure
2. Update scripts to write intent files instead of calling porch directly
3. Monitor status files for completion
4. Update error handling to check failed directory

### Configuration Migration
Version 1.0 introduces new configuration options. Update your deployment scripts:

```bash
# Old approach (manual)
porch -intent intent.json -out ./packages

# New approach (automatic)
cp intent.json handoff/
# conductor-loop processes automatically
```

## Contributing

For development setup and contribution guidelines, see [DEVELOPER.md](../../docs/conductor-loop/DEVELOPER.md).

## License

This project is part of the Nephoran Intent Operator and follows the same licensing terms.
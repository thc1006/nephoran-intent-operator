# Conductor Loop - API Reference

This document provides detailed specifications for the conductor-loop APIs, file formats, and integration interfaces.

## Table of Contents

- [Intent File Format](#intent-file-format)
- [Status File Format](#status-file-format)
- [Porch Integration Interface](#porch-integration-interface)
- [Health Check Endpoints](#health-check-endpoints)
- [Metrics Endpoints](#metrics-endpoints)
- [Configuration API](#configuration-api)
- [Error Codes](#error-codes)

## Intent File Format

### Overview

Intent files are JSON documents that describe network operations to be performed. They must follow specific naming conventions and schema validation.

### Naming Convention

```
intent-{identifier}.json
```

**Pattern**: `^intent-[a-zA-Z0-9\-_]+\.json$`

**Examples**:
- `intent-scale-up-001.json`
- `intent-deployment-cnf-sim.json`
- `intent-emergency-scale-down.json`

### Schema Specification

The intent schema is defined in `docs/contracts/intent.schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://nephoran.com/schemas/intent.schema.json",
  "title": "ScalingIntent",
  "type": "object",
  "additionalProperties": false,
  "required": ["intent_type", "target", "namespace", "replicas"],
  "properties": {
    "intent_type": {
      "const": "scaling",
      "description": "Type of intent operation (MVP supports only scaling)"
    },
    "target": {
      "type": "string",
      "minLength": 1,
      "maxLength": 253,
      "pattern": "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
      "description": "Kubernetes Deployment name (CNF simulator)"
    },
    "namespace": {
      "type": "string",
      "minLength": 1,
      "maxLength": 63,
      "pattern": "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
      "description": "Kubernetes namespace"
    },
    "replicas": {
      "type": "integer",
      "minimum": 0,
      "maximum": 100,
      "description": "Desired number of replicas"
    },
    "reason": {
      "type": "string",
      "maxLength": 512,
      "description": "Human-readable reason for the scaling operation"
    },
    "source": {
      "type": "string",
      "enum": ["user", "planner", "test", "automation"],
      "description": "Source of the intent request"
    },
    "correlation_id": {
      "type": "string",
      "maxLength": 64,
      "pattern": "^[a-zA-Z0-9\-_]+$",
      "description": "Optional correlation ID for request tracing"
    },
    "priority": {
      "type": "string",
      "enum": ["low", "normal", "high", "critical"],
      "default": "normal",
      "description": "Processing priority"
    },
    "timeout": {
      "type": "string",
      "pattern": "^[0-9]+[smh]$",
      "default": "30s",
      "description": "Maximum processing timeout (e.g., '30s', '5m', '1h')"
    },
    "metadata": {
      "type": "object",
      "additionalProperties": true,
      "description": "Additional metadata for the intent"
    }
  }
}
```

### Intent Examples

#### Basic Scaling Intent
```json
{
  "intent_type": "scaling",
  "target": "cnf-simulator",
  "namespace": "oran-sim",
  "replicas": 3,
  "reason": "Scheduled scale-up for peak hours",
  "source": "planner",
  "correlation_id": "daily-scale-001"
}
```

#### Emergency Scaling Intent
```json
{
  "intent_type": "scaling",
  "target": "critical-service",
  "namespace": "production",
  "replicas": 10,
  "reason": "Emergency response to high load",
  "source": "automation",
  "priority": "critical",
  "timeout": "10s",
  "correlation_id": "emergency-2024-08-15-001",
  "metadata": {
    "alert_id": "ALT-12345",
    "incident_id": "INC-67890",
    "automated_response": true
  }
}
```

#### Scale-Down Intent
```json
{
  "intent_type": "scaling",
  "target": "test-workload",
  "namespace": "testing",
  "replicas": 0,
  "reason": "Test environment cleanup",
  "source": "user",
  "priority": "low",
  "correlation_id": "cleanup-test-env"
}
```

### Validation Rules

#### Schema Validation
```bash
# Validate intent file against schema
ajv validate -s docs/contracts/intent.schema.json -d intent-example.json

# Enhanced security validation
conductor-loop-validator --security-mode=strict --schema=intent.schema.json --file=intent-example.json
```

#### Security Validation

**Input Sanitization Rules**:
```yaml
security_validation:
  path_traversal:
    enabled: true
    blocked_patterns:
      - "../"
      - "..\\"
      - "%2e%2e"
      - "..%2f"
      - "..%5c"
    max_depth: 3
    
  command_injection:
    enabled: true
    blocked_characters: [";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]"]
    blocked_keywords: ["rm", "del", "format", "curl", "wget", "nc", "ncat"]
    
  size_limits:
    max_file_size: "10MB"
    max_field_length: 1024
    max_array_length: 100
    max_nesting_depth: 10
    
  content_validation:
    require_utf8: true
    block_null_bytes: true
    block_control_chars: true
    normalize_unicode: true
```

**Advanced Validation Engine**:
```go
type SecurityValidator struct {
    schema          *Schema
    pathValidator   *PathValidator
    contentScanner  *ContentScanner
    threatDetector  *ThreatDetector
}

func (sv *SecurityValidator) ValidateIntent(data []byte) (*ValidationResult, error) {
    result := &ValidationResult{
        Valid:           false,
        SecurityChecks:  make(map[string]bool),
        Violations:      []string{},
        ThreatLevel:     "NONE",
    }
    
    // 1. Basic format validation
    if err := sv.validateFormat(data, result); err != nil {
        return result, err
    }
    
    // 2. Schema compliance
    if err := sv.validateSchema(data, result); err != nil {
        return result, err
    }
    
    // 3. Security-specific validation
    if err := sv.validateSecurity(data, result); err != nil {
        return result, err
    }
    
    // 4. Threat detection
    if err := sv.detectThreats(data, result); err != nil {
        return result, err
    }
    
    result.Valid = len(result.Violations) == 0
    return result, nil
}
```

#### Business Logic Validation

1. **Replica Limits**: 
   - Minimum: 0 (scale to zero allowed)
   - Maximum: 100 (configurable per environment, strict limit: 1000)
   - **Security**: Prevents resource exhaustion attacks

2. **Namespace Restrictions**:
   - Cannot target system namespaces (kube-system, kube-public, etc.)
   - Must exist in cluster and be accessible
   - **Security**: Prevents privilege escalation to system namespaces
   - Namespace name validated against Kubernetes naming rules

3. **Target Validation**:
   - Must be a valid Kubernetes Deployment name (DNS-1123 compliant)
   - Deployment must exist in specified namespace
   - **Security**: Prevents targeting of unauthorized resources
   - Name validated against injection patterns

4. **Priority Handling**:
   - `critical`: Processed immediately, bypass queue (requires authorization)
   - `high`: Priority queue processing (rate limited)
   - `normal`: Standard queue processing
   - `low`: Background processing (lowest priority)
   - **Security**: Priority levels audited and require appropriate permissions

5. **Content Security Validation**:
   - All string fields validated for malicious content
   - JSON structure depth limited to prevent parser attacks
   - Field length limits enforced to prevent buffer overflow
   - Unicode normalization applied to prevent encoding attacks

## Status File Format

### Overview

Status files are generated after each intent processing attempt. They provide detailed information about the processing result and are stored in the `status/` subdirectory.

### File Naming

```
{intent-filename}.status
```

**Example**: `intent-scale-up-001.json.status`

### Status Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "IntentProcessingStatus",
  "type": "object",
  "required": ["intent_file", "status", "timestamp", "processed_by"],
  "properties": {
    "intent_file": {
      "type": "string",
      "description": "Original intent filename"
    },
    "status": {
      "type": "string",
      "enum": ["success", "failed", "timeout", "skipped"],
      "description": "Processing result status"
    },
    "message": {
      "type": "string",
      "description": "Human-readable status message"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "Processing completion timestamp (ISO 8601)"
    },
    "processed_by": {
      "type": "string",
      "description": "Component that processed the intent"
    },
    "worker_id": {
      "type": "integer",
      "description": "ID of the worker that processed the intent"
    },
    "processing_duration": {
      "type": "string",
      "pattern": "^[0-9]+\\.?[0-9]*[a-z]+$",
      "description": "Time taken to process the intent"
    },
    "mode": {
      "type": "string",
      "enum": ["direct", "structured"],
      "description": "Processing mode used"
    },
    "porch_path": {
      "type": "string",
      "description": "Path to porch executable used"
    },
    "correlation_id": {
      "type": "string",
      "description": "Correlation ID from the intent"
    },
    "error_details": {
      "type": "object",
      "properties": {
        "error_code": {
          "type": "string",
          "description": "Machine-readable error code"
        },
        "error_message": {
          "type": "string",
          "description": "Detailed error message"
        },
        "porch_exit_code": {
          "type": "integer",
          "description": "Exit code from porch execution"
        },
        "porch_stdout": {
          "type": "string",
          "description": "Standard output from porch command"
        },
        "porch_stderr": {
          "type": "string",
          "description": "Standard error from porch command"
        }
      }
    },
    "retry_count": {
      "type": "integer",
      "description": "Number of retry attempts"
    },
    "next_retry": {
      "type": "string",
      "format": "date-time",
      "description": "Timestamp for next retry attempt"
    }
  }
}
```

### Status Examples

#### Successful Processing
```json
{
  "intent_file": "intent-scale-up-001.json",
  "status": "success",
  "message": "Successfully scaled cnf-simulator to 3 replicas",
  "timestamp": "2024-08-15T10:30:45Z",
  "processed_by": "conductor-loop",
  "worker_id": 2,
  "processing_duration": "2.45s",
  "mode": "structured",
  "porch_path": "/usr/local/bin/porch",
  "correlation_id": "daily-scale-001"
}
```

#### Failed Processing
```json
{
  "intent_file": "intent-invalid-target.json",
  "status": "failed",
  "message": "Failed to process intent: deployment not found",
  "timestamp": "2024-08-15T10:35:12Z",
  "processed_by": "conductor-loop",
  "worker_id": 1,
  "processing_duration": "1.2s",
  "mode": "direct",
  "porch_path": "/usr/local/bin/porch",
  "correlation_id": "test-001",
  "error_details": {
    "error_code": "DEPLOYMENT_NOT_FOUND",
    "error_message": "Deployment 'invalid-target' not found in namespace 'test'",
    "porch_exit_code": 1,
    "porch_stdout": "",
    "porch_stderr": "Error: deployment.apps \"invalid-target\" not found"
  },
  "retry_count": 0,
  "next_retry": "2024-08-15T10:40:12Z"
}
```

#### Timeout Processing
```json
{
  "intent_file": "intent-long-running.json",
  "status": "timeout",
  "message": "Processing timed out after 30s",
  "timestamp": "2024-08-15T10:31:00Z",
  "processed_by": "conductor-loop",
  "worker_id": 0,
  "processing_duration": "30s",
  "mode": "structured",
  "porch_path": "/usr/local/bin/porch",
  "correlation_id": "long-task-001",
  "error_details": {
    "error_code": "PROCESSING_TIMEOUT",
    "error_message": "Porch command timed out after 30s",
    "porch_exit_code": -1
  },
  "retry_count": 1,
  "next_retry": "2024-08-15T10:33:00Z"
}
```

## Porch Integration Interface

### Command Execution

The conductor-loop integrates with Porch via command-line interface execution.

#### Direct Mode
```bash
porch -intent /path/to/intent.json -out /output/directory
```

#### Structured Mode
```bash
porch -intent /path/to/intent.json -out /output/directory -structured
```

### Expected Porch Behavior

#### Success Response
- **Exit Code**: 0
- **Stdout**: Success message or package information
- **Stderr**: Empty or informational messages only

#### Failure Response
- **Exit Code**: Non-zero (1-255)
- **Stdout**: May contain partial output
- **Stderr**: Error description

### Porch Output Format

#### Direct Mode Output
```
Package successfully created: cnf-simulator-v1.2.3
Deployment target: oran-sim/cnf-simulator
Replicas updated: 1 -> 3
```

#### Structured Mode Output
```json
{
  "status": "success",
  "package_name": "cnf-simulator-v1.2.3",
  "deployment": {
    "namespace": "oran-sim",
    "name": "cnf-simulator",
    "previous_replicas": 1,
    "new_replicas": 3
  },
  "timestamp": "2024-08-15T10:30:45Z",
  "duration": "2.45s"
}
```

### Porch Error Codes

| Exit Code | Description | Action |
|-----------|-------------|--------|
| 0 | Success | Continue processing |
| 1 | Invalid intent format | Move to failed, no retry |
| 2 | Deployment not found | Move to failed, no retry |
| 3 | Insufficient permissions | Retry after delay |
| 4 | Network connectivity issue | Retry after delay |
| 5 | Resource quota exceeded | Retry after delay |
| 124 | Command timeout | Retry with longer timeout |
| 125 | Porch binary not found | Alert operations |
| 126 | Permission denied | Alert operations |
| 127 | Command not found | Alert operations |

## Health Check Endpoints

### Security Considerations for Health Endpoints

**Authentication**: Health endpoints support optional authentication via:
- Bearer tokens for CI/CD integration
- mTLS certificates for service mesh environments  
- API keys for monitoring systems

**Rate Limiting**: Health endpoints are rate-limited to prevent abuse:
- 100 requests per minute per IP
- 1000 requests per hour per authenticated client
- Burst allowance of 10 requests

**Information Disclosure Protection**: Health responses exclude sensitive information:
- No internal file paths or configuration details
- No database connection strings or credentials
- No detailed error messages that could aid attackers

### Overview

Health check endpoints are available when conductor-loop runs in Kubernetes mode or with the `--http-server` flag. All endpoints implement security controls including authentication, rate limiting, and sanitized responses.

### Endpoints

#### /healthz
**Purpose**: Basic health check with security validation

**Method**: GET

**Headers**:
```
Authorization: Bearer <token>  (optional)
X-API-Key: <api-key>          (optional)
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-08-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "security": {
    "encryption_status": "enabled",
    "validation_status": "active",
    "threat_level": "normal"
  }
}
```

**Status Codes**:
- `200`: Service is healthy and secure
- `401`: Authentication required
- `403`: Access denied
- `429`: Rate limit exceeded
- `503`: Service is unhealthy

#### /readyz
**Purpose**: Readiness check (can accept new work)

**Method**: GET

**Response**:
```json
{
  "status": "ready",
  "timestamp": "2024-08-15T10:30:00Z",
  "checks": {
    "filesystem_writable": "ok",
    "porch_available": "ok",
    "worker_pool": "ok",
    "queue_capacity": "ok"
  },
  "queue_size": 2,
  "active_workers": 1,
  "max_workers": 4
}
```

**Status Codes**:
- `200`: Service is ready
- `503`: Service is not ready

#### /livez
**Purpose**: Liveness check (process is responsive)

**Method**: GET

**Response**:
```json
{
  "status": "live",
  "timestamp": "2024-08-15T10:30:00Z",
  "pid": 12345,
  "memory_usage": "45.2MB",
  "goroutines": 15
}
```

**Status Codes**:
- `200`: Service is live
- `503`: Service is not responsive

### Health Check Configuration

```yaml
# Kubernetes probe configuration
livenessProbe:
  httpGet:
    path: /livez
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
  timeoutSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 5
  failureThreshold: 12
```

## Metrics Endpoints

### /metrics

**Purpose**: Prometheus metrics endpoint

**Method**: GET

**Content-Type**: `text/plain; version=0.0.4; charset=utf-8`

### Available Metrics

#### File Processing Metrics
```prometheus
# Total files processed by status
conductor_files_processed_total{status="success"} 150
conductor_files_processed_total{status="failed"} 5

# Processing duration histogram
conductor_files_processing_duration_seconds_bucket{le="1"} 45
conductor_files_processing_duration_seconds_bucket{le="5"} 140
conductor_files_processing_duration_seconds_bucket{le="10"} 150
conductor_files_processing_duration_seconds_bucket{le="+Inf"} 155

# Current queue size
conductor_queue_size 3

# Active workers
conductor_active_workers 2

# Maximum workers configured
conductor_max_workers 4
```

#### State Management Metrics
```prometheus
# State entries total
conductor_state_entries_total 1250

# State file operations
conductor_state_operations_total{operation="load"} 1
conductor_state_operations_total{operation="save"} 155
conductor_state_operations_total{operation="cleanup"} 7

# State file size in bytes
conductor_state_file_size_bytes 245760
```

#### Porch Integration Metrics
```prometheus
# Porch execution duration
conductor_porch_execution_duration_seconds_bucket{le="1"} 80
conductor_porch_execution_duration_seconds_bucket{le="5"} 145
conductor_porch_execution_duration_seconds_bucket{le="10"} 150
conductor_porch_execution_duration_seconds_bucket{le="+Inf"} 155

# Porch timeouts
conductor_porch_timeout_total 2

# Porch execution by mode
conductor_porch_executions_total{mode="direct"} 75
conductor_porch_executions_total{mode="structured"} 80
```

#### File Manager Metrics
```prometheus
# File operations
conductor_file_operations_total{operation="move_processed"} 150
conductor_file_operations_total{operation="move_failed"} 5
conductor_file_operations_total{operation="cleanup"} 50

# Directory sizes
conductor_directory_files{directory="handoff"} 3
conductor_directory_files{directory="processed"} 150
conductor_directory_files{directory="failed"} 5
```

#### System Metrics
```prometheus
# Standard Go metrics
go_memstats_heap_inuse_bytes 8388608
go_memstats_stack_inuse_bytes 655360
go_goroutines 15

# Process metrics
process_cpu_seconds_total 125.67
process_memory_bytes 47185920
process_open_fds 12
process_start_time_seconds 1692087000
```

### Metric Labels

#### Common Labels
- `instance`: Instance identifier (hostname or pod name)
- `job`: Prometheus job name
- `version`: Application version

#### Status Labels
- `status`: `success`, `failed`, `timeout`, `skipped`
- `mode`: `direct`, `structured`
- `operation`: Various operation types
- `directory`: Directory name for file metrics

### Custom Metrics Configuration

```yaml
# metrics.yaml
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
  custom_labels:
    environment: "production"
    cluster: "main"
    region: "us-west-2"
  
  # Enable/disable metric groups
  file_processing: true
  state_management: true
  porch_integration: true
  file_manager: true
  system_metrics: true
```

## Configuration API

### Secure Configuration Management

#### Configuration Security Principles

**Configuration as Code**: All configuration should be version-controlled and audited
**Least Privilege**: Configure only required permissions and capabilities
**Defense in Depth**: Multiple security layers in configuration
**Zero Trust**: Validate all configuration inputs and defaults

#### Secure Configuration File Format

The conductor-loop can be configured via JSON configuration file specified with `--config` flag. **Never include secrets directly in configuration files.**

```json
{
  "handoff_dir": "/data/handoff",
  "porch_path": "/usr/local/bin/porch",
  "mode": "structured",
  "out_dir": "/data/output",
  "debounce_duration": "500ms",
  "max_workers": 4,
  "cleanup_after": "168h",
  "log_level": "info",
  "log_format": "json",
  "security": {
    "enabled": true,
    "encryption": {
      "state_encryption": true,
      "log_encryption": true,
      "algorithm": "AES-256-GCM"
    },
    "validation": {
      "strict_mode": true,
      "max_file_size": "10MB",
      "allowed_extensions": [".json"],
      "path_validation": true,
      "content_scanning": true
    },
    "access_control": {
      "rbac_enabled": true,
      "audit_enabled": true,
      "rate_limiting": true
    },
    "hardening": {
      "disable_debug_endpoints": true,
      "secure_headers": true,
      "tls_min_version": "1.3"
    }
  },
  "metrics": {
    "enabled": true,
    "port": 9090,
    "path": "/metrics",
    "authentication": {
      "enabled": true,
      "method": "bearer_token"
    },
    "tls": {
      "enabled": true,
      "cert_file": "/etc/tls/server.crt",
      "key_file": "/etc/tls/server.key"
    }
  },
  "health": {
    "enabled": true,
    "port": 8080,
    "paths": {
      "health": "/healthz",
      "ready": "/readyz",
      "live": "/livez"
    },
    "authentication": {
      "enabled": false,
      "monitoring_exemption": true
    },
    "rate_limiting": {
      "enabled": true,
      "requests_per_minute": 100
    }
  },
  "processing": {
    "timeout": "30s",
    "retry_attempts": 3,
    "retry_delay": "30s",
    "priority_queues": true,
    "resource_limits": {
      "max_memory": "512MB",
      "max_cpu": "1000m",
      "max_file_descriptors": 1024
    }
  },
  "porch": {
    "validation_timeout": "5s",
    "execution_timeout": "30s",
    "security": {
      "command_validation": true,
      "argument_sanitization": true,
      "environment_isolation": true
    },
    "environment": {
      "PORCH_LOG_LEVEL": "info",
      "PORCH_SECURITY_MODE": "strict"
    }
  },
  "audit": {
    "enabled": true,
    "log_level": "detailed",
    "retention_days": 90,
    "tamper_protection": true,
    "destinations": [
      {
        "type": "file",
        "path": "/var/log/conductor-loop/audit.log",
        "encryption": true
      },
      {
        "type": "syslog",
        "endpoint": "syslog.security.company.com:514",
        "protocol": "tls"
      }
    ]
  }
}
```

#### Secret Management Integration

**Kubernetes Secrets**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: conductor-loop-secrets
type: Opaque
stringData:
  api-key: "secure-api-key-here"
  tls-cert: |
    -----BEGIN CERTIFICATE-----
    [certificate content]
    -----END CERTIFICATE-----
  tls-key: |
    -----BEGIN PRIVATE KEY-----
    [private key content]
    -----END PRIVATE KEY-----
```

**HashiCorp Vault Integration**:
```json
{
  "secrets": {
    "vault": {
      "enabled": true,
      "address": "https://vault.company.com",
      "auth_method": "kubernetes",
      "role": "conductor-loop",
      "secrets_path": "secret/conductor-loop"
    }
  }
}
```

**External Secrets Operator**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: conductor-loop-vault
spec:
  provider:
    vault:
      server: "https://vault.company.com"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "conductor-loop"
```

### Environment Variable Override

Configuration values can be overridden using environment variables:

```bash
# Override configuration with environment variables
export CONDUCTOR_HANDOFF_DIR="/custom/handoff"
export CONDUCTOR_MAX_WORKERS="8"
export CONDUCTOR_LOG_LEVEL="debug"
export CONDUCTOR_METRICS_ENABLED="true"
export CONDUCTOR_METRICS_PORT="9090"
```

**Naming Convention**: `CONDUCTOR_<CONFIG_KEY>=<value>`

Nested configuration keys use underscores:
- `metrics.enabled` → `CONDUCTOR_METRICS_ENABLED`
- `porch.execution_timeout` → `CONDUCTOR_PORCH_EXECUTION_TIMEOUT`

### Dynamic Configuration

#### Configuration Reload

Send `SIGHUP` to reload configuration without restarting:

```bash
# Reload configuration
kill -HUP $(pgrep conductor-loop)

# Kubernetes
kubectl exec -n conductor-loop conductor-loop-0 -- kill -HUP 1
```

#### Configuration Validation

```bash
# Validate configuration file
conductor-loop --config=config.json --validate-config

# Output
Configuration validation: PASSED
- Handoff directory: /data/handoff (writable)
- Porch path: /usr/local/bin/porch (executable)
- Output directory: /data/output (writable)
- Log level: info (valid)
- Workers: 4 (within limits)
```

## Error Codes

### Application Error Codes

| Code | Description | Category | Action |
|------|-------------|----------|--------|
| `E001` | Invalid intent file format | Validation | Move to failed |
| `E002` | Intent file too large | Validation | Move to failed |
| `E003` | Schema validation failed | Validation | Move to failed |
| `E004` | Duplicate correlation ID | Business Logic | Move to failed |
| `E005` | Handoff directory not writable | Configuration | Alert operations |
| `E006` | Porch binary not found | Configuration | Alert operations |
| `E007` | Porch execution timeout | Processing | Retry |
| `E008` | Porch permission denied | Configuration | Alert operations |
| `E009` | State file corruption | State Management | Backup and reset |
| `E010` | Worker pool exhausted | Resource | Scale up or throttle |
| `E011` | Disk space insufficient | Resource | Cleanup or expand |
| `E012` | Memory limit exceeded | Resource | Restart or scale |

### HTTP Error Responses

#### 400 Bad Request
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid intent file format",
    "details": {
      "validation_errors": [
        "Field 'intent_type' is required",
        "Field 'replicas' must be a positive integer"
      ]
    },
    "timestamp": "2024-08-15T10:30:00Z"
  }
}
```

#### 503 Service Unavailable
```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Worker pool is at capacity",
    "details": {
      "queue_size": 100,
      "max_queue_size": 100,
      "active_workers": 4,
      "max_workers": 4
    },
    "timestamp": "2024-08-15T10:30:00Z",
    "retry_after": "30s"
  }
}
```

#### 500 Internal Server Error
```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "State file corruption detected",
    "details": {
      "error_id": "ERR-2024-08-15-001",
      "backup_created": "/backup/state-backup-1692087000.json"
    },
    "timestamp": "2024-08-15T10:30:00Z"
  }
}
```

### Error Recovery

#### Automatic Recovery
1. **Transient Errors**: Retry with exponential backoff
2. **State Corruption**: Automatic backup and reset
3. **Resource Exhaustion**: Queue throttling
4. **Porch Timeout**: Increase timeout and retry

#### Manual Recovery
1. **Configuration Errors**: Require operator intervention
2. **Permission Issues**: Fix permissions and restart
3. **Disk Full**: Clean up space and restart
4. **Binary Missing**: Install dependencies and restart

### Monitoring Integration

#### Alert Rules for Error Codes

```yaml
# Prometheus alert rules
groups:
- name: conductor-loop-errors
  rules:
  - alert: ConductorLoopValidationErrors
    expr: increase(conductor_errors_total{code=~"E00[1-4]"}[5m]) > 5
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High rate of validation errors"
      
  - alert: ConductorLoopConfigurationErrors
    expr: increase(conductor_errors_total{code=~"E00[5-8]"}[5m]) > 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Configuration error detected"
      
  - alert: ConductorLoopResourceErrors
    expr: increase(conductor_errors_total{code=~"E01[0-2]"}[5m]) > 0
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Resource constraint detected"
```

This API reference provides comprehensive documentation for all interfaces, file formats, and integration points of the conductor-loop component.
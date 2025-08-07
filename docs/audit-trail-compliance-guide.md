# Comprehensive Audit Trail & Compliance Guide

## Executive Summary

The Nephoran Intent Operator includes a comprehensive audit trail and compliance system that provides enterprise-grade security monitoring, regulatory compliance, and forensic capabilities. This system captures, protects, and analyzes all security-relevant events across the platform while maintaining high performance and reliability.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Audit Event Types](#audit-event-types)
4. [Backend Integrations](#backend-integrations)
5. [Compliance Frameworks](#compliance-frameworks)
6. [Log Integrity & Tamper Protection](#log-integrity--tamper-protection)
7. [Retention Policies](#retention-policies)
8. [Query & Analysis](#query--analysis)
9. [Configuration](#configuration)
10. [Monitoring & Alerting](#monitoring--alerting)
11. [Best Practices](#best-practices)
12. [Troubleshooting](#troubleshooting)

## Overview

### Key Features

- **Comprehensive Event Capture**: All authentication, authorization, data access, system changes, and security events
- **Multi-Backend Support**: Elasticsearch, Splunk, SIEM systems, file-based, cloud providers
- **Compliance Ready**: SOC2, ISO 27001, PCI DSS, HIPAA, GDPR compliance frameworks
- **Tamper Protection**: Cryptographic integrity protection with hash chains and digital signatures
- **Advanced Analytics**: Security threat detection, anomaly analysis, and compliance reporting
- **High Performance**: Sub-2ms latency, 1000+ events/sec throughput, minimal resource overhead
- **Enterprise Security**: End-to-end encryption, PKI integration, secure transport

### Audit Trail Scope

The audit system captures events across all system components:

```
┌─────────────────────────────────────────────────────────────┐
│                    Audit Event Sources                      │
├─────────────────────────────────────────────────────────────┤
│ • API Gateway & Controllers    • Authentication & OAuth2    │
│ • Kubernetes Admission Hooks   • Network Intent Processing  │
│ • O-RAN Interface Operations   • LLM & RAG Interactions     │
│ • Configuration Changes        • Security Events            │
│ • Data Access & Modifications  • System Administration      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Audit Processing Engine                   │
├─────────────────────────────────────────────────────────────┤
│ • Event Enrichment          • Integrity Protection          │
│ • Compliance Mapping        • Performance Optimization      │
│ • Security Analysis         • Query & Analytics             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Storage Backends                         │
├─────────────────────────────────────────────────────────────┤
│ • Elasticsearch/OpenSearch  • Splunk Enterprise             │
│ • SIEM Systems (ArcSight)   • Cloud Audit Services          │
│ • File-based Archives       • Compliance Databases          │
└─────────────────────────────────────────────────────────────┘
```

## Architecture

### Core Components

#### 1. Audit System Core (`pkg/audit/audit_system.go`)

The central audit engine that orchestrates all audit operations:

```go
type AuditSystem struct {
    config          *AuditConfig
    backends        map[string]Backend
    eventChannel    chan *AuditEvent
    integrityMgr    *IntegrityManager
    complianceMgr   *ComplianceManager
    retentionMgr    *RetentionManager
    queryEngine     *QueryEngine
}
```

**Key Features:**
- **Batched Processing**: Events processed in configurable batches (100-10,000 events)
- **Async Architecture**: Non-blocking event ingestion with goroutine pools
- **Circuit Breakers**: Automatic failover and recovery for backend failures
- **Performance Metrics**: Comprehensive Prometheus metrics for monitoring

#### 2. Event Types & Schemas (`pkg/audit/events.go`)

Structured event definitions with rich metadata:

```go
type AuditEvent struct {
    // Core identification
    EventID     string    `json:"event_id"`
    Timestamp   time.Time `json:"timestamp"`
    Type        EventType `json:"type"`
    Category    string    `json:"category"`
    Severity    string    `json:"severity"`
    
    // Context information
    Actor    *ActorContext    `json:"actor,omitempty"`
    Resource *ResourceContext `json:"resource,omitempty"`
    Network  *NetworkContext  `json:"network,omitempty"`
    System   *SystemContext   `json:"system,omitempty"`
    
    // Event details
    Message     string                 `json:"message"`
    Details     map[string]interface{} `json:"details,omitempty"`
    Outcome     string                 `json:"outcome"`
    Risk        string                 `json:"risk_level"`
    
    // Compliance and retention
    Compliance     []string      `json:"compliance_standards,omitempty"`
    RetentionLevel string        `json:"retention_level"`
    Classification string        `json:"data_classification"`
}
```

#### 3. Backend Integrations (`pkg/audit/backends/`)

Support for multiple audit backends with consistent interfaces:

- **Elasticsearch/OpenSearch**: Production-ready with bulk indexing, templates, lifecycle management
- **Splunk HEC**: Enterprise integration with batch processing and health monitoring
- **SIEM Systems**: Generic CEF/LEEF format support for security systems
- **File-based**: High-performance structured logging with rotation
- **Cloud Providers**: AWS CloudTrail, Azure Monitor, GCP Cloud Audit integration
- **Webhook**: Custom endpoint integration for specialized systems

#### 4. Integrity Protection (`pkg/audit/integrity.go`)

Cryptographic protection against tampering:

```go
type IntegrityManager struct {
    hashChain    *HashChain
    signer       *DigitalSigner
    verifier     *ChainVerifier
    keyManager   KeyManager
}

type HashChain struct {
    Algorithm    string `json:"algorithm"`     // SHA-256
    CurrentHash  string `json:"current_hash"`
    PreviousHash string `json:"previous_hash"`
    BlockNumber  int64  `json:"block_number"`
}
```

**Protection Methods:**
- **Hash Chains**: Sequential cryptographic linking of log entries
- **Digital Signatures**: RSA-PSS signatures for non-repudiation
- **Key Management**: Integration with external PKI or automated key generation
- **Verification**: Real-time and batch integrity verification

## Audit Event Types

### Authentication Events

```yaml
Authentication Success:
  Type: AUTHENTICATION
  Severity: INFO
  Includes: user_id, session_id, auth_method, mfa_status, client_ip

Authentication Failure:
  Type: AUTHENTICATION_FAILURE  
  Severity: WARN
  Includes: attempted_user, failure_reason, client_ip, user_agent
  
Session Events:
  Type: SESSION_START/SESSION_END
  Severity: INFO
  Includes: session_duration, activities_performed
```

### Authorization Events

```yaml
Access Granted:
  Type: AUTHORIZATION
  Severity: INFO
  Includes: user_id, resource, permissions_granted, access_method

Access Denied:
  Type: AUTHORIZATION_FAILURE
  Severity: WARN
  Includes: user_id, requested_resource, denied_permissions, reason

Privilege Escalation:
  Type: PRIVILEGE_ESCALATION
  Severity: CRITICAL
  Includes: from_role, to_role, escalation_method, approver
```

### Data Access Events

```yaml
Data Read:
  Type: DATA_ACCESS
  Severity: INFO
  Includes: data_type, classification, access_method, query_details

Data Modification:
  Type: DATA_MODIFICATION
  Severity: WARN
  Includes: modified_fields, old_values, new_values, modification_type

Data Export:
  Type: DATA_EXPORT
  Severity: HIGH
  Includes: exported_data, destination, export_method, approval_status
```

### System Events

```yaml
Configuration Changes:
  Type: CONFIGURATION_CHANGE
  Severity: HIGH
  Includes: component, changed_settings, old_config, new_config

Software Installation:
  Type: SOFTWARE_CHANGE
  Severity: MEDIUM
  Includes: software_name, version, installation_method, approval

System Access:
  Type: SYSTEM_ACCESS
  Severity: INFO
  Includes: access_type, system_component, elevated_privileges
```

### Security Events

```yaml
Security Violations:
  Type: SECURITY_VIOLATION
  Severity: CRITICAL
  Includes: violation_type, security_policy, mitigation_actions

Suspicious Activity:
  Type: SUSPICIOUS_ACTIVITY
  Severity: HIGH
  Includes: activity_pattern, confidence_level, related_events

Intrusion Detection:
  Type: INTRUSION_DETECTED
  Severity: CRITICAL
  Includes: detection_method, threat_indicators, response_actions
```

### O-RAN Specific Events

```yaml
Network Function Operations:
  Type: NETWORK_FUNCTION_OPERATION
  Severity: INFO
  Includes: nf_type, operation, parameters, outcome

Policy Operations:
  Type: POLICY_OPERATION
  Severity: MEDIUM
  Includes: policy_type, policy_id, operation, affected_resources

Network Slice Operations:
  Type: SLICE_OPERATION
  Severity: HIGH
  Includes: slice_id, operation, qos_parameters, tenant
```

## Backend Integrations

### Elasticsearch/OpenSearch Configuration

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: elasticsearch-audit
spec:
  backends:
    - name: elasticsearch
      type: elasticsearch
      config:
        url: "https://elasticsearch.company.com:9200"
        index_pattern: "audit-logs-%{+YYYY.MM.dd}"
        username: "audit_user"
        password_secret_ref:
          name: elastic-credentials
          key: password
        tls:
          enabled: true
          ca_cert_secret_ref:
            name: elastic-ca
            key: ca.crt
        bulk_size: 1000
        flush_interval: "10s"
        index_template:
          enabled: true
          number_of_shards: 2
          number_of_replicas: 1
        lifecycle_policy:
          enabled: true
          hot_phase_days: 7
          warm_phase_days: 30
          cold_phase_days: 90
          delete_after_days: 2555  # 7 years
```

### Splunk Integration Configuration

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: splunk-audit
spec:
  backends:
    - name: splunk
      type: splunk
      config:
        hec_url: "https://splunk.company.com:8088/services/collector"
        hec_token_secret_ref:
          name: splunk-credentials
          key: hec_token
        index: "nephoran_audit"
        source: "nephoran_intent_operator"
        sourcetype: "_json"
        batch_size: 500
        batch_timeout: "5s"
        tls:
          enabled: true
          verify_certificate: true
        health_check:
          enabled: true
          interval: "30s"
```

### SIEM Integration (CEF Format)

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: siem-audit
spec:
  backends:
    - name: arcsight
      type: siem
      config:
        format: "cef"
        endpoint: "https://arcsight.company.com/cef"
        vendor: "Nephoran"
        product: "Intent Operator"
        version: "1.0"
        device_event_class_id_mapping:
          AUTHENTICATION: "100"
          AUTHENTICATION_FAILURE: "101"
          AUTHORIZATION_FAILURE: "102"
          SECURITY_VIOLATION: "200"
        severity_mapping:
          INFO: 3
          WARN: 5
          HIGH: 8
          CRITICAL: 10
```

### File-based Configuration

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: file-audit
spec:
  backends:
    - name: file
      type: file
      config:
        file_path: "/var/log/nephoran/audit/audit.log"
        format: "json"
        rotation:
          max_size: "100MB"
          max_files: 10
          max_age: "30d"
          compress: true
        permissions: "0600"
        sync_write: false  # For performance
        buffer_size: "64KB"
```

## Compliance Frameworks

### SOC2 Type II Compliance

The audit system provides comprehensive SOC2 compliance evidence:

#### Security (CC6)
- **CC6.1**: Access controls and user authentication
- **CC6.2**: Authorization and privilege management
- **CC6.3**: System access logging and monitoring
- **CC6.6**: Network security controls
- **CC6.7**: Data transmission security
- **CC6.8**: System development lifecycle controls

#### Availability (A1)
- **A1.1**: System availability monitoring
- **A1.2**: Capacity management and scaling
- **A1.3**: System backup and recovery

#### Processing Integrity (PI1)
- **PI1.1**: Data processing accuracy
- **PI1.2**: Input validation and error handling
- **PI1.3**: Processing completeness

#### Confidentiality (C1)
- **C1.1**: Data classification and handling
- **C1.2**: Encryption and key management
- **C1.3**: Data access controls

#### Privacy (P1)
- **P1.1**: Personal data collection and processing
- **P1.2**: Data retention and disposal
- **P1.3**: Privacy notice and consent

### ISO 27001 Compliance

Comprehensive coverage of Annex A controls:

#### Access Control (A.9)
- **A.9.1**: User access management
- **A.9.2**: User authentication
- **A.9.3**: Privileged access management
- **A.9.4**: Access control audit trails

#### Cryptography (A.10)
- **A.10.1**: Encryption policies and procedures
- **A.10.2**: Key management

#### Operations Security (A.12)
- **A.12.4**: Logging and monitoring
- **A.12.6**: Management of technical vulnerabilities
- **A.12.7**: Information systems audit considerations

#### Communications Security (A.13)
- **A.13.1**: Network security management
- **A.13.2**: Information transfer security

#### System Acquisition (A.14)
- **A.14.1**: Security in development and support processes
- **A.14.2**: Security in supplier relationships

#### Incident Management (A.16)
- **A.16.1**: Management of information security incidents

### PCI DSS Compliance

For environments processing cardholder data:

#### Requirement 2: Default Passwords and Security Parameters
- System configuration change auditing
- Default account monitoring

#### Requirement 3: Protect Stored Cardholder Data
- Data access logging and monitoring
- Encryption key management auditing

#### Requirement 4: Encrypt Transmission of Cardholder Data
- Network transmission security auditing

#### Requirement 7: Restrict Access by Business Need-to-Know
- Access control and privilege management
- Role-based access auditing

#### Requirement 8: Identify and Authenticate Access
- User authentication and session management
- Multi-factor authentication monitoring

#### Requirement 10: Track and Monitor All Access
- Comprehensive audit trail requirements
- Log integrity and protection

#### Requirement 11: Regularly Test Security Systems
- Vulnerability assessment auditing
- Penetration testing activity logging

### Compliance Configuration Example

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: multi-compliance-audit
spec:
  compliance:
    standards:
      - name: "SOC2"
        enabled: true
        controls:
          - "CC6.1"
          - "CC6.2"
          - "CC6.3"
          - "A1.1"
        reporting:
          enabled: true
          schedule: "weekly"
          format: "pdf"
      - name: "ISO27001"
        enabled: true
        controls:
          - "A.9.1"
          - "A.9.2"
          - "A.12.4"
        evidence_collection: true
        assessment_schedule: "monthly"
      - name: "PCI_DSS"
        enabled: true
        requirements:
          - "REQ_2"
          - "REQ_7"
          - "REQ_8"
          - "REQ_10"
        cardholder_data_monitoring: true
  retention:
    default_period: "7y"
    compliance_overrides:
      SOC2: "7y"
      ISO27001: "3y" 
      PCI_DSS: "1y"
    legal_hold:
      enabled: true
      notification_endpoint: "https://legal.company.com/hold"
```

## Log Integrity & Tamper Protection

### Hash Chain Implementation

Each audit log entry is cryptographically linked to the previous entry:

```go
type HashChainEntry struct {
    EventID         string    `json:"event_id"`
    Timestamp       time.Time `json:"timestamp"`
    ContentHash     string    `json:"content_hash"`     // SHA-256 of event data
    PreviousHash    string    `json:"previous_hash"`    // Hash of previous entry
    ChainHash       string    `json:"chain_hash"`       // Combined hash
    BlockNumber     int64     `json:"block_number"`
    DigitalSignature string   `json:"signature,omitempty"`
}
```

### Digital Signature

RSA-PSS signatures provide non-repudiation:

```yaml
signature_config:
  algorithm: "RSA-PSS"
  key_size: 3072
  hash_function: "SHA-256"
  salt_length: 32
  key_rotation_days: 90
  signature_frequency: "every_block"  # or "daily", "hourly"
```

### Key Management Options

#### 1. Internal Key Generation

```yaml
key_management:
  type: "internal"
  key_store: "kubernetes_secret"
  key_size: 3072
  rotation_schedule: "90d"
  backup_enabled: true
  backup_encryption: "AES-256-GCM"
```

#### 2. External PKI Integration

```yaml
key_management:
  type: "external_pki"
  ca_url: "https://pki.company.com/ca"
  certificate_profile: "audit_signing"
  key_escrow: true
  revocation_check: true
```

#### 3. Hardware Security Module (HSM)

```yaml
key_management:
  type: "hsm"
  hsm_endpoint: "pkcs11://hsm.company.com"
  partition: "audit_partition"
  authentication: "password"
  credential_secret_ref:
    name: hsm-credentials
    key: partition_password
```

### Integrity Verification

#### Continuous Verification

```yaml
integrity_verification:
  continuous_check:
    enabled: true
    interval: "1h"
    batch_size: 1000
    alert_on_failure: true
    
  scheduled_verification:
    enabled: true
    schedule: "0 2 * * *"  # Daily at 2 AM
    full_chain_check: true
    performance_monitoring: true
    
  on_demand_verification:
    enabled: true
    api_endpoint: "/api/v1/audit/verify"
    max_range: "30d"
```

#### Verification Report

```json
{
  "verification_id": "verify_20241201_120000",
  "timestamp": "2024-12-01T12:00:00Z",
  "status": "SUCCESS",
  "range": {
    "start_time": "2024-11-01T00:00:00Z",
    "end_time": "2024-12-01T12:00:00Z"
  },
  "statistics": {
    "total_entries": 1500000,
    "verified_entries": 1500000,
    "failed_entries": 0,
    "chain_breaks": 0,
    "signature_failures": 0
  },
  "performance": {
    "verification_duration": "45.2s",
    "entries_per_second": 33185,
    "memory_usage": "256MB"
  },
  "compliance": {
    "soc2_compliant": true,
    "iso27001_compliant": true,
    "pci_dss_compliant": true
  }
}
```

## Retention Policies

### Multi-Tier Storage Architecture

```yaml
retention_config:
  storage_tiers:
    - name: "hot"
      duration: "30d"
      storage_class: "ssd"
      query_performance: "high"
      cost_factor: 1.0
      
    - name: "warm" 
      duration: "365d"
      storage_class: "standard"
      query_performance: "medium"
      cost_factor: 0.5
      
    - name: "cold"
      duration: "2555d"  # 7 years
      storage_class: "archive"
      query_performance: "low"
      cost_factor: 0.1
      restoration_time: "12h"
      
    - name: "frozen"
      duration: "indefinite"
      storage_class: "deep_archive"
      legal_hold: true
      cost_factor: 0.01
      restoration_time: "48h"
```

### Event-Specific Retention

```yaml
retention_policies:
  by_event_type:
    AUTHENTICATION: "1y"
    AUTHENTICATION_FAILURE: "3y"
    AUTHORIZATION_FAILURE: "7y"
    SECURITY_VIOLATION: "10y"
    DATA_ACCESS: "7y"
    CONFIGURATION_CHANGE: "5y"
    
  by_severity:
    INFO: "1y"
    WARN: "3y"
    HIGH: "7y"
    CRITICAL: "10y"
    
  by_compliance:
    SOC2: "7y"
    ISO27001: "3y"
    PCI_DSS: "1y"
    HIPAA: "6y"
    GDPR: "3y"
```

### Legal Hold Management

```yaml
legal_hold:
  enabled: true
  notification:
    endpoint: "https://legal.company.com/api/holds"
    authentication: "bearer_token"
    token_secret_ref:
      name: legal-api-credentials
      key: bearer_token
      
  policies:
    - name: "litigation_hold"
      trigger_events:
        - "SECURITY_VIOLATION" 
        - "DATA_BREACH"
        - "POLICY_VIOLATION"
      hold_duration: "indefinite"
      approval_required: true
      
    - name: "investigation_hold"
      trigger_events:
        - "SUSPICIOUS_ACTIVITY"
        - "FRAUD_DETECTED"
      hold_duration: "2y"
      auto_release: true
```

## Query & Analysis

### Query Engine API

#### Basic Query

```bash
# Query authentication failures
curl -X POST "https://audit-api.company.com/query" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2024-11-01T00:00:00Z",
    "end_time": "2024-12-01T00:00:00Z",
    "event_types": ["AUTHENTICATION_FAILURE"],
    "limit": 100,
    "sort_by": "timestamp",
    "sort_order": "desc"
  }'
```

#### Advanced Query with Aggregation

```bash
# Security analysis with aggregation
curl -X POST "https://audit-api.company.com/query" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2024-11-01T00:00:00Z",
    "end_time": "2024-12-01T00:00:00Z",
    "event_types": ["AUTHENTICATION_FAILURE", "SECURITY_VIOLATION"],
    "group_by": ["client_ip", "user_id"],
    "aggregations": {
      "event_count": "count",
      "unique_users": "unique",
      "time_distribution": "histogram"
    }
  }'
```

#### Text Search

```bash
# Full-text search
curl -X POST "https://audit-api.company.com/search" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "failed login administrator",
    "time_range": "7d",
    "highlight": true
  }'
```

### Security Analysis API

#### Threat Detection

```bash
# Security analysis
curl -X POST "https://audit-api.company.com/analyze/security" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "time_range": "24h",
    "include_recommendations": true,
    "threat_intelligence": true
  }'
```

Response:
```json
{
  "analysis_id": "analysis_20241201_120000",
  "timestamp": "2024-12-01T12:00:00Z",
  "risk_score": 35.7,
  "suspicious_activities": [
    {
      "type": "brute_force",
      "severity": "HIGH",
      "confidence": 0.85,
      "description": "Multiple failed authentication attempts from 192.168.1.100",
      "evidence_count": 25,
      "mitre_technique": "T1110",
      "recommendations": [
        "Enable account lockout policy",
        "Implement rate limiting"
      ]
    }
  ],
  "compliance_violations": [
    {
      "standard": "SOC2",
      "control": "CC6.1",
      "severity": "MEDIUM",
      "description": "Unauthorized access attempt detected"
    }
  ]
}
```

#### User Behavior Analysis

```bash
# User activity analysis
curl -X GET "https://audit-api.company.com/users/admin@company.com/activity" \
  -H "Authorization: Bearer $TOKEN" \
  -G -d "time_range=7d" -d "include_anomalies=true"
```

## Configuration

### Complete Configuration Example

```yaml
apiVersion: nephoran.io/v1
kind: AuditTrail
metadata:
  name: enterprise-audit
  namespace: nephoran-system
spec:
  # Core audit system configuration
  core:
    enabled: true
    log_level: "INFO"
    batch_size: 1000
    batch_timeout: "10s"
    worker_pool_size: 10
    buffer_size: 10000
    
    # Performance settings
    performance:
      max_events_per_second: 1000
      async_processing: true
      compression: "gzip"
      
    # Security settings
    security:
      encryption_at_rest: true
      encryption_algorithm: "AES-256-GCM"
      tls_required: true
      min_tls_version: "1.2"
      
  # Event capture configuration
  events:
    capture_all: false
    event_types:
      - AUTHENTICATION
      - AUTHENTICATION_FAILURE
      - AUTHORIZATION
      - AUTHORIZATION_FAILURE
      - DATA_ACCESS
      - DATA_MODIFICATION
      - CONFIGURATION_CHANGE
      - SECURITY_VIOLATION
      - SUSPICIOUS_ACTIVITY
      
    enrichment:
      enabled: true
      include_request_headers: false
      include_response_bodies: false
      max_field_length: 1024
      sensitive_fields:
        - "password"
        - "token" 
        - "key"
        - "secret"
        
    filtering:
      exclude_health_checks: true
      exclude_metrics_endpoints: true
      exclude_internal_services: true
      custom_filters:
        - field: "user_id"
          operator: "not_equals"
          value: "system"
          
  # Backend configurations
  backends:
    - name: elasticsearch
      type: elasticsearch
      enabled: true
      primary: true
      config:
        url: "https://elasticsearch.company.com:9200"
        index_pattern: "audit-logs-%{+YYYY.MM.dd}"
        bulk_size: 1000
        flush_interval: "10s"
        
    - name: splunk
      type: splunk
      enabled: true
      primary: false
      config:
        hec_url: "https://splunk.company.com:8088"
        index: "nephoran_audit"
        batch_size: 500
        
    - name: file
      type: file
      enabled: true
      primary: false
      config:
        file_path: "/var/log/nephoran/audit.log"
        rotation:
          max_size: "100MB"
          max_files: 10
          
  # Integrity protection
  integrity:
    enabled: true
    hash_algorithm: "SHA-256"
    chain_validation: true
    digital_signatures:
      enabled: true
      algorithm: "RSA-PSS"
      key_size: 3072
      rotation_days: 90
      
  # Compliance frameworks
  compliance:
    standards:
      - name: "SOC2"
        enabled: true
        controls: ["CC6.1", "CC6.2", "CC6.3"]
        reporting_schedule: "weekly"
        
      - name: "ISO27001"
        enabled: true
        controls: ["A.9.1", "A.9.2", "A.12.4"]
        assessment_schedule: "monthly"
        
      - name: "PCI_DSS"
        enabled: false
        
  # Retention policies
  retention:
    default_period: "7y"
    storage_tiers:
      - name: "hot"
        duration: "30d"
        storage_class: "ssd"
      - name: "warm"
        duration: "365d"
        storage_class: "standard"
      - name: "cold"
        duration: "2555d"
        storage_class: "archive"
        
    legal_hold:
      enabled: true
      notification_endpoint: "https://legal.company.com/holds"
      
  # Query and analysis
  query:
    enabled: true
    api_endpoint: "/api/v1/audit"
    max_query_range: "30d"
    max_results: 100000
    cache_duration: "1h"
    
    security_analysis:
      enabled: true
      threat_detection: true
      anomaly_detection: true
      ml_models: ["isolation_forest", "one_class_svm"]
      
  # Monitoring and alerting  
  monitoring:
    prometheus:
      enabled: true
      metrics_port: 9090
      
    alerting:
      enabled: true
      alert_manager_url: "http://alertmanager:9093"
      rules:
        - name: "high_failure_rate"
          condition: "authentication_failures > 100"
          severity: "warning"
          duration: "5m"
          
        - name: "security_violation"
          condition: "security_violations > 0"
          severity: "critical"
          duration: "0s"
          
status:
  phase: "Running"
  backends:
    elasticsearch:
      status: "Connected"
      last_sync: "2024-12-01T12:00:00Z"
      events_processed: 1500000
    splunk:
      status: "Connected" 
      last_sync: "2024-12-01T12:00:00Z"
      events_processed: 1500000
  integrity:
    chain_status: "Valid"
    last_verification: "2024-12-01T11:00:00Z"
    signature_status: "Valid"
  compliance:
    soc2_status: "Compliant"
    iso27001_status: "Compliant"
    last_assessment: "2024-12-01T10:00:00Z"
```

### Environment Variables

```bash
# Core configuration
AUDIT_SYSTEM_ENABLED=true
AUDIT_LOG_LEVEL=INFO
AUDIT_BATCH_SIZE=1000
AUDIT_WORKER_POOL_SIZE=10

# Security settings
AUDIT_ENCRYPTION_ENABLED=true
AUDIT_TLS_REQUIRED=true
AUDIT_MIN_TLS_VERSION=1.2

# Backend configuration
AUDIT_ELASTICSEARCH_URL=https://elasticsearch.company.com:9200
AUDIT_SPLUNK_HEC_URL=https://splunk.company.com:8088

# Integrity protection
AUDIT_INTEGRITY_ENABLED=true
AUDIT_DIGITAL_SIGNATURES_ENABLED=true
AUDIT_KEY_ROTATION_DAYS=90

# Compliance
AUDIT_SOC2_ENABLED=true
AUDIT_ISO27001_ENABLED=true
AUDIT_RETENTION_PERIOD=7y

# Monitoring
AUDIT_PROMETHEUS_ENABLED=true
AUDIT_ALERTING_ENABLED=true
```

## Monitoring & Alerting

### Prometheus Metrics

The audit system exports comprehensive metrics:

#### Core Audit Metrics
```prometheus
# Event processing metrics
audit_events_total{type="AUTHENTICATION", status="success"} 15847
audit_events_processing_duration_seconds{backend="elasticsearch"} 0.002
audit_batch_processing_duration_seconds{backend="elasticsearch"} 0.045

# Backend health metrics
audit_backend_status{backend="elasticsearch", status="up"} 1
audit_backend_events_processed_total{backend="elasticsearch"} 1500000
audit_backend_errors_total{backend="elasticsearch"} 12

# Integrity metrics
audit_integrity_verifications_total{status="success"} 24
audit_integrity_chain_breaks_total 0
audit_signature_failures_total 0

# Compliance metrics
audit_compliance_violations_total{standard="SOC2", control="CC6.1"} 0
audit_compliance_assessments_total{standard="ISO27001"} 30
audit_retention_policy_violations_total 0

# Query metrics
audit_queries_total{backend="elasticsearch", status="success"} 1547
audit_query_duration_seconds{backend="elasticsearch"} 0.125
audit_security_analyses_total{status="success"} 47

# Performance metrics
audit_system_cpu_usage 0.15
audit_system_memory_usage_bytes 268435456
audit_event_queue_size 245
audit_processing_lag_seconds 0.8
```

#### Grafana Dashboard Queries

**Event Processing Rate:**
```promql
rate(audit_events_total[5m])
```

**Backend Health:**
```promql
audit_backend_status{status="up"}
```

**Integrity Status:**
```promql
increase(audit_integrity_chain_breaks_total[1h])
```

**Compliance Violations:**
```promql
increase(audit_compliance_violations_total[24h])
```

### Alerting Rules

#### High-Priority Alerts

```yaml
groups:
  - name: audit.critical
    rules:
      - alert: AuditIntegrityFailure
        expr: increase(audit_integrity_chain_breaks_total[5m]) > 0
        for: 0s
        labels:
          severity: critical
        annotations:
          summary: "Audit log integrity failure detected"
          description: "Audit log chain break or signature failure detected"
          
      - alert: AuditBackendDown
        expr: audit_backend_status{status="up"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Audit backend {{ $labels.backend }} is down"
          
      - alert: SecurityViolationDetected
        expr: increase(audit_events_total{type="SECURITY_VIOLATION"}[1m]) > 0
        for: 0s
        labels:
          severity: critical
        annotations:
          summary: "Security violation detected in audit logs"
          
  - name: audit.warning
    rules:
      - alert: HighAuthenticationFailureRate
        expr: rate(audit_events_total{type="AUTHENTICATION_FAILURE"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
          
      - alert: AuditProcessingLag
        expr: audit_processing_lag_seconds > 30
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Audit processing lag detected"
          
      - alert: ComplianceViolation
        expr: increase(audit_compliance_violations_total[1h]) > 0
        for: 0s
        labels:
          severity: warning
        annotations:
          summary: "Compliance violation detected"
```

## Best Practices

### Security Best Practices

1. **Least Privilege Access**
   ```bash
   # Create dedicated service account
   kubectl create serviceaccount audit-system -n nephoran-system
   
   # Minimal RBAC permissions
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: audit-system
   rules:
   - apiGroups: [""]
     resources: ["secrets", "configmaps"]
     verbs: ["get", "list"]
   - apiGroups: ["nephoran.io"]
     resources: ["audittrails"]
     verbs: ["get", "list", "watch", "update"]
   ```

2. **Network Security**
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: audit-system-netpol
   spec:
     podSelector:
       matchLabels:
         app: audit-system
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - podSelector:
           matchLabels:
             app: nephoran-controller
     egress:
     - to: []
       ports:
       - protocol: TCP
         port: 9200  # Elasticsearch
       - protocol: TCP
         port: 8088  # Splunk HEC
   ```

3. **Encryption Configuration**
   ```yaml
   security:
     encryption_at_rest:
       enabled: true
       algorithm: "AES-256-GCM"
       key_management: "external_kms"
       kms_endpoint: "https://kms.company.com"
     
     encryption_in_transit:
       enabled: true
       min_tls_version: "1.3"
       cipher_suites:
         - "TLS_AES_256_GCM_SHA384"
         - "TLS_CHACHA20_POLY1305_SHA256"
   ```

### Performance Optimization

1. **Batch Processing Configuration**
   ```yaml
   performance:
     batch_size: 1000          # Events per batch
     batch_timeout: "10s"      # Max wait time
     worker_pool_size: 20      # Concurrent workers
     buffer_size: 50000        # Event buffer
     compression: "zstd"       # Compression algorithm
   ```

2. **Backend Optimization**
   ```yaml
   elasticsearch:
     bulk_size: 5000
     bulk_timeout: "30s"
     refresh_interval: "30s"
     number_of_shards: 3
     number_of_replicas: 1
     
   splunk:
     batch_size: 1000
     compression: true
     keep_alive: true
     connection_pool_size: 10
   ```

3. **Resource Limits**
   ```yaml
   resources:
     requests:
       memory: "256Mi"
       cpu: "200m"
     limits:
       memory: "1Gi"
       cpu: "500m"
       
   # JVM tuning for high throughput
   jvm_options:
     - "-Xms512m"
     - "-Xmx1024m"
     - "-XX:+UseG1GC"
     - "-XX:MaxGCPauseMillis=200"
   ```

### Operational Best Practices

1. **Health Monitoring**
   ```bash
   # Monitor audit system health
   kubectl get audittrail enterprise-audit -o jsonpath='{.status.phase}'
   
   # Check backend connectivity
   kubectl logs -l app=audit-system -c backend-health-checker
   
   # Verify integrity status
   curl -s https://audit-api.company.com/health/integrity | jq '.chain_status'
   ```

2. **Backup and Recovery**
   ```bash
   # Backup audit configuration
   kubectl get audittrail -o yaml > audit-config-backup.yaml
   
   # Export audit data
   curl -X POST "https://audit-api.company.com/export" \
     -d '{"start_time":"2024-01-01T00:00:00Z","format":"json"}' \
     > audit-export.jsonl
     
   # Verify backup integrity
   audit-verify --file audit-export.jsonl --check-signatures
   ```

3. **Disaster Recovery Testing**
   ```yaml
   # DR testing schedule
   disaster_recovery:
     testing_schedule: "quarterly"
     scenarios:
       - "primary_backend_failure"
       - "network_partition"
       - "key_compromise"
       - "data_corruption"
     recovery_time_objective: "4h"
     recovery_point_objective: "1h"
   ```

### Compliance Best Practices

1. **Evidence Collection**
   ```bash
   # Generate compliance report
   kubectl exec -it audit-system-pod -- \
     audit-compliance-report \
     --standard SOC2 \
     --period "2024-01-01:2024-12-31" \
     --output compliance-report.pdf
   ```

2. **Regular Assessment**
   ```yaml
   compliance_automation:
     assessments:
       - standard: "SOC2"
         schedule: "monthly"
         automated: true
         controls: ["CC6.1", "CC6.2", "CC6.3"]
         
       - standard: "ISO27001" 
         schedule: "quarterly"
         automated: false
         external_auditor: true
   ```

3. **Legal Hold Management**
   ```bash
   # Place legal hold
   curl -X POST "https://audit-api.company.com/legal-hold" \
     -H "Authorization: Bearer $LEGAL_TOKEN" \
     -d '{
       "case_id": "CASE-2024-001",
       "custodian": "user@company.com",
       "date_range": {
         "start": "2024-01-01T00:00:00Z",
         "end": "2024-06-30T23:59:59Z"
       },
       "reason": "litigation_hold"
     }'
   ```

## Troubleshooting

### Common Issues

#### 1. Backend Connection Failures

**Symptoms:**
```
audit_backend_status{backend="elasticsearch"} 0
ERROR: Failed to connect to Elasticsearch: connection refused
```

**Resolution:**
```bash
# Check network connectivity
kubectl exec -it audit-system-pod -- curl -v https://elasticsearch.company.com:9200

# Verify credentials
kubectl get secret elastic-credentials -o jsonpath='{.data.password}' | base64 -d

# Check certificate validity
kubectl exec -it audit-system-pod -- openssl s_client -connect elasticsearch.company.com:9200

# Update backend configuration
kubectl patch audittrail enterprise-audit --type='merge' -p='{
  "spec": {
    "backends": [{
      "name": "elasticsearch",
      "config": {
        "url": "https://new-elasticsearch.company.com:9200"
      }
    }]
  }
}'
```

#### 2. Integrity Chain Breaks

**Symptoms:**
```
audit_integrity_chain_breaks_total 1
CRITICAL: Hash chain verification failed at block 15847
```

**Resolution:**
```bash
# Check integrity status
curl https://audit-api.company.com/integrity/status

# Run detailed verification
kubectl exec -it audit-system-pod -- \
  audit-verify --range "2024-12-01:2024-12-02" --verbose

# If corruption confirmed, restore from backup
kubectl exec -it audit-system-pod -- \
  audit-restore --backup-file /backups/audit-20241201.tar.gz \
  --verify-integrity
```

#### 3. High Processing Lag

**Symptoms:**
```
audit_processing_lag_seconds 120
audit_event_queue_size 50000
```

**Resolution:**
```bash
# Scale up worker pool
kubectl patch audittrail enterprise-audit --type='merge' -p='{
  "spec": {
    "core": {
      "worker_pool_size": 50,
      "batch_size": 2000
    }
  }
}'

# Increase resources
kubectl patch deployment audit-system --type='merge' -p='{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "audit-system",
          "resources": {
            "limits": {
              "cpu": "1000m",
              "memory": "2Gi"
            }
          }
        }]
      }
    }
  }
}'

# Check backend performance
kubectl exec -it audit-system-pod -- \
  audit-benchmark --backend elasticsearch --duration 60s
```

### Diagnostic Commands

```bash
# System health check
audit-health-check --comprehensive

# Performance analysis
audit-performance-report --period 24h

# Integrity verification
audit-verify --full-chain --signatures

# Compliance assessment
audit-compliance-check --standard SOC2 --detailed

# Query performance analysis
audit-query-analyzer --slow-queries --optimize

# Export diagnostic bundle
audit-diagnostic-export --include-logs --include-configs --output diag.tar.gz
```

### Log Analysis

```bash
# Search for errors
kubectl logs -l app=audit-system | grep -i error

# Analyze processing patterns
kubectl logs -l app=audit-system | grep "batch_processed" | \
  awk '{print $1, $NF}' | sort

# Monitor real-time events
kubectl logs -f -l app=audit-system --since=1m

# Performance metrics
kubectl logs -l app=audit-system | grep "performance_metrics" | tail -10
```

## API Reference

### REST API Endpoints

#### Core Audit Operations
```
POST   /api/v1/audit/events              # Submit audit event
GET    /api/v1/audit/events              # Query audit events  
POST   /api/v1/audit/query               # Advanced query
POST   /api/v1/audit/search              # Full-text search
GET    /api/v1/audit/stats               # System statistics
```

#### Security Analysis
```
POST   /api/v1/audit/analyze/security    # Security analysis
GET    /api/v1/audit/analyze/threats     # Threat detection
GET    /api/v1/audit/analyze/anomalies   # Anomaly detection
POST   /api/v1/audit/analyze/user        # User behavior analysis
```

#### Integrity & Compliance
```
GET    /api/v1/audit/integrity/status    # Integrity status
POST   /api/v1/audit/integrity/verify    # Verify integrity
GET    /api/v1/audit/compliance/status   # Compliance status
POST   /api/v1/audit/compliance/report   # Generate report
```

#### System Management
```
GET    /api/v1/audit/health              # System health
GET    /api/v1/audit/metrics             # Prometheus metrics
POST   /api/v1/audit/export              # Data export
POST   /api/v1/audit/legal-hold          # Legal hold
```

### CLI Tools

```bash
# Query tool
auditctl query --start "2024-12-01" --type AUTHENTICATION_FAILURE --limit 100

# Analysis tool
auditctl analyze --security --period 7d --format json

# Integrity tool
auditctl verify --range "2024-12-01:2024-12-02" --check-signatures

# Export tool
auditctl export --start "2024-01-01" --format jsonl --output audit.jsonl

# Compliance tool
auditctl compliance --standard SOC2 --report --period 2024
```

---

This comprehensive audit trail and compliance system provides enterprise-grade security monitoring and regulatory compliance capabilities for the Nephoran Intent Operator, ensuring complete visibility and accountability across all system operations.
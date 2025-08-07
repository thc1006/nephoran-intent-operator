# O1 Interface Implementation Guide

## Overview

The O1 interface in the Nephoran Intent Operator provides comprehensive FCAPS (Fault, Configuration, Accounting, Performance, Security) management for O-RAN network functions following the O-RAN.WG10.O1-Interface.0-v07.00 specification.

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        SMO (Service Management & Orchestration)│
└────────────────────────┬───────────────────────────────────┘
                         │ O1 Interface
┌────────────────────────▼───────────────────────────────────┐
│                    O1 Controller                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │  Fault   │ │  Config  │ │Accounting│ │   Perf   │     │
│  │   Mgmt   │ │   Mgmt   │ │   Mgmt   │ │   Mgmt   │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Security │ │ NETCONF  │ │   YANG   │ │Streaming │     │
│  │   Mgmt   │ │  Server  │ │  Models  │ │ Service  │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
└─────────────────────────────────────────────────────────────┘
                         │ NETCONF/SSH/TLS
┌────────────────────────▼───────────────────────────────────┐
│                    Network Functions                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │   O-DU   │ │   O-CU   │ │Near-RT   │ │   5G     │     │
│  │          │ │          │ │   RIC    │ │   Core   │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

1. **O1 Controller** (`pkg/oran/o1/o1_controller.go`)
   - Kubernetes controller managing O1Interface CRDs
   - Orchestrates FCAPS managers lifecycle
   - Handles status reporting and reconciliation

2. **NETCONF Server** (`pkg/oran/o1/netconf_server.go`)
   - Full NETCONF 1.1 protocol implementation
   - SSH and TLS transport support
   - Session management and capabilities negotiation

3. **YANG Model Registry** (`pkg/oran/o1/yang_models_extended.go`)
   - Complete O-RAN WG10 YANG models
   - Model validation and schema management
   - XPath filtering and data transformation

4. **FCAPS Managers**
   - **Fault Manager** (`pkg/oran/o1/fault_manager.go`): Alarm correlation, root cause analysis
   - **Configuration Manager** (`pkg/oran/o1/config_manager.go`): Version control, drift detection
   - **Accounting Manager** (`pkg/oran/o1/accounting_manager.go`): Usage tracking, billing
   - **Performance Manager** (`pkg/oran/o1/performance_manager.go`): KPI collection, anomaly detection
   - **Security Manager** (`pkg/oran/o1/security_manager.go`): Certificate management, threat detection

5. **SMO Integration** (`pkg/oran/o1/smo_integration.go`)
   - Service registration and discovery
   - Hierarchical management support
   - Multi-vendor interoperability

6. **Streaming Service** (`pkg/oran/o1/o1_streaming.go`)
   - WebSocket-based real-time streaming
   - Multiple stream types (alarms, performance, configuration)
   - QoS and backpressure handling

## Deployment

### Prerequisites

- Kubernetes 1.19+
- Nephoran Intent Operator installed
- Network access to managed elements
- Valid credentials for NETCONF access

### Installation

1. **Deploy the O1Interface CRD:**

```bash
kubectl apply -f config/crd/bases/nephoran.io_o1interfaces.yaml
```

2. **Create credentials secret:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: o1-credentials
  namespace: default
type: Opaque
data:
  username: YWRtaW4=  # base64 encoded
  password: cGFzc3dvcmQxMjM=  # base64 encoded
  privateKey: <base64-encoded-ssh-key>  # Optional for SSH
```

3. **Create O1Interface resource:**

```yaml
apiVersion: nephoran.io/v1
kind: O1Interface
metadata:
  name: ran-o1-interface
  namespace: default
spec:
  host: "10.0.0.100"
  port: 830
  protocol: ssh
  credentials:
    usernameRef:
      name: o1-credentials
      key: username
    passwordRef:
      name: o1-credentials
      key: password
  fcaps:
    faultManagement:
      enabled: true
      correlationEnabled: true
      rootCauseAnalysis: true
      severityFilter:
        - CRITICAL
        - MAJOR
      alertManagerConfig:
        url: http://alertmanager:9093
    configurationManagement:
      enabled: true
      gitOpsEnabled: true
      versioningEnabled: true
      maxVersions: 10
      driftDetection: true
      driftCheckInterval: 300
    accountingManagement:
      enabled: true
      collectionInterval: 60
      billingEnabled: true
      fraudDetection: true
      retentionDays: 90
    performanceManagement:
      enabled: true
      collectionInterval: 15
      aggregationPeriods:
        - 5min
        - 15min
        - 1hour
      anomalyDetection: true
      prometheusConfig:
        url: http://prometheus:9090
        pushGatewayURL: http://pushgateway:9091
      grafanaConfig:
        url: http://grafana:3000
        dashboardID: o1-dashboard
    securityManagement:
      enabled: true
      certificateManagement: true
      intrusionDetection: true
      complianceMonitoring: true
      auditInterval: 3600
  smoConfig:
    endpoint: http://smo.o-ran.local:8080
    registrationEnabled: true
    hierarchicalManagement: true
    serviceDiscovery:
      type: kubernetes
  streamingConfig:
    enabled: true
    webSocketPort: 8080
    streamTypes:
      - alarms
      - performance
      - configuration
      - events
    qosLevel: Reliable
    maxConnections: 100
    rateLimiting:
      requestsPerSecond: 100
      burstSize: 200
  yangModels:
    - name: o-ran-hardware
      version: "1.0"
      source: configmap
      configMapRef:
        name: yang-models
        key: o-ran-hardware.yang
  highAvailability:
    enabled: true
    replicas: 2
    failoverTimeout: 30
  securityConfig:
    tlsConfig:
      enabled: true
      minVersion: TLS1.2
    oauth2Config:
      enabled: true
      provider: keycloak
      clientID: o1-interface
      clientSecretRef:
        name: oauth2-secret
        key: client-secret
```

## Usage

### Connecting to Managed Elements

The O1 interface automatically establishes NETCONF connections to managed elements based on the configuration:

```bash
# Check connection status
kubectl get o1interface ran-o1-interface -o jsonpath='{.status.connectionStatus}'

# View NETCONF capabilities
kubectl get o1interface ran-o1-interface -o jsonpath='{.status.connectionStatus.capabilities}'
```

### Fault Management

#### Viewing Active Alarms

```bash
# Get active alarms
kubectl exec -it <o1-controller-pod> -- o1ctl alarms list --interface ran-o1-interface

# Filter by severity
kubectl exec -it <o1-controller-pod> -- o1ctl alarms list --severity CRITICAL,MAJOR

# Get alarm details
kubectl exec -it <o1-controller-pod> -- o1ctl alarms get <alarm-id>
```

#### Alarm Correlation

The system automatically correlates related alarms:

```bash
# View correlated alarm groups
kubectl exec -it <o1-controller-pod> -- o1ctl alarms correlations

# Get root cause analysis
kubectl exec -it <o1-controller-pod> -- o1ctl alarms root-cause <managed-element>
```

### Configuration Management

#### Applying Configurations

```bash
# Apply configuration from file
kubectl exec -it <o1-controller-pod> -- o1ctl config apply -f config.xml

# Get current configuration
kubectl exec -it <o1-controller-pod> -- o1ctl config get --xpath "/interfaces"

# View configuration history
kubectl exec -it <o1-controller-pod> -- o1ctl config history
```

#### Configuration Versioning and Rollback

```bash
# List configuration versions
kubectl exec -it <o1-controller-pod> -- o1ctl config versions

# Rollback to previous version
kubectl exec -it <o1-controller-pod> -- o1ctl config rollback <version-id>

# Compare configurations
kubectl exec -it <o1-controller-pod> -- o1ctl config diff <version1> <version2>
```

#### Drift Detection

```bash
# Check for configuration drift
kubectl exec -it <o1-controller-pod> -- o1ctl config drift-check

# Enable automatic drift correction
kubectl exec -it <o1-controller-pod> -- o1ctl config drift-correction enable
```

### Performance Management

#### KPI Collection

```bash
# Start KPI collection
kubectl exec -it <o1-controller-pod> -- o1ctl perf collect \
  --kpis throughput,latency,packet_loss \
  --interval 15s \
  --aggregation 5min,15min

# Get current KPIs
kubectl exec -it <o1-controller-pod> -- o1ctl perf get --kpi throughput

# View aggregated metrics
kubectl exec -it <o1-controller-pod> -- o1ctl perf aggregated --period 15min
```

#### Anomaly Detection

```bash
# Enable anomaly detection
kubectl exec -it <o1-controller-pod> -- o1ctl perf anomaly-detection enable \
  --metrics latency,packet_loss \
  --algorithm isolation_forest

# View detected anomalies
kubectl exec -it <o1-controller-pod> -- o1ctl perf anomalies
```

### Security Management

#### Certificate Management

```bash
# Generate certificate
kubectl exec -it <o1-controller-pod> -- o1ctl security cert generate \
  --cn "ne.o-ran.local" \
  --validity 365

# Install certificate
kubectl exec -it <o1-controller-pod> -- o1ctl security cert install <cert-file>

# Check certificate expiry
kubectl exec -it <o1-controller-pod> -- o1ctl security cert check-expiry

# Rotate certificates
kubectl exec -it <o1-controller-pod> -- o1ctl security cert rotate
```

#### Intrusion Detection

```bash
# Enable IDS
kubectl exec -it <o1-controller-pod> -- o1ctl security ids enable \
  --rules brute_force,port_scan,dos_attack

# View security events
kubectl exec -it <o1-controller-pod> -- o1ctl security events

# Get threat analysis
kubectl exec -it <o1-controller-pod> -- o1ctl security threats
```

### Accounting Management

#### Usage Collection

```bash
# Start usage collection
kubectl exec -it <o1-controller-pod> -- o1ctl accounting collect \
  --metrics data_volume,session_count,bandwidth

# Get usage records
kubectl exec -it <o1-controller-pod> -- o1ctl accounting records \
  --start "2024-01-01" \
  --end "2024-01-31"

# Calculate billing
kubectl exec -it <o1-controller-pod> -- o1ctl accounting billing <record-id>
```

#### Fraud Detection

```bash
# Enable fraud detection
kubectl exec -it <o1-controller-pod> -- o1ctl accounting fraud-detection enable

# View fraud alerts
kubectl exec -it <o1-controller-pod> -- o1ctl accounting fraud-alerts
```

### Real-time Streaming

#### WebSocket Connection

```javascript
// JavaScript example for WebSocket streaming
const ws = new WebSocket('ws://o1-interface:8080/stream');

// Subscribe to alarm stream
ws.send(JSON.stringify({
  type: 'subscribe',
  stream: 'alarms',
  filter: 'severity=CRITICAL'
}));

// Handle incoming messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};
```

#### Stream Types

- **Alarms**: Real-time alarm notifications
- **Performance**: Live KPI updates
- **Configuration**: Configuration change events
- **Events**: General system events
- **Logs**: System and application logs
- **Topology**: Network topology changes
- **Status**: Component status updates

### SMO Integration

#### Registration

```bash
# Register with SMO
kubectl exec -it <o1-controller-pod> -- o1ctl smo register

# Check registration status
kubectl exec -it <o1-controller-pod> -- o1ctl smo status

# Update registration
kubectl exec -it <o1-controller-pod> -- o1ctl smo update
```

#### Hierarchical Management

```bash
# Establish hierarchy
kubectl exec -it <o1-controller-pod> -- o1ctl smo hierarchy create \
  --parent parent-ne \
  --children child-ne-1,child-ne-2

# Propagate configuration
kubectl exec -it <o1-controller-pod> -- o1ctl smo propagate-config \
  --parent parent-ne \
  --config config.xml
```

## YANG Models

### Supported O-RAN YANG Models

- `o-ran-hardware.yang` - Hardware management
- `o-ran-software-management.yang` - Software inventory and management
- `o-ran-performance-management.yang` - Performance metrics
- `o-ran-fault-management.yang` - Fault and alarm management
- `o-ran-file-management.yang` - File transfer and management
- `o-ran-troubleshooting.yang` - Diagnostic and troubleshooting
- `o-ran-security.yang` - Security management
- `o-ran-cm.yang` - Configuration management
- `o-ran-supervision.yang` - Supervision and watchdog
- `o-ran-beamforming.yang` - Beamforming configuration

### Custom YANG Models

You can add custom YANG models via ConfigMaps:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-yang-models
data:
  custom-model.yang: |
    module custom-model {
      namespace "urn:custom:model:1.0";
      prefix cm;
      
      container custom-config {
        leaf parameter {
          type string;
        }
      }
    }
```

## Monitoring and Observability

### Prometheus Metrics

The O1 interface exposes comprehensive metrics:

```
# NETCONF operations
o1_netconf_operations_total{operation="get-config",status="success"}
o1_netconf_operation_duration_seconds{operation="edit-config"}
o1_netconf_sessions_active

# Fault management
o1_active_alarms_total{severity="CRITICAL"}
o1_alarm_correlation_groups
o1_root_cause_analysis_duration_seconds

# Configuration management
o1_configuration_versions_total
o1_configuration_drift_detected_total
o1_configuration_rollbacks_total

# Performance management
o1_collected_kpis_total{kpi="throughput"}
o1_anomalies_detected_total
o1_aggregation_operations_total

# Security management
o1_certificate_expiry_days
o1_security_threats_detected_total
o1_compliance_score

# Accounting management
o1_usage_records_collected_total
o1_billing_calculations_total
o1_fraud_alerts_total

# Streaming
o1_stream_connections_active
o1_streamed_events_total{type="alarm"}
o1_stream_backpressure_events_total
```

### Grafana Dashboards

Pre-built dashboards are available for:

- FCAPS Overview
- Alarm Management
- Configuration Status
- Performance Analytics
- Security Monitoring
- Usage and Billing
- Streaming Statistics

### Distributed Tracing

The O1 interface integrates with Jaeger for distributed tracing:

```yaml
# Enable tracing in O1Interface
spec:
  observability:
    tracing:
      enabled: true
      endpoint: http://jaeger:14268/api/traces
      samplingRate: 0.1
```

## Troubleshooting

### Common Issues

#### Connection Failed

```bash
# Check credentials
kubectl get secret o1-credentials -o yaml

# Verify network connectivity
kubectl exec -it <o1-controller-pod> -- nc -zv <host> <port>

# Check NETCONF server logs
kubectl logs <o1-controller-pod> -c netconf-server
```

#### Performance Issues

```bash
# Check resource usage
kubectl top pod <o1-controller-pod>

# Analyze slow operations
kubectl exec -it <o1-controller-pod> -- o1ctl debug slow-ops

# Enable debug logging
kubectl exec -it <o1-controller-pod> -- o1ctl debug log-level DEBUG
```

#### Streaming Issues

```bash
# Check WebSocket connections
kubectl exec -it <o1-controller-pod> -- o1ctl stream connections

# View stream statistics
kubectl exec -it <o1-controller-pod> -- o1ctl stream stats

# Reset stream buffers
kubectl exec -it <o1-controller-pod> -- o1ctl stream reset-buffers
```

### Debug Commands

```bash
# Enable debug mode
kubectl exec -it <o1-controller-pod> -- o1ctl debug enable

# Capture NETCONF traffic
kubectl exec -it <o1-controller-pod> -- o1ctl debug capture --duration 60s

# Export debug bundle
kubectl exec -it <o1-controller-pod> -- o1ctl debug export > debug-bundle.tar.gz
```

## API Reference

### REST API Endpoints

```
GET    /api/v1/o1interfaces                    # List all O1 interfaces
GET    /api/v1/o1interfaces/{name}             # Get specific interface
POST   /api/v1/o1interfaces/{name}/connect     # Establish connection
DELETE /api/v1/o1interfaces/{name}/disconnect  # Close connection

# Fault Management
GET    /api/v1/alarms                          # List alarms
GET    /api/v1/alarms/{id}                     # Get alarm details
POST   /api/v1/alarms/{id}/clear               # Clear alarm
GET    /api/v1/alarms/correlations             # Get correlations

# Configuration Management
GET    /api/v1/config                          # Get configuration
POST   /api/v1/config                          # Apply configuration
GET    /api/v1/config/versions                 # List versions
POST   /api/v1/config/rollback/{version}       # Rollback

# Performance Management
GET    /api/v1/metrics                         # Get metrics
POST   /api/v1/metrics/collect                 # Start collection
GET    /api/v1/metrics/aggregated              # Get aggregated data
GET    /api/v1/metrics/anomalies               # Get anomalies

# Security Management
GET    /api/v1/security/certificates           # List certificates
POST   /api/v1/security/certificates           # Generate certificate
GET    /api/v1/security/threats                # Get threats
GET    /api/v1/security/compliance             # Compliance status

# Accounting Management
GET    /api/v1/accounting/records              # Get usage records
GET    /api/v1/accounting/billing              # Get billing info
GET    /api/v1/accounting/fraud                # Fraud alerts
```

### WebSocket API

```javascript
// Connection
ws://o1-interface:8080/stream

// Message format
{
  "type": "subscribe|unsubscribe|filter",
  "stream": "alarms|performance|configuration|events",
  "filter": "xpath or regex expression",
  "qos": "BestEffort|Reliable|Guaranteed"
}

// Response format
{
  "type": "data|error|ack",
  "stream": "stream-type",
  "timestamp": "ISO8601",
  "data": { /* stream-specific data */ }
}
```

## Best Practices

### Security

1. **Use TLS for NETCONF connections**
   - Configure minimum TLS version 1.2
   - Use strong cipher suites
   - Regularly rotate certificates

2. **Implement RBAC**
   - Define roles for different operations
   - Use least privilege principle
   - Audit access regularly

3. **Enable security monitoring**
   - Configure intrusion detection
   - Monitor failed authentication attempts
   - Set up security alerts

### Performance

1. **Optimize KPI collection**
   - Use appropriate collection intervals
   - Enable aggregation for historical data
   - Implement data retention policies

2. **Manage streaming connections**
   - Set connection limits
   - Implement rate limiting
   - Use appropriate QoS levels

3. **Resource management**
   - Configure resource limits
   - Enable horizontal scaling
   - Monitor resource usage

### Reliability

1. **Enable high availability**
   - Deploy multiple replicas
   - Configure failover timeouts
   - Test failover scenarios

2. **Implement backup strategies**
   - Regular configuration backups
   - Export critical data
   - Test restoration procedures

3. **Monitor system health**
   - Set up health checks
   - Configure alerting
   - Regular maintenance windows

## Migration Guide

### From Legacy O1 Implementation

1. **Export existing configurations**
2. **Map legacy YANG models to O-RAN models**
3. **Update credentials and connection parameters**
4. **Test with non-production elements first**
5. **Gradual rollout with monitoring**

### Version Upgrades

1. **Review release notes for breaking changes**
2. **Backup current configurations**
3. **Test in staging environment**
4. **Plan maintenance window**
5. **Execute upgrade with rollback plan**

## Support

### Documentation

- [O-RAN Alliance Specifications](https://www.o-ran.org/specifications)
- [NETCONF RFC 6241](https://tools.ietf.org/html/rfc6241)
- [YANG RFC 7950](https://tools.ietf.org/html/rfc7950)

### Community

- GitHub Issues: [nephoran-intent-operator/issues](https://github.com/thc1006/nephoran-intent-operator/issues)
- Slack Channel: #nephoran-o1-interface
- Mailing List: nephoran-dev@googlegroups.com

### Commercial Support

For enterprise support, contact: support@nephoran.io
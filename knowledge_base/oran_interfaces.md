# O-RAN Interfaces Specification for Nephoran Intent Operator

## Overview
O-RAN (Open Radio Access Network) Alliance defines standardized interfaces for disaggregated RAN architecture. This document covers A1, E2, O1, and O2 interfaces critical for the Nephoran Intent Operator's orchestration capabilities.

## A1 Interface - Policy and Enrichment

### Purpose
The A1 interface connects the Non-RT RIC (Non-Real-Time RAN Intelligent Controller) to the Near-RT RIC for policy-based management and AI/ML model deployment.

### Protocol Specification
- **Transport**: HTTP/2 with JSON payload
- **Architecture**: RESTful API
- **Port**: Typically 8080 or 9000
- **Authentication**: OAuth 2.0 with JWT tokens
- **Specification**: O-RAN.WG2.A1-v06.00

### A1 Policy Types

#### Policy Type: Traffic Steering (TS-2)
Used for intelligent traffic routing across cells and network slices.

```json
{
  "policy_type_id": "TS-2",
  "policy_type_version": "1.0.0",
  "description": "Traffic Steering policy for network slice optimization",
  "policy_schema": {
    "type": "object",
    "properties": {
      "slice_dnn": {
        "type": "string",
        "description": "Data Network Name for slice identification"
      },
      "priority_level": {
        "type": "integer",
        "minimum": 1,
        "maximum": 10,
        "description": "Priority level (1=lowest, 10=highest)"
      },
      "target_throughput_mbps": {
        "type": "integer",
        "description": "Target throughput in Mbps"
      },
      "latency_threshold_ms": {
        "type": "integer",
        "description": "Maximum acceptable latency in milliseconds"
      }
    },
    "required": ["slice_dnn", "priority_level"]
  }
}
```

#### Policy Type: QoS Management (QM-1)
Controls Quality of Service parameters for radio bearers.

```json
{
  "policy_type_id": "QM-1",
  "policy_type_version": "2.0.0",
  "description": "QoS management policy for radio resource allocation",
  "policy_schema": {
    "type": "object",
    "properties": {
      "5qi": {
        "type": "integer",
        "enum": [1, 2, 3, 4, 5, 6, 7, 8, 9],
        "description": "5G QoS Identifier"
      },
      "priority_level": {
        "type": "integer",
        "minimum": 1,
        "maximum": 127
      },
      "packet_delay_budget_ms": {
        "type": "integer",
        "description": "Packet delay budget in milliseconds"
      },
      "packet_error_rate": {
        "type": "number",
        "minimum": 0,
        "maximum": 1,
        "description": "Target packet error rate (e.g., 0.001 = 10^-3)"
      },
      "averaging_window_ms": {
        "type": "integer",
        "default": 2000
      }
    },
    "required": ["5qi", "priority_level"]
  }
}
```

#### Policy Type: Handover Optimization (HO-3)
Manages mobility and handover decisions.

```json
{
  "policy_type_id": "HO-3",
  "policy_type_version": "1.5.0",
  "description": "Handover optimization for seamless mobility",
  "policy_schema": {
    "type": "object",
    "properties": {
      "handover_trigger_rsrp_dbm": {
        "type": "integer",
        "description": "RSRP threshold in dBm to trigger handover"
      },
      "handover_hysteresis_db": {
        "type": "integer",
        "default": 3,
        "description": "Hysteresis value in dB"
      },
      "time_to_trigger_ms": {
        "type": "integer",
        "enum": [40, 64, 80, 100, 128, 160, 256, 320, 480, 512, 640],
        "description": "Time to trigger handover in milliseconds"
      },
      "target_cell_selection": {
        "type": "string",
        "enum": ["strongest_rsrp", "load_balancing", "ml_prediction"]
      }
    },
    "required": ["handover_trigger_rsrp_dbm", "time_to_trigger_ms"]
  }
}
```

### A1 API Endpoints

#### Create Policy Instance
```http
POST /a1-p/policytypes/{policy_type_id}/policies/{policy_instance_id}
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "slice_dnn": "enterprise_slice_1",
  "priority_level": 8,
  "target_throughput_mbps": 500,
  "latency_threshold_ms": 10
}
```

**Response**:
```json
{
  "policy_instance_id": "policy-12345",
  "policy_type_id": "TS-2",
  "status": "ACTIVE",
  "created_at": "2026-02-21T14:30:00Z",
  "enforcement_scope": ["cell-001", "cell-002", "cell-003"]
}
```

#### Query Policy Status
```http
GET /a1-p/policytypes/{policy_type_id}/policies/{policy_instance_id}/status
Authorization: Bearer {jwt_token}
```

**Response**:
```json
{
  "policy_instance_id": "policy-12345",
  "status": "ACTIVE",
  "enforcement_status": {
    "cell-001": "ENFORCED",
    "cell-002": "ENFORCED",
    "cell-003": "PENDING"
  },
  "last_updated": "2026-02-21T14:35:00Z"
}
```

#### Delete Policy Instance
```http
DELETE /a1-p/policytypes/{policy_type_id}/policies/{policy_instance_id}
Authorization: Bearer {jwt_token}
```

### NetworkIntent to A1 Policy Mapping

The Nephoran Intent Operator translates natural language intents to A1 policies:

**User Intent**: "Prioritize enterprise slice traffic with 500 Mbps throughput and < 10ms latency"

**Generated A1 Policy**:
```json
{
  "policy_type_id": "TS-2",
  "policy_instance_id": "nephoran-ts-20260221-143000",
  "policy_data": {
    "slice_dnn": "enterprise_slice_1",
    "priority_level": 9,
    "target_throughput_mbps": 500,
    "latency_threshold_ms": 10
  },
  "notification_destination": "http://nephoran-operator:8080/a1-notifications"
}
```

## E2 Interface - Near-RT Control and Monitoring

### Purpose
The E2 interface connects the Near-RT RIC to O-RAN E2 nodes (CU, DU, gNB) for real-time monitoring and control with millisecond-level latency requirements.

### Protocol Specification
- **Transport**: SCTP (Stream Control Transmission Protocol)
- **Encoding**: ASN.1 PER (Packed Encoding Rules)
- **Port**: 36421, 36422 (configurable)
- **Latency Requirement**: < 10ms (near-real-time)
- **Specification**: O-RAN.WG3.E2AP-v03.00

### E2 Service Models (E2SM)

#### E2SM-KPM (Key Performance Metrics)
Collects performance measurements from RAN nodes.

**Supported KPIs**:
- RRC Connection Success Rate
- Handover Success Rate
- Packet Loss Rate
- Throughput (UL/DL)
- PRB Utilization
- Active UE Count
- Average CQI (Channel Quality Indicator)

**Subscription Request**:
```json
{
  "ranFunctionID": 1,
  "ricActionType": "REPORT",
  "ricActionDefinition": {
    "styleType": "measurement",
    "measurementInfoList": [
      {
        "measurementTypeName": "DRB.PdcpSduVolumeDL",
        "measurementTypeID": 1
      },
      {
        "measurementTypeName": "RRU.PrbUsedDl",
        "measurementTypeID": 2
      }
    ]
  },
  "ricSubsequentAction": {
    "ricSubsequentActionType": "CONTINUE",
    "ricTimeToWait": "w10ms"
  }
}
```

#### E2SM-RC (RAN Control)
Enables RAN control actions (handover, power control, etc.).

**Control Message Example**:
```json
{
  "ranFunctionID": 2,
  "ricControlHeader": {
    "ueID": "imsi-208930000000001",
    "controlType": "HANDOVER_COMMAND"
  },
  "ricControlMessage": {
    "targetCell": "NRCellDU-002",
    "cause": "load_balancing",
    "priority": "high"
  },
  "ricControlAckRequest": "ACK"
}
```

#### E2SM-NI (Network Interface)
Manages inter-node interfaces (X2, Xn).

### E2 Message Types

1. **E2 Setup Request/Response**: Establish E2 connection
2. **RIC Subscription Request/Response**: Subscribe to RAN measurements
3. **RIC Indication**: Periodic or event-driven measurement reports
4. **RIC Control Request/Acknowledge**: Send control commands to RAN
5. **RIC Service Update**: Update available RAN functions

### E2 Termination in Nephoran

The Nephoran Intent Operator integrates with O-RAN SC E2 Termination component:

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: e2-kpm-subscription
spec:
  intent: "Subscribe to DL throughput and PRB utilization metrics from gNB-001"
  targetComponent: e2-termination
  parameters:
    ranFunctionID: 1
    e2NodeID: "gNB-001"
    measurements:
      - DRB.PdcpSduVolumeDL
      - RRU.PrbUsedDl
    reportingPeriod: 1000ms
    granularityPeriod: 100ms
```

## O1 Interface - Configuration Management

### Purpose
The O1 interface provides FCAPS (Fault, Configuration, Accounting, Performance, Security) management for O-RAN network elements using NETCONF/YANG protocols.

### Protocol Specification
- **Transport**: NETCONF over SSH (port 830) or TLS (port 6513)
- **Data Modeling**: YANG (RFC 7950)
- **Operations**: `<get>`, `<get-config>`, `<edit-config>`, `<copy-config>`, `<delete-config>`
- **Notifications**: NETCONF event notifications (RFC 5277)
- **Specification**: O-RAN.WG4.MP-v10.00

### YANG Models

#### O-RAN Hardware Management
```yang
module o-ran-hardware {
  namespace "urn:o-ran:hardware:1.0";
  prefix "o-ran-hw";

  container hardware {
    list component {
      key "name";
      leaf name {
        type string;
      }
      leaf class {
        type identityref {
          base ianahw:hardware-class;
        }
      }
      leaf physical-index {
        type int32;
      }
      container state {
        leaf admin-state {
          type enumeration {
            enum locked;
            enum unlocked;
            enum shutting-down;
          }
        }
        leaf oper-state {
          type enumeration {
            enum enabled;
            enum disabled;
          }
        }
        leaf availability-state {
          type enumeration {
            enum normal;
            enum degraded;
            enum faulty;
          }
        }
      }
    }
  }
}
```

#### O-RAN Performance Management
```yang
module o-ran-performance-management {
  namespace "urn:o-ran:performance-management:1.0";
  prefix "o-ran-pm";

  container performance-management {
    list measurement-job {
      key "job-id";
      leaf job-id {
        type string;
      }
      leaf administrative-state {
        type enumeration {
          enum locked;
          enum unlocked;
        }
      }
      leaf granularity-period {
        type uint32;
        units "seconds";
        description "Measurement collection period";
      }
      leaf-list measurement-types {
        type string;
        description "List of KPI measurement types";
      }
      container file-upload {
        leaf server-url {
          type inet:uri;
        }
        leaf username {
          type string;
        }
        leaf password {
          type string;
        }
      }
    }
  }
}
```

### O1 Configuration Examples

#### Configure UPF N3 Interface
```xml
<rpc message-id="101" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <upf-config xmlns="urn:o-ran:upf:1.0">
        <interface>
          <name>N3</name>
          <type>gtp-u</type>
          <ipv4-address>10.100.200.3</ipv4-address>
          <ipv4-subnet>10.100.200.0/24</ipv4-subnet>
          <mtu>1500</mtu>
          <admin-state>unlocked</admin-state>
        </interface>
        <interface>
          <name>N6</name>
          <type>data-network</type>
          <ipv4-address>192.168.100.1</ipv4-address>
          <ipv4-gateway>192.168.100.254</ipv4-gateway>
          <admin-state>unlocked</admin-state>
        </interface>
      </upf-config>
    </config>
  </edit-config>
</rpc>
```

#### Query Operational State
```xml
<rpc message-id="102" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <hardware xmlns="urn:o-ran:hardware:1.0">
        <component>
          <state/>
        </component>
      </hardware>
    </filter>
  </get>
</rpc>
```

**Response**:
```xml
<rpc-reply message-id="102" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <data>
    <hardware xmlns="urn:o-ran:hardware:1.0">
      <component>
        <name>RU-001</name>
        <class>o-ran-hw:O-RAN-RADIO-UNIT</class>
        <state>
          <admin-state>unlocked</admin-state>
          <oper-state>enabled</oper-state>
          <availability-state>normal</availability-state>
        </state>
      </component>
    </hardware>
  </data>
</rpc-reply>
```

### NetworkIntent to O1 Configuration

**User Intent**: "Configure UPF N3 interface with IP 10.0.0.1 and N6 interface with IP 192.168.1.1"

**Generated O1 Configuration**:
```xml
<config>
  <interface name="N3">
    <address>10.0.0.1</address>
    <subnet>10.0.0.0/24</subnet>
  </interface>
  <interface name="N6">
    <address>192.168.1.1</address>
    <gateway>192.168.1.254</gateway>
  </interface>
</config>
```

## O2 Interface - Cloud Infrastructure Management

### Purpose
The O2 interface manages cloud infrastructure and resources for O-RAN deployment using IMS (Infrastructure Management Service) aligned with O-RAN specifications.

### Protocol Specification
- **Transport**: RESTful HTTP/HTTPS
- **Port**: 30280 (configurable)
- **Authentication**: OAuth 2.0
- **Data Format**: JSON
- **Specification**: O-RAN.WG6.O2IMS-v01.00

### O2 Resource Types

#### Deployment Manager Server (DMS)
Manages O-Cloud resource pools and deployment lifecycle.

```json
{
  "deploymentManagerId": "dms-001",
  "name": "O-Cloud Primary DMS",
  "description": "Deployment manager for Kubernetes cluster thc1006-ubuntu-22",
  "supportedResourceTypes": [
    "compute",
    "network",
    "storage"
  ],
  "capabilities": {
    "kubernetes_version": "1.35.1",
    "cni": "cilium",
    "csi": "local-path-provisioner",
    "dra_enabled": true
  },
  "oCloudId": "ocloud-k8s-1",
  "serviceUri": "http://o2ims-service:30280"
}
```

#### Resource Pool
Represents available infrastructure resources.

```json
{
  "resourcePoolId": "pool-gpu-001",
  "name": "GPU Resource Pool",
  "description": "NVIDIA RTX 5080 GPU resources",
  "location": "datacenter-1",
  "resources": {
    "gpu": {
      "nvidia.com/gpu": 1,
      "nvidia.com/gpu-memory": "16Gi"
    },
    "cpu": {
      "total_cores": 16,
      "available_cores": 12
    },
    "memory": {
      "total_gi": 64,
      "available_gi": 48
    }
  },
  "capabilities": [
    "dra-gpu",
    "cuda-12.6"
  ]
}
```

### O2 API Endpoints

#### List Resource Pools
```http
GET /o2ims-infrastructureInventory/v1/resourcePools
Authorization: Bearer {jwt_token}
```

**Response**:
```json
{
  "resourcePools": [
    {
      "resourcePoolId": "pool-001",
      "name": "Compute Pool 1",
      "resources": {
        "cpu_cores": 64,
        "memory_gb": 256,
        "storage_gb": 1000
      }
    }
  ]
}
```

#### Create Deployment Request
```http
POST /o2ims-infrastructureInventory/v1/deploymentRequests
Content-Type: application/json
Authorization: Bearer {jwt_token}

{
  "deploymentRequestId": "deploy-free5gc-001",
  "resourceTypeId": "5g-core-upf",
  "resourcePoolId": "pool-001",
  "parameters": {
    "replicas": 3,
    "cpu_per_replica": "2000m",
    "memory_per_replica": "2Gi",
    "network_attachments": ["n3-network", "n6-network"]
  }
}
```

### Integration with Nephoran

The Nephoran Intent Operator queries O2 interface for resource availability before deploying network functions:

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: upf-deployment-with-resources
spec:
  intent: "Deploy UPF with GPU acceleration in pool-gpu-001"
  targetComponent: upf
  parameters:
    replicas: 2
    resourcePool: pool-gpu-001
    resources:
      requests:
        nvidia.com/gpu: 1
        cpu: 2000m
        memory: 2Gi
```

## Interface Integration Matrix

| Interface | Protocol | Port | Latency | Use Case | Nephoran Integration |
|-----------|----------|------|---------|----------|---------------------|
| **A1** | HTTP/2 JSON | 9000 | 100ms - 1s | Policy management | NetworkIntent → A1 Policy |
| **E2** | SCTP ASN.1 | 36421 | < 10ms | Real-time control | E2 metrics → Intent feedback |
| **O1** | NETCONF/YANG | 830 | 1s - 10s | Configuration mgmt | Intent → O1 config |
| **O2** | REST JSON | 30280 | 100ms - 1s | Infrastructure mgmt | Resource query + allocation |

## References
- **O-RAN.WG2.A1-v06.00**: A1 Interface Specification
- **O-RAN.WG3.E2AP-v03.00**: E2 Application Protocol
- **O-RAN.WG4.MP-v10.00**: O1 Management Plane Specification
- **O-RAN.WG6.O2IMS-v01.00**: O2 Infrastructure Management Service
- **3GPP TS 28.541**: Management and orchestration of networks and network slicing

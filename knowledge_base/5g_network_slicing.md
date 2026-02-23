# 5G Network Slicing for Nephoran Intent Operator

## Overview
Network slicing is a key feature of 5G that enables the creation of multiple virtual networks (slices) over a shared physical infrastructure. Each slice is optimized for specific use cases with unique performance, security, and functionality requirements.

## Network Slice Types

### eMBB (Enhanced Mobile Broadband)
- **Use Case**: High-speed internet, video streaming, AR/VR
- **Characteristics**:
  - High data rates: > 1 Gbps DL, > 500 Mbps UL
  - Moderate latency: 20-50ms
  - Moderate reliability: 99.9% uptime
- **S-NSSAI**: SST=1, SD=010203 (example)
- **DNN**: internet, video-stream
- **QoS Profile**: 5QI=9 (best-effort), 5QI=8 (premium)

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: embb-slice
spec:
  intent: "Create eMBB slice for high-speed internet with 1 Gbps throughput"
  sliceType: eMBB
  parameters:
    sst: 1
    sd: "010203"
    dnn: internet
    throughput:
      downlink: 1000Mbps
      uplink: 500Mbps
    latency: 50ms
    reliability: 99.9
    targetUEs: 10000
```

### URLLC (Ultra-Reliable Low-Latency Communications)
- **Use Case**: Industrial automation, autonomous vehicles, remote surgery
- **Characteristics**:
  - Ultra-low latency: 1-5ms
  - Ultra-high reliability: 99.999% uptime
  - Moderate data rates: 1-100 Mbps
- **S-NSSAI**: SST=2, SD=112233 (example)
- **DNN**: urllc, industrial, v2x
- **QoS Profile**: 5QI=1 (GBR), 5QI=2 (GBR)

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: urllc-slice
spec:
  intent: "Deploy URLLC slice for industrial automation with < 5ms latency"
  sliceType: URLLC
  parameters:
    sst: 2
    sd: "112233"
    dnn: industrial
    latency: 5ms
    reliability: 99.999
    packetErrorRate: 0.00001
    dedicatedUPF: true
    priorityLevel: 1  # Highest priority
```

### mMTC (Massive Machine-Type Communications)
- **Use Case**: IoT sensors, smart cities, agriculture monitoring
- **Characteristics**:
  - Massive device density: 1M devices/km²
  - Low data rates: 1-100 kbps
  - Extended coverage: > 10km
  - Low power consumption: 10+ years battery life
- **S-NSSAI**: SST=3, SD=334455 (example)
- **DNN**: iot, sensors
- **QoS Profile**: 5QI=9 (non-GBR)

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: mmtc-slice
spec:
  intent: "Create mMTC slice for 100,000 IoT sensors with extended coverage"
  sliceType: mMTC
  parameters:
    sst: 3
    sd: "334455"
    dnn: iot
    deviceCount: 100000
    dataBudget: 100kbps-per-device
    coverage: extended
    powerSaving: true
```

## S-NSSAI (Single Network Slice Selection Assistance Information)

### Structure
- **SST (Slice/Service Type)**: 8-bit field
  - 1: eMBB
  - 2: URLLC
  - 3: mMTC
  - 4: V2X (Vehicle-to-Everything)
  - 128-255: Operator-specific
- **SD (Slice Differentiator)**: 24-bit field (optional)
  - Used to differentiate between multiple slices of the same SST
  - Example: "010203" (hex) = slice variant 1

### Examples
```json
{
  "slices": [
    {
      "s_nssai": {
        "sst": 1,
        "sd": "010203"
      },
      "description": "eMBB for consumer internet",
      "dnn": "internet"
    },
    {
      "s_nssai": {
        "sst": 1,
        "sd": "010204"
      },
      "description": "eMBB for enterprise video conferencing",
      "dnn": "enterprise"
    },
    {
      "s_nssai": {
        "sst": 2,
        "sd": "112233"
      },
      "description": "URLLC for factory automation",
      "dnn": "industrial"
    }
  ]
}
```

## DNN (Data Network Name)

### Purpose
DNN identifies the external data network (DN) that the UE wants to access. It is similar to APN (Access Point Name) in 4G.

### Common DNN Values
- **internet**: Public internet access
- **ims**: IP Multimedia Subsystem (voice/video calls)
- **enterprise**: Private enterprise network
- **industrial**: Industrial IoT network
- **iot**: Consumer IoT network
- **v2x**: Vehicle-to-Everything communication

### DNN Configuration in SMF

```yaml
dnns:
  - dnn: internet
    dns:
      ipv4: 8.8.8.8
      ipv6: 2001:4860:4860::8888
    ue_ip_pool: 10.60.0.0/16
    upf_selection:
      - upf-edge-1
      - upf-edge-2
  - dnn: enterprise
    dns:
      ipv4: 192.168.1.1
    ue_ip_pool: 10.70.0.0/16
    upf_selection:
      - upf-enterprise
    mtu: 1500
    session_ambr:
      uplink: 500Mbps
      downlink: 1000Mbps
```

## QoS Flow Management

### 5QI (5G QoS Identifier)

| 5QI | Resource Type | Priority | Packet Delay Budget | Packet Error Rate | Use Case |
|-----|---------------|----------|---------------------|-------------------|----------|
| 1 | GBR | 20 | 100ms | 10⁻² | Conversational Voice |
| 2 | GBR | 40 | 150ms | 10⁻³ | Conversational Video |
| 3 | GBR | 30 | 50ms | 10⁻³ | Real-time Gaming |
| 4 | GBR | 50 | 300ms | 10⁻⁶ | Non-conversational Video |
| 5 | GBR | 10 | 100ms | 10⁻⁶ | IMS Signaling |
| 6 | Non-GBR | 60 | 300ms | 10⁻⁶ | Video (buffered) |
| 7 | Non-GBR | 70 | 100ms | 10⁻³ | Voice, Video, Gaming |
| 8 | Non-GBR | 80 | 300ms | 10⁻⁶ | Video (buffered) |
| 9 | Non-GBR | 90 | 300ms | 10⁻⁶ | Default bearer |

### GBR (Guaranteed Bit Rate) vs Non-GBR

#### GBR Flow
- **Guaranteed bandwidth**: Network reserves specific bandwidth
- **Admission control**: Flow rejected if resources unavailable
- **Use case**: Voice calls, live video streaming
- **Configuration**:
  ```json
  {
    "qos_flow": {
      "5qi": 1,
      "gbr_uplink": "100kbps",
      "gbr_downlink": "100kbps",
      "max_fbr_uplink": "150kbps",
      "max_fbr_downlink": "150kbps"
    }
  }
  ```

#### Non-GBR Flow
- **Best-effort delivery**: No guaranteed bandwidth
- **Shared resources**: Uses available capacity
- **Use case**: Web browsing, file download
- **Configuration**:
  ```json
  {
    "qos_flow": {
      "5qi": 9,
      "priority_level": 127,
      "averaging_window": 2000
    }
  }
  ```

### QoS Flow Configuration Example

```yaml
pdu_sessions:
  - session_id: 1
    dnn: internet
    s_nssai:
      sst: 1
      sd: "010203"
    qos_flows:
      - qfi: 1  # QoS Flow Identifier
        5qi: 9
        priority_level: 127
        arp:  # Allocation and Retention Priority
          priority_level: 15
          pre_emption_capability: NOT_PRE_EMPT
          pre_emption_vulnerability: PRE_EMPTABLE
      - qfi: 2
        5qi: 5  # IMS signaling
        priority_level: 10
        gbr_uplink: 64kbps
        gbr_downlink: 64kbps
```

## Network Slice Deployment via Nephoran

### Slice Creation Workflow

1. **User Intent**: "Create enterprise slice with 500 Mbps guaranteed throughput and 10ms latency"

2. **LLM Processing**: Ollama processes intent with RAG context from knowledge base

3. **NetworkIntent Generation**:
   ```yaml
   apiVersion: intent.nephoran.com/v1
   kind: NetworkIntent
   metadata:
     name: enterprise-slice-001
   spec:
     intent: "Create enterprise slice with 500 Mbps guaranteed throughput and 10ms latency"
     sliceType: eMBB
     parameters:
       sst: 1
       sd: "010204"
       dnn: enterprise
       qos:
         5qi: 3
         gbrDownlink: 500Mbps
         gbrUplink: 250Mbps
       latency: 10ms
       reliability: 99.99
       dedicatedResources:
         upf:
           replicas: 2
           cpu: 4000m
           memory: 4Gi
   ```

4. **A1 Policy Generation**:
   ```json
   {
     "policy_type_id": "SLICE-1",
     "policy_instance_id": "enterprise-slice-001",
     "policy_data": {
       "slice_id": {
         "sst": 1,
         "sd": "010204"
       },
       "dnn": "enterprise",
       "resource_allocation": {
         "upf_instances": 2,
         "cpu_per_upf": "4000m",
         "memory_per_upf": "4Gi"
       },
       "qos_requirements": {
         "5qi": 3,
         "priority_level": 30,
         "gbr_dl": 500000000,
         "gbr_ul": 250000000,
         "pdb_ms": 10
       }
     }
   }
   ```

5. **Infrastructure Provisioning**:
   - Create dedicated UPF instances
   - Configure SMF with DNN and S-NSSAI
   - Update AMF with slice support
   - Configure PCF with QoS policies

### Multi-Slice Configuration Example

```yaml
apiVersion: intent.nephoran.com/v1
kind: NetworkSlice
metadata:
  name: multi-tenant-slices
  namespace: free5gc
spec:
  slices:
    - name: consumer-internet
      sst: 1
      sd: "010203"
      dnn: internet
      capacity:
        maxUEs: 50000
        throughputDL: 10Gbps
        throughputUL: 5Gbps
      upf:
        replicas: 3
        image: free5gc/upf:v3.4.3
        resources:
          cpu: 2000m
          memory: 2Gi

    - name: enterprise-private
      sst: 1
      sd: "010204"
      dnn: enterprise
      capacity:
        maxUEs: 1000
        throughputDL: 2Gbps
        throughputUL: 1Gbps
      upf:
        replicas: 2
        image: free5gc/upf:v3.4.3
        resources:
          cpu: 4000m
          memory: 4Gi
        affinity:
          nodeAffinity:
            requiredDuringSchedulingIgnoredDuringExecution:
              nodeSelectorTerms:
              - matchExpressions:
                - key: zone
                  operator: In
                  values:
                  - enterprise-edge

    - name: iot-sensors
      sst: 3
      sd: "334455"
      dnn: iot
      capacity:
        maxUEs: 100000
        dataBudgetPerDevice: 1kbps
      upf:
        replicas: 1
        image: free5gc/upf:v3.4.3
        resources:
          cpu: 1000m
          memory: 1Gi
```

## Slice Isolation and Security

### Network Isolation

#### Per-Slice UPF Instances
Deploy dedicated UPF instances per slice to ensure data plane isolation:

```yaml
# UPF for enterprise slice
apiVersion: apps/v1
kind: Deployment
metadata:
  name: upf-enterprise
  namespace: free5gc
  labels:
    slice: enterprise
    sst: "1"
    sd: "010204"
spec:
  replicas: 2
  selector:
    matchLabels:
      app: upf
      slice: enterprise
  template:
    metadata:
      labels:
        app: upf
        slice: enterprise
    spec:
      containers:
      - name: upf
        image: free5gc/upf:v3.4.3
        env:
        - name: DNN
          value: "enterprise"
        - name: S_NSSAI_SST
          value: "1"
        - name: S_NSSAI_SD
          value: "010204"
```

#### Network Policy for Slice Isolation

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: enterprise-slice-isolation
  namespace: free5gc
spec:
  podSelector:
    matchLabels:
      slice: enterprise
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only allow traffic from SMF
  - from:
    - podSelector:
        matchLabels:
          app: smf
    ports:
    - protocol: UDP
      port: 8805  # PFCP
  egress:
  # Only allow traffic to enterprise data network
  - to:
    - podSelector:
        matchLabels:
          network: enterprise
```

### Resource Isolation

#### Resource Quota per Slice

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: enterprise-slice-quota
  namespace: free5gc
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
  scopeSelector:
    matchExpressions:
    - operator: In
      scopeName: PriorityClass
      values:
      - enterprise-slice
```

#### PriorityClass for Slice QoS

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: enterprise-slice
value: 1000000  # High priority
globalDefault: false
description: "Enterprise slice with guaranteed resources"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: consumer-slice
value: 100000  # Lower priority
globalDefault: false
description: "Consumer slice with best-effort resources"
```

## Monitoring and Analytics

### Slice-Specific Metrics

#### Prometheus Metrics

```
# Throughput per slice
upf_throughput_bytes_total{slice="enterprise", dnn="enterprise"} 524288000

# Active UEs per slice
amf_active_ues{sst="1", sd="010204"} 450

# PDU sessions per slice
smf_active_sessions{sst="1", sd="010204", dnn="enterprise"} 780

# QoS flow metrics
smf_qos_flow_violations{5qi="3", slice="enterprise"} 12

# Resource utilization per slice
upf_cpu_usage_percent{slice="enterprise"} 65.3
upf_memory_usage_percent{slice="enterprise"} 72.1
```

#### Grafana Dashboard Query

```promql
# Average latency per slice
histogram_quantile(0.95,
  rate(upf_packet_latency_seconds_bucket{slice="enterprise"}[5m])
)

# Throughput per slice (last 5 min)
rate(upf_throughput_bytes_total{slice="enterprise"}[5m])

# Session success rate per slice
(
  rate(smf_pdu_session_success_total{slice="enterprise"}[5m])
  /
  rate(smf_pdu_session_attempts_total{slice="enterprise"}[5m])
) * 100
```

## Troubleshooting Slice Issues

### Common Problems

#### Slice Selection Failure
- **Symptom**: UE registration succeeds but slice selection fails
- **Diagnosis**: Check AMF logs for NSSAI negotiation
  ```bash
  kubectl logs -n free5gc deployment/amf | grep -i nssai
  ```
- **Solution**: Verify S-NSSAI configured in AMF matches UE request

#### QoS Flow Establishment Failure
- **Symptom**: PDU session established but QoS flow creation fails
- **Diagnosis**: Check SMF logs for QoS validation
  ```bash
  kubectl logs -n free5gc deployment/smf | grep -E "5QI|QoS"
  ```
- **Solution**: Verify PCF policies allow requested 5QI and GBR values

#### UPF PFCP Association Failure for Specific Slice
- **Symptom**: UPF cannot associate with SMF for specific DNN
- **Diagnosis**: Check UPF configuration for DNN support
- **Solution**: Update UPF configuration to include DNN and S-NSSAI

### Verification Commands

```bash
# Check slice configuration in AMF
kubectl exec -n free5gc deployment/amf -- cat /free5gc/config/amfcfg.yaml | grep -A 10 plmnSupportList

# Verify UPF instances per slice
kubectl get pods -n free5gc -l app=upf -o custom-columns=NAME:.metadata.name,SLICE:.metadata.labels.slice

# Check SMF DNN configuration
kubectl exec -n free5gc deployment/smf -- cat /free5gc/config/smfcfg.yaml | grep -A 20 "userplane_information"

# Monitor slice metrics
kubectl exec -n free5gc deployment/prometheus -- curl -s http://localhost:9090/api/v1/query?query=upf_throughput_bytes_total
```

## References
- **3GPP TS 23.501**: System Architecture for 5G (Section 5.15: Network Slicing)
- **3GPP TS 28.541**: Management of Network Slicing
- **3GPP TS 23.503**: Policy and Charging Control Framework
- **O-RAN.WG2**: AI/ML for Network Slicing Optimization
- **GSMA NG.116**: Generic Network Slice Template

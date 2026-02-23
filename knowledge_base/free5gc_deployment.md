# Free5GC Deployment Guide for Nephoran Intent Operator

## Overview
Free5GC is an open-source 5G Core Network implementation that provides production-ready network functions following 3GPP Release 16 specifications. This document provides deployment guidance for integrating Free5GC with the Nephoran Intent Operator.

## Network Functions Architecture

### Control Plane Network Functions

#### AMF (Access and Mobility Management Function)
- **Purpose**: Handles UE registration, mobility management, and connection management
- **Interfaces**: N1 (UE), N2 (RAN), N8 (UDM), N11 (SMF), N12 (AUSF), N14 (AMF)
- **Default Configuration**:
  - Container Image: `free5gc/amf:v3.4.3`
  - Replicas: 3 (for high availability)
  - Port: 8000 (SBI), 38412 (NGAP/SCTP)
  - Resource Limits: CPU 500m, Memory 512Mi
  - PLMN: MCC=208, MNC=93
- **NetworkIntent Example**:
  ```yaml
  apiVersion: intent.nephoran.com/v1
  kind: NetworkIntent
  metadata:
    name: amf-deployment
  spec:
    intent: "Deploy AMF with 3 replicas for high availability"
    targetComponent: amf
    parameters:
      replicas: 3
      image: "registry.example.com/5g/amf:1.2.3"
      highAvailability: true
  ```

#### SMF (Session Management Function)
- **Purpose**: Session establishment, modification, and release; UPF selection
- **Interfaces**: N4 (UPF), N7 (PCF), N10 (UDM), N11 (AMF)
- **Default Configuration**:
  - Container Image: `free5gc/smf:v3.4.3`
  - Replicas: 2
  - Port: 8000 (SBI), 8805 (PFCP/UDP)
  - DNN Support: internet, ims, enterprise
- **Session Management**:
  - PDU Session Capacity: 1000+ concurrent sessions per instance
  - Session Timeout: 3600 seconds (configurable)
  - QoS Flow Management: 5QI profiles (1, 5, 7, 9)

#### UDM (Unified Data Management)
- **Purpose**: Subscriber authentication, authorization data management
- **Interfaces**: N8 (AMF), N10 (SMF), N13 (AUSF)
- **Default Configuration**:
  - Container Image: `free5gc/udm:v3.4.3`
  - Replicas: 2
  - Port: 8000 (SBI)
  - Backend: MongoDB 8.0+

#### AUSF (Authentication Server Function)
- **Purpose**: 5G-AKA authentication, EAP-AKA' authentication
- **Interfaces**: N12 (AMF), N13 (UDM)
- **Default Configuration**:
  - Container Image: `free5gc/ausf:v3.4.3`
  - Replicas: 2
  - Port: 8000 (SBI)

#### NRF (Network Repository Function)
- **Purpose**: Service discovery, NF registration, subscription management
- **Interfaces**: SBI (all NFs)
- **Default Configuration**:
  - Container Image: `free5gc/nrf:v3.4.3`
  - Replicas: 2 (active-standby)
  - Port: 8000 (SBI)
  - Database: MongoDB for NF profiles

#### PCF (Policy Control Function)
- **Purpose**: Policy decision, charging control, QoS management
- **Interfaces**: N7 (SMF), N15 (AMF)
- **Default Configuration**:
  - Container Image: `free5gc/pcf:v3.4.3`
  - Replicas: 2
  - Port: 8000 (SBI)

### User Plane Network Functions

#### UPF (User Plane Function)
- **Purpose**: Packet routing, forwarding, QoS enforcement, traffic reporting
- **Interfaces**: N3 (gNB), N4 (SMF), N6 (Data Network), N9 (UPF)
- **Default Configuration**:
  - Container Image: `free5gc/upf:v3.4.3`
  - Replicas: 2-3 (based on traffic load)
  - Port: 8805 (PFCP/UDP), 2152 (GTP-U/UDP)
  - Resource Limits: CPU 2000m, Memory 2Gi
- **O1 Configuration Example**:
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
    <dnn name="internet">
      <ipPool>10.60.0.0/16</ipPool>
    </dnn>
  </config>
  ```
- **NetworkIntent Scaling Example**:
  ```yaml
  apiVersion: intent.nephoran.com/v1
  kind: NetworkIntent
  metadata:
    name: upf-scaling
  spec:
    intent: "Scale UPF to 3 replicas for increased throughput"
    targetComponent: upf
    parameters:
      replicas: 3
      image: "registry.example.com/5g/upf:4.5.6"
      autoScaling:
        enabled: true
        minReplicas: 2
        maxReplicas: 5
        targetCPU: 70
  ```

## Deployment Prerequisites

### Infrastructure Requirements
- **Kubernetes**: 1.32.3+ (1.35.1 recommended for DRA support)
- **CNI**: Cilium 1.14+ with eBPF datapath (10-20 Gbps throughput)
- **Storage**: Persistent volumes for MongoDB (10Gi minimum)
- **Networking**: Multus CNI for N3/N6 interfaces (optional but recommended)

### Database Requirements
- **MongoDB**: 8.0+ (required for NRF, UDM, PCF, AUSF)
  - Deployment: StatefulSet with 3 replicas (replica set)
  - Storage: 10Gi PVC per replica
  - Resource Limits: CPU 1000m, Memory 2Gi
  - Connection String: `mongodb://mongodb-0.mongodb:27017,mongodb-1.mongodb:27017,mongodb-2.mongodb:27017/free5gc?replicaSet=rs0`

### Namespace Configuration
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: free5gc
  labels:
    name: free5gc
    component: 5g-core
    managed-by: nephoran-intent-operator
```

## Deployment Sequence

### Phase 1: Database Layer
1. Deploy MongoDB StatefulSet
2. Initialize replica set
3. Create free5gc database and collections
4. Verify connectivity

### Phase 2: Core Services
1. Deploy NRF (service discovery foundation)
2. Deploy UDM + AUSF (authentication services)
3. Deploy PCF (policy control)
4. Verify all NFs register with NRF

### Phase 3: Control Plane
1. Deploy AMF (access management)
2. Deploy SMF (session management)
3. Verify N1/N2 interfaces ready

### Phase 4: User Plane
1. Deploy UPF instances (2-3 replicas)
2. Configure N3/N4/N6 interfaces
3. Verify PFCP association with SMF

## Kubernetes Manifests

### NRF Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrf
  namespace: free5gc
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nrf
  template:
    metadata:
      labels:
        app: nrf
        nf-type: nrf
    spec:
      containers:
      - name: nrf
        image: free5gc/nrf:v3.4.3
        ports:
        - containerPort: 8000
          name: sbi
        env:
        - name: GIN_MODE
          value: release
        - name: DB_URI
          value: mongodb://mongodb:27017
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /nnrf-nfm/v1/nf-instances
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: nrf
  namespace: free5gc
spec:
  selector:
    app: nrf
  ports:
  - port: 8000
    targetPort: 8000
    name: sbi
  type: ClusterIP
```

### AMF Deployment with High Availability
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf
  namespace: free5gc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: amf
  template:
    metadata:
      labels:
        app: amf
        nf-type: amf
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - amf
              topologyKey: kubernetes.io/hostname
      containers:
      - name: amf
        image: free5gc/amf:v3.4.3
        ports:
        - containerPort: 8000
          name: sbi
        - containerPort: 38412
          name: ngap
          protocol: SCTP
        env:
        - name: NRF_URI
          value: http://nrf:8000
        - name: PLMN_MCC
          value: "208"
        - name: PLMN_MNC
          value: "93"
        resources:
          requests:
            cpu: 300m
            memory: 384Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /namf-comm/v1/health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
```

## Integration with Nephoran Intent Operator

### NetworkIntent Processing Flow
1. User submits natural language intent: "Deploy Free5GC AMF with high availability"
2. LLM processes intent via RAG pipeline (Ollama + Weaviate)
3. Intent translated to NetworkIntent CRD
4. Nephoran controller watches NetworkIntent resources
5. Controller generates Kubernetes manifests (Deployment, Service, ConfigMap)
6. Manifests applied via Porch package orchestration
7. Status updates reflected in NetworkIntent CR status field

### A1 Policy Integration
Free5GC network functions can be controlled via O-RAN A1 interface policies:

```json
{
  "policy_type_id": "5GC-SMF-1",
  "policy_data": {
    "target_nf": "smf",
    "action": "update_session_policy",
    "parameters": {
      "default_qos": "5QI_9",
      "session_ambr_ul": "100Mbps",
      "session_ambr_dl": "500Mbps"
    }
  }
}
```

### O1 Interface Management
Network configuration management via NETCONF/YANG models:

```xml
<rpc message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <free5gc xmlns="urn:free5gc:config">
        <upf>
          <name>upf-edge-1</name>
          <n3-interface>
            <address>10.100.200.3</address>
          </n3-interface>
          <n6-interface>
            <address>192.168.100.1</address>
          </n6-interface>
        </upf>
      </free5gc>
    </config>
  </edit-config>
</rpc>
```

## Performance Tuning

### CPU and Memory Optimization
- **AMF**: 500m CPU for 1000 UEs, add 100m per additional 500 UEs
- **SMF**: 500m CPU for 500 PDU sessions, linear scaling
- **UPF**: 2000m CPU for 1 Gbps throughput, 4000m for 5 Gbps

### Network Performance (Cilium eBPF)
- **Expected Throughput**: 10-20 Gbps (virtual environment)
- **Latency**: < 10ms (pod-to-pod), < 50ms (E2E control plane)
- **Connection Rate**: 10,000+ sessions/second (SMF+UPF)

### Database Optimization (MongoDB)
- Enable connection pooling: maxPoolSize=50
- Use WiredTiger storage engine with compression
- Create indexes on frequently queried fields (subscriber IMSI, session ID)

## Troubleshooting

### Common Issues

#### NF Registration Failures
- **Symptom**: NFs unable to register with NRF
- **Diagnosis**: Check NRF logs: `kubectl logs -n free5gc deployment/nrf`
- **Solution**: Verify MongoDB connectivity, check NRF_URI environment variable

#### PFCP Association Failures (SMF-UPF)
- **Symptom**: SMF cannot establish PFCP association with UPF
- **Diagnosis**: Check SMF/UPF logs for PFCP heartbeat messages
- **Solution**: Verify UDP port 8805 connectivity, check firewall rules

#### PDU Session Establishment Failures
- **Symptom**: UE registration succeeds but PDU session fails
- **Diagnosis**: Check SMF logs for NAS message processing
- **Solution**: Verify UPF N3/N6 interface configuration, check DNN configuration in SMF

### Verification Commands
```bash
# Check all Free5GC pods
kubectl get pods -n free5gc

# Check NF registration status
kubectl exec -n free5gc deployment/nrf -- curl -s http://localhost:8000/nnrf-nfm/v1/nf-instances

# Check MongoDB connectivity
kubectl exec -n free5gc mongodb-0 -- mongosh --eval "rs.status()"

# Check UPF PFCP association
kubectl exec -n free5gc deployment/smf -- cat /tmp/smf.log | grep PFCP

# Monitor real-time logs
kubectl logs -n free5gc -f deployment/amf
```

## References
- **3GPP TS 23.501**: System Architecture for the 5G System
- **3GPP TS 23.502**: Procedures for the 5G System
- **3GPP TS 29.500**: 5G System Technical Realization
- **Free5GC Documentation**: https://free5gc.org/
- **Nephio Free5GC Packages**: https://github.com/nephio-project/catalog/tree/main/free5gc

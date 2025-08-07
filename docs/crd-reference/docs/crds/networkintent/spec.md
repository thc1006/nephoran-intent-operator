# NetworkIntent Specification Reference

## Full Specification Structure

```yaml
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: <string>              # Required: Resource name
  namespace: <string>          # Required: Namespace
  labels:                      # Optional: Labels
    <key>: <value>
  annotations:                 # Optional: Annotations
    <key>: <value>
spec:
  intent: <string>             # Required: Natural language intent
  intentType: <string>         # Required: Type of operation
  priority: <string>           # Optional: Processing priority
  targetComponents: []         # Optional: Target network functions
  resourceConstraints:         # Optional: Resource requirements
    cpu: <string>
    memory: <string>
    storage: <string>
    maxCpu: <string>
    maxMemory: <string>
  parameters: <object>         # Optional: Raw parameters
  parametersMap:               # Optional: String parameters
    <key>: <value>
  processedParameters:         # Optional: Structured parameters
    networkFunction: <string>
    deploymentConfig: <object>
    performanceRequirements: <object>
    scalingPolicy: <object>
    securityPolicy: <object>
  targetNamespace: <string>    # Optional: Deployment namespace
  targetCluster: <string>      # Optional: Target cluster
  networkSlice: <string>       # Optional: Network slice ID
  region: <string>             # Optional: Deployment region
  timeoutSeconds: <integer>    # Optional: Processing timeout
  maxRetries: <integer>        # Optional: Retry attempts
```

## Field Reference

### spec.intent

**Type:** `string`  
**Required:** Yes  
**Validation:**
- MinLength: 10
- MaxLength: 2000
- Must contain telecommunications keywords

**Description:**  
The natural language description of the desired network operation. This field is processed by the LLM to extract structured parameters.

**Examples:**
```yaml
intent: "Deploy a production-ready AMF with high availability and auto-scaling"
intent: "Scale the UPF to handle 50000 concurrent sessions"
intent: "Optimize the network slice for low latency URLLC traffic"
```

**Best Practices:**
- Use clear, specific language
- Include component names explicitly
- Specify requirements (HA, scaling, performance)
- Mention environment (production, staging, test)

---

### spec.intentType

**Type:** `string` (enum)  
**Required:** Yes  
**Default:** `"deployment"`  
**Allowed Values:**
- `deployment` - Deploy new network functions
- `scaling` - Scale existing deployments
- `optimization` - Optimize performance
- `maintenance` - Perform maintenance tasks

**Description:**  
Categorizes the intent for appropriate processing pipeline selection.

**Examples:**
```yaml
intentType: deployment    # For new deployments
intentType: scaling       # For capacity adjustments
intentType: optimization  # For performance tuning
intentType: maintenance   # For updates/patches
```

---

### spec.priority

**Type:** `string` (enum)  
**Required:** No  
**Default:** `"medium"`  
**Allowed Values:**
- `low` - Batch processing
- `medium` - Standard processing
- `high` - Expedited processing
- `critical` - Emergency operations

**Description:**  
Determines the processing order and resource allocation for the intent.

**Restrictions:**
- `critical` priority only allowed for `maintenance` intentType
- Higher priority intents preempt lower priority ones

**Examples:**
```yaml
priority: high      # Production deployments
priority: critical  # Emergency fixes
priority: low       # Test environments
```

---

### spec.targetComponents

**Type:** `[]string` (enum array)  
**Required:** No  
**Validation:**
- MinItems: 1 (if specified)
- MaxItems: 20
- No duplicates allowed

**Allowed Values:**

#### 5G Core Network Functions
| Component | Description |
|-----------|-------------|
| `AMF` | Access and Mobility Management Function |
| `SMF` | Session Management Function |
| `UPF` | User Plane Function |
| `NRF` | Network Repository Function |
| `AUSF` | Authentication Server Function |
| `UDM` | Unified Data Management |
| `PCF` | Policy Control Function |
| `NSSF` | Network Slice Selection Function |
| `NEF` | Network Exposure Function |
| `SMSF` | SMS Function |
| `BSF` | Binding Support Function |
| `UDR` | Unified Data Repository |
| `UDSF` | Unstructured Data Storage Function |
| `CHF` | Charging Function |
| `NWDAF` | Network Data Analytics Function |
| `SCP` | Service Communication Proxy |
| `SEPP` | Security Edge Protection Proxy |

#### RAN Components
| Component | Description |
|-----------|-------------|
| `gNodeB` | 5G Base Station |
| `O-DU` | O-RAN Distributed Unit |
| `O-CU-CP` | O-RAN Centralized Unit Control Plane |
| `O-CU-UP` | O-RAN Centralized Unit User Plane |
| `Near-RT-RIC` | Near Real-Time RAN Intelligent Controller |
| `Non-RT-RIC` | Non Real-Time RAN Intelligent Controller |
| `O-eNB` | O-RAN evolved NodeB |
| `SMO` | Service Management and Orchestration |
| `rApp` | Non-RT RIC Application |
| `xApp` | Near-RT RIC Application |

**Examples:**
```yaml
targetComponents:
  - AMF
  - SMF
  - UPF
```

---

### spec.resourceConstraints

**Type:** `object`  
**Required:** No  

**Fields:**

#### resourceConstraints.cpu
**Type:** `string` (Quantity)  
**Pattern:** `^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`  
**Description:** CPU request (cores or millicores)

#### resourceConstraints.memory
**Type:** `string` (Quantity)  
**Pattern:** `^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`  
**Description:** Memory request

#### resourceConstraints.storage
**Type:** `string` (Quantity)  
**Pattern:** `^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`  
**Description:** Storage request

#### resourceConstraints.maxCpu
**Type:** `string` (Quantity)  
**Description:** Maximum CPU limit

#### resourceConstraints.maxMemory
**Type:** `string` (Quantity)  
**Description:** Maximum memory limit

**Validation Rules:**
- maxCpu must be >= cpu
- maxMemory must be >= memory
- Critical priority requires minimum 500m CPU and 1Gi memory

**Examples:**
```yaml
resourceConstraints:
  cpu: "2"          # 2 cores
  memory: "4Gi"     # 4 GiB
  storage: "10Gi"   # 10 GiB
  maxCpu: "4"       # Max 4 cores
  maxMemory: "8Gi"  # Max 8 GiB
```

---

### spec.parameters

**Type:** `runtime.RawExtension`  
**Required:** No  
**Description:** Raw JSON/YAML parameters from LLM processing

**Note:** This field preserves unknown fields and is typically populated by the controller.

---

### spec.parametersMap

**Type:** `map[string]string`  
**Required:** No  
**Description:** Simple key-value parameters for testing and simple configurations

**Examples:**
```yaml
parametersMap:
  replicas: "3"
  version: "v2.1.0"
  environment: "production"
```

---

### spec.processedParameters

**Type:** `object`  
**Required:** No  
**Description:** Structured parameters after LLM processing

**Fields:**

#### processedParameters.networkFunction
**Type:** `string`  
**Description:** Target network function type

#### processedParameters.deploymentConfig
**Type:** `runtime.RawExtension`  
**Description:** Deployment-specific configuration

#### processedParameters.performanceRequirements
**Type:** `runtime.RawExtension`  
**Description:** Performance and QoS requirements

#### processedParameters.scalingPolicy
**Type:** `runtime.RawExtension`  
**Description:** Auto-scaling behavior configuration

#### processedParameters.securityPolicy
**Type:** `runtime.RawExtension`  
**Description:** Security requirements and policies

---

### spec.targetNamespace

**Type:** `string`  
**Required:** No  
**Pattern:** `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`  
**MaxLength:** 63  
**Description:** Kubernetes namespace for deployment

**Examples:**
```yaml
targetNamespace: "5g-core-prod"
targetNamespace: "oran-edge-1"
```

---

### spec.targetCluster

**Type:** `string`  
**Required:** No  
**Pattern:** `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`  
**MaxLength:** 253  
**Description:** Target Kubernetes cluster identifier

**Examples:**
```yaml
targetCluster: "central-cluster"
targetCluster: "edge-site-west-1"
```

---

### spec.networkSlice

**Type:** `string`  
**Required:** No  
**Pattern:** `^[0-9A-Fa-f]{6}-[0-9A-Fa-f]{6}$`  
**Description:** Network slice identifier (S-NSSAI format)

**Format:** `XXXXXX-YYYYYY` where:
- XXXXXX: Slice/Service Type (SST)
- YYYYYY: Slice Differentiator (SD)

**Examples:**
```yaml
networkSlice: "000001-000001"  # eMBB slice
networkSlice: "000002-000100"  # URLLC slice
networkSlice: "000003-ABC123"  # Custom slice
```

---

### spec.region

**Type:** `string`  
**Required:** No  
**Pattern:** `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`  
**MaxLength:** 64  
**Description:** Deployment region for geo-distributed deployments

**Examples:**
```yaml
region: "us-west-2"
region: "europe-central"
region: "asia-pacific"
```

---

### spec.timeoutSeconds

**Type:** `integer`  
**Required:** No  
**Default:** 300  
**Minimum:** 30  
**Maximum:** 3600  
**Description:** Timeout for intent processing in seconds

**Examples:**
```yaml
timeoutSeconds: 600   # 10 minutes
timeoutSeconds: 1800  # 30 minutes
```

---

### spec.maxRetries

**Type:** `integer`  
**Required:** No  
**Default:** 3  
**Minimum:** 0  
**Maximum:** 10  
**Description:** Maximum number of retry attempts for failed operations

**Examples:**
```yaml
maxRetries: 5   # Retry up to 5 times
maxRetries: 0   # No retries
```

## Validation Rules Summary

1. **Intent Validation**
   - Must contain telecommunications keywords
   - Length between 10-2000 characters

2. **Component Compatibility**
   - No duplicate components
   - Scaling operations cannot target NRF or NSSF
   - Maximum 20 components per intent

3. **Resource Validation**
   - Max resources must be >= min resources
   - Critical priority requires minimum resources
   - Valid Kubernetes quantity format

4. **Priority Rules**
   - Critical priority only for maintenance operations
   - Default to medium if not specified

5. **Identifier Formats**
   - Namespace: DNS-1123 label
   - Cluster: DNS-1123 subdomain
   - Network slice: Hexadecimal format
   - Region: Lowercase alphanumeric with hyphens

## Default Values

| Field | Default Value |
|-------|---------------|
| `intentType` | `"deployment"` |
| `priority` | `"medium"` |
| `timeoutSeconds` | `300` |
| `maxRetries` | `3` |

## Immutable Fields

Once created, the following fields cannot be modified:
- `spec.intent`
- `spec.intentType`
- `spec.networkSlice`

## Required Combinations

Certain field combinations have additional requirements:

### Production Deployments
```yaml
priority: high
resourceConstraints:
  cpu: "2"      # Minimum for production
  memory: "4Gi"  # Minimum for production
```

### Network Slicing
```yaml
networkSlice: "000001-000001"
targetComponents:
  - AMF
  - SMF
  - UPF  # Required for slice
```

### Multi-Region
```yaml
region: "us-west-2"
targetCluster: "west-cluster"  # Must match region
```

## See Also

- [Status Reference](status.md) - Understanding status fields
- [Examples](examples.md) - Real-world usage patterns
- [Troubleshooting](troubleshooting.md) - Common validation errors
# NetworkIntent Webhook Validation API Documentation

## Overview

The NetworkIntent Webhook Validation API provides comprehensive security and validation controls for Kubernetes NetworkIntent resources in the Nephoran Intent Operator. This admission webhook ensures that all NetworkIntent resources meet strict telecommunications industry standards and security requirements.

## API Specification

### Webhook Endpoint

```
POST /validate/networkintents
```

**Content-Type:** `application/json`

### Request Format

The webhook receives Kubernetes AdmissionReview requests with the following structure:

```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": {
    "uid": "string",
    "kind": {
      "group": "nephoran.io",
      "version": "v1",
      "kind": "NetworkIntent"
    },
    "resource": {
      "group": "nephoran.io",
      "version": "v1",
      "resource": "networkintents"
    },
    "operation": "CREATE|UPDATE|DELETE",
    "object": {
      "apiVersion": "nephoran.io/v1",
      "kind": "NetworkIntent",
      "metadata": {
        "name": "string",
        "namespace": "string"
      },
      "spec": {
        "intent": "string"
      }
    }
  }
}
```

### Response Format

```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "response": {
    "uid": "string",
    "allowed": true|false,
    "result": {
      "code": 200|400|403|422,
      "message": "string",
      "reason": "string"
    }
  }
}
```

## Validation Rules

### 1. Intent Content Validation

#### Requirements
- Intent must not be empty or whitespace-only
- Maximum length: 1000 characters
- Must contain only alphanumeric characters, spaces, and safe punctuation (`- _ . , ; : ( ) [ ]`)
- Cannot contain control characters (except newlines and tabs)
- Must be UTF-8 encoded

#### Examples

**Valid Intents:**
```json
{
  "intent": "Deploy AMF network function with high availability configuration"
}
```

**Invalid Intents:**
```json
{
  "intent": ""  // Empty intent
}

{
  "intent": "Deploy AMF\u0001"  // Control character
}

{
  "intent": "Very long intent that exceeds 1000 characters..."  // Too long
}
```

### 2. Security Validation

#### Malicious Pattern Detection

The webhook scans for and blocks the following patterns:

- **Script Injection:** `<script>`, `javascript:`, `onerror=`, `onclick=`
- **SQL Injection:** `UNION SELECT`, `DROP TABLE`, `INSERT INTO`, `DELETE FROM`, `OR 1=1`
- **Command Injection:** `;`, `&&`, `||`, `$(`, backticks, `|`
- **Path Traversal:** `../`, `..\\`, `/etc/`, `/proc/`
- **LDAP Injection:** `*)(cn=*`, `*)(uid=*`
- **Encoded Characters:** `\\x`, `\\u`, `%` followed by hex

#### Examples

**Blocked Intents:**
```json
{
  "intent": "Deploy AMF <script>alert('xss')</script>"
}

{
  "intent": "Setup network; DROP TABLE users; --"
}

{
  "intent": "Configure UPF $(cat /etc/passwd)"
}
```

### 3. Telecommunications Relevance Validation

#### Required Keywords

Intent must contain at least one telecommunications-related keyword from the following categories:

- **5G Core Network Functions:** AMF, SMF, UPF, AUSF, UDM, UDR, PCF, NRF, NSSF, SCP, SEPP
- **O-RAN Components:** O-CU, O-DU, O-RU, RIC, xApp, rApp, SMO
- **Network Services:** CNF, VNF, network slice, QoS, PLMN
- **Technologies:** 5G, LTE, ORAN, O-RAN, URLLC, eMBB, mMTC

#### Minimum Score Threshold

The intent must achieve a telecommunications relevance score of at least 0.3 based on:
- Keyword frequency and importance
- Context relevance
- Technical depth

#### Examples

**Valid Telecommunications Intents:**
```json
{
  "intent": "Deploy AMF instance with high availability for 5G core network"
}

{
  "intent": "Configure O-RAN Near-RT RIC with xApp orchestration capabilities"
}

{
  "intent": "Setup network slice for URLLC applications with 1ms latency"
}
```

**Invalid Non-Telecommunications Intents:**
```json
{
  "intent": "Deploy web application with database backend"
}

{
  "intent": "Create user management system"
}
```

### 4. Complexity Validation

#### Constraints
- Minimum words: 3
- Maximum words: 100
- Maximum word length: 30 characters
- Maximum consecutive repeated words: 3
- Must contain recognizable dictionary words
- No excessive special character usage

#### Business Logic Rules
- Must contain actionable verbs (deploy, configure, setup, scale, enable, etc.)
- Cannot contain contradictory terms (enable/disable, start/stop, create/delete)
- Cannot be overly vague ("do something", "make it work", "handle everything")

### 5. Resource Naming Validation

#### Kubernetes Name Requirements
- Length: 3-63 characters
- Must start and end with alphanumeric character
- Can contain hyphens in the middle
- Must be lowercase
- Cannot use reserved prefixes: `system-`, `admin-`, `kube-`

#### Examples

**Valid Names:**
```json
{
  "metadata": {
    "name": "amf-production-intent"
  }
}

{
  "metadata": {
    "name": "network-slice-urllc-001"
  }
}
```

**Invalid Names:**
```json
{
  "metadata": {
    "name": "ab"  // Too short
  }
}

{
  "metadata": {
    "name": "system-intent"  // Reserved prefix
  }
}
```

## HTTP Status Codes

| Status Code | Description | Example Scenario |
|-------------|-------------|-------------------|
| 200 OK | Request allowed | Valid NetworkIntent |
| 400 Bad Request | Malformed request | Invalid JSON structure |
| 403 Forbidden | Security violation | Malicious pattern detected |
| 422 Unprocessable Entity | Validation failure | Non-telecommunications intent |

## Error Response Examples

### Security Violation (403)
```json
{
  "response": {
    "allowed": false,
    "result": {
      "code": 403,
      "message": "Intent contains malicious pattern: script injection detected",
      "reason": "SecurityViolation"
    }
  }
}
```

### Validation Failure (422)
```json
{
  "response": {
    "allowed": false,
    "result": {
      "code": 422,
      "message": "Intent must be telecommunications-related and contain relevant technical keywords",
      "reason": "ValidationFailure"
    }
  }
}
```

### Content Policy Violation (422)
```json
{
  "response": {
    "allowed": false,
    "result": {
      "code": 422,
      "message": "Intent contains contradictory terms: cannot both enable and disable the same service",
      "reason": "BusinessLogicViolation"
    }
  }
}
```

## Configuration

### Default Settings
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: networkintent-webhook-config
  namespace: nephoran-system
data:
  max_intent_length: "1000"
  min_telecom_score: "0.3"
  max_complexity_words: "100"
  enable_security_scanning: "true"
  allowed_operations: "CREATE,UPDATE"
```

### Security Settings
```yaml
security:
  enable_malicious_pattern_detection: true
  enable_content_filtering: true
  enable_encoding_validation: true
  block_script_injection: true
  block_sql_injection: true
  block_command_injection: true
  block_path_traversal: true
```

## Monitoring and Metrics

### Prometheus Metrics

```
# Webhook request counts
networkintent_webhook_requests_total{operation="CREATE|UPDATE|DELETE",result="allowed|denied"}

# Validation latency
networkintent_webhook_validation_duration_seconds{validator="security|telecom|complexity"}

# Security violations
networkintent_webhook_security_violations_total{violation_type="script|sql|command|traversal"}
```

### Health Check Endpoint

```
GET /healthz
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-01T12:00:00Z",
  "checks": {
    "validator": "ok",
    "decoder": "ok",
    "security_scanner": "ok"
  }
}
```

## Client Examples

### cURL Example
```bash
curl -X POST https://nephoran-webhook.nephoran-system.svc.cluster.local/validate/networkintents \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "admission.k8s.io/v1",
    "kind": "AdmissionReview",
    "request": {
      "uid": "12345-67890",
      "operation": "CREATE",
      "object": {
        "apiVersion": "nephoran.io/v1",
        "kind": "NetworkIntent",
        "metadata": {
          "name": "amf-deployment",
          "namespace": "telecom-core"
        },
        "spec": {
          "intent": "Deploy AMF with high availability configuration"
        }
      }
    }
  }'
```

### Go Client Example
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    
    admissionv1 "k8s.io/api/admission/v1"
    nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func validateNetworkIntent(intent *nephoranv1.NetworkIntent) error {
    reviewRequest := &admissionv1.AdmissionReview{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "admission.k8s.io/v1",
            Kind:       "AdmissionReview",
        },
        Request: &admissionv1.AdmissionRequest{
            UID:       "test-request",
            Operation: admissionv1.Create,
            Object: runtime.RawExtension{
                Object: intent,
            },
        },
    }

    data, err := json.Marshal(reviewRequest)
    if err != nil {
        return err
    }

    resp, err := http.Post(
        "https://nephoran-webhook.nephoran-system.svc.cluster.local/validate/networkintents",
        "application/json",
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var reviewResponse admissionv1.AdmissionReview
    if err := json.NewDecoder(resp.Body).Decode(&reviewResponse); err != nil {
        return err
    }

    if !reviewResponse.Response.Allowed {
        return fmt.Errorf("validation failed: %s", reviewResponse.Response.Result.Message)
    }

    return nil
}
```

### Python Client Example
```python
import json
import requests

def validate_network_intent(name, namespace, intent_text):
    payload = {
        "apiVersion": "admission.k8s.io/v1",
        "kind": "AdmissionReview",
        "request": {
            "uid": "python-client-request",
            "operation": "CREATE",
            "object": {
                "apiVersion": "nephoran.io/v1",
                "kind": "NetworkIntent",
                "metadata": {
                    "name": name,
                    "namespace": namespace
                },
                "spec": {
                    "intent": intent_text
                }
            }
        }
    }
    
    response = requests.post(
        "https://nephoran-webhook.nephoran-system.svc.cluster.local/validate/networkintents",
        headers={"Content-Type": "application/json"},
        json=payload
    )
    
    result = response.json()
    
    if not result["response"]["allowed"]:
        raise ValueError(f"Validation failed: {result['response']['result']['message']}")
    
    return True

# Example usage
try:
    validate_network_intent(
        "amf-deployment", 
        "telecom-core", 
        "Deploy AMF with high availability configuration"
    )
    print("Intent validation successful!")
except ValueError as e:
    print(f"Validation error: {e}")
```

## Performance Considerations

### Latency Targets
- **P50 Latency:** < 10ms
- **P95 Latency:** < 50ms  
- **P99 Latency:** < 100ms

### Throughput
- **Maximum RPS:** 1000 requests per second
- **Concurrent Requests:** 100

### Resource Requirements
```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

## Security Considerations

### TLS Configuration
- **Minimum TLS Version:** 1.3
- **Certificate Validation:** Required
- **Mutual TLS:** Recommended for production

### Authentication
- **Kubernetes Service Account:** Required
- **RBAC Permissions:** Limited to admission review
- **Network Policies:** Restrict access to authorized namespaces

### Audit Logging
All webhook decisions are logged with the following format:
```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "level": "INFO",
  "message": "NetworkIntent validation",
  "fields": {
    "uid": "12345-67890",
    "operation": "CREATE",
    "name": "amf-deployment",
    "namespace": "telecom-core",
    "allowed": true,
    "duration_ms": 15,
    "validator_results": {
      "security": "passed",
      "telecom_relevance": "passed",
      "complexity": "passed"
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **Intent Rejected as Non-Telecommunications**
   - **Cause:** Missing telecommunications keywords
   - **Solution:** Include 5G core, O-RAN, or network service terms

2. **Security Pattern Detected**
   - **Cause:** Intent contains potentially malicious patterns
   - **Solution:** Remove special characters and script-like syntax

3. **Complexity Violation**
   - **Cause:** Intent too complex or contains repeated words
   - **Solution:** Simplify intent and remove redundant terms

4. **Name Validation Failure**
   - **Cause:** Kubernetes naming rules violation
   - **Solution:** Use lowercase alphanumeric names with hyphens

### Debug Mode
Enable debug logging by setting:
```yaml
env:
- name: LOG_LEVEL
  value: "debug"
```

This provides detailed validation step information for troubleshooting.

---

## Changelog

### v1.0.0 (2025-01-01)
- Initial API specification
- Core validation rules implementation
- Security pattern detection
- Telecommunications relevance scoring
- Resource naming validation
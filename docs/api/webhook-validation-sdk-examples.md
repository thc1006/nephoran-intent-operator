# NetworkIntent Webhook Validation SDK Examples

## Overview

This document provides comprehensive SDK examples and client library usage for the NetworkIntent Webhook Validation API. The examples demonstrate how to integrate with the webhook from various programming languages and frameworks.

## Go SDK Examples

### Basic Validation Client

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    admissionv1 "k8s.io/api/admission/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    
    nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// WebhookValidationClient provides validation capabilities
type WebhookValidationClient struct {
    endpoint string
    client   *http.Client
}

// NewWebhookValidationClient creates a new validation client
func NewWebhookValidationClient(endpoint string) *WebhookValidationClient {
    return &WebhookValidationClient{
        endpoint: endpoint,
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// ValidateIntent validates a NetworkIntent against the webhook
func (c *WebhookValidationClient) ValidateIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*ValidationResult, error) {
    // Create admission review request
    raw, err := json.Marshal(intent)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal intent: %w", err)
    }

    reviewRequest := &admissionv1.AdmissionReview{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "admission.k8s.io/v1",
            Kind:       "AdmissionReview",
        },
        Request: &admissionv1.AdmissionRequest{
            UID:       generateUID(),
            Operation: admissionv1.Create,
            Object: runtime.RawExtension{
                Raw: raw,
            },
            Kind: &metav1.GroupVersionKind{
                Group:   "nephoran.io",
                Version: "v1",
                Kind:    "NetworkIntent",
            },
        },
    }

    // Send request to webhook
    reqBody, err := json.Marshal(reviewRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/validate/networkintents", bytes.NewReader(reqBody))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    // Parse response
    var reviewResponse admissionv1.AdmissionReview
    if err := json.NewDecoder(resp.Body).Decode(&reviewResponse); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &ValidationResult{
        Allowed:    reviewResponse.Response.Allowed,
        Message:    reviewResponse.Response.Result.Message,
        StatusCode: int(reviewResponse.Response.Result.Code),
        Reason:     reviewResponse.Response.Result.Reason,
    }, nil
}

// ValidationResult represents the validation outcome
type ValidationResult struct {
    Allowed    bool   `json:"allowed"`
    Message    string `json:"message"`
    StatusCode int    `json:"status_code"`
    Reason     string `json:"reason"`
}

func generateUID() string {
    return fmt.Sprintf("sdk-client-%d", time.Now().UnixNano())
}

// Example usage
func main() {
    client := NewWebhookValidationClient("https://nephoran-webhook.nephoran-system.svc.cluster.local")
    
    intent := &nephoranv1.NetworkIntent{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "nephoran.io/v1",
            Kind:       "NetworkIntent",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name:      "amf-deployment",
            Namespace: "telecom-core",
        },
        Spec: nephoranv1.NetworkIntentSpec{
            Intent: "Deploy AMF with high availability configuration for 5G core network",
        },
    }

    result, err := client.ValidateIntent(context.Background(), intent)
    if err != nil {
        fmt.Printf("Validation error: %v\n", err)
        return
    }

    if result.Allowed {
        fmt.Println("Intent validation successful!")
    } else {
        fmt.Printf("Intent rejected: %s (reason: %s)\n", result.Message, result.Reason)
    }
}
```

### Batch Validation Client

```go
// BatchValidationClient handles multiple validations concurrently
type BatchValidationClient struct {
    client       *WebhookValidationClient
    maxConcurrency int
}

// NewBatchValidationClient creates a batch validation client
func NewBatchValidationClient(endpoint string, maxConcurrency int) *BatchValidationClient {
    return &BatchValidationClient{
        client:       NewWebhookValidationClient(endpoint),
        maxConcurrency: maxConcurrency,
    }
}

// ValidateIntents validates multiple intents concurrently
func (c *BatchValidationClient) ValidateIntents(ctx context.Context, intents []*nephoranv1.NetworkIntent) ([]*ValidationResult, error) {
    results := make([]*ValidationResult, len(intents))
    errors := make([]error, len(intents))
    
    sem := make(chan struct{}, c.maxConcurrency)
    var wg sync.WaitGroup
    
    for i, intent := range intents {
        wg.Add(1)
        go func(idx int, intent *nephoranv1.NetworkIntent) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            result, err := c.client.ValidateIntent(ctx, intent)
            results[idx] = result
            errors[idx] = err
        }(i, intent)
    }
    
    wg.Wait()
    
    // Check for any errors
    for _, err := range errors {
        if err != nil {
            return results, fmt.Errorf("batch validation failed: %w", err)
        }
    }
    
    return results, nil
}
```

## Python SDK Examples

### Basic Validation Client

```python
import asyncio
import json
import logging
from dataclasses import dataclass
from typing import Dict, Any, Optional
import aiohttp
import uuid


@dataclass
class ValidationResult:
    """Result of NetworkIntent validation"""
    allowed: bool
    message: str
    status_code: int
    reason: str


class WebhookValidationClient:
    """Async client for NetworkIntent webhook validation"""
    
    def __init__(self, endpoint: str, timeout: int = 10):
        self.endpoint = endpoint
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self.logger = logging.getLogger(__name__)
    
    async def validate_intent(
        self, 
        name: str, 
        namespace: str, 
        intent_text: str,
        operation: str = "CREATE"
    ) -> ValidationResult:
        """
        Validate a NetworkIntent
        
        Args:
            name: Name of the NetworkIntent resource
            namespace: Namespace for the resource
            intent_text: The natural language intent
            operation: Kubernetes operation (CREATE, UPDATE, DELETE)
            
        Returns:
            ValidationResult with validation outcome
        """
        
        # Create admission review request
        admission_request = {
            "apiVersion": "admission.k8s.io/v1",
            "kind": "AdmissionReview",
            "request": {
                "uid": str(uuid.uuid4()),
                "operation": operation,
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
                },
                "kind": {
                    "group": "nephoran.io",
                    "version": "v1",
                    "kind": "NetworkIntent"
                }
            }
        }
        
        # Send validation request
        async with aiohttp.ClientSession(timeout=self.timeout) as session:
            try:
                async with session.post(
                    f"{self.endpoint}/validate/networkintents",
                    json=admission_request,
                    headers={"Content-Type": "application/json"}
                ) as response:
                    
                    if response.status != 200:
                        raise aiohttp.ClientResponseError(
                            request_info=response.request_info,
                            history=response.history,
                            status=response.status,
                            message=f"HTTP {response.status}"
                        )
                    
                    result = await response.json()
                    
                    # Extract validation result
                    admission_response = result["response"]
                    return ValidationResult(
                        allowed=admission_response["allowed"],
                        message=admission_response.get("result", {}).get("message", ""),
                        status_code=admission_response.get("result", {}).get("code", 200),
                        reason=admission_response.get("result", {}).get("reason", "")
                    )
                    
            except aiohttp.ClientError as e:
                self.logger.error(f"Validation request failed: {e}")
                raise


# Example usage
async def main():
    client = WebhookValidationClient(
        "https://nephoran-webhook.nephoran-system.svc.cluster.local"
    )
    
    # Valid telecommunications intent
    try:
        result = await client.validate_intent(
            name="amf-deployment",
            namespace="telecom-core",
            intent_text="Deploy AMF with high availability configuration for 5G core network"
        )
        
        if result.allowed:
            print("✅ Intent validation successful!")
        else:
            print(f"❌ Intent rejected: {result.message}")
            print(f"   Reason: {result.reason}")
            
    except Exception as e:
        print(f"🔥 Validation error: {e}")

    # Invalid intent (non-telecommunications)
    try:
        result = await client.validate_intent(
            name="web-app",
            namespace="default",
            intent_text="Deploy a web application with database"
        )
        
        if result.allowed:
            print("✅ Intent validation successful!")
        else:
            print(f"❌ Intent rejected: {result.message}")
            
    except Exception as e:
        print(f"🔥 Validation error: {e}")


if __name__ == "__main__":
    asyncio.run(main())
```

### Batch Processing Client

```python
import asyncio
from typing import List, Tuple

class BatchValidationClient:
    """Client for batch validation of NetworkIntents"""
    
    def __init__(self, endpoint: str, max_concurrent: int = 10):
        self.client = WebhookValidationClient(endpoint)
        self.semaphore = asyncio.Semaphore(max_concurrent)
    
    async def validate_intent_batch(
        self, 
        intents: List[Tuple[str, str, str]]  # [(name, namespace, intent_text)]
    ) -> List[ValidationResult]:
        """
        Validate multiple intents concurrently
        
        Args:
            intents: List of (name, namespace, intent_text) tuples
            
        Returns:
            List of ValidationResult objects
        """
        
        async def validate_single(intent_data):
            async with self.semaphore:
                name, namespace, intent_text = intent_data
                return await self.client.validate_intent(name, namespace, intent_text)
        
        tasks = [validate_single(intent) for intent in intents]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Convert exceptions to failed ValidationResults
        processed_results = []
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                processed_results.append(ValidationResult(
                    allowed=False,
                    message=f"Validation failed: {str(result)}",
                    status_code=500,
                    reason="InternalError"
                ))
            else:
                processed_results.append(result)
        
        return processed_results

# Example batch usage
async def batch_example():
    client = BatchValidationClient(
        "https://nephoran-webhook.nephoran-system.svc.cluster.local",
        max_concurrent=5
    )
    
    intents = [
        ("amf-deployment", "telecom-core", "Deploy AMF with high availability"),
        ("smf-config", "telecom-core", "Configure SMF with UPF integration"),
        ("slice-setup", "telecom-core", "Setup network slice for URLLC"),
        ("invalid-intent", "default", "Make coffee and cookies"),  # Invalid
        ("oran-deploy", "oran-system", "Deploy O-RAN Near-RT RIC components")
    ]
    
    results = await client.validate_intent_batch(intents)
    
    for i, (intent_data, result) in enumerate(zip(intents, results)):
        name, namespace, intent_text = intent_data
        status = "✅ ALLOWED" if result.allowed else "❌ REJECTED"
        print(f"{i+1}. {name}: {status}")
        if not result.allowed:
            print(f"   Reason: {result.message}")
```

## JavaScript/Node.js SDK Examples

### TypeScript Client

```typescript
// types.ts
export interface NetworkIntent {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  spec: {
    intent: string;
  };
}

export interface ValidationResult {
  allowed: boolean;
  message: string;
  statusCode: number;
  reason: string;
}

export interface AdmissionReview {
  apiVersion: string;
  kind: string;
  request: {
    uid: string;
    operation: 'CREATE' | 'UPDATE' | 'DELETE';
    object: NetworkIntent;
    kind: {
      group: string;
      version: string;
      kind: string;
    };
  };
}

// webhook-client.ts
import axios, { AxiosInstance, AxiosError } from 'axios';
import { v4 as uuidv4 } from 'uuid';

export class WebhookValidationClient {
  private client: AxiosInstance;
  private endpoint: string;

  constructor(endpoint: string, timeout: number = 10000) {
    this.endpoint = endpoint;
    this.client = axios.create({
      timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async validateIntent(
    name: string,
    namespace: string,
    intentText: string,
    operation: 'CREATE' | 'UPDATE' | 'DELETE' = 'CREATE'
  ): Promise<ValidationResult> {
    
    const admissionRequest: AdmissionReview = {
      apiVersion: 'admission.k8s.io/v1',
      kind: 'AdmissionReview',
      request: {
        uid: uuidv4(),
        operation,
        object: {
          apiVersion: 'nephoran.io/v1',
          kind: 'NetworkIntent',
          metadata: {
            name,
            namespace,
          },
          spec: {
            intent: intentText,
          },
        },
        kind: {
          group: 'nephoran.io',
          version: 'v1',
          kind: 'NetworkIntent',
        },
      },
    };

    try {
      const response = await this.client.post(
        `${this.endpoint}/validate/networkintents`,
        admissionRequest
      );

      const admissionResponse = response.data.response;
      
      return {
        allowed: admissionResponse.allowed,
        message: admissionResponse.result?.message || '',
        statusCode: admissionResponse.result?.code || 200,
        reason: admissionResponse.result?.reason || '',
      };

    } catch (error) {
      if (axios.isAxiosError(error)) {
        const axiosError = error as AxiosError;
        throw new Error(
          `Validation request failed: ${axiosError.message} (${axiosError.response?.status})`
        );
      }
      throw error;
    }
  }

  async validateIntentBatch(
    intents: Array<{name: string; namespace: string; intent: string}>,
    concurrency: number = 5
  ): Promise<ValidationResult[]> {
    
    const chunks = this.chunkArray(intents, concurrency);
    const results: ValidationResult[] = [];

    for (const chunk of chunks) {
      const chunkPromises = chunk.map(intent =>
        this.validateIntent(intent.name, intent.namespace, intent.intent)
      );
      
      const chunkResults = await Promise.allSettled(chunkPromises);
      
      chunkResults.forEach(result => {
        if (result.status === 'fulfilled') {
          results.push(result.value);
        } else {
          results.push({
            allowed: false,
            message: `Validation failed: ${result.reason}`,
            statusCode: 500,
            reason: 'InternalError',
          });
        }
      });
    }

    return results;
  }

  private chunkArray<T>(array: T[], chunkSize: number): T[][] {
    const chunks: T[][] = [];
    for (let i = 0; i < array.length; i += chunkSize) {
      chunks.push(array.slice(i, i + chunkSize));
    }
    return chunks;
  }
}

// Example usage
async function main() {
  const client = new WebhookValidationClient(
    'https://nephoran-webhook.nephoran-system.svc.cluster.local'
  );

  try {
    // Single validation
    const result = await client.validateIntent(
      'amf-deployment',
      'telecom-core',
      'Deploy AMF with high availability configuration for 5G core network'
    );

    if (result.allowed) {
      console.log('✅ Intent validation successful!');
    } else {
      console.log(`❌ Intent rejected: ${result.message}`);
      console.log(`   Reason: ${result.reason}`);
    }

    // Batch validation
    const intents = [
      {
        name: 'amf-deployment',
        namespace: 'telecom-core',
        intent: 'Deploy AMF with high availability'
      },
      {
        name: 'smf-config',
        namespace: 'telecom-core', 
        intent: 'Configure SMF with UPF integration'
      },
      {
        name: 'invalid-intent',
        namespace: 'default',
        intent: 'Make coffee and cookies'  // Invalid
      }
    ];

    const batchResults = await client.validateIntentBatch(intents);
    
    batchResults.forEach((result, index) => {
      const intent = intents[index];
      const status = result.allowed ? '✅ ALLOWED' : '❌ REJECTED';
      console.log(`${index + 1}. ${intent.name}: ${status}`);
      if (!result.allowed) {
        console.log(`   Reason: ${result.message}`);
      }
    });

  } catch (error) {
    console.error('🔥 Validation error:', error);
  }
}

if (require.main === module) {
  main().catch(console.error);
}
```

## Java SDK Examples

### Spring Boot Integration

```java
// WebhookValidationClient.java
package com.nephoran.validation;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;

import java.time.Duration;
import java.util.UUID;

@Service
public class WebhookValidationClient {
    
    private final WebClient webClient;
    private final ObjectMapper objectMapper;
    private final String endpoint;
    
    public WebhookValidationClient(String endpoint) {
        this.endpoint = endpoint;
        this.webClient = WebClient.builder()
            .baseUrl(endpoint)
            .build();
        this.objectMapper = new ObjectMapper();
    }
    
    public Mono<ValidationResult> validateIntent(
            String name, 
            String namespace, 
            String intentText) {
        
        AdmissionReview request = createAdmissionRequest(name, namespace, intentText);
        
        return webClient.post()
            .uri("/validate/networkintents")
            .bodyValue(request)
            .retrieve()
            .bodyToMono(AdmissionReview.class)
            .timeout(Duration.ofSeconds(10))
            .map(response -> ValidationResult.builder()
                .allowed(response.getResponse().getAllowed())
                .message(response.getResponse().getResult().getMessage())
                .statusCode(response.getResponse().getResult().getCode())
                .reason(response.getResponse().getResult().getReason())
                .build())
            .onErrorMap(throwable -> new ValidationException(
                "Failed to validate intent: " + throwable.getMessage(), throwable));
    }
    
    private AdmissionReview createAdmissionRequest(String name, String namespace, String intentText) {
        NetworkIntent intent = NetworkIntent.builder()
            .apiVersion("nephoran.io/v1")
            .kind("NetworkIntent")
            .metadata(ObjectMeta.builder()
                .name(name)
                .namespace(namespace)
                .build())
            .spec(NetworkIntentSpec.builder()
                .intent(intentText)
                .build())
            .build();
            
        return AdmissionReview.builder()
            .apiVersion("admission.k8s.io/v1")
            .kind("AdmissionReview")
            .request(AdmissionRequest.builder()
                .uid(UUID.randomUUID().toString())
                .operation("CREATE")
                .object(intent)
                .kind(GroupVersionKind.builder()
                    .group("nephoran.io")
                    .version("v1")
                    .kind("NetworkIntent")
                    .build())
                .build())
            .build();
    }
}

// ValidationResult.java
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ValidationResult {
    private boolean allowed;
    private String message;
    private int statusCode;
    private String reason;
}

// Example Spring Boot controller
@RestController
@RequestMapping("/api/v1/intents")
@RequiredArgsConstructor
public class IntentController {
    
    private final WebhookValidationClient validationClient;
    
    @PostMapping("/validate")
    public Mono<ResponseEntity<ValidationResponse>> validateIntent(
            @RequestBody @Valid IntentValidationRequest request) {
        
        return validationClient.validateIntent(
                request.getName(),
                request.getNamespace(), 
                request.getIntent())
            .map(result -> {
                if (result.isAllowed()) {
                    return ResponseEntity.ok(ValidationResponse.builder()
                        .status("ALLOWED")
                        .message("Intent validation successful")
                        .build());
                } else {
                    return ResponseEntity.status(HttpStatus.UNPROCESSABLE_ENTITY)
                        .body(ValidationResponse.builder()
                            .status("REJECTED")
                            .message(result.getMessage())
                            .reason(result.getReason())
                            .build());
                }
            })
            .onErrorReturn(ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                .body(ValidationResponse.builder()
                    .status("ERROR")
                    .message("Internal validation error")
                    .build()));
    }
}
```

## Curl Examples

### Basic Validation

```bash
#!/bin/bash

WEBHOOK_URL="https://nephoran-webhook.nephoran-system.svc.cluster.local"

# Valid telecommunications intent
curl -X POST "${WEBHOOK_URL}/validate/networkintents" \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "admission.k8s.io/v1",
    "kind": "AdmissionReview",
    "request": {
      "uid": "test-request-001",
      "operation": "CREATE",
      "object": {
        "apiVersion": "nephoran.io/v1",
        "kind": "NetworkIntent",
        "metadata": {
          "name": "amf-deployment",
          "namespace": "telecom-core"
        },
        "spec": {
          "intent": "Deploy AMF with high availability configuration for 5G core network"
        }
      },
      "kind": {
        "group": "nephoran.io",
        "version": "v1",
        "kind": "NetworkIntent"
      }
    }
  }' | jq '.'
```

### Invalid Intent Test

```bash
# Invalid non-telecommunications intent
curl -X POST "${WEBHOOK_URL}/validate/networkintents" \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "admission.k8s.io/v1",
    "kind": "AdmissionReview",
    "request": {
      "uid": "test-request-002",
      "operation": "CREATE",
      "object": {
        "apiVersion": "nephoran.io/v1",
        "kind": "NetworkIntent",
        "metadata": {
          "name": "web-app",
          "namespace": "default"
        },
        "spec": {
          "intent": "Deploy a web application with database backend"
        }
      }
    }
  }' | jq '.response | {allowed, result: {code, message, reason}}'
```

### Security Violation Test

```bash
# Security violation (script injection)
curl -X POST "${WEBHOOK_URL}/validate/networkintents" \
  -H "Content-Type: application/json" \
  -d '{
    "apiVersion": "admission.k8s.io/v1",
    "kind": "AdmissionReview",
    "request": {
      "uid": "test-request-003",
      "operation": "CREATE",
      "object": {
        "apiVersion": "nephoran.io/v1",
        "kind": "NetworkIntent",
        "metadata": {
          "name": "malicious-intent",
          "namespace": "default"
        },
        "spec": {
          "intent": "Deploy AMF <script>alert('xss')</script>"
        }
      }
    }
  }' | jq '.response.result'
```

## Testing and Validation

### Integration Test Suite

```go
// integration_test.go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type WebhookIntegrationTestSuite struct {
    suite.Suite
    client *WebhookValidationClient
}

func (s *WebhookIntegrationTestSuite) SetupSuite() {
    s.client = NewWebhookValidationClient(
        "https://nephoran-webhook.nephoran-system.svc.cluster.local",
    )
}

func (s *WebhookIntegrationTestSuite) TestValidTelecomIntents() {
    testCases := []struct {
        name   string
        intent string
    }{
        {
            name:   "AMF Deployment",
            intent: "Deploy AMF with high availability configuration",
        },
        {
            name:   "O-RAN Setup",
            intent: "Configure O-RAN Near-RT RIC with xApp orchestration",
        },
        {
            name:   "Network Slice",
            intent: "Setup network slice for URLLC applications",
        },
    }
    
    for _, tc := range testCases {
        s.Run(tc.name, func() {
            intent := createTestIntent(tc.name, "telecom-core", tc.intent)
            result, err := s.client.ValidateIntent(context.Background(), intent)
            
            assert.NoError(s.T(), err)
            assert.True(s.T(), result.Allowed, "Expected intent to be allowed")
            assert.Equal(s.T(), 200, result.StatusCode)
        })
    }
}

func (s *WebhookIntegrationTestSuite) TestInvalidIntents() {
    testCases := []struct {
        name           string
        intent         string
        expectedReason string
    }{
        {
            name:           "Non-Telecom Intent",
            intent:         "Deploy a web application with database",
            expectedReason: "ValidationFailure",
        },
        {
            name:           "Script Injection",
            intent:         "Deploy AMF <script>alert('xss')</script>",
            expectedReason: "SecurityViolation",
        },
        {
            name:           "Empty Intent",
            intent:         "",
            expectedReason: "ValidationFailure",
        },
    }
    
    for _, tc := range testCases {
        s.Run(tc.name, func() {
            intent := createTestIntent(tc.name, "default", tc.intent)
            result, err := s.client.ValidateIntent(context.Background(), intent)
            
            assert.NoError(s.T(), err)
            assert.False(s.T(), result.Allowed, "Expected intent to be rejected")
            assert.Equal(s.T(), tc.expectedReason, result.Reason)
        })
    }
}

func (s *WebhookIntegrationTestSuite) TestPerformanceRequirements() {
    intent := createTestIntent("performance-test", "telecom-core", 
        "Deploy AMF with high availability configuration")
    
    start := time.Now()
    result, err := s.client.ValidateIntent(context.Background(), intent)
    duration := time.Since(start)
    
    assert.NoError(s.T(), err)
    assert.True(s.T(), result.Allowed)
    assert.Less(s.T(), duration, 100*time.Millisecond, "Validation should complete within 100ms")
}

func TestWebhookIntegration(t *testing.T) {
    suite.Run(t, new(WebhookIntegrationTestSuite))
}
```

## Best Practices

### Error Handling

1. **Implement Circuit Breaker Pattern**
2. **Use Exponential Backoff for Retries** 
3. **Handle Timeout Scenarios Gracefully**
4. **Log All Validation Attempts for Debugging**

### Performance Optimization

1. **Use Connection Pooling**
2. **Implement Request Batching for Multiple Validations**
3. **Cache Validation Results When Appropriate**
4. **Monitor Latency and Throughput Metrics**

### Security Considerations

1. **Always Use HTTPS for Production**
2. **Implement Proper Certificate Validation**
3. **Use Service Account Tokens for Authentication**
4. **Log Security Violations for Monitoring**

---

This SDK documentation provides comprehensive examples for integrating with the NetworkIntent Webhook Validation API across multiple programming languages and frameworks. Choose the implementation that best fits your technology stack and requirements.
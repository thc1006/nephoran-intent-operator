# Nephoran Intent Operator Go SDK Guide

## Overview

The Nephoran Intent Operator Go SDK provides a comprehensive client library for interacting with the Nephoran Intent Operator API. This production-ready SDK offers:

- Type-safe API interactions with full OpenAPI 3.0 compliance
- OAuth2 authentication with automatic token management
- Built-in retry logic and circuit breaker patterns
- Structured error handling with context preservation
- Comprehensive logging and tracing support
- High-performance connection pooling
- Enterprise-grade security features

## Installation

```bash
go get github.com/nephoran/intent-operator-sdk/v1
```

## Quick Start

### Basic Client Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/nephoran/intent-operator-sdk/v1/client"
    "github.com/nephoran/intent-operator-sdk/v1/types"
)

func main() {
    // Initialize client with OAuth2 authentication
    config := &client.Config{
        BaseURL: "https://api.nephoran.com/v1",
        AuthConfig: &client.OAuth2Config{
            ClientID:     "your-client-id",
            ClientSecret: "your-client-secret",
            TokenURL:     "https://auth.nephoran.com/oauth/token",
            Scopes:       []string{"intent.read", "intent.write", "intent.execute"},
        },
        Timeout: 30 * time.Second,
    }

    c, err := client.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer c.Close()

    // Create a network intent
    intent := &types.NetworkIntent{
        APIVersion: "nephoran.io/v1alpha1",
        Kind:       "NetworkIntent",
        Metadata: types.ObjectMeta{
            Name:      "amf-production",
            Namespace: "telecom-5g",
            Labels: map[string]string{
                "environment": "production",
                "component":   "amf",
            },
        },
        Spec: types.NetworkIntentSpec{
            Intent:   "Deploy a high-availability AMF instance for production with auto-scaling",
            Priority: "high",
            Parameters: map[string]interface{}{
                "replicas":         3,
                "enableAutoScaling": true,
            },
        },
    }

    result, err := c.Intents.Create(context.Background(), intent)
    if err != nil {
        log.Fatalf("Intent creation failed: %v", err)
    }

    log.Printf("Intent created: %s", result.Metadata.Name)
}
```

### Advanced Client Configuration

```go
package main

import (
    "context"
    "crypto/tls"
    "net/http"
    "time"

    "github.com/nephoran/intent-operator-sdk/v1/client"
)

func main() {
    // Advanced configuration with custom transport and middleware
    config := &client.Config{
        BaseURL: "https://api.nephoran.com/v1",
        AuthConfig: &client.OAuth2Config{
            ClientID:     "your-client-id",
            ClientSecret: "your-client-secret",
            TokenURL:     "https://auth.nephoran.com/oauth/token",
            Scopes:       []string{"intent.read", "intent.write", "rag.query"},
        },
        HTTPClient: &http.Client{
            Timeout: 60 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
                TLSClientConfig: &tls.Config{
                    MinVersion: tls.VersionTLS12,
                },
            },
        },
        RetryConfig: &client.RetryConfig{
            MaxRetries: 3,
            BackoffStrategy: client.ExponentialBackoff,
            RetryableStatusCodes: []int{429, 500, 502, 503, 504},
        },
        CircuitBreakerConfig: &client.CircuitBreakerConfig{
            MaxRequests:  100,
            Interval:     60 * time.Second,
            Timeout:      30 * time.Second,
            ReadyToTrip:  func(counts client.Counts) bool {
                return counts.ConsecutiveFailures > 5
            },
        },
    }

    c, err := client.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer c.Close()

    // Client ready for production use
}
```

## Core API Operations

### Intent Management

#### Creating Network Intents

```go
// 5G Core AMF Deployment
func createAMFIntent(client *client.Client) error {
    ctx := context.Background()
    
    intent := &types.NetworkIntent{
        APIVersion: "nephoran.io/v1alpha1",
        Kind:       "NetworkIntent",
        Metadata: types.ObjectMeta{
            Name:      "production-amf",
            Namespace: "telecom-5g",
            Labels: map[string]string{
                "environment": "production",
                "nf-type":     "amf",
                "release":     "rel16",
            },
        },
        Spec: types.NetworkIntentSpec{
            Intent:   "Deploy high-availability AMF with N1/N2 interface support for 100,000 concurrent UE registrations",
            Priority: "high",
            Parameters: map[string]interface{}{
                "replicas":              5,
                "enableAutoScaling":     true,
                "maxConcurrentUEs":      100000,
                "n1InterfaceEnabled":    true,
                "n2InterfaceEnabled":    true,
                "sessionPersistence":    true,
                "loadBalancingStrategy": "round-robin",
            },
            TargetClusters: []string{"cluster-east-1", "cluster-west-1"},
            ORANCompliance: true,
        },
    }

    result, err := client.Intents.Create(ctx, intent)
    if err != nil {
        return fmt.Errorf("failed to create AMF intent: %w", err)
    }

    log.Printf("AMF Intent created: %s (ID: %s)", result.Metadata.Name, result.Metadata.UID)
    return nil
}

// Network Slice Creation
func createNetworkSlice(client *client.Client) error {
    ctx := context.Background()
    
    sliceIntent := &types.NetworkIntent{
        APIVersion: "nephoran.io/v1alpha1",
        Kind:       "NetworkSlice",
        Metadata: types.ObjectMeta{
            Name:      "embb-premium-slice",
            Namespace: "network-slicing",
            Labels: map[string]string{
                "slice-type":    "eMBB",
                "service-tier":  "premium",
                "sla-level":     "gold",
            },
        },
        Spec: types.NetworkIntentSpec{
            Intent: "Create eMBB network slice for premium mobile broadband with 1Gbps guaranteed bandwidth",
            Priority: "medium",
            Parameters: map[string]interface{}{
                "sliceType":           "eMBB",
                "guaranteedBitRate":   "1Gbps",
                "maxBitRate":          "10Gbps", 
                "latencyBudget":       "50ms",
                "reliabilityTarget":   "99.9%",
                "isolationLevel":      "high",
                "qosFlowProfiles": []map[string]interface{}{
                    {
                        "5qi": 1,
                        "arp": map[string]interface{}{
                            "priority": 2,
                            "preemption": map[string]interface{}{
                                "capability":    "may-preempt",
                                "vulnerability": "not-preemptable",
                            },
                        },
                    },
                },
            },
            NetworkSlice: "s-nssai-01-000001",
        },
    }

    result, err := client.Intents.Create(ctx, sliceIntent)
    if err != nil {
        return fmt.Errorf("failed to create network slice: %w", err)
    }

    log.Printf("Network Slice created: %s", result.Metadata.Name)
    return nil
}

// O-RAN Near-RT RIC Deployment
func createORANIntent(client *client.Client) error {
    ctx := context.Background()
    
    ranIntent := &types.NetworkIntent{
        APIVersion: "nephoran.io/v1alpha1",
        Kind:       "RANIntent",
        Metadata: types.ObjectMeta{
            Name:      "near-rt-ric-production",
            Namespace: "o-ran",
            Labels: map[string]string{
                "component":     "near-rt-ric",
                "oran-release":  "r5",
                "deployment":    "production",
            },
        },
        Spec: types.NetworkIntentSpec{
            Intent: "Deploy Near-RT RIC with traffic steering, QoS optimization, and anomaly detection xApps",
            Priority: "high",
            Parameters: map[string]interface{}{
                "xApps": []string{
                    "traffic-steering",
                    "qos-optimizer", 
                    "anomaly-detector",
                    "load-balancer",
                },
                "e2Interfaces": []string{
                    "E2-KPM",
                    "E2-RC", 
                    "E2-NI",
                    "E2-CCC",
                },
                "a1PolicyTypes": []string{
                    "traffic_steering_policy",
                    "qos_management_policy",
                },
                "ricPlatform": map[string]interface{}{
                    "version":           "1.3.6",
                    "enablePersistence": true,
                    "messageBusConfig": map[string]interface{}{
                        "type":       "rmr",
                        "instances":  3,
                    },
                },
            },
            ORANCompliance: true,
        },
    }

    result, err := client.Intents.Create(ctx, ranIntent)
    if err != nil {
        return fmt.Errorf("failed to create O-RAN intent: %w", err)
    }

    log.Printf("O-RAN Intent created: %s", result.Metadata.Name)
    return nil
}
```

#### Listing and Filtering Intents

```go
func listIntents(client *client.Client) error {
    ctx := context.Background()
    
    // List all intents with pagination
    listOptions := &client.ListOptions{
        Limit:     20,
        Offset:    0,
        Namespace: "telecom-5g",
        Labels:    "environment=production",
    }

    intents, err := client.Intents.List(ctx, listOptions)
    if err != nil {
        return fmt.Errorf("failed to list intents: %w", err)
    }

    fmt.Printf("Found %d intents (total: %d)\n", 
        len(intents.Items), intents.Metadata.TotalItems)

    for _, intent := range intents.Items {
        fmt.Printf("- %s/%s: %s (Status: %s)\n", 
            intent.Metadata.Namespace,
            intent.Metadata.Name,
            intent.Spec.Intent[:50]+"...",
            intent.Status.Phase)
    }

    return nil
}

// Filter intents by status
func filterIntentsByStatus(client *client.Client, status string) ([]*types.NetworkIntent, error) {
    ctx := context.Background()
    
    listOptions := &client.ListOptions{
        FieldSelector: fmt.Sprintf("status.phase=%s", status),
        Limit:         100,
    }

    result, err := client.Intents.List(ctx, listOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to filter intents: %w", err)
    }

    return result.Items, nil
}
```

#### Watching Intent Events

```go
func watchIntentEvents(client *client.Client, namespace, name string) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Watch intent processing events using Server-Sent Events
    eventStream, err := client.Intents.WatchEvents(ctx, namespace, name)
    if err != nil {
        return fmt.Errorf("failed to watch events: %w", err)
    }
    defer eventStream.Close()

    fmt.Printf("Watching events for intent %s/%s\n", namespace, name)

    for event := range eventStream.Events() {
        switch event.Type {
        case types.EventTypeProcessing:
            fmt.Printf("[%s] Processing: %s\n", event.Timestamp, event.Data["stage"])
        case types.EventTypeContextRetrieved:
            docs := event.Data["documents"].(float64)
            score := event.Data["relevance_score"].(float64)
            fmt.Printf("[%s] Context retrieved: %.0f documents (score: %.2f)\n", 
                event.Timestamp, docs, score)
        case types.EventTypeDeploymentStarted:
            resources := event.Data["resources"].(float64)
            fmt.Printf("[%s] Deployment started: %.0f resources\n", 
                event.Timestamp, resources)
        case types.EventTypeError:
            fmt.Printf("[%s] Error: %s\n", event.Timestamp, event.Data["message"])
            return fmt.Errorf("intent processing failed: %s", event.Data["message"])
        }

        // Check if context is cancelled
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }

    return nil
}
```

### LLM Processing

```go
func processIntentWithLLM(client *client.Client, intentText string) (*types.LLMProcessingResponse, error) {
    ctx := context.Background()

    request := &types.LLMProcessingRequest{
        Intent: intentText,
        Context: []string{
            "5G Core network functions require high availability",
            "Auto-scaling based on connection count and CPU utilization", 
            "Production deployments need comprehensive monitoring",
        },
        Parameters: &types.LLMParameters{
            Model:       "gpt-4o-mini",
            Temperature: 0.3,
            MaxTokens:   2000,
        },
    }

    response, err := client.LLM.Process(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("LLM processing failed: %w", err)
    }

    fmt.Printf("Processed intent with confidence: %.2f\n", response.Confidence)
    fmt.Printf("Identified network functions: %v\n", response.NetworkFunctions)
    fmt.Printf("Deployment strategy: %s\n", response.DeploymentStrategy)

    return response, nil
}

// Streaming LLM processing
func processIntentWithStreaming(client *client.Client, intentText string) error {
    ctx := context.Background()

    request := &types.LLMProcessingRequest{
        Intent:    intentText,
        Streaming: true,
        Parameters: &types.LLMParameters{
            Model:       "gpt-4o-mini",
            Temperature: 0.2,
        },
    }

    stream, err := client.LLM.ProcessStream(ctx, request)
    if err != nil {
        return fmt.Errorf("failed to start streaming: %w", err)
    }
    defer stream.Close()

    fmt.Println("Streaming LLM response:")
    for chunk := range stream.Chunks() {
        if chunk.Error != nil {
            return fmt.Errorf("streaming error: %w", chunk.Error)
        }
        fmt.Print(chunk.Content)
    }
    fmt.Println()

    return nil
}
```

### RAG Knowledge Retrieval

```go
func queryKnowledgeBase(client *client.Client, query string, domain string) (*types.RAGResponse, error) {
    ctx := context.Background()

    request := &types.RAGQuery{
        Query: query,
        Context: &types.RAGContext{
            Domain:     domain,
            Technology: "5g-core",
        },
        MaxResults:      5,
        Threshold:       0.8,
        IncludeMetadata: true,
    }

    response, err := client.RAG.Query(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("RAG query failed: %w", err)
    }

    fmt.Printf("Retrieved %d results in %dms\n", 
        len(response.Results), response.Metadata.ProcessingTimeMs)

    for i, result := range response.Results {
        fmt.Printf("\n--- Result %d (Score: %.3f) ---\n", i+1, result.Score)
        fmt.Printf("Source: %s\n", result.Source)
        fmt.Printf("Text: %s\n", result.Text[:200]+"...")
        
        if result.Metadata != nil {
            fmt.Printf("Document Type: %s, Section: %s\n", 
                result.Metadata.DocumentType, result.Metadata.Section)
        }
    }

    return response, nil
}

// Retrieve knowledge base information
func listKnowledgeBaseDocs(client *client.Client) error {
    ctx := context.Background()

    params := &client.KnowledgeBaseParams{
        Domain:       "o-ran",
        DocumentType: "specification",
        Limit:        10,
    }

    docs, err := client.RAG.ListDocuments(ctx, params)
    if err != nil {
        return fmt.Errorf("failed to list knowledge base documents: %w", err)
    }

    fmt.Printf("Knowledge Base Documents (%s domain):\n", params.Domain)
    for _, doc := range docs.Documents {
        fmt.Printf("- %s (%s) - %d chunks, updated: %s\n",
            doc.Title, doc.DocumentType, doc.ChunkCount, doc.LastUpdated)
    }

    return nil
}
```

### Error Handling

```go
package main

import (
    "errors"
    "fmt"

    "github.com/nephoran/intent-operator-sdk/v1/client"
    "github.com/nephoran/intent-operator-sdk/v1/types"
)

func handleAPIErrors(err error) {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "INTENT_VALIDATION_FAILED":
            fmt.Printf("Validation Error: %s\n", apiErr.Message)
            if apiErr.Details != nil {
                if field, ok := apiErr.Details["field"].(string); ok {
                    fmt.Printf("Field: %s\n", field)
                }
                if suggestions, ok := apiErr.Details["suggestions"].([]interface{}); ok {
                    fmt.Println("Suggestions:")
                    for _, suggestion := range suggestions {
                        fmt.Printf("  - %s\n", suggestion)
                    }
                }
            }
        case "RATE_LIMIT_EXCEEDED":
            fmt.Printf("Rate limit exceeded: %s\n", apiErr.Message)
            // Implement exponential backoff retry
        case "INSUFFICIENT_PERMISSIONS":
            fmt.Printf("Permission denied: %s\n", apiErr.Message)
            // Handle authentication/authorization issues
        default:
            fmt.Printf("API Error [%s]: %s\n", apiErr.Code, apiErr.Message)
        }
    } else {
        fmt.Printf("SDK Error: %v\n", err)
    }
}

// Retry with exponential backoff
func createIntentWithRetry(client *client.Client, intent *types.NetworkIntent, maxRetries int) (*types.NetworkIntent, error) {
    var lastErr error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := client.Intents.Create(context.Background(), intent)
        if err == nil {
            return result, nil
        }

        lastErr = err
        
        var apiErr *client.APIError
        if errors.As(err, &apiErr) {
            // Don't retry client errors (4xx)
            if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
                return nil, err
            }
        }

        // Exponential backoff
        backoffTime := time.Duration(1<<attempt) * time.Second
        fmt.Printf("Attempt %d failed, retrying in %v...\n", attempt+1, backoffTime)
        time.Sleep(backoffTime)
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

## Advanced Features

### Batch Operations

```go
func batchCreateIntents(client *client.Client, intents []*types.NetworkIntent) error {
    ctx := context.Background()
    
    // Create intents concurrently with controlled parallelism
    semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests
    results := make(chan error, len(intents))
    
    for _, intent := range intents {
        go func(intent *types.NetworkIntent) {
            semaphore <- struct{}{} // Acquire semaphore
            defer func() { <-semaphore }() // Release semaphore
            
            _, err := client.Intents.Create(ctx, intent)
            results <- err
        }(intent)
    }
    
    var errors []error
    for i := 0; i < len(intents); i++ {
        if err := <-results; err != nil {
            errors = append(errors, err)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("batch operation failed with %d errors: %v", len(errors), errors[0])
    }
    
    return nil
}
```

### Custom Middleware

```go
// Request logging middleware
func loggingMiddleware(next client.RoundTripper) client.RoundTripper {
    return client.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
        start := time.Now()
        
        log.Printf("→ %s %s", req.Method, req.URL.Path)
        
        resp, err := next.RoundTrip(req)
        
        duration := time.Since(start)
        if err != nil {
            log.Printf("← %s %s: ERROR %v (%v)", req.Method, req.URL.Path, err, duration)
        } else {
            log.Printf("← %s %s: %d (%v)", req.Method, req.URL.Path, resp.StatusCode, duration)
        }
        
        return resp, err
    })
}

// Add middleware to client
func createClientWithMiddleware() (*client.Client, error) {
    config := &client.Config{
        BaseURL: "https://api.nephoran.com/v1",
        AuthConfig: &client.OAuth2Config{
            ClientID:     "your-client-id",
            ClientSecret: "your-client-secret",
            TokenURL:     "https://auth.nephoran.com/oauth/token",
            Scopes:       []string{"intent.read", "intent.write"},
        },
        Middleware: []client.Middleware{
            loggingMiddleware,
            // Add more middleware as needed
        },
    }

    return client.NewClient(config)
}
```

### Testing Utilities

```go
package main

import (
    "testing"
    "net/http"
    "net/http/httptest"

    "github.com/nephoran/intent-operator-sdk/v1/client"
    "github.com/nephoran/intent-operator-sdk/v1/testing/mock"
)

func TestIntentCreation(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/intents" && r.Method == "POST" {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusCreated)
            w.Write([]byte(`{
                "apiVersion": "nephoran.io/v1alpha1",
                "kind": "NetworkIntent",
                "metadata": {
                    "name": "test-intent",
                    "namespace": "default",
                    "uid": "12345"
                },
                "spec": {
                    "intent": "Test intent"
                },
                "status": {
                    "phase": "Pending"
                }
            }`))
        }
    }))
    defer server.Close()

    // Create client with mock server
    config := &client.Config{
        BaseURL: server.URL,
        AuthConfig: &client.OAuth2Config{
            ClientID:     "test-client",
            ClientSecret: "test-secret", 
            TokenURL:     server.URL + "/oauth/token",
        },
    }

    c, err := client.NewClient(config)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    // Test intent creation
    intent := &types.NetworkIntent{
        APIVersion: "nephoran.io/v1alpha1",
        Kind:       "NetworkIntent",
        Metadata: types.ObjectMeta{
            Name:      "test-intent",
            Namespace: "default",
        },
        Spec: types.NetworkIntentSpec{
            Intent: "Test intent",
        },
    }

    result, err := c.Intents.Create(context.Background(), intent)
    if err != nil {
        t.Errorf("Intent creation failed: %v", err)
    }

    if result.Metadata.Name != "test-intent" {
        t.Errorf("Expected name 'test-intent', got '%s'", result.Metadata.Name)
    }
}

// Use SDK's built-in mock utilities
func TestWithMockClient(t *testing.T) {
    mockClient := mock.NewMockClient()
    
    // Configure mock responses
    mockClient.Intents.OnCreate().Return(&types.NetworkIntent{
        Metadata: types.ObjectMeta{
            Name: "mock-intent",
            UID:  "mock-uid",
        },
        Status: types.NetworkIntentStatus{
            Phase: "Deployed",
        },
    }, nil)

    // Run tests with mock client
    intent := &types.NetworkIntent{
        Metadata: types.ObjectMeta{Name: "test"},
        Spec:     types.NetworkIntentSpec{Intent: "test intent"},
    }

    result, err := mockClient.Intents.Create(context.Background(), intent)
    if err != nil {
        t.Errorf("Mock creation failed: %v", err)
    }

    if result.Status.Phase != "Deployed" {
        t.Errorf("Expected status 'Deployed', got '%s'", result.Status.Phase)
    }
}
```

## Best Practices

### Production Configuration

```go
func createProductionClient() (*client.Client, error) {
    config := &client.Config{
        BaseURL: "https://api.nephoran.com/v1",
        AuthConfig: &client.OAuth2Config{
            ClientID:     os.Getenv("NEPHORAN_CLIENT_ID"),
            ClientSecret: os.Getenv("NEPHORAN_CLIENT_SECRET"),
            TokenURL:     "https://auth.nephoran.com/oauth/token",
            Scopes:       []string{"intent.read", "intent.write", "intent.execute", "rag.query"},
        },
        HTTPClient: &http.Client{
            Timeout: 60 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        50,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
                DisableKeepAlives:   false,
                TLSClientConfig: &tls.Config{
                    MinVersion: tls.VersionTLS12,
                },
            },
        },
        RetryConfig: &client.RetryConfig{
            MaxRetries:           3,
            BackoffStrategy:      client.ExponentialBackoff,
            RetryableStatusCodes: []int{429, 500, 502, 503, 504},
            RetryOnTimeout:       true,
        },
        CircuitBreakerConfig: &client.CircuitBreakerConfig{
            MaxRequests: 100,
            Interval:    60 * time.Second,
            Timeout:     30 * time.Second,
            ReadyToTrip: func(counts client.Counts) bool {
                failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
                return counts.Requests >= 10 && failureRatio >= 0.6
            },
        },
        RateLimiter: client.NewTokenBucketLimiter(100, time.Minute), // 100 requests per minute
    }

    return client.NewClient(config)
}
```

### Resource Management

```go
func properResourceManagement() error {
    // Always create clients with proper resource management
    client, err := createProductionClient()
    if err != nil {
        return err
    }
    defer client.Close() // Important: Always close the client

    // Use contexts with timeouts for all operations
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Perform operations
    intents, err := client.Intents.List(ctx, &client.ListOptions{Limit: 10})
    if err != nil {
        return fmt.Errorf("failed to list intents: %w", err)
    }

    // Process results...
    _ = intents

    return nil
}
```

This comprehensive Go SDK guide provides production-ready code examples for all major API operations, error handling patterns, advanced features, and best practices for enterprise deployments.
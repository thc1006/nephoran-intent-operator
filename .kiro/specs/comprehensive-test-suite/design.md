# Design Document

## Overview

The comprehensive test suite will provide complete test coverage for the Nephoran Intent Operator using the existing Ginkgo/Gomega framework. The design builds upon the current test infrastructure found in `pkg/controllers/suite_test.go` and extends it to cover all system components with both unit and integration tests.

The test suite will be organized into four main categories:
1. **Controller Tests**: Testing E2NodeSet and NetworkIntent controller reconcile logic
2. **Integration Tests**: End-to-end workflow testing across all components
3. **API Tests**: CRD validation and schema enforcement
4. **Service Tests**: Individual service component testing with mocking

## Architecture

### Test Infrastructure Components

#### 1. Enhanced Test Suite Foundation
- **Base Suite**: Extend existing `pkg/controllers/suite_test.go` to support multiple test categories
- **Environment Setup**: Utilize existing envtest configuration with additional CRD loading
- **Shared Utilities**: Create reusable test utilities for common operations
- **Mock Framework**: Implement comprehensive mocking for external dependencies

#### 2. Test Organization Structure
```
pkg/
├── controllers/
│   ├── suite_test.go (enhanced)
│   ├── e2nodeset_controller_test.go (new)
│   ├── networkintent_controller_test.go (enhanced)
│   └── integration_test.go (new)
├── api/
│   └── v1/
│       ├── api_validation_test.go (new)
│       └── crd_schema_test.go (new)
├── llm/
│   ├── client_test.go (enhanced)
│   └── llm_integration_test.go (enhanced)
├── git/
│   └── client_test.go (new)
└── testutils/
    ├── fixtures.go (new)
    ├── mocks.go (new)
    └── helpers.go (new)
```

#### 3. Mock Infrastructure
- **Git Client Mock**: Mock GitOps operations for testing
- **LLM Client Mock**: Mock LLM processor responses
- **Kubernetes Client Mock**: Mock Kubernetes API interactions
- **HTTP Server Mock**: Mock external service dependencies

## Components and Interfaces

### 1. Controller Test Components

#### E2NodeSet Controller Tests
- **Reconcile Loop Testing**: Verify scaling operations and status updates
- **Error Handling**: Test failure scenarios and recovery mechanisms
- **Resource Management**: Validate ConfigMap creation/deletion for simulated E2 nodes
- **Status Updates**: Ensure proper status reporting and condition management

#### NetworkIntent Controller Tests
- **Intent Processing**: Test LLM integration and parameter extraction
- **GitOps Integration**: Verify Git operations and deployment file generation
- **Retry Logic**: Test exponential backoff and retry mechanisms
- **Condition Management**: Validate status condition updates throughout processing

### 2. Integration Test Components

#### End-to-End Workflow Tests
- **Complete Pipeline**: Test from natural language intent to deployed resources
- **Multi-Component Interaction**: Verify communication between all system components
- **GitOps Flow**: Test complete GitOps workflow including repository operations
- **Error Propagation**: Ensure proper error handling across component boundaries

#### Cross-Component Tests
- **Controller Interaction**: Test interactions between different controllers
- **Service Dependencies**: Verify proper handling of service dependencies
- **Resource Lifecycle**: Test complete resource creation, update, and deletion cycles

### 3. API Validation Test Components

#### CRD Schema Tests
- **Schema Validation**: Test CRD schema enforcement and validation rules
- **Field Validation**: Verify required fields, data types, and constraints
- **Backward Compatibility**: Ensure schema changes maintain compatibility
- **OpenAPI Validation**: Test OpenAPI schema generation and validation

#### Resource Validation Tests
- **NetworkIntent Validation**: Test NetworkIntent resource validation
- **E2NodeSet Validation**: Test E2NodeSet resource validation
- **Status Field Validation**: Verify status field updates and constraints

### 4. Service Test Components

#### LLM Service Tests
- **Intent Processing**: Test natural language processing and parameter extraction
- **Response Validation**: Verify LLM response format and content validation
- **Error Handling**: Test timeout, retry, and failure scenarios
- **Performance Testing**: Validate processing time and resource usage

#### Git Service Tests
- **Repository Operations**: Test Git clone, commit, and push operations
- **File Management**: Verify deployment file generation and management
- **Authentication**: Test Git authentication and authorization
- **Conflict Resolution**: Test merge conflict handling and resolution

## Data Models

### Test Data Fixtures

#### NetworkIntent Test Fixtures
```go
type NetworkIntentFixture struct {
    Name      string
    Namespace string
    Intent    string
    Expected  NetworkIntentSpec
}

var NetworkIntentFixtures = []NetworkIntentFixture{
    {
        Name:      "upf-deployment",
        Namespace: "5g-core",
        Intent:    "Deploy UPF with 3 replicas",
        Expected: NetworkIntentSpec{
            Intent: "Deploy UPF with 3 replicas",
            Parameters: runtime.RawExtension{
                Raw: []byte(`{"replicas": 3, "type": "upf"}`),
            },
        },
    },
    // Additional fixtures...
}
```

#### E2NodeSet Test Fixtures
```go
type E2NodeSetFixture struct {
    Name      string
    Namespace string
    Replicas  int32
    Expected  E2NodeSetStatus
}

var E2NodeSetFixtures = []E2NodeSetFixture{
    {
        Name:      "test-e2nodeset",
        Namespace: "default",
        Replicas:  3,
        Expected: E2NodeSetStatus{
            ReadyReplicas: 3,
        },
    },
    // Additional fixtures...
}
```

### Mock Data Models

#### Mock LLM Responses
```go
type MockLLMResponse struct {
    Intent   string
    Response string
    Error    error
    Delay    time.Duration
}

var MockLLMResponses = map[string]MockLLMResponse{
    "upf-deployment": {
        Intent: "Deploy UPF with 3 replicas",
        Response: `{
            "type": "NetworkFunctionDeployment",
            "replicas": 3,
            "image": "upf:latest"
        }`,
        Error: nil,
        Delay: 100 * time.Millisecond,
    },
    // Additional mock responses...
}
```

## Error Handling

### Test Error Scenarios

#### Controller Error Tests
- **Reconcile Failures**: Test controller reconcile loop error handling
- **Resource Conflicts**: Test handling of resource conflicts and race conditions
- **External Service Failures**: Test behavior when external services are unavailable
- **Timeout Handling**: Test timeout scenarios and recovery mechanisms

#### Integration Error Tests
- **Partial Failures**: Test scenarios where some components succeed and others fail
- **Network Failures**: Test network connectivity issues and recovery
- **Authentication Failures**: Test authentication and authorization failures
- **Resource Exhaustion**: Test behavior under resource constraints

#### API Error Tests
- **Validation Failures**: Test API validation error responses
- **Schema Violations**: Test handling of schema validation failures
- **Permission Errors**: Test RBAC and permission error scenarios
- **Rate Limiting**: Test API rate limiting and throttling

### Error Recovery Testing

#### Retry Mechanism Tests
- **Exponential Backoff**: Verify proper exponential backoff implementation
- **Max Retry Limits**: Test maximum retry count enforcement
- **Retry Conditions**: Test which errors trigger retries vs immediate failures
- **State Consistency**: Ensure system state remains consistent during retries

## Testing Strategy

### Test Categories and Scope

#### Unit Tests
- **Individual Function Testing**: Test individual functions and methods in isolation
- **Mock Dependencies**: Use mocks for all external dependencies
- **Fast Execution**: Ensure unit tests execute quickly (< 100ms each)
- **High Coverage**: Target 90%+ code coverage for unit tests

#### Integration Tests
- **Component Integration**: Test integration between system components
- **Real Dependencies**: Use real Kubernetes API server via envtest
- **Moderate Execution Time**: Allow longer execution time (< 5s each)
- **Realistic Scenarios**: Test realistic usage scenarios and workflows

#### End-to-End Tests
- **Complete Workflows**: Test complete user workflows from start to finish
- **External Dependencies**: Include external service dependencies where possible
- **Longer Execution Time**: Allow extended execution time (< 30s each)
- **User Scenarios**: Focus on real user scenarios and use cases

### Test Execution Strategy

#### Parallel Execution
- **Test Isolation**: Ensure tests can run in parallel without interference
- **Resource Management**: Manage shared resources to prevent conflicts
- **Namespace Isolation**: Use unique namespaces for each test
- **Cleanup Procedures**: Implement proper cleanup to prevent test pollution

#### Windows Compatibility
- **Path Handling**: Use cross-platform path handling throughout tests
- **File Operations**: Ensure file operations work correctly on Windows
- **Process Management**: Handle process creation and management cross-platform
- **Environment Variables**: Use appropriate environment variable handling

#### Performance Testing
- **Benchmark Tests**: Include benchmark tests for performance-critical components
- **Resource Usage**: Monitor memory and CPU usage during tests
- **Scalability Testing**: Test system behavior under load
- **Performance Regression**: Detect performance regressions in CI/CD

### Test Data Management

#### Fixture Management
- **Centralized Fixtures**: Maintain test fixtures in centralized location
- **Fixture Factories**: Implement factory patterns for creating test data
- **Data Cleanup**: Ensure proper cleanup of test data after execution
- **Data Isolation**: Prevent test data from affecting other tests

#### Mock Management
- **Mock Lifecycle**: Manage mock creation, configuration, and cleanup
- **Mock Verification**: Verify mock interactions and expectations
- **Mock Reusability**: Create reusable mocks for common scenarios
- **Mock Documentation**: Document mock behavior and usage patterns

## Implementation Phases

### Phase 1: Foundation and Controller Tests
1. Enhance existing test suite infrastructure
2. Implement E2NodeSet controller tests
3. Enhance NetworkIntent controller tests
4. Create shared test utilities and mocks

### Phase 2: API and Service Tests
1. Implement CRD validation tests
2. Create API schema tests
3. Enhance LLM service tests
4. Implement Git service tests

### Phase 3: Integration and End-to-End Tests
1. Create integration test framework
2. Implement cross-component integration tests
3. Create end-to-end workflow tests
4. Add performance and benchmark tests

### Phase 4: Enhancement and Optimization
1. Optimize test execution performance
2. Enhance Windows compatibility
3. Add additional error scenarios
4. Improve test documentation and maintenance
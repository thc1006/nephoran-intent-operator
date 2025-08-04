# Nephoran Intent Operator - Comprehensive Fix Prompts for Claude Code

## Project Context Summary for Claude Code

The Nephoran Intent Operator is a production-ready Kubernetes operator that integrates Large Language Model (LLM) processing with O-RAN network function management. The project consists of 4 main services (llm-processor, nephio-bridge, oran-adaptor, rag-api) with 18 Go packages and comprehensive RAG pipeline implementation.

**Current Project Structure:**
- `api/v1/`: CRD definitions (NetworkIntent, E2NodeSet, ManagedElement)
- `cmd/`: Service entry points for 4 microservices
- `pkg/`: 18 core packages including controllers, llm, rag, security, monitoring
- `deployments/`: Kubernetes manifests and Kustomize overlays
- `scripts/`: Cross-platform automation scripts
- `tests/`: 40+ test files with Ginkgo framework

**Technology Stack:**
- Go 1.24.0 with controller-runtime v0.19.7
- Weaviate v1.26.0-rc.1 for vector database
- OpenTelemetry v1.37.0 for observability
- Prometheus for monitoring
- Multi-stage Docker builds with distroless images

## Priority 1: Security Hardening (Critical)

### 1.1 API Key Management and Secrets Security

```
SECURITY HARDENING: Enhanced API Key Management and Secrets Protection

**Context**: The project handles sensitive API keys for OpenAI, Weaviate, and other external services. Current implementation may expose keys in logs or configuration files.

**Files to modify:**
- pkg/llm/*.go (all LLM client files)
- pkg/rag/api.py
- cmd/llm-processor/main.go
- deployments/kustomize/base/ (all Kubernetes manifests)

**Requirements:**

1. **Implement Kubernetes Secret-based API Key Management:**
   - Replace all hardcoded API keys with Kubernetes Secret references
   - Add secret validation in initialization code
   - Implement automatic secret rotation support
   - Add secret encryption at rest configuration

2. **Enhance LLM Client Security:**
   - Add request/response sanitization to prevent sensitive data logging
   - Implement secure HTTP client with proper TLS verification
   - Add rate limiting and circuit breaker patterns
   - Create audit logging for all LLM API calls

3. **Secure Configuration Management:**
   - Move all sensitive configuration to Kubernetes Secrets
   - Add configuration validation and sanitization
   - Implement secure default configurations
   - Add configuration drift detection

4. **Container Security Hardening:**
   - Ensure all containers run as non-root users
   - Add security contexts with proper capabilities
   - Implement network policies for service isolation
   - Add Pod Security Standards compliance

**Expected Output:**
- Kubernetes Secret manifests for all API keys
- Updated deployment configurations with secret references
- Enhanced security contexts in all containers
- Audit logging configuration for sensitive operations
```

### 1.2 RBAC and Authorization Enhancement

```
RBAC SECURITY: Kubernetes RBAC Refinement and Authorization Enhancement

**Context**: Current RBAC permissions may be overly broad. Need to implement principle of least privilege with granular permissions.

**Files to modify:**
- config/rbac/ (all RBAC files)
- deployments/kustomize/base/rbac.yaml
- pkg/controllers/*_controller.go (RBAC annotations)

**Requirements:**

1. **Granular RBAC Implementation:**
   - Review and minimize all ClusterRole permissions
   - Implement resource-specific permissions (get, list, watch, create, update, patch, delete)
   - Add namespace-scoped roles where appropriate
   - Create service-specific ServiceAccounts

2. **Advanced Authorization:**
   - Implement admission controllers for custom validation
   - Add mutating webhooks for security policy enforcement
   - Create validating webhooks for configuration validation
   - Add OPA (Open Policy Agent) integration for policy enforcement

3. **Security Monitoring:**
   - Add RBAC violation monitoring and alerting
   - Implement audit logging for all authorization decisions
   - Create security dashboards for access patterns
   - Add anomaly detection for unusual access patterns

**Expected Output:**
- Refined RBAC configurations with minimal required permissions
- Service-specific ServiceAccount configurations
- Admission controller implementations
- Security monitoring and alerting setup
```

## Priority 2: Dependency Management and Stability

### 2.1 Dependency Version Stabilization

```
DEPENDENCY STABILIZATION: Production-Ready Dependency Management

**Context**: Project uses weaviate v1.26.0-rc.1 (release candidate) and has 150+ dependencies that need security audit and stabilization.

**Files to modify:**
- go.mod
- go.sum
- requirements-rag.txt
- Dockerfile (all service Dockerfiles)

**Requirements:**

1. **Upgrade to Stable Versions:**
   - Replace weaviate v1.26.0-rc.1 with latest stable version
   - Update all dependencies to latest stable versions
   - Resolve any version conflicts and compatibility issues
   - Test all functionality after version updates

2. **Security Vulnerability Assessment:**
   - Run comprehensive vulnerability scans on all dependencies
   - Update dependencies with known security vulnerabilities
   - Add automated dependency vulnerability monitoring
   - Create security update procedures

3. **Dependency Management Optimization:**
   - Remove unused dependencies to reduce attack surface
   - Add dependency pinning with exact versions
   - Implement automated dependency update testing
   - Create dependency approval workflow

4. **Go Module Optimization:**
   - Clean up go.mod and remove redundant replace directives
   - Optimize build times by reducing unnecessary dependencies
   - Add module verification and checksum validation
   - Implement private module repository if needed

**Expected Output:**
- Updated go.mod with stable dependency versions
- Vulnerability scan reports and remediation
- Automated dependency update pipeline
- Optimized build configuration
```

### 2.2 Build System Optimization

```
BUILD SYSTEM ENHANCEMENT: Performance and Reliability Optimization

**Context**: Current Makefile supports cross-platform builds but can be optimized for better performance and reliability.

**Files to modify:**
- Makefile
- Dockerfile (all service Dockerfiles) 
- .dockerignore
- scripts/build-*.sh

**Requirements:**

1. **Build Performance Optimization:**
   - Implement advanced Docker layer caching strategies
   - Add parallel build optimizations for multi-stage builds
   - Optimize Go build flags for production deployment
   - Add build artifact caching and reuse

2. **Cross-Platform Build Enhancement:**
   - Add support for ARM64 architecture
   - Implement universal binary builds where applicable
   - Add platform-specific optimization flags
   - Create automated multi-platform testing

3. **Build Reliability Improvements:**
   - Add comprehensive build validation and testing
   - Implement build reproducibility with hermetic builds
   - Add build failure analysis and recovery
   - Create build metrics and monitoring

4. **Security Integration:**
   - Add container image scanning in build pipeline
   - Implement SBOM (Software Bill of Materials) generation
   - Add code signing for build artifacts
   - Create secure build environment isolation

**Expected Output:**
- Optimized Makefile with improved performance
- Enhanced Dockerfiles with better caching
- Multi-platform build support
- Integrated security scanning and validation
```

## Priority 3: Production Monitoring and Observability

### 3.1 Advanced Monitoring Implementation

```
MONITORING ENHANCEMENT: Production-Grade Observability Stack

**Context**: Basic Prometheus monitoring exists but needs comprehensive dashboards, alerting, and SLO monitoring.

**Files to modify:**
- pkg/monitoring/*.go
- deployments/monitoring/
- pkg/controllers/*_controller.go (add metrics)
- cmd/*/main.go (add monitoring setup)

**Requirements:**

1. **Comprehensive Metrics Collection:**
   - Add business metrics for intent processing success/failure rates
   - Implement SLI/SLO monitoring for all critical operations
   - Add resource utilization metrics (CPU, memory, network, storage)
   - Create custom metrics for LLM API performance and costs

2. **Advanced Alerting System:**
   - Implement multi-level alerting (info, warning, critical)
   - Add intelligent alert routing based on severity and team
   - Create alert fatigue reduction with smart grouping
   - Add escalation procedures and on-call integration

3. **Production Dashboards:**
   - Create executive dashboards for business KPIs
   - Add operational dashboards for system health
   - Implement service-specific dashboards for troubleshooting
   - Create capacity planning and forecasting dashboards

4. **Distributed Tracing Enhancement:**
   - Add end-to-end tracing for intent processing workflows
   - Implement correlation IDs across all services
   - Add performance profiling and bottleneck identification
   - Create trace-based alerting for anomaly detection

**Expected Output:**
- Comprehensive Grafana dashboard suite
- Production-ready alerting configuration
- Enhanced metrics collection across all services  
- Distributed tracing implementation
```

### 3.2 Disaster Recovery and High Availability

```
DISASTER RECOVERY: High Availability and Business Continuity Implementation

**Context**: Project lacks comprehensive disaster recovery procedures and high availability configuration.

**Files to modify:**
- deployments/kustomize/overlays/ (all environments)
- pkg/recovery/ (create new package)
- scripts/disaster-recovery/ (create new directory)
- docs/disaster-recovery.md (create)

**Requirements:**

1. **High Availability Configuration:**
   - Implement multi-replica deployments with proper anti-affinity
   - Add automatic failover mechanisms for all services
   - Create cross-region deployment capabilities
   - Implement database clustering and replication for Weaviate

2. **Backup and Recovery Automation:**
   - Add automated backup procedures for all stateful components
   - Implement point-in-time recovery for configuration data
   - Create knowledge base backup and restoration procedures
   - Add automated backup validation and integrity checking

3. **Disaster Recovery Procedures:**
   - Create comprehensive runbooks for disaster scenarios
   - Implement automated disaster detection and response
   - Add business continuity planning and testing procedures
   - Create recovery time and recovery point objective monitoring

4. **Data Protection and Compliance:**
   - Implement encryption at rest and in transit
   - Add data retention and purging policies
   - Create audit trails for all data operations
   - Add compliance reporting and monitoring

**Expected Output:**
- Multi-region high availability deployment
- Automated backup and recovery system
- Comprehensive disaster recovery runbooks
- Data protection and compliance framework
```

## Priority 4: Performance Optimization

### 4.1 RAG System Performance Enhancement

```
RAG PERFORMANCE OPTIMIZATION: Vector Database and Processing Pipeline Enhancement

**Context**: RAG system processes large telecom documents and may experience performance bottlenecks during concurrent operations.

**Files to modify:**
- pkg/rag/*.go (all RAG components)
- pkg/rag/api.py
- pkg/rag/weaviate_pool.go
- cmd/rag-api/main.go

**Requirements:**

1. **Vector Database Optimization:**
   - Implement advanced connection pooling for Weaviate
   - Add query optimization and caching strategies
   - Create batch processing for embedding operations
   - Add index optimization and maintenance procedures

2. **Document Processing Enhancement:**
   - Implement streaming processing for large documents
   - Add parallel processing capabilities for document chunks
   - Create intelligent chunking with overlap optimization
   - Add document preprocessing and validation pipeline

3. **Caching Strategy Implementation:**
   - Add multi-level caching (Redis, in-memory, database)
   - Implement intelligent cache invalidation strategies
   - Create cache warming procedures for frequently accessed data
   - Add cache performance monitoring and optimization

4. **API Performance Optimization:**
   - Implement request batching and queuing
   - Add API response compression and optimization
   - Create connection reuse and HTTP/2 support
   - Add request/response time monitoring and alerting

**Expected Output:**
- Optimized RAG processing pipeline
- Enhanced vector database performance
- Intelligent caching implementation
- Performance monitoring and alerting
```

### 4.2 LLM Integration Performance

```
LLM PERFORMANCE: API Call Optimization and Cost Management

**Context**: LLM API calls are critical path operations that can impact user experience and operational costs.

**Files to modify:**
- pkg/llm/*.go (all LLM client files)
- pkg/controllers/networkintent_controller.go
- cmd/llm-processor/main.go

**Requirements:**

1. **API Call Optimization:**
   - Implement intelligent request batching and queuing
   - Add request deduplication and caching
   - Create adaptive timeout and retry strategies
   - Add request prioritization and throttling

2. **Cost Management:**
   - Implement token usage tracking and budgeting
   - Add cost alerts and spending controls
   - Create usage analytics and optimization recommendations
   - Add model selection optimization based on use case

3. **Error Handling and Resilience:**
   - Implement comprehensive error handling and recovery
   - Add circuit breaker patterns for API failures
   - Create graceful degradation for service outages
   - Add fallback mechanisms and alternative providers

4. **Performance Monitoring:**
   - Add detailed metrics for API performance and costs
   - Create performance dashboards and alerting
   - Implement SLA monitoring and reporting
   - Add capacity planning and forecasting

**Expected Output:**
- Optimized LLM client implementations
- Cost management and monitoring system
- Resilient error handling and recovery
- Performance analytics and optimization
```

## Priority 5: Code Quality and Maintainability

### 5.1 Code Refactoring and Standardization

```
CODE QUALITY ENHANCEMENT: Refactoring and Standardization Initiative

**Context**: Large codebase with 18 packages needs standardization, refactoring, and technical debt reduction.

**Files to modify:**
- pkg/*/*.go (review all packages for consistency)
- pkg/shared/ (consolidate common functionality)
- pkg/llm/interface.go (remove redundancy)

**Requirements:**

1. **Interface and Abstraction Cleanup:**
   - Consolidate redundant interfaces (e.g., llm/interface.go)
   - Create consistent abstraction layers across packages
   - Implement proper dependency injection patterns
   - Add interface documentation and usage examples

2. **Error Handling Standardization:**
   - Create consistent error types and handling patterns
   - Implement structured error logging and reporting
   - Add error correlation and tracing capabilities
   - Create error recovery and retry strategies

3. **Configuration Management Unification:**
   - Consolidate configuration management across services
   - Implement configuration validation and type safety
   - Add configuration hot-reloading capabilities
   - Create configuration documentation and examples

4. **Code Organization and Structure:**
   - Refactor packages for better separation of concerns
   - Implement consistent naming conventions
   - Add comprehensive code documentation
   - Create development guidelines and standards

**Expected Output:**
- Refactored and standardized codebase
- Consolidated configuration management
- Improved error handling and logging
- Enhanced code documentation and standards
```

### 5.2 Testing Enhancement and Coverage

```
TESTING IMPROVEMENT: Comprehensive Test Coverage and Quality Enhancement

**Context**: Current testing framework with 40+ test files needs enhancement for better coverage and reliability.

**Files to modify:**
- pkg/controllers/*_test.go
- pkg/llm/*_test.go  
- pkg/rag/*_test.go
- Add missing test files for untested packages

**Requirements:**

1. **Test Coverage Enhancement:**
   - Achieve >90% code coverage across all packages
   - Add integration tests for complex workflows
   - Create end-to-end tests for complete user scenarios
   - Add performance and load testing capabilities

2. **Test Quality Improvement:**
   - Implement proper test isolation and cleanup
   - Add comprehensive test data management
   - Create realistic test scenarios and edge cases
   - Add test documentation and maintenance procedures

3. **Test Automation and CI/CD:**
   - Add automated test execution in CI pipeline
   - Implement test result reporting and analysis
   - Create test failure analysis and debugging tools
   - Add test performance monitoring and optimization

4. **Advanced Testing Strategies:**
   - Add chaos engineering and fault injection testing
   - Implement security testing and vulnerability assessment
   - Create compliance and regulatory testing
   - Add test environment management and provisioning

**Expected Output:**
- Comprehensive test suite with >90% coverage
- Automated testing pipeline with CI/CD integration
- Advanced testing strategies implementation
- Test quality and maintenance procedures
```

## Implementation Priority and Execution Plan

### Phase 1 (Immediate - Week 1-2): Critical Security Fixes
1. API Key Management and Secrets Security
2. RBAC and Authorization Enhancement  
3. Dependency Version Stabilization

### Phase 2 (Short Term - Week 3-4): Stability and Performance
1. Build System Optimization
2. RAG System Performance Enhancement
3. LLM Integration Performance

### Phase 3 (Medium Term - Week 5-6): Observability and Recovery
1. Advanced Monitoring Implementation
2. Disaster Recovery and High Availability

### Phase 4 (Long Term - Week 7-8): Quality and Maintainability  
1. Code Refactoring and Standardization
2. Testing Enhancement and Coverage

## Success Metrics and Validation

- **Security**: Zero high-severity vulnerabilities, proper secret management
- **Performance**: <2s intent processing, >99.9% availability
- **Quality**: >90% test coverage, standardized code practices
- **Observability**: Complete monitoring coverage, automated alerting
- **Reliability**: Automated disaster recovery, validated backup procedures

Each prompt is designed to be self-contained and provides complete context for Claude Code to understand the current implementation and execute the necessary improvements.
# Configuration Changes: Input Sanitization and Configurable Constants

This document summarizes the security improvements and configurability enhancements made to the Nephoran Intent Operator.

## Overview

The changes implement comprehensive input sanitization and make all hard-coded constants configurable via environment variables. This improves both security and operational flexibility.

## Changes Made

### 1. Created `pkg/config/constants.go`

**Purpose**: Centralized configuration management with environment variable support.

**Key Features**:
- All hard-coded constants extracted and made configurable
- Environment variable parsing with fallbacks to sensible defaults
- Comprehensive validation of all configuration values
- Support for all data types: strings, integers, floats, durations, arrays
- Configuration printing utility for debugging

**Configuration Categories**:
- **Controller Configuration**: retry limits, timeouts, git paths
- **Security Configuration**: input/output length limits, allowed domains, blocked keywords
- **Resilience Configuration**: circuit breaker thresholds, failure rates, recovery timeouts
- **Timeout Configuration**: operation-specific timeout values
- **Resource Limits**: CPU/memory defaults for deployments
- **Monitoring Configuration**: metrics ports, health check intervals

**Environment Variables Added**:
```bash
# Controller Configuration
NEPHORAN_MAX_RETRIES=3
NEPHORAN_RETRY_DELAY=30s
NEPHORAN_TIMEOUT=5m
NEPHORAN_GIT_DEPLOY_PATH=networkintents

# Security Configuration
NEPHORAN_MAX_INPUT_LENGTH=10000
NEPHORAN_MAX_OUTPUT_LENGTH=100000
NEPHORAN_ALLOWED_DOMAINS=kubernetes.io,3gpp.org,o-ran.org
NEPHORAN_BLOCKED_KEYWORDS=exploit,hack,backdoor

# Circuit Breaker Configuration  
NEPHORAN_CB_FAILURE_THRESHOLD=5
NEPHORAN_CB_RECOVERY_TIMEOUT=60s
NEPHORAN_CB_REQUEST_TIMEOUT=30s

# Timeout Configuration
NEPHORAN_LLM_TIMEOUT=30s
NEPHORAN_GIT_TIMEOUT=60s
NEPHORAN_KUBERNETES_TIMEOUT=30s

# Resource Defaults
NEPHORAN_CPU_REQUEST_DEFAULT=100m
NEPHORAN_MEMORY_REQUEST_DEFAULT=128Mi
```

### 2. Created `pkg/config/validation.go`

**Purpose**: Comprehensive validation framework for all configuration values.

**Key Features**:
- **Type-safe Validation**: Validates ports, timeouts, percentages, URLs, file paths
- **Kubernetes Resource Validation**: Validates resource quantity formats (100m, 1Gi)
- **Security Validation**: Domain name format, URL format, file path safety
- **Boundary Checking**: Ensures values are within acceptable operational ranges
- **Descriptive Error Messages**: Clear feedback on validation failures

**Validation Rules Implemented**:
- Port numbers (1-65535)
- Timeouts (positive durations)  
- Retry counts (0-100)
- Percentages/failure rates (0.0-1.0)
- Circuit breaker thresholds (1-1000)
- Input/output length limits (with safety bounds)
- File paths (prevents directory traversal)
- Domain names (format validation)
- URLs (format validation)
- Kubernetes resource quantities

### 3. Enhanced Input Sanitization

**Security Improvements**:
- **LLM Sanitizer Integration**: Uses existing comprehensive sanitization system
- **Configuration-Driven**: All sanitization parameters now configurable
- **Runtime Validation**: Both input and output validation with configurable limits
- **Logging Integration**: Security events logged with correlation IDs

**Sanitization Features** (leverages existing `pkg/security/llm_sanitizer.go`):
- 40+ regex patterns for prompt injection detection
- Configurable blocked keywords
- Input/output length validation
- Context boundary enforcement to prevent prompt confusion
- Malicious manifest detection (privileged containers, host access, etc.)
- Cryptocurrency mining detection
- Data exfiltration pattern blocking

### 4. Updated `cmd/main.go`

**Integration Improvements**:
- **Configuration Loading**: Loads and validates configuration at startup
- **Early Validation**: Fails fast with clear error messages for invalid config
- **Debug Support**: `--print-config` flag to display current configuration
- **Logging Enhancement**: Comprehensive startup logging showing all configuration values

### 5. Updated Controllers

**Controller Enhancements**:
- **Dynamic Configuration**: All controllers use configurable constants instead of hardcoded values
- **Configurable Backoff**: Exponential backoff calculations use configurable parameters
- **Configurable Finalizers**: Even finalizer names are configurable
- **Security Integration**: LLM processor fully integrated with configurable sanitization

## Security Benefits

### Critical Security Improvements

1. **Input Validation**: All user input sanitized before LLM processing
2. **Output Validation**: All LLM output validated before execution  
3. **Configuration Security**: Dangerous configuration values rejected at startup
4. **Boundary Enforcement**: Clear context boundaries prevent prompt confusion
5. **Resource Limits**: Configurable limits prevent resource exhaustion attacks

### OWASP Compliance

The changes address multiple OWASP Top 10 categories:
- **A03:2021 - Injection**: Comprehensive input sanitization  
- **A04:2021 - Insecure Design**: Defense-in-depth configuration
- **A05:2021 - Security Misconfiguration**: Configuration validation
- **A09:2021 - Security Logging**: Comprehensive audit logging

## Operational Benefits

### Deployment Flexibility

1. **Environment-Specific Configuration**: Different values per environment
2. **Runtime Tuning**: Adjust behavior without code changes
3. **Resource Optimization**: Tune resource limits per cluster capacity
4. **Monitoring Customization**: Configure ports and intervals per deployment

### Operational Excellence

1. **Configuration Validation**: Prevents misconfigurations at startup
2. **Observability**: Clear logging of all configuration values
3. **Debugging Support**: Print configuration utility for troubleshooting
4. **Documentation**: Self-documenting through validation error messages

## Example Usage

### Basic Deployment
```bash
# Use defaults
./nephoran-operator
```

### Production Deployment
```bash
# Secure production configuration
export NEPHORAN_MAX_RETRIES=5
export NEPHORAN_LLM_TIMEOUT=60s
export NEPHORAN_MAX_INPUT_LENGTH=5000
export NEPHORAN_BLOCKED_KEYWORDS=exploit,hack,backdoor,admin,root
export NEPHORAN_CB_FAILURE_THRESHOLD=3

./nephoran-operator
```

### Debug Configuration
```bash
# Print current configuration and exit
./nephoran-operator --print-config
```

### Development Environment
```bash
# Relaxed limits for development
export NEPHORAN_MAX_INPUT_LENGTH=50000
export NEPHORAN_LLM_TIMEOUT=120s
export NEPHORAN_MAX_RETRIES=10

./nephoran-operator
```

## Backward Compatibility

- **Default Values**: All defaults match previous hardcoded values
- **Optional Configuration**: All environment variables are optional
- **Graceful Fallbacks**: Invalid configuration falls back to safe defaults where possible
- **Clear Error Messages**: Configuration errors provide clear guidance

## Testing

### Test Coverage

- **Unit Tests**: Configuration loading and validation
- **Integration Tests**: Environment variable parsing
- **Boundary Tests**: Edge cases and invalid values
- **Security Tests**: Injection attempt blocking

### Test Results
```bash
# Configuration tests pass
go test ./pkg/config/ -run "TestLoadConstants"    # PASS
go test ./pkg/config/ -run "TestValidateConstants" # PASS
```

## Conclusion

These changes significantly enhance both the security and operational flexibility of the Nephoran Intent Operator:

1. **Security**: Comprehensive input/output sanitization with configurable parameters
2. **Flexibility**: All operational parameters configurable via environment variables  
3. **Reliability**: Extensive validation prevents misconfigurations
4. **Maintainability**: Centralized configuration management
5. **Observability**: Clear logging and debugging support

The system now provides production-ready security while maintaining the flexibility needed for different deployment scenarios.
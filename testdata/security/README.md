# Security Test Data

This directory contains test data specifically designed to test security aspects of the conductor-loop implementation.

## Files

### Malicious Intent Files

- **path-traversal-intent.json**: Tests path traversal protection with directory traversal attempts in target fields
- **command-injection-intent.json**: Tests command injection prevention with shell metacharacters in fields
- **large-payload-intent.json**: Tests resource exhaustion protection with large data payloads
- **unicode-exploit-intent.json**: Tests handling of unicode control characters and potential exploits
- **malformed-json-intent.json**: Tests handling of malformed JSON that could cause parsing issues

### Mock Executables

- **mock-malicious-porch.bat**: Mock executable that simulates malicious behavior for testing security controls

## Security Test Categories

### 1. Path Traversal Tests
Tests protection against attempts to access files outside the intended directory structure using:
- Relative path traversal (`../../../etc/passwd`)
- Windows path traversal (`..\\..\\windows\\system32`)
- Absolute paths (`/etc/passwd`)
- URL-encoded traversal sequences

### 2. Command Injection Tests
Tests prevention of command injection through:
- Shell metacharacters (`;`, `|`, `&`, `&&`, `||`)
- Command substitution (`$(command)`, `` `command` ``)
- Process substitution (`<(command)`)

### 3. Input Validation Tests
Tests handling of malicious input data:
- Oversized payloads (buffer overflow attempts)
- Unicode control characters
- Null bytes and other special characters
- Malformed JSON structures
- Nested JSON bombs

### 4. Resource Exhaustion Tests
Tests protection against resource exhaustion:
- Large file processing
- Many simultaneous files
- Deep JSON nesting
- Memory exhaustion attempts

### 5. File System Security Tests
Tests secure file system operations:
- Symlink handling
- Special file types (FIFO, device files)
- File permission validation
- Directory traversal prevention

## Test Execution

These test files are used by the security test suites in:
- `cmd/conductor-loop/security_test.go`
- `internal/loop/security_unit_test.go`
- `internal/porch/executor_security_test.go`
- `cmd/conductor-loop/integration_security_test.go`

## Expected Behavior

All security tests should:
1. **Fail safely**: Malicious input should be rejected or handled gracefully without system compromise
2. **No path traversal**: Files should only be created in expected directories
3. **No command injection**: Shell commands should not be executed from user input
4. **Resource limits**: System resources should not be exhausted
5. **Audit trail**: Security events should be logged appropriately

## Security Principles Tested

1. **Defense in Depth**: Multiple layers of security validation
2. **Fail Secure**: System fails to a secure state when encountering malicious input
3. **Least Privilege**: Operations are performed with minimal required permissions
4. **Input Validation**: All user input is validated and sanitized
5. **Output Encoding**: Output is properly encoded to prevent injection attacks

## Running Security Tests

```bash
# Run all security tests
go test -v ./cmd/conductor-loop/ -run "Security|security"
go test -v ./internal/loop/ -run "Security|security"
go test -v ./internal/porch/ -run "Security|security"

# Run comprehensive security test suite
go test -v ./cmd/conductor-loop/ -run "TestComprehensiveSecuritySuite"

# Run with race detection
go test -race -v ./... -run "Security|security"
```

## Security Test Coverage

The security tests cover:
- ✅ Path traversal prevention
- ✅ Command injection prevention  
- ✅ Input validation and sanitization
- ✅ Resource exhaustion protection
- ✅ File system security
- ✅ Concurrent access security
- ✅ State management security
- ✅ Configuration validation security
- ✅ Error handling security
- ✅ Permission validation

## Compliance

These tests help ensure compliance with:
- OWASP Top 10 security risks
- CWE (Common Weakness Enumeration) categories
- Security best practices for file processing systems
- Kubernetes security standards
- O-RAN security guidelines

## Maintenance

Security test data should be:
- Reviewed regularly for new attack vectors
- Updated when new vulnerabilities are discovered
- Expanded to cover additional security scenarios
- Validated against real-world attack patterns
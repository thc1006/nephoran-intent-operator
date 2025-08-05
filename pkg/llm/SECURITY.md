# LLM Client TLS Security Implementation

## Overview

This document describes the security implementation for TLS verification control in the LLM client. The implementation follows security best practices and defense-in-depth principles to ensure production safety while allowing controlled flexibility for development environments.

## Security Design Principles

1. **Secure by Default**: TLS verification is always enabled by default
2. **Defense in Depth**: Multiple layers of security checks
3. **Explicit Opt-in**: Insecure behavior requires explicit configuration
4. **Audit Trail**: All security-relevant actions are logged
5. **Fail Securely**: Any security violation results in safe failure

## Implementation Details

### TLS Configuration

The client enforces the following TLS security settings:

- **Minimum TLS Version**: TLS 1.2
- **Cipher Suites**: Only secure ECDHE cipher suites with AES-GCM
- **Server Cipher Preference**: Enabled to prevent downgrade attacks

### Dual-Control Mechanism

To disable TLS verification (ONLY for development), both conditions must be met:

1. Set `SkipTLSVerification: true` in `ClientConfig`
2. Set environment variable `ALLOW_INSECURE_CLIENT=true`

If only one condition is met, the client will:
- Log a security violation
- Panic to prevent insecure operation

### Security Logging

The implementation provides comprehensive security logging:

- **Security Violations**: Logged as ERROR with full context
- **Insecure Operations**: Logged as WARN with recommendations
- **Audit Trail**: All TLS-related decisions are logged

## Usage Examples

### Secure Production Usage (Default)

```go
// Default secure configuration
client := NewClient("https://api.example.com")

// Or with explicit config
config := ClientConfig{
    APIKey:      "your-api-key",
    ModelName:   "gpt-4",
    MaxTokens:   2048,
    BackendType: "openai",
    Timeout:     60 * time.Second,
    // SkipTLSVerification is false by default
}
client := NewClientWithConfig("https://api.example.com", config)
```

### Development Usage (Insecure)

⚠️ **WARNING**: Only use in development/testing environments!

```go
// Step 1: Set environment variable
os.Setenv("ALLOW_INSECURE_CLIENT", "true")

// Step 2: Configure client
config := ClientConfig{
    APIKey:              "dev-key",
    ModelName:           "gpt-4",
    MaxTokens:           2048,
    BackendType:         "openai",
    Timeout:             60 * time.Second,
    SkipTLSVerification: true, // Explicitly request insecure mode
}

// This will log a security warning but allow the connection
client := NewClientWithConfig("https://dev.example.com", config)
```

## Security Checklist

- [ ] Never set `ALLOW_INSECURE_CLIENT=true` in production
- [ ] Always use HTTPS URLs for LLM endpoints
- [ ] Monitor security logs for violation attempts
- [ ] Regularly update TLS cipher suites as needed
- [ ] Consider using mutual TLS for additional security

## OWASP References

This implementation addresses the following OWASP concerns:

- **A02:2021 – Cryptographic Failures**: Enforces strong TLS configuration
- **A05:2021 – Security Misconfiguration**: Secure defaults with explicit opt-in
- **A09:2021 – Security Logging and Monitoring Failures**: Comprehensive security logging

## Testing

Run the security tests to verify the implementation:

```bash
go test -v ./pkg/llm -run TestTLSVerificationSecurity
```

## Migration Guide

For existing code using the LLM client:

1. **No changes needed** for secure usage - existing code continues to work
2. **For development environments** requiring self-signed certificates:
   - Add `SkipTLSVerification: true` to your config
   - Set `ALLOW_INSECURE_CLIENT=true` in your development environment
   - Never commit these settings to version control

## Security Recommendations

1. **Certificate Pinning**: Consider implementing certificate pinning for known endpoints
2. **Mutual TLS**: For high-security environments, implement mTLS
3. **Network Segmentation**: Run LLM clients in isolated network segments
4. **Regular Audits**: Review security logs regularly for anomalies
5. **Dependency Scanning**: Keep all dependencies updated

## Incident Response

If you encounter a security violation log:

1. **Investigate immediately** - someone may be attempting to bypass security
2. **Check environment variables** - ensure ALLOW_INSECURE_CLIENT is not set in production
3. **Review code changes** - look for unauthorized SkipTLSVerification settings
4. **Report to security team** - follow your organization's security incident process
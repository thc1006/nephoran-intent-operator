# NetworkIntent Admission Webhook

This package implements a comprehensive validating admission webhook for NetworkIntent resources, providing defense-in-depth validation beyond the CRD OpenAPI schema validation.

## Overview

The NetworkIntent admission webhook ensures that all NetworkIntent resources meet telecommunications domain requirements, security standards, and business logic rules before being persisted in the cluster. This provides an additional layer of protection and validation that goes beyond basic schema validation.

## Validation Layers

### 1. Basic Content Validation
- **Empty Content Check**: Ensures intent is not empty or only whitespace
- **Character Set Validation**: Allows only printable ASCII and common Unicode characters
- **Control Character Detection**: Prevents malicious control characters
- **Line Break Limits**: Prevents excessive line breaks (max 50) and tabs (max 20)

### 2. Security Validation
- **Script Injection Protection**: Detects and blocks script tags, JavaScript handlers
- **SQL Injection Prevention**: Identifies SQL injection patterns
- **Command Injection Detection**: Blocks shell command injection attempts
- **Path Traversal Prevention**: Prevents directory traversal attacks
- **Protocol Handler Blocking**: Blocks suspicious protocol handlers (file:, ftp:, etc.)
- **Encoded Content Detection**: Identifies potential base64-encoded malicious content
- **Special Character Limits**: Prevents excessive use of special characters

### 3. Telecommunications Relevance Validation
- **Keyword Matching**: Validates presence of telecommunications-specific keywords
- **Domain Coverage**: Covers 5G Core, O-RAN, network functions, and infrastructure terms
- **Minimum Requirements**: Requires at least 1 telecommunications keyword
- **Comprehensive Dictionary**: 80+ telecommunications terms including:
  - 5G Core: AMF, SMF, UPF, NSSF, NRF, UDM, etc.
  - O-RAN: RIC, xApp, rApp, E2, A1, O1, O2, etc.
  - Network Functions: VNF, CNF, network slicing, QoS, etc.
  - Infrastructure: Kubernetes, scaling, monitoring, etc.

### 4. Complexity and Structure Validation
- **Word Count Limits**: Maximum 300 words per intent
- **Sentence Limits**: Maximum 20 sentences per intent
- **Repetition Detection**: Prevents excessive consecutive word repetition (max 5)
- **Word Distribution**: Validates reasonable word length distribution
- **Average Length Checks**: Ensures reasonable average word length (max 20 chars)

### 5. Business Logic Validation
- **Action Verb Requirements**: Ensures intents contain actionable verbs
- **Contradiction Detection**: Identifies contradictory statements
- **Vagueness Prevention**: Rejects overly vague intents
- **Naming Conventions**: Validates Kubernetes naming best practices
- **Coherence Checks**: Ensures intent is logically coherent and actionable

## Configuration

### Default Complexity Rules
```go
DefaultComplexityRules = ComplexityRules{
    MaxWords:               300,  // Maximum number of words
    MaxSentences:          20,   // Maximum number of sentences
    MaxConsecutiveRepeats: 5,    // Maximum consecutive repeated words
    MinTelecomKeywords:    1,    // Minimum telecom keywords required
}
```

### Webhook Configuration
- **Path**: `/validate-nephoran-io-v1-networkintent`
- **Operations**: CREATE, UPDATE
- **Failure Policy**: Fail
- **Side Effects**: None
- **Timeout**: 30 seconds

## Deployment

### Prerequisites
1. **cert-manager**: For TLS certificate management
2. **Kubernetes 1.19+**: For admission webhook support
3. **ValidatingAdmissionWebhook**: API enabled

### Installation Steps

1. **Deploy Certificate Resources**:
   ```bash
   kubectl apply -f config/webhook/manifests.yaml
   ```

2. **Enable Webhook in Manager**:
   ```bash
   # Set environment variable or command flag
   export ENABLE_WEBHOOKS=true
   ```

3. **Generate and Install Certificates**:
   ```bash
   # cert-manager will automatically generate certificates
   # based on the Certificate resource in manifests.yaml
   ```

4. **Verify Deployment**:
   ```bash
   kubectl get validatingwebhookconfiguration nephoran-intent-operator-validating-webhook-configuration
   kubectl get certificates -n nephoran-intent-operator-system
   ```

## Testing

### Unit Tests
Run comprehensive unit tests covering all validation scenarios:
```bash
cd pkg/webhooks
go test -v ./...
```

### Integration Tests
Test with real Kubernetes resources:
```bash
# Apply valid intents (should succeed)
kubectl apply -f config/samples/webhook_deployment_example.yaml

# Test invalid intents (should be rejected)
kubectl apply -f - <<EOF
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: invalid-intent
  namespace: default
spec:
  intent: "Make me coffee"  # Should be rejected - no telecom keywords
EOF
```

### Validation Examples

#### Valid Intents ✅
- `"Deploy a high-availability AMF instance for production with auto-scaling"`
- `"Create a network slice for URLLC with 1ms latency requirements"`
- `"Configure QoS policies for enhanced mobile broadband services"`
- `"Setup O-RAN Near-RT RIC with xApp orchestration capabilities"`

#### Invalid Intents ❌
- `""` - Empty intent
- `"Make me coffee"` - No telecommunications keywords
- `"<script>alert('xss')</script>Deploy AMF"` - Security violation
- `"Deploy AMF; DROP TABLE users;"` - SQL injection attempt
- `"Do something with the network"` - Too vague
- `"Enable and disable AMF"` - Contradictory

## Security Features

### Multi-Layer Security
1. **Input Sanitization**: Character set and content validation
2. **Pattern Matching**: Regex-based malicious pattern detection
3. **Injection Prevention**: SQL, script, and command injection protection
4. **Path Security**: Directory traversal prevention
5. **Protocol Security**: Suspicious protocol handler blocking

### Security Patterns Detected
- Script injection: `<script>`, `javascript:`, `onload=`
- SQL injection: `UNION SELECT`, `DROP TABLE`, `OR 1=1`
- Command injection: `rm -rf`, `$(cmd)`, `| bash`
- Path traversal: `../`, `/etc/passwd`
- Encoded attacks: Base64 patterns, hex encoding

## Monitoring and Observability

### Metrics
The webhook exposes metrics for monitoring:
- Validation request counts
- Validation success/failure rates
- Processing latency
- Error categorization

### Logging
Structured logging provides detailed information:
- Request details (namespace, name, operation)
- Validation results and reasons
- Performance metrics
- Security event logging

### Health Checks
- Webhook endpoint health monitoring
- Certificate expiration tracking
- Service availability checks

## Troubleshooting

### Common Issues

1. **Certificate Problems**:
   ```bash
   # Check certificate status
   kubectl describe certificate nephoran-intent-operator-serving-cert -n nephoran-intent-operator-system
   
   # Check cert-manager logs
   kubectl logs -n cert-manager -l app=cert-manager
   ```

2. **Webhook Not Called**:
   ```bash
   # Verify webhook configuration
   kubectl describe validatingwebhookconfiguration nephoran-intent-operator-validating-webhook-configuration
   
   # Check service endpoints
   kubectl get endpoints -n nephoran-intent-operator-system
   ```

3. **Validation Failures**:
   ```bash
   # Check webhook logs
   kubectl logs -n nephoran-intent-operator-system -l control-plane=controller-manager
   
   # Test with verbose kubectl
   kubectl apply -f intent.yaml -v=8
   ```

### Debug Mode
Enable debug logging for detailed validation information:
```bash
# Set log level to debug
export LOG_LEVEL=debug
```

## Best Practices

### Intent Writing Guidelines
1. **Be Specific**: Clearly describe what should be deployed/configured
2. **Use Action Verbs**: Start with deploy, configure, setup, scale, etc.
3. **Include Context**: Specify environment, requirements, constraints
4. **Avoid Contradictions**: Don't include conflicting requirements
5. **Use Telecom Terms**: Include relevant telecommunications keywords

### Security Considerations
1. **Input Validation**: Never trust user input, validate everything
2. **Principle of Least Privilege**: Minimal required permissions
3. **Defense in Depth**: Multiple validation layers
4. **Audit Logging**: Log all validation decisions
5. **Regular Updates**: Keep security patterns updated

### Performance Optimization
1. **Efficient Validation**: Order checks by computational cost
2. **Early Termination**: Fail fast on obvious violations
3. **Caching**: Cache validation results where appropriate
4. **Timeout Management**: Set appropriate timeouts
5. **Resource Limits**: Monitor memory and CPU usage

## Future Enhancements

### Planned Features
1. **Machine Learning Integration**: AI-powered intent classification
2. **Context-Aware Validation**: Cluster-state-aware validation
3. **Policy Engine Integration**: Open Policy Agent (OPA) support
4. **Advanced Analytics**: Intent pattern analysis
5. **Multi-Language Support**: Intent validation in multiple languages

### Extensibility
The webhook is designed for extensibility:
- Pluggable validation rules
- Configurable complexity thresholds
- Custom security patterns
- Industry-specific keyword dictionaries
- Integration with external validation services

## Contributing

### Adding New Validation Rules
1. Create validation function following the pattern: `validate[Feature](input) error`
2. Add comprehensive unit tests
3. Update documentation and examples
4. Consider performance impact
5. Ensure security implications are reviewed

### Adding Security Patterns
1. Add pattern to `SecurityPatterns` slice
2. Include test cases for the pattern
3. Document the threat model
4. Consider false positive impact
5. Update threat intelligence regularly
# Endpoint Validation Package

This package provides comprehensive endpoint URL validation to prevent runtime DNS errors and configuration issues.

## Features

- **URL Format Validation**: Validates URL structure and scheme (http/https only)
- **DNS Resolution Check**: Optional DNS hostname resolution validation
- **Reachability Check**: Optional HTTP connectivity verification
- **Clear Error Messages**: Helpful error messages with configuration suggestions
- **Security**: Prevents SSRF by only allowing http/https schemes

## Usage

### Basic Validation

```go
import "github.com/thc1006/nephoran-intent-operator/pkg/validation"

// Simple validation with defaults (no DNS/reachability checks)
config := validation.DefaultValidationConfig()
err := validation.ValidateEndpoint("http://service:8080", "My Service", config)
if err != nil {
    log.Fatal(err)
}
```

### DNS Validation

```go
// Enable DNS resolution checking
config := &validation.ValidationConfig{
    ValidateDNS:          true,
    ValidateReachability: false,
    Timeout:              5 * time.Second,
}

err := validation.ValidateEndpoint("http://porch-server:8080", "Porch Server", config)
if err != nil {
    // Error will include DNS-specific messages like:
    // "Porch Server endpoint validation failed: cannot resolve hostname (DNS lookup failed)"
    log.Fatal(err)
}
```

### Multiple Endpoints

```go
endpoints := map[string]string{
    "A1 Mediator": "http://a1-mediator:8080",
    "LLM Service": "http://ollama:11434",
    "Porch Server": "http://porch-server:7007",
}

config := validation.DefaultValidationConfig()
errors := validation.ValidateEndpoints(endpoints, config)
for _, err := range errors {
    log.Printf("Validation error: %v", err)
}
```

### Startup Validation (main.go pattern)

```go
func main() {
    // Parse flags
    flag.StringVar(&a1Endpoint, "a1-endpoint", "", "A1 Mediator URL")
    flag.Parse()

    // Validate endpoints before starting services
    if err := validateEndpoints(); err != nil {
        log.Fatal(err)
    }

    // Start services...
}

func validateEndpoints() error {
    endpoints := map[string]string{
        "A1 Mediator": getA1Endpoint(), // From flag or env var
        "LLM Service": getLLMEndpoint(),
    }

    config := validation.DefaultValidationConfig()
    errors := validation.ValidateEndpoints(endpoints, config)

    if len(errors) > 0 {
        // Format errors with suggestions
        for _, err := range errors {
            serviceName := extractServiceName(err.Error())
            suggestion := validation.GetCommonErrorSuggestions(serviceName)
            log.Printf("%v\n  → %s", err, suggestion)
        }
        return fmt.Errorf("endpoint validation failed")
    }
    return nil
}
```

## Configuration Options

### ValidationConfig

```go
type ValidationConfig struct {
    // ValidateDNS enables DNS resolution checking
    // Default: false (DNS check can be slow)
    ValidateDNS bool

    // ValidateReachability enables HTTP connectivity checking
    // Default: false (requires service to be running)
    ValidateReachability bool

    // Timeout for reachability checks
    // Default: 5 seconds
    Timeout time.Duration
}
```

### Default Behavior

By default (using `DefaultValidationConfig()`):
- ✅ URL format validation (always enabled)
- ✅ Scheme validation (http/https only)
- ✅ Hostname presence check
- ❌ DNS resolution (disabled by default)
- ❌ HTTP reachability (disabled by default)

This ensures fast startup validation without requiring services to be running.

## Error Messages

### Invalid Scheme
```
A1 Mediator endpoint validation failed: unsupported URL scheme 'ftp' (only http/https allowed)
  → Set A1_MEDIATOR_URL environment variable or --a1-endpoint flag.
     Example: http://service-ricplt-a1mediator-http.ricplt:8080
```

### Missing Hostname
```
LLM Service endpoint validation failed: missing hostname in URL
  → Set LLM_PROCESSOR_URL environment variable or --llm-endpoint flag.
     Example: http://ollama-service:11434
```

### DNS Resolution Failure
```
Porch Server endpoint validation failed: cannot resolve hostname (DNS lookup failed):
lookup porch-server: no such host. Check if service is deployed and DNS is configured
  → Set PORCH_SERVER_URL environment variable or --porch-server flag.
     Example: http://porch-server:7007
```

### Connection Refused
```
RAG Service endpoint validation failed: endpoint unreachable (connection refused).
Check if service is running
  → Set RAG_API_URL environment variable. Example: http://rag-service:8000
```

## Command-Line Flags

When integrated into `cmd/main.go`, the following flags are available:

```bash
# Basic endpoint configuration
--a1-endpoint="http://a1-mediator:8080"
--llm-endpoint="http://ollama:11434"
--porch-server="http://porch-server:7007"

# Enable DNS validation at startup (may slow startup)
--validate-endpoints-dns
```

## Environment Variables

Endpoints can also be configured via environment variables:

```bash
# A1 Mediator
export A1_MEDIATOR_URL="http://service-ricplt-a1mediator-http.ricplt:8080"
export A1_ENDPOINT="http://a1-mediator:8080"  # Alternative

# LLM Service
export LLM_PROCESSOR_URL="http://ollama-service:11434"
export LLM_ENDPOINT="http://ollama:11434"  # Alternative

# Porch Server
export PORCH_SERVER_URL="http://porch-server:7007"

# RAG Service
export RAG_API_URL="http://rag-service:8000"
```

**Precedence**: Command-line flags > Environment variables

## Security Considerations

### SSRF Prevention

The validator only allows `http://` and `https://` schemes, preventing SSRF attacks via:
- `file://` URLs (local file access)
- `ftp://` URLs (FTP access)
- `ssh://` URLs (SSH access)
- `data://` URLs (data URIs)

### URL Validation

All URLs are parsed and validated before use:
```go
// This will fail validation:
validateEndpoint("file:///etc/passwd", "Malicious", config)
// Error: unsupported URL scheme 'file' (only http/https allowed)
```

## Testing

Run tests:
```bash
go test -v ./pkg/validation/...
```

Run with DNS validation (requires network):
```bash
go test -v ./pkg/validation/... -run "TestValidateEndpoint_DNSResolution"
```

## Integration Example

See `cmd/main.go` for full integration example with:
- Flag parsing
- Environment variable fallback
- Clear error messages
- Startup validation before controller initialization

## Performance

- **URL validation**: ~1μs per endpoint
- **DNS resolution**: ~10-100ms per hostname (first lookup)
- **HTTP reachability**: ~100ms-5s per endpoint (depends on timeout)

**Recommendation**: Use default config (no DNS/reachability) for fast startup, enable DNS checking with `--validate-endpoints-dns` flag when debugging configuration issues.

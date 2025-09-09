# LLM Provider System - Runbook

## Overview

The Nephoran Intent Operator now supports pluggable LLM providers for natural language processing. This system converts free-form text into structured NetworkIntent JSON that conforms to `docs/contracts/intent.schema.json`.

## Supported Providers

- **OFFLINE** - Deterministic rule-based parsing (default)
- **OPENAI** - OpenAI GPT models (stub implementation)
- **ANTHROPIC** - Anthropic Claude models (stub implementation)

## Environment Variables

### Required Configuration

```bash
# Provider Selection (defaults to OFFLINE if not set)
export LLM_PROVIDER="OFFLINE"    # Options: OFFLINE, OPENAI, ANTHROPIC

# LLM Configuration (for OPENAI/ANTHROPIC providers)
export LLM_API_KEY="your-api-key-here"
export LLM_MODEL="gpt-4o-mini"          # Provider-specific model
export LLM_TIMEOUT="30"                 # Timeout in seconds (default: 30)
export LLM_MAX_RETRIES="3"              # Max retry attempts (default: 3)

# OpenAI Specific (optional - uses LLM_API_KEY if not set)
export OPENAI_API_KEY="sk-your-openai-key"

# Anthropic Specific (optional - uses LLM_API_KEY if not set)  
export ANTHROPIC_API_KEY="sk-ant-your-anthropic-key"
```

### Service Configuration

```bash
# Service Settings (existing)
export PORT="8080"
export LOG_LEVEL="info"
export AUTH_ENABLED="false"     # Set to true for production
export TLS_ENABLED="false"      # Set to true for production
```

## Quick Start

### 1. OFFLINE Provider (No API Keys Required)

```bash
# Set environment
export LLM_PROVIDER="OFFLINE"

# Start the service
make llm-offline-demo

# Test with curl (in another terminal)
curl -X POST http://localhost:8080/api/v2/process-intent \
  -H "Content-Type: application/json" \
  -d '{"intent": "scale nf-sim to 5 in ns ran-a"}'
```

Expected response:
```json
{
  "status": "success",
  "result": {
    "intent_type": "scaling",
    "target": "nf-sim", 
    "namespace": "ran-a",
    "replicas": 5,
    "source": "user",
    "reason": "Processed offline scaling request using pattern: scale TARGET to NUMBER in namespace NAMESPACE",
    "created_at": "2025-09-04T...",
    "status": "pending",
    "priority": 5
  },
  "processing_time": "1.234ms",
  "provider_metadata": {
    "provider": "OFFLINE",
    "model": "deterministic-v1",
    "confidence": 1.0,
    "tokens_used": 0
  },
  "timestamp": "2025-09-04T..."
}
```

### 2. OpenAI Provider (Stub)

```bash
# Set environment
export LLM_PROVIDER="OPENAI"
export LLM_API_KEY="sk-your-openai-key"
export LLM_MODEL="gpt-4o-mini"

# Start the service
go run cmd/llm-processor/main.go

# Test
curl -X POST http://localhost:8080/api/v2/process-intent \
  -H "Content-Type: application/json" \
  -d '{"intent": "increase O-DU replicas to 8 due to high traffic"}'
```

### 3. Anthropic Provider (Stub)

```bash
# Set environment  
export LLM_PROVIDER="ANTHROPIC"
export LLM_API_KEY="sk-ant-your-anthropic-key"
export LLM_MODEL="claude-3-haiku-20240307"

# Start the service
go run cmd/llm-processor/main.go

# Test
curl -X POST http://localhost:8080/api/v2/process-intent \
  -H "Content-Type: application/json" \
  -d '{"intent": "scale down 5G core AMF to 2 instances"}'
```

## Supported Intent Patterns

The OFFLINE provider recognizes these natural language patterns:

### Scaling Patterns

1. **Scale with target and number:**
   - "scale nf-sim to 3"
   - "scale O-DU to 5 in namespace oran-du"
   - "scale up RIC xApp to 8 replicas"

2. **Set replica patterns:**
   - "set nf-sim replicas to 4"
   - "set O-CU replicas to 6 in ns oran-cu"

3. **Increase/decrease patterns:**
   - "increase AMF to 10 instances"
   - "decrease SMF to 2 replicas in ns 5g-core"

### Network Function Recognition

The system recognizes O-RAN and 5G network function names:
- **O-RAN**: O-DU, O-CU, O-RU, RIC, xApp, E2SIM, VES, A1
- **5G Core**: AMF, SMF, UPF, PCF, AUSF, UDM, NRF, NSSF
- **Generic**: nf-sim, network-function, workload

### Namespace Detection

- Explicit: "in namespace ran-a", "in ns oran-du"  
- Implicit: Defaults to "default"
- Auto-detection: Recognizes standard O-RAN namespaces

## API Endpoints

### New Provider-Based Endpoint

```bash
POST /api/v2/process-intent
Content-Type: application/json

{
  "intent": "natural language scaling request"
}
```

### Legacy Endpoint (Still Supported)

```bash
POST /process-intent
Content-Type: application/json  

{
  "spec": {
    "intent": "natural language scaling request"
  }
}
```

### Health Checks

```bash
GET /healthz          # Health check
GET /readyz           # Readiness check  
GET /metrics          # Prometheus metrics (if enabled)
```

## Build and Deploy

### Development

```bash
# Build
go build ./cmd/llm-processor

# Run tests
go test ./internal/llm/providers/...
go test ./cmd/llm-processor/...

# Demo target
make llm-offline-demo
```

### Production

```bash
# Build production binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o llm-processor ./cmd/llm-processor

# Set production environment variables
export AUTH_ENABLED="true"
export TLS_ENABLED="true"
export LLM_PROVIDER="OPENAI"  # or ANTHROPIC
export LLM_API_KEY="your-production-api-key"
export TLS_CERT_PATH="/path/to/cert.pem"
export TLS_KEY_PATH="/path/to/key.pem"
export JWT_SECRET_KEY="your-jwt-secret"

# Run
./llm-processor
```

## Troubleshooting

### Common Issues

1. **"Provider not found" error:**
   ```bash
   export LLM_PROVIDER="OFFLINE"  # Valid options: OFFLINE, OPENAI, ANTHROPIC
   ```

2. **"API key validation failed":**
   ```bash
   # OpenAI keys start with 'sk-'
   export LLM_API_KEY="sk-your-openai-key"
   
   # Anthropic keys start with 'sk-ant-'
   export LLM_API_KEY="sk-ant-your-anthropic-key"
   ```

3. **"Schema validation failed":**
   - Check that the returned JSON has required fields: `intent_type`, `target`, `namespace`, `replicas`
   - Ensure replica count is between 1-100
   - Verify target and namespace follow Kubernetes naming conventions

4. **"Handoff file not created":**
   - Check write permissions in the handoff directory (default: `./handoff/`)
   - Verify the service has filesystem access
   - Look for error logs during intent processing

### Logs and Debugging

```bash
# Enable debug logging
export LOG_LEVEL="debug"

# Start service and check logs
go run cmd/llm-processor/main.go 2>&1 | jq '.'
```

### Validation

Verify your setup:

```bash
# Check provider initialization
curl http://localhost:8080/healthz

# Test provider response format
curl -X POST http://localhost:8080/api/v2/process-intent \
  -H "Content-Type: application/json" \
  -d '{"intent": "scale nf-sim to 3"}' | jq '.'

# Check handoff directory
ls -la ./handoff/
```

## Schema Reference

The output JSON conforms to `docs/contracts/intent.schema.json`:

### Required Fields
- `intent_type`: "scaling"
- `target`: Network function name (string, 1-253 chars)
- `namespace`: Kubernetes namespace (string, 1-63 chars)  
- `replicas`: Desired replica count (integer, 1-100)

### Optional Fields
- `reason`: Explanation for the scaling operation
- `source`: Source of the request ("user", "planner", "test")
- `correlation_id`: Tracking ID
- `priority`: Priority level (0-10, default: 5)
- `constraints`: Additional constraints object
- `created_at`/`updated_at`: ISO 8601 timestamps
- `status`: Current processing status

## Make Targets

```bash
make llm-offline-demo    # Start interactive demo with OFFLINE provider
make build-llm-processor # Build llm-processor binary
make test-llm-providers  # Run LLM provider tests
make validate-schema     # Validate JSON schemas
make ci-full            # Complete CI pipeline
```

## Integration with Intent-Ingest

The LLM processor works with the existing `cmd/intent-ingest` service:

1. **LLM Processor** processes natural language → structured JSON
2. **Intent Ingest** validates and processes structured NetworkIntent
3. **Handoff Directory** (`./handoff/`) transfers data between services
4. **Downstream Processing** handles the validated intents

Files are created as `./handoff/intent-{timestamp}.json` and consumed by the intent-ingest service.

## API Key Management

### Security Best Practices

1. **Never commit API keys to version control**
2. **Use environment variables or secret management systems**
3. **Rotate keys regularly**
4. **Use least-privilege API keys when possible**

### Key Formats

- **OpenAI**: `sk-proj-...` or `sk-...` (51+ characters)
- **Anthropic**: `sk-ant-api03-...` (varies)

## Support

For issues:
1. Check logs with `LOG_LEVEL=debug`
2. Verify environment variables are set correctly
3. Test with OFFLINE provider first (no API keys needed)
4. Validate JSON output against the schema
5. Check handoff directory permissions and content
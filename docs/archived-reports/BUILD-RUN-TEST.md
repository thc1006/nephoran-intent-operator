# BUILD-RUN-TEST Guide for porch-direct

## Overview

The `porch-direct` CLI tool reads an intent JSON file and interacts with the Porch API to create/update KRM packages for network function scaling.

## Prerequisites

- Go 1.24+ installed
- Access to a Porch API endpoint (or use dry-run mode)
- Valid intent JSON file (conforming to `docs/contracts/intent.schema.json`)

## Building

### Windows PowerShell
```powershell
# Build the CLI tool
go build -o porch-direct.exe ./cmd/porch-direct

# Or with optimizations
go build -ldflags="-s -w" -o porch-direct.exe ./cmd/porch-direct
```

### Linux/Mac
```bash
# Build the CLI tool
go build -o porch-direct ./cmd/porch-direct

# Or with optimizations
go build -ldflags="-s -w" -o porch-direct ./cmd/porch-direct
```

## Running

### Basic Usage

```bash
# Generate KRM package to filesystem (default mode)
./porch-direct --intent sample-intent.json

# Dry-run mode (writes to ./out/)
./porch-direct --intent sample-intent.json --dry-run

# Minimal package (Deployment + Kptfile only)
./porch-direct --intent sample-intent.json --minimal

# Submit to Porch API
./porch-direct --intent sample-intent.json \
  --porch http://localhost:9443 \
  --repo ran-packages \
  --package gnb-scaling \
  --workspace default \
  --namespace ran-sim

# Submit with auto-approval
./porch-direct --intent sample-intent.json \
  --porch http://localhost:9443 \
  --auto-approve
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--intent` | (required) | Path to intent JSON file |
| `--out` | `examples/packages/direct` | Output directory for generated KRM |
| `--dry-run` | `false` | Dry-run mode (no files written) |
| `--minimal` | `false` | Generate minimal package |
| `--repo` | (auto-resolved) | Target Porch repository |
| `--package` | (auto-resolved) | Target package name |
| `--workspace` | `default` | Workspace identifier |
| `--namespace` | `default` | Target namespace |
| `--porch` | `http://localhost:9443` | Porch API base URL |
| `--auto-approve` | `false` | Auto-approve package proposals |

### Sample Intent Files

Create a test intent file:

```json
{
  "intent_type": "scaling",
  "target": "gnb",
  "namespace": "ran-sim",
  "replicas": 3,
  "reason": "Increased traffic load",
  "source": "planner",
  "correlation_id": "test-001"
}
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./cmd/porch-direct
go test ./pkg/porch
go test ./internal/generator

# Run with verbose output
go test -v ./...
```

### Integration Tests

```bash
# Test dry-run mode
./porch-direct --intent test-intent.json --dry-run

# Verify output files
ls -la ./out/
cat ./out/porch-package-request.json
cat ./out/overlays/deployment.yaml
```

### E2E Test with Local Porch

```bash
# 1. Start local Porch server (if available)
# kubectl port-forward -n porch-system svc/porch-server 9443:443

# 2. Create test intent
cat > test-intent.json <<EOF
{
  "intent_type": "scaling",
  "target": "gnb",
  "namespace": "ran-sim",
  "replicas": 5,
  "source": "test"
}
EOF

# 3. Submit to Porch
./porch-direct --intent test-intent.json \
  --porch http://localhost:9443 \
  --repo test-repo \
  --package test-package

# 4. Verify package creation
# Check Porch UI or use kpt CLI
```

## Troubleshooting

### Common Issues

1. **"go.mod not found" error**
   - Ensure you're running from the project root
   - Check that go.mod exists in the repository

2. **"intent validation failed" error**
   - Verify intent JSON conforms to schema
   - Check required fields: intent_type, target, namespace, replicas

3. **"failed to connect to Porch" error**
   - Verify Porch URL is correct
   - Check network connectivity
   - Use --dry-run mode for testing without Porch

4. **"package not created" error**
   - Check Porch repository exists
   - Verify permissions to create packages
   - Review Porch server logs

### Debug Mode

```bash
# Enable verbose logging
export DEBUG=true
./porch-direct --intent sample-intent.json --dry-run

# Check generated files
find ./out -type f -exec echo {} \; -exec cat {} \;
```

## Development Workflow

1. **Make changes** to the code
2. **Build** the binary: `go build -o porch-direct ./cmd/porch-direct`
3. **Test** with sample intent: `./porch-direct --intent sample-intent.json --dry-run`
4. **Run tests**: `go test ./...`
5. **Submit PR** with test results

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test porch-direct
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: go build ./cmd/porch-direct
      - run: go test -cover ./...
      - run: ./porch-direct --intent sample-intent.json --dry-run
```

## Performance Testing

```bash
# Generate multiple intents
for i in {1..10}; do
  echo '{"intent_type":"scaling","target":"gnb'$i'","namespace":"test","replicas":'$i'}' > intent-$i.json
done

# Time execution
time for i in {1..10}; do
  ./porch-direct --intent intent-$i.json --dry-run
done
```

## Security Considerations

- Never commit Porch API credentials
- Use environment variables for sensitive data
- Validate all intent JSON inputs
- Run with minimal permissions
- Review generated KRM for security implications
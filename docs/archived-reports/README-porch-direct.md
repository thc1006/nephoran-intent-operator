# Porch Direct CLI - RUNBOOK

## Overview

The `porch-direct` CLI is a minimal Porch client that implements Nephio R5 PackageRevision lifecycle management. It reads network scaling intents from JSON files and creates corresponding KRM (Kubernetes Resource Model) packages in Porch repositories.

## Nephio R5 Concepts

### PackageRevision Lifecycle States

The CLI implements the complete Nephio R5 PackageRevision lifecycle:

1. **DRAFT** - Package created but not yet submitted for review
2. **REVIEW** - Package submitted and awaiting approval  
3. **APPROVED** - Package approved and ready for publishing
4. **PUBLISHED** - Package published and available for deployment

### Package Repository Structure

The CLI follows Nephio R5 package repository conventions:
- Repository: `mvp-repo` (configurable via `--repo`)
- Package path: `{repo}/{package-name}/{revision}`
- Default package: `ran-scale/nf-sim`

## Installation & Build

```bash
# Build the CLI
go build ./cmd/porch-direct

# Install (optional)
go install ./cmd/porch-direct
```

## Configuration

### Authentication

The CLI supports token-based authentication for Porch API access:

```bash
# Using service account token file
porch-direct --intent intent.json --token /var/run/secrets/kubernetes.io/serviceaccount/token

# Using kubeconfig token (extract manually)
kubectl config view --raw | jq -r '.users[0].user.token' > token.txt
porch-direct --intent intent.json --token token.txt
```

### Intent Schema

Intent JSON must follow the contract defined in `docs/contracts/intent.schema.json`:

```json
{
  "intent_type": "scaling",
  "target": "cnf-simulator",
  "namespace": "default", 
  "replicas": 5,
  "reason": "Load increase detected",
  "source": "planner",
  "correlation_id": "scale-001"
}
```

## Usage Examples

### Basic Usage

```bash
# Generate KRM package locally (no Porch interaction)
./porch-direct --intent examples/intent.json --out examples/packages/local

# Dry-run with Porch API
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443 --dry-run

# Submit to Porch (DRAFT state only)
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443
```

### Complete Lifecycle Automation

```bash
# Full automation: DRAFT → REVIEW → APPROVED → PUBLISHED
./porch-direct \
  --intent examples/intent.json \
  --porch-api http://localhost:9443 \
  --token /path/to/token \
  --auto-approve \
  --auto-publish
```

### Manual Lifecycle Control

```bash
# Step 1: Create package (DRAFT state)
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443

# Step 2: Submit for review (DRAFT → REVIEW) - happens automatically

# Step 3: Approve package (REVIEW → APPROVED)  
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443 --auto-approve

# Step 4: Publish package (APPROVED → PUBLISHED)
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443 --auto-approve --auto-publish
```

### Custom Repository and Package Names

```bash
# Use custom repository and package names
./porch-direct \
  --intent examples/intent.json \
  --porch-api http://localhost:9443 \
  --repo my-custom-repo \
  --package my-scaling-package \
  --namespace production
```

## Command Line Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--intent` | Path to intent JSON file | - | ✓ |
| `--porch-api` | Porch API base URL | `http://localhost:9443` | - |
| `--token` | Path to service account token file | - | - |
| `--repo` | Target Porch repository name | Auto-detected | - |
| `--package` | Target package name | Auto-generated | - |
| `--workspace` | Workspace identifier | `default` | - |
| `--namespace` | Target namespace | `default` | - |
| `--out` | Output directory for local generation | `examples/packages/direct` | - |
| `--dry-run` | Show what would be done without executing | `false` | - |
| `--minimal` | Generate minimal package (Deployment + Kptfile only) | `false` | - |
| `--auto-approve` | Automatically approve packages | `false` | - |
| `--auto-publish` | Automatically publish approved packages | `false` | - |

## Local Development Setup

### Prerequisites

- Go 1.24+
- Access to Nephio R5 Porch instance
- Service account token or kubeconfig with Porch permissions

### Setting up Local Porch (Kind Cluster)

```bash
# Create kind cluster
kind create cluster --name porch-demo

# Install Porch (refer to Nephio R5 installation guide)
kubectl apply -f https://github.com/nephio-project/porch/releases/download/v2.0.0/install.yaml

# Port-forward Porch API
kubectl port-forward -n porch svc/porch 9443:443
```

### Testing Against Local Porch

```bash
# Test with dry-run first
./porch-direct --intent pkg/porch/testdata/valid_intent.json --dry-run

# Test against local Porch instance  
./porch-direct \
  --intent pkg/porch/testdata/valid_intent.json \
  --porch-api http://localhost:9443 \
  --repo mvp-repo \
  --package test-scaling
```

## Generated KRM Package Structure

The CLI generates standard Nephio R5 KRM packages:

```
{package-name}/
├── Kptfile                    # Package metadata
├── deployment-patch.yaml      # Deployment scaling patch
├── kustomization.yaml         # Kustomize configuration
├── README.md                  # Package documentation
└── overlays/
    ├── deployment.yaml        # Deployment overlay
    └── configmap.yaml         # Intent ConfigMap
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   ```bash
   # Verify token is valid
   curl -H "Authorization: Bearer $(cat token.txt)" http://localhost:9443/api/v1/repositories
   ```

2. **Package Creation Fails**
   ```bash
   # Check Porch repository exists
   kubectl get repositories -n porch
   
   # Verify intent schema
   ./porch-direct --intent examples/intent.json --dry-run
   ```

3. **Connection Issues**
   ```bash
   # Test Porch API connectivity
   curl http://localhost:9443/healthz
   ```

### Debug Mode

Enable detailed logging:

```bash
export KLOG_V=2
./porch-direct --intent examples/intent.json --porch-api http://localhost:9443
```

## Integration with Nephio R5 Pipelines

The CLI integrates with standard Nephio R5 GitOps workflows:

1. **Intent Processing**: CLI processes natural language intents via LLM
2. **Package Generation**: Creates KRM packages following Nephio conventions  
3. **Repository Management**: Stores packages in Porch repositories
4. **Lifecycle Management**: Manages DRAFT → REVIEW → APPROVED → PUBLISHED states
5. **Deployment**: Published packages trigger GitOps deployment workflows

## API Reference

### Porch REST API Endpoints Used

- `GET /api/v1/repositories/{repo}/packages/{package}` - Get package
- `POST /api/v1/repositories/{repo}/packages` - Create package
- `PATCH /api/v1/packagerevisions/{name}` - Update revision status
- `POST /api/v1/packagerevisions/{name}/resources` - Add resources

### Intent JSON Schema

Refer to `docs/contracts/intent.schema.json` for the complete schema specification.

## Security Considerations

- Store service account tokens securely
- Use RBAC to limit Porch permissions
- Validate intent JSON before processing
- Audit package creation and approvals
- Use TLS for Porch API communication in production

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Run test suite (`go test ./...`)
5. Commit changes (`git commit -m 'Add amazing feature'`)
6. Push to branch (`git push origin feature/amazing-feature`)
7. Open Pull Request

## References

- [Nephio R5 Documentation](https://nephio.org/docs/v2.0/)
- [Porch API Reference](https://github.com/nephio-project/porch/tree/main/docs)
- [KRM Package Specification](https://googlecontainertools.github.io/kpt/guides/packages/)
- [Kustomize Documentation](https://kustomize.io/)
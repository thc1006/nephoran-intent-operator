# GitOps Migration Guide for Nephoran Intent Operator

## Overview

This guide provides instructions for migrating from the current deployment system to the optimized GitOps-friendly deployment pipeline.

## Key Improvements

### 1. **Idempotent Operations**
- All deployment operations are now idempotent (safe to run multiple times)
- No in-place modifications of Kustomize files
- Clean separation between source control and runtime configuration

### 2. **GitOps Compatibility**
- Declarative configuration via environment variables
- Version-controlled manifests without runtime modifications
- Support for ArgoCD, Flux, and other GitOps tools

### 3. **Enhanced Features**
- Multi-architecture support (amd64/arm64)
- Dry-run mode for validation
- Better error handling and validation
- Clean separation of build and deploy phases

## Migration Steps

### Step 1: Review Current State

```bash
# Check current deployment
kubectl get deployments -n nephoran-system

# Review current images
kubectl get deployments -n nephoran-system -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'
```

### Step 2: Update Scripts

Replace the old `deploy.sh` with the new optimized version:

```bash
# Backup old script
mv deploy.sh deploy-legacy.sh

# Use new optimized script
mv deploy-optimized.sh deploy.sh
chmod +x deploy.sh
```

### Step 3: Update Kustomize Structure

The new GitOps-friendly overlays are located in:
```
deployments/kustomize/overlays/gitops/
├── base/
├── local/
├── dev/
├── staging/
└── production/
```

### Step 4: Environment-Specific Deployment

#### Local Development
```bash
# Build and deploy locally
./deploy.sh --env local all

# Or with specific tag
./deploy.sh --env local --tag my-feature all
```

#### Development Environment
```bash
# Deploy to dev with registry push
./deploy.sh --env dev --tag dev-$(git rev-parse --short HEAD) all
```

#### Staging Environment
```bash
# Deploy specific version to staging
./deploy.sh --env staging --tag v1.2.0-rc1 deploy
```

#### Production Environment
```bash
# Production deployment (always use specific versions)
./deploy.sh --env production --tag v1.2.0 --dry-run deploy

# If dry-run looks good, deploy
./deploy.sh --env production --tag v1.2.0 deploy
```

## GitOps Integration

### ArgoCD Application Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nephoran-staging
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/yourusername/nephoran-intent-operator
    targetRevision: main
    path: deployments/kustomize/overlays/gitops/staging
  destination:
    server: https://kubernetes.default.svc
    namespace: nephoran-staging
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

### Flux Kustomization Example

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: nephoran-production
  namespace: flux-system
spec:
  interval: 10m
  path: "./deployments/kustomize/overlays/gitops/production"
  prune: true
  sourceRef:
    kind: GitRepository
    name: nephoran-intent-operator
  targetNamespace: nephoran-production
  validation: client
  postBuild:
    substitute:
      IMAGE_TAG: "v1.2.0"
```

## Environment Variables

Configure deployment behavior via environment variables:

```bash
# Set default registry
export NEPHORAN_REGISTRY=us-central1-docker.pkg.dev/myproject/nephoran

# Set default namespace
export NEPHORAN_NAMESPACE=nephoran-custom

# Set default tag
export NEPHORAN_TAG=latest
```

## CI/CD Integration

### GitHub Actions
```yaml
- name: Deploy to Staging
  run: |
    ./deploy.sh \
      --env staging \
      --tag ${{ github.sha }} \
      --registry ${{ secrets.REGISTRY }} \
      deploy
```

### Jenkins Pipeline
```groovy
stage('Deploy to Production') {
    steps {
        sh '''
            ./deploy.sh \
              --env production \
              --tag ${VERSION} \
              --dry-run deploy
        '''
        input 'Deploy to production?'
        sh '''
            ./deploy.sh \
              --env production \
              --tag ${VERSION} \
              deploy
        '''
    }
}
```

## Rollback Procedures

### Quick Rollback
```bash
# List recent deployments
kubectl rollout history deployment -n nephoran-production

# Rollback to previous version
kubectl rollout undo deployment llm-processor -n nephoran-production

# Rollback to specific revision
kubectl rollout undo deployment llm-processor --to-revision=3 -n nephoran-production
```

### GitOps Rollback
```bash
# Revert Git commit
git revert <commit-hash>
git push

# GitOps controller will automatically sync
```

## Validation and Testing

### Pre-deployment Validation
```bash
# Dry run to see what would be deployed
./deploy.sh --env staging --dry-run deploy

# Validate Kustomize output
kustomize build deployments/kustomize/overlays/gitops/staging
```

### Post-deployment Validation
```bash
# Check deployment status
kubectl get all -n nephoran-staging

# Verify pod health
kubectl get pods -n nephoran-staging -o wide

# Check recent events
kubectl get events -n nephoran-staging --sort-by='.lastTimestamp'
```

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check image pull secrets
   kubectl get secret nephoran-gcr-secret -n nephoran-staging
   
   # Verify registry authentication
   gcloud auth configure-docker us-central1-docker.pkg.dev
   ```

2. **Resource Constraints**
   ```bash
   # Check resource quotas
   kubectl describe resourcequota -n nephoran-production
   
   # Check node resources
   kubectl top nodes
   ```

3. **Network Policy Issues**
   ```bash
   # List network policies
   kubectl get networkpolicy -n nephoran-production
   
   # Debug with temporary allow-all policy
   kubectl apply -f debug-network-policy.yaml
   ```

## Best Practices

1. **Version Management**
   - Always use specific tags in production
   - Use semantic versioning (v1.2.3)
   - Tag images with Git commit SHA for traceability

2. **Security**
   - Review network policies before deployment
   - Ensure pod security policies are applied
   - Validate image signatures in production

3. **Monitoring**
   - Set up alerts for deployment failures
   - Monitor resource usage after deployment
   - Track deployment frequency and success rate

4. **Documentation**
   - Document any environment-specific configurations
   - Keep runbooks updated
   - Maintain change logs for production deployments

## Cleanup Old Resources

After successful migration:

```bash
# Remove old kustomization files with runtime modifications
rm -rf deployments/kustomize/overlays/local/kustomization.yaml.bak
rm -rf deployments/kustomize/overlays/remote/kustomization.yaml.bak

# Archive legacy scripts
mkdir -p legacy
mv deploy-legacy.sh legacy/
mv deploy-windows.ps1 legacy/
mv deploy-cross-platform.ps1 legacy/
```

## Support

For issues or questions:
1. Check deployment logs: `kubectl logs -n <namespace> <pod>`
2. Review this guide and troubleshooting section
3. Contact the platform team for assistance
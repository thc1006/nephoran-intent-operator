# Webhook Deployment with cert-manager

This directory contains the Kubernetes manifests for deploying the Nephoran Intent Operator admission webhooks with automated TLS certificate management using cert-manager.

## Overview

The webhook deployment consists of:

- **Mutating Webhook**: Provides default values for NetworkIntent resources
- **Validating Webhook**: Validates NetworkIntent resources according to business rules
- **cert-manager Integration**: Automated TLS certificate provisioning and CA injection

## Architecture

### Trust Model

The webhook deployment uses a hierarchical certificate trust model:

1. **Root CA (Self-Signed)**: `nephoran-ca-cert` - Created by cert-manager using a self-signed issuer
2. **Intermediate CA Issuer**: `nephoran-ca-issuer` - Uses the root CA to issue webhook certificates
3. **Webhook Server Certificate**: `webhook-serving-cert` - Server certificate for webhook TLS

### Certificate Lifecycle

- **Root CA**: Valid for 1 year, renewed 30 days before expiration
- **Webhook Certificate**: Valid for 90 days, renewed 10 days before expiration
- **Automatic Renewal**: cert-manager handles all certificate renewals automatically

## Files Structure

```
config/webhook/
├── README.md                    # This documentation
├── kustomization.yaml          # Main kustomization file
├── serviceaccount.yaml         # Service account for webhook manager
├── service.yaml                # Service exposing webhook endpoints
├── deployment.yaml             # Webhook manager deployment
├── mutating-webhook.yaml       # MutatingWebhookConfiguration
├── validating-webhook.yaml     # ValidatingWebhookConfiguration
└── rbac-patch.yaml            # RBAC permissions for webhook
```

## Prerequisites

### 1. cert-manager Installation

The webhook deployment requires cert-manager to be installed in the cluster:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=300s
```

### 2. Required Tools

- `kubectl` - Kubernetes command-line tool
- `kustomize` - Kubernetes configuration management tool
- `docker` - For building webhook container images

## Deployment

### Quick Deployment

Use the provided Makefile target for one-command deployment:

```bash
make deploy-webhook
```

This will:
1. Build the webhook manager Docker image
2. Verify cert-manager is installed
3. Apply the kustomize configuration
4. Wait for certificates and deployment to be ready

### Manual Deployment

1. **Build the webhook image**:
   ```bash
   make docker-build-webhook
   ```

2. **Deploy using kustomize**:
   ```bash
   kustomize build config/default | kubectl apply -f -
   ```

3. **Verify deployment**:
   ```bash
   kubectl get certificates -n nephoran-system
   kubectl get deployment webhook-manager -n nephoran-system
   ```

## Configuration

### Environment Variables

The webhook manager supports the following environment variables:

- `WEBHOOK_PORT`: Port for webhook server (default: 9443)
- `CERT_DIR`: Directory containing TLS certificates (default: /tmp/k8s-webhook-server/serving-certs)
- `METRICS_BIND_ADDRESS`: Address for metrics endpoint (default: :8080)
- `HEALTH_PROBE_BIND_ADDRESS`: Address for health probes (default: :8081)

### Security Configuration

The deployment follows security best practices:

- **Pod Security**: Runs as non-root user (UID 65532)
- **Filesystem**: Read-only root filesystem
- **Capabilities**: All capabilities dropped
- **seccomp**: Runtime default profile
- **Network Policies**: Restricted communication (when enabled)

### Resource Limits

Default resource configuration:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 64Mi
```

## Webhook Endpoints

The webhook manager exposes the following endpoints:

### Mutating Webhook
- **Path**: `/mutate-intent-v1alpha1-networkintent`
- **Purpose**: Apply default values to NetworkIntent resources
- **Operations**: CREATE, UPDATE

### Validating Webhook
- **Path**: `/validate-intent-v1alpha1-networkintent`
- **Purpose**: Validate NetworkIntent resources
- **Operations**: CREATE, UPDATE, DELETE

### Health Endpoints
- **Liveness**: `/healthz` (port 8081)
- **Readiness**: `/readyz` (port 8081)
- **Metrics**: `/metrics` (port 8080)

## Validation Rules

The validating webhook enforces the following rules:

1. **Intent Type**: Must be "scaling"
2. **Replicas**: Must be >= 0
3. **Target**: Must be non-empty string
4. **Namespace**: Must be non-empty string

## Default Values

The mutating webhook applies these defaults:

- **Source**: Set to "user" if not specified

## Troubleshooting

### Common Issues

#### 1. Certificate Not Ready

**Symptoms**: Webhook pods failing to start, certificate not in Ready state

**Diagnosis**:
```bash
kubectl describe certificate webhook-serving-cert -n nephoran-system
kubectl describe issuer nephoran-ca-issuer -n nephoran-system
```

**Solutions**:
- Verify cert-manager is running: `kubectl get pods -n cert-manager`
- Check cert-manager logs: `kubectl logs -n cert-manager -l app=cert-manager`
- Ensure DNS names in certificate match service name and namespace

#### 2. Webhook Connection Refused

**Symptoms**: API server cannot connect to webhook, admission failures

**Diagnosis**:
```bash
kubectl logs -n nephoran-system -l app=webhook-manager
kubectl describe mutatingwebhookconfiguration nephoran-networkintent-mutating
```

**Solutions**:
- Verify webhook service is available: `kubectl get svc webhook-manager -n nephoran-system`
- Check webhook deployment status: `kubectl get deployment webhook-manager -n nephoran-system`
- Validate CA bundle injection: Check that `caBundle` field is populated in webhook configurations

#### 3. CA Bundle Not Injected

**Symptoms**: Empty `caBundle` in webhook configurations

**Diagnosis**:
```bash
kubectl get mutatingwebhookconfiguration nephoran-networkintent-mutating -o yaml | grep caBundle
```

**Solutions**:
- Verify cert-manager.io/inject-ca-from annotation is correct
- Check certificate is in Ready state
- Restart cert-manager if needed: `kubectl rollout restart deployment/cert-manager -n cert-manager`

### Debug Commands

```bash
# Check overall webhook health
kubectl get all -n nephoran-system

# View webhook logs
kubectl logs -n nephoran-system -l app=webhook-manager -f

# Test webhook connectivity
kubectl run test-pod --image=curlimages/curl --rm -it -- \
  curl -k https://webhook-manager.nephoran-system:9443/readyz

# View certificate details
kubectl get certificate webhook-serving-cert -n nephoran-system -o yaml

# Check webhook configurations
kubectl get mutatingwebhookconfiguration nephoran-networkintent-mutating -o yaml
kubectl get validatingwebhookconfiguration nephoran-networkintent-validating -o yaml
```

## Testing

### End-to-End Testing

Run comprehensive E2E tests:

```bash
./hack/run-e2e.sh
```

### Manual Testing

1. **Test validation (should fail)**:
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: intent.nephoran.io/v1alpha1
   kind: NetworkIntent
   metadata:
     name: test-invalid
   spec:
     intentType: "invalid"
     replicas: -1
   EOF
   ```

2. **Test defaulting and validation (should succeed)**:
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: intent.nephoran.io/v1alpha1
   kind: NetworkIntent
   metadata:
     name: test-valid
   spec:
     intentType: "scaling"
     replicas: 3
     target: "my-deployment"
     namespace: "default"
   EOF
   ```

3. **Verify defaults were applied**:
   ```bash
   kubectl get networkintent test-valid -o yaml | grep source
   ```

## Monitoring

### Metrics

The webhook exposes Prometheus metrics on port 8080:

- `webhook_admission_requests_total`: Total admission requests processed
- `webhook_admission_request_duration_seconds`: Request processing duration
- `webhook_admission_failures_total`: Total admission failures

### Alerts

Consider setting up alerts for:

- Certificate expiration (< 7 days)
- Webhook admission failures (> 5% error rate)
- Webhook unavailability (> 30 seconds)

## Security Considerations

### Trust Boundaries

- **API Server ↔ Webhook**: Mutual TLS using cert-manager issued certificates
- **cert-manager ↔ Webhook**: CA injection via Kubernetes annotations
- **Network Isolation**: Consider implementing NetworkPolicies for additional security

### Certificate Rotation

- Certificates are automatically rotated by cert-manager
- No manual intervention required for normal operations
- Monitor certificate expiration dates in production

### RBAC Permissions

The webhook requires minimal RBAC permissions:

- Read/write access to NetworkIntent resources
- Read access to webhook configurations (for health checks)
- Event creation for audit logging

## Production Considerations

### High Availability

For production deployments, consider:

- Multiple webhook replicas (minimum 2)
- Pod anti-affinity rules
- Resource quotas and limits
- Horizontal Pod Autoscaler (HPA)

### Backup and Recovery

- Certificate backup: cert-manager stores certificates as Kubernetes secrets
- Configuration backup: Include webhook configurations in cluster backup
- Recovery: Standard Kubernetes restore procedures apply

### Updates and Maintenance

- Rolling updates supported via Kubernetes deployments
- Zero-downtime updates with multiple replicas
- Health checks ensure traffic only routes to ready pods

## References

- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Kubernetes Admission Controllers](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
- [controller-runtime Webhook Documentation](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html)
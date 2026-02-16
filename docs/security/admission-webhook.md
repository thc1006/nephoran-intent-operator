# NetworkIntent Admission Webhooks

This document describes the admission webhook implementation for the NetworkIntent CRD.

## Overview

The NetworkIntent resource uses both **Mutating** and **Validating** admission webhooks to ensure data consistency and enforce business rules. The webhooks are processed in the following order:

1. **Mutating Webhook** - Applies defaults and normalizes the resource
2. **Validating Webhook** - Validates the resource against business rules

## Webhook Behavior

### Mutating Webhook (Defaulting)

The mutating webhook applies the following defaults:

- **`spec.source`**: Defaults to `"user"` if not specified

### Validating Webhook (Validation)

The validating webhook enforces these rules:

- **`spec.intentType`**: Must be `"scaling"` (only supported type)
- **`spec.replicas`**: Must be >= 0 (negative values rejected)
- **`spec.target`**: Must be non-empty
- **`spec.namespace`**: Must be non-empty
- **`spec.source`**: Must be one of: `"user"`, `"planner"`, `"test"`

## Quick Verification

### Prerequisites

1. Install the CRD:
   ```bash
   kubectl apply -f deployments/crds/intent.nephoran.com_networkintents.yaml
   ```

2. Deploy the webhook manager:
   ```bash
   make deploy-webhook
   # Or manually:
   kubectl apply -k config/default/
   ```

### Test Examples

We provide two example manifests to demonstrate webhook behavior:

#### Valid NetworkIntent (Demonstrates Defaulting)

```bash
# Apply valid intent without source field
kubectl apply -f examples/admission/ok-scale.yaml

# Verify source was defaulted to "user"
kubectl get networkintent valid-scale-intent -o jsonpath='{.spec.source}'
# Output: user
```

#### Invalid NetworkIntent (Demonstrates Validation)

```bash
# Attempt to apply invalid intent with negative replicas
kubectl apply -f examples/admission/bad-neg.yaml
# Error from server (Forbidden): error validating data: 
# ValidationError(NetworkIntent.spec.replicas): invalid value: -1, 
# must be >= 0
```

### Automated Verification

Run the verification script to test all webhook behaviors:

```bash
# Run admission webhook tests
./hack/verify-admission.sh

# Expected output:
# ✓ Valid NetworkIntent accepted and source defaulted to 'user'
# ✓ Invalid NetworkIntent rejected with correct error message
# ✓ Webhook order verification completed
# ✓ ALL TESTS PASSED
```

## Webhook Implementation Details

### Code Structure

- **Types**: `api/intent/v1alpha1/networkintent_types.go`
- **Webhook Logic**: `api/intent/v1alpha1/webhook.go`
- **Tests**: `api/intent/v1alpha1/webhook_test.go`
- **Manager**: `cmd/webhook-manager/main.go`

### Webhook Paths

- **Mutating**: `/mutate-intent-nephoran-com-v1alpha1-networkintent`
- **Validating**: `/validate-intent-nephoran-com-v1alpha1-networkintent`

### TLS Configuration

The webhook uses cert-manager for TLS certificate management:

- Self-signed issuer for development
- Certificate mounted at `/tmp/k8s-webhook-server/serving-certs/`
- Service: `webhook-service.nephoran-system.svc:9443`

## Troubleshooting

### Check Webhook Deployment

```bash
# Check webhook manager is running
kubectl get deployment webhook-manager -n nephoran-system

# Check webhook configurations
kubectl get mutatingwebhookconfigurations
kubectl get validatingwebhookconfigurations

# View webhook logs
kubectl logs deployment/webhook-manager -n nephoran-system
```

### Common Issues

1. **Webhook not called**: Ensure webhook configurations point to the correct service
2. **TLS errors**: Check cert-manager is installed and certificates are valid
3. **Defaulting not working**: Verify mutating webhook is registered and comes before validating webhook
4. **Validation too strict**: Review validation rules in `webhook.go`

## Development

### Running Tests

```bash
# Unit tests
go test ./api/intent/v1alpha1 -v

# E2E verification
./hack/verify-admission.sh
```

### Modifying Webhooks

1. Update webhook logic in `api/intent/v1alpha1/webhook.go`
2. Regenerate manifests: `make manifests`
3. Run tests: `go test ./api/intent/v1alpha1 -v`
4. Deploy changes: `make deploy-webhook`
5. Verify: `./hack/verify-admission.sh`
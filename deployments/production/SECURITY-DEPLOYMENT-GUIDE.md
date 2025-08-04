# Security-Hardened Deployment Guide

This guide provides step-by-step instructions for deploying the Nephoran Intent Operator with comprehensive security hardening.

## Prerequisites

1. Kubernetes cluster v1.28+
2. kubectl configured with cluster access
3. Kustomize v5.0+
4. cert-manager installed (for webhook certificates)
5. External secrets operator (recommended for production)

## Step 1: Create Namespace with Pod Security Standards

```bash
kubectl create namespace nephoran-system
kubectl label namespace nephoran-system \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

## Step 2: Install cert-manager (if not already installed)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

## Step 3: Generate Webhook Certificates

```bash
# Create CA and certificate for webhook
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: nephoran-webhook-cert
  namespace: nephoran-system
spec:
  secretName: nephoran-webhook-certs
  issuerRef:
    name: nephoran-ca-issuer
    kind: Issuer
  dnsNames:
  - nephoran-webhook-service.nephoran-system.svc
  - nephoran-webhook-service.nephoran-system.svc.cluster.local
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: nephoran-ca-issuer
  namespace: nephoran-system
spec:
  selfSigned: {}
EOF
```

## Step 4: Create Production Secrets

### Option A: Manual Secret Creation (Development Only)

```bash
# Create actual secrets - DO NOT commit these values!
kubectl create secret generic llm-api-keys \
  --namespace=nephoran-system \
  --from-literal=openai-api-key="YOUR_ACTUAL_OPENAI_KEY" \
  --from-literal=mistral-api-key="YOUR_ACTUAL_MISTRAL_KEY" \
  --from-literal=anthropic-api-key="YOUR_ACTUAL_ANTHROPIC_KEY" \
  --from-literal=google-api-key="YOUR_ACTUAL_GOOGLE_KEY" \
  --from-literal=api-key="YOUR_ACTUAL_API_KEY"

kubectl create secret generic jwt-secret \
  --namespace=nephoran-system \
  --from-literal=jwt-secret-key="$(openssl rand -base64 32)"

kubectl create secret generic oauth2-credentials \
  --namespace=nephoran-system \
  --from-literal=client-id="YOUR_OAUTH_CLIENT_ID" \
  --from-literal=client-secret="YOUR_OAUTH_CLIENT_SECRET" \
  --from-literal=auth-url="https://your-auth-provider.com/oauth2/authorize" \
  --from-literal=token-url="https://your-auth-provider.com/oauth2/token"
```

### Option B: External Secrets Operator (Recommended for Production)

```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets-system \
  --create-namespace

# Apply external secrets configuration
kubectl apply -f deployments/production/external-secrets-example.yaml
```

## Step 5: Deploy Security-Hardened Configuration

```bash
# Deploy using Kustomize overlay
kubectl apply -k deployments/kustomize/overlays/security-hardened/

# Wait for webhook to be ready
kubectl wait --for=condition=ready pod \
  -l app=nephoran-webhook \
  -n nephoran-system \
  --timeout=300s
```

## Step 6: Verify Security Configuration

```bash
# Check namespace labels
kubectl get namespace nephoran-system -o yaml | grep pod-security

# Verify RBAC
kubectl auth can-i --list \
  --as=system:serviceaccount:nephoran-system:llm-processor \
  -n nephoran-system

# Check network policies
kubectl get networkpolicies -n nephoran-system

# Verify webhook configuration
kubectl get validatingwebhookconfigurations nephoran-validating-webhook
```

## Step 7: Test Security Controls

### Test 1: RBAC Restrictions

```bash
# This should fail - service account can't create deployments
kubectl auth can-i create deployments \
  --as=system:serviceaccount:nephoran-system:llm-processor \
  -n nephoran-system
```

### Test 2: Network Policy

```bash
# Deploy a test pod and verify it can't communicate
kubectl run test-pod --image=busybox -n nephoran-system -- sleep 3600
kubectl exec -it test-pod -n nephoran-system -- wget -qO- http://llm-processor:8080
# Should timeout due to network policy
```

### Test 3: Webhook Validation

```bash
# Try to create an invalid ManagedElement
cat <<EOF | kubectl apply -f -
apiVersion: nephoran.com/v1
kind: ManagedElement
metadata:
  name: invalid-element
  namespace: nephoran-system
spec:
  deploymentName: test
  host: test.example.com
  credentials:
    username: "plaintext-username"  # Should be rejected
    password: "plaintext-password"  # Should be rejected
EOF
```

## Step 8: Configure Monitoring

```bash
# Deploy security scanner
kubectl apply -f deployments/production/security-scanner.yaml

# Check security events
kubectl get events -n nephoran-system --field-selector reason=FailedValidation
```

## Security Checklist

- [ ] Namespace has Pod Security Standards labels
- [ ] All secrets are created using external secret management
- [ ] Webhook certificates are properly configured
- [ ] Network policies are enforced
- [ ] RBAC follows least privilege principle
- [ ] All containers run as non-root
- [ ] Resource limits are configured
- [ ] Security scanning is enabled
- [ ] Audit logging is configured
- [ ] Backup and disaster recovery plan exists

## Troubleshooting

### Webhook Issues

```bash
# Check webhook logs
kubectl logs -n nephoran-system -l app=nephoran-webhook

# Verify certificate
kubectl get certificate -n nephoran-system nephoran-webhook-cert -o yaml
```

### Secret Loading Issues

```bash
# Check if secrets are mounted
kubectl exec -n nephoran-system deployment/llm-processor -- ls -la /secrets/

# Verify secret content (be careful with sensitive data)
kubectl exec -n nephoran-system deployment/llm-processor -- cat /secrets/llm/openai-api-key
```

### Network Policy Issues

```bash
# Test connectivity
kubectl run debug --image=nicolaka/netshoot -n nephoran-system -it --rm
```

## Production Best Practices

1. **Secret Rotation**: Implement automated secret rotation every 90 days
2. **Audit Logging**: Enable Kubernetes audit logging for security events
3. **Backup**: Regular backup of secrets and configurations
4. **Monitoring**: Set up alerts for security violations
5. **Compliance**: Regular security audits and penetration testing
6. **Updates**: Keep all components updated with security patches

## Security Contacts

- Security Team: security@nephoran.com
- Incident Response: incident-response@nephoran.com
- On-call: +1-xxx-xxx-xxxx
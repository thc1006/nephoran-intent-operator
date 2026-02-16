# Kubernetes 1.35 Production Deployment Security Checklist

**Version**: 1.0
**Date**: 2026-02-16
**Applicability**: Nephoran Intent Operator on K8s 1.35 with controller-runtime 0.23.1

---

## Pre-Deployment Validation

### 1. Pod Security Standards

- [ ] Verify `nephoran-system` namespace enforces `restricted` Pod Security Standard
- [ ] Confirm Pod Security Standards version is set to `v1.35` or `latest`
- [ ] Validate all deployments have explicit `securityContext` at pod and container level
- [ ] Confirm `runAsNonRoot: true` on all pods
- [ ] Confirm `readOnlyRootFilesystem: true` on all containers (except Weaviate with justification)
- [ ] Confirm `allowPrivilegeEscalation: false` on all containers
- [ ] Confirm `capabilities.drop: [ALL]` on all containers
- [ ] Verify `seccompProfile.type: RuntimeDefault` on all pods
- [ ] Document any exception capabilities (e.g., NET_RAW for O-RAN adaptor)
- [ ] Remove all PodSecurityPolicy (PSP) references (deprecated since K8s 1.21, removed in 1.25)

### 2. RBAC

- [ ] No wildcard (`*`) in apiGroups, resources, or verbs in any ClusterRole
- [ ] No `cluster-admin` ClusterRoleBinding for application service accounts
- [ ] Secret access restricted to named secrets via `resourceNames`
- [ ] Webhook registration RBAC restricted to specific webhook names via `resourceNames`
- [ ] ConfigMap access scoped to specific namespace (not cluster-wide)
- [ ] Leader election uses Lease objects (not ConfigMaps)
- [ ] No `escalate` or `bind` verbs for non-admin roles
- [ ] Service accounts use dedicated accounts (not `default`)
- [ ] `automountServiceAccountToken: false` where token is not needed

### 3. Network Policies

- [ ] Default deny-all policy exists for each namespace
- [ ] All egress rules specify explicit destination IP blocks or namespace selectors (not `to: []`)
- [ ] Health check ingress restricted to kube-system namespace or kubelet IP ranges
- [ ] Inter-pod communication restricted to specific ports (not port range 1-65535)
- [ ] External HTTPS egress limited to known API endpoints
- [ ] DNS egress (port 53) restricted to kube-dns/CoreDNS IP
- [ ] All pods covered by at least one network policy
- [ ] O-RAN interface ports (830, 38000, 36421, 36422) restricted to known RIC addresses

### 4. Webhook Validation

- [ ] All webhook endpoints use TLS with certificate rotation
- [ ] `failurePolicy: Fail` set on validating webhooks
- [ ] `sideEffects: None` set on all webhooks
- [ ] Webhook validation logic is consistent across all API versions (v1, v1alpha1)
- [ ] Character allowlist blocks injection characters (`<`, `>`, `"`, `'`, backtick, `$`, `\`)
- [ ] Command injection patterns checked in all webhook paths
- [ ] SQL injection patterns checked in all webhook paths
- [ ] XSS patterns checked in all webhook paths
- [ ] Error messages do not reveal internal validation logic
- [ ] Upper bound on replicas enforced (configurable, default 1000)
- [ ] Maximum intent length enforced (1000 characters)

### 5. TLS / Encryption

- [ ] TLS 1.3 minimum for all internal communication
- [ ] Strong cipher suites only (AES-GCM, ChaCha20-Poly1305)
- [ ] All certificates have at least 2048-bit RSA or 256-bit ECDSA keys
- [ ] Certificate expiration monitoring in place (alert at 30 days)
- [ ] mTLS configured for service-to-service communication
- [ ] Webhook TLS certificates auto-rotated (cert-manager recommended)
- [ ] No `InsecureSkipVerify: true` in production code
- [ ] All Ingress resources have TLS configured with HTTP-to-HTTPS redirect

### 6. Secrets Management

- [ ] All secrets encrypted at rest (Kubernetes EncryptionConfiguration)
- [ ] No plaintext credentials in ConfigMaps
- [ ] API keys stored as hashes (not plaintext)
- [ ] Secret rotation policy defined and enforced (90-day maximum)
- [ ] External secret management (Vault, CSP Secrets Manager) integration verified
- [ ] No hardcoded secrets in source code (run SecretValidator scan)
- [ ] Service account tokens have bounded lifetime

### 7. Container Security

- [ ] All container images use specific tags (not `latest`)
- [ ] Container images from approved registries only
- [ ] Container image vulnerability scan passes (no Critical/High CVEs)
- [ ] AppArmor or SELinux profiles applied
- [ ] Resource limits (CPU, memory) set on all containers
- [ ] Resource quotas set on namespace
- [ ] LimitRange configured for namespace
- [ ] No NodePort services (disabled via ResourceQuota)
- [ ] emptyDir volumes have sizeLimit set

### 8. Dependency Security

- [ ] `govulncheck` passes with no critical vulnerabilities
- [ ] `go mod tidy` produces clean output
- [ ] No deprecated dependencies in use (golang/mock, x509.IsEncryptedPEMBlock)
- [ ] All `k8s.io/*` packages at consistent v0.35.x versions
- [ ] Dependabot or Snyk enabled for automated CVE monitoring
- [ ] SBOM generated for deployment artifacts

### 9. Monitoring and Audit

- [ ] Audit logging implemented and writing to persistent storage
- [ ] Security events logged with structured data (user, action, resource, timestamp)
- [ ] Falco or runtime security monitoring deployed
- [ ] Prometheus alerts configured for security events
- [ ] Rate limiting active on all API endpoints
- [ ] Rate limiter is thread-safe (uses proper synchronization)
- [ ] Failed authentication attempts logged and alerted

### 10. Incident Response Readiness

- [ ] Incident response runbook documented
- [ ] Contact information for security team documented
- [ ] Backup and recovery procedure tested
- [ ] Network segmentation validated (blast radius containment)
- [ ] Forensic log collection procedure documented
- [ ] Security patch deployment process documented

---

## Post-Deployment Validation

- [ ] Run security validation suite: `go test ./tests/security/... -v`
- [ ] Run webhook security tests: `go test ./tests/security/k8s_135_webhook_security_test.go -v`
- [ ] Verify Pod Security Standards enforcement with a test pod
- [ ] Verify network policies by testing blocked connections
- [ ] Verify webhook rejection of malicious payloads
- [ ] Check audit log entries for deployment activities
- [ ] Validate certificate expiration dates
- [ ] Run `kubectl auth can-i` checks for each service account

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Security Auditor | | | |
| DevOps Lead | | | |
| Product Owner | | | |
| SRE On-Call | | | |

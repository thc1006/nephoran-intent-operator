---
name: Bug Report
about: Report a bug to help us improve the Nephoran Intent Operator
title: "[BUG] "
labels: ["bug", "needs-triage"]
assignees: []
---

## Bug Description

A clear and concise description of what the bug is.

## Steps to Reproduce

Please provide detailed steps to reproduce the behavior:

1. Go to '...'
2. Click on '...'
3. Execute command '...'
4. See error

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

A clear and concise description of what actually happened.

## Environment Information

**System Information:**
- OS: [e.g., macOS 12.0, Ubuntu 22.04, Windows 11]
- Architecture: [e.g., amd64, arm64]

**Software Versions:**
- Go version: [e.g., 1.23.0]
- Kubernetes version: [e.g., 1.28.0] 
- Docker version: [e.g., 24.0.0]
- Nephoran version: [e.g., v0.1.0]

**Cluster Information:**
- Kubernetes distribution: [e.g., kind, minikube, EKS, GKE, AKS]
- Node count: [e.g., 3]
- Resource constraints: [e.g., CPU/Memory limits]

## Error Logs

Please include relevant log output. Use code blocks for formatting:

```
Insert error logs here
```

**Controller Logs:**
```bash
kubectl logs -n nephoran-system deployment/nephio-bridge --tail=100
```

**LLM Processor Logs:**
```bash  
kubectl logs -n nephoran-system deployment/llm-processor --tail=100
```

**System Events:**
```bash
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'
```

## Configuration Files

Please include relevant configuration files:

**NetworkIntent Resource:**
```yaml
# Include the NetworkIntent that caused the issue
```

**Values/Config:**
```yaml
# Include relevant Helm values or configuration
```

## Screenshots/Recordings

If applicable, add screenshots or screen recordings to help explain the problem.

## Impact Assessment

**Severity:** [Critical/High/Medium/Low]

**Impact Description:**
- How does this affect your usage?
- Is there a workaround available?
- How many users/systems are affected?

## Additional Context

Add any other context about the problem here:
- Does this happen consistently or intermittently?
- When did this first occur?
- Has this ever worked correctly?
- Any recent changes that might be related?

## Possible Solution

If you have ideas on how to fix this, please share them here.

## Checklist

Please confirm the following:

- [ ] I have searched the existing issues to make sure this hasn't been reported before
- [ ] I have provided sufficient information to reproduce the issue
- [ ] I have included relevant logs and error messages
- [ ] I have tested with the latest version of the software
- [ ] I have provided environment details

---

**Thank you for taking the time to report this bug! We'll investigate and respond as soon as possible.**
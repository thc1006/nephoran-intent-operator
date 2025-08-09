# Nephoran Intent Operator - Runbooks Index

**Version:** 3.0  
**Last Updated:** January 2025  
**Purpose:** Master index for all operational runbooks

## Quick Navigation

### 🚨 Emergency Response
- [Incident Response Runbook](./incident-response-runbook.md) - P0-P3 incident procedures
- [Disaster Recovery Runbook](./disaster-recovery-runbook.md) - DR procedures and RTO/RPO targets
- [Security Incident Response](./security-incident-response.md) - Security breach procedures

### 📊 Daily Operations
- [Master Operational Runbook](./operational-runbook-master.md) - Comprehensive daily/weekly/monthly procedures
- [Production Operations Runbook](./production-operations-runbook.md) - Production-specific operations

### 📈 Monitoring & Performance
- [Monitoring & Alerting Runbook](./monitoring-alerting-runbook.md) - Monitoring setup and alert response
- [Performance Degradation Runbook](./performance-degradation.md) - Performance troubleshooting
- [SLA Monitoring Guide](../sla-monitoring-system.md) - SLA tracking and validation

### 🔧 Troubleshooting
- [Troubleshooting Guide](./troubleshooting-guide.md) - Common issues and solutions
- [Network Intent Processing Failure](./network-intent-processing-failure.md) - Intent-specific issues

### 🔒 Security Operations
- [Security Operations Runbook](./security-operations-runbook.md) - Security monitoring and compliance
- [Security Hardening Guide](../security/hardening-guide.md) - Security best practices

### 🗄️ Specialized Components
- [Weaviate Operations](../../deployments/weaviate/OPERATIONAL-RUNBOOK.md) - Vector database operations
- [Weaviate Deployment](../../deployments/weaviate/DEPLOYMENT-RUNBOOK.md) - Weaviate deployment procedures

## Runbook Usage Guidelines

### When to Use Each Runbook

| Scenario | Primary Runbook | Supporting Documents |
|----------|----------------|---------------------|
| Service down/degraded | [Incident Response](./incident-response-runbook.md) | [Troubleshooting Guide](./troubleshooting-guide.md) |
| Daily health checks | [Master Operational](./operational-runbook-master.md) | [Monitoring & Alerting](./monitoring-alerting-runbook.md) |
| Performance issues | [Performance Degradation](./performance-degradation.md) | [SLA Monitoring](../sla-monitoring-system.md) |
| Security incident | [Security Incident Response](./security-incident-response.md) | [Security Operations](./security-operations-runbook.md) |
| Disaster recovery | [Disaster Recovery](./disaster-recovery-runbook.md) | [Production Operations](./production-operations-runbook.md) |
| Weaviate issues | [Weaviate Operations](../../deployments/weaviate/OPERATIONAL-RUNBOOK.md) | [Weaviate Deployment](../../deployments/weaviate/DEPLOYMENT-RUNBOOK.md) |

### Incident Severity Matrix

| Severity | Response Time | Resolution Time | Runbook |
|----------|--------------|-----------------|---------|
| P0 - Critical | 15 min | 4 hours | [Incident Response](./incident-response-runbook.md) |
| P1 - High | 30 min | 8 hours | [Incident Response](./incident-response-runbook.md) |
| P2 - Medium | 1 hour | 24 hours | [Master Operational](./operational-runbook-master.md) |
| P3 - Low | 4 hours | 72 hours | [Troubleshooting Guide](./troubleshooting-guide.md) |

## Quick Reference Commands

### System Health Check
```bash
# Quick system health check
kubectl get pods -n nephoran-system
kubectl get nodes -o wide
kubectl top nodes
kubectl get networkintents --all-namespaces
```

### Service Status Check
```bash
# Check all service endpoints
curl -k https://nephoran-api.company.com/health
curl -k https://llm-processor.nephoran-system:8080/healthz
curl -k https://rag-api.nephoran-system:5001/health
curl -k https://weaviate.nephoran-system:8080/v1/.well-known/ready
```

### Emergency Restart
```bash
# Emergency service restart (use with caution)
kubectl rollout restart deployment/nephoran-controller -n nephoran-system
kubectl rollout restart deployment/llm-processor -n nephoran-system
kubectl rollout restart deployment/rag-api -n nephoran-system
kubectl rollout status deployment --timeout=300s -n nephoran-system
```

## Runbook Maintenance

### Update Schedule
- **Weekly:** Review and update operational procedures based on incidents
- **Monthly:** Update troubleshooting guides with new issues
- **Quarterly:** Comprehensive review of all runbooks
- **Annually:** Major version update with lessons learned

### Contributing
To update or add runbooks:
1. Follow the standard runbook template
2. Include version and last updated date
3. Add cross-references to related runbooks
4. Update this index
5. Submit PR with review from operations team

### Archive Policy
Superseded runbooks are archived in `docs/runbooks/archive/` with:
- Archive date
- Reason for archival
- Reference to replacement runbook

## Contact Information

### On-Call Rotation
- **Primary:** Check PagerDuty schedule
- **Escalation:** Platform Team Lead → Engineering Manager → VP Engineering

### Communication Channels
- **Slack:** #nephoran-alerts (alerts), #nephoran-operations (discussion)
- **Email:** nephoran-ops@company.com
- **War Room:** https://meet.company.com/nephoran-incident

### External Support
- **OpenAI API:** support@openai.com
- **Weaviate:** support@weaviate.io
- **Cloud Provider:** [Provider-specific support]

## Related Documentation

### Operations Guides
- [Production Deployment Guide](../operations/01-production-deployment-guide.md)
- [Maintenance & Upgrade Procedures](../operations/04-maintenance-upgrade-procedures.md)
- [Performance Tuning](../operations/PERFORMANCE-TUNING.md)

### Architecture Documentation
- [System Architecture](../architecture.md)
- [LLM Processor Specifications](../LLM-Processor-Technical-Specifications.md)
- [RAG System Architecture](../RAG-System-Architecture.md)

### Compliance & Security
- [Compliance Audit Documentation](../operations/05-compliance-audit-documentation.md)
- [Security Implementation](../SECURITY-IMPLEMENTATION.md)
- [OAuth2 Security Guide](../OAuth2-Authentication-Guide.md)

---

**Note:** This is a living document. Report issues or suggestions to the operations team via Slack or email.
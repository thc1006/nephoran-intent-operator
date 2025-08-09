# Runbook Consolidation Plan

## Analysis Summary

### Current Runbook Distribution

1. **Weaviate-Specific Runbooks** (Keep Separate):
   - `./deployments/weaviate/DEPLOYMENT-RUNBOOK.md` - Detailed Weaviate deployment procedures
   - `./deployments/weaviate/OPERATIONAL-RUNBOOK.md` - Weaviate operations with latest optimizations

2. **General Operational Runbooks** (To Consolidate):
   - `./docs/OPERATIONAL-RUNBOOK.md` - General operational procedures
   - `./docs/monitoring/operational-runbook.md` - Monitoring-specific operations
   - `./docs/monitoring-operations-runbook.md` - Comprehensive monitoring operations
   - `./docs/operations/02-monitoring-alerting-runbooks.md` - Monitoring and alerting procedures

3. **Disaster Recovery Runbooks** (Keep Separate):
   - `./docs/dr-runbook.md` - Comprehensive DR procedures

4. **Production Operations** (Already Consolidated):
   - `./docs/runbooks/production-operations-runbook.md` - Comprehensive production ops

## Consolidation Strategy

### 1. Primary Consolidated Runbooks

#### A. Master Operational Runbook (`operational-runbook-master.md`)
Consolidates:
- General operational procedures
- Daily/weekly/monthly operations
- Service health checks
- Scaling operations
- Cache management
- Knowledge base management

#### B. Monitoring & Alerting Runbook (`monitoring-alerting-runbook.md`)
Consolidates:
- All monitoring procedures
- Alert response procedures
- Prometheus/Grafana configurations
- SLA monitoring
- Performance metrics

#### C. Incident Response Runbook (`incident-response-runbook.md`)
Consolidates:
- Incident classification
- Response procedures by severity
- Troubleshooting guides
- Post-incident activities

### 2. Specialized Runbooks (Keep Separate)

#### A. Weaviate Operations (`weaviate-operations-runbook.md`)
- Link from deployments/weaviate
- Contains Weaviate-specific procedures

#### B. Disaster Recovery (`disaster-recovery-runbook.md`)
- Comprehensive DR procedures
- Backup/restore operations
- Business continuity

#### C. Security Operations (`security-operations-runbook.md`)
- Security monitoring
- Incident response
- Compliance procedures

### 3. Cross-References Structure

All runbooks will cross-reference each other with clear navigation:
- Master index in README.md
- Clear section headers with links
- Scenario-based navigation

## Content Overlap Analysis

### Duplicated Content to Merge:

1. **Health Check Procedures**:
   - Found in 4 different files
   - Will consolidate into single comprehensive procedure

2. **Alert Response Procedures**:
   - P1-P4 classifications appear in 3 files
   - Will create unified response matrix

3. **Performance Monitoring**:
   - Metrics collection procedures in 3 files
   - Will create single monitoring framework

4. **Troubleshooting Guides**:
   - Common issues spread across multiple files
   - Will create comprehensive troubleshooting section

### Unique Content to Preserve:

1. **Weaviate-Specific**:
   - Storage class detection
   - Circuit breaker patterns
   - Key rotation procedures
   - HNSW optimization

2. **Monitoring-Specific**:
   - Component health monitoring scripts
   - SLA validation procedures
   - Capacity planning automation

3. **DR-Specific**:
   - RTO/RPO requirements
   - Failover procedures
   - Recovery validation

## Implementation Steps

1. Create master runbook structure
2. Extract and merge common procedures
3. Eliminate duplication
4. Add cross-references
5. Create navigation index
6. Validate completeness
7. Archive old runbooks

## File Disposition

### Files to Archive:
- `./docs/OPERATIONAL-RUNBOOK.md` → Archive after consolidation
- `./docs/monitoring/operational-runbook.md` → Archive after consolidation
- `./docs/monitoring-operations-runbook.md` → Archive after consolidation

### Files to Keep (with updates):
- `./deployments/weaviate/*.md` → Keep, add references to master runbooks
- `./docs/dr-runbook.md` → Move to runbooks/, update references
- `./docs/runbooks/*.md` → Update with consolidated content

### New Files to Create:
- `./docs/runbooks/README.md` → Master index
- `./docs/runbooks/operational-runbook-master.md` → Consolidated operations
- `./docs/runbooks/monitoring-alerting-runbook.md` → Consolidated monitoring
- `./docs/runbooks/security-operations-runbook.md` → Security procedures